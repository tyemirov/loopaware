// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboardShell } from '../helpers/fixtures.js';
import { capturePageScreenshot, ensureScreenshotDirectory } from '../helpers/screenshot.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const MAX_HEADER_CONTENT_GAP_PIXELS = 40;

async function expectSitesPanelToFillViewport(page) {
  await expect.poll(async () => page.evaluate(() => {
    const header = document.querySelector('mpr-header').getBoundingClientRect();
    const footer = document.querySelector('#dashboard-footer footer').getBoundingClientRect();
    const panel = document.querySelector('#site-dashboard-view > .col-lg-4 > .card').getBoundingClientRect();
    return {
      header: { top: header.top, bottom: header.bottom, height: header.height },
      footer: { top: footer.top, height: footer.height },
      panel: { top: panel.top, bottom: panel.bottom, height: panel.height },
      headerHeight: document.body.style.getPropertyValue('--dashboard-header-height'),
      footerHeight: document.body.style.getPropertyValue('--dashboard-footer-height'),
      clearsShell: panel.top > header.bottom && panel.bottom < footer.top,
      matchingMargins: Math.abs((panel.top - header.bottom) - (footer.top - panel.bottom)) <= 1
    };
  })).toMatchObject({ matchingMargins: true, clearsShell: true });
}

test('desktop Sites panel fills the viewport with matching shell margins', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 1000 });
  await openDashboardShell(page, config, adminUser, { localStorage: { loopaware_public_theme: 'dark' } });
  await expectSitesPanelToFillViewport(page);
  await capturePageScreenshot(page, ensureScreenshotDirectory('dashboard-I063'), 'sites-panel');
  await page.locator('#site-search-toggle-button').click();
  await expect(page.locator('#site-search-input')).toBeVisible();
  await expectSitesPanelToFillViewport(page);
  await page.setViewportSize({ width: 1024, height: 650 });
  await expectSitesPanelToFillViewport(page);
  await page.evaluate(() => window.scrollTo(0, 250));
  await expectSitesPanelToFillViewport(page);
  await page.locator('mpr-header').evaluate((header) => { header.style.paddingBottom = '20px'; });
  await page.locator('#dashboard-footer footer').evaluate((footer) => { footer.style.paddingTop = '30px'; });
  await expectSitesPanelToFillViewport(page);
});

test('narrow dashboard keeps Sites above site details', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openDashboardShell(page, config, adminUser);
  const panel = page.locator('#site-dashboard-view > .col-lg-4 > .card');
  const details = page.locator('#site-dashboard-view > .col-lg-8 > .card').first();
  const panelBox = await panel.boundingBox();
  const detailsBox = await details.boundingBox();
  expect(detailsBox.y).toBeGreaterThanOrEqual(panelBox.y + panelBox.height);
  expect(panelBox.x + panelBox.width).toBeLessThanOrEqual(390);
});

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
