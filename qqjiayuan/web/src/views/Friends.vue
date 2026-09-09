<template>
  <div>
    <div class="module-title">我的好友|<a href="javascript:;" @click="tab='recent'">最近联系</a>|<a href="javascript:;" @click="loadNews();tab='news'">新鲜事</a>|<a href="javascript:;" @click="loadBlack();tab='black'">黑名单</a></div>

    <!-- ===== 好友列表 ===== -->
    <template v-if="tab === 'friends'">
      <div class="module-content" v-if="friends.length">
        <div v-for="f in friends" :key="f.id" class="row">
          <img :src="f.online ? '/static/image/home.gif' : '/static/image/home1.png'" alt="好友">
          <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ displayName(f) }}</font></a>
          <span v-if="f.remark && f.remark !== f.nickname" class="txt-fade">(备注:{{ f.remark }})</span>
          <span v-if="f.group_name" class="txt-fade">[{{ f.group_name }}]</span>
          <span :style="{ color: f.online ? '#1a9e1a' : '#999' }">{{ f.online ? '在线' : '离线' }}</span><br>
          <span class="txt-fade">功能:
            <a href="javascript:;" @click="$router.push('/messages/'+f.id)">发家信</a>.
            <a href="javascript:;" @click="editRemark(f)">备注</a>.
            <a href="javascript:;" @click="editGroup(f)">分组</a>.
            <a href="javascript:;" @click="removeFriend(f)" style="color:#c00">删除</a>.
            <a href="javascript:;" @click="blackFriend(f)" style="color:#c00">拉黑</a>
          </span>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有好友，去添加一个吧</span></div>
      <div class="module-content">(共{{ friends.length }}个好友)　头像说明：<img src="/static/image/home.gif" alt="." />在线　<img src="/static/image/home1.png" alt="." />离线</div>

      <div class="module-content">
        <form @submit.prevent="goSearch">
          输入ID号码或昵称搜索好友：
          <input type="text" v-model.trim="word" size="12" maxlength="20">
          <input type="submit" value="搜好友">
        </form>
      </div>
    </template>

    <!-- ===== 最近联系 ===== -->
    <template v-if="tab === 'recent'">
      <div class="list" v-if="conversations.length">
        <div v-for="c in conversations" :key="c.user_id" class="row">
          <img src="/static/image/home.gif" alt="好友">
          <a href="javascript:;" @click="$router.push('/messages/'+c.user_id)"><font :color="c.color || '#004299'">{{ c.nickname }}</font></a>
          <span class="txt-fade">：{{ (c.last_content || '').slice(0, 12) }}</span>
          <span class="txt-fade">（{{ fmtShort(c.last_at) }}）</span>
          [<a href="javascript:;" @click="$router.push('/messages/'+c.user_id)">发家信</a>]
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无最近联系</span></div>
    </template>

    <!-- ===== 好友新鲜事 ===== -->
    <template v-if="tab === 'news'">
      <div class="module-content" v-if="news.length">
        <div v-for="(n,i) in news" :key="n.type+'_'+n.id" class="row">
          {{ i+1 }}.{{ timeDiff(n.created_at) }} <a href="javascript:;" @click="$router.push('/user/'+n.user_id)"><font :color="n.color || '#004299'">{{ n.nickname }}</font></a>
          {{ n.type === 'thread' ? '发表帖子' : '回复帖子' }}：
          <a href="javascript:;" @click="$router.push('/thread/'+n.thread_id)">{{ n.title }}</a>
          <span class="txt-fade">（{{ n.board_name }}）</span><br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无新鲜事，多关注下好友吧</span></div>
      <div class="module-content"><a href="javascript:;" @click="loadNews">刷新</a></div>
    </template>

    <!-- ===== 黑名单 ===== -->
    <template v-if="tab === 'black'">
      <div class="module-content" v-if="blacks.length">
        <div v-for="(b,i) in blacks" :key="b.id" class="row">
          {{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+b.friend_id)"><font :color="b.color || '#004299'">{{ b.nickname }}</font></a>
          <span class="txt-fade">（{{ fmt(b.add_time) }}）</span>
          [<a href="javascript:;" @click="unblack(b)" style="color:#1a9e1a">解除</a>]<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">您没有黑名单</span></div>
      <div class="module-content txt-fade">拉黑后不再接受对方信息，并自动解除好友关系。</div>
    </template>

    <!-- 收到的好友申请 -->
    <div class="module-title">【好友请求】({{ requests.length }})</div>
    <ul class="dtuser" v-if="requests.length">
      <li v-for="r in requests" :key="'r'+r.apply_id">
        <a href="javascript:;" @click="$router.push('/user/'+r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a> ({{ r.id }})<span v-if="r.remark">（{{ r.remark }}）</span>（{{ fmt(r.created_at) }}）<br>
        <a href="javascript:;" @click="handle(r.apply_id, 'add')">互加</a>.<a href="javascript:;" @click="handle(r.apply_id, 'pass')">通过</a>.<a href="javascript:;" @click="handle(r.apply_id, 'reject')">拒绝</a>.<a href="javascript:;" @click="handle(r.apply_id, 'ignore')">忽略</a>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">暂无好友请求</span></div>

    <!-- 功能导航 -->
    <div class="module-title">功能导航</div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/inbox')">全部消息</a>|<a href="javascript:;" @click="$router.push('/groups')">分组管理</a>|<a href="javascript:;" @click="$router.push('/search')">好友查找</a><br>
      验证设置：
      <select v-model.number="policy" @change="savePolicy">
        <option :value="0">允许所有人添加</option>
        <option :value="1">通过验证后添加</option>
        <option :value="2">拒绝任何人添加</option>
      </select>
    </div>

    <!-- 备注弹窗 -->
    <div class="module-content" v-if="remarkTarget">
      <b>设置 {{ remarkTarget.nickname }} 的备注：</b>
      <input type="text" v-model.trim="remarkVal" maxlength="30" size="12">
      <button class="btn" @click="saveRemark">保存</button>
      <a href="javascript:;" @click="remarkTarget=null">取消</a>
    </div>

    <!-- 分组移动弹窗 -->
    <div class="module-content" v-if="groupTarget">
      <b>把 {{ groupTarget.nickname }} 移动到：</b>
      <select v-model.number="moveGroupId">
        <option :value="0">未分组</option>
        <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
      </select>
      <button class="btn" @click="saveGroup">移动</button>
      <a href="javascript:;" @click="groupTarget=null">取消</a>
    </div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Friends',
  data () {
    return {
      tab: 'friends',
      friends: [], conversations: [], requests: [], groups: [],
      blacks: [], news: [],
      policy: 0,
      word: '',
      remarkTarget: null, remarkVal: '',
      groupTarget: null, moveGroupId: 0,
      msg: '', okMsg: ''
    }
  },
  mounted () {
    if (this.$route.query.tab) this.tab = this.$route.query.tab
    this.load()
  },
  methods: {
    load () {
      api.get('/friends').then(r => {
        if (r.code === 0) {
          this.friends = r.data.friends || []
          this.requests = r.data.requests || []
          this.groups = r.data.groups || []
          this.policy = r.data.friend_policy
        } else this.msg = r.msg
      })
      api.get('/messages/conversations').then(r => { if (r.code === 0) this.conversations = r.data })
    },
    loadNews () { api.get('/friends/news').then(r => { if (r.code === 0) { this.news = r.data.list || [] } else { this.msg = r.msg } }) },
    loadBlack () { api.get('/friends/black').then(r => { if (r.code === 0) { this.blacks = r.data || [] } else { this.msg = r.msg } }) },
    displayName (f) { return f.remark ? f.remark : f.nickname },
    editRemark (f) { this.remarkTarget = f; this.remarkVal = f.remark || ''; this.msg = '' },
    saveRemark () {
      api.post('/friends/' + this.remarkTarget.id + '/remark', { remark: this.remarkVal }).then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.remarkTarget = null; this.load() } else this.msg = r.msg
      })
    },
    editGroup (f) { this.groupTarget = f; this.moveGroupId = f.group_id || 0 },
    saveGroup () {
      api.post('/friends/' + this.groupTarget.id + '/group', { group_id: this.moveGroupId }).then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.groupTarget = null; this.load() } else this.msg = r.msg
      })
    },
    handle (id, act) {
      api.post('/friends/apply/' + id + '/handle?act=' + act).then(r => {
        if (r.code === 0) { this.okMsg = r.data; this.load() } else this.msg = r.msg
      })
    },
    removeFriend (f) {
      if (!window.confirm('确定删除好友「' + this.displayName(f) + '」吗？')) return
      api.delete('/friends/' + f.id).then(r => { if (r.code === 0) { this.okMsg = r.data; this.load() } else this.msg = r.msg })
    },
    blackFriend (f) {
      if (!window.confirm('确定把「' + this.displayName(f) + '」加入黑名单吗？\n将不再接受对方的信息，并解除好友关系。')) return
      api.post('/friends/black/' + f.id).then(r => { if (r.code === 0) { this.okMsg = r.data; this.load() } else this.msg = r.msg })
    },
    unblack (b) {
      api.delete('/friends/black/' + b.id).then(r => { if (r.code === 0) { this.okMsg = r.data; this.loadBlack() } else this.msg = r.msg })
    },
    savePolicy () {
      api.post('/friends/policy', { friend_policy: this.policy }).then(r => {
        if (r.code === 0) { this.okMsg = r.data } else this.msg = r.msg
      })
    },
    goSearch () { if (this.word) this.$router.push('/search?word=' + encodeURIComponent(this.word)) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    },
    fmtShort (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    },
    timeDiff (t) {
      if (!t) return ''
      const s = Math.floor((Date.now() - new Date(t).getTime()) / 1000)
      if (s < 60) return s + '秒前'
      if (s < 3600) return Math.floor(s / 60) + '分钟前'
      if (s < 86400) return Math.floor(s / 3600) + '小时前'
      return Math.floor(s / 86400) + '天前'
    }
  }
}
</script>