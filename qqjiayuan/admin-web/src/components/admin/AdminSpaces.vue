<template>
  <div class="admin-spaces">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按昵称/号码搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
      </div>
      <el-alert v-if="listError" :title="listError" type="error" show-icon closable @close="listError = ''" style="margin-bottom:12px" />
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="username" label="号码" width="90" />
        <el-table-column prop="nickname" label="昵称" width="120" show-overflow-tooltip />
        <el-table-column prop="name" label="空间名称" width="160" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.name || '—' }}</template>
        </el-table-column>
        <el-table-column prop="signature" label="空间签名" min-width="180" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.signature || '—' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '正常' : '关闭' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="开通时间" width="160">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEditor(row)">编辑资料</el-button>
              <el-button size="mini" :type="row.status === 1 ? 'danger' : 'success'" plain @click="toggleStatus(row)">
                {{ row.status === 1 ? '关闭' : '开启' }}
              </el-button>
              <el-dropdown trigger="click" @command="handleCommand($event, row)" style="margin-left:4px">
                <el-button size="mini" icon="el-icon-more" plain>内容维护</el-button>
                <el-dropdown-menu slot="dropdown">
                  <el-dropdown-item command="moods">心情管理</el-dropdown-item>
                  <el-dropdown-item command="articles">日志管理</el-dropdown-item>
                  <el-dropdown-item command="albums">相册/照片</el-dropdown-item>
                  <el-dropdown-item command="messages">留言管理</el-dropdown-item>
                  <el-dropdown-item command="visitors">访客记录</el-dropdown-item>
                </el-dropdown-menu>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <el-dialog :title="'编辑空间资料：' + (editorRow.nickname || editorRow.username || '')" :visible.sync="editorDlg" width="620px" :close-on-click-modal="false">
      <el-alert v-if="editorError" :title="editorError" type="error" show-icon style="margin-bottom:14px" />
      <el-form label-width="90px" v-loading="editorLoading">
        <el-form-item label="空间名称"><el-input v-model.trim="editorForm.name" maxlength="50" show-word-limit /></el-form-item>
        <el-form-item label="空间签名"><el-input v-model.trim="editorForm.signature" maxlength="200" show-word-limit /></el-form-item>
        <el-form-item label="空间介绍"><el-input v-model.trim="editorForm.intro" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="editorDlg = false">取 消</el-button><el-button type="primary" :loading="saveLoading" @click="saveEditor">保存资料</el-button></div>
    </el-dialog>

    <el-dialog title="空间内容维护" :visible.sync="workspaceDlg" width="1080px" top="5vh" :close-on-click-modal="false">
      <div class="workspace-heading" v-if="workspaceRow"><b>{{ workspaceRow.nickname || workspaceRow.username }}</b><span> / {{ workspaceRow.name || '未命名空间' }}</span></div>
      <el-tabs v-model="activePanel" @tab-click="switchPanel">
        <el-tab-pane label="心情" name="moods">
          <el-alert v-if="panelError.moods" :title="panelError.moods" type="error" show-icon closable @close="$set(panelError, 'moods', '')" />
          <el-table :data="moodList" size="small" v-loading="panelLoading.moods" style="margin-top:12px">
            <el-table-column prop="content" label="内容" min-width="390" show-overflow-tooltip><template slot-scope="{row}"><el-button type="text" @click="viewMood(row)">{{ row.content }}</el-button></template></el-table-column>
            <el-table-column prop="created_at" label="时间" width="170"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
            <el-table-column label="状态" width="75"><template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '正常' : '已删' }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="80"><template slot-scope="{row}"><el-button size="mini" type="danger" plain :disabled="row.status !== 1" @click="deleteMood(row)">删除</el-button></template></el-table-column>
          </el-table>
          <el-pagination v-if="moodTotal" background layout="total, prev, pager, next" :total="moodTotal" :page-size="moodSize" :current-page="moodPage" @current-change="p => { moodPage = p; loadMoods() }" />
        </el-tab-pane>
        <el-tab-pane label="日志" name="articles">
          <el-alert v-if="panelError.articles" :title="panelError.articles" type="error" show-icon closable @close="$set(panelError, 'articles', '')" />
          <el-table :data="articleList" size="small" v-loading="panelLoading.articles" style="margin-top:12px">
            <el-table-column prop="title" label="标题" min-width="390" show-overflow-tooltip><template slot-scope="{row}"><el-button type="text" @click="viewArticle(row)">{{ row.title }}</el-button></template></el-table-column>
            <el-table-column prop="created_at" label="时间" width="170"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
            <el-table-column label="状态" width="75"><template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '正常' : '已删' }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="80"><template slot-scope="{row}"><el-button size="mini" type="danger" plain :disabled="row.status !== 1" @click="deleteArticle(row)">删除</el-button></template></el-table-column>
          </el-table>
          <el-pagination v-if="articleTotal" background layout="total, prev, pager, next" :total="articleTotal" :page-size="articleSize" :current-page="articlePage" @current-change="p => { articlePage = p; loadArticles() }" />
        </el-tab-pane>
        <el-tab-pane label="相册/照片" name="albums">
          <el-alert v-if="panelError.albums" :title="panelError.albums" type="error" show-icon closable @close="$set(panelError, 'albums', '')" />
          <el-table :data="albumList" size="small" v-loading="panelLoading.albums" style="margin-top:12px">
            <el-table-column prop="name" label="相册名称" min-width="250" />
            <el-table-column label="封面" width="90"><template slot-scope="{row}"><img v-if="row.cover" class="album-cover" :src="$pic(row.cover)" :alt="row.name"><span v-else>—</span></template></el-table-column>
            <el-table-column prop="count" label="照片数" width="90" />
            <el-table-column prop="created_at" label="创建时间" width="170"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
            <el-table-column label="操作" width="180"><template slot-scope="{row}"><el-button size="mini" type="primary" plain icon="el-icon-picture" @click="loadPhotos(row)">查看照片</el-button><el-button size="mini" type="danger" plain icon="el-icon-delete" @click="deleteAlbum(row)">删除</el-button></template></el-table-column>
          </el-table>
          <el-pagination v-if="albumTotal" background layout="total, prev, pager, next" :total="albumTotal" :page-size="albumSize" :current-page="albumPage" @current-change="p => { albumPage = p; loadAlbums() }" />
        </el-tab-pane>
        <el-tab-pane label="留言" name="messages">
          <el-alert v-if="panelError.messages" :title="panelError.messages" type="error" show-icon closable @close="$set(panelError, 'messages', '')" />
          <el-table :data="messageList" size="small" v-loading="panelLoading.messages" style="margin-top:12px">
            <el-table-column prop="from_nickname" label="留言人" width="150"><template slot-scope="{row}"><el-button type="text" @click="viewUser(row.from_user_id)">{{ row.from_nickname || ('用户' + row.from_user_id) }}</el-button></template></el-table-column>
            <el-table-column prop="content" label="内容" min-width="390" show-overflow-tooltip />
            <el-table-column prop="created_at" label="时间" width="170"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
            <el-table-column label="状态" width="75"><template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '正常' : '已删' }}</el-tag></template></el-table-column>
            <el-table-column label="操作" width="80"><template slot-scope="{row}"><el-button size="mini" type="danger" plain :disabled="row.status !== 1" @click="deleteMessage(row)">删除</el-button></template></el-table-column>
          </el-table>
          <el-pagination v-if="messageTotal" background layout="total, prev, pager, next" :total="messageTotal" :page-size="messageSize" :current-page="messagePage" @current-change="p => { messagePage = p; loadMessages() }" />
        </el-tab-pane>
        <el-tab-pane label="访客" name="visitors">
          <el-alert v-if="panelError.visitors" :title="panelError.visitors" type="error" show-icon closable @close="$set(panelError, 'visitors', '')" />
          <el-table :data="visitorList" size="small" v-loading="panelLoading.visitors" style="margin-top:12px">
            <el-table-column label="访客" min-width="220"><template slot-scope="{row}"><el-button type="text" @click="viewUser(row.user_id)">{{ row.nickname || ('用户' + row.user_id) }}</el-button></template></el-table-column>
            <el-table-column prop="created_at" label="访问时间" width="190"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
          </el-table>
          <el-pagination v-if="visitorTotal" background layout="total, prev, pager, next" :total="visitorTotal" :page-size="visitorSize" :current-page="visitorPage" @current-change="p => { visitorPage = p; loadVisitors() }" />
        </el-tab-pane>
      </el-tabs>
      <div slot="footer"><el-button @click="workspaceDlg = false">关 闭</el-button></div>
    </el-dialog>

    <el-dialog title="照片列表" :visible.sync="photoDlg" width="820px">
      <div class="photo-title">{{ selectedAlbum.name || '相册' }} <span class="help-line">共 {{ photoTotal }} 张</span></div>
      <el-alert v-if="photoError" :title="photoError" type="error" show-icon />
      <el-table :data="photoList" size="small" v-loading="photoLoading">
        <el-table-column label="照片" width="120"><template slot-scope="{row}"><img class="photo-preview" :src="$pic(row.file)" :alt="row.caption || '照片'" @click="previewPhoto(row)"></template></el-table-column>
        <el-table-column prop="caption" label="说明" min-width="300"><template slot-scope="{row}">{{ row.caption || '—' }}</template></el-table-column>
        <el-table-column prop="created_at" label="上传时间" width="180"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="80"><template slot-scope="{row}"><el-button size="mini" type="danger" plain @click="deletePhoto(row)">删除</el-button></template></el-table-column>
      </el-table>
      <el-pagination v-if="photoTotal" background layout="total, prev, pager, next" :total="photoTotal" :page-size="photoSize" :current-page="photoPage" @current-change="p => { photoPage = p; loadPhotos(selectedAlbum) }" />
    </el-dialog>

    <el-dialog title="内容查看" :visible.sync="contentDlg" width="700px"><h3>{{ contentTitle }}</h3><div class="content-view">{{ contentText }}</div><div slot="footer"><el-button @click="contentDlg = false">关 闭</el-button></div></el-dialog>
    <el-dialog title="用户资料" :visible.sync="userDlg" width="520px"><el-descriptions v-if="userDetail" :column="2" border size="small" v-loading="userLoading"><el-descriptions-item label="号码">{{ userDetail.username || userDetail.id }}</el-descriptions-item><el-descriptions-item label="昵称">{{ userDetail.nickname || '—' }}</el-descriptions-item><el-descriptions-item label="性别">{{ userDetail.gender === 2 ? '女' : '男' }}</el-descriptions-item><el-descriptions-item label="状态">{{ userDetail.status === 1 ? '正常' : '封禁' }}</el-descriptions-item></el-descriptions><el-alert v-if="userError" :title="userError" type="error" show-icon /><div slot="footer"><el-button @click="userDlg = false">关 闭</el-button></div></el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminSpaces',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, word: '', loading: false, listError: '',
      editorDlg: false, editorRow: {}, editorForm: { name: '', signature: '', intro: '' }, editorLoading: false, editorError: '', saveLoading: false,
      workspaceDlg: false, workspaceRow: null, activePanel: 'moods',
      panelLoading: { moods: false, articles: false, albums: false, messages: false, visitors: false },
      panelError: { moods: '', articles: '', albums: '', messages: '', visitors: '' },
      moodList: [], moodPage: 1, moodSize: 10, moodTotal: 0,
      articleList: [], articlePage: 1, articleSize: 10, articleTotal: 0,
      albumList: [], albumPage: 1, albumSize: 10, albumTotal: 0,
      messageList: [], messagePage: 1, messageSize: 20, messageTotal: 0,
      visitorList: [], visitorPage: 1, visitorSize: 10, visitorTotal: 0,
      selectedAlbum: {}, photoDlg: false, photoList: [], photoPage: 1, photoSize: 10, photoTotal: 0, photoLoading: false, photoError: '',
      contentDlg: false, contentTitle: '', contentText: '', userDlg: false, userDetail: null, userLoading: false, userError: ''
    }
  },
  mounted () { this.load() },
  methods: {
    normalize (r, fallbackSize) {
      const data = r && r.data
      if (Array.isArray(data)) return { list: data, total: data.length, page: 1, size: fallbackSize }
      if (data && Array.isArray(data.list)) return { list: data.list, total: Number(data.total) || data.list.length, page: Number(data.page) || 1, size: Number(data.size) || fallbackSize }
      return { list: [], total: 0, page: 1, size: fallbackSize }
    },
    isFallback (r) { return r && [404, 405, 501].indexOf(Number(r.code)) !== -1 },
    fallbackGet (urls, config, index) {
      index = index || 0
      return api.get(urls[index], config).then(r => (this.isFallback(r) && index < urls.length - 1) ? this.fallbackGet(urls, config, index + 1) : r)
    },
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true; this.listError = ''
      api.get('/admin/spaces', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        if (r.code === 0) { const out = this.normalize(r, this.size); this.list = out.list; this.total = out.total; this.page = out.page }
        else this.listError = r.msg || '空间列表加载失败'
      }).catch(() => { this.listError = '空间列表加载失败，请稍后重试' }).finally(() => { this.loading = false })
    },
    openEditor (row) {
      if (!row) return
      this.editorRow = row; this.editorError = ''; this.editorForm = { name: row.name || '', signature: row.signature || '', intro: row.intro || '' }; this.editorDlg = true
    },
    saveEditor () {
      if (!this.editorRow.id) return
      this.saveLoading = true; this.editorError = ''
      api.put('/admin/spaces/' + this.editorRow.id, this.editorForm).then(r => {
        if (r.code !== 0) { this.editorError = r.msg || '保存失败'; return }
        this.$message.success('空间资料已保存'); this.editorDlg = false; this.editorRow = Object.assign({}, this.editorRow, this.editorForm); this.load()
      }).catch(() => { this.editorError = '保存失败，请稍后重试' }).finally(() => { this.saveLoading = false })
    },
    toggleStatus (row) {
      const target = row.status === 1 ? 0 : 1
      this.$confirm(target === 0 ? `确定关闭「${row.nickname}」的空间吗？` : `确定开启「${row.nickname}」的空间吗？`, '提示', { type: 'warning' }).then(() => {
        row._statusLoading = true
        return api.put(`/admin/spaces/${row.id}/status`, { status: target }).then(r => { if (r.code === 0) { this.$message.success(target === 0 ? '已关闭' : '已开启'); this.load() } else this.$message.error(r.msg || '状态更新失败') }).catch(() => this.$message.error('状态更新失败，请稍后重试')).finally(() => { row._statusLoading = false })
      }).catch(() => {})
    },
    handleCommand (cmd, row) { this.workspaceRow = row; this.workspaceDlg = true; this.activePanel = cmd; this.resetPanel(cmd); this.loadPanel(cmd) },
    switchPanel (tab) { this.resetPanel(tab.name); this.loadPanel(tab.name) },
    resetPanel (name) { this.$set(this.panelError, name, '') },
    loadPanel (name) { if (name === 'moods') this.loadMoods(); else if (name === 'articles') this.loadArticles(); else if (name === 'albums') this.loadAlbums(); else if (name === 'messages') this.loadMessages(); else if (name === 'visitors') this.loadVisitors() },
    loadMoods () {
      const key = 'moods'; this.$set(this.panelLoading, key, true); this.$set(this.panelError, key, '')
      this.fallbackGet(['/admin/spaces/' + this.workspaceRow.user_id + '/moods', '/space/' + this.workspaceRow.user_id + '/moods'], { params: { user_id: this.workspaceRow.user_id, page: this.moodPage, size: this.moodSize } }).then(r => { if (r.code !== 0) { this.$set(this.panelError, key, r.msg || '心情加载失败'); return }; const out = this.normalize(r, this.moodSize); this.moodList = out.list; this.moodTotal = out.total; this.moodPage = out.page }).catch(() => this.$set(this.panelError, key, '心情加载失败，请稍后重试')).finally(() => this.$set(this.panelLoading, key, false))
    },
    loadArticles () {
      const key = 'articles'; this.$set(this.panelLoading, key, true); this.$set(this.panelError, key, '')
      this.fallbackGet(['/admin/spaces/' + this.workspaceRow.user_id + '/articles', '/space/' + this.workspaceRow.user_id + '/articles'], { params: { user_id: this.workspaceRow.user_id, page: this.articlePage, size: this.articleSize } }).then(r => { if (r.code !== 0) { this.$set(this.panelError, key, r.msg || '日志加载失败'); return }; const out = this.normalize(r, this.articleSize); this.articleList = out.list; this.articleTotal = out.total; this.articlePage = out.page }).catch(() => this.$set(this.panelError, key, '日志加载失败，请稍后重试')).finally(() => this.$set(this.panelLoading, key, false))
    },
    loadAlbums () {
      const key = 'albums'; this.$set(this.panelLoading, key, true); this.$set(this.panelError, key, '')
      this.fallbackGet(['/admin/spaces/' + this.workspaceRow.user_id + '/albums', '/space/' + this.workspaceRow.user_id + '/albums'], { params: { user_id: this.workspaceRow.user_id, page: this.albumPage, size: this.albumSize } }).then(r => { if (r.code !== 0) { this.$set(this.panelError, key, r.msg || '相册加载失败'); return }; const out = this.normalize(r, this.albumSize); this.albumList = out.list; this.albumTotal = out.total; this.albumPage = out.page }).catch(() => this.$set(this.panelError, key, '相册加载失败，请稍后重试')).finally(() => this.$set(this.panelLoading, key, false))
    },
    loadMessages () {
      const key = 'messages'; this.$set(this.panelLoading, key, true); this.$set(this.panelError, key, '')
      this.fallbackGet(['/admin/spaces/' + this.workspaceRow.user_id + '/messages', '/space/' + this.workspaceRow.user_id + '/messages'], { params: { user_id: this.workspaceRow.user_id, page: this.messagePage, size: this.messageSize } }).then(r => { if (r.code !== 0) { this.$set(this.panelError, key, r.msg || '留言加载失败'); return }; const out = this.normalize(r, this.messageSize); this.messageList = out.list; this.messageTotal = out.total; this.messagePage = out.page }).catch(() => this.$set(this.panelError, key, '留言加载失败，请稍后重试')).finally(() => this.$set(this.panelLoading, key, false))
    },
    loadVisitors () {
      const key = 'visitors'; this.$set(this.panelLoading, key, true); this.$set(this.panelError, key, '')
      this.fallbackGet(['/admin/spaces/' + this.workspaceRow.user_id + '/visitors', '/space/' + this.workspaceRow.user_id + '/visitors'], { params: { user_id: this.workspaceRow.user_id, page: this.visitorPage, size: this.visitorSize } }).then(r => { if (r.code !== 0) { this.$set(this.panelError, key, r.msg || '访客加载失败'); return }; const out = this.normalize(r, this.visitorSize); this.visitorList = out.list; this.visitorTotal = out.total; this.visitorPage = out.page }).catch(() => this.$set(this.panelError, key, '访客加载失败，请稍后重试')).finally(() => this.$set(this.panelLoading, key, false))
    },
    deleteMood (row) { this.confirmDelete('确定删除该心情？', '/admin/moods/' + row.id, row, '心情') },
    deleteArticle (row) { this.confirmDelete('确定删除该日志？', '/admin/articles/' + row.id, row, '日志') },
    deleteMessage (row) { this.confirmDelete('确定删除该留言？', '/admin/messages/' + row.id, row, '留言', ['/admin/space-messages/' + row.id]) },
    confirmDelete (text, url, row, label, alternatives) {
      this.$confirm(text, '提示', { type: 'warning' }).then(() => {
        const urls = [url].concat(alternatives || []); return this.deleteFallback(urls, 0).then(r => { if (r.code === 0) { this.$message.success(label + '已删除'); row.status = 0 } else this.$message.error(r.msg || label + '删除失败') }).catch(() => this.$message.error(label + '删除失败，请稍后重试'))
      }).catch(() => {})
    },
    deleteFallback (urls, i) { return api.delete(urls[i]).then(r => (this.isFallback(r) && i < urls.length - 1) ? this.deleteFallback(urls, i + 1) : r) },
    viewMood (row) { this.contentTitle = '心情'; this.contentText = row.content || ''; this.contentDlg = true },
    viewArticle (row) { this.contentTitle = row.title || '日志'; this.contentText = row.content || '暂无正文'; this.contentDlg = true },
    loadPhotos (album) {
      this.selectedAlbum = album; this.photoPage = 1; this.photoDlg = true; this.fetchPhotos()
    },
    fetchPhotos () {
      this.photoLoading = true; this.photoError = ''
      this.fallbackGet(['/admin/albums/' + this.selectedAlbum.id + '/photos', '/admin/spaces/' + this.workspaceRow.user_id + '/albums/' + this.selectedAlbum.id + '/photos'], { params: { page: this.photoPage, size: this.photoSize, album_id: this.selectedAlbum.id } }).then(r => { if (r.code !== 0) { this.photoError = r.msg || '照片加载失败'; return }; const out = this.normalize(r, this.photoSize); this.photoList = out.list; this.photoTotal = out.total }).catch(() => { this.photoError = '照片加载失败，请稍后重试' }).finally(() => { this.photoLoading = false })
    },
    deleteAlbum (row) {
      this.$confirm('删除相册不会自动恢复，确定继续吗？', '提示', { type: 'warning' }).then(() => this.deleteFallback(['/admin/albums/' + row.id, '/admin/spaces/albums/' + row.id], 0).then(r => { if (r.code === 0) { this.$message.success('相册已删除'); this.loadAlbums() } else this.$message.error(r.msg || '相册删除失败') }).catch(() => this.$message.error('相册删除失败，请稍后重试'))).catch(() => {})
    },
    deletePhoto (row) {
      this.$confirm('确定删除这张照片？', '提示', { type: 'warning' }).then(() => this.deleteFallback(['/admin/photos/' + row.id, '/admin/albums/photos/' + row.id], 0).then(r => { if (r.code === 0) { this.$message.success('照片已删除'); this.fetchPhotos() } else this.$message.error(r.msg || '照片删除失败') }).catch(() => this.$message.error('照片删除失败，请稍后重试'))).catch(() => {})
    },
    previewPhoto (row) { this.$alert('<img src="' + this.$pic(row.file) + '" style="max-width:100%;max-height:60vh">', row.caption || '照片预览', { dangerouslyUseHTMLString: true, customClass: 'photo-dialog' }).catch(() => {}) },
    viewUser (id) {
      this.userDlg = true; this.userDetail = null; this.userError = ''; this.userLoading = true
      api.get('/users/' + id).then(r => { if (r.code === 0) this.userDetail = r.data; else this.userError = r.msg || '用户资料加载失败' }).catch(() => { this.userError = '用户资料加载失败，请稍后重试' }).finally(() => { this.userLoading = false })
    },
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' }
  }
}
</script>

<style scoped>
.admin-spaces .workspace-heading { padding: 0 0 12px; color: #606266; }
.admin-spaces .workspace-heading span { color: #909399; }
.admin-spaces .album-cover { width: 48px; height: 36px; object-fit: cover; border-radius: 3px; vertical-align: middle; }
.admin-spaces .photo-preview { width: 82px; height: 58px; object-fit: cover; cursor: pointer; border-radius: 3px; }
.admin-spaces .photo-title { margin-bottom: 12px; font-size: 14px; color: #303133; }
.admin-spaces .content-view { white-space: pre-wrap; word-break: break-word; line-height: 1.8; min-height: 100px; color: #606266; }
.admin-spaces .help-line { margin-left: 8px; color: #909399; font-size: 12px; }
</style>
