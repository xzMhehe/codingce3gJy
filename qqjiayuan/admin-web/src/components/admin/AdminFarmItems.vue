<template>
  <div class="farm-admin">
    <el-tabs v-model="tab" type="card">
      <!-- ============ 化肥管理 ============ -->
      <el-tab-pane label="化肥管理" name="muck">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <div class="grow" />
            <el-button type="primary" icon="el-icon-plus" @click="openDlg('muck', null)">新增化肥</el-button>
          </div>
          <el-table :data="mucks" v-loading="loading" stripe border size="medium">
            <el-table-column prop="id" label="ID" width="64" align="center" />
            <el-table-column label="名称" min-width="120">
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="效果" width="180" align="center">
              <template slot-scope="{row}"><span class="td-mono">成熟提前 {{ txtMin(row.speed) }}</span></template>
            </el-table-column>
            <el-table-column label="价格(G)" width="110" align="center">
              <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.price }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg('muck', row)" />
                <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del('muck', row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <!-- ============ 陷阱管理 ============ -->
      <el-tab-pane label="陷阱管理" name="trap">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <div class="grow" />
            <el-button type="primary" icon="el-icon-plus" @click="openDlg('trap', null)">新增陷阱</el-button>
          </div>
          <el-table :data="traps" v-loading="loading" stripe border size="medium">
            <el-table-column prop="id" label="ID" width="64" align="center" />
            <el-table-column label="名称" min-width="120">
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="触发几率" width="180" align="center">
              <template slot-scope="{row}">
                <el-progress :percentage="row.rate" :stroke-width="12" :format="p => p + '%'" style="width:140px" />
              </template>
            </el-table-column>
            <el-table-column label="价格(G)" width="110" align="center">
              <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.price }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg('trap', row)" />
                <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del('trap', row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog :title="dlgTitle" :visible.sync="dlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="20" />
        </el-form-item>
        <el-form-item v-if="tabType === 'muck'" label="提前成熟">
          <el-input-number v-model.number="form.speed" :min="1" :max="720" />
          <span class="help-line" style="margin-left:10px">分钟</span>
        </el-form-item>
        <el-form-item v-else label="触发几率">
          <el-input-number v-model.number="form.rate" :min="1" :max="100" />
          <span class="help-line" style="margin-left:10px">%</span>
        </el-form-item>
        <el-form-item label="价格(G)">
          <el-input-number v-model.number="form.price" :min="0" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminFarmItems',
  data () {
    return {
      tab: 'muck', tabType: 'muck',
      mucks: [], traps: [], loading: false,
      dlg: false, saving: false,
      form: { id: 0, name: '', speed: 10, rate: 30, price: 100 }
    }
  },
  computed: {
    dlgTitle () { return (this.form.id ? '编辑' : '新增') + (this.tabType === 'muck' ? '化肥' : '陷阱') }
  },
  mounted () { this.load() },
  methods: {
    txtMin (min) {
      if (!min) return '0分钟'
      if (min < 60) return min + '分钟'
      return Math.floor(min / 60) + '小时' + (min % 60) + '分钟'
    },
    load () {
      this.loading = true
      Promise.all([api.get('/admin/farm-mucks'), api.get('/admin/farm-traps')]).then(rs => {
        this.loading = false
        if (rs[0].code === 0) this.mucks = rs[0].data
        if (rs[1].code === 0) this.traps = rs[1].data
      })
    },
    openDlg (ty, row) {
      this.tabType = ty
      this.form = row ? { ...row } : { id: 0, name: '', speed: 10, rate: 30, price: 100 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写名称'); return }
      this.saving = true
      const base = '/admin/farm-' + (this.tabType === 'muck' ? 'mucks' : 'traps')
      const body = this.tabType === 'muck'
        ? { name: this.form.name, speed: this.form.speed, price: this.form.price }
        : { name: this.form.name, rate: this.form.rate, price: this.form.price }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      const call = this.form.id ? api.put(base + '/' + this.form.id, body) : api.post(base, body)
      call.then(r => { if (r.code === 0) done(); else { this.saving = false; this.$message.error(r.msg) } })
    },
    del (ty, row) {
      this.$confirm('删除将同时清理玩家背包中的该道具，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/farm-' + (ty === 'muck' ? 'mucks' : 'traps') + '/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
