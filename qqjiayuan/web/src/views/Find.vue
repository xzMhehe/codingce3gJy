<template>
  <div>
    <div class="bar">【找回资料】</div>
    <div class="module-content">
      <p>【提交下面任意信息皆可找回登陆ID账号】<br>
      1. 忘记号码：输入你的昵称查询<br>
      2. 忘记密码：凭号码联系客服重置<br></p>
      <div class="line"></div>
      <form @submit.prevent="doFind">
        <div class="form-item">
          <label>你的昵称:</label>
          <input type="text" v-model.trim="nickname" maxlength="20">
        </div>
        <div class="form-item"><button class="btn" type="submit">查 找</button></div>
      </form>
      <div v-if="list.length" class="deep" style="border:1px solid #9FC6EC;padding:5px;margin:5px 0">
        <p v-for="u in list" :key="u.id">
          昵称 {{ u.nickname }} —— 家园号码 <b style="color:#c00">{{ u.username }}</b>（{{ u.created_at }} 注册）
        </p>
      </div>
      <p v-else-if="searched" class="empty">没有找到匹配的账号</p>
      <div class="line"></div>
      <p>忘记密码请到 <a href="javascript:;" @click="$router.push('/channel/4')">客服中心</a> 发帖申诉，管理员核实后会帮你重置。</p>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Find',
  data () { return { nickname: '', list: [], searched: false } },
  methods: {
    doFind () {
      api.get('/auth/find', { params: { nickname: this.nickname } }).then(r => {
        this.searched = true
        this.list = r.code === 0 ? r.data : []
      })
    }
  }
}
</script>
