# LoopAware Forge correction

This repository-owned package uses the Node runtime sources from `node-forge@1.4.0`.
The published upstream package came from the npm registry.
The upstream BSD-3-Clause and GPL-2.0 license text remains in `LICENSE`.
This package excludes the upstream browser bundles, which contain the uncorrected code.

The RSA verifier rejects unconsumed elements in the nested DigestAlgorithm sequence.
It also rejects NULL parameters that contain bytes.
The element-count correction follows [upstream PR #1152](https://github.com/digitalbazaar/forge/pull/1152).
The NULL-content check is an additional local correction.
The original defect is [GHSA-86w9-cpqp-85rv](https://github.com/advisories/GHSA-86w9-cpqp-85rv), also tracked in [upstream issue #1149](https://github.com/digitalbazaar/forge/issues/1149).

`make mobile-forge-security-check` tests the public verifier from both Expo consumers.
It verifies valid signatures, malformed signatures, certificate operations, and the installed package identity.
`make security-audit` requires this check and retains the npm audit threshold.
The native preparation checks test the same correction in the prepared dependency tree.

The local package name is `@loopaware/node-forge`, with version `1.4.0-loopaware.1`.
The npm audit does not match upstream Forge advisories to this local package name.
The behavioral check proves this correction. It does not replace review of future upstream advisories.
Before an update, review the upstream Forge changes and repeat the public verification tests.
