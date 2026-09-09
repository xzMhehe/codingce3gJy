<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按手机号/号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:240px" @keyup.enter.native="load" @clear="load" />
        <el-button type="primary" icon="el-icon-search" @click="load">搜索</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="编号" width="70" />
        <el-table-column label="会员" min-width="140">
          <template slot-scope="{row}">
            <a :href="'/api/admin/users/' + row.user_id + '/detail'" @click.prevent=""><b>{{ row.nickname || '—' }}</b></a>
            <span class="txt-fade">（{{ row.user_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column label="状态" width="90">
          <template slot-scope="{row}">
            <el-tag :type="{0:'warning',1:'success',2:'danger'}[row.status]" size="mini">
              {{ {0:'待审核',1:'已通过',2:'已拒绝'}[row.status] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="提交时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
        <el-table-column label="审核时间" width="170"><template slot-scope="{row}">{{ row.handled_at ? fmt(row.handled_at) : '—' }}</template></el-table-column>
        <el-table-column label="操作" width="180" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <template v-if="row.status === 0">
              <el-button size="mini" type="success" plain @click="audit(row, 1)">通过</el-button>
              <el-button size="mini" type="danger" plain @click="audit(row, 2)">拒绝</el-button>
            </template>
            <el-button size="mini" plain @click="remove(row)">删除</el-button>
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
  name: 'AdminUserPhones',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/phones', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    audit (row, status) {
      api.post('/admin/phones/' + row.id + '/audit', { status }).then(r => {
        if (r.code === 0) { this.$message.success(r.data); this.load() } else this.$message.error(r.msg)
      })
    },
    remove (row) {
      this.$confirm('确定删除该手机验证记录吗？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/phones/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>