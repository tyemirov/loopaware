#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: ./down.sh [local|computercat]

When no argument is provided, the script opens an interactive selector.
USAGE
}

resolve_stack_target() {
  local selected_index=0
  local key=""
  local escape_suffix=""
  local option_count=0
  local menu_lines=0
  local index=0
  local -a option_keys=(
    "local"
    "computercat"
  )
  local -a option_labels=(
    "localhost   : http://localhost:8080 (docker-compose.yml)"
    "computercat : https://computercat.tyemirov.net:4443 (docker-compose.computercat.yml)"
  )

  if [[ ! -t 0 || ! -t 1 ]]; then
    echo "error: non-interactive runs must pass an explicit target: ./down.sh [local|computercat]" >&2
    exit 1
  fi

  option_count="${#option_keys[@]}"
  menu_lines=$((option_count + 1))

  while true; do
    printf '\r\033[2KSelect shutdown target (use Up/Down arrows and Enter):\n' >&2
    for index in "${!option_keys[@]}"; do
      printf '\r\033[2K' >&2
      if [[ "${index}" -eq "${selected_index}" ]]; then
        printf '> %s\n' "${option_labels[index]}" >&2
      else
        printf '  %s\n' "${option_labels[index]}" >&2
      fi
    done

    if ! IFS= read -rsn1 key; then
      echo "error: failed to read shutdown target selection." >&2
      exit 1
    fi

    case "${key}" in
      "")
        printf '%s' "${option_keys[selected_index]}"
        return 0
        ;;
      $'\x1b')
        escape_suffix=""
        IFS= read -rsn2 -t 0.1 escape_suffix || true
        case "${escape_suffix}" in
          "[A")
            if [[ "${selected_index}" -eq 0 ]]; then
              selected_index=$((option_count - 1))
            else
              selected_index=$((selected_index - 1))
            fi
            ;;
          "[B")
            if [[ "${selected_index}" -eq $((option_count - 1)) ]]; then
              selected_index=0
            else
              selected_index=$((selected_index + 1))
            fi
            ;;
        esac
        ;;
    esac

    printf '\033[%dA' "${menu_lines}" >&2
  done
}

mode="${1:-}"

case "${mode}" in
  "" )
    mode="$(resolve_stack_target)"
    ;;
  -h|--help)
    usage
    exit 0
    ;;
esac

case "${mode}" in
  local|localhost)
    docker compose -f docker-compose.yml down
    ;;
  computercat)
    docker compose --env-file configs/.env.loopaware.computercat -f docker-compose.computercat.yml down
    ;;
  *)
    echo "Usage: ./down.sh [local|computercat]" >&2
    exit 1
    ;;
esac
