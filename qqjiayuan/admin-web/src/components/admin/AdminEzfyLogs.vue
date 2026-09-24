<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- ============ 出征记录 ============ -->
        <el-tab-pane label="出征记录" name="orders">
          <div class="toolbar">
            <el-input v-model="orderWord" placeholder="用户ID / 玩家昵称搜索" clearable style="width:200px"
                      @keyup.enter.native="orderPage = 1; loadOrders()" />
            <el-select v-model="orderType" style="width:130px" @change="orderPage = 1; loadOrders()">
              <el-option label="全部类型" :value="0" />
              <el-option v-for="(n, t) in typeNames" :key="t" :label="n" :value="Number(t)" />
            </el-select>
            <el-select v-model="orderStatus" style="width:130px" @change="orderPage = 1; loadOrders()">
              <el-option label="全部状态" :value="-1" />
              <el-option v-for="(n, s) in statusNames" :key="s" :label="n" :value="Number(s)" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="orderPage = 1; loadOrders()">查询</el-button>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadOrders">刷新</el-button>
          </div>
          <!-- ★ 列宽按实测内容宽度定：「耗油」原 90px 装不下 2,446,000,177（需 106px） -->
          <el-table :data="orders" v-loading="orderLoading" stripe border max-height="560">
            <el-table-column prop="id" label="订单ID" width="85" align="center" />
            <el-table-column prop="user_id" label="用户ID" width="85" align="center" />
            <el-table-column label="玩家" min-width="120" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.player_name || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="home_num" label="家园号" width="90" align="center" />
            <el-table-column label="类型" width="90" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini">{{ row.type_name }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="目标" width="100" align="center">
              <template slot-scope="{row}">{{ row.target_x }},{{ row.target_y }}</template>
            </el-table-column>
            <el-table-column label="状态" width="95" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="orderTag(row.status)">{{ statusNames[row.status] || row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="耗油" width="115" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.oil_used) }}</span></template>
            </el-table-column>
            <el-table-column label="创建时间" width="170" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="170" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-view" @click="openOrderDetail(row)">详情</el-button>
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delOrder(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ orderTotal }}</b> 条 · 每页 {{ orderSize }} 条</div>
            <el-pagination v-show="orderTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="orderTotal" :page-size="orderSize"
                           :current-page="orderPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { orderPage = p; loadOrders() }"
                           @size-change="s => { orderSize = s; orderPage = 1; loadOrders() }" />
          </div>
        </el-tab-pane>

        <!-- ============ 世界聊天 ============ -->
        <el-tab-pane label="世界聊天" name="chats">
          <div class="toolbar">
            <el-input v-model="chatWord" placeholder="玩家名 / 内容搜索" clearable style="width:220px"
                      @keyup.enter.native="chatPage = 1; loadChats()" />
            <el-button type="primary" icon="el-icon-search" @click="chatPage = 1; loadChats()">查询</el-button>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadChats">刷新</el-button>
          </div>
          <el-table :data="chats" v-loading="chatLoading" stripe border max-height="560">
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column label="玩家" width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.user_name }}</span></template>
            </el-table-column>
            <el-table-column label="内容" min-width="300" show-overflow-tooltip>
              <template slot-scope="{row}">{{ row.content }}</template>
            </el-table-column>
            <el-table-column label="时间" width="150" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delChat(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ chatTotal }}</b> 条 · 每页 {{ chatSize }} 条</div>
            <el-pagination v-show="chatTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="chatTotal" :page-size="chatSize"
                           :current-page="chatPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { chatPage = p; loadChats() }"
                           @size-change="s => { chatSize = s; chatPage = 1; loadChats() }" />
          </div>
        </el-tab-pane>

        <!-- ============ 交易所挂单 ============ -->
        <el-tab-pane label="交易所挂单" name="exchanges">
          <div class="toolbar">
            <el-input v-model="exWord" placeholder="卖家名搜索" clearable style="width:200px"
                      @keyup.enter.native="exPage = 1; loadExchanges()" />
            <el-select v-model="exStatus" style="width:130px" @change="exPage = 1; loadExchanges()">
              <el-option label="全部状态" :value="-1" />
              <el-option label="在售" :value="0" />
              <el-option label="成交" :value="1" />
              <el-option label="下架" :value="2" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="exPage = 1; loadExchanges()">查询</el-button>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadExchanges">刷新</el-button>
          </div>
          <!-- ★ 同上：数量列原 110px 装不下千分位大数 -->
          <el-table :data="exchanges" v-loading="exLoading" stripe border max-height="560">
            <el-table-column prop="id" label="挂单ID" width="90" align="center" />
            <el-table-column prop="seller_name" label="卖家" min-width="130" show-overflow-tooltip />
            <el-table-column label="资源" width="95" align="center">
              <template slot-scope="{row}">{{ row.type_name }}</template>
            </el-table-column>
            <el-table-column label="数量" width="150" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.es_count) }}</span></template>
            </el-table-column>
            <el-table-column label="总价(黄金)" width="150" align="center">
              <template slot-scope="{row}"><span class="td-gold">{{ fmtN(row.total_price) }}</span></template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 0 ? 'success' : 'info'">{{ row.status_name }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="系统单" width="90" align="center">
              <template slot-scope="{row}">{{ row.is_system === 1 ? '是' : '否' }}</template>
            </el-table-column>
            <el-table-column label="时间" width="170" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delExchange(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ exTotal }}</b> 条 · 每页 {{ exSize }} 条</div>
            <el-pagination v-show="exTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="exTotal" :page-size="exSize"
                           :current-page="exPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { exPage = p; loadExchanges() }"
                           @size-change="s => { exSize = s; exPage = 1; loadExchanges() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    <!-- 出征记录详情 -->
    <el-dialog title="出征记录详情" :visible.sync="orderDlg" width="720px" :close-on-click-modal="false">
      <div v-if="orderRow" class="od">
        <div class="od-sec">基本信息</div>
        <el-row :gutter="10">
          <el-col :span="8"><b>订单ID</b>：{{ orderRow.id }}</el-col>
          <el-col :span="8"><b>玩家</b>：{{ orderRow.player_name || '—' }}（{{ orderRow.user_id }}）</el-col>
          <el-col :span="8"><b>家园号</b>：{{ orderRow.home_num || '—' }}</el-col>
        </el-row>
        <el-row :gutter="10" style="margin-top:6px">
          <el-col :span="8"><b>命令</b>：{{ orderRow.type_name }}</el-col>
          <el-col :span="8"><b>目标</b>：{{ targetTypeName(orderRow.target_type) }} ({{ orderRow.target_x }},{{ orderRow.target_y }})</el-col>
          <el-col :span="8"><b>状态</b>：{{ statusNames[orderRow.status] || orderRow.status }}</el-col>
        </el-row>
        <el-row :gutter="10" style="margin-top:6px">
          <el-col :span="8"><b>所属城市</b>：{{ orderRow.city_id }}</el-col>
          <el-col :span="8"><b>耗油</b>：{{ fmtN(orderRow.oil_used) }}</el-col>
          <el-col :span="8"><b>宿营</b>：{{ orderRow.wait_min ? orderRow.wait_min + ' 分钟' : '无' }}</el-col>
        </el-row>

        <div class="od-sec">带队军官</div>
        <div>{{ orderRow.officer || '（无）' }}</div>

        <div class="od-sec">出征部队</div>
        <div v-if="troopList.length">
          <span v-for="(t, i) in troopList" :key="'ot' + i" class="od-tag">
            {{ t.text ? t.name : (t.name + '×' + fmtN(t.count)) }}
          </span>
        </div>
        <div v-else class="td-sub">（无）</div>

        <div class="od-sec">携带 / 待带回资源</div>
        <div>携带：{{ orderRow.res_text || resText(orderRow.resources) || '—' }}</div>
        <div>待带回：{{ orderRow.carry_text || resText(orderRow.carry) || '—' }}</div>

        <div class="od-sec">时间线</div>
        <el-row :gutter="10">
          <el-col :span="8"><b>创建</b>：{{ fmtTime(orderRow.created_at) }}</el-col>
          <el-col :span="8"><b>出发</b>：{{ fmtTime(orderRow.start_time) }}</el-col>
          <el-col :span="8"><b>到达</b>：{{ fmtTime(orderRow.arrive_time) }}</el-col>
        </el-row>
        <el-row :gutter="10" style="margin-top:6px">
          <el-col :span="8"><b>返航</b>：{{ fmtTime(orderRow.return_time) }}</el-col>
          <el-col :span="16"><b>更新</b>：{{ fmtTime(orderRow.updated_at) }}</el-col>
        </el-row>

        <div class="od-sec">结果 / 返回部队</div>
        <pre class="od-pre">{{ orderRow.result || '（无）' }}</pre>
      </div>
      <div slot="footer">
        <el-button type="primary" @click="orderDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyLogs',
  data () {
    return {
      tab: 'orders',
      typeNames: { 1: '侦查', 2: '掠夺', 3: '征服', 4: '采集', 5: '运输', 6: '增援', 7: '派遣' },
      statusNames: { 0: '行进中', 1: '驻守中', 2: '返回中', 3: '已完成', 4: '已阵亡' },
      orders: [], orderTotal: 0, orderPage: 1, orderSize: 5, orderLoading: false,
      orderWord: '', orderType: 0, orderStatus: -1,
      chats: [], chatTotal: 0, chatPage: 1, chatSize: 5, chatLoading: false, chatWord: '',
      exchanges: [], exTotal: 0, exPage: 1, exSize: 5, exLoading: false, exWord: '', exStatus: -1,
      orderDlg: false, orderRow: null
    }
  },
  mounted () { this.loadOrders() },
  computed: {
    // 出征记录详情里的部队列表
    troopList () {
      const r = this.orderRow
      if (!r) return []
      // 后端已经把部队整理成「步兵×100 卡车×50」，优先用它（跨模块不用再拉兵种表）
      if (r.troops_text) return [{ name: r.troops_text, count: 0, text: true }]
      return this.parseGroups(r.troops).map(g => ({
        name: '兵种' + g.troopId, count: g.count
      }))
    }
  },
  methods: {
    // 解析 [{"troopId":1,"count":100}] / 兼容旧格式
    parseGroups (raw) {
      if (!raw) return []
      try {
        const arr = JSON.parse(raw)
        if (!Array.isArray(arr)) return []
        return arr.map(x => ({ troopId: x.troopId || x.troop_id || 0, count: x.count || 0 }))
      } catch (e) { return [] }
    },
    targetTypeName (t) { return ({ 1: '野地', 2: '寇城', 3: '玩家城' })[t] || ('类型' + t) },
    // 资源 JSON → 「粮100 钢50」这种
    resText (raw) {
      if (!raw || raw === '{}') return '—'
      try {
        const o = JSON.parse(raw)
        const names = { food: '粮', steel: '钢', oil: '油', rare: '稀', gold: '金' }
        const parts = Object.keys(names)
          .filter(k => Number(o[k]) > 0)
          .map(k => names[k] + this.fmtN(o[k]))
        return parts.length ? parts.join(' ') : '—'
      } catch (e) { return raw }
    },
    openOrderDetail (row) {
      this.orderRow = row
      this.orderDlg = true
    },
    // 后端返回的是 RFC3339（2026-09-19T15:45:27.639+08:00），表格里直接显示又长又乱
    fmtTime (s) {
      if (!s) return '—'
      const d = new Date(s)
      if (isNaN(d.getTime())) return String(s).slice(0, 19).replace('T', ' ')
      const p = n => (n < 10 ? '0' : '') + n
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) +
        ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    orderTag (s) {
      return ({ 0: 'warning', 1: 'primary', 2: 'info', 3: 'success', 4: 'danger' })[s] || 'info'
    },
    onTab () {
      if (this.tab === 'orders' && this.orders.length === 0) this.loadOrders()
      if (this.tab === 'chats' && this.chats.length === 0) this.loadChats()
      if (this.tab === 'exchanges' && this.exchanges.length === 0) this.loadExchanges()
    },
    loadOrders () {
      this.orderLoading = true
      api.get('/admin/ezfy-orders', { params: { page: this.orderPage, size: this.orderSize, word: this.orderWord, type: this.orderType, status: this.orderStatus } }).then(r => {
        this.orderLoading = false
        if (r.code === 0) {
          this.orders = r.data.list
          this.orderTotal = r.data.total
          this.orderPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    delOrder (row) {
      this.$confirm('确认删除该条出征订单？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-orders/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadOrders() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadChats () {
      this.chatLoading = true
      api.get('/admin/ezfy-chats', { params: { page: this.chatPage, size: this.chatSize, word: this.chatWord } }).then(r => {
        this.chatLoading = false
        if (r.code === 0) {
          this.chats = r.data.list
          this.chatTotal = r.data.total
          this.chatPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    delChat (row) {
      this.$confirm('确认删除该条聊天记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-chats/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadChats() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadExchanges () {
      this.exLoading = true
      api.get('/admin/ezfy-exchanges', { params: { page: this.exPage, size: this.exSize, word: this.exWord, status: this.exStatus } }).then(r => {
        this.exLoading = false
        if (r.code === 0) {
          this.exchanges = r.data.list
          this.exTotal = r.data.total
          this.exPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    delExchange (row) {
      this.$confirm('确认删除该条交易所挂单？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-exchanges/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadExchanges() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
