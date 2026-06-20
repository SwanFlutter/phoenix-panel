#!/usr/bin/env bash
# ============================================================
# PHOENIX PANEL — Ubuntu installer
#
# Usage:
#   sudo ./install.sh docker      # build + run via docker compose (recommended)
#   sudo ./install.sh native      # build a binary + install a systemd service
#   sudo ./install.sh github      # clone from GitHub + install
#
# Run this from the root of the phoenix-panel source tree (where go.mod lives).
# It is idempotent: safe to re-run to update an existing install.
# ============================================================
set -euo pipefail

GO_VERSION="1.22.5"
INSTALL_DIR="/opt/phoenix"
SERVICE_NAME="phoenix-panel"
GITHUB_REPO="SwanFlutter/phoenix-panel"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log()     { printf '\033[1;36m[phoenix]\033[0m %s\n' "$*"; }
err()     { printf '\033[1;31m[phoenix:error]\033[0m %s\n' "$*" >&2; }
die()     { err "$*"; exit 1; }
success() { printf '\033[1;32m%s\033[0m\n' "$*"; }
bold()    { printf '\033[1m%s\033[0m\n' "$*"; }

require_root() {
  [ "$(id -u)" -eq 0 ] || die "please run as root (sudo ./install.sh ...)"
}

# ── Post-install summary ─────────────────────────────────────────────────────
print_summary() {
  local mode="$1"   # docker | native

  # Read values from the .env that was written during ensure_env
  local admin_user admin_pass base_url port
  admin_user="$(grep -E '^PHOENIX_ADMIN_USERNAME=' "$SRC_DIR/.env" | cut -d= -f2-)"
  admin_pass="$(grep -E '^PHOENIX_ADMIN_PASSWORD='  "$SRC_DIR/.env" | cut -d= -f2-)"
  base_url="$(grep   -E '^PHOENIX_BASE_URL='        "$SRC_DIR/.env" | cut -d= -f2-)"
  port="$(grep       -E '^PHOENIX_PORT='            "$SRC_DIR/.env" | cut -d= -f2-)"
  admin_user="${admin_user:-admin}"
  port="${port:-8080}"

  # Derive a panel URL: prefer PHOENIX_BASE_URL, fall back to server IP
  local panel_url="$base_url"
  if [ -z "$panel_url" ] || [ "$panel_url" = "https://panel.example.com" ]; then
    local server_ip
    server_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    panel_url="http://${server_ip}:${port}"
  fi

  echo ""
  echo ""
  success "╔══════════════════════════════════════════════════════════╗"
  success "║          🎉  PHOENIX PANEL INSTALLED SUCCESSFULLY         ║"
  success "╚══════════════════════════════════════════════════════════╝"
  echo ""
  bold   "  ┌─ ACCESS INFORMATION ──────────────────────────────────┐"
  printf "  │  %-20s  \033[1;33m%s\033[0m\n" "Panel URL:"      "$panel_url"
  printf "  │  %-20s  \033[1;33m%s\033[0m\n" "Admin API:"      "$panel_url/api/admin/login"
  printf "  │  %-20s  \033[1;33m%s\033[0m\n" "Health check:"   "$panel_url/healthz"
  bold   "  ├─ CREDENTIALS ────────────────────────────────────────┤"
  printf "  │  %-20s  \033[1;32m%s\033[0m\n" "Username:"       "$admin_user"
  printf "  │  %-20s  \033[1;32m%s\033[0m\n" "Password:"       "$admin_pass"
  bold   "  ├─ USEFUL COMMANDS ────────────────────────────────────┤"
  if [ "$mode" = "docker" ]; then
    printf "  │  %-20s  %s\n" "View logs:"    "docker compose -f $SRC_DIR/docker-compose.yml logs -f panel"
    printf "  │  %-20s  %s\n" "Restart:"      "docker compose -f $SRC_DIR/docker-compose.yml restart panel"
    printf "  │  %-20s  %s\n" "Stop:"         "docker compose -f $SRC_DIR/docker-compose.yml down"
    printf "  │  %-20s  %s\n" "Update:"       "bash <(curl -fsSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh) docker"
  else
    printf "  │  %-20s  %s\n" "View logs:"    "journalctl -u $SERVICE_NAME -f"
    printf "  │  %-20s  %s\n" "Restart:"      "systemctl restart $SERVICE_NAME"
    printf "  │  %-20s  %s\n" "Stop:"         "systemctl stop $SERVICE_NAME"
    printf "  │  %-20s  %s\n" "Config file:"  "$INSTALL_DIR/.env"
  fi
  bold   "  └───────────────────────────────────────────────────────┘"
  echo ""
  printf "  \033[1;31m⚠  Save these credentials — the password will not be shown again.\033[0m\n"
  echo ""
}

ensure_env() {
  if [ ! -f "$SRC_DIR/.env" ]; then
    log "no .env found — creating one from .env.example"
    cp "$SRC_DIR/.env.example" "$SRC_DIR/.env"
    local secret
    secret="$(openssl rand -hex 32)"
    sed -i "s|^PHOENIX_JWT_SECRET=.*|PHOENIX_JWT_SECRET=${secret}|" "$SRC_DIR/.env"
    log "generated a random PHOENIX_JWT_SECRET"
  fi

  # ── Interactive prompts ──────────────────────────────────────────────────
  # Collect required values that are still at their placeholder defaults.

  # 1. PHOENIX_ADMIN_PASSWORD
  local cur_pass
  cur_pass="$(grep -E '^PHOENIX_ADMIN_PASSWORD=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ -z "$cur_pass" ] || [ "$cur_pass" = "admin-change-me" ]; then
    echo ""
    log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log "  SETUP — enter your panel admin credentials"
    log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    local admin_pass=""
    while [ -z "$admin_pass" ]; do
      read -rsp "  Admin password (min 8 chars): " admin_pass
      echo ""
      if [ "${#admin_pass}" -lt 8 ]; then
        err "  Password too short — must be at least 8 characters."
        admin_pass=""
      fi
    done
    sed -i "s|^PHOENIX_ADMIN_PASSWORD=.*|PHOENIX_ADMIN_PASSWORD=${admin_pass}|" "$SRC_DIR/.env"
    log "  ✔ admin password saved"
  fi

  # 2. PHOENIX_BASE_URL
  local cur_url
  cur_url="$(grep -E '^PHOENIX_BASE_URL=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ -z "$cur_url" ] || [ "$cur_url" = "https://panel.example.com" ]; then
    echo ""
    log "  Enter the public URL of this server (used for subscription links)."
    log "  Example: https://vpn.mydomain.com   or   http://1.2.3.4:8080"
    local base_url=""
    while [ -z "$base_url" ]; do
      read -rp "  Base URL: " base_url
      if [[ ! "$base_url" =~ ^https?:// ]]; then
        err "  URL must start with http:// or https://"
        base_url=""
      fi
    done
    base_url="${base_url%/}"   # strip trailing slash
    sed -i "s|^PHOENIX_BASE_URL=.*|PHOENIX_BASE_URL=${base_url}|" "$SRC_DIR/.env"
    log "  ✔ base URL saved"
  fi

  # 3. PHOENIX_DB_PASSWORD  (only needed when driver=postgres)
  local cur_driver
  cur_driver="$(grep -E '^PHOENIX_DB_DRIVER=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ "$cur_driver" = "postgres" ]; then
    local cur_dbpass
    cur_dbpass="$(grep -E '^PHOENIX_DB_PASSWORD=' "$SRC_DIR/.env" | cut -d= -f2-)"
    if [ -z "$cur_dbpass" ] || [ "$cur_dbpass" = "change-me" ]; then
      echo ""
      log "  PostgreSQL is selected as the database driver."
      local db_pass=""
      while [ -z "$db_pass" ]; do
        read -rsp "  Database password for user 'phoenix': " db_pass
        echo ""
        if [ -z "$db_pass" ]; then
          err "  Database password cannot be empty."
        fi
      done
      sed -i "s|^PHOENIX_DB_PASSWORD=.*|PHOENIX_DB_PASSWORD=${db_pass}|" "$SRC_DIR/.env"
      log "  ✔ database password saved"
    fi
  fi

  echo ""
  log "configuration complete — .env is ready."
  echo ""
}

# Check if git is installed
check_git() {
  if ! command -v git >/dev/null 2>&1; then
    log "installing git"
    apt-get update -y
    apt-get install -y git
  else
    log "git already installed: $(git --version)"
  fi
}

# Clone or update repository from GitHub
clone_from_github() {
  local repo_dir="/opt/phoenix-panel-src"
  
  if [ -d "$repo_dir" ]; then
    log "repository already exists at $repo_dir, updating..."
    ( cd "$repo_dir" && git pull origin main )
  else
    log "cloning ${GITHUB_REPO} from GitHub..."
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$repo_dir"
  fi
  
  SRC_DIR="$repo_dir"
  log "repository ready at $SRC_DIR"
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
  check_git
  clone_from_github
  install_docker_engine
  ensure_env
  log "building and starting the stack via docker compose"
  ( cd "$SRC_DIR" && docker compose up -d --build )
  log "waiting for health..."
  for _ in $(seq 1 30); do
    if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then
      print_summary "docker"
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
  check_git
  clone_from_github
  ensure_env
  build_binary

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
  print_summary "native"
}

main() {
  require_root
  local mode="${1:-}"
  case "$mode" in
    docker) run_docker ;;
    native) install_native ;;
    github) clone_from_github && log "repository cloned successfully" ;;
    *) die "usage: sudo ./install.sh [docker|native|github]" ;;
  esac
}

main "$@"
