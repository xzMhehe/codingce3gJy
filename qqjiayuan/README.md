# 家园社区（QQ家园复刻版）

> 致敬 2008 年腾讯 QQ家园 /  家园社区。当年天天签到、盖楼灌水、聊天交友的日子，用现代技术栈复刻回来。
> 2015年6月20日 QQ家园正式停止运营，但情怀不散。

基于 **MySQL + Gin(Go 1.26) + GORM + Vue 2.x + RBAC + 前后端分离** 实现的完整社区平台。
游戏（抢车位、好友买卖、魔法花园、阳光牧场、精武堂、召唤之王、纵横四海……）陆续在做，先把家园搭起来 —— 游戏大厅已留好入口（敬请期待）。

## 功能一览

| 模块 | 说明 |
|------|------|
| 用户系统 | 注册自动分配家园号码（靓号）、登录、找回资料、修改资料/密码、头像选择、昵称颜色（情怀功能） |
| 马甲/勋章 | 昵称前的图标串（706.jpg 职务章、3.gif/501.gif/704.gif/803.gif/804.gif/103.gif/15.gif/903.gif/45.gif 等演示站原版素材）；等级图标 v{N}.gif 随等级自动生成；等级称号（新人报到→…→册封骑士→一代宗师）；后台「马甲管理」可新建/删除勋章、给用户授予/摘下 |
| RBAC 权限 | 用户-角色-权限三级模型，内置 超级管理员/管理员/版主/普通会员 四角色，后台可自建角色、分配权限 |
| 社区广场 | 公告/广播/活动、在线人数、最新居民、社区头条（精华）、四大频道最新帖、友友动态、社区搜索 |
| 论坛盖楼 | 分区-板块两级（公共论坛/家族大厅/同城客栈/社区服务）、发帖、回复盖楼（自动楼层）、浏览/回复计数、置顶/精华、分页 |
| 每日签到 | 签到得经验金币（连续签到奖励递增）、连续天数榜、全站签到统计 |
| 好友 | 按号码加好友、申请（互加/通过/拒绝/忽略）、删除；好友备注、分组移动、黑名单、好友新鲜事（复刻诺哈 wap_friend 单方向模型，好友策略 允许/需验证/拒绝） |
| 私信 | 会话列表（未读数）、单人往来、已读回执 |
| 聊天室 | 全园公共聊天，3 秒轮询自动刷新 |
| 消息通知 | 回复通知、好友申请通知、系统通知，全部已读 |
| 管理后台 | 概览统计、用户管理（封禁/重置密码/分配角色/设马甲）、板块管理、帖子管理（置顶/精华/删帖）、公告管理、角色权限管理、马甲管理 |
| 等级经验 | 发帖+10经验+5金币，回帖+5经验+2金币，签到+20经验；等级按经验自动计算 |

## 目录结构

```
qqjiayuan/
├── server/                 # Go 后端（Gin + GORM）
│   ├── main.go
│   ├── config.yaml         # 端口 / MySQL / JWT 配置
│   ├── internal/
│   │   ├── config/         # 配置加载
│   │   ├── model/          # GORM 模型（用户/角色/权限/板块/帖子/回复/签到/好友/私信/聊天室/通知/公告）
│   │   ├── middleware/     # CORS、JWT 认证、RBAC 权限中间件
│   │   ├── handler/        # 认证/用户/板块/帖子/广场/签到/好友/私信/聊天室/通知/管理后台
│   │   ├── router/         # 路由注册（生产模式托管前端 dist）
│   │   └── seed/           # 建表 + 幂等种子数据（板块/角色权限/示例帖子）
│   └── pkg/                # 数据库初始化、JWT 工具、统一响应
└── web/                    # Vue 2.7 前端（vue-router + vuex + axios）
    └── src/
        ├── views/          # 广场/登录/注册/板块/帖子/签到/好友/私信/聊天室/导航/游戏大厅/管理后台...
        ├── components/admin/  # 管理后台五个面板
        └── assets/style.css   # 复刻自演示站的 WAP 风格样式

另外还有一个独立管理端：
└── admin-web/              # Vue 2.7 管理后台（8001 端口，/admin-ui/ 入口，独立于 web）
```

## 快速开始

### 0. 环境要求

- Go 1.26+、Node 14+（开发 Vue2 用）、MySQL 5.7+/8.x
- 已在本机 3306 端口运行 MySQL

### 1. 配置数据库

编辑 `server/config.yaml`：

```yaml
mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "你的密码"
  dbname: qq_jiayuan
```

数据库无需手工建表：首次启动 GORM 自动建表 + 写入种子数据
（也可先执行 `CREATE DATABASE qq_jiayuan DEFAULT CHARACTER SET utf8mb4;`）。

### 2. 启动后端（同时托管前端）

```bash
cd server
go mod tidy      # 首次拉取依赖
go run .
```

服务启动在 `http://127.0.0.1:8080`，直接浏览器访问即是完整站点
（`web_dir` 指向 `../web/dist`，需要先执行第 3 步构建前端；未构建时仅提供 API）。

### 3. 构建前端（首次 / 前端有改动时）

```bash
cd web
npm install
npm run build
```

### 4. 前后端分离开发模式（可选）

```bash
cd web && npm run serve     # 前端热更新在 http://localhost:8000，/api 代理到 8080
cd server && go run .       # 后端照常
```

### 5. 开发阶段热更新启动（推荐：改代码自动生效，免手动重编译）

后端用 **air** 监听 `.go` 文件变化自动重编译重启；两个前端 `npm run serve` 自带 HMR 热模块替换。

```bash
# ① 后端（air 热重载，监听 server 下 .go/.yaml 变化）
cd qqjiayuan/server
air                    # 首次需安装: go install github.com/air-verse/air@latest

# ② 用户端（HMR 热更新，端口 8000，/api 代理到 8080）
cd qqjiayuan/web
npm run serve

# ③ 管理端（HMR 热更新，端口 8001，入口 /admin-ui/，/api 与 /static 代理到 8080）
cd qqjiayuan/admin-web
npm run serve
```

> 说明：
> - 改 Go 代码后 air 自动 `build + restart`，无需手动 `go run`；编译失败会输出到 `server/build-errors.log`。
> - 改 Vue 代码后浏览器自动热更新，无需刷新（个别情况 F5 一下）。
> - 生产部署仍用 `npm run build` + `go build`，热更新仅限开发阶段。
> - air 配置见 `server/.air.toml`（监听 .go/.yaml/.toml/.html，排除 tmp/dist/node_modules）。

## 演示账号

| 账号 | 密码 | 角色 |
|------|------|------|
| 10000 | admin123 | 站长小Q（超级管理员，可进管理后台） |
| 10001 ~ 10005 | 123456 | 云起、安珞、咏荷、闲云野鹤、蓝天（普通会员） |

也可在注册页随便注册新号，自动分配下一个靓号。

## 内置权限点（RBAC）

| 权限码 | 说明 |
|--------|------|
| admin:access | 进入管理后台 |
| user:manage | 封禁/解封/重置密码/分配角色 |
| board:manage | 板块增删改 |
| thread:manage | 置顶/精华/删帖删回复 |
| announcement:manage | 公告广播增删改 |
| role:manage | 角色与权限分配 |

中间件用法：`router.DELETE("/x", JWTAuth(db, secret), RequirePerm(db, "thread:manage"), handler)`

## 主要 API 一览

```
POST /api/auth/register          注册（自动分配号码）
POST /api/auth/login             登录
GET  /api/auth/me                当前用户（含角色权限）
GET  /api/auth/find              找回资料（凭昵称查号码）
GET  /api/plaza                  广场聚合数据
GET  /api/boards                 板块树
GET  /api/boards/:id/threads     板块帖子列表（分区聚合子板块）
POST /api/boards/:id/threads     发帖
GET  /api/threads/:id            帖子详情+楼层
POST /api/threads/:id/replies    回复盖楼
POST /api/signin                 每日签到
GET  /api/signin/info            签到状态与排行榜
GET/POST /api/friends            好友列表/申请（备注/分组/在线/申请处理/黑名单/新鲜事）
POST /api/messages               发私信
GET/POST /api/chat               聊天室拉取/发言
GET  /api/notifications          通知列表
GET  /api/admin/*                管理后台（RBAC 校验）
```

## 一键脚本

- `start.bat` —— 编译并启动后端（前端已构建时直接整站可用）

## 致谢与说明

- 页面风格复刻自「家园演示源码」中的  WAP 社区（蓝条、黄条、模块标题、小Q报时）。
- 本项目仅供学习交流与情怀复刻，请勿用于商业用途。
