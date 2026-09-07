<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的收藏</div>
    <div class="name">我的收藏<br></div>
    <div class="module-title">
      <a href="javascript:;" :style="{ fontWeight: ft === 'all' ? 'bold' : 'normal' }" @click="ft='all'">全部</a> |
      <a href="javascript:;" :style="{ fontWeight: ft === '2' ? 'bold' : 'normal' }" @click="ft='2'">帖子</a> |
      <a href="javascript:;" :style="{ fontWeight: ft === '4' ? 'bold' : 'normal' }" @click="ft='4'">日志</a> |
      <a href="javascript:;" :style="{ fontWeight: ft === '5' ? 'bold' : 'normal' }" @click="ft='5'">相册</a> |
      <a href="javascript:;" :style="{ fontWeight: ft === '6' ? 'bold' : 'normal' }" @click="ft='6'">商品</a>
    </div>
    <div class="list" v-if="list.length">
      <div v-for="f in list" :key="f.id" class="row">
        <a href="javascript:;" @click="go(f)">{{ f.name }}</a>
        <span class="txt-fade">({{ typeName(f.f_type) }})</span>
        <a href="javascript:;" @click="del(f.id)">[取消收藏]</a><br>
      </div>
    </div>
    <span v-else class="empty">还没有收藏</span>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Favorites',
  data () { return { list: [], ft: 'all' } },
  mounted () { this.load() },
  methods: {
    load () {
      const params = this.ft === 'all' ? {} : { f_type: this.ft }
      api.get('/favorites', { params }).then(r => { if (r.code === 0) this.list = r.data || [] })
    },
    del (id) { api.delete('/favorites/' + id).then(() => this.load()) },
    typeName (t) {
      return { 1: '论坛', 2: '帖子', 3: '家族', 4: '日志', 5: '相册', 6: '商品' }[t] || '其他'
    },
    go (f) {
      if (f.f_type === 2) this.$router.push('/thread/' + f.ref_id)
      else if (f.f_type === 6) this.$router.push('/market/' + f.ref_id)
      else if (f.f_type === 4) this.$router.push('/space/article/' + f.ref_id)
      else this.$router.push('/')
    }
  }
}
</script>