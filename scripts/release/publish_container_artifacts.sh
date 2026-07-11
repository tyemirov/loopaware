#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  publish_container_artifacts.sh [--preflight-only]

Loads container archives prepared by make release, pushes platform images, and
creates the version and latest manifests. It never builds an image.

Options:
  --preflight-only  Verify artifacts, tools, and registry authentication without pushing
USAGE
}

preflight_only="false"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --preflight-only) preflight_only="true"; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/docker_identity.sh"
assert_local_docker_endpoint
command -v gh >/dev/null 2>&1 || { echo "error: gh is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 1; }
docker buildx version >/dev/null 2>&1 || { echo "error: docker buildx is required" >&2; exit 1; }
required_platforms="${PUBLISH_PLATFORMS:-}"
[[ "${required_platforms}" == "linux/amd64" ]] || { echo "error: container publication requires PUBLISH_PLATFORMS=linux/amd64" >&2; exit 1; }

repo_root="$(git rev-parse --show-toplevel)"
export UV_CACHE_DIR="${repo_root}/.cache/uv"
artifact_dir="$(git rev-parse --git-path mprlab-release)"
[[ "${artifact_dir}" == /* ]] || artifact_dir="${repo_root}/${artifact_dir}"
helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/release_helper.py"
"${helper}" verify-release-artifact >/dev/null
release_values_output="$(python3 - "${artifact_dir}/manifest.json" <<'PY'
import json
import sys
manifest = json.load(open(sys.argv[1], encoding="utf-8"))
print(manifest["version"])
print(manifest["source_commit"])
PY
)"
readarray -t release_values <<<"${release_values_output}"
[[ "${#release_values[@]}" -eq 2 ]] || { echo "error: release manifest returned incomplete container provenance" >&2; exit 1; }
release_version="${release_values[0]}"
release_source_commit="${release_values[1]}"
[[ "${release_version}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || { echo "error: container publication requires a stable vMAJOR.MINOR.PATCH release" >&2; exit 1; }
publish_timeout="${PUBLISH_CONTAINER_TIMEOUT_SECONDS:-1200}"
[[ "${publish_timeout}" =~ ^[1-9][0-9]*$ ]] || { echo "error: PUBLISH_CONTAINER_TIMEOUT_SECONDS must be a positive integer" >&2; exit 1; }

descriptor_output="$(find "${artifact_dir}/payloads/containers" -mindepth 2 -maxdepth 2 -name container.json -type f | LC_ALL=C sort)"
descriptors=()
if [[ -n "${descriptor_output}" ]]; then
  mapfile -t descriptors <<<"${descriptor_output}"
fi
[[ "${#descriptors[@]}" -gt 0 ]] || { echo "error: no prepared container artifacts found; run make release" >&2; exit 1; }

verify_ghcr_push_access() {
  local image="$1"
  local registry_username="$2"
  local registry_token="$3"
  local repository_path="${image#ghcr.io/}"
  local response_headers
  local response_body
  response_headers="$(mktemp)"
  response_body="$(mktemp)"
  local create_status
  create_status="$(curl --silent --show-error --config - \
    --dump-header "${response_headers}" \
    --output "${response_body}" \
    --request POST \
    --write-out '%{http_code}' \
    "https://ghcr.io/v2/${repository_path}/blobs/uploads/" <<EOF_CURL
user = "${registry_username}:${registry_token}"
EOF_CURL
)"
  if [[ "${create_status}" != "202" ]]; then
    echo "error: GHCR push-authority preflight failed for ${image} with HTTP ${create_status}: $(head -c 1024 "${response_body}")" >&2
    rm -f "${response_headers}" "${response_body}"
    exit 1
  fi
  local upload_location
  upload_location="$(awk 'tolower($1) == "location:" { sub(/^[^:]*:[[:space:]]*/, ""); sub(/\r$/, ""); print; exit }' "${response_headers}")"
  [[ -n "${upload_location}" ]] || { echo "error: GHCR push-authority preflight returned no upload location for ${image}" >&2; rm -f "${response_headers}" "${response_body}"; exit 1; }
  if [[ "${upload_location}" == /* ]]; then
    upload_location="https://ghcr.io${upload_location}"
  fi
  local delete_status
  delete_status="$(curl --silent --show-error --config - \
    --output "${response_body}" \
    --request DELETE \
    --write-out '%{http_code}' \
    "${upload_location}" <<EOF_CURL
user = "${registry_username}:${registry_token}"
EOF_CURL
)"
  rm -f "${response_headers}" "${response_body}"
  [[ "${delete_status}" == "202" || "${delete_status}" == "204" ]] || { echo "error: GHCR preflight upload cleanup failed for ${image} with HTTP ${delete_status}" >&2; exit 1; }
}

verify_prepared_container_archive() {
  local archive="$1"
  local local_ref="$2"
  local expected_image_id="$3"
  local expected_platform="$4"
  python3 - "${archive}" "${local_ref}" "${expected_image_id}" "${expected_platform}" "${release_version}" "${release_source_commit}" <<'PY'
import hashlib
import json
import sys
import tarfile

archive_path, local_ref, expected_image_id, expected_platform, expected_version, expected_revision = sys.argv[1:]
with tarfile.open(archive_path, "r") as archive:
    try:
        manifest_bytes = archive.extractfile("manifest.json").read()
    except (KeyError, AttributeError):
        raise SystemExit("prepared container archive has no Docker manifest.json")
    manifest = json.loads(manifest_bytes)
    if not isinstance(manifest, list) or len(manifest) != 1:
        raise SystemExit(f"prepared container archive must contain exactly one image, got {len(manifest) if isinstance(manifest, list) else 'invalid'}")
    entry = manifest[0]
    if local_ref not in entry.get("RepoTags", []):
        raise SystemExit(f"prepared container archive does not contain expected tag {local_ref}")
    config_name = entry.get("Config")
    if not isinstance(config_name, str) or not config_name:
        raise SystemExit("prepared container archive has no image config")
    try:
        config_bytes = archive.extractfile(config_name).read()
    except (KeyError, AttributeError):
        raise SystemExit("prepared container archive image config is missing")

actual_image_id = "sha256:" + hashlib.sha256(config_bytes).hexdigest()
if actual_image_id != expected_image_id:
    raise SystemExit(f"prepared container image id {actual_image_id} does not match descriptor {expected_image_id}")
config = json.loads(config_bytes)
actual_platform = f"{config.get('os')}/{config.get('architecture')}"
if actual_platform != expected_platform:
    raise SystemExit(f"prepared container platform {actual_platform} does not match descriptor {expected_platform}")
labels = (config.get("config") or {}).get("Labels") or {}
expected_labels = {
    "org.opencontainers.image.version": expected_version,
    "org.opencontainers.image.revision": expected_revision,
    "org.opencontainers.image.source": "https://github.com/tyemirov/loopaware",
}
for key, expected in expected_labels.items():
    if labels.get(key) != expected:
        raise SystemExit(f"prepared container label {key} is {labels.get(key)!r}, expected {expected!r}")
PY
}

verify_container_archive_loadability() {
  local archive="$1"
  local local_ref="$2"
  local expected_image_id="$3"
  local expected_platform="$4"
  local existing_image_id=""
  existing_image_id="$(docker image inspect "${local_ref}" --format '{{.Id}}' 2>/dev/null || true)"

  restore_local_ref() {
    if [[ -z "${existing_image_id}" ]]; then
      docker image rm --force "${local_ref}" >/dev/null 2>&1 || true
    elif [[ "${existing_image_id}" != "${expected_image_id}" ]]; then
      docker tag "${existing_image_id}" "${local_ref}" >/dev/null
    fi
  }

  local load_output
  if ! load_output="$(timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker load --input "${archive}" 2>&1)"; then
    restore_local_ref
    echo "error: prepared container archive cannot be loaded: ${archive}" >&2
    echo "${load_output}" >&2
    exit 1
  fi
  local loaded_image_id
  loaded_image_id="$(docker image inspect "${local_ref}" --format '{{.Id}}')"
  if [[ "${loaded_image_id}" != "${expected_image_id}" ]]; then
    restore_local_ref
    echo "error: loaded preflight image does not match prepared container descriptor" >&2
    exit 1
  fi
  local loaded_platform
  loaded_platform="$(docker image inspect "${local_ref}" --format '{{.Os}}/{{.Architecture}}')"
  if [[ "${loaded_platform}" != "${expected_platform}" ]]; then
    restore_local_ref
    echo "error: loaded preflight image platform ${loaded_platform} does not match prepared ${expected_platform}" >&2
    exit 1
  fi
  restore_local_ref
}

push_platform_digest=""
push_platform_image() {
  local platform_ref="$1"
  local push_output
  local push_status
  set +e
  push_output="$(timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker push "${platform_ref}" 2>&1)"
  push_status=$?
  set -e
  printf '%s\n' "${push_output}"
  if [[ "${push_status}" -ne 0 ]]; then
    echo "error: container push failed for ${platform_ref} with status ${push_status}" >&2
    return "${push_status}"
  fi
  if ! push_platform_digest="$(awk '
    {
      for (field_index = 1; field_index <= NF; field_index += 1) {
        if ($field_index == "digest:" && $(field_index + 1) ~ /^sha256:[0-9a-f]+$/) {
          if (digest != "" && digest != $(field_index + 1)) {
            exit 2
          }
          digest = $(field_index + 1)
        }
      }
    }
    END {
      if (digest == "") {
        exit 1
      }
      print digest
    }
  ' <<<"${push_output}")"; then
    echo "error: container push returned no unique registry digest for ${platform_ref}" >&2
    return 1
  fi
  [[ "${push_platform_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || {
    echo "error: container push returned an invalid registry digest for ${platform_ref}: ${push_platform_digest}" >&2
    return 1
  }
}

published_linux_amd64_digest() {
  local image_ref="$1"
  local raw_index
  raw_index="$(docker buildx imagetools inspect "${image_ref}" --raw)"
  python3 - "${raw_index}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
manifests = document.get("manifests", [])
if not isinstance(manifests, list) or len(manifests) != 1:
    raise SystemExit(f"published image must contain exactly one manifest, got {len(manifests) if isinstance(manifests, list) else 'invalid'}")
matches = [
    manifest.get("digest")
    for manifest in manifests
    if manifest.get("platform", {}).get("os") == "linux"
    and manifest.get("platform", {}).get("architecture") == "amd64"
]
if len(matches) != 1:
    raise SystemExit(f"published image must contain exactly one linux/amd64 manifest, got {len(matches)}")
digest = matches[0]
if not isinstance(digest, str):
    raise SystemExit("published linux/amd64 manifest has no digest")
print(digest)
PY
}

remote_raw_or_missing() {
  local image_ref="$1"
  local output
  if output="$(docker buildx imagetools inspect "${image_ref}" --raw 2>&1)"; then
    printf '%s\n' "${output}"
    return
  fi
  if grep -Eqi 'manifest unknown|not found|no such manifest' <<<"${output}"; then
    printf '%s\n' '__MISSING__'
    return
  fi
  echo "error: cannot inspect existing container reference ${image_ref}" >&2
  echo "${output}" >&2
  return 2
}

remote_single_manifest_digest() {
  local image_ref="$1"
  local expected_image_id="$2"
  local raw_manifest
  raw_manifest="$(remote_raw_or_missing "${image_ref}")"
  if [[ "${raw_manifest}" == "__MISSING__" ]]; then
    printf '%s\n' '__MISSING__'
    return
  fi
  local manifest_digest
  manifest_digest="$(docker buildx imagetools inspect "${image_ref}" | awk '/^Digest:/ {print $2; exit}')"
  [[ "${manifest_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: existing platform tag ${image_ref} has an invalid digest" >&2; return 2; }
  python3 - "${raw_manifest}" "${expected_image_id}" "${manifest_digest}" <<'PY'
import json
import sys

manifest = json.loads(sys.argv[1])
config_digest = (manifest.get("config") or {}).get("digest")
if config_digest != sys.argv[2]:
    raise SystemExit(f"existing platform tag config {config_digest!r} differs from prepared image {sys.argv[2]}")
print(sys.argv[3])
PY
}

remote_version_platform_digest() {
  local image_ref="$1"
  local expected_image_id="$2"
  local raw_index
  raw_index="$(remote_raw_or_missing "${image_ref}")"
  if [[ "${raw_index}" == "__MISSING__" ]]; then
    printf '%s\n' '__MISSING__'
    return
  fi
  local platform_digest
  platform_digest="$(python3 - "${raw_index}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
manifests = document.get("manifests", [])
if not isinstance(manifests, list) or len(manifests) != 1:
    raise SystemExit(f"existing version index must contain exactly one manifest, got {len(manifests) if isinstance(manifests, list) else 'invalid'}")
matches = [
    manifest.get("digest")
    for manifest in manifests
    if manifest.get("platform", {}).get("os") == "linux"
    and manifest.get("platform", {}).get("architecture") == "amd64"
]
if len(matches) != 1:
    raise SystemExit(f"existing version index must contain exactly one linux/amd64 manifest, got {len(matches)}")
print(matches[0])
PY
)"
  [[ "${platform_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: existing version index ${image_ref} has an invalid platform digest" >&2; return 2; }
  local raw_manifest
  local repository="${image_ref%:*}"
  raw_manifest="$(remote_raw_or_missing "${repository}@${platform_digest}")"
  [[ "${raw_manifest}" != "__MISSING__" ]] || { echo "error: existing version index ${image_ref} references a missing platform manifest" >&2; return 2; }
  python3 - "${raw_manifest}" "${expected_image_id}" "${platform_digest}" <<'PY'
import json
import sys

manifest = json.loads(sys.argv[1])
config_digest = (manifest.get("config") or {}).get("digest")
if config_digest != sys.argv[2]:
    raise SystemExit(f"existing version config {config_digest!r} differs from prepared image {sys.argv[2]}")
print(sys.argv[3])
PY
}

remote_single_platform_index_digest() {
  local image_ref="$1"
  local raw_index
  raw_index="$(remote_raw_or_missing "${image_ref}")"
  if [[ "${raw_index}" == "__MISSING__" ]]; then
    printf '%s\n' '__MISSING__'
    return
  fi
  python3 - "${raw_index}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
manifests = document.get("manifests", [])
if not isinstance(manifests, list) or len(manifests) != 1:
    raise SystemExit(
        f"existing mutable index must contain exactly one manifest, got {len(manifests) if isinstance(manifests, list) else 'invalid'}"
    )
platform = manifests[0].get("platform", {})
if platform.get("os") != "linux" or platform.get("architecture") != "amd64":
    raise SystemExit("existing mutable index must contain exactly one linux/amd64 manifest")
digest = manifests[0].get("digest")
if not isinstance(digest, str):
    raise SystemExit("existing mutable index has no linux/amd64 digest")
print(digest)
PY
}

for descriptor in "${descriptors[@]}"; do
  metadata="$(python3 - "${descriptor}" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
if data.get("schema_version") != 1 or data.get("artifact_kind") != "mprlab.container":
    raise SystemExit("invalid container artifact descriptor")
print(data["name"])
print(data["image"])
print(data["version"])
for platform in data["platforms"]:
    print("\t".join([platform["platform"], platform["token"], platform["local_ref"], platform["image_id"], platform["archive"], platform["sha256"]]))
PY
)"
  name="$(sed -n '1p' <<<"${metadata}")"
  image="$(sed -n '2p' <<<"${metadata}")"
  version="$(sed -n '3p' <<<"${metadata}")"
  [[ "${version}" == "${release_version}" ]] || { echo "error: ${name} was prepared for ${version}, expected ${release_version}" >&2; exit 1; }
  [[ "${image}" == "ghcr.io/tyemirov/loopaware" ]] || { echo "error: ${name} publication image must be ghcr.io/tyemirov/loopaware" >&2; exit 1; }
  validated_platforms=()
  while IFS=$'\t' read -r platform token local_ref expected_image_id archive_relative expected_sha256; do
    [[ -n "${platform}" ]] || continue
    validated_platforms+=("${platform}")
    archive="${artifact_dir}/${archive_relative}"
    actual_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
    [[ "${actual_sha256}" == "${expected_sha256}" ]] || { echo "error: container archive hash mismatch: ${archive_relative}" >&2; exit 1; }
    verify_prepared_container_archive "${archive}" "${local_ref}" "${expected_image_id}" "${platform}"
    if [[ "${preflight_only}" == "true" ]]; then
      verify_container_archive_loadability "${archive}" "${local_ref}" "${expected_image_id}" "${platform}"
    fi
  done < <(tail -n +4 <<<"${metadata}")
  validated_platform_list="$(IFS=,; printf '%s' "${validated_platforms[*]}")"
  [[ "${validated_platform_list}" == "${required_platforms}" ]] || { echo "error: ${name} prepared platforms ${validated_platform_list} do not match required ${required_platforms}" >&2; exit 1; }
done

registry_username=""
registry_token=""
registry_access_verified="false"
if python3 - "${descriptors[@]}" <<'PY'
import json
import sys
raise SystemExit(0 if any(json.load(open(path, encoding="utf-8"))["image"].startswith("ghcr.io/") for path in sys.argv[1:]) else 1)
PY
then
  registry_username="$(gh api user --jq .login)"
  registry_token="$(gh auth token)"
fi

for descriptor in "${descriptors[@]}"; do
  metadata="$(python3 - "${descriptor}" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
if data.get("schema_version") != 1 or data.get("artifact_kind") != "mprlab.container":
    raise SystemExit("invalid container artifact descriptor")
print(data["name"])
print(data["image"])
print(data["version"])
for platform in data["platforms"]:
    print("\t".join([platform["platform"], platform["token"], platform["local_ref"], platform["image_id"], platform["archive"], platform["sha256"]]))
PY
)"
  name="$(sed -n '1p' <<<"${metadata}")"
  image="$(sed -n '2p' <<<"${metadata}")"
  version="$(sed -n '3p' <<<"${metadata}")"
  [[ "${version}" == "${release_version}" ]] || { echo "error: ${name} was prepared for ${version}, expected ${release_version}" >&2; exit 1; }
  if [[ "${image}" == */* && "${image%%/*}" == *.* && "${image}" != ghcr.io/* ]]; then
    echo "error: unsupported explicit container registry for ${image}" >&2
    exit 1
  fi
  sources=()
  actual_platforms=()
  pushed_linux_amd64_digest=""
  canonical_expected_image_id="$(sed -n '4p' <<<"${metadata}" | cut -f4)"
  [[ "${canonical_expected_image_id}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: prepared container descriptor has an invalid image id" >&2; exit 1; }
  existing_version_platform_digest="$(remote_version_platform_digest "${image}:${version}" "${canonical_expected_image_id}")"
  existing_platform_tag_digest="$(remote_single_manifest_digest "${image}:${version}-linux-amd64" "${canonical_expected_image_id}")"
  existing_latest_platform_digest="$(remote_single_platform_index_digest "${image}:latest")"
  if [[ "${existing_latest_platform_digest}" != "__MISSING__" && ! "${existing_latest_platform_digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "error: existing latest index ${image}:latest has an invalid platform digest" >&2
    exit 1
  fi
  if [[ "${existing_version_platform_digest}" != "__MISSING__" ]]; then
    [[ "${existing_platform_tag_digest}" != "__MISSING__" ]] || {
      echo "error: immutable version ${image}:${version} exists but its versioned linux/amd64 tag is missing" >&2
      exit 1
    }
    [[ "${existing_platform_tag_digest}" == "${existing_version_platform_digest}" ]] || {
      echo "error: immutable version ${image}:${version} and ${image}:${version}-linux-amd64 resolve to different platform digests" >&2
      exit 1
    }
  fi

  if [[ "${registry_access_verified}" == "false" ]]; then
    if [[ "${preflight_only}" == "true" ]]; then
      command -v curl >/dev/null 2>&1 || { echo "error: curl is required for GHCR push-authority preflight" >&2; exit 1; }
      verify_ghcr_push_access "${image}" "${registry_username}" "${registry_token}"
    else
      printf '%s' "${registry_token}" | timeout -k 30s -s SIGKILL 30s docker login ghcr.io --username "${registry_username}" --password-stdin
    fi
    registry_access_verified="true"
  fi

  while IFS=$'\t' read -r platform token local_ref expected_image_id archive_relative expected_sha256; do
    [[ -n "${platform}" ]] || continue
    actual_platforms+=("${platform}")
    archive="${artifact_dir}/${archive_relative}"
    actual_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
    [[ "${actual_sha256}" == "${expected_sha256}" ]] || { echo "error: container archive hash mismatch: ${archive_relative}" >&2; exit 1; }
    verify_prepared_container_archive "${archive}" "${local_ref}" "${expected_image_id}" "${platform}"
    if [[ "${preflight_only}" == "true" ]]; then
      sources+=("${platform}")
      continue
    fi
    if [[ "${existing_version_platform_digest}" != "__MISSING__" ]]; then
      sources+=("${image}@${existing_version_platform_digest}")
      pushed_linux_amd64_digest="${existing_version_platform_digest}"
      continue
    fi
    if [[ "${existing_platform_tag_digest}" != "__MISSING__" ]]; then
      sources+=("${image}@${existing_platform_tag_digest}")
      pushed_linux_amd64_digest="${existing_platform_tag_digest}"
      continue
    fi
    timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker load --input "${archive}" >/dev/null
    actual_image_id="$(docker image inspect "${local_ref}" --format '{{.Id}}')"
    [[ "${actual_image_id}" == "${expected_image_id}" ]] || { echo "error: loaded image does not match prepared ${name} ${platform}" >&2; exit 1; }
    actual_loaded_platform="$(docker image inspect "${local_ref}" --format '{{.Os}}/{{.Architecture}}')"
    [[ "${actual_loaded_platform}" == "${platform}" ]] || { echo "error: loaded image platform ${actual_loaded_platform} does not match prepared ${platform}" >&2; exit 1; }
    platform_ref="${image}:${version}-${token}"
    docker tag "${local_ref}" "${platform_ref}"
    echo "==> [publish] Pushing ${platform_ref}"
    push_platform_image "${platform_ref}"
    sources+=("${image}@${push_platform_digest}")
    if [[ "${platform}" == "linux/amd64" ]]; then
      pushed_linux_amd64_digest="${push_platform_digest}"
    fi
  done < <(tail -n +4 <<<"${metadata}")

  [[ "${#sources[@]}" -gt 0 ]] || { echo "error: ${name} has no prepared platforms" >&2; exit 1; }
  actual_platform_list="$(IFS=,; printf '%s' "${actual_platforms[*]}")"
  [[ "${actual_platform_list}" == "${required_platforms}" ]] || { echo "error: ${name} prepared platforms ${actual_platform_list} do not match required ${required_platforms}" >&2; exit 1; }
  if [[ "${preflight_only}" == "true" ]]; then
    echo "Verified prepared container publication inputs for ${name}:${version}."
    continue
  fi
  [[ "${pushed_linux_amd64_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: pushed linux/amd64 digest is missing for ${image}:${version}" >&2; exit 1; }
  published_platform_tag_digest="$(remote_single_manifest_digest "${image}:${version}-linux-amd64" "${canonical_expected_image_id}")"
  [[ "${published_platform_tag_digest}" == "${pushed_linux_amd64_digest}" ]] || {
    echo "error: published ${image}:${version}-linux-amd64 digest ${published_platform_tag_digest} does not match selected digest ${pushed_linux_amd64_digest}" >&2
    exit 1
  }
  if [[ "${existing_version_platform_digest}" == "__MISSING__" ]]; then
    echo "==> [publish] Creating ${image}:${version}"
    timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker buildx imagetools create --tag "${image}:${version}" "${sources[@]}"
  else
    echo "==> [publish] Preserving immutable existing ${image}:${version}"
  fi
  version_digest="$(docker buildx imagetools inspect "${image}:${version}" | awk '/^Digest:/ {print $2; exit}')"
  [[ "${version_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: published version digest is invalid for ${image}:${version}: ${version_digest:-<missing>}" >&2; exit 1; }
  version_platform_digest="$(published_linux_amd64_digest "${image}:${version}")"
  [[ "${version_platform_digest}" == "${pushed_linux_amd64_digest}" ]] || { echo "error: published ${image}:${version} linux/amd64 digest ${version_platform_digest} does not match pushed digest ${pushed_linux_amd64_digest}" >&2; exit 1; }
  echo "==> [publish] Updating ${image}:latest"
  timeout -k "${publish_timeout}s" -s SIGKILL "${publish_timeout}s" docker buildx imagetools create --tag "${image}:latest" "${sources[@]}"
  latest_digest="$(docker buildx imagetools inspect "${image}:latest" | awk '/^Digest:/ {print $2; exit}')"
  [[ "${latest_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "error: published latest digest is invalid for ${image}: ${latest_digest:-<missing>}" >&2; exit 1; }
  [[ "${version_digest}" == "${latest_digest}" ]] || { echo "error: published version and latest digests differ for ${image}" >&2; exit 1; }
  latest_platform_digest="$(published_linux_amd64_digest "${image}:latest")"
  [[ "${latest_platform_digest}" == "${pushed_linux_amd64_digest}" ]] || { echo "error: published ${image}:latest linux/amd64 digest ${latest_platform_digest} does not match pushed digest ${pushed_linux_amd64_digest}" >&2; exit 1; }
  echo "Published ${image}:${version} at ${version_digest}."
done

unset registry_token

if [[ "${preflight_only}" == "true" ]]; then
  echo "Container publication preflight passed; the prepared archive loaded with its exact image id, its temporary local tag was removed or restored, transient empty GHCR upload sessions were deleted, and no image was published."
fi
