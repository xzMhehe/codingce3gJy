<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按店铺名/店主搜索" prefix-icon="el-icon-search" clearable style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <span class="help-line">C2C 店铺列表（对齐诺哈 商城管理）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="店铺名" min-width="160" />
        <el-table-column prop="nickname" label="店主" min-width="120" />
        <el-table-column prop="username" label="号码" width="100" />
        <el-table-column prop="goods_cnt" label="商品数" width="90" align="center" />
        <el-table-column label="状态" width="90">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '营业中' : '已关闭' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开店时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminShops',
  data () { return { list: [], total: 0, page: 1, size: 10, word: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/shops', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>