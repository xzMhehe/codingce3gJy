<template>
  <div class="ezfy-page">
    <div class="home-wrap">
      <div class="title-bar">二战风云-红色警戒【1区】</div>

      <!-- 顶部导航(每页都有, 复刻原版) -->
      <div class="top-nav">
        <a href="javascript:;" @click="go('chat')">聊天</a>
        <a href="javascript:;" @click="go('mail')">邮箱</a>
        <a href="javascript:;" @click="go('reports')">军情</a>
        <a href="javascript:;" @click="go('tasks')">任务</a>
        <a href="javascript:;" @click="go('friends')">好友</a>
        <a href="javascript:;" @click="go('home')">首页</a>
      </div>

      <!-- ============ 首页(cityHome) ============ -->
      <template v-if="cur === 'home'">
        <div class="old-line" v-for="n in notices.slice(0, 2)" :key="'n' + n.id">
          <img class="logo-title" src="/static/ezfy/notice.gif" alt="."/>
          <a class="red" href="javascript:;" @click="openNotice(n)">{{ n.title }}</a>
        </div>

        <div class="old-line">
          <span class="city-name">{{ city.name }}({{ city.x }},{{ city.y }})</span>
          <a href="javascript:;" @click="go('cities')">切换城市</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/corps.png" title="军团" alt="."/>
          军团：
          <a href="javascript:;" @click="go('corps')" v-if="!myCorps">加入军团</a>
          <a href="javascript:;" @click="go('corps')" v-else>[{{ myCorps.name }}]</a>
        </div>
        <div class="old-line">声望：{{ profile.prestige }}</div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/jx.png" title="军衔" alt="."/>
          <a href="javascript:;" @click="go('rank')">军衔</a>:{{ rankName }}
        </div>
        <div class="old-line">每日签到：<a href="javascript:;" @click="go('welfare')">签到</a></div>

        <div class="old-line">
          <a href="javascript:;" @click="go('builds')">资源</a>.
          <a href="javascript:;" @click="go('acade')">军官</a>.
          <a href="javascript:;" @click="go('troops')">军队</a>.
          <a href="javascript:;" @click="go('techs')">科技</a>.
          <a href="javascript:;" @click="go('defence')">城防</a>.
          <a href="javascript:;" @click="go('info')">统帅</a>
        </div>

        <div class="old-line">现有资源/产量:
          <a href="javascript:;" @click="go('exchange')">购买</a>
          <a href="javascript:;" @click="go('mall')">增产</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/gold.png" title="黄金" alt="."/>
          <a href="javascript:;" @click="go('res/gold')">黄金:</a>{{ city.gold }}/{{ city.gold_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/rice.png" title="粮食" alt="."/>
          <a href="javascript:;" @click="go('res/food')">粮食:</a>{{ city.food }}/{{ city.food_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/steel.png" title="钢铁" alt="."/>
          <a href="javascript:;" @click="go('res/steel')">钢铁:</a>{{ city.steel }}/{{ city.steel_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/oil.png" title="石油" alt="."/>
          <a href="javascript:;" @click="go('res/oil')">石油:</a>{{ city.oil }}/{{ city.oil_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/mine.png" title="稀矿" alt="."/>
          <a href="javascript:;" @click="go('res/rare')">稀矿:</a>{{ city.rare }}/{{ city.rare_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/person.png" title="人口" alt="."/>
          人口/空闲:{{ city.pop }}/{{ freePop }}
          <a href="javascript:;" @click="go('convene')">召集</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/feelings.png" title="民心" alt="."/>
          民心/民怨:{{ city.feelings }}/{{ city.grievance }}
          <a href="javascript:;" @click="go('placate')">安抚</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/tax.png" title="税率" alt="."/>
          <a href="javascript:;" @click="go('taxset')">税率:</a>{{ city.tax_rate }}%
        </div>

        <div class="old-line">
          <a href="javascript:;" @click="go('buildm')">军事区</a>&nbsp;
          <a href="javascript:;" @click="go('buildm')">建造</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('builds')">资源区</a>&nbsp;
          <a href="javascript:;" @click="go('builds')">建造</a>
        </div>
        <div class="old-line">
          训练军队
          <a href="javascript:;" @click="go('troop')">[造兵]</a>&nbsp;
          <a href="javascript:;" @click="go('defence')">[建防]</a>
        </div>
        <div class="old-line">
          前往
          <a href="javascript:;" @click="go('map')">地图</a>
          出征
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('techs')">研究科技</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('citystatus')">城市状态</a>
          <a href="javascript:;" @click="go('wilds')">附属野地</a>
        </div>
        <div class="old-line">【世界聊天】<a href="javascript:;" @click="go('chat')">进入</a></div>
        <div class="old-line" v-for="ch in worldChats.slice(-5)" :key="'wc' + ch.id">
          [{{ ch.user_name }}]{{ ch.content }}
        </div>

        <br/>
        <div class="old-line">
          <a href="javascript:;" @click="go('buildm')">军事</a>
          <a href="javascript:;" @click="go('builds')">资源</a>
          <a href="javascript:;" @click="go('map')">地图</a>
          <a href="javascript:;" @click="go('corps')">军团</a>
          <a href="javascript:;" @click="go('rank')">排行</a>
          <a href="javascript:;" @click="go('bag')">背包</a>
          <a href="javascript:;" @click="go('mall')">商城</a>
          <a href="javascript:;" @click="go('acade')">宝物</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('activity')">活动</a>
          <a href="javascript:;" @click="go('welfare')">福利</a>
          <a href="javascript:;" @click="go('notices')">公告</a>
          <a href="javascript:;" @click="go('exchange')">交易</a>
          <a href="javascript:;" @click="go('liaison')">联络</a>
          <a href="javascript:;" @click="go('cityhall')">市政</a>
          <a href="javascript:;" @click="go('chat')">聊天</a>
        </div>
      </template>

      <!-- ============ 世界聊天(chat) ============ -->
      <template v-else-if="cur === 'chat'">
        <div class="panel">
          <div class="panel-title">聊天频道</div>
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: chatChannel === 1 }" @click="switchChannel(1)">公共</a>|
            <a v-if="chatHasCorps" href="javascript:;" :class="{ on: chatChannel === 2 }" @click="switchChannel(2)">军团</a>|
            <a href="javascript:;" :class="{ on: chatChannel === 4 }" @click="switchChannel(4)">系统</a>|
            <a href="javascript:;" @click="go('mail')">私聊</a>
          </div>

          <!-- 系统频道: 系统公告 + 系统消息(只读) -->
          <template v-if="chatChannel === 4">
            <div class="panel-title">系统公告</div>
            <div class="old-line" v-for="n in chatNotices" :key="'cn' + n.id">
              <span v-if="n.is_top" class="red">[置顶]</span>
              <b>{{ n.title }}</b><br/>
              <span class="gray">{{ n.content }}</span>
            </div>
            <div class="old-line gray" v-if="!chatNotices.length">(暂无系统公告)</div>
            <div class="panel-title">系统消息</div>
            <div class="old-line" v-for="ch in worldChats" :key="'cs' + ch.id">
              <span class="orange">[系统]</span>
              <span class="gray">{{ fmtTime(ch.created_at) }}</span>
              {{ ch.user_name }}说: {{ ch.content }}
            </div>
            <div class="old-line gray" v-if="!worldChats.length">(暂无系统消息)</div>
          </template>

          <!-- 公共 / 军团频道 -->
          <template v-else>
            <div class="panel-title">
              {{ chatChannel === 2 ? '军团聊天(' + chatCorpsName + ')' : '世界聊天' }}({{ chatPlayers }}人)
            </div>
            <div class="old-line gray">每次发言消耗一个喇叭(最大25个字)</div>
            <div class="old-line" v-for="ch in worldChats" :key="'c' + ch.id">
              [<span class="orange">{{ chatChannel === 2 ? '军团' : '公共' }}</span>]
              <span class="gray">{{ fmtTime(ch.created_at) }}</span>
              <a href="javascript:;" @click="openUser(ch.user_id)">{{ ch.user_name }}</a> 说: {{ ch.content }}
            </div>
            <div class="old-line" v-if="!worldChats.length">(暂无消息, 快来说点什么吧)</div>
          </template>

          <br/>
          <template v-if="chatCanSend">
            <input v-model="chatMsg" style="width:72%" maxlength="25" @keyup.enter="doChatSend"/>
            <button v-if="chatCooldown <= 0" @click="doChatSend">发送</button>
            <button v-else disabled class="gray">冷却中 {{ chatCooldown }}s</button>
          </template>
          <span v-else class="gray">(系统频道仅系统可发言)</span>
          <button @click="loadChats">刷新</button>
        </div>
      </template>

      <!-- ============ 邮箱(mail) ============ -->
      <template v-else-if="cur === 'mail'">
        <div class="panel">
          <div class="panel-title">邮箱(家园私信)</div>
          <div class="old-line" v-for="m in mails" :key="'m' + m.id">
            <span :class="{ red: m.is_read === 0 }">{{ m.sender }}</span>:
            {{ m.content }} <span class="gray">({{ fmtTime(m.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!mails.length">(暂无私信)</div>
          <br/>
          <button @click="loadMails">刷新</button>
        </div>
      </template>

      <!-- ============ 情报/军情(reports) ============ -->
      <template v-else-if="cur === 'reports'">
        <div class="panel">
          <div class="panel-title">军情战报 <button @click="loadReports">刷新</button></div>
          <div class="old-line" v-for="r in reports" :key="'r' + r.id">
            <a href="javascript:;" @click="openReport(r)">
              <span v-if="r.is_read === 0" class="red">[新]</span>{{ r.title }}</a>
            <span class="gray">({{ fmtTime(r.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!reports.length">(暂无战报)</div>
          <template v-if="curReport">
            <div class="panel-title">{{ curReport.title }}</div>
            <pre class="report-pre">{{ curReport.content }}</pre>
            <template v-if="curReport.detail">
              <div class="old-line"><a href="javascript:;" @click="showDetail = !showDetail">[展开/收起逐回合详情]</a></div>
              <pre class="report-pre" v-if="showDetail">{{ curReport.detail }}</pre>
            </template>
          </template>
        </div>
      </template>

      <!-- ============ 好友(friends) ============ -->
      <template v-else-if="cur === 'friends'">
        <div class="panel">
          <div class="panel-title">搜索玩家(按家园号码或昵称)</div>
          <div class="old-line">
            <input v-model="friendKeyword" placeholder="输入家园号码或昵称" style="width:150px"/>
            <button @click="doFriendSearch">[搜索]</button>
          </div>
          <table v-if="friendSearchDone">
            <tr><th>号码</th><th>昵称</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="u in friendSearchList" :key="'fs' + u.id">
              <td>{{ u.num }}</td>
              <td>{{ u.nickname }}</td>
              <td>Lv.{{ u.level }}</td>
              <td>
                <span :class="u.online ? 'green' : 'gray'">{{ u.online ? '在线' : '离线' }}</span>
              </td>
              <td>
                <span v-if="u.is_friend" class="gray">已是好友</span>
                <span v-else-if="u.applied" class="orange">已申请</span>
                <a v-else href="javascript:;" @click="doAddFriend(u)">[加好友]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="friendSearchDone && !friendSearchList.length">(没找到这位友友, 换个号码或昵称试试)</div>
        </div>

        <div class="panel">
          <div class="panel-title">家园好友({{ friends.length }})</div>
          <table>
            <tr><th>昵称</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="f in friends" :key="'f' + f.id">
              <td>{{ f.nickname }}</td>
              <td>Lv.{{ f.level }}</td>
              <td><span :class="f.online ? 'green' : 'gray'">{{ f.online ? '在线' : '离线' }}</span></td>
              <td>
                <a href="javascript:;" @click="go('mail')">[私聊]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!friends.length">(还没有好友, 用上面的搜索找找老友吧)</div>
        </div>
      </template>

      <!-- ============ 任务(tasks) ============ -->
      <template v-else-if="cur === 'tasks'">
        <template v-for="g in taskGroups">
          <div class="panel-title" :key="'tg' + g.id">{{ g.name }}<span v-if="g.reset_type === 1">(每日)</span></div>
          <div class="panel" :key="'gl' + g.id">
            <div class="old-line" v-for="t in g.tasks" :key="t.id">
              <b>{{ t.name }}</b> {{ t.current }}/{{ t.target }}
              <span v-if="t.status === 2" class="gray">[已领取]</span>
              <a v-else-if="t.status === 1" href="javascript:;" @click="doAward(t)">[领奖]</a>
              <br/>
              <span class="gray">奖励:{{ rewardText(t.reward) }}</span>
            </div>
          </div>
        </template>
      </template>

      <!-- ============ 城市列表(cities) ============ -->
      <template v-else-if="cur === 'cities'">
        <div class="panel">
          <div class="panel-title">我的城市列表</div>
          <div class="old-line" v-for="ct in cities" :key="'ct' + ct.id">
            <b>{{ ct.name }}</b><span v-if="ct.id === city.id" class="red">[当前]</span><br/>
            坐标({{ ct.x }},{{ ct.y }}) 城级{{ ct.city_level }}
            金{{ ct.gold }} 粮{{ ct.food }} 钢{{ ct.steel }} 油{{ ct.oil }} 稀矿{{ ct.rare }}<br/>
            <a v-if="ct.id !== city.id" href="javascript:;" @click="doSwitch(ct)">[切换]</a>
            <a href="javascript:;" @click="go('rename')">[改名]</a>
          </div>
          <br/>
          <div class="panel-title">平原起新城 (消耗10万黄金)</div>
          <div class="old-line">
            坐标X: <input v-model="newCityX" type="number" style="width:70px"/>
            坐标Y: <input v-model="newCityY" type="number" style="width:70px"/>
            <button @click="doCreateCity">建新城</button>
          </div>
          <div class="gray" style="font-size:12px">只能在平原建造; 新城自带基础建筑(市政厅/民居/农田1级), 建造后可在上方列表切换操作。</div>
        </div>
      </template>

      <!-- ============ 资源详情(res/:type) ============ -->
      <template v-else-if="cur === 'res'">
        <div class="panel" v-if="resDetail">
          资源详情:<br/>
          名称: {{ resNames[resType] }}<br/>
          储量: {{ resDetail.stock }} / 容纳: {{ resDetail.cap }}
          <span v-if="resDetail.store_tech > 0"> [储存技术Lv{{ resDetail.store_tech }}: 容量+{{ resDetail.store_tech * 2 }}%]</span>
          <br/>
          基础产量(每小时): {{ resDetail.base }}
          <span v-if="resDetail.tech_prod > 0"> [科技+{{ resDetail.tech_prod * 10 }}%]</span>
          <br/>
          加成产量(每小时): {{ resDetail.bonus }}<br/>
          耗量(每小时): {{ resDetail.consume }}<br/>
          <template v-if="resType === 'food'">
            军队耗粮: {{ resDetail.troop_consume || 0 }}
            <span v-if="resDetail.supply_tech > 0"> [补给技巧Lv{{ resDetail.supply_tech }}: -{{ resDetail.supply_tech * 2 }}%]</span>
            <span v-else class="gray"> [补给技巧未研究, 无减免]</span><br/>
          </template>
          总产量(每小时): <span :class="{ red: resDetail.total < 0 }">{{ resDetail.total }}</span><br/>
          <br/>
          {{ resDes[resType] }}
          <br/>
          <a href="javascript:;" @click="go('builds')">[资源区]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 建筑区(buildm/builds) ============ -->
      <template v-else-if="cur === 'buildm' || cur === 'builds'">
        <div class="panel">
          <div class="panel-title">{{ cur === 'buildm' ? '军事区' : '资源区' }}
            <span class="gray">({{ areaCount }}/{{ areaCap }})</span></div>
          <div class="old-line">
            <a href="javascript:;" @click="go(cur === 'buildm' ? 'builds' : 'buildm')">[{{ cur === 'buildm' ? '资源区' : '军事区' }}]</a>
            <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
          <div class="old-line">建造中队列数：{{ buildQueueCount }}</div>
          <div class="old-line">
            数量/最大：{{ areaCount }}/{{ areaCap }}
          </div>
          <div class="old-line" v-for="b in zoneBuildings" :key="b.id ? ('zb-b' + b.id) : ('zb-p' + b.building_id)">
            <template v-if="b.id">
              <b>{{ b.name }}</b>
              <a v-if="bEntry(b.building_id)" href="javascript:;" @click="goEntry(b.building_id)">[{{ bEntry(b.building_id).label }}]</a>
              ({{ b.level }}级)
              <span v-if="b.status === 0">{{ b.effect }}</span>
              <span v-else class="orange">施工中 {{ remain(b.end_time) }}
                <a href="javascript:;" @click="doSpeedBuilding()">[加速]</a></span>
              <br/>
              <span v-if="b.status === 0 && b.level > 0 && b.level < b.max_level">
                <a href="javascript:;" @click="doUpgrade(b)">[升级]</a>
                <a href="javascript:;" @click="doMaxLevel(b)">[一键{{ b.max_level - 1 }}级]</a>
              </span>
              <span v-if="b.status === 0 && b.level === 0"><a href="javascript:;" @click="doUpgrade(b)">[建成中待完成]</a></span>
              <span v-if="b.can_delete === 1 && b.status === 0 && b.level > 0"><a href="javascript:;" @click="doDeleteBuilding(b)">[拆除]</a></span>
              <span v-if="b.next_effect" class="gray">下一级:{{ b.next_effect }}</span>
            </template>
            <template v-else>
              <b>{{ b.name }}</b>(可建造<template v-if="b.built_count">, 已建{{ b.built_count }}个</template>)<br/>
              {{ b.des }}<br/>
              <a href="javascript:;" @click="doBuild(b)">[建造]</a>
              <span class="gray">造价: 粮{{ b.cost.food }} 钢{{ b.cost.steel }} 油{{ b.cost.oil }} 稀{{ b.cost.rare }} 金{{ b.cost.gold }} 需{{ Math.ceil(b.time / 60) }}分钟</span>
            </template>
            <br/>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 造兵(troop) ============ -->
      <template v-else-if="cur === 'troop'">
        <div class="panel">
          <div class="panel-title">训练军队(军工厂合计{{ factoryTotal }}级, 队列{{ queues.length }}/{{ factoryTotal }})</div>
          <div class="old-line green" v-if="troopsData.train_discount > 0">
            节日活动·造兵打折：资源消耗 -{{ troopsData.train_discount }}%（下方为折后价）
          </div>
          <div class="old-line">人口:{{ troopsData.pop }} 空闲:{{ freePop }} | 围墙:{{ troopsData.wall_level }}级</div>
          <div class="old-line" v-for="t in trainCfgs" :key="'tt' + t.id">
            <b>{{ t.name }}</b>({{ troopTypeName(t.type) }}) 血{{ t.health }} 防{{ t.defence }} 速{{ t.speed }} 射程{{ t.attack_range }} 负重{{ t.carry }}<br/>
            消耗: 粮{{ t.cost.food }} 钢{{ t.cost.steel }} 油{{ t.cost.oil }} 稀{{ t.cost.rare }} 训练{{ t.train_time }}秒/个<br/>
            前提: {{ t.require || '无' }}
            <a href="javascript:;" @click="openTrain(t)">[训练]</a><br/>
          </div>
          <template v-if="trainSel">
            <div class="panel-title">训练 {{ trainSel.name }}</div>
            <div class="old-line">
              数量: <input v-model="trainCount" type="number" min="1" style="width:80px"/>
              <label><input type="checkbox" v-model="trainSplit"/>分批(多军工厂同时训练)</label><br/>
              预计耗时: {{ Math.ceil(trainSel.train_time * (trainCount || 0) / (trainSplit ? Math.max(1, factoryFree) : 1) / 60) }}分钟<br/>
              <button @click="doTrain()">开始训练</button>
            </div>
          </template>
          <div class="panel-title">训练队列({{ queues.length }})</div>
          <div class="old-line" v-for="q in queues" :key="'q' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
          </div>
          <div class="old-line" v-if="!queues.length">(队列为空)</div>
          <a href="javascript:;" @click="go('defence')">[去建城防]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 建防(defence) ============ -->
      <template v-else-if="cur === 'defence'">
        <div class="panel">
          <div class="panel-title">城防设施(围墙{{ troopsData.wall_level }}级, 城防空间受限)</div>
          <div class="old-line" v-for="t in defenceCfgs" :key="'dt' + t.id">
            <b>{{ t.name }}</b>({{ troopTypeName(t.type) }}) 血{{ t.health }} 防{{ t.defence }} 射程{{ t.attack_range }}<br/>
            消耗: 粮{{ t.cost.food }} 钢{{ t.cost.steel }} 油{{ t.cost.oil }} 稀{{ t.cost.rare }} 训练{{ t.train_time }}秒/个<br/>
            前提: {{ t.require || '无' }}
            <a href="javascript:;" @click="openTrain(t)">[建造]</a><br/>
          </div>
          <template v-if="trainSel">
            <div class="panel-title">建造 {{ trainSel.name }}</div>
            <div class="old-line">
              数量: <input v-model="trainCount" type="number" min="1" style="width:80px"/><br/>
              <button @click="doTrain()">开始建造</button>
            </div>
          </template>
          <div class="panel-title">现有城防</div>
          <div class="old-line" v-for="t in defenceTroops" :key="'dft' + t.troop_id">
            {{ t.name }}×{{ t.count }}
          </div>
          <div class="old-line" v-if="!defenceTroops.length">(尚无城防设施)</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军队总览(troops) ============ -->
      <template v-else-if="cur === 'troops'">
        <div class="panel">
          <div class="panel-title">城内军队</div>
          <table>
            <tr><th>兵种</th><th>类型</th><th>数量</th></tr>
            <tr v-for="t in troopsData.troops" :key="'tv' + t.troop_id">
              <td>{{ t.name }}</td><td>{{ troopTypeName(t.type) }}</td><td>{{ t.count }}</td>
            </tr>
          </table>
          <div class="old-line" v-if="!troopsData.troops.length">(城内无部队)</div>
          <br/>
          <div class="panel-title">训练队列({{ queues.length }})</div>
          <div class="old-line" v-for="q in queues" :key="'tq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
          </div>
          <div class="old-line" v-if="!queues.length">(队列为空)</div>
          <br/>
          <a href="javascript:;" @click="go('troop')">[造兵]</a>
          <a href="javascript:;" @click="go('defence')">[建防]</a>
          <a href="javascript:;" @click="go('hq')">[司令部]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 司令部(hq) ============ -->
      <template v-else-if="cur === 'hq'">
        <div class="panel">
          <div class="panel-title">司令部: 兵种战斗配置</div>
          <div class="old-line gray">每个兵种可分别设置 进攻/防守 的默认攻击对象与前进停止:</div>
          <table>
            <tr><th>兵种</th><th>进攻目标</th><th>进攻</th><th>防守目标</th><th>防守</th></tr>
            <tr v-for="t in troopsData.cfgs" :key="'cfg' + t.id">
              <td>{{ t.name }}</td>
              <td>
                <select v-model="targetCfg[t.id].atk" style="width:80px">
                  <option :value="0">最近目标</option>
                  <option v-for="tt in troopsData.cfgs" :key="'a' + tt.id" :value="tt.id">{{ tt.name }}</option>
                </select>
              </td>
              <td>
                <select v-model="targetCfg[t.id].atkMove" style="width:56px">
                  <option :value="1">前进</option><option :value="0">停止</option>
                </select>
              </td>
              <td>
                <select v-model="targetCfg[t.id].def" style="width:80px">
                  <option :value="0">最近目标</option>
                  <option v-for="tt in troopsData.cfgs" :key="'d' + tt.id" :value="tt.id">{{ tt.name }}</option>
                </select>
              </td>
              <td>
                <select v-model="targetCfg[t.id].defMove" style="width:56px">
                  <option :value="1">前进</option><option :value="0">停止</option>
                </select>
              </td>
            </tr>
          </table>
          <div class="old-line"><button @click="doSaveTargets">[保存全部配置]</button></div>
          <br/>
          <div class="panel-title">出征队列({{ orders.length }})</div>
          <table>
            <tr><th>类型</th><th>目标</th><th>统帅</th><th>状态</th><th></th></tr>
            <tr v-for="o in orders" :key="'o' + o.id">
              <td>{{ o.type_name }}</td>
              <td>({{ o.target_x }},{{ o.target_y }})</td>
              <td>{{ o.officer || '无' }}</td>
              <td>{{ orderStatusText(o) }}</td>
              <td>
                <a href="javascript:;" @click="openOrder(o)">[详情]</a>
                <a v-if="o.order_type === 7 && (o.status === 0 || o.status === 1)" class="red"
                   href="javascript:;" @click="doRecall(o)">[召回]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!orders.length">(暂无出征部队)</div>
          <br/>
          <div class="panel-title">伤兵营</div>
          <table>
            <tr><th>兵种</th><th>数量</th><th>操作</th></tr>
            <tr v-for="w in woundedList(0)" :key="'w' + w.id">
              <td>{{ w.name }}</td><td>{{ w.count }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[恢复]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(0).length">(伤兵营无伤兵)</div>
          <div class="old-line" v-if="woundedList(0).length"><button @click="doRecoverAll(0)">[全部恢复]</button></div>
          <br/>
          <div class="panel-title">逃兵营</div>
          <table>
            <tr><th>兵种</th><th>数量</th><th>操作</th></tr>
            <tr v-for="w in woundedList(1)" :key="'dsw' + w.id">
              <td>{{ w.name }}</td><td>{{ w.count }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[召回]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(1).length">(逃兵营无逃兵)</div>
          <div class="old-line" v-if="woundedList(1).length"><button @click="doRecoverAll(1)">[全部召回]</button></div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 科技(techs) ============ -->
      <template v-else-if="cur === 'techs'">
        <div class="panel">
          <div class="panel-title">【科技中心】:{{ techsData.academy }}级</div>
          <div class="old-line" v-for="t in techsData.techs" :key="'te' + t.tech_id">
            <b>{{ t.name }}</b> {{ t.level }}/{{ t.max_level }}级
            <span class="gray">[需科研中心{{ t.academy_need }}级]</span><br/>
            {{ t.effect }}<br/>
            <span v-if="t.researching" class="orange">研究中 {{ remain(t.end_time) }}
              <a href="javascript:;" @click="doSpeedTech()">[加速]</a>
              <a href="javascript:;" @click="doCancelTech(t)">[取消]</a></span>
            <span v-else-if="t.level < t.max_level">
              <a href="javascript:;" @click="doResearch(t)">[研究{{ t.level + 1 }}级]</a>
              <span class="gray">耗: 粮{{ t.next_cost.food }} 钢{{ t.next_cost.steel }} 油{{ t.next_cost.oil }} 稀{{ t.next_cost.rare }} 金{{ t.next_cost.gold }} 需{{ Math.ceil(t.next_time / 60) }}分钟</span>
            </span>
            <span v-else class="gray">[已满级]</span>
            <br/>
          </div>
          <div class="old-line gray">同一时间只能研究一项科技；[取消] 会全额退还本次研究消耗。</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 地图(map) ============ -->
      <template v-else-if="cur === 'map'">
        <div class="panel">
          <div class="panel-title">地图</div>
          <div class="old-line">
            {{ city.name }}({{ city.x }},{{ city.y }})
            <a href="javascript:;" @click="go('cities')">切换城市</a>
          </div>
          <div class="old-line">
            输入坐标查找：
            <a href="javascript:;" @click="toggleStars">[收藏列表{{ mapStars.length ? '(' + mapStars.length + ')' : '' }}]</a>
          </div>
          <div class="old-line">
            横坐标：<input v-model="jumpX" type="number" placeholder="(1~500)" style="width:70px"/>
            纵坐标：<input v-model="jumpY" type="number" placeholder="(1~500)" style="width:70px"/>
            <button @click="doJump">[查找]</button>
          </div>
          <template v-if="showStars">
            <div class="panel-title">收藏列表</div>
            <div class="old-line" v-for="s in mapStars" :key="'st' + s.id">
              <a href="javascript:;" @click="jumpTo(s.x, s.y)">{{ s.name }}({{ s.x }},{{ s.y }})</a>
              <a href="javascript:;" @click="delStar(s)">[删除]</a>
            </div>
            <div class="old-line gray" v-if="!mapStars.length">(收藏列表为空, 在地图上选中目标后可收藏)</div>
          </template>
          <div class="old-line">
            <a href="javascript:;" @click="moveMap(-mapR, 0)">[向上]</a>
            <a href="javascript:;" @click="moveMap(0, mapR)">[向右]</a>
            <a href="javascript:;" @click="moveMap(mapR, 0)">[向下]</a>
            <a href="javascript:;" @click="moveMap(0, -mapR)">[向左]</a>
            <a href="javascript:;" @click="loadMap()">[回到本城]</a>
            <a href="javascript:;" @click="go('orders')">[出征队列]</a>
          </div>
          <div class="ezfy-map">
            <div v-for="(row, ri) in mapRows" :key="'mr' + ri" class="ezfy-map-row">
              <span v-for="cell in row" :key="cell.x + '_' + cell.y"
                    class="ezfy-cell" :class="cellClass(cell)"
                    @click="openCell(cell)">
                {{ cellText(cell) }}
              </span>
            </div>
          </div>
          <div class="old-line">当前坐标中心:({{ mapCx }} , {{ mapCy }})</div>
          <div class="old-line gray">
            城=城市 寇=寇城 墟=废墟 海=海洋 数字=等级<br/>
            陆地野地按地形分: 平原/草原/森林/盆地/丘陵/沼泽/山地
          </div>
          <template v-if="selCell">
            <div class="panel-title">目标({{ selCell.x }},{{ selCell.y }})</div>
            <div class="old-line" v-if="selDetail">
              <b>{{ selDetail.terrain_name }}</b> 等级{{ selDetail.level }}
              <span class="gray" v-if="selDetail.type === 2">(海野)</span>
              <span class="gray" v-else-if="selDetail.type === 3">(寇城)</span>
              <span class="gray" v-else>(陆地野地)</span><br/>
              <span v-for="tp in selDetail.troops" :key="'sp' + tp.troop_id">{{ tp.name }}约{{ tp.min }}-{{ tp.max }} </span><br/>
              掠夺资源约:{{ selDetail.res_min }}-{{ selDetail.res_max }}
            </div>
            <div class="old-line" v-else>
              {{ selCell.name }}
              <span v-if="selCell.city_level">{{ selCell.city_level }}级</span>
              <span v-if="selCell.owner">城主:{{ selCell.owner }}</span>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="addStar">[收藏该坐标]</a>
              <span v-if="selCell.area_type === 3">【归属: {{ selCell.owner || '无' }}】</span>
            </div>
            <div class="old-line" v-if="selCell.area_type === 3 && !selCell.mine">
              <a href="javascript:;" @click="declareWar">[对城主宣战(24小时后生效)]</a>
              <span v-if="warText" class="orange">{{ warText }}</span>
            </div>
            <div class="old-line" v-if="selCell.area_type !== 3 && selCell.name !== '寇城(废墟)'">
              出征:
              <select v-model="orderType">
                <option v-for="(n, i) in orderNames.slice(1)" :key="i" :value="i + 1"
                        v-if="orderAvail(i + 1)">{{ n }}</option>
              </select>
              <a href="javascript:;" @click="go('orderpre')">[选择部队出征]</a>
            </div>
            <div class="old-line" v-if="orderType === 5 && selCell.area_type !== 3">
              <span class="red">运输目标必须是城市, 请先选择自己/同盟的城市</span>
            </div>
          </template>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 出征确认(orderpre) ============ -->
      <template v-else-if="cur === 'orderpre'">
        <div class="panel" v-if="selCell">
          <div class="panel-title">{{ selCell.name }}<span v-if="selCell.level">({{ selCell.level }})</span> ({{ selCell.x }},{{ selCell.y }})</div>
          <div class="old-line">出征命令：<b>{{ orderNames[orderType] }}</b></div>
          <div class="old-line">
            集结令：
            <select disabled title="原版未实现该功能">
              <option>暂未开放</option>
            </select>
          </div>
          <div class="old-line">
            指挥军官：
            <select v-model="orderOfficer">
              <option value="0">未指定</option>
              <option v-for="o in onDutyOfficers" :key="'od' + o.id" :value="o.name">
                {{ o.name }}({{ o.level }}级) 军事{{ o.military }} 忠诚{{ o.loyalty }}
              </option>
            </select>
            <span v-if="orderType === 7" class="red">(派遣必须选择)</span>
            <span v-else-if="orderType === 6" class="gray">(增援后军官调任目标城市)</span>
            <br/>
            <span v-if="curOfficerBonus" class="green">军官战斗加成: 攻击+{{ curOfficerBonus }}%</span>
            <span v-if="!onDutyOfficers.length" class="gray">(暂无可用军官, 可前往军校招募)</span>
          </div>
          <div class="old-line">
            <div v-for="t in attackTroops" :key="'at' + t.troop_id">
              {{ t.name }}:<input type="number" min="0" :max="t.count"
                                  v-model="orderTroops[t.troop_id]"
                                  :placeholder="'0~' + t.count" style="width:90px"/>
            </div>
            <span v-if="!attackTroops.length" class="red">城内无可出征部队</span>
          </div>
          <div class="old-line">
            <div>携带资源：</div>
            <div>黄金：<input v-model="trGold" type="number" :placeholder="'0~' + city.gold" style="width:90px"/></div>
            <div>粮食：<input v-model="trFood" type="number" :placeholder="'0~' + city.food" style="width:90px"/></div>
            <div>钢铁：<input v-model="trSteel" type="number" :placeholder="'0~' + city.steel" style="width:90px"/></div>
            <div>石油：<input v-model="trOil" type="number" :placeholder="'0~' + city.oil" style="width:90px"/></div>
            <div>稀矿：<input v-model="trRare" type="number" :placeholder="'0~' + city.rare" style="width:90px"/></div>
            <span class="gray" v-if="orderType === 5">(运输命令必须携带资源或部队)</span>
          </div>
          <div class="old-line">
            宿营：
            <input v-model="waitH" type="number" min="0" max="24" style="width:50px"/> 时
            <input v-model="waitM" type="number" min="0" max="60" style="width:50px"/> 分
            (最多宿营24小时)
          </div>
          <div class="old-line">
            出征前请先计算消耗，否则可能无法出征成功
            <div>
              <button @click="doCalc">[计算]</button>
              油耗：<span class="orange">{{ orderCalc ? orderCalc.oil_used : '' }}</span>
              /负重：<span class="orange">{{ orderCalc ? orderCalc.carry : '' }}</span><br/>
              耗时：<span class="orange">{{ orderCalc ? orderCalc.need_time : '' }}</span>
              <span v-if="orderCalc" class="gray">(单程{{ orderCalc.travel_time }}<template v-if="orderCalc.wait_min">, 宿营{{ orderCalc.wait_min }}分</template>)</span>
            </div>
            <div v-if="orderCalc && !orderCalc.oil_enough" class="red">
              石油不足：需要{{ orderCalc.oil_used }}，当前只有{{ orderCalc.oil_have }}
            </div>
          </div>
          <hr/>
          <div class="old-line">
            <button @click="doOrder()">[出征]</button>
            <a href="javascript:;" @click="go('map')">[返回地图]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="go('map')">[前往地图]</a></div>
        </div>
      </template>

      <!-- ============ 命令详情 / 军队动态详情(orderview) ============ -->
      <template v-else-if="cur === 'orderview'">
        <div class="panel" v-if="curOrder">
          <div class="panel-title">军队动态详情</div>
          出发地:{{ curOrder.from_name }}<br/>
          目的地:{{ curOrder.target_name }}({{ curOrder.target_x }},{{ curOrder.target_y }})<br/>
          命令:{{ curOrder.type_name }}<br/>
          军官:{{ curOrder.officer || '无(未带军官)' }}<br/>
          统帅:{{ nick }}<br/>
          状态:{{ curOrder.status_name || orderStatusText(curOrder) }}<br/>
          出发时间:{{ curOrder.start_text }}<br/>
          到达时间:{{ curOrder.arrive_text }}<br/>
          <template v-if="curOrder.return_text">返航时间:{{ curOrder.return_text }}<br/></template>
          耗油:{{ curOrder.oil_used }}<br/>
          <hr/>
          【进攻方军队】<br/>
          <span v-for="(t, i) in curOrder.troops" :key="'ot' + i">{{ t.name }}:{{ t.count }}<br/></span>
          <span v-if="!curOrder.troops.length" class="gray">(未携带部队)</span>
          <template v-if="curOrder.resources && curOrder.resources.length">
            <br/>军队携带资源:<br/>
            <span v-for="(r, i) in curOrder.resources" :key="'or' + i">{{ r.name }}:{{ r.count }}<br/></span>
          </template>
          <hr/>
          <div class="old-line">
            <a href="javascript:;" @click="go('hq')">[指挥(司令部)]</a>
            <a v-if="curOrder.order_type === 7 && (curOrder.status === 0 || curOrder.status === 1)"
               class="red" href="javascript:;" @click="doRecall(curOrder)">[召回]</a>
            <a v-if="curOrder.order_type === 7 && curOrder.status === 1"
               href="javascript:;" @click="go('wilds')">[采集]</a>
            <a v-if="curOrder.report_id" href="javascript:;" @click="jumpReport(curOrder.report_id)">[查看战报]</a>
          </div>
          <a href="javascript:;" @click="go('orders')">[返回出征队列]</a>
        </div>
      </template>

      <!-- ============ 出征队列(orders) ============ -->
      <template v-else-if="cur === 'orders'">
        <div class="panel">
          <div class="panel-title">出征队列({{ orders.length }})</div>
          <table>
            <tr><th>命令</th><th>目标</th><th>状态</th><th>军官</th><th>抵达时间</th><th>操作</th></tr>
            <tr v-for="o in orders" :key="'odl' + o.id">
              <td>{{ o.type_name }}</td>
              <td>({{ o.target_x }},{{ o.target_y }})</td>
              <td>{{ orderStatusText(o) }}</td>
              <td>{{ o.officer || '无' }}</td>
              <td>{{ o.arrive_text }}</td>
              <td>
                <a href="javascript:;" @click="openOrder(o)">[查看]</a>
                <a v-if="o.order_type === 7 && (o.status === 0 || o.status === 1)" class="red"
                   href="javascript:;" @click="doRecall(o)">[召回]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!orders.length">(暂无出征部队)</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 城市状态(citystatus) ============ -->
      <template v-else-if="cur === 'citystatus'">
        <div class="panel">
          <div class="panel-title">城市状态</div>
          城市: {{ city.name }}({{ city.x }},{{ city.y }}) {{ continent }}<br/>
          市政厅: {{ city.city_level }}级<br/>
          人口: {{ city.pop }}/{{ city.pop_max }} (空闲{{ freePop }})<br/>
          民心/民怨: {{ city.feelings }}/{{ city.grievance }} 税率: {{ city.tax_rate }}%<br/>
          建筑: {{ buildings.length }}座 (军事+资源区 {{ areaCount }}/{{ areaCap }})<br/>
          军队: {{ totalTroops }} (城外{{ marching }}支队伍行进, {{ occupying }}支驻守)<br/>
          附属野地: {{ wildlands.length }}/{{ city.city_level }}<br/>
          训练队列: {{ queues.length }} | 未读军情: {{ unreadReports }}<br/>
          <span v-if="protectedUntil" class="green">[免战保护中]</span>
          <span v-if="boostUntil" class="orange">[增产中]</span>
          <br/>
          <a href="javascript:;" @click="go('cityhall')">[市政厅]</a>
          <a href="javascript:;" @click="go('wareset')">[仓库调配]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 附属野地(wilds) ============ -->
      <template v-else-if="cur === 'wilds'">
        <div class="panel">
          <div class="panel-title">占领野地({{ wildlands.length }}/{{ city.city_level }})</div>
          <table>
            <tr><th>坐标</th><th>地形</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="w in wildlands" :key="'wd' + w.id">
              <td>({{ w.x }},{{ w.y }})</td>
              <td>{{ w.terrain_name }}<span class="gray" v-if="w.wild_type === 2">(海野)</span></td>
              <td>{{ w.level }}</td>
              <td>{{ w.status === 0 ? '空闲' : '采集中' }}</td>
              <td>
                <a v-if="w.status === 0" href="javascript:;" @click="openWildGather(w)">[采集]</a>
                <span v-else class="gray">采集中</span>
                <a class="red" href="javascript:;" @click="doAbandon(w)">[放弃]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!wildlands.length">(尚未占领任何野地)</div>
          <br/>
          <template v-if="occupies.length">
            <div class="panel-title">被占领城市({{ occupies.length }})</div>
            <table>
              <tr><th>坐标</th><th>城市</th><th>原属</th><th>操作</th></tr>
              <tr v-for="o in occupies" :key="'oc' + o.id">
                <td>({{ o.x }},{{ o.y }})</td>
                <td>{{ o.city_name }}</td>
                <td>{{ o.def_user }}</td>
                <td>
                  <a href="javascript:;" @click="doOccupy('build', o)">[建立城市]</a>
                  <a class="red" href="javascript:;" @click="doOccupy('destroy', o)">[摧毁]</a>
                  <a href="javascript:;" @click="doOccupy('return', o)">[放弃归还]</a>
                </td>
              </tr>
            </table>
            <br/>
          </template>
          <a href="javascript:;" @click="go('map')">[前往地图占领]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 召集(convene) ============ -->
      <template v-else-if="cur === 'convene'">
        <div class="panel">
          <div class="panel-title">召集人口</div>
          当前人口: {{ city.pop }} / 民居容纳: {{ city.pop_max }}<br/>
          花费 10万黄金 召集 10万人口(不受民居容纳上限限制, 可突破上限)<br/>
          <div class="old-line">黄金: {{ city.gold }}</div>
          <button @click="doConvene">[召集]</button>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 仓库调配(wareset) ============ -->
      <template v-else-if="cur === 'wareset'">
        <div class="panel">
          <div class="panel-title">仓库({{ ware.level }}级)</div>
          <div class="old-line">
            保护总量:{{ ware.total }}
            <span class="gray">(保护额度内的资源不会被敌人抢夺走)</span>
          </div>
          <div class="old-line" v-if="ware.level < 1">
            <span class="red">尚未建造仓库, 无法保护资源</span>
            <a href="javascript:;" @click="go('builds')">[前往资源区建造]</a>
          </div>
          <div class="old-line" v-else-if="ware.level < 10">
            升级仓库可提升保护量, 下一级:{{ ware.next_total }}
            <a href="javascript:;" @click="go('builds')">[前往资源区]</a>
          </div>
          <div class="old-line" v-else>仓库已满级(保护量{{ ware.total }})</div>
          <div class="panel-title">调配保护比例(四项合计不超过100%)</div>
          <div class="old-line" v-for="r in ware.res" :key="'wr' + r.key">
            {{ r.name }}: 保护{{ r.protect }} / 现有{{ r.have }}
            <input v-model="wareRatio[r.key]" type="number" min="0" max="100" style="width:60px"/>%
          </div>
          <div class="old-line">
            当前合计:{{ wareSum }}%
            <span v-if="wareSum > 100" class="red">(超出100%, 无法保存)</span>
            <span v-else class="gray">(未分配部分不产生保护)</span>
          </div>
          <div class="old-line">
            <button @click="doWareSet" :disabled="wareSum > 100">[保存比例]</button>
          </div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
        </div>
      </template>

      <!-- ============ 安抚(placate) ============ -->
      <template v-else-if="cur === 'placate'">
        <div class="panel">
          <div class="panel-title">安抚民心</div>
          当前民心: {{ city.feelings }} / 民怨: {{ city.grievance }}<br/>
          安抚花费 民怨×100 黄金, 可清零民怨并回升民心。<br/>
          <div class="old-line">黄金: {{ city.gold }} | 预计花费: {{ city.grievance * 100 }}</div>
          <button @click="doPlacate">[安抚]</button>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 税率(taxset) ============ -->
      <template v-else-if="cur === 'taxset'">
        <div class="panel">
          <div class="panel-title">税率设置</div>
          当前税率: {{ city.tax_rate }}%<br/>
          <span class="gray">税率越高黄金收入越多, 但民心下降越快: ≤10%民心+2/时, ≤20%+1, ≤40%不变, ≤60%-1, 更高-2; 民怨≥50时产量减半。</span><br/>
          <div class="old-line">
            新税率: <input v-model="taxInput" type="number" min="0" max="100" style="width:70px"/>%
            <button @click="doTax">[设置]</button>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 改名(rename) ============ -->
      <template v-else-if="cur === 'rename'">
        <div class="panel">
          <div class="panel-title">城市改名</div>
          <div class="old-line">
            新城市名: <input v-model="renameInput" style="width:60%"/>
            <button @click="doRename">[确定]</button>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 市政厅(cityhall) ============ -->
      <template v-else-if="cur === 'cityhall'">
        <div class="panel">
          <div class="panel-title">市政厅({{ city.city_level }}级)</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('taxset')">[税率设置]</a>
            <a href="javascript:;" @click="go('sourceset')">[调整生产]</a>
            <a href="javascript:;" @click="go('citymove')">[地图搬迁]</a>
            <a href="javascript:;" @click="go('rename')">[城市更名]</a>
            <a href="javascript:;" @click="go('cities')">[城市列表/迁建]</a>
            <a href="javascript:;" @click="go('citystatus')">[城市状态]</a>
            <a href="javascript:;" @click="go('wilds')">[占领野地]</a>
            <a href="javascript:;" @click="go('wareset')">[仓库调配]</a>
          </div>
          <div class="panel-title">全部建筑总览</div>
          <table>
            <tr><th>建筑</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="b in buildings" :key="'hb' + b.id">
              <td>{{ b.name }}</td>
              <td>{{ b.level }}/{{ b.max_level }}</td>
              <td>{{ b.status === 0 ? '空闲' : '施工中 ' + remain(b.end_time) }}</td>
              <td>
                <a v-if="b.status === 0 && b.level > 0 && b.level < b.max_level" href="javascript:;" @click="doUpgrade(b)">[升级]</a>
                <span v-if="b.status !== 0"><a href="javascript:;" @click="doSpeedBuilding()">[加速]</a></span>
              </td>
            </tr>
          </table>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 城市迁移(citymove) 复刻 city/cityHallMove.html ============ -->
      <template v-else-if="cur === 'citymove'">
        <div class="panel">
          <div class="panel-title">市政厅 → 城市迁移</div>
          <div class="old-line">当前城市：{{ city.name }}({{ city.x }},{{ city.y }})</div>
          <div class="old-line">黄金：{{ moveInfo.gold }} / 每次迁城消耗 {{ moveInfo.gold_cost }}</div>
          <div class="old-line gray">
            使用迁城计划可改变一次城市坐标，只能迁移到选定区域随机坐标(未被占领的平原)
          </div>
          <div class="old-line gray">
            使用高级迁城计划可改变一次城市坐标，可以迁移到指定坐标(未被占领的平原)
          </div>
          <hr/>
          <div class="old-line">
            使用【迁城计划】
            <select v-model="moveArea">
              <option v-for="a in moveInfo.areas" :key="'ma' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="doMoveCity('low')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【高级迁城计划】<br/>
            请确认您输入的坐标是非被占领的平原，沿海平原无法直接迁移城市<br/>
            横坐标 x：<input v-model="moveX" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="moveY" type="number" style="width:80px"/>
            <button @click="doMoveCity('high')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【沿海迁城计划】<br/>
            请确认您输入的坐标是非被占领的沿海平原<br/>
            横坐标 x：<input v-model="moveX2" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="moveY2" type="number" style="width:80px"/>
            <button @click="doMoveCity('sea')">确认迁城</button>
          </div>
          <div class="old-line gray">迁城后附属野地不会随城迁移, 需要重新占领。</div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 调整生产(sourceset) 复刻 city/sourceSet.html ============ -->
      <template v-else-if="cur === 'sourceset'">
        <div class="panel">
          <div class="panel-title">调整生产（开工率）</div>
          <div class="old-line gray">开工率影响该资源的实际产量：实际产量 = 基础产量 × 开工率 / 100</div>
          <div class="old-line">
            粮食：
            <input v-model="rateFood" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateFood = 1">[最小]</a>
            <a href="javascript:;" @click="rateFood = 100">[最大]</a>
          </div>
          <div class="old-line">
            钢铁：
            <input v-model="rateSteel" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateSteel = 1">[最小]</a>
            <a href="javascript:;" @click="rateSteel = 100">[最大]</a>
          </div>
          <div class="old-line">
            石油：
            <input v-model="rateOil" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateOil = 1">[最小]</a>
            <a href="javascript:;" @click="rateOil = 100">[最大]</a>
          </div>
          <div class="old-line">
            稀矿：
            <input v-model="rateRare" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateRare = 1">[最小]</a>
            <a href="javascript:;" @click="rateRare = 100">[最大]</a>
          </div>
          <div class="old-line" style="color:#0000cc">请检查输入数字是否合理（范围 1~100）</div>
          <div class="old-line">
            <button @click="doSaveRate">确认调整</button>
          </div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军团(corps) ============ -->
      <template v-else-if="cur === 'corps'">
        <template v-if="myCorps">
          <div class="panel">
            <div class="panel-title">我的军团:{{ myCorps.name }}({{ myCorps.member_count }}人)</div>
            公告: {{ myCorps.notice || '无' }}<br/>
            <div class="old-line">
              <template v-if="isLeader">
                <a href="javascript:;" @click="openNoticeEdit()">[修改公告]</a>
                <a class="red" href="javascript:;" @click="doLeaveCorps()">[解散军团]</a>
              </template>
              <a v-else href="javascript:;" @click="doLeaveCorps()">[退出军团]</a>
            </div>
            <div class="panel-title">军团成员</div>
            <table>
              <tr><th>成员</th><th>职位</th><th>声望</th><th>军衔</th></tr>
              <tr v-for="m in corpsMembers" :key="'cm' + m.user_id">
                <td>{{ m.name }}</td>
                <td>{{ m.title }}</td>
                <td>{{ m.prestige }}</td>
                <td>{{ m.rank_name }}</td>
              </tr>
            </table>
            <div class="old-line" v-if="isLeader && corpsMembers.length > 1">
              踢人:
              <select v-model="kickUserId" style="width:30%">
                <option v-for="m in corpsMembers" v-if="!m.is_leader" :key="'kc' + m.user_id" :value="m.user_id">{{ m.name }}</option>
              </select>
              <button @click="doKick">[踢出]</button>
            </div>
          </div>
        </template>
        <div class="panel">
          <div class="panel-title">军团列表</div>
          <table>
            <tr><th>军团</th><th>人数</th><th>战力</th><th>操作</th></tr>
            <tr v-for="cp in corpsList" :key="'cp' + cp.id">
              <td>{{ cp.name }}</td>
              <td>{{ cp.member_count }}</td>
              <td>{{ cp.battle_score }}</td>
              <td><a v-if="!myCorps" href="javascript:;" @click="doJoinCorps(cp)">[加入]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!corpsList.length">(暂无军团)</div>
        </div>
        <div class="panel" v-if="myCorps">
          <div class="panel-title">军团聊天</div>
          <div class="old-line" v-for="m in corpsChats" :key="'cc' + m.id">
            [{{ m.user_name }}]:{{ m.content }}
          </div>
          <div class="old-line" v-if="!corpsChats.length">(暂无消息)</div>
          <div class="old-line">
            <input v-model="corpsMsg" style="width:64%"/>
            <button @click="doCorpsChat">发送</button>
            <button @click="loadCorps">刷新</button>
          </div>
        </div>
        <div class="panel" v-if="!myCorps">
          <div class="panel-title">创建军团</div>
          <div class="old-line">
            军团名: <input v-model="corpsName" style="width:50%"/>
            <button @click="doCreateCorps">[创建]</button>
          </div>
        </div>
        <a href="javascript:;" @click="go('home')">[返回首页]</a>
      </template>

      <!-- ============ 排行(rank) ============ -->
      <template v-else-if="cur === 'rank'">
        <div class="panel">
          <div class="panel-title">军衔声望榜</div>
          <table>
            <tr><th>名次</th><th>统帅</th><th>声望</th><th>军衔</th></tr>
            <tr v-for="r in rankData.prestige" :key="'rp' + r.rank">
              <td>{{ r.rank }}</td><td>{{ r.name }}</td><td>{{ r.prestige }}</td><td>{{ r.rank_name }}</td>
            </tr>
          </table>
          <div class="panel-title">兵力榜</div>
          <table>
            <tr><th>名次</th><th>统帅</th><th>城市</th><th>兵力</th></tr>
            <tr v-for="r in rankData.troops" :key="'rt' + r.rank">
              <td>{{ r.rank }}</td><td>{{ r.role_name }}</td><td>{{ r.city_name }}</td><td>{{ r.count }}</td>
            </tr>
          </table>
          <div class="panel-title">军团榜</div>
          <table>
            <tr><th>名次</th><th>军团</th><th>人数</th><th>战力</th></tr>
            <tr v-for="r in rankData.corps" :key="'rc' + r.rank">
              <td>{{ r.rank }}</td><td>{{ r.name }}</td><td>{{ r.member_count }}</td><td>{{ r.battle_score }}</td>
            </tr>
          </table>
          <div class="panel-title">军衔晋升表</div>
          <div class="old-line" v-for="(r, i) in rankData.ranks" :key="'rk' + i">
            {{ r.name }}({{ r.post }}) 需声望{{ r.need }}
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 背包(bag) ============ -->
      <template v-else-if="cur === 'bag'">
        <div class="panel">
          <div class="panel-title">背包 <button @click="loadBag">刷新</button></div>
          <div class="old-line" v-for="it in bagItems" :key="'bi' + it.cfg_id">
            <b>{{ it.name }}</b>×{{ it.count }}
            <a href="javascript:;" @click="openUse(it)">[使用]</a><br/>
            <span class="gray">{{ it.description }}</span>

            <!-- 使用面板: 数量 + (军官类道具)目标军官/技能 -->
            <div v-if="useItem && useItem.cfg_id === it.cfg_id" class="use-box">
              数量:
              <input v-model="useCount" type="number" min="1" :max="it.count" style="width:60px"/>
              <span class="gray">/{{ it.count }}</span>
              <a href="javascript:;" @click="useCount = it.count">[全部]</a><br/>

              <template v-if="needOfficer(it)">
                军官:
                <select v-model="useOfficerId">
                  <option :value="0">请选择军官</option>
                  <option v-for="o in bagOfficers" :key="'bo' + o.id" :value="o.id">
                    {{ o.name }} Lv{{ o.level }} {{ o.status_name }}
                  </option>
                </select><br/>
              </template>
              <template v-if="it.item_type === 11">
                技能:
                <select v-model="useSkillId">
                  <option :value="0">请选择技能</option>
                  <option v-for="s in bagSkills" :key="'bs' + s.id" :value="s.id">
                    {{ s.name }}({{ s.effect }})
                  </option>
                </select><br/>
              </template>
              <button @click="doUse(it)">[确认使用]</button>
              <a href="javascript:;" @click="useItem = null">[取消]</a>
            </div>
          </div>
          <div class="old-line" v-if="!bagItems.length">(背包空空如也)</div>
          <a href="javascript:;" @click="go('mall')">[前往商城]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 商城(mall) ============ -->
      <template v-else-if="cur === 'mall'">
        <div class="panel">
          <div class="panel-title">商城(黄金{{ city.gold }})</div>
          <div class="old-line" v-for="it in mallItems" :key="'mi' + it.id">
            <b>{{ it.name }}</b> {{ it.price_gold }}黄金
            <a href="javascript:;" @click="openBuy(it)">[购买]</a><br/>
            <span class="gray">{{ it.description }}</span>
            <div v-if="buyItem && buyItem.id === it.id" class="use-box">
              数量:
              <input v-model="buyCount" type="number" min="1" max="99" style="width:60px"/>
              <span class="gray">合计 {{ it.price_gold * (parseInt(buyCount) || 0) }} 黄金</span>
              <button @click="doBuy(it)">[确认购买]</button>
              <a href="javascript:;" @click="buyItem = null">[取消]</a>
            </div>
          </div>
          <a href="javascript:;" @click="go('bag')">[背包]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 交易行(exchange) ============ -->
      <template v-else-if="cur === 'exchange'">
        <div class="panel">
          <div class="panel-title">资源交易行(黄金{{ exchangeGold }})</div>
          <div class="old-line gray">购买他人挂单的资源; 也可挂单出售资源换取黄金。</div>
          <table>
            <tr><th>卖家</th><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in exchangeOrders" :key="'eo' + e.id">
              <td>{{ e.seller_name }}</td>
              <td>{{ e.type_name }}</td>
              <td>{{ e.count }}</td>
              <td>{{ e.total_price }}</td>
              <td><a href="javascript:;" @click="doExchangeBuy(e)">[购买]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!exchangeOrders.length">(暂无在售订单)</div>
          <br/>
          <div class="panel-title">我的挂单</div>
          <div class="old-line" v-for="e in exchangeMine" :key="'em' + e.id">
            {{ e.type_name }}×{{ e.count }} 售{{ e.total_price }}黄金
            <a href="javascript:;" @click="doExchangeCancel(e)">[下架]</a>
          </div>
          <div class="old-line" v-if="!exchangeMine.length">(无在售挂单)</div>
          <br/>
          <div class="panel-title">挂单出售</div>
          <div class="old-line">
            资源:
            <select v-model="sellType" style="width:70px">
              <option value="1">粮食</option><option value="2">钢铁</option>
              <option value="3">石油</option><option value="4">稀矿</option>
            </select><br/>
            数量: <input v-model="sellCount" type="number" style="width:90px"/><br/>
            总价(黄金): <input v-model="sellPrice" type="number" style="width:90px"/><br/>
            <button @click="doExchangeSell">[挂单出售]</button>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 活动(activity) ============ -->
      <template v-else-if="cur === 'activity'">
        <div class="panel">
          <div class="panel-title">活动</div>
          <template v-if="activities.length">
            <div class="old-line" v-for="a in activities" :key="'ac' + a.id">
              <b>{{ a.name }}</b>
              <span :class="a.running ? 'green' : 'gray'">[{{ a.running ? '进行中' : '未开启' }}]</span>
              <span class="orange">{{ a.effect }}</span><br/>
              {{ a.des }}<br/>
              <span v-if="a.running" class="gray">剩余: {{ fmtLeft(a.left_sec) }}</span>
              <span v-else-if="a.start_time && a.end_time" class="gray">
                时间: {{ fmtTime(a.start_time) }} ~ {{ fmtTime(a.end_time) }}
              </span>
            </div>
          </template>
          <div class="old-line gray" v-else>(暂无节日活动, 敬请期待)</div>
          <hr/>
          <div class="old-line gray">活动类型: 资源增产 / 造兵打折 / 建造加速 / 研究加速 / 声望加成</div>
          <hr/>
          <div class="old-line">[开服活动] 新手礼包、每周福利、市政厅等级礼包持续发放中, 前往<a href="javascript:;" @click="go('welfare')">[福利]</a>领取。</div>
          <div class="old-line">[征战天下] 征服野地/寇城可获得军功声望, 声望晋升军衔!</div>
          <div class="old-line">[物资兑换] 交易所开放资源交易, 低买高卖赚黄金。</div>
          <a href="javascript:;" @click="go('map')">[前往地图]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 福利(welfare) ============ -->
      <template v-else-if="cur === 'welfare'">
        <div class="panel">
          <div class="panel-title">每日签到</div>
          <div class="old-line">
            <span v-if="welfare.signed_today">今日已签到(连续{{ welfare.sign_count }}天)</span>
            <a v-else href="javascript:;" @click="doSign">[签到领奖]</a>
            | 声望:{{ welfare.prestige }}({{ welfare.rank_name }})
          </div>
          <div class="old-line" v-for="r in welfare.rewards" :key="'sr' + r.day">
            第{{ r.day }}天:{{ r.reward }}
          </div>
          <div class="panel-title">礼包</div>
          <div class="old-line">
            <a href="javascript:;" @click="doGift('newbie')">{{ welfare.gifts.newbie ? '[新手礼包已领]' : '[新手礼包]' }}</a>
            <a href="javascript:;" @click="doGift('weekly')">{{ welfare.gifts.weekly ? '[每周福利已领]' : '[每周福利]' }}</a><br/>
            <a href="javascript:;" @click="doGift('level10')">{{ welfare.gifts.level10 ? '[市政厅10级礼包已领]' : '[市政厅10级礼包]' }}</a>
            <a href="javascript:;" @click="doGift('level20')">{{ welfare.gifts.level20 ? '[市政厅20级礼包已领]' : '[市政厅20级礼包]' }}</a><br/>
            <a href="javascript:;" @click="doGift('level30')">{{ welfare.gifts.level30 ? '[市政厅30级礼包已领]' : '[市政厅30级礼包]' }}</a>
            <a href="javascript:;" @click="doGift('level40')">{{ welfare.gifts.level40 ? '[市政厅40级礼包已领]' : '[市政厅40级礼包]' }}</a>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 公告(notices) ============ -->
      <template v-else-if="cur === 'notices'">
        <div class="panel">
          <div class="panel-title">公告</div>
          <div class="old-line" v-for="n in notices" :key="'nn' + n.id">
            <span v-if="n.is_top" class="red">[置顶]</span>
            <a href="javascript:;" @click="openNotice(n)">{{ n.title }}</a>
          </div>
          <template v-if="curNotice">
            <div class="panel-title">{{ curNotice.title }}</div>
            <div class="old-line">{{ curNotice.content }}</div>
          </template>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 联络中心(liaison) ============ -->
      <template v-else-if="cur === 'liaison'">
        <div class="panel">
          <div class="panel-title">联络中心({{ liaison.level }}级)</div>
          <div class="old-line gray">
            联络中心是盟友间互相联络的建筑。<br/>
            1级可加入联盟, 2级可创建联盟(消耗{{ liaison.create_cost }}黄金);<br/>
            每级多 1 支盟友驻军、多 {{ liaison.member_per_level }} 人联盟人数上限。
          </div>
          <template v-if="liaison.level < 1">
            <div class="old-line red">尚未建造联络中心, 无法加入或创建联盟</div>
            <div class="old-line"><a href="javascript:;" @click="go('buildm')">[前往军事区建造]</a></div>
          </template>

          <div class="panel-title">我的联盟</div>
          <template v-if="liaison.my_corps">
            <div class="old-line">
              <b>{{ liaison.my_corps.name }}</b>
              (成员{{ liaison.member_count }}/{{ liaison.member_cap }})<br/>
              <span class="gray">{{ liaison.my_corps.notice || '暂无公告' }}</span>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="go('corps')">[进入军团]</a>
            </div>
          </template>
          <template v-else>
            <div class="old-line gray">尚未加入联盟</div>
            <div class="old-line">
              <a href="javascript:;" @click="go('corps')">[加入联盟]</a>
              <a v-if="liaison.can_create" href="javascript:;" @click="go('corps')">[创建联盟]</a>
              <span v-else class="red">(需2级联络中心才能创建联盟)</span>
            </div>
          </template>

          <div class="panel-title">盟军驻军({{ liaison.garrison_used }}/{{ liaison.garrison_cap }})</div>
          <table>
            <tr><th>来自城市</th><th>军官</th><th>驻军</th></tr>
            <tr v-for="g in liaison.garrisons" :key="'lg' + g.id">
              <td>{{ g.from_city }}</td>
              <td>{{ g.officer || '无' }}</td>
              <td>
                <span v-for="(t, i) in g.troops" :key="'lgt' + i">{{ t.name }}×{{ t.count }} </span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!liaison.garrisons.length">(暂无盟军驻军)</div>
          <div class="old-line gray">
            升级联络中心可接收更多盟友驻军; 联盟成员可用「增援」把部队派到你的城市协防。
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 统帅(info) ============ -->
      <template v-else-if="cur === 'info'">
        <div class="panel">
          <div class="panel-title">统帅信息</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('builds')">资源</a> ·
            <a href="javascript:;" @click="go('acade')">军官</a> ·
            <a href="javascript:;" @click="go('troops')">军队</a> ·
            <a href="javascript:;" @click="go('techs')">科技</a> ·
            <a href="javascript:;" @click="go('defence')">城防</a> ·
            个人
          </div>
          账号：{{ userBrief.account }}<br/>
          昵称：{{ profile.nickname }}<br/>
          阵营：{{ profile.camp === 2 ? '轴心国' : '同盟国' }}<br/>
          声望：{{ profile.prestige }}<br/>
          军衔：{{ rankName }}({{ rankPost }})<br/>
          城市数：{{ cities.length }}<br/>
          人口数：{{ city.pop }}<br/>
          军官数：{{ officerCount }}<br/>
          <br/>
          ID：{{ profile.user_id }}<br/>
          等级：{{ userBrief.level }}<br/>
          特权：普通用户<br/>
          VIP等级：普通用户<br/>
          经验：{{ userBrief.exp }}<br/>
          状态：正常<br/>
          <br/>
          总兵力：{{ totalTroops }}<br/>
          城外行进：{{ marching }}支 | 驻守采集：{{ occupying }}支<br/>
          占领野地：{{ wildlands.length }}块<br/>
          <a href="javascript:;" @click="go('friends')">[申请好友]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军官/学院(acade) ============ -->
      <template v-else-if="cur === 'acade'">
        <div class="panel-title">
          参谋部({{ officerData.staff_level }}级)
          <a href="javascript:;" @click="switchAcade('search')">去招募</a> |
          <a href="javascript:;" @click="switchAcade('captive')">战俘营</a>
        </div>
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: acadeTab === 'officer' }" @click="switchAcade('officer')">军官</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'scheme' }" @click="switchAcade('scheme')">计谋</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'search' }" @click="switchAcade('search')">招募</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'mayor' }" @click="switchAcade('mayor')">任命市长</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'equip' }" @click="switchAcade('equip')">装备</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'skill' }" @click="switchAcade('skill')">技能</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'generals' }" @click="switchAcade('generals')">名将图鉴</a>
        </div>

        <!-- 军官列表 -->
        <div class="panel" v-if="acadeTab === 'officer'">
          <div class="old-line">
            军校{{ officerData.academy_level }}级, 参谋部{{ officerData.staff_level }}级
            (容纳{{ officerData.capacity }}名军官), 当前{{ officerData.used }}名
          </div>
          <div class="old-line">
            黄金:{{ officerData.gold }}
            <a href="javascript:;" @click="switchAcade('search')">[招募名将]</a>
          </div>
          <hr/>
          <div class="old-line">我的军官({{ myOfficers.length }}):</div>
          <table>
            <tr><th>名称</th><th>星</th><th>等级</th><th>经验</th><th>后勤</th><th>军事</th><th>学识</th><th>忠诚</th><th>攻</th><th>防</th><th>职位</th><th>状态</th><th>操作</th></tr>
            <tr v-for="o in myOfficers" :key="'of' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.star }}</td>
              <td>{{ o.level }}</td>
              <td>{{ o.exp }}</td>
              <td>{{ o.logistics }}</td>
              <td>{{ o.military }}</td>
              <td>{{ o.learning }}</td>
              <td>{{ o.loyalty }}</td>
              <td>{{ o.attack }}</td>
              <td>{{ o.defence }}</td>
              <td>{{ o.position_name }}</td>
              <td>
                <span :class="{ orange: o.status === 1 }">{{ o.status_name }}</span>
              </td>
              <td>
                <a href="javascript:;" @click="openOfficer(o.id)">[详情]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!myOfficers.length">(暂无军官, 先去招募吧)</div>
          <div class="old-line">
            前去<a href="javascript:;" @click="switchAcade('captive')">[战俘营]</a>
            <span class="gray" v-if="captiveOfficers.length">({{ captiveOfficers.length }}名俘虏待收编)</span>
          </div>
        </div>

        <!-- 招募 -->
        <div class="panel" v-else-if="acadeTab === 'search'">
          <div class="old-line">
            军校({{ recruitData.academy_level }}级)：
            <span v-if="recruitData.refresh_left !== undefined">
              今日刷新:{{ recruitData.refresh_left }}/{{ recruitData.refresh_limit }}次
            </span>
            <a href="javascript:;" @click="doRefreshRecruit">[刷新]</a>
          </div>
          <div class="old-line gray">
            军校等级决定每日候选数量, 参谋部{{ recruitData.staff_level }}级(已用{{ recruitData.used }}/{{ recruitData.capacity }}),
            招募费用 = 名将等级 × 500 黄金
          </div>
          <div class="old-line" v-if="!recruitData.academy_level">尚未建造军校, 无法招募军官</div>
          <table v-else>
            <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
            <tr v-for="g in recruitData.candidates" :key="'rc' + g.id">
              <td>{{ g.name }}</td>
              <td>{{ g.level }}级</td>
              <td>{{ g.star }}星</td>
              <td>{{ g.logistics }}/{{ g.military }}/{{ g.learning }}</td>
              <td>{{ g.cost }}</td>
              <td><a href="javascript:;" @click="doRecruit(g)">[招募]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="recruitData.academy_level && !recruitData.candidates.length">(今日候选已全部招募或刷新)</div>
          <div class="old-line">前去<a href="javascript:;" @click="switchAcade('officer')">[军官]</a></div>
        </div>

        <!-- 任命市长 -->
        <div class="panel" v-else-if="acadeTab === 'mayor'">
          <div class="old-line gray">参谋部: 市长(产量+10%+后勤属性)、城守(守城防御+10%)</div>
          <table>
            <tr><th>名称</th><th>等级</th><th>忠诚</th><th>当前职位</th><th>操作</th></tr>
            <tr v-for="o in myOfficers" :key="'my' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.level }}</td>
              <td>{{ o.loyalty }}</td>
              <td>{{ o.position_name }}</td>
              <td>
                <a v-if="o.position !== 1 && o.status === 0" href="javascript:;" @click="doPosition(o, 1)">[任命市长]</a>
                <a v-if="o.position !== 2 && o.status === 0" href="javascript:;" @click="doPosition(o, 2)">[任命城守]</a>
                <a v-if="o.position !== 0" href="javascript:;" @click="doPosition(o, 0)">[卸任]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerData.officers.length">(暂无军官)</div>
        </div>

        <!-- 装备 -->
        <div class="panel" v-else-if="acadeTab === 'equip'">
          <div class="old-line">我的装备({{ equipData.bag.length }})</div>
          <table>
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>要求等级</th><th>状态</th></tr>
            <tr v-for="e in equipData.bag" :key="'eq' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
              <td>
                <span v-if="e.worn" class="gray">{{ e.worn_by }}已穿戴</span>
                <a v-else href="javascript:;" @click="switchAcade('officer')">[去穿戴]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!equipData.bag.length">(背包暂无装备, 战胜野地/寇城有概率掉落)</div>
          <hr/>
          <div class="old-line">装备图鉴({{ equipData.all.length }})</div>
          <table>
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>需求等级</th></tr>
            <tr v-for="e in equipData.all" :key="'ea' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
            </tr>
          </table>
        </div>

        <!-- 技能 -->
        <div class="panel" v-else-if="acadeTab === 'skill'">
          <div class="old-line">军官技能(每名武将最多3个, 学习1万金/个):</div>
          <table>
            <tr><th>技能</th><th>效果</th></tr>
            <tr v-for="s in skillData.skills" :key="'sk' + s.id">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
            </tr>
          </table>
          <hr/>
          <div class="old-line">我的军官:</div>
          <table>
            <tr><th>名称</th><th>已学技能</th><th>操作</th></tr>
            <tr v-for="o in skillData.officers" :key="'sko' + o.id">
              <td>{{ o.name }}</td>
              <td>
                <span v-if="o.skills.length">{{ o.skills.join('、') }}</span>
                <span v-else class="gray">无({{ o.skill_count }}/3)</span>
              </td>
              <td><a href="javascript:;" @click="openOfficer(o.id)">[学习/遗忘]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!skillData.officers.length">(暂无军官)</div>
        </div>

        <!-- 计谋(复刻原版 acade/scheme.html: 12 条计谋说明, 各需信号弹) -->
        <div class="panel" v-else-if="acadeTab === 'scheme'">
          <div class="old-line">说明：计谋需要进入相应界面才可以使用</div>
          <div class="old-line" v-for="(s, i) in schemes" :key="'sc' + i">
            {{ i + 1 }}.{{ s.name }}：<br/>
            {{ s.des }}<br/>
            需要信号弹：{{ s.bullet }}
            <br/>--------------------
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('bag')">[背包(信号弹)]</a>
            <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
        </div>

        <!-- 战俘营(复刻原版 acade/conquer.html) -->
        <div class="panel" v-else-if="acadeTab === 'captive'">
          <div class="old-line">
            参谋部({{ officerData.staff_level }}级)
            <a href="javascript:;" @click="switchAcade('search')">去招募</a> |
            战俘营
          </div>
          <table>
            <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
            <tr v-for="o in captiveOfficers" :key="'cp' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.level }}级</td>
              <td>{{ o.star }}星</td>
              <td>{{ o.logistics }}/{{ o.military }}/{{ o.learning }}</td>
              <td>免费</td>
              <td>
                <a href="javascript:;" @click="doCaptive(o, 'recruit')">[雇佣]</a>
                <a href="javascript:;" @click="doCaptive(o, 'free')">[释放]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!captiveOfficers.length">(战俘营暂无俘虏, 战胜寇城/高级野地有几率俘获守将)</div>
          <div class="old-line">前去<a href="javascript:;" @click="switchAcade('officer')">[军官]</a></div>
        </div>

        <!-- 名将图鉴 -->
        <div class="panel" v-else-if="acadeTab === 'generals'">
          <div class="old-line">名将图鉴(共{{ generalData.generals.length }}名, 按等级排序)</div>
          <table>
            <tr><th>名称</th><th>等级</th><th>星级</th><th>军/后/学</th><th>获取渠道</th><th>状态</th></tr>
            <tr v-for="g in generalData.generals" :key="'gg' + g.id">
              <td>{{ g.name }}</td>
              <td>{{ g.level }}</td>
              <td>{{ g.star }}</td>
              <td>{{ g.military }}/{{ g.logistics }}/{{ g.learning }}</td>
              <td>{{ g.source }}</td>
              <td>
                <span v-if="g.owned" class="green">已拥有</span>
                <span v-else class="gray">未拥有</span>
              </td>
            </tr>
          </table>
        </div>
      </template>

      <!-- ============ 军官详情(officerdetail) ============ -->
      <template v-else-if="cur === 'officerdetail'">
        <div class="panel" v-if="officerDetail.officer">
          <div class="panel-title">{{ officerDetail.officer.name }}</div>
          星级:{{ officerDetail.officer.star }}
          等级:{{ officerDetail.officer.level }}
          经验:{{ officerDetail.officer.exp }}/{{ officerDetail.officer.exp_need }}<br/>
          军事:{{ officerDetail.officer.military }}
          后勤:{{ officerDetail.officer.logistics }}
          学习:{{ officerDetail.officer.learning }}<br/>
          忠诚:{{ officerDetail.officer.loyalty }}
          职位:{{ officerDetail.officer.position_name }}
          状态:{{ officerDetail.officer.status_name }}<br/>
          <div class="old-line">
            <button @click="doGrant">[赏赐+10忠诚(1万金)]</button>
            <button v-if="officerDetail.officer.status !== 1 && officerDetail.officer.position === 0"
                    @click="doExile">[流放]</button>
            <span v-if="officerDetail.officer.status === 1" class="gray">(出征中, 归来后才能流放)</span>
            <span v-else-if="officerDetail.officer.position !== 0" class="gray">(市长/城守, 卸任后才能流放)</span>
          </div>
          <hr/>
          已学技能({{ officerDetail.skills.length }}/3):
          <table>
            <tr><th>技能</th><th>效果</th><th>操作</th></tr>
            <tr v-for="s in officerDetail.skills" :key="'ds' + s.name">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doForget(s.name)">[遗忘]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.skills.length">(未学任何技能)</div>
          <br/>
          可学技能(1万金/个):
          <table>
            <tr><th>名称</th><th>效果</th><th>操作</th></tr>
            <tr v-for="s in officerDetail.all_skills" :key="'ls' + s.id">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doLearn(s)">[学习]</a></td>
            </tr>
          </table>
          <hr/>
          已穿戴装备:
          <table>
            <tr><th>名称</th><th>类型</th><th>军事</th><th>后勤</th><th>学习</th><th>操作</th></tr>
            <tr v-for="e in officerDetail.equipped" :key="'de' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.military }}</td>
              <td>{{ e.logistics }}</td>
              <td>{{ e.learning }}</td>
              <td><a href="javascript:;" @click="doUnequip(e.id)">[卸下]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.equipped.length">(未穿戴装备)</div>
          <hr/>
          装备背包:
          <table>
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>要求等级</th><th>操作</th></tr>
            <tr v-for="e in officerDetail.bag" :key="'db' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
              <td>
                <a v-if="!e.worn" href="javascript:;" @click="doEquip(e)">[穿戴]</a>
                <span v-else class="gray">已穿戴</span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.bag.length">(背包暂无装备)</div>
          <div class="old-line"><a href="javascript:;" @click="go('acade')">[返回军官]</a></div>
        </div>
      </template>

      <!-- 底部返回(非首页) -->
      <template v-if="cur !== 'home'">
        <br/>
        <div class="bottom-nav"><a href="javascript:;" @click="go('home')">[返回游戏]</a></div>
        <div class="footer">WAP报时:{{ nowText }}</div>
      </template>
    </div>
  </div>
</template>

<script>
import api from '../api'

const RES_NAMES = { gold: '黄金', food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿' }

// 资源说明(黄金一段复刻原版 resourceA.html, 其余按同样语气补全)
const RES_DES = {
  gold: '黄金是二战世界里重要的交易货币，可以用来购买玩家出售的各种资源，也用于支付雇佣军官的薪资和招募费用，通过战争、税收、任务可获得。',
  food: '粮食是维持军队的根本，人口增长与部队训练都离不开它，军队每小时还会消耗粮食，一旦断粮部队将大量逃散，通过农田产出、掠夺、任务可获得。',
  steel: '钢铁是制造武器装备的基础材料，建造建筑、训练陆军部队、研究军事科技都需要消耗钢铁，通过炼钢厂产出、掠夺、任务可获得。',
  oil: '石油驱动着一切机械化部队，出征行军需要消耗石油，训练装甲与航空部队同样需要石油，通过石油基地产出、掠夺、任务可获得。',
  rare: '稀矿是尖端军事工业的原料，用于生产重型装备与高级兵种，产量稀少因此格外珍贵，通过稀矿厂产出、掠夺、任务可获得。'
}

export default {
  name: 'Ezfy',
  data () {
    return {
      cur: 'home',
      resNames: RES_NAMES,
      resDes: RES_DES,
      profile: { prestige: 0, camp: 1, nickname: '' },
      userBrief: { account: '', level: 0, exp: 0 },
      officerCount: 0,
      activities: [],
      rankName: '列兵',
      rankPost: '士兵',
      city: {},
      cities: [],
      continent: '',
      protectedUntil: false,
      boostUntil: false,
      buildings: [],
      buildingPool: [],
      troopsData: { troops: [], queues: [], wounded: [], cfgs: [], pop: 0, pop_used: 0, wall_level: 0, train_discount: 0 },
      techsData: { techs: [], academy: 0 },
      wildlands: [],
      occupies: [],
      queues: [],
      marching: 0,
      occupying: 0,
      unreadReports: 0,
      reports: [],
      notices: [],
      curNotice: null,
      worldChats: [],
      chatPlayers: 0,
      chatMsg: '',
      chatChannel: 1,
      chatHasCorps: false,
      chatCorpsName: '',
      chatCanSend: true,
      chatCooldown: 0,
      chatNotices: [],
      mails: [],
      friends: [],
      friendKeyword: '',
      friendSearchList: [],
      friendSearchDone: false,
      taskGroups: [],
      // 计谋(复刻原版 acade/scheme.html, 共 12 条, 消耗信号弹)
      schemes: [
        { name: '恫疑虚喝', des: '恫疑虚喝', bullet: 4 },
        { name: '隐真示假', des: '隐真示假', bullet: 4 },
        { name: '十面埋伏', des: '十面埋伏', bullet: 7 },
        { name: '欲擒故纵', des: '欲擒故纵', bullet: 7 },
        { name: '偷梁换柱', des: '偷梁换柱', bullet: 6 },
        { name: '反客为主', des: '反客为主', bullet: 6 },
        { name: '先发制人', des: '使我军与敌军城市直接进入可战争状态，可战争时间为军官学识×1分钟，最多持续6小时，中计城市6小时内不再中计', bullet: 12 },
        { name: '未雨绸缪', des: '未雨绸缪', bullet: 12 },
        { name: '虚实相乱', des: '虚实相乱', bullet: 4 },
        { name: '调虎离山', des: '调虎离山', bullet: 4 },
        { name: '各个击破', des: '各个击破', bullet: 8 },
        { name: '舍车保帅', des: '舍车保帅', bullet: 8 }
      ],
      welfare: { rewards: [], gifts: {} },
      rankData: { prestige: [], troops: [], corps: [], ranks: [] },
      orders: [],
      curReport: null,
      curOrder: null,
      showDetail: true,
      corpsList: [],
      myCorps: null,
      corpsMembers: [],
      corpsChats: [],
      corpsName: '',
      corpsMsg: '',
      kickUserId: 0,
      mallItems: [],
      bagItems: [],
      bagOfficers: [],
      bagSkills: [],
      useItem: null,
      useCount: 1,
      useOfficerId: 0,
      useSkillId: 0,
      buyItem: null,
      buyCount: 1,
      exchangeOrders: [],
      exchangeMine: [],
      exchangeGold: 0,
      acadeTab: 'officer',
      officerData: { officers: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0 },
      recruitData: { candidates: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0, refresh_left: 0, refresh_limit: 5 },
      skillData: { skills: [], officers: [], gold: 0 },
      equipData: { bag: [], all: [] },
      generalData: { generals: [] },
      officerDetail: { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 },
      sellType: '1',
      sellCount: 0,
      sellPrice: 0,
      trainSel: null,
      trainCount: 10,
      trainSplit: false,
      taxInput: 20,
      renameInput: '',
      newCityX: '',
      newCityY: '',
      targetCfg: {},
      resType: 'gold',
      resDetail: null,
      mapCells: [],
      mapCx: 0,
      mapCy: 0,
      mapR: 7,
      selCell: null,
      selDetail: null,
      warText: '',
      orderType: 2,
      orderTroops: {},
      onDutyOfficers: [],
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
      waitM: 0,
      orderCalc: null,
      jumpX: '',
      jumpY: '',
      mapStars: [],
      showStars: false,
      // 城市迁移
      moveInfo: { areas: [], gold_cost: 200000, gold: 0 },
      moveArea: 1,
      moveX: '',
      moveY: '',
      moveX2: '',
      moveY2: '',
      // 调整生产(开工率)
      rateFood: 100,
      rateSteel: 100,
      rateOil: 100,
      rateRare: 100,
      orderNames: ['', '侦查', '掠夺', '征服', '采集', '运输', '增援', '派遣'],
      timer: null,
      nowTimer: null,
      nowText: ''
    }
  },
  computed: {
    nick () {
      return this.$store.state.user ? this.$store.state.user.nickname : ''
    },
    freePop () {
      const used = this.troopsData.pop_used || 0
      const v = this.city.pop - used
      return v > 0 ? v : 0
    },
    queueNames () {
      return this.queues
    },
    zoneBuildings () {
      // 复刻原版 BuildingController：军事区 = type 2/3/4，资源区 = type 1
      const isM = this.cur === 'buildm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      const built = this.buildings.filter(b => inZone(b.type))
      const pool = (this.buildingPool || []).filter(p => inZone(p.type))
      return built.concat(pool)
    },
    buildQueueCount () {
      return this.buildings.filter(b => b.status !== 0).length
    },
    // 正式军官(不含俘虏)
    myOfficers () {
      return (this.officerData.officers || []).filter(o => o.is_captive !== 1)
    },
    // 战俘营: 未出征的俘虏
    captiveOfficers () {
      return (this.officerData.officers || []).filter(o => o.is_captive === 1 && o.status !== 1)
    },
    trainCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type !== 4)
    },
    curOfficerBonus () {
      for (const o of this.onDutyOfficers) {
        if (o.name === this.orderOfficer) return o.battle_bonus
      }
      return 0
    },
    wareSum () {
      const r = this.wareRatio
      return (parseInt(r.food) || 0) + (parseInt(r.steel) || 0) + (parseInt(r.oil) || 0) + (parseInt(r.rare) || 0)
    },
    defenceCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type === 4)
    },
    defenceTroops () {
      return (this.troopsData.troops || []).filter(t => t.type === 4)
    },
    attackTroops () {
      return (this.troopsData.troops || []).filter(t => t.type !== 4)
    },
    totalTroops () {
      let sum = 0
      for (const t of this.troopsData.troops) sum += t.count
      return sum
    },
    areaCount () {
      return this.buildings.filter(b => b.type === 1 || b.type === 2 || b.type === 3).length
    },
    areaCap () {
      return 33
    },
    factoryTotal () {
      return this.buildings.filter(b => b.building_id === 14).reduce((s, b) => s + b.level, 0)
    },
    factoryFree () {
      return Math.max(1, this.factoryTotal - this.queues.length)
    },
    isLeader () {
      return !!(this.myCorps && this.profile.user_id && this.myCorps.leader_user_id === this.profile.user_id)
    },
    mapRows () {
      const rows = []
      const size = this.mapR * 2 + 1
      for (let i = 0; i < size; i++) {
        rows.push(this.mapCells.slice(i * size, (i + 1) * size))
      }
      return rows
    }
  },
  mounted () {
    // 沉浸式: 去掉 body 默认的 5px 外边距, 标题条才能贴满屏幕上方与左右
    document.body.classList.add('ezfy-immersive')
    this.load()
    this.loadChats()
    this.loadNotices()
    this.loadCorps()
    this.nowText = this.fmtNow()
    this.nowTimer = setInterval(() => { this.nowText = this.fmtNow() }, 10000)
    this.timer = setInterval(() => {
      if (this.cur === 'home') this.load()
      if (this.cur === 'chat') this.loadChats()
    }, 30000)
  },
  beforeDestroy () {
    document.body.classList.remove('ezfy-immersive')
    if (this.timer) clearInterval(this.timer)
    if (this.nowTimer) clearInterval(this.nowTimer)
  },
  methods: {
    notOpen (what) {
      alert(what + '暂未开放, 敬请期待')
    },
    go (t) {
      if (t.indexOf('res/') === 0) {
        this.resType = t.slice(4)
        this.cur = 'res'
        this.loadRes()
        return
      }
      this.cur = t
      if (t === 'home') this.load()
      else if (t === 'troops' || t === 'troop' || t === 'defence') this.loadTroops()
      else if (t === 'hq') { this.loadTroops().then(() => this.loadTargets()); this.loadOrders() }
      else if (t === 'techs') this.loadTechs()
      else if (t === 'map') { this.loadMap(); this.loadStars() }
      else if (t === 'reports') this.loadReports()
      else if (t === 'mail') this.loadMails()
      else if (t === 'friends') this.loadFriends()
      else if (t === 'liaison') this.loadLiaison()
      else if (t === 'tasks') this.loadTasks()
      else if (t === 'welfare') this.loadWelfare()
      else if (t === 'rank') this.loadRank()
      else if (t === 'bag') this.loadBag()
      else if (t === 'mall') this.loadMall()
      else if (t === 'exchange') this.loadExchange()
      else if (t === 'corps') this.loadCorps()
      else if (t === 'orders') this.loadOrders()
      else if (t === 'notices') this.loadNotices()
      else if (t === 'wilds') this.loadWilds()
      else if (t === 'orderpre') { this.orderCalc = null; this.loadTroops(); this.loadOnDutyOfficers() }
      else if (t === 'acade') this.loadAcade()
      else if (t === 'wareset') this.loadWare()
      else if (t === 'citymove') this.loadMoveInfo()
      else if (t === 'sourceset') this.loadProduce()
      else if (t === 'activity') this.loadActivity()
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
    load () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) {
          const d = r.data
          this.profile = d.profile
          this.userBrief = { account: d.account || '', level: d.user_level || 0, exp: d.user_exp || 0 }
          this.officerCount = d.officer_count || 0
          this.rankName = d.rank_name
          this.rankPost = d.rank_post
          this.city = d.city
          this.cities = d.cities
          this.continent = d.continent
          this.protectedUntil = d.protected
          this.boostUntil = d.boost
          this.buildings = d.buildings
          this.buildingPool = d.building_pool || []
          this.wildlands = d.wildlands
          this.queues = d.queues
          this.marching = d.marching
          this.occupying = d.occupying
          this.unreadReports = d.unread_reports
          this.taxInput = d.city.tax_rate
        }
      })
    },
    loadTroops () {
      return api.get('/games/ezfy/troops').then(r => {
        if (r.code === 0) {
          this.troopsData = r.data
          if (r.data.cfgs.length && !this.trainSel) this.trainSel = null
          // 司令部配置表按兵种建键, 兵种数据后到时要补齐, 否则渲染会取到 undefined
          this.ensureTargetCfg()
        }
      })
    },
    // 保证每个兵种在 targetCfg 里都有条目(司令部页 v-model 直接取 targetCfg[id].atk)
    ensureTargetCfg () {
      const cfg = Object.assign({}, this.targetCfg)
      for (const t of (this.troopsData.cfgs || [])) {
        if (!cfg[t.id]) cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1 }
      }
      this.targetCfg = cfg
    },
    loadTechs () {
      api.get('/games/ezfy/techs').then(r => {
        if (r.code === 0) this.techsData = r.data
      })
    },
    loadChats () {
      api.get('/games/ezfy/chat?channel=' + this.chatChannel).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.worldChats = d.chats || []
          this.chatNotices = d.notices || []
          this.chatPlayers = d.players
          this.chatHasCorps = !!d.has_corps
          this.chatCorpsName = d.corps_name || ''
          this.chatCanSend = d.can_send !== false
          // 军团频道无军团时会降级为公共频道, 同步 tab 高亮
          if (d.channel && d.channel !== this.chatChannel) this.chatChannel = d.channel
        }
      })
    },
    switchChannel (ch) {
      this.chatChannel = ch
      this.chatMsg = ''
      this.loadChats()
    },
    loadMails () {
      api.get('/messages/inbox').then(r => {
        if (r.code === 0) this.mails = (r.data.list || []).slice(0, 30)
      })
    },
    loadFriends () {
      api.get('/friends').then(r => {
        if (r.code === 0) this.friends = r.data.friends || []
      })
    },
    // ---- 好友搜索/添加(复刻 addToFriend) ----
    doFriendSearch () {
      const kw = (this.friendKeyword || '').trim()
      if (!kw) { alert('请输入家园号码或昵称'); return }
      api.get('/friends/search?keyword=' + encodeURIComponent(kw)).then(r => {
        if (r.code === 0) {
          this.friendSearchList = r.data.list || []
          this.friendSearchDone = true
        } else {
          alert(r.msg || '搜索失败')
        }
      })
    },
    doAddFriend (u) {
      const remark = prompt('给 ' + u.nickname + ' 的验证信息(可留空)', '')
      if (remark === null) return
      api.post('/friends', { target_id: u.id, remark: remark }).then(r => {
        if (r.code === 0) {
          alert(r.data && r.data.msg ? r.data.msg : '已发送好友申请')
          this.doFriendSearch()
          this.loadFriends()
        } else {
          alert(r.msg || '添加失败')
        }
      })
    },
    loadReports () {
      api.get('/games/ezfy/reports').then(r => {
        if (r.code === 0) this.reports = r.data.reports
      })
    },
    loadTasks () {
      api.get('/games/ezfy/tasks').then(r => {
        if (r.code === 0) this.taskGroups = r.data.groups
      })
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
    loadBag () {
      api.get('/games/ezfy/bag').then(r => {
        if (r.code === 0) {
          this.bagItems = r.data.items
          this.bagOfficers = r.data.officers || []
          this.bagSkills = r.data.skills || []
        }
      })
    },
    loadMall () {
      api.get('/games/ezfy/mall').then(r => {
        if (r.code === 0) this.mallItems = r.data.items
      })
    },
    loadExchange () {
      api.get('/games/ezfy/exchange').then(r => {
        if (r.code === 0) {
          this.exchangeOrders = r.data.orders
          this.exchangeMine = r.data.mine
          this.exchangeGold = r.data.gold
        }
      })
    },
    loadCorps () {
      api.get('/games/ezfy/corps/list').then(r => {
        if (r.code === 0) {
          this.corpsList = r.data.corps
          this.myCorps = r.data.my_corps
        }
      })
      api.get('/games/ezfy/corps/members').then(r => {
        if (r.code === 0) this.corpsMembers = r.data.members
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
        if (r.code === 0) this.notices = r.data.notices
      })
    },    loadWilds () {
      api.get('/games/ezfy/city/wildfull').then(r => {
        if (r.code === 0) {
          this.wildlands = r.data.wildlands
          this.occupies = r.data.occupies
        }
      })
    },
    loadTargets () {
      return api.get('/games/ezfy/targets').then(r => {
        // 先按兵种表建全默认值, 再覆盖服务端已保存的配置
        const cfg = {}
        for (const t of (this.troopsData.cfgs || [])) {
          cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1 }
        }
        if (r.code === 0) {
          for (const t of r.data.targets) {
            cfg[t.troop_id] = { atk: t.atk_target_troop, atkMove: t.atk_move, def: t.def_target_troop, defMove: t.def_move }
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
    loadMap () {
      api.get('/games/ezfy/map').then(r => {
        if (r.code === 0) {
          this.mapCells = r.data.cells
          this.mapCx = r.data.cx
          this.mapCy = r.data.cy
        }
      })
    },
    moveMap (dx, dy) {
      api.get('/games/ezfy/map?x=' + (this.mapCx + dx) + '&y=' + (this.mapCy + dy) + '&r=' + this.mapR).then(r => {
        if (r.code === 0) {
          this.mapCells = r.data.cells
          this.mapCx = r.data.cx
          this.mapCy = r.data.cy
        }
      })
    },
    // ---- 地图坐标查找 / 收藏列表(复刻原版地图页) ----
    jumpTo (x, y) {
      api.get('/games/ezfy/map?x=' + x + '&y=' + y + '&r=' + this.mapR).then(r => {
        if (r.code === 0) {
          this.mapCells = r.data.cells
          this.mapCx = r.data.cx
          this.mapCy = r.data.cy
        }
      })
    },
    doJump () {
      const x = parseInt(this.jumpX)
      const y = parseInt(this.jumpY)
      if (!x || !y || x < 1 || x > 500 || y < 1 || y > 500) {
        alert('请输入 1~500 之间的横纵坐标')
        return
      }
      this.jumpTo(x, y)
    },
    loadStars () {
      api.get('/games/ezfy/map/stars').then(r => {
        if (r.code === 0) this.mapStars = r.data.stars || []
      })
    },
    toggleStars () {
      this.showStars = !this.showStars
      if (this.showStars) this.loadStars()
    },
    addStar () {
      if (!this.selCell) return
      const def = this.selCell.name + '(' + this.selCell.x + ',' + this.selCell.y + ')'
      const name = prompt('备注名(最多16字):', def)
      if (name === null) return
      api.post('/games/ezfy/map/stars', { x: this.selCell.x, y: this.selCell.y, name: name }).then(r => {
        if (r.code === 0) { this.showStars = true; this.loadStars() } else alert(r.msg || '收藏失败')
      })
    },
    delStar (s) {
      api.post('/games/ezfy/map/stars/delete', { id: s.id }).then(r => {
        if (r.code === 0) this.loadStars()
      })
    },
    // ---- 城市迁移(复刻 city/cityHallMove.html) ----
    loadMoveInfo () {
      api.get('/games/ezfy/city/move').then(r => {
        if (r.code === 0) {
          this.moveInfo = r.data
          if (r.data.areas && r.data.areas.length) this.moveArea = r.data.areas[0].id
        }
      })
    },
    doMoveCity (type) {
      const body = { type: type }
      if (type === 'low') {
        body.area_id = this.moveArea
      } else if (type === 'high') {
        const x = parseInt(this.moveX)
        const y = parseInt(this.moveY)
        if (!x || !y) { alert('请输入横纵坐标'); return }
        body.x = x; body.y = y
      } else {
        const x = parseInt(this.moveX2)
        const y = parseInt(this.moveY2)
        if (!x || !y) { alert('请输入横纵坐标'); return }
        body.x = x; body.y = y
      }
      const label = type === 'low' ? '迁城计划' : (type === 'high' ? '高级迁城计划' : '沿海迁城计划')
      if (!confirm('确认使用【' + label + '】迁移城市吗？将消耗 ' + this.moveInfo.gold_cost + ' 黄金。')) return
      api.post('/games/ezfy/city/move', body).then(r => {
        if (r.code === 0) {
          alert(r.data.msg)
          this.load()
          this.loadMoveInfo()
        } else alert(r.msg || '迁城失败')
      })
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
        if (!n || n < 1 || n > 100) { alert('开工率需在 1~100 之间'); return }
      }
      api.post('/games/ezfy/city/produce', {
        rate_food: parseInt(this.rateFood), rate_steel: parseInt(this.rateSteel),
        rate_oil: parseInt(this.rateOil), rate_rare: parseInt(this.rateRare)
      }).then(r => {
        this.alert(r)
        if (r.code === 0) this.loadProduce()
      })
    },
    // ---- 建筑操作 ----
    doBuild (b) {
      api.post('/games/ezfy/build', { building_id: b.building_id || b.bid }).then(r => {
        this.alert(r)
        if (r.code === 0) this.load()
      })
    },
    doUpgrade (b) {
      api.post('/games/ezfy/building/upgrade', { record_id: b.id }).then(r => {
        this.alert(r)
        if (r.code === 0) this.load()
      })
    },
    doMaxLevel (b) {
      api.post('/games/ezfy/building/max-level', { record_id: b.id }).then(r => {
        this.alert(r)
        if (r.code === 0) this.load()
      })
    },
    doDeleteBuilding (b) {
      api.post('/games/ezfy/building/delete', { record_id: b.id }).then(r => {
        this.alert(r)
        if (r.code === 0) this.load()
      })
    },
    doSpeedBuilding () {
      api.post('/games/ezfy/building/speed', { minutes: 10 }).then(r => {
        this.alert(r, '当前没有正在施工的建筑')
        if (r.code === 0) this.load()
      })
    },
    // 建筑名后的特殊入口(复刻原版 militaryIndex 里各建筑指向的功能页)
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
        14: { label: '训练', cur: 'troop' },
        15: { label: '联络中心', cur: 'liaison' }
      }
      return map[bid] || null
    },
    goEntry (bid) {
      const e = this.bEntry(bid)
      if (!e) return
      if (e.tab) this.acadeTab = e.tab
      this.go(e.cur)
    },
    // ---- 城市操作 ----
    doConvene () {
      api.post('/games/ezfy/city/convene', {}).then(r => this.alert(r))
    },
    doPlacate () {
      api.post('/games/ezfy/city/placate', {}).then(r => this.alert(r))
    },
    doTax () {
      api.post('/games/ezfy/city/tax', { tax_rate: parseInt(this.taxInput) || 0 }).then(r => this.alert(r))
    },
    doRename () {
      api.post('/games/ezfy/city/rename', { name: this.renameInput }).then(r => this.alert(r))
    },
    doCreateCity () {
      api.post('/games/ezfy/city/create', { x: parseInt(this.newCityX) || 0, y: parseInt(this.newCityY) || 0 }).then(r => this.alert(r))
    },
    doSwitch (ct) {
      api.post('/games/ezfy/city/switch', { city_id: ct.id }).then(r => {
        if (r.code === 0) {
          this.load()
          this.cur = 'home'
        } else alert(r.msg)
      })
    },
    doAbandon (w) {
      api.post('/games/ezfy/city/abandon-wild', { wildland_id: w.id }).then(r => this.alert(r))
    },
    doOccupy (op, o) {
      if (!window.confirm(op === 'build' ? '确定将该城市正式建立为自己的城市吗?' :
        op === 'destroy' ? '确定摧毁该城市吗? 城市及其建筑/部队将全部消失, 不可恢复!' :
          '确定将城市归还给原玩家吗?')) return
      api.post('/games/ezfy/city/occupy/' + op, { occupy_id: o.id }).then(r => this.alert(r))
    },
    openWildGather (w) {
      api.post('/games/ezfy/order', {
        order_type: 4, target_type: 1, target_x: w.x, target_y: w.y, target_id: w.id
      }).then(r => this.alert(r))
    },
    // ---- 军队 ----
    openTrain (t) {
      this.trainSel = t
      this.trainCount = 10
    },
    doTrain () {
      if (!this.trainSel) return
      api.post('/games/ezfy/troops/train', {
        troop_id: this.trainSel.id, count: parseInt(this.trainCount) || 0, split: this.trainSplit
      }).then(r => this.alert(r))
    },
    doRecover (w) {
      api.post('/games/ezfy/troops/recover', { troop_id: w.troop_id, type: w.type }).then(r => this.alert(r))
    },
    doRecoverAll (t) {
      api.post('/games/ezfy/troops/recover', { all: true, type: t }).then(r => this.alert(r))
    },
    // ---- 科技 ----
    doResearch (t) {
      api.post('/games/ezfy/techs/research', { tech_id: t.tech_id }).then(r => this.alert(r))
    },
    doSpeedTech () {
      api.post('/games/ezfy/techs/speed', { minutes: 10 }).then(r => {
        this.alert(r, '没有研究中的科技')
        if (r.code === 0) this.loadTechs()
      })
    },
    doCancelTech (t) {
      if (!confirm('确定取消研究「' + t.name + '」吗？本次消耗将全额退还。')) return
      api.post('/games/ezfy/techs/cancel', { tech_id: t.tech_id }).then(r => {
        this.alert(r)
        if (r.code === 0) this.loadTechs()
      })
    },
    // ---- 司令部 ----
    doSaveTargets () {
      const list = []
      for (const tid in this.targetCfg) {
        const t = this.targetCfg[tid]
        list.push(api.post('/games/ezfy/targets', {
          troop_id: parseInt(tid), atk_target_troop: t.atk, atk_move: t.atkMove,
          def_target_troop: t.def, def_move: t.defMove
        }))
      }
      Promise.all(list).then(() => alert('战斗配置已保存'))
    },
    openOrder (o) {
      api.get('/games/ezfy/orders/' + o.id).then(r => {
        if (r.code === 0) {
          this.curOrder = r.data
          this.cur = 'orderview'
        }
      })
    },
    jumpReport (rid) {
      api.get('/games/ezfy/reports/' + rid).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
          this.showDetail = false
          this.cur = 'reports'
        }
      })
    },
    doRecall (o) {
      api.post('/games/ezfy/order/recall', { order_id: o.id }).then(r => this.alert(r))
    },
    orderStatusText (o) {
      if (o.status === 0) return '行进中 ' + this.remain(o.arrive_time)
      if (o.status === 1) return (o.order_type === 7 ? '驻守采集' : '已到达')
      if (o.status === 2) return '返回中 ' + this.remain(o.return_time)
      if (o.status === 3) return '已完成'
      return '全队阵亡'
    },
    // ---- 地图/出征 ----
    cellText (cell) {
      if (cell.area_type === 3) return '城'
      if (cell.name === '寇城(废墟)') return '墟'
      if (cell.area_type === 2) return '寇' + cell.level
      if (cell.terrain === 8) return '海' + cell.level
      // 陆地野地按地形名显示(平原/草原/森林/盆地/丘陵/沼泽/山地)
      return (cell.terrain_name || '野') + cell.level
    },
    cellClass (cell) {
      if (cell.mine) return 'ezfy-mine'
      if (cell.area_type === 3) return 'ezfy-city'
      if (cell.area_type === 2) return 'ezfy-kou'
      if (cell.terrain === 8) return 'ezfy-sea'
      return 'ezfy-wild'
    },
    openCell (cell) {
      this.selCell = cell
      this.selDetail = null
      this.warText = ''
      if (cell.area_type === 3) {
        if (!cell.mine && cell.user_id) this.checkWar()
        return
      }
      const ttype = cell.area_type === 2 ? 3 : (cell.terrain === 8 ? 2 : 1)
      api.get('/games/ezfy/map/wildland?x=' + cell.x + '&y=' + cell.y + '&type=' + ttype).then(r => {
        if (r.code === 0) this.selDetail = r.data
      })
    },
    orderAvail (t) {
      if (t === 4 || t === 7) return false
      if ((t === 5 || t === 6) && this.selCell && this.selCell.area_type !== 3) return false
      return true
    },
    declareWar () {
      if (!this.selCell || !this.selCell.city_id) return
      api.post('/games/ezfy/war/declare', { city_id: this.selCell.city_id }).then(r => {
        this.alert(r)
        this.checkWar()
      })
    },
    checkWar () {
      if (!this.selCell || !this.selCell.user_id) return
      api.get('/games/ezfy/war/status?target_user_id=' + this.selCell.user_id).then(r => {
        if (r.code === 0) this.warText = r.data.text
      })
    },
    // 出征表单 → 请求体(部队/资源/军官/宿营)
    orderBody () {
      const body = {
        order_type: this.orderType,
        target_type: this.selCell.area_type === 3 ? 3 : (this.selCell.area_type === 2 ? 2 : 1),
        target_x: this.selCell.x,
        target_y: this.selCell.y,
        wait_min: (parseInt(this.waitH) || 0) * 60 + (parseInt(this.waitM) || 0)
      }
      if (this.selCell.area_type === 3 && this.selCell.city_id) {
        body.target_id = this.selCell.city_id
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
      return body
    },
    // 复刻原版出征页的 [计算]: 预览油耗/负重/耗时, 不下达命令
    doCalc () {
      if (!this.selCell) return
      api.post('/games/ezfy/order/preview', this.orderBody()).then(r => {
        if (r.code === 0) this.orderCalc = r.data
        else this.alert(r)
      })
    },
    doOrder () {
      if (!this.selCell) return
      api.post('/games/ezfy/order', this.orderBody()).then(r => {
        if (r.code === 0) {
          alert(r.data.msg)
          this.orderTroops = {}
          this.orderOfficer = '0'
          this.orderCalc = null
          this.trFood = 0; this.trSteel = 0; this.trOil = 0; this.trRare = 0; this.trGold = 0
          this.waitH = 0; this.waitM = 0
          this.load()
          this.cur = 'orders'
          this.loadOrders()
        } else alert(r.msg)
      })
    },
    loadOnDutyOfficers () {
      api.get('/games/ezfy/officers/onduty').then(r => {
        if (r.code === 0) this.onDutyOfficers = r.data.officers || []
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
      if (this.wareSum > 100) { alert('四项比例合计不能超过100%'); return }
      api.post('/games/ezfy/city/warehouse', {
        food: parseInt(this.wareRatio.food) || 0,
        steel: parseInt(this.wareRatio.steel) || 0,
        oil: parseInt(this.wareRatio.oil) || 0,
        rare: parseInt(this.wareRatio.rare) || 0
      }).then(r => {
        if (r.code === 0) {
          alert(r.data.msg)
          this.loadWare()
        } else alert(r.msg)
      })
    },
    // ---- 聊天/邮箱 ----
    doChatSend () {
      const msg = (this.chatMsg || '').trim()
      if (!msg) return
      if (this.chatCooldown > 0) {
        alert('发言冷却中，还需 ' + this.chatCooldown + ' 秒')
        return
      }
      api.post('/games/ezfy/chat', { content: msg, channel: this.chatChannel }).then(r => {
        if (r.code === 0) {
          this.chatMsg = ''
          // 复刻原版聊天: 每次发言 30 秒冷却
          this.startChatCooldown(30)
          this.loadChats()
        } else alert(r.msg)
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
    openUser (uid) {
      if (!uid) return
      this.$router.push('/user/' + uid)
    },
    openNotice (n) {
      this.curNotice = n
      this.cur = 'notices'
    },
    // ---- 军团 ----
    doCreateCorps () {
      api.post('/games/ezfy/corps/create', { name: this.corpsName }).then(r => this.alert(r))
    },
    doJoinCorps (cp) {
      api.post('/games/ezfy/corps/join', { corps_id: cp.id }).then(r => this.alert(r))
    },
    doLeaveCorps () {
      if (!window.confirm(this.isLeader ? '军团长退出将解散军团, 确定?' : '确定退出军团?')) return
      api.post('/games/ezfy/corps/leave', {}).then(r => this.alert(r))
    },
    openNoticeEdit () {
      const n = window.prompt('输入军团公告', this.myCorps ? this.myCorps.notice : '')
      if (n !== null) api.post('/games/ezfy/corps/notice', { notice: n }).then(r => this.alert(r))
    },
    doKick () {
      if (!this.kickUserId) return alert('请选择成员')
      api.post('/games/ezfy/corps/kick', { user_id: this.kickUserId }).then(r => this.alert(r))
    },
    doCorpsChat () {
      api.post('/games/ezfy/corps/chat', { content: this.corpsMsg }).then(r => {
        if (r.code === 0) {
          this.corpsMsg = ''
          this.loadCorps()
        } else alert(r.msg)
      })
    },
    // ---- 商城/背包/交易 ----
    openBuy (it) {
      this.buyItem = it
      this.buyCount = 1
    },
    doBuy (it) {
      const n = parseInt(this.buyCount) || 0
      if (n < 1 || n > 99) { alert('数量需在 1-99 之间'); return }
      api.post('/games/ezfy/mall/buy', { cfg_id: it.id, count: n }).then(r => {
        if (r.code === 0) {
          alert(r.data && r.data.msg ? r.data.msg : '购买成功')
          this.buyItem = null
          this.load()
          this.loadMall()
          this.loadBag()
        } else alert(r.msg || '购买失败')
      })
    },
    needOfficer (it) {
      return it.item_type === 10 || it.item_type === 11 || it.item_type === 12
    },
    openUse (it) {
      this.useItem = it
      this.useCount = 1
      this.useOfficerId = 0
      this.useSkillId = 0
    },
    doUse (it) {
      const body = { cfg_id: it.cfg_id, count: parseInt(this.useCount) || 1 }
      if (this.needOfficer(it)) {
        if (!this.useOfficerId) { alert('请先选择要使用的军官'); return }
        body.officer_id = this.useOfficerId
      }
      if (it.item_type === 11) {
        if (!this.useSkillId) { alert('请选择要学习的技能'); return }
        body.skill_id = this.useSkillId
      }
      api.post('/games/ezfy/bag/use', body).then(r => {
        if (r.code === 0) {
          alert(r.data && r.data.msg ? r.data.msg : '使用成功')
          this.useItem = null
          this.loadBag()
          this.load()
        } else alert(r.msg || '使用失败')
      })
    },
    doExchangeSell () {
      api.post('/games/ezfy/exchange/sell', {
        es_type: parseInt(this.sellType), es_count: parseInt(this.sellCount) || 0,
        total_price: parseInt(this.sellPrice) || 0
      }).then(r => this.alert(r))
    },
    doExchangeBuy (e) {
      api.post('/games/ezfy/exchange/buy', { id: e.id }).then(r => this.alert(r))
    },
    doExchangeCancel (e) {
      api.post('/games/ezfy/exchange/cancel', { id: e.id }).then(r => this.alert(r))
    },
    // ---- 任务/福利 ----
    doAward (t) {
      api.post('/games/ezfy/tasks/award', { task_id: t.id }).then(r => this.alert(r))
    },
    doSign () {
      api.post('/games/ezfy/welfare/sign', {}).then(r => this.alert(r))
    },
    doGift (t) {
      api.post('/games/ezfy/welfare/gift/' + t, {}).then(r => this.alert(r))
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
    troopTypeName (t) {
      return { 1: '海军', 2: '陆军', 3: '空军', 4: '城防' }[t] || '部队'
    },
    remain (endTime) {
      if (!endTime) return ''
      const ms = endTime - Date.now()
      if (ms <= 0) return '已完成'
      const s = Math.floor(ms / 1000)
      if (s < 60) return s + '秒'
      const m = Math.floor(s / 60)
      if (m < 60) return m + '分' + (s % 60) + '秒'
      const h = Math.floor(m / 60)
      return h + '时' + (m % 60) + '分'
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
      return p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    fmtNow () {
      const d = new Date()
      const p = n => String(n).padStart(2, '0')
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    },
    alert (r, fallback) {
      if (r.code === 0) {
        alert(r.data && r.data.msg ? r.data.msg : (fallback || '操作成功'))
        this.load()
      } else {
        alert(r.msg || '操作失败')
      }
    },
    // ---- 军官/学院 ----
    loadAcade () {
      api.get('/games/ezfy/officers').then(r => {
        if (r.code === 0) this.officerData = r.data
      })
      this.loadAcadeTab()
    },
    loadAcadeTab () {
      if (this.acadeTab === 'search') this.loadRecruit()
      else if (this.acadeTab === 'skill') this.loadAcadeSkills()
      else if (this.acadeTab === 'equip') this.loadAcadeEquip()
      else if (this.acadeTab === 'scheme') this.loadAcadeGenerals()
      else {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      }
    },
    switchAcade (tab) {
      this.acadeTab = tab
      if (tab === 'officer' || tab === 'mayor' || tab === 'captive') {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      } else if (tab === 'search') this.loadRecruit()
      else if (tab === 'skill') this.loadAcadeSkills()
      else if (tab === 'equip') this.loadAcadeEquip()
      else if (tab === 'generals') this.loadAcadeGenerals()
    },
    loadRecruit () {
      api.get('/games/ezfy/acade/recruit').then(r => {
        if (r.code === 0) this.recruitData = r.data
      })
    },
    loadAcadeSkills () {
      api.get('/games/ezfy/officers/skills').then(r => {
        if (r.code === 0) this.skillData = r.data
      })
    },
    loadAcadeEquip () {
      api.get('/games/ezfy/officers/equipments').then(r => {
        if (r.code === 0) this.equipData = r.data
      })
    },
    loadAcadeGenerals () {
      api.get('/games/ezfy/officers/generals').then(r => {
        if (r.code === 0) this.generalData = r.data
      })
    },
    openOfficer (id) {
      this.cur = 'officerdetail'
      this.loadOfficerDetail(id)
    },
    loadOfficerDetail (id) {
      api.get('/games/ezfy/officers/' + id).then(r => {
        if (r.code === 0) this.officerDetail = r.data
      })
    },
    doRefreshRecruit () {
      api.post('/games/ezfy/acade/recruit/refresh', {}).then(r => {
        if (r.code !== 0) alert(r.msg || '刷新失败')
        this.loadRecruit()
      })
    },
    doRecruit (g) {
      if (!confirm('确定招募 ' + g.name + ' 吗? 需要 ' + g.cost + ' 黄金')) return
      api.post('/games/ezfy/acade/recruit/' + g.id, {}).then(r => {
        if (r.code !== 0) alert(r.msg || '招募失败')
        this.loadRecruit()
      })
    },
    doGrant () {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/grant', {}).then(r => {
        if (r.code !== 0) alert(r.msg || '赏赐失败')
        this.loadOfficerDetail(id)
      })
    },
    doLearn (s) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'learn', skill_id: s.id }).then(r => {
        if (r.code !== 0) alert(r.msg || '学习失败')
        this.loadOfficerDetail(id)
      })
    },
    doForget (name) {
      const id = this.officerDetail.officer.id
      const sid = this.skillIdByName(name)
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'forget', skill_id: sid }).then(r => {
        if (r.code !== 0) alert(r.msg || '遗忘失败')
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
        if (r.code !== 0) alert(r.msg || '穿戴失败')
        this.loadOfficerDetail(id)
      })
    },
    doUnequip (equipId) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: equipId, op: 'off' }).then(r => {
        if (r.code !== 0) alert(r.msg || '卸下失败')
        this.loadOfficerDetail(id)
      })
    },
    doPosition (o, pos) {
      api.post('/games/ezfy/officers/' + o.id + '/position', { position: pos }).then(r => {
        if (r.code !== 0) alert(r.msg || '任命失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
      })
    },
    doCaptive (o, op) {
      if (op === 'free' && !confirm('确定释放俘虏 ' + o.name + ' 吗?')) return
      api.post('/games/ezfy/officers/' + o.id + '/captive', { op: op }).then(r => {
        if (r.code !== 0) alert(r.msg || '操作失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
      })
    },
    doExile () {
      if (!confirm('确定流放该武将吗? 流放后无法找回!')) return
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/exile', {}).then(r => {
        if (r.code !== 0) alert(r.msg || '流放失败')
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
  font-size: 16px;
  color: #333;
  /* 根容器左右不再用负 margin: 会溢出 #app 产生横向滚动条.
     铺满由内部 .title-bar 的 margin:0 -8px 抵消 padding 实现 */
  margin: 0;
  padding: 0 8px 20px;
}
.ezfy-page .home-wrap {
  font-size: 16px;
}
.ezfy-page a {
  text-decoration: none;
  margin: 0 1px;
  color: #0645ad;
}
.ezfy-page a:hover { text-decoration: underline; }
.ezfy-page .title-bar {
  background: #2f4156;
  color: #fff;
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
  padding: 4px 0;
}
.ezfy-page .top-nav a {
  display: inline-block;
  padding: 2px 5px;
  font-size: 15px;
}
.ezfy-page .panel { margin-top: 8px; padding: 2px; }
.ezfy-page .acade-tab {
  padding: 3px 0;
  font-size: 14px;
  color: #666;
}
.ezfy-page .acade-tab a { color: #2f4156; }
.ezfy-page .acade-tab a.on { color: #c0392b; font-weight: bold; }
.ezfy-page .use-box {
  margin: 4px 0 6px 8px;
  padding: 4px 6px;
  border-left: 2px solid #d8d5cc;
  line-height: 1.9;
}
.ezfy-page .panel-title {
  font-size: 15px;
  font-weight: bold;
  color: #2f4156;
  margin: 6px 0 2px;
}
.ezfy-page .old-line { padding: 2px 0; word-break: break-all; }
.ezfy-page .city-name { font-size: 16px; font-weight: bold; color: #2f4156; }
.ezfy-page table {
  width: 100%;
  border-collapse: collapse;
  font-size: 15px;
}
.ezfy-page table th {
  color: #2f4156;
  padding: 3px 6px;
  text-align: left;
  font-weight: bold;
  border-bottom: 1px solid #ccc;
}
.ezfy-page table td {
  padding: 3px 6px;
  border-bottom: 1px dotted #ddd;
}
/* 返回按钮与 [造兵]/[建防]/[退出军团] 等普通操作链接同款: 纯文字链接, 无填充 */
.ezfy-page .bottom-nav { margin-top: 10px; padding: 4px 0; text-align: left; }
.ezfy-page .footer { text-align: center; font-size: 13px; color: #999; padding: 4px 0 10px; }
.ezfy-page .logo-title { height: 14px; vertical-align: -2px; }
.ezfy-page .red { color: #c0392b; }
.ezfy-page .gray { color: #999; }
.ezfy-page .green { color: #27763c; }
.ezfy-page .orange { color: #b8860b; }
.ezfy-page input[type="text"],
.ezfy-page input[type="number"],
.ezfy-page input:not([type]),
.ezfy-page select,
.ezfy-page textarea {
  border: 1px solid #999;
  border-radius: 0;
  padding: 3px 4px;
  font-size: 15px;
  background: #fff;
  color: #333;
}
.ezfy-page button {
  border: 1px solid #888;
  border-radius: 0;
  background: #e8e5dd;
  color: #333;
  font-size: 14px;
  padding: 2px 8px;
  cursor: pointer;
}
.ezfy-page button:hover { background: #d8d5cc; }
.ezfy-page .report-pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: inherit;
  font-size: 14px;
  background: #fff;
  border: 1px solid #ddd;
  padding: 6px;
  margin: 4px 0;
}
/* 地图 */
.ezfy-map { padding: 4px 0; overflow-x: auto; }
.ezfy-map-row { white-space: nowrap; }
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
</style>
