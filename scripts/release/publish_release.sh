#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
export UV_CACHE_DIR="${repo_root}/.cache/uv"
helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper is not executable: ${helper}" >&2; exit 1; }

exec "${helper}" publish-prepared-release "$@"
