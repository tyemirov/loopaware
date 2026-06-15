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
compose_project_name="${COMPOSE_PROJECT_NAME}"

get_process_start_time() {
  ps -o lstart= -p "$1" 2>/dev/null | sed 's/^[[:space:]]*//'
}

down_stack() {
  docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null 2>&1 || true
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
  local parent_start_time=""

  parent_start_time="$(get_process_start_time "${parent_pid}")"

  if command -v setsid >/dev/null 2>&1; then
    setsid nohup bash -c '
      parent_pid="$1"
      parent_start_time="$2"
      compose_file="$3"
      compose_project_name="$4"

      get_process_start_time() {
        ps -o lstart= -p "$1" 2>/dev/null | sed "s/^[[:space:]]*//"
      }

      while true; do
        if [[ -n "${parent_start_time}" ]]; then
          current_parent_start_time="$(get_process_start_time "${parent_pid}")"
          if [[ -z "${current_parent_start_time}" || "${current_parent_start_time}" != "${parent_start_time}" ]]; then
            break
          fi
        elif ! kill -0 "${parent_pid}" >/dev/null 2>&1; then
          break
        fi
        sleep 1
      done

      docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null 2>&1 || true
    ' bash "${parent_pid}" "${parent_start_time}" "${compose_file}" "${compose_project_name}" >/dev/null 2>&1 </dev/null &
  else
    nohup bash -c '
      parent_pid="$1"
      parent_start_time="$2"
      compose_file="$3"
      compose_project_name="$4"

      get_process_start_time() {
        ps -o lstart= -p "$1" 2>/dev/null | sed "s/^[[:space:]]*//"
      }

      while true; do
        if [[ -n "${parent_start_time}" ]]; then
          current_parent_start_time="$(get_process_start_time "${parent_pid}")"
          if [[ -z "${current_parent_start_time}" || "${current_parent_start_time}" != "${parent_start_time}" ]]; then
            break
          fi
        elif ! kill -0 "${parent_pid}" >/dev/null 2>&1; then
          break
        fi
        sleep 1
      done

      docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null 2>&1 || true
    ' bash "${parent_pid}" "${parent_start_time}" "${compose_file}" "${compose_project_name}" >/dev/null 2>&1 </dev/null &
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

docker compose -f "${compose_file}" -p "${compose_project_name}" up --build -d

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
if [[ -n "${LOOPAWARE_PLAYWRIGHT_CHANNEL:-}" ]]; then
  echo "Using Playwright browser channel ${LOOPAWARE_PLAYWRIGHT_CHANNEL}; skipping bundled browser install."
elif ! (cd "${repo_root}/tests" && node --input-type=module -e "import { chromium } from '@playwright/test'; import fs from 'fs'; const path = chromium.executablePath(); if (!fs.existsSync(path)) process.exit(1);"); then
  npm --prefix "${repo_root}/tests" exec -- playwright install chromium
fi
integration_suite=${LOOPAWARE_TEST_SUITE:-test:all}
env -u NO_COLOR npm --prefix "${repo_root}/tests" run "${integration_suite}"
