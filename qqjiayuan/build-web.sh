#!/usr/bin/env bash
#
# QQ JiaYuan - build both frontends (production)
#
# Produces web/dist and admin-web/dist, which the Go server hosts.
# Needs Node + npm on PATH. Missing node_modules triggers npm install.
#
# Usage:  ./build-web.sh
#
set -euo pipefail

cd "$(dirname "$0")"

REG="${NPM_REGISTRY:-https://registry.npmmirror.com}"

echo "============================================================"
echo "  QQ JiaYuan  -  BUILD FRONTEND"
echo "============================================================"
echo

if ! command -v node >/dev/null 2>&1; then
  echo "  [ERROR] node not found in PATH."
  exit 1
fi
if ! command -v npm >/dev/null 2>&1; then
  echo "  [ERROR] npm not found in PATH."
  exit 1
fi
echo "  node $(node -v)"
echo "  npm  $(npm -v)"
echo

build_one () {
  local dir="$1"
  echo "[build] $dir"
  if [ ! -d "$dir/node_modules" ]; then
    echo "  node_modules missing - npm install, first run is slow"
    ( cd "$dir" && npm install --registry="$REG" )
  fi
  ( cd "$dir" && npm run build )
  echo "  ok"
  echo
}

build_one web
build_one admin-web

echo "============================================================"
echo "  Done. Output:"
echo "    web/dist"
echo "    admin-web/dist"
echo
echo "  Next: copy both dist folders to the server, then start"
echo "  server-2012r2.exe. Do not forget web/dist/static."
echo "============================================================"
