package ezfy

import (
	"strings"
	"testing"
)

// ★★ 2026-10-18 事故：家园 SPA 访问 `/#/games/ezfy?cur=home` 无限刷新，
// 地址栏变成 `http://host:8080/:8081/:8081/:8081/.../#/games/ezfy?sso_token=...`。
//
// 根因：路由守卫（web/src/router/index.js）里自己又写了一份区服跳转，没把
// `game_servers.host` 的 ':8081' 写法解析成 `http(s)://<当前域名>:8081`，
// 直接把 ':8081' 赋给 location.href → 浏览器按**相对路径**解析 →
// 8080 的 SPA fallback 又把页面吐回来 → 守卫再跳 → 路径每次多一层、无限刷新。
//
// 修复：区服地址解析抽成**唯一实现** `web/src/utils/zoneHost.js`
// （resolveZoneHost + isAbsoluteHttpUrl + looksLikeBrokenZonePath），
// 三个跳转点（路由守卫 / Games.vue enterServer / Ezfy.vue exitToHome）全部走它，
// 且跳转前必须确认是绝对 http(s) 地址；已出现坏形态地址时一律回首页不再跳。
//
// 这组护栏是**静态断言**（读前端源码），防止再出现「一处修了另一处没修」。

// TestZoneHostSingleImplementation resolveZoneHost 只能有一处实现（utils/zoneHost.js），
// 其它文件不得再内联同样的函数。
func TestZoneHostSingleImplementation(t *testing.T) {
	impl := strings.ReplaceAll(rawFile(t, "../../../../web/src/utils/zoneHost.js"), "\r\n", "\n")
	if !strings.Contains(impl, "export function resolveZoneHost") {
		t.Fatal("utils/zoneHost.js 必须导出 resolveZoneHost（区服地址解析的唯一实现）")
	}
	if !strings.Contains(impl, "export function isAbsoluteHttpUrl") {
		t.Fatal("utils/zoneHost.js 必须导出 isAbsoluteHttpUrl（跳转前护栏）")
	}
	if !strings.Contains(impl, "export function looksLikeBrokenZonePath") {
		t.Fatal("utils/zoneHost.js 必须导出 looksLikeBrokenZonePath（坏形态地址兜底）")
	}
	// Games.vue 不得再有内联实现（事故前它有一份本地 resolveSrvHost，守卫没同步）
	games := strings.ReplaceAll(rawFile(t, "../../../../web/src/views/Games.vue"), "\r\n", "\n")
	if strings.Contains(games, "function resolveSrvHost") {
		t.Fatal("Games.vue 仍有内联 resolveSrvHost —— 区服解析必须共用 utils/zoneHost.js，别再各写一套")
	}
}

// TestZoneJumpersAllGuarded 三个整页跳转点必须：护栏 + 坏形态兜底；
// 前两个还要用 resolveZoneHost 解析（Ezfy.vue 的 homeUrl 是后端 config 的绝对地址，无 `:端口` 原值，不适用）。
func TestZoneJumpersAllGuarded(t *testing.T) {
	cases := []struct {
		file        string
		needResolve bool
	}{
		{"../../../../web/src/router/index.js", true},
		{"../../../../web/src/views/Games.vue", true},
		{"../../../../web/src/views/ezfy/Ezfy.vue", false},
	}
	for _, c := range cases {
		src := strings.ReplaceAll(rawFile(t, c.file), "\r\n", "\n")
		if !strings.Contains(src, "from '../utils/zoneHost'") &&
			!strings.Contains(src, "from '../../utils/zoneHost'") {
			t.Fatalf("%s 没有引入 utils/zoneHost —— 区服跳转必须共用唯一实现", c.file)
		}
		if c.needResolve && !strings.Contains(src, "resolveZoneHost(") {
			t.Fatalf("%s 没有调用 resolveZoneHost 解析区服地址（`:端口` 写法会被当相对路径）", c.file)
		}
		if !strings.Contains(src, "isAbsoluteHttpUrl(") {
			t.Fatalf("%s 跳转前没有用 isAbsoluteHttpUrl 护栏（相对地址赋给 location.href = 无限刷新）", c.file)
		}
		if !strings.Contains(src, "looksLikeBrokenZonePath(") {
			t.Fatalf("%s 没有坏形态地址兜底（looksLikeBrokenZonePath → 回首页，不再跳）", c.file)
		}
	}
}

// TestRouterGuardNeverHrefsRelativeAddress 路由守卫里 location.href 的赋值右边必须是解析后的绝对地址，
// 不得再出现 `zone.host` 原始值直接拼接。
func TestRouterGuardNeverHrefsRelativeAddress(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "../../../../web/src/router/index.js"), "\r\n", "\n")
	if strings.Contains(src, "window.location.href = zone.host") {
		t.Fatal("路由守卫仍把 zone.host 原样赋给 location.href —— `:8081` 会被当相对路径，无限刷新")
	}
	if !strings.Contains(src, `window.location.href = host.replace(/\/+$/, '') + '/#' + p`) {
		t.Fatal("路由守卫的跳转目标必须用解析后的绝对地址（host）拼 `/#` + 站内路由")
	}
}
