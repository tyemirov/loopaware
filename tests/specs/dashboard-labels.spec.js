// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

const labelCases = [
  { selector: '#new-site-button', text: 'New site' },
  { selector: '#widget-test-button', text: 'Test' },
  { selector: '#copy-widget-snippet', text: 'Copy' },
  { selector: '#subscribe-test-button', text: 'Test' },
  { selector: '#copy-subscribe-widget-snippet', text: 'Copy' },
  { selector: '#traffic-test-button', text: 'Test' },
  { selector: '#copy-traffic-widget-snippet', text: 'Copy' },
  { selector: '#session-timeout-confirm-button', text: 'Yes' },
  { selector: '#session-timeout-dismiss-button', text: 'No' },
  { selector: '#session-timeout-message', text: 'Log out due to inactivity?' },
  { selector: 'h5:has-text("Site details")', text: 'Site details' },
  { selector: 'h5:has-text("Feedback widget")', text: 'Feedback widget' },
  { selector: 'h5:has-text("Subscribers widget")', text: 'Subscribers widget' },
  { selector: 'h5:has-text("Traffic widget")', text: 'Traffic widget' },
  { selector: 'label[for="widget-show-message-input"]', text: 'Show message input' },
  { selector: 'label[for="widget-show-sentiment-buttons"]', text: 'Show sentiment buttons' },
  { selector: '#dashboard-section-tab-feedback', text: 'Feedback' },
  { selector: '#dashboard-section-tab-subscriptions', text: 'Subscriptions' },
  { selector: '#dashboard-section-tab-traffic', text: 'Traffic' },
  { selector: '#traffic-pages-heading', text: 'Pages' },
  { selector: '#traffic-devices-heading', text: 'Devices' },
  { selector: '#traffic-locations-heading', text: 'Locations' },
  { selector: 'label[for="traffic-interval-1day"]', text: '1 day' },
  { selector: 'label[for="traffic-interval-30days"]', text: '30 days' },
  { selector: 'label[for="traffic-interval-all"]', text: 'All' },
  { selector: '#traffic-export-button', text: 'Download CSV' },
  { selector: '#dashboard-section-tab-sentry', text: 'LA Sentry' },
  { selector: 'h5:has-text("LA Sentry clients")', text: 'LA Sentry clients' },
  { selector: 'h5:has-text("LA Sentry issues")', text: 'LA Sentry issues' },
  { selector: 'label[for="sentry-ingest-endpoint"]', text: 'Ingest endpoint' },
  { selector: 'label[for="sentry-ingest-token"]', text: 'Ingest token' },
  { selector: 'label[for="sentry-browser-snippet"]', text: 'Browser snippet' },
  { selector: '#settings-modal .card-header:has-text("Account")', text: 'Account' },
  { selector: '#site-dashboard-view .card-header:has(#new-site-button)', text: 'Sites' },
  { selector: '#settings-modal-title', text: 'Account Settings' },
  { selector: '#settings-modal h2:has-text("Reports")', text: 'Reports' },
  { selector: '#settings-open-all-sites-traffic', text: 'All sites traffic' },
  { selector: '#all-sites-traffic-view h1', text: 'Reports' },
  { selector: '#all-sites-traffic-back-button', text: 'Back to site dashboard' },
  { selector: '#global-traffic-report-library h2', text: 'Report library' },
  { selector: '#global-traffic-report-create-button', text: 'New report' },
  { selector: '#global-traffic-report-title', text: 'All sites traffic' },
  { selector: 'label[for="global-traffic-report-name"]', text: 'Report name' },
  { selector: '#global-traffic-report-select-all-sites', text: 'Select all' },
  { selector: '#global-traffic-report-clear-sites', text: 'Clear' },
  { selector: 'label[for="all-sites-traffic-report-enabled"]', text: 'Enable scheduled reports' },
  { selector: 'label[for="all-sites-traffic-report-send-time"]', text: 'Send at' },
  { selector: 'label[for="all-sites-traffic-report-timezone"]', text: 'Timezone' },
  { selector: 'label[for="all-sites-traffic-report-recipient"]', text: 'Recipient email' },
  { selector: 'h2:has-text("Auto logout")', text: 'Auto logout' },
  { selector: 'label[for="edit-site-origin"]', text: 'Allowed origins' },
  { selector: 'label[for="widget-placement-bottom-offset"]', text: 'Bottom offset' },
  { selector: 'label[for="widget-accent-color"]', text: 'Accent color' },
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
