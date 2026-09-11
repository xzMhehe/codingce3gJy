<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="家园号" width="90" />
        <el-table-column label="昵称" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nick || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="性别" width="70" align="center">
          <template slot-scope="{row}">{{ sexNames[row.sex] || '保密' }}</template>
        </el-table-column>
        <el-table-column label="等级" width="70" align="center">
          <template slot-scope="{row}"><span class="lv">{{ row.level }}</span>级</template>
        </el-table-column>
        <el-table-column label="经验" width="130" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.exp }}</span> / {{ row.next_exp }}</template>
        </el-table-column>
        <el-table-column prop="title_name" label="头衔" width="90" align="center" />
        <el-table-column prop="honor" label="荣誉" width="70" align="center" />
        <el-table-column prop="energy" label="能量" width="70" align="center" />
        <el-table-column label="G币" width="100" align="center">
          <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.coins }}</template>
        </el-table-column>
        <el-table-column label="元宝" width="100" align="center">
          <template slot-scope="{row}"><i class="el-icon-goods td-gold"></i>{{ row.yuanbao }}</template>
        </el-table-column>
        <el-table-column label="修炼" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.practicing === 1 ? 'warning' : 'info'" size="mini">{{ row.practicing === 1 ? '修炼中' : '空闲' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created" label="创建时间" width="160" />
        <el-table-column label="操作" width="130" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <!-- 详情（档案 + 装备 + 背包） -->
    <el-dialog title="玩家详情" :visible.sync="detailDlg" width="640px" :close-on-click-modal="false">
      <template v-if="detail">
        <el-descriptions :column="3" size="medium" border>
          <el-descriptions-item label="家园号">{{ detail.player.user_id }}</el-descriptions-item>
          <el-descriptions-item label="昵称">{{ detail.nickname || '—' }}</el-descriptions-item>
          <el-descriptions-item label="头衔">{{ detail.title_name }}</el-descriptions-item>
          <el-descriptions-item label="等级">{{ detail.player.level }}</el-descriptions-item>
          <el-descriptions-item label="经验">{{ detail.player.exp }}</el-descriptions-item>
          <el-descriptions-item label="能量">{{ detail.player.energy }}</el-descriptions-item>
          <el-descriptions-item label="G币">{{ detail.coins }}</el-descriptions-item>
          <el-descriptions-item label="元宝">{{ detail.yuanbao }}</el-descriptions-item>
          <el-descriptions-item label="荣誉">{{ detail.player.honor }}</el-descriptions-item>
        </el-descriptions>
        <div class="sub-title">装备（八槽）</div>
        <el-table :data="detail.slots" size="mini" border>
          <el-table-column prop="slot" label="部位" width="100" align="center" />
          <el-table-column prop="item_id" label="道具ID" width="90" align="center" />
          <el-table-column prop="item_name" label="装备" min-width="120" show-overflow-tooltip />
        </el-table>
        <div class="sub-title">背包（{{ detail.bag.length }} 种）</div>
        <el-table :data="detail.bag" size="mini" border max-height="260">
          <el-table-column prop="item_id" label="道具ID" width="90" align="center" />
          <el-table-column prop="name" label="名称" min-width="120" show-overflow-tooltip />
          <el-table-column prop="cat" label="分类" width="90" align="center" />
          <el-table-column prop="amount" label="数量" width="80" align="center" />
        </el-table>
      </template>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 编辑 -->
    <el-dialog title="编辑玩家" :visible.sync="editDlg" width="600px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <div class="form-grid">
          <el-form-item label="等级">
            <el-input-number v-model.number="form.level" :min="1" :max="9999" />
          </el-form-item>
          <el-form-item label="经验">
            <el-input-number v-model.number="form.exp" :min="0" />
          </el-form-item>
          <el-form-item label="能量">
            <el-input-number v-model.number="form.energy" :min="0" />
          </el-form-item>
          <el-form-item label="头衔">
            <el-select v-model.number="form.title" style="width:140px">
              <el-option v-for="(n, i) in titles" :key="i" :label="n" :value="i" />
            </el-select>
          </el-form-item>
          <el-form-item label="气血">
            <el-input-number v-model.number="form.cur_hp" :min="0" />
          </el-form-item>
          <el-form-item label="气力">
            <el-input-number v-model.number="form.cur_mp" :min="0" />
          </el-form-item>
        </div>
        <div class="sub-title">能量分配</div>
        <div class="form-grid">
          <el-form-item label="气血">
            <el-input-number v-model.number="form.e_hp" :min="0" />
          </el-form-item>
          <el-form-item label="气力">
            <el-input-number v-model.number="form.e_mp" :min="0" />
          </el-form-item>
          <el-form-item label="速度">
            <el-input-number v-model.number="form.e_spd" :min="0" />
          </el-form-item>
          <el-form-item label="攻击">
            <el-input-number v-model.number="form.e_atk" :min="0" />
          </el-form-item>
          <el-form-item label="防御">
            <el-input-number v-model.number="form.e_def" :min="0" />
          </el-form-item>
        </div>
        <div class="sub-title">货币</div>
        <div class="form-grid">
          <el-form-item label="G币">
            <el-input-number v-model.number="form.coins" :min="0" />
          </el-form-item>
          <el-form-item label="元宝">
            <el-input-number v-model.number="form.yuanbao" :min="0" />
          </el-form-item>
        </div>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminJwtPlayers',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, word: '',
      detailDlg: false, detail: null,
      editDlg: false, saving: false, editUid: 0, form: {},
      sexNames: { 0: '保密', 1: '男', 2: '女' },
      titles: ['无名小卒', '少侠', '侠士', '大侠', '豪杰', '宗师', '霸者', '圣者', '武尊', '武王', '至尊战神']
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/jwt-players', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDetail (row) {
      this.detail = null
      this.detailDlg = true
      api.get('/admin/jwt-players/' + row.user_id + '/detail').then(r => {
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    openEdit (row) {
      this.editUid = row.user_id
      this.form = {
        level: row.level, exp: row.exp, energy: row.energy, title: row.title,
        e_hp: row.e_hp, e_mp: row.e_mp, e_spd: row.e_spd, e_atk: row.e_atk, e_def: row.e_def,
        cur_hp: row.cur_hp, cur_mp: row.cur_mp, coins: row.coins, yuanbao: row.yuanbao
      }
      this.editDlg = true
    },
    save () {
      this.saving = true
      api.put('/admin/jwt-players/' + this.editUid, this.form).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.form-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0 14px; }
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 12px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
