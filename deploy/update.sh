#!/bin/bash
# Update deploy: pull + rebuild + restart
set -e
APP_DIR="/www/wwwroot/meow.solusijasa.com"
cd "$APP_DIR"
git pull origin main
go mod tidy
CGO_ENABLED=0 go build -o gateway .
systemctl restart meow-gateway
echo "Updated. Status:"
systemctl status meow-gateway --no-pager
