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
exec go run ./cmd/api \
  -addr "${ADDR:-:8080}" \
  -data "${DATA_FILE:-data/zoo.db}" \
  -uploads "${UPLOAD_DIR:-data/uploads}" \
  "$@"
