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
        <el-table-column prop="nickname" label="昵称" min-width="110" show-overflow-tooltip />
        <el-table-column label="故乡" min-width="220" show-overflow-tooltip>
          <template slot-scope="{row}">{{ addr(row, 'home') }}</template>
        </el-table-column>
        <el-table-column label="现居" min-width="220" show-overflow-tooltip>
          <template slot-scope="{row}">{{ addr(row, 'live') }}</template>
        </el-table-column>
        <el-table-column label="更新时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog :title="'编辑地址：' + (form.nickname || '')" :visible.sync="dlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="70px" size="small">
        <div class="addr-group">故乡</div>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="国家"><el-input v-model.trim="form.home_nation" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="省份"><el-input v-model.trim="form.home_prov" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="城市"><el-input v-model.trim="form.home_city" maxlength="20" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="区县"><el-input v-model.trim="form.home_dist" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="邮编"><el-input v-model.trim="form.home_zip" maxlength="10" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="详细"><el-input v-model.trim="form.home_addr" maxlength="100" /></el-form-item>
        <div class="addr-group">现居</div>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="国家"><el-input v-model.trim="form.live_nation" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="省份"><el-input v-model.trim="form.live_prov" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="城市"><el-input v-model.trim="form.live_city" maxlength="20" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="区县"><el-input v-model.trim="form.live_dist" maxlength="20" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="邮编"><el-input v-model.trim="form.live_zip" maxlength="10" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="详细"><el-input v-model.trim="form.live_addr" maxlength="100" /></el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const FIELDS = ['home_nation', 'home_prov', 'home_city', 'home_dist', 'home_addr', 'home_zip',
  'live_nation', 'live_prov', 'live_city', 'live_dist', 'live_addr', 'live_zip']

export default {
  name: 'AdminUserAddresses',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '', dlg: false, form: {} } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/user-addresses', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    addr (row, p) {
      const s = [row[p + '_nation'], row[p + '_prov'], row[p + '_city'], row[p + '_dist'], row[p + '_addr'], row[p + '_zip']].filter(Boolean).join(' ')
      return s || '—'
    },
    openDlg (row) {
      const f = { id: row.id, nickname: row.nickname }
      FIELDS.forEach(k => { f[k] = row[k] || '' })
      this.form = f
      this.dlg = true
    },
    save () {
      const body = {}
      FIELDS.forEach(k => { body[k] = this.form[k] })
      api.put('/admin/user-addresses/' + this.form.id, body).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>

<style scoped>
.addr-group { font-weight: 600; color: #409eff; margin: 4px 0 12px; border-left: 3px solid #409eff; padding-left: 8px; }
</style>
