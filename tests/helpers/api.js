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

export async function listMobileApps(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/mobile-apps`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`list_mobile_apps_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function createMobileApp(config, cookie, site, payload) {
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${site.id}/mobile-apps`,
    method: 'POST',
    cookie,
    body: {
      client_id: payload.clientId || '',
      platform: payload.platform,
      app_identifier: payload.appIdentifier,
      display_name: payload.displayName || ''
    }
  });
  if (!response.ok) {
    throw new Error(`create_mobile_app_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
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
    path: '/public/feedback',
    method: 'POST',
    origin: resolveSiteOrigin(site),
    clientIP,
    body: {
      site_id: site.id,
      contact: payload.contact,
      message: payload.message,
      sentiment: payload.sentiment || ''
    }
  });
  if (!response.ok) {
    throw new Error(`create_feedback_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function createMobileFeedback(config, site, mobileApp, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/public/mobile-feedback',
    method: 'POST',
    clientIP,
    body: {
      site_id: site.id,
      mobile_client_id: mobileApp.client_id || mobileApp.clientId,
      contact: payload.contact,
      message: payload.message,
      sentiment: payload.sentiment || '',
      screen: payload.screen || {},
      app: payload.app || {},
      context: payload.context || {}
    }
  });
  if (!response.ok) {
    throw new Error(`create_mobile_feedback_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function createWidgetTestFeedback(config, cookie, site, payload) {
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${site.id}/widget-test/feedback`,
    method: 'POST',
    cookie,
    body: {
      contact: payload.contact,
      message: payload.message,
      sentiment: payload.sentiment || ''
    }
  });
  if (!response.ok) {
    throw new Error(`create_widget_test_feedback_failed:${response.status}:${JSON.stringify(body)}`);
  }
  return body;
}

export async function createSubscription(config, site, payload) {
  const clientIP = payload && payload.clientIP ? payload.clientIP : nextClientIP();
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: '/public/subscriptions',
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
    path: '/public/subscriptions/confirm',
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
    path: `/public/subscriptions/confirm-link?token=${encodeURIComponent(token)}`,
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
  if (payload.screenResolution) {
    urlParams.set('screen_resolution', payload.screenResolution);
  }
  if (payload.viewport) {
    urlParams.set('viewport', payload.viewport);
  }
  if (payload.timezone) {
    urlParams.set('timezone', payload.timezone);
  }
  if (payload.locale) {
    urlParams.set('locale', payload.locale);
  }
  const headers = Object.assign({}, payload.headers || {});
  if (payload.userAgent) {
    headers['User-Agent'] = payload.userAgent;
  }
  const { response, payload: body } = await apiRequest({
    baseURL: config.baseURL,
    path: `/public/visits?${urlParams.toString()}`,
    method: 'GET',
    origin: resolveSiteOrigin(site),
    clientIP,
    headers
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

export async function fetchDeviceBreakdown(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/visits/devices`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`device_breakdown_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function fetchLocationDistribution(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/visits/locations`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`location_distribution_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function fetchTrafficReportSchedule(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/traffic-report-schedule`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`traffic_report_schedule_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

function portfolioTrafficReportQuery(reportId) {
  const normalizedReportId = typeof reportId === 'string' ? reportId.trim() : '';
  if (!normalizedReportId) {
    return '';
  }
  const params = new URLSearchParams();
  params.set('report_id', normalizedReportId);
  return `?${params.toString()}`;
}

export async function fetchPortfolioTrafficReports(config, cookie) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/reports/traffic/portfolio/reports',
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_reports_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function createPortfolioTrafficReport(config, cookie, report) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: '/api/reports/traffic/portfolio/reports',
    method: 'POST',
    cookie,
    body: report
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_report_create_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function updatePortfolioTrafficReport(config, cookie, reportId, report) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/reports/traffic/portfolio/reports/${encodeURIComponent(reportId)}`,
    method: 'PUT',
    cookie,
    body: report
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_report_update_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function fetchPortfolioTrafficReport(config, cookie, reportId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/reports/traffic/portfolio${portfolioTrafficReportQuery(reportId)}`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_report_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function fetchPortfolioTrafficReportSchedule(config, cookie, reportId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/reports/traffic/portfolio/schedule${portfolioTrafficReportQuery(reportId)}`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_report_schedule_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function saveTrafficReportSchedule(config, cookie, siteId, schedule) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/traffic-report-schedule`,
    method: 'PUT',
    cookie,
    body: schedule
  });
  if (!response.ok) {
    throw new Error(`traffic_report_schedule_save_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function savePortfolioTrafficReportSchedule(config, cookie, schedule, reportId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/reports/traffic/portfolio/schedule${portfolioTrafficReportQuery(reportId)}`,
    method: 'PUT',
    cookie,
    body: schedule
  });
  if (!response.ok) {
    throw new Error(`portfolio_traffic_report_schedule_save_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function rotateSentryToken(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/sentry/token`,
    method: 'POST',
    cookie
  });
  if (!response.ok) {
    throw new Error(`sentry_token_rotate_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function captureSentryError(config, token, event) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: '/sentry/errors',
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`
    },
    body: event
  });
  if (!response.ok) {
    throw new Error(`sentry_capture_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function captureBrowserSentryError(config, event, origin, clientIP) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: '/sentry/browser-errors',
    method: 'POST',
    origin,
    clientIP: clientIP || nextClientIP(),
    body: event
  });
  if (!response.ok) {
    throw new Error(`sentry_browser_capture_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function listSentryIssues(config, cookie, siteId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/sentry/issues`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`sentry_issues_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function getSentryIssueDetail(config, cookie, siteId, issueId) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/sentry/issues/${issueId}`,
    method: 'GET',
    cookie
  });
  if (!response.ok) {
    throw new Error(`sentry_issue_detail_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}

export async function updateSentryIssueStatus(config, cookie, siteId, issueId, status) {
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    path: `/api/sites/${siteId}/sentry/issues/${issueId}`,
    method: 'PATCH',
    cookie,
    body: { status }
  });
  if (!response.ok) {
    throw new Error(`sentry_issue_update_failed:${response.status}:${JSON.stringify(payload)}`);
  }
  return payload;
}
