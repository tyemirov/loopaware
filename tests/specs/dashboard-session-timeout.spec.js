// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboardShell, waitForLogoutOverlayOrRedirect } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

async function openDashboardWithHooks(page) {
  await openDashboardShell(page, config, adminUser);
  await page.waitForFunction(() => {
    const hooks = window.__loopawareDashboardIdleTestHooks;
    const settings = window.__loopawareDashboardSettingsTestHooks;
    if (!hooks || !settings || typeof settings.readSessionTimeoutStartRequested !== 'function') {
      return false;
    }
    return settings.readSessionTimeoutStartRequested();
  });
}

async function forcePrompt(page) {
  await page.evaluate(() => {
    if (window.__loopawareDashboardIdleTestHooks) {
      window.__loopawareDashboardIdleTestHooks.forcePrompt();
    }
  });
}

async function forceLogout(page) {
  await page.evaluate(() => {
    if (window.__loopawareDashboardIdleTestHooks) {
      window.__loopawareDashboardIdleTestHooks.forceLogout();
    }
  });
}

async function dispatchTheme(page, theme) {
  await page.evaluate((mode) => {
    const footer = document.getElementById('dashboard-footer');
    if (footer) {
      footer.dispatchEvent(new CustomEvent('mpr-footer:theme-change', { detail: { theme: mode } }));
    }
  }, theme);
}

test('force prompt shows timeout banner', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forcePrompt(page);
  await expect(page.locator('#session-timeout-notification')).toBeVisible();
  await expect(page.locator('#session-timeout-message')).toContainText('Log out due to inactivity?');
});

test('dismiss hides timeout banner', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forcePrompt(page);
  await page.locator('#session-timeout-dismiss-button').click();
  await expect(page.locator('#session-timeout-notification')).toHaveAttribute('aria-hidden', 'true');
});

test('confirm logs out to login', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forcePrompt(page);
  await page.locator('#session-timeout-confirm-button').click();
  const redirectedToLogin = await waitForLogoutOverlayOrRedirect(page, { redirectPattern: /\/login/ });
  if (redirectedToLogin) {
    return;
  }
  await expect(page.locator('main')).toBeHidden();
});

test('force logout redirects automatically', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forceLogout(page);
  const redirectedToLogin = await waitForLogoutOverlayOrRedirect(page, { redirectPattern: /\/login/ });
  expect(redirectedToLogin).toBe(true);
});

test('timeout banner stays anchored to the viewport bottom', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forcePrompt(page);
  const banner = page.locator('#session-timeout-notification');
  await expect(banner).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(() => {
        const bannerElement = document.getElementById('session-timeout-notification');
        const footerRoot = document.querySelector('#dashboard-footer [data-mpr-footer="root"]');
        if (!bannerElement || !footerRoot) {
          return null;
        }
        const bannerRect = bannerElement.getBoundingClientRect();
        const footerRect = footerRoot.getBoundingClientRect();
        return Math.abs(bannerRect.bottom - footerRect.top);
      })
    )
    .toBeLessThanOrEqual(1);
});

test('timeout banner uses dark theme classes', async ({ page }) => {
  await openDashboardWithHooks(page);
  await dispatchTheme(page, 'dark');
  await forcePrompt(page);
  await expect(page.locator('#session-timeout-notification')).toHaveClass(/bg-dark-subtle/);
});

test('timeout banner uses light theme classes', async ({ page }) => {
  await openDashboardWithHooks(page);
  await dispatchTheme(page, 'light');
  await forcePrompt(page);
  await expect(page.locator('#session-timeout-notification')).toHaveClass(/bg-body-secondary/);
});

test('timeout banner is polite aria live region', async ({ page }) => {
  await openDashboardWithHooks(page);
  await expect(page.locator('#session-timeout-notification')).toHaveAttribute('aria-live', 'polite');
});

test('timeout banner action buttons are visible', async ({ page }) => {
  await openDashboardWithHooks(page);
  await forcePrompt(page);
  await expect(page.locator('#session-timeout-confirm-button')).toBeVisible();
  await expect(page.locator('#session-timeout-dismiss-button')).toBeVisible();
});
