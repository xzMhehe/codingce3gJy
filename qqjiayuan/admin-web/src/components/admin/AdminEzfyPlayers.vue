<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="玩家昵称 / 用户ID / 家园号搜索" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <!-- ★ 2026-10-05 本页已有「查询」按钮（点它就会重新 load），这个「刷新」按钮功能重复、容易误点 → 去掉。 -->
      </div>
      <!-- ★ 列宽按实测内容宽度定：ID 类 8 位数字需 74px+，留足余量到 100；
           昵称类设 min-width 作弹性列分摊宽屏多余宽度（只留 1 个弹性列时会被拉到 600px+）。
           注：小屏/pad 上横向滚动是既有适配行为，不靠压缩列宽去消除。 -->
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="100" align="center" />
        <el-table-column prop="home_num" label="家园号" width="100" align="center" />
        <el-table-column label="家园昵称" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.home_nick || '—' }}</template>
        </el-table-column>
        <el-table-column label="玩家昵称" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname }}</span></template>
        </el-table-column>
        <el-table-column label="阵营" width="85" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.camp === 2 ? 'danger' : 'primary'">{{ row.camp_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="军功声望" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.prestige }}</span></template>
        </el-table-column>
        <el-table-column label="军衔" width="100" align="center">
          <template slot-scope="{row}">{{ row.rank_name }}</template>
        </el-table-column>
        <el-table-column prop="city_count" label="城池" width="70" align="center" />
        <el-table-column label="更新时间" width="170" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="175" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-present" title="发放" @click="openGrant(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删号" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 详情（档案+军官+城池+背包+军团+最近出征） -->
    <el-dialog title="玩家详情" :visible.sync="detailDlg" width="940px" :close-on-click-modal="false">
      <template v-if="detail">
        <!-- 玩家头卡 -->
        <div class="player-head">
          <div class="ph-avatar">{{ (detail.player.nickname || '?').slice(0, 1) }}</div>
          <div class="ph-main">
            <div class="ph-name">
              {{ detail.player.nickname }}
              <el-tag size="mini" :type="detail.player.camp === 2 ? 'danger' : 'primary'">{{ detail.camp_name }}</el-tag>
              <el-tag size="mini" type="info" effect="plain">{{ detail.rank_name }}</el-tag>
            </div>
            <div class="ph-sub">
              家园号 <span class="td-mono">{{ detail.home_num || '—' }}</span> · 家园昵称 {{ detail.home_nick || '—' }}
            </div>
          </div>
          <div class="ph-stats">
            <div class="ph-stat"><div class="ph-num">{{ fmtNum(detail.player.prestige) }}</div><div class="ph-lab">军功声望</div></div>
            <div class="ph-stat"><div class="ph-num">{{ fmtNum(detail.player.diamond || 0) }}</div><div class="ph-lab">钻石余额</div></div>
            <div class="ph-stat"><div class="ph-num">{{ detail.cities.length }}</div><div class="ph-lab">城池</div></div>
            <div class="ph-stat"><div class="ph-num">{{ (detail.officers || []).length }}</div><div class="ph-lab">军官</div></div>
          </div>
        </div>
        <el-tabs v-model="detailTab" class="detail-tabs">
          <!-- 军官（默认展示） -->
          <el-tab-pane :label="'军官 (' + (detail.officers || []).length + ')'" name="officers">
            <el-table :data="detail.officers" size="mini" border max-height="360">
              <el-table-column prop="id" label="ID" width="70" align="center" />
              <el-table-column label="姓名" min-width="100">
                <template slot-scope="{row}">
                  <span class="td-main">{{ row.name }}</span>
                  <span v-if="row.star > 1" class="star-mark">★{{ row.star }}</span>
                </template>
              </el-table-column>
              <el-table-column label="星级" width="70" align="center">
                <template slot-scope="{row}">{{ row.star }}星</template>
              </el-table-column>
              <el-table-column label="类型" width="100" align="center">
                <template slot-scope="{row}">
                  <a v-if="row.general_name" href="javascript:;" class="td-gen" @click="showGeneral(row)">名将[原名]</a>
                  <el-tag v-else size="mini" type="info" effect="plain">{{ row.type_name }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="level" label="等级" width="60" align="center" />
              <el-table-column label="经验" width="90" align="center">
                <template slot-scope="{row}">{{ fmtNum(row.exp) }}</template>
              </el-table-column>
              <el-table-column label="军/学/后" width="130" align="center">
                <template slot-scope="{row}">
                  <span class="td-attr">{{ row.military }}/{{ row.learning }}/{{ row.logistics }}</span>
                </template>
              </el-table-column>
              <el-table-column label="忠诚" width="70" align="center">
                <template slot-scope="{row}">
                  <span :class="{ 'loyalty-low': row.loyalty <= 30 }">{{ row.loyalty }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="position_name" label="职位" width="70" align="center" />
              <el-table-column label="状态" width="80" align="center">
                <template slot-scope="{row}">
                  <el-tag size="mini" :type="statusType(row)">{{ row.status_name }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="city_name" label="所属城市" min-width="90" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <!-- 城池 -->
          <el-tab-pane :label="'城池 (' + detail.cities.length + ')'" name="cities">
            <el-table :data="detail.cities" size="mini" border max-height="360">
              <el-table-column prop="id" label="城池ID" width="80" align="center" />
              <el-table-column prop="name" label="城名" min-width="100" />
              <el-table-column label="坐标" width="90" align="center">
                <template slot-scope="{row}">{{ row.x }},{{ row.y }}</template>
              </el-table-column>
              <el-table-column prop="city_level" label="市政厅" width="80" align="center" />
              <el-table-column label="总兵力" width="90" align="center">
                <!-- ★ 2026-10-05 总兵力超过万显示「万」、超过亿显示「亿」
                     （原来用 fmtNum 原样输出，几十亿的数字根本读不出来；悬停可看完整值） -->
                <template slot-scope="{row}"><span :title="fmtNum(row.troop_total || 0)">{{ fmtWan(row.troop_total || 0) }}</span></template>
              </el-table-column>
              <el-table-column label="黄金" width="80" align="center">
                <template slot-scope="{row}"><span :title="fmtNum(row.gold)">{{ fmtWan(row.gold) }}</span></template>
              </el-table-column>
              <el-table-column label="粮食" width="80" align="center">
                <template slot-scope="{row}"><span :title="fmtNum(row.food)">{{ fmtWan(row.food) }}</span></template>
              </el-table-column>
              <el-table-column label="钢铁" width="80" align="center">
                <template slot-scope="{row}"><span :title="fmtNum(row.steel)">{{ fmtWan(row.steel) }}</span></template>
              </el-table-column>
              <el-table-column label="石油" width="80" align="center">
                <template slot-scope="{row}"><span :title="fmtNum(row.oil)">{{ fmtWan(row.oil) }}</span></template>
              </el-table-column>
              <el-table-column label="稀矿" width="80" align="center">
                <template slot-scope="{row}"><span :title="fmtNum(row.rare)">{{ fmtWan(row.rare) }}</span></template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <!-- 背包 -->
          <!-- ★ 2026-10-05 背包标题去掉「行」这个单位，只展示数量 -->
          <el-tab-pane :label="'背包 (' + detail.bag.length + ')'" name="bag">
            <el-table :data="detail.bag" size="mini" border max-height="360">
              <el-table-column prop="id" label="行ID" width="80" align="center" />
              <el-table-column prop="cfg_id" label="道具ID" width="90" align="center" />
              <el-table-column prop="item_name" label="道具名" min-width="120" />
              <el-table-column prop="count" label="数量" width="90" align="center" />
            </el-table>
          </el-tab-pane>
          <!-- 军团 -->
          <el-tab-pane :label="'军团 (' + detail.corps.length + ')'" name="corps">
            <el-table :data="detail.corps" size="mini" border max-height="360">
              <el-table-column prop="id" label="记录ID" width="90" align="center" />
              <el-table-column prop="corps_name" label="军团名" min-width="120" />
              <el-table-column prop="title" label="职位" width="110" align="center" />
              <el-table-column label="身份" width="90" align="center">
                <template slot-scope="{row}">{{ row.is_leader === 1 ? '军团长' : '成员' }}</template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <!-- 最近出征 -->
          <el-tab-pane :label="'最近出征 (' + detail.orders.length + ')'" name="orders">
            <el-table :data="detail.orders" size="mini" border max-height="360">
              <el-table-column prop="id" label="订单ID" width="90" align="center" />
              <el-table-column label="类型" width="80" align="center">
                <template slot-scope="{row}">{{ typeNames[row.order_type] || row.order_type }}</template>
              </el-table-column>
              <el-table-column label="目标" width="100" align="center">
                <template slot-scope="{row}">{{ row.target_x }},{{ row.target_y }}</template>
              </el-table-column>
              <el-table-column label="状态" width="90" align="center">
                <template slot-scope="{row}">{{ statusNames[row.status] || row.status }}</template>
              </el-table-column>
              <el-table-column label="时间" width="150" align="center">
                <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </template>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 编辑玩家（★ 2026-10-09 优化：每项显示「当前值」、保存前二次确认、昵称必填校验） -->
    <el-dialog :title="'编辑玩家 —— ' + (editCur.nickname || '')" :visible.sync="editDlg" width="500px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="玩家昵称">
          <el-input v-model="form.nickname" maxlength="20" style="width:200px" placeholder="1~20 字" />
          <div class="td-gray" style="font-size:12px">当前：{{ editCur.nickname || '—' }}</div>
        </el-form-item>
        <el-form-item label="军功声望">
          <el-input-number v-model.number="form.prestige" :min="0" :step="1000" style="width:200px" />
          <div class="td-gray" style="font-size:12px">
            当前：{{ fmtNum(editCur.prestige) }}（声望影响军衔与「可建城数」）
          </div>
        </el-form-item>
        <el-form-item label="阵营">
          <el-radio-group v-model="form.camp">
            <el-radio :label="1">同盟国</el-radio>
            <el-radio :label="2">轴心国</el-radio>
          </el-radio-group>
          <div class="td-gray" style="font-size:12px">
            当前：{{ editCur.camp === 2 ? '轴心国' : '同盟国' }}（只影响兵种显示名与阵营标识）
          </div>
        </el-form-item>
      </el-form>
      <div class="td-gray" style="font-size:12px">保存后立即生效。改动会写进玩家的二战档案。</div>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 发放 / 扣除资源（★ 2026-10-09 优化：可负数扣除、快捷填充、显示目标城与当前存量、钻石并入同一个「发放」按钮） -->
    <el-dialog :title="'发放 / 扣除资源 —— ' + grantName" :visible.sync="grantDlg" width="620px" :close-on-click-modal="false">
      <div class="grant-head">
        <div>目标城市：<b>{{ grantCity || '（无城池）' }}</b></div>
        <div class="td-gray" style="font-size:12px">
          当前：黄金 {{ fmtNum(grantCur.gold) }} · 粮食 {{ fmtNum(grantCur.food) }} ·
          钢铁 {{ fmtNum(grantCur.steel) }} · 石油 {{ fmtNum(grantCur.oil) }} · 稀矿 {{ fmtNum(grantCur.rare) }}
          <span v-if="grantDiamond"> · 钻石 {{ fmtNum(grantDiamond) }}</span>
        </div>
      </div>
      <el-form label-width="90px" style="margin-top:10px">
        <el-form-item label="快捷填充">
          <el-button v-for="p in presets" :key="'ps' + p.v" size="mini" plain @click="fillAll(p.v)">{{ p.t }}</el-button>
          <el-button size="mini" plain type="danger" @click="fillAll(0)">清零</el-button>
        </el-form-item>
        <el-form-item v-for="f in resFields" :key="f.k" :label="f.t">
          <el-input-number v-model.number="grant[f.k]" :step="step" :precision="0" style="width:200px" />
          <span class="td-gray" style="margin-left:8px">当前 {{ fmtNum(grantCur[f.k]) }}</span>
        </el-form-item>
        <el-form-item label="钻石">
          <el-input-number v-model.number="grant.diamond" :min="-9999999" :max="9999999" :step="100" style="width:200px" />
          <span class="td-gray" style="margin-left:8px">当前 {{ fmtNum(grantDiamond) }}</span>
        </el-form-item>
        <el-form-item label="步进">
          <el-radio-group v-model="step" size="mini">
            <el-radio-button :label="1000">1千</el-radio-button>
            <el-radio-button :label="10000">1万</el-radio-button>
            <el-radio-button :label="100000">10万</el-radio-button>
            <el-radio-button :label="1000000">100万</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div class="td-gray" style="font-size:12px; line-height:1.7">
        · 填<b>负数</b>即为「扣除」（例如 -10000 = 扣 1 万）；<br/>
        · 资源发放<b>不受仓储上限限制</b>（可以超上限堆着）；钻石可正可负。
      </div>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyPlayers',
  data () {
    return {
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      detailDlg: false, detail: null, detailTab: 'officers',
      editDlg: false, saving: false, editId: 0, form: {}, editCur: {},
      grantDlg: false, grantId: 0, grant: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, diamond: 0 },
      grantDiamond: 0,
      // ★ 2026-10-09 发放对话框：目标城/当前存量/快捷填充/步进
      grantName: '', grantCity: '', grantCur: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },      step: 10000,
      presets: [{ t: '1万', v: 10000 }, { t: '10万', v: 100000 }, { t: '100万', v: 1000000 }, { t: '1000万', v: 10000000 }],
      resFields: [
        { k: 'gold', t: '黄金' }, { k: 'food', t: '粮食' }, { k: 'steel', t: '钢铁' },
        { k: 'oil', t: '石油' }, { k: 'rare', t: '稀矿' }
      ],
      typeNames: { 1: '侦查', 2: '掠夺', 3: '征服', 4: '采集', 5: '运输', 6: '增援', 7: '派遣' },
      statusNames: { 0: '行进中', 1: '驻守中', 2: '返回中', 3: '已完成', 4: '已阵亡' }
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    fmtNum (n) { return n == null ? 0 : Number(n).toLocaleString() },
    // 城市资源缩写：<1万原样；≥1万 万；万单位的整数位超过5位（即≥10亿）转亿
    fmtWan (n) {
      if (n == null) n = 0
      if (n >= 1000000000) {
        const y = n / 100000000
        return (Number.isInteger(y) ? y : y.toFixed(1)) + '亿'
      }
      if (n >= 10000) {
        const w = n / 10000
        return (Number.isInteger(w) ? w : w.toFixed(1)) + '万'
      }
      return Number(n).toLocaleString()
    },
    // ★ 名将军官：点击「名将[原名]」查看原名将名字 + 获取时间 + 获取方式
    showGeneral (row) {
      const gid = row.general_id || 0
      const name = row.general_name || ('名将#' + gid)
      const star = row.general_star > 0 ? row.general_star + '星' : ''
      const lines = ['原名将：' + name + (star ? '（' + star + '）' : '')]
      if (row.get_way) lines.push('获取方式：' + row.get_way)
      if (row.get_time) lines.push('获取时间：' + this.fmtTime(row.get_time))
      this.$alert(lines.join('<br/>'), '军官「' + row.name + '」来源', { dangerouslyUseHTMLString: true, confirmButtonText: '知道了' })
    },
    statusType (row) { return row.is_captive === 1 ? 'danger' : (row.status === 1 ? 'warning' : 'success') },
    load () {
      this.loading = true
      api.get('/admin/ezfy-players', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
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
      api.get('/admin/ezfy-players/' + row.user_id + '/detail').then(r => {
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    openEdit (row) {
      this.editId = row.user_id
      this.editCur = { nickname: row.nickname, prestige: row.prestige, camp: row.camp }
      this.form = { nickname: row.nickname, prestige: row.prestige, camp: row.camp }
      this.editDlg = true
    },
    save () {
      // ★ 2026-10-09 加校验 + 二次确认（原来点了直接改，昵称填空/只空格会被后端静默忽略）
      const nick = (this.form.nickname || '').trim()
      if (!nick) { this.$message.warning('玩家昵称不能为空'); return }
      if (nick.length > 20) { this.$message.warning('玩家昵称最多 20 个字'); return }
      if (this.form.prestige == null || this.form.prestige < 0) { this.$message.warning('军功声望不能为负'); return }
      const changes = []
      if (nick !== this.editCur.nickname) changes.push('昵称：' + this.editCur.nickname + ' → ' + nick)
      if (this.form.prestige !== this.editCur.prestige) {
        changes.push('军功声望：' + this.fmtNum(this.editCur.prestige) + ' → ' + this.fmtNum(this.form.prestige))
      }
      if (this.form.camp !== this.editCur.camp) {
        changes.push('阵营：' + (this.editCur.camp === 2 ? '轴心国' : '同盟国') + ' → ' + (this.form.camp === 2 ? '轴心国' : '同盟国'))
      }
      if (!changes.length) { this.$message.info('没有改动'); return }
      this.$confirm('确认修改？\n· ' + changes.join('\n· '), '编辑玩家', { type: 'warning' }).then(() => {
        this.saving = true
        api.put('/admin/ezfy-players/' + this.editId, { nickname: nick, prestige: this.form.prestige, camp: this.form.camp }).then(r => {
          this.saving = false
          if (r.code === 0) {
            this.editDlg = false
            this.$message.success(r.data.msg || '已保存')
            this.load()
          } else this.$message.error(r.msg)
        }).catch(() => { this.saving = false })
      }).catch(() => {})
    },
    // ★ 2026-10-09 发放对话框：先把「目标城 + 当前存量 + 钻石余额」拉出来（原来只拉钻石，看不到会发到哪座城）
    openGrant (row) {
      this.grantId = row.user_id
      this.grantName = row.nickname || ('玩家' + row.user_id)
      this.grant = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, diamond: 0 }
      this.grantDiamond = 0
      this.grantCity = ''
      this.grantCur = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }
      this.grantDlg = true
      api.get('/admin/ezfy-players/' + row.user_id + '/detail').then(r => {
        if (r.code !== 0 || !r.data) return
        if (r.data.player) this.grantDiamond = r.data.player.diamond || 0
        const cs = r.data.cities || []
        if (cs.length) {
          // 后端发放落在玩家「当前所在城」（getOrCreateCity），列表里第一座通常就是它
          const c = cs[0]
          this.grantCity = c.name + '(' + c.x + ',' + c.y + ')'
          this.grantCur = { gold: c.gold || 0, food: c.food || 0, steel: c.steel || 0, oil: c.oil || 0, rare: c.rare || 0 }
        }
      })
    },
    // 快捷填充：五项资源一起填（0 = 清零）
    fillAll (v) {
      this.resFields.forEach(f => { this.grant[f.k] = v })
    },
    // ★ 2026-10-09 一次「发放」同时处理资源与钻石（原来钻石是另一个按钮，容易漏点）
    doGrant () {
      const g = this.grant
      const hasRes = g.gold !== 0 || g.food !== 0 || g.steel !== 0 || g.oil !== 0 || g.rare !== 0
      const dia = parseInt(g.diamond) || 0
      if (!hasRes && !dia) { this.$message.warning('请先填写要发放（或扣除）的数量'); return }
      const lines = []
      const push = (t, v) => { if (v !== 0) lines.push(t + (v > 0 ? '+' : '') + this.fmtNum(v)) }
      push('黄金', g.gold); push('粮食', g.food); push('钢铁', g.steel)
      push('石油', g.oil); push('稀矿', g.rare); push('钻石', dia)
      const anyNeg = [g.gold, g.food, g.steel, g.oil, g.rare, dia].some(v => v < 0)
      this.$confirm('给「' + this.grantName + '」' + (anyNeg ? '发放 / 扣除' : '发放') + '：\n· ' +
        lines.join('\n· ') + '\n（资源入玩家当前所在城，不受仓储上限限制）', '确认发放', { type: anyNeg ? 'warning' : 'info' }).then(() => {
        this.saving = true
        // 先发资源（一次请求），再发钻石（独立接口，可正可负）
        const doDiamond = () => {
          if (!dia) { this.grantDlg = false; this.$message.success('已发放'); this.load(); return }
          api.post('/admin/ezfy-players/' + this.grantId + '/diamond', { amount: dia, mode: 'add' }).then(rd => {
            this.saving = false
            if (rd.code === 0) {
              this.grantDlg = false
              this.$message.success((rd.data && rd.data.msg) || '已发放')
              this.load()
            } else this.$message.error(rd.msg || '钻石发放失败')
          }).catch(() => { this.saving = false })
        }
        if (!hasRes) { this.saving = false; doDiamond(); return }
        api.post('/admin/ezfy-players/' + this.grantId + '/grant', g).then(r => {
          this.saving = false
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已发放')
            doDiamond()
          } else this.$message.error(r.msg)
        }).catch(() => { this.saving = false })
      }).catch(() => {})
    },
    del (row) {
      this.$confirm('删除玩家「' + row.nickname + '」将同时清除城池/部队/科技/出征/背包等全部游戏数据，不可恢复！', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-players/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
/* 玩家详情头卡 */
.player-head { display: flex; align-items: center; gap: 16px; background: #f5f7fa; border: 1px solid #ebeef5; border-radius: 6px; padding: 14px 16px; margin-bottom: 14px; }
.ph-avatar { width: 52px; height: 52px; border-radius: 50%; background: linear-gradient(135deg, #409eff, #7c5cf0); color: #fff; font-size: 24px; font-weight: 600; display: flex; align-items: center; justify-content: center; flex: none; }
.ph-main { flex: 1; min-width: 0; }
.ph-name { font-size: 18px; font-weight: 600; color: #1f2d3d; display: flex; align-items: center; gap: 8px; }
.ph-sub { font-size: 12px; color: #909399; margin-top: 6px; }
.ph-stats { display: flex; gap: 28px; flex: none; }
.ph-stat { text-align: center; }
.ph-num { font-size: 18px; font-weight: 600; color: #303133; }
.ph-lab { font-size: 12px; color: #909399; margin-top: 2px; }
.star-mark { color: #e6a23c; margin-left: 4px; font-size: 12px; }
.td-gen { color: #e6a23c; cursor: pointer; text-decoration: underline; font-size: 12px; }
.td-attr { margin-right: 8px; color: #606266; font-size: 12px; }
.loyalty-low { color: #f56c6c; font-weight: 600; }
</style>
