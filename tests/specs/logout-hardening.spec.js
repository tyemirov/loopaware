// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

test('logout overlay appears and content hides on logout event', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Verify content is visible initially
  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('#logout-overlay')).toBeHidden();

  // Trigger logout event
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });

  // Verify overlay is visible and content is hidden (via our new CSS)
  await expect(page.locator('#logout-overlay')).toBeVisible();
  
  // Content should be hidden via display: none !important on body.logging-out
  const isMainHidden = await page.evaluate(() => {
    const main = document.querySelector('main');
    return main && window.getComputedStyle(main).display === 'none';
  });
  expect(isMainHidden).toBe(true);

  const isHeaderHidden = await page.evaluate(() => {
    const header = document.querySelector('mpr-header');
    return header && window.getComputedStyle(header).display === 'none';
  });
  expect(isHeaderHidden).toBe(true);
});

test('logout overlay appears and content hides on unauthenticated event', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Trigger unauthenticated event
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-ui:auth:unauthenticated'));
  });

  // Verify overlay is visible and content is hidden
  await expect(page.locator('#logout-overlay')).toBeVisible();
  
  const isMainHidden = await page.evaluate(() => {
    const main = document.querySelector('main');
    return main && window.getComputedStyle(main).display === 'none';
  });
  expect(isMainHidden).toBe(true);
});

test('logout overlay appears and content hides on session timeout confirm', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Force the session timeout prompt
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (win.__loopawareDashboardIdleTestHooks) {
      win.__loopawareDashboardIdleTestHooks.forcePrompt();
    }
  });
  
  await expect(page.locator('#session-timeout-notification')).toBeVisible();
  
  // Click confirm (Yes)
  await page.locator('#session-timeout-confirm-button').click();

  // Verify overlay is visible and content is hidden
  await expect(page.locator('#logout-overlay')).toBeVisible();
  
  const isMainHidden = await page.evaluate(() => {
    const main = document.querySelector('main');
    return main && window.getComputedStyle(main).display === 'none';
  });
  expect(isMainHidden).toBe(true);
});

test('manual window.logout() call triggers overlay', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Intercept the fetch call to /auth/logout so it doesn't actually redirect yet
  await page.route('**/auth/logout', async route => {
    // Just hang or delay
    await new Promise(resolve => setTimeout(resolve, 5000));
  });

  // Call window.logout()
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (typeof win.logout === 'function') {
      win.logout();
    }
  });

  // Verify overlay is visible immediately even while fetch is "pending"
  await expect(page.locator('#logout-overlay')).toBeVisible();
  
  const isMainHidden = await page.evaluate(() => {
    const main = document.querySelector('main');
    return main && window.getComputedStyle(main).display === 'none';
  });
  expect(isMainHidden).toBe(true);
});
