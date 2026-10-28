// @ts-check
import assert from 'node:assert/strict';
import { test } from '../scripts/native-test.mjs';
import { createServer } from 'node:http';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { extname } from 'node:path';
import { chromium } from '@playwright/test';
import { installExternalAssetStubs } from './helpers/externalAssets.js';

const WEB_ROOT = new URL('../web/', import.meta.url);
const CANONICAL_KEY = 'loopaware_public_theme';
const YAML_DEPENDENCY_URL = 'https://cdn.jsdelivr.net/npm/js-yaml@4.3.0/dist/js-yaml.min.js';
const RETIRED_KEYS = ['loopaware_landing_theme', 'landing_theme', 'loopaware_theme', 'loopaware_dashboard_theme'];
const TYPES = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.yaml': 'text/yaml' };

async function yamlDependency() {
  const directory = new URL('../.cache/resource-theme/', import.meta.url);
  const file = new URL('js-yaml.min.js', directory);
  let bytes;
  try { bytes = await readFile(file); }
  catch (error) {
    if (error.code !== 'ENOENT') throw error;
    const response = await fetch(YAML_DEPENDENCY_URL, { signal: AbortSignal.timeout(15_000) });
    if (!response.ok) throw new Error(`resource_theme_yaml_download:${response.status}`);
    bytes = Buffer.from(await response.arrayBuffer());
  }
  assert.equal(createHash('sha384').update(bytes).digest('base64'), '0zxS50HhMqXyT0WdkhYMK1yK+EpwgVEIHYc1RW1+JgesjsL7Rwqh0WfQSwEDyDH9');
  await mkdir(directory, { recursive: true });
  await writeFile(file, bytes);
  return bytes;
}

test('real resource page uses one theme writer and restores pointer and keyboard choices', async () => {
  const yamlBytes = await yamlDependency();
  let origin;
  const server = createServer(async (request, response) => {
    const pathname = new URL(request.url, 'http://localhost').pathname;
    const file = pathname.endsWith('/') ? `${pathname}index.html` : pathname;
    try {
      let bytes = await readFile(new URL(`.${file}`, WEB_ROOT));
      if (pathname === '/config-ui.yaml') bytes = Buffer.from(bytes.toString().replace('http://localhost:8090', origin));
      response.writeHead(200, { 'Content-Type': TYPES[extname(file)] || 'application/octet-stream' });
      response.end(bytes);
    } catch {
      response.writeHead(404); response.end();
    }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', () => resolve(undefined)));
  origin = `http://127.0.0.1:${/** @type {import('node:net').AddressInfo} */ (server.address()).port}`;
  let browser;
  try {
    browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();
    await page.addInitScript(() => {
      const originalGet = Storage.prototype.getItem;
      const originalSet = Storage.prototype.setItem;
      originalSet.call(localStorage, 'loopaware_landing_theme', 'light');
      const state = /** @type {any} */ (window);
      state.themeStorageOperations = [];
      Storage.prototype.getItem = function (key) {
        state.themeStorageOperations.push({ operation: 'read', key });
        return originalGet.call(this, key);
      };
      Storage.prototype.setItem = function (key, value) {
        state.themeStorageOperations.push({ operation: 'write', key, value });
        return originalSet.call(this, key, value);
      };
    });
    await installExternalAssetStubs(page, { baseOrigin: origin });
    await page.route(YAML_DEPENDENCY_URL, async route => route.fulfill({
      body: yamlBytes, contentType: 'application/javascript',
    }));
    await page.route('https://cdn.jsdelivr.net/npm/bootstrap@**', route => route.fulfill({ body: '', contentType: 'text/css' }));
    await page.route('https://fonts.googleapis.com/**', route => route.fulfill({ body: '', contentType: 'text/css' }));
    await page.route('**/auth/session', route => route.fulfill({ status: 204, body: '' }));
    await page.route('**/auth/nonce', route => route.fulfill({ json: { nonce: 'resource-theme-fixture' } }));
    await page.goto(`${origin}/resources/feedback-widget/`);
    const control = page.locator('mpr-footer [data-mpr-theme-toggle="control"]');
    await control.waitFor({ state: 'visible' });
    const storageOperations = () => page.evaluate(() => /** @type {any} */ (window).themeStorageOperations);
    assert.deepEqual((await storageOperations()).filter(entry => RETIRED_KEYS.includes(entry.key)), []);
    assert.equal(await page.locator('html').getAttribute('data-bs-theme'), 'dark');
    /** @type {Array<['light' | 'dark', boolean]>} */
    const choices = [['light', false], ['dark', true]];
    for (const [mode, keyboard] of choices) {
      await page.evaluate(() => { /** @type {any} */ (window).themeStorageOperations = []; });
      if (keyboard) { await control.focus(); await page.keyboard.press('Space'); }
      else await control.click();
      assert.equal(await page.locator('html').getAttribute('data-bs-theme'), mode);
      assert.equal(await page.locator('body').getAttribute('data-bs-theme'), mode);
      assert.deepEqual((await storageOperations()).filter(entry => entry.operation === 'write'), [{ operation: 'write', key: CANONICAL_KEY, value: mode }]);
      assert.deepEqual((await storageOperations()).filter(entry => RETIRED_KEYS.includes(entry.key)), []);
      await page.reload();
      await control.waitFor({ state: 'visible' });
      assert.equal(await page.locator('html').getAttribute('data-bs-theme'), mode);
      assert.equal(await page.evaluate(() => /** @type {any} */ (window).MPRUI.getThemeMode()), mode);
      assert.deepEqual((await storageOperations()).filter(entry => RETIRED_KEYS.includes(entry.key)), []);
    }
  } finally {
    await browser?.close();
    await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve(undefined)));
  }
});
