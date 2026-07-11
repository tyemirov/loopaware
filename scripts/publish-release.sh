#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
source "${repo_root}/scripts/release/repository_identity.sh"
assert_no_github_repository_override
assert_canonical_github_origin "${repo_root}" LoopAware "tyemirov/loopaware"
pipeline="${repo_root}/scripts/release/publish_release.sh"
[[ -x "${pipeline}" ]] || {
  echo "error: repository-owned prepared-release publish pipeline not found: ${pipeline}" >&2
  exit 1
}

exec "${pipeline}" "$@"
