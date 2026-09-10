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
        <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
        <el-table-column label="QQ" width="130" align="center">
          <template slot-scope="{row}">{{ row.qq || '—' }}</template>
        </el-table-column>
        <el-table-column label="邮箱" min-width="170" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.mail || '—' }}</template>
        </el-table-column>
        <el-table-column label="手机" width="130" align="center">
          <template slot-scope="{row}">{{ row.phone || '—' }}</template>
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

    <el-dialog :title="'编辑联系方式：' + (form.nickname || '')" :visible.sync="dlg" width="420px" :close-on-click-modal="false">
      <el-form label-width="70px" size="small">
        <el-form-item label="QQ"><el-input v-model.trim="form.qq" maxlength="11" style="width:240px" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model.trim="form.mail" maxlength="50" style="width:240px" /></el-form-item>
        <el-form-item label="手机"><el-input v-model.trim="form.phone" maxlength="11" style="width:240px" /></el-form-item>
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

export default {
  name: 'AdminUserContacts',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '', dlg: false, form: {} } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/user-contacts', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      this.form = { id: row.id, nickname: row.nickname, qq: row.qq || '', mail: row.mail || '', phone: row.phone || '' }
      this.dlg = true
    },
    save () {
      api.put('/admin/user-contacts/' + this.form.id, { qq: this.form.qq, mail: this.form.mail, phone: this.form.phone }).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>
