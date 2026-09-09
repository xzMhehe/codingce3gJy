<template>
  <div>
    <!-- ================= 空间未开通 ================= -->
    <template v-if="state === 'notopened'">
      <template v-if="isOwner || !uInfo.id">
        {{ uInfo.nickname || '您' }}，您的空间尚未开通！<br>
        在这里，你可以关注好友的动态，也可以每天用说说分享心情，用日志记录感悟，用照片记录生活，你还等什么，加入我们，体验完美手机生活吧！<br>
        <a href="javascript:;" @click="showOpen = !showOpen">马上开通</a><br>
        <template v-if="showOpen">
          <!-- 开通空间表单 -->
          <b>【开通空间】</b><br>
          <form @submit.prevent="openSpace">
            空间名称:<input type="text" v-model.trim="openForm.name" maxlength="20" /><br>
            空间签名:<input type="text" v-model.trim="openForm.signature" maxlength="120" /><br>
            空间说明:<input type="text" v-model.trim="openForm.intro" maxlength="300" /><br>
            <input type="submit" value="确定开通" />
          </form>
          <p v-if="tip" style="color:#e05a00;padding:2px 5px">{{ tip }}</p>
        </template>
      </template>
      <template v-else>
        该用户的空间尚未开通！<br>
      </template>
    </template>

    <!-- ================= 开通成功 ================= -->
    <template v-else-if="state === 'opened'">
      <b>【开通空间】</b><br>
      恭喜！开通成功！<br>
      <a href="javascript:;" @click="load()">进入空间</a><br>
    </template>

    <!-- ================= 空间已开通 ================= -->
    <template v-else-if="state === 'active'">
      <!-- 子导航：个人中心 | 主页 | 好友 | 个人档（对齐诺哈 blog：个人中心|主页|好友|个人档） -->
      <a href="javascript:;" v-if="subTab !== 'center'" @click="subTab = 'center'">个人中心</a><span v-else>个人中心</span>|
      <a href="javascript:;" v-if="subTab !== 'home'" @click="subTab = 'home'">主页</a><span v-else>主页</span>|
      <a href="javascript:;" v-if="subTab !== 'friends'" @click="subTab = 'friends'">好友</a><span v-else>好友</span>|
      <a href="javascript:;" v-if="subTab !== 'profile'" @click="subTab = 'profile'">个人档</a><span v-else>个人档</span>
      <br>

      <!-- ========== 个人中心 ========== -->
      <template v-if="subTab === 'center'">
        <form v-if="isOwner" @submit.prevent="addMood">
          你在想些什么呢？<br>
          <textarea v-model.trim="moodContent" rows="3"></textarea><br>
          <input type="submit" value="发表" />
        </form>
        <template v-if="isOwner">
          <a href="javascript:;" @click="showLogForm = !showLogForm">写日志</a>.<a href="javascript:;" @click="showPhotoForm = !showPhotoForm">传相片</a>.<a href="javascript:;" @click="showFileForm = !showFileForm">传文件</a><br>
          <template v-if="showLogForm">
            <form @submit.prevent="addArticle">
              日志标题:<input type="text" v-model.trim="articleForm.title" maxlength="100" /><br>
              日志内容:<br>
              <textarea v-model.trim="articleForm.content" rows="5"></textarea><br>
              <input type="submit" value="发表日志" /> <a href="javascript:;" @click="showLogForm = false">取消</a>
            </form>
          </template>
          <template v-if="showPhotoForm">
            <form @submit.prevent="addPhoto">
              选择相册:
              <select v-model.number="photoForm.albumId">
                <option v-for="al in albums" :key="al.id" :value="al.id">{{ al.name }}({{ al.count || 0 }})</option>
                <option :value="0">+ 新建相册</option>
              </select>
              <input v-if="photoForm.albumId === 0" type="text" v-model.trim="photoForm.newAlbum" maxlength="50" placeholder="新相册名" size="10" />
              <br>
              相片说明:<input type="text" v-model.trim="photoForm.caption" maxlength="100" size="15" /><br>
              <input type="file" accept="image/*" @change="onPhotoFile" /> <span v-if="photoForm.name" class="txt-fade">{{ photoForm.name }}</span><br>
              <input type="submit" value="上传" /> <a href="javascript:;" @click="showPhotoForm = false">取消</a>
            </form>
          </template>
          <template v-if="showFileForm">
            <form @submit.prevent="addFile">
              文件名称:<input type="text" v-model.trim="fileForm.name" maxlength="100" placeholder="如 音乐.mp3" size="15" /><br>
              <input type="file" @change="onFile" /> <span v-if="fileForm.fileName" class="txt-fade">{{ fileForm.fileName }}</span><br>
              <input type="submit" value="上传" /> <a href="javascript:;" @click="showFileForm = false">取消</a>
            </form>
          </template>
        </template>

        【<a href="javascript:;" @click="subTab = 'home'">空间动态</a>】<br>
        <template v-if="moods.length">
          <div v-for="(m, i) in moods" :key="m.id">
            {{ i + 1 }}.{{ m.content }}({{ fmtTime(m.created_at) }})<br>
            <a href="javascript:;" @click="toggleMoodComments(m)">评论({{ m.comment_count || 0 }})</a>.<a href="javascript:;" @click="forwardMood(m.id)">转发({{ m.forward_count || 0 }})</a><template v-if="isOwner">.<a href="javascript:;" @click="delMood(m.id)">删除</a></template><br>
            <template v-if="m._showComments">
              <div v-for="c in m.comments" :key="c.id">
                {{ c.nickname || '' }}:{{ c.content }}
              </div>
              <form @submit.prevent="addMoodComment(m)">
                <input type="text" v-model.trim="m._commentText" maxlength="300" size="20" />
                <input type="submit" value="评论" />
              </form>
            </template>
          </div>
          <span v-if="moodPageTotal > 1">(第<b>{{ moodPage }}</b>/{{ moodPageTotal }}页/共{{ moodTotal }}条记录) <a href="javascript:;" v-if="moodPage > 1" @click="loadMoods(moodPage - 1)">&lt;&lt;上一页</a> <a href="javascript:;" v-if="moodPage < moodPageTotal" @click="loadMoods(moodPage + 1)">下一页&gt;&gt;</a></span>
        </template>
        <span v-else>暂无动态</span>

        【<a href="javascript:;" @click="$router.push('/friends')">好友动态</a>】<br>

        【<a href="javascript:;" @click="$router.push('/home')">功能导航</a>】<br>
        <a href="javascript:;" @click="$router.push('/friends')">好友</a>.<a href="javascript:;" @click="$router.push('/messages')">信箱</a>.<a href="javascript:;" @click="$router.push('/wallet')">账户</a>.<a href="javascript:;" @click="$router.push('/noble')">贵族</a><br>
        <a href="javascript:;" @click="$router.push('/')">社区</a>.<a href="javascript:;" @click="$router.push('/families')">家族</a>.<a href="javascript:;" @click="$router.push('/my-threads')">帖子</a>.<a href="javascript:;" @click="$router.push('/my-threads')">我的帖子</a><br>
      </template>

      <!-- ========== 主页 ========== -->
      <template v-if="subTab === 'home'">
        【<a href="javascript:;" @click="subTab = 'center'">心情</a>】<br>
        <template v-if="moods.length">
          <div v-for="(m, i) in moods" :key="'hm'+m.id">
            {{ i + 1 }}.{{ m.content }}({{ fmtTime(m.created_at) }})<br>
            <a href="javascript:;" @click="toggleMoodComments(m)">评论({{ m.comment_count || 0 }})</a>.<a href="javascript:;" @click="forwardMood(m.id)">转发({{ m.forward_count || 0 }})</a><template v-if="isOwner">.<a href="javascript:;" @click="delMood(m.id)">删除</a></template><br>
            <template v-if="m._showComments">
              <div v-for="c in m.comments" :key="c.id">
                {{ c.nickname || '' }}:{{ c.content }}
              </div>
              <form @submit.prevent="addMoodComment(m)">
                <input type="text" v-model.trim="m._commentText" maxlength="300" size="20" />
                <input type="submit" value="评论" />
              </form>
            </template>
          </div>
          <span v-if="moodPageTotal > 1">(第<b>{{ moodPage }}</b>/{{ moodPageTotal }}页/共{{ moodTotal }}条记录) <a href="javascript:;" v-if="moodPage > 1" @click="loadMoods(moodPage - 1)">&lt;&lt;上一页</a> <a href="javascript:;" v-if="moodPage < moodPageTotal" @click="loadMoods(moodPage + 1)">下一页&gt;&gt;</a></span>
        </template>
        <span v-else>暂无心情</span><br>

        【<a href="javascript:;" @click="$router.push('/space/article/'+a.id)">日志</a>】<br>
        <template v-if="articles.length">
          <div v-for="a in articles" :key="'ha'+a.id">
            <a href="javascript:;" @click="$router.push('/space/article/'+a.id)">{{ a.title }}</a>({{ fmtTime(a.created_at) }})<template v-if="isOwner">.<a href="javascript:;" @click="delArticle(a.id)">删除</a></template>
          </div>
        </template>
        <span v-else>暂无日志</span><br>

        【<a href="javascript:;" @click="curAlbum = null">相册</a>】<br>
        <template v-if="albums.length">
          <span v-for="al in albums" :key="al.id">
            <a href="javascript:;" @click="showAlbum(al)">
              <template v-if="al.cover && al.cover.indexOf('data:') === 0"><img :src="al.cover" alt="." style="width:46px;height:46px;object-fit:cover;vertical-align:middle"></template>
              <template v-else-if="al.cover"><img :src="'/static/picture/' + al.cover" alt="." style="width:46px;height:46px;object-fit:cover;vertical-align:middle" onerror="this.style.display='none'"></template>
              {{ al.name }}({{ al.count || 0 }})
            </a> </span>
          <div v-if="curAlbum">
            <b>【{{ curAlbum.name }}】</b> <a href="javascript:;" @click="curAlbum = null">收起</a><br>
            <template v-if="curAlbum.photos.length">
              <div v-for="p in curAlbum.photos" :key="p.id">
                <img :src="p.photo_base64 || ''" alt="." style="max-width:180px;display:block" />
                <span v-if="p.caption" class="txt-fade">{{ p.caption }}</span>
                <template v-if="isOwner">.<a href="javascript:;" @click="delPhoto(p.id)">删除</a></template><br>
              </div>
            </template>
            <span v-else>这个相册还没有照片</span>
          </div>
        </template>
        <span v-else>暂无相册</span><br>

        【<a href="javascript:;">留言</a>】<br>
        <template v-if="spaceMsgs.length">
          <div v-for="msg in spaceMsgs" :key="'msg'+msg.id">
            <a href="javascript:;" @click="$router.push('/user/'+msg.from_user_id)">{{ msg.from_nickname }}</a>:{{ msg.content }}({{ fmtTime(msg.created_at) }})<template v-if="isOwner || msg.from_user_id === user.id">.<a href="javascript:;" @click="delSpaceMsg(msg.id)">删除</a></template>
          </div>
        </template>
        <span v-else>暂无留言</span>
        <form v-if="isLogin && !isOwner" @submit.prevent="addSpaceMsg">
          <input type="text" v-model.trim="spaceMsgContent" maxlength="300" size="20" placeholder="给TA留言" />
          <label><input type="checkbox" v-model="spaceMsgPrivate" />悄悄话</label>
          <input type="submit" value="留言" />
        </form><br>

        【<a href="javascript:;" @click="loadFiles">文件</a>】<br>
        <template v-if="files.length">
          <div v-for="f in files" :key="'f'+f.id">
            <a href="javascript:;" @click="downloadFile(f)">{{ f.name }}</a>
            <span class="txt-fade">（{{ f.size }}B / {{ f.clicks }}次下载）</span>
            <template v-if="isOwner">.<a href="javascript:;" style="color:#c00" @click="delFile(f.id)">删除</a></template><br>
          </div>
        </template>
        <span v-else>暂无文件</span><br>

        【<a href="javascript:;">访客</a>】<br>
        <template v-if="visitors.length">
          <div v-for="(v, i) in visitors" :key="'v'+v.id">
            {{ i + 1 }}.<a href="javascript:;" @click="$router.push('/user/'+v.user_id)">{{ v.nickname }}</a><br>
          </div>
          (第<b>1</b>/1页/共{{ visitors.length }}条记录)
        </template>
        <span v-else>暂无访客</span>
      </template>

      <!-- ========== 好友（对齐诺哈 friend_list.asp） ========== -->
      <template v-if="subTab === 'friends'">
        【<a href="javascript:;" @click="loadFriends">好友</a>】<br>
        <template v-if="friends.length">
          <div v-for="f in friends" :key="'f'+f.id">
            <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a>
            <span class="txt-fade">Lv.{{ f.level }}</span>
            <a href="javascript:;" @click="$router.push('/messages/'+f.id)">[家信]</a><br>
          </div>
        </template>
        <span v-else>暂无好友</span><br>
      </template>

      <!-- ========== 个人档 ========== -->
      <template v-if="subTab === 'profile'">
        【个人资料】<a href="javascript:;" v-if="isOwner" @click="$router.push('/profile')">修改</a><br>
        会员号码:{{ uInfo.username }}<br>
        会员昵称:{{ uInfo.nickname }}<br>
        性别年龄:{{ uInfo.gender === 2 ? '女' : '男' }}<br>
        【空间资料】<a href="javascript:;" v-if="isOwner" @click="showEditSpace = !showEditSpace">修改</a><br>
        空间名称:{{ space.name || uInfo.nickname + '的空间' }}<br>
        空间签名:{{ space.signature || '暂无' }}<br>
        空间介绍:{{ space.intro || '暂无' }}<br>
        <template v-if="showEditSpace && isOwner">
          <form @submit.prevent="updateSpace">
            空间名称:<input type="text" v-model.trim="editForm.name" maxlength="50" /><br>
            空间签名:<input type="text" v-model.trim="editForm.signature" maxlength="200" /><br>
            空间介绍:<input type="text" v-model.trim="editForm.intro" maxlength="500" /><br>
            <input type="submit" value="保存" /> <a href="javascript:;" @click="showEditSpace = false">取消</a>
          </form>
        </template>
      </template>

      ----------<br>
      <!-- 底部链接 -->
      <template v-if="subTab === 'home'">
        <a href="javascript:;" @click="subTab = 'center'">个人中心</a>.<a href="javascript:;" @click="$router.push('/profile')">管理</a>.<a href="javascript:;" @click="$router.push('/')">论坛</a><br>
      </template>
      <template v-else-if="subTab === 'profile'">
        <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>.<a href="javascript:;" @click="$router.push('/')">论坛</a><br>
      </template>
      <template v-else>
        <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>.<a href="javascript:;" @click="$router.push('/profile')">管理</a>.<a href="javascript:;" @click="$router.push('/')">论坛</a><br>
      </template>
    </template>

    <!-- 底部提示 -->
    <p v-if="tip" style="color:#1a9e1a;padding:3px 5px">{{ tip }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Space',
  data () {
    return {
      loading: true,
      state: 'notopened', // notopened | opened | active
      space: null,
      uInfo: {},
      subTab: 'home',
      moods: [],
      articles: [],
      albums: [],
      friends: [],
      files: [],
      spaceMsgs: [],
      visitors: [],
      moodPage: 1,
      moodPageTotal: 1,
      moodTotal: 0,
      moodContent: '',
      showOpen: false,
      showEditSpace: false,
      showLogForm: false,
      showPhotoForm: false,
      showFileForm: false,
      photoForm: { albumId: 0, newAlbum: '', caption: '', file: null, name: '' },
      fileForm: { name: '', file: null, fileName: '' },
      curAlbum: null,
      spaceMsgPrivate: false,
      openForm: { name: '', signature: '', intro: '' },
      editForm: { name: '', signature: '', intro: '' },
      articleForm: { title: '', content: '' },
      spaceMsgContent: '',
      tip: ''
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    userId () { return Number(this.$route.params.userId) || this.user.id },
    isOwner () { return this.isLogin && this.user.id === this.userId }
  },
  watch: {
    '$route': 'load',
    subTab (v) { this.loadTab(v) }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      // 记录访问
      if (this.isLogin && this.userId !== this.user.id) {
        api.post('/space/visit/' + this.userId).catch(() => {})
      }
      api.get('/space/' + this.userId).then(r => {
        this.loading = false
        if (r.code === 0 && r.data && r.data.space) {
          this.space = r.data.space
          this.state = this.space.status === 1 ? 'active' : 'notopened'
          this.editForm = { name: this.space.name || '', signature: this.space.signature || '', intro: this.space.intro || '' }
          this.subTab = this.isOwner ? 'center' : 'home'
          this.loadTab(this.subTab)
        } else {
          this.state = 'notopened'
        }
      }).catch(() => { this.loading = false; this.state = 'notopened' })
      this.loadUserInfo()
    },
    loadUserInfo () {
      api.get('/users/' + this.userId).then(r => {
        if (r.code === 0) this.uInfo = r.data
      })
    },
    loadTab (v) {
      if (v === 'home' || v === 'center') this.loadMoods(1)
      this.loadArticles()
      this.loadAlbums()
      this.loadSpaceMsgs()
      this.loadVisitors()
      this.loadFiles()
      if (v === 'friends') this.loadFriends()
    },
    loadFiles () {
      api.get('/space/' + this.userId + '/files', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.files = r.data || []
      })
    },
    onFile (e) {
      const f = e.target.files && e.target.files[0]
      if (!f) return
      if (f.size > 5 * 1024 * 1024) { this.tip = '文件太大，请压缩到 5MB 以内'; return }
      const reader = new FileReader()
      reader.onload = () => { this.fileForm.file = reader.result; this.fileForm.fileName = f.name }
      reader.readAsDataURL(f)
    },
    addFile () {
      if (!this.fileForm.file) { this.tip = '请选择要上传的文件'; return }
      const name = this.fileForm.name || this.fileForm.fileName || 'file'
      api.post('/space/file', { name, file_base64: this.fileForm.file }).then(r => {
        if (r.code === 0) {
          this.tip = '上传成功'
          this.fileForm = { name: '', file: null, fileName: '' }
          this.showFileForm = false
          this.loadFiles()
        } else { this.tip = r.msg || '上传失败' }
      }).catch(() => { this.tip = '上传失败，请稍后再试' })
    },
    downloadFile (f) {
      api.get('/space/files/' + f.id + '/download').then(r => {
        if (r.code !== 0) { this.tip = r.msg || '下载失败'; return }
        const a = document.createElement('a')
        a.href = r.data.base64
        a.download = r.data.name || 'file'
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        this.loadFiles()
      })
    },
    delFile (id) {
      api.delete('/space/file/' + id).then(() => this.loadFiles())
    },
    loadFriends () {
      api.get('/space/' + this.userId + '/friends', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.friends = r.data || []
      })
    },
    loadMoods (page) {
      api.get('/space/' + this.userId + '/moods', { params: { user_id: this.userId, page: page } }).then(r => {
        if (r.code === 0) {
          this.moods = (r.data.list || []).map(m => ({ ...m, _showComments: false, _commentText: '' }))
          this.moodPage = r.data.page
          this.moodTotal = r.data.total
          this.moodPageTotal = Math.ceil(r.data.total / r.data.size) || 1
        }
      })
    },
    loadArticles () {
      api.get('/space/' + this.userId + '/articles', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.articles = r.data.list || []
      })
    },
    loadAlbums () {
      api.get('/space/' + this.userId + '/albums', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.albums = r.data || []
      })
    },
    loadSpaceMsgs () {
      api.get('/space/' + this.userId + '/messages', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.spaceMsgs = r.data.list || []
      })
    },
    loadVisitors () {
      api.get('/space/' + this.userId + '/visitors', { params: { user_id: this.userId } }).then(r => {
        if (r.code === 0) this.visitors = r.data || []
      })
    },
    openSpace () {
      if (!this.isLogin) {
        this.$router.push('/login?redirect=' + encodeURIComponent(this.$route.fullPath))
        return
      }
      if (!this.openForm.name) {
        this.tip = '请填写空间名称'
        return
      }
      api.post('/space', this.openForm).then(r => {
        if (r.code === 0) {
          this.tip = ''
          this.state = 'opened'
        } else {
          this.tip = r.msg || '开通失败，请稍后再试'
        }
      }).catch(() => {
        this.tip = '网络异常，请稍后再试'
      })
    },
    updateSpace () {
      api.put('/space', this.editForm).then(r => {
        if (r.code === 0) {
          this.space.name = this.editForm.name
          this.space.signature = this.editForm.signature
          this.space.intro = this.editForm.intro
          this.showEditSpace = false
          this.tip = '修改成功'
        }
      })
    },
    addMood () {
      if (!this.moodContent) return
      api.post('/space/mood', { content: this.moodContent }).then(r => {
        if (r.code === 0) {
          this.moodContent = ''
          this.loadMoods(1)
        }
      })
    },
    delMood (id) {
      api.delete('/space/mood/' + id).then(() => this.loadMoods(this.moodPage))
    },
    toggleMoodComments (m) {
      this.$set(m, '_showComments', !m._showComments)
    },
    addMoodComment (m) {
      if (!m._commentText) return
      api.post('/space/mood/' + m.id + '/comment', { content: m._commentText }).then(r => {
        if (r.code === 0) {
          m._commentText = ''
          this.loadMoods(this.moodPage)
        }
      })
    },
    forwardMood (id) {
      api.post('/space/mood/' + id + '/forward').then(r => {
        if (r.code === 0) this.loadMoods(1)
      })
    },
    addArticle () {
      if (!this.articleForm.title || !this.articleForm.content) return
      api.post('/space/article', this.articleForm).then(r => {
        if (r.code === 0) {
          this.articleForm = { title: '', content: '' }
          this.showLogForm = false
          this.loadArticles()
        }
      })
    },
    delArticle (id) {
      api.delete('/space/article/' + id).then(() => this.loadArticles())
    },
    addSpaceMsg () {
      if (!this.spaceMsgContent) return
      api.post('/space/message/' + this.userId, { content: this.spaceMsgContent, mtype: this.spaceMsgPrivate ? 1 : 0 }).then(r => {
        if (r.code === 0) {
          this.spaceMsgContent = ''
          this.spaceMsgPrivate = false
          this.loadSpaceMsgs()
        }
      })
    },
    onPhotoFile (e) {
      const f = e.target.files && e.target.files[0]
      if (!f) return
      if (f.size > 3 * 1024 * 1024) { this.tip = '图片太大，请压缩到 3MB 以内'; return }
      const reader = new FileReader()
      reader.onload = () => { this.photoForm.file = reader.result; this.photoForm.name = f.name }
      reader.readAsDataURL(f)
    },
    addPhoto () {
      if (!this.photoForm.file) { this.tip = '请选择要上传的相片'; return }
      const doUpload = (albumId) => {
        api.post('/space/albums/' + albumId + '/photos', {
          caption: this.photoForm.caption,
          photo_base64: this.photoForm.file,
          format: (this.photoForm.name.split('.').pop() || 'jpg').toLowerCase()
        }).then(r => {
          if (r.code === 0) {
            this.tip = '上传成功'
            this.photoForm = { albumId: 0, newAlbum: '', caption: '', file: null, name: '' }
            this.showPhotoForm = false
            this.loadAlbums()
          } else { this.tip = r.msg || '上传失败' }
        }).catch(() => { this.tip = '上传失败，请稍后再试' })
      }
      if (this.photoForm.albumId > 0) { doUpload(this.photoForm.albumId); return }
      // 新建相册
      if (!this.photoForm.newAlbum) { this.tip = '请选择相册或填写新相册名'; return }
      api.post('/space/album', { name: this.photoForm.newAlbum }).then(r => {
        if (r.code === 0) doUpload(r.data.id)
      })
    },
    showAlbum (al) {
      api.get('/space/albums/' + al.id + '/photos').then(r => {
        if (r.code === 0) this.curAlbum = { id: al.id, name: al.name, photos: r.data.list || [] }
      })
    },
    delPhoto (id) {
      api.delete('/space/photos/' + id).then(() => {
        if (this.curAlbum) this.showAlbum({ id: this.curAlbum.id, name: this.curAlbum.name })
        this.loadAlbums()
      })
    },
    delSpaceMsg (id) {
      api.delete('/space/message/' + id).then(() => this.loadSpaceMsgs())
    },
    fmtTime (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>
