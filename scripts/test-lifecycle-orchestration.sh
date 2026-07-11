#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

parser_probe_directory="${temporary_directory}/env-parser"
mkdir -p "${parser_probe_directory}"
printf '%s\n' 'PATH=/tmp' >"${parser_probe_directory}/unknown.env"
set +e
parser_unknown_output="$("${repo_root}/scripts/release/parse_release_env.py" "${parser_probe_directory}/unknown.env" 2>&1)"
parser_unknown_status=$?
set -e
[[ "${parser_unknown_status}" -ne 0 ]]
[[ "${parser_unknown_output}" == *"key PATH is not part of the release env contract"* ]]

parser_execution_marker="${parser_probe_directory}/executed"
printf "MOBILE_IOS_ASC_APP_ID='\"; touch %s; #'\n" "${parser_execution_marker}" >"${parser_probe_directory}/literal.env"
parser_literal_output="$({
  cd "${repo_root}"
  bash -c 'set -euo pipefail; source ./scripts/release/load_release_env.sh; load_release_env_file "$1"; printf "%s\n" "$MOBILE_IOS_ASC_APP_ID"' _ "${parser_probe_directory}/literal.env"
})"
[[ "${parser_literal_output}" == "\"; touch ${parser_execution_marker}; #" ]]
[[ ! -e "${parser_execution_marker}" ]]

make_fixture="${temporary_directory}/make-fixture"
mkdir -p "${make_fixture}/scripts/release" "${make_fixture}/scripts"
git -C "${temporary_directory}" init -b master "${make_fixture}" >/dev/null
cp "${repo_root}/Makefile" "${make_fixture}/Makefile"
cp "${repo_root}/scripts/release/with_lifecycle_lock.sh" "${make_fixture}/scripts/release/with_lifecycle_lock.sh"
cp "${repo_root}/scripts/release/run_lifecycle.sh" "${make_fixture}/scripts/release/run_lifecycle.sh"
cat >"${make_fixture}/scripts/release-preflight.sh" <<'EOF_FAIL_RELEASE'
#!/usr/bin/env bash
exit 41
EOF_FAIL_RELEASE
cat >"${make_fixture}/scripts/publish.sh" <<'EOF_FAIL_PUBLISH'
#!/usr/bin/env bash
exit 42
EOF_FAIL_PUBLISH
cat >"${make_fixture}/scripts/deploy.sh" <<'EOF_FAIL_DEPLOY'
#!/usr/bin/env bash
exit 43
EOF_FAIL_DEPLOY
chmod +x \
  "${make_fixture}/scripts/release-preflight.sh" \
  "${make_fixture}/scripts/publish.sh" \
  "${make_fixture}/scripts/deploy.sh" \
  "${make_fixture}/scripts/release/run_lifecycle.sh" \
  "${make_fixture}/scripts/release/with_lifecycle_lock.sh"

set +e
make_release_injection_output="$(make -C "${make_fixture}" --no-print-directory 'RELEASE_ENV_FILE="; true; #' release-dry-run 2>&1)"
make_release_injection_status=$?
make_publish_injection_output="$(make -C "${make_fixture}" --no-print-directory 'RELEASE_ENV_FILE="; true; #' publish 2>&1)"
make_publish_injection_status=$?
make_deploy_injection_output="$(make -C "${make_fixture}" --no-print-directory 'GATEWAY_DIR="; true; #' deploy 2>&1)"
make_deploy_injection_status=$?
set -e
[[ "${make_release_injection_status}" -ne 0 ]]
[[ "${make_publish_injection_status}" -ne 0 ]]
[[ "${make_deploy_injection_status}" -ne 0 ]]
[[ "${make_release_injection_output}" == *"Error 41"* ]]
[[ "${make_publish_injection_output}" == *"Error 42"* ]]
[[ "${make_deploy_injection_output}" == *"Error 43"* ]]
[[ "${make_release_injection_output}" != *"command not found"* ]]
[[ "${make_publish_injection_output}" != *"command not found"* ]]
[[ "${make_deploy_injection_output}" != *"command not found"* ]]

make_function_marker="${temporary_directory}/make-function-executed"
set +e
make_function_output="$(make -C "${make_fixture}" --no-print-directory 'MOBILE_RELEASE_TIMESTAMP=$(shell touch '"${make_function_marker}"')' release-dry-run 2>&1)"
make_function_status=$?
make_flags_output="$(MAKEFLAGS=-i make -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
make_flags_status=$?
make_no_execute_output="$(make -n -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
make_no_execute_status=$?
bash_env_output="$(BASH_ENV="${temporary_directory}/untrusted-bash-env" make -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
bash_env_status=$?
node_options_output="$(NODE_OPTIONS=--require=untrusted make -C "${make_fixture}" --no-print-directory publish-dry-run 2>&1)"
node_options_status=$?
docker_host_output="$(DOCKER_HOST=tcp://production.example.invalid:2376 make -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
docker_host_status=$?
shell_override_output="$(SHELL=/usr/bin/true make -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
shell_override_status=$?
set -e
[[ "${make_function_status}" -ne 0 && ! -e "${make_function_marker}" ]]
[[ "${make_function_output}" == *"Error 41"* ]]
[[ "${make_flags_status}" -ne 0 && "${make_flags_output}" == *"reject Make's ignore-errors mode"* ]]
[[ "${make_no_execute_status}" -ne 0 && "${make_no_execute_output}" == *"reject Make's no-execute mode"* ]]
[[ "${bash_env_status}" -ne 0 && "${bash_env_output}" == *"BASH_ENV is not supported"* ]]
[[ "${node_options_status}" -ne 0 && "${node_options_output}" == *"NODE_OPTIONS is not supported"* ]]
[[ "${docker_host_status}" -ne 0 && "${docker_host_output}" == *"DOCKER_HOST is not supported"* ]]
[[ "${shell_override_status}" -ne 0 && "${shell_override_output}" == *"Error 41"* ]]

bash_function_marker="${temporary_directory}/bash-function-executed"
clean_runner_output="$(/bin/sh "${repo_root}/scripts/release/run_lifecycle.sh" -c 'printf clean-runner')"
[[ "${clean_runner_output}" == "clean-runner" ]]
set +e
# Ubuntu's /bin/sh removes Bash's non-POSIX exported-function key before the
# runner can inspect it, so use Bash to exercise the runner's rejection path.
bash_function_output="$(env "BASH_FUNC_git%%=() { touch ${bash_function_marker}; }" bash --noprofile --norc "${repo_root}/scripts/release/run_lifecycle.sh" -c 'git --version' 2>&1)"
bash_function_status=$?
shell_options_output="$(env SHELLOPTS=xtrace /bin/sh "${repo_root}/scripts/release/run_lifecycle.sh" -c true 2>&1)"
shell_options_status=$?
set -e
[[ "${bash_function_status}" -ne 0 && ! -e "${bash_function_marker}" ]]
[[ "${bash_function_output}" == *"BASH_FUNC_git%% is not supported"* ]]
[[ "${shell_options_status}" -ne 0 && "${shell_options_output}" == *"SHELLOPTS is not supported"* ]]

fake_bash_directory="${temporary_directory}/fake-bash"
mkdir "${fake_bash_directory}"
ln -s /usr/bin/true "${fake_bash_directory}/bash"
set +e
fake_bash_output="$(PATH="${fake_bash_directory}:${PATH}" make -C "${make_fixture}" --no-print-directory release-dry-run 2>&1)"
fake_bash_status=$?
set -e
[[ "${fake_bash_status}" -ne 0 ]]
[[ "${fake_bash_output}" == *"lifecycle requires Bash from a canonical system or Homebrew path"* ]]

docker_identity_bin="${temporary_directory}/docker-identity-bin"
mkdir "${docker_identity_bin}"
cat >"${docker_identity_bin}/docker" <<'EOF_DOCKER_IDENTITY'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1 $2" == "context show" ]]; then
  printf '%s\n' fixture
  exit 0
fi
if [[ "$1 $2" == "context inspect" ]]; then
  printf '%s\n' "${FAKE_DOCKER_ENDPOINT}"
  exit 0
fi
exit 97
EOF_DOCKER_IDENTITY
chmod +x "${docker_identity_bin}/docker"
set +e
remote_docker_output="$({
  PATH="${docker_identity_bin}:${PATH}" FAKE_DOCKER_ENDPOINT=tcp://production.example.invalid:2376 \
    bash -c 'source "$1"; assert_local_docker_endpoint' _ "${repo_root}/scripts/release/docker_identity.sh"
} 2>&1)"
remote_docker_status=$?
set -e
[[ "${remote_docker_status}" -ne 0 ]]
[[ "${remote_docker_output}" == *"canonical lifecycle requires a local Docker endpoint"* ]]
PATH="${docker_identity_bin}:${PATH}" FAKE_DOCKER_ENDPOINT=unix:///tmp/fixture-docker.sock \
  bash -c 'source "$1"; assert_local_docker_endpoint' _ "${repo_root}/scripts/release/docker_identity.sh"

identity_repository="${temporary_directory}/identity-repository"
git -C "${temporary_directory}" init -b master "${identity_repository}" >/dev/null
git -C "${identity_repository}" remote add origin git@github.com:tyemirov/loopaware.git
(
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_canonical_github_origin "${identity_repository}" LoopAware tyemirov/loopaware
)
git -C "${identity_repository}" config 'url.ssh://evil.invalid/.insteadOf' 'git@github.com:'
set +e
instead_of_output="$({
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_canonical_github_origin "${identity_repository}" LoopAware tyemirov/loopaware
} 2>&1)"
instead_of_status=$?
set -e
[[ "${instead_of_status}" -ne 0 ]]
[[ "${instead_of_output}" == *"effective origin fetch URL must resolve to the canonical GitHub repository"* ]]
git -C "${identity_repository}" config --unset-all 'url.ssh://evil.invalid/.insteadOf'
git -C "${identity_repository}" config 'url.ssh://evil.invalid/.pushInsteadOf' 'git@github.com:'
set +e
push_instead_of_output="$({
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_canonical_github_origin "${identity_repository}" LoopAware tyemirov/loopaware
} 2>&1)"
push_instead_of_status=$?
set -e
[[ "${push_instead_of_status}" -ne 0 ]]
[[ "${push_instead_of_output}" == *"effective origin push URL must resolve to the canonical GitHub repository"* ]]

set +e
integration_override_output="$(LOOPAWARE_BASE_URL=https://production.example.invalid "${repo_root}/tests/scripts/run-integration.sh" 2>&1)"
integration_override_status=$?
set -e
[[ "${integration_override_status}" -ne 0 ]]
[[ "${integration_override_output}" == *"rejects inherited LOOPAWARE_BASE_URL"* ]]

remote_fixture="${temporary_directory}/remote-state"
remote_origin="${temporary_directory}/remote-state-origin.git"
git init --bare "${remote_origin}" >/dev/null
git -C "${temporary_directory}" init -b master "${remote_fixture}" >/dev/null
git -C "${remote_fixture}" config user.name "Remote State Contract"
git -C "${remote_fixture}" config user.email "remote-state@mprlab.invalid"
printf 'source\n' >"${remote_fixture}/tracked.txt"
git -C "${remote_fixture}" add tracked.txt
git -C "${remote_fixture}" commit -m "Add remote state fixture" >/dev/null
git -C "${remote_fixture}" tag -a v1.2.2 -m "Release v1.2.2"
git -C "${remote_fixture}" remote add origin "${remote_origin}"
git -C "${remote_fixture}" push -u origin master >/dev/null
git -C "${remote_fixture}" push origin v1.2.2 >/dev/null
git --git-dir="${remote_origin}" symbolic-ref HEAD refs/heads/master
(
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_remote_default_and_release_tags "${remote_fixture}" fixture
)
printf 'unpublished\n' >>"${remote_fixture}/tracked.txt"
git -C "${remote_fixture}" add tracked.txt
git -C "${remote_fixture}" commit -m "Add unpublished non-release commit" >/dev/null
set +e
remote_drift_output="$({
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_remote_default_and_release_tags "${remote_fixture}" fixture allow-prepared-release
} 2>&1)"
remote_drift_status=$?
set -e
[[ "${remote_drift_status}" -ne 0 ]]
[[ "${remote_drift_output}" == *"is not the one exact locally prepared release"* ]]

prepared_remote_fixture="${temporary_directory}/prepared-remote-state"
git clone "${remote_origin}" "${prepared_remote_fixture}" >/dev/null
git -C "${prepared_remote_fixture}" config user.name "Remote State Contract"
git -C "${prepared_remote_fixture}" config user.email "remote-state@mprlab.invalid"
printf '# Changelog\n' >"${prepared_remote_fixture}/CHANGELOG.md"
git -C "${prepared_remote_fixture}" add CHANGELOG.md
git -C "${prepared_remote_fixture}" commit -m "Release v1.2.3" >/dev/null
git -C "${prepared_remote_fixture}" tag -a v1.2.3 -m "Release v1.2.3"
(
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_remote_default_and_release_tags "${prepared_remote_fixture}" fixture allow-prepared-release
)
set +e
strict_prepared_output="$({
  source "${repo_root}/scripts/release/repository_identity.sh"
  assert_remote_default_and_release_tags "${prepared_remote_fixture}" fixture
} 2>&1)"
strict_prepared_status=$?
set -e
[[ "${strict_prepared_status}" -ne 0 ]]
[[ "${strict_prepared_output}" == *"does not match origin/master"* ]]

attestation_repository="${temporary_directory}/attestation-repository"
attestation_remote_assets="${temporary_directory}/attestation-assets"
attestation_bin="${temporary_directory}/attestation-bin"
mkdir -p "${attestation_repository}/scripts/release" "${attestation_remote_assets}" "${attestation_bin}"
git -C "${temporary_directory}" init -b master "${attestation_repository}" >/dev/null
git -C "${attestation_repository}" config user.name "Publication Attestation Contract"
git -C "${attestation_repository}" config user.email "publication-attestation@mprlab.invalid"
printf 'fixture\n' >"${attestation_repository}/tracked.txt"
git -C "${attestation_repository}" add tracked.txt
git -C "${attestation_repository}" commit -m "Add attestation fixture" >/dev/null
attestation_commit="$(git -C "${attestation_repository}" rev-parse HEAD)"
git -C "${attestation_repository}" remote add origin git@github.com:tyemirov/loopaware.git
cp "${repo_root}/scripts/release/record_publication.sh" "${attestation_repository}/scripts/release/record_publication.sh"
cp "${repo_root}/scripts/release/repository_identity.sh" "${attestation_repository}/scripts/release/repository_identity.sh"
cat >"${attestation_repository}/scripts/release/release_helper.py" <<'EOF_ATTESTATION_HELPER'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "verify-release-artifact" ]]
EOF_ATTESTATION_HELPER
chmod +x "${attestation_repository}/scripts/release/record_publication.sh" "${attestation_repository}/scripts/release/release_helper.py"
attestation_artifact_directory="${attestation_repository}/.git/mprlab-release"
mkdir -p "${attestation_artifact_directory}"
python3 - "${attestation_artifact_directory}/manifest.json" "${attestation_commit}" <<'PY_ATTESTATION_MANIFEST'
import json
import pathlib
import sys

pathlib.Path(sys.argv[1]).write_text(
    json.dumps(
        {
            "schema_version": 2,
            "artifact_kind": "mprlab.release",
            "version": "v1.2.3",
            "source_commit": sys.argv[2],
            "release_commit": sys.argv[2],
            "payloads": [],
        }
    ),
    encoding="utf-8",
)
PY_ATTESTATION_MANIFEST
cat >"${attestation_bin}/gh" <<'EOF_ATTESTATION_GH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1 $2" == "release view" ]]; then
  if [[ -f "${ATTESTATION_REMOTE_ASSETS}/publication.json" ]]; then
    printf '%s\n' '{"assets":[{"name":"publication.json"}]}'
  else
    printf '%s\n' '{"assets":[]}'
  fi
  exit 0
fi
if [[ "$1 $2" == "release upload" ]]; then
  cp "$4" "${ATTESTATION_REMOTE_ASSETS}/publication.json"
  exit 0
fi
if [[ "$1 $2" == "release download" ]]; then
  destination=""
  shift 3
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --repo|--pattern) shift 2 ;;
      --dir) destination="$2"; shift 2 ;;
      *) printf 'unexpected attestation gh argument: %s\n' "$1" >&2; exit 97 ;;
    esac
  done
  cp "${ATTESTATION_REMOTE_ASSETS}/publication.json" "${destination}/publication.json"
  exit 0
fi
printf 'unexpected attestation gh command: %s\n' "$*" >&2
exit 97
EOF_ATTESTATION_GH
chmod +x "${attestation_bin}/gh"
attestation_manifest_sha256="$(shasum -a 256 "${attestation_artifact_directory}/manifest.json" | awk '{print $1}')"
attestation_output="$({
  cd "${attestation_repository}"
  PATH="${attestation_bin}:${PATH}" ATTESTATION_REMOTE_ASSETS="${attestation_remote_assets}" LOOPAWARE_RELEASE_MANIFEST_SHA256="${attestation_manifest_sha256}" \
    ./scripts/release/record_publication.sh
})"
[[ "${attestation_output}" == *"Recorded complete publication for v1.2.3."* ]]
cmp "${attestation_artifact_directory}/publication.json" "${attestation_remote_assets}/publication.json"
attestation_verify_output="$({
  cd "${attestation_repository}"
  PATH="${attestation_bin}:${PATH}" ATTESTATION_REMOTE_ASSETS="${attestation_remote_assets}" LOOPAWARE_RELEASE_MANIFEST_SHA256="${attestation_manifest_sha256}" \
    ./scripts/release/record_publication.sh --verify-only
})"
[[ "${attestation_verify_output}" == *"Verified complete publication for v1.2.3."* ]]
printf ' ' >>"${attestation_remote_assets}/publication.json"
set +e
attestation_drift_output="$({
  cd "${attestation_repository}"
  PATH="${attestation_bin}:${PATH}" ATTESTATION_REMOTE_ASSETS="${attestation_remote_assets}" LOOPAWARE_RELEASE_MANIFEST_SHA256="${attestation_manifest_sha256}" \
    ./scripts/release/record_publication.sh --verify-only
} 2>&1)"
attestation_drift_status=$?
set -e
[[ "${attestation_drift_status}" -ne 0 ]]
[[ "${attestation_drift_output}" == *"published completion attestation differs from the local release"* ]]

fixture_repository="${temporary_directory}/loopaware"
mkdir -p "${fixture_repository}/scripts/release"
git -C "${temporary_directory}" init -b master "${fixture_repository}" >/dev/null
git -C "${fixture_repository}" config user.name "Lifecycle Orchestration Contract"
git -C "${fixture_repository}" config user.email "lifecycle-orchestration@mprlab.invalid"
printf 'fixture\n' >"${fixture_repository}/tracked.txt"
git -C "${fixture_repository}" add tracked.txt
git -C "${fixture_repository}" commit -m "Add lifecycle fixture" >/dev/null
fixture_commit="$(git -C "${fixture_repository}" rev-parse HEAD)"

cp "${repo_root}/scripts/release/with_lifecycle_lock.sh" "${fixture_repository}/scripts/release/with_lifecycle_lock.sh"
cp "${repo_root}/scripts/publish.sh" "${fixture_repository}/scripts/publish.sh"

cat >"${fixture_repository}/scripts/nested-lock.sh" <<'EOF_NESTED_LOCK'
#!/usr/bin/env bash
set -euo pipefail
exec ./scripts/release/with_lifecycle_lock.sh nested /usr/bin/true
EOF_NESTED_LOCK
chmod +x \
  "${fixture_repository}/scripts/nested-lock.sh" \
  "${fixture_repository}/scripts/publish.sh" \
  "${fixture_repository}/scripts/release/with_lifecycle_lock.sh"

set +e
nested_lock_output="$({
  cd "${fixture_repository}"
  ./scripts/release/with_lifecycle_lock.sh outer ./scripts/nested-lock.sh
} 2>&1)"
nested_lock_status=$?
set -e
[[ "${nested_lock_status}" -ne 0 ]]
[[ "${nested_lock_output}" == *"lifecycle operation is already locked"* ]]
[[ ! -e "${fixture_repository}/.git/mprlab-lifecycle.lock" ]]

lock_success_output="$({
  cd "${fixture_repository}"
  ./scripts/release/with_lifecycle_lock.sh verification printf '%s\n' 'lock recovered'
})"
[[ "${lock_success_output}" == "lock recovered" ]]
[[ ! -e "${fixture_repository}/.git/mprlab-lifecycle.lock" ]]

manifest_directory="${fixture_repository}/.git/mprlab-release"
manifest_path="${manifest_directory}/manifest.json"
stage_log="${temporary_directory}/publish-stages.log"
mkdir -p "${manifest_directory}"
python3 - "${manifest_path}" "${fixture_commit}" <<'PY_MANIFEST'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
commit = sys.argv[2]
path.write_text(
    json.dumps(
        {
            "schema_version": 2,
            "artifact_kind": "mprlab.release",
            "version": "v1.2.3",
            "source_commit": commit,
            "release_commit": commit,
        },
        sort_keys=True,
    )
    + "\n",
    encoding="utf-8",
)
PY_MANIFEST

cat >"${fixture_repository}/scripts/publish-preflight.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' preflight >>"${STAGE_LOG}"
EOF_STAGE
cat >"${fixture_repository}/scripts/publish-release.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' release >>"${STAGE_LOG}"
EOF_STAGE
cat >"${fixture_repository}/scripts/release/publish_container_artifacts.sh" <<'EOF_CONTAINER_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' container >>"${STAGE_LOG}"
if [[ "${MUTATE_RELEASE_MANIFEST:-0}" == "1" ]]; then
  printf ' ' >>"$(git rev-parse --git-path mprlab-release)/manifest.json"
fi
EOF_CONTAINER_STAGE
cat >"${fixture_repository}/scripts/publish-react-native.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' react-native >>"${STAGE_LOG}"
EOF_STAGE
cat >"${fixture_repository}/scripts/publish-mobile.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' mobile >>"${STAGE_LOG}"
[[ "${FAIL_MOBILE:-0}" != "1" ]] || exit 42
EOF_STAGE
cat >"${fixture_repository}/scripts/release/record_publication.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' attestation >>"${STAGE_LOG}"
EOF_STAGE
chmod +x \
  "${fixture_repository}/scripts/publish-preflight.sh" \
  "${fixture_repository}/scripts/publish-release.sh" \
  "${fixture_repository}/scripts/release/publish_container_artifacts.sh" \
  "${fixture_repository}/scripts/release/record_publication.sh" \
  "${fixture_repository}/scripts/publish-react-native.sh" \
  "${fixture_repository}/scripts/publish-mobile.sh"

: >"${stage_log}"
happy_output="$({
  cd "${fixture_repository}"
  STAGE_LOG="${stage_log}" ./scripts/publish.sh
})"
[[ "${happy_output}" == *"LoopAware publication complete for v1.2.3."* ]]
[[ "$(paste -sd, "${stage_log}")" == "preflight,release,container,react-native,mobile,attestation" ]]

printf '%s\n' '{"schema":"mprlab.loopaware-publication.v1"}' >"${manifest_directory}/publication.json"
: >"${stage_log}"
already_published_output="$({
  cd "${fixture_repository}"
  STAGE_LOG="${stage_log}" ./scripts/publish.sh
})"
rm "${manifest_directory}/publication.json"
[[ "${already_published_output}" == *"publication was already completed for v1.2.3; no provider upload was repeated"* ]]
[[ "$(paste -sd, "${stage_log}")" == "attestation" ]]

: >"${stage_log}"
set +e
partial_publish_output="$({
  cd "${fixture_repository}"
  STAGE_LOG="${stage_log}" FAIL_MOBILE=1 ./scripts/publish.sh
} 2>&1)"
partial_publish_status=$?
set -e
[[ "${partial_publish_status}" -eq 42 ]]
[[ "$(paste -sd, "${stage_log}")" == "preflight,release,container,react-native,mobile" ]]
[[ "${partial_publish_output}" != *"publication complete"* ]]

: >"${stage_log}"
set +e
drift_output="$({
  cd "${fixture_repository}"
  STAGE_LOG="${stage_log}" MUTATE_RELEASE_MANIFEST=1 ./scripts/publish.sh
} 2>&1)"
drift_status=$?
set -e
[[ "${drift_status}" -ne 0 ]]
[[ "${drift_output}" == *"prepared release manifest changed during publication; refusing to mix release identities"* ]]
[[ "$(paste -sd, "${stage_log}")" == "preflight,release,container" ]]

echo "lifecycle orchestration contract checks passed"
