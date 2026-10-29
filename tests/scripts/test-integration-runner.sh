#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
runtime_dir=$(mktemp -d)
fake_bin="${runtime_dir}/bin"
docker_log="${runtime_dir}/docker.log"
mkdir -p "${fake_bin}" "${repo_root}/.cache"

cleanup() {
  rm -rf "${runtime_dir}"
}
trap cleanup EXIT

cat > "${fake_bin}/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$1 $2" == "context show" ]]; then
  echo default
  exit 0
fi
if [[ "$1 $2" == "context inspect" ]]; then
  echo unix:///tmp/docker.sock
  exit 0
fi

echo "$*" >> "${LOOPAWARE_FAKE_DOCKER_LOG}"
if [[ "$*" == *" down "* && -n "${LOOPAWARE_FAKE_SUITE_MARKER:-}" && -f "${LOOPAWARE_FAKE_SUITE_MARKER}" ]]; then
  count=0
  if [[ -f "${LOOPAWARE_FAKE_DOWN_COUNTER}" ]]; then read -r count < "${LOOPAWARE_FAKE_DOWN_COUNTER}"; fi
  count=$((count + 1))
  printf '%s\n' "${count}" > "${LOOPAWARE_FAKE_DOWN_COUNTER}"
  if [[ "${count}" -eq "${LOOPAWARE_FAKE_DOWN_TARGET}" ]]; then
    echo "controlled-cleanup-failure" >&2
    exit 29
  fi
fi
if [[ "$*" == *" up "* && "${LOOPAWARE_FAKE_STARTUP_FAILURE:-0}" == 1 ]]; then
  echo "API health check failed" >&2
  exit 1
fi
if [[ "$*" == *" port loopaware-api 8080" ]]; then
  echo 127.0.0.1:55000
fi
DOCKER
chmod +x "${fake_bin}/docker"

cat > "${fake_bin}/curl" <<'CURL'
#!/usr/bin/env bash
exit 0
CURL
chmod +x "${fake_bin}/curl"

cat > "${fake_bin}/npm" <<'NPM'
#!/usr/bin/env bash
echo "npm $*" >> "${LOOPAWARE_FAKE_DOCKER_LOG}"
if [[ "$*" == *" run "* && -n "${LOOPAWARE_FAKE_SUITE_MARKER:-}" ]]; then
  printf '%s\n' entered > "${LOOPAWARE_FAKE_SUITE_MARKER}"
  if [[ "${LOOPAWARE_FAKE_SUITE_STATUS}" -ne 0 ]]; then echo controlled-suite-failure >&2; fi
  exit "${LOOPAWARE_FAKE_SUITE_STATUS}"
fi
exit 0
NPM
chmod +x "${fake_bin}/npm"

lock_dir="${repo_root}/.cache/loopaware-integration.lock"
mkdir "${lock_dir}"

set +e
runner_output="$(PATH="${fake_bin}:${PATH}" "${script_dir}/run-integration.sh" 2>&1)"
runner_status=$?
set -e

if [[ "${runner_status}" -ne 75 ]]; then
  echo "Expected concurrent integration run status 75, got ${runner_status}." >&2
  echo "${runner_output}" >&2
  exit 1
fi

expected_message="Integration test topology is already owned by another process."
if [[ "${runner_output}" != *"${expected_message}"* ]]; then
  echo "Expected concurrent integration ownership error." >&2
  echo "${runner_output}" >&2
  exit 1
fi

rmdir "${lock_dir}"
LOOPAWARE_FAKE_DOCKER_LOG="${docker_log}" \
  LOOPAWARE_TEST_SHARD=2/3 \
  LOOPAWARE_PLAYWRIGHT_CHANNEL=chrome \
  PATH="${fake_bin}:${PATH}" \
  "${script_dir}/run-integration.sh"
if [[ -e "${lock_dir}" ]]; then
  echo "First sequential integration run kept the topology lock." >&2
  exit 1
fi

LOOPAWARE_FAKE_DOCKER_LOG="${docker_log}" \
  LOOPAWARE_PLAYWRIGHT_CHANNEL=chrome \
  PATH="${fake_bin}:${PATH}" \
  "${script_dir}/run-integration.sh"
if [[ -e "${lock_dir}" ]]; then
  echo "Second sequential integration run kept the topology lock." >&2
  exit 1
fi

if ! grep -F -- "-p loopaware-integration up --build --wait --wait-timeout 60" "${docker_log}" >/dev/null; then
  echo "Expected the integration runner to wait for API health." >&2
  cat "${docker_log}" >&2
  exit 1
fi
if ! grep -F -- "run test:all -- --shard=2/3" "${docker_log}" >/dev/null; then
  echo "Expected the integration runner to pass the selected shard to Playwright." >&2
  exit 1
fi

: > "${docker_log}"
set +e
runner_output="$(LOOPAWARE_FAKE_DOCKER_LOG="${docker_log}" \
  LOOPAWARE_FAKE_STARTUP_FAILURE=1 \
  LOOPAWARE_PLAYWRIGHT_CHANNEL=chrome \
  PATH="${fake_bin}:${PATH}" \
  "${script_dir}/run-integration.sh" 2>&1)"
runner_status=$?
set -e
if [[ "${runner_status}" -eq 0 || "${runner_output}" != *"Integration stack startup failed"* ]]; then
  echo "Expected the integration runner to report API startup failure." >&2
  echo "${runner_output}" >&2
  exit 1
fi
if ! grep -F -- "-p loopaware-integration logs --no-color --tail 100" "${docker_log}" >/dev/null; then
  echo "Expected service logs after integration startup failure." >&2
  exit 1
fi
if [[ -e "${lock_dir}" ]]; then
  echo "Failed integration startup kept the topology lock." >&2
  exit 1
fi
if grep -F -- "npm " "${docker_log}" >/dev/null; then
  echo "Failed integration startup must not start the test suite." >&2
  exit 1
fi

for down_target in 1 2; do
  for suite_status in 0 23; do
    suite_marker="${runtime_dir}/suite-${down_target}-${suite_status}"
    down_counter="${runtime_dir}/down-${down_target}-${suite_status}"
    set +e
    runner_output="$(LOOPAWARE_FAKE_DOCKER_LOG="${docker_log}" \
      LOOPAWARE_FAKE_SUITE_MARKER="${suite_marker}" \
      LOOPAWARE_FAKE_SUITE_STATUS="${suite_status}" \
      LOOPAWARE_FAKE_DOWN_COUNTER="${down_counter}" \
      LOOPAWARE_FAKE_DOWN_TARGET="${down_target}" \
      LOOPAWARE_PLAYWRIGHT_CHANNEL=chrome \
      PATH="${fake_bin}:${PATH}" "${script_dir}/run-integration.sh" 2>&1)"
    runner_status=$?
    set -e
    expected_status=29
    if [[ "${suite_status}" -ne 0 ]]; then expected_status="${suite_status}"; fi
    if [[ "${runner_status}" -ne "${expected_status}" || "${runner_output}" != *"Integration stack cleanup failed (status 29)"* || "${runner_output}" != *"controlled-cleanup-failure"* ]]; then
      echo "Cleanup path ${down_target} lost status or diagnostics (${runner_status}, expected ${expected_status})." >&2
      echo "${runner_output}" >&2
      exit 1
    fi
    if [[ "${down_target}" -eq 1 && "${runner_output}" != *"Integration cleanup guardian failed (status 29)"* ]]; then
      echo "Guardian failure was lost after main cleanup." >&2
      exit 1
    fi
    if [[ "${suite_status}" -ne 0 && "${runner_output}" != *"controlled-suite-failure"* ]]; then
      echo "Original suite failure was lost during cleanup." >&2
      exit 1
    fi
    if [[ -e "${lock_dir}" ]]; then
      echo "Cleanup failure kept the topology lock." >&2
      exit 1
    fi
    diagnostic_path="$(printf '%s\n' "${runner_output}" | sed -n 's/^Integration cleanup diagnostics retained: //p')"
    [[ "${diagnostic_path}" == "${repo_root}/.cache/loopaware-integration-cleanup."* && -f "${diagnostic_path}" ]] || {
      echo "Expected the exact owned cleanup diagnostic file." >&2
      exit 1
    }
    rm "${diagnostic_path}"
  done
done

network_config="${runtime_dir}/compose.json"
LOOPAWARE_TEST_NETWORK_PREFIX=10.212.34 docker compose -f "${repo_root}/tests/docker-compose.yml" config --format json > "${network_config}"
python3 - "${network_config}" <<'PYTHON'
import json
import sys
with open(sys.argv[1]) as config_file:
    config = json.load(config_file)
assert config["networks"]["default"]["ipam"]["config"][0]["subnet"] == "10.212.34.0/24"
assert config["services"]["loopaware-proxy"]["networks"]["default"]["ipv4_address"] == "10.212.34.10"
assert config["services"]["loopaware-api"]["environment"]["TRUSTED_PROXY_CIDR"] == "10.212.34.10/32"
PYTHON
for invalid_prefix in '' 172.30.242 10.256.2 10.2.3.4 10.02.3 garbage; do
  set +e
  runner_output="$(LOOPAWARE_FAKE_DOCKER_LOG="${docker_log}" \
    LOOPAWARE_TEST_NETWORK_PREFIX="${invalid_prefix}" \
    PATH="${fake_bin}:${PATH}" "${script_dir}/run-integration.sh" 2>&1)"
  runner_status=$?
  set -e
  if [[ "${runner_status}" -eq 0 || "${runner_output}" != *"LOOPAWARE_TEST_NETWORK_PREFIX must use 10.<octet>.<octet>"* ]]; then
    echo "Expected invalid network prefix rejection: ${invalid_prefix}." >&2
    echo "${runner_output}" >&2
    exit 1
  fi
done

echo "Integration runner ownership checks passed."
