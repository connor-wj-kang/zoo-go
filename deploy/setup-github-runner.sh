#!/usr/bin/env bash
# Install or repair the GitHub Actions self-hosted runner on the ECS deploy host.
# Usage (on the server):
#   curl/scp this script, then:
#   RUNNER_TOKEN=XXXX sudo -E bash deploy/setup-github-runner.sh
#
# Get RUNNER_TOKEN from:
#   gh api -X POST repos/connor-wj-kang/zoo-go/actions/runners/registration-token --jq .token
# Token expires in ~1 hour. Only needed for first configure / reconfigure.
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/connor-wj-kang/zoo-go}"
RUNNER_VERSION="${RUNNER_VERSION:-2.338.2}"
RUNNER_USER="${RUNNER_USER:-github-runner}"
RUNNER_HOME="${RUNNER_HOME:-/opt/actions-runner}"
RUNNER_NAME="${RUNNER_NAME:-$(hostname -s)}"
LABELS="${LABELS:-self-hosted,Linux,X64}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root (sudo)" >&2
  exit 1
fi

if ! id "$RUNNER_USER" &>/dev/null; then
  useradd --system --create-home --home-dir "/home/$RUNNER_USER" --shell /bin/bash "$RUNNER_USER"
fi

# Node is required by miniprogram-ci upload.
if ! command -v node >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y ca-certificates curl gnupg
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y nodejs
  else
    echo "install Node.js 20+ manually, then re-run" >&2
    exit 1
  fi
fi

mkdir -p "$RUNNER_HOME"
chown -R "$RUNNER_USER:$RUNNER_USER" "$RUNNER_HOME"
cd "$RUNNER_HOME"

if [[ ! -f ./config.sh ]]; then
  curl -fsSL -o actions-runner.tgz \
    "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
  tar xzf actions-runner.tgz
  rm -f actions-runner.tgz
  chown -R "$RUNNER_USER:$RUNNER_USER" "$RUNNER_HOME"
fi

if [[ ! -f .runner ]]; then
  if [[ -z "${RUNNER_TOKEN:-}" ]]; then
    echo "missing RUNNER_TOKEN for first-time configure" >&2
    echo "  gh api -X POST repos/connor-wj-kang/zoo-go/actions/runners/registration-token --jq .token" >&2
    exit 1
  fi
  sudo -u "$RUNNER_USER" ./config.sh --unattended \
    --url "$REPO_URL" \
    --token "$RUNNER_TOKEN" \
    --name "$RUNNER_NAME" \
    --labels "$LABELS" \
    --work _work \
    --replace
fi

./svc.sh install "$RUNNER_USER"
./svc.sh start
./svc.sh status

echo
echo "OK: runner should show Online in GitHub → Settings → Actions → Runners"
echo "Add this machine public IP to WeChat MP → 开发设置 → 小程序代码上传 → IP 白名单"
ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}' || true
