<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="军团名搜索" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="620">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column label="军团名" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column prop="leader_name" label="军团长" width="110" show-overflow-tooltip />
        <el-table-column prop="leader_user_id" label="团长ID" width="80" align="center" />
        <el-table-column prop="member_count" label="成员数" width="80" align="center" />
        <el-table-column prop="notice" label="军团公告" min-width="200" show-overflow-tooltip />
        <el-table-column label="创建时间" width="170" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="210" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-user" title="成员" @click="openMembers(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-chat-dot-round" title="军团聊天" @click="openChats(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="解散" @click="dissolve(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 成员 -->
    <el-dialog :title="'军团成员 · ' + (cur.name || '')" :visible.sync="memberDlg" width="720px" :close-on-click-modal="false">
      <el-table :data="members" v-loading="loadingMember" stripe border max-height="420">
        <el-table-column prop="id" label="记录ID" width="80" align="center" />
        <el-table-column prop="player_name" label="玩家" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.player_name || '—' }}</span></template>
        </el-table-column>
        <el-table-column prop="home_num" label="家园号" width="90" align="center" />
        <el-table-column prop="user_id" label="用户ID" width="80" align="center" />
        <el-table-column label="身份" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.is_leader === 1 ? 'danger' : 'info'">{{ row.role_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="头衔" width="100" align="center" />
        <el-table-column prop="prestige" label="声望" width="90" align="center" />
        <el-table-column prop="rank_name" label="军衔" width="110" align="center" />
        <el-table-column label="操作" width="170" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain :disabled="row.is_leader === 1"
                       @click="transfer(row)">设为团长</el-button>
            <el-button size="mini" type="danger" plain :disabled="row.is_leader === 1"
                       @click="kick(row)">移出</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div slot="footer">
        <el-button @click="memberDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 军团聊天 -->
    <el-dialog :title="'军团聊天 · ' + (cur.name || '')" :visible.sync="chatDlg" width="720px" :close-on-click-modal="false">
      <el-table :data="chats" v-loading="loadingChat" stripe border max-height="420">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
        <el-table-column prop="user_name" label="玩家" width="130" show-overflow-tooltip />
        <el-table-column prop="content" label="内容" min-width="300" show-overflow-tooltip />
        <el-table-column label="时间" width="170" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delChat(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ chatTotal }}</b> 条 · 每页 {{ chatSize }} 条</div>
        <el-pagination v-show="chatTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="chatTotal" :page-size="chatSize"
                       :current-page="chatPage" :page-sizes="[5, 10, 20, 50, 100]"
                       @current-change="p => { chatPage = p; loadChats() }"
                       @size-change="s => { chatSize = s; chatPage = 1; loadChats() }" />
      </div>
      <div slot="footer">
        <el-button @click="chatDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 编辑军团 -->
    <el-dialog title="编辑军团" :visible.sync="editDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="军团名">
          <el-input v-model="form.name" maxlength="20" style="width:240px" />
        </el-form-item>
        <el-form-item label="军团公告">
          <el-input v-model="form.notice" type="textarea" :rows="3" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="转让团长">
          <el-select v-model="form.leader_user_id" filterable clearable placeholder="不转让则留空" style="width:280px">
            <el-option v-for="m in members" :key="m.user_id"
                       :label="(m.player_name || m.user_id) + '（' + m.role_name + '）'" :value="m.user_id" />
          </el-select>
          <div class="td-sub" style="margin-top:4px">成员列表需先打开「成员」弹窗加载</div>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyCorps',
  data () {
    return {
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      cur: {},
      memberDlg: false, members: [], loadingMember: false,
      chatDlg: false, chats: [], chatTotal: 0, chatPage: 1, chatSize: 5, loadingChat: false,
      editDlg: false, saving: false, form: {}
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-corps', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openMembers (row) {
      this.cur = row
      this.members = []
      this.memberDlg = true
      api.get('/admin/ezfy-corps/' + row.id + '/members').then(r => {
        if (r.code === 0) this.members = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openChats (row) {
      this.cur = row
      this.chatPage = 1
      this.chatDlg = true
      this.loadChats()
    },
    loadChats () {
      this.loadingChat = true
      api.get('/admin/ezfy-corps/' + this.cur.id + '/chats', {
        params: { page: this.chatPage, size: this.chatSize }
      }).then(r => {
        this.loadingChat = false
        if (r.code === 0) {
          this.chats = r.data.list
          this.chatTotal = r.data.total
          this.chatPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    delChat (row) {
      api.delete('/admin/ezfy-corps-chats/' + row.id).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadChats() } else this.$message.error(r.msg)
      })
    },
    transfer (row) {
      this.$confirm('确认把「' + (row.player_name || row.user_id) + '」设为军团长？原团长将变为普通成员。', '提示', { type: 'warning' }).then(() => {
        api.put('/admin/ezfy-corps/' + this.cur.id, { leader_user_id: row.user_id }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已转让'); this.openMembers(this.cur); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    kick (row) {
      this.$confirm('把「' + (row.player_name || row.user_id) + '」移出军团？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-corps/' + this.cur.id + '/members/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已移出'); this.openMembers(this.cur); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openEdit (row) {
      this.cur = row
      this.form = { name: row.name, notice: row.notice, leader_user_id: null }
      // 预加载成员，便于选择转让团长
      if (!this.members.length || this.cur.id !== row.id) {
        api.get('/admin/ezfy-corps/' + row.id + '/members').then(r => {
          if (r.code === 0) this.members = r.data.list
        })
      }
      this.editDlg = true
    },
    save () {
      const body = { name: this.form.name, notice: this.form.notice }
      if (this.form.leader_user_id) body.leader_user_id = this.form.leader_user_id
      this.saving = true
      api.put('/admin/ezfy-corps/' + this.cur.id, body).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    dissolve (row) {
      this.$confirm('解散将清除所有成员与军团聊天记录，确认解散「' + row.name + '」？', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-corps/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已解散'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
