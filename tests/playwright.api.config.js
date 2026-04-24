// @ts-check
import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config.js';

const apiSpecsPattern = /api-(admin|public|sentry)\.spec\.js/;

export default defineConfig({
  ...baseConfig,
  testMatch: apiSpecsPattern
});
