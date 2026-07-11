#!/usr/bin/env bash

load_release_env_file() {
  local env_file="$1"
  local loader_directory
  local export_file
  loader_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  export_file="$(mktemp)"
  if ! "${loader_directory}/parse_release_env.py" "${env_file}" >"${export_file}"; then
    rm -f "${export_file}"
    return 1
  fi
  # The parser emits only validated names and shlex-quoted literal values.
  # shellcheck disable=SC1090
  source "${export_file}"
  rm -f "${export_file}"
}
