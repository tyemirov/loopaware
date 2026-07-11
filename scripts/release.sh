#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"

if [[ -v RELEASE_ENV_FILE ]] && [[ -n "${RELEASE_ENV_FILE}" ]]; then
  env_file="${RELEASE_ENV_FILE}"
else
  env_file="${repo_root}/configs/.env.loopaware"
fi
if [[ "${env_file}" != /* ]]; then
  env_file="${repo_root}/${env_file}"
fi
[[ -f "${env_file}" ]] || { echo "error: mobile release env file not found: ${env_file}" >&2; exit 1; }
set -a
# shellcheck disable=SC1090
source "${env_file}"
set +a

pipeline="${repo_root}/scripts/release/prepare_release.sh"
[[ -x "${pipeline}" ]] || {
  echo "error: repository-owned release pipeline not found: ${pipeline}" >&2
  exit 1
}

exec "${pipeline}" "$@"
