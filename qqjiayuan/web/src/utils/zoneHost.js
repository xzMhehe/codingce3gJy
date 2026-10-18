// 区服入口地址解析 —— **唯一实现**（2026-10-18 修复「二战页无限刷新」）
//
// 为什么要有这个文件：
//   `game_servers.host` 支持三种写法（与后端 `router.go::ezfyZoneURL` 同一套约定）：
//     ''                 → 本服同源（走站内路由，不整页跳）
//     ':8082' / '//:8082' → 跟随**当前访问的域名** + 该端口（多机共享同一个家园库时用）
//     'http(s)://...'    → 原样（真正的跨机器/跨域入口）
//
//   ★★ 事故（2026-10-18 13:20）：`router/index.js` 的路由守卫里**自己又写了一份**跳转逻辑，
//   没做 `:端口` 解析，直接把 `:8081` 当 URL 赋给 `location.href` → 浏览器按**相对路径**
//   解析成 `http://host:8080/:8081`，8080 又返回 SPA 首页 → 守卫再跳一次 → 路径越拼越长
//   （`/:8081/:8081/:8081/...`）、页面无限刷新。
//
//   所以：**所有需要跳区服的地方都 import 这里的 resolveZoneHost**，
//   并且跳转前必须再确认一次「是绝对 http(s) 地址」——见 isAbsoluteHttpUrl。

/**
 * 把 `game_servers.host` 的原始写法解析成可跳转的绝对地址。
 * 解析不出来（空串）时返回 ''，调用方走站内路由。
 * @param {string} raw 后端下发的 host 原值
 * @returns {string} 绝对地址（http(s)://...）或 ''
 */
export function resolveZoneHost (raw) {
  const h = (raw || '').trim()
  if (!h) return ''
  // ':8082' → 当前协议 + 当前域名 + 该端口
  if (h.charAt(0) === ':') return window.location.protocol + '//' + window.location.hostname + h
  // '//:8082' → 同上（少了协议，取当前协议）
  if (h.indexOf('//:') === 0) return window.location.protocol + '//' + window.location.hostname + h.slice(2)
  // '//host:port'（协议相对）→ 补当前协议
  if (h.indexOf('//') === 0) return window.location.protocol + h
  // 裸 'host:port'（没有协议）→ 补当前协议，避免被当相对路径
  if (/^[^/?#]+:\d+(\/|$)/.test(h)) return window.location.protocol + '//' + h
  return h
}

/**
 * 是否是可整页跳转的绝对 http(s) 地址。
 * ★ 护栏：只要不是绝对地址，**绝不**赋给 `location.href`
 *   （相对地址会被浏览器按当前目录解析，反复拼接 = 无限刷新）。
 * @param {string} u
 * @returns {boolean}
 */
export function isAbsoluteHttpUrl (u) {
  return /^https?:\/\/[^/?#]+/i.test((u || '').trim())
}

/**
 * 当前地址栏是否已经出现「端口被当路径」的坏形态（`/ :8081 / :8081 / ...`）。
 * 用于兜底：一旦检测到，直接回首页，绝不再跳一次。
 */
export function looksLikeBrokenZonePath (pathname) {
  return /(^|\/):\d+(\/|$)/.test(pathname || '')
}
