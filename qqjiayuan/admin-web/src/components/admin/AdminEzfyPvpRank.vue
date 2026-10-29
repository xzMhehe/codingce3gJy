<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>排行榜</span>
      </div>

      <!-- ★ 2026-10-14 用户需求：「风云榜」页改名「排行榜」，下面设 风云榜 / 战力榜 两个 tab -->
      <!-- ★ 2026-10-15 用户需求：把「军衔声望榜」也放进排行榜，且同样支持不参与排名 -->
      <el-tabs v-model="tab" class="cfg-tabs" @tab-click="onTabChange">
        <el-tab-pane label="风云榜" name="pvp" />
        <el-tab-pane label="战力榜" name="power" />
        <el-tab-pane label="军衔声望榜" name="prestige" />
      </el-tabs>

      <!-- ==================== 风云榜 tab ==================== -->
      <template v-if="tab === 'pvp'">
        <div class="toolbar">
          <!-- ★ 开启/关闭：开启 = 现在这样按周统计；关闭 = 不进行统计（历史保留） -->
          <!-- ★ 2026-10-29 用户「所有冗余提示都放到 i 里面去 或者去掉，看着太乱了」：
               说明文字全部收进 i 图标的 tooltip，工具栏只留控件本身。 -->
          <span class="sw-label">风云榜统计</span>
          <el-switch v-model="enabled" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关"
                     :loading="toggling" @change="onToggle" />
          <el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:300px;line-height:1.7">
              开：按自然周（周一 00:00 ~ 周日 23:59）统计玩家对战战绩。<br>
              关：不再统计新的玩家对战战绩，历史周榜保留。
            </div>
            <i class="el-icon-info hint-i"></i>
          </el-tooltip>
          <span class="sw-div"></span>
          <!-- ★ 2026-10-29 用户「单灭敌加分上限加个开关，默认开有上限，关掉期间灭敌无上限」 -->
          <span class="sw-label">灭敌分上限</span>
          <el-switch v-model="killCapOn" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关"
                     :loading="killCapToggling" @change="onToggleKillCap" />
          <el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:320px;line-height:1.7">
              开（默认）：单场灭敌分 = 20×log10(加权灭敌)，最多 200 分。<br>
              关：灭敌分不封顶，灭敌越多分越高（仍受单场规模上限 400 / 800 约束）。<br>
              只影响之后结算的战斗，已入账积分不回算。
            </div>
            <i class="el-icon-info hint-i"></i>
          </el-tooltip>
          <div class="grow" />
          <!-- ★ 2026-10-26 锁分只锁当期：按钮固定锁「本周」，跨周自动失效 -->
          <el-button v-if="!lockWeek" type="danger" plain icon="el-icon-lock" :loading="locking"
                     title="锁分后本周风云榜终止竞争：战绩不再累计积分，榜单冻结在锁定前（下周一自动失效）"
                     @click="onLock(true)">锁分（本周）</el-button>
          <el-button v-else type="success" plain icon="el-icon-unlock" :loading="locking"
                     title="本周风云榜已冻结（不再累计积分），下周一自动失效；点此解除"
                     @click="onLock(false)">已锁分（本周），点击解锁</el-button>
          <!-- ★ 2026-10-15 风云榜查询：按游戏ID(user_id)精确查找玩家（清空 = 全量榜单） -->
          <el-input v-model="pvpWord" placeholder="游戏ID" clearable style="width:160px"
                    @keyup.enter.native="doPvpSearch()" />
          <el-button type="primary" icon="el-icon-search" @click="doPvpSearch()">查询</el-button>
          <el-select v-model="week" placeholder="选择统计周" style="width:200px" @change="loadPvp(true)">
            <el-option v-for="w in weeks" :key="w" :label="w" :value="w" />
          </el-select>
          <!-- ★ 2026-10-26 积分明细：分页 + 按游戏ID/游戏名检索 + 关联查看本场战报 -->
          <el-button type="primary" plain icon="el-icon-tickets" @click="openDetail()">积分明细</el-button>
          <!-- ★ 2026-10-18 风云榜奖励发放：给前三名 / 参与者发宝箱·道具·钻石，走系统邮件领取 -->
          <el-button type="warning" icon="el-icon-present" @click="openReward()">发放奖励</el-button>
        </div>

        <el-table :data="pvpPageList" v-loading="loading" stripe border>
          <el-table-column label="名次" width="70" align="center">
            <template slot-scope="{row}">
              <span class="td-rank" :class="row.rank <= 3 ? 'top' : ''">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="玩家" min-width="110" show-overflow-tooltip />
          <el-table-column prop="corps" label="军团" min-width="110" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.corps || '—' }}</template>
          </el-table-column>
          <el-table-column prop="points" label="积分" width="90" align="right">
            <template slot-scope="{row}"><b>{{ row.points.toLocaleString() }}</b></template>
          </el-table-column>
          <el-table-column prop="kills" label="灭敌" width="100" align="right">
            <template slot-scope="{row}">{{ row.kills.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column prop="losses" label="战损比" width="90" align="center">
            <template slot-scope="{row}">
              <el-tooltip :content="'灭敌 ' + row.kills.toLocaleString() + ' ÷ 战损 ' + row.losses.toLocaleString()" placement="top">
                <span>{{ row.losses > 0 ? Math.round((row.kills / row.losses) * 10) / 10 : (row.kills > 0 ? '∞' : '—') }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="battles" label="参战" width="80" align="center" />
          <el-table-column prop="wins" label="胜场" width="80" align="center" />
          <el-table-column label="胜率" width="80" align="center">
            <template slot-scope="{row}">{{ (Math.round(row.win_rate * 10) / 10) }}%</template>
          </el-table-column>
          <!-- ★ 操作列：不参与排名（把该玩家从榜单剔除；历史战绩保留，可恢复） -->
          <el-table-column label="操作" width="130" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-remove-outline"
                         :title="'「' + boardLabel[tab] + '」不参与排名'" @click="openExclude(row)">不参与排名</el-button>
            </template>
          </el-table-column>
        </el-table>
        <!-- ★ 2026-10-14 风云榜分页（默认每页 10 条） -->
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ pvpTotal }}</b> 名<el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:320px;line-height:1.7">
              榜单按「积分」降序；同积分按灭敌多、胜场多、战损少排前。<br>
              昵称 / 军团为写榜时的快照（改名、换军团不影响历史周）。
            </div>
            <i class="el-icon-info hint-i"></i>
          </el-tooltip></div>
          <el-pagination v-show="pvpTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="pvpTotal" :page-size="pvpSize" :current-page="pvpPage"
                         :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { pvpPage = p }"
                         @size-change="s => { pvpSize = s; pvpPage = 1 }" />
        </div>

        <!-- ★ 2026-10-18 「本期已发放奖励」：该周已发放的记录（谁、什么档位、发了什么奖励、什么时候），
             分页 + 局部刷新（点分页只刷这一块），随周切换自动刷新 -->
        <div class="panel-title">本期已发放奖励（共 <b>{{ rewardRecordTotal }}</b> 条）</div>
        <el-table :data="rewardRecords" v-loading="rewardLoading" stripe border size="mini">
          <el-table-column label="名次" width="80" align="center">
            <template slot-scope="{row}">{{ row.rank_no > 3 ? '参与' : row.rank_no }}</template>
          </el-table-column>
          <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.name || ('ID ' + row.user_id) }}</template>
          </el-table-column>
          <el-table-column prop="tier_name" label="档位" width="100" align="center">
            <template slot-scope="{row}">
              <el-tag size="mini" :type="row.tier === 'join' ? 'info' : 'warning'">{{ row.tier_name }}</el-tag>
            </template>
          </el-table-column>
          <!-- ★ 2026-10-18 发放明细：钻石/宝箱[名]×N、道具[名]×N，一眼看出发了什么 -->
          <el-table-column label="发放明细" min-width="260" show-overflow-tooltip>
            <template slot-scope="{row}">
              <span v-if="row.detail" class="rw-detail">{{ row.detail }}</span>
              <span v-else class="td-gray">—</span>
            </template>
          </el-table-column>
          <el-table-column label="发放时间" width="170" align="center">
            <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ rewardRecordTotal }}</b> 条 · 每页 {{ rewardRecordSize }} 条</div>
          <el-pagination v-show="rewardRecordTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="rewardRecordTotal" :page-size="rewardRecordSize" :current-page="rewardRecordPage"
                         :page-sizes="[10, 20, 50]"
                         @current-change="p => { rewardRecordPage = p; loadRewardRecords() }"
                         @size-change="s => { rewardRecordSize = s; rewardRecordPage = 1; loadRewardRecords() }" />
        </div>
      </template>

      <!-- ==================== 战力榜 tab ==================== -->
      <template v-if="tab === 'power'">
        <div class="toolbar">
          <el-input v-model="powerWord" placeholder="昵称 / 用户ID" clearable style="width:220px"
                    @keyup.enter.native="powerPage = 1; loadPower()" />
          <el-button type="primary" icon="el-icon-search" @click="powerPage = 1; loadPower()">查询</el-button>
          <div class="grow" />
          <el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:320px;line-height:1.7">
              战力 = 科技 + 建筑 + 兵种（权重在「二战系统配置」可调），与游戏内战力榜同口径。
            </div>
            <i class="el-icon-info hint-i"></i>
          </el-tooltip>
        </div>
        <el-table :data="powerList" v-loading="powerLoading" stripe border>
          <el-table-column label="名次" width="70" align="center">
            <template slot-scope="{row}">
              <span class="td-rank" :class="row.rank <= 3 ? 'top' : ''">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
          <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.name || '—' }}</template>
          </el-table-column>
          <el-table-column prop="city_name" label="主城" min-width="110" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.city_name || '—' }}</template>
          </el-table-column>
          <el-table-column label="战力" width="110" align="right">
            <template slot-scope="{row}"><b>{{ row.power.toLocaleString() }}</b></template>
          </el-table-column>
          <el-table-column label="科技战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.tech_power.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="建筑战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.build_power.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="兵种战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.troop_power.toLocaleString() }}</template>
          </el-table-column>
          <!-- ★ 2026-10-15 战力榜「不参与排名」；★ 2026-10-16 改用**战力榜自己那一份**名单（排除后重新排名） -->
          <el-table-column label="操作" width="130" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-remove-outline"
                         :title="'「' + boardLabel[tab] + '」不参与排名'" @click="openExclude(row)">不参与排名</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ powerTotal }}</b> 条 · 每页 {{ powerSize }} 条</div>
          <el-pagination v-show="powerTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="powerTotal" :page-size="powerSize" :current-page="powerPage"
                         :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { powerPage = p; loadPower() }"
                         @size-change="s => { powerSize = s; powerPage = 1; loadPower() }" />
        </div>
      </template>

      <!-- ==================== 军衔声望榜 tab ==================== -->
      <template v-if="tab === 'prestige'">
        <div class="toolbar">
          <el-input v-model="prestigeWord" placeholder="昵称 / 用户ID" clearable style="width:220px"
                    @keyup.enter.native="prestigePage = 1; loadPrestige()" />
          <el-button type="primary" icon="el-icon-search" @click="prestigePage = 1; loadPrestige()">查询</el-button>
          <div class="grow" />
          <el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:320px;line-height:1.7">
              与游戏内「军衔声望榜」同口径：按声望降序；名次为剔除「不参与排名」后的全服排名。
            </div>
            <i class="el-icon-info hint-i"></i>
          </el-tooltip>
        </div>
        <el-table :data="prestigeList" v-loading="prestigeLoading" stripe border>
          <el-table-column label="名次" width="70" align="center">
            <template slot-scope="{row}">
              <span class="td-rank" :class="row.rank <= 3 ? 'top' : ''">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
          <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.name || '—' }}</template>
          </el-table-column>
          <el-table-column label="军衔" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">
              <el-tag size="mini" type="warning">{{ row.rank_name }}</el-tag>
              <span class="td-sub" v-if="row.rank_post">({{ row.rank_post }})</span>
            </template>
          </el-table-column>
          <el-table-column label="声望" width="120" align="right">
            <template slot-scope="{row}"><b>{{ row.prestige.toLocaleString() }}</b></template>
          </el-table-column>
          <el-table-column label="可建城" width="80" align="center">
            <template slot-scope="{row}">{{ row.city_max }}</template>
          </el-table-column>
          <el-table-column label="已建城" width="80" align="center">
            <template slot-scope="{row}">{{ row.city_count }}</template>
          </el-table-column>
          <!-- ★ 2026-10-15 军衔声望榜「不参与排名」；★ 2026-10-16 改用**本榜自己那一份**名单 -->
          <el-table-column label="操作" width="130" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-remove-outline"
                         :title="'「' + boardLabel[tab] + '」不参与排名'" @click="openExclude(row)">不参与排名</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ prestigeTotal }}</b> 条 · 每页 {{ prestigeSize }} 条</div>
          <el-pagination v-show="prestigeTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="prestigeTotal" :page-size="prestigeSize" :current-page="prestigePage"
                         :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { prestigePage = p; loadPrestige() }"
                         @size-change="s => { prestigeSize = s; prestigePage = 1; loadPrestige() }" />
        </div>
      </template>

      <!-- ★ 2026-10-16 不参与排名名单：**三个榜各自独立**（切换 tab 看/改的是该榜自己的名单，
           在风云榜标记不会影响战力榜、军衔声望榜，反之亦然） -->
      <div class="panel-title">「{{ boardLabel[tab] }}」不参与排名（{{ excludedNow.length }} 人）</div>
      <el-table :data="excludedNow" v-loading="excludedLoading" stripe border size="mini">
        <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
        <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.name || '—' }}</template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.reason || '—' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="加入时间" width="160" align="center" />
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="success" plain icon="el-icon-refresh-left"
                       :title="'恢复参与「' + boardLabel[tab] + '」排名'" @click="removeExclude(row)">恢复排名</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- ★ 2026-10-26 风云榜「积分明细」：分页看全服每一场的入账明细（0 分也记），
         可按游戏ID / 游戏名检索，并能关联查看本场战报 -->
    <el-dialog title="风云榜积分明细" :visible.sync="detail.on" width="980px" append-to-body>
      <div class="toolbar" style="margin-bottom:8px">
        <el-select v-model="detail.week" style="width:180px" @change="loadDetail(1)">
          <el-option v-for="w in weeks" :key="'dt' + w" :label="w" :value="w" />
        </el-select>
        <el-input v-model="detail.word" placeholder="游戏ID 或 游戏名" clearable style="width:200px;margin-left:8px"
                  @keyup.enter.native="loadDetail(1)" @clear="loadDetail(1)" />
        <el-button type="primary" icon="el-icon-search" style="margin-left:8px" @click="loadDetail(1)">查询</el-button>
        <div class="grow" />
        <el-tooltip placement="top" effect="light">
          <div slot="content" style="max-width:320px;line-height:1.7">
            每场玩家对战、攻守双方各一条（0 分也记）；点行尾「战报」看本场完整战报。
          </div>
          <i class="el-icon-info hint-i"></i>
        </el-tooltip>
      </div>
      <el-table :data="detail.list" v-loading="detail.loading" size="mini" border stripe max-height="440">
        <el-table-column label="时间" width="110" align="center">
          <template slot-scope="{row}">{{ row.ts_fmt }}</template>
        </el-table-column>
        <el-table-column label="玩家" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.name || '—' }}<span class="td-sub">（ID {{ row.user_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column label="对手" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.foe_nick || '—' }}<span class="td-sub" v-if="row.foe_id">（ID {{ row.foe_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="70" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.is_win ? 'success' : (row.is_draw ? 'info' : 'danger')">
              {{ row.is_win ? '胜' : (row.is_draw ? '平' : '负') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="灭敌" width="100" align="right">
          <template slot-scope="{row}">{{ (row.kills || 0).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="战损" width="100" align="right">
          <template slot-scope="{row}">{{ (row.losses || 0).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="积分" width="90" align="right">
          <template slot-scope="{row}"><b>{{ row.points }}</b></template>
        </el-table-column>
        <el-table-column label="战报" width="88" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" plain icon="el-icon-document" @click="openDetailReport(row)">战报</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ detail.total }}</b> 条 · 每页 {{ detail.size }} 条</div>
        <el-pagination v-show="detail.total > 0" small background layout="sizes, prev, pager, next, jumper"
                       :total="detail.total" :page-size="detail.size" :current-page="detail.page"
                       :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { detail.page = p; loadDetail() }"
                       @size-change="s => { detail.size = s; detail.page = 1; loadDetail() }" />
      </div>
      <div slot="footer"><el-button @click="detail.on = false">关 闭</el-button></div>
    </el-dialog>

    <!-- ★ 2026-10-26 本场战报（从积分明细点开；战报被清理时明确提示） -->
    <el-dialog :title="detailReport.title || '本场战报'" :visible.sync="detailReport.on" width="720px" append-to-body>
      <div v-loading="detailReport.loading" style="min-height:80px">
        <pre v-if="detailReport.content" class="report-pre">{{ detailReport.content }}</pre>
        <pre v-if="detailReport.detail" class="report-pre report-pre-detail">{{ detailReport.detail }}</pre>
        <div v-if="!detailReport.loading && !detailReport.content && !detailReport.detail" class="td-gray">
          {{ detailReport.err || '（战报内容为空）' }}
        </div>
      </div>
      <div slot="footer"><el-button @click="detailReport.on = false">关 闭</el-button></div>
    </el-dialog>

    <!-- ★★ 2026-10-18 风云榜奖励发放（用户需求）：
         给第 1/2/3 名和参与者分别发 宝箱 / 道具 / 钻石；奖励**不直接进背包**，
         以系统邮件发给玩家，玩家在邮箱点 [领取] 才到账，超过领取期限作废。
         同一玩家同一周只发一次（后端 ezfy_pvp_reward 唯一索引），重复点只补发没发过的人。
         ★ 2026-10-18 用户「提示冗余了，用 i 然后鼠标滑动展示那种，发放页面要简约」：
           大段说明收进标题旁的 i 图标 tooltip，四档压成单行。 -->
    <el-dialog title="风云榜奖励发放" :visible.sync="reward.on" width="880px" :close-on-click-modal="false"
               append-to-body>
      <el-form label-width="84px" size="small" class="rw-form">
        <el-form-item label="统计周">
          <el-select v-model="reward.week" style="width:190px" @change="loadRewardStatus()">
            <el-option v-for="w in weeks" :key="'rw' + w" :label="w" :value="w" />
          </el-select>
          <span class="td-sub" style="margin-left:10px">
            榜单 {{ pvpTotal }} 人，有效参战 {{ rewardEligible }} 人<span v-if="reward.status">，已发放 {{ reward.status.sent }} 人</span>
          </span>
          <el-tooltip placement="top" effect="light">
            <div slot="content" style="max-width:320px;line-height:1.7">
              奖励不直接进背包，以系统邮件发送，玩家在「邮箱」点 [领取] 才到账；超过领取期限未领取自动作废。<br>
              当周积分 0 视为「没有效参与」，不发奖（前三名也一样）；可设最低积分门槛进一步过滤。<br>
              同一玩家同一周只发一次，重复点「确认发放」只会补发还没发过的人。
            </div>
            <i class="el-icon-info rw-info"></i>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="领取期限">
          <el-input-number v-model="reward.expire_days" :min="1" :max="90" size="small"
                           controls-position="right" style="width:96px" />
          <span class="td-sub" style="margin-left:6px">天未领作废</span>
          <span class="rw-sep">|</span>
          <span class="td-sub">最低积分</span>
          <el-input-number v-model="reward.min_points" :min="0" :max="100000000" size="small"
                           controls-position="right" style="width:110px;margin-left:4px" />
          <span class="td-sub" style="margin-left:6px">分（低于此分不发奖，0 = 只要有积分就发）</span>
        </el-form-item>
        <el-form-item label="参与奖">
          <el-radio-group v-model="reward.joinAll">
            <el-radio :label="false">第 4 ~ 第 {{ reward.join_limit }} 名</el-radio>
            <el-radio :label="true">全部参战玩家</el-radio>
          </el-radio-group>
          <el-input-number v-if="!reward.joinAll" v-model="reward.join_limit" :min="4" :max="1000"
                           size="small" controls-position="right" style="width:88px;margin-left:4px" />
        </el-form-item>
        <!-- ★ 2026-10-18 补发：发过一次还能再给参与者补发（已发过的人也会再发一封，玩家邮件多一封可领的奖励）
             ★ 2026-10-18 展示优化：开启补发后下面弹琥珀色警示条，一眼看清当前是补发模式 -->
        <el-form-item label="补发模式">
          <el-switch v-model="reward.resend" active-text="补发" inactive-text="正常发放" />
          <span class="td-sub" style="margin-left:6px">
            {{ reward.resend ? '已发过的人也会再发一次（用于给参与者补发宝箱）' : '默认只发没发过的人（同一玩家同一周只发一次）' }}
          </span>
        </el-form-item>
        <div v-if="reward.resend" class="rw-warn">
          <i class="el-icon-warning-outline"></i>
          <span><b>补发模式已开启</b>：符合条件的玩家（含已发过的人）会<b>再收到一封</b>对应档位的新邮件、可再领一次；
            仅用于给参与者补发遗漏的宝箱 / 道具，请确认后再点「确认发放」。</span>
        </div>
      </el-form>

      <!-- 四档 → 真表格：档位 / 钻石 / 宝箱 / 道具 四列（table-layout:fixed + colgroup 保证表头与内容严格对齐） -->
      <table class="rw-tb">
        <colgroup>
          <col style="width:76px" />
          <col style="width:132px" />
          <col />
          <col />
        </colgroup>
        <thead>
          <tr><th>档位</th><th>钻石</th><th>宝箱</th><th>道具</th></tr>
        </thead>
        <tbody>
          <tr v-for="t in reward.tiers" :key="'rt' + t.key">
            <td class="rw-name">{{ t.label }}</td>

            <td>
              <el-input-number v-model="t.diamond" :min="0" :max="100000000" size="mini"
                               controls-position="right" style="width:112px" />
            </td>

            <td>
              <!-- ★ 2026-10-18 修复「宝箱清空了不能再加了」：v-if 只在最后一行显示加号，
                   删空数组后没有行 → 加号也跟着消失 → 永远加不回来。
                   改成：数组为空时也单独给一个加号行。 -->
              <div class="rw-line" v-for="(c, i) in t.chests" :key="'c' + i">
                <el-select v-model="c.chest_id" size="mini" filterable placeholder="选择宝箱" style="flex:1;min-width:96px">
                  <el-option v-for="o in chestOptions" :key="'co' + o.id" :label="o.name" :value="o.id" />
                </el-select>
                <!-- ★ 2026-10-18 数量优化：加 × 前缀、输入框加大（用户「宝箱、道具数量还是小呢」） -->
                <span class="rw-x">×</span>
                <el-input-number v-model="c.count" :min="1" :max="100000" size="small"
                                 controls-position="right" style="width:112px;flex:0 0 112px" />
                <i class="el-icon-close rw-del" title="移除" @click="t.chests.splice(i, 1)"></i>
                <i v-if="i === t.chests.length - 1" class="el-icon-plus rw-add" title="再加一个宝箱"
                   @click="t.chests.push({ chest_id: null, count: 1 })"></i>
              </div>
              <div v-if="t.chests.length === 0" class="rw-line">
                <span class="rw-empty">（无）</span>
                <i class="el-icon-plus rw-add" title="加一个宝箱"
                   @click="t.chests.push({ chest_id: null, count: 1 })"></i>
              </div>
            </td>

            <td>
              <div class="rw-line" v-for="(it, i) in t.items" :key="'i' + i">
                <el-select v-model="it.cfg_id" size="mini" filterable placeholder="选择道具" style="flex:1;min-width:110px">
                  <el-option v-for="o in itemOptions" :key="'io' + o.id" :label="o.name" :value="o.id" />
                </el-select>
                <!-- ★ 2026-10-18 数量优化：加 × 前缀、输入框加大 -->
                <span class="rw-x">×</span>
                <el-input-number v-model="it.count" :min="1" :max="100000" size="small"
                                 controls-position="right" style="width:112px;flex:0 0 112px" />
                <i class="el-icon-close rw-del" title="移除" @click="t.items.splice(i, 1)"></i>
                <i v-if="i === t.items.length - 1" class="el-icon-plus rw-add" title="再加一个道具"
                   @click="t.items.push({ cfg_id: null, count: 1 })"></i>
              </div>
              <div v-if="t.items.length === 0" class="rw-line">
                <span class="rw-empty">（无）</span>
                <i class="el-icon-plus rw-add" title="加一个道具"
                   @click="t.items.push({ cfg_id: null, count: 1 })"></i>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div slot="footer">
        <el-button @click="reward.on = false">取消</el-button>
        <el-button type="warning" :loading="reward.saving" @click="doReward">确认发放</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-14 排行榜（原「风云榜」）：
//   风云榜 tab —— 每周玩家对战战绩周榜（可开关统计、可把玩家设为「不参与排名」）
//   战力榜 tab —— 复用游戏内战力口径（科技 + 建筑 + 兵种），管理端全量分页查看
// ★ 2026-10-15 用户：「排行榜 刷新按钮去掉；风云榜里加个查询按钮能根据游戏ID查询玩家」
//   —— 去掉头部 [刷新]（进页面/切周自动拉数据）；风云榜 tab 加「游戏ID」输入 + [查询]
//      （后端一次性下发整周全量榜单，查询走前端精确过滤，清空回到全量）。
// ★ 2026-10-15 用户：把「军衔声望榜」也放进排行榜，且同样支持不参与排名
//   —— 新增 军衔声望榜 tab（按声望降序、展示当前军衔，口径与玩家端一致）。
// ★ 2026-10-16 用户：「风云榜和战力榜、军衔声望榜 都可以单独不参与排名，现在公用一个不对」
//   —— 排除名单拆成**三份互相独立**（后端 ezfy_pvp_exclude.board = pvp/power/prestige）；
//      页面下方「不参与排名」列表**只展示当前 tab 那一份**，标记/恢复也只影响该榜。
export default {
  name: 'AdminEzfyPvpRank',
  data () {
    return {
      tab: 'pvp',
      // ★ 2026-10-16 榜单中文名（tab 名即 board 名：pvp/power/prestige）
      boardLabel: { pvp: '风云榜', power: '战力榜', prestige: '军衔声望榜' },
      // ---- 风云榜 ----
      loading: false,
      toggling: false,
      locking: false, // ★ 2026-10-18 锁分按钮 loading
      enabled: 1,
      // ★ 2026-10-29 灭敌分上限开关（1 = 有上限 200 分 / 0 = 无上限）
      killCapOn: 1,
      killCapToggling: false,
      lockWeek: '', // ★ 2026-10-18 锁分：当前被锁定的周（空 = 未锁定）
      week: '',
      weeks: [],
      list: [],
      // ★ 2026-10-18 「本期已发放奖励」明细（风云榜页下方展示，随周切换刷新；分页局部刷新）
      rewardRecords: [],
      rewardLoading: false,
      rewardRecordTotal: 0,
      rewardRecordPage: 1,
      rewardRecordSize: 10,
      // ★ 2026-10-14 风云榜分页（默认每页 10 条）——后端一次性返回整周榜单，前端切片
      pvpPage: 1,
      pvpSize: 10,
      pvpWord: '', // ★ 2026-10-15 风云榜查询：按游戏ID(user_id)精确查找玩家
      // ★ 2026-10-26 积分明细（分页 + 按游戏ID/游戏名检索 + 关联本场战报）
      detail: { on: false, loading: false, week: '', word: '', list: [], total: 0, page: 1, size: 20 },
      detailReport: { on: false, loading: false, title: '', content: '', detail: '', err: '' },
      // ★ 2026-10-16 三榜各自的「不参与排名」名单（互不影响）
      excluded: { pvp: [], power: [], prestige: [] },
      // ---- 战力榜 ----
      powerLoading: false,
      powerPage: 1,
      powerSize: 10,
      powerTotal: 0,
      powerWord: '',
      powerList: [],
      // ---- 军衔声望榜（★ 2026-10-15 与玩家端口径一致，支持不参与排名）----
      prestigeLoading: false,
      prestigePage: 1,
      prestigeSize: 10,
      prestigeTotal: 0,
      prestigeWord: '',
      prestigeList: [],
      // ---- ★ 2026-10-18 风云榜奖励发放 ----
      reward: {
        on: false,
        week: '',
        expire_days: 7,
        min_points: 0,
        join_limit: 20,
        joinAll: false, // true = 第 4 名及以后全部参战玩家
        saving: false,
        status: null,
        resend: false, // ★ 2026-10-18 补发：true = 已发过的人也会再发一封（用于给参与者补发宝箱）
        // 每档默认各带一行空宝箱 / 空道具 —— 弹窗一打开就是「一行一档」，最紧凑
        // （空行在 doReward 里按 chest_id/cfg_id 为空自动跳过）
        tiers: [
          { key: 'rank1', label: '第 1 名', diamond: 0, chests: [{ chest_id: null, count: 1 }], items: [{ cfg_id: null, count: 1 }] },
          { key: 'rank2', label: '第 2 名', diamond: 0, chests: [{ chest_id: null, count: 1 }], items: [{ cfg_id: null, count: 1 }] },
          { key: 'rank3', label: '第 3 名', diamond: 0, chests: [{ chest_id: null, count: 1 }], items: [{ cfg_id: null, count: 1 }] },
          { key: 'join', label: '参与奖', diamond: 0, chests: [{ chest_id: null, count: 1 }], items: [{ cfg_id: null, count: 1 }] }
        ]
      },
      chestOptions: [],
      itemOptions: [],
    }
  },
  mounted () {
    this.loadPvp()
  },
  computed: {
    // ★ 2026-10-16 当前 tab 对应的榜单名（= 后端 board）
    board () { return this.tab },
    // ★ 2026-10-16 页面下方只展示「当前 tab」那一份排除名单
    excludedNow () { return this.excluded[this.tab] || [] },
    // ★ 2026-10-16 排除名单表格的 loading 跟随当前 tab（原来固定用风云榜的 loading）
    excludedLoading () {
      if (this.tab === 'power') return this.powerLoading
      if (this.tab === 'prestige') return this.prestigeLoading
      return this.loading
    },
    // ★ 2026-10-14 风云榜分页：后端一次性返回整周榜单，这里按页切片展示
    // ★ 2026-10-15 风云榜查询：填了游戏ID 就只展示该玩家（查不到 = 空），清空回到全量榜单
    pvpShown () {
      if (!this.pvpWord) return this.list
      const uid = Number(this.pvpWord)
      if (isNaN(uid)) return []
      return this.list.filter(r => r.user_id === uid)
    },
    // ★ 2026-10-18 有效参战人数 = 当周积分 > 0 且 ≥ 最低积分门槛（随门槛改动实时变化）
    rewardEligible () {
      const st = this.reward.status
      if (!st || !st.list) return this.pvpTotal
      const min = Number(this.reward.min_points) || 0
      return st.list.filter(x => x.points > 0 && x.points >= min).length
    },
    // ★ 2026-10-15 修复「排行榜空白」：pvpShown 是计算属性，取值用 this.pvpShown，不能当函数调用
    pvpTotal () { return this.pvpShown.length },
    pvpPageList () {
      const start = (this.pvpPage - 1) * this.pvpSize
      return this.pvpShown.slice(start, start + this.pvpSize)
    }
  },
  methods: {
    // ---- ★ 2026-10-18 风云榜奖励发放 ----
    openReward () {
      this.reward.on = true
      this.reward.week = this.week || this.weeks[0] || ''
      this.reward.resend = false // ★ 2026-10-18 每次打开默认「正常发放」，避免误按成补发
      if (!this.chestOptions.length) {
        api.get('/admin/ezfy-chest-grant/options').then(r => {
          if (r.code === 0) this.chestOptions = r.data.list || []
        })
      }
      if (!this.itemOptions.length) {
        api.get('/admin/ezfy-item-grant/options').then(r => {
          if (r.code === 0) this.itemOptions = r.data.list || []
        })
      }
      this.loadRewardStatus()
    },
    // 该周已发放情况（弹窗里显示「已发放 N 人」，避免以为没发又点一次）
    loadRewardStatus () {
      api.get('/admin/ezfy-pvp-rank/reward', { params: { week: this.reward.week } }).then(r => {
        if (r.code === 0) this.reward.status = r.data
      })
    },
    doReward () {
      const tiers = {}
      let any = false
      for (const t of this.reward.tiers) {
        const o = { diamond: t.diamond || 0, chests: [], items: [] }
        for (const c of t.chests) {
          if (c.chest_id) { o.chests.push({ chest_id: c.chest_id, count: c.count || 1 }); any = true }
        }
        for (const it of t.items) {
          if (it.cfg_id) { o.items.push({ cfg_id: it.cfg_id, count: it.count || 1 }); any = true }
        }
        if (o.diamond > 0) any = true
        tiers[t.key] = o
      }
      if (!any) { this.$message.warning('请至少给一个档位配置奖励（宝箱 / 道具 / 钻石）'); return }
      // ★ 2026-10-18 补发模式提示：管理员能清楚知道自己是在补发（已发过的也会再收到）
      const tip = this.reward.resend
        ? '按【补发】给「' + this.reward.week + '」风云榜补发奖励：已发过的玩家也会再收到一封新邮件（用于给参与者补发宝箱等）。确定继续？'
        : '确定给「' + this.reward.week + '」风云榜发放奖励吗？发放后不可撤回；同一玩家同一周只会发一次。'
      this.$confirm(tip, '确认发放', { type: 'warning' }).then(() => {
        this.reward.saving = true
        api.post('/admin/ezfy-pvp-rank/reward', {
          week: this.reward.week,
          expire_days: this.reward.expire_days,
          min_points: this.reward.min_points || 0,
          join_limit: this.reward.joinAll ? 0 : this.reward.join_limit,
          resend: this.reward.resend || false, // ★ 2026-10-18 补发：true = 已发过的人也会再发一封
          tiers: tiers
        }).then(r => {
          this.reward.saving = false
          if (r.code === 0) {
            this.$message.success((r.data && r.data.msg) || '已发放')
            this.loadRewardStatus()
          } else this.$message.error(r.msg || '发放失败')
        }).catch(() => { this.reward.saving = false })
      }).catch(() => {})
    },
    // ★ 2026-10-15 风云榜查询：按游戏ID精确查找玩家（查不到提示，便于核对输入）
    doPvpSearch () {
      this.pvpPage = 1
      if (!this.pvpWord) return
      const uid = Number(this.pvpWord)
      if (isNaN(uid)) {
        this.$message.warning('请输入数字游戏ID')
        return
      }
      if (!this.list.some(r => r.user_id === uid)) {
        this.$message.info('该周榜单中没有此玩家')
      }
    },
    onTabChange () {
      if (this.tab === 'power' && !this.powerList.length) this.loadPower()
      // ★ 2026-10-15 军衔声望榜 tab：首次进入或数据被清空时加载
      if (this.tab === 'prestige' && !this.prestigeList.length) this.loadPrestige()
    },
    loadPvp (resetPage) {
      this.loading = true
      // ★ 2026-10-14 切周（resetPage）后回到第 1 页；单纯刷新保留当前页
      if (resetPage === true) this.pvpPage = 1
      const params = {}
      if (this.week) params.week = this.week
      api.get('/admin/ezfy-pvp-rank', { params }).then(r => {
        if (r.code === 0) {
          this.weeks = r.data.weeks || []
          this.week = r.data.week
          this.list = r.data.ranks || []
          // ★ 2026-10-16 三榜名单独立：这里拿到的是「风云榜」那一份
          this.$set(this.excluded, 'pvp', r.data.excluded || [])
          // 后端 1 = 开 / 0 = 关
          this.enabled = r.data.enabled ? 1 : 0
          // ★ 2026-10-29 灭敌分上限开关（缺字段 = 老后端 → 按默认「有上限」）
          this.killCapOn = r.data.kill_cap_on === false || r.data.kill_cap_on === 0 ? 0 : 1
          // ★ 2026-10-18 锁分状态：锁定周（空 = 未锁定）
          this.lockWeek = r.data.lock_week || ''
          if (this.weeks.indexOf(this.week) < 0 && this.week) this.weeks.unshift(this.week)
          this.loadRewardRecords()
        } else {
          // ★ 2026-10-15 防「空白页」：接口异常时明确提示，而不是静默一片空白
          this.$message.error(r.msg || '加载排行榜失败')
          this.list = []
          this.$set(this.excluded, 'pvp', [])
        }
      }).catch(() => {
        this.$message.error('加载排行榜失败，请检查后端服务')
        this.list = []
        this.$set(this.excluded, 'pvp', [])
      }).finally(() => { this.loading = false })
    },
    onToggle (val) {
      this.toggling = true
      api.post('/admin/ezfy-pvp-rank/toggle', { on: val }).then(r => {
        if (r.code === 0) this.$message.success(r.data.msg || '已保存')
        else {
          this.$message.error(r.msg)
          this.enabled = val ? 0 : 1 // 失败回滚开关显示
        }
      }).catch(() => { this.enabled = val ? 0 : 1 })
        .finally(() => { this.toggling = false })
    },
    // ★ 2026-10-29 灭敌分上限开关：开 = 单场灭敌分最多 200；关 = 无上限（只影响之后的结算）
    onToggleKillCap (val) {
      this.killCapToggling = true
      api.post('/admin/ezfy-pvp-rank/kill-cap', { on: val }).then(r => {
        if (r.code === 0) this.$message.success(r.data.msg || '已保存')
        else {
          this.$message.error(r.msg)
          this.killCapOn = val ? 0 : 1 // 失败回滚开关显示
        }
      }).catch(() => { this.killCapOn = val ? 0 : 1 })
        .finally(() => { this.killCapToggling = false })
    },
    // ★ 2026-10-26 锁分 / 解锁：只锁**当期（本周）**，跨周自动失效
    onLock (lock) {
      if (lock) {
        this.$confirm('锁分后本周风云榜终止竞争：战绩不再累计积分，榜单冻结在锁定前（下周一自动失效）。确定锁分？',
          '风云榜锁分', { type: 'warning' }).then(() => {
          this.doLock(1, '')
        }).catch(() => {})
      } else {
        this.doLock(0, '')
      }
    },
    doLock (lock, week) {
      this.locking = true
      api.post('/admin/ezfy-pvp-rank/lock', { week, lock }).then(r => {
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已保存')
          this.lockWeek = lock ? (r.data.week || '本周') : ''
        } else this.$message.error(r.msg)
      }).catch(() => {}).finally(() => { this.locking = false })
    },
    // ★ 2026-10-26 积分明细：分页 + 检索（游戏ID 精确 / 游戏名模糊）
    openDetail () {
      this.detail.on = true
      this.detail.week = this.week || this.weeks[0] || ''
      this.detail.page = 1
      this.loadDetail()
    },
    loadDetail (resetPage) {
      if (resetPage === 1) this.detail.page = 1
      this.detail.loading = true
      const params = { week: this.detail.week, page: this.detail.page, size: this.detail.size }
      if (this.detail.word) params.word = this.detail.word
      api.get('/admin/ezfy-pvp-detail', { params }).then(r => {
        if (r.code === 0) {
          this.detail.list = r.data.list || []
          this.detail.total = r.data.total || 0
        } else this.$message.error(r.msg || '加载积分明细失败')
      }).catch(() => { this.detail.list = []; this.detail.total = 0 })
        .finally(() => { this.detail.loading = false })
    },
    // 关联查看本场战报（战报被定期清理时后端会明确提示）
    openDetailReport (row) {
      this.detailReport = { on: true, loading: true, title: '本场战报', content: '', detail: '', err: '' }
      api.get('/admin/ezfy-pvp-detail/report', { params: { id: row.id } }).then(r => {
        if (r.code === 0) {
          this.detailReport.title = r.data.title || '本场战报'
          this.detailReport.content = r.data.content || ''
          this.detailReport.detail = r.data.detail || ''
        } else {
          this.detailReport.err = r.msg || '本场战报已被清理'
        }
      }).catch(() => { this.detailReport.err = '加载战报失败' })
        .finally(() => { this.detailReport.loading = false })
    },
    // ★ 2026-10-18 「本期已发放奖励」明细（随查看的周切换自动刷新；分页局部刷新，只刷这一块）
    loadRewardRecords () {
      this.rewardLoading = true
      api.get('/admin/ezfy-pvp-rank/reward/records', {
        params: { week: this.week, page: this.rewardRecordPage, size: this.rewardRecordSize }
      }).then(r => {
        if (r.code === 0) {
          this.rewardRecords = r.data.list || []
          this.rewardRecordTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      }).catch(() => { this.rewardRecords = [] })
        .finally(() => { this.rewardLoading = false })
    },
    // 时间格式化（created_at 为 ISO 字符串 → YYYY-MM-DD HH:mm）
    fmtTime (v) {
      if (!v) return '—'
      const d = new Date(v)
      if (isNaN(d.getTime())) return String(v)
      const p = n => String(n).padStart(2, '0')
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) +
        ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    },
    // ★ 2026-10-16 标记「不参与排名」：带上当前 tab 对应的榜单（board），
    //   只影响该榜；成功后只刷新当前 tab 的数据（不再三榜齐刷）。
    openExclude (row) {
      const bn = this.boardLabel[this.tab] || '榜单'
      this.$prompt('把「' + (row.name || row.user_id) + '」设为「' + bn + '」不参与排名？可填原因（选填）。',
        bn + ' · 不参与排名', { inputPlaceholder: '原因，例如：刷分 / 违规', inputValue: '' })
        .then(({ value }) => {
          api.post('/admin/ezfy-pvp-rank/exclude', { user_id: row.user_id, reason: value || '', board: this.tab }).then(r => {
            if (r.code === 0) {
              this.$message.success(r.data.msg || '已设置')
              this.reloadCurrent()
            } else this.$message.error(r.msg)
          })
        }).catch(() => {})
    },
    removeExclude (row) {
      const bn = this.boardLabel[this.tab] || '榜单'
      this.$confirm('恢复「' + (row.name || row.user_id) + '」参与「' + bn + '」排名？', '提示', { type: 'warning' }).then(() => {
        // 走排除记录的 id（不是 user_id）—— 接口是按记录主键删的（一条记录只属于一个榜）
        api.delete('/admin/ezfy-pvp-rank/exclude/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已恢复')
            this.reloadCurrent()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-10-16 只刷新当前 tab（含它自己那份排除名单）
    reloadCurrent () {
      if (this.tab === 'power') this.loadPower()
      else if (this.tab === 'prestige') this.loadPrestige()
      else this.loadPvp()
    },
    loadPower () {
      this.powerLoading = true
      const params = { page: this.powerPage, size: this.powerSize }
      if (this.powerWord) params.word = this.powerWord
      api.get('/admin/ezfy-power-rank', { params }).then(r => {
        if (r.code === 0) {
          this.powerList = r.data.list || []
          this.powerTotal = r.data.total || 0
          this.$set(this.excluded, 'power', r.data.excluded || [])
        }
      }).finally(() => { this.powerLoading = false })
    },
    // ★ 2026-10-15 军衔声望榜（新 tab）：按声望降序，与玩家端口径一致；
    //   管理端剔除「不参与排名」名单后重排，操作列复用 openExclude（★ 2026-10-16 用本榜自己的名单）。
    loadPrestige () {
      this.prestigeLoading = true
      const params = { page: this.prestigePage, size: this.prestigeSize }
      if (this.prestigeWord) params.word = this.prestigeWord
      api.get('/admin/ezfy-prestige-rank', { params }).then(r => {
        if (r.code === 0) {
          this.prestigeList = r.data.list || []
          this.prestigeTotal = r.data.total || 0
          this.$set(this.excluded, 'prestige', r.data.excluded || [])
        }
      }).finally(() => { this.prestigeLoading = false })
    }
  }
}
</script>

<style>
  /* ★ 2026-10-14 补齐公共样式（card-head / toolbar / grow / pager-bar / pager-info）——
     之前本文件没 import 也没重定义 `.card-head`，header 里的「排行榜」标题和「刷新」按钮
     因为没有 flex 布局而贴在一起（用户反馈）。 */
  @import './farm-admin.css';
  .td-rank { font-weight: 600; color: #909399; }
  .td-rank.top { color: #e6a23c; }
  .cfg-tabs { margin-bottom: 6px; }
  .sw-label { font-size: 13px; color: #606266; margin-right: 6px; }
  /* ★ 2026-10-29 冗余说明统一收进 i 图标 tooltip（原来一大段灰字挤在工具栏里） */
  .hint-i { color: #a8b3c2; cursor: help; font-size: 14px; vertical-align: -1px; }
  .hint-i:hover { color: #409eff; }
  /* 两个开关之间的细分隔线（只是视觉分组，不占空间） */
  .sw-div { width: 1px; height: 16px; background: #e4e8ee; margin: 0 2px; }
  .panel-title { font-size: 14px; font-weight: 600; color: #303133; margin: 16px 0 8px; }
  /* ★ 2026-10-18 风云榜奖励发放弹窗：真表格（table-layout:fixed，表头与内容严格对齐） */
  .rw-form .el-form-item { margin-bottom: 8px; }
  .rw-info { margin-left: 6px; color: #909399; cursor: help; font-size: 14px; vertical-align: -1px; }
  .rw-sep { color: #dcdfe6; margin: 0 10px; }
  .rw-tb { width: 100%; border-collapse: collapse; table-layout: fixed; border: 1px solid #ebeef5; border-radius: 4px; }
  .rw-tb th {
    padding: 7px 10px; background: #f5f7fa; font-size: 12px; font-weight: 400;
    color: #909399; text-align: left; border-bottom: 1px solid #ebeef5;
  }
  .rw-tb td { padding: 8px 10px; vertical-align: top; border-bottom: 1px solid #ebeef5; }
  .rw-tb tbody tr:last-child td { border-bottom: 0; }
  .rw-tb tbody tr:nth-child(even) td { background: #fafbfc; }
  .rw-tb td.rw-name { font-size: 13px; font-weight: 600; color: #303133; line-height: 28px; }
  .rw-line { display: flex; align-items: center; gap: 5px; }
  .rw-line + .rw-line { margin-top: 5px; }
  .rw-del, .rw-add { cursor: pointer; font-size: 13px; flex: 0 0 auto; }
  .rw-del { color: #c0c4cc; }
  .rw-del:hover { color: #f56c6c; }
  .rw-add { color: #c0c4cc; }
  .rw-add:hover { color: #409eff; }
  .rw-empty { color: #c0c4cc; font-size: 13px; line-height: 28px; }
  /* ★ 2026-10-18 数量优化：× 前缀 + 更大输入框 */
  .rw-x { color: #606266; font-size: 14px; font-weight: 600; flex: 0 0 auto; line-height: 28px; }
  /* ★ 2026-10-18 补发模式琥珀色警示条 */
  .rw-warn {
    display: flex; align-items: flex-start; gap: 6px;
    margin: 2px 0 12px; padding: 8px 12px; line-height: 1.6;
    font-size: 13px; color: #7a5c00;
    background: #fdf6ec; border: 1px solid #f5dab1; border-radius: 4px;
  }
  .rw-warn i { font-size: 16px; margin-top: 2px; }
  /* ★ 2026-10-18 发放明细：奖励摘要显眼 */
  .rw-detail { color: #b8860b; font-size: 13px; }
  /* ★ 2026-10-26 积分明细 → 本场战报：等宽字体保留战报原始换行 */
  .report-pre {
    margin: 0; padding: 10px 12px; max-height: 420px; overflow: auto;
    font-family: Menlo, Consolas, monospace; font-size: 12.5px; line-height: 1.7;
    color: #303133; background: #fafbfc; border: 1px solid #ebeef5; border-radius: 4px;
    white-space: pre-wrap; word-break: break-all;
  }
  .report-pre + .report-pre { margin-top: 10px; }
  .report-pre-detail { background: #f8f9fb; color: #606266; }
</style>
