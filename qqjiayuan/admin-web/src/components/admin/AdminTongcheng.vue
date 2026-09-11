<template>
  <div>
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab">
        <!-- 省份与城市 -->
        <el-tab-pane label="省份与城市" name="boards">
          <div class="toolbar">
            <span class="txt-fade">同城客栈（省份→城市带区号）</span>
            <div class="grow" />
            <el-button type="primary" icon="el-icon-plus" @click="openProv(null)">新增省份</el-button>
          </div>
          <el-table :data="provinces" v-loading="loading" stripe highlight-current-row @current-change="selProv">
            <el-table-column prop="name" label="省份" min-width="120" />
            <el-table-column label="城市数" width="90" align="center">
              <template slot-scope="{row}">{{ (row.cities || []).length }}</template>
            </el-table-column>
            <el-table-column prop="sort" label="排序" width="80" align="center" />
            <el-table-column label="状态" width="90" header-align="center">
              <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag></template>
            </el-table-column>
            <el-table-column label="操作" width="200" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain @click="openProv(row)">编辑</el-button>
                <el-button size="mini" type="danger" plain @click="delProv(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="toolbar" style="margin-top:18px" v-if="prov">
            <span class="txt-fade">【{{ prov.name }}】的城市（点击行选择省份）</span>
            <div class="grow" />
            <el-button type="primary" icon="el-icon-plus" @click="openCity(null)">新增城市</el-button>
          </div>
          <el-table :data="cities" v-if="prov" stripe>
            <el-table-column prop="name" label="城市" min-width="110" />
            <el-table-column prop="city_code" label="区号" width="90" align="center" />
            <el-table-column prop="description" label="简介" min-width="160" />
            <el-table-column prop="thread_count2" label="帖子数" width="90" align="center" />
            <el-table-column prop="click" label="人气" width="80" align="center" />
            <el-table-column prop="sort" label="排序" width="70" align="center" />
            <el-table-column label="状态" width="90" header-align="center">
              <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag></template>
            </el-table-column>
            <el-table-column label="操作" width="260" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain @click="openCity(row)">编辑</el-button>
                <el-button size="mini" type="primary" plain @click="openManagers(row)">同城管理</el-button>
                <el-button size="mini" type="danger" plain @click="delCity(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="txt-fade" style="padding:14px 0">点击上方省份行查看其下的城市</div>
        </el-tab-pane>

        <!-- 同城管理（参考 wap_manage） -->
        <el-tab-pane label="同城管理" name="managers">
          <div class="toolbar">
            <el-select v-model="mCityId" placeholder="选择城市" style="width:280px" clearable>
              <el-option v-for="c in allCities" :key="'mc'+c.id" :label="provName(c.parent_id) + ' / ' + c.name" :value="c.id" />
            </el-select>
            <div class="grow" />
            <el-button type="primary" icon="el-icon-plus" :disabled="!mCityId" @click="openAddManager">任命管理</el-button>
          </div>
          <el-table :data="managers" v-loading="mLoading" stripe>
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column label="用户" min-width="160">
              <template slot-scope="{row}">{{ row.user ? row.user.username + ' ' + row.user.nickname : row.user_id }}</template>
            </el-table-column>
            <el-table-column prop="title" label="职务" width="160" />
            <el-table-column prop="sort" label="排序" width="80" align="center" />
            <el-table-column label="操作" width="120" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain @click="delManager(row)">免除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-if="mCityId && !managers.length" class="txt-fade" style="padding:14px 0">该城市暂无同城管理</div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 省份编辑 -->
    <el-dialog :title="provForm.id ? '编辑省份' : '新增省份'" :visible.sync="provDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="70px">
        <el-form-item label="省份"><el-input v-model.trim="provForm.name" maxlength="30" /></el-form-item>
        <el-form-item label="简介"><el-input v-model.trim="provForm.description" maxlength="100" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="provForm.sort" :min="0" /></el-form-item>
        <el-form-item label="显示" v-if="provForm.id"><el-switch v-model="provForm.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="provDlg = false">取 消</el-button><el-button type="primary" @click="saveProv">保 存</el-button></div>
    </el-dialog>

    <!-- 城市编辑 -->
    <el-dialog :title="cityForm.id ? '编辑城市' : '新增城市'" :visible.sync="cityDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="70px">
        <el-form-item label="城市"><el-input v-model.trim="cityForm.name" maxlength="30" /></el-form-item>
        <el-form-item label="区号"><el-input v-model.trim="cityForm.city_code" maxlength="10" placeholder="如 0722" /></el-form-item>
        <el-form-item label="简介"><el-input v-model.trim="cityForm.description" maxlength="100" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="cityForm.sort" :min="0" /></el-form-item>
        <el-form-item label="显示" v-if="cityForm.id"><el-switch v-model="cityForm.status" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="cityDlg = false">取 消</el-button><el-button type="primary" @click="saveCity">保 存</el-button></div>
    </el-dialog>

    <!-- 任命管理 -->
    <el-dialog title="任命同城管理" :visible.sync="mDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="70px">
        <el-form-item label="城市"><el-input :value="mCityName" disabled /></el-form-item>
        <el-form-item label="用户ID"><el-input-number v-model="mForm.user_id" :min="10000" :step="1" style="width:100%" /></el-form-item>
        <el-form-item label="职务"><el-input v-model.trim="mForm.title" maxlength="30" placeholder="如：馆长 / 城市管理员" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="mForm.sort" :min="0" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="mDlg = false">取 消</el-button><el-button type="primary" @click="saveManager">保 存</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminTongcheng',
  data () {
    return {
      tab: 'boards', loading: false, provinces: [], prov: null,
      provDlg: false, provForm: { id: 0, name: '', description: '', sort: 0, status: 1 },
      cityDlg: false, cityForm: { id: 0, name: '', city_code: '', description: '', sort: 0, status: 1 },
      mCityId: 0, managers: [], mLoading: false, mDlg: false, mForm: { user_id: 10000, title: '', sort: 0 }
    }
  },
  computed: {
    cities () { return this.prov ? (this.prov.cities || []) : [] },
    allCities () {
      const arr = []
      this.provinces.forEach(p => (p.cities || []).forEach(c => arr.push(c)))
      return arr
    },
    mCityName () {
      const c = this.allCities.find(x => x.id === this.mCityId)
      return c ? c.name : ''
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/tongcheng').then(r => {
        this.loading = false
        if (r.code === 0) {
          this.provinces = r.data.provinces || []
          if (this.prov) {
            this.prov = this.provinces.find(p => p.id === this.prov.id) || null
          }
          this.loadManagers()
        }
      })
    },
    provName (pid) {
      const p = this.provinces.find(x => x.id === pid)
      return p ? p.name : ''
    },
    selProv (row) { this.prov = row },
    openProv (row) {
      this.provForm = row ? { id: row.id, name: row.name, description: row.description, sort: row.sort, status: row.status } : { id: 0, name: '', description: '', sort: 0, status: 1 }
      this.provDlg = true
    },
    saveProv () {
      if (!this.provForm.name) { this.$message.warning('请填写省份'); return }
      const p = this.provForm
      const req = p.id ? api.put('/admin/tongcheng/provinces/' + p.id, p) : api.post('/admin/tongcheng/provinces', p)
      req.then(r => { if (r.code === 0) { this.provDlg = false; this.load() } else this.$message.error(r.msg) })
    },
    delProv (row) {
      this.$confirm('确定删除省份「' + row.name + '」？需先删除其下城市', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/tongcheng/provinces/' + row.id).then(r => { if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg) })
      }).catch(() => {})
    },
    openCity (row) {
      if (!this.prov) { this.$message.warning('请先选择省份'); return }
      this.cityForm = row ? { id: row.id, name: row.name, city_code: row.city_code, description: row.description, sort: row.sort, status: row.status } : { id: 0, name: '', city_code: '', description: '', sort: 0, status: 1 }
      this.cityDlg = true
    },
    saveCity () {
      if (!this.cityForm.name) { this.$message.warning('请填写城市'); return }
      const f = Object.assign({ province_id: this.prov.id }, this.cityForm)
      const req = f.id ? api.put('/admin/tongcheng/cities/' + f.id, f) : api.post('/admin/tongcheng/cities', f)
      req.then(r => { if (r.code === 0) { this.cityDlg = false; this.load() } else this.$message.error(r.msg) })
    },
    delCity (row) {
      this.$confirm('确定删除城市「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/tongcheng/cities/' + row.id).then(r => { if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg) })
      }).catch(() => {})
    },
    openManagers (row) {
      this.tab = 'managers'
      this.mCityId = row.id
    },
    loadManagers () {
      if (!this.mCityId) { this.managers = []; return }
      this.mLoading = true
      api.get('/admin/tongcheng/managers/' + this.mCityId).then(r => {
        this.mLoading = false
        if (r.code === 0) this.managers = r.data || []
      })
    },
    openAddManager () {
      this.mForm = { user_id: 10000, title: '同城管理员', sort: 0 }
      this.mDlg = true
    },
    saveManager () {
      if (!this.mForm.title) { this.$message.warning('请填写职务名称'); return }
      api.post('/admin/tongcheng/managers', { board_id: this.mCityId, user_id: this.mForm.user_id, title: this.mForm.title, sort: this.mForm.sort }).then(r => {
        if (r.code === 0) { this.mDlg = false; this.loadManagers(); this.load() } else this.$message.error(r.msg)
      })
    },
    delManager (row) {
      api.delete('/admin/tongcheng/managers/' + row.id).then(r => { if (r.code === 0) { this.$message.success('已免除'); this.loadManagers(); this.load() } else this.$message.error(r.msg) })
    }
  },
  watch: {
    mCityId () { this.loadManagers() }
  }
}
</script>
