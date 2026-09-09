<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <span class="help-line">家园列表（活跃点排行）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column label="主人" min-width="120">
          <template slot-scope="{row}"><b>{{ row.nickname }}</b></template>
        </el-table-column>
        <el-table-column label="家园名" min-width="120"><template slot-scope="{row}">{{ row.name || '—' }}</template></el-table-column>
        <el-table-column prop="point" label="活跃点" width="90" align="center" />
        <el-table-column label="等级" width="80" align="center">
          <template slot-scope="{row}">{{ homeLevel(row.point) }}</template>
        </el-table-column>
        <el-table-column prop="moods" label="心情数" width="90" align="center" />
        <el-table-column prop="visitors" label="访客数" width="90" align="center" />
        <el-table-column prop="messages" label="留言数" width="90" align="center" />
        <el-table-column label="最近活跃" width="120"><template slot-scope="{row}">{{ fmtDay(row.last_active_date) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminHomes',
  data () { return { list: [], total: 0, page: 1, size: 10, word: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/homes', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    homeLevel (p) {
      if (!p || p < 0) p = 0
      return Math.floor((Math.sqrt(16 + 4 * p) - 4) / 2)
    },
    fmtDay (d) { return d || '—' }
  }
}
</script>