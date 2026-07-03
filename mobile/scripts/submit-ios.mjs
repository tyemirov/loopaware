// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const mobileRoot = path.resolve(repoRoot, "mobile");

const profileName = process.env.MOBILE_IOS_SUBMIT_PROFILE || process.env.MOBILE_SUBMIT_PROFILE || "production";
const ascAppIdFromEnvironment = String(
  process.env.MOBILE_IOS_ASC_APP_ID || process.env.LOOPAWARE_MOBILE_IOS_ASC_APP_ID || "",
).trim();
const easCommand = process.env.MOBILE_EAS || "npx eas-cli";
const submitArgs = String(process.env.MOBILE_SUBMIT_ARGS || "").trim();
const iosSubmitArgs = String(process.env.MOBILE_IOS_SUBMIT_ARGS || "").trim();

const easJSONPath = path.join(mobileRoot, "eas.json");
const easJSONSource = fs.readFileSync(easJSONPath, "utf8");
const easJSON = JSON.parse(easJSONSource);
const iosProfile = easJSON.submit?.[profileName]?.ios;

if (!iosProfile || typeof iosProfile !== "object") {
  fail(`mobile_ios_submit_profile_missing: ${profileName}`);
}

const ascAppIdFromProfile = String(iosProfile.ascAppId || "").trim();
if (ascAppIdFromProfile && ascAppIdFromEnvironment && ascAppIdFromProfile !== ascAppIdFromEnvironment) {
  fail("mobile_ios_submit_asc_app_id_conflict");
}

const ascAppId = ascAppIdFromProfile || ascAppIdFromEnvironment;
if (!/^[1-9][0-9]*$/.test(ascAppId)) {
  fail(
    "mobile_ios_submit_missing_asc_app_id: set MOBILE_IOS_ASC_APP_ID or LOOPAWARE_MOBILE_IOS_ASC_APP_ID to the numeric App Store Connect app id, or commit ascAppId in mobile/eas.json",
  );
}

if (ascAppIdFromProfile) {
  process.exit(runEASSubmit());
} else {
  const tempDirectory = fs.mkdtempSync(path.join(os.tmpdir(), "loopaware-ios-submit-"));
  const backupPath = path.join(tempDirectory, "eas.json");
  fs.writeFileSync(backupPath, easJSONSource);
  iosProfile.ascAppId = ascAppId;
  fs.writeFileSync(easJSONPath, `${JSON.stringify(easJSON, null, 2)}\n`);
  let status = 1;
  try {
    status = runEASSubmit();
  } finally {
    fs.copyFileSync(backupPath, easJSONPath);
    fs.rmSync(tempDirectory, { recursive: true, force: true });
  }
  process.exit(status);
}

function runEASSubmit() {
  const command = [
    easCommand,
    "submit",
    "--platform",
    "ios",
    "--profile",
    shellQuote(profileName),
    "--latest",
    "--non-interactive",
    submitArgs,
    iosSubmitArgs,
  ]
    .filter(Boolean)
    .join(" ");
  const result = spawnSync(command, {
    cwd: mobileRoot,
    env: process.env,
    shell: true,
    stdio: "inherit",
  });
  if (result.error) {
    throw result.error;
  }
  return result.status ?? 1;
}

/**
 * @param {string} value
 * @returns {string}
 */
function shellQuote(value) {
  return `'${String(value).replaceAll("'", "'\\''")}'`;
}

/**
 * @param {string} message
 * @returns {never}
 */
function fail(message) {
  console.error(message);
  process.exit(1);
}
