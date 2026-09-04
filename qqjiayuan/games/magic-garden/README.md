# 魔法花园 · 独立前后端

魔法花园是一个**可以独立运行**的前后端小应用，也能作为  家园社区的**一个游戏模块内嵌**。

## 一、单独拿出来运行（不依赖主站）

只安装 Go，无需 Node、无需 MySQL：

```bash
cd games/magic-garden/server
go run main.go
```

浏览器打开 **http://localhost:8090** 即可游玩（种植→生长→收获，赚金币）。

- 后端：标准库实现（仅 `net/http` + `encoding/json`），零第三方依赖
- 数据：内存存储（按昵称隔离），重启信息清空，方便体验
- 前端：独立的 `web/index.html`（原生 JS，参考站 WAP 风格）

改端口：`set PORT=8000 && go run main.go`

## 二、内嵌进主站（qqjiayuan）

主站内的魔法花园是**另一套完整版**（GORM + MySQL 持久化 + Vue 游戏页）：

- 后端：`server/internal/model/garden.go` + `server/internal/handler/garden.go`
- 接口：`/api/games/garden/{view|plant|harvest}`
- 前端：`web/src/views/Garden.vue`（路由 `/#/games/garden`）
- 入口：游戏大厅 `web/src/views/Games.vue` 里的「魔法花园」

主站版与独立版玩法一致，数据各自独立，可按需选用。

## 三、目录结构

```
games/magic-garden/
├── server/
│   └── main.go        # 独立后端（标准库，单独 go run）
├── web/
│   └── index.html     # 独立前端（原生 JS，单独网页）
└── README.md
```

> 想接 MySQL/持久化：把 `main.go` 里 `users` 内存 map 换成数据库即可，API 契约不变。
