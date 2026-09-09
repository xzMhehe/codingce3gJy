<template>
  <div>
    <!-- 面包屑 -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;综合排行榜</div>

    <!-- 标题（诺哈 rank.asp：【综合排行】） -->
    <div class="name">【综合排行】</div>

    <!-- 选项卡（诺哈：按：评分.收藏.阅读.评论，当前项纯文本，其余链接） -->
    <div class="module-content">
      按：
      <template v-for="(t, i) in tabs">
        <template v-if="i > 0">.</template>
        <a v-if="type !== t.key" :key="t.key" href="javascript:;" @click="switchTab(t.key)">{{ t.name }}</a>
        <span v-else :key="t.key">{{ t.name }}</span>
      </template>
      <br>
    </div>

    <!-- 排行列表（诺哈：N.昵称 编号行） -->
    <div class="list">
      <div class="row" v-for="(r, i) in rows" :key="r.id">
        <span class="no">{{ i + 1 }}.</span>
        <a href="javascript:;" @click="$router.push('/user/' + r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a>({{ r.value }}{{ r.unit }})<br>
      </div>
      <div v-if="!rows.length" class="row"><span class="txt-fade">暂无记录！</span><br></div>
    </div>

    <!-- 我的排名 -->
    <div class="item">
      我的排名：<b>{{ myRank }}</b>名<br>
    </div>

    <!-- 页脚导航 -->
    <div class="module-content" style="margin-top:6px">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>.<a href="javascript:;" @click="$router.push('/box')">用户中心</a>.<a href="javascript:;" @click="$router.push('/security')">安全中心</a>.<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a><br>
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
/* 排行序号与昵称间距（同「最新帖子」） */
.list .row .no { margin-right: 4px; color: #999; }
</style>