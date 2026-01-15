#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/langfuse_stack.sh [command]

Commands:
  start     Stop any existing stack, then build & start Langfuse services
  stop      Stop Langfuse services and remove volumes/networks
  status    Show current Langfuse container status
  help      Show this message
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

COMMAND="$1"
shift || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

COMPOSE_FILE="langfuse-docker-compose.yml"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "Langfuse compose file not found at $COMPOSE_FILE" >&2
  exit 1
fi

log_step() {
  printf '\n==> %s\n' "$1"
}

case "$COMMAND" in
  start)
    log_step "Stopping previous Langfuse stack"
    docker compose -f "$COMPOSE_FILE" down --volumes --remove-orphans >/dev/null 2>&1 || true

    log_step "Building and starting Langfuse stack"
    docker compose -f "$COMPOSE_FILE" up -d --build

    log_step "Langfuse container status"
    docker compose -f "$COMPOSE_FILE" ps

    printf '\n   • Configure your initial org/user via http://localhost:3001 before running Sleep Tracker\n'
    ;;
  stop)
    log_step "Stopping Langfuse stack"
    docker compose -f "$COMPOSE_FILE" down --volumes --remove-orphans
    ;;
  status)
    log_step "Langfuse container status"
    docker compose -f "$COMPOSE_FILE" ps
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "Unknown command: $COMMAND" >&2
    usage >&2
    exit 1
    ;;
esac
