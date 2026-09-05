<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的百宝箱&gt;编辑资料</div>

    <form @submit.prevent="save">
      <div class="module-title">编辑资料</div>
      <div class="module-content">
        昵称:<input type="text" name="name" v-model.trim="form.nickname" emptyok="true" maxlength="12" style="width:140px"><br>
        性别:<select name="gender" v-model.number="form.gender">
          <option :value="1">男</option>
          <option :value="2">女</option>
        </select><br>
        年龄:<input type="text" name="age" v-model.number="form.age" emptyok="true" maxlength="2" size="2" style="width:44px">
        <span class="txt-fade">（留空则按生日自动计算）</span><br>
        生日:<input type="text" name="y" v-model.number="form.birth_year" emptyok="true" maxlength="4" size="3" style="width:55px">年
          <input type="text" name="m" v-model.number="form.birth_month" emptyok="true" maxlength="2" size="1" style="width:32px">月
          <input type="text" name="d" v-model.number="form.birth_day" emptyok="true" maxlength="2" size="1" style="width:32px">日
        <span class="txt-fade">（填完整生日自动算年龄）</span><br>
        签名:<input type="text" name="sign" v-model.trim="form.signature" maxlength="120" style="width:180px"><br>
        简介:<input type="text" name="intro" v-model.trim="form.introduction" maxlength="200" style="width:180px"><br>
        城市:<a href="javascript:;" @click="tip('城市设置')">去设置</a><br>
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
        birth_year: 0, birth_month: 0, birth_day: 0,
        signature: '', introduction: '' },
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
        this.form = {
          nickname: d.nickname || '',
          gender: d.gender || 1,
          age: d.age || 0,
          birth_year: d.birth_year || 0,
          birth_month: d.birth_month || 0,
          birth_day: d.birth_day || 0,
          signature: d.signature || '',
          introduction: d.introduction || ''
        }
      }
    })
  },
  methods: {
    // 生日三字段齐全 → 自动算年龄
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
    // 只填年龄 → 推出生年份（保留已有月日，缺省用当前月日）
    syncBirthFromAge () {
      const age = Number(this.form.age) || 0
      if (!(age > 0 && age < 150)) return
      if (Number(this.form.birth_year) > 0) return
      const now = new Date()
      this.form.birth_year = now.getFullYear() - age
      if (!Number(this.form.birth_month)) this.form.birth_month = now.getMonth() + 1
      if (!Number(this.form.birth_day)) this.form.birth_day = now.getDate()
    },
    // 提交前规范化：空输入一律转 0，避免后端 int 解析失败
    normalize () {
      return {
        nickname: this.form.nickname,
        gender: Number(this.form.gender) === 2 ? 2 : 1,
        age: Number(this.form.age) || 0,
        birth_year: Number(this.form.birth_year) || 0,
        birth_month: Number(this.form.birth_month) || 0,
        birth_day: Number(this.form.birth_day) || 0,
        signature: this.form.signature || '',
        introduction: this.form.introduction || ''
      }
    },
    save () {
      const body = this.normalize()
      // 保存前再兜底算一次联动，保证库里年月日/年龄一致
      this.syncAgeFromBirth()
      this.syncBirthFromAge()
      body.age = Number(this.form.age) || 0
      body.birth_year = Number(this.form.birth_year) || 0
      body.birth_month = Number(this.form.birth_month) || 0
      body.birth_day = Number(this.form.birth_day) || 0
      api.put('/users/me', body).then(r => {
        if (r.code === 0) {
          this.okTip = '资料已保存'
          this.tip = ''
          api.get('/auth/me').then(m => {
            if (m.code === 0) {
              const d = m.data
              this.form = {
                nickname: d.nickname || '',
                gender: d.gender || 1,
                age: d.age || 0,
                birth_year: d.birth_year || 0,
                birth_month: d.birth_month || 0,
                birth_day: d.birth_day || 0,
                signature: d.signature || '',
                introduction: d.introduction || ''
              }
            }
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
