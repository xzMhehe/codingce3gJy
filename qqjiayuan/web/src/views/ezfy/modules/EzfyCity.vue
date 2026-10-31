<template>
  <div>
    <template v-if="ezfy.cur === 'cities'">
        <div class="panel">
          <div class="panel-title">我的城市列表</div>
          <div class="old-line" v-for="ct in ezfy.cities" :key="'ct' + ct.id">
            <!-- ★ 2026-09-29 城市列表改版：一行一座城 = 城市名(坐标)；点城市名**切换**；
                 [运输][派遣][弃城] 只在非当前城显示（当前城无操作） -->
            <!-- ★ 2026-10-07 修复编译错误：v-if / v-else 之间不能夹文本，
                 原来写成 `<a v-if>…</a>(坐标)` + `<span v-else>…</span>(坐标)`，
                 Vue 2 报「text between v-if and v-else will be ignored」。
                 改成各自用 <template> 包住，坐标仍留在链接/文字**外面**（不变成可点区域）。 -->
            <template v-if="ct.id !== ezfy.city.id">
              <a class="city-name" href="javascript:;" @click="ezfy.doSwitch(ct)" title="切换为当前城市">{{ ct.name }}</a>({{ ct.x }},{{ ct.y }})
            </template>
            <template v-else>
              <span class="city-name">{{ ct.name }}</span>({{ ct.x }},{{ ct.y }})
            </template>
            <template v-if="ct.id !== ezfy.city.id && !ct.occupied">
              <a href="javascript:;" @click="ezfy.doTransportTo(ct)">[运输]</a>
              <a href="javascript:;" @click="ezfy.doDispatchTo(ct)">[派遣]</a>
              <a class="red" href="javascript:;" @click="ezfy.doDestroyCity(ct)">[弃城]</a>
            </template>
            <!-- ★ 2026-10-07 赎城：自己的城市被人占领后，不显示[运输][派遣][弃城] -->
            <!-- ★ 2026-10-25 用户需求「改成 玩家赎回 无需 占领者同意，手续费系统收 20%」：
                 现在是**一步成交**（扣钻 → 城市立刻返还 → 占领方实收 80%），所以主按钮改叫 [赎回]。
                 `ct.ransoming` 只剩「老版本遗留的待处理申请」会为真（新流程不再产生），
                 那时仍显示 [撤赎城] 让玩家把押金撤回来。 -->
            <template v-else-if="ct.id !== ezfy.city.id && ct.occupied">
              <a v-if="!ct.ransoming" href="javascript:;" @click="ezfy.doRansom(ct)">[赎回]</a>
              <a v-else class="red" href="javascript:;" @click="ezfy.doRansomCancel(ct)">[撤赎城]</a>
              <!-- ★ 2026-10-28 用户需求「28 小时 [赎城] 加一个 [弃城]，可以自己弃城」：
                   占领保护期（默认 28 小时，二战系统配置可调）**满了**才显示 [弃城]
                   （后端下发 can_abandon_occupy）；期内只有 [赎回]，不能弃。
                   弃城 = 和普通弃城同口径：物理删除城市，坐标恢复原地形。 -->
              <a v-if="ct.can_abandon_occupy" class="red" href="javascript:;" @click="ezfy.doAbandonOccupied(ct)">[弃城]</a>
            </template>
          </div>

          <br/>
          <!-- ★ 2026-10-09 用户反馈「起新城现在是仅黄金」：文案原来写死「消耗10万黄金」，
               而后端早已改成**五项资源各 10 万**（粮食/钢铁/石油/稀矿/黄金），且坐标必须是自己的附属野地。 -->
          <div class="panel-title">起新城 (资源各10万)</div>
          <div class="old-line gray">
            军衔「{{ ezfy.rankData.mine ? ezfy.rankData.mine.rank_name : ezfy.rankName }}」可建
            <b>{{ ezfy.rankData.mine ? ezfy.rankData.mine.city_max : '-' }}</b> 座，
            已有 <b>{{ ezfy.cities.length }}</b> 座
            <span v-if="ezfy.rankData.mine && ezfy.cities.length >= ezfy.rankData.mine.city_max" class="red">
              —— 已达上限，提升声望可解锁更多
            </span>
          </div>
          <div class="old-line">
            坐标X: <input v-model="ezfy.newCityX" type="number" style="width:70px"/>
            坐标Y: <input v-model="ezfy.newCityY" type="number" style="width:70px"/>
            <a href="javascript:;" @click="ezfy.doCreateCity">[新建]</a>
          </div>
          <!-- ★ 2026-09-25 去掉内联 font-size:14px，改为继承全站统一字号（--fs） -->
          <!-- ★ 2026-10-24 文案简化：内陆城市→陆城；沿海城市→海城 -->
          <div class="gray">
            <b>平原</b> → 陆城; <b>海平</b> → 海城<br/>
            <!-- 沿海城市(可建航海协会、训练海军)。其他地形(含海洋)不能建城; 新城自带基础建筑(市政厅/民居/农田1级), 建造后可在上方列表切换操作。 -->
          </div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'citystatus'">
        <div class="panel">
          <div class="panel-title">城市状态</div>
          城市: {{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }}) {{ ezfy.continent }}
          <!-- ★ 类型文案取 /view 顶层下发的 city_kind（原来读 city.city_kind，而嵌套的 city 对象里没有这个字段，
               于是这里永远显示「陆地城市」，和城市列表的「海城」对不上 —— 用户反馈的 bug） -->
          <span :class="ezfy.cityIsSea ? 'green' : 'gray'">[{{ ezfy.cityKindLabel }}]</span><br/>
          市政厅: {{ ezfy.city.city_level }}级<br/>
          <!-- ★ 2026-10-25 用户要求：本页显示当前城市的市长/城守是哪个军官（数据见 loadOfficerPosts），
               且**分两行**显示（原来挤在一行太长）。 -->
          市长: <span :class="ezfy.cityMayor ? '' : 'gray'">{{ ezfy.cityMayor || '未任命' }}</span><br/>
          城守: <span :class="ezfy.cityGuard ? '' : 'gray'">{{ ezfy.cityGuard || '未任命' }}</span><br/>
          <!-- ★ 2026-10-22 人口上限只由民居容量决定；★ 2026-10-23 超限人口在下一次懒结算直接夹平。 -->
          <!-- ★ 2026-10-25 用户要求压缩成一行：人口: 559354/553756 上限: 56万 -->
          人口: {{ ezfy.city.pop }}/{{ ezfy.freePop }} 上限: {{ ezfy.popCap > 0 ? ezfy.fmtBig(ezfy.popCap) : '无' }}<br/>
          民心/民怨: {{ ezfy.city.feelings }}/{{ ezfy.city.grievance }} 税率: {{ ezfy.city.tax_rate }}%<br/>
          <!-- ★ 2026-10-25 用户要求简写：军事+资源区 → 军+资 -->
          建筑: {{ ezfy.buildings.length }}座 (军+资 {{ ezfy.areaCount }}/{{ ezfy.areaCap }})<br/>
          <!-- ★ 2026-10-25 用户要求简写：城外N支队伍行进 → 城外N支 -->
          军队: {{ ezfy.totalTroops }} (城外{{ ezfy.marching }}支, {{ ezfy.occupying }}支驻守)<br/>
          附属野地: {{ ezfy.wildlands.length }}/{{ ezfy.city.city_level }}<br/>
          训练队列: {{ ezfy.queues.length }} | 未读军情: {{ ezfy.unreadReports }}<br/>
          <!-- ★ 2026-10-15 免战状态 + 时效（剩余时间每秒本地重算，见 Ezfy.vue::peaceText） -->
          <span v-if="ezfy.peaceText" class="green">{{ ezfy.peaceText }}</span>
          <span v-if="ezfy.boostUntil" class="orange">[增产中]</span>
          <br/>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[市政厅]</a>
          <a href="javascript:;" @click="ezfy.go('wareset')">[仓库调配]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'convene'">
        <div class="panel">
          <div class="panel-title">召集人口</div>
          <!-- ★ 2026-10-22 人口上限只由民居容量决定：
               人口未满 → 召集加人口；人口已满 → 召集只加临时空闲（总人口不变）。
               ★ 2026-10-23 临时空闲可以暂时超过总人口，之后自然回落。 -->
          当前人口: {{ ezfy.city.pop }}（空闲 {{ ezfy.freePop }}）<br/>
          民居上限: {{ ezfy.popCap > 0 ? ezfy.fmtBig(ezfy.popCap) : '无' }}<br/>
          <span v-if="ezfy.popAtHouseCap" class="gray">人口已满，召集只加临时空闲</span><br v-if="ezfy.popAtHouseCap"/>
          <span v-if="ezfy.popExtra > 0" class="orange">临时空闲 +{{ ezfy.fmtBig(ezfy.popExtra) }}</span><br v-if="ezfy.popExtra > 0"/>
          <span v-if="ezfy.popRecover > 0" class="gray">待恢复人口: {{ ezfy.fmtBig(ezfy.popRecover) }}（召集可恢复）</span><br v-if="ezfy.popRecover > 0"/>
          <!-- 花费粮食读管理端配置；召集人口 = 当前民居上限 × 配置百分比（默认 10%），勿再写死固定人数 -->
          花费 {{ ezfy.fmtBig(ezfy.conveneFoodCost) }}{{ ezfy.resNames.food }} 召集 {{ ezfy.fmtBig(ezfy.convenePopGain) }}{{ ezfy.popAtHouseCap ? '空闲人口' : '人口' }}<br/>
          <div class="old-line">{{ ezfy.resNames.food }}: {{ ezfy.city.food }}</div>
          <!-- ★ 2026-10-09 「去掉按钮样式」：改成与全站一致的 [链接] 写法
               （原来用 <button>，样式和别处不统一）；卡控由后端 + doConvene 双保险。 -->
          <a v-if="!ezfy.conveneBlocked" href="javascript:;" @click="ezfy.doConvene">[召集]</a>
          <span v-else class="gray">[召集]</span>
          <span v-if="ezfy.conveneBlocked" class="gray"> 暂无可用的民居容量</span>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'wareset'">
        <div class="panel">
          <div class="panel-title">仓库({{ ezfy.ware.level }}级)</div>
          <div class="old-line">
            保护总量:{{ ezfy.ware.total }}
            <span class="gray">(保护额度内的资源不会被敌人抢夺走)</span>
          </div>
          <div class="old-line" v-if="ezfy.ware.level < 1">
            <span class="red">尚未建造仓库, 无法保护资源</span>
            <a href="javascript:;" @click="ezfy.go('builds')">[前往资源区建造]</a>
          </div>
          <div class="old-line" v-else-if="ezfy.ware.level < 10">
            升级仓库可提升保护量, 下一级:{{ ezfy.ware.next_total }}
            <a href="javascript:;" @click="ezfy.go('builds')">[前往资源区]</a>
          </div>
          <div class="old-line" v-else>仓库已满级(保护量{{ ezfy.ware.total }})</div>
          <div class="panel-title">调配保护比例(四项合计不超过100%)</div>
          <div class="old-line" v-for="r in ezfy.ware.res" :key="'wr' + r.key">
            {{ r.name }}: 保护{{ r.protect }} / 现有{{ r.have }}
            <input v-model="ezfy.wareRatio[r.key]" type="number" min="0" max="100" style="width:60px"/>%
          </div>
          <div class="old-line">
            当前合计:{{ ezfy.wareSum }}%
            <span v-if="ezfy.wareSum > 100" class="red">(超出100%, 无法保存)</span>
            <span v-else class="gray">(未分配部分不产生保护)</span>
          </div>
          <div class="old-line">
            <button @click="ezfy.doWareSet" :disabled="ezfy.wareSum > 100">[保存比例]</button>
          </div>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[返回市政厅]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'placate'">
        <div class="panel">
          <div class="panel-title">安抚民心</div>
          当前民心: {{ ezfy.city.feelings }} / 民怨: {{ ezfy.city.grievance }}<br/>
          <div class="old-line">{{ ezfy.resNames.gold }}: {{ ezfy.city.gold }} | 安抚花费: {{ ezfy.placate.gold }}</div>
          <!-- ★ 冷却用 placateNow 每秒本地重算，不靠重新拉接口（沿用全站倒计时同一套做法） -->
          <!-- ★ 2026-10-25 用户要求「安抚也去掉按钮样式」：与同行的 [返回]/[返回首页] 统一成纯文字链接
               （button 有边框+底色，和文字链接并排很跳）。冷却时用灰色文字，仍不可点。 -->
          <template v-if="ezfy.placateCdLeft > 0">
            <span class="gray">冷却中，还需 {{ ezfy.durText(ezfy.placateCdLeft / 1000) }}</span><br/>
            <span class="gray">[安抚]</span>
          </template>
          <a v-else href="javascript:;" @click="ezfy.doPlacate">[安抚]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'taxset'">
        <div class="panel">
          <div class="panel-title">税率设置</div>
          当前税率: {{ ezfy.city.tax_rate }}% / 民心: {{ ezfy.city.feelings }}<br/>
          <div class="old-line">
            新税率: <input v-model="ezfy.taxInput" type="number" min="0" max="100" style="width:70px"/>%
            <button @click="ezfy.doTax">[设置]</button>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'rename'">
        <div class="panel">
          <div class="panel-title">城市改名</div>
          <div class="old-line">
            新城市名: <input v-model="ezfy.renameInput" style="width:60%"/>
            <button @click="ezfy.doRename">[确定]</button>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'cityhall'">
        <div class="panel">
          <div class="panel-title">市政厅({{ ezfy.city.city_level }}级)</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('taxset')">[税率设置]</a>
            <a href="javascript:;" @click="ezfy.go('sourceset')">[调整生产]</a>
            <a href="javascript:;" @click="ezfy.go('citymove')">[地图搬迁]</a>
            <a href="javascript:;" @click="ezfy.go('rename')">[城市更名]</a>
            <a href="javascript:;" @click="ezfy.go('cities')">[城市列表/迁建]</a>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('citystatus')">[城市状态]</a>
            <a href="javascript:;" @click="ezfy.go('wilds')">[占领野地]</a>
            <a href="javascript:;" @click="ezfy.go('wareset')">[仓库调配]</a>
            <a href="javascript:;" @click="ezfy.go('srcstat')">[资源统计]</a>
            <a href="javascript:;" @click="ezfy.go('troopstat')">[军队统计]</a>
          </div>
          <div class="panel-title">全部建筑总览</div>
          <table>
            <tr><th>建筑</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <template v-for="b in ezfy.buildings">
              <tr :key="'hb' + b.id">
                <td>{{ b.name }}</td>
                <td>{{ b.level }}/{{ b.max_level }}</td>
                <td>{{ b.status === 0 ? '空闲' : '施工中 ' + ezfy.remain(b.end_time, ezfy.gatherNow) }}</td>
                <td>
                  <a v-if="b.status === 0 && b.level > 0 && b.level < b.max_level" href="javascript:;" @click="ezfy.doUpgrade(b)">[升级]</a>
                  <!-- ★ 与建筑页同一口径：按背包里的建筑加速道具档位渲染 -->
                  <template v-if="b.status !== 0">
                    <a v-for="a in ezfy.accItems(3)" :key="'hsb' + b.id + '_' + a.cfg_id" href="javascript:;"
                       @click="ezfy.doSpeedBuilding(b, a)">[加速{{ ezfy.accLabel(a) }}]</a>
                    <span class="gray" v-if="!ezfy.accItems(3).length">(无加速道具)</span>
                    <!-- ★ 2026-09-27：施工中可取消（零退还，建筑保留当前等级） -->
                    <a href="javascript:;" class="red" @click="ezfy.doCancelUpgrade(b)">[取消]</a>
                  </template>
                </td>
              </tr>
              <tr v-if="ezfy.inlineTip && ezfy.inlineTip.bid === b.id" :key="'hbt' + b.id">
                <td colspan="4" class="build-tip" :class="ezfy.inlineTip.type">
                  <span>{{ ezfy.inlineTip.text }}</span>
                  <a href="javascript:;" @click="ezfy.inlineTip = null">[关闭]</a>
                </td>
              </tr>
            </template>
          </table>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'srcstat'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('cityhall')">市政厅</a>-&gt;资源统计
          </div>
          <div class="old-line">【{{ ezfy.city.name }}】:</div>
          <!-- ★ 2026-10-05 修复「玩家反馈离线资源不涨」的显示口径：
               斜杠后的分母从「仓库容量(city.xxx_cap)」改成**资源真正的收敛点**
               （全局资源最大值 res_max，线上 61 亿）。仓库容量按设计**不参与产量收敛**，
               原来那样显示会让玩家以为「还没满、怎么不涨」。到顶时明确标「已满」。 -->
          <div class="old-line">
            {{ ezfy.resNames.gold }}：{{ ezfy.city.gold }}/{{ ezfy.resMax.gold }}<span v-if="ezfy.isResFull('gold')">（已满）</span><br/>
            {{ ezfy.resNames.food }}：{{ ezfy.city.food }}/{{ ezfy.resMax.food }}<span v-if="ezfy.isResFull('food')">（已满）</span><br/>
            {{ ezfy.resNames.steel }}：{{ ezfy.city.steel }}/{{ ezfy.resMax.steel }}<span v-if="ezfy.isResFull('steel')">（已满）</span><br/>
            {{ ezfy.resNames.oil }}：{{ ezfy.city.oil }}/{{ ezfy.resMax.oil }}<span v-if="ezfy.isResFull('oil')">（已满）</span><br/>
            {{ ezfy.resNames.rare }}：{{ ezfy.city.rare }}/{{ ezfy.resMax.rare }}<span v-if="ezfy.isResFull('rare')">（已满）</span>
          </div>
          <div class="old-line gray">
            斜杠后为<b>资源最大值</b>：产量涨到这里就不再增加（标「已满」即已到顶，不是卡住）。<br/>
            仓库容量（{{ ezfy.city.gold_cap }} 等）只影响被掠夺时的保护量，不影响产量上限。
          </div>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[返回]</a>
          <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'troopstat'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('cityhall')">市政厅</a>-&gt;军队统计
          </div>
          <div class="old-line">【{{ ezfy.city.name }}】:</div>
          <div class="old-line" v-for="t in ezfy.troopsData.troops" :key="'ts' + t.troop_id">
            <a href="javascript:;" @click="ezfy.openTroopView(t.troop_id)">{{ t.name }}</a>：{{ t.count }}
          </div>
          <div class="old-line gray" v-if="!ezfy.troopsData.troops.length">(城内无部队)</div>
          <div class="old-line">合计：{{ ezfy.totalTroops }}</div>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[返回]</a>
          <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'citymove'">
        <div class="panel">
          <div class="panel-title">市政厅 → 城市迁移</div>
          <div class="old-line">当前城市：{{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }})　所属洲：{{ ezfy.moveInfo.city ? ezfy.moveInfo.city.continent : '—' }}</div>
          <div class="old-line gray">
            迁城需要消耗对应道具，道具可在【商城】用黄金或钻石购买（功能相同）。
          </div>
          <hr/>
          <div class="old-line">
            使用【迁城计划】：持有 {{ ezfy.moveItemCounts.low }} 个
            <span class="gray">(仅陆地城市可用，迁移到所选大洲内未被占领的平原)</span>
          </div>
          <div class="old-line">
            迁入大洲：
            <select v-model="ezfy.moveContinent">
              <option v-for="a in ezfy.moveInfo.areas" :key="'ma' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="ezfy.doMoveCity('low')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【高级迁城计划】：持有 {{ ezfy.moveItemCounts.high }} 个
          </div>
          <div class="old-line gray">
            仅陆地城市可用，请确认您输入的坐标是未被占领的陆地平原，海平无法直接迁移城市
          </div>
          <div class="old-line">
            横坐标 x：<input v-model="ezfy.moveX" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="ezfy.moveY" type="number" style="width:80px"/>
            <button @click="ezfy.doMoveCity('high')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【沿海迁城计划】：持有 {{ ezfy.moveItemCounts.sea }} 个
            <span class="gray">(沿海城市专用，迁移到海平)</span>
          </div>
          <div class="old-line">
            迁入大洲：
            <select v-model="ezfy.moveContinentSea">
              <option v-for="a in ezfy.moveInfo.areas" :key="'ms' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="ezfy.doMoveCity('sea')">按大洲迁城</button>
          </div>
          <div class="old-line gray">或指定坐标（必须是未被占领的海平）：</div>
          <div class="old-line">
            横坐标 x：<input v-model="ezfy.moveX2" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="ezfy.moveY2" type="number" style="width:80px"/>
            <button @click="ezfy.doMoveCity('sea')">按坐标迁城</button>
          </div>
          <div class="old-line gray">迁城后附属野地不会随城迁移, 需要重新占领。</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('mall')">[去商城买迁城道具]</a>
            <a href="javascript:;" @click="ezfy.go('bag')">[打开背包]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'sourceset'">
        <div class="panel">
          <div class="panel-title">调整生产（开工率）</div>
          <div class="old-line gray">开工率影响该资源的实际产量：实际产量 = 基础产量 × 开工率 / 100；设为 0 表示停产，且该资源的资源建筑不再占用人口（空闲人口会相应增加）</div>
          <div class="old-line">
            {{ ezfy.resNames.food }}：
            <input v-model="ezfy.rateFood" type="number" min="0" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateFood = 0">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateFood = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.steel }}：
            <input v-model="ezfy.rateSteel" type="number" min="0" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateSteel = 0">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateSteel = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.oil }}：
            <input v-model="ezfy.rateOil" type="number" min="0" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateOil = 0">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateOil = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.rare }}：
            <input v-model="ezfy.rateRare" type="number" min="0" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateRare = 0">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateRare = 100">[最大]</a>
          </div>
          <div class="old-line" style="color:#0000cc">请检查输入数字是否合理（范围 0~100，0 = 停产且不占用人口）</div>
          <div class="old-line">
            <button @click="ezfy.doSaveRate">确认调整</button>
          </div>
          <a href="javascript:;" @click="ezfy.go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyCity',
  inject: ['ezfy']
}
</script>
