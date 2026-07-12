#!/usr/bin/env node
// @ts-check
/// <reference types="node" />

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import { createMobileCalVerVersion } from "./mobile-calver-version.mjs";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const defaultMobileDir = path.join(repoRoot, "mobile");
const publishSchema = "loopaware.mobile-android-play-publish.v1";
const bundleSchema = "loopaware.mobile-android-bundle.v1";
const releaseIdentitySchema = "loopaware.mobile-android-release-identity.v1";
const releaseIdentityFileName = "android-release-identity.json";
const androidPublisherApiBase = "https://androidpublisher.googleapis.com/androidpublisher/v3/applications";
const androidPublisherUploadBase = "https://androidpublisher.googleapis.com/upload/androidpublisher/v3/applications";
const androidPublisherScope = "https://www.googleapis.com/auth/androidpublisher";
const defaultTrack = "internal";
const defaultStatus = "completed";

class PublishError extends Error {
  /**
   * @param {string} message
   */
  constructor(message) {
    super(message);
    this.name = "PublishError";
  }
}

try {
  const args = parseArgs(process.argv.slice(2));
  const result = await publishAndroidBundle(args);
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
} catch (error) {
  if (error instanceof PublishError) {
    process.stderr.write(`mobile android play publish failed: ${error.message}\n`);
    process.exit(2);
  }
  throw error;
}

/**
 * @typedef {{
 *   mobileDir: string;
 *   aab: string;
 *   mapping: string;
 *   buildManifest: string;
 *   packageName: string;
 *   quotaProject: string;
 *   track: string;
 *   status: string;
 *   releaseName: string;
 *   versioning: import("./mobile-calver-version.mjs").MobileCalVerVersion;
 *   dryRun: boolean;
 * }} AndroidPublishArgs
 */

/**
 * @param {string[]} argv
 * @returns {AndroidPublishArgs}
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
      throw new PublishError(`unexpected positional argument: ${token}`);
    }
    const equalsIndex = token.indexOf("=");
    if (equalsIndex > 0) {
      options.set(token.slice(2, equalsIndex), token.slice(equalsIndex + 1));
      continue;
    }
    const optionName = token.slice(2);
    const optionValue = argv[index + 1];
    if (!optionValue || optionValue.startsWith("--")) {
      throw new PublishError(`missing value for --${optionName}`);
    }
    options.set(optionName, optionValue);
    index += 1;
  }
  const allowedOptions = new Set(["mobile-dir", "aab", "mapping", "build-manifest", "release-timestamp"]);
  for (const optionName of options.keys()) {
    if (!allowedOptions.has(optionName)) {
      throw new PublishError(`unknown option: --${optionName}`);
    }
  }

  const mobileDir = resolvePath(options.get("mobile-dir") || defaultMobileDir);
  const releaseIdentity = readAndroidReleaseIdentity(mobileDir);
  let versioning;
  try {
    versioning = createMobileCalVerVersion(
      options.get("release-timestamp") ||
        process.env.MOBILE_RELEASE_TIMESTAMP ||
        process.env.LOOPAWARE_MOBILE_RELEASE_TIMESTAMP ||
        "",
    );
  } catch (error) {
    throw new PublishError(error instanceof Error ? error.message : String(error));
  }
  const appConfig = readAndroidAppConfig(mobileDir, versioning);
  const packageName = releaseIdentity.packageName;
  if (packageName !== appConfig.packageName) {
    throw new PublishError(`package name mismatch: app.config.js has ${appConfig.packageName}, publish target is ${packageName}`);
  }
  if (packageName !== releaseIdentity.packageName) {
    throw new PublishError(`package name mismatch: release identity has ${releaseIdentity.packageName}, publish target is ${packageName}`);
  }
  const aab = resolvePath(options.get("aab") || defaultAabPath(mobileDir, appConfig.version));
  const mapping = resolvePath(options.get("mapping") || matchingOutputPath(aab, "-mapping.txt"));
  const buildManifest = resolvePath(options.get("build-manifest") || matchingOutputPath(aab, ".json"));
  const track = defaultTrack;
  const status = defaultStatus;
  const releaseName = appConfig.version;
  const quotaProject = releaseIdentity.googleCloudProjectId;
  requireTrack(track);
  requireReleaseStatus(status);

  return {
    mobileDir,
    aab,
    mapping,
    buildManifest,
    packageName,
    quotaProject,
    track,
    status,
    releaseName,
    versioning,
    dryRun: flags.has("dry-run"),
  };
}

/**
 * @param {AndroidPublishArgs} args
 * @returns {Promise<Record<string, unknown>>}
 */
async function publishAndroidBundle(args) {
  const appConfig = readAndroidAppConfig(args.mobileDir, args.versioning);
  requireFile(args.aab, "Android App Bundle");
  requireFile(args.mapping, "R8 deobfuscation mapping file");
  const buildArtifact = readAndroidBuildManifest(args.buildManifest, args.aab, args.mapping, appConfig, args.versioning);
  if (buildArtifact.androidPackage !== args.packageName) {
    throw new PublishError(`build manifest package mismatch: manifest has ${buildArtifact.androidPackage}, publish target is ${args.packageName}`);
  }

  const plan = {
    schema: publishSchema,
    status: args.dryRun ? "planned" : "submitted",
    androidPackage: args.packageName,
    versionName: buildArtifact.versionName,
    versionCode: buildArtifact.versionCode,
    sourceVersionCode: buildArtifact.sourceVersionCode,
    versionCodeSource: buildArtifact.versionCodeSource,
    buildManifest: args.buildManifest,
    track: args.track,
    releaseName: args.releaseName,
    releaseStatus: args.status,
    versioning: args.versioning,
    aab: args.aab,
    aabSha256: sha256File(args.aab),
    deobfuscationFile: args.mapping,
    deobfuscationSha256: sha256File(args.mapping),
    quotaProject: args.quotaProject,
  };
  if (args.dryRun) {
    await verifyAndroidPublisherAccess(args);
    return {
      ...plan,
      publisherAccess: "verified",
    };
  }

  const token = accessTokenFromApplicationDefaultCredentials();
  const authHeaders = googleAuthHeaders(token, args.quotaProject);
  const edit = await requestJson({
    method: "POST",
    url: publisherUrl(args.packageName, "edits"),
    headers: authHeaders,
    label: "create Android Publisher edit",
  });
  const editId = requireResponseString(edit.id, "edit id");
  let editCommitted = false;
  try {
    const existingBundles = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/bundles`),
      headers: authHeaders,
      label: "inspect existing Android Publisher bundles",
    });
    assertPublishableVersionCode(existingBundles, buildArtifact.versionCode);
    const existingTrack = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/tracks/${encodeURIComponent(args.track)}`),
      headers: authHeaders,
      label: `inspect existing Android Publisher ${args.track} track`,
    });
    const existingTrackState = validatedTrackState(existingTrack);
    const bundle = await requestJson({
      method: "POST",
      url: publisherUploadUrl(args.packageName, `edits/${encodeURIComponent(editId)}/bundles`, { uploadType: "media" }),
      headers: { ...authHeaders, "Content-Type": "application/octet-stream" },
      body: fs.readFileSync(args.aab),
      label: "upload Android App Bundle",
    });
    const uploadedVersionCode = requirePositiveInteger(bundle.versionCode, "uploaded bundle versionCode");
    if (uploadedVersionCode !== buildArtifact.versionCode) {
      throw new PublishError(`uploaded bundle versionCode ${uploadedVersionCode} does not match build manifest ${buildArtifact.versionCode}`);
    }
    assertBundleSha256(bundle, plan.aabSha256, "uploaded Android App Bundle");

    await requestJson({
      method: "POST",
      url: publisherUploadUrl(
        args.packageName,
        `edits/${encodeURIComponent(editId)}/apks/${uploadedVersionCode}/deobfuscationFiles/proguard`,
        { uploadType: "media" },
      ),
      headers: { ...authHeaders, "Content-Type": "application/octet-stream" },
      body: fs.readFileSync(args.mapping),
      label: "upload Android deobfuscation mapping",
    });

    await requestJson({
      method: "PUT",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/tracks/${encodeURIComponent(args.track)}`),
      headers: { ...authHeaders, "Content-Type": "application/json" },
      body: Buffer.from(
        JSON.stringify({
          releases: [
            ...existingTrackState.releases,
            {
              name: args.releaseName,
              versionCodes: [String(uploadedVersionCode)],
              status: args.status,
            },
          ],
        }),
      ),
      label: `update ${args.track} track`,
    });

    const commit = await requestJson({
      method: "POST",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}:commit`, {
        changesInReviewBehavior: "ERROR_IF_IN_REVIEW",
      }),
      headers: authHeaders,
      label: "commit Android Publisher edit",
    });
    editCommitted = true;
    await verifyCommittedAndroidPublication(
      args,
      uploadedVersionCode,
      plan.aabSha256,
      existingTrackState,
      authHeaders,
    );

    return {
      ...plan,
      editId,
      committedEditId: commit.id || editId,
      uploadedVersionCode,
    };
  } catch (error) {
    if (editCommitted) {
      const message = error instanceof Error ? error.message : String(error);
      throw new PublishError(
        `Android Publisher edit ${editId} was committed, but post-publication verification failed: ${message}; inspect Google Play before preparing another single-use versionCode`,
      );
    }
    try {
      await requestJson({
        method: "DELETE",
        url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}`),
        headers: authHeaders,
        label: "delete failed Android Publisher release edit",
      });
    } catch (cleanupError) {
      const originalMessage = error instanceof Error ? error.message : String(error);
      const cleanupMessage = cleanupError instanceof Error ? cleanupError.message : String(cleanupError);
      throw new PublishError(`${originalMessage}; failed edit cleanup: ${cleanupMessage}`);
    }
    throw error;
  }
}

/**
 * Creates and deletes an empty Play edit, writing the unchanged track inside
 * that edit so the dry run proves exact track-update authority without
 * uploading a bundle or committing a live-track change.
 * @param {AndroidPublishArgs} args
 */
async function verifyAndroidPublisherAccess(args) {
  const token = accessTokenFromApplicationDefaultCredentials();
  const authHeaders = googleAuthHeaders(token, args.quotaProject);
  const edit = await requestJson({
    method: "POST",
    url: publisherUrl(args.packageName, "edits"),
    headers: authHeaders,
    label: "verify Android Publisher edit creation",
  });
  const editId = requireResponseString(edit.id, "preflight edit id");
  /** @type {unknown} */
  let verificationError;
  try {
    const track = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/tracks/${encodeURIComponent(args.track)}`),
      headers: authHeaders,
      label: `verify Android Publisher ${args.track} track access`,
    });
    const bundles = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/bundles`),
      headers: authHeaders,
      label: "verify Android Publisher bundle inventory access",
    });
    assertPublishableVersionCode(bundles, args.versioning.androidVersionCode);
    validatedTrackState(track);
    await requestJson({
      method: "PUT",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/tracks/${encodeURIComponent(args.track)}`),
      headers: { ...authHeaders, "Content-Type": "application/json" },
      body: Buffer.from(JSON.stringify(track)),
      label: `verify Android Publisher ${args.track} track update authority`,
    });
  } catch (error) {
    verificationError = error;
  }
  try {
    await requestJson({
      method: "DELETE",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}`),
      headers: authHeaders,
      label: "delete Android Publisher preflight edit",
    });
  } catch (cleanupError) {
    if (verificationError) {
      const originalMessage = verificationError instanceof Error ? verificationError.message : String(verificationError);
      const cleanupMessage = cleanupError instanceof Error ? cleanupError.message : String(cleanupError);
      throw new PublishError(`${originalMessage}; failed edit cleanup: ${cleanupMessage}`);
    }
    throw cleanupError;
  }
  if (verificationError) {
    throw verificationError;
  }
}

/**
 * @param {Record<string, unknown>} payload
 * @param {number} preparedVersionCode
 */
function assertPublishableVersionCode(payload, preparedVersionCode) {
  const bundles = Array.isArray(payload.bundles) ? payload.bundles : [];
  const versionCodes = bundles.map((bundle) => requirePositiveInteger(bundle?.versionCode, "existing bundle versionCode"));
  if (versionCodes.includes(preparedVersionCode)) {
    throw new PublishError(`Android versionCode ${preparedVersionCode} is already present in Google Play`);
  }
  const maximumVersionCode = versionCodes.length ? Math.max(...versionCodes) : 0;
  if (preparedVersionCode <= maximumVersionCode) {
    throw new PublishError(
      `Android versionCode ${preparedVersionCode} must be greater than existing Google Play maximum ${maximumVersionCode}`,
    );
  }
}

/**
 * @param {Record<string, unknown>} track
 * @returns {{ releases: Record<string, unknown>[]; versionCodes: string[] }}
 */
function validatedTrackState(track) {
  const releases = track.releases === undefined ? [] : track.releases;
  if (!Array.isArray(releases)) {
    throw new PublishError("Android Publisher track releases are invalid");
  }
  const retained = new Set();
  /** @type {Record<string, unknown>[]} */
  const preservedReleases = [];
  for (const release of releases) {
    if (!release || typeof release !== "object") {
      throw new PublishError("Android Publisher track contains an invalid release");
    }
    if (release.status !== "completed") {
      throw new PublishError(
        `Android Publisher track contains ${String(release.status || "unknown")} release state; refusing to replace an active/manual rollout`,
      );
    }
    if (!Array.isArray(release.versionCodes)) {
      throw new PublishError("Android Publisher track release has invalid versionCodes");
    }
    for (const rawVersionCode of release.versionCodes) {
      const versionCode = String(requirePositiveInteger(rawVersionCode, "track versionCode"));
      if (retained.has(versionCode)) {
        throw new PublishError(`Android Publisher track repeats versionCode ${versionCode}`);
      }
      retained.add(versionCode);
    }
    preservedReleases.push(JSON.parse(JSON.stringify(release)));
  }
  return {
    releases: preservedReleases,
    versionCodes: [...retained].sort((left, right) => Number(left) - Number(right)),
  };
}

/**
 * @param {Record<string, unknown>} bundle
 * @param {string} expectedSha256
 * @param {string} label
 */
function assertBundleSha256(bundle, expectedSha256, label) {
  const actualSha256 = String(bundle.sha256 || "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(actualSha256) || actualSha256 !== expectedSha256) {
    throw new PublishError(`${label} SHA-256 ${actualSha256 || "<missing>"} does not match prepared ${expectedSha256}`);
  }
}

/**
 * @param {AndroidPublishArgs} args
 * @param {number} versionCode
 * @param {string} expectedSha256
 * @param {{ releases: Record<string, unknown>[]; versionCodes: string[] }} existingTrackState
 * @param {Record<string, string>} authHeaders
 */
async function verifyCommittedAndroidPublication(args, versionCode, expectedSha256, existingTrackState, authHeaders) {
  const edit = await requestJson({
    method: "POST",
    url: publisherUrl(args.packageName, "edits"),
    headers: authHeaders,
    label: "create Android Publisher verification edit",
  });
  const editId = requireResponseString(edit.id, "verification edit id");
  /** @type {unknown} */
  let verificationError;
  try {
    const bundles = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/bundles`),
      headers: authHeaders,
      label: "verify committed Android bundle",
    });
    const matchingBundle = Array.isArray(bundles.bundles)
      ? bundles.bundles.find((bundle) => Number(bundle?.versionCode) === versionCode)
      : undefined;
    if (!matchingBundle || typeof matchingBundle !== "object") {
      throw new PublishError(`committed Android versionCode ${versionCode} is missing from Google Play`);
    }
    assertBundleSha256(matchingBundle, expectedSha256, "committed Android App Bundle");
    const track = await requestJson({
      method: "GET",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}/tracks/${encodeURIComponent(args.track)}`),
      headers: authHeaders,
      label: `verify committed Android Publisher ${args.track} track`,
    });
    const committedTrackState = validatedTrackState(track);
    const committedVersionCodes = new Set(committedTrackState.versionCodes);
    for (const requiredVersionCode of [...existingTrackState.versionCodes, String(versionCode)]) {
      if (!committedVersionCodes.has(requiredVersionCode)) {
        throw new PublishError(`committed Android track is missing retained versionCode ${requiredVersionCode}`);
      }
    }
    const committedReleaseInventory = committedTrackState.releases.map(canonicalJson);
    for (const existingRelease of existingTrackState.releases) {
      const expectedRelease = canonicalJson(existingRelease);
      const matchIndex = committedReleaseInventory.indexOf(expectedRelease);
      if (matchIndex < 0) {
        throw new PublishError("committed Android track changed metadata for an existing release");
      }
      committedReleaseInventory.splice(matchIndex, 1);
    }
    const expectedNewRelease = canonicalJson({
      name: args.releaseName,
      versionCodes: [String(versionCode)],
      status: args.status,
    });
    if (committedReleaseInventory.length !== 1 || committedReleaseInventory[0] !== expectedNewRelease) {
      throw new PublishError("committed Android track does not contain the exact new release without metadata loss");
    }
  } catch (error) {
    verificationError = error;
  }
  try {
    await requestJson({
      method: "DELETE",
      url: publisherUrl(args.packageName, `edits/${encodeURIComponent(editId)}`),
      headers: authHeaders,
      label: "delete Android Publisher verification edit",
    });
  } catch (cleanupError) {
    if (verificationError) {
      throw new PublishError(
        `${verificationError instanceof Error ? verificationError.message : String(verificationError)}; failed verification edit cleanup: ${cleanupError instanceof Error ? cleanupError.message : String(cleanupError)}`,
      );
    }
    throw cleanupError;
  }
  if (verificationError) {
    throw verificationError;
  }
}

/**
 * @param {string} mobileDir
 * @param {import("./mobile-calver-version.mjs").MobileCalVerVersion} versioning
 * @returns {{ version: string; packageName: string }}
 */
function readAndroidAppConfig(mobileDir, versioning) {
  const appConfigPath = path.join(mobileDir, "app.config.js");
  requireFile(appConfigPath, "Expo app config");
  const require = createRequire(import.meta.url);
  const previousVersion = process.env.LOOPAWARE_MOBILE_VERSION;
  const previousVersionCode = process.env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE;
  process.env.LOOPAWARE_MOBILE_VERSION = versioning.releaseVersion;
  process.env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE = String(versioning.androidVersionCode);
  delete require.cache[require.resolve(appConfigPath)];
  try {
    const config = require(appConfigPath);
    const expoConfig = config.expo || {};
    const androidConfig = expoConfig.android || {};
    return {
      version: requireString(expoConfig.version, `expo.version in ${appConfigPath}`),
      packageName: requireString(androidConfig.package, `expo.android.package in ${appConfigPath}`),
    };
  } finally {
    if (previousVersion === undefined) {
      delete process.env.LOOPAWARE_MOBILE_VERSION;
    } else {
      process.env.LOOPAWARE_MOBILE_VERSION = previousVersion;
    }
    if (previousVersionCode === undefined) {
      delete process.env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE;
    } else {
      process.env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE = previousVersionCode;
    }
  }
}

/**
 * @param {string} mobileDir
 * @returns {{ googleCloudProjectId: string; googleCloudProjectNumber: string; packageName: string; webClientId: string; iosClientId: string; androidClientId: string }}
 */
function readAndroidReleaseIdentity(mobileDir) {
  const identityPath = path.join(mobileDir, releaseIdentityFileName);
  requireFile(identityPath, "Android release identity");
  const identity = JSON.parse(fs.readFileSync(identityPath, "utf8"));
  if (identity.schema !== releaseIdentitySchema) {
    throw new PublishError(`invalid Android release identity schema in ${identityPath}`);
  }
  return {
    googleCloudProjectId: requireString(identity.googleCloudProjectId, `googleCloudProjectId in ${identityPath}`),
    googleCloudProjectNumber: requireString(identity.googleCloudProjectNumber, `googleCloudProjectNumber in ${identityPath}`),
    packageName: requireString(identity.packageName, `packageName in ${identityPath}`),
    webClientId: requireString(identity.webClientId, `webClientId in ${identityPath}`),
    iosClientId: requireString(identity.iosClientId, `iosClientId in ${identityPath}`),
    androidClientId: requireString(identity.androidClientId, `androidClientId in ${identityPath}`),
  };
}

/**
 * @param {string} manifestPath
 * @param {string} aabPath
 * @param {string} mappingPath
 * @param {{ version: string; packageName: string }} appConfig
 * @param {import("./mobile-calver-version.mjs").MobileCalVerVersion} expectedVersioning
 * @returns {{ androidPackage: string; versionName: string; versionCode: number; sourceVersionCode: number; versionCodeSource: string }}
 */
function readAndroidBuildManifest(manifestPath, aabPath, mappingPath, appConfig, expectedVersioning) {
  requireFile(manifestPath, "Android bundle build manifest");
  const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
  if (manifest.schema !== bundleSchema) {
    throw new PublishError(`invalid Android bundle build manifest schema in ${manifestPath}`);
  }
  if (manifest.status !== "passed") {
    throw new PublishError(`Android bundle build manifest is not passed: ${manifestPath}`);
  }
  const manifestAab = resolvePath(requireString(manifest.output, `output in ${manifestPath}`));
  const manifestMapping = resolvePath(requireString(manifest.deobfuscationFile, `deobfuscationFile in ${manifestPath}`));
  if (manifestAab !== aabPath) {
    throw new PublishError(`Android bundle build manifest output mismatch: ${manifestAab} != ${aabPath}`);
  }
  if (manifestMapping !== mappingPath) {
    throw new PublishError(`Android bundle build manifest mapping mismatch: ${manifestMapping} != ${mappingPath}`);
  }
  const aabSha256 = sha256File(aabPath);
  const mappingSha256 = sha256File(mappingPath);
  if (manifest.sha256 !== aabSha256) {
    throw new PublishError(`Android App Bundle hash changed since build manifest: ${aabPath}`);
  }
  if (manifest.deobfuscationSha256 !== mappingSha256) {
    throw new PublishError(`R8 deobfuscation mapping hash changed since build manifest: ${mappingPath}`);
  }
  const versionName = requireString(manifest.versionName, `versionName in ${manifestPath}`);
  if (versionName !== appConfig.version) {
    throw new PublishError(`build manifest versionName ${versionName} does not match app.config.js ${appConfig.version}`);
  }
  if (!manifest.versioning || typeof manifest.versioning !== "object") {
    throw new PublishError(`Android bundle build manifest is missing versioning in ${manifestPath}`);
  }
  const versioning = /** @type {Record<string, any>} */ (manifest.versioning);
  const canonicalVersioning = /** @type {Record<string, unknown>} */ (expectedVersioning);
  for (const field of ["releaseTimestamp", "releaseVersion", "buildCode", "iosBuildNumber", "androidVersionCode", "buildCodeSource"]) {
    if (versioning[field] !== canonicalVersioning[field]) {
      throw new PublishError(`Android bundle versioning.${field} does not match the publication release identity`);
    }
  }
  return {
    androidPackage: requireString(manifest.androidPackage, `androidPackage in ${manifestPath}`),
    versionName,
    versionCode: requirePositiveInteger(manifest.versionCode, `versionCode in ${manifestPath}`),
    sourceVersionCode: requirePositiveInteger(manifest.sourceVersionCode, `sourceVersionCode in ${manifestPath}`),
    versionCodeSource: requireString(manifest.versionCodeSource, `versionCodeSource in ${manifestPath}`),
  };
}

/**
 * @returns {string}
 */
function accessTokenFromApplicationDefaultCredentials() {
  const result = spawnSync("gcloud", ["auth", "application-default", "print-access-token"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.error) {
    throw new PublishError(`could not run gcloud for ADC token: ${result.error.message}`);
  }
  if (result.status !== 0) {
    const detail = `${result.stdout || ""}${result.stderr || ""}`.trim();
    throw new PublishError(
      `could not read Google Application Default Credentials token with ${androidPublisherScope}; ` +
        `run gcloud auth application-default login --scopes=${androidPublisherScope},https://www.googleapis.com/auth/cloud-platform` +
        (detail ? `: ${detail}` : ""),
    );
  }
  const token = result.stdout.trim();
  if (!token) {
    throw new PublishError("gcloud returned an empty Application Default Credentials token");
  }
  return token;
}

/**
 * @param {string} token
 * @param {string} quotaProject
 * @returns {Record<string, string>}
 */
function googleAuthHeaders(token, quotaProject) {
  /** @type {Record<string, string>} */
  const headers = {
    Authorization: `Bearer ${token}`,
  };
  if (quotaProject) {
    headers["X-Goog-User-Project"] = quotaProject;
  }
  return headers;
}

/**
 * @param {{ method: string; url: string; headers: Record<string, string>; body?: Buffer; label: string }} request
 * @returns {Promise<Record<string, unknown>>}
 */
async function requestJson(request) {
  const response = await fetch(request.url, {
    method: request.method,
    headers: request.headers,
    body: request.body ? new Uint8Array(request.body) : undefined,
  });
  const body = await response.text();
  if (!response.ok) {
    throw new PublishError(`${request.label} failed with HTTP ${response.status}: ${body.slice(0, 4096)}`);
  }
  if (!body.trim()) {
    return {};
  }
  const payload = JSON.parse(body);
  if (!payload || typeof payload !== "object") {
    throw new PublishError(`${request.label} returned a non-object JSON response`);
  }
  return payload;
}

/**
 * @param {string} packageName
 * @param {string} pathSuffix
 * @returns {string}
 */
function publisherUrl(packageName, pathSuffix, query = {}) {
  const base = `${androidPublisherApiBase}/${encodeURIComponent(packageName)}/${pathSuffix}`;
  const encodedQuery = new URLSearchParams(query).toString();
  return encodedQuery ? `${base}?${encodedQuery}` : base;
}

/**
 * @param {unknown} value
 * @returns {string}
 */
function canonicalJson(value) {
  /**
   * @param {unknown} input
   * @returns {unknown}
   */
  function normalize(input) {
    if (Array.isArray(input)) {
      return input.map(normalize);
    }
    if (input && typeof input === "object") {
      return Object.fromEntries(
        Object.entries(input)
          .sort(([left], [right]) => left.localeCompare(right))
          .map(([key, child]) => [key, normalize(child)]),
      );
    }
    return input;
  }
  return JSON.stringify(normalize(value));
}

/**
 * @param {string} packageName
 * @param {string} pathSuffix
 * @param {Record<string, string>} query
 * @returns {string}
 */
function publisherUploadUrl(packageName, pathSuffix, query) {
  return `${androidPublisherUploadBase}/${encodeURIComponent(packageName)}/${pathSuffix}?${new URLSearchParams(query).toString()}`;
}

/**
 * @param {string} mobileDir
 * @param {string} version
 * @returns {string}
 */
function defaultAabPath(mobileDir, version) {
  const safeVersion = version.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "") || "release";
  return path.join(mobileDir, "dist", `loopaware-${safeVersion}-android-release.aab`);
}

/**
 * @param {string} outputPath
 * @param {string} suffix
 * @returns {string}
 */
function matchingOutputPath(outputPath, suffix) {
  const parsed = path.parse(outputPath);
  return path.join(parsed.dir, `${parsed.name}${suffix}`);
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
 * @param {unknown} value
 * @param {string} label
 * @returns {string}
 */
function requireString(value, label) {
  if (!value || typeof value !== "string") {
    throw new PublishError(`missing ${label}`);
  }
  return value;
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {number}
 */
function requirePositiveInteger(value, label) {
  const numberValue = Number(value);
  if (!Number.isInteger(numberValue) || numberValue <= 0) {
    throw new PublishError(`${label} must be a positive integer`);
  }
  return numberValue;
}

/**
 * @param {unknown} value
 * @param {string} label
 * @returns {string}
 */
function requireResponseString(value, label) {
  if (!value || typeof value !== "string") {
    throw new PublishError(`Android Publisher response is missing ${label}`);
  }
  return value;
}

/**
 * @param {string} filePath
 * @param {string} label
 */
function requireFile(filePath, label) {
  if (!fs.existsSync(filePath) || !fs.statSync(filePath).isFile()) {
    throw new PublishError(`missing ${label}: ${filePath}`);
  }
  if (fs.statSync(filePath).size <= 0) {
    throw new PublishError(`empty ${label}: ${filePath}`);
  }
}

/**
 * @param {string} track
 */
function requireTrack(track) {
  if (!/^[A-Za-z0-9._-]+$/.test(track)) {
    throw new PublishError(`invalid Play track: ${track}`);
  }
}

/**
 * @param {string} status
 */
function requireReleaseStatus(status) {
  if (!new Set(["completed", "draft", "inProgress", "halted"]).has(status)) {
    throw new PublishError(`invalid Play release status: ${status}`);
  }
}

/**
 * @param {string} filePath
 * @returns {string}
 */
function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filePath));
  return hash.digest("hex");
}
