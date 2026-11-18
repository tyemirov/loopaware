// @ts-check
(function () {
  const script = document.currentScript;
  if (!(script instanceof HTMLScriptElement)) throw new Error('count_script_missing');
  const scriptURL = new URL(script.src);
  const siteID = scriptURL.searchParams.get('site_id');
  if (!siteID) throw new Error('count_site_missing');
  /** @type {Readonly<Record<string, string>>} */
  const publicOrigins = Object.freeze({
    'https://loopaware.mprlab.com': 'https://loopaware-api.mprlab.com',
    'https://tyemirov.github.io': 'https://loopaware-api.mprlab.com'
  });
  const apiOrigin = publicOrigins[scriptURL.origin] || scriptURL.origin;
  const endpoint = new URL(`/public/sites/${encodeURIComponent(siteID)}/visit-counts`, apiOrigin);
  script.dataset.countState = 'loading';
  fetch(endpoint, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}',
    credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store', redirect: 'error'
  }).then((response) => {
    script.dataset.countState = response.status === 204 ? 'ready' : 'unavailable';
  }, () => { script.dataset.countState = 'unavailable'; });
}());
