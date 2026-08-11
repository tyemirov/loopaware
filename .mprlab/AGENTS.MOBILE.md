# AGENTS.MOBILE.md

## Scope

This guide gives mobile client rules for iOS and Android applications.

Use it for native projects, React Native, Expo, app stores, mobile authentication, purchases, notifications, deep links, and platform code.

Obey root `AGENTS.md` and `.mprlab/POLICY.md` for shared workflow and confident-programming rules.

## Principles

- Treat mobile as a first-class product surface, not a browser clone.
- Preserve one canonical mobile contract for each workflow. Do not add legacy aliases, compatibility storage reads, or platform fallback behavior unless the product contract explicitly requires it.
- Validate at mobile boundaries and keep screens/components focused on rendering validated state and emitting user intent.
- Represent platform capabilities as explicit typed objects or closed action sets instead of scattering unrelated `Platform.OS` branches.
- Make iOS and Android divergence visible in named adapters, config, tests, and issue notes.
- Keep native/generated artifacts and source-owned configuration separate.

## Boundary Validation

Validate once at these edges:

- API responses and mobile backend clients.
- App configuration and environment manifests.
- Deep links, universal links, app links, and OAuth callback URLs.
- Auth session results, token exchanges, refresh paths, and sign-out flows.
- Secure storage reads and writes.
- Push notification payloads, permissions, and registration tokens.
- In-app purchase products, receipts, entitlement payloads, and restore flows.
- Clipboard, camera, photos, files, contacts, location, and other device APIs.
- Native module responses and bridge payloads.

After validation, core mobile state and UI code must use domain values, not raw payload maps.

## State And Storage

- Store secrets and tokens only in platform secure storage.
- Do not use generic local storage for credentials, refresh tokens, purchase receipts, or private auth state.
- Keep persisted shapes current. If a persisted schema changes, write a bounded migration into the current schema and remove compatibility bridges after migration.
- Do not fabricate empty user, entitlement, subscription, token, or config objects when a boundary payload is invalid.
- Keep cache invalidation and refresh behavior explicit.

## Platform Boundaries

- Keep platform-specific behavior in named adapters or modules.
- Do not use inline conditionals that mix iOS, Android, and shared business logic in screens.
- Name platform differences in tests and documentation when behavior legitimately diverges.
- Keep permissions, entitlements, associated domains, URL schemes, intent filters, and store identifiers aligned with checked-in config.
- Do not assume simulator behavior proves device, store, OAuth, notification, or production callback behavior.

## Build And Store Publication

- Build each iOS and Android store artifact on an operator-controlled build host.
- Use the native platform toolchain to create a signed `.ipa` or `.aab` artifact.
- Seal the artifact identity before publication.
- Publish the sealed artifact directly to App Store Connect or Google Play.
- Keep release and publication commands in repository Make targets or package scripts.
- Release, publication, and deployment must consume prebuilt mobile store artifacts.

## UI And UX

- Screens must receive validated state and call explicit commands.
- Keep loading, error, empty, offline, unauthorized, and entitlement-required states explicit.
- Do not use silent retries or hidden fallbacks that show a false successful state.
- Handle safe areas, keyboard movement, orientation, accessibility labels, dynamic type, and reduced-motion settings intentionally.
- Keep destructive, purchase, auth, and permission actions explicit.

## Testing And Validation

- Prefer tests through mobile entrypoints, mobile backend clients, config validators, and app-flow harnesses.
- Use `.mprlab/POLICY.md` for validation.
- During the change, run the smallest mobile target that validates the changed contract.
- Validate auth, deep links, purchases, secure storage, notification permissions, and API contract parsing at their boundaries.
- When touching iOS or Android generated/native projects, run the platform-specific prepare/build command documented by the repo.
- Do not claim production mobile readiness from web-only, simulator-only, or unit-only evidence.

## Review Checklist

- [ ] Boundary payloads are validated once and converted to domain values.
- [ ] Screens render validated state and emit explicit intent.
- [ ] Secrets use platform secure storage only.
- [ ] Platform-specific code is isolated in named adapters.
- [ ] iOS and Android differences are explicit and tested or documented.
- [ ] Native/generated files were changed only through documented ownership paths.
- [ ] Store publication consumes signed prebuilt mobile store artifacts.
- [ ] Repo-native mobile validation passed or blockers are documented.
