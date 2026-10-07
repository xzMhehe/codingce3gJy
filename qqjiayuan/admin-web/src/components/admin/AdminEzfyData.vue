<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <!-- ★ 数据表切换：下拉框改为 Tab（，切换更直观） -->
      <el-tabs v-model="table" class="cfg-tabs" @tab-click="onTabChange">
        <el-tab-pane v-for="t in tables" :key="t.k" :label="t.n" :name="t.k" />
      </el-tabs>
      <div class="toolbar">
        <!-- ★ 2026-09-27 装备道具配置默认只展示「用户商城在售」的装备，可切换查看全部 -->
        <el-switch v-if="table === 'equipments'" v-model="mallOnly" active-text="仅看商城上架"
                   @change="page = 1; load()" />
        <el-tooltip v-if="table === 'equipments'" placement="top">
          <div slot="content" style="max-width:340px;line-height:1.6">
            展示到用户商城需同时满足：<br>
            ① 有价格（黄金或钻石 > 0）；<br>
            ② 类型为「军官装备」；<br>
            ③ 非第一批套装件（set_id&gt;0 且无系列号的只能开宝箱，不直购）
          </div>
          <i class="el-icon-question" style="margin-left:6px;color:#909399;cursor:pointer" />
        </el-tooltip>
        <el-input v-model="word" :placeholder="searchPh" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <!-- ★ 2026-10-07 道具配置：按「商城分类」筛选。
             分类在库里大多是空的（真实分类是算出来的）→ 选项由后端按同一口径算好回传，前端不再写第二套规则。 -->
        <el-select v-if="table === 'items'" v-model="cat" clearable placeholder="全部分类" style="width:170px"
                   @change="page = 1; load()">
          <el-option v-for="c in cats" :key="c" :label="c" :value="c" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <!-- ★ 2026-09-28 钻石流水 / 2026-10-02 道具使用 是只读视图，隐藏「新增」按钮 -->
        <el-button v-if="!isReadOnly" type="success" icon="el-icon-plus" @click="openCreate">新增</el-button>
        <!-- ★ 道具配置专属：发放道具（按玩家昵称/游戏ID搜索目标，道具入背包），2026-09-26 从玩家信息管理移到这里 -->
        <el-button v-if="table === 'items'" type="warning" plain icon="el-icon-present" @click="openItemGrant">发放道具</el-button>
      </div>
      <el-table :data="rows" v-loading="loading" stripe border max-height="620">
        <el-table-column v-for="col in cols" :key="col.k" :label="col.n" :width="col.w" :min-width="col.minW"
                         :align="(col.w || col.minW) ? 'center' : 'left'" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="col.fmt === 'stock'" :class="row[col.k] < 0 ? 'unlimited' : ''">
              {{ row[col.k] < 0 ? '无上限' : row[col.k] }}
            </span>
            <!-- ★ 状态 / 枚举列：1、0 这种编码玩家看不懂，统一渲染成文字标签 -->
            <el-tag v-else-if="col.dict" size="mini" :type="dictOf(col, row[col.k]).t">
              {{ dictOf(col, row[col.k]).n }}
            </el-tag>
            <!-- ★ 时间列：库里是毫秒时间戳，格式化成日期时间 -->
            <span v-else-if="col.fmt === 'time'">{{ fmtTime(row[col.k]) }}</span>
            <!-- ★ 道具参数列：param1 的含义随类型变化 → 按类型渲染成「含义 + 数值 + 单位」 -->
            <span v-else-if="col.fmt === 'param'">{{ paramText(row) }}</span>
            <!-- ★ 宝箱价格列：黄金价/钻石价合并显示（宝箱只用其中一种货币，分两列总有一列是 0） -->
            <span v-else-if="col.fmt === 'price'">{{ priceText(row) }}</span>
            <span v-else>{{ fmt(row[col.k]) }}</span>
          </template>
        </el-table-column>
        <!-- ★ 操作按钮统一成图标按钮（与其他二战页面一致，原为「编辑/删除」文字按钮）；
             ★ 2026-09-27 套装装备配置（宝箱）专属：加「奖池」按钮，开箱奖池在弹窗里维护（宝箱是套装装备唯一产出渠道）；
             ★ 2026-09-28 钻石流水 / 2026-10-02 道具使用 是只读视图，不渲染操作列 -->
        <el-table-column v-if="!isReadOnly" label="操作" width="160" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button v-if="table === 'chests'" size="mini" type="success" plain icon="el-icon-s-grid" title="配置奖池" @click="openChestPool(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="doDelete(row)" />
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

      <!-- 新增/编辑对话框 -->
      <el-dialog :title="formId ? '编辑' + tableName : '新增' + tableName" :visible.sync="showForm" width="600px" append-to-body>
        <el-form label-width="140px" size="small">
          <el-form-item v-for="f in formFields" :key="f.k" :label="f.n" :required="f.req">
            <el-select v-if="f.opts" v-model="form[f.k]" style="width:240px"
                       :filterable="!!f.filterable" :allow-create="!!f.allowCreate" default-first-option>
              <el-option v-for="o in f.opts" :key="o.v" :label="o.n" :value="o.v" />
            </el-select>
            <el-input-number v-else-if="f.t === 'num'" v-model="form[f.k]"
                             :min="f.min === undefined ? 0 : f.min" style="width:180px" />
            <!-- ★ 时间字段用日期选择器（value-format=timestamp → 表单值仍是毫秒时间戳，后端不用改） -->
            <el-date-picker v-else-if="f.t === 'time'" v-model="form[f.k]" type="datetime"
                            placeholder="选择日期时间" value-format="timestamp"
                            :picker-options="{ firstDayOfWeek: 1 }" style="width:240px" />
            <el-input v-else-if="f.t === 'text'" v-model="form[f.k]" type="textarea" :rows="2" />
            <el-input v-else v-model="form[f.k]" :maxlength="f.max || 50" style="width:320px" />
          </el-form-item>
        </el-form>
        <!-- [说明·不显示在界面] 提示：城池为玩家运行数据，修改后玩家下次进入游戏生效 -->
        <div slot="footer">
          <el-button size="small" @click="showForm = false">取 消</el-button>
          <el-button size="small" type="primary" :loading="saving" @click="doSave">保 存</el-button>
        </div>
      </el-dialog>

      <!-- ★ 发放道具：目标玩家按昵称/游戏ID搜索选择，道具入背包，2026-09-26 从玩家信息管理移来 -->
      <el-dialog title="发放道具" :visible.sync="grantDlg" width="580px" :close-on-click-modal="false">
        <el-form label-width="90px" size="small">
          <el-form-item label="发放对象">
            <div>
              <el-input v-model="grantPlayer" placeholder="玩家昵称 / 游戏ID" clearable style="width:210px"
                        @keyup.enter.native="searchGrantPlayer" />
              <el-button size="small" type="primary" plain icon="el-icon-search" @click="searchGrantPlayer">搜索</el-button>
            </div>
            <el-select v-if="grantPlayers.length" v-model="grantUserId" placeholder="在搜索结果中选择玩家"
                       style="margin-top:6px;width:100%"
                       :filterable="grantPlayers.length > 1" default-first-option>
              <el-option v-for="p in grantPlayers" :key="p.user_id"
                         :label="p.nickname + '（ID:' + p.user_id + (p.home_num ? ' / 家园:' + p.home_num : '') + '）'"
                         :value="p.user_id" />
            </el-select>
            <div v-if="grantUserId" class="grant-target">已选：{{ grantTargetName }}</div>
          </el-form-item>
          <el-form-item label="道具">
            <div v-for="(it, i) in grantItems" :key="i" class="grant-item-row">
              <el-select v-model="it.cfg_id" filterable placeholder="选择道具" style="width:230px">
                <el-option v-for="o in grantItemOpts" :key="o.id" :label="o.id + ' — ' + o.name" :value="o.id" />
              </el-select>
              <span class="grant-x">×</span>
              <el-input-number v-model.number="it.count" :min="1" :max="9999" controls-position="right" style="width:110px" />
              <el-button type="text" class="danger-btn" @click="grantItems.splice(i, 1)">删除</el-button>
            </div>
            <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="grantItems.push({ cfg_id: '', count: 1 })">添加道具</el-button>
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button size="small" @click="grantDlg = false">取 消</el-button>
          <el-button size="small" type="primary" :loading="grantSaving" @click="doItemGrant">发 放</el-button>
        </div>
      </el-dialog>

      <!-- ★ 套装装备配置 → 宝箱奖池：开箱按奖池权重随机出装备/道具（2026-09-27 从「军官管理 → 宝箱」迁来） -->
      <el-dialog :title="'奖池 · ' + chPoolChest.name" :visible.sync="chPoolDlg" width="1000px">
        <div class="toolbar">
          <span class="td-sub">共 {{ chPool.length }} 条 · 权重合计 {{ chWeightSum }}（每条的「概率」= 权重 ÷ 合计）</span>
          <div class="grow" />
          <el-select v-model.number="chBulkSetId" filterable clearable placeholder="按套装批量加入" style="width:240px">
            <el-option v-for="s in equipSets" :key="'bs' + s.id" :label="s.id + ' · ' + s.name" :value="s.id" />
          </el-select>
          <el-input-number v-model.number="chBulkWeight" :min="1" controls-position="right" style="width:110px" />
          <el-button type="success" plain icon="el-icon-plus" :loading="saving" @click="doChestBulkAdd">批量加入</el-button>
          <el-button type="primary" icon="el-icon-plus" @click="openChestItemCreate">加一条</el-button>
          <el-button plain icon="el-icon-refresh" @click="loadChestPool">刷新</el-button>
        </div>
        <el-table :data="chPool" size="mini" border stripe max-height="460">
          <el-table-column prop="id" label="ID" width="60" align="center" />
          <el-table-column prop="kind_name" label="类型" width="70" align="center" />
          <!-- ★ 奖品列：主行显示名称，ID 只作小字后缀（原来把「（cfg_id 3001）」摆在正文里，全是数字看不懂） -->
          <el-table-column prop="name" label="奖品" min-width="200" show-overflow-tooltip>
            <template slot-scope="{row}">
              <span class="td-main">{{ row.name || '（奖品已不存在）' }}</span>
              <span class="td-sub">#{{ row.ref_id }}</span>
            </template>
          </el-table-column>
          <!-- ★ 品质：按档位给颜色（口径 = 装备品质 普通/稀有/史诗/传说） -->
          <el-table-column prop="quality" label="品质" width="80" align="center">
            <template slot-scope="{row}">
              <el-tag v-if="row.quality" size="mini" :type="qualityTagOf(row.quality)">{{ row.quality }}</el-tag>
              <span v-else class="td-sub">—</span>
            </template>
          </el-table-column>
          <el-table-column prop="count" label="数量" width="60" align="center" />
          <el-table-column prop="weight" label="权重" width="70" align="center" />
          <el-table-column label="概率" width="80" align="center">
            <template slot-scope="{row}">{{ row.rate }}%</template>
          </el-table-column>
          <el-table-column label="操作" width="130" align="center">
            <template slot-scope="{row}">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openChestItemEdit(row)" />
              <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delChestItem(row)" />
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-info" style="margin-top:8px">
          奖品直接按名称搜索选择（装备/道具/套装都在下拉里），不用再去别处查 ID。
        </div>
      </el-dialog>

      <!-- ★ 新增/编辑 奖池条目 -->
      <!-- ★ 2026-10-07 改造：奖品不再让运营手填数字 cfg_id（得先去别的页面查 ID），
           改成按名称搜索的下拉（装备/道具/套装分别取各自的配置列表），品质也改成标准档位下拉。 -->
      <el-dialog :title="chif.id ? '编辑奖池条目' : '新增奖池条目'" :visible.sync="chItemDlg"
                 width="620px" :close-on-click-modal="false">
        <el-form label-width="110px" size="small">
          <el-form-item label="奖品类型">
            <el-radio-group v-model.number="chif.kind">
              <el-radio :label="1">单件装备</el-radio>
              <el-radio :label="2">道具</el-radio>
              <el-radio :label="3">整套装备</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item :label="chestKindLabel[chif.kind] || '奖品'" required>
            <el-select v-if="chif.kind === 3" v-model.number="chif.ref_id" filterable clearable
                       placeholder="搜索套装名 / ID" style="width:380px">
              <el-option v-for="s in equipSets" :key="'es' + s.id" :label="s.id + ' · ' + s.name" :value="s.id" />
            </el-select>
            <el-select v-else-if="chif.kind === 2" v-model.number="chif.ref_id" filterable clearable
                       placeholder="搜索道具名 / ID" style="width:380px">
              <el-option v-for="o in itemOpts" :key="'it' + o.id" :label="o.label" :value="o.id" />
            </el-select>
            <el-select v-else v-model.number="chif.ref_id" filterable clearable
                       placeholder="搜索装备名 / ID" style="width:380px">
              <el-option v-for="o in equipOpts" :key="'eq' + o.id" :label="o.label" :value="o.id" />
            </el-select>
          </el-form-item>
          <el-row :gutter="10">
            <el-col :span="8">
              <el-form-item label="数量">
                <el-input-number v-model.number="chif.count" :min="1" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="权重">
                <el-input-number v-model.number="chif.weight" :min="0" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <!-- ★ 品质改成标准档位下拉（与装备品质口径一致），也允许自定义输入 -->
              <el-form-item label="品质标签">
                <el-select v-model="chif.quality" filterable allow-create clearable
                           placeholder="选择或输入" style="width:100%">
                  <el-option v-for="q in qualityOpts" :key="q" :label="q" :value="q" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="备注">
            <el-input v-model="chif.des" maxlength="200" />
          </el-form-item>
        </el-form>
        <!-- [说明·不显示在界面] 权重越大越容易抽到；全部为 0 时按等概率。 -->
        <div slot="footer">
          <el-button @click="chItemDlg = false">取 消</el-button>
          <el-button type="primary" :loading="saving" @click="doChestItemSave">保 存</el-button>
        </div>
      </el-dialog>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

// ★ 枚举列的展示口径：库里存的是 1 / 0 这种编码，直接显示出来没人看得懂。
//   统一在这里转成文字标签（n = 文字，t = el-tag 颜色，缺省为默认蓝）。
//   口径与下方 FORMS 里的下拉选项保持一致，改一处即可。
const DICTS = {
  // 任务类型 / 任务配置的 status
  onoff: {
    1: { n: '启用', t: 'success' },
    0: { n: '停用', t: 'info' }
  },
  // 节日活动的 status
  actStatus: {
    1: { n: '开启', t: 'success' },
    0: { n: '关闭', t: 'info' }
  },
  // 节日活动类型
  actType: {
    1: { n: '资源增产' }, 2: { n: '造兵打折' }, 3: { n: '建造加速' },
    4: { n: '研究加速' }, 5: { n: '声望加成' }
  },
  // 任务类型重置方式
  resetType: {
    0: { n: '一次性', t: 'info' }, 1: { n: '每日', t: 'success' }, 2: { n: '每周', t: 'warning' }
  },
  // ★ 2026-09-27 装备品质（口径与军官装备管理一致：1灰/2蓝/3紫/4橙）
  tier: {
    1: { n: '初级', t: 'info' }, 2: { n: '中级', t: 'primary' },
    3: { n: '高级', t: 'warning' }, 4: { n: '特殊', t: 'danger' }
  },
  // 宝箱上架状态
  chestOn: {
    1: { n: '上架', t: 'success' }, 0: { n: '下架', t: 'info' }
  },
  // 道具类型
  // ★ 2026-10-07 补全到 30（原来只映射到 16，17~30 在列表里**直接显示数字**，
  //   用户反馈「分类还有数字」就是这里）。
  itemType: {
    1: { n: '资源包' }, 2: { n: '黄金包' },
    3: { n: '建筑加速' }, 4: { n: '训练加速' }, 5: { n: '科技加速' },
    6: { n: '建筑图纸' }, 7: { n: '增产' }, 8: { n: '免战' },
    9: { n: '招生简章' }, 10: { n: '经验书' }, 11: { n: '军官技能书' }, 12: { n: '重修书' },
    13: { n: '改名卡' }, 14: { n: '阵营转换道具' }, 15: { n: '出征道具' }, 16: { n: '迁城道具' },
    17: { n: '高级迁城道具' }, 18: { n: '沿海迁城道具' },
    19: { n: '星级徽章' }, 20: { n: '计谋道具' }, 21: { n: '军官改名卡' },
    22: { n: '为爱发电卡' }, 23: { n: '为爱发电高级卡' },
    24: { n: '建筑加速%' }, 25: { n: '训练加速%' }, 26: { n: '科技加速%' },
    27: { n: '粮食包' }, 28: { n: '钢铁包' }, 29: { n: '石油包' }, 30: { n: '稀矿包' }
  },
  // 任务行为（库里的 task_type 是 build_upgrade 这种英文代码，列表直接显示没人看得懂）
  taskAction: {
    build_upgrade: { n: '升级建筑' },
    train_troop: { n: '训练部队' },
    tech_research: { n: '研究科技' },
    occupy_wild: { n: '占领野地' },
    battle_wild: { n: '攻打野地' },
    battle_kou: { n: '攻打寇城' },
    kill_enemy: { n: '消灭敌军' },
    city_level: { n: '市政厅等级' },
    army_count: { n: '总兵力' },
    wild_count: { n: '野地数量' },
    has_city: { n: '拥有首个城池' }
  }
}

// 任务行为下拉选项（与上面 taskAction 同一份文案，避免两处口径不一致）
const TASK_ACTIONS = Object.keys(DICTS.taskAction).map(k => ({ v: k, n: DICTS.taskAction[k].n }))
// 道具类型下拉选项（同理，与 DICTS.itemType 同源）
const ITEM_TYPES = Object.keys(DICTS.itemType).map(k => ({ v: Number(k), n: DICTS.itemType[k].n }))

// ★ 2026-10-07 道具的「参数(param1)」含义**随类型变化**（资源量 / 分钟数 / 百分比 / 小时数 / 经验值…），
//   列表里原来只写「参数」两个字，运营根本不知道这个数字是什么 → 按类型渲染成「含义 + 数值 + 单位」。
//   label = 编辑表单里该字段的标签（跟着所选类型变），text(v) = 列表单元格文案。
//   ★ 口径来自后端结算代码（ezfy.go 的 `switch cfg.ItemType`），改玩法时要一起看。
const PARAM_META = {
  1: { label: '四资源各增加的数量', text: v => '四资源各 +' + v },
  2: { label: '黄金增加的数量', text: v => '黄金 +' + v },
  27: { label: '粮食增加的数量', text: v => '粮食 +' + v },
  28: { label: '钢铁增加的数量', text: v => '钢铁 +' + v },
  29: { label: '石油增加的数量', text: v => '石油 +' + v },
  30: { label: '稀矿增加的数量', text: v => '稀矿 +' + v },
  3: { label: '加速的分钟数', text: v => '加速 ' + v + ' 分钟' },
  4: { label: '加速的分钟数', text: v => '加速 ' + v + ' 分钟' },
  5: { label: '加速的分钟数', text: v => '加速 ' + v + ' 分钟' },
  24: { label: '加速比例（%）', text: v => '剩余时间 -' + v + '%' },
  25: { label: '加速比例（%）', text: v => '剩余时间 -' + v + '%' },
  26: { label: '加速比例（%）', text: v => '剩余时间 -' + v + '%' },
  7: { label: '产量提升比例（%）', text: v => '产量 +' + v + '%（24 小时）' },
  8: { label: '免战小时数', text: v => '免战 ' + v + ' 小时' },
  9: { label: '不使用（固定刷新军校候选）', text: () => '—（刷新军校候选）' },
  10: { label: '获得的经验值', text: v => '经验 +' + v },
  11: { label: '不使用（固定免费学 1 个技能）', text: () => '—（免费学 1 个技能）' },
  12: { label: '不使用（重置军官属性点）', text: () => '—（重置属性点）' },
  13: { label: '不使用', text: () => '—' },
  14: { label: '不使用', text: () => '—' },
  15: { label: '每个集结令提升的出征上限', text: v => '每个 +' + v + ' 出征上限' },
  16: { label: '不使用（迁城在市政厅操作）', text: () => '—（在市政厅使用）' },
  17: { label: '不使用（迁城在市政厅操作）', text: () => '—（在市政厅使用）' },
  18: { label: '不使用（迁城在市政厅操作）', text: () => '—（在市政厅使用）' },
  19: { label: '不使用（升星概率/加成走管理端升星配置）', text: () => '—（走升星配置）' },
  20: { label: '不使用（发动计谋固定扣 1 个）', text: () => '—（发动计谋消耗）' },
  21: { label: '不使用（在军官详情页使用）', text: () => '—（在军官详情使用）' },
  22: { label: '每天可领的钻石数', text: v => '每天 +' + v + ' 钻石（共 30 天）' },
  23: { label: '每天可领的钻石数', text: v => '每天 +' + v + ' 钻石（共 30 天）' }
}

// ★ 2026-10-07 商城分类下拉（顺序/文案与后端 ezfyCategoryOrder 对齐）
//   ★「钻石道具 / 黄金道具」不只是分类，还会**锁定支付货币**（钻石道具只能用钻石买，反之亦然）。
const ITEM_CATEGORY_OPTS = [
  { v: '', n: '（留空 = 按类型自动归类）' },
  { v: '资源道具', n: '资源道具' }, { v: '加速道具', n: '加速道具' },
  { v: '增益道具', n: '增益道具' }, { v: '建筑图纸', n: '建筑图纸' },
  { v: '军官道具', n: '军官道具' }, { v: '计谋道具', n: '计谋道具' },
  { v: '身份道具', n: '身份道具' }, { v: '出征道具', n: '出征道具' },
  { v: '迁城道具', n: '迁城道具' },
  { v: '钻石道具', n: '钻石道具（锁定：只能用钻石买）' },
  { v: '黄金道具', n: '黄金道具（锁定：只能用黄金买）' },
  { v: '其他', n: '其他' }
]
// ★ 宝箱奖池条目可改字段（走专用接口 /admin/ezfy-chests/:id/pool，宝箱是套装装备唯一产出渠道）
const CI_KEYS = ['kind', 'ref_id', 'count', 'weight', 'quality', 'des']

// ★ 2026-10-07 奖池条目：kind 1/2/3 对应「奖品」这一栏该填什么（表单标签跟着类型变）
const CHEST_KIND_LABEL = { 1: '单件装备', 2: '道具', 3: '整套装备' }
// ★ 品质档位（口径与装备品质一致；库里现用值就是这四档）——允许自定义输入，这里只做常用档位提示
const QUALITY_OPTS = ['普通', '稀有', '史诗', '传说']
// 品质标签配色（列表里按档位给颜色）
const QUALITY_TAG = { 普通: 'info', 稀有: 'primary', 史诗: 'warning', 传说: 'danger' }

// 各数据表的展示列（k=字段, n=列名, w=列宽, dict=枚举文字映射）
const COLS = {
  activities: [
    { k: 'id', n: 'ID', w: 60 },
    { k: 'name', n: '活动名', w: 150 },
    // ★ 原为 type_name / status_txt —— 表里根本没有这两列，列表一直显示「—」，
    //   改成真实字段 type / status + 文字映射
    { k: 'type', n: '类型', w: 110, dict: 'actType' },
    { k: 'param', n: '参数', w: 90 },
    // ★ 库里存的是毫秒时间戳（如 1758652800000），直接显示没人看得懂 → 格式化成日期时间
    { k: 'start_time', n: '开始时间', w: 160, fmt: 'time' },
    { k: 'end_time', n: '结束时间', w: 160, fmt: 'time' },
    { k: 'status', n: '状态', w: 90, dict: 'actStatus' },
    { k: 'des', n: '说明' }
  ],
  buildings: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '建筑名', w: 110 }, { k: 'type', n: '类型', w: 70 },
    { k: 'max_level', n: '最高等级', w: 90 }, { k: 'unique_flag', n: '唯一', w: 60 },
    { k: 'can_delete', n: '可拆除', w: 80 }, { k: 'pre_building', n: '前置建筑' }, { k: 'des', n: '描述' }
  ],
  buildingLevels: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'building_id', n: '建筑ID', w: 80 }, { k: 'level', n: '等级', w: 70 },
    { k: 'pop', n: '人口', w: 70 }, { k: 'food', n: '粮食', w: 90 }, { k: 'steel', n: '钢铁', w: 90 },
    { k: 'oil', n: '石油', w: 90 }, { k: 'rare', n: '稀矿', w: 90 }, { k: 'gold', n: '黄金', w: 100 },
    { k: 'build_time', n: '耗时(秒)', w: 90 }, { k: 'capacity', n: '容量', w: 90 }, { k: 'effect', n: '效果' }
  ],
  troops: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '兵种名', w: 110 }, { k: 'type', n: '兵种类型', w: 90 },
    { k: 'health', n: '生命', w: 70 }, { k: 'atk_ground', n: '对地', w: 70 }, { k: 'atk_sea', n: '对海', w: 70 },
    { k: 'atk_air', n: '对空', w: 70 }, { k: 'defence', n: '防御', w: 70 }, { k: 'speed', n: '速度', w: 70 },
    { k: 'carry', n: '载量', w: 70 }, { k: 'pop', n: '占用人口', w: 90 }, { k: 'train_time', n: '训练(秒)', w: 90 }
  ],
  techs: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '科技名', w: 130 }, { k: 'type', n: '类型', w: 70 },
    { k: 'max_level', n: '最高等级', w: 90 }, { k: 'pre_building', n: '需科研中心', w: 100 },
    { k: 'pre_tech', n: '前置科技', w: 90 }, { k: 'pre_tech_level', n: '前置等级', w: 90 }, { k: 'effect', n: '效果' }
  ],
  techLevels: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'tech_id', n: '科技ID', w: 80 }, { k: 'level', n: '等级', w: 70 },
    { k: 'food', n: '粮食', w: 90 }, { k: 'steel', n: '钢铁', w: 90 }, { k: 'oil', n: '石油', w: 90 },
    { k: 'rare', n: '稀矿', w: 90 }, { k: 'gold', n: '黄金', w: 100 }, { k: 'research_time', n: '耗时(秒)', w: 90 }
  ],
  wildlands: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'type', n: '类型', w: 90 }, { k: 'level', n: '等级', w: 70 },
    { k: 'res_min', n: '资源下限', w: 100 }, { k: 'res_max', n: '资源上限', w: 100 },
    { k: 'officer_min', n: '军官下限', w: 90 }, { k: 'officer_max', n: '军官上限', w: 90 },
    { k: 'troops', n: '守军' }, { k: 'des', n: '描述' }
  ],
  // ★ 2026-10-07 道具配置列重排（用户反馈「管理的模糊、分类还有数字、很乱」）：
  //   ① 类型走 itemType 字典（补全到 30，不再出现数字）；
  //   ② 分类列显示后端算好的**生效分类**（category_name，口径 = 用户端商城），
  //      原列名「分类/货币」把「商城分类」和「计价货币」两个概念混在一起；
  //   ③ 参数列按类型渲染含义（PARAM_META），不再是一个看不懂的裸数字；
  //   ④ 去掉一直是空的「图标」列（icon 全库为空、用户端也不渲染），字段仍在编辑表单里可改；
  //   ⑤ 道具名 + 描述 两个弹性列（el-table 只把多余宽度分给 min-width 列，
  //      只留 1 个弹性列时它会被拉到 600px+，这就是「某列莫名很宽」的根因）。
  items: [
    // ★ 列宽按「列宽体检」（.workbuddy-ai/scripts/table-colwidth-audit.js）实测内容宽度定：
    //   每列都留 ≥14px 余量，长文本列（道具名/描述）走 min-width 弹性，
    //   描述列按库里最长描述（43 字 ≈ 229px）给到 230，不再截断。
    { k: 'id', n: 'ID', w: 46 },
    { k: 'name', n: '道具名', minW: 140 },
    { k: 'item_type', n: '类型', w: 92, dict: 'itemType' },
    { k: 'category_name', n: '商城分类', w: 86 },
    { k: 'param1', n: '参数', w: 140, fmt: 'param' },
    { k: 'price_gold', n: '黄金价', w: 70 }, { k: 'price_diamond', n: '钻石价', w: 70 },
    { k: 'stock', n: '库存', w: 66, fmt: 'stock' },
    { k: 'description', n: '描述', minW: 230 }
  ],
  // ★ 2026-09-27 商城「装备」的价格定义迁到这里维护（「装备道具配置」）。
  //   这里只改价格/库存/身份字段，**不包含**军事/后勤/学识/战斗属性 ——
  //   属性由「军官装备管理 → 散件装备 / 套装管理」单独维护，避免同一字段两处能改。
  // ★ 2026-09-29 装备名用 min-width 弹性铺满：本表列全是固定宽，大屏下表格右侧（库存列和
  //   操作列之间）会被 el-table 塞一个空白填充列，装备名弹性拉伸后空列消失。
  equipments: [
    { k: 'id', n: 'ID', w: 56 }, { k: 'name', n: '装备名', minW: 140 }, { k: 'type', n: '类型', w: 70 },
    { k: 'tier', n: '品质', w: 76, dict: 'tier' }, { k: 'slot', n: '部位', w: 76 },
    { k: 'set_id', n: '套装ID', w: 80 }, { k: 'level', n: '需求等级', w: 80 },
    { k: 'series', n: '系列', w: 90 },
    { k: 'price_gold', n: '黄金价', w: 90 }, { k: 'price_diamond', n: '钻石价', w: 90 },
    { k: 'stock', n: '库存', w: 76, fmt: 'stock' }
  ],
  // ★ 2026-09-27 商城「宝箱」的价格定义 + 上架/库存 + 奖池迁到这里维护（「套装装备配置」）。
  //   宝箱是套装装备的唯一产出渠道，奖池在行内「奖池」按钮的弹窗里维护。
  // ★ 2026-10-07 宝箱配置列重排（用户反馈「列宽忽宽忽窄/被切」）：
  //   体检发现原 10 列里有 **341px 死白** —— 黄金价 90 只用到 29、宝箱名 140 只用到 70、
  //   单次上限 80 只用到 29…，而「说明/奖池说明」反而被切（需要 310/324，实际只有 108/179）。
  //   改法：① 黄金价 + 钻石价 合并成一列「价格」（宝箱只会用其中一种货币，分两列**总有一列是 0 占地方**）；
  //        ② 其余列按内容宽度收紧；③ 让出的宽度给两个长文本列。
  //   注：长文本列仍会截断（10 列塞不进 955px），鼠标悬停有 tooltip 看全文。
  chests: [
    { k: 'id', n: 'ID', w: 46 },
    // ★ 宝箱名固定宽（不给 min-width）：否则大屏下它也会被拉伸到 188px，
    //   和两个长文本列一起「忽宽忽窄」；弹性列只留真正需要空间的「说明/奖池说明」。
    { k: 'name', n: '宝箱名', w: 100 },
    { k: 'price', n: '价格', w: 90, fmt: 'price' },
    { k: 'stock', n: '库存', w: 62, fmt: 'stock' },
    { k: 'open_max', n: '单次上限', w: 82 },
    { k: 'enabled', n: '上架', w: 66, dict: 'chestOn' },
    { k: 'sort_no', n: '排序', w: 58 },
    { k: 'des', n: '说明', minW: 205 },
    { k: 'effect', n: '奖池说明', minW: 245 }
  ],
  taskTypes: [
    { k: 'id', n: 'ID', w: 56 }, { k: 'name', n: '类型名', w: 110 }, { k: 'code', n: '代码', w: 110 },
    { k: 'reset_type', n: '重置', w: 80, dict: 'resetType' }, { k: 'sort_no', n: '排序', w: 60 },
    // ★ 最后一列不固定宽、只给最小宽，让表格自动铺满整个卡片（用户反馈右侧大空白）
    { k: 'status', n: '状态', minW: 90, dict: 'onoff' }
  ],
  tasks: [
    { k: 'id', n: 'ID', w: 56 }, { k: 'name', n: '任务名', w: 120 },
    // ★ task_type 存的是 build_upgrade 这类英文代码 → 列表里转成中文
    { k: 'task_type', n: '任务行为', w: 110, dict: 'taskAction' },
    { k: 'target', n: '目标数', w: 70 }, { k: 'reward_gold', n: '黄金', w: 76 }, { k: 'reward_food', n: '粮食', w: 70 },
    { k: 'reward_prestige', n: '声望', w: 60 },     { k: 'sort_no', n: '排序', w: 56 },
    // ★ 同上：最后一列弹性铺满
    { k: 'status', n: '状态', minW: 90, dict: 'onoff' }
  ],
  cities: [
    { k: 'id', n: '城池ID', w: 80 }, { k: 'user_id', n: '用户ID', w: 80 }, { k: 'name', n: '城名', w: 110 },
    { k: 'x', n: 'X', w: 60 }, { k: 'y', n: 'Y', w: 60 }, { k: 'city_level', n: '市政厅', w: 80 },
    { k: 'pop', n: '人口', w: 90 }, { k: 'gold', n: '黄金', w: 100 }, { k: 'food', n: '粮食', w: 90 },
    { k: 'steel', n: '钢铁', w: 90 }, { k: 'oil', n: '石油', w: 90 }, { k: 'rare', n: '稀矿', w: 90 },
    { k: 'feelings', n: '民心', w: 70 }, { k: 'tax_rate', n: '税率', w: 70 }
  ],
  // ★ 2026-09-28 玩家钻石流水（只读）：按玩家ID/昵称搜索；created_at 由后端格式化好
  diamondLogs: [
    { k: 'id', n: '流水ID', w: 80 },
    { k: 'user_id', n: '玩家ID', w: 90 },
    { k: 'nickname', n: '玩家昵称', w: 120 },
    { k: 'change', n: '变动', w: 100 },
    { k: 'balance', n: '变动后余额', w: 110 },
    { k: 'reason', n: '变动原因', minW: 160 },
    { k: 'created_at', n: '发生时间', minW: 150 }
  ],
  // ★ 2026-10-02 玩家道具使用流水（只读）：按玩家ID/昵称/道具名搜索；created_at 由后端格式化好
  itemUseLogs: [
    { k: 'id', n: '记录ID', w: 80 },
    { k: 'user_id', n: '玩家ID', w: 90 },
    { k: 'nickname', n: '玩家昵称', w: 120 },
    { k: 'cfg_id', n: '道具ID', w: 80 },
    { k: 'item_name', n: '道具名', w: 120 },
    { k: 'item_type', n: '道具类型', w: 90, dict: 'itemType' },
    { k: 'count', n: '数量', w: 70 },
    { k: 'reason', n: '消耗原因', minW: 140 },
    { k: 'created_at', n: '发生时间', minW: 150 }
  ]
}

// 各数据表的编辑表单字段（t: num=数字 input=单行 text=多行；opts=下拉）
const FORMS = {
  buildings: [
    { k: 'name', n: '建筑名', t: 'input', req: true, max: 50 },
    { k: 'type', n: '类型', t: 'num', opts: [{ v: 1, n: '1 资源' }, { v: 2, n: '2 军事' }, { v: 3, n: '3 城防' }, { v: 4, n: '4 市政' }] },
    { k: 'max_level', n: '最高等级', t: 'num' },
    { k: 'unique_flag', n: '唯一建筑', t: 'num', opts: [{ v: 0, n: '0 可多座' }, { v: 1, n: '1 唯一' }] },
    { k: 'can_delete', n: '可拆除', t: 'num', opts: [{ v: 0, n: '0 不可' }, { v: 1, n: '1 可' }] },
    { k: 'pre_building', n: '前置建筑', t: 'input', max: 255 },
    { k: 'des', n: '描述', t: 'text' }
  ],
  buildingLevels: [
    { k: 'building_id', n: '建筑ID', t: 'num' },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'pop', n: '提供人口', t: 'num' },
    { k: 'food', n: '粮食消耗', t: 'num' }, { k: 'steel', n: '钢铁消耗', t: 'num' },
    { k: 'oil', n: '石油消耗', t: 'num' }, { k: 'rare', n: '稀矿消耗', t: 'num' },
    { k: 'gold', n: '黄金消耗', t: 'num' },
    { k: 'build_time', n: '建造耗时(秒)', t: 'num' },
    { k: 'capacity', n: '仓储容量', t: 'num' },
    { k: 'effect', n: '效果', t: 'text' }
  ],
  troops: [
    { k: 'name', n: '兵种名', t: 'input', req: true, max: 50 },
    { k: 'name_axis', n: '轴心国名', t: 'input', max: 50 },
    { k: 'name_ally', n: '同盟国名', t: 'input', max: 50 },
    { k: 'type', n: '类型', t: 'num', opts: [{ v: 1, n: '1 海军' }, { v: 2, n: '2 陆军' }, { v: 3, n: '3 空军' }, { v: 4, n: '4 城防' }] },
    { k: 'health', n: '生命', t: 'num' },
    { k: 'atk_sea', n: '对海攻击', t: 'num' }, { k: 'atk_ground', n: '对地攻击', t: 'num' },
    { k: 'atk_air', n: '对空攻击', t: 'num' }, { k: 'atk_def', n: '对防攻击', t: 'num' },
    { k: 'defence', n: '防御', t: 'num' }, { k: 'speed', n: '速度', t: 'num' },
    { k: 'attack_range', n: '射程', t: 'num' }, { k: 'carry', n: '运载量', t: 'num' },
    { k: 'pop', n: '占用人口', t: 'num' },
    { k: 'food_keep', n: '耗粮/时/个', t: 'num' }, { k: 'oil_keep', n: '耗油/时/个', t: 'num' },
    { k: 'food', n: '粮食造价', t: 'num' }, { k: 'steel', n: '钢铁造价', t: 'num' },
    { k: 'oil', n: '石油造价', t: 'num' }, { k: 'rare', n: '稀矿造价', t: 'num' },
    { k: 'train_time', n: '训练耗时(秒)', t: 'num' },
    { k: 'require', n: '前置要求', t: 'input', max: 500 },
    { k: 'icon', n: '图标', t: 'input', max: 50 },
    { k: 'repair_rate', n: '战损修复率%', t: 'num' }
  ],
  techs: [
    { k: 'name', n: '科技名', t: 'input', req: true, max: 50 },
    { k: 'type', n: '类型', t: 'num', opts: [{ v: 1, n: '1 生产' }, { v: 2, n: '2 军事' }, { v: 3, n: '3 辅助' }] },
    { k: 'max_level', n: '最高等级', t: 'num' },
    { k: 'pre_building', n: '需科研中心等级', t: 'num' },
    { k: 'pre_tech', n: '前置科技ID', t: 'num' },
    { k: 'pre_tech_level', n: '前置科技等级', t: 'num' },
    { k: 'effect', n: '效果', t: 'text' },
    { k: 'des', n: '描述', t: 'text' }
  ],
  techLevels: [
    { k: 'tech_id', n: '科技ID', t: 'num' },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'food', n: '粮食消耗', t: 'num' }, { k: 'steel', n: '钢铁消耗', t: 'num' },
    { k: 'oil', n: '石油消耗', t: 'num' }, { k: 'rare', n: '稀矿消耗', t: 'num' },
    { k: 'gold', n: '黄金消耗', t: 'num' },
    { k: 'research_time', n: '研究耗时(秒)', t: 'num' },
    { k: 'effect', n: '效果', t: 'text' }
  ],
  wildlands: [
    { k: 'type', n: '类型', t: 'num', opts: [{ v: 1, n: '1 陆地野地' }, { v: 2, n: '2 海野' }, { v: 3, n: '3 寇城' }] },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'troops', n: '守军JSON', t: 'text' },
    { k: 'res_min', n: '资源下限', t: 'num' },
    { k: 'res_max', n: '资源上限', t: 'num' },
    { k: 'officer_min', n: '军官下限', t: 'num' },
    { k: 'officer_max', n: '军官上限', t: 'num' },
    { k: 'treasure', n: '宝物', t: 'input', max: 100 },
    { k: 'des', n: '描述', t: 'text' }
  ],
  activities: [
    { k: 'name', n: '活动名', t: 'input', req: true, max: 50 },
    { k: 'type', n: '类型', t: 'num', opts: [
      { v: 1, n: '资源增产' }, { v: 2, n: '造兵打折' }, { v: 3, n: '建造加速' },
      { v: 4, n: '研究加速' }, { v: 5, n: '声望加成' }] },
    { k: 'param', n: '参数(%)', t: 'num' },
    // ★ 时间字段用日期选择器（原来要手填毫秒时间戳，运营没法用）
    { k: 'start_time', n: '开始时间', t: 'time' },
    { k: 'end_time', n: '结束时间', t: 'time' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 1, n: '开启' }, { v: 0, n: '关闭' }] },
    { k: 'des', n: '说明', t: 'text' }
  ],
  // ★ 2026-10-07 道具表单重排：按「是什么 → 归哪类 → 参数 → 价格 → 库存」的顺序，
  //   分类不再允许随手输入（原 allow-create 能把同一分类写成多种写法），改成固定下拉；
  //   「留空 = 按类型自动归类」是推荐用法（后端 ezfyCategoryOf 会推导）。
  items: [
    { k: 'name', n: '道具名', t: 'input', req: true, max: 50 },
    { k: 'item_type', n: '类型', t: 'num', opts: ITEM_TYPES },
    { k: 'category', n: '商城分类', t: 'input', filterable: true, opts: ITEM_CATEGORY_OPTS },
    // 参数(param1)的含义随类型变化 → 标签由 computed formFields 按所选类型动态改写
    { k: 'param1', n: '参数', t: 'num' },
    { k: 'price_gold', n: '黄金售价', t: 'num' },
    { k: 'price_diamond', n: '钻石售价', t: 'num' },
    { k: 'stock', n: '库存（-1 = 无上限，可随便买；0 = 售罄）', t: 'num', min: -1 },
    { k: 'icon', n: '图标文件名（可留空）', t: 'input', max: 50 },
    { k: 'description', n: '描述', t: 'text' }
  ],
  // ★ 2026-09-27 装备道具配置：只维护价格/库存/身份字段（属性在军官装备管理改）
  equipments: [
    { k: 'name', n: '装备名', t: 'input', req: true, max: 50 },
    { k: 'type', n: '类型（武器/防具/饰品/套装）', t: 'input', max: 20 },
    { k: 'tier', n: '品质', t: 'num', opts: [
      { v: 1, n: '1 初级' }, { v: 2, n: '2 中级' }, { v: 3, n: '3 高级' }, { v: 4, n: '4 特殊' }] },
    { k: 'slot', n: '部位', t: 'input', max: 20 },
    { k: 'set_id', n: '所属套装ID（0=散件）', t: 'num' },
    { k: 'level', n: '需求等级', t: 'num' },
    { k: 'series', n: '系列', t: 'input', max: 30 },
    { k: 'price_gold', n: '黄金售价', t: 'num' },
    { k: 'price_diamond', n: '钻石售价', t: 'num' },
    { k: 'stock', n: '库存（-1 = 无上限）', t: 'num', min: -1 }
  ],
  // ★ 2026-09-27 套装装备配置（宝箱）：价格/库存/上架 + 说明；奖池走行内「奖池」按钮弹窗
  chests: [
    { k: 'name', n: '宝箱名', t: 'input', req: true, max: 100 },
    { k: 'price_gold', n: '黄金售价', t: 'num' },
    { k: 'price_diamond', n: '钻石售价', t: 'num' },
    { k: 'stock', n: '库存（-1 = 无上限）', t: 'num', min: -1 },
    { k: 'open_max', n: '单次开箱上限', t: 'num', min: 1 },
    { k: 'enabled', n: '上架状态', t: 'num', opts: [{ v: 1, n: '上架' }, { v: 0, n: '下架' }] },
    { k: 'sort_no', n: '排序', t: 'num' },
    { k: 'des', n: '说明', t: 'text' },
    { k: 'effect', n: '奖池说明', t: 'text' }
  ],
  taskTypes: [
    { k: 'name', n: '类型名', t: 'input', req: true, max: 50 },
    { k: 'code', n: '代码', t: 'input', max: 30 },
    { k: 'reset_type', n: '重置方式', t: 'num', opts: [{ v: 1, n: '每日' }, { v: 2, n: '每周' }, { v: 0, n: '一次性' }] },
    { k: 'sort_no', n: '排序', t: 'num' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 1, n: '启用' }, { v: 0, n: '停用' }] }
  ],
  tasks: [
    { k: 'name', n: '任务名', t: 'input', req: true, max: 100 },
    { k: 'task_type', n: '任务行为', t: 'input', opts: TASK_ACTIONS },
    { k: 'target', n: '目标数量', t: 'num' },
    { k: 'reward_gold', n: '黄金奖励', t: 'num' }, { k: 'reward_food', n: '粮食奖励', t: 'num' },
    { k: 'reward_steel', n: '钢铁奖励', t: 'num' }, { k: 'reward_oil', n: '石油奖励', t: 'num' },
    { k: 'reward_rare', n: '稀矿奖励', t: 'num' },
    { k: 'reward_prestige', n: '声望奖励', t: 'num' },
    { k: 'sort_no', n: '排序', t: 'num' },
    // ★ 分类原来填数字 ID，人看不懂 → 由 computed 注入任务类型下拉（选项名来自任务类型表）
    { k: 'type_id', n: '任务分类', t: 'num' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 1, n: '启用' }, { v: 0, n: '停用' }] }
  ],
  cities: [
    { k: 'name', n: '城名', t: 'input', req: true, max: 50 },
    { k: 'city_level', n: '市政厅等级', t: 'num' },
    { k: 'feelings', n: '民心', t: 'num' }, { k: 'grievance', n: '民怨', t: 'num' },
    { k: 'tax_rate', n: '税率%', t: 'num' },
    { k: 'pop', n: '人口', t: 'num' }, { k: 'pop_max', n: '人口上限', t: 'num' },
    { k: 'gold', n: '黄金', t: 'num' }, { k: 'food', n: '粮食', t: 'num' },
    { k: 'steel', n: '钢铁', t: 'num' }, { k: 'oil', n: '石油', t: 'num' }, { k: 'rare', n: '稀矿', t: 'num' },
    { k: 'gold_cap', n: '黄金上限', t: 'num' }, { k: 'food_cap', n: '粮食上限', t: 'num' },
    { k: 'steel_cap', n: '钢铁上限', t: 'num' }, { k: 'oil_cap', n: '石油上限', t: 'num' },
    { k: 'rare_cap', n: '稀矿上限', t: 'num' }
  ]
}

export default {
  name: 'AdminEzfyData',
  data () {
    return {
      // ★ 只保留「没有专属模块」的表 —— 建筑/兵种/科技/野地/城池
      //   已在各自模块里维护，放这里会和那些模块重复（同一个字段两处能改）。
      tables: [
        { k: 'items', n: '道具配置' },
        // ★ 2026-09-27 商城「装备 | 宝箱」的价格定义迁到这里
        { k: 'equipments', n: '装备道具配置' },
        { k: 'chests', n: '宝箱配置' },
        { k: 'activities', n: '节日活动' },
        { k: 'taskTypes', n: '任务类型' },
        { k: 'tasks', n: '任务配置' },
        // ★ 2026-09-28 玩家钻石流水（只读查看，按玩家ID/昵称搜索）
        { k: 'diamondLogs', n: '钻石流水' },
        // ★ 2026-10-02 玩家道具使用流水（只读查看，按玩家ID/昵称/道具名搜索）
        { k: 'itemUseLogs', n: '道具使用' }
      ],
      moved: [
        { k: 'buildings', n: '建筑配置', to: '建筑管理 → 总建筑配置' },
        { k: 'buildingLevels', n: '建筑等级', to: '建筑管理 → 总建筑配置 → 等级配置' },
        { k: 'troops', n: '兵种配置', to: '兵种管理 → 兵种配置' },
        { k: 'techs', n: '科技配置', to: '科技管理 → 科技配置' },
        { k: 'techLevels', n: '科技等级', to: '科技管理 → 科技配置 → 等级配置' },
        { k: 'wildlands', n: '野地配置', to: '地图管理 → 野地类型' },
        { k: 'cities', n: '玩家城池', to: '城市管理' }
      ],
      table: 'items', word: '',
      // ★ 2026-10-07 道具配置的「商城分类」筛选：cat = 当前选中分类，cats = 可选项（后端回传）
      cat: '', cats: [],
      // ★ 2026-09-27 装备道具配置默认只展示用户商城上架的装备
      mallOnly: true,
      rows: [], total: 0, page: 1, size: 5, loading: false,
      showForm: false, saving: false,
      form: {}, formId: 0,
      // ★ 动态字典：任务分类名来自「任务类型」表，不能写死在前端
      //   dynDicts.taskType = { id: { n: 类型名 } }，供 dictOf 兜底查询
      dynDicts: { taskType: {} },
      taskTypeOpts: [],
      // ★ 发放道具对话框状态
      grantDlg: false, grantSaving: false,
      grantPlayer: '', grantPlayers: [], grantUserId: 0,
      grantItemOpts: [], grantItems: [],
      // ★ 套装装备配置（宝箱）→ 奖池管理状态（2026-09-27 从「军官管理 → 宝箱」迁来）
      chPoolDlg: false, chPoolChest: {}, chPool: [], chWeightSum: 0,
      chItemDlg: false, chif: {}, chBulkSetId: 0, chBulkWeight: 100,
      equipSets: [],
      // ★ 2026-10-07 奖池条目编辑：奖品按名称选 → 装备/道具候选列表（懒加载）
      equipOpts: [], itemOpts: [],
      // 模板里只能访问实例属性，模块级 const 要挂到 data 上
      chestKindLabel: CHEST_KIND_LABEL, qualityOpts: QUALITY_OPTS
    }
  },
  computed: {
    cols () { return COLS[this.table] || [] },
    // 只读视图（钻石流水 / 道具使用）：无新增、无编辑、无删除操作列
    isReadOnly () { return this.table === 'diamondLogs' || this.table === 'itemUseLogs' },
    searchPh () {
      if (this.table === 'diamondLogs') return '玩家ID / 昵称搜索'
      if (this.table === 'itemUseLogs') return '玩家ID / 昵称 / 道具名搜索'
      return '名称 / ID 搜索'
    },
    // 任务配置的「任务分类」要选类型名而不是填数字 ID → 动态注入任务类型下拉
    formFields () {
      const list = FORMS[this.table] || []
      if (this.table === 'tasks') return list.map(f => f.k === 'type_id' ? { ...f, opts: this.taskTypeOpts } : f)
      // ★ 道具配置：「参数(param1)」的含义随「类型」变化 → 标签跟着所选类型走，
      //   运营填数字前就知道这个参数是什么（例：类型=建筑加速 → 参数（加速的分钟数））
      if (this.table === 'items') {
        const meta = PARAM_META[this.form.item_type]
        return list.map(f => f.k === 'param1' && meta ? { ...f, n: '参数（' + meta.label + '）' } : f)
      }
      return list
    },
    tableName () {
      const t = this.tables.find(t => t.k === this.table)
      return t ? t.n : '数据'
    },
    // 发放对象选中后的展示（昵称 + ID），来自搜索列表
    grantTargetName () {
      const p = this.grantPlayers.find(p => p.user_id === this.grantUserId)
      return p ? (p.nickname + '（ID:' + p.user_id + '）') : ''
    }
  },
  mounted () { this.load(); this.loadTaskTypeDict() },
  methods: {
    load () {
      this.loading = true
      const params = { page: this.page, size: this.size, word: this.word }
      // ★ 装备道具配置：默认只展示用户商城上架的装备（mall=1 后端按商城条件过滤）
      if (this.table === 'equipments' && this.mallOnly) params.mall = 1
      // ★ 道具配置：按商城分类筛选（分类是推导值，后端按同一口径算好 id 集合再过滤）
      if (this.table === 'items' && this.cat) params.cat = this.cat
      api.get('/admin/ezfy-data/' + this.table, { params }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.rows = r.data.list
          this.total = r.data.total
          this.page = r.data.page
          // ★ 分类选项由后端回传（口径唯一，前端不再重复实现分类推导规则）
          if (this.table === 'items') this.cats = r.data.cats || []
        } else this.$message.error(r.msg)
      })
    },
    // ★ 道具参数列：param1 的含义随类型变化 → 按类型渲染成「含义 + 数值 + 单位」
    paramText (row) {
      const raw = row.param1
      if (raw === null || raw === undefined || raw === '') return '—'
      const meta = PARAM_META[row.item_type]
      return meta ? meta.text(Number(raw)) : String(raw)
    },
    // ★ 宝箱价格：库里「黄金价 / 钻石价」只会用其中一个 → 合并成一列并标出货币，
    //   否则两列里总有一列是 0（白占 90px，也是「列宽忽宽忽窄」的观感来源之一）。
    priceText (row) {
      const g = Number(row.price_gold || 0)
      const d = Number(row.price_diamond || 0)
      if (g > 0 && d > 0) return '黄金 ' + g + ' / 钻石 ' + d
      if (d > 0) return '钻石 ' + d
      if (g > 0) return '黄金 ' + g
      return '未定价'
    },
    fmt (v) {
      if (v === null || v === undefined) return '—'
      return String(v)
    },
    // 毫秒时间戳 → 本地日期时间（0 / 空 = 未设置）
    fmtTime (v) {
      const n = Number(v)
      if (!n) return '—'
      const d = new Date(n < 1e12 ? n * 1000 : n) // 兼容秒级时间戳
      return d.toLocaleString('zh-CN', { hour12: false })
    },
    // 切换数据表 Tab：清空搜索词与分类筛选、回到第 1 页重新拉取
    onTabChange (tab) {
      this.table = tab.name
      this.word = ''
      this.cat = ''
      this.page = 1
      this.load()
    },
    // 枚举值 → { n: 文字, t: el-tag 颜色 }；先查静态字典再查动态字典，都没有就原样显示
    dictOf (col, v) {
      const d = DICTS[col.dict] || this.dynDicts[col.dict]
      return (d && d[v]) || { n: this.fmt(v), t: '' }
    },
    // 任务分类的名字映射 + 编辑表单下拉选项（数据来自「任务类型」表，不能写死）
    loadTaskTypeDict () {
      api.get('/admin/ezfy-data/taskTypes', { params: { page: 1, size: 200 } }).then(r => {
        if (r.code !== 0) return
        const m = {}
        const opts = []
        const list = r.data.list || []
        list.forEach(t => {
          m[t.id] = { n: t.name }
          opts.push({ v: t.id, n: t.name })
        })
        this.$set(this.dynDicts, 'taskType', m)
        this.taskTypeOpts = opts
      })
    },
    // 空值口径：数字用 0、日期用 null（日期选择器要 null 才显示占位）、其余空串
    blankVal (fd) {
      if (fd.t === 'num') return 0
      if (fd.t === 'time') return null
      return ''
    },
    openCreate () {
      const f = {}
      this.formFields.forEach(fd => { f[fd.k] = this.blankVal(fd) })
      this.form = f
      this.formId = 0
      this.showForm = true
    },
    openEdit (row) {
      const f = {}
      this.formFields.forEach(fd => {
        const v = row[fd.k]
        if (v === null || v === undefined) f[fd.k] = this.blankVal(fd)
        // 时间字段：库里 0 表示未设置，转成 null 才不会显示成 1970-01-01
        else f[fd.k] = fd.t === 'time' ? (Number(v) > 0 ? Number(v) : null) : v
      })
      this.form = f
      this.formId = row.id
      this.showForm = true
    },
    doSave () {
      const reqField = this.formFields.find(f => f.req)
      if (reqField && !String(this.form[reqField.k] || '').trim()) {
        this.$message.warning('请填写' + reqField.n)
        return
      }
      this.saving = true
      const p = this.formId
        ? api.put('/admin/ezfy-data/' + this.table + '/' + this.formId, this.form)
        : api.post('/admin/ezfy-data/' + this.table, this.form)
      p.then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.showForm = false; this.load() }
        else this.$message.error(r.msg)
      })
    },
    doDelete (row) {
      this.$confirm('确认删除「' + (row.name || 'ID' + row.id) + '」？删除后不可恢复！', '删除确认', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-data/' + this.table + '/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ============ 道具配置 → 发放道具（2026-09-26 从玩家信息管理移到这里） ============
    openItemGrant () {
      this.grantDlg = true
      this.grantPlayer = ''; this.grantPlayers = []; this.grantUserId = 0
      this.grantItems = [{ cfg_id: '', count: 1 }]
      // ★ 2026-09-27 修复「发放道具不全 / 检索迁城无匹配」：
      //   旧实现走 /admin/ezfy-data/items 分页接口，pageOf 会把 size>100 钳制回落 10，
      //   下拉永远只拿到前 10 件道具（迁城计划等选不到）。
      //   改为专用全量接口 /admin/ezfy-item-grant/options（id+name 全量返回，不走分页钳制）。
      api.get('/admin/ezfy-item-grant/options').then(r => {
        if (r.code === 0) this.grantItemOpts = r.data.list || []
      })
    },
    // 按昵称 / 游戏ID 搜索目标玩家（昵称模糊、ID 精确，后端返回最多 20 条）
    searchGrantPlayer () {
      const word = (this.grantPlayer || '').trim()
      if (!word) { this.$message.warning('请输入玩家昵称或游戏ID'); return }
      api.get('/admin/ezfy-item-grant/players', { params: { word } }).then(r => {
        if (r.code !== 0) { this.$message.error(r.msg); this.grantPlayers = []; return }
        this.grantPlayers = r.data.list || []
        // 只有一条结果时自动选中
        this.grantUserId = this.grantPlayers.length === 1 ? this.grantPlayers[0].user_id : 0
        if (!this.grantPlayers.length) this.$message.warning('未找到匹配的玩家')
      })
    },
    doItemGrant () {
      const target = this.grantUserId || ((this.grantPlayer || '').trim())
      if (!target) { this.$message.warning('请先搜索并选择要发放的玩家'); return }
      const items = this.grantItems.filter(it => it.cfg_id && it.count > 0)
      if (!items.length) { this.$message.warning('请添加要发放的道具'); return }
      this.grantSaving = true
      api.post('/admin/ezfy-item-grant', { player: String(target), items }).then(r => {
        this.grantSaving = false
        if (r.code === 0) { this.$message.success(r.data.msg || '发放成功'); this.grantDlg = false }
        else this.$message.error(r.msg || '发放失败')
      }).catch(() => { this.grantSaving = false })
    },
    // ============ 套装装备配置（宝箱）→ 奖池管理（2026-09-27 从「军官管理 → 宝箱」迁来） ============
    openChestPool (row) {
      this.chPoolChest = row
      this.chBulkSetId = 0
      this.chBulkWeight = 100
      this.loadPoolRefs()
      this.loadChestPool()
      this.chPoolDlg = true
    },
    // ★ 2026-10-07 奖池条目编辑要「按名称选奖品」→ 懒加载三份候选：
    //   套装列表（批量加入 + 整套奖品）、装备列表、道具列表。
    loadPoolRefs () {
      if (!this.equipSets.length) {
        api.get('/admin/ezfy-equip-sets').then(r => { if (r.code === 0) this.equipSets = r.data.list })
      }
      if (!this.equipOpts.length) {
        api.get('/admin/ezfy-equipments', { params: { set_id: '' } }).then(r => {
          if (r.code !== 0) return
          const list = r.data.list || []
          this.equipOpts = list.map(e => ({
            id: e.id,
            label: e.id + ' · ' + e.name + (e.tier_name ? '（' + e.tier_name + (e.slot ? ' · ' + e.slot : '') + '）' : '')
          }))
        })
      }
      if (!this.itemOpts.length) this.loadItemOpts(1)
    },
    // 道具列表接口是分页的（size 上限 100）→ 循环取全量，保证下拉里能搜到所有道具
    loadItemOpts (page) {
      api.get('/admin/ezfy-data/items', { params: { page, size: 100 } }).then(r => {
        if (r.code !== 0) return
        const list = r.data.list || []
        this.itemOpts = this.itemOpts.concat(list.map(it => ({ id: it.id, label: it.id + ' · ' + it.name })))
        if (this.itemOpts.length < r.data.total && page < 5) this.loadItemOpts(page + 1)
      })
    },
    // 奖池列表的「品质」标签配色（口径 = 装备品质 普通/稀有/史诗/传说）
    qualityTagOf (q) {
      return QUALITY_TAG[q] || ''
    },
    loadChestPool () {
      api.get('/admin/ezfy-chests/' + this.chPoolChest.id + '/pool').then(r => {
        if (r.code === 0) {
          this.chPool = r.data.list
          this.chWeightSum = r.data.weight_sum
        } else this.$message.error(r.msg)
      })
    },
    openChestItemCreate () {
      this.loadPoolRefs()
      this.chif = { kind: 1, ref_id: 0, count: 1, weight: 100, quality: '', des: '' }
      this.chItemDlg = true
    },
    openChestItemEdit (row) {
      this.loadPoolRefs()
      const f = { id: row.id }
      CI_KEYS.forEach(k => { f[k] = row[k] })
      this.chif = f
      this.chItemDlg = true
    },
    doChestItemSave () {
      if (!this.chif.ref_id) { this.$message.warning('请选择奖品'); return }
      const body = {}
      CI_KEYS.forEach(k => { body[k] = this.chif[k] })
      const isNew = !this.chif.id
      this.saving = true
      const req = isNew
        ? api.post('/admin/ezfy-chests/' + this.chPoolChest.id + '/pool', body)
        : api.put('/admin/ezfy-chest-items/' + this.chif.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.chItemDlg = false; this.$message.success(r.msg || '已保存'); this.loadChestPool() }
        else this.$message.error(r.msg)
      })
    },
    delChestItem (row) {
      api.delete('/admin/ezfy-chest-items/' + row.id).then(r => {
        if (r.code === 0) { this.$message.success(r.msg || '已删除'); this.loadChestPool() } else this.$message.error(r.msg)
      })
    },
    doChestBulkAdd () {
      if (!this.chBulkSetId) { this.$message.warning('请选择要批量加入的套装'); return }
      this.saving = true
      api.post('/admin/ezfy-chests/' + this.chPoolChest.id + '/pool/bulk',
        { set_id: this.chBulkSetId, weight: this.chBulkWeight }).then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success(r.msg || '已加入'); this.loadChestPool() } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.danger-btn { color: #f56c6c; }
.unlimited { color: #67c23a; font-weight: 600; }
/* 数据表切换 Tab：与下方工具栏贴近一些，别留一大块空白 */
.cfg-tabs >>> .el-tabs__header { margin-bottom: 10px; }
/* 发放道具：已选玩家提示 + 道具行 */
.grant-target { margin-top: 6px; color: #67c23a; font-size: 12px; }
.grant-item-row { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.grant-x { color: #909399; }
</style>
