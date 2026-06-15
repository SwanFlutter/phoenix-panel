#!/usr/bin/env bash
# ============================================================
# PHOENIX PANEL — Ubuntu installer
#
# Usage:
#   sudo ./install.sh docker      # build + run via docker compose (recommended)
#   sudo ./install.sh native      # build a binary + install a systemd service
#
# Run this from the root of the phoenix-panel source tree (where go.mod lives).
# It is idempotent: safe to re-run to update an existing install.
# ============================================================
set -euo pipefail

GO_VERSION="1.22.5"
INSTALL_DIR="/opt/phoenix"
SERVICE_NAME="phoenix-panel"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log()  { printf '\033[1;36m[phoenix]\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31m[phoenix:error]\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

require_root() {
  [ "$(id -u)" -eq 0 ] || die "please run as root (sudo ./install.sh ...)"
}

ensure_env() {
  if [ ! -f "$SRC_DIR/.env" ]; then
    log "no .env found — creating one from .env.example"
    cp "$SRC_DIR/.env.example" "$SRC_DIR/.env"
    local secret
    secret="$(openssl rand -hex 32)"
    sed -i "s|^PHOENIX_JWT_SECRET=.*|PHOENIX_JWT_SECRET=${secret}|" "$SRC_DIR/.env"
    log "generated a random PHOENIX_JWT_SECRET"
    err "IMPORTANT: edit $SRC_DIR/.env and set PHOENIX_ADMIN_PASSWORD, PHOENIX_BASE_URL,"
    err "and PHOENIX_DB_PASSWORD before the panel will start correctly."
  fi
  # Hard fail early if the admin password is still empty.
  if ! grep -qE '^PHOENIX_ADMIN_PASSWORD=.+' "$SRC_DIR/.env"; then
    die "PHOENIX_ADMIN_PASSWORD is empty in .env — set it, then re-run."
  fi
}

# ---------------- Docker path ----------------
install_docker_engine() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "docker + compose already present"
    return
  fi
  log "installing Docker engine + compose plugin"
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
}

run_docker() {
  install_docker_engine
  ensure_env
  log "building and starting the stack via docker compose"
  ( cd "$SRC_DIR" && docker compose up -d --build )
  log "waiting for health..."
  for _ in $(seq 1 30); do
    if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then
      log "panel is healthy at http://localhost:8080"
      return
    fi
    sleep 2
  done
  err "panel did not become healthy in time — check: docker compose logs -f panel"
}

# ---------------- Native path ----------------
install_go() {
  if command -v go >/dev/null 2>&1; then
    log "go already installed: $(go version)"
    return
  fi
  log "installing Go ${GO_VERSION}"
  local arch tarball
  arch="$(dpkg --print-architecture)"
  case "$arch" in
    amd64) arch="amd64" ;;
    arm64) arch="arm64" ;;
    *) die "unsupported architecture: $arch" ;;
  esac
  tarball="go${GO_VERSION}.linux-${arch}.tar.gz"
  curl -fsSLO "https://go.dev/dl/${tarball}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$tarball"
  rm -f "$tarball"
  export PATH="$PATH:/usr/local/go/bin"
  echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
}

build_binary() {
  install_go
  export PATH="$PATH:/usr/local/go/bin"
  log "building phoenix binary"
  ( cd "$SRC_DIR" && GOFLAGS=-mod=mod CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w" -o "$SRC_DIR/bin/phoenix" ./cmd/phoenix )
  log "build complete: $SRC_DIR/bin/phoenix"
}

install_native() {
  build_binary
  ensure_env

  id phoenix >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin phoenix
  mkdir -p "$INSTALL_DIR/data"
  install -m 0755 "$SRC_DIR/bin/phoenix" "$INSTALL_DIR/phoenix"
  install -m 0640 "$SRC_DIR/.env" "$INSTALL_DIR/.env"
  chown -R phoenix:phoenix "$INSTALL_DIR"

  log "installing systemd unit"
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<UNIT
[Unit]
Description=PHOENIX PANEL
After=network.target postgresql.service

[Service]
Type=simple
User=phoenix
Group=phoenix
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/phoenix
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=${INSTALL_DIR}/data

[Install]
WantedBy=multi-user.target
UNIT

  systemctl daemon-reload
  systemctl enable --now "${SERVICE_NAME}"
  sleep 2
  systemctl --no-pager status "${SERVICE_NAME}" || true
  log "done. logs: journalctl -u ${SERVICE_NAME} -f"
}

main() {
  require_root
  local mode="${1:-}"
  case "$mode" in
    docker) run_docker ;;
    native) install_native ;;
    *) die "usage: sudo ./install.sh [docker|native]" ;;
  esac
}

main "$@"
