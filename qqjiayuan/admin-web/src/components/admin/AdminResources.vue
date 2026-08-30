<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="category" placeholder="全部分类" clearable style="width:140px" @change="search">
          <el-option value="badge" label="勋章图标" />
          <el-option value="avatar" label="头像" />
          <el-option value="game" label="游戏Logo" />
          <el-option value="priv" label="特权等级" />
          <el-option value="other" label="其他" />
        </el-select>
        <el-input v-model="word" placeholder="搜索文件名/名称" prefix-icon="el-icon-search" clearable
                  style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">筛选</el-button>
        <div class="grow" />
        <el-button icon="el-icon-refresh" :loading="syncing" @click="sync">同步目录</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="预览" width="80">
          <template slot-scope="{row}"><img :src="'/static/' + row.file" class="preview" :alt="row.name"></template>
        </el-table-column>
        <el-table-column prop="file" label="文件" min-width="200" show-overflow-tooltip />
        <el-table-column prop="name" label="名称" width="140" show-overflow-tooltip />
        <el-table-column label="分类" width="100">
          <template slot-scope="{row}">{{ catName(row.category) }}</template>
        </el-table-column>
        <el-table-column label="特权等级" width="90">
          <template slot-scope="{row}">{{ row.category === 'priv' && row.level ? 'Lv.' + row.level : '—' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '停用' : '启用' }}</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50, 100]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 编辑 模态框 -->
    <el-dialog title="编辑资源" :visible.sync="dlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="文件">
          <el-input :value="form.file" disabled />
        </el-form-item>
        <el-form-item label="预览">
          <img :src="'/static/' + form.file" style="max-width:80px;max-height:40px" alt="预览">
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width:100%">
            <el-option value="badge" label="勋章图标" />
            <el-option value="avatar" label="头像" />
            <el-option value="game" label="游戏Logo" />
            <el-option value="priv" label="特权等级" />
            <el-option value="other" label="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="特权等级" v-if="form.category === 'priv'">
          <el-input-number v-model="form.level" :min="1" :max="99" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
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
  name: 'AdminResources',
  data () {
    return { list: [], total: 0, page: 1, size: 10, category: '', word: '', loading: false, dlg: false, syncing: false, form: {} }
  },
  mounted () { this.load() },
  methods: {
    catName (c) {
      return { badge: '勋章图标', avatar: '头像', game: '游戏Logo', priv: '特权等级', other: '其他' }[c] || c
    },
    load () {
      this.loading = true
      api.get('/admin/resources', { params: { page: this.page, size: this.size, category: this.category, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    search () { this.page = 1; this.load() },
    openDlg (row) {
      this.form = { id: row.id, file: row.file, name: row.name, category: row.category, level: row.level, status: row.status }
      this.dlg = true
    },
    save () {
      api.put(`/admin/resources/${this.form.id}`, this.form).then(x => {
        if (x.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(x.msg)
      })
    },
    toggle (row) {
      api.put(`/admin/resources/${row.id}`, { name: row.name, category: row.category, level: row.level, status: row.status === 1 ? 0 : 1 }).then(x => {
        if (x.code === 0) { this.$message.success(row.status === 1 ? '已停用' : '已启用'); this.load() } else this.$message.error(x.msg)
      })
    },
    sync () {
      this.syncing = true
      api.post('/resources/sync').then(r => {
        this.syncing = false
        if (r.code === 0) {
          this.$message.success('同步完成，新增登记 ' + r.data.added + ' 项')
          this.load()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>
