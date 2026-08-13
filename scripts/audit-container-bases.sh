#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
repository_root=$(CDPATH= cd -- "${script_dir}/.." && pwd)
dockerfile="${repository_root}/Dockerfile"

build_base='FROM golang:1.26.6-alpine3.24@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df AS build'
runtime_base='FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b'

if ! grep -Fqx "${build_base}" "${dockerfile}"; then
  echo "container_base_audit_failed: Dockerfile build base must use the approved multi-architecture Go 1.26.6/Alpine 3.24 digest" >&2
  exit 1
fi

if ! grep -Fqx "${runtime_base}" "${dockerfile}"; then
  echo "container_base_audit_failed: Dockerfile runtime base must use the approved multi-architecture Alpine 3.24 digest" >&2
  exit 1
fi

from_count=$(grep -Ec '^FROM ' "${dockerfile}")
pinned_count=$(grep -Ec '^FROM [^ ]+@sha256:[0-9a-f]{64}([[:space:]]|$)' "${dockerfile}")
if [ "${from_count}" -ne 2 ] || [ "${pinned_count}" -ne "${from_count}" ]; then
  echo "container_base_audit_failed: every Dockerfile stage must use an immutable sha256 manifest-list digest" >&2
  exit 1
fi
