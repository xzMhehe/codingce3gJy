#!/bin/bash
# verify-admin-proxy.sh - verify home reverse-proxy to WWII admin endpoints
# Usage: bash verify-admin-proxy.sh <home-8080> <token-file>
H="${1:-http://127.0.0.1:8080}"
T=$(cat "${2:-/tmp/tok}")
echo "=== ADMIN ZONE 1 ==="
curl -s -m 8 "$H/api/admin/ezfy/zone" -H "Authorization: Bearer $T" -H "X-Ezfy-Zone: 1" | head -c 200
echo
echo "=== ADMIN ZONE 2 ==="
curl -s -m 8 "$H/api/admin/ezfy/zone" -H "Authorization: Bearer $T" -H "X-Ezfy-Zone: 2" | head -c 200
echo
echo "=== ADMIN NO TOKEN (expect 401/403) ==="
curl -s -m 8 "$H/api/admin/ezfy/zone" -H "X-Ezfy-Zone: 1" | head -c 120
echo
