<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom:12px">
        <template slot="title">
          本页只维护「没有专属模块」的零散配置表。以下配置已迁到对应模块（避免两处重复维护）：
          <span v-for="(m, i) in moved" :key="m.k">
            <b>{{ m.n }}</b> → {{ m.to }}<span v-if="i < moved.length - 1">；</span>
          </span>
        </template>
      </el-alert>
      <!-- ★ 数据表切换：下拉框改为 Tab（用户要求，切换更直观） -->
      <el-tabs v-model="table" class="cfg-tabs" @tab-click="onTabChange">
        <el-tab-pane v-for="t in tables" :key="t.k" :label="t.n" :name="t.k" />
      </el-tabs>
      <div class="toolbar">
        <el-input v-model="word" placeholder="名称 / ID 搜索" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="success" icon="el-icon-plus" @click="openCreate">新增</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
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
            <span v-else>{{ fmt(row[col.k]) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button type="text" size="mini" @click="openEdit(row)">编辑</el-button>
            <el-button type="text" size="mini" class="danger-btn" @click="doDelete(row)">删除</el-button>
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
  // 道具类型
  itemType: {
    1: { n: '资源包' }, 2: { n: '黄金包' }, 3: { n: '建筑加速' }, 4: { n: '训练加速' },
    5: { n: '科技加速' }, 6: { n: '建筑图纸' }, 7: { n: '增产' }, 8: { n: '免战' },
    9: { n: '招生简章' }, 10: { n: '经验书' }, 11: { n: '军官技能书' }, 12: { n: '重修书' },
    13: { n: '改名卡' }, 14: { n: '阵营转换道具' }, 15: { n: '出征道具' }, 16: { n: '迁城道具' }
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
    wild_count: { n: '野地数量' }
  }
}

// 任务行为下拉选项（与上面 taskAction 同一份文案，避免两处口径不一致）
const TASK_ACTIONS = Object.keys(DICTS.taskAction).map(k => ({ v: k, n: DICTS.taskAction[k].n }))
// 道具类型下拉选项（同理，与 DICTS.itemType 同源）
const ITEM_TYPES = Object.keys(DICTS.itemType).map(k => ({ v: Number(k), n: DICTS.itemType[k].n }))

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
  items: [
    { k: 'id', n: 'ID', w: 56 }, { k: 'name', n: '道具名', w: 110 }, { k: 'item_type', n: '类型', w: 90, dict: 'itemType' },
    { k: 'category', n: '分类/货币', w: 96 }, { k: 'param1', n: '参数', w: 70 },
    { k: 'price_gold', n: '黄金价', w: 80 }, { k: 'price_diamond', n: '钻石价', w: 80 },
    { k: 'stock', n: '库存', w: 76, fmt: 'stock' }, { k: 'icon', n: '图标', w: 66 },
    { k: 'description', n: '描述' }
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
  items: [
    { k: 'name', n: '道具名', t: 'input', req: true, max: 50 },
    { k: 'stock', n: '库存（-1 = 无上限，可随便买；0 = 售罄）', t: 'num', min: -1 },
    { k: 'item_type', n: '类型', t: 'num', opts: ITEM_TYPES },
    { k: 'param1', n: '参数', t: 'num' },
    { k: 'price_gold', n: '黄金售价', t: 'num' },
    { k: 'price_diamond', n: '钻石售价', t: 'num' },
    // ★ 用户要求：道具可配「钻石道具 / 黄金道具」；钻石道具只能钻石买，黄金道具只能黄金买。
    //   用户端商城会按这两个分类分开展示（也可选别的分类名，或直接输入自定义分类）。
    { k: 'category', n: '分类 / 货币类型', t: 'input', max: 30, filterable: true, allowCreate: true, opts: [
      { v: '钻石道具', n: '钻石道具（只能用钻石买）' },
      { v: '黄金道具', n: '黄金道具（只能用黄金买）' },
      { v: '资源道具', n: '资源道具' }, { v: '加速道具', n: '加速道具' },
      { v: '建筑图纸', n: '建筑图纸' }, { v: '增益道具', n: '增益道具' },
      { v: '军官道具', n: '军官道具' }, { v: '身份道具', n: '身份道具' },
      { v: '出征道具', n: '出征道具' }, { v: '迁城道具', n: '迁城道具' },
      { v: '其他', n: '其他' }
    ] },
    { k: 'icon', n: '图标', t: 'input', max: 50 },
    { k: 'description', n: '描述', t: 'text' }
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
        { k: 'activities', n: '节日活动' },
        { k: 'taskTypes', n: '任务类型' },
        { k: 'tasks', n: '任务配置' }
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
      rows: [], total: 0, page: 1, size: 5, loading: false,
      showForm: false, saving: false,
      form: {}, formId: 0,
      // ★ 动态字典：任务分类名来自「任务类型」表，不能写死在前端
      //   dynDicts.taskType = { id: { n: 类型名 } }，供 dictOf 兜底查询
      dynDicts: { taskType: {} },
      taskTypeOpts: []
    }
  },
  computed: {
    cols () { return COLS[this.table] || [] },
    // 任务配置的「任务分类」要选类型名而不是填数字 ID → 动态注入任务类型下拉
    formFields () {
      const list = FORMS[this.table] || []
      if (this.table !== 'tasks') return list
      return list.map(f => f.k === 'type_id' ? { ...f, opts: this.taskTypeOpts } : f)
    },
    tableName () {
      const t = this.tables.find(t => t.k === this.table)
      return t ? t.n : '数据'
    }
  },
  mounted () { this.load(); this.loadTaskTypeDict() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/ezfy-data/' + this.table, { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.rows = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
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
    // 切换数据表 Tab：清空搜索词、回到第 1 页重新拉取
    onTabChange (tab) {
      this.table = tab.name
      this.word = ''
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
</style>
