# LA-200 — mpr-ui gaps for GAuss/GAuth integration

LoopAware already delegates authentication to GAuss: every login funnels through `/login` (GAuss renders Google’s OAuth UI, manages nonce storage, and issues the session cookie). The new `mpr-ui` header/footer DSL assumes a different stack – Google Identity Services in the browser plus the TAuth helper described in `tools/mpr-ui/docs/custom-elements.md` and `tools/mpr-ui/docs/integration-guide.md`. When we try to adopt `<mpr-header>` + `<mpr-footer>` unchanged we cannot hook those components into the existing GAuss flow, because the declarative API only speaks “GIS + TAuth”.

This note captures what is missing in the DSL and suggests concrete extensions so the LoopAware dashboard can load the newest mpr-ui bundle without re-implementing GAuss.

## 1. Header hard-depends on GIS and the TAuth helper

- `<mpr-header>` calls `createAuthHeader` immediately (see `tools/mpr-ui/mpr-ui.js`, `createAuthHeader` around line 2182). That helper:
  - Requires `site-id` (a Google client ID) and injects `https://accounts.google.com/gsi/client`.
  - POSTs `base-url + /auth/nonce` then `base-url + /auth/google` exactly like the TAuth reference flow.
  - Requires `window.initAuthClient` (served by TAuth at `/static/auth-client.js`) to bootstrap/refresh sessions.
- None of these moving parts exist in GAuss. LoopAware already proxies `/auth/google`/`/auth/logout` to GAuss, but those HTTP handlers immediately redirect to GAuss/Google; we never expose nonce endpoints or TAuth’s JS helper.

**Missing DSL capability:** opt out of GIS/TAuth entirely and let the header operate in a “server-managed” mode for GAuss.

**Coding suggestion:** add an `auth-mode` attribute with at least two values:

```html
<!-- New mode for GAuss -->
<mpr-header
  auth-mode="gauss"
  login-url="/auth/google"
  logout-url="/auth/logout"
  session-endpoint="/api/me"
  profile-field-map='{"name":"user.name","email":"user.email","avatar":"user.avatar_url"}'
></mpr-header>
```

When `auth-mode="gauss"`:

1. Skip `createAuthHeader` (and the GIS script) entirely.
2. Poll `session-endpoint` to learn whether the GAuss session cookie is present; update the profile chip with the mapped fields.
3. Render the CTA as `<a href="/auth/google">` so GAuss keeps controlling the OAuth redirect.
4. Emit `mpr-ui:auth:authenticated` / `mpr-ui:auth:unauthenticated` based on GAuss’ JSON instead of TAuth state.

## 2. No way to feed LoopAware’s existing profile data into `<mpr-header>`

Even if we could disable GIS, the current implementation only understands the profile payload that TAuth returns (`{ name, email, picture, expires_at }`). LoopAware already exposes `/api/me` with GAuss-derived fields (`role`, `avatar.url`, etc.) but there is no declarative way to teach the header how to consume it.

**Missing DSL capability:** configurable profile wiring so the header can display GAuss metadata (display name, avatar, role badge) without rewriting `mpr-ui`.

**Coding suggestion:** introduce a `profile-endpoint` + `profile-shape` attribute. Example:

```html
<mpr-header
  auth-mode="gauss"
  profile-endpoint="/api/me"
  profile-shape='{
    "name": "payload.name",
    "email": "payload.email",
    "avatar": "payload.avatar.url",
    "role": "payload.role"
  }'
></mpr-header>
```

`mpr-ui` would fetch `profile-endpoint`, evaluate the mapping (dot-notation path, similar to how Alpine’s `x-model` modifiers work), and populate the profile chip. The same mapping can drive new CSS hooks (`data-mpr-header-role="admin"`) so LoopAware can skin the GAuss role badge declaratively.

## 3. Logout/sign-out remains hardwired to TAuth’s POST contract

`mpr-header` currently POSTs `base-url + logout-path` (`createAuthHeader` → `performLogout`). GAuss, however, expects a GET redirect to `/logout` which clears the GAuss session and then redirects back to `/login`. Re-implementing GAuss as a POST API would break every other GAuss consumer.

**Missing DSL capability:** configure the logout strategy (GET redirect vs. POST fetch) and optionally skip the fetch entirely when GAuss already serves the right HTML.

**Coding suggestion:** add a `logout-mode` attribute with two strategies:

```html
<mpr-header
  auth-mode="gauss"
  logout-mode="link"
  logout-url="/logout?return=/login"
></mpr-header>
```

- `logout-mode="fetch"` (default) keeps today’s POST behaviour.
- `logout-mode="link"` renders the button as a standard `<form method="get">` or `<a>` and lets GAuss own the redirect chain.

## 4. Footer/theme wiring already works but needs documentation

`<mpr-footer>` does not touch authentication; LoopAware can drop it in today as long as we load `mpr-ui.css` first, then `mpr-ui.js` (the bundle auto-registers the custom elements). No Alpine wiring or `mprFooter()` factory is required on `mpr-ui` v0.2.0+.

## Next steps

1. Extend `mpr-ui` with the GAuss-friendly attributes above.
2. Once the bundle ships the new DSL, replace the Bootstrap header/footer snippets in LoopAware’s static pages under `web/` with `<mpr-header>` / `<mpr-footer>`.
3. Delete the bespoke theme toggle + footer wiring scattered across `web/` markup as the new components take over.

Until `auth-mode="gauss"` (or equivalent) lands upstream we cannot adopt the latest mpr-ui header in LoopAware without breaking Google Sign-In.
