<template>
  <div>
    <template v-if="ezfy.cur === 'rank'">
        <div class="panel">
          <!-- ★ 2026-09-24 军衔晋升表/军衔声望榜/兵力榜/军团榜做成 tab 分开展示 -->
          <!-- ★ 2026-10-14 tab 顺序：风云榜置首（用户要求） -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.rankTab === 'pvp' }" @click="ezfy.showPvpRank()">风云榜</a>|
            <a href="javascript:;" :class="{ on: ezfy.rankTab === 'prestige' }" @click="ezfy.rankTab = 'prestige'">军衔声望榜</a>|
            <a href="javascript:;" :class="{ on: ezfy.rankTab === 'troops' }" @click="ezfy.rankTab = 'troops'">战力榜</a>|
            <a href="javascript:;" :class="{ on: ezfy.rankTab === 'corps' }" @click="ezfy.rankTab = 'corps'">军团榜</a>|
            <a href="javascript:;" :class="{ on: ezfy.rankTab === 'ranks' }" @click="ezfy.rankTab = 'ranks'">军衔晋升表</a>
          </div>

          <!-- 军衔晋升表 tab（静态参照表 + 我的晋升，★ 2026-09-28 加宝物门槛） -->
          <template v-if="ezfy.rankTab === 'ranks'">
          <div class="panel-title">我的晋升</div>
          <div v-if="ezfy.rankData.mine" class="old-line">
            当前军衔：<b>{{ ezfy.rankData.mine.rank_name }}</b>（可建 {{ ezfy.rankData.mine.city_max }} 座，已有 {{ ezfy.rankData.mine.city_count }} 座）
          </div>
          <div v-if="ezfy.rankData.mine && ezfy.rankData.mine.next" class="old-line">
            下一军衔：<b>{{ ezfy.rankData.mine.next.name }}</b>（需要声望 <b>{{ ezfy.rankData.mine.next.need }}</b>，当前 {{ ezfy.rankData.mine.prestige }}）
            <button v-if="ezfy.canPromote()" :disabled="ezfy.promoting" @click="ezfy.doPromote">{{ ezfy.promoting ? '[晋升中…]' : '[晋升]' }}</button>
          </div>
          <div v-else-if="ezfy.rankData.mine" class="old-line green">已晋升至最高军衔「{{ ezfy.rankData.mine.rank_name }}」！</div>

          <div class="panel-title">军衔晋升表</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 22%"><col style="width: 16%"><col style="width: 20%"><col style="width: 14%"><col style="width: 10%">
            </colgroup>
            <tr><th>军衔</th><th>职位</th><th>声望</th><th>宝物</th><th>城数</th></tr>
            <template v-for="(r, i) in ezfy.rankData.ranks">
              <tr :key="'rk' + i">
                <td>
                  <span v-html="ezfy.rankIcon(r.id)"></span>
                  <span :class="r.name === (ezfy.rankData.mine ? ezfy.rankData.mine.rank_name : ezfy.rankName) ? 'red' : ''">{{ r.name }}</span>
                </td>
                <td>{{ r.post }}</td>
                <td>{{ r.need }}</td>
                <td>
                  <a href="javascript:;" @click="ezfy.showTreasureRow = ezfy.showTreasureRow === i ? -1 : i">[宝物]</a>
                </td>
                <td>{{ r.city_max }}</td>
              </tr>
              <tr v-if="ezfy.showTreasureRow === i" :key="'rt' + i" class="rank-treasure-row">
                <td>所需宝物</td>
                <td colspan="3" class="gray">{{ r.treasures || '该军衔无需宝物' }}</td>
                <td></td>
              </tr>
            </template>
          </table>
          </template>

          <!-- 军衔声望榜 tab -->
          <template v-if="ezfy.rankTab === 'prestige'">
          <div class="panel-title">军衔声望榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th>声望</th><th>军衔</th></tr>
            <tr v-for="r in ezfy.rankData.prestige" :key="'rp' + r.rank" :class="ezfy.rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="ezfy.openPlayer(r.user_id)">{{ r.name }}</a></td>
              <td>{{ r.prestige }}</td>
              <td>{{ r.rank_name }}<span style="margin-left:4px" v-html="ezfy.rankIcon(ezfy.rankIdByName(r.rank_name))"></span></td>
            </tr>
          </table>
          </template>

          <!-- 战力榜 tab（★ 2026-10-02 兵力榜 → 战力榜：科技/建筑/兵种柔和折算） -->
          <template v-if="ezfy.rankTab === 'troops'">
          <div class="panel-title">战力榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 18%"><col style="width: 40%"><col style="width: 42%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th title="战力 = 科技 + 建筑 + 兵种；兵种按「数量 × 占用人口」计（侦察机 1 人口 / 航母 55 人口），单兵种分最高 5 万">战力</th></tr>
            <template v-for="(r, i) in ezfy.rankData.troops">
              <tr :key="'rt' + r.rank" :class="ezfy.rankRowCls(r.rank)">
                <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
                <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="ezfy.openPlayer(r.user_id)">{{ r.role_name }}</a></td>
                <td><a href="javascript:;" @click="ezfy.showPowerRow = ezfy.showPowerRow === r.rank ? -1 : r.rank"><b>{{ ezfy.fmtN(r.power) }}</b></a></td>
              </tr>
              <tr v-if="ezfy.showPowerRow === r.rank" :key="'rp' + r.rank" class="rank-treasure-row">
                <td>战力明细</td>
                <td colspan="2" class="gray">科技 {{ ezfy.fmtN(r.tech_power) }} / 建筑 {{ ezfy.fmtN(r.build_power) }} / 兵种 {{ ezfy.fmtN(r.troop_power) }}</td>
              </tr>
            </template>
          </table>
          </template>

          <!-- 军团榜 tab（★ 2026-10-16 改为按「军团战力」排名：= Σ 成员战力，口径同战力榜） -->
          <template v-if="ezfy.rankTab === 'corps'">
          <div class="panel-title">军团榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>军团</th><th>人数</th><th>战力</th></tr>
            <tr v-for="r in ezfy.rankData.corps" :key="'rc' + r.rank" :class="ezfy.rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span>{{ r.name }}</td><td>{{ r.member_count }}</td><td>{{ ezfy.fmtN(r.power) }}</td>
            </tr>
          </table>
          </template>

          <!-- ★ 2026-10-14 风云榜 tab：按自然周统计玩家对战战绩（灭敌/战损/场次/胜场） -->
          <template v-if="ezfy.rankTab === 'pvp'">
          <div class="panel-title">
            风云榜
            <select v-if="ezfy.pvpRank.weeks && ezfy.pvpRank.weeks.length" :value="ezfy.pvpRank.week"
              style="margin-left:8px;font-size:12px" @change="ezfy.loadPvpRank($event.target.value)">
              <option v-for="w in ezfy.pvpRank.weeks.indexOf(ezfy.pvpRank.week) < 0 ? [ezfy.pvpRank.week].concat(ezfy.pvpRank.weeks) : ezfy.pvpRank.weeks" :key="w" :value="w">{{ w }}</option>
            </select>
            <span v-if="ezfy.pvpLoading" class="gray" style="margin-left:8px">加载中…</span>
            <span v-else class="gray" style="margin-left:8px" title="行动分：进攻20/防守15；胜负分：胜40/平15/负10；杀敌分按灭敌兵种价值加权对数计（最高200）；兵种越贵、兵团越大分越多（最多×3.5）；兵力不足约1万步兵/70航母不计分（出征上限由司令部+集结令决定）；小仗最多400分，大兵团会战（约1万航母/150万步兵）封顶800；侦察机/卡车不计分；同对手每周封顶20000">按积分排名 · 大兵团分多 · 同对手周封顶</span>
          </div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 18%"><col style="width: 40%"><col style="width: 42%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th>积分</th></tr>
            <template v-if="ezfy.pvpRank.ranks && ezfy.pvpRank.ranks.length">
              <template v-for="r in ezfy.pvpRank.ranks">
                <tr :key="'pv' + r.rank"
                  :class="ezfy.rankRowCls(r.rank)"
                  :style="ezfy.pvpRank.mine && ezfy.pvpRank.mine.user_id === r.user_id ? 'background:#fff7e6;font-weight:bold' : ''">
                  <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
                  <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="ezfy.openPlayer(r.user_id)">{{ r.name }}</a></td>
                  <td><a href="javascript:;" @click="ezfy.showPvpRow = ezfy.showPvpRow === r.rank ? -1 : r.rank"><b :title="'具体积分 ' + ezfy.fmtN(r.points) + ' 分'">{{ ezfy.fmtProd(r.points) }}</b></a></td>
                </tr>
                <!-- ★ 2026-10-16 战绩明细简约化（适合手机）：两行紧凑展示，展开显示具体积分 -->
                <tr v-if="ezfy.showPvpRow === r.rank" :key="'pvd' + r.rank" class="rank-treasure-row">
                  <td colspan="3" class="gray pvp-detail">
                    <div style="line-height:1.9">具体积分 <b>{{ ezfy.fmtN(r.points) }}</b> 分 · 灭敌 <span :title="'灭敌 ' + ezfy.fmtN(r.kills)">{{ ezfy.fmtProd(r.kills) }}</span> · 战损 <span :title="'战损 ' + ezfy.fmtN(r.losses)">{{ ezfy.fmtProd(r.losses) }}</span></div>
                    <div style="line-height:1.9">参战 {{ r.battles }} 场 · 胜率 {{ Math.round(r.win_rate * 10) / 10 }}% · 战损比 {{ r.losses > 0 ? (Math.round((r.kills / r.losses) * 10) / 10) : (r.kills > 0 ? '∞' : '—') }}</div>
                  </td>
                </tr>
              </template>
            </template>
            <!-- ★ 2026-10-14 风云榜统计开关关闭时明确提示，而不是让玩家看到「暂无记录」 -->
            <tr v-else><td colspan="3" class="gray">{{ ezfy.pvpRank.enabled === false ? '风云榜暂未开启' : '该周暂无玩家对战记录' }}</td></tr>
          </table>
          <div v-if="ezfy.pvpRank.mine && ezfy.pvpRank.mine.user_id" class="old-line">
            我的战绩（{{ ezfy.pvpRank.week }}）：积分 <b :title="'具体积分 ' + ezfy.fmtN(ezfy.pvpRank.mine.points) + ' 分'">{{ ezfy.fmtProd(ezfy.pvpRank.mine.points) }}</b> · 参战 <b>{{ ezfy.pvpRank.mine.battles }}</b> 场 · 胜 <b>{{ ezfy.pvpRank.mine.wins }}</b> 场 · 灭敌 <b :title="'灭敌 ' + ezfy.fmtN(ezfy.pvpRank.mine.kills)">{{ ezfy.fmtProd(ezfy.pvpRank.mine.kills) }}</b> / 战损 <b :title="'战损 ' + ezfy.fmtN(ezfy.pvpRank.mine.losses)">{{ ezfy.fmtProd(ezfy.pvpRank.mine.losses) }}</b>
          </div>
          <!-- ★ 2026-10-16 「我的战绩」场次明细：就地展开/收起（不弹模态框） -->
          <div v-if="ezfy.pvpDetail.show" class="pvp-inline">
            <div v-if="ezfy.pvpDetail.loading" class="gray" style="padding:6px 4px">加载中…</div>
            <template v-else>
              <div v-if="!ezfy.pvpDetail.list.length" class="gray" style="padding:6px 4px">本周暂无对战记录</div>
              <div v-for="(it, i) in ezfy.pvpDetail.list" :key="'pvd' + i"
                style="display:flex;align-items:center;gap:6px;padding:5px 2px;border-bottom:1px dashed #eee;font-size:13px">
                <span class="gray" style="width:86px;flex:none">{{ it.ts_fmt }}</span>
                <span :class="it.is_win ? 'green' : (it.is_draw ? 'orange' : 'red')" style="width:16px;flex:none">
                  {{ it.is_win ? '胜' : (it.is_draw ? '平' : '负') }}</span>
                <span style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">vs {{ it.foe_nick || '—' }}
                  <span class="gray" v-if="it.kills || it.losses">（灭{{ ezfy.fmtProd(it.kills) }} / 损{{ ezfy.fmtProd(it.losses) }}）</span>
                </span>
                <b :class="it.points > 0 ? 'green' : 'gray'" style="flex:none">{{ it.points > 0 ? '+' + ezfy.fmtN(it.points) : '+0' }}</b>
              </div>
              <div style="display:flex;align-items:center;gap:10px;padding:6px 2px;font-size:13px">
                <a href="javascript:;" @click="ezfy.pvpDetailPage(-1)">上一页</a>
                <span class="gray">{{ ezfy.pvpDetail.page }}/{{ Math.max(1, Math.ceil(ezfy.pvpDetail.total / ezfy.pvpDetail.size)) }}</span>
                <a href="javascript:;" @click="ezfy.pvpDetailPage(1)">下一页</a>
              </div>
            </template>
          </div>
          </template>

          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
          <a v-if="ezfy.pvpRank.mine && ezfy.pvpRank.mine.user_id" href="javascript:;" style="margin-left:6px" @click="ezfy.togglePvpDetail()">{{ ezfy.pvpDetail.show ? '[收回场次明细]' : '[场次明细]' }}</a>
        </div>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化拆分：军衔/榜单页模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyRank',
  inject: ['ezfy']
}
</script>

<style scoped>
/* ★ 2026-10-16 「我的战绩」场次明细：就地展开（不弹模态框） */
.pvp-inline {
  margin: 4px 0 10px; padding: 6px 8px;
  background: #fbfbfb; border: 1px solid #eee; border-radius: 6px;
}
.pvp-inline a { color: #0055aa; }
/* ★ 2026-10-16 电脑端明细条太宽：收窄到面板约一半宽；手机端保持全宽 */
@media (min-width: 769px) {
  .pvp-inline { max-width: 50%; }
}
</style>
