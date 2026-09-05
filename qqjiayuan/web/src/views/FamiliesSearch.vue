<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;搜索家族<br></div>

    <div class="module-title">【搜索家族】</div>
    <div class="list">
      <div class="row" v-for="(f,i) in rows" :key="f.id">
        {{ i + 1 }}.<a href="javascript:;" @click="$router.push('/family/'+f.id)">{{ f.name }}</a>
        <span class="txt-fade">（{{ f.category || '未分类' }}，{{ f.members || 0 }}人）</span><br>
      </div>
    </div>
    <div class="module-content" v-if="searched && !rows.length"><span class="empty">没有找到相关家族，换个关键词试试</span></div>

    <div class="module-content">(第<b>1</b>/1页/共{{ rows.length }}条记录)<br></div>

    <div class="module-content">
      <form @submit.prevent="doSearch">
        <input type="text" v-model.trim="wd" maxlength="30" size="12">
        <input type="submit" value="搜索">
      </form>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;搜索家族<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamiliesSearch',
  data () { return { wd: '', rows: [], searched: false } },
  watch: { '$route.params.wd': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.wd = this.$route.params.wd || ''
      if (!this.wd) { this.rows = []; this.searched = false; return }
      api.get('/families/search', { params: { wd: this.wd } }).then(r => {
        this.searched = true
        if (r.code === 0) this.rows = r.data
      })
    },
    doSearch () {
      if (this.wd) this.$router.push('/families/search/' + encodeURIComponent(this.wd))
    }
  }
}
</script>
