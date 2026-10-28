# LoopAware Deepmerge Correction

This package uses the upstream sources from `deepmerge@4.3.1`.
The package name and version remain `deepmerge` and `4.3.1`.
The upstream MIT license remains in `license.txt`.
`PROVENANCE.json` records the source, advisory, review date, and correction owner.

The correction rejects `__proto__` in both object copy loops before the loops read its value.
The source loop retains its existing inherited-property check.
The correction applies to `index.js`, `dist/cjs.js`, and `dist/umd.js`.
[Upstream PR #274](https://github.com/TehShrike/deepmerge/pull/274) contains the proposed correction.
That pull request is closed. This package contains a local correction, not an upstream release.
`PROTOTYPE-CORRECTION.patch` records the local source changes.

`make mobile-deepmerge-security-check` verifies every installed copy through its CJS and UMD entries.
It verifies unsafe keys, null targets, nested values, array clones, ordinary merges, symbols, and custom merge behavior.
The checks also exercise the public Expo config and React Native config APIs.
`clone:false` references and arbitrary callback results retain their caller-defined behavior.
The source `index.js` requires the upstream build dependency `is-mergeable-object`.
Runtime qualification uses the shipped CJS and UMD files.

`make mobile-dependency-audit-check` verifies the correction before it accepts the exact upstream advisory GHSA-gcqx-w48m-w2h4.
The adapter rejects unknown advisories, unqualified package copies, invalid reports, and npm failures.
Other repository audits retain their existing commands.
A registry report can still contain the upstream advisory after this local correction.
`make mobile-deepmerge-prepared-check` verifies the selected prepared dependency tree after canonical preparation.

`make mobile-deepmerge-installation-check` verifies both native npm installation modes in isolated public fixtures.
The fixtures include the tracked `.npmrc` policy.
They do not replace canonical native preparation.
Review the upstream package and advisory before the review date in `mobile/config/dependency-audit.json`.
