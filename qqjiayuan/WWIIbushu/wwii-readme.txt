============================================
二战征途 独立部署说明(1 区 + 2 区)
============================================

【为什么独立部署】
  二战内容(配置/玩法/前端)改动只重打二战包、重启二战服,
  家园服全程不用动 → 玩家无感知、不停服。
  支持开多区: 每区一个二战服实例 + 一个独立二战库。

【架构与数据库】
  家园服(home, 8080): 账号/家园/管理端;二战管理端路径反代到二战服;不连二战库
  二战服(wwii, 8081=1区 / 8082=2区): 只跑二战;连家园库(只读)+ 二战独立库(读写)
    1 区二战库: qq_ezzt
    2 区二战库: qq_ezzt2   ★ 独立,与 1 区不共用(2 区开区用 bootstrap=fresh,绝不搬 1 区数据)
    家园库    : qq_jiayuan ★ 全区共用(账号/昵称/管理端权限仍在家园)
  二战服启动时自动建库(空库建表+灌种子),无需手工建表。

【一键部署(Mac / Windows 打包机)】
  1. 打包(产出 Linuxbushu + WWIIbushu 两个包):
     Windows: .\pack-linxu.bat
     Mac:     ./pack-linxu.sh
  2. 部署家园包到主力机器(可选辅助机器),照旧:
     .\tools\deploy.bat -p '密码' -h 124.222.208.92 --no-backup
  3. 部署二战 1 区(不碰家园,家园不停服):
     Windows: .\tools\deploy-wwii.bat -p '密码' -h 124.222.208.92 --no-backup
     Mac:     ./tools/deploy-wwii.sh -p '密码' -h 124.222.208.92 --no-backup
  4. 开 2 区(独立库 qq_ezzt2 + fresh,自动克隆并启动 /opt/wwii-2):
     Windows: .\tools\deploy-wwii.bat -p '密码' -h 39.105.151.141 --zone2 --no-backup
     Mac:     ./tools/deploy-wwii.sh -p '密码' -h 39.105.151.141 --zone2 --no-backup
  (2 区也可在服务器二战目录里手动跑 ./开2区.sh 或 ./zone2.sh,作用相同; Windows 打包的包内只含 zone2.sh 别名)

【1 区部署(二战独立包 WWIIbushu,手动方式)】
  1. 解压: tar -xzf WWIIbushu.tar.gz -C /opt   (得到 /opt/WWIIbushu)
  2. 编辑 /opt/WWIIbushu/server/config.yaml:
     - mysql.password 填 MySQL root 密码(家园库 qq_jiayuan,可为远程 IP)
     - 确认 ezfy_mysql.dbname = qq_ezzt(1 区独立库,自动建库)
     - 确认 ezfy_mysql.bootstrap = migrate(空库且家园库有残留二战数据时自动搬库;已是老库则无害)
     - ★ jwt.secret 必须与家园包 server/config.yaml 完全一致(打包时已自动对齐)
     - server.web_dir 已指向 /opt/WWIIbushu/web/dist,换目录请同步改
  3. 启动: chmod +x *.sh server/server && ./start.sh   (二战 1 区 http://127.0.0.1:8081)
  4. 家园服(Linuxbushu 包)照常启动(模式 home,配置不用改)。
  5. 家园服管理端『游戏大厅』确认二战区服条目: 1 区 host=http://服务器IP:8081, status=1

【2 区开区】
  1. 在已部署好的 1 区二战目录执行: ./开2区.sh (包内还有 ASCII 别名 ./zone2.sh,同一脚本,Windows 产物用这个)
     默认: 目标 /opt/wwii-2、端口 8082、库 qq_ezzt2、区名【2区】诺曼底登陆
     可带参: ./开2区.sh /srv/wwii-2 8082 qq_ezzt2 诺曼底登陆
     脚本会自动: 复制目录 + 改 port/zone_id/zone_name/zone_sub + 改 ezfy_mysql.dbname + bootstrap=fresh
  2. 编辑 /opt/wwii-2/server/config.yaml 再确认:
     - ezfy_mysql.dbname = qq_ezzt2(独立库,与 1 区 qq_ezzt 不共用)
     - ezfy_mysql.bootstrap = fresh(★ 必须!否则空库会被误搬 1 区数据)
     - server.zone_name = 2区, server.zone_sub = 诺曼底登陆 → 标题「二战征途-【2区】诺曼底登陆」
     - jwt.secret 与家园一致
  3. 启动: /opt/wwii-2/start.sh  (二战 2 区 http://127.0.0.1:8082)
  4. 家园服管理端『游戏大厅』给二战加 2 区服条目:
     name=二战征途-【2区】诺曼底登陆, host=http://服务器IP:8082, path=/games/ezfy, status=1
     (sort 越大越靠后;玩家默认进第一个=1 区)
  5. 玩家从家园游戏列表进二战 → 守卫读 /api/ezfy/zone 跳默认区;
     管理端二战菜单顶栏『管理区』下拉可切 2 区。
     ★ 2 区玩家在二战首页刷新只显示本区(二战服 zone 接口只下发本区),不会闪出 1 区信息。

【不停服更新二战】
  只改二战内容 → 重打/重传二战包(或只传新 server 二进制 + web/dist)
  → 二战服 ./stop.sh && ./start.sh
  → 家园服全程不用动,玩家无感知。
  (家园包有更新才需要重启家园服;两个包是两个独立部署,不要求同版本。)

【排障】
  - 二战服起来但管理端二战菜单 502: 看家园日志「二战管理端反代失败」,
    检查 game_servers 的 host 是否可达(能从家园机访问到二战服端口)。
  - SSO 跳转后一直回登录页: 两边(家园/二战)jwt.secret 不一致。
  - 2 区出现 1 区数据: 开区时 qq_ezzt2 已有数据,或 bootstrap 不是 fresh。
  - 二战服起不来: 看 /opt/WWIIbushu/logs/server.log(家园库连不上会 Fatal)。
  - 二战服端口被占用: 改 config 的 server.port 换端口。
