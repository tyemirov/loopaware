// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

const labelCases = [
  { selector: '#new-site-button', text: 'New site' },
  { selector: '#widget-test-button', text: 'Test' },
  { selector: '#copy-widget-snippet', text: 'Copy snippet' },
  { selector: '#subscribe-test-button', text: 'Test' },
  { selector: '#copy-subscribe-widget-snippet', text: 'Copy snippet' },
  { selector: '#traffic-test-button', text: 'Test' },
  { selector: '#copy-traffic-widget-snippet', text: 'Copy snippet' },
  { selector: '#session-timeout-confirm-button', text: 'Yes' },
  { selector: '#session-timeout-dismiss-button', text: 'No' },
  { selector: '#session-timeout-message', text: 'Log out due to inactivity?' },
  { selector: 'h5:has-text("Site details")', text: 'Site details' },
  { selector: 'h5:has-text("Feedback widget")', text: 'Feedback widget' },
  { selector: 'h5:has-text("Subscribers widget")', text: 'Subscribers widget' },
  { selector: 'h5:has-text("Traffic widget")', text: 'Traffic widget' },
  { selector: '#dashboard-section-tab-feedback', text: 'Feedback' },
  { selector: '#dashboard-section-tab-subscriptions', text: 'Subscriptions' },
  { selector: '#dashboard-section-tab-traffic', text: 'Traffic' },
  { selector: '.card-header:has-text("Account")', text: 'Account' },
  { selector: '.card-header:has-text("Sites")', text: 'Sites' },
  { selector: '#settings-modal-title', text: 'Account Settings' },
  { selector: 'h2:has-text("Auto logout")', text: 'Auto logout' },
  { selector: 'label[for="edit-site-origin"]', text: 'Allowed origins' },
  { selector: 'label[for="widget-placement-bottom-offset"]', text: 'Bottom offset' },
  { selector: 'label[for="settings-auto-logout-prompt-seconds"]', text: 'Show reminder after' },
  { selector: 'label[for="settings-auto-logout-logout-seconds"]', text: 'Sign out after' }
];

for (const labelCase of labelCases) {
  test(`dashboard label ${labelCase.text} ${labelCase.selector}`, async ({ page }) => {
    await openDashboard(page, config, adminUser);
    await expect(page.locator(labelCase.selector)).toContainText(labelCase.text);
  });
}

test('dashboard new site mode marks button active', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  const newSiteButton = page.locator('#new-site-button');
  await newSiteButton.click();
  await expect(newSiteButton).toHaveClass(/btn-primary/);
});
