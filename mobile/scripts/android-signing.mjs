// @ts-check
import { spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { X509Certificate } from "node:crypto";
import { repositorySigningFile, executeSigningTool } from "./signing-inputs.mjs";

/** Resolve the already captured private inputs through the authoritative runtime.
 * @param {string} gateway
 * @param {string} privateRepositoryRoot
 * @param {NodeJS.ProcessEnv} [environment]
 * @returns {NodeJS.ProcessEnv}
 */
export function capturedAndroidSigningEnvironment(gateway, privateRepositoryRoot, environment = process.env) {
  const keys = ["LOOPAWARE_ANDROID_KEYSTORE", "LOOPAWARE_ANDROID_STORE_PASSWORD", "LOOPAWARE_ANDROID_KEY_ALIAS", "LOOPAWARE_ANDROID_KEY_PASSWORD"];
  const response = spawnSync(gateway, ["private-environment-operation"], {
    input: JSON.stringify({ root: privateRepositoryRoot, keys }),
    env: environment, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"],
  });
  if (response.error || response.status !== 0) throw new Error("Android signing could not resolve the captured private inputs.");
  let captured;
  try { captured = JSON.parse(response.stdout); }
  catch { throw new Error("Android signing received invalid captured private inputs."); }
  if (!captured || typeof captured !== "object" || Array.isArray(captured) ||
      Object.keys(captured).sort().join() !== [...keys].sort().join() ||
      keys.some(key => typeof captured[key] !== "string" || !captured[key])) {
    throw new Error("Android signing received invalid captured private inputs.");
  }
  return { ...environment, ...captured };
}

/** Require the existing upload identity from private repository inputs.
 * @param {string} sourceRepositoryRoot
 * @param {string} privateRepositoryRoot
 * @param {NodeJS.ProcessEnv} environment
 * @param {import('./signing-inputs.mjs').SigningTool} [execute]
 */
export async function androidSigningEnvironment(sourceRepositoryRoot, privateRepositoryRoot, environment, execute = executeSigningTool) {
  const keystore = await repositorySigningFile(privateRepositoryRoot, environment.LOOPAWARE_ANDROID_KEYSTORE, "LOOPAWARE_ANDROID_KEYSTORE");
  for (const name of ["LOOPAWARE_ANDROID_STORE_PASSWORD", "LOOPAWARE_ANDROID_KEY_ALIAS", "LOOPAWARE_ANDROID_KEY_PASSWORD"]) {
    if (!environment[name]) throw new Error(`Signing requires ${name} in configs/.env.loopaware.`);
  }
  const javaHome = environment.ANDROID_STUDIO_JAVA_HOME || environment.JAVA_HOME;
  if (!javaHome) throw new Error("Android release requires the JDK location in ANDROID_STUDIO_JAVA_HOME or JAVA_HOME.");
  const identity = JSON.parse(await readFile(join(sourceRepositoryRoot, "mobile/android-release-identity.json"), "utf8"));
  const certificate = await execute(join(javaHome, "bin/keytool"), ["-exportcert", "-rfc", "-keystore", keystore, "-alias", environment.LOOPAWARE_ANDROID_KEY_ALIAS ?? "", "-storepass:env", "LOOPAWARE_ANDROID_STORE_PASSWORD"], environment);
  if (certificate.status !== 0) throw new Error("Android upload certificate could not be read.");
  if (new X509Certificate(certificate.stdout).fingerprint256 !== identity.uploadKey.sha256) throw new Error("Android signing certificate differs from the registered upload identity.");
  /** @type {NodeJS.ProcessEnv} */
  const result = { ...environment, LOOPAWARE_ANDROID_KEYSTORE: keystore, JAVA_HOME: javaHome, PATH: `${join(javaHome, "bin")}:${environment.PATH || ""}` };
  for (const name of ["GH_TOKEN", "GITHUB_TOKEN", "NPM_API_KEY", "NPM_TOKEN", "APP_STORE_CONNECT_API_KEY_PATH", "APP_STORE_CONNECT_API_KEY_ID", "APP_STORE_CONNECT_API_ISSUER_ID", "GOOGLE_PLAY_SERVICE_ACCOUNT_KEY_PATH"]) delete result[name];
  return result;
}
