#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/deploy.sh [options]

Deploys the LoopAware backend through mprlab-gateway, then publishes GitHub Pages only
after backend verification succeeds.

Options:
  --gateway-dir <path>       Gateway checkout. Default: $GATEWAY_DIR or /Users/tyemirov/Development/mprlab-gateway
  --manifest <path>          App deploy manifest. Default: $APP_MANIFEST or deploy/app.yml
  --tag <value>              Release tag to deploy Pages from. Default: v* tag pointing at HEAD
  --pages-workflow <value>   GitHub Pages workflow file/name. Default: $PAGES_WORKFLOW or pages.yml
  --skip-ci                  Skip the local make ci deployment gate
  --skip-backend             Skip gateway backend deployment
  --skip-pages               Skip Pages workflow dispatch
  --skip-pages-verify        Skip public Pages URL verification
  --pages-url <url>          Pages URL to verify. Default: $PAGES_URL or https://loopaware.mprlab.com/
  --help                     Show this help text
USAGE
}

GATEWAY_DIR="${GATEWAY_DIR:-/Users/tyemirov/Development/mprlab-gateway}"
APP_MANIFEST="${APP_MANIFEST:-deploy/app.yml}"
PAGES_WORKFLOW="${PAGES_WORKFLOW:-pages.yml}"
PAGES_URL="${PAGES_URL:-https://loopaware.mprlab.com/}"
TAG="${DEPLOY_TAG:-}"
SKIP_CI="false"
SKIP_BACKEND="false"
SKIP_PAGES="false"
SKIP_PAGES_VERIFY="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --gateway-dir)
      [[ $# -ge 2 ]] || { echo "error: --gateway-dir requires a value" >&2; exit 1; }
      GATEWAY_DIR="$2"
      shift 2
      ;;
    --manifest)
      [[ $# -ge 2 ]] || { echo "error: --manifest requires a value" >&2; exit 1; }
      APP_MANIFEST="$2"
      shift 2
      ;;
    --tag)
      [[ $# -ge 2 ]] || { echo "error: --tag requires a value" >&2; exit 1; }
      TAG="$2"
      shift 2
      ;;
    --pages-workflow)
      [[ $# -ge 2 ]] || { echo "error: --pages-workflow requires a value" >&2; exit 1; }
      PAGES_WORKFLOW="$2"
      shift 2
      ;;
    --skip-ci)
      SKIP_CI="true"
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
    --pages-url)
      [[ $# -ge 2 ]] || { echo "error: --pages-url requires a value" >&2; exit 1; }
      PAGES_URL="$2"
      shift 2
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

if [[ "${APP_MANIFEST}" != /* ]]; then
  APP_MANIFEST="${repo_root}/${APP_MANIFEST}"
fi
[[ -f "${APP_MANIFEST}" ]] || { echo "error: deploy manifest not found: ${APP_MANIFEST}" >&2; exit 1; }
[[ -d "${GATEWAY_DIR}" ]] || { echo "error: gateway checkout not found: ${GATEWAY_DIR}" >&2; exit 1; }

if [[ -z "${TAG}" ]]; then
  TAG="$(git tag --points-at HEAD --list 'v*' --sort=-version:refname | head -n 1)"
fi

if [[ "${SKIP_PAGES}" != "true" ]]; then
  [[ -n "${TAG}" ]] || { echo "error: no v* release tag points at HEAD; pass --tag or deploy from a release commit" >&2; exit 1; }
  if [[ ! "${TAG}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    echo "error: release tag must match vMAJOR.MINOR.PATCH (got: ${TAG})" >&2
    exit 1
  fi
  command -v gh >/dev/null 2>&1 || { echo "error: gh is required for Pages deployment" >&2; exit 1; }
  git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null || { echo "error: release tag ${TAG} does not exist locally" >&2; exit 1; }
  remote_tag_refs="$(git ls-remote --tags origin "refs/tags/${TAG}" "refs/tags/${TAG}^{}")"
  remote_tag_sha="$(awk '$2 ~ /\^\{\}$/ { peeled = $1 } $2 !~ /\^\{\}$/ { direct = $1 } END { if (peeled != "") print peeled; else print direct }' <<<"${remote_tag_refs}")"
  tag_sha="$(git rev-list -n 1 "${TAG}")"
  if [[ "${remote_tag_sha}" != "${tag_sha}" ]]; then
    echo "error: release tag ${TAG} is not pushed to origin" >&2
    exit 1
  fi
fi

if [[ "${SKIP_CI}" != "true" && ( "${SKIP_BACKEND}" != "true" || "${SKIP_PAGES}" != "true" ) ]]; then
  echo "==> [deploy] Running make ci before deployment"
  timeout -k 1200s -s SIGKILL 1200s make ci
fi

if [[ "${SKIP_BACKEND}" != "true" ]]; then
  echo "==> [deploy] Deploying LoopAware backend through mprlab-gateway"
  timeout -k 1200s -s SIGKILL 1200s make -C "${GATEWAY_DIR}" MPRLAB_APP_MANIFEST="${APP_MANIFEST}" deploy-loopaware
fi

if [[ "${SKIP_PAGES}" != "true" ]]; then
  echo "==> [deploy] Publishing GitHub Pages from ${TAG} after backend verification"
  trigger_started_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  timeout -k 60s -s SIGKILL 60s gh workflow run "${PAGES_WORKFLOW}" --ref "${TAG}"

  run_id=""
  for _ in $(seq 1 40); do
    run_id="$(
      gh run list \
        --workflow "${PAGES_WORKFLOW}" \
        --event workflow_dispatch \
        --json databaseId,headBranch,createdAt \
        --jq ".[] | select(.headBranch == \"${TAG}\" and .createdAt >= \"${trigger_started_at}\") | .databaseId" \
        | head -n 1
    )"
    if [[ -n "${run_id}" ]]; then
      break
    fi
    sleep 3
  done
  [[ -n "${run_id}" ]] || { echo "error: could not find dispatched Pages workflow run for ${TAG}" >&2; exit 1; }
  timeout -k 1200s -s SIGKILL 1200s gh run watch "${run_id}" --exit-status
fi

if [[ "${SKIP_PAGES_VERIFY}" != "true" ]]; then
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for Pages verification" >&2; exit 1; }
  echo "==> [deploy] Verifying ${PAGES_URL}"
  timeout -k 60s -s SIGKILL 60s curl --fail --silent --show-error --location --max-time 30 "${PAGES_URL}" >/dev/null
fi

echo "LoopAware deploy complete"
