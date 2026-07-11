#!/usr/bin/env bash
set -euo pipefail

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
if [[ -f "${env_file}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${env_file}"
  set +a
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

if [[ -v CLIENT_REACT_NATIVE_NPM ]] && [[ -n "${CLIENT_REACT_NATIVE_NPM}" ]]; then
  npm_command="${CLIENT_REACT_NATIVE_NPM}"
else
  npm_command="npm"
fi

helper="${repo_root}/scripts/release/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper not found: ${helper}" >&2; exit 1; }
"${helper}" verify-release-artifact >/dev/null

manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
readarray -t package_values < <(python3 - "${manifest_path}" <<'PY'
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
)
tarball="${package_values[0]}"

package_json="$(tar -xOf "${tarball}" package/package.json)"
readarray -t identity < <(python3 - "${package_json}" <<'PY'
import json
import sys

package = json.loads(sys.argv[1])
print(package["name"])
print(package["version"])
PY
)
package_name="${identity[0]}"
package_version="${identity[1]}"
package_spec="${package_name}@${package_version}"
prepared_integrity="$(node - "${tarball}" <<'NODE'
const fs = require("node:fs");
const crypto = require("node:crypto");
const archive = fs.readFileSync(process.argv[2]);
process.stdout.write(`sha512-${crypto.createHash("sha512").update(archive).digest("base64")}`);
NODE
)"

published_json="$(${npm_command} view "${package_spec}" dist.integrity --json 2>/dev/null || true)"
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
if [[ -n "${published_integrity}" ]]; then
  [[ "${published_integrity}" == "${prepared_integrity}" ]] || {
    echo "error: ${package_spec} already exists with different content" >&2
    exit 1
  }
  echo "${package_spec} is already published with the prepared integrity."
  exit 0
fi

publish_args=()
if [[ -v CLIENT_REACT_NATIVE_PUBLISH_ARGS ]] && [[ -n "${CLIENT_REACT_NATIVE_PUBLISH_ARGS}" ]]; then
  read -r -a publish_args <<<"${CLIENT_REACT_NATIVE_PUBLISH_ARGS}"
fi
echo "==> [publish] Publishing prepared ${package_spec}"
"${npm_command}" publish "${tarball}" "${publish_args[@]}"
