// @ts-check
import assert from "node:assert/strict";
import { readFileSync, readdirSync, mkdtempSync, mkdirSync, cpSync, writeFileSync, symlinkSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, join, dirname } from "node:path";
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
assert.equal(readFileSync(`${prepared}/ci_scripts/ci_post_clone.sh`, "utf8"), readFileSync("mobile/cloud/ci_post_clone.sh", "utf8"));
assert.ok(project.includes(".xcode.env.local"), "The bundle phase must load its declared cloud Node environment.");
console.info("Prepared Apple inputs contain automatic signing and the cloud dependency hook.");

const require = createRequire(import.meta.url);
const releaseVersion = require("../../mobile/app.config.js").expo.ios.version;
assert.equal(typeof releaseVersion, "string", "The Apple source config must declare its release version.");
assert.match(releaseVersion, /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
const snapshot = JSON.parse(readFileSync("mobile/prepared/app.config.snapshot.json", "utf8")).expo;
assert.equal(snapshot.ios.version, releaseVersion, "The prepared Apple config must match the committed release version.");
const infoPlist = readFileSync(`${prepared}/LoopAware/Info.plist`, "utf8");
assert.equal(/<key>CFBundleShortVersionString<\/key>\s*<string>([^<]+)<\/string>/.exec(infoPlist)?.[1], releaseVersion);
console.info("The prepared Apple version matches the committed release version.");

const fixture = mkdtempSync(join(tmpdir(), "loopaware cloud environment "));
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
    const env = { ...process.env, CI_PRIMARY_REPOSITORY_PATH: fixture, TEST_NODE_PREFIX: dirname(dirname(process.execPath)) };
    delete env.NODE_ENV;
    const hook = spawnSync("/bin/sh", ["-c", `
        brew() { if [ "$1" = --prefix ]; then printf '%s\\n' "$TEST_NODE_PREFIX"; fi; }
        npm() { :; }
        pod() { :; }
        . "$1"
    `, "cloud-hook-test", resolve("mobile/cloud/ci_post_clone.sh")], { env, encoding: "utf8" });
    assert.equal(hook.status, 0, hook.stderr);
    const config = spawnSync("/bin/sh", ["-c", '. "$1"; exec "$NODE_BINARY" "$2" "$3" "$3"', "archive-phase-test",
        join(mobile, "ios/.xcode.env.local"), resolve("mobile/node_modules/expo-constants/scripts/getAppConfig.js"), mobile],
        { cwd: mobile, env, encoding: "utf8" });
    assert.equal(config.status, 0, config.stderr);
    const exported = JSON.parse(readFileSync(join(mobile, "app.config"), "utf8"));
    assert.ok(exported.plugins.includes("./plugins/withStoreBuild.cjs"));
    assert.ok(!exported.plugins.includes("expo-dev-client"));
    console.info("A separate Xcode phase exports the production Expo config without expo-dev-client.");
} finally {
    rmSync(fixture, { recursive: true, force: true });
}
