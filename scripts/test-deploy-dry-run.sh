#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

fixture_repository="${temporary_directory}/loopaware"
remote_repository="${temporary_directory}/origin.git"
fake_bin="${temporary_directory}/bin"
command_log="${temporary_directory}/commands.log"
gh_command_log="${temporary_directory}/gh-commands.log"
release_asset_source="${temporary_directory}/release-assets"
real_make="$(command -v make)"
real_python3="$(command -v python3)"
become_password_file="${temporary_directory}/gateway-sudo-password"
prompt_log="${temporary_directory}/prompts.log"
mkdir -p \
  "${fixture_repository}/scripts/release" \
  "${fixture_repository}/.mprlab/deploy/ansible/inventory" \
  "${fixture_repository}/.mprlab/deploy/ansible/playbooks" \
  "${fake_bin}" \
  "${release_asset_source}"
printf '%s\n' 'fixture-password' >"${become_password_file}"
chmod 0600 "${become_password_file}"
cp "${repo_root}/scripts/deploy.sh" "${fixture_repository}/scripts/deploy.sh"
cp "${repo_root}/scripts/run-app-ansible-deploy.sh" "${fixture_repository}/scripts/run-app-ansible-deploy.sh"
cp "${repo_root}/scripts/release/repository_identity.sh" "${fixture_repository}/scripts/release/repository_identity.sh"
cp "${repo_root}/scripts/release/with_lifecycle_lock.sh" "${fixture_repository}/scripts/release/with_lifecycle_lock.sh"
cp "${repo_root}/scripts/release/run_lifecycle.sh" "${fixture_repository}/scripts/release/run_lifecycle.sh"
cp "${repo_root}/.mprlab/deploy/ansible/ansible.cfg" "${fixture_repository}/.mprlab/deploy/ansible/ansible.cfg"
cp "${repo_root}/Makefile" "${fixture_repository}/Makefile"

cat >"${fixture_repository}/.mprlab/deploy/ansible/inventory/hosts.yml" <<'YAML_INVENTORY'
---
all:
  children:
    gateway:
      hosts:
        production-fixture:
          ansible_connection: local
YAML_INVENTORY
printf '%s\n' '---' >"${fixture_repository}/.mprlab/deploy/ansible/playbooks/preflight-local.yml"
printf '%s\n' '---' >"${fixture_repository}/.mprlab/deploy/ansible/playbooks/deploy.yml"

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
git -C "${fixture_repository}" add Makefile scripts .mprlab
git -C "${fixture_repository}" commit -m "Add app-owned deploy fixture" >/dev/null
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
if [[ "${1:-}" == "-C" && "${3:-}" == "ls-remote" && "${4:-}" == "--symref" && "${5:-}" == "origin" && "${6:-}" == "HEAD" && "$2" == "${LOOPAWARE_FIXTURE_DIR}" ]]; then
  exec /usr/bin/git ls-remote --symref "${LOOPAWARE_REMOTE_REPOSITORY}" HEAD
fi
if [[ "${1:-}" == "ls-remote" && "${2:-}" == "--tags" && "${3:-}" == "origin" && "${PWD}" == "${LOOPAWARE_FIXTURE_DIR}" ]]; then
  exec /usr/bin/git ls-remote --tags "${LOOPAWARE_REMOTE_REPOSITORY}" "${@:4}"
fi
exec /usr/bin/git "$@"
EOF_GIT

cat >"${fake_bin}/uvx" <<'EOF_UVX'
#!/usr/bin/env bash
set -euo pipefail
printf 'uvx|image=%s|%s\n' "${LOOPAWARE_DEPLOY_IMAGE_REF:-}" "$*" >>"${COMMAND_LOG}"
if [[ "$*" == *"ansible-playbook"* && "$*" == *"preflight-local.yml"* && "${FAKE_MUTATE_LOOPAWARE:-0}" == "1" ]]; then
  touch "${LOOPAWARE_FIXTURE_DIR}/concurrent-change"
fi
if [[ "$*" == *"ansible-inventory"* || "$*" == *"ansible-playbook"* ]]; then
  exit 0
fi
printf 'unexpected uvx command: %s\n' "$*" >&2
exit 97
EOF_UVX

cat >"${fake_bin}/python3" <<'EOF_PYTHON3'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == *'getpass.getpass("Gateway sudo password: ")'* ]]; then
  printf '%s\n' 'prompt|Gateway sudo password:' >>"${PROMPT_LOG}"
  printf '%s\n' 'fixture-password'
  exit 0
fi
exec "${REAL_PYTHON3}" "$@"
EOF_PYTHON3

chmod +x "${fake_bin}/docker" "${fake_bin}/gh" "${fake_bin}/git" "${fake_bin}/python3" "${fake_bin}/uvx"
fixture_repository="$(git -C "${fixture_repository}" rev-parse --show-toplevel)"
export FAKE_RELEASE_ASSETS="${release_asset_source}"
export LOOPAWARE_FIXTURE_DIR="${fixture_repository}"
export LOOPAWARE_REMOTE_REPOSITORY="${remote_repository}"
export GH_COMMAND_LOG="${gh_command_log}"
export PROMPT_LOG="${prompt_log}"
export REAL_PYTHON3="${real_python3}"

run_fixture_lifecycle() {
  local target="$1"
  shift
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE="${become_password_file}" \
    "${real_make}" -C "${fixture_repository}" --no-print-directory "$@" "${target}"
}

: >"${command_log}"
: >"${gh_command_log}"
dry_run_output="$(run_fixture_lifecycle deploy-dry-run)"
[[ "${dry_run_output}" == *"LoopAware app-owned backend preflight passed; production hosts were not contacted and production state was not changed."* ]]
[[ "${dry_run_output}" == *"LoopAware deploy dry run passed; production hosts were not contacted and production state was not changed."* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "3" ]]
sed -n '1p' "${command_log}" | grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir '
sed -n '1p' "${command_log}" | grep -Fq -- '--verify-only'
sed -n '2p' "${command_log}" | grep -Fq 'uvx|image=ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa|--python 3.13 --from ansible-core==2.19.8 ansible-inventory'
sed -n '3p' "${command_log}" | grep -Fq 'ansible-playbook --inventory localhost,'
sed -n '3p' "${command_log}" | grep -Fq 'preflight-local.yml'
! grep -Fq '/.mprlab/deploy/ansible/playbooks/deploy.yml' "${command_log}"
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]

: >"${command_log}"
: >"${gh_command_log}"
deploy_output="$(run_fixture_lifecycle deploy)"
[[ "${deploy_output}" == *"LoopAware deploy complete"* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "5" ]]
sed -n '1p' "${command_log}" | grep -Fq -- '--verify-only'
sed -n '4p' "${command_log}" | grep -Fq "ansible-playbook --become-password-file ${become_password_file} --inventory "
! sed -n '4p' "${command_log}" | grep -Fq -- '--ask-become-pass'
sed -n '4p' "${command_log}" | grep -Fq '/.mprlab/deploy/ansible/playbooks/deploy.yml'
sed -n '5p' "${command_log}" | grep -Fq 'pages|--branch gh-pages --url https://loopaware.mprlab.com/ --expected-domain loopaware.mprlab.com --version v1.2.3 --artifact-dir '
[[ "$(sed -n '5p' "${command_log}")" != *"--verify-only"* ]]

: >"${command_log}"
: >"${prompt_log}"
(
  cd "${fixture_repository}"
  env -u LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE \
    PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" \
    scripts/run-app-ansible-deploy.sh \
    --mode deploy \
    --image-ref ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
)
[[ "$(cat "${prompt_log}")" == 'prompt|Gateway sudo password:' ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "3" ]]
sed -n '3p' "${command_log}" | grep -Fq 'ansible-playbook --become-password-file '
! sed -n '3p' "${command_log}" | grep -Fq -- '--ask-become-pass'
interactive_password_file="$(sed -n '3p' "${command_log}" | sed -n 's/.*--become-password-file \([^ ]*\) --inventory.*/\1/p')"
[[ -n "${interactive_password_file}" ]]
[[ ! -e "${interactive_password_file}" ]]

: >"${command_log}"
set +e
image_override_output="$(LOOPAWARE_DEPLOY_IMAGE_REF=ghcr.io/example/override@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee run_fixture_lifecycle deploy-dry-run 2>&1)"
image_override_status=$?
set -e
[[ "${image_override_status}" -ne 0 ]]
[[ "${image_override_output}" == *"LOOPAWARE_DEPLOY_IMAGE_REF is derived from the published release and cannot be overridden"* ]]
[[ ! -s "${command_log}" ]]

: >"${command_log}"
set +e
partial_output="$(cd "${fixture_repository}" && PATH="${fake_bin}:${PATH}" bash scripts/deploy.sh --skip-backend 2>&1)"
partial_status=$?
set -e
[[ "${partial_status}" -ne 0 ]]
[[ "${partial_output}" == *"partial deploy flags are not supported by the canonical lifecycle"* ]]
[[ ! -s "${command_log}" ]]

: >"${command_log}"
: >"${gh_command_log}"
set +e
pages_failure_output="$(FAKE_PAGES_FAIL=1 run_fixture_lifecycle deploy-dry-run 2>&1)"
pages_failure_status=$?
set -e
[[ "${pages_failure_status}" -ne 0 ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "1" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]

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
partial_publication_output="$(run_fixture_lifecycle deploy-dry-run 2>&1)"
partial_publication_status=$?
set -e
mv "${release_asset_source}/publication.json.valid" "${release_asset_source}/publication.json"
[[ "${partial_publication_status}" -ne 0 ]]
[[ "${partial_publication_output}" == *"does not have the exact complete-publication attestation"* ]]
[[ ! -s "${command_log}" ]]
[[ "$(wc -l <"${gh_command_log}" | tr -d ' ')" == "1" ]]

: >"${command_log}"
set +e
checkout_drift_output="$(FAKE_MUTATE_LOOPAWARE=1 run_fixture_lifecycle deploy-dry-run 2>&1)"
checkout_drift_status=$?
set -e
[[ "${checkout_drift_status}" -ne 0 ]]
[[ "${checkout_drift_output}" == *"LoopAware checkout changed after deploy preflight began"* ]]
grep -Fq 'preflight-local.yml' "${command_log}"
rm "${fixture_repository}/concurrent-change"

: >"${command_log}"
set +e
ansible_override_output="$(cd "${fixture_repository}" && PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" ANSIBLE_CONFIG=/tmp/override scripts/run-app-ansible-deploy.sh --mode dry-run --image-ref ghcr.io/tyemirov/loopaware@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 2>&1)"
ansible_override_status=$?
set -e
[[ "${ansible_override_status}" -ne 0 ]]
[[ "${ansible_override_output}" == *"ANSIBLE_CONFIG is owned by the LoopAware deployment controller"* ]]
[[ ! -s "${command_log}" ]]

if rg -n '(\.\./mprlab-gateway|GATEWAY_DIR|make[[:space:]]+-C[[:space:]].*mprlab-gateway)' "${repo_root}/Makefile" "${repo_root}/scripts/deploy.sh"; then
  echo 'app deploy still depends on mutable gateway source' >&2
  exit 1
fi

echo "deploy dry-run contract checks passed"
