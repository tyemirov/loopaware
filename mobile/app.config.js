const loopAwareScheme = "loopaware";
const loopAwarePackageScheme = "com.mprlab.loopaware";
const iosBundleIdentifier = process.env.LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER || "com.mprlab.loopaware";
const androidPackage = process.env.LOOPAWARE_MOBILE_ANDROID_PACKAGE || "com.mprlab.loopaware";
const googleIosRedirectUri =
  process.env.LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI || process.env.TAUTH_TENANT_GOOGLE_IOS_REDIRECT_URI_LOOPAWARE || "";
const googleIosClientId = process.env.LOOPAWARE_MOBILE_GOOGLE_IOS_CLIENT_ID || process.env.TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE || "";
const googleIosRedirectScheme = redirectUriScheme(googleIosRedirectUri) || reverseGoogleClientIdScheme(googleIosClientId);
const iosRedirectSchemes = googleIosRedirectScheme ? [googleIosRedirectScheme] : [];

module.exports = {
  expo: {
    name: "LoopAware",
    slug: "loopaware-mobile",
    version: "1.0.0",
    orientation: "portrait",
    icon: "./assets/icon.png",
    scheme: [loopAwareScheme, loopAwarePackageScheme],
    userInterfaceStyle: "light",
    ios: {
      bundleIdentifier: iosBundleIdentifier,
      supportsTablet: true,
      ...(iosRedirectSchemes.length ? { scheme: iosRedirectSchemes } : {}),
    },
    android: {
      package: androidPackage,
      adaptiveIcon: {
        backgroundColor: "#E6F4FE",
        foregroundImage: "./assets/android-icon-foreground.png",
        backgroundImage: "./assets/android-icon-background.png",
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
      edgeToEdgeEnabled: true,
      predictiveBackGestureEnabled: false,
    },
    web: {
      favicon: "./assets/favicon.png",
    },
    plugins: ["expo-web-browser", "expo-secure-store", "expo-dev-client"],
    extra: {
      loopAware: {
        apiBaseUrl: process.env.LOOPAWARE_MOBILE_API_BASE_URL || "https://loopaware-api.mprlab.com",
        tauthBaseUrl: process.env.LOOPAWARE_MOBILE_TAUTH_BASE_URL || "https://tauth-api.mprlab.com",
        tauthTenantId: process.env.LOOPAWARE_MOBILE_TAUTH_TENANT_ID || "loopaware",
      },
    },
  },
};

function reverseGoogleClientIdScheme(clientId) {
  const normalizedClientId = String(clientId || "").trim();
  if (!normalizedClientId.endsWith(".apps.googleusercontent.com")) {
    return "";
  }
  return `com.googleusercontent.apps.${normalizedClientId.replace(".apps.googleusercontent.com", "")}`;
}

function redirectUriScheme(redirectUri) {
  const normalizedRedirectUri = String(redirectUri || "").trim();
  const separatorIndex = normalizedRedirectUri.indexOf(":");
  if (separatorIndex <= 0) {
    return "";
  }
  return normalizedRedirectUri.slice(0, separatorIndex);
}
