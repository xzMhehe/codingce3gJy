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
        <el-button type="success" icon="el-icon-upload2" @click="upDlg = true">上传图片</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column label="预览" width="80">
          <template slot-scope="{row}"><img :src="srcOf(row)" class="preview" :alt="row.name"></template>
        </el-table-column>
        <el-table-column label="文件" min-width="200" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.file }}
            <el-tag v-if="row.has_data" size="mini" type="success" style="margin-left:4px">库存</el-tag>
          </template>
        </el-table-column>
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
        <el-table-column label="操作" width="230" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '停用' : '启用' }}</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
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

    <!-- 上传图片（base64 入库） -->
    <el-dialog title="上传图片（存入数据库）" :visible.sync="upDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="名称">
          <el-input v-model.trim="upForm.name" maxlength="30" placeholder="资源名称" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="upForm.category" style="width:100%">
            <el-option value="badge" label="勋章图标" />
            <el-option value="avatar" label="头像" />
            <el-option value="game" label="游戏Logo" />
            <el-option value="priv" label="特权等级" />
            <el-option value="other" label="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="特权等级" v-if="upForm.category === 'priv'">
          <el-input-number v-model="upForm.level" :min="1" :max="99" />
        </el-form-item>
        <el-form-item label="图片">
          <input type="file" accept="image/*" @change="onFile" />
          <div v-if="upForm.preview" style="margin-top:8px">
            <img :src="upForm.preview" style="max-width:120px;max-height:60px" alt="预览">
            <div class="help-line">{{ upForm.fileName }}（{{ upForm.size }}）</div>
          </div>
        </el-form-item>
      </el-form>
      <div class="help-line" style="margin:0 0 10px 90px">支持 JPG/PNG/GIF/WEBP/BMP，1MB 以内；上传后以 base64 存入数据库。</div>
      <div slot="footer">
        <el-button @click="upDlg = false">取 消</el-button>
        <el-button type="primary" :loading="uploading" @click="doUpload">上传</el-button>
      </div>
    </el-dialog>

    <!-- 编辑 模态框 -->
    <el-dialog title="编辑资源" :visible.sync="dlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="文件">
          <el-input :value="form.file" disabled />
        </el-form-item>
        <el-form-item label="预览">
          <img :src="srcOf(form)" style="max-width:80px;max-height:40px" alt="预览">
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
    return {
      list: [], total: 0, page: 1, size: 10, category: '', word: '', loading: false,
      dlg: false, syncing: false, form: {},
      upDlg: false, uploading: false,
      upForm: { name: '', category: 'other', level: 1, data: '', preview: '', fileName: '', size: '' }
    }
  },
  mounted () { this.load() },
  methods: {
    catName (c) {
      return { badge: '勋章图标', avatar: '头像', game: '游戏Logo', priv: '特权等级', other: '其他' }[c] || c
    },
    srcOf (row) {
      if (!row || !row.file) return ''
      return row.file.indexOf('db/') === 0 ? '/api/res/' + row.file : '/static/' + row.file
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
      this.form = { id: row.id, file: row.file, name: row.name, category: row.category, level: row.level, status: row.status, has_data: row.has_data }
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
    del (row) {
      this.$confirm(`删除资源「${row.name || row.file}」？${row.has_data ? '库存图片将一并删除。' : ''}`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/resources/' + row.id).then(x => {
          if (x.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(x.msg)
        })
      }).catch(() => {})
    },
    onFile (e) {
      const f = e.target.files && e.target.files[0]
      if (!f) return
      if (f.size > 1024 * 1024) { this.$message.error('图片不能超过 1MB'); return }
      const reader = new FileReader()
      reader.onload = () => {
        this.upForm.data = reader.result
        this.upForm.preview = reader.result
        this.upForm.fileName = f.name
        this.upForm.size = (f.size / 1024).toFixed(1) + 'KB'
        if (!this.upForm.name) this.upForm.name = f.name.replace(/\.[^.]+$/, '').slice(0, 30)
      }
      reader.readAsDataURL(f)
    },
    doUpload () {
      if (!this.upForm.name || !this.upForm.data) { this.$message.warning('请填写名称并选择图片'); return }
      this.uploading = true
      api.post('/admin/resources/upload', {
        name: this.upForm.name, category: this.upForm.category, level: this.upForm.level, data: this.upForm.data
      }).then(r => {
        this.uploading = false
        if (r.code === 0) {
          this.$message.success('上传成功：' + r.data.file)
          this.upDlg = false
          this.upForm = { name: '', category: 'other', level: 1, data: '', preview: '', fileName: '', size: '' }
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.uploading = false; this.$message.error('上传失败') })
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