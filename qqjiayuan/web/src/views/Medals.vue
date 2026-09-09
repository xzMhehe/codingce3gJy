<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;勋章<br></div>

    <div class="module-title">
      <a :class="{ cur: cur === 'mine' }" href="javascript:;" @click="cur='mine'">我的勋章</a>
      <a :class="{ cur: cur === 'shop' }" href="javascript:;" @click="cur='shop'">勋章大全</a>
    </div>

    <!-- 我的勋章 -->
    <template v-if="cur === 'mine'">
      <div class="module-content" v-if="mine.length">
        <div v-for="m in mine" :key="m.id" class="medal-row">
          <img class="medal-ico" :src="'/static/picture/' + m.icon" :alt="m.name">
          <b>{{ m.name }}</b>
          <span class="txt-fade">{{ m.remark }}</span>
          <template v-if="m.expire_at">
            <span class="txt-fade">（剩{{ daysLeft(m.expire_at) }}天到期）</span>
          </template>
          <template v-else>
            <span class="txt-fade">（永久）</span>
          </template>
          <br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">你还没有勋章，勋章由社区授予，快去活跃发帖吧！</span></div>
    </template>

    <!-- 勋章大全 -->
    <template v-if="cur === 'shop'">
      <div class="module-content txt-fade">勋章由社区管理员授予，荣誉的象征。以下是全部勋章：</div>
      <div class="module-content" v-if="shop.length">
        <div v-for="b in shop" :key="b.id" class="medal-row">
          <img class="medal-ico" :src="'/static/picture/' + b.icon" :alt="b.name">
          <b>{{ b.name }}</b>
          <span class="txt-fade">{{ b.remark }}</span>
          <template v-if="b.period"><span class="txt-fade">（有效期 {{ b.period }} 天）</span></template>
          <br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无勋章</span></div>
    </template>

    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;勋章<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Medals',
  data () { return { cur: 'mine', mine: [], shop: [] } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/my-medals').then(r => { if (r.code === 0) this.mine = r.data.list || [] })
      api.get('/badges').then(r => { if (r.code === 0) this.shop = r.data || [] })
    },
    daysLeft (t) {
      if (!t) return 0
      const d = Math.ceil((new Date(t).getTime() - Date.now()) / 86400000)
      return Math.max(0, d)
    }
  }
}
</script>

<style scoped>
.medal-row { padding: 6px 2px; border-bottom: 1px dotted #dfe8f2; }
.medal-ico { width: 22px; height: 22px; vertical-align: middle; margin-right: 4px; }
</style>