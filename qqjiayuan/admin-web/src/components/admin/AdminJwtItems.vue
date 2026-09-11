<template>
  <div class="farm-admin">
    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card s-green">
        <div class="stat-ico el-icon-goods" />
        <div class="stat-info">
          <div class="stat-num">{{ list.length }}</div>
          <div class="stat-lab">当前页道具</div>
        </div>
      </div>
      <div class="stat-card s-blue">
        <div class="stat-ico el-icon-sell" />
        <div class="stat-info">
          <div class="stat-num">{{ countSrc('shop') }}</div>
          <div class="stat-lab">商店在售</div>
        </div>
      </div>
      <div class="stat-card s-purple">
        <div class="stat-ico el-icon-magic-stick" />
        <div class="stat-info">
          <div class="stat-num">{{ countSrc('forge') }}</div>
          <div class="stat-lab">锻造产物</div>
        </div>
      </div>
      <div class="stat-card s-orange">
        <div class="stat-ico el-icon-coin" />
        <div class="stat-info">
          <div class="stat-num">{{ maxPrice() }}</div>
          <div class="stat-lab">本页最高价(G币)</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="catFilter" placeholder="全部分类" clearable style="width:130px" @change="page = 1; load()">
          <el-option v-for="(n, c) in catNames" :key="c" :label="n" :value="c" />
        </el-select>
        <el-select v-model="srcFilter" placeholder="全部来源" clearable style="width:120px" @change="page = 1; load()">
          <el-option v-for="(n, s) in srcNames" :key="s" :label="n" :value="s" />
        </el-select>
        <el-input v-model="word" placeholder="搜索道具名" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增道具</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="名称" min-width="120">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="分类" width="90" align="center">
          <template slot-scope="{row}"><el-tag size="mini" :type="catTag(row.cat)">{{ catNames[row.cat] || row.cat }}</el-tag></template>
        </el-table-column>
        <el-table-column label="来源" width="90" align="center">
          <template slot-scope="{row}"><el-tag size="mini" :type="srcTag(row.src)">{{ srcNames[row.src] || row.src }}</el-tag></template>
        </el-table-column>
        <el-table-column label="价格" width="110" align="center">
          <template slot-scope="{row}">
            <i :class="row.currency === 'yuanbao' ? 'el-icon-goods td-gold' : 'el-icon-coin td-blue'"></i>{{ row.price }}
            <span class="td-sub">/{{ currencyNames[row.currency] || row.currency }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="level" label="等级" width="70" align="center" />
        <el-table-column label="加成" min-width="150">
          <template slot-scope="{row}">
            <span v-if="row.atk" class="td-mono">攻{{ row.atk }}</span>
            <span v-if="row.def" class="td-mono"> 防{{ row.def }}</span>
            <span v-if="row.hp" class="td-mono"> 血{{ row.hp }}</span>
            <span v-if="row.mp" class="td-mono"> 力{{ row.mp }}</span>
            <span v-if="row.spd" class="td-mono"> 速{{ row.spd }}</span>
            <span v-if="row.hit" class="td-mono"> 命中{{ row.hit }}</span>
            <span v-if="row.crit" class="td-mono"> 暴{{ row.crit }}</span>
            <span v-if="row.dodge" class="td-mono"> 闪{{ row.dodge }}</span>
            <span v-if="!row.atk && !row.def && !row.hp && !row.mp && !row.spd && !row.hit && !row.crit && !row.dodge" class="td-muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="恢复" width="110" align="center">
          <template slot-scope="{row}">
            <span v-if="row.recover_hp" class="td-green">血{{ row.recover_hp }}</span>
            <span v-if="row.recover_mp" class="td-green"> 力{{ row.recover_mp }}</span>
            <span v-if="!row.recover_hp && !row.recover_mp" class="td-muted">—</span>
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

    <el-dialog :title="form.id ? '编辑道具' : '新增道具'" :visible.sync="dlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <div class="form-grid">
          <el-form-item label="名称" required>
            <el-input v-model.trim="form.name" maxlength="30" />
          </el-form-item>
          <el-form-item label="分类">
            <el-select v-model="form.cat" style="width:150px">
              <el-option v-for="(n, c) in catNames" :key="c" :label="n" :value="c" />
            </el-select>
          </el-form-item>
          <el-form-item label="来源">
            <el-select v-model="form.src" style="width:150px">
              <el-option v-for="(n, s) in srcNames" :key="s" :label="n" :value="s" />
            </el-select>
          </el-form-item>
          <el-form-item label="货币">
            <el-select v-model="form.currency" style="width:150px">
              <el-option v-for="(n, c) in currencyNames" :key="c" :label="n" :value="c" />
            </el-select>
          </el-form-item>
          <el-form-item label="价格">
            <el-input-number v-model.number="form.price" :min="0" />
          </el-form-item>
          <el-form-item label="需要等级">
            <el-input-number v-model.number="form.level" :min="0" />
          </el-form-item>
        </div>
        <div class="sub-title">战斗加成</div>
        <div class="form-grid">
          <el-form-item label="攻击"><el-input-number v-model.number="form.atk" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="防御"><el-input-number v-model.number="form.def" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="气血"><el-input-number v-model.number="form.hp" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="气力"><el-input-number v-model.number="form.mp" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="速度"><el-input-number v-model.number="form.spd" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="命中"><el-input-number v-model.number="form.hit" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="暴击"><el-input-number v-model.number="form.crit" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="闪避"><el-input-number v-model.number="form.dodge" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="恢复气血"><el-input-number v-model.number="form.recover_hp" :min="0" controls-position="right" /></el-form-item>
          <el-form-item label="恢复气力"><el-input-number v-model.number="form.recover_mp" :min="0" controls-position="right" /></el-form-item>
        </div>
        <div class="sub-title">锻造（来源=锻造时有效）</div>
        <div class="form-grid">
          <el-form-item label="材料需求">
            <el-input v-model="form.mats" placeholder="如：图纸x1,精铁x3" maxlength="100" />
          </el-form-item>
          <el-form-item label="锻造费(元宝)">
            <el-input-number v-model.number="form.fee" :min="0" />
          </el-form-item>
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
  name: 'AdminJwtItems',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', catFilter: '', srcFilter: '',
      dlg: false, saving: false, form: {},
      catNames: {
        medicine: '药品', weapon: '武器', helmet: '头盔', armor: '盔甲', shoes: '战鞋',
        necklace: '项链', bracelet: '手镯', ring: '戒指', medal: '勋章', material: '材料', other: '其他'
      },
      srcNames: { shop: '商店', forge: '锻造', battle: '比武' },
      currencyNames: { coins: 'G币', yuanbao: '元宝' },
      formDef: {
        id: 0, name: '', cat: 'other', src: 'shop', price: 0, currency: 'coins', level: 1,
        atk: 0, def: 0, hp: 0, mp: 0, spd: 0, hit: 0, crit: 0, dodge: 0,
        recover_hp: 0, recover_mp: 0, mats: '', fee: 0, status: 1, desc: ''
      }
    }
  },
  mounted () { this.load() },
  methods: {
    countSrc (s) { return this.list.filter(x => x.src === s).length },
    maxPrice () { return this.list.reduce((m, x) => Math.max(m, x.currency === 'coins' ? x.price : 0), 0) },
    catTag (c) { return { medicine: 'success', weapon: 'danger', helmet: 'warning', armor: '', shoes: '', necklace: 'success', bracelet: '', ring: 'warning', medal: 'danger', material: 'info', other: 'info' }[c] || 'info' },
    srcTag (s) { return { shop: 'success', forge: 'warning', battle: 'info' }[s] || 'info' },
    load () {
      this.loading = true
      api.get('/admin/jwt-items', { params: { page: this.page, size: this.size, word: this.word, cat: this.catFilter, src: this.srcFilter } }).then(r => {
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
      if (!this.form.name) { this.$message.warning('请填写道具名'); return }
      this.saving = true
      const body = { ...this.form }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      const call = this.form.id
        ? api.put('/admin/jwt-items/' + this.form.id, body)
        : api.post('/admin/jwt-items', body)
      call.then(r => { if (r.code === 0) done(); else { this.saving = false; this.$message.error(r.msg) } })
    },
    del (row) {
      this.$confirm('删除道具将同时清理玩家背包与已装备槽位，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/jwt-items/' + row.id).then(r => {
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
.td-green { color: #43a047; font-weight: 600; }
</style>
