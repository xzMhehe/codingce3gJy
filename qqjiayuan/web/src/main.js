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

// 改为函数式组件：每次父组件更新都会重新执行 render，
// 否则数据（如帖子楼主信息）异步加载后槽内文字变化不会触发重渲染，一直卡在首次的「?」
const DEFAULT_NICK_COLOR = '#004299'
const ntComponent = {
  name: 'NickText',
  functional: true,
  render (h, ctx) {
    const txt = (ctx.children || []).map(v => v.text || '').join('')
    const cs = ((ctx.props.color) || '').split(',').map(s => s.trim()).filter(s => /^#[0-9a-f]{3,6}$/i.test(s))
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
