// @ts-check
import assert from "node:assert/strict";
import { readFileSync, readdirSync, mkdtempSync, mkdirSync, cpSync, writeFileSync, symlinkSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
const prepared = "mobile/prepared/ios";
const project = readFileSync(`${prepared}/LoopAware.xcodeproj/project.pbxproj`, "utf8");
const nativeRequire = createRequire(resolve("mobile/package.json"));
const parsedProject = nativeRequire("xcode").project(`${prepared}/LoopAware.xcodeproj/project.pbxproj`);
parsedProject.parseSync();
const targets = parsedProject.pbxNativeTargetSection();
const scheme = readFileSync(`${prepared}/LoopAware.xcodeproj/xcshareddata/xcschemes/LoopAware.xcscheme`, "utf8");
for (const reference of scheme.matchAll(/BlueprintIdentifier\s*=\s*"([^"]+)"/g)) {
    assert.ok(targets[reference[1]], `Shared scheme references absent native target ${reference[1]}`);
}
assert.ok(project.includes("CODE_SIGN_STYLE = Automatic;"), "The prepared Apple target must use automatic signing.");
assert.ok(project.includes(".xcode.env.local"), "The bundle phase must load its declared Node environment.");
console.info("Prepared Apple inputs contain a valid shared scheme and Node environment input.");

const require = createRequire(import.meta.url);
const releaseVersion = require("../../mobile/app.config.js").expo.ios.version;
assert.equal(typeof releaseVersion, "string", "The Apple source config must declare its release version.");
assert.match(releaseVersion, /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
const snapshot = JSON.parse(readFileSync("mobile/prepared/app.config.snapshot.json", "utf8")).expo;
assert.equal(snapshot.ios.version, releaseVersion, "The prepared Apple config must match the committed release version.");
const infoPlist = readFileSync(`${prepared}/LoopAware/Info.plist`, "utf8");
assert.equal(/<key>CFBundleShortVersionString<\/key>\s*<string>([^<]+)<\/string>/.exec(infoPlist)?.[1], '$(MARKETING_VERSION)');
assert.equal(/<key>CFBundleVersion<\/key>\s*<string>([^<]+)<\/string>/.exec(infoPlist)?.[1], '$(CURRENT_PROJECT_VERSION)');
const configurations = Object.entries(parsedProject.pbxXCBuildConfigurationSection())
    .filter(([id, value]) => !id.endsWith('_comment') && String(value.buildSettings?.PRODUCT_BUNDLE_IDENTIFIER).replace(/^"|"$/g, '') === snapshot.ios.bundleIdentifier);
assert.ok(configurations.length > 0);
for (const [, value] of configurations) assert.equal(String(value.buildSettings.MARKETING_VERSION).replace(/^"|"$/g, ''), releaseVersion);
console.info("The prepared Apple metadata uses the native version and build number settings.");

const fixture = mkdtempSync(join(tmpdir(), "loopaware native environment "));
try {
    const mobile = join(fixture, "mobile/prepared");
    mkdirSync(join(mobile, "ios/Pods"), { recursive: true });
    mkdirSync(join(mobile, "scripts"));
    writeFileSync(join(mobile, "scripts/verify-store-preparation.mjs"), "");
    for (const name of ["app.config.js", "package.json", "plugins"]) {
        cpSync(resolve("mobile/prepared", name), join(mobile, name), { recursive: true });
    }
    mkdirSync(join(mobile, "node_modules"));
    for (const name of readdirSync("mobile/node_modules")) {
        if (name === "expo-dev-client") continue;
        symlinkSync(resolve("mobile/node_modules", name), join(mobile, "node_modules", name));
    }
    const env = { ...process.env, NODE_ENV: "production" };
    const config = spawnSync(process.execPath, [
        resolve("mobile/node_modules/expo-constants/scripts/getAppConfig.js"), mobile, mobile],
        { cwd: mobile, env, encoding: "utf8" });
    assert.equal(config.status, 0, config.stderr);
    const exported = JSON.parse(readFileSync(join(mobile, "app.config"), "utf8"));
    assert.ok(exported.plugins.includes("./plugins/withStoreBuild.cjs"));
    assert.ok(!exported.plugins.includes("expo-dev-client"));
    console.info("A separate Xcode phase exports the production Expo config without expo-dev-client.");
} finally {
    rmSync(fixture, { recursive: true, force: true });
}
