// @ts-check
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { resolve, dirname } from "node:path";
import test from "node:test";

const mobile = resolve(import.meta.dirname, "../prepared");
const require = createRequire(import.meta.url);
const project = require("xcode").project(resolve(mobile, "ios/LoopAware.xcodeproj/project.pbxproj"));
project.parseSync();
const phases = Object.entries(project.hash.project.objects.PBXShellScriptBuildPhase)
  .filter(([id, phase]) => !id.endsWith("_comment") &&
    phase.name === '"[Expo Dev Launcher] Strip Local Network Keys for Release"');
assert.equal(phases.length, 1);
const script = JSON.parse(phases[0][1].shellScript);
const nativeOptions = { skip: process.platform !== "darwin" && "Requires the Xcode macOS toolchain" };
const plistRelativePath = "App Space.app/Info.plist";
const developmentDescription = "Expo Dev Launcher uses the local network to discover and connect to development servers running on your computer.";

/** @param {(fixture: string, plist: string) => void} run */
function withFixture(run) {
  const fixture = mkdtempSync(resolve(tmpdir(), "loopaware-metadata-"));
  try {
    const plist = resolve(fixture, plistRelativePath);
    mkdirSync(dirname(plist), { recursive: true });
    run(fixture, plist);
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
}

/** @param {string} fixture @param {string} configuration */
function runPhase(fixture, configuration) {
  const result = spawnSync("/bin/sh", ["-c", script], {
    env: { ...process.env, CONFIGURATION: configuration, PROJECT_DIR: resolve(mobile, "ios"),
      TARGET_BUILD_DIR: fixture, INFOPLIST_PATH: plistRelativePath,
      SDKROOT: "/invalid/inherited/iphoneos/sdk" }, encoding: "utf8",
  });
  assert.ifError(result.error);
  return result;
}

/** @param {string} plist */
function readMetadata(plist) {
  return JSON.parse(execFileSync("/usr/bin/plutil", ["-convert", "json", "-o", "-", plist], { encoding: "utf8" }));
}

test("the generated Release metadata phase rejects a malformed property list", nativeOptions, () => {
  withFixture((fixture, plist) => {
    writeFileSync(plist, "malformed property list\n");
    const result = runPhase(fixture, "Release");
    assert.notEqual(result.status, 0, "Invalid release metadata must stop the build phase");
    assert.match(result.stderr, /Apple release metadata cleanup failed/);
    assert.equal(readFileSync(plist, "utf8"), "malformed property list\n");
  });
});

test("the Release metadata phase rejects an absent built property list", nativeOptions, () => {
  withFixture((fixture) => {
    const result = runPhase(fixture, "Release");
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /Apple release metadata cleanup failed/);
  });
});

test("Release cleanup removes only development metadata and preserves binary property-list values", nativeOptions, () => {
  withFixture((fixture, plist) => {
    writeFileSync(plist, `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>com.mprlab.loopaware</string>
<key>NSBonjourServices</key><array><string>_expo._tcp</string><string>_EXPO._TCP.</string><string>_production._tcp</string><string>_expo._tcp.custom</string></array>
<key>NSLocalNetworkUsageDescription</key><string>A custom description that mentions Expo Dev Launcher.</string>
<key>CustomData</key><data>AQID</data>
<key>CustomDate</key><date>2026-01-02T03:04:05Z</date>
<key>CustomBoolean</key><true/>
</dict></plist>\n`);
    execFileSync("/usr/bin/plutil", ["-convert", "binary1", plist]);
    const result = runPhase(fixture, "Release");
    assert.equal(result.status, 0, result.stderr);
    assert.equal(readFileSync(plist).subarray(0, 8).toString(), "bplist00");
    const extract = (/** @type {string} */ key) => execFileSync("/usr/bin/plutil",
      ["-extract", key, "raw", "-o", "-", plist], { encoding: "utf8" }).trimEnd();
    const services = JSON.parse(execFileSync("/usr/bin/plutil",
      ["-extract", "NSBonjourServices", "json", "-o", "-", plist], { encoding: "utf8" }));
    assert.deepEqual(services, ["_production._tcp", "_expo._tcp.custom"]);
    assert.equal(extract("NSLocalNetworkUsageDescription"), "A custom description that mentions Expo Dev Launcher.");
    assert.equal(extract("CFBundleIdentifier"), "com.mprlab.loopaware");
    assert.equal(extract("CustomBoolean"), "true");
    const xml = execFileSync("/usr/bin/plutil", ["-convert", "xml1", "-o", "-", plist], { encoding: "utf8" });
    assert.match(xml, /<data>\s*AQID\s*<\/data>/);
    assert.match(xml, /<date>2026-01-02T03:04:05Z<\/date>/);
  });
});

test("Release cleanup removes an empty service list and the default development description", nativeOptions, () => {
  withFixture((fixture, plist) => {
    writeFileSync(plist, JSON.stringify({ CFBundleIdentifier: "com.mprlab.loopaware",
      NSBonjourServices: ["_expo._tcp"], NSLocalNetworkUsageDescription: developmentDescription }));
    execFileSync("/usr/bin/plutil", ["-convert", "xml1", plist]);
    const result = runPhase(fixture, "Release");
    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(readMetadata(plist), { CFBundleIdentifier: "com.mprlab.loopaware" });
    const original = readFileSync(plist);
    const repeat = runPhase(fixture, "Release");
    assert.equal(repeat.status, 0, repeat.stderr);
    assert.deepEqual(readFileSync(plist), original);
  });
});

test("Release cleanup rejects invalid metadata types before changing the file", nativeOptions, () => {
  withFixture((fixture, plist) => {
    for (const invalid of [
      { NSBonjourServices: "_expo._tcp" },
      { NSBonjourServices: ["_expo._tcp", 12] },
      { NSBonjourServices: ["_expo._tcp"], NSLocalNetworkUsageDescription: 12 },
      ["invalid root"],
    ]) {
      writeFileSync(plist, JSON.stringify(invalid));
      execFileSync("/usr/bin/plutil", ["-convert", "xml1", plist]);
      const original = readFileSync(plist);
      const result = runPhase(fixture, "Release");
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, /Apple release metadata cleanup failed/);
      assert.deepEqual(readFileSync(plist), original);
    }
  });
});

test("Debug builds preserve development metadata without starting the cleanup tool", nativeOptions, () => {
  withFixture((fixture, plist) => {
    writeFileSync(plist, "unchanged Debug fixture\n");
    const result = runPhase(fixture, "Debug");
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stderr, "");
    assert.equal(readFileSync(plist, "utf8"), "unchanged Debug fixture\n");
  });
});

test("the cleanup phase declares its built metadata dependency", () => {
  assert.ok(phases[0][1].inputPaths.includes('"$(TARGET_BUILD_DIR)/$(INFOPLIST_PATH)"'));
});
