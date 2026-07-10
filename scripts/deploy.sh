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
  --image <value>            Backend image repository. Default: $DOCKER_IMAGE or ghcr.io/tyemirov/loopaware
  --tag <value>              Published release tag. Default: exact v* tag at HEAD
  --skip-image-verify        Skip release tag/latest image digest verification
  --skip-backend             Skip gateway backend deployment
  --skip-pages               Skip app-owned Pages resources
  --skip-pages-verify        Skip public Pages URL verification
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
IMAGE_REPOSITORY="$(env_or_default DOCKER_IMAGE ghcr.io/tyemirov/loopaware)"
TAG="$(env_or_default DEPLOY_TAG "")"
SKIP_IMAGE_VERIFY="false"
SKIP_BACKEND="false"
SKIP_PAGES="false"
SKIP_PAGES_VERIFY="false"

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

while [[ $# -gt 0 ]]; do
  case "$1" in
    --gateway-dir)
      [[ $# -ge 2 ]] || { echo "error: --gateway-dir requires a value" >&2; exit 1; }
      GATEWAY_DIR="$2"
      shift 2
      ;;
    --image)
      [[ $# -ge 2 ]] || { echo "error: --image requires a value" >&2; exit 1; }
      IMAGE_REPOSITORY="$2"
      shift 2
      ;;
    --tag)
      [[ $# -ge 2 ]] || { echo "error: --tag requires a value" >&2; exit 1; }
      TAG="$2"
      shift 2
      ;;
    --skip-image-verify)
      SKIP_IMAGE_VERIFY="true"
      shift
      ;;
    --skip-backend)
      SKIP_BACKEND="true"
      shift
      ;;
    --skip-pages)
      SKIP_PAGES="true"
      shift
      ;;
    --skip-pages-verify)
      SKIP_PAGES_VERIFY="true"
      shift
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

if [[ -z "${TAG}" ]]; then
  TAG="$(git tag --points-at HEAD --list 'v*' --sort=-version:refname | head -n 1)"
fi

if [[ "${SKIP_BACKEND}" != "true" || "${SKIP_PAGES}" != "true" ]]; then
  [[ -n "${TAG}" ]] || { echo "error: no v* release tag points at HEAD; run make publish before deploy" >&2; exit 1; }
  if [[ ! "${TAG}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    echo "error: release tag must match vMAJOR.MINOR.PATCH (got: ${TAG})" >&2
    exit 1
  fi
  git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null || { echo "error: release tag ${TAG} does not exist locally" >&2; exit 1; }
  remote_tag_refs="$(git ls-remote --tags origin "refs/tags/${TAG}" "refs/tags/${TAG}^{}")"
  remote_tag_sha="$(awk '$2 ~ /\^\{\}$/ { peeled = $1 } $2 !~ /\^\{\}$/ { direct = $1 } END { if (peeled != "") print peeled; else print direct }' <<<"${remote_tag_refs}")"
  tag_sha="$(git rev-list -n 1 "${TAG}")"
  if [[ "${remote_tag_sha}" != "${tag_sha}" ]]; then
    echo "error: release tag ${TAG} is not pushed to origin" >&2
    exit 1
  fi
fi

if [[ "${SKIP_IMAGE_VERIFY}" != "true" && "${SKIP_BACKEND}" != "true" ]]; then
  command -v docker >/dev/null 2>&1 || { echo "error: docker is required for image verification" >&2; exit 1; }
  docker buildx version >/dev/null 2>&1 || { echo "error: docker buildx is required for image verification" >&2; exit 1; }
  [[ -n "${TAG}" ]] || { echo "error: no v* release tag points at HEAD; run make publish before deploy" >&2; exit 1; }
  echo "==> [deploy] Verifying ${IMAGE_REPOSITORY}:latest matches ${TAG}"
  release_digest="$(image_digest "${IMAGE_REPOSITORY}:${TAG}")"
  latest_digest="$(image_digest "${IMAGE_REPOSITORY}:latest")"
  if [[ "${release_digest}" != "${latest_digest}" ]]; then
    echo "error: ${IMAGE_REPOSITORY}:latest digest ${latest_digest} does not match ${TAG} digest ${release_digest}; run make publish first" >&2
    exit 1
  fi
fi

if [[ "${SKIP_BACKEND}" != "true" ]]; then
  echo "==> [deploy] Deploying LoopAware backend through mprlab-gateway"
  timeout --foreground -k 1200s -s SIGKILL 1200s make -C "${GATEWAY_DIR}" deploy-loopaware-backend
fi

if [[ "${SKIP_PAGES}" != "true" ]]; then
  pages_args=()
  [[ "${SKIP_PAGES_VERIFY}" == "true" ]] && pages_args+=(--skip-verify)
  echo "==> [deploy] Activating the published Pages artifact for ${TAG}"
  PAGES_VERSION="${TAG}" PAGES_DEPLOY_ARGS="${pages_args[*]}" make --no-print-directory pages-deploy
fi

echo "LoopAware deploy complete"
