<template>
  <div class="farm-admin">
    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card s-green">
        <div class="stat-ico el-icon-magic-stick" />
        <div class="stat-info">
          <div class="stat-num">{{ list.length }}</div>
          <div class="stat-lab">当前页技能</div>
        </div>
      </div>
      <div class="stat-card s-blue">
        <div class="stat-ico el-icon-s-flag" />
        <div class="stat-info">
          <div class="stat-num">{{ countAct(1) }}</div>
          <div class="stat-lab">主动技能</div>
        </div>
      </div>
      <div class="stat-card s-purple">
        <div class="stat-ico el-icon-s-check" />
        <div class="stat-info">
          <div class="stat-num">{{ countAct(0) }}</div>
          <div class="stat-lab">被动技能</div>
        </div>
      </div>
      <div class="stat-card s-orange">
        <div class="stat-ico el-icon-coin" />
        <div class="stat-info">
          <div class="stat-num">{{ maxCoef() }}</div>
          <div class="stat-lab">最高伤害系数</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="actFilter" placeholder="全部类型" clearable style="width:130px" @change="page = 1; load()">
          <el-option label="主动技能" :value="1" />
          <el-option label="被动技能" :value="0" />
        </el-select>
        <el-input v-model="word" placeholder="搜索技能名" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增技能</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="名称" min-width="120">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.act === 1 ? 'danger' : 'success'" size="mini">{{ row.act === 1 ? '主动' : '被动' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="level" label="等级要求" width="90" align="center" />
        <el-table-column label="价格" width="110" align="center">
          <template slot-scope="{row}">
            <i :class="row.currency === 'yuanbao' ? 'el-icon-goods td-gold' : 'el-icon-coin td-blue'"></i>{{ row.price }}
            <span class="td-sub">/{{ currencyNames[row.currency] || row.currency }}</span>
          </template>
        </el-table-column>
        <el-table-column label="武器限制" width="90" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ row.weapon_req || '无限制' }}</span></template>
        </el-table-column>
        <el-table-column label="系数" width="80" align="center">
          <template slot-scope="{row}"><span v-if="row.act === 1" class="td-mono">{{ row.coef }}</span><span v-else class="td-muted">—</span></template>
        </el-table-column>
        <el-table-column label="命中/暴击" width="100" align="center">
          <template slot-scope="{row}">
            <span v-if="row.act === 1" class="td-mono">{{ row.hit }}% / {{ row.crit }}%</span>
            <span v-else class="td-muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="被动加成" min-width="150">
          <template slot-scope="{row}">
            <span v-if="row.act === 0">
              <span v-if="row.p_atk" class="td-mono">攻{{ row.p_atk }}</span>
              <span v-if="row.p_def" class="td-mono"> 防{{ row.p_def }}</span>
              <span v-if="row.p_hp" class="td-mono"> 血{{ row.p_hp }}</span>
              <span v-if="row.p_mp" class="td-mono"> 力{{ row.p_mp }}</span>
              <span v-if="row.crit" class="td-mono"> 暴{{ row.crit }}</span>
              <span v-if="!row.p_atk && !row.p_def && !row.p_hp && !row.p_mp && !row.crit" class="td-muted">—</span>
            </span>
            <span v-else class="td-muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg(row)" />
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <el-dialog :title="form.id ? '编辑技能' : '新增技能'" :visible.sync="dlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <div class="form-grid">
          <el-form-item label="名称" required>
            <el-input v-model.trim="form.name" maxlength="30" />
          </el-form-item>
          <el-form-item label="类型">
            <el-radio-group v-model.number="form.act">
              <el-radio :label="1">主动</el-radio>
              <el-radio :label="0">被动</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="等级要求">
            <el-input-number v-model.number="form.level" :min="1" />
          </el-form-item>
          <el-form-item label="价格">
            <el-input-number v-model.number="form.price" :min="0" />
          </el-form-item>
          <el-form-item label="货币">
            <el-select v-model="form.currency" style="width:150px">
              <el-option v-for="(n, c) in currencyNames" :key="c" :label="n" :value="c" />
            </el-select>
          </el-form-item>
          <el-form-item label="武器限制">
            <el-input v-model="form.weapon_req" maxlength="20" placeholder="无限制" />
          </el-form-item>
        </div>
        <div v-if="form.act === 1" class="sub-title">主动技能参数</div>
        <div v-if="form.act === 1" class="form-grid">
          <el-form-item label="伤害系数"><el-input-number v-model.number="form.coef" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="命中(%)"><el-input-number v-model.number="form.hit" :min="0" :max="100" controls-position="right" /></el-form-item>
          <el-form-item label="暴击(%)"><el-input-number v-model.number="form.crit" :min="0" :max="100" controls-position="right" /></el-form-item>
          <el-form-item label="暴击倍数(%)"><el-input-number v-model.number="form.crit_mul" :min="100" controls-position="right" /></el-form-item>
          <el-form-item label="附加闪避"><el-input-number v-model.number="form.dodge_add" :min="0" controls-position="right" /></el-form-item>
        </div>
        <div v-else class="sub-title">被动技能加成</div>
        <div v-else class="form-grid">
          <el-form-item label="攻击"><el-input-number v-model.number="form.p_atk" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="防御"><el-input-number v-model.number="form.p_def" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="气血"><el-input-number v-model.number="form.p_hp" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="气力"><el-input-number v-model.number="form.p_mp" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="暴击(%)"><el-input-number v-model.number="form.crit" :min="0" :max="100" controls-position="right" /></el-form-item>
        </div>
        <el-form-item label="描述">
          <el-input v-model="form.desc" type="textarea" :rows="2" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model.number="form.status">
            <el-radio :label="1">上架</el-radio>
            <el-radio :label="0">下架</el-radio>
          </el-radio-group>
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
  name: 'AdminJwtSkills',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', actFilter: null,
      dlg: false, saving: false, form: {},
      currencyNames: { coins: 'G币', yuanbao: '元宝' },
      formDef: {
        id: 0, name: '', act: 1, level: 1, price: 0, currency: 'coins', weapon_req: '无限制',
        coef: 100, hit: 90, crit: 5, crit_mul: 150, dodge_add: 0,
        p_atk: 0, p_def: 0, p_hp: 0, p_mp: 0, status: 1, desc: ''
      }
    }
  },
  mounted () { this.load() },
  methods: {
    countAct (a) { return this.list.filter(x => x.act === a).length },
    maxCoef () { return this.list.reduce((m, x) => Math.max(m, x.act === 1 ? x.coef : 0), 0) },
    load () {
      this.loading = true
      api.get('/admin/jwt-skills', { params: { page: this.page, size: this.size, word: this.word, act: this.actFilter } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      this.form = row ? { ...row } : { ...this.formDef }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写技能名'); return }
      this.saving = true
      const body = { ...this.form }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      const call = this.form.id
        ? api.put('/admin/jwt-skills/' + this.form.id, body)
        : api.post('/admin/jwt-skills', body)
      call.then(r => { if (r.code === 0) done(); else { this.saving = false; this.$message.error(r.msg) } })
    },
    del (row) {
      this.$confirm('删除技能将同时清理玩家已学记录，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/jwt-skills/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.form-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0 14px; }
.form-grid .el-input-number { width: 100%; }
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 12px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
