#!/bin/sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "error: usage: run_lifecycle.sh <command> [args...]" >&2
  exit 2
fi

for variable_name in BASH_ENV ENV NODE_OPTIONS NODE_PATH PYTHONHOME PYTHONPATH DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH; do
  eval "variable_value=\${${variable_name}-}"
  if [ -n "${variable_value}" ]; then
    echo "error: ${variable_name} is not supported by the canonical lifecycle" >&2
    exit 1
  fi
done

blocked_shell_environment="$(
  /usr/bin/env | /usr/bin/awk -F= '$1 ~ /^BASH_FUNC_/ || $1 == "SHELLOPTS" || $1 == "BASHOPTS" { print $1; exit }'
)"
if [ -n "${blocked_shell_environment}" ]; then
  echo "error: ${blocked_shell_environment} is not supported by the canonical lifecycle" >&2
  exit 1
fi

bash_path="$(command -v bash || true)"
case "${bash_path}" in
  /bin/bash|/usr/bin/bash|/usr/local/bin/bash|/opt/homebrew/bin/bash) ;;
  *)
    echo "error: lifecycle requires Bash from a canonical system or Homebrew path, got ${bash_path:-<missing>}" >&2
    exit 1
    ;;
esac
if ! "${bash_path}" -c 'test "${BASH_VERSINFO[0]}" -ge 4'; then
  echo "error: lifecycle requires Bash 4 or newer, got ${bash_path}" >&2
  exit 1
fi

unset BASH_ENV ENV NODE_OPTIONS NODE_PATH PYTHONHOME PYTHONPATH DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH CDPATH
exec "${bash_path}" "$@"
