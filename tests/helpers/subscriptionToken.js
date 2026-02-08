// @ts-check
import * as crypto from 'node:crypto';

function base64UrlEncode(buffer) {
  return buffer.toString('base64').replace(/=+$/, '').replace(/\+/g, '-').replace(/\//g, '_');
}

export function buildSubscriptionConfirmationToken(secret, subscriberId, siteId, email, ttlSeconds) {
  const trimmedSecret = String(secret || '').trim();
  if (!trimmedSecret) {
    throw new Error('missing_secret');
  }
  const normalizedSubscriberId = String(subscriberId || '').trim();
  const normalizedSiteId = String(siteId || '').trim();
  const normalizedEmail = String(email || '').trim().toLowerCase();
  if (!normalizedSubscriberId || !normalizedSiteId || !normalizedEmail) {
    throw new Error('missing_fields');
  }
  const ttl = Number(ttlSeconds || 0);
  if (!Number.isFinite(ttl) || ttl <= 0) {
    throw new Error('invalid_ttl');
  }
  const payload = {
    subscriber_id: normalizedSubscriberId,
    site_id: normalizedSiteId,
    email: normalizedEmail,
    exp: Math.floor(Date.now() / 1000) + ttl
  };
  const payloadBuffer = Buffer.from(JSON.stringify(payload));
  const encodedPayload = base64UrlEncode(payloadBuffer);
  const hmac = crypto.createHmac('sha256', trimmedSecret);
  hmac.update(encodedPayload);
  const signature = base64UrlEncode(hmac.digest());
  return `${encodedPayload}.${signature}`;
}
