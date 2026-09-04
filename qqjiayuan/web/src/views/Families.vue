<template>
  <div>
    <!-- 面包屑 -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">广场</a>&gt;家族天地</div>

    <div class="list">
      <div class="row"><img src="/static/image/jzsy.gif" alt="家族">&nbsp;三潴家族，以家为名，结伴江湖。</div>
    </div>

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
      <p><button class="btn" @click="createFamily">提交申请（500金币）</button></p>
    </div>

    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>

    <!-- 特色家族 -->
    <div class="module-title">特色家族（{{ curCat || '全部' }}）<span class="name"><a href="javascript:;" @click="cat=''">全部分类</a></span></div>
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

    <!-- 家族服务 -->
    <div class="module-title">家族服务</div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/families')">家族排行</a>.<a href="javascript:;" @click="createOpen = true; cat=''">创建家族</a>.<a href="javascript:;" @click="$router.push('/channel/1')">逛论坛</a>.<a href="javascript:;" @click="$router.push('/')">回广场</a><br>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">广场</a>&gt;家族天地</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Families',
  data () {
    return {
      list: [], acts: [], mine: null,
      categories: ['同城同乡', '青春校园', '打工生涯', '明星粉丝', '情感男女', '娱乐八卦', '军人风采', '舞文弄墨', '游戏动漫', '科技数码', '时尚生活', '其他'],
      cat: '', wd: '', createOpen: false, form: { name: '', slogan: '', description: '', category: '' }, msg: '', okMsg: ''
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
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
      api.get('/families/activities').then(r => { if (r.code === 0) this.acts = r.data })
      if (this.isLogin) {
        api.get('/families/mine').then(r => { if (r.code === 0) this.mine = r.data })
      }
    },
    join (f) {
      api.post('/families/' + f.id + '/join').then(r => {
        if (r.code === 0) { this.okMsg = '已加入【' + f.name + '】，欢迎回家！'; this.load() }
        else this.msg = r.msg
      })
    },
    createFamily () {
      this.msg = ''; this.okMsg = ''
      if (!this.form.name) { this.msg = '请填写家族名称'; return }
      api.post('/families', this.form).then(r => {
        if (r.code === 0) { this.okMsg = '家族创建成功！'; this.createOpen = false; this.load() }
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
