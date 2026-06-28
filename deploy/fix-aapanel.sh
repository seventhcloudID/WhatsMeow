#!/bin/bash
# Fix deploy aaPanel - folder site sudah ada isinya
set -e

APP_DIR="/www/wwwroot/meow.solusijasa.com"
REPO="https://github.com/seventhcloudID/WhatsMeow.git"

echo "==> Fix deploy WhatsMeow di $APP_DIR"

systemctl stop meow-gateway 2>/dev/null || true

# Install Go jika belum ada
install_go() {
  GO_VER="1.26.4"
  echo "==> Install Go ${GO_VER}..."
  curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  grep -q '/usr/local/go/bin' /etc/profile 2>/dev/null || echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
}

if ! command -v go &>/dev/null; then
  if [ -x /usr/local/go/bin/go ]; then
    export PATH="/usr/local/go/bin:$PATH"
  else
    install_go
  fi
fi
export PATH="/usr/local/go/bin:$PATH"
echo "    Go: $(go version)"

# Git safe.directory (fix dubious ownership aaPanel)
git config --global --add safe.directory "$APP_DIR" 2>/dev/null || true

# Pull source — fallback rsync jika git pull gagal
cd "$APP_DIR"
if [ -d ".git" ] && git pull origin main 2>/dev/null; then
  echo "==> Git pull OK"
else
  echo "==> Git pull gagal, clone via temp..."
  TMP=$(mktemp -d)
  git clone --depth 1 "$REPO" "$TMP"
  rsync -av --exclude '.user.ini' --exclude '.env' "$TMP/" "$APP_DIR/"
  rm -rf "$TMP"
fi

# .env
if [ ! -f .env ]; then
  cp .env.example .env
  RAND_KEY=$(openssl rand -hex 24 2>/dev/null || head -c 24 /dev/urandom | xxd -p)
  sed -i "s/your-secret-api-key/$RAND_KEY/" .env
  echo ""
  echo "API KEY baru: $RAND_KEY"
  echo "Simpan API key ini!"
fi

mkdir -p data
CGO_ENABLED=0 go mod tidy
CGO_ENABLED=0 go build -o gateway .

chmod +x gateway
chown -R www:www "$APP_DIR" 2>/dev/null || chown -R www:www "$APP_DIR"/* "$APP_DIR"/.* 2>/dev/null || true

systemctl daemon-reload
systemctl restart meow-gateway
sleep 1
systemctl status meow-gateway --no-pager

echo ""
echo "Test: curl http://127.0.0.1:8081/health"
