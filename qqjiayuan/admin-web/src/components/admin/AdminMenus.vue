<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="help-line">菜单维护：可改名/图标/排序/隐藏，点「修改」单独保存，保存后侧边栏立即生效</span>
      </div>
      <el-table :data="pagedRows" v-loading="loading" row-key="key" default-expand-all
                :tree-props="{ children: 'children' }">
        <el-table-column label="菜单" min-width="240">
          <template slot-scope="{row}">
            <i :class="row.icon" style="margin-right:6px;color:#7c8aa5" />
            <span :style="{ color: row.isGroup ? '#409eff' : '' }">{{ row.name }}</span>
            <el-tag v-if="row._ov" size="mini" type="info" style="margin-left:6px">自定义</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="key" prop="key" min-width="130" />
        <el-table-column label="排序" width="80" align="center">
          <template slot-scope="{row}">{{ row.sort }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.hidden ? 'danger' : 'success'">{{ row.hidden ? '隐藏' : '显示' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEdit(row)">修改</el-button>
            <el-button size="mini" plain :disabled="!row._ov" @click="resetOne(row)">恢复默认</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[5, 10, 20, 50]"
                     @current-change="p => { page = p }" @size-change="s => { size = s; page = 1 }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog title="修改菜单" :visible.sync="dlgShow" width="420px" append-to-body>
      <el-form label-width="80px" size="small">
        <el-form-item label="菜单名称"><el-input v-model="edit.name" maxlength="20" /></el-form-item>
        <el-form-item label="图标">
          <el-input v-model="edit.icon" placeholder="el-icon-xxx">
            <template slot="prepend"><i :class="edit.icon" /></template>
          </el-input>
        </el-form-item>
        <el-form-item label="排序"><el-input-number v-model="edit.sort" :min="-999" :max="999" controls-position="right" /></el-form-item>
        <el-form-item label="隐藏"><el-switch v-model="edit.hidden" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="dlgShow = false">取 消</el-button>
        <el-button size="small" type="primary" @click="saveOne">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'
import { cloneMenu } from '../../menu'

export default {
  name: 'AdminMenus',
  data () { return { rows: [], loading: false, dlgShow: false, edit: {}, page: 1, size: 5, total: 0 } },
  computed: {
    // 一级分组才参与分页，子菜单随所属分组整树展示
    pagedRows () {
      const start = (this.page - 1) * this.size
      return this.rows.slice(start, start + this.size)
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/menus').then(ov => {
        this.loading = false
        if (!ov || ov.code !== 0) return
        const keyMap = {}
        ;(ov.data || []).forEach(o => { keyMap[o.key] = o })
        const walk = items => items.map((it, i) => {
          const o = keyMap[it.key] || {}
          const n = {
            ...it,
            name: o.name || it.name,
            icon: o.icon || it.icon,
            sort: o.id ? o.sort : i + 1,
            hidden: o.id ? o.hidden : 0,
            isGroup: !!it.children,
            _ov: !!o.id
          }
          if (it.children) n.children = walk(it.children)
          return n
        })
        this.rows = walk(cloneMenu())
        this.total = this.rows.length
      })
    },
    openEdit (row) {
      this.edit = { ...row }
      this.dlgShow = true
    },
    saveOne () {
      const { key, name, icon, sort, hidden } = this.edit
      api.put('/admin/menus', { key, name, icon, perm: this.edit.perm || '', sort: sort || 0, hidden: hidden || 0 }).then(r => {
        if (r.code === 0) {
          this.$message.success('已保存')
          this.dlgShow = false
          window.dispatchEvent(new Event('menu-updated'))
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    resetOne (row) {
      api.delete('/admin/menus/' + row.key).then(r => {
        if (r.code === 0) {
          this.$message.success('已恢复默认')
          window.dispatchEvent(new Event('menu-updated'))
          this.load()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>
