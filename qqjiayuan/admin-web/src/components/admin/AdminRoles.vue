<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新建角色</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="角色名" min-width="120">
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
        <el-table-column label="操作" width="280" fixed="right" header-align="center">
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
    <el-dialog :title="'分配权限：' + (permRole ? permRole.name : '')" :visible.sync="permDlg" width="560px" :close-on-click-modal="false">
      <el-input v-model.trim="permSearch" size="small" clearable placeholder="搜索权限名 / 权限码 / 分组（如：二战、风云、ezfy）"
                style="margin-bottom:8px" />
      <div style="max-height:420px;overflow-y:auto;padding-right:6px">
        <template v-for="(g, gi) in permGroups">
          <div :key="'g'+gi" style="margin:8px 0 4px;font-weight:600;color:#606266">
            {{ g.name }}（{{ g.items.length }}）
          </div>
          <el-checkbox-group v-model="permIds" :key="'c'+gi" style="padding-left:6px">
            <el-checkbox v-for="p in g.items" :key="p.id" :label="p.id" style="display:inline-block;margin:4px 14px 4px 0;width:47%">
              {{ p.name }}（{{ p.code }}）
            </el-checkbox>
          </el-checkbox-group>
        </template>
        <div v-if="!permGroups.length" class="help-line" style="padding:8px 4px">没有匹配的权限</div>
      </div>
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
      permDlg: false, permRole: null, permIds: [], permSearch: '',
      form: { id: 0, name: '', code: '', remark: '' }
    }
  },
  mounted () { this.load() },
  computed: {
    // ★ 2026-09-27 「二战游戏分配权限的菜单不全」：按分组（permissions.remark）分组展示，
    //   支持按权限名/权限码/分组关键词搜索，二战（风云/ezfy）权限一目了然。
    permGroups () {
      const kw = (this.permSearch || '').trim().toLowerCase()
      const hits = kw ? this.perms.filter(p => {
        return (p.name || '').toLowerCase().includes(kw) ||
               (p.code || '').toLowerCase().includes(kw) ||
               (p.remark || '').toLowerCase().includes(kw)
      }) : this.perms
      const groups = []
      const map = {}
      hits.forEach(p => {
        const g = p.remark || '其他'
        if (!map[g]) { map[g] = { name: g, items: [] }; groups.push(map[g]) }
        map[g].items.push(p)
      })
      return groups
    }
  },
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
