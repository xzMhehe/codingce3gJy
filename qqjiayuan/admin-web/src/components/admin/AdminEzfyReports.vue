<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="玩家ID / 家园号 / 昵称" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="type" placeholder="战报类型" clearable style="width:140px; margin-left: 8px">
          <el-option v-for="o in typeOptions" :key="o.v" :label="o.n" :value="o.v" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="归属账号" width="90" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.user_id }}</span></template>
        </el-table-column>
        <el-table-column label="玩家" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.player_name }}</span>
            <span class="td-gray">({{ row.home_num }})</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="tagType(row.type_name)">{{ row.type_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="preview" label="内容预览" min-width="220" show-overflow-tooltip />
        <el-table-column label="已读" width="70" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.is_read ? 'info' : 'warning'">{{ row.is_read ? '已读' : '未读' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="170" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
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
    <el-dialog :title="detail ? detail.type_name + ' · ' + detail.title : '战报详情'"
               :visible.sync="detailDlg" width="720px" :close-on-click-modal="false">
      <div v-if="detail">
        <el-descriptions :column="2" size="medium" border>
          <el-descriptions-item label="归属账号">{{ detail.user_id }}</el-descriptions-item>
          <el-descriptions-item label="玩家">{{ detail.player_name }}（{{ detail.home_num }}）</el-descriptions-item>
          <el-descriptions-item label="战报类型">
            <el-tag size="small" :type="tagType(detail.type_name)">{{ detail.type_name }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="时间">{{ fmtTime(detail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag size="mini" :type="detail.is_read ? 'info' : 'warning'">{{ detail.is_read ? '已读' : '未读' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="标题" :span="2">{{ detail.title }}</el-descriptions-item>
          <el-descriptions-item label="城市ID">{{ detail.city_id || '—' }}</el-descriptions-item>
          <el-descriptions-item label="订单ID">{{ detail.order_id || '—' }}</el-descriptions-item>
        </el-descriptions>
        <div class="sub-title">战报正文</div>
        <div class="log-box">{{ detail.content || '（空）' }}</div>
        <template v-if="detail.detail">
          <div class="sub-title">附加明细</div>
          <div class="log-box">{{ detail.detail }}</div>
        </template>
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
  name: 'AdminEzfyReports',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', type: 0,
      typeOptions: [
        { v: 0, n: '全部类型' },
        { v: 1, n: '侦察' },
        { v: 2, n: '掠夺/战斗' },
        { v: 3, n: '征服' },
        { v: 4, n: '战斗' },
        { v: 5, n: '采集/派遣' },
        { v: 6, n: '系统' }
      ],
      detailDlg: false, detail: null
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      const params = { page: this.page, size: this.size, word: this.word }
      if (this.type > 0) params.type = this.type
      api.get('/admin/ezfy-reports', { params }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    tagType (name) {
      switch (name) {
        case '征服': case '占领': case '城破': case '掠夺': case '摧毁': return 'danger'
        case '被掠夺': case '被征服': case '被侦查': case '预警': return 'warning'
        case '侦查': case '采集': case '驻守采集': case '运输': case '派遣': case '增援': case '返航': return 'info'
        default: return 'success'
      }
    },
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    openDetail (row) {
      this.detail = null
      this.detailDlg = true
      api.get('/admin/ezfy-reports/' + row.id).then(r => {
        if (r.code === 0) {
          // ★ 2026-10-07 修复详情 bug：后端返回的是 { report, player_name, home_num, type_name }，
          //   玩家/家园号/类型名都在 report **外层**。原来只取 r.data.report，
          //   导致弹窗标题显示「undefined · 标题」、玩家显示「（undefined）」、类型标签恒为默认色。
          const d = r.data || {}
          const rep = d.report || {}
          this.detail = Object.assign({}, rep, {
            player_name: d.player_name || rep.player_name || '',
            home_num: d.home_num || '',
            type_name: d.type_name || ''
          })
        } else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    remove (row) {
      this.$confirm('确认删除战报【' + row.title + '】？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-reports/' + row.id).then(r => {
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
.log-box { max-height: 320px; overflow-y: auto; border: 1px solid #ebeef5; border-radius: 4px; padding: 8px 10px; background: #fafbfc; line-height: 1.7; white-space: pre-wrap; word-break: break-all; font-size: 13px; color: #303133; }
</style>
