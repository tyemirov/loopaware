# LoopAware Braces correction

This repository-owned package uses the runtime sources from `braces@3.0.3`.
The registry archive matches the original npm lock integrity.
The upstream MIT license remains unchanged in `LICENSE`.

The depth correction follows [upstream PR #72](https://github.com/micromatch/braces/pull/72), at commit `d0d575e55e74a4e0218e5248fafb79efc3e54ebb`.
The parser limits combined brace and parenthesis nesting to 100 levels.
The compile, expand, and stringify walkers also limit caller-supplied AST depth.
The `maxDepth` option can reduce this limit. It cannot increase or disable the limit.
Two additional local checks bound the AST parent traversal used by expansion.
The stringify depth correction preserves the original parent context and `escapeInvalid` behavior.
The original defect is [GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm), also tracked in [upstream issue #70](https://github.com/micromatch/braces/issues/70).

`make mobile-braces-security-check` tests the public Braces APIs through the Metro and React Native CLI dependency paths.
It also verifies Micromatch and Fast-glob pattern operations.
The security audit and native preparation checks require this gate in both dependency trees.

The local package name is `@loopaware/braces`, with version `3.0.3-loopaware.1`.
The npm audit does not match upstream Braces advisories to this local package name.
The behavioral gate proves this correction. It does not replace review of future upstream advisories.
Before an update, review the upstream changes and repeat the public verification tests.
