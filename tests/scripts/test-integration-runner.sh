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

if ! grep -F -- "-p loopaware-integration up --build -d" "${docker_log}" >/dev/null; then
  echo "Expected the canonical integration Compose project." >&2
  cat "${docker_log}" >&2
  exit 1
fi

echo "Integration runner ownership checks passed."
