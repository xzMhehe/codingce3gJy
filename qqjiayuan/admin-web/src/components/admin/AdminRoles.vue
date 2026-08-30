<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新建角色</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="角色名" min-width="110">
          <template slot-scope="{row}"><b>{{ row.name }}</b></template>
        </el-table-column>
        <el-table-column prop="code" label="编码" width="120" />
        <el-table-column label="权限" min-width="240">
          <template slot-scope="{row}">
            <el-tag v-for="p in (row.permissions || [])" :key="p.id" size="mini" style="margin:1px 4px 1px 0">{{ p.name }}</el-tag>
            <span v-if="!(row.permissions || []).length" class="help-line">无</span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="说明" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="280" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" icon="el-icon-key" @click="openPerms(row)">分配权限</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" v-if="row.code !== 'super_admin' && row.code !== 'member'" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 新建 / 编辑角色 模态框 -->
    <el-dialog :title="form.id ? '编辑角色' : '新建角色'" :visible.sync="dlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="角色名">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model.trim="form.code" maxlength="30" :disabled="!!form.id" placeholder="英文编码，如 operator" />
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

    <!-- 分配权限 模态框 -->
    <el-dialog :title="'分配权限：' + (permRole ? permRole.name : '')" :visible.sync="permDlg" width="520px" :close-on-click-modal="false">
      <el-checkbox-group v-model="permIds">
        <el-checkbox v-for="p in perms" :key="p.id" :label="p.id" style="display:block;margin:6px 0">
          {{ p.name }}（{{ p.code }}）<span class="help-line"> {{ p.remark }}</span>
        </el-checkbox>
      </el-checkbox-group>
      <div slot="footer">
        <el-button @click="permDlg = false">取 消</el-button>
        <el-button type="primary" @click="savePerms">保存权限</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminRoles',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      perms: [], dlg: false,
      permDlg: false, permRole: null, permIds: [],
      form: { id: 0, name: '', code: '', remark: '' }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/roles', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
      api.get('/admin/permissions').then(r => { if (r.code === 0) this.perms = r.data })
    },
    openDlg (row) {
      if (row) this.form = { id: row.id, name: row.name, code: row.code, remark: row.remark }
      else this.form = { id: 0, name: '', code: '', remark: '' }
      this.dlg = true
    },
    save () {
      if (!this.form.name || !this.form.code) { this.$message.error('角色名和编码必填'); return }
      const call = this.form.id ? api.put('/admin/roles/' + this.form.id, this.form) : api.post('/admin/roles', this.form)
      call.then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    openPerms (row) {
      this.permRole = row
      this.permIds = (row.permissions || []).map(p => p.id)
      this.permDlg = true
    },
    savePerms () {
      api.put(`/admin/roles/${this.permRole.id}/perms`, { perm_ids: this.permIds }).then(r => {
        if (r.code === 0) { this.$message.success('权限已保存'); this.permDlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除角色「${row.name}」吗？该角色用户将失去对应权限。`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/roles/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
