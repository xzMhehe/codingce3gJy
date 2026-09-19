<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- ============ 1. 军官列表（总览） ============ -->
        <el-tab-pane label="军官列表" name="overview">
          <div class="toolbar">
            <span class="td-sub">军官体系共 7 张表：3 张配置表 + 4 张玩家实例表。下面是当前数据量概览。</span>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadOverview">刷新</el-button>
          </div>
          <el-row :gutter="14" v-loading="loadingOv" class="ov-row">
            <el-col :span="6" v-for="c in ovCards" :key="c.label">
              <div class="stat-card ov-card" @click="goTab(c.tab)">
                <div class="ov-num">{{ fmtN(c.val) }}</div>
                <div class="ov-label">{{ c.label }}</div>
                <div class="ov-hint">{{ c.hint }}</div>
              </div>
            </el-col>
          </el-row>
          <div class="sub-title">七张表一览</div>
          <el-table :data="ovTables" size="mini" border stripe>
            <el-table-column prop="tab" label="页签" width="170">
              <template slot-scope="{row}">
                <el-link type="primary" @click="goTab(row.key)">{{ row.tab }}</el-link>
              </template>
            </el-table-column>
            <el-table-column prop="table" label="数据表" width="200">
              <template slot-scope="{row}"><span class="td-mono">{{ row.table }}</span></template>
            </el-table-column>
            <el-table-column prop="kind" label="性质" width="110" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.kind === '配置表' ? 'warning' : 'success'">{{ row.kind }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="count" label="记录数" width="110" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.count) }}</span></template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="240" show-overflow-tooltip />
          </el-table>
        </el-tab-pane>

        <!-- ============ 2. 名将列表（配置表，可增删改 + 分发） ============ -->
        <el-tab-pane label="名将列表" name="generals">
          <div class="toolbar">
            <el-input v-model="gWord" placeholder="名将名 / ID" clearable style="width:200px"
                      @keyup.enter.native="loadGenerals" />
            <el-button type="primary" icon="el-icon-search" @click="loadGenerals">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openGeneralCreate">新增名将</el-button>
            <el-button type="warning" icon="el-icon-present" @click="openGrant">分发给玩家</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadGenerals">刷新</el-button>
          </div>
          <el-table :data="gPaged" v-loading="loadingG" stripe border>
            <el-table-column prop="id" label="ID" width="45" align="center" />
            <el-table-column prop="name" label="名将" min-width="135" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="55" align="center" />
            <el-table-column label="星级" width="75" align="center">
              <template slot-scope="{row}">{{ '★'.repeat(row.star) }}</template>
            </el-table-column>
            <el-table-column prop="military" label="武力" width="55" align="center" />
            <el-table-column prop="logistics" label="后勤" width="55" align="center" />
            <el-table-column prop="learning" label="学识" width="55" align="center" />
            <el-table-column label="可招募" width="68" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.recruit === 1 ? 'success' : 'info'">{{ row.recruit === 1 ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="拥有玩家" width="80" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.owned_count }}</span></template>
            </el-table-column>
            <el-table-column prop="source" label="来源" min-width="135" show-overflow-tooltip />
            <el-table-column prop="skill" label="组合技" min-width="135" show-overflow-tooltip />
            <el-table-column label="操作" width="190" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="warning" plain icon="el-icon-present" title="分发给玩家" @click="openGrant(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openGeneralEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delGeneral(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ generals.length }}</b> 条 · 每页 {{ gSize }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="generals.length"
                           :page-size="gSize" :current-page="gPage" :page-sizes="[20, 50, 100]"
                           @current-change="p => { gPage = p }"
                           @size-change="s => { gSize = s; gPage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- ============ 3. 玩家军官列表（ezfy_officer） ============ -->
        <el-tab-pane label="玩家军官列表" name="officers">
          <div class="toolbar">
            <el-input v-model="word" placeholder="军官名 / 城名 / 城池ID" clearable style="width:220px"
                      @keyup.enter.native="page = 1; load()" />
            <el-select v-model="captive" style="width:130px" @change="page = 1; load()">
              <el-option label="全部军官" :value="-1" />
              <el-option label="仅俘虏" :value="1" />
              <el-option label="非俘虏" :value="0" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-magic-stick" @click="openGen">一键生成军官</el-button>
            <el-button type="warning" icon="el-icon-present" @click="openGrant">发放名将</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
          </div>
          <el-table :data="list" v-loading="loading" stripe border max-height="600">
            <el-table-column prop="id" label="ID" width="45" align="center" />
            <el-table-column prop="name" label="姓名" width="100" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="星级" width="65" align="center">
              <template slot-scope="{row}">{{ '★'.repeat(row.star) }}</template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="48" align="center" />
            <el-table-column label="武力/后勤/学识" width="120" align="center">
              <template slot-scope="{row}">
                <span class="td-mono">{{ row.military }} / {{ row.logistics }} / {{ row.learning }}</span>
              </template>
            </el-table-column>
            <el-table-column label="忠诚" width="58" align="center">
              <template slot-scope="{row}">
                <span :class="row.loyalty < 30 ? 'td-danger' : ''">{{ row.loyalty }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="pos_name" label="任命" width="58" align="center" />
            <el-table-column label="状态" width="68" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.is_captive === 1 ? 'danger' : 'success'">
                  {{ row.is_captive === 1 ? '俘虏' : row.status_name }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="city_name" label="归属城池" width="100" show-overflow-tooltip />
            <el-table-column label="归属玩家" width="130" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span class="td-main">{{ row.owner_name || '—' }}</span>
                <span class="td-muted">（{{ row.home_num || '—' }}）</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
                <el-button size="mini" type="warning" plain :icon="row.is_captive === 1 ? 'el-icon-unlock' : 'el-icon-lock'"
                           :title="row.is_captive === 1 ? '释放' : '设为俘虏'" @click="toggleCaptive(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="解雇" @click="del(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                           :current-page="page" :page-sizes="[15, 30, 50]"
                           @current-change="p => { page = p; load() }"
                           @size-change="s => { size = s; page = 1; load() }" />
          </div>
        </el-tab-pane>

        <!-- ============ 4. 军官技能列表（ezfy_cfg_skill） ============ -->
        <el-tab-pane label="军官技能列表" name="skills">
          <div class="toolbar">
            <el-input v-model="sWord" placeholder="技能名 / 效果 / ID" clearable style="width:220px"
                      @keyup.enter.native="loadSkills" />
            <el-button type="primary" icon="el-icon-search" @click="loadSkills">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openSkillCreate">新增技能</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadSkills">刷新</el-button>
          </div>
          <el-table :data="sPaged" v-loading="loadingS" stripe border>
            <el-table-column prop="id" label="ID" width="45" align="center" />
            <el-table-column prop="name" label="技能名" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="85" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="skillTag(row.type)">{{ row.type_name }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="effect" label="效果" min-width="220" show-overflow-tooltip />
            <el-table-column label="使用军官数" width="100" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.use_count }}</span></template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="200" show-overflow-tooltip />
            <el-table-column label="操作" width="140" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openSkillEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delSkill(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ skills.length }}</b> 条 · 每页 {{ sSize }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="skills.length"
                           :page-size="sSize" :current-page="sPage" :page-sizes="[20, 50, 100]"
                           @current-change="p => { sPage = p }"
                           @size-change="s => { sSize = s; sPage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- ============ 5. 玩家军官技能列表 ============ -->
        <el-tab-pane label="玩家军官技能列表" name="ownedSkills">
          <div class="toolbar">
            <el-input v-model="osWord" placeholder="军官名 / 军官ID / 城池ID" clearable style="width:220px"
                      @keyup.enter.native="osPage = 1; loadOwnedSkills()" />
            <el-select v-model="osSkill" style="width:150px" filterable clearable placeholder="按技能筛选"
                       @change="osPage = 1; loadOwnedSkills()">
              <el-option v-for="s in skills" :key="s.name" :label="s.name" :value="s.name" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="osPage = 1; loadOwnedSkills()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openSkillAssign">给军官加技能</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadOwnedSkills">刷新</el-button>
          </div>
          <el-table :data="ownedSkills" v-loading="loadingOS" stripe border max-height="600">
            <el-table-column prop="officer_id" label="军官ID" width="70" align="center" />
            <el-table-column prop="officer_name" label="军官" width="125" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.officer_name }}</span></template>
            </el-table-column>
            <el-table-column prop="skill_name" label="技能" width="110" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-blue">{{ row.skill_name }}</span></template>
            </el-table-column>
            <el-table-column prop="effect" label="效果" min-width="180" show-overflow-tooltip />
            <el-table-column prop="city_name" label="归属城池" width="100" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" width="130" show-overflow-tooltip />
            <el-table-column label="操作" width="90" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="遗忘" @click="removeSkill(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ osTotal }}</b> 条 · 每页 {{ osSize }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="osTotal" :page-size="osSize"
                           :current-page="osPage" :page-sizes="[15, 30, 50]"
                           @current-change="p => { osPage = p; loadOwnedSkills() }"
                           @size-change="s => { osSize = s; osPage = 1; loadOwnedSkills() }" />
          </div>
        </el-tab-pane>

        <!-- ============ 6. 军官装备列表（ezfy_cfg_equipment） ============ -->
        <el-tab-pane label="军官装备列表" name="equips">
          <div class="toolbar">
            <el-input v-model="eWord" placeholder="装备名 / 类型 / ID" clearable style="width:220px"
                      @keyup.enter.native="loadEquips" />
            <el-button type="primary" icon="el-icon-search" @click="loadEquips">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openEquipCreate">新增装备</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadEquips">刷新</el-button>
          </div>
          <el-table :data="ePaged" v-loading="loadingE" stripe border>
            <el-table-column prop="id" label="ID" width="45" align="center" />
            <el-table-column prop="name" label="装备名" min-width="140" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="70" align="center" />
            <el-table-column label="品质" width="70" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="tierTag(row.tier)">{{ row.tier_name }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="military" label="军事" width="60" align="center" />
            <el-table-column prop="logistics" label="后勤" width="55" align="center" />
            <el-table-column prop="learning" label="学识" width="55" align="center" />
            <el-table-column prop="level" label="需求等级" width="80" align="center" />
            <el-table-column label="持有数" width="75" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.owned_count }}</span></template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="180" show-overflow-tooltip />
            <el-table-column label="操作" width="140" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEquipEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delEquip(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ equips.length }}</b> 条 · 每页 {{ eSize }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="equips.length"
                           :page-size="eSize" :current-page="ePage" :page-sizes="[20, 50, 100]"
                           @current-change="p => { ePage = p }"
                           @size-change="s => { eSize = s; ePage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- ============ 7. 玩家军官装备列表（ezfy_equipment） ============ -->
        <el-tab-pane label="玩家军官装备列表" name="ownedEquips">
          <div class="toolbar">
            <el-input v-model="oeWord" placeholder="装备名 / 装备ID / 军官ID / 城池ID" clearable style="width:240px"
                      @keyup.enter.native="oePage = 1; loadOwnedEquips()" />
            <el-select v-model="oeType" style="width:120px" clearable placeholder="类型"
                       @change="oePage = 1; loadOwnedEquips()">
              <el-option label="武器" value="武器" />
              <el-option label="防具" value="防具" />
              <el-option label="饰品" value="饰品" />
              <el-option label="珠宝" value="珠宝" />
            </el-select>
            <el-select v-model="oeEquipped" style="width:130px" @change="oePage = 1; loadOwnedEquips()">
              <el-option label="全部" :value="-1" />
              <el-option label="已穿戴" :value="1" />
              <el-option label="未穿戴" :value="0" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="oePage = 1; loadOwnedEquips()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-present" @click="openEquipGrant">给玩家发装备</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadOwnedEquips">刷新</el-button>
          </div>
          <el-table :data="ownedEquips" v-loading="loadingOE" stripe border max-height="600">
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column prop="name" label="装备名" width="120" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="65" align="center" />
            <el-table-column label="品质" width="65" align="center">
              <template slot-scope="{row}">{{ row.tier_name }}</template>
            </el-table-column>
            <el-table-column label="需求等级" width="75" align="center">
              <template slot-scope="{row}">{{ row.level }}</template>
            </el-table-column>
            <el-table-column prop="owner_name" label="持有玩家" width="140" show-overflow-tooltip />
            <el-table-column prop="city_name" label="归属城池" width="110" show-overflow-tooltip />
            <el-table-column label="穿戴军官" width="110" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.officer_id > 0" class="td-blue">{{ row.officer_name }}（{{ row.officer_id }}）</span>
                <span v-else class="td-muted">未穿戴</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑/穿戴" @click="openEquipOwnedEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delEquipOwned(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ oeTotal }}</b> 条 · 每页 {{ oeSize }} 条</div>
            <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="oeTotal" :page-size="oeSize"
                           :current-page="oePage" :page-sizes="[15, 30, 50]"
                           @current-change="p => { oePage = p; loadOwnedEquips() }"
                           @size-change="s => { oeSize = s; oePage = 1; loadOwnedEquips() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- ============ 编辑军官 ============ -->
    <el-dialog title="编辑军官" :visible.sync="editDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="姓名"><el-input v-model="form.name" maxlength="100" style="width:240px" /></el-form-item>
        <el-form-item label="等级 / 星级">
          <el-input-number v-model.number="form.level" :min="1" controls-position="right" style="width:130px" />
          <el-input-number v-model.number="form.star" :min="1" :max="10" controls-position="right" style="width:130px;margin-left:8px" />
        </el-form-item>
        <el-form-item label="武力 / 后勤 / 学识">
          <el-input-number v-model.number="form.military" :min="0" controls-position="right" style="width:120px" />
          <el-input-number v-model.number="form.logistics" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
          <el-input-number v-model.number="form.learning" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
        </el-form-item>
        <el-form-item label="忠诚">
          <el-input-number v-model.number="form.loyalty" :min="0" :max="100" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">归零会离职</span>
        </el-form-item>
        <el-form-item label="任命">
          <el-radio-group v-model="form.position">
            <el-radio :label="0">无</el-radio>
            <el-radio :label="1">市长</el-radio>
            <el-radio :label="2">城守</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :label="0">在职</el-radio>
            <el-radio :label="1">出征中</el-radio>
            <el-radio :label="2">被俘</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 一键生成军官 ============ -->
    <el-dialog title="一键生成军官" :visible.sync="genDlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="120px" size="small">
        <el-form-item label="归属玩家" required>
          <el-input-number v-model.number="genForm.user_id" :min="1" controls-position="right" style="width:100%" />
          <span class="td-sub">军官会挂在该玩家的主城下</span>
        </el-form-item>
        <el-form-item label="生成数量">
          <el-input-number v-model.number="genForm.count" :min="1" :max="20" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="等级上限">
          <el-input-number v-model.number="genForm.max_level" :min="1" :max="200" controls-position="right" style="width:100%" />
        </el-form-item>
      </el-form>
      <em>
        随机生成名字 / 等级 / 星级；<b>属性上限取自同星级名将的最大值</b>，保证不会超过名将。
      </em>
      <div slot="footer">
        <el-button @click="genDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGen">生 成</el-button>
      </div>
    </el-dialog>

    <!-- ============ 发放名将 ============ -->
    <el-dialog title="发放名将" :visible.sync="grantDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="玩家ID" required>
          <el-input-number v-model.number="grantForm.user_id" :min="1" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">即用户ID（不是家园号）</span>
        </el-form-item>
        <el-form-item label="名将" required>
          <el-select v-model="grantForm.general_id" filterable style="width:280px">
            <el-option v-for="g in generals" :key="g.id"
                       :label="g.id + ' · ' + g.name + '（' + '★'.repeat(g.star) + '）'" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <em>提示：同一名将不能重复发放给同一玩家</em>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增/编辑 名将 ============ -->
    <el-dialog :title="gf.id ? ('编辑名将 · ' + gf.name) : '新增名将'" :visible.sync="gDlg"
               width="900px" :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="名将名称" required>
              <el-input v-model="gf.name" maxlength="100" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="等级(强度)">
              <el-input-number v-model.number="gf.level" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="武力"><el-input-number v-model.number="gf.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="后勤"><el-input-number v-model.number="gf.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="学识"><el-input-number v-model.number="gf.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="星级">
              <el-input-number v-model.number="gf.star" :min="1" :max="10" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="可入军校候选">
              <el-switch v-model="gf.recruit" :active-value="1" :inactive-value="0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="获取渠道">
          <el-input v-model="gf.source" maxlength="255" placeholder="例如：10亿金子 / 商店购买" />
        </el-form-item>
        <el-form-item label="获取条件">
          <el-input v-model="gf.get_condition" maxlength="255" placeholder="例如：任务 / 505" />
        </el-form-item>
        <el-form-item label="组合技/技能">
          <el-input v-model="gf.skill" maxlength="500" placeholder="例如：全技能+2，攻击+20%" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="gf.des" type="textarea" :rows="2" maxlength="500" />
        </el-form-item>
      </el-form>
      <em>保存后立即生效（后端会重载配置缓存，不用重启）</em>
      <div slot="footer">
        <el-button @click="gDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGeneralSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增/编辑 技能 ============ -->
    <el-dialog :title="sf.id ? ('编辑技能 · ' + sf.name) : '新增技能'" :visible.sync="sDlg"
               width="620px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="技能名称" required>
          <el-input v-model="sf.name" maxlength="50" style="width:240px" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model.number="sf.type">
            <el-radio :label="1">攻击类</el-radio>
            <el-radio :label="2">防御类</el-radio>
            <el-radio :label="3">辅助类</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="效果">
          <el-input v-model="sf.effect" maxlength="100" placeholder="例如：攻击+10%" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="sf.des" type="textarea" :rows="3" maxlength="500" />
        </el-form-item>
      </el-form>
      <em v-if="sf.id">改名会同步更新所有已学该技能的军官</em>
      <em v-else>新技能会立刻出现在玩家军校的技能学习列表里</em>
      <div slot="footer">
        <el-button @click="sDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSkillSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 给军官加技能 ============ -->
    <el-dialog title="给军官加技能" :visible.sync="saDlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="军官" required>
          <el-select v-model="saForm.officer_id" filterable style="width:320px" placeholder="按军官ID / 姓名搜索">
            <el-option v-for="o in pickers.officers" :key="o.id"
                       :label="o.id + ' · ' + o.name + '（城池' + o.city_id + '，已有' + o.skill_count + '个技能）'"
                       :value="o.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="技能" required>
          <el-select v-model="saForm.skill_name" filterable style="width:320px" placeholder="选择技能">
            <el-option v-for="s in pickers.skills" :key="s.name" :label="s.name + '（' + s.effect + '）'" :value="s.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <em>管理端加技能不消耗黄金、不受 3 个技能上限限制</em>
      <div slot="footer">
        <el-button @click="saDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSkillAssign">确 定</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增/编辑 装备配置 ============ -->
    <el-dialog :title="ef.id ? ('编辑装备 · ' + ef.name) : '新增装备'" :visible.sync="eDlg"
               width="880px" :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="装备名称" required>
              <el-input v-model="ef.name" maxlength="50" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="类型">
              <el-select v-model="ef.type" style="width:100%">
                <el-option label="武器" value="武器" />
                <el-option label="防具" value="防具" />
                <el-option label="饰品" value="饰品" />
                <el-option label="珠宝" value="珠宝" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="品质">
              <el-select v-model.number="ef.tier" style="width:100%">
                <el-option label="初级" :value="1" />
                <el-option label="中级" :value="2" />
                <el-option label="高级" :value="3" />
                <el-option label="特殊" :value="4" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="穿戴等级需求">
              <el-input-number v-model.number="ef.level" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="军事加成"><el-input-number v-model.number="ef.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="后勤加成"><el-input-number v-model.number="ef.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="学识加成"><el-input-number v-model.number="ef.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="说明">
          <el-input v-model="ef.des" maxlength="200" />
        </el-form-item>
      </el-form>
      <em>保存后立即生效（后端会重载配置缓存）</em>
      <div slot="footer">
        <el-button @click="eDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEquipSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 给玩家发装备 ============ -->
    <el-dialog title="给玩家发装备" :visible.sync="egDlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="玩家ID" required>
          <el-input-number v-model.number="egForm.user_id" :min="1" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">即用户ID（不是家园号）</span>
        </el-form-item>
        <el-form-item label="装备" required>
          <el-select v-model="egForm.cfg_id" filterable style="width:320px">
            <el-option v-for="e in equips" :key="e.id"
                       :label="e.id + ' · ' + e.name + '（' + e.type + '·' + e.tier_name + '，需求Lv' + e.level + '）'"
                       :value="e.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model.number="egForm.count" :min="1" :max="50" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">最多 50 件</span>
        </el-form-item>
      </el-form>
      <em>装备会进入玩家主城背包，未穿戴</em>
      <div slot="footer">
        <el-button @click="egDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEquipGrant">发 放</el-button>
      </div>
    </el-dialog>

    <!-- ============ 编辑玩家装备 / 穿戴 ============ -->
    <el-dialog :title="'编辑玩家装备 · ' + oef.name" :visible.sync="oeDlg" width="880px" :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-form-item label="持有玩家">
          <span class="td-main">{{ oef.owner_name || '—' }}</span>
          <span class="td-muted">（{{ oef.city_name }}）</span>
        </el-form-item>
        <el-form-item label="装备名称">
          <el-input v-model="oef.name" maxlength="50" style="width:240px" />
        </el-form-item>
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="类型">
              <el-select v-model="oef.type" style="width:100%">
                <el-option label="武器" value="武器" />
                <el-option label="防具" value="防具" />
                <el-option label="饰品" value="饰品" />
                <el-option label="珠宝" value="珠宝" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="品质">
              <el-input-number v-model.number="oef.tier" :min="1" :max="4" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="需求等级">
              <el-input-number v-model.number="oef.level" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="军事"><el-input-number v-model.number="oef.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="后勤"><el-input-number v-model.number="oef.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="学识"><el-input-number v-model.number="oef.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="穿戴军官">
          <el-select v-model.number="oef.officer_id" filterable clearable style="width:320px" placeholder="不穿戴（0）">
            <el-option label="不穿戴（0）" :value="0" />
            <el-option v-for="o in ownerOfficers" :key="o.id" :label="o.id + ' · ' + o.name" :value="o.id" />
          </el-select>
          <div class="td-sub" style="margin-top:4px">只能选该玩家自己城池里的军官</div>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="oeDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEquipOwnedSave">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const G_KEYS = ['name', 'level', 'military', 'logistics', 'learning', 'star', 'source', 'get_condition', 'skill', 'des', 'recruit']
const S_KEYS = ['name', 'effect', 'type', 'des']
const E_KEYS = ['name', 'type', 'tier', 'military', 'logistics', 'learning', 'level', 'des']
const OE_KEYS = ['name', 'type', 'tier', 'military', 'logistics', 'learning', 'level', 'officer_id']

export default {
  name: 'AdminEzfyOfficers',
  data () {
    return {
      tab: 'overview',
      saving: false,
      // 1 总览
      ov: {}, loadingOv: false,
      // 2 名将
      generals: [], loadingG: false, gWord: '', gPage: 1, gSize: 20,
      gDlg: false, gf: {},
      // 3 玩家军官
      list: [], total: 0, page: 1, size: 15, loading: false, word: '', captive: -1,
      editDlg: false, editId: 0, form: {},
      grantDlg: false, grantForm: { user_id: 1, general_id: 0 },
      // 4 技能
      skills: [], loadingS: false, sWord: '', sPage: 1, sSize: 20,
      sDlg: false, sf: {},
      // 5 玩家军官技能
      ownedSkills: [], osTotal: 0, osPage: 1, osSize: 15, loadingOS: false,
      osWord: '', osSkill: '',
      saDlg: false, saForm: { officer_id: 0, skill_name: '' },
      // 6 装备
      equips: [], loadingE: false, eWord: '', ePage: 1, eSize: 20,
      eDlg: false, ef: {},
      // 7 玩家装备
      ownedEquips: [], oeTotal: 0, oePage: 1, oeSize: 15, loadingOE: false,
      oeWord: '', oeType: '', oeEquipped: -1,
      egDlg: false, egForm: { user_id: 1, cfg_id: 0, count: 1 },
      oeDlg: false, oef: {}, ownerOfficers: [],
      // 下拉数据
      pickers: { generals: [], skills: [], officers: [] },
      genDlg: false, genForm: { user_id: 10007, count: 3, max_level: 60 }
    }
  },
  computed: {
    // 配置表类列表（后端一次给全）在前端分页
    gPaged () { const st = (this.gPage - 1) * this.gSize; return this.generals.slice(st, st + this.gSize) },
    sPaged () { const st = (this.sPage - 1) * this.sSize; return this.skills.slice(st, st + this.sSize) },
    ePaged () { const st = (this.ePage - 1) * this.eSize; return this.equips.slice(st, st + this.eSize) },
    ovCards () {
      const o = this.ov || {}
      return [
        { label: '名将配置', val: o.generals, hint: 'ezfy_cfg_general', tab: 'generals' },
        { label: '技能配置', val: o.skills, hint: 'ezfy_cfg_skill', tab: 'skills' },
        { label: '装备配置', val: o.equipments, hint: 'ezfy_cfg_equipment', tab: 'equips' },
        { label: '玩家军官', val: o.officers, hint: 'ezfy_officer', tab: 'officers' },
        { label: '已学技能', val: o.owned_skills, hint: '军官技能 JSON 展开', tab: 'ownedSkills' },
        { label: '玩家装备', val: o.owned_equipments, hint: 'ezfy_equipment', tab: 'ownedEquips' },
        { label: '已任命', val: o.appointed, hint: '市长 / 城守', tab: 'officers' },
        { label: '俘虏中', val: o.captives, hint: 'is_captive=1', tab: 'officers' }
      ]
    },
    ovTables () {
      const o = this.ov || {}
      return [
        { key: 'generals', tab: '名将列表', table: 'ezfy_cfg_general', kind: '配置表', count: o.generals, des: '名将（31 名），可增删改、可分发到玩家' },
        { key: 'skills', tab: '军官技能列表', table: 'ezfy_cfg_skill', kind: '配置表', count: o.skills, des: '技能（每名军官最多学 3 个），可增删改' },
        { key: 'equips', tab: '军官装备列表', table: 'ezfy_cfg_equipment', kind: '配置表', count: o.equipments, des: '装备 18 件 + 地形珠宝 8 件，可增删改' },
        { key: 'officers', tab: '玩家军官列表', table: 'ezfy_officer', kind: '实例表', count: o.officers, des: '玩家拥有的军官，含属性/忠诚/任命/俘虏状态' },
        { key: 'ownedSkills', tab: '玩家军官技能列表', table: 'ezfy_officer.skill', kind: '实例表', count: o.owned_skills, des: '把每名军官的 skill JSON 摊平成一行一条技能' },
        { key: 'ownedEquips', tab: '玩家军官装备列表', table: 'ezfy_equipment', kind: '实例表', count: o.owned_equipments, des: '玩家背包里的装备，可穿戴到军官' }
      ]
    }
  },
  watch: {
    gWord () { this.gPage = 1 },
    sWord () { this.sPage = 1 },
    eWord () { this.ePage = 1 }
  },
  mounted () {
    this.loadOverview()
    this.loadGenerals()
  },
  methods: {
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    skillTag (t) {
      return ({ 1: 'danger', 2: 'primary', 3: 'warning' })[t] || 'info'
    },
    tierTag (t) {
      return ({ 1: 'info', 2: 'primary', 3: 'warning', 4: 'danger' })[t] || 'info'
    },
    goTab (k) {
      this.tab = k
      this.onTab()
    },
    onTab () {
      if (this.tab === 'overview') this.loadOverview()
      if (this.tab === 'generals') this.loadGenerals()
      if (this.tab === 'officers') this.load()
      if (this.tab === 'skills') this.loadSkills()
      if (this.tab === 'ownedSkills') this.loadOwnedSkills()
      if (this.tab === 'equips') this.loadEquips()
      if (this.tab === 'ownedEquips') this.loadOwnedEquips()
    },
    // ---------- 1 总览 ----------
    loadOverview () {
      this.loadingOv = true
      api.get('/admin/ezfy-officer-overview').then(r => {
        this.loadingOv = false
        if (r.code === 0) this.ov = r.data
        else this.$message.error(r.msg)
      })
    },
    // ---------- 2 名将 ----------
    loadGenerals () {
      this.loadingG = true
      api.get('/admin/ezfy-generals', { params: { word: this.gWord } }).then(r => {
        this.loadingG = false
        if (r.code === 0) this.generals = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openGeneralCreate () {
      this.gf = {
        name: '', level: 130, military: 100, logistics: 100, learning: 100,
        star: 5, source: '', get_condition: '', skill: '', des: '', recruit: 1
      }
      this.gDlg = true
    },
    openGeneralEdit (row) {
      const f = { id: row.id }
      G_KEYS.forEach(k => { f[k] = row[k] })
      this.gf = f
      this.gDlg = true
    },
    doGeneralSave () {
      if (!String(this.gf.name || '').trim()) { this.$message.warning('名将名称不能为空'); return }
      const isNew = !this.gf.id
      const body = {}
      G_KEYS.forEach(k => { body[k] = this.gf[k] })
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-generals', body) : api.put('/admin/ezfy-generals/' + this.gf.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.gDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadGenerals(); this.loadPickers(); this.loadOverview() }
        else this.$message.error(r.msg)
      })
    },
    delGeneral (row) {
      this.$confirm('删除名将「' + row.name + '」会同时回收 ' + row.owned_count +
        ' 名玩家已拥有的该军官。确认删除？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-generals/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadGenerals(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 3 玩家军官 ----------
    load () {
      this.loading = true
      api.get('/admin/ezfy-officers', {
        params: { page: this.page, size: this.size, word: this.word, captive: this.captive }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openEdit (row) {
      this.editId = row.id
      const keys = ['name', 'level', 'star', 'military', 'logistics', 'learning', 'loyalty', 'position', 'status']
      const f = {}
      keys.forEach(k => { f[k] = row[k] })
      this.form = f
      this.editDlg = true
    },
    doEdit () {
      this.saving = true
      api.put('/admin/ezfy-officers/' + this.editId, this.form).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    toggleCaptive (row) {
      const to = row.is_captive !== 1
      api.post('/admin/ezfy-officers/' + row.id + '/captive', { captive: to }).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已更新'); this.load(); this.loadOverview() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('解雇军官「' + row.name + '」后该军官将从玩家参谋部移除，确认解雇？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-officers/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已解雇'); this.load(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // 一键生成军官（随机名字/等级/星级，属性不超过名将）
    openGen () {
      this.genForm = { user_id: this.genForm.user_id || 10007, count: 3, max_level: 60 }
      this.genDlg = true
    },
    doGen () {
      if (!this.genForm.user_id) { this.$message.warning('请填写归属玩家'); return }
      this.saving = true
      api.post('/admin/ezfy-officers/gen', this.genForm).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.genDlg = false
          this.$message.success(r.data.msg || '已生成')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    openGrant (row) {
      const gid = row && row.id ? row.id : (this.generals.length ? this.generals[0].id : 0)
      this.grantForm = { user_id: 1, general_id: gid }
      this.grantDlg = true
    },
    doGrant () {
      if (!this.grantForm.general_id) { this.$message.warning('请选择名将'); return }
      this.saving = true
      api.post('/admin/ezfy-officers/grant', this.grantForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.grantDlg = false; this.$message.success(r.data.msg || '已发放'); this.load(); this.loadOverview() }
        else this.$message.error(r.msg)
      })
    },
    // ---------- 4 技能 ----------
    loadSkills () {
      this.loadingS = true
      api.get('/admin/ezfy-skills', { params: { word: this.sWord } }).then(r => {
        this.loadingS = false
        if (r.code === 0) this.skills = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openSkillCreate () {
      this.sf = { name: '', effect: '', type: 1, des: '' }
      this.sDlg = true
    },
    openSkillEdit (row) {
      const f = { id: row.id }
      S_KEYS.forEach(k => { f[k] = row[k] })
      this.sf = f
      this.sDlg = true
    },
    doSkillSave () {
      if (!String(this.sf.name || '').trim()) { this.$message.warning('技能名称不能为空'); return }
      const isNew = !this.sf.id
      const body = {}
      S_KEYS.forEach(k => { body[k] = this.sf[k] })
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-skills', body) : api.put('/admin/ezfy-skills/' + this.sf.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.sDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadSkills(); this.loadPickers(); this.loadOverview() }
        else this.$message.error(r.msg)
      })
    },
    delSkill (row) {
      this.$confirm('删除技能「' + row.name + '」会从 ' + row.use_count +
        ' 名军官身上摘除该技能。确认删除？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-skills/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadSkills(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 5 玩家军官技能 ----------
    loadOwnedSkills () {
      this.loadingOS = true
      api.get('/admin/ezfy-officer-skills-owned', {
        params: { page: this.osPage, size: this.osSize, word: this.osWord, skill: this.osSkill }
      }).then(r => {
        this.loadingOS = false
        if (r.code === 0) {
          this.ownedSkills = r.data.list
          this.osTotal = r.data.total
          this.osPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadPickers () {
      api.get('/admin/ezfy-officer-pickers').then(r => {
        if (r.code === 0) this.pickers = r.data
      })
    },
    openSkillAssign () {
      this.loadPickers()
      this.saForm = { officer_id: 0, skill_name: '' }
      this.saDlg = true
    },
    doSkillAssign () {
      if (!this.saForm.officer_id || !this.saForm.skill_name) { this.$message.warning('请选择军官与技能'); return }
      this.saving = true
      api.post('/admin/ezfy-officer-skills-owned', this.saForm).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.saDlg = false
          this.$message.success(r.data.msg || '已添加')
          this.loadOwnedSkills()
          this.loadOverview()
        } else this.$message.error(r.msg)
      })
    },
    removeSkill (row) {
      this.$confirm('让「' + row.officer_name + '」遗忘「' + row.skill_name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-officer-skills-owned', {
          data: { officer_id: row.officer_id, skill_name: row.skill_name }
        }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已遗忘'); this.loadOwnedSkills(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 6 装备配置 ----------
    loadEquips () {
      this.loadingE = true
      api.get('/admin/ezfy-equipments', { params: { word: this.eWord } }).then(r => {
        this.loadingE = false
        if (r.code === 0) this.equips = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openEquipCreate () {
      this.ef = { name: '', type: '武器', tier: 1, military: 0, logistics: 0, learning: 0, level: 1, des: '' }
      this.eDlg = true
    },
    openEquipEdit (row) {
      const f = { id: row.id }
      E_KEYS.forEach(k => { f[k] = row[k] })
      this.ef = f
      this.eDlg = true
    },
    doEquipSave () {
      if (!String(this.ef.name || '').trim()) { this.$message.warning('装备名称不能为空'); return }
      const isNew = !this.ef.id
      const body = {}
      E_KEYS.forEach(k => { body[k] = this.ef[k] })
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-equipments', body) : api.put('/admin/ezfy-equipments/' + this.ef.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.eDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadEquips(); this.loadOverview() }
        else this.$message.error(r.msg)
      })
    },
    delEquip (row) {
      this.$confirm('删除装备配置「' + row.name + '」会同时清掉玩家背包里的 ' + row.owned_count +
        ' 件。确认删除？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-equipments/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadEquips(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 7 玩家装备 ----------
    loadOwnedEquips () {
      this.loadingOE = true
      api.get('/admin/ezfy-equipments-owned', {
        params: {
          page: this.oePage, size: this.oeSize, word: this.oeWord,
          type: this.oeType || '', equipped: this.oeEquipped
        }
      }).then(r => {
        this.loadingOE = false
        if (r.code === 0) {
          this.ownedEquips = r.data.list
          this.oeTotal = r.data.total
          this.oePage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openEquipGrant () {
      if (!this.equips.length) this.loadEquips()
      this.egForm = { user_id: 1, cfg_id: this.equips.length ? this.equips[0].id : 0, count: 1 }
      this.egDlg = true
    },
    doEquipGrant () {
      if (!this.egForm.cfg_id) { this.$message.warning('请选择装备'); return }
      this.saving = true
      api.post('/admin/ezfy-equipments-owned', this.egForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.egDlg = false; this.$message.success(r.data.msg || '已发放'); this.loadOwnedEquips(); this.loadOverview() }
        else this.$message.error(r.msg)
      })
    },
    openEquipOwnedEdit (row) {
      const f = { id: row.id, owner_name: row.owner_name, city_name: row.city_name, user_id: row.user_id }
      OE_KEYS.forEach(k => { f[k] = row[k] })
      this.oef = f
      // 只列出该玩家自己城池里的军官，避免把装备穿到别人的军官上
      this.ownerOfficers = []
      api.get('/admin/ezfy-officers', { params: { page: 1, size: 200, city_id: row.city_id } }).then(r => {
        if (r.code === 0) this.ownerOfficers = r.data.list
      })
      this.oeDlg = true
    },
    doEquipOwnedSave () {
      const body = {}
      OE_KEYS.forEach(k => { body[k] = this.oef[k] })
      this.saving = true
      api.put('/admin/ezfy-equipments-owned/' + this.oef.id, body).then(r => {
        this.saving = false
        if (r.code === 0) { this.oeDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadOwnedEquips() }
        else this.$message.error(r.msg)
      })
    },
    delEquipOwned (row) {
      this.$confirm('确认删除「' + row.owner_name + '」的 ' + row.name + '？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-equipments-owned/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadOwnedEquips(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.td-danger { color: #f56c6c; font-weight: 600; }
.ov-row { margin-bottom: 6px; }
.ov-card {
  display: block; text-align: center; cursor: pointer;
  padding: 16px 10px; margin-bottom: 14px;
}
.ov-num { font-size: 24px; font-weight: 700; color: #1f2d3d; line-height: 1; }
.ov-label { font-size: 13px; color: #5b6b82; margin-top: 7px; font-weight: 600; }
.ov-hint { font-size: 11px; color: #b6c2d2; margin-top: 3px; }
.sub-title {
  font-size: 13px; font-weight: 600; color: #1f2d3d;
  margin: 10px 0 8px; padding-left: 6px; border-left: 3px solid #409eff;
}
</style>
