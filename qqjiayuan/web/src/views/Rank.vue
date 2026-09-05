<template>
  <div>
    <div class="module-title">
      <a v-for="t in types" :key="t.key" :class="{ cur: type === t.key }" href="javascript:;" @click="type=t.key; load()">{{ t.name }}</a>
    </div>
    <div class="list">
      <div v-for="(r,i) in rows" :key="r.id" class="row">
        {{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a>
        <span class="txt-fade">—— {{ r.value }}</span>
      </div>
    </div>
    <div class="module-content" v-if="!rows.length"><span class="empty">暂无排行数据</span></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Rank',
  data () {
    return {
      type: 'coins', rows: [],
      types: [
        { key: 'coins', name: 'G币排行' }, { key: 'exp', name: '经验排行' },
        { key: 'level', name: '等级排行' }, { key: 'sign', name: '签到排行' }
      ]
    }
  },
  mounted () { this.load() },
  methods: {
    load () { api.get('/rank?type=' + this.type).then(r => { if (r.code === 0) this.rows = r.data }) }
  }
}
</script>

<style scoped>
.module-title a { margin-right: 10px; }
.module-title a.cur { color: #c00; font-weight: bold; }
</style>
