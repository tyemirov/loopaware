#!/usr/bin/env python3
"""Verify the exact canonical LoopAware payload set before release mutation."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import io
import json
import pathlib
import subprocess
import tarfile
from typing import Any


CANONICAL_IMAGE = "ghcr.io/tyemirov/loopaware"
CANONICAL_IMAGE_SOURCE = "https://github.com/tyemirov/loopaware"
CANONICAL_NPM_NAME = "@loopaware/react-native"
CANONICAL_NPM_PUBLISH_CONFIG = {
    "access": "public",
    "registry": "https://registry.npmjs.org/",
}
CANONICAL_PAGES_DOMAIN = "loopaware.mprlab.com"
CANONICAL_MOBILE_RUNTIME = {
    "apiBaseUrl": "https://loopaware-api.mprlab.com",
    "tauthBaseUrl": "https://tauth-api.mprlab.com",
    "tauthTenantId": "loopaware",
}
CANONICAL_IOS_REDIRECT_SCHEME = "com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg"


def fail(message: str) -> None:
    raise SystemExit(f"staged release artifact verification failed: {message}")


def read_json(path: pathlib.Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        fail(f"{path} is not valid JSON: {error}")
    if not isinstance(value, dict):
        fail(f"{path} must contain a JSON object")
    return value


def sha256(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def require_file(path: pathlib.Path) -> pathlib.Path:
    if not path.is_file() or path.is_symlink():
        fail(f"required payload is missing or unsafe: {path}")
    if path.stat().st_size <= 0:
        fail(f"required payload is empty: {path}")
    return path


def require_manifest_path(value: Any, expected: pathlib.Path, label: str) -> None:
    if not isinstance(value, str) or pathlib.Path(value).resolve() != expected.resolve():
        fail(f"{label} does not identify {expected}")


def parse_timestamp(value: Any, label: str) -> dt.datetime:
    if not isinstance(value, str) or not value:
        fail(f"{label} is empty")
    try:
        parsed = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        fail(f"{label} is not an ISO-8601 timestamp: {value}")
    if parsed.tzinfo is None:
        fail(f"{label} must include a timezone: {value}")
    return parsed


def verify_mobile_payloads(asset_root: pathlib.Path, release_timestamp: str) -> None:
    ios_ipa = require_file(asset_root / "loopaware-ios.ipa")
    ios_manifest = read_json(require_file(asset_root / "loopaware-ios.json"))
    if ios_manifest.get("schema") != "loopaware.mobile-ios-archive.v1" or ios_manifest.get("status") != "passed":
        fail("iOS build manifest is not a passed loopaware.mobile-ios-archive.v1 artifact")
    if (ios_manifest.get("app") or {}).get("bundleIdentifier") != "com.mprlab.loopaware":
        fail("iOS build manifest has the wrong bundle identifier")
    expected_ios_runtime = {**CANONICAL_MOBILE_RUNTIME, "iosRedirectSchemes": [CANONICAL_IOS_REDIRECT_SCHEME]}
    if ios_manifest.get("runtimeConfig") != expected_ios_runtime:
        fail("iOS build manifest has a noncanonical runtime configuration")
    if ios_manifest.get("signing") != {"developmentTeam": "Z9ZW6HDGML", "style": "automatic"}:
        fail("iOS build manifest has a noncanonical signing configuration")
    ios_payload = ios_manifest.get("ipa") or {}
    require_manifest_path(ios_payload.get("path"), ios_ipa, "iOS build manifest IPA path")
    if ios_payload.get("sha256") != sha256(ios_ipa) or ios_payload.get("sizeBytes") != ios_ipa.stat().st_size:
        fail("iOS IPA does not match its build manifest")

    android_aab = require_file(asset_root / "loopaware-android.aab")
    android_mapping = require_file(asset_root / "loopaware-android-mapping.txt")
    android_manifest = read_json(require_file(asset_root / "loopaware-android.json"))
    if android_manifest.get("schema") != "loopaware.mobile-android-bundle.v1" or android_manifest.get("status") != "passed":
        fail("Android build manifest is not a passed loopaware.mobile-android-bundle.v1 artifact")
    if android_manifest.get("androidPackage") != "com.mprlab.loopaware":
        fail("Android build manifest has the wrong package identifier")
    if android_manifest.get("runtimeConfig") != CANONICAL_MOBILE_RUNTIME:
        fail("Android build manifest has a noncanonical runtime configuration")
    require_manifest_path(android_manifest.get("output"), android_aab, "Android build manifest bundle path")
    require_manifest_path(android_manifest.get("deobfuscationFile"), android_mapping, "Android build manifest mapping path")
    if android_manifest.get("sha256") != sha256(android_aab) or android_manifest.get("sizeBytes") != android_aab.stat().st_size:
        fail("Android App Bundle does not match its build manifest")
    if android_manifest.get("deobfuscationSha256") != sha256(android_mapping):
        fail("Android mapping file does not match its build manifest")
    required_android_results = {
        "zipIntegrity": "passed",
        "jarSignature": "passed",
        "releaseSigner": "passed",
        "bundletoolValidated": True,
        "r8Minification": "enabled",
    }
    for field, expected in required_android_results.items():
        if android_manifest.get(field) != expected:
            fail(f"Android build manifest {field} is {android_manifest.get(field)!r}, expected {expected!r}")
    if ios_manifest.get("versioning") != android_manifest.get("versioning"):
        fail("iOS and Android artifacts do not share one release versioning identity")
    mobile_timestamp = (ios_manifest.get("versioning") or {}).get("releaseTimestamp")
    if parse_timestamp(mobile_timestamp, "mobile releaseTimestamp") != parse_timestamp(release_timestamp, "staging release_timestamp"):
        fail("mobile artifacts do not use the staging release timestamp")


def safe_tar_files(archive_path: pathlib.Path) -> dict[str, bytes]:
    files: dict[str, bytes] = {}
    with tarfile.open(archive_path, "r:*") as archive:
        for member in archive.getmembers():
            raw_name = member.name
            while raw_name.startswith("./"):
                raw_name = raw_name[2:]
            if raw_name in ("", "."):
                continue
            path = pathlib.PurePosixPath(raw_name)
            if path.is_absolute() or ".." in path.parts or member.issym() or member.islnk():
                fail(f"unsafe archive member in {archive_path}: {member.name}")
            if member.isdir():
                continue
            if not member.isfile():
                fail(f"unsupported archive member in {archive_path}: {member.name}")
            extracted = archive.extractfile(member)
            if extracted is None:
                fail(f"archive member cannot be read in {archive_path}: {member.name}")
            if raw_name in files:
                fail(f"duplicate archive member in {archive_path}: {raw_name}")
            files[raw_name] = extracted.read()
    return files


def verify_react_native_payload(package_path: pathlib.Path) -> None:
    files = safe_tar_files(package_path)
    package_json_bytes = files.get("package/package.json")
    if package_json_bytes is None:
        fail("React Native package has no package/package.json")
    package = json.loads(package_json_bytes)
    if package.get("name") != CANONICAL_NPM_NAME:
        fail(f"React Native package name is not {CANONICAL_NPM_NAME}")
    if package.get("publishConfig") != CANONICAL_NPM_PUBLISH_CONFIG:
        fail("React Native package publishConfig is not the canonical public npm registry contract")
    if not isinstance(package.get("version"), str) or not package["version"]:
        fail("React Native package version is empty")


def verify_container_payloads(container_root: pathlib.Path, version: str, source_commit: str) -> None:
    descriptor = read_json(require_file(container_root / "container.json"))
    if descriptor.get("schema_version") != 1 or descriptor.get("artifact_kind") != "mprlab.container":
        fail("container descriptor has an invalid contract")
    if descriptor.get("name") != "loopaware" or descriptor.get("image") != CANONICAL_IMAGE:
        fail("container descriptor has a noncanonical name or image")
    if descriptor.get("version") != version:
        fail("container descriptor version does not match the release")
    platforms = descriptor.get("platforms")
    if not isinstance(platforms, list) or len(platforms) != 1:
        fail("container descriptor must contain exactly one platform")
    platform = platforms[0]
    expected_archive_path = "payloads/containers/loopaware/linux-amd64.tar"
    expected_local_ref = f"mprlab-release.local/loopaware:{version}-linux-amd64"
    expected_fields = {
        "platform": "linux/amd64",
        "token": "linux-amd64",
        "local_ref": expected_local_ref,
        "archive": expected_archive_path,
    }
    for field, expected in expected_fields.items():
        if platform.get(field) != expected:
            fail(f"container descriptor {field} is {platform.get(field)!r}, expected {expected!r}")
    archive_path = require_file(container_root / "linux-amd64.tar")
    if platform.get("sha256") != sha256(archive_path):
        fail("container archive does not match its descriptor")

    files = safe_tar_files(archive_path)
    manifest_bytes = files.get("manifest.json")
    if manifest_bytes is None:
        fail("container archive has no Docker manifest.json")
    manifest = json.loads(manifest_bytes)
    if not isinstance(manifest, list) or len(manifest) != 1:
        fail("container archive must contain exactly one image")
    image = manifest[0]
    if expected_local_ref not in image.get("RepoTags", []):
        fail("container archive does not contain its canonical local tag")
    config_name = image.get("Config")
    if not isinstance(config_name, str) or config_name not in files:
        fail("container archive image config is missing")
    config_bytes = files[config_name]
    actual_image_id = "sha256:" + hashlib.sha256(config_bytes).hexdigest()
    if platform.get("image_id") != actual_image_id:
        fail("container archive image id does not match its descriptor")
    config = json.loads(config_bytes)
    if f"{config.get('os')}/{config.get('architecture')}" != "linux/amd64":
        fail("container archive image config is not linux/amd64")
    labels = (config.get("config") or {}).get("Labels") or {}
    expected_labels = {
        "org.opencontainers.image.revision": source_commit,
        "org.opencontainers.image.version": version,
        "org.opencontainers.image.source": CANONICAL_IMAGE_SOURCE,
    }
    for field, expected in expected_labels.items():
        if labels.get(field) != expected:
            fail(f"container label {field} is {labels.get(field)!r}, expected {expected!r}")


def git_archive_files(repo_root: pathlib.Path, source_commit: str) -> dict[str, bytes]:
    result = subprocess.run(
        ["git", "archive", f"{source_commit}:web"],
        cwd=repo_root,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if result.returncode != 0:
        fail(f"cannot reconstruct Pages source: {result.stderr.decode(errors='replace').strip()}")
    files: dict[str, bytes] = {}
    with tarfile.open(fileobj=io.BytesIO(result.stdout), mode="r:") as archive:
        for member in archive.getmembers():
            if not member.isfile():
                continue
            path = pathlib.PurePosixPath(member.name)
            if {"tests", "node_modules"}.intersection(path.parts):
                continue
            extracted = archive.extractfile(member)
            if extracted is None:
                fail(f"cannot read reconstructed Pages member {member.name}")
            files[path.as_posix()] = extracted.read()
    return files


def verify_pages_payload(
    archive_path: pathlib.Path,
    repo_root: pathlib.Path,
    version: str,
    source_commit: str,
    release_timestamp: str,
) -> None:
    actual_files = safe_tar_files(archive_path)
    expected_files = git_archive_files(repo_root, source_commit)
    expected_files[".nojekyll"] = b""
    expected_files["CNAME"] = f"{CANONICAL_PAGES_DOMAIN}\n".encode()
    expected_files[".mprlab-release.json"] = (
        json.dumps(
            {
                "schema_version": 1,
                "release_version": version,
                "source_commit": source_commit,
                "release_timestamp": release_timestamp,
            },
            indent=2,
            sort_keys=True,
        )
        + "\n"
    ).encode()
    if actual_files.keys() != expected_files.keys():
        missing = sorted(expected_files.keys() - actual_files.keys())
        extra = sorted(actual_files.keys() - expected_files.keys())
        fail(f"Pages archive inventory differs from source; missing={missing}, extra={extra}")
    changed = sorted(path for path, payload in expected_files.items() if actual_files[path] != payload)
    if changed:
        fail(f"Pages archive content differs from source: {changed}")


def verify(args: argparse.Namespace) -> None:
    artifact_dir = pathlib.Path(args.artifact_dir).resolve()
    repo_root = pathlib.Path(args.repo_root).resolve()
    source_commit = subprocess.run(
        ["git", "rev-parse", "--verify", f"{args.source_commit}^{{commit}}"],
        cwd=repo_root,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.strip()
    staging = read_json(artifact_dir / "staging.json")
    expected_staging = {
        "schema_version": 1,
        "artifact_kind": "mprlab.release.staging",
        "version": args.version,
        "source_commit": source_commit,
    }
    for field, expected in expected_staging.items():
        if staging.get(field) != expected:
            fail(f"staging {field} is {staging.get(field)!r}, expected {expected!r}")
    release_timestamp = staging.get("release_timestamp")
    if not isinstance(release_timestamp, str) or not release_timestamp:
        fail("staging release_timestamp is empty")

    payload_root = artifact_dir / "payloads"
    actual_paths = {
        path.relative_to(artifact_dir).as_posix()
        for path in payload_root.rglob("*")
        if path.is_file() and not path.is_symlink()
    }
    react_native_paths = sorted(
        path for path in actual_paths if pathlib.PurePosixPath(path).match("payloads/release-assets/loopaware-react-native-*.tgz")
    )
    if len(react_native_paths) != 1:
        fail(f"expected exactly one React Native package, got {react_native_paths}")
    expected_paths = {
        "payloads/containers/loopaware/container.json",
        "payloads/containers/loopaware/linux-amd64.tar",
        "payloads/release-assets/loopaware-android-mapping.txt",
        "payloads/release-assets/loopaware-android.aab",
        "payloads/release-assets/loopaware-android.json",
        "payloads/release-assets/loopaware-ios.ipa",
        "payloads/release-assets/loopaware-ios.json",
        "payloads/release-assets/pages.tar.gz",
        react_native_paths[0],
    }
    if actual_paths != expected_paths:
        fail(
            "payload inventory is not the exact canonical nine-file set; "
            f"missing={sorted(expected_paths - actual_paths)}, extra={sorted(actual_paths - expected_paths)}"
        )
    for relative_path in expected_paths:
        require_file(artifact_dir / relative_path)

    asset_root = payload_root / "release-assets"
    verify_mobile_payloads(asset_root, release_timestamp)
    verify_react_native_payload(artifact_dir / react_native_paths[0])
    verify_container_payloads(payload_root / "containers" / "loopaware", args.version, source_commit)
    verify_pages_payload(asset_root / "pages.tar.gz", repo_root, args.version, source_commit, release_timestamp)

    print("Verified exact nine-file staged release artifact contract.")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", required=True)
    parser.add_argument("--repo-root", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--source-commit", required=True)
    verify(parser.parse_args())


if __name__ == "__main__":
    main()
