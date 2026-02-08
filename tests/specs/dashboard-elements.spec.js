// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

const elementIds = [
  'user-avatar',
  'user-name',
  'user-email',
  'user-role',
  'new-site-button',
  'delete-site-button',
  'site-created-at',
  'site-status',
  'site-form',
  'edit-site-name',
  'edit-site-origin',
  'edit-site-owner',
  'sites-list',
  'empty-sites-message',
  'site-search-toggle-button',
  'site-search-input',
  'dashboard-section-tabs',
  'dashboard-section-tab-feedback',
  'dashboard-section-tab-subscriptions',
  'dashboard-section-tab-traffic',
  'widget-test-button',
  'copy-widget-snippet',
  'widget-snippet',
  'subscribe-test-button',
  'copy-subscribe-widget-snippet',
  'subscribe-widget-snippet',
  'traffic-test-button',
  'copy-traffic-widget-snippet',
  'traffic-widget-snippet',
  'feedback-table-body',
  'subscribers-table-body',
  'top-pages-table-body',
  'visit-count',
  'unique-visitor-count',
  'feedback-count',
  'subscriber-count',
  'traffic-status',
  'messages-search-toggle-button',
  'messages-search-input',
  'session-timeout-notification',
  'session-timeout-confirm-button',
  'session-timeout-dismiss-button',
  'settings-modal',
  'settings-auto-logout-enabled',
  'settings-auto-logout-prompt-seconds',
  'settings-auto-logout-logout-seconds'
];

for (const elementId of elementIds) {
  test(`dashboard renders #${elementId}`, async ({ page }) => {
    await openDashboard(page, config, adminUser);
    await expect(page.locator(`#${elementId}`)).toHaveCount(1);
  });
}
