<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar"><span class="help-line">会员通信地址：故乡 / 现居（对齐诺哈 会员地址）</span></div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column prop="nickname" label="昵称" min-width="100" />
        <el-table-column label="故乡" min-width="200">
          <template slot-scope="{row}">{{ addr(row, 'home') }}</template>
        </el-table-column>
        <el-table-column label="现居" min-width="200">
          <template slot-scope="{row}">{{ addr(row, 'live') }}</template>
        </el-table-column>
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
  name: 'AdminUserAddresses',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/user-addresses', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    addr (row, p) {
      const s = [row[p + '_nation'], row[p + '_prov'], row[p + '_city'], row[p + '_dist'], row[p + '_addr'], row[p + '_zip']].filter(Boolean).join(' ')
      return s || '—'
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>