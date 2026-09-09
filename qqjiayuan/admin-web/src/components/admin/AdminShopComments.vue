<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <span class="help-line">商品评价</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="goods_name" label="商品" min-width="150" show-overflow-tooltip />
        <el-table-column prop="user" label="评价人" min-width="110" />
        <el-table-column label="评级" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="{1:'success',2:'warning',3:'danger'}[row.d_type]" size="mini">{{ dtypeName(row.d_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
        <el-table-column label="店主回复" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.reply || '—' }}</template>
        </el-table-column>
        <el-table-column label="时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="90" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
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
  name: 'AdminShopComments',
  data () { return { list: [], total: 0, page: 1, size: 10, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/shop-comments', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除这条评价？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/shop-comments/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    dtypeName (t) { return { 1: '好评', 2: '中评', 3: '差评' }[t] || '好评' },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>