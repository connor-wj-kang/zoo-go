#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.local}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "缺少本地环境文件：$ENV_FILE" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

if [[ "${DEV_AUTH:-false}" != "true" && -z "${WECHAT_APP_SECRET:-}" ]]; then
  echo "WECHAT_APP_SECRET 未配置，且 DEV_AUTH 未开启" >&2
  exit 1
fi

if [[ -n "${WECHAT_APP_SECRET:-}" && "${#WECHAT_APP_SECRET}" -ne 32 ]]; then
  echo "WECHAT_APP_SECRET 长度异常：${#WECHAT_APP_SECRET}，正常应为 32" >&2
  exit 1
fi

cd "$ROOT_DIR"

start_openobserve() {
  if [[ "${SKIP_OPENOBSERVE:-false}" == "true" ]]; then
    echo "跳过 OpenObserve（SKIP_OPENOBSERVE=true）"
    return 0
  fi
  if [[ -z "${O2_PASS:-}" ]]; then
    echo "O2_PASS 未配置，无法启动 OpenObserve（可在 .env.local 设置，或 SKIP_OPENOBSERVE=true）" >&2
    exit 1
  fi
  if ! command -v docker >/dev/null 2>&1; then
    echo "未找到 docker，无法启动 OpenObserve" >&2
    exit 1
  fi

  local port="${O2_PORT:-5080}"
  echo "启动 OpenObserve（localhost:${port}）..."
  docker compose up -d openobserve

  local url="http://127.0.0.1:${port}/"
  local i
  for i in $(seq 1 60); do
    if curl -sf -o /dev/null "$url"; then
      echo "OpenObserve 已就绪"
      return 0
    fi
    sleep 1
  done
  echo "OpenObserve 在 ${port} 上未在 60s 内就绪，请检查：docker logs openobserve" >&2
  exit 1
}

start_openobserve

exec go run ./cmd/api \
  -addr "${ADDR:-:8080}" \
  -data "${DATA_FILE:-data/zoo.db}" \
  -uploads "${UPLOAD_DIR:-data/uploads}" \
  "$@"
