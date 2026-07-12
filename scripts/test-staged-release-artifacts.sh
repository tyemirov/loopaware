#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

set +e
release_control_output="$(make -C "${repo_root}" --no-print-directory RELEASE_ARGS="--version banana; true" release 2>&1)"
release_control_status=$?
set -e
[[ "${release_control_status}" -ne 0 ]]
[[ "${release_control_output}" == *"RELEASE_ARGS is not supported; the canonical release lifecycle accepts no raw shell arguments"* ]]

set +e
artifact_override_output="$(RELEASE_ARTIFACT_TARGETS="pages-artifact" "${repo_root}/scripts/release/prepare_release.sh" --dry-run 2>&1)"
artifact_override_status=$?
set -e
[[ "${artifact_override_status}" -ne 0 ]]
[[ "${artifact_override_output}" == *"release requires the canonical artifact target set"* ]]

container_identity_repository="${temporary_directory}/container-identity-source"
container_identity_artifact_directory="${temporary_directory}/container-identity-artifacts"
container_identity_bin="${temporary_directory}/container-identity-bin"
container_identity_inspect_id="sha256:1111111111111111111111111111111111111111111111111111111111111111"
mkdir -p "${container_identity_repository}" "${container_identity_artifact_directory}" "${container_identity_bin}"
printf 'FROM scratch\n' >"${container_identity_repository}/Dockerfile"
printf '%s\n' '{"schema_version":1,"artifact_kind":"mprlab.release.staging","version":"v1.2.3","source_commit":"2222222222222222222222222222222222222222","release_timestamp":"2026-07-11T00:00:00+00:00"}' >"${container_identity_artifact_directory}/staging.json"
cat >"${container_identity_bin}/docker" <<'SH_CONTAINER_IDENTITY_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == "context show" ]]; then
  printf '%s\n' fixture
  exit 0
fi
if [[ "$*" == "context inspect fixture --format {{.Endpoints.docker.Host}}" ]]; then
  printf '%s\n' 'unix:///tmp/fixture-docker.sock'
  exit 0
fi
if [[ "$*" == "buildx version" || "$1 $2" == "buildx build" ]]; then
  exit 0
fi
if [[ "$1 $2" == "image inspect" ]]; then
  if [[ "$*" == *"{{.Os}}/{{.Architecture}}"* ]]; then
    printf '%s\n' linux/amd64
  else
    printf '%s\n' "${CONTAINER_IDENTITY_INSPECT_ID}"
  fi
  exit 0
fi
if [[ "$1" == "save" && "$2" == "--output" ]]; then
  python3 - "$3" <<'PY_CONTAINER_IDENTITY_ARCHIVE'
import hashlib
import io
import json
import pathlib
import tarfile
import sys

archive_path = pathlib.Path(sys.argv[1])
config = {
    "os": "linux",
    "architecture": "amd64",
    "config": {
        "Labels": {
            "org.opencontainers.image.revision": "2222222222222222222222222222222222222222",
            "org.opencontainers.image.version": "v1.2.3",
            "org.opencontainers.image.source": "https://github.com/tyemirov/loopaware",
        }
    },
}
config_bytes = json.dumps(config, sort_keys=True, separators=(",", ":")).encode()
config_digest = hashlib.sha256(config_bytes).hexdigest()
manifest_bytes = json.dumps(
    [{"Config": f"{config_digest}.json", "RepoTags": ["mprlab-release.local/loopaware:v1.2.3-linux-amd64"], "Layers": []}]
).encode()
with tarfile.open(archive_path, "w") as archive:
    for name, payload in (("manifest.json", manifest_bytes), (f"{config_digest}.json", config_bytes)):
        info = tarfile.TarInfo(name)
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
PY_CONTAINER_IDENTITY_ARCHIVE
  exit 0
fi
printf 'unexpected docker command: %s\n' "$*" >&2
exit 97
SH_CONTAINER_IDENTITY_DOCKER
chmod +x "${container_identity_bin}/docker"
container_identity_output="$(
  cd "${container_identity_repository}"
  PATH="${container_identity_bin}:${PATH}" CONTAINER_IDENTITY_INSPECT_ID="${container_identity_inspect_id}" \
    RELEASE_VERSION="v1.2.3" RELEASE_ARTIFACT_DIR="${container_identity_artifact_directory}" \
    "${repo_root}/scripts/release/prepare_container_artifact.sh" \
      --name loopaware \
      --image ghcr.io/tyemirov/loopaware \
      --file Dockerfile \
      --context . \
      --platforms linux/amd64
)"
[[ "${container_identity_output}" == *"Prepared container artifact loopaware for linux/amd64."* ]]
python3 - "${repo_root}" "${container_identity_artifact_directory}" "${container_identity_inspect_id}" <<'PY_CONTAINER_IDENTITY_CHECK'
import json
import pathlib
import subprocess
import sys

repo_root = pathlib.Path(sys.argv[1])
artifact_directory = pathlib.Path(sys.argv[2])
inspect_id = sys.argv[3]
descriptor = json.loads((artifact_directory / "payloads/containers/loopaware/container.json").read_text(encoding="utf-8"))
descriptor_id = descriptor["platforms"][0]["image_id"]
archive_id = subprocess.run(
    [
        sys.executable,
        str(repo_root / "scripts/release/container_archive_image_id.py"),
        str(artifact_directory / "payloads/containers/loopaware/linux-amd64.tar"),
    ],
    check=True,
    stdout=subprocess.PIPE,
    text=True,
).stdout.strip()
if descriptor_id != archive_id:
    raise SystemExit(f"descriptor image_id {descriptor_id} does not match archive identity {archive_id}")
if descriptor_id == inspect_id:
    raise SystemExit("descriptor image_id incorrectly used Docker inspect identity")
PY_CONTAINER_IDENTITY_CHECK

fail_fast_repository="${temporary_directory}/mobile-fail-fast-source"
fail_fast_artifact_directory="${temporary_directory}/mobile-fail-fast-artifacts"
fail_fast_bin="${temporary_directory}/mobile-fail-fast-bin"
android_builder_sentinel="${temporary_directory}/android-builder-ran"
mkdir -p "${fail_fast_repository}/mobile" "${fail_fast_artifact_directory}" "${fail_fast_bin}"
printf '{"name":"mobile-fail-fast-fixture","private":true}\n' >"${fail_fast_repository}/mobile/package.json"
git -C "${fail_fast_repository}" init -b master >/dev/null
git -C "${fail_fast_repository}" config user.name "Mobile Artifact Contract"
git -C "${fail_fast_repository}" config user.email "mobile-artifact@mprlab.invalid"
git -C "${fail_fast_repository}" add mobile/package.json
git -C "${fail_fast_repository}" commit -m "Add mobile artifact fixture" >/dev/null
fail_fast_source_commit="$(git -C "${fail_fast_repository}" rev-parse HEAD)"
cat >"${fail_fast_bin}/npm" <<'SH_NPM'
#!/bin/sh
exit 0
SH_NPM
cat >"${fail_fast_bin}/node" <<'SH_NODE'
#!/bin/sh
case "$1" in
  mobile/scripts/build-ios-archive.mjs)
    echo "fixture iOS artifact builder failed" >&2
    exit 23
    ;;
  mobile/scripts/build-android-bundle.mjs)
    : >"${ANDROID_BUILDER_SENTINEL:?}"
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
SH_NODE
chmod +x "${fail_fast_bin}/npm" "${fail_fast_bin}/node"
set +e
fail_fast_output="$(
  cd "${fail_fast_repository}"
  PATH="${fail_fast_bin}:${PATH}" \
    ANDROID_BUILDER_SENTINEL="${android_builder_sentinel}" \
    make -f "${repo_root}/Makefile" --no-print-directory \
      RELEASE_ARTIFACT_DIR="${fail_fast_artifact_directory}" \
      RELEASE_SOURCE_COMMIT="${fail_fast_source_commit}" \
      mobile-release-artifacts 2>&1
)"
fail_fast_status=$?
set -e
[[ "${fail_fast_status}" -ne 0 ]]
[[ "${fail_fast_output}" == *"fixture iOS artifact builder failed"* ]]
[[ ! -e "${android_builder_sentinel}" ]]

fixture_repository="${temporary_directory}/source"
artifact_directory="${temporary_directory}/artifact"
mkdir -p "${fixture_repository}/web" "${artifact_directory}/payloads/release-assets" "${artifact_directory}/payloads/containers/loopaware"
git -C "${fixture_repository}" init -b master >/dev/null
git -C "${fixture_repository}" config user.name "Staged Release Contract"
git -C "${fixture_repository}" config user.email "staged-release@mprlab.invalid"
printf '<!doctype html><title>Fixture</title>\n' >"${fixture_repository}/web/index.html"
git -C "${fixture_repository}" add web/index.html
git -C "${fixture_repository}" commit -m "Add release source" >/dev/null
source_commit="$(git -C "${fixture_repository}" rev-parse HEAD)"
fixture_remote="${temporary_directory}/origin.git"
git init --bare "${fixture_remote}" >/dev/null
git -C "${fixture_repository}" remote add origin "${fixture_remote}"
git -C "${fixture_repository}" push -u origin master >/dev/null
git -C "${fixture_repository}" symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/master
set +e
invalid_version_output="$(
  cd "${fixture_repository}"
  RELEASE_ARTIFACT_TARGETS="mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact" \
    "${repo_root}/scripts/release/prepare_release.sh" --dry-run --version banana 2>&1
)"
invalid_version_status=$?
set -e
[[ "${invalid_version_status}" -ne 0 ]]
[[ "${invalid_version_output}" == *"must use deployable stable vMAJOR.MINOR.PATCH versions"* ]]

python3 - "${fixture_repository}" "${artifact_directory}" "${source_commit}" <<'PY_FIXTURE'
import hashlib
import io
import json
import pathlib
import sys
import tarfile

repo_root = pathlib.Path(sys.argv[1]).resolve()
artifact = pathlib.Path(sys.argv[2]).resolve()
source_commit = sys.argv[3]
assets = artifact / "payloads" / "release-assets"
containers = artifact / "payloads" / "containers" / "loopaware"
version = "v1.2.3"
release_timestamp = "2026-07-11T00:00:00+00:00"

(artifact / "staging.json").write_text(
    json.dumps(
        {
            "schema_version": 1,
            "artifact_kind": "mprlab.release.staging",
            "version": version,
            "source_commit": source_commit,
            "release_timestamp": release_timestamp,
        }
    ),
    encoding="utf-8",
)

ios_ipa = assets / "loopaware-ios.ipa"
android_aab = assets / "loopaware-android.aab"
android_mapping = assets / "loopaware-android-mapping.txt"
ios_ipa.write_bytes(b"fixture-ios-ipa\n")
android_aab.write_bytes(b"fixture-android-aab\n")
android_mapping.write_bytes(b"fixture-android-mapping\n")
digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
versioning = {
    "releaseTimestamp": "2026-07-11T00:00:00.000Z",
    "releaseVersion": "2026.7.11",
    "buildCode": 1,
    "iosBuildNumber": "1",
    "androidVersionCode": 1,
    "buildCodeSource": "fixture",
}
runtime_config = {
    "apiBaseUrl": "https://loopaware-api.mprlab.com",
    "tauthBaseUrl": "https://tauth-api.mprlab.com",
    "tauthTenantId": "loopaware",
}
(assets / "loopaware-ios.json").write_text(
    json.dumps(
        {
            "schema": "loopaware.mobile-ios-archive.v1",
            "status": "passed",
            "app": {"bundleIdentifier": "com.mprlab.loopaware"},
            "versioning": versioning,
            "runtimeConfig": {
                **runtime_config,
                "iosRedirectSchemes": ["com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg"],
            },
            "signing": {"developmentTeam": "Z9ZW6HDGML", "style": "automatic"},
            "ipa": {"path": str(ios_ipa), "sha256": digest(ios_ipa), "sizeBytes": ios_ipa.stat().st_size},
        }
    ),
    encoding="utf-8",
)
(assets / "loopaware-android.json").write_text(
    json.dumps(
        {
            "schema": "loopaware.mobile-android-bundle.v1",
            "status": "passed",
            "androidPackage": "com.mprlab.loopaware",
            "versioning": versioning,
            "runtimeConfig": runtime_config,
            "output": str(android_aab),
            "sha256": digest(android_aab),
            "sizeBytes": android_aab.stat().st_size,
            "deobfuscationFile": str(android_mapping),
            "deobfuscationSha256": digest(android_mapping),
            "zipIntegrity": "passed",
            "jarSignature": "passed",
            "releaseSigner": "passed",
            "bundletoolValidated": True,
            "r8Minification": "enabled",
        }
    ),
    encoding="utf-8",
)

package_json = json.dumps(
    {
        "name": "@loopaware/react-native",
        "version": "0.1.0",
        "publishConfig": {"access": "public", "registry": "https://registry.npmjs.org/"},
    }
).encode()
with tarfile.open(assets / "loopaware-react-native-0.1.0.tgz", "w:gz") as archive:
    info = tarfile.TarInfo("package/package.json")
    info.size = len(package_json)
    archive.addfile(info, io.BytesIO(package_json))

config = {
    "os": "linux",
    "architecture": "amd64",
    "config": {
        "Labels": {
            "org.opencontainers.image.revision": source_commit,
            "org.opencontainers.image.version": version,
            "org.opencontainers.image.source": "https://github.com/tyemirov/loopaware",
        }
    }
}
config_bytes = json.dumps(config, sort_keys=True, separators=(",", ":")).encode()
config_digest = hashlib.sha256(config_bytes).hexdigest()
container_manifest = json.dumps(
    [{"Config": f"{config_digest}.json", "RepoTags": [f"mprlab-release.local/loopaware:{version}-linux-amd64"], "Layers": []}]
).encode()
container_archive = containers / "linux-amd64.tar"
with tarfile.open(container_archive, "w") as archive:
    for name, payload in (("manifest.json", container_manifest), (f"{config_digest}.json", config_bytes)):
        info = tarfile.TarInfo(name)
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
(containers / "container.json").write_text(
    json.dumps(
        {
            "schema_version": 1,
            "artifact_kind": "mprlab.container",
            "name": "loopaware",
            "image": "ghcr.io/tyemirov/loopaware",
            "version": version,
            "platforms": [
                {
                    "platform": "linux/amd64",
                    "token": "linux-amd64",
                    "local_ref": f"mprlab-release.local/loopaware:{version}-linux-amd64",
                    "image_id": f"sha256:{config_digest}",
                    "archive": "payloads/containers/loopaware/linux-amd64.tar",
                    "sha256": digest(container_archive),
                }
            ],
        }
    ),
    encoding="utf-8",
)

marker = json.dumps(
    {
        "schema_version": 1,
        "release_version": version,
        "source_commit": source_commit,
        "release_timestamp": release_timestamp,
    },
    indent=2,
    sort_keys=True,
).encode() + b"\n"
page_files = {
    "index.html": (repo_root / "web" / "index.html").read_bytes(),
    ".nojekyll": b"",
    "CNAME": b"loopaware.mprlab.com\n",
    ".mprlab-release.json": marker,
}
with tarfile.open(assets / "pages.tar.gz", "w:gz") as archive:
    for name, payload in page_files.items():
        info = tarfile.TarInfo(name)
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
PY_FIXTURE

verify=(
  "${repo_root}/scripts/release/verify_staged_artifacts.py"
  --artifact-dir "${artifact_directory}"
  --repo-root "${fixture_repository}"
  --version v1.2.3
  --source-commit "${source_commit}"
)
"${verify[@]}" >/dev/null

ios_manifest="${artifact_directory}/payloads/release-assets/loopaware-ios.json"
android_manifest="${artifact_directory}/payloads/release-assets/loopaware-android.json"
python3 - "${ios_manifest}" "${android_manifest}" <<'PY_TIMESTAMP_DRIFT'
import json
import pathlib
import sys

for raw_path in sys.argv[1:]:
    path = pathlib.Path(raw_path)
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["versioning"]["releaseTimestamp"] = "2026-07-12T00:00:00.000Z"
    path.write_text(json.dumps(payload), encoding="utf-8")
PY_TIMESTAMP_DRIFT
set +e
timestamp_output="$("${verify[@]}" 2>&1)"
timestamp_status=$?
set -e
[[ "${timestamp_status}" -ne 0 ]]
[[ "${timestamp_output}" == *"mobile artifacts do not use the staging release timestamp"* ]]
python3 - "${ios_manifest}" "${android_manifest}" <<'PY_TIMESTAMP_RESTORE'
import json
import pathlib
import sys

for raw_path in sys.argv[1:]:
    path = pathlib.Path(raw_path)
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["versioning"]["releaseTimestamp"] = "2026-07-11T00:00:00.000Z"
    path.write_text(json.dumps(payload), encoding="utf-8")
PY_TIMESTAMP_RESTORE

python3 - "${ios_manifest}" <<'PY_RUNTIME_DRIFT'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["runtimeConfig"]["apiBaseUrl"] = "https://wrong.invalid"
path.write_text(json.dumps(payload), encoding="utf-8")
PY_RUNTIME_DRIFT
set +e
runtime_output="$("${verify[@]}" 2>&1)"
runtime_status=$?
set -e
[[ "${runtime_status}" -ne 0 ]]
[[ "${runtime_output}" == *"iOS build manifest has a noncanonical runtime configuration"* ]]
python3 - "${ios_manifest}" <<'PY_RUNTIME_RESTORE'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["runtimeConfig"]["apiBaseUrl"] = "https://loopaware-api.mprlab.com"
path.write_text(json.dumps(payload), encoding="utf-8")
PY_RUNTIME_RESTORE

mapping="${artifact_directory}/payloads/release-assets/loopaware-android-mapping.txt"
mv "${mapping}" "${mapping}.missing"
set +e
missing_output="$("${verify[@]}" 2>&1)"
missing_status=$?
set -e
mv "${mapping}.missing" "${mapping}"
[[ "${missing_status}" -ne 0 ]]
[[ "${missing_output}" == *"payload inventory is not the exact canonical nine-file set"* ]]

pages_archive="${artifact_directory}/payloads/release-assets/pages.tar.gz"
cp "${pages_archive}" "${temporary_directory}/pages-original.tar.gz"
python3 - "${pages_archive}" <<'PY_TAMPER'
import io
import sys
import tarfile

with tarfile.open(sys.argv[1], "r:gz") as archive:
    files = [(member.name, archive.extractfile(member).read()) for member in archive.getmembers() if member.isfile()]
files.append(("extra.html", b"not from source\n"))
with tarfile.open(sys.argv[1], "w:gz") as archive:
    for name, payload in files:
        info = tarfile.TarInfo(name)
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
PY_TAMPER
set +e
tampered_output="$("${verify[@]}" 2>&1)"
tampered_status=$?
set -e
[[ "${tampered_status}" -ne 0 ]]
[[ "${tampered_output}" == *"Pages archive inventory differs from source"* ]]

cp "${temporary_directory}/pages-original.tar.gz" "${pages_archive}"
"${verify[@]}" >/dev/null
printf '# Changelog\n\n## v1.2.3\n\n- Fixture release.\n' >"${fixture_repository}/CHANGELOG.md"
git -C "${fixture_repository}" add CHANGELOG.md
git -C "${fixture_repository}" commit -m "Release v1.2.3" >/dev/null
release_commit="$(git -C "${fixture_repository}" rev-parse HEAD)"
git -C "${fixture_repository}" tag -a v1.2.3 -m "Release v1.2.3"
notes_file="${temporary_directory}/notes.md"
printf '## v1.2.3\n\n- Fixture release.\n' >"${notes_file}"
(
  cd "${fixture_repository}"
  "${repo_root}/scripts/release/release_helper.py" write-release-artifact \
    --version v1.2.3 \
    --source-commit "${source_commit}" \
    --release-commit "${release_commit}" \
    --notes-file "${notes_file}" \
    --default-branch master \
    --release-timestamp 2026-07-11T00:00:00+00:00 \
    --artifact-dir "${artifact_directory}" >/dev/null
)
prepared_artifact_directory="$(git -C "${fixture_repository}" rev-parse --git-path mprlab-release)"
if [[ "${prepared_artifact_directory}" != /* ]]; then
  prepared_artifact_directory="${fixture_repository}/${prepared_artifact_directory}"
fi
mkdir -p "${prepared_artifact_directory}"
cp -R "${artifact_directory}/." "${prepared_artifact_directory}/"
prepared_output="$({
  cd "${fixture_repository}"
  RELEASE_ARTIFACT_TARGETS="mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact" \
    "${repo_root}/scripts/release/prepare_release.sh" --dry-run
})"
[[ "${prepared_output}" == *"release_already_prepared=true"* ]]
[[ "${prepared_output}" == *"version=v1.2.3"* ]]
[[ "${prepared_output}" != *"next_version=v1.2.4"* ]]

python3 - "${prepared_artifact_directory}/manifest.json" <<'PY_NAIVE_TIMESTAMP'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
manifest = json.loads(path.read_text(encoding="utf-8"))
manifest["release_timestamp"] = "2026-07-11T00:00:00"
path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY_NAIVE_TIMESTAMP
set +e
naive_timestamp_output="$({
  cd "${fixture_repository}"
  RELEASE_ARTIFACT_TARGETS="mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact" \
    "${repo_root}/scripts/release/prepare_release.sh" --dry-run
} 2>&1)"
naive_timestamp_status=$?
set -e
[[ "${naive_timestamp_status}" -ne 0 ]]
[[ "${naive_timestamp_output}" == *"release_timestamp must include a timezone"* ]]

echo "staged release artifact contract checks passed"
