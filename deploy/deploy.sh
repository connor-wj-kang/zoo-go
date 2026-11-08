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

if [[ -n "$IMAGE_TAG" ]]; then
  if grep -q '^ZOO_IMAGE=' .env; then
    sed -i.bak "s|^ZOO_IMAGE=.*|ZOO_IMAGE=$IMAGE_TAG|" .env
    rm -f .env.bak
  else
    echo "ZOO_IMAGE=$IMAGE_TAG" >> .env
  fi
fi

docker compose pull
docker compose up -d --remove-orphans
docker compose ps
