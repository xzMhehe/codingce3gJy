import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import './assets/style.css'

Vue.config.productionTip = false

// 演示站图片素材路径（static 已整体拷贝到 public/static）
// 库存图片（管理端上传，存数据库 base64）：db/xxx.gif → /api/res/db/xxx.gif
Vue.prototype.$pic = f => {
  if (!f) return ''
  if (f.indexOf('db/') === 0) return '/api/res/' + f
  return '/static/picture/' + f
}

// nt组件：昵称颜色渲染。color 传单色时整个昵称一个颜色；
// 传「#D2B48C,#800080,#DAA520,...」逗号分隔序列时逐字变色（复刻诺哈招牌逐字彩色昵称）
const DEFAULT_NICK_COLOR = '#004299'
const ntComponent = {
  name: 'NickText',
  props: { color: { type: String, default: '' } },
  computed: {
    pieces () {
      const slot = this.$slots.default || []
      return slot.map(v => v.text || '').join('')
    }
  },
  render (h) {
    const txt = this.pieces
    const cs = (this.color || '').split(',').map(s => s.trim()).filter(s => /^#[0-9a-f]{3,6}$/.test(s))
    if (cs.length <= 1) {
      return h('font', { attrs: { color: cs[0] || DEFAULT_NICK_COLOR } }, txt)
    }
    const chars = Array.from(txt)
    return h('span', chars.map((ch, i) => h('font', { attrs: { color: cs[i % cs.length] } }, ch)))
  }
}
Vue.component('ntext', ntComponent)

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
