<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="table" style="width:140px" @change="page = 1; load()">
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
        <el-table-column label="操作" width="130" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button type="text" size="mini" @click="openEdit(row)">编辑</el-button>
            <el-button type="text" size="mini" class="danger-btn" @click="doDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />

      <!-- 新增/编辑对话框 -->
      <el-dialog :title="form.id ? '编辑' + tableName : '新增' + tableName" :visible.sync="showForm" width="560px" append-to-body>
        <el-form label-width="130px" size="small">
          <el-form-item v-for="f in formFields" :key="f.k" :label="f.n" :required="f.req">
            <!-- 物品使用效果结构化编辑器 -->
            <template v-if="f.k === 'effect' && table === 'items'">
              <div class="drop-editor">
                <div class="eff-grid">
                  <div v-for="e in effNums" :key="e.k" class="eff-cell">
                    <span class="eff-label">{{ e.n }}</span>
                    <el-input-number v-model="eff[e.k]" :min="0" size="mini" controls-position="right" style="width:104px" />
                  </div>
                </div>
                <div class="eff-checks">
                  <el-checkbox v-model="eff.full">万能果（气血法力全恢复）</el-checkbox>
                  <el-checkbox v-model="eff.box">宝箱（开启随机奖励）</el-checkbox>
                  <el-checkbox v-model="eff.teleport">腾云符（使用时选目的地）</el-checkbox>
                  <el-checkbox v-model="eff.skill_sect">门派秘籍（学习本门派技能）</el-checkbox>
                </div>
                <div class="eff-cell" style="margin-bottom:6px">
                  <span class="eff-label">学会指定技能（技能ID，0=不学）</span>
                  <el-input-number v-model="eff.skill" :min="0" size="mini" controls-position="right" style="width:104px" />
                </div>
                <div class="eff-cell">
                  <span class="eff-label">回城地点（卷轴类）</span>
                  X <el-input-number v-model="eff.goto_x" :min="0" size="mini" controls-position="right" style="width:90px" />
                  Y <el-input-number v-model="eff.goto_y" :min="0" size="mini" controls-position="right" style="width:90px" />
                </div>
                <em>填 0 / 不勾选 表示无此效果，保存后自动生成配置，无需手写格式</em>
              </div>
            </template>
            <!-- NPC掉落结构化编辑器 -->
            <template v-else-if="f.k === 'drops' && table === 'npcs'">
              <div class="drop-editor">
                <div v-for="(d, i) in dropRows" :key="'dr' + i" class="drop-row">
                  <el-select v-model="d.type" style="width:86px" @change="d.id = 0; d.opts = []">
                    <el-option value="item" label="物品" />
                    <el-option value="equip" label="装备" />
                  </el-select>
                  <el-select v-model="d.id" filterable remote
                             :remote-method="w => searchDrop(w, i)" :loading="d.loading"
                             placeholder="搜索名称或ID" style="width:230px">
                    <el-option v-for="o in d.opts" :key="o.v" :label="o.n" :value="o.v" />
                  </el-select>
                  <el-input-number v-model="d.pct" :min="0" :max="100" :step="5" style="width:110px" />
                  <span class="pct">%</span>
                  <el-button type="text" class="danger-btn" @click="dropRows.splice(i, 1)">删除</el-button>
                </div>
                <div>
                  <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="addDrop">添加掉落</el-button>
                </div>
                <em>概率为百分比（30 表示 30% 掉率），保存后自动换算存储</em>
              </div>
            </template>
            <!-- 功能NPC传送目的地结构化编辑器 -->
            <template v-else-if="f.k === 'teles' && table === 'mapnpcs'">
              <div class="drop-editor">
                <div v-for="(t, i) in teleRows" :key="'tl' + i" class="drop-row">
                  <el-select v-model="t.pick" filterable remote :remote-method="w => searchNode(w, i)"
                             :loading="t.loading" placeholder="搜索地图节点" style="width:170px"
                             @change="v => pickNode(v, i)">
                    <el-option v-for="o in t.opts" :key="o.v" :label="o.n" :value="o.v" />
                  </el-select>
                  <el-input v-model="t.name" placeholder="显示名称" style="width:120px" />
                  <el-input-number v-model="t.dtx" :min="0" size="mini" controls-position="right" style="width:86px" />
                  <el-input-number v-model="t.dty" :min="0" size="mini" controls-position="right" style="width:86px" />
                  <el-button type="text" class="danger-btn" @click="teleRows.splice(i, 1)">删除</el-button>
                </div>
                <div>
                  <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="addTele">添加传送目的地</el-button>
                </div>
                <em>可搜索选择地图节点自动填充名称和坐标，也可手动填写；玩家点击该NPC时会显示这些传送选项</em>
              </div>
            </template>
            <el-select v-else-if="f.opts" v-model="form[f.k]" style="width:220px">
              <el-option v-for="o in f.opts" :key="o.v" :label="o.n" :value="o.v" />
            </el-select>
            <el-input-number v-else-if="f.t === 'num'" v-model="form[f.k]" :min="f.min !== undefined ? f.min : 0" style="width:180px" />
            <el-input v-else-if="f.t === 'text'" v-model="form[f.k]" type="textarea" :rows="2" />
            <el-input v-else v-model="form[f.k]" :maxlength="f.max || 50" style="width:320px" />
          </el-form-item>
        </el-form>
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
  items: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'category', n: '分类', w: 70 },
    { k: 'level', n: '等级', w: 70 }, { k: 'price', n: '价格', w: 90 }, { k: 'bean_price', n: '金豆价', w: 80 },
    { k: 'bind', n: '绑定', w: 60 }, { k: 'desc', n: '描述' }
  ],
  equips: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'category', n: '部位', w: 70 },
    { k: 'sect', n: '门派', w: 70 }, { k: 'level', n: '等级', w: 70 }, { k: 'price', n: '价格', w: 90 },
    { k: 'hp', n: '气血', w: 70 }, { k: 'atk', n: '攻', w: 60 }, { k: 'mg', n: '魔', w: 60 },
    { k: 'def', n: '防', w: 60 }, { k: 'desc', n: '描述' }
  ],
  npcs: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'level', n: '等级', w: 70 },
    { k: 'kind', n: '类型', w: 60 }, { k: 'max_hp', n: '气血', w: 80 }, { k: 'atk', n: '攻', w: 60 },
    { k: 'def', n: '防', w: 60 }, { k: 'exp_reward', n: '经验', w: 70 }, { k: 'money_reward', n: '银两', w: 70 },
    { k: 'take', n: '台词' }
  ],
  skills: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 130 }, { k: 'category', n: '类型', w: 70 },
    { k: 'sect', n: '门派', w: 70 }, { k: 'mp_cost', n: '耗蓝', w: 70 }, { k: 'multiplier', n: '威力%', w: 80 },
    { k: 'learn_level', n: '可学等级', w: 80 }, { k: 'desc', n: '描述' }
  ],
  maps: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '节点名', w: 150 }, { k: 'dtx', n: 'X', w: 60 },
    { k: 'dty', n: 'Y', w: 60 }, { k: 'up', n: '北', w: 70 }, { k: 'down', n: '南', w: 70 },
    { k: 'left', n: '西', w: 70 }, { k: 'right', n: '东', w: 70 }, { k: 'desc', n: '描述' }
  ],
  bosses: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'level', n: '等级', w: 70 },
    { k: 'max_hp', n: '气血', w: 90 }, { k: 'atk', n: '攻', w: 70 }, { k: 'def', n: '防', w: 70 },
    { k: 'take', n: '台词' }
  ],
  pets: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'level', n: '等级', w: 70 },
    { k: 'max_hp', n: '气血', w: 90 }, { k: 'atk', n: '攻', w: 70 }, { k: 'mg', n: '魔', w: 70 },
    { k: 'def', n: '防', w: 70 }
  ],
  titles: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 150 }, { k: 'hp', n: '气血', w: 70 },
    { k: 'atk', n: '攻', w: 60 }, { k: 'def', n: '防', w: 60 }, { k: 'mg', n: '魔', w: 60 },
    { k: 'desc', n: '描述' }
  ],
  spawns: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'npc_id', n: 'NPC', w: 70 }, { k: 'dtx', n: 'X', w: 60 },
    { k: 'dty', n: 'Y', w: 60 }, { k: 'difficulty', n: '难度', w: 80 }
  ],
  mapnpcs: [
    { k: 'id', n: 'ID', w: 70 }, { k: 'name', n: '名称', w: 120 }, { k: 'dtx', n: 'X', w: 60 },
    { k: 'dty', n: 'Y', w: 60 }, { k: 'npc_id', n: '战斗NPC', w: 80 }, { k: 'img', n: '图片', w: 100 },
    { k: 'shop', n: '服务', w: 100 }, { k: 'dialogue', n: '对话' }
  ]
}

// 各数据表的编辑表单字段（t: num=数字 input=单行 text=多行；opts=下拉）
const FORMS = {
  items: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'desc', n: '描述', t: 'text', max: 255 },
    { k: 'category', n: '分类', t: 'num', opts: [{ v: 1, n: '1 卷轴秘籍' }, { v: 2, n: '2 宝石' }, { v: 4, n: '4 礼包特殊' }, { v: 5, n: '5 药品食物' }, { v: 6, n: '6 任务剧情' }, { v: 8, n: '8 宝箱' }] },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'price', n: '银两价', t: 'num' },
    { k: 'bean_price', n: '金豆价', t: 'num' },
    { k: 'weight', n: '重量', t: 'num' },
    { k: 'bind', n: '绑定', t: 'num', opts: [{ v: 0, n: '0 不绑定' }, { v: 1, n: '1 绑定' }] },
    { k: 'effect', n: '使用效果', t: 'text', max: 500 }
  ],
  equips: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'desc', n: '描述', t: 'text', max: 255 },
    { k: 'category', n: '部位', t: 'num', opts: [{ v: 1, n: '1 法宝' }, { v: 2, n: '2 坐骑' }, { v: 3, n: '3 武器' }, { v: 4, n: '4 护甲' }, { v: 5, n: '5 头盔' }, { v: 6, n: '6 靴子' }, { v: 7, n: '7 项链' }, { v: 8, n: '8 手镯' }] },
    { k: 'sect', n: '门派', t: 'num', opts: [{ v: 0, n: '0 通用' }, { v: 1, n: '1 将军府' }, { v: 2, n: '2 龙宫' }, { v: 3, n: '3 月宫' }, { v: 4, n: '4 方寸山' }, { v: 5, n: '5 普陀山' }, { v: 6, n: '6 全门派' }, { v: 7, n: '7 无限制' }] },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'price', n: '银两价', t: 'num' },
    { k: 'bean_price', n: '金豆价', t: 'num' },
    { k: 'weight', n: '重量', t: 'num' },
    { k: 'bind', n: '绑定', t: 'num', opts: [{ v: 0, n: '0 不绑定' }, { v: 1, n: '1 绑定' }] },
    { k: 'hp', n: '气血加成', t: 'num' }, { k: 'atk', n: '攻击加成', t: 'num' },
    { k: 'mg', n: '魔攻加成', t: 'num' }, { k: 'def', n: '防御加成', t: 'num' },
    { k: 'bg', n: '冰攻', t: 'num' }, { k: 'hg', n: '火攻', t: 'num' }, { k: 'lg', n: '雷攻', t: 'num' },
    { k: 'bf', n: '冰防', t: 'num' }, { k: 'hf', n: '火防', t: 'num' }, { k: 'lf', n: '雷防', t: 'num' }
  ],
  npcs: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'kind', n: '类型', t: 'num', opts: [{ v: 0, n: '0 功能NPC' }, { v: 1, n: '1 战斗NPC' }] },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'max_hp', n: '气血上限', t: 'num' }, { k: 'hp', n: '气血', t: 'num' },
    { k: 'max_mp', n: '法力上限', t: 'num' }, { k: 'mp', n: '法力', t: 'num' },
    { k: 'atk', n: '攻击', t: 'num' }, { k: 'mg', n: '魔攻', t: 'num' },
    { k: 'def', n: '防御', t: 'num' }, { k: 'mf', n: '魔防', t: 'num' },
    { k: 'bg', n: '冰攻', t: 'num' }, { k: 'hg', n: '火攻', t: 'num' }, { k: 'lg', n: '雷攻', t: 'num' },
    { k: 'bf', n: '冰防', t: 'num' }, { k: 'hf', n: '火防', t: 'num' }, { k: 'lf', n: '雷防', t: 'num' },
    { k: 'exp_reward', n: '经验奖励', t: 'num' }, { k: 'money_reward', n: '银两奖励', t: 'num' },
    { k: 'take', n: '被打语', t: 'input', max: 255 },
    { k: 'drops', n: '战斗掉落', t: 'text', max: 1000 }
  ],
  skills: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'desc', n: '描述', t: 'text', max: 255 },
    { k: 'category', n: '类型', t: 'num', opts: [{ v: 1, n: '1 普攻' }, { v: 2, n: '2 门派攻击' }, { v: 3, n: '3 捕捉' }, { v: 4, n: '4 查看' }] },
    { k: 'sect', n: '门派', t: 'num', opts: [{ v: 0, n: '0 通用' }, { v: 1, n: '1 将军府' }, { v: 2, n: '2 龙宫' }, { v: 3, n: '3 月宫' }, { v: 4, n: '4 方寸山' }, { v: 5, n: '5 普陀山' }] },
    { k: 'mp_cost', n: '耗蓝', t: 'num' },
    { k: 'multiplier', n: '威力%', t: 'num' },
    { k: 'learn_level', n: '可学等级', t: 'num' }
  ],
  maps: [
    { k: 'name', n: '节点名', t: 'input', req: true, max: 50 },
    { k: 'desc', n: '描述', t: 'text', max: 255 },
    { k: 'dtx', n: 'X(区域)', t: 'num' },
    { k: 'dty', n: 'Y(节点)', t: 'num' },
    { k: 'up', n: '北出口(如1_2)', t: 'input', max: 20 },
    { k: 'down', n: '南出口', t: 'input', max: 20 },
    { k: 'left', n: '西出口', t: 'input', max: 20 },
    { k: 'right', n: '东出口', t: 'input', max: 20 },
    { k: 'up_jump', n: '北传送', t: 'input', max: 20 },
    { k: 'down_jump', n: '南传送', t: 'input', max: 20 },
    { k: 'left_jump', n: '西传送', t: 'input', max: 20 },
    { k: 'right_jump', n: '东传送', t: 'input', max: 20 }
  ],
  bosses: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'max_hp', n: '气血上限', t: 'num' }, { k: 'hp', n: '气血', t: 'num' },
    { k: 'max_mp', n: '法力上限', t: 'num' }, { k: 'mp', n: '法力', t: 'num' },
    { k: 'atk', n: '攻击', t: 'num' }, { k: 'mg', n: '魔攻', t: 'num' },
    { k: 'def', n: '防御', t: 'num' }, { k: 'mf', n: '魔防', t: 'num' },
    { k: 'bg', n: '冰攻', t: 'num' }, { k: 'hg', n: '火攻', t: 'num' }, { k: 'lg', n: '雷攻', t: 'num' },
    { k: 'bf', n: '冰防', t: 'num' }, { k: 'hf', n: '火防', t: 'num' }, { k: 'lf', n: '雷防', t: 'num' },
    { k: 'take', n: '被打语', t: 'input', max: 255 }
  ],
  pets: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'level', n: '等级', t: 'num' },
    { k: 'star', n: '星级', t: 'num' },
    { k: 'quality', n: '品质(1-4)', t: 'num' },
    { k: 'max_hp', n: '气血上限', t: 'num' }, { k: 'hp', n: '气血', t: 'num' },
    { k: 'max_mp', n: '法力上限', t: 'num' }, { k: 'mp', n: '法力', t: 'num' },
    { k: 'atk', n: '攻击', t: 'num' }, { k: 'mg', n: '魔攻', t: 'num' },
    { k: 'def', n: '防御', t: 'num' }, { k: 'mf', n: '魔防', t: 'num' }
  ],
  titles: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'desc', n: '描述', t: 'text', max: 255 },
    { k: 'hp', n: '气血加成', t: 'num' }, { k: 'atk', n: '攻击加成', t: 'num' },
    { k: 'def', n: '防御加成', t: 'num' }, { k: 'mg', n: '魔攻加成', t: 'num' }
  ],
  spawns: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'npc_id', n: 'NPC ID', t: 'num' },
    { k: 'dtx', n: 'X(区域)', t: 'num' },
    { k: 'dty', n: 'Y(节点,0=随机池)', t: 'num' },
    { k: 'difficulty', n: '难度', t: 'input', max: 10 }
  ],
  mapnpcs: [
    { k: 'name', n: '名称', t: 'input', req: true, max: 50 },
    { k: 'dtx', n: 'X(区域)', t: 'num' },
    { k: 'dty', n: 'Y(节点)', t: 'num' },
    { k: 'npc_id', n: '战斗NPC ID(0无)', t: 'num' },
    { k: 'img', n: '图片文件名', t: 'input', max: 50 },
    { k: 'dialogue', n: '对话', t: 'text', max: 255 },
    { k: 'shop', n: '服务类型', t: 'select', opts: [{ v: '', n: '无服务' }, { v: 'medicine', n: '药店' }, { v: 'weapon', n: '武器店' }, { v: 'armor', n: '防具店' }, { v: 'jewel', n: '首饰店' }, { v: 'grocery', n: '杂货店' }, { v: 'pet', n: '宠物店' }, { v: 'bank', n: '银行' }, { v: 'warehouse', n: '仓库' }, { v: 'rest', n: '客栈休息' }] },
    { k: 'teles', n: '传送目的地', t: 'text', max: 1000 }
  ]
}

// 物品效果可编辑数值项（对应存储键）
const EFF_NUMS = [
  { k: 'hp', n: '恢复气血' }, { k: 'mp', n: '恢复法力' },
  { k: 'maxhp', n: '永久+气血' }, { k: 'maxmp', n: '永久+法力' },
  { k: 'atk', n: '永久+攻击' }, { k: 'def', n: '永久+防御' }, { k: 'mg', n: '永久+魔攻' },
  { k: 'daily', n: '每日限用' }, { k: 'vip', n: 'VIP祝福(分钟)' }, { k: 'beans', n: '获得金豆' }
]

export default {
  name: 'AdminXyData',
  data () {
    return {
      tables: [
        { k: 'items', n: '物品' }, { k: 'equips', n: '装备' }, { k: 'npcs', n: 'NPC' },
        { k: 'skills', n: '技能' }, { k: 'maps', n: '地图节点' }, { k: 'bosses', n: 'BOSS' },
        { k: 'pets', n: '宠物种族' }, { k: 'titles', n: '头衔' }, { k: 'spawns', n: '刷怪点' },
        { k: 'mapnpcs', n: '功能NPC' }
      ],
      table: 'items', word: '',
      rows: [], total: 0, page: 1, size: 10, loading: false,
      showForm: false, saving: false,
      form: {}, formId: 0,
      dropRows: [],
      effNums: EFF_NUMS,
      eff: {},
      teleRows: []
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
      api.get('/admin/xy-data/' + this.table, { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
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
      this.dropRows = []
      this.teleRows = []
      this.eff = this.blankEff()
      this.showForm = true
    },
    openEdit (row) {
      const f = {}
      this.formFields.forEach(fd => { f[fd.k] = row[fd.k] === null || row[fd.k] === undefined ? (fd.t === 'num' ? 0 : '') : row[fd.k] })
      this.form = f
      this.formId = row.id
      this.loadDrops(row.drops)
      this.loadEffect(row.effect)
      this.loadTeles(row.teles)
      this.showForm = true
    },
    // ---------- 物品使用效果结构化编辑 ----------
    // 存储为扁平JSON {"hp":50,"daily":5,"full":1,"goto":"2_1"}，编辑时拆为数值/勾选
    blankEff () {
      const e = {}
      this.effNums.forEach(x => { e[x.k] = 0 })
      e.full = false; e.box = false; e.teleport = false; e.skill_sect = false
      e.skill = 0; e.goto_x = 0; e.goto_y = 0
      return e
    },
    loadEffect (str) {
      let obj = {}
      try { obj = JSON.parse(str || '{}') } catch (e) { obj = {} }
      const e = this.blankEff()
      this.effNums.forEach(x => { if (obj[x.k]) e[x.k] = Number(obj[x.k]) || 0 })
      e.full = obj.full === 1; e.box = obj.box === 1
      e.teleport = obj.teleport === 1; e.skill_sect = obj.skill_sect === 1
      if (obj.skill) e.skill = Number(obj.skill) || 0
      if (obj.goto) {
        const p = String(obj.goto).split('_')
        e.goto_x = Number(p[0]) || 0
        e.goto_y = Number(p[1]) || 0
      }
      this.eff = e
    },
    serializeEffect () {
      const e = this.eff
      const out = {}
      this.effNums.forEach(x => { if (e[x.k] > 0) out[x.k] = e[x.k] })
      if (e.full) out.full = 1
      if (e.box) out.box = 1
      if (e.teleport) out.teleport = 1
      if (e.skill_sect) out.skill_sect = 1
      if (e.skill > 0) out.skill = e.skill
      if (e.goto_x > 0 || e.goto_y > 0) out.goto = e.goto_x + '_' + e.goto_y
      const keys = Object.keys(out)
      return keys.length ? JSON.stringify(out) : ''
    },
    // ---------- 功能NPC传送目的地结构化编辑 ----------
    // 存储为 [{"name":"龙宫","dtx":2,"dty":1}]
    loadTeles (str) {
      let arr = []
      try { arr = JSON.parse(str || '[]') } catch (e) { arr = [] }
      this.teleRows = arr.filter(t => t && (t.name || t.dtx > 0 || t.dty > 0)).map(t => ({
        pick: '', name: t.name || '', dtx: Number(t.dtx) || 0, dty: Number(t.dty) || 0, opts: [], loading: false
      }))
    },
    addTele () {
      this.teleRows.push({ pick: '', name: '', dtx: 0, dty: 0, opts: [], loading: false })
    },
    searchNode (w, i) {
      const row = this.teleRows[i]
      if (!row) return
      if (!w || !String(w).trim()) { row.opts = []; return }
      row.loading = true
      api.get('/admin/xy-data/maps', { params: { word: w, size: 20 } }).then(r => {
        row.loading = false
        if (r.code === 0) {
          row.opts = r.data.list.map(m => ({ v: m.dtx + '_' + m.dty, n: m.name + ' (' + m.dtx + '_' + m.dty + ')' }))
        }
      })
    },
    pickNode (v, i) {
      const row = this.teleRows[i]
      if (!row || !v) return
      const opt = row.opts.find(o => o.v === v)
      const p = String(v).split('_')
      row.dtx = Number(p[0]) || 0
      row.dty = Number(p[1]) || 0
      if (opt) row.name = opt.n.split(' (')[0]
    },
    serializeTeles () {
      const arr = this.teleRows
        .filter(t => t.name || t.dtx > 0 || t.dty > 0)
        .map(t => ({ name: t.name, dtx: t.dtx, dty: t.dty }))
      return arr.length ? JSON.stringify(arr) : ''
    },
    // ---------- NPC掉落结构化编辑 ----------
    // 存储格式 [{"type":"item/equip","id":1,"rate":万分比}]，编辑时换算为百分比
    loadDrops (str) {
      let arr = []
      try { arr = JSON.parse(str || '[]') } catch (e) { arr = [] }
      this.dropRows = []
      for (const d of arr) {
        if (!d || !d.id) continue
        const row = { type: d.type === 'equip' ? 'equip' : 'item', id: Number(d.id), pct: Math.round((d.rate || 0) / 100), opts: [], loading: false }
        this.dropRows.push(row)
        // 回显名称：按ID查一次
        api.get('/admin/xy-data/' + row.type, { params: { word: row.id, size: 1 } }).then(r => {
          if (r.code === 0 && r.data.list && r.data.list.length) {
            const it = r.data.list[0]
            row.opts = [{ v: it.id, n: it.name + ' (#' + it.id + ')' }]
          }
        })
      }
    },
    addDrop () {
      this.dropRows.push({ type: 'item', id: 0, pct: 100, opts: [], loading: false })
    },
    searchDrop (w, i) {
      const row = this.dropRows[i]
      if (!row) return
      if (!w || !String(w).trim()) { row.opts = []; return }
      row.loading = true
      api.get('/admin/xy-data/' + row.type, { params: { word: w, size: 20 } }).then(r => {
        row.loading = false
        if (r.code === 0) row.opts = r.data.list.map(it => ({ v: it.id, n: it.name + ' (#' + it.id + ')' }))
      })
    },
    serializeDrops () {
      const arr = this.dropRows.filter(d => d.id).map(d => ({ type: d.type, id: d.id, rate: Math.round(d.pct * 100) }))
      return arr.length ? JSON.stringify(arr) : ''
    },
    doSave () {
      const reqField = this.formFields.find(f => f.req)
      if (reqField && !String(this.form[reqField.k] || '').trim()) {
        this.$message.warning('请填写' + reqField.n)
        return
      }
      if (this.table === 'npcs') this.form.drops = this.serializeDrops()
      if (this.table === 'items') this.form.effect = this.serializeEffect()
      if (this.table === 'mapnpcs') this.form.teles = this.serializeTeles()
      this.saving = true
      const p = this.formId
        ? api.put('/admin/xy-data/' + this.table + '/' + this.formId, this.form)
        : api.post('/admin/xy-data/' + this.table, this.form)
      p.then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.showForm = false; this.load() }
        else this.$message.error(r.msg)
      })
    },
    doDelete (row) {
      this.$confirm('确认删除「' + (row.name || 'ID' + row.id) + '」？删除后不可恢复！', '删除确认', { type: 'warning' }).then(() => {
        api.delete('/admin/xy-data/' + this.table + '/' + row.id).then(r => {
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
.drop-editor { width: 100%; }
.drop-row { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.drop-row .pct { color: #909399; }
.drop-editor em { font-size: 12px; color: #909399; }
.eff-grid { display: flex; flex-wrap: wrap; gap: 8px 12px; margin-bottom: 10px; }
.eff-cell { display: flex; align-items: center; gap: 6px; }
.eff-label { font-size: 12px; color: #606266; white-space: nowrap; }
.eff-checks { display: flex; flex-wrap: wrap; gap: 4px 16px; margin-bottom: 10px; }
</style>
