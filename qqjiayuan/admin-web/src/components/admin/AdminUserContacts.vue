<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar"><span class="help-line">会员联系方式：QQ / 邮箱 / 手机（对齐诺哈 会员联系）</span></div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column prop="nickname" label="昵称" min-width="110" />
        <el-table-column label="QQ" min-width="110"><template slot-scope="{row}">{{ row.qq || '—' }}</template></el-table-column>
        <el-table-column label="邮箱" min-width="160"><template slot-scope="{row}">{{ row.mail || '—' }}</template></el-table-column>
        <el-table-column label="手机" min-width="120"><template slot-scope="{row}">{{ row.phone || '—' }}</template></el-table-column>
        <el-table-column label="更新时间" width="170"><template slot-scope="{row}">{{ fmt(row.updated_at) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminUserContacts',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/user-contacts', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>