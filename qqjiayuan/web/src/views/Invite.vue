<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;邀请</div>
    <div class="name">邀请开通家园<br></div>
    <div class="module-content" v-if="data">
      <span class="txt-fade">{{ data.rule_text }}</span><br>
      把下面的链接/邀请码发给好友，TA 注册时填写邀请码，你和 TA 各得 <b style="color:#e05a00">50 G币</b>！<br>
      <br>
      我的邀请码:<b style="color:#e05a00">{{ data.code }}</b><br>
      <span class="txt-fade">{{ data.reward }}</span><br>
      <input type="text" :value="data.link" readonly size="30" /> <a href="javascript:;" @click="copy(data.link)">复制</a><br>
      <a href="javascript:;" @click="$router.push(data.link)">去注册页填写邀请码&gt;&gt;</a><br>
    </div>

    <div class="module-title">已邀请的居民({{ data ? data.invited_count : 0 }})</div>
    <div class="list" v-if="data && data.invited.length">
      <div v-for="u in data.invited" :key="u.id" class="row">
        <a href="javascript:;" @click="$router.push('/user/'+u.id)">{{ u.nickname }}</a>(号码{{ u.username }}) {{ fmt(u.created_at) }}<br>
      </div>
    </div>
    <span v-else class="empty">还没有邀请到好友，快把你的邀请码发出去吧！</span>
    <p v-if="tip" style="color:#1a9e1a;padding:3px 5px">{{ tip }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Invite',
  data () { return { data: null, tip: '' } },
  mounted () { api.get('/invite').then(r => { if (r.code === 0) this.data = r.data }) },
  methods: {
    copy (s) {
      const inp = document.createElement('input')
      inp.value = s; document.body.appendChild(inp); inp.select()
      document.execCommand('copy'); document.body.removeChild(inp)
      this.tip = '已复制到剪贴板'
    },
    fmt (t) { return t ? t.slice(0, 10) : '' }
  }
}
</script>