<template>
  <div class="farm-admin">
    <el-tabs v-model="tab" type="card" @tab-click="onTab">
      <!-- ============ 用户数据 ============ -->
      <el-tab-pane label="玩家数据" name="users">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="userWord" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                      @keyup.enter.native="userPage = 1; loadUsers()" />
            <el-button type="primary" icon="el-icon-search" @click="userPage = 1; loadUsers()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="users" v-loading="userLoading" stripe>
            <el-table-column prop="user_id" label="家园号" width="90" />
            <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
            <el-table-column label="等级" width="80" align="center">
              <template slot-scope="{row}"><span class="lv">{{ row.level }}</span>级</template>
            </el-table-column>
            <el-table-column prop="point" label="经验" width="80" align="center" />
            <el-table-column prop="love" label="爱心" width="80" align="center" />
            <el-table-column prop="contri" label="贡献" width="80" align="center" />
            <el-table-column prop="cars" label="车辆" width="70" align="center" />
            <el-table-column prop="garage" label="车库" width="70" align="center" />
            <el-table-column prop="parked" label="被占车位" width="85" align="center" />
            <el-table-column label="操作" width="110" header-align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="userTotal"
                         :page-size="userSize" :current-page="userPage" :page-sizes="[10, 20, 50]"
                         @current-change="p => { userPage = p; loadUsers() }"
                         @size-change="s => { userSize = s; userPage = 1; loadUsers() }" />
        </el-card>
      </el-tab-pane>

      <!-- ============ 日志流水 ============ -->
      <el-tab-pane label="日志流水" name="logs">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-radio-group v-model="logType" size="small" @change="logPage = 1; loadLogs()">
              <el-radio-button label="join">加入记录</el-radio-button>
              <el-radio-button label="msg">游戏消息</el-radio-button>
            </el-radio-group>
            <el-input v-model="logWord" placeholder="家园号/昵称/关键词" clearable style="width:200px"
                      @keyup.enter.native="logPage = 1; loadLogs()" />
            <el-button type="primary" icon="el-icon-search" @click="logPage = 1; loadLogs()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="logs" v-loading="logLoading" stripe>
            <!-- 加入记录 -->
            <template v-if="logType === 'join'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="user_id" label="家园号" width="110" />
              <el-table-column prop="name" label="昵称(加入时)" min-width="140" show-overflow-tooltip />
              <el-table-column prop="created_at" label="加入时间" width="170" />
            </template>
            <!-- 游戏消息 -->
            <template v-else>
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="from" label="发送人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="to_uid" label="接收人(家园号)" width="130" />
              <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
              <el-table-column label="状态" width="80" align="center">
                <template slot-scope="{row}">
                  <el-tag :type="row.status === 0 ? 'warning' : 'info'" size="mini">{{ row.status === 0 ? '未读' : '已读' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="时间" width="170" />
            </template>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="logTotal"
                         :page-size="logSize" :current-page="logPage" :page-sizes="[15, 30, 50]"
                         @current-change="p => { logPage = p; loadLogs() }"
                         @size-change="s => { logSize = s; logPage = 1; loadLogs() }" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 编辑玩家数据 -->
    <el-dialog title="编辑玩家数据" :visible.sync="editDlg" width="420px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="昵称">
          <el-input :value="editRow.nickname" disabled />
        </el-form-item>
        <el-form-item label="爱心">
          <el-input-number v-model="editForm.love" :min="0" />
        </el-form-item>
        <el-form-item label="经验">
          <el-input-number v-model="editForm.point" :min="0" />
          <span class="help-line" style="margin-left:10px">等级 = 经验 ÷ 100 = {{ Math.floor(editForm.point / 100) }}级</span>
        </el-form-item>
        <el-form-item label="贡献">
          <el-input-number v-model="editForm.contri" :min="0" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="editSaving" @click="saveEdit">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminParkData',
  data () {
    return {
      tab: 'users',
      // 玩家数据
      users: [], userTotal: 0, userPage: 1, userSize: 10, userLoading: false, userWord: '',
      // 日志流水
      logs: [], logTotal: 0, logPage: 1, logSize: 15, logLoading: false, logType: 'join', logWord: '',
      // 编辑
      editDlg: false, editSaving: false, editRow: {}, editForm: { love: 0, point: 0, contri: 0 }
    }
  },
  mounted () { this.loadUsers() },
  methods: {
    onTab () {
      if (this.tab === 'users' && this.users.length === 0) this.loadUsers()
      if (this.tab === 'logs' && this.logs.length === 0) this.loadLogs()
    },
    loadUsers () {
      this.userLoading = true
      api.get('/admin/park-users', { params: { page: this.userPage, size: this.userSize, word: this.userWord } }).then(r => {
        this.userLoading = false
        if (r.code === 0) {
          this.users = r.data.list
          this.userTotal = r.data.total
          this.userPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadLogs () {
      this.logLoading = true
      api.get('/admin/park-logs', { params: { page: this.logPage, size: this.logSize, type: this.logType, word: this.logWord } }).then(r => {
        this.logLoading = false
        if (r.code === 0) {
          this.logs = r.data.list
          this.logTotal = r.data.total
          this.logPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openEdit (row) {
      this.editRow = row
      this.editForm = { love: row.love || 0, point: row.point || 0, contri: row.contri || 0 }
      this.editDlg = true
    },
    saveEdit () {
      this.editSaving = true
      api.put('/admin/park-users/' + this.editRow.user_id, this.editForm).then(r => {
        this.editSaving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.loadUsers()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
