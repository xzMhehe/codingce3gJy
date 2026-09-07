<template>
  <div>
    <el-tabs v-model="tab" type="card" @tab-click="onTab">
      <!-- ============ 用户游戏数据 ============ -->
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
            <el-table-column prop="name" label="花园名" min-width="130" show-overflow-tooltip />
            <el-table-column label="等级" width="110">
              <template slot-scope="{row}">
                <span class="lv">{{ row.level }}</span>
                <span class="lv-name">{{ row.level_name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="point" label="经验" width="80" />
            <el-table-column prop="lands" label="花圃" width="70" />
            <el-table-column prop="map_got" label="点亮图谱" width="90" />
            <el-table-column prop="bag_kinds" label="背包种数" width="90" />
            <el-table-column prop="flower_kinds" label="花篮种数" width="90" />
            <el-table-column prop="bottle_kinds" label="花瓶种数" width="90" />
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

      <!-- ============ 游戏日志/流水 ============ -->
      <el-tab-pane label="日志流水" name="logs">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-radio-group v-model="logType" size="small" @change="logPage = 1; loadLogs()">
              <el-radio-button label="gift">送花记录</el-radio-button>
              <el-radio-button label="msg">花园消息</el-radio-button>
              <el-radio-button label="sign">签到记录</el-radio-button>
              <el-radio-button label="pick">采摘记录</el-radio-button>
            </el-radio-group>
            <el-input v-model="logWord" placeholder="关键词搜索" clearable style="width:200px"
                      @keyup.enter.native="logPage = 1; loadLogs()" />
            <el-button type="primary" icon="el-icon-search" @click="logPage = 1; loadLogs()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="logs" v-loading="logLoading" stripe>
            <!-- 送花 -->
            <template v-if="logType === 'gift'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="from_nick" label="赠送人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="to_nick" label="接收人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="flower" label="花朵" min-width="110" />
              <el-table-column prop="amount" label="数量" width="70" />
              <el-table-column prop="remark" label="留言" min-width="160" show-overflow-tooltip />
              <el-table-column prop="created_at" label="时间" width="150" />
            </template>
            <!-- 花园消息 -->
            <template v-else-if="logType === 'msg'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="from_nick" label="操作人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="to_nick" label="花园主" min-width="110" show-overflow-tooltip />
              <el-table-column prop="remark" label="内容" min-width="200" show-overflow-tooltip />
              <el-table-column prop="created_at" label="时间" width="150" />
            </template>
            <!-- 签到 -->
            <template v-else-if="logType === 'sign'">
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="uid" label="家园号" width="90" />
              <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
              <el-table-column prop="sign_date" label="签到日期" width="120" />
              <el-table-column prop="week_day" label="星期" width="80">
                <template slot-scope="{row}">{{ weekName(row.week_day) }}</template>
              </el-table-column>
              <el-table-column prop="day_no" label="本轮第几天" width="100" />
            </template>
            <!-- 采摘 -->
            <template v-else>
              <el-table-column prop="id" label="ID" width="70" />
              <el-table-column prop="nickname" label="采摘人" min-width="110" show-overflow-tooltip />
              <el-table-column prop="owner_nick" label="园主" min-width="110" show-overflow-tooltip />
              <el-table-column prop="flower" label="花朵" min-width="110" />
              <el-table-column prop="land_id" label="花圃ID" width="90" />
            </template>
          </el-table>
          <el-pagination background layout="total, sizes, prev, pager, next" :total="logTotal"
                         :page-size="logSize" :current-page="logPage" :page-sizes="[15, 30, 50]"
                         @current-change="p => { logPage = p; loadLogs() }"
                         @size-change="s => { logSize = s; logPage = 1; loadLogs() }" />
        </el-card>
      </el-tab-pane>

      <!-- ============ 游戏排行榜 ============ -->
      <el-tab-pane label="排行榜" name="rank">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <div class="grow" />
            <el-button type="primary" icon="el-icon-refresh" @click="loadRank">刷新</el-button>
          </div>
          <el-table :data="rank" v-loading="rankLoading" stripe>
            <el-table-column label="名次" width="80">
              <template slot-scope="{row}">
                <span :class="['rank-no', 'r' + (row.rank > 3 ? 0 : row.rank)]">{{ row.rank }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="user_id" label="家园号" width="90" />
            <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="name" label="花园名" min-width="130" show-overflow-tooltip />
            <el-table-column label="等级" width="110">
              <template slot-scope="{row}">
                <span class="lv">{{ row.level }}</span>
                <span class="lv-name">{{ row.level_name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="point" label="经验" width="80" />
            <el-table-column prop="lands" label="花圃" width="70" />
            <el-table-column prop="map_got" label="点亮图谱" width="90" />
            <el-table-column prop="basket_cnt" label="花篮累计" width="90" />
            <el-table-column prop="bottle_cnt" label="花瓶累计" width="90" />
          </el-table>
          <el-pagination background layout="total, prev, pager, next" :total="rankTotal"
                         :page-size="rankSize" :current-page="rankPage"
                         @current-change="p => { rankPage = p; loadRank() }" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 用户游戏数据详情 -->
    <el-dialog title="用户游戏数据" :visible.sync="detailDlg" width="880px" :close-on-click-modal="false">
      <div v-loading="detailLoading">
        <template v-if="detail.has_garden">
          <el-descriptions :title="detail.nickname + '（' + detail.user_id + '）' + ' · ' + detail.garden.name" :column="4" border size="small" style="margin-bottom:16px">
            <el-descriptions-item label="等级">{{ detail.garden.level }}（{{ detail.garden.level_name }}）</el-descriptions-item>
            <el-descriptions-item label="经验">{{ detail.garden.point }} / {{ detail.garden.need }}</el-descriptions-item>
            <el-descriptions-item label="花圃">{{ detail.garden.lands }}块</el-descriptions-item>
            <el-descriptions-item label="点亮图谱">{{ detail.garden.map_got }}</el-descriptions-item>
            <el-descriptions-item label="普通">{{ detail.garden.common }}</el-descriptions-item>
            <el-descriptions-item label="独特">{{ detail.garden.festival }}</el-descriptions-item>
            <el-descriptions-item label="珍稀">{{ detail.garden.scarce }}</el-descriptions-item>
            <el-descriptions-item label="公告">{{ detail.garden.notice }}</el-descriptions-item>
            <el-descriptions-item label="操作">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEditGarden">编辑花园</el-button>
            </el-descriptions-item>
          </el-descriptions>

          <el-tabs v-model="subTab" type="border-card" size="small">
            <el-tab-pane label="花圃" name="plots">
              <el-table :data="detail.plots" size="small" stripe>
                <el-table-column prop="plot" label="地块" width="60" />
                <el-table-column prop="seed" label="花种" min-width="90" />
                <el-table-column prop="name" label="花朵" min-width="90" />
                <el-table-column label="状态" width="80">
                  <template slot-scope="{row}">
                    <el-tag :type="statusTag(row.status)" size="mini">{{ statusName(row.status) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="打理" width="120">
                  <template slot-scope="{row}">
                    {{ row.drys ? '浇' : '未浇' }} / {{ row.weed ? '锄' : '未锄' }} / {{ row.pest ? '捉' : '未捉' }}
                  </template>
                </el-table-column>
                <el-table-column prop="amount" label="数量" width="70" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="背包" name="bag">
              <div class="toolbar" style="margin-bottom:10px">
                <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="openItem(null,'bag')">新增花种</el-button>
                <span class="help-line">可修改数量（0=移除该花种）</span>
              </div>
              <el-table :data="detail.bag" size="small" stripe>
                <el-table-column prop="name" label="花种" min-width="140" />
                <el-table-column prop="amount" label="数量" width="100" />
                <el-table-column label="操作" width="120" header-align="center">
                  <template slot-scope="{row}">
                    <el-button size="mini" type="primary" plain @click="openItem(row,'bag')">修改</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="花篮" name="flowers">
              <div class="toolbar" style="margin-bottom:10px">
                <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="openItem(null,'flowers')">新增花朵</el-button>
                <span class="help-line">可修改数量（0=移除该花朵）</span>
              </div>
              <el-table :data="detail.flowers" size="small" stripe>
                <el-table-column prop="flower" label="花朵" min-width="140" />
                <el-table-column prop="count" label="数量" width="100" />
                <el-table-column label="操作" width="120" header-align="center">
                  <template slot-scope="{row}">
                    <el-button size="mini" type="primary" plain @click="openItem(row,'flowers')">修改</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="花瓶" name="bottle">
              <div class="toolbar" style="margin-bottom:10px">
                <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="openItem(null,'bottle')">新增花朵</el-button>
                <span class="help-line">可修改数量（0=移除该花朵）</span>
              </div>
              <el-table :data="detail.bottle" size="small" stripe>
                <el-table-column prop="flower" label="花朵" min-width="140" />
                <el-table-column prop="count" label="数量" width="100" />
                <el-table-column label="操作" width="120" header-align="center">
                  <template slot-scope="{row}">
                    <el-button size="mini" type="primary" plain @click="openItem(row,'bottle')">修改</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
          </el-tabs>
        </template>
        <el-empty v-else description="该用户还未开通魔法花园" />
      </div>
    </el-dialog>

    <!-- 编辑花园基础信息 -->
    <el-dialog title="编辑花园" :visible.sync="gardenDlg" width="480px" :close-on-click-modal="false" append-to-body>
      <el-form label-width="90px">
        <el-form-item label="花园名">
          <el-input v-model.trim="gardenForm.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="公告">
          <el-input v-model.trim="gardenForm.notice" maxlength="100" />
        </el-form-item>
        <el-form-item label="采摘权限">
          <el-select v-model.number="gardenForm.config" style="width:100%">
            <el-option :value="0" label="全部可采摘" />
            <el-option :value="1" label="仅好友" />
            <el-option :value="2" label="禁止采摘" />
          </el-select>
        </el-form-item>
        <el-form-item label="花圃数">
          <el-input-number v-model="gardenForm.lands" :min="1" :max="10" />
        </el-form-item>
        <el-form-item label="G币">
          <el-input-number v-model="gardenForm.coins" :min="0" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="gardenDlg = false">取 消</el-button>
        <el-button type="primary" :loading="gardenSaving" @click="saveGarden">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 编辑背包/花篮/花瓶条目 -->
    <el-dialog :title="itemTitle" :visible.sync="itemDlg" width="440px" :close-on-click-modal="false" append-to-body>
      <el-form label-width="80px">
        <el-form-item :label="itemLabelName">
          <el-input v-model.trim="itemForm.name" maxlength="30" placeholder="输入名称" />
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
  name: 'AdminGardenData',
  data () {
    return {
      tab: 'users',
      // 用户数据
      users: [], userTotal: 0, userPage: 1, userSize: 10, userLoading: false, userWord: '',
      // 日志流水
      logs: [], logTotal: 0, logPage: 1, logSize: 15, logLoading: false, logType: 'gift', logWord: '',
      // 排行榜
      rank: [], rankTotal: 0, rankPage: 1, rankSize: 20, rankLoading: false,
      // 详情
      detailDlg: false, detailLoading: false, detail: { has_garden: false }, subTab: 'plots',
      // 编辑花园
      gardenDlg: false, gardenSaving: false, gardenForm: { name: '', notice: '', config: 0, lands: 2, coins: 0 },
      // 编辑条目（背包/花篮/花瓶）
      itemDlg: false, itemSaving: false, itemType: 'bag', itemForm: { name: '', amount: 0 }
    }
  },
  mounted () { this.loadUsers() },
  computed: {
    itemTitle () {
      return { bag: '编辑背包花种', flowers: '编辑花篮花朵', bottle: '编辑花瓶花朵' }[this.itemType] || '编辑'
    },
    itemLabelName () {
      return this.itemType === 'bag' ? '花种名' : '花朵名'
    }
  },
  methods: {
    onTab () {
      if (this.tab === 'users' && this.users.length === 0) this.loadUsers()
      if (this.tab === 'logs' && this.logs.length === 0) this.loadLogs()
      if (this.tab === 'rank' && this.rank.length === 0) this.loadRank()
    },
    loadUsers () {
      this.userLoading = true
      api.get('/admin/garden-users', { params: { page: this.userPage, size: this.userSize, word: this.userWord } }).then(r => {
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
      api.get('/admin/garden-logs', { params: { page: this.logPage, size: this.logSize, type: this.logType, word: this.logWord } }).then(r => {
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
      api.get('/admin/garden-rank', { params: { page: this.rankPage, size: this.rankSize } }).then(r => {
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
      this.detail = { has_garden: false }
      this.subTab = 'plots'
      this.loadDetail(row.user_id)
    },
    loadDetail (uid) {
      api.get('/admin/garden-users/' + uid).then(r => {
        this.detailLoading = false
        if (r.code === 0) this.detail = r.data
        else this.$message.error(r.msg)
      })
    },
    // ===== 编辑花园 =====
    openEditGarden () {
      const g = this.detail.garden || {}
      this.gardenForm = {
        name: g.name || '', notice: g.notice || '', config: g.config || 0,
        lands: g.lands || 2, coins: this.detail.coins != null ? this.detail.coins : 0
      }
      this.gardenDlg = true
    },
    saveGarden () {
      const uid = this.detail.user_id
      this.gardenSaving = true
      const gardenPayload = {
        name: this.gardenForm.name, notice: this.gardenForm.notice,
        config: this.gardenForm.config, lands: this.gardenForm.lands
      }
      api.put('/admin/garden-users/' + uid + '/garden', gardenPayload).then(r => {
        if (r.code === 0) {
          const coinsPayload = { coins: this.gardenForm.coins }
          return api.put('/admin/garden-users/' + uid + '/coins', coinsPayload).then(rc => {
            if (rc.code !== 0) throw new Error(rc.msg)
          })
        } else throw new Error(r.msg)
      }).then(() => {
        this.gardenSaving = false
        this.gardenDlg = false
        this.$message.success('已保存')
        this.detailLoading = true
        this.loadDetail(uid)
      }).catch(e => {
        this.gardenSaving = false
        this.$message.error(e.message || '保存失败')
      })
    },
    // ===== 编辑条目（背包/花篮/花瓶） =====
    openItem (row, type) {
      this.itemType = type
      this.itemForm = { name: row ? (row.name || row.flower || '') : '', amount: row ? (row.amount != null ? row.amount : row.count) : 0 }
      this.itemDlg = true
    },
    saveItem () {
      const uid = this.detail.user_id
      const name = this.itemForm.name
      if (!name) { this.$message.warning('请填写名称'); return }
      this.itemSaving = true
      let call
      if (this.itemType === 'bag') {
        call = api.put('/admin/garden-users/' + uid + '/bag', { name, amount: this.itemForm.amount })
      } else {
        call = api.put('/admin/garden-users/' + uid + '/flowers/' + this.itemType, { flower: name, amount: this.itemForm.amount })
      }
      call.then(r => {
        this.itemSaving = false
        if (r.code === 0) {
          this.itemDlg = false
          this.$message.success('已保存')
          this.detailLoading = true
          this.loadDetail(uid)
        } else this.$message.error(r.msg)
      })
    },
    weekName (d) { return ['', '周一', '周二', '周三', '周四', '周五', '周六', '周日'][d] || d },
    statusName (s) { return ['空', '生长中', '成熟'][s] || s },
    statusTag (s) { return ['info', 'warning', 'success'][s] || 'info' }
  }
}
</script>

<style scoped>
.lv { color: #409eff; font-weight: 700; margin-right: 6px; }
.lv-name { color: #97a8be; font-size: 12px; }
.rank-no {
  display: inline-block; width: 28px; height: 28px; line-height: 28px; text-align: center;
  border-radius: 50%; font-weight: 700; color: #97a8be; background: #f0f2f5;
}
.rank-no.r1 { background: linear-gradient(135deg, #f6c445, #e8a20a); color: #fff; }
.rank-no.r2 { background: linear-gradient(135deg, #c9d2dc, #a6b3c2); color: #fff; }
.rank-no.r3 { background: linear-gradient(135deg, #e09a6a, #c9793f); color: #fff; }
</style>