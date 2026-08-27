#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
test_config_dir="${repo_root}/tests/configs"
compose_file="${repo_root}/tests/docker-compose.yml"
cleanup_guardian_pid=""
cleanup_complete=0

for variable_name in LOOPAWARE_BASE_URL LOOPAWARE_API_BASE_URL LOOPAWARE_ENV_FILE COMPOSE_PROJECT_NAME DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH; do
  if [[ -n "${!variable_name:-}" ]]; then
    echo "Integration runner rejects inherited ${variable_name}; canonical local test isolation is mandatory." >&2
    exit 1
  fi
done
docker_context="$(docker context show)"
docker_endpoint="$(docker context inspect "${docker_context}" --format '{{.Endpoints.docker.Host}}')"
[[ "${docker_endpoint}" == unix://* || "${docker_endpoint}" == npipe://* ]] || {
  echo "Integration runner requires a local Docker context, got ${docker_context}: ${docker_endpoint}" >&2
  exit 1
}

mkdir -p "${repo_root}/.cache"
integration_lock_dir="${repo_root}/.cache/loopaware-integration.lock"
if ! mkdir "${integration_lock_dir}" 2>/dev/null; then
  echo "Integration test topology is already owned by another process." >&2
  exit 75
fi

export LOOPAWARE_BASE_URL=http://localhost:8090
export LOOPAWARE_ENV_FILE=${test_config_dir}/loopaware.env
export COMPOSE_PROJECT_NAME=loopaware-integration
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

  if [[ -n "${cleanup_guardian_pid}" ]] && kill -0 "${cleanup_guardian_pid}" >/dev/null 2>&1; then
    kill "${cleanup_guardian_pid}" >/dev/null 2>&1 || true
    wait "${cleanup_guardian_pid}" >/dev/null 2>&1 || true
  fi

  down_stack
  rmdir "${integration_lock_dir}" >/dev/null 2>&1 || true
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

  nohup bash -c '
    parent_pid="$1"
    parent_start_time="$2"
    compose_file="$3"
    compose_project_name="$4"
    integration_lock_dir="$5"

    get_process_start_time() {
      ps -o lstart= -p "$1" 2>/dev/null | sed "s/^[[:space:]]*//"
    }

    cleanup_topology() {
      trap - EXIT INT TERM HUP QUIT
      docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null 2>&1 || true
      rmdir "${integration_lock_dir}" >/dev/null 2>&1 || true
      exit 0
    }
    trap cleanup_topology EXIT INT TERM HUP QUIT

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

    cleanup_topology
  ' bash "${parent_pid}" "${parent_start_time}" "${compose_file}" "${compose_project_name}" "${integration_lock_dir}" >/dev/null 2>&1 </dev/null &

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

api_host_address="$(docker compose -f "${compose_file}" -p "${compose_project_name}" port loopaware-api 8080)"
[[ "${api_host_address}" == 127.0.0.1:* ]] || {
  echo "Integration backend did not publish an isolated loopback port: ${api_host_address}" >&2
  exit 1
}
export LOOPAWARE_API_BASE_URL="http://${api_host_address}"

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
