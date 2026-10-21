<template>
  <div>
    <template v-if="ezfy.cur === 'reports'">
        <div class="panel">
          <!-- 复刻 report/index.html: 军队动态 . 驻军 . 军情警讯 . 战斗报告 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.reportTab === 1 }" @click="ezfy.switchReportTab(1)">军队动态</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 2 }" @click="ezfy.switchReportTab(2)">驻军</a><span
              v-if="ezfy.dynStation.length" class="green">({{ ezfy.dynStation.length }})</span><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 3 }" @click="ezfy.switchReportTab(3)">军情警讯</a><span
              v-if="ezfy.reportCounts[1]" class="red">({{ ezfy.reportCounts[1] }})</span><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 4 }" @click="ezfy.switchReportTab(4)">战斗报告</a><span
              v-if="ezfy.reportCounts[2]" class="red">({{ ezfy.reportCounts[2] }})</span>
            <!-- ★ 2026-10-09 「没军团的不展示 军团战报 tab」：
                 有军团（myCorps 已知）或 /reports/counts 回了 has_corps=true 才显示。
                 reportHasCorps 为 null（还没拿到）时按 myCorps 判断，避免团员刷新时 tab 闪一下不见。 -->
            <span v-if="ezfy.myCorps || ezfy.reportHasCorps"><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 5 }" @click="ezfy.switchReportTab(5)">军团战报</a></span>
          </div>

          <!-- ===== 军队动态: 行进/战斗/返航中的部队(出征/侦查/掠夺/运输/增援等) ===== -->
          <template v-if="ezfy.reportTab === 1">
            <div class="old-line" v-for="o in ezfy.dynMarchPaged" :key="'dy' + o.id">
              命令：{{ o.type_name }} <a v-if="!o.is_defend" href="javascript:;" @click="ezfy.openOrder(o)">查看</a><br/>
              <!-- ★ 2026-09-28 同驻军：外出部队也标出「哪个城出来的」(敌军来袭的防守视角无此字段, 故 v-if) -->
              <span v-if="o.from_city">起点：{{ o.from_city }}({{ o.from_x }},{{ o.from_y }})<br/></span>
              目标：<span v-if="o.act_type" class="red">[{{ ezfy.actTag(o.act_type) }}]</span>{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})
              <span v-if="o.is_defend" class="red">(敌军来袭)</span><br/>
              状态：{{ o.status_name }}
              <template v-if="o.can_command">
                <a href="javascript:;" class="red" @click="ezfy.openBattle(o.id)">[指挥]</a>
                <span class="gray">第{{ o.battle_round || 1 }}/{{ o.battle_max }}回合</span>
              </template>
              <!-- ★ 2026-09-30 行军计谋：出征中(0)/返回中(2)只显示一个 [计谋]，
                   点击进入计谋页选择 神兵天降(去程减80%)/战略转移(回程减360分钟) -->
              <template v-if="o.status === 0 || o.status === 2">
                <a class="green" href="javascript:;" @click="ezfy.openScheme(o)">[计谋]</a>
              </template>
              <!-- ★ 2026-10-09 军队动态里也能直接取消（原来只有出征队列能取消，用户反馈找不到）；
                   等待中(6) 是用户明确要求的：目标被抢先攻打、排队中可自行撤兵。
                   守方视角的「敌军来袭」行不能取消（那不是自己的部队）。 -->
              <template v-if="!o.is_defend && (o.status === 0 || o.status === 1 || o.status === 6)">
                <a class="red" href="javascript:;" @click="ezfy.doRecall(o)">[取消]</a>
              </template>
              <br/>
              军官：{{ o.officer || '无' }}<br/>
              {{ o.time_label }}：{{ o._lt || o.time_text }}<br/>
              <!-- ★ 2026-10-21 采集产出也走 carry（待带回，可超负重上限，点[停止采集]/[一键收获]入城）；
                   「运输」在途物资同样显示在这里。 -->
              <span v-if="o.carry_total > 0" class="green">
                在途物资：{{ ezfy.fmtN(o.carry.food) }}粮/{{ ezfy.fmtN(o.carry.steel) }}钢/{{ ezfy.fmtN(o.carry.oil) }}油/{{ ezfy.fmtN(o.carry.rare) }}稀/{{ ezfy.fmtN(o.carry.gold) }}金
                （负重 {{ ezfy.fmtN(o.carry_total) }}/{{ ezfy.fmtN(o.carry_cap) }}）
              </span>
              <br/>
              --------------------
            </div>
            <div class="old-line" v-if="!ezfy.dynMarch.length">(当前没有在外的部队)</div>
            <!-- ★ 2026-10-08 分页条**始终显示**（只要有数据），默认每页 5 条；切城市重新查询后同样有分页 -->
            <div class="ezfy-pager" v-if="ezfy.dynMarch.length">
              <a href="javascript:;" :class="{ gray: ezfy.dynPageCur <= 1 }" @click="ezfy.sectionPagerGo('dyn', -1)">上一页</a>
              <span class="gray">第 {{ ezfy.dynPageCur }}/{{ ezfy.dynMarchTotalPages }} 页（共 {{ ezfy.dynMarch.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.dynPageCur >= ezfy.dynMarchTotalPages }" @click="ezfy.sectionPagerGo('dyn', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 驻军: 到达野地后常驻采集的部队(满一个采集周期结算一期) ===== -->
          <template v-else-if="ezfy.reportTab === 2">
            <div class="old-line">
              <!-- ★ 采集玩法说明：驻守空闲需点[采集]；满一个采集周期结算一期（资源+宝物）；
                   未满一期按真实时长连续折算（不整点取整）；装不下的部分也先存「待带回」，可超负重上限，
                   点[停止采集]/[一键收获]后一并进原出发城市；[召回]撤兵。 -->
              <a href="javascript:;" @click="ezfy.doCollectAll">[一键采集]</a>
              <a href="javascript:;" @click="ezfy.doHarvestAll">[一键收获]</a>
              <a href="javascript:;" @click="ezfy.doRecallAll">[一键召回]</a>
            </div>
            <div class="old-line" v-for="o in ezfy.dynStationPaged" :key="'st' + o.id">
              命令：{{ o.type_name }} <a href="javascript:;" @click="ezfy.openOrder(o)">查看</a><br/>
              <!-- ★ 2026-09-28 显示这支部队是「哪个城出来的」(辨识番号) -->
              <span v-if="o.from_city">起点：{{ o.from_city }}({{ o.from_x }},{{ o.from_y }})<br/></span>
              目标：{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})<br/>
              军官：{{ o.officer || '无' }}<br/>
              {{ o.time_label }}：{{ o._lt || (o._lg ? o._lg.timeText : o.time_text) }}
              <span v-if="o.status === 1 && !o.arrive_time"><a href="javascript:;" class="red" @click="ezfy.startCollect(o)">[采集]</a></span>
              <span v-else-if="o.status === 1 && o.arrive_time"><a href="javascript:;" class="red" @click="ezfy.stopCollect(o)">[停止]</a></span>
              <!-- ★ 2026-10-19 驻守部队（status=1 采集队 / status=3 增援驻军）都能单支召回。
                   增援到附属野地的部队是**纯驻军**（order_type=6）：只驻守、不采集，
                   敌军进攻时作为守军参战；玩家可在这里单支召回，或用顶部 [一键召回]。 -->
              <span v-if="o.status === 1 || o.status === 3"><a href="javascript:;" class="red" @click="ezfy.doRecall(o)">[召回]</a></span><br/>
              <!-- ★ 2026-09-28 采集中部队: 实时累加显示本期已采资源(每秒由 liveGather 重算)。
                   ★★ 2026-10-21 新规则：产出全部先存部队「待带回(carry)」（可超负重上限，不丢弃），
                   点[停止采集]/[一键收获]后一并进原出发城市。
                   ★★ 2026-10-09 修复「采集队列会整块消失（隐藏情况）」：
                     liveGather 在「客户端时钟略慢于服务端 / 缺 gather 字段」时返回 null → o._lg 为 undefined，
                     而原来这里直接读 o._lg.food → 渲染期 TypeError → **整个列表渲染失败被隐藏**。
                     现在把 `o._lg` 也放进条件，拿不到就退化成「统计中…」，不再崩。 -->
              <template v-if="o.status === 1 && o.arrive_time && o._lg">
                <!-- ★ 2026-10-11 ：采集数量超过万/亿就按「万/亿」展示，且**不四舍五入**
                     （与首页产量同一套 fmtProd：截断取整，展示值不会大于真实值）。
                     「本期已采」→「已采」。 -->
                <span class="green">已采：{{ ezfy.fmtProd(o._lg.food) }}粮/{{ ezfy.fmtProd(o._lg.steel) }}钢/{{ ezfy.fmtProd(o._lg.oil) }}油/{{ ezfy.fmtProd(o._lg.rare) }}稀/{{ ezfy.fmtProd(o._lg.gold) }}金</span>
                <span class="gray">（总 {{ ezfy.fmtProd(o._lg.total) }}，负重 {{ ezfy.fmtProd(o._lg.total) }}/{{ ezfy.fmtProd(o.carry_cap) }}）</span>
                <!-- ★★★ 2026-10-21 新规则：装不下的部分不再直接入库 —— 全部存「待带回」，可超负重上限，
                     点[停止采集]/[一键收获]后一并进原出发城市（总 可以大于负重上限）。 -->
              </template>
              <span v-else-if="o.status === 1 && o.arrive_time" class="gray">已采：统计中…</span>
              <span v-else-if="o.status === 1" class="gray">已采：暂无</span>
              <br/>
              --------------------
            </div>
            <div class="old-line" v-if="!ezfy.dynStation.length">(当前没有驻守的部队(采集/驻军))</div>
            <!-- ★ 2026-10-08 分页条**始终显示**（只要有数据），默认每页 5 条 -->
            <div class="ezfy-pager" v-if="ezfy.dynStation.length">
              <a href="javascript:;" :class="{ gray: ezfy.dynStationPageCur <= 1 }" @click="ezfy.sectionPagerGo('sta', -1)">上一页</a>
              <span class="gray">第 {{ ezfy.dynStationPageCur }}/{{ ezfy.dynStationTotalPages }} 页（共 {{ ezfy.dynStation.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.dynStationPageCur >= ezfy.dynStationTotalPages }" @click="ezfy.sectionPagerGo('sta', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 军情警讯: 别人打我 ===== -->
          <template v-else-if="ezfy.reportTab === 3">
            <!-- ★ 2026-10-06 去掉 [刷新]（切 tab / 进页面会自动 loadReports） -->
            <!-- ★ 2026-10-05 「没用的页面提示去掉」：情报等级公式/还能看到什么 不再展示（文案保留注释里） -->
            <!-- ★ 2026-10-06 军情警讯也能删除：一键删除全部军情警讯（doClearReports 传 category=1） -->
            <!-- ★ 2026-10-13 军情警讯**不受城市限制**：列表/徽标/一键删除都是「全部城市」口径 -->
            <div class="old-line">
              <a href="javascript:;" @click="ezfy.doClearReports">[一键删除]</a>
            </div>
            <div class="old-line" v-for="r in ezfy.repPaged" :key="'rw' + r.id">
              <a href="javascript:;" @click="ezfy.openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>
                <span v-if="ezfy.intelTag(r)" class="orange">[{{ ezfy.intelTag(r) }}] </span>{{ ezfy.intelTitle(r) }}</a>
              <span class="gray">({{ ezfy.fmtTime(r.created_at) }})</span>
            </div>
            <!-- ★ 2026-10-11 区分「加载中」与「真的没有」：切分区时列表先清空、接口回来前显示「加载中…」，
                 避免「战斗报告(1) 却显示暂无」这种误导。 -->
            <div class="old-line gray" v-if="!ezfy.reports.length && ezfy.repLoading">(加载中…)</div>
            <div class="old-line" v-else-if="!ezfy.reports.length">(暂无军情警讯)</div>
            <!-- ★ 2026-10-08 分页条**始终显示**（只要有数据），默认每页 5 条 -->
            <div class="ezfy-pager" v-if="ezfy.reports.length">
              <a href="javascript:;" :class="{ gray: ezfy.repPageCur <= 1 }" @click="ezfy.sectionPagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ ezfy.repPageCur }}/{{ ezfy.repTotalPages }} 页（共 {{ ezfy.reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.repPageCur >= ezfy.repTotalPages }" @click="ezfy.sectionPagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战斗报告: 我打别人 + 战报查询 ===== -->
          <!-- ★ 2026-10-16 战斗报告也**不受城市限制**：列表/徽标/一键删除都是「全部城市」口径
               （后端 Reports 已不按 city_id 过滤，见 Ezfy.vue loadReports / doClearReports）。 -->
          <template v-else>
            <div class="old-line">
              战报查询:
              <input v-model="ezfy.reportWord" placeholder="输入关键字" style="width:110px"
                     @keyup.enter="ezfy.loadReports"/>
              <a href="javascript:;" @click="ezfy.loadReports">[查询]</a>
              <!-- ★ 2026-09-26 查询右边加 [一键删除]（物理删除自己名下全部战报，节约服务器资源）
                   ★ 2026-09-30 军团战报是团员的战报，不能一键删除 -->
              <a v-if="ezfy.reportTab !== 5" href="javascript:;" @click="ezfy.doClearReports">[一键删除]</a>
              <a v-if="ezfy.reportWord" href="javascript:;" @click="ezfy.reportWord = ''; ezfy.loadReports()">[清空]</a>
            </div>
            <div class="old-line" v-for="r in ezfy.repPaged" :key="'rb' + r.id">
              <a href="javascript:;" @click="ezfy.openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>
                <span v-if="ezfy.reportTab === 5 && r.owner_name" class="blue">{{ r.owner_name }}：</span>
                <span class="orange">[{{ r.type_name }}]</span> {{ r.title }}</a>
              <span class="gray">({{ ezfy.fmtTime(r.created_at) }})</span>
            </div>
            <!-- ★ 2026-10-11 同上：加载中 vs 真的没有 -->
            <div class="old-line gray" v-if="!ezfy.reports.length && ezfy.repLoading">(加载中…)</div>
            <div class="old-line" v-else-if="!ezfy.reports.length">{{ ezfy.reportTab === 5 ? '(暂无军团战报)' : '(暂无战斗报告)' }}</div>
            <!-- ★ 2026-10-08 分页条**始终显示**（只要有数据），默认每页 5 条 -->
            <div class="ezfy-pager" v-if="ezfy.reports.length">
              <a href="javascript:;" :class="{ gray: ezfy.repPageCur <= 1 }" @click="ezfy.sectionPagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ ezfy.repPageCur }}/{{ ezfy.repTotalPages }} 页（共 {{ ezfy.reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.repPageCur >= ezfy.repTotalPages }" @click="ezfy.sectionPagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战报详情（已改为独立页面 reportview，不再行内展开） ===== -->
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'reportview'">
        <div class="panel" v-if="ezfy.curReport">
          <!-- ★ 战报详情页也保留「军队动态 . 驻军 . 军情警讯 . 战斗报告」导航 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.reportTab === 1 }" @click="ezfy.goReportTab(1)">军队动态</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 2 }" @click="ezfy.goReportTab(2)">驻军</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 3 }" @click="ezfy.goReportTab(3)">军情警讯</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 4 }" @click="ezfy.goReportTab(4)">战斗报告</a>
            <!-- ★ 2026-10-09 战报详情页的导航同样带上「军团战报」（仅在有军团时），
                 否则从军团战报点进详情后这一栏没有高亮项。 -->
            <span v-if="ezfy.myCorps || ezfy.reportHasCorps"><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.reportTab === 5 }" @click="ezfy.goReportTab(5)">军团战报</a></span>
          </div>
          <div class="panel-title">{{ ezfy.curReport.title }}</div>
          <!-- ★ 2026-09-29 战报上色（自己绿、敌军红）：被掠夺/被征服报告里守方是自己 → 守方绿、攻方红 -->
          <div v-for="(seg, i) in ezfy.reportNiceLines(ezfy.curReport.content)" :key="'rc' + i" class="rpt-ln">
            <template v-if="seg.mode === 'pair'">
              <span :class="seg.left.cls">{{ seg.left.text }}</span><span :class="seg.right.cls">{{ seg.right.text }}</span>
            </template>
            <span v-else :class="seg.cls">{{ seg.text }}</span>
          </div>
          <template v-if="ezfy.curReport.detail">
            <div class="old-line"><a href="javascript:;" @click="ezfy.showDetail = !ezfy.showDetail">[展开/收起逐回合详情]</a></div>
            <div v-if="ezfy.showDetail" class="rpt-ln" v-for="(seg, j) in ezfy.reportNiceLines(ezfy.curReport.detail)" :key="'rd' + j">
              <template v-if="seg.mode === 'pair'">
                <span :class="seg.left.cls">{{ seg.left.text }}</span><span :class="seg.right.cls">{{ seg.right.text }}</span>
              </template>
              <span v-else :class="seg.cls">{{ seg.text }}</span>
            </div>
          </template>
          <div class="old-line">
            <!-- ★ 2026-10-09 别人的战报（军团战报里同团团员的）不能删：后端只允许删自己名下的，
                 点了必然「战报不存在」→ 干脆不显示 [删除]。 -->
            <a v-if="!ezfy.curReport.user_id || ezfy.curReport.user_id === ezfy.profile.user_id"
               href="javascript:;" class="red" @click="ezfy.delReport(ezfy.curReport)">[删除]</a>
            <a href="javascript:;" @click="ezfy.go('reports')">[返回]</a>
            <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
          </div>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyReport',
  inject: ['ezfy']
}
</script>
