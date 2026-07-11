#!/usr/bin/env bash
set -euo pipefail

[[ $# -ge 2 ]] || { echo "error: usage: with_lifecycle_lock.sh <operation> <command> [args...]" >&2; exit 2; }
operation="$1"
shift
[[ "${operation}" =~ ^[a-z][a-z-]*$ ]] || { echo "error: invalid lifecycle lock operation: ${operation}" >&2; exit 2; }

repo_root="$(git rev-parse --show-toplevel)"
git_common_dir="$(git -C "${repo_root}" rev-parse --git-common-dir)"
if [[ "${git_common_dir}" != /* ]]; then
  git_common_dir="${repo_root}/${git_common_dir}"
fi
lock_dir="${git_common_dir}/mprlab-lifecycle.lock"
if ! mkdir "${lock_dir}" 2>/dev/null; then
  if [[ -d "${lock_dir}" ]]; then
    echo "error: lifecycle operation is already locked: ${lock_dir}" >&2
    echo "error: if no release, publish, or deploy process is active, remove that stale directory explicitly and retry" >&2
  else
    echo "error: cannot create lifecycle lock directory: ${lock_dir}" >&2
    echo "error: the Git common directory must be writable by the release operator" >&2
  fi
  exit 1
fi
printf '%s\n' "operation=${operation}" "pid=$$" >"${lock_dir}/owner"

cleanup() {
  rm -f "${lock_dir}/owner"
  rmdir "${lock_dir}"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

"$@"
