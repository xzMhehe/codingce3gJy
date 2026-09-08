<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的百宝箱&gt;编辑资料</div>

    <!-- 会员资料（对齐诺哈 profile_edit：昵称/性别/年龄/生日类型/阳历阴历/签名/简介） -->
    <form @submit.prevent="save">
      <div class="module-title">会员资料</div>
      <div class="module-content">
        昵称:<input type="text" v-model.trim="form.nickname" maxlength="12" style="width:140px"><br>
        性别:<select v-model.number="form.gender">
          <option :value="1">男</option>
          <option :value="2">女</option>
        </select><br>
        年龄:<input type="text" v-model.number="form.age" maxlength="2" size="2" style="width:44px">
        <span class="txt-fade">（留空则按生日自动计算）</span><br>
        生日类型:<select v-model.number="form.birth_type">
          <option :value="1">阳历</option>
          <option :value="0">阴历</option>
        </select><br>
        阳历生日:<input type="text" v-model.number="form.birth_year" maxlength="4" size="3" style="width:55px">年
          <input type="text" v-model.number="form.birth_month" maxlength="2" size="1" style="width:32px">月
          <input type="text" v-model.number="form.birth_day" maxlength="2" size="1" style="width:32px">日
        <span class="txt-fade">（填完整生日自动算年龄）</span><br>
        阴历生日:<input type="text" v-model.trim="form.lunar" maxlength="20" style="width:120px" placeholder="如：腊月初八"><br>
        签名:<input type="text" v-model.trim="form.signature" maxlength="120" style="width:180px"><br>
        简介:<input type="text" v-model.trim="form.introduction" maxlength="200" style="width:180px"><br>
        城市:<a href="javascript:;" @click="tip('城市设置')">去设置</a><br>
      </div>

      <!-- 通信地址（对齐诺哈 address.asp：故乡/现居） -->
      <div class="module-title">通信地址</div>
      <div class="module-content">
        <b>故乡</b><br>
        国家:<input type="text" v-model.trim="addr.home_nation" maxlength="20" style="width:60px">
        省:<input type="text" v-model.trim="addr.home_prov" maxlength="20" style="width:60px">
        市:<input type="text" v-model.trim="addr.home_city" maxlength="20" style="width:60px"><br>
        区县:<input type="text" v-model.trim="addr.home_dist" maxlength="20" style="width:60px">
        地址:<input type="text" v-model.trim="addr.home_addr" maxlength="100" style="width:120px"><br>
        邮编:<input type="text" v-model.trim="addr.home_zip" maxlength="10" style="width:60px"><br>
        <b>现居</b><br>
        国家:<input type="text" v-model.trim="addr.live_nation" maxlength="20" style="width:60px">
        省:<input type="text" v-model.trim="addr.live_prov" maxlength="20" style="width:60px">
        市:<input type="text" v-model.trim="addr.live_city" maxlength="20" style="width:60px"><br>
        区县:<input type="text" v-model.trim="addr.live_dist" maxlength="20" style="width:60px">
        地址:<input type="text" v-model.trim="addr.live_addr" maxlength="100" style="width:120px"><br>
        邮编:<input type="text" v-model.trim="addr.live_zip" maxlength="10" style="width:60px"><br>
      </div>

      <!-- 好友策略 / 个性设置（对齐诺哈 wap_user.friend / config） -->
      <div class="module-title">个人设置</div>
      <div class="module-content">
        加好友:<select v-model.number="form.friend_policy">
          <option :value="0">允许任何人</option>
          <option :value="1">需要验证</option>
          <option :value="2">拒绝任何人</option>
        </select><br>
        每页帖子数:<select v-model.number="form.per_page">
          <option v-for="n in [5,10,15,20]" :key="n" :value="n">{{ n }} 条</option>
        </select><br>
        <input type="submit" value=" 确定编辑 ">
      </div>
    </form>

    <p v-if="tip" style="color:#c00">{{ tip }}</p>
    <p v-if="okTip" style="color:#1a9e1a">{{ okTip }}</p>

    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的百宝箱&gt;编辑资料</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Profile',
  data () {
    return {
      form: { nickname: '', gender: 1, age: 0,
        birth_year: 0, birth_month: 0, birth_day: 0, birth_type: 1, lunar: '',
        signature: '', introduction: '', friend_policy: 0, per_page: 10 },
      addr: { home_nation: '中国', home_prov: '', home_city: '', home_dist: '', home_addr: '', home_zip: '',
        live_nation: '中国', live_prov: '', live_city: '', live_dist: '', live_addr: '', live_zip: '' },
      tip: '', okTip: ''
    }
  },
  watch: {
    'form.birth_year' () { this.syncAgeFromBirth() },
    'form.birth_month' () { this.syncAgeFromBirth() },
    'form.birth_day' () { this.syncAgeFromBirth() },
    'form.age' () { this.syncBirthFromAge() }
  },
  mounted () {
    api.get('/auth/me').then(r => {
      if (r.code === 0) {
        const d = r.data
        let perPage = 10
        if (d.config) { const p = parseInt(d.config.split(',')[0]); if (p >= 5 && p <= 20) perPage = p }
        this.form = {
          nickname: d.nickname || '',
          gender: d.gender || 1,
          age: d.age || 0,
          birth_year: d.birth_year || 0,
          birth_month: d.birth_month || 0,
          birth_day: d.birth_day || 0,
          birth_type: d.birth_type === 0 ? 0 : 1,
          lunar: d.lunar || '',
          signature: d.signature || '',
          introduction: d.introduction || '',
          friend_policy: d.friend_policy || 0,
          per_page: perPage
        }
      }
    })
    api.get('/me/address').then(r => {
      if (r.code === 0 && r.data && r.data.user_id) this.addr = { ...this.addr, ...r.data }
    })
  },
  methods: {
    syncAgeFromBirth () {
      const y = Number(this.form.birth_year) || 0
      const m = Number(this.form.birth_month) || 0
      const d = Number(this.form.birth_day) || 0
      if (!(y > 0 && m > 0 && d > 0)) return
      const now = new Date()
      let age = now.getFullYear() - y
      if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age--
      if (age >= 0) this.form.age = age
    },
    syncBirthFromAge () {
      const age = Number(this.form.age) || 0
      if (!(age > 0 && age < 150)) return
      if (Number(this.form.birth_year) > 0) return
      const now = new Date()
      this.form.birth_year = now.getFullYear() - age
      if (!Number(this.form.birth_month)) this.form.birth_month = now.getMonth() + 1
      if (!Number(this.form.birth_day)) this.form.birth_day = now.getDate()
    },
    save () {
      this.syncAgeFromBirth()
      this.syncBirthFromAge()
      const solar = (this.form.birth_year && this.form.birth_month && this.form.birth_day)
        ? this.form.birth_year + '-' + this.form.birth_month + '-' + this.form.birth_day : ''
      const body = {
        nickname: this.form.nickname,
        gender: Number(this.form.gender) === 2 ? 2 : 1,
        age: Number(this.form.age) || 0,
        birth_year: Number(this.form.birth_year) || 0,
        birth_month: Number(this.form.birth_month) || 0,
        birth_day: Number(this.form.birth_day) || 0,
        birth_type: this.form.birth_type,
        solar: solar,
        lunar: this.form.lunar || '',
        signature: this.form.signature || '',
        introduction: this.form.introduction || '',
        friend_policy: this.form.friend_policy,
        per_page: this.form.per_page
      }
      api.put('/users/me', body).then(r => {
        if (r.code === 0) {
          api.put('/me/address', this.addr).then(a => {
            if (a.code === 0) { this.okTip = '资料已保存'; this.tip = '' }
            else { this.tip = a.msg || '地址保存失败'; this.okTip = '' }
          })
        } else {
          this.tip = r.msg
          this.okTip = ''
        }
      })
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('百宝箱·' + title)) }
  }
}
</script>