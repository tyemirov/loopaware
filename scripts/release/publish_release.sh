#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper is not executable: ${helper}" >&2; exit 1; }

exec "${helper}" publish-prepared-release "$@"
