<template>
  <div>
    <!-- 面包屑 panav（诺哈：社区 > 分类名） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;{{ board.name }}<br>
    </div>

    <!-- 简介 -->
    <div class="module-content">{{ board.description }}</div>

    <!-- 子版块列表（诺哈 category.asp 编号列表；有分类则按分类分组） -->
    <template v-if="board.categories && board.categories.length">
      <div class="module-content" style="padding:0">
        <div class="module-title" v-for="cat in board.categories" :key="'ct'+cat.id">【{{ cat.name }}】</div>
        <div class="list">
          <div class="row" v-for="(s, i) in flattenCats(board.categories)" :key="'cl'+s.id">
            <template v-if="i === 0">0.</template><template v-else>{{ i }}.</template><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a><br>
          </div>
        </div>
      </div>
    </template>

    <!-- 无分类：直接编号列出所有子版块 -->
    <template v-else-if="board.children && board.children.length">
      <div class="list">
        <div class="row" v-for="(s, i) in board.children" :key="'c'+s.id">
          {{ i+1 }}.<a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a><br>
        </div>
      </div>
    </template>
    <div class="module-content" v-else>该分区下暂无版块<br></div>

    <!-- 搜索（诺哈按版块名搜索） -->
    <div class="module-content">
      <form @submit.prevent="">
        [按名称搜索]<br>
        <input type="text" v-model.trim="wd" maxlength="30">
        <button class="btn small" @click.prevent="doSearch">搜索</button>
      </form>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;{{ board.name }}<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Channel',
  data () { return { board: {}, wd: '' } },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      this.wd = this.$route.query.wd || ''
      api.get('/boards').then(r => {
        if (r.code !== 0) return
        const target = r.data.find(b => String(b.id) === String(id))
        if (!target) return
        this.board = target
        if (!target.categories || !target.categories.length) {
          // 保证 children 存在
          this.board = Object.assign({}, target)
        }
      })
    },
    flattenCats (cats) {
      const arr = []
      cats.forEach(cat => (cat.boards || []).forEach(b => arr.push(b)))
      return arr
    },
    doSearch () {
      if (!this.wd) return
      // 站内搜索
      this.$router.push('/search?w=' + encodeURIComponent(this.wd))
    }
  }
}
</script>