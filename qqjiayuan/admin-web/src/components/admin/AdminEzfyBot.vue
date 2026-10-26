<template>
  <div class="farm-admin">
    <!-- ★ 2026-10-18 顶栏布局优化：统计/开关+操作按钮一行、数值配置一行，分组清晰 -->
    <el-card shadow="never" class="box" style="margin-bottom:12px">
      <div class="bot-bar">
        <div class="bot-bar-row">
          <span class="td-main">人机 <b>{{ list.length }}</b>/{{ max }}
            <el-tooltip placement="top" :content="tips.overview"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <span class="bot-sep" />
          <span class="bot-item">攻击玩家
            <el-tooltip placement="top" :content="tips.atk"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <el-switch v-model="attackOn" :active-value="1" :inactive-value="0" @change="saveConfig" />
          <span class="bot-sep" />
          <span class="bot-item">人机免耗粮
            <el-tooltip placement="top" :content="tips.foodFree"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <el-switch v-model="foodFree" :active-value="1" :inactive-value="0" @change="saveConfig" />
          <div class="grow" />
          <el-button type="primary" icon="el-icon-video-play" @click="openActAll">一键立即行动</el-button>
          <el-button type="primary" icon="el-icon-plus" @click="createDlg = true">生成人机</el-button>
          <el-button type="danger" plain icon="el-icon-delete" :disabled="!list.length" @click="clearAll">清空全部</el-button>
        </div>
        <div class="bot-bar-row">
          <span class="bot-item">每人每日出征</span>
          <el-input-number v-model.number="atkDailyCap" :min="0" :max="100" size="mini"
                           controls-position="right" style="width:90px" @change="saveConfig" />
          <span class="bot-sep" />
          <span class="bot-item">最低声望
            <el-tooltip placement="top" :content="tips.minPrestige"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <el-input-number v-model.number="minPrestige" :min="0" :step="100000"
                           controls-position="right" style="width:130px" @change="saveConfig" />
          <span class="bot-sep" />
          <span class="bot-item">征兵线
            <el-tooltip placement="top" :content="tips.train"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <el-input-number v-model.number="trainThreshold" :min="10000" :step="100000"
                           controls-position="right" style="width:130px" @change="saveConfig" />
          <span class="bot-sep" />
          <!-- ★ 2026-10-26 目标最小兵力：只打城内兵力 ≥ 该值的玩家城（默认 10000，0 = 不限） -->
          <span class="bot-item">目标最小兵力
            <el-tooltip placement="top" :content="tips.targetMinTroops"><i class="el-icon-info cfg-tip" /></el-tooltip>
          </span>
          <el-input-number v-model.number="targetMinTroops" :min="0" :step="10000"
                           controls-position="right" style="width:130px" @change="saveConfig" />
          <span class="bot-sep" />
          <span class="bot-item">出征时间段</span>
          <el-input v-model="atkWindows" placeholder="留空=全天 如 20:00-22:00,08:00-12:00"
                    style="width:230px" maxlength="120" @change="saveConfig" />
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="昵称 / 用户ID 搜索" clearable style="width:220px"
                  @keyup.enter.native="load" @clear="load" />
        <el-button type="primary" icon="el-icon-search" @click="load">查询</el-button>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="95" align="center" />
        <el-table-column label="昵称" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname }}</span></template>
        </el-table-column>
        <el-table-column label="难度" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="levelTag(row.level)">{{ row.level_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="阵营" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.camp === 2 ? 'danger' : 'primary'">{{ row.camp_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主城" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="row.city_id">{{ row.city_name }}（{{ row.x }},{{ row.y }}）</span>
            <span v-else class="td-gray">—</span>
          </template>
        </el-table-column>
        <el-table-column label="兵力" width="110" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.troop_total) }}</span></template>
        </el-table-column>
        <el-table-column label="集结令" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.gather_count) }}</span></template>
        </el-table-column>
        <!-- ★ 2026-10-17 攻频/出兵：按难度生效值（0=默认 新手6·30万 / 普通8·35万 / 精英10·40万） -->
        <el-table-column label="攻频/出兵" width="150" align="center">
          <template slot-scope="{row}">
            <span class="td-mono">{{ ezfyAtkTypeName(row.atk_type_eff) }} {{ row.atk_per_hour_eff }}次/h · {{ fmtNum(row.atk_troop_eff) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="声望" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.prestige) }}</span></template>
        </el-table-column>
        <!-- ★ 2026-10-18 最后行动：不再展示具体日期时间（占地方），只留上次动作文字 -->
        <el-table-column label="最后行动" min-width="90" align="center">
          <template slot-scope="{row}">
            <span v-if="row.last_action" class="td-gray">{{ row.last_action }}</span>
            <span v-else-if="row.last_act_at" class="td-gray">已行动</span>
            <span v-else class="td-gray">未行动</span>
          </template>
        </el-table-column>
        <el-table-column label="今日出征" width="90" align="center">
          <template slot-scope="{row}">{{ row.atk_count_today || 0 }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '运行中' : '已暂停' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="460" align="center" fixed="right">
          <template slot-scope="{row}">
            <!-- ★ 2026-10-19 发放资源/占领城市 已收进「编辑人机」对话框 tab（外置按钮过多） -->
            <el-button size="mini" plain icon="el-icon-edit" title="编辑（昵称/声望/城市名/人口/出征/发放/占领等）" @click="openEdit(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-video-play" title="立即行动一轮" @click="actNow(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-aim" title="攻击目标（查看/指定/删除）" @click="openTargets(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-map-location" title="迁城（输入坐标）" @click="openRelocate(row)" />
            <el-button size="mini" plain icon="el-icon-tickets" title="行为日志" @click="openLogs(row)" />
            <el-button size="mini" plain icon="el-icon-document" title="出征战报" @click="openReports(row)" />
            <el-button size="mini" :type="row.enabled ? 'warning' : 'success'" plain
                       :icon="row.enabled ? 'el-icon-video-pause' : 'el-icon-video-play'"
                       :title="row.enabled ? '暂停' : '启用'" @click="toggle(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 生成人机 -->
    <el-dialog title="生成人机" :visible.sync="createDlg" width="440px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="数量">
          <el-input-number v-model.number="createCount" :min="1" :max="10" controls-position="right"
                           style="width:120px" />
        </el-form-item>
        <el-form-item label="难度档">
          <el-radio-group v-model="createLevel">
            <el-radio :label="1">新手陪练</el-radio>
            <el-radio :label="2">普通</el-radio>
            <el-radio :label="3">精英</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="加速开局">
          <el-switch v-model="createBoost" />
          <el-tooltip placement="top" :content="tips.boost"><i class="el-icon-info cfg-tip" style="margin-left:6px" /></el-tooltip>
        </el-form-item>
        <!-- ★ 2026-10-17 昵称马甲：生成时自定义名字（仅单条生成可用；留空走自动词库） -->
        <el-form-item label="昵称马甲">
          <el-input v-model="createNickname" placeholder="留空 = 随机军事昵称（仅 1 个时可用）" style="width:260px" maxlength="20" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="createDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">生 成</el-button>
      </div>
    </el-dialog>

    <!-- 一键立即行动（全部启用人机各连跑 N 轮） -->
    <el-dialog title="一键立即行动" :visible.sync="actAllDlg" width="440px" :close-on-click-modal="false">
      <el-form label-width="120px">
        <el-form-item label="行动轮数">
          <el-input-number v-model.number="actAllRounds" :min="1" :max="50" controls-position="right"
                           style="width:140px" />
          <el-tooltip placement="top" content="后台 8 并发执行，对全部运行中的人机各连跑 N 轮；执行中再点会被拒绝"><i class="el-icon-info cfg-tip" style="margin-left:6px" /></el-tooltip>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="actAllDlg = false">取 消</el-button>
        <el-button type="primary" :loading="actAllSaving" @click="doActAll">立即执行</el-button>
      </div>
    </el-dialog>

    <!-- 编辑人机（昵称/声望/城市名/人口/建筑/科技/兵种/军官） -->
    <el-dialog :title="'编辑人机 —— ' + (cur ? cur.nickname : '')" :visible.sync="editDlg" width="1000px"
               :close-on-click-modal="false">
      <div v-loading="editLoading" style="min-height:300px">
        <el-tabs v-model="editTab" type="card">
          <el-tab-pane label="基本信息" name="base">
            <el-form label-width="100px" label-position="left">
              <el-form-item label="游戏昵称">
                <el-input v-model="editForm.nickname" maxlength="20" show-word-limit style="width:220px" />
                <el-tooltip placement="top" content="同步改家园昵称 + 二战昵称（地图/聊天/战报一致）"><i class="el-icon-info cfg-tip" style="margin-left:6px" /></el-tooltip>
              </el-form-item>
              <el-form-item label="军功声望">
                <el-input-number v-model.number="editForm.prestige" :min="0" :step="1000"
                                 controls-position="right" style="width:160px" />
              </el-form-item>
              <el-form-item label="城市名">
                <el-input v-model="editForm.city_name" maxlength="20" show-word-limit style="width:220px" />
                <span class="td-gray" style="margin-left:6px">当前主城</span>
              </el-form-item>
              <!-- ★ 2026-10-26 编辑页精简：人口三项（空闲/上限/待恢复）合并成一行，原来占 3 行 -->
              <el-form-item label="人口">
                <span class="td-gray">空闲</span>
                <el-input-number v-model.number="editForm.freePop" :min="0" :step="1000" size="small"
                                 controls-position="right" style="width:120px;margin-left:4px" />
                <span class="td-gray" style="margin-left:10px">上限</span>
                <el-input-number v-model.number="editForm.pop_max" :min="0" :step="1000" size="small"
                                 controls-position="right" style="width:120px;margin-left:4px" />
                <span class="td-gray" style="margin-left:10px">待恢复</span>
                <el-input-number v-model.number="editForm.pop_recover" :min="0" :step="1000" size="small"
                                 controls-position="right" style="width:120px;margin-left:4px" />
                <el-tooltip placement="top" content="空闲人口 = 总人口 − 建筑占用 − 训练占用 − 待恢复（保存时自动反推总人口）；上限配置后当前人口同步到上限，别人侦察即显示该值">
                  <i class="el-icon-info cfg-tip" style="margin-left:6px" />
                </el-tooltip>
                <span class="td-gray" style="margin-left:6px">总人口 {{ fmtNum(editForm.pop) }}</span>
              </el-form-item>
              <!-- ★ 2026-10-17 出征设置：攻击频率 + 单次出兵，0=按难度默认（新手6次/8h→普通8/精英10 + 30/35/40万） -->
              <el-form-item label="攻频/出兵">
                <el-input-number v-model.number="editForm.atk_per_hour" :min="0" :max="60" :step="1"
                                 controls-position="right" style="width:100px" />
                <span class="td-gray" style="margin-left:6px">次/小时 (0=默认 {{ editForm.atk_per_hour_eff }})</span>
                <el-input-number v-model.number="editForm.atk_troop" :min="0" :step="10000"
                                 controls-position="right" style="width:140px;margin-left:16px" />
                <span class="td-gray" style="margin-left:6px">出兵 (0=默认 {{ fmtNum(editForm.atk_troop_eff) }})</span>
                <el-select v-model.number="editForm.atk_type" style="width:110px;margin-left:16px" size="mini">
                  <el-option label="征服(默认)" :value="0" />
                  <el-option label="侦察" :value="1" />
                  <el-option label="掠夺" :value="2" />
                  <el-option label="征服" :value="3" />
                </el-select>
                <span class="td-gray" style="margin-left:6px">出征类型</span>
              </el-form-item>
              <!-- ★ 2026-10-19 自动发言开关（人机是否像真人一样在世界/城市频道自动发言） -->
              <el-form-item label="自动发言">
                <el-switch v-model="editForm.chat_on" :active-value="1" :inactive-value="0" />
                <span class="td-gray" style="margin-left:6px">关闭后人机不再自动发言（玩家端无从察觉它是机器人）</span>
              </el-form-item>
              <!-- ★ 2026-10-25 打野地开关（是否攻击真人玩家的附属野地；默认关闭） -->
              <el-form-item label="打野地">
                <el-switch v-model="editForm.attack_wild" :active-value="1" :inactive-value="0" />
                <el-tooltip placement="top" :content="tips.attackWild"><i class="el-icon-info cfg-tip" style="margin-left:6px" /></el-tooltip>
                <span class="td-gray" style="margin-left:6px">{{ editForm.attack_wild ? '会征服真人玩家的附属野地' : '不打玩家的附属野地（只打城市）' }}</span>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="建筑等级" name="build">
            <el-table :data="editBuildings" size="mini" border max-height="300" style="width:100%">
              <el-table-column prop="name" label="建筑" min-width="120" />
              <el-table-column label="等级" width="180" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.level" :min="0" :max="row.max_level" size="mini"
                                   controls-position="right" style="width:100px" />
                  <span v-if="row.status" class="td-gray" style="font-size:12px">施工中</span>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="科技等级" name="tech">
            <el-table :data="editTechs" size="mini" border max-height="300" style="width:100%">
              <el-table-column prop="name" label="科技" min-width="120" />
              <el-table-column label="等级" width="180" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.level" :min="0" :max="row.max_level" size="mini"
                                   controls-position="right" style="width:100px" />
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="兵种数量" name="troop">
            <!-- ★ 2026-10-17 兵种名收窄、数量列占大：人机兵力可达亿级，需要足够输入宽度 -->
            <el-table :data="editTroops" size="mini" border max-height="300" style="width:100%">
              <el-table-column prop="name" label="兵种" min-width="90" show-overflow-tooltip />
              <el-table-column label="数量" min-width="300" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.count" :min="0" :step="1000"
                                   controls-position="right" style="width:260px" />
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="军官" name="officer">
            <div class="toolbar" style="margin-bottom:8px">
              <span class="td-main">发放军官</span>
              <el-select v-model="grantGeneralId" filterable placeholder="选择军官池/名将" size="mini" style="width:200px">
                <el-option v-for="g in generalOptions" :key="g.id" :label="(g.kind === 2 ? '★' : '') + g.name + '（' + g.star + '星）'" :value="g.id" />
              </el-select>
              <el-button size="mini" type="success" plain icon="el-icon-plus" :loading="grantSaving"
                         @click="grantGeneral">发放到主城</el-button>
              <el-tooltip placement="top" content="加点：直接填三维最终值，差额自动从「待分配点」扣减，待分配点不够会保存失败；待分配点也可直接调（管理端给点）"><i class="el-icon-info cfg-tip" /></el-tooltip>
              <div class="grow" />
            </div>
            <el-table :data="editOfficers" size="mini" border max-height="320" style="width:100%">
              <el-table-column prop="name" label="军官" min-width="90" show-overflow-tooltip />
              <el-table-column prop="star" label="星" width="45" align="center" />
              <!-- ★ 2026-10-18 编辑人机军官页精简：等级/经验 合并、三维属性 合并为一列，技能列加「随机」 -->
              <el-table-column label="等级/经验" width="120" align="center">
                <template slot-scope="{row}">
                  <div style="display:flex; flex-direction:column; gap:4px">
                    <el-input-number v-model="row.level" :min="1" :max="350" size="mini"
                                     controls-position="right" style="width:100px" placeholder="等级" />
                    <el-input-number v-model="row.exp" :min="0" :step="1000" size="mini"
                                     controls-position="right" style="width:100px" placeholder="经验" />
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="待分配点" width="100" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.free_points" :min="0" :step="10" size="mini"
                                   controls-position="right" style="width:90px" />
                </template>
              </el-table-column>
              <el-table-column label="三维属性" width="250" align="center">
                <template slot="header">
                  <span>三维属性</span>
                  <el-tooltip placement="top" content="顺序：军事 / 后勤 / 学识，填最终值，差额自动从待分配点扣减">
                    <i class="el-icon-info cfg-tip" style="margin-left:2px" />
                  </el-tooltip>
                </template>
                <template slot-scope="{row}">
                  <div style="display:flex; align-items:center; justify-content:center; gap:6px">
                    <el-input-number v-model="row.military" :min="0" :step="10" size="mini"
                                     controls-position="right" style="width:74px" placeholder="军事" />
                    <el-input-number v-model="row.logistics" :min="0" :step="10" size="mini"
                                     controls-position="right" style="width:74px" placeholder="后勤" />
                    <el-input-number v-model="row.learning" :min="0" :step="10" size="mini"
                                     controls-position="right" style="width:74px" placeholder="学识" />
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="技能（最多3个）" min-width="240">
                <template slot-scope="{row}">
                  <div style="display:flex; align-items:center; gap:6px">
                    <el-select v-model="row.skills" multiple filterable collapse-tags size="mini" style="flex:1"
                               placeholder="选技能">
                      <el-option v-for="s in skillOptions" :key="s" :label="s" :value="s" />
                    </el-select>
                    <!-- ★ 2026-10-18 随机 3 个不重复技能（对齐人机自动配技能，不消耗技能书） -->
                    <el-button size="mini" type="success" plain icon="el-icon-refresh"
                               title="随机配置 3 个不重复技能" @click="randomSkills(row)">随机</el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <!-- ★ 2026-10-19 发放资源 + 占领城市 收进编辑人机 tab（外置按钮过多，用户要求挪进来） -->
          <el-tab-pane label="发放/占领" name="grant">
            <div class="td-main" style="margin-bottom:4px">发放资源</div>
            <el-form label-width="56px" label-position="left">
              <div style="display:flex; flex-wrap:wrap; gap:4px 28px">
                <el-form-item v-for="f in resFields" :key="f.k" :label="f.t">
                  <el-input-number v-model.number="grantForm[f.k]" :step="10000" :precision="0"
                                   controls-position="right" style="width:130px" />
                  <div class="td-gray" style="font-size:12px">当前 {{ fmtNum(grantCur[f.k]) }}</div>
                </el-form-item>
                <el-form-item label="集结令">
                  <el-input-number v-model.number="grantForm.gather" :step="1000" :precision="0"
                                   controls-position="right" style="width:130px" />
                  <div class="td-gray" style="font-size:12px">当前 {{ fmtNum(grantCur.gather) }}
                    <el-tooltip effect="dark" content="人机出征时自动烧集结令扩出征上限（每个 +10 万），烧完列表实时可见" placement="top">
                      <i class="el-icon-question" style="cursor:pointer"></i>
                    </el-tooltip>
                  </div>
                </el-form-item>
              </div>
              <el-form-item>
                <el-button type="primary" size="mini" :loading="grantSaving" @click="doGrant">发 放</el-button>
                <span class="td-gray" style="font-size:12px;margin-left:8px">可填负数扣除；发完人机会立刻行动消化资源；不发钻石。</span>
              </el-form-item>
            </el-form>
            <el-divider style="margin:8px 0" />
            <div class="toolbar" style="margin-bottom:8px">
              <span class="td-main">占领城市</span>
              <span class="td-gray" style="font-size:12px;margin-left:8px">出征胜利→直接征服；可归还 / 摧毁 / 同意赎回（人机收 80% 押金，20% 系统收）</span>
              <div class="grow" />
              <el-button size="mini" type="warning" plain icon="el-icon-refresh" :loading="occRedeemAllLoading"
                         @click="redeemAllOccupy">统一赎回全部</el-button>
            </div>
            <el-table :data="occList" v-loading="occLoading" size="mini" border max-height="300"
                      :empty-text="'该人机暂无占领的城市（出征胜利后会直接征服目标城）'">
              <el-table-column prop="city_name" label="城市" min-width="110" show-overflow-tooltip />
              <el-table-column label="坐标" width="100" align="center">
                <template slot-scope="{row}">（{{ row.x }},{{ row.y }}）</template>
              </el-table-column>
              <el-table-column prop="def_name" label="原主人" min-width="110" show-overflow-tooltip>
                <template slot-scope="{row}">{{ row.def_name || ('ID ' + row.def_uid) }}</template>
              </el-table-column>
              <el-table-column label="状态" width="90" align="center">
                <template slot-scope="{row}">
                  <el-tag size="mini" :type="row.status === 1 ? 'danger' : 'info'">{{ row.status_txt }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="赎回申请" width="90" align="center">
                <template slot-scope="{row}">{{ row.ransom_count || 0 }}</template>
              </el-table-column>
              <el-table-column label="占领时间" width="140" align="center">
                <template slot-scope="{row}">{{ row.created_at }}</template>
              </el-table-column>
              <el-table-column label="操作" width="210" align="center">
                <template slot-scope="{row}">
                  <template v-if="row.status === 1">
                    <el-button size="mini" type="success" plain @click="occReturn(row)">归还</el-button>
                    <el-button size="mini" type="danger" plain @click="occDestroy(row)">摧毁</el-button>
                    <el-button size="mini" type="warning" plain :disabled="!row.ransom_count"
                               @click="occRedeem(row)">同意赎回</el-button>
                  </template>
                  <span v-else class="td-gray">—</span>
                </template>
              </el-table-column>
            </el-table>
            <div class="pager-bar">
              <div class="pager-info">共 <b>{{ occTotal }}</b> 条</div>
              <el-pagination v-show="occTotal > 0" small background layout="prev, pager, next" :total="occTotal"
                             :page-size="occSize" :current-page="occPage"
                             @current-change="p => { occPage = p; loadOccupy() }" />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="editSaving" @click="saveEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 攻击目标（人机可自己找，也可管理端指定；可挂多个） -->
    <el-dialog :title="'攻击目标 —— ' + (cur ? cur.nickname : '')" :visible.sync="tgtDlg" width="640px"
               :close-on-click-modal="false">
      <div class="toolbar" style="margin-bottom:10px">
        <el-input v-model.number="tgtInput" placeholder="目标玩家ID / 游戏号" clearable style="width:200px" />
        <el-button type="primary" size="mini" icon="el-icon-plus" :loading="tgtSaving" @click="addTarget">加入计划</el-button>
        <el-button type="danger" size="mini" plain icon="el-icon-s-custom" :loading="tgtSaving" @click="attackNearest">出征最近敌对城</el-button>
        <span class="td-gray" style="font-size:12px">目标必须是真人；加入后该人机下一轮自动宣战（24h 生效）</span>
      </div>
      <el-table :data="targets" v-loading="tgtLoading" size="mini" border max-height="300">
        <el-table-column prop="target_uid" label="目标ID" width="95" align="center" />
        <el-table-column prop="nickname" label="目标昵称" min-width="110" show-overflow-tooltip />
        <el-table-column label="状态" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.status === 2 ? 'success' : (row.status === 3 ? 'info' : 'warning')">{{ row.status_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="70" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除计划" @click="delTarget(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div slot="footer"><el-button @click="tgtDlg = false">关 闭</el-button></div>
    </el-dialog>

    <!-- 迁城（管理端直接输入坐标，省去等迁城计划） -->
    <el-dialog :title="'迁城 —— ' + (cur ? cur.nickname : '')" :visible.sync="relocateDlg" width="440px"
               :close-on-click-modal="false">
      <el-form label-width="80px" label-position="left">
        <el-form-item label="目标坐标">
          <el-input-number v-model.number="relocateForm.x" :min="1" :max="499" controls-position="right"
                           style="width:130px" />
          <span style="margin:0 4px">,</span>
          <el-input-number v-model.number="relocateForm.y" :min="1" :max="499" controls-position="right"
                           style="width:130px" />
        </el-form-item>
        <el-form-item label="当前坐标">
          <span class="td-gray">{{ cur ? cur.city_name + '(' + cur.x + ',' + cur.y + ')' : '—' }}</span>
        </el-form-item>
        <div class="td-gray" style="font-size:12px">只能迁到平原或沿海平原，且目标坐标不能已有城市；迁完人机下一轮立即按新城行动</div>
      </el-form>
      <div slot="footer">
        <el-button @click="relocateDlg = false">取 消</el-button>
        <el-button type="primary" :loading="relocateSaving" @click="doRelocate">迁 移</el-button>
      </div>
    </el-dialog>

    <!-- 行为日志 -->
    <el-dialog :title="'行为日志 —— ' + (cur ? cur.nickname : '全部')" :visible.sync="logsDlg" width="820px">
      <el-table :data="logs" v-loading="logsLoading" size="mini" border max-height="420">
        <el-table-column prop="created_at" label="时间" width="170" align="center" />
        <el-table-column prop="nickname" label="人机" width="120" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" width="110" align="center" />
        <el-table-column label="结果" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.result ? 'success' : 'info'">{{ row.result ? '成功' : '失败/跳过' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" min-width="280" show-overflow-tooltip />
        <!-- ★ 2026-10-18 物理删除：单条删除 + 顶部清空该人机全部（用户需求） -->
        <el-table-column label="操作" width="80" align="center">
          <template slot-scope="s">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除该条日志"
                       @click="delLog(s.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ logTotal }}</b> 条
          <el-button v-if="cur" size="mini" type="danger" plain icon="el-icon-delete" style="margin-left:8px"
                     @click="clearLogs(cur)">清空该人机全部日志</el-button>
        </div>
        <el-pagination v-show="logTotal > 0" small background layout="prev, pager, next" :total="logTotal"
                       :page-size="logSize" :current-page="logPage"
                       @current-change="p => { logPage = p; loadLogs() }" />
      </div>
      <div slot="footer"><el-button @click="logsDlg = false">关 闭</el-button></div>
    </el-dialog>

    <!-- 出征战报（★ 2026-10-17 管理员看人机战报：复用全局战报查询，按 user_id 过滤） -->
    <el-dialog :title="'出征战报 —— ' + (cur ? cur.nickname : '')" :visible.sync="reportsDlg" width="860px">
      <el-table :data="reports" v-loading="reportsLoading" size="mini" border max-height="430">
        <!-- ★ 2026-10-18 时间列格式化：原来直接吐后端 ISO 时间串（2026-10-18T14:30:25+08:00），看不清 → 统一 fmtTime -->
        <el-table-column label="时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="type_name" label="类型" width="110" align="center" />
        <!-- ★ 2026-10-18 出征战报弹窗：攻击方（= 该人机）/ 被攻击方（守方玩家名） -->
        <el-table-column prop="player_name" label="攻击方" width="120" show-overflow-tooltip />
        <el-table-column label="被攻击方" width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.target_name || '—' }}</template>
        </el-table-column>
        <el-table-column prop="preview" label="内容" min-width="300" show-overflow-tooltip />
        <el-table-column label="操作" width="80" align="center">
          <template slot-scope="s">
            <el-button size="mini" plain icon="el-icon-view" @click="openReportDetail(s.row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ reportTotal }}</b> 条战报</div>
        <el-pagination v-show="reportTotal > 0" small background layout="prev, pager, next" :total="reportTotal"
                       :page-size="reportSize" :current-page="reportPage"
                       @current-change="p => { reportPage = p; loadReports() }" />
      </div>
      <div slot="footer"><el-button @click="reportsDlg = false">关 闭</el-button></div>
    </el-dialog>

    <!-- 战报详情（★ 2026-10-17 点「详情」拉全文：content + detail） -->
    <el-dialog :title="reportDetail.title || '战报详情'" :visible.sync="reportDetailDlg" width="720px">
      <div class="report-detail" v-if="reportDetail">
        <pre v-if="reportDetail.content" class="report-pre">{{ reportDetail.content }}</pre>
        <pre v-if="reportDetail.detail" class="report-pre report-pre-detail">{{ reportDetail.detail }}</pre>
        <div v-if="!reportDetail.content && !reportDetail.detail" class="td-gray">（战报内容为空）</div>
      </div>
      <div slot="footer"><el-button @click="reportDetailDlg = false">关 闭</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-17 人机（AI 玩家）管理 —— 与「玩家信息管理」完全独立的模块。
// 列表/生成/清空/启停/难度/编辑/发资源/行为日志/风控开关。
// 后端：/api/admin/ezfy-bot/*（权限 module:ezfyBot）
// 说明文案一律走 tooltip（用户要求：别平铺到页面）。
export default {
  name: 'AdminEzfyBot',
  data () {
    return {
      list: [], max: 50, loading: false, word: '',
      attackOn: 1, atkDailyCap: 3, foodFree: 1, minPrestige: 500000, trainThreshold: 500000,
      targetMinTroops: 10000,
      atkWindows: '', saving: false,
      // 攻击目标
      tgtDlg: false, tgtLoading: false, tgtSaving: false, tgtInput: '', targets: [],
      // 迁城（管理端直接输入坐标）
      relocateDlg: false, relocateSaving: false, relocateForm: { x: 1, y: 1 },
      // 编辑快照（保存时只提交有变化的 tab —— 修复「保存卡顿」：几十条 UPDATE 串行跨 WAN 太慢）
      editSnap: { b: '', t: '', tr: '', o: '' },
      createDlg: false, createCount: 3, createLevel: 2, createBoost: true, createNickname: '',
      // 一键立即行动（全部人机连跑 N 轮）
      actAllDlg: false, actAllSaving: false, actAllRounds: 5,
      // 编辑（昵称/声望/城市名/人口/建筑/科技/兵种/军官）
      editDlg: false, editLoading: false, editSaving: false, editTab: 'base',
      editForm: { nickname: '', prestige: 0, city_name: '', pop: 0, pop_max: 0, pop_recover: 0, freePop: 0,
        atk_per_hour: 0, atk_troop: 0, atk_per_hour_eff: 0, atk_troop_eff: 0,
        atk_type: 0, atk_type_eff: 3, chat_on: 1, attack_wild: 0 },
      editSnapA: { h: 0, t: 0, ty: 0, aw: 0 },
      editBuildings: [],
      editTechs: [],
      editTroops: [],
      editOfficers: [],
      // 发放军官 + 技能下拉
      grantGeneralId: 0, generalOptions: [], skillOptions: [],
      // 发放资源（已收进编辑对话框「发放/占领」tab）
      grantSaving: false,
      grantForm: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, gather: 0 },
      grantCur: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, gather: 0 },
      resFields: [
        { k: 'gold', t: '黄金' }, { k: 'food', t: '粮食' }, { k: 'steel', t: '钢铁' },
        { k: 'oil', t: '石油' }, { k: 'rare', t: '稀矿' }
      ],
      // 行为日志
      logsDlg: false, logs: [], logTotal: 0, logPage: 1, logSize: 20, logsLoading: false,
      // ★ 2026-10-17 出征战报
      reportsDlg: false, reports: [], reportTotal: 0, reportPage: 1, reportSize: 20, reportsLoading: false,
      // ★ 2026-10-18 人机占领的城市（收进编辑对话框「发放/占领」tab）
      occList: [], occTotal: 0, occPage: 1, occSize: 10, occLoading: false,
      // ★ 2026-10-18 占领城市操作：统一赎回 loading
      occRedeemAllLoading: false,
      // 战报详情（★ 2026-10-17 点「详情」拉全文单独弹窗）
      reportDetailDlg: false, reportDetail: { title: '', content: '', detail: '' },
      cur: null,
      tips: {
        overview: '人机走和真人完全相同的接口（建城/升级/造兵/宣战），每 60 秒一轮、每轮最多 10 个，带租约锁（多进程只有一台真正执行）；人机不出现在「玩家信息管理」里。',
        atk: '关闭 = 人机只发育、不打扰玩家；开启后受「每人每日出征上限」与新手保护约束（建城不满 3 天、兵力过低的目标不会被打）。',
        foodFree: '开启后人机城内军队不扣维护粮、出征也免油耗（真人照常扣），让人机专注发育与出击。',
        minPrestige: '人机自动挑攻击目标时，只打声望 ≥ 此值的真人玩家（0 = 不限制）；声望越高越优先被选中。',
        train: '人机城内部队总数低于该值就执行征兵行动（默认 50 万）；填低于 1 万会自动回落到 50 万。',
        targetMinTroops: '人机自动挑攻击目标时，只打城内兵力 ≥ 此值的真人玩家城（默认 10000，0 = 不限）。避免人机去啃空城 / 几千兵的小号；管理端手动指定目标不受此限。',
        window: '人机只在指定时间段内出征（如 20:00-22:00,08:00-12:00，支持跨天 22:00-02:00）；留空 = 全天。发育/集结等其它行动不受限。',
        attackWild: '默认关闭：人机只打城市，不碰真人玩家的附属野地。开启后 —— 宣战等待期先征服目标玩家的野地，周围没有可打的城市时也会兜底打玩家野地（一律征服，胜利后立刻放弃，不占领）。',
        boost: '给一笔启动资源（50 万粮 / 20 万钢油稀）。不开 = 严格从零（1 级市政厅 + 5000 资源），攒够召集人口的 20 万粮要很久；开 = 运营初始化，当天就能造兵。'
      }
    }
  },
  watch: {
    // ★ 2026-10-19 发放/占领 tab：切进来才拉当前资源 + 占领城市（避免每次打开编辑都多查两个接口）
    editTab (v) {
      if (v === 'grant' && this.cur) this.loadGrantTab()
    }
  },
  mounted () {
    this.load()
  },
  methods: {
    fmtNum (n) { return n == null ? 0 : Number(n).toLocaleString() },
    // ★ 2026-10-17 出征类型名（1侦察 2掠夺 3征服）
    ezfyAtkTypeName (t) { return ({ 1: '侦察', 2: '掠夺', 3: '征服' })[t] || '征服' },
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    levelTag (l) { return ({ 1: 'info', 2: '', 3: 'danger' })[l] || '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-bot/list', { params: { word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.max = r.data.max || 50
          this.attackOn = r.data.attack_on ? 1 : 0
          this.atkDailyCap = r.data.atk_daily_cap || 0
          this.foodFree = r.data.food_free ? 1 : 0
          if (r.data.min_prestige !== undefined && r.data.min_prestige !== null) {
            this.minPrestige = r.data.min_prestige
          }
          if (r.data.train_threshold !== undefined && r.data.train_threshold !== null) {
            this.trainThreshold = r.data.train_threshold
          }
          // ★ 2026-10-26 目标最小兵力：**0 合法 = 不限**，不能用 || 兜底（会把 0 改回默认）
          if (r.data.target_min_troops !== undefined && r.data.target_min_troops !== null) {
            this.targetMinTroops = Number(r.data.target_min_troops)
          }
          this.atkWindows = r.data.atk_windows || ''
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    saveConfig () {
      api.post('/admin/ezfy-bot/config', {
        attack_on: this.attackOn, atk_daily_cap: this.atkDailyCap, food_free: this.foodFree,
        min_prestige: this.minPrestige, train_threshold: this.trainThreshold, atk_windows: this.atkWindows,
        target_min_troops: this.targetMinTroops
      }).then(r => {
        if (r.code === 0) this.$message.success('已保存')
        else this.$message.error(r.msg)
      })
    },
    doCreate () {
      this.saving = true
      api.post('/admin/ezfy-bot/create', { count: this.createCount, level: this.createLevel, boost: this.createBoost, nickname: this.createNickname || '' }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.createDlg = false
          this.$message.success(r.data.msg || '已生成')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    // 一键立即行动：全部启用人机各连跑 N 轮
    openActAll () { this.actAllRounds = 5; this.actAllDlg = true },
    doActAll () {
      this.actAllSaving = true
      api.post('/admin/ezfy-bot/act-all', { rounds: this.actAllRounds }).then(r => {
        this.actAllSaving = false
        if (r.code === 0) {
          this.actAllDlg = false
          this.$message.success(r.data.msg || '已执行')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.actAllSaving = false })
    },
    clearAll () {
      this.$confirm('确认清空全部人机？\n将物理删除每个人机的家园账号与二战全部数据，不可恢复。',
        '清空人机', { type: 'error' }).then(() => {
        api.post('/admin/ezfy-bot/clear').then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已清空')
            this.load()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    toggle (row) {
      api.post('/admin/ezfy-bot/' + row.user_id + '/toggle', { enabled: row.enabled ? 0 : 1 }).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已更新'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    // ---- 编辑（昵称/声望/城市名/人口/建筑/科技/兵种/军官）----
    openEdit (row) {
      this.cur = row
      this.editTab = 'base'
      this.editBuildings = []
      this.editTechs = []
      this.editTroops = []
      this.editOfficers = []
      this.editForm = { nickname: '', prestige: 0, city_name: '', pop: 0, pop_max: 0, pop_recover: 0,
        atk_per_hour: 0, atk_troop: 0, atk_per_hour_eff: 0, atk_troop_eff: 0,
        atk_type: 0, atk_type_eff: 3,
        chat_on: row.chat_on === undefined || row.chat_on === null ? 1 : row.chat_on,
        attack_wild: row.attack_wild ? 1 : 0 }
      this.editSnapA = { h: row.atk_per_hour || 0, t: row.atk_troop || 0, ty: row.atk_type || 0,
        aw: row.attack_wild ? 1 : 0 }
      this.editDlg = true
      this.editLoading = true
      api.get('/admin/ezfy-bot/' + row.user_id + '/buildings').then(r => {
        this.editLoading = false
        if (r.code === 0) {
          this.editForm = {
            nickname: r.data.nickname || '', prestige: r.data.prestige || 0,
            city_name: r.data.city_name || '',
            pop: r.data.pop || 0, pop_max: r.data.pop_max || 0, pop_recover: r.data.pop_recover || 0,
            freePop: r.data.free_pop || 0,
            // ★ 2026-10-17 出征设置（原值 + 按难度生效值，列表接口已返回）
            atk_per_hour: row.atk_per_hour || 0, atk_troop: row.atk_troop || 0,
            atk_per_hour_eff: row.atk_per_hour_eff || 0, atk_troop_eff: row.atk_troop_eff || 0,
            atk_type: row.atk_type || 0, atk_type_eff: row.atk_type_eff || 3,
            chat_on: row.chat_on === undefined || row.chat_on === null ? 1 : row.chat_on,
            attack_wild: row.attack_wild ? 1 : 0
          }
          // 只列已建造的建筑（level>0），未建造的坑位是噪音
          this.editBuildings = (r.data.buildings || []).filter(x => x.level > 0)
          this.editTechs = r.data.techs || []
          this.editTroops = r.data.troops || []
          this.editOfficers = (r.data.officers || []).map(o => ({
            id: o.id, name: o.name, star: o.star, level: o.level, exp: o.exp,
            military: o.military, logistics: o.logistics, learning: o.learning,
            free_points: o.free_points || 0, skills: o.skills || []
          }))
          this.snapEdit()
          this.loadEditOptions()
        } else { this.editDlg = false; this.$message.error(r.msg) }
      }).catch(() => { this.editLoading = false })
    },
    loadEditOptions () {
      // 军官池（普通+名将）供「发放军官」下拉
      api.get('/admin/ezfy-generals', { params: { page: 1, size: 500 } }).then(r => {
        if (r.code === 0) this.generalOptions = r.data.list || []
      })
      // 技能配置（名去重）供军官技能下拉
      api.get('/admin/ezfy-skills', { params: { page: 1, size: 500 } }).then(r => {
        if (r.code === 0) {
          const seen = {}
          this.skillOptions = (r.data.list || []).filter(x => x.name && !seen[x.name] ? (seen[x.name] = 1, true) : false).map(x => x.name)
        }
      })
    },
    grantGeneral () {
      if (!this.grantGeneralId) { this.$message.warning('请先选择军官池/名将'); return }
      this.grantSaving = true
      api.post('/admin/ezfy-officers/grant', { user_id: this.cur.user_id, general_id: this.grantGeneralId }).then(r => {
        this.grantSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已发放')
          // 重新拉取军官列表
          api.get('/admin/ezfy-bot/' + this.cur.user_id + '/buildings').then(r2 => {
            if (r2.code === 0) {
              this.editOfficers = (r2.data.officers || []).map(o => ({
                id: o.id, name: o.name, star: o.star, level: o.level, exp: o.exp,
                military: o.military, logistics: o.logistics, learning: o.learning,
                free_points: o.free_points || 0, skills: o.skills || []
              }))
              this.snapEdit()
            }
          })
        } else this.$message.error(r.msg)
      }).catch(() => { this.grantSaving = false })
    },
    // ★ 2026-10-18 随机 3 个不重复技能（对齐人机自动配技能逻辑，不消耗技能书）
    randomSkills (o) {
      if (!this.skillOptions.length) { this.$message.warning('技能池为空，请稍后重试'); return }
      const pool = this.skillOptions.slice()
      for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        const t = pool[i]; pool[i] = pool[j]; pool[j] = t
      }
      o.skills = pool.slice(0, 3)
    },
    // 快照四个列表（保存时逐个 diff，只提交有变化的 —— 不变的 tab 传空数组，后端零 SQL）
    snapEdit () {
      this.editSnap = {
        b: JSON.stringify(this.editBuildings.map(x => ({ id: x.id, level: x.level }))),
        t: JSON.stringify(this.editTechs.map(x => ({ tech_id: x.tech_id, level: x.level }))),
        tr: JSON.stringify(this.editTroops.map(x => ({ troop_id: x.troop_id, count: x.count }))),
        o: JSON.stringify(this.editOfficers.map(x => ({ id: x.id, level: x.level, exp: x.exp,
          free_points: x.free_points, military: x.military, logistics: x.logistics,
          learning: x.learning, skills: x.skills })))
      }
    },
    saveEdit () {
      const bNow = JSON.stringify(this.editBuildings.map(x => ({ id: x.id, level: x.level })))
      const tNow = JSON.stringify(this.editTechs.map(x => ({ tech_id: x.tech_id, level: x.level })))
      const trNow = JSON.stringify(this.editTroops.map(x => ({ troop_id: x.troop_id, count: x.count })))
      const oNow = JSON.stringify(this.editOfficers.map(x => ({ id: x.id, level: x.level, exp: x.exp,
        free_points: x.free_points, military: x.military, logistics: x.logistics,
        learning: x.learning, skills: x.skills })))
      this.editSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/edit', {
        nickname: this.editForm.nickname,
        prestige: this.editForm.prestige,
        city_name: this.editForm.city_name,
        free_pop: this.editForm.freePop,
        pop_max: this.editForm.pop_max,
        pop_recover: this.editForm.pop_recover,
        buildings: bNow === this.editSnap.b ? [] : this.editBuildings.map(x => ({ id: x.id, level: x.level })),
        techs: tNow === this.editSnap.t ? [] : this.editTechs.map(x => ({ tech_id: x.tech_id, level: x.level })),
        troops: trNow === this.editSnap.tr ? [] : this.editTroops.map(x => ({ troop_id: x.troop_id, count: x.count })),
        officers: oNow === this.editSnap.o ? [] : this.editOfficers.map(x => ({ id: x.id, level: x.level, exp: x.exp,
          free_points: x.free_points, military: x.military, logistics: x.logistics,
          learning: x.learning, skills: x.skills })),
        // ★ 2026-10-17 出征设置：改了才提交（null=不更新；0=恢复按难度默认）
        atk_per_hour: this.editSnapA.h === this.editForm.atk_per_hour ? null : this.editForm.atk_per_hour,
        atk_troop: this.editSnapA.t === this.editForm.atk_troop ? null : this.editForm.atk_troop,
        atk_type: this.editSnapA.ty === this.editForm.atk_type ? null : this.editForm.atk_type,
        // ★ 2026-10-19 自动发言开关（基本信息 tab）
        chat_on: this.editForm.chat_on === undefined || this.editForm.chat_on === null
          ? 1 : (this.editForm.chat_on ? 1 : 0),
        // ★ 2026-10-25 打野地开关（基本信息 tab）：改了才提交（null=不更新；默认 0=不打玩家野地）
        attack_wild: (this.editSnapA.aw ? 1 : 0) === (this.editForm.attack_wild ? 1 : 0)
          ? null : (this.editForm.attack_wild ? 1 : 0)
      }).then(r => {
        this.editSaving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.editSaving = false })
    },
    actNow (row) {
      this.$confirm('让「' + row.nickname + '」立刻执行一轮决策？\n（会真实执行它这轮选中的动作：建/升建筑、造兵、宣战等）',
        '立即行动', { type: 'info' }).then(() => {
        api.post('/admin/ezfy-bot/' + row.user_id + '/act').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已执行'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    del (row) {
      this.$confirm('确认删除人机「' + row.nickname + '」？\n将物理删除它的家园账号与二战全部数据，不可恢复。',
        '删除人机', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-bot/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-10-19 发放/占领 tab：切进来拉当前资源 + 占领城市（原 openGrant/openOccupy 合并进编辑对话框）
    loadGrantTab () {
      this.grantForm = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, gather: 0 }
      this.grantCur = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, gather: this.cur.gather_count || 0 }
      api.get('/admin/ezfy-players/' + this.cur.user_id + '/detail').then(r => {
        if (r.code !== 0 || !r.data) return
        const cs = r.data.cities || []
        if (cs.length) {
          this.grantCur = {
            gold: cs[0].gold || 0, food: cs[0].food || 0, steel: cs[0].steel || 0,
            oil: cs[0].oil || 0, rare: cs[0].rare || 0
          }
        }
      })
      this.occList = []
      this.occPage = 1
      this.loadOccupy()
    },
    doGrant () {
      const g = this.grantForm
      if (!g.gold && !g.food && !g.steel && !g.oil && !g.rare && !g.gather) {
        this.$message.warning('请先填写要发放的数量')
        return
      }
      this.grantSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/grant', g).then(r => {
        this.grantSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已发放')
          this.load()
          this.loadGrantTab()
        } else this.$message.error(r.msg)
      }).catch(() => { this.grantSaving = false })
    },
    // ---- 攻击目标 ----
    openTargets (row) {
      this.cur = row
      this.tgtInput = ''
      this.targets = []
      this.tgtDlg = true
      this.loadTargets()
    },
    loadTargets () {
      this.tgtLoading = true
      api.get('/admin/ezfy-bot/' + this.cur.user_id + '/targets').then(r => {
        this.tgtLoading = false
        if (r.code === 0) this.targets = r.data.list || []
        else this.$message.error(r.msg)
      }).catch(() => { this.tgtLoading = false })
    },
    addTarget () {
      const v = parseInt(this.tgtInput) || 0
      if (!v) { this.$message.warning('请填写目标玩家ID或游戏号'); return }
      this.tgtSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/target', { target_uid: v }).then(r => {
        this.tgtSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已加入计划')
          this.tgtInput = ''
          this.loadTargets()
        } else this.$message.error(r.msg)
      }).catch(() => { this.tgtSaving = false })
    },
    delTarget (row) {
      api.delete('/admin/ezfy-bot/target/' + row.id).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadTargets() }
        else this.$message.error(r.msg)
      })
    },
    // ★ 2026-10-17 出征最近敌对真人城（非自己、非同盟，自动宣战出击）
    attackNearest () {
      if (!this.cur) return
      this.tgtSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/attack-nearest').then(r => {
        this.tgtSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已锁定')
          this.loadTargets()
        } else this.$message.error(r.msg)
      }).catch(() => { this.tgtSaving = false })
    },
    // ---- 迁城（管理端直接输入坐标） ----
    openRelocate (row) {
      this.cur = row
      this.relocateForm = { x: row.x || 1, y: row.y || 1 }
      this.relocateDlg = true
    },
    doRelocate () {
      this.relocateSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/relocate', this.relocateForm).then(r => {
        this.relocateSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已迁移')
          this.relocateDlg = false
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.relocateSaving = false })
    },
    openLogs (row) {
      this.cur = row
      this.logs = []
      this.logTotal = 0
      this.logPage = 1
      this.logsDlg = true
      this.loadLogs()
    },
    loadLogs () {
      this.logsLoading = true
      api.get('/admin/ezfy-bot/logs', {
        params: { uid: this.cur ? this.cur.user_id : 0, page: this.logPage, size: this.logSize }
      }).then(r => {
        this.logsLoading = false
        if (r.code === 0) {
          this.logs = r.data.list || []
          this.logTotal = r.data.total
        }
      }).catch(() => { this.logsLoading = false })
    },
    // ★ 2026-10-18 物理删除单条行为日志
    delLog (row) {
      this.$confirm('确定删除该条行为日志？删除后不可恢复。', '删除日志', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-bot/log/' + row.id + '/delete').then(r => {
          if (r.code === 0) {
            this.$message.success('已删除')
            this.loadLogs()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-10-18 清空该人机全部行为日志
    clearLogs (row) {
      this.$confirm('确定清空「' + (row.nickname || row.nick_name || '该人机') + '」的全部行为日志？删除后不可恢复。',
        '清空日志', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-bot/logs/clear', { bot_uid: row.user_id }).then(r => {
          if (r.code === 0) {
            this.$message.success('已清空')
            this.logPage = 1
            this.loadLogs()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-10-17 出征战报：复用全局战报接口按 game_uid 过滤（人机 GameUID == UserID）
    openReports (row) {
      this.cur = row
      this.reports = []
      this.reportPage = 1
      this.reportsDlg = true
      this.loadReports()
    },
    loadReports () {
      this.reportsLoading = true
      api.get('/admin/ezfy-reports', { params: { uid: this.cur.user_id, page: this.reportPage, size: this.reportSize } }).then(r => {
        this.reportsLoading = false
        if (r.code === 0) {
          this.reports = r.data.list || []
          this.reportTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      }).catch(() => { this.reportsLoading = false })
    },
    // ★ 2026-10-18 人机占领的城市（已收进编辑对话框「发放/占领」tab，由 loadGrantTab 触发加载）
    loadOccupy () {
      this.occLoading = true
      api.get('/admin/ezfy-bot/' + this.cur.user_id + '/occupy', { params: { page: this.occPage, size: this.occSize } }).then(r => {
        this.occLoading = false
        if (r.code === 0) {
          this.occList = r.data.list || []
          this.occTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      }).catch(() => { this.occLoading = false })
    },
    // ★ 2026-10-18 占领城市操作（管理员代人机处理，与玩家端「附属野地」同口径）
    occReturn (row) {
      this.$confirm('将城市「' + row.city_name + '」（' + row.x + ',' + row.y + '）归还给原主人' + (row.def_name || ('ID ' + row.def_uid)) + '？',
        '归还城市', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-bot/occupy/' + row.id + '/op', { op: 'return' }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已归还'); this.loadOccupy() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    occDestroy (row) {
      this.$confirm('将城市「' + row.city_name + '」（' + row.x + ',' + row.y + '）彻底摧毁？\n建筑/部队/军官等全部清除，坐标恢复为平原，原主人将收到战报。',
        '摧毁城市', { type: 'error' }).then(() => {
        api.post('/admin/ezfy-bot/occupy/' + row.id + '/op', { op: 'destroy' }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已摧毁'); this.loadOccupy() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    occRedeem (row) {
      this.$confirm('同意「' + (row.def_name || ('ID ' + row.def_uid)) + '」赎回城市「' + row.city_name + '」？\n城市归还原主人，人机收到押金（已扣 20% 手续费）。',
        '同意赎回', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-bot/occupy/' + row.id + '/op', { op: 'redeem' }).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已同意赎回'); this.loadOccupy() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    redeemAllOccupy () {
      this.$confirm('统一同意该人机名下全部待处理的赎城申请？\n每单与「同意赎回」同口径（城市归还 + 人机收 80% 押金）。',
        '统一赎回', { type: 'warning' }).then(() => {
        this.occRedeemAllLoading = true
        api.post('/admin/ezfy-bot/occupy/redeem/all', { uid: this.cur.user_id }).then(r => {
          this.occRedeemAllLoading = false
          if (r.code === 0) { this.$message.success(r.data.msg || '完成'); this.loadOccupy() }
          else this.$message.error(r.msg)
        }).catch(() => { this.occRedeemAllLoading = false })
      }).catch(() => {})
    },
    // ★ 2026-10-17 战报详情：列表只给预览，点「详情」拉全文（content/detail）单独弹窗，列表不臃肿
    openReportDetail (row) {
      this.reportDetail = { title: row.title || row.type_name, content: '', detail: '' }
      this.reportDetailDlg = true
      api.get('/admin/ezfy-reports/' + row.id).then(r => {
        if (r.code === 0) {
          // ★ 2026-10-18 修复详情查询空：后端返回 { report, player_name, home_num, type_name }，正文在 report 里
          const d = r.data || {}
          this.reportDetail = d.report || this.reportDetail
        }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
/* ★ 2026-10-18 顶栏布局优化：两行分组，组间竖分隔线 */
.bot-bar { display: flex; flex-direction: column; gap: 10px; }
.bot-bar-row { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.bot-sep { width: 1px; height: 16px; background: #dcdfe6; margin: 0 6px; flex: none; }
.bot-item { color: #606266; font-size: 13px; white-space: nowrap; }
/* ★ label 提示图标（与「二战系统配置」页同款，鼠标悬停显示） */
.cfg-tip { margin-left: 3px; padding: 1px 3px; color: #909399; font-size: 13px; cursor: help; vertical-align: -1px; }
.cfg-tip:hover { color: #409eff; }
/* ★ 战报详情：全文白底黑字，保留换行；detail 与 content 之间留分隔 */
.report-pre { margin: 0 0 10px; padding: 10px; background: #fff; border: 1px solid #ebeef5;
  border-radius: 4px; white-space: pre-wrap; word-break: break-all; font-size: 13px; line-height: 1.7; }
.report-pre-detail { margin-top: 10px; }
</style>
