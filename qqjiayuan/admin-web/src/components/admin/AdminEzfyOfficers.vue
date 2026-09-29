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

        <!-- ============ 2. 军官池（普通军官 + 名将，可增删改 + 分发） ============ -->
        <el-tab-pane label="军官池" name="generals">
          <div class="toolbar">
            <el-radio-group v-model.number="gKind" size="small" @change="gPage = 1; loadGenerals()">
              <el-radio-button :label="0">全部</el-radio-button>
              <el-radio-button :label="1">普通军官</el-radio-button>
              <el-radio-button :label="2">名将</el-radio-button>
            </el-radio-group>
            <el-input v-model="gWord" placeholder="军官名 / ID" clearable style="width:180px"
                      @keyup.enter.native="loadGenerals" />
            <el-button type="primary" icon="el-icon-search" @click="loadGenerals">查询</el-button>
            <span class="td-sub">
              普通军官 {{ gKindCounts.normal }} 名（军校招募从这里抽）· 名将 {{ gKindCounts.general }} 名（只能发放）
            </span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openGeneralCreate">新增军官</el-button>
            <el-button type="warning" icon="el-icon-present" @click="openGrant">分发给玩家</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadGenerals">刷新</el-button>
          </div>
          <el-table :data="gPaged" v-loading="loadingG" stripe border>
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column prop="name" label="军官" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="88" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.kind === 1 ? 'success' : 'warning'">
                  {{ row.kind === 1 ? '普通军官' : '名将' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="55" align="center" />
            <el-table-column label="星级" width="75" align="center">
              <template slot-scope="{row}">{{ '★'.repeat(row.star) }}</template>
            </el-table-column>
            <el-table-column prop="military" label="军事" width="55" align="center" />
            <el-table-column prop="logistics" label="后勤" width="55" align="center" />
            <el-table-column prop="learning" label="学识" width="55" align="center" />
            <el-table-column label="权重" width="60" align="center">
              <template slot-scope="{row}">
                <span v-if="row.kind === 1" class="td-mono">{{ row.weight }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column label="可招募" width="68" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.recruit === 1 ? 'success' : 'info'">{{ row.recruit === 1 ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="拥有玩家" width="80" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.owned_count }}</span></template>
            </el-table-column>
            <el-table-column prop="source" label="来源" min-width="150" show-overflow-tooltip />
            <el-table-column prop="des" label="组合技" min-width="120" show-overflow-tooltip />
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="warning" plain icon="el-icon-present" title="分发给玩家" @click="openGrant(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openGeneralEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delGeneral(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ gTotal }}</b> 条 · 每页 {{ gSize }} 条</div>
            <el-pagination v-show="gTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="gTotal"
                           :page-size="gSize" :current-page="gPage" :page-sizes="[10, 20, 50, 100]"
                           @current-change="p => { gPage = p; loadGenerals() }"
                           @size-change="s => { gSize = s; gPage = 1; loadGenerals() }" />
          </div>
        </el-tab-pane>

        <!-- ============ 2d. 计谋（ezfy_cfg_scheme，发动消耗信号弹） ============ -->
        <el-tab-pane label="计谋" name="schemes">
          <div class="toolbar">
            <el-input v-model="scWord" placeholder="计谋名 / ID" clearable style="width:200px"
                      @keyup.enter.native="loadSchemes" />
            <el-button type="primary" icon="el-icon-search" @click="loadSchemes">查询</el-button>
            <span class="td-sub">发动计谋消耗「信号弹」（道具 ID 24，可在「道具配置」里改价与上架）</span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openSchemeCreate">新增计谋</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadSchemes">刷新</el-button>
          </div>
          <el-table :data="schemes" v-loading="loadingSc" stripe border>
            <el-table-column prop="id" label="ID" width="55" align="center" />
            <el-table-column prop="name" label="计谋" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="280" show-overflow-tooltip />
            <el-table-column label="消耗信号弹" width="100" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.bullet }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="110" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.kind === 1 ? 'warning' : 'info'">
                  {{ row.kind === 1 ? '先发制人' : '说明型' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="可战争时长(分)" width="120" align="center">
              <template slot-scope="{row}">
                <span v-if="row.kind === 1" class="td-mono">{{ row.war_minutes }} / 上限 {{ row.war_max_minutes }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column label="上架" width="70" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '已上架' : '已下架' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="sort_no" label="排序" width="60" align="center" />
            <el-table-column label="操作" width="130" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openSchemeEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delScheme(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-info" style="margin-top:8px">共 <b>{{ schemes.length }}</b> 条计谋</div>
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
            <el-table-column label="是否名将" width="76" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.is_general ? 'warning' : 'info'">
                  {{ row.is_general ? '名将' : '普通' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="姓名" min-width="115" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="星级" width="80" align="center">
              <template slot-scope="{row}">{{ '★'.repeat(row.star) }}</template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="48" align="center" />
            <el-table-column label="军事/后勤/学识" width="120" align="center">
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
            <el-table-column label="状态" width="85" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.is_captive === 1 ? 'danger' : 'success'">
                  {{ row.is_captive === 1 ? '俘虏' : row.status_name }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="city_name" label="归属城池" width="100" show-overflow-tooltip />
            <el-table-column label="归属玩家" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span class="td-main">{{ row.owner_name || '—' }}</span>
                <span class="td-muted">（{{ row.home_num || '—' }}）</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" align="center" fixed="right">
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
            <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                           :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
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
            <el-table-column prop="name" label="技能名" min-width="150" show-overflow-tooltip>
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
            <el-table-column label="操作" width="140" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openSkillEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delSkill(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ skills.length }}</b> 条 · 每页 {{ sSize }} 条</div>
            <el-pagination v-show="skills.length > 0" small background layout="sizes, prev, pager, next, jumper" :total="skills.length"
                           :page-size="sSize" :current-page="sPage" :page-sizes="[5, 10, 20, 50, 100]"
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
            <el-table-column prop="owner_name" label="归属玩家" min-width="150" show-overflow-tooltip />
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="遗忘" @click="removeSkill(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ osTotal }}</b> 条 · 每页 {{ osSize }} 条</div>
            <el-pagination v-show="osTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="osTotal" :page-size="osSize"
                           :current-page="osPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { osPage = p; loadOwnedSkills() }"
                           @size-change="s => { osSize = s; osPage = 1; loadOwnedSkills() }" />
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
            <el-table-column prop="name" label="装备名" min-width="150" show-overflow-tooltip>
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
            <el-table-column label="操作" width="180" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑/穿戴" @click="openEquipOwnedEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delEquipOwned(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ oeTotal }}</b> 条 · 每页 {{ oeSize }} 条</div>
            <el-pagination v-show="oeTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="oeTotal" :page-size="oeSize"
                           :current-page="oePage" :page-sizes="[5, 10, 20, 50, 100]"
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
        <el-form-item label="军事/后勤/学识">
          <el-input-number v-model.number="form.military" :min="0" controls-position="right" style="width:120px" />
          <el-input-number v-model.number="form.logistics" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
          <el-input-number v-model.number="form.learning" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
        </el-form-item>
        <!-- ★ 2026-09-22：原始属性 + 可用属性点（重修书洗点回退到 base，并退回可用点） -->
        <el-form-item label="原始属性">
          <el-input-number v-model.number="form.base_military" :min="0" controls-position="right" style="width:120px" />
          <el-input-number v-model.number="form.base_logistics" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
          <el-input-number v-model.number="form.base_learning" :min="0" controls-position="right" style="width:120px;margin-left:6px" />
          <span class="td-sub" style="margin-left:8px">重修书洗点后回到这个值</span>
        </el-form-item>
        <el-form-item label="可用属性点">
          <el-input-number v-model.number="form.free_points" :min="0" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">每升 1 级得 1 点，玩家自己分配</span>
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
      <!-- [说明·不显示在界面] 随机生成名字 / 等级 / 星级；<b>属性上限取自同星级名将的最大值</b>，保证不会超过名将。 -->
      <div slot="footer">
        <el-button @click="genDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGen">生 成</el-button>
      </div>
    </el-dialog>

    <!-- ============ 发放名将 ============ -->
    <el-dialog title="发放名将" :visible.sync="grantDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="玩家" required>
          <el-input v-model="grantWord" placeholder="玩家昵称 / 游戏ID" clearable style="width:200px"
                    @keyup.enter.native="searchGrantPlayer" />
          <el-button type="primary" plain icon="el-icon-search" @click="searchGrantPlayer">搜索</el-button>
          <span v-if="grantPlayers.length" class="td-sub" style="margin-left:8px">找到 {{ grantPlayers.length }} 位</span>
        </el-form-item>
        <el-form-item v-if="grantPlayers.length" label="选择玩家" required>
          <el-select v-model="grantForm.user_id" filterable style="width:320px" placeholder="选择玩家">
            <el-option v-for="p in grantPlayers" :key="p.user_id"
                       :label="p.nickname + '（ID ' + p.user_id + (p.home_num ? ' · 家园号 ' + p.home_num : '') + '）'"
                       :value="p.user_id" />
          </el-select>
        </el-form-item>
        <el-form-item label="军官" required>
          <el-select v-model="grantForm.general_id" filterable style="width:320px">
            <el-option v-for="g in grantCandidates" :key="g.id"
                       :label="g.id + ' · ' + g.name + '（' + (g.kind === 1 ? '普通军官' : '名将') + '·' + '★'.repeat(g.star) + '）'" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 提示：同名将不能重复发放给同一玩家；发放的是「军官池」里的模板（属性取自池子） -->
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增/编辑 军官（军官池：普通军官 / 名将） ============ -->
    <el-dialog :title="gf.id ? ('编辑军官 · ' + gf.name) : '新增军官'" :visible.sync="gDlg"
               width="900px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="军官类型">
          <el-radio-group v-model.number="gf.kind">
            <el-radio :label="1">普通军官（军校招募从池子抽）</el-radio>
            <el-radio :label="2">名将（只能由管理端发放）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-row :gutter="10">
          <el-col :span="12">
            <el-form-item label="军官名称" required>
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
          <el-col :span="8"><el-form-item label="军事"><el-input-number v-model.number="gf.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="后勤"><el-input-number v-model.number="gf.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="学识"><el-input-number v-model.number="gf.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="星级">
              <el-input-number v-model.number="gf.star" :min="1" :max="10" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="抽取权重">
              <el-input-number v-model.number="gf.weight" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="可招募">
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
          <el-input v-model="gf.des" maxlength="500" placeholder="例如：全技能+2，攻击+20%" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="gf.skill" maxlength="500" placeholder="例如：任务 / 商店买 / 活动中心换取" />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 保存后立即生效（后端会重载配置缓存，不用重启） -->
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
      <!-- [说明·不显示在界面] 改名会同步更新所有已学该技能的军官 -->
      <!-- [说明·不显示在界面] 新技能会立刻出现在玩家军校的技能学习列表里 -->
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
      <!-- [说明·不显示在界面] 管理端加技能不消耗黄金、不受 3 个技能上限限制 -->
      <div slot="footer">
        <el-button @click="saDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSkillAssign">确 定</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增/编辑 计谋 ============ -->
    <el-dialog :title="scf.id ? ('编辑计谋 · ' + scf.name) : '新增计谋'" :visible.sync="scDlg"
               width="680px" :close-on-click-modal="false">
      <el-form label-width="130px" size="small">
        <el-form-item label="计谋名称" required>
          <el-input v-model="scf.name" maxlength="50" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="scf.des" type="textarea" :rows="3" maxlength="500" />
        </el-form-item>
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="消耗信号弹">
              <el-input-number v-model.number="scf.bullet" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="排序">
              <el-input-number v-model.number="scf.sort_no" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="上架">
              <el-switch v-model="scf.enabled" :active-value="1" :inactive-value="0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="计谋类型">
          <el-radio-group v-model.number="scf.kind">
            <el-radio :label="0">说明型（只消耗信号弹 + 记录战报）</el-radio>
            <el-radio :label="1">先发制人（使双方进入可战争状态）</el-radio>
          </el-radio-group>
        </el-form-item>
        <template v-if="scf.kind === 1">
          <el-row :gutter="10">
            <el-col :span="12">
              <el-form-item label="每点学识分钟数">
                <el-input-number v-model.number="scf.war_minutes" :min="1" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="时长上限(分钟)">
                <el-input-number v-model.number="scf.war_max_minutes" :min="1" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>
      </el-form>
      <!-- [说明·不显示在界面] 「信号弹」是道具配置里的 <b>ID 24</b>（ItemType 20），价格与上架在「数据管理 → 道具配置」里改。
        先发制人按「发动方军官学识 = 可战争分钟数」计算，再按上限截断。 -->
      <div slot="footer">
        <el-button @click="scDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSchemeSave">保 存</el-button>
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
      <!-- [说明·不显示在界面] 装备会进入玩家主城背包，未穿戴 -->
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

const G_KEYS = ['name', 'level', 'military', 'logistics', 'learning', 'star', 'source', 'get_condition', 'skill', 'des', 'recruit', 'kind', 'weight']
const S_KEYS = ['name', 'effect', 'type', 'des']
// ★ 计谋（消耗信号弹）
const SC_KEYS = ['name', 'des', 'bullet', 'kind', 'war_minutes', 'war_max_minutes', 'enabled', 'sort_no']
const OE_KEYS = ['name', 'type', 'tier', 'military', 'logistics', 'learning', 'level', 'officer_id']
// ★ 军官编辑里可改的字段（含原始属性 + 可用属性点）
const O_EDIT_KEYS = ['name', 'level', 'star', 'military', 'logistics', 'learning', 'loyalty',
  'position', 'status', 'base_military', 'base_logistics', 'base_learning', 'free_points']

export default {
  name: 'AdminEzfyOfficers',
  data () {
    return {
      tab: 'overview',
      saving: false,
      // 1 总览
      ov: {}, loadingOv: false,
      // 2 军官池（普通军官 + 名将）
      generals: [], gTotal: 0, loadingG: false, gWord: '', gKind: 0, gKindCounts: { normal: 0, general: 0 },
      gPage: 1, gSize: 20,
      gDlg: false, gf: {},
      // 2d 计谋
      schemes: [], loadingSc: false, scWord: '', scDlg: false, scf: {},
      // 3 玩家军官
      list: [], total: 0, page: 1, size: 5, loading: false, word: '', captive: -1,
      editDlg: false, editId: 0, form: {},
      grantDlg: false, grantForm: { user_id: 0, general_id: 0 },
      // ★ 2026-09-28 发放名将：玩家搜索（昵称/游戏ID）—— 原来只能手填用户ID，
      //   而管理员手上通常只有玩家的昵称或游戏里看到的游戏ID，根本填不出来。
      grantWord: '', grantPlayers: [],
      // 4 技能
      skills: [], loadingS: false, sWord: '', sPage: 1, sSize: 5,
      sDlg: false, sf: {},
      // 5 玩家军官技能
      ownedSkills: [], osTotal: 0, osPage: 1, osSize: 5, loadingOS: false,
      osWord: '', osSkill: '',
      saDlg: false, saForm: { officer_id: 0, skill_name: '' },
      // 6 装备
      equips: [], loadingE: false,
      // 7 玩家装备
      ownedEquips: [], oeTotal: 0, oePage: 1, oeSize: 5, loadingOE: false,
      oeWord: '', oeType: '', oeEquipped: -1,
      egDlg: false, egForm: { user_id: 1, cfg_id: 0, count: 1 },
      oeDlg: false, oef: {}, ownerOfficers: [],
      // 下拉数据
      pickers: { generals: [], skills: [], officers: [] },
      genDlg: false, genForm: { user_id: 10007, count: 3, max_level: 60 }
    }
  },
  computed: {
    // ★ 军官池改由后端分页（普通军官有 1000 条，前端一次拿全太卡）
    gPaged () { return this.generals },
    // 配置表类列表（后端一次给全）在前端分页
    sPaged () { const st = (this.sPage - 1) * this.sSize; return this.skills.slice(st, st + this.sSize) },
    // ★ 发放军官时的候选：优先用下拉数据（含 kind），退回到当前列表
    grantCandidates () {
      if (this.pickers.generals && this.pickers.generals.length) return this.pickers.generals
      return this.generals || []
    },
    ovCards () {
      const o = this.ov || {}
      return [
        { label: '军官池', val: o.generals, hint: 'ezfy_cfg_general', tab: 'generals' },
        { label: '技能配置', val: o.skills, hint: 'ezfy_cfg_skill', tab: 'skills' },
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
        { key: 'generals', tab: '军官池', table: 'ezfy_cfg_general', kind: '配置表', count: o.generals, des: '普通军官（军校招募从池子抽）+ 名将（只能发放），可增删改' },
        { key: 'skills', tab: '军官技能列表', table: 'ezfy_cfg_skill', kind: '配置表', count: o.skills, des: '技能（每名军官最多学 3 个），可增删改' },
        { key: 'officers', tab: '玩家军官列表', table: 'ezfy_officer', kind: '实例表', count: o.officers, des: '玩家拥有的军官，含属性/原始属性/可用点数/忠诚/任命/俘虏状态' },
        { key: 'ownedSkills', tab: '玩家军官技能列表', table: 'ezfy_officer.skill', kind: '实例表', count: o.owned_skills, des: '把每名军官的 skill JSON 摊平成一行一条技能' },
        { key: 'ownedEquips', tab: '玩家军官装备列表', table: 'ezfy_equipment', kind: '实例表', count: o.owned_equipments, des: '玩家背包里的装备，可穿戴到军官' }
      ]
    }
  },
  watch: {
    gWord () { this.gPage = 1 },
    sWord () { this.sPage = 1 }
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
    goTab (k) {
      this.tab = k
      this.onTab()
    },
    onTab () {
      if (this.tab === 'overview') this.loadOverview()
      if (this.tab === 'generals') this.loadGenerals()
      if (this.tab === 'schemes') this.loadSchemes()
      if (this.tab === 'officers') this.load()
      if (this.tab === 'skills') this.loadSkills()
      if (this.tab === 'ownedSkills') this.loadOwnedSkills()
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
    // ---------- 2 军官池（普通军官 + 名将） ----------
    loadGenerals () {
      this.loadingG = true
      const params = { word: this.gWord, page: this.gPage, size: this.gSize }
      if (this.gKind) params.kind = this.gKind
      api.get('/admin/ezfy-generals', { params }).then(r => {
        this.loadingG = false
        if (r.code === 0) {
          this.generals = r.data.list
          this.gTotal = r.data.total
          if (r.data.kind_counts) this.gKindCounts = r.data.kind_counts
        } else this.$message.error(r.msg)
      })
    },
    openGeneralCreate () {
      this.gf = {
        name: '', level: 130, military: 100, logistics: 100, learning: 100,
        star: 5, source: '', get_condition: '', skill: '', des: '', recruit: 1,
        // ★ 默认按普通军官建（军校招募池），要建名将就切一下
        kind: this.gKind === 2 ? 2 : 1, weight: 100
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
      if (!String(this.gf.name || '').trim()) { this.$message.warning('军官名称不能为空'); return }
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
      const extra = row.kind === 1
        ? '该军官是「普通军官」，删掉后军校就不会再刷到它。'
        : '删除名将会同时回收玩家已拥有的该军官。'
      this.$confirm('删除「' + row.name + '」？' + extra + '（已拥有 ' + row.owned_count + ' 名）',
        '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-generals/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadGenerals(); this.loadOverview() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 2d 计谋（消耗信号弹） ----------
    loadSchemes () {
      this.loadingSc = true
      api.get('/admin/ezfy-schemes', { params: { word: this.scWord } }).then(r => {
        this.loadingSc = false
        if (r.code === 0) this.schemes = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openSchemeCreate () {
      this.scf = {
        name: '', des: '', bullet: 4, kind: 0,
        war_minutes: 60, war_max_minutes: 360, enabled: 1, sort_no: 0
      }
      this.scDlg = true
    },
    openSchemeEdit (row) {
      const f = { id: row.id }
      SC_KEYS.forEach(k => { f[k] = row[k] })
      this.scf = f
      this.scDlg = true
    },
    doSchemeSave () {
      if (!String(this.scf.name || '').trim()) { this.$message.warning('计谋名称不能为空'); return }
      const body = {}
      SC_KEYS.forEach(k => { body[k] = this.scf[k] })
      const isNew = !this.scf.id
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-schemes', body) : api.put('/admin/ezfy-schemes/' + this.scf.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.scDlg = false; this.$message.success(r.msg || '已保存'); this.loadSchemes() }
        else this.$message.error(r.msg)
      })
    },
    delScheme (row) {
      this.$confirm('删除计谋「' + row.name + '」？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-schemes/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.msg || '已删除'); this.loadSchemes() } else this.$message.error(r.msg)
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
      const f = {}
      O_EDIT_KEYS.forEach(k => { f[k] = row[k] })
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
      this.loadPickers()
      const list = this.grantCandidates || []
      const gid = row && row.id ? row.id : (list.length ? list[0].id : 0)
      this.grantForm = { user_id: 0, general_id: gid }
      // ★ 每次打开都清掉上一次的搜索结果，避免「看着选了玩家、其实还是上一位」
      this.grantWord = ''
      this.grantPlayers = []
      this.grantDlg = true
    },
    // ★ 2026-09-28 按昵称 / 游戏ID 搜索目标玩家
    //   （与「发放道具」用同一个后端 handler，返回最多 20 条：user_id + nickname + home_num）
    searchGrantPlayer () {
      const word = (this.grantWord || '').trim()
      if (!word) { this.$message.warning('请输入玩家昵称或游戏ID'); return }
      api.get('/admin/ezfy-officers/players', { params: { word } }).then(r => {
        if (r.code !== 0) { this.$message.error(r.msg); this.grantPlayers = []; return }
        this.grantPlayers = r.data.list || []
        // 只有一条结果时自动选中，省一次点击
        this.grantForm.user_id = this.grantPlayers.length === 1 ? this.grantPlayers[0].user_id : 0
        if (!this.grantPlayers.length) this.$message.warning('未找到匹配的玩家')
      })
    },
    doGrant () {
      if (!this.grantForm.user_id) { this.$message.warning('请先搜索并选择玩家'); return }
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
    // ---------- 6 装备配置（装备池） ----------
    // ★ 2026-09-27 配置/属性已迁到「数据管理 → 装备道具配置」和「军官装备管理」，
    //   这里只保留 loadEquips 给「给玩家发装备」下拉用（全量拉取，不做散件/套装筛选）
    loadEquips () {
      this.loadingE = true
      api.get('/admin/ezfy-equipments').then(r => {
        this.loadingE = false
        if (r.code === 0) this.equips = r.data.list
        else this.$message.error(r.msg)
      })
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
/* ★ 装备六项战斗属性的紧凑一行（label + 小输入框） */
.attr-row { display: flex; flex-wrap: wrap; gap: 6px 14px; align-items: center; font-size: 12px; color: #5b6b82; }
</style>
