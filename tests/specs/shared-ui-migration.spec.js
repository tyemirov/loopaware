// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { enableAutoGoogleCredentialOnClick, installExternalAssetStubs } from '../helpers/externalAssets.js';
import { installTauthStub } from '../helpers/tauthStub.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, waitForDashboardReady } from '../helpers/fixtures.js';

const config = resolveTestConfig();
test.use({ actionTimeout: 5000 });

test.beforeEach(async ({ page }) => {
  await installExternalAssetStubs(page, config);
});

test('served shared config declares both current provider maps', async ({ page, request }) => {
  await installTauthStub(page, config);
  const response = await request.get('/config-ui.yaml');
  expect(response.status()).toBe(200);
  const source = await response.text();
  await page.goto('/privacy');
  await page.waitForFunction(() => Boolean(/** @type {any} */ (window).jsyaml));
  const parsed = await page.evaluate(text => /** @type {any} */ (window).jsyaml.load(text), source);
  expect(parsed.environments).toHaveLength(2);
  for (const environment of parsed.environments) {
    expect(environment.auth).toEqual({
      tauthUrl: environment.description === 'Production' ? 'https://tauth-api.mprlab.com' : '',
      tenantId: 'loopaware',
      logoutPath: '/auth/logout',
      sessionPath: '/auth/session',
      providers: {
        google: {
          enabled: true,
          clientId: '281540686395-b0ndglao5r6u6qih2etrtqudf6t0qbt0.apps.googleusercontent.com',
          loginPath: '/auth/google',
          noncePath: '/auth/nonce',
        },
        apple: { enabled: false },
        password: { enabled: false },
      },
    });
  }
});

for (const width of [390, 1280]) {
  test(`public footer uses the current menu at ${width}px`, async ({ page }) => {
    await installTauthStub(page, config);
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/privacy');
    const footer = page.locator('mpr-footer');
    await expect(footer).toHaveAttribute('menu', /"placement":"top"/);
    await footer.getByRole('button', { name: 'Built by Marco Polo Research Lab', exact: true }).click();
    await expect(footer.getByRole('link', { name: 'Gravity Notes', exact: true })).toHaveAttribute('href', 'https://gravity.mprlab.com');
    await page.keyboard.press('Escape');
    await expect(footer.getByRole('button', { name: 'Built by Marco Polo Research Lab', exact: true })).toHaveAttribute('aria-expanded', 'false');
  });
}

for (const width of [390, 1280]) {
  for (const authOrigin of ['frontend', 'direct']) {
    test(`shared recovery preserves one site mutation at ${width}px with ${authOrigin} TAuth`, async ({ page }) => {
      test.setTimeout(30000);
      await page.setViewportSize({ width, height: 900 });
      const tauthOrigin = authOrigin === 'direct' ? 'http://localhost:18081' : config.baseOrigin;
      await page.route('**/config-ui.yaml', async route => {
        const response = await route.fetch();
        const body = (await response.text()).replace('tauthUrl: ""', `tauthUrl: "${tauthOrigin}"`);
        await route.fulfill({ response, body });
      });
      const authRequests = [];
      page.on('request', request => {
        const url = new URL(request.url());
        if (url.pathname.startsWith('/auth/')) authRequests.push({ origin: url.origin, path: url.pathname });
      });
      const user = buildAdminUser(config);
      const cookie = buildSessionCookie(config, user);
      await installTauthStub(page, config, { sessionCookieValue: cookie.value });
      await page.goto('/login');
      await enableAutoGoogleCredentialOnClick(page);
      await page.locator('mpr-header').getByRole('button', { name: 'Sign in with Google', exact: true }).click();
      await expect(page).toHaveURL(/\/app\/?$/);
      await waitForDashboardReady(page, { allowEmptySites: true });

      let readAttempts = 0;
      let mutationAttempts = 0;
      let obsoleteRefreshRequests = 0;
      await page.route('**/auth/refresh', route => {
        obsoleteRefreshRequests += 1;
        return route.fulfill({ status: 405, body: '' });
      });
      await page.route('**/api/sites', route => {
        const method = route.request().method();
        if (method === 'GET' && ++readAttempts === 1) return route.fulfill({ status: 401, body: '{"error":"unauthorized"}', contentType: 'application/json' });
        if (method === 'POST' && ++mutationAttempts === 1) return route.fulfill({ status: 401, body: '{"error":"unauthorized"}', contentType: 'application/json' });
        return route.continue();
      });
      await page.reload();
      await waitForDashboardReady(page, { allowEmptySites: true });
      expect(readAttempts).toBeGreaterThanOrEqual(2);
      expect(obsoleteRefreshRequests).toBe(0);

      const name = buildUniqueName('Shared Recovery');
      await page.locator('#new-site-button').click();
      await page.locator('#edit-site-name').fill(name);
      await page.locator('#edit-site-origin').fill(buildUniqueOrigin('shared-recovery'));
      await page.locator('#edit-site-owner').fill(user.email);
      await page.locator('#edit-site-owner').press('Tab');
      await expect(page.locator('#sites-list [data-site-id].active')).toContainText(name);
      expect(mutationAttempts).toBe(2);
      const response = await page.request.get('/api/sites');
      expect(response.status()).toBe(200);
      const body = await response.json();
      expect(body.sites.filter(site => site.name === name)).toHaveLength(1);
      await page.locator('mpr-user [data-mpr-user="trigger"]').click();
      await page.locator('mpr-user [data-mpr-user="logout"]').click();
      await expect(page).toHaveURL(/\/login\/?$/);
      expect((await page.request.get('/api/me')).status()).toBe(401);
      expect(authRequests.length).toBeGreaterThan(0);
      expect(authRequests.every(request => request.origin === tauthOrigin)).toBe(true);
      for (const path of ['/auth/nonce', '/auth/google', '/auth/session', '/auth/logout']) {
        expect(authRequests.some(request => request.path === path)).toBe(true);
      }
    });
  }
}
