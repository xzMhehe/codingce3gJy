#!/bin/bash
# wwii-patch-cfg.sh — fill WWII config mysql host/password from the home config
# on the server (fallback /opt/config.yaml). WWII mode connects to the home DB
# read-only; ezfy_mysql inherits host/user/password from the mysql block, so
# patching host+password here is enough (DB may be a remote RDS).
# Usage: bash wwii-patch-cfg.sh <target-config.yaml> [home-config.yaml]
set -u
CFG="${1:-/opt/WWIIbushu/server/config.yaml}"
HOME_CFG="${2:-/opt/Linuxbushu/server/config.yaml}"
[ -f "$HOME_CFG" ] || HOME_CFG=/opt/config.yaml

PASS=$(sed -n 's/^[[:space:]]*password:[[:space:]]*//p' "$HOME_CFG" 2>/dev/null | head -n1 | tr -d '"' | tr -d '\r')
HOST=$(sed -n 's/^[[:space:]]*host:[[:space:]]*//p' "$HOME_CFG" 2>/dev/null | head -n1 | tr -d '"' | tr -d '\r')

if [ -n "$HOST" ]; then
  sed -i "s/^[[:space:]]*host:[[:space:]]*.*/  host: \"$HOST\"/" "$CFG"
  echo "OK host=$HOST"
else
  echo "WARN: no mysql host found in $HOME_CFG"
fi
if [ -n "$PASS" ]; then
  sed -i "s/^[[:space:]]*password:[[:space:]]*.*/  password: \"$PASS\"/" "$CFG"
  echo "OK password filled (len ${#PASS})"
else
  echo "WARN: no mysql password found in $HOME_CFG"
fi
