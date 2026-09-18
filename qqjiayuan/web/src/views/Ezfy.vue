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
          <div class="panel-title">世界聊天({{ chatPlayers }}人)</div>
          <div class="old-line" v-for="ch in worldChats" :key="'c' + ch.id">
            [{{ ch.user_name }}]:{{ ch.content }}
          </div>
          <div class="old-line" v-if="!worldChats.length">(暂无消息, 快来说点什么吧)</div>
          <br/>
          <input v-model="chatMsg" style="width:72%" @keyup.enter="doChatSend"/>
          <button @click="doChatSend">发送</button>
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
          <div class="panel-title">家园好友</div>
          <table>
            <tr><th>昵称</th><th>等级</th><th>状态</th></tr>
            <tr v-for="f in friends" :key="'f' + f.id">
              <td>{{ f.nickname }}</td>
              <td>Lv.{{ f.level }}</td>
              <td><span :class="f.online ? 'green' : 'gray'">{{ f.online ? '在线' : '离线' }}</span></td>
            </tr>
          </table>
          <div class="old-line" v-if="!friends.length">(还没有好友, 去家园社区添加)</div>
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
          <a href="javascript:;" @click="go('builds')">[资源区]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 建筑区(buildm/builds) ============ -->
      <template v-else-if="cur === 'buildm' || cur === 'builds'">
        <div class="panel">
          <div class="panel-title">{{ cur === 'buildm' ? '军事区(含城防)' : '资源区' }}
            <span class="gray">({{ areaCount }}/{{ areaCap }})</span></div>
          <div class="old-line" v-for="b in zoneBuildings" :key="'zb' + (b.id || b.bid)">
            <template v-if="b.id">
              <b>{{ b.name }}</b> {{ b.level }}级
              <span v-if="b.status === 0">{{ b.effect }}</span>
              <span v-else class="orange">施工中 {{ remain(b.end_time) }}
                <a href="javascript:;" @click="doSpeedBuilding()">[加速]</a></span>
              <br/>
              <span v-if="b.status === 0 && b.level > 0 && b.level < b.max_level">
                <a href="javascript:;" @click="doUpgrade(b)">[升级]</a>
                <a href="javascript:;" @click="doMaxLevel(b)">[满级]</a>
              </span>
              <span v-if="b.status === 0 && b.level === 0"><a href="javascript:;" @click="doUpgrade(b)">[建成中待完成]</a></span>
              <span v-if="b.can_delete === 1 && b.status === 0 && b.level > 0"><a href="javascript:;" @click="doDeleteBuilding(b)">[拆除]</a></span>
              <span v-if="b.next_effect" class="gray">下一级:{{ b.next_effect }}</span>
            </template>
            <template v-else>
              <b>{{ b.name }}</b>(未建造)<br/>
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
          <div class="panel-title">科技研究(科研中心{{ techsData.academy }}级)</div>
          <div class="old-line" v-for="t in techsData.techs" :key="'te' + t.tech_id">
            <b>{{ t.name }}</b> {{ t.level }}/{{ t.max_level }}级
            <span class="gray">[需科研中心{{ t.academy_need }}级]</span><br/>
            {{ t.effect }}<br/>
            <span v-if="t.researching" class="orange">研究中 {{ remain(t.end_time) }}
              <a href="javascript:;" @click="doSpeedTech()">[加速]</a></span>
            <span v-else-if="t.level < t.max_level">
              <a href="javascript:;" @click="doResearch(t)">[研究{{ t.level + 1 }}级]</a>
              <span class="gray">耗: 粮{{ t.next_cost.food }} 钢{{ t.next_cost.steel }} 油{{ t.next_cost.oil }} 稀{{ t.next_cost.rare }} 金{{ t.next_cost.gold }} 需{{ Math.ceil(t.next_time / 60) }}分钟</span>
            </span>
            <span v-else class="gray">[已满级]</span>
            <br/>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 地图(map) ============ -->
      <template v-else-if="cur === 'map'">
        <div class="panel">
          <div class="panel-title">世界地图({{ mapCx }},{{ mapCy }} 附近)</div>
          <div class="old-line">
            <a href="javascript:;" @click="moveMap(-mapR, 0)">[西]</a>
            <a href="javascript:;" @click="moveMap(mapR, 0)">[东]</a>
            <a href="javascript:;" @click="moveMap(0, -mapR)">[北]</a>
            <a href="javascript:;" @click="moveMap(0, mapR)">[南]</a>
            <a href="javascript:;" @click="loadMap()">[回城]</a>
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
          <div class="old-line gray">城=城市 寇=寇城 海=海野 野=野地 数字=等级</div>
          <template v-if="selCell">
            <div class="panel-title">目标({{ selCell.x }},{{ selCell.y }})</div>
            <div class="old-line" v-if="selDetail">
              {{ selDetail.name }} 等级{{ selDetail.level }}<br/>
              <span v-for="tp in selDetail.troops" :key="'sp' + tp.troop_id">{{ tp.name }}约{{ tp.min }}-{{ tp.max }} </span><br/>
              掠夺资源约:{{ selDetail.res_min }}-{{ selDetail.res_max }}
            </div>
            <div class="old-line" v-else>
              {{ selCell.name }}
              <span v-if="selCell.city_level">{{ selCell.city_level }}级</span>
              <span v-if="selCell.owner">城主:{{ selCell.owner }}</span>
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
          <div class="panel-title">{{ orderNames[orderType] }} → ({{ selCell.x }},{{ selCell.y }})</div>
          <div class="old-line" v-if="orderType === 5">
            运输资源:<br/>
            粮食<input v-model="trFood" type="number" style="width:70px"/><br/>
            钢铁<input v-model="trSteel" type="number" style="width:70px"/><br/>
            石油<input v-model="trOil" type="number" style="width:70px"/><br/>
            稀矿<input v-model="trRare" type="number" style="width:70px"/><br/>
            黄金<input v-model="trGold" type="number" style="width:70px"/><br/>
          </div>
          <div class="old-line" v-else>
            选择部队(城内可用):<br/>
            <div v-for="t in attackTroops" :key="'at' + t.troop_id">
              {{ t.name }}(有{{ t.count }}):
              <input type="number" min="0" v-model="orderTroops[t.troop_id]" style="width:80px"/>
            </div>
            <span v-if="!attackTroops.length" class="red">城内无可出征部队</span>
          </div>
          <div class="old-line">
            带队军官:
            <select v-model="orderOfficer">
              <option value="">无</option>
              <option v-for="o in onDutyOfficers" :key="'od' + o.id" :value="o.name">
                {{ o.name }} Lv{{ o.level }} 军事{{ o.military }} 忠诚{{ o.loyalty }}
              </option>
            </select>
            <span v-if="orderType === 7" class="red">(派遣必须选择)</span>
            <span v-else-if="orderType === 6" class="gray">(增援后军官调任目标城市)</span>
            <br/>
            <span v-if="curOfficerBonus" class="green">军官战斗加成: 攻击+{{ curOfficerBonus }}%</span>
            <span v-if="!onDutyOfficers.length" class="gray">(暂无可用军官, 可前往军校招募)</span>
          </div>
          <div class="old-line">
            <button @click="doOrder()">出 发</button>
            <a href="javascript:;" @click="go('map')">[返回地图]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="go('map')">[前往地图]</a></div>
        </div>
      </template>

      <!-- ============ 命令详情(order view) ============ -->
      <template v-else-if="cur === 'orderview'">
        <div class="panel" v-if="curOrder">
          <div class="panel-title">{{ curOrder.type_name }}命令详情</div>
          目标: ({{ curOrder.target_x }},{{ curOrder.target_y }})<br/>
          状态: {{ orderStatusText(curOrder) }}<br/>
          统帅: {{ curOrder.officer || '无(未带军官)' }}<br/>
          耗油: {{ curOrder.oil_used }}<br/>
          部队:<br/>
          <div class="old-line" v-for="(t, i) in curOrder.troops" :key="'ot' + i">
            {{ t.name }}×{{ t.count }}
          </div>
          <div class="old-line" v-if="curOrder.report_id">
            <a href="javascript:;" @click="jumpReport(curOrder.report_id)">[查看战报]</a>
          </div>
          <a href="javascript:;" @click="go('orders')">[返回命令列表]</a>
        </div>
      </template>

      <!-- ============ 出征队列(orders) ============ -->
      <template v-else-if="cur === 'orders'">
        <div class="panel">
          <div class="panel-title">出征队列({{ orders.length }})</div>
          <table>
            <tr><th>类型</th><th>目标</th><th>统帅</th><th>状态</th><th>操作</th></tr>
            <tr v-for="o in orders" :key="'odl' + o.id">
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 附属野地(wilds) ============ -->
      <template v-else-if="cur === 'wilds'">
        <div class="panel">
          <div class="panel-title">占领野地({{ wildlands.length }}/{{ city.city_level }})</div>
          <table>
            <tr><th>坐标</th><th>类型</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="w in wildlands" :key="'wd' + w.id">
              <td>({{ w.x }},{{ w.y }})</td>
              <td>{{ w.wild_type === 2 ? '海野' : '陆地野地' }}</td>
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
            <a href="javascript:;" @click="go('rename')">[城市改名]</a>
            <a href="javascript:;" @click="go('cities')">[城市列表/迁建]</a>
            <a href="javascript:;" @click="go('citystatus')">[城市状态]</a>
            <a href="javascript:;" @click="go('wilds')">[附属野地]</a>
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
            <a href="javascript:;" @click="doUse(it)">[使用]</a><br/>
            <span class="gray">{{ it.description }}</span>
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
            <b>{{ it.name }}</b> {{ it.price_gold }}黄金 <a href="javascript:;" @click="doBuy(it)">[购买]</a><br/>
            <span class="gray">{{ it.description }}</span>
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
          <div class="old-line">[开服活动] 新手礼包、每周福利、市政厅等级礼包持续发放中, 前往<a href="javascript:;" @click="go('welfare')">[福利]</a>领取。</div>
          <div class="old-line">[征战天下] 征服野地/寇城可获得军功声望, 声望晋升军衔!</div>
          <div class="old-line">[物资兑换] 交易所开放资源交易, 低买高卖赚黄金。</div>
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

      <!-- ============ 联络(liaison) ============ -->
      <template v-else-if="cur === 'liaison'">
        <div class="panel">
          <div class="panel-title">联络</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('friends')">[好友列表]</a>
            <a href="javascript:;" @click="go('mail')">[邮箱]</a>
            <a href="javascript:;" @click="go('chat')">[世界聊天]</a>
            <a href="javascript:;" @click="go('corps')">[军团]</a>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 统帅(info) ============ -->
      <template v-else-if="cur === 'info'">
        <div class="panel">
          <div class="panel-title">统帅信息</div>
          昵称: {{ profile.nickname }}<br/>
          阵营: {{ profile.camp === 2 ? '轴心国' : '同盟国' }}<br/>
          军功声望: {{ profile.prestige }} ({{ rankName }}/{{ rankPost }})<br/>
          城池: {{ cities.length }}座<br/>
          总兵力: {{ totalTroops }}<br/>
          城外行进: {{ marching }}支 | 驻守采集: {{ occupying }}支<br/>
          占领野地: {{ wildlands.length }}块<br/>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军官/学院(acade) ============ -->
      <template v-else-if="cur === 'acade'">
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: acadeTab === 'officer' }" @click="switchAcade('officer')">军官</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'scheme' }" @click="switchAcade('scheme')">计谋</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'search' }" @click="switchAcade('search')">招募</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'mayor' }" @click="switchAcade('mayor')">任命市长</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'equip' }" @click="switchAcade('equip')">装备</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'skill' }" @click="switchAcade('skill')">技能</a>
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
          <div class="old-line">我的军官({{ officerData.officers.length }}):</div>
          <table>
            <tr><th>名称</th><th>星</th><th>等级</th><th>经验</th><th>军事</th><th>后勤</th><th>学习</th><th>忠诚</th><th>职位</th><th>状态</th><th>操作</th></tr>
            <tr v-for="o in officerData.officers" :key="'of' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.star }}</td>
              <td>{{ o.level }}</td>
              <td>{{ o.exp }}</td>
              <td>{{ o.military }}</td>
              <td>{{ o.logistics }}</td>
              <td>{{ o.learning }}</td>
              <td>{{ o.loyalty }}</td>
              <td>{{ o.position_name }}</td>
              <td>
                <span :class="{ orange: o.status === 1, red: o.is_captive === 1 }">{{ o.status_name }}</span>
              </td>
              <td>
                <a href="javascript:;" @click="openOfficer(o.id)">[详情]</a>
                <template v-if="o.is_captive === 1 && o.status !== 1">
                  <a href="javascript:;" @click="doCaptive(o, 'recruit')">[收编]</a>
                  <a href="javascript:;" @click="doCaptive(o, 'free')">[释放]</a>
                </template>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerData.officers.length">(暂无军官, 先去招募吧)</div>
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
            <tr v-for="o in officerData.officers" :key="'my' + o.id">
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

        <!-- 计谋(名将图鉴) -->
        <div class="panel" v-else-if="acadeTab === 'scheme'">
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
        <div class="bottom-nav"><a class="btn-return" href="javascript:;" @click="go('home')">返回游戏</a></div>
        <div class="footer">WAP报时:{{ nowText }}</div>
      </template>
    </div>
  </div>
</template>

<script>
import api from '../api'

const RES_NAMES = { gold: '黄金', food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿' }

export default {
  name: 'Ezfy',
  data () {
    return {
      cur: 'home',
      resNames: RES_NAMES,
      profile: { prestige: 0, camp: 1, nickname: '' },
      rankName: '列兵',
      rankPost: '士兵',
      city: {},
      cities: [],
      continent: '',
      protectedUntil: false,
      boostUntil: false,
      buildings: [],
      troopsData: { troops: [], queues: [], wounded: [], cfgs: [], pop: 0, pop_used: 0, wall_level: 0 },
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
      mails: [],
      friends: [],
      taskGroups: [],
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
      orderOfficer: '',
      trFood: 0,
      trSteel: 0,
      trOil: 0,
      trRare: 0,
      trGold: 0,
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
      const isM = this.cur === 'buildm'
      const zoneIds = isM ? [7, 8, 13, 14, 18, 19, 20, 21] : [2, 3, 4, 5, 6, 12]
      const built = this.buildings.filter(b => zoneIds.indexOf(b.building_id) !== -1)
      const builtIds = {}
      built.forEach(b => { builtIds[b.building_id] = true })
      const pool = isM
        ? [
          { bid: 8, name: '科研中心', type: 2, max_level: 10, des: '研究城市各方面科技', cost: { food: 300, steel: 1200, oil: 600, rare: 300, gold: 0 }, time: 1800 },
          { bid: 13, name: '司令部', type: 2, max_level: 10, des: '军队出征指挥中心', cost: { food: 2000, steel: 3000, oil: 1500, rare: 800, gold: 0 }, time: 3600 },
          { bid: 14, name: '军工厂', type: 2, max_level: 10, des: '生产现代化部队装备设施(最多5个)', cost: { food: 1500, steel: 2600, oil: 1200, rare: 600, gold: 0 }, time: 2700 },
          { bid: 18, name: '停机坪', type: 2, max_level: 10, des: '生产航空部队', cost: { food: 4000, steel: 6000, oil: 3000, rare: 1500, gold: 0 }, time: 5400 },
          { bid: 19, name: '航海协会', type: 2, max_level: 10, des: '生产海军部队(只能建在沿海城市)', cost: { food: 5000, steel: 8000, oil: 4000, rare: 2000, gold: 0 }, time: 6300 },
          { bid: 20, name: '运输站', type: 2, max_level: 10, des: '提高部队行动速度', cost: { food: 2500, steel: 4000, oil: 2000, rare: 1000, gold: 0 }, time: 4500 },
          { bid: 21, name: '雷达站', type: 2, max_level: 10, des: '对敌军入侵进行预警', cost: { food: 3000, steel: 5000, oil: 2500, rare: 1200, gold: 0 }, time: 5000 },
          { bid: 7, name: '围墙', type: 3, max_level: 10, des: '城市防御主导, 可建造碉堡等城防', cost: { food: 4500, steel: 3000, oil: 1500, rare: 700, gold: 0 }, time: 1600 }
        ]
        : [
          { bid: 2, name: '民居', type: 2, max_level: 10, des: '提供人口上限(最多10个)', cost: { food: 100, steel: 500, oil: 100, rare: 50, gold: 0 }, time: 3 },
          { bid: 3, name: '农田', type: 1, max_level: 10, des: '为城市提供粮食生产', cost: { food: 50, steel: 50, oil: 150, rare: 300, gold: 0 }, time: 45 },
          { bid: 4, name: '炼钢厂', type: 1, max_level: 10, des: '为城市提供钢铁资源', cost: { food: 50, steel: 50, oil: 300, rare: 150, gold: 0 }, time: 45 },
          { bid: 5, name: '石油基地', type: 1, max_level: 10, des: '为城市提供石油资源', cost: { food: 180, steel: 500, oil: 150, rare: 210, gold: 0 }, time: 86 },
          { bid: 6, name: '稀矿厂', type: 1, max_level: 10, des: '为城市提供稀矿资源', cost: { food: 600, steel: 500, oil: 210, rare: 200, gold: 0 }, time: 125 },
          { bid: 12, name: '仓库', type: 2, max_level: 10, des: '储存资源, 提高资源容量上限', cost: { food: 800, steel: 600, oil: 400, rare: 200, gold: 0 }, time: 900 }
        ]
      return built.concat(pool.filter(p => !builtIds[p.bid]))
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
      else if (t === 'hq') { this.loadTroops(); this.loadOrders(); this.loadTargets() }
      else if (t === 'techs') this.loadTechs()
      else if (t === 'map') this.loadMap()
      else if (t === 'reports') this.loadReports()
      else if (t === 'mail') this.loadMails()
      else if (t === 'friends' || t === 'liaison') this.loadFriends()
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
      else if (t === 'orderpre') { this.loadTroops(); this.loadOnDutyOfficers() }
      else if (t === 'acade') this.loadAcade()
    },
    load () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) {
          const d = r.data
          this.profile = d.profile
          this.rankName = d.rank_name
          this.rankPost = d.rank_post
          this.city = d.city
          this.cities = d.cities
          this.continent = d.continent
          this.protectedUntil = d.protected
          this.boostUntil = d.boost
          this.buildings = d.buildings
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
      api.get('/games/ezfy/troops').then(r => {
        if (r.code === 0) {
          this.troopsData = r.data
          if (r.data.cfgs.length && !this.trainSel) this.trainSel = null
        }
      })
    },
    loadTechs () {
      api.get('/games/ezfy/techs').then(r => {
        if (r.code === 0) this.techsData = r.data
      })
    },
    loadChats () {
      api.get('/games/ezfy/chat').then(r => {
        if (r.code === 0) {
          this.worldChats = r.data.chats
          this.chatPlayers = r.data.players
        }
      })
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
        if (r.code === 0) this.bagItems = r.data.items
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
      api.get('/games/ezfy/targets').then(r => {
        const cfg = {}
        if (r.code === 0) {
          for (const t of r.data.targets) {
            cfg[t.troop_id] = { atk: t.atk_target_troop, atkMove: t.atk_move, def: t.def_target_troop, defMove: t.def_move }
          }
        }
        for (const t of this.troopsData.cfgs) {
          if (!cfg[t.id]) cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1 }
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
    // ---- 建筑操作 ----
    doBuild (b) {
      api.post('/games/ezfy/build', { building_id: b.bid }).then(r => this.alert(r))
    },
    doUpgrade (b) {
      api.post('/games/ezfy/building/upgrade', { record_id: b.id }).then(r => this.alert(r))
    },
    doMaxLevel (b) {
      api.post('/games/ezfy/building/max-level', { record_id: b.id }).then(r => this.alert(r))
    },
    doDeleteBuilding (b) {
      api.post('/games/ezfy/building/delete', { record_id: b.id }).then(r => this.alert(r))
    },
    doSpeedBuilding () {
      api.post('/games/ezfy/building/speed', { minutes: 10 }).then(r => this.alert(r, '当前没有正在施工的建筑'))
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
      api.post('/games/ezfy/techs/speed', { minutes: 10 }).then(r => this.alert(r, '没有研究中的科技'))
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
      return '野' + cell.level
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
    doOrder () {
      if (!this.selCell) return
      const body = {
        order_type: this.orderType,
        target_type: this.selCell.area_type === 3 ? 3 : (this.selCell.area_type === 2 ? 2 : 1),
        target_x: this.selCell.x,
        target_y: this.selCell.y
      }
      if (this.selCell.area_type === 3 && this.selCell.city_id) {
        body.target_id = this.selCell.city_id
      }
      if (this.orderType === 5) {
        body.resources = {
          food: parseInt(this.trFood) || 0, steel: parseInt(this.trSteel) || 0,
          oil: parseInt(this.trOil) || 0, rare: parseInt(this.trRare) || 0,
          gold: parseInt(this.trGold) || 0
        }
      } else {
        const troops = []
        for (const k in this.orderTroops) {
          const n = parseInt(this.orderTroops[k]) || 0
          if (n > 0) troops.push({ troopId: parseInt(k), count: n })
        }
        body.troops = troops
      }
      if (this.orderOfficer) body.officer = this.orderOfficer
      api.post('/games/ezfy/order', body).then(r => {
        if (r.code === 0) {
          alert(r.data.msg)
          this.orderTroops = {}
          this.orderOfficer = ''
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
    // ---- 聊天/邮箱 ----
    doChatSend () {
      api.post('/games/ezfy/chat', { content: this.chatMsg }).then(r => {
        if (r.code === 0) {
          this.chatMsg = ''
          this.loadChats()
        } else alert(r.msg)
      })
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
    doBuy (it) {
      api.post('/games/ezfy/mall/buy', { cfg_id: it.id, count: 1 }).then(r => this.alert(r))
    },
    doUse (it) {
      api.post('/games/ezfy/bag/use', { cfg_id: it.cfg_id }).then(r => this.alert(r))
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
      if (tab === 'officer' || tab === 'mayor') {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      } else if (tab === 'search') this.loadRecruit()
      else if (tab === 'skill') this.loadAcadeSkills()
      else if (tab === 'equip') this.loadAcadeEquip()
      else if (tab === 'scheme') this.loadAcadeGenerals()
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
.ezfy-page .bottom-nav { margin-top: 10px; padding: 4px 0; text-align: left; }
.ezfy-page .bottom-nav .btn-return {
  display: inline-block;
  background: #2f4156;
  color: #fff;
  font-weight: bold;
  font-size: 14px;
  padding: 4px 20px;
  margin: 4px 0 2px;
  border-radius: 4px;
  border: 1px solid #46607d;
  box-shadow: 0 1px 3px rgba(31, 51, 87, 0.3);
}
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
  font-size: 12px;
  cursor: pointer;
  color: #444;
}
.ezfy-mine { background: #ffe9b0; border-color: #d0a030; color: #803000; }
.ezfy-city { background: #d8e4f0; border-color: #90a8c0; }
.ezfy-kou { background: #f0d8d8; border-color: #c09090; }
.ezfy-sea { background: #c8e0f0; border-color: #80a8c8; }
.ezfy-wild { background: #e8f0d8; border-color: #b8c89a; }
</style>
