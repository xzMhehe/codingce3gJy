<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新建马甲</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="图标" width="80">
          <template slot-scope="{row}"><img class="bicon" :src="'/static/picture/' + row.icon" :alt="row.name"></template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="110" />
        <el-table-column prop="remark" label="说明" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 新建 / 编辑马甲 模态框 -->
    <el-dialog :title="form.id ? '编辑马甲：' + form.name : '新建马甲'" :visible.sync="dlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="图标">
          <el-select v-model="form.icon" placeholder="选择图标" style="width:100%">
            <el-option v-for="ic in icons" :key="ic" :value="ic" :label="ic">
              <img class="bicon" :src="'/static/picture/' + ic" :alt="ic">{{ ic }}
            </el-option>
          </el-select>
          <img v-if="form.icon" class="bicon" :src="'/static/picture/' + form.icon" alt="预览" style="margin-top:4px">
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model.trim="form.remark" maxlength="100" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminBadges',
  data () {
    return { list: [], total: 0, page: 1, size: 10, loading: false, icons: [], dlg: false, form: { id: 0, name: '', icon: '', remark: '' } }
  },
  mounted () {
    this.load()
    api.get('/badge-presets').then(r => {
      if (r.code === 0) {
        this.icons = r.data.badge_icons
        if (!this.form.icon && this.icons.length) this.form.icon = this.icons[0]
      }
    })
  },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/badges', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      if (row) this.form = { id: row.id, name: row.name, icon: row.icon, remark: row.remark }
      else this.form = { id: 0, name: '', icon: this.icons[0] || '', remark: '' }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.error('请填写名称'); return }
      const call = this.form.id ? api.put('/admin/badges/' + this.form.id, this.form) : api.post('/admin/badges', this.form)
      call.then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除马甲「${row.name}」吗？将同时从所有用户身上摘下。`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/badges/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
