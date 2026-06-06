#!/bin/bash
set -e
REPO=/home/ubuntu/onex-token-lab
GITHUB=https://github.com/zaragoza444/shiva-blockchain.git
export PATH=/usr/local/go/bin:$PATH

if [ ! -d "$REPO/.git" ]; then
  if [ -d "$REPO" ]; then
    cd "$REPO"
    git init
    git remote add origin "$GITHUB" 2>/dev/null || git remote set-url origin "$GITHUB"
    git fetch origin main
    git checkout -f main 2>/dev/null || git checkout -b main
    git reset --hard origin/main
  else
    git clone "$GITHUB" "$REPO"
  fi
else
  cd "$REPO"
  git fetch origin main
  git reset --hard origin/main
fi

cd "$REPO"
go build -o "$REPO/bin/bsc-launcher" ./bsc-launcher/server

if [ ! -f "$REPO/bsc-launcher/.env" ]; then
  cp "$REPO/bsc-launcher/.env.production.example" "$REPO/bsc-launcher/.env"
fi

sudo systemctl restart onex-token-lab
sleep 2
curl -s http://127.0.0.1:9340/health
echo
systemctl is-active onex-token-lab
