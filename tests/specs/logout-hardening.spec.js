// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

test('logout overlay appears and content hides on logout event', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

  // Verify content is visible initially
  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('#logout-overlay')).toBeHidden();

  // Trigger logout event
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });

  // Verify overlay is visible OR we already redirected
  const overlay = page.locator('#logout-overlay');
  await expect(async () => {
    if (page.url().includes('/login')) return;
    await expect(overlay).toBeVisible();
  }).toPass({ timeout: 15000 });
  
  if (page.url().includes('/login')) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('logout overlay appears and content hides on unauthenticated event', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

  // Trigger unauthenticated event
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-ui:auth:unauthenticated'));
  });

  // Verify overlay is visible OR we already redirected
  const overlay = page.locator('#logout-overlay');
  await expect(async () => {
    if (page.url().includes('/login')) return;
    await expect(overlay).toBeVisible();
  }).toPass();
  
  if (page.url().includes('/login')) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('logout overlay appears and content hides on session timeout confirm', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Wait for session timeout manager to be started
  await expect(async () => {
    const started = await page.evaluate(() => {
      const win = /** @type {any} */ (window);
      return !!(win.__loopawareDashboardIdleTestHooks && win.__loopawareDashboardIdleTestHooks.started());
    });
    expect(started).toBe(true);
  }).toPass();

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

  // Verify overlay is visible OR we already redirected
  const overlay = page.locator('#logout-overlay');
  await expect(async () => {
    // If we already navigated, it's a pass
    if (page.url().includes('/login')) return;
    
    await expect(overlay).toBeVisible();
  }).toPass();
  
  if (page.url().includes('/login')) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    // If navigation happened during evaluate, ignore it
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('manual window.logout() call triggers overlay', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

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
