<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="txt-fade">设置广场 T台秀（膜拜）上榜用户</span>
        <div class="grow" />
      </div>
      <el-form inline>
        <el-form-item label="当前上榜">
          <el-tag type="warning">{{ cur.nickname || '—' }}（{{ cur.id || '自动' }} · 经验 {{ cur.exp || 0 }}）</el-tag>
        </el-form-item>
        <el-form-item label="指定号码">
          <el-input v-model="uid" placeholder="输入用户号码" style="width:180px" />
          <el-button type="primary" @click="setTtou">设为上榜</el-button>
          <el-button type="danger" plain @click="clearTtou">移除指定</el-button>
        </el-form-item>
      </el-form>
      <div class="module-content">留空则自动取经验最高用户；指定号码后固定该用户上榜。用户点「我要上榜」提交的申请见下方列表，可一键设为秀主。</div>
    </el-card>

    <el-card shadow="never" class="box" style="margin-top:12px">
      <div class="toolbar">
        <span class="txt-fade">上榜申请（{{ total }} 条待选）</span>
        <div class="grow" />
        <el-button size="mini" @click="loadApplies">刷新</el-button>
      </div>
      <el-table :data="applies" size="mini" border stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="申请人" min-width="160">
          <template slot-scope="{ row }">
            <el-avatar v-if="row.avatar_base64" :src="row.avatar_base64" :size="24" shape="square" style="vertical-align:middle;margin-right:6px" />
            <el-avatar v-else-if="row.avatar" :src="'/static/picture/' + row.avatar" :size="24" shape="square" style="vertical-align:middle;margin-right:6px" />
            <span :style="{ color: row.color || '#333' }">{{ row.nickname }}</span>
            <span class="txt-fade">({{ row.username }})</span>
          </template>
        </el-table-column>
        <el-table-column prop="slogan" label="上榜宣言" min-width="200" show-overflow-tooltip />
        <el-table-column prop="created_at" label="申请时间" width="150">
          <template slot-scope="{ row }">{{ row.created_at ? row.created_at.replace('T', ' ').slice(0, 16) : '' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template slot-scope="{ row }">
            <el-button type="primary" size="mini" @click="accept(row)">设为秀主</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        style="margin-top:10px;text-align:right"
        background layout="prev, pager, next, total"
        :total="total" :page-size="size" :current-page.sync="page"
        @current-change="loadApplies" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminTtou',
  data () { return { cur: {}, uid: '', applies: [], total: 0, page: 1, size: 10 } },
  mounted () { this.load(); this.loadApplies() },
  methods: {
    load () { api.get('/admin/ttou').then(r => { if (r.code === 0) this.cur = r.data }) },
    loadApplies () {
      api.get('/admin/ttou/applies', { params: { page: this.page, size: this.size } }).then(r => {
        if (r.code === 0) { this.applies = r.data.list || []; this.total = r.data.total || 0 }
      })
    },
    setTtou () {
      if (!this.uid) { this.$message.warning('请填写用户号码'); return }
      api.put('/admin/ttou', { user_id: Number(this.uid) }).then(r => { if (r.code === 0) { this.$message.success('已设置'); this.load() } else this.$message.error(r.msg) })
    },
    clearTtou () {
      api.delete('/admin/ttou').then(r => { if (r.code === 0) { this.$message.success('已移除指定，恢复自动'); this.load() } else this.$message.error(r.msg) })
    },
    accept (row) {
      this.$confirm(`将「${row.nickname}」设为当前秀主？`, '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ttou/applies/' + row.id + '/accept').then(r => {
          if (r.code === 0) { this.$message.success('已设为秀主'); this.load(); this.loadApplies() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
