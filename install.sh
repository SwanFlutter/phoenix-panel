#!/usr/bin/env bash
# ============================================================
# PHOENIX PANEL — Ubuntu installer
#
# Usage (run as root):
#   bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) docker
#   bash <(curl -fsSL https://raw.githubusercontent.com/SwanFlutter/phoenix-panel/main/install.sh) native
# ============================================================
set -euo pipefail

GO_VERSION="1.22.5"
INSTALL_DIR="/opt/phoenix"
SERVICE_NAME="phoenix-panel"
GITHUB_REPO="SwanFlutter/phoenix-panel"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_PORT=8080

log()     { printf '\033[1;36m[phoenix]\033[0m %s\n' "$*"; }
err()     { printf '\033[1;31m[phoenix:error]\033[0m %s\n' "$*" >&2; }
die()     { err "$*"; exit 1; }
success() { printf '\033[1;32m%s\033[0m\n' "$*"; }
bold()    { printf '\033[1m%s\033[0m\n' "$*"; }
info()    { printf '\033[1;33m%s\033[0m\n' "$*"; }

require_root() {
  [ "$(id -u)" -eq 0 ] || die "لطفاً با دسترسی root اجرا کنید (sudo)"
}

# ── اطمینان از اینکه stdin به ترمینال وصله (نه pipe از curl) ──────────────
reattach_tty() {
  # وقتی اسکریپت از طریق bash <(curl ...) اجرا می‌شه، stdin بسته‌ست.
  # /dev/tty رو مستقیم باز می‌کنیم تا read بتونه از کاربر بخونه.
  exec </dev/tty
}

# ── خواندن پسورد از ترمینال (بدون echo) ──────────────────────────────────
read_password() {
  local prompt="$1"
  local var_name="$2"
  local pass=""

  # غیرفعال کردن echo ترمینال
  if [ -t 0 ]; then
    stty -echo 2>/dev/null || true
  fi
  printf "%s" "$prompt"
  read -r pass </dev/tty || pass=""
  echo ""
  if [ -t 0 ]; then
    stty echo 2>/dev/null || true
  fi
  # برگشت مقدار از طریق nameref یا eval
  eval "$var_name=\"\$pass\""
}

# ── خواندن متن معمولی از ترمینال ──────────────────────────────────────────
read_input() {
  local prompt="$1"
  local var_name="$2"
  local val=""
  printf "%s" "$prompt"
  read -r val </dev/tty || val=""
  eval "$var_name=\"\$val\""
}

# ── تنظیم محیط (.env) ─────────────────────────────────────────────────────
ensure_env() {
  # وصل کردن stdin به ترمینال (برای bash <(curl ...) )
  reattach_tty

  if [ ! -f "$SRC_DIR/.env" ]; then
    log "فایل .env پیدا نشد — کپی از .env.example"
    cp "$SRC_DIR/.env.example" "$SRC_DIR/.env"
    local secret
    secret="$(openssl rand -hex 32)"
    sed -i "s|^PHOENIX_JWT_SECRET=.*|PHOENIX_JWT_SECRET=${secret}|" "$SRC_DIR/.env"
    log "یک PHOENIX_JWT_SECRET تصادفی ساخته شد"
  fi

  echo ""
  bold "  ╔══════════════════════════════════════════════════════╗"
  bold "  ║          تنظیم اولیه پنل فونیکس                     ║"
  bold "  ╚══════════════════════════════════════════════════════╝"
  echo ""

  # ── ۱. پسورد ادمین ──────────────────────────────────────────────────────
  local cur_pass
  cur_pass="$(grep -E '^PHOENIX_ADMIN_PASSWORD=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ -z "$cur_pass" ] || [ "$cur_pass" = "admin-change-me" ]; then
    local admin_pass=""
    while [ "${#admin_pass}" -lt 8 ]; do
      read_password "  🔑 پسورد ادمین (حداقل ۸ کاراکتر): " admin_pass
      if [ "${#admin_pass}" -lt 8 ]; then
        err "  پسورد باید حداقل ۸ کاراکتر باشد."
      fi
    done
    sed -i "s|^PHOENIX_ADMIN_PASSWORD=.*|PHOENIX_ADMIN_PASSWORD=${admin_pass}|" "$SRC_DIR/.env"
    log "  ✔ پسورد ادمین ذخیره شد"
  else
    log "  ✔ پسورد ادمین از قبل تنظیم شده"
  fi

  # ── ۲. آدرس پایه (Base URL) ──────────────────────────────────────────────
  local cur_url
  cur_url="$(grep -E '^PHOENIX_BASE_URL=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ -z "$cur_url" ] || [ "$cur_url" = "https://panel.example.com" ]; then
    local server_ip
    server_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"

    echo ""
    info "  ───────────────────────────────────────────────────────"
    info "  🌐  آدرس پایه پنل (برای لینک‌های سابسکریپشن استفاده می‌شود)"
    info "  ───────────────────────────────────────────────────────"
    echo ""
    echo "    [1]  دامنه دارم و SSL دارد    (مثال: https://vpn.example.com)"
    echo "    [2]  ندارم — از IP و پورت پیش‌فرض استفاده کن  (http://${server_ip}:${DEFAULT_PORT})"
    echo ""

    local choice=""
    while [ "$choice" != "1" ] && [ "$choice" != "2" ]; do
      read_input "  انتخاب خود را وارد کنید [1/2]: " choice
      if [ "$choice" != "1" ] && [ "$choice" != "2" ]; then
        err "  لطفاً ۱ یا ۲ را وارد کنید."
      fi
    done

    local base_url=""
    if [ "$choice" = "1" ]; then
      while true; do
        read_input "  آدرس دامنه خود را وارد کنید (مثال: https://vpn.example.com): " base_url
        base_url="${base_url%/}"
        if [[ "$base_url" =~ ^https://[a-zA-Z0-9] ]]; then
          break
        else
          err "  آدرس باید با https:// شروع شود."
        fi
      done
    else
      base_url="http://${server_ip}:${DEFAULT_PORT}"
      log "  آدرس پیش‌فرض انتخاب شد: $base_url"
    fi

    sed -i "s|^PHOENIX_BASE_URL=.*|PHOENIX_BASE_URL=${base_url}|" "$SRC_DIR/.env"
    log "  ✔ آدرس پایه ذخیره شد: $base_url"
  else
    log "  ✔ آدرس پایه از قبل تنظیم شده: $cur_url"
  fi

  # ── ۳. پسورد دیتابیس (فقط برای postgres) ───────────────────────────────
  local cur_driver
  cur_driver="$(grep -E '^PHOENIX_DB_DRIVER=' "$SRC_DIR/.env" | cut -d= -f2-)"
  if [ "$cur_driver" = "postgres" ]; then
    local cur_dbpass
    cur_dbpass="$(grep -E '^PHOENIX_DB_PASSWORD=' "$SRC_DIR/.env" | cut -d= -f2-)"
    if [ -z "$cur_dbpass" ] || [ "$cur_dbpass" = "change-me" ]; then
      echo ""
      log "  درایور دیتابیس PostgreSQL انتخاب شده."
      local db_pass=""
      while [ -z "$db_pass" ]; do
        read_password "  🗄️  پسورد دیتابیس (کاربر phoenix): " db_pass
        if [ -z "$db_pass" ]; then
          err "  پسورد دیتابیس نمی‌تواند خالی باشد."
        fi
      done
      sed -i "s|^PHOENIX_DB_PASSWORD=.*|PHOENIX_DB_PASSWORD=${db_pass}|" "$SRC_DIR/.env"
      log "  ✔ پسورد دیتابیس ذخیره شد"
    fi
  fi

  echo ""
  log "تنظیمات کامل شد — فایل .env آماده است."
  echo ""
}

# ── خلاصه پس از نصب ──────────────────────────────────────────────────────
print_summary() {
  local mode="$1"

  local admin_user admin_pass base_url port
  admin_user="$(grep -E '^PHOENIX_ADMIN_USERNAME=' "$SRC_DIR/.env" | cut -d= -f2-)"
  admin_pass="$(grep -E '^PHOENIX_ADMIN_PASSWORD='  "$SRC_DIR/.env" | cut -d= -f2-)"
  base_url="$(grep   -E '^PHOENIX_BASE_URL='        "$SRC_DIR/.env" | cut -d= -f2-)"
  port="$(grep       -E '^PHOENIX_PORT='            "$SRC_DIR/.env" | cut -d= -f2-)"
  admin_user="${admin_user:-admin}"
  port="${port:-$DEFAULT_PORT}"

  local panel_url="$base_url"
  if [ -z "$panel_url" ] || [ "$panel_url" = "https://panel.example.com" ]; then
    local server_ip
    server_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    panel_url="http://${server_ip}:${port}"
  fi

  echo ""
  echo ""
  success "╔══════════════════════════════════════════════════════════╗"
  success "║        🎉  پنل فونیکس با موفقیت نصب شد                  ║"
  success "╚══════════════════════════════════════════════════════════╝"
  echo ""
  bold   "  ┌─ اطلاعات دسترسی ──────────────────────────────────────┐"
  printf "  │  %-22s  \033[1;33m%s\033[0m\n" "آدرس پنل:"          "$panel_url"
  printf "  │  %-22s  \033[1;33m%s\033[0m\n" "لاگین ادمین:"        "$panel_url/api/admin/login"
  printf "  │  %-22s  \033[1;33m%s\033[0m\n" "بررسی سلامت:"        "$panel_url/healthz"
  bold   "  ├─ اطلاعات ورود ───────────────────────────────────────┤"
  printf "  │  %-22s  \033[1;32m%s\033[0m\n" "نام کاربری:"         "$admin_user"
  printf "  │  %-22s  \033[1;32m%s\033[0m\n" "پسورد:"              "$admin_pass"
  bold   "  ├─ دستورات مفید ───────────────────────────────────────┤"
  if [ "$mode" = "docker" ]; then
    printf "  │  %-22s  %s\n" "مشاهده لاگ:"   "docker compose -f $SRC_DIR/docker-compose.yml logs -f panel"
    printf "  │  %-22s  %s\n" "ری‌استارت:"     "docker compose -f $SRC_DIR/docker-compose.yml restart panel"
    printf "  │  %-22s  %s\n" "توقف:"          "docker compose -f $SRC_DIR/docker-compose.yml down"
    printf "  │  %-22s  %s\n" "آپدیت:"         "bash <(curl -fsSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install.sh) docker"
  else
    printf "  │  %-22s  %s\n" "مشاهده لاگ:"   "journalctl -u $SERVICE_NAME -f"
    printf "  │  %-22s  %s\n" "ری‌استارت:"     "systemctl restart $SERVICE_NAME"
    printf "  │  %-22s  %s\n" "توقف:"          "systemctl stop $SERVICE_NAME"
    printf "  │  %-22s  %s\n" "فایل تنظیمات:"  "$INSTALL_DIR/.env"
  fi
  bold   "  └───────────────────────────────────────────────────────┘"
  echo ""
  printf "  \033[1;31m⚠  این اطلاعات را ذخیره کنید — پسورد دیگر نمایش داده نخواهد شد.\033[0m\n"
  echo ""
}

# ── نصب git ──────────────────────────────────────────────────────────────
check_git() {
  if ! command -v git >/dev/null 2>&1; then
    log "نصب git..."
    apt-get update -y
    apt-get install -y git
  else
    log "git نصب است: $(git --version)"
  fi
}

# ── کلون یا آپدیت مخزن از GitHub ─────────────────────────────────────────
clone_from_github() {
  local repo_dir="/opt/phoenix-panel-src"

  if [ -d "$repo_dir" ]; then
    log "مخزن از قبل وجود دارد — آپدیت..."
    ( cd "$repo_dir" && git pull origin main )
  else
    log "کلون کردن ${GITHUB_REPO} از GitHub..."
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$repo_dir"
  fi

  SRC_DIR="$repo_dir"
  log "مخزن آماده است: $SRC_DIR"
}

# ── نصب Docker ───────────────────────────────────────────────────────────
install_docker_engine() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "Docker و Compose از قبل نصب هستند"
    return
  fi
  log "نصب Docker engine + compose plugin..."
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
  log "ساخت image و راه‌اندازی با docker compose..."
  ( cd "$SRC_DIR" && docker compose up -d --build )
  log "منتظر راه‌اندازی سرویس..."
  for _ in $(seq 1 30); do
    if curl -fsS http://localhost:${DEFAULT_PORT}/healthz >/dev/null 2>&1; then
      print_summary "docker"
      return
    fi
    sleep 2
  done
  err "پنل در زمان مقرر راه‌اندازی نشد — بررسی کنید: docker compose logs -f panel"
}

# ── نصب Go ───────────────────────────────────────────────────────────────
install_go() {
  if command -v go >/dev/null 2>&1; then
    log "Go نصب است: $(go version)"
    return
  fi
  log "نصب Go ${GO_VERSION}..."
  local arch tarball
  arch="$(dpkg --print-architecture)"
  case "$arch" in
    amd64) arch="amd64" ;;
    arm64) arch="arm64" ;;
    *) die "معماری پشتیبانی نمی‌شود: $arch" ;;
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
  log "ساخت باینری phoenix..."
  ( cd "$SRC_DIR" && GOFLAGS=-mod=mod CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w" -o "$SRC_DIR/bin/phoenix" ./cmd/phoenix )
  log "ساخت کامل شد: $SRC_DIR/bin/phoenix"
}

install_native() {
  check_git
  clone_from_github
  ensure_env
  build_binary

  id phoenix >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin phoenix
  mkdir -p "$INSTALL_DIR/data"
  install -m 0755 "$SRC_DIR/bin/phoenix" "$INSTALL_DIR/phoenix"
  install -m 0640 "$SRC_DIR/.env"        "$INSTALL_DIR/.env"
  chown -R phoenix:phoenix "$INSTALL_DIR"

  log "نصب systemd unit..."
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
    github) clone_from_github && log "مخزن با موفقیت کلون شد" ;;
    *) die "نحوه استفاده: sudo ./install.sh [docker|native|github]" ;;
  esac
}

main "$@"
