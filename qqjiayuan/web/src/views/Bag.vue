<template>
  <div>
    <div class="name">我的仓库<br></div>
    <div class="module-content">
      <span class="txt-fade">购买的道具与鲜花都在这里</span>
      [<a href="javascript:;" @click="$router.push('/shop')">返回商城</a>]
    </div>

    <div class="list">
      <div class="row" v-for="ug in list" :key="ug.id" style="padding:4px 2px;border-bottom:1px dotted #dfe8f2">
        <img v-if="ug.icon" class="gicon" :src="'/static/picture/' + ug.icon" :alt="ug.name">
        <b>{{ ug.name }}</b>
        <span class="txt-fade">×{{ ug.count }}（{{ ug.category }}）</span>
        <div class="txt-fade">{{ ug.desc }}</div>
        [<a href="javascript:;" @click="use(ug)">使用</a>]
      </div>
    </div>
    <div class="module-content" v-if="!list.length"><span class="empty">仓库空空如也，去商城逛逛吧～</span></div>

    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Bag',
  data () { return { list: [], msg: '', okMsg: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/bag').then(r => { if (r.code === 0) this.list = r.data.list || [] })
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
          this.load()
        } else this.msg = r.msg
      })
    }
  }
}
</script>

<style scoped>
.gicon { width: 30px; height: 30px; vertical-align: middle; margin-right: 3px; }
</style>
