<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;{{ fam.name || '…' }}<br></div>

    <div v-if="!loading && !fam.id" class="module-content"><span class="empty">家族不存在或已解散</span></div>

    <template v-if="fam.id">
      <!-- 主页导航（参考站：主页|论坛|聊室|娱乐|家人） -->
      <div class="module-title">
        主页 | <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/forum')">论坛</a>
        | <a href="javascript:;" @click="$router.push({ path: '/chat', query: { family_id: fam.id } })">聊室</a>
        | <a href="javascript:;" @click="$router.push('/games')">游戏</a>
        | <a href="javascript:;" @click="$router.push('/family/'+fam.id+'#members')">家人</a>
        | <a href="javascript:;" @click="doFavor"><font :color="fam.favored ? '#1a9e1a' : '#004299'">{{ fam.favored ? '★已收藏' : '☆收藏该家' }}</font></a><br>
      </div>

      <!-- 家族风采（参考站：家族名+在线人数+等级行+我的积分/职称） -->
      <div class="module-content">
        <b style="font-size:16px;color:#004299">{{ fam.name }}家族</b><br>
        <template v-if="fam.category">[{{ fam.category }}] </template>
        <span class="txt-fade">（<a href="javascript:;" @click="showOnline">{{ fam.online || 0 }}</a>人在线）</span><br>
        <a href="javascript:;" @click="showTree">等级</a>:{{ fam.tree_level }}({{ fam.tree_exp }}/{{ fam.tree_level * 100 }})<br>
        <template v-if="fam.my_role">
          我的家族积分:{{ fam.my_exp || 0 }}<br>
          我的职称:<b style="color:#004299">{{ fam.my_title || '初级家人' }}</b> <a href="javascript:;" @click="$router.push('/family/levels')">>></a><br>
        </template>
        <span class="txt-fade">乐斗积分 {{ fam.battle_score }}　族斗荣誉点 {{ fam.war_points || 0 }}　族长：<a href="javascript:;" @click="$router.push('/user/'+fam.owner_id)"><font :color="fam.owner && fam.owner.color || '#004299'">{{ fam.owner ? fam.owner.nickname : '?' }}</font></a></span>
        <p style="color:#888">{{ fam.slogan || '（暂无口号）' }}</p>
        <p>{{ fam.description }}</p>
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
        <table style="width:100%;border-collapse:collapse"><tbody><tr>
          <td style="width:74px"><img src="/static/picture/tree.gif" width="65" height="50" alt="守护树"></td>
          <td valign="top">
            <a href="javascript:;" @click="doTree">抚摸</a>/<a href="javascript:;" @click="doTree">拥抱</a>守护树<br>
            参与<a href="javascript:;" @click="$router.push('/family/'+fam.id+'/war')">族斗</a>为族争光<br>
            <a href="javascript:;" @click="doSign">每日分财富</a><span class="txt-fade">（签到领奖）</span>
          </td>
        </tr></tbody></table>
        <template v-if="fam.signed_today"><span style="color:#1a9e1a">今日任务已全部完成 ✓</span></template>
      </div>
      <div class="module-content" v-else><span class="empty">加入家族后可参与今日任务</span></div>

      <!-- 家族动态 -->
      <div class="module-title">家族动态</div>
      <div class="module-content" v-if="acts.length">
        <div v-for="(a,i) in acts.slice(0,10)" :key="'a'+a.id" class="row00">
          {{ i+1 }}.({{ ago(a.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+a.user_id)"><font :color="a.user && a.user.color || '#004299'">{{ a.user ? a.user.nickname : '神秘友友' }}</font></a>{{ a.content }}
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">本家族还没有动态</span></div>

      <!-- 功能导航（参考站：乐斗.活动.邀好友.签到 / 族斗.心情.反馈.收藏夹） -->
      <div class="module-title">功能导航</div>
      <div class="module-content plist">
        <div class="row00" v-if="fam.my_role">
          <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/battle')">乐斗</a>.<a href="javascript:;" @click="$router.push('/activities')">活动</a>.<a href="javascript:;" @click="$router.push('/friends')">邀好友</a>.<a href="javascript:;" @click="doSign">签到</a><br>
          <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/war')">族斗</a>.<a href="javascript:;" @click="$router.push('/mood')">心情</a>.<a href="javascript:;" @click="$router.push('/notices')">公告</a>.<a href="javascript:;" @click="doFavor">{{ fam.favored ? '取消收藏' : '收藏夹' }}</a><br>
          <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/forum')">家族论坛</a>.<a href="javascript:;" @click="doTree">守护树</a>.<a href="javascript:;" @click="doLeave" style="color:#c00">退出家族</a><br>
          <template v-if="fam.signed_today"><span style="color:#1a9e1a">今天已在家族签到 ✓</span><br></template>
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
        <a href="javascript:;" @click="$router.push('/families')">去家族首页</a><br>
      </div>
      <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
      <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>

      <!-- 家人（成员） -->
      <div class="module-title" id="members">家人：在线{{ fam.online || 0 }}人/总{{ fam.member_count || 0 }}人<template v-if="isOwner">　<font color="#1a9e1a">（族长可管理成员）</font></template></div>
      <ul class="dtuser">
        <li v-for="m in fam.members" :key="m.id">
          <template v-if="m.role === 'owner'">👑</template><template v-else-if="m.role === 'admin'">⭐</template><template v-else>◆</template>
          <a href="javascript:;" @click="$router.push('/user/'+m.user_id)"><font :color="m.user && m.user.color || '#004299'">{{ m.user ? m.user.nickname : '友友' }}</font></a>
          <span class="txt-fade">（{{ roleName(m.role) }}，贡献 {{ m.exp }}）</span>
          <template v-if="isOwner && m.role !== 'owner'"><a href="javascript:;" style="color:#c00" @click="removeMember(m.user_id)">[移除]</a></template>
        </li>
      </ul>

      <!-- 家族公告 -->
      <div class="module-title">家族公告</div>
      <div class="module-content">{{ fam.announcement || '（还没有公告）' }}</div>
      <div class="module-content" v-if="isOwner">
        <textarea v-model.trim="ann" maxlength="500" rows="3"></textarea>
        <p><button class="btn" @click="saveAnn">发布公告</button></p>
      </div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Family',
  data () {
    return { fam: {}, acts: [], hot: [], allFamilies: [], visitId: 0, loading: true, msg: '', okMsg: '', ann: '' }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    isOwner () { return this.isLogin && this.fam.my_role === 'owner' }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    roleName (r) { return r === 'owner' ? '族长' : (r === 'admin' ? '长老' : '成员') },
    load () {
      this.loading = true
      const id = this.$route.params.id
      api.get('/families/' + id).then(r => {
        this.loading = false
        if (r.code === 0) { this.fam = r.data; this.ann = r.data.announcement || '' }
        else this.fam = {}
      })
      api.get('/families/' + id + '/activities').then(r => { if (r.code === 0) this.acts = r.data })
      api.get('/families/' + id + '/hot').then(r => { if (r.code === 0) this.hot = r.data })
      api.get('/families').then(r => { if (r.code === 0) this.allFamilies = r.data })
    },
    showOnline () { this.msg = ''; this.okMsg = '在线家人可在家人列表查看（10 分钟内活跃）' },
    showTree () {
      const el = document.getElementById('tree-sec')
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
    removeMember (userId) {
      if (!window.confirm('确定将该成员移出家族吗？')) return
      api.post('/families/' + this.fam.id + '/members/' + userId + '/remove').then(r => {
        if (r.code === 0) { this.okMsg = r.data || '已移除'; this.load() }
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
    doSign () {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fam.id + '/signin').then(r => {
        if (r.code === 0) { this.okMsg = '家族签到成功 +20经验 +5G币'; this.load() }
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
    saveAnn () {
      this.msg = ''; this.okMsg = ''
      api.put('/families/' + this.fam.id + '/ann', { announcement: this.ann }).then(r => {
        if (r.code === 0) { this.okMsg = '公告已更新'; this.load() }
        else this.msg = r.msg
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
