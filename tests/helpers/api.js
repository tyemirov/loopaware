// @ts-check
import { buildCookieHeader } from './auth.js';

function normalizeBaseURL(baseURL) {
  return String(baseURL || '').replace(/\/+$/, '');
}

let clientIPCounter = 1;
function nextClientIP() {
  const suffix = clientIPCounter % 250;
  clientIPCounter += 1;
  return `10.0.0.${suffix || 1}`;
}

function resolveSiteOrigin(site) {
  if (!site || typeof site !== 'object') {
    return '';
  }
  if (typeof site.allowed_origin === 'string' && site.allowed_origin) {
    return site.allowed_origin;
  }
  if (typeof site.allowedOrigin === 'string' && site.allowedOrigin) {
    return site.allowedOrigin;
  }
  return '';
}

export async function apiRequest(options) {
  const resolvedOptions = options || {};
  const baseURL = normalizeBaseURL(resolvedOptions.baseURL);
  const path = resolvedOptions.path || '';
  const url = path.startsWith('http') ? path : `${baseURL}${path.startsWith('/') ? '' : '/'}${path}`;
  const headers = Object.assign({}, resolvedOptions.headers || {});
  if (resolvedOptions.clientIP) {
    const normalizedClientIP = String(resolvedOptions.clientIP).trim();
    if (normalizedClientIP) {
      headers['X-Forwarded-For'] = normalizedClientIP;
      headers['X-Real-IP'] = normalizedClientIP;
    }
  }
  if (resolvedOptions.cookie) {
    headers.Cookie = buildCookieHeader(resolvedOptions.cookie);
  }
  const rawBody = resolvedOptions.rawBody;
  const hasRawBody = rawBody !== undefined && rawBody !== null;
  const hasJSONBody = resolvedOptions.body !== undefined && resolvedOptions.body !== null;
  if (!headers['Content-Type'] && (hasRawBody || hasJSONBody)) {
    headers['Content-Type'] = resolvedOptions.contentType || 'application/json';
  }
  if (resolvedOptions.origin) {
    headers.Origin = resolvedOptions.origin;
    headers.Referer = resolvedOptions.origin;
  }
  let requestBody;
  if (hasRawBody) {
    requestBody = rawBody;
  } else if (hasJSONBody) {
    requestBody = JSON.stringify(resolvedOptions.body);
  }
  const response = await fetch(url, {
    method: resolvedOptions.method || 'GET',
    headers,
    body: requestBody
  });
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    const payload = await response.json();
    return { response, payload };
  }
  const textPayload = await response.text();
  return { response, payload: textPayload };
}

export async function createSite(config, cookie, site) {
  const origin = resolveSiteOrigin(site);
  const payload = {
    name: site.name,
    allowed_origin: origin || site.allowedOrigin,
    owner_email: site.ownerEmail
  };
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/sites',
    method: 'POST',
    cookie,
    body: payload
  });
  if (!response.ok) {
    throw new Error(`create_site_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function updateSite(config, cookie, siteId, update) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}`,
    method: 'PATCH',
    cookie,
    body: update
  });
  if (!response.ok) {
    throw new Error(`update_site_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function listSites(config, cookie) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/sites',
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`list_sites_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function listMessages(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/messages`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`list_messages_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function listSubscribers(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/subscribers`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`list_subscribers_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function createFeedback(config, site, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/feedback',
    method: 'POST',
    origin: resolveSiteOrigin(site),
    clientIP,
    body: {
      site_id: site.id,
      contact: payload.contact,
      message: payload.message
    }
  });
  if (!response.ok) {
    throw new Error(`create_feedback_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function createSubscription(config, site, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/subscriptions',
    method: 'POST',
    origin: resolveSiteOrigin(site),
    clientIP,
    body: {
      site_id: site.id,
      email: payload.email,
      name: payload.name || '',
      source_url: payload.sourceUrl || ''
    }
  });
  if (!response.ok) {
    throw new Error(`create_subscription_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function confirmSubscription(config, site, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/subscriptions/confirm',
    method: 'POST',
    origin: resolveSiteOrigin(site),
    clientIP,
    body: {
      site_id: site.id,
      email: payload.email
    }
  });
  if (!response.ok) {
    throw new Error(`confirm_subscription_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function confirmSubscriptionLink(config, token) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/subscriptions/confirm-link?token=${encodeURIComponent(token)}`,
    method: 'GET'
  });
  if (!response.ok) {
    throw new Error(`confirm_link_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function collectVisit(config, site, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const urlParams = new URLSearchParams();
  urlParams.set('site_id', site.id);
  if (payload.url) {
    urlParams.set('url', payload.url);
  }
  if (payload.visitorId) {
    urlParams.set('visitor_id', payload.visitorId);
  }
  if (payload.referrer) {
    urlParams.set('referrer', payload.referrer);
  }
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/visits?${urlParams.toString()}`,
    method: 'GET',
    origin: resolveSiteOrigin(site),
    clientIP,
    headers: payload.userAgent ? { 'User-Agent': payload.userAgent } : {}
  });
  if (!response.ok) {
    throw new Error(`collect_visit_failed:${response.status}:${String(body)}`);
  }
  return body;
}

export async function fetchVisitStats(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/visits/stats`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`visit_stats_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}
