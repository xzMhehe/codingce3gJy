<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- ================= 军衔配置 ================= -->
        <el-tab-pane label="军衔配置" name="cfg">
          <div class="toolbar">
            <span class="td-sub">
              复刻原版军衔页：军衔等级 / 职位 / 需要声望 / <b>可建城数</b>。
              「可建城数」就是该军衔下玩家能拥有的城市数量上限。
            </span>
            <div class="grow" />
            <el-button type="warning" plain icon="el-icon-refresh-left" @click="resetRanks">恢复默认</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadRanks">刷新</el-button>
          </div>
          <el-table :data="ranks" v-loading="loadingRank" stripe border>
            <el-table-column prop="level" label="等级" width="70" align="center" />
            <el-table-column prop="name" label="军衔" min-width="120">
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="post" label="职位" width="110" align="center" />
            <el-table-column label="需要声望" width="110" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.need_prestige }}</span></template>
            </el-table-column>
            <el-table-column label="可建城数" width="110" align="center">
              <template slot-scope="{row}"><span class="td-blue">{{ row.city_max }}</span></template>
            </el-table-column>
            <el-table-column label="该军衔玩家" width="120" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.player_count }}</span></template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="150" show-overflow-tooltip />
            <el-table-column label="操作" width="110" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- ================= 玩家军衔 ================= -->
        <el-tab-pane label="玩家军衔维护" name="players">
          <div class="toolbar">
            <el-input v-model="word" placeholder="昵称 / 用户ID / 游戏ID" clearable style="width:220px"
                      @keyup.enter.native="page = 1; loadPlayers()" />
            <el-button type="primary" icon="el-icon-search" @click="page = 1; loadPlayers()">查询</el-button>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadPlayers">刷新</el-button>
          </div>
          <el-table :data="players" v-loading="loading" stripe border>
            <el-table-column prop="user_id" label="用户ID" width="80" align="center" />
            <el-table-column prop="game_uid" label="游戏ID" width="100" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.game_uid || row.user_id }}</span></template>
            </el-table-column>
            <el-table-column prop="home_num" label="家园号码" width="100" align="center" />
            <el-table-column label="玩家" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.nickname || row.player_name || '—' }}</span></template>
            </el-table-column>
            <el-table-column label="军衔" width="120" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" type="warning">{{ row.rank_name }}</el-tag>
                <span class="td-sub">({{ row.rank_post }})</span>
              </template>
            </el-table-column>
            <el-table-column label="声望" width="100" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.prestige }}</span></template>
            </el-table-column>
            <el-table-column label="可建城数" width="100" align="center">
              <template slot-scope="{row}"><span class="td-blue">{{ row.city_max }}</span></template>
            </el-table-column>
            <el-table-column label="已有城" width="90" align="center">
              <template slot-scope="{row}">
                <span :class="row.over_limit ? 'td-red' : 'td-mono'">{{ row.city_count }}</span>
                <el-tag v-if="row.over_limit" size="mini" type="danger" style="margin-left:4px">超限</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="设为指定军衔" @click="openSetRank(row)" />
                <el-button size="mini" type="warning" plain icon="el-icon-s-claim" title="直接改声望" @click="openSetPrestige(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
            <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                           :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { page = p; loadPlayers() }"
                           @size-change="s => { size = s; page = 1; loadPlayers() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 编辑军衔 -->
    <el-dialog :title="'编辑军衔 · ' + (form.level ? 'Lv.' + form.level + ' ' + form.name : '')"
               :visible.sync="dlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="军衔名" required>
              <el-input v-model="form.name" maxlength="20" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="职位">
              <el-input v-model="form.post" maxlength="20" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="需要声望">
              <el-input-number v-model.number="form.need_prestige" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="可建城数">
              <el-input-number v-model.number="form.city_max" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="说明">
          <el-input v-model="form.des" maxlength="200" />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 改完立即生效（后端会重载配置缓存，不用重启） -->
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 设为指定军衔 -->
    <el-dialog :title="'设置军衔 · ' + (setRow.nickname || setRow.user_id)"
               :visible.sync="setDlg" width="600px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="当前">
          <span class="td-main">{{ setRow.rank_name }}({{ setRow.rank_post }})</span>
          <span class="td-sub">声望 {{ setRow.prestige }} · 可建 {{ setRow.city_max }} 座 · 已有 {{ setRow.city_count }} 座</span>
        </el-form-item>
        <el-form-item label="设为军衔">
          <el-select v-model="setRankId" style="width:100%">
            <el-option v-for="r in ranks" :key="r.id"
                       :label="'Lv.' + r.level + ' ' + r.name + '（' + r.post + '，需声望 ' + r.need_prestige + '，可建 ' + r.city_max + ' 座）'"
                       :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 设置后会把该玩家声望写成对应门槛值，军衔与可建城数立即生效 -->
      <div slot="footer">
        <el-button @click="setDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSetRank">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 直接改声望 -->
    <el-dialog :title="'设置声望 · ' + (preRow.nickname || preRow.user_id)"
               :visible.sync="preDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="当前声望">
          <span class="td-mono">{{ preRow.prestige }}</span>
          <span class="td-sub">（{{ preRow.rank_name }}）</span>
        </el-form-item>
        <el-form-item label="设为">
          <el-input-number v-model.number="preValue" :min="0" controls-position="right" style="width:100%" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="preDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSetPrestige">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyRanks',
  data () {
    return {
      tab: 'cfg',
      ranks: [], loadingRank: false,
      players: [], total: 0, page: 1, size: 5, word: '', loading: false,
      dlg: false, form: {}, saving: false,
      setDlg: false, setRow: {}, setRankId: 0,
      preDlg: false, preRow: {}, preValue: 0
    }
  },
  mounted () { this.loadRanks(); this.loadPlayers() },
  methods: {
    onTab () { if (this.tab === 'players') this.loadPlayers(); else this.loadRanks() },
    loadRanks () {
      this.loadingRank = true
      api.get('/admin/ezfy-ranks').then(r => {
        this.loadingRank = false
        if (r.code === 0) this.ranks = r.data.list || []
        else this.$message.error(r.msg)
      })
    },
    resetRanks () {
      this.$confirm('把军衔表恢复成内置默认（列兵…五星上将，可建城数=等级）？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-ranks/reset', {}).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已恢复'); this.loadRanks() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openEdit (row) {
      this.form = Object.assign({}, row)
      this.dlg = true
    },
    doSave () {
      this.saving = true
      api.put('/admin/ezfy-ranks/' + this.form.id, {
        name: this.form.name, post: this.form.post,
        need_prestige: this.form.need_prestige, city_max: this.form.city_max,
        des: this.form.des || ''
      }).then(r => {
        this.saving = false
        if (r.code === 0) { this.dlg = false; this.$message.success(r.data.msg || '已保存'); this.loadRanks() }
        else this.$message.error(r.msg)
      })
    },
    loadPlayers () {
      this.loading = true
      api.get('/admin/ezfy-rank-players', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.players = r.data.list || []
          this.total = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    openSetRank (row) {
      this.setRow = row
      this.setRankId = row.rank_id || 1
      this.setDlg = true
    },
    doSetRank () {
      this.saving = true
      api.put('/admin/ezfy-rank-players/' + this.setRow.user_id, { rank_id: this.setRankId }).then(r => {
        this.saving = false
        if (r.code === 0) { this.setDlg = false; this.$message.success(r.data.msg || '已设置'); this.loadPlayers(); this.loadRanks() }
        else this.$message.error(r.msg)
      })
    },
    openSetPrestige (row) {
      this.preRow = row
      this.preValue = row.prestige
      this.preDlg = true
    },
    doSetPrestige () {
      this.saving = true
      api.post('/admin/ezfy-rank-players/' + this.preRow.user_id + '/prestige', { prestige: this.preValue }).then(r => {
        this.saving = false
        if (r.code === 0) { this.preDlg = false; this.$message.success(r.data.msg || '已设置'); this.loadPlayers(); this.loadRanks() }
        else this.$message.error(r.msg)
      })
    }
  },
  watch: {
    word () { this.page = 1 }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
