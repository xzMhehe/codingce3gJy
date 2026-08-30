<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增游戏</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="Logo" width="80">
          <template slot-scope="{row}">
            <img v-if="row.logo" :src="'/static/image/' + row.logo" class="logo-prev" :alt="row.name">
            <el-tag v-else size="mini" type="info">文字标</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="名称" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.name }}
            <el-tag v-if="row.url" size="mini" type="success" style="margin-left:4px">有官网</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="分类" width="100">
          <template slot-scope="{row}">{{ row.category === 'net' ? '网络游戏' : '社区游戏' }}</template>
        </el-table-column>
        <el-table-column prop="stars" label="星级" width="110" />
        <el-table-column label="网址" min-width="170" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.url || '—' }}</template>
        </el-table-column>
        <el-table-column label="论坛板块" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ boardName(row.board_id) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 新增 / 编辑游戏 模态框 -->
    <el-dialog :title="form.id ? '编辑游戏：' + form.name : '新增游戏'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="分类">
          <el-radio-group v-model="form.category">
            <el-radio label="net">网络游戏</el-radio>
            <el-radio label="com">社区游戏</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Logo">
          <el-select v-model="form.logo" placeholder="（文字标）" clearable style="width:100%">
            <el-option v-for="l in logos" :key="l" :value="l" :label="l">
              <img class="bicon" :src="'/static/image/' + l" :alt="l">{{ l }}
            </el-option>
          </el-select>
          <img v-if="form.logo" :src="'/static/image/' + form.logo" class="logo-prev" alt="预览" style="margin-top:4px">
        </el-form-item>
        <el-form-item label="网址">
          <el-input v-model.trim="form.url" maxlength="200" placeholder="游戏官网/入口地址，未开发留空" />
        </el-form-item>
        <el-form-item label="星级">
          <el-rate v-model="starsNum" style="margin-top:6px" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model.trim="form.desc" maxlength="100" />
        </el-form-item>
        <el-form-item label="论坛板块">
          <el-select v-model.number="form.board_id" placeholder="不绑定" clearable style="width:100%">
            <el-option-group v-for="ch in channels" :key="ch.id" :label="ch.name">
              <el-option v-for="b in ch.children" :key="b.id" :value="b.id" :label="b.name" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="上架">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
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
  name: 'AdminGames',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      channels: [], logos: [], dlg: false,
      starsNum: 3,
      form: this.blank()
    }
  },
  mounted () {
    this.load()
    api.get('/badge-presets').then(r => { if (r.code === 0) this.logos = r.data.game_logos })
    api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
  },
  methods: {
    blank () {
      return { id: 0, name: '', category: 'com', logo: '', stars: '★★★', desc: '', url: '', board_id: 0, sort: 0, status: 1 }
    },
    resetForm () { this.form = this.blank(); this.starsNum = 3; this.err = ''; this.msg = '' },
    load () {
      this.loading = true
      api.get('/admin/games', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    boardName (id) {
      for (const ch of this.channels) {
        for (const b of ch.children || []) {
          if (b.id === id) return b.name
        }
      }
      return id ? '板块#' + id : '未绑定'
    },
    openDlg (row) {
      if (row) {
        this.form = {
          id: row.id, name: row.name, category: row.category, logo: row.logo,
          stars: row.stars, desc: row.desc, url: row.url || '',
          board_id: row.board_id, sort: row.sort, status: row.status
        }
        this.starsNum = (row.stars || '').split('').filter(c => c === '★').length
      } else {
        this.form = this.blank()
        this.starsNum = 3
      }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.error('请填写游戏名'); return }
      this.form.stars = '★'.repeat(this.starsNum) + '☆'.repeat(5 - this.starsNum > 0 ? 5 - this.starsNum : 0)
      const payload = {
        name: this.form.name, category: this.form.category, logo: this.form.logo,
        stars: this.form.stars, desc: this.form.desc, url: this.form.url,
        board_id: this.form.board_id, sort: this.form.sort, status: this.form.status
      }
      const call = this.form.id ? api.put('/admin/games/' + this.form.id, payload) : api.post('/admin/games', payload)
      call.then(r => {
        if (r.code === 0) {
          this.$message.success(this.form.id ? '已保存' : `已新增「${this.form.name}」`)
          this.resetForm()
          this.dlg = false
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除游戏「${row.name}」吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/games/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
