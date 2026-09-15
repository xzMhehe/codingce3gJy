<template>
  <div>
    <!-- 城市设置（复刻诺哈 wap_city 设置省市：选省份→选城市→保存） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/box')">我的百宝箱</a>&gt;城市设置<br></div>

    <div class="module-title">【城市设置】</div>
    <div class="module-content">
      当前设置：<ntext v-if="savedProvince" style="font-weight:bold">{{ savedProvince }} · {{ savedCity || '全省' }}</ntext><span v-else class="txt-fade">尚未设置城市</span><br>
      <span class="txt-fade" v-if="savedCity">设置后同城会按你的城市推荐老乡。</span><br>
    </div>

    <template v-if="cur === 'prov'">
      <div class="module-title">请选择省份 ({{ provinces.length }})</div>
      <div class="list">
        <div class="row" v-for="(p, i) in provinces" :key="p.id">
          {{ i + 1 }}.<a href="javascript:;" @click="pickProv(p)">{{ p.name }}</a>({{ p.city_count }}城)<br>
        </div>
        <div class="row" v-if="!provinces.length">暂无省份资料，请先让管理员在后台维护同城数据。<br></div>
      </div>
    </template>

    <template v-else-if="cur === 'city'">
      <div class="bar sub"><a href="javascript:;" @click="goProv">选择省份</a>&gt;{{ prov.name }}<br></div>
      <div class="module-title">【{{ prov.name }}】选择城市</div>
      <div class="list">
        <div class="row" v-for="(c, i) in cities" :key="c.id">
          {{ i + 1 }}.<a href="javascript:;" @click="pickCity(c)">{{ c.name }}</a><template v-if="c.city_code">({{ c.city_code }})</template><template v-if="c.online > 0"><font color="#1a9e1a">[在线{{ c.online }}]</font></template><br>
        </div>
        <div class="row" v-if="!cities.length">该省份暂无城市数据。<br></div>
      </div>
    </template>

    <div class="okmsg" v-if="okMsg">{{ okMsg }}</div>
    <div class="errmsg" v-if="msg">{{ msg }}</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'CitySet',
  data () { return { cur: 'prov', me: {}, provinces: [], prov: {}, cities: [], savedProvince: '', savedCity: '', okMsg: '', msg: '' } },
  mounted () { this.load() },
  watch: {
    okMsg (v) { if (this._ok) clearTimeout(this._ok); if (v) this._ok = setTimeout(() => { this.okMsg = '' }, 4000) },
    msg (v) { if (this._m) clearTimeout(this._m); if (v) this._m = setTimeout(() => { this.msg = '' }, 4000) }
  },
  beforeDestroy () { if (this._ok) clearTimeout(this._ok); if (this._m) clearTimeout(this._m) },
  methods: {
    load () {
      // 当前设置回显（表单保存时也需要这些字段）
      api.get('/auth/me').then(r => {
        if (r.code === 0) {
          this.me = r.data
          const c = r.data.city || ''
          if (c) {
            const parts = c.split(' ')
            this.savedProvince = parts[0] || ''
            this.savedCity = parts[1] || ''
          }
        }
      }).catch(() => {})
      api.get('/tongcheng').then(r => {
        if (r.code === 0) this.provinces = r.data.provinces || []
      })
    },
    pickProv (p) {
      this.prov = p
      api.get('/tongcheng/province/' + p.id).then(r => {
        if (r.code === 0) { this.cities = r.data.list || []; this.cur = 'city' } else this.msg = r.msg
      })
    },
    goProv () { this.cur = 'prov' },
    pickCity (c) {
      const name = [this.prov.name, c.name].filter(Boolean).join(' ')
      // 乐观更新：点完城市"当前设置"立即就位，接口只是后台确认
      this.savedProvince = this.prov.name
      this.savedCity = c.name
      if (this.$store.state.user) {
        this.$store.commit('setUser', {
          token: this.$store.state.token,
          user: Object.assign({}, this.$store.state.user, { city: name })
        })
      }
      // 专用接口只更新 city 字段，一次请求原子生效，不依赖资料快照
      api.put('/me/city', { city: name }).then(r => {
        if (r.code === 0) {
          this.okMsg = '城市设置成功：' + name
          this.cur = 'prov'
        } else this.msg = r.msg || '保存失败，请重试'
      }).catch(() => { this.msg = '网络异常，保存失败' })
    }
  }
}
</script>
