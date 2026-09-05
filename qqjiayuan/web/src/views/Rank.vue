<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;综合排行榜</div>
    <div class="note"></div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/box')">用户中心</a>|<a href="javascript:;" @click="$router.push('/security')">安全中心</a>|<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>|家园排行<br>
    </div>

    <div class="tab-nav">
      <a v-for="t in tabs" :key="t.key" :class="['tab-btn', { active: type === t.key }]" href="javascript:;" @click="switchTab(t.key)">{{ t.name }}</a>
    </div>

    <div class="tab-content">
      <div v-for="(r, i) in rows" :key="r.id" class="rank-item" :class="'rk' + Math.min(i + 1, 3)">
        {{ i + 1 }}.<a href="javascript:;" @click="$router.push('/user/' + r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a><span v-if="r.unit" class="rk-val">({{ r.value }}{{ r.unit }})</span>
      </div>
      <div v-if="!rows.length" class="rank-item"><span class="txt-fade">暂无排行数据</span></div>
      <div class="my-rank">我的排名：<b>{{ myRank }}</b>名</div>
    </div>

    <div class="module-content" style="margin-top:6px">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>|<a href="javascript:;" @click="$router.push('/box')">用户中心</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Rank',
  data () {
    return {
      type: 'noble',
      rows: [],
      myRank: 0,
      tabs: [
        { key: 'topic', name: '月发' },
        { key: 'reply', name: '月回' },
        { key: 'noble', name: '超Q' },
        { key: 'coins', name: 'G币' },
        { key: 'yuanbao', name: '元宝' },
        { key: 'exp', name: '论坛' },
        { key: 'home', name: '家园' },
        { key: 'level', name: '等级' }
      ]
    }
  },
  mounted () { this.load() },
  methods: {
    switchTab (key) { this.type = key; this.load() },
    load () {
      api.get('/rank', { params: { type: this.type } }).then(r => {
        if (r.code === 0) {
          this.rows = r.data.list || []
          this.myRank = r.data.my_rank || 0
        }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.tab-nav { padding: 5px; background: #f0f0f0; overflow-x: auto; white-space: nowrap; }
.tab-btn { display: inline-block; padding: 5px 10px; margin: 2px; border: 1px solid #ccc; background: #fff; text-decoration: none; font-size: 12px; border-radius: 3px; }
.tab-btn.active { background: #007bff; color: #fff; border-color: #007bff; }
.tab-content { padding: 10px 5px; }
.rank-item { padding: 5px 0; border-bottom: 1px solid #eee; }
.rank-item.rk1 { color: #ff6600; font-weight: bold; }
.rank-item.rk2 { color: #888; }
.rank-item.rk3 { color: #cd7f32; }
.rk-val { color: #999; }
.my-rank { padding: 8px; background: #e7f3ff; margin-top: 10px; border-radius: 5px; font-size: 12px; }
</style>
