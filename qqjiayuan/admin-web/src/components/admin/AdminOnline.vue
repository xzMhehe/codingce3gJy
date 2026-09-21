<template>
  <div>
    <!-- 在线查看：登录用户 + 在线游客（30 分钟滑动窗口，与用户端 /online 口径一致） -->
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="label">在线用户 / 游客（近 30 分钟）</span>
        <div class="grow" />
        <el-button icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="序号" width="70" align="center">
          <template slot-scope="{ $index }">{{ (page - 1) * size + $index + 1 }}</template>
        </el-table-column>
        <el-table-column label="类型" width="80" align="center">
          <template slot-scope="{ row }">
            <el-tag :type="row.is_guest ? 'info' : 'success'" size="mini">{{ row.is_guest ? '游客' : '用户' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="名称" min-width="180" show-overflow-tooltip>
          <template slot-scope="{ row }">
            <template v-if="row.is_guest">家园社区游客</template>
            <template v-else><b>{{ row.nickname }}</b>({{ row.username }})</template>
          </template>
        </el-table-column>
        <el-table-column label="IP" width="160">
          <template slot-scope="{ row }">
            <span v-if="row.ip">{{ row.ip }}</span>
            <span v-else class="txt-none">—</span>
          </template>
        </el-table-column>
        <el-table-column label="最后活跃" width="180">
          <template slot-scope="{ row }">{{ fmtTime(row.last_active_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{ row }">
            <el-tag :type="row.banned ? 'danger' : 'success'" size="mini">{{ row.banned ? '已封禁' : '正常' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{ row }">
            <el-button v-if="row.banned" size="mini" type="success" plain icon="el-icon-unlock"
                       @click="unban(row.ban_id, row.ip)">解封</el-button>
            <el-button v-else size="mini" type="danger" plain icon="el-icon-lock"
                       :disabled="!row.ip" @click="openBan(row.ip)">封禁</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[20, 50, 100]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- IP 封禁名单 -->
    <el-card shadow="never" class="box" style="margin-top:16px">
      <div class="toolbar">
        <el-input v-model="banWord" placeholder="按 IP / 原因搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="searchBans" @clear="searchBans" />
        <el-button type="primary" icon="el-icon-search" @click="searchBans">搜索</el-button>
        <div class="grow" />
        <el-button type="danger" icon="el-icon-lock" @click="openBan('')">手动封禁</el-button>
      </div>
      <el-table :data="bans" v-loading="banLoading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="ip" label="IP" width="180" />
        <el-table-column label="原因" min-width="200" show-overflow-tooltip>
          <template slot-scope="{ row }">
            <span v-if="row.reason">{{ row.reason }}</span>
            <span v-else class="txt-none">—</span>
          </template>
        </el-table-column>
        <el-table-column label="操作人" width="140" show-overflow-tooltip>
          <template slot-scope="{ row }">
            <span v-if="row.admin_name">{{ row.admin_name }}</span>
            <span v-else class="txt-none">—</span>
          </template>
        </el-table-column>
        <el-table-column label="封禁时间" width="180">
          <template slot-scope="{ row }">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{ row }">
            <el-button size="mini" type="success" plain icon="el-icon-unlock" @click="unban(row.id, row.ip)">解封</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="banTotal"
                     :page-size="banSize" :current-page="banPage" :page-sizes="[20, 50, 100]"
                     @current-change="p => { banPage = p; loadBans() }"
                     @size-change="s => { banSize = s; banPage = 1; loadBans() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 封禁弹窗 -->
    <el-dialog :title="banForm.locked ? '封禁 IP' : '手动封禁 IP'" :visible.sync="banDlg"
               width="480px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="IP 地址">
          <el-input v-model.trim="banForm.ip" maxlength="45" placeholder="如 127.0.0.1"
                    :disabled="banForm.locked" />
        </el-form-item>
        <el-form-item label="封禁原因">
          <el-input v-model.trim="banForm.reason" maxlength="100" placeholder="选填，例如：恶意刷屏" />
        </el-form-item>
      </el-form>
      <div class="ban-warn">
        <i class="el-icon-warning"></i>
        <div>
          封禁后该 IP 将<b>无法访问全站（含本管理端）</b>，访问页面会被重定向到「服务不可用」。<br>
          若误封了您自己的 IP，需在数据库执行：<br>
          <code>DELETE FROM ip_bans WHERE ip='{{ banForm.ip }}';</code>
        </div>
      </div>
      <div slot="footer">
        <el-button @click="banDlg = false">取消</el-button>
        <el-button type="danger" :loading="banSaving" @click="doBan">确认封禁</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminOnline',
  data () {
    return {
      // 在线列表
      list: [], total: 0, page: 1, size: 20, loading: false,
      // 封禁名单
      bans: [], banTotal: 0, banPage: 1, banSize: 20, banWord: '', banLoading: false,
      banDlg: false, banSaving: false,
      banForm: { ip: '', reason: '', locked: false }
    }
  },
  mounted () {
    this.load()
    this.loadBans()
  },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/online', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          this.page = r.data.page || 1
          this.size = r.data.size || 20
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    loadBans () {
      this.banLoading = true
      api.get('/admin/ip-bans', { params: { page: this.banPage, size: this.banSize, word: this.banWord } }).then(r => {
        this.banLoading = false
        if (r.code === 0) {
          this.bans = r.data.list || []
          this.banTotal = r.data.total || 0
          this.banPage = r.data.page || 1
          this.banSize = r.data.size || 20
        } else this.$message.error(r.msg)
      }).catch(() => { this.banLoading = false })
    },
    searchBans () { this.banPage = 1; this.loadBans() },
    openBan (ip) {
      this.banForm = { ip: ip || '', reason: '', locked: !!ip }
      this.banDlg = true
    },
    doBan () {
      if (!this.banForm.ip) { this.$message.error('请填写要封禁的 IP'); return }
      this.banSaving = true
      api.post('/admin/ip-bans', { ip: this.banForm.ip, reason: this.banForm.reason }).then(r => {
        this.banSaving = false
        if (r.code === 0) {
          this.$message.success('已封禁 ' + this.banForm.ip)
          this.banDlg = false
          this.load()
          this.loadBans()
        } else this.$message.error(r.msg)
      }).catch(() => { this.banSaving = false })
    },
    unban (id, ip) {
      if (!id) { this.$message.error('未找到该 IP 的封禁记录，请刷新后重试'); return }
      this.$confirm('确定解封 IP「' + ip + '」吗？解封后该 IP 可立即恢复访问。', '解封', {
        type: 'warning', confirmButtonText: '确认解封'
      }).then(() => {
        api.delete('/admin/ip-bans/' + id).then(r => {
          if (r.code === 0) {
            this.$message.success('已解封 ' + ip)
            this.load()
            this.loadBans()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmtTime (t) {
      if (!t) return '—'
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' +
        p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>

<style scoped>
.txt-none { color: #c0c4cc; }
.ban-warn {
  display: flex;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 8px;
  background: #fef0f0;
  border: 1px solid #fde2e2;
  color: #f56c6c;
  font-size: 13px;
  line-height: 1.8;
}
.ban-warn i { font-size: 16px; margin-top: 2px; }
.ban-warn code {
  display: inline-block;
  margin-top: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  background: #fff;
  color: #f56c6c;
  font-family: Consolas, Monaco, monospace;
  word-break: break-all;
}
</style>
