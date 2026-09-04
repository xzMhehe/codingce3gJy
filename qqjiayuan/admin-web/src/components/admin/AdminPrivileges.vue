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
        <el-table-column prop="cost" label="金币" width="80" />
        <el-table-column prop="gain" label="成长" width="80" />
        <el-table-column prop="days" label="天数" width="80" />
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
        <el-table-column label="蓝钻等级" width="110">
          <template slot-scope="{row}"><el-input-number v-model="row.blue_exp" size="mini" :min="0" :max="999999" /></template>
        </el-table-column>
        <el-table-column label="超Q等级" width="110">
          <template slot-scope="{row}"><el-input-number v-model="row.qq_exp" size="mini" :min="0" :max="999999" /></template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right" header-align="center">
          <template slot-scope="{row}"><el-button size="mini" type="primary" plain @click="saveUser(row)">保存</el-button></template>
        </el-table-column>
      </el-table>
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
        <el-form-item label="成长"><el-input-number v-model="form.gain" :min="0" /></el-form-item>
        <el-form-item label="天数"><el-input-number v-model="form.days" :min="0" /></el-form-item>
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
    return { tab: 'plans', plans: [], users: [], loading: false, dlg: false, form: { id: 0, type: 'blue', name: '', cost: 0, gain: 0, days: 0 } }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      const url = this.tab === 'plans' ? '/admin/privileges/plans' : '/admin/privileges/users'
      api.get(url).then(r => {
        this.loading = false
        if (r.code === 0) { if (this.tab === 'plans') this.plans = r.data; else this.users = r.data.list }
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, type: row.type, name: row.name, cost: row.cost, gain: row.gain, days: row.days } : { id: 0, type: 'blue', name: '', cost: 0, gain: 0, days: 0 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写方案名'); return }
      const body = { type: this.form.type, name: this.form.name, cost: this.form.cost, gain: this.form.gain, days: this.form.days }
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
    }
  },
  watch: { tab () { this.load() } }
}
</script>
