#!/usr/bin/env python3
"""Parse the shared LoopAware dotenv as data and emit safe credential exports."""

from __future__ import annotations

import pathlib
import re
import shlex
import sys


EXPORTED_KEYS = {
    "APP_STORE_CONNECT_API_ISSUER_ID",
    "APP_STORE_CONNECT_API_KEY_ID",
    "APP_STORE_CONNECT_API_KEY_PATH",
    "CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE",
    "CLOUDSDK_CONFIG",
    "GOOGLE_APPLICATION_CREDENTIALS",
    "LOOPAWARE_ANDROID_KEYSTORE_PROPERTIES",
    "LOOPAWARE_ANDROID_UPLOAD_STORE_FILE",
    "MOBILE_IOS_ALLOW_PROVISIONING_UPDATES",
    "MOBILE_IOS_ASC_APP_ID",
    "MOBILE_IOS_PROVIDER_PUBLIC_ID",
    "MOBILE_IOS_PROVISIONING_PROFILE",
    "MOBILE_IOS_SIGNING_CERTIFICATE",
    "MOBILE_IOS_SIGNING_KEYCHAIN",
    "MOBILE_IOS_SIGNING_KEYCHAIN_PASSWORD",
    "MOBILE_IOS_SIGNING_KEYCHAIN_PASSWORD_ENV",
    "MOBILE_IOS_SIGNING_KEYCHAIN_PASSWORD_FILE",
    "MOBILE_IOS_SIGNING_STYLE",
    "NODE_AUTH_TOKEN",
    "NPM_API_KEY",
}

APPLICATION_KEYS = {
    "ADMINS",
    "APP_ADDR",
    "DB_DRIVER",
    "DB_DSN",
    "GOOGLE_CLIENT_ID",
    "PINGUIN_ADDR",
    "PINGUIN_AUTH_TOKEN",
    "PINGUIN_TENANT_ID",
    "PUBLIC_BASE_URL",
    "SESSION_SECRET",
    "TAUTH_BASE_URL",
    "TAUTH_JWT_SIGNING_KEY",
    "TAUTH_SESSION_COOKIE_NAME",
    "TAUTH_TENANT_ID",
}

KEY_PATTERN = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")


def fail(path: pathlib.Path, line_number: int, message: str) -> None:
    raise SystemExit(f"release env validation failed: {path}:{line_number}: {message}")


def decode_quoted(path: pathlib.Path, line_number: int, raw: str) -> str:
    quote = raw[0]
    if len(raw) < 2 or raw[-1] != quote:
        fail(path, line_number, "quoted value is not terminated")
    value = raw[1:-1]
    if quote == "'":
        return value
    decoded: list[str] = []
    index = 0
    escapes = {"n": "\n", "r": "\r", "t": "\t", '"': '"', "\\": "\\"}
    while index < len(value):
        character = value[index]
        if character != "\\":
            decoded.append(character)
            index += 1
            continue
        index += 1
        if index >= len(value) or value[index] not in escapes:
            fail(path, line_number, "double-quoted value contains an unsupported escape")
        decoded.append(escapes[value[index]])
        index += 1
    return "".join(decoded)


def parse(path: pathlib.Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for line_number, raw_line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            fail(path, line_number, "expected KEY=VALUE")
        key, raw_value = line.split("=", 1)
        key = key.strip()
        if not KEY_PATTERN.fullmatch(key):
            fail(path, line_number, f"invalid key {key!r}")
        if key in values:
            fail(path, line_number, f"duplicate key {key}")
        if key not in EXPORTED_KEYS and key not in APPLICATION_KEYS:
            fail(path, line_number, f"key {key} is not part of the release env contract")
        raw_value = raw_value.strip()
        if raw_value.startswith(("'", '"')):
            value = decode_quoted(path, line_number, raw_value)
        else:
            value = raw_value
        if "\x00" in value or "\n" in value or "\r" in value:
            fail(path, line_number, f"value for {key} must be one line")
        values[key] = value
    return values


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: parse_release_env.py <env-file>")
    path = pathlib.Path(sys.argv[1]).expanduser().resolve()
    if not path.is_file():
        raise SystemExit(f"release env validation failed: file not found: {path}")
    values = parse(path)
    for key in sorted(EXPORTED_KEYS & values.keys()):
        print(f"export {key}={shlex.quote(values[key])}")


if __name__ == "__main__":
    main()
