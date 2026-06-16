// @ts-check
import { test, expect } from '@playwright/test';

const resourcePages = [
  {
    path: '/resources',
    canonical: 'https://loopaware.mprlab.com/resources',
    title: 'LoopAware Resources',
  },
  {
    path: '/resources/feedback-widget',
    canonical: 'https://loopaware.mprlab.com/resources/feedback-widget',
    title: 'Feedback Widget for Websites',
  },
  {
    path: '/resources/subscriber-capture',
    canonical: 'https://loopaware.mprlab.com/resources/subscriber-capture',
    title: 'Email Subscriber Capture for Product Sites',
  },
  {
    path: '/resources/privacy-first-analytics',
    canonical: 'https://loopaware.mprlab.com/resources/privacy-first-analytics',
    title: 'Privacy-First Website Analytics',
  },
  {
    path: '/resources/la-sentry-monitoring',
    canonical: 'https://loopaware.mprlab.com/resources/la-sentry-monitoring',
    title: 'LA Sentry Error Monitoring',
  },
  {
    path: '/resources/self-hosted-feedback',
    canonical: 'https://loopaware.mprlab.com/resources/self-hosted-feedback',
    title: 'Self-Hosted Feedback and Analytics',
  },
  {
    path: '/resources/saas-feedback',
    canonical: 'https://loopaware.mprlab.com/resources/saas-feedback',
    title: 'SaaS Feedback and Product Signal Dashboard',
  },
  {
    path: '/resources/agency-client-sites',
    canonical: 'https://loopaware.mprlab.com/resources/agency-client-sites',
    title: 'Feedback and Analytics for Agency Client Sites',
  },
  {
    path: '/resources/lightweight-analytics',
    canonical: 'https://loopaware.mprlab.com/resources/lightweight-analytics',
    title: 'Lightweight Analytics Pixel',
  },
];

test('login page exposes canonical SEO metadata', async ({ request }) => {
  const response = await request.get('/login');
  expect(response.status()).toBe(200);

  const html = await response.text();
  expect(html).toContain('Feedback, analytics, and LA Sentry monitoring');
  expect(html).toContain('<meta name="description" content="Collect feedback, grow your email list, understand site traffic, and catch developer errors from one dashboard. LoopAware helps teams turn customer signals into faster product decisions." />');
  expect(html).toContain('<h3 class="h5 fw-bold">LA Sentry</h3>');
  expect(html).toContain('src=".../la-sentry.js?site_id=YOUR_SITE_ID"');
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
  expect(robots).toContain('Allow: /resources');
  expect(robots).toContain('Disallow: /app/');
  expect(robots).toContain('Sitemap: https://loopaware.mprlab.com/sitemap.xml');

  const sitemapResponse = await request.get('/sitemap.xml');
  expect(sitemapResponse.status()).toBe(200);
  const sitemap = await sitemapResponse.text();
  expect(sitemap).toContain('<loc>https://loopaware.mprlab.com/login</loc>');
  expect(sitemap).toContain('<loc>https://loopaware.mprlab.com/pricing</loc>');
  for (const page of resourcePages) {
    expect(sitemap).toContain(`<loc>${page.canonical}</loc>`);
  }
});

test('resource pages are crawlable and internally linked', async ({ request }) => {
  for (const page of resourcePages) {
    const response = await request.get(page.path);
    expect(response.status()).toBe(200);

    const html = await response.text();
    expect(html).toContain(page.title);
    expect(html).toContain('<meta name="robots" content="index,follow,max-image-preview:large" />');
    expect(html).toContain(`<link rel="canonical" href="${page.canonical}" />`);
    expect(html).toContain('<a href="/resources">Resources</a>');
    expect(html).toContain('LoopAware');
    expect(html).not.toContain('<meta name="robots" content="noindex');
  }
});

test('resource index links every focused resource page', async ({ request }) => {
  const response = await request.get('/resources');
  expect(response.status()).toBe(200);

  const html = await response.text();
  for (const page of resourcePages.filter((resourcePage) => resourcePage.path !== '/resources')) {
    expect(html).toContain(`href="${page.path}"`);
  }
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
