#!/bin/bash
# wwii-patch-cfg.sh — fill WWII config mysql host/password from the home config
# on the server (fallback /opt/config.yaml). WWII mode connects to the home DB
# read-only; ezfy_mysql inherits host/user/password from the mysql block, so
# patching host+password here is enough (DB may be a remote RDS).
# Optional 3rd arg: home base URL (e.g. http://1.2.3.4:8080) written into
# server.home_url — the WWII front-end "back to home" button jumps there.
# Usage: bash wwii-patch-cfg.sh <target-config.yaml> [home-config.yaml] [home-base-url]
set -u
CFG="${1:-/opt/WWIIbushu/server/config.yaml}"
HOME_CFG="${2:-/opt/Linuxbushu/server/config.yaml}"
HOME_BASE="${3:-}"
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
# server.home_url: drop any existing line, then insert after zone_sub. Some old
# live configs lack zone_sub, so fall back to zone_id / the server: block header.
# (Old templates generated before 2026-10-17 have no home_url line, so
# delete-then-insert is safe.)
if [ -n "$HOME_BASE" ]; then
  sed -i '/^[[:space:]]*home_url:/d' "$CFG"
  if grep -qE '^[[:space:]]*zone_sub:' "$CFG"; then
    sed -i "/^[[:space:]]*zone_sub:/a\\  home_url: \"$HOME_BASE\"" "$CFG"
  elif grep -qE '^[[:space:]]*zone_id:' "$CFG"; then
    sed -i "/^[[:space:]]*zone_id:/a\\  home_url: \"$HOME_BASE\"" "$CFG"
  else
    sed -i "/^server:/a\\  home_url: \"$HOME_BASE\"" "$CFG"
  fi
  echo "OK home_url=$HOME_BASE"
fi
