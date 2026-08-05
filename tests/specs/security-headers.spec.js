// @ts-check
import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

const staticContentSecurityPolicy = readFileSync(
  new URL('../../configs/content-security-policy.txt', import.meta.url),
  'utf8'
).trim();

const expectedEdgeHeaders = Object.freeze({
  'content-security-policy': `${staticContentSecurityPolicy}; frame-ancestors 'none'`,
  'cross-origin-opener-policy': 'same-origin-allow-popups',
  'permissions-policy': 'accelerometer=(), autoplay=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()',
  'referrer-policy': 'strict-origin-when-cross-origin',
  'x-content-type-options': 'nosniff',
  'x-frame-options': 'DENY',
  'x-permitted-cross-domain-policies': 'none'
});

/**
 * @param {Record<string, string>} headers
 * @returns {void}
 */
function expectHardeningHeaders(headers) {
  for (const [headerName, headerValue] of Object.entries(expectedEdgeHeaders)) {
    expect(headers[headerName]).toBe(headerValue);
  }
  expect(headers['strict-transport-security']).toBeUndefined();
}

test('proxy serves hardening headers on static frontend documents', async ({ request }) => {
  const response = await request.get('/login');
  expect(response.status()).toBe(200);

  expectHardeningHeaders(response.headers());
});

test('proxy serves hardening headers on proxied api responses', async ({ request }) => {
  const response = await request.get('/api/me');
  expect(response.status()).toBe(401);

  expectHardeningHeaders(response.headers());
});

test('proxy permits browser connections to the trusted jsDelivr asset origin', async ({ request }) => {
  const response = await request.get('/');
  expect(response.status()).toBe(200);

  const connectSourceDirective = response
    .headers()['content-security-policy']
    .split('; ')
    .find((directive) => directive.startsWith('connect-src '));
  expect(connectSourceDirective?.split(' ')).toContain('https://cdn.jsdelivr.net');
});
