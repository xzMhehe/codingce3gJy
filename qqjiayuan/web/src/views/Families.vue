<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;家族<br></div>
    <img src="/static/image/jiazu.gif" alt="家族" style="max-width:100%;vertical-align:middle">

    <!-- 我的家族 -->
    <div class="module-title">我的家族</div>
    <div class="module-content" v-if="mine">
      欢迎回家 → <a href="javascript:;" @click="$router.push('/family/'+mine.id)"><b style="color:#004299">{{ mine.name }}</b></a>
      <span class="txt-fade">（{{ mine.role==='owner' ? '族长' : '成员' }}，{{ mine.members || 0 }}人）</span>
      <br><a href="javascript:;" @click="$router.push('/family/'+mine.id)">进入家族&gt;&gt;</a>
    </div>
    <div class="module-content" v-else>
      加入家族一起在  家园社区闯荡吧，我们为你推荐：<br>
      <template v-if="list.length">
        最佳匹配：<a href="javascript:;" @click="$router.push('/family/'+list[0].id)">{{ list[0].name }}</a>
        (<a href="javascript:;" @click="join(list[0])">快速加入</a>)<br>
      </template>
      <a href="javascript:;" @click="createOpen = !createOpen">{{ createOpen ? '收起' : '申请我的家族' }}</a><br>
    </div>

    <!-- 创建家族 -->
    <div class="module-content" v-if="createOpen && isLogin && !mine" style="border-top:1px solid #9FC6EC;border-bottom:1px solid #9FC6EC;background:#FAFDFF">
      <div class="module-title">创建家族</div>
      <p>家族名称：<input type="text" v-model.trim="form.name" maxlength="14" style="width:62%"></p>
      <p>家族口号：<input type="text" v-model.trim="form.slogan" maxlength="30" style="width:62%"></p>
      <p>家族类别：<select v-model="form.category" style="width:66%">
        <option value="">选择类别</option>
        <option v-for="ct in categories" :key="ct" :value="ct">{{ ct }}</option>
      </select></p>
      <p>家族介绍：<textarea v-model.trim="form.description" maxlength="60" rows="2"></textarea></p>
      <p><button class="btn" @click="createFamily">提交申请（通过后扣 500 金币）</button></p>
    </div>

    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>

    <!-- 特色家族 -->
    <div class="module-title">特色家族 <span class="name"><a v-if="isAdmin" href="javascript:;" @click="goAdmin">审核</a></span></div>
    <div class="module-content" v-for="f in featured" :key="'fd'+f.id">
      ★.<a href="javascript:;" @click="$router.push('/family/'+f.id)"><b>{{ f.name }}</b></a>({{ f.members || 0 }}人)<br>
      简介：{{ f.description || '（暂无简介）' }}<br>
      <span class="txt-fade">族长：<a href="javascript:;" @click="$router.push('/user/'+f.owner_id)"><font :color="f.owner && f.owner.color || '#004299'">{{ f.owner ? f.owner.nickname : '?' }}</font></a></span>
    </div>
    <div class="module-content" v-if="!featured.length"><span class="empty">暂无特色家族</span></div>

    <!-- 家族大看台.家族活动 -->
    <div class="module-title">家族大看台.家族活动</div>
    <div class="module-content" v-if="activityThreads.length">
      <div class="row00" v-for="t in activityThreads" :key="'t'+t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅)<br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无家族活动</span></div>

    <!-- 家族列表（分类筛选） -->
    <div class="module-title">家族列表（{{ curCat || '全部' }}）<span class="name"><a href="javascript:;" @click="cat=''">全部分类</a></span></div>
    <div class="module-content" v-for="f in filteredList" :key="'f'+f.id">
      ★.<a href="javascript:;" @click="$router.push('/family/'+f.id)"><b>{{ f.name }}</b></a>({{ f.members || 0 }}人)<template v-if="f.role"><span style="color:#1a9e1a">[已加入]</span></template><br>
      简介：{{ f.description || '（暂无简介）' }}<br>
      <span class="txt-fade">族长：<a href="javascript:;" @click="$router.push('/user/'+f.owner_id)"><font :color="f.owner && f.owner.color || '#004299'">{{ f.owner ? f.owner.nickname : '?' }}</font></a>　守护树 Lv.{{ f.tree_level }}　积分 {{ f.battle_score }}</span>
    </div>
    <div class="module-content" v-if="!filteredList.length"><span class="empty">该分类下暂无家族</span></div>

    <!-- 家族类别 -->
    <div class="module-title">家族类别</div>
    <div class="module-content">
      <template v-for="(ct,i) in categories"><a :key="'c'+ct" href="javascript:;" @click="cat = ct">{{ ct }}</a><template v-if="i < categories.length-1">.</template></template>
    </div>

    <!-- 友联家族 -->
    <div class="module-title">友联家族</div>
    <div class="module-content"><span class="empty">暂无友联家族，快去结识兄弟家族吧</span></div>

    <!-- 家族区动态 -->
    <div class="module-title">家族区动态</div>
    <div class="module-content" v-if="acts.length">
      <div v-for="a in acts" :key="'a'+a.id" class="row00">
        <a href="javascript:;" @click="$router.push('/user/'+a.user_id)"><font :color="a.user && a.user.color || '#004299'">{{ a.user ? a.user.nickname : '神秘友友' }}</font></a>
        {{ a.content }} <span class="dtime">({{ ago(a.created_at) }})</span>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">还没有家族动态</span></div>

    <!-- 家族搜索 -->
    <div class="module-title">家族搜索</div>
    <div class="module-content">
      <form @submit.prevent="">
        请输入家族名或ID：
        <input type="text" v-model.trim="wd" maxlength="30" size="12">
        <input type="submit" value="搜索">
      </form>
    </div>

    <!-- 待审家族 -->
    <div class="module-title">待审家族 <span class="name"><a href="javascript:;" @click="pendingOpen = !pendingOpen">{{ pendingOpen ? '收起' : '查看' }}</a></span></div>
    <div class="module-content" v-if="pendingOpen">
      <div v-for="p in pendingList" :key="'p'+p.id" class="row00">
        <a href="javascript:;" @click="joinPending(p)"><b>{{ p.name }}</b></a>（{{ p.category || '未分类' }}）<span class="txt-fade">族长：{{ p.owner || '?' }}　{{ fmtDate(p.created_at) }}</span><br>
      </div>
      <div v-if="!pendingList.length"><span class="empty">暂无待审家族</span></div>
    </div>

    <!-- 家族服务 -->
    <div class="module-title">家族服务</div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/families')">家族排行</a>.<a href="javascript:;" @click="createOpen = true; cat=''">创建家族</a>.<a href="javascript:;" @click="pendingOpen = !pendingOpen; loadPending()">待审家族</a>.<a href="javascript:;" @click="$router.push('/channel/1')">逛论坛</a>.<a href="javascript:;" @click="$router.push('/')">回广场</a><br>
    </div>

  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Families',
  data () {
    return {
      list: [], acts: [], mine: null, featured: [], activityThreads: [], pendingList: [],
      categories: ['同城同乡', '青春校园', '打工生涯', '明星粉丝', '情感男女', '娱乐八卦', '军人风采', '舞文弄墨', '游戏动漫', '科技数码', '时尚生活', '其他'],
      cat: '', wd: '', createOpen: false, pendingOpen: false,
      form: { name: '', slogan: '', description: '', category: '' }, msg: '', okMsg: ''
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    isAdmin () { return this.$store.getters.isAdmin },
    curCat () { return this.cat },
    filteredList () {
      let arr = this.list
      if (this.cat) arr = arr.filter(f => f.category === this.cat)
      if (this.wd) arr = arr.filter(f => (f.name || '').indexOf(this.wd) >= 0 || (f.slogan || '').indexOf(this.wd) >= 0)
      return arr
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/families' + (this.cat ? '?category=' + encodeURIComponent(this.cat) : '')).then(r => { if (r.code === 0) this.list = r.data })
      api.get('/families/featured').then(r => { if (r.code === 0) this.featured = r.data })
      api.get('/families/activity-threads').then(r => { if (r.code === 0) this.activityThreads = r.data })
      api.get('/families/activities').then(r => { if (r.code === 0) this.acts = r.data })
      this.loadPending()
      if (this.isLogin) {
        api.get('/families/mine').then(r => { if (r.code === 0) this.mine = r.data })
      }
    },
    loadPending () {
      api.get('/families/pending').then(r => { if (r.code === 0) this.pendingList = r.data })
    },
    join (f) {
      api.post('/families/' + f.id + '/join').then(r => {
        if (r.code === 0) { this.okMsg = '已加入【' + f.name + '】，欢迎回家！'; this.load() }
        else this.msg = r.msg
      })
    },
    joinPending (p) {
      this.pendingOpen = true
      this.loadPending()
    },
    createFamily () {
      this.msg = ''; this.okMsg = ''
      if (!this.form.name) { this.msg = '请填写家族名称'; return }
      api.post('/families', this.form).then(r => {
        if (r.code === 0) { this.okMsg = '申请已提交，等待管理员审核（通过后扣 500 金币）！'; this.createOpen = false; this.load() }
        else this.msg = r.msg
      })
    },
    goAdmin () { window.open('/admin-ui/', '_blank') },
    fmtDate (t) {
      if (!t) return ''
      const d = new Date(t)
      return (d.getMonth() + 1) + '月' + d.getDate() + '日'
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
