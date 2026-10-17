<template>
  <div class="farm-admin">
    <!-- 顶栏：统计 + 风控 + 生成/清空（★ 说明一律 tooltip，不上页面） -->
    <el-card shadow="never" class="box" style="margin-bottom:12px">
      <div class="toolbar">
        <span class="td-main">人机 <b>{{ list.length }}</b>/{{ max }}
          <el-tooltip placement="top" :content="tips.overview"><i class="el-icon-info cfg-tip" /></el-tooltip>
        </span>
        <el-divider direction="vertical" />
        <span>攻击玩家
          <el-tooltip placement="top" :content="tips.atk"><i class="el-icon-info cfg-tip" /></el-tooltip>
        </span>
        <el-switch v-model="attackOn" :active-value="1" :inactive-value="0" @change="saveConfig" />
        <span>每人每日出征</span>
        <el-input-number v-model.number="atkDailyCap" :min="0" :max="100" size="mini"
                         controls-position="right" style="width:90px" @change="saveConfig" />
        <span>人机免耗粮
          <el-tooltip placement="top" :content="tips.foodFree"><i class="el-icon-info cfg-tip" /></el-tooltip>
        </span>
        <el-switch v-model="foodFree" :active-value="1" :inactive-value="0" @change="saveConfig" />
        <span>最低声望
          <el-tooltip placement="top" :content="tips.minPrestige"><i class="el-icon-info cfg-tip" /></el-tooltip>
        </span>
        <el-input-number v-model.number="minPrestige" :min="0" :step="100000"
                         controls-position="right" style="width:130px" @change="saveConfig" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-video-play" @click="openActAll">一键立即行动</el-button>
        <el-button type="primary" icon="el-icon-plus" @click="createDlg = true">生成人机</el-button>
        <el-button type="danger" plain icon="el-icon-delete" :disabled="!list.length" @click="clearAll">清空全部</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="昵称 / 用户ID 搜索" clearable style="width:220px"
                  @keyup.enter.native="load" @clear="load" />
        <el-button type="primary" icon="el-icon-search" @click="load">查询</el-button>
        <el-button icon="el-icon-refresh" :loading="loading" @click="load">刷新</el-button>
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
        <el-table-column label="市政厅" width="80" align="center">
          <template slot-scope="{row}">{{ row.city_level || '—' }}</template>
        </el-table-column>
        <el-table-column label="兵力" width="110" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.troop_total) }}</span></template>
        </el-table-column>
        <el-table-column label="声望" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.prestige) }}</span></template>
        </el-table-column>
        <el-table-column label="最后行动" width="170" align="center">
          <template slot-scope="{row}">
            <span v-if="row.last_act_at">{{ fmtTime(row.last_act_at) }}</span>
            <span v-else class="td-gray">未行动</span>
            <div class="td-gray" style="font-size:12px">{{ row.last_action || '' }}</div>
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
        <el-table-column label="操作" width="440" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" plain icon="el-icon-edit" title="编辑（昵称/声望/城市名/建筑）" @click="openEdit(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-present" title="发放资源" @click="openGrant(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-video-play" title="立即行动一轮" @click="actNow(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-aim" title="攻击目标（查看/指定/删除）" @click="openTargets(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-map-location" title="迁城（输入坐标）" @click="openRelocate(row)" />
            <el-button size="mini" plain icon="el-icon-tickets" title="行为日志" @click="openLogs(row)" />
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
              <el-form-item label="空闲人口">
                <el-input-number v-model.number="editForm.freePop" :min="0" :step="1000"
                                 controls-position="right" style="width:160px" />
                <el-tooltip placement="top" content="空闲人口 = 总人口 − 建筑占用 − 训练占用 − 待恢复；保存时自动反推总人口，训练占用不受影响"><i class="el-icon-info cfg-tip" style="margin-left:6px" /></el-tooltip>
                <span class="td-gray" style="margin-left:6px">总人口 {{ fmtNum(editForm.pop) }}（自动反推）</span>
              </el-form-item>
              <el-form-item label="人口上限">
                <el-input-number v-model.number="editForm.pop_max" :min="0" :step="1000"
                                 controls-position="right" style="width:160px" />
              </el-form-item>
              <el-form-item label="待恢复人口">
                <el-input-number v-model.number="editForm.pop_recover" :min="0" :step="1000"
                                 controls-position="right" style="width:160px" />
                <span class="td-gray" style="margin-left:6px">空闲人口 = 总人口 − 建筑/训练占用 − 待恢复</span>
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
              <el-table-column label="等级" width="110" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.level" :min="1" :max="350" size="mini"
                                   controls-position="right" style="width:90px" />
                </template>
              </el-table-column>
              <el-table-column label="经验" width="120" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.exp" :min="0" :step="1000" size="mini"
                                   controls-position="right" style="width:100px" />
                </template>
              </el-table-column>
              <el-table-column label="待分配点" width="110" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.free_points" :min="0" :step="10" size="mini"
                                   controls-position="right" style="width:90px" />
                </template>
              </el-table-column>
              <el-table-column label="军事" width="100" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.military" :min="0" :step="10" size="mini"
                                   controls-position="right" style="width:85px" />
                </template>
              </el-table-column>
              <el-table-column label="后勤" width="100" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.logistics" :min="0" :step="10" size="mini"
                                   controls-position="right" style="width:85px" />
                </template>
              </el-table-column>
              <el-table-column label="学识" width="100" align="center">
                <template slot-scope="{row}">
                  <el-input-number v-model="row.learning" :min="0" :step="10" size="mini"
                                   controls-position="right" style="width:85px" />
                </template>
              </el-table-column>
              <el-table-column label="技能（最多3个）" min-width="200">
                <template slot-scope="{row}">
                  <el-select v-model="row.skills" multiple filterable collapse-tags size="mini" style="width:190px"
                             placeholder="选技能">
                    <el-option v-for="s in skillOptions" :key="s" :label="s" :value="s" />
                  </el-select>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="editSaving" @click="saveEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 发放资源 -->
    <el-dialog :title="'发放资源 —— ' + (cur ? cur.nickname : '')" :visible.sync="grantDlg" width="500px"
               :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item v-for="f in resFields" :key="f.k" :label="f.t">
          <el-input-number v-model.number="grantForm[f.k]" :step="10000" :precision="0"
                           controls-position="right" style="width:160px" />
          <span class="td-gray" style="margin-left:8px">当前 {{ fmtNum(grantCur[f.k]) }}</span>
        </el-form-item>
      </el-form>
      <div class="td-gray" style="font-size:12px">
        可填负数扣除；发完人机会立刻行动消化资源；不发钻石。
      </div>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="grantSaving" @click="doGrant">发 放</el-button>
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
        <el-table-column prop="detail" label="详情" min-width="300" show-overflow-tooltip />
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ logTotal }}</b> 条</div>
        <el-pagination v-show="logTotal > 0" small background layout="prev, pager, next" :total="logTotal"
                       :page-size="logSize" :current-page="logPage"
                       @current-change="p => { logPage = p; loadLogs() }" />
      </div>
      <div slot="footer"><el-button @click="logsDlg = false">关 闭</el-button></div>
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
      attackOn: 1, atkDailyCap: 3, foodFree: 1, minPrestige: 500000, saving: false,
      // 攻击目标
      tgtDlg: false, tgtLoading: false, tgtSaving: false, tgtInput: '', targets: [],
      // 迁城（管理端直接输入坐标）
      relocateDlg: false, relocateSaving: false, relocateForm: { x: 1, y: 1 },
      // 编辑快照（保存时只提交有变化的 tab —— 修复「保存卡顿」：几十条 UPDATE 串行跨 WAN 太慢）
      editSnap: { b: '', t: '', tr: '', o: '' },
      createDlg: false, createCount: 3, createLevel: 2, createBoost: true,
      // 一键立即行动（全部人机连跑 N 轮）
      actAllDlg: false, actAllSaving: false, actAllRounds: 5,
      // 编辑（昵称/声望/城市名/人口/建筑/科技/兵种/军官）
      editDlg: false, editLoading: false, editSaving: false, editTab: 'base',
      editForm: { nickname: '', prestige: 0, city_name: '', pop: 0, pop_max: 0, pop_recover: 0, freePop: 0 },
      editBuildings: [],
      editTechs: [],
      editTroops: [],
      editOfficers: [],
      // 发放军官 + 技能下拉
      grantGeneralId: 0, generalOptions: [], skillOptions: [],
      // 发放资源
      grantDlg: false, grantSaving: false,
      grantForm: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      grantCur: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      resFields: [
        { k: 'gold', t: '黄金' }, { k: 'food', t: '粮食' }, { k: 'steel', t: '钢铁' },
        { k: 'oil', t: '石油' }, { k: 'rare', t: '稀矿' }
      ],
      // 行为日志
      logsDlg: false, logs: [], logTotal: 0, logPage: 1, logSize: 20, logsLoading: false,
      cur: null,
      tips: {
        overview: '人机走和真人完全相同的接口（建城/升级/造兵/宣战），每 60 秒一轮、每轮最多 10 个，带租约锁（多进程只有一台真正执行）；人机不出现在「玩家信息管理」里。',
        atk: '关闭 = 人机只发育、不打扰玩家；开启后受「每人每日出征上限」与新手保护约束（建城不满 3 天、兵力过低的目标不会被打）。',
        foodFree: '开启后人机城内军队不扣维护粮、出征也免油耗（真人照常扣），让人机专注发育与出击。',
        minPrestige: '人机自动挑攻击目标时，只打声望 ≥ 此值的真人玩家（0 = 不限制）；声望越高越优先被选中。',
        boost: '给一笔启动资源（50 万粮 / 20 万钢油稀）。不开 = 严格从零（1 级市政厅 + 5000 资源），攒够召集人口的 20 万粮要很久；开 = 运营初始化，当天就能造兵。'
      }
    }
  },
  mounted () {
    this.load()
  },
  methods: {
    fmtNum (n) { return n == null ? 0 : Number(n).toLocaleString() },
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
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    saveConfig () {
      api.post('/admin/ezfy-bot/config', {
        attack_on: this.attackOn, atk_daily_cap: this.atkDailyCap, food_free: this.foodFree,
        min_prestige: this.minPrestige
      }).then(r => {
        if (r.code === 0) this.$message.success('已保存')
        else this.$message.error(r.msg)
      })
    },
    doCreate () {
      this.saving = true
      api.post('/admin/ezfy-bot/create', { count: this.createCount, level: this.createLevel, boost: this.createBoost }).then(r => {
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
      this.editForm = { nickname: '', prestige: 0, city_name: '', pop: 0, pop_max: 0, pop_recover: 0 }
      this.editDlg = true
      this.editLoading = true
      api.get('/admin/ezfy-bot/' + row.user_id + '/buildings').then(r => {
        this.editLoading = false
        if (r.code === 0) {
          this.editForm = {
            nickname: r.data.nickname || '', prestige: r.data.prestige || 0,
            city_name: r.data.city_name || '',
            pop: r.data.pop || 0, pop_max: r.data.pop_max || 0, pop_recover: r.data.pop_recover || 0,
            freePop: r.data.free_pop || 0
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
          learning: x.learning, skills: x.skills }))
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
    openGrant (row) {
      this.cur = row
      this.grantForm = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }
      this.grantCur = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }
      this.grantDlg = true
      api.get('/admin/ezfy-players/' + row.user_id + '/detail').then(r => {
        if (r.code !== 0 || !r.data) return
        const cs = r.data.cities || []
        if (cs.length) {
          this.grantCur = {
            gold: cs[0].gold || 0, food: cs[0].food || 0, steel: cs[0].steel || 0,
            oil: cs[0].oil || 0, rare: cs[0].rare || 0
          }
        }
      })
    },
    doGrant () {
      const g = this.grantForm
      if (!g.gold && !g.food && !g.steel && !g.oil && !g.rare) {
        this.$message.warning('请先填写要发放的数量')
        return
      }
      this.grantSaving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/grant', g).then(r => {
        this.grantSaving = false
        if (r.code === 0) {
          this.grantDlg = false
          this.$message.success(r.data.msg || '已发放')
          this.load()
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
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
/* ★ label 提示图标（与「二战系统配置」页同款，鼠标悬停显示） */
.cfg-tip { margin-left: 3px; padding: 1px 3px; color: #909399; font-size: 13px; cursor: help; vertical-align: -1px; }
.cfg-tip:hover { color: #409eff; }
</style>
