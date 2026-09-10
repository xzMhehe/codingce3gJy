<template>
  <div class="farm-admin">
    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card s-green">
        <div class="stat-ico el-icon-suitcase" />
        <div class="stat-info">
          <div class="stat-num">{{ list.length }}</div>
          <div class="stat-lab">种子总数</div>
        </div>
      </div>
      <div class="stat-card s-blue">
        <div class="stat-ico el-icon-time" />
        <div class="stat-info">
          <div class="stat-num">{{ countMulti() }}</div>
          <div class="stat-lab">多季作物</div>
        </div>
      </div>
      <div class="stat-card s-purple">
        <div class="stat-ico el-icon-medal" />
        <div class="stat-info">
          <div class="stat-num">{{ maxLevel() }}</div>
          <div class="stat-lab">最高种植等级</div>
        </div>
      </div>
      <div class="stat-card s-orange">
        <div class="stat-ico el-icon-coin" />
        <div class="stat-info">
          <div class="stat-num">{{ maxSeedPrice() }}</div>
          <div class="stat-lab">最贵种子(G币)</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索种子名称" clearable style="width:230px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增种子</el-button>
      </div>

      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="名称" min-width="100">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="种植等级" width="90" align="center">
          <template slot-scope="{row}"><i class="el-icon-medal td-gold"></i>{{ row.level }}级</template>
        </el-table-column>
        <el-table-column label="种子价(G)" width="100" align="center">
          <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.seed_price }}</template>
        </el-table-column>
        <el-table-column label="果实单价(G)" width="105" align="center">
          <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.price }}</template>
        </el-table-column>
        <el-table-column label="收获季数" width="85" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.cycle > 1 ? 'warning' : 'info'" size="mini">{{ row.cycle }}季</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="成熟时间" width="120" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ txtMin(row.aging) }}</span></template>
        </el-table-column>
        <el-table-column label="再熟时间" width="120" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.again ? txtMin(row.again) : '—' }}</span></template>
        </el-table-column>
        <el-table-column label="每季产量" width="85" align="center">
          <template slot-scope="{row}"><span class="td-blue">{{ row.yield }}</span></template>
        </el-table-column>
        <el-table-column label="每季经验" width="85" align="center">
          <template slot-scope="{row}"><span class="td-blue">{{ row.point }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg(row)" />
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ filtered.length }}</b> 条 · 每页 {{ pageSize }} 条</div>
        <el-pagination small background layout="sizes, prev, pager, next, jumper"
          :total="filtered.length" :page-size.sync="pageSize" :current-page.sync="page"
          :page-sizes="[5, 10, 20, 50]" @size-change="page = 1" />
      </div>
    </el-card>

    <el-dialog :title="form.id ? '编辑种子' : '新增种子'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="20" />
        </el-form-item>
        <el-form-item label="种植等级">
          <el-input-number v-model.number="form.level" :min="1" :max="60" />
        </el-form-item>
        <el-form-item label="果实单价(G)">
          <el-input-number v-model.number="form.price" :min="1" />
          <span class="help-line" style="margin-left:10px">种子价 = 单价 × 5 × 季数 = {{ form.price * 5 * Math.max(form.cycle, 1) }} G币</span>
        </el-form-item>
        <el-form-item label="收获季数">
          <el-input-number v-model.number="form.cycle" :min="1" :max="9" />
          <span class="help-line" style="margin-left:10px">大于 1 为多季作物，收获后自动进入下一季</span>
        </el-form-item>
        <el-form-item label="成熟分钟">
          <el-input-number v-model.number="form.aging" :min="1" />
          <template v-if="form.cycle > 1">
            <span style="margin: 0 8px">再熟</span>
            <el-input-number v-model.number="form.again" :min="1" />
          </template>
        </el-form-item>
        <el-form-item label="每季产量">
          <el-input-number v-model.number="form.yield" :min="1" :max="99" />
        </el-form-item>
        <el-form-item label="每季经验">
          <el-input-number v-model.number="form.point" :min="0" :max="999" />
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
  name: 'AdminFarmSeeds',
  data () {
    return {
      list: [], loading: false, dlg: false, saving: false,
      kw: '', page: 1, pageSize: 10,
      form: { id: 0, name: '', level: 1, price: 2, cycle: 1, aging: 15, again: 0, yield: 8, point: 4 }
    }
  },
  computed: {
    filtered () {
      const k = this.kw.trim().toLowerCase()
      return this.list.filter(x => !k || (x.name || '').toLowerCase().includes(k))
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) }
  },
  mounted () { this.load() },
  methods: {
    countMulti () { return this.list.filter(x => x.cycle > 1).length },
    maxLevel () { return this.list.reduce((m, x) => Math.max(m, x.level || 0), 0) },
    maxSeedPrice () { return this.list.reduce((m, x) => Math.max(m, x.seed_price || 0), 0) },
    txtMin (min) {
      if (!min) return '0分钟'
      if (min < 60) return min + '分钟'
      return Math.floor(min / 60) + '小时' + (min % 60) + '分钟'
    },
    load () {
      this.loading = true
      api.get('/admin/farm-seeds').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
      })
    },
    openDlg (row) {
      this.form = row ? { ...row } : { id: 0, name: '', level: 1, price: 2, cycle: 1, aging: 15, again: 0, yield: 8, point: 4 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写种子名称'); return }
      this.saving = true
      const body = { name: this.form.name, level: this.form.level, price: this.form.price,
        cycle: this.form.cycle, aging: this.form.aging, again: this.form.cycle > 1 ? this.form.again : 0,
        yield: this.form.yield, point: this.form.point }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      const call = this.form.id
        ? api.put('/admin/farm-seeds/' + this.form.id, body)
        : api.post('/admin/farm-seeds', body)
      call.then(r => { if (r.code === 0) done(); else { this.saving = false; this.$message.error(r.msg) } })
    },
    del (row) {
      this.$confirm('删除种子将同时清理玩家背包中的该种子，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/farm-seeds/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
