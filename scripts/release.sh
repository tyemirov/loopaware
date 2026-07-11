#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
source "${repo_root}/scripts/release/repository_identity.sh"
assert_no_github_repository_override
assert_canonical_github_origin "${repo_root}" LoopAware "tyemirov/loopaware"
assert_remote_default_and_release_tags "${repo_root}" LoopAware allow-prepared-release

if [[ -v RELEASE_ENV_FILE ]] && [[ -n "${RELEASE_ENV_FILE}" ]]; then
  env_file="${RELEASE_ENV_FILE}"
else
  env_file="${repo_root}/configs/.env.loopaware"
fi
if [[ "${env_file}" != /* ]]; then
  env_file="${repo_root}/${env_file}"
fi
[[ -f "${env_file}" ]] || { echo "error: mobile release env file not found: ${env_file}" >&2; exit 1; }
source "${repo_root}/scripts/release/load_release_env.sh"
load_release_env_file "${env_file}"

pipeline="${repo_root}/scripts/release/prepare_release.sh"
[[ -x "${pipeline}" ]] || {
  echo "error: repository-owned release pipeline not found: ${pipeline}" >&2
  exit 1
}

exec "${pipeline}" "$@"
