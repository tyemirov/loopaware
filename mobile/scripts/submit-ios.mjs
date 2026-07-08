#!/usr/bin/env node
// @ts-check
/// <reference types="node" />

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { createMobileCalVerVersion } from "./mobile-calver-version.mjs";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const defaultMobileDir = path.join(repoRoot, "mobile");
const archiveSchema = "loopaware.mobile-ios-archive.v1";
const submitSchema = "loopaware.mobile-ios-app-store-connect-submit.v1";
const defaultAppPasswordEnv = "MOBILE_IOS_APP_SPECIFIC_PASSWORD";

class SubmitError extends Error {
  /**
   * @param {string} message
   */
  constructor(message) {
    super(message);
    this.name = "SubmitError";
  }
}

try {
  const args = parseArgs(process.argv.slice(2));
  const result = submitIOSArchive(args);
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
} catch (error) {
  if (error instanceof SubmitError) {
    process.stderr.write(`mobile ios submit failed: ${error.message}\n`);
    process.exit(2);
  }
  throw error;
}

/**
 * @typedef {{
 *   mobileDir: string;
 *   manifest: string;
 *   ipa: string;
 *   ascApiKeyId: string;
 *   ascApiIssuerId: string;
 *   ascApiKeyPath: string;
 *   appleId: string;
 *   appPasswordEnv: string;
 *   providerPublicId: string;
 *   versioning: import("./mobile-calver-version.mjs").MobileCalVerVersion;
 *   dryRun: boolean;
 * }} IOSSubmitArgs
 */

/**
 * @param {string[]} argv
 * @returns {IOSSubmitArgs}
 */
function parseArgs(argv) {
  const options = new Map();
  const flags = new Set();
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--dry-run") {
      flags.add("dry-run");
      continue;
    }
    if (!token.startsWith("--")) {
      throw new SubmitError(`unexpected positional argument: ${token}`);
    }
    const equalsIndex = token.indexOf("=");
    if (equalsIndex > 0) {
      options.set(token.slice(2, equalsIndex), token.slice(equalsIndex + 1));
      continue;
    }
    const optionName = token.slice(2);
    const optionValue = argv[index + 1];
    if (!optionValue || optionValue.startsWith("--")) {
      throw new SubmitError(`missing value for --${optionName}`);
    }
    options.set(optionName, optionValue);
    index += 1;
  }

  const mobileDir = resolvePath(options.get("mobile-dir") || defaultMobileDir);
  let versioning;
  try {
    versioning = createMobileCalVerVersion(
      options.get("release-timestamp") ||
        process.env.MOBILE_RELEASE_TIMESTAMP ||
        process.env.LOOPAWARE_MOBILE_RELEASE_TIMESTAMP ||
        "",
    );
  } catch (error) {
    throw new SubmitError(error instanceof Error ? error.message : String(error));
  }
  return {
    mobileDir,
    manifest: resolvePath(options.get("manifest") || defaultManifestPath(mobileDir, versioning.releaseVersion)),
    ipa: resolvePath(options.get("ipa") || ""),
    ascApiKeyId: String(options.get("asc-api-key-id") || process.env.APP_STORE_CONNECT_API_KEY_ID || process.env.ASC_API_KEY_ID || "").trim(),
    ascApiIssuerId: String(
      options.get("asc-api-issuer-id") || process.env.APP_STORE_CONNECT_API_ISSUER_ID || process.env.ASC_API_ISSUER_ID || "",
    ).trim(),
    ascApiKeyPath: resolvePath(options.get("asc-api-key-path") || process.env.APP_STORE_CONNECT_API_KEY_PATH || process.env.ASC_API_KEY_PATH || ""),
    appleId: String(options.get("apple-id") || process.env.MOBILE_IOS_APPLE_ID || process.env.APPLE_ID || "").trim(),
    appPasswordEnv: String(options.get("app-password-env") || defaultAppPasswordEnv),
    providerPublicId: String(options.get("provider-public-id") || process.env.MOBILE_IOS_PROVIDER_PUBLIC_ID || "").trim(),
    versioning,
    dryRun: flags.has("dry-run"),
  };
}

/**
 * @param {IOSSubmitArgs} args
 * @returns {Record<string, unknown>}
 */
function submitIOSArchive(args) {
  requireExecutable(which("xcrun"), "xcrun");
  const manifest = readArchiveManifest(args.manifest);
  const ipaPath = args.ipa || String(manifest.ipa.path);
  requireFile(ipaPath, "App Store Connect IPA");
  const ipaSha256 = sha256File(ipaPath);
  if (ipaSha256 !== manifest.ipa.sha256) {
    throw new SubmitError(`iOS IPA hash changed since build manifest: ${ipaPath}`);
  }
  validateUploadInputs(args);

  const plan = {
    schema: submitSchema,
    status: args.dryRun ? "planned" : "submitted",
    bundleIdentifier: manifest.app.bundleIdentifier,
    version: manifest.app.version,
    buildNumber: manifest.app.buildNumber,
    buildManifest: args.manifest,
    ipa: ipaPath,
    ipaSha256,
    versioning: manifest.versioning,
  };
  if (args.dryRun) {
    return plan;
  }

  const command = ["xcrun", "altool", "--upload-package", ipaPath];
  if (args.ascApiKeyId) {
    command.push("--api-key", args.ascApiKeyId, "--api-issuer", args.ascApiIssuerId, "--p8-file-path", args.ascApiKeyPath);
  } else {
    command.push("-u", args.appleId, "-p", `@env:${args.appPasswordEnv}`);
  }
  if (args.providerPublicId) {
    command.push("--provider-public-id", args.providerPublicId);
  }
  run(command);
  return {
    ...plan,
    tool: "xcrun altool",
  };
}

/**
 * @param {string} manifestPath
 * @returns {{ app: { bundleIdentifier: string; version: string; buildNumber: string }; ipa: { path: string; sha256: string; sizeBytes: number }; versioning: Record<string, unknown> }}
 */
function readArchiveManifest(manifestPath) {
  requireFile(manifestPath, "iOS archive build manifest");
  const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
  if (manifest.schema !== archiveSchema) {
    throw new SubmitError(`invalid iOS archive build manifest schema in ${manifestPath}`);
  }
  if (manifest.status !== "passed") {
    throw new SubmitError(`iOS archive build manifest is not passed: ${manifestPath}`);
  }
  if (!manifest.app || typeof manifest.app !== "object") {
    throw new SubmitError("iOS archive build manifest is missing app metadata");
  }
  if (!manifest.ipa || typeof manifest.ipa !== "object") {
    throw new SubmitError("iOS archive build manifest is missing IPA metadata");
  }
  const app = manifest.app;
  const ipa = manifest.ipa;
  const appMetadata = {
    bundleIdentifier: requireString(app.bundleIdentifier, "app.bundleIdentifier"),
    version: requireString(app.version, "app.version"),
    buildNumber: requirePositiveIntegerString(app.buildNumber, "app.buildNumber"),
  };
  return {
    app: appMetadata,
    ipa: {
      path: resolvePath(requireString(ipa.path, "ipa.path")),
      sha256: requireSHA256(ipa.sha256, "ipa.sha256"),
      sizeBytes: requirePositiveInteger(ipa.sizeBytes, "ipa.sizeBytes"),
    },
    versioning: requireArchiveVersioning(manifest.versioning, appMetadata),
  };
}

/**
 * @param {unknown} value
 * @param {{ version: string; buildNumber: string }} app
 * @returns {Record<string, unknown>}
 */
function requireArchiveVersioning(value, app) {
  if (!value || typeof value !== "object") {
    throw new SubmitError("iOS archive build manifest is missing versioning metadata");
  }
  const versioning = /** @type {Record<string, unknown>} */ (value);
  const releaseVersion = requireString(versioning.releaseVersion, "versioning.releaseVersion");
  const iosBuildNumber = requirePositiveIntegerString(versioning.iosBuildNumber, "versioning.iosBuildNumber");
  if (releaseVersion !== app.version) {
    throw new SubmitError(`iOS archive manifest versioning.releaseVersion is ${releaseVersion}, expected ${app.version}`);
  }
  if (iosBuildNumber !== app.buildNumber) {
    throw new SubmitError(`iOS archive manifest versioning.iosBuildNumber is ${iosBuildNumber}, expected ${app.buildNumber}`);
  }
  return {
    releaseTimestamp: requireString(versioning.releaseTimestamp, "versioning.releaseTimestamp"),
    releaseVersion,
    buildCode: requirePositiveInteger(versioning.buildCode, "versioning.buildCode"),
    iosBuildNumber,
    androidVersionCode: requirePositiveInteger(versioning.androidVersionCode, "versioning.androidVersionCode"),
    buildCodeSource: requireString(versioning.buildCodeSource, "versioning.buildCodeSource"),
  };
}

/**
 * @param {IOSSubmitArgs} args
 */
function validateUploadInputs(args) {
  const ascValues = [args.ascApiKeyId, args.ascApiIssuerId, args.ascApiKeyPath];
  if (ascValues.some(Boolean)) {
    if (!ascValues.every(Boolean)) {
      throw new SubmitError("App Store Connect API key upload requires APP_STORE_CONNECT_API_KEY_ID, APP_STORE_CONNECT_API_ISSUER_ID, and APP_STORE_CONNECT_API_KEY_PATH");
    }
    requireFile(args.ascApiKeyPath, "App Store Connect API private key");
    return;
  }
  if (args.appleId && process.env[args.appPasswordEnv]) {
    return;
  }
  throw new SubmitError(
    "iOS upload requires either App Store Connect API key inputs " +
      "(APP_STORE_CONNECT_API_KEY_ID, APP_STORE_CONNECT_API_ISSUER_ID, APP_STORE_CONNECT_API_KEY_PATH) " +
      `or MOBILE_IOS_APPLE_ID/APPLE_ID plus an app-specific password in ${args.appPasswordEnv}`,
  );
}

/**
 * @param {string} mobileDir
 * @param {string} version
 * @returns {string}
 */
function defaultManifestPath(mobileDir, version) {
  const safeVersion = version.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "") || "release";
  return path.join(mobileDir, "dist", `loopaware-${safeVersion}-ios-app-store-connect.json`);
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {string}
 */
function requireString(value, label) {
  if (!value || typeof value !== "string") {
    throw new SubmitError(`missing ${label}`);
  }
  return value;
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {string}
 */
function requireSHA256(value, label) {
  const text = requireString(value, label);
  if (!/^[0-9a-f]{64}$/.test(text)) {
    throw new SubmitError(`${label} must be a lowercase sha256 hex digest`);
  }
  return text;
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {number}
 */
function requirePositiveInteger(value, label) {
  const numberValue = Number(value);
  if (!Number.isInteger(numberValue) || numberValue <= 0) {
    throw new SubmitError(`${label} must be a positive integer`);
  }
  return numberValue;
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {string}
 */
function requirePositiveIntegerString(value, label) {
  const text = requireString(value, label);
  if (!/^[1-9][0-9]*$/.test(text)) {
    throw new SubmitError(`${label} must be a positive integer string`);
  }
  return text;
}

/**
 * @param {string} filePath
 * @param {string} label
 */
function requireFile(filePath, label) {
  if (!fs.existsSync(filePath) || !fs.statSync(filePath).isFile()) {
    throw new SubmitError(`missing ${label}: ${filePath}`);
  }
  if (fs.statSync(filePath).size <= 0) {
    throw new SubmitError(`empty ${label}: ${filePath}`);
  }
}

/**
 * @param {string} executablePath
 * @param {string} label
 */
function requireExecutable(executablePath, label) {
  if (!executablePath) {
    throw new SubmitError(`missing required executable: ${label}`);
  }
}

/**
 * @param {string} name
 * @returns {string}
 */
function which(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  return result.status === 0 ? result.stdout.trim() : "";
}

/**
 * @param {string} value
 * @returns {string}
 */
function resolvePath(value) {
  if (!value) {
    return "";
  }
  if (value === "~") {
    return process.env.HOME || "";
  }
  if (value.startsWith("~/")) {
    return path.join(process.env.HOME || "", value.slice(2));
  }
  return path.resolve(value);
}

/**
 * @param {string} pathToHash
 * @returns {string}
 */
function sha256File(pathToHash) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(pathToHash));
  return hash.digest("hex");
}

/**
 * @param {string[]} command
 */
function run(command) {
  process.stdout.write(`+ ${command.join(" ")}\n`);
  const result = spawnSync(command[0], command.slice(1), {
    env: process.env,
    stdio: "inherit",
    encoding: "utf8",
  });
  if (result.error) {
    throw new SubmitError(`command failed: ${result.error.message}`);
  }
  if (result.status !== 0) {
    throw new SubmitError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${command.join(" ")}`);
  }
}
