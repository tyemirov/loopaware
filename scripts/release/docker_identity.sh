#!/usr/bin/env bash

assert_local_docker_endpoint() {
  local variable_name
  for variable_name in DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH; do
    [[ -z "${!variable_name:-}" ]] || {
      echo "error: ${variable_name} is not supported by the canonical lifecycle" >&2
      return 1
    }
  done

  command -v docker >/dev/null 2>&1 || {
    echo "error: docker is required" >&2
    return 1
  }

  local context
  local endpoint
  context="$(docker context show)"
  [[ -n "${context}" ]] || {
    echo "error: Docker context could not be resolved" >&2
    return 1
  }
  endpoint="$(docker context inspect "${context}" --format '{{.Endpoints.docker.Host}}')"
  case "${endpoint}" in
    unix://*|npipe://*) ;;
    *)
      echo "error: canonical lifecycle requires a local Docker endpoint, got ${endpoint:-<missing>} from context ${context}" >&2
      return 1
      ;;
  esac
}
