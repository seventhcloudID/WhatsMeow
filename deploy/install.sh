#!/bin/bash
set -e

APP_DIR="/www/wwwroot/meow.solusijasa.com"
REPO="https://github.com/seventhcloudID/WhatsMeow.git"
SERVICE="meow-gateway"

echo "==> WhatsMeow Gateway - Install VPS"
echo "    Directory: $APP_DIR"

# Go minimal 1.21
if ! command -v go &>/dev/null; then
  echo "==> Install Go..."
  GO_VER="1.26.4"
  curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  grep -q '/usr/local/go/bin' /etc/profile || echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
fi

export PATH="/usr/local/go/bin:$PATH"
echo "    Go: $(go version)"

mkdir -p "$APP_DIR"
cd "$APP_DIR"

if [ ! -d ".git" ]; then
  if [ -z "$(ls -A "$APP_DIR" 2>/dev/null)" ]; then
    git clone "$REPO" .
  else
    echo "ERROR: Folder tidak kosong dan belum ada .git"
    echo "       Kosongkan dulu atau clone manual: git clone $REPO $APP_DIR"
    exit 1
  fi
else
  echo "==> Git pull..."
  git pull origin main
fi

echo "==> Build binary..."
go mod tidy
CGO_ENABLED=0 go build -o gateway .

mkdir -p data
chmod +x gateway

if [ ! -f .env ]; then
  cp .env.example .env
  # Ganti API key random
  RAND_KEY=$(openssl rand -hex 24 2>/dev/null || head -c 24 /dev/urandom | xxd -p)
  sed -i "s/your-secret-api-key/$RAND_KEY/" .env
  echo ""
  echo "==> .env dibuat. API KEY: $RAND_KEY"
  echo "    Simpan API key ini!"
fi

# systemd
cat > /etc/systemd/system/${SERVICE}.service <<EOF
[Unit]
Description=WhatsMeow WhatsApp Gateway
After=network.target

[Service]
Type=simple
User=www
Group=www
WorkingDirectory=${APP_DIR}
ExecStart=${APP_DIR}/gateway
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

chown -R www:www "$APP_DIR"

systemctl daemon-reload
systemctl enable ${SERVICE}
systemctl restart ${SERVICE}

echo ""
echo "==> Selesai!"
echo "    Status : systemctl status ${SERVICE}"
echo "    Log    : journalctl -u ${SERVICE} -f"
echo "    Health : curl http://127.0.0.1:8081/health"
echo ""
echo "Langkah berikutnya: setup reverse proxy Nginx (lihat deploy/nginx.conf)"
