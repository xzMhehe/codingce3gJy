<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;{{ board.name }}
    </div>
    <div class="module-content">{{ board.description }}</div>
    <template v-for="ch in subGroups">
      <div class="module-title" :key="'t'+ch.id">【{{ ch.name }}】</div>
      <div class="module-content" :key="'c'+ch.id">
        <template v-for="(s,i) in ch.children">
          <a :key="'l'+s.id" href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a><span :key="'s'+s.id" v-if="(i+1)%3!==0 && i<ch.children.length-1"> . </span><br v-else-if="i<ch.children.length-1" :key="'b'+s.id">
        </template>
      </div>
    </template>
    <div class="module-content">
      <form @submit.prevent="">
        [按名称搜索]<br>
        <input type="text" v-model.trim="wd" maxlength="30">
        <button class="btn small" @click.prevent="$router.push('/board/'+$route.params.id+'?wd='+encodeURIComponent(wd))">搜索</button>
      </form>
    </div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;{{ board.name }}
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Channel',
  data () { return { board: {}, subGroups: [], wd: '' } },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/boards').then(r => {
        if (r.code !== 0) return
        const target = r.data.find(b => String(b.id) === String(id))
        if (!target) return
        this.board = target
        if (target.children && target.children.length) {
          this.subGroups = [{ id: target.id, name: target.name, children: target.children }]
        }
        this.wd = this.$route.query.wd || ''
      })
    }
  }
}
</script>
