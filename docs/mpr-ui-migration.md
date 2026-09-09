# Shared UI Migration

I042 prepares LoopAware for the mpr-ui I009 publication.

## Candidate

The tests use revision `768f25936497c5aabd426197d21c2100b6e5d9a1` from mpr-ui B069.
The test helper verifies each candidate file before browser use.
Production URLs retain literal `@latest`.

| File | SHA-256 |
| --- | --- |
| `mpr-ui-config.js` | `3f56fbd212a516d2bd8b0b95f73ae7ad82952c10d8d5f4e6f8b44d3233f01304` |
| `mpr-ui.js` | `3e725dbe911470ca934cb46456369479b6ac232eee5ccba2582bf8d939259ae8` |
| `mpr-ui.css` | `351bbf6c15054528a651571d8c8bd85536eea76c3e574f9335e6cd413878923f` |

## Application Contract

Both config environments use `auth.providers`.
Google retains its client ID, login path, and nonce path.
Apple and password providers are disabled.
The common config retains the tenant, TAuth origin, logout path, and session path.

The Go footer renderer and 41 static footers use the sectioned `menu` contract.
The conversion preserves project labels, destinations, and utility links.
The attribution is in the menu button.
The asset audit checks 49 HTML entries and 44 MPR UI declarations.
Other external asset pins, integrity checks, and CSP checks remain active.

Protected management requests use `MPRUI.authenticatedFetch` with the header.
The server installs authentication middleware before the protected route handlers.
The client declares `mutationReplay: "authorization-before-domain-work"` for this API contract.
Dashboard startup waits for shared orchestration and authentication.
After session recovery, the application clears its ready transition.
The inactivity logout flow reads the current `auth-config` attribute.

## Local Evidence

The config and footer browser tests failed before the source changes.
The Go renderer test failed before the menu conversion.
The recovery test detected two obsolete refresh requests before the transport change.
The new tests then passed against the shared candidate.
All 100 authentication, logout, and browser-security checks passed.
Final `make ci` passed, including all 472 integration scenarios, Go race checks, and mobile preparation checks.
The complete local log is `/tmp/loopaware-i042-ci-final2.log`.

The four authentication scenarios cover frontend and direct TAuth origins at 390px and 1280px.
Each scenario verifies Google login, reload, read recovery, mutation recovery, one persisted site, and logout.
The tests use the real application, Go API, and local database.
The Google and TAuth fixtures control the external authentication responses.
The API validates signed local session cookies.
These tests establish the application contract under those fixture conditions.
Real Google acceptance remains a separate central I009 gate.

Run the focused browser scenarios through the existing integration target:

```bash
LOOPAWARE_TEST_SUITE=test:shared-ui make test-integration
```

## Public Evidence And Central Activation

The [public observations](mpr-ui-public-assets-2026-09-09.json) record seven HTTP 200 responses from one network location.
The observation date is September 9, 2026.
The website responses declare a 600-second cache lifetime.
The shared CDN responses declare `max-age=604800, s-maxage=43200`.
All three published shared asset digests differ from the candidate digests.
These observations establish HTTP responses and asset identities.

Central I009 owns the shared publication and production acceptance gates:

1. Qualify each consumer against one final shared candidate.
2. Record the final consumer commit and native CI result.
3. Obey the central maintenance and cache plan for coordinated publication.
4. Have the operator publish the shared assets and the LoopAware Pages artifact.
5. Verify the release marker and public bytes from each required network location.
6. Verify real Google login, restoration, protected requests, and logout in the production browser.

I042 closes after its repository changes and applicable validation pass.
