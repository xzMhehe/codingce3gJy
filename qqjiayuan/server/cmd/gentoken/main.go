package main

import (
	"fmt"
	"os"
	"strconv"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/pkg/authutil"
)

// 生成登录 token（调试 / 验证用）。
//
//	go run ./cmd/gentoken              → 默认 uid=10000「站长小Q」
//	go run ./cmd/gentoken 10007        → 指定 uid
//	go run ./cmd/gentoken 10007 夜凌云  → 指定 uid + 昵称
//
// ★ 2026-09-25：原来 uid 写死 10000，命令行参数被忽略 —— 验「A 打 B」这类需要**两个玩家**
// 的场景时，两个 token 其实是同一个人，表现为「目标太近了」（自己打自己的城）。
// 现在把 uid/昵称做成**可选参数**，默认值不变，老用法照旧。
func main() {
	uid := uint(10000)
	nick := "站长小Q"
	if len(os.Args) > 1 {
		v, err := strconv.ParseUint(os.Args[1], 10, 64)
		if err != nil || v == 0 {
			fmt.Fprintln(os.Stderr, "uid 必须是正整数, 收到:", os.Args[1])
			os.Exit(1)
		}
		uid = uint(v)
	}
	if len(os.Args) > 2 && os.Args[2] != "" {
		nick = os.Args[2]
	}
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}
	t, err := authutil.GenerateToken(uid, nick, cfg.Jwt.Secret, 24)
	if err != nil {
		panic(err)
	}
	fmt.Println(t)
}
