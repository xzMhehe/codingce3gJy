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
            <!-- ★ 2026-10-07 赎城：自己的城市被人占领后，不显示[运输][派遣][弃城]，
                 改为 [赎城]（花钻石赎回，需占领方同意）/ [撤赎城]（撤销待处理请求） -->
            <template v-else-if="ct.id !== ezfy.city.id && ct.occupied">
              <a v-if="!ct.ransoming" href="javascript:;" @click="ezfy.doRansom(ct)">[赎城]</a>
              <a v-else class="red" href="javascript:;" @click="ezfy.doRansomCancel(ct)">[撤赎城]</a>
            </template>
          </div>

          <br/>
          <div class="panel-title">起新城 (消耗10万{{ ezfy.resNames.gold }})</div>
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
            <button @click="ezfy.doCreateCity">建新城</button>
          </div>
          <!-- ★ 2026-09-25 去掉内联 font-size:14px，改为继承全站统一字号（--fs） -->
          <div class="gray">
            <b>平原</b> → 内陆城市; <b>沿海平原</b> → 沿海城市<br/>
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
          人口: {{ ezfy.city.pop }}/{{ ezfy.housePopLimitOn ? ezfy.city.pop_max : '不限' }} (空闲{{ ezfy.freePop }})<br/>
          民心/民怨: {{ ezfy.city.feelings }}/{{ ezfy.city.grievance }} 税率: {{ ezfy.city.tax_rate }}%<br/>
          建筑: {{ ezfy.buildings.length }}座 (军事+资源区 {{ ezfy.areaCount }}/{{ ezfy.areaCap }})<br/>
          军队: {{ ezfy.totalTroops }} (城外{{ ezfy.marching }}支队伍行进, {{ ezfy.occupying }}支驻守)<br/>
          附属野地: {{ ezfy.wildlands.length }}/{{ ezfy.city.city_level }}<br/>
          训练队列: {{ ezfy.queues.length }} | 未读军情: {{ ezfy.unreadReports }}<br/>
          <span v-if="ezfy.protectedUntil" class="green">[免战保护中]</span>
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
          当前人口: {{ ezfy.city.pop }} / 民居容纳: {{ ezfy.housePopLimitOn ? ezfy.city.pop_max : '不限' }}<br/>
          <!-- ★ 2026-09-26：全局硬性人口上限（管理端配置，0 表示不限），超过则禁止召集 -->
          <template v-if="ezfy.convenePopMax > 0">
            召集人口上限: {{ ezfy.fmtBig(ezfy.convenePopMax) }}<br/>
          </template>
          <!-- ★ 2026-09-26：提示文案随「民居容量限制 / 召集人口灵活配置」两个开关变化，
               花费粮食/获得人口都读管理端配置（默认各 10 万），勿再写死 -->
          <template v-if="!ezfy.housePopLimitOn">
            花费 {{ ezfy.fmtBig(ezfy.conveneFoodCost) }}{{ ezfy.resNames.food }} 召集 {{ ezfy.fmtBig(ezfy.convenePopGain) }}人口(民居容量限制已关闭, 人口无上限)<br/>
          </template>
          <template v-else-if="ezfy.conveneFlexibleOn">
            花费 {{ ezfy.fmtBig(ezfy.conveneFoodCost) }}{{ ezfy.resNames.food }} 召集 {{ ezfy.fmtBig(ezfy.convenePopGain) }}人口(不受民居容纳上限限制, 可突破上限)<br/>
          </template>
          <template v-else>
            花费 {{ ezfy.fmtBig(ezfy.conveneFoodCost) }}{{ ezfy.resNames.food }} 召集 {{ ezfy.fmtBig(ezfy.convenePopGain) }}人口(受民居容纳上限限制, 满员后无法召集)<br/>
          </template>
          <div class="old-line">{{ ezfy.resNames.food }}: {{ ezfy.city.food }}</div>
          <button @click="ezfy.doConvene" :disabled="ezfy.conveneBlocked">[召集]</button>
          <span v-if="ezfy.conveneBlocked" class="gray">已达人口上限, 无法召集</span>
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
          <template v-if="ezfy.placateCdLeft > 0">
            <span class="gray">冷却中，还需 {{ ezfy.durText(ezfy.placateCdLeft / 1000) }}</span><br/>
            <button class="gray" disabled>[安抚]</button>
          </template>
          <button v-else @click="ezfy.doPlacate">[安抚]</button>
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
          <div class="old-line gray">设置后民心将联动为 {{ 100 - (parseInt(ezfy.taxInput) || 0) }}</div>
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
            <span class="gray">(迁移到所选大洲内未被占领的平原)</span>
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
            请确认您输入的坐标是非被占领的平原，沿海平原无法直接迁移城市
          </div>
          <div class="old-line">
            横坐标 x：<input v-model="ezfy.moveX" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="ezfy.moveY" type="number" style="width:80px"/>
            <button @click="ezfy.doMoveCity('high')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【沿海迁城计划】：持有 {{ ezfy.moveItemCounts.sea }} 个
            <span class="gray">(沿海城市专用，迁移到沿海平原)</span>
          </div>
          <div class="old-line">
            迁入大洲：
            <select v-model="ezfy.moveContinentSea">
              <option v-for="a in ezfy.moveInfo.areas" :key="'ms' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="ezfy.doMoveCity('sea')">按大洲迁城</button>
          </div>
          <div class="old-line gray">或指定坐标（必须是未被占领的沿海平原）：</div>
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
          <div class="old-line gray">开工率影响该资源的实际产量：实际产量 = 基础产量 × 开工率 / 100</div>
          <div class="old-line">
            {{ ezfy.resNames.food }}：
            <input v-model="ezfy.rateFood" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateFood = 1">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateFood = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.steel }}：
            <input v-model="ezfy.rateSteel" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateSteel = 1">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateSteel = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.oil }}：
            <input v-model="ezfy.rateOil" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateOil = 1">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateOil = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ ezfy.resNames.rare }}：
            <input v-model="ezfy.rateRare" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="ezfy.rateRare = 1">[最小]</a>
            <a href="javascript:;" @click="ezfy.rateRare = 100">[最大]</a>
          </div>
          <div class="old-line" style="color:#0000cc">请检查输入数字是否合理（范围 1~100）</div>
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
