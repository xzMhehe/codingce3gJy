<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;婚恋中心<br></div>

    <!-- 我的婚恋状态 -->
    <div class="module-title">【我的婚恋状态】<a href="javascript:;" @click="load">刷新</a></div>
    <div class="module-content">
      <template v-if="status.married">
        <font color="#e05a00">已婚</font>，伴侣：<a href="javascript:;" @click="$router.push('/user/'+status.partner.id)"><font :color="status.partner.color || '#004299'">{{ status.partner.nickname }}</font></a>
        <template v-if="status.baby">，宝宝：{{ status.baby }}</template>
        <br>
        <a href="javascript:;" @click="divorce" style="color:#c00">离婚（999G币）</a><br>
      </template>
      <template v-else>
        <span class="txt-fade">单身贵族</span><br>
      </template>
    </div>

    <!-- 求婚 -->
    <div class="module-title">【求婚】（需要 999 G币）</div>
    <div class="module-content" v-if="!status.married">
      求婚号码：<input type="text" v-model.number="proposeTo" size="10" maxlength="10"><br>
      真情表白：<input type="text" v-model.trim="proposeMsg" maxlength="200" size="24"><br>
      <input type="submit" value="确定求婚" @click="propose"><br>
      <span class="help-line">温馨提示：求婚需要 999 G币；双方须均为单身。</span><br>
    </div>
    <div class="module-content" v-else><span class="txt-fade">已婚状态下不能求婚</span></div>

    <!-- 收到的求婚 -->
    <div class="module-title">【收到的求婚】({{ status.incoming.length }})</div>
    <div class="module-content" v-if="status.incoming.length">
      <div v-for="p in status.incoming" :key="'i'+p.id">
        <a href="javascript:;" @click="$router.push('/user/'+p.from_id)"><font :color="p.color || '#004299'">{{ p.from }}</font></a> 向您求婚！<br>
        <span class="txt-fade">表白：{{ p.message }}（{{ fmt(p.created_at) }}）</span><br>
        <a href="javascript:;" @click="handle(p.id, 'accept')">同意</a>.<a href="javascript:;" @click="handle(p.id, 'reject')">拒绝</a><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无收到的求婚</span></div>

    <!-- 我发出的求婚 -->
    <div class="module-title">【我发出的求婚】({{ status.outgoing.length }})</div>
    <div class="module-content" v-if="status.outgoing.length">
      <div v-for="p in status.outgoing" :key="'o'+p.id">
        向 <a href="javascript:;" @click="$router.push('/user/'+p.to_id)"><font :color="p.color || '#004299'">{{ p.to }}</font></a> 求婚（等待对方处理，{{ fmt(p.created_at) }}）<br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无发出的求婚</span></div>

    <!-- 已婚列表 -->
    <div class="module-title">【已婚伴侣】</div>
    <div class="module-content" v-if="couples.length">
      <div v-for="cp in couples" :key="'c'+cp.id">
        <a href="javascript:;" @click="$router.push('/user/'+cp.a.id)"><font :color="cp.a.color || '#004299'">{{ cp.a.nickname }}</font></a>
        ❤
        <a href="javascript:;" @click="$router.push('/user/'+cp.b.id)"><font :color="cp.b.color || '#004299'">{{ cp.b.nickname }}</font></a>
        <span class="txt-fade">（{{ fmt(cp.created_at) }}）</span><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无已婚伴侣</span></div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;婚恋中心<br></div>

    <p style="color:#1a9e1a;padding:2px 5px" v-if="okMsg">{{ okMsg }}</p>
    <p style="color:#c00;padding:2px 5px" v-if="msg">{{ msg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Marriage',
  data () {
    return { status: { married: false, partner: {}, baby: '', incoming: [], outgoing: [] }, couples: [], proposeTo: null, proposeMsg: '', okMsg: '', msg: '' }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/marriage/status').then(r => { if (r.code === 0) this.status = r.data })
      api.get('/marriage/list').then(r => { if (r.code === 0) this.couples = r.data || [] })
    },
    propose () {
      this.msg = ''; this.okMsg = ''
      if (!this.proposeTo) { this.msg = '请输入求婚号码'; return }
      if (!this.proposeMsg) { this.msg = '不表白点什么？'; return }
      api.post('/marriage/propose', { to: this.proposeTo, message: this.proposeMsg }).then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.proposeMsg = ''; this.load() } else this.msg = r.msg
      })
    },
    handle (id, action) {
      api.post('/marriage/' + id + '/handle', { action }).then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.msg = ''; this.load() } else this.msg = r.msg
      })
    },
    divorce () {
      if (!window.confirm('确定要离婚吗？需要 999 G币手续费。')) return
      api.post('/marriage/divorce').then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.load() } else this.msg = r.msg
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>