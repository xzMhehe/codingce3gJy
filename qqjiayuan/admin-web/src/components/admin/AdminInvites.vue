<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="help-line">会员邀请/推荐关系（对齐诺哈 会员推荐）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="code" label="邀请码" width="120" />
        <el-table-column prop="inviter_nick" label="邀请人" min-width="120" />
        <el-table-column label="状态" width="100">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '已使用' : '未使用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="被邀请人" min-width="120">
          <template slot-scope="{row}">{{ row.used_nick || '—' }}</template>
        </el-table-column>
        <el-table-column label="使用时间" width="170"><template slot-scope="{row}">{{ fmt(row.used_at) }}</template></el-table-column>
        <el-table-column label="创建时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
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
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/invites', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>