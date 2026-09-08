<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-tabs v-model="tab" style="width:100%">
          <el-tab-pane label="特权方案" name="plans" />
          <el-tab-pane label="用户特权" name="users" />
        </el-tabs>
        <div class="grow" />
        <el-button v-if="tab === 'plans'" type="success" @click="batch('blue')">一键开通蓝钻</el-button>
        <el-button v-if="tab === 'plans'" type="warning" @click="batch('qq')">一键开通超Q</el-button>
        <el-button v-if="tab === 'plans'" type="primary" icon="el-icon-plus" @click="openDlg(null)">新增方案</el-button>
      </div>

      <!-- 方案管理 -->
      <el-table v-if="tab === 'plans'" :data="plans" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="类型" width="90">
          <template slot-scope="{row}"><el-tag :type="row.type === 'blue' ? 'primary' : 'warning'" size="mini">{{ row.type === 'blue' ? '蓝钻' : '超Q' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="name" label="方案" min-width="180" />
        <el-table-column prop="cost" label="金币" width="70" />
        <el-table-column prop="gain" label="赠送" width="70" />
        <el-table-column prop="speed" label="速度/天" width="80" />
        <el-table-column prop="days" label="天数" width="70" />
        <el-table-column label="限购/库存/销量" min-width="130" header-align="center">
          <template slot-scope="{row}">{{ row.limit || '不限' }}/{{ row.stock || '不限' }}/{{ row.sales }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 用户特权 -->
      <el-table v-else :data="users" v-loading="loading" stripe>
        <el-table-column prop="id" label="号码" width="90" />
        <el-table-column label="昵称" min-width="130">
          <template slot-scope="{row}"><font :color="row.color || '#333'">{{ row.nickname }}</font></template>
        </el-table-column>
        <el-table-column label="蓝钻等级" min-width="180">
          <template slot-scope="{row}">
            <div class="priv-cell">
              <el-tag :type="row.blue_exp > 0 ? 'primary' : 'info'" size="mini">Lv.{{ row.blue_lv }}</el-tag>
              <el-input-number v-model="row.blue_exp" size="mini" :min="0" :max="999999" controls-position="right" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="超Q等级" min-width="180">
          <template slot-scope="{row}">
            <div class="priv-cell">
              <el-tag :type="row.qq_exp > 0 ? 'warning' : 'info'" size="mini">Lv.{{ row.qq_lv }}</el-tag>
              <el-input-number v-model="row.qq_exp" size="mini" :min="0" :max="999999" controls-position="right" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right" header-align="center" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openOne(row, 'blue')">开蓝钻</el-button>
            <el-button size="mini" type="warning" plain @click="openOne(row, 'qq')">开超Q</el-button>
            <el-button size="mini" type="success" plain @click="saveUser(row)">保存</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 用户分页 -->
      <div v-if="tab === 'users'" class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ pageSize }} 条</div>
        <el-pagination
          small
          background
          layout="sizes, prev, pager, next, jumper"
          :total="total"
          :page-size.sync="pageSize"
          :current-page.sync="page"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="page = 1; load()"
          @current-change="load"
        />
      </div>
    </el-card>

    <el-dialog :title="form.id ? '编辑方案' : '新增方案'" :visible.sync="dlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio label="blue">蓝钻</el-radio>
            <el-radio label="qq">超Q</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称"><el-input v-model.trim="form.name" maxlength="40" /></el-form-item>
        <el-form-item label="金币"><el-input-number v-model="form.cost" :min="0" /></el-form-item>
        <el-form-item label="成长赠送"><el-input-number v-model="form.gain" :min="0" /></el-form-item>
        <el-form-item label="成长速度/天"><el-input-number v-model="form.speed" :min="0" /></el-form-item>
        <el-form-item label="天数"><el-input-number v-model="form.days" :min="0" /></el-form-item>
        <el-form-item label="每号限购"><el-input-number v-model="form.limit" :min="0" /></el-form-item>
        <el-form-item label="库存"><el-input-number v-model="form.stock" :min="0" /></el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminPrivileges',
  data () {
    return { tab: 'plans', plans: [], users: [], loading: false, dlg: false, page: 1, pageSize: 10, total: 0, form: { id: 0, type: 'blue', name: '', cost: 0, gain: 0, speed: 0, days: 0, limit: 0, stock: 0 } }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      if (this.tab === 'plans') {
        api.get('/admin/privileges/plans').then(r => {
          this.loading = false
          if (r.code === 0) this.plans = r.data
        })
        return
      }
      api.get('/admin/privileges/users', { params: { page: this.page, size: this.pageSize } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.users = r.data.list; this.total = r.data.total }
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, type: row.type, name: row.name, cost: row.cost, gain: row.gain, speed: row.speed, days: row.days, limit: row.limit, stock: row.stock } : { id: 0, type: 'blue', name: '', cost: 0, gain: 0, speed: 0, days: 0, limit: 0, stock: 0 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写方案名'); return }
      const body = { type: this.form.type, name: this.form.name, cost: this.form.cost, gain: this.form.gain, speed: this.form.speed, days: this.form.days, limit: this.form.limit, stock: this.form.stock }
      if (this.form.id) api.put('/admin/privileges/plans/' + this.form.id, body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      else api.post('/admin/privileges/plans', body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
    },
    del (row) {
      api.delete('/admin/privileges/plans/' + row.id).then(() => this.load())
    },
    batch (type) {
      this.$confirm('确定给所有用户开通' + (type === 'blue' ? '蓝钻' : '超Q') + '吗？', '提示').then(() => {
        api.post('/admin/privileges/batch', { type }).then(r => { if (r.code === 0) this.$message.success('已给所有用户开通'); else this.$message.error(r.msg) })
      }).catch(() => {})
    },
    saveUser (row) {
      api.put('/admin/privileges/users/' + row.id, { blue_exp: row.blue_exp, qq_exp: row.qq_exp }).then(r => { if (r.code === 0) this.$message.success('已保存') })
    },
    openOne (row, type) {
      this.$confirm('确定给用户「' + row.nickname + '」开通' + (type === 'blue' ? '蓝钻' : '超Q') + '吗？', '提示').then(() => {
        api.post('/admin/privileges/users/' + row.id + '/open', { type }).then(r => {
          if (r.code === 0) { this.$message.success('已开通'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  },
  watch: {
    tab () { this.page = 1; this.load() }
  }
}
</script>

<style scoped>
.priv-cell { display: flex; align-items: center; gap: 8px; }
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
</style>
