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
      <div class="toolbar">
        <el-select v-model="table" style="width:150px" @change="page = 1; load()">
          <el-option v-for="t in tables" :key="t.k" :label="t.n" :value="t.k" />
        </el-select>
        <el-input v-model="word" placeholder="名称 / ID 搜索" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="success" icon="el-icon-plus" @click="openCreate">新增</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="rows" v-loading="loading" stripe border max-height="620">
        <el-table-column v-for="col in cols" :key="col.k" :label="col.n" :width="col.w"
                         :align="col.w ? 'center' : 'left'" show-overflow-tooltip>
          <template slot-scope="{row}">{{ fmt(row[col.k]) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="130" align="center">
          <template slot-scope="{row}">
            <el-button type="text" size="mini" @click="openEdit(row)">编辑</el-button>
            <el-button type="text" size="mini" class="danger-btn" @click="doDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[10, 20, 50]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>

      <!-- 新增/编辑对话框 -->
      <el-dialog :title="formId ? '编辑' + tableName : '新增' + tableName" :visible.sync="showForm" width="600px" append-to-body>
        <el-form label-width="140px" size="small">
          <el-form-item v-for="f in formFields" :key="f.k" :label="f.n" :required="f.req">
            <el-select v-if="f.opts" v-model="form[f.k]" style="width:220px">
              <el-option v-for="o in f.opts" :key="o.v" :label="o.n" :value="o.v" />
            </el-select>
            <el-input-number v-else-if="f.t === 'num'" v-model="form[f.k]" :min="0" style="width:180px" />
            <el-input v-else-if="f.t === 'text'" v-model="form[f.k]" type="textarea" :rows="2" />
            <el-input v-else v-model="form[f.k]" :maxlength="f.max || 50" style="width:320px" />
          </el-form-item>
        </el-form>
        <em v-if="table === 'cities'">提示：城池为玩家运行数据，修改后玩家下次进入游戏生效</em>
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

// 各数据表的展示列（k=字段, n=列名, w=列宽）
const COLS = {
  activities: [
    { k: 'id', n: 'ID', w: 60 },
    { k: 'name', n: '活动名', w: 150 },
    { k: 'type_name', n: '类型', w: 110 },
    { k: 'param', n: '参数', w: 90 },
    { k: 'start_time', n: '开始', w: 160 },
    { k: 'end_time', n: '结束', w: 160 },
    { k: 'status_txt', n: '状态', w: 90 },
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
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '道具名', w: 130 }, { k: 'item_type', n: '类型', w: 90 },
    { k: 'param1', n: '参数', w: 90 }, { k: 'price_gold', n: '黄金价', w: 100 }, { k: 'icon', n: '图标', w: 90 },
    { k: 'description', n: '描述' }
  ],
  taskTypes: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '类型名', w: 130 }, { k: 'code', n: '代码', w: 130 },
    { k: 'reset_type', n: '重置', w: 90 }, { k: 'sort_no', n: '排序', w: 70 }, { k: 'status', n: '状态', w: 70 }
  ],
  tasks: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '任务名', w: 150 }, { k: 'task_type', n: '类型', w: 100 },
    { k: 'target', n: '目标数', w: 80 }, { k: 'reward_gold', n: '黄金', w: 90 }, { k: 'reward_food', n: '粮食', w: 80 },
    { k: 'reward_prestige', n: '声望', w: 70 }, { k: 'sort_no', n: '排序', w: 70 }, { k: 'status', n: '状态', w: 70 }
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
      { v: 1, n: '1 资源增产' }, { v: 2, n: '2 造兵打折' }, { v: 3, n: '3 建造加速' },
      { v: 4, n: '4 研究加速' }, { v: 5, n: '5 声望加成' }] },
    { k: 'param', n: '参数(%)', t: 'num' },
    { k: 'start_time', n: '开始时间(毫秒)', t: 'num' },
    { k: 'end_time', n: '结束时间(毫秒)', t: 'num' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 0, n: '0 关闭' }, { v: 1, n: '1 开启' }] },
    { k: 'des', n: '说明', t: 'text' }
  ],
  items: [
    { k: 'name', n: '道具名', t: 'input', req: true, max: 50 },
    { k: 'item_type', n: '类型', t: 'num', opts: [{ v: 1, n: '1 资源包' }, { v: 2, n: '2 黄金包' }, { v: 3, n: '3 建筑加速' }, { v: 4, n: '4 训练加速' }, { v: 5, n: '5 科技加速' }, { v: 6, n: '6 建筑图纸' }, { v: 7, n: '7 增产' }, { v: 8, n: '8 免战' }, { v: 9, n: '9 招生简章' }, { v: 10, n: '10 经验书' }, { v: 11, n: '11 军官技能书' }, { v: 12, n: '12 重修书' }, { v: 13, n: '13 改名卡' }, { v: 14, n: '14 阵营转换道具' }] },
    { k: 'param1', n: '参数', t: 'num' },
    { k: 'price_gold', n: '黄金售价', t: 'num' },
    { k: 'icon', n: '图标', t: 'input', max: 50 },
    { k: 'description', n: '描述', t: 'text' }
  ],
  taskTypes: [
    { k: 'name', n: '类型名', t: 'input', req: true, max: 50 },
    { k: 'code', n: '代码', t: 'input', max: 30 },
    { k: 'reset_type', n: '重置方式', t: 'num', opts: [{ v: 0, n: '0 一次性' }, { v: 1, n: '1 每日' }] },
    { k: 'sort_no', n: '排序', t: 'num' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 1, n: '1 启用' }, { v: 0, n: '0 停用' }] }
  ],
  tasks: [
    { k: 'name', n: '任务名', t: 'input', req: true, max: 100 },
    { k: 'task_type', n: '任务类型代码', t: 'input', max: 30 },
    { k: 'target', n: '目标数量', t: 'num' },
    { k: 'reward_gold', n: '黄金奖励', t: 'num' }, { k: 'reward_food', n: '粮食奖励', t: 'num' },
    { k: 'reward_steel', n: '钢铁奖励', t: 'num' }, { k: 'reward_oil', n: '石油奖励', t: 'num' },
    { k: 'reward_rare', n: '稀矿奖励', t: 'num' },
    { k: 'reward_prestige', n: '声望奖励', t: 'num' },
    { k: 'sort_no', n: '排序', t: 'num' },
    { k: 'type_id', n: '分类ID', t: 'num' },
    { k: 'status', n: '状态', t: 'num', opts: [{ v: 1, n: '1 启用' }, { v: 0, n: '0 停用' }] }
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
      rows: [], total: 0, page: 1, size: 10, loading: false,
      showForm: false, saving: false,
      form: {}, formId: 0
    }
  },
  computed: {
    cols () { return COLS[this.table] || [] },
    formFields () { return FORMS[this.table] || [] },
    tableName () {
      const t = this.tables.find(t => t.k === this.table)
      return t ? t.n : '数据'
    }
  },
  mounted () { this.load() },
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
    openCreate () {
      const f = {}
      this.formFields.forEach(fd => { f[fd.k] = fd.t === 'num' ? 0 : '' })
      this.form = f
      this.formId = 0
      this.showForm = true
    },
    openEdit (row) {
      const f = {}
      this.formFields.forEach(fd => { f[fd.k] = row[fd.k] === null || row[fd.k] === undefined ? (fd.t === 'num' ? 0 : '') : row[fd.k] })
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
</style>
