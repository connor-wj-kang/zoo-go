#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="$ROOT_DIR/deploy"
IMAGE_TAG="${1:-}"

cd "$DEPLOY_DIR"

if [[ ! -f .env ]]; then
  echo "missing $DEPLOY_DIR/.env — copy from .env.example and fill secrets" >&2
  exit 1
fi

mkdir -p "$ROOT_DIR/data/uploads" "$ROOT_DIR/data/openobserve" "$ROOT_DIR/secrets"
# Container runs as uid 10001; host-mounted SQLite needs write access.
if [[ "$(id -u)" -eq 0 ]]; then
  chown -R 10001:10001 "$ROOT_DIR/data" || true
fi

if [[ -n "$IMAGE_TAG" ]]; then
  if grep -q '^ZOO_IMAGE=' .env; then
    sed -i.bak "s|^ZOO_IMAGE=.*|ZOO_IMAGE=$IMAGE_TAG|" .env
    rm -f .env.bak
  else
    echo "ZOO_IMAGE=$IMAGE_TAG" >> .env
  fi
fi

# Registry images should be pulled; local tags like zoo-go:local can skip.
zoo_image="$(grep -E '^ZOO_IMAGE=' .env | cut -d= -f2- || true)"
if [[ "$zoo_image" == */* ]]; then
  docker compose pull
fi
docker compose up -d --remove-orphans
docker compose ps
