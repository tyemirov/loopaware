// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboardShell } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const MAX_HEADER_CONTENT_GAP_PIXELS = 40;

test('dashboard content starts close to the sticky header', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  const header = page.locator('mpr-header > header.mpr-header');
  const firstDashboardCard = page.locator('main .card').first();
  await expect(header).toBeVisible();
  await expect(firstDashboardCard).toBeVisible();

  const headerBox = await header.boundingBox();
  const firstDashboardCardBox = await firstDashboardCard.boundingBox();
  if (!headerBox || !firstDashboardCardBox) {
    throw new Error('dashboard_layout_boxes_missing');
  }

  const layout = {
    headerBottom: Math.round(headerBox.y + headerBox.height),
    cardTop: Math.round(firstDashboardCardBox.y),
    gap: Math.round(firstDashboardCardBox.y - (headerBox.y + headerBox.height))
  };

  expect(layout.cardTop).toBeGreaterThan(layout.headerBottom);
  expect(layout.gap, `dashboard gap measured ${layout.gap}px`).toBeLessThanOrEqual(MAX_HEADER_CONTENT_GAP_PIXELS);
});

test('dashboard side column contains only the sites card', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  const sideColumnCards = page.locator('#site-dashboard-view > .col-lg-4 > .card');
  await expect(sideColumnCards).toHaveCount(1);
  await expect(sideColumnCards.first().locator('.card-header')).toContainText('Sites');
  await expect(page.locator('#site-dashboard-view > .col-lg-4 .card-header:has-text("Account")')).toHaveCount(0);
});
