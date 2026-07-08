// @ts-check

const loopAwareScheme = "loopaware";
const loopAwarePackageScheme = "com.mprlab.loopaware";
const loopAwareGold = "#D4AF37";
const loopAwareWhite = "#FFFFFF";
const mobileVersion = optionalCalVerVersion(process.env.LOOPAWARE_MOBILE_VERSION || "2026.6.19", "LOOPAWARE_MOBILE_VERSION");
const isProductionNativeBuild = process.env.NODE_ENV === "production";
const iosBundleIdentifier = process.env.LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER || "com.mprlab.loopaware";
const androidPackage = process.env.LOOPAWARE_MOBILE_ANDROID_PACKAGE || "com.mprlab.loopaware";
const iosBuildNumber = optionalPositiveIntegerString(process.env.LOOPAWARE_MOBILE_IOS_BUILD_NUMBER || "", "LOOPAWARE_MOBILE_IOS_BUILD_NUMBER");
const androidVersionCode = optionalPositiveInteger(process.env.LOOPAWARE_MOBILE_ANDROID_VERSION_CODE || "", "LOOPAWARE_MOBILE_ANDROID_VERSION_CODE");
const googleIosRedirectUri =
  process.env.LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI || process.env.TAUTH_TENANT_GOOGLE_IOS_REDIRECT_URI_LOOPAWARE || "";
const googleIosClientId = process.env.LOOPAWARE_MOBILE_GOOGLE_IOS_CLIENT_ID || process.env.TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE || "";
const googleIosRedirectScheme = redirectUriScheme(googleIosRedirectUri) || reverseGoogleClientIdScheme(googleIosClientId);
const iosRedirectSchemes = googleIosRedirectScheme ? [googleIosRedirectScheme] : [];

module.exports = {
  expo: {
    name: "LoopAware",
    slug: "loopaware-mobile",
    version: mobileVersion,
    orientation: "portrait",
    icon: "./assets/icon.png",
    scheme: [loopAwareScheme, loopAwarePackageScheme],
    userInterfaceStyle: "light",
    primaryColor: loopAwareGold,
    ios: {
      bundleIdentifier: iosBundleIdentifier,
      supportsTablet: true,
      ...(iosBuildNumber ? { buildNumber: iosBuildNumber } : {}),
      ...(iosRedirectSchemes.length ? { scheme: iosRedirectSchemes } : {}),
    },
    android: {
      package: androidPackage,
      ...(androidVersionCode ? { versionCode: androidVersionCode } : {}),
      adaptiveIcon: {
        backgroundColor: loopAwareWhite,
        foregroundImage: "./assets/android-icon-foreground.png",
        monochromeImage: "./assets/android-icon-monochrome.png",
      },
      intentFilters: [
        {
          action: "VIEW",
          autoVerify: false,
          data: [{ scheme: loopAwareScheme }, { scheme: loopAwarePackageScheme }],
          category: ["BROWSABLE", "DEFAULT"],
        },
      ],
      predictiveBackGestureEnabled: false,
    },
    web: {
      favicon: "./assets/favicon.png",
    },
    plugins: mobilePlugins(isProductionNativeBuild),
    extra: {
      loopAware: {
        apiBaseUrl: process.env.LOOPAWARE_MOBILE_API_BASE_URL || "https://loopaware-api.mprlab.com",
        tauthBaseUrl: process.env.LOOPAWARE_MOBILE_TAUTH_BASE_URL || "https://tauth-api.mprlab.com",
        tauthTenantId: process.env.LOOPAWARE_MOBILE_TAUTH_TENANT_ID || "loopaware",
      },
    },
  },
};

/**
 * @param {string | undefined | null} clientId
 * @returns {string}
 */
function reverseGoogleClientIdScheme(clientId) {
  const normalizedClientId = String(clientId || "").trim();
  if (!normalizedClientId.endsWith(".apps.googleusercontent.com")) {
    return "";
  }
  return `com.googleusercontent.apps.${normalizedClientId.replace(".apps.googleusercontent.com", "")}`;
}

/**
 * @param {string | undefined | null} redirectUri
 * @returns {string}
 */
function redirectUriScheme(redirectUri) {
  const normalizedRedirectUri = String(redirectUri || "").trim();
  const separatorIndex = normalizedRedirectUri.indexOf(":");
  if (separatorIndex <= 0) {
    return "";
  }
  return normalizedRedirectUri.slice(0, separatorIndex);
}

/**
 * @param {string} value
 * @param {string} label
 * @returns {string}
 */
function optionalPositiveIntegerString(value, label) {
  const normalizedValue = String(value || "").trim();
  if (!normalizedValue) {
    return "";
  }
  if (!/^[1-9][0-9]*$/.test(normalizedValue)) {
    throw new Error(`${label} must be a positive integer`);
  }
  return normalizedValue;
}

/**
 * @param {string} value
 * @param {string} label
 * @returns {number | undefined}
 */
function optionalPositiveInteger(value, label) {
  const normalizedValue = optionalPositiveIntegerString(value, label);
  return normalizedValue ? Number(normalizedValue) : undefined;
}

/**
 * @param {boolean} productionNativeBuild
 * @returns {string[]}
 */
function mobilePlugins(productionNativeBuild) {
  const plugins = ["expo-web-browser", "expo-secure-store", "expo-system-ui"];
  if (!productionNativeBuild) {
    plugins.push("expo-dev-client");
  }
  return plugins;
}

/**
 * @param {string} value
 * @param {string} label
 * @returns {string}
 */
function optionalCalVerVersion(value, label) {
  const normalizedValue = String(value || "").trim();
  if (!/^[1-9][0-9]{3}\.(?:[1-9]|1[0-2])\.(?:[1-9]|[12][0-9]|3[01])$/.test(normalizedValue)) {
    throw new Error(`${label} must use YYYY.M.D CalVer`);
  }
  return normalizedValue;
}
