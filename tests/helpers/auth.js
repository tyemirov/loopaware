// @ts-check
import jwt from 'jsonwebtoken';

const defaultIssuer = 'tauth';

export function buildSessionToken(options) {
  const resolvedOptions = options || {};
  const signingKey = resolvedOptions.signingKey || '';
  if (!signingKey) {
    throw new Error('missing_signing_key');
  }
  const tenantId = resolvedOptions.tenantId || 'loopaware';
  const email = resolvedOptions.email || 'admin@example.com';
  const displayName = resolvedOptions.displayName || email;
  const avatarUrl = resolvedOptions.avatarUrl || '';
  const userId = resolvedOptions.userId || 'test-user';
  const nowSeconds = Math.floor(Date.now() / 1000);
  const payload = {
    tenant_id: tenantId,
    user_id: userId,
    user_email: email,
    user_display_name: displayName,
    user_avatar_url: avatarUrl,
    user_roles: [],
    iat: nowSeconds - 60
  };
  return jwt.sign(payload, signingKey, {
    issuer: resolvedOptions.issuer || defaultIssuer,
    subject: userId,
    expiresIn: resolvedOptions.expiresInSeconds || 3600
  });
}

export function buildSessionCookie(config, options) {
  const sessionToken = buildSessionToken({
    signingKey: config.signingKey,
    tenantId: config.tenantId,
    email: options.email,
    displayName: options.displayName,
    avatarUrl: options.avatarUrl,
    issuer: options.issuer
  });
  const baseURL = new URL(config.baseURL);
  return {
    name: config.sessionCookieName,
    value: sessionToken,
    domain: baseURL.hostname,
    path: '/',
    httpOnly: false,
    secure: baseURL.protocol === 'https:'
  };
}

export function buildCookieHeader(cookie) {
  return `${cookie.name}=${cookie.value}`;
}
