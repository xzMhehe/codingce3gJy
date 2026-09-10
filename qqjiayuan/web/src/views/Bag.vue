<template>
  <div>
    <!-- 面包屑（复刻诺哈 my_bag.asp：我的花园>背包 → 这里：用户中心>我的仓库） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">用户中心</a>&gt;我的仓库<br>
    </div>
    <div class="name">我的仓库<br></div>

    <!-- 平铺编号列表（复刻诺哈 my_bag.asp：n.名称 (数量) [使用]） -->
    <div class="module-content" v-if="loaded">
      <span v-for="(ug, i) in list" :key="ug.id">{{ i + 1 + (page - 1) * 10 }}.<img v-if="ug.icon" class="gicon" :src="'/static/picture/' + ug.icon" :alt="ug.name" @error="hideIcon">{{ ug.name }} ({{ ug.count }})<template v-if="ug.category !== '鲜花'"> <a href="javascript:;" @click="use(ug)">使用</a></template><template v-else><span class="txt-fade">帖子下方【送花】使用</span></template><br></span>
      <span v-if="!list.length" class="empty">没有道具了，先去<a href="javascript:;" @click="$router.push('/shop')">商店</a>买点吧。</span>
    </div>
    <!-- 分页（复刻诺哈：下页.上页 + 跳页输入） -->
    <div class="module-content" v-if="totalPages > 1">
      <a href="javascript:;" v-if="page < totalPages" @click="load(page + 1)">下页</a><span v-if="page < totalPages && page > 1">.</span><a href="javascript:;" v-if="page > 1" @click="load(page - 1)">上页</a>
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}条记录)<br>
      第<input type="text" v-model.number="jumpPage" size="2">页 <input type="submit" value="前往" @click="jump">
    </div>

    <div class="module-content">----------<br>
      [<a href="javascript:;" @click="$router.push('/shop')">去商店购买</a>]
      [<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>]
    </div>

    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>

    <!-- 底部面包屑 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">用户中心</a>&gt;我的仓库<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Bag',
  data () { return { list: [], page: 1, total: 0, totalPages: 1, jumpPage: 1, loaded: false, msg: '', okMsg: '' } },
  watch: {
    $route: 'sync'
  },
  mounted () { this.sync() },
  methods: {
    sync () {
      this.load(this.$route.query.page ? parseInt(this.$route.query.page) || 1 : 1)
    },
    load (p) {
      this.page = p
      api.get('/bag', { params: { page: p } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          const size = r.data.size || 10
          this.totalPages = Math.max(1, Math.ceil(this.total / size))
        }
        this.loaded = true
      })
    },
    jump () {
      const p = Math.floor(this.jumpPage || 0)
      if (p >= 1 && p <= this.totalPages) { this.load(p) } else { this.msg = '页码超出范围！' }
    },
    hideIcon (e) {
      // 图标缺失兜底：隐藏破图（诺哈 wap 版本无图标）
      e.target.style.display = 'none'
    },
    use (ug) {
      this.msg = ''
      this.okMsg = ''
      let body = {}
      if (ug.name === '改名卡') {
        const nick = prompt('请输入新的昵称（2~12个字符）：', '')
        if (!nick) return
        body = { nickname: nick.trim() }
      }
      api.post('/bag/' + ug.id + '/use', body).then(r => {
        if (r.code === 0) {
          this.okMsg = r.data.msg
          this.load(this.page)
        } else this.msg = r.msg
      })
    }
  }
}
</script>

<style scoped>
.gicon { width: 30px; height: 30px; vertical-align: middle; margin-right: 3px; }
</style>
