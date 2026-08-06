// @ts-check
/// <reference types="node" />

const mobileReleaseEpoch = Date.UTC(2020, 0, 1, 0, 0, 0);
const androidVersionCodeMaximum = 2100000000;
const millisecondsPerSecond = 1000;

/**
 * @typedef {{
 *   releaseTimestamp: string;
 *   releaseVersion: string;
 *   buildCode: number;
 *   iosBuildNumber: string;
 *   androidVersionCode: number;
 *   buildCodeSource: string;
 * }} MobileCalVerVersion
 */

/**
 * @param {string | undefined | null} rawTimestamp
 * @returns {MobileCalVerVersion}
 */
export function createMobileCalVerVersion(rawTimestamp = "") {
  const releaseDate = parseReleaseTimestamp(rawTimestamp);
  const buildCode = buildCodeFromDate(releaseDate);
  return {
    releaseTimestamp: releaseDate.toISOString(),
    releaseVersion: `${releaseDate.getUTCFullYear()}.${releaseDate.getUTCMonth() + 1}.${releaseDate.getUTCDate()}`,
    buildCode,
    iosBuildNumber: String(buildCode),
    androidVersionCode: buildCode,
    buildCodeSource: "calver_utc_seconds_since_2020_01_01",
  };
}

/**
 * @param {string | undefined | null} rawTimestamp
 * @returns {Date}
 */
function parseReleaseTimestamp(rawTimestamp) {
  const normalizedTimestamp = String(rawTimestamp || "").trim();
  if (!normalizedTimestamp) {
    return new Date();
  }
  if (/^[1-9][0-9]*$/.test(normalizedTimestamp)) {
    return new Date(Number(normalizedTimestamp) * millisecondsPerSecond);
  }
  const parsedMilliseconds = Date.parse(normalizedTimestamp);
  if (!Number.isFinite(parsedMilliseconds)) {
    throw new Error(`invalid_mobile_release_timestamp: ${normalizedTimestamp}`);
  }
  return new Date(parsedMilliseconds);
}

/**
 * @param {Date} releaseDate
 * @returns {number}
 */
function buildCodeFromDate(releaseDate) {
  const releaseMilliseconds = releaseDate.getTime();
  if (!Number.isFinite(releaseMilliseconds)) {
    throw new Error("invalid_mobile_release_timestamp: timestamp did not produce a finite date");
  }
  if (releaseMilliseconds < mobileReleaseEpoch) {
    throw new Error("invalid_mobile_release_timestamp: timestamp must be on or after 2020-01-01T00:00:00Z");
  }
  const buildCode = Math.floor((releaseMilliseconds - mobileReleaseEpoch) / millisecondsPerSecond);
  if (buildCode <= 0 || buildCode > androidVersionCodeMaximum) {
    throw new Error(`invalid_mobile_release_timestamp: generated Android versionCode ${buildCode} is outside Google Play bounds`);
  }
  return buildCode;
}
