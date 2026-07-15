#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf '%s\n' 'Usage: scripts/run-app-ansible-deploy.sh --mode dry-run|deploy --image-ref <immutable-ref>'
}

mode=""
image_ref=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      [[ $# -ge 2 ]] || { echo 'error: --mode requires a value' >&2; exit 1; }
      mode="$2"
      shift 2
      ;;
    --image-ref)
      [[ $# -ge 2 ]] || { echo 'error: --image-ref requires a value' >&2; exit 1; }
      image_ref="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown app deployment argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

[[ "${mode}" == 'dry-run' || "${mode}" == 'deploy' ]] || { echo 'error: --mode must be dry-run or deploy' >&2; exit 1; }
[[ "${image_ref}" =~ ^ghcr\.io/tyemirov/loopaware@sha256:[0-9a-f]{64}$ ]] || { echo 'error: --image-ref must be the immutable LoopAware GHCR digest' >&2; exit 1; }
[[ -z "${ANSIBLE_CONFIG:-}" ]] || { echo 'error: ANSIBLE_CONFIG is owned by the LoopAware deployment controller' >&2; exit 1; }
[[ -z "${ANSIBLE_INVENTORY:-}" ]] || { echo 'error: use LOOPAWARE_ANSIBLE_INVENTORY for the canonical app deployment inventory' >&2; exit 1; }
command -v uvx >/dev/null 2>&1 || { echo 'error: uvx is required for the pinned LoopAware Ansible controller' >&2; exit 1; }

repo_root="$(git rev-parse --show-toplevel)"
inventory_path="${LOOPAWARE_ANSIBLE_INVENTORY:-${repo_root}/.mprlab/deploy/ansible/inventory/hosts.yml}"
[[ -f "${inventory_path}" && ! -L "${inventory_path}" ]] || {
  echo "error: LoopAware deployment inventory not found: ${inventory_path}; create it from .mprlab/deploy/ansible/inventory/hosts.yml.example" >&2
  exit 1
}

ansible_config="${repo_root}/.mprlab/deploy/ansible/ansible.cfg"
ansible_local_temp="${repo_root}/.cache/ansible-local"
mkdir -p "${ansible_local_temp}"
export ANSIBLE_CONFIG="${ansible_config}"
export ANSIBLE_LOCAL_TEMP="${ansible_local_temp}"
export LOOPAWARE_DEPLOY_IMAGE_REF="${image_ref}"
export LOOPAWARE_DEPLOY_REPO_ROOT="${repo_root}"

ansible_tool=(uvx --python 3.13 --from ansible-core==2.19.8)
timeout --foreground -k 1200s -s SIGKILL 1200s "${ansible_tool[@]}" ansible-inventory --inventory "${inventory_path}" --list >/dev/null
timeout --foreground -k 1200s -s SIGKILL 1200s "${ansible_tool[@]}" ansible-playbook --inventory localhost, "${repo_root}/.mprlab/deploy/ansible/playbooks/preflight-local.yml"

if [[ "${mode}" == 'dry-run' ]]; then
  echo 'LoopAware app-owned backend preflight passed; production hosts were not contacted and production state was not changed.'
  exit 0
fi

become_flags=()
if [[ -n "${LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE:-}" ]]; then
  [[ -f "${LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE}" && ! -L "${LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE}" ]] || {
    echo 'error: LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE must name a regular file' >&2
    exit 1
  }
  become_flags+=(--become-password-file "${LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE}")
else
  become_flags+=(--ask-become-pass)
fi
timeout --foreground -k 1200s -s SIGKILL 1200s "${ansible_tool[@]}" ansible-playbook "${become_flags[@]}" --inventory "${inventory_path}" "${repo_root}/.mprlab/deploy/ansible/playbooks/deploy.yml"
