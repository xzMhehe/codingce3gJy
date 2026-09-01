<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按板块名称搜索" prefix-icon="el-icon-search" clearable
                  style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增板块</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="名称" min-width="170" show-overflow-tooltip>
          <template slot-scope="{row}">
            <template v-if="row.parent_id">└ </template>
            <b v-else>{{ row.name }}</b>
            <span v-if="row.parent_id">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="层级" width="90">
          <template slot-scope="{row}">{{ row.parent_id ? '子板块' : '分区' }}</template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="240" show-overflow-tooltip />
        <el-table-column prop="sort" label="排序" width="60" />
        <el-table-column label="状态" width="70">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="thread_count" label="帖数" width="60" />
        <el-table-column label="操作" width="180" fixed="right" header-align="center">
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

    <!-- 编辑 / 新增板块 模态框 -->
    <el-dialog :title="form.id ? '编辑板块：' + form.name : '新增板块'" :visible.sync="dlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="上级">
          <el-select v-model.number="form.parent_id" style="width:100%">
            <el-option :value="0" label="作为分区" />
            <el-option v-for="ch in channels" :key="ch.id" :value="ch.id" :label="ch.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model.trim="form.description" maxlength="200" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="显示" inactive-text="隐藏" />
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
  name: 'AdminBoards',
  data () {
    return {
      list: [], channels: [], total: 0, page: 1, pages: 1, size: 10, word: '', loading: false,
      dlg: false, delId: 0,
      form: { id: 0, name: '', parent_id: 0, description: '', sort: 0, status: 1 }
    }
  },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/boards', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
          this.channels = this.list.filter(b => b.parent_id === 0)
          api.get('/admin/boards', { params: { page: 1, size: 100 } }).then(x => {
            if (x.code === 0) this.channels = x.data.list.filter(b => b.parent_id === 0)
          })
        } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      if (row) {
        this.form = { id: row.id, name: row.name, parent_id: row.parent_id, description: row.description, sort: row.sort, status: row.status }
      } else {
        this.form = { id: 0, name: '', parent_id: 0, description: '', sort: 0, status: 1 }
      }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.error('请填写名称'); return }
      const call = this.form.id ? api.put('/admin/boards/' + this.form.id, this.form) : api.post('/admin/boards', this.form)
      call.then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除「${row.name}」吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/boards/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
