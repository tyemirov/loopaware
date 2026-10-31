// @ts-check
/// <reference types="node" />

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { Script } from "node:vm";
import React from "react";
import { act, create } from "react-test-renderer";
import ts from "@typescript/typescript6";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const require = createRequire(import.meta.url);
let scenarioCount = 0;
const mobileRoot = resolve(import.meta.dirname, "..");
const account = { email: "owner@example.com", name: "Operator", role: "admin" };
const site = {
  id: "aggregate-site", name: "Aggregate site", traffic_profile: "aggregate", access_role: "admin",
  allowed_origin: "https://example.com", favicon_url: "", unique_visitor_count: null, visit_count: 12,
};
/** @typedef {{ date: string, page_views: number, unique_visitors: number | null }} TrendPoint */
const trendPoints = [
  { date: "2026-10-19", page_views: 13, unique_visitors: 3 },
  { date: "2026-10-20", page_views: 0, unique_visitors: 0 },
  { date: "2026-10-21", page_views: 7, unique_visitors: 5 },
  { date: "2026-10-22", page_views: 0, unique_visitors: 0 },
];

// FileReader is a native I/O boundary. Keep the real API requests and image state under test.
class ProtocolFileReader {
  /** @type {string | null} */ result = null;
  /** @type {Error | null} */ error = null;
  /** @type {(() => void) | null} */ onload = null;
  /** @type {(() => void) | null} */ onerror = null;
  aborted = false;
  /** @param {"none" | "callback" | "throw"} [failure] */
  constructor(failure = "none") { this.failure = failure; }
  /** @param {Blob} blob */
  readAsDataURL(blob) {
    if (this.failure === "throw") throw new Error("controlled-synchronous-reader-failure");
    if (this.failure === "callback") { this.error = new Error("controlled-image-conversion-failure"); this.onerror?.(); return; }
    void blob.arrayBuffer().then(bytes => {
      if (this.aborted) return;
      this.result = `data:${blob.type};base64,${Buffer.from(bytes).toString("base64")}`;
      this.onload?.();
    }, error => { this.error = error; this.onerror?.(); });
  }
  abort() { this.aborted = true; }
}

await scenario("mobile avatar resolves the authenticated API image address", {
  initialSession: true, account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  assert(app.requests.includes("/api/me/avatar"), "the photo uses the authenticated fetch transport");
  await app.waitFor(() => app.imageUris().some(value => value.startsWith("data:image/png;base64,")));
});

await scenario("mobile avatar shows the email initial until the photo loads and after a load failure", {
  initialSession: true, account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  assert.equal(app.buttonText("Account menu"), "O");
  await app.avatarEvent("load");
  assert.equal(app.buttonText("Account menu"), "");
  await app.avatarEvent("error");
  assert.equal(app.buttonText("Account menu"), "O");
});

await scenario("mobile avatar keeps the email initial when the backend inline SVG cannot be decoded", {
  initialSession: true, account: { ...account, avatar: { url: "data:image/svg+xml;base64,PHN2Zy8+" } },
}, async app => {
  assert.equal(app.buttonText("Account menu"), "O");
  await app.avatarEvent("error");
  assert.equal(app.buttonText("Account menu"), "O");
});

await scenario("mobile avatar resets on an account change and ignores the previous image completion", {
  initialSession: true, account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  await app.waitFor(() => app.imageUris().length > 0);
  const oldImageLoad = app.avatarCallback("load");
  await app.avatarEvent("load");
  app.setAccount({ email: "zoe@example.com", name: "Zoe", role: "user", avatar: { url: "/api/me/avatar?v=123" } });
  await app.refresh();
  assert.equal(app.buttonText("Account menu"), "Z");
  await act(async () => oldImageLoad());
  assert.equal(app.buttonText("Account menu"), "Z");
  await app.avatarEvent("load");
  assert.equal(app.buttonText("Account menu"), "");
  await app.press("Account menu"); await app.press("Settings"); await app.press("Sign out");
  assert.equal(app.imageUris().length, 0);
});

await scenario("mobile account photo recovers an expired API session once", {
  initialSession: true, avatarMode: "refresh", account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  await app.waitFor(() => app.imageUris().some(uri => uri.startsWith("data:image/png;base64,")));
  assert.equal(app.requests.filter(path => path === "/auth/refresh").length, 1);
  assert.equal(app.requests.filter(path => path === "/api/me/avatar").length, 2);
  await app.waitFor(() => app.imageUris().some(value => value.startsWith("data:image/png;base64,")));
});
await scenario("mobile account photo accepts the API generic binary response", {
  initialSession: true, avatarMode: "octet-stream", account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  await app.waitFor(() => app.imageUris().some(value => value.startsWith("data:application/octet-stream;base64,")));
  await app.avatarEvent("load");
  assert.equal(app.buttonText("Account menu"), "");
});
for (const avatarMode of /** @type {const} */ (["expired", "failed", "invalid", "oversized", "conversion", "conversion-throw"])) {
  await scenario(`mobile account photo keeps the initial and account menu after ${avatarMode}`, {
    initialSession: true, avatarMode, account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
  }, async app => {
    app.expectSignedIn();
    assert.equal(app.buttonText("Account menu"), "O");
    assert.equal(app.imageUris().length, 0);
    await app.press("Account menu"); await app.press("Settings");
    assert.match(app.text(), /owner@example.com/);
  });
}

await scenario("mobile photo recovers on foreground and resets same-image decoding across overlapping requests", {
  initialSession: true, avatarMode: "failed", account: { ...account, avatar: { url: "/api/me/avatar?v=123" } },
}, async app => {
  assert.equal(app.imageUris().length, 0);
  app.recoverPhoto();
  await app.system("light", true);
  await app.waitFor(() => app.imageUris().some(uri => uri.startsWith("data:image/png;base64,")));
  await app.avatarEvent("load"); assert.equal(app.buttonText("Account menu"), "");
  const previousError = app.avatarCallback("error");
  app.holdNextPhoto();
  await app.system("light", true);
  assert.equal(app.buttonText("Account menu"), "O");
  await app.system("light", true);
  assert.equal(app.buttonText("Account menu"), "O");
  await app.avatarEvent("load");
  const currentImage = app.imageUris()[0];
  await act(async () => previousError());
  await app.releasePhoto();
  assert.equal(app.imageUris()[0], currentImage);
  assert.equal(app.buttonText("Account menu"), "");
  assert.equal(app.requests.filter(path => path === "/auth/refresh").length, 0);
});

for (const [action, tab, heading, detail] of [
  ["View feedback", "Feedback", "Feedback", "No feedback yet."],
  ["View subscribers", "Subscribers", "Subscribers", "No subscribers yet."],
  ["View traffic", "Traffic", "Traffic", "Daily counts"],
  ["View health", "Health", "Health", "Health monitor is not available."],
  ["View Sentry", "LA Sentry", "LA Sentry", "No LA Sentry issues."],
]) await scenario(`overview card ${action} opens the corresponding tab`, { initialSession: true, multipleSites: true }, async app => {
  await app.press("Other site"); await app.press("Traffic"); await app.press("7d"); await app.press("Overview");
  const requestCount = app.requests.length;
  const dismissals = app.keyboardDismissals();
  await app.press(action);
  assert(app.selected(tab));
  assert.match(app.text(), new RegExp(detail.replaceAll(".", "\\.")));
  assert.equal(app.focused(), `header:${heading}`);
  assert.equal(app.keyboardDismissals(), dismissals + 1);
  assert.equal(app.lastInterval(), "7days");
  assert(app.selected("Other site"));
  assert.equal(app.requests.length, requestCount);
  assert(!app.buttonLabels().some(label => label.startsWith("View ")));
});

for (const platform of ["ios", "android"]) await scenario(`${platform} site filter matches names and origins while preserving selection`, {
  initialSession:true, multipleSites:true, platform:/** @type {"ios"|"android"} */(platform),
}, async app => {
  assert.equal(app.buttonText("Find sites").trim(), "");
  assert(!app.expanded("Find sites")); assert.equal(app.inputValue("Filter sites"), null);
  assert.doesNotMatch(app.text(), /Find sites|Filter sites/);
  await app.press("Find sites"); assert(app.expanded("Find sites"));
  assert.equal(app.inputValue("Filter sites"), "");
  assert(app.siteFilterInline(), "the input shares the horizontal Sites heading row");
  const requestCount=app.requests.length;
  await app.type("Filter sites", "  aGgReGaTe  ");
  assert(app.buttonLabels().includes("Aggregate site")); assert(!app.buttonLabels().includes("Other site"));
  await app.type("Filter sites", "OTHER.EXAMPLE");
  assert(!app.buttonLabels().includes("Aggregate site")); assert(app.buttonLabels().includes("Other site"));
  assert.match(app.text(), /Requests 12/);
  await app.type("Filter sites", "nothing-matches");
  assert.match(app.text(), /No sites match/); assert.match(app.text(), /Requests 12/);
  assert(app.expanded("Find sites")); assert.equal(app.requests.length,requestCount);
  await app.press("Clear site filter");
  assert.equal(app.inputValue("Filter sites"), ""); assert.equal(app.focused(), "search:Filter sites");
  assert(app.buttonLabels().includes("Aggregate site")); assert(app.buttonLabels().includes("Other site"));
  await app.type("Filter sites", "other");
  const dismissed=app.keyboardDismissals();
  await app.press("Other site");
  assert.match(app.text(), /Requests 23/); assert.equal(app.inputValue("Filter sites"), "other");
  assert.equal(app.keyboardDismissals(),dismissed+1);
  await app.press("Find sites");
  assert.equal(app.inputValue("Filter sites"), null); assert(!app.expanded("Find sites"));
  assert.equal(app.keyboardDismissals(),dismissed+2);
  assert(app.buttonLabels().includes("Aggregate site")); assert(app.buttonLabels().includes("Other site"));
  await app.press("Find sites"); assert.equal(app.inputValue("Filter sites"), "");
});
for (const separator of [" ", ",", ";"]) await scenario(`site filter matches a secondary configured origin separated by ${JSON.stringify(separator)}`, {
  initialSession:true, siteList:() => [{...site,allowed_origin:`https://primary.example${separator}https://Second.Example:8443`}],
}, async app => {
  await app.press("Find sites"); await app.type("Filter sites", "SECOND.EXAMPLE");
  assert(app.buttonLabels().includes("Aggregate site")); assert.doesNotMatch(app.text(), /No sites match/);
  await app.type("Filter sites", " "); assert(app.buttonLabels().includes("Aggregate site"));
});
let currentFilterSites=[site,{...site,id:"other-site",name:"Other site",allowed_origin:"https://other.example"}];
await scenario("site filter derives matches from updated sites without changing the selected dashboard", {
  initialSession:true,siteList:()=>currentFilterSites,
}, async app => {
  await app.press("Find sites"); await app.type("Filter sites", "brand"); assert.match(app.text(),/No sites match/);
  currentFilterSites=[site,{...site,id:"other-site",name:"Brand studio",allowed_origin:"https://other.example"}];
  await app.refresh();
  assert.equal(app.inputValue("Filter sites"),"brand"); assert(app.buttonLabels().includes("Brand studio"));
  assert.match(app.text(),/Requests 12/);
});
await scenario("site filter clears when the authenticated dashboard is replaced", {initialSession:true,multipleSites:true}, async app => {
  await app.press("Find sites"); await app.type("Filter sites","other");
  await app.press("Account menu"); await app.press("Settings"); await app.press("Sign out"); await app.signIn();
  assert.equal(app.inputValue("Filter sites"),null); assert(!app.expanded("Find sites"));
  assert(app.buttonLabels().includes("Aggregate site")); assert(app.buttonLabels().includes("Other site"));
});

const intervalChoices = [["All", "all", 40], ["30d", "30days", 30], ["7d", "7days", 7], ["1d", "1day", 1]];
for (const trafficProfile of ["detailed", "aggregate"]) await scenario(`${trafficProfile} traffic applies every interval to all permitted panels`, {
  initialSession: true, trafficProfile: /** @type {"detailed"|"aggregate"} */ (trafficProfile), multipleSites: true,
  intervalCounts: {all:40,"30days":30,"7days":7,"1day":1},
  trend: (_siteID, interval) => [{date:"2026-10-27",page_views:interval === "7days" ? 7 : 1,unique_visitors:trafficProfile === "aggregate" ? null : 1}],
}, async app => {
  await app.press("Traffic");
  assert.deepEqual(app.buttonLabels().filter(label => intervalChoices.some(([choice]) => choice === label)), ["All","30d","7d","1d"]);
  /** @param {string} siteID @param {string} interval @param {number} count @param {number} requestStart */
  const expectWindow = (siteID, interval, count, requestStart) => {
    const urls=app.requestURLs.slice(requestStart).map(value => new URL(value));
    const traffic=urls.filter(url => /\/visits\//.test(url.pathname));
    const endpoints=trafficProfile === "aggregate" ? ["stats","trend"] : ["stats","trend","attribution","engagement","devices","locations"];
    assert.deepEqual(traffic.map(url => url.pathname.split("/").at(-1)).sort(), endpoints.sort());
    for (const url of traffic) {
      assert.equal(url.pathname.split("/")[3],siteID);
      assert.equal(url.searchParams.get("interval"), interval);
      assert.equal(url.searchParams.get("sort"), trafficProfile === "detailed" && ["stats", "attribution", "devices", "locations"].some(suffix => url.pathname.endsWith(`/visits/${suffix}`)) ? "unique_visitors" : null);
    }
    assert.match(app.text(),new RegExp(`${trafficProfile === "aggregate" ? "Accepted requests" : "Page views"} ${count}\\b`));
    if (trafficProfile === "detailed") {
      assert(app.text().includes(`/page-${interval}`)); assert(app.text().includes(`source-${interval}`)); assert(app.text().includes(`Place ${interval}`));
      assert.match(app.text(),new RegExp(`Unique visitors ${count}\\b`));
    } else assert.match(app.text(),/Unique visitors.*unavailable/);
  };
  for (const [label, interval, count] of intervalChoices) {
    const requestStart=app.requestURLs.length;
    await app.press(String(label));
    assert(app.selected(String(label)), `${label} is selected`);
    for (const [other] of intervalChoices) if (other !== label) assert(!app.selected(String(other)));
    expectWindow("aggregate-site",String(interval),Number(count),requestStart);
  }
  await app.press("7d");
  const requestStart=app.requestURLs.length;
  await app.press("Other site");
  assert(app.selected("7d")); expectWindow("other-site","7days",107,requestStart);
  assert.equal(app.requests.some(path => /\/(attribution|engagement|devices|locations)$/.test(path)),trafficProfile === "detailed");
});

await scenario("avatar opens account settings", { initialSession: true }, async (app) => {
  assert.doesNotMatch(app.text(), /Sign out/);
  await app.press("Account menu");
  await app.press("Settings");
  assert.match(app.text(), /owner@example.com/);
  assert.match(app.text(), /Account role/);
  assert.match(app.text(), /Sign out/);
});

await scenario("settings preserve selection and show distinct account and site roles", { initialSession: true, multipleSites: true }, async (app) => {
  await app.press("Other site"); await app.press("Traffic"); await app.press("1d");
  await app.press("Account menu"); await app.press("Close account menu");
  assert.doesNotMatch(app.text(), /Settings/);
  await app.press("Account menu"); await app.press("Settings");
  assert.match(app.text(), /Email owner@example.com Account role Admin/);
  assert.match(app.text(), /Aggregate site Site role:\s+Admin/);
  assert.match(app.text(), /Other site Site role:\s+Team member/);
  await app.press("Back to dashboard");
  assert.match(app.text(), /Other site/); assert.match(app.text(), /Daily counts/);
  assert.match(app.text(), /Accepted requests 23/); assert.equal(app.lastInterval(),"1day");
  assert.doesNotMatch(app.text(), /Sign out/);
});
for (const platform of ["ios", "android"]) await scenario(`${platform} menu dismissal restores focus and settings receives focus`, {initialSession:true,platform:/** @type {"ios"|"android"} */(platform)}, async app => {
  await app.press("Account menu"); assert.equal(app.focused(),"button:Settings");
  await app.press("Close account menu"); assert.equal(app.focused(),"button:Account menu");
  await app.press("Account menu"); await app.back(); assert.equal(app.focused(),"button:Account menu");
  await app.press("Account menu"); await app.press("Settings"); assert.equal(app.focused(),"header:Settings");
});
await scenario("settings show an account image and non-admin role", {
  initialSession: true, account:{email:"team@example.com",name:"Teammate",role:"user",avatar:{url:"https://example.com/avatar.png"}},
}, async app => {
  await app.waitFor(() => app.imageUris().some(value => value.startsWith("data:image/png;base64,")));
  await app.press("Account menu"); await app.press("Settings");
  assert.match(app.text(), /team@example.com Account role User/);
});
await scenario("menu and settings support back navigation and email initials", {initialSession:true,account:{email:"zoe@example.com",name:"Operator",role:"user"}}, async app => {
  assert.equal(app.buttonText("Account menu"),"Z");
  await app.press("Account menu"); await app.back(); assert.doesNotMatch(app.text(), /Settings/);
  await app.press("Account menu"); await app.press("Settings"); await app.back(); app.expectSignedIn();
});
for (const empty of [false,true]) await scenario(`settings show ${empty ? "empty" : "failed"} site role state`, {initialSession:true,emptySites:empty,sitesFailure:!empty}, async app => {
  await app.press("Account menu"); await app.press("Settings"); assert.match(app.text(),/owner@example.com Account role Admin/);
  assert.match(app.text(),empty ? /No site access/ : /Site roles could not be loaded/);
});
await scenario("failed sign-out stays in settings with an actionable error", {initialSession:true,logoutFailure:true}, async app => {
  await app.press("Account menu"); await app.press("Settings"); await app.press("Sign out");
  assert.match(app.text(), /Could not sign out.*500/); assert.match(app.text(), /Account role/);
  await app.press("Back to dashboard"); app.expectSignedIn();
});
await scenario("themes follow system changes, persist, and apply throughout the app", {initialSession:true,systemTheme:"dark"}, async app => {
  assert.equal(app.background(),"#101816"); assert.equal(app.statusBarStyle(),"light");
  await app.press("Account menu"); await app.press("Settings"); assert(app.selected("System"));
  await app.press("Light"); assert.equal(app.nativeAppearance(),"light"); assert.equal(app.background(),"#f5f7f6"); assert.equal(app.storedTheme(),"light");
  await app.system("light"); await app.system("dark"); assert.equal(app.background(),"#f5f7f6"); assert(app.selected("Light"));
  await app.press("Dark"); await app.system("light"); assert.equal(app.background(),"#101816"); assert(app.selected("Dark"));
  await app.press("Back to dashboard"); assert.equal(app.background(),"#101816");
  await app.remount(); assert.equal(app.background(),"#101816");
  await app.press("Account menu"); await app.press("Settings"); assert(app.selected("Dark"));
  await app.press("System"); assert.equal(app.nativeAppearance(),"unspecified"); assert.equal(app.background(),"#f5f7f6");
  await app.system("dark"); assert.equal(app.background(),"#101816");
  await app.system("light",true); assert.equal(app.background(),"#f5f7f6");
  assert.equal(app.storedTheme(),"system");
  await app.press("Dark"); await app.press("Sign out"); assert.equal(app.background(),"#101816");
  assert.match(app.text(), /Sign in with Google/);
});
await scenario("theme writes are serialized and retain the latest selected choice", {initialSession:true,holdWrites:true,failFirstWrite:true}, async app => {
  await app.press("Account menu"); await app.press("Settings");
  await app.press("Dark"); await app.press("Light"); assert(app.selected("Light")); assert.match(app.text(),/Saving theme/);
  await app.releaseWrite(); assert(app.selected("Light")); assert.match(app.text(),/Saving theme/); assert.doesNotMatch(app.text(),/Could not save/);
  await app.releaseWrite(); assert.equal(app.storedTheme(),"light"); assert(app.selected("Light"));
});
await scenario("theme write errors remain visible and the selected choice can recover", {initialSession:true,storageFailure:true}, async app => {
  await app.press("Account menu"); await app.press("Settings"); await app.press("Dark");
  assert.equal(app.background(),"#101816"); assert.match(app.text(),/Could not save theme preference/);
  assert.equal(app.storedTheme(),undefined); app.recoverStorage(); await app.press("Dark");
  assert.equal(app.storedTheme(),"dark"); assert.doesNotMatch(app.text(),/Could not save/);
});
for (const corrupt of [false,true]) await scenario(`theme read ${corrupt ? "invalid value" : "failure"} is visible and recoverable`, {
  initialSession:true, readFailure:!corrupt, storedTheme:corrupt ? "obsolete-theme" : undefined,
}, async app => {
  await app.press("Account menu"); await app.press("Settings"); assert.match(app.text(),/Could not load theme preference/);
  assert(!app.selected("System")); app.recoverStorage();
  if (!corrupt) { await app.system("dark",true); assert(app.selected("System")); }
  await app.press("Light"); assert.equal(app.storedTheme(),"light");
});

for (const recentVisits of [[], [{
  url: "https://example.com/individual-visit", path: "/individual-visit", ip: "192.0.2.1", country: "Test country",
  browser: "Test browser", user_agent: "Test agent", referrer: "https://referrer.example/", visitor_id: "individual-visitor",
  occurred_at: 1793131200,
}]]) {
  await scenario(`mobile Traffic omits ${recentVisits.length ? "populated" : "empty"} recent visits`, {
    initialSession: true, trafficProfile: "detailed", trend: () => trendPoints, recentVisits,
  }, async (app) => {
    await app.press("Traffic");
    const expectTraffic = () => {
      assert.doesNotMatch(app.text(), /Recent visits|No recent visits|individual-visit|individual-visitor|referrer\.example|Test browser/);
      for (const section of ["Page views", "Unique visitors", "Returning", "Pages/visitor", "Trend", "Top pages", "Attribution", "Devices", "Locations"]) {
        assert.ok(app.text().includes(section), `${section} remains visible`);
      }
      assert.equal(renderedTrend(app.text())[0][0], "2026-10-21");
    };
    expectTraffic();
    await app.press("1d");
    expectTraffic();
    await app.press("Other site");
    expectTraffic();
  });
}

await scenario("mobile traffic uses unique counts for values, bars, and ranked breakdowns", {
  initialSession: true, trafficProfile: "detailed", uniqueTraffic: true, trend: () => trendPoints,
}, async app => {
  assert.equal(app.buttonText("View traffic"), "Unique visitors 8 12 page views");
  await app.press("Traffic");
  assert.match(app.text(), /Unique visitors 8/);
  assert.match(app.text(), /Page views 12/);
  assert.match(app.text(), /Returning 25% 2 returning visitors/);
  for (const suffix of ["stats", "attribution", "devices", "locations"]) {
    assert(app.requestURLs.some(address => { const url = new URL(address); return url.pathname.endsWith(`/visits/${suffix}`) && url.searchParams.get("sort") === "unique_visitors"; }));
  }
  assert.deepEqual(app.bar("2026-10-21"), { value: "5 unique visitors, 7 page views", width: "100%" });
  assert.deepEqual(app.bar("2026-10-19"), { value: "3 unique visitors, 13 page views", width: "60%" });
  assert.deepEqual(app.bar("2026-10-22"), { value: "0 unique visitors, 0 page views", width: "0%" });
  for (const [reach, busy] of [["/reach", "/busy"], ["reach-source", "busy-source"], ["Mobile", "Desktop"]]) {
    assert.deepEqual(app.bar(reach), { value: "3 unique visitors, 3 page views", width: "100%" });
    assert.deepEqual(app.bar(busy), { value: "1 unique visitors, 20 page views", width: "33%" });
    assert(app.text().indexOf(reach) < app.text().indexOf(busy));
  }
  assert.equal(app.bar("Reach place"), null);
  assert(app.buttonLabels().includes("Show Reach place on map"));
  await app.press("Show Busy place on map");
  assert.match(app.text(), /Busy place 1 unique visitor 20 page views/);
  await app.press("7d");
  assert(app.requestURLs.some(address => address.includes("interval=7days") && address.includes("sort=unique_visitors")));
});

await scenario("mobile Locations uses a geographic map and keeps unknown signals off the map", {
  initialSession: true, trafficProfile: "detailed", locationPoints: [
    { label: "Los Angeles", source: "edge_geo", signal: "city", city: "Los Angeles", region: "California", country: "US", latitude: 34.05, longitude: -118.24, confidence: 1, visit_count: 37, unique_visitor_count: 23 },
    { label: "Moscow", source: "timezone", signal: "timezone", city: "", region: "", country: "RU", latitude: 55.75, longitude: 37.62, confidence: 0.4, visit_count: 3, unique_visitor_count: 3 },
    { label: "UTC", source: "timezone", signal: "timezone", city: "", region: "", country: "", latitude: 0, longitude: 0, confidence: 0.1, visit_count: 16, unique_visitor_count: 16 },
  ],
}, async app => {
  await app.press("Traffic");
  assert(app.mapImageUri()?.startsWith("data:image/svg+xml,"));
  assert(app.buttonLabels().includes("Show Los Angeles on map"));
  assert(app.buttonLabels().includes("Show Moscow on map"));
  assert(!app.buttonLabels().includes("Show UTC on map"));
  assert.match(app.text(), /Los Angeles 23 unique visitors 37 page views/);
  assert.match(app.text(), /Location unavailable UTC 16 unique visitors 16 page views/);
  await app.press("Show Moscow on map");
  assert.match(app.text(), /Moscow 3 unique visitors 3 page views/);
  await app.press("7d");
  assert.match(app.text(), /Los Angeles 23 unique visitors 37 page views/);
});

const responsiveLocations = [
  { label: "Los Angeles", source: "edge_geo", signal: "city", city: "Los Angeles", region: "California", country: "US", latitude: 34.05, longitude: -118.24, confidence: 1, visit_count: 37, unique_visitor_count: 23 },
  { label: "San Francisco", source: "edge_geo", signal: "city", city: "San Francisco", region: "California", country: "US", latitude: 37.77, longitude: -122.42, confidence: 1, visit_count: 12, unique_visitor_count: 8 },
  { label: "New York", source: "edge_geo", signal: "city", city: "New York", region: "New York", country: "US", latitude: 40.71, longitude: -74.01, confidence: 1, visit_count: 7, unique_visitor_count: 5 },
];
/** @type {Array<[string, {width:number,height:number,scale:number,fontScale:number}, number, number]>} */
const responsiveViewports = [
  ["narrow iPhone", {width:320,height:568,scale:2,fontScale:1}, 258, 258],
  ["standard Android phone", {width:393,height:852,scale:3,fontScale:1}, 331, 331],
  ["tablet portrait with large text", {width:768,height:1024,scale:2,fontScale:1.5}, 706, 706],
  ["tablet landscape", {width:1194,height:834,scale:2,fontScale:1}, 1132, 720],
];
for (const [name, viewport, contentWidth, expectedWidth] of responsiveViewports) await scenario(`location map adapts to ${name}`, {
  initialSession: true, trafficProfile: "detailed", locationPoints: responsiveLocations,
  platform: name.includes("Android") ? "android" : "ios", viewport,
}, async app => {
  await app.press("Traffic");
  await app.layoutMap(contentWidth);
  assert.deepEqual(app.mapGeometry(), {width: expectedWidth, aspectRatio: 2.4, centered: true});
  assert.deepEqual(app.mapMarkerSizes(), [44, 44, 44]);
  assert(app.buttonLabels().includes("Select San Francisco location"));
  await app.press("Select San Francisco location");
  assert.match(app.text(), /San Francisco 8 unique visitors 12 page views/);
  assert(app.selected("Show San Francisco on map"));
  await app.press("Show New York on map");
  assert(app.selected("Select New York location"));
  assert.match(app.text(), /New York 5 unique visitors 7 page views/);
});
await scenario("location map preserves selection across tablet rotation", {
  initialSession: true, trafficProfile: "detailed", locationPoints: responsiveLocations,
  viewport: {width:768,height:1024,scale:2,fontScale:1},
}, async app => {
  await app.press("Traffic");
  await app.layoutMap(706);
  await app.press("Select New York location");
  await app.resize(1194, 834, 1132);
  assert.deepEqual(app.mapGeometry(), {width:720,aspectRatio:2.4,centered:true});
  assert(app.selected("Select New York location"));
  await app.resize(768, 1024, 706);
  assert.deepEqual(app.mapGeometry(), {width:706,aspectRatio:2.4,centered:true});
  assert.match(app.text(), /New York 5 unique visitors 7 page views/);
});

await scenario("detailed Trend ranks unique visitors and shows unique values and supporting page views", {
  initialSession: true, trafficProfile: "detailed", trend: () => trendPoints,
}, async (app) => {
  await app.press("Traffic");
  assert.deepEqual(renderedTrend(app.text()), [
    ["2026-10-21", "7", "5"], ["2026-10-19", "13", "3"],
    ["2026-10-22", "0", "0"], ["2026-10-20", "0", "0"],
  ], app.text());
});
await scenario("detailed Trend ranks only the existing ten-day display window", {
  initialSession: true, trafficProfile: "detailed",
  trend: () => Array.from({ length: 11 }, (_, index) => ({
    date: `2026-10-${String(index + 10).padStart(2, "0")}`, page_views: 30 - index, unique_visitors: index === 0 ? 999 : index,
  })),
}, async (app) => {
  await app.press("Traffic");
  const rows = renderedTrend(app.text());
  assert.equal(rows.length, 10);
  assert.equal(rows[0][0], "2026-10-20");
  assert.equal(rows.at(-1)?.[0], "2026-10-11");
  assert.equal(rows.some(([date]) => date === "2026-10-10"), false);
});
let otherSiteTrend = [{ date: "2026-10-23", page_views: 12, unique_visitors: 2 }, { date: "2026-10-24", page_views: 8, unique_visitors: 6 }];
await scenario("Trend order follows interval changes, site changes, and refreshed data", {
  initialSession: true, trafficProfile: "detailed",
  trend: (siteID, interval) => siteID === "other-site"
    ? otherSiteTrend
    : interval === "1day" ? [{ date: "2026-10-27", page_views: 9, unique_visitors: 4 }] : trendPoints,
}, async (app) => {
  await app.press("Traffic");
  await app.press("1d");
  assert.deepEqual(renderedTrend(app.text()), [["2026-10-27", "9", "4"]]);
  await app.press("All");
  assert.equal(renderedTrend(app.text())[0][0], "2026-10-21");
  await app.press("Other site");
  assert.deepEqual(renderedTrend(app.text()).map(([date]) => date), ["2026-10-24", "2026-10-23"]);
  otherSiteTrend = [{ date: "2026-10-23", page_views: 15, unique_visitors: 9 }, { date: "2026-10-24", page_views: 8, unique_visitors: 6 }];
  await app.refresh();
  assert.deepEqual(renderedTrend(app.text()), [["2026-10-23", "15", "9"], ["2026-10-24", "8", "6"]]);
});
await scenario("empty detailed Trend keeps its empty state", {
  initialSession: true, trafficProfile: "detailed", trend: () => [],
}, async (app) => {
  await app.press("Traffic");
  assert.match(app.text(), /No trend data yet/);
  assert.deepEqual(renderedTrend(app.text()), []);
});
await scenario("aggregate Daily counts preserve date order and unavailable unique counts", {
  initialSession: true, trend: () => trendPoints.map(point => ({ ...point, unique_visitors: null })),
}, async (app) => {
  await app.press("Traffic");
  assert.match(app.text(), /Daily counts/);
  assert.deepEqual(renderedTrend(app.text()), trendPoints.map(point => [point.date, String(point.page_views), "Unavailable"]));
});

await scenario("fresh login survives dashboard failure", { initialSession: false, dashboardFailure: true }, async (app) => {
  await app.signIn();
  app.expectSignedIn();
  assert.match(app.text(), /Request failed with 409/);
  assert.match(app.text(), /Aggregate site/);
  assert.match(app.text(), /Site unavailable/);
});
await scenario("restored session survives dashboard failure", { initialSession: true, dashboardFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Aggregate site/);
  assert.match(app.text(), /Request failed with 409/);
});
await scenario("aggregate dashboard skips unavailable metrics", { initialSession: false }, async (app) => {
  await app.signIn();
  app.expectSignedIn();
  assert.match(app.text(), /Aggregate site/);
  assert.doesNotMatch(app.text(), /Request failed|Site unavailable/);
  assert.equal(app.requests.some((path) => /\/(attribution|engagement|devices|locations)$/.test(path)), false);
});
await scenario("authentication rejection stays signed out", { initialSession: false, rejectLogin: true }, async (app) => {
  await app.signIn();
  assert.match(app.text(), /Sign in with Google/);
  assert.doesNotMatch(app.text(), /Sign out/);
  assert.match(app.text(), /Request failed with 401/);
  assert.equal(app.requests.includes("/api/sites"), false);
});
await scenario("site discovery failure stays authenticated", { initialSession: true, sitesFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Sites unavailable/);
  assert.doesNotMatch(app.text(), /No sites/);
});
await scenario("late dashboard data cannot replace another selected site", { initialSession: true, deferDashboard: true }, async (app) => {
  app.expectSignedIn();
  await app.press("Other site");
  assert.match(app.text(), /Requests 23/);
  await app.press("Reports");
  assert.match(app.text(), /Visits 35/);
  assert.doesNotMatch(app.text(), /All-sites report data unavailable/);
  await app.release();
  assert.match(app.text(), /Visits 35/);
  await app.press("Overview");
  assert.match(app.text(), /Requests 23/);
  assert.doesNotMatch(app.text(), /Requests 12/);
});
await scenario("late site discovery cannot replace another selected site", { initialSession: true, deferSites: true }, async (app) => {
  app.expectSignedIn();
  await app.refresh();
  await app.press("Other site");
  assert.match(app.text(), /Requests 23/);
  await app.press("Reports");
  assert.match(app.text(), /Visits 58/);
  await app.release();
  assert.match(app.text(), /Visits 58/);
  await app.press("Overview");
  assert.match(app.text(), /Requests 23/);
  assert.doesNotMatch(app.text(), /Requests 12/);
});
await scenario("portfolio failure preserves the selected-site dashboard", { initialSession: true, portfolioFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Requests 12/);
  assert.match(app.text(), /Request failed with 500/);
  await app.press("Reports");
  assert.match(app.text(), /All-sites report data unavailable/);
});
await scenario("late portfolio data cannot replace a new session's reports", { initialSession: true, deferPortfolio: true }, async (app) => {
  await app.press("Other site");
  assert.match(app.text(), /Requests 23/);
  await app.press("Account menu");
  await app.press("Settings");
  await app.press("Sign out");
  assert.match(app.text(), /Sign in with Google/);
  await app.signIn();
  await app.press("Reports");
  assert.match(app.text(), /Visits 58/);
  await app.release();
  assert.match(app.text(), /Visits 58/);
  assert.doesNotMatch(app.text(), /Visits 35/);
});
await scenario("fresh login cleanup cannot enable sign-in during a newer Google prompt", { initialSession: false, deferPortfolio: true }, async (app) => {
  await app.signIn("heldResponse");
  app.expectSignedIn();
  await app.press("Other site");
  assert.match(app.text(), /Requests 23/);
  await app.press("Account menu");
  await app.press("Settings");
  await app.press("Sign out");
  await app.startPendingSignIn();
  app.expectSignInPending();
  await app.release();
  app.expectSignInPending();
  await app.completeSignIn();
  app.expectSignedIn();
  await app.press("Reports");
  assert.match(app.text(), /Visits 58/);
  assert.doesNotMatch(app.text(), /Visits 35/);
});

let currentFaviconURL = `/api/sites/${site.id}/favicon?v=1`;
for (const faviconFormat of /** @type {const} */ (["png", "svg", "ico"])) await scenario(`site badges show a collected ${faviconFormat} favicon without another text row`, {
  initialSession: true, faviconFormat, siteList: () => [{ ...site, favicon_url: `/api/sites/${site.id}/favicon?v=1` }],
}, async app => {
  assert(app.requests.includes(`/api/sites/${site.id}/favicon`));
  await app.waitFor(() => Boolean(app.siteIcon("Aggregate site")));
  const icon = app.siteIcon("Aggregate site");
  assert(icon);
  assert.match(icon, /^data:image\//);
  assert.equal(app.siteIconDecorative("Aggregate site"), true);
  assert(app.selected("Aggregate site"));
  await app.press("Find sites"); await app.type("Filter sites", "aggregate");
  assert.equal(app.siteIcon("Aggregate site"), icon);
});

await scenario("site badges keep their text when a favicon fails and load a changed favicon URL", {
  initialSession: true, faviconFormat: "png", siteList: () => [{ ...site, favicon_url: currentFaviconURL }],
}, async app => {
  await app.waitFor(() => Boolean(app.siteIcon("Aggregate site")));
  await app.siteIconError("Aggregate site");
  assert.equal(app.siteIcon("Aggregate site"), null);
  assert(app.buttonText("Aggregate site").includes("Aggregate site"));
  currentFaviconURL = `/api/sites/${site.id}/favicon?v=2`;
  await app.refresh();
  await app.waitFor(() => Boolean(app.siteIcon("Aggregate site")));
});

await scenario("site badges omit absent and failed favicon images", {
  initialSession: true, faviconFailure: true, siteList: () => [{ ...site, favicon_url: "" }, { ...site, id: "other-site", name: "Other site", favicon_url: "/api/sites/other-site/favicon?ts=1" }],
}, async app => {
  assert(!app.requests.includes(`/api/sites/${site.id}/favicon`));
  assert.equal(app.siteIcon("Aggregate site"), null);
  assert.equal(app.siteIcon("Other site"), null);
  await app.press("Other site");
  assert(app.selected("Other site"));
});

await scenario("site favicons retry on foreground return and reset across accounts", {
  initialSession: true, siteList: () => [{ ...site, favicon_url: "/api/sites/aggregate-site/favicon?ts=1" }],
}, async app => {
  await app.siteIconError("Aggregate site");
  assert.equal(app.siteIcon("Aggregate site"), null);
  await app.system("light", true);
  await app.waitFor(() => Boolean(app.siteIcon("Aggregate site")));
  const before = app.requests.filter(path => path.endsWith("/favicon")).length;
  app.setAccount({email:"new@example.com",name:"New owner",role:"admin"});
  await app.refresh();
  await app.waitFor(() => Boolean(app.siteIcon("Aggregate site")));
  assert(app.requests.filter(path => path.endsWith("/favicon")).length > before);
});

console.log(`${scenarioCount} mobile app flow checks passed`);

/**
 * @param {string} name
 * @param {{ initialSession: boolean, uniqueTraffic?: boolean, locationPoints?: import("../src/types").LocationPoint[], viewport?: {width:number,height:number,scale:number,fontScale:number}, platform?: "ios"|"android", logoutFailure?: boolean, multipleSites?: boolean, siteList?: () => Array<typeof site>, intervalCounts?: Record<string,number>, faviconFailure?: boolean, faviconFormat?: "png" | "svg" | "ico", emptySites?: boolean, avatarMode?: "refresh" | "expired" | "failed" | "invalid" | "oversized" | "conversion" | "conversion-throw" | "octet-stream", failFirstWrite?: boolean, account?: import("../src/types").Account, systemTheme?: "light" | "dark", storedTheme?: string, storageFailure?: boolean, readFailure?: boolean, holdWrites?: boolean, trafficProfile?: "detailed" | "aggregate", trend?: (siteID: string, interval: string) => TrendPoint[], recentVisits?: import("../src/types").VisitLogEntry[], dashboardFailure?: boolean, rejectLogin?: boolean, sitesFailure?: boolean, deferDashboard?: boolean, deferSites?: boolean, portfolioFailure?: boolean, deferPortfolio?: boolean }} options
 * @param {(app: { signIn: (waitFor?: "idle" | "heldResponse") => Promise<void>, startPendingSignIn: () => Promise<void>, completeSignIn: () => Promise<void>, expectSignInPending: () => void, press: (label: string) => Promise<void>, refresh: () => Promise<void>, release: () => Promise<void>, expectSignedIn: () => void, text: () => string, requests: string[], requestURLs: string[], buttonLabels: () => string[], mapImageUri: () => string | null, mapGeometry: () => {width:number|string,aspectRatio:number,centered:boolean}, mapMarkerSizes: () => number[], layoutMap: (width:number) => Promise<void>, resize: (width:number,height:number,contentWidth:number) => Promise<void>, type: (label:string,value:string) => Promise<void>, inputValue: (label:string) => string|null, expanded: (label:string) => boolean, keyboardDismissals: () => number, lastInterval: () => string, background: () => string, selected: (label: string) => boolean, storedTheme: () => string | undefined, system: (scheme: "light" | "dark", foreground?: boolean) => Promise<void>, remount: () => Promise<void>, recoverPhoto: () => void, holdNextPhoto: () => void, releasePhoto: () => Promise<void>, recoverStorage: () => void, releaseWrite: () => Promise<void>, bar: (label:string) => {value:string,width:string}|null, waitFor: (predicate: () => boolean) => Promise<void>, imageUris: () => string[], siteIcon: (label: string) => string | null, siteIconDecorative: (label: string) => boolean, siteIconError: (label: string) => Promise<void>, siteFilterInline: () => boolean, avatarEvent: (event: "load" | "error") => Promise<void>, avatarCallback: (event: "load" | "error") => () => void, setAccount: (account: import("../src/types").Account) => void, back: () => Promise<void>, buttonText: (label:string) => string, focused: () => string, nativeAppearance: () => string, statusBarStyle: () => string }) => Promise<void>} check
 */
async function scenario(name, options, check) {
  let authenticated = options.initialSession;
  let avatarMode = options.avatarMode;
  let holdNextPhoto = false;
  /** @type {(() => void) | undefined} */ let releasePhoto;
  let currentAccount = options.account ?? account;
  let lastInterval = "";
  let pendingAvatarRequests = 0;
  let imageRequestCount = 0;
  let allocatedImageBlobs = 0;
  let closedImageBlobs = 0;
  /** @type {Device} */
  const device = { storage: new Map(), scheme: options.systemTheme ?? "light", listeners: new Set(), foreground: new Set(), readFailure: options.readFailure ?? false, writeFailure: options.storageFailure ?? false, holdWrites: options.holdWrites ?? false, pendingWrites: [], failFirstWrite: options.failFirstWrite ?? false, writeCount: 0, back: new Set(), platform:options.platform ?? "ios", viewport:options.viewport ?? {width:375,height:667,scale:2,fontScale:1}, tags:new Map(), focused:[], nativeOverride: null, keyboardDismissals: 0, appState: "active", imageReadFailure: avatarMode === "conversion" ? "callback" : avatarMode === "conversion-throw" ? "throw" : "none" };
  if (options.storedTheme !== undefined) device.storage.set("loopaware.mobile.theme", options.storedTheme);
  const visibleSite = { ...site, traffic_profile: options.trafficProfile ?? "aggregate" };
  /** @type {string[]} */
  const requests = [];
  /** @type {string[]} */
  const requestURLs = [];
  let releaseResponse = () => {};
  let deferred = options.deferDashboard || options.deferPortfolio;
  let responseHeld = false;
  let responseConsumed = () => {};
  const consumed = new Promise(resolve => { responseConsumed = () => resolve(undefined); });
  /** @type {typeof fetch} */
  const fetcher = async (input, init) => {
    const path = new URL(String(input)).pathname;
    const deferredRequest = (options.deferSites && path === "/api/sites" && requests.filter(value => value === path).length === 1)
      || (options.deferDashboard && path === `/api/sites/${site.id}/visits/stats`)
      || (options.deferPortfolio && path === "/api/reports/traffic/portfolio" && !requests.includes(path));
    const imageRequest = path.endsWith("/favicon") || path === "/api/me/avatar" || String(input) === "https://example.com/avatar.png";
    const holdPhoto = imageRequest && holdNextPhoto;
    if (holdPhoto) holdNextPhoto = false;
    if (imageRequest) {
      assert.equal(init?.credentials, String(input) === "https://example.com/avatar.png" ? "omit" : "include");
      assert.equal(init?.cache, "no-store");
    }
    const trackedImageRequest = imageRequest || path === "/auth/refresh";
    if (trackedImageRequest) pendingAvatarRequests++;
    let response;
    try { response = await fetch(String(input) === "https://example.com/avatar.png" ? `${origin}/external-avatar` : input, holdPhoto ? { ...init, signal: undefined } : init); }
    finally { if (trackedImageRequest) pendingAvatarRequests--; }
    if (imageRequest) {
      const readBlob = response.blob.bind(response);
      response.blob = async () => {
        const blob = await readBlob(); allocatedImageBlobs++;
        return Object.assign(blob, { close() { closedImageBlobs++; } });
      };
    }
    if (deferredRequest) {
      const readJSON = response.json.bind(response);
      response.json = async () => {
        const data = await readJSON();
        responseConsumed();
        return data;
      };
    }
    if (holdPhoto) await new Promise(resolve => { releasePhoto = () => resolve(undefined); });
    return response;
  };
  const server = createServer((request, response) => {
    const requestURL = new URL(request.url || "/", "http://localhost");
    const path = requestURL.pathname;
    if (path.endsWith("/favicon")) {
      requests.push(path); requestURLs.push(requestURL.href);
      if (options.faviconFailure) { response.writeHead(503); response.end(); return; }
      response.writeHead(200, {"Content-Type": {png:"image/png",svg:"image/svg+xml",ico:"image/x-icon"}[options.faviconFormat || "png"]});
      response.end(Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jC5kAAAAASUVORK5CYII=", "base64")); return;
    }
    if (path === "/api/me/avatar" || path === "/external-avatar") {
      requests.push(path); requestURLs.push(requestURL.href);
      imageRequestCount++;
      if (avatarMode === "expired" || (avatarMode === "refresh" && imageRequestCount === 1)) {
        response.writeHead(401, {"Content-Type":"application/json"}); response.end('{"error":"unauthorized"}'); return;
      }
      if (avatarMode === "failed") { response.writeHead(503); response.end(); return; }
      if (avatarMode === "invalid") { response.writeHead(200, {"Content-Type":"text/html"}); response.end("invalid image"); return; }
      if (avatarMode === "oversized") { response.writeHead(200, {"Content-Type":"image/png"}); response.end(Buffer.alloc((1 << 20) + 1)); return; }
      response.writeHead(200, {"Content-Type": avatarMode === "octet-stream" ? "application/octet-stream" : "image/png", "Cache-Control":"no-cache"});
      response.end(Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jC5kAAAAASUVORK5CYII=", "base64"));
      return;
    }
    const interval = requestURL.searchParams.get("interval") || "30days";
    const uniqueOrder = requestURL.searchParams.get("sort") === "unique_visitors";
    const ranked = (/** @type {"path"|"value"|"device_type"|"label"} */ key, /** @type {string} */ reach, /** @type {string} */ busy) => {
      const rows = [{[key]:busy,visit_count:20,unique_visitor_count:1},{[key]:reach,visit_count:3,unique_visitor_count:3}];
      return uniqueOrder ? rows.reverse() : rows;
    };
    requests.push(path);
    requestURLs.push(requestURL.href);
    const visitCount = options.intervalCounts ? options.intervalCounts[interval] + (path.includes("other-site") ? 100 : 0) : path.includes("other-site") ? 23 : 12;
    if (path.endsWith("/visits/stats")) lastInterval = interval;
    let status = 200;
    /** @type {unknown} */
    let payload;
    switch (path) {
      case "/api/me": status = authenticated ? 200 : 401; payload = authenticated ? currentAccount : { error: "unauthorized" }; break;
      case "/auth/refresh": status = avatarMode === "refresh" ? 200 : 401; payload = status === 200 ? {} : { error: "missing_session" }; break;
      case "/auth/nonce": payload = { nonce: "issued-nonce" }; break;
      case "/auth/google/native/config":
        payload = { clients: [{ platform: "ios", client_id: "ios-client", redirect_uris: ["loopaware:/oauth"] }], authorization_endpoint: "https://accounts.example/auth", token_endpoint: "https://accounts.example/token", scopes: ["openid"] }; break;
      case "/auth/google/native":
        assert.equal(request.headers["x-tauth-tenant"], "loopaware");
        authenticated = !options.rejectLogin;
        status = authenticated ? 200 : 401; payload = authenticated ? currentAccount : { error: "invalid_google_token" }; break;
      case "/auth/logout": status = options.logoutFailure ? 500 : 200; if (!options.logoutFailure) authenticated = false; payload = options.logoutFailure ? {error:"logout_failed"} : {}; break;
      case "/api/sites": status = options.sitesFailure ? 500 : 200; payload = options.sitesFailure ? { error: "query_failed" } : { sites: options.siteList ? options.siteList() : options.emptySites ? [] : options.deferDashboard || options.deferSites || options.deferPortfolio || options.trend || options.multipleSites ? [visibleSite, { ...visibleSite, id: "other-site", name: "Other site", allowed_origin:"https://other.example", access_role: "team_member" }] : [visibleSite] }; break;
      case "/api/reports/traffic/portfolio/reports": payload = { reports: [], available_sites: [] }; break;
      case "/api/reports/traffic/portfolio":
        status = options.portfolioFailure ? 500 : 200;
        payload = options.portfolioFailure ? { error: "query_failed" } : { site_count: 2, visit_count: requests.filter(value => value === path).length === 1 ? 35 : 58, unique_visitor_count: null, trend: [], sites: [] }; break;
      case "/api/reports/traffic/portfolio/schedule": payload = null; break;
      default:
        if (options.trafficProfile === "detailed" && /\/visits\/(attribution|engagement|devices|locations)$/.test(path)) {
          payload = {
            sources: options.uniqueTraffic ? ranked("value","reach-source","busy-source") : options.intervalCounts ? [{value:`source-${interval}`,visit_count:visitCount,unique_visitor_count:visitCount}] : [], mediums: [], campaigns: [], device_types: options.uniqueTraffic ? ranked("device_type","mobile","desktop") : options.intervalCounts ? [{device_type:"mobile",visit_count:visitCount,unique_visitor_count:visitCount}] : [], top_resolutions: [], top_viewports: [], locations: options.locationPoints ?? (options.uniqueTraffic ? ranked("label","Reach place","Busy place").map((point, index) => ({...point, latitude: index ? 38 : 51, longitude: index ? -122 : 0, source:"edge_geo",signal:"city", country:"US",region:"",city:point.label,confidence:1})) : options.intervalCounts ? [{label:`Place ${interval}`,visit_count:visitCount,unique_visitor_count:visitCount,latitude:51,longitude:0,source:"edge_geo",signal:"city",country:"GB",region:"",city:"Place",confidence:1}] : []),
            tracked_visitor_count: 8, returning_visitor_count: 2, returning_visitor_rate: 0.25, average_pages_per_visitor: 1.5,
            depth_distribution: {}, observed_time_distribution: {},
          };
        } else if (/\/visits\/(attribution|engagement|devices|locations)$/.test(path) || (options.dashboardFailure && path.endsWith("/visits/stats"))) {
          status = 409; payload = { error: "metric_unavailable", traffic_profile: "aggregate" };
        } else if (path.endsWith("/visits/stats")) {
          payload = { site_id: path.split("/")[3], interval, traffic_profile: visibleSite.traffic_profile, visit_count: visitCount, unique_visitor_count: options.trafficProfile === "detailed" ? options.intervalCounts ? visitCount : 8 : null, top_pages: options.uniqueTraffic ? ranked("path","/reach","/busy") : options.intervalCounts ? [{path:`/page-${interval}`,visit_count:visitCount,unique_visitor_count:visitCount}] : null, recent_visits: options.recentVisits ?? null };
        } else if (path.endsWith("/visits/trend")) {
          payload = { site_id: path.split("/")[3], interval, trend: options.trend ? options.trend(path.split("/")[3], interval) : [] };
        } else {
          const field = { messages: "messages", subscribers: "subscribers", issues: "issues", "mobile-apps": "mobile_apps", team: "team_members" }[path.split("/").at(-1) || ""];
          payload = field ? { [field]: [] } : null;
        }
    }
    const send = () => {
      response.writeHead(status, { "Content-Type": "application/json" });
      response.end(JSON.stringify(payload));
    };
    if ((options.deferSites && path === "/api/sites" && requests.filter(value => value === path).length === 2)
      || (options.deferDashboard && path === `/api/sites/${site.id}/visits/stats`)
      || (options.deferPortfolio && path === "/api/reports/traffic/portfolio" && requests.filter(value => value === path).length === 1)) {
      responseHeld = true;
      releaseResponse = send;
    } else send();
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", () => resolve(undefined)));
  const address = server.address();
  assert(address && typeof address === "object");
  const origin = `http://127.0.0.1:${address.port}`;
  /** @type {() => Promise<unknown>} */
  let prompt = async () => undefined;
  let completePrompt = () => {};
  const { App, assetLoads } = loadApp(origin, fetcher, () => prompt(), device);
  /** @type {import("react-test-renderer").ReactTestRenderer | undefined} */
  let renderer;
  function tree() { assert(renderer); return renderer; }
  function createApp() {
    return create(React.createElement(App), { createNodeMock(element) {
      const tag=device.tags.size+1;
      const props = /** @type {{accessibilityRole?:string,accessibilityLabel?:string,children?:React.ReactNode}} */ (element.props);
      device.tags.set(tag, `${props.accessibilityRole ?? "text"}:${props.accessibilityLabel ?? textContent(props.children)}`);
      return {tag,scrollTo(){},focus(){device.focused.push(`${props.accessibilityRole ?? "text"}:${props.accessibilityLabel ?? ""}`);}};
    } });
  }
  try {
    await act(async () => { renderer = createApp(); });
    await settle();
    const text = () => textContent(tree().toJSON());
    /** @param {string} label */
    function findButton(label) {
      return tree().root.findAll(node => String(node.type) === "Pressable" && (node.props.accessibilityLabel === label || textContent(node.props.children).includes(label)))[0];
    }
    /** @param {string} label @param {"idle" | "heldResponse"} waitFor */
    async function press(label, waitFor = "idle") {
      deferred = waitFor === "heldResponse";
      const button = findButton(label);
      assert(button, `${name}: ${label}`);
      assert.notEqual(button.props.disabled, true, `${name}: ${label} is disabled`);
      // Site selection events start an asynchronous load without returning its promise.
      await act(async () => { button.props.onPress(); });
      await settle();
    }
    /** @param {"load" | "error"} event */
    function avatarCallback(event) {
      const image = tree().root.findAll(node => String(node.type) === "Image" && node.props.source?.uri)[0];
      assert(image, `${name}: account image`);
      const callback = image.props[event === "load" ? "onLoad" : "onError"];
      assert.equal(typeof callback, "function", `${name}: native image ${event} callback`);
      return /** @type {() => void} */ (callback);
    }
    await check({
      text, requests, requestURLs, buttonLabels: () => tree().root.findAll(node => String(node.type) === "Pressable").map(node => node.props.accessibilityLabel ?? textContent(node.props.children)),
      mapImageUri: () => tree().root.findAll(node => String(node.type) === "FaviconImage" && node.props.accessibilityLabel === "World map")[0]?.props.source?.uri ?? null,
      mapGeometry() {
        const frame = tree().root.findAll(node => node.props.testID === "location-map-frame")[0];
        const canvas = tree().root.findAll(node => node.props.testID === "location-map-canvas")[0];
        assert(frame && canvas);
        const style = Array.isArray(canvas.props.style) ? Object.assign({}, ...canvas.props.style.filter(Boolean)) : canvas.props.style;
        return {width:style.width,aspectRatio:style.aspectRatio,centered:frame.props.style.alignItems === "center"};
      },
      mapMarkerSizes() { return tree().root.findAll(node => String(node.type) === "Pressable" && String(node.props.accessibilityLabel).startsWith("Show ") && String(node.props.accessibilityLabel).endsWith(" on map")).map(node => node.props.style[0].width); },
      async layoutMap(width) {
        const frame = tree().root.findAll(node => node.props.testID === "location-map-frame")[0];
        assert(frame);
        await act(async () => frame.props.onLayout({nativeEvent:{layout:{width,height:width/2.4,x:0,y:0}}}));
        await settle();
      },
      async resize(width,height,contentWidth) {
        device.viewport = {...device.viewport,width,height};
        await act(async () => tree().update(React.createElement(App)));
        await settle();
        const frame = tree().root.findAll(node => node.props.testID === "location-map-frame")[0];
        assert(frame);
        await act(async () => frame.props.onLayout({nativeEvent:{layout:{width:contentWidth,height:contentWidth/2.4,x:0,y:0}}}));
        await settle();
      },
      async type(label,value) { const input=tree().root.findAll(node => String(node.type) === "TextInput" && node.props.accessibilityLabel === label)[0]; assert(input,`${name}: ${label}`); await act(async()=>{input.props.onChangeText(value);}); await settle(); },
      inputValue(label) { return tree().root.findAll(node => String(node.type) === "TextInput" && node.props.accessibilityLabel === label)[0]?.props.value ?? null; },
      expanded(label) { return findButton(label)?.props.accessibilityState?.expanded === true; },
      keyboardDismissals() { return device.keyboardDismissals; }, lastInterval: () => lastInterval,
      background() { return tree().root.findAll(node => String(node.type) === "SafeAreaView")[0].props.style.backgroundColor; },
      selected(label) { return findButton(label)?.props.accessibilityState?.selected === true; },
      storedTheme() { return device.storage.get("loopaware.mobile.theme"); },
      nativeAppearance() { return device.nativeOverride ?? "unspecified"; },
      focused() { return device.focused.at(-1) ?? ""; },
      statusBarStyle() { return tree().root.findAll(node => String(node.type) === "StatusBar")[0].props.style; },
      buttonText(label) { const button = findButton(label); assert(button); return textContent(button.findAll(node => String(node.type) === "Text").map(node => node.props.children)); },
      async back() { await act(async () => { const modal=tree().root.findAll(node => String(node.type) === "Modal")[0]; if (modal) modal.props.onRequestClose(); else for (const callback of device.back) callback(); }); await settle(); },
      siteIcon(label) { return findButton(label)?.findAll(node => String(node.type) === "FaviconImage")[0]?.props.source?.uri ?? null; },
      siteIconDecorative(label) { return findButton(label)?.findAll(node => String(node.type) === "FaviconImage")[0]?.props.accessible === false; },
      async siteIconError(label) { await waitFor(() => Boolean(findButton(label)?.findAll(node => String(node.type) === "FaviconImage")[0])); const icon = findButton(label)?.findAll(node => String(node.type) === "FaviconImage")[0]; assert(icon); await act(async () => icon.props.onError()); await settle(); },
      siteFilterInline() {
        const input = tree().root.findAll(node => String(node.type) === "TextInput" && node.props.accessibilityLabel === "Filter sites")[0];
        const row = input?.parent?.parent;
        return row?.props.style?.flexDirection === "row"
          && row.findAll(node => String(node.type) === "Text" && node.props.children === "Sites").length === 1
          && row.findAll(node => String(node.type) === "Pressable" && node.props.accessibilityLabel === "Find sites").length === 1;
      },
      avatarCallback,
      async avatarEvent(event) { await waitFor(() => tree().root.findAll(node => String(node.type) === "Image" && node.props.source?.uri).length > 0); const callback = avatarCallback(event); await act(async () => callback()); await settle(); },
      setAccount(account) { currentAccount = account; },
      bar(label) {
        const row = tree().root.findAll(node => String(node.type) === "View" && node.props.accessibilityLabel === label)[0];
        if (!row) return null;
        const fill = row.findAll(node => String(node.type) === "View" && Array.isArray(node.props.style) && node.props.style.some((/** @type {{width?:string}} */ style) => typeof style?.width === "string"))[0];
        return { value: row.props.accessibilityValue.text, width: fill?.props.style.find((/** @type {{width?:string}} */ style) => typeof style?.width === "string").width };
      },
      waitFor,
      imageUris() { return tree().root.findAll(node => String(node.type) === "Image").map(node => node.props.source?.uri).filter(Boolean); },
      async system(scheme, foreground = false) {
        device.scheme = scheme;
        await act(async () => {
          if (foreground) { device.appState = "background"; for (const notify of device.foreground) notify(); device.appState = "active"; }
          for (const notify of (foreground ? device.foreground : device.listeners)) notify();
        });
        await settle();
      },
      async remount() { await act(async () => { renderer?.unmount(); }); await act(async () => { renderer = createApp(); }); await settle(); },
      recoverPhoto() { avatarMode = undefined; device.imageReadFailure = "none"; },
      holdNextPhoto() { holdNextPhoto = true; },
      async releasePhoto() { assert(releasePhoto); await act(async () => releasePhoto?.()); await settle(); },
      recoverStorage() { device.readFailure = false; device.writeFailure = false; },
      async releaseWrite() { const release = device.pendingWrites.shift(); assert(release); await act(async () => { release(); }); await settle(); },
      press,
      async refresh() {
        deferred = true;
        const control = tree().root.findAll(node => String(node.type) === "ScrollView" && node.props.refreshControl)[0]?.props.refreshControl;
        assert(control);
        await act(async () => { control.props.onRefresh(); });
        await settle();
      },
      async release() {
        deferred = false;
        await act(async () => { releaseResponse(); await consumed; });
        await settle();
      },
      expectSignedIn() { assert(findButton("Account menu"), name); assert.doesNotMatch(text(), /Sign in with Google/, name); },
      expectSignInPending() {
        const button = findButton("Opening Google") || findButton("Sign in with Google");
        assert(button, `${name}: sign-in screen is visible`);
        assert.equal(button.props.disabled, true, `${name}: pending Google login keeps sign-in disabled`);
      },
      async signIn(waitFor = "idle") { await press("Sign in with Google", waitFor); },
      async startPendingSignIn() {
        let promptStarted = () => {};
        const started = new Promise(resolve => { promptStarted = () => resolve(undefined); });
        const pending = new Promise(resolve => { completePrompt = () => resolve(undefined); });
        prompt = () => { promptStarted(); return pending; };
        await press("Sign in with Google");
        await act(async () => { await started; });
      },
      async completeSignIn() {
        await act(async () => { completePrompt(); });
        await settle();
      },
    });
    scenarioCount += 1;
    assert(assetLoads() > 0, `${name}: real screen components rendered`);
  } finally {
    if (renderer) await act(async () => renderer?.unmount());
    server.closeAllConnections();
    await new Promise(resolve => server.close(resolve));
    assert.equal(closedImageBlobs, allocatedImageBlobs, `${name}: release native image bytes`);
  }

  /** @param {() => boolean} predicate */
  async function waitFor(predicate) {
    for (let attempt = 0; attempt < 50; attempt++) {
      if (predicate()) return;
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    }
    assert(predicate(), `${name}: expected image state did not arrive`);
  }

  async function settle() {
    // Flush real HTTP responses and the React effects they trigger.
    for (let attempt = 0; attempt < 50; attempt++) {
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
      await act(async () => { for (const node of tree().root.findAll(node => String(node.type) === "Text" && node.props.onLayout)) node.props.onLayout(); });
      if (deferred && responseHeld) return;
      if (pendingAvatarRequests === 0 && !tree().root.findAll(node => String(node.type) === "ActivityIndicator").length) return;
    }
    throw new Error(`${name}: app did not finish loading`);
  }
}

/** @param {string} text */
function renderedTrend(text) {
  const pattern = /(\d{4}-\d{2}-\d{2})\s+([\d,]+)\s+(unique|requests)(?:\s+([\d,]+)\s+page views)?/g;
  return [...text.matchAll(pattern)].map(match => [match[1], match[3] === "unique" ? match[4] : match[2], match[3] === "unique" ? match[2] : "Unavailable"]);
}

/** @param {unknown} node @returns {string} */
function textContent(node) {
  if (node === null || node === undefined || typeof node === "boolean") return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(textContent).join(" ");
  const element = /** @type {{ children?: unknown, props?: { children?: unknown } }} */ (node);
  return textContent(element.children ?? element.props?.children);
}

/** @typedef {{storage: Map<string,string>, scheme: "light"|"dark", listeners: Set<()=>void>, foreground: Set<()=>void>, readFailure:boolean,writeFailure:boolean,holdWrites:boolean,pendingWrites:Array<()=>void>,failFirstWrite:boolean,writeCount:number,back:Set<()=>boolean>,platform:"ios"|"android",viewport:{width:number,height:number,scale:number,fontScale:number},tags:Map<number,string>,focused:string[], nativeOverride:"light"|"dark"|null,keyboardDismissals:number,appState:"active"|"background",imageReadFailure:"none"|"callback"|"throw"}} Device */
/** @param {string} origin @param {typeof fetch} fetcher @param {() => Promise<unknown>} prompt @param {Device} device */
function loadApp(origin, fetcher, prompt, device) {
  let assetLoadCount = 0;
  /** @param {{visible:boolean,onShow:()=>void,onDismiss:()=>void,children:React.ReactNode}} props */
  function NativeModal(props) {
    const previous=React.useRef(false);
    React.useEffect(() => {
      if (props.visible && !previous.current) props.onShow();
      if (!props.visible && previous.current && device.platform === "ios") props.onDismiss();
      previous.current=props.visible;
    }, [props.visible]);
    return props.visible ? React.createElement("Modal", props, props.children) : null;
  }
  const nativeComponents = ["ActivityIndicator", "Image", "Pressable", "RefreshControl", "ScrollView", "Text", "TextInput", "View", "Modal"];
  /** @type {Record<string, unknown>} */
  const mocks = {
    "./assets/icon.png": 1,
    "react-native": { ...Object.fromEntries(nativeComponents.map(name => [name, name])), Platform: { OS: device.platform }, useWindowDimensions: () => device.viewport, Keyboard: {dismiss(){device.keyboardDismissals++;}},
      AccessibilityInfo: { setAccessibilityFocus: (/** @type {number} */ tag) => { const label=device.tags.get(tag); assert(label); device.focused.push(label); } },
      findNodeHandle: (/** @type {{tag:number}|null} */ node) => node?.tag ?? null,
      Modal: NativeModal,
      BackHandler: { addEventListener: (/** @type {string} */ name, /** @type {()=>boolean} */ callback) => {device.back.add(callback); return {remove(){device.back.delete(callback);}};} },
      Appearance: { getColorScheme: () => device.nativeOverride ?? device.scheme, setColorScheme: (/** @type {"light"|"dark"|"unspecified"} */ value) => {device.nativeOverride=value === "unspecified" ? null : value; for (const notify of device.listeners) notify();}, addChangeListener: (/** @type {()=>void} */ notify) => { device.listeners.add(notify); return {remove(){ device.listeners.delete(notify); }}; } },
      AppState: { get currentState() { return device.appState; }, addEventListener: (/** @type {string} */ event, /** @type {(state:string)=>void} */ callback) => { const notify = () => callback(device.appState); device.foreground.add(notify); return {remove(){ device.foreground.delete(notify); }}; } },
      StyleSheet: { create: (/** @type {unknown} */ value) => value, hairlineWidth: 1 } },
    "react-native-safe-area-context": { SafeAreaProvider: "SafeAreaProvider", SafeAreaView: "SafeAreaView" },
    "expo-status-bar": { StatusBar: "StatusBar" },
    "expo-image": { Image: "FaviconImage" },
    "expo-secure-store": {
      getItemAsync: async (/** @type {string} */ key) => { if (key === "loopaware.mobile.theme" && device.readFailure) throw new Error("controlled-read-failure"); return device.storage.get(key) ?? null; },
      setItemAsync: async (/** @type {string} */ key, /** @type {string} */ value) => {
        if (key === "loopaware.mobile.theme") {
          const write=++device.writeCount;
          if (device.holdWrites) await new Promise(resolve => device.pendingWrites.push(() => resolve(undefined)));
          if (device.writeFailure || (device.failFirstWrite && write === 1)) throw new Error("controlled-write-failure");
        }
        device.storage.set(key, value);
      },
    },
    "expo-constants": { __esModule: true, default: { expoConfig: { extra: { loopAware: { apiBaseUrl: origin, tauthBaseUrl: origin, tauthTenantId: "loopaware" } } } } },
    "expo-crypto": { randomUUID: () => "nonce" },
    "expo-web-browser": { maybeCompleteAuthSession() {} },
    "expo-auth-session": {
      ResponseType: { Code: "code" },
      AuthRequest: class {
        codeVerifier = "pkce-verifier";
        async promptAsync() { await prompt(); return { type: "success", params: { code: "authorization-code" } }; }
      },
      exchangeCodeAsync: async () => ({ idToken: "google-id-token" }),
    },
  };
  /** @type {Map<string, { exports: any }>} */
  const modules = new Map();
  /** @param {string} filename @returns {any} */
  function load(filename) {
    const cached = modules.get(filename);
    if (cached) return cached.exports;
    const module = { exports: {} };
    modules.set(filename, module);
    const compiled = ts.transpileModule(readFileSync(filename, "utf8"), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true }, fileName: filename });
    const execute = new Script(`(function(require, module, exports, fetch, requestAnimationFrame, cancelAnimationFrame, FileReader) { ${compiled.outputText}\n})`, { filename }).runInThisContext();
    execute((/** @type {string} */ name) => {
      if (name === "./assets/icon.png") assetLoadCount += 1;
      if (name in mocks) return mocks[name];
      if (name.startsWith(".")) return load(resolve(dirname(filename), `${name}.ts`));
      return require(name);
    }, module, module.exports, fetcher, (/** @type {()=>void} */ callback) => setTimeout(callback,0), (/** @type {ReturnType<typeof setTimeout>} */ timer) => clearTimeout(timer), class extends ProtocolFileReader { constructor() { super(device.imageReadFailure); } });
    return module.exports;
  }
  return { App: load(resolve(mobileRoot, "App.tsx")).default, assetLoads: () => assetLoadCount };
}
