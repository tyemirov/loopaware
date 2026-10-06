# LoopAware MCP Adoption

## Status And Scope

P003 defines this implementation plan. F017 owns implementation and source validation.
The user selected LoopAware as the consumer of released `github.com/tyemirov/utils/mcpauth`.
This plan excludes migrations in LLM Proxy, ISSUES.md, and TAuth.
It supersedes the consumer sequence in the [shared design](../../utils/.mprlab/MCP-INTEGRATION.md).

The recommended first scope contains six read-only tools.
Site mutations, team changes, subscriber exports, token rotation, probes, and notifications remain outside this scope.
Implementation does not authorize publication or production activation.

## Confirmed Source Facts

The source review used the primary LoopAware checkout on October 6, 2026.
The package manager resolved `utils@latest` to `v0.19.1`.
That released module contains `mcpauth` and uses MCP Go SDK v1.8.0.
LoopAware currently requires utils v0.17.1, TAuth v1.2.7, and Go 1.27.1.
Resolved versions record inspection evidence. Dependency commands must use `@latest` during implementation.

| Source | Confirmed behavior | Implementation consequence |
| --- | --- | --- |
| [Current authentication](../internal/api/auth.go) | LoopAware verifies TAuth sessions and identifies accounts by normalized email. | Associate OAuth subjects with these existing accounts. |
| [User model](../internal/model/models.go) | Email is the account primary key. Sites and memberships also use email. | Add an identity link without changing these canonical ownership fields. |
| [Site access](../internal/api/site_access.go) | Administrators, owners, creators, and team members can read permitted sites. | Use the same access policy for REST and MCP. |
| [REST handlers](../internal/api/admin.go) | Queries, authorization, response mapping, and favicon work share handlers. | Extract query services and keep transport-specific work outside them. |
| [Health reads](../internal/api/site_health_monitor.go) | The read handler returns stored monitor state. A separate operation runs probes. | Return stored observations without a probe. |
| [Sentry reads](../internal/api/sentry.go) | Issue detail includes raw diagnostic payloads. Lists have no pagination. | Define bounded MCP projections and pagination. |
| [Traffic profiles](../internal/api/traffic_profile.go) | Aggregate sites cannot expose individual visits or visitor metrics. | Apply this restriction before MCP response construction. |
| [Production manifest](deploy/resources.yml) | The API hostname has network restrictions. Its tenant declaration has no OAuth resource. | Declare public MCP paths separately and add tenant OAuth policy. |

TAuth browser session tokens use issuer `tauth`.
OAuth access tokens use the configured authorization-server URL as issuer.
These issuer values differ by design.
For one deployment and tenant, both flows use the same TAuth account ID as their subject.
TAuth verifies provider email before it issues the browser session.
Its public session claims do not contain an email-verification field.

The [Gateway tenant contract](../../mprlab-gateway/README.md) supports `tenant.oauth` resources, scopes, clients, and metadata documents.
The Gateway also supports access policy on individual route handlers.
This plan requires no Gateway schema change.
These source facts do not prove the current production runtime configuration.

## Ownership And Request Flow

| Owner | Responsibility |
| --- | --- |
| TAuth | Browser authentication, consent, authorization codes, tokens, refresh, revocation, and public keys |
| Official MCP SDK | Protocol parsing, discovery, schemas, tool dispatch, transport, and bearer middleware |
| `utils/mcpauth` | Verified token conversion, typed principal access, session binding, and safe verifier errors |
| LoopAware | Account association, role policy, site access, tool scopes, projections, limits, and diagnostics |

The browser keeps the current `mpr-ui` and TAuth authentication contract.
An MCP client obtains an OAuth token from TAuth.
Each MCP request passes through the TAuth validator and shared verifier.
The verifier resolves an existing identity link and constructs an immutable principal.
The tool applies its scope and the query service applies resource access.
The transport returns the explicit MCP projection.

Do not add a JWT validator, JWKS client, tool registry, or OAuth login flow to LoopAware.
Do not call LoopAware REST endpoints from MCP handlers.
Do not construct a synthetic Gin context.

## Account Association

Add one `UserIdentity` model and database table.
Use `(oauth_issuer, tenant_id, subject)` as its unique external key.
Associate that key with the existing normalized `User.Email`.
Add a uniqueness constraint for `(oauth_issuer, tenant_id, user_email)`.
Keep the association immutable during this implementation.

Create the association only during normal backend authorization of a verified TAuth browser session.
Use the session validator before any association work.
Require the expected tenant, a nonempty verified user ID, and a nonempty verified email.
Require agreement between the session user ID and its subject.
Use the configured OAuth issuer for the same TAuth deployment.
Do not copy the browser token issuer into the OAuth identity key.

Persist the account and association in one database transaction.
Accept an existing association only when all values match.
Reject a changed email, conflicting subject, or conflicting account association without an overwrite.
Return a safe operational failure when the transaction fails.
Do not continue with an authorized principal after a failed account or association write.
Keep avatar retrieval outside the identity transaction.

An MCP request cannot create, update, or repair an association.
Reject an unresolved subject before access to product data.
The account must first complete the ordinary dashboard flow.
Do not infer an email from a subject, provider identity, client argument, or token string.

No ownership migration or guessed backfill is necessary.
Existing accounts acquire associations through verified dashboard use.
Do not associate historical rows from email alone.
Account email changes or TAuth account merges require a separate, verified migration contract.

Resolve the linked account before each MCP request.
Derive administrator status from the current configured administrator policy.
The database does not supply administrator roles.
Read current site ownership and team membership during each tool operation.
Do not retain authorization decisions between requests.

Require a real TAuth browser-session to OAuth-token test for subject correspondence.
Use one configured deployment, tenant, and account in that test.
Do not treat matching configuration strings as sufficient identity proof.

## Endpoint, Protocol, And OAuth

Use these production identifiers from the existing API and TAuth declarations:

| Item | Selected contract |
| --- | --- |
| Protected resource and audience | `https://loopaware-api.mprlab.com/mcp` |
| Resource metadata | `https://loopaware-api.mprlab.com/.well-known/oauth-protected-resource/mcp` |
| OAuth issuer | `https://tauth-api.mprlab.com` |
| JWKS URL | `https://tauth-api.mprlab.com/oauth/jwks` |
| Tenant | `loopaware` |
| Transport | Stateless Streamable HTTP with JSON responses |
| Protocol | `2026-07-28` only |

Declare environment-specific values in the canonical backend YAML.
Add one `mcp` block with `resource_url`, `oauth_issuer`, `jwks_url`, and `allowed_origins`.
Use the existing `auth.tauth.tenant_id` as the tenant authority.
Derive the metadata URL from the protected resource URL.
Reject absent, inconsistent, or invalid configuration before the server starts.
Use local HTTP URLs only in explicit local and test configurations.

Construct one public `tauth/pkg/oauthvalidator` during startup.
Configure the exact issuer, audience, and JWKS URL.
Keep its `RequiredScopes` empty.
Use SDK middleware for the endpoint scope and tool handlers for tool scopes.
Verify the expected tenant after token validation.
Translate known token failures to `mcpauth.ErrUnauthenticated`.
Report unexpected failures once and return the shared safe public error.

The inspected TAuth validator classifies token parsing and key retrieval failures as `ErrInvalidToken`.
The adapter cannot recover operational causes that the validator does not expose.
Test the actual JWKS failure contract without an invented diagnostic cause.

Construct tokens through `mcpauth.NewVerifiedToken`.
Supply verified expiration and scopes with the resolved principal.
Construct the binding from the canonical issuer, tenant, subject, client ID, and grant ID tuple.
Keep that binding stable across refresh within the same grant.
Do not use the credential or its hash as the binding.
Stateless operation retains no protected MCP session.

Configure `Stateless: true`, `JSONResponse: true`, and `SupportedProtocolVersions: []string{"2026-07-28"}`.
Require exactly one `MCP-Protocol-Version: 2026-07-28` header on each MCP POST.
Pass the request body unchanged to the SDK after edge checks.
Use SDK checks for required modern metadata, matching versions, and removed methods.
The SDK version option alone does not reject every old initialization request.
Do not add a second JSON-RPC parser or a compatibility path.

Reject duplicate `Authorization` headers before SDK bearer handling.
Accept OAuth bearer credentials only on `/mcp`.
Do not accept dashboard cookies as MCP credentials.
Use `auth.RequireBearerToken` with endpoint scope `loopaware:mcp` and the metadata URL.
Use the SDK protected-resource metadata handler.
Advertise the selected issuer, resource, and supported scopes.
Keep metadata retrieval public.
Use `Cache-Control: no-store` on protected MCP responses.
Prevent shared caching of account-specific tool results.

Return `401` for missing or invalid bearer credentials and unresolved identities.
Return `403` for a valid token without the endpoint scope.
Return `400` for invalid protocol headers after successful bearer authorization.
Return `413` for an excessive request body.
Use safe MCP tool errors after protocol dispatch.
Distinguish scope denial, unavailable resources, invalid arguments, and operational failure with stable codes.
Do not return database, provider, or credential text in errors.

Extend LoopAware's existing `tauth_tenant` with the protected resource and six scopes.
Use TAuth's authorization-code flow with PKCE S256 and explicit resource selection.
Set the access-token lifetime to `5m`.
Set refresh-token and consent lifetimes to `720h`.
Keep authorization-request and code lifetimes under the existing TAuth server policy.
The inspected production policy uses `5m` for authorization requests and `1m` for codes.
OAuth revocation prevents further refresh through TAuth.
Locally verified access tokens can remain valid until their five-minute expiry.
Do not claim immediate access-token revocation from local JWT validation.
Enable TAuth client metadata documents for supported clients.
Declare static clients only when their actual client IDs and redirect URLs are known.
Do not invent production callback URLs or add a LoopAware OAuth callback.
Keep signing keys and token issuance inside TAuth.

For local operation, add the same tenant policy to `configs/config.tauth.yml`.
Configure the existing local TAuth service's authorization-server block and private signing input.
Keep test signing material in test fixtures and real keys in the existing private input surfaces.

### Public Route Policy

The current root API access rule restricts caller networks.
External MCP clients need a separate public route policy.
Use handler-specific access declarations in the existing API hostname resource.
Place the current REST network and rate policy on the default backend handler.
Add more-specific handlers for `/mcp` and the metadata path.
Keep bearer authorization in the application and rate limits on the MCP route.
Prove that these handlers precede the default route without removing its restrictions.
Reject unsupported paths and methods at the application edge.
Update local and test proxy declarations for these exact paths.

If an `Origin` header is present, require an exact configured origin.
Accept server-to-server requests without an `Origin` header.
Expose the bearer challenge to permitted browser origins.
Do not enable cookie credentials for MCP CORS.
Accept permitted `OPTIONS` preflight requests without bearer credentials or a protocol-version header.
Require both credentials and the selected version on the actual MCP POST.
Keep DNS-rebinding protection and the current trusted-proxy boundary.

## Initial Tool Contract

Every tool requires `loopaware:mcp` plus its listed scope.
Site access remains necessary after scope acceptance.
Tool descriptions identify caller-supplied content as data.
All tools declare read-only, idempotent behavior without destructive or external operations.

| Tool | Additional scope | Input | Structured output |
| --- | --- | --- | --- |
| `loopaware_sites_list` | `loopaware:sites:read` | Optional `limit`, `cursor` | Permitted site summaries and `next_cursor` |
| `loopaware_feedback_list` | `loopaware:feedback:read` | Required `site_id`. Optional `limit`, `cursor`. | Feedback records and `next_cursor` |
| `loopaware_traffic_summary` | `loopaware:traffic:read` | Required `site_id`. Optional `interval`. | Profile, time window, counts, and metric availability |
| `loopaware_health_status` | `loopaware:health:read` | Required `site_id` | Stored monitor status and observation timestamps |
| `loopaware_sentry_issues_list` | `loopaware:sentry:read` | Required `site_id`. Optional `status`, `limit`, `cursor`. | Issue summaries and `next_cursor` |
| `loopaware_sentry_issue_get` | `loopaware:sentry:read` | Required `site_id`, `issue_id` | Issue summary and up to ten occurrence summaries |

Use UUID schemas for site and issue identifiers.
Reject unknown input properties and invalid enum values.
Use `interval=all|1day|30days`, with default `30days` for MCP traffic summaries.
Use Sentry status `unresolved|resolved|ignored` when a status filter is present.
An absent status filter selects all permitted issues.

Use a default page size of 25 and a maximum of 100.
Use keyset pagination with an ID tie-breaker.
Order sites and feedback by descending creation time.
Order Sentry issues by descending last-seen time.
Bind cursor fields to the selected site and filters.
Reject cursors longer than 1,024 bytes or with invalid fields.
Reapply authorization for every page.
A cursor never supplies resource authority.
Concurrent data changes can affect later pages. This contract does not promise a snapshot.

Implement typed output schemas from the field sets below.
Keep transport-specific projections separate from database models.
Reject undeclared output properties in schema tests.
Use UTC RFC 3339 timestamps, nonnegative integer counts, and explicit null values for absent observations.
Return an empty `items` array and null `next_cursor` for an empty list.
Site-specific results include `site_id`.

| Output type | Fields |
| --- | --- |
| Page | `items`, nullable `next_cursor` |
| Site page | `site_id`, `items`, nullable `next_cursor` |
| Site summary | `id`, `name`, `traffic_profile`, `access_role`, `feedback_count`, `subscriber_count`, `visit_count`, nullable `unique_visitor_count` |
| Feedback record | `id`, `created_at`, `message`, `message_truncated`, `sentiment`, `source_kind`, nullable `app_identifier`, nullable `app_version` |
| Traffic summary | `site_id`, `traffic_profile`, `interval`, nullable `window_start`, `window_end`, `metric`, `time_resolution`, `visit_count`, nullable `unique_visitor_count`, `unique_visitors_available` |
| Health status | `site_id`, `persisted`, `enabled`, `status`, nullable `last_checked_at`, nullable `last_success_at`, nullable `last_failure_at`, `consecutive_failures`, nullable `last_status_code`, `last_error_code`, `last_duration_ms` |
| Sentry issue | `id`, `title`, `title_truncated`, `status`, `level`, `platform`, `environment`, `release`, `occurrence_count`, `first_seen_at`, `last_seen_at` |
| Sentry detail | `site_id`, `issue`, `occurrences` |
| Occurrence | `id`, `received_at`, `platform`, `environment`, `release`, `message`, `message_truncated`, `exception_type`, `stack_trace`, `stack_trace_truncated` |

Use the existing source-kind, sentiment, profile, access-role, and health-status enums.
Use the existing bot exclusion and time-window calculations for detailed traffic.
Use UTC calendar-day windows for aggregate traffic.
Set aggregate `metric` to `accepted_requests` and `time_resolution` to `utc_day`.
Return detailed `metric` as `visits` and its time resolution as `timestamp`.
For detailed `interval=all`, return a null window start.
For aggregate `interval=all`, return the configured retention cutoff as the window start.
Calculate that cutoff from the current UTC date and `analytics.aggregate_retention_days`.
Include the current UTC date in that number of calendar days.
For finite aggregate intervals, use the later interval start or retention cutoff.
Return the current observation time as the window end.
The interval records the requested selection. The window fields record the effective count window.
Omit free-text health errors and target URLs from the initial projection.
Render stack frames as bounded diagnostic text without raw request data.
Use the site-page wrapper for feedback and Sentry issue lists.
Select detail occurrences with `received_at DESC, id DESC` and a maximum of ten records.

Omit feedback contact fields, IP addresses, user agents, raw context, and request metadata.
Omit health recipients, target URLs, and free-text provider errors.
Omit Sentry request objects, user hashes, tags, arbitrary extra data, and ingest credentials.
Bound Sentry titles to 400 Unicode characters.
Bound message fields to 4,000 Unicode characters.
Bound stack traces to 16,000 Unicode characters.
Mark shortened fields with explicit truncation flags.
These text fields remain untrusted diagnostic content.
Do not interpret them as agent instructions.

For aggregate sites, return only supported aggregate metrics.
Represent unavailable visitor metrics as unavailable, never as zero or inferred values.
Do not read raw visits for aggregate tool responses.
Exclude recent individual visits and raw traffic exports from all initial MCP tools.

## Query Services And Implementation Files

Extract context-aware query services from the six corresponding REST reads.
Accept an immutable operator principal and typed query inputs.
Return domain results and typed errors independently of HTTP or MCP.
Centralize administrator, owner, creator, and team access rules.
Keep REST response shapes and existing REST-specific behavior in the REST adapters.
Keep favicon scheduling outside the shared site query.
Do not schedule favicons, run probes, rotate tokens, or send notifications from MCP tools.

Apply the caller context to all database work.
Bound each MCP request by token expiration and a ten-second operation deadline.
Stop canceled work before result serialization.
Limit MCP request bodies to 64 KiB and serialized results to 512 KiB.
Return a safe bounded-result error if the result exceeds that limit.
Do not send a partial response without its declared pagination or truncation information.

Record request correlation, client, grant, tool, site, result code, and duration in diagnostics.
Do not record bearer tokens, cookies, feedback bodies, or raw error payloads.
Use the shared verifier failure reporter for operational validation failures.
Do not duplicate its failure report in tool diagnostics.

| Area | Expected file scope |
| --- | --- |
| Identity | `internal/model/user_identity.go`, `internal/identity/`, `internal/storage/database.go` |
| Shared policy and queries | `internal/application/`, affected `internal/api/` read handlers and authentication |
| MCP adapter | `internal/mcp/` with startup constructor, schemas, tools, projections, and edge policy |
| Server wiring | `cmd/server/main.go`, `cmd/server/routes.go`, `internal/serverconfig/config.go` |
| Config and routes | `configs/config.loopaware.yml`, local TAuth/proxy declarations, `.mprlab/deploy/resources.yml`, test configs |
| Dependency and checks | `go.mod`, `go.sum`, `Makefile` |
| Acceptance | Go HTTP tests, real TAuth OAuth fixtures, Playwright scenarios, proxy policy tests |
| Documentation | `README.md`, `ARCHITECTURE.md`, this contract, terminology, and F017 |

## Test-Driven Execution

1. Read F017 and the current repository guides before implementation.
2. Record a satisfactory initial `make ci` result for the selected source state.
3. Add focused Make targets for MCP HTTP tests, OAuth flow tests, and new-package coverage.
4. Add the first public `/mcp` integration scenario before production changes.
5. Record its expected failure because MCP behavior is absent.
6. Resolve `github.com/tyemirov/utils/mcpauth@latest`, TAuth, and the official SDK through the package manager.
7. Keep the existing Go toolchain unless a resolved dependency requires a change.
8. Implement the identity association with real database integration tests.
9. Add characterization tests for affected REST reads before query extraction.
10. Extract query services while those REST tests pass.
11. Construct validators, shared verification, metadata, and the stateless SDK server during startup.
12. Add the six tool schemas, scope checks, projections, and public HTTP tests.
13. Add local OAuth configuration and route policy fixtures.
14. Verify identity correspondence through the real TAuth service and shared browser flow.
15. Complete the acceptance matrix and race tests.
16. Require 100 percent statement coverage for new MCP, identity, and query-service packages.
17. Add these checks to canonical CI without reducing existing coverage or scenarios.
18. Run final `make ci` after the last source, test, config, dependency, or build change.
19. Obtain independent architecture review of changes and evidence.
20. Update current documentation and close F017 after source acceptance passes.

Use locally signed OAuth tokens and a local JWKS endpoint for HTTP rejection scenarios.
Use the real public TAuth validator, SDK middleware, SDK client, and LoopAware database.
Use a real TAuth service for login, consent, code exchange, refresh, and subject correspondence.
Use a supported test-only verified account and provider fixture through the shared authentication flow.
Do not inject a final session cookie as browser-flow proof.
Keep ordinary dashboard authentication regression tests in the complete suite.

## Acceptance Matrix

| Area | Required observable evidence |
| --- | --- |
| Protocol | Official-client discovery, tools/list, and all six tools succeed at the selected version. |
| Version boundary | Missing, duplicate, old, and future headers fail. Missing or mismatched metadata fails. |
| Removed behavior | Old initialization fails. Old notifications cannot cause product work or establish a session. |
| Stateless transport | No session ID appears. Session deletion and unsupported methods fail without product work. |
| Token validation | Wrong key, algorithm, type, issuer, audience, tenant, and expiration cannot access data. |
| HTTP policy | Missing, malformed, or duplicate credentials fail. Cookies alone cannot authorize MCP. |
| Discovery | Public metadata and bearer challenges advertise the exact resource, issuer, and scopes. |
| Identity | Real browser and OAuth flows resolve the same account. Unresolved subjects cannot access data. |
| Identity transaction | Concurrent matching links succeed. Conflicting links, changed emails, and failed writes cannot authorize. |
| Role policy | Administrators use current configuration. Owners, creators, and team members see only permitted resources. |
| Access changes | A removed team member loses access on the next request. Scope never replaces site authorization. |
| Site isolation | A valid issue ID from another site cannot pass a selected-site detail request. |
| Scopes | Missing endpoint scope fails at HTTP. Each missing tool scope fails before its query. |
| Pagination | Bounds, cursor filters, malformed cursors, tie-breakers, and final pages match the declared contract. |
| Output | Typed schemas match. Restricted fields remain absent and long text has explicit truncation markers. |
| Aggregate privacy | Aggregate sites expose no visits, visitor identifiers, pages, devices, or location records. |
| Read-only behavior | No tool schedules favicons, probes targets, sends messages, or changes domain rows. |
| Cancellation | Disconnect, expiry, and deadline stop database work and prevent a late result. |
| Failures | Database and dependency failures remain safe. Diagnostics retain one operational report without credentials. |
| Routing | External MCP and metadata paths pass. Existing REST network restrictions still reject excluded callers. |
| Origins and limits | Unknown origins, excessive bodies, oversized results, and proxy rate limits match policy. |
| Browser preflight | Permitted OPTIONS succeeds without credentials. The subsequent POST still requires bearer and version checks. |
| Refresh and revocation | Refreshed tokens retain account access. Revocation prevents refresh without an immediate-JWT-revocation claim. |
| Regression | Existing REST, browser, mobile, security, config, and race gates remain satisfactory. |

## Open Decisions And Activation

The six-tool scope and conservative output projection are the recommended first release contract.
The user can extend this contract before F017 implementation starts.
Exact production clients and any static redirect declarations remain operational inputs.
Verify target-client support for `2026-07-28` before its activation.
Do not add an older protocol path to accommodate an unsupported client.

Source closure requires implemented behavior and the applicable passing tests.
It does not require production publication, deployment, or a live external client.
Activation later requires the prepared application artifact, tenant OAuth policy, public route policy, and actual client configuration.
Verify those production surfaces separately when activation is authorized.

## Plan Validation

The architecture review confirmed the package boundary and identified the identity transaction, issuer distinction, route policy, and strict-version checks.
Planning validation examines source contracts and changed documentation only.
This document does not establish implemented MCP behavior or production availability.
The final independent review found no remaining plan blockers.
The new plan, terminology, and issue entries passed scoped document checks and source review.
Governor retained twelve existing template differences. This task added no differences.
`git diff --check` passed. Application files and dependencies remain unchanged.

## References

- [Released package guide](../../utils/mcpauth/README.md)
- [TAuth OAuth validator](../../TAuth/pkg/oauthvalidator/validator.go)
- [TAuth session identity](../../TAuth/internal/authkit/jwthelper.go)
- [TAuth OAuth session resolver](../../TAuth/internal/authkit/oauth_browser_sessions.go)
- [TAuth authorization-server manifest](../../TAuth/.mprlab/deploy/resources.yml)
- [Gateway route policy test](../../mprlab-gateway/internal/lifecycle/selected_manifest_isolation_integration_test.go)
- [Official MCP Go SDK protocol guide](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/docs/protocol.md)
- [Official SDK version policy](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/server.go)
