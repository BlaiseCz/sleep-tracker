#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/fresh_start.sh [options]

Options:
  -f, --force    Recreate .env and docker-compose.yml from their *.example templates.
  -h, --help     Show this message.
EOF
}

FORCE=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--force)
      FORCE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

log_step() {
  printf '\n==> %s\n' "$1"
}

log_info() {
  printf '   • %s\n' "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd docker
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 (docker compose) is required." >&2
  exit 1
fi

prepare_file() {
  local src="$1"
  local dest="$2"
  local label="$3"

  if [[ ! -f "$src" ]]; then
    echo "Template missing: $src" >&2
    exit 1
  fi

  if [[ -f "$dest" ]]; then
    if [[ "$FORCE" == true ]]; then
      rm -f "$dest"
      cp "$src" "$dest"
      log_info "Reset $label from template"
    else
      log_info "Found existing $label ($dest); leaving as is"
    fi
  else
    cp "$src" "$dest"
    log_info "Copied $label template to $dest"
  fi
}

log_step "Preparing configuration templates"
prepare_file ".env.example" ".env" ".env"
prepare_file "docker-compose.yml.example" "docker-compose.yml" "docker-compose.yml"

log_step "Stopping any leftover containers"
docker compose down --volumes --remove-orphans >/dev/null 2>&1 || true

log_step "Building and starting services"
docker compose up -d --build

log_step "Current container status"
docker compose ps

log_info "API base URL: http://localhost:8080"
log_info "Docs: http://localhost:8080/swagger/index.html"
log_info "Tail logs with: docker compose logs -f api"
log_info "Seed sample data by running: SEED=true docker compose up api"
