// @ts-check
import { test, expect } from '@playwright/test';

test('login page exposes canonical SEO metadata', async ({ request }) => {
  const response = await request.get('/login');
  expect(response.status()).toBe(200);

  const html = await response.text();
  expect(html).toContain('Privacy-first feedback widget and traffic analytics');
  expect(html).toContain('<meta name="description" content="Collect feedback, grow your email list, and understand site traffic from one dashboard. LoopAware helps teams turn customer signals into faster product decisions." />');
  expect(html).toContain('<link rel="canonical" href="https://loopaware.mprlab.com/login" />');
  expect(html).toContain('<meta property="og:url" content="https://loopaware.mprlab.com/login" />');
  expect(html).toContain('<meta name="twitter:card" content="summary_large_image" />');
  expect(html).toContain('"@type": "SoftwareApplication"');
});

test('pricing page exposes pricing metadata and faq schema', async ({ request }) => {
  const response = await request.get('/pricing');
  expect(response.status()).toBe(200);

  const html = await response.text();
  expect(html).toContain('<link rel="canonical" href="https://loopaware.mprlab.com/pricing" />');
  expect(html).toContain('LoopAware Pricing | Free, Pro, and Business plans');
  expect(html).toContain('"@type": "FAQPage"');
  expect(html).toContain('"name": "Pro"');
  expect(html).toContain('"price": "39"');
});

test('robots and sitemap publish the public crawl surface', async ({ request }) => {
  const robotsResponse = await request.get('/robots.txt');
  expect(robotsResponse.status()).toBe(200);
  const robots = await robotsResponse.text();
  expect(robots).toContain('Allow: /login');
  expect(robots).toContain('Allow: /pricing');
  expect(robots).toContain('Disallow: /app/');
  expect(robots).toContain('Sitemap: https://loopaware.mprlab.com/sitemap.xml');

  const sitemapResponse = await request.get('/sitemap.xml');
  expect(sitemapResponse.status()).toBe(200);
  const sitemap = await sitemapResponse.text();
  expect(sitemap).toContain('<loc>https://loopaware.mprlab.com/login</loc>');
  expect(sitemap).toContain('<loc>https://loopaware.mprlab.com/pricing</loc>');
});

test('private and token pages opt out of indexing', async ({ request }) => {
  const dashboardHtml = await (await request.get('/app')).text();
  const confirmHtml = await (await request.get('/subscriptions/confirm')).text();
  const unsubscribeHtml = await (await request.get('/subscriptions/unsubscribe')).text();
  const demoHtml = await (await request.get('/subscribe-demo')).text();

  expect(dashboardHtml).toContain('<meta name="robots" content="noindex,nofollow,noarchive" />');
  expect(confirmHtml).toContain('<meta name="robots" content="noindex,nofollow,noarchive" />');
  expect(unsubscribeHtml).toContain('<meta name="robots" content="noindex,nofollow,noarchive" />');
  expect(demoHtml).toContain('<meta name="robots" content="noindex,nofollow,noarchive" />');
});

test('web manifest and icons are available for crawlers and devices', async ({ request }) => {
  const manifestResponse = await request.get('/site.webmanifest');
  expect(manifestResponse.status()).toBe(200);
  const manifest = await manifestResponse.text();
  expect(manifest).toContain('"name": "LoopAware"');
  expect(manifest).toContain('"/loopaware-logo-512x512.png"');

  const iconResponse = await request.get('/loopaware-logo-512x512.png');
  expect(iconResponse.status()).toBe(200);
  expect(iconResponse.headers()['content-type']).toContain('image/png');
});
