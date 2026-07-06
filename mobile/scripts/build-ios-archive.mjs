#!/usr/bin/env node
// @ts-check
/// <reference types="node" />

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const defaultMobileDir = path.join(repoRoot, "mobile");
const defaultBuildDir = path.join(os.tmpdir(), "loopaware-mobile-ios-archive");
const archiveSchema = "loopaware.mobile-ios-archive.v1";
const defaultIosDistributionCertificate = "Apple Distribution";

class BuildError extends Error {
  /**
   * @param {string} message
   */
  constructor(message) {
    super(message);
    this.name = "BuildError";
  }
}

try {
  const args = parseArgs(process.argv.slice(2));
  const metadata = buildIOSArchive(args);
  process.stdout.write(`${JSON.stringify(metadata, null, 2)}\n`);
} catch (error) {
  if (error instanceof BuildError) {
    process.stderr.write(`mobile ios archive failed: ${error.message}\n`);
    process.exit(2);
  }
  throw error;
}

/**
 * @typedef {{
 *   mobileDir: string;
 *   buildDir: string;
 *   output: string;
 *   manifest: string;
 *   scheme: string;
 *   configuration: string;
 *   developmentTeam: string;
 *   signingStyle: "automatic" | "manual";
 *   provisioningProfile: string;
 *   signingCertificate: string;
 *   allowProvisioningUpdates: boolean;
 *   ascApiKeyId: string;
 *   ascApiIssuerId: string;
 *   ascApiKeyPath: string;
 *   iosBuildNumber: string;
 *   preflightOnly: boolean;
 *   keepBuildDir: boolean;
 * }} IOSArchiveArgs
 */

/**
 * @param {string[]} argv
 * @returns {IOSArchiveArgs}
 */
function parseArgs(argv) {
  const options = new Map();
  const flags = new Set();
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (["--preflight-only", "--keep-build-dir", "--allow-provisioning-updates", "--no-allow-provisioning-updates"].includes(token)) {
      flags.add(token.slice(2));
      continue;
    }
    if (!token.startsWith("--")) {
      throw new BuildError(`unexpected positional argument: ${token}`);
    }
    const equalsIndex = token.indexOf("=");
    if (equalsIndex > 0) {
      options.set(token.slice(2, equalsIndex), token.slice(equalsIndex + 1));
      continue;
    }
    const optionName = token.slice(2);
    const optionValue = argv[index + 1];
    if (!optionValue || optionValue.startsWith("--")) {
      throw new BuildError(`missing value for --${optionName}`);
    }
    options.set(optionName, optionValue);
    index += 1;
  }

  const signingStyle = String(options.get("signing-style") || process.env.MOBILE_IOS_SIGNING_STYLE || "automatic");
  if (signingStyle !== "automatic" && signingStyle !== "manual") {
    throw new BuildError("--signing-style must be automatic or manual");
  }
  const iosBuildNumber = requirePositiveIntegerString(
    options.get("ios-build-number") ||
      process.env.MOBILE_IOS_BUILD_NUMBER ||
      process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER ||
      "",
    "MOBILE_IOS_BUILD_NUMBER",
  );

  return {
    mobileDir: resolvePath(options.get("mobile-dir") || defaultMobileDir),
    buildDir: resolvePath(options.get("build-dir") || defaultBuildDir),
    output: options.has("output") ? resolvePath(options.get("output") || "") : "",
    manifest: options.has("manifest") ? resolvePath(options.get("manifest") || "") : "",
    scheme: String(options.get("scheme") || ""),
    configuration: String(options.get("configuration") || "Release"),
    developmentTeam: String(
      options.get("development-team") ||
        process.env.MOBILE_IOS_DEVELOPMENT_TEAM ||
        process.env.LOOPAWARE_MOBILE_IOS_DEVELOPMENT_TEAM ||
        process.env.APPLE_TEAM_ID ||
        "",
    ).trim(),
    signingStyle,
    provisioningProfile: String(options.get("provisioning-profile") || process.env.MOBILE_IOS_PROVISIONING_PROFILE || "").trim(),
    signingCertificate: String(options.get("signing-certificate") || process.env.MOBILE_IOS_SIGNING_CERTIFICATE || "").trim(),
    allowProvisioningUpdates:
      flags.has("no-allow-provisioning-updates")
        ? false
        : flags.has("allow-provisioning-updates") || envTruthy("MOBILE_IOS_ALLOW_PROVISIONING_UPDATES"),
    ascApiKeyId: String(options.get("asc-api-key-id") || process.env.APP_STORE_CONNECT_API_KEY_ID || process.env.ASC_API_KEY_ID || "").trim(),
    ascApiIssuerId: String(
      options.get("asc-api-issuer-id") || process.env.APP_STORE_CONNECT_API_ISSUER_ID || process.env.ASC_API_ISSUER_ID || "",
    ).trim(),
    ascApiKeyPath: resolvePath(options.get("asc-api-key-path") || process.env.APP_STORE_CONNECT_API_KEY_PATH || process.env.ASC_API_KEY_PATH || ""),
    iosBuildNumber,
    preflightOnly: flags.has("preflight-only"),
    keepBuildDir: flags.has("keep-build-dir"),
  };
}

/**
 * @param {IOSArchiveArgs} args
 * @returns {Record<string, unknown>}
 */
function buildIOSArchive(args) {
  requireFile(path.join(args.mobileDir, "package.json"), "mobile package.json");
  requireFile(path.join(args.mobileDir, "package-lock.json"), "mobile package-lock.json");
  requireExecutable(which("npm"), "npm");
  requireExecutable(which("npx"), "npx");
  requireExecutable(which("xcodebuild"), "xcodebuild");
  requireExecutable(which("unzip"), "unzip");
  requireExecutable(which("plutil"), "plutil");
  validateSigningInputs(args);
  validateAscKeyInputs(args);

  const appConfig = readAppConfig(args.mobileDir, args.iosBuildNumber);
  const outputPath = args.output || defaultOutputPath(args.mobileDir, appConfig.version);
  const manifestPath = args.manifest || matchingOutputPath(outputPath, ".json");

  if (args.preflightOnly) {
    return {
      schema: archiveSchema,
      status: "preflight-passed",
      bundleIdentifier: appConfig.bundleIdentifier,
      version: appConfig.version,
      buildNumber: args.iosBuildNumber,
      output: outputPath,
      buildManifest: manifestPath,
    };
  }

  if (args.buildDir === "/" || args.buildDir === os.tmpdir()) {
    throw new BuildError(`unsafe build directory: ${args.buildDir}`);
  }
  fs.rmSync(args.buildDir, { recursive: true, force: true });
  const buildMobileDir = path.join(args.buildDir, "mobile");
  copyMobileProject(args.mobileDir, buildMobileDir);

  const env = buildEnvironment(args, appConfig);
  run(["npm", "ci"], { cwd: buildMobileDir, env });
  run(["npx", "--no-install", "expo", "prebuild", "--platform", "ios", "--no-install"], { cwd: buildMobileDir, env });
  run(["node", "scripts/fix-ios-project-warnings.mjs"], { cwd: buildMobileDir, env });
  run(["npx", "--no-install", "pod-install", "ios"], { cwd: buildMobileDir, env });

  const workspace = findWorkspace(path.join(buildMobileDir, "ios"));
  const scheme = args.scheme || appWorkspaceScheme(workspace, appConfig.name);
  const archivePath = path.join(args.buildDir, "archive", `${scheme}.xcarchive`);
  const exportDir = path.join(args.buildDir, "export");
  const derivedDataPath = path.join(args.buildDir, "derived-data");
  const exportOptionsPath = path.join(args.buildDir, "export-options.plist");
  writeExportOptions(exportOptionsPath, args, appConfig);

  const archiveCommand = [
    "xcodebuild",
    "-workspace",
    workspace,
    "-scheme",
    scheme,
    "-configuration",
    args.configuration,
    "-sdk",
    "iphoneos",
    "-destination",
    "generic/platform=iOS",
    "-archivePath",
    archivePath,
    "-derivedDataPath",
    derivedDataPath,
  ];
  appendXcodeAuthArgs(archiveCommand, args);
  archiveCommand.push("clean", "archive", ...archiveBuildSettings(args, appConfig));
  run(archiveCommand, { cwd: buildMobileDir, env });

  const exportCommand = [
    "xcodebuild",
    "-exportArchive",
    "-archivePath",
    archivePath,
    "-exportPath",
    exportDir,
    "-exportOptionsPlist",
    exportOptionsPath,
  ];
  appendXcodeAuthArgs(exportCommand, args);
  run(exportCommand, { cwd: buildMobileDir, env });

  const generatedIPA = findIPA(exportDir);
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.copyFileSync(generatedIPA, outputPath);

  /** @type {Record<string, unknown>} */
  const metadata = {
    ...validateIPA(outputPath, appConfig, args.iosBuildNumber),
    archivePath,
    buildNumber: args.iosBuildNumber,
    bundleIdentifier: appConfig.bundleIdentifier,
    exportMethod: "app-store-connect",
    output: outputPath,
    sizeBytes: fs.statSync(outputPath).size,
    target: "app-store-connect",
    version: appConfig.version,
  };
  metadata.buildManifest = writeBuildManifest(manifestPath, metadata);

  if (!args.keepBuildDir) {
    fs.rmSync(args.buildDir, { recursive: true, force: true });
  }
  return metadata;
}

/**
 * @param {string} mobileDir
 * @param {string} iosBuildNumber
 * @returns {{ name: string; version: string; bundleIdentifier: string }}
 */
function readAppConfig(mobileDir, iosBuildNumber) {
  const appConfigPath = path.join(mobileDir, "app.config.js");
  requireFile(appConfigPath, "Expo app config");
  const require = createRequire(import.meta.url);
  const previousBuildNumber = process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER;
  process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER = iosBuildNumber;
  delete require.cache[require.resolve(appConfigPath)];
  try {
    const config = require(appConfigPath);
    const expoConfig = config.expo || {};
    const iosConfig = expoConfig.ios || {};
    return {
      name: requireString(expoConfig.name, "expo.name"),
      version: requireString(expoConfig.version, "expo.version"),
      bundleIdentifier: requireString(iosConfig.bundleIdentifier, "expo.ios.bundleIdentifier"),
    };
  } finally {
    if (previousBuildNumber === undefined) {
      delete process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER;
    } else {
      process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER = previousBuildNumber;
    }
  }
}

/**
 * @param {IOSArchiveArgs} args
 */
function validateSigningInputs(args) {
  if (!args.developmentTeam) {
    throw new BuildError("iOS archive requires MOBILE_IOS_DEVELOPMENT_TEAM or APPLE_TEAM_ID before xcodebuild signing");
  }
  if (args.signingStyle === "manual" && !args.provisioningProfile) {
    throw new BuildError("manual iOS signing requires MOBILE_IOS_PROVISIONING_PROFILE or --provisioning-profile");
  }
}

/**
 * @param {IOSArchiveArgs} args
 */
function validateAscKeyInputs(args) {
  const values = [args.ascApiKeyId, args.ascApiIssuerId, args.ascApiKeyPath];
  if (!values.some(Boolean)) {
    return;
  }
  if (!values.every(Boolean)) {
    throw new BuildError("App Store Connect API key inputs require APP_STORE_CONNECT_API_KEY_ID, APP_STORE_CONNECT_API_ISSUER_ID, and APP_STORE_CONNECT_API_KEY_PATH");
  }
  requireFile(args.ascApiKeyPath, "App Store Connect API private key");
}

/**
 * @param {IOSArchiveArgs} args
 * @param {{ version: string; bundleIdentifier: string }} appConfig
 * @returns {NodeJS.ProcessEnv}
 */
function buildEnvironment(args, appConfig) {
  return {
    ...process.env,
    CI: "1",
    EXPO_NO_TELEMETRY: "1",
    NODE_ENV: "production",
    LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER: appConfig.bundleIdentifier,
    LOOPAWARE_MOBILE_IOS_BUILD_NUMBER: args.iosBuildNumber,
  };
}

/**
 * @param {IOSArchiveArgs} args
 * @param {{ version: string; bundleIdentifier: string }} appConfig
 * @returns {string[]}
 */
function archiveBuildSettings(args, appConfig) {
  const settings = [
    `PRODUCT_BUNDLE_IDENTIFIER=${appConfig.bundleIdentifier}`,
    `CURRENT_PROJECT_VERSION=${args.iosBuildNumber}`,
    `MARKETING_VERSION=${appConfig.version}`,
    args.signingStyle === "automatic" ? "CODE_SIGN_STYLE=Automatic" : "CODE_SIGN_STYLE=Manual",
    `DEVELOPMENT_TEAM=${args.developmentTeam}`,
  ];
  const signingCertificate = args.signingCertificate || (args.signingStyle === "manual" ? defaultIosDistributionCertificate : "");
  if (args.signingStyle === "manual") {
    settings.push(`PROVISIONING_PROFILE_SPECIFIER=${args.provisioningProfile}`);
  }
  if (signingCertificate) {
    settings.push(`CODE_SIGN_IDENTITY=${signingCertificate}`);
  }
  return settings;
}

/**
 * @param {string} exportOptionsPath
 * @param {IOSArchiveArgs} args
 * @param {{ bundleIdentifier: string }} appConfig
 */
function writeExportOptions(exportOptionsPath, args, appConfig) {
  const provisioningProfiles =
    args.signingStyle === "manual"
      ? `<key>provisioningProfiles</key>
  <dict>
    <key>${escapeXML(appConfig.bundleIdentifier)}</key>
    <string>${escapeXML(args.provisioningProfile)}</string>
  </dict>`
      : "";
  const signingCertificate = args.signingStyle === "manual" && args.signingCertificate ? `<key>signingCertificate</key>
  <string>${escapeXML(args.signingCertificate)}</string>` : "";
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>destination</key>
  <string>export</string>
  <key>distributionBundleIdentifier</key>
  <string>${escapeXML(appConfig.bundleIdentifier)}</string>
  <key>manageAppVersionAndBuildNumber</key>
  <false/>
  <key>method</key>
  <string>app-store-connect</string>
  <key>signingStyle</key>
  <string>${escapeXML(args.signingStyle)}</string>
  <key>stripSwiftSymbols</key>
  <true/>
  <key>teamID</key>
  <string>${escapeXML(args.developmentTeam)}</string>
  <key>uploadSymbols</key>
  <true/>
  ${provisioningProfiles}
  ${signingCertificate}
</dict>
</plist>
`;
  fs.mkdirSync(path.dirname(exportOptionsPath), { recursive: true });
  fs.writeFileSync(exportOptionsPath, xml, "utf8");
}

/**
 * @param {string[]} command
 * @param {IOSArchiveArgs} args
 */
function appendXcodeAuthArgs(command, args) {
  if (args.allowProvisioningUpdates) {
    command.push("-allowProvisioningUpdates");
  }
  if (args.ascApiKeyPath) {
    command.push(
      "-authenticationKeyPath",
      args.ascApiKeyPath,
      "-authenticationKeyID",
      args.ascApiKeyId,
      "-authenticationKeyIssuerID",
      args.ascApiIssuerId,
    );
  }
}

/**
 * @param {string} iosDir
 * @returns {string}
 */
function findWorkspace(iosDir) {
  const workspaces = fs.readdirSync(iosDir).filter((entry) => entry.endsWith(".xcworkspace")).sort();
  if (!workspaces.length) {
    throw new BuildError(`no generated Xcode workspace found under ${iosDir}`);
  }
  return path.join(iosDir, workspaces[0]);
}

/**
 * @param {string} workspace
 * @param {string} appName
 * @returns {string}
 */
function appWorkspaceScheme(workspace, appName) {
  const result = runAndRead(["xcodebuild", "-list", "-json", "-workspace", workspace]);
  const payload = JSON.parse(result);
  const rawSchemes = /** @type {unknown[]} */ (Array.isArray(payload.workspace?.schemes) ? payload.workspace.schemes : []);
  const schemes = rawSchemes.map((scheme) => String(scheme));
  if (schemes.includes(appName)) {
    return appName;
  }
  const appLikeSchemes = schemes.filter((scheme) => !/^(Pods-|React|RCT|Expo|EX|Yoga|hermes|RN|FBLazyVector)/.test(scheme));
  if (appLikeSchemes.length === 1) {
    return appLikeSchemes[0];
  }
  throw new BuildError(`could not identify app scheme in ${workspace}; pass --scheme explicitly. Schemes: ${schemes.join(", ")}`);
}

/**
 * @param {string} exportDir
 * @returns {string}
 */
function findIPA(exportDir) {
  const ipas = fs.readdirSync(exportDir).filter((entry) => entry.endsWith(".ipa")).sort();
  if (!ipas.length) {
    throw new BuildError(`xcodebuild export did not create an IPA under ${exportDir}`);
  }
  return path.join(exportDir, ipas[0]);
}

/**
 * @param {string} ipaPath
 * @param {{ version: string; bundleIdentifier: string }} appConfig
 * @param {string} iosBuildNumber
 * @returns {Record<string, unknown>}
 */
function validateIPA(ipaPath, appConfig, iosBuildNumber) {
  run(["unzip", "-t", ipaPath], { quiet: true });
  const names = runAndRead(["unzip", "-Z1", ipaPath]).split("\n").filter(Boolean);
  const infoEntry = names.find((name) => /^Payload\/[^/]+\.app\/Info\.plist$/.test(name));
  if (!infoEntry) {
    throw new BuildError(`IPA is missing Payload/*.app/Info.plist: ${ipaPath}`);
  }
  const infoBuffer = runAndBuffer(["unzip", "-p", ipaPath, infoEntry]);
  const plist = JSON.parse(runAndRead(["plutil", "-convert", "json", "-o", "-", "-"], { input: infoBuffer }));
  if (plist.CFBundleIdentifier !== appConfig.bundleIdentifier) {
    throw new BuildError(`IPA CFBundleIdentifier is ${plist.CFBundleIdentifier}, expected ${appConfig.bundleIdentifier}`);
  }
  if (String(plist.CFBundleShortVersionString) !== appConfig.version) {
    throw new BuildError("IPA CFBundleShortVersionString does not match mobile/app.config.js expo.version");
  }
  if (String(plist.CFBundleVersion) !== iosBuildNumber) {
    throw new BuildError("IPA CFBundleVersion does not match MOBILE_IOS_BUILD_NUMBER");
  }
  return {
    schema: archiveSchema,
    status: "passed",
    infoPlist: infoEntry,
    iosBundleValidated: true,
    sha256: sha256File(ipaPath),
    zipIntegrity: "passed",
  };
}

/**
 * @param {string} outputPath
 * @param {Record<string, unknown>} metadata
 * @returns {string}
 */
function writeBuildManifest(outputPath, metadata) {
  const payload = {
    schema: archiveSchema,
    status: "passed",
    createdAt: new Date().toISOString(),
    app: {
      bundleIdentifier: metadata.bundleIdentifier,
      version: metadata.version,
      buildNumber: metadata.buildNumber,
    },
    export: {
      method: metadata.exportMethod,
      target: metadata.target,
    },
    ipa: {
      path: metadata.output,
      sha256: metadata.sha256,
      sizeBytes: metadata.sizeBytes,
    },
  };
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8");
  return outputPath;
}

/**
 * @param {string} mobileDir
 * @param {string} version
 * @returns {string}
 */
function defaultOutputPath(mobileDir, version) {
  const safeVersion = version.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "") || "release";
  return path.join(mobileDir, "dist", `loopaware-${safeVersion}-ios-app-store-connect.ipa`);
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
 * @param {string} source
 * @param {string} destination
 */
function copyMobileProject(source, destination) {
  fs.mkdirSync(destination, { recursive: true });
  const ignoredNames = new Set(["node_modules", ".expo", "dist", "android", "ios"]);
  for (const entry of fs.readdirSync(source, { withFileTypes: true })) {
    if (ignoredNames.has(entry.name)) {
      continue;
    }
    fs.cpSync(path.join(source, entry.name), path.join(destination, entry.name), { recursive: true, verbatimSymlinks: true });
  }
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
    return os.homedir();
  }
  if (value.startsWith("~/")) {
    return path.join(os.homedir(), value.slice(2));
  }
  return path.resolve(value);
}

/**
 * @param {string | unknown} value
 * @param {string} label
 * @returns {string}
 */
function requireString(value, label) {
  if (!value || typeof value !== "string") {
    throw new BuildError(`missing ${label}`);
  }
  return value;
}

/**
 * @param {string | unknown} value
 * @param {string} label
 * @returns {string}
 */
function requirePositiveIntegerString(value, label) {
  const normalizedValue = String(value || "").trim();
  if (!/^[1-9][0-9]*$/.test(normalizedValue)) {
    throw new BuildError(`${label} must be a positive integer`);
  }
  return normalizedValue;
}

/**
 * @param {string} value
 * @returns {string}
 */
function escapeXML(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&apos;");
}

/**
 * @param {string} name
 * @returns {boolean}
 */
function envTruthy(name) {
  return ["1", "true", "yes", "on"].includes(String(process.env[name] || "").trim().toLowerCase());
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
 * @param {string} filePath
 * @param {string} label
 */
function requireFile(filePath, label) {
  if (!fs.existsSync(filePath) || !fs.statSync(filePath).isFile()) {
    throw new BuildError(`missing ${label}: ${filePath}`);
  }
  if (fs.statSync(filePath).size <= 0) {
    throw new BuildError(`empty ${label}: ${filePath}`);
  }
}

/**
 * @param {string} executablePath
 * @param {string} label
 */
function requireExecutable(executablePath, label) {
  if (!executablePath) {
    throw new BuildError(`missing required executable: ${label}`);
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
 * @param {string[]} command
 * @param {{ cwd?: string; env?: NodeJS.ProcessEnv; quiet?: boolean }} [options]
 */
function run(command, options = {}) {
  process.stdout.write(`+ ${command.join(" ")}${options.cwd ? ` in ${options.cwd}` : ""}\n`);
  const result = spawnSync(command[0], command.slice(1), {
    cwd: options.cwd,
    env: options.env,
    stdio: options.quiet ? "pipe" : "inherit",
    encoding: "utf8",
  });
  if (result.error) {
    throw new BuildError(`command failed: ${result.error.message}`);
  }
  if (result.status !== 0) {
    if (options.quiet) {
      process.stderr.write(`${result.stdout || ""}${result.stderr || ""}`);
    }
    throw new BuildError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${command.join(" ")}`);
  }
}

/**
 * @param {string[]} command
 * @param {{ input?: Buffer }} [options]
 * @returns {string}
 */
function runAndRead(command, options = {}) {
  const result = spawnSync(command[0], command.slice(1), {
    input: options.input,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (result.error) {
    throw new BuildError(`command failed: ${result.error.message}`);
  }
  if (result.status !== 0) {
    throw new BuildError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${command.join(" ")} ${result.stderr || ""}`);
  }
  return result.stdout;
}

/**
 * @param {string[]} command
 * @returns {Buffer}
 */
function runAndBuffer(command) {
  const result = spawnSync(command[0], command.slice(1), { stdio: "pipe" });
  if (result.error) {
    throw new BuildError(`command failed: ${result.error.message}`);
  }
  if (result.status !== 0) {
    throw new BuildError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${command.join(" ")}`);
  }
  return result.stdout;
}
