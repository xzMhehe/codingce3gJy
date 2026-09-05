<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;{{ fam.name || '…' }}<br></div>

    <div v-if="!loading && !fam.id" class="module-content"><span class="empty">家族不存在或已解散</span></div>

    <template v-if="fam.id">
      <!-- 主页导航（参考站：主页|论坛|聊室|娱乐|家人） -->
      <div class="module-title">
        主页 | <a href="javascript:;" @click="$router.push('/family/'+fam.id+'/forum')">论坛</a>
        | <a href="javascript:;" @click="$router.push({ path: '/chat', query: { family_id: fam.id } })">聊室</a>
        | <a href="javascript:;" @click="$router.push('/channel/1')">娱乐</a>
        | <a href="javascript:;" @click="$router.push('/family/'+fam.id+'#members')">家人</a><br>
      </div>

      <!-- 家族风采 -->
      <div class="module-content">
        <b style="font-size:16px;color:#004299">{{ fam.name }}家族</b><br>
        <template v-if="fam.category">[{{ fam.category }}] </template>
        <span class="txt-fade">（<a href="javascript:;" @click="showOnline">{{ fam.online || 0 }}</a>人在线）</span><br>
        <span class="txt-fade">等级:{{ fam.tree_level }}({{ fam.tree_exp }}/{{ fam.tree_level * 100 }})　乐斗积分 {{ fam.battle_score }}　族长：<a href="javascript:;" @click="$router.push('/user/'+fam.owner_id)"><font :color="fam.owner && fam.owner.color || '#004299'">{{ fam.owner ? fam.owner.nickname : '?' }}</font></a></span>
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

      <!-- 家族动态 -->
      <div class="module-title">家族动态</div>
      <div class="module-content" v-if="acts.length">
        <div v-for="(a,i) in acts" :key="'a'+a.id" class="row00">
          {{ i+1 }}.({{ ago(a.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+a.user_id)"><font :color="a.user && a.user.color || '#004299'">{{ a.user ? a.user.nickname : '神秘友友' }}</font></a>{{ a.content }}
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">本家族还没有动态</span></div>

      <!-- 功能导航（参考站：乐斗.活动.邀好友.签到 / 族斗.心情.反馈.收藏夹） -->
      <div class="module-title">功能导航</div>
      <div class="module-content plist">
        <div class="row00" v-if="fam.my_role">
          <a href="javascript:;" @click="doBattle">乐斗</a>.<a href="javascript:;" @click="$router.push('/channel/1')">活动</a>.<a href="javascript:;" @click="$router.push('/friends')">邀好友</a>.<a href="javascript:;" @click="doSign">签到</a><br>
          <a href="javascript:;" @click="doTree">族斗</a>.<a href="javascript:;" @click="$router.push('/mood')">心情</a>.<a href="javascript:;" @click="$router.push('/channel/4')">反馈</a>.<a href="javascript:;" @click="doFavor">收藏夹</a><br>
          <template v-if="fam.signed_today"><span style="color:#1a9e1a">今天已在家族签到 ✓</span><br></template>
        </div>
        <div class="row00" v-else>
          你还不是本家族成员，<a href="javascript:;" @click="doJoin">快速加入</a><br>
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
      <div class="module-title" id="members">家人：在线{{ fam.online || 0 }}人/总{{ fam.member_count || 0 }}人</div>
      <ul class="dtuser">
        <li v-for="m in fam.members" :key="m.id">
          <template v-if="m.role === 'owner'">👑</template><template v-else-if="m.role === 'admin'">⭐</template><template v-else>◆</template>
          <a href="javascript:;" @click="$router.push('/user/'+m.user_id)"><font :color="m.user && m.user.color || '#004299'">{{ m.user ? m.user.nickname : '友友' }}</font></a>
          <span class="txt-fade">（{{ roleName(m.role) }}，贡献 {{ m.exp }}）</span>
        </li>
      </ul>

      <!-- 守护树 -->
      <div class="module-title">守护树</div>
      <div class="module-content">
        <table style="width:100%;border-collapse:collapse"><tbody><tr>
          <td style="width:56px"><img src="/static/picture/tree.gif" width="48" height="48" alt="守护树"></td>
          <td valign="top">
            家族守护树 <b style="color:#004299">Lv.{{ fam.tree_level }}</b>（成长值 {{ fam.tree_exp }} / 下一级 {{ (fam.tree_level)*100 }}）　今日签到 {{ fam.tree_today || 0 }} 人<br>
            <span class="txt-fade">成员每日抚摸可 +30 成长值，满 100 升 1 级，等级越高家族越兴旺。</span>
          </td>
        </tr></tbody></table>
      </div>

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
    goVisit () {
      if (this.visitId > 0) this.$router.push('/family/' + this.visitId)
    },
    doFavor () { this.msg = '收藏夹功能开发中，敬请期待'; this.okMsg = '' },
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
        if (r.code === 0) { this.okMsg = '家族签到成功 +20经验 +5金币'; this.load() }
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
    doBattle () {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fam.id + '/battle').then(r => {
        if (r.code === 0) {
          this.okMsg = (r.data.win ? '🏆 乐斗获胜！' : '乐斗惜败，参与有奖~') + ' 对手【' + r.data.opponent + '】，积分 ' + (r.data.gain > 0 ? '+' : '') + r.data.gain + ' → ' + r.data.score
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
