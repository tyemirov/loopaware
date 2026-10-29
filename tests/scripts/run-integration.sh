#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
test_config_dir="${repo_root}/tests/configs"
compose_file="${repo_root}/tests/docker-compose.yml"
cleanup_guardian_pid=""
cleanup_diagnostic_file=""
cleanup_readiness_dir=""
cleanup_complete=0

export LOOPAWARE_TEST_NETWORK_PREFIX="${LOOPAWARE_TEST_NETWORK_PREFIX-10.203.242}"
network_octet='(0|[1-9][0-9]?|1[0-9]{2}|2[0-4][0-9]|25[0-5])'
network_prefix_pattern="^10\.${network_octet}\.${network_octet}$"
if [[ ! "${LOOPAWARE_TEST_NETWORK_PREFIX}" =~ ${network_prefix_pattern} ]]; then
  echo "LOOPAWARE_TEST_NETWORK_PREFIX must use 10.<octet>.<octet> with decimal octets from 0 through 255." >&2
  exit 2
fi

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
  local status=0
  if docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null; then
    return 0
  else
    status=$?
    echo "Integration stack cleanup failed (status ${status})" >&2
    return "${status}"
  fi
}

close_guardian_readiness() {
  local cleanup_status=0
  local status=0
  exec 7>&- 8<&- 9>&-
  if [[ -n "${cleanup_readiness_dir}" && -d "${cleanup_readiness_dir}" ]]; then
    if rm "${cleanup_readiness_dir}/ready"; then :; else
      cleanup_status=$?
      echo "Integration guardian readiness FIFO cleanup failed (status ${cleanup_status})" >&2
    fi
    if rmdir "${cleanup_readiness_dir}"; then :; else
      status=$?
      echo "Integration guardian readiness directory cleanup failed (status ${status})" >&2
      if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
    fi
  fi
  return "${cleanup_status}"
}

cleanup() {
  local original_status="$1"
  local cleanup_status=0
  local status=0
  if [[ "${cleanup_complete}" -eq 1 ]]; then
    return "${original_status}"
  fi

  cleanup_complete=1
  trap - EXIT INT TERM HUP QUIT

  if [[ -n "${cleanup_guardian_pid}" ]]; then
    if kill -0 "${cleanup_guardian_pid}" >/dev/null 2>&1; then
      kill "${cleanup_guardian_pid}" >/dev/null 2>&1 || true
    fi
    if wait "${cleanup_guardian_pid}"; then
      :
    else
      cleanup_status=$?
      cat "${cleanup_diagnostic_file}" >&2
      echo "Integration cleanup guardian failed (status ${cleanup_status})" >&2
    fi
  fi
  if close_guardian_readiness; then :; else
    status=$?
    if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
  fi

  if down_stack; then
    :
  else
    status=$?
    if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
  fi
  if [[ -d "${integration_lock_dir}" ]]; then
    if rmdir "${integration_lock_dir}"; then
      :
    else
      status=$?
      echo "Integration lock cleanup failed (status ${status})" >&2
      if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
    fi
  fi
  if [[ -n "${cleanup_diagnostic_file}" ]]; then
    if [[ "${cleanup_status}" -eq 0 ]]; then
      if rm "${cleanup_diagnostic_file}"; then :; else cleanup_status=$?; fi
    else
      echo "Integration cleanup diagnostics retained: ${cleanup_diagnostic_file}" >&2
    fi
  fi
  if [[ "${original_status}" -ne 0 ]]; then return "${original_status}"; fi
  return "${cleanup_status}"
}

on_signal() {
  local exit_code="$1"
  cleanup "${exit_code}" || exit "${exit_code}"
  exit "${exit_code}"
}

start_cleanup_guardian() {
  local parent_pid="$$"
  local parent_start_time=""

  parent_start_time="$(get_process_start_time "${parent_pid}")"
  cleanup_diagnostic_file="$(mktemp "${repo_root}/.cache/loopaware-integration-cleanup.XXXXXX")"
  cleanup_readiness_dir="$(mktemp -d "${repo_root}/.cache/loopaware-integration-ready.XXXXXX")"
  mkfifo "${cleanup_readiness_dir}/ready"
  exec 7<>"${cleanup_readiness_dir}/ready"
  exec 8<"${cleanup_readiness_dir}/ready"
  exec 9>"${cleanup_readiness_dir}/ready"

  nohup bash -c '
    exec 7>&- 8<&-
    parent_pid="$1"
    parent_start_time="$2"
    compose_file="$3"
    compose_project_name="$4"
    integration_lock_dir="$5"
    cleanup_readiness_dir="$6"

    get_process_start_time() {
      ps -o lstart= -p "$1" 2>/dev/null | sed "s/^[[:space:]]*//"
    }

    cleanup_topology() {
      local cleanup_status=0
      local status=0
      trap - EXIT INT TERM HUP QUIT
      exec 9>&-
      if [[ -d "${cleanup_readiness_dir}" ]]; then
        if rm "${cleanup_readiness_dir}/ready"; then :; else
          cleanup_status=$?
          echo "Integration guardian readiness FIFO cleanup failed (status ${cleanup_status})" >&2
        fi
        if rmdir "${cleanup_readiness_dir}"; then :; else
          status=$?
          echo "Integration guardian readiness directory cleanup failed (status ${status})" >&2
          if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
        fi
      fi
      if docker compose -f "${compose_file}" -p "${compose_project_name}" down -v --remove-orphans >/dev/null; then
        :
      else
        status=$?
        echo "Integration stack cleanup failed (status ${status})" >&2
        if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
      fi
      if [[ -d "${integration_lock_dir}" ]]; then
        if rmdir "${integration_lock_dir}"; then
          :
        else
          status=$?
          echo "Integration lock cleanup failed (status ${status})" >&2
          if [[ "${cleanup_status}" -eq 0 ]]; then cleanup_status="${status}"; fi
        fi
      fi
      exit "${cleanup_status}"
    }
    trap cleanup_topology EXIT INT TERM HUP QUIT
    printf "%s\n" READY >&9
    exec 9>&-

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
  ' bash "${parent_pid}" "${parent_start_time}" "${compose_file}" "${compose_project_name}" "${integration_lock_dir}" "${cleanup_readiness_dir}" >"${cleanup_diagnostic_file}" 2>&1 </dev/null &

  cleanup_guardian_pid=$!
  exec 7>&- 9>&-
  local readiness=""
  if IFS= read -r readiness <&8 && [[ "${readiness}" == READY ]]; then
    close_guardian_readiness
  else
    echo "Integration cleanup guardian exited before readiness." >&2
    local startup_status=1
    if wait "${cleanup_guardian_pid}"; then :; else startup_status=$?; fi
    if close_guardian_readiness; then :; else :; fi
    return "${startup_status}"
  fi
}
trap 'cleanup "$?"' EXIT
trap 'on_signal 130' INT
trap 'on_signal 143' TERM
trap 'on_signal 129' HUP
trap 'on_signal 131' QUIT

start_cleanup_guardian

down_stack

if ! docker compose -f "${compose_file}" -p "${compose_project_name}" up --build --wait --wait-timeout 60; then
  echo "Integration stack startup failed; service logs follow." >&2
  docker compose -f "${compose_file}" -p "${compose_project_name}" logs --no-color --tail 100 >&2
  exit 1
fi

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

npm --prefix "${repo_root}/tests" ci
if [[ -n "${LOOPAWARE_PLAYWRIGHT_CHANNEL:-}" ]]; then
  echo "Using Playwright browser channel ${LOOPAWARE_PLAYWRIGHT_CHANNEL}; skipping bundled browser install."
elif ! (cd "${repo_root}/tests" && node --input-type=module -e "import { chromium } from '@playwright/test'; import fs from 'fs'; const path = chromium.executablePath(); if (!fs.existsSync(path)) process.exit(1);"); then
  npm --prefix "${repo_root}/tests" exec -- playwright install chromium
fi
integration_suite=${LOOPAWARE_TEST_SUITE:-test:all}
suite_arguments=()
if [[ -n "${LOOPAWARE_TEST_SHARD:-}" ]]; then
  suite_arguments=(-- "--shard=${LOOPAWARE_TEST_SHARD}")
fi
env -u NO_COLOR npm --prefix "${repo_root}/tests" run "${integration_suite}" "${suite_arguments[@]}"
