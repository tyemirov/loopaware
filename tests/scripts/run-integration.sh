#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
config_dir="${repo_root}/configs"
compose_file="${repo_root}/docker-compose.integration.yml"

ensure_env_file() {
  local target="$1"
  local example="$2"
  if [[ -f "${target}" ]]; then
    return
  fi
  if [[ ! -f "${example}" ]]; then
    echo "Missing env template: ${example}" >&2
    exit 1
  fi
  cp "${example}" "${target}"
}

ensure_env_file "${config_dir}/.env.loopaware.integration" "${config_dir}/.env.loopaware.integration.example"
ensure_env_file "${config_dir}/.env.tauth.integration" "${config_dir}/.env.tauth.integration.example"
ensure_env_file "${config_dir}/.env.pinguin.integration" "${config_dir}/.env.pinguin.integration.example"
ensure_env_file "${config_dir}/.env.ghttp.integration" "${config_dir}/.env.ghttp.integration.example"

export LOOPAWARE_BASE_URL=${LOOPAWARE_BASE_URL:-http://localhost:8090}
export LOOPAWARE_ENV_FILE=${LOOPAWARE_ENV_FILE:-${config_dir}/.env.loopaware.integration}
export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-loopaware-integration-$(date +%s)}

cleanup() {
  docker compose -f "${compose_file}" down -v --remove-orphans
}
trap cleanup EXIT

docker compose -f "${compose_file}" down -v --remove-orphans

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
