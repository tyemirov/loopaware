// @ts-check
import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config.js';

const apiSpecsPattern = /api-(admin|public|la-sentry)\.spec\.js/;

export default defineConfig({
  ...baseConfig,
  testIgnore: apiSpecsPattern
});
