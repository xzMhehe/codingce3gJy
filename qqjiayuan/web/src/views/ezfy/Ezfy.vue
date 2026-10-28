<template>
  <div class="ezfy-page">
    <!-- ★ 2026-10-12 二战独立登录：未登录（无二战 ezfy_token）时游戏整体不渲染，只显示下面二战自己的登录界面。
         登录走二战专属 /api/ezfy/login（首登同步家园账号），token 单独存，与家园彻底分离。 -->
    <!-- ★★ 2026-10-15 家园已登录 → 先走 SSO 静默换二战登录态，这期间显示轻量占位，不闪登录框
         （原来会先渲染出二战登录框、SSO 成功后再整页刷新 —— 玩家进二战会看到「登陆掠影」）。
         ssoPending 在 data() 初始化（首帧就占位，连一帧登录框都不会闪）。 -->
    <div v-if="!isLogin && ssoPending" class="wwii-login"
         style="max-width:340px;margin:80px auto 0;padding:24px;background:#f8f8f9;border:1px solid #e0e0e0;border-radius:10px;box-shadow:0 2px 12px rgba(0,0,0,.08);">
      <div style="font-size:18px;font-weight:bold;text-align:center;margin-bottom:12px;">{{ zoneTitle }}</div>
      <div style="text-align:center;padding:14px 0 16px;color:#888;font-size:14px;">正在进入{{ zoneTitle }}…</div>
      <!-- ★★ 2026-10-18 手动入口**常显**（用户：「点此进入能不能别几秒后，直接就有」）。
           正常 SSO 几百毫秒就自动 reload 进游戏，这个链接只是一闪而过；
           真卡住时玩家第一眼就能点，不用干等。 -->
      <div style="text-align:center;padding:0 0 4px;font-size:14px;">
        <a href="javascript:;" style="display:inline-block;padding:2px 10px;color:#2b6cb0;text-decoration:underline;" @click="manualEnter">点此进入</a>
      </div>
    </div>
    <div v-else-if="!isLogin" class="wwii-login"
         style="max-width:340px;margin:80px auto 0;padding:24px;background:#f8f8f9;border:1px solid #e0e0e0;border-radius:10px;box-shadow:0 2px 12px rgba(0,0,0,.08);">
      <div style="font-size:18px;font-weight:bold;text-align:center;margin-bottom:4px;">{{ zoneTitle }}</div>
      <!-- ★★ 2026-10-15 入口选服：目前单进程=单区（一区），先做「服务器选择」占位；
           以后二区/更多区各起一个实例后，这里会下发全区列表供切换 -->
      <div style="display:flex;align-items:center;margin-bottom:12px;">
        <span style="flex-shrink:0;color:#888;font-size:13px;margin-right:8px;">服务器</span>
        <select v-model="zoneSel" class="wwii-select"
                style="flex:1;box-sizing:border-box;height:34px;padding:0 8px;border:1px solid #e0e0e0;border-radius:4px;background:#fff;font-size:14px;color:#2b2b2b;outline:none;">
          <option v-for="z in zoneList" :key="z.id" :value="z.id">{{ z.name }}</option>
        </select>
      </div>
      <!-- class=wwii-input + 行内 style(含 !important) 双保险，保证账号/密码/验证码输入框宽高完全一致 -->
      <!-- ★★ 2026-10-18 防账号串台：家园已登录时这里锁定为**当前家园号**（只读灰底）。
           原来允许随便输另一个家园号 → 玩家「A 登录家园、二战里输 B」→ 二战身份变 B、
           家园 token 也被后端回传的 home_token 换成 B = 账号乱。 -->
      <input v-model="wwiiName" :readonly="!!homeUid" :placeholder="homeUid ? '当前家园账号（已锁定）' : '家园号'" class="wwii-input"
             :style="homeUid ? { background: '#f2f2f3', color: '#666', cursor: 'not-allowed' } : null"
             style="display:block;box-sizing:border-box;width:100% !important;max-width:100%;height:38px;padding:10px;border:1px solid #e0e0e0;border-radius:4px;background:#fff;font-size:14px;"
             @keyup.enter="wwiiLogin()"/>
      <input v-model="wwiiPwd" type="password" placeholder="家园密码" class="wwii-input"
             style="display:block;box-sizing:border-box;width:100% !important;max-width:100%;height:38px;padding:10px;border:1px solid #e0e0e0;border-radius:4px;background:#fff;font-size:14px;"
             @keyup.enter="wwiiLogin()"/>
      <!-- 密码错误满 3 次后展示验证码（算式图片，点了换一张） -->
      <div v-if="wwiiNeedCaptcha" style="display:flex;align-items:center;gap:8px;margin-bottom:10px;">
        <input v-model="wwiiCaptcha" placeholder="验证码" class="wwii-input"
               style="margin-bottom:0;box-sizing:border-box;width:100% !important;max-width:100%;height:38px;padding:10px;border:1px solid #e0e0e0;border-radius:4px;background:#fff;font-size:14px;"
               @keyup.enter="wwiiLogin()"/>
        <img v-if="wwiiCaptchaImg" :src="wwiiCaptchaImg" alt="验证码" title="点击换一张"
             style="height:38px;flex-shrink:0;border:1px solid #e0e0e0;border-radius:4px;background:#fff;cursor:pointer;"
             @click="loadWwiiCaptcha"/>
      </div>
      <div v-if="wwiiErr" style="color:#c0392b;font-size:13px;margin-bottom:10px;text-align:center;">{{ wwiiErr }}</div>
      <button @click="wwiiLogin()" :disabled="wwiiBusy" class="wwii-submit"
              style="width:100%;padding:10px;border:0;border-radius:6px;font-size:15px;cursor:pointer;">
        {{ wwiiBusy ? '登录中…' : '进入二战征途' }}
      </button>
    </div>
    <div class="home-wrap" v-if="isLogin">
      <div class="title-bar">{{ zoneTitle }}</div>

      <!-- 顶部导航(每页都有) -->
      <div class="top-nav">
        <!-- ★★ 2026-10-23 好友 / 邮箱 / 军情 共用同一个「裸感叹号」造型：
             亮黄 #FCE224 → 黄 #FED903 → 橙 #FDA504 竖向渐变 + 砖红 #8E2525 细描边。
             渐变只定义一次（放 0×0 的 svg 里），三个角标都引用它，避免重复 ID。 -->
        <svg class="ezfy-nav-warn-defs" aria-hidden="true" focusable="false"
             style="position:absolute;width:0;height:0;overflow:hidden">
          <defs>
            <linearGradient id="ezfyNavWarnGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#fce224"/>
              <stop offset="55%" stop-color="#fed903"/>
              <stop offset="100%" stop-color="#fda504"/>
            </linearGradient>
          </defs>
        </svg>
        <!-- ★ 2026-10-23 聊天频道（世界/军团/综合）没有「未读」概念，不放角标。
             ★★ 2026-10-23 用户纠正：「聊天不需要感叹号，是**好友**那块需要」——
             未读私聊的入口在「好友 → 会话列表」，所以感叹号挂在「好友」上。 -->
        <a href="javascript:;" @click="go('chat')">聊天</a>
        <!-- ★ 2026-10-23 邮箱：有未读邮件时显示感叹号（后端 /unread 的 mail） -->
        <a href="javascript:;" @click="go('mail')">邮箱<svg
          v-if="unreadMail > 0" class="ezfy-nav-warn" viewBox="0 0 8 14"
          preserveAspectRatio="none" role="img">
          <title>有 {{ unreadMail }} 封未读邮件</title>
          <rect x="2.4" y="1.75" width="3.2" height="7" rx="1.6"
                fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
          <circle cx="4" cy="11.4" r="1.75"
                  fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
        </svg></a>
        <!-- ★★ 2026-10-13 「有没看的预警或没看的被袭战报 → 军情后面加个感叹号图标」
             （类似首页公告行那个小喇叭）：未看数 = 未读的**军情警讯** + 未读的
             **被侦查/被掠夺/被征服/城破**战报，由 /view 的 `unread_warns` 下发
             （后端 `ezfyReportAlertCond`，只算被动挨打类；自己主动出击的战果不计入）。
             进过军情页把战报标已读后，下次 /view 刷新角标自动消失。 -->
        <a href="javascript:;" @click="go('reports')">军情<svg
          v-if="unreadWarns > 0" class="ezfy-nav-warn" viewBox="0 0 8 14"
          preserveAspectRatio="none" role="img">
          <title>有 {{ unreadWarns }} 条未看的军情警讯/被袭战报</title>
          <rect x="2.4" y="1.75" width="3.2" height="7" rx="1.6"
                fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
          <circle cx="4" cy="11.4" r="1.75"
                  fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
        </svg></a>
        <a href="javascript:;" @click="go('tasks')">任务</a>
        <!-- ★ 2026-10-23 好友：有未读**好友私聊**时显示感叹号（后端 /unread 的 chat）。
             私聊入口在「好友 → 会话列表」，所以角标挂这里（用户：聊天不需要，好友需要）。 -->
        <a href="javascript:;" @click="go('friends')">好友<svg
          v-if="unreadChat > 0" class="ezfy-nav-warn" viewBox="0 0 8 14"
          preserveAspectRatio="none" role="img">
          <title>有 {{ unreadChat }} 条未读私聊</title>
          <rect x="2.4" y="1.75" width="3.2" height="7" rx="1.6"
                fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
          <circle cx="4" cy="11.4" r="1.75"
                  fill="url(#ezfyNavWarnGrad)" stroke="#8e2525" stroke-width="1.15"/>
        </svg></a>
        <a href="javascript:;" @click="go('home')">首页</a>
      </div>

      <!-- 页面内操作结果（代替 alert 弹窗；原版本来就没有弹窗交互）
           ★ 用户反馈「提示都堆在页面顶部，看不出点了哪」→ 改为浮现在最近一次点击的附近，
           不再占一整行置顶。多条目向下轻微错开避免完全重叠。 -->
      <div v-for="(m, i) in msgs" :key="m.id" class="ezfy-msg" :class="'ezfy-msg-' + m.type"
           :style="msgStyle(m, i)">
        {{ m.text }}
        <a href="javascript:;" class="ezfy-msg-close" @click="closeMsg(m.id)">[关闭]</a>
      </div>

      <!-- 页面内确认条（代替 confirm/prompt 弹窗）
           ★ 2026-09-25 用户反馈「军官详情的一键穿戴，确认跑到页面最上面去了，要滚上去才能点」：
             改成和提示条(.ezfy-msg)同一套做法 —— **浮在刚才点的那个控件旁边**（位置由 askStyle 算），
             不再永远钉在页面顶部。DOM 位置留着不动，靠 position:fixed 浮起来。 -->
      <div class="ezfy-ask" v-if="askBox.show" ref="askBox" :style="askStyle">
        <div class="ezfy-ask-text">{{ askBox.text }}</div>
        <div class="ezfy-ask-row" v-if="askBox.input">
          <input v-model="askBox.value" :placeholder="askBox.placeholder" style="width:60%"/>
        </div>
        <div class="ezfy-ask-row">
          <a href="javascript:;" class="ezfy-ask-ok" @click="askConfirm(true)">[确定]</a>
          <a href="javascript:;" class="ezfy-ask-cancel" @click="askConfirm(false)">[取消]</a>
        </div>
      </div>

      <!-- 二级导航（资源/军官/军队/科技/城防/统帅）—— 只在对应页面显示，位置固定在顶部，不再有的在底部 -->
      <div class="old-line ezfy-subnav" v-if="showSubnav">
        <a href="javascript:;" :class="{ on: cur === 'builds' }" @click="go('builds')">资源</a>.
        <a href="javascript:;" :class="{ on: cur === 'acade' || cur === 'officerdetail' }" @click="go('acade')">军官</a>.
        <a href="javascript:;" :class="{ on: isArmyPage }" @click="go('troops')">军队</a>.
        <a href="javascript:;" :class="{ on: cur === 'techs' }" @click="go('techs')">科技</a>.
        <a href="javascript:;" :class="{ on: cur === 'defence' }" @click="go('defence')">城防</a>.
        <a href="javascript:;" :class="{ on: cur === 'info' }" @click="go('info')">统帅</a>
      </div>

      <!-- ============ 首页(cityHome) ============ -->
      <template v-if="cur === 'home'">
      <!-- ★ 2026-10-10 首页布局切换：homeLayout===2 → 老布局（保留原样）；否则 → 新布局（战争主题简约版，一屏放全）。
           ★ 2026-10-10 优化：老/新两套首页各自拆成独立子组件（modules/EzfyHomeOld.vue / modules/EzfyHomeNew.vue），
             与其它页面同一套 inject['ezfy'] 模式，内联大模板不再留在 shell，避免刷新时两个布局互相闪现的痕迹。 -->
      <!-- ★ 2026-10-10 防「首次进入(本地无缓存)新布局掠影」：本地没有布局缓存时，先不渲染首页，
           等 /view 下发权威布局（homeLayoutReady）后再渲染，避免「配置的是老布局却先画一帧新布局」。
           有缓存（含 seed 固定老布局的账号刷过一次之后）→ 首帧即渲染，不闪白。 -->
      <template v-if="homeLayoutReady">
        <ezfy-home-old v-if="homeLayout === 2"></ezfy-home-old>
        <ezfy-home-new v-else></ezfy-home-new>
      </template>
      </template>

      <!-- ============ 世界聊天(chat) ============ -->
      <!-- ============ chat —— 已拆到 modules/Ezfychat.vue ============ -->
      <ezfy-chat v-else-if="cur === 'chat'"></ezfy-chat>

      <!-- ============ 邮箱(mail) ============ -->
      <!-- ============ mail —— 已拆到 modules/Ezfymail.vue ============ -->
      <ezfy-mail v-else-if="cur === 'mail'"></ezfy-mail>

      <!-- ============ 情报/军情(reports) ============ -->
      <!-- ============ reports —— 已拆到 modules/Ezfyreport.vue ============ -->
      <ezfy-report v-else-if="cur === 'reports'"></ezfy-report>

      <!-- ============ 战报详情(reportview) ============ -->
      <!-- ============ reportview —— 已拆到 modules/Ezfyreport.vue ============ -->
      <ezfy-report v-else-if="cur === 'reportview'"></ezfy-report>

      <!-- ============ 好友(friends) ============ -->
      <!-- ============ friends —— 已拆到 modules/Ezfysocial.vue ============ -->
      <ezfy-social v-else-if="cur === 'friends'"></ezfy-social>

      <!-- ============ 任务(tasks) ============ -->
      <!-- ============ tasks —— 已拆到 modules/Ezfysocial.vue ============ -->
      <ezfy-social v-else-if="cur === 'tasks'"></ezfy-social>

      <!-- ============ 城市列表(cities) ============ -->
      <!-- ============ cities —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'cities'"></ezfy-city>

      <!-- ============ 资源详情(res/:type) ============ -->
      <!-- ============ res —— 已拆到 modules/Ezfybuild.vue ============ -->
      <ezfy-build v-else-if="cur === 'res'"></ezfy-build>

      <!-- ============ 建筑区(buildm/builds) ============ -->
      <!-- ============ 军事区 / 资源区(buildm / builds) 复刻 building/militaryIndex.html ============ -->
      <!-- ============ buildm/builds —— 已拆到 modules/Ezfybuild.vue ============ -->
      <ezfy-build v-else-if="cur === 'buildm' || cur === 'builds' || cur === 'bdetail'"></ezfy-build>

      <!-- ============ 建造页(buildpre) 复刻 building/preCreateMilitary.html / preCreateSource.html ============ -->
      <!-- ============ buildpre —— 已拆到 modules/Ezfybuild.vue ============ -->
      <ezfy-build v-else-if="cur === 'buildpre'"></ezfy-build>

      <!-- ============ 造兵(troop) ============ -->
      <!-- ============ troop —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'troop'"></ezfy-army>

      <!-- ============ 城防(defence) 复刻 city/troopDefence.html ============ -->
      <!-- ============ defence —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'defence'"></ezfy-army>

      <!-- ============ 军队总览(troops) ============ -->
      <!-- ============ troops —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'troops'"></ezfy-army>

      <!-- ============ 司令部(hq) ============ -->
      <!-- ============ hq —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'hq'"></ezfy-army>

      <!-- ============ 科技(techs) ============ -->
      <!-- ============ techs —— 已拆到 modules/Ezfytech.vue ============ -->
      <ezfy-tech v-else-if="cur === 'techs'"></ezfy-tech>

      <!-- ============ 科技详情(techpre) 2026-09-28 点[研究]进详情页查看资源/条件再确认 ============ -->
      <!-- ============ techpre —— 已拆到 modules/Ezfytech.vue ============ -->
      <ezfy-tech v-else-if="cur === 'techpre'"></ezfy-tech>

      <!-- ============ 地图(map) ============ -->
      <!-- ============ map —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'map'"></ezfy-map>

      <!-- ============ 军工厂(factory) 复刻 city/cityFactory.html ============ -->
      <!-- ============ factory —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'factory'"></ezfy-army>

      <!-- ============ 兵种详情(troopview) 复刻 city/cityTroopView.html ============ -->
      <!-- ============ troopview —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'troopview'"></ezfy-army>

      <!-- ============ 训练确认(trainpre) 复刻 city/createTroop.html / city/createDefence.html ============ -->
      <!-- ============ trainpre —— 已拆到 modules/Ezfyarmy.vue ============ -->
      <ezfy-army v-else-if="cur === 'trainpre'"></ezfy-army>

      <!-- ============ 目标详情(wildview) 复刻 map/mapView.html ============ -->
      <!-- ============ wildview —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'wildview'"></ezfy-map>

      <!-- ============ 出征确认(orderpre) ============ -->
      <!-- ★ 用户反馈「出征页面看着好乱」→ 按 ①兵力 ②军官 ③集结令 ④随军资源 ⑤宿营 ⑥计算 分区，
           每区一个小标题，输入项改成两列网格，整体高度比原来短很多。 -->
      <!-- ============ orderpre —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'orderpre'"></ezfy-map>

      <!-- ============ 战场指挥室(battle) 复刻《战斗机制》§1 ============
           入口：军情 → 军队动态 → [指挥]
           每回合 30 秒：前 25 秒可下达前进/暂停/后退，后 5 秒锁定并由服务器结算，最多 40 回合 -->
      <!-- ============ battle —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'battle'"></ezfy-map>

      <!-- ============ 命令详情 / 军队动态详情(orderview) ============ -->
      <!-- ============ orderview —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'orderview'"></ezfy-map>

      <!-- ============ 出征队列(orders) ============ -->
      <!-- ★ 2026-09-25 「出征队列按照军队动态那种展示」：
           数据源改成 /reports/dynamics（与「军情 → 军队动态」同一个接口、同一套字段），
           渲染样式也照抄军队动态的竖排块（命令/目标/状态/军官/时间/待带回/操作），
           比原来的 6 列表格信息全得多（原来没有待带回、没有指挥室入口）。
           ★ 同时修「刷新后消失」：本页数据在 go('orders') 里重新拉，
             不再依赖内存里的旧数组；命令详情页也带了 oid 到 URL（见 syncUrl/restoreFromUrl）。 -->

      <!-- ============ 计谋(scheme)：行军计谋专用页 ============ -->
      <!-- ★ 2026-09-30 军队动态/出征队列的 [计谋] 跳到这里，
           展示信号弹持有量与神兵天降/战略转移说明，对当前部队去程/回程使用；
           使用完返回「军情 → 军队动态」。 -->
      <!-- ============ scheme —— 已拆到 modules/Ezfyscheme.vue ============ -->
      <ezfy-scheme v-else-if="cur === 'scheme'"></ezfy-scheme>

      <!-- ============ orders —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'orders'"></ezfy-map>

      <!-- ============ 城市状态(citystatus) ============ -->
      <!-- ============ citystatus —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'citystatus'"></ezfy-city>

      <!-- ============ 附属野地(wilds) ============ -->
      <!-- ============ wilds —— 已拆到 modules/Ezfymap.vue ============ -->
      <ezfy-map v-else-if="cur === 'wilds'"></ezfy-map>

      <!-- ============ 召集(convene) ============ -->
      <!-- ============ convene —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'convene'"></ezfy-city>

      <!-- ============ 仓库调配(wareset) ============ -->
      <!-- ============ wareset —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'wareset'"></ezfy-city>

      <!-- ============ 安抚(placate) ============ -->
      <!-- ============ placate —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'placate'"></ezfy-city>

      <!-- ============ 税率(taxset) ============ -->
      <!-- ============ taxset —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'taxset'"></ezfy-city>

      <!-- ============ 改名(rename) ============ -->
      <!-- ============ rename —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'rename'"></ezfy-city>

      <!-- ============ 市政厅(cityhall) ============ -->
      <!-- ============ cityhall —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'cityhall'"></ezfy-city>

      <!-- ============ 市政厅→资源统计(srcstat) 复刻 city/citySourceList.html ============ -->
      <!-- ============ srcstat —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'srcstat'"></ezfy-city>

      <!-- ============ 市政厅→军队统计(troopstat) 复刻 city/cityTroopList.html ============ -->
      <!-- ============ troopstat —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'troopstat'"></ezfy-city>

      <!-- ============ 城市迁移(citymove) 复刻 city/cityHallMove.html ============ -->
      <!-- ============ citymove —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'citymove'"></ezfy-city>

      <!-- ============ 调整生产(sourceset) 复刻 city/sourceSet.html ============ -->
      <!-- ============ sourceset —— 已拆到 modules/Ezfycity.vue ============ -->
      <ezfy-city v-else-if="cur === 'sourceset'"></ezfy-city>

      <!-- ============ 军团(corps) ============ -->
      <!-- ============ corps —— 已拆到 modules/Ezfycorps.vue ============ -->
      <ezfy-corps v-else-if="cur === 'corps'"></ezfy-corps>

      <!-- ============ 排行(rank) ============ -->
      <!-- ============ rank —— 已拆到 modules/Ezfyrank.vue ============ -->
      <ezfy-rank v-else-if="cur === 'rank'"></ezfy-rank>

      <!-- ============ 背包(bag) ============ -->
      <!-- ============ bag —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'bag'"></ezfy-shop>

      <!-- ============ 宝物(treasure) ============ -->
      <!-- ============ treasure —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'treasure'"></ezfy-shop>

      <!-- ============ 商城(mall) ============ -->
      <!-- ============ mall —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'mall'"></ezfy-shop>

      <!-- ============ 商城·道具购买详情页（跳转新页面确认） ============ -->
      <!-- ============ mallbuy —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'mallbuy'"></ezfy-shop>

      <!-- ============ 商城·装备散件购买详情页 ============ -->
      <!-- ============ equipbuy —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'equipbuy'"></ezfy-shop>

      <!-- ============ 商城·宝箱开箱详情页 ============ -->
      <!-- ============ chestopen —— 已拆到 modules/Ezfyshop.vue ============ -->
      <!-- ★ 2026-10-09 新增宝箱内容页（chestview，查看奖池+单项详情） -->
      <ezfy-shop v-else-if="cur === 'chestopen' || cur === 'chestview'"></ezfy-shop>

      <!-- ============ 商城·宝箱购买页（chestbuy，买进背包不抽奖） ============ -->
      <ezfy-shop v-else-if="cur === 'chestbuy'"></ezfy-shop>

      <!-- ============ 我的待开宝箱（chestbag） ============ -->
      <ezfy-shop v-else-if="cur === 'chestbag'"></ezfy-shop>

      <!-- ============ 交易行(exchange) ============ -->
      <!-- ============ exchange —— 已拆到 modules/Ezfyshop.vue ============ -->
      <ezfy-shop v-else-if="cur === 'exchange'"></ezfy-shop>

      <!-- ============ 活动(activity) ============ -->
      <!-- ============ activity —— 已拆到 modules/Ezfyevent.vue ============ -->
      <ezfy-event v-else-if="cur === 'activity'"></ezfy-event>

      <!-- ============ 福利(welfare) ============ -->
      <!-- ============ welfare —— 已拆到 modules/Ezfyevent.vue ============ -->
      <ezfy-event v-else-if="cur === 'welfare'"></ezfy-event>

      <!-- ============ 公告(notices) —— 已拆到 modules/EzfyNotices.vue ============ -->
      <ezfy-notices v-else-if="cur === 'notices'"></ezfy-notices>

      <!-- ============ 联络中心(liaison) 复刻 liaison/liaisonIndex.html ============ -->
      <!-- ============ liaison —— 已拆到 modules/Ezfyevent.vue ============ -->
      <ezfy-event v-else-if="cur === 'liaison'"></ezfy-event>

      <!-- ============ 统帅(info) ============ -->
      <!-- ============ info —— 已拆到 modules/Ezfyacademy.vue ============ -->
      <ezfy-academy v-else-if="cur === 'info'"></ezfy-academy>

      <!-- ============ 他人统帅信息(playerinfo) 复刻 PlayerController.infoOther + user/info.html ============ -->
      <!-- 游戏是沉浸式的: 点玩家名只看这一页(二战风云的数据), 不允许跳去家园个人主页 /user/:id -->
      <!-- ============ playerinfo —— 已拆到 modules/Ezfyacademy.vue ============ -->
      <ezfy-academy v-else-if="cur === 'playerinfo'"></ezfy-academy>

      <!-- ============ 访问记录(visits) ============ -->
      <ezfy-academy v-else-if="cur === 'visits'"></ezfy-academy>

      <!-- ============ 军官/学院(acade) ============ -->
      <!-- ============ acade —— 已拆到 modules/Ezfyacademy.vue ============ -->
      <ezfy-academy v-else-if="cur === 'acade'"></ezfy-academy>

      <!-- ============ 军官详情(officerdetail) ============ -->
      <!-- ============ officerdetail —— 已拆到 modules/Ezfyacademy.vue ============ -->
      <ezfy-academy v-else-if="cur === 'officerdetail'"></ezfy-academy>

      <!-- ★ 2026-09-25：独立的「装备详情页」已删除。
           它原来只展示 名字/状态/属性 三行，且唯一的入口就是各装备表里的 [查看] 按钮；
           现在那三样都并进了「点装备名展开的详情卡」，页面没人能进 → 删掉，免得留死代码。
           （如果以后又要一个独立页，从 git 历史里捞回来即可。） -->

      <!-- 底部导航(每页都有, 复刻原版 cityHome.html 的 8+7 两行) -->
      <!-- ★ 2026-10-10 新首页布局：底部 16 项与首页主网格重复（有些还是电脑端功能）→ 首页新布局整块隐藏，
           老布局与其它页面保留 -->
      <template v-if="!isNewHome">
      <br/>
      <div class="old-line ezfy-bottom-nav">
        <a href="javascript:;" :class="{ on: cur === 'buildm' || (cur === 'bdetail' && buildInfoZone === 'm') }" @click="go('buildm')">军事</a>
        <a href="javascript:;" :class="{ on: cur === 'builds' || (cur === 'bdetail' && buildInfoZone === 's') }" @click="go('builds')">资源</a>
        <a href="javascript:;" :class="{ on: cur === 'map' }" @click="go('map')">地图</a>
        <a href="javascript:;" :class="{ on: cur === 'corps' }" @click="go('corps')">军团</a>
        <a href="javascript:;" :class="{ on: cur === 'rank' }" @click="go('rank')">排行</a>
        <a href="javascript:;" :class="{ on: cur === 'bag' }" @click="go('bag')">背包</a>
        <a href="javascript:;" :class="{ on: cur === 'mall' }" @click="go('mall')">商城</a>
        <a href="javascript:;" :class="{ on: cur === 'treasure' }" @click="go('treasure')">宝物</a>
      </div>
      <div class="old-line ezfy-bottom-nav">
        <a href="javascript:;" :class="{ on: cur === 'activity' }" @click="go('activity')">活动</a>
        <a href="javascript:;" :class="{ on: cur === 'welfare' }" @click="go('welfare')">福利</a>
        <a href="javascript:;" :class="{ on: cur === 'notices' }" @click="go('notices')">公告</a>
        <a href="javascript:;" :class="{ on: cur === 'exchange' }" @click="go('exchange')">交易</a>
        <a href="javascript:;" :class="{ on: cur === 'liaison' }" @click="go('liaison')">联络</a>
        <a href="javascript:;" :class="{ on: cur === 'cityhall' }" @click="go('cityhall')">市政</a>
        <a href="javascript:;" :class="{ on: cur === 'chat' }" @click="go('chat')">聊天</a>
        <!-- ★ 2026-09-24  底部导航「首页」换成「家园」(全局页脚已对沉浸式页面隐藏, 这里作为离开游戏的出口) -->
        <a href="javascript:;" @click="exitToHome()">家园</a>
      </div>
      </template>
      <hr/>
      <div>小Q报时：{{ nowText }}</div>
      <!-- <div>联系我们：QQ群 431442049</div> -->
    </div>
  </div>
</template>

<script>
import api from '../../api'

import { isAbsoluteHttpUrl, looksLikeBrokenZonePath } from '../../utils/zoneHost'
// ★★ 2026-10-18 从 JWT 里解 uid：跨端口时 user 对象读不到，只能靠 token 判断「家园是不是换了人」
import { jwtUid } from '../../store'

// ★ 资源显示名的「兜底默认值」。真正的名字由后端 /games/ezfy/res-cfg 下发
//   （管理端「资源管理 → 资源名称维护」可改），改名后全站展示跟随。
const RES_NAMES = { gold: '黄金', food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿' }
const RES_SHORT = { gold: '金', food: '粮', steel: '钢', oil: '油', rare: '稀' }

// 资源说明(黄金一段复刻原版 resourceA.html, 其余按同样语气补全)
// ★ 用 {key} 占位符, 由 applyResNames 把管理端配的资源名填进去 —— 改名后说明也跟着变
const RES_DES_TPL = {
  gold: '{gold}是二战世界里重要的交易货币，可以用来购买玩家出售的各种资源，也用于支付雇佣军官的薪资和招募费用，通过战争、税收、任务可获得。',
  food: '{food}是维持军队的根本，人口增长与部队训练都离不开它，军队每小时还会消耗{food}，一旦断粮部队将大量逃散，通过农田产出、掠夺、任务可获得。',
  steel: '{steel}是制造武器装备的基础材料，建造建筑、训练陆军部队、研究军事科技都需要消耗{steel}，通过炼钢厂产出、掠夺、任务可获得。',
  oil: '{oil}驱动着一切机械化部队，出征行军需要消耗{oil}，训练装甲与航空部队同样需要{oil}，通过石油基地产出、掠夺、任务可获得。',
  rare: '{rare}是尖端军事工业的原料，用于生产重型装备与高级兵种，产量稀少因此格外珍贵，通过稀矿厂产出、掠夺、任务可获得。'
}
function buildResDes (names) {
  const out = {}
  Object.keys(RES_DES_TPL).forEach(k => {
    out[k] = RES_DES_TPL[k].replace(/\{(\w+)\}/g, (m, key) => names[key] || m)
  })
  return out
}

export default {
  name: 'Ezfy',
  // ★ 2026-10-03 模块化：各页面模块拆到 ezfy/modules/*.vue，
  //   子组件通过 inject 拿回外壳实例（共享数据/方法仍集中在 shell，模块只负责模板）。
  provide () {
    return { ezfy: this }
  },
  components: {
    // ★ 2026-10-10 首页老/新布局各自独立成子组件（拆自 shell 内联大模板，避免刷新闪现痕迹）
    'ezfy-home-old': () => import('./modules/EzfyHomeOld.vue'),
    'ezfy-home-new': () => import('./modules/EzfyHomeNew.vue'),
    'ezfy-notices': () => import('./modules/EzfyNotices.vue'),
    'ezfy-chat': () => import('./modules/EzfyChat.vue'),
    'ezfy-mail': () => import('./modules/EzfyMail.vue'),
    'ezfy-report': () => import('./modules/EzfyReport.vue'),
    'ezfy-social': () => import('./modules/EzfySocial.vue'),
    'ezfy-build': () => import('./modules/EzfyBuild.vue'),
    'ezfy-city': () => import('./modules/EzfyCity.vue'),
    'ezfy-army': () => import('./modules/EzfyArmy.vue'),
    'ezfy-tech': () => import('./modules/EzfyTech.vue'),
    'ezfy-map': () => import('./modules/EzfyMap.vue'),
    'ezfy-scheme': () => import('./modules/EzfyScheme.vue'),
    'ezfy-corps': () => import('./modules/EzfyCorps.vue'),
    'ezfy-rank': () => import('./modules/EzfyRank.vue'),
    'ezfy-shop': () => import('./modules/EzfyShop.vue'),
    'ezfy-event': () => import('./modules/EzfyEvent.vue'),
    'ezfy-academy': () => import('./modules/EzfyAcademy.vue')
  },
  data () {
    return {
      cur: 'home',
      prevCur: 'home', // ★ 2026-09-29 上一页（各页 [返回] goBack 用）
      // ★ 2026-10-12 二战独立登录界面（未登录时根组件只渲染它）：
      // ★★ 2026-10-18 家园已登录 → 预填当前家园号（模板里配合 readonly 锁定，防账号串台）
      wwiiName: String(this.$store.getters.homeUid || ''),
      wwiiPwd: '',
      wwiiErr: '',
      wwiiBusy: false,
      // ★★ 2026-10-15 区名写活 + 入口选服：/api/ezfy/zone 下发本区信息（单进程=单区，先返回自己）。
      //   zoneTitle 为完整标题（如「二战征途-【1区】红色警戒」），zoneList/zoneSel 供登录页选服。
      zoneList: [],
      zoneSel: 1,
      // ★★ 2026-10-17 二战独立部署（多区）：兜底标题不再写死「1区」。
      //   原来写死 1 区名 → 2 区玩家刷新时接口没回来先闪「二战征途-【1区】红色警戒」掠影。
      //   现在优先用上次缓存的本区名（刷新首帧即正确），没有缓存回落通用游戏名。
      zoneTitle: localStorage.getItem('ezfy_zone_title') || '二战征途',
      // ★★ 2026-10-17 二战独立部署：家园服公网地址（zone 接口下发）。
      //   二战服是独立 SPA，「家园」按钮不能用站内路由（/home 不存在），要整页跳回家园服。
      homeUrl: '',
      ssoPending: !!this.$store.state.token,
      // ★★ 2026-10-18 SSO 请求序号：手动「点此进入」与自动 SSO 并发时只认最新一次响应
      ssoSeq: 0,
      wwiiCaptcha: '',        // 验证码答案
      wwiiCaptchaId: '',      // 验证码 ID
      wwiiCaptchaImg: '',     // 验证码 base64 图
      wwiiNeedCaptcha: false, // 密码错误满 3 次 → 显示验证码
      nowText: '', // ★ 页脚小Q报时(每秒刷新, 与 App.vue 同一格式)
      // ★ 2026-09-28 「累计采集/采集资源实时变化」：每秒本地 tick 的时间基准
      gatherNow: 0,
      // ★ 2026-09-28 「军队动态/出征队列」倒计时自动刷新：拉取 dynamics 时的本地时间戳，
      //   战斗中部队的本回合剩余是「相对剩余」，用它当基点往前推算。
      // ★ 2026-09-28 「倒计时归零 → 自动重拉」的三个运行态标记（详见 checkDueRefresh）：
      //   _dynRefreshing  : 本轮重拉是否还在进行（防同波多次触发）
      //   _dynRefreshedAt : 上次重拉的本地时刻（★ 2026-10-08 节流从 3s 收紧到 1s，减少「返航到点/出征抵达」刷新延迟）
      //   _dynFired       : 「订单id@到点时刻」→ **已重试次数**（★ 2026-10-08 由「是否已触发」改成计数，
      //                     见 checkDueRefresh：到点后最多重试 5 次，防后端结算慢导致界面卡住）
      //   _dynRetryAt     : 上次「到点重试」的本地时刻（重试间隔 2 秒）
      //     ★ 用 Object.create(null) 而不是 {} —— 它只是个计数表，不需要响应式，
      //       用 {} 会让 Vue 递归侦听每个动态加的 key，纯属浪费。
      //   ★★ 2026-10-11 修复「有战报但列表空」：Vue 2 的 initData 对 `_` 开头字段不 proxy 到实例，
      //       必须全部去掉 `_` 前缀（见 repCache 等字段同款说明）。
      dynAt: 0,
      dynRefreshing: false,
      dynRefreshedAt: 0,
      dynFired: Object.create(null),
      dynRetryAt: 0,
      // ★ 2026-10-11 军情分页 tab 切换流畅化（用户：「军队动态/驻军/军情警讯/战斗报告/军团战报
      //   这些 tab 切换还是卡顿」）。原来每次切 tab 都：清空列表 → 打 2~3 个接口 → 等回来才渲染，
      //   于是「点一下先白一下」。现在改成「缓存即时渲染 + 后台刷新 + 请求去重」：
      //   · repCache  : 每个 tab 的战报列表缓存，key=`tab|城市|搜索词` → {reports,radar,recon,intel,at}。
      //                  切 tab 先用缓存立刻渲染，不再清空；同一 tab 2 秒内重复点不重复打接口。
      //   · dynKey/dynLoading   : 军队动态「上次拉取的城市」+「请求是否在途」（连点 tab 不堆请求）。
      //   · countsLoading : 徽标接口「请求是否在途」（这个不设 TTL，保证未读数及时清）。
      //   · repLoadingKey : 战报列表**正在请求的那个缓存 key**（'' = 空闲）。
      //     ★★ 必须按 key 去重，不能用全局布尔（2026-10-11 事故：原来用全局 `_repLoading`，
      //        切分区时「别的分区的请求在途」会把本分区这次请求也挡掉 → 本分区列表永远空着，
      //        用户报「优化的我连战报都看不到了」。）
      //   · repLoading（响应式）: 当前分区是否在加载 → 模板显示「(加载中…)」而不是「(暂无战斗报告)」。
      //   ★ 用 Object.create(null) 而非 {}：纯缓存表，不需要 Vue 递归侦听。
      //   ★★ 2026-10-11 修复「有战报但列表空」：Vue 2 的 initData 对 `_` 开头字段不 proxy 到实例，
      //      `this.repCache` 恒为 undefined → 一访问就抛 TypeError → switchReportTab 的
      //      applyCachedReports 直接崩掉、loadReports 永远不执行。这里全部去掉 `_` 前缀。
      repCache: Object.create(null),
      dynKey: null,
      dynLoading: false,
      countsLoading: false,
      repLoadingKey: '',
      repLoading: false,
      resNames: RES_NAMES,
      // ★ 2026-09-28 头部资源栏「/」右侧展示每小时产量（与资源详情页同口径）
      resProd: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      // ★ 2026-10-05 资源**真正的收敛点**（ezfy_cfg_limit.res_max_*，线上 61 亿）。
      //   原来界面上「当前/仓储上限」的分母是 city.xxx_cap —— 它按设计**不参与产量收敛**，
      //   玩家看到「42亿/55亿」以为还能涨，实际早被全局上限卡住 → 报「离线资源不涨」。
      //   后端已随 /view 下发 res_max，这里据它显示「已满」。
      resMax: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      // ★ 2026-09-28 安抚参数（后端下发，管理端可配：5 万黄金 / 民怨-2 / 民心+1 / 15 分钟冷却）
      placate: { gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 },
      // ★ 安抚冷却的每秒时间基准（由 tickClock 驱动，供 placateCdLeft 计算属性用）
      placateNow: 0,
      placateAt: 0, // 安抚冷却快照(/view 里的 cd_left)的取回时刻
      resShort: RES_SHORT,
      resDes: buildResDes(RES_NAMES),
      profile: { prestige: 0, camp: 1, nickname: '' },
      // ★ 2026-10-10 首页布局：0/1=新布局 2=老布局（/view 的 profile.home_layout 下发，
      //   统帅页单选切换；玩家 10000 由后端 seed 固定为老布局）。
      // ★ 2026-10-10 防「刷新时新布局掠影」：初值从 localStorage 恢复（上次切过的布局），
      //   让首帧就按正确布局渲染，不再先画新布局再等 /view 回来改成老布局。
      homeLayout: this.cachedHomeLayout(),
      // ★ 2026-10-10 防「首次进入(本地无缓存)新布局掠影」：没有缓存时先不渲染首页，
      //   等 /view 下发权威布局（或兜底超时）再渲染；有缓存则照旧首帧即渲染、不闪白。
      homeLayoutReady: this.hasCachedHomeLayout(),
      userBrief: { account: '', level: 0, exp: 0 },
      officerCount: 0,
      activities: [],
      rankName: '列兵',
      rankPost: '士兵',
      city: {},
      cities: [],
      continent: '',
      // ★ 当前城市类型（/view 顶层下发）：city 对象里没有这两个字段，必须单独存
      cityKindRaw: '',
      cityIsSea: false,
      protectedUntil: false,
      // ★ 2026-10-15 免战保护令到期时间戳(ms, 0 = 无免战)：/view 下发，城市状态页展示时效
      //   （只下发布尔值没法显示「剩余多久」，见 peaceText 计算属性）
      protectedUntilMs: 0,
      boostUntil: false,
      buildings: [],
      buildingPool: [],
      // ★ 军事区/资源区各自上限（/view 下发，默认各 36）
      militaryCap: 36,
      resourceCap: 36,
      // ★ 2026-10-22 建造队列：基础上限 + 征募告示临时层数（/buildings 下发）
      buildQueueBase: 5,
      buildQueueBonus: 0,
      buildQueueBonusUntil: 0,
      // ★ 2026-09-26 召集消耗粮食 / 召集人口比例（/view 下发，默认 10 万粮食 / 10%；原来写死固定人口）
      //   ★ 2026-10-22 人口上限只由民居容量 pop_max 决定，不再有额外开关/上限配置。
      conveneFoodCost: 100000,
      convenePopPct: 10,
      // ★ 2026-10-23 召集间隔（秒，/view 下发）：普通默认 3 秒；人口满默认 300 秒。
      //   二战系统配置可调，前端按钮冷却与提示文案都跟随配置。
      conveneCooldownSec: 3,
      conveneCapCooldownSec: 300,
      troopsData: { troops: [], queues: [], wounded: [], cfgs: [], pop: 0, pop_used: 0, wall_level: 0, train_discount: 0 },
      // ★ 占用人口（只有训练队列里没出厂的新兵占）：/view 与 /troops 都会下发，谁后到用谁
      popUsed: 0,
      // ★ 待恢复人口（训练完的人口不瞬间归还，按「空闲人口恢复满时间」线性恢复）：
      //   空闲人口 = 人口 − 占用 − 待恢复 + 额外空闲（见 freePop）。/view 与 /troops 都会下发。
      popRecover: 0,
      // ★ 2026-10-23 额外空闲人口：人口达到民居上限后召集产生，
      //   允许空闲人口暂时超过总人口；超过部分 5 分钟内回落。
      popExtra: 0,
      // 与 popUsed 同一次响应里的「人口」，保证「空闲 = 人口 - 占用」恒成立
      // （分开取 city.pop / troopsData.pop 会出现 1000-996=5 这种对不上的显示）
      cityPop: 0,
      techsData: { techs: [], academy: 0 },
      // ★ 2026-09-28 科技详情页选中的科技（列表点[研究]进入 techpre 时赋值）
      techSel: null,
      wildlands: [],
      occupies: [],
      // ★ 2026-10-07 赎城：待占领方处理的赎回请求 + 赎城金额（管理端可配）
      ransoms: [],
      ransomBusy: false, // ★ 2026-10-07 赎城连点卡控：请求未返回前忽略再次点击
      ransomCost: 500,
      queues: [],
      marching: 0,
      occupying: 0,
      unreadReports: 0,
      // ★ 2026-10-13 未看的「军情警讯(预警) + 战斗报告」数量 —— 顶部导航「军情」后面的感叹号角标
      unreadWarns: 0,
      // ★ 2026-10-23 顶部导航「聊天 / 邮箱」感叹号角标：未读私聊 / 未读邮件。
      //   由轻量接口 GET /games/ezfy/unread 下发（30s 轮询 + 进游戏先拉一次）。
      unreadChat: 0,
      unreadMail: 0,
      reports: [],
      // ★ 军情分区分页：默认每页 5 条
      dynPage: 1, dynSize: 5,
      dynStationPage: 1, dynStationSize: 5,
      // ★ 2026-10-08 出征队列分页（默认每页 5 条，上一页/下一页）
      queuePage: 1, queueSize: 5,
      repPage: 1, repSize: 5,
      notices: [],
      // ★ 2026-10-12 访问记录：我的统帅页被谁看过（信息页「占领野地」下方点开）
      visitRecords: [],
      // ★ 公告分页（「公告也变成分页，下一页上一页那种」）：默认每页 5 条
      noticePage: 1, noticeSize: 5,
      // ★ 首页外露公告（条数由管理端「建筑上限配置」里的「首页公告条数」决定，默认 1）
      homeNotices: [],
      curNotice: null,
      worldChats: [],
      homeChats: [],
      chatPlayers: 0, chatCorpsPlayers: 0,
      chatMsg: '',
      chatChannel: 1,
      chatHasCorps: false,
      chatCorpsName: '',
      chatCanSend: true,
      chatCooldown: 0,
      // ★ 聊天分页（后端按时间降序返回，最新的在第 1 页最上面）
      chatPage: 1,
      chatTotal: 0,
      chatSize: 15,
      pmContent: '',
      // ★ 私聊：当前会话对象 + 与该对象的聊天记录 + 会话列表
      pmPeer: null,
      pmChat: [],
      // ★ 2026-10-13 私聊发送用「好友游戏ID」而非昵称（游戏昵称/家园昵称不一致会解析失败→收不到）；
      //   聊天记录分页（新到旧，前端反转成旧到新展示）
      pmPeerUid: 0,
      pmChatPage: 1, pmChatTotal: 0, pmChatSize: 20,
      // ★ 2026-10-13 好友页 tab 内子视图：convs=会话列表（微信式） friends=好友列表
      friendView: 'convs',
      pmConvs: [],
      pmConvPage: 1, pmConvTotal: 0, pmConvSize: 10,
      friends: [],
      friendKeyword: '',
      friendSearchList: [],
      friendSearchDone: false,
      friendApplies: { inbox: [], outbox: [] },
      // ★ 2026-10-13 好友页四 tab：list好友 / apply申请 / sent已发送 / search查找
      friendTab: 'list',
      // ★ 2026-10-13 邮箱页：四 tab + 未读计数 + 分页 + 写邮件
      mailTab: 'inbox',
      mailInbox: [], mailSystem: [], mailSent: [],
      mailUnread: { inbox: 0, system: 0, total: 0 },
      mailTotal: 0, mailPage: 1, mailSize: 10,
      mailOpen: 0,
      // ★ 2026-10-18 领取防抖：正在领取的邮件 id（0=空闲），按钮变「领取中…」且不可再点
      mailClaiming: 0,
      mailCompose: { to: '', subject: '', content: '' },
      taskGroups: [],
      // ★ 任务分类 tab(0=默认第一个分类: 新手/日常/每周); -1=为爱发电卡。
      //   2026-09-27 fix: 刷新后记住上次所在 tab（localStorage 恢复）
      taskTab: this.restoreTaskTab(),
      // ★ 2026-09-27 为爱发电卡（管理端未发放时为空数组，对应 tab 不显示；多卡可叠加）
      loveCards: [],
      // ★ 计谋（配置由后端下发，发动消耗「信号弹」）
      // ★ 2026-10-10 道具 cfg_id 已全体迁到 1001+（与装备号段彻底分开）→ 别写死数字，
      //   一律用后端 /schemes 下发的 bullet_item_id。
      schemeData: { schemes: [], bullet_name: '信号弹', bullet_have: 0, bullet_item_id: 0 },
      // ★ 2026-09-30 行军计谋页：当前选中的部队（军队动态/出征队列的 [计谋] 带入）
      schemeOrder: null,
      schemeX: '', schemeY: '',
      welfare: { rewards: [], gifts: {} },
      // ★ 2026-09-28 福利页 tab: 0每日签到 / 1礼包 / 2宝物签到
      welfareTab: 0,
      rankData: { prestige: [], troops: [], corps: [], ranks: [] },
      // ★ 排行页 tab: ranks军衔晋升表 / prestige军衔声望榜 / troops兵力榜 / corps军团榜
      rankTab: 'pvp', // ★ 2026-10-14 风云榜置首，进排行页默认落点改风云榜
      // ★ 2026-10-14 风云榜（每周玩家对战战绩周榜）：pvpRank.week 有值 = 已加载当前周
      pvpRank: { week: '', weeks: [], ranks: [], mine: {} },
      pvpLoading: false,
      // ★ 2026-10-10 晋升防抖：防止快速连点一次晋升多级（promote 期间/短暂冷却内忽略再次点击）
      promoting: false,
      // ★ 军衔晋升表中「宝物」点击展开的行下标（-1 = 收起）
      showTreasureRow: -1,
      // ★ 战力榜中「战力」点击展开明细的行 rank（-1 = 收起）
      showPowerRow: -1,
      // ★ 2026-10-14 风云榜中「积分」点击展开明细的行 rank（-1 = 收起），与战力榜同款交互
      showPvpRow: -1,
      // ★ 2026-10-16 风云榜「我的战绩」场次明细（分页倒序，独立表——战报会周期性清理，明细留存）
      pvpDetail: { show: false, loading: false, list: [], total: 0, page: 1, size: 20, week: '' },
      orders: [],
      buildZone: 'm',
      buildSel: null,
      // ★ 2026-10-16 军事区/资源区「点建筑名 → 建筑详情页」要看的建筑 id（0 = 不在详情页）。
      //   只存 id，展示的数据由 computed buildInfo 现取 —— 30s 轮询/升级完工后效果文字自动跟着变。
      buildInfoId: 0,
      curReport: null,
      reportTab: 1,
      // ★ 2026-10-09 「没军团不显示军团战报 tab」：由 /reports/counts 的 has_corps 下发。
      //   null = 还没拿到（tab 先按 myCorps 判断，避免团员刷新时 tab 闪一下不见）。
      reportHasCorps: null,
      reportWord: '',
      reportCounts: {},
      // ★ 自己城市的雷达站等级（决定「来袭/被侦查」预警能不能收到）
      reportRadar: 0,
      // ★ 2026-09-25：情报等级 = 雷达站 + 侦察技巧（后端算好下发），前端据此提示还差多少
      reportRecon: 0, reportIntel: 0,
      dynamics: [],
      // ★ 战场指挥室（军情 → 军队动态 → [指挥]）：每回合 30 秒，前 25 秒可下指令
      battleData: {
        order_id: 0, target_name: '', target_x: 0, target_y: 0, target_type: 0,
        round: 0, max_round: 40, status: 1, win: 0, atk_cmd: '', atk_cmds: {},
        phase: 'cmd', round_left_ms: 0, round_ms: 30000, cmd_window_ms: 25000,
        attackers: [], defenders: [], atk_total: 0, def_total: 0,
        is_atk: true, pvp: false, can_auto: true,
        atk_locked: 0, def_locked: 0, my_locked: 0, can_lock: false,
        head: [], actions: [], done: false
      },
      battleOrderId: 0,     // 正在指挥的出征订单 id
      battleLeftMs: 0,      // 本地倒计时（毫秒，每秒自减；归零时拉服务端推进回合）
      battleTimer: null,
      // ★★ 2026-10-12 逐兵种指令/目标的**本地待确认值**（select 的 :value 优先取它）：
      //   一选就落到这里 → 下拉立刻显示新值、且不会被服务端轮询回包弹回；
      //   服务端回包与本地一致时自动清除该项，之后完全以服务端为准（见 syncBattleLocal）。
      battleCmds: {},       // troop_id -> 指令（advance/hold/retreat/''）
      battleTargets: {},    // troop_id -> 优先攻击目标（0=最近目标）
      battleCmdInFlight: false, // 指令/目标 API 往返期间暂停轮询，防止整体刷 state 打断选择
      // ★ 2026-10-06 指挥室「展开/收起回合详情」：默认展开（默认只看回合标题时信息太少）
      battleDetailOn: true,
      curOrder: null,
      showDetail: true,
      corpsList: [],
      myCorps: null,
      corpsMembers: [],
      corpsChats: [],
      corpsName: '',
      corpsMsg: '',
      kickUserId: 0,
      // ★ 2026-09-25 军团页拆成「军团信息 / 军团外交 / 军团宣战 / 军团商城」四个子栏，
      //   纯前端 tab 切换（照抄 rank/acade 页的 .acade-tab 写法），数据按需懒加载。
      // ★ 2026-09-30 再加「军团列表 / 军团聊天」独立 tab → 六个。
      corpsTab: 'info',        // info军团信息 / list军团列表 / chat军团聊天 / diplomacy军团外交 / war军团宣战 / mall军团商城
      // ★ 2026-09-30 入团审核：军团的待审申请 / 审核开关 / 我是否已提交申请
      corpsApplies: [],
      myCorpsNeedReview: 0,
      myApplyStatus: 0,        // 0无申请 1待审 2通过 3拒绝
      corpsPoints: 0,          // 军团总积分（/corps/members 或 /corps/relations 的 my_corps.points）
      corpsRelations: null,    // /corps/relations 全量数据（null=还没拉过）
      corpsWars: null,         // /corps/war 全量数据（null=还没拉过）
      // 军团商城：loaded 标记避免每次切 tab 重复拉（兑换/宣战等操作后才会重新拉）
      corpsMall: { loaded: false, my_points: 0, corps_points: 0, items: [], my_bought: {} },
      mallItems: [],
      // ★ 第九轮：商城分类 + 分页 + 钻石余额
      mallCatsList: [], mallCat: '', mallPage: 1, mallPageSize: 10, mallDiamond: 0,
      // ★ 商城分栏：item=道具（原有） equipment=装备套装（用黄金/钻石买）
      mallTab: 'item',
      // ★ 单次购买数量上限（管理端「建筑上限配置」页维护，2026-10-22 默认 999）
      mallBuyMax: 999,
      // ★ 2026-10-11 背包单件道具上限（管理端「系统配置」页维护，默认 5000）
      mallBagMax: 5000,
      bagItems: [],
      bagTreasures: [],
      // ★ 2026-10-11 背包分栏：item=道具 / equip=军官装备
      bagTab: 'item',
      // ★ 2026-10-11 背包「军官装备」tab：非宝物装备（军官装备/武器/防具/饰品），武器防具可卖黄金
      bagEquips: [],
      sellEquip: null,      // 正在出售的装备 {cfg_id,name,type,tier,count,unworn,sell_price}
      sellEquipCount: 1,
      // ★ 2026-10-11 宝物库存上限（管理端「系统配置」可配，默认 5000）：宝物页展示「宝物(N/上限)」
      treasureMax: 5000,
      // ★ 2026-10-08 宝物出售给系统（统一 20万黄金/件，收10%手续费）
      sellTreasure: null,          // 正在出售的宝物对象 {cfg_id,name,count}
      sellTreasureCount: 1,
      // ★ 背包只展示道具（2026-10-06 宝物已独立成「宝物」导航页，去掉宝物子tab）
      bagWord: '', bagPage: 1, bagPageSize: 10, bagCat: '',
      // ★ 2026-10-11 背包「军官装备」tab 也分页（用户：「背包也要做分页 不然展示太多」）
      bagEquipPage: 1, bagEquipPageSize: 10,
      equipWord: '', equipPage: 1, equipPageSize: 10,        // 我的装备
      equipTab: 'my',                                      // 装备页子tab: my我的装备 / set我的套装 / all装备图鉴
      hqTab: 0,                                          // ★ 司令部子tab: 0兵种配置 / 1出征队列 / 2伤兵营 / 3逃兵营 / 4预设编队 (localStorage 记忆)
      woundNums: {},                                     // ★ 2026-10-06 伤兵营/逃兵营行内「恢复数量」输入（key=伤兵记录id，0/空=全部）
      healAllDiamond: 100,                // ★ 2026-10-14 伤兵营钻石一键恢复全部伤兵单价（系统配置下发，默认100）
      // ★ 2026-09-28 预设编队（司令部保存的出征模板：军官+集结令+兵力，不含目标/随军资源/宿营）
      presets: [],                                       // 预设列表 [{id,name,officer,gather,troops,troop_total}]
      presetSel: 0,                                      // 出征页「预设编队」下拉选中 id (0=不使用)
      presetName: '',                                    // 新增预设的名称输入
      presetAdding: false,                               // 预设编队 tab 是否正在「新增预设」表单模式
      presetMax: 10,                                     // 单账号预设数量上限（与后端 ezfyPresetMax 一致）
      // ★ 2026-09-25：套装一览是否显示「全部套装」（含还没拥有的）—— 方便玩家横向对比
      setShowAll: false,
      // ★ 2026-09-25 用户反馈「套装的加成玩家看不到、不知道买完套装给军官用哪个」：
      //   全部套装配置（含加成/部位/我拥有几件）单独拉一次并缓存，装备页·商城页·军官页共用。
      //   detailRowId = 装备表里正展开详情卡的那一**行**（存装备 id，不是 set_id！
      //   存 set_id 的话同一套的每一行都会各自展开一张一样的卡 —— 实测 10 件套会蹦出 10 张）。
      //   detailMode = 展开的是哪一份内容：'item' 这件自己的加成 / 'set' 套装加成
      //   （「点名称看该装备的加成，点套装看套装的加成」）。
      allSets: [],
      detailRowId: 0,
      detailMode: '',
      equipAllWord: '', equipAllPage: 1, equipAllPageSize: 10, // 装备图鉴
      officerBagWord: '', officerBagPage: 1, officerBagPageSize: 10, // 军官详情里的背包装备
      // 军官详情页签: attr属性 / scheme计谋 / equip装备 / general名将背景
      // ★ 2026-10-25 「技能」页签已去掉：学技能/遗忘挪到「属性」页的技能1~3 三行
      officerDetailTab: 'attr',
      officerRenameInput: '',   // ★ 2026-10-24 怀旧军官详情：名称输入框（点 [改名] 用输入框里的名字）
      officerTreasureOpen: false, // ★ 2026-09-28 赏赐宝物：展开的未穿戴宝物列表
      // ★ 2026-10-10 性能（玩家反馈「军官详情非常卡」）：背包装备不再随军官详情下发，
      //   切到装备背包/装备/赏赐宝物时才按需拉 /officers/:id/backpack，切换军官时清空。
      officerBag: [],       // 懒加载的背包装备（含已穿戴/被俘过滤后的全局背包）
      officerBagSets: [],   // 懒加载的「可一键穿戴套装」列表
      officerBagFor: 0,     // 当前 officerBag/officerBagSets 属于哪个军官 id（防切换串数据）
      officerBagLoaded: 0,  // 0 未拉 / 1 已对当前军官拉好
      // ★ 2026-09-25：equipDetail / equipDetailBack 已随「装备详情页」一起删除
      bagOfficers: [],
      bagSkills: [],
      useItem: null,
      useCount: 1,
      useOfficerId: 0,
      inlineTip: null, // 建筑区操作的内联提示 { bid, text, type }
      buyItem: null,
      buyCount: 1,
      buyPayWith: 'gold', // ★ 双渠道道具的支付方式选择（gold / diamond）
      exchangeOrders: [],
      exchangeMine: [],
      exchangeGold: 0,
      // ★ 交易行双分页：卖家挂单(exchangePage/exchangeSize/exchangeTotal)、我的挂单(exchangeMPage/...)
      exchangePage: 1, exchangeSize: 10, exchangeTotal: 0,
      exchangeMPage: 1, exchangeMSize: 10, exchangeMTotal: 0,
      // ★ 2026-10-04 交易行挂单上限（后端 ExchangeList 下发 sell_max = 交易所等级×2）
      exchangeSellMax: 0,
      exFilter: 0, // ★ 2026-09-24 卖家挂单资源类别检索(0=全部 1粮食 2钢铁 3石油 4稀矿)
      acadeTab: 'officer',
      // ★ 2026-10-04 军官模块 6 个 tab 的数据缓存（进页预取 + 切换秒开）。
      //   ⚠️ 注意：_ 开头的属性不能放 data()（Vue2 不会代理到 this，读了是 undefined），
      //   改用 withAcadeCache 里的懒初始化（与 _onceMap 同一套模式）。
      officerData: { officers: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, captive_capacity: 0, captive_used: 0, gold: 0 },
      recruitData: { candidates: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0, refresh_left: 0, refresh_limit: 5 },
      skillData: { skills: [], officers: [], gold: 0 },
      equipData: { bag: [], all: [], sets: [] },
      generalData: { generals: [] },
      generalPage: 1, generalPageSize: 10, // ★ 2026-10-15 名将图鉴分页（默认每页 10）
      officerDetail: { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 },
      // ★ 2026-10-01 修复「点击军官有时候空白」：加载失败/武将不在当前城时，
      //   不再静默留空，页面展示该错误并引导返回军官列表
      officerDetailError: '',
      // ★ 装备商城（套装用黄金/钻石购买）
      equipShop: { slots: [], items: [], gold: 0, diamond: 0 },
      // ★ 商城散件：部位筛选 + 检索 + 分页（「按部位分组表格 + 检索」）
      shopSlot: '', shopWord: '', shopPage: 1, shopSize: 20,
      // ★ 宝箱奖池：点名字才展开（「别直接展示」）+ 检索 + 分页
      chestPoolId: 0, chestPoolWord: '', chestPoolPage: 1, chestPoolSize: 20,
      bagDescId: 0,   // 背包里「点 [说明] 展开」的那件道具（0 = 都没展开）
      equipShopBuy: null,     // 正在填写购买数量的商品
      equipShopCount: 1,
      equipShopPay: 'gold',   // gold | diamond
      // ★ 宝箱（用钻石/黄金买，开箱按权重出套装件）
      chestData: { chests: [], gold: 0, diamond: 0 },
      chestOpen: null, chestCount: 1, chestPay: 'diamond', chestResult: [], chestLastCost: '',
      // ★ 2026-10-14 我的待开宝箱（商城买的、管理端发放的，自己开箱）
      chestBag: { list: [], total: 0 },
      // ★ 2026-10-14 商城宝箱「购买」独立页：目标宝箱 + 数量（买进背包不抽奖）
      chestBuyTarget: null, buyChestCount: 1,
      // ★ 2026-10-14 背包宝箱自定义开箱数量
      bagChestCount: 1,
      // ★ 2026-10-14 背包宝箱：开箱中锁（防连点）+ [说明]展开的宝箱 id
      chestOpening: false, bagChestDescId: 0,
      // ★ 2026-10-09 宝箱内容页（chestview）「点奖品名查看详情」：当前展开的奖品下标（-1 = 均收起）
      chestViewDetailIdx: -1,
      // ★ 2026-10-13 商城宝箱行内「可开出」预览：当前展开的宝箱 id（0 = 均收起）
      chestDescId: 0,
      sellType: '1',
      sellCount: 0,
      sellPrice: 0,
      // ★ 2026-09-30 向系统出售资源：每100单位比例 / 手续费 / 丢量提示
      sellSysType: '1',
      sellSysCount: 0,
      sysSellRatio: { 1: 10, 2: 10, 3: 20, 4: 25 },
      sysSellFee: 30,
      // ★ 2026-10-22 每日每种资源回收上限（0 = 不限）+ 今日已售数量
      sysSellDailyLimit: 1000000,
      sysSellUsed: { 1: 0, 2: 0, 3: 0, 4: 0 },
      sysSellLostWarn: '',
      goldMax: 0,
      trainSel: null,
      trainCount: 10,
      trainSplit: true,
      trainBusy: false, // ★ 2026-10-25 训练页 [开始训练] 连点卡控：请求未返回 / 未满 1 秒时忽略再次点击
      trainMode: 'troop', // troop=训练(createTroop) / defence=建造(createDefence)
      troopViewId: 0,     // 兵种详情页当前兵种 id
      troopViewBack: 'troops', // 兵种详情页 [返回] 回到哪一页
      // 页面内消息 + 内联确认（替代 alert/confirm/prompt 弹窗）
      msgs: [], askBox: { show: false, text: '', input: false, placeholder: '', value: '' },
      // ★ 确认条实测高度（渲染后量一次）—— askStyle 靠它把确认条夹在视口内，
      //   估算值只做首次兜底，避免「按钮在最底部时确认条被切掉一半」。
      askH: 0,
      // 统帅页自助（游戏ID/家园号码、改昵称、改阵营）
      selfInfo: {}, renameEditing: false, renameInput: '',
      playerInfo: null,        // 他人统帅信息(复刻 infoOther)
      playerInfoBack: 'chat',  // 他人统帅页 [返回] 回到哪一页
      corpsMailContent: '',    // 军团邮件群发内容
      myCorpsTitle: '',        // ★ 我在军团的职位(副团长/参谋长)，副团长可发军团邮件
      taxInput: 20,
      renameInput: '',
      newCityX: '',
      newCityY: '',
      targetCfg: {},
      resType: 'gold',
      resDetail: null,
      mapCells: [],
      // ★ 2026-10-13 地图请求自增序号：连点方向键会堆一串在途请求，响应**谁后到谁覆盖** →
      //   画面来回跳、还白渲染好几遍。只应用「最新一次请求」的响应（见 loadMap/jumpTo）。
      mapSeq: 0,
      // ★ 2026-10-25 详情请求序号：连点两个格子时，先发的响应可能后到，
      //   会把**上一块地**的详情（含 is_starred）安到当前格子上 →「没收藏却显示已收藏」。
      //   只应用最新一次 openCell 的响应。
      detailSeq: 0,
      mapCx: 0,
      mapCy: 0,
      mapR: 2, // 视野半径 → 5×5 表格(复刻 map/index.html)
      selCell: null,
      selDetail: null,
      eliteCell: null,
      warText: '',
      warStatus: 0, // 0未宣战 / 1宣战待生效 / 2交战中
      // ★ 管理端「宣战功能」开关（/war/status 下发 war_require）：
      //   false = 不需要宣战，掠夺/征服直接可点。默认 true（开关默认开）。
      warRequire: true,
      // ★ 2026-09-25 军团交战期也能掠夺/征服（无需个人宣战）。
      //   atWar = /war/status 下发的 at_war（个人宣战已生效 或 管理端关掉宣战开关 或 军团交战期）；
      //   corpsWar = /war/status 下发的 corps_war（{active, corps_name, text}）。
      atWar: false,
      corpsWar: null,
      orderType: 2,
      orderTroops: {},
      onDutyOfficers: [],
      // 本城军官全量列表（含出征中/俘虏）：出征页用它解释「为什么没有可带队军官」
      cityOfficers: [],
      // ★ 2026-10-25 城市状态页：本城市长/城守的军官名（空 = 未任命），见 loadOfficerPosts
      cityMayor: '',
      cityGuard: '',
      orderOfficer: '0',
      ware: { level: 0, total: 0, next_total: 0, res: [], ratio_sum: 0 },
      wareRatio: { food: 25, steel: 25, oil: 25, rare: 25 },
      liaison: { level: 0, can_join: false, can_create: false, create_cost: 50000,
        member_per_level: 10, my_corps: null, member_count: 0, member_cap: 0,
        garrison_cap: 0, garrison_used: 0, garrisons: [] },
      trFood: 0,
      trSteel: 0,
      trOil: 0,
      trRare: 0,
      trGold: 0,
      waitH: 0,
      // ★ 2026-10-07 出征页「自动战斗」选择：1=是（默认；打野地 / AI 寇城抵达即自动打完，无需指挥）
      //   0=否（抵达后进指挥室部署）。打玩家城市时该项只读、恒为 0（见 orderIsPlayerTarget / orderBody）。
      orderAutoBattle: 1,
      waitM: 0,
      orderCalc: null,
      // ★ 本次出征使用几个集结令（数量由管理端配置决定，不写死）
      orderGather: 0,
      // ★ 集结令配置：随 /view 一起下发（一进页面就是准确值）。
      //   原来只有 /order/preview 才返回 gather_max，前端在没点[计算]前兜底写死 50，
      //   结果管理端配了 999 也只能填 50 —— 用户反馈的 bug。
      gatherCfg: { max: 0, per: 0, have: 0 },
      jumpX: '',
      jumpY: '',
      mapStars: [],
      showStars: false,
      // 城市迁移
      // ★ 第十二轮：迁城区域改成「按大洲」（与地图所属洲一致），且迁城消耗道具
      moveInfo: { areas: [], items: [], gold_cost: 200000, gold: 0, city: {} },
      moveArea: 1,          // 兼容旧字段（= moveContinent）
      moveContinent: 1,     // 迁城计划迁入的洲（默认欧洲）
      moveContinentSea: 1,  // 沿海迁城计划迁入的洲
      moveItems: [],        // 三种迁城道具的持有量/定价
      moveX: '',
      moveY: '',
      moveX2: '',
      moveY2: '',
      // 调整生产(开工率)
      rateFood: 100,
      rateSteel: 100,
      rateOil: 100,
      rateRare: 100,
      orderNames: ['', '侦查', '掠夺', '征服', '采集', '运输', '增援', '驻守采集', '派遣'],
      timer: null
    }
  },
  // ★ 2026-09-27 建筑操作的内联提示自动消失：设置后过几秒清掉，
  //   避免「已取消升级/已开始升级」这类提示一直挂在建筑行上不消失。
  //   [关闭] 仍然可用（直接置 null）。
  watch: {
    inlineTip (n) {
      if (this._tipTimer) { clearTimeout(this._tipTimer); this._tipTimer = null }
      if (n) this._tipTimer = setTimeout(() => { this.inlineTip = null }, 5000)
    },
    // ★ 2026-09-28 随军资源上限跟随所选兵种：兵力变化时防抖重算负重(orderCalc.carry)，
    //   让 resQtyMax 的上限(=负重×装载技术加成)始终与当前部队一致。
    //   没选任何部队(归零)时把随军资源一并清空，配合 orderResDisabled 整块禁用。
    orderTroopTotal (n) {
      if (n <= 0) {
        if (this._resCalcT) { clearTimeout(this._resCalcT); this._resCalcT = null }
        this.trFood = 0; this.trSteel = 0; this.trOil = 0; this.trRare = 0; this.trGold = 0
        return
      }
      if (this._resCalcT) clearTimeout(this._resCalcT)
      this._resCalcT = setTimeout(() => this.doCalc(), 300)
    }
  },
  computed: {
    // ★ 2026-10-07 出征目标是不是「玩家城市」：
    //   地图格 area_type=3 表示玩家城 → 出征页「自动战斗」强制「否」且只读
    //   （守方是真人，必须留指挥机会）。野地 / AI 寇城为 1/2，不受限。
    // ★★ 2026-10-19 扩展到**有玩家驻军的附属野地**（用户：「攻击有玩家驻守的附属野地
    //   不能自动战斗，还是进入指挥」）：野地详情下发 has_defender（该坐标上有
    //   增援驻军6/驻守采集7 的活跃订单）→ 守方同样是真人部队，必须手动指挥。
    //   后端 processArrive 也有同一判定（不信任前端提交的 auto_battle）。
    orderIsPlayerTarget () {
      if (this.selCell && this.selCell.area_type === 3) return true
      return !!(this.selDetail && this.selDetail.has_defender)
    },
    // ★ 2026-10-07 「自动战斗」的**生效值**：打玩家城市恒为 0（前端只读、后端也强制），
    //   其余按玩家在出征页所选。模板高亮与提交都用它，保证「显示 = 实际提交」。
    orderAutoBattleVal () {
      return this.orderIsPlayerTarget ? 0 : (this.orderAutoBattle ? 1 : 0)
    },
    // ★ 2026-09-28 赏赐宝物：背包里未穿戴的**采集宝物**（可在军官详情操作区展开选择）
    //   ★ 只认采集宝物（后端 bag 条目带 treasure 标记，名字取自 9 种野地珠宝）——
    //     步枪/钢盔/合金装甲这类普通装备不能换忠诚；宝物签到抽的也是同一池，所以签到宝物可用。
    // ★ 2026-10-05 相同宝物按名称合并成一行带数量（原来逐件列出，背包几十件时列表太长）；
    //   保留一个代表 id 供 [赏赐] 用（同组宝物任一件都可赏赐）。
    officerTreasures () {
      // ★ 2026-10-10 背包装备改懒加载：从 officerBag 取（点「赏赐宝物」前已按需拉取）
      const rows = ((this.officerBag) || [])
        .filter(e => e.treasure && !e.worn)
      const byName = {}
      for (const e of rows) {
        const k = e.name || ('#' + (e.cfg_id || e.id))
        if (!byName[k]) byName[k] = {
          id: e.id, cfg_id: e.cfg_id, name: e.name,
          tier: e.tier, tier_name: e.tier_name, count: 0
        }
        byName[k].count++
      }
      return Object.values(byName)
    },
    // ★ 2026-10-25 军官属性页的 3 个技能槽（固定 3 行：学了显示名字/等级，没学是空槽）
    //   学技能已改为「消耗 1 本技能书随机学一个」，所以三行的 [学习] 完全等价。
    //   上限按 max(3, 已学数)：老数据万一超过 3 个也不会被藏起来（否则那几条没法遗忘）。
    officerSkillSlots () {
      const list = (this.officerDetail && this.officerDetail.skills) || []
      const n = Math.max(3, list.length)
      const out = []
      for (let i = 0; i < n; i++) out.push(list[i] || null)
      return out
    },
    // ★ 2026-10-13 开箱结果明细：同名同品质合并成一行 ×N（几十个箱子结果不刷屏），高品质排前
    chestResultGroups () {
      const rank = { '传说': 0, '传奇': 0, '特殊': 0, '史诗': 1, '高级': 1, '稀有': 2, '中级': 2, '优良': 3, '普通': 3 }
      const byK = {}
      for (const r of (this.chestResult || [])) {
        const k = (r.name || '?') + '|' + (r.quality || '普通')
        if (!byK[k]) byK[k] = { name: r.name || '?', quality: r.quality || '普通', n: 0 }
        byK[k].n++
      }
      return Object.values(byK).sort((a, b) =>
        ((rank[a.quality] === undefined ? 9 : rank[a.quality]) - (rank[b.quality] === undefined ? 9 : rank[b.quality])) ||
        (a.name < b.name ? -1 : 1))
    },
    // ★ 任务分类 tab: 当前展示的任务组(新手/日常/每周; taskTab=0 或无效时回落到第一组)
    taskGroupCur () {
      const gs = this.taskGroups || []
      if (!gs.length) return { id: 0, name: '', reset_type: 0, tasks: [] }
      return gs.find(g => g.id === this.taskTab) || gs[0]
    },
    // ★ 2026-09-27 为爱发电卡合计可领天数（多卡叠加），>0 才有「领取」按钮
    loveTotalClaimable () {
      const cs = this.loveCards || []
      let n = 0
      for (const c of cs) n += (c.claimable || 0)
      return n
    },
    // ★ 二级导航（资源/军官/军队/科技/城防/统帅）：只在对应页面显示，位置固定在页面顶部
    //   军队的几个子页（兵种/兵种详情/训练/工厂）也算「军队」，一并显示，保持导航不中断
    isArmyPage () {
      return ['troops', 'troop', 'troopview', 'trainpre', 'factory'].indexOf(this.cur) >= 0
    },
    showSubnav () {
      return ['buildm', 'builds', 'acade', 'officerdetail', 'techs', 'techpre', 'defence', 'info'].indexOf(this.cur) >= 0 || this.isArmyPage
    },
    // ★ 2026-10-10 是否「首页-新布局」：新布局下底部 16 项导航与首页主网格重复 → 整块隐藏；
    //   老布局（homeLayout===2）与其它页面保留底部导航。
    isNewHome () {
      return this.cur === 'home' && this.homeLayout !== 2
    },
    // ★ 玩家当前军衔等级 id（用于首页/统帅信息展示对应军衔星级图标）
    myRankId () { return this.rankIdByName(this.rankName) },
    // ★★ 2026-10-20 统帅页「战争状态」：按宣战待生效 / 交战中分组
    //   状态由后端按 effect_time 现算（库里 status 可能还没推进）。
    pendingWars () { return (this.selfInfo.wars || []).filter(w => w.status === 1) },
    activeWars () { return (this.selfInfo.wars || []).filter(w => w.status === 2) },
    // 改名提示：首次免费 / 之后消耗改名卡
    renameHint () {
      const d = this.selfInfo || {}
      if (!d.game_uid && !d.nickname) return '首次改名免费'
      if (d.rename_free) return '首次改名免费'
      return '首次免费已用掉，再次改名需消耗「改名卡」×1（当前持有 ' + (d.rename_card_count || 0) + ' 张）'
    },
    // 阵营提示：首次免费 / 之后消耗阵营转换道具
    campHint () {
      const d = this.selfInfo || {}
      if (!d.game_uid && !d.nickname) return '首次转换阵营免费'
      if (d.camp_free) return '首次转换阵营免费'
      return '首次免费已用掉，再次转换需消耗「阵营转换道具」×1（当前持有 ' + (d.camp_item_count || 0) + ' 个）'
    },
    nick () {
      // 优先二战账号，其次家园账号（二战独立后家园身份可能为空）
      return (this.$store.state.ezfyUser && this.$store.state.ezfyUser.nickname) ||
        (this.$store.state.user ? this.$store.state.user.nickname : '')
    },
    // 私聊里区分「我」和「对方」
    myUserId () {
      return (this.$store.state.ezfyUser && this.$store.state.ezfyUser.id) ||
        (this.$store.state.user ? this.$store.state.user.id : 0)
    },
    // ★ 2026-10-12 二战独立登录：本地有没有「二战专属登录态」（ezfy_token）。
    //   只是没登录 → 根组件渲染二战自己的登录界面，不依赖家园 token、不弹回家园。
    isLogin () {
      return !!this.$store.getters.isEzfyLogin
    },
    // ★★ 2026-10-18 当前家园账号（家园号）。有值时二战登录页的账号框锁定成它，
    //   杜绝「A 登录家园、再用 B 登二战」的账号串台。
    //   ★ 从 token 里解（store getter），不读 user 对象 —— 跨端口时 user 在别的源里读不到。
    homeUid () {
      return String(this.$store.getters.homeUid || '')
    },
    // ★★ 2026-10-18 本地二战登录态属于哪个家园号（登录/SSO 时写入 ezfyUser.id）
    ezfyUid () {
      const u = this.$store.state.ezfyUser
      return (u && u.id) ? String(u.id) : ''
    },
    freePop () {
      // ★ 占用人口由后端统一算：只有「训练中、还没出厂」的新兵占人口。
      //   已训练完成的部队（城内驻军 / 出征在外）都不占人口位置（用户 2026-09-21 的规则）。
      //   原来只减 troopsData.pop_used（且只有进过「军队」页才有值）→ 首页空闲人口会显示成满人口。
      // ★ 2026-10-13 再扣掉「待恢复人口 pop_recover」：训练完的士兵占用的人口不再瞬间归还，
      //   而是按「空闲人口恢复满时间」一点点恢复（见后端 ezfySettlePopRecover / ezfyPopFree）。
      //   不扣的话，训练完的一瞬间空闲人口会虚增 —— 用户反馈的「训练完立即恢复」bug。
      // ★ 2026-10-23 再加上「额外空闲人口 pop_extra」：人口满后召集产生，允许空闲暂时超过总人口。
      const used = (this.popUsed || 0) + (this.popRecover || 0)
      const pop = this.cityPop || this.city.pop || 0
      const v = pop - used + (this.popExtra || 0)
      return v > 0 ? v : 0
    },
    // ★ 2026-10-22 人口上限只由民居容量（pop_max）决定（0 = 没有民居/未知）。
    popCap () {
      return (this.city && this.city.pop_max) || 0
    },
    // ★ 2026-10-22 召集人口 = 当前民居上限 × 后台配置百分比（默认 10%）。
    //   与后端 ezfyConvenePopGainOf 同口径：向下取整，至少 1。
    convenePopGain () {
      const pm = (this.city && this.city.pop_max) || 0
      const pct = this.convenePopPct || 0
      const gain = Math.floor(pm * pct / 100)
      return gain > 0 ? gain : 1
    },
    // 当前人口是否已达到/超过民居上限。
    // 人口已满时召集仍可用：总人口不增加，本次召集量全部转为空闲人口
    // （间隔由二战系统配置控制，默认 300 秒）。
    popAtHouseCap () {
      const pm = (this.city && this.city.pop_max) || 0
      const pop = (this.city && this.city.pop) || this.cityPop || 0
      return pm > 0 && pop >= pm
    },
    // 召集冷却：普通 / 人口达到或超过民居上限分别读二战系统配置（默认 3 秒 / 300 秒，
    // 与后端 ezfyConveneCooldownMs 同口径）。管理端改完 /view 立即带回新值。
    conveneCooldownMs () {
      const raw = this.popAtHouseCap ? this.conveneCapCooldownSec : this.conveneCooldownSec
      const sec = Math.max(1, Math.min(1000000000, Number(raw) || (this.popAtHouseCap ? 300 : 3)))
      return sec * 1000
    },
    // ★ 2026-10-15 免战保护令文案（城市状态页）：「[免战保护中] 剩余 2小时30分」。
    //   剩余时间每秒随 gatherNow 重算（全站统一做法：后端下发绝对到点时间戳，前端本地倒计时）；
    //   到点 / 后端没下发到期时间 → 只显示状态（不显示剩余）。
    peaceText () {
      if (!this.protectedUntil) return ''
      const until = this.protectedUntilMs
      if (!until) return '[免战保护中]'
      const ms = until - (this.gatherNow || Date.now())
      if (ms <= 0) return '[免战保护中]'
      const s = Math.floor(ms / 1000)
      const h = Math.floor(s / 3600)
      const m = Math.floor((s % 3600) / 60)
      const left = h > 0 ? (h + '小时' + m + '分')
        : (m > 0 ? (m + '分' + (s % 60) + '秒') : (s + '秒'))
      return '[免战保护中] 剩余 ' + left
    },
    queueNames () {
      return this.queues
    },
    // ★ 2026-10-22 人口上限只由民居容量决定：
    //   人口未满 → 召集加人口（最多加到民居上限）；
    // ★ 2026-10-23 人口已满 → 仍可召集，总人口不增加，空闲人口增加后回落；
    //   因此按钮只受「民居容量为 0」禁用，冷却由 doConvene + 后端卡控。
    conveneBlocked () {
      const pm = (this.city && this.city.pop_max) || 0
      return pm <= 0
    },
    // 当前建筑分区: 'm' 军事区 / 's' 资源区
    // 复刻原版 BuildingController: 军事区 = type 2/3/4, 资源区 = type 1
    zone () {
      if (this.cur === 'builds') return 's'
      if (this.cur === 'buildpre') return this.buildZone
      // ★ 2026-10-16 建筑详情页按**那栋建筑自己的分区**算（它的名字就是从那个区点进来的）
      if (this.cur === 'bdetail') return this.buildInfoZone
      return 'm'
    },
    zoneBuildings () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      const built = this.buildings.filter(b => inZone(b.type))
      const pool = (this.buildingPool || []).filter(p => inZone(p.type))
      return built.concat(pool)
    },
    // 已建建筑(军事区/资源区主页面只列这些)
    zoneBuilt () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      const built = this.buildings.filter(b => inZone(b.type))
      // ★ 军事区：民居(building_id=2)固定排到列表最下面（其余顺序不变）
      if (isM) {
        const res = built.filter(b => b.building_id === 2)
        if (res.length) return built.filter(b => b.building_id !== 2).concat(res)
      }
      return built
    },
    // ★ 2026-10-16 军事区/资源区点建筑名进来的「建筑详情」页要展示的那栋建筑。
    //   按 id 从 buildings 现取：升级完工 / 30s 轮询刷新后，当前级效果文字自动跟着最新等级走。
    buildInfo () {
      if (!this.buildInfoId) return null
      return this.buildings.find(b => b.id === this.buildInfoId) || null
    },
    // 建筑详情页属于哪个区：'m' 军事区 / 's' 资源区（供 [返回xx区] 与底部导航高亮）
    buildInfoZone () {
      const b = this.buildInfo
      return (b && b.type === 1) ? 's' : 'm'
    },
    // 可建造的建筑(「建造」按钮进去的那一页)
    zonePool () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      return (this.buildingPool || []).filter(p => inZone(p.type))
    },
    buildQueueCount () {
      return this.buildings.filter(b => b.status !== 0).length
    },
    // ★ 2026-10-22 征募告示：只有未过期时层数才计入上限（倒计时随 gatherNow 自动失效）
    buildQueueActiveBonus () {
      const now = this.gatherNow || Date.now()
      return this.buildQueueBonusUntil > now ? (this.buildQueueBonus || 0) : 0
    },
    buildQueueLimit () {
      const base = this.buildQueueBase > 0 ? this.buildQueueBase : 5
      return base + this.buildQueueActiveBonus
    },
    // 正式军官(不含俘虏)
    myOfficers () {
      return (this.officerData.officers || []).filter(o => o.is_captive !== 1)
    },
    // ★ 军官最高等级（后端下发 max_level，默认 150）—— 达到即「满级」，不再升级
    officerMaxLevel () {
      const v = parseInt(this.officerData.max_level)
      return v > 0 ? v : 150
    },
    // ★ 2026-09-29 军官详情页逐人上限：名将 350 / 普通军官 150（后端 officerDetail.max_level）
    detailOfficerMaxLevel () {
      const o = this.officerDetail && this.officerDetail.officer
      if (!o) return 150
      const v = parseInt(o.max_level)
      return v > 0 ? v : this.officerMaxLevel
    },
    // 战俘营: 未出征的俘虏
    // ★ 2026-10-06 用户规则：战俘营**按城市分**（绑在玩家城市上）——
    //   后端下发的 captives 已经是「当前城」的俘虏，容量也只看当前城参谋部等级 × 4。
    //   （2026-09-29 那版跨城汇总已取消；旧后端没有 captives 字段时回落到当前城过滤。）
    captiveOfficers () {
      if (this.officerData && Array.isArray(this.officerData.captives)) {
        return (this.officerData.captives || []).filter(o => o.status !== 1)
      }
      return (this.officerData.officers || []).filter(o => o.is_captive === 1 && o.status !== 1)
    },
    // 宝物总件数（9 种采集珠宝的持有数之和，用于「宝物(N/上限)」展示）
    treasureTotal () {
      return (this.bagTreasures || []).reduce((s, t) => s + (parseInt(t.count) || 0), 0)
    },
    // 参谋部军官位是否已满(招募前先拦一道, 避免点了才报「容量不足」)
    officerFull () {
      const c = this.recruitData.capacity || this.officerData.capacity || 0
      const u = this.recruitData.used !== undefined ? this.recruitData.used : this.officerData.used
      return c > 0 && (u || 0) >= c
    },
    trainCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type !== 4)
    },
    // ★ 2026-09-28 首页点「军队」看到全部兵种（数量为 0 的也显示）+ 训练操作。
    //   城内军队表遍历全部兵种配置（**不含城防兵种 type 4**，城防只在「城防」页展示），
    //   数量从 troops 里取（没有=0），行动行自带 troop_id。
    armyRows () {
      const troops = this.troopsData.troops || []
      return (this.troopsData.cfgs || []).filter(c => c.type !== 4).map(c => {
        const row = troops.find(x => x.troop_id === c.id)
        return Object.assign({}, c, { count: row ? row.count : 0, troop_id: c.id })
      })
    },
    wareSum () {
      const r = this.wareRatio
      return (parseInt(r.food) || 0) + (parseInt(r.steel) || 0) + (parseInt(r.oil) || 0) + (parseInt(r.rare) || 0)
    },
    // 集结令数量(背包里查；背包还没加载时用 /view 下发的 gather_have 兜底)
    gatherCount () {
      const it = (this.bagItems || []).find(x => x.name === '集结令')
      const n = it ? it.count : 0
      return n > 0 ? n : (this.gatherCfg.have || 0)
    },
    // ★ 集结令单次上限：以 /view 下发的 gatherCfg.max 为准（读 ezfy_cfg_limit.gather_max_per_order，
    //   管理端「建筑上限配置」页可维护，默认 99）。
    //   ⚠️ 这里**不能**在没有数据时直接返回 99 去夹输入值 —— 那正是「配了 999 只能用 99」的 bug：
    //   进页面时 orderCalc 还是 null，一改数字就被夹回 99，后面再点[计算]也救不回来了。
    orderCapMax () {
      const m = this.gatherCfg.max || (this.orderCalc && this.orderCalc.gather_max)
      return m > 0 ? m : 99
    },
    // 真正可用的上限 = min(管理端上限, 背包实际持有量)
    gatherMax () { return Math.min(this.orderCapMax, this.gatherCount) },
    // ★ 每个集结令提升的出征上限，同样以接口下发为准（读 ezfy_cfg_item.param1，缺省 10 万）
    orderCapPer () {
      const p = this.gatherCfg.per || (this.orderCalc && this.orderCalc.gather_per)
      return p > 0 ? p : 100000
    },
    // ★ 2026-09-25 出征页「兵种数量搭配」改成一行一个兵种 + 滑动条联动：
    //   本地实时合计本次出兵数量（拖滑块/填数字立刻刷新，不用等点 [计算]）。
    //   口径与后端一致：只算可出征兵种（trainCfgs 已滤掉城防 type=4），0 值不计。
    orderTroopTotal () {
      let sum = 0
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k], 10)
        if (n > 0) sum += n
      }
      return sum
    },
    // 是否超出出征上限（口径同后端 troop_over_cap：管理端关掉上限开关时不判超）
    orderOverCap () {
      const c = this.orderCalc
      if (!c || c.cap_unlimited || !this.orderCapApplies) return false
      return this.orderTroopTotal > (c.troop_cap || 0)
    },
    // 上限文案：没算过时显示 —；运输(5)也纳入携带上限（2026-10-02 用户反馈「运输无上限是bug」）
    orderCapText () {
      const c = this.orderCalc
      if (!c) return '—'
      // ★ 2026-10-02 派遣(8) 非战斗状态/免战期间由后端返回 cap_unlimited=true → 显示「无上限」；其余显示 troop_cap
      return c.cap_unlimited ? '无上限' : this.fmtN(c.troop_cap)
    },
    // ★ 2026-10-02 所有出征类型（含运输5）都做携带上限判定；
    //   只有派遣(8)在非战斗状态/免战期间由后端 cap_unlimited=true 放行。
    orderCapApplies () {
      return true
    },
    // ★ 2026-09-28 随军资源：负重上限 = 所带兵种负重之和 × 装载技术加成。
    //   直接采用 [计算]（orderCalc.carry，后端 ezfyCarryCapOf 已含科技加成）作为唯一口径，
    //   与出征/运输/采集的负重校验一致；选兵种变化时由 orderTroopTotal watcher 重算。
    orderResCap () {
      const c = this.orderCalc
      return c ? (c.carry || 0) : 0
    },
    // ★ 2026-09-29 修正：运输(5)油耗按携带资源量另算、直接从城里扣，**不占部队负重** ——
    //   减油耗会形成回环（填资源→油耗变大→上限变小，但油又不随资源刷新），所以运输用满负重当上限。
    //   其余（含派遣8）油耗按兵种计、需占负重 → 负重 − 油耗。
    orderResUsableCap () {
      const c = this.orderCalc
      if (this.orderType === 5) return Math.max(0, c ? (c.carry || 0) : 0)
      return Math.max(0, this.orderResCap - this.oilUsed)
    },
    // 行军油耗（随军资源占用的负重需要从负重上限里先扣掉）
    oilUsed () {
      const c = this.orderCalc
      return c ? (c.oil_used || 0) : 0
    },
    // 随军资源五项之和（重量 = 占用负重）
    orderResTotal () {
      return ['gold', 'food', 'steel', 'oil', 'rare'].reduce((s, k) => s + this.resQty(k), 0)
    },
    // 没选任何部队 → 随军资源整块禁用（滑块/数字/[最大] 都灰掉，也不能填写）
    // ★ 2026-10-17 增援(6)=驻守，不带资源（盟友间资源运输靠运输(5)）→ 同样禁用
    orderResDisabled () {
      return this.orderTroopTotal <= 0 || this.orderType === 6
    },
    // 是否超出负重上限（红了要玩家减资源或多带部队）
    orderResOver () {
      return !this.orderResDisabled && this.orderResTotal > this.orderResUsableCap
    },
    // 随军资源行列表（沿用兵力行的渲染结构：名称+现有+滑块+数字+[最大]）
    resFields () {
      const r = this.resNames || {}
      return [
        { key: 'gold', name: r.gold },
        { key: 'food', name: r.food },
        { key: 'steel', name: r.steel },
        { key: 'oil', name: r.oil },
        { key: 'rare', name: r.rare }
      ]
    },
    // ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服（要先在「附属野地」页[放弃]）。
    //   判据取后端 WildlandView 下发的 mine —— 它与详情页显示的「归属：我」是同一个来源
    //   （野地记录 → 城市 → UserID），前端不自己猜归属，避免两边口径漂移。
    //   area_type === 3 是玩家城市，走另一套规则（宣战/同盟），这里排除掉。
    isOwnWild () {
      return !!(this.selDetail && this.selDetail.mine &&
        this.selCell && this.selCell.area_type !== 3)
    },
    // ★★ 2026-10-18 原「采集中野地不能打」的前端拦截已随新规则移除
    //   （攻方获胜歼灭守军=驻军+采集部队, 采集资源随军损失）——isWildCollecting 判定不再需要。
    // ★★ 2026-10-13 「出征确认」命令下拉里**要列出的命令**（按 orderNames 的固定顺序：
    //   侦查→掠夺→征服→采集→运输→增援）。判据见 orderAvail —— 与地图详情页那几个命令按钮**同源**，
    //   于是「地图上点不了的」在下拉里也不会出现（用户：还没宣战就只能侦查，那下拉里就只有侦查）。
    orderAvailList () {
      const out = []
      for (let t = 1; t <= 8; t++) if (this.orderAvail(t)) out.push(t)
      return out
    },
    defenceCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type === 4)
    },
    defenceTroops () {
      return (this.troopsData.troops || []).filter(t => t.type === 4)
    },
    // 城防建造队列(复刻 troopDefence.html 的「正在建造」)
    defenceQueues () {
      const ids = this.defenceCfgs.map(t => t.id)
      return (this.queues || []).filter(q => ids.indexOf(q.troop_id) >= 0)
    },
    // 兵种详情页当前兵种
    troopView () {
      return (this.troopsData.cfgs || []).find(t => t.id === this.troopViewId) || null
    },
    // 训练确认页可训练上限: 资源 / 人口(城防为城防空间) 取最小
    maxTrainable () {
      if (!this.trainSel) return 0
      const c = this.trainSel.cost || {}
      const ct = this.troopsData.city || this.city || {}
      const caps = []
      if (c.food > 0) caps.push(Math.floor((ct.food || 0) / c.food))
      if (c.steel > 0) caps.push(Math.floor((ct.steel || 0) / c.steel))
      if (c.oil > 0) caps.push(Math.floor((ct.oil || 0) / c.oil))
      if (c.rare > 0) caps.push(Math.floor((ct.rare || 0) / c.rare))
      if (this.trainMode === 'defence') {
        // 城防设施每个占 1 点城防空间(复刻 GameServiceImpl 的 used += count)
        const space = (this.troopsData.defence_space || 0) - (this.troopsData.defence_space_used || 0)
        caps.push(Math.max(0, space))
      } else if (this.trainSel.pop > 0) {
        caps.push(Math.floor(this.freePop / this.trainSel.pop))
      }
      const m = caps.length ? Math.min.apply(null, caps) : 0
      return Math.max(0, m)
    },
    // 训练确认页预计耗时(复刻 createTroop.html 的「时间」: 单个耗时 × 数量 ÷ 并行工厂数)
    // ★ 2026-10-21 [全部工厂] 并行数 = 还有空位的可用军工厂数；
    //   数量比厂数少时只会有「数量」座厂真正分到队列，所以再夹一次 min(可用厂数, 数量)。
    trainEstimateText () {
      if (!this.trainSel) return '0秒'
      const n = parseInt(this.trainCount) || 0
      const par = this.trainMode === 'defence'
        ? 1
        : (this.trainSplit ? Math.max(1, this.splitFactoryCount) : 1)
      // 后端按「每条队列分到 ceil(总数/厂数) 个」建队列，最长的那条决定总耗时；
      // 不能用 ceil(单个耗时 × 总数 / 厂数)，否则 7 秒×10 个分 3 厂会算成 24 秒（实际 28 秒）。
      return this.durText(this.trainSel.train_time * Math.ceil(n / par))
    },
    attackTroops () {
      return (this.troopsData.troops || []).filter(t => t.type !== 4)
    },
    totalTroops () {
      let sum = 0
      for (const t of this.troopsData.troops) sum += t.count
      return sum
    },
    // ★★ 2026-10-25 军官属性页「军事/学识/后勤」后面的 (+N) 到底表示什么 —— 用户最终口径：
    //
    //	(+N) = **比「军官池原始属性」多出来的全部加成** = 属性点加点 + 装备 + 套装
    //
    //	用户给的两组对照：
    //	  · 池子 300/200/150、349 点全加军事 → `军事：649(+349)`（只有加点，没有装备）
    //	  · 装备+套装 军事+1100+96 / 后勤+550+49 / 学识+550+49
    //	    → `军事：1845(+1196) 学识：799(+599) 后勤：749(+599)`（加点已含在 649 里）
    //
    //	★ 上一版我按「只算装备」处理（total − military），结果玩家把 349 点加完看不到任何变化，
    //	  用户反馈「穿装备能展示，属性的没展示出来」→ 改成 total − base_*。
    //	  base_military/logistics/learning 就是池子原始属性（后端 officerBaseAttr），
    //	  所以 total − base 恒等于「加点 + 装备 + 套装」，两处示例都对得上。
    //	  负值按 0（老数据 base 可能缺失/为 0，防御性处理）。
    officerAttrPlus () {
      const o = (this.officerDetail && this.officerDetail.officer) || null
      const d = (total, base) => Math.max(0, (total || 0) - (base || 0))
      if (!o) return { mil: 0, log: 0, lea: 0 }
      return {
        mil: d(o.military_total, o.base_military),
        log: d(o.logistics_total, o.base_logistics),
        lea: d(o.learning_total, o.base_learning)
      }
    },
    areaCount () {
      return this.buildings.filter(b => b.type >= 1 && b.type <= 4).length
    },
    // ★★ 2026-10-25 修「城市状态页 建筑: 0座 (军事+资源区 0/66)」：
    //   ① 数量恒 0 → 进页没拉 /buildings（见 go() 的 citystatus 分支，已合并修掉死代码）；
    //   ② 上限原来**写死 66**（33+33 时代的旧值），后端 ezfy_cfg_limit 现在是 36+36=72，
    //      管理端还能改 → 改成跟着 /view、/buildings 下发的 military_cap/resource_cap 走。
    areaCap () {
      return (this.militaryCap || 0) + (this.resourceCap || 0)
    },
    // ★ 当前分区（军事区/资源区）各自的建筑数（分开统计，不再混用 areaCount）
    zoneCount () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      return this.buildings.filter(b => inZone(b.type)).length
    },
    // ★ 当前分区上限：军事区/资源区各 36（与后端 ezfy_cfg_limit 现值一致）
    zoneCap () {
      return this.zone === 'm' ? this.militaryCap : this.resourceCap
    },
    factoryTotal () {
      return this.buildings.filter(b => b.building_id === 14).reduce((s, b) => s + b.level, 0)
    },
    factories () {
      return this.buildings.filter(b => b.building_id === 14)
    },
    // ★ 2026-10-05 「只展示选中的军工厂等级」：不再列「各座9/9/9/...」，
    //   只显示本城军工厂的最高等级（[仅此工厂] 训练用的那座，也是队列上限的直观值）。
    factoryLevels () {
      let m = 0
      for (const f of this.factories) if (f.level > m) m = f.level
      return m || 0
    },
    // ★ 2026-10-21 [全部工厂] 平分的分母 = 还有空位的可用军工厂数。
    //   后端 /troops 下发 factories: [{id, level, used, free, ready}]，
    //   这里按兵种需求过滤，并排除升级中 / 队列已满的厂。
    //   （旧接口没有 factories 字段时退回旧口径，避免页面空白。）
    eligibleFactories () {
      const need = this.trainSel ? (this.trainSel.need_factory || 0) : 0
      const list = (this.troopsData && this.troopsData.factories) || []
      if (list.length) {
        return list.filter(f => f.ready !== false && (f.level || 0) >= need &&
          (f.used || 0) < (f.level || 0)).length
      }
      return this.factories.filter(f => need <= 0 || f.level >= need).length
    },
    // 实际会参与平分的厂数：可用厂数不会超过本次训练数量（数量 1 时只开 1 条队列）。
    splitFactoryCount () {
      const avail = this.eligibleFactories
      const n = parseInt(this.trainCount) || 0
      return n > 0 ? Math.min(avail, n) : avail
    },
    factoryFree () {
      return Math.max(1, this.factoryTotal - this.queues.length)
    },
    isLeader () {
      return !!(this.myCorps && this.profile.user_id && this.myCorps.leader_user_id === this.profile.user_id)
    },
    // ★ 军团邮件权限：军团长 或 副团长
    canMailCorps () {
      return this.isLeader || this.myCorpsTitle === '副团长'
    },
    // ★ 首页外露的公告 = 后端按「首页公告条数」配置下发的 home_notices（默认 1 条）
    topNotices () {
      if (this.homeNotices && this.homeNotices.length) return this.homeNotices
      return (this.notices || []).filter(n => n.is_top).slice(0, 1)
    },
    // ★ 商城：按分类过滤 + 分页（分类为空 = 全部）
    mallFiltered () {
      const cat = this.mallCat
      const list = this.mallItems || []
      return cat ? list.filter(i => i.category === cat) : list
    },
    // ★ 2026-10-20 分类页签显示数量，玩家一眼知道哪类有货、有多少。
    // ★ 2026-10-25 分页列表里的分类分组标题已按用户要求去掉 → 这个计数没人用了，一并删除
    //   （顶部 [全部][资源][加速]… 那排筛选按钮本来就不带数字）。
    // ============ ★ 通用「检索 + 分页」小工具（背包 / 装备列表共用） ============
    // 后端一次性把列表下发，检索与分页都在前端做（这些列表不会大到需要服务端分页）。
    // ★ 背包分类（按首次出现顺序，与商城的归类口径一致：后端 ezfyItemCategory）
    bagCats () {
      const out = []
      ;(this.bagItems || []).forEach(i => {
        const c = i.category || '其他'
        if (out.indexOf(c) < 0) out.push(c)
      })
      return out
    },
    bagFiltered () {
      let list = this.bagItems || []
      if (this.bagCat) list = list.filter(i => (i.category || '其他') === this.bagCat)
      const w = (this.bagWord || '').trim().toLowerCase()
      if (!w) return list
      return list.filter(i => String(i.name || '').toLowerCase().includes(w) ||
        String(i.description || '').toLowerCase().includes(w))
    },
    bagTotalPages () { return Math.max(1, Math.ceil(this.bagFiltered.length / this.bagPageSize)) },
    bagPaged () {
      const p = Math.min(Math.max(1, this.bagPage), this.bagTotalPages)
      return this.bagFiltered.slice((p - 1) * this.bagPageSize, p * this.bagPageSize)
    },
    // ★ 2026-10-11 背包「军官装备」tab 分页（与道具 tab 同一套写法）
    bagEquipTotalPages () { return Math.max(1, Math.ceil((this.bagEquips || []).length / this.bagEquipPageSize)) },
    bagEquipPaged () {
      const p = Math.min(Math.max(1, this.bagEquipPage), this.bagEquipTotalPages)
      return (this.bagEquips || []).slice((p - 1) * this.bagEquipPageSize, p * this.bagEquipPageSize)
    },
    // 我的装备（可按 名称 / 部位 / 套装 / 类型 检索）
    // ★ 2026-09-25：set_id → 套装配置（含加成）。装备表/商城表/军官装备表都从这里查，
    //   这样「没拥有的套装」也能看到穿齐给什么（原来只有 /officers/equipments 的 sets，
    //   且只含我至少有一件的 → 商城里看不到加成，玩家不知道买哪套）。
    setMap () {
      const m = {}
      for (const s of (this.allSets || [])) m[s.id] = s
      return m
    },
    // ★ 2026-09-25 确认条的定位（用户反馈「军官详情的一键穿戴，确认跑到页面最上面，要滚上去才能点」）：
    //   和提示条 msgStyle 同一套思路 —— 浮在**刚才点的那个控件**旁边，而不是永远钉在页面顶部。
    //   与提示条的区别：确认条更高（文案 + 可选输入框 + [确定][取消] 两个按钮），所以：
    //     ① 优先放控件**下方**；下方放不下就翻到**上方**；上下都放不下就夹进视口；
    //     ② 高度用 askH（渲染后实测），首次用估算值兜底，保证按钮在屏幕最底部时也不会被切掉。
    //   ⚠️ _lastClicked 是 mounted 里那个**捕获阶段** click 监听记下的 <a>/<button>；
    //      ask() 都是在点击处理里调的，所以这里拿到的一定是触发它的那个控件。
    //   ⚠️ 必须是 computed（不能放 methods）：模板里是 :style="askStyle"，
    //      方法会被当成函数对象直接绑上去。
    askStyle () {
      const vw = window.innerWidth
      const vh = window.innerHeight
      const w = Math.min(360, vw - 16)
      const H = this.askH || (this.askBox.input ? 150 : 104)
      const el = this._lastClicked
      const rect = el && typeof el.getBoundingClientRect === 'function' ? el.getBoundingClientRect() : null
      const inView = rect && rect.left < vw && rect.right > 0 && rect.top < vh && rect.bottom > 0
      let left, top
      if (rect && inView) {
        left = Math.round(rect.left + rect.width / 2 - w / 2)
        left = Math.max(8, Math.min(left, vw - w - 8))
        const below = rect.bottom + 8
        if (below + H <= vh - 8) top = below
        else if (rect.top - 8 - H >= 8) top = rect.top - 8 - H
        else top = below
      } else {
        left = Math.round((vw - w) / 2)
        top = 8
      }
      top = Math.max(8, Math.min(Math.round(top), vh - H - 8))
      return { position: 'fixed', left: left + 'px', top: top + 'px',
        width: w + 'px', zIndex: 10000 }
    },
    // ★ 2026-09-25 套装一览要显示的内容：默认「我有的」；切到全部时列**所有套装**（含没拥有的），
    //   两种来源统一成同一形状（have/need/active/slots/effect + 加成字段），模板只认一套字段。
    setListShown () {
      if (!this.setShowAll) {
        return this.mySetProgress.map(s => Object.assign({}, s, {
          owned: s.have, have: s.have, need: s.need, active: s.active,
          slots: (this.setOf(s.id) && this.setOf(s.id).slots) || []
        }))
      }
      return (this.allSets || []).map(s => Object.assign({}, s, {
        have: s.owned || 0,
        need: Math.max(0, (s.parts || 0) - (s.owned || 0)),
        active: (s.parts || 0) > 0 && (s.owned || 0) >= s.parts
      }))
    },
    // ★「我的套装」：只列**玩家已拥有**的套装（从背包聚合），并算还差几件才生效。
    //   原来这里铺的是「全部套装」= 图鉴，玩家根本分不清哪个是自己有的。
    mySetProgress () {
      const bag = this.equipData.bag || []
      const have = {}
      bag.forEach(e => { if (e.set_id) have[e.set_id] = (have[e.set_id] || 0) + 1 })
      return (this.equipData.sets || [])
        .filter(s => have[s.id])
        .map(s => Object.assign({}, s, {
          have: have[s.id],
          need: Math.max(0, (s.parts || 0) - have[s.id]),
          active: (s.parts || 0) > 0 && have[s.id] >= s.parts
        }))
    },
    equipFiltered () {
      const w = (this.equipWord || '').trim().toLowerCase()
      const list = (this.equipData && this.equipData.bag) || []
      if (!w) return list
      return list.filter(e => [e.name, e.slot, e.type, e.set_name, e.series, e.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
    },
    // ★ 2026-09-29 「我的装备」按同名同件聚合展示（名称 ×N），减少分页数；
    //   状态列判断还有几件可穿戴（未穿戴件数）。细节行沿用 first 的字段，无需改。
    equipGroups () {
      const map = {}, order = []
      for (const e of this.equipFiltered) {
        const k = [e.name, e.level, e.slot, e.type].join('|')
        if (!map[k]) {
          map[k] = Object.assign({}, e, { first: e, count: 0, worn: 0 })
          order.push(map[k])
        }
        map[k].count++
        if (e.worn) map[k].worn++
      }
      return order
    },
    equipTotalPages () { return Math.max(1, Math.ceil(this.equipGroups.length / this.equipPageSize)) },
    equipPaged () {
      const p = Math.min(Math.max(1, this.equipPage), this.equipTotalPages)
      return this.equipGroups.slice((p - 1) * this.equipPageSize, p * this.equipPageSize)
    },
    // 装备图鉴
    equipAllFiltered () {
      const w = (this.equipAllWord || '').trim().toLowerCase()
      const list = (this.equipData && this.equipData.all) || []
      if (!w) return list
      return list.filter(e => [e.name, e.slot, e.type, e.set_name, e.series, e.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
    },
    equipAllTotalPages () { return Math.max(1, Math.ceil(this.equipAllFiltered.length / this.equipAllPageSize)) },
    equipAllPaged () {
      const p = Math.min(Math.max(1, this.equipAllPage), this.equipAllTotalPages)
      return this.equipAllFiltered.slice((p - 1) * this.equipAllPageSize, p * this.equipAllPageSize)
    },
    // ★ 2026-10-15 名将图鉴分页（与装备图鉴同款，纯前端切片，无任何提示）
    generalTotalPages () { return Math.max(1, Math.ceil((this.generalData.generals || []).length / this.generalPageSize)) },
    generalPaged () {
      const p = Math.min(Math.max(1, this.generalPage), this.generalTotalPages)
      return (this.generalData.generals || []).slice((p - 1) * this.generalPageSize, p * this.generalPageSize)
    },
    // ★ 2026-09-29 军官详情「装备背包 / 已穿戴装备」同一个装备叠加展示（名称 × N），
    //   有空余部位就能继续穿戴（对应部位没被占用），没空余就不能穿戴。
    //   分组 key = cfg_id（同配置的装备实例 = 同一件装备）；老数据没有 cfg_id 时兜底 name|slot|set_id|tier。
    // 已穿戴装备分组（装备 tab）：同 cfg 的叠加成一行，数量 >1 时显示 ×N
    officerEquipGroups () {
      // ★ 2026-10-10 背包装备改懒加载：用 officerBag 建 cfg 映射（进装备页前已按需拉取）
      const bag = this.officerBag || []
      const cfgOf = {}
      for (const e of bag) if (e.cfg_id) cfgOf[e.id] = e.cfg_id
      const groups = {}
      const order = []
      for (const e of (this.officerDetail && this.officerDetail.equipped) || []) {
        const key = cfgOf[e.id] ? 'c' + cfgOf[e.id] : this.equipGroupKey(e)
        let g = groups[key]
        if (!g) {
          g = { key: key, name: e.name, slot: e.slot || e.type || '', type: e.type,
                set_id: e.set_id, set_name: e.set_name, tier: e.tier,
                tier_name: e.tier_name, level: e.level, first: e, items: [], count: 0 }
          groups[key] = g
          order.push(key)
        }
        g.items.push(e)
        g.count++
      }
      return order.map(k => groups[k])
    },
    // 当前军官已占用的部位集合（决定背包装备还能不能穿戴）
    officerEquipSlots () {
      const s = {}
      for (const g of this.officerEquipGroups) s[g.slot] = true
      return s
    },
    // ★ 2026-10-25 用户需求：「已穿戴装备」要列出**所有部位**，没穿的显示「无」，
    //   否则玩家不知道有哪些部位可以穿戴。
    //   固定清单 = 游戏里实际存在的 14 个部位（按身体从上到下排），
    //   实际穿戴里若出现清单外的部位（以后新增/老数据），自动追加到末尾，不丢信息。
    //   一个部位正常只穿一件 → 同部位多件时只取第一件展示（正常不会发生）。
    officerEquipSlotRows () {
      const ORDER = ['头部', '肩部', '胸部', '腰部', '足部', '手部', '武器', '防具', '珠宝',
        '挂件', '饰品', '勋章', '名将勋章', '名将史册']
      const bySlot = {}
      for (const g of this.officerEquipGroups) {
        if (!g.slot) continue
        if (!bySlot[g.slot]) bySlot[g.slot] = g
      }
      const slots = ORDER.slice()
      for (const g of this.officerEquipGroups) {
        if (g.slot && slots.indexOf(g.slot) < 0) slots.push(g.slot)
      }
      return slots.map(s => ({ slot: s, g: bySlot[s] || null }))
    },
    // 军官详情里的装备背包：按 cfg 叠加（只算未穿戴的），带 canEquip（该部位还有空余就能穿）
    officerBagGroups () {
      const w = (this.officerBagWord || '').trim().toLowerCase()
      // ★ 2026-10-10 背包装备改懒加载：从 officerBag 取（切「装备背包」tab 前已按需拉取）
      const bag = this.officerBag || []
      const groups = {}
      const order = []
      const slots = this.officerEquipSlots
      for (const e of bag) {
        if (!e) continue
        if (e.worn) continue
        const key = this.equipGroupKey(e)
        let g = groups[key]
        if (!g) {
          g = { key: key, name: e.name || '', slot: e.slot || e.type || '', type: e.type,
                set_id: e.set_id || 0, set_name: e.set_name || '', series: e.series || '', tier: e.tier || 0,
                tier_name: e.tier_name || '', level: e.level || 0, first: e, items: [], count: 0,
                canEquip: false }
          groups[key] = g
          order.push(key)
        }
        g.items.push(e)
        g.count++
      }
      let list = order.map(k => groups[k])
      // 搜索词作用于分组行
      if (w) list = list.filter(g => [g.name, g.slot, g.type, g.set_name, g.series, g.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
      // 部位有空余 = 军官身上还没穿同部位装备 → 可以穿戴（后端同部位唯一兜底）
      for (const g of list) g.canEquip = !slots[g.slot]
      return list
    },
    officerBagTotalPages () {
      return Math.max(1, Math.ceil(this.officerBagGroups.length / this.officerBagPageSize))
    },
    officerBagPaged () {
      const p = Math.min(Math.max(1, this.officerBagPage), this.officerBagTotalPages)
      return this.officerBagGroups.slice((p - 1) * this.officerBagPageSize, p * this.officerBagPageSize)
    },
    mallTotalPages () {
      return Math.max(1, Math.ceil(this.mallFiltered.length / this.mallPageSize))
    },
    mallPaged () {
      const p = Math.min(Math.max(1, this.mallPage), this.mallTotalPages)
      return this.mallFiltered.slice((p - 1) * this.mallPageSize, p * this.mallPageSize)
    },
    // 某坐标是否海城(海洋地形 8)
    // ★ 海城判据：**以后端返回的 is_sea 为准**。
    //   后端规则：海城 = 建在「沿海平原」(ezfyTerrainEx==9) 上。
    //   旧版前端自己用坐标哈希算「地形==8(海洋)」，真正的海城被判成陆地城市。
    isSeaAt () {
      return ct => {
        if (!ct) return false
        if (typeof ct.is_sea === 'boolean') return ct.is_sea
        return !!ct.is_sea
      }
    },
    // ★ 当前城市类型文案：统一为「海城 / 陆城」（后端 city_kind 为准；2026-10-25 用户要求短文案）
    cityKindLabel () {
      if (this.cityKindRaw) return this.cityKindRaw
      return this.cityIsSea ? '海城' : '陆城'
    },
    // ★ 资源详情页显示对应科技名（1 种植 / 2 炼钢 / 3 勘探 / 4 冶炼）
    resTechName () {
      const m = { food: '种植技术', steel: '炼钢技术', oil: '勘探技术', rare: '冶炼技术' }
      return m[this.resType] || ''
    },
    // ★ 聊天分页总页数（后端按时间降序返回）
    chatTotalPages () {
      return Math.max(1, Math.ceil(this.chatTotal / this.chatSize))
    },
    // ★ 军队动态 = 行进/战斗/返航中的部队(不含驻守采集/驻军);
    //   驻军 = 驻守采集(status=1) + 出站驻军(status=3, 增援盟友城市, ★ 2026-10-02)
    dynMarch () {
      return this.dynamics.filter(o => o.status !== 1 && o.status !== 3)
    },
    dynStation () {
      return this.dynamics.filter(o => o.status === 1 || o.status === 3)
    },
    // ★ 2026-09-25 出征队列页的数据 = 军队动态全集（行进/战斗/返航/驻守都在内，一次展示完）
    queueItems () {
      // ★ 2026-09-28 采集实况 _lg + 倒计时实况 _lt 都依赖 gatherNow，每秒一起重算
      return (this.dynamics || []).map(o => this.withLg(this.withLive(o)))
    },
    // ★ 2026-09-28 安抚冷却剩余毫秒（依赖 placateNow 每秒重算，到点自动放行按钮）
    //   后端的 cd_left 是拉 /view 那一刻的快照，这里用本地时间基准往前推，避免只靠刷新。
    placateCdLeft () {
      const total = (this.placate && this.placate.cooldown_min ? this.placate.cooldown_min : 15) * 60000
      const snap = (this.placate && this.placate.cd_left) || 0
      if (snap <= 0) return 0
      // 快照剩余 = snap（拉接口的那一刻）；本地已经流逝的时间 = 从 load 到现在的差
      const elapsed = this.placateNow && this.placateAt ? this.placateNow - this.placateAt : 0
      return Math.max(0, Math.min(total, snap - elapsed))
    },
    // ★ 军情分区分页（默认每页 5 条，可上一页/下一页）
    dynMarchTotalPages () {
      return Math.max(1, Math.ceil(this.dynMarch.length / this.dynSize))
    },
    dynStationTotalPages () {
      return Math.max(1, Math.ceil(this.dynStation.length / this.dynStationSize))
    },
    // ★ 2026-10-08 出征队列分页（数据集 = dynamics 全集，与 queueItems 同源）
    queueTotalPages () {
      return Math.max(1, Math.ceil((this.dynamics || []).length / this.queueSize))
    },
    queuePaged () {
      const list = this.dynamics || []
      const p = Math.min(Math.max(1, this.queuePage), this.queueTotalPages)
      return list.slice((p - 1) * this.queueSize, p * this.queueSize).map(o => this.withLg(this.withLive(o)))
    },
    // ★ 战场指挥室：本回合剩余秒数 / 倒计时条百分比（最后 5 秒条变红）
    battleLeftText () {
      return Math.max(0, Math.ceil(this.battleLeftMs / 1000)) + ' 秒'
    },
    battleBarPct () {
      const total = this.battleData.round_ms || 30000
      const pct = this.battleLeftMs * 100 / total
      return Math.max(0, Math.min(100, pct))
    },
    // ★ 2026-10-10 战场位置条：每行 = 一对「攻-守」，**左右并排相向**（用户反馈「一行只有一个兵种」：
    //   旧实现按 troop_id 配对，攻守兵种不同 → 几乎每行都只剩一侧）。
    //   现在改为**按下标逐对配对**（各自按 troop_id 排序后 zip）：第 i 行 = 攻方第 i 个兵种 vs
    //   守方第 i 个兵种，保证每行都有攻有守、面对面。一方兵种更多时多出来的行另一侧留空。
    battlePosRows () {
      const byTroop = (a, b) => a.troop_id - b.troop_id
      const atk = (this.battleData.attackers || []).filter(u => u && u.troop_id).slice().sort(byTroop)
      const def = (this.battleData.defenders || []).filter(u => u && u.troop_id).slice().sort(byTroop)
      const n = Math.max(atk.length, def.length)
      const rows = []
      // ★ 2026-10-13 位置条的 left 在这里**一次算好**（不再由模板各自调 battlePosLeft）——
      //   这样才能做「同一行保序」：我方图标恒在敌方左边，贴身时不重叠、也不互相穿过。
      //   ★★ 保序**只把敌方往右推，我方位置永远保持真实** ——
      //     否则「敌方战列舰」(行1) 会被推到左边，和同 pos 的「敌方航母」(行2) 错开，
      //     看起来就不是「贴身」了。
      // ★★ 2026-10-20 守方视角（用户反馈「我是防守方，动画攻防位置不对、颜色不对」）：
      //   位置条固定「我方在左、敌方在右」—— 攻方看就是原来的「攻左守右」；
      //   守方看则把整条战线**镜像**（我方守军左、敌方攻军右），颜色也随之变
      //   「我方绿 / 敌方红」，与表格行、行动日志的配色口径一致。
      const isAtk = !!this.battleData.is_atk
      const GAP = 9 // % —— 约等于一个 26px 图标在手机屏行宽里的占比
      for (let i = 0; i < n; i++) {
        const a = atk[i] || null
        const d = def[i] || null
        let al = a ? this.battlePosLeft(a) : null
        let dl = d ? this.battlePosLeft(d) : null
        if (!isAtk) {
          if (al !== null) al = 100 - al
          if (dl !== null) dl = 100 - dl
        }
        if (al !== null && dl !== null) {
          if (isAtk && dl - al < GAP) {
            al = Math.max(3, dl - GAP)   // 我方(攻)左推，敌方(守)不动 → 同 pos 的单位纵向对齐
          } else if (!isAtk && al - dl < GAP) {
            al = Math.min(97, dl + GAP)  // 我方(守)不动，敌方(攻)右推 → 同 pos 的单位纵向对齐
          }
        }
        if (al !== null && al < 3) al = 3
        if (al !== null && al > 97) al = 97
        if (dl !== null && dl < 3) dl = 3
        if (dl !== null && dl > 97) dl = 97
        rows.push({ id: i, atk: a, def: d, atkLeft: al, defLeft: dl })
      }
      return rows
    },
    // ★ 战场指挥室：把行动日志按回合分组，并让**最新回合排最上**（旧回合往下沉）
    battleRounds () {
      const acts = (this.battleData && this.battleData.actions) || []
      const groups = []
      let cur = null
      for (const a of acts) {
        if (/^第\d+回合:/.test(a)) {
          cur = { line: a, items: [] }
          groups.push(cur)
        } else {
          if (!cur) { cur = { line: null, items: [] }; groups.push(cur) }
          cur.items.push(a)
        }
      }
      return groups.slice().reverse()
    },
    // ★ 商城装备：部位筛选 + 检索 + 分页
    //   ★ 2026-10-14 用户「商城 装备 下面再加一个 [套装] tab 专门是套装散件」：
    //     `equipment` 分栏只留**非套装件**（set_id=0 的散件/系列件），
    //     新增 `set` 分栏只看**套装件**（set_id>0）。两栏各自独立分页/检索。
    shopSlots () {
      return (this.equipShop.slots || []).map(s => s.slot)
    },
    // isSetTab：当前是否在「套装」分栏（决定 shopAll 取 set_id>0 还是 =0）
    isSetsTab () {
      return this.mallTab === 'set'
    },
    shopAll () {
      let list = []
      ;(this.equipShop.slots || []).forEach(s => { list = list.concat(s.pieces || []) })
      // ★ 按分栏切分：套装件（set_id>0）只在「套装」栏显示，非套装件只在「装备」栏显示
      const wantSet = this.isSetsTab
      list = list.filter(p => wantSet ? (p.set_id > 0) : !(p.set_id > 0))
      if (this.shopSlot) list = list.filter(p => p.slot === this.shopSlot)
      const w = (this.shopWord || '').trim()
      if (w) list = list.filter(p => (p.name || '').indexOf(w) >= 0 || (p.slot || '').indexOf(w) >= 0)
      return list
    },
    shopTotalPages () {
      return Math.max(1, Math.ceil(this.shopAll.length / this.shopSize))
    },
    shopPaged () {
      const p = Math.min(Math.max(1, this.shopPage), this.shopTotalPages)
      return this.shopAll.slice((p - 1) * this.shopSize, p * this.shopSize)
    },
    // ★ 宝箱奖池（点宝箱名字才展开）
    chestPoolCur () {
      return (this.chestData.chests || []).find(x => x.id === this.chestPoolId) || null
    },
    chestPoolAll () {
      const ch = this.chestPoolCur
      if (!ch) return []
      let list = ch.pool || []
      const w = (this.chestPoolWord || '').trim()
      if (w) list = list.filter(p => (p.name || '').indexOf(w) >= 0)
      return list
    },
    chestPoolTotalPages () {
      return Math.max(1, Math.ceil(this.chestPoolAll.length / this.chestPoolSize))
    },
    chestPoolPaged () {
      const p = Math.min(Math.max(1, this.chestPoolPage), this.chestPoolTotalPages)
      return this.chestPoolAll.slice((p - 1) * this.chestPoolSize, p * this.chestPoolSize)
    },
    // ★★ 2026-10-08 「军情三区（军队动态/驻军/军情警讯）分页条始终显示」：
    //   原来分页条 `v-if="length > size"`，条数不足一页时整条不渲染 —— 切城市重新查询后
    //   条数变少（或本来就只有几支），玩家看不到任何分页入口，以为这几个列表没有分页。
    //   现在只要有数据就渲染分页条（第 1/1 页 + 灰掉的上一页/下一页）。
    //   ★ 同时把「显示用页码」统一夹紧：数据变少时（部队返航 / 切城重查）原页码可能越界，
    //     原来分页条写「第 3/2 页」而列表显示的却是第 2 页内容（列表内已夹紧、分页条没夹紧）。
    //     列表切片与分页条现在共用这三个值，两边永远一致。
    dynPageCur () {
      return Math.min(Math.max(1, this.dynPage), this.dynMarchTotalPages)
    },
    dynStationPageCur () {
      return Math.min(Math.max(1, this.dynStationPage), this.dynStationTotalPages)
    },
    repPageCur () {
      return Math.min(Math.max(1, this.repPage), this.repTotalPages)
    },
    dynMarchPaged () {
      const p = this.dynPageCur
      return this.dynMarch.slice((p - 1) * this.dynSize, p * this.dynSize).map(o => this.withLive(o))
    },
    dynStationPaged () {
      const p = this.dynStationPageCur
      return this.dynStation.slice((p - 1) * this.dynStationSize, p * this.dynStationSize).map(o => this.withLg(this.withLive(o)))
    },
    repTotalPages () {
      return Math.max(1, Math.ceil(this.reports.length / this.repSize))
    },
    repPaged () {
      const p = this.repPageCur
      return this.reports.slice((p - 1) * this.repSize, p * this.repSize)
    },
    // ★ 公告分页（「公告也变成分页，下一页上一页那种」），与军情同一套写法
    noticeTotalPages () {
      return Math.max(1, Math.ceil(this.notices.length / this.noticeSize))
    },
    // ★ 交易行分页（服务端分页：后端返回 total/page/size）
    exchangeTotalPages () {
      return Math.max(1, Math.ceil(this.exchangeTotal / this.exchangeSize))
    },
    exchangeMTotalPages () {
      return Math.max(1, Math.ceil(this.exchangeMTotal / this.exchangeMSize))
    },
    // ★ 2026-10-05 该资源是否已到「资源最大值」。
    //   到顶后产量**不再累加**（后端 calcResourceD → ezfyAddResMax：cur >= max 直接原样返回），
    //   所以玩家会看到数字不动 —— 这里明确标「已满」，避免被当成「资源不涨」的 bug 上报。
    isResFull (k) {
      const max = Number((this.resMax || {})[k] || 0)
      const cur = Number((this.city || {})[k] || 0)
      return max > 0 && cur >= max
    },
    noticePaged () {
      const p = Math.min(Math.max(1, this.noticePage), this.noticeTotalPages)
      return this.notices.slice((p - 1) * this.noticeSize, p * this.noticeSize)
    },
    // 翻页步长 = 一整屏(2r+1)。地图方向按钮：左右动 x（左-右+），上下动 y（上+下-）。
    mapStep () {
      return this.mapR * 2 + 1
    },
    mapRows () {
      // 边长以**实际返回的格子数**为准(后端返回 n×n), 避免与 mapR 不一致时整表错位
      const n = this.mapCells.length
      const size = n > 0 && Number.isInteger(Math.sqrt(n)) ? Math.round(Math.sqrt(n)) : this.mapR * 2 + 1
      const rows = []
      for (let i = 0; i < size; i++) {
        rows.push(this.mapCells.slice(i * size, (i + 1) * size))
      }
      return rows
    },
    // 三种迁城道具的持有数量，模板里直接用（'low'/'high'/'sea' → 个数）
    moveItemCounts () {
      const out = { low: 0, high: 0, sea: 0 }
      for (const it of (this.moveItems || [])) {
        if (out[it.code] !== undefined) out[it.code] = it.count || 0
      }
      return out
    },
    // ★★ 2026-10-25 当前格子是否已收藏 —— **必须是 computed，不能是 methods**：
    //   模板写的是 {{ ezfy.isCellStarred ? '已收藏' : '收藏' }}（不带括号），
    //   定义在 methods 里时 ezfy.isCellStarred 取到的是**函数对象**（恒为真）
    //   → ① 未收藏也显示「已收藏」；② addStar 里 if (this.isCellStarred) 恒真，
    //   永远走「取消收藏」分支 → 点多少次都收藏不上（本次 bug 的根因）。
    //   ★ 唯一来源 = 收藏列表（mapStars）：按钮文字必须和玩家点开「收藏列表」看到的一致。
    //   ★ 用 Number() 比较：后端 x/y 是 bigint，万一序列化成字符串，严格 === 会漏判。
    //   ★ 不再读 selDetail.is_starred —— 详情是异步的，切格子瞬间可能还是上一块地的快照。
    isCellStarred () {
      const c = this.selCell
      if (!c) return false
      return (this.mapStars || []).some(s => Number(s.x) === Number(c.x) && Number(s.y) === Number(c.y))
    }
  },
  mounted () {
    // ★★ 2026-10-18 家园退出 → 二战联动（跨端口场景）：
    //   二战 token 存在 8081/8082 自己的 localStorage 里，家园在 8080 退出时**清不到它**
    //   （localStorage 按 origin 隔离）→ 刷新后仍显示「已登录」，玩家反馈「退了还有缓存」。
    //   判据：有 ezfy_token 却拿不到 jy_token（localStorage 和 cookie 都没有）。
    //   而二战登录 / SSO **必然同时下发家园 token**（后端 home_token），所以「只有二战 token、
    //   没有家园 token」只可能出现在家园已退出（或被清）之后 → 直接清掉二战登录态回登录框。
    if (this.isLogin && !this.$store.state.token) {
      this.$store.commit('ezfyLogout')
    }
    // ★★ 2026-10-18 家园换人（A 退出、B 登录）→ 本地二战 token 还是 A 的，必须作废重换。
    //   二战页自己没有退出入口，玩家换号只能「家园退出 A → B 登录家园 → 再进二战」；
    //   若这里不比对，8081/8082 会带着 A 的 ezfy_token 直接进游戏 = 又一种账号串台。
    if (this.isLogin && this.homeUid && this.ezfyUid && this.homeUid !== this.ezfyUid) {
      this.$store.commit('ezfyLogout')
    }
    // ★★ 2026-10-15 区名写活：先拉 /api/ezfy/zone 拿本区信息（标题/选服列表），再走登录流程。
    //   接口是公开的，未登录也要能拿到，好让登录页标题/选服显示正确区名。
    this.loadZone()
    // ★ 2026-10-12 二战独立登录：未登录时不启动游戏（不打接口、不起轮询，避免 401 弹跳），
    //   根组件只渲染二战自己的登录界面；登录成功后由 wwiiLogin() 重载进入游戏。
    if (!this.isLogin) {
      document.title = this.zoneTitle
      // ★ 2026-10-12 家园已登录（有家园 token）→ 静默换取二战登录态，不再要求二次登录；
      //   家园没登录就安静地显示下面二战自己的登录框。
      // ★★ 2026-10-17 二战独立部署：URL 带 sso_token（家园跨服跳转拼的）→ 优先走它换二战 token，
      //   与「本地 jy_token」路径互不冲突（跨 host 后家园源的 jy_token 在本源读不到，只能靠 sso_token）。
      if (!this.ssoFromUrl()) {
        this.trySsoLogin()
      }
      return
    }
    // 沉浸式: 去掉 body 默认的 5px 外边距, 标题条才能贴满屏幕上方与左右
    document.body.classList.add('ezfy-immersive')
    // ★ 2026-09-29 浏览器标签页标题：二战征途改为游戏名（不再显示默认的「家园社区」）
    // ★★ 2026-10-15 区名写活：跟后端配置走（二战征途-【1区】红色警戒 / 二战征途-【2区】铁血）
    document.title = this.zoneTitle
    // ★ 2026-09-28 司令部子 tab: 刷新后仍是上次选中的 tab（localStorage）
    this.hqTab = this.restoreHqTab()
    // ★ 2026-09-27 iPhone 字体再修复：旧方案用 `@supports (-webkit-touch-callout: none)`
    //   只在桌面(无头 Chrome)测过为 false，但 iOS Safari 的 @supports 解析器并不认识
    //   -webkit-touch-callout 这个属性 → 条件永远不成立 → iPhone 一直回落到宋体-简
    //   (Songti SC)，细灰发虚。改为 JS 判 iOS 加 body.ezfy-ios，再靠 CSS 切苹方/系统字体。
    //   iPadOS 桌面模式 UA 是 Macintosh + 带触摸，用 onTouchEnd 一并圈进来。
    const ua = navigator.userAgent
    if (/iPhone|iPad|iPod/i.test(ua) || (/Macintosh/.test(ua) && 'ontouchend' in document)) {
      document.body.classList.add('ezfy-ios')
    }
    // 沉浸式卡控①: 游戏内任何 <a href="/..."> 都不允许跳出 /games/ezfy 回到家园站点
    // (捕获阶段拦截, 只拦站内绝对路径链接)
    document.addEventListener('click', this.blockEscape, true)
    // ★ 记录最近点击的**元素**（页面内导航、按钮、格子的鼠标落点）：
    //   操作结果的提示要浮现在「刚才点的那个控件」附近，而不是固定页面顶部或原始鼠标坐标。
    //   存元素引用而非 clientX/Y —— API 响应是异步的，回来时页面可能已滚动，
    //   用元素 getBoundingClientRect() 实时取位置不会飘走。
    this._capClick = (e) => {
      // 注意：Vue 的 @click 编译后不会在 DOM 上留下任何属性，因此不能用
      // 属性选择器去找"绑了点击事件的元素"（'[@click]' 还是非法选择器，
      // 会让 closest 直接抛 SyntaxError）。合法做法：a/button 用 closest 命中；
      // 其余情况退回点击目标元素本身（同样有 getBoundingClientRect 可定位）。
      const t = e.target
      let el = null
      if (t && typeof t.closest === 'function') {
        el = t.closest('a,button')
      }
      this._lastClicked = el || (t && t.tagName ? t : null)
    }
    document.addEventListener('click', this._capClick, true)
    // 沉浸式卡控②: 浏览器后退不退出游戏, 而是回到游戏上一页(与幻想西游 Xiyou.vue 一致)
    history.pushState({ __ezfyGuard: true }, '')
    this._onBack = () => {
      if (location.hash.split('?')[0].indexOf('/games/ezfy') >= 0) {
        this.go('home')
      }
      history.pushState({ __ezfyGuard: true }, '')
    }
    window.addEventListener('popstate', this._onBack)
    this.load()
    // ★ 2026-10-23 顶部导航「聊天/邮箱」感叹号：进游戏先拉一次未读数
    this.loadUnread()
    this.loadResCfg()
    // ★ 2026-10-10 兜底：/view 若异常或超时（首次进入、本地无布局缓存）也要让首页渲染出来，
    //   否则首页会一直空白。正常 /view 会先返回并置 homeLayoutReady=true，这里只是保险。
    this._homeLayoutFallback = setTimeout(() => { this.homeLayoutReady = true }, 2500)
    // ★ 刷新后回到刷新前所在的页面（每次刷新都跑首页，不对）
    //   页面状态写在 URL 的 ?cur= 上，onload 时读回来重放 go() 的加载逻辑。
    // ★ 2026-10-03 接口懒加载：先恢复页面，再按当前页加载数据——
    //   /chat、/corps(成员/聊天)、/rank 等页面接口只在进入对应页面时才拉，
    //   不再进游戏就无条件白打一堆接口。
    this.restoreFromUrl()
    // 首页专属数据（首页模板才渲染）：签到状态/世界聊天预览/首页公告/我的军团名。
    // 非首页落地时由 restoreFromUrl → go() 的各分支各自加载。
    if (this.cur === 'home') {
      this.loadWelfare()
      this.loadHomeChats()
      this.loadNotices()
      this.loadCorpsSummary()
    }
    this.timer = setInterval(() => {
      if (this.cur === 'home') { this.load(); this.loadResCfg(); this.loadHomeChats() }
      if (this.cur === 'chat') this.loadChats()
      // ★ 2026-10-23 未读私聊/邮件角标：不分页面，30s 统一刷新一次
      this.loadUnread()
      // ★ 2026-10-05 修复「建筑升级等级不动 / 训练不出厂 / 切城后页面串数据」：
      //   /view 瘦身后这些列表不再由 /view 下发，非首页页面必须自己轮询刷新，
      //   否则建筑完工/训练出厂在页面上永远不更新。
      this.refreshPageData()
    }, 30000)
    // ★ 页脚小Q报时: 每秒刷新(与 App.vue 页脚同一格式)
    this.tickClock()
    this.clockTimer = setInterval(this.tickClock, 1000)
  },
  beforeDestroy () {
    document.body.classList.remove('ezfy-immersive')
    document.body.classList.remove('ezfy-ios')
    document.removeEventListener('click', this.blockEscape, true)
    if (this._capClick) document.removeEventListener('click', this._capClick, true)
    if (this._onBack) window.removeEventListener('popstate', this._onBack)
    if (this.timer) clearInterval(this.timer)
    if (this.clockTimer) clearInterval(this.clockTimer)
    if (this._tipTimer) clearTimeout(this._tipTimer)
    if (this._homeLayoutFallback) clearTimeout(this._homeLayoutFallback)
    this.stopBattleTimer()
  },
  methods: {
    // 装备分组 key：同 cfg 的装备实例视为同一件装备（老数据无 cfg_id 时兜底 name|slot|set_id|tier）
    equipGroupKey (e) {
      if (e && e.cfg_id) return 'c' + e.cfg_id
      return 'n' + (e.name || '') + '|' + (e.slot || e.type || '') + '|' + (e.set_id || 0) + '|' + (e.tier || 0)
    },
    // 页脚小Q报时(与 App.vue tick 同款格式)
    tickClock () {
      // ★ 2026-09-28 让「累计采集/采集资源」实时变化：tickClock 每秒已被 clockTimer 调用，
      //   这里顺手把它升为一个响应式时间基准，模板上的 liveGather() 每秒重算。
      this.gatherNow = Date.now()
      // ★ 安抚冷却倒计时也复用这个每秒基准（避免再起一个定时器）
      this.placateNow = this.gatherNow
      const d = new Date()
      const p = n => (n < 10 ? '0' + n : '' + n)
      this.nowText = d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
      // ★ 2026-09-28 修复「抵达时间归零后卡在 0 秒不动」：
      //   后端是**懒结算**（请求进来才按时间补算），前端那个倒计时只是本地推算，
      //   归零后如果没人去请求接口，状态就永远停在「抵达时间: 0秒」——
      //   看起来像卡死，实际是少了一次「到点了，去拉最新状态」的触发。
      //   这里每秒检查一次：只要**有部队的倒计时归零了**且当前在会走时间的页面上，
      //   就主动重拉一次军队动态（后端顺手把该结算的结算掉）。
      this.checkDueRefresh()
      // ★ 2026-10-16 建筑完工「即时局部刷新」：也是「倒计时归零 → 主动去拉最新状态」
      //   的同一思路，见 checkBuildDue / tickClock 的说明。
      this.checkBuildDue()
    },
    // ★ 倒计时归零 → 自动重拉军队动态（见 tickClock 里的说明）
    //   为什么不用固定间隔轮询：大部分时间没有到点的部队，固定轮询纯浪费请求；
    //   触发式只在「真的有部队到点」时打一次接口，1 核 1G 的线上更友好。
    //   生效页面：军情(军队动态/驻军/军情警讯/战斗报告)、出征队列、战报详情。
    //   节流：最少间隔 3 秒拉一次，避免多个部队同时到点时连环打接口。
    //   ★ 去重：同一订单到点只补拉一次（记在 _dynFired 里）——
    //     否则后端万一结算失败、仍返回 status=0 + 过去的 arrive_time，
    //     就会每 3 秒无限打接口（1 核 1G 上这属于事故级浪费）。
    checkDueRefresh () {
      const onDynPage = this.cur === 'reports' || this.cur === 'reportview' || this.cur === 'orders'
      if (!onDynPage || !this.dynamics || !this.dynamics.length) return
      const now = Date.now()
      // 上一轮刷新还没结束 / 距上次刷新不足 1 秒 → 跳过（★ 2026-10-08 原 3s，收敛减少「返航/抵达」刷新延迟）
      if (this.dynRefreshing) return
      if (this.dynRefreshedAt && now - this.dynRefreshedAt < 1000) return
      if (!this.dynFired) this.dynFired = Object.create(null)
      // 长时间挂机会往 _dynFired 里累积 key（每条订单每一轮倒计时一个），
      // 超过 200 个就整体清空 —— 已完成的订单不会再出现在 dynamics 里，
      // 清掉不会导致「重复触发」，只是给还在跑的订单重新计一次数。
      const firedKeys = Object.keys(this.dynFired)
      if (firedKeys.length > 200) this.dynFired = Object.create(null)
      // 只认「还在走」的倒计时：出征(0)看 arrive_time，返航(2)看 return_time。
      // 战斗中(5)看 battle_left_ms（相对剩余，基准 dynAt —— 与 liveLeft 同口径）：
      //   battleTimer 只在战场指挥室页跑，这里在 orders/军情页兜底，到点拉一次让后端结算回合。
      // 用「订单id+到点时刻」当 key —— 这样同一订单后续新一轮倒计时（重新出征/返航）
      // 能再次触发，而同一轮到点只触发一次。
      let needRes = false
      const due = this.dynamics.some(o => {
        if (!o) return false
        let at = 0
        if (o.status === 0 && o.arrive_time > 0) at = o.arrive_time
        else if (o.status === 2 && o.return_time > 0) at = o.return_time
        else if (o.status === 5 && o.battle_left_ms > 0) at = (this.dynAt || 0) + o.battle_left_ms
        if (!at || at > now) return false
        const key = o.id + '@' + at
        // ★★ 2026-10-08 修复「部队到了、界面没自动刷新」（用户反馈）：
        //   原来「同一订单同一到点时刻只触发一次」—— 万一那一次后端还没结算完
        //   （懒结算 + 请求排队），订单仍是 status=0 + 已经过去的 arrive_time，
        //   而 key 已记 → **之后永远不再触发** → 界面卡在「出征 / 0秒」直到玩家手动刷新。
        //   现在改成**有限重试**：首次立即触发；之后每 2 秒重试一次，同一订单最多 5 次；
        //   仍不变化就放弃 —— 既修了「卡住不刷」，也守住「后端真卡死时不能无限打接口」
        //   这条 1 核 1G 的红线。
        //   ★ 2026-10-08 加固：重试 3 次/3 秒 → **5 次/2 秒**（共约 10 秒窗口）。
        //     实测「并发请求被 per-uid 重入守卫挡掉」时，往往要等下一次请求才结算完，
        //     窗口太短会在极端情况下停在「结算中…」不再自动恢复。
        const tries = this.dynFired[key] || 0
        if (tries >= 5) return false
        if (tries > 0 && now - (this.dynRetryAt || 0) < 2000) return false
        this.dynFired[key] = tries + 1
        // 返航到达 = 待带回资源入库 → 资源栏要刷新；
        // 出征抵达本身不改资源（只是状态变成驻守/进入战斗），不必刷。
        if (o.status === 2) needRes = true
        return true
      })
      if (!due) return
      this.dynRetryAt = now
      this.dynRefreshing = true
      this.dynRefreshedAt = now
      this.loadDynamics()
      // ★ 只有「返航到达」这种真的会改动城市资源的情况才补一次资源刷新。
      //   注意这里**不调整个 load()** —— load() 会连带完整懒结算与军衔等重逻辑，太重；
      //   资源栏只需要 /view 的增量，走下面这个轻量分支。
      if (needRes) this.refreshRes()
      // loadDynamics 是 promise 链，没有返回值可 await；用一个短定时器放开闸门，
      // 保证同一波到点只触发一次（下一个 tick 不会重复打）。（★ 2026-10-08 解锁从 1500ms 收紧到 700ms）
      setTimeout(() => { this.dynRefreshing = false }, 700)
    },
    // ★★ 2026-10-16 建筑完工「即时局部刷新」（用户反馈：已完成建筑不自动刷新）：
    //   建筑「施工中 x秒」倒计时是纯本地推算，归零后 remain() 显示「已完成」，
    //   但等级/状态要等**服务端懒结算**——原来只能靠 30s 全局轮询带一次 /buildings，
    //   于是界面停在「施工中 已完成」最多 30 秒，玩家看着就像没刷新（旧版甚至要手动重进页面）。
    //   这里移植 checkDueRefresh 的「触发式」思想：每秒扫一眼，建筑页上只要有
    //   「已到点还没结算」的建筑就立刻 loadBuildings()（后端 checkBuildingDone 顺手结算），
    //   行马上翻成新等级、队列数同步收回；没有任何到点的建筑时零请求。
    //   节流/去重口径与 checkDueRefresh 一致：最小间隔 1s；同一建筑同一到点最多重试 5 次，
    //   共约 10 秒窗口（懒结算被 per-uid 重入守卫挡掉时常要多等一次请求才结算完）。
    checkBuildDue () {
      if (this.cur !== 'buildm' && this.cur !== 'builds' && this.cur !== 'cityhall' && this.cur !== 'bdetail') return
      if (!this.buildings || !this.buildings.length) return
      const now = Date.now()
      if (this.buildRefreshing) return
      if (this.buildRefreshedAt && now - this.buildRefreshedAt < 1000) return
      if (!this.buildFired) this.buildFired = Object.create(null)
      if (Object.keys(this.buildFired).length > 200) this.buildFired = Object.create(null)
      const doneB = this.buildings.find(b => b && b.status !== 0 && b.end_time > 0 && b.end_time <= now)
      if (!doneB) return
      const key = doneB.id + '@' + doneB.end_time
      const tries = this.buildFired[key] || 0
      if (tries >= 5) return
      if (tries > 0 && now - (this.buildRetryAt || 0) < 2000) return
      this.buildFired[key] = tries + 1
      this.buildRefreshing = true
      this.buildRefreshedAt = now
      this.loadBuildings()
      setTimeout(() => { this.buildRefreshing = false }, 700)
    },
    // ★ 轻量刷新头部资源栏（不触发 loadRank 等重查询）——返航物资入库后调用
    refreshRes () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) {
          this.profile = r.data.profile
          this.city = r.data.city
          this.resProd = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, r.data.res_prod || {})
          this.resMax = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, r.data.res_max || {})
          this.placate = Object.assign({ gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 }, r.data.placate || {})
          this.placateAt = Date.now() // 记住安抚冷却快照的取回时刻，供 placateCdLeft 本地推算
        }
      })
    },
    // ★ 2026-09-28 安抚改为「固定花费 + 15 分钟冷却」后，旧口径不再适用：
    //   原来文案写死「民怨×100」，现在花费/效果/冷却都以后端下发为准（见 View 的 placate 块）。
    //   安抚后立刻重拉 /view 刷新冷却倒计时，避免连点。
    async doPlacate () {
      if (!await this.ask('确定安抚民心吗？（花费 ' + this.placate.gold + ' ' + this.resNames.gold +
        '，民怨 -' + this.placate.grievance + '、民心 +' + this.placate.feelings +
        '；每 ' + this.placate.cooldown_min + ' 分钟可安抚一次）')) return
      api.post('/games/ezfy/city/placate', {}).then(r => {
        this.alert(r, '安抚完成', () => { this.load() })
      })
    },
    // 退出游戏回家园 —— 游戏内唯一的合法出口(底部导航最后的「家园」, 原「首页」)。
    // 用 @click 而不是 <a href>, 这样不会被下面的 blockEscape 拦掉。
    exitToHome () {
      // ★★ 2026-10-18 二战点「家园」= 二战登出（用户：「再二战进家园（底部家园导航）也要作废
      //   二战的 token，家园再进二战就继续生成新的 token」）。
      //   作废 + 清本地后跳回家园；下次从家园进二战由 /api/ezfy/sso 重新签发新 token。
      //   这样二战 token 不会长期残留，换号（退家园 A → 登 B → 进二战）也不会拿 A 的旧 token 继续玩。
      //   ★ 用 fetch + keepalive：整页跳转后请求仍会送达；失败也照常跳，不阻塞玩家回家园。
      this.ezfySignOut()
      // ★★ 2026-10-17 二战独立部署：二战服是独立 SPA，站内没有 /home 路由。
      //   有 home_url（zone 接口下发）→ 整页跳回家园服；没有（home 模式同进程）→ 走站内路由。
      //   ★ 2026-10-18 护栏：home_url 必须是**绝对 http(s) 地址**才整页跳 ——
      //     万一配成 ':8080' 这类相对写法，赋给 location.href 会被浏览器按当前目录拼接，
      //     造成 /:8080/:8080/... 无限刷新；地址栏已出现坏形态时同样只回站内。
      if (this.homeUrl && isAbsoluteHttpUrl(this.homeUrl) &&
          !looksLikeBrokenZonePath(window.location.pathname)) {
        // ★★ 2026-10-18 落点 = 家园「游戏大厅」(/#/games)，不再落在广场首页
        //   （用户：「点家园跳转 家园 /games 游戏页面 别去广场了」）。
        //   home_url 若配置时已带 # 路由则原样尊重；否则统一拼 /#/games。
        window.location.href = this.homeUrl.indexOf('#') >= 0
          ? this.homeUrl
          : this.homeUrl.replace(/\/+$/, '') + '/#/games'
        // ★★ 2026-10-18 兜底：跳转成功的话页面会卸载，这个定时器不会执行；
        //   万一地址无效/被拦（页面还在），把 Vuex 也同步清掉，避免「界面还停在已登录、本地却没 token」。
        setTimeout(() => { this.$store.commit('ezfyLogout') }, 1200)
        return
      }
      this.$store.commit('ezfyLogout')
      // 站内出口（home 模式同进程）同样落在游戏大厅
      this.$router.push('/games')
    },
    // 作废当前二战 token 并清掉本地登录态（点「家园」出口时调用）
    ezfySignOut () {
      const t = this.$store.state.ezfyToken
      if (t) {
        try {
          fetch('/api/ezfy/logout', {
            method: 'POST',
            headers: { Authorization: 'Bearer ' + t },
            keepalive: true
          }).catch(() => {})
        } catch (e) { /* 忽略：不能让登出失败挡住回家园 */ }
      }
      // ★★ 2026-10-18 只清存储、**不 commit**（用户：「二战点击家园也卡顿，导致回不去家园」）：
      //   commit('ezfyLogout') 会把 isLogin 变 false → 整个游戏界面卸载、重渲染成登录框
      //   （游戏 DOM 很大，几百 ms），而页面马上要跳走，这次重渲染纯属浪费，
      //   玩家感知就是「点了没反应 / 卡一下才走」。
      localStorage.removeItem('ezfy_token')
      localStorage.removeItem('ezfy_user')
    },
    // 沉浸式卡控①: 拦掉一切会把玩家带出游戏的站内链接
    // 兼容 '/home' 与 '/#/home' 两种写法; 不动浏览器的前进/后退(那由下面的 popstate 守卫接管)
    blockEscape (e) {
      const a = e.target && e.target.closest ? e.target.closest('a[href]') : null
      if (!a) return
      const href = a.getAttribute('href') || ''
      if (!href || href.charAt(0) !== '/') return          // 相对路径 / javascript: / 外链: 不管
      const raw = href.charAt(1) === '#' ? href.slice(1) : href
      const path = (raw.charAt(0) === '#' ? raw.slice(1) : raw).split('?')[0]
      if (path.indexOf('/games/ezfy') === 0) return         // 游戏内: 放行
      e.preventDefault()
      e.stopPropagation()
      this.notify('游戏内不能跳回家园。如需离开游戏, 请点底部导航「家园」。')
    },
    notOpen (what) {
      this.notify(what + '暂未开放, 敬请期待')
    },
    // ============ 页面状态与 URL 同步（刷新后停在原页面） ============
    //
    // 背景：游戏是单页应用（Ezfy.vue 靠 cur 切换 60+ 个分支），
    //   以前 cur 只存在内存里，一刷新就回落到 data 里的默认值 'home' ——
    //   用户反馈「刷新前在哪个页面刷新后还是哪个页面」。
    //
    // 做法：把 cur（以及资源页的 resType）挂到 URL 的查询串上，
    //   刷新/收藏/分享都能回到同一页。
    //
    // ★ 必须用 history.replaceState 而不是改 location.hash：
    //   本组件在 mounted 里注册了 popstate 守卫（后退回首页），
    //   如果用 pushState / location 赋值，会污染历史栈、把守卫带乱。
    //   replaceState 只改当前条目的 URL，不产生新历史，最安全。

    // syncUrl 把当前页面写进 URL（replaceState，不产生历史记录）
    syncUrl () {
      try {
        const hash = location.hash || ''
        const qi = hash.indexOf('?')
        const base = qi >= 0 ? hash.slice(0, qi) : hash
        const params = new URLSearchParams(qi >= 0 ? hash.slice(qi + 1) : '')
        params.set('cur', this.cur)
        if (this.cur === 'res' && this.resType) params.set('res', this.resType)
        else params.delete('res')
        // ★ 2026-09-25：命令详情页的订单 id 也写进 URL，刷新后才能恢复（原来只在内存里）
        if (this.cur === 'orderview' && this.curOrder && this.curOrder.id) {
          params.set('oid', this.curOrder.id)
        } else params.delete('oid')
        // ★ 2026-09-27 他人统帅信息要查看的玩家 user_id 也写进 URL（同 orderview），
        //   否则刷新后 playerInfo 拿不到 pid 会一直卡「正在加载统帅信息…」。
        if (this.cur === 'playerinfo' && this.playerInfo && this.playerInfo.user_id) {
          params.set('pid', this.playerInfo.user_id)
        } else params.delete('pid')
        // ★ 2026-09-28 修复「在战斗报告页刷新后跳回军队动态」：
        //   reportTab 原来只在内存里，刷新就回落到 data 默认值 1。
        //   reports(列表页) / reportview(战报详情页) 两个页面都把分区 id 挂到 URL 上。
        if ((this.cur === 'reports' || this.cur === 'reportview') && this.reportTab) {
          params.set('rtab', this.reportTab)
        } else params.delete('rtab')
        // ★ 2026-10-16 建筑详情页(bdetail)要看的建筑 id 也写进 URL，否则刷新后 buildInfo 找不到建筑
        if (this.cur === 'bdetail' && this.buildInfoId) params.set('bid', this.buildInfoId)
        else params.delete('bid')
        const qs = params.toString()
        const next = base + (qs ? '?' + qs : '')
        history.replaceState(history.state, '', location.pathname + location.search + next)
      } catch (e) { /* URL 同步失败不影响游戏本身 */ }
    },

    // restoreFromUrl 启动时从 URL 读回页面并重放加载逻辑
    restoreFromUrl () {
      let cur = ''
      let res = ''
      let oid = ''
      let pid = ''
      let rtab = 0
      try {
        const hash = location.hash || ''
        const qi = hash.indexOf('?')
        if (qi >= 0) {
          const params = new URLSearchParams(hash.slice(qi + 1))
          cur = params.get('cur') || ''
          res = params.get('res') || ''
          oid = params.get('oid') || ''
          pid = params.get('pid') || ''
          rtab = parseInt(params.get('rtab') || '0', 10) || 0
        }
      } catch (e) {}
      if (res) this.resType = res
      // ★ 2026-09-28 军情页的分区(军队动态/驻军/军情警讯/战斗报告)也随 URL 恢复，
      //   必须在 go('reports') **之前**赋值：go() 里就是 `switchReportTab(this.reportTab)`。
      if (rtab >= 1 && rtab <= 5) this.reportTab = rtab
      // ★ 2026-09-25 修复「命令详情页刷新后消失」：curOrder 不在 URL 里，
      //   刷新时必须按 oid 重新拉一次详情；拉不到就退回出征队列（绝不留空白页）。
      if (cur === 'orderview') {
        if (parseInt(oid, 10) > 0) this.loadOrderView(oid)
        else this.go('orders')
        return
      }
      // ★ 2026-09-27 修复「他人统帅信息刷新后丢失」：按 pid 重新拉一次；
      //   拿不到 pid 就退回首页（同 orderview，绝不留空白/卡加载页）。
      if (cur === 'playerinfo') {
        const u = parseInt(pid, 10)
        if (u > 0) {
          this.playerInfo = { user_id: u } // 先占位，syncUrl 才会保留 pid
          this.go('playerinfo')
          this.loadPlayerInfo(u)
        } else {
          this.go('home')
        }
        return
      }
      // ★ 2026-09-30 计谋页刷新兜底：schemeOrder 只在内存里，刷新后丢了就回军队动态
      //   （绝不能停在空白的计谋页；信号弹持有量可以重新拉）。
      if (cur === 'scheme' && !this.schemeOrder) {
        this.go('reports')
        return
      }
      // ★ 2026-10-16 建筑详情页(bdetail)刷新兜底：建筑 id 在 URL 的 bid 里，
      //   取不到就退回军事区（绝不停在空白详情页）。
      //   ⚠️ buildInfoId 必须在 go() **之前**赋值：go() 里 `t !== this.cur` 时会把
      //   buildInfoId 清零，只有 `t === 'bdetail'` 才保留 —— 所以顺序不能反。
      if (cur === 'bdetail') {
        const bid = parseInt(params.get('bid') || '0', 10) || 0
        if (bid > 0) {
          this.buildInfoId = bid
          this.go('bdetail')
        } else {
          this.go('buildm')
        }
        return
      }
      // 没写 cur、或就是 home：保持默认首页即可（go('home') 会重复拉一遍数据）
      if (!cur || cur === 'home') return
      // ★ 复用 go()：所有页面分支的加载逻辑都在它里面，
      //   在这里重写一遍必然漏，直接重放最省事也最不容易出错。
      this.go(cur)
    },
    // ---- ★ 2026-10-12 二战独立登录 ----
    // ★★ 2026-10-15 区名写活 + 入口选服：拉本区信息（公开接口，未登录也能拿）。
    //   目前单进程=单区，后端先返回自己；以后多区时这里可扩展为全区列表（登录页下拉切换）。
    loadZone () {
      api.get('/ezfy/zone').then(r => {
        if (r.code === 0) {
          const list = r.data.list || []
          this.zoneList = list
          if (r.data.current) this.zoneSel = r.data.current
          const cur = list.find(z => z.id === this.zoneSel) || list[0]
          if (cur && cur.name) {
            this.zoneTitle = cur.name
            document.title = cur.name
            // ★★ 2026-10-17 缓存本区名：刷新首帧就能显示正确区名，不再先闪兜底文案
            localStorage.setItem('ezfy_zone_title', cur.name)
          }
          // ★★ 2026-10-17 二战独立部署：家园服地址随 zone 接口下发，「家园」按钮整页跳转用
          if (r.data.home_url) this.homeUrl = r.data.home_url
        }
      }).catch(() => {}) // 拿不到就用 data() 里的兜底标题，不影响登录
    },
    // ★★ 2026-10-17 二战独立部署：家园跨服跳转把 jy_token 拼进 hash query（sso_token，不进 HTTP 请求）。
    //   本地无 ezfy_token 且 URL 带 sso_token → 先写进家园 token（让 trySsoLogin 能拿到），
    //   再走现有 SSO 流程换二战 token；成功后由 trySsoLogin 用 history.replaceState 清掉 sso_token。
    //   有 sso_token 时优先走它（跨 host 后 localStorage 按源隔离，家园源的 jy_token 在这读不到）。
    ssoFromUrl () {
      const m = (location.hash || '').match(/[?&]sso_token=([^&]*)/)
      if (!m || !m[1]) return false
      const ssoToken = decodeURIComponent(m[1])
      if (!ssoToken) return false
      // ★★ 2026-10-18 本地已经是**同一个家园号**的二战登录态 → 清掉 URL 里的 token 直接进游戏
      //   （省一次 SSO 往返）。不同号 / 未登录 → 必须以 sso_token 为准重换，
      //   否则「家园换了 B、8081 还带着 A 的 ezfy_token」会直接进 A 的游戏 = 账号串台。
      const ssoUid = String(jwtUid(ssoToken) || '')
      if (this.isLogin && ssoUid && ssoUid === this.ezfyUid) {
        this.stripSsoToken()
        return false
      }
      this.$store.commit('setToken', ssoToken)
      this.trySsoLogin()
      return true
    },
    // 从地址栏清掉一次性的 sso_token（保留 cur=home 等其它 hash query；
    //   hash 路由下 replaceState 只改地址、不触发导航）
    stripSsoToken () {
      const qi = location.hash.indexOf('?')
      if (qi < 0) return
      const hashPath = location.hash.slice(0, qi)
      const params = new URLSearchParams(location.hash.slice(qi + 1))
      if (!params.has('sso_token')) return
      params.delete('sso_token')
      const rest = params.toString()
      history.replaceState(null, '', location.pathname + location.search + (rest ? hashPath + '?' + rest : hashPath))
    },
    // 家园已登录 → 自动换二战登录态（免二次登录）。
    //   家园 token 有效 → 后端 /api/ezfy/sso 直接签发二战 token（不再验密码）；
    //   换成功就硬刷新进游戏（本应用是 hash 路由，与 wwiiLogin 同一套做法）。
    //   失败（家园没登录/已失效）**不提示错误**，保持显示二战登录框，玩家可自己输密码。
    //   ★ /ezfy/sso 不在 /games/ezfy/ 前缀下 → api 拦截器会自动带**家园** token（正是这里要的）。
    trySsoLogin () {
      if (!this.$store.state.token) return // 家园没登录：不打扰，直接显示二战登录框
      // ★★ 2026-10-18 请求序号：手动点「点此进入」可能和自动 SSO 并发，两个请求各自签发 token，
      //   后到的会把 session_at 顶掉 → 先到的 token 失效，玩家可能拿着失效 token 进游戏。
      //   只认最新一次请求的结果，过期响应直接丢弃。
      const seq = ++this.ssoSeq
      this.wwiiBusy = true
      this.ssoPending = true // 占位兜底（data() 已按家园登录态初始化）
      // ★★ 2026-10-18 超时收到 4s（原来吃 axios 全局 15s）：跨机 SSO 超过 4s 基本就是卡了，
      //   再等下去玩家只能干看着。超时 → 走 catch 撤掉占位、露出登录框，玩家可自己输密码进。
      api.post('/ezfy/sso', {}, { timeout: 4000 }).then(r => {
        if (seq !== this.ssoSeq) return // 过期响应（玩家又点了一次），丢弃
        if (r.code === 0 && r.data && r.data.token) {
          this.$store.commit('setEzfyToken', r.data.token)
          this.$store.commit('setEzfyUser', { id: r.data.game_id, nickname: r.data.nickname || '' })
          // ★ 2026-10-17 二战独立部署：sso_token 是一次性的，换到 ezfy_token 后从地址栏清掉
          //   （保留 cur=home 等其它 hash query；hash 路由下 replaceState 只改地址、不触发导航）。
          const qi = location.hash.indexOf('?')
          if (qi >= 0) {
            const hashPath = location.hash.slice(0, qi)
            const params = new URLSearchParams(location.hash.slice(qi + 1))
            params.delete('sso_token')
            const rest = params.toString()
            history.replaceState(null, '', location.pathname + location.search + (rest ? hashPath + '?' + rest : hashPath))
          }
          window.location.reload()
          return
        }
        // SSO 失败（二战未建档等）→ 撤掉占位，露出二战登录框让玩家自己输密码
        this.ssoPending = false
        this.wwiiBusy = false
      }).catch(() => {
        if (seq !== this.ssoSeq) return // 过期响应（超时后又重试了），别把新请求的占位撤掉
        this.ssoPending = false
        this.wwiiBusy = false
      })
    },
    // ★★ 2026-10-18 占位页「点此进入」：常显，玩家随时可点。
    //   已有二战 token（SSO 其实成功了、只是没跳走）→ 直接刷新进游戏；
    //   没有家园 token（SSO 无从谈起）→ 刷新一次露出登录框，绝不再自动重试（避免死循环刷新）；
    //   其余情况重走一次 SSO（序号机制会自动作废上一次未完成的请求）。
    manualEnter () {
      if (this.$store.getters.isEzfyLogin) { window.location.reload(); return }
      if (!this.$store.state.token) { window.location.reload(); return }
      this.trySsoLogin()
    },
    // 输入家园号(=二战游戏ID) + 家园密码，走二战自己的登录接口 /api/ezfy/login：
    //   首登用「家园号+家园密码」校验并同步建档，签发二战专属 token；
    //   二战登录态只存 ezfy_token，与家园账号体系彻底分离。
    wwiiLogin () {
      if (this.wwiiBusy) return
      // ★★ 2026-10-18 家园已登录时强制用当前家园号（输入框虽已只读，这里再兜一层防篡改）
      const name = (this.homeUid || this.wwiiName || '').trim()
      const pwd = this.wwiiPwd || ''
      if (!name || !pwd) {
        this.wwiiErr = '请输入家园号和密码'
        return
      }
      this.wwiiErr = ''
      this.wwiiBusy = true
      const body = { name, password: pwd }
      if (this.wwiiNeedCaptcha) {
        body.captcha_id = this.wwiiCaptchaId
        body.captcha = this.wwiiCaptcha
      }
      api.post('/ezfy/login', body).then(r => {
        this.wwiiBusy = false
        if (r.code === 0 && r.data && r.data.token) {
          // 二战专属 token 与身份（与家园 token 分开存，各自过期）
          this.$store.commit('setEzfyToken', r.data.token)
          this.$store.commit('setEzfyUser', { id: r.data.game_id, nickname: r.data.nickname || '' })
          // ★ 2026-10-12 家园号+家园密码登二战，也让家园进入已登录态：后端同秘密签发 home_token，
          //   这里写进家园 jy_token，点【家园】不再要求重新登录。
          if (r.data.home_token) {
            const uname = r.data.nickname || ''
            this.$store.commit('setUser', { token: r.data.home_token, user: { id: r.data.game_id, name: uname, nickname: uname } })
          }
          // 硬刷新进游戏：本应用是 hash 路由（/#/games/ezfy?cur=home）——
          // 必须带 `#` 跳，否则走 `location.href='/games/ezfy'` 会落到服务器根路径、
          // 被 vue-router 当作 `/`（家园首页）→ 未家园登录就弹去 /#/login。
          window.location.hash = '#/games/ezfy?cur=home'
          window.location.reload()
        } else {
          this.wwiiErr = r.msg || '登录失败，请重试'
          if (r.data && r.data.need_captcha) {
            // 密码错误满 3 次，立即弹出验证码框
            this.wwiiNeedCaptcha = true
            this.loadWwiiCaptcha()
          } else if (this.wwiiNeedCaptcha) {
            // 验证码一次性，失败后换一张
            this.loadWwiiCaptcha()
          }
        }
      }).catch(() => {
        this.wwiiBusy = false
        this.wwiiErr = '网络异常，请重试'
      })
    },
    // 拉一张新的算式验证码（点图片换一张）
    loadWwiiCaptcha () {
      this.wwiiCaptcha = ''
      api.get('/auth/captcha').then(r => {
        if (r.code === 0) {
          this.wwiiCaptchaImg = r.data.img
          this.wwiiCaptchaId = r.data.id
        }
      })
    },
    // ---- 统帅页自助 ----
    loadSelfInfo () {
      return api.get('/games/ezfy/profile/self').then(r => {
        if (r.code === 0) {
          this.selfInfo = r.data
          this.renameInput = r.data.nickname || ''
        }
      }).catch(() => {})
    },
    // ★ 2026-10-12 访问记录列表
    loadVisits () {
      api.get('/games/ezfy/visits').then(r => {
        if (r.code === 0) this.visitRecords = r.data.list || []
      }).catch(() => {})
    },
    startRename () {
      this.renameInput = this.selfInfo.nickname || this.profile.nickname || ''
      this.renameEditing = true
    },
    // ★ 2026-10-11 昵称不能含 emoji（与后端 ezfyHasEmoji 同口径；前端先给即时提示，不用等接口）
    //   覆盖 Unicode 表情区块：1F000–1FAFF / 2600–27BF / 2B00–2BFF / FE00–FE0F / 200D / 20E3
    //   + ©®‼⁉™ℹ〰〽㊗㊙（会被当 emoji 渲染）；**放行** ★☆（常用装饰符号）、← → ↑ ↓（常用箭头）。
    hasEmoji (s) {
      const str = String(s || '')
      for (let i = 0; i < str.length; i++) {
        const c = str.codePointAt(i)
        if (c > 0xFFFF) i++ // 代理对：跳过低位，避免重复判断
        if ((c >= 0x1F000 && c <= 0x1FAFF) ||
            (c >= 0x2600 && c <= 0x27BF && c !== 0x2605 && c !== 0x2606) ||
            (c >= 0x2B00 && c <= 0x2BFF) ||
            (c >= 0xFE00 && c <= 0xFE0F) ||
            c === 0x200D || c === 0x20E3 ||
            c === 0x00A9 || c === 0x00AE || c === 0x203C || c === 0x2049 ||
            c === 0x2122 || c === 0x2139 || c === 0x3030 || c === 0x303D ||
            c === 0x3297 || c === 0x3299) return true
      }
      return false
    },
    async doPlayerRename () {
      const name = String(this.renameInput || '').trim()
      if (name.length < 2 || name.length > 12) { this.notify('昵称长度需在 2~12 个字符之间'); return }
      // ★ 2026-10-11 「[修改昵称] 不能用表情」
      if (this.hasEmoji(name)) { this.notify('昵称不能包含表情符号'); return }
      const free = this.selfInfo.rename_free
      const tip = free ? '确认改名为「' + name + '」吗？（首次免费）'
        : '确认改名为「' + name + '」吗？将消耗「改名卡」×1（当前 ' +
          (this.selfInfo.rename_card_count || 0) + ' 张）'
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/profile/rename', { nickname: name }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.renameEditing = false
          this.loadSelfInfo()
          this.load()
        } else this.notify(r.msg)
      })
    },
    async doChangeCamp (camp) {
      const cur = this.selfInfo.camp || this.profile.camp
      if (cur === camp) { this.notify('当前已经是「' + (camp === 2 ? '轴心国' : '同盟国') + '」'); return }
      const free = this.selfInfo.camp_free
      const tip = free ? '确认转换为「' + (camp === 2 ? '轴心国' : '同盟国') + '」吗？（首次免费）'
        : '确认转换为「' + (camp === 2 ? '轴心国' : '同盟国') + '」吗？将消耗「阵营转换道具」×1（当前 ' +
          (this.selfInfo.camp_item_count || 0) + ' 个）'
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/profile/camp', { camp: camp }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadSelfInfo()
          this.load()
        } else this.notify(r.msg)
      })
    },
    // ---- 首页布局切换（老布局/新布局，统帅页单选，即改即生效） ----
    // ★ 2026-10-10 防「刷新时新布局掠影」：把上次用的布局存 localStorage（key 带用户），
    //   刷新首帧就用它渲染。只要本地没有缓存（如清缓存/换账号首登），才退回 /view 下发值。
    cachedHomeLayout () {
      try {
        const uid = ((this.$store.state.ezfyUser && this.$store.state.ezfyUser.id) ||
          (this.$store.state.user && this.$store.state.user.id)) || 'x'
        const k = 'ezfy_home_layout_' + uid
        return localStorage.getItem(k) === '2' ? 2 : 0
      } catch (e) { return 0 }
    },
    // ★ 2026-10-10 本地是否已有布局缓存（决定首帧能不能直接渲染首页）。
    //   没有 → 说明是首次进入/清过缓存，先不渲染首页，等 /view 权威布局，避免「新布局掠影」。
    hasCachedHomeLayout () {
      try {
        const uid = ((this.$store.state.ezfyUser && this.$store.state.ezfyUser.id) ||
          (this.$store.state.user && this.$store.state.user.id)) || 'x'
        return localStorage.getItem('ezfy_home_layout_' + uid) !== null
      } catch (e) { return false }
    },
    saveHomeLayout (v) {
      try {
        const uid = ((this.$store.state.ezfyUser && this.$store.state.ezfyUser.id) ||
          (this.$store.state.user && this.$store.state.user.id)) || 'x'
        localStorage.setItem('ezfy_home_layout_' + uid, String(v === 2 ? 2 : 1))
      } catch (e) {}
    },
    setHomeLayout (l) {
      const v = l === 2 ? 2 : 1
      this.saveHomeLayout(v)
      api.post('/games/ezfy/profile/layout', { layout: v }).then(r => {
        if (r.code === 0) {
          this.homeLayout = v
          this.loadSelfInfo()
          this.notify(r.msg)
        } else this.notify(r.msg)
      })
    },
    go (t) {
      // ★ 2026-09-29 各页 [返回]：回到上一页；无有效上一页则回首页
      if (t === 'back') { this.go(this.prevCur && this.prevCur !== this.cur ? this.prevCur : 'home'); return }
      // ★ 2026-10-25 「这次是从战报详情返回列表」：要保住原来的分页位置，
      //   所以必须在下面 `this.cur = t` 之前记下来（之后再判断 this.cur 已经是新页了）。
      const fromReportView = this.cur === 'reportview'
      // ★ 2026-10-04 修复「资源详情页刷新后丢了」：刷新时 URL 恢复的是 cur='res' + res=xxx，
      //   原分支只认 'res/xxx'，cur='res' 落进 default 什么都不加载 → 页面空白。
      if (t === 'res' || t.indexOf('res/') === 0) {
        if (t.indexOf('res/') === 0) this.resType = t.slice(4)
        this.cur = 'res'
        this.syncUrl()
        this.loadRes()
        return
      }
      // ★ 离开战场页就停掉倒计时轮询，避免在后台一直打接口
      if (t !== 'battle') this.stopBattleTimer()
      // ★ 2026-09-29 记录上一页，供各页 [返回]（goBack）回到上一页
      // ★ 2026-10-16 换页时清掉「建筑详情」要看的建筑 id（它只属于 bdetail 页）；
      //   去 bdetail 自己时保留（openBuildInfo 先设 id 再 go）。
      if (t !== this.cur) { this.prevCur = this.cur; if (t !== 'bdetail') this.buildInfoId = 0 }
      this.cur = t
      this.syncUrl()
      if (t === 'home') {
        this.load()
        this.loadWelfare()
        this.loadHomeChats()
        // ★ 2026-10-03 接口懒加载：首页数据改为进首页时拉齐
        //   （公告外露、我的军团名；原来靠 mounted 无条件预拉，切回首页会空）
        this.loadNotices()
        this.loadCorpsSummary()
      }
      else if (t === 'troops' || t === 'troop' || t === 'defence' ||
               t === 'troopview' || t === 'trainpre' || t === 'troopstat') {
        this.loadTroops()
        // ★ 2026-10-04 /view 瘦身：军队页顶部「军工厂合计N级」由 buildings 算，
        //   建筑数据已移到 /buildings，进军队相关页面一并懒加载
        this.loadBuildings()
        // ★ 训练页(troop)的队列行要按背包里的训练加速道具档位渲染 [加速] 按钮
        if (t === 'troop' || t === 'troops') this.loadBag()
      }
      // ★ 2026-10-23 召集页/城市状态页显示人口、待恢复、临时空闲：进页先拉一次 /view，
      //   避免用首页或其他页留下的旧快照（临时空闲 5 分钟内会回落，旧值会误导玩家）。
      else if (t === 'convene') {
        this.load()
      }
      // ★★ 2026-10-25 修复城市状态页「建筑: 0座 (0/66) / 军队: 0 / 附属野地: 0/level」：
      //   原来这里和下面那条 `else if (t === 'citystatus')` **各写了一遍**，而 if/else 链
      //   只认第一个命中的分支 → 下面那条是**死代码**，进本页只跑了 /view；
      //   偏偏 /view 瘦身后不再下发 buildings/queues/wildlands（见 2026-10-04 瘦身注释），
      //   于是这些计数恒为 0（从首页直接点[城市状态]时最明显）。
      //   现在 citystatus 在**这一条**里把 4 个接口一次拉齐；convene 只关心人口，仍只拉 /view。
      else if (t === 'citystatus') {
        this.load()
        this.loadBuildings()
        this.loadTroops()
        this.loadWilds()
        // ★ 2026-10-25 市长/城守：/view 不下发职位 → 单独拉一次 /officers（后端有页面级缓存）
        this.loadOfficerPosts()
      }
      else if (t === 'hq') {
        // ★ 2026-09-28 预设编队 tab 需要：本城坐标(/view)、集结令(gatherMax)、军官(onDutyOfficers)、背包(集结令数量)
        this.load()
        this.loadTroops().then(() => this.loadTargets())
        this.loadOrders()
        this.loadPresets()
        this.loadOnDutyOfficers()
        this.loadBag()
        // 进来就先算一次：让「油耗/负重/本次出兵上限」立刻是准确值（若正处在预设编队 tab）
        this.$nextTick(() => { if (this.hqTab === 4) this.presetCalc() })
      }
      // ★ 2026-09-26 修复「加速道具买完实际使用不生效」：科技/建筑/训练页的 [加速]
      //   现在按「背包里实际拥有的加速道具档位」渲染按钮，所以进页时要拿到背包数据。
      else if (t === 'techs') { this.loadTechs(); this.loadBag() }
      // ★ 2026-10-04 /view 瘦身：建筑列表/可建造池移到 /buildings，进入建筑相关页懒加载
      else if (t === 'buildm' || t === 'builds' || t === 'cityhall' || t === 'bdetail') { this.loadBuildings(); this.loadBag() }
      // ★ 2026-10-04 /view 瘦身：地图页「占领野地」面板数据移到 /city/wildfull 懒加载
      else if (t === 'map') { this.backToMap(); this.loadStars(); this.loadWilds() }
      // ★ 2026-09-28 军情页：分区由 reportTab 决定，且 switchReportTab 内部会 syncUrl，
      //   刷新后 reportTab 已由 restoreFromUrl 从 ?rtab= 恢复，所以不会跳回「军队动态」。
      // ★ 2026-10-25 从战报详情 [返回]：**保住原分页**（用户：在第 3 页点进去看，回来又跑回第 1 页）
      else if (t === 'reports') { this.curReport = null; this.switchReportTab(this.reportTab, fromReportView) }
      // ★ 2026-09-30 计谋页（行军计谋专用）：进页/刷新都重拉信号弹持有量
      else if (t === 'scheme') this.loadSchemes()
      else if (t === 'mail') { this.loadMail(this.mailTab, true) }
      // ★ 2026-10-13 好友页：进入即拉会话列表 + 好友列表（微信式会话列表优先展示）
      else if (t === 'friends') { this.friendView = 'convs'; this.loadPmConvs(); this.loadFriends() }
      else if (t === 'liaison') this.loadLiaison()
      else if (t === 'tasks') this.loadTasks()
      else if (t === 'welfare') this.loadWelfare()
      // ★ 2026-10-03 接口懒加载：聊天数据只在进入聊天页时拉（原 mounted 无条件拉）
      else if (t === 'chat') this.loadChats()
      else if (t === 'rank') { this.loadRank(); if (this.rankTab === 'pvp') this.loadPvpRank() }
      // 城市列表页要显示「军衔可建城数」，所以也拉一次军衔数据
      // ★ 2026-09-28 兜底：cities 只在 /view 里下发，万一初始化那次请求失败（或列表为空）
      //   → 进页时补拉一次 /view，别让玩家看到一个空列表还不知道为什么。
      else if (t === 'cities') { this.loadRank(); if (!this.cities.length) this.load() }
      else if (t === 'bag') { this.loadBag(); if (this.bagTab === 'chest') this.loadChestBag() }
      else if (t === 'treasure') this.loadBag()   // ★ 宝物页：展示采集宝物（bagTreasures）
      else if (t === 'mall') {
        this.loadMall(true)
        if (this.mallTab === 'equipment') this.loadEquipShop()
        if (this.mallTab === 'chest') { this.loadChests(); this.loadChestBag() }
      }
      // ★ 2026-10-09 宝箱内容页：数据还没拉到就拉一次（刷新/直接带 URL 进来时 chestData 是空的）
      else if (t === 'chestview') {
        if (!this.chestData.chests.length) this.loadChests()
      }
      // ★ 2026-10-14 我的待开宝箱页：进页即拉
      else if (t === 'chestbag') this.loadChestBag()
      else if (t === 'exchange') this.loadExchange()
      else if (t === 'corps') {
        // ★ 2026-09-25 进入军团页重置子栏缓存 → 再次进入时拉到最新数据；
        //   单纯切 tab 不会重复拉接口（见 switchCorpsTab）。
        this.corpsRelations = null
        this.corpsWars = null
        this.corpsMall = { loaded: false, my_points: 0, corps_points: 0, items: [], my_bought: {} }
        this.loadCorps()
        this.switchCorpsTab(this.corpsTab)
      }
      else if (t === 'orders') {
        // ★ 2026-09-25 「出征队列按照军队动态那种展示」→ 数据源与军队动态统一：
        //   每次进页都重新拉一遍（刷新页面后也是走这里），队列不会再「刷新就消失」。
        this.loadDynamics()
      }
      else if (t === 'battle') {
        // ★ 战场页刷新后不能只靠 URL 恢复：订单 id 只存在内存里，刷新就丢了。
        //   所以带 id 就直接拉，没带就从后端找回当前进行中的那场战斗。
        if (this.battleOrderId) this.loadBattle()
        else this.resumeBattle()
      }
      else if (t === 'notices') this.loadNotices()
      else if (t === 'wilds') this.loadWilds()
      else if (t === 'orderpre') {
        // ★ 必须一起加载背包：出征页的「集结令」要读 bagItems，
        //   只进背包页才 loadBag 的话，出征页会永远显示「0个」且输入框被禁用。
        //
        // ★ 用户反馈「出征还有上次留的数据」→ 每次进出征页都把上次填的东西清干净
        //   （兵力/军官/携带资源/宿营/集结令），否则上一次的部队数量会残留，
        //   很容易误发一支自己没打算派的队伍。
        this.resetOrderForm()
        // ★ 2026-09-28 预设编队下拉：每次进出征页刷新预设列表（别人/别城新增的预设也能及时出现）
        this.loadPresets()
        // ★ 出征页顶部要显示「出发城市」，随军资源的「城内现有」也取自 /view，
        //   所以这里必须连 /view 一起拉 —— 否则切换城市后出征页仍显示上一座城的名字/资源。
        // ★★ 2026-10-13 性能（用户反馈「点[出征]卡顿」）：
        //   ① `/view` 是最重的接口（十几条查询 + 懒结算），但出征页只需要它里面的
        //      「出发城市名/坐标 + 城内现有资源」→ 让到**下一帧之后**再发，页面骨架先出来、别拖首屏；
        //   ② `loadCityOfficers` 已挪进 `loadOnDutyOfficers` 按需触发（只在真的没有可带队军官时），
        //      这里不再无条件发 —— 并发请求从 6 个降到 5 个（浏览器同域上限 6，正好避开排队）。
        setTimeout(() => this.load(), 0)
        this.loadTroops()
        this.loadOnDutyOfficers()
        this.loadBag()
        // 进来就先算一次：让「本次出兵 0 / 上限 N」和集结令上限立刻是准确值
        // （原来要玩家自己点[计算]才会显示）
        this.$nextTick(() => this.scheduleOrderCalc())
      }
      // ★ 2026-10-04 /view 瘦身：军官页顶部「占领野地 N 块」移到 /city/wildfull 懒加载
      else if (t === 'acade') { this.loadAcade(); this.loadWilds() }
      else if (t === 'wareset') this.loadWare()
      else if (t === 'citymove') this.loadMoveInfo()
      else if (t === 'sourceset') this.loadProduce()
      else if (t === 'activity') this.loadActivity()
      else if (t === 'info') this.loadSelfInfo()
      else if (t === 'visits') this.loadVisits()
      else if (t === 'factory') { this.loadTroops(); this.loadBuildings() }
      // ★★ 2026-10-25 原来这里还有一条 `else if (t === 'citystatus') {...}`，
      //   但 if/else 链上面（convene 那一段）已经命中 citystatus → **永远是死代码**，
      //   造成「建筑: 0座 (0/66)」。已合并到上面那一条，这里删掉，别再写第二份。
    },
    // 节日活动
    loadActivity () {
      api.get('/games/ezfy/activity').then(r => {
        if (r.code === 0) this.activities = r.data.activities || []
      })
    },
    fmtLeft (sec) {
      if (!sec || sec <= 0) return '已结束'
      const d = Math.floor(sec / 86400)
      const hh = Math.floor((sec % 86400) / 3600)
      const mm = Math.floor((sec % 3600) / 60)
      if (d > 0) return d + '天' + hh + '小时'
      if (hh > 0) return hh + '小时' + mm + '分'
      return mm + '分'
    },
    // ★ 2026-10-04 拆出 applyView：/view 与切城接口返回同一份数据结构，
    //   切城时直接 applyView(r.data) 完成全量刷新，不再二次请求 /view。
    load () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) this.applyView(r.data)
      })
    },
    applyView (d) {
      if (!d) return
      this.profile = d.profile
      // ★ 2026-10-10 首页布局：0/1=新布局 2=老布局（后端 /view 随 profile 下发）
      const hl = d.profile && d.profile.home_layout
      this.homeLayout = (hl === undefined || hl === null) ? 0 : Number(hl)
      // ★ 2026-10-10 后端下发的是权威布局：顺手缓存到 localStorage，
      //   这样纯 seed 固定（如玩家 10000 固定老布局、从不手动切换）的账号也防「刷新新布局掠影」。
      if (hl !== undefined && hl !== null) this.saveHomeLayout(Number(hl))
      // ★ 2026-10-10 权威布局已到 → 放行首页渲染（首次进入不再先画一帧新布局）。
      this.homeLayoutReady = true
      this.userBrief = { account: d.account || '', level: d.user_level || 0, exp: d.user_exp || 0 }
      this.officerCount = d.officer_count || 0
      this.rankName = d.rank_name
      this.rankPost = d.rank_post
      // ★ 2026-10-05 修复「切城后页面串数据（感觉所有城市都一样）」：
      //   城市真的变了（切城/弃城）就把按城缓存的页面数据清掉，避免渲染出上一座城的
      //   建筑/军队/野地；页面重新进入时 go() 会各自重新拉取。首页轮询时城市没变不清。
      const cityChanged = !!d.city && (!this.city || d.city.id !== this.city.id)
      this.city = d.city
      // ★ 2026-10-11 修复「首帧 city 未就绪 → 战报按 city_id=0 拉取显示暂无」：
      //   首次进入军情页时 /view 还没回来，restoreFromUrl→go('reports') 会用 city=undefined
      //   发请求并写进 '4|0|' 缓存。city 就绪后这里补一次强制刷新，覆盖成当前城的真实战报。
      if ((this.cur === 'reports' || this.cur === 'reportview') &&
          (this.reportTab === 3 || this.reportTab === 4 || this.reportTab === 5) &&
          !this.repCache[this.repCacheKey(this.reportTab)]) {
        this.loadReports()
      }
      // ★ 2026-09-28：头部资源栏「/」右侧展示每小时产量
      this.resProd = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, d.res_prod || {})
      this.resMax = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, d.res_max || {})
      // ★★ 2026-09-28 修复「切换城市 → 城市列表空了」：
      //   1bafa1a（格式、民心民怨）新增安抚参数时，把原本这一行 `this.cities = d.cities`
      //   覆盖删掉了。而 cities 在 data 里初值就是 []，**全文件再无第二处赋值** ——
      //   于是城市列表恒为空（「已有 0 座」、建城页也一直显示 0）。
      //   后端 /view 一直在下发 cities（ezfy.go 的 View → h.cityViews），这里接住即可。
      //   ⚠️ 改 load() 时别再把这一行弄丢：它是 cities 的唯一数据源。
      this.cities = d.cities || []
      // ★ 2026-10-07 赎回金额（管理端可配，默认 500）：城市列表 [赎回] 确认弹窗展示
      this.ransomCost = d.ransom_cost || 500
      this.placate = Object.assign({ gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 }, d.placate || {})
      this.placateAt = Date.now() // 安抚冷却快照时刻（见 placateCdLeft）
      this.continent = d.continent
      this.cityKindRaw = d.city_kind || ''
      this.cityIsSea = !!d.is_sea
      this.protectedUntil = d.protected
      // ★ 2026-10-15 免战到期时间戳（城市状态页展示时效；老后端不下发 → 0，退化成只显示「[免战保护中]」）
      this.protectedUntilMs = d.protected_until || 0
      this.boostUntil = d.boost
      // ★ 2026-10-04 修复「一键加速后建筑没了」：/view 已不再下发 buildings，
      //   这里若无条件赋空数组，会把页面已懒加载好的建筑列表整个清空。
      //   改为「下发才覆盖、不下发保留」，建筑列表只由 /buildings 管理。
      // ★ 2026-10-05 城市真变了 → 清掉按城数据（上面 cityChanged），防止串城显示。
      if (cityChanged) {
        this.buildings = []
        this.buildingPool = []
        this.wildlands = []
        this.queues = []
      }
      if (d.buildings !== undefined) this.buildings = d.buildings || []
      if (d.building_pool !== undefined) this.buildingPool = d.building_pool || []
      this.militaryCap = d.military_cap || 33
      this.resourceCap = d.resource_cap || 33
      // ★ 2026-09-26 召集消耗/收益（后端保证 >= 1，兜底默认 10 万）
      //   ★ 2026-10-22 收益改为「当前民居上限的 N%」，默认 10%；
      //   人口上限只由民居 pop_max 决定，不再有额外开关/硬上限。
      this.conveneFoodCost = Number(d.convene_food_cost) || 100000
      this.convenePopPct = Math.min(1000000000, Math.max(1, Number(d.convene_pop_pct) || 10))
      // ★ 2026-10-23 召集间隔（秒）：普通 / 人口满分别读配置，默认 3 / 300。
      this.conveneCooldownSec = Math.min(1000000000, Math.max(1, Number(d.convene_cooldown_seconds) || 3))
      this.conveneCapCooldownSec = Math.min(1000000000, Math.max(1, Number(d.convene_cap_cooldown_seconds) || 300))
      // ★ 2026-10-17 训练一键加速开关（1 开 / 0 关，默认开）：训练页据此展示/隐藏两个一键加速按钮
      this.speedTrainOn = d.speed_train_on === undefined || d.speed_train_on === null
        ? true : !!Number(d.speed_train_on)
      // ★ 2026-10-04 同上：/view 不再下发 wildlands/queues，下发才覆盖、不下发保留
      if (d.wildlands !== undefined) this.wildlands = d.wildlands || []
      if (d.queues !== undefined) this.queues = d.queues || []
      this.marching = d.marching
      this.occupying = d.occupying
      this.unreadReports = d.unread_reports
      // ★ 2026-10-13 顶部导航「军情」感叹号角标：未看的 军情警讯 + 被侦查/被掠夺/被征服战报
      //   （后端 unread_warns，口径见 `ezfyReportAlertCond`）
      this.unreadWarns = d.unread_warns || 0
      // ★★ 后端旧版本不下发 unread_warns 时兜底问一次 counts —— 否则角标只有「点进军情页」
      //   才会被 loadReportCounts 点亮（用户反馈「满足条件不提示，必需点军情才有，这没意义」）。
      if (d.unread_warns === undefined) this.loadReportCounts()
      // ★ 占用人口随 /view 一起下发：首页/城市状态页的「空闲人口」不再依赖
      //   「有没有进过军队页」（原来没进过就按 0 算，空闲人口显示成满人口）
      this.popUsed = d.pop_used || 0
      // ★ 待恢复人口（训练完的人口慢慢恢复的那部分，见 freePop）
      this.popRecover = d.pop_recover || 0
      // ★ 额外空闲人口（人口满后召集产生，允许空闲暂时超过总人口，5 分钟内回落）
      this.popExtra = d.pop_extra || 0
      this.cityPop = d.city.pop || 0
      this.taxInput = d.city.tax_rate
      this.applyResNames(d.res_names)
      // ★ 集结令配置（管理端可配，默认 99）：跟着 /view 一起下发，
      //   这样一进页面（还没点[计算]）输入框的上限就是对的。
      this.gatherCfg = {
        max: d.gather_max > 0 ? d.gather_max : 0,
        per: d.gather_per > 0 ? d.gather_per : 100000,
        have: d.gather_have || 0
      }
      // ★ 首页要显示「每日签到：已签到/签到」，但 /view 不下发 welfare。
      //   不补这一下，签到完回首页仍显示「签到」——用户反馈的 bug。
      if (this.cur === 'home') this.loadWelfare()
    },
    // 资源显示名：把后端下发/读取到的名字合并进兜底值
    applyResNames (d) {
      if (!d || typeof d !== 'object') return
      const n = Object.assign({}, RES_NAMES)
      const sh = Object.assign({}, RES_SHORT)
      Object.keys(RES_NAMES).forEach(k => { if (d[k]) n[k] = d[k] })
      const sd = d._short || {}
      Object.keys(RES_SHORT).forEach(k => { if (sd[k]) sh[k] = sd[k] })
      this.resNames = n
      this.resShort = sh
      this.resDes = buildResDes(n)
    },
    // 单独拉一次（有些页面不经过 /view）
    loadResCfg () {
      return api.get('/games/ezfy/res-cfg').then(r => {
        if (r.code === 0) this.applyResNames(r.data)
      }).catch(() => {})
    },
    loadTroops () {
      return api.get('/games/ezfy/troops').then(r => {
        if (r.code === 0) {
          this.troopsData = r.data
          // ★ 2026-10-06 列表刷新后清掉行内「恢复数量」输入（旧记录的 id 可能复用，避免残留数量误填）
          this.woundNums = {}
          // ★ 2026-10-04 /view 瘦身：训练队列列表随 /troops 下发（不再由 /view 提供）。
          //   这里同步喂给 this.queues，军队页/城市状态页模板继续读 ezfy.queues 无需改动。
          this.queues = r.data.queues || []
          // 占用人口（训练中；已训练完成的部队不占人口）
          this.popUsed = r.data.pop_used || 0
          // ★ 待恢复人口（训练完的人口慢慢恢复的那部分，见 freePop）
          this.popRecover = r.data.pop_recover || 0
          // ★ 额外空闲人口（人口满后召集产生，允许空闲暂时超过总人口）
          this.popExtra = r.data.pop_extra || 0
          this.cityPop = r.data.pop || 0
          // ★ 2026-10-14 伤兵营钻石一键恢复全部伤兵单价（系统配置下发，默认100）
          this.healAllDiamond = r.data.heal_all_diamond || 100
          if (r.data.cfgs.length && !this.trainSel) this.trainSel = null
          // 司令部配置表按兵种建键, 兵种数据后到时要补齐, 否则渲染会取到 undefined
          this.ensureTargetCfg()
        }
      })
    },
    // ★ 2026-10-04 /view 瘦身：建筑列表/可建造池移到 /buildings（进入建筑/军队等页面时懒加载）。
    //   数据喂给 this.buildings / this.buildingPool，模板与 computed 无需改动。
    // ★ 2026-10-05 30s 轮询按当前页刷新对应数据（建筑完工/训练出厂/科技完成/野地状态实时可见；
    //   切城后停留在旧页时也会在 30s 内拉回新城数据，消除「所有城市都一样」的错觉）。
    refreshPageData () {
      const t = this.cur
      if (t === 'buildm' || t === 'builds' || t === 'cityhall' || t === 'buildpre' || t === 'bdetail') {
        this.loadBuildings()
      } else if (t === 'troops' || t === 'troop' || t === 'defence' || t === 'factory' ||
                 t === 'troopview' || t === 'trainpre' || t === 'troopstat') {
        this.loadTroops()
        this.loadBuildings()
      } else if (t === 'techs' || t === 'techpre') {
        this.loadTechs()
      } else if (t === 'wilds') {
        this.loadWilds()
      } else if (t === 'orders' || t === 'reports' || t === 'reportview') {
        // ★ 2026-10-16 军情/出征队列/军队动态：30s 轮询兜底拉一次军队动态，
        //   覆盖「没有倒计时可听（如等待驻军/情报列表）」时长期不刷新的情况。
        //   触发式 checkDueRefresh 负责到点即时刷新，这里是慢速兜底，两者互不冲突。
        this.loadDynamics(true)
      }
    },
    loadBuildings () {
      return api.get('/games/ezfy/buildings').then(r => {
        if (r.code === 0) {
          this.buildings = r.data.buildings || []
          this.buildingPool = r.data.pool || []
          if (r.data.military_cap) this.militaryCap = r.data.military_cap
          if (r.data.resource_cap) this.resourceCap = r.data.resource_cap
          if (r.data.build_queue_base > 0) this.buildQueueBase = r.data.build_queue_base
          this.buildQueueBonus = r.data.build_queue_bonus || 0
          this.buildQueueBonusUntil = r.data.build_queue_bonus_until || 0
        }
      })
    },
    // 保证每个兵种在 targetCfg 里都有条目(司令部页 v-model 直接取 targetCfg[id].atk)
    ensureTargetCfg () {
      const cfg = Object.assign({}, this.targetCfg)
      for (const t of (this.troopsData.cfgs || [])) {
        // ★ 2026-10-28 defCount = 兵种防守数量（空 = 全部），与后端 ezfy_city_target.def_count 同口径
        if (!cfg[t.id]) cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1, defCount: '' }
      }
      this.targetCfg = cfg
    },
    loadTechs () {
      // ★ 2026-09-28 多城研究：科技页按**当前城**视角加载（科研中心等级/研究限制）
      api.get('/games/ezfy/techs?city_id=' + (this.city ? this.city.id : 0)).then(r => {
        if (r.code === 0) this.techsData = r.data
      })
    },
    // ★ 2026-10-23 顶部导航「聊天 / 邮箱」感叹号角标：未读私聊 + 未读邮件。
    //   后端一次返回两个数（两条 COUNT 并行），前端 30s 轮询；页面内已读操作后再主动刷新。
    loadUnread () {
      if (!this.isLogin) return
      api.get('/games/ezfy/unread').then(r => {
        if (r.code === 0 && r.data) {
          this.unreadChat = r.data.chat || 0
          this.unreadMail = r.data.mail || 0
        }
      })
    },
    loadChats () {
      api.get('/games/ezfy/chat?channel=' + this.chatChannel +
              '&page=' + this.chatPage + '&size=' + this.chatSize).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.worldChats = d.chats || []
          this.chatPlayers = d.players
          this.chatHasCorps = !!d.has_corps
          this.chatCorpsName = d.corps_name || ''
          this.chatCorpsPlayers = d.corps_players || 0
          this.chatCanSend = d.can_send !== false
          this.chatTotal = d.total || 0
          if (d.page) this.chatPage = d.page
          // ★ 军团频道但没军团：后端会降级成公共频道并把世界消息塞回来，
          //   这里按「军团频道」的语义清空，改为显示 0 人 + 引导。
          if (this.chatChannel === 2 && !this.chatHasCorps) {
            this.worldChats = []
            this.chatPlayers = 0
            this.chatCorpsPlayers = 0
            this.chatCanSend = false
            this.chatTotal = 0
          }
        }
      })
    },
    // 聊天翻页（-1 上一页 / 1 下一页）
    chatGo (delta) {
      const next = this.chatPage + delta
      if (next < 1 || next > this.chatTotalPages) return
      this.chatPage = next
      this.loadChats()
    },
    switchChannel (ch) {
      this.chatChannel = ch
      this.chatMsg = ''
      this.chatPage = 1
      this.loadChats()
    },
    // 首页「世界聊天」预览: 汇总 个人/同盟/系统 三个来源并带标识
    loadHomeChats () {
      api.get('/games/ezfy/chat/home').then(r => {
        if (r.code === 0) {
          this.homeChats = r.data.chats || []
          this.chatPlayers = r.data.players
        }
      })
    },
    // ★★ 2026-10-13 用户规则「消息全部都用二战自己的库」：
    //   删掉了这里原来的 doSendPm / loadMails / loadPmCandidates —— 它们打的是**家园**的
    //   消息接口（POST /messages、GET /messages/conversations、GET /messages/inbox），
    //   写/读的都是家园库，二战界面上要么是死代码、要么拉回来根本没人渲染。
    //   二战的消息一律走 /games/ezfy/*（privchat / mail / chat / corps/chat），
    //   后端也只读写二战库（private_messages / ezfy_mail / ezfy_chat / ezfy_corps_chat）。
    // ★ 私聊：打开与某位玩家的聊天（选中谁就聊谁，并带出历史记录）
    //   userId 为空时只进页面、显示会话列表。
    // ★ 顶部导航「邮箱」入口：直接进邮箱页（私聊已迁到好友页，见 openFriendPm）
    //   userId 有值（统帅信息页 [发私聊/发私信]）：是好友 → 好友页开私聊面板；
    //   非好友 → 邮箱「写邮件」，收件人自动填好对方昵称（符合「非好友用邮箱联系」规则）。
    openPm (userId) {
      if (userId) {
        const f = this.friends.find(x => x.user_id === userId)
        if (f) { this.cur = 'friends'; this.friendTab = 'list'; this.openFriendPm(f); return }
        this.cur = 'mail'
        this.setMailTab('compose')
        this.mailCompose.to = (this.playerInfo && this.playerInfo.nickname) || ''
        this.loadMail('inbox', true)
        return
      }
      this.go('mail')
    },
    // ---- 好友页私聊（微信式：会话列表 ⇄ 对话线程，仅好友之间可私聊） ----
    openFriendPm (f) {
      this.cur = 'friends'
      this.friendTab = 'list'
      this.friendView = 'convs'
      this.pmPeer = null
      this.pmChat = []
      this.pmContent = ''
      // ★ 发送统一用「好友游戏ID」：数值 → 后端按 game_uid 精确解析，不会因昵称不一致而失败
      this.pmPeerUid = f.game_uid || 0
      this.pmChatPage = 1
      this.selectPm(f.user_id)
      this.loadPmConvs()
    },
    // 会话列表点某条会话 → 直接打开对话线程（c 来自 convs 接口，带 gid/gname）
    openConv (c) {
      this.cur = 'friends'
      this.friendTab = 'list'
      this.friendView = 'convs'
      this.pmPeer = null
      this.pmChat = []
      this.pmContent = ''
      this.pmPeerUid = c.gid || 0
      this.pmChatPage = 1
      this.selectPm(c.user_id)
    },
    // 返回会话列表
    closePm () {
      this.pmPeer = null
      this.pmChat = []
      this.pmContent = ''
      this.pmPeerUid = 0
      this.pmChatPage = 1
      this.pmChatTotal = 0
      this.loadPmConvs()
    },
    switchFriendView (v) {
      this.friendView = v
      if (v === 'convs') this.closePm()
    },
    doSendPrivchat () {
      if (!this.pmPeer) { this.notify('请先选择要私聊的好友'); return }
      const content = (this.pmContent || '').trim()
      if (!content) { this.notify('请填写内容'); return }
      // ★ 优先用好友游戏ID发送；没有游戏ID时才回退游戏昵称
      const to = this.pmPeerUid ? String(this.pmPeerUid) : (this.pmPeer.gname || this.pmPeer.nickname)
      api.post('/games/ezfy/privchat/send', { to: to, content: content }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送')
          this.pmContent = ''
          this.pmChatPage = 1
          this.selectPm(this.pmPeer.id)
        } else this.notify(r.msg || '发送失败')
      })
    },
    // ---- 好友页四 tab ----
    setFriendTab (t) {
      this.friendTab = t
      if (t !== 'list') this.closePm()
    },
    // ---- 邮箱页 ----
    setMailTab (t) {
      this.mailTab = t
      this.mailOpen = 0
      if (t !== 'compose') { this.mailPage = 1; this.loadMail(t, true) }
    },
    loadMail (tab, force) {
      const t = tab || this.mailTab
      if (t === 'compose') return
      if (!force && this._mailLoading === t) return
      this._mailLoading = t
      api.get('/games/ezfy/mail', { params: { tab: t, page: this.mailPage, size: this.mailSize } }).then(r => {
        this._mailLoading = ''
        if (r.code === 0) {
          const d = r.data
          if (t === 'inbox') this.mailInbox = d.list || []
          else if (t === 'system') this.mailSystem = d.list || []
          else this.mailSent = d.list || []
          this.mailTotal = d.total || 0
          this.mailPage = d.page || 1
          this.mailSize = d.size || this.mailSize
          this.mailUnread = d.unread || { inbox: 0, system: 0, total: 0 }
          // ★ 2026-10-23 顶部导航「邮箱」感叹号：邮件页拿到的未读总数与角标同口径
          this.unreadMail = this.mailUnread.total || 0
        } else this.notify(r.msg || '加载失败')
      }).catch(() => { this._mailLoading = '' })
    },
    mailPrev () { if (this.mailPage > 1) { this.mailPage--; this.loadMail(this.mailTab, true) } },
    mailNext () { if (this.mailPage * this.mailSize < this.mailTotal) { this.mailPage++; this.loadMail(this.mailTab, true) } },
    mailList (tab) { return tab === 'inbox' ? this.mailInbox : tab === 'system' ? this.mailSystem : this.mailSent },
    mailSender (m) { return m.sender_id === 0 ? '系统' : (m.sender_name || ('玩家' + m.sender_id)) },
    mailToggle (m) {
      this.mailOpen = this.mailOpen === m.id ? 0 : m.id
      if (m.is_read === 0) this.doMailRead(m)
    },
    doMailRead (m) {
      if (m.is_read !== 0) return
      m.is_read = 1
      this.mailUnread.total = Math.max(0, (this.mailUnread.total || 0) - 1)
      this.unreadMail = Math.max(0, (this.unreadMail || 0) - 1)
      if (this.mailTab === 'inbox') this.mailUnread.inbox = Math.max(0, (this.mailUnread.inbox || 0) - 1)
      else if (this.mailTab === 'system') this.mailUnread.system = Math.max(0, (this.mailUnread.system || 0) - 1)
      api.post('/games/ezfy/mail/read', { id: m.id })
    },
    doMailReadAll () {
      api.post('/games/ezfy/mail/readall', { tab: this.mailTab }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已全部标记已读')
          this.loadMail(this.mailTab, true)
          this.loadUnread()
        } else this.notify(r.msg || '操作失败')
      })
    },
    doMailDelete (m) {
      this.ask('确定删除这封邮件吗？（只删除你自己视角的副本）').then(ok => {
        if (!ok) return
        api.post('/games/ezfy/mail/delete', { id: m.id }).then(r => {
          if (r.code === 0) { this.notify(r.data && r.data.msg ? r.data.msg : '邮件已删除'); this.mailOpen = 0; this.loadMail(this.mailTab, true) }
          else this.notify(r.msg || '操作失败')
        })
      })
    },
    // ★ 2026-10-18 领取：前端防抖（同一封邮件请求未回之前不再发第二次），后端还有 300ms 限流兜底。
    //   真正的「不重复发奖」由后端 `WHERE claimed = 0` 条件更新保证。
    doMailClaim (m) {
      if (!m || m.claimed === 1 || !m.can_claim) return
      if (this.mailClaiming) return // 已有领取请求在飞，忽略连点
      this.mailClaiming = m.id
      api.post('/games/ezfy/mail/claim', { id: m.id }).then(r => {
        this.mailClaiming = 0
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '领取成功')
          m.claimed = 1
          m.can_claim = false
          this.loadMail(this.mailTab, true)
        } else this.notify(r.msg || '领取失败')
      }).catch(() => { this.mailClaiming = 0 })
    },
    doSendMail () {
      const to = (this.mailCompose.to || '').trim()
      const subject = (this.mailCompose.subject || '').trim()
      const content = (this.mailCompose.content || '').trim()
      if (!to) { this.notify('请填写收件人（昵称或游戏ID）'); return }
      if (!content) { this.notify('请填写邮件内容'); return }
      api.post('/games/ezfy/mail/send', { to: to, subject: subject, content: content }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '邮件已发送')
          this.mailCompose = { to: '', subject: '', content: '' }
          this.loadMail('inbox', true)
          this.setMailTab('sent')
        } else this.notify(r.msg || '发送失败')
      })
    },
    loadPmConvs (page) {
      if (!page) page = this.pmConvPage
      if (page < 1) page = 1
      // ★ 2026-10-13 私聊会话走游戏内接口（游戏ID+游戏名），不再用家园名字
      api.get('/games/ezfy/privchat/convs', { params: { page, size: this.pmConvSize } }).then(r => {
        if (r.code === 0) {
          this.pmConvs = r.data.list || []
          this.pmConvPage = r.data.page || 1
          this.pmConvTotal = r.data.total || 0
          this.pmConvSize = r.data.size || this.pmConvSize
        }
      })
    },
    // ★ 会话列表翻页：上一页 / 下一页
    pmConvPrev () {
      if (this.pmConvPage > 1) this.loadPmConvs(this.pmConvPage - 1)
    },
    pmConvNext () {
      if (this.pmConvPage * this.pmConvSize < this.pmConvTotal) this.loadPmConvs(this.pmConvPage + 1)
    },
    // 切换聊天对象：拉出「我和 TA」的私聊记录（分页，新到旧→前端反转成旧到新展示）
    // ★ 2026-10-13 走游戏内接口，对方显示游戏ID+游戏名（不用家园名字）
    selectPm (userId) {
      if (!userId) return
      api.get('/games/ezfy/privchat/with/' + userId, { params: { page: this.pmChatPage, size: this.pmChatSize } }).then(r => {
        if (r.code === 0) {
          this.pmPeer = r.data.peer || null
          const list = r.data.list || []
          this.pmChat = list.slice().reverse()
          this.pmChatTotal = r.data.total || 0
          this.pmChatPage = r.data.page || 1
          this.pmChatSize = r.data.size || this.pmChatSize
          // ★ 2026-10-23 打开会话后后端已把该好友的私聊标为已读 → 刷新顶部「聊天」角标
          this.loadUnread()
          // ★ 2026-10-13 原来这里还会 this.loadMails() 打家园的 /messages/inbox，
          //   结果拉回来没人渲染（已随「消息全部用二战库」一起删掉）。
          this.loadPmConvs()
        } else {
          this.notify(r.msg || '打开会话失败')
        }
      })
    },
    // 私聊记录翻页（好友私聊面板）
    pmChatPrev () {
      if (this.pmChatPage > 1) { this.pmChatPage--; this.selectPm(this.pmPeer.id) }
    },
    pmChatNext () {
      if (this.pmChatPage * this.pmChatSize < this.pmChatTotal) { this.pmChatPage++; this.selectPm(this.pmPeer.id) }
    },
    // ★ 游戏内好友（不碰家园 /friends）
    loadFriends () {
      api.get('/games/ezfy/friends').then(r => {
        if (r.code === 0) this.friends = r.data.list || []
      })
      this.loadFriendApplies()
    },
    loadFriendApplies () {
      api.get('/games/ezfy/friends/applies').then(r => {
        if (r.code === 0) this.friendApplies = { inbox: r.data.inbox || [], outbox: r.data.outbox || [] }
      })
    },
    // ---- 好友搜索/添加(复刻 addToFriend) ----
    doFriendSearch () {
      const kw = (this.friendKeyword || '').trim()
      if (!kw) { this.notify('请输入游戏ID / 玩家号码 / 昵称'); return }
      // ★ 先走游戏内搜索（支持「游戏ID」，不随家园号码变化）；
      // ★ 只搜游戏内玩家（不回落家园 /friends/search，避免把家园好友混进来）
      api.get('/games/ezfy/friends/search?keyword=' + encodeURIComponent(kw)).then(r => {
        if (r.code === 0) {
          this.friendSearchList = r.data.list || []
          this.friendSearchDone = true
        } else {
          this.notify(r.msg || '搜索失败')
        }
      }).catch(() => this.notify('搜索失败'))
    },
    async doAddFriend (u) {
      const remark = await this.ask('给 ' + u.nickname + ' 的验证信息（可留空）', { input: true, placeholder: '可留空' })
      if (remark === null) return
      api.post('/games/ezfy/friends/apply', { target_id: u.user_id, remark: remark }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送好友申请')
          this.doFriendSearch()
          this.loadFriends()
        } else {
          this.notify(r.msg || '添加失败')
        }
      })
    },
    // 处理好友申请（同意/拒绝）
    doHandleApply (a, agree) {
      api.post('/games/ezfy/friends/handle', { apply_id: a.apply_id, agree: agree }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : (agree ? '已同意' : '已拒绝'))
          this.loadFriends()
        } else this.notify(r.msg || '操作失败')
      })
    },
    // 删除游戏内好友
    doDelFriend (f) {
      this.ask('确定解除与「' + f.nickname + '」的游戏好友关系吗？').then(ok => {
        if (!ok) return
        api.post('/games/ezfy/friends/delete', { friend_id: f.user_id }).then(r => {
          if (r.code === 0) { this.notify(r.data && r.data.msg ? r.data.msg : '已解除'); this.loadFriends() }
          else this.notify(r.msg || '操作失败')
        })
      })
    },
    // soft=true：切 tab 触发的**软刷新** —— 命中缓存(2 秒内)/请求在途就跳过，
    //   避免「切一下打一次接口」把服务器压住（线上 1 核）；内容已由 applyCachedReports 先行渲染。
    // soft=false/缺省：强制拉取（[查询]/[清空]/删除后/军团战报自愈 等都需要拿到最新）。
    loadReports (soft) {
      // ★ 2026-10-09 「没军团不显示军团战报 tab」：?rtab=5 残留 / 退团后残留 → 回落「战斗报告」。
      //   只在**已确认没有军团**时回落（reportHasCorps===false），避免刚进军情页时把团员的 tab 误踢掉。
      if (this.reportTab === 5 && this.reportHasCorps === false) this.reportTab = 4
      // category: 1 军情警讯 2 战斗报告(战报查询)；reportTab===5 → 军团战报(corps)
      const tab = this.reportTab
      const cat = tab === 3 ? 1 : 2
      const key = this.repCacheKey(tab)
      // ★ 严格判 `=== true`：模板里 `@click="ezfy.loadReports"` 会把**点击事件对象**当第一个参数传进来
      //   （不是布尔），那种情况要当**强制刷新**处理，不能被误判成软刷新。
      if (soft === true) {
        const cached = this.repCache[key]
        // ★ 2026-10-11 切 tab 流畅化：该分区刚拉过(2s) → 直接用缓存（内容已由 applyCachedReports 渲染好）
        if (cached && Date.now() - cached.at < 2000) return
        // ★★ 去重**只按同一个 key**：同一分区已有请求在途 → 让它回来填，避免并发重复请求。
        //    原来用的是全局 `_repLoading`：切分区时「别的分区的请求在途」会把本分区这次请求也挡掉，
        //    而 applyCachedReports 已经把列表清空 → 本分区永远显示「暂无」（用户报「看不到战报」）。
        if (this.repLoadingKey === key) return
      }
      let url = '/games/ezfy/reports?'
      if (tab === 5) url += 'corps=1'
      // ★★ 2026-10-16 「战斗报告也改成查全部」：后端列表**已不再按城市过滤**
      //   （category=1/2 都是全账号口径，见后端 Reports）→ 这里传的 city_id 只用于
      //   页面级「雷达/侦察/情报等级」兜底（军情警讯还有后端逐条下发的权威口径）。
      //   tab3 仍传 0：其页面级字段本就落到主城，逐条字段才是权威。
      else url += 'category=' + cat + '&city_id=' + (tab === 3 ? 0 : (this.city ? this.city.id : 0))
      if (this.reportWord) url += '&word=' + encodeURIComponent(this.reportWord)
      this.repLoadingKey = key
      this.repLoading = true
      api.get(url).then(r => {
        if (this.repLoadingKey === key) this.repLoadingKey = ''
        if (this.repCacheKey(this.reportTab) === key) this.repLoading = false
        if (r.code === 0) {
          if (r.data.has_corps !== undefined) this.reportHasCorps = !!r.data.has_corps
          // ★ 2026-10-09 「战斗报告(14) 数量问题」：军团战报接口返回的 counts 是
          //   **本军团团员 PvP 战报的总数**（不是自己的未读数）→ 拿它更新徽标会让
          //   「战斗报告(N)」显示成全团的数量。徽标只认 /reports/counts 与 3/4 分区的 counts。
          if (tab !== 5) this.reportCounts = r.data.counts || {}
          // 自愈：军团战报接口明确回 has_corps=false（已退团）→ 回落到战斗报告再拉一次
          if (tab === 5 && r.data.has_corps === false) {
            this.reportTab = 4
            this.loadReports()
            return
          }
          const data = {
            reports: r.data.reports || [],
            radar: r.data.radar || 0,
            recon: r.data.recon || 0,
            intel: r.data.intel || r.data.radar || 0,
            at: Date.now()
          }
          // 写缓存（按 key，城市/搜索词变化时旧缓存自然不被命中）
          this.repCache[key] = data
          // ★ 只有**仍停在原分区、且城市/搜索词没变**时才替换列表 ——
          //   否则慢响应回来会把用户已经切到的别的分区内容覆盖掉。
          if (this.reportTab === tab && this.repCacheKey(tab) === key) {
            this.reports = data.reports
            this.reportRadar = data.radar
            this.reportRecon = data.recon
            this.reportIntel = data.intel
            if (soft !== true) this.repPage = 1
          }
        }
      }).catch(() => {
        if (this.repLoadingKey === key) this.repLoadingKey = ''
        if (this.repCacheKey(this.reportTab) === key) this.repLoading = false
      })
    },
    // ★ 2026-10-01 徽标数字专用轻量接口（独立于 loadReports：
    //   军情页落在任意分区都要刷新徽标，但不能因此把没看的战报标记已读）
    //   ★ 2026-10-01 军情按当前城过滤：徽标数字也只统计当前城的战报
    //   ★ 2026-10-09 顺带取 has_corps：军情页据此决定要不要显示「军团战报」tab
    //     （这个接口进页/切分区必调，比 myCorps 更早、更稳）。
    //   ★ 2026-10-11 切 tab 流畅化：加「请求在途」守卫（连点 tab 不堆请求），
    //     **不设 TTL** —— 徽标必须及时反映「已看战报被标记已读」，不能拖。
    loadReportCounts () {
      if (this.countsLoading) return
      this.countsLoading = true
      api.get('/games/ezfy/reports/counts?city_id=' + (this.city ? this.city.id : 0)).then(r => {
        this.countsLoading = false
        if (r.code === 0) {
          this.reportCounts = r.data.counts || {}
          // ★★ 2026-10-13 顶部导航「军情」感叹号角标：**优先用后端下发的 unread_warns**
          //   （只算被动挨打类，见后端 `ezfyReportAlertCond`），不要再拿两个 tab 徽标相加 ——
          //   徽标是全量口径（含自己刷野地的「掠夺报告」动辄几千条），相加会让角标永远闪、提示失去意义。
          //   仅当后端旧版本没这个字段时才回落到「军情警讯 + 战斗报告」的旧口径。
          this.unreadWarns = (r.data.unread_warns !== undefined)
            ? (r.data.unread_warns || 0)
            : ((this.reportCounts[1] || 0) + (this.reportCounts[2] || 0))
          if (r.data.has_corps !== undefined) this.reportHasCorps = !!r.data.has_corps
        }
      }).catch(() => { this.countsLoading = false })
    },
    // ★ 2026-09-24 军情警讯列表加 [防守报告]/[预警] 标签（其余类型无标签）
    intelTag (r) {
      const t = (r && r.title) || ''
      if (t.indexOf('守卫报告') === 0) return '防守报告'
      if (t.indexOf('军情警报') === 0) return '预警'
      return ''
    },
    // 守卫报告标题去掉冗余的「守卫报告: 」前缀，保留 城市名(坐标)
    intelTitle (r) {
      const t = (r && r.title) || ''
      if (t.indexOf('守卫报告: ') === 0) return t.slice('守卫报告: '.length)
      return t
    },
    // ---- 军队动态 ----
    // ★★ 2026-10-08 军情页分区按 `this.city.id` 过滤：军队动态 / 驻军 / 战斗报告
    //   （后端 ReportDynamics 已支持 city_id 过滤：订单按 city_id(出发城) 过滤、
    //   守方战场按 battle.target_id(被打城) 过滤）。
    //   ★★ 2026-10-13 例外：**军情警讯不再按城市**（不受城市限制，查全部城市，见 loadReports 的 url 构造）。
    //   ★ 出征队列(orders)、地图页「出征队列」面板不在这四个类别里，仍展示**全部在途部队**
    //     （每条都带 from_city 标明来源城，见后端下发的 from_city/from_x/from_y）。
    // soft=true：切 tab 触发的软刷新 —— 同一城市 2 秒内刚拉过 / 已有请求在途就跳过，
    //   避免连点 tab 堆一串请求（线上 1 核扛不住）。缺省=强制拉（动作后刷新、到点重拉等）。
    loadDynamics (soft) {
      // ★★ 2026-10-07 修复「出征后跳转出征队列是空的 / 打野地也没了」：
      //   原来**所有页面**都按 `city_id = 当前所在城` 过滤，而 `doOrder` 里
      //   `load()`（刷新城市）与 `go('orders')`（拉队列）是**并发**的，出征瞬间
      //   `this.city` 可能还是旧值 / 尚未回来 → 过滤条件对不上 → 队列恒为空。
      //   现在只在**军情页**按城过滤，出征队列/地图队列面板保持全集，问题不复现。
      const onlyCity = (this.cur === 'reports' || this.cur === 'reportview')
      const cid = (onlyCity && this.city) ? this.city.id : 0
      // ★ 2026-10-11 切 tab 流畅化：软刷新时去重（在途/2 秒内）→ 连点 tab 不再重复打接口
      //   ★ 严格判 `=== true`：模板里 `@click="ezfy.loadDynamics"` 会把点击事件对象当参数传进来，
      //     那种情况要当强制刷新（见 loadReports 同款说明）。
      if (soft === true) {
        if (this.dynLoading) return
        if (this.dynKey === cid && this.dynAt && Date.now() - this.dynAt < 2000) return
      }
      this.dynLoading = true
      api.get('/games/ezfy/reports/dynamics?city_id=' + cid).then(r => {
        this.dynLoading = false
        if (r.code === 0) {
          this.dynamics = r.data.dynamics || []
          // ★ 2026-09-28 倒计时自动刷新的时间基点：以「拿到数据的这一刻」为准，
          //   战斗中的 battle_left_ms 是相对剩余，必须配上它才能每秒往前推算。
          this.dynAt = Date.now()
          this.dynKey = cid
          this.dynPage = 1
          this.queuePage = 1
        }
      }).catch(() => { this.dynLoading = false })
    },
    // ================= 战场指挥室（军情 → 军队动态 → [指挥]）=================
    // 每回合 30 秒：前 25 秒可下达前进/暂停/后退，后 5 秒锁定并由服务器结算，最多 40 回合。
    // 倒计时在前端本地自减，归零时拉一次服务端 —— 服务端是懒结算，请求时按时间补算回合。
    openBattle (orderId) {
      this.battleOrderId = orderId
      this.battleLeftMs = 0
      this.battleData = Object.assign({}, this.battleData, {
        order_id: orderId, done: false, actions: [], round: 0, win: 0
      })
      this.go('battle')
      this.loadBattle()
    },
    loadBattle () {
      if (!this.battleOrderId) return
      api.get('/games/ezfy/battle', { params: { order_id: this.battleOrderId } }).then(r => {
        if (r.code !== 0) {
          this.notify(r.msg || '战场不存在')
          this.leaveBattle()
          return
        }
        this.applyBattleState(r.data)
        if (r.data.done) this.stopBattleTimer()
        else this.startBattleTimer()
      })
    },
    startBattleTimer () {
      this.stopBattleTimer()
      this.battleTimer = setInterval(() => {
        if (this.battleData.done) { this.stopBattleTimer(); return }
        // ★ 2026-10-06 指令/目标 API 往返期间不轮询：避免整体替换 battleData 把
        //   正在操作的下拉弹回旧值（用户反馈「指挥模块下拉不能自己选」）。
        if (this.battleCmdInFlight) return
        this.battleLeftMs -= 1000
        if (this.battleLeftMs <= 0) {
          this.battleLeftMs = 0
          this.loadBattle()   // 到点 → 拉服务端推进一回合
        }
      }, 1000)
    },
    stopBattleTimer () {
      if (this.battleTimer) { clearInterval(this.battleTimer); this.battleTimer = null }
    },
    // ★★ 2026-10-13 战场位置条坐标映射：**攻守共用同一把尺**（用户反馈「【守方】战列舰 已与攻方航母贴身，
    //   原地攻击，但位置条上两个图标离得老远」）。
    //   后端坐标语义：攻方老家 0、守方老家 6000（初始相距 6000），前进 = 攻方 pos 增大 / 守方 pos 减小，
    //   各自最多后退 10000 → 坐标空间 [-10000, 16000]，**贴身的判据就是两个 pos 只差 1**。
    //   旧实现是**分区映射**（攻方恒在左半区 [3%,46%]、守方恒在右半区 [54%,97%]）：它按「各自相对老家的
    //   进度」定位，于是哪怕两个单位已经贴脸，图标之间仍固定隔着 ~24% 行宽 —— 看着像隔着半个战场，
    //   和行动日志里的「贴身」完全对不上。
    //   现在按**绝对 pos** 映射（攻守同一把尺）：pos 相近 → 图标自然挨在一起，贴身时基本重合；
    //   而同一对实际交战单位的 pos 天然有序（攻方 pos 恒小于守方 pos），所以照样不会穿过。
    //   三段线性：两侧撤退区各压到 15%，中间主战区放大到 64% —— 交战区的位移看得更清楚。
    //   ★ 同一行的「保序 + 最小间隔」在 battlePosRows 里做（那里才能同时看到攻守两侧）。
    battlePosLeft (u) {
      const p = (u && u.pos !== undefined && u.pos !== null && !isNaN(u.pos)) ? u.pos : 3000
      if (p <= 0) return 3 + (Math.max(-10000, p) + 10000) / 10000 * 15   // 攻方撤退区 [-10000,0] → [3%,18%]
      if (p >= 6000) return 82 + (Math.min(16000, p) - 6000) / 10000 * 15 // 守方撤退区 [6000,16000] → [82%,97%]
      return 18 + p / 6000 * 64                                          // 主战区 [0,6000] → [18%,82%]
    },
    // ★ 2026-10-10 位置条兵种名：只显示首字（图标下 26px 宽放不下全名）。
    //   全名 + 位置仍保留在 title 提示里。
    battleUnitName (u) {
      const n = (u && u.name) || ''
      return n ? n.charAt(0) : ''
    },
    // 兵种图标：位置条直接复用「军队(城内军队/城防)」页面同一套图标 —— ezfy.troopIco(id)。
    // 这里把战斗单位(用 troop_id)包成 troopIco 需要的 {id} 结构（阵营配色沿用玩家自己的阵营）。
    battleUnitCfg (u) {
      return { id: u && u.troop_id }
    },
    // ★★ 2026-10-12 逐兵种「指挥 / 目标」下拉的回填（用户反馈三点）：
    //   ① 点[全军前进/待命/后退]后，单兵种的「指挥」下拉没跟着变（还显示"前进"）；
    //   ② 强制刷新后下拉被打回「最近 / 前进」，看不到刷新前自己的选择。
    //   根因：这两个 select 是**非受控**的（不绑 value）→ 服务端状态怎么变，DOM 里的选中项都不动。
    //   改法：select 绑 `:value="battleCmdSel(u)"` 做受控回填，但**优先返回本地待确认值**
    //   （battleCmds/battleTargets）→ 选完立刻显示新值，也不会被轮询回包弹回（老方案的老毛病）。
    //   服务端回包与本地值一致时清掉本地项（syncBattleLocal），之后完全以服务端为准。
    battleCmdSel (u) {
      const k = u && u.troop_id
      if (k === undefined || k === null) return ''
      const v = this.battleCmds[k]
      if (v !== undefined) return v
      return u.cmd || ''
    },
    battleTargetSel (u) {
      const k = u && u.troop_id
      if (k === undefined || k === null) return '0'
      const v = this.battleTargets[k]
      if (v !== undefined) return String(v)
      return String(u.target_troop || 0)
    },
    // 服务端状态回来了：本地待确认值与服务端一致 → 清掉本地项（此后以服务端为准）
    syncBattleLocal (st) {
      if (!st) return
      const own = st.is_atk ? (st.attackers || []) : (st.defenders || [])
      own.forEach(u => {
        const k = u.troop_id
        if (this.battleCmds[k] !== undefined && this.battleCmds[k] === (u.cmd || '')) {
          this.$delete(this.battleCmds, k)
        }
        if (this.battleTargets[k] !== undefined && String(this.battleTargets[k]) === String(u.target_troop || 0)) {
          this.$delete(this.battleTargets, k)
        }
      })
    },
    // 统一应用服务端战场状态：所有入口都走它，避免漏掉下拉本地值的同步清理
    applyBattleState (st) {
      if (!st) return
      this.battleData = st
      this.battleLeftMs = st.round_left_ms || 0
      this.syncBattleLocal(st)
    },
    // troopId 省略 = 全军快捷指令；给了 troopId = 给该兵种**单独**下指令
    sendBattleCmd (cmd, troopId) {
      if (!this.battleOrderId) return
      const send = cmd || ''
      // ★ 2026-10-12 先写本地待确认值 → 下拉立刻显示新值；**全军命令则给自己这一方所有兵种都写上**，
      //   这样每个单兵种的「指挥」下拉会跟着一起变（用户反馈「全军命令生效了但单兵种下拉不动」）
      const own = this.battleData.is_atk ? (this.battleData.attackers || []) : (this.battleData.defenders || [])
      if (troopId > 0) this.$set(this.battleCmds, troopId, send)
      else own.forEach(u => this.$set(this.battleCmds, u.troop_id, send))
      this.battleCmdInFlight = true
      api.post('/games/ezfy/battle/cmd', {
        order_id: this.battleOrderId, troop_id: troopId || 0, cmd: send
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '指令失败'); return }
        const st = r.data && r.data.state
        if (st) this.applyBattleState(st)
        this.notify((troopId ? '该兵种' : '全军') + '已设' + this.battleCmdName(send))
        if (r.data && r.data.done) this.stopBattleTimer()
      }).finally(() => { this.battleCmdInFlight = false })
    },
    // ★ 指挥时逐兵种改「优先攻击目标」（2026-09-23 ）：
    //   默认值来自司令部「兵种战斗配置」，这里改的只是**本场战斗**，不回写司令部。
    //   target = 0 表示「最近目标」；守方没有该兵种时服务器会自动回落打最近的。
    sendBattleTarget (troopId, val) {
      if (!this.battleOrderId) return
      const target = parseInt(val, 10) || 0
      // ★ 2026-10-12 同上：先写本地待确认值 → 下拉立刻显示该目标，刷新时也不会被打回「最近」
      if (troopId > 0) this.$set(this.battleTargets, troopId, target)
      const opt = (this.battleData.target_options || []).find(o => o.id === target)
      this.battleCmdInFlight = true
      api.post('/games/ezfy/battle/target', {
        order_id: this.battleOrderId, troop_id: troopId, target_troop: target
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '目标设置失败'); return }
        const st = r.data && r.data.state
        if (st) this.applyBattleState(st)
        this.notify('该兵种目标已设为' + (opt ? opt.name : '最近'))
        if (r.data && r.data.done) this.stopBattleTimer()
      }).finally(() => { this.battleCmdInFlight = false })
    },
    doBattleAuto () {
      if (!this.battleOrderId) return
      api.post('/games/ezfy/battle/auto', { order_id: this.battleOrderId }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '操作失败'); return }
        const st = r.data && r.data.state
        if (st) this.applyBattleState(st)
        this.notify('已按「全军前进」打完这场战斗，战报稍后可在军情里查看')
        this.stopBattleTimer()
      })
    },
    // ★ 2026-10-06 保存/取消本回合配置：
    //   保存 = 本回合伤害锁定（禁改指令/目标），攻守双方都保存或到点即结算；
    //   取消 = 仅限本回合尚未结算时，解除锁定以便重新配置。
    saveBattleConfig () {
      if (!this.battleOrderId) return
      api.post('/games/ezfy/battle/lock', { order_id: this.battleOrderId, lock: true }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '保存失败'); return }
        const st = r.data && r.data.state
        if (st) this.applyBattleState(st)
        this.notify(st && st.done ? '双方配置已锁定，本回合结算' : '本回合配置已锁定')
        if (r.data && r.data.done) this.stopBattleTimer()
      })
    },
    cancelBattleConfig () {
      if (!this.battleOrderId) return
      api.post('/games/ezfy/battle/lock', { order_id: this.battleOrderId, lock: false }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '取消失败'); return }
        const st = r.data && r.data.state
        if (st) this.applyBattleState(st)
        this.notify('已取消锁定，可重新配置')
        if (r.data && r.data.done) this.stopBattleTimer()
      })
    },
    battleCmdName (c) {
      return c === '' || c == null ? '默认' : (c === 'hold' ? '待命' : (c === 'retreat' ? '后退' : '前进'))
    },
    // ★ 2026-10-06 指挥室：展开/收起逐回合详情（收起时只留回合标题，方便快速翻战况）
    toggleBattleDetail () {
      this.battleDetailOn = !this.battleDetailOn
    },
    // ★ 战场指挥室：行动日志按攻守上色 —— 我方绿色（跟随 is_atk），敌军红色
    battleLineClass (text) {
      const t = text || ''
      const isAtk = /^【攻方】/.test(t)
      const isDef = /^【守方】/.test(t)
      if (!isAtk && !isDef) return 'gray'
      const mineAtk = !!this.battleData.is_atk
      if (isAtk) return mineAtk ? 'green' : 'red'
      return mineAtk ? 'red' : 'green'
    },
    // ★ 2026-10-06 战报视角判断：被掠夺/被征服/城破报告里玩家是守方 ——
    //   这时「守方=自己(绿)、攻方=敌军(红)」；其他战报（掠夺/征服/战斗）里玩家是攻方，
    //   「攻方=自己(绿)、守方=敌军(红)」。返回 true 表示当前战报玩家在守方。
    rptViewerIsDef () {
      const t = (this.curReport && this.curReport.title) || ''
      return /^被(掠夺|征服)报告/.test(t) || /^城破报告/.test(t)
    },
    // ★ 战报详情/逐回合详情按行上色（看不清谁是谁 → 视觉区分）。
    //   颜色始终「自己绿、敌军红」：rpt-atk=绿、rpt-def=红，攻/守哪边是自己看战报视角。
    //   返回 [{mode:'line'|'pair', text?, cls?, left?, right?}]。
    //   ★ 2026-10-09 「战斗加成」已拆成**两行**（攻方一行 / 守方一行）→ 走 mode:'line' 上色；
    //     这里保留老战报（单行、攻守用 ' | ' 分隔）的 pair 分支 —— 库里存量战报还是老格式。
    reportNiceLines (raw) {
      return String(raw || '').split('\n').map(ln => {
        const bi = ln.indexOf(' | ')
        if (ln.indexOf('战斗加成') >= 0 && bi > 0) {
          const left = ln.slice(0, bi + 1)
          const right = ln.slice(bi + 1)
          return {
            mode: 'pair',
            left: { text: left, cls: this.reportSideClass(left) },
            right: { text: right, cls: this.reportSideClass(right) }
          }
        }
        return { mode: 'line', text: ln, cls: this.reportLineClass(ln) }
      })
    },
    reportLineClass (ln) {
      const t = ln || ''
      // ★ 2026-10-09 战斗加成两行版：「战斗加成: 攻方 …」「战斗加成: 守方 …」
      //   （老的单行版含 ' | '，已被上面 reportNiceLines 拦掉，不会走到这里）
      if (t.indexOf('战斗加成') >= 0) {
        const a = t.indexOf('攻方') >= 0
        const d = t.indexOf('守方') >= 0
        if (a && !d) return this.rptViewerIsDef() ? 'rpt-def' : 'rpt-atk'
        if (d && !a) return this.rptViewerIsDef() ? 'rpt-atk' : 'rpt-def'
        return ''
      }
      const isAtk = /【攻方】|\[胜]攻方|\[平]攻方|【攻方军官】/.test(t) || (t.indexOf('攻方:') >= 0 && t.indexOf('守方:') < 0)
      const isDef = /【守方】|\[败]守方|\[平]守方|【守方军官】/.test(t) || (t.indexOf('守方:') >= 0 && t.indexOf('攻方:') < 0)
      if (isAtk && !isDef) return this.rptViewerIsDef() ? 'rpt-def' : 'rpt-atk'
      if (isDef && !isAtk) return this.rptViewerIsDef() ? 'rpt-atk' : 'rpt-def'
      return ''
    },
    reportSideClass (seg) {
      if (seg.indexOf('守方') >= 0 && seg.indexOf('攻方') < 0) return this.rptViewerIsDef() ? 'rpt-atk' : 'rpt-def'
      if (seg.indexOf('攻方') >= 0 && seg.indexOf('守方') < 0) return this.rptViewerIsDef() ? 'rpt-def' : 'rpt-atk'
      return ''
    },
    // resumeBattle 刷新页面后从后端找回「进行中的战斗」（订单 id 没存在 URL 里）
    resumeBattle () {
      api.get('/games/ezfy/reports/dynamics').then(r => {
        const list = (r.code === 0 && r.data && r.data.dynamics) || []
        const one = list.find(x => x.can_command)
        if (!one) {
          this.notify('当前没有正在进行的战斗')
          this.go('reports')
          return
        }
        this.battleOrderId = one.id
        this.loadBattle()
      })
    },
    leaveBattle () {
      this.stopBattleTimer()
      this.battleOrderId = 0
      this.go('reports')
    },
    // ★ 商城：切部位时回到第 1 页（否则会停在上一部位的分页位置看到空白）
    setShopSlot (s) {
      this.shopSlot = s
      this.shopPage = 1
    },
    // ★ 2026-10-09 宝箱内容页：点宝箱名/[查看] 进独立「宝箱内容」页看奖池与单项详情
    //   （原来在商城页内联展开奖池，检索/分页/详情全堆一页太乱，拆成独立页）
    openChestView (ch) {
      this.chestPoolId = ch.id
      this.chestPoolWord = ''
      this.chestPoolPage = 1
      this.chestViewDetailIdx = -1
      this.cur = 'chestview'
    },
    // ★ 背包：点 [说明] 展开/收起道具说明
    toggleBagDesc (cfgId) {
      this.bagDescId = (this.bagDescId === cfgId) ? 0 : cfgId
    },
    // ★ 背包：切分类时回到第 1 页
    setBagCat (c) {
      this.bagCat = c
      this.bagPage = 1
    },
    // ★ 开箱快捷数量（[5]/[10] 不能超过单次上限）
    setChestCount (n) {
      const max = (this.chestOpen && this.chestOpen.open_max) || n
      this.chestCount = Math.min(n, max)
    },
    // ★ 2026-10-25 keepPage：从「战报详情」[返回] 列表时**保住原页码**。
    //
    //	用户反馈：「我在第三页，点进去一条战报，再返回现在跑到第一页了，应该还是第三页」。
    //	原实现无条件 `repPage = 1`，于是详情页那条 `go('reports')` 一回来就回到第 1 页。
    //	★ 只在**同一个分区**时保留（换分区必须回第 1 页，否则会停在新分区的空白页）；
    //	  军队动态/驻军的分页（dynPage/dynStationPage）不属于战报详情这条路径，照旧重置。
    switchReportTab (t, keepPage) {
      const sameTab = t === this.reportTab
      this.reportTab = t
      this.curReport = null
      // ★ 切换分区时回到第 1 页，避免停在上一次的分页位置看到空白
      if (!(keepPage && sameTab)) this.repPage = 1
      this.dynPage = 1
      this.dynStationPage = 1
      // ★ 2026-09-28 分区写进 URL，否则刷新后回落到默认的「军队动态」(t=1)
      this.syncUrl()
      // ★★ 2026-10-11 切 tab 流畅化（用户：「这些 tab 切换还是卡顿」）：
      //   ① **先用缓存把内容渲染出来**（`applyCachedReports`）—— 不再「清空列表 → 等接口回来才显示」，
      //      点一下立刻有内容，切回来更是瞬开；
      //   ② 徽标 + 军队动态/驻军 + 战报列表都改成**软刷新**（soft=true）：命中缓存/请求在途就跳过，
      //      连点 tab 不会堆一串请求把服务器（线上 1 核）压住；
      //   ③ 没有缓存时才回落原来的「先清空占位」行为（首次进入某分区）。
      this.applyCachedReports(t)
      this.loadReportCounts(true)
      this.loadDynamics(true)
      if (t === 3 || t === 4 || t === 5) {
        this.loadReports(true)
      }
    },
    // 战报列表缓存 key：分区 + 城市 + 搜索词（任一变化即视为不同数据集，自然失效）
    // ★ 2026-10-13 军情警讯(t=3)不受城市限制 → 城市位**恒为 0**：
    //   切城市时军情警讯的数据本来就不变，key 保持一致 → 命中缓存、不重复打接口（也不会串城）。
    repCacheKey (t) {
      const cid = t === 3 ? 0 : ((this.city && this.city.id) || 0)
      return t + '|' + cid + '|' + (this.reportWord || '')
    },
    // 用缓存把某个分区的战报列表**立刻**渲染出来（无缓存则清空 + 标「加载中」）
    applyCachedReports (t) {
      if (t !== 3 && t !== 4 && t !== 5) return
      const c = this.repCache[this.repCacheKey(t)]
      if (!c) {
        // 首次进入该分区：没有缓存 → 清空，等接口回来（与旧行为一致，避免显示别的分区残留）。
        // ★ 标 repLoading=true → 模板显示「(加载中…)」，而不是误导性的「(暂无战斗报告)」。
        this.reports = []
        this.repLoading = true
        return
      }
      this.reports = c.reports
      this.repLoading = false
      this.reportRadar = c.radar
      this.reportRecon = c.recon
      this.reportIntel = c.intel
      // ★ 徽标不从这里恢复：loadReports 回的 counts 是**标记已读之前**算的（偏大），
      //   而 switchReportTab 紧接着会调 loadReportCounts()（无 TTL）拿准确值 ——
      //   这里不写，避免徽标数字先跳到偏大的值再回落（闪一下）。
    },
    // ★ 战报详情页(reportview)顶部的分区导航：先回列表页再切到对应分区，
    //   直接调 switchReportTab 会停在 reportview 页面上。
    goReportTab (t) {
      this.stopBattleTimer()
      this.cur = 'reports'
      // ★ 2026-10-25 传 keepPage=true：点「当前所在分区」= 回列表，页码不该被重置；
      //   点别的分区时 switchReportTab 内部会判定 sameTab=false，照旧回第 1 页。
      this.switchReportTab(t, true)   // 内部已 syncUrl，会把 cur=reports 与 rtab 一起写回
    },
    doCollectAll () {
      // ★★★ 2026-10-15 多 tab 防刷：一键采集账号级 3 秒共享卡控 + 请求防抖（防连点/多tab重复下达采集）
      if (!this.opGuard('collectAll', 3000)) return
      this.once('collectAll', () =>
      api.post('/games/ezfy/wild/collect-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          // ★ 2026-10-09 「点了采集没反应」：附属野地列表(loadWilds)也要重拉，
          //   否则列表还显示 [开始采集]，看起来像没生效（load() 已不再下发 wildlands）。
          this.loadWilds()
        } else this.notify(r.msg)
      })
      )
    },
    // ★ 单支空闲驻军开始采集（2026-09-24 采集空闲化）
    startCollect (o) {
      const oid = (o && typeof o === 'object') ? o.id : o
      api.post('/games/ezfy/wild/start-collect', { order_id: oid }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
          // ★ 2026-10-09 「点采集不管用」：附属野地列表必须一起刷新
          //   （原来只刷 dynamics + load()，而 load() 已不下发 wildlands → 列表停在「[开始采集]」）。
          this.loadWilds()
        } else this.notify(r.msg)
      })
    },
    // ★ 一键收获：对每支采集中部队结算产出(资源直接入起点城市, 宝物进背包), 并停止采集原地待命
    //   —— 与单支 [停止] 同一个功能，只是一个批量一个单个。
    //   （玩法细则见代码注释：满一期结算资源+宝物；不满一期按真实时长连续折算资源、无宝物；
    //     负重装满后超出部分丢弃、资源不再增加，但宝物继续；点[停止采集]/[一键收获]清空后才能重新采资源。）
    async doHarvestAll () {
      if (!await this.ask('确定收获所有采集中的部队吗？')) return
      // ★★★ 2026-10-15 多 tab 防刷：一键收获账号级 3 秒共享卡控（确认通过后才占冷却，误点取消不占）
      if (!this.opGuard('harvestAll', 3000)) return
      this.harvestAllReq(false)
    },
    // ★ 2026-09-30 出发城市资源已达「配置的资源最大值」时，后端返回 confirm 标记;
    //   弹确认框，玩家确认后带 force=true 重发才真正收获（否则批量产出入库会因资源上限被丢弃）。
    harvestAllReq (force) {
      // ★★★ 2026-10-15 请求防抖：同一批收获请求未返回时忽略后续点击（force 重发在上一发返回后才发起，不受影响）
      this.once('harvestAll', () =>
      api.post('/games/ezfy/wild/harvest-all', { force }).then(r => {
        if (r.code === 0) {
          if (r.data && r.data.confirm) {
            this.ask(r.data.msg).then(ok => {
              if (ok) this.harvestAllReq(true)
            })
            return
          }
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
          this.loadWilds() // ★ 2026-10-09 附属野地列表一起刷新（否则状态停在旧值）
        } else this.notify(r.msg)
      })
      )
    },
    // ★ 一键召回：先结算未入城产出, 部队返航(资源已在收获/停止时入城, 召回只是撤兵)
    //   （玩法细则见代码注释：返航前结算未入城产出，满一期给资源+宝物，不满一期只按时长折算资源，无宝物。）
    async doRecallAll () {
      if (!await this.ask('确定召回所有驻守部队吗？')) return
      // ★★★ 2026-10-15 多 tab 防刷：一键召回账号级 3 秒共享卡控（确认通过后才占冷却，误点取消不占）
      if (!this.opGuard('recallAll', 3000)) return
      this.once('recallAll', () =>
      api.post('/games/ezfy/wild/recall-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
          this.loadWilds() // ★ 2026-10-09 附属野地列表一起刷新
        } else this.notify(r.msg)
      })
      )
    },
    // ★ 单支采集部队「停止采集」(= 单支收获): 满一期结算资源+宝物, 不满只结算按采集时长的资源(无宝物);
    //   资源直接入起点城市; 部队原地待命不回城。
    async stopCollect (o) {
      const name = o.target_name + '(' + (o.target_x || 0) + ',' + (o.target_y || 0) + ')'
      if (!await this.ask('确定停止「' + name + '」采集吗？')) return
      this.stopCollectReq(o, false)
    },
    // ★ 2026-09-30 出发城市资源已达「配置的资源最大值」时，后端返回 confirm 标记;
    //   弹确认框，玩家确认后带 force=true 重发才真正停止结算。
    stopCollectReq (o, force) {
      api.post('/games/ezfy/wild/stop-collect', { order_id: o.id, force }).then(r => {
        if (r.code === 0) {
          if (r.data && r.data.confirm) {
            this.ask(r.data.msg).then(ok => {
              if (ok) this.stopCollectReq(o, true)
            })
            return
          }
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
          this.loadWilds() // ★ 2026-10-09 附属野地列表一起刷新（否则「[停止]」后仍显示采集中）
        } else this.notify(r.msg)
      })
    },
    loadTasks () {
      api.get('/games/ezfy/tasks').then(r => {
        if (r.code === 0) {
          this.taskGroups = r.data.groups || []
          // ★ 为爱发电卡：未发放时 love_cards 为空数组 → 隐藏对应 tab；若点过想去但被删除/清空则回落
          this.loveCards = r.data.love_cards || []
          if (this.taskTab === -1 && !this.loveCards.length) this.taskTab = 0
          this.selectTaskTab(this.taskTab)
        }
      })
    },
    // ★ 2026-09-27 任务 tab 选择并持久化（重启/刷新后仍在原 tab）
    selectTaskTab (v) {
      this.taskTab = v
      try { window.localStorage.setItem('ezfy_task_tab', String(v)) } catch (e) {}
    },
    // ★ 2026-09-27 恢复上次任务 tab（localStorage）
    restoreTaskTab () {
      try {
        const v = parseInt(window.localStorage.getItem('ezfy_task_tab') || '0', 10)
        return isNaN(v) ? 0 : v
      } catch (e) { return 0 }
    },
    // ★ 2026-09-28 司令部子 tab 切换并持久化（刷新后仍在原 tab）
    selectHqTab (v) {
      this.hqTab = v
      try { window.localStorage.setItem('ezfy_hq_tab', String(v)) } catch (e) {}
      // ★ 2026-09-28 预设编队 tab：拉预设 + 重算预览（预设页共享出征表单状态 orderOfficer/orderGather/orderTroops）
      if (v === 4) {
        this.loadPresets()
        this.$nextTick(() => this.presetCalc())
      }
    },
    // ★ 2026-09-28 恢复上次司令部子 tab（localStorage）
    restoreHqTab () {
      try {
        const v = parseInt(window.localStorage.getItem('ezfy_hq_tab') || '0', 10)
        return isNaN(v) ? 0 : v
      } catch (e) { return 0 }
    },
    // ★ 2026-09-27 为爱发电卡领取（多卡叠加、漏领累加、封顶30天）
    // ★ 2026-10-15 3 秒卡控 + 请求未返回前忽略再点（与后端条件原子更新双保险，防连点刷钻石）
    doLoveCardClaim () {
      if (Date.now() - (this.lastLoveClaimAt || 0) < 3000) { this.notify('操作过于频繁, 请 3 秒后再试'); return }
      this.once('loveCardClaim', () => api.post('/games/ezfy/love-card/claim', {}).then(r => {
        if (r && r.code === 0) {
          if (r.data) {
            if (r.data.love_cards) this.loveCards = r.data.love_cards
            if (this.loveCards.length === 0) this.taskTab = 0
            this.notify(r.data.msg || '已领取')
          }
          // 成功才进冷却（失败/今日无钻石可立即重试）
          this.lastLoveClaimAt = Date.now()
          this.load()
        } else {
          this.notify(r && r.msg ? r.msg : '领取失败')
        }
      }))
    },
    loadWelfare () {
      api.get('/games/ezfy/welfare').then(r => {
        if (r.code === 0) this.welfare = r.data
      })
    },
    loadRank () {
      api.get('/games/ezfy/rank').then(r => {
        if (r.code === 0) this.rankData = r.data
      })
    },
    // ★ 2026-10-14 风云榜：切到该 tab 时总是重新拉最新榜单（修复「点风云榜不刷新」）
    showPvpRank () {
      this.rankTab = 'pvp'
      this.loadPvpRank()
    },
    // ★ 2026-10-14 风云榜：按自然周加载玩家对战周榜（week 不传 = 本周）
    loadPvpRank (week) {
      if (this.pvpLoading) return
      this.pvpLoading = true
      const q = week ? '?week=' + encodeURIComponent(week) : ''
      api.get('/games/ezfy/rank/pvp' + q).then(r => {
        if (r.code === 0) {
          this.pvpRank = r.data
          // ★ 2026-10-16 明细就地展开中：切周后明细跟着切到新周第 1 页
          if (this.pvpRank.week && this.pvpDetail.show) {
            if (this.pvpDetail.week !== this.pvpRank.week) {
              this.pvpDetail.week = this.pvpRank.week
              this.pvpDetail.page = 1
              this.loadPvpDetail()
            }
          }
        }
      }).finally(() => { this.pvpLoading = false })
    },
    // ★ 2026-10-16 风云榜「我的战绩」场次明细：就地展开/收起（收起时不重新拉数据）
    togglePvpDetail () {
      const d = this.pvpDetail
      if (d.show) {
        d.show = false
        return
      }
      if (!this.pvpRank || !this.pvpRank.mine) return
      const week = this.pvpRank.week || ''
      if (d.week !== week || !d.list.length) { // 首次打开/切周后 → 拉第 1 页
        d.week = week
        d.page = 1
        d.list = []
        d.total = 0
        this.loadPvpDetail()
      }
      d.show = true
    },
    // ★ 2026-10-16 拉取当前页场次明细（分页倒序）
    loadPvpDetail () {
      const d = this.pvpDetail
      if (d.loading || !d.week) return
      d.loading = true
      const q = '?week=' + encodeURIComponent(d.week) + '&page=' + d.page + '&size=' + d.size
      api.get('/games/ezfy/rank/pvp/detail' + q).then(r => {
        if (r.code === 0 && r.data) {
          d.list = r.data.list || []
          d.total = r.data.total || 0
          if (r.data.page > 1 && d.list.length === 0) { // 翻过头（数据刚被清）→ 回到末页重拉
            d.page = r.data.page - 1
            this.loadPvpDetail()
          }
        }
      }).finally(() => { d.loading = false })
    },
    // 场次明细翻页（step = ±1）
    pvpDetailPage (step) {
      const d = this.pvpDetail
      const pages = Math.max(1, Math.ceil(d.total / d.size))
      const p = d.page + step
      if (p < 1 || p > pages) return
      d.page = p
      this.loadPvpDetail()
    },
    // ★ 2026-09-28 军衔晋升：声望达标 + 背包宝物足够才能点[晋升]
    canPromote () {
      const m = this.rankData.mine
      if (!m || !m.next || !m.next.treasures) return false
      if (m.prestige < m.next.need) return false
      return m.next.treasures.every(t => t.have >= t.count)
    },
    doPromote () {
      // ★ 2026-10-10 晋升防抖：promote 请求进行中 / 短暂冷却期内忽略再次点击，
      //   防止玩家快速连点一次晋升多级（每次点击都成功晋升一级、宝物连扣）。
      if (this.promoting) return
      if (!this.canPromote()) return
      const next = this.rankData.mine.next
      const req = next.treasures.map(t => t.name + '×' + t.count).join('、')
      if (!window.confirm('确认消耗宝物「' + req + '」晋升至「' + next.name + '」？')) return
      this.promoting = true
      const unlock = () => {
        // 请求完成后短暂冷却，再放开晋升按钮（防止成功后立刻连点下一级）
        setTimeout(() => { this.promoting = false }, 800)
      }
      api.post('/games/ezfy/promote', {}).then(r => {
        if (r.code === 0) {
          alert('恭喜晋升至「' + r.data.rank_name + '」！')
          this.loadRank()
          this.load()
        } else {
          alert(r.msg || '晋升失败')
        }
        unlock()
      }).catch(() => {
        alert('晋升失败，请稍后再试')
        unlock()
      })
    },
    // ★ 军衔名 → 军衔等级 id：优先取军衔表，军衔表未加载时回落内置 20 级（与后端种子一致）
    rankIdByName (name) {
      if (!name) return 1
      const arr = this.rankData.ranks || []
      const hit = arr.find(x => x.name === name)
      if (hit) return hit.id
      const builtin = ['列兵', '上等兵', '下士', '中士', '上士', '军士长', '准尉', '少尉', '中尉', '上尉',
        '大尉', '少校', '中校', '上校', '大校', '少将', '中将', '上将', '大将', '五星上将']
      const i = builtin.indexOf(name)
      return i >= 0 ? i + 1 : 1
    },
    // ★ 军衔星级图标（20 级）：圆角徽章 + 金星，底色按 兵/士/尉/校/将 五大类区分，星数类内递增
    rankIcon (id) {
      const idn = Number(id) || 1
      const tiers = [[1, '#7BA544'], [3, '#6E7B8B'], [7, '#2F6F9F'], [12, '#8A63C9'], [16, '#C0392B']]
      let color = '#999'
      let from = 1
      for (const t of tiers) { if (idn >= t[0]) { color = t[1]; from = t[0] } }
      const stars = Math.max(1, Math.min(5, idn - from + 1))
      let layout
      if (stars === 1) layout = [[10, 10, 4.0]]
      else if (stars === 2) layout = [[6.9, 10, 2.9], [13.1, 10, 2.9]]
      else if (stars === 3) layout = [[5.6, 10, 2.4], [10, 10, 2.4], [14.4, 10, 2.4]]
      else if (stars === 4) layout = [[5.0, 10, 2.0], [8.3, 10, 2.0], [11.7, 10, 2.0], [15.0, 10, 2.0]]
      else layout = [[5.6, 7.6, 2.0], [10, 7.6, 2.0], [14.4, 7.6, 2.0], [7.9, 12.8, 2.0], [12.1, 12.8, 2.0]]
      const star = (cx, cy, R) => {
        const r = R * 0.42
        const p = []
        for (let k = 0; k < 10; k++) {
          const a = (Math.PI / 5) * k - Math.PI / 2
          const rad = (k % 2 === 0) ? R : r
          p.push((cx + rad * Math.cos(a)).toFixed(2) + ',' + (cy + rad * Math.sin(a)).toFixed(2))
        }
        return '<polygon points="' + p.join(' ') + '" fill="#FFD34E"/>'
      }
      const polys = layout.map(s => star(s[0], s[1], s[2])).join('')
      return '<svg class="ezfy-rank-ico" viewBox="0 0 20 20" width="18" height="18" style="vertical-align:-4px;margin-right:4px" role="img">' +
        '<rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="' + color + '"/>' + polys + '</svg>'
    },
    // ★ 2026-10-06 城内军队兵种图标：20 个兵种逐一画剪影（20×20 自绘 SVG，贴切兵种形态）。
    //   阵营不同配色/角标不同一眼可分：同盟(camp=1)=浅底深剪影+金星角标；轴心(camp=2)=深底白剪影+铁十字角标。
    troopIco (t) {
      const id = t ? t.id : 0
      const camp = (t && t.camp) || this.selfInfo.camp || this.profile.camp || 1
      const axis = camp === 2
      const bg = axis ? '#1E2A38' : '#E7ECF1'
      const fg = axis ? '#fff' : '#2A3646'
      // ★ 2026-10-07 图标外观优化（用户：「图标还是丑」）：
      //   ① 外框加一圈细描边 —— 原来只有一块实心浅灰，边界发糊、像「脏方块」；
      //   ② 阵营角标放大（0.32→0.42）并贴角，避免在小画布上糊成噪点。
      const edge = axis ? '#3E5170' : '#C3D0DD'
      const mk = (inner) =>
        '<svg class="ezfy-ico" viewBox="0 0 20 20" role="img">' +
        '<rect x="0.6" y="0.6" width="18.8" height="18.8" rx="3.6" fill="' + bg + '" stroke="' + edge + '" stroke-width="0.7"/>' +
        '<g fill="' + fg + '">' + inner.replace(/\{B\}/g, bg) + '</g>' +
        (axis
          ? '<g fill="#C9CFD6" transform="translate(13.2 2.4) scale(0.42)"><path d="M10 2.5 L14 5.5 L18.5 5.5 L14.5 10 L18.5 14.5 L14 14.5 L10 17.5 L6 14.5 L1.5 14.5 L5.5 10 L1.5 5.5 L6 5.5 Z"/></g>'
          : '<g fill="#D9A320" transform="translate(13.4 2.2) scale(0.42)"><path d="M10 0 L12.4 5.2 L18 5.7 L13.9 9.3 L15.4 14.8 L10 11.9 L4.6 14.8 L6.1 9.3 L2 5.7 L7.6 5.2 Z"/></g>') +
        '</svg>'
      // ★ 2026-10-08 按用户参考图重画：纯色实心剪影、极简加粗、统一侧视朝左、零内部细节，
      //   靠轮廓特征区分兵种（炮塔大小分轻重坦/机翼角度分机型/舰体长度分舰型），小尺寸更清晰。
      switch (id) {
        case 1: // 步兵·侧立持枪士兵(头盔+躯干+跨步双腿+朝左斜杠步枪)
          return mk('<path d="M9.9 2.9a2.5 2.5 0 0 1 4.6 0.9l-0.4 2.1h-4.2Z"/><rect x="7.2" y="8.4" width="5.6" height="2" rx="1"/><rect x="6.2" y="10.3" width="7.6" height="3" rx="1.4"/><path d="M8.6 13l-0.3 3h-2.1l0.2-3.3Z"/><path d="M11.5 12.9l0.2 3.2h2.2l-0.3-3.5Z"/><rect x="5.9" y="15.7" width="2.5" height="1.4" rx="0.7"/><rect x="10.9" y="15.7" width="2.5" height="1.4" rx="0.7"/><path d="M3.2 6.6l6.2 3.1-0.7 1.1-5.5-2.8Z"/>')
        case 2: // 摩托化·边斗摩托车(前/后轮+车身+边斗+边斗轮)
          return mk('<circle cx="4.2" cy="13.3" r="2.2"/><circle cx="11.6" cy="13.3" r="2.2"/><path d="M3.8 11 2.2 5.6h3.6l0.7 3.2h3.4l1.3 1.6H5Z"/><rect x="13.8" y="8.8" width="3.4" height="3.3" rx="0.8"/><circle cx="16" cy="13.6" r="1.6"/><rect x="6.2" y="6.2" width="1.8" height="3.4" rx="0.9"/><circle cx="6.8" cy="5.4" r="1.1"/><path d="M3.4 4.5l2.6 0.9M3 5.2l2.4 0.7"/>')
        case 3: // 卡车·厢式运输车(货厢+驾驶室+底盘+双轮)
          return mk('<rect x="2.8" y="5.2" width="6.6" height="5" rx="0.5"/><path d="M9.6 5.4h3.6l2.8 2.8l0.1 2h-6.5Z"/><rect x="11" y="8.2" width="1.8" height="1.6" rx="0.4"/><rect x="1.8" y="12.4" width="16.4" height="1.3" rx="0.65"/><circle cx="5.4" cy="13.9" r="2"/><circle cx="13.2" cy="13.9" r="2"/>')
        case 4: // 装甲车·轮式装甲运兵车(方体车身+炮塔+三排轮)
          return mk('<path d="M2.4 8.6h15.2l-1.5 1.6H3.9Z"/><path d="M4 6.2l1.8-2h8.4l1.8 2Z"/><path d="M7.2 4.2h2.6L9.1 5.9H6.2Z"/><path d="M7.5 6.3h1.6l0.7-2-.9 0.1Z"/><circle cx="4.6" cy="12.9" r="1.9"/><circle cx="9.2" cy="12.9" r="1.9"/><circle cx="13.8" cy="12.9" r="1.9"/>')
        case 5: // 轻型坦克·紧凑车体+小炮塔+短炮管(朝左)
          return mk('<rect x="2.8" y="11.7" width="14.4" height="2.5" rx="1.25"/><circle cx="4.7" cy="13" r="0.8" fill="{B}"/><circle cx="7.4" cy="13" r="0.8" fill="{B}"/><circle cx="10.1" cy="13" r="0.8" fill="{B}"/><circle cx="12.8" cy="13" r="0.8" fill="{B}"/><path d="M3.7 8h12.6v2H3.7Z"/><path d="M6.2 5.2h3.4v2.6H6.2Z"/><rect x="2" y="5.7" width="2.5" height="1.1" rx="0.55"/>')
        case 6: // 重型坦克·厚重车体+大炮塔+长炮管(朝左, 与大炮塔齐平)
          return mk('<rect x="2.6" y="11.4" width="14.8" height="2.9" rx="1.45"/><circle cx="4.9" cy="12.9" r="0.9" fill="{B}"/><circle cx="7.9" cy="12.9" r="0.9" fill="{B}"/><circle cx="10.9" cy="12.9" r="0.9" fill="{B}"/><circle cx="13.9" cy="12.9" r="0.9" fill="{B}"/><path d="M3.5 8.1h13v2h-13Z"/><path d="M5.8 4.4h6.2l1.9 2.3H4.5Z"/><rect x="0.9" y="5.2" width="3" height="1.3" rx="0.55"/><rect x="0.5" y="6" width="1.1" height="1.4" rx="0.4"/><rect x="1.7" y="8.9" width="3.4" height="1.2" rx="0.6"/>')
        case 7: // 突击炮·无炮塔楔形车体+短粗炮(朝左)
          return mk('<rect x="2.8" y="11.7" width="14.4" height="2.6" rx="1.3"/><path d="M3.6 8.4 8.6 4.3h6.6l2.4 4.1Z"/><rect x="1.2" y="5.3" width="2.4" height="1.5" rx="0.6"/>')
        case 8: // 火箭·卡车搭载倾斜火箭发射架
          return mk('<g transform="translate(2.4 7.2) rotate(-22)"><rect x="0" y="-2" width="8.4" height="1.7" rx="0.5"/><rect x="0" y="0" width="8.4" height="1.7" rx="0.5"/></g><rect x="2.9" y="8.6" width="7.8" height="3.2" rx="0.5"/><path d="M11.4 8.8 h3.6 l2.2 2.4 v0.6 h-5.8 Z"/><circle cx="5.6" cy="13.9" r="1.9"/><circle cx="12.4" cy="13.9" r="1.9"/>')
        case 9: // 侦察机·平直高翼小型机(单发)
          return mk('<path d="M10 1.9 l1.4 2.8 h-2.8 Z"/><rect x="8.7" y="4.7" width="2.6" height="11.3" rx="1"/><rect x="2.4" y="7.6" width="15.2" height="2" rx="0.9"/><rect x="6.2" y="14.4" width="7.6" height="1.5" rx="0.7"/>')
        case 10: // 歼击机·后掠/三角翼箭形(朝左)
          return mk('<path d="M9.9 1.8l1.3 3h-2.6Z"/><rect x="8.4" y="4.8" width="3.2" height="9.6" rx="0.8"/><path d="M3.8 8.2l4 0.9-1 2.4-3.6-1Z"/><path d="M12.6 8.2l4-0.4-0.2 2.8-3.8-0.9Z"/><path d="M9.6 14.8h1v3.6h-1Z"/>')
        case 11: // 轰炸机·宽平直翼+四发引擎重机身
          return mk('<path d="M9.9 2.2 l1.3 2.8 h-2.6 Z"/><rect x="8.8" y="5" width="2.4" height="10.9" rx="1"/><rect x="1.2" y="7.4" width="17.6" height="2.2" rx="1"/><path d="M2.6 6.4 h1.8 v1.6 h-1.8 Z"/><path d="M5.2 6.2 h1.8 v1.7 h-1.8 Z"/><path d="M13 6.2 h1.8 v1.7 h-1.8 Z"/><path d="M15.6 6.4 h1.8 v1.6 h-1.8 Z"/>')
        case 12: // 特种兵·持枪突进步兵
          return mk('<path d="M10 1.2a6 6 0 0 1 6.2 5.6 1 1 0 0 1-1 1.1l-10.4 0a1 1 0 0 1-1-1.1A6 6 0 0 1 10 1.2Z"/><path d="M7.8 8.7V5.2M10 9V3.6M12.2 8.7V5.2" stroke="{B}" stroke-width="0.6" fill="none"/><path d="M8.2 8.9l0.9 4.4M11.8 8.9l-0.9 4.4" stroke="{B}" stroke-width="0.55" fill="none"/><rect x="8.4" y="13.2" width="3.2" height="2.2" rx="1"/><rect x="7.6" y="15.2" width="4.8" height="3" rx="1.2"/><rect x="6.8" y="17.5" width="2" height="1.9" rx="0.9"/><rect x="11" y="17.5" width="2" height="1.9" rx="0.9"/>')
        case 13: // 驱逐舰·流线舰体朝左+炮塔+舰桥+烟囱
          return mk('<path d="M1.9 11.7 H14.8 L13.4 15 H4.7 Z"/><path d="M5 9.4 h2.6 v2.2 H4.6 Z"/><path d="M8.7 6.8 h2.7 v4.8 h-2.7 Z"/><path d="M8.9 4.4 h1.6 v2.4 h-1.6 Z"/><path d="M5.6 9.6 h1.9 v0.9 H5.6 Z"/><path d="M12 9.2 h1.4 v0.8 h-1.4 Z"/>')
        case 14: // 潜艇·艇身朝左+指挥塔+潜望镜
          return mk('<path d="M1.2 10a7.2 2.8 0 0 1 14.6 0 7.2 2.8 0 0 1-14.6 0Z"/><rect x="7.6" y="7.6" width="3.2" height="2.6" rx="0.9"/><path d="M4.8 6.2l0.4 4.6h-2.2l0.2-4Z"/><path d="M13 9.8l1.6-1.1 0.2 3.1-1.2 0.4Z"/><rect x="8.6" y="5.4" width="0.9" height="2.2" rx="0.45"/>')
        case 15: // 战列舰·长舰体朝左+多主炮塔+高舰桥
          return mk('<path d="M1.5 11.4 H16.4 L13.6 15 H4 Z"/><path d="M8.2 6.6 h3 v4.7 h-3 Z"/><path d="M8.6 4.4 h1.6 v2.2 h-1.6 Z"/><path d="M3.9 8.5 h2.3 v2.6 H3.9 Z"/><path d="M2.1 8.9 h1.8 v1 H2.1 Z"/><path d="M12.6 8.6 h2.3 v2.5 h-2.3 Z"/><path d="M12.4 9 h1.3 v1 h-1.3 Z"/>')
        case 16: // 航母·平直甲板+斜角跑道+舰岛(朝左)
          return mk('<path d="M1.4 11.6h17L15.6 15h-12Z"/><rect x="1" y="7.6" width="18" height="3.9" rx="0.8"/><path d="M2.8 10.6l6.2-2.6 5 2.3v-2l-5-2.3-6.2 2.6Z" fill="{B}"/><rect x="12.8" y="5.4" width="2.6" height="2.4" rx="0.7"/><path d="M13.4 3.7h1v1.7h-1Z"/>')
        case 17: // 碉堡·堡垒+射孔
          return mk('<path d="M3.4 13.4 V8.9a6.6 6.6 0 0 1 13.2 0v4.5Z"/><rect x="6" y="13.2" width="8" height="1.1" rx="0.5"/><rect x="8.9" y="9.6" width="2.2" height="1" rx="0.5" fill="{B}"/><rect x="4.4" y="10" width="0.8" height="0.8" rx="0.4" fill="{B}"/><rect x="6.5" y="10" width="0.8" height="0.8" rx="0.4" fill="{B}"/><rect x="12.7" y="10" width="0.8" height="0.8" rx="0.4" fill="{B}"/><rect x="14.8" y="10" width="0.8" height="0.8" rx="0.4" fill="{B}"/>')
        case 18: // 榴弹炮·大轮+盾+高仰角短炮管(朝左)
          return mk('<circle cx="5.6" cy="13.2" r="2.4"/><circle cx="13.6" cy="13.2" r="2.4"/><path d="M8 6.6 5.2 4.4l-1.3 0.9 2.7 2.3Z"/><rect x="5.2" y="7.6" width="2.2" height="5.6" rx="0.5"/><path d="M5.4 13.2 3.6 16h1.6l1.4-2.4Z"/><path d="M5.4 13.2 7.6 16.4H6l-1.2-2.6Z"/>')
        case 19: // 反坦克炮·长细炮管朝左+炮盾+开腿支架+单轮
          return mk('<path d="M2 8.4 H9.4 v1.4 H2 Z"/><path d="M7.9 5.4 h2.6 v4.6 H7.9 Z"/><circle cx="9.9" cy="13.4" r="2.1"/><path d="M11.6 9.4 l3.4 4.2 h-0.7 l-3.1 -3.8 Z"/><path d="M11.6 9.4 l2.2 5.4 h-0.7 l-1.9 -4.8 Z"/>')
        case 20: // 防空炮·双管高射炮朝上+两侧大轮
          return mk('<rect x="8.4" y="7" width="3.4" height="4.4" rx="1"/><path d="M7.6 3.2l4.4 1.5-0.4 0.8-4.4-1.3Z"/><path d="M11.6 2.9l4.6-0.7 0.3 0.8-4.4 1Z"/><path d="M8.6 5l4.4-1.2v1.5l-4.4 1.2Z"/><circle cx="5.6" cy="13" r="2.2"/><circle cx="14.4" cy="13" r="2.2"/>')
        // ★ 2026-10-22 新增「海军轰炸机」(id 21)：俯视后掠翼双发 + 翼下两枚鱼雷，
        //   与轰炸机(11, 宽平直翼四发) 一眼区分；机头朝上，与 9/10/11 同一视角。
        case 21: // 海军轰炸机·后掠翼双发+翼下挂鱼雷
          return mk('<path d="M10 1.4l1.4 3.2h-2.8Z"/><rect x="8.7" y="4.6" width="2.6" height="10.6" rx="1.1"/><path d="M8.8 8.2 2 11.4l0.3 1.6 6.5-1.8Z"/><path d="M11.2 8.2 18 11.4l-0.3 1.6-6.5-1.8Z"/><rect x="6.2" y="14" width="7.6" height="1.6" rx="0.8"/><rect x="2.8" y="13.3" width="3.4" height="1.2" rx="0.6"/><rect x="13.8" y="13.3" width="3.4" height="1.2" rx="0.6"/>')
        default: // 通用兵种兜底
          return mk('<circle cx="10" cy="10" r="4.6"/>')
      }
    },
    // ★ 榜单前三名奖台化：给冠/亚/季军整行上色，超出三名的行不加类
    rankRowCls (rank) {
      if (rank === 1) return 'rank-1'
      if (rank === 2) return 'rank-2'
      if (rank === 3) return 'rank-3'
      return ''
    },
    loadBag () {
      return api.get('/games/ezfy/bag').then(r => {
        if (r.code === 0) {
          this.bagItems = r.data.items
          this.bagTreasures = r.data.treasures || []
          // ★ 2026-10-11 背包「军官装备」tab
          this.bagEquips = r.data.equips || []
          // 出售后列表可能变短 → 页码夹紧，避免停在空白页
          if (this.bagEquipPage > this.bagEquipTotalPages) this.bagEquipPage = this.bagEquipTotalPages
          // ★ 2026-10-11 宝物库存上限（默认 5000）
          this.treasureMax = parseInt(r.data.treasure_max) > 0 ? parseInt(r.data.treasure_max) : 5000
          this.bagOfficers = r.data.officers || []
          this.bagSkills = r.data.skills || []
        }
      })
    },
    // ★ 2026-10-08 宝物出售给系统：打开某件宝物的出售面板
    openSellTreasure (t) {
      this.sellTreasure = t
      this.sellTreasureCount = 1
    },
    // ★ 2026-10-08 确认出售宝物（统一 20万黄金/件，收 10% 手续费 → 实得 18万/件）
    doSellTreasure () {
      const n = parseInt(this.sellTreasureCount) || 0
      if (!this.sellTreasure) return
      if (n <= 0) { this.notify('请填写出售数量'); return }
      if (n > this.sellTreasure.count) { this.notify('超过现有数量'); return }
      api.post('/games/ezfy/exchange/treasure-sell', {
        cfg_id: this.sellTreasure.cfg_id, count: n
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '出售失败'); return }
        this.notify(r.msg || '出售成功')
        this.sellTreasure = null
        this.loadBag()
      })
    },
    // ★ 2026-10-11 背包分栏：道具 / 军官装备 / 宝箱
    setBagTab (t) {
      this.bagTab = t
      this.sellEquip = null
      this.bagEquipPage = 1
      // ★ 2026-10-14 宝箱 tab：切进来就拉一次待开宝箱（商城买的/管理端发放的）
      if (t === 'chest') this.loadChestBag()
    },
    // ★ 2026-10-11 背包出售装备（武器/防具）：打开出售面板
    openSellEquip (e) {
      this.sellEquip = e
      this.sellEquipCount = 1
    },
    // ★ 2026-10-11 确认出售装备：按品质单价 1万~20万/件，黄金入当前城市，**物理删除**
    doSellEquip () {
      const n = parseInt(this.sellEquipCount) || 0
      if (!this.sellEquip) return
      if (n <= 0) { this.notify('请填写出售数量'); return }
      if (n > (this.sellEquip.unworn || 0)) { this.notify('超过未穿戴数量'); return }
      api.post('/games/ezfy/equip/sell', {
        cfg_id: this.sellEquip.cfg_id, count: n
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '出售失败'); return }
        this.notify(r.msg || '出售成功')
        this.sellEquip = null
        this.loadBag()
      })
    },
    // ★ 商城分栏切换（item 道具 / equipment 装备 / set 套装 / chest 宝箱）
    switchMallTab (tab) {
      this.mallTab = tab
      // ★ 装备与套装共用同一份 equipShop 数据（按 set_id 分流），两边都要拉
      if (tab === 'equipment' || tab === 'set') this.loadEquipShop()
      if (tab === 'chest') { this.loadChests(); this.loadChestBag() }
      // 切栏回第 1 页（否则会停在上一栏的分页位置看到空白）
      this.shopPage = 1
    },
    // ★ 2026-10-14 我的待开宝箱：管理端发放的宝箱在背包里，凑齐了来商城「宝箱」页打开
    loadChestBag () {
      api.get('/games/ezfy/chest/bag').then(r => {
        if (r.code === 0) this.chestBag = r.data || { list: [], total: 0 }
      })
    },
    // 开管理端发放的待开宝箱（不花钱，详情同购买开箱）
    openBagChest (b) { this.runOpenBagChest(b, 1) },
    quickOpenBagChest (b, n) {
      // 数量超过背包持有自动封顶
      n = Math.max(1, Math.min(n, b.count || n))
      this.runOpenBagChest(b, n)
    },
    // ★ 2026-10-14 背包宝箱「全部」开
    openBagChestAll (b) { if (b.count > 0) this.runOpenBagChest(b, b.count) },
    // ★ 2026-10-14 背包宝箱自定义数量开（输入非法/超量自动封顶到持有数）
    openBagChestCustom (b) {
      let n = parseInt(this.bagChestCount) || 0
      if (n <= 0) { this.notify('请填写开箱数量'); return }
      if (b.count > 0 && n > b.count) { this.notify('数量超过持有 ' + b.count + ' 个，已按持有数开'); n = b.count }
      this.runOpenBagChest(b, n)
    },
    runOpenBagChest (b, n) {
      if (this.chestOpening) { this.notify('正在开箱，请稍候…'); return } // 锁：上一个还没结束
      if (n <= 0) { this.notify('请选择开箱数量'); return }
      // ★ 2026-10-25 开箱防抖（用户要求）：1 秒账号级卡控（localStorage 多 tab 共享），
      //   与后端 ChestBagOpen 的 1 秒卡控同口径；chestOpening 只管「请求未返回」这段。
      if (!this.opGuard('chestOpenBag', 1000)) return
      this.chestOpening = true
      // 请求期间禁止再开（前端锁 + 后端批量抽奖都防并发重复）
      api.post('/games/ezfy/chest/open-bag', { chest_id: b.chest_id, count: n }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '开箱失败'); return }
        this.notify(r.msg || '开箱成功')
        this.chestResult = (r.data && r.data.results) || []
        // ★ 开箱结果就地展示（复用商城宝箱那套聚合着色）；刷新待开数量 + 背包（开出的道具/装备）
        this.chestLastCost = ''
        this.loadChestBag()
        this.loadBag()
      }).finally(() => { this.chestOpening = false })
    },
    // ★ 2026-10-14 背包宝箱 [说明]：点开/收起奖池
    toggleBagChestDesc (id) {
      this.bagChestDescId = this.bagChestDescId === id ? 0 : id
    },
    // ★ 宝箱（用钻石/黄金买，开箱按权重出套装件）
    loadChests () {
      api.get('/games/ezfy/chest').then(r => {
        if (r.code === 0) {
          this.chestData = r.data
          // ★ 2026-10-09 宝箱内容页刷新兜底：进页时 chestData 还没拉到，拉完补一个默认宝箱
          //   （chestPoolCur 从 chestData 里按 chestPoolId 找，找不到就落到第一个）
          if (!this.chestPoolCur && (r.data.chests || []).length) {
            this.chestPoolId = r.data.chests[0].id
          }
        }
      })
    },
    openChestBuy (ch) {
      this.chestOpen = ch
      this.chestPoolId = ch.id   // 开箱页 [查看奖池] 能回到同一个宝箱的内容页
      this.chestCount = 1
      this.chestPay = ch.price_diamond > 0 ? 'diamond' : 'gold'
      // ★ 跳转到独立开箱详情页确认
      this.chestResult = []
      this.cur = 'chestopen'
    },
    // ★ 2026-10-14 商城宝箱「购买」独立页：进购买页选数量，确认后宝箱进背包（不抽奖）
    goBuyChest (ch) {
      this.chestBuyTarget = ch
      this.buyChestCount = 1
      this.cur = 'chestbuy'
    },
    // ★ 2026-10-14 购买页数量快捷 [1][5][10]
    buyChestSet (n) {
      this.buyChestCount = n
    },
    // ★ 2026-10-14 提交宝箱购买：POST /chest/buy（只买不开，入 ezfy_user_chest 背包）
    submitBuyChest (ch, n) {
      if (!ch) return
      if (!(n > 0)) { this.notify('请选择购买数量'); return }
      const maxN = ch.open_max || n
      n = Math.max(1, Math.min(n, maxN))
      // 并发防抖：连点不会重复下单（1 秒账号级卡控 + 请求未返回忽略，与后端 chest_buy 同口径）
      if (!this.opGuard('chestBuy', 1000)) return
      this.once('chestBuy' + ch.id, () =>
        api.post('/games/ezfy/chest/buy', {
          chest_id: ch.id, count: n, currency: ch.price_diamond > 0 ? 'diamond' : 'gold'
        }).then(r => {
          if (r.code !== 0) { this.notify(r.msg || '购买失败'); return }
          this.notify(r.msg || '购买成功，宝箱已放入背包')
          this.chestBuyTarget = null
          // 回到商城宝箱 tab；刷新余额/库存 + 背包宝箱数（买到的进背包）
          this.mallTab = 'chest'
          this.cur = 'mall'
          this.loadChests()
          this.loadChestBag()
        }))
    },
    // ★ 2026-10-13 快捷开箱：商城宝箱列表直接 [开1]/[开5]/[开10]，数量超过单次上限自动封顶，
    //   开完结果就地展示在商城宝箱 tab（不用再跳独立开箱页看列表）
    quickOpenChest (ch, n) {
      if (ch.sold_out) { this.notify('该宝箱已售罄'); return }
      this.chestOpen = ch
      this.setChestCount(n)
      this.doOpenChest(ch)
    },
    // ★ 2026-10-13 商城宝箱行：点宝箱名展开/收起「可开出」预览（不跳奖池页也知道能开出啥）
    toggleChestDesc (ch) {
      this.chestDescId = (this.chestDescId === ch.id) ? 0 : ch.id
    },
    // ★ 2026-10-13 奖池预览摘要：同名同为档合并计数、高档排前，只取前 max 项
    chestPoolTop (pool, max) {
      const rank = { '传说': 0, '传奇': 0, '特殊': 0, '史诗': 1, '高级': 1, '稀有': 2, '中级': 2, '优良': 3, '普通': 3 }
      const byK = {}
      for (const p of (pool || [])) {
        const q = p.quality || '普通'
        const k = (p.name || '?') + '|' + q
        if (!byK[k]) byK[k] = { name: p.name || '?', quality: q, n: 0 }
        byK[k].n += (p.count || 1)
      }
      return Object.values(byK).sort((a, b) => {
        const ra = rank[a.quality] === undefined ? 9 : rank[a.quality]
        const rb = rank[b.quality] === undefined ? 9 : rank[b.quality]
        return (ra - rb) || (a.name < b.name ? -1 : 1)
      }).slice(0, max || 6)
    },
    doOpenChest (ch) {
      const n = parseInt(this.chestCount) || 0
      if (n <= 0) { this.notify('请填写开箱数量'); return }
      if (n > ch.open_max) { this.notify('单次最多开 ' + ch.open_max + ' 个'); return }
      const cur = (ch.price_diamond > 0 && ch.price_gold > 0) ? this.chestPay : (ch.price_diamond > 0 ? 'diamond' : 'gold')
      // ★ 2026-10-25 开箱防抖（用户要求）：1 秒账号级卡控（localStorage 多 tab 共享），
      //   与后端 ChestOpen 的 1 秒卡控同口径；下面的 once 只管「请求未返回」这段。
      if (!this.opGuard('chestOpen', 1000)) return
      // ★ 2026-10-08 连点防抖：一次点击 = 一次开箱（后端已加条件原子扣款/扣库存）
      this.once('chestOpen' + ch.id, () => api.post('/games/ezfy/chest/open',
        { chest_id: ch.id, count: n, currency: cur }).then(r => {
          if (r.code !== 0) { this.notify(r.msg || '开箱失败'); return }
          this.notify(r.msg || '开箱成功')
          this.chestResult = (r.data && r.data.results) || []
          this.chestOpen = null
          // ★ 2026-10-13 开箱扣款已实时回账：结果行余额直接取本次响应，不等 loadChests 回来
          //   （loadChests 有缓存/延迟，会导致「钻石没局部更新」）
          // ★★ 2026-10-13 修「开宝箱后商城标题/道具页钻石余额不刷新」：
          //   商城标题用的是 mallDiamond、装备商城用的是 equipShop.diamond，
          //   这两个只由 loadMall()/loadEquipShop() 赋值，开箱这条路径原来一个都没调
          //   → 余额一直停在开箱前的旧值。这里统一用后端返回的扣款后余额就地回填。
          if (r.data) {
            if (r.data.gold !== undefined) {
              this.chestData.gold = r.data.gold
              this.equipShop.gold = r.data.gold
            }
            if (r.data.diamond !== undefined) {
              this.chestData.diamond = r.data.diamond
              this.equipShop.diamond = r.data.diamond
              this.mallDiamond = r.data.diamond
            }
          }
          this.loadBag()
          this.load()
          this.loadChests()
          // ★ 2026-10-13 简约化：开箱结果回「商城 → 宝箱」tab 就地聚合展示（品质着色 + 花费 + 余额），
          //   玩家不用再跳独立开箱页看结果列表
          const price = ch.price_diamond > 0 ? ch.price_diamond : ch.price_gold
          this.chestLastCost = (price * n) + (cur === 'diamond' ? '钻' : this.resNames.gold)
          this.cur = 'mall'
          this.mallTab = 'chest'
        }))
    },
    // ★ 装备商城（套装件，黄金/钻石购买）
    loadEquipShop () {
      this.loadEquipSets()
      api.get('/games/ezfy/equipshop').then(r => {
        if (r.code === 0) this.equipShop = r.data
      })
    },
    openEquipBuy (p) {
      this.equipShopBuy = p
      this.equipShopCount = 1
      // 散件统一钻石结算（定价 100~500 钻）；只有管理端把钻石价清 0 时才回落黄金
      this.equipShopPay = p.price_diamond > 0 ? 'diamond' : 'gold'
      // ★ 跳转到独立购买详情页确认
      this.cur = 'equipbuy'
    },
    doBuyEquip (p) {
      const n = parseInt(this.equipShopCount) || 0
      if (n <= 0) { this.notify('请填写购买数量'); return }
      const cur = p.price_diamond > 0 ? 'diamond' : 'gold'
      // ★ 2026-10-08 连点防抖：一次点击 = 一次购买（后端已加条件原子扣款/扣库存）
      this.once('equipBuy' + p.id, () => api.post('/games/ezfy/equipshop/buy',
        { cfg_id: p.id, count: n, currency: cur }).then(r => {
          if (r.code !== 0) { this.notify(r.msg || '购买失败'); return }
          this.notify(r.msg || '购买成功')
          this.equipShopBuy = null
          this.load()
          this.loadBag()
          // ★ 购买成功后返回商城（go('mall') 会自动刷新装备商城）
          this.go('mall')
        }))
    },
    // ★ 拉商城数据。resetPage = true 时才回到第 1 页（进商城页签时用）。
    //   买完道具的刷新**不能**重置页码 —— 否则玩家在第 3 页买个东西就被弹回第 1 页。
    loadMall (resetPage) {
      api.get('/games/ezfy/mall').then(r => {
        if (r.code === 0) {
          this.mallItems = r.data.items
          // ★ 分类页签 + 钻石余额（钻石只能管理端充值）
          this.mallCatsList = r.data.categories || []
          this.mallDiamond = r.data.diamond || 0
          // ★ 单次购买上限（管理端可配，2026-10-22 默认 999）
          this.mallBuyMax = parseInt(r.data.buy_max) > 0 ? parseInt(r.data.buy_max) : 999
          // ★ 2026-10-11 背包单件道具上限（默认 5000）
          this.mallBagMax = parseInt(r.data.bag_max) > 0 ? parseInt(r.data.bag_max) : 5000
          if (this.mallCat && this.mallCatsList.indexOf(this.mallCat) < 0) this.mallCat = ''
          if (resetPage) this.mallPage = 1
          else if (this.mallPage > this.mallTotalPages) this.mallPage = this.mallTotalPages
        }
      })
    },
    loadExchange () {
      api.get('/games/ezfy/exchange', {
        params: {
          page: this.exchangePage, size: this.exchangeSize,
          mpage: this.exchangeMPage, msize: this.exchangeMSize,
          es_type: this.exFilter
        }
      }).then(r => {
        if (r.code === 0) {
          this.exchangeOrders = r.data.orders || []
          this.exchangeTotal = r.data.total || 0
          this.exchangePage = r.data.page || 1
          this.exchangeSize = r.data.size || 10
          this.exchangeMine = r.data.mine || []
          this.exchangeMTotal = r.data.mtotal || 0
          this.exchangeMPage = r.data.mpage || 1
          this.exchangeMSize = r.data.msize || 10
          this.exchangeGold = r.data.gold
          // ★ 2026-10-04 玩家挂单上限（交易所等级×2，ExchangeSell 同口径卡控）
          if (r.data.sell_max != null) this.exchangeSellMax = r.data.sell_max
          // ★ 2026-09-30 向系统出售资源：读回回收比例 / 手续费 / 黄金上限
          if (r.data.sys_sell_ratio) this.sysSellRatio = r.data.sys_sell_ratio
          if (r.data.sys_sell_fee) this.sysSellFee = r.data.sys_sell_fee
          // ★ 2026-10-22 每日每种资源回收上限 + 今日已售（0 = 不限）
          if (r.data.sys_sell_daily_limit != null) this.sysSellDailyLimit = Number(r.data.sys_sell_daily_limit) || 0
          if (r.data.sys_sell_used) this.sysSellUsed = r.data.sys_sell_used
          if (r.data.gold_max != null) this.goldMax = r.data.gold_max
        }
      })
    },
    // ★ 2026-09-24 卖家挂单类别检索: 切换类别回到第一页再加载
    onExFilter () {
      this.exchangePage = 1
      this.loadExchange()
    },
    // ★ 2026-10-03 接口懒加载：首页头部只显示「我的军团名」，仅需 /corps/list；
    //   成员/聊天等数据进军团页（go('corps') / switchCorpsTab）时由 loadCorps() 再拉，
    //   避免进游戏就无条件打 /corps/members 和 /corps/chats。
    loadCorpsSummary () {
      api.get('/games/ezfy/corps/list').then(r => {
        if (r.code === 0) {
          this.corpsList = r.data.corps
          this.myCorps = r.data.my_corps
          // ★ 2026-09-30 入团审核开关（我的军团）
          this.myCorpsNeedReview = (r.data.my_corps && r.data.my_corps.need_review) ? 1 : 0
          // 未入团时：从列表里找「我是否已申请且待审」的军团状态
          if (!this.myCorps) this.myApplyStatus = 0
        }
      })
    },
    loadCorps () {
      api.get('/games/ezfy/corps/list').then(r => {
        if (r.code === 0) {
          this.corpsList = r.data.corps
          this.myCorps = r.data.my_corps
          // ★ 2026-09-30 入团审核开关（我的军团）
          this.myCorpsNeedReview = (r.data.my_corps && r.data.my_corps.need_review) ? 1 : 0
          // 未入团时：从列表里找「我是否已申请且待审」的军团状态
          if (!this.myCorps) this.myApplyStatus = 0
          // 团长：进入信息 tab 时拉待审申请
          if (this.isLeader && this.corpsTab === 'info') this.loadCorpsApplies()
        }
      })
      api.get('/games/ezfy/corps/members').then(r => {
        if (r.code === 0) {
          this.corpsMembers = r.data.members
          // ★ 后端下发「我在军团的职位」，副团长也能发军团邮件
          this.myCorpsTitle = r.data.my_title || ''
          // ★ 2026-09-25 军团信息 tab 显示军团总积分（新字段 corps_points）
          this.corpsPoints = r.data.corps_points || 0
        }
      })
      api.get('/games/ezfy/corps/chats').then(r => {
        if (r.code === 0) this.corpsChats = r.data.chats
      })
    },
    loadOrders () {
      api.get('/games/ezfy/orders').then(r => {
        if (r.code === 0) this.orders = r.data.orders
      })
    },
    loadNotices () {
      api.get('/games/ezfy/notices').then(r => {
        if (r.code === 0) {
          this.notices = r.data.notices
          // ★ 公告列表每次重新加载都回到第 1 页（否则刷新后可能停在超出范围的空页）
          this.noticePage = 1
          // ★ 首页外露公告由管理端配置条数（默认 1 条），后端直接下发 home_notices
          this.homeNotices = r.data.home_notices || []
        }
      })
    },
    loadWilds () {
      api.get('/games/ezfy/city/wildfull').then(r => {
        if (r.code === 0) {
          this.wildlands = r.data.wildlands
          this.occupies = r.data.occupies
          // ★ 2026-10-07 赎城：占领方在「占领城市」面板处理赎回请求
          this.ransoms = r.data.ransoms || []
        }
      })
    },
    loadTargets () {
      // ★ 2026-10-28 兵种战斗配置是**城市维度**的（每城一套）：显式带上当前城，
      //   不依赖后端「当前城」默认值，切城后立刻看到该城自己的配置。
      const cid = this.city && this.city.id ? this.city.id : 0
      const url = '/games/ezfy/targets' + (cid ? '?city_id=' + cid : '')
      return api.get(url).then(r => {
        // 先按兵种表建全默认值, 再覆盖服务端已保存的配置
        const cfg = {}
        for (const t of (this.troopsData.cfgs || [])) {
          cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1, defCount: '' }
        }
        if (r.code === 0) {
          for (const t of r.data.targets) {
            cfg[t.troop_id] = {
              atk: t.atk_target_troop, atkMove: t.atk_move,
              def: t.def_target_troop, defMove: t.def_move,
              // 0 = 全部（默认）→ 输入框留空显示占位「全部」；>0 = 参与防御的兵力上限
              defCount: t.def_count > 0 ? t.def_count : ''
            }
          }
        }
        this.targetCfg = cfg
      })
    },
    loadRes () {
      api.get('/games/ezfy/resources').then(r => {
        if (r.code === 0) {
          this.city = r.data.city
          this.resDetail = r.data.calc[this.resType]
        }
      })
    },
    // 地图数据统一入口(loadMap/moveMap/jumpTo 共用)
    applyMap (d) {
      this.mapCells = d.cells
      this.mapCx = d.cx
      this.mapCy = d.cy
      this.eliteCell = d.elite && d.elite.x ? d.elite : null
    },
    loadMap () {
      // ★ 必须带 r: 不带的话后端用默认半径返回, 格子数与前端切行用的 mapR 不一致,
      //   整张网格会错位(本城就不在中心格了)
      const seq = ++this.mapSeq
      api.get('/games/ezfy/map?r=' + this.mapR).then(r => {
        if (r.code === 0 && seq === this.mapSeq) this.applyMap(r.data)
      })
    },
    // ★ 2026-09-25 用户反馈「[返回地图] 怎么都回到初始的地图页，我都移动好多次了」：
    //   地图视野中心(mapCx/mapCy)一直存在内存里，进详情页再回来时要按**离开时的中心**刷新，
    //   而不是重新 loadMap() 回本城；只有这次会话还没加载过地图(如刷新后直接进地图/详情)才回本城。
    //   (「回到本城」按钮仍然走 loadMap()，保持原样)
    backToMap () {
      if (this.mapCells && this.mapCells.length) this.jumpTo(this.mapCx, this.mapCy)
      else this.loadMap()
    },
    moveMap (dx, dy) {
      this.jumpTo(this.mapCx + dx, this.mapCy + dy)
    },
    // ---- 地图坐标查找 / 收藏列表(复刻原版地图页) ----
    jumpTo (x, y) {
      const seq = ++this.mapSeq
      api.get('/games/ezfy/map?x=' + x + '&y=' + y + '&r=' + this.mapR)
        .then(r => { if (r.code === 0 && seq === this.mapSeq) this.applyMap(r.data) })
    },
    // ★ 2026-10-04 附属野地坐标点击 → 进地图并居中到该坐标（跳坐标只打一次地图接口）
    openMapAt (x, y) {
      this.cur = 'map'
      this.loadStars()
      // ★ 2026-10-04 /view 瘦身：地图页「占领野地」面板数据随进入一并懒加载
      this.loadWilds()
      this.jumpTo(x, y)
    },
    doJump () {
      const x = parseInt(this.jumpX)
      const y = parseInt(this.jumpY)
      if (!x || !y || x < 1 || x > 500 || y < 1 || y > 500) {
        this.notify('请输入 1~500 之间的横纵坐标')
        return
      }
      this.jumpTo(x, y)
    },
    loadStars () {
      // ★ 2026-10-25 返回 Promise，调用方可按需等它回来
      return api.get('/games/ezfy/map/stars').then(r => {
        if (r.code === 0) this.mapStars = r.data.stars || []
        return this.mapStars
      })
    },
    toggleStars () {
      this.showStars = !this.showStars
      if (this.showStars) this.loadStars()
    },
    // ★★ 2026-10-25 「收藏」按钮 = 一个状态开关，名字严格跟着收藏状态走：
    //   · 未收藏 → 按钮「收藏」，点它弹备注名 → 收藏成功 → 变「已收藏」
    //   · 已收藏 → 按钮「已收藏」，点它**按坐标**取消收藏 → 变回「收藏」
    //   ★★ 状态**只认收藏列表**（/map/stars 全量返回）：这样「按钮文字」与玩家点开
    //     「收藏列表」看到的内容**永远一致**，可以自己核对；不再读详情接口的 is_starred 快照
    //     （快照是异步的，切换格子时容易串味 → 就是「没收藏却显示已收藏」的根源）。
    async addStar () {
      if (!this.selCell) return
      const x = this.selCell.x
      const y = this.selCell.y
      if (this.isCellStarred) {
        // ★ 已收藏 → 按坐标取消（不依赖收藏列表里那条记录的 id，点了必定有结果）
        api.post('/games/ezfy/map/stars/delete-xy', { x: x, y: y }).then(r => {
          if (r.code === 0) {
            // 本地先摘掉 → 按钮立刻变「收藏」，随后拉列表校正
            this.mapStars = this.mapStars.filter(s => !(Number(s.x) === Number(x) && Number(s.y) === Number(y)))
            this.loadStars()
            this.notify(r.msg || '已取消收藏')
            return
          }
          // ★ 老后端没有 delete-xy（404/参数错误）→ 回落老的按 id 删，保证功能不哑
          const hit = this.mapStars.find(s => Number(s.x) === Number(x) && Number(s.y) === Number(y))
          if (hit && hit.id) { this.delStar(hit); return }
          this.notify(r.msg || '取消收藏失败')
        })
        return
      }
      // 备注名用「地形名(等级)」, 坐标由列表模板统一拼, 别在这里重复带上
      // ★ 2026-09-25：地图格子上城市统一显示「城市」，收藏备注名要用具体城市名，否则收藏列表分不清
      const def = this.selCell.area_type === 3
        ? ((this.selCell.name || '城市') + (this.selCell.owner ? '(' + this.selCell.owner + ')' : ''))
        : this.cellText(this.selCell)
      const name = await this.ask('备注名（最多16字）', { input: true, value: def })
      if (name === null) return
      api.post('/games/ezfy/map/stars', { x: x, y: y, name: name }).then(r => {
        if (r.code === 0) {
          // ★ 2026-10-25 用户要求：「收藏列表」只在点它时展开/收起，**收藏成功后不要自动展开**
          //   （原来这里 showStars = true，收藏一次列表就被顶出来，玩家以为点了别的东西）
          // 本地先插一条（临时 id=0）→ 按钮立刻变「已收藏」，随后拉列表换成真实 id
          if (!this.mapStars.some(s => Number(s.x) === Number(x) && Number(s.y) === Number(y))) {
            this.mapStars = this.mapStars.concat([{ id: 0, x: x, y: y, name: name }])
          }
          this.loadStars()
          this.notify(r.msg || '已收藏')
        } else {
          // 「该坐标已收藏」= 库里其实有、本地没同步 → 拉列表让它显示「已收藏」，下次点击就是取消
          this.loadStars()
          this.notify(r.msg || '收藏失败')
        }
      })
    },
    delStar (s) {
      api.post('/games/ezfy/map/stars/delete', { id: s.id }).then(r => {
        if (r.code === 0) {
          this.mapStars = this.mapStars.filter(it => it.id !== s.id)
          this.loadStars()
        }
      })
    },
    // ---- 城市迁移(复刻 city/cityHallMove.html) ----
    // ★ 第十二轮：区域 = 大洲；迁城消耗道具（迁城计划/高级迁城计划/沿海迁城计划）
    loadMoveInfo () {
      api.get('/games/ezfy/city/move').then(r => {
        if (r.code === 0) {
          this.moveInfo = r.data
          this.moveItems = r.data.items || []
          const def = r.data.default_continent || 1
          if (r.data.areas && r.data.areas.length) {
            // 默认选中「欧洲」（后端下发的默认洲）；没有就取第一个
            const hit = r.data.areas.some(a => a.id === def)
            const pick = hit ? def : r.data.areas[0].id
            this.moveContinent = pick
            this.moveContinentSea = pick
            this.moveArea = pick
          }
        }
      })
    },
    async doMoveCity (type) {
      const body = { type: type }
      if (type === 'low') {
        body.continent_id = this.moveContinent
      } else if (type === 'high') {
        const x = parseInt(this.moveX)
        const y = parseInt(this.moveY)
        if (!x || !y) { this.notify('请输入横纵坐标'); return }
        body.x = x; body.y = y
      } else {
        // 沿海迁城：填了坐标就按坐标迁，没填就按所选大洲随机找沿海平原
        const x = parseInt(this.moveX2)
        const y = parseInt(this.moveY2)
        if (x && y) {
          body.x = x; body.y = y
        } else {
          body.continent_id = this.moveContinentSea
        }
      }
      const label = type === 'low' ? '迁城计划' : (type === 'high' ? '高级迁城计划' : '沿海迁城计划')
      const kind = (this.moveItems || []).find(i => i.code === type) || {}
      const have = kind.count || 0
      if (have < 1) {
        this.notify('背包里没有【' + label + '】，请先到商城购买')
        return
      }
      const contName = this.areaName(type === 'sea' ? this.moveContinentSea : this.moveContinent)
      const where = type === 'low' ? '迁入【' + contName + '】'
        : (type === 'sea' && !parseInt(this.moveX2) ? '迁入【' + contName + '】的沿海平原' : '迁到指定坐标')
      if (!await this.ask('确认使用【' + label + '】×1 ' + where + '吗？（当前持有 ' + have + ' 个）')) return
      api.post('/games/ezfy/city/move', body).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.load()
          this.loadMoveInfo()
        } else this.notify(r.msg || '迁城失败')
      })
    },
    areaName (id) {
      const a = (this.moveInfo.areas || []).find(x => x.id === id)
      return a ? a.name : '—'
    },
    // ---- 调整生产(复刻 city/sourceSet.html) ----
    loadProduce () {
      api.get('/games/ezfy/city/produce').then(r => {
        if (r.code === 0) {
          this.rateFood = r.data.rate_food
          this.rateSteel = r.data.rate_steel
          this.rateOil = r.data.rate_oil
          this.rateRare = r.data.rate_rare
        }
      })
    },
    doSaveRate () {
      const vals = [this.rateFood, this.rateSteel, this.rateOil, this.rateRare]
      for (const v of vals) {
        const n = parseInt(v)
        if (isNaN(n) || n < 0 || n > 100) { this.notify('开工率需在 0~100 之间'); return }
      }
      api.post('/games/ezfy/city/produce', {
        rate_food: parseInt(this.rateFood), rate_steel: parseInt(this.rateSteel),
        rate_oil: parseInt(this.rateOil), rate_rare: parseInt(this.rateRare)
      }).then(r => {
        this.alert(r, '生产比例已调整')
        if (r.code === 0) this.loadProduce()
      })
    },
    // ---- 建筑操作 ----
    // ---- 建造页(复刻 preCreateMilitary / preCreateSource) ----
    openBuildPre (zone) {
      this.buildZone = zone || (this.cur === 'builds' ? 's' : 'm')
      this.buildSel = null
      // ★ 2026-10-04 /view 瘦身：可建造池已移到 /buildings，进建造页前懒加载
      this.loadBuildings()
      this.cur = 'buildpre'
    },
    openBuildDetail (b) {
      this.buildSel = b
    },
    // 训练一键加速(本城 / 所有城市)
    doSpeedTrainAll () {
      // ★★★ 2026-10-15 多 tab 防刷：统一账号级 3 秒共享卡控（localStorage 同锁，所有 tab 共用一个冷却窗口）
      if (!this.opGuard('speedTrainAll', 3000)) return
      this.once('speedTrain', () =>
        api.post('/games/ezfy/troops/speed-all', { all_city: false }).then(r => {
          if (r.code === 0) {
            this.notify(r.msg)
            this.load()
            this.loadBuildings() // ★ 2026-10-04 加速后建筑列表保持最新（/view 已不下发 buildings）
          } else this.notify(r.msg)
        })
      )
    },
    async doSpeedTrainAllCity () {
      if (!await this.ask('确定对所有城市的训练队列一键加速吗?(按剩余时间消耗' + this.resNames.gold + ')')) return
      // ★★★ 2026-10-15 多 tab 防刷：确认通过后才占用 3 秒共享卡控（误点取消不占冷却）
      if (!this.opGuard('speedTrainAll', 3000)) return
      this.once('speedTrain', () =>
        api.post('/games/ezfy/troops/speed-all', { all_city: true }).then(r => {
          if (r.code === 0) {
            this.notify(r.msg)
            this.load()
            this.loadBuildings() // ★ 2026-10-04 加速后建筑列表保持最新（/view 已不下发 buildings）
          } else this.notify(r.msg)
        })
      )
    },
    doBuild (b) {
      // ★ 建造成功后跳回对应分区（资源区→资源区、军事区→军事区），并刷新建筑列表
      // ★ 用户反馈「连点会出现多条」→ 防抖：一次点击只下达一条建造命令
      this.once('build', () =>
        api.post('/games/ezfy/build', { building_id: b.building_id || b.bid }).then(r => {
          if (r.code === 0) {
            this.load()
            this.go(this.buildZone === 'm' ? 'buildm' : 'builds')
          } else this.notify(r.msg || '建造失败', 'error')
        })
      )
    },
    doUpgrade (b) {
      api.post('/games/ezfy/building/upgrade', { record_id: b.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : '建筑已开始升级', type: 'ok' }
          this.load()
          this.loadBuildings() // ★ 2026-10-04 /view 瘦身：建筑列表已由 /buildings 懒加载提供
        } else this.inlineTip = { bid: b.id, text: r.msg || '升级失败', type: 'error' }
      })
    },
    doMaxLevel (b) {
      // ★ 2026-09-25 用户纠正「一键9级 = 一键升级到 9 级，而不是升级满」：
      //   按钮文案是「一键{{max_level-1}}级」，就把目标等级一起发给后端（target_level），
      //   后端按目标级结算资源/图纸并停在那一级（不越过 9→10 这道要建筑图纸的坎）。
      // ★ 2026-09-30 一键固定只到 9 级（去掉「一键11级/一键12级」）。
      const target = 9
      api.post('/games/ezfy/building/max-level', { record_id: b.id, target_level: target }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : ('已一键升级到' + target + '级'), type: 'ok' }
          this.load()
          this.loadBuildings() // ★ 2026-10-04 /view 瘦身：建筑列表已由 /buildings 懒加载提供
        } else this.inlineTip = { bid: b.id, text: r.msg || '升级失败', type: 'error' }
      })
    },
    // ★ 2026-09-25 「拆除询问下玩家是否拆除，玩家可能按错了」→ 先二次确认再拆
    //   （拆除是逐级降级，降到 0 级才彻底移除，提示里把这个后果说清楚）
    async doDeleteBuilding (b) {
      const nm = (b && b.name) || '该建筑'
      const lv = b && b.level ? ('' + b.level + '级') : ''
      const ok = await this.ask('确定拆除「' + nm + '」' + lv + '吗？\n拆除是逐级降级（每次降 1 级），降到 0 级才彻底移除，且不退还建造资源。')
      if (!ok) return
      api.post('/games/ezfy/building/delete', { record_id: b.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : '建筑已拆除', type: 'ok' }
          this.load()
          this.loadBuildings() // ★ 2026-10-04 /view 瘦身：建筑列表已由 /buildings 懒加载提供
        } else this.inlineTip = { bid: b.id, text: r.msg || '拆除失败', type: 'error' }
      })
    },
    // ============ 加速道具（建筑 3 / 训练 4 / 科技 5）============
    // ★ 2026-09-26 修复「加速道具买完实际使用不生效」：
    //   根因是**训练页 / 科技页的加速入口根本没接这些道具** ——
    //   训练页只有 [训练一键加速]（花黄金），科技页的 [加速] 调 /techs/speed
    //   （免费减 10 分钟，minutes 还能由前端随便传），于是商城买的
    //   「训练加速30分钟/2小时」「科技加速30分钟/2小时」买完在游戏里无处可用。
    //   现在三个页面统一口径：按**背包里实际拥有**的档位渲染 [加速30分钟]/[加速2小时]，
    //   点哪档就用哪档（顺带解决「买了 2 小时却自动用了 30 分钟」的困惑）。
    accItems (type) {
      return (this.bagItems || [])
        .filter(i => i.item_type === type && i.count > 0)
        .sort((a, x) => (a.param1 || 0) - (x.param1 || 0))
    },
    // 加速道具的档位名：120 →「2小时」，30 →「30分钟」；百分比档(24/25/26) →「30%」
    accLabel (a) {
      const m = (a && a.param1) || 0
      if (a && (a.item_type === 24 || a.item_type === 25 || a.item_type === 26)) return m + '%'
      if (m >= 60 && m % 60 === 0) return (m / 60) + '小时'
      return m + '分钟'
    },
    // 统一的「用道具加速」调用（cfg_id 走 /bag/use，后端按 item_type 决定加速目标）
    //   targetId：指定目标（建筑升级记录 id / 训练队列 id），0/undefined = 后端自动挑最早一条
    //   done：成功后的回调（刷新对应页面的数据）
    async useSpeedItem (item, type, targetId, done) {
      await this.loadBag()
      const it = (item && item.cfg_id) ? item : this.accItems(type)[0]
      if (!it) {
        this.notify('没有对应的加速道具，去商城购买后再加速', 'error')
        return
      }
      const body = { cfg_id: it.cfg_id, count: 1, city_id: this.city.id }
      if (targetId) body.record_id = targetId
      // ★★★ 2026-10-15 多 tab 防刷：训练/科技加速道具统一账号级 3 秒共享卡控（localStorage 同锁）
      if (!this.opGuard('speedItem', 3000)) return
      api.post('/games/ezfy/bag/use', body).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '加速成功', 'ok')
          this.loadBag()
          if (typeof done === 'function') done()
        } else this.notify(r.msg || '加速失败', 'error')
      })
    },
    async doSpeedBuilding (b, item) {
      // ★ 建筑加速必须消耗「建筑加速道具」(item_type=3)，没有道具则无法加速
      const bid = b && b.id
      await this.loadBag()
      const it = (item && item.cfg_id) ? item : this.accItems(3)[0]
      if (!it) {
        this.inlineTip = { bid, text: '没有建筑加速道具，去商城购买后再加速', type: 'error' }
        return
      }
      // ★ 带上 city_id：多城时「不传 city_id 走当前城」容易和玩家正在看的城错位
      // ★ 2026-09-27 带上 record_id：点哪条建筑就减哪条，避免后端总是挑最早结束的一条
      // ★★★ 2026-10-15 多 tab 防刷：建筑加速道具统一账号级 3 秒共享卡控（localStorage 同锁）
      if (!this.opGuard('speedItem', 3000)) return
      this.once('speedBuilding', () =>
      api.post('/games/ezfy/bag/use', { cfg_id: it.cfg_id, count: 1, city_id: this.city.id, record_id: bid }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid, text: (r.data && r.data.msg) ? r.data.msg : '加速成功', type: 'ok' }
          this.load()
          this.loadBuildings() // ★ 2026-10-04 /view 瘦身：建筑列表已由 /buildings 懒加载提供
          this.loadBag()
        } else this.inlineTip = { bid, text: r.msg || '加速失败', type: 'error' }
      })
      )
    },
    // ★ 2026-09-27 严重漏洞修复：取消建筑升级**零退还**（资源与建筑图纸不退，建筑保留当前等级）。
    //   旧逻辑「全额退还」会让人「一键升满→取消」反复刷资源/图纸。
    async doCancelUpgrade (b) {
      if (!b || b.status === 0) return
      const bid = b.id
      if (!await this.ask('确定取消「' + (b.name || '该建筑') + '」的升级吗？\n' +
        '已消耗的资源与建筑图纸将不会退还，建筑保留当前等级。')) return
      api.post('/games/ezfy/building/cancel', { record_id: bid, city_id: this.city.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid, text: (r.data && r.data.msg) ? r.data.msg : '已取消升级', type: 'ok' }
          this.load()
          this.loadBuildings() // ★ 2026-10-04 /view 瘦身：建筑列表已由 /buildings 懒加载提供
          this.loadBag()
        } else this.inlineTip = { bid, text: r.msg || '取消失败', type: 'error' }
      })
    },
    // 训练加速（训练页队列行 / 建筑页底部的「训练加速道具」入口）
    // ★ 2026-09-27 修复「没有按指定目标扣减」：训练页队列行带上 q.id，点哪条减哪条；
    //   建筑页底部入口 q 为 null → record_id=0，后端自动挑最早的一条
    doSpeedTrain (q, item) {
      this.useSpeedItem(item, 4, q ? q.id : 0, () => { this.loadTroops(); this.load() })
    },
    // 建筑名后的特殊入口(复刻原版 militaryIndex 里各建筑指向的功能页)
    // ★ 2026-10-16 玩家口径：这几个建筑**保持原样** —— 名字直接进下面这个功能页，
    //   不加任何额外按钮（所以「建筑简介」只挂在原本点不动的建筑上，见 EzfyBuild.vue）。
    bEntry (bid) {
      const map = {
        1: { label: '市政厅', cur: 'cityhall' },
        2: null,
        7: { label: '城防', cur: 'defence' },
        8: { label: '科技', cur: 'techs' },
        9: { label: '军校', cur: 'acade', tab: 'search' },
        10: { label: '参谋部', cur: 'acade', tab: 'officer' },
        11: { label: '交易所', cur: 'exchange' },
        12: { label: '仓库调配', cur: 'wareset' },
        13: { label: '司令部', cur: 'hq' },
        14: { label: '军工厂', cur: 'factory' },
        15: { label: '联络中心', cur: 'liaison' }
      }
      return map[bid] || null
    },
    // ★ 2026-10-16 军事区/资源区：点建筑名 → 跳「建筑详情」页（简介 + 当前级效果 + 下一级效果）。
    //   只对**原本点不动的建筑**生效；有功能页的建筑（军工厂/司令部…）名字仍直接进功能页。
    openBuildInfo (b) {
      this.buildInfoId = b.id
      this.go('bdetail')
    },
    goEntry (bid) {
      const e = this.bEntry(bid)
      if (!e) return
      if (e.tab) this.acadeTab = e.tab
      this.go(e.cur)
    },
    // 召集冷却毫秒数 → 提示文案（3 秒 / 5 分钟 / 1 分 30 秒），与后端提示同口径。
    conveneCooldownText (ms) {
      const sec = Math.max(1, Math.ceil(Number(ms || 0) / 1000))
      if (sec >= 60) {
        if (sec % 60 === 0) return (sec / 60) + ' 分钟'
        return Math.floor(sec / 60) + ' 分 ' + (sec % 60) + ' 秒'
      }
      return sec + ' 秒'
    },
    // ---- 城市操作 ----
    doConvene () {
      // ★ 2026-10-09 ：卡控 5 秒 → **3 秒**（与后端 /city/convene 双保险，防连点刷人口）
      // ★ 2026-10-22 ：人口达到/超过民居上限时改为 **5 分钟**（已满时总人口不涨，只加空闲）
      // ★ 2026-10-23 ：两个间隔都可从二战系统配置调整；这里只做前端双保险，后端仍会校验。
      // ★ 2026-10-21 ：冷却**按城市**分别记（与后端同口径）——A 城冷却中不影响 B 城。
      const cid = this.city ? this.city.id : 0
      const atByCity = this._conveneAtByCity || (this._conveneAtByCity = {})
      const cd = this.conveneCooldownMs
      if (Date.now() - (atByCity[cid] || 0) < cd) {
        this.notify('操作过于频繁, 请 ' + this.conveneCooldownText(cd) + '后再试')
        return
      }
      api.post('/games/ezfy/city/convene', { city_id: cid }).then(r => {
        if (r.code === 0) {
          atByCity[cid] = Date.now()
          // ★ 2026-10-23 召集成功后立即同步人口/待恢复/临时空闲，别等首页 30 秒轮询；
          //   否则页面上会继续显示召集前的空闲人口，玩家以为临时空闲没加上。
          this.load()
        }
        this.alert(r, '召集完成')
      })
    },
    // doPlacate 已上移到 /view 加载处（那里有 placate 参数，用于拼确认文案），此处不再重复定义
    doTax () {
      api.post('/games/ezfy/city/tax', { tax_rate: parseInt(this.taxInput) || 0 })
        .then(r => this.alert(r, '税率已调整', () => { this.load() }))
    },
    doRename () {
      api.post('/games/ezfy/city/rename', { name: this.renameInput }).then(r => this.alert(r, '城市已更名'))
    },
    doCreateCity () {
      api.post('/games/ezfy/city/create', { x: parseInt(this.newCityX) || 0, y: parseInt(this.newCityY) || 0 }).then(r => this.alert(r, '新城建造成功'))
    },
    // ★ 分页翻页（which: 'dyn' 军队动态 / 'sta' 驻军 / 'rep' 战报列表 / 'notice' 公告）
    //   ⚠️ 方法名必须与下方通用 pagerGo 不同：同名时对象字面量后定义会覆盖先定义，
    //   曾导致网易/公告「下一页」点到的是通用版 pagerGo（cur 传字符串 → 返回原值 → 无反应）。
    sectionPagerGo (which, delta) {
      if (which === 'dyn') {
        this.dynPage = Math.min(this.dynMarchTotalPages, Math.max(1, this.dynPage + delta))
      } else if (which === 'sta') {
        this.dynStationPage = Math.min(this.dynStationTotalPages, Math.max(1, this.dynStationPage + delta))
      } else if (which === 'queue') {
        this.queuePage = Math.min(this.queueTotalPages, Math.max(1, this.queuePage + delta))
      } else if (which === 'notice') {
        this.noticePage = Math.min(this.noticeTotalPages, Math.max(1, this.noticePage + delta))
      } else if (which === 'exo') {
        this.exchangePage = Math.min(this.exchangeTotalPages, Math.max(1, this.exchangePage + delta))
        this.loadExchange()
      } else if (which === 'exm') {
        this.exchangeMPage = Math.min(this.exchangeMTotalPages, Math.max(1, this.exchangeMPage + delta))
        this.loadExchange()
      } else {
        this.repPage = Math.min(this.repTotalPages, Math.max(1, this.repPage + delta))
      }
    },
    // ★ 城市列表 [派遣]：像出征一样，把当前城市的部队/军官/随军资源送到自己另一座城
    doDispatchTo (ct) {
      if (ct.id === this.city.id) { this.notify('不能派遣到当前所在城市'); return }
      this.selCell = {
        x: ct.x, y: ct.y, area_type: 3, city_id: ct.id, user_id: ct.user_id,
        name: ct.name, level: ct.city_level, mine: true, occupied: true
      }
      this.selDetail = null
      this.orderType = 8
      this.orderCalc = null
      this.go('orderpre')
    },
    async doDestroyCity (ct) {
      const cur = this.city && ct.id === this.city.id ? '（这是当前所在城市，弃城后会自动切换到其他城市）' : ''
      const ok = await this.ask('确定弃城「' + ct.name + '」吗？' + cur + '该城市的建筑、部队、军官、野地都会一并消失，' +
        '坐标会恢复为普通平原。此操作不可恢复！')
      if (!ok) return
      api.post('/games/ezfy/city/destroy', { city_id: ct.id }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          // 摧毁的是当前城时后端会自动切到别的城 → 整体重载（含军官，军官跟城走）
          // ★ 2026-10-04 与切城一致：只拉新城接口。loadTechs 在此时 this.city 仍是旧城 id，
          //   会打到刚被摧毁的城；科技页进页时会按新城懒加载，这里不再预取。
          this.load()
          this.loadTroops()
          this.loadOnDutyOfficers()
        } else this.notify(r.msg)
      })
    },
    doSwitch (ct) {
      api.post('/games/ezfy/city/switch', { city_id: ct.id }).then(r => {
        if (r.code === 0) {
          // ★ 2026-10-04 切城只调新城的接口（用户反馈「切城卡顿 + 切完还打旧城接口」）：
          //   · 后端切城接口现在直接返回新城完整 view 数据 → 一次请求完成
          //     「切城 + 全量刷新」，资源栏同步更新，不再二次 GET /view（无 3s 延迟）。
          //   · 旧后端没有 view 字段时回落到 load()。
          if (r.data && r.data.city) this.applyView(r.data)
          else this.load()
          this.cur = 'home'
        } else this.notify(r.msg)
      })
    },
    // ★ 第九轮：从城市列表直接发起「运输」到自己的另一座城市
    doTransportTo (ct) {
      if (ct.id === this.city.id) { this.notify('不能运输到当前所在城市'); return }
      this.selCell = {
        x: ct.x, y: ct.y, area_type: 3, city_id: ct.id, user_id: ct.user_id,
        name: ct.name, level: ct.city_level, mine: true, occupied: true
      }
      this.selDetail = null
      this.orderType = 5
      this.orderCalc = null
      this.go('orderpre')
    },
    async doAbandon (w) {
      if (!await this.ask('确定放弃野地(' + w.x + ',' + w.y + ')吗?')) return
      api.post('/games/ezfy/city/abandon-wild', { wildland_id: w.id }).then(r => this.alert(r, '已放弃该野地', () => {
        // ★ 2026-10-05 修复「放弃附属野地后列表不刷新」：成功后重拉附属野地列表
        //   （含野地 + 占领城市两块，wildfull 接口返回）
        this.loadWilds()
      }))
    },
    // ★ 2026-09-28 「附属野地」列表里 [停止] 用：野地记录 → stopCollect 需要的最小订单对象
    //   stopCollect 只用 id 发请求、用 target_name/target_x/target_y 拼提示文案，故这里够了。
    wildOrderArg (w, orderId) {
      return {
        id: orderId || w.gather_order_id,
        target_name: w.terrain === 8 ? '海底森林' : (w.terrain_name || '野地'),
        target_x: w.x,
        target_y: w.y
      }
    },
    async doOccupy (op, o) {
      if (!await this.ask(op === 'build' ? '确定将该城市正式建立为自己的城市吗?' :
        op === 'destroy' ? '确定摧毁该城市吗? 城市及其建筑/部队将全部消失, 不可恢复!' :
          '确定将城市归还给原玩家吗?')) return
      // ★ 2026-10-16 占城操作（建立/摧毁/归还）成功后补一次局部刷新：
      //   之前只弹「操作已提交」，alert 自带的 load() 只刷 /view 不刷 wildfull 页数据，
      //   卡顿/慢网下占领城市面板停在旧行（[建立] 按钮仍在），这里对齐采集/放弃等操作
      //   统一走 loadWilds() 重拉附属野地+占领城市+赎城面板；once 防连点重复下单。
      this.once('occupy' + o.id, () => api.post('/games/ezfy/city/occupy/' + op, { occupy_id: o.id })
        .then(r => this.alert(r, '操作已提交', () => this.loadWilds())))
    },
    // ---- 赎回（★ 2026-10-25 改版）：被占城市原主人花钻石**立即赎回**，无需占领方同意 ----
    // 系统收取 20% 手续费（占领方实收 80%）；下面 doRansomCancel/doRansomHandle 只为处理
    // 老版本遗留的 status=0 待审申请（新流程不再产生），处理完就可以整块删掉。
    // 按城市ID找待处理的赎城申请（无则 null），供「占领城市」面板行内审批
    ransomOf (cityId) {
      return this.ransoms.find(r => r.city_id === cityId) || null
    },
    // 申请对应的城市已不在占领列表（被建立城市/摧毁/归还等并发残留），兜底行仍可审批
    orphanRansoms () {
      const occ = new Set(this.occupies.map(o => o.city_id))
      return this.ransoms.filter(r => !occ.has(r.city_id))
    },
    async doRansom (ct) {
      // ★★ 2026-10-25 用户需求「赎回 改成 玩家赎回 无需 占领者同意，手续费系统收 20%」：
      //   从「发起申请 → 占领方审批」改成**一步成交**，确认框必须把「立即生效 + 20% 手续费」说清楚。
      //   防抖三道（前端锁 / 后端 2 秒节流 / 事务内原子置终态），核心防刷钻在后端。
      if (this.ransomBusy) return
      this.ransomBusy = true
      try {
        const cost = this.ransomCost || 500
        const fee = Math.floor(cost / 5)
        if (!await this.ask(`确定花费 ${cost} 钻石立即赎回城市[${ct.name}](${ct.x},${ct.y})吗？\n` +
          '无需占领方同意，确认后城市立即返还、民心恢复。\n' +
          `其中 ${fee} 钻石为系统手续费(20%)，${cost - fee} 钻石归占领方。`)) return
        const r = await api.post('/games/ezfy/city/ransom', { city_id: ct.id })
        this.alert(r, r && r.code === 0 ? '赎回成功' : '赎回失败')
        this.go('cities')
      } finally {
        this.ransomBusy = false
      }
    },
    async doRansomCancel (ct) {
      // ★ 2026-10-15 赎城防抖（与 doRansom 同一把锁）：请求未返回前连点直接忽略
      if (this.ransomBusy) return
      this.ransomBusy = true
      try {
        if (!await this.ask(`确定撤销对城市[${ct.name}](${ct.x},${ct.y})的赎城请求吗? 押金将自动退回。`)) return
        const r = await api.post('/games/ezfy/city/ransom/cancel', { city_id: ct.id })
        this.alert(r, '已撤销')
        this.go('cities')
      } finally {
        this.ransomBusy = false
      }
    },
    // ★ 2026-10-28 用户需求「自己的城市被占领 28 小时 [赎城] 加一个 [弃城]，可以自己弃城」：
    //   占领保护期（默认 28 小时，二战系统配置可调）**满了**之后，原主人才能自己弃城；
    //   期内只能 [赎回]。弃城与普通 [弃城] 同口径：物理删除城市、坐标恢复原地形，不可恢复。
    //   按钮显隐由后端 `can_abandon_occupy` 驱动（保护期没到就不显示，零冗余文案）。
    async doAbandonOccupied (ct) {
      // 与 doRansom 共用同一把锁：请求未返回前连点直接忽略
      if (this.ransomBusy) return
      this.ransomBusy = true
      try {
        if (!await this.ask(`确定放弃被占领的城市[${ct.name}](${ct.x},${ct.y})吗？\n` +
          '该城市的建筑、部队、军官、野地都会一并消失，坐标恢复为原地形。\n' +
          '此操作不可恢复，弃城后这座城市不会再返还给你！')) return
        const r = await api.post('/games/ezfy/city/occupied/abandon', { city_id: ct.id })
        this.alert(r, r && r.code === 0 ? '已弃城' : '弃城失败')
        this.go('cities')
      } finally {
        this.ransomBusy = false
      }
    },
    async doRansomHandle (rm, op) {
      // ★ 2026-10-15 赎城防抖（与 doRansom 同一把锁）：请求未返回前连点直接忽略
      if (this.ransomBusy) return
      this.ransomBusy = true
      try {
        // ★ 2026-10-15 20% 手续费：同意时系统收取，占领方实收 80%（与后端 r.Cost - r.Cost/5 同口径）
        const receive = rm.cost - Math.floor(rm.cost / 5)
        if (!await this.ask(op === 1 ? `确定同意赎城吗? 你将收到 ${receive} 钻石(已扣20%手续费), 城市[${rm.city_name}](${rm.x},${rm.y})将归还原主人。` :
          `确定拒绝该赎城请求吗? 原主人的 ${rm.cost} 钻石押金将退回。`)) return
        const r = await api.post('/games/ezfy/city/ransom/handle', { ransom_id: rm.id, op })
        this.alert(r, '已处理')
        this.loadWilds()
      } finally {
        this.ransomBusy = false
      }
    },
    // 野地列表 → [采集]：进「出征页」选兵种后再下达命令
    // ★ 原来直接 POST 且没带 troops，后端必然返回「请选择出征部队」，
    //   表现就是「点了采集没反应 / 采集发不出去」。
    openWildGather (w) {
      this.selCell = { x: w.x, y: w.y, area_type: 1, level: w.level,
        name: (w.terrain_name || '野地'), wild_id: w.id }
      this.selDetail = { id: w.id, level: w.level, x: w.x, y: w.y }
      this.orderType = 4
      this.orderCalc = null
      this.go('orderpre')
    },
    // ★★ 2026-10-18 野地列表 → [增援]：派兵驻守自己的附属野地（不占采集名额, 与采集部队并存）
    openWildReinforce (w) {
      this.selCell = { x: w.x, y: w.y, area_type: 1, level: w.level,
        name: (w.terrain_name || '野地'), wild_id: w.id }
      this.selDetail = { id: w.id, level: w.level, x: w.x, y: w.y }
      this.orderType = 6
      this.orderCalc = null
      this.go('orderpre')
    },
    // ---- 军队 ----
    // 兵种详情(复刻 cityTroopView.html): 军队页/军工厂页/城防页点兵种名进来
    openTroopView (troopId) {
      this.troopViewId = troopId
      this.troopViewBack = this.cur === 'defence' ? 'defence'
        : (this.cur === 'factory' ? 'factory' : (this.cur === 'troop' ? 'troop' : 'troops'))
      this.go('troopview')
    },
    // 训练/建造确认页(复刻 createTroop.html / createDefence.html)
    openTrainPre (t, mode) {
      this.trainSel = t
      this.trainMode = mode || 'troop'
      this.trainCount = 10
      this.trainSplit = true // ★ 2026-10-18 训练页操作选项默认「全部工厂」
      this.go('trainpre')
    },
    // ★ 2026-09-28 城内军队表改为遍历 armyRows（全兵种含 0 数量），原 quickTrain 已无引用，移除。
    // ★ 2026-09-28 训练页「(最多 N)」后面加 [最大]，一键填成上限。
    //   上限口径与页面上显示的「(最多 N)」完全一致（maxTrainable computed：
    //   资源 / 人口（城防为城防空间）取最小），点了不会填出个填不下的数。
    //   上限为 0 时不可点（灰掉），与出征页 [最大] 的处理保持一致。
    setTrainMax () {
      const max = this.maxTrainable
      if (max <= 0) return
      this.trainCount = max
    },
    // ★★ 2026-10-25 用户要求：「[开始训练] 前端防抖 + 后端多点卡控 + 1 秒延迟」。
    //   前端两层：① trainBusy（请求未返回前忽略点击）；② 无论成败，1 秒冷却后才解锁
    //   —— 与后端 1 秒窗口口径一致，连点只会下达一次征兵命令（不会扣两次资源）。
    doTrainPre () {
      if (!this.trainSel) return
      if (this.trainBusy) return // 连点/请求未返回：直接忽略，不重复下达命令
      const n = parseInt(this.trainCount) || 0
      if (n <= 0) { this.notify('请填写建造数量'); return }
      this.trainBusy = true
      const unlock = () => {
        setTimeout(() => { this.trainBusy = false }, 1000)
      }
      api.post('/games/ezfy/troops/train', {
        troop_id: this.trainSel.id, count: n, split: this.trainSplit
      }).then(r => {
        unlock()
        this.alert(r, '征兵已开始', () => {
          // ★ 2026-10-21 「用完就恢复」：征兵当下人口就转入待恢复，立刻刷一次军队数据，
          //   让「空闲人口」这一秒就扣下去（不要等造完/下次轮询才看到变化）。
          this.loadTroops()
          // ★ 2026-10-05 训练完跳回军工厂页（不再停留在具体兵种训练页）；城防则回城防页
          this.go(this.trainMode === 'defence' ? 'defence' : 'factory')
        })
      }, unlock)
    },
    // 拆除城防设施(复刻 troopDefence.html 每行的 [拆除])
    async doDismiss (t) {
      const have = this.troopCount(t.id)
      if (have <= 0) { this.notify('城内没有该城防设施'); return }
      if (!await this.ask('确定拆除全部 ' + t.name + '×' + have + ' 吗?')) return
      api.post('/games/ezfy/troops/dismiss', { troop_id: t.id }).then(r => {
        this.alert(r, '城防设施已拆除')
        if (r.code === 0) this.loadTroops()
      })
    },
    // 取消训练队列（征兵序列玩家可以自己取消，资源全额退还）
    async doCancelTrain (q) {
      const ok = await this.ask('确定取消「' + q.name + '×' + q.count + '」的训练吗？' +
        '取消会收取 10% 手续费，其余资源退还（不受仓储上限影响）。')
      if (!ok) return
      api.post('/games/ezfy/troops/train/cancel', { queue_id: q.id }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '已取消训练，资源已退还', 'ok')
          this.loadTroops()
          this.load()
        } else this.notify(r.msg || '取消失败', 'error')
      })
    },
    // ★ 解散部队（军队页面要有解散按钮，数量由玩家自己输入）
    async doDisband (t) {
      const input = await this.ask('解散「' + t.name + '」多少个？（当前 ' + t.count + ' 个）\n' +
        '解散后兵力直接销毁，不退还任何资源。', { input: true, value: '1', placeholder: '数量' })
      if (input === null || input === undefined) return
      const n = parseInt(input, 10)
      if (!n || n <= 0) { this.notify('解散数量必须是大于 0 的整数', 'error'); return }
      if (n > t.count) { this.notify('解散数量不能超过当前数量 ' + t.count, 'error'); return }
      api.post('/games/ezfy/troops/disband', {
        city_id: this.city ? this.city.id : 0, troop_id: t.troop_id, count: n
      }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : ('已解散 ' + t.name + '×' + n), 'ok')
          this.loadTroops()
          this.load()
        } else this.notify(r.msg || '解散失败', 'error')
      })
    },
    // 秒 → 「1分0秒 / 5分20秒 / 57秒」(复刻 createTroop.html 的「时间」)
    durText (sec) {
      const s = Math.max(0, Math.floor(Number(sec) || 0))
      if (s < 60) return s + '秒'
      const m = Math.floor(s / 60)
      if (m < 60) return m + '分' + (s % 60) + '秒'
      const hh = Math.floor(m / 60)
      return hh + '小时' + (m % 60) + '分'
    },
    // ★ 2026-09-28 「累计采集/采集资源实时变化、累加展示，不能只靠刷新」：
    //   采集中部队由后端下发 gather = { start_ms, period_ms, per_food/steel/oil/rare }，
    //   前端每一秒(由 gatherNow 驱动)本地外推出「累计时长 + 累计产出的资源」。
    //   ★★ 2026-10-21 用户要求：**已采行必须逐秒滚动** —— 本地按 elapsed/period_ms
    //   连续等比例估算展示（不要搞「整点小时」那一套，那只是后台实际入库的口径；
    //   到点结算以后端数据为准，这里是预览）。
    //   ★ 2026-10-21 海野倍率：后端 per_* 已按海野（海底森林/岛屿/渔场/矿礁）乘过海野倍率，前端直接外推即可。
    //   ★★ 2026-10-21 用户规则修正：采集达到负重上限后，本次超出部分丢弃，之后资源不再增加，
    //   但宝物仍按周期继续；预览按 carry_cap 封顶，并提示「负重已满，资源不再增加」。
    liveGather (o) {
      const g = o && o.gather
      if (!g || !g.period_ms) return null
      const now = this.gatherNow || Date.now()
      const elapsed = Math.max(0, now - g.start_ms) // 毫秒
      if (elapsed <= 0) return null
      // ★ 2026-10-21 连续等比例外推：已采 = 本期产出 × (已驻守毫秒 / 一期毫秒)
      const mk = v => Math.floor(((v || 0) * elapsed) / g.period_ms)
      let food = mk(g.per_food)
      let steel = mk(g.per_steel)
      let oil = mk(g.per_oil)
      let rare = mk(g.per_rare)
      const gold = 0
      const cap = o.carry_cap || 0
      const rawTotal = food + steel + oil + rare
      const full = cap > 0 && rawTotal >= cap
      if (full) {
        // 后端只保留负重装得下的部分，超出直接丢弃；预览同样封顶。
        let left = cap
        const take = v => {
          const n = Math.min(v, left)
          left -= n
          return n
        }
        food = take(food)
        steel = take(steel)
        oil = take(oil)
        rare = take(rare)
      }
      const total = food + steel + oil + rare
      return { timeText: this.durText(elapsed / 1000), food, steel, oil, rare, gold, total, cap, full }
    },
    // ★ 2026-09-28 给单条军队动态附上实时采集视图 _lg（有 gather 才算采集中）；复用来避免模板算两遍。
    withLg (o) {
      const g = this.liveGather(o)
      return g ? Object.assign({}, o, { _lg: g }) : o
    },
    // ★ 2026-09-28 「军队动态 / 出征队列的『抵达时间：12秒』也要自动刷新」：
    //   倒计时原来是后端算好的一次性字符串(time_text)，只有重新拉接口才会变，
    //   静止不动看起来像卡死。改法与采集实况一致 —— 后端下发**绝对到点时间戳**，
    //   前端每秒(由 gatherNow 驱动)本地重算剩余。
    //   覆盖三种会走的倒计时：status=0 抵达(arrive_time) / status=2 返回(return_time) /
    //   战斗中(5) 本回合剩余(battle_left_ms 基点)。驻守采集累计 / 空闲待机 / 等待指挥不动。
    //
    // ★★ 归零后必须「让位」给后端，否则会卡在 0 秒（2026-09-28 用户报的 bug）：
    //   到点之后本地推算就失效了（后端该结算了），这时如果还继续返回「0秒」，
    //   就会**盖住**后端刚下发的新状态文案（如「驻守(空闲) / 待机」），用户永远看到 0 秒。
    //   处理：过了到点时刻再走一个 2 秒缓冲（等自动重拉把新数据换回来）就返回 ''，
    //   让模板回落到 o.time_text —— 也就是后端说了算。
    liveLeft (o) {
      if (!o) return ''
      const now = this.gatherNow || Date.now()
      // 战斗中：本回合剩余 = 拉数据那一刻的剩余 - 已过去的时间
      if (o.status === 5) {
        if (!o.battle_left_ms && o.battle_left_ms !== 0) return ''
        const left = Math.max(0, (o.battle_left_ms || 0) - (now - (this.dynAt || now)))
        return this.durText(left / 1000)
      }
      // 出征中：距抵达还差多久
      if (o.status === 0 && o.arrive_time > 0) {
        const left = o.arrive_time - now
        if (left <= 0) {
          // ★★ 2026-10-08 到点后的三种处理（用户报「到了以后倒计时乱 / 卡在 0 秒」）：
          //   ① 刚过点（2 秒内）→ 交回后端文案，等 checkDueRefresh 拉新状态；
          //   ② 过点 2 秒还没换状态 → 说明后端这次没结算完（懒结算 + per-uid 重入守卫：
          //      同一玩家的并发请求只有一个能结算，其余被挡掉，拿到的仍是 status=0 + 已过去的
          //      arrive_time）。这时后端文案是「0秒」，玩家会以为卡死 —— 明确显示「结算中…」，
          //      同时 checkDueRefresh 仍在按 2 秒间隔重试；
          //   ③ 重试用尽后仍不变 → 列表停在「结算中…」，玩家可手动刷新（不再显示误导性的 0 秒）。
          if (now - o.arrive_time > 2000) return '结算中…'
          return ''
        }
        return this.durText(left / 1000)
      }
      // 返航中：距回城还差多久（同上：到点即让位）
      if (o.status === 2 && o.return_time > 0) {
        const left = o.return_time - now
        if (left <= 0) return ''
        return this.durText(left / 1000)
      }
      return ''
    },
    // 有倒计时就附上实时视图 _lt（无则返回原样，模板回落到后端给的 time_text）
    withLive (o) {
      const t = this.liveLeft(o)
      return t ? Object.assign({}, o, { _lt: t }) : o
    },
    doRecover (w) {
      // ★ 2026-10-15 3 秒卡控（与后端 /troops/recover 双保险，防连点）
      if (Date.now() - (this.lastRecoverAt || 0) < 3000) { this.notify('操作过于频繁, 请 3 秒后再试'); return }
      // ★ 2026-10-06 按数量恢复：用户填了恢复数量就只恢复这么多，留空/0 = 全部恢复该兵种
      const n = parseInt(this.woundNums && this.woundNums[w.id], 10) || 0
      const body = { troop_id: w.troop_id, type: w.type }
      if (n > 0 && n < w.count) body.count = n
      // ★ 2026-09-25 用户反馈「伤兵救治后要手动刷新页面才显示」→ 成功后重拉军队数据(含伤兵营/逃兵营)
      api.post('/games/ezfy/troops/recover', body)
        .then(r => {
          // 只有真正成功才进冷却，失败/资源不足不影响重试
          if (r.code === 0) this.lastRecoverAt = Date.now()
          this.alert(r, '伤兵已恢复', () => { this.loadTroops() })
        })
    },
    doRecoverAll (t) {
      // ★ 2026-10-15 3 秒卡控（与后端 /troops/recover 双保险，防连点）
      if (Date.now() - (this.lastRecoverAt || 0) < 3000) { this.notify('操作过于频繁, 请 3 秒后再试'); return }
      api.post('/games/ezfy/troops/recover', { all: true, type: t })
        .then(r => {
          if (r.code === 0) this.lastRecoverAt = Date.now()
          this.alert(r, '伤兵已恢复', () => this.loadTroops())
        })
    },
    // ★ 2026-10-14 钻石一键恢复全部伤兵（单价由二战系统配置下发 heal_all_diamond，默认 100 钻）
    // ★ 2026-10-26 逃兵营（type=1）也走这里：同样 100 钻、同样 3 秒卡控，文案按类型区分。
    doRecoverAllDiamond (t) {
      // ★ 2026-10-15 3 秒卡控：放在 confirm 之前，连点不会重复弹确认框（与后端双保险）
      if (Date.now() - (this.lastRecoverAt || 0) < 3000) { this.notify('操作过于频繁, 请 3 秒后再试'); return }
      const cost = this.healAllDiamond || 100
      const label = Number(t) === 1 ? '逃兵' : '伤兵'
      if (!window.confirm('花费 ' + cost + ' 钻石恢复全部' + label + '？')) return
      api.post('/games/ezfy/troops/recover', { all: true, type: t, use_diamond: true })
        .then(r => {
          if (r.code === 0) { this.mallDiamond = null; this.lastRecoverAt = Date.now() }
          this.alert(r, label + '已全部恢复', () => { this.loadTroops() })
        })
    },
    // ---- 科技 ----
    // ★ 2026-09-28 科技列表不显示资源消耗/前置条件，点[研究]进详情页(techpre)查看后再确认
    openTechPre (t) {
      this.techSel = t
      this.go('techpre')
    },
    doTechPre () {
      if (!this.techSel) return
      // ★ 2026-09-28 多城研究：带上当前城（研究限制/扣资源都在当前城）
      api.post('/games/ezfy/techs/research', { tech_id: this.techSel.tech_id, city_id: this.city ? this.city.id : 0 }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '科技研究已开始', 'ok')
          this.go('techs')
          this.loadTechs()
        } else this.notify(r.msg || '研究失败', 'error')
      })
    },
    // ★ 2026-09-26 修复「科技加速道具买完实际使用不生效」：
    //   原来这里调 POST /techs/speed {minutes:10} —— 后端既不校验道具也不扣任何东西，
    //   等于「免费无限减 10 分钟」（minutes 还能被改成任意值，直接把研究刷完）。
    //   商城卖的「科技加速30分钟/2小时」因此完全没有使用入口。
    //   现在改成消耗科技加速道具(item_type=5)，与建筑页 [加速] 同一口径。
    doSpeedTech (item) {
      this.useSpeedItem(item, 5, 0, () => { this.loadTechs() })
    },
    async doCancelTech (t) {
      if (!await this.ask('确定取消研究「' + t.name + '」吗？取消研究将不退还已消耗的资源。')) return
      // ★ 2026-09-28 多城研究：取消只作用于**当前城**的研究队列；取消不退款
      api.post('/games/ezfy/techs/cancel', { tech_id: t.tech_id, city_id: this.city ? this.city.id : 0 }).then(r => {
        this.alert(r, '研究已取消，已消耗资源不退还')
        if (r.code === 0) this.loadTechs()
      })
    },
    // ---- 司令部 ----
    // ★ 2026-10-25 用户要求「军官列表简约」：equipTip（列表里的「(装备 军事+5 后勤+5 学识+5)」）
    //   已从 EzfyAcademy 列表删除，装备加成统一进军官详情（officerEquipText）看。
    // ★ 2026-09-25：openEquipDetail（跳独立「装备详情页」）已随页面一起删除 ——
    //   各装备表不再有 [查看] 按钮，改成点装备名/套装名在该行下面展开详情卡。
    // ★ 装备六项战斗属性的展示文案（伤害/防御/生命/移动距离/暴击几率/暴击伤害）
    equipAttrText (e) {
      if (!e) return ''
      const parts = []
      if (e.dmg) parts.push('伤害+' + e.dmg + '%')
      if (e.def) parts.push('防御+' + e.def + '%')
      if (e.hp) parts.push('生命+' + e.hp + '%')
      if (e.move) parts.push('移动距离+' + e.move + '%')
      if (e.crit) parts.push('暴击几率+' + e.crit + '%')
      if (e.crit_dmg) parts.push('暴击伤害+' + e.crit_dmg + '%')
      // 老装备（军/后/学 三维）走另一套展示
      if (e.military) parts.push('军事+' + e.military)
      if (e.logistics) parts.push('后勤+' + e.logistics)
      if (e.learning) parts.push('学识+' + e.learning)
      return parts.join(' ')
    },
    // ★ 2026-10-18 简约化：装备行不再展开详情卡，悬停（原生 title）看加成/套装信息
    equipItemTip (e) {
      if (!e) return ''
      const lines = []
      lines.push(e.name + ' [' + (e.tier_name || '普通') + ']')
      lines.push('部位：' + (e.slot || e.type) + (e.level ? ' · ' + e.level + '级' : ''))
      lines.push('装备加成：' + (this.equipAttrText(e) || '无'))
      if (e.set_id) {
        const s = this.setOf(e.set_id)
        if (s) lines.push('所属套装：' + s.name + '（穿齐 ' + s.parts + ' 件才生效）')
      }
      return lines.join('\n')
    },
    // ★ 2026-10-18 套装名悬停提示：套装加成 + 我的进度
    equipSetTip (setId) {
      const s = this.setOf(setId)
      if (!s) return ''
      const lines = []
      lines.push(s.name + '（穿齐 ' + s.parts + ' 件才生效）')
      lines.push('套装加成：' + (this.setBonusText(setId) || '无'))
      lines.push('我的进度：已拥有 ' + (s.owned || 0) + '/' + s.parts + ' 件')
      return lines.join('\n')
    },
    // ★ 2026-10-14 装备商城统一价格展示：套装件 1~7 是**黄金价**、其余（套装 8~17 / 系列件 / 散件）是**钻石价**。
    //   返回 { cur:'gold'|'diamond', val:数字, text:'50,000金' / '1500钻' }，表格与购买确认页共用，避免两处各写一套。
    equipPrice (e) {
      if (!e) return { cur: 'diamond', val: 0, text: '—' }
      const gold = parseInt(e.price_gold) || 0
      const dia = parseInt(e.price_diamond) || 0
      if (dia > 0) return { cur: 'diamond', val: dia, text: this.fmtN(dia) + '钻' }
      if (gold > 0) return { cur: 'gold', val: gold, text: this.fmtN(gold) + '金' }
      return { cur: 'diamond', val: 0, text: '—' }
    },
    // ★ 2026-10-14 商城装备列表的「简约名」：表格已有部位列 → 名称不再拼「·头盔 / ·护肩」后缀。
    //   只在**结尾恰好是「·+本件部位」**时去掉（套装件「海军上将套装(Admiral)·头盔」→「海军上将套装(Admiral)」；
    //   系列件「革命者[围巾]」结尾不是·部位 → 原样）。详情/购买页保持完整名，列表才用短名。
    equipShortName (e) {
      if (!e) return ''
      const n = String(e.name || '')
      const s = String(e.slot || '')
      if (s && n.length > s.length + 1 && n.slice(-(s.length + 1)) === '·' + s) {
        return n.slice(0, n.length - s.length - 1)
      }
      return n
    },
    // 军官详情里「装备战斗加成」一行
    officerBattleText (b) {
      if (!b) return ''
      const parts = []
      if (b.dmg) parts.push('伤害+' + b.dmg + '%')
      if (b.def) parts.push('防御+' + b.def + '%')
      if (b.hp) parts.push('生命+' + b.hp + '%')
      if (b.move) parts.push('移动距离+' + b.move + '%')
      if (b.crit) parts.push('暴击几率+' + b.crit + '%')
      if (b.crit_dmg) parts.push('暴击伤害+' + b.crit_dmg + '%')
      return parts.join(' ')
    },
    // ★ 2026-10-10 军官详情「装备战斗加成」改展示**单件**装备的加成（三维 + 六项）：
    //   减去已激活套装的加成，避免与「套装」行重复（单件六项清零后曾完全撞车）。
    //   顺序：先三维（军事/后勤/学识），再六项百分比。
    officerEquipText (o) {
      if (!o) return ''
      const parts = []
      // 单件三维 = 装备总三维(单件+套装) − 已激活套装三维
      const mi = (o.equip_military || 0) - (o.set_military || 0)
      const lo = (o.equip_logistics || 0) - (o.set_logistics || 0)
      const le = (o.equip_learning || 0) - (o.set_learning || 0)
      if (mi) parts.push('军事+' + mi)
      if (lo) parts.push('后勤+' + lo)
      if (le) parts.push('学识+' + le)
      // 单件六项 = battle 全量六项 − 已激活套装六项
      const b = o.battle || {}
      const sv = {}
      ;(o.set_progress || []).forEach(sp => {
        if (!sp.active) return
        ;['dmg', 'def', 'hp', 'move', 'crit', 'crit_dmg'].forEach(k => { sv[k] = (sv[k] || 0) + (sp[k] || 0) })
      })
      ;[['dmg', '伤害'], ['def', '防御'], ['hp', '生命'], ['move', '移动距离'], ['crit', '暴击几率'], ['crit_dmg', '暴击伤害']]
        .forEach(([k, label]) => {
          const v = (b[k] || 0) - (sv[k] || 0)
          if (v) parts.push(label + '+' + v + '%')
        })
      return parts.join(' ')
    },
    // 防御兵种（城防 type=4：碉堡/榴弹炮/反坦克炮/防空炮…）固定阵地
    isDefenceTroop (t) {
      return !!t && t.type === 4
    },
    doSaveTargets () {
      const list = []
      // ★ 防御兵种(type=4) 固定阵地：前进/停止一律按「停止」提交
      const defIds = {}
      ;(this.troopsData.cfgs || []).forEach(c => { if (c.type === 4) defIds[c.id] = true })
      // ★ 2026-10-28 防守数量上限 = 系统配置「单城兵力上限」；按**当前城**保存（每城一套配置）
      const max = this.troopsData.troop_max || 0
      const cid = this.city && this.city.id ? this.city.id : 0
      for (const tid in this.targetCfg) {
        const t = this.targetCfg[tid]
        const isDef = !!defIds[parseInt(tid)]
        // 留空 / 0 = 全部；负数当 0；超过上限按上限（服务端还会再夹一次）
        let dc = parseInt(t.defCount, 10)
        if (!dc || dc < 0) dc = 0
        if (max > 0 && dc > max) dc = max
        // 夹完回写界面：手机上粘贴了一个超大数，保存后立刻看到被收窄成上限值（不留错觉）
        const shown = dc > 0 ? dc : ''
        if (t.defCount !== shown) t.defCount = shown
        list.push(api.post('/games/ezfy/targets', {
          city_id: cid,
          troop_id: parseInt(tid), atk_target_troop: t.atk,
          atk_move: isDef ? 0 : t.atkMove,
          def_target_troop: t.def,
          def_move: isDef ? 0 : t.defMove,
          def_count: dc
        }))
      }
      // ★ 2026-10-28 不再「无论成败都提示已保存」：有失败就把后端文案弹出来（缺列/城已丢等）
      Promise.all(list).then(rs => {
        const bad = (rs || []).filter(r => r && r.code !== 0)
        if (bad.length) this.notify((bad[0] && bad[0].msg) || '保存失败，请重试')
        else this.notify('战斗配置已保存')
      })
    },
    openOrder (o) {
      this.loadOrderView(o.id)
    },
    // ★ 2026-09-25 统一进「命令详情」的入口：拉详情 + 用 go() 把页面写进 URL。
    //   原来 openOrder 直接改 this.cur（不写 URL），刷新后就掉回上一个页面，
    //   看起来就是「出征队列/详情刷新后消失」。现在带 oid 进 URL，
    //   刷新时 restoreFromUrl 会按 oid 重新拉回来（见下方）。
    loadOrderView (orderId) {
      const oid = parseInt(orderId, 10)
      if (!oid) return
      api.get('/games/ezfy/orders/' + oid).then(r => {
        if (r.code === 0) {
          this.curOrder = r.data
          this.go('orderview')
        } else {
          this.notify(r.msg || '命令不存在或已结束')
          this.go('orders')
        }
      })
    },
    jumpReport (rid) {
      api.get('/games/ezfy/reports/' + rid).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
          // ★ 2026-10-06 战斗报告默认收起「逐回合详情」（点[展开/收起逐回合详情]查看）
          this.showDetail = false
          this.go('reportview')
        }
      })
    },
    // 点开战报: 拉详情 + 标已读，跳转独立「战报详情」页（不再行内展开）
    openReport (r) {
      if (!r) return
      api.get('/games/ezfy/reports/' + r.id).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
        } else {
          // 详情接口异常时至少把列表里的内容显示出来
          this.curReport = r
        }
        // ★ 2026-10-06 战斗报告默认收起「逐回合详情」；指挥室默认展开可自行收起
        this.showDetail = false
        const item = this.reports.find(x => x.id === r.id)
        if (item) item.is_read = 1
        // openReport 可能从「军情警讯」进也可能从「战斗报告」进，这里记录它来自哪个分区
        // ★ 2026-09-30 军团战报(category='corps')回到军团 tab
        this.reportTab = (r.category === 1) ? 3 : (r.category === 'corps' ? 5 : 4)
        // ★ 2026-10-09 能从军团战报列表点进来 ⇒ 一定有军团（否则该 tab 根本不显示），
        //   顺手置位，保证「战报详情页」顶部导航也能高亮/显示「军团战报」。
        if (r.category === 'corps') this.reportHasCorps = true
        this.go('reportview')
      })
    },
    // ★ 删除战报：不需要二次确认（）
    // ★ 2026-09-26 「战报查询 [查询] 右边加个 [一键删除]，物理删除吧节约服务器资源」：
    //   删的是**自己名下全部战报**（后端 DELETE，不软删），不可恢复 → 先走页面内确认条问一次。
    //   ⚠️ 有搜索词时列表只是筛选，删除范围仍是全部 —— 文案里写清楚，别让玩家以为只删列表里那几条。
    // ★ 2026-10-01 军情按当前城过滤：一键删除也只删**当前城市**的战报
    // ★ 2026-10-13 军情警讯不受城市限制 → 军情警讯的删除也按**全部城市**
    // ★★ 2026-10-16 战斗报告同样改成全部城市口径（列表/徽标/删除一致；后端 Reports 已不按城市过滤）
    //   —— 若删除仍只删当前城，会出现「删完列表还剩一堆」的矛盾。
    async doClearReports () {
      if (!this.reports.length) {
        this.notify('暂无战报可删除')
        return
      }
      // ★ 2026-10-06 军情警讯/战斗报告分区删除：category 口径与 loadReports/ezfyReportCounts 一致
      const scopeName = this.reportTab === 3 ? '军情警讯' : '战斗报告'
      const scopeTxt = '全部城市'
      if (!await this.ask('确定删除【' + scopeTxt + '】的全部' + scopeName + '吗？\n（物理删除，不可恢复；' +
        (this.reportWord ? '当前只是搜索筛选，删除范围仍是' + scopeTxt + '全部' + scopeName : '共 ' + this.reports.length + ' 条') + '）')) return
      api.post('/games/ezfy/reports/clear', {
        city_id: 0,
        category: this.reportTab === 3 ? 1 : 2
      }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : scopeName + '已全部删除')
          this.repPage = 1
          this.loadReports()
        } else this.notify(r.msg || '删除失败')
      })
    },
    delReport (r) {
      if (!r) return
      api.post('/games/ezfy/reports/' + r.id + '/delete', {}).then(res => {
        if (res.code === 0) {
          this.notify('战报已删除')
          this.curReport = null
          this.go('reports')
        } else {
          this.notify(res.msg || '删除失败')
        }
      })
    },
    // ★ 取消出征命令（「出征队列可以取消」）
    //   不限命令类型：行进中(0)/驻守中(1)都能取消，部队原路返回出发城市；
    //   运输/派遣带出去的随军资源也会随部队一起带回来。
    async doRecall (o) {
      const name = (o && o.type_name) || '该命令'
      if (!await this.ask('确定取消「' + name + '」吗？部队将原路返回出发城市。')) return
      api.post('/games/ezfy/order/recall', { order_id: o.id })
        .then(r => this.alert(r, name + '已取消', () => { this.loadOrders(); this.loadDynamics() }))
    },
    // ★ 2026-10-02 城主遣返盟军驻军：驻军返航回出发城市（联络中心页）
    async expelGarrison (g) {
      if (!await this.ask('确定遣返来自「' + g.from_city + '」的这支驻军吗？部队将原路返回出发城市。')) return
      api.post('/games/ezfy/liaison/expel-garrison', { order_id: g.id })
        .then(r => this.alert(r, '已遣返', () => { this.loadLiaison(); this.loadDynamics() }))
    },
    // ★ 2026-09-30 行军计谋：神兵天降(13,去程减80%)/战略转移(14,回程减360分钟)，耗 7 信号弹
    //   从计谋页(scheme)进入，使用后返回「军情 → 军队动态」。
    async doMarchScheme (o, schemeId) {
      const schemeName = schemeId === 13 ? '神兵天降' : '战略转移'
      if (!await this.ask('对 (' + o.target_x + ',' + o.target_y + ') 这支部队使用「' + schemeName + '」吗？消耗信号弹×7，每种计谋每支部队限一次。')) return
      api.post('/games/ezfy/scheme/use', { scheme_id: schemeId, order_id: o.id })
        .then(r => {
          if (r.code !== 0) { this.notify(r.msg || '发动失败'); return }
          this.notify('「' + schemeName + '」已生效')
          this.loadOrders()
          this.loadDynamics()
          this.loadBag()
          this.loadSchemes() // 刷新信号弹持有量（计谋页顶部）
          // ★ 2026-09-30 使用完返回军情军队动态页（）
          this.go('reports')
        })
    },
    // ★ 2026-09-30 打开行军计谋页：记录目标部队并跳到「计谋」页
    openScheme (o) {
      this.schemeOrder = o
      this.loadSchemes() // 拉最新信号弹持有量
      this.go('scheme')
    },
    orderStatusText (o) {
      if (o.status === 0) return '行进中 ' + this.remain(o.arrive_time)
      // ★ 2026-09-24 采集空闲化: 到达野地后空闲待命(需手工[采集]), 开始采集才显示「驻守采集」
      if (o.status === 1) return (o.order_type === 7 ? (o.arrive_time > 0 ? '驻守采集' : '驻守(空闲)') : '已到达')
      if (o.status === 2) return '返回中 ' + this.remain(o.return_time)
      if (o.status === 3) return '已完成'
      if (o.status === 6) return '等待中(目标已被进攻, 排队等待交战)'
      if (o.status === 5) return '战斗中 第' + (o.battle_round || 1) + '回合'
      return '全队阵亡'
    },
    // ---- 地图/出征 ----
    // ★ 军队动态里活动目标的坐标标识（打活动野地/活动寇/特殊城市的部队）
    actTag (actType) {
      if (actType === 1) return '活动'
      if (actType === 2) return '活动寇'
      if (actType === 3) return '特殊城市'
      return ''
    },
    // ★ 2026-10-01 地图世界 500×500（后端 ezfyWorldSize=500，有效坐标 1~499）：
    //   视野中心靠边时周边会带出越界格（如跳 1,1 后出现 -1,-1），统一判定为「空地」。
    isMapOOB (cell) {
      return !cell || cell.x < 1 || cell.x > 499 || cell.y < 1 || cell.y > 499
    },
    cellText (cell) {
      // 复刻 map/index.html: 格子文案为「名称(等级)」；★ 现在每格第二行统一显示坐标，
      //   所以这里一律只返回「名称」部分，本城也不再拼 (x,y)，避免和下面那行重复。
      // ★ 2026-09-25 玩家城市名太长，格子会被撑变形 → 地图上（含本城）统一显示「城市」，
      //   具体城市名/城主点进目标详情页再看（鼠标悬停的 title 里也有全称，见 cellTip）。
      // ★ 2026-10-01 越界格（地图外不存在的地方）统一显示「空地」
      if (this.isMapOOB(cell)) return '空地'
      if (cell.area_type === 3) return '城市'
      // 活动目标: 复刻 mapView.html 的「活动野地N级 / 活动寇N级 / 特殊城市N级」
      if (cell.act_type === 1) return '活动(' + cell.act_level + ')'
      if (cell.act_type === 2) return '活动寇(' + cell.act_level + ')'
      if (cell.act_type === 3) return '特殊(' + cell.act_level + ')'
      if (cell.name === '寇城(废墟)') return '墟'
      if (cell.area_type === 2) return '寇(' + cell.level + ')'
      if (cell.terrain === 8) {
        // ★ 纯海洋/海底森林分开显示(用户规范): 海洋不带等级, 海野显示海底森林(N级)
        if (cell.is_ocean) return '海洋'
        return '海底森林(' + cell.level + ')'
      }
      // 陆地野地按地形名显示(平原/草原/森林/盆地/丘陵/沼泽/岛屿)
      return (cell.terrain_name || '野') + '(' + cell.level + ')'
    },
    // ★ 格子悬浮提示：城市名字在格子里会被截断，鼠标悬停看全称。
    //   坐标已固定显示在格子第二行，这里不再重复拼。
    //   ★ 2026-09-25：格子上一律显示「城市」(含本城)，所以本城也给出全称提示。
    cellTip (cell) {
      if (this.isMapOOB(cell)) return '空地（地图外，不存在）'
      if (cell.area_type === 3) {
        return (cell.name || '城市') + (cell.owner ? ' · 城主 ' + cell.owner : '') +
          ' (' + cell.x + ',' + cell.y + ')'
      }
      return this.cellText(cell)
    },
    cellClass (cell) {
      if (this.isMapOOB(cell)) return 'ezfy-empty'
      if (cell.mine) return 'ezfy-mine'
      // ★ 2026-10-08 敌人城市标红（个人战争/宣战、军团敌对、军团战争的对象城市）
      if (cell.area_type === 3 && cell.enemy) return 'ezfy-enemy'
      // ★ 2026-09-30 带名将守将的活动野地：特殊标识（优先于普通活动野地）
      if (cell.act_type === 1 && cell.act_officer) return 'ezfy-act-named'
      if (cell.act_type === 1) return 'ezfy-act-wild'
      if (cell.act_type === 2) return 'ezfy-act-kou'
      if (cell.act_type === 3) return 'ezfy-act-city'
      if (cell.area_type === 3) return 'ezfy-city'
      if (cell.area_type === 2) return 'ezfy-kou'
      // ★ 2026-10-20 渔场(10)/矿礁(11)也是海野，沿用海洋格的浅蓝底
      if (cell.terrain === 8 || cell.terrain === 10 || cell.terrain === 11) return 'ezfy-sea'
      return 'ezfy-wild'
    },
    // 点「发现精英中立城市」→ 直接进该寇城的目标详情(里面就是 侦查/掠夺/征服 出征入口),
    // 而不是仅仅把地图挪过去
    openElite () {
      if (!this.eliteCell) return
      const c = this.eliteCell
      this.openCell({ x: c.x, y: c.y, area_type: 2, level: c.level || 0, name: '寇城' })
    },
    openCell (cell) {
      // ★ 2026-10-01 越界「空地」格不可点（地图外没有目标详情）
      if (this.isMapOOB(cell)) {
        this.notify('该坐标在地图外（空地），没有目标')
        return
      }
      this.selCell = cell
      this.selDetail = null
      this.warText = ''
      this.warStatus = 0
      this.cur = 'wildview'
      // ★ 2026-09-30 进入详情刷新收藏状态：避免用旧地图星标的旧数据导致「没收藏却显示已收藏」
      this.loadStars()
      if (cell.area_type === 3) {
        if (!cell.mine && cell.user_id) this.checkWar()
        return
      }
      // ★ 2026-10-05 这里传的 type 现在只是**兼容用**：服务端已改成按地形自己判定走哪套配置
      //   （海洋 8 / 岛屿 7 / 渔场 10 / 矿礁 11 → 海野；寇城 → 3；其余 → 陆地野地），
      //   传了也不影响结果。
      //   ⚠️ 别再把服务端改回「以客户端 type 为准」—— 这个表达式会把岛屿(7) 推成 1（陆地野地），
      //   导致详情页预览陆军守军、实际打起来却是海军（战报对不上）。
      const isSeaWild = cell.terrain === 7 || cell.terrain === 8 ||
        cell.terrain === 10 || cell.terrain === 11
      const ttype = cell.area_type === 2 ? 3 : (isSeaWild ? 2 : 1)
      // 顺带记住这块地上「我的野地记录 id」，采集/派遣下单要用
      const mine = (this.wildlands || []).find(w => w.x === cell.x && w.y === cell.y)
      if (mine) cell.wild_id = mine.id
      // ★ 2026-10-25 只应用「最新一次 openCell」的响应：连点两个格子时先发的请求可能后到，
      //   会把上一块地的详情（含 is_starred）盖到当前格子上 →「没收藏却显示已收藏」。
      const seq = ++this.detailSeq
      api.get('/games/ezfy/map/wildland?x=' + cell.x + '&y=' + cell.y + '&type=' + ttype).then(r => {
        if (r.code === 0 && seq === this.detailSeq) this.selDetail = r.data
      })
    },
    // 详情页里选命令 → 进出征页(复刻 mapView 的 [侦查][掠夺][征服])
    pickOrder (t) {
      // ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服 ——
      //   要先到「附属野地」页把这块地[放弃]（放弃后该坐标恢复为中立野地，才能再打）。
      //   按钮已灰掉，这里再兜一层（防老页面缓存/键盘操作绕过）；后端 createOrder 有同样校验。
      if ((t === 1 || t === 2 || t === 3) && this.isOwnWild) {
        this.notify('这是你自己的附属野地, 不能' + (this.orderNames[t] || '') +
          '; 请先在「附属野地」里[放弃]该野地')
        return
      }
      // ★★ 2026-10-18 用户规则更新: 采集中的野地允许被攻打(攻方获胜则歼灭该野地
      //   的驻军+采集部队, 采集资源随之损失); 原「采集中不能打」的前端拦截已移除
      //   (后端 createOrder 拦截同样移除)。
      this.orderType = t
      this.orderCalc = null
      this.go('orderpre')
    },
    // ★★ 2026-10-13 「出征确认」命令下拉的**可选命令**判据 —— 与地图详情页那几个命令按钮的
    //   显示条件**完全同源**（不另发明一套），保证「地图上能点的」和「下拉里能选的」永远一致。
    //   用户规则：还没宣战就只能侦查 → 下拉里就**只列侦查**，不能选的命令一律不出现。
    //   ★ 覆盖全部进出征页的入口：地图详情页(侦查/掠夺/征服/采集/运输/增援)、
    //     城市列表(运输/派遣)、野地列表(采集)。
    orderAvail (t) {
      const c = this.selCell
      if (!c) return false
      const isCity = c.area_type === 3
      const isMineCity = isCity && !!c.mine                     // 自己的城（地图不给命令，入口在城市列表的 [运输]/[派遣]）
      const isAllyCity = isCity && !c.mine && !!c.ally          // 同盟成员的城
      // ① 自己的城：只能 运输(5) / 派遣(8)
      if (isMineCity) return t === 5 || t === 8
      // ② 同盟成员的城：运输(5) / 增援(6)
      if (isAllyCity) return t === 5 || t === 6
      // ③ 采集(4)：只有「自己的附属野地」（有 wild_id = 从我的野地进来的；或详情页判定的 mine）
      if (t === 4) return !isCity && (!!c.wild_id || (this.isOwnWild && !!c.occupied))
      // ④ 驻守采集(7)：只在「附属野地」页对已有采集队操作，出征页不给
      if (t === 7) return false
      // ⑤ 运输(5) 只对城；增援(6) 除城市外, 还允许对自己的附属野地驻守（★★ 2026-10-18）
      if (t === 5) return false
      if (t === 6) return isCity ? false : (!!c.wild_id || (this.isOwnWild && !!c.occupied))
      // ⑥ 侦查(1)：非自己的城常显；野地只要不是自己的附属野地
      if (t === 1) return isCity ? !c.mine : !this.isOwnWild
      // ⑦ 掠夺(2)/征服(3)
      if (t === 2 || t === 3) {
        if (isCity) {
          // 与地图按钮同判据：宣战已生效 / 管理端关了宣战开关 / 军团交战期
          return this.warStatus === 2 || !this.warRequire || this.atWar
        }
        // 野地：自己的附属野地要先[放弃]
        // ★★ 2026-10-18 采集中不再拦截（攻方获胜将歼灭驻军+采集部队, 采集资源随军损失）
        return !this.isOwnWild
      }
      return false
    },
    // ★ 2026-10-13 出征确认页改命令：按新命令重算（不同命令的出征上限完全不同 ——
    //   采集/派遣无上限、运输/增援另一套、掠夺/征服受集结令+军官加成），算完会把兵力夹回新上限。
    onOrderTypeChange () {
      this.orderCalc = null
      this.$nextTick(() => this.scheduleOrderCalc())
    },
    declareWar () {
      if (!this.selCell || !this.selCell.city_id) return
      // ★ 用户规则「同盟玩家不能宣战」：按钮已经藏了，这里再兜一层，防止老页面缓存绕过
      if (this.selCell.ally) { this.notify('同盟成员之间不能宣战'); return }
      api.post('/games/ezfy/war/declare', { city_id: this.selCell.city_id }).then(r => {
        this.alert(r, '宣战成功')
        this.checkWar()
      })
    },
    // 未宣战/宣战未生效时点掠夺/征服 → 只提示，不进出征页
    warBlock (name) {
      if (!this.selCell || !this.selCell.user_id) return
      if (this.warStatus === 1) {
        this.alert({ code: 1, msg: (this.warText || '宣战尚未生效') + '，生效后方可' + name }, '宣战尚未生效')
      } else {
        this.alert({ code: 1, msg: '需先对 ' + (this.selCell.owner || '对方') + ' 宣战，宣战 24 小时后生效，生效后方可' + name }, '需先宣战')
      }
    },
    checkWar () {
      if (!this.selCell || !this.selCell.user_id) return
      api.get('/games/ezfy/war/status?target_user_id=' + this.selCell.user_id).then(r => {
        if (r.code === 0) {
          this.warText = r.data.text
          this.warStatus = r.data.status || 0
          // ★ 管理端「宣战功能」开关：关掉时不需要宣战，掠夺/征服直接可点
          //   （后端 isAtWar 同时恒为 true，两边口径一致）。
          //   注意不能靠把 warStatus 伪造成 2 —— 那样同盟城市的 运输/增援 会被误判而消失。
          this.warRequire = r.data.war_require !== false
          // ★ 2026-09-25 接军团交战期字段
          //   at_war = 可掠夺/征服（个人宣战已生效 或 管理端关掉宣战开关 或 军团交战期）；
          //   corps_war = {active, corps_name, text}，active 时详情页显示绿色提示。
          this.atWar = !!r.data.at_war
          this.corpsWar = r.data.corps_war || null
        }
      })
    },
    // ★★ 2026-09-28 「[最大] 按钮点的不是当前兵种最大兵力，应该是出征还剩多少的最大」，
    //   且「滑动滚轮也加下剩余可出征兵种最大卡控」。
    //
    //   本兵种可填上限 = min(城内现有, 出征上限剩余)：
    //     - 城内现有   = troopCount(id)（不能派出城里没有的兵）
    //     - 上限剩余   = 出征上限 − **其它**兵种已填合计（把本兵种算作 0 时的剩余额度）
    //
    //   这样「每个兵种都拉满」正好凑满上限，总量天然不会超 —— 不需要事后校正。
    //   例：上限 10000，A 已填 3000、B 已填 2000 → C 的上限 = 10000−5000 = 5000；
    //       A 自己的上限 = 10000−2000 = 8000（再受城内现有封顶）。
    //
    //   下列情况只按「城内现有」卡（返回 troopCount）：
    //     - 还没点过 [计算]（orderCalc 为空，上限未知）→ 不能凭空编一个上限；
    //     - 管理端把「出征上限」开关关了（cap_unlimited）→ 本来就不限。
    //   ★ 2026-10-02 运输(5)/派遣(8)也参与携带上限判定（orderCapApplies 恒 true）。
    orderQtyMax (id) {
      // ★ 2026-10-09 「自己被处于指挥时，指挥里面的兵种不能再出征」：
      //   本城正在被敌军攻击（防御指挥室进行中）时，参战兵种不能出征（后端 createOrder 会硬拦），
      //   这里把它们的可派上限置 0 → 滑块/数字框/[最大] 一起禁用，避免玩家白填一遍再被拒。
      //   锁定列表由 /order/preview 下发（locked_troops）；不参与防御(def_move=-1)的兵种不在其中。
      if (this.troopLocked(id)) return 0
      const own = this.troopCount(id)
      const c = this.orderCalc
      // 上司关了上限开关 → 不限
      if (c && c.cap_unlimited) return own
      // ★ 攻略 2026-09-28：还没拿到出征上限(troop_cap)时，不能退回「城内总数」(own)，
      //   否则 orderCalc null 的首帧滑块会放开到整个城内量(如 98 万航母)而不是上限(如 12.9 万)。
      //   直接返回 0 → 滑块在计算完成前保持禁用，等 doCalc 落地后自动放开到正确上限。
      if (!c) return 0
      const remain = (c.troop_cap || 0) - (this.orderTroopTotal - this.orderQty(id))
      // remain 可能为负（其它兵种已经把额度吃超了）→ 本兵种只能填 0
      return Math.max(0, Math.min(own, remain))
    },
    // troopLocked 该兵种是否被「防御指挥中」锁定（参战兵种不能出征）。
    //   ★ 2026-10-09 用户规则：自己被处于指挥时，指挥（防御战斗）里面的兵种不能再出征；
    //     司令部标了「不参与防御」的兵种不参与当次防御 → 不锁、照常可出征。
    //   数据来源：/order/preview 的 def_locked / locked_troops（点了 [计算] 后才有）。
    troopLocked (tid) {
      const c = this.orderCalc
      if (!c || !c.def_locked) return false
      const list = c.locked_troops || []
      const n = Number(tid)
      for (let i = 0; i < list.length; i++) {
        if (Number(list[i]) === n) return true
      }
      return false
    },
    // ★ 2026-10-09 野地/寇城「掉落宝物」的可读化：后端下发的是结构化 JSON
    //   [{"name":"防弹衣","count":1,"pct":2},...]，原来直接 print 出来是一串花括号（用户反馈）。
    //   → 「防弹衣、青铜军刀」。
    //   ★ **概率和数量都不展示**（只列宝物名；pct/count 属后台配置信息）。
    //   ⚠️ 老数据可能是纯文本（如「中级/高级」）→ 解析失败就原样返回，不吞掉内容。
    fmtWildDrop (raw) {
      const s = String(raw || '').trim()
      if (!s) return ''
      let rows
      try { rows = JSON.parse(s) } catch (e) { return s }
      if (!Array.isArray(rows)) return s
      const parts = []
      for (let i = 0; i < rows.length; i++) {
        const name = String((rows[i] || {}).name || '').trim()
        if (name) parts.push(name)
      }
      return parts.length ? parts.join('、') : s
    },
    // ★ 上限变小后（改集结令 / 换带队军官 / 换城市）把已填兵力重新夹进新上限，
    //   否则「本次出兵」会一直红着超限，而滑块又因为 max 变成 0 和数字框显示不一致。
    //   策略：按兵种 id 升序依次分配剩余额度（先到先得），结果稳定可预期。
    clampOrderTroops () {
      const c = this.orderCalc
      if (!c || c.cap_unlimited || !this.orderCapApplies) return
      if (this.orderTroopTotal <= (c.troop_cap || 0)) return
      let left = c.troop_cap || 0
      const ids = Object.keys(this.orderTroops)
        .filter(k => this.orderQty(k) > 0)
        .sort((a, b) => Number(a) - Number(b))
      for (const id of ids) {
        const give = Math.max(0, Math.min(this.orderQty(id), left))
        left -= give
        this.$set(this.orderTroops, id, give)
      }
    },
    // 当前该兵种已填的出征数量（没填过 = 0）；滑块与数字框都绑它，保证两边显示一致
    orderQty (id) {
      const v = this.orderTroops[id]
      if (v === undefined || v === null || v === '') return 0
      const n = parseInt(v, 10)
      return isNaN(n) || n < 0 ? 0 : n
    },
    // ★ 2026-09-25 出征页「兵种数量搭配」：滑块 + 数字框 **双向联动**
    //   - 拖动滑块 → 数字框跟着变；填数字 → 滑块跟着走；两边共用 orderTroops[id] 一个值。
    //   - 用 $set 写对象键：orderTroops 初始是 {}，直接赋值新键 Vue2 侦测不到，
    //     滑块动完数字框不会刷新（这就是「联动」失效的原因）。
    //   - ★ 2026-09-28 夹紧上限由「城内现有」改成 orderQtyMax(id)
    //     = min(城内现有, 出征上限剩余) —— 滑块也要按「剩余可出征」卡控。
    //   - 夹紧后若数值没变（如本来已是上限又填了更大的数），Vue 不会重渲染，
    //     DOM 里会留着用户填的非法数字 → 这里手动把输入框内容回写，保证「看到的 = 提交的」。
    onOrderQtyInput (id, ev) {
      const max = this.orderQtyMax(id)
      let n = parseInt(ev.target.value, 10)
      if (isNaN(n) || n < 0) n = 0
      if (n > max) n = max
      this.$set(this.orderTroops, id, n)
      if (ev && ev.target && String(n) !== String(ev.target.value)) {
        ev.target.value = String(n)
      }
    },
    // [最大] = 一键带上「该兵种还能派出的最大数量」（滑到最后、数字框同步）
    //   ★ 2026-09-28 用户纠正：「不是当前兵种最大兵力，应该是出征还剩多少的最大」。
    //   所以取 orderQtyMax = min(城内现有, 出征上限剩余)，而不是城内现有。
    setOrderQtyMax (id) {
      const max = this.orderQtyMax(id)
      if (max <= 0) return
      this.$set(this.orderTroops, id, max)
    },
    // 随军资源：当前已填数值（任何一行都校验成非负整数）
    resQty (key) {
      const v = this.resVal(key)
      if (v === undefined || v === null || v === '') return 0
      const n = parseInt(v, 10)
      return isNaN(n) || n < 0 ? 0 : n
    },
    // ★ 随军资源：单行上限 = min(城内现有, 可用负重 - 其它行重量)
    //   ★ 修正 2026-09-28：(cap - others) 是【绝对上限】，不能再加 cur（旧写法 = cur + (cap - others)，
    //     满负重时反而 allowed=2*cur>cap 还能往右滑）。用绝对上限后：
    //     满负重时 allowed = 本行当前值 → 只能左收不能右加，其它行同样被卡住 —— 实现「负重满了就滑不动」。
    //   可用负重 = 负重 - 行军油耗（orderResUsableCap）。
    resQtyMax (key) {
      if (this.orderResDisabled) return 0
      const avail = this.resAvail(key)
      const others = this.orderResTotal - this.resQty(key)   // 其它行当前重量
      const allowed = this.orderResUsableCap - others          // 本行绝对上限（扣掉别行占用）
      let mx = Math.min(avail, allowed)
      if (mx < 0) mx = 0
      return Math.floor(mx)
    },
    // 该资源行单个输入：夹到 [0, resQtyMax]，并双向联动（滑块/数字框共用 trXxx 一个值）
    onResInput (key, ev) {
      const max = this.resQtyMax(key)
      let n = parseInt(ev.target.value, 10)
      if (isNaN(n) || n < 0) n = 0
      if (n > max) n = max
      this.setResVal(key, n)
      if (ev && ev.target && String(n) !== String(ev.target.value)) {
        ev.target.value = String(n)
      }
    },
    // [最大]：这行一次拖到「当前负重剩余还能塞下的最大值」（仍不超城内现有）
    setResMax (key) {
      const max = this.resQtyMax(key)
      if (max <= 0) return
      this.setResVal(key, max)
    },
    // 取/写随军资源五项（gold/food/steel/oil/rare → trXxx）
    resVal (key) {
      if (key === 'gold') return this.trGold
      if (key === 'food') return this.trFood
      if (key === 'steel') return this.trSteel
      if (key === 'oil') return this.trOil
      return this.trRare
    },
    setResVal (key, n) {
      if (key === 'gold') this.trGold = n
      else if (key === 'food') this.trFood = n
      else if (key === 'steel') this.trSteel = n
      else if (key === 'oil') this.trOil = n
      else this.trRare = n
    },
    // 城内现有该资源数量
    resAvail (key) {
      const c = this.city || {}
      if (key === 'gold') return c.gold || 0
      if (key === 'food') return c.food || 0
      if (key === 'steel') return c.steel || 0
      if (key === 'oil') return c.oil || 0
      return c.rare || 0
    },
    // 出征表单 → 请求体(部队/资源/军官/宿营)
    orderBody () {
      const body = {
        order_type: this.orderType,
        target_type: this.selCell.area_type === 3 ? 3 : (this.selCell.area_type === 2 ? 2 : 1),
        target_x: this.selCell.x,
        target_y: this.selCell.y,
        wait_min: (parseInt(this.waitH) || 0) * 60 + (parseInt(this.waitM) || 0),
        // ★ 2026-10-07 自动战斗（出征页可配）：打玩家城市(target_type=3)强制 0
        //   （真人守方必须留指挥机会，后端也会再兜一道）；打野地 / AI 寇城按玩家所选。
        auto_battle: this.orderAutoBattleVal
      }
      if (this.selCell.area_type === 3 && this.selCell.city_id) {
        body.target_id = this.selCell.city_id
      }
      // ★ 采集(4)/派遣(7) 后端要按 target_id 校验「这块野地是你占领的」，
      //   而地图格子只有坐标 → 这里按坐标在「我的野地」里反查记录 id。
      //   （之前没带 target_id，采集/派遣永远发不出去）
      // ★★ 2026-10-18 增援(6) 到自己的附属野地同理要带 target_id（野地 id）
      if (this.orderType === 4 || this.orderType === 7 ||
        (this.orderType === 6 && this.selCell.area_type !== 3)) {
        if (this.selCell.wild_id) {
          body.target_id = this.selCell.wild_id
        } else {
          const w = (this.wildlands || []).find(x => x.x === this.selCell.x && x.y === this.selCell.y)
          if (w) body.target_id = w.id
        }
      }
      const troops = []
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k]) || 0
        if (n > 0) troops.push({ troopId: parseInt(k), count: n })
      }
      body.troops = troops
      body.resources = {
        food: parseInt(this.trFood) || 0, steel: parseInt(this.trSteel) || 0,
        oil: parseInt(this.trOil) || 0, rare: parseInt(this.trRare) || 0,
        gold: parseInt(this.trGold) || 0
      }
      if (this.orderOfficer && this.orderOfficer !== '0') body.officer = this.orderOfficer
      // ★ 集结令个数：每个 +10 万出征上限，单次最多 10 个
      body.gather = parseInt(this.orderGather) || 0
      return body
    },
    // 复刻原版出征页的 [计算]: 预览油耗/负重/耗时, 不下达命令
    doCalc () {
      if (!this.selCell) return
      api.post('/games/ezfy/order/preview', this.orderBody()).then(r => {
        if (r.code === 0) {
          this.orderCalc = r.data
          // ★ 2026-09-28：上限可能因「集结令 / 带队军官 / 城市」变化而变小，
          //   这里立刻把已填兵力夹回新上限内，保证滑块 max 与数字框始终一致、不会红着超限。
          this.clampOrderTroops()
        } else this.alert(r, '计算失败')
      })
    },
    // ★ 2026-10-16 出征卡顿：所有会触发 /order/preview 的入口（改集结令/切军官/应用预设/进出征页
    //   自动算一次）统一走 300ms 防抖 —— 后端该接口是 7 条并行 DB 查询 + 纯只读重算，连点/连改
    //   会堆一串请求排队卡住页面。与兵种滑块 watcher 共用同一个 _resCalcT 定时器，互斥收敛。
    scheduleOrderCalc () {
      if (this._resCalcT) clearTimeout(this._resCalcT)
      this._resCalcT = setTimeout(() => this.doCalc(), 300)
    },
    // ★ 清空出征表单（用户反馈「出征还有上次留的数据」）
    //   每次进出征页都重置，避免上次的兵力/资源被误当成这次的出征内容。
    resetOrderForm () {
      this.orderTroops = {}
      this.orderOfficer = '0'
      this.orderGather = 0
      this.orderCalc = null
      this.trFood = 0
      this.trSteel = 0
      this.trOil = 0
      this.trRare = 0
      this.trGold = 0
      this.waitH = 0
      this.waitM = 0
      this.presetSel = 0
      // ★ 2026-10-07 自动战斗每次进出征页重置为「是」（野地 / AI 寇城默认；打玩家城时强制否、此值不生效）
      this.orderAutoBattle = 1
    },
    // ---- 预设编队 ----
    // 拉取我的预设列表（出征页下拉 + 司令部预设 tab 共用）
    loadPresets () {
      api.get('/games/ezfy/presets').then(r => {
        if (r.code === 0) this.presets = r.data.presets || []
      })
    },
    // ★ 预设页 [计算]：与 doCalc 的区别——预设不含目标，传本城坐标按 0 距离估算油耗/负重，
    //   出兵上限（troop_cap）与集结令/军官加成照常生效，返回后照旧把已填兵力夹回新上限。
    presetCalc () {
      // ★ 2026-09-29 预设编队永远按「出征(掠夺)」口径预览：强制 orderType=2，
      //   否则全局 orderType 若残留为运输(5)/派遣(8)（无出征上限），滑块上限会被放成城内总数
      this.orderType = 2
      const c = this.city
      if (!c || !c.x || !c.y) return
      api.post('/games/ezfy/order/preview', {
        // 与 orderBody() 同口径：不传 city_id（后端按主城/当前城结算）
        order_type: 2,
        target_x: c.x,
        target_y: c.y,
        troops: this.presetTroopGroups(),
        resources: {},
        officer: this.orderOfficer && this.orderOfficer !== '0' ? this.orderOfficer : '',
        wait_min: 0,
        gather: parseInt(this.orderGather) || 0
      }).then(r => {
        if (r.code === 0) {
          this.orderCalc = r.data
          this.clampOrderTroops()
        } else this.alert(r, '计算失败')
      })
    },
    // 预设页/出征页提交用的兵力数组（orderTroops → [{troopId,count}]，只带 >0 的）
    presetTroopGroups () {
      const out = []
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k]) || 0
        if (n > 0) out.push({ troopId: parseInt(k), count: n })
      }
      return out
    },
    // 预设页集结令改动：夹到 [0, gatherMax] 后按 0 距离重算
    onPresetGatherChange () {
      let n = parseInt(this.orderGather) || 0
      if (isNaN(n) || n < 0) n = 0
      const cap = this.gatherMax
      if (n > cap) n = cap
      this.orderGather = n
      this.presetCalc()
    },
    startPresetAdd () {
      // 进入新增表单：清空共享表单状态，从当前城现有/军官起步
      this.resetOrderForm()
      this.orderType = 2   // ★ 2026-09-29 预设模板始终按「出征(掠夺)」口径，别继承运输/派遣的无上限状态
      this.presetName = ''
      this.presetAdding = true
      // ★ 2026-09-29 立即按 0 距离算一次：让「本次出兵 / 上限」立刻显示准确值（如 0 / 99000），不用等玩家点 [计算]
      this.$nextTick(() => this.presetCalc())
    },
    cancelPresetAdd () {
      this.presetAdding = false
      this.resetOrderForm()
    },
    // 保存预设：名称校验；军官 '0' → 空串；兵力 = 当前表单搭配
    savePreset () {
      const name = (this.presetName || '').trim()
      if (!name) { this.alert({ code: 400, msg: '请填写预设名称' }); return }
      if (name.length > 20) { this.alert({ code: 400, msg: '预设名称最多20字' }); return }
      api.post('/games/ezfy/presets', {
        name,
        officer: this.orderOfficer && this.orderOfficer !== '0' ? this.orderOfficer : '',
        gather: parseInt(this.orderGather) || 0,
        troops: this.presetTroopGroups()
      }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.presetAdding = false
          this.presetName = ''
          this.resetOrderForm()
          this.loadPresets()
        } else this.alert(r, '保存失败')
      })
    },
    deletePreset (p) {
      api.post('/games/ezfy/presets/delete', { id: p.id }).then(r => {
        if (r.code === 0) {
          if (this.presetSel === p.id) this.presetSel = 0
          this.loadPresets()
        } else this.alert(r, '删除失败')
      })
    },
    // ★ 出征页选中预设 → 回填：
    //   军官不在当前城可带队列表 → 回填空；兵力夹 min(预设, orderQtyMax)=可出征上限；
    //   集结令夹 [0, gatherMax]；最后按真实目标 doCalc 重算（异步，clampOrderTroops 兜底夹回上限）。
    applyPreset () {
      const p = this.presets.find(x => x.id === this.presetSel)
      if (!p) return
      // ① 军官：不在 onDutyOfficers → 回填空
      const has = p.officer && this.onDutyOfficers.some(o => o.name === p.officer)
      this.orderOfficer = has ? p.officer : '0'
      // ③ 兵力：逐个夹到当前可出征上限（兵力不够时只能填最大）
      this.orderTroops = {}
      for (const t of (p.troops || [])) {
        const max = this.orderQtyMax(t.troopId)
        if (max > 0) this.$set(this.orderTroops, t.troopId, Math.min(t.count, max))
      }
      // ② 集结令：夹到当前可用上限
      let g = parseInt(p.gather) || 0
      if (isNaN(g) || g < 0) g = 0
      const cap = this.gatherMax
      this.orderGather = g > cap ? cap : g
      // ⑥ 用真实目标重算（异步；内部会把已填兵力夹回新上限）
      this.scheduleOrderCalc()
    },
    // 改集结令数量后立刻重算，让「本次出兵 / 上限」即时刷新
    // ★ 手填数字：夹到 [0, 可用上限]，可用上限 = min(管理端配置, 背包持有量)。
    //   注意上限取自 orderCapMax（跟着 /view 下发），不再在没数据时硬夹 50。
    onGatherChange () {
      let n = parseInt(this.orderGather) || 0
      if (isNaN(n) || n < 0) n = 0
      const cap = this.gatherMax
      if (n > cap) n = cap
      this.orderGather = n
      this.scheduleOrderCalc()
    },
    doOrder () {
      if (!this.selCell) return
      // ★ 用户反馈「连点会出现多条」→ 防抖：一次点击只下达一条出征命令
      this.once('order', () => api.post('/games/ezfy/order', this.orderBody()).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.orderTroops = {}
          this.orderOfficer = '0'
          this.orderCalc = null
          this.trFood = 0; this.trSteel = 0; this.trOil = 0; this.trRare = 0; this.trGold = 0
          this.waitH = 0; this.waitM = 0
          this.orderGather = 0
          this.load()
          this.loadBag()
          // ★★ 2026-09-26 修复「点出征后跳转出征队列是空的」：
          //   出征队列页（cur==='orders'）的数据源是 `queueItems` → `this.dynamics`，
          //   由 `loadDynamics()` 填充；而这里原来直接 `cur='orders'` + `loadOrders()`
          //   （loadOrders 填的是另一份数据，orders 页根本不读它）→ 页面永远空。
          //   改成走 `go('orders')`，它内部就是 `loadDynamics()`，以后再加数据源也不会漏。
          this.go('orders')
        } else this.notify(r.msg)
      }))
    },
    loadOnDutyOfficers () {
      api.get('/games/ezfy/officers/onduty').then(r => {
        if (r.code === 0) {
          this.onDutyOfficers = r.data.officers || []
          // ★ 2026-10-19 提示已简化成「(暂无可带队军官，可前往军校招募)」，
          //   不再需要本城军官全量列表来解释「为什么」→ 省掉这次请求。
        }
      })
    },
    // ★ 2026-10-25 城市状态页：本城的市长(position=1)/城守(position=2) 是哪个军官。
    //
    //	/view 不下发职位，所以单独拉一次 /officers（就是军校「我的军官」那个接口，
    //	后端有页面级缓存，重复进页不贵）。未任命 → 空串，模板显示「未任命」。
    loadOfficerPosts () {
      api.get('/games/ezfy/officers').then(r => {
        if (r.code !== 0 || !r.data) return
        const list = r.data.officers || []
        const mayor = list.find(o => o.position === 1)
        const guard = list.find(o => o.position === 2)
        this.cityMayor = mayor ? mayor.name : ''
        this.cityGuard = guard ? guard.name : ''
      })
    },
    // ---- 联络中心 ----
    loadLiaison () {
      api.get('/games/ezfy/liaison').then(r => {
        if (r.code === 0) this.liaison = r.data
      })
    },
    // ---- 仓库保护 ----
    loadWare () {
      api.get('/games/ezfy/city/warehouse').then(r => {
        if (r.code === 0) {
          this.ware = r.data
          const ratio = {}
          for (const it of r.data.res) ratio[it.key] = it.ratio
          this.wareRatio = ratio
        }
      })
    },
    doWareSet () {
      if (this.wareSum > 100) { this.notify('四项比例合计不能超过100%'); return }
      api.post('/games/ezfy/city/warehouse', {
        food: parseInt(this.wareRatio.food) || 0,
        steel: parseInt(this.wareRatio.steel) || 0,
        oil: parseInt(this.wareRatio.oil) || 0,
        rare: parseInt(this.wareRatio.rare) || 0
      }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadWare()
        } else this.notify(r.msg)
      })
    },
    // ---- 聊天/邮箱 ----
    doChatSend () {
      const msg = (this.chatMsg || '').trim()
      if (!msg) return
      if (this.chatCooldown > 0) {
        this.notify('发言冷却中，还需 ' + this.chatCooldown + ' 秒')
        return
      }
      api.post('/games/ezfy/chat', { content: msg, channel: this.chatChannel }).then(r => {
        if (r.code === 0) {
          this.chatMsg = ''
          // 复刻原版聊天: 每次发言 30 秒冷却
          this.startChatCooldown(30)
          // ★ 降序展示：新消息在第 1 页最上面，发完回到第 1 页
          this.chatPage = 1
          this.loadChats()
          // ★ 用户反馈「世界聊天后回首页发现没出来我刚发的」：
          //   一并刷新首页聊天预览，避免回首页还要 Ctrl+F5。
          this.loadHomeChats()
        } else this.notify(r.msg)
      })
    },
    startChatCooldown (sec) {
      this.chatCooldown = sec
      if (this.chatTimer) clearInterval(this.chatTimer)
      this.chatTimer = setInterval(() => {
        this.chatCooldown -= 1
        if (this.chatCooldown <= 0) {
          this.chatCooldown = 0
          clearInterval(this.chatTimer)
          this.chatTimer = null
        }
      }, 1000)
    },
    // 聊天/好友里的玩家名 → 家园个人主页
    // 昵称逐字颜色: color 可能是逗号分隔的多色序列(如 "#f00,#0f0")
    nickChars (name) {
      return [...String(name || '')]
    },
    nickColorAt (color, i) {
      if (!color) return {}
      const cs = String(color).split(',').map(x => x.trim()).filter(Boolean)
      if (!cs.length) return {}
      return { color: cs[i % cs.length] }
    },
    // 查看他人统帅信息(复刻 PlayerController.infoOther)
    // 入口: 首页聊天 / 聊天频道 / 邮箱发件人 / 好友 / 军团成员 / 军团聊天 / 排行 —— 点玩家名
    // ★ 游戏是沉浸式的: 点玩家名只能看「二战风云」的统帅信息,
    //   **不要**跳到家园站点的个人主页(/user/:id), 那样就跳出游戏了。
    openPlayer (uid) {
      if (!uid) return
      this.playerInfoBack = this.cur === 'playerinfo' ? this.playerInfoBack : this.cur
      this.playerInfo = null
      this.go('playerinfo')
      this.loadPlayerInfo(uid)
    },
    loadPlayerInfo (uid) {
      api.get('/games/ezfy/player/' + uid).then(r => {
        if (r.code === 0) this.playerInfo = r.data
        else this.alert(r, '获取统帅信息失败')
      })
    },
    async doAddFriendById () {
      if (!this.playerInfo) return
      const name = this.playerInfo.nickname
      const remark = await this.ask('给 ' + name + ' 的验证信息（可留空）', { input: true, placeholder: '可留空' })
      if (remark === null) return
      api.post('/games/ezfy/friends/apply', { target_id: this.playerInfo.user_id, remark: remark }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送好友申请')
          this.loadPlayerInfo(this.playerInfo.user_id)
        } else {
          this.notify(r.msg || '申请失败')
        }
      })
    },
    // 军团邮件群发(复刻 CorpsController.mail, 仅军团长)
    doCorpsMail () {
      const c = (this.corpsMailContent || '').trim()
      if (!c) { this.notify('请填写邮件内容'); return }
      api.post('/games/ezfy/corps/mail', { content: c }).then(r => {
        this.alert(r, '军团邮件已群发')
        if (r.code === 0) this.corpsMailContent = ''
      })
    },
    openNotice (n) {
      this.curNotice = n
      this.cur = 'notices'
    },
    // ---- 军团 ----
    doCreateCorps () {
      api.post('/games/ezfy/corps/create', { name: this.corpsName })
        .then(r => this.alert(r, '联盟已创建', () => this.loadCorps()))
    },
    doJoinCorps (cp) {
      // ★ 2026-09-30 入口改用 /corps/apply：open 军团直接入团，需审核军团落申请待审核
      api.post('/games/ezfy/corps/apply', { corps_id: cp.id })
        .then(r => {
          this.alert(r, (r.data && r.data.need_review ? '申请已提交' : '已加入联盟'), () => {
            if (r.data && r.data.need_review) this.myApplyStatus = 1
            this.loadCorps()
          })
        })
    },
    async doLeaveCorps () {
      if (!await this.ask(this.isLeader ? '军团长退出将解散军团, 确定?' : '确定退出军团?')) return
      api.post('/games/ezfy/corps/leave', {})
        .then(r => this.alert(r, '已退出联盟', () => this.loadCorps()))
    },
    async openNoticeEdit () {
      const n = await this.ask('输入军团公告', { input: true, value: (this.myCorps ? this.myCorps.notice : '') })
      if (n !== null) api.post('/games/ezfy/corps/notice', { notice: n })
        .then(r => this.alert(r, '公告已更新', () => this.loadCorps()))
    },
    doKick () {
      if (!this.kickUserId) return this.notify('请选择成员')
      api.post('/games/ezfy/corps/kick', { user_id: this.kickUserId })
        .then(r => this.alert(r, '已踢出', () => this.loadCorps()))
    },
    // ★ 军团长任命副团长/参谋长（title 传空 = 撤职）
    async doSetCorpsTitle (m, title) {
      const label = title || '普通成员'
      const ok = await this.ask('确定把「' + m.name + '」任命为「' + label + '」吗？' +
        (title === '副团长' ? '\n副团长可以群发军团邮件。' : ''))
      if (!ok) return
      api.post('/games/ezfy/corps/member/title', { user_id: m.user_id, title: title })
        .then(r => this.alert(r, '已任命为' + label, () => this.loadCorps()))
    },
    doCorpsChat () {
      api.post('/games/ezfy/corps/chat', { content: this.corpsMsg }).then(r => {
        if (r.code === 0) {
          this.corpsMsg = ''
          this.loadCorps()
        } else this.notify(r.msg)
      })
    },
    // ★ 2026-09-30 入团审核：军团长查看待审申请
    loadCorpsApplies () {
      api.get('/games/ezfy/corps/apply').then(r => {
        if (r.code === 0) {
          this.corpsApplies = r.data.applies || []
          this.myCorpsNeedReview = r.data.need_review ? 1 : 0
        }
      })
    },
    // op 1=通过 2=拒绝
    doApplyHandle (a, op) {
      api.post('/games/ezfy/corps/apply/handle', { apply_id: a.apply_id, op: op })
        .then(r => this.alert(r, op === 1 ? '已通过' : '已拒绝', () => this.loadCorpsApplies()))
    },
    // 军团长开/关入团审核
    doToggleNeedReview () {
      api.post('/games/ezfy/corps/need-review', { need_review: this.myCorpsNeedReview ? 0 : 1 })
        .then(r => this.alert(r, '已切换审核开关', () => {
          this.loadCorps()
        }))
    },
    // ---- ★ 2026-09-25 军团外交 / 军团宣战 / 军团商城 ----
    // 切子栏：只在数据还没拉过时才请求（避免每次切 tab 重复打接口）
    switchCorpsTab (tab) {
      this.corpsTab = tab
      if (tab === 'info') {
        this.loadCorps()
        if (this.isLeader) this.loadCorpsApplies()
      } else if (tab === 'list' || tab === 'chat') {
        this.loadCorps()
      } else if (tab === 'diplomacy') {
        if (!this.corpsRelations) this.loadCorpsRelations()
      } else if (tab === 'war') {
        if (!this.corpsWars) this.loadCorpsWars()
        // 宣战 tab 复用外交的军团列表（[宣战] 入口），没拉过就补一次
        if (!this.corpsRelations) this.loadCorpsRelations()
      } else if (tab === 'mall') {
        if (!this.corpsMall.loaded) this.loadCorpsMall()
      }
    },
    loadCorpsRelations () {
      return api.get('/games/ezfy/corps/relations').then(r => {
        if (r.code === 0) {
          this.corpsRelations = r.data
          // ★ 外交数据也带 my_corps.points，顺手同步军团总积分（与 /corps/members 同源）
          if (r.data && r.data.my_corps) this.corpsPoints = r.data.my_corps.points || 0
        }
      })
    },
    loadCorpsWars () {
      return api.get('/games/ezfy/corps/war').then(r => {
        if (r.code === 0) this.corpsWars = r.data
      })
    },
    loadCorpsMall () {
      return api.get('/games/ezfy/corps/mall').then(r => {
        if (r.code === 0) {
          this.corpsMall = {
            loaded: true,
            in_corps: !!r.data.in_corps,
            my_points: r.data.my_points || 0,
            corps_points: r.data.corps_points || 0,
            items: r.data.items || [],
            my_bought: r.data.my_bought || {}
          }
        }
      })
    },
    // 某商品我已购数量（my_bought 以商品 id 为键）
    corpsBought (id) {
      const mb = this.corpsMall.my_bought || {}
      return parseInt(mb[id]) || 0
    },
    // [兑换] 是否可点：未售罄 且 未超限购
    // ★ stock 是「总库存」(-1=无限)，剩余要减掉已售 sold
    corpsMallCanBuy (item) {
      if (item.stock >= 0 && (item.stock - item.sold) <= 0) return false
      const bought = this.corpsBought(item.id)
      if (item.limit > 0 && bought >= item.limit) return false
      return true
    },
    // 军团名查找（外交/宣战操作的确认文案用）
    corpsNameOf (corpsId) {
      const list = (this.corpsRelations && this.corpsRelations.corps_list) || []
      const hit = list.filter(c => c.id === corpsId)[0]
      return hit ? hit.name : ('#' + corpsId)
    },
    // 军团长标记关系：type 0取消 / 1友好 / 2敌对（后端也会校验权限）
    async setCorpsRelation (corpsId, type) {
      const name = this.corpsNameOf(corpsId)
      const tip = type === 0 ? ('确定取消对「' + name + '」的关系标记吗？')
        : ('确定把「' + name + '」标记为' + (type === 1 ? '友好' : '敌对') + '吗？\n（友好/敌对军团均可宣战）')
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/corps/relation', { corps_id: corpsId, type: type }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg || '关系已更新')
          this.loadCorpsRelations()
        } else this.notify(r.msg || '操作失败')
      })
    },
    // 军团长对某军团宣战（后端校验权限/是否已在宣战中）
    async declareCorpsWar (corpsId) {
      const name = this.corpsNameOf(corpsId)
      const ok = await this.ask('确定对「' + name + '」宣战吗？\n宣战后 12 小时生效，48 小时后整场结束；生效期间双方成员可互相掠夺/征服。')
      if (!ok) return
      api.post('/games/ezfy/corps/war/declare', { corps_id: corpsId }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg || '宣战成功')
          this.loadCorpsWars()
          this.loadCorpsRelations()
        } else this.notify(r.msg || '宣战失败')
      })
    },
    // 军团商城兑换：数量用内联输入弹窗询问（参考 openNoticeEdit 的 ask({input:true}) 写法）
    async doCorpsMallBuy (item) {
      const bought = this.corpsBought(item.id)
      const byLimit = item.limit > 0 ? (item.limit - bought) : 9999
      const byStock = item.stock < 0 ? 9999 : (item.stock - item.sold)
      const byPoint = item.price > 0 ? Math.floor((this.corpsMall.my_points || 0) / item.price) : 9999
      const maxN = Math.max(0, Math.min(byLimit, byStock, byPoint))
      if (maxN <= 0) {
        if (byPoint <= 0) this.notify('军团积分不足：兑换 1 个需要 ' + item.price + ' 积分，我现有 ' + this.corpsMall.my_points)
        else if (byStock <= 0) this.notify('该商品已售罄')
        else this.notify('已达限购上限（' + item.limit + '）')
        return
      }
      const input = await this.ask('兑换「' + item.name + '」数量（1-' + maxN + '，单价 ' + item.price +
        ' 积分，我现有 ' + this.corpsMall.my_points + '）', { input: true, value: '1' })
      if (input === null) return
      const n = parseInt(input, 10)
      if (isNaN(n) || n < 1 || n > maxN) { this.notify('数量需在 1-' + maxN + ' 之间'); return }
      // ★ 2026-10-08 连点防抖：一次点击 = 一次兑换（后端军团商城本来就是条件原子扣减）
      this.once('corpsBuy' + item.id, () => api.post('/games/ezfy/corps/mall/buy',
        { id: item.id, count: n }).then(r => {
          if (r.code === 0) {
            this.notify(r.msg || '兑换成功')
            this.loadCorpsMall()
            this.load()
          } else this.notify(r.msg || '兑换失败')
        }))
    },
    // ---- 商城/背包/交易 ----
    // ★ 第九轮：分类切换 / 翻页
    setMallCat (c) {
      this.mallCat = c
      this.mallPage = 1
    },
    // ★ 2026-10-21 商城分类 chip 只显示两字：去掉尾巴「道具」（原值仍用于筛选/统计）。
    mallCatShort (c) {
      const s = String(c || '')
      if (s === '建筑图纸') return '图纸'
      return s.length > 2 && s.endsWith('道具') ? s.slice(0, -2) : s
    },
    mallGo (d) {
      const p = this.mallPage + d
      if (p >= 1 && p <= this.mallTotalPages) this.mallPage = p
    },
    // ★ 通用翻页（检索条件变了要把页码归 1，见各处的 setXxxWord）
    pagerGo (d, cur, total) {
      const p = cur + d
      return (p >= 1 && p <= total) ? p : cur
    },
    bagGo (d) { this.bagPage = this.pagerGo(d, this.bagPage, this.bagTotalPages) },
    // ★ 2026-10-11 背包「军官装备」tab 翻页
    bagEquipGo (d) { this.bagEquipPage = this.pagerGo(d, this.bagEquipPage, this.bagEquipTotalPages) },
    equipGo (d) { this.equipPage = this.pagerGo(d, this.equipPage, this.equipTotalPages) },
    equipAllGo (d) { this.equipAllPage = this.pagerGo(d, this.equipAllPage, this.equipAllTotalPages) },
    // ★ 2026-10-15 名将图鉴翻页（静默，无提示）
    generalGo (d) { this.generalPage = this.pagerGo(d, this.generalPage, this.generalTotalPages) },
    officerBagGo (d) {
      this.officerBagPage = this.pagerGo(d, this.officerBagPage, this.officerBagTotalPages)
    },
    openBuy (it) {
      this.buyItem = it
      this.buyCount = 1
      // ★ 双渠道道具默认用黄金（多数玩家手上黄金比钻石多）
      this.buyPayWith = it.dual_pay ? 'gold' : (it.is_diamond ? 'diamond' : 'gold')
      // ★ 跳转到独立购买详情页确认（不再行内展开）
      this.cur = 'mallbuy'
    },
    // ★ 单次可买上限 = min(管理端配置的单次上限, 库存, 背包剩余容量)。
    //   无限库存(-1)的道具只看配置值。原来这里写死 999，和 doBuy 里的 99 打架。
    //   ★ 2026-10-11 叠加「背包单件上限」：已有 have 个，最多再买 bagMax - have 个。
    buyMaxOf (it) {
      const cfgMax = parseInt(this.mallBuyMax) > 0 ? parseInt(this.mallBuyMax) : 999
      const bagMax = parseInt(this.mallBagMax) > 0 ? parseInt(this.mallBagMax) : 5000
      const bagLeft = Math.max(0, bagMax - (parseInt(it.have) || 0))
      let cap = cfgMax
      if (!it.unlimited) {
        const st = parseInt(it.stock) || 0
        cap = Math.min(cap, st)
      }
      cap = Math.min(cap, bagLeft)
      return Math.max(0, cap)
    },
    doBuy (it) {
      const n = parseInt(this.buyCount) || 0
      const bagMax = parseInt(this.mallBagMax) > 0 ? parseInt(this.mallBagMax) : 5000
      const have = parseInt(it.have) || 0
      // ★ 2026-10-11 背包上限：已达上限直接拦，剩余不足给出可买数
      if (have >= bagMax) { this.notify('「' + it.name + '」已达背包上限 ' + bagMax + ' 个，无法再购买'); return }
      const maxBuy = this.buyMaxOf(it)
      if (n < 1 || n > maxBuy) { this.notify('数量需在 1-' + maxBuy + ' 之间'); return }
      if (have + n > bagMax) {
        this.notify('「' + it.name + '」背包上限 ' + bagMax + ' 个，当前已有 ' + have + ' 个，最多再买 ' + (bagMax - have) + ' 个')
        return
      }
      if (!it.unlimited && it.stock !== undefined && n > it.stock) {
        this.notify(it.stock > 0 ? ('库存不足，最多买 ' + it.stock + ' 个') : '该道具已售罄')
        return
      }
      // ★ 支付方式：双渠道道具由玩家选；单渠道按道具属性定
      const payWith = it.dual_pay ? (this.buyPayWith || 'gold') : (it.is_diamond ? 'diamond' : 'gold')
      if (payWith === 'diamond') {
        const cost = (it.price_diamond || 0) * n
        if (cost > this.mallDiamond) {
          this.notify('钻石不足：需要 ' + cost + ' 钻石，当前余额 ' + this.mallDiamond +
            (it.dual_pay ? '（可改用黄金购买）' : '（钻石仅可由管理员充值）'))
          return
        }
      } else {
        const cost = (it.price_gold || 0) * n
        if (cost > (this.city.gold || 0)) {
          this.notify('黄金不足：需要 ' + cost + '，当前 ' + (this.city.gold || 0))
          return
        }
      }
      // ★★ 2026-10-08 连点防抖（用户反馈「秒点一次购买能到多个」）：
      //   一次点击 = 一次购买 —— 请求未返回前忽略后续点击（与交易行 doExchangeBuy 同款）。
      //   后端同时加了条件原子扣款/扣库存（ezfySpendGold / ezfySpendDiamond / ezfySpendStock），
      //   这里只是第一道闸，防止手抖/连点一次打出 N 个请求。
      this.once('mallBuy' + it.id, () => api.post('/games/ezfy/mall/buy',
        { cfg_id: it.id, count: n, pay_with: payWith }).then(r => {
          if (r.code === 0) {
            this.notify(r.data && r.data.msg ? r.data.msg : '购买成功')
            this.buyItem = null
            this.load()
            this.loadBag()
            // ★ 购买成功后返回商城（go('mall') 会自动刷新商城数据）
            this.go('mall')
          } else this.notify(r.msg || '购买失败')
        }))
    },
    needOfficer (it) {
      // ★ 19 = 星级徽章，也要选军官（漏了它会没有「军官:」下拉，玩家没法用）
      return it.item_type === 10 || it.item_type === 11 || it.item_type === 12 || it.item_type === 19
    },
    openUse (it) {
      this.useItem = it
      this.useCount = 1
      this.useOfficerId = 0
    },
    doUse (it) {
      const body = { cfg_id: it.cfg_id, count: parseInt(this.useCount) || 1, city_id: this.city.id }
      if (this.needOfficer(it)) {
        if (!this.useOfficerId) { this.notify('请先选择要使用的军官'); return }
        body.officer_id = this.useOfficerId
      }
      // ★ 2026-10-25 技能书（item_type 11）= 随机学一个技能，不再传 skill_id（后端也会忽略）
      // ★★ 2026-10-25 连点防抖（用户反馈「部分道具 瞬间多点 能用好多次」）：
      //   ① opGuard：1 秒账号级卡控，localStorage 让**所有 tab 共享同一把锁**
      //      （与后端 UseItem 的 1 秒卡控同口径，加速类后端是 3 秒）；
      //   ② once：请求还没返回时忽略后续点击（一次点击 = 一次使用）。
      //   ⚠️ 别只留一个：opGuard 防「点完等一下再点」，once 防「同一瞬间连点」。
      if (!this.opGuard('bagUse', 1000)) return
      this.once('bagUse', () => api.post('/games/ezfy/bag/use', body).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '使用成功')
          this.useItem = null
          this.loadBag()
          this.load()
          // ★ 2026-10-22 征募告示会改变当前城市队列上限，立刻刷新建筑数据
          if (it.item_type === 33) this.loadBuildings()
        } else this.notify(r.msg || '使用失败')
      }))
    },
    doExchangeSell () {
      // ★ 2026-10-06 连点防抖：一次点击 = 一次挂单（请求未返回时忽略后续点击）
      this.once('exSell', () => api.post('/games/ezfy/exchange/sell', {
        es_type: parseInt(this.sellType), es_count: parseInt(this.sellCount) || 0,
        total_price: parseInt(this.sellPrice) || 0
      }).then(r => this.alert(r, '挂单已发布', () => {
        this.sellCount = 0
        this.sellPrice = 0
        this.exchangeMPage = 1
        this.loadExchange()
      })))
    },
    doExchangeBuy (e) {
      // ★ 2026-10-06 连点防抖（同一张挂单）
      this.once('exBuy' + e.id, () => api.post('/games/ezfy/exchange/buy', { id: e.id }).then(r => this.alert(r, '购买成功', () => this.loadExchange())))
    },
    doExchangeCancel (e) {
      // ★ 2026-10-06 连点防抖（同一张挂单）
      this.once('exCancel' + e.id, () => api.post('/games/ezfy/exchange/cancel', { id: e.id }).then(r => this.alert(r, '挂单已撤销', () => this.loadExchange())))
    },
    // ★ 2026-10-22 今日该资源已售数量 / 剩余可售（上限 0 = 不限）
    sysSellUsedCount () {
      return Number((this.sysSellUsed || {})[this.sellSysType] || 0)
    },
    sysSellRemain () {
      const limit = Number(this.sysSellDailyLimit) || 0
      if (limit <= 0) return -1
      const remain = limit - this.sysSellUsedCount()
      return remain > 0 ? remain : 0
    },
    // ★ 2026-09-30 向系统出售资源：预览应得黄金（与后端同口径整数除法）
    sysSellPreview () {
      const count = parseInt(this.sellSysCount) || 0
      const ratio = this.sysSellRatio[this.sellSysType] || 0
      if (count <= 0 || !ratio) return 0
      const base = Math.floor(count * ratio / 100)
      const fee = this.sysSellFee || 0
      return Math.floor(base * (100 - fee) / 100)
    },
    doExchangeSysSell () {
      // ★ 2026-10-06 连点防抖：一次点击 = 一次向系统出售
      this.once('exSysSell', () => {
        this.sysSellLostWarn = ''
        return api.post('/games/ezfy/exchange/sys-sell', {
          es_type: parseInt(this.sellSysType), es_count: parseInt(this.sellSysCount) || 0
        }).then(r => {
          if (r.code === 0 && r.data && r.data.gold_lost) {
            this.sysSellLostWarn = '黄金累加超过黄金上限，超出 ' + (r.data.lost_gold || 0) + ' 已丢失'
          }
          this.alert(r, '已售出', () => {
            this.sellSysCount = 0
            this.exchangeMPage = 1
            this.loadExchange()
          })
        })
      })
    },
    // ---- 任务/福利 ----
    doAward (t) {
      api.post('/games/ezfy/tasks/award', { task_id: t.id }).then(r => {
        // ★ 领奖成功后本地把状态置为「已领取」，否则按钮一直停在 [领奖]（用户反馈的 bug）
        if (r && r.code === 0) t.status = 2
        this.alert(r, '奖励已领取')
      })
    },
    doSign () {
      // ★ 签到后必须重新拉 welfare 与资源，否则：
      //   ① 首页「每日签到」还显示「签到」（应为「已签到」）—— 用户反馈的 bug
      //   ② 资源数字不刷新，看起来像「签到后资源没加/反而少了」
      api.post('/games/ezfy/welfare/sign', {}).then(r => {
        this.alert(r, '签到成功', () => {
          this.loadWelfare()
          this.load()
          if (this.cur === 'builds') this.loadRes()
        })
      })
    },
    doGift (t) {
      api.post('/games/ezfy/welfare/gift/' + t, {}).then(r => this.alert(r, '礼包已领取'))
    },
    setWelfareTab (n) {
      this.welfareTab = n
    },
    doTreasureSign () {
      // ★ 2026-09-28 宝物签到：领完刷新 welfare（更新连续天数与今日已签），宝物即时入背包
      api.post('/games/ezfy/welfare/treasure-sign', {}).then(r => {
        this.alert(r, '宝物签到成功', () => {
          this.loadWelfare()
        })
      })
    },
    rewardText (rw) {
      if (!rw) return ''
      const parts = []
      if (rw.gold) parts.push('金' + rw.gold)
      if (rw.food) parts.push('粮' + rw.food)
      if (rw.steel) parts.push('钢' + rw.steel)
      if (rw.oil) parts.push('油' + rw.oil)
      if (rw.rare) parts.push('稀' + rw.rare)
      if (rw.prestige) parts.push('声望' + rw.prestige)
      return parts.join(' ')
    },
    woundedList (type) {
      return (this.troopsData.wounded || []).filter(w => w.type === type)
    },
    // ★ 恢复/召回全部伤兵需要的黄金合计（单价由后端下发 heal_gold）
    woundedHealCost (type) {
      return this.woundedList(type).reduce((s, w) => s + (w.heal_gold || 0) * (w.count || 0), 0)
    },
    troopTypeName (t) {
      return { 1: '海军', 2: '陆军', 3: '空军', 4: '城防' }[t] || '部队'
    },
    // 城内某兵种数量(军工厂页显示 兵种:数量)
    troopCount (tid) {
      const t = (this.troopsData.troops || []).find(x => x.troop_id === tid)
      return t ? t.count : 0
    },
    // ★ 2026-10-05 修复「建筑/科技倒计时不自动减少」：tick 参数 = 父组件每秒刷新的
    //   gatherNow（响应式），子组件模板里把 ezfy.gatherNow 传进来 → 子组件每秒随父组件
    //   时钟重渲染，倒计时实时走动（原来只调 ezfy.remain() 无响应式依赖，子组件不重渲染）。
    remain (endTime, tick) {
      if (!endTime) return ''
      const ms = endTime - (tick || Date.now())
      if (ms <= 0) return '已完成'
      const s = Math.floor(ms / 1000)
      if (s < 60) return s + '秒'
      const m = Math.floor(s / 60)
      if (m < 60) return m + '分' + (s % 60) + '秒'
      const h = Math.floor(m / 60)
      return h + '时' + (m % 60) + '分'
    },
    // ★ 数字千分位（军队动态的「待带回」和出征页的「本次出兵/上限」都用它）
    //   之前模板里引用了 fmtN 但方法从未定义 → Vue 渲染直接抛
    //   "TypeError: _vm.fmtN is not a function"，整页白掉，且只在对应分支被渲染时才暴露。
    // ★ 2026-09-28 首页头部产量缩写：≥1万 时按「万」缩成紧凑形式, 保留负号; 不足1万显示原值
    //   采用向下取整(截断), 不四舍五入——保证展示值≤真实值, 玩家不会被夸大误导
    //   例如 -46393790 → -4639万, 113840 → 11.3万, 54566 → 5.4万
    // ★ 2026-10-05 ：达到「亿」(≥1e8) 就换成「亿」单位, 比 4639万 更好读；
    //   首页的**现有资源**(city.*)和**每小时产量**(resProd.*)都走这个函数, 一起生效。
    //   同样向下取整(截断)、保留 1 位小数(千万位截断)。例如 123456789 → 1.2亿, 543000000 → 5.4亿
    // ★ 2026-10-10 新首页布局资源图标（5 种资源，与老布局同一套 20×20 图形）
    resIcon (k) {
      const ico = {
        gold: '<svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-linecap="round" stroke-linejoin="round" fill="none"><path d="M6.4 5.8 L10 10.3 L13.6 5.8" stroke-width="1.6"/><path d="M10 6 V14.2" stroke-width="1.6"/><path d="M7.6 8.9 H12.4" stroke-width="1.4"/><path d="M7.6 11.7 H12.4" stroke-width="1.4"/></g></svg>',
        food: '<svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-width="1.5" stroke-linecap="round" fill="none"><path d="M7.2 6.2 C6.7 5.4 7.3 4.1 7.9 3.5"/><path d="M12.8 6.2 C13.3 5.4 12.7 4.1 12.1 3.5"/></g><path d="M5 8.6 H15 C15 8.6 14.7 12.2 13.3 13.7 C12.2 14.8 10.9 15.4 10 15.4 C9.1 15.4 7.8 14.8 6.7 13.7 C5.3 12.2 5 8.6 5 8.6 Z" fill="#FFD700"/></svg>',
        steel: '<svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M4.2 8.4 L6.6 15.6 H13.4 L15.8 8.4 Z" fill="#FFD700"/><path d="M5.6 10.2 H14.4" stroke="#C9AFF0" stroke-width="1.3" stroke-linecap="round"/></svg>',
        oil: '<svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 3.6 C10 3.6 6.1 8.1 6.1 11.2 C6.1 13.5 7.8 15.3 10 15.3 C12.2 15.3 13.9 13.5 13.9 11.2 C13.9 8.1 10 3.6 10 3.6 Z" fill="#FFD700"/><circle cx="8.6" cy="11.4" r="0.9" fill="#C9AFF0"/></svg>',
        rare: '<svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M5.8 6.4 H14.2 L11.7 9.2 L10 15.6 L8.3 9.2 Z" fill="#FFD700"/><path d="M5.8 6.4 H10 L8.3 9.2 Z" fill="#C9AFF0"/></svg>'
      }
      return ico[k] || ''
    },
    fmtProd (n) {
      const v = Number(n)
      const neg = v < 0
      const a = Math.abs(v)
      if (a >= 100000000) {
        const y = Math.floor(a / 100000000)
        const rem = a % 100000000
        // 非整整亿但剩余≥千万时补 1 位小数(百万位截断), 否则整数亿
        const s = rem >= 10000000 ? (y + '.' + Math.floor(rem / 10000000)) : String(y)
        return (neg ? '-' : '') + s + '亿'
      }
      if (a >= 10000) {
        const w = Math.floor(a / 10000)
        const rem = a % 10000
        // 非整整万但剩余≥千时补 1 位小数(百位截断), 否则整数万
        let s = ''
        if (rem >= 1000) {
          s = w + '.' + Math.floor(rem / 1000) // 万分位→十分之一万, 只取1位(截断)
        } else {
          s = String(w)
        }
        return (neg ? '-' : '') + s + '万'
      }
      return (neg ? '-' : '') + a.toLocaleString('en-US')
    },
    fmtN (n) {
      const v = Number(n)
      if (!isFinite(v)) return '0'
      return v.toLocaleString('en-US')
    },
    // ★ 宝箱奖池/开箱结果的品质着色（普通/稀有/史诗/传说，给不同颜色区分）
    // ★ 军官装备品质同套配色：初级→灰 / 中级→蓝 / 高级→紫 / 特殊→橙（一眼看出哪个好）
    qualityClass (q) {
      if (q === '传说' || q === '传奇' || q === '特殊') return 'q-legend'
      if (q === '史诗' || q === '高级') return 'q-epic'
      if (q === '稀有' || q === '中级') return 'q-rare'
      return 'q-normal'
    },
    // ★ 把后端下发的创建时间格式化成年-月-日（公告标题后的发布时间）。
    //   入参可能是 "2026-09-23T11:11:28+08:00" 或已是 "2006-01-02 15:04" 字符串。
    fmtDate (s) {
      if (!s) return ''
      // 取 ISO 字符串最前面一段 yyyy-MM-dd；兼容已格式化的 "2006-01-02 ..."
      const m = String(s).match(/(\d{4})-(\d{2})-(\d{2})/)
      return m ? m[1] + '-' + m[2] + '-' + m[3] : ''
    },
    // ★ 大数加单位（万/亿），资源详情里动辄十几位数字，不缩一下没法看
    fmtBig (n) {
      const v = Number(n || 0)
      if (!isFinite(v)) return '0'
      const abs = Math.abs(v)
      const sign = v < 0 ? '-' : ''
      if (abs >= 1e8) return sign + (abs / 1e8).toFixed(2).replace(/\.?0+$/, '') + '亿'
      if (abs >= 1e4) return sign + (abs / 1e4).toFixed(2).replace(/\.?0+$/, '') + '万'
      return String(v)
    },
    fmtTime (t) {
      if (!t) return ''
      let d
      if (typeof t === 'number') {
        d = new Date(t < 1e12 ? t * 1000 : t)
      } else {
        // 后端时间可能是 ISO8601(带时区, 如 2026-09-18T21:15:25.054+08:00)
        // 或 "YYYY-MM-DD HH:mm:ss"; 后者在部分浏览器需把 - 换成 /
        d = new Date(t)
        if (isNaN(d.getTime())) d = new Date(String(t).replace(/-/g, '/'))
      }
      if (isNaN(d.getTime())) return ''
      const p = n => String(n).padStart(2, '0')
      // ★ 战报时间要带年份（原来是 MM-DD HH:mm:ss，跨年就分不清）
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) +
        ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    // ★ 2026-10-22 聊天频道（世界/军团/系统）时间只显示到分钟，不显示秒。
    //   战报/邮件等仍用 fmtTime（需要秒级时间）；这里只做聊天展示裁剪。
    fmtChatTime (t) {
      const s = this.fmtTime(t)
      return s ? s.slice(0, 16) : ''
    },
    // ---- 页面内提示 / 确认（替代 alert / confirm / prompt）----
    notify (text, type) {
      const t = String(text === undefined || text === null ? '' : text)
      if (!t) return
      this._msgSeq = (this._msgSeq || 0) + 1
      const id = this._msgSeq
      this.msgs.push({ id: id, text: t, type: type || this.guessMsgType(t) })
      if (this.msgs.length > 6) this.msgs.shift()
      // ★ 2026-09-27 不需要玩家确定的提示 3 秒→1.5 秒后自动消失（也可点 [关闭] 手动收起）
      setTimeout(() => this.closeMsg(id), 1500)
    },
    // 提示条样式：fixed 定位在刚才点击的**控件正下方居中**（留 12px 间隙），
    // 不再用原始鼠标坐标，避免盖住按钮本身。元素已滚出视口时兜底顶部居中。
    // 多条时向下错开 36px，防重叠。
    msgStyle (m, i) {
      const vw = window.innerWidth
      const vh = window.innerHeight
      const w = Math.min(320, vw - 16)
      const step = 36
      let left, top
      const el = this._lastClicked
      const rect = el && typeof el.getBoundingClientRect === 'function' ? el.getBoundingClientRect() : null
      // 元素在视口内（横向不越界、纵向可见）才按它定位；否则退到顶部居中
      const inView = rect && rect.left < vw && rect.right > 0 && rect.top < vh && rect.bottom > 0
      if (rect && inView) {
        let cx = rect.left + rect.width / 2
        left = Math.round(cx - w / 2)
        left = Math.max(8, Math.min(left, vw - w - 8))
        let below = rect.bottom + 12 + i * step
        if (below + 40 > vh) {
          // 元素靠底：改成出现在元素上方
          top = Math.max(8, rect.top - 12 - 40)
        } else {
          top = Math.round(below)
        }
      } else {
        left = Math.round((vw - w) / 2)
        top = 8
      }
      return { position: 'fixed', left: left + 'px', top: top + 'px',
        maxWidth: w + 'px', zIndex: 9999, boxShadow: '0 2px 8px rgba(0,0,0,.25)' }
    },
    guessMsgType (t) {
      if (/失败|不足|错误|不能|无法|没有|请先|需要/.test(t)) return 'error'
      if (/成功|已|完成/.test(t)) return 'ok'
      return 'info'
    },
    closeMsg (id) {
      this.msgs = this.msgs.filter(m => m.id !== id)
    },
    // 内联确认：返回 true/false；带输入框时返回输入的字符串或 null（取消）
    ask (text, opts) {
      opts = opts || {}
      return new Promise(resolve => {
        this._askResolve = resolve
        this.askBox = {
          show: true, text: String(text || ''), input: !!opts.input,
          placeholder: opts.placeholder || '', value: opts.value || ''
        }
        // 渲染后量一次真实高度给 askStyle 用（带输入框的会高不少）
        this.askH = 0
        this.$nextTick(() => {
          const el = this.$refs.askBox
          if (el && el.offsetHeight) this.askH = el.offsetHeight
        })
      })
    },
    askConfirm (ok) {
      const box = this.askBox
      const r = this._askResolve
      this._askResolve = null
      this.askBox = { show: false, text: '', input: false, placeholder: '', value: '' }
      if (!r) return
      if (!ok) { r(box.input ? null : false); return }
      r(box.input ? box.value : true)
    },
    // 统一的接口结果提示（原来弹 alert，现在落到页面消息区）
    // ★ after：成功后要额外刷新的数据（如军团信息）。只刷 /view 不够 ——
    //   军团数据来自 /games/ezfy/corps/*，不重新拉就会「加入后还显示未加入」。
    //
    // ★ 「提示 ok 改成具体的描述，比如领取就是领取成功，不知道的就是操作成功」：
    //   所以每个调用点都要传 fallback（领取成功 / 购买成功 / 建造命令已下达 …），
    //   只有真的无从判断时才回落「操作成功」。
    alert (r, fallback, after) {
      if (r && r.code === 0) {
        const m = (r.data && r.data.msg) ? r.data.msg : (fallback || '操作成功')
        this.notify(m, 'ok')
        this.load()
        if (typeof after === 'function') after()
      } else {
        this.notify((r && r.msg) ? r.msg : '操作失败', 'error')
      }
    },
    // ★ 用户反馈「连点会出现多条」→ 「一次点击 = 一条命令」的操作统一走这里防抖：
    //   同一个 key 的请求还没返回时，后续点击直接忽略（不会重复下单）。
    //   fn 需要返回 Promise（api.post/get 都是），请求结束（无论成败）自动解锁。
    once (key, fn) {
      if (!this._onceMap) this._onceMap = {}
      if (this._onceMap[key]) return
      this._onceMap[key] = true
      const release = () => { this._onceMap[key] = false }
      let ret
      try {
        ret = fn()
      } catch (e) {
        release()
        throw e
      }
      if (ret && typeof ret.then === 'function') ret.then(release, release)
      else release()
    },
    // ★★ 2026-10-15 账号级 3 秒卡控（防「一个账号开多个 tab 连点刷资源/刷兵」）：
    //   冷却时间戳写进 localStorage —— 同一浏览器 / 同一账号下**所有 tab 共享同一把锁**，
    //   从源头把「多 tab 同时连点加速/采集/收获/召回」卡死；内存同时留一份兜底（浏览器禁 localStorage 时）。
    //   返回 true 放行 / false 拦截（拦截时已 notify 提示剩余秒数）。
    opGuard (key, ms) {
      const gap = (ms && ms > 0) ? ms : 3000
      const now = Date.now()
      if (!this._opGuardAt) this._opGuardAt = {}
      let last = this._opGuardAt[key] || 0
      const uid = (this.$store.state.ezfyUser && this.$store.state.ezfyUser.id) || 0
      const lsKey = 'ezfy_guard_' + uid + '_' + key
      try {
        const ls = parseInt(window.localStorage.getItem(lsKey), 10) || 0
        if (ls > last) last = ls
      } catch (e) {}
      if (now - last < gap) {
        const left = Math.max(1, Math.ceil((gap - (now - last)) / 1000))
        this.notify('操作过于频繁, 请 ' + left + ' 秒后再试', 'error')
        return false
      }
      try { window.localStorage.setItem(lsKey, String(now)) } catch (e) {}
      this._opGuardAt[key] = now
      return true
    },
    // ---- 军官/学院 ----
    loadAcade () {
      // ★ 2026-10-04 军官模块 tab 数据「缓存先行 + 后台刷新」：进页打当前 tab，
      //   其余 5 个 tab 的数据并行预取，之后点哪个 tab 都是秒开。
      this.loadAcadeOfficers()
      this.loadAcadeTab()
      this.prefetchAcadeTabs()
      // ★ 2026-10-13 军校页要显示「招生简章」持有量 → 进页顺带拉一次背包
      this.loadBag()
    },
    loadAcadeTab () {
      if (this.acadeTab === 'search') this.loadRecruit()
      else if (this.acadeTab === 'skill') this.loadAcadeSkills()
      else if (this.acadeTab === 'equip') this.loadAcadeEquip()
      else if (this.acadeTab === 'scheme') this.loadSchemes()
      else if (this.acadeTab === 'generals') this.loadAcadeGenerals()
      else this.loadAcadeOfficers()
    },
    // 军官 / 任命市长 / 战俘营 共用同一份军官列表数据
    loadAcadeOfficers () {
      this.withAcadeCache('officers', () => api.get('/games/ezfy/officers'), d => { this.officerData = d })
    },
    // ★ 2026-10-04 缓存优先加载：命中先渲染缓存（切换秒开），后台刷新成功后覆盖。
    //   同一 key 的请求去重（_acadeLoading），连点不会重复打接口。
    withAcadeCache (key, fetcher, setter) {
      // ★ _ 开头属性 Vue2 不代理 data()，必须懒初始化（与 _onceMap 同模式）
      if (!this._acadeCache) this._acadeCache = {}
      if (!this._acadeLoading) this._acadeLoading = {}
      const apply = d => { this._acadeCache[key] = d; setter(d) }
      if (this._acadeCache[key]) {
        setter(this._acadeCache[key])
        if (!this._acadeLoading[key]) {
          this._acadeLoading[key] = true
          fetcher().then(r => { if (r && r.code === 0) apply(r.data) })
            .catch(() => {}).then(() => { this._acadeLoading[key] = false })
        }
        return
      }
      if (this._acadeLoading[key]) return
      this._acadeLoading[key] = true
      fetcher().then(r => { if (r && r.code === 0) apply(r.data) })
        .catch(() => {}).then(() => { this._acadeLoading[key] = false })
    },
    // ★ 2026-10-04 进军官页时把其余 tab 的数据后台预取（只补没有缓存的 key，
    //   已缓存的交给各自 tab 的「后台刷新」保新鲜），切换零等待。
    prefetchAcadeTabs () {
      const jobs = [
        ['recruit', () => api.get('/games/ezfy/acade/recruit'), d => { this.recruitData = d }],
        ['skill', () => api.get('/games/ezfy/officers/skills'), d => { this.skillData = d }],
        ['equip', () => { this.loadEquipSets(); return api.get('/games/ezfy/officers/equipments') }, d => { this.equipData = d }],
        ['scheme', () => api.get('/games/ezfy/schemes'), d => { this.schemeData = d }],
        ['generals', () => api.get('/games/ezfy/officers/generals'), d => { this.generalData = d }]
      ]
      for (const [key, fetcher, setter] of jobs) {
        if (!this._acadeCache[key] && !this._acadeLoading[key]) this.withAcadeCache(key, fetcher, setter)
      }
    },
    switchAcade (tab) {
      this.acadeTab = tab
      // ★ 进装备页时把检索/分页复位，避免上次的搜索词把列表筛空（看着像「装备没了」）
      if (tab === 'equip') {
        this.equipTab = 'my'
        this.equipWord = ''
        this.equipPage = 1
        this.equipAllWord = ''
        this.equipAllPage = 1
      }
      // ★ 2026-10-04 全部走缓存优先加载：军官/市长/战俘营共用 officerData
      if (tab === 'officer' || tab === 'mayor' || tab === 'captive') this.loadAcadeOfficers()
      else if (tab === 'search') this.loadRecruit()
      else if (tab === 'skill') this.loadAcadeSkills()
      else if (tab === 'equip') this.loadAcadeEquip()
      else if (tab === 'scheme') this.loadSchemes()
      else if (tab === 'generals') this.loadAcadeGenerals()
    },
    // ★ 计谋列表（配置由后端下发，含持有信号弹数量）
    loadSchemes () {
      this.withAcadeCache('scheme', () => api.get('/games/ezfy/schemes'), d => { this.schemeData = d })
    },
    // ★ 秘密策反：预计降低忠诚（学识 ÷ 10，最高 99，至少 1）—— 与后端同口径，仅用于前端提示
    schemeSecretDrop (learning) {
      const drop = Math.floor((Number(learning) || 0) / 10)
      return Math.min(99, Math.max(1, drop))
    },
    async doScheme (s) {
      const b = this.schemeData
      if (b.bullet_have < s.bullet) { this.notify('「' + b.bullet_name + '」不足'); return }
      const body = { scheme_id: s.id }
      const coordKind = s.kind === 1 || s.kind === 6
      if (coordKind) {
        const x = parseInt(this.schemeX)
        const y = parseInt(this.schemeY)
        if (!x || !y) { this.notify('「' + s.name + '」需要填写目标城市坐标（x / y）'); return }
        body.target_x = x
        body.target_y = y
      }
      if (s.kind === 6) {
        const officer = this.officerDetail && this.officerDetail.officer
        if (!officer || !officer.id) { this.notify('请先选择用计军官'); return }
        body.officer_id = officer.id
      }
      const tip = s.kind === 6
        ? ('确认对 (' + body.target_x + ',' + body.target_y + ') 发动「' + s.name + '」吗？消耗 ' + b.bullet_name + '×' + s.bullet + '，将降低目标城守忠诚（最低保留 1 点）')
        : coordKind
          ? ('确认对 (' + body.target_x + ',' + body.target_y + ') 发动「' + s.name + '」吗？消耗 ' + b.bullet_name + '×' + s.bullet)
          : ('确认发动「' + s.name + '」吗？消耗 ' + b.bullet_name + '×' + s.bullet)
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/scheme/use', body).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '发动失败'); return }
        this.notify(r.msg || '计谋已发动')
        delete this._acadeCache['scheme'] // ★ 信号弹数量变了, 失效计谋缓存
        this.loadSchemes()
        this.loadBag()
        this.load()
      })
    },
    loadRecruit () {
      this.withAcadeCache('recruit', () => api.get('/games/ezfy/acade/recruit'), d => { this.recruitData = d })
    },
    loadAcadeSkills () {
      this.withAcadeCache('skill', () => api.get('/games/ezfy/officers/skills'), d => { this.skillData = d })
    },
    loadAcadeEquip () {
      this.loadEquipSets()
      this.withAcadeCache('equip', () => api.get('/games/ezfy/officers/equipments'), d => { this.equipData = d })
    },
    // ★ 2026-09-25：全部套装配置（含加成/部位/我拥有几件）—— 只拉一次，各页共用。
    //   装备页/商城页/军官装备页都要「set_id → 这套穿齐给什么」，所以做成缓存。
    loadEquipSets () {
      if (this.allSets && this.allSets.length) return Promise.resolve()
      return api.get('/games/ezfy/equipsets').then(r => {
        if (r.code === 0) this.allSets = r.data.sets || []
      }).catch(() => {})
    },
    // set_id → 套装配置（没有/散件返回 null）
    setOf (setId) {
      return (setId > 0 && this.setMap[setId]) ? this.setMap[setId] : null
    },
    // 套装加成文案（复用装备属性的展示口径：六项战斗属性 + 军事/后勤/学识）
    setBonusText (setId) {
      const s = this.setOf(setId)
      return s ? this.equipAttrText(s) : ''
    },
    // 点装备名 / 套装名 → 在**这一行**下面展开详情卡；再点同一个收起。
    // ★ 参数 equipId 是**这一行装备的 id**，不是 set_id：同一套会有 9~11 行，
    //   按 set_id 展开的话每行都会各蹦一张一样的卡（实测 10 件套蹦 10 张）。
    // ★ 2026-09-25 「名称点击查看该装备的加成，点击套装显示套装的加成」：
    //   两个入口看**不同**的内容，用 mode 区分（'item' 只看这件自己的加成 / 'set' 只看套装加成）。
    //   同一个入口点两次 = 收起；点另一个入口 = 直接换成另一份内容（不收起）。
    toggleDetail (equipId, mode) {
      if (!equipId) return
      const m = mode || 'item'
      if (this.detailRowId === equipId && this.detailMode === m) {
        this.detailRowId = 0
        this.detailMode = ''
        return
      }
      this.detailRowId = equipId
      this.detailMode = m
    },
    loadAcadeGenerals () {
      this.withAcadeCache('generals', () => api.get('/games/ezfy/officers/generals'), d => { this.generalData = d })
    },
    openOfficer (id) {
      this.cur = 'officerdetail'
      // ★ 换军官时把详情页签、背包装备的检索/分页复位
      this.officerDetailTab = 'attr'
      this.officerBagWord = ''
      this.officerBagPage = 1
      // ★ 2026-10-10 切换军官 → 清掉旧军官的懒加载背包（响应「切换哪个获取哪个」）
      this.officerBag = []
      this.officerBagSets = []
      this.officerBagFor = 0
      this.officerBagLoaded = 0
      // ★ 2026-10-01 修复「点击军官有时候空白」：进页先清掉旧军官数据/错误，
      //   加载完成前显示「加载中...」，避免残留上一名军官的详情或白屏
      this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
      this.officerDetailError = ''
      this.loadOfficerDetail(id)
    },
    // ★ 2026-10-10 性能（详情降载）：背包装备按需拉取，同一军官只拉一次。
    //   装备背包 / 装备(一键穿套装) / 赏赐宝物 三处共用这一份懒加载数据。
    async ensureOfficerBag () {
      const o = this.officerDetail && this.officerDetail.officer
      if (!o || !o.id) return
      // 已对当前军官拉好 → 直接复用，不再重复请求
      if (this.officerBagLoaded && this.officerBagFor === o.id) return
      const id = o.id
      this.officerBag = []
      this.officerBagSets = []
      this.officerBagLoaded = 0
      try {
        const r = await api.get('/games/ezfy/officers/' + id + '/backpack')
        // 可能已切换军官 → 丢弃这次结果，避免串数据
        if (!this.officerDetail || !this.officerDetail.officer || this.officerDetail.officer.id !== id) return
        if (r.code === 0) {
          this.officerBag = (r.data && r.data.bag) || []
          this.officerBagSets = (r.data && r.data.bag_sets) || []
        } else {
          this.notify(r.msg || '背包装备加载失败, 请重试')
        }
      } catch (e) { /* 网络异常：保持为空，不打扰，切回来会重试 */ }
      this.officerBagLoaded = 1
      this.officerBagFor = id
    },
    // ★ 2026-10-10 穿/卸装备后背包会变 → 失效缓存并静默重拉一次（界面就地刷新）
    refreshOfficerBag () {
      this.officerBagLoaded = 0
      this.officerBagFor = 0
      this.ensureOfficerBag()
    },
    // ★★ 2026-10-19 静默重拉背包（不清空旧数据，避免列表闪一下空白）：
    //   赏赐宝物这类「只减一件」的操作后，用它把数量校准到服务端真实值。
    //   与 ensureOfficerBag 的区别：后者会先 `officerBag = []`（首次加载用），
    //   这里保留现有行，请求回来整体替换。
    async reloadOfficerBag () {
      const o = this.officerDetail && this.officerDetail.officer
      if (!o || !o.id) return
      const id = o.id
      try {
        const r = await api.get('/games/ezfy/officers/' + id + '/backpack')
        // 期间切了军官 → 丢弃这次结果，避免串数据
        if (!this.officerDetail || !this.officerDetail.officer || this.officerDetail.officer.id !== id) return
        if (r.code === 0) {
          this.officerBag = (r.data && r.data.bag) || []
          this.officerBagSets = (r.data && r.data.bag_sets) || []
          this.officerBagLoaded = 1
          this.officerBagFor = id
        }
      } catch (e) { /* 网络异常：保留本地乐观更新的结果 */ }
    },
    // ★ 2026-10-10 打开军官详情页签：装备/装备背包 需要背包数据 → 先懒加载再切页
    // ★ 2026-10-19 计谋 tab 从军校页搬到军官详情 → 切到它时按需拉计谋配置/信号弹持有量
    //   （军校页那个入口已删，所以这里必须自己保证数据到位，否则进 tab 是空列表）
    openOfficerDetailTab (tab) {
      if ((tab === 'equip' || tab === 'bag') && !(this.officerBagLoaded && this.officerBagFor === (this.officerDetail && this.officerDetail.officer && this.officerDetail.officer.id))) {
        this.ensureOfficerBag()
      }
      if (tab === 'scheme') this.loadSchemes()
      this.officerDetailTab = tab
    },
    loadOfficerDetail (id) {
      // ★ 2026-10-01 防串数据：连续点多名军官时，只认最后一次请求的结果
      const seq = (this._officerDetailSeq = (this._officerDetailSeq || 0) + 1)
      // ★ 军官详情的「已穿戴装备 / 装备背包」要显示套装加成 → 一并把套装配置拉上
      this.loadEquipSets()
      api.get('/games/ezfy/officers/' + id).then(r => {
        if (seq !== this._officerDetailSeq) return
        if (r.code === 0) {
          this.officerDetailError = ''
          this.officerDetail = r.data
          // ★ 2026-10-24 怀旧军官详情：加载后同步名称输入框（改名直接用输入框的值）
          this.officerRenameInput = (r.data && r.data.officer && r.data.officer.name) || ''
          // ★ 2026-10-04 兜底：接口成功但没带回军官（异常/旧后端）→ 明确提示，
          //   别让页面永远停在「加载中...」
          if (!r.data || !r.data.officer) {
            this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
            this.officerDetailError = '军官数据加载异常，请返回列表重试'
          }
        } else {
          // 加载失败（如武将已调往别的城市 / 已被流放）→ 不再静默空白：
          // 提示原因并刷新军官列表，让列表与后端状态一致
          this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
          this.officerDetailError = r.msg || '加载军官详情失败, 请重试'
          this.loadAcade()
        }
      }).catch(() => {
        if (seq !== this._officerDetailSeq) return
        this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
        this.officerDetailError = '网络开小差了, 请稍后重试'
      })
    },
    // 军校直接使用招生简章刷新（不占每小时次数；不用跳背包）
    async doUseRecruitTicket () {
      if (!this.recruitData.academy_level) { this.notify('需要先建造军校'); return }
      await this.loadBag() // 持有量按名字统计, 先拉一次背包保证准确
      if (this.recruitTicketCount() <= 0) { this.notify('没有「招生简章」，可到商城购买'); return }
      if (!await this.ask('确认使用「招生简章」×1 刷新军校候选名将吗？（不占用每小时次数）')) return
      api.post('/games/ezfy/acade/recruit/ticket', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          delete this._acadeCache['recruit'] // ★ 候选已重刷, 失效招募缓存（与 [刷新] 同口径）
          this.loadAcade()
          this.loadBag()
        } else this.notify(r.msg)
      })
    },
    // 背包里某道具的数量
    bagCount (cfgId) {
      const it = (this.bagItems || []).find(x => x.cfg_id === cfgId)
      return it ? it.count : 0
    },
    // ★ 2026-10-13 招生简章持有量：按名字统计（道具 cfg_id 已全体迁到 1001+，
    //   老号 13 查不到 → 持有量恒 0 → 「背包有却提示没有」）
    recruitTicketCount () {
      const it = (this.bagItems || []).find(x => x.name === '招生简章')
      return it ? it.count : 0
    },
    doRefreshRecruit () {
      api.post('/games/ezfy/acade/recruit/refresh', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '刷新失败')
        delete this._acadeCache['recruit'] // ★ 候选已重刷, 失效招募缓存
        this.loadRecruit()
      })
    },
    async doRecruit (g) {
      if (!await this.ask('确定雇佣 ' + g.name + ' 吗? 需要 ' + g.cost + ' ' + this.resNames.gold)) return
      api.post('/games/ezfy/acade/recruit/hire', { key: g.key }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '雇佣失败')
        delete this._acadeCache['recruit'] // ★ 候选被领走 + 军官数变化, 失效招募/军官缓存
        delete this._acadeCache['officers']
        this.loadRecruit()
        this.loadAcade()
      })
    },
    // ★ 2026-09-28 洗点入口放到军官详情页（原来要背包里翻「军官洗点卡」使用）。
    //   对当前军官直接消耗 1 张洗点卡；后端 case 12 兜底校验（出征中/无卡等）。
    async doRespec () {
      const o = this.officerDetail.officer
      if (!o) return
      await this.loadBag() // 军官详情页不常驻背包数据，先拉一次再判断持有量
      const card = this.bagItems.find(x => x.item_type === 12)
      if (!card) { this.notify('背包没有「军官洗点卡」，可在商城购买'); return }
      if (!await this.ask('确认对 ' + o.name + ' 使用「军官洗点卡」×1 吗？\n属性将重置为军官池初始属性，已分配的点退回待分配点（等级/经验/技能保留）')) return
      // ★ 2026-10-25 连点防抖：与背包「使用」共用同一把 1 秒锁（后端 UseItem 也是 1 秒）
      if (!this.opGuard('bagUse', 1000)) return
      this.once('bagUse', () => api.post('/games/ezfy/bag/use', { cfg_id: card.cfg_id, count: 1, city_id: this.city.id, officer_id: o.id }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '洗点完成')
          this.loadOfficerDetail(o.id)
        } else this.notify(r.msg || '洗点失败')
      }))
    },
    doGrant () {
      const o = this.officerDetail.officer
      if (!o) return
      const id = o.id
      api.post('/games/ezfy/officers/' + id + '/grant', {}).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '赏赐失败'); return }
        this.notify('赏赐成功, 忠诚+10')
        // ★ 2026-10-05 局部刷新（，别全页面刷新）：就地 忠诚+10、黄金-1万，零接口重拉
        // ★★ 2026-10-19 与「赏赐宝物」同口径：Number 转换 + $set 写入 + 服务端校准
        //   （直接赋值在字段缺失/类型为字符串时不会触发视图更新，玩家看到「赏赐了没反应」）
        this.$set(o, 'loyalty', Math.min(100, (Number(o.loyalty) || 0) + 10))
        if (this.officerDetail && this.officerDetail.gold !== undefined) {
          this.officerDetail.gold = Math.max(0, this.officerDetail.gold - 10000)
        }
        const lst = (this.officerData && this.officerData.officers) || []
        for (const m of lst) { if (m.id === id) { m.loyalty = o.loyalty; break } }
        this.loadOfficerDetail(id)
      })
    },
    // ★ 2026-09-28 宝物可赏赐给军官加忠诚，品质不同加的不同（最高 +50）。
    //   点 [赏赐宝物] 展开可选列表（equipData.bag 里未穿戴的）；点某件 [赏赐] 确认后消耗该件并加忠诚。
    treasureLoyaltyGain (tier) {
      // 与后端 ezfyTreasureLoyalty 对齐：tier 1~4（初级/中级/高级/特殊）→ +10/+20/+35/+50
      return [0, 10, 20, 35, 50][tier] || 0
    },
    async doTreasureGrant (e) {
      const o = this.officerDetail.officer
      if (!o) return
      if (o.loyalty >= 100) { this.notify('忠诚已满, 无需赏赐'); return }
      // ★ 2026-09-28 修 bug：模板原来写 @click="doTreasureGrant"（不带括号），
      //   Vue 会把**原生 MouseEvent** 当宝物对象传进来 —— 于是弹出
      //   「赏赐「undefined」(普通) …」，点确认还会用 undefined 当 equip_id 提交。
      //   现在按「有没有装备 id」判断：事件对象/空值一律走展开分支，彻底免疫。
      if (!e || !e.id) {
        // 点「赏赐宝物」按钮：按需拉背包装备（改懒加载后不再整包重拉详情）再展开选择列表
        this.ensureOfficerBag()
        this.officerTreasureOpen = !this.officerTreasureOpen
        return
      }
      const gain = this.treasureLoyaltyGain(e.tier) || 10
      if (!await this.ask('赏赐「' + e.name + '」(' + (e.tier_name || '普通') + ') 给 ' + o.name + ' 吗？\n消耗这件宝物, 忠诚 +' + gain)) return
      api.post('/games/ezfy/officers/' + o.id + '/treasure-grant', { equip_id: e.id }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '赏赐失败'); return }
        this.notify(r.data && r.data.msg ? r.data.msg : ('赏赐成功, 忠诚 +' + gain))
        // ★ 2026-10-24 不再收列表：保持展开，赏赐后数量就地 -1、忠诚就地 +N，玩家看得见局部刷新
        //   （原来成功分支在服务端返回 code=400 时根本不执行；列表也总是被关掉，
        //     数量变化被遮住，用户以为「没刷新」）
        // ★ 2026-10-05 局部刷新（，别全页面刷新）：就地更新、零接口重拉——
        //   · 忠诚 +gain（封顶 100）
        //   · 从详情背包里移除已赏赐的那件宝物（数量 -1，computed officerTreasures 自动刷新，
        //     已赏赐的宝物不会再出现在列表里可点）
        //   · 军官列表里同步该军官忠诚
        // ★★ 2026-10-19 修「赏赐后忠诚 / 宝物数量不刷新」：
        //   ① 忠诚先转 Number 再算（后端 JSON 若给字符串，"50"+10 会拼成 "5010"）；
        //   ② 用 $set 写入，避免字段原本不存在时 Vue 2 侦测不到（直接赋值不触发视图更新）；
        //   ③ 本地更新只是**乐观反馈**，随后用服务端数据校准（详情 = 忠诚/三维，背包 = 剩余数量）。
        //      原来只靠本地推算 + splice，一旦 id 对不上或响应式没触发，界面就停在旧值。
        this.$set(o, 'loyalty', Math.min(100, (Number(o.loyalty) || 0) + gain))
        const bag = this.officerBag || []
        const idx = bag.findIndex(x => x.id === e.id)
        if (idx >= 0) bag.splice(idx, 1)
        const lst = (this.officerData && this.officerData.officers) || []
        for (const m of lst) { if (m.id === o.id) { m.loyalty = o.loyalty; break } }
        // 向服务端校准（静默，不闪屏）：详情整体替换 → 忠诚/属性一定是最新；
        // 背包走 reload 而不是 refresh（后者会先清空，列表会闪一下空）
        this.loadOfficerDetail(o.id)
        this.reloadOfficerBag()
      })
    },
    // ★ 属性加点（每升 1 级得 1 点，只影响自己的军官）
    doAddAttr (attr, count) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/attr', { attr, count }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '加点失败'); return }
        this.notify(r.msg || '加点成功')
        this.loadOfficerDetail(id)
      })
    },
    async doAddAttrAll (attr) {
      const free = this.officerDetail.officer.free_points
      const name = { military: '军事', logistics: '后勤', learning: '学识' }[attr] || attr
      if (!await this.ask('把剩余 ' + free + ' 点全部加到「' + name + '」吗？')) return
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/attr/all', { attr }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '加点失败'); return }
        this.notify(r.msg || '加点成功')
        this.loadOfficerDetail(id)
      })
    },
    // ★ 升星（消耗「星级徽章」；成功率/每星加点/上限均可后台配置）
    // ★ 2026-10-25 用户要求：**不给玩家展示成功率和每星加成**（那是运维参数），
    //   确认框只问「要不要升星」，失败提示也只有「升星失败」。
    async doStarUp () {
      const o = this.officerDetail.officer
      if (o.star >= o.star_max) { this.notify('星级已达上限'); return }
      if (o.star_card <= 0) { this.notify('没有「星级徽章」，可在商城购买或开宝箱获得'); return }
      if (!await this.ask('使用 1 枚「星级徽章」给 ' + o.name + ' 升星吗？（星级 ' + o.star + '→' + (o.star + 1) + '）')) return
      const id = o.id
      api.post('/games/ezfy/officers/' + id + '/starup', {}).then(r => {
        this.notify(r.msg || (r.code === 0 ? '升星成功' : '升星失败'))
        this.loadOfficerDetail(id)
        this.loadBag()
        this.loadAcade()
      })
    },
    // ★ 军官改名（消耗「军官改名卡」，只改玩家自己的军官）
    //   ⚠️ 方法名不能叫 doRename：城市改名已用该名，对象字面量后定义会覆盖先定义，
    //   导致城市改名的 [确定] 跑到军官改名逻辑（读 officerDetail 报错）。
    doOfficerRename () {
      const o = this.officerDetail.officer
      if (o.is_captive === 1) { this.notify('俘虏不能改名, 请先在军校收编'); return }
      // ★ 2026-10-23 在外采集/出征中禁止改名（订单按军官名关联，改名会导致归队/重复出征异常）
      if (o.status === 1) { this.notify('军官在外采集/出征中, 归队后才能改名'); return }
      if (o.rename_card <= 0) { this.notify('没有「军官改名卡」，可在商城购买'); return }
      const name = (this.officerRenameInput || '').trim()
      if (!name) { this.notify('请输入新名字'); return }
      if (name.length < 2 || name.length > 12) { this.notify('名字需 2~12 个字符'); return }
      const id = o.id
      api.post('/games/ezfy/officers/' + id + '/rename', { name }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '改名失败')
        this.loadOfficerDetail(id)
        this.loadBag()
      })
    },
    // ★ 2026-10-25 学技能改为「随机学一个」：不再传 skill_id（后端从技能池随机挑未学的），
    //   成功后用后端返回的文案告诉玩家随机到了哪个技能（如「学会技能【尖兵突击】Lv.6」）。
    doLearnRandom () {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'learn' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '学习失败')
        else this.notify(r.msg || '学习成功')
        this.loadOfficerDetail(id)
      })
    },
    doForget (name) {
      const id = this.officerDetail.officer.id
      const sid = this.skillIdByName(name)
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'forget', skill_id: sid }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '遗忘失败')
        this.loadOfficerDetail(id)
      })
    },
    skillIdByName (name) {
      for (const s of this.officerDetail.all_skills) {
        if (s.name === name) return s.id
      }
      return 0
    },
    doEquip (e) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: e.id, op: 'on' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '穿戴失败')
        this.loadOfficerDetail(id)
        this.refreshOfficerBag()
      })
    },
    // ★ 2026-09-29 装备背包叠加行 [穿戴]：穿组内第一件未穿戴的
    //   （同一 cfg 属性一致，穿哪件都一样；穿完该部位即占满，剩余同款不能再穿）
    doEquipGroup (g) {
      if (!g || !g.items || !g.items.length) return
      this.doEquip(g.first)
    },
    doUnequip (equipId) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: equipId, op: 'off' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '卸下失败')
        this.loadOfficerDetail(id)
        this.refreshOfficerBag()
      })
    },
    async doUnequipAll () {
      const id = this.officerDetail.officer.id
      if (!await this.ask('确定卸下该军官身上全部 ' + this.officerDetail.equipped.length + ' 件装备吗?')) return
      api.post('/games/ezfy/officers/' + id + '/unequip-all', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '卸下失败')
        else this.notify(r.msg || '已卸下全部装备')
        this.loadOfficerDetail(id)
        this.refreshOfficerBag()
      })
    },
    async doEquipSet (s) {
      const id = this.officerDetail.officer.id
      if (!await this.ask('确定一键穿戴「' + s.name + '」整套吗?（同部位已穿戴的会自动卸下让位）')) return
      api.post('/games/ezfy/officers/' + id + '/equip-set', { set_id: s.set_id }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '穿戴失败')
        else this.notify(r.msg || '穿戴成功')
        this.loadOfficerDetail(id)
        this.refreshOfficerBag()
      })
    },
    doPosition (o, pos) {
      api.post('/games/ezfy/officers/' + o.id + '/position', { position: pos }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '任命失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
        // ★ 任命/卸任后同步刷新出征页的带队军官列表：若玩家随后立即出征，
        //   下拉里必须是最新的可选军官（避免「刚任命的市长/城守不出现在下拉」的观感）
        // ★ 2026-10-25 删掉 `this.loadCityOfficers()`：**这个方法全项目不存在**（旧名残留），
        //   调用只会抛 TypeError 并被 Promise 吞掉，什么都不做。
        this.loadOnDutyOfficers()
      })
    },
    // ★ 2026-10-24 怀旧军官详情：任命入口（军校页任命逻辑的详情页版本）
    // ★ 2026-10-25 支持三个动作：1 市长 / 2 城守 / 0 卸任（用户要求属性页也能任命城守）
    doAppoint (pos) {
      const o = this.officerDetail && this.officerDetail.officer
      if (!o) return
      if (o.status === 1) { this.notify('军官在外采集/出征中, 归来后才能任命'); return }
      if (o.is_captive === 1) { this.notify('俘虏不能任命'); return }
      const tips = { 0: '已卸任', 1: '已任命为市长', 2: '已任命为城守' }
      api.post('/games/ezfy/officers/' + o.id + '/position', { position: pos }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '任命失败'); return }
        this.notify(tips[pos] || '操作成功')
        // 任命后立刻重拉详情：状态行会显示「（市长）/（城守）」，玩家能直接看到生效
        this.loadOfficerDetail(o.id)
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
        // ★ 2026-10-25 同上：`loadCityOfficers` 不存在（旧名残留），删掉
        this.loadOnDutyOfficers()
      })
    },
    async doCaptive (o, op) {
      if (op === 'free' && !await this.ask('确定释放俘虏 ' + o.name + ' 吗?')) return
      api.post('/games/ezfy/officers/' + o.id + '/captive', { op: op }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '操作失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
      })
    },
    // ★ 2026-10-11 战俘营 [一键释放]：释放**当前城**全部俘虏（后端物理删除，只在日志文件留痕）。
    //   与单个释放同口径：出征中的俘虏后端会跳过。
    async releaseAllCaptives () {
      const n = this.captiveOfficers.length
      if (!n) return
      if (!await this.ask('确定一键释放本城全部 ' + n + ' 名俘虏吗?（释放后不可恢复）')) return
      // 连点防抖：请求未返回前忽略后续点击（与 doCaptive / 商城购买同款）
      this.once('releaseCaptives', () => api.post('/games/ezfy/officers/captive/release-all', {}).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '操作失败'); return }
        this.notify((r.data && r.data.msg) || '已一键释放')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
      }))
    },
    async doExile () {
      if (!await this.ask('确定流放该武将吗? 流放后无法找回!')) return
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/exile', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '流放失败')
        this.go('acade')
      })
    }
  }
}
</script>

<style>
/* ===== 二战风云 怀旧文字WAP主题(复刻 stzb-fk city-theme.css), 与家园页面一致靠左排布 ===== */

/* 沉浸式: 原版 style.css 给 body 留了 5px 外边距, 会让深色标题条四周露白边.
   进入本页时由 Ezfy.vue 的 mounted 挂上这个 class, 离开时移除 */
body.ezfy-immersive { margin: 0; }

.ezfy-page {
  background: #fff;
  min-height: 100%;
  color: #333;
  /* ★ 字体族: 2026-09-24 用户反馈「微软雅黑不好看」→ 改复古宋体风
     （3GQQ 时代 WAP 文字游戏的主流样式, 标题/正文统一宋体更有怀旧味）。
     回退链: Windows→宋体/SimSun, macOS→宋体-简(Songti SC), 其余→serif。
     ★ 2026-09-25：**只保留在桌面**。手机上换成系统黑体，见下面的 @supports 段。 */
  font-family: '宋体', 'SimSun', 'Songti SC', 'NSimSun', '新宋体', serif;
  /* ★ 2026-09-25 「页面文字大小除了地图，全部改成和首页导航(聊天/邮箱/军情/任务/好友/首页)一样大」：
     首页导航字号 17px 定为**全站唯一基准** —— 除地图格(.ezfy-map-table / .ezfy-cell)外，
     页面所有文字都取这个变量。以后要整体调大小，只改这一行。
     例外（都是「非正文」或布局硬约束，已在各自规则里注明）：
       ① .title-bar 标题栏 18px —— 用户此前明确点名「勿动」；
       ② .panel-title 小标题 18px —— 标题与正文拉开一档；
       ③ 地图格 + 战场指挥室表格 —— 手机上列数太多，必须缩（见各自媒体查询）。 */
  --fs: 17px;
  font-size: var(--fs);
  /* ★ 2026-09-25 iPhone 修复①：iOS Safari 会按视口宽度**自动放大/缩小正文**，
     不锁死的话同一段文字在 iPhone 和桌面看到的字号不一致（用户反馈「手机上字体不好看」）。
     100% = 完全按 CSS 里写的字号来，不做任何自动缩放。 */
  -webkit-text-size-adjust: 100%;
  text-size-adjust: 100%;
  line-height: 1.5;
  /* 根容器左右不再用负 margin: 会溢出 #app 产生横向滚动条.
     铺满由内部 .title-bar 的 margin:0 -8px 抵消 padding 实现 */
  margin: 0;
  padding: 0 8px 20px;
}
/* ★ 2026-09-25 iPhone 修复②（关键的一条）：**iOS 上没有「宋体 / SimSun」这两个字体**，
   于是回退到系统自带的 **Songti SC（宋体-简）** —— 它是衬线体、笔画极细，
   小字号在 Retina 屏上又细又灰、边缘发虚，观感比 Windows 的 SimSun 差一大截，
   这就是用户说的「iPhone 上字体不好看」。
   处理（★ 「iPhone 单独用 iPhone 自己的字体，其他设备不变」）：
     · **桌面（Windows/macOS）保留复古宋体**，一行不动；
     · **只有 iPhone / iPad 换成苹果自带的系统字体** —— 下面这份就是 iPhone 上的常驻字体：
         `-apple-system` / `system-ui` → 苹果系统字体（西文 San Francisco + 中文 **苹方**）
         `'PingFang SC'` 苹方（iOS 9+ 中文默认）｜ `'Heiti SC'` 黑体-简（更老的 iOS）
         `'Hiragino Sans GB'` 冬青黑体简体中文（兜底）
       与《镜花缘》(Jingwt.vue)、《西游记》(Xiyou.vue) 的字体族方向一致。
   怎么只圈到苹果设备：JS 判 iOS（mounted 里给 body 挂 .ezfy-ios），Apple 专属、绝不漏判。
   ⚠️ 不要用 `@media (hover:none) and (pointer:coarse)` —— 那会把安卓也一起改了,
     用户明确要求「其他设备不变」。
   ★ 2026-09-27：旧实现用 `@supports (-webkit-touch-callout: none)`，但 iOS Safari 的
     @supports 解析器不认 -webkit-touch-callout → 条件永不成立 → iPhone 一直回落宋体-简
     (Songti SC) 细灰发虚。故改用 JS 判定（见 mounted），比 @supports 可靠。 */
body.ezfy-ios .ezfy-page,
body.ezfy-ios .ezfy-page button,
body.ezfy-ios .ezfy-page input,
body.ezfy-ios .ezfy-page select,
body.ezfy-ios .ezfy-page textarea {
  font-family: -apple-system, system-ui, 'PingFang SC', 'Heiti SC',
               'Hiragino Sans GB', 'Helvetica Neue', sans-serif;
}
/* 表单控件/按钮默认不继承字体族, 显式补上(原版也是 body,button,input,select,textarea 一起设) */
.ezfy-page button,
.ezfy-page input,
.ezfy-page select,
.ezfy-page textarea {
  font-family: inherit;
}
.ezfy-page .home-wrap {
  font-size: var(--fs);
}
.ezfy-page a {
  text-decoration: none;
  margin: 0 1px;
  color: #0645ad;
}
.ezfy-page a:hover { text-decoration: underline; }
.ezfy-page .title-bar {
  background: #050709;
  color: #fff;
  /* ★ 2026-09-25：用户此前明确点名「二战征途-【1区】红色警戒」这一行勿动 → 保留 18px，
     没有跟着全站统一成 var(--fs)。要一起拉平说一声。 */
  font-size: 18px;
  font-weight: bold;
  text-align: left;
  padding: 6px 10px;
  letter-spacing: 1px;
  margin: 0 -8px;
  box-sizing: border-box;
}
.ezfy-page .top-nav {
  text-align: left;
  /* ★ 底部不留 padding: 导航链接自身已有 2px 上下 padding, 再加 4px 会让「紧跟它后面
     的第一个区块」上方凭空多出 4px 留白(导航字→下一行的距离比行与行之间大一截)。
     顶部那 4px 要留 —— 上面是深色标题条, 需要这段间隔。 */
  padding: 4px 0 0;
}
.ezfy-page .top-nav a {
  display: inline-block;
  /* ★ 用户「聊天/邮箱/军情/任务/好友/首页 间隔稍微大一点」→ 0 改 4px；
     随后「又大了 稍微小一点」→ 收到 3px（相邻两词间距 3+3 = 6px）。
     ★ 必须与下面 .ezfy-subnav a 保持同一个值，否则「点进去二级菜单变宽/变窄」。 */
  padding: 2px 3px;
  line-height: 1.35;
  /* ★★ 这一行就是全站字号基准（--fs）的来源：用户 2026-09-25 要求
     「页面文字除了地图，全部和首页导航一样大」→ 其它地方一律写 var(--fs)。 */
  font-size: var(--fs);
}
/* ★ 用户反馈「导航整体有点靠右」：每个链接自带 padding + 1px margin，
   于是第一个链接「聊天」的文字比下面正文行(公告/新城市…)右移。
   去掉首链接的左侧留白 → 整条导航左移，与正文左对齐。 */
/* ★ 2026-10-23 顶部导航里新增了 0×0 的 SVG（共用感叹号渐变）→ 首链接不再一定是
   :first-child，改用 :first-of-type 才能继续选中「聊天」，保持左边对齐。 */
.ezfy-page .top-nav a:first-of-type { margin-left: 0; padding-left: 0; }
/* ★★ 2026-10-13 顶部导航「军情」的未读角标：有未看的 军情警讯(预警)/战斗报告 时，
   在「军情」右边显示一个感叹号（提示方式类似首页公告行前的小喇叭）。
   ★ 造型：**裸感叹号**（不加圆圈底，用户要求「别用圆圈包着」），用内联 SVG 画；
     配色取公告小喇叭 notice.gif 的**主体色**（亮黄 #FCE224 → 黄 #FED903 → 橙 #FDA504），
     压一道砖红 #8E2525 细描边（小喇叭里的那个红）—— 纯黄在浅底上会看不清。
   ⚠️ 尺寸 7×12px（用户「再扁一点点」→ 高 14 收到 12、viewBox 同步压到 8×14）。
   ★ 2026-10-13 用户「再窄点」→ 宽 7px 收到 5px，并给 svg 加 `preserveAspectRatio="none"`：
     只压**横向**（高度不变）→ 感叹号变瘦长；若不加这句，SVG 会等比缩放，改 width 只会整体变小、
     宽度比例纹丝不动（这也是上次「改窄了但看着没变」的原因）。
   靠 vertical-align 微调，不能把 .top-nav 那一行撑高（导航行高 = 全站字号基准）。 */
.ezfy-page .top-nav .ezfy-nav-warn {
  display: inline-block;
  width: 5px; height: 12px;
  margin-left: 3px;
  vertical-align: -1px;
  /* 极淡的深红描边光晕，浅色导航底上边缘更实（只有一个元素，开销可忽略） */
  filter: drop-shadow(0 0 .5px rgba(142, 37, 37, .55));
  animation: ezfyNavWarnPulse 1.1s ease-in-out infinite;
}
@keyframes ezfyNavWarnPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: .45; }
}
/* 二级导航(资源/军官/军队/科技/城防/统帅) —— 复刻原版军队/城防/兵种页里的那行 */
/* ★ 配色按默认 #004299，当前选中 #c0392b 加粗
   ★ 用户反馈「点进去后间隔变大，首页里的这个导航就对」：
     根因是这里用 inline-block(会把换行空白算成一个空格宽)，
     而首页导航用的是 inline。改成 inline 并收窄 padding，
     与首页视觉完全一致。
   ★★ 2026-10-09 「点了的导航加个颜色区分」→ 当前项由黑色改为全站统一的 #c0392b。 */
.ezfy-page .ezfy-subnav a {
  display: inline;
  padding: 0 1px;
  /* ★ 2026-09-29 「资源.军官.军队.科技.城防.统帅 间隔小一点点」：margin 1px → 0 */
  margin: 0;
  /* ★ 字号与 .top-nav a 统一（同一个变量，改一处两处一起变） */
  font-size: var(--fs);
  color: #004299;
}
.ezfy-page .ezfy-subnav a.on { color: #c0392b; font-weight: bold; }
/* ★ 用户「点进去 资源/军官/军队/科技/城防/统帅 左边 和 聊天 的『聊』字对齐」：
   首链接左侧 padding/margin 归零 —— 与 .top-nav a:first-child 同源。
   两者父容器(.old-line / .top-nav)左右 padding 都是 0，归零后文字左边缘必定对齐。 */
.ezfy-page .ezfy-subnav a:first-child { margin-left: 0; padding-left: 0; }
/* ★ 2026-09-29 首页那行「军事.资源.军官.军队.科技.城防.统帅」跨平台间距不一致：
   win/mac/手机 渲染同一段 inline 换行空白，空格宽度因字体各不相同。
   改成 flex（gap 固定 3px）+ 每个链接稳距，彻底消除换行空白导致的参差。 */
.ezfy-page .old-line.home-nav2 {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 3px;
}
.ezfy-page .old-line.home-nav2 a {
  color: #004299;
  margin: 0;
  padding: 0 1px;
}
.ezfy-page .old-line.home-nav2 a.on { color: #000; font-weight: bold; }
/* ★ 2026-10-07 首页曾经给关键区块加过 .home-block 的 10px 分组留白，
   「间隔去掉」→ 已全部移除。
   现在首页是**完全连续的一列**，不再有任何人为分组间距。 */
/* 首页内容容器：行距比原版 1.5 略紧一点（用户：「比之前紧凑了是吧，上下间隔大一点点」→ 取 1.45）。
   只作用于首页，不影响其它页面。 */
.ezfy-page .home-body { line-height: 1.45; }

/* ====================== 首页-新布局（战争简约版，2026-10-10 白底优化） ====================== */
/* 白底 + 浅灰卡片 + 深红点缀：保留战争风格（城市/按钮红黑强调、浅灰"军牌"卡片），
   不破坏整体浅色氛围；一屏放全功能、去重后的入口按钮。只作用于首页新布局。 */
.ezfy-page .war-home {
  margin-top: 6px;
  padding: 6px 8px 8px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  color: #333;
  line-height: 1.35;
  /* ★ 2026-10-10 用户：「别铺满整屏幕 铺一半」→ 桌面限到约半屏宽（约 720px），手机上仍全宽 */
  max-width: 720px;
}
.ezfy-page .war-home a { margin: 0 2px; }
.ezfy-page .war-home a:hover { text-decoration: none; }
.ezfy-page .war-home .ezfy-ico { vertical-align: -5px; }
/* 头部：城市 + 军衔/声望/军团/签到 */
.ezfy-page .war-head {
  border-bottom: 1px dashed #d5d9e0;
  padding-bottom: 4px;
  margin-bottom: 5px;
}
.ezfy-page .war-head .city-name {
  color: #8b2f2f; /* 战争深红，突出主城 */
  font-weight: bold;
  font-size: calc(var(--fs) + 1px);
}
.ezfy-page .war-stats {
  margin-top: 2px;
  display: flex;
  flex-wrap: wrap;
  gap: 0 10px;
}
.ezfy-page .war-stat { color: #555; font-size: var(--fs); }
/* 资源 5 列网格（图标 + 现有/每小时产量） */
.ezfy-page .war-res {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 4px;
}
.ezfy-page .war-res-cell {
  background: #f7f8fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 3px 2px 4px;
  text-align: center;
  min-width: 0;
}
.ezfy-page .war-res-cell .ezfy-ico { width: 24px; height: 24px; }
.ezfy-page .war-res-name { display: block; font-size: var(--fs); line-height: 1.2; }
.ezfy-page .war-res-val {
  margin-top: 1px;
  font-size: 13px;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
}
.ezfy-page .war-res-val i { font-style: normal; color: #999; margin: 0 1px; }
.ezfy-page .war-res-ops {
  grid-column: 1 / -1;
  text-align: left;
  font-size: var(--fs);
  padding-top: 2px;
}
/* 人口/民心/税率 一行 */
.ezfy-page .war-row2 {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin: 5px 0;
}
.ezfy-page .war-cell2 {
  flex: 1 1 auto;
  background: #f7f8fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 3px 6px;
  font-size: var(--fs);
  color: #333;
  white-space: nowrap;
}
/* 功能入口按钮网格（4 列，去重后全部功能） */
.ezfy-page .war-btns {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 4px;
}
.ezfy-page .war-btn {
  display: block;
  text-align: center;
  padding: 5px 2px;
  background: #f0f2f5;
  border: 1px solid #d5d9e0;
  border-radius: 4px;
  color: #333 !important;
  font-size: var(--fs);
  margin: 0;
}
.ezfy-page .war-btn:hover {
  color: #c0392b !important; /* 战争红 */
  border-color: #c0392b;
  background: #fdf3f2;
  text-decoration: none;
}
/* 世界聊天 */
.ezfy-page .war-chat {
  margin-top: 5px;
  border-top: 1px dashed #d5d9e0;
  padding-top: 4px;
  color: #333;
}
@media (max-width: 480px) {
  .ezfy-page .war-home { max-width: 100%; }
  .ezfy-page .war-res { grid-template-columns: repeat(3, 1fr); }
  .ezfy-page .war-btns { grid-template-columns: repeat(3, 1fr); }
}
/* 军衔/排行页所有表格：数据水平 + 垂直居中（）*/
/* ★ 排行页四个表格统一宽度（「表格有的大有的小，统一整齐」→ 又要求「太长占页面，改50%」）：
   width:50% 占 panel 一半宽度，table-layout:fixed 配合各表 colgroup 比例分列，长文本自动折行 */
.ezfy-page .ezfy-rank-table {
  width: 50%;
  min-width: 360px;
  table-layout: fixed;
  border-collapse: collapse;
}
.ezfy-page .ezfy-rank-table th,
.ezfy-page .ezfy-rank-table td {
  text-align: center;
  vertical-align: middle;
  overflow-wrap: break-word;
}
/* ★ 排行榜优化（「榜单太单调、没有追榜动力」）：
   名次做成奖牌徽章，前三名金/银/铜；冠亚季军整行按金/银/铜着色 + 冠军皇冠；
   普通行斑马纹 + 悬停。军衔晋升表是静态参照表（无奖牌徽章、无 rank 类），
   不受这些榜单高亮影响。 */
.ezfy-page .rank-medal {
  display: inline-block;
  min-width: 24px;
  height: 24px;
  line-height: 22px;
  padding: 0 8px;
  border-radius: 12px;
  background: #dedede;
  color: #666;
  font-weight: bold;
  /* ★ 2026-09-25 随全站统一：13 → var(--fs)（奖牌里的名次数字也是页面文字） */
  font-size: var(--fs);
  text-align: center;
  border: 1px solid transparent;
  box-sizing: border-box;
}
.ezfy-page .rank-medal.m1 {                          /* 金 */
  background: linear-gradient(180deg, #ffe08a, #f2c94c);
  color: #7a4b00;
  border-color: #e0b93c;
  box-shadow: 0 0 6px rgba(242, 201, 76, .8);
  /* 冠军奖牌大一号，随基准同步 +2 */
  font-size: calc(var(--fs) + 2px);
  min-width: 26px; height: 26px; line-height: 24px; border-radius: 13px;
}
.ezfy-page .rank-medal.m2 {                          /* 银 */
  background: linear-gradient(180deg, #f2f5f7, #cbd4da);
  color: #455a64;
  border-color: #b9c4cb;
  font-size: var(--fs);
}
.ezfy-page .rank-medal.m3 {                          /* 铜 */
  background: linear-gradient(180deg, #f2cdab, #e0a370);
  color: #5d3a1a;
  border-color: #c88a58;
  font-size: var(--fs);
}
/* 冠军皇冠 */
.ezfy-page .rank-crown {
  display: inline-block;
  margin-right: 4px;
  color: #e6a817;
  font-size: 18px;
  vertical-align: middle;
  text-shadow: 0 1px 2px rgba(122, 75, 0, .35);
}
.ezfy-page .ezfy-rank-table tr:nth-child(even) td { background: #faf8f2; }
.ezfy-page .ezfy-rank-table tr:hover td { background: #f0ecdf; }
/* 军衔晋升表「宝物」点击展开行（★ 2026-09-28 宝物列改为点击展示，展开行做浅色底色区分） */
.ezfy-page .ezfy-rank-table tr.rank-treasure-row td { background: #fffbe8; font-weight: normal; }
.ezfy-page .ezfy-rank-table tr.rank-treasure-row:hover td { background: #fffbe8; }
/* 冠/亚/季军整行着色（置于悬停/斑马纹之后，确保三者之上仍保持奖牌底色） */
.ezfy-page .ezfy-rank-table tr.rank-1 td { background: #fdeebb; }
.ezfy-page .ezfy-rank-table tr.rank-2 td { background: #eef2f5; }
.ezfy-page .ezfy-rank-table tr.rank-3 td { background: #f6e3d0; }
.ezfy-page .ezfy-rank-table tr.rank-1 td:first-child,
.ezfy-page .ezfy-rank-table tr.rank-2 td:first-child,
.ezfy-page .ezfy-rank-table tr.rank-3 td:first-child { font-weight: bold; }
/* 学院(acade)页所有表格：数据水平 + 垂直居中（）*/
.ezfy-page .ezfy-plain-table th,
.ezfy-page .ezfy-plain-table td {
  text-align: center;
  vertical-align: middle;
}
/* ★ 2026-10-25 商城「全部」分类分组标题（tr.mall-cat-row）已按用户要求去掉，样式一并删除 */
/* ★ 战场指挥室：本回合倒计时条（指令期绿色、锁定后红色） */
.ezfy-page .ezfy-battle-bar {
  width: 100%;
  max-width: 420px;
  height: 8px;
  margin: 3px 0;
  background: #e6e6e6;
  border: 1px solid #c8c8c8;
  overflow: hidden;
}
.ezfy-page .ezfy-battle-bar > i {
  display: block;
  height: 100%;
  transition: width 0.9s linear;
}
.ezfy-page .ezfy-battle-bar > i.on { background: #27763c; }
.ezfy-page .ezfy-battle-bar > i.lock { background: #c0392b; }
/* ★ 2026-10-10 战场位置条：每行一对「我方-敌方」兵种（我方左、敌方右、同一行面对面），
   行内图标按统一 pos 轴定位，轮询回新 pos 时平滑滑动。
   ★ 2026-10-20 守方视角由 JS 镜像坐标；class 仍用 .atk=我方绿 / .def=敌方红。 */
.ezfy-page .ezfy-battle-posbars { margin: 4px 0 3px; }
.ezfy-page .ezfy-battle-pair {
  display: flex;
  align-items: flex-start;
  gap: 4px;
}
.ezfy-page .ezfy-battle-pair + .ezfy-battle-pair { margin-top: 2px; }
.ezfy-page .ezfy-pair-side {
  flex: 0 0 auto;
  width: 12px;
  font-size: 11px;
  font-weight: bold;
  line-height: 1;
  padding-top: 8px;
}
.ezfy-page .ezfy-pair-side.atk { color: #27763c; }
.ezfy-page .ezfy-pair-side.def { color: #c0392b; }
.ezfy-page .ezfy-battle-poslane {
  position: relative;
  flex: 1 1 auto;
  height: 40px;
  border: 1px solid #d4d4d4;
  border-radius: 3px;
  background: linear-gradient(90deg, #eefaf0, #fff 45%, #fff 55%, #fdeeec);
  overflow: hidden;
}
/* ★ 2026-10-10 内缩 14px 的跑道：图标(26px 宽、居中)贴到两端时也不会越出格子边界。 */
.ezfy-page .ezfy-battle-postrack {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 14px;
  right: 14px;
}
.ezfy-page .ezfy-battle-unit {
  position: absolute;
  top: 1px;
  width: 26px;
  margin-left: -13px;
  text-align: center;
  /* pos 每次轮询更新 → left 变化 → 平滑滑动 */
  transition: left 0.6s linear;
}
.ezfy-page .ezfy-battle-unit .ezfy-unit-ico {
  display: block;
  box-sizing: border-box;
  width: 26px;
  height: 26px;
  margin: 0 auto;
  border: 2px solid #b8b8b8;
  border-radius: 4px;
  background: #f4f4f4;
  overflow: hidden;
  line-height: 0;
}
/* ★ 2026-10-10 图标按攻守上色区分（用户反馈「图标区分不出攻守」）：
   攻=绿框、守=红框，与行首「攻/守」标签同色。 */
.ezfy-page .ezfy-battle-unit.atk .ezfy-unit-ico { border-color: #27763c; background: #eefaf0; }
.ezfy-page .ezfy-battle-unit.def .ezfy-unit-ico { border-color: #c0392b; background: #fdeeec; }
.ezfy-page .ezfy-battle-unit .ezfy-unit-ico svg {
  display: block;
  width: 100%;
  height: 100%;
}
.ezfy-page .ezfy-battle-unit .ezfy-unit-name {
  display: block;
  max-width: 48px;
  margin: 0 auto;
  font-size: 10px;
  line-height: 1.1;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ezfy-page .ezfy-battle-unit.atk .ezfy-unit-name { color: #27763c; }
.ezfy-page .ezfy-battle-unit.def .ezfy-unit-name { color: #c0392b; }
.ezfy-page .ezfy-battle-unit.dead { opacity: .4; }
/* ★ 战场指挥室：双方兵力表按攻守着色 —— 我方整行浅绿(enemy 浅红)、方标签加粗 */
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-self td { background: #eef7f0 !important; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-enemy td { background: #fdeeec !important; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-self td.ezfy-side-lbl { font-weight: bold; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-enemy td.ezfy-side-lbl { font-weight: bold; }
/* ★ 战场指挥室：行动日志 —— 最新回合排最上，回合标题加粗，回合块间留白 */
.ezfy-page .ezfy-battle-legend { font-size: var(--fs); }
.ezfy-page .ezfy-round-block {
  margin: 4px 0 6px;
  padding-left: 6px;
  border-left: 2px solid #ddd;
}
.ezfy-page .ezfy-round-title {
  font-weight: bold;
  color: #333;
  margin-bottom: 2px;
}
.ezfy-page .ezfy-round-line {
  line-height: 1.6;
  font-size: var(--fs);
}
.ezfy-page .ezfy-round-line.green { color: #1d5c2e; }
.ezfy-page .ezfy-round-line.red { color: #a02a1e; }
.ezfy-page .ezfy-round-line.gray { color: #777; }
/* ★ 战报详情/逐回合详情按行上色：始终「自己绿、敌军红」—— rpt-atk=绿(自己/己方)、rpt-def=红(敌军)，
   攻/守分配给哪种色随战报视角(被掠夺/被征服=守方)翻转，见 reportLineClass。 */
.ezfy-page .rpt-ln { line-height: 1.6; }
.ezfy-page .rpt-ln > span { display: inline; white-space: pre-wrap; }
.ezfy-page .rpt-atk { color: #1d5c2e; }
.ezfy-page .rpt-def { color: #a02a1e; }
/* ★ 商城「装备 / 道具」的分类筛选：flex 自动换行。
   原 grid repeat(6, max-content) 固定 6 列，分类名较长时(手机端)整排溢出容器形成横向滑动；
   2026-09-24 改为流式排列，超过一行宽度就换行(背包分类共用此类)。 */
.ezfy-page .ezfy-slot-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  margin: 2px 0;
  /* ★ 2026-09-25 随全站统一：15 → var(--fs)（分类链接也是页面文字，不再单独小一号） */
  font-size: var(--fs);
}
.ezfy-page .ezfy-slot-grid > a { white-space: nowrap; }
/* ★ 2026-10-14 筛选网格的选中高亮（用户「切小类别切换没有高亮，玩家不知道在哪个小类别下」）：
   .ezfy-slot-grid 用于 商城道具分类 / 装备部位筛选 / 背包分类 三处，
   原来只绑了 .on class 没写样式 → 选中项和未选看起来一样。
   口径与 .acade-tab a.on 一致：红 + 加粗。 */
.ezfy-page .ezfy-slot-grid > a.on { color: #c0392b; font-weight: bold; }
/* ★ 2026-10-21 商城分类网格：参考装备部位筛选的样式（左对齐、按内容宽度、10px 间距），
   只是强制每行 4 个。★ 不能再用 repeat(4, 1fr)：桌面端会把 4 个分类摊到整行、
   把文字居中，和装备分类那种紧凑左对齐完全不像。
   ★ 2026-10-25 用户「装备/套装的分类也按一行四个」→ 装备/套装分栏的「部位筛选」也套这个类
     （两处共用：道具分类 mallCat / 装备部位 shopSlot）。 */
.ezfy-page .ezfy-slot-grid.mall-cat-grid {
  display: grid;
  grid-template-columns: repeat(4, max-content);
  justify-content: start;
  gap: 2px 10px;
}
/* ★ 2026-09-25 套装加成展示（用户反馈「套装的加成看不到，不知道买完套装给军官用哪个」）
   三处展示，全部是**纯文本竖排**，窄屏不会横向溢出：
     ① .set-card  —— 装备表里点套装名，在该行下面**跨整行**展开的加成卡
        （刻意不新增表格列：手机上加一列就挤爆，跨整行展开才能完整显示）
     ② .set-block —— 装备页「套装一览」里一套一块，方便上下对比
     ③ .set-mini  —— 商城把「所属套装」塞在名称下面一行（商城表没有套装列） */
.ezfy-page .set-card-row > td {
  /* 表格单元格默认是居中/右对齐，加成卡必须左对齐才好读 */
  text-align: left;
  padding: 3px 6px 6px 0;
}
.ezfy-page .set-card {
  background: #faf8f2;
  border: 1px solid #d8d5cc;
  border-left: 3px solid #2f6fb5;
  border-radius: 3px;
  padding: 4px 8px;
  line-height: 1.7;
}
.ezfy-page .set-card .sc-h { margin-bottom: 2px; }
.ezfy-page .set-card .sc-h .gray { margin-left: 4px; }
.ezfy-page .set-card .sc-b { word-break: break-word; }
.ezfy-page .set-mini { line-height: 1.6; color: #666; word-break: break-word; }
.ezfy-page .set-block {
  margin: 4px 0 6px;
  padding: 4px 8px;
  background: #faf8f2;
  border-left: 3px solid #2f6fb5;
  border-radius: 3px;
}
.ezfy-page .set-block .sb-h { line-height: 1.7; }
.ezfy-page .set-block .sb-h .gray { margin: 0 2px; }
.ezfy-page .set-block .sb-b { line-height: 1.7; word-break: break-word; }
/* 「只看我有的 / 全部套装·可对比」切换（复用页面上的方括号链接风格，选中变红加粗） */
.ezfy-page .set-tab a { margin-right: 8px; }
.ezfy-page .set-tab a.on { color: #c0392b; font-weight: bold; }
/* ★ 购买 / 开箱面板：卡片式，和上方表格拉开层次（原来只是行内一条左边框，挤成一坨） */
.ezfy-page .ezfy-buy-box {
  margin: 8px 0;
  padding: 8px 12px;
  max-width: 560px;
  background: #faf8f2;
  border: 1px solid #d8d5cc;
  border-radius: 3px;
}
.ezfy-page .ezfy-buy-box .bb-title {
  font-weight: bold;
  color: #2f4156;
  padding-bottom: 4px;
  margin-bottom: 6px;
  border-bottom: 1px solid #e8e4d8;
}
.ezfy-page .ezfy-buy-box .bb-row { line-height: 2.2; }
.ezfy-page .ezfy-buy-box .bb-row input[type="number"] { width: 70px; }
.ezfy-page .ezfy-buy-box .bb-total { color: #c0392b; }
/* ★ 商城购买确认内联行：直接展开在「购买」按钮所在行的正下方，淡色背景区分
     （「确认在购买按钮附近」，不再用表格底部那个独立面板） */
.ezfy-page .ezfy-buy-inline td {
  background: #faf8f2;
  text-align: left;
  line-height: 2.3;
}
.ezfy-page .ezfy-buy-inline > td > span { margin: 0 4px; }
.ezfy-page .ezfy-buy-inline select { margin-right: 12px; }
.ezfy-page .ezfy-buy-inline button { margin: 0 8px 0 12px; }
/* 自适应高度文本域（私聊等） */
.ezfy-page .ezfy-auto-textarea {
  width: 100%;
  box-sizing: border-box;
  min-height: 44px;
  max-height: 160px;
  padding: 4px 6px;
  /* ★ 2026-09-25 随全站统一：15 → var(--fs)（聊天框里的字和正文一样大） */
  font-size: var(--fs);
  line-height: 1.5;
  font-family: inherit;
  border: 1px solid #c8c8c8;
  border-radius: 3px;
  resize: vertical;
  overflow-y: auto;
}
/* 司令部·兵种战斗配置：一兵种一块，窄屏不遮盖 */
.ezfy-page .ezfy-tgt-block {
  padding: 4px 2px; margin: 4px 0; border-bottom: 1px dashed #e2e2e2;
}
.ezfy-page .ezfy-tgt-name { font-weight: bold; color: #2f4156; margin-bottom: 2px; }
.ezfy-page .ezfy-tgt-row { display: flex; align-items: center; flex-wrap: wrap; gap: 4px; margin: 2px 0; }
.ezfy-page .ezfy-tgt-lab { color: #666; min-width: 56px; display: inline-block; }
/* ★ 2026-10-28 司令部·「防守数量」输入框：手机手指能点中（高 30px），
   宽度固定不挤占后面的「城内 N」；桌面端同样观感一致。 */
.ezfy-page .ezfy-tgt-num {
  width: 100px; height: 30px; padding: 0 6px; box-sizing: border-box;
  border: 1px solid #c9d2dc; border-radius: 3px; background: #fff; color: #333;
  font-size: var(--fs); line-height: 28px;
}
.ezfy-page .ezfy-tgt-num:focus { border-color: #2f6fb3; outline: none; }
/* 页面内消息（替代 alert 弹窗）：现在由 msgStyle 固定定位在点击点附近 */
.ezfy-page .ezfy-msg {
  padding: 4px 8px; border-radius: 3px;
  /* ★ 2026-09-25 随全站统一：14 → var(--fs)（提示条也是页面文字） */
  font-size: var(--fs); line-height: 1.5; border-left: 3px solid #999; background: #f5f5f5;
}

.ezfy-page .ezfy-msg-ok { border-left-color: #27763c; background: #eef7f0; color: #1d5c2e; }
.ezfy-page .ezfy-msg-error { border-left-color: #c0392b; background: #fdeeec; color: #a02a1e; }
.ezfy-page .ezfy-msg-info { border-left-color: #2f6f9f; background: #eef4fa; color: #235b85; }
/* 页面内确认条（替代 confirm / prompt 弹窗）
   ★ 2026-09-25 用户反馈「军官详情的一键穿戴，确认跑到页面最上面，要滚上去才能点」：
     改成**浮层** —— 位置由 askStyle() 按「刚才点的那个控件」实时算（position:fixed 内联下发），
     所以这里 margin 归零、加阴影和更实的底色，让它看起来是「贴着按钮弹出来的」，
     而不是页面正文里的一段。z-index 比提示条(.ezfy-msg 9999)高，避免被提示条压住。 */
.ezfy-page .ezfy-ask {
  margin: 0;
  padding: 8px 10px;
  border: 1px solid #d8c890;
  background: #fffbe8;
  border-radius: 4px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, .22);
  box-sizing: border-box;
}
/* ★ 2026-09-25 随全站统一：14 → var(--fs) */
.ezfy-page .ezfy-ask-text { font-size: var(--fs); color: #7a5c10; margin-bottom: 6px; word-break: break-word; }
.ezfy-page .ezfy-ask-row { margin-top: 4px; }
/* 确认/取消做成有点击区的按钮样式：原来只是两个文字链接，手机上是「看不到的 1 字宽」的靶子，
   手指点不准（用户反馈「交互太离谱」）。 */
.ezfy-page .ezfy-ask-ok,
.ezfy-page .ezfy-ask-cancel {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 3px;
  border: 1px solid #c9c5ba;
  background: #fff;
}
.ezfy-page .ezfy-ask-ok { font-weight: bold; color: #27763c; border-color: #9cc4a6; margin-right: 10px; }
.ezfy-page .ezfy-ask-cancel { color: #777; }
.ezfy-page .ezfy-ask-ok:hover { background: #eef7f0; }
.ezfy-page .ezfy-ask-cancel:hover { background: #f2f2f2; }
/* 底部 15 项导航(复刻原版 cityHome.html 的两行) */
.ezfy-page .ezfy-bottom-nav {
  padding: 1px 0;
  /* ★ 2026-09-25 随全站统一：与首页导航同号（同一个变量） */
  font-size: var(--fs);
  line-height: 1.75;
}
/* ★ 间隔对齐原版 .old-line a 的 margin: 0 1px；配色按 默认 #004299 / 选中 #c0392b */
.ezfy-page .ezfy-bottom-nav a {
  display: inline;
  padding: 0 1px;
  margin: 0 1px;
  color: #004299;
}
.ezfy-page .ezfy-bottom-nav a.on {
  font-weight: bold;
  color: #c0392b;
}
/* 顶部导航里的「家园」——游戏内唯一的出口, 稍微标一下 */
.ezfy-page .top-nav a.ezfy-exit {
  color: #2f4156;
  font-weight: bold;
}
.ezfy-page .panel { margin-top: 8px; padding: 2px; }
.ezfy-page .acade-tab {
  padding: 3px 0;
  font-size: var(--fs);   /* ★ 2026-09-25 随全站统一（原来写死 17px） */
  color: #666;
}
.ezfy-page .acade-tab a { color: #2f4156; }
.ezfy-page .acade-tab a.on { color: #c0392b; font-weight: bold; }
/* ★★ 2026-10-09 导航交互：**当前所在的那一项不可再点**（避免重复点击 → 重复请求/重复加载），
   并且已经用颜色高亮区分（上面三条 .on 规则）。
   覆盖：二级导航(资源/军官/军队/科技/城防/统帅)、底部导航(军事/资源/地图/…/市政)、
        各模块 tab(.acade-tab：军官/计谋/招募…、军情四区、军团六栏、商城/排行/签到…)。
   ⚠️ **顶部「聊天 邮箱 军情 任务 好友 首页」(.top-nav) 不在内** ——
      用户明确要求那几个「点了还能点」（再点=重新拉一遍，例如军情刷新）。
   用 pointer-events:none 而不是改 @click：点击直接落到父元素，处理函数不会执行。 */
.ezfy-page .ezfy-subnav a.on,
.ezfy-page .ezfy-bottom-nav a.on,
.ezfy-page .old-line.home-nav2 a.on,
/* ★ 2026-10-14 筛选网格（商城分类/装备部位/背包分类）同样适用「当前项不可再点」 */
.ezfy-page .ezfy-slot-grid > a.on,
.ezfy-page .acade-tab a.on {
  pointer-events: none;
  cursor: default;
}
/* ★ 2026-09-29 司令部 tab(兵种配置/出征队列/伤兵营/逃兵营/预设编队) 间隔大一点点，
   仅这组生效（其余 acade-tab 不带 hq-tab 类，间隔保持不变） */
.ezfy-page .acade-tab.hq-tab a { margin-right: 10px; }
/* ★ 2026-09-28 tab 之间的「.」分隔符：原写法用 &nbsp;.&nbsp;（不换行空格 U+00A0），
   Windows 中文宋体(SimSun) 里 U+00A0 占一个**全角字宽**(17px)，两个就 34px，
   「军队动态 .  驻军 .  军情警讯」在 Win 上间隔拉到 40px+；
   macOS 回退到 Songti SC / 苹方，其 U+00A0 是半角宽度(约 4~5px)，所以看着正常。
   —— 根因是字形宽度差异，不是谁写错了数值。
   修法：不再引入空格字符，改用 span 承载「.」并显式给左右 margin，
   任何平台、任何字体下宽度都完全一致。 */
.ezfy-page .acade-sep {
  display: inline-block;
  margin: 0 4px;
}
/* ★ 2026-09-28 首页同一行里两个文字链接的间隔。
   ⚠️ 不能用空格字符控制间隔：`&nbsp;`(U+00A0) 在 Windows 宋体下是**全角**(约 16px)、
   在 macOS(Songti SC / 苹方) 下是**半角**(约 4px)，同一份代码两端差 4 倍；
   而源码里的换行只会折叠成 1 个半角空格(约 4px)，又太挤。
   统一改用定宽 span，任何平台/字体下宽度完全一致。
   「[造兵] [建防] 之间少一点点」（原来 &nbsp; 太宽）、
   「城市状态 / 附属野地 之间大一点点」（原来只有一个折叠空格，太挤）。 */
.ezfy-page .home-gap { display: inline-block; width: 10px; height: 1em; }
.ezfy-page .use-box {
  margin: 4px 0 6px 8px;
  padding: 4px 6px;
  border-left: 2px solid #d8d5cc;
  line-height: 1.9;
}
/* ★ 2026-09-29 城市列表：城市名(坐标) 可点击切换；当前城无下划线但保留粗体 */
.ezfy-page .old-line .city-name {
  font-weight: bold;
  color: var(--link, #004299);
  text-decoration: underline;
  cursor: pointer;
  margin-right: 8px;
}
.ezfy-page .old-line span.city-name {
  color: inherit;
  text-decoration: none;
  cursor: default;
}
/* ★ 建筑区操作的内联提示：独占一行贴在所点建筑行下方，带 [关闭]，不自动消失 */
.ezfy-page .build-tip {
  display: block;
  width: 100%;
  margin: 3px 0 4px;
  padding: 3px 8px;
  border-left: 3px solid #c0392b;
  background: #fbf3f2;
  color: #c0392b;
  line-height: 1.6;
  box-sizing: border-box;
}
.ezfy-page .build-tip.ok {
  border-left-color: #2e7d32;
  background: #eef7ee;
  color: #2e7d32;
}
.ezfy-page .build-tip a { margin-left: 8px; color: #999; }
.ezfy-page .panel-title {
  /* ★ 2026-09-25：小标题比正文大一号（正文已统一为 var(--fs)）。
     这是「标题 vs 正文」的层次，不是漏改 —— 要一起拉平说一声即可。 */
  font-size: calc(var(--fs) + 1px);
  font-weight: bold;
  color: #2f4156;
  margin: 6px 0 2px;
}
/* ★ 2026-10-10 参谋部标题里的「去招募 | 战俘营」链接缩小到正文大小（不跟大标题） */
.ezfy-page .panel-title .staff-links {
  font-size: var(--fs);
  font-weight: normal;
}
.ezfy-page .old-line { padding: 2px 0; word-break: break-all; }
/* ★ 2026-10-17 统帅称号徽章（昵称下方、阵营上方，见 modules/EzfyAcademy.vue）。
   简约但有仪式感：深底 + 金边 + 金字的窄标签，前缀小菱形，像军功章绶带；
   按称号类型换底色（t1 战功 赤金 / t2 榜单 紫金 / t3 活动 靛青 / t4 荣誉 银灰）。
   ★ 不写解释性文案、不加卡片/色块背景，保持老式 WAP 文字页的排版节奏。 */
.ezfy-page .ezfy-title-line { padding: 3px 0 2px; }
.ezfy-page .ezfy-title-badge {
  display: inline-block;
  padding: 1px 9px 2px;
  margin: 0 5px 3px 0;
  font-size: 12px;
  line-height: 17px;
  font-weight: bold;
  letter-spacing: 2px;
  border-radius: 2px;
  color: #ffe9b0;
  background: linear-gradient(180deg, #b8342a 0%, #8e1f18 100%);
  border: 1px solid #d9a53c;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, .12), 0 1px 2px rgba(0, 0, 0, .22);
  text-shadow: 0 1px 1px rgba(0, 0, 0, .55);
  vertical-align: middle;
}
.ezfy-page .ezfy-title-badge::before {
  content: '◆';
  margin-right: 4px;
  font-size: 7px;
  letter-spacing: 0;
  color: #f7d478;
  vertical-align: 1px;
}
.ezfy-page .ezfy-title-t1 { background: linear-gradient(180deg, #b8342a 0%, #8e1f18 100%); }
.ezfy-page .ezfy-title-t2 { background: linear-gradient(180deg, #7a2a86 0%, #4a1055 100%); }
.ezfy-page .ezfy-title-t3 { background: linear-gradient(180deg, #1f5f8b 0%, #123a58 100%); }
.ezfy-page .ezfy-title-t4 { background: linear-gradient(180deg, #5c6b7a 0%, #37424d 100%); }
/* ★ 建筑行「升级 / 一键满级 / 拆除」三个操作间隔再大一点（） */
.ezfy-page .build-act { display: inline-block; }
.ezfy-page .build-act a { margin: 0 5px; }
/* ★ 用户反馈「[赏赐…][流放] 这俩按钮之间来点间距」→ 军官详情的操作按钮行统一拉开间距。
   只作用在带 .officer-actions 的行上，不动其它页面的按钮。
   ★ 2026-09-28 「按钮样式去掉」→ 全部改文字链接，这里跟着改为 <a> 的间距。 */
.ezfy-page .old-line.officer-actions a { margin-right: 10px; }
.ezfy-page .old-line.officer-actions a:last-child { margin-right: 0; }
/* ★ 出征确认页(orderpre)分区：① ② ③ … 小标题 + 等宽列网格。
   原来所有内容都是一串 .old-line 平铺，兵种/资源/宿营/计算混在一起，用户反馈「看着好乱」。
   ★ 用户反馈「兵力还是竖着展示，整齐一点」→ 兵力改成 **CSS Grid 等宽列**：
     - 用 grid（不是 flex-wrap）：同一列的宽度完全一致，纵向也对得齐；
     - minmax 让窗口变窄时自动减列（窄屏也不会退化成一兵种一行）；
     - 名称超长用省略号（完整名字放 title），输入框固定宽度，**现有数量独立一列右对齐** ——
       原来是「输入框里塞占位符 0~59108」，框一窄就被截成 0-0，很难看。
   注意：.of-cell 仍保持 .old-line 的 2px 上下 padding，整页行距节奏不变。 */
.ezfy-page .of-sec {
  /* ★ 2026-09-25 随全站统一：写死 17px → var(--fs)（靠加粗+下虚线区分层次） */
  font-size: var(--fs);
  font-weight: bold;
  color: #2f4156;
  margin: 8px 0 2px;
  padding-bottom: 1px;
  border-bottom: 1px dashed #d8d5cc;
}
/* 分区标题里的补充说明（★ 2026-09-25 随全站统一：14 → var(--fs)，靠颜色+不加粗弱化） */
.ezfy-page .of-sec .of-hint { font-size: var(--fs); font-weight: normal; color: #8a8a8a; }
/* ★ 2026-10-07 出征页「自动战斗」选项：方括号链接式（与全站 [出征] / [返回] 一致），
   去掉原生 radio 圆点；选中态变红加粗（同 .acade-tab a.on 口径）；
   打玩家城时整项置灰、不可点（.disabled）。 */
.ezfy-page .of-opt { margin-right: 14px; }
.ezfy-page .of-opt.on { color: #c0392b; font-weight: bold; }
.ezfy-page .of-opt.disabled { color: #b6c2d2; }
.ezfy-page .of-grid {
  display: grid;
  gap: 0 16px;
  align-items: center;
}
/* ★ 2026-09-25 「兵种数量搭配改成一行一个兵种」：
   原来是 3 列网格（一个兵种占一个格子，得横向找），现在**每个兵种独占一行**，行内从左到右：
     兵种名(定宽) │ 现有 N │ ————— 滑动条 ————— │ [数字框] │ [最大]
   滑块与数字框双向联动（拖滑块数字跟着变 / 填数字滑块跟着走），[最大] 一键全带。
   行内分两段：名称+现有（.of-name/.of-avail）与控件段（.of-ctl），
   窄屏时控件段整段换行（不会把滑块挤成 10px 没法拖）。 */
.ezfy-page .of-rows { display: block; }
.ezfy-page .of-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 0;
  line-height: 24px;
  min-width: 0;
}
.ezfy-page .of-row .of-name {
  /* 兵种名定宽 → 每行的「现有 N」和滑块都从同一个 x 开始，纵向严格对齐 */
  flex: 0 0 auto;
  width: 11em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  color: #555;
}
.ezfy-page .of-row .of-avail {
  /* 定宽放得下「现有 24,946,000」（线上资源数是十几位） */
  flex: 0 0 auto;
  width: 9.5em;
  text-align: left;
  color: #8a8a8a;
}
.ezfy-page .of-row .of-ctl {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 auto;
  min-width: 0;
}
.ezfy-page .of-row input.of-range {
  flex: 1 1 auto;
  min-width: 70px;
  height: 20px;
  margin: 0;
  /* 现代浏览器直接给滑块上主题色，不必再手写 ::-webkit-slider-thumb */
  accent-color: #2f6fb5;
  cursor: pointer;
}
.ezfy-page .of-row input.of-num {
  flex: 0 0 auto;
  width: 76px;
  text-align: right;
}
.ezfy-page .of-row .of-max { flex: 0 0 auto; white-space: nowrap; }
/* 城内没有该兵种：整行灰掉（滑块也禁用） */
.ezfy-page .of-row.of-off .of-name,
.ezfy-page .of-row.of-off .of-avail,
.ezfy-page .of-row.of-off .of-max { color: #b3b3b3; }
.ezfy-page .of-row.of-off input.of-range { opacity: .45; }
/* ★ 2026-09-28 出征上限额度已用尽（本兵种本次最多可派 0）：[最大] 点了也没用 → 灰掉，
   与同时被 :disabled 禁用的滑块/数字框保持一致，避免「点了没反应」的困惑。 */
.ezfy-page .of-row .of-max.of-max-off { color: #b3b3b3; cursor: default; }
/* ★ 2026-09-28 训练页 [最大]（紧跟在「(最多 N)」后面）：一键把建造数量填成上限。
   上限为 0 时同样灰掉 —— 与出征页 [最大] 的处理保持一致，避免「点了没反应」的困惑。 */
.ezfy-page .train-max { white-space: nowrap; margin-left: 6px; }
.ezfy-page .train-max.train-max-off { color: #b3b3b3; cursor: default; }
/* ★ 2026-10-25 训练页 [开始训练]：去掉按钮样式（文字链接），1 秒冷却期内灰掉提示不可点 */
.ezfy-page .train-start { white-space: nowrap; }
.ezfy-page .train-start.train-start-off { color: #b3b3b3; cursor: default; }
@media (max-width: 700px) {
  /* 窄屏：名称+现有占第一行，滑块/数字/[最大] 整段换到第二行 */
  .ezfy-page .of-row { flex-wrap: wrap; row-gap: 0; }
  .ezfy-page .of-row .of-name { width: auto; max-width: 60%; }
  .ezfy-page .of-row .of-avail { width: auto; flex: 1 1 auto; }
  .ezfy-page .of-row .of-ctl { flex: 1 1 100%; }
}
/* 随军资源：名称只有 2 个字，列可以窄一点。
   ★ 下限不能太小：线上资源是**十几位**的数（如 14,101,854,318 ≈ 119px），
     230px 的格子装不下「名称+输入框+数量」，数字会顶到隔壁格子的名称上（看着像串行）。
     300px = 名称 4em(64) + 输入 70 + 数量 119 + 间距 10 + 余量。 */
.ezfy-page .of-grid-res { grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); }
.ezfy-page .of-cell {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 2px 0;
  line-height: 24px;
  min-width: 0;
}
.ezfy-page .of-cell .of-name {
  /* ★ 「文字左最起，按照第一列兵种那块」：
     兵种名**定宽**（不随名字长短伸缩），这样每行的输入框都从同一个 x 开始，
     三列在所有行里纵向严格对齐 —— 这才是「左起对齐」。
     ★ 定宽还有一层必要性：第 3 列「现有数量」的数字长度是会变的
       （24,946,000 比 0 宽 15px）。名称若是弹性宽度，长数字会把输入框往左顶，
       同一列里输入框的 x 就不一致了（实测差 15px）—— 定宽才能钉死。
     11em 放得下线上最长的兵种名「齐柏林伯爵级航空母舰」= 10 个汉字(≈160px)，
     原来的 9.6em(154px) 会把最后两个字截成省略号（实测线上就是这样）。 */
  flex: 0 0 auto;
  width: 11em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  color: #555;
}
.ezfy-page .of-cell input.of-num {
  flex: 0 0 auto;
  width: 70px;
  /* ★ 「二列三列文字右对齐」：第 2 列（输入框里的数字）右对齐 */
  text-align: right;
}
/* 第 3 列「现有数量」：紧跟输入框、左起（不再 margin-left:auto 推到格子最右），
   因为输入框是定宽，所以这一列在所有行也从同一个 x 开始，自然对齐 */
.ezfy-page .of-cell .of-avail {
  flex: 0 0 auto;
  min-width: 4.4em;
  text-align: left;
  color: #8a8a8a;
}
/* 资源名只有 2 个字，定宽窄一点，别浪费横向空间 */
.ezfy-page .of-grid-res .of-cell .of-name { width: 4em; }
.ezfy-page .of-cell.of-off .of-name,
.ezfy-page .of-cell.of-off .of-avail { color: #b3b3b3; }
/* ★ 首页【置顶公告】区：公告行本身和其它 .old-line 一样是 28px 高，问题出在**上下留白不等**。
   上方那 4px 额外留白已由 .top-nav 去掉底部 padding 解决（见上），
   这里只需给下方补 2px，让公告行与上一行、下一行的留白相等。
   注意: 用 margin-bottom 而不是「抵消上方」的负 margin —— 因为导航和公告行之间还可能插入
   .ezfy-ask（操作结果提示条），负 margin 会把这 4px 从提示条的下边距里扣掉，
   公告行就会贴住提示条（实测只剩 5px）。 */
.ezfy-page .ezfy-notices { margin: 0 0 2px; }
/* ★ 2026-09-25 随全站统一：写死 17px → var(--fs) */
.ezfy-page .city-name { font-size: var(--fs); font-weight: bold; color: #2f4156; }
/* ★ 表格默认用「原版模板的朴素样式」: 无边框、字号对齐正文。
   ★ 2026-09-24 「表格固定宽度、切 tab 不因字数不一样变动」：
     table-layout:fixed + width:100% → 列宽按列数均分固定，与单元格内容完全无关，
     切换页面/翻页时列不跳；长文本由下方 th/td 的 word-break 自动折行。
     地图格子与战场指挥室两表内容结构特殊，保持原来自适应（见下方覆盖规则）。 */
.ezfy-page table {
  table-layout: fixed;
  width: 100%;
  /* ★ 2026-09-24 「表格靠左展示」：封顶 920px 左对齐，桌面不铺满整行 */
  max-width: 920px;
  margin: 0;
  border-collapse: collapse;
  /* ★ 表格字号对齐正文（2026-09-25 起 = var(--fs)，窄屏也不再单独缩一档） */
  font-size: var(--fs);
}
/* 地图 5×5 格子 / 战场指挥室：恢复按内容自适应，不参与全局固定均分 */
.ezfy-page .ezfy-map-table,
.ezfy-page .ezfy-battle-tbl {
  table-layout: auto;
  width: auto;
  max-width: 100%;
}
.ezfy-page table th,
.ezfy-page table td {
  word-break: break-word;
  overflow-wrap: anywhere;
  border: 0;
  text-align: left;
  vertical-align: top;
  padding: 2px 10px 2px 0;
}
/* ★ 军队总览「城内军队」表：只把单元格内容里左右居中 + 垂直居中（），表格本身不居中 */
.ezfy-page table.ezfy-center-tbl th,
.ezfy-page table.ezfy-center-tbl td {
  text-align: center;
  vertical-align: middle;
}
.ezfy-page table.ezfy-center-tbl td a { margin: 0 3px; }
/* ★ 2026-10-07 图标列（城内军队表）：原来是 width:22px + padding:0，图标紧贴「兵种」列，
   用户反馈「图标和兵种列贴的有点近，稍微远一点点」→ 右侧留 10px（列宽 32 = 图标 22 + 间距 10）。 */
.ezfy-page table.ezfy-center-tbl th.ico-cell,
.ezfy-page table.ezfy-center-tbl td.ico-cell {
  width: 32px;
  padding: 2px 10px 2px 0;
  text-align: left;
}
/* ★ 2026-10-07 军队页图标放大：全局 .ezfy-ico 是 18px，画 20×20 的兵种图形细节会糊成一团
   （用户：「图标还是丑」）→ 这张表里单独放到 22px 并垂直居中。只影响城内军队表，不动全局。 */
.ezfy-page table.ezfy-center-tbl .ezfy-ico {
  width: 22px;
  height: 22px;
  vertical-align: middle;
  margin-right: 0;
}
/* ★ 2026-09-25 「展示兵种名字的列」「军官装备名称的列」以前是整列居中，看着丑 →
   **单元格内容改成左对齐 + 垂直居中**；**表头 th 保持居中不变**。
   用法：给这些「名称类」单元格加 class="nm"（不是按列号，按语义，加列/挪列都不会失效）。
   放置位置：紧跟在 .ezfy-center-tbl / .ezfy-plain-table 的居中规则之后 ——
   选择器权重 (0,2,2) 与 .ezfy-center-tbl 相同、高于 .ezfy-plain-table 的 (0,2,1)，
   靠顺序取胜；以后调那两张表的居中规则不会把这个覆盖掉。
   ⚠️ 只加在「名称」列上（兵种名 / 装备名），数量·等级·属性·操作那些列保持原样，
      别图省事用 td:first-child —— 战场指挥室表的兵种在第 2 列，商城装备表的名称也在第 2 列。 */
.ezfy-page table td.nm {
  text-align: left;
  vertical-align: middle;
}
/* 名称列左对齐后，链接自带的左右 margin 会让文字比列边多缩 3px，去掉更齐 */
.ezfy-page table td.nm a { margin-left: 0; margin-right: 0; }
/* ★ 2026-09-25 当天追加：用户问「对应列标题是不是也居左更好」→ **是**。
   列宽是靠左的（兵种名/装备名都从左边起），表头却居中悬在列中间，两者不在一条竖线上，
   看着就是「标题和数据对不上」—— 表头对齐跟随列内容对齐是通行做法，这里也一样。
   ⚠️ 只打在**该列自己的 th** 上；像「已穿戴装备」「装备背包」那种 colspan 跨列的段标题
     保持居中不变（它是分节标题，不是列标题）。 */
.ezfy-page table th.nm { text-align: left; }
/* ★ 交易行表格美化（2026-09-24 「页面做好看点」）：细边框 + 表头底色 + 斑马纹 */
.ezfy-page table.ezfy-ex-tbl {
  border-collapse: collapse;
  border: 1px solid #cfc9b6;
  margin: 2px 0 4px;
}
.ezfy-page table.ezfy-ex-tbl th,
.ezfy-page table.ezfy-ex-tbl td {
  border: 0;
  text-align: left;
  vertical-align: middle;
  padding: 3px 10px;
  border-right: 1px solid #e3ded0;
}
.ezfy-page table.ezfy-ex-tbl th {
  background: #efe9d9;
  color: #2f4156;
  border-bottom: 1px solid #d9d2bd;
}
.ezfy-page table.ezfy-ex-tbl td { border-bottom: 1px solid #eee9dc; }
.ezfy-page table.ezfy-ex-tbl tr:last-child td { border-bottom: 0; }
.ezfy-page table.ezfy-ex-tbl tr:nth-child(even) td { background: #faf7ee; }
.ezfy-page table th {
  color: #2f4156;
  font-weight: bold;
  white-space: nowrap;
  padding-bottom: 4px;
}
/* ★ WAP 窄屏：固定均分后列会变窄，表头放开折行，长表头（如「装备战斗加成」）不被截断/溢出 */
@media (max-width: 700px) {
  .ezfy-page table th { white-space: normal; word-break: break-word; }
}
/* 返回按钮与 [造兵]/[建防]/[退出军团] 等普通操作链接同款: 纯文字链接, 无填充 */
.ezfy-page .bottom-nav { margin-top: 10px; padding: 4px 0; text-align: left; }
.ezfy-page .footer { text-align: center; font-size: var(--fs); color: #999; padding: 4px 0 10px; }
.ezfy-page .logo-title { height: 14px; vertical-align: -2px; }
/* ★ 首页功能图标(军团/军衔/资源/人口/民心/税率)：统一用内联 SVG 圆角徽章，替换原来风格杂乱的 PNG */
.ezfy-page .ezfy-ico { width: 18px; height: 18px; vertical-align: -4px; margin-right: 3px; }
.ezfy-page .red { color: #c0392b; }
.ezfy-page .gray { color: #999; }
.ezfy-page .green { color: #27763c; }
.ezfy-page .orange { color: #b8860b; }
.ezfy-page .q-normal { color: #999; }
.ezfy-page .q-rare { color: #4dabff; }
.ezfy-page .q-epic { color: #9b59b6; }
.ezfy-page .q-legend { color: #e67e22; }
/* ★ 2026-10-12 二战登录框：统一账号/密码/验证码输入框。
      必须对 box-sizing/padding/border/height/font-size 等都加 !important —— 全局那组
      `.ezfy-page input[type="text"] / input:not([type]) / select / textarea`（本文件后面）
      与 `.wwii-login input.wwii-input` 的优先级**完全相同**（都是 0,2,1），但它写在后面 →
      后写的赢，会把 `font-size` 打回 var(--fs)=16px。
      而密码框是 `input[type="password"]`，**不被那组全局规则命中** → 仍是 14px。
      结果就是「家园号」框 16px、「家园密码」框 14px：文字大小不一、看着没对齐
      （高度同样会被内联 height 与全局 padding 拉扯）。所以这里逐项 !important 锁死。
      (margin-bottom 不加 !important，验证码框用内联 margin-bottom:0 对齐图片) */
.wwii-login input.wwii-input,
.wwii-login input.wwii-input[type="password"] {
  display: block;
  box-sizing: border-box !important;
  width: 100% !important;
  height: 38px !important;
  padding: 10px !important;
  margin: 0 0 10px 0;
  border: 1px solid #e0e0e0 !important;
  border-radius: 4px !important;
  background: #fff !important;
  color: #333 !important;
  font-size: 14px !important;
  line-height: 1.4 !important;
}
.wwii-login input.wwii-input:focus { border-color: #888 !important; }
.wwii-login .wwii-submit {
  background: #2b2b2b;
  color: #fff;
  transition: background .2s;
}
.wwii-login .wwii-submit:hover { background: #444; }
.wwii-login .wwii-submit:disabled { background: #999; cursor: not-allowed; }
.ezfy-page input[type="text"],
.ezfy-page input[type="number"],
.ezfy-page input:not([type]),
.ezfy-page select,
.ezfy-page textarea {
  border: 1px solid #999;
  border-radius: 0;
  padding: 3px 4px;
  /* ★ 2026-09-25 随全站统一：表单 16 → var(--fs)（所有文字一样大） */
  font-size: var(--fs);
  background: #fff;
  color: #333;
}
.ezfy-page button {
  border: 1px solid #888;
  border-radius: 0;
  background: #e8e5dd;
  color: #333;
  /* ★ 2026-09-25 随全站统一：按钮 15 → var(--fs)（所有文字一样大） */
  font-size: var(--fs);
  padding: 2px 8px;
  cursor: pointer;
}
.ezfy-page button:hover { background: #d8d5cc; }
.ezfy-page .report-pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: inherit;
  /* ★ 2026-09-25 随全站统一：战报正文 15 → var(--fs) */
  font-size: var(--fs);
  background: #fff;
  border: 1px solid #ddd;
  padding: 6px;
  margin: 4px 0;
}
/* 地图 */
.ezfy-map { padding: 4px 0; overflow-x: auto; }
.ezfy-map-row { white-space: nowrap; }
/* 复刻 map/index.html 的 5×5 <table>:
   参考**没有任何表格 CSS**, 就是浏览器默认样式 —— 单元格按内容自适应宽度、
   无底色、无边框。所以这里只做三件事:
   ① 抵消全局 `.ezfy-page table td` 的虚线下边框  ② 单元格紧凑  ③ 不折行
   ★ 「格子下面加个坐标，排列整齐一点」→ 每格变成**两行**：
     第一行 名称(等级)，第二行 (x,y)；见下面的 .ezfy-cell-name / .ezfy-cell-xy。 */
.ezfy-page .ezfy-map-table {
  /* ★ 2026-09-25 用户反馈「点击地图表格列宽会变，向上/向下切换时因为字数不一样看着丑」：
     原因 = 浏览器默认的 auto 布局按每格内容宽度分配列宽（「城市」1 格 vs 「海底森林(5)」6 字），
     移动地图/内容一变列宽就跳。改成 table-layout: fixed → 5 列恒等宽，位置稳定。
     ★★ 同一天用户二次纠正：「列宽固定对了，但表格变大了，表格还和之前一样只不过列宽固定」。
        上一版写的是 width:100% + min-width:660px，表格被撑满整个面板（桌面实测 964px、
        列宽 178px）—— 这就是「表格变大」的原因。现在改成**固定总宽**（2026-09-26 定档 582px）：
          改动前：325px(全是短文案) ~ 361px(出现「海底森林(9)」) ← 会随内容变宽变窄（用户不要）
          现在：  恒定宽，fixed 布局把 5 列均分成 90px/列       ← 只由宽度公式决定，永不随内容变
        90px 这一档是 2026-09-26 用户拍板的：他要求「海底森林(10)」**必须一行显示**，
        「不满足的话左右就再加」。所以列宽按最坏情况量 —— 最长的两种文案「海底森林(10)」
        「沿海平原(10)」都是「4 个汉字 + (10)」= 15px×4 + 括号数字 ≈ 86~88px，90px 留余量；
        比它短的必然一行（「平原(10)」58px、「活动寇(9)」68px、「寇(10)」43px）。
        ★ 中途试过「列宽 64px + 名称折两行」（表只要 446px），用户明确要一行，已弃用该方案。
     ★ 列宽能真正定死，下面两条缺一不可（都是实测踩出来的）：
       ① 表格宽度必须是**确定值**，不能 width:auto —— fixed 布局 + width:auto 时 Chrome 仍会
          用内容的 min-content 把列撑宽（实测 325px 的表会被「海底森林(10)」涨到 436px），前功尽弃；
       ② 格子里那个 <a> 必须是 display:block —— inline-block 的 shrink-to-fit 会取 min-content
          （= 整串文字宽 86px），直接溢出格子压到邻格，也不会按格子宽度换行（见下面 a 的规则）。 */
  table-layout: fixed;
  width: 582px;
  min-width: 0;
  border-collapse: separate;
  /* ★ 「坐标和坐标之间间隔小了，上下左右都再来点」→ 8px 3px 放大到 12px 6px；
     随后又要求「上下间隔加一点」→ 纵向 6px → 10px；再次要求「上下坐标间隔再大一些」
     → 纵向 10px → 14px；2026-09-25 又反馈「地图区上下还是紧，上下间距再大一点点」
     → 纵向 14px → 18px；当天看过对比图后**拍板用 22px 那档** → 18px → **22px**。
     ★★ 2026-09-26 「地图坐标左右之间的间隔大一点，但**第一列左边间隔保持不变**」，
        横向最终定档 **22px**（中途试过 20px，用户看过之后说「-2px 去掉吧」→ 回到 22）。
        border-spacing 是**内外一起加**的 —— 第一个值同时管「列与列之间」和「表格左右两边的
        外边距」，所以横向一加大，第一列也会跟着右推；这里用 margin-left 把整表左移抵消
        （负值只吃到表格最左边那段空白，第一列内容仍在表格内）：
          · 第一列左边距 = margin-left + 横向间距 = 12px  ← 与最开始完全一致 ✔
          · 列与列之间   = 横向间距：12 → 20 → **22**（定档）
          · 总宽 = 6×横向间距 + 5×列宽（90，见上面「列宽」那段）→ 6×22 + 5×90 = 582px
        ★ 要再调间距：只改第一个数，另两处按这两条式子跟着改：
            width = 6×横向间距 + 5×90      margin-left = 12 − 横向间距
          窄屏那份在下面的 @media (max-width: 420px) 里（横向 8px、margin-left -2px）。 */
  border-spacing: 22px 22px;
  margin: 8px 0 8px -10px;   /* 左 -10px = 12 − 22，抵消第一列被 border-spacing 右推 */
}
.ezfy-page .ezfy-map-table td {
  padding: 0;
  border: 0;
  text-align: center;        /* 居中指的是「表格里的内容」居中 */
  vertical-align: top;       /* 两行格子按顶对齐, 免得行高不一导致上下抖动 */
  white-space: nowrap;
}
.ezfy-page .ezfy-map-table a {
  /* ★ 必须是 block，不能是 inline-block：fixed 布局下格子宽度是定死的，
     inline-block 的 shrink-to-fit 会取 min-content(=整串文案宽，如「海底森林(10)」83px)，
     结果 <a> 比格子还宽 → 文字压到邻格、省略号也不生效。改成 block 后宽度=格子宽，
     里面的 .ezfy-cell-name 才能正常 overflow:hidden + 省略号。 */
  display: block;            /* 块级容器, 才能装上下两行 + 按格子宽度截断 */
  padding: 0;
  margin: 0;
  /* ★ 第一行(名称/等级，如「海(8)」)的字号：历史 16→15→14，2026-09-25 用户反馈「地图看着小了」
     → 加大 1 号回到 **15px**。坐标行有自己独立的值(14px)，不受这里影响。
     窄屏同理 12 → 13，见下面媒体查询。 */
  font-size: 15px;
  line-height: 1.3;
  /* ★ 「坐标上颜色 + 野地类型也上色，不然玩家不知道能点」→ 两行都用站内链接蓝；
     本城(.ezfy-mine)与活动目标(.ezfy-act-*)的颜色是有含义的，下面单独覆盖，不受影响。 */
  color: #0645ad;
  background: none;
  border: 0;
  text-decoration: none;
  text-align: center;        /* 两行都相对格子中心对齐 */
}
/* 第一行：名称(等级) */
/* ★ 2026-09-26 「海底森林(10)」**一行显示**（先前是 nowrap + 省略号，
   被截成「海底森…」，等级跟着看不见）。列宽已按最长文案放到 90px（见上面「列宽」那段），
   所有文案天然一行，这里不再需要省略号；white-space:normal 只作兜底：万一某台机器字体更宽，
   宁可折行也不截断或压到邻格（CSS 不允许在「(」后断行，折也是折成「海底森林」/「(10)」）。 */
.ezfy-page .ezfy-map-table a .ezfy-cell-name {
  display: block;
  white-space: normal;
}
/* 第二行：坐标 (x,y)。★ 「坐标上颜色，不然玩家不知道能点」→ 站内链接蓝 #0645ad；
   字号定稿过程：12 → 11 →「坐标那行大 1 号」12 →「(272,227) 大 1 号」13px；
   2026-09-25 用户反馈「地图坐标看着小了」→ 加大 1 号到 **14px**。
   ★ margin-top 是「上下坐标之间再大一点点」——第一行缩到 14 后两行几乎一样大，
   需要这点缝把它们分开，不然两行糊成一块。 */
.ezfy-page .ezfy-map-table a .ezfy-cell-xy {
  display: block;
  font-size: 14px;
  line-height: 1.25;
  margin-top: 4px;
  font-weight: normal;       /* 本城/活动城名字加粗, 坐标不跟着加粗 */
  color: #0645ad;
}
/* 本城加粗标一下, 其余一律朴素文字 */
.ezfy-page .ezfy-map-table a.ezfy-mine { font-weight: bold; color: #c0392b; }
/* ★ 2026-10-08 敌人城市标红（个人战争/宣战、军团敌对、军团战争的对象）：
   红色加粗，地图上一眼认出敌人 */
.ezfy-page .ezfy-map-table a.ezfy-enemy { font-weight: bold; color: #ff0000; }
/* 活动目标配色照 mapView.html: 活动野地橙 / 活动寇城品红 / 特殊城市红 */
.ezfy-page .ezfy-map-table a.ezfy-act-wild { font-weight: bold; color: #ff6600; }
.ezfy-page .ezfy-map-table a.ezfy-act-kou { font-weight: bold; color: #ff00ff; }
.ezfy-page .ezfy-map-table a.ezfy-act-city { font-weight: bold; color: #d00000; }
/* ★ 2026-09-30 带名将守将的活动野地：紫红高亮，与普通活动野地(橙)区分 */
.ezfy-page .ezfy-map-table a.ezfy-act-named { font-weight: bold; color: #cc22ff; text-decoration: underline; }
/* ★ 地图方向导航「向上/向右/向下/向左/回到本城」：
   全局 .ezfy-page a 只有 margin: 0 1px, 五个词挤成一串。「间隙稍微大一点」
   → 每个链接右侧留 8px（含标签间空格约 12px 一档），末项不留，右侧不至于飘出去。 */
.ezfy-page .ezfy-dir-nav a {
  display: inline-block;
  margin: 0 8px 0 0;
}
.ezfy-page .ezfy-dir-nav a:last-child { margin-right: 0; }
/* ★ 坐标查找行：横坐标/纵坐标同一行，[查找] 跟行末；窄屏由下面的媒体查询收窄输入框 */
.ezfy-page .ezfy-map-jump input { width: 80px; margin-right: 4px; }
.ezfy-cell {
  display: inline-block;
  width: 36px;
  height: 26px;
  line-height: 26px;
  text-align: center;
  margin: 1px;
  background: #e8f0d8;
  border: 1px solid #b8c89a;
  font-size: 11px;
  cursor: pointer;
  color: #444;
  white-space: nowrap;
}
.ezfy-mine { background: #ffe9b0; border-color: #d0a030; color: #803000; }
.ezfy-city { background: #d8e4f0; border-color: #90a8c0; }
.ezfy-kou { background: #f0d8d8; border-color: #c09090; }
.ezfy-sea { background: #c8e0f0; border-color: #80a8c8; }
.ezfy-wild { background: #e8f0d8; border-color: #b8c89a; }
/* ★ 2026-10-01 地图外越界格：灰色「空地」（不可点），不再显示负坐标地形 */
.ezfy-page .ezfy-map-table .ezfy-empty {
  display: block;
  padding: 0;
  margin: 0;
  font-size: 15px;
  line-height: 1.3;
  color: #b0b0a8;
  background: #f4f4f0;
  border: 0;
}

/* ============ WAP 窄屏适配(手机) ============
   目标: 360px / 320px 下不出现横向溢出, 表格不挤成一坨。
   实测基准: iPhone SE 320、常见安卓 360/390。
   ★ 2026-09-25 「页面文字除了地图全部和首页导航一样大」→
     **这里不再整体缩一档**（原来 表格14/正文15/标题16/导航15 那套阶梯已删除），
     正文·标题·导航·表格·按钮·表单一律沿用 var(--fs)。
   仍然保留缩小的只有两类（属于布局硬约束，不缩就必然溢出）：
     ① 地图格（5 列 × 两行文字）
     ② 战场指挥室表格（7 列，见文件末尾它自己的媒体查询）
   另保留表格单元格内边距（省空间，与字号无关）。 */
@media (max-width: 420px) {
  .ezfy-page table th,
  .ezfy-page table td { padding: 4px 4px; }
  /* 地图格子: 间距按窄屏收紧, 保证 320px 下 5 列不溢出
     ★ 格子已是两行(名称 + 坐标)，窄屏两行都缩一档，行高收紧免得整表变高太多；
       第一行跟着桌面一起缩(15 → 13)，坐标行同样随桌面(14 → 13)，两行之间留同样的缝。 */
  .ezfy-page .ezfy-map-table a { font-size: 12px; line-height: 1.25; }
  .ezfy-page .ezfy-map-table a .ezfy-cell-xy { font-size: 12px; }
  /* 窄屏纵向间距同步收一档(桌面 22px → 窄屏 12px)：纵向间距只影响表格高度、不影响列宽，
     所以这里不需要像横向那样压到极限，留出和桌面接近的呼吸感 */
  /* ★ 2026-09-26 桌面把横向间距加大后，用户反馈「手机端没同步，适配一下」：
     窄屏横向也同步加宽一档 —— 6px → **8px**，并把整表左移 2px 抵消（margin-left = 6 − 8），
     这样**第一列左边距仍是 6px**、和以前一致，只有列与列之间变宽。
     总宽 = 6×8 + 5×52 = 308px（320px 的机器上仍放得下，不触发 .panel 横向滚动）。
     ★ 窄屏名称行仍是「不折行 + 省略号」：列宽只有 ~52px，装不下「海底森林(10)」
       （13px 下约 75px），硬折行会折在名字中间（「海底森」/「林(10)」）反而更难看。
     ★★ 2026-09-27 用户反馈「手机端进海底森林地名展示不全」：name 被省略号截成「海底森…」。
         根因是 5 列 @52px 在 320px 宽的窄屏上无论如何都装不下「海底森林(10)」。
         这次把列宽提到能装下名字的一档（~76px），并顺手把左右间距 8px → **10px**：
         总宽 = 6×10 + 5×76 = 440px > 屏幕宽，由 .panel 的 overflow-x:auto 兜底横向滑动，
         好处是地名整条显示（不再省略）、间距更松；代价是窄屏要看全 5 列需左右滑动。
         （若更希望窄屏不滑动、整表一屏放下，则必须接受地名省略，二选一。） */
  .ezfy-page .ezfy-map-table { width: 440px; border-spacing: 10px 12px; margin-left: -4px; }
  /* ★★ 2026-09-27 地名还是被省略号截掉(「海底森…」)：根本原因是上面的 ellipsis 规则在
     窄屏仍生效，而「海底森林(9/10)」≈73~80px，卡在 76px 列宽边界内会因 padding 超宽被截。
     这里在窄屏直接覆盖成**不省略、格子内能折行**：宁可折行也绝不丢字；
     配合字体 13→12px 后「海底森林(10)」仅约 64px，可一行放下、基本不会触发折行。 */
  .ezfy-page .ezfy-map-table a .ezfy-cell-name {
    overflow: visible; text-overflow: clip; white-space: normal; word-break: break-all;
  }
  /* 坐标查找行在 320px 下也要待在一行内 */
  .ezfy-page .ezfy-map-jump input { width: 62px; margin-right: 2px; }
  /* 方向导航窄屏间距同步收一档(桌面 8px → 窄屏 6px) */
  .ezfy-page .ezfy-dir-nav a { margin-right: 6px; }
  .ezfy-page input, .ezfy-page select { max-width: 100%; }
  /* 出征页兵力行：320px 下名称列收窄，滑块才有拖动空间（字号不缩，仍 var(--fs)） */
  .ezfy-page .of-row .of-name { width: 9em; }
  .ezfy-page .of-row input.of-num { width: 64px; }
  /* 随军资源格子：名称+输入框+现有数量在 320px 下要放得下 */
  .ezfy-page .of-cell input.of-num { width: 56px; }
  .ezfy-page .of-cell .of-avail { min-width: 3.6em; }
}
/* 最后一道保险: 万一还有个别元素偏宽, 让它在页面内滚动而不是把整页撑开 */
.ezfy-page .panel { max-width: 100%; overflow-x: auto; }
/* ============ 军团三页（军团信息 / 军团外交 / 军团宣战）WAP 适配（2026-09-28 ） ============
   手机上：宽表格（成员 5 列 / 外交 7 列 / 宣战 7 列）挤成一坨 → 隐藏次要列 + 收紧内边距；
   输入框/下拉（内联 width 15%~60%）过宽 → 收窄并限制 max-width。 */
@media (max-width: 700px) {
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td { padding: 4px 4px; }
  /* ★ 2026-09-28 军团信息/外交/宣战 多列表格：手机端改为横向滚动, 不再硬塞窄屏挤压错位;
       单元格不换行, 超出面板左右滑动即可看全, 布局不乱 */
  .ezfy-page .ezfy-corps-tbl,
  .ezfy-page .ezfy-dip-tbl,
  .ezfy-page .ezfy-war-tbl,
  .ezfy-page .ezfy-war-list-tbl {
    display: block;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    white-space: nowrap;
    width: 100%;
  }
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td,
  .ezfy-page .ezfy-dip-tbl th,
  .ezfy-page .ezfy-dip-tbl td,
  .ezfy-page .ezfy-war-tbl th,
  .ezfy-page .ezfy-war-tbl td,
  .ezfy-page .ezfy-war-list-tbl th,
  .ezfy-page .ezfy-war-list-tbl td {
    white-space: nowrap;
  }
  /* 输入框/下拉：手机上一律收窄（!important 覆盖内联宽度） */
  .ezfy-page select.corps-kick-sel { width: 55vw !important; max-width: 200px; }
  .ezfy-page input.corps-mail-input { width: 55vw !important; max-width: 240px; }
  .ezfy-page input.corps-msg-input { width: 40vw !important; max-width: 180px; }
  .ezfy-page input.corps-name-input { width: 45vw !important; max-width: 180px; }
  /* ★ 2026-09-28 军团信息/外交/宣战：手机端输入类占整行, 按钮自动换行, 不再窄屏挤一起 */
  .ezfy-page input.corps-mail-input,
  .ezfy-page input.corps-msg-input,
  .ezfy-page input.corps-name-input,
  .ezfy-page select.corps-kick-sel {
    display: block;
    margin: 3px 0;
  }
  /* 外交「全部军团」7 列：藏次要列「团长」「积分」 */
  .ezfy-page .ezfy-dip-tbl th:nth-child(2),
  .ezfy-page .ezfy-dip-tbl td:nth-child(2),
  .ezfy-page .ezfy-dip-tbl th:nth-child(4),
  .ezfy-page .ezfy-dip-tbl td:nth-child(4) { display: none; }
  /* 宣战「记录」7 列：藏次要列「我方身份」「宣告时间」 */
  .ezfy-page .ezfy-war-tbl th:nth-child(2),
  .ezfy-page .ezfy-war-tbl td:nth-child(2),
  .ezfy-page .ezfy-war-tbl th:nth-child(4),
  .ezfy-page .ezfy-war-tbl td:nth-child(4) { display: none; }
  /* 宣战「全部军团(可宣战)」5 列：藏「团长」 */
  .ezfy-page .ezfy-war-list-tbl th:nth-child(2),
  .ezfy-page .ezfy-war-list-tbl td:nth-child(2) { display: none; }
}
@media (max-width: 420px) {
  .ezfy-page .ezfy-corps-tbl { font-size: 12px; }
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td { padding: 3px 3px; }
  /* 成员表「任命」列（军团长：[副团长][参谋长][撤职]）竖排，加大触点 */
  .ezfy-page .ezfy-mem-tbl td a { display: inline-block; margin: 2px 0; }
  /* 外交「全部军团」再藏「人数」 */
  .ezfy-page .ezfy-dip-tbl th:nth-child(3),
  .ezfy-page .ezfy-dip-tbl td:nth-child(3) { display: none; }
  /* 宣战「记录」再藏「对方战绩」 */
  .ezfy-page .ezfy-war-tbl th:nth-child(6),
  .ezfy-page .ezfy-war-tbl td:nth-child(6) { display: none; }
}
/* ============ 战场指挥室 WAP 适配 ============
   指挥室表格 7 列（方/兵种/剩余/初始/位置/目标/指挥）在手机上挤成一坨，
   窄屏逐步收缩：≤700px 藏「初始」、指挥按钮竖排加大触点；≤420px 再藏「位置」、下拉收紧。 */
@media (max-width: 700px) {
  .ezfy-page .ezfy-battle-tbl th:nth-child(4),
  .ezfy-page .ezfy-battle-tbl td:nth-child(4) { display: none; }
  .ezfy-page .ezfy-battle-tbl td.ezfy-cmd a {
    display: block;
    margin: 4px 0;
    font-size: 15px;
  }
  .ezfy-page .ezfy-battle-cmds a {
    display: inline-block;
    margin: 4px 10px 4px 0;
    font-size: 15px;
  }
  .ezfy-page .ezfy-battle-tbl th,
  .ezfy-page .ezfy-battle-tbl td { padding: 6px 5px; }
}
@media (max-width: 420px) {
  .ezfy-page .ezfy-battle-tbl th:nth-child(5),
  .ezfy-page .ezfy-battle-tbl td:nth-child(5) { display: none; }
  .ezfy-page .ezfy-battle-tbl { font-size: 12px; }
  .ezfy-page .ezfy-battle-tbl select {
    width: 76px !important; /* 覆盖内联 width:96px */
    max-width: 24vw;
    font-size: 12px;
    padding: 1px 0;
  }
}
/* ★ 分页条（军情三区 / 公告 / 聊天 / 邮件 / 私信 / 背包 / 商城…共用一套）
   ★ 2026-10-21 用户「分页不对，居中干啥，变左边去和原来位置一样」→ 左对齐。
   ★ 2026-10-25 用户「分页间隔太大了，按一个空格间隔吧，就像 上一页 1/4 下一页，位置不变」：
     原固定三栏（4.5em / 10em / 4.5em）把「1/4」居中在 10em 里 → 两侧撑出大片空白。
     改 flex 左对齐 + gap 0.4em（≈一个半角空格宽），条子起始位置不变，仅把间距压回单空格。 */
.ezfy-page .ezfy-pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4em;
  margin: 8px 0 4px;
  /* ★ 2026-09-25 随全站统一：分页条 15 → var(--fs)（所有文字一样大） */
  font-size: var(--fs);
}
.ezfy-page .ezfy-pager > a,
.ezfy-page .ezfy-pager > span {
  margin: 0;
  white-space: nowrap;
}
.ezfy-page .ezfy-pager > a {
  text-decoration: none;
}
.ezfy-page .ezfy-pager a.gray {
  color: #999;
  pointer-events: none;
}
/* 极窄屏（≤350px）：字号降 2px（间距由 gap 控制，不再需要三栏宽度）。 */
@media (max-width: 350px) {
  .ezfy-page .ezfy-pager {
    font-size: calc(var(--fs) - 2px);
  }
}
</style>
