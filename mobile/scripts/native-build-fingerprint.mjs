// @ts-check
/// <reference types="node" />

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

const mobileRoot = path.resolve(import.meta.dirname, "..");
const platform = process.argv[2] === "android" ? "android" : "ios";
const sharedNativeInputs = [
  "app.config.js",
  "package-lock.json",
  "assets/icon.png",
  "assets/splash-icon.png",
];
const platformNativeInputs =
  platform === "android"
    ? ["assets/android-icon-foreground.png", "assets/android-icon-monochrome.png", "scripts/prepare-android-project.mjs"]
    : ["scripts/fix-ios-project-warnings.mjs"];
const nativeEnvironmentInputs =
  platform === "android"
    ? [["LOOPAWARE_MOBILE_ANDROID_PACKAGE", environmentValue("LOOPAWARE_MOBILE_ANDROID_PACKAGE", "com.mprlab.loopaware")]]
    : [
        ["LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER", environmentValue("LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER", "com.mprlab.loopaware")],
        [
          "LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI",
          environmentValue("LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI", environmentValue("TAUTH_TENANT_GOOGLE_IOS_REDIRECT_URI_LOOPAWARE")),
        ],
        [
          "LOOPAWARE_MOBILE_GOOGLE_IOS_CLIENT_ID",
          environmentValue("LOOPAWARE_MOBILE_GOOGLE_IOS_CLIENT_ID", environmentValue("TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE")),
        ],
      ];

const hash = crypto.createHash("sha256");
hash.update(`${platform}\n`);
for (const relativePath of [...sharedNativeInputs, ...platformNativeInputs]) {
  const absolutePath = path.join(mobileRoot, relativePath);
  hash.update(`${relativePath}\0`);
  hash.update(fs.readFileSync(absolutePath));
  hash.update("\0");
}
for (const [environmentName, environmentInput] of nativeEnvironmentInputs) {
  hash.update(`${environmentName}\0`);
  hash.update(environmentInput);
  hash.update("\0");
}

console.log(hash.digest("hex"));

/**
 * @param {string} name
 * @param {string} [fallbackValue]
 * @returns {string}
 */
function environmentValue(name, fallbackValue = "") {
  return String(process.env[name] || fallbackValue).trim();
}
