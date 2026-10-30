// @ts-check
import assert from "node:assert/strict";
import { cpSync, mkdtempSync, readFileSync, rmSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { execFileSync } from "node:child_process";

const mobile = resolve("mobile");
const fixture = mkdtempSync(join(tmpdir(), "loopaware-ios-scenes-"));
try {
    for (const name of ["app.config.js", "package.json", "package-lock.json", "plugins", "assets"]) {
        cpSync(join(mobile, name), join(fixture, name), { recursive: true });
    }
    symlinkSync(join(mobile, "node_modules"), join(fixture, "node_modules"), "dir");
    execFileSync(process.execPath, [join(mobile, "node_modules/expo/bin/cli"), "prebuild", "--platform", "ios", "--no-install"], {
        cwd: fixture, env: { ...process.env, CI: "1", NODE_ENV: "production", EXPO_NO_DOTENV: "1" }, stdio: "inherit",
    });
    const plist = readFileSync(join(fixture, "ios/LoopAware/Info.plist"), "utf8");
    const delegate = readFileSync(join(fixture, "ios/LoopAware/AppDelegate.swift"), "utf8");
    assert.match(plist, /<key>ITSAppUsesNonExemptEncryption<\/key>\s*<false\s*\/>/, "Generated iOS application must declare its exempt system encryption use.");
    assert.match(plist, /<key>UIApplicationSceneManifest<\/key>/, "Generated iOS application must adopt the scene lifecycle.");
    assert.match(plist, /<key>UISceneDelegateClassName<\/key>\s*<string>EXExpoAppSceneDelegate<\/string>/);
    assert.match(delegate, /class AppDelegate: ExpoAppDelegate, ExpoReactNativeFactoryProvider/);
    assert.doesNotMatch(delegate, /UIWindow\(frame: UIScreen\.main\.bounds\)|factory\.startReactNative\(/);
    const properties = JSON.parse(readFileSync(join(fixture, "ios/Podfile.properties.json"), "utf8"));
    assert.equal(properties["ios.buildReactNativeFromSource"], "true");
    assert.equal(properties.EXPO_USE_PRECOMPILED_MODULES, "false");
    console.info("Expo generates the iOS scene lifecycle and preserves native source dependencies.");
} finally {
    rmSync(fixture, { recursive: true, force: true });
}
