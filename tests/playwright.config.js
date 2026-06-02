// @ts-check
import { defineConfig } from '@playwright/test';

const baseURL = process.env.LOOPAWARE_BASE_URL || 'http://localhost:8090';
const browserChannel = process.env.LOOPAWARE_PLAYWRIGHT_CHANNEL?.trim();
const videoMode = browserChannel ? 'off' : 'retain-on-failure';
/** @type {NonNullable<import('@playwright/test').PlaywrightTestConfig['use']>} */
const browserOptions = {
  baseURL,
  headless: true,
  viewport: { width: 1280, height: 720 },
  ignoreHTTPSErrors: true,
  screenshot: 'only-on-failure',
  trace: 'retain-on-failure',
  video: videoMode
};

if (browserChannel) {
  browserOptions.channel = browserChannel;
}

export default defineConfig({
  testDir: './specs',
  fullyParallel: false,
  workers: 1,
  timeout: 90_000,
  expect: {
    timeout: 15_000
  },
  reporter: [['list']],
  use: browserOptions
});
