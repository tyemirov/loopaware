#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/deploy.sh [options]

Deploys published LoopAware artifacts. The backend is deployed through
mprlab-gateway, then the published Pages archive is activated locally.

Options:
  --gateway-dir <path>       Gateway checkout. Default: $GATEWAY_DIR or sibling ../mprlab-gateway
  --dry-run                  Validate the exact deploy inputs without production changes
  --help                     Show this help text
USAGE
}

env_or_default() {
  local name="$1"
  local fallback="$2"
  local value=""
  if [[ -v "${name}" ]]; then
    value="${!name}"
  fi
  if [[ -n "${value}" ]]; then
    printf "%s\n" "${value}"
  else
    printf "%s\n" "${fallback}"
  fi
}

GATEWAY_DIR="$(env_or_default GATEWAY_DIR "")"
IMAGE_REPOSITORY="ghcr.io/tyemirov/loopaware"
TAG=""
PAGES_BRANCH="$(env_or_default PAGES_BRANCH gh-pages)"
PAGES_URL="$(env_or_default PAGES_URL https://loopaware.mprlab.com/)"
DRY_RUN="false"
release_artifact_directory=""
gateway_lock_dir=""

cleanup() {
  if [[ -n "${release_artifact_directory}" ]]; then
    rm -rf "${release_artifact_directory}"
  fi
  if [[ -n "${gateway_lock_dir}" ]]; then
    rm -f "${gateway_lock_dir}/owner"
    rmdir "${gateway_lock_dir}"
  fi
}
trap cleanup EXIT

image_digest() {
  local image_ref="$1"
  local inspect_output
  if ! inspect_output="$(docker buildx imagetools inspect "$image_ref" 2>&1)"; then
    echo "error: ${image_ref} is not published in the registry; run make publish from clean master to publish the release Docker image before deploy" >&2
    echo "${inspect_output}" >&2
    exit 1
  fi
  local digest
  digest="$(awk '/^Digest:/ { print $2; exit }' <<<"${inspect_output}")"
  [[ -n "${digest}" ]] || { echo "error: could not resolve digest for ${image_ref}; run make publish before deploy" >&2; exit 1; }
  printf "%s\n" "${digest}"
}

verify_published_image_provenance() {
  local image_ref="$1"
  local expected_version="$2"
  local expected_revision="$3"
  local expected_image_id="$4"
  local raw_index
  raw_index="$(docker buildx imagetools inspect "${image_ref}" --raw)"
  local platform_manifest_digest
  platform_manifest_digest="$(python3 - "${raw_index}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
manifests = document.get("manifests", [])
if not isinstance(manifests, list) or len(manifests) != 1:
    raise SystemExit(f"published image must contain exactly one manifest, got {len(manifests) if isinstance(manifests, list) else 'invalid'}")
matches = [
    manifest
    for manifest in manifests
    if manifest.get("platform", {}).get("os") == "linux"
    and manifest.get("platform", {}).get("architecture") == "amd64"
]
if len(matches) != 1:
    raise SystemExit(f"published image must contain exactly one linux/amd64 manifest, got {len(matches)}")
print(matches[0]["digest"])
PY
)"
  [[ "${platform_manifest_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: published linux/amd64 manifest digest is invalid" >&2; exit 1; }
  local raw_manifest
  raw_manifest="$(docker buildx imagetools inspect "${IMAGE_REPOSITORY}@${platform_manifest_digest}" --raw)"
  python3 - "${raw_manifest}" "${expected_image_id}" <<'PY'
import json
import sys

manifest = json.loads(sys.argv[1])
config_digest = (manifest.get("config") or {}).get("digest")
if config_digest != sys.argv[2]:
    raise SystemExit(f"published image config {config_digest!r} does not match prepared descriptor {sys.argv[2]}")
PY
  local labels_json
  labels_json="$(docker buildx imagetools inspect "${IMAGE_REPOSITORY}@${platform_manifest_digest}" --format '{{json .Image.Config.Labels}}')"
  python3 - "${labels_json}" "${expected_version}" "${expected_revision}" <<'PY'
import json
import sys

labels = json.loads(sys.argv[1])
if not isinstance(labels, dict):
    raise SystemExit("published image has no OCI labels")
expected = {
    "org.opencontainers.image.version": sys.argv[2],
    "org.opencontainers.image.revision": sys.argv[3],
    "org.opencontainers.image.source": "https://github.com/tyemirov/loopaware",
}
for field, value in expected.items():
    if labels.get(field) != value:
        raise SystemExit(f"published image label {field} is {labels.get(field)!r}, expected {value!r}")
PY
}

verify_release_container_descriptor() {
  local manifest_path="$1"
  local descriptor_path="$2"
  local expected_version="$3"
  local expected_release_commit="$4"
  local expected_source_commit="$5"
  python3 - "${manifest_path}" "${descriptor_path}" "${expected_version}" "${expected_release_commit}" "${expected_source_commit}" <<'PY'
import hashlib
import json
import pathlib
import re
import sys

manifest_path = pathlib.Path(sys.argv[1])
descriptor_path = pathlib.Path(sys.argv[2])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
if manifest.get("schema_version") != 2 or manifest.get("artifact_kind") != "mprlab.release":
    raise SystemExit("published release manifest has an invalid contract")
expected = {
    "version": sys.argv[3],
    "release_commit": sys.argv[4],
    "source_commit": sys.argv[5],
    "default_branch": "master",
}

for field, value in expected.items():
    if manifest.get(field) != value:
        raise SystemExit(f"published release manifest {field} is {manifest.get(field)!r}, expected {value!r}")
entry = next(
    (item for item in manifest.get("payloads", []) if item.get("path") == "payloads/containers/loopaware/container.json"),
    None,
)
if entry is None:
    raise SystemExit("published release has no canonical container descriptor")
payload = descriptor_path.read_bytes()
if entry.get("size") != len(payload) or entry.get("sha256") != hashlib.sha256(payload).hexdigest():
    raise SystemExit("published container descriptor does not match the release manifest")
descriptor = json.loads(payload)
if descriptor.get("schema_version") != 1 or descriptor.get("artifact_kind") != "mprlab.container":
    raise SystemExit("published container descriptor has an invalid contract")
if descriptor.get("name") != "loopaware" or descriptor.get("image") != "ghcr.io/tyemirov/loopaware":
    raise SystemExit("published container descriptor has a noncanonical image identity")
if descriptor.get("version") != sys.argv[3]:
    raise SystemExit("published container descriptor has the wrong version")
platforms = descriptor.get("platforms")
if not isinstance(platforms, list) or len(platforms) != 1:
    raise SystemExit("published container descriptor must contain exactly one platform")
platform = platforms[0]
if platform.get("platform") != "linux/amd64" or platform.get("token") != "linux-amd64":
    raise SystemExit("published container descriptor has a noncanonical platform")
image_id = platform.get("image_id")
if not isinstance(image_id, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", image_id):
    raise SystemExit("published container descriptor has an invalid image id")
print(image_id)
PY
}

verify_publication_attestation() {
  local manifest_path="$1"
  local publication_path="$2"
  local expected_version="$3"
  local expected_release_commit="$4"
  local expected_source_commit="$5"
  python3 - "${manifest_path}" "${publication_path}" "${expected_version}" "${expected_release_commit}" "${expected_source_commit}" <<'PY'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
attestation = json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8"))
expected = {
    "schema": "mprlab.loopaware-publication.v1",
    "status": "complete",
    "version": sys.argv[3],
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
if attestation != expected:
    raise SystemExit("published release does not have the exact complete-publication attestation")
PY
}

assert_clean_default_branch() {
  local directory="$1"
  local label="$2"
  local dirty_status
  dirty_status="$(git -C "${directory}" status --short --untracked-files=all)"
  [[ -z "${dirty_status}" ]] || {
    echo "error: ${label} deployment checkout is dirty" >&2
    printf '%s\n' "${dirty_status}" >&2
    exit 1
  }
  local remote_head_refs
  remote_head_refs="$(git -C "${directory}" ls-remote --symref origin HEAD)"
  local remote_default_ref
  local remote_default_sha
  remote_default_ref="$(awk '$1 == "ref:" && $3 == "HEAD" { print $2; exit }' <<<"${remote_head_refs}")"
  remote_default_sha="$(awk '$2 == "HEAD" { print $1; exit }' <<<"${remote_head_refs}")"
  [[ -n "${remote_default_ref}" && -n "${remote_default_sha}" ]] || { echo "error: ${label} origin default branch could not be resolved" >&2; exit 1; }
  local current_branch
  local expected_branch
  local local_head_sha
  current_branch="$(git -C "${directory}" branch --show-current)"
  expected_branch="${remote_default_ref#refs/heads/}"
  [[ "${current_branch}" == "${expected_branch}" ]] || { echo "error: ${label} deployment must run from ${expected_branch}, got ${current_branch:-detached HEAD}" >&2; exit 1; }
  local_head_sha="$(git -C "${directory}" rev-parse HEAD)"
  [[ "${local_head_sha}" == "${remote_default_sha}" ]] || { echo "error: ${label} deployment checkout does not match origin/${expected_branch}" >&2; exit 1; }
  printf '%s\n' "${remote_default_sha}"
}

acquire_gateway_lock() {
  local git_common_dir
  local candidate_lock_dir
  git_common_dir="$(git -C "${GATEWAY_DIR}" rev-parse --git-common-dir)"
  if [[ "${git_common_dir}" != /* ]]; then
    git_common_dir="${GATEWAY_DIR}/${git_common_dir}"
  fi
  candidate_lock_dir="${git_common_dir}/mprlab-loopaware-deploy.lock"
  if ! mkdir "${candidate_lock_dir}" 2>/dev/null; then
    if [[ -d "${candidate_lock_dir}" ]]; then
      echo "error: gateway LoopAware deployment is already locked: ${candidate_lock_dir}" >&2
    else
      echo "error: cannot create gateway deployment lock: ${candidate_lock_dir}" >&2
    fi
    exit 1
  fi
  gateway_lock_dir="${candidate_lock_dir}"
  printf '%s\n' "pid=$$" >"${gateway_lock_dir}/owner"
}

assert_gateway_unchanged() {
  local expected_sha="$1"
  local current_sha
  local current_branch
  local dirty_status
  current_sha="$(git -C "${GATEWAY_DIR}" rev-parse HEAD)"
  current_branch="$(git -C "${GATEWAY_DIR}" branch --show-current)"
  dirty_status="$(git -C "${GATEWAY_DIR}" status --short --untracked-files=all)"
  [[ "${current_sha}" == "${expected_sha}" && "${current_branch}" == "master" && -z "${dirty_status}" ]] || {
    echo "error: gateway checkout changed after deploy preflight began" >&2
    exit 1
  }
}

assert_loopaware_unchanged() {
  local expected_sha="$1"
  local current_sha
  local current_branch
  local dirty_status
  current_sha="$(git -C "${repo_root}" rev-parse HEAD)"
  current_branch="$(git -C "${repo_root}" branch --show-current)"
  dirty_status="$(git -C "${repo_root}" status --short --untracked-files=all)"
  [[ "${current_sha}" == "${expected_sha}" && "${current_branch}" == "master" && -z "${dirty_status}" ]] || {
    echo "error: LoopAware checkout changed after deploy preflight began" >&2
    exit 1
  }
}

[[ -z "${DEPLOY_TAG:-}" ]] || { echo "error: DEPLOY_TAG is not supported; deploy uses the exact release tag at default-branch HEAD" >&2; exit 1; }
[[ -z "${MPRLAB_DEPLOY_PREFLIGHT_ONLY:-}" ]] || { echo "error: MPRLAB_DEPLOY_PREFLIGHT_ONLY is gateway-owned and cannot override the canonical lifecycle" >&2; exit 1; }
[[ -z "${MPRLAB_LOOPAWARE_IMAGE_REF:-}" ]] || { echo "error: MPRLAB_LOOPAWARE_IMAGE_REF is derived from the published release and cannot be overridden" >&2; exit 1; }
[[ -z "${MPRLAB_GATEWAY_EXPECTED_COMMIT:-}" ]] || { echo "error: MPRLAB_GATEWAY_EXPECTED_COMMIT is derived from the verified gateway checkout and cannot be overridden" >&2; exit 1; }
requested_dry_run="false"
requested_help="false"
for argument in "$@"; do
  [[ "${argument}" == "--dry-run" ]] && requested_dry_run="true"
  [[ "${argument}" == "--help" || "${argument}" == "-h" ]] && requested_help="true"
done
[[ "${requested_dry_run}" != "true" || "${requested_help}" != "true" ]] || {
  echo "error: dry-run lifecycle validation cannot be replaced by help output" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --gateway-dir)
      [[ $# -ge 2 ]] || { echo "error: --gateway-dir requires a value" >&2; exit 1; }
      GATEWAY_DIR="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    --skip-backend|--skip-pages|--skip-pages-verify)
      echo "error: partial deploy flags are not supported by the canonical lifecycle" >&2
      exit 1
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

command -v git >/dev/null 2>&1 || { echo "error: git is required" >&2; exit 1; }

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"
source "${repo_root}/scripts/release/repository_identity.sh"
assert_no_github_repository_override
assert_canonical_github_origin "${repo_root}" LoopAware "tyemirov/loopaware"
loopaware_remote_default_sha="$(assert_clean_default_branch "${repo_root}" LoopAware)"

resolve_gateway_dir() {
  if [[ -n "${GATEWAY_DIR}" ]]; then
    printf "%s\n" "${GATEWAY_DIR}"
    return
  fi
  printf "%s\n" "${repo_root}/../mprlab-gateway"
}

GATEWAY_DIR="$(resolve_gateway_dir)"
[[ -n "${GATEWAY_DIR}" ]] || { echo "error: gateway checkout not found; set GATEWAY_DIR=/path/to/mprlab-gateway or pass --gateway-dir" >&2; exit 1; }
[[ -d "${GATEWAY_DIR}" ]] || { echo "error: gateway checkout not found: ${GATEWAY_DIR}" >&2; exit 1; }
git -C "${GATEWAY_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1 || { echo "error: gateway checkout is not a Git worktree: ${GATEWAY_DIR}" >&2; exit 1; }
assert_canonical_github_origin "${GATEWAY_DIR}" mprlab-gateway "MarcoPoloResearchLab/mprlab-gateway"
gateway_commit="$(assert_clean_default_branch "${GATEWAY_DIR}" mprlab-gateway)"
acquire_gateway_lock
assert_gateway_unchanged "${gateway_commit}"
gateway_preflight_contract="$(make -C "${GATEWAY_DIR}" --no-print-directory deploy-preflight-contract 2>/dev/null)" || {
  echo "error: gateway default branch does not implement the required non-deploying preflight handshake" >&2
  exit 1
}
[[ "${gateway_preflight_contract}" == "mprlab.loopaware-deploy.v2" ]] || {
  echo "error: gateway returned an unsupported non-deploying preflight contract: ${gateway_preflight_contract:-<empty>}" >&2
  exit 1
}
make -C "${GATEWAY_DIR}" --no-print-directory test-loopaware-deploy-preflight-contract
assert_gateway_unchanged "${gateway_commit}"
assert_loopaware_unchanged "${loopaware_remote_default_sha}"

head_release_tags=()
while IFS= read -r head_release_tag; do
  [[ -n "${head_release_tag}" ]] || continue
  head_release_tags+=("${head_release_tag}")
done < <(git tag --points-at HEAD --list 'v*' --sort=version:refname)
[[ "${#head_release_tags[@]}" -gt 0 ]] || { echo "error: no v* release tag points at HEAD; run make publish before deploy" >&2; exit 1; }
[[ "${#head_release_tags[@]}" -eq 1 ]] || {
  echo "error: expected exactly one v* release tag at HEAD, got ${#head_release_tags[@]}: ${head_release_tags[*]}" >&2
  exit 1
}
TAG="${head_release_tags[0]}"

if [[ ! "${TAG}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "error: release tag must match vMAJOR.MINOR.PATCH (got: ${TAG})" >&2
  exit 1
fi
git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null || { echo "error: release tag ${TAG} does not exist locally" >&2; exit 1; }
remote_tag_refs="$(git ls-remote --tags origin "refs/tags/${TAG}" "refs/tags/${TAG}^{}")"
remote_tag_sha="$(awk '$2 ~ /\^\{\}$/ { peeled = $1 } $2 !~ /\^\{\}$/ { direct = $1 } END { if (peeled != "") print peeled; else print direct }' <<<"${remote_tag_refs}")"
tag_sha="$(git rev-list -n 1 "${TAG}")"
head_sha="${loopaware_remote_default_sha}"
[[ "${tag_sha}" == "${head_sha}" ]] || { echo "error: release tag ${TAG} does not point at repository HEAD" >&2; exit 1; }
if [[ "${remote_tag_sha}" != "${tag_sha}" ]]; then
  echo "error: release tag ${TAG} is not pushed to origin" >&2
  exit 1
fi
expected_source_commit="$(git rev-parse "${TAG}^{commit}^")"

command -v gh >/dev/null 2>&1 || { echo "error: gh is required for release and Pages verification" >&2; exit 1; }
assert_loopaware_unchanged "${loopaware_remote_default_sha}"
release_artifact_directory="$(mktemp -d)"
gh release download "${TAG}" --repo tyemirov/loopaware \
  --pattern manifest.json \
  --pattern container.json \
  --pattern publication.json \
  --pattern pages.tar.gz \
  --dir "${release_artifact_directory}"
verify_publication_attestation \
  "${release_artifact_directory}/manifest.json" \
  "${release_artifact_directory}/publication.json" \
  "${TAG}" \
  "${tag_sha}" \
  "${expected_source_commit}"
prepared_image_id="$(verify_release_container_descriptor \
  "${release_artifact_directory}/manifest.json" \
  "${release_artifact_directory}/container.json" \
  "${TAG}" \
  "${tag_sha}" \
  "${expected_source_commit}")"

command -v docker >/dev/null 2>&1 || { echo "error: docker is required for image verification" >&2; exit 1; }
docker buildx version >/dev/null 2>&1 || { echo "error: docker buildx is required for image verification" >&2; exit 1; }
echo "==> [deploy] Verifying ${IMAGE_REPOSITORY}:latest matches ${TAG}"
release_digest="$(image_digest "${IMAGE_REPOSITORY}:${TAG}")"
latest_digest="$(image_digest "${IMAGE_REPOSITORY}:latest")"
if [[ "${release_digest}" != "${latest_digest}" ]]; then
  echo "error: ${IMAGE_REPOSITORY}:latest digest ${latest_digest} does not match ${TAG} digest ${release_digest}; run make publish first" >&2
  exit 1
fi
exact_image_ref="${IMAGE_REPOSITORY}@${release_digest}"
verify_published_image_provenance "${exact_image_ref}" "${TAG}" "${expected_source_commit}" "${prepared_image_id}"

echo "==> [deploy] Validating the published Pages artifact for ${TAG}"
assert_loopaware_unchanged "${loopaware_remote_default_sha}"
"${repo_root}/scripts/release/deploy_pages_artifact.sh" \
  --branch "${PAGES_BRANCH}" \
  --url "${PAGES_URL}" \
  --expected-domain "loopaware.mprlab.com" \
  --version "${TAG}" \
  --artifact-dir "${release_artifact_directory}" \
  --verify-only
pages_repository_permission="$(gh repo view tyemirov/loopaware --json viewerPermission --jq .viewerPermission)"
[[ "${pages_repository_permission}" == "ADMIN" ]] || { echo "error: GitHub identity requires repository ADMIN permission for Pages branch and configuration updates" >&2; exit 1; }

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "==> [deploy-dry-run] Validating the exact LoopAware gateway backend target"
  assert_loopaware_unchanged "${loopaware_remote_default_sha}"
  assert_gateway_unchanged "${gateway_commit}"
  timeout --foreground -k 1200s -s SIGKILL 1200s make -C "${GATEWAY_DIR}" MPRLAB_GATEWAY_EXPECTED_COMMIT="${gateway_commit}" MPRLAB_LOOPAWARE_IMAGE_REF="${exact_image_ref}" deploy-loopaware-backend-preflight
else
  echo "==> [deploy] Deploying LoopAware backend through mprlab-gateway"
  assert_loopaware_unchanged "${loopaware_remote_default_sha}"
  assert_gateway_unchanged "${gateway_commit}"
  timeout --foreground -k 1200s -s SIGKILL 1200s make -C "${GATEWAY_DIR}" MPRLAB_GATEWAY_EXPECTED_COMMIT="${gateway_commit}" MPRLAB_LOOPAWARE_IMAGE_REF="${exact_image_ref}" deploy-loopaware-backend
fi

assert_gateway_unchanged "${gateway_commit}"
assert_loopaware_unchanged "${loopaware_remote_default_sha}"

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "LoopAware deploy dry run passed; production hosts were not contacted and production state was not changed."
  exit 0
fi

echo "==> [deploy] Activating the published Pages artifact for ${TAG}"
"${repo_root}/scripts/release/deploy_pages_artifact.sh" \
  --branch "${PAGES_BRANCH}" \
  --url "${PAGES_URL}" \
  --expected-domain "loopaware.mprlab.com" \
  --version "${TAG}" \
  --artifact-dir "${release_artifact_directory}"

echo "LoopAware deploy complete"
