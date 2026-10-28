#!/usr/bin/env bash
#
# 统一入口验收脚本
#
#   用法：
#     ./verify-gateway.sh                        # 默认测 47.102.48.155
#     ./verify-gateway.sh 47.102.48.155
#     ./verify-gateway.sh 47.102.48.155 --quick  # 只测主入口
#
# 在哪跑都行：本机 Mac 直接跑（走公网），或者 scp 到 155 上跑。
# 只做只读探测，不改任何东西。
#
set -u

# ★ 本机常配 HTTP_PROXY（127.0.0.1:xxxxx），不关掉的话 curl 走代理，
#   会拿到假的 "upstream connect failed" 而不是 nginx 的真实响应。
export no_proxy='*'
export NO_PROXY='*'

HOST="${1:-47.102.48.155}"
QUICK=0
[ "${2:-}" = "--quick" ] && QUICK=1

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; DIM=$'\033[2m'; NC=$'\033[0m'
PASS=0; FAIL=0; WARN=0

ok()   { PASS=$((PASS+1)); printf "  ${GREEN}✓${NC} %s\n" "$*"; }
bad()  { FAIL=$((FAIL+1)); printf "  ${RED}✗${NC} %s\n" "$*"; }
warn() { WARN=$((WARN+1)); printf "  ${YELLOW}!${NC} %s\n" "$*"; }
sec()  { printf "\n${DIM}── %s ─────────────────────────────────────${NC}\n" "$*"; }

# curl 只取状态码；连不上时 curl 自己会输出 000，这里兜底一次
code() {
  local c
  c="$(curl -s -o /dev/null -m 6 -w '%{http_code}' "$1" 2>/dev/null)"
  printf '%s' "${c:-000}"
}

check_page() { # $1=url $2=名字
  local c; c="$(code "$1")"
  case "$c" in
    200) ok "$2  ($c)  $1" ;;
    000) bad "$2  连不上  $1" ;;
    502|503|504) bad "$2  后端没起/网关不通  ($c)  $1" ;;
    *)   warn "$2  非 200  ($c)  $1" ;;
  esac
}

printf "\n统一入口验收 —— %s\n" "$HOST"

# ============================================================
sec "1. 端口连通性 / 页面可达"
# ============================================================
check_page "http://$HOST:8080/" "8080 家园(主入口)"
check_page "http://$HOST:8081/" "8081 二战1区"
check_page "http://$HOST:8082/" "8082 二战2区"
if [ "$QUICK" = "0" ]; then
  check_page "http://$HOST:8083/" "8083 210家园(副入口)"
  check_page "http://$HOST:8084/" "8084 210二战1区"
  check_page "http://$HOST:8085/" "8085 210二战2区"
fi

# ============================================================
sec "2. 接口健康"
# ============================================================
H="$(curl -s -m 6 "http://$HOST:8080/api/health" 2>/dev/null || true)"
if printf '%s' "$H" | grep -q '"code":0'; then
  ok "8080 /api/health → $(printf '%s' "$H" | head -c 80)"
else
  bad "8080 /api/health 异常：$(printf '%s' "$H" | head -c 120)"
fi

check_page "http://$HOST:8080/admin-ui/" "8080 管理端 /admin-ui/"

# ============================================================
sec "3. 二战实例自报身份（wwii 模式，host 恒为空是正常的）"
# ============================================================
for p in 8081 8082; do
  Z="$(curl -s -m 6 "http://$HOST:$p/api/ezfy/zone" 2>/dev/null || true)"
  if [ -z "$Z" ]; then
    bad "$p /api/ezfy/zone 无响应"
    continue
  fi
  if printf '%s' "$Z" | grep -q '"code":0'; then
    NAME="$(printf '%s' "$Z" | sed -n 's/.*"name":"\([^"]*\)".*/\1/p')"
    HU="$(printf '%s' "$Z" | sed -n 's/.*"home_url":"\([^"]*\)".*/\1/p')"
    ok "$p → 区名「${NAME}」  返回家园 ${HU}"
    case "$HU" in
      *127.0.0.1*) bad "  ↑ home_url 指向 127.0.0.1，玩家点「返回家园」会挂" ;;
    esac
  else
    bad "$p zone 返回异常：$(printf '%s' "$Z" | LC_ALL=C head -c 120)"
  fi
done

# ============================================================
sec "4. 家园下发的二战入口（home 模式，★ 跳转是否配好只看这里）"
# ============================================================
G="$(curl -s -m 6 "http://$HOST:8080/api/ezfy/zone" 2>/dev/null || true)"
if [ -z "$G" ]; then
  bad "8080 /api/ezfy/zone 无响应 —— 家园没起来？"
else
  if printf '%s' "$G" | grep -q '127\.0\.0\.1'; then
    bad "区服入口里出现 127.0.0.1 —— nginx 的 Host 头没传对（见方案第七节坑 2）"
  fi
  if printf '%s' "$G" | grep -qE '"host":"(:[0-9]+|https?://[^"]+)"'; then
    ok "区服入口 → $(printf '%s' "$G" | LC_ALL=C tr -d '\n' | head -c 150)"
  elif printf '%s' "$G" | grep -q '"host":""'; then
    warn "host 为空 = 同源模式，玩家点二战不会整页跳（单机部署才这样）"
  else
    warn "没解析出 host：$(printf '%s' "$G" | LC_ALL=C head -c 150)"
  fi
fi

# ============================================================
sec "5. 直连后端（只在 155 本机有意义）"
# ============================================================
if [ "$HOST" = "127.0.0.1" ] || [ "$HOST" = "localhost" ]; then
  for p in 18080 18081 18082; do
    c="$(code "http://127.0.0.1:$p/api/health")"
    [ "$c" = "200" ] && ok "后端 $p 活着" || bad "后端 $p 不通 ($c)"
  done
  echo
  echo "  监听地址（18080~18082 应只绑 127.0.0.1，不能是 0.0.0.0）："
  ss -lntp 2>/dev/null | grep -E ':1808[0-2]' | sed 's/^/    /' || echo "    (ss 不可用或没监听)"
else
  printf "  ${DIM}跳过（$HOST 不是本机）${NC}\n"
fi

# ============================================================
printf "\n"
printf "结果：${GREEN}通过 %d${NC}  ${RED}失败 %d${NC}  ${YELLOW}提醒 %d${NC}\n" "$PASS" "$FAIL" "$WARN"
if [ "$FAIL" -gt 0 ]; then
  printf "\n${RED}有失败项，按上面的端口号回看方案第 5/7 节。${NC}\n"
  exit 1
fi
printf "\n${GREEN}入口验收通过。${NC}\n"
