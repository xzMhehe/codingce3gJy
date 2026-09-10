# 商城复刻计划(诺哈商店中心风格)

## Context

用户要求按 `/Users/mxz/Downloads/诺哈三代破解版`(ASP+WML 老站源码)复刻商城与道具商城,认为现在的实现"不对"。经调研确认:

- 诺哈商店(`wap/home/money/shop_*.asp`)是**商店中心**:平铺编号列表 → 详情(销售价格/库存数量/销售数量/销售时间/结束时间 + `购买.赠送`)→ 数量+支付密码两步确认 → 扣库存加销量;支持**赠送**给指定会员并发站内信。
- 用户已确认范围:**① 道具商城照搬诺哈风格重做**(商品仍是鲜花/改名卡等道具,复用 goods/user_goods 表,购买仅 G币支付);**② 新增诺哈原味「货币商店」**(新表 money_shop,花一种货币买另一种货币礼包);③ 财务中心加【购买货币】入口;④ 列表形式照搬诺哈平铺编号列表(无分类筛选)。
- 项目已有支付密码体系(users.pay_pass bcrypt + Security.vue),购买/赠送需校验。
- 管理端 AdminGoods.vue 目前是死代码(Dashboard 已 import 但无渲染槽、无菜单),需一并接通。

## 一、后端(server/)

### 1. 模型与迁移
- [model/good.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/model/good.go):`Good` 补 `Stock int`、`Sales int`、`EndTime *time.Time`(NULL=长期);GoodPresets 补 Stock 值。
- [model/economy.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/model/economy.go):新增 `MoneyShop` 模型(表 `money_shop`),字段:`Name/MType(column:mtype,coins|yuanbao|jinzuan|youquan)/Money(每份数量)/PType(column:ptype)/Price/Stock/Sales/Status(1上架,沿用项目约定)/AddTime/EndTime`。
- [seed/seed.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/seed/seed.go):
  - AutoMigrate 加 `&model.MoneyShop{}`;
  - HasColumn 模式为 goods 补列 `stock`(首次加列时回填 999)/`sales`/`end_time`;
  - 新增 `seedMoneyShop(db)` 幂等补种,示例礼包:1000G币礼包(卖10元宝)、10000G币豪华礼包(88元宝)、10元宝特惠包(10000G币)、100元宝礼包(100000G币)、5张友友券礼包(2元宝)、1金钻礼包(50元宝);EndTime 统一 2027-12-31。

### 2. 支付密码 helper
- [handler/user.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/user.go) PayPassSet 后新增 `verifyPayPass(db, uid, payPass) string`:未设置→"您还未设置支付密码,请先到安全中心设置!";bcrypt 不匹配→"支付密码错误!"。

### 3. 改造 handler/good.go(GoodHandler)
- `List` 重写:去分类;`status=1 AND (end_time IS NULL OR end_time>NOW())`,`Order("id DESC")`,`pageOf(c,10)`;保留余额返回,删 categories。
- 新增 `Detail`(`GET /goods/:id`,公开):404 文案"商品不存在或已下架"。
- `Buy` 重写:入参 `{num, pay_pass}`(删 currency 分支,仅 G币)。校验顺序(文案照诺哈):商品存在→上架→未过结束时间("商品已过销售时间!")→verifyPayPass→`num<1`("购买数量错误!")→`num>Stock`("购买数量超过库存数量!")→余额("您的G币不足!")→执行:gorm.Expr 原子扣款+addWalletLog(复用 kind "buy")+入 UserGood+`stock-=num, sales+=num`。
- 新增 `SendPreview`(`GET /goods/:id/send-preview?to=&num=`,认证):校验商品/号码/数量/库存,返回昵称+总价,不扣款。
- 新增 `Send`(`POST /goods/:id/send`,认证,`{to, num, pay_pass}`):校验链同上 + "请填写赠送号码!"/"会员号码不正确!"/"不能赠送给自己!";执行:扣款流水、道具入对方背包(user_goods upsert)、库存销量、PrivateMessage 站内信"恭喜!赠送了您N个「商品名」。"。
- `AdminList/AdminCreate/AdminUpdate` 补 stock/sales/end_time 字段映射。

### 4. 新增 handler/money_shop.go(MoneyShopHandler)
结构照 good.go:List(公开,10/页,id 倒序,登录返回四币种余额)、Detail(公开)、Buy(`{num, pay_pass}`,按 PType 校验对应余额,currencyName 用 economy.go 已有函数;扣 PType 加 MType 各一笔+两笔流水)、Send(对方钱包直接加款+流水+站内信"恭喜!赠送了您N(货币名)。")、SendPreview、AdminList/AdminCreate/AdminUpdate/AdminDelete(`MType/PType 必须∈四币种`)。Buy/Send 加 verifyTime 3 秒频控(替代诺哈 VMoney 防重)。

### 5. Wallet 接口 + 路由
- [handler/economy.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/economy.go) Wallet 响应加 `money_shop`(最新5个在售货币商品)。
- [router/router.go](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/router/router.go):公开组加 `GET /goods/:id`、`GET /money-shop`、`GET /money-shop/:id`;认证组加 `POST /goods/:id/send`、`GET /goods/:id/send-preview`、`POST /money-shop/:id/buy|send`、`GET /money-shop/:id/send-preview`;admin 组加 `/money-shops` CRUD(perm admin:access)。

## 二、web 前端(Vue2.7 WAP 风格)

### 1. 重写 [Shop.vue](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/web/src/views/Shop.vue)(`?view=` 多视图,同 Wallet.vue 模式)
- **list**:编号连续(`i+1+(page-1)*10`)逐条 `N.<a>商品名</a>`;空列表"暂无销售。";分页照诺哈:`下页.上页` + `(第X/Y页/共N记录)` + `第[input]页 [前往]`;顶部保留我的G币/[我的仓库];删除分类导航/图标/行内购买。
- **detail**:`<b>商品名</b>`/`销售价格：20(G币)`/`库存数量：N`/`销售数量：N`/`销售时间：`/`结束时间：` + `购买.赠送` + `----------`。
- **buy**:`【购买道具】`;先查 `/me/paypass`,未设置显示"您还未设置支付密码,请先到安全中心设置!"并阻断;数量+支付密码输入 → **buyOk** 视图(购买数量/购买总价 → `确定购买`)→ 成功"购买成功!"(提交期禁用按钮)。
- **send**:`【赠送道具】`;会员号码+数量+支付密码 → send-preview 校验 → **sendOk** 视图(`赠送会员：昵称 (号码)`/`赠送金额：N个(商品名)`/`购买总价`)→ `确定赠送` → "赠送成功!"。
- 面包屑 `家园>用户中心>我的钱包>商店`,样式沿用现有 class,不新增样式。

### 2. 新增 MoneyShop.vue
同骨架,接口 `/money-shop` 系列;detail 多一行 `销售货币：N(货币名)`;标题【购买货币】/【赠送货币】;路由 `/money-shop`(router/index.js 注册,meta auth)。

### 3. Wallet.vue + 导航
- Wallet.vue G币兑换区块后插【购买货币】区块:`1.<a>商品名</a>` 最新5条(数据来自 /api/wallet 的 money_shop)。
- Nav.vue 商城行加"货币商店"入口;YouQuan.vue 中"支持友友券支付"计数文案调整(新购买流程仅 G币)。

## 三、admin-web(ElementUI)

- [AdminGoods.vue](file:///Users/mxz/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/src/components/admin/AdminGoods.vue):表格+表单补 库存/销量/结束时间(留空=长期)。
- 新增 AdminMoneyShop.vue(以 AdminGoods 为模板,含四币种 el-select)。
- Dashboard.vue:补 `<admin-goods>`/`<admin-money-shop>` 渲染槽(接通死代码);App.vue 菜单加"道具商城管理""货币商店管理"。

## 实施顺序
后端模型/迁移/种子 → handler(helper→good→money_shop→economy) → 路由 → web(Shop→MoneyShop→Wallet→router→Nav) → admin-web → 重启服务验证。

## 验证
- 重启后端(Go 1.26.8: `export PATH="$HOME/go-sdk/go1.26.8/bin:$PATH" && go run .`,qqjiayuan/server);web(8000)、admin-web(Node 19)已运行,热更新生效。
- 接口:`GET /api/money-shop`、`GET /api/goods`(无 categories、id 倒序、含 stock/sales/end_time)、`GET /api/wallet`(含 money_shop);购买货币后查 users 双币种变动+wallet_logs 两行+money_shop 库存销量;购买道具查 user_goods 叠加;赠送后对方背包/钱包入账、/messages 收到站内信。
- 浏览器:用户端 10007/admin123(先在 /security 设置支付密码),走完 列表→详情→购买→赠送 与 货币商店全流程;管理端 10000/admin123 管理两商店。
- 边界:未设支付密码/密码错误/数量0/超库存/余额不足/空号码/不存在号码/赠送自己/过期商品/3秒连点,文案逐条对照上表。

## 风险
- 新购买流程仅 G币(youquan_price 字段保留但 UI 移除友友券支付,YouQuan.vue 文案同步调整)。
- goods.stock 只在首次加列时回填 999,不覆盖管理员后续设置。
- AdminGoods 原为死代码,必须同时补 Dashboard 渲染槽+App 菜单才可见。
- 扣款/库存用 gorm.Expr 原子更新;校验-扣减非事务,极端并发可能轻微超卖(可后续包 db.Transaction,不阻塞本期)。
