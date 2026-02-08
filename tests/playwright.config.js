// @ts-check
import { defineConfig } from '@playwright/test';

const baseURL = process.env.LOOPAWARE_BASE_URL || 'http://localhost:8090';

export default defineConfig({
  testDir: './specs',
  fullyParallel: false,
  workers: 1,
  timeout: 90_000,
  expect: {
    timeout: 15_000
  },
  reporter: [['list']],
  use: {
    baseURL,
    headless: true,
    viewport: { width: 1280, height: 720 },
    ignoreHTTPSErrors: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'retain-on-failure'
  }
});
