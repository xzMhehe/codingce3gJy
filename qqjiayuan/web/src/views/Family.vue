<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;{{ fam.name || '…' }}<br></div>

    <div v-if="!loading && !fam.id" class="module-content"><span class="empty">家族不存在或已解散</span></div>

    <template v-if="fam.id">
      <!-- 主页导航（参考站：主页|论坛|聊室|娱乐|家人） -->
      <div class="module-title">
        主页 | <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/forum')">论坛</a>
        | <a href="javascript:;" @click="$router.push({ path: '/chat', query: { family_id: fam.id } })">聊室</a>
        | <a href="javascript:;" @click="$router.push('/games')">娱乐</a>
        | <a href="javascript:;" @click="goMembers">家人</a><br>
      </div>

      <!-- 家族风采（参考站：名称+在线人数+家族等级+我的积分/职称） -->
      <div class="module-content">
        <b style="font-size:16px;color:#004299">{{ fam.name }}家族</b>
        <span class="txt-fade">（<a href="javascript:;" @click="showOnline">{{ fam.online || 0 }}</a>人在线）</span><br>
        <a href="javascript:;" @click="showTree">等级</a>:{{ fam.tree_level }}({{ fam.tree_exp }}/{{ fam.tree_level * 100 }})<br>
        <template v-if="fam.my_role">
          我的家族积分:{{ fam.my_exp || 0 }}<br>
          我的职称:<img :src="'/static/picture/family_lv/' + titleIcon(fam.my_title) + '.gif'" width="16" height="16" :alt="fam.my_title">{{ fam.my_title || '初级家人' }} <a href="javascript:;" @click="$router.push('/family/levels')">>></a><br>
        </template>
      </div>

      <!-- 家族热点 -->
      <div class="module-title">家族热点</div>
      <div class="module-content" v-if="hot.length">
        <div v-for="t in hot" :key="'h'+t.id">
          [{{ t.tag }}]<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a><br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">家族论坛还没有帖子，去抢个头楼吧</span></div>

      <!-- 今日任务（参考站：抚摸/拥抱守护树、参与族斗为族争光、每日分财富） -->
      <div class="module-title" id="tree-sec">今日任务</div>
      <div class="module-content" v-if="fam.my_role">
        <img src="/static/picture/tree.gif" width="130" height="100" alt="守护树"><br>
        <a href="javascript:;" @click="doTree">抚摸</a>/<a href="javascript:;" @click="doTree">拥抱</a>守护树<br>
        参与<a href="javascript:;" @click="$router.push('/family/'+fam.id+'/war')">族斗</a>为族争光<br>
        <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/welfare')">每日分财富</a>最高100000GB<br>
      </div>
      <div class="module-content" v-else><span class="empty">加入家族后可参与今日任务</span></div>

      <!-- 家族动态 -->
      <div class="module-title">家族动态</div>
      <div class="module-content" v-if="acts.length">
        <div v-for="(a,i) in acts.slice(0,5)" :key="'a'+a.id">
          {{ i+1 }}.({{ ago(a.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+a.user_id)"><font :color="a.user && a.user.color || '#004299'">{{ a.user ? a.user.nickname : '神秘友友' }}</font></a>{{ a.content }}<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">本家族还没有动态</span></div>

      <!-- 功能导航（参考站：乐斗.活动.邀好友.签到 / 族斗.心情.反馈.收藏夹） -->
      <div class="module-title">功能导航</div>
      <div class="module-content plist">
        <div class="row00" v-if="fam.my_role">
          <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/battle')">乐斗</a>.<a href="javascript:;" @click="$router.push('/activities')">活动</a>.<a href="javascript:;" @click="$router.push('/friends')">邀好友</a>.<a href="javascript:;" @click="$router.push('/family/'+fam.id+'/welfare')">签到</a><br>
          <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/war')">族斗</a>.<a href="javascript:;" @click="$router.push('/mood')">心情</a>.<a href="javascript:;" @click="$router.push('/')">反馈</a>.<a href="javascript:;" @click="doFavor">{{ fam.favored ? '取消收藏' : '收藏夹' }}</a><br>
          <a href="javascript:;" @click="doLeave" style="color:#c00">退出家族</a><br>
        </div>
        <div class="row00" v-else>
          你还不是本家族成员，<a href="javascript:;" @click="doJoin">快速加入</a>.<a href="javascript:;" @click="doFavor">{{ fam.favored ? '取消收藏' : '收藏该家' }}</a><br>
        </div>
        <!-- 串门：跳到其他家族 -->
        <form style="margin-top:4px" @submit.prevent="goVisit">
          <select v-model.number="visitId">
            <option :value="0">选择家族去串门</option>
            <option v-for="f in allFamilies" :key="'v'+f.id" :value="f.id" v-if="f.id !== fam.id">{{ f.name }}</option>
          </select>
          <input type="submit" value="[我去串门]">
        </form>
      </div>

      <!-- 家人/访客（参考站：家人：在线N人/总N人 + 访客：今天N人） -->
      <div class="module-content" id="members">
        家人：在线<a href="javascript:;" @click="showOnline">{{ fam.online || 0 }}</a>人/总{{ fam.member_count || 0 }}人<br>
        访客：今天{{ fam.visits_today || 0 }}人<br>
      </div>

      <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
      <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Family',
  data () {
    return { fam: {}, acts: [], hot: [], allFamilies: [], visitId: 0, loading: true, msg: '', okMsg: '' }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      const id = this.$route.params.id
      api.get('/families/' + id).then(r => {
        this.loading = false
        if (r.code === 0) { this.fam = r.data }
        else this.fam = {}
      })
      api.get('/families/' + id + '/activities').then(r => { if (r.code === 0) this.acts = r.data })
      api.get('/families/' + id + '/hot').then(r => { if (r.code === 0) this.hot = r.data })
      api.get('/families').then(r => { if (r.code === 0) this.allFamilies = r.data })
    },
    // 职称对应等级图标（家人等级列表 01-31）
    titleIcon (t) {
      const map = { '见习家人': 1, '初级家人': 2, '活跃家人': 3, '中级家人': 5, '有爱家人': 7, '高级家人': 9, '荣誉家人': 11, '勤劳家人': 13, '骨干家人': 13, '精英家人': 15, '智慧家人': 17, '炫耀家人': 19, '金鼎家人': 21, '元老家人': 21, '美丽家人': 24, '忠义家人': 27, '枭雄家人': 31, '族长': 31 }
      return map[t] || 2
    },
    showOnline () { this.msg = ''; this.okMsg = '在线家人可在家人列表查看（10 分钟内活跃）' },
    showTree () {
      const el = document.getElementById('tree-sec')
      if (el) el.scrollIntoView()
    },
    goMembers () {
      const el = document.getElementById('members')
      if (el) el.scrollIntoView()
    },
    goVisit () {
      if (this.visitId > 0) this.$router.push('/family/' + this.visitId)
    },
    doFavor () {
      this.msg = ''; this.okMsg = ''
      if (!this.isLogin) { this.msg = '请先登录'; return }
      api.post('/families/' + this.fam.id + '/favorite').then(r => {
        if (r.code === 0) { this.fam.favored = r.data.favored; this.okMsg = r.data.msg; this.load() }
        else this.msg = r.msg
      })
    },
    doLeave () {
      this.msg = ''; this.okMsg = ''
      if (!window.confirm('确定退出「' + this.fam.name + '」家族吗？')) return
      api.post('/families/' + this.fam.id + '/leave').then(r => {
        if (r.code === 0) { this.okMsg = r.data || '已退出家族'; this.load() }
        else this.msg = r.msg
      })
    },
    doJoin () {
      this.msg = ''; this.okMsg = ''
      if (!this.isLogin) { this.msg = '请先登录'; return }
      api.post('/families/' + this.fam.id + '/join').then(r => {
        if (r.code === 0) { this.okMsg = '加入成功，欢迎回家！'; this.load() }
        else this.msg = r.msg
      })
    },
    doTree () {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fam.id + '/tree').then(r => {
        if (r.code === 0) {
          this.okMsg = (r.data.leveled ? '🎉 守护树升级到 Lv.' + r.data.tree_level + '！' : '守护树成长 +30') + '，当前 Lv.' + r.data.tree_level
          this.load()
        } else this.msg = r.msg
      })
    },
    ago (t) {
      if (!t) return ''
      const diff = Math.max(0, (Date.now() - new Date(t).getTime()) / 1000)
      if (diff < 60) return Math.floor(diff) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
      return Math.floor(diff / 86400) + '天前'
    }
  }
}
</script>
