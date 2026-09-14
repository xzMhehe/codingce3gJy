# 家园社区 — Windows Server 2012 R2 部署清单

> 目标环境：Windows Server 2012 R2 数据中心版 64 位中文版
> 部署形态：Go 后端单进程托管用户端（`web/dist`）与管理端（`admin-web/dist`），MySQL 本机存储
> 使用方式：**先读第零、零之二节**（讲清 Go 的"不支持"到底意味着什么，以及服务器上哪些组件必装、哪些可选），再从上往下依次执行。**不要跳步。**
> 本文档里的所有下载地址与版本号均已逐个核实。

---

## 零、先厘清一件事：「官方不支持」≠「跑不起来」

这两件事必须分开看，否则会觉得文档自相矛盾：

| 说法 | 含义 | 性质 |
|------|------|------|
| **官方不支持 2012 R2** | Go 团队不再测试它、不再保证它、出问题不会修 | **支持政策** |
| **跑不起来** | 二进制在这台机器上执行失败（报错 / 闪退） | **技术事实** |

Go 1.21 把 2012 R2 移出支持范围，主要动作是**停掉构建机和测试**，并不是运行时会主动拒绝 2012 R2。而且 2012 R2 的内核是 **Windows 8.1**，比 Win7 新得多 —— Go 1.21+ 在旧 Windows 上的已知崩溃点里，好几条是**专门针对 Win7** 的（`ProcessPrng` 在 Win8 之前不存在、控制台句柄复制、`-race` 依赖 `api-ms-win-core-synch`），2012 R2 未必命中。

**结论：装是能装的，但跑不跑得起来必须实测。** 实测点在**第六节**（真正跑一次 `server.exe`）。

还有一个更重要的点：**服务器上其实可以不装 Go。** Go 编译出的是纯静态二进制，服务器只需要那个 `server.exe` 文件，不需要工具链。装 Go 只是为了"能在服务器上直接编译"这个便利。

---

## 零之二、版本选型（服务器照这张表装）

| 组件 | 是否必须装 | 该装的版本 | 说明 |
|------|-----------|-----------|------|
| **MySQL** | **必须** | **8.0.x** | 8.0 全系列支持 Win7 / Server 2008 R2 及更高；8.4 / 9.x 只支持 Server 2016+ |
| **VC++ 运行库** | **必须** | 2015–2022 x64 | MySQL 8.0 的运行前提，2012 R2 默认不带 |
| **Node.js** | 可选 | **16.20.2** `x64` | 只在服务器上构建前端才需要。Node 16 是**最后一个**对 2012 R2 提供 Tier 1 支持的版本 |
| **Go** | 可选 | **1.26.6** `windows-amd64` | 只在服务器上编译后端才需要。⚠️ 官方不支持 2012 R2，见上一节 |
| **Git** | 可选 | 2.24.0+ | 只在服务器上编译后端才需要（Go 1.25+ 要求 Git ≥ 2.24.0） |

> **推荐做法**：MySQL 和 VC++ 必装；**Node 和 Go 都别装** —— 前端和后端都在开发机编好，把产物拷过来。这样服务器上只剩一个变量要验证（`server.exe` 能不能跑），排查面最小。
>
> 如果你已经装了 Node 22.23.2 / Go 1.27.1 又不想折腾，也可以：**直接跳到第六节实测 `server.exe` 能不能跑**，能跑的话连工具链都不用管。

> **另一件事**：2012 R2 已于 **2023-10-10 结束支持**，付费扩展安全更新（ESU）**2026-10-13 到期**（就在本月）。到期后不再有任何安全补丁。这一条和组件版本无关，但部署前你应该知道。

### 下载地址（均已验证可下载）

| 用途 | 文件 | 地址 |
|------|------|------|
| MySQL | MySQL 8.0 系列 | `https://dev.mysql.com/downloads/mysql/8.0.html` |
| VC++ 运行库 | `vc_redist.x64.exe` | 微软官网搜 "Visual C++ Redistributable 2015-2022" |
| Node.js（可选） | `node-v16.20.2-x64.msi` | `https://nodejs.org/dist/v16.20.2/node-v16.20.2-x64.msi` |
| Go（可选） | `go1.26.6.windows-amd64.msi` | `https://go.dev/dl/go1.26.6.windows-amd64.msi` |

### 装 Node / Go 之前先卸载旧版

只在你要重装这两个组件时才需要：

1. 控制面板 → 程序和功能 → 卸载 `Node.js`（22.23.2）和 `Go Programming Language`（1.27.1）
2. 卸载后**手动删残留目录**：`C:\Program Files\nodejs`、`C:\Program Files\Go`、`%APPDATA%\npm`
3. 再装新版本，装完**开一个新的 CMD 窗口**验证（旧窗口的环境变量不会刷新）

---

## 一、工具链验证（可选 —— 只在打算于服务器上编译时才需要）

> **如果你按推荐做法走**（开发机编好产物再拷过来，服务器上不装 Node / Go），**这一节整节跳过**，直接去第二节装 MySQL。
> 决定方案成立与否的关键验证是**第六节**（真正跑一次 `server.exe`），不是这一节。

### 1. 确认工具链版本

打开**一个新的** CMD 窗口（Win+R → `cmd`），逐条执行：

```cmd
ver
node -v
npm -v
go version
```

**期望输出**：

```
Microsoft Windows [版本 6.3.9600]      ← 6.3.9600 就是 Server 2012 R2
v16.20.2                               ← Node 必须是 16.x
8.19.4                                 ← 随 Node 16 自带
go version go1.26.6 windows/amd64      ← Go 必须是 1.26.x
```

对照下表判断：

| 命令输出 | 含义 | 后续路线 |
|---------|------|---------|
| `node -v` 打印 `v16.20.2` | Node 官方支持 2012 R2 | 可在服务器上构建前端（4.2 路线 A） |
| `node -v` 仍是 `v22.x` | 旧版本没卸干净 | 回「零之二、版本选型」重装 |
| `node -v` 闪退 / 报「找不到入口点」 | 装的还是 18+ 的版本 | 重装 16.20.2 |
| `go version` 打印 `go1.26.6 windows/amd64` | Go 工具链能启动 | 可编译后端，**但必须先做下面第 2 步实测** |
| `go version` 报错 / 闪退 | Go 工具链跑不起来 | 走 4.1 路线 B，改在开发机交叉编译 |

### 2. 给 Go 单独做一次"最小程序"实测

**Go 官方不支持 2012 R2，所以"`go` 命令能敲"不等于"编出来的程序能跑"。** 用 **PowerShell** 编一个最小程序验证（比 CMD 的 echo 拼源码可靠）：

```powershell
$d = "$env:TEMP\gotest"
New-Item -ItemType Directory -Force -Path $d | Out-Null
Set-Location $d
@'
package main

import "fmt"

func main() { fmt.Println("GO OK ON 2012R2") }
'@ | Set-Content -Encoding UTF8 main.go
go mod init gotest
go build -o gotest.exe .
.\gotest.exe
```

- 打印出 `GO OK ON 2012R2` → Go 在 2012 R2 上可用，继续往下做
- 报错或闪退（典型：`找不到入口点`、`api-ms-win-*.dll 缺失`）→ **直接跳到附录 A**，别在这上面继续耗

> **诊断技巧**：如果程序一闪而过看不清报错，在已打开的窗口里执行 `.\gotest.exe`，报错会留在窗口里。
> 注意这个测试验证的是"**Go 工具链**能不能在这台机器上跑"。最终要跑的 `server.exe` 能不能跑，看第六节。

**本步验收**：明确 Node 和 Go 工具链在服务器上是否可用。结论决定第 4 步走哪条路。

---

## 二、系统依赖准备

- [ ] **安装 VC++ 2015–2022 Redistributable (x64)** —— MySQL 8.0 的运行前提，2012 R2 默认不带
      - 下载 `vc_redist.x64.exe` 安装后**重启**
- [ ] **确认时区为 `(UTC+08:00) 北京`**
      - 后端连接串是 `...&parseTime=True&loc=Local`，依赖系统时区。时区错会导致所有时间戳偏移
      - 命令：`tzutil /g`，设置：`tzutil /s "China Standard Time"`
- [ ] **规划磁盘**：MySQL 数据目录和日志不要放系统盘（2012 R2 系统盘常偏小）
- [ ] **杀软排除**：把 MySQL 数据目录（`datadir`）和应用目录加入杀毒软件/Defender 排除列表，否则 MySQL 性能会明显劣化

---

## 三、数据库（MySQL）

### 3.1 安装

**装 MySQL 8.0 系列**（`https://dev.mysql.com/downloads/mysql/8.0.html`）。

- 8.0 全系列的支持范围覆盖 Windows 7 / Server 2008 R2 及更高版本，**2012 R2 在支持范围内**
- **不要装 8.4 / 9.x** —— 那些只支持 Server 2016 / 2019 / 2022
- 装之前先确认第二节的 **VC++ 2015–2022 Redistributable (x64)** 已安装，否则 MySQL 起不来

安装时**务必**：

- 字符集选 `utf8mb4`
- 勾选「Install as Windows Service」并设为自动启动
- 记住 root 密码

> **兜底方案**：如果 8.0 在这台机器上装不上，退到 **MySQL 5.7** 也能跑 —— 驱动 `go-sql-driver/mysql v1.8.1` 明确声明支持 `MySQL 5.7+`，而且项目代码里没有任何 MySQL 8 专有语法（无窗口函数、CTE、`JSON_TABLE`、`utf8mb4_0900` 排序规则）。

### 3.2 一键初始化：dbinit 工具（推荐）

`server\dbinit-2012r2.exe` 是配套的数据库初始化工具，做三件事：

1. **建库** —— `CREATE DATABASE IF NOT EXISTS`，已存在则跳过
2. **建表** —— `AutoMigrate` 全部 185 张表
3. **灌种子数据** —— 板块、角色权限、公告、演示站居民、幻想西游的全部游戏数据（走 `go:embed`，不依赖外部文件）

**全部幂等，可以重复执行。** 首次跑大约十几秒。

用法（在 `D:\qqjiayuan\server\` 下）：

```cmd
cd /d D:\qqjiayuan\server
dbinit-2012r2.exe
```

> ⚠️ **跑之前先创建 `config.yaml`** —— 工具靠它拿 MySQL 连接信息。内容见**第五节**。
> 顺序上可以这样安排：先把 `config.yaml` 建好（第零节 → 第二节 → 第五节），再回来跑 `dbinit`。

它默认读同目录的 `config.yaml`，也可以显式指定：

```cmd
dbinit-2012r2.exe -config D:\qqjiayuan\server\config.yaml
```

**关于账号权限**：建库需要 `CREATE` 权限。建议**第一次用 root 跑**（临时把 `config.yaml` 的 `user`/`password` 改成 root），
初始化完成后再按 3.3 建受限账号，并把 `config.yaml` 改回 `jiayuan`。

> 看到 `初始化完成` 就是成功了。
> 日志里可能出现 `Error 1060 Duplicate column`、`Error 1062 Duplicate entry`、`record not found` 这类信息 ——
> 都是幂等逻辑的正常输出，**不影响结果**。
>
> 如果建库失败，工具会列出常见原因（MySQL 没启动 / 密码错 / 账号无 CREATE 权限）。

### 3.3 建专用数据库账号（不要用 root 跑应用）

```sql
CREATE USER 'jiayuan'@'localhost' IDENTIFIED BY '换成你的强密码';
GRANT ALL PRIVILEGES ON qq_jiayuan.* TO 'jiayuan'@'localhost';
FLUSH PRIVILEGES;
```

建完把 `config.yaml` 的 `user` / `password` 改成这个账号。

### 3.4 手工初始化（不想用工具时的备选）

**第一步，手工建库**（用 Navicat，或命令行）：

```sql
CREATE DATABASE IF NOT EXISTS qq_jiayuan
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
```

> 排序规则用 `utf8mb4_unicode_ci`，与项目导出文件一致（`qq_jiayuan.sql` 里用的就是这个），且 MySQL 5.7 / 8.x 都支持。

**第二步，建表 + 灌种子**：直接启动 `server.exe` 就行 —— 它启动时会自动 `AutoMigrate` 建全部表并写种子数据。

> ⚠️ `server.exe` **不会建库**（它直接连 `config.yaml` 里指定的库，库不存在就报 `Unknown database` 然后退出），
> 所以第一步不能省。这也是单独提供 `dbinit` 工具的原因。

**另一条路：导入现有导出文件**（想把开发机的数据原样搬过去时用）

```cmd
cd /d D:\qqjiayuan\db
"D:\mysql\bin\mysql.exe" --default-character-set=utf8mb4 -u root -p qq_jiayuan < qq_jiayuan.sql
```

- **`--default-character-set=utf8mb4` 不能省**，否则所有中文会变成 `????`
- 导入文件放在**纯英文路径**下（如 `D:\qqjiayuan\db\`），避免中文路径在 936 代码页下解析出错
- 该文件是 Navicat 导出、UTF-8（65001）、含 67 张表、**不含 `CREATE DATABASE`**，所以必须先建好库再导

### 3.5 验收

```sql
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'qq_jiayuan';
```

跑完 `dbinit` 或启动过 `server.exe` 后，应为 **185 左右**（随版本迭代会增减）。

再抽查种子数据：

```sql
SELECT COUNT(*) FROM users;   -- 管理员 + 演示站居民，约 22
SELECT COUNT(*) FROM games;   -- 19
SELECT id, name, category FROM games WHERE name LIKE '%西游%';   -- 幻想西游 / com
```

- 中文必须正常显示（`幻想西游`），**不能是 `????`**
- 如果 `games` 是 0 张，说明初始化没跑完，重跑一次 `dbinit`

---

## 四、准备部署产物

产物有两种做法：在服务器上直接编译，或在开发机编译好拷过来。**前端强烈建议用后者**（少一层依赖就少一处故障点）；后端看第一节第 2 步的实测结果。

### 4.1 后端 `server.exe`

**路线 A：服务器上编译**（仅当第一节第 2 步的 Go 最小程序实测通过）

```cmd
cd /d D:\qqjiayuan\src\server
set GOPROXY=https://goproxy.cn,direct
go mod download
go build -trimpath -ldflags="-s -w" -o D:\qqjiayuan\server\server.exe .
```

- 服务器不能上外网时，改用 `set GOFLAGS=-mod=vendor`，并在开发机先执行 `go mod vendor` 把 `vendor/` 目录一起拷过来
- 编译后把 `server.exe` 挪到 `D:\qqjiayuan\server\`

**路线 B：开发机交叉编译后拷贝**（推荐，不依赖服务器工具链）

在开发机上执行：

```bash
cd qqjiayuan/server
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o server.exe .
```

然后把 `server.exe` 拷到服务器 `D:\qqjiayuan\server\`。

> `CGO_ENABLED=0` 保证生成纯静态二进制，服务器上不需要任何 C 运行库。
> 开发机需要有能编这个项目的 Go（`go.mod` 要求 `go 1.26`）。

### 4.2 前端产物

**路线 A：服务器上构建**（Node 16 官方支持 2012 R2，可用）

```cmd
cd /d D:\qqjiayuan\src\web
npm install --registry=https://registry.npmmirror.com
npm run build

cd /d D:\qqjiayuan\src\admin-web
npm install --registry=https://registry.npmmirror.com
npm run build
```

**路线 B：开发机构建后拷贝**（推荐）

在项目根目录跑现成的构建脚本：

```bash
# macOS / Linux
./build-web.sh

# Windows
build-web.bat
```

它会把两个前端依次构建好。等价的原始命令是：

```bash
cd qqjiayuan/web && npm run build
cd ../admin-web && npm run build
```

然后把 `web/dist`、`admin-web/dist` 两个目录整体拷到服务器。

> ⚠️ **别漏 `web/dist/static`** —— 里面是全部演示站图片素材（几千个 gif/图片），漏了页面会大面积裂图。
>
> ⚠️ **构建必须在普通终端里跑**。构建过程会先清空再重建 `dist` 目录，在受限/沙箱化的终端里会被批量删除保护拦住而失败。
> 如果报 `SAFE_DELETE_BULK_CONFIRM_REQUIRED` 或 `EEXIST ... mkdir`，换个正常的终端窗口重跑即可。

### 4.3 服务器上的最终目录结构

```
D:\qqjiayuan\
├── server\
│   ├── server-2012r2.exe   ← 后端可执行文件（补丁版，推荐）
│   ├── server.exe          ← 后端可执行文件（官方版，兜底）
│   ├── dbinit-2012r2.exe   ← 数据库初始化工具（第三节用）
│   ├── dbinit.exe          ← 数据库初始化工具（官方版，兜底）
│   └── config.yaml         ← 配置文件（第五节生成）
├── web\
│   └── dist\               ← 用户端产物
│       ├── index.html
│       ├── js\
│       ├── css\
│       └── static\         ← 图片素材，别漏
├── admin-web\
│   └── dist\               ← 管理端产物
│       └── index.html
└── logs\                   ← 日志目录（手工新建）
```

> `server-2012r2.exe` 和 `dbinit-2012r2.exe` 是用**打了旧 Windows 兼容补丁的 Go 工具链**编译的
> （`net` 的 socket 创建回退、`runtime` 的 `ProcessPrng` 回退、`os` 的目录枚举回退），
> 在 2012 R2 上成功率明显更高。不带 `-2012r2` 后缀的那两个是官方 Go 编译的兜底版本。
> 先试带后缀的，不行再换另一个。

---

## 五、配置 `config.yaml`

在 `D:\qqjiayuan\server\` 下创建 `config.yaml`：

```yaml
server:
  port: 8080
  # ⚠️ 必须写绝对路径！见下方说明
  web_dir: "D:/qqjiayuan/web/dist"
  admin_web_dir: "D:/qqjiayuan/admin-web/dist"

mysql:
  host: 127.0.0.1
  port: 3306
  user: jiayuan
  password: "换成 3.3 里设的强密码"
  dbname: qq_jiayuan

jwt:
  secret: "换成一串 64 位以上的随机字符串"
  expire_hours: 168
```

**为什么必须写绝对路径：**

`main.go` 里 `config.yaml` 和 `web_dir` 都是相对路径。一旦注册成 Windows 服务，服务进程的默认工作目录是 `C:\Windows\System32`，后果是：

- 找不到 `config.yaml` → 直接打印「读取配置失败」并退出
- 或者 API 正常但前端整站 404（`web_dir` 指不到目录时，静态托管会**静默不注册**，没有任何报错）

改成绝对路径可以彻底规避这个问题。路径用正斜杠 `/` 即可，Go 在 Windows 上两种都认。

**生成 JWT secret：**

```cmd
powershell -Command "([guid]::NewGuid().ToString('N')) + ([guid]::NewGuid().ToString('N'))"
```

> 默认值 `qq-jiayuan-secret-2026-do-not-leak` 必须换掉，否则任何人拿到源码就能伪造任意用户的登录令牌。

---

## 六、首次试运行 —— 这一步才是决定方案成败的关键验证

> **前面第零节说的"Go 官方不支持 2012 R2，必须实测"，实测点就是这里。**
> 因为 `server.exe` 里嵌着 Go 运行时，它能不能在这台机器上启动，只能真正跑一次才知道。

**这一步一定要在 CMD 里跑，不要直接做服务。** 做服务后报错会进日志文件，排查麻烦得多。

```cmd
chcp 65001
cd /d D:\qqjiayuan\server
server.exe
```

预期输出：

```
MySQL 连接成功: qq_jiayuan
==========================================
  家园社区 服务端已启动  http://127.0.0.1:8080
  默认管理员：账号 10000 / 密码 admin123
==========================================
```

> `chcp 65001` 是为了让中文日志正常显示（否则 936 代码页下会乱码）。这只影响显示，不影响功能。

**另开一个 CMD 窗口验证**（注意：Server 2012 R2 **没有自带 `curl.exe`**，用 PowerShell）：

```powershell
Invoke-WebRequest http://127.0.0.1:8080/api/health -UseBasicParsing | Select-Object -Expand Content
```

应返回 `{"code":0,"msg":"ok","data":"家园社区 API 运行中"}`。

再验证前端：

```powershell
(Invoke-WebRequest http://127.0.0.1:8080/ -UseBasicParsing).StatusCode          # 期望 200
(Invoke-WebRequest http://127.0.0.1:8080/admin-ui/ -UseBasicParsing).StatusCode # 期望 200
```

**本步验收**：

- ✅ **通过** → 按 `Ctrl+C` 停掉，继续第七节注册服务
- ❌ **`server.exe` 启动即报错 / 闪退**（典型：`找不到入口点`、`api-ms-win-*.dll 缺失`、无输出直接退出）
  → 这就是"**Go 跑不起来**"，直接看 **附录 A**

---

## 七、注册为 Windows 服务

### 方案一：NSSM（推荐）

1. 下载 NSSM（`nssm.cc`，单文件 exe），解压到 `D:\tools\nssm\`
2. 以**管理员身份**打开 CMD：

```cmd
D:\tools\nssm\nssm.exe install qqjiayuan "D:\qqjiayuan\server\server.exe"
D:\tools\nssm\nssm.exe set qqjiayuan AppDirectory "D:\qqjiayuan\server"
D:\tools\nssm\nssm.exe set qqjiayuan AppParameters "-config D:\qqjiayuan\server\config.yaml"
D:\tools\nssm\nssm.exe set qqjiayuan AppStdout "D:\qqjiayuan\logs\server-out.log"
D:\tools\nssm\nssm.exe set qqjiayuan AppStderr "D:\qqjiayuan\logs\server-err.log"
D:\tools\nssm\nssm.exe set qqjiayuan AppRotateFiles 1
D:\tools\nssm\nssm.exe set qqjiayuan AppRotateBytes 10485760
D:\tools\nssm\nssm.exe set qqjiayuan AppExit Default Restart
D:\tools\nssm\nssm.exe set qqjiayuan AppRestartDelay 5000
D:\tools\nssm\nssm.exe set qqjiayuan Start SERVICE_AUTO_START
D:\tools\nssm\nssm.exe start qqjiayuan
```

> `AppDirectory` 是关键项 —— 它设置服务的工作目录，绕过第五节说的相对路径陷阱。

### 方案二：任务计划程序（免下载备选）

1. 任务计划程序 → 创建任务
2. 常规：勾选「不管用户是否登录都要运行」+「使用最高权限运行」
3. 触发器：启动时
4. 操作 → 启动程序：`D:\qqjiayuan\server\server.exe`
5. **起始于**：`D:\qqjiayuan\server` ← 等价于 NSSM 的 AppDirectory，**必填**
6. 设置：勾选「如果任务失败，按以下频率重新启动」

### 验收

```cmd
sc query qqjiayuan
```

应显示 `STATE : 4 RUNNING`。重启服务器，确认服务自动拉起。

---

## 八、防火墙

```cmd
netsh advfirewall firewall add rule name="qqjiayuan-8080" dir=in action=allow protocol=TCP localport=8080
```

- 只在内网使用时，把 `localport=8080` 那条加上 `remoteip=192.168.0.0/16` 限制来源
- **数据库 3306 绝对不要对公网开放**。如需远程连接，只放行内网段
- 若 80 端口被 IIS 占用，要么停掉 IIS，要么把 `config.yaml` 的 `port` 改成 80 并处理冲突

---

## 九、安全加固（上线前必须做完）

- [ ] 换掉 `config.yaml` 里的 `jwt.secret`（第五节）
- [ ] MySQL 用专用账号 `jiayuan`，不用 root；密码为强密码
- [ ] **登录管理端后立即修改管理员密码**（默认 `10000 / admin123`）
- [ ] 删除或修改演示账号密码（如 `10007 / admin123`）
- [ ] `config.yaml` 不要提交进 Git（确认 `.gitignore` 已覆盖）
- [ ] 3306 不对公网开放
- [ ] 关闭不必要的 Windows 服务（如 IIS、打印服务）
- [ ] 如对外网开放，建议前置一层 HTTPS 反代；注意 **2012 R2 不支持 TLS 1.3**，且需确认 TLS 1.2 已启用、SSL3/TLS 1.0/1.1 已禁用

---

## 十、上线验收清单

逐条打勾，全部通过才算部署完成：

- [ ] `sc query qqjiayuan` → `RUNNING`
- [ ] `http://<服务器IP>:8080/api/health` 返回 `code: 0`
- [ ] `http://<服务器IP>:8080/` 用户端首页正常，**图片不裂**（验证 `web/dist/static` 拷全了）
- [ ] `http://<服务器IP>:8080/admin-ui/` 管理端可打开（**末尾斜杠不能省**）
- [ ] 用 `10000 / admin123` 能登录管理端
- [ ] 用户端能注册新账号、能登录、能发帖、能签到
- [ ] 用户端 `/#/games` 游戏大厅能看到「幻想西游」，点进去能创建角色
- [ ] 页面上的中文**全部正常显示**，无乱码、无 `????`
- [ ] 服务器重启后服务自动恢复，数据仍在
- [ ] `D:\qqjiayuan\logs\server-err.log` 无 panic

---

## 十一、故障排查表

| 现象 | 最可能原因 | 处理 |
|------|-----------|------|
| 服务启动后立刻停止 | 工作目录不对，读不到 `config.yaml` | 检查 NSSM 的 `AppDirectory`；或 `config.yaml` 改绝对路径 |
| 控制台「读取配置失败」 | 同上 | 同上 |
| API 正常但前端整站 404 | `web_dir` 指向的目录不存在 → 静态托管被静默跳过 | 核对路径拼写；确认 `D:\qqjiayuan\web\dist\index.html` 真实存在 |
| 页面能开但图片大面积裂图 | `web/dist/static` 没拷全 | 重新完整拷贝 `web/dist` |
| 所有中文变成 `????` | 导入数据库时漏了 `--default-character-set=utf8mb4` | 重新导入 |
| 数据库连不上 | 未装 VC++ Redistributable / MySQL 服务未启动 / 密码错 | 装 VC++ 后重启；`sc query mysql`；核对 `config.yaml` |
| `server.exe` 双击闪退 | 缺少运行环境或直接崩溃 | 在 CMD 里运行看报错原文 |
| 时间戳全部偏移几小时 | 服务器时区不对 | `tzutil /s "China Standard Time"` 后重启服务 |
| `node -v` 报错找不到入口点 | 装的还是 Node 18+ 的版本 | 卸干净后重装 **16.20.2** |
| `node -v` 仍是 `v22.x` | 旧版本残留 | 控制面板卸载 + 删 `C:\Program Files\nodejs`、`%APPDATA%\npm` 后重装 |
| `go version` 无法执行 / 最小程序跑不起来 | Go 1.26 官方不支持 2012 R2 | 走 4.1 路线 B（开发机交叉编译）；或按附录 A 用 go-win7 补丁版工具链 |
| 前端构建报 `error:0308010C` | OpenSSL 兼容问题 | 构建前执行 `set NODE_OPTIONS=--openssl-legacy-provider` |
| 日志文件里中文乱码 | 用 936 代码页的控制台查看 UTF-8 日志 | 先 `chcp 65001` 再 `type` 日志，或直接用编辑器打开 |

---

## 十二、日常运维

**备份**（建议加进任务计划，每天凌晨执行）：

```cmd
if not exist D:\backup mkdir D:\backup
for /f %%i in ('powershell -Command "Get-Date -Format yyyyMMdd"') do set TODAY=%%i
mysqldump --default-character-set=utf8mb4 -u jiayuan -p qq_jiayuan > D:\backup\qq_jiayuan_%TODAY%.sql
```

> 用 PowerShell 取日期而不是 `%date%` 截取，是为了避开中文版 `%date%` 格式差异导致的文件名错乱。
> `--default-character-set=utf8mb4` 同样不能省，否则备份文件里的中文已经是坏的了。

**升级流程**：

1. 停服务：`nssm stop qqjiayuan`
2. 备份数据库
3. 覆盖 `server.exe`、`web/dist`、`admin-web/dist`
4. 起服务：`nssm start qqjiayuan`
5. 看 `server-err.log`，确认无 panic

**回滚**：保留上一个版本的 `server.exe` 和 `dist` 目录，回滚即覆盖回去 + 恢复数据库备份。

> 数据库结构升级由后端启动时的 `AutoMigrate` 自动完成（只加不删，幂等）。但**回滚代码不会回滚表结构**，所以涉及表结构变更的版本，回滚时必须一并恢复数据库备份。

---

## 附录 A：如果实测发现 Go 跑不起来

按优先级从高到低选：

1. **换系统到 Server 2016 / 2019 / 2022**（最省事，强烈推荐）
   2012 R2 下个月 ESU 就到期，而且 Go 这一项无论如何都绕不开"官方不支持"。在这上面省下的时间，会以"排查诡异运行时崩溃"的形式加倍还回来。

2. **用打了旧 Windows 兼容补丁的 Go 工具链重新编译后端**
   - [XTLS/go-win7](https://github.com/XTLS/go-win7)
   - [thongtech/go-legacy-win7](https://github.com/thongtech/go-legacy-win7)

   它们专门回退了 Win7/8/8.1/Server 2012(R2) 的兼容性补丁（`ProcessPrng` → `RtlGenRandom`、`WSA_FLAG_NO_HANDLE_INHERIT` 回退、控制台句柄、PE 头版本号等），目前跟到 go1.27.1。
   用它在开发机交叉编译出的 `server.exe`，在 2012 R2 上的成功率明显更高。

3. **把后端挪到 Linux**
   2012 R2 上只留 MySQL，或者数据库也一起挪走，Windows 这台只当客户端。

---

## 附录 B：生产启动脚本 `start-prod.bat`

项目自带的 `start.bat` **不能用于生产** —— 它会尝试在服务器上执行 `npm install` 和 `go build`。生产用下面这个（放在 `D:\qqjiayuan\`）：

```bat
@echo off
chcp 65001 >nul
title QQ JiaYuan Server
cd /d "%~dp0server"

echo ==========================================
echo   QQ JiaYuan - Production
echo   Listening on http://127.0.0.1:8080
echo   Press Ctrl+C to stop
echo ==========================================
echo.

if not exist "server.exe" (
    echo [ERROR] server.exe not found in %CD%
    pause
    exit /b 1
)
if not exist "config.yaml" (
    echo [ERROR] config.yaml not found in %CD%
    pause
    exit /b 1
)

server.exe -config "%~dp0server\config.yaml"
pause
```

> 脚本输出刻意用英文：2012 R2 中文版的 936 代码页配合 `chcp 65001` 处理 `.bat` 内的中文容易乱码，生产脚本保持纯 ASCII 最稳。
> 如果 `.bat` 里确实要写中文，请把文件另存为 **ANSI/GBK** 编码，不要用 UTF-8。

---

## 附录 C：版本与支持状态速查

### 服务器该装的版本

| 组件 | 版本 | 官方支持 2012 R2 |
|------|------|----------------|
| Go | **1.26.6** `windows-amd64` | ❌ 官方不支持（**但可能实际能跑**，必须实测；且不装也行，见第零节） |
| Node.js | **16.20.2** `x64` | ✅ Tier 1 支持 |
| npm | 8.19.4（随 Node 16） | ✅ |
| MySQL | **8.0.x** | ✅ 支持 |
| Git | 2.24.0+ | — |
| VC++ 运行库 | 2015–2022 x64 | — |

### 为什么是这些版本

| 组件 | 依据 |
|------|------|
| Node.js **16.20.2** | Node 官方 BUILDING.md：Windows x64 最低 `>= Windows 8.1/2012 R2`，Tier 1 |
| Node.js 18+ | Node 18 的 BUILDING.md 改为 `>= Windows 10/Server 2016`，2012 R2 降为"实验性" |
| MySQL **8.0.x** | 支持范围覆盖 Windows 7 / Server 2008 R2 及更高 |
| MySQL 8.4 / 9.x | 官方支持列表只有 Server 2016 / 2019 / 2022 |
| **Go 1.26.6** | `go.mod` 要求 `go 1.26`；而最后一个支持 2012 R2 的 Go 是 **1.20**（1.21 起移除支持）——两个条件没有交集 |

### 端口与服务

| 项目 | 地址 |
|------|------|
| 后端 API | `http://<服务器IP>:8080/api/` |
| 用户端 | `http://<服务器IP>:8080/` |
| 管理端 | `http://<服务器IP>:8080/admin-ui/`（**末尾斜杠不能省**） |
| MySQL | `127.0.0.1:3306`（**不要对公网开放**） |

### 系统支持时间线

| 事项 | 日期 |
|------|------|
| Windows Server 2012 R2 常规支持结束 | 2023-10-10 |
| Windows Server 2012 R2 付费 ESU 到期 | **2026-10-13** |

---

## 附录 D：开发调试脚本（服务器上跑 dev 模式）

生产部署用本文档正文的流程。如果你想在服务器上以**开发模式**（热重载）跑起来调试，用这两个脚本，放在 `qqjiayuan` 根目录：

| 脚本 | 作用 |
|------|------|
| `start-dev-win2012.bat` | 启动后端 :8080（air 热重载）+ 用户端 :8000 + 管理端 :8001，各开一个窗口 |
| `stop-dev-win2012.bat` | 停止上述三个服务 |

**用法**：右键 → **以管理员身份运行**（否则防火墙规则加不了、taskkill 可能被拒）。

`start-dev-win2012.bat` 会自动做这些事：检查 go/node/npm 是否在 PATH → 配 goproxy.cn / npmmirror 镜像 →
把 `%GOPATH%\bin` 加进 PATH → 缺 `node_modules` 时 `npm install` → 检测 air（没有就装，装不上则回退 `go run .`）→
加防火墙规则 → 开三个窗口启动服务。前端带 `--host 0.0.0.0`，可以从别的机器用 `http://服务器IP:8000` 访问。

`stop-dev-win2012.bat` 按「窗口标题 → air.exe → 监听端口」三轮停服务。**先杀 air 再杀端口**这个顺序不能反 ——
否则 air 会在你杀掉端口后立刻重新编译并把后端拉起来。它**不会**动生产服务，停生产服务请用 `nssm stop qqjiayuan`。

> 这两个脚本是**纯 ASCII + CRLF** 写的。这是刻意的：cmd.exe 用 OEM 代码页（中文版 936）读 `.bat`，
> 里面写 UTF-8 中文会乱码甚至破坏解析。改脚本时请保持纯 ASCII。
>
> 如果 `QQJY-backend` 窗口显示报错而不是启动横幅，说明 Go 二进制在这台机器上跑不起来 → 看附录 A。

---

## 附录 E：随文档附带的文件清单

### 要拷到服务器的可执行文件（放在 `D:\qqjiayuan\server\` 下）

| 文件 | 大小 | 作用 | 先试哪个 |
|------|------|------|---------|
| `server-2012r2.exe` | 约 28.6 MB | 后端服务 | ✅ **先试这个** |
| `dbinit-2012r2.exe` | 约 10.8 MB | 数据库初始化工具 | ✅ **先试这个** |
| `server.exe` | 约 28.6 MB | 后端服务（官方 Go 编译） | 上面那个跑不起来时换它 |
| `dbinit.exe` | 约 10.8 MB | 数据库初始化工具（官方 Go 编译） | 同上 |

两组文件的差别**只在编译用的 Go 工具链**：

- 带 `-2012r2` 的用**打了旧 Windows 兼容补丁**的 Go 编译，含三处对 2012 R2 有实际意义的回退：
  - `net` —— socket 创建时 `WSA_FLAG_NO_HANDLE_INHERIT` 被拒则回退（**最关键，直接决定 HTTP 监听能否建立**）
  - `runtime` —— `ProcessPrng` 不可用时回退 `RtlGenRandom`
  - `os` —— 目录枚举回退（`seed` 扫 `static/picture`、`static/image` 时会用到）
- 不带后缀的用官方 Go 编译，作为兜底。

四个都是 `PE32+ x86-64 console` + `CGO_ENABLED=0` 纯静态，服务器上不需要任何 C 运行库。

### 放在 `qqjiayuan` 根目录的脚本

| 文件 | 作用 |
|------|------|
| `build-web.bat` / `build-web.sh` | 构建两个前端产物（`web/dist` + `admin-web/dist`）。Windows 用 `.bat`，macOS/Linux 用 `.sh` |
| `start-dev-win2012.bat` | dev 模式启动（air 热重载 + 两个前端 dev server） |
| `stop-dev-win2012.bat` | 停止 dev 模式的服务 |

### 最小启动顺序

```
1. 装 MySQL 8.0 + VC++ 2015-2022 运行库
2. 建 config.yaml（第五节，注意用绝对路径 + 换掉 jwt secret）
3. dbinit-2012r2.exe     ← 建库 + 建表 + 种子数据，看到"初始化完成"
4. 建 jiayuan 账号，把 config.yaml 的 user/password 改用它（3.3）
5. server-2012r2.exe     ← 起服务，看到启动横幅就成功
6. 浏览器开 http://服务器IP:8080/
```

> 第 6 步之前，`web/dist` 和 `admin-web/dist` 必须已经构建好（否则页面打不开）。
> 在项目根目录跑 `build-web.bat`（Windows）或 `./build-web.sh`（macOS/Linux）即可，它会依次构建两个前端。
> **首次构建要先 `npm install`，比较慢**；之后改代码只需重跑这个脚本。
>
> 不想构建也可以先用 dev 模式验证：跑 `start-dev-win2012.bat`，它用 dev server 实时编译，不需要 `dist`。

> 第 5 步之后如果要配成 Windows 服务，用附录 B 的 `start-prod.bat` 或第七节的 NSSM。
> 附录 B 的脚本里写死了 `server.exe` 这个文件名 —— 如果你实际用的是 `server-2012r2.exe`，
> 要么把它重命名成 `server.exe`，要么改脚本里那两处文件名。
