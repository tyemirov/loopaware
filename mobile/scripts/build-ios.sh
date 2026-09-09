#!/bin/sh

case "${MPRLAB_GATEWAY_EXECUTABLE-}" in
  /*) ;;
  *)
    printf 'MPRLAB_GATEWAY_EXECUTABLE must identify the authoritative Gateway executable\n' >&2
    exit 2
    ;;
esac

exec "$MPRLAB_GATEWAY_EXECUTABLE" apple-cloud-operation "$@"
