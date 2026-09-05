<template>
  <div>
    <div class="bar">【个人主页】</div>
    <div class="module-content">
      <p>
        昵称：<b><font :color="u.color || '#004299'">{{ u.nickname }}</font></b>
        （{{ u.username }}）Lv.{{ u.level }} · G币 {{ u.coins }} · 经验 {{ u.exp }}
      </p>
      <p>签名：{{ u.signature || '这个人很懒，什么都没留下' }}</p>
      <p>性别：{{ u.gender === 2 ? '小姐姐' : '小哥哥' }} · 注册于 {{ fmt(u.created_at) }}</p>
      <p>角色：<span class="tag" v-for="r in (u.roles || [])" :key="r.id" style="margin-right:4px">{{ r.name }}</span></p>
    </div>
    <div class="module-title">【修改资料】</div>
    <div class="module-content">
      <form @submit.prevent="save">
        <div class="form-item">
          <label>昵称:</label>
          <input type="text" v-model.trim="form.nickname" maxlength="20">
        </div>
        <div class="form-item">
          <label>个性签名（50字内）:</label>
          <input type="text" v-model.trim="form.signature" maxlength="50">
        </div>
        <div class="form-item">
          <label>昵称颜色（情怀功能，如 #DAA520）:</label>
          <input type="text" v-model.trim="form.color" maxlength="10" placeholder="#004299">
        </div>
        <div class="form-item">
          <label>头像:</label>
          <div>
            <label v-for="av in avatars" :key="av" style="display:inline-block;margin:2px 6px 2px 0;cursor:pointer">
              <input type="radio" name="avatar" :value="av" v-model="form.avatar">
              <img :src="$pic(av)" width="48" height="48" alt="头像" style="vertical-align:middle;border-radius:4px">
            </label>
          </div>
        </div>
        <div class="form-item">
          <label>性别:</label>
          <select v-model.number="form.gender">
            <option :value="1">小哥哥</option>
            <option :value="2">小姐姐</option>
          </select>
        </div>
        <div class="form-item"><button class="btn" type="submit">保存资料</button></div>
      </form>
      <p v-if="tip" style="color:#c00">{{ tip }}</p>
      <p v-if="okTip" style="color:#1a9e1a">{{ okTip }}</p>
    </div>
    <div class="module-title">【修改密码】</div>
    <div class="module-content">
      <form @submit.prevent="changePwd">
        <div class="form-item">
          <label>原密码:</label>
          <input type="password" v-model.trim="pwd.old" maxlength="20">
        </div>
        <div class="form-item">
          <label>新密码（6-20位）:</label>
          <input type="password" v-model.trim="pwd.new" maxlength="20">
        </div>
        <div class="form-item"><button class="btn gray" type="submit">修改密码</button></div>
      </form>
      <p v-if="pwdTip" style="color:#c00">{{ pwdTip }}</p>
      <p v-if="pwdOk" style="color:#1a9e1a">{{ pwdOk }}</p>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Profile',
  data () {
    return {
      u: {},
      form: { nickname: '', signature: '', color: '', gender: 1, avatar: '' },
      avatars: [],
      pwd: { old: '', new: '' },
      tip: '', okTip: '', pwdTip: '', pwdOk: ''
    }
  },
  mounted () {
    this.load()
    api.get('/badge-presets').then(r => {
      if (r.code === 0) this.avatars = r.data.avatars
    })
  },
  methods: {
    load () {
      api.get('/auth/me').then(r => {
        if (r.code === 0) {
          this.u = r.data
          this.form = {
            nickname: r.data.nickname, signature: r.data.signature,
            color: r.data.color, gender: r.data.gender, avatar: r.data.avatar || ''
          }
        }
      })
    },
    save () {
      api.put('/users/me', this.form).then(r => {
        if (r.code === 0) {
          this.okTip = '资料已保存'
          this.tip = ''
          this.load()
          this.$store.commit('setUser', { user: null })
          api.get('/auth/me').then(m => {
            if (m.code === 0) this.$store.commit('setUser', { user: m.data })
          })
        } else {
          this.tip = r.msg
          this.okTip = ''
        }
      })
    },
    changePwd () {
      api.put('/auth/password', this.pwd).then(r => {
        if (r.code === 0) {
          this.pwdOk = '密码已修改'
          this.pwdTip = ''
          this.pwd = { old: '', new: '' }
        } else {
          this.pwdTip = r.msg
          this.pwdOk = ''
        }
      })
    },
    fmt (t) { return t ? new Date(t).toISOString().slice(0, 10) : '' }
  }
}
</script>
