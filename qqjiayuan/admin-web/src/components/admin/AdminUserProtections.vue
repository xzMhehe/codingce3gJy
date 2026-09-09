<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar"><span class="help-line">会员密保设置</span></div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column prop="nickname" label="昵称" min-width="110" />
        <el-table-column label="密保问题" min-width="220">
          <template slot-scope="{row}">{{ questionOf(row.issue) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="{row}"><el-tag type="success" size="mini">已设置</el-tag></template>
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

const QUESTIONS = ['您母亲的姓名是？', '您父亲的姓名是？', '您的小学名称是？', '您的出生地是？',
  '您最爱的电影是？', '您第一只宠物的名字是？', '您最喜欢的运动是？', '您母亲的生日是？']

export default {
  name: 'AdminUserProtections',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/user-protections', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    questionOf (i) { return QUESTIONS[i - 1] || ('问题' + i) },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>