<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/friends')">好友列表</a>&gt;分组管理</div>

    <div class="module-title">【分组管理】<a href="javascript:;" @click="showAdd = !showAdd">添加分组</a></div>
    <div class="module-content" v-if="showAdd">
      分组名：<input type="text" v-model.trim="newName" maxlength="12" size="12">
      排序：<input type="text" v-model.number="newSort" size="3" placeholder="0">
      <button class="btn" @click="createGroup">添加</button>
      <div class="help-line">最多10个分组，名称 1-12 个字，排序 0-99。</div>
    </div>

    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>

    <div class="module-title">我的分组</div>
    <div class="module-content" v-if="groups.length">
      <div v-for="(g,gi) in groups" :key="g.id">
        <a href="javascript:;" @click="toggle(g.id)">{{ gi+1 }}.{{ g.name }}</a>（{{ g.count }}人）
        <a href="javascript:;" style="color:#c00" @click="del(g.id)">删除</a>
        <div v-if="openGroup === g.id" style="margin-left:10px">
          <div v-for="m in g.members" :key="'m'+m.id" class="row">
            <a href="javascript:;" @click="$router.push('/user/'+m.id)"><font :color="m.color || '#004299'">{{ m.nickname }}</font></a>
            [<a href="javascript:;" @click="removeFriend(g.id, m.id)">移除</a>]
          </div>
          <div class="module-content">
            加入好友：
            <select v-model.number="selFriends[g.id]">
              <option :value="0">选择好友</option>
              <option v-for="f in friends" :key="f.id" :value="f.id">{{ f.nickname }}</option>
            </select>
            <button class="btn" @click="addFriend(g.id)">加入</button>
          </div>
        </div>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">还没有分组，点「添加分组」创建一个</span></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Group',
  data () {
    return { groups: [], friends: [], newName: '', newSort: 0, showAdd: false, openGroup: 0, selFriends: {}, msg: '', okMsg: '' }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/friend-groups').then(r => { if (r.code === 0) this.groups = r.data })
      api.get('/friends').then(r => { if (r.code === 0) this.friends = r.data.friends })
    },
    toggle (id) { this.openGroup = this.openGroup === id ? 0 : id },
    createGroup () {
      this.msg = ''; this.okMsg = ''
      if (!this.newName) { this.msg = '请填写分组名'; return }
      api.post('/friend-groups', { name: this.newName, sort: this.newSort }).then(r => {
        if (r.code === 0) { this.okMsg = '分组已添加'; this.newName = ''; this.newSort = 0; this.showAdd = false; this.load() }
        else this.msg = r.msg
      })
    },
    del (id) {
      api.delete('/friend-groups/' + id).then(r => { if (r.code === 0) { this.okMsg = '已删除分组'; this.load() } else this.msg = r.msg })
    },
    addFriend (gid) {
      const fid = this.selFriends[gid] || 0
      if (!fid) { this.msg = '请选择好友'; return }
      api.post('/friend-groups/' + gid + '/friends', { friend_id: fid }).then(r => {
        if (r.code === 0) { this.okMsg = '已加入分组'; this.selFriends[gid] = 0; this.load() }
        else this.msg = r.msg
      })
    },
    removeFriend (gid, fid) {
      api.delete('/friend-groups/' + gid + '/friends/' + fid).then(r => { if (r.code === 0) { this.okMsg = '已移除'; this.load() } else this.msg = r.msg })
    }
  }
}
</script>
