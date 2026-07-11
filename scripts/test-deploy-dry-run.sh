#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

fixture_repository="${temporary_directory}/loopaware"
remote_repository="${temporary_directory}/origin.git"
gateway_directory="${temporary_directory}/gateway"
gateway_remote_repository="${temporary_directory}/gateway-origin.git"
fake_bin="${temporary_directory}/bin"
command_log="${temporary_directory}/commands.log"
gh_command_log="${temporary_directory}/gh-commands.log"
gateway_contract_log="${temporary_directory}/gateway-contract.log"
release_asset_source="${temporary_directory}/release-assets"
real_make="$(command -v make)"
mkdir -p "${fixture_repository}/scripts/release" "${gateway_directory}" "${fake_bin}" "${release_asset_source}"
cp "${repo_root}/scripts/deploy.sh" "${fixture_repository}/scripts/deploy.sh"
cp "${repo_root}/scripts/release/repository_identity.sh" "${fixture_repository}/scripts/release/repository_identity.sh"
cp "${repo_root}/scripts/release/with_lifecycle_lock.sh" "${fixture_repository}/scripts/release/with_lifecycle_lock.sh"
cp "${repo_root}/scripts/release/run_lifecycle.sh" "${fixture_repository}/scripts/release/run_lifecycle.sh"
cp "${repo_root}/Makefile" "${fixture_repository}/Makefile"

cat >"${fixture_repository}/scripts/release/deploy_pages_artifact.sh" <<'EOF_PAGES'
#!/usr/bin/env bash
set -euo pipefail
printf 'pages|%s\n' "$*" >>"${COMMAND_LOG}"
[[ " $* " != *" --skip-verify "* ]] || { printf 'dry run accepted a Pages skip override\n' >&2; exit 95; }
if [[ " $* " == *" --verify-only "* ]]; then
  [[ "${FAKE_PAGES_FAIL:-0}" != "1" ]] || exit 42
fi
EOF_PAGES
chmod +x "${fixture_repository}/scripts/release/deploy_pages_artifact.sh"

git init --bare "${remote_repository}" >/dev/null
git -C "${fixture_repository}" init -b master >/dev/null
git -C "${fixture_repository}" config user.name "Deploy Dry Run Contract"
git -C "${fixture_repository}" config user.email "deploy-dry-run@mprlab.invalid"
git -C "${fixture_repository}" add Makefile scripts
git -C "${fixture_repository}" commit -m "Add deploy source fixture" >/dev/null
git -C "${fixture_repository}" commit --allow-empty -m "Release fixture" >/dev/null
git -C "${fixture_repository}" remote add origin "git@github.com:tyemirov/loopaware.git"
git -C "${fixture_repository}" tag -a v1.2.3 -m "Release v1.2.3"
git -C "${fixture_repository}" push "${remote_repository}" master refs/tags/v1.2.3 >/dev/null
git --git-dir="${remote_repository}" symbolic-ref HEAD refs/heads/master
release_commit="$(git -C "${fixture_repository}" rev-parse HEAD)"
source_commit="$(git -C "${fixture_repository}" rev-parse HEAD^)"
printf 'fixture pages\n' >"${release_asset_source}/pages.tar.gz"
cat >"${release_asset_source}/container.json" <<'JSON_CONTAINER'
{
  "schema_version": 1,
  "artifact_kind": "mprlab.container",
  "name": "loopaware",
  "image": "ghcr.io/tyemirov/loopaware",
  "version": "v1.2.3",
  "platforms": [{
    "platform": "linux/amd64",
    "token": "linux-amd64",
    "local_ref": "mprlab-release.local/loopaware:v1.2.3-linux-amd64",
    "image_id": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    "archive": "payloads/containers/loopaware/linux-amd64.tar",
    "sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
  }]
}
JSON_CONTAINER
python3 - "${release_asset_source}/manifest.json" "${release_asset_source}/container.json" "${release_asset_source}/pages.tar.gz" "${release_commit}" "${source_commit}" <<'PY_RELEASE_MANIFEST'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
descriptor_path = pathlib.Path(sys.argv[2])
pages_path = pathlib.Path(sys.argv[3])

def entry(path, relative):
    payload = path.read_bytes()
    return {"path": relative, "size": len(payload), "sha256": hashlib.sha256(payload).hexdigest()}

manifest = {
    "schema_version": 2,
    "artifact_kind": "mprlab.release",
    "version": "v1.2.3",
    "release_commit": sys.argv[4],
    "source_commit": sys.argv[5],
    "default_branch": "master",
    "release_timestamp": "2026-07-11T00:00:00+00:00",
    "payloads": [
        entry(descriptor_path, "payloads/containers/loopaware/container.json"),
        entry(pages_path, "payloads/release-assets/pages.tar.gz"),
    ],
}
manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
publication = {
    "schema": "mprlab.loopaware-publication.v1",
    "status": "complete",
    "version": "v1.2.3",
    "release_commit": sys.argv[4],
    "source_commit": sys.argv[5],
    "release_manifest_sha256": hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
    "completed_stages": [
        "github-release",
        "ghcr",
        "npm",
        "app-store-connect-upload-accepted",
        "google-play",
    ],
}
(manifest_path.parent / "publication.json").write_text(json.dumps(publication), encoding="utf-8")
PY_RELEASE_MANIFEST

git init --bare "${gateway_remote_repository}" >/dev/null
git -C "${gateway_directory}" init -b master >/dev/null
git -C "${gateway_directory}" config user.name "Gateway Deploy Dry Run Contract"
git -C "${gateway_directory}" config user.email "gateway-deploy-dry-run@mprlab.invalid"
cat >"${gateway_directory}/Makefile" <<'EOF_GATEWAY_MAKE'
deploy-preflight-contract:
	@printf '%s\n' 'mprlab.loopaware-deploy.v2'
test-loopaware-deploy-preflight-contract:
	@true
deploy-loopaware-backend-preflight:
	@true
deploy-loopaware-backend:
	@true
EOF_GATEWAY_MAKE
git -C "${gateway_directory}" add Makefile
git -C "${gateway_directory}" commit -m "Add gateway fixture" >/dev/null
git -C "${gateway_directory}" remote add origin "git@github.com:MarcoPoloResearchLab/mprlab-gateway.git"
git -C "${gateway_directory}" push "${gateway_remote_repository}" master >/dev/null
git --git-dir="${gateway_remote_repository}" symbolic-ref HEAD refs/heads/master

cat >"${fake_bin}/docker" <<'EOF_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == "buildx version" ]]; then
  exit 0
fi
if [[ "$*" == buildx\ imagetools\ inspect* ]]; then
  if [[ "$*" == *"--raw"* ]]; then
    if [[ "$*" == *"ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa --raw"* ]]; then
      printf '%s\n' '{"manifests":[{"digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","platform":{"os":"linux","architecture":"amd64"}}]}'
      exit 0
    fi
    if [[ "$*" == *"ghcr.io/tyemirov/loopaware@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb --raw"* ]]; then
      printf '%s\n' '{"schemaVersion":2,"config":{"digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}}'
      exit 0
    fi
    printf 'image provenance was not inspected through a pinned digest: %s\n' "$*" >&2
    exit 96
  fi
  if [[ "$*" == *"--format"* ]]; then
    source_commit="$(/usr/bin/git rev-parse 'v1.2.3^{commit}^')"
    printf '{"org.opencontainers.image.revision":"%s","org.opencontainers.image.version":"v1.2.3","org.opencontainers.image.source":"https://github.com/tyemirov/loopaware"}\n' "${source_commit}"
    exit 0
  fi
  printf '%s\n' 'Name: fixture' 'Digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  exit 0
fi
printf 'unexpected docker command: %s\n' "$*" >&2
exit 97
EOF_DOCKER

cat >"${fake_bin}/make" <<'EOF_MAKE'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == *"test-loopaware-deploy-preflight-contract"* ]]; then
  printf '%s\n' 'test-loopaware-deploy-preflight-contract' >>"${GATEWAY_CONTRACT_LOG}"
  exec /usr/bin/make "$@"
fi
if [[ "$*" == *"deploy-preflight-contract"* ]]; then
  printf '%s\n' 'deploy-preflight-contract' >>"${GATEWAY_CONTRACT_LOG}"
  exec /usr/bin/make "$@"
fi
printf 'make|%s\n' "$*" >>"${COMMAND_LOG}"
expected_commit="$(git -C "${GATEWAY_FIXTURE_DIR}" rev-parse HEAD)"
if [[ "$*" == *"MPRLAB_GATEWAY_EXPECTED_COMMIT=${expected_commit}"* && "$*" == *"MPRLAB_LOOPAWARE_IMAGE_REF=ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"* && "$*" == *"deploy-loopaware-backend-preflight"* ]]; then
  if [[ "${FAKE_MUTATE_LOOPAWARE:-0}" == "1" ]]; then touch "${LOOPAWARE_FIXTURE_DIR}/concurrent-change"; fi
  if [[ "${FAKE_MUTATE_GATEWAY:-0}" == "1" ]]; then touch "${GATEWAY_FIXTURE_DIR}/concurrent-change"; fi
  exit 0
fi
if [[ "$*" == *"MPRLAB_GATEWAY_EXPECTED_COMMIT=${expected_commit}"* && "$*" == *"MPRLAB_LOOPAWARE_IMAGE_REF=ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"* && "$*" == *"deploy-loopaware-backend"* && "$*" != *"deploy-loopaware-backend-preflight"* ]]; then
  if [[ "${FAKE_MUTATE_LOOPAWARE:-0}" == "1" ]]; then touch "${LOOPAWARE_FIXTURE_DIR}/concurrent-change"; fi
  if [[ "${FAKE_MUTATE_GATEWAY:-0}" == "1" ]]; then touch "${GATEWAY_FIXTURE_DIR}/concurrent-change"; fi
  exit 0
fi
printf 'unexpected make command: %s\n' "$*" >&2
exit 97
EOF_MAKE
cat >"${fake_bin}/gh" <<'EOF_GH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1 $2" == "release download" ]]; then
  printf 'release-download|%s\n' "$*" >>"${GH_COMMAND_LOG}"
  destination=""
  shift 3
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --repo) [[ "$2" == "tyemirov/loopaware" ]]; shift 2 ;;
      --pattern) shift 2 ;;
      --dir) destination="$2"; shift 2 ;;
      *) printf 'unexpected gh release argument: %s\n' "$1" >&2; exit 97 ;;
    esac
  done
  cp "${FAKE_RELEASE_ASSETS}/manifest.json" "${destination}/manifest.json"
  cp "${FAKE_RELEASE_ASSETS}/container.json" "${destination}/container.json"
  cp "${FAKE_RELEASE_ASSETS}/publication.json" "${destination}/publication.json"
  cp "${FAKE_RELEASE_ASSETS}/pages.tar.gz" "${destination}/pages.tar.gz"
  exit 0
fi
if [[ "$*" == "repo view tyemirov/loopaware --json viewerPermission --jq .viewerPermission" ]]; then
  printf '%s\n' "${FAKE_GH_PERMISSION:-ADMIN}"
  exit 0
fi
printf 'unexpected gh command: %s\n' "$*" >&2
exit 97
EOF_GH
cat >"${fake_bin}/git" <<'EOF_GIT'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "-C" && "${3:-}" == "ls-remote" && "${4:-}" == "--symref" && "${5:-}" == "origin" && "${6:-}" == "HEAD" ]]; then
  case "$2" in
    "${LOOPAWARE_FIXTURE_DIR}") exec /usr/bin/git ls-remote --symref "${LOOPAWARE_REMOTE_REPOSITORY}" HEAD ;;
    "${GATEWAY_FIXTURE_DIR}") exec /usr/bin/git ls-remote --symref "${GATEWAY_REMOTE_REPOSITORY}" HEAD ;;
  esac
fi
if [[ "${1:-}" == "ls-remote" && "${2:-}" == "--tags" && "${3:-}" == "origin" && "${PWD}" == "${LOOPAWARE_FIXTURE_DIR}" ]]; then
  exec /usr/bin/git ls-remote --tags "${LOOPAWARE_REMOTE_REPOSITORY}" "${@:4}"
fi
exec /usr/bin/git "$@"
EOF_GIT
chmod +x "${fake_bin}/docker" "${fake_bin}/make" "${fake_bin}/gh" "${fake_bin}/git"
fixture_repository="$(git -C "${fixture_repository}" rev-parse --show-toplevel)"
gateway_directory="$(git -C "${gateway_directory}" rev-parse --show-toplevel)"
export FAKE_RELEASE_ASSETS="${release_asset_source}"
export GATEWAY_FIXTURE_DIR="${gateway_directory}"
export LOOPAWARE_FIXTURE_DIR="${fixture_repository}"
export GATEWAY_REMOTE_REPOSITORY="${gateway_remote_repository}"
export LOOPAWARE_REMOTE_REPOSITORY="${remote_repository}"
export GH_COMMAND_LOG="${gh_command_log}"
export GATEWAY_CONTRACT_LOG="${gateway_contract_log}"

dry_run_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run
)"
[[ "${dry_run_output}" == *"LoopAware deploy dry run passed; production hosts were not contacted and production state was not changed."* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "2" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]
[[ "$(sed -n '1p' "${gateway_contract_log}")" == "deploy-preflight-contract" ]]
[[ "$(sed -n '2p' "${gateway_contract_log}")" == "test-loopaware-deploy-preflight-contract" ]]
grep -Fq 'release-download|release download v1.2.3 --repo tyemirov/loopaware --pattern manifest.json --pattern container.json --pattern publication.json --pattern pages.tar.gz --dir ' "${gh_command_log}"
sed -n '1p' "${command_log}" | grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir '
sed -n '1p' "${command_log}" | grep -Fq -- '--verify-only'
sed -n '2p' "${command_log}" | grep -Fq 'make|-C'
gateway_commit="$(git -C "${gateway_directory}" rev-parse HEAD)"
sed -n '2p' "${command_log}" | grep -Fq "MPRLAB_GATEWAY_EXPECTED_COMMIT=${gateway_commit}"
sed -n '2p' "${command_log}" | grep -Fq 'MPRLAB_LOOPAWARE_IMAGE_REF=ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
sed -n '2p' "${command_log}" | grep -Fq 'deploy-loopaware-backend-preflight'
[[ "$(sed -n '2p' "${command_log}")" != *"MPRLAB_DEPLOY_PREFLIGHT_ONLY="* ]]

: >"${command_log}"
set +e
pages_args_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    PAGES_DEPLOY_ARGS="--skip-verify" \
    deploy-dry-run 2>&1
)"
pages_args_status=$?
set -e
[[ "${pages_args_status}" -ne 0 ]]
[[ "${pages_args_output}" == *"PAGES_DEPLOY_ARGS is not supported"* ]]
[[ ! -s "${command_log}" ]]

gateway_git_common_dir="$(git -C "${gateway_directory}" rev-parse --git-common-dir)"
if [[ "${gateway_git_common_dir}" != /* ]]; then gateway_git_common_dir="${gateway_directory}/${gateway_git_common_dir}"; fi
contending_gateway_lock="${gateway_git_common_dir}/mprlab-loopaware-deploy.lock"
mkdir "${contending_gateway_lock}"
printf '%s\n' 'pid=other' >"${contending_gateway_lock}/owner"
set +e
contending_lock_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
contending_lock_status=$?
set -e
[[ "${contending_lock_status}" -ne 0 ]]
[[ "${contending_lock_output}" == *"gateway LoopAware deployment is already locked"* ]]
[[ -f "${contending_gateway_lock}/owner" ]]
[[ "$(<"${contending_gateway_lock}/owner")" == "pid=other" ]]
rm "${contending_gateway_lock}/owner"
rmdir "${contending_gateway_lock}"

git -C "${fixture_repository}" config remote.origin.pushurl https://example.invalid/not-loopaware.git
set +e
pushurl_output="$(
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory GATEWAY_DIR="${gateway_directory}" deploy-dry-run 2>&1
)"
pushurl_status=$?
set -e
[[ "${pushurl_status}" -ne 0 ]]
[[ "${pushurl_output}" == *"origin pushurl overrides are not supported"* ]]
git -C "${fixture_repository}" config --unset-all remote.origin.pushurl

set +e
gh_host_output="$(
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" GH_HOST=example.invalid \
  "${real_make}" -C "${fixture_repository}" --no-print-directory GATEWAY_DIR="${gateway_directory}" deploy-dry-run 2>&1
)"
gh_host_status=$?
set -e
[[ "${gh_host_status}" -ne 0 ]]
[[ "${gh_host_output}" == *"GH_HOST must be github.com"* ]]

: >"${command_log}"
set +e
loopaware_drift_output="$(
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" FAKE_MUTATE_LOOPAWARE=1 \
  "${real_make}" -C "${fixture_repository}" --no-print-directory GATEWAY_DIR="${gateway_directory}" deploy-dry-run 2>&1
)"
loopaware_drift_status=$?
set -e
[[ "${loopaware_drift_status}" -ne 0 ]]
[[ "${loopaware_drift_output}" == *"LoopAware checkout changed after deploy preflight began"* ]]
grep -Fq 'deploy-loopaware-backend-preflight' "${command_log}"
rm "${fixture_repository}/concurrent-change"

: >"${command_log}"
set +e
gateway_drift_output="$(
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" FAKE_MUTATE_GATEWAY=1 \
  "${real_make}" -C "${fixture_repository}" --no-print-directory GATEWAY_DIR="${gateway_directory}" deploy-dry-run 2>&1
)"
gateway_drift_status=$?
set -e
[[ "${gateway_drift_status}" -ne 0 ]]
[[ "${gateway_drift_output}" == *"gateway checkout changed after deploy preflight began"* ]]
grep -Fq 'deploy-loopaware-backend-preflight' "${command_log}"
rm "${gateway_directory}/concurrent-change"

: >"${command_log}"
: >"${gh_command_log}"
: >"${gateway_contract_log}"
deploy_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy
)"
[[ "${deploy_output}" == *"LoopAware deploy complete"* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "3" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]
[[ "$(sed -n '1p' "${gateway_contract_log}")" == "deploy-preflight-contract" ]]
[[ "$(sed -n '2p' "${gateway_contract_log}")" == "test-loopaware-deploy-preflight-contract" ]]
sed -n '1p' "${command_log}" | grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir '
sed -n '1p' "${command_log}" | grep -Fq -- '--verify-only'
sed -n '2p' "${command_log}" | grep -Fq "MPRLAB_GATEWAY_EXPECTED_COMMIT=${gateway_commit}"
sed -n '2p' "${command_log}" | grep -Fq 'MPRLAB_LOOPAWARE_IMAGE_REF=ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
sed -n '2p' "${command_log}" | grep -Fq 'deploy-loopaware-backend'
[[ "$(sed -n '2p' "${command_log}")" != *"deploy-loopaware-backend-preflight"* ]]
[[ "$(sed -n '2p' "${command_log}")" != *"MPRLAB_DEPLOY_PREFLIGHT_ONLY="* ]]
sed -n '3p' "${command_log}" | grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir '
[[ "$(sed -n '3p' "${command_log}")" != *"--verify-only"* ]]
verified_artifact_dir="$(sed -n '1p' "${command_log}" | sed -E 's/.*--artifact-dir ([^ ]+).*/\1/')"
activated_artifact_dir="$(sed -n '3p' "${command_log}" | sed -E 's/.*--artifact-dir ([^ ]+).*/\1/')"
[[ -n "${verified_artifact_dir}" && "${verified_artifact_dir}" == "${activated_artifact_dir}" ]]

cp "${release_asset_source}/publication.json" "${release_asset_source}/publication.json.valid"
python3 - "${release_asset_source}/publication.json" <<'PY_TAMPER_PUBLICATION'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
publication = json.loads(path.read_text(encoding="utf-8"))
publication["status"] = "partial"
path.write_text(json.dumps(publication), encoding="utf-8")
PY_TAMPER_PUBLICATION
: >"${command_log}"
: >"${gh_command_log}"
set +e
partial_publication_output="$(
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory GATEWAY_DIR="${gateway_directory}" deploy-dry-run 2>&1
)"
partial_publication_status=$?
set -e
mv "${release_asset_source}/publication.json.valid" "${release_asset_source}/publication.json"
[[ "${partial_publication_status}" -ne 0 ]]
[[ "${partial_publication_output}" == *"does not have the exact complete-publication attestation"* ]]
[[ ! -s "${command_log}" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]

cp "${release_asset_source}/container.json" "${release_asset_source}/container.json.valid"
cp "${release_asset_source}/manifest.json" "${release_asset_source}/manifest.json.valid"
cp "${release_asset_source}/publication.json" "${release_asset_source}/publication.json.valid"
python3 - "${release_asset_source}/container.json" "${release_asset_source}/manifest.json" <<'PY_TAMPER_DESCRIPTOR'
import hashlib
import json
import pathlib
import sys

descriptor_path = pathlib.Path(sys.argv[1])
manifest_path = pathlib.Path(sys.argv[2])
descriptor = json.loads(descriptor_path.read_text(encoding="utf-8"))
descriptor["platforms"][0]["image_id"] = "sha256:" + ("e" * 64)
descriptor_payload = json.dumps(descriptor).encode()
descriptor_path.write_bytes(descriptor_payload)
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
entry = next(
    item
    for item in manifest["payloads"]
    if item["path"] == "payloads/containers/loopaware/container.json"
)
entry["size"] = len(descriptor_payload)
entry["sha256"] = hashlib.sha256(descriptor_payload).hexdigest()
manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
publication_path = manifest_path.parent / "publication.json"
publication = json.loads(publication_path.read_text(encoding="utf-8"))
publication["release_manifest_sha256"] = hashlib.sha256(manifest_path.read_bytes()).hexdigest()
publication_path.write_text(json.dumps(publication), encoding="utf-8")
PY_TAMPER_DESCRIPTOR
: >"${command_log}"
: >"${gh_command_log}"
set +e
descriptor_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
descriptor_status=$?
set -e
mv "${release_asset_source}/container.json.valid" "${release_asset_source}/container.json"
mv "${release_asset_source}/manifest.json.valid" "${release_asset_source}/manifest.json"
mv "${release_asset_source}/publication.json.valid" "${release_asset_source}/publication.json"
[[ "${descriptor_status}" -ne 0 ]]
[[ "${descriptor_output}" == *"does not match prepared descriptor"* ]]
[[ ! -s "${command_log}" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]

assert_rejected_gateway_override() {
  local variable_name="$1"
  local variable_value="$2"
  local expected_message="$3"
  local override_output
  local override_status
  : >"${command_log}"
  : >"${gh_command_log}"
  set +e
  override_output="$(
    env \
      PATH="${fake_bin}:${PATH}" \
      COMMAND_LOG="${command_log}" \
      "${variable_name}=${variable_value}" \
      "${real_make}" -C "${fixture_repository}" --no-print-directory \
        GATEWAY_DIR="${gateway_directory}" \
        deploy-dry-run 2>&1
  )"
  override_status=$?
  set -e
  [[ "${override_status}" -ne 0 ]]
  [[ "${override_output}" == *"${expected_message}"* ]]
  [[ ! -s "${command_log}" ]]
  [[ ! -s "${gh_command_log}" ]]
}

assert_rejected_gateway_override \
  MPRLAB_DEPLOY_PREFLIGHT_ONLY \
  1 \
  "MPRLAB_DEPLOY_PREFLIGHT_ONLY is gateway-owned and cannot override the canonical lifecycle"
assert_rejected_gateway_override \
  MPRLAB_LOOPAWARE_IMAGE_REF \
  ghcr.io/example/override@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee \
  "MPRLAB_LOOPAWARE_IMAGE_REF is derived from the published release and cannot be overridden"
assert_rejected_gateway_override \
  MPRLAB_GATEWAY_EXPECTED_COMMIT \
  ffffffffffffffffffffffffffffffffffffffff \
  "MPRLAB_GATEWAY_EXPECTED_COMMIT is derived from the verified gateway checkout and cannot be overridden"

: >"${command_log}"
set +e
help_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    DEPLOY_ARGS="--help" \
    deploy-dry-run 2>&1
)"
help_status=$?
set -e
[[ "${help_status}" -ne 0 ]]
[[ "${help_output}" == *"DEPLOY_ARGS is not supported; the canonical deploy lifecycle accepts no raw shell arguments"* ]]
[[ ! -s "${command_log}" ]]

set +e
mutating_dry_run_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    DEPLOY_ARGS="--dry-run" \
    deploy 2>&1
)"
mutating_dry_run_status=$?
set -e
[[ "${mutating_dry_run_status}" -ne 0 ]]
[[ "${mutating_dry_run_output}" == *"DEPLOY_ARGS is not supported; the canonical deploy lifecycle accepts no raw shell arguments"* ]]
[[ ! -s "${command_log}" ]]

: >"${command_log}"
set +e
pages_permission_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  FAKE_GH_PERMISSION="WRITE" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
pages_permission_status=$?
set -e
[[ "${pages_permission_status}" -ne 0 ]]
[[ "${pages_permission_output}" == *"requires repository ADMIN permission"* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "1" ]]

: >"${command_log}"
touch "${gateway_directory}/untracked-deploy-input"
set +e
dirty_gateway_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
dirty_gateway_status=$?
set -e
rm "${gateway_directory}/untracked-deploy-input"
[[ "${dirty_gateway_status}" -ne 0 ]]
[[ "${dirty_gateway_output}" == *"mprlab-gateway deployment checkout is dirty"* ]]
[[ ! -s "${command_log}" ]]

git -C "${gateway_directory}" config remote.origin.url "git@github.com:example/not-the-gateway.git"
set +e
wrong_gateway_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
wrong_gateway_status=$?
set -e
[[ "${wrong_gateway_status}" -ne 0 ]]
[[ "${wrong_gateway_output}" == *"origin fetch URL must resolve to the canonical GitHub repository MarcoPoloResearchLab/mprlab-gateway"* ]]
[[ ! -s "${command_log}" ]]
git -C "${gateway_directory}" config remote.origin.url "git@github.com:MarcoPoloResearchLab/mprlab-gateway.git"

cat >"${gateway_directory}/Makefile" <<'EOF_GATEWAY_UNSAFE'
deploy-loopaware-backend:
	@echo unsafe deploy
EOF_GATEWAY_UNSAFE
git -C "${gateway_directory}" add Makefile
git -C "${gateway_directory}" commit -m "Remove preflight-only contract" >/dev/null
git -C "${gateway_directory}" push "${gateway_remote_repository}" master >/dev/null
: >"${command_log}"
set +e
unsafe_gateway_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
unsafe_gateway_status=$?
set -e
[[ "${unsafe_gateway_status}" -ne 0 ]]
[[ "${unsafe_gateway_output}" == *"does not implement the required non-deploying preflight handshake"* ]]
[[ ! -s "${command_log}" ]]
cat >"${gateway_directory}/Makefile" <<'EOF_GATEWAY_SAFE'
deploy-preflight-contract:
	@printf '%s\n' 'mprlab.loopaware-deploy.v2'
test-loopaware-deploy-preflight-contract:
	@true
deploy-loopaware-backend-preflight:
	@true
deploy-loopaware-backend:
	@true
EOF_GATEWAY_SAFE
git -C "${gateway_directory}" add Makefile
git -C "${gateway_directory}" commit -m "Restore preflight-only contract" >/dev/null
git -C "${gateway_directory}" push "${gateway_remote_repository}" master >/dev/null

: >"${command_log}"
set +e
failure_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  FAKE_PAGES_FAIL=1 \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
failure_status=$?
set -e
[[ "${failure_status}" -ne 0 ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "1" ]]
grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir ' "${command_log}"
grep -Fq -- '--verify-only' "${command_log}"
[[ "${failure_output}" != *"Validating the exact LoopAware gateway backend target"* ]]

: >"${command_log}"
set +e
skip_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    DEPLOY_ARGS="--skip-backend; true" \
    deploy 2>&1
)"
skip_status=$?
set -e
[[ "${skip_status}" -ne 0 ]]
[[ "${skip_output}" == *"DEPLOY_ARGS is not supported; the canonical deploy lifecycle accepts no raw shell arguments"* ]]
[[ ! -s "${command_log}" ]]

git -C "${fixture_repository}" tag -a v1.2.4 -m "Ambiguous release tag"
git -C "${fixture_repository}" push "${remote_repository}" refs/tags/v1.2.4 >/dev/null
: >"${command_log}"
: >"${gh_command_log}"
set +e
ambiguous_tag_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
ambiguous_tag_status=$?
set -e
[[ "${ambiguous_tag_status}" -ne 0 ]]
[[ "${ambiguous_tag_output}" == *"expected exactly one v* release tag at HEAD, got 2"* ]]
[[ ! -s "${command_log}" ]]
[[ ! -s "${gh_command_log}" ]]
git -C "${fixture_repository}" push "${remote_repository}" :refs/tags/v1.2.4 >/dev/null
git -C "${fixture_repository}" tag -d v1.2.4 >/dev/null

git -C "${fixture_repository}" commit --allow-empty -m "Move past release tag" >/dev/null
git -C "${fixture_repository}" push "${remote_repository}" master >/dev/null
: >"${command_log}"
set +e
stale_output="$(
  PATH="${fake_bin}:${PATH}" \
  COMMAND_LOG="${command_log}" \
  "${real_make}" -C "${fixture_repository}" --no-print-directory \
    GATEWAY_DIR="${gateway_directory}" \
    deploy-dry-run 2>&1
)"
stale_status=$?
set -e
[[ "${stale_status}" -ne 0 ]]
[[ "${stale_output}" == *"no v* release tag points at HEAD"* ]]
[[ ! -s "${command_log}" ]]

echo "deploy dry-run contract checks passed"
