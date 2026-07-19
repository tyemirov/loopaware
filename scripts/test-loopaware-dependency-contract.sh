#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

canonical_input="${temporary_directory}/canonical.json"
mismatched_input="${temporary_directory}/mismatched.json"
missing_cookie_input="${temporary_directory}/missing-cookie.json"
missing_tenant_input="${temporary_directory}/missing-tenant.json"

python3 - "${canonical_input}" "${mismatched_input}" "${missing_cookie_input}" "${missing_tenant_input}" <<'PY_FIXTURES'
import json
import pathlib
import sys

loopaware = """TAUTH_TENANT_ID=loopaware
TAUTH_JWT_SIGNING_KEY=fixture-loopaware-signing-key
TAUTH_SESSION_COOKIE_NAME=app_session_loopaware
PINGUIN_AUTH_TOKEN=fixture-pinguin-token
PINGUIN_TENANT_ID=loopaware
"""
tauth = "TAUTH_JWT_SIGNING_KEY_LOOPAWARE=fixture-loopaware-signing-key\n"
pinguin = """GRPC_AUTH_TOKEN=fixture-pinguin-token
PINGUIN_TENANT_ID_LA=loopaware
"""

fixtures = (
    (sys.argv[1], {"loopaware": loopaware, "tauth": tauth, "pinguin": pinguin}),
    (
        sys.argv[2],
        {
            "loopaware": loopaware,
            "tauth": "TAUTH_JWT_SIGNING_KEY_LOOPAWARE=fixture-other-signing-key\n",
            "pinguin": pinguin,
        },
    ),
    (
        sys.argv[3],
        {
            "loopaware": loopaware.replace("TAUTH_SESSION_COOKIE_NAME=app_session_loopaware\n", ""),
            "tauth": tauth,
            "pinguin": pinguin,
        },
    ),
    (
        sys.argv[4],
        {
            "loopaware": loopaware.replace("TAUTH_TENANT_ID=loopaware\n", ""),
            "tauth": tauth,
            "pinguin": pinguin,
        },
    ),
)
for path, payload in fixtures:
    pathlib.Path(path).write_text(json.dumps(payload), encoding="utf-8")
PY_FIXTURES

canonical_output="$(python3 "${repo_root}/scripts/verify-loopaware-dependency-contract.py" --contract-only <"${canonical_input}")"
[[ "${canonical_output}" == "LoopAware dependency identity contract passed." ]]

set +e
mismatch_output="$(python3 "${repo_root}/scripts/verify-loopaware-dependency-contract.py" --contract-only <"${mismatched_input}" 2>&1)"
mismatch_status=$?
set -e
[[ ${mismatch_status} -ne 0 ]]
[[ "${mismatch_output}" == "deployment identity mismatch: loopaware:TAUTH_JWT_SIGNING_KEY != tauth:TAUTH_JWT_SIGNING_KEY_LOOPAWARE" ]]
[[ "${mismatch_output}" != *"fixture-loopaware-signing-key"* ]]
[[ "${mismatch_output}" != *"fixture-other-signing-key"* ]]

set +e
missing_cookie_output="$(python3 "${repo_root}/scripts/verify-loopaware-dependency-contract.py" --contract-only <"${missing_cookie_input}" 2>&1)"
missing_cookie_status=$?
set -e
[[ ${missing_cookie_status} -ne 0 ]]
[[ "${missing_cookie_output}" == "loopaware: missing required TAUTH_SESSION_COOKIE_NAME" ]]

set +e
missing_tenant_output="$(python3 "${repo_root}/scripts/verify-loopaware-dependency-contract.py" --contract-only <"${missing_tenant_input}" 2>&1)"
missing_tenant_status=$?
set -e
[[ ${missing_tenant_status} -ne 0 ]]
[[ "${missing_tenant_output}" == "loopaware: missing required TAUTH_TENANT_ID" ]]

printf '%s\n' 'LoopAware dependency contract checks passed.'
