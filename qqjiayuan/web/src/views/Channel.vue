<template>
  <div>
    <!-- 面包屑（诺哈论坛天地：家园>论坛） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;{{ board.name || '论坛' }}<br></div>

    <!-- 论坛工具箱（诺哈：草稿/帖子工具入口） -->
    <div class="module-content" v-if="board.name"><a href="javascript:;" @click="$router.push('/my-threads')">论坛工具箱</a><br></div>

    <!-- 同城行（诺哈论坛页：同城：大理.邯郸.保定） -->
    <div class="module-content" v-if="cities.length">
      <a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>：<a v-for="(ct, i) in cities" :key="'ct'+ct.id" href="javascript:;" @click="$router.push('/tongcheng/province/' + ct.id)">{{ ct.name }}</a><template v-if="i < cities.length - 1">.</template><br>
    </div>

    <!-- 子版块行（诺哈：版块名.版块名.版块名） -->
    <div class="module-content" v-if="subs.length">
      <template v-for="(s, i) in subs"><a :key="'s'+s.id" href="javascript:;" @click="$router.push('/board/' + s.id)">{{ s.name }}</a><template v-if="i < subs.length - 1">.</template></template><br>
    </div>

    <!-- 各版块最新帖（诺哈：版块名标题 + 3 条主题(N阅) + 更多热点>>） -->
    <template v-for="s in activeSubs">
      <div class="module-title" :key="'bt'+s.id"><a href="javascript:;" @click="$router.push('/board/' + s.id)">{{ s.name }}</a></div>
      <div class="module-content" :key="'bc'+s.id">
        <div v-for="t in s.threads" :key="'t'+t.id"><a href="javascript:;" @click="$router.push('/thread/' + t.id)">{{ t.title }}({{ t.view_count }}阅)</a><br></div>
        <a href="javascript:;" @click="$router.push('/board/' + s.id)">更多热点&gt;&gt;</a><br>
      </div>
    </template>

    <!-- 热点版块 -->
    <div class="module-content" v-if="subs.length">热点版块<br>
      <template v-for="(s, i) in subs"><a :key="'h'+s.id" href="javascript:;" @click="$router.push('/board/' + s.id)">{{ s.name }}</a><template v-if="i < subs.length - 1">.</template></template><br>
    </div>

    <!-- 论坛导航 / 服务论坛 -->
    <div class="module-content"><a href="javascript:;" @click="$router.push('/nav')">论坛导航</a>.<a href="javascript:;" @click="$router.push('/channel/4')">服务论坛</a><br></div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;{{ board.name || '论坛' }}<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Channel',
  data () { return { board: {}, subs: [], cities: [] } },
  computed: {
    activeSubs () { return this.subs.filter(s => s.threads && s.threads.length) }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/boards/' + id + '/forum').then(r => {
        if (r.code !== 0) {
          // 非分区板块（如同城省份）→ 交给版块页处理
          this.$router.replace('/board/' + id)
          return
        }
        this.board = r.data.board || {}
        this.subs = r.data.subs || []
      }).catch(() => {})
      // 同城行（诺哈论坛页固定行）：取同城省份前三个
      api.get('/tongcheng').then(r => {
        if (r.code === 0) this.cities = (r.data.provinces || []).slice(0, 3)
      })
    }
  }
}
</script>
