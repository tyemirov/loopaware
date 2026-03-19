#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
test_config_dir="${repo_root}/tests/configs"
compose_file="${repo_root}/tests/docker-compose.yml"
test_web_root=$(mktemp -d "${repo_root}/tests/.runtime-web.XXXXXX")

cp -R "${repo_root}/web/." "${test_web_root}/"
cp "${repo_root}/configs/config.frontend.yml" "${test_web_root}/config.yml"
chmod -R a+rX "${test_web_root}"

export LOOPAWARE_BASE_URL=${LOOPAWARE_BASE_URL:-http://localhost:8090}
export LOOPAWARE_ENV_FILE=${LOOPAWARE_ENV_FILE:-${test_config_dir}/loopaware.env}
export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-loopaware-integration-$(date +%s)}
export LOOPAWARE_TEST_WEB_ROOT="${test_web_root}"

cleanup() {
  docker compose -f "${compose_file}" down -v --remove-orphans
  rm -rf "${test_web_root}"
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
