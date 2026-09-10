<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/tool')">工具</a>&gt;{{ titles[name] || '工具' }}<br></div>

    <!-- 姓名/翻译/手机/IP/天气：输入查询 -->
    <template v-if="name === 'name'">
      <div class="module-content">您的姓名：<br>
        <input type="text" v-model.trim="word" maxlength="8"> <button class="btn small" @click="doQuery">评分</button><br>
      </div>
      <div class="module-content" v-if="result">
        <b>评分结果</b><br>
        {{ result.name }}：<font color="#ff0000"><b>{{ result.score }}分</b></font><br>
        个性：{{ result.persona }}<br>
        事业：{{ result.career }}<br>
        情缘：{{ result.love }}<br>
        （此打分仅供娱乐）<br>
      </div>
      <div class="module-content" v-if="err"><font color="#ff0000">{{ err }}</font><br></div>
    </template>

    <template v-else-if="name === 'translate'">
      <div class="module-content">请输入中英文（自动互译）：<br>
        <input type="text" v-model.trim="word" maxlength="200"> <button class="btn small" @click="doQuery">翻译</button><br>
      </div>
      <div class="module-content" v-if="result">
        <b>翻译结果</b><br>
        {{ result.translated }}<br>
      </div>
      <div class="module-content" v-if="err"><font color="#ff0000">{{ err }}</font><br></div>
    </template>

    <template v-else-if="name === 'phone'">
      <div class="module-content">请输入手机号码:<br>
        <input type="text" v-model.trim="word" maxlength="11"> <button class="btn small" @click="doQuery">查询</button><br>
      </div>
      <div class="module-content" v-if="result">
        <b>查询结果</b><br>
        手机号码：{{ result.phone }}<br>
        运营商：{{ result.corp || '未知' }}<br>
        <template v-if="result.province">归属省份：{{ result.province }}<br>
        归属城市：{{ result.city }}<br></template>
        <template v-if="result.note"><font color="#999999">{{ result.note }}</font><br></template>
      </div>
      <div class="module-content" v-if="err"><font color="#ff0000">{{ err }}</font><br></div>
    </template>

    <template v-else-if="name === 'ip'">
      <div class="module-content">IP地址或者域名：<br>
        <input type="text" v-model.trim="word" :placeholder="'留空查本机 ' + selfIp" maxlength="60"> <button class="btn small" @click="doQuery">查询</button><br>
      </div>
      <div class="module-content" v-if="result">
        <b>查询结果</b><br>
        IP 地址：{{ result.ip }}<br>
        归属地：{{ result.location }}<br>
        运营商：{{ result.isp }}<br>
      </div>
      <div class="module-content" v-if="err"><font color="#ff0000">{{ err }}</font><br></div>
    </template>

    <template v-else-if="name === 'weather'">
      <div class="module-content">请输入城市名或区号:<br>
        <input type="text" v-model.trim="word" maxlength="20"> <button class="btn small" @click="doQuery">查询天气</button><br>
        快捷：<span v-for="(ct, i) in quickCities" :key="'qc'+ct"><a href="javascript:;" @click="word = ct; doQuery()">{{ ct }}</a>{{ i < quickCities.length - 1 ? '.' : '' }}</span><br>
      </div>
      <div class="module-content" v-if="result">
        <b>{{ result.city }}天气</b><br>
        当前：{{ result.current.weather }} {{ result.current.temp }}℃，湿度{{ result.current.humidity }}%，风速{{ result.current.wind }}m/s<br>
        <template v-for="d in result.daily"><br>{{ d.date }} {{ d.weather }} {{ d.min }}~{{ d.max }}℃</template><br>
      </div>
      <div class="module-content" v-if="err"><font color="#ff0000">{{ err }}</font><br></div>
    </template>

    <!-- 编码转换（本地，诺哈 code.asp：十进制/十六进制实体） -->
    <template v-else-if="name === 'code'">
      <div class="module-content">请输入:<br>
        <input type="text" v-model.trim="word" maxlength="100"> <button class="btn small" @click="doCode">查询</button><br>
      </div>
      <div class="module-content" v-if="codeOut">
        十进制：{{ codeOut.dec }}<br>
        十六进制：{{ codeOut.hex }}<br>
      </div>
    </template>

    <!-- 客户信息（复刻 client.asp） -->
    <template v-else-if="name === 'client'">
      <div class="module-content">
        【客户信息】<br>
        访问ＩＰ：{{ info.ip }}<br>
        来源地址：{{ info.referer || '直接访问' }}<br>
        浏览工具：{{ info.ua }}<br>
        接受文档：{{ info.accept }}<br>
        屏幕分辨率：{{ screenW }}×{{ screenH }}<br>
      </div>
    </template>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/tool')">便民中心</a>&gt;{{ titles[name] || '工具' }}<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ToolPage',
  data () {
    return {
      name: '', word: '', result: null, err: '', info: {}, codeOut: null,
      screenW: window.screen.width, screenH: window.screen.height,
      quickCities: ['北京', '上海', '广州', '深圳', '郑州', '成都', '杭州'],
      titles: { name: '姓名评分', translate: '英汉互译', phone: '手机归属', ip: 'ＩＰ查询', weather: '天气预报', code: '编码转换', client: '客户信息' }
    }
  },
  computed: {
    selfIp () { return this.info.ip || '' }
  },
  watch: { '$route': 'init' },
  mounted () { this.init() },
  methods: {
    init () {
      this.name = this.$route.params.name || ''
      this.result = null
      this.err = ''
      this.codeOut = null
      this.word = ''
      if (this.name === 'client') {
        api.get('/tool/client').then(r => { if (r.code === 0) this.info = r.data || {} })
      } else if (this.name === 'ip') {
        // 预取本机 IP 便于留空查询
        api.get('/tool/client').then(r => { if (r.code === 0) this.info = r.data || {} })
      }
    },
    doQuery () {
      this.result = null
      this.err = ''
      if (!this.word) return
      const url = '/tool/' + this.name
      const param = this.name === 'name' ? { name: this.word }
        : this.name === 'translate' ? { text: this.word }
          : this.name === 'phone' ? { phone: this.word }
            : this.name === 'ip' ? { ip: this.word || '' } : { city: this.word }
      api.get(url, { params: param }).then(r => {
        if (r.code === 0) this.result = r.data
        else this.err = r.msg || '系统繁忙，请稍候再试。'
      })
    },
    doCode () {
      this.codeOut = null
      if (!this.word) return
      let dec = ''
      let hex = ''
      for (const ch of this.word) {
        const n = ch.codePointAt(0)
        dec += '&#' + n + ';'
        hex += '&#x' + n.toString(16) + ';'
      }
      this.codeOut = { dec, hex }
    }
  }
}
</script>
