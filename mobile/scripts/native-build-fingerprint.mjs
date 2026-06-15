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
    ? ["assets/android-icon-foreground.png", "assets/android-icon-monochrome.png"]
    : ["scripts/fix-ios-project-warnings.mjs"];

const hash = crypto.createHash("sha256");
hash.update(`${platform}\n`);
for (const relativePath of [...sharedNativeInputs, ...platformNativeInputs]) {
  const absolutePath = path.join(mobileRoot, relativePath);
  hash.update(`${relativePath}\0`);
  hash.update(fs.readFileSync(absolutePath));
  hash.update("\0");
}

console.log(hash.digest("hex"));
