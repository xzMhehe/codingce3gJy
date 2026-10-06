<template>
  <div>
    <template v-if="ezfy.cur === 'map'">
        <div class="panel">
          <!-- 复刻 map/index.html 游戏区: 城市行 → 地图：→ 坐标查找 → 精英城市 → 5×5 表格 → 当前中心 → 方向 -->
          <div class="old-line">
            {{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }})
            <a href="javascript:;" @click="ezfy.go('cities')">切换城市</a>
          </div>
          <div class="old-line">地图：</div>
          <div class="old-line">
            输入坐标查找：
            <a href="javascript:;" @click="ezfy.toggleStars">收藏列表</a>
          </div>
          <!-- ★ 「横坐标、纵坐标 一行」：两个输入框合并同一行，[查找] 跟在行末 -->
          <div class="old-line ezfy-map-jump">
            横坐标：<input v-model="ezfy.jumpX" type="number" placeholder="(1~500)"/>
            纵坐标：<input v-model="ezfy.jumpY" type="number" placeholder="(1~500)"/>
            <button @click="ezfy.doJump">[查找]</button>
          </div>
          <div class="old-line" v-if="ezfy.eliteCell">
            发现精英中立城市：
            <a class="red" href="javascript:;" @click="ezfy.openElite">[寇({{ ezfy.eliteCell.x }},{{ ezfy.eliteCell.y }})]</a>
          </div>
          <template v-if="ezfy.showStars">
            <div class="panel-title">收藏列表</div>
            <div class="old-line" v-for="s in ezfy.mapStars" :key="'st' + s.id">
              <a href="javascript:;" @click="ezfy.jumpTo(s.x, s.y)">{{ s.name }}({{ s.x }},{{ s.y }})</a>
              <a href="javascript:;" @click="ezfy.delStar(s)">[删除]</a>
            </div>
            <div class="old-line gray" v-if="!ezfy.mapStars.length">(收藏列表为空, 在地图上选中目标后可收藏)</div>
          </template>
          <!-- ★ 「格子下面加个坐标，排列整齐一点」：
               每格两行 —— 第一行名称(等级)，第二行 (x,y)；
               第二行用站内链接蓝 #0645ad，让玩家一眼知道格子能点。 -->
          <table class="ezfy-map-table">
            <tr v-for="(row, ri) in ezfy.mapRows" :key="'mr' + ri">
              <td v-for="cell in row" :key="cell.x + '_' + cell.y">
                <!-- ★ 2026-10-01 修复「地图输入 1,1 跳转后还有负号坐标」：
                     地图世界 500×500，有效坐标 1~499；视野中心靠边时周边格子会越界，
                     这些不存在的格子不再显示负坐标地形，统一按「空地」展示（不可点）。 -->
                <a v-if="!ezfy.isMapOOB(cell)" href="javascript:;" :class="ezfy.cellClass(cell)" :title="ezfy.cellTip(cell)" @click="ezfy.openCell(cell)">
                  <span class="ezfy-cell-name">{{ ezfy.cellText(cell) }}</span>
                  <span class="ezfy-cell-xy">({{ cell.x }},{{ cell.y }})</span>
                </a>
                <span v-else class="ezfy-empty">{{ ezfy.cellText(cell) }}</span>
              </td>
            </tr>
          </table>
          <div class="old-line">当前坐标中心:({{ ezfy.mapCx }} , {{ ezfy.mapCy }})</div>
          <!-- ★ 「向上/向右/向下/向左/回到本城 间隙稍微大一点」→ 见 .ezfy-dir-nav a -->
          <!-- ★ 2026-10-05 「向右、向左 调换下位置（功能不变）用着不习惯」
               → 顺序改为 向上 / 向左 / 向下 / 向右（各自 @click 的方向不变，只换摆放位置）。 -->
          <div class="old-line ezfy-dir-nav">
            <a href="javascript:;" @click="ezfy.moveMap(-ezfy.mapStep, 0)">向上</a>
            <a href="javascript:;" @click="ezfy.moveMap(0, -ezfy.mapStep)">向左</a>
            <a href="javascript:;" @click="ezfy.moveMap(ezfy.mapStep, 0)">向下</a>
            <a href="javascript:;" @click="ezfy.moveMap(0, ezfy.mapStep)">向右</a>
            <a href="javascript:;" @click="ezfy.loadMap()">回到本城</a>
          </div>
          <div class="old-line gray">
            城=城市 寇=寇城 墟=废墟 海=海洋 括号内为等级; 点格子进入目标详情
          </div>
          <div class="old-line gray">
            <span class="orange">活动</span>=活动野地 <span style="color:#ff00ff">活动寇</span>=活动寇城
            <span class="red">特殊</span>=特殊城市
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('orders')">出征队列</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'wildview'">
        <div class="panel" v-if="ezfy.selCell">
          <div class="old-line" v-if="ezfy.selDetail">
            所属区域：{{ ezfy.selDetail.continent }}
            <a href="javascript:;" @click="ezfy.addStar">{{ ezfy.isCellStarred ? '已收藏' : '收藏' }}</a>
          </div>
          <template v-if="ezfy.selDetail">
            <!-- 活动目标(活动野地/活动寇城/特殊城市): 复刻 activityIndex.html 的说明 + 守军/奖励预览 -->
            <template v-if="ezfy.selDetail.act_type">
              <div class="old-line orange">
                {{ ezfy.selDetail.act_name }}{{ ezfy.selDetail.act_level }}级 ({{ ezfy.selCell.x }},{{ ezfy.selCell.y }}) —— {{ ezfy.selDetail.act_desc }}
              </div>
              <div class="old-line" v-if="ezfy.selDetail.officer_name">
                守将：<span :class="ezfy.selDetail.officer_kind === 2 ? 'orange' : ''"><b>{{ ezfy.selDetail.officer_name }}</b></span>
                <span v-if="ezfy.selDetail.officer_kind === 2" class="orange">（名将</span>
                <span v-else class="gray">（普通军官</span>{{ ezfy.selDetail.officer_star }}★，可俘虏）
              </div>
              <div class="old-line">
                胜利奖励：{{ ezfy.resShort.food }}/{{ ezfy.resShort.steel }}/{{ ezfy.resShort.oil }}/{{ ezfy.resNames.rare }} 各{{ ezfy.selDetail.res_min }}<!--
                ★ 2026-09-25：管理端「野地获取资源倍率」>1 时标出来，让玩家知道为什么比平时多
                （后端已把倍率乘进 res_min，这里只是加个说明；=1 时不显示，不占版面） -->
                <span v-if="ezfy.selDetail.res_mult && ezfy.selDetail.res_mult > 1" class="green">（资源倍率×{{ ezfy.selDetail.res_mult }}）</span>，{{ ezfy.resNames.gold }}{{ ezfy.selDetail.gold }}，
                军功声望+{{ ezfy.selDetail.prestige }}，必定掉落宝物
              </div>
              <div class="old-line" v-if="ezfy.selDetail.jewel">采集可获得：{{ ezfy.selDetail.jewel }}</div>
              <div class="old-line red">活动目标无法占领，战胜只结算奖励(不占附属野地上限)</div>
            </template>
            <!-- 纯海洋: 无野地/守军/出征按钮 -->
            <template v-else-if="ezfy.selDetail.is_ocean">
              <div class="old-line">【海洋】({{ ezfy.selCell.x }},{{ ezfy.selCell.y }})</div>
              <div class="old-line">地形：海洋</div>
            </template>
            <!-- 陆地野地/海底森林/寇城 -->
            <template v-else>
              <div class="old-line">【{{ ezfy.selDetail.type === 3 ? '寇城' : ezfy.selDetail.terrain_name }}({{ ezfy.selDetail.level }}级)】({{ ezfy.selCell.x }},{{ ezfy.selCell.y }})</div>
              <div class="old-line">地形：{{ ezfy.selDetail.terrain_name }}</div>
              <div class="old-line" v-if="ezfy.selDetail.type === 3">地块：寇城</div>
              <div class="old-line" v-else>野地等级：{{ ezfy.selDetail.level }}级</div>
              <div class="old-line" v-if="ezfy.selDetail.type === 3">
                掉落宝物：{{ ezfy.selDetail.treasure || '普通宝物' }}
              </div>
              <div class="old-line" v-else>归属：{{ ezfy.selDetail.owner || '中立' }}</div>
              <div class="old-line" v-if="ezfy.selDetail.gather_res">
                <!-- ★ 2026-09-28 用户反馈：平原/沿海平原(特殊平原,可建航海协会)采集只有粮食、可建城市，不再显示「可能获得宝物」 -->
                <template v-if="ezfy.selDetail.terrain === 1 || ezfy.selDetail.terrain === 9">采集可以获得{{ ezfy.selDetail.gather_res }}，可建立城市。</template>
                <template v-else>采集可以获得{{ ezfy.selDetail.gather_res }}，可能获得{{ (ezfy.selDetail.treasures || []).join('、') }}。</template>
              </div>
            </template>
          </template>
          <div class="old-line" v-else>
            {{ ezfy.selCell.name }}({{ ezfy.selCell.x }},{{ ezfy.selCell.y }})
            <!-- ★ 2026-09-29 玩家城市也能收藏：玩家城不开 selDetail，原来的[收藏]被 selDetail 条件挡住了 -->
            <a href="javascript:;" style="margin-left:8px" @click="ezfy.addStar">{{ ezfy.isCellStarred ? '已收藏' : '收藏' }}</a>
          </div>
          <div class="old-line" v-if="!ezfy.selDetail && ezfy.selCell.owner">城主: {{ ezfy.selCell.owner }}</div>
          <!-- ★ 玩家城：展示城主的同盟（军团）名 —— 没加入军团显示「无」 -->
          <div class="old-line" v-if="ezfy.selCell.area_type === 3">
            同盟: 
            <b :class="ezfy.selCell.corps_name ? (ezfy.selCell.ally ? 'green' : '') : 'gray'">
              {{ ezfy.selCell.corps_name || '无' }}
            </b>
            <span v-if="ezfy.selCell.ally" class="green">（你的同盟成员）</span>
          </div>
          <hr/>
          <!-- ★ 按钮文案统一加方括号（「侦查 掠夺 征服 也加上 []」），
               与已有的 [宣战]/[返回地图]/[查找] 保持同一种「按钮」写法。
               同一行里的 运输/增援/采集 同属动作按钮，一并统一，免得一行里两种写法。 -->
          <!-- ① 本城：不给侦查/掠夺/征服（自己的城市不能打自己），只提示一句 -->
          <div class="old-line" v-if="ezfy.selCell.area_type === 3 && ezfy.selCell.mine">
            <a href="javascript:;" @click="ezfy.go('citystatus')">[城市状态]</a>
          </div>
          <!-- ② 别人的城：侦查/掠夺/征服 常显；掠夺/征服 需宣战生效(status=2)才可点，
               未宣战/待生效时置灰并提示，宣战入口只在没宣战(status=0)时出现。
               ★ 管理端「宣战功能」关掉时（warRequire=false）不需要宣战 → 掠夺/征服直接可点、
                 不再出现 [宣战] 入口，也不再显示「未宣战」状态文案。 -->
          <div class="old-line" v-else-if="ezfy.selCell.area_type === 3">
            <a href="javascript:;" @click="ezfy.pickOrder(1)">[侦查]</a>&nbsp;
            <!-- ★ 2026-09-25 军团交战期（atWar）无需个人宣战即可掠夺/征服 -->
            <a v-if="ezfy.warStatus === 2 || !ezfy.warRequire || ezfy.atWar" href="javascript:;" @click="ezfy.pickOrder(2)">[掠夺]</a>
            <a v-else href="javascript:;" class="gray" @click="ezfy.warBlock('掠夺')">[掠夺]</a>&nbsp;
            <a v-if="ezfy.warStatus === 2 || !ezfy.warRequire || ezfy.atWar" href="javascript:;" @click="ezfy.pickOrder(3)">[征服]</a>
            <a v-else href="javascript:;" class="gray" @click="ezfy.warBlock('征服')">[征服]</a>&nbsp;
            <!-- ★ 2026-09-25 军团交战期绿色提示（文案可用后端下发的 corps_war.text） -->
            <div class="old-line green" v-if="ezfy.corpsWar && ezfy.corpsWar.active">
              军团交战期：{{ ezfy.corpsWar.text || ('与【' + (ezfy.corpsWar.corps_name || '敌方军团') + '】处于交战状态，无需个人宣战即可掠夺/征服') }}
            </div>
            <!-- ★ 运输/增援 只对「同盟(同一军团)成员的城市」显示；宣战中一律不显示
                 （不需要宣战时「交战中」这个概念不成立，所以照常显示） -->
            <template v-if="ezfy.selCell.ally && (ezfy.warStatus !== 2 || !ezfy.warRequire)">
              <a href="javascript:;" @click="ezfy.pickOrder(5)">[运输]</a>&nbsp;
              <a href="javascript:;" @click="ezfy.pickOrder(6)">[增援]</a>&nbsp;
            </template>
            <!-- ★ 用户规则「同盟玩家不能宣战」→ 同盟成员不出现 [宣战] 入口，只给提示 -->
            <template v-if="ezfy.selCell.ally">
              <span class="green">同盟成员之间不能宣战</span>
            </template>
            <a v-else-if="ezfy.warStatus === 0 && ezfy.warRequire" href="javascript:;" @click="ezfy.declareWar">[宣战]</a>
            <!-- 同盟时不再叠「未宣战」这类状态文案，避免读成「不能宣战未宣战」 -->
            <span v-if="ezfy.warText && !ezfy.selCell.ally && ezfy.warRequire" class="orange">{{ ezfy.warText }}</span>
          </div>
          <!-- ③ 野地/寇城/海洋 -->
          <div class="old-line" v-else-if="ezfy.selCell.name !== '寇城(废墟)' && !(ezfy.selDetail && ezfy.selDetail.is_ocean)">
            <!-- ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服（要先[放弃]）→
                 三个命令灰掉、点了给提示（判据 isOwnWild 取后端下发的 selDetail.mine，
                 与「归属：我」同源，不靠前端猜）；[采集] 不受影响。 -->
            <a href="javascript:;" :class="{ gray: ezfy.isOwnWild }" @click="ezfy.pickOrder(1)">[侦查]</a><span class="home-gap"></span>
            <a href="javascript:;" :class="{ gray: ezfy.isOwnWild }" @click="ezfy.pickOrder(2)">[掠夺]</a><span class="home-gap"></span>
            <a href="javascript:;" :class="{ gray: ezfy.isOwnWild }" @click="ezfy.pickOrder(3)">[征服]</a><span class="home-gap"></span>
            <a v-if="ezfy.selCell.occupied && ezfy.isOwnWild" href="javascript:;" @click="ezfy.pickOrder(4)">[采集]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('map')">[返回地图]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="ezfy.go('map')">[前往地图]</a></div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'orderpre'">
        <div class="panel" v-if="ezfy.selCell">
          <div class="panel-title">出征确认 · {{ ezfy.orderNames[ezfy.orderType] }}</div>
          <!-- ★ 用户反馈「切换城市后感觉出征页还是切换前那个城」→
               出征页原来只写「目标」，看不出这支部队是从哪座城出发的。
               把**出发城市**显式写在最上面，玩家一眼就能确认用的是哪座城的兵/军官/资源。 -->
          <div class="old-line">
            出发城市：<b>{{ ezfy.city.name }}</b>
            <span v-if="ezfy.city.x || ezfy.city.y">({{ ezfy.city.x }},{{ ezfy.city.y }})</span>
          </div>
          <div class="old-line">
            目标：<b>{{ ezfy.selCell.name }}</b><span v-if="ezfy.selCell.level">({{ ezfy.selCell.level }}级)</span>
            ({{ ezfy.selCell.x }},{{ ezfy.selCell.y }})
          </div>
          <hr/>

          <!-- ★ 2026-09-28 出征页顺序重排：①指挥军官 → ②出征集结令 → ③选择兵力
               （军官放第一个 → 集结令 → 兵种；军团/属性用「军/学/后」简写、不展示忠诚）
               预设编队下拉：选中后回填 军官/集结令/兵力（兵力夹到可出征上限，军官不在当前城则回填空） -->
          <!-- 预设编队 -->
          <div class="of-sec">预设编队</div>
          <div class="old-line">
            <select v-model="ezfy.presetSel" @change="ezfy.applyPreset" style="width:200px">
              <option :value="0">不使用预设</option>
              <option v-for="p in ezfy.presets" :key="'ps' + p.id" :value="p.id">
                {{ p.name }}({{ ezfy.fmtN(p.troop_total) }}兵<template v-if="p.officer">·{{ p.officer }}</template>)
              </option>
            </select>
            <a href="javascript:;" @click="ezfy.go('hq'); $nextTick(() => ezfy.selectHqTab(4))">[管理预设]</a>
          </div>
          <!-- ① 军官 -->
          <div class="of-sec">① 指挥军官</div>
          <div class="old-line">
            <select v-model="ezfy.orderOfficer" @change="ezfy.doCalc">
              <option value="0">未指定</option>
              <option v-for="o in ezfy.onDutyOfficers" :key="'od' + o.id" :value="o.name">
                {{ o.name }}({{ o.level }}级) 军{{ o.military_total || o.military }}
                <span class="green" v-if="o.equip_military">(装+{{ o.equip_military }})</span>
                学{{ o.learning_total || o.learning }} 后{{ o.logistics_total || o.logistics }}
              </option>
            </select>
            <span v-if="ezfy.orderType === 7" class="red">(派遣必须选择)</span>
            <span v-else-if="ezfy.orderType === 6" class="gray">(增援后军官调任目标城市)</span>
            <span v-else-if="ezfy.orderType === 8" class="gray">(派遣后军官随军调往目标城市)</span>
            <span v-if="ezfy.curOfficerBonus" class="green"> 军官战斗加成: 攻击+{{ ezfy.curOfficerBonus }}%</span>
            <span v-if="!ezfy.onDutyOfficers.length" class="gray">
              (「{{ ezfy.city.name }}」暂无可带队军官<template v-if="ezfy.cityOfficers.length">：本城 {{ ezfy.cityOfficers.length }} 名军官都在出征中或为俘虏</template>；
              军官跟着城市走，别的城的军官不能在这里出征，可前往军校招募)
            </span>
          </div>

          <!-- ② 集结令 -->
          <div class="of-sec">② 出征集结令</div>
          <div class="old-line">
            使用
            <input type="number" min="0" :max="ezfy.gatherMax" v-model.number="ezfy.orderGather"
                   :disabled="ezfy.gatherCount <= 0" @change="ezfy.onGatherChange" style="width:80px"/>
            个
            <span class="gray">（背包里有 {{ ezfy.gatherCount }} 个）</span>
          </div>
          <div class="old-line" v-if="ezfy.attackTroops.length">
            <span :class="ezfy.orderOverCap ? 'red' : 'green'">
              本次出兵 <b>{{ ezfy.fmtN(ezfy.orderTroopTotal) }}</b> / 上限 <b>{{ ezfy.orderCapText }}</b>
              <template v-if="ezfy.orderOverCap">—— 超出上限，请减少兵力或加用集结令</template>
            </span>
          </div>

          <!-- ③ 兵力 -->
          <div class="of-sec">③ 选择兵力</div>
          <div class="of-rows">
            <div class="of-row" v-for="t in ezfy.trainCfgs" :key="'at' + t.id"
                 :class="{ 'of-off': ezfy.troopCount(t.id) <= 0 }"
                 :title="t.name + '（现有 ' + ezfy.fmtN(ezfy.troopCount(t.id)) + '，本次最多可派 ' + ezfy.fmtN(ezfy.orderQtyMax(t.id)) + '）'">
              <span class="of-name">{{ t.name }}</span>
              <span class="of-avail">现有 {{ ezfy.fmtN(ezfy.troopCount(t.id)) }}</span>
              <span class="of-ctl">
                <input type="range" class="of-range" min="0" step="1"
                       :max="ezfy.troopCount(t.id)" :value="ezfy.orderQty(t.id)"
                       :disabled="ezfy.orderQtyMax(t.id) <= 0"
                       @input="ezfy.onOrderQtyInput(t.id, $event)"/>
                <input type="number" class="of-num" min="0" placeholder="0"
                       :max="ezfy.orderQtyMax(t.id)" :value="ezfy.orderQty(t.id)"
                       :disabled="ezfy.orderQtyMax(t.id) <= 0"
                       @input="ezfy.onOrderQtyInput(t.id, $event)"/>
                <a href="javascript:;" class="of-max"
                   :class="{ 'of-max-off': ezfy.orderQtyMax(t.id) <= 0 }"
                   @click="ezfy.setOrderQtyMax(t.id)">[最大]</a>
              </span>
            </div>
          </div>
          <div class="old-line red" v-if="!ezfy.attackTroops.length">城内无可出征部队</div>

          <!-- ④ 随军资源 -->
           <!-- （右侧灰字是城内现有；上限 = 所带兵种负重之和 × 装载技术加成，没带部队时不能填） -->
          <div class="of-sec">④ 随军资源
            <!-- <span class="of-hint">（每行拖滑块或直接填数字；上限 = 所带兵种负重之和 × 装载技术加成；没选部队时禁用）</span> -->
          </div>
          <div class="of-rows">
            <div class="of-row" v-for="res in ezfy.resFields" :key="res.key"
                 :class="{ 'of-off': ezfy.orderResDisabled }"
                 :title="res.name + '（城内现有 ' + ezfy.fmtN(ezfy.resAvail(res.key)) + '，本次最多 ' + ezfy.fmtN(ezfy.resQtyMax(res.key)) + '）'">
              <span class="of-name">{{ res.name }}</span>
              <span class="of-avail">现有 {{ ezfy.fmtN(ezfy.resAvail(res.key)) }}</span>
              <span class="of-ctl">
                <input type="range" class="of-range" min="0" step="1"
                       :max="ezfy.resQtyMax(res.key)" :value="ezfy.resQty(res.key)"
                       :disabled="ezfy.orderResDisabled"
                       @input="ezfy.onResInput(res.key, $event)"/>
                <input type="number" class="of-num" min="0" placeholder="0"
                       :max="ezfy.resQtyMax(res.key)" :value="ezfy.resQty(res.key)"
                       :disabled="ezfy.orderResDisabled"
                       @input="ezfy.onResInput(res.key, $event)"/>
                <a href="javascript:;" class="of-max"
                   :class="{ 'of-max-off': ezfy.orderResDisabled || ezfy.resQtyMax(res.key) <= 0 }"
                   @click="ezfy.setResMax(res.key)">[最大]</a>
              </span>
            </div>
          </div>
          <div class="old-line" v-if="!ezfy.orderResDisabled">
            随军总量：<b :class="ezfy.orderResOver ? 'red' : 'green'">{{ ezfy.fmtN(ezfy.orderResTotal) }}</b>
            / 可用负重 <b>{{ ezfy.fmtN(ezfy.orderResUsableCap) }}</b>
            <span v-if="ezfy.orderResOver" class="red">
              —— 超出（<template v-if="ezfy.orderType !== 5">负重{{ ezfy.fmtN(ezfy.orderResCap) }} − 油耗{{ ezfy.fmtN(ezfy.oilUsed) }}</template><template v-else>负重{{ ezfy.fmtN(ezfy.orderResCap) }}</template>），请减少资源或多带部队
            </span>
            <span v-else class="gray">
              <template v-if="ezfy.orderType !== 5">（负重{{ ezfy.fmtN(ezfy.orderResCap) }} − 油耗{{ ezfy.fmtN(ezfy.oilUsed) }}，含装载技术加成）</template>
              <template v-else>（负重{{ ezfy.fmtN(ezfy.orderResCap) }}，运输油耗另扣不占负重；含装载技术加成）</template>
            </span>
          </div>
          <div class="old-line gray" v-if="ezfy.orderType === 5">
            运输：自己城市之间 / 同盟成员之间都能运；必须带部队来装货，能运多少看<b>负重</b>，一般用卡车；
            可以不带队军官；送完部队会返回出发城市。
          </div>
          <div class="old-line gray" v-else-if="ezfy.orderType === 7">派遣必须选择带队军官。</div>
          <div class="old-line gray" v-else-if="ezfy.orderType === 8">
            派遣：把自己的部队 / 军官 / 随军资源送到<b>自己的另一座城市</b>；必须带部队，
            能带多少资源看<b>负重</b>，军官会随军调往目标城市。
          </div>

          <!-- ⑤ 宿营 -->
           <!-- （抵达后停留，可选） -->
          <div class="of-sec">⑤ 宿营</div>
          <div class="old-line">
            <input v-model="ezfy.waitH" type="number" min="0" max="24" style="width:50px"/> 时
            <input v-model="ezfy.waitM" type="number" min="0" max="60" style="width:50px"/> 分
          </div>

          <!-- ⑥ 计算 / 出征 -->
          <div class="of-sec">⑥ 消耗预览</div>
          <div class="old-line">
            <button @click="ezfy.doCalc">[计算]</button>
            油耗：<span class="orange">{{ ezfy.orderCalc ? ezfy.orderCalc.oil_used : '—' }}</span>
            &nbsp;/&nbsp;负重：<span class="orange">{{ ezfy.orderCalc ? ezfy.orderCalc.carry : '—' }}</span>
            &nbsp;/&nbsp;耗时：<span class="orange">{{ ezfy.orderCalc ? ezfy.orderCalc.need_time : '—' }}</span>
          </div>
          <div class="old-line red" v-if="ezfy.orderCalc && !ezfy.orderCalc.oil_enough">
            {{ ezfy.resNames.oil }}不足：需要{{ ezfy.orderCalc.oil_used }}，当前只有{{ ezfy.orderCalc.oil_have }}
          </div>
          <hr/>
          <div class="old-line">
            <button @click="ezfy.doOrder()">[出征]</button>
            <a href="javascript:;" @click="ezfy.go('map')">[返回地图]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="ezfy.go('map')">[前往地图]</a></div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'battle'">
        <div class="panel-title">
          【战场指挥】{{ ezfy.battleData.target_name }}({{ ezfy.battleData.target_x }},{{ ezfy.battleData.target_y }})
        </div>
        <div class="panel">
          <div class="old-line">
            第 {{ ezfy.battleData.round }}/{{ ezfy.battleData.max_round }} 回合
            <span v-if="ezfy.battleData.done" class="red">（战斗已结束<span v-if="ezfy.battleData.draw"> · 40回合平局</span>）</span>
            <span v-else-if="ezfy.battleData.phase === 'cmd'" class="green">（指令期，可下达命令）</span>
            <span v-else class="red">（已锁定，等待结算）</span>
          </div>
          <!-- 本回合倒计时（最后 5 秒锁定：条变红 = 已锁定）
               ★ 规则说明一律不写进界面（），记在这里：
                 · 每回合 30 秒，前 25 秒（cmd_window_ms）可下达指令，后 5 秒锁定由服务器结算；
                 · 指令是**逐兵种**的（「自己带的兵种都能指挥，就是单独指挥」）；
                 · 没下指令的兵种按司令部「兵种战斗配置」行动；
                 · 兵种目标同样是**逐兵种**的：默认取司令部配置，指挥时可改（0 = 最近目标），
                   守方没有该兵种时服务器自动回落打最近的（2026-09-23 ）；
                 · [自动战斗] = 自己全部军队前进，一口气打完。 -->
          <div class="old-line" v-if="!ezfy.battleData.done">
            {{ ezfy.battleData.time_label || '本回合剩余' }}：<b>{{ ezfy.battleLeftText }}</b>
            <div class="ezfy-battle-bar">
              <i :class="ezfy.battleData.phase === 'cmd' ? 'on' : 'lock'" :style="{ width: ezfy.battleBarPct + '%' }"></i>
            </div>
          </div>
          <!-- ★ 2026-10-06 指挥全部**下拉即时发送**（非受控 select：不绑 value/v-model，点选即 @change 发送，
               永远不会被轮询弹回；服务器状态只更新 select 旁的文字提示）。
               恢复[全军前进/待命/后退]（troop_id=0 走老的全军统一键），保存配置已合入下拉直接生效；
               到点(30s)服务端自动结算，锁定概念保留给后端（对方锁定状态照常提示）。 -->
          <div class="old-line ezfy-battle-cmds" v-if="!ezfy.battleData.done">
            <a href="javascript:;" @click="ezfy.sendBattleCmd('advance', 0)">[全军前进]</a>
            <a href="javascript:;" @click="ezfy.sendBattleCmd('hold', 0)">[全军待命]</a>
            <a href="javascript:;" @click="ezfy.sendBattleCmd('retreat', 0)">[全军后退]</a>
            <a v-if="ezfy.battleData.can_auto && !ezfy.battleData.my_locked" href="javascript:;" @click="ezfy.doBattleAuto">[自动战斗]</a>
            <a v-if="ezfy.battleData.my_locked" href="javascript:;" class="red" @click="ezfy.cancelBattleConfig">[取消配置]</a>
            <span v-if="ezfy.battleData.my_locked" class="red">· 已锁定</span>
            <span v-else-if="ezfy.battleData.atk_locked && ezfy.battleData.def_locked" class="green">· 对方已锁定</span>
            <span v-else-if="ezfy.battleData.atk_locked || ezfy.battleData.def_locked" class="green">· 对方已锁定待结算</span>
          </div>
          <!-- 双方兵力 + 逐兵种指挥（指令 + 优先攻击目标） -->
          <table class="ezfy-plain-table ezfy-battle-tbl">
            <tr>
              <th>方</th><th class="nm">兵种</th><th>剩余</th><th>初始</th><th>位置</th>
              <th v-if="!ezfy.battleData.done">目标</th>
              <th v-if="!ezfy.battleData.done">指挥</th>
            </tr>
            <tr v-for="u in ezfy.battleData.attackers" :key="'ba' + u.troop_id"
                :class="ezfy.battleData.is_atk ? 'ezfy-row-self' : 'ezfy-row-enemy'">
              <td class="ezfy-side-lbl"><span :class="ezfy.battleData.is_atk ? 'green' : 'red'">攻</span></td>
              <td class="nm">{{ u.name }}</td>
              <td>{{ ezfy.fmtN(u.count) }}</td><td>{{ ezfy.fmtN(u.initial) }}</td><td>{{ u.pos }}</td>
              <!-- ★ 兵种目标（2026-09-23 ）：默认 = 司令部「兵种战斗配置」，
                   指挥时玩家可逐兵种改；0 = 最近目标（守方没有该兵种时服务器自动打最近的）。
                   ★ 守方视角(is_atk=false)时这里显示 AI，指挥控件渲染到守方行上。 -->
              <td v-if="!ezfy.battleData.done">
                <template v-if="ezfy.battleData.is_atk">
                  <!-- ★ 2026-10-06 v3 非受控下拉：不绑 value/v-model，select 永远可点选（除非已锁定），
                       选择即 @change 发送；服务器回包只更新旁边文字，选中项不被重置 -->
                  <select @change="ezfy.sendBattleTarget(u.troop_id, $event.target.value)" style="width:96px">
                    <option v-for="op in (ezfy.battleData.target_options || [])"
                            :key="'to' + u.troop_id + '_' + op.id" :value="op.id">{{ op.name }}</option>
                  </select>
                </template>
                <span v-else class="gray">-</span>
              </td>
              <!-- ★ 2026-10-06 逐兵种指令改为下拉（前进/后退/待命/默认）；my_locked 时禁用 -->
              <td v-if="!ezfy.battleData.done" class="ezfy-cmd">
                <template v-if="ezfy.battleData.is_atk">
                  <!-- 指令：非受控下拉，点选即发 -->
                  <select @change="ezfy.sendBattleCmd($event.target.value, u.troop_id)" style="width:96px">
                    <option value="advance">前进</option>
                    <option value="hold">待命</option>
                    <option value="retreat">后退</option>
                    <option value="">默认</option>
                  </select>
                </template>
                <span v-else class="gray">{{ ezfy.battleData.is_atk ? (ezfy.battleData.def_name || 'AI') : (ezfy.battleData.atk_name || 'AI') }}</span>
              </td>
            </tr>
            <tr v-for="u in ezfy.battleData.defenders" :key="'bd' + u.troop_id"
                :class="!ezfy.battleData.is_atk ? 'ezfy-row-self' : 'ezfy-row-enemy'">
              <td class="ezfy-side-lbl"><span :class="!ezfy.battleData.is_atk ? 'green' : 'red'">守</span></td>
              <td class="nm">{{ u.name }}</td>
              <td>{{ ezfy.fmtN(u.count) }}</td><td>{{ ezfy.fmtN(u.initial) }}</td><td>{{ u.pos }}</td>
              <td v-if="!ezfy.battleData.done">
                <template v-if="!ezfy.battleData.is_atk">
                  <!-- 目标：非受控下拉，点选即发 -->
                  <select @change="ezfy.sendBattleTarget(u.troop_id, $event.target.value)" style="width:96px">
                    <option v-for="op in (ezfy.battleData.target_options || [])"
                            :key="'to' + u.troop_id + '_' + op.id" :value="op.id">{{ op.name }}</option>
                  </select>
                </template>
                <span v-else class="gray">-</span>
              </td>
              <!-- ★ 2026-10-06 逐兵种指令改为下拉（前进/后退/待命/默认）；my_locked 时禁用 -->
              <td v-if="!ezfy.battleData.done" class="ezfy-cmd">
                <template v-if="!ezfy.battleData.is_atk">
                  <!-- 指令：非受控下拉，点选即发 -->
                  <select @change="ezfy.sendBattleCmd($event.target.value, u.troop_id)" style="width:96px">
                    <option value="advance">前进</option>
                    <option value="hold">待命</option>
                    <option value="retreat">后退</option>
                    <option value="">默认</option>
                  </select>
                </template>
                <span v-else class="gray">{{ ezfy.battleData.is_atk ? (ezfy.battleData.def_name || 'AI') : (ezfy.battleData.atk_name || 'AI') }}</span>
              </td>
            </tr>
          </table>
          <div class="old-line">
            战场态势：攻方 {{ ezfy.fmtN(ezfy.battleData.atk_total) }} · 守方 {{ ezfy.fmtN(ezfy.battleData.def_total) }}
          </div>
          <!-- ★ 2026-10-06 指挥室展示双方军官：后端战场简介(head)里已写入
               「【攻方军官】… / 【守方军官】…」，原来前端不渲染这组行 → 指挥时看不到军官。 -->
          <div class="old-line ezfy-battle-head" v-for="(hd, i) in (ezfy.battleData.head || [])" :key="'bh' + i">{{ hd }}</div>
          <!-- 行动日志：最新回合在最上、已过回合在下；我方绿色、敌军红色 -->
          <div class="old-line ezfy-battle-legend">
            <span class="green">■ 我方</span>&nbsp;<span class="red">■ 敌军</span>
          </div>
          <template v-if="ezfy.battleRounds.length">
            <div class="old-line ezfy-round-block" v-for="(g, gi) in ezfy.battleRounds" :key="'bg' + gi">
              <div v-if="g.line" class="ezfy-round-title">{{ g.line }}</div>
              <div v-for="(li, idx) in g.items" :key="'bl' + gi + '_' + idx"
                   class="ezfy-round-line" :class="ezfy.battleLineClass(li)">{{ li }}</div>
            </div>
          </template>
          <div class="old-line gray" v-else>(暂无行动)</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.leaveBattle">[返回军队动态]</a>
          </div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'orderview'">
        <div class="panel" v-if="ezfy.curOrder">
          <div class="panel-title">军队动态详情</div>
          出发地:{{ ezfy.curOrder.from_name }}<br/>
          目的地:{{ ezfy.curOrder.target_name }}({{ ezfy.curOrder.target_x }},{{ ezfy.curOrder.target_y }})<br/>
          命令:{{ ezfy.curOrder.type_name }}<br/>
          军官:{{ ezfy.curOrder.officer || '无(未带军官)' }}<br/>
          统帅:{{ ezfy.nick }}<br/>
          状态:{{ (ezfy.curOrder.order_type === 7 && ezfy.curOrder.status === 1) ? ezfy.orderStatusText(ezfy.curOrder) : (ezfy.curOrder.status_name || ezfy.orderStatusText(ezfy.curOrder)) }}<br/>
          出发时间:{{ ezfy.curOrder.start_text }}<br/>
          到达时间:{{ ezfy.curOrder.arrive_text }}<br/>
          <template v-if="ezfy.curOrder.return_text">返航时间:{{ ezfy.curOrder.return_text }}<br/></template>
          耗油:{{ ezfy.curOrder.oil_used }}<br/>
          <hr/>
          【进攻方军队】<br/>
          <span v-for="(t, i) in ezfy.curOrder.troops" :key="'ot' + i">{{ t.name }}:{{ t.count }}<br/></span>
          <span v-if="!ezfy.curOrder.troops.length" class="gray">(未携带部队)</span>
          <template v-if="ezfy.curOrder.resources && ezfy.curOrder.resources.length">
            <br/>军队携带资源:<br/>
            <span v-for="(r, i) in ezfy.curOrder.resources" :key="'or' + i">{{ r.name }}:{{ r.count }}<br/></span>
          </template>
          <hr/>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('hq')">[指挥(司令部)]</a>
            <a v-if="ezfy.curOrder.status === 0 || ezfy.curOrder.status === 1"
               class="red" href="javascript:;" @click="ezfy.doRecall(ezfy.curOrder)">[取消出征]</a>
            <a v-if="ezfy.curOrder.order_type === 7 && ezfy.curOrder.status === 1 && !ezfy.curOrder.arrive_time"
               class="red" href="javascript:;" @click="ezfy.startCollect(ezfy.curOrder)">[采集]</a>
            <a v-if="ezfy.curOrder.order_type === 7 && ezfy.curOrder.status === 1 && ezfy.curOrder.arrive_time"
               href="javascript:;" @click="ezfy.go('wilds')">[查看野地]</a>
            <a v-if="ezfy.curOrder.report_id" href="javascript:;" @click="ezfy.jumpReport(ezfy.curOrder.report_id)">[查看战报]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('orders')">[返回出征队列]</a>
        </div>
        <!-- ★ 2026-09-25：原来这里没有 v-else —— 刷新后 curOrder 丢了就整页空白（用户报「刷新页面消失」）。
             现在给兜底提示 + 回队列的入口，任何情况下都不会白屏。 -->
        <div class="panel" v-else>
          <div class="old-line">命令不存在或已结束</div>
          <div class="old-line gray">可能该命令已经完成/被取消，战报里仍可查到。</div>
          <a href="javascript:;" @click="ezfy.go('orders')">[返回出征队列]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'orders'">
        <div class="panel">
          <div class="panel-title">出征队列({{ ezfy.queueItems.length }})</div>
          <div class="old-line gray">
            包含行军中 / 战斗中 / 返航中 / 驻守采集的全部部队；驻守空闲的部队需点 [采集] 才开始采集。
          </div>
          <div class="old-line" v-for="o in ezfy.queueItems" :key="'oq' + o.id">
            命令：{{ o.type_name }} <a v-if="!o.is_defend" href="javascript:;" @click="ezfy.openOrder(o)">查看</a><br/>
            目标：<span v-if="o.act_type" class="red">[{{ ezfy.actTag(o.act_type) }}]</span>{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})
            <span v-if="o.is_defend" class="red">(敌军来袭)</span><br/>
            状态：{{ o.status_name }}
            <template v-if="o.can_command">
              <a href="javascript:;" class="red" @click="ezfy.openBattle(o.id)">[指挥]</a>
              <span class="gray">第{{ o.battle_round || 1 }}/{{ o.battle_max }}回合</span>
            </template>
            <template v-else-if="o.status === 1 && !o.arrive_time">
              <a href="javascript:;" class="red" @click="ezfy.startCollect(o)">[采集]</a>
            </template>
            <template v-else-if="o.status === 1 && o.arrive_time">
              <a href="javascript:;" class="red" @click="ezfy.stopCollect(o)">[停止]</a>
            </template>
            <!-- ★ 2026-09-30 行军计谋：出征中(0)/返回中(2)显示 [计谋]，进入计谋页选择神兵天降/战略转移 -->
            <template v-if="o.status === 0 || o.status === 2">
              <a class="green" href="javascript:;" @click="ezfy.openScheme(o)">[计谋]</a>
            </template>
            <br/>
            军官：{{ o.officer || '无' }}<br/>
            {{ o.time_label }}：{{ o._lt || (o._lg ? o._lg.timeText : o.time_text) }}<br/>
            <!-- ★ 2026-09-28 采集中部队: 实时累加显示本期已采资源(每秒由 liveGather 重算)。
                 规则已改为「收获即入起点城市」，故不再显示「需召回返航后入库」。 -->
            <template v-if="o.status === 1 && o.arrive_time">
              <span class="green">本期已采：{{ ezfy.fmtN(o._lg.food) }}粮/{{ ezfy.fmtN(o._lg.steel) }}钢/{{ ezfy.fmtN(o._lg.oil) }}油/{{ ezfy.fmtN(o._lg.rare) }}稀/{{ ezfy.fmtN(o._lg.gold) }}金</span><br/>
              <span class="gray">总 {{ ezfy.fmtN(o._lg.total) }}（负重 {{ ezfy.fmtN(o._lg.total) }}/{{ ezfy.fmtN(o.carry_cap) }}）</span>
              <!-- ★ 2026-10-05 去掉「负重已满, 超出部分会直接入库(可停止或收获)。」这行提示
                   —— 属于无用提示，玩家看负重条就够。规则仍然生效（见后端
                   processArrive 采集分支：超出负重部分直接入起点城市，不丢弃），只是不再在界面上提示。
                   对应字段 o._lg.full 仍在用（下面负重进度条按它变色），不要一起删。 -->
              <br/>
            </template>
            <br/>
            <span v-if="o.status === 0 || o.status === 1">
              <a href="javascript:;" class="red" @click="ezfy.doRecall(o)">[取消]</a><br/>
            </span>
            --------------------
          </div>
          <div class="old-line" v-if="!ezfy.queueItems.length">(暂无出征部队)</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.loadDynamics">[刷新]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'wilds'">
        <div class="panel">
          <div class="panel-title">占领野地({{ ezfy.wildlands.length }}/{{ ezfy.city.city_level }})</div>
          <table>
            <tr><th>坐标</th><th>地形</th><th>所属洲</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="w in ezfy.wildlands" :key="'wd' + w.id">
              <td><a href="javascript:;" @click="ezfy.openMapAt(w.x, w.y)" title="点击查看地图位置">({{ w.x }},{{ w.y }})</a></td>
              <td>{{ w.terrain === 8 ? '海底森林' : w.terrain_name }}</td>
              <td>{{ w.continent || '—' }}</td>
              <td>{{ w.level }}</td>
              <td>
                <template v-if="w.status === 1">采集中</template>
                <template v-else-if="w.idle_order_id">驻守(空闲)</template>
                <template v-else>空闲</template>
              </td>
              <td>
                <!-- ★ 2026-09-28 平原/沿海平原可建城、采集无宝物, 附属野地列表不再提供采集, 仅保留[放弃] -->
                <a v-if="w.status === 0 && !w.idle_order_id && w.terrain !== 1 && w.terrain !== 9" href="javascript:;" @click="ezfy.openWildGather(w)">[采集]</a>
                <a v-else-if="w.idle_order_id && w.terrain !== 1 && w.terrain !== 9" class="red" href="javascript:;" @click="ezfy.startCollect(w.idle_order_id)">[开始采集]</a>
                <!-- ★ 2026-09-28 用户反馈「采集中只能[放弃]，没法[停止]」：
                     后端现在会下发 gather_order_id（见 ezfy.go 的 wildViews），
                     这里据此构造一个最小的订单对象喂给 stopCollect（它只用到 id/target_* 拼提示文案）。 -->
                <a v-if="w.status === 1 && w.gather_order_id" class="red" href="javascript:;"
                   @click="ezfy.stopCollect(ezfy.wildOrderArg(w))">[停止]</a>
                <span v-if="w.status === 1" class="gray">采集中</span>
                <!-- ★ 2026-09-28 用户反馈「采集中 别展示 放弃按钮」：
                     采集中(status=1)时操作列只留 [停止]；[放弃] 会让整块野地连同采集部队一起处理掉，
                     应当先停止采集再放弃，故采集中隐藏。 -->
                <a v-if="w.status !== 1" class="red" href="javascript:;" @click="ezfy.doAbandon(w)">[放弃]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.wildlands.length">(尚未占领任何野地)</div>
          <br/>
          <template v-if="ezfy.occupies.length">
            <div class="panel-title">被占领城市({{ ezfy.occupies.length }})</div>
            <table>
              <tr><th>坐标</th><th>城市</th><th>原属</th><th>操作</th></tr>
              <tr v-for="o in ezfy.occupies" :key="'oc' + o.id">
                <td>({{ o.x }},{{ o.y }})</td>
                <td>{{ o.city_name }}</td>
                <td>{{ o.def_user }}</td>
                <td>
                  <a href="javascript:;" @click="ezfy.doOccupy('build', o)">[建立城市]</a>
                  <a class="red" href="javascript:;" @click="ezfy.doOccupy('destroy', o)">[摧毁]</a>
                  <a href="javascript:;" @click="ezfy.doOccupy('return', o)">[放弃归还]</a>
                </td>
              </tr>
            </table>
            <br/>
          </template>
          <a href="javascript:;" @click="ezfy.go('map')">[前往地图占领]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化：地图/野地/出征/战场/队列等页面模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyMap',
  inject: ['ezfy']
}
</script>
