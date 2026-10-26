<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- ============ 玩家建筑 ============ -->
        <el-tab-pane label="玩家建筑" name="list">
          <div class="toolbar">
            <el-input v-model="word" placeholder="城名 / 城池ID" clearable style="width:200px"
                      @keyup.enter.native="page = 1; load()" />
            <el-select v-model="status" style="width:130px" @change="page = 1; load()">
              <el-option label="全部状态" :value="-1" />
              <el-option label="空闲" :value="0" />
              <el-option label="建造中" :value="1" />
              <el-option label="升级中" :value="2" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openCreate">给城池添加建筑</el-button>
        <!-- ★ 2026-10-05 本页已有「查询」按钮（点它就会重新 load），这个「刷新」按钮功能重复、容易误点 → 去掉。 -->
          </div>
          <el-table :data="list" v-loading="loading" stripe border max-height="620">
            <el-table-column prop="id" label="ID" width="75" align="center" />
            <el-table-column prop="cfg_name" label="建筑" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.building_id) }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="85" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="typeTag(row.cfg_type)">{{ buildTypes[row.cfg_type] || '—' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="等级" width="105" align="center">
              <template slot-scope="{row}">
                <span class="lv">Lv.</span>{{ row.level }}<span class="td-muted"> / {{ row.max_level }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="95" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 0 ? 'success' : 'warning'">{{ row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="city_name" label="所属城池" min-width="130" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" min-width="130" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="95" align="center" />
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
                <el-button size="mini" type="success" plain icon="el-icon-check" title="立即完成"
                           :disabled="row.status === 0" @click="finish(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="del(row)" />
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

        <!-- ============ 总建筑配置 ============ -->
        <el-tab-pane label="总建筑配置" name="cfg">
          <div class="toolbar">
            <el-input v-model="cfgWord" placeholder="建筑名 / ID" clearable style="width:200px"
                      @keyup.enter.native="cfgPage = 1; loadCfgs" />
            <el-button type="primary" icon="el-icon-search" @click="cfgPage = 1; loadCfgs">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openCfgCreate">新增建筑配置</el-button>
        <!-- ★ 2026-10-05 本页已有「查询」按钮（点它就会重新 load），这个「刷新」按钮功能重复、容易误点 → 去掉。 -->
          </div>
          <el-table :data="cfgPaged" v-loading="loadingCfg" stripe border max-height="620">
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column prop="name" label="建筑名" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="75" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="typeTag(row.type)">{{ row.type_name || buildTypes[row.type] }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="max_level" label="最高等级" width="90" align="center" />
            <el-table-column label="等级配置" width="100" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.level_count > 0 ? 'success' : 'danger'" style="cursor:pointer"
                        @click="openLevels(row)">
                  {{ row.level_count }} 条
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="唯一" width="70" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.unique_flag === 1 ? 'warning' : 'info'">{{ row.unique_flag === 1 ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="可拆除" width="80" align="center">
              <template slot-scope="{row}">{{ row.can_delete === 1 ? '是' : '否' }}</template>
            </el-table-column>
            <el-table-column prop="pre_building" label="前置建筑" width="130" show-overflow-tooltip />
            <el-table-column prop="des" label="说明" min-width="260" show-overflow-tooltip />
            <el-table-column label="操作" width="185" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="warning" plain icon="el-icon-s-order" title="等级配置" @click="openLevels(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openCfgEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delCfg(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ cfgShown.length }}</b> 条 · 每页 {{ cfgSize }} 条</div>
            <el-pagination v-show="cfgShown.length > 0" small background layout="sizes, prev, pager, next, jumper"
                           :total="cfgShown.length" :page-size="cfgSize"
                           :current-page="cfgPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { cfgPage = p }"
                           @size-change="s => { cfgSize = s; cfgPage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- ============ 建筑队列（★ 2026-10-26 由独立菜单「建筑队列管理」合并进来） ============ -->
        <el-tab-pane label="建筑队列" name="queue">
          <div class="toolbar">
            <el-input v-model="qWord" placeholder="城名 / 城池ID" clearable style="width:200px"
                      @keyup.enter.native="qPage = 1; loadQueue()" />
            <el-button type="primary" icon="el-icon-search" @click="qPage = 1; loadQueue()">查询</el-button>
            <div class="grow" />
            <el-tag size="mini" type="warning">共 {{ qTotal }} 项在队列</el-tag>
            <el-button type="warning" plain icon="el-icon-s-claim" @click="qFinishReady">完成可完成的</el-button>
            <el-button type="success" icon="el-icon-s-claim" @click="qFinishAll">一键完成全部</el-button>
          </div>
          <!-- ★ 列宽按实测内容宽度定：「模式」原 95px 装不下「一键满级连锁」标签（需 113px） -->
          <el-table :data="qList" v-loading="qLoading" stripe border max-height="620">
            <el-table-column prop="id" label="ID" width="65" align="center" />
            <el-table-column prop="cfg_name" label="建筑" min-width="140" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.building_id) }}</span></template>
            </el-table-column>
            <el-table-column label="等级" width="105" align="center">
              <template slot-scope="{row}">
                <span class="lv">Lv.</span>{{ row.level }} <span class="td-muted">→ {{ row.level + 1 }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" type="warning">{{ row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="剩余时间" width="110" align="center">
              <template slot-scope="{row}">
                <span :class="row.remain_sec <= 0 ? 'td-blue' : ''">{{ fmtRemain(row.remain_sec) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="模式" width="120" align="center">
              <template slot-scope="{row}">
                <el-tag v-if="row.chain_build === 1" size="mini" type="danger">一键满级连锁</el-tag>
                <span v-else class="td-sub">单次</span>
              </template>
            </el-table-column>
            <el-table-column prop="city_name" label="所属城池" min-width="120" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" min-width="120" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="90" align="center" />
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="success" plain @click="qFinish(row)">完成</el-button>
                <el-button size="mini" type="primary" plain @click="openSpeed(row)">加速</el-button>
                <el-button size="mini" type="warning" plain @click="qCancel(row)">取消</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ qTotal }}</b> 条 · 每页 {{ qSize }} 条</div>
            <el-pagination v-show="qTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="qTotal" :page-size="qSize"
                           :current-page="qPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { qPage = p; loadQueue() }"
                           @size-change="s => { qSize = s; qPage = 1; loadQueue() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 给城池添加建筑 -->
    <el-dialog title="给城池添加建筑" :visible.sync="createDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="城池ID" required>
          <el-input-number v-model.number="createForm.city_id" :min="1" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">可先在「玩家建筑」按城名查出城池ID</span>
        </el-form-item>
        <el-form-item label="建筑" required>
          <el-select v-model="createForm.building_id" filterable style="width:260px">
            <el-option v-for="b in buildCfgs" :key="b.id" :label="b.id + ' · ' + b.name + '（' + (buildTypes[b.type] || '') + '）'" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="初始等级">
          <el-input-number v-model.number="createForm.level" :min="1" controls-position="right" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="createDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">添 加</el-button>
      </div>
    </el-dialog>

    <!-- 编辑玩家建筑 -->
    <el-dialog title="编辑建筑" :visible.sync="editDlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="建筑">
          <span class="td-main">{{ form.cfg_name }}（{{ form.city_name }}）</span>
        </el-form-item>
        <el-form-item label="等级">
          <el-input-number v-model.number="form.level" :min="0" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">最高 {{ form.max_level }} 级</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :label="0">空闲</el-radio>
            <el-radio :label="1">建造中</el-radio>
            <el-radio :label="2">升级中</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 新增 / 编辑 建筑配置 -->
    <el-dialog :title="cfgForm.id ? ('编辑建筑配置 · ' + cfgForm.name) : '新增建筑配置'"
               :visible.sync="cfgDlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="建筑名" required>
          <el-input v-model="cfgForm.name" maxlength="50" style="width:280px" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model.number="cfgForm.type">
            <el-radio :label="1">资源</el-radio>
            <el-radio :label="2">军事</el-radio>
            <el-radio :label="3">城防</el-radio>
            <el-radio :label="4">市政</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="最高等级">
          <el-input-number v-model.number="cfgForm.max_level" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="唯一建筑">
          <el-switch v-model="cfgForm.unique_flag" :active-value="1" :inactive-value="0" />
          <span class="td-sub" style="margin-left:8px">全城只能有 1 座（如市政厅）</span>
        </el-form-item>
        <el-form-item label="可拆除">
          <el-switch v-model="cfgForm.can_delete" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="前置建筑">
          <el-input v-model="cfgForm.pre_building" maxlength="255" placeholder="例如：市政厅3级" style="width:280px" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="cfgForm.des" type="textarea" :rows="2" maxlength="500" />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 保存后立即生效（后端会重载配置缓存） -->
      <div slot="footer">
        <el-button @click="cfgDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCfgSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 等级配置 -->
    <el-dialog :title="'等级配置 · ' + lvBuilding.name + '（' + lvRows.length + ' 条）'"
               :visible.sync="lvDlg" width="1080px" top="6vh" :close-on-click-modal="false">
      <div class="toolbar">
        <el-button type="success" size="small" icon="el-icon-plus" @click="openLvCreate">新增等级</el-button>
        <div class="grow" />
        <span class="td-sub">建造耗时单位：秒；容量对仓储/农田等生效</span>
      </div>
      <el-table :data="lvRows" v-loading="loadingLv" size="mini" stripe border max-height="460">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="level" label="等级" width="55" align="center">
          <template slot-scope="{row}"><span class="lv">Lv.</span>{{ row.level }}</template>
        </el-table-column>
        <el-table-column prop="pop" label="人口" width="65" align="center" />
        <el-table-column label="粮" width="85" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.food) }}</span></template></el-table-column>
        <el-table-column label="钢" width="85" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.steel) }}</span></template></el-table-column>
        <el-table-column label="油" width="85" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.oil) }}</span></template></el-table-column>
        <el-table-column label="稀" width="85" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.rare) }}</span></template></el-table-column>
        <el-table-column label="黄金" width="85" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.gold) }}</span></template></el-table-column>
        <el-table-column label="建造(秒)" width="80" align="center"><template slot-scope="{row}"><span class="td-mono">{{ row.build_time }}</span></template></el-table-column>
        <el-table-column label="容量" width="90" align="center"><template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.capacity) }}</span></template></el-table-column>
        <el-table-column prop="effect" label="效果" min-width="130" show-overflow-tooltip />
        <el-table-column label="操作" width="130" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openLvEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delLv(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 新增 / 编辑 等级配置 -->
    <el-dialog :title="lvForm.id ? ('编辑等级配置 · Lv.' + lvForm.level) : '新增等级配置'"
               :visible.sync="lvEditDlg" width="900px" append-to-body :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="等级" required>
              <el-input-number v-model.number="lvForm.level" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="人口">
              <el-input-number v-model.number="lvForm.pop" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="建造(秒)">
              <el-input-number v-model.number="lvForm.build_time" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="6"><el-form-item label="粮"><el-input-number v-model.number="lvForm.food" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="钢"><el-input-number v-model.number="lvForm.steel" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="油"><el-input-number v-model.number="lvForm.oil" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="稀"><el-input-number v-model.number="lvForm.rare" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="黄金"><el-input-number v-model.number="lvForm.gold" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="容量"><el-input-number v-model.number="lvForm.capacity" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="效果">
          <el-input v-model="lvForm.effect" maxlength="500" placeholder="例如：每小时产粮 +1200" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="lvEditDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doLvSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 建筑队列·加速建造（★ 2026-10-26 由独立菜单合并进来） -->
    <el-dialog title="加速建造" :visible.sync="speedDlg" width="420px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="建筑">
          <span class="td-main">{{ speedRow.cfg_name }}（{{ speedRow.city_name }}）</span>
        </el-form-item>
        <el-form-item label="加速分钟">
          <el-input-number v-model.number="speedMinutes" :min="1" :max="1440" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button size="mini" plain @click="speedMinutes = 10">10 分钟</el-button>
          <el-button size="mini" plain @click="speedMinutes = 60">1 小时</el-button>
          <el-button size="mini" plain @click="speedMinutes = 360">6 小时</el-button>
          <el-button size="mini" plain @click="speedMinutes = 1440">24 小时</el-button>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="speedDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSpeed">加 速</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const CFG_KEYS = ['name', 'type', 'max_level', 'unique_flag', 'can_delete', 'pre_building', 'des']
const LV_KEYS = ['level', 'pop', 'food', 'steel', 'oil', 'rare', 'gold', 'build_time', 'capacity', 'effect']

export default {
  name: 'AdminEzfyBuildings',
  data () {
    return {
      tab: 'list',
      list: [], total: 0, page: 1, size: 5, loading: false, word: '', status: -1,
      buildTypes: { 1: '资源', 2: '军事', 3: '城防', 4: '市政' },
      // 玩家建筑
      createDlg: false, createForm: { city_id: 1, building_id: 0, level: 1 },
      editDlg: false, editId: 0, form: {},
      // 总建筑配置
      buildCfgs: [], cfgWord: '', loadingCfg: false, cfgPage: 1, cfgSize: 5,
      cfgDlg: false, cfgForm: {},
      // 等级配置
      lvDlg: false, lvBuilding: {}, lvRows: [], loadingLv: false,
      lvEditDlg: false, lvForm: {},
      // ★ 2026-10-26 建筑队列（原独立菜单「建筑队列管理」合并成这里的 tab）
      qList: [], qTotal: 0, qPage: 1, qSize: 5, qLoading: false, qWord: '',
      speedDlg: false, speedRow: {}, speedMinutes: 10,
      saving: false
    }
  },
  computed: {
    cfgShown () {
      const w = String(this.cfgWord || '').trim().toLowerCase()
      if (!w) return this.buildCfgs
      return this.buildCfgs.filter(b =>
        String(b.id) === w || String(b.name || '').toLowerCase().indexOf(w) >= 0
      )
    },
    // 总建筑配置是后端一次性返回的全量列表，这里做前端切片分页
    cfgPaged () {
      const st = (this.cfgPage - 1) * this.cfgSize
      return this.cfgShown.slice(st, st + this.cfgSize)
    }
  },
  mounted () { this.load(); this.loadCfgs() },
  methods: {
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    typeTag (t) {
      return ({ 1: 'success', 2: 'danger', 3: 'warning', 4: 'primary' })[t] || 'info'
    },
    onTab () {
      if (this.tab === 'cfg') this.loadCfgs()
      else if (this.tab === 'queue') this.loadQueue()
    },
    // ---- 建筑队列（★ 2026-10-26 从独立页面合并进来） ----
    fmtRemain (sec) {
      if (sec <= 0) return '可完成'
      const h = Math.floor(sec / 3600)
      const m = Math.floor((sec % 3600) / 60)
      const s = sec % 60
      if (h > 0) return h + ' 时 ' + m + ' 分'
      if (m > 0) return m + ' 分 ' + s + ' 秒'
      return s + ' 秒'
    },
    loadQueue () {
      this.qLoading = true
      api.get('/admin/ezfy-build-queue', { params: { page: this.qPage, size: this.qSize, word: this.qWord } }).then(r => {
        this.qLoading = false
        if (r.code === 0) {
          this.qList = r.data.list || []
          this.qTotal = r.data.total || 0
          this.qPage = r.data.page || 1
        } else this.$message.error(r.msg)
      }).catch(() => { this.qLoading = false })
    },
    qFinish (row) {
      api.post('/admin/ezfy-build-queue/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已完成'); this.loadQueue() } else this.$message.error(r.msg)
      })
    },
    openSpeed (row) {
      this.speedRow = row
      this.speedMinutes = 10
      this.speedDlg = true
    },
    doSpeed () {
      this.saving = true
      api.post('/admin/ezfy-build-queue/' + this.speedRow.id + '/speed', { minutes: this.speedMinutes }).then(r => {
        this.saving = false
        if (r.code === 0) { this.speedDlg = false; this.$message.success(r.data.msg || '已加速'); this.loadQueue() }
        else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    qCancel (row) {
      this.$confirm('取消后该建筑的建造/升级进度将丢失（等级不变），确认取消？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-build-queue/' + row.id + '/cancel').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已取消'); this.loadQueue() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    qFinishAll () {
      if (!this.qTotal) { this.$message.info('当前没有建造中的建筑'); return }
      this.$confirm('将立即完成队列中全部 ' + this.qTotal + ' 个建筑（每个完成一步），确认继续？', '一键完成', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-build-queue/finish-all').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已全部完成'); this.loadQueue() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 完成所有「剩余时间已到」的建筑 —— 定期手动点，替代自动协程
    qFinishReady () {
      this.$confirm('将立即完成所有「剩余时间已到」的建筑（未到时间的不动），确认继续？', '完成可完成的', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-build-queue/finish-ready').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已完工'); this.loadQueue() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 玩家建筑 ----
    load () {
      this.loading = true
      api.get('/admin/ezfy-buildings', {
        params: { page: this.page, size: this.size, word: this.word, status: this.status }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openCreate () {
      this.createForm = { city_id: 1, building_id: this.buildCfgs.length ? this.buildCfgs[0].id : 0, level: 1 }
      this.createDlg = true
    },
    doCreate () {
      if (!this.createForm.building_id) { this.$message.warning('请选择建筑'); return }
      this.saving = true
      api.post('/admin/ezfy-buildings', this.createForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.createDlg = false; this.$message.success(r.data.msg || '已添加'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    openEdit (row) {
      this.editId = row.id
      this.form = {
        cfg_name: row.cfg_name, city_name: row.city_name, max_level: row.max_level,
        level: row.level, status: row.status
      }
      this.editDlg = true
    },
    doEdit () {
      this.saving = true
      api.put('/admin/ezfy-buildings/' + this.editId, { level: this.form.level, status: this.form.status }).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    finish (row) {
      api.post('/admin/ezfy-buildings/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已完成'); this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确认删除建筑「' + (row.cfg_name || row.id) + '」？删除后该城池将失去此建筑。', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-buildings/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 总建筑配置 ----
    loadCfgs () {
      this.loadingCfg = true
      api.get('/admin/ezfy-building-cfg', { params: { word: this.cfgWord } }).then(r => {
        this.loadingCfg = false
        if (r.code === 0) {
          this.buildCfgs = r.data.list
          // 删除/筛选后当前页可能越界，回退到最后一页，避免表格空白
          const maxPage = Math.max(1, Math.ceil(this.cfgShown.length / this.cfgSize))
          if (this.cfgPage > maxPage) this.cfgPage = maxPage
        } else this.$message.error(r.msg)
      })
    },
    openCfgCreate () {
      this.cfgForm = { name: '', type: 2, max_level: 10, unique_flag: 0, can_delete: 1, pre_building: '', des: '' }
      this.cfgDlg = true
    },
    openCfgEdit (row) {
      const f = { id: row.id }
      CFG_KEYS.forEach(k => { f[k] = row[k] })
      this.cfgForm = f
      this.cfgDlg = true
    },
    doCfgSave () {
      if (!String(this.cfgForm.name || '').trim()) { this.$message.warning('建筑名不能为空'); return }
      const isNew = !this.cfgForm.id
      const body = {}
      CFG_KEYS.forEach(k => { body[k] = this.cfgForm[k] })
      this.saving = true
      const req = isNew
        ? api.post('/admin/ezfy-building-cfg', body)
        : api.put('/admin/ezfy-building-cfg/' + this.cfgForm.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.cfgDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadCfgs() }
        else this.$message.error(r.msg)
      })
    },
    delCfg (row) {
      this.$confirm('删除建筑配置「' + row.name + '」会同时删除它的 ' + row.level_count +
        ' 条等级配置，以及玩家已建的该建筑实例。确认删除？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-building-cfg/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadCfgs(); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 等级配置 ----
    openLevels (row) {
      this.lvBuilding = row
      this.lvDlg = true
      this.loadLevels()
    },
    loadLevels () {
      this.loadingLv = true
      api.get('/admin/ezfy-building-levels', { params: { building_id: this.lvBuilding.id } }).then(r => {
        this.loadingLv = false
        if (r.code === 0) this.lvRows = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openLvCreate () {
      const nextLv = this.lvRows.length ? Math.max.apply(null, this.lvRows.map(x => x.level)) + 1 : 1
      this.lvForm = {
        building_id: this.lvBuilding.id, level: nextLv, pop: 0,
        food: 0, steel: 0, oil: 0, rare: 0, gold: 0, build_time: 60, capacity: 0, effect: ''
      }
      this.lvEditDlg = true
    },
    openLvEdit (row) {
      const f = { id: row.id, building_id: row.building_id }
      LV_KEYS.forEach(k => { f[k] = row[k] })
      this.lvForm = f
      this.lvEditDlg = true
    },
    doLvSave () {
      const isNew = !this.lvForm.id
      const body = { building_id: this.lvForm.building_id }
      LV_KEYS.forEach(k => { body[k] = this.lvForm[k] })
      this.saving = true
      const req = isNew
        ? api.post('/admin/ezfy-building-levels', body)
        : api.put('/admin/ezfy-building-levels/' + this.lvForm.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.lvEditDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadLevels(); this.loadCfgs() }
        else this.$message.error(r.msg)
      })
    },
    delLv (row) {
      this.$confirm('确认删除 Lv.' + row.level + ' 的等级配置？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-building-levels/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadLevels(); this.loadCfgs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.td-small { font-size: 12px; }
</style>
