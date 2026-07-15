#!/usr/bin/env python3
"""Validate LoopAware dependency identities and read-only authenticated canaries."""

from __future__ import annotations

import argparse
import base64
import hashlib
import hmac
import json
import sys
import time
import urllib.error
import urllib.request


def fail(message: str) -> None:
    raise SystemExit(message)


def read_dotenv(label: str, document: str) -> dict[str, str]:
    values: dict[str, str] = {}
    for line_number, raw_line in enumerate(document.splitlines(), 1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            fail(f"{label}:{line_number}: expected KEY=value")
        name, raw_value = line.split("=", 1)
        name = name.strip()
        if not name or not name.replace("_", "A").isalnum() or not name[0].isalpha():
            fail(f"{label}:{line_number}: invalid environment name")
        if name in values:
            fail(f"{label}:{line_number}: duplicate environment name {name}")
        value = raw_value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            value = value[1:-1]
        if "\n" in value or "\r" in value:
            fail(f"{label}:{line_number}: {name} must be a single-line value")
        values[name] = value
    return values


def require(values: dict[str, str], name: str, label: str) -> str:
    value = values.get(name, "").strip()
    if not value:
        fail(f"{label}: missing required {name}")
    return value


def require_equal(
    left: dict[str, str],
    left_name: str,
    left_label: str,
    right: dict[str, str],
    right_name: str,
    right_label: str,
) -> None:
    if not hmac.compare_digest(
        require(left, left_name, left_label),
        require(right, right_name, right_label),
    ):
        fail(f"deployment identity mismatch: {left_label}:{left_name} != {right_label}:{right_name}")


def base64url(payload: bytes) -> str:
    return base64.urlsafe_b64encode(payload).rstrip(b"=").decode("ascii")


def session_token(signing_key: str, tenant_id: str, subject: str) -> str:
    now = int(time.time())
    header = {"alg": "HS256", "typ": "JWT"}
    claims = {
        "iss": "tauth",
        "sub": subject,
        "iat": now - 30,
        "exp": now + 120,
        "tenant_id": tenant_id,
        "user_id": subject,
        "user_email": "loopaware-deploy-canary@mprlab.com",
        "user_display_name": "LoopAware Deploy Canary",
        "user_avatar_url": "",
        "user_roles": ["admin"],
    }
    encoded_header = base64url(json.dumps(header, separators=(",", ":"), sort_keys=True).encode())
    encoded_claims = base64url(json.dumps(claims, separators=(",", ":"), sort_keys=True).encode())
    signing_input = f"{encoded_header}.{encoded_claims}".encode("ascii")
    signature = hmac.new(signing_key.encode(), signing_input, hashlib.sha256).digest()
    return f"{encoded_header}.{encoded_claims}.{base64url(signature)}"


def verify_authenticated_get(url: str, origin: str, cookie_name: str, token: str, label: str) -> None:
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/json",
            "Cookie": f"{cookie_name}={token}",
            "Origin": origin,
            "User-Agent": "loopaware-deploy-canary/1",
        },
        method="GET",
    )
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            status = response.status
            response.read(4096)
    except urllib.error.HTTPError as error:
        status = error.code
        error.read(4096)
    except urllib.error.URLError as error:
        fail(f"{label} canary request failed: {error.reason}")
    if status != 200:
        fail(f"{label} authenticated canary returned HTTP {status}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--contract-only", action="store_true")
    arguments = parser.parse_args()
    payload = json.load(sys.stdin)
    if set(payload) != {"loopaware", "tauth", "pinguin"} or not all(
        isinstance(payload[name], str) for name in payload
    ):
        fail("dependency contract input must contain exact loopaware, tauth, and pinguin dotenv documents")

    loopaware = read_dotenv("loopaware", payload["loopaware"])
    tauth = read_dotenv("tauth", payload["tauth"])
    pinguin = read_dotenv("pinguin", payload["pinguin"])
    require_equal(loopaware, "TAUTH_TENANT_ID", "loopaware", tauth, "TAUTH_TENANT_ID_LOOPAWARE", "tauth")
    require_equal(loopaware, "TAUTH_JWT_SIGNING_KEY", "loopaware", tauth, "TAUTH_JWT_SIGNING_KEY_LOOPAWARE", "tauth")
    require_equal(loopaware, "TAUTH_SESSION_COOKIE_NAME", "loopaware", tauth, "TAUTH_SESSION_COOKIE_NAME_LOOPAWARE", "tauth")
    require_equal(loopaware, "PINGUIN_AUTH_TOKEN", "loopaware", pinguin, "GRPC_AUTH_TOKEN", "pinguin")
    require_equal(loopaware, "PINGUIN_TENANT_ID", "loopaware", pinguin, "PINGUIN_TENANT_ID_LA", "pinguin")

    if arguments.contract_only:
        print("LoopAware dependency identity contract passed.")
        return

    verify_authenticated_get(
        "https://tauth-api.mprlab.com/me",
        "https://loopaware.mprlab.com",
        require(tauth, "TAUTH_SESSION_COOKIE_NAME_LOOPAWARE", "tauth"),
        session_token(
            require(tauth, "TAUTH_JWT_SIGNING_KEY_LOOPAWARE", "tauth"),
            "loopaware",
            "loopaware-deploy-tauth-canary",
        ),
        "TAuth LoopAware tenant",
    )
    verify_authenticated_get(
        "https://pinguin-api.mprlab.com/api/tenants",
        "https://pinguin.mprlab.com",
        require(pinguin, "TAUTH_COOKIE_NAME", "pinguin"),
        session_token(
            require(pinguin, "TAUTH_SIGNING_KEY", "pinguin"),
            "pinguin",
            "loopaware-deploy-pinguin-canary",
        ),
        "Pinguin",
    )
    print("LoopAware dependency identities and authenticated canaries passed.")


if __name__ == "__main__":
    main()
