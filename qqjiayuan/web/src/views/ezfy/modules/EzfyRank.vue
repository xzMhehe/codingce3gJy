<template>
  <div>
    <template v-if="ezfy.cur === 'rank'">
        <div class="panel">
          <!-- ★ 2026-09-24 用户要求：军衔晋升表/军衔声望榜/兵力榜/军团榜做成 tab 分开展示 -->
          <div class="acade-tab">
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
            <div class="gray">
              需要宝物：
              <span v-for="t in ezfy.rankData.mine.next.treasures" :key="'tr' + t.name">
                {{ t.name }}×{{ t.count }}（背包{{ t.have }}）
                <span :class="t.have >= t.count ? 'green' : 'red'">{{ t.have >= t.count ? '足够' : '不足' }}</span>；
              </span>
            </div>
            <button v-if="ezfy.canPromote()" @click="ezfy.doPromote">[晋升]</button>
            <span v-else class="gray">声望达标且宝物足够后才能晋升（宝物通过野地采集获得）</span>
          </div>
          <div v-else-if="ezfy.rankData.mine" class="old-line green">已晋升至最高军衔「{{ ezfy.rankData.mine.rank_name }}」！</div>

          <div class="panel-title">军衔晋升表</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 10%"><col style="width: 20%"><col style="width: 14%"><col style="width: 18%"><col style="width: 12%"><col style="width: 8%">
            </colgroup>
            <tr><th>等级</th><th>军衔</th><th>职位</th><th>声望</th><th>宝物</th><th>城数</th></tr>
            <template v-for="(r, i) in ezfy.rankData.ranks">
              <tr :key="'rk' + i">
                <td>{{ i + 1 }}</td>
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
                <td colspan="4" class="gray">{{ r.treasures || '该军衔无需宝物' }}</td>
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
          <div class="old-line gray">战力 = 科技 + 建筑 + 兵种（按你的最好城市折算），点击战力查看明细</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 13%"><col style="width: 32%"><col style="width: 30%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th>城市</th><th>战力</th></tr>
            <template v-for="(r, i) in ezfy.rankData.troops">
              <tr :key="'rt' + r.rank" :class="ezfy.rankRowCls(r.rank)">
                <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
                <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="ezfy.openPlayer(r.user_id)">{{ r.role_name }}</a></td>
                <td>{{ r.city_name }}</td>
                <td><a href="javascript:;" @click="ezfy.showPowerRow = ezfy.showPowerRow === r.rank ? -1 : r.rank"><b>{{ ezfy.fmtN(r.power) }}</b></a></td>
              </tr>
              <tr v-if="ezfy.showPowerRow === r.rank" :key="'rp' + r.rank" class="rank-treasure-row">
                <td>战力明细</td>
                <td colspan="3" class="gray">科技 {{ ezfy.fmtN(r.tech_power) }} / 建筑 {{ ezfy.fmtN(r.build_power) }} / 兵种 {{ ezfy.fmtN(r.troop_power) }}</td>
              </tr>
            </template>
          </table>
          </template>

          <!-- 军团榜 tab -->
          <template v-if="ezfy.rankTab === 'corps'">
          <div class="panel-title">军团榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>军团</th><th>人数</th><th>战力</th></tr>
            <tr v-for="r in ezfy.rankData.corps" :key="'rc' + r.rank" :class="ezfy.rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span>{{ r.name }}</td><td>{{ r.member_count }}</td><td>{{ r.battle_score }}</td>
            </tr>
          </table>
          </template>

          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
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
