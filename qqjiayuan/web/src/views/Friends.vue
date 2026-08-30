<template>
  <div>
    <div class="bar">【好友】</div>
    <div class="module-title">【收到的好友申请】({{ requests.length }})</div>
    <ul class="dtuser" v-if="requests.length">
      <li v-for="r in requests" :key="'r'+r.apply_id">
        <a href="javascript:;" @click="$router.push('/user/'+r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a>
        请求加你为好友（{{ fmt(r.created_at) }}）<br>
        <button class="btn small" @click="handle(r.apply_id, 'accept')">同 意</button>
        <button class="btn small gray" @click="handle(r.apply_id, 'reject')">拒 绝</button>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">暂无申请</span></div>

    <div class="module-title">【我的好友】({{ friends.length }})</div>
    <ul class="dtuser" v-if="friends.length">
      <li v-for="f in friends" :key="'f'+f.id">
        <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a>
        Lv.{{ f.level }} {{ f.sign }}<br>
        <button class="btn small" @click="$router.push('/messages/'+f.id)">发私信</button>
        <button class="btn small gray" @click="remove(f.id)">删除好友</button>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">还没有好友，快去广场认识新朋友</span></div>

    <div class="module-title">【添加好友】</div>
    <div class="module-content">
      <form @submit.prevent="add">
        <div class="form-item">
          <label>对方家园号码:</label>
          <input type="text" v-model.trim="targetNo" placeholder="例如 10002">
        </div>
        <div class="form-item"><button class="btn" type="submit">发送好友申请</button></div>
      </form>
      <p v-if="tip" style="color:#c00">{{ tip }}</p>
      <p v-if="okTip" style="color:#1a9e1a">{{ okTip }}</p>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Friends',
  data () { return { friends: [], requests: [], targetNo: '', tip: '', okTip: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/friends').then(r => {
        if (r.code === 0) {
          this.friends = r.data.friends
          this.requests = r.data.requests
        }
      })
    },
    add () {
      this.tip = ''
      this.okTip = ''
      api.post('/friends', { target_id: parseInt(this.targetNo) }).then(r => {
        if (r.code === 0) {
          this.okTip = typeof r.data === 'string' ? r.data : '申请已发送'
          this.load()
        } else this.tip = r.msg
      })
    },
    handle (id, action) {
      api.post(`/friends/${id}/handle`, { action }).then(r => {
        alert(r.code === 0 ? r.data : r.msg)
        this.load()
      })
    },
    remove (id) {
      if (!confirm('确定删除这位好友吗？')) return
      api.delete('/friends/' + id).then(r => {
        if (r.code === 0) this.load()
        else alert(r.msg)
      })
    },
    fmt (t) { return new Date(t).toISOString().slice(0, 10) }
  }
}
</script>
