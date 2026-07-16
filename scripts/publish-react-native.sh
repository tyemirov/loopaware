#!/usr/bin/env bash
set -euo pipefail

[[ -z "${NODE_OPTIONS:-}" && -z "${NODE_PATH:-}" ]] || {
  echo "error: NODE_OPTIONS and NODE_PATH are not supported by npm publication" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage:
  publish-react-native.sh [--preflight-only]

Options:
  --preflight-only  Validate the prepared package and npm write authority without publishing
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

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

if [[ -v RELEASE_ENV_FILE ]] && [[ -n "${RELEASE_ENV_FILE}" ]]; then
  env_file="${RELEASE_ENV_FILE}"
else
  env_file="${repo_root}/configs/.env.loopaware"
fi
if [[ "${env_file}" != /* ]]; then
  env_file="${repo_root}/${env_file}"
fi
[[ -f "${env_file}" ]] || { echo "error: React Native publication env file not found: ${env_file}" >&2; exit 1; }
source "${repo_root}/scripts/release/load_release_env.sh"
load_release_env_file "${env_file}"

if [[ -n "${CLIENT_REACT_NATIVE_PUBLISH_ARGS:-}" ]]; then
  echo "error: React Native publication arguments are not part of the canonical lifecycle contract" >&2
  exit 1
fi

if [[ -v NPM_API_KEY ]] && [[ -n "${NPM_API_KEY}" ]]; then
  export NODE_AUTH_TOKEN="${NPM_API_KEY}"
  npmrc_dir="$(mktemp -d)"
  cleanup() {
    rm -rf "${npmrc_dir}"
  }
  trap cleanup EXIT
  echo "//registry.npmjs.org/:_authToken=${NODE_AUTH_TOKEN}" > "${npmrc_dir}/.npmrc"
  export NPM_CONFIG_USERCONFIG="${npmrc_dir}/.npmrc"
fi

npm_command="npm"
npm_registry() {
  env -u NPM_CONFIG_DRY_RUN -u npm_config_dry_run "${npm_command}" "$@"
}

helper="${repo_root}/scripts/release/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper not found: ${helper}" >&2; exit 1; }
"${helper}" verify-release-artifact >/dev/null

manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
package_values_output="$(python3 - "${manifest_path}" <<'PY'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
matches = [
    entry
    for entry in manifest.get("payloads", [])
    if pathlib.PurePosixPath(entry.get("path", "")).match(
        "payloads/release-assets/loopaware-react-native-*.tgz"
    )
]
if len(matches) != 1:
    raise SystemExit("prepared release must contain exactly one React Native package tarball")
entry = matches[0]
tarball = manifest_path.parent / entry["path"]
digest = hashlib.sha256(tarball.read_bytes()).hexdigest()
if digest != entry["sha256"]:
    raise SystemExit("prepared React Native package hash does not match the release manifest")
print(tarball)
PY
)"
readarray -t package_values <<<"${package_values_output}"
[[ "${#package_values[@]}" -eq 1 ]] || { echo "error: prepared React Native package lookup returned incomplete values" >&2; exit 1; }
tarball="${package_values[0]}"

package_json="$(tar -xOf "${tarball}" package/package.json)"
identity_output="$(python3 - "${package_json}" <<'PY'
import json
import sys

package = json.loads(sys.argv[1])
if package.get("name") != "@loopaware/react-native":
    raise SystemExit("prepared React Native package name is not @loopaware/react-native")
if package.get("publishConfig") != {"access": "public", "registry": "https://registry.npmjs.org/"}:
    raise SystemExit("prepared React Native package publishConfig is not the canonical public npm registry contract")
import re
if not isinstance(package.get("version"), str) or not re.fullmatch(r"(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", package["version"]):
    raise SystemExit("prepared React Native package version must be stable MAJOR.MINOR.PATCH")
print(package["name"])
print(package["version"])
PY
)"
readarray -t identity <<<"${identity_output}"
[[ "${#identity[@]}" -eq 2 ]] || { echo "error: prepared React Native package identity is incomplete" >&2; exit 1; }
package_name="${identity[0]}"
package_version="${identity[1]}"
package_spec="${package_name}@${package_version}"
canonical_registry="https://registry.npmjs.org/"
prepared_integrity="$(node - "${tarball}" <<'NODE'
const fs = require("node:fs");
const crypto = require("node:crypto");
const archive = fs.readFileSync(process.argv[2]);
process.stdout.write(`sha512-${crypto.createHash("sha512").update(archive).digest("base64")}`);
NODE
)"

published_error="$(mktemp)"
set +e
published_json="$(npm_registry view "${package_spec}" dist.integrity --json --registry "${canonical_registry}" 2>"${published_error}")"
published_status=$?
set -e
if [[ "${published_status}" -ne 0 ]]; then
  if grep -Eq 'E404|404 Not Found|is not in this registry' "${published_error}"; then
    published_json=""
  else
    echo "error: npm registry lookup failed for ${package_spec}" >&2
    head -c 2048 "${published_error}" >&2
    rm -f "${published_error}"
    exit 1
  fi
fi
rm -f "${published_error}"
published_integrity="$(python3 - "${published_json}" <<'PY'
import json
import sys

raw = sys.argv[1].strip()
if not raw:
    print("")
else:
    value = json.loads(raw)
    print(value if isinstance(value, str) else "")
PY
)"
version_already_published="false"
if [[ -n "${published_integrity}" ]]; then
  [[ "${published_integrity}" == "${prepared_integrity}" ]] || {
    echo "error: ${package_spec} already exists with different content" >&2
    exit 1
  }
  version_already_published="true"
fi

package_error="$(mktemp)"
set +e
package_name_json="$(npm_registry view "${package_name}" name --json --registry "${canonical_registry}" 2>"${package_error}")"
package_status=$?
set -e
if [[ "${package_status}" -ne 0 ]]; then
  if grep -Eq 'E404|404 Not Found|is not in this registry' "${package_error}"; then
    echo "error: ${package_name} must be bootstrapped once before the canonical lifecycle can prove write authority without publishing a version" >&2
  else
    echo "error: npm package lookup failed for ${package_name}" >&2
    head -c 2048 "${package_error}" >&2
  fi
  rm -f "${package_error}"
  exit 1
fi
rm -f "${package_error}"
registered_package_name="$(python3 - "${package_name_json}" <<'PY'
import json
import sys

value = json.loads(sys.argv[1])
print(value if isinstance(value, str) else "")
PY
)"
[[ "${registered_package_name}" == "${package_name}" ]] || {
  echo "error: npm package lookup returned ${registered_package_name:-<empty>}, expected ${package_name}" >&2
  exit 1
}

verify_public_status() {
  local status_json="$1"
  local label="$2"
  python3 - "${status_json}" "${package_name}" "${label}" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
package_name = sys.argv[2]
if not isinstance(payload, dict) or payload.get(package_name) != "public":
    raise SystemExit(f"{sys.argv[3]} returned {payload!r}; expected {package_name} to be public")
PY
}

dist_tags_json="$(npm_registry view "${package_name}" dist-tags --json --registry "${canonical_registry}")"
latest_version="$(python3 - "${dist_tags_json}" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
print(payload.get("latest", "") if isinstance(payload, dict) else "")
PY
)"
[[ -n "${latest_version}" ]] || { echo "error: npm package ${package_name} has no latest dist-tag" >&2; exit 1; }
version_relation="$(python3 - "${package_version}" "${latest_version}" <<'PY'
import re
import sys

pattern = re.compile(r"^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)$")
if not pattern.fullmatch(sys.argv[2]):
    raise SystemExit(f"npm latest version is not stable MAJOR.MINOR.PATCH: {sys.argv[2]}")
prepared = tuple(map(int, sys.argv[1].split(".")))
latest = tuple(map(int, sys.argv[2].split(".")))
print("older" if prepared < latest else "same" if prepared == latest else "newer")
PY
)"
[[ "${version_relation}" != "older" ]] || { echo "error: prepared ${package_spec} is older than npm latest ${latest_version}; refusing to move latest backward" >&2; exit 1; }
if [[ "${version_relation}" == "same" && "${version_already_published}" != "true" ]]; then
  echo "error: npm latest points to missing prepared version ${package_version}" >&2
  exit 1
fi

visibility_before="$(npm_registry access get status "${package_name}" --json --registry "${canonical_registry}")"
verify_public_status "${visibility_before}" "npm visibility preflight"
visibility_write="$(npm_registry access set status=public "${package_name}" --json --registry "${canonical_registry}")"
verify_public_status "${visibility_write}" "npm write-authority preflight"
visibility_after="$(npm_registry access get status "${package_name}" --json --registry "${canonical_registry}")"
verify_public_status "${visibility_after}" "npm visibility verification"

if [[ "${preflight_only}" == "true" ]]; then
  if [[ "${version_already_published}" == "true" ]]; then
    if [[ "${latest_version}" == "${package_version}" ]]; then
      echo "React Native publication preflight passed; ${package_spec} has the prepared integrity, the package is public, and latest already points to ${package_version}."
    else
      echo "React Native publication preflight passed; ${package_spec} has the prepared integrity, the package is public, and make publish will move latest from ${latest_version} to ${package_version}."
    fi
    exit 0
  fi
  echo "==> [publish-preflight] Validating prepared ${package_spec}"
  npm_registry publish "${tarball}" --dry-run --registry "${canonical_registry}" --access public --tag latest >/dev/null
  echo "React Native publication preflight passed; the existing package remained public and no npm version was published."
  exit 0
fi

if [[ "${version_already_published}" == "true" ]]; then
  if [[ "${latest_version}" != "${package_version}" ]]; then
    echo "==> [publish] Updating ${package_name}@latest to ${package_version}"
    npm_registry dist-tag add "${package_spec}" latest --registry "${canonical_registry}"
  fi
else
  echo "==> [publish] Publishing prepared ${package_spec}"
  npm_registry publish "${tarball}" --dry-run=false --registry "${canonical_registry}" --access public --tag latest
fi

verified_integrity_json="$(npm_registry view "${package_spec}" dist.integrity --json --registry "${canonical_registry}")"
verified_integrity="$(python3 - "${verified_integrity_json}" <<'PY'
import json
import sys

value = json.loads(sys.argv[1])
print(value if isinstance(value, str) else "")
PY
)"
[[ "${verified_integrity}" == "${prepared_integrity}" ]] || { echo "error: published ${package_spec} integrity does not match the prepared tarball" >&2; exit 1; }
verified_dist_tags_json="$(npm_registry view "${package_name}" dist-tags --json --registry "${canonical_registry}")"
verified_latest_version="$(python3 - "${verified_dist_tags_json}" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
print(payload.get("latest", "") if isinstance(payload, dict) else "")
PY
)"
[[ "${verified_latest_version}" == "${package_version}" ]] || { echo "error: npm latest points to ${verified_latest_version:-<empty>}, expected ${package_version}" >&2; exit 1; }
verified_visibility="$(npm_registry access get status "${package_name}" --json --registry "${canonical_registry}")"
verify_public_status "${verified_visibility}" "npm post-publication visibility verification"
echo "Published ${package_spec} with matching integrity, public visibility, and latest dist-tag."
