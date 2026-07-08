#!/usr/bin/env node
// @ts-check
/// <reference types="node" />

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { createMobileCalVerVersion } from "./mobile-calver-version.mjs";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const defaultMobileDir = path.join(repoRoot, "mobile");
const defaultBuildDir = path.join(os.tmpdir(), "loopaware-mobile-android-aab");
const defaultCredentialDir = path.join(os.homedir(), ".local", "share", "loopaware", "android-upload");
const defaultKeystoreProperties = path.join(defaultCredentialDir, "keystore.properties");
const defaultKeystore = path.join(defaultCredentialDir, "loopaware-upload-key.jks");
const defaultAndroidSdkRoot = path.join(os.homedir(), "Library", "Android", "sdk");
const releaseIdentityFileName = "android-release-identity.json";
const releaseIdentitySchema = "loopaware.mobile-android-release-identity.v1";
const signingEnvPrefix = "LOOPAWARE_ANDROID_UPLOAD";

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
  const metadata = buildAndroidBundle(args);
  process.stdout.write(`${JSON.stringify(metadata, null, 2)}\n`);
} catch (error) {
  if (error instanceof BuildError) {
    process.stderr.write(`mobile android bundle failed: ${error.message}\n`);
    process.exit(2);
  }
  throw error;
}

/**
 * @typedef {{
 *   mobileDir: string;
 *   buildDir: string;
 *   output: string;
 *   keystoreProperties: string;
 *   keystore: string;
 *   javaHome: string;
 *   androidSdkRoot: string;
 *   versioning: import("./mobile-calver-version.mjs").MobileCalVerVersion;
 *   keepBuildDir: boolean;
 * }} BundleArgs
 */

/**
 * @param {string[]} argv
 * @returns {BundleArgs}
 */
function parseArgs(argv) {
  const options = new Map();
  const flags = new Set();
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--keep-build-dir") {
      flags.add("keep-build-dir");
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

  let versioning;
  try {
    versioning = createMobileCalVerVersion(
      options.get("release-timestamp") ||
        process.env.MOBILE_RELEASE_TIMESTAMP ||
        process.env.LOOPAWARE_MOBILE_RELEASE_TIMESTAMP ||
        "",
    );
  } catch (error) {
    throw new BuildError(error instanceof Error ? error.message : String(error));
  }

  return {
    mobileDir: resolvePath(options.get("mobile-dir") || defaultMobileDir),
    buildDir: resolvePath(options.get("build-dir") || defaultBuildDir),
    output: options.has("output") ? resolvePath(options.get("output") || "") : "",
    keystoreProperties: resolvePath(
      options.get("keystore-properties") ||
        process.env.LOOPAWARE_ANDROID_KEYSTORE_PROPERTIES ||
        defaultKeystoreProperties,
    ),
    keystore: resolvePath(options.get("keystore") || process.env.LOOPAWARE_ANDROID_UPLOAD_STORE_FILE || defaultKeystore),
    javaHome: resolveJavaHome(options.get("java-home") || process.env.JAVA_HOME || ""),
    androidSdkRoot: resolvePath(
      options.get("android-sdk-root") ||
        process.env.ANDROID_SDK_ROOT ||
        process.env.ANDROID_HOME ||
        defaultAndroidSdkRoot,
    ),
    versioning,
    keepBuildDir: flags.has("keep-build-dir"),
  };
}

/**
 * @param {BundleArgs} args
 * @returns {Record<string, unknown>}
 */
function buildAndroidBundle(args) {
  requireFile(path.join(args.mobileDir, "package.json"), "mobile package.json");
  requireFile(path.join(args.mobileDir, "package-lock.json"), "mobile package-lock.json");
  requireDirectory(args.androidSdkRoot, "Android SDK root");
  requireExecutable(path.join(args.javaHome, "bin", "java"), "java");
  requireExecutable(path.join(args.javaHome, "bin", "jarsigner"), "jarsigner");
  requireExecutable(path.join(args.javaHome, "bin", "keytool"), "keytool");
  requireExecutable(which("npm"), "npm");
  requireExecutable(which("bundletool"), "bundletool");
  requireExecutable(which("unzip"), "unzip");

  const signing = readSigningProperties(args.keystoreProperties, args.keystore);
  const releaseIdentity = readAndroidReleaseIdentity(args.mobileDir);
  const uploadKeySha256 = verifyUploadKeyFingerprint(signing, args.javaHome, releaseIdentity.uploadKeySha256);
  if (args.buildDir === "/" || args.buildDir === os.tmpdir()) {
    throw new BuildError(`unsafe build directory: ${args.buildDir}`);
  }

  fs.rmSync(args.buildDir, { recursive: true, force: true });
  const buildMobileDir = path.join(args.buildDir, "mobile");
  copyMobileProject(args.mobileDir, buildMobileDir);

  const env = buildEnvironment(args.javaHome, args.androidSdkRoot);
  env.LOOPAWARE_MOBILE_VERSION = args.versioning.releaseVersion;
  env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE = String(args.versioning.androidVersionCode);
  run(["npm", "ci"], { cwd: buildMobileDir, env });
  run(["npx", "expo", "prebuild", "--platform", "android", "--no-install"], { cwd: buildMobileDir, env });
  writeLocalProperties(path.join(buildMobileDir, "android", "local.properties"), args.androidSdkRoot);
  enableReleaseMinification(path.join(buildMobileDir, "android", "gradle.properties"));
  patchReleaseSigning(path.join(buildMobileDir, "android", "app", "build.gradle"));

  /** @type {NodeJS.ProcessEnv} */
  const gradleEnv = { ...env };
  gradleEnv.NODE_ENV = "production";
  gradleEnv[`${signingEnvPrefix}_STORE_FILE`] = signing.storeFile;
  gradleEnv[`${signingEnvPrefix}_STORE_PASSWORD`] = signing.storePassword;
  gradleEnv[`${signingEnvPrefix}_KEY_ALIAS`] = signing.keyAlias;
  gradleEnv[`${signingEnvPrefix}_KEY_PASSWORD`] = signing.keyPassword;
  run(["./gradlew", "--no-daemon", "bundleRelease"], { cwd: path.join(buildMobileDir, "android"), env: gradleEnv });

  const generatedBundle = path.join(buildMobileDir, "android", "app", "build", "outputs", "bundle", "release", "app-release.aab");
  requireFile(generatedBundle, "generated release app bundle");
  const manifest = readBundleManifest(generatedBundle);
  if (manifest.packageName !== releaseIdentity.packageName) {
    throw new BuildError(`generated bundle package ${manifest.packageName} does not match Android release identity ${releaseIdentity.packageName}`);
  }
  if (manifest.versionName !== args.versioning.releaseVersion) {
    throw new BuildError(`generated bundle versionName ${manifest.versionName} does not match CalVer ${args.versioning.releaseVersion}`);
  }
  if (Number(manifest.versionCode) !== args.versioning.androidVersionCode) {
    throw new BuildError(`generated bundle versionCode ${manifest.versionCode} does not match CalVer build code ${args.versioning.androidVersionCode}`);
  }
  const outputPath = args.output || defaultOutputPath(args.mobileDir, manifest.versionName);
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.copyFileSync(generatedBundle, outputPath);
  const mappingOutputPath = copyDeobfuscationFile(buildMobileDir, outputPath);
  const validation = validateBundle(outputPath, args.javaHome, uploadKeySha256);

  if (!args.keepBuildDir) {
    fs.rmSync(args.buildDir, { recursive: true, force: true });
  }

  /** @type {Record<string, unknown>} */
  const metadata = {
    schema: "loopaware.mobile-android-bundle.v1",
    status: "passed",
    androidPackage: manifest.packageName,
    versionName: manifest.versionName,
    versionCode: Number(manifest.versionCode),
    sourceVersionCode: args.versioning.androidVersionCode,
    versionCodeSource: args.versioning.buildCodeSource,
    versionCodePolicy: "CalVer UTC release timestamp seconds since 2020-01-01",
    versioning: args.versioning,
    output: outputPath,
    sha256: sha256File(outputPath),
    sizeBytes: fs.statSync(outputPath).size,
    deobfuscationFile: mappingOutputPath,
    deobfuscationSha256: sha256File(mappingOutputPath),
    keystore: signing.storeFile,
    uploadKeySha256,
    signerOwner: validation.signerOwner,
    signerSha256: validation.signerSha256,
    zipIntegrity: "passed",
    jarSignature: "passed",
    releaseSigner: "passed",
    bundletoolValidated: validation.bundletoolValidated,
    r8Minification: "enabled",
    resourceShrinking: "disabled",
  };
  metadata.buildManifest = writeBuildManifest(outputPath, metadata);
  return metadata;
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
 * @param {string} explicitJavaHome
 * @returns {string}
 */
function resolveJavaHome(explicitJavaHome) {
  const candidates = [
    explicitJavaHome,
    process.env.ANDROID_STUDIO_JAVA_HOME || "",
    "/Applications/Android Studio.app/Contents/jbr/Contents/Home",
    "/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home",
    "/opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home",
  ]
    .filter(Boolean)
    .map(resolvePath);

  for (const candidate of candidates) {
    if (fs.existsSync(path.join(candidate, "bin", "java")) && fs.existsSync(path.join(candidate, "bin", "jarsigner"))) {
      return candidate;
    }
  }

  const javaPath = which("java");
  const jarsignerPath = which("jarsigner");
  if (javaPath && jarsignerPath) {
    return path.dirname(path.dirname(javaPath));
  }

  throw new BuildError(`could not find a JDK with java and jarsigner; searched ${candidates.join(", ")}`);
}

/**
 * @param {string} propertiesPath
 * @param {string} keystorePath
 * @returns {{ storeFile: string; storePassword: string; keyAlias: string; keyPassword: string }}
 */
function readSigningProperties(propertiesPath, keystorePath) {
  requireFile(propertiesPath, "Android upload signing properties");
  const properties = readProperties(propertiesPath);
  const storeFile = resolveKeystorePath(properties.storeFile || keystorePath, propertiesPath);
  const keyAlias = requireProperty(properties, "keyAlias", propertiesPath);
  const storePassword = requireProperty(properties, "storePassword", propertiesPath);
  const keyPassword = requireProperty(properties, "keyPassword", propertiesPath);

  requireFile(storeFile, "Android upload keystore");
  return { storeFile, storePassword, keyAlias, keyPassword };
}

/**
 * @param {string} mobileDir
 * @returns {{ packageName: string; uploadKeySha256: string }}
 */
function readAndroidReleaseIdentity(mobileDir) {
  const identityPath = path.join(mobileDir, releaseIdentityFileName);
  requireFile(identityPath, "Android release identity");
  const identity = JSON.parse(fs.readFileSync(identityPath, "utf8"));
  if (identity.schema !== releaseIdentitySchema) {
    throw new BuildError(`invalid Android release identity schema in ${identityPath}`);
  }
  return {
    packageName: requireString(identity.packageName, `packageName in ${identityPath}`),
    uploadKeySha256: normalizeSHA256Fingerprint(requireString(identity.uploadKey?.sha256, `uploadKey.sha256 in ${identityPath}`)),
  };
}

/**
 * @param {{ storeFile: string; storePassword: string; keyAlias: string }} signing
 * @param {string} javaHome
 * @param {string} expectedSha256
 * @returns {string}
 */
function verifyUploadKeyFingerprint(signing, javaHome, expectedSha256) {
  const certificateOutput = runAndRead(
    [
      path.join(javaHome, "bin", "keytool"),
      "-list",
      "-v",
      "-keystore",
      signing.storeFile,
      "-storepass",
      signing.storePassword,
      "-alias",
      signing.keyAlias,
    ],
    {
      label: `${path.join(javaHome, "bin", "keytool")} -list -v -keystore ${signing.storeFile} -alias ${signing.keyAlias}`,
    },
  );
  const actualSha256 = certificateSHA256FromOutput(certificateOutput, "Android upload keystore certificate");
  if (actualSha256 !== expectedSha256) {
    throw new BuildError(`Android upload key SHA-256 ${actualSha256} does not match release identity ${expectedSha256}`);
  }
  return actualSha256;
}

/**
 * @param {string} rawPath
 * @param {string} propertiesPath
 * @returns {string}
 */
function resolveKeystorePath(rawPath, propertiesPath) {
  const expandedPath = resolvePath(rawPath);
  if (path.isAbsolute(expandedPath)) {
    return expandedPath;
  }
  return path.resolve(path.dirname(propertiesPath), expandedPath);
}

/**
 * @param {Record<string, string>} properties
 * @param {string} key
 * @param {string} propertiesPath
 * @returns {string}
 */
function requireProperty(properties, key, propertiesPath) {
  const value = properties[key] || "";
  if (!value) {
    throw new BuildError(`missing signing property ${key} in ${propertiesPath}`);
  }
  return value;
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
    const sourcePath = path.join(source, entry.name);
    const destinationPath = path.join(destination, entry.name);
    fs.cpSync(sourcePath, destinationPath, { recursive: true, verbatimSymlinks: true });
  }
}

/**
 * @param {string} javaHome
 * @param {string} androidSdkRoot
 * @returns {NodeJS.ProcessEnv}
 */
function buildEnvironment(javaHome, androidSdkRoot) {
  return {
    ...process.env,
    JAVA_HOME: javaHome,
    ANDROID_HOME: androidSdkRoot,
    ANDROID_SDK_ROOT: androidSdkRoot,
    PATH: [
      path.join(javaHome, "bin"),
      path.join(androidSdkRoot, "platform-tools"),
      path.join(androidSdkRoot, "cmdline-tools", "latest", "bin"),
      process.env.PATH || "",
    ]
      .filter(Boolean)
      .join(path.delimiter),
  };
}

/**
 * @param {string} localPropertiesPath
 * @param {string} androidSdkRoot
 */
function writeLocalProperties(localPropertiesPath, androidSdkRoot) {
  fs.mkdirSync(path.dirname(localPropertiesPath), { recursive: true });
  fs.writeFileSync(localPropertiesPath, `sdk.dir=${androidSdkRoot}\n`, "utf8");
}

/**
 * @param {string} gradlePropertiesPath
 */
function enableReleaseMinification(gradlePropertiesPath) {
  const properties = readProperties(gradlePropertiesPath);
  properties["android.enableMinifyInReleaseBuilds"] = "true";
  properties["android.enableShrinkResourcesInReleaseBuilds"] = "false";
  writeProperties(gradlePropertiesPath, properties);
}

/**
 * @param {string} buildGradlePath
 */
function patchReleaseSigning(buildGradlePath) {
  let text = fs.readFileSync(buildGradlePath, "utf8");
  const projectRootLine = "def projectRoot = rootDir.getAbsoluteFile().getParentFile().getAbsolutePath()";
  const signingDefinitions = `
def uploadStoreFile = System.getenv("${signingEnvPrefix}_STORE_FILE") ?: ""
def uploadStorePassword = System.getenv("${signingEnvPrefix}_STORE_PASSWORD") ?: ""
def uploadKeyAlias = System.getenv("${signingEnvPrefix}_KEY_ALIAS") ?: ""
def uploadKeyPassword = System.getenv("${signingEnvPrefix}_KEY_PASSWORD") ?: ""
if (!uploadStoreFile || !uploadStorePassword || !uploadKeyAlias || !uploadKeyPassword) {
    throw new GradleException("LoopAware release signing requires ${signingEnvPrefix}_* environment variables")
}
`.trimEnd();
  if (!text.includes(signingDefinitions)) {
    text = text.replace(projectRootLine, `${projectRootLine}\n${signingDefinitions}`);
  }

  const debugSigningBlock = `    signingConfigs {
        debug {
            storeFile file('debug.keystore')
            storePassword 'android'
            keyAlias 'androiddebugkey'
            keyPassword 'android'
        }
    }`;
  const releaseSigningBlock = `    signingConfigs {
        debug {
            storeFile file('debug.keystore')
            storePassword 'android'
            keyAlias 'androiddebugkey'
            keyPassword 'android'
        }
        release {
            storeFile file(uploadStoreFile)
            storePassword uploadStorePassword
            keyAlias uploadKeyAlias
            keyPassword uploadKeyPassword
        }
    }`;
  if (!text.includes(releaseSigningBlock)) {
    if (!text.includes(debugSigningBlock)) {
      throw new BuildError(`could not find generated signingConfigs block in ${buildGradlePath}`);
    }
    text = text.replace(debugSigningBlock, releaseSigningBlock);
  }

  const debugReleaseLine = "            signingConfig signingConfigs.debug";
  const releaseSigningLine = "            signingConfig signingConfigs.release";
  const buildTypesIndex = text.indexOf("    buildTypes {");
  if (buildTypesIndex === -1) {
    throw new BuildError(`could not find buildTypes block in ${buildGradlePath}`);
  }
  const releaseIndex = text.indexOf("        release {", buildTypesIndex);
  if (releaseIndex === -1) {
    throw new BuildError(`could not find release buildType in ${buildGradlePath}`);
  }
  const releaseEnd = findGradleBlockEnd(text, releaseIndex);
  let releaseBlock = text.slice(releaseIndex, releaseEnd);
  if (!releaseBlock.includes(releaseSigningLine)) {
    if (!releaseBlock.includes(debugReleaseLine)) {
      throw new BuildError(`could not find release signingConfig line in ${buildGradlePath}`);
    }
    releaseBlock = releaseBlock.replace(debugReleaseLine, releaseSigningLine);
    text = text.slice(0, releaseIndex) + releaseBlock + text.slice(releaseEnd);
  }
  const patchedReleaseBlock = text.slice(releaseIndex, releaseIndex + releaseBlock.length);
  if (patchedReleaseBlock.includes(debugReleaseLine) || !patchedReleaseBlock.includes(releaseSigningLine)) {
    throw new BuildError(`release buildType is not configured for upload-key signing in ${buildGradlePath}`);
  }

  fs.writeFileSync(buildGradlePath, text, "utf8");
}

/**
 * @param {string} text
 * @param {number} blockStart
 * @returns {number}
 */
function findGradleBlockEnd(text, blockStart) {
  const braceStart = text.indexOf("{", blockStart);
  if (braceStart === -1) {
    throw new BuildError("could not find Gradle block opening brace");
  }
  let depth = 0;
  for (let index = braceStart; index < text.length; index += 1) {
    const character = text[index];
    if (character === "{") {
      depth += 1;
    } else if (character === "}") {
      depth -= 1;
      if (depth === 0) {
        return index + 1;
      }
    }
  }
  throw new BuildError("could not find Gradle block closing brace");
}

/**
 * @param {string} bundlePath
 * @returns {{ packageName: string; versionCode: string; versionName: string }}
 */
function readBundleManifest(bundlePath) {
  const packageName = runAndRead(["bundletool", "dump", "manifest", `--bundle=${bundlePath}`, "--xpath=/manifest/@package"]).trim();
  const versionCode = runAndRead([
    "bundletool",
    "dump",
    "manifest",
    `--bundle=${bundlePath}`,
    "--xpath=/manifest/@android:versionCode",
  ]).trim();
  const versionName = runAndRead([
    "bundletool",
    "dump",
    "manifest",
    `--bundle=${bundlePath}`,
    "--xpath=/manifest/@android:versionName",
  ]).trim();
  if (!packageName || !versionCode || !versionName) {
    throw new BuildError("could not read package, versionCode, and versionName from generated app bundle");
  }
  return { packageName, versionCode, versionName };
}

/**
 * @param {string} mobileDir
 * @param {string} versionName
 * @returns {string}
 */
function defaultOutputPath(mobileDir, versionName) {
  const safeVersion = versionName.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "") || "release";
  return path.join(mobileDir, "dist", `loopaware-${safeVersion}-android-release.aab`);
}

/**
 * @param {string} buildMobileDir
 * @param {string} outputPath
 * @returns {string}
 */
function copyDeobfuscationFile(buildMobileDir, outputPath) {
  const generatedMapping = path.join(buildMobileDir, "android", "app", "build", "outputs", "mapping", "release", "mapping.txt");
  requireFile(generatedMapping, "R8 deobfuscation mapping file");
  const mappingOutputPath = outputPath.replace(/\.aab$/, "-mapping.txt");
  fs.copyFileSync(generatedMapping, mappingOutputPath);
  if (fs.statSync(mappingOutputPath).size === 0) {
    throw new BuildError(`empty R8 deobfuscation mapping file: ${mappingOutputPath}`);
  }
  return mappingOutputPath;
}

/**
 * @param {string} outputPath
 * @param {Record<string, unknown>} metadata
 * @returns {string}
 */
function writeBuildManifest(outputPath, metadata) {
  const manifestPath = outputPath.replace(/\.aab$/, ".json");
  const payload = {
    ...metadata,
    createdAt: new Date().toISOString(),
  };
  fs.writeFileSync(manifestPath, `${JSON.stringify(payload, null, 2)}\n`, "utf8");
  return manifestPath;
}

/**
 * @param {string} bundlePath
 * @param {string} javaHome
 * @param {string} expectedSignerSha256
 * @returns {{ signerOwner: string; signerSha256: string; bundletoolValidated: boolean }}
 */
function validateBundle(bundlePath, javaHome, expectedSignerSha256) {
  run(["unzip", "-t", bundlePath], { quiet: true });
  run([path.join(javaHome, "bin", "jarsigner"), "-verify", bundlePath], { quiet: true });
  const certificateOutput = runAndRead([path.join(javaHome, "bin", "keytool"), "-printcert", "-jarfile", bundlePath]);
  if (
    certificateOutput.includes("Owner: CN=Android Debug, OU=Android, O=Unknown") ||
    certificateOutput.includes("Issuer: CN=Android Debug, OU=Android, O=Unknown")
  ) {
    throw new BuildError("generated app bundle is signed with the Android debug certificate");
  }
  const signerOwner = certificateOutput
    .split("\n")
    .find((line) => line.startsWith("Owner: "))
    ?.replace("Owner: ", "")
    .trim();
  if (!signerOwner) {
    throw new BuildError("could not find signer owner in generated app bundle");
  }
  const signerSha256 = certificateSHA256FromOutput(certificateOutput, "generated app bundle certificate");
  if (signerSha256 !== expectedSignerSha256) {
    throw new BuildError(`generated app bundle signer SHA-256 ${signerSha256} does not match release identity ${expectedSignerSha256}`);
  }
  run(["bundletool", "validate", `--bundle=${bundlePath}`], { quiet: true });
  return { signerOwner, signerSha256, bundletoolValidated: true };
}

/**
 * @param {string} certificateOutput
 * @param {string} label
 * @returns {string}
 */
function certificateSHA256FromOutput(certificateOutput, label) {
  const match = certificateOutput.match(/SHA256:\s*([0-9A-Fa-f:]+)/);
  if (!match) {
    throw new BuildError(`could not find SHA-256 fingerprint in ${label}`);
  }
  return normalizeSHA256Fingerprint(match[1]);
}

/**
 * @param {string} value
 * @returns {string}
 */
function normalizeSHA256Fingerprint(value) {
  const hex = value.replace(/[^0-9A-Fa-f]/g, "").toUpperCase();
  if (!/^[0-9A-F]{64}$/.test(hex)) {
    throw new BuildError(`invalid SHA-256 fingerprint: ${value}`);
  }
  return hex.match(/../g)?.join(":") || "";
}

/**
 * @param {unknown} value
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
 * @param {string} pathToRead
 * @returns {Record<string, string>}
 */
function readProperties(pathToRead) {
  requireFile(pathToRead, "properties file");
  /** @type {Record<string, string>} */
  const properties = {};
  for (const rawLine of fs.readFileSync(pathToRead, "utf8").split("\n")) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) {
      continue;
    }
    const equalsIndex = line.indexOf("=");
    if (equalsIndex <= 0) {
      throw new BuildError(`invalid properties line in ${pathToRead}: ${rawLine}`);
    }
    properties[line.slice(0, equalsIndex).trim()] = line.slice(equalsIndex + 1).trim();
  }
  return properties;
}

/**
 * @param {string} pathToWrite
 * @param {Record<string, string>} properties
 */
function writeProperties(pathToWrite, properties) {
  fs.writeFileSync(
    pathToWrite,
    Object.entries(properties)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, value]) => `${key}=${value}`)
      .join("\n") + "\n",
    "utf8",
  );
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
 * @param {{ cwd?: string; env?: NodeJS.ProcessEnv; quiet?: boolean; label?: string }} [options]
 */
function run(command, options = {}) {
  process.stdout.write(`+ ${options.label || command.join(" ")}${options.cwd ? ` in ${options.cwd}` : ""}\n`);
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
      if (result.stdout) {
        process.stderr.write(result.stdout);
      }
      if (result.stderr) {
        process.stderr.write(result.stderr);
      }
    }
    throw new BuildError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${command.join(" ")}`);
  }
}

/**
 * @param {string[]} command
 * @param {{ label?: string }} [options]
 * @returns {string}
 */
function runAndRead(command, options = {}) {
  const result = spawnSync(command[0], command.slice(1), { encoding: "utf8", stdio: "pipe" });
  if (result.error) {
    throw new BuildError(`command failed: ${result.error.message}`);
  }
  if (result.status !== 0) {
    if (result.stdout) {
      process.stderr.write(result.stdout);
    }
    if (result.stderr) {
      process.stderr.write(result.stderr);
    }
    throw new BuildError(`command failed with exit ${String(result.status ?? result.signal ?? "unknown")}: ${options.label || command.join(" ")}`);
  }
  return result.stdout;
}

/**
 * @param {string} name
 * @returns {string}
 */
function which(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8", stdio: "pipe" });
  return result.status === 0 ? result.stdout.trim() : "";
}

/**
 * @param {string} pathToRequire
 * @param {string} label
 */
function requireFile(pathToRequire, label) {
  if (!pathToRequire || !fs.existsSync(pathToRequire) || !fs.statSync(pathToRequire).isFile()) {
    throw new BuildError(`missing ${label}: ${pathToRequire}`);
  }
}

/**
 * @param {string} pathToRequire
 * @param {string} label
 */
function requireDirectory(pathToRequire, label) {
  if (!pathToRequire || !fs.existsSync(pathToRequire) || !fs.statSync(pathToRequire).isDirectory()) {
    throw new BuildError(`missing ${label}: ${pathToRequire}`);
  }
}

/**
 * @param {string} pathToRequire
 * @param {string} label
 */
function requireExecutable(pathToRequire, label) {
  if (!pathToRequire || !fs.existsSync(pathToRequire)) {
    throw new BuildError(`missing executable for ${label}: ${pathToRequire}`);
  }
  try {
    fs.accessSync(pathToRequire, fs.constants.X_OK);
  } catch (error) {
    throw new BuildError(`file is not executable for ${label}: ${pathToRequire}`);
  }
}
