// @ts-check
const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');
const { getDefaultConfig: getExpoConfig } = require('expo/metro-config');

// Expo supplies a Metro fork with separate type declarations. Both native bundlers validate this configuration.
const expoConfig = /** @type {Parameters<typeof mergeConfig>[1]} */ (/** @type {unknown} */ (getExpoConfig(__dirname)));
module.exports = mergeConfig(getDefaultConfig(__dirname), expoConfig);
