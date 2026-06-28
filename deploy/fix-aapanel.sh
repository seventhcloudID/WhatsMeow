#!/bin/bash
# Fix deploy aaPanel - folder site sudah ada isinya
set -e

APP_DIR="/www/wwwroot/meow.solusijasa.com"
REPO="https://github.com/seventhcloudID/WhatsMeow.git"

echo "==> Fix deploy WhatsMeow di $APP_DIR"

systemctl stop meow-gateway 2>/dev/null || true

# Pastikan Go ada
if ! command -v go &>/dev/null; then
  if [ -d /usr/local/go/bin ]; then
    export PATH="/usr/local/go/bin:$PATH"
  else
    echo "Install Go dulu:"
    echo "  curl -fsSL https://go.dev/dl/go1.26.4.linux-amd64.tar.gz | tar -C /usr/local -xz"
    echo "  export PATH=\$PATH:/usr/local/go/bin"
    exit 1
  fi
fi
export PATH="/usr/local/go/bin:$PATH"
echo "Go: $(go version)"

# Clone ke temp, sync ke folder site (abaikan .user.ini aaPanel)
TMP=$(mktemp -d)
git clone --depth 1 "$REPO" "$TMP"
rsync -av --exclude '.user.ini' "$TMP/" "$APP_DIR/"
rm -rf "$TMP"

cd "$APP_DIR"

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
