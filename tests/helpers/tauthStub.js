// @ts-check

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ sessionCookieName?: string }} config
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number, exchangeDelayMs?: number, sessionCookieValue?: string }} [options]
 * @returns {Promise<void>}
 */
export async function installTauthStub(page, config, options) {
  const browserPage = /** @type {any} */ (page);
  const resolvedOptions = options || {};
  const sessionCookieName = config.sessionCookieName || 'app_session';
  const sessionCookieValue = typeof resolvedOptions.sessionCookieValue === 'string'
    ? resolvedOptions.sessionCookieValue.trim()
    : '';
  const signature = JSON.stringify({
    sessionCookieName,
    sessionCookieValue,
    delayMs: resolvedOptions.delayMs || 0,
    bootstrapDelayMs: resolvedOptions.bootstrapDelayMs || 0,
    currentUserDelayMs: resolvedOptions.currentUserDelayMs || 0,
    exchangeDelayMs: resolvedOptions.exchangeDelayMs || 0
  });
  if (browserPage.__loopawareTauthEndpointStubSignature === signature) {
    return;
  }
  if (typeof browserPage.__loopawareTauthEndpointStubSignature === 'string') {
    throw new Error('tauth_stub_already_installed_with_different_options');
  }
  browserPage.__loopawareTauthEndpointStubSignature = signature;
  await page.addInitScript(() => {
    const win = /** @type {any} */ (window);
    if (!win.__loopawareTestTauthRuntime || typeof win.__loopawareTestTauthRuntime !== 'object') {
      win.__loopawareTestTauthRuntime = { profile: null, exchangeProfile: null, nonceCounter: 0 };
    }
  });

  const endpointDelayMs = Number.isFinite(resolvedOptions.delayMs) ? Math.max(0, Number(resolvedOptions.delayMs)) : 0;
  const bootstrapDelayMs = Number.isFinite(resolvedOptions.bootstrapDelayMs)
    ? Math.max(0, Number(resolvedOptions.bootstrapDelayMs))
    : 0;
  const currentUserDelayMs = Number.isFinite(resolvedOptions.currentUserDelayMs)
    ? Math.max(0, Number(resolvedOptions.currentUserDelayMs))
    : 0;
  const exchangeDelayMs = Number.isFinite(resolvedOptions.exchangeDelayMs)
    ? Math.max(0, Number(resolvedOptions.exchangeDelayMs))
    : 0;
  let nonceCounter = 0;

  function delay(milliseconds) {
    const duration = Math.max(0, Number(milliseconds) || 0);
    if (duration <= 0) {
      return Promise.resolve();
    }
    return new Promise((resolve) => {
      setTimeout(resolve, duration);
    });
  }

  function decodeBase64Url(input) {
    const normalized = String(input || '').replace(/-/g, '+').replace(/_/g, '/');
    const padding = normalized.length % 4;
    const padded = padding === 2 ? `${normalized}==` : padding === 3 ? `${normalized}=` : normalized;
    if (padding === 1) {
      return '';
    }
    try {
      return Buffer.from(padded, 'base64').toString('utf8');
    } catch (error) {
      return '';
    }
  }

  function profileFromSessionCookieValue(cookieValue) {
    const parts = String(cookieValue || '').split('.');
    if (parts.length < 2) {
      return null;
    }
    const payload = decodeBase64Url(parts[1]);
    if (!payload) {
      return null;
    }
    let claims = null;
    try {
      claims = JSON.parse(payload);
    } catch (error) {
      return null;
    }
    if (!claims || typeof claims !== 'object') {
      return null;
    }
    const email = typeof claims.user_email === 'string' ? claims.user_email.trim() : '';
    const display = typeof claims.user_display_name === 'string' ? claims.user_display_name.trim() : email;
    const avatarUrl = typeof claims.user_avatar_url === 'string' ? claims.user_avatar_url.trim() : '';
    const userId = typeof claims.user_id === 'string' ? claims.user_id.trim() : '';
    const roles = Array.isArray(claims.user_roles) ? claims.user_roles.slice() : [];
    if (!email && !display && !avatarUrl && !userId) {
      return null;
    }
    return {
      user_id: userId,
      user_email: email,
      email,
      display,
      avatar_url: avatarUrl,
      roles
    };
  }

  function cookieValueFromHeader(cookieHeader) {
    const prefix = `${sessionCookieName}=`;
    const parts = String(cookieHeader || '').split(';');
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.startsWith(prefix)) {
        return trimmed.slice(prefix.length);
      }
    }
    return '';
  }

  function profileFromRequest(route) {
    const cookieHeader = route.request().headers().cookie || '';
    return profileFromSessionCookieValue(cookieValueFromHeader(cookieHeader));
  }

  function defaultExchangeProfile() {
    return profileFromSessionCookieValue(sessionCookieValue) || {
      user_id: 'test-user',
      user_email: 'user@example.com',
      email: 'user@example.com',
      display: 'Test User',
      avatar_url: '',
      roles: []
    };
  }

  function sessionCookieHeader() {
    if (!sessionCookieValue) {
      return '';
    }
    return `${sessionCookieName}=${sessionCookieValue}; Path=/; SameSite=Lax`;
  }

  async function fulfillJSON(route, status, body, headers) {
    await route.fulfill({
      status,
      contentType: 'application/json; charset=utf-8',
      headers: headers || {},
      body: JSON.stringify(body)
    });
  }

  async function fulfillNoContent(route) {
    await route.fulfill({
      status: 204,
      body: ''
    });
  }

  await page.route('**/tauth.js', async (route) => {
    await route.fulfill({
      status: 410,
      contentType: 'text/plain; charset=utf-8',
      body: 'LoopAware tests must not load tauth.js'
    });
  });

  await page.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (!['/me', '/auth/session', '/auth/refresh', '/auth/nonce', '/auth/google', '/auth/logout'].includes(url.pathname)) {
      await route.fallback();
      return;
    }

    await delay(endpointDelayMs);

    if (url.pathname === '/auth/nonce') {
      nonceCounter += 1;
      await fulfillJSON(route, 200, { nonce: `test-nonce-${nonceCounter}` });
      return;
    }

    if (url.pathname === '/me') {
      await delay(Math.max(bootstrapDelayMs, currentUserDelayMs));
      const profile = profileFromRequest(route);
      if (!profile) {
        await fulfillJSON(route, 401, { error: 'unauthorized' });
        return;
      }
      await fulfillJSON(route, 200, profile);
      return;
    }

    if (url.pathname === '/auth/session') {
      await delay(Math.max(bootstrapDelayMs, currentUserDelayMs));
      const profile = profileFromRequest(route);
      if (!profile) {
        await fulfillNoContent(route);
        return;
      }
      await fulfillJSON(route, 200, profile);
      return;
    }

    if (url.pathname === '/auth/refresh') {
      await delay(Math.max(bootstrapDelayMs, currentUserDelayMs));
      const profile = profileFromRequest(route);
      if (!profile) {
        await fulfillJSON(route, 401, { error: 'unauthorized' });
        return;
      }
      await fulfillJSON(route, 200, { ok: true });
      return;
    }

    if (url.pathname === '/auth/google') {
      await delay(exchangeDelayMs);
      let payload = {};
      try {
        payload = JSON.parse(request.postData() || '{}');
      } catch (error) {
        await fulfillJSON(route, 400, { error: 'invalid_json' });
        return;
      }
      const credential = typeof payload.google_id_token === 'string' ? payload.google_id_token : '';
      const nonceToken = typeof payload.nonce_token === 'string' ? payload.nonce_token : '';
      const nonceBoundCredential = `stub-google-credential::${nonceToken}`;
      if (!credential || !nonceToken || credential !== nonceBoundCredential) {
        await fulfillJSON(route, 400, { error: 'invalid_credential' });
        return;
      }
      const headers = {};
      const cookieHeader = sessionCookieHeader();
      if (cookieHeader) {
        headers['Set-Cookie'] = cookieHeader;
      }
      await fulfillJSON(route, 200, defaultExchangeProfile(), headers);
      return;
    }

    if (url.pathname === '/auth/logout') {
      const expireHeader = `${sessionCookieName}=; Path=/; Max-Age=0; SameSite=Lax`;
      await fulfillJSON(route, 200, { ok: true }, { 'Set-Cookie': expireHeader });
      return;
    }

    await route.fallback();
  });
}
