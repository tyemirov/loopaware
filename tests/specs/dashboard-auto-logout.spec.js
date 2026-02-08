// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const secondaryUser = buildAdminUser(config, {
  email: `secondary-${Date.now()}@example.com`,
  displayName: 'Secondary User',
  userId: `secondary-user-${Date.now()}`
});

async function openSettingsModal(page, user) {
  await openDashboard(page, config, user);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await expect(page.locator('#settings-modal')).toBeVisible();
}

async function readAutoLogoutHooks(page) {
  await page.waitForFunction(() => window.__loopawareDashboardSettingsTestHooks);
  return page.evaluate(() => {
    const hooks = window.__loopawareDashboardSettingsTestHooks;
    return {
      settings: hooks.readAutoLogoutSettings(),
      minPrompt: hooks.minPromptSeconds,
      maxPrompt: hooks.maxPromptSeconds,
      minLogout: hooks.minLogoutSeconds,
      maxLogout: hooks.maxLogoutSeconds,
      minGap: hooks.minimumGapSeconds
    };
  });
}

test('auto logout defaults within bounds', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  const data = await readAutoLogoutHooks(page);
  expect(data.settings.enabled).toBe(true);
  expect(data.settings.promptSeconds).toBeGreaterThanOrEqual(data.minPrompt);
  expect(data.settings.promptSeconds).toBeLessThanOrEqual(data.maxPrompt);
  expect(data.settings.logoutSeconds).toBeGreaterThanOrEqual(data.minLogout);
  expect(data.settings.logoutSeconds).toBeLessThanOrEqual(data.maxLogout);
  expect(data.settings.logoutSeconds).toBeGreaterThanOrEqual(data.settings.promptSeconds + data.minGap);
});

test('auto logout toggle disables inputs', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const toggle = page.locator('#settings-auto-logout-enabled');
  await toggle.setChecked(false);
  await expect(page.locator('#settings-auto-logout-prompt-seconds')).toBeDisabled();
  await expect(page.locator('#settings-auto-logout-logout-seconds')).toBeDisabled();
  await expect(page.locator('#settings-auto-logout-fields')).toHaveAttribute('aria-disabled', 'true');
});

test('auto logout toggle re-enables inputs', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const toggle = page.locator('#settings-auto-logout-enabled');
  await toggle.setChecked(false);
  await toggle.setChecked(true);
  await expect(page.locator('#settings-auto-logout-prompt-seconds')).toBeEnabled();
  await expect(page.locator('#settings-auto-logout-logout-seconds')).toBeEnabled();
  await expect(page.locator('#settings-auto-logout-fields')).toHaveAttribute('aria-disabled', 'false');
});

test('auto logout prompt validation', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const data = await readAutoLogoutHooks(page);
  const promptInput = page.locator('#settings-auto-logout-prompt-seconds');
  await promptInput.fill(String(data.minPrompt - 1));
  await promptInput.blur();
  await expect(promptInput).toHaveClass(/is-invalid/);
  await expect(page.locator('#settings-auto-logout-prompt-error')).toContainText('Enter a whole number');
});

test('auto logout logout validation', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const data = await readAutoLogoutHooks(page);
  const logoutInput = page.locator('#settings-auto-logout-logout-seconds');
  await logoutInput.fill(String(data.minLogout - 1));
  await logoutInput.blur();
  await expect(logoutInput).toHaveClass(/is-invalid/);
  await expect(page.locator('#settings-auto-logout-logout-error')).toContainText('Enter a whole number');
});

test('auto logout gap validation', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const data = await readAutoLogoutHooks(page);
  const promptInput = page.locator('#settings-auto-logout-prompt-seconds');
  const logoutInput = page.locator('#settings-auto-logout-logout-seconds');
  const promptValue = data.minPrompt + 10;
  const logoutValue = promptValue + data.minGap - 1;
  await promptInput.fill(String(promptValue));
  await logoutInput.fill(String(logoutValue));
  await logoutInput.blur();
  const gapMessage = await page.locator('#settings-auto-logout-fields').getAttribute('data-gap-message');
  await expect(page.locator('#settings-auto-logout-logout-error')).toContainText(gapMessage || 'Choose a sign-out time');
});

test('auto logout persists settings in local storage', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  const promptInput = page.locator('#settings-auto-logout-prompt-seconds');
  const logoutInput = page.locator('#settings-auto-logout-logout-seconds');
  await promptInput.fill('120');
  await logoutInput.fill('240');
  await logoutInput.blur();
  const storageKey = `loopaware_dashboard_auto_logout:${encodeURIComponent(adminUser.email.toLowerCase())}`;
  const stored = await page.evaluate((key) => localStorage.getItem(key), storageKey);
  const parsed = stored ? JSON.parse(stored) : null;
  expect(parsed).not.toBeNull();
  expect(parsed.enabled).toBe(true);
  expect(parsed.prompt_seconds).toBe(120);
  expect(parsed.logout_seconds).toBe(240);
});

test('auto logout restores values on reload', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  await page.locator('#settings-auto-logout-prompt-seconds').fill('180');
  await page.locator('#settings-auto-logout-logout-seconds').fill('360');
  await page.locator('#settings-auto-logout-logout-seconds').blur();
  await page.reload({ waitUntil: 'domcontentloaded' });
  await openSettingsModal(page, adminUser);
  await expect(page.locator('#settings-auto-logout-prompt-seconds')).toHaveValue('180');
  await expect(page.locator('#settings-auto-logout-logout-seconds')).toHaveValue('360');
});

test('auto logout settings are scoped per user', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  await page.locator('#settings-auto-logout-prompt-seconds').fill('300');
  await page.locator('#settings-auto-logout-logout-seconds').fill('600');
  await page.locator('#settings-auto-logout-logout-seconds').blur();
  const secondaryPage = await page.context().newPage();
  await openDashboard(secondaryPage, config, secondaryUser, { allowEmptySites: true });
  await secondaryPage.waitForFunction(() => window.__loopawareDashboardSettingsTestHooks);
  const secondarySettings = await secondaryPage.evaluate(() => window.__loopawareDashboardSettingsTestHooks.readAutoLogoutSettings());
  expect(secondarySettings.promptSeconds).not.toBe(300);
  expect(secondarySettings.logoutSeconds).not.toBe(600);
  await secondaryPage.close();
});

test('disabling auto logout hides session timeout prompt', async ({ page }) => {
  await openSettingsModal(page, adminUser);
  await page.locator('#settings-auto-logout-enabled').setChecked(false);
  await page.evaluate(() => {
    if (window.__loopawareDashboardIdleTestHooks) {
      window.__loopawareDashboardIdleTestHooks.forcePrompt();
    }
  });
  await expect(page.locator('#session-timeout-notification')).toHaveAttribute('aria-hidden', 'true');
});
