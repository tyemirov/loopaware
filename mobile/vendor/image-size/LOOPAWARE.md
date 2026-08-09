# LoopAware image-size fork

This package is a repository-owned fork of `image-size` 1.2.1. It preserves
the CommonJS callable API required by Metro 0.84.4 and applies the current
denial-of-service corrections before an upstream patched release exists.

The fork rejects ICNS entries shorter than their eight-byte header and rejects
ISO base media boxes whose declared size cannot advance the parser. These
boundaries address CVE-2025-71330 and CVE-2025-71329. The upstream MIT license
is preserved in `LICENSE`.
