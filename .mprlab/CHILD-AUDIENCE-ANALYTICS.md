# Restricted Analytics For Child-Audience Applications

## Status And Scope

P001 records this design. F016 owns its implementation.
The source review used LoopAware commit `11c4b6fda3f4a24a3be54870edefa451ff478d9e` on September 8, 2026.
F016 implements the contract below. Production activation requires the proxy checks and release verification in this document.

Allergy Wheel targets ages six and older.
Its owner requires automatic analytics, automatic fonts, and feedback through a parental gate.
The owner authorized a plan or implementation for necessary LoopAware changes.
On September 8, 2026, the owner selected automatic LoopAware counts for the native game.
Remove native Google Analytics. Keep browser Google Analytics and automatic fonts.

The LoopAware capability supplies automatic daily counts without persistent visitor records.
It applies to a selected site. Other sites retain their declared traffic profile.
Store approval and legal compliance require separate evidence.

## Current Evidence

| Source | Observed behavior | Required change |
| --- | --- | --- |
| `web/pixel.js` | Reads or creates a visitor identifier. Sends URL, referrer, timezone, locale, and display data. | Supply a client that sends only the site identifier. |
| `internal/api/visit_collector.go` | Adds the client IP address, user agent, location headers, and an exact event time. | Enforce the restricted contract before persistent storage. |
| `internal/model/visit.go` | Stores individual visits, including IP addresses and device data. | Store daily counts in a separate table. |
| `internal/api/middleware.go` | Writes the client IP address and user agent to each request log. | Remove identifying fields from collector logs, including error paths. |
| `internal/task/visit_rollup.go` | Contains a retention job. The reviewed production command does not construct this job. | Connect retention to the production scheduler and verify execution. |
| `internal/api/site_stats.go` | Calculates visitor counts and detailed reports from individual visits. | Declare which metrics the restricted profile can supply. |
| `internal/api/admin.go` | Site deletion does not explicitly delete visit rows, visit rollups, or subscribers. | Verify complete site-data deletion through the public API. |
| `web/privacy/index.html` | Describes adult accounts, retained visitor data, and a restriction on child data. | Distinguish operator accounts, restricted analytics, and parent feedback. |

The policy text alone cannot establish the required collection behavior.
The source review does not prove the deployed version, retained production data, or proxy log contents.

## Contract And Ownership

LoopAware owns the traffic profile, collector, persistent data, reports, retention, and service disclosures.
Allergy Wheel owns its client calls, audience, parental gate, store declarations, and native verification.
The gateway owns generic deployment and proxy configuration.
Application identifiers and child-audience policy remain outside gateway code.

### Site Profile

Add one `traffic_profile` field with the values `detailed` and `aggregate`.
Use `detailed` as the default for existing sites through one explicit schema migration.
Select the profile when the site is created.
Reject profile changes after creation.
Use a dedicated aggregate site for native analytics. Keep existing website and feedback sites separate.
Expose the current profile through the existing protected site representation.
Reject an unknown profile at the API boundary.

For an aggregate site, reject calls to the detailed visit collector.
Do not permit a request parameter to override the stored profile.
Treat the profile as collection policy, not proof of child-audience approval.

### Restricted Collector

Create `POST /public/sites/{site_id}/visit-counts` for aggregate sites.
Accept an empty JSON object. Reject query data, unknown body fields, and oversized bodies.
Validate the site and its configured origins before the count operation.
Return `204` only after the atomic count operation succeeds.
Return a typed error when the site profile does not permit the operation.
Keep public site identifiers separate from credentials.

Persist only the site identifier, UTC date, and accepted request count.
Use one unique database key for the site and date.
Use an atomic increment to preserve concurrent requests.
Keep exact event times, IP addresses, hashes of IP addresses, URLs, and device data outside this table.
Do not create a raw visit before the count operation.
Do not use a visitor identifier or an event identifier to remove duplicate requests.

The metric counts accepted requests. It does not measure unique persons, sessions, or verified installations.
The client sends one request at application startup and does not retry an uncertain response.
Network failure must not prevent the game from opening or operating offline.

### Client And Network Boundary

Document the restricted HTTP client contract and supply a browser adapter.
Allergy Wheel owns its bundled native HTTP adapter.
Keep cookies, browser storage, referrers, game preferences, and contact data outside the request.
Use credentials omission and a no-referrer policy where the client API supports these controls.
Do not load the detailed pixel as an intermediate step.
If a browser adapter reads a public profile, read it before any visitor or device data.

HTTP transport still supplies network information to the receiver.
Describe that transient processing accurately in the data inventory.
Use network identity only for bounded abuse prevention when necessary.
The aggregate collector has no visitor-based abuse counter.
Apply request-size limits without creating visitor records.

Use route-level log rules that also cover malformed requests, unknown sites, and rejected requests.
Permit route templates, response status, and aggregate service metrics in collector logs.
Exclude IP addresses, user agents, referrers, request bodies, and identifying query data.
Apply equivalent controls to the proxy and any external request processor.
Verify these controls before the profile is selected in production.

### Reports And Retention

Supply daily counts in the dashboard, site summaries, and scheduled reports.
Mark unique visitors, engagement, device reports, and location reports as unavailable for aggregate sites.
Do not represent an unavailable metric as zero.
Define mixed-profile portfolio reports so their totals do not imply complete unique-visitor coverage.
Export only daily counts for aggregate sites.

Use 90 days as the default retention period for daily counts.
Declare this value in the canonical backend configuration.
Run the cleanup job at startup and each day with an injected clock.
Expose cleanup failures through operational status without request data.
Verify retention through the production command, not only a direct job test.

Existing sites keep their detailed profile. Profile conversion is outside F016.
The native analytics site starts with no historical visitor records.
Site deletion removes its daily counts and related site data in one transaction.
A restored database runs the same retention check before the service accepts requests.
Backups and deletion requests remain subject to the service's operator procedures.

### Parent Feedback

Keep feedback separate from automatic analytics.
Permit parent contact details and message content only through the documented parent action.
The existing feedback service has a separate data declaration and retention policy.
Its parent-supplied contact details and messages remain outside the aggregate analytics site.
Keep feedback records outside visitor matching and analytics reports.
Verify the full deletion path without sending test messages to the production service.

A parental gate controls access. It does not prove parental consent or that the sender is an adult.
The selected collection basis and disclosures must cover the actual feedback behavior.

## Application And Provider Requirements

The LoopAware profile cannot control Google Analytics or Google Fonts requests.
Google documents measurements without cookies when analytics storage consent is denied.
These requests can still contain transport and page information.
See [Google consent mode](https://developers.google.com/tag-platform/security/concepts/consent-mode).

Google Play requires permitted child-audience services and accurate data declarations.
See [Google Play Families requirements](https://support.google.com/googleplay/android-developer/answer/9893335?hl=en).
Apple limits analytics that transmit identifiable child, network, or device information in child-directed applications.
See [Apple review guidelines](https://developer.apple.com/app-store/review/guidelines/).

The FTC describes a limited exception for analytics used only to support internal operations.
The exception has conditions on collected data and its use.
It does not establish worldwide compliance or store approval for this application.
See [FTC COPPA guidance](https://www.ftc.gov/business-guidance/resources/complying-coppa-frequently-asked-questions).

For Allergy Wheel, verify these subjects separately:

1. Replace native Google Analytics with restricted LoopAware counts.
2. Verify that the native candidate makes no Google Analytics requests.
3. Verify font request data and the applicable disclosures.
4. Verify parent feedback collection, retention, and deletion.
5. Capture network and storage evidence from the final native candidate on both platforms.
6. Update privacy text and store declarations from that evidence.

Do not substitute a general no-data-collection declaration for the actual application inventory.

## Implementation Sequence

F016 owns one complete restricted analytics capability.
Implement the following steps in order:

1. Add failing HTTP and client integration cases for restricted collection, rejected detailed collection, and log redaction.
2. Implement the typed site profile, daily table, atomic collector, and native client contract.
3. Connect the protected site API, dashboard, mobile operator client, exports, and scheduled reports.
4. Implement bounded retention and complete site-data deletion.
5. Verify the proxy contract and record any required change under its owning repository.
6. Update the service documentation and privacy text to match verified behavior.
7. Run the focused repository targets and final `make ci`.

Repository completion requires the requested code, tests, and documentation.
Production deployment and store submission remain separate operations.

After repository validation, deploy the LoopAware capability through its canonical lifecycle.
Verify the deployed collector and proxy with synthetic data on a dedicated test site.
Then connect Allergy Wheel to the verified contract and build a new sealed release.
Run `make publish` against that release after the application verification and declarations are completed.
Verify store processing, review, and public availability as separate results.

## Acceptance Cases

| Boundary | Required evidence |
| --- | --- |
| Collection | Concurrent valid requests produce the exact daily count and no individual visit rows. |
| Rejection | Unknown fields, incorrect profiles, disallowed origins, and oversized input produce no count change. |
| Client | Startup uses the restricted endpoint without visitor, device, game, or contact data. |
| Failure | Offline operation and provider errors preserve the complete game flow. |
| Logs | Success, validation failure, proxy rejection, and database failure expose no identifying request fields. |
| Reports | Aggregate counts remain available, and unavailable visitor metrics are explicit. |
| Retention | The production scheduler removes expired rows at the specified boundary. |
| Removal | Site deletion removes its records and preserves unrelated sites. |
| Backup restore | Restored data remains subject to the same removal and retention rules. |
| Native application | Android and iOS evidence matches the final artifact and store declarations. |

## Current Release Boundary

Allergy Wheel build `1788901126`, version `1.0.0`, was uploaded to Google Play as a draft.
Its Google listing includes three native screenshots and is ready for review.
The Data safety form remains incomplete.
The current candidate still uses the existing analytics contract.
This plan does not establish permission to submit inaccurate declarations or prove that the draft is ready for public release.
