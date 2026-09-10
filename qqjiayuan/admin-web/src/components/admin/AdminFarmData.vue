<template>
  <div class="farm-admin">
    <el-tabs v-model="tab" type="card" @tab-click="onTab">
      <!-- ============ 用户数据 ============ -->
      <el-tab-pane label="用户数据" name="users">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="userWord" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                      @keyup.enter.native="userPage = 1; loadUsers()" />
            <el-button type="primary" icon="el-icon-search" @click="userPage = 1; loadUsers()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="users" v-loading="userLoading" stripe>
            <el-table-column prop="user_id" label="家园号" width="90" />
            <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="name" label="农场名" min-width="130" show-overflow-tooltip />
            <el-table-column label="等级" width="90" align="center">
              <template slot-scope="{row}"><span class="lv">{{ row.level }}</span>级</template>
            </el-table-column>
            <el-table-column label="经验" width="100" align="center">
              <template slot-scope="{row}">{{ row.point }} / {{ row.need }}</template>
            </el-table-column>
            <el-table-column prop="lands" label="菜地" width="70" align="center" />
            <el-table-column prop="mucks" label="施肥次" width="80" align="center" />
            <el-table-column prop="bag_kinds" label="背包种数" width="85" align="center" />
            <el-table-column prop="wh_kinds" label="仓库种数" width="85" align="center" />
            <el-table-column prop="slave_n" label="奴隶" width="70" align="center" />
            <el-table-column label="操作" width="110" header-align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-view" @click="openDetail(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="userTotal"
                         :page-size="userSize" :current-page="userPage" :page-sizes="[10, 20, 50]"
                         @current-change="p => { userPage = p; loadUsers() }"
                         @size-change="s => { userSize = s; userPage = 1; loadUsers() }" />
        </el-card>
      </el-tab-pane>

      <!-- ============ 日志流水 ============ -->
      <el-tab-pane label="日志流水" name="logs">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-radio-group v-model="logType" size="small" @change="logPage = 1; loadLogs()">
              <el-radio-button label="msg">农场消息</el-radio-button>
              <el-radio-button label="steal">偷菜记录</el-radio-button>
              <el-radio-button label="slave">奴隶记录</el-radio-button>
            </el-radio-group>
            <el-input v-model="logWord" placeholder="家园号/昵称/关键词" clearable style="width:200px"
                      @keyup.enter.native="logPage = 1; loadLogs()" />
            <el-button type="primary" icon="el-icon-search" @click="logPage = 1; loadLogs()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="logs" v-loading="logLoading" stripe>
            <!-- 农场消息 -->
            <template v-if="logType === 'msg'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="from" label="发送人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="to_uid" label="接收人(家园号)" width="120" />
              <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
              <el-table-column label="状态" width="80" align="center">
                <template slot-scope="{row}">
                  <el-tag :type="row.status === 0 ? 'warning' : 'info'" size="mini">{{ row.status === 0 ? '未读' : '已读' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="时间" width="150" />
            </template>
            <!-- 偷菜记录 -->
            <template v-else-if="logType === 'steal'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="thief" label="小偷" min-width="110" show-overflow-tooltip />
              <el-table-column prop="owner" label="被偷农场主" min-width="110" show-overflow-tooltip />
              <el-table-column prop="land_id" label="菜地ID" width="90" />
              <el-table-column prop="created_at" label="时间" width="150" />
            </template>
            <!-- 奴隶记录 -->
            <template v-else>
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="slave" label="奴隶" min-width="110" show-overflow-tooltip />
              <el-table-column prop="owner" label="农场主" min-width="110" show-overflow-tooltip />
              <el-table-column prop="name" label="奴隶名" min-width="100" />
              <el-table-column label="惩罚/安抚" width="100" align="center">
                <template slot-scope="{row}">{{ row.punish }} / {{ row.appease }}</template>
              </el-table-column>
              <el-table-column prop="created_at" label="被抓时间" width="150" />
            </template>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="logTotal"
                         :page-size="logSize" :current-page="logPage" :page-sizes="[15, 30, 50]"
                         @current-change="p => { logPage = p; loadLogs() }"
                         @size-change="s => { logSize = s; logPage = 1; loadLogs() }" />
        </el-card>
      </el-tab-pane>

      <!-- ============ 排行榜 ============ -->
      <el-tab-pane label="排行榜" name="rank">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <div class="grow" />
            <el-button type="primary" icon="el-icon-refresh" @click="loadRank">刷新</el-button>
          </div>
          <el-table :data="rank" v-loading="rankLoading" stripe>
            <el-table-column label="名次" width="80" align="center">
              <template slot-scope="{row}">
                <span :class="['rank-no', 'r' + (row.rank > 3 ? 0 : row.rank)]">{{ row.rank }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="user_id" label="家园号" width="90" />
            <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="name" label="农场名" min-width="130" show-overflow-tooltip />
            <el-table-column label="等级" width="80" align="center">
              <template slot-scope="{row}"><span class="lv">{{ row.level }}</span>级</template>
            </el-table-column>
            <el-table-column prop="point" label="经验" width="80" align="center" />
            <el-table-column prop="lands" label="菜地" width="70" align="center" />
          </el-table>
          <el-pagination background layout="total, prev, pager, next" :total="rankTotal"
                         :page-size="rankSize" :current-page="rankPage"
                         @current-change="p => { rankPage = p; loadRank() }" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 用户农场数据详情 -->
    <el-dialog title="用户农场数据" :visible.sync="detailDlg" width="920px" :close-on-click-modal="false">
      <div v-loading="detailLoading">
        <template v-if="detail.has_farm">
          <el-descriptions :title="detail.nickname + '（' + detail.user_id + '）' + ' · ' + detail.farm.name" :column="4" border size="small" style="margin-bottom:16px">
            <el-descriptions-item label="等级">{{ detail.farm.level }}级</el-descriptions-item>
            <el-descriptions-item label="经验">{{ detail.farm.point }} / {{ detail.farm.need }}</el-descriptions-item>
            <el-descriptions-item label="施肥次数">{{ detail.farm.mucks }}</el-descriptions-item>
            <el-descriptions-item label="摘取权限">{{ cstealName(detail.farm.csteal) }}</el-descriptions-item>
            <el-descriptions-item label="操作">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEditFarm">编辑农场</el-button>
            </el-descriptions-item>
          </el-descriptions>

          <el-tabs v-model="subTab" type="border-card" size="small">
            <el-tab-pane label="菜地" name="lands">
              <el-table :data="detail.lands" size="small" stripe>
                <el-table-column prop="sort" label="地块" width="60" />
                <el-table-column prop="name" label="作物" min-width="90">
                  <template slot-scope="{row}">{{ row.name || '空地' }}</template>
                </el-table-column>
                <el-table-column label="状态" width="170">
                  <template slot-scope="{row}">{{ landStatus(row) }}</template>
                </el-table-column>
                <el-table-column label="打理" width="120">
                  <template slot-scope="{row}">
                    {{ row.need_water ? '待浇' : '—' }} / {{ row.need_weed ? '待锄' : '—' }} / {{ row.need_pest ? '待除虫' : '—' }}
                  </template>
                </el-table-column>
                <el-table-column prop="yield" label="剩余产量" width="85" align="center" />
                <el-table-column prop="trap" label="陷阱" width="70" align="center">
                  <template slot-scope="{row}">{{ row.trap ? '有' : '无' }}</template>
                </el-table-column>
                <el-table-column label="操作" width="110" align="center">
                  <template slot-scope="{row}">
                    <el-button v-if="row.type === 1" size="mini" type="warning" plain @click="clearLand(row)">收回作物</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="背包" name="bag">
              <div class="toolbar" style="margin-bottom:10px">
                <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="openItem(null, 'bag')">新增道具</el-button>
                <span class="help-line">可修改数量（0=移除该道具）</span>
              </div>
              <el-table :data="detail.bag" size="small" stripe>
                <el-table-column label="类型" width="90" align="center">
                  <template slot-scope="{row}">
                    <el-tag :type="['', 'success', 'warning', 'danger'][row.dtype] || 'info'" size="mini">{{ typeName(row.dtype) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="140" />
                <el-table-column prop="amount" label="数量" width="90" align="center" />
                <el-table-column label="操作" width="110" align="center">
                  <template slot-scope="{row}">
                    <el-button size="mini" type="primary" plain @click="openItem(row, 'bag')">修改</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="仓库" name="wh">
              <div class="toolbar" style="margin-bottom:10px">
                <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="openItem(null, 'wh')">新增果实</el-button>
                <span class="help-line">可修改数量（0=移除），单价由对应种子决定</span>
              </div>
              <el-table :data="detail.warehouse" size="small" stripe>
                <el-table-column prop="name" label="果实" min-width="140" />
                <el-table-column prop="amount" label="数量" width="90" align="center" />
                <el-table-column prop="price" label="单价(G)" width="90" align="center" />
                <el-table-column label="操作" width="110" align="center">
                  <template slot-scope="{row}">
                    <el-button size="mini" type="primary" plain @click="openItem(row, 'wh')">修改</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="奴隶" name="slaves">
              <el-table :data="detail.slaves" size="small" stripe>
                <el-table-column prop="name" label="奴隶名" min-width="100" />
                <el-table-column prop="fid" label="家园号" width="90" align="center" />
                <el-table-column prop="punish" label="惩罚次数" width="90" align="center" />
                <el-table-column prop="appease" label="安抚次数" width="90" align="center" />
                <el-table-column prop="created_at" label="被抓时间" width="150" />
              </el-table>
              <el-empty v-if="!detail.slaves || !detail.slaves.length" description="没有奴隶记录" />
            </el-tab-pane>
          </el-tabs>
        </template>
        <el-empty v-else description="该用户还未开通开心农场" />
      </div>
    </el-dialog>

    <!-- 编辑农场基础信息 -->
    <el-dialog title="编辑农场" :visible.sync="farmDlg" width="480px" :close-on-click-modal="false" append-to-body>
      <el-form label-width="90px">
        <el-form-item label="农场名">
          <el-input v-model.trim="farmForm.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="等级">
          <el-input-number v-model="farmForm.level" :min="1" :max="99" />
        </el-form-item>
        <el-form-item label="经验">
          <el-input-number v-model="farmForm.point" :min="0" />
          <span class="help-line" style="margin-left:10px">升级所需 {{ (farmForm.level + 1) * 10 }} 点</span>
        </el-form-item>
        <el-form-item label="施肥次数">
          <el-input-number v-model="farmForm.mucks" :min="0" :max="99" />
        </el-form-item>
        <el-form-item label="摘取权限">
          <el-select v-model.number="farmForm.csteal" style="width:100%">
            <el-option :value="0" label="所有人可以摘取" />
            <el-option :value="1" label="仅好友可以摘取" />
            <el-option :value="4" label="禁止任何人摘取" />
          </el-select>
        </el-form-item>
        <el-form-item label="菜地数">
          <el-input-number v-model="farmForm.lands" :min="1" :max="20" />
          <span class="help-line" style="margin-left:10px">减少时只能回收空地</span>
        </el-form-item>
        <el-form-item label="G币">
          <el-input-number v-model="farmForm.coins" :min="0" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="farmDlg = false">取 消</el-button>
        <el-button type="primary" :loading="farmSaving" @click="saveFarm">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 编辑背包/仓库条目 -->
    <el-dialog :title="itemTitle" :visible.sync="itemDlg" width="460px" :close-on-click-modal="false" append-to-body>
      <el-form label-width="90px">
        <template v-if="itemIsNew && itemType === 'bag'">
          <el-form-item label="道具类型">
            <el-select v-model.number="itemForm.dtype" style="width:100%" @change="itemForm.oid = 0">
              <el-option :value="1" label="种子" />
              <el-option :value="2" label="化肥" />
              <el-option :value="3" label="陷阱" />
            </el-select>
          </el-form-item>
          <el-form-item label="具体道具">
            <el-select v-model.number="itemForm.oid" style="width:100%" filterable>
              <el-option v-for="o in itemOptions" :key="o.id" :value="o.id" :label="o.name + (o.seed_price ? '（种子价' + o.seed_price + 'G）' : '')" />
            </el-select>
          </el-form-item>
        </template>
        <el-form-item v-else-if="itemIsNew" label="果实名称">
          <el-input v-model.trim="itemForm.name" maxlength="30" placeholder="须与种子名一致，如：白萝卜" />
        </el-form-item>
        <el-form-item v-else label="名称">
          <el-input v-model="itemForm.name" disabled />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="itemForm.amount" :min="0" :max="99999" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="itemDlg = false">取 消</el-button>
        <el-button type="primary" :loading="itemSaving" @click="saveItem">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminFarmData',
  data () {
    return {
      tab: 'users',
      // 用户数据
      users: [], userTotal: 0, userPage: 1, userSize: 10, userLoading: false, userWord: '',
      // 日志流水
      logs: [], logTotal: 0, logPage: 1, logSize: 15, logLoading: false, logType: 'msg', logWord: '',
      // 排行榜
      rank: [], rankTotal: 0, rankPage: 1, rankSize: 20, rankLoading: false,
      // 详情
      detailDlg: false, detailLoading: false, detail: { has_farm: false }, subTab: 'lands',
      // 编辑农场
      farmDlg: false, farmSaving: false, farmForm: { name: '', level: 1, point: 0, mucks: 3, csteal: 0, lands: 4, coins: 0 },
      // 编辑条目（背包/仓库）
      itemDlg: false, itemSaving: false, itemType: 'bag', itemIsNew: false, itemRow: null,
      itemForm: { dtype: 1, oid: 0, name: '', amount: 0 },
      // 新增道具时的候选
      seeds: [], mucks: [], traps: []
    }
  },
  mounted () { this.loadUsers() },
  computed: {
    itemTitle () {
      if (this.itemIsNew) return this.itemType === 'bag' ? '新增背包道具' : '新增仓库果实'
      return this.itemType === 'bag' ? '编辑背包道具' : '编辑仓库果实'
    },
    itemOptions () {
      return { 1: this.seeds, 2: this.mucks, 3: this.traps }[this.itemForm.dtype] || []
    }
  },
  methods: {
    onTab () {
      if (this.tab === 'users' && this.users.length === 0) this.loadUsers()
      if (this.tab === 'logs' && this.logs.length === 0) this.loadLogs()
      if (this.tab === 'rank' && this.rank.length === 0) this.loadRank()
    },
    cstealName (v) { return { 0: '所有人', 1: '仅好友', 4: '禁止' }[v] || '所有人' },
    typeName (d) { return { 1: '种子', 2: '化肥', 3: '陷阱', 11: '果实' }[d] || d },
    landStatus (l) {
      if (l.type === 0) return l.plow ? '已翻空地' : '未开垦'
      if (l.mature) return '已成熟'
      return l.status_txt || '生长中'
    },
    loadUsers () {
      this.userLoading = true
      api.get('/admin/farm-users', { params: { page: this.userPage, size: this.userSize, word: this.userWord } }).then(r => {
        this.userLoading = false
        if (r.code === 0) {
          this.users = r.data.list
          this.userTotal = r.data.total
          this.userPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadLogs () {
      this.logLoading = true
      api.get('/admin/farm-logs', { params: { page: this.logPage, size: this.logSize, type: this.logType, word: this.logWord } }).then(r => {
        this.logLoading = false
        if (r.code === 0) {
          this.logs = r.data.list
          this.logTotal = r.data.total
          this.logPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadRank () {
      this.rankLoading = true
      api.get('/admin/farm-rank', { params: { page: this.rankPage, size: this.rankSize } }).then(r => {
        this.rankLoading = false
        if (r.code === 0) {
          this.rank = r.data.list
          this.rankTotal = r.data.total
          this.rankPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDetail (row) {
      this.detailDlg = true
      this.detailLoading = true
      this.detail = { has_farm: false }
      this.subTab = 'lands'
      this.loadDetail(row.user_id)
    },
    loadDetail (uid) {
      api.get('/admin/farm-users/' + uid).then(r => {
        this.detailLoading = false
        if (r.code === 0) this.detail = r.data
        else this.$message.error(r.msg)
      })
    },
    // ===== 编辑农场 =====
    openEditFarm () {
      const f = this.detail.farm || {}
      this.farmForm = {
        name: f.name || '', level: f.level || 1, point: f.point || 0,
        mucks: f.mucks != null ? f.mucks : 3, csteal: f.csteal || 0,
        lands: (this.detail.lands || []).length || 4,
        coins: this.detail.coins != null ? this.detail.coins : 0
      }
      this.farmDlg = true
    },
    saveFarm () {
      const uid = this.detail.user_id
      this.farmSaving = true
      const farmPayload = {
        name: this.farmForm.name, level: this.farmForm.level, point: this.farmForm.point,
        mucks: this.farmForm.mucks, csteal: this.farmForm.csteal, lands: this.farmForm.lands
      }
      api.put('/admin/farm-users/' + uid + '/farm', farmPayload).then(r => {
        if (r.code !== 0) throw new Error(r.msg)
        return api.put('/admin/farm-users/' + uid + '/coins', { coins: this.farmForm.coins }).then(rc => {
          if (rc.code !== 0) throw new Error(rc.msg)
        })
      }).then(() => {
        this.farmSaving = false
        this.farmDlg = false
        this.$message.success('已保存')
        this.detailLoading = true
        this.loadDetail(uid)
        this.loadUsers()
      }).catch(e => {
        this.farmSaving = false
        this.$message.error(e.message || '保存失败')
      })
    },
    // ===== 收回菜地作物 =====
    clearLand (l) {
      const uid = this.detail.user_id
      this.$confirm('收回菜地 ' + l.sort + ' 上的作物「' + (l.name || '') + '」？', '提示', { type: 'warning' }).then(() => {
        api.put('/admin/farm-users/' + uid + '/land/' + l.id + '/clear').then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已收回')
            this.detailLoading = true
            this.loadDetail(uid)
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ===== 编辑条目（背包/仓库） =====
    openItem (row, type) {
      this.itemType = type
      this.itemIsNew = !row
      this.itemRow = row
      if (row) {
        this.itemForm = { dtype: row.dtype || 1, oid: row.oid || 0, name: row.name || '', amount: row.amount || 0 }
      } else {
        this.itemForm = { dtype: 1, oid: 0, name: '', amount: 0 }
        if (this.itemType === 'bag') this.loadItemOptions()
      }
      this.itemDlg = true
    },
    loadItemOptions () {
      Promise.all([api.get('/admin/farm-seeds'), api.get('/admin/farm-mucks'), api.get('/admin/farm-traps')]).then(rs => {
        if (rs[0].code === 0) this.seeds = rs[0].data
        if (rs[1].code === 0) this.mucks = rs[1].data
        if (rs[2].code === 0) this.traps = rs[2].data
      })
    },
    saveItem () {
      const uid = this.detail.user_id
      this.itemSaving = true
      let payload
      if (this.itemType === 'bag') {
        const opt = this.itemOptions.find(o => o.id === this.itemForm.oid)
        payload = { dtype: this.itemForm.dtype, oid: this.itemForm.oid, name: this.itemIsNew ? (opt ? opt.name : '') : this.itemForm.name, amount: this.itemForm.amount }
        if (this.itemIsNew && !payload.oid) { this.itemSaving = false; this.$message.warning('请选择道具'); return }
      } else {
        payload = { id: this.itemIsNew ? 0 : this.itemRow.id, name: this.itemForm.name, amount: this.itemForm.amount }
        if (this.itemIsNew && !payload.name) { this.itemSaving = false; this.$message.warning('请填写果实名称'); return }
      }
      api.put('/admin/farm-users/' + uid + '/bag/' + (this.itemType === 'bag' ? 'bag' : 'warehouse'), payload).then(r => {
        this.itemSaving = false
        if (r.code === 0) {
          this.itemDlg = false
          this.$message.success('已保存')
          this.detailLoading = true
          this.loadDetail(uid)
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
