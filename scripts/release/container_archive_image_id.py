#!/usr/bin/env python3
"""Print the canonical config-digest image ID from a Docker archive."""

from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import tarfile


def fail(message: str) -> None:
    raise SystemExit(f"container archive image identity failed: {message}")


def archive_image_id(archive_path: pathlib.Path) -> str:
    try:
        with tarfile.open(archive_path, "r:*") as archive:
            try:
                manifest_file = archive.extractfile("manifest.json")
            except KeyError:
                fail("archive has no Docker manifest.json")
            if manifest_file is None:
                fail("archive manifest cannot be read")
            manifest = json.loads(manifest_file.read())
            if not isinstance(manifest, list) or len(manifest) != 1:
                fail("archive must contain exactly one image")
            config_name = manifest[0].get("Config")
            if not isinstance(config_name, str) or not config_name:
                fail("archive has no image config")
            try:
                config_file = archive.extractfile(config_name)
            except KeyError:
                fail("archive image config is missing")
            if config_file is None:
                fail("archive image config cannot be read")
            config_bytes = config_file.read()
    except (OSError, tarfile.TarError, json.JSONDecodeError) as error:
        fail(str(error))
    return "sha256:" + hashlib.sha256(config_bytes).hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("archive", type=pathlib.Path)
    args = parser.parse_args()
    print(archive_image_id(args.archive))


if __name__ == "__main__":
    main()
