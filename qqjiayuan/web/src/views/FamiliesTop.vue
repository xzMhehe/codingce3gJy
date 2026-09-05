<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;排行榜<br></div>

    <div class="module-title">【家族排行榜】</div>
    <div class="list">
      <div class="row" v-for="t in types" :key="t.key">
        {{ t.no }}.<a href="javascript:;" :style="type===t.key ? 'color:#c00;font-weight:bold' : ''" @click="$router.push({ query: { bt: t.key } })">{{ t.name }}</a><br>
      </div>
    </div>
    <div class="module-content">----------<br></div>

    <div class="list">
      <div class="row" v-for="(f,i) in rows" :key="f.id">
        {{ i + 1 }}.<a href="javascript:;" @click="$router.push('/family/'+f.id)">{{ f.name }}</a>
        <span class="txt-fade">—— {{ valueText(f) }}</span><br>
      </div>
      <div class="row" v-if="!rows.length"></div>
    </div>
    <div class="module-content" v-if="!rows.length"><span class="empty">暂无排行数据</span></div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;排行榜<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamiliesTop',
  data () {
    return {
      rows: [],
      types: [
        { key: 1, no: 1, name: '规模排行榜' },
        { key: 2, no: 2, name: '活跃排行榜' },
        { key: 3, no: 3, name: '人气排行榜' },
        { key: 4, no: 4, name: '等级排行榜' }
      ]
    }
  },
  computed: {
    type () { const n = parseInt(this.$route.query.bt || 1); return n >= 1 && n <= 4 ? n : 1 }
  },
  watch: { '$route.query.bt': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/families/top', { params: { bt: this.type } }).then(r => {
        if (r.code === 0) this.rows = r.data
      })
    },
    valueText (f) {
      switch (this.type) {
        case 1: return f.members + '人'
        case 2: return '今日签到 ' + f.active + ' 人'
        case 3: return '乐斗积分 ' + f.score
        case 4: return '守护树 Lv.' + f.level
        default: return ''
      }
    }
  }
}
</script>
