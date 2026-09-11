<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column label="归属账号" width="90" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.user_id }}</span></template>
        </el-table-column>
        <el-table-column label="对战双方" min-width="170" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.my_nick }}</span>
            <span class="td-gray"> VS </span>
            <span class="td-main">{{ row.opp_nick }}</span>
          </template>
        </el-table-column>
        <el-table-column label="双方等级" width="100" align="center">
          <template slot-scope="{row}">{{ row.my_level }} vs {{ row.opp_level }}</template>
        </el-table-column>
        <el-table-column label="结果" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.result === '胜利' ? 'success' : (row.result === '失败' ? 'danger' : 'info')">{{ row.result }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="奖励" width="100" align="center">
          <template slot-scope="{row}">+{{ row.exp }}经验/+{{ row.coin }}G</template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="150" />
        <el-table-column label="操作" width="150" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="战报详情" @click="openDetail(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="remove(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <!-- 战报详情 -->
    <el-dialog title="比武记录详情" :visible.sync="detailDlg" width="640px" :close-on-click-modal="false">
      <div v-if="detail">
        <el-descriptions :column="2" size="medium" border>
          <el-descriptions-item label="归属账号">{{ detail.user_id }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ detail.created_at }}</el-descriptions-item>
          <el-descriptions-item label="玩家1">
            {{ detail.my_nick }}{{ detail.my_uid !== detail.user_id ? '（' + detail.my_uid + '）' : '' }}
            ({{ detail.my_level }}级 气血 {{ detail.my_cur_hp }}/{{ detail.my_max_hp }})
          </el-descriptions-item>
          <el-descriptions-item label="玩家2">
            {{ detail.opp_nick }}{{ detail.opp_uid !== detail.user_id ? '（' + detail.opp_uid + '）' : '' }}
            ({{ detail.opp_level }}级 气血 {{ detail.opp_cur_hp }}/{{ detail.opp_max_hp }})
          </el-descriptions-item>
          <el-descriptions-item label="结果">
            <el-tag :type="detail.result === '胜利' ? 'success' : (detail.result === '失败' ? 'danger' : 'info')" size="small">{{ detail.result }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="奖励">+{{ detail.exp }} 经验 / +{{ detail.coin }} G币</el-descriptions-item>
        </el-descriptions>
        <div class="sub-title">整场战报（{{ logLines.length }} 条）</div>
        <div class="log-box">
          <div v-for="(l, i) in logLines" :key="i" class="log-line">{{ i + 1 }}.{{ l }}</div>
        </div>
      </div>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminJwtRecords',
  data () {
    return { list: [], total: 0, page: 1, size: 10, loading: false, word: '', detailDlg: false, detail: null }
  },
  mounted () { this.load() },
  computed: {
    logLines () {
      if (!this.detail || !this.detail.logs) return []
      return this.detail.logs.split('\n').filter(l => l.trim() !== '')
    }
  },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/jwt-records', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDetail (row) {
      this.detail = null
      this.detailDlg = true
      api.get('/admin/jwt-records/' + row.id + '/detail').then(r => {
        if (r.code === 0) this.detail = r.data.detail
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    remove (row) {
      this.$confirm('确认删除该比武记录【' + row.my_nick + ' VS ' + row.opp_nick + '】？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/jwt-records/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            this.load()
          } else this.$message.error(r.msg)
        })
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 12px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
.log-box { max-height: 320px; overflow-y: auto; border: 1px solid #ebeef5; border-radius: 4px; padding: 8px 10px; background: #fafbfc; line-height: 1.7; }
.log-line { font-size: 13px; color: #303133; }
</style>