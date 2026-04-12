#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
test_config_dir="${repo_root}/tests/configs"
compose_file="${repo_root}/tests/docker-compose.yml"
cleanup_guardian_pid=""
cleanup_complete=0

export LOOPAWARE_BASE_URL=${LOOPAWARE_BASE_URL:-http://localhost:8090}
export LOOPAWARE_ENV_FILE=${LOOPAWARE_ENV_FILE:-${test_config_dir}/loopaware.env}
export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-loopaware-integration-$(date +%s)}

down_stack() {
  docker compose -f "${compose_file}" down -v --remove-orphans >/dev/null 2>&1 || true
}

cleanup() {
  if [[ "${cleanup_complete}" -eq 1 ]]; then
    return 0
  fi

  cleanup_complete=1
  trap - EXIT INT TERM HUP QUIT

  down_stack

  if [[ -n "${cleanup_guardian_pid}" ]]; then
    kill "${cleanup_guardian_pid}" >/dev/null 2>&1 || true
    wait "${cleanup_guardian_pid}" >/dev/null 2>&1 || true
  fi
}

on_signal() {
  local exit_code="$1"
  cleanup
  exit "${exit_code}"
}

start_cleanup_guardian() {
  local parent_pid="$$"

  if command -v setsid >/dev/null 2>&1; then
    setsid nohup bash -c '
      parent_pid="$1"
      compose_file="$2"

      while kill -0 "${parent_pid}" >/dev/null 2>&1; do
        sleep 1
      done

      docker compose -f "${compose_file}" down -v --remove-orphans >/dev/null 2>&1 || true
    ' bash "${parent_pid}" "${compose_file}" >/dev/null 2>&1 </dev/null &
  else
    nohup bash -c '
      parent_pid="$1"
      compose_file="$2"

      while kill -0 "${parent_pid}" >/dev/null 2>&1; do
        sleep 1
      done

      docker compose -f "${compose_file}" down -v --remove-orphans >/dev/null 2>&1 || true
    ' bash "${parent_pid}" "${compose_file}" >/dev/null 2>&1 </dev/null &
  fi

  cleanup_guardian_pid=$!
}
trap cleanup EXIT
trap 'on_signal 130' INT
trap 'on_signal 143' TERM
trap 'on_signal 129' HUP
trap 'on_signal 131' QUIT

start_cleanup_guardian

down_stack

docker compose -f "${compose_file}" up --build -d

ready=false
for _ in $(seq 1 60); do
  if curl -fsS "${LOOPAWARE_BASE_URL}/login" >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
  done

if [[ "${ready}" != "true" ]]; then
  echo "Integration stack did not become ready at ${LOOPAWARE_BASE_URL}" >&2
  exit 1
fi

npm --prefix "${repo_root}/tests" install
if ! (cd "${repo_root}/tests" && node --input-type=module -e "import { chromium } from '@playwright/test'; import fs from 'fs'; const path = chromium.executablePath(); if (!fs.existsSync(path)) process.exit(1);"); then
  npm --prefix "${repo_root}/tests" exec -- playwright install
fi
integration_suite=${LOOPAWARE_TEST_SUITE:-test:all}
npm --prefix "${repo_root}/tests" run "${integration_suite}"
