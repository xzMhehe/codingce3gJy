<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="内容 / 昵称 / 家园号 / 用户ID" clearable style="width:250px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="isRead" style="width:130px" @change="page = 1; load()">
          <el-option label="全部" :value="-1" />
          <el-option label="未读" :value="0" />
          <el-option label="已读" :value="1" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="warning" icon="el-icon-delete" @click="openClear">清空某玩家私聊</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="640">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column label="发送方" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.sender_name || ('用户' + row.sender_id) }}</span>
            <span class="td-muted">（{{ row.sender_num || row.sender_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column label="接收方" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.receiver_name || ('用户' + row.receiver_id) }}</span>
            <span class="td-muted">（{{ row.receiver_num || row.receiver_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="内容" min-width="280" show-overflow-tooltip />
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.is_read === 1 ? 'success' : 'warning'">{{ row.is_read === 1 ? '已读' : '未读' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="150" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 清空某玩家私聊 -->
    <el-dialog title="清空某玩家私聊" :visible.sync="clearDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="120px" size="small">
        <el-form-item label="用户ID" required>
          <el-input-number v-model.number="clearForm.user_id" :min="1" controls-position="right" />
          <div class="td-sub" style="margin-top:4px">即会员用户ID（不是家园号）</div>
        </el-form-item>
        <el-form-item label="清空范围">
          <el-radio-group v-model="clearForm.dir">
            <el-radio label="all">全部（收发）</el-radio>
            <el-radio label="from">该玩家发出的</el-radio>
            <el-radio label="to">该玩家收到的</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <em>提示：此操作会物理删除私信记录，不可恢复</em>
      <div slot="footer">
        <el-button @click="clearDlg = false">取 消</el-button>
        <el-button type="danger" :loading="saving" @click="doClear">清 空</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyPrivchat',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, word: '', isRead: -1,
      clearDlg: false, clearForm: { user_id: 1, dir: 'all' }, saving: false
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-privchats', {
        params: { page: this.page, size: this.size, word: this.word, is_read: this.isRead }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确认删除这条私信记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-privchats/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openClear () {
      this.clearForm = { user_id: 1, dir: 'all' }
      this.clearDlg = true
    },
    doClear () {
      this.$confirm('清空后不可恢复，确认继续？', '危险操作', { type: 'error' }).then(() => {
        this.saving = true
        api.post('/admin/ezfy-privchats/clear', this.clearForm).then(r => {
          this.saving = false
          if (r.code === 0) { this.clearDlg = false; this.$message.success(r.data.msg || '已清空'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
