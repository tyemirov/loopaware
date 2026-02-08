// @ts-check
import { buildSessionCookie } from './auth.js';

export async function applySessionCookie(context, config, user) {
  const cookie = buildSessionCookie(config, user);
  await context.addCookies([cookie]);
  return cookie;
}

export async function setLocalStorage(page, entries) {
  const payload = entries || {};
  await page.addInitScript((values) => {
    if (!window || !window.localStorage) {
      return;
    }
    Object.entries(values).forEach(([key, value]) => {
      if (value === null || typeof value === 'undefined') {
        window.localStorage.removeItem(key);
        return;
      }
      window.localStorage.setItem(key, String(value));
    });
  }, payload);
}

export async function waitForVisible(page, selector) {
  const locator = page.locator(selector);
  await locator.waitFor({ state: 'visible' });
  return locator;
}

export async function waitForHidden(page, selector) {
  const locator = page.locator(selector);
  await locator.waitFor({ state: 'hidden' });
  return locator;
}

export async function getBoundingBox(page, selector) {
  const locator = await waitForVisible(page, selector);
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error(`missing_bounding_box:${selector}`);
  }
  return box;
}

export function parseRgb(value) {
  const match = /rgb\((\d+),\s*(\d+),\s*(\d+)\)/.exec(String(value || ''));
  if (!match) {
    throw new Error(`invalid_rgb:${value}`);
  }
  return { red: Number(match[1]), green: Number(match[2]), blue: Number(match[3]) };
}

export async function injectFetchInterceptor(page) {
  await page.addInitScript(() => {
    if (typeof window !== 'object' || !window) {
      return;
    }
    if (!window.__loopawareFetchIntercept) {
      window.__loopawareFetchIntercept = { requests: [], storageKey: '__loopawareFetchRequests' };
    }
    const intercept = window.__loopawareFetchIntercept;
    if (!intercept.originalFetch && typeof window.fetch === 'function') {
      intercept.originalFetch = window.fetch;
    }
    if (!intercept.originalApiFetch && typeof window.apiFetch === 'function') {
      intercept.originalApiFetch = window.apiFetch;
    }
    intercept.requests = [];
    const storageKey = intercept.storageKey || '__loopawareFetchRequests';
    const persistRequests = () => {
      if (typeof sessionStorage === 'undefined') {
        return;
      }
      try {
        sessionStorage.setItem(storageKey, JSON.stringify(intercept.requests));
      } catch (error) {}
    };
    persistRequests();
    const captureRequest = (resource, init) => {
      const record = { url: '', method: 'GET', body: '', status: 0 };
      if (typeof resource === 'string') {
        record.url = resource;
      } else if (resource && typeof resource.url === 'string') {
        record.url = resource.url;
        if (resource.method && typeof resource.method === 'string') {
          record.method = resource.method;
        }
      }
      if (init && typeof init.method === 'string') {
        record.method = init.method;
      }
      if (init && typeof init.body === 'string') {
        record.body = init.body;
      }
      intercept.requests.push(record);
      persistRequests();
      return record;
    };
    const wrapFetchLike = (originalFn) => {
      return function(resource, init) {
        const record = captureRequest(resource, init);
        let result;
        try {
          result = originalFn.apply(this, arguments);
        } catch (error) {
          record.status = 0;
          persistRequests();
          throw error;
        }
        if (result && typeof result.then === 'function') {
          return result
            .then((response) => {
              if (response && typeof response.status === 'number') {
                record.status = response.status;
                persistRequests();
              }
              return response;
            })
            .catch((error) => {
              record.status = 0;
              persistRequests();
              throw error;
            });
        }
        if (result && typeof result.status === 'number') {
          record.status = result.status;
          persistRequests();
        }
        return result;
      };
    };
    if (typeof intercept.originalFetch === 'function') {
      window.fetch = wrapFetchLike(intercept.originalFetch);
    }
    if (typeof intercept.originalApiFetch === 'function') {
      window.apiFetch = wrapFetchLike(intercept.originalApiFetch);
    }
  });
}

export async function readFetchRequests(page) {
  return page.evaluate(() => {
    let combined = [];
    if (window.__loopawareFetchIntercept && Array.isArray(window.__loopawareFetchIntercept.requests)) {
      combined = combined.concat(window.__loopawareFetchIntercept.requests);
    }
    let storageKey = '__loopawareFetchRequests';
    if (window.__loopawareFetchIntercept && typeof window.__loopawareFetchIntercept.storageKey === 'string') {
      storageKey = window.__loopawareFetchIntercept.storageKey;
    }
    if (typeof sessionStorage !== 'undefined') {
      try {
        const stored = sessionStorage.getItem(storageKey);
        if (stored) {
          const parsed = JSON.parse(stored);
          if (Array.isArray(parsed)) {
            combined = combined.concat(parsed);
          }
        }
      } catch (error) {}
    }
    return combined;
  });
}
