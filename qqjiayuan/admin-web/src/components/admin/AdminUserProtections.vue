<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" align="center" />
        <el-table-column prop="nickname" label="昵称" min-width="130" show-overflow-tooltip />
        <el-table-column label="密保问题" min-width="230" show-overflow-tooltip>
          <template slot-scope="{row}">{{ questionOf(row.issue) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}"><el-tag type="success" size="mini">已设置</el-tag></template>
        </el-table-column>
        <el-table-column label="更新时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain @click="clear(row)">清除</el-button>
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

const QUESTIONS = ['您母亲的姓名是？', '您父亲的姓名是？', '您的小学名称是？', '您的出生地是？',
  '您最爱的电影是？', '您第一只宠物的名字是？', '您最喜欢的运动是？', '您母亲的生日是？']

export default {
  name: 'AdminUserProtections',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '' } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/user-protections', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    clear (row) {
      this.$confirm('确定清除「' + row.nickname + '」的密保吗？清除后该会员需重新设置。', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/user-protections/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已清除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    questionOf (i) { return QUESTIONS[i - 1] || ('问题' + i) },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>
