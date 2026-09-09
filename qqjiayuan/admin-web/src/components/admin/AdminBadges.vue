<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-tabs v-model="tab" style="width:100%">
          <el-tab-pane label="勋章商店" name="shop" />
          <el-tab-pane label="会员勋章" name="medals" />
        </el-tabs>
        <div class="grow" />
        <el-button v-if="tab === 'shop'" type="primary" icon="el-icon-plus" @click="openDlg(null)">新增勋章</el-button>
        <el-button v-if="tab === 'medals'" type="primary" icon="el-icon-plus" @click="grantOpen = true">授予勋章</el-button>
      </div>

      <!-- 勋章商店 -->
      <el-table v-if="tab === 'shop'" :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="图标" width="80">
          <template slot-scope="{row}"><img class="bicon" :src="'/static/picture/' + row.icon" :alt="row.name"></template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="110" />
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column prop="price" label="价格" width="70" />
        <el-table-column label="有效期" width="90">
          <template slot-scope="{row}">{{ row.period ? row.period + '天' : '永久' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="70">
          <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="remark" label="说明" min-width="150" show-overflow-tooltip />
        <el-table-column label="操作" width="150" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 会员勋章 -->
      <template v-else>
        <div class="toolbar" style="margin-bottom:12px">
          <el-input v-model="medalUid" placeholder="会员号码" style="width:130px" />
          <el-button size="small" icon="el-icon-search" @click="loadMedals">搜索</el-button>
          <span class="help-line" style="margin-left:8px">过期勋章已自动清除</span>
        </div>
        <el-table :data="medals" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column label="会员" min-width="120">
            <template slot-scope="{row}"><font :color="row.color || '#333'">{{ row.nickname }}</font></template>
          </el-table-column>
          <el-table-column label="勋章" min-width="150">
            <template slot-scope="{row}">
              <img class="bicon" v-if="row.badge_icon" :src="'/static/picture/' + row.badge_icon" :alt="row.badge_name">{{ row.badge_name }}
            </template>
          </el-table-column>
          <el-table-column prop="sort" label="排序" width="70" />
          <el-table-column label="授予时间" min-width="140">
            <template slot-scope="{row}">{{ fmt(row.granted_at) }}</template>
          </el-table-column>
          <el-table-column label="到期时间" min-width="140">
            <template slot-scope="{row}">
              <el-tag v-if="row.expire_at" type="warning" size="mini">{{ fmt(row.expire_at) }}</el-tag>
              <span v-else class="txt-fade">永久</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="revoke(row)">回收</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <el-pagination v-if="tab === 'shop'" background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 勋章商店：新增/编辑 模态框 -->
    <el-dialog :title="form.id ? '编辑勋章：' + form.name : '新增勋章'" :visible.sync="dlg" width="500px" :close-on-click-modal="false">
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
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="价格"><el-input-number v-model="form.price" :min="0" /></el-form-item>
        <el-form-item label="有效期">
          <el-input-number v-model="form.period" :min="0" />
          <span class="help-line" style="margin-left:6px">天（0=永久）</span>
        </el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="下架" /></el-form-item>
        <el-form-item label="说明">
          <el-input v-model.trim="form.remark" maxlength="100" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </div>
    </el-dialog>

    <!-- 会员勋章：授予 模态框 -->
    <el-dialog title="授予勋章" :visible.sync="grantOpen" width="460px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="会员号码"><el-input v-model.number="grantForm.user_id" placeholder="输入家园号码" /></el-form-item>
        <el-form-item label="勋章">
          <el-select v-model.number="grantForm.badge_id" placeholder="选择勋章" style="width:100%">
            <el-option v-for="b in list" :key="b.id" :value="b.id" :label="b.name">
              <img class="bicon" v-if="b.icon" :src="'/static/picture/' + b.icon" :alt="b.name">{{ b.name }}
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="有效天数"><el-input-number v-model.number="grantForm.days" :min="0" /> <span class="help-line">天（0=永久，默认按勋章有效期）</span></el-form-item>
        <el-form-item label="排序"><el-input-number v-model.number="grantForm.sort" :min="0" /></el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="grantOpen = false">取 消</el-button>
        <el-button type="primary" @click="grantOne">确 定</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminBadges',
  data () {
    return {
      tab: 'shop',
      list: [], total: 0, page: 1, size: 10, loading: false, icons: [], dlg: false,
      medals: [], medalUid: '', grantOpen: false,
      form: { id: 0, name: '', icon: '', remark: '', sort: 0, price: 0, period: 0, status: 1 },
      grantForm: { user_id: null, badge_id: null, days: 0, sort: 0 }
    }
  },
  mounted () {
    this.load()
    api.get('/badge-presets').then(r => {
      if (r.code === 0) {
        this.icons = r.data.badge_icons
        if (!this.form.icon && this.icons.length) this.form.icon = this.icons[0]
      }
    })
    this.loadMedals()
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
    loadMedals () {
      api.get('/admin/user-badges', { params: { uid: this.medalUid } }).then(r => {
        if (r.code === 0) this.medals = r.data || []
      })
    },
    openDlg (row) {
      if (row) this.form = { id: row.id, name: row.name, icon: row.icon, remark: row.remark, sort: row.sort, price: row.price, period: row.period, status: row.status }
      else this.form = { id: 0, name: '', icon: this.icons[0] || '', remark: '', sort: 0, price: 0, period: 0, status: 1 }
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
      this.$confirm(`确定删除勋章「${row.name}」吗？将同时从所有用户身上摘下。`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/badges/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    grantOne () {
      if (!this.grantForm.user_id || !this.grantForm.badge_id) { this.$message.error('请填写会员号码并选择勋章'); return }
      api.post('/admin/user-badges/grant', this.grantForm).then(r => {
        if (r.code === 0) {
          this.$message.success('已授予「' + r.data.name + '」' + (r.data.expire_at ? '（' + this.fmt(r.data.expire_at) + '到期）' : '（永久）'))
          this.grantOpen = false
          this.grantForm = { user_id: null, badge_id: null, days: 0, sort: 0 }
          this.loadMedals()
        } else this.$message.error(r.msg)
      })
    },
    revoke (row) {
      this.$confirm(`确定回收「${row.nickname}」的「${row.badge_name}」勋章吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/user-badges/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已回收'); this.loadMedals() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) {
      if (!t) return '—'
      const d = new Date(t)
      return d.getFullYear() + '-' + (d.getMonth() + 1 < 10 ? '0' : '') + (d.getMonth() + 1) + '-' + (d.getDate() < 10 ? '0' : '') + d.getDate() + ' ' + (d.getHours() < 10 ? '0' : '') + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  },
  watch: {
    tab (v) { if (v === 'medals') this.loadMedals() }
  }
}
</script>