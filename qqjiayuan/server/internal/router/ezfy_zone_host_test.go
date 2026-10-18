package router

import "testing"

// 护栏：区服入口 host 的三种写法（★ 2026-10-18 多机共享同一个家园库）。
//
// 事故背景：两台机器（39 / 124）共用同一个家园库 → `game_servers` 只有一份。
// 原来 host 写死 `http://124.222.208.92:8082`，导致在 39 上点「2 区」也跳到 124 机器。
// 现在允许写「:8082」= 跟随当前访问的域名 + 该端口。
func TestEzfyZoneURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		scheme  string
		reqHost string
		want    string
	}{
		{"空=本服同源", "", "http", "39.105.151.141:8080", ""},
		{"纯空白=本服同源", "   ", "http", "39.105.151.141:8080", ""},
		{"相对端口-跟随域名", ":8082", "http", "39.105.151.141:8080", "http://39.105.151.141:8082"},
		{"相对端口-另一台机器", ":8082", "http", "124.222.208.92:8080", "http://124.222.208.92:8082"},
		{"相对端口-域名访问", ":8081", "https", "game.example.com", "https://game.example.com:8081"},
		{"相对端口-双斜杠写法", "//:8082", "http", "39.105.151.141:8080", "http://39.105.151.141:8082"},
		{"相对端口-IPv6", ":8082", "http", "[2001:db8::1]:8080", "http://[2001:db8::1]:8082"},
		{"绝对地址原样", "http://124.222.208.92:8082", "http", "39.105.151.141:8080", "http://124.222.208.92:8082"},
		{"绝对地址带尾斜杠原样", "https://a.example.com/", "http", "39.105.151.141:8080", "https://a.example.com/"},
	}
	for _, c := range cases {
		if got := ezfyZoneURL(c.raw, c.scheme, c.reqHost); got != c.want {
			t.Errorf("%s: ezfyZoneURL(%q,%q,%q) = %q, want %q", c.name, c.raw, c.scheme, c.reqHost, got, c.want)
		}
	}
}

// 反代走本机回环：`:8082` → `http://127.0.0.1:8082`（不绕公网/DNS）。
func TestEzfyZoneLoopback(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"", ""},
		{":8082", "http://127.0.0.1:8082"},
		{"//:8081", "http://127.0.0.1:8081"},
		{"http://10.0.0.5:8081", "http://10.0.0.5:8081"},
	}
	for _, c := range cases {
		if got := ezfyZoneLoopback(c.raw); got != c.want {
			t.Errorf("ezfyZoneLoopback(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}
