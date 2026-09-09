// @ts-check
/// <reference types="node" />

import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const mobileRoot = path.resolve(import.meta.dirname, "..");
const androidRoot = path.join(mobileRoot, "android");
const localPropertiesPath = path.join(androidRoot, "local.properties");
const requiredAndroidFiles = ["settings.gradle", "gradlew", "app/build.gradle"];

if (fs.existsSync(androidRoot) && !requiredAndroidFiles.every((relativePath) => fs.existsSync(path.join(androidRoot, relativePath)))) {
  fs.rmSync(androidRoot, { recursive: true, force: true });
}

const expoExecutable = path.join(mobileRoot, "node_modules", ".bin", "expo");
const prebuildResult = spawnSync(expoExecutable, ["prebuild", "--platform", "android", "--no-install"], {
  cwd: mobileRoot,
  env: process.env,
  stdio: "inherit",
});

if (prebuildResult.error) {
  throw new Error(`android_prebuild_failed: ${prebuildResult.error.message}`);
}

if (prebuildResult.status !== 0) {
  throw new Error(`android_prebuild_failed: exit status ${String(prebuildResult.status ?? prebuildResult.signal ?? "unknown")}`);
}

writeLocalProperties();

function writeLocalProperties() {
  const androidSdkRoot = process.env.ANDROID_SDK_ROOT || process.env.ANDROID_HOME || "";
  if (!androidSdkRoot) {
    throw new Error("android_sdk_root_missing: set ANDROID_SDK_ROOT or ANDROID_HOME");
  }

  fs.mkdirSync(androidRoot, { recursive: true });
  fs.writeFileSync(localPropertiesPath, `sdk.dir=${androidSdkRoot}\n`);
}
