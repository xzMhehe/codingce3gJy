<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="create">生成邀请码</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="邀请码" width="130" align="center">
          <template slot-scope="{row}"><span class="code">{{ row.code }}</span></template>
        </el-table-column>
        <el-table-column label="邀请人" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">
            <template v-if="row.user_id">{{ row.inviter_nick || row.user_id }}</template>
            <el-tag v-else size="mini" type="info">系统</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '已使用' : '未使用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="被邀请人" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.used_nick || '—' }}</template>
        </el-table-column>
        <el-table-column label="使用时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.used_at) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button v-if="row.status !== 1" size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminInvites',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '' } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/invites', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    create () {
      this.$prompt('归属会员家园号（留空生成系统邀请码）', '生成邀请码', { inputValue: '' }).then(({ value }) => {
        const uid = parseInt(value) || 0
        api.post('/admin/invites', { user_id: uid }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已生成'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    del (row) {
      this.$confirm('确定删除邀请码「' + row.code + '」吗？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/invites/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>

<style scoped>
.code { font-family: monospace; font-weight: 600; letter-spacing: 1px; }
</style>
