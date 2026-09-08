<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="help-line">分类用于在分区下对子板块二次分组（参考诺哈「分区→分类→版块」三级）。</span>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增分类</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="所属分区" min-width="140">
          <template slot-scope="{row}">{{ channelName(row.parent_id) }}</template>
        </el-table-column>
        <el-table-column prop="name" label="分类名称" min-width="160" />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="操作" width="160">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="form.id ? '编辑分类' : '新增分类'" :visible.sync="dlg" width="440px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="所属分区">
          <el-select v-model.number="form.parent_id" style="width:100%">
            <el-option v-for="ch in channels" :key="ch.id" :value="ch.id" :label="ch.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
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
  name: 'AdminBoardCategories',
  data () {
    return { list: [], channels: [], loading: false, dlg: false, form: { id: 0, parent_id: 0, name: '', sort: 0 } }
  },
  mounted () { this.load(); this.loadChannels() },
  methods: {
    loadChannels () {
      api.get('/admin/boards', { params: { page: 1, size: 100 } }).then(r => { if (r.code === 0) this.channels = r.data.list.filter(b => b.parent_id === 0) })
    },
    channelName (id) {
      const c = this.channels.find(x => x.id === id)
      return c ? c.name : ('#' + id)
    },
    load () {
      this.loading = true
      api.get('/admin/board-categories').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
        else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, parent_id: row.parent_id, name: row.name, sort: row.sort } : { id: 0, parent_id: this.channels[0] ? this.channels[0].id : 0, name: '', sort: 0 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.error('请填写分类名称'); return }
      if (!this.form.parent_id) { this.$message.error('请选择所属分区'); return }
      const call = this.form.id ? api.put('/admin/board-categories/' + this.form.id, this.form) : api.post('/admin/board-categories', this.form)
      call.then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除分类「${row.name}」吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/board-categories/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>