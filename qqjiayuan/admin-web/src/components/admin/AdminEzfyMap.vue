<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <!-- 坐标查询 -->
      <div class="toolbar">
        <span class="td-sub">坐标查询：</span>
        <el-input-number v-model.number="lookup.x" :min="0" :max="999" controls-position="right" style="width:130px" />
        <el-input-number v-model.number="lookup.y" :min="0" :max="999" controls-position="right" style="width:130px" />
        <el-button type="primary" icon="el-icon-search" @click="doLookup">查询</el-button>
        <div class="grow" />
      </div>
      <el-alert v-if="lookupResult" type="info" :closable="true" @close="lookupResult = null" show-icon style="margin-bottom:12px">
        <template slot="title">
          坐标 ({{ lookupResult.x }},{{ lookupResult.y }}) ·
          地形 {{ lookupResult.terrain_name }} ·
          大陆 {{ lookupResult.continent }} ·
          野地等级 {{ lookupResult.wild_level }}
          <span v-if="lookupResult.city"> · 城池「{{ lookupResult.city.name }}」（{{ lookupResult.city.player_name || '无主' }} · {{ lookupResult.city.home_num }}）</span>
          <span v-if="lookupResult.area"> · 区域类型 {{ areaTypes[lookupResult.area.area_type] || lookupResult.area.area_type }}</span>
        </template>
      </el-alert>

      <el-tabs v-model="tab" @tab-click="reload">
        <!-- ================= 城市坐标 ================= -->
        <el-tab-pane label="城市坐标" name="cities">
          <div class="toolbar">
            <el-input v-model="cityWord" placeholder="城名 / 用户ID" clearable style="width:200px"
                      @keyup.enter.native="cityPage = 1; loadCities()" />
            <el-button type="primary" icon="el-icon-search" @click="cityPage = 1; loadCities()">查询</el-button>
          </div>
          <el-table :data="cities" v-loading="loadingCity" stripe border>
            <el-table-column prop="id" label="城池ID" width="80" align="center" />
            <el-table-column prop="name" label="城名" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="terrain_name" label="地形" width="85" align="center" />
            <el-table-column prop="continent" label="大陆" width="90" align="center" />
            <el-table-column prop="city_level" label="市政厅" width="80" align="center" />
            <el-table-column prop="player_name" label="归属玩家" min-min-width="135" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="95" align="center" />
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ cityTotal }}</b> 条 · 每页 {{ citySize }} 条</div>
            <el-pagination v-show="cityTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="cityTotal" :page-size="citySize"
                           :current-page="cityPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { cityPage = p; loadCities() }"
                           @size-change="s => { citySize = s; cityPage = 1; loadCities() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 野地维护（全量 + 可编辑） ================= -->
        <!-- ================= 地图格子（改土地类型 / 设寇城·活动寇城） ================= -->
        <el-tab-pane label="地图格子" name="tiles">
          <div class="toolbar">
            <el-input-number v-model.number="tileX" :min="0" :max="999" controls-position="right" style="width:120px" />
            <el-input-number v-model.number="tileY" :min="0" :max="999" controls-position="right" style="width:120px" />
            <el-button type="primary" icon="el-icon-search" @click="loadTileCell">查询该格</el-button>
            <span class="td-sub">坐标 x / y（世界范围 50~450）</span>
            <div class="grow" />
            <el-select v-model="tileMarkFilter" style="width:150px" @change="tilePage = 1; loadTiles()">
              <el-option label="全部标记" :value="-1" />
              <el-option label="寇城" :value="1" />
              <el-option label="活动寇城" :value="2" />
              <el-option label="活动野地" :value="3" />
              <el-option label="特殊城市" :value="4" />
            </el-select>
            <el-input v-model="tileWord" placeholder="坐标 x,y 或备注" clearable style="width:170px"
                      @keyup.enter.native="tilePage = 1; loadTiles()" />
            <!-- ★ 2026-10-05 「管理端删除做好批量删除」：勾选后一次性物理删除 -->
            <el-button type="danger" plain icon="el-icon-delete" :disabled="!tileSel.length" @click="batchDelTiles">
              批量删除（已选 {{ tileSel.length }}）
            </el-button>
          </div>

          <el-alert type="info" :closable="false" show-icon style="margin-bottom:10px">
            <template slot="title">
              地图默认是「坐标哈希」推导的，这里做**覆盖**：改了立即生效（不用重启）。
              地形留「不覆盖」就按默认；标记留「无」也按默认。
            </template>
          </el-alert>

          <!-- 该格现状 + 编辑 -->
          <el-card shadow="never" class="box" v-if="tileCell">
            <div class="sub-title">
              坐标 ({{ tileCell.x }},{{ tileCell.y }}) 现状
              <span class="td-sub" v-if="tileCell.has_override">（已有覆盖）</span>
              <span class="td-sub" v-else>（按地图默认规则）</span>
            </div>
            <el-row :gutter="12">
              <el-col :span="8">默认地形：<b>{{ tileCell.hash_terrain_name }}</b></el-col>
              <el-col :span="8">默认标记：<b>{{ tileCell.hash_mark_name }}</b></el-col>
              <el-col :span="8">野地等级：<b>{{ tileCell.wildland_level }}</b></el-col>
            </el-row>
            <el-row :gutter="12" style="margin-top:6px">
              <el-col :span="8">生效地形：<b class="td-blue">{{ tileCell.eff_terrain_name }}</b></el-col>
              <el-col :span="8">生效标记：<b class="td-blue">{{ tileCell.eff_mark_name }}</b></el-col>
              <el-col :span="8">
                <span v-if="tileCell.city_name">该格已有城池：<b>{{ tileCell.city_name }}</b></span>
                <span v-else-if="tileCell.wild_owner">该格已被占：<b>{{ tileCell.wild_owner }}</b></span>
                <span v-else class="td-sub">该格没有城池/占领</span>
              </el-col>
            </el-row>
            <el-divider />
            <el-form label-width="110px" size="small" inline>
              <el-form-item label="土地类型">
                <el-select v-model.number="tileForm.terrain" style="width:150px">
                  <el-option label="不覆盖（按默认）" :value="0" />
                  <el-option v-for="(n, t) in terrainNames" :key="'tt' + t" :label="n" :value="Number(t)" />
                </el-select>
              </el-form-item>
              <el-form-item label="标记">
                <el-select v-model.number="tileForm.mark_kind" style="width:150px">
                  <el-option label="无（按默认）" :value="0" />
                  <el-option label="寇城" :value="1" />
                  <el-option label="活动寇城" :value="2" />
                  <el-option label="活动野地" :value="3" />
                  <el-option label="特殊城市" :value="4" />
                </el-select>
              </el-form-item>
              <el-form-item label="活动等级" v-if="tileForm.mark_kind >= 2">
                <el-input-number v-model.number="tileForm.mark_level" :min="1" :max="3"
                                 controls-position="right" style="width:120px" />
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="tileForm.des" maxlength="200" style="width:220px" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="saving" @click="saveTile">保存并生效</el-button>
                <el-button v-if="tileCell.has_override" type="danger" plain @click="clearTile">清除覆盖</el-button>
              </el-form-item>
            </el-form>
          </el-card>

          <el-table :data="tiles" v-loading="loadingTile" stripe border max-height="480"
                    @selection-change="s => tileSel = s">
            <el-table-column type="selection" width="46" align="center" />
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column label="坐标" width="110" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column label="土地类型" width="120" align="center">
              <template slot-scope="{row}">
                <span v-if="row.terrain > 0">{{ row.terrain_name }}</span>
                <span v-else class="td-sub">不覆盖</span>
              </template>
            </el-table-column>
            <el-table-column label="标记" width="120" align="center">
              <template slot-scope="{row}">
                <el-tag v-if="row.mark_kind > 0" size="mini"
                        :type="row.mark_kind === 2 ? 'warning' : (row.mark_kind === 4 ? 'danger' : (row.mark_kind === 3 ? 'success' : 'info'))">
                  {{ row.mark_name }}<span v-if="row.mark_kind >= 2">{{ row.mark_level }}级</span>
                </el-tag>
                <span v-else class="td-sub">无</span>
              </template>
            </el-table-column>
            <el-table-column prop="des" label="备注" min-width="180" show-overflow-tooltip />
            <el-table-column label="更新时间" width="170" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.updated_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="160" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑"
                           @click="tileX = row.x; tileY = row.y; loadTileCell()" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除覆盖" @click="delTile(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ tileTotal }}</b> 条 · 每页 {{ tileSize }} 条</div>
            <el-pagination v-show="tileTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="tileTotal" :page-size="tileSize"
                           :current-page="tilePage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { tilePage = p; loadTiles() }"
                           @size-change="s => { tileSize = s; tilePage = 1; loadTiles() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 活动野地配置 ================= -->
        <el-tab-pane label="活动野地" name="actwild">
          <div class="toolbar">
            <el-input v-model="awWord" placeholder="坐标 x,y 或备注" clearable style="width:180px"
                      @keyup.enter.native="awPage = 1; loadActWilds()" />
            <el-select v-model="awEnabled" style="width:130px" @change="awPage = 1; loadActWilds()">
              <el-option label="全部" :value="-1" />
              <el-option label="已启用" :value="1" />
              <el-option label="已关闭" :value="0" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="awPage = 1; loadActWilds()">查询</el-button>
            <span class="td-sub">活动野地 = 区别于普通野地、可打活动（守军/奖励可配）；关 = 普通野地</span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openAwCreate">新增活动野地</el-button>
            <!-- ★ 2026-10-05 「管理端删除做好批量删除」 -->
            <el-button type="danger" plain icon="el-icon-delete" :disabled="!awSel.length" @click="batchDelAws">
              批量删除（已选 {{ awSel.length }}）
            </el-button>
          </div>

          <el-alert type="info" :closable="false" show-icon style="margin-bottom:10px">
            <template slot="title">
              给坐标设置活动野地配置：启用开关打开 = 该格按活动野地玩法（玩家可出征打活动、赢奖励但不占领）；
              关闭或删除 = 该格按普通野地处理。等级/守军/奖励不填则用默认活动档。
            </template>
          </el-alert>

          <el-table :data="actWilds" v-loading="loadingAw" stripe border @selection-change="s => awSel = s">
            <el-table-column type="selection" width="46" align="center" />
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column label="坐标" width="110" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column label="启用" width="90" align="center">
              <template slot-scope="{row}">
                <el-switch :value="row.enabled === 1" @change="toggleAw(row)" />
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="70" align="center">
              <template slot-scope="{row}">{{ row.level || '默认' }}</template>
            </el-table-column>
            <el-table-column label="守军" min-width="170" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.troops" class="td-blue">{{ awTroopText(row.troops) }}</span>
                <span v-else class="td-sub">默认</span>
              </template>
            </el-table-column>
            <el-table-column label="奖励" min-width="150">
              <template slot-scope="{row}">
                <div v-if="row.res" class="td-mono">资源: {{ fmtN(row.res) }}</div>
                <div v-else class="td-sub">资源: 默认</div>
                <div v-if="row.gold" class="td-mono">黄金: {{ fmtN(row.gold) }}</div>
                <div v-else class="td-sub">黄金: 默认</div>
                <div v-if="row.prestige" class="td-mono">声望: {{ fmtN(row.prestige) }}</div>
                <div v-else class="td-sub">声望: 默认</div>
              </template>
            </el-table-column>
            <el-table-column label="宝物" min-width="110" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.treasures" class="td-blue">{{ awTreasureText(row.treasures) }}</span>
                <span v-else-if="row.jewel">{{ row.jewel }}</span>
                <span v-else class="td-sub">默认</span>
              </template>
            </el-table-column>
            <el-table-column label="守将" width="140" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.officer_id > 0" class="td-blue">{{ row.officer_name }}（{{ row.officer_star }}★）</span>
                <span v-else class="td-sub">无</span>
              </template>
            </el-table-column>
            <el-table-column prop="des" label="备注" min-width="140" show-overflow-tooltip />
            <el-table-column label="操作" width="215" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="success" plain icon="el-icon-view" title="查看被打记录" @click="openAwAttacks(row)">记录</el-button>
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openAwEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delAw(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ awTotal }}</b> 条 · 每页 {{ awSize }} 条</div>
            <el-pagination v-show="awTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="awTotal" :page-size="awSize"
                           :current-page="awPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { awPage = p; loadActWilds() }"
                           @size-change="s => { awSize = s; awPage = 1; loadActWilds() }" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="野地维护" name="wildlands">
          <div class="toolbar">
            <el-input v-model="wildWord" placeholder="城名 / 城池ID / 坐标" clearable style="width:190px"
                      @keyup.enter.native="wildPage = 1; loadWilds()" />
            <el-select v-model="wildType" style="width:130px" @change="wildPage = 1; loadWilds()">
              <el-option label="全部类型" :value="-1" />
              <el-option label="陆地野地" :value="1" />
              <el-option label="海野" :value="2" />
              <el-option label="寇城" :value="3" />
            </el-select>
            <el-select v-model="wildStatus" style="width:120px" @change="wildPage = 1; loadWilds()">
              <el-option label="全部状态" :value="-1" />
              <el-option label="空闲" :value="0" />
              <el-option label="采集中" :value="1" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="wildPage = 1; loadWilds()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openWildCreate">新增野地</el-button>
            <!-- ★ 2026-10-05 「管理端删除做好批量删除」 -->
            <el-button type="danger" plain icon="el-icon-delete" :disabled="!wildSel.length" @click="batchDelWilds">
              批量删除（已选 {{ wildSel.length }}）
            </el-button>
          </div>
          <el-table :data="wilds" v-loading="loadingWild" stripe border @selection-change="s => wildSel = s">
            <el-table-column type="selection" width="46" align="center" />
            <el-table-column prop="id" label="ID" width="65" align="center" />
            <el-table-column label="坐标" width="100" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="terrain_name" label="地形" width="85" align="center" />
            <el-table-column label="野地类型" width="95" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.wild_type === 2 ? 'primary' : (row.wild_type === 3 ? 'danger' : 'success')">
                  {{ row.type_name || wildTypes[row.wild_type] || row.wild_type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="65" align="center" />
            <el-table-column label="状态" width="85" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 1 ? 'warning' : 'success'">{{ row.status_name || row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="gain" label="产出" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-mono td-small">{{ row.gain || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="city_name" label="占领城池" min-width="125" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" min-min-width="135" show-overflow-tooltip />
            <el-table-column label="操作" width="215" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="info" plain icon="el-icon-view" title="该等级野地配置" @click="openWildCfgOf(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openWildEdit(row)" />
                <el-button size="mini" type="success" plain icon="el-icon-check" title="立即完成采集"
                           :disabled="row.status !== 1" @click="finishWild(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delWild(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ wildTotal }}</b> 条 · 每页 {{ wildSize }} 条</div>
            <el-pagination v-show="wildTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="wildTotal" :page-size="wildSize"
                           :current-page="wildPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { wildPage = p; loadWilds() }"
                           @size-change="s => { wildSize = s; wildPage = 1; loadWilds() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 野地类型维护 ================= -->
        <el-tab-pane label="野地类型" name="wildcfg">
          <div class="toolbar">
            <el-select v-model="wcType" style="width:140px" @change="wcPage = 1; loadWildCfgs()">
              <el-option label="全部类型" :value="-1" />
              <el-option label="陆地野地" :value="1" />
              <el-option label="海野" :value="2" />
              <el-option label="寇城" :value="3" />
            </el-select>
            <el-input-number v-model.number="wcLevel" :min="0" :max="20" controls-position="right" style="width:130px" />
            <el-button type="primary" icon="el-icon-search" @click="wcPage = 1; loadWildCfgs()">查询</el-button>
            <span class="td-sub">等级填 0 = 不限</span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openWcCreate">新增野地类型</el-button>
          </div>
          <el-table :data="wildCfgs" v-loading="loadingWc" stripe border>
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column label="类型" width="95" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.type === 2 ? 'primary' : (row.type === 3 ? 'danger' : 'success')">
                  {{ row.type_name }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="level" label="等级" width="65" align="center" />
            <el-table-column label="守军" min-width="220" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="troopText(row.troops)">{{ troopText(row.troops) }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column label="产出区间" width="150" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.res_min) }} ~ {{ fmtN(row.res_max) }}</span></template>
            </el-table-column>
            <el-table-column label="守军军官" width="140" align="center">
              <template slot-scope="{row}">
                <span v-if="row.officer_name" class="td-main">{{ row.officer_name }}</span>
                <span v-else class="td-sub">无</span>
              </template>
            </el-table-column>
            <el-table-column prop="treasure" label="宝物" width="120" show-overflow-tooltip />
            <el-table-column label="道具掉落" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.drop_items" class="td-blue">{{ row.drop_items }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column prop="des" label="说明" min-width="150" show-overflow-tooltip />
            <el-table-column label="操作" width="140" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openWcEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delWc(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ wcTotal }}</b> 条 · 每页 {{ wcSize }} 条</div>
            <el-pagination v-show="wcTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="wcTotal" :page-size="wcSize"
                           :current-page="wcPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { wcPage = p; loadWildCfgs() }"
                           @size-change="s => { wcSize = s; wcPage = 1; loadWildCfgs() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 占领记录 ================= -->
        <el-tab-pane label="占领记录" name="occupy">
          <div class="toolbar">
            <el-select v-model="occStatus" style="width:140px" @change="occPage = 1; loadOccupy()">
              <el-option label="全部" :value="-1" />
              <el-option label="占领中" :value="1" />
              <el-option label="已归还" :value="2" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="occPage = 1; loadOccupy()">查询</el-button>
          </div>
          <el-table :data="occupies" v-loading="loadingOcc" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="city_name" label="被占城池" min-width="140" show-overflow-tooltip />
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="atk_name" label="占领方" min-width="135" show-overflow-tooltip />
            <el-table-column prop="def_name" label="原属方" min-width="135" show-overflow-tooltip />
            <el-table-column label="状态" width="100" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 1 ? 'danger' : 'success'">{{ row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="时间" width="170" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="210" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="warning" plain :disabled="row.status !== 1" @click="release(row)">解除占领</el-button>
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delOccupy(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ occTotal }}</b> 条 · 每页 {{ occSize }} 条</div>
            <el-pagination v-show="occTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="occTotal" :page-size="occSize"
                           :current-page="occPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { occPage = p; loadOccupy() }"
                           @size-change="s => { occSize = s; occPage = 1; loadOccupy() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 地图区域 ================= -->
        <el-tab-pane label="地图区域（落库）" name="areas">
          <div class="toolbar">
            <el-select v-model="areaType" style="width:150px" @change="areaPage = 1; loadAreas()">
              <el-option label="全部类型" :value="-1" />
              <el-option label="空地" :value="0" />
              <el-option label="野地(已占)" :value="1" />
              <el-option label="寇城" :value="2" />
              <el-option label="玩家城" :value="3" />
              <el-option label="资源田" :value="4" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="areaPage = 1; loadAreas()">查询</el-button>
          </div>
          <el-table :data="areas" v-loading="loadingArea" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="type_name" label="区域类型" width="110" align="center" />
            <el-table-column prop="terrain_name" label="地形" width="85" align="center" />
            <el-table-column prop="level" label="等级" width="65" align="center" />
            <el-table-column prop="owner_id" label="占领城市ID" width="110" align="center" />
            <el-table-column prop="hp" label="耐久" width="95" align="center" />
            <el-table-column prop="troops" label="守军" min-width="180" show-overflow-tooltip />
            <el-table-column prop="resources" label="资源" min-width="140" show-overflow-tooltip />
            <el-table-column label="操作" width="100" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delArea(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ areaTotal }}</b> 条 · 每页 {{ areaSize }} 条</div>
            <el-pagination v-show="areaTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="areaTotal" :page-size="areaSize"
                           :current-page="areaPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { areaPage = p; loadAreas() }"
                           @size-change="s => { areaSize = s; areaPage = 1; loadAreas() }" />
          </div>
        </el-tab-pane>

        <!-- ================= 坐标收藏 ================= -->
        <el-tab-pane label="坐标收藏" name="stars">
          <el-table :data="stars" v-loading="loadingStar" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="owner_name" label="玩家" min-width="140" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="100" align="center" />
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="terrain_name" label="地形" width="85" align="center" />
            <el-table-column prop="name" label="备注名" min-width="150" show-overflow-tooltip />
            <el-table-column label="收藏时间" width="170" align="center">
              <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delStar(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ starTotal }}</b> 条 · 每页 {{ starSize }} 条</div>
            <el-pagination v-show="starTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="starTotal" :page-size="starSize"
                           :current-page="starPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { starPage = p; loadStars() }"
                           @size-change="s => { starSize = s; starPage = 1; loadStars() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- ============ 新增 / 编辑 玩家野地 ============ -->
    <el-dialog :title="wf.id ? ('编辑野地 #' + wf.id) : '新增野地'" :visible.sync="wildDlg"
               width="760px" :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="归属城池" required>
              <el-input-number v-model.number="wf.city_id" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="野地类型">
              <el-select v-model.number="wf.wild_type" style="width:100%">
                <el-option label="陆地野地" :value="1" />
                <el-option label="海野" :value="2" />
                <el-option label="寇城" :value="3" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="坐标 X">
              <el-input-number v-model.number="wf.x" :min="0" :max="999" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="坐标 Y">
              <el-input-number v-model.number="wf.y" :min="0" :max="999" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="等级">
              <el-input-number v-model.number="wf.level" :min="1" :max="20" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态">
              <el-select v-model.number="wf.status" style="width:100%">
                <el-option label="空闲" :value="0" />
                <el-option label="采集中" :value="1" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="产出">
          <el-input v-model="wf.gain" maxlength="500" placeholder="例如：粮食 12000 / 钢铁 3000" />
        </el-form-item>
        <el-alert v-if="wf.id && wildCfgMatch" type="info" :closable="false" show-icon style="margin-bottom:10px">
          <template slot="title">
            该坐标对应配置：{{ wildCfgMatch.type_name }} Lv.{{ wildCfgMatch.level }} ·
            产出区间 {{ fmtN(wildCfgMatch.res_min) }} ~ {{ fmtN(wildCfgMatch.res_max) }}
            <span v-if="wildCfgMatch.treasure"> · 宝物 {{ wildCfgMatch.treasure }}</span>
          </template>
        </el-alert>
      </el-form>
      <div slot="footer">
        <el-button @click="wildDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doWildSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增 / 编辑 野地类型配置 ============ -->
    <el-dialog :title="wc.id ? ('编辑野地类型 · ' + wc.type_name + ' Lv.' + wc.level) : '新增野地类型'"
               :visible.sync="wcDlg" width="800px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="野地类型" required>
              <el-select v-model.number="wc.type" style="width:100%">
                <el-option label="陆地野地" :value="1" />
                <el-option label="海野" :value="2" />
                <el-option label="寇城" :value="3" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="等级" required>
              <el-input-number v-model.number="wc.level" :min="1" :max="20" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="产出下限">
              <el-input-number v-model.number="wc.res_min" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="产出上限">
              <el-input-number v-model.number="wc.res_max" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="守军军官">
          <el-select v-model.number="wc.officer_id" filterable clearable placeholder="不设守将（打下来也俘不到军官）"
                     style="width:100%">
            <el-option v-for="g in generals" :key="'gen' + g.id"
                       :label="g.name + '（' + g.star + '星  军事' + g.military + ' 后勤' + g.logistics + ' 学识' + g.learning + '）'"
                       :value="g.id" />
          </el-select>
          <span class="td-sub">最多 1 个，且只能从军官池里选；打赢后有概率俘虏这名军官</span>
        </el-form-item>
        <el-form-item label="守军搭配">
          <!-- ★ 原来是裸 JSON 文本框（[[兵种ID,最小,最大],...]），改成可视化行编辑 -->
          <div v-for="(r, i) in wcTroops" :key="'wt' + i" class="wild-troop-row">
            <el-select v-model.number="r.troop_id" filterable placeholder="选择兵种" style="width:220px">
              <el-option v-for="t in troopCfgs" :key="'wtc' + t.id"
                         :label="t.name + '（' + t.type_name + '）'" :value="t.id" />
            </el-select>
            <span class="td-sub">数量</span>
            <el-input-number v-model.number="r.min" :min="0" controls-position="right" style="width:120px" />
            <span class="td-sub">~</span>
            <el-input-number v-model.number="r.max" :min="0" controls-position="right" style="width:120px" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="wcTroops.splice(i, 1)" />
          </div>
          <div class="old-line" v-if="!wcTroops.length">（暂无守军，野地将没有防守部队）</div>
          <el-button size="mini" type="success" plain icon="el-icon-plus" @click="addWcTroop">添加兵种</el-button>
        </el-form-item>
        <el-form-item label="宝物掉落">
          <!-- ★ 2026-10-05 宝物也搞成下拉选择 + 可配概率（运营不用手写 JSON） -->
          <div v-for="(r, i) in wcTreasures" :key="'wtv' + i" class="wild-troop-row">
            <el-select v-model="r.name" filterable placeholder="选择宝物" style="width:220px">
              <!-- ★ 2026-10-05 全部装备（含武器/防具/饰品），狙击步枪、参谋指北针等也能选并配概率 -->
              <el-option v-for="j in treasures" :key="'wtvj' + j.id" :label="j.name + '（' + j.type + '）'" :value="j.name" />
            </el-select>
            <span class="td-sub">数量</span>
            <el-input-number v-model.number="r.count" :min="1" controls-position="right" style="width:90px" />
            <span class="td-sub">概率%</span>
            <el-input-number v-model.number="r.pct" :min="1" :max="100" controls-position="right" style="width:90px" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="wcTreasures.splice(i, 1)" />
          </div>
          <div class="old-line" v-if="!wcTreasures.length">（未配置宝物掉落，打赢不掉宝物）</div>
          <el-button size="mini" type="success" plain icon="el-icon-plus" @click="addWcTreasure">添加宝物</el-button>
        </el-form-item>
        <el-form-item label="商城道具掉落">
          <!-- ★ 2026-10-05 下拉选择 + 数量 + 概率%（运营不用手写 JSON） -->
          <div v-for="(r, i) in wcDrops" :key="'wdp' + i" class="wild-troop-row">
            <el-select v-model.number="r.cfg_id" filterable placeholder="选择商城道具" style="width:220px">
              <el-option v-for="it in itemCfgs" :key="'wdp' + it.id" :label="it.name" :value="it.id" />
            </el-select>
            <span class="td-sub">数量</span>
            <el-input-number v-model.number="r.count" :min="1" controls-position="right" style="width:90px" />
            <span class="td-sub">概率%</span>
            <el-input-number v-model.number="r.pct" :min="1" :max="100" controls-position="right" style="width:90px" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="wcDrops.splice(i, 1)" />
          </div>
          <div class="old-line" v-if="!wcDrops.length">（未配置商城道具掉落，打赢不掉道具）</div>
          <el-button size="mini" type="success" plain icon="el-icon-plus" @click="addWcDrop">添加道具</el-button>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="wc.des" type="textarea" :rows="2" maxlength="500" show-word-limit />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 保存后立即生效（后端会重载配置缓存，不用重启） -->
      <div slot="footer">
        <el-button @click="wcDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doWcSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 新增 / 编辑 活动野地配置 ============ -->
    <el-dialog :title="aw.id ? ('编辑活动野地 · ' + aw.x + ',' + aw.y) : '新增活动野地'"
               :visible.sync="awDlg" width="620px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="X 坐标" required>
              <el-input-number v-model.number="aw.x" :min="1" :max="500" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Y 坐标" required>
              <el-input-number v-model.number="aw.y" :min="1" :max="500" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="启用">
              <el-switch v-model="aw.enabled" :active-value="1" :inactive-value="0"
                         active-text="活动野地" inactive-text="普通野地" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="活动等级">
              <el-input-number v-model.number="aw.level" :min="0" :max="3" controls-position="right" style="width:100%" />
              <span class="td-sub">0 = 默认</span>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="守军配置">
          <!-- ★ 2026-09-29 原来是裸 JSON 文本框（[[兵种id,数量],...]），对非程序员不友好，改成可视化行编辑 -->
          <div v-for="(r, i) in awTroops" :key="'awt' + i" class="wild-troop-row">
            <el-select v-model.number="r.troop_id" filterable placeholder="选择兵种" style="width:300px">
              <el-option v-for="t in troopCfgs" :key="'awtc' + t.id"
                         :label="t.name" :value="t.id" />
            </el-select>
            <span class="td-sub">数量</span>
            <el-input-number v-model.number="r.count" :min="0" controls-position="right" style="width:150px" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="awTroops.splice(i, 1)" />
          </div>
          <div class="old-line" v-if="!awTroops.length">（未配置守军，用默认活动守军）</div>
          <el-button size="mini" type="success" plain icon="el-icon-plus" @click="addAwTroop">添加兵种</el-button>
          <span class="td-sub" style="margin-left:8px">不填则用默认活动守军</span>
        </el-form-item>
        <el-form-item label="资源奖励">
          <el-input-number v-model.number="aw.res" :min="0" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="黄金奖励">
          <el-input-number v-model.number="aw.gold" :min="0" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="声望奖励">
          <el-input-number v-model.number="aw.prestige" :min="0" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="必掉宝物">
          <!-- ★ 2026-09-29 宝物配置优化：和守军一样支持多行（选宝物 + 数量），胜利后按配置掉落 -->
          <div v-for="(tr, ti) in awTreasures" :key="'awj' + ti" class="wild-troop-row">
            <el-select v-model.number="tr.treasure_id" filterable placeholder="选择宝物" style="width:300px">
              <el-option v-for="j in jewels" :key="'awjc' + j.id" :label="j.name" :value="j.id" />
            </el-select>
            <span class="td-sub">数量</span>
            <el-input-number v-model.number="tr.count" :min="1" controls-position="right" style="width:120px" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="awTreasures.splice(ti, 1)" />
          </div>
          <div class="old-line" v-if="!awTreasures.length">（未配置，用默认地形珠宝）</div>
          <el-button size="mini" type="success" plain icon="el-icon-plus" @click="addAwTreasure">添加宝物</el-button>
          <span class="td-sub" style="margin-left:8px">胜利后按配置掉落多件；不填则用默认地形珠宝</span>
        </el-form-item>
        <el-form-item label="守将军官">
          <div style="display:flex;align-items:center;gap:10px;width:100%;flex-wrap:wrap">
            <el-radio-group v-model="awOfficerKind" size="small" @change="awOfficerKindChange">
              <el-radio-button :label="1">普通</el-radio-button>
              <el-radio-button :label="2">名将</el-radio-button>
            </el-radio-group>
            <!-- ★ 2026-09-29 星级筛选：先选类型、再按星级缩小范围，方便检索 -->
            <el-select v-model.number="awStarFilter" clearable placeholder="星级" style="width:90px">
              <el-option v-for="s in awStarOptions" :key="'awst' + s" :label="s + '★'" :value="s" />
            </el-select>
            <el-select v-model.number="aw.officer_id" filterable clearable placeholder="不设守将（打赢也俘不到军官）" style="flex:1;min-width:180px"
                       :disabled="awOfficerKind === 0">
              <el-option v-for="g in awGeneralsFiltered" :key="'awg' + g.id"
                         :label="g.name + '（' + g.star + '星' + '）'"
                         :value="g.id" />
            </el-select>
          </div>
          <span class="td-sub">先选「普通/名将」→ 可再按星级筛选；不设则打赢俘不到军官</span>
        </el-form-item>
        <el-form-item label="被俘概率%">
          <el-input-number v-model.number="aw.capture_rate" :min="0" :max="100" controls-position="right" style="width:160px" />
          <span class="td-sub">0 = 不俘虏；1~100 按该百分比（100=必俘虏）</span>
        </el-form-item>
        <el-form-item label="可抓次数">
          <el-input-number v-model.number="aw.max_capture" :min="0" controls-position="right" style="width:160px" />
          <span class="td-sub">同一玩家最多可抓该守将几次（默认1，抓满后概率归0）；0 = 不限</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="aw.des" maxlength="200" show-word-limit />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="awDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="saveAw">保 存</el-button>
      </div>
    </el-dialog>

    <!-- ============ 活动野地 · 被打记录（2026-10-01） ============ -->
    <el-dialog title="活动野地被攻打记录" :visible.sync="awAttDlg" width="720px" :close-on-click-modal="false">
      <div class="td-sub" style="margin-bottom:10px" v-if="awAttRow">
        坐标：<b class="td-mono">{{ awAttRow.x }},{{ awAttRow.y }}</b>
        <span v-if="awAttRow.des"> · 备注：{{ awAttRow.des }}</span>
        <span v-if="awAttRow.enabled !== 1" style="color:#f56c6c">（当前已关闭，以下为历史记录）</span>
      </div>
      <el-table :data="awAttList" v-loading="awAttLoading" stripe border size="small">
        <el-table-column prop="id" label="战场ID" width="90" align="center" />
        <el-table-column label="攻打玩家" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-blue">{{ row.player_name || ('玩家' + row.user_id) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="目标" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.target_name || '活动野地' }}</template>
        </el-table-column>
        <el-table-column label="结果" width="90" align="center">
          <template slot-scope="{row}">
            <span :class="row.win === 1 ? 'td-green' : (row.win === 2 ? 'td-red' : 'td-sub')">{{ row.result }}</span>
          </template>
        </el-table-column>
        <el-table-column label="俘虏军官" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="row.captive && row.captive.indexOf('上限') >= 0" class="td-red">{{ row.captive }}</span>
            <span v-else-if="row.captive && row.captive !== '未俘虏'" class="td-green">{{ row.captive }}</span>
            <span v-else class="td-sub">未俘虏</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="150" align="center" />
      </el-table>
      <div class="pager-bar" style="margin-top:10px">
        <div class="pager-info">共 <b>{{ awAttTotal }}</b> 条</div>
        <el-pagination v-show="awAttTotal > 0" small background layout="sizes, prev, pager, next" :total="awAttTotal"
                       :page-size="awAttSize" :current-page="awAttPage" :page-sizes="[10, 20, 50]"
                       @current-change="p => { awAttPage = p; loadAwAttacks() }"
                       @size-change="s => { awAttSize = s; awAttPage = 1; loadAwAttacks() }" />
      </div>
      <div slot="footer">
        <el-button @click="awAttDlg = false">关 闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const WILD_KEYS = ['city_id', 'x', 'y', 'wild_type', 'level', 'gain', 'status']
const WC_KEYS = ['type', 'level', 'troops', 'res_min', 'res_max',
  'officer_min', 'officer_max', 'officer_id', 'treasure', 'drop_items', 'des']

function emptyWild () {
  return { id: 0, city_id: 1, x: 250, y: 250, wild_type: 1, level: 1, gain: '', status: 0 }
}
function emptyWc () {
  return { id: 0, type: 1, level: 1, troops: '', res_min: 0, res_max: 0,
    officer_min: 0, officer_max: 0, officer_id: 0, treasure: '', drop_items: '', des: '',
    // 弹窗内的可视化行（保存时序列化回 JSON 字段）
    wcTroops: [], wcDrops: [], wcTreasures: [] }
}

export default {
  name: 'AdminEzfyMap',
  data () {
    return {
      tab: 'cities',
      lookup: { x: 250, y: 250 }, lookupResult: null,
      wildTypes: { 1: '陆地野地', 2: '海野', 3: '寇城' },
      areaTypes: { 0: '空地', 1: '野地(已占)', 2: '寇城', 3: '玩家城', 4: '资源田' },
      cities: [], cityTotal: 0, cityPage: 1, citySize: 5, cityWord: '', loadingCity: false,
      wilds: [], wildTotal: 0, wildPage: 1, wildSize: 5, wildWord: '', wildType: -1, wildStatus: -1, loadingWild: false,
      // ★ 2026-10-05 「管理端删除做好批量删除」：三个列表各自的勾选集合
      tileSel: [], wildSel: [], awSel: [],
      wildCfgs: [], wcTotal: 0, wcPage: 1, wcSize: 5, wcType: -1, wcLevel: 0, loadingWc: false,
      occupies: [], occTotal: 0, occPage: 1, occSize: 5, occStatus: -1, loadingOcc: false,
      areas: [], areaTotal: 0, areaPage: 1, areaSize: 5, areaType: -1, loadingArea: false,
      stars: [], starTotal: 0, starPage: 1, starSize: 5, loadingStar: false,
      // 野地类型弹窗的下拉数据：兵种列表 + 军官池
      troopCfgs: [], generals: [],
      // 地图格子覆盖
      tiles: [], tileTotal: 0, tilePage: 1, tileSize: 5, loadingTile: false,
      tileX: 250, tileY: 250, tileCell: null, tileWord: '', tileMarkFilter: -1,
      tileForm: { terrain: 0, mark_kind: 0, mark_level: 1, des: '' },
      terrainNames: { 1: '平原', 2: '草原', 3: '森林', 4: '盆地', 5: '丘陵', 6: '沼泽', 7: '山地', 8: '海洋', 9: '沿海平原' },
      wildDlg: false, wf: emptyWild(), wildCfgMatch: null,
      wcDlg: false, wc: emptyWc(),
      // 活动野地配置（2026-09-29）
      actWilds: [], awTotal: 0, awPage: 1, awSize: 15, awWord: '', awEnabled: -1, loadingAw: false,
      awDlg: false, aw: { x: 250, y: 250, enabled: 1, level: 1, troops: '', res: 0, gold: 0, prestige: 0, jewel: '', des: '', officer_id: 0, officer_name: '', treasures: '', capture_rate: 0, max_capture: 1 },
      awOfficerKind: 1, // 活动野地守将类型：0未选 1普通 2名将（按下拉里军官的 kind 推断）
      awStarFilter: 0, // 活动野地守将星级筛选（0=全部）
      awTroops: [], // 活动野地守军可视化行 [{troop_id,count},...]（保存时序列化成 [[tid,count]]）
      awTreasures: [], // 活动野地必掉宝物可视化行 [{treasure_id,count},...]（保存时序列化成 [[cfg_id,count]]）
      jewels: [], // 可采集珠宝下拉（/admin/ezfy-map/options 返回）
      itemCfgs: [], // ★ 2026-10-05 商城道具下拉（/admin/ezfy-map/options 返回）
      treasures: [], // ★ 2026-10-05 野地类型「宝物掉落」下拉：全部装备（武器/防具/饰品/珠宝）
      // 活动野地 · 被打记录模态框（2026-10-01）
      awAttDlg: false, awAttLoading: false, awAttRow: null, awAttList: [], awAttTotal: 0, awAttPage: 1, awAttSize: 10,
      saving: false
    }
  },
  computed: {
    // 弹窗里的「守军搭配」行：直接映射到 wc.wcTroops（模板里要 v-for + splice）
    wcTroops: {
      get () {
        if (!this.wc.wcTroops) this.$set(this.wc, 'wcTroops', [])
        return this.wc.wcTroops
      },
      set (v) { this.$set(this.wc, 'wcTroops', v) }
    },
    // ★ 2026-10-05 「商城道具掉落」行（cfg_id + 数量 + 概率%）
    wcDrops: {
      get () {
        if (!this.wc.wcDrops) this.$set(this.wc, 'wcDrops', [])
        return this.wc.wcDrops
      },
      set (v) { this.$set(this.wc, 'wcDrops', v) }
    },
    // ★ 2026-10-05 「宝物掉落」行（名称 + 数量 + 概率%）
    wcTreasures: {
      get () {
        if (!this.wc.wcTreasures) this.$set(this.wc, 'wcTreasures', [])
        return this.wc.wcTreasures
      },
      set (v) { this.$set(this.wc, 'wcTreasures', v) }
    },
    // 活动野地守将军官下拉：按类型(普通kind=1/名将kind=2)过滤军官池
    awKindGenerals () {
      const k = this.awOfficerKind
      if (k !== 1 && k !== 2) return []
      return (this.generals || []).filter(g => Number(g.kind) === k)
    },
    // 星级筛选可选项：当前类型下出现的不同星级（去重、升序）
    awStarOptions () {
      const set = new Set(this.awKindGenerals.map(g => Number(g.star) || 1))
      return Array.from(set).sort((a, b) => a - b)
    },
    // 守将军官下拉：类型 + 星级双重筛选
    awGeneralsFiltered () {
      let list = this.awKindGenerals
      const s = Number(this.awStarFilter) || 0
      if (s > 0) list = list.filter(g => (Number(g.star) || 1) === s)
      // ★ 已选中的守将即使不满足当前类型/星级筛选也保证可见，否则编辑时下拉里看不到已配军官（星级跟丢）
      const sel = Number(this.aw.officer_id) || 0
      if (sel > 0 && !list.some(g => Number(g.id) === sel)) {
        const hit = (this.generals || []).find(g => Number(g.id) === sel)
        if (hit) list = [hit].concat(list)
      }
      return list
    }
  },
  mounted () { this.loadCities(); this.loadMapOptions(); this.loadTiles(); this.loadActWilds() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    doLookup () {
      api.get('/admin/ezfy-map/lookup', { params: { x: this.lookup.x, y: this.lookup.y } }).then(r => {
        if (r.code === 0) this.lookupResult = r.data
        else this.$message.error(r.msg)
      })
    },
    reload () {
      if (this.tab === 'cities') this.loadCities()
      else if (this.tab === 'wildlands') this.loadWilds()
      else if (this.tab === 'wildcfg') this.loadWildCfgs()
      else if (this.tab === 'occupy') this.loadOccupy()
      else if (this.tab === 'areas') this.loadAreas()
      else if (this.tab === 'stars') this.loadStars()
      else if (this.tab === 'actwild') this.loadActWilds()
    },
    loadCities () {
      this.loadingCity = true
      api.get('/admin/ezfy-map/cities', {
        params: { page: this.cityPage, size: this.citySize, word: this.cityWord }
      }).then(r => {
        this.loadingCity = false
        if (r.code === 0) {
          this.cities = r.data.list
          this.cityTotal = r.data.total
          this.cityPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    // ---- 野地维护 ----
    loadWilds () {
      this.loadingWild = true
      api.get('/admin/ezfy-wildlands', {
        params: { page: this.wildPage, size: this.wildSize, word: this.wildWord,
          type: this.wildType, status: this.wildStatus }
      }).then(r => {
        this.loadingWild = false
        if (r.code === 0) {
          this.wilds = r.data.list || []
          this.wildTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    openWildCreate () {
      this.wf = emptyWild()
      this.wildCfgMatch = null
      this.wildDlg = true
    },
    openWildEdit (row) {
      this.wf = {
        id: row.id, city_id: row.city_id, x: row.x, y: row.y,
        wild_type: row.wild_type, level: row.level, gain: row.gain, status: row.status
      }
      this.wildCfgMatch = row.has_cfg
        ? { type_name: row.type_name, level: row.level, res_min: row.cfg_res_min,
          res_max: row.cfg_res_max, treasure: row.cfg_treasure }
        : null
      this.wildDlg = true
    },
    // 「查看该等级野地配置」：跳到野地类型页并按类型/等级过滤
    openWildCfgOf (row) {
      this.wcType = row.wild_type
      this.wcLevel = row.level
      this.wcPage = 1
      this.tab = 'wildcfg'
      this.loadWildCfgs()
    },
    doWildSave () {
      const body = {}
      WILD_KEYS.forEach(k => { if (this.wf[k] !== null && this.wf[k] !== undefined) body[k] = this.wf[k] })
      this.saving = true
      const req = this.wf.id
        ? api.put('/admin/ezfy-wildlands/' + this.wf.id, body)
        : api.post('/admin/ezfy-wildlands', body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.wildDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadWilds() }
        else this.$message.error(r.msg)
      })
    },
    finishWild (row) {
      api.post('/admin/ezfy-wildlands/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已标记完成'); this.loadWilds() } else this.$message.error(r.msg)
      })
    },
    delWild (row) {
      this.$confirm('删除坐标 (' + row.x + ',' + row.y + ') 的野地记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-wildlands/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadWilds() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 野地类型配置 ----
    loadWildCfgs () {
      this.loadingWc = true
      api.get('/admin/ezfy-wild-cfg', {
        params: { page: this.wcPage, size: this.wcSize, type: this.wcType, level: this.wcLevel }
      }).then(r => {
        this.loadingWc = false
        if (r.code === 0) {
          this.wildCfgs = r.data.list || []
          this.wcTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    // ---- 地图格子覆盖 ----
    loadTiles () {
      this.loadingTile = true
      api.get('/admin/ezfy-map-tiles', {
        params: { page: this.tilePage, size: this.tileSize, word: this.tileWord, mark_kind: this.tileMarkFilter }
      }).then(r => {
        this.loadingTile = false
        if (r.code === 0) {
          this.tiles = r.data.list || []
          this.tileTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    // 查某格：默认规则 vs 生效结果
    loadTileCell () {
      const x = Number(this.tileX)
      const y = Number(this.tileY)
      if (!(x >= 0) || !(y >= 0)) { this.$message.warning('请填写坐标'); return }
      api.get('/admin/ezfy-map-tile', { params: { x: x, y: y } }).then(r => {
        if (r.code !== 0) { this.$message.error(r.msg); return }
        this.tileCell = r.data
        const ov = r.data.override || {}
        this.tileForm = {
          terrain: ov.terrain || 0,
          mark_kind: ov.mark_kind || 0,
          mark_level: ov.mark_level || 1,
          des: ov.des || ''
        }
      })
    },
    saveTile () {
      if (!this.tileCell) { this.$message.warning('请先查询坐标'); return }
      this.saving = true
      api.post('/admin/ezfy-map-tiles', Object.assign({
        x: Number(this.tileCell.x), y: Number(this.tileCell.y)
      }, this.tileForm)).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已保存')
          this.loadTileCell()
          this.loadTiles()
        } else this.$message.error(r.msg)
      })
    },
    clearTile () {
      const ov = (this.tileCell || {}).override || {}
      if (!ov.id) return
      this.$confirm('清除坐标 (' + ov.x + ',' + ov.y + ') 的覆盖，恢复按地图默认规则？', '提示',
        { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-map-tiles/' + ov.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已清除')
            this.loadTileCell()
            this.loadTiles()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-10-05 「管理端删除做好批量删除」：三个列表通用。
    //   ⚠️ 都是**物理删除**（后端模型没有 gorm.DeletedAt，Delete 即真 DELETE 行）。
    batchDelTiles () { this.doBatchDel('tiles') },
    batchDelWilds () { this.doBatchDel('wilds') },
    batchDelAws () { this.doBatchDel('aws') },
    doBatchDel (kind) {
      const map = {
        tiles: { sel: this.tileSel, url: '/admin/ezfy-map-tiles/batch-delete', name: '格子覆盖', reload: () => this.loadTiles() },
        wilds: { sel: this.wildSel, url: '/admin/ezfy-wildlands/batch-delete', name: '野地记录', reload: () => this.loadWilds() },
        aws: { sel: this.awSel, url: '/admin/ezfy-act-wilds/batch-delete', name: '活动野地配置', reload: () => this.loadActWilds() }
      }[kind]
      if (!map || !map.sel.length) return
      this.$confirm('确认**物理删除**勾选的 ' + map.sel.length + ' 条' + map.name + '？删除后不可恢复。',
        '批量删除', { type: 'warning', confirmButtonText: '确定删除' }).then(() => {
        api.post(map.url, { ids: map.sel.map(x => x.id) }).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            map.reload()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delTile (row) {
      this.$confirm('删除坐标 (' + row.x + ',' + row.y + ') 的覆盖？删除后按地图默认规则。', '提示',
        { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-map-tiles/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadTiles() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 活动野地配置（2026-09-29） ----
    loadActWilds () {
      this.loadingAw = true
      api.get('/admin/ezfy-act-wilds', {
        params: { page: this.awPage, size: this.awSize, word: this.awWord, enabled: this.awEnabled }
      }).then(r => {
        this.loadingAw = false
        if (r.code === 0) {
          this.actWilds = r.data.list || []
          this.awTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    openAwCreate () {
      this.aw = { id: 0, x: 250, y: 250, enabled: 1, level: 1, troops: '', res: 0, gold: 0, prestige: 0, jewel: '', des: '', officer_id: 0, officer_name: '', treasures: '', capture_rate: 0, max_capture: 1 }
      this.awOfficerKind = 1 // ★ 默认守将类型 = 普通
      this.awStarFilter = 5  // ★ 默认星级筛选 = 5 星
      // ★ 默认直接选中一个「普通 5 星」守将（有就放进下拉）
      const g = (this.generals || []).find(x => Number(x.kind) === 1 && (Number(x.star) || 1) === 5)
      this.aw.officer_id = g ? Number(g.id) : 0
      this.aw.officer_name = g ? g.name : ''
      this.awTroops = []
      this.awTreasures = []
      this.awDlg = true
    },
    // ★ 2026-10-01 活动野地 · 查看被打记录（模态框展示，分页）
    openAwAttacks (row) {
      this.awAttRow = row
      this.awAttPage = 1
      this.awAttDlg = true
      this.loadAwAttacks()
    },
    loadAwAttacks () {
      if (!this.awAttRow || !this.awAttRow.id) return
      this.awAttLoading = true
      api.get('/admin/ezfy-act-wilds/' + this.awAttRow.id + '/attacks', {
        params: { page: this.awAttPage, size: this.awAttSize }
      }).then(r => {
        this.awAttLoading = false
        if (r.code === 0) {
          this.awAttList = r.data.list || []
          this.awAttTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      }).catch(() => { this.awAttLoading = false })
    },
    openAwEdit (row) {
      this.aw = Object.assign({}, row)
      // 根据已选军官推断类型（能查到该军官 -> 用其 kind）
      const g = (this.generals || []).find(x => Number(x.id) === Number(row.officer_id))
      this.awOfficerKind = g ? (Number(g.kind) === 2 ? 2 : 1) : 1
      // ★ 2026-09-29 修复「编辑活动野地 星级带不过来」：原来每次都重置为 0，
      //   已配守将的星级不再随行进来。现在用列表下发的 officer_star 回填，让星级筛选带上。
      this.awStarFilter = Number(row.officer_star) || 0
      this.awTroops = this.parseAwTroops(row.troops)
      this.awTreasures = this.parseAwTreasures(row.treasures)
      this.awDlg = true
    },
    // 切换普通/名将 时清空已选军官 + 星级筛选（不同类型不能保留旧选择）
    awOfficerKindChange () {
      this.aw.officer_id = 0
      this.aw.officer_name = ''
      this.awStarFilter = 0
    },
    // 解析 [[兵种id,数量],...] JSON → 可视化行
    parseAwTroops (raw) {
      const out = []
      try {
        const arr = JSON.parse(raw || '[]')
        if (Array.isArray(arr)) {
          arr.forEach(r => {
            if (Array.isArray(r) && r.length >= 2 && Number(r[0]) > 0) {
              out.push({ troop_id: Number(r[0]) || 0, count: Number(r[1]) || 0 })
            }
          })
        }
      } catch (e) { /* 历史脏数据忽略 */ }
      return out
    },
    addAwTroop () {
      const first = this.troopCfgs[0]
      if (this.troopCfgs.length) this.awTroops.push({ troop_id: first.id, count: 100 })
    },
    // 解析 [[宝物cfg_id,数量],...] JSON → 可视化行
    parseAwTreasures (raw) {
      const out = []
      try {
        const arr = JSON.parse(raw || '[]')
        if (Array.isArray(arr)) {
          arr.forEach(r => {
            if (Array.isArray(r) && r.length >= 2 && Number(r[0]) > 0) {
              out.push({ treasure_id: Number(r[0]) || 0, count: Number(r[1]) || 1 })
            }
          })
        }
      } catch (e) { /* 历史脏数据忽略 */ }
      return out
    },
    addAwTreasure () {
      const first = this.jewels[0]
      this.awTreasures.push({ treasure_id: first ? first.id : 0, count: 1 })
    },
    saveAw () {
      const a = this.aw
      if (!(Number(a.x) > 0) || !(Number(a.y) > 0)) { this.$message.warning('请填写坐标 x/y'); return }
      // 可视化行 → [[兵种id,数量],...]（没配任何行 = 留空用默认守军）
      const rows = (this.awTroops || []).filter(r => Number(r.troop_id) > 0 && Number(r.count) > 0)
        .map(r => [Number(r.troop_id), Number(r.count)].map(n => Number(n)))
      const troops = rows.length ? JSON.stringify(rows) : ''
      // 宝物可视化行 → [[cfg_id,数量],...]
      const tRows = (this.awTreasures || []).filter(r => Number(r.treasure_id) > 0 && Number(r.count) > 0)
        .map(r => [Number(r.treasure_id), Number(r.count)].map(n => Number(n)))
      const treasures = tRows.length ? JSON.stringify(tRows) : ''
      // ★ 2026-10-05 修复「编辑会新增一个」：把 id 一起带上。
      //   服务端带了 id 就**按 id 更新那一行**（坐标也跟着改，等于「移动这条配置」）；
      //   不带 id 才按 (x,y) upsert —— 原来编辑时改了坐标就会新增一条、旧的还留着。
      const payload = {
        id: Number(a.id) || 0,
        x: Number(a.x), y: Number(a.y),
        enabled: a.enabled === 1 ? 1 : 0,
        level: Number(a.level) || 0,
        troops: troops,
        res: Number(a.res) || 0,
        gold: Number(a.gold) || 0,
        prestige: Number(a.prestige) || 0,
        jewel: a.jewel || '',
        des: a.des || '',
        officer_id: Number(a.officer_id) || 0,
        treasures: treasures,
        capture_rate: Number(a.capture_rate) || 0,
        max_capture: Number(a.max_capture) >= 0 ? Number(a.max_capture) : 1
      }
      this.saving = true
      api.post('/admin/ezfy-act-wilds', payload).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已保存')
          this.awDlg = false
          this.loadActWilds()
        } else this.$message.error(r.msg)
      })
    },
    toggleAw (row) {
      api.post('/admin/ezfy-act-wilds/' + row.id + '/toggle').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已切换'); this.loadActWilds() } else this.$message.error(r.msg)
      })
    },
    delAw (row) {
      this.$confirm('删除坐标 (' + row.x + ',' + row.y + ') 的活动野地配置？删除后该格按默认判定。', '提示',
        { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-act-wilds/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadActWilds() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // 下拉数据（兵种 + 军官池 + 珠宝）
    loadMapOptions () {
      api.get('/admin/ezfy-map/options').then(r => {
        if (r.code === 0) {
          this.troopCfgs = r.data.troops || []
          this.generals = r.data.generals || []
          this.jewels = r.data.jewels || []
          this.itemCfgs = r.data.items || [] // ★ 2026-10-05 商城道具下拉
          this.treasures = r.data.treasures || [] // ★ 2026-10-05 宝物掉落下拉（全部装备）
        }
      })
    },
    // 活动野地列表：把守军 [[兵种id,数量]] 显示成「步兵 100、卡车 50」这种好读形式
    awTroopText (raw) {
      const rows = this.parseAwTroops(raw)
      if (!rows.length) return ''
      return rows.map(r => {
        const t = (this.troopCfgs || []).find(x => Number(x.id) === Number(r.troop_id))
        return (t ? t.name : ('兵种' + r.troop_id)) + ' ' + Number(r.count).toLocaleString()
      }).join('、')
    },
    // 活动野地列表：把必掉宝物 [[cfg_id,数量]] 显示成「红宝石 x2、蓝宝石 x1」
    awTreasureText (raw) {
      const rows = this.parseAwTreasures(raw)
      if (!rows.length) return ''
      return rows.map(r => {
        const j = (this.jewels || []).find(x => Number(x.id) === Number(r.treasure_id))
        return (j ? j.name : ('宝物' + r.treasure_id)) + ' x' + Number(r.count).toLocaleString()
      }).join('、')
    },
    // 把 troops JSON 串 [[tid,min,max],...] 解析成可视化行
    parseWcTroops (raw) {
      const out = []
      try {
        const arr = JSON.parse(raw || '[]')
        if (Array.isArray(arr)) {
          arr.forEach(r => {
            if (Array.isArray(r) && r.length >= 3) {
              out.push({ troop_id: Number(r[0]) || 0, min: Number(r[1]) || 0, max: Number(r[2]) || 0 })
            }
          })
        }
      } catch (e) { /* 历史脏数据忽略 */ }
      return out
    },
    // 列表里把守军显示成「步兵 100~200、卡车 50」这种好读的形式
    troopText (raw) {
      const rows = this.parseWcTroops(raw)
      if (!rows.length) return ''
      const nameOf = id => {
        const t = this.troopCfgs.find(x => x.id === id)
        return t ? t.name : ('兵种' + id)
      }
      return rows.map(r => nameOf(r.troop_id) + ' ' + r.min + '~' + r.max).join('、')
    },
    addWcTroop () {
      if (!this.wc.wcTroops) this.$set(this.wc, 'wcTroops', [])
      const first = this.troopCfgs[0]
      this.wc.wcTroops.push({ troop_id: first ? first.id : 0, min: 100, max: 200 })
    },
    // ★ 2026-10-05 商城道具掉落行：新增一行（默认概率 100%）
    addWcDrop () {
      if (!this.wc.wcDrops) this.$set(this.wc, 'wcDrops', [])
      this.wc.wcDrops.push({ cfg_id: (this.itemCfgs[0] || {}).id || 0, count: 1, pct: 100 })
    },
    // ★ 2026-10-05 宝物掉落行：新增一行（默认概率 100%）
    addWcTreasure () {
      if (!this.wc.wcTreasures) this.$set(this.wc, 'wcTreasures', [])
      this.wc.wcTreasures.push({ name: (this.treasures[0] || {}).name || '', count: 1, pct: 100 })
    },
    openWcCreate () {
      this.wc = emptyWc()
      this.wcDlg = true
    },
    openWcEdit (row) {
      this.wc = Object.assign(emptyWc(), row)
      this.$set(this.wc, 'wcTroops', this.parseWcTroops(row.troops))
      // ★ 2026-10-05 反序列化「商城道具掉落」[[id,count,pct]...] → 可视化行
      this.$set(this.wc, 'wcDrops', this.parseWcDrops(row.drop_items))
      // ★ 2026-10-05 反序列化「宝物掉落」[{name,count,pct}...] → 可视化行（老文本值兜底成一行）
      this.$set(this.wc, 'wcTreasures', this.parseWcTreasures(row.treasure))
      this.wcDlg = true
    },
    // ★ 2026-10-05 [[道具id,数量,概率%]...] → [{cfg_id,count,pct}]；空/非法 → []
    parseWcDrops (raw) {
      if (!raw) return []
      try {
        const rows = JSON.parse(raw)
        if (!Array.isArray(rows)) return []
        return rows.filter(r => Array.isArray(r) && r[0] > 0)
          .map(r => ({ cfg_id: Number(r[0]), count: Number(r[1]) || 1, pct: Number(r[2]) > 0 ? Number(r[2]) : 100 }))
      } catch (e) { return [] }
    },
    // ★ 2026-10-05 [{name,count,pct}...] → 可视化行；老文本（如「珠宝(平原)」）兜底成一行
    parseWcTreasures (raw) {
      if (!raw) return []
      try {
        const rows = JSON.parse(raw)
        if (Array.isArray(rows)) {
          return rows.filter(r => r && r.name)
            .map(r => ({ name: r.name, count: Number(r.count) || 1, pct: Number(r.pct) > 0 ? Number(r.pct) : 100 }))
        }
        return []
      } catch (e) {
        return [{ name: raw, count: 1, pct: 100 }]
      }
    },
    doWcSave () {
      // ★ 把可视化行序列化回后端要的 [[兵种ID,最小,最大],...]
      const rows = (this.wc.wcTroops || [])
        .filter(r => r.troop_id > 0)
        .map(r => {
          const lo = Number(r.min) || 0
          const hi = Math.max(lo, Number(r.max) || 0)
          return [Number(r.troop_id), lo, hi]
        })
      this.wc.troops = JSON.stringify(rows)
      // ★ 2026-10-05 序列化「商城道具掉落」[[id,count,pct]...]
      this.wc.drop_items = JSON.stringify((this.wc.wcDrops || [])
        .filter(d => d.cfg_id > 0)
        .map(d => [Number(d.cfg_id), Number(d.count) || 1, Number(d.pct) > 0 ? Number(d.pct) : 100]))
      // ★ 2026-10-05 序列化「宝物掉落」[{name,count,pct}...]
      this.wc.treasure = JSON.stringify((this.wc.wcTreasures || [])
        .filter(t => t.name)
        .map(t => ({ name: t.name, count: Number(t.count) || 1, pct: Number(t.pct) > 0 ? Number(t.pct) : 100 })))
      const body = {}
      WC_KEYS.forEach(k => { if (this.wc[k] !== null && this.wc[k] !== undefined) body[k] = this.wc[k] })
      this.saving = true
      const req = this.wc.id
        ? api.put('/admin/ezfy-wild-cfg/' + this.wc.id, body)
        : api.post('/admin/ezfy-wild-cfg', body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.wcDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadWildCfgs() }
        else this.$message.error(r.msg)
      })
    },
    delWc (row) {
      this.$confirm('删除「' + row.type_name + ' Lv.' + row.level + '」的野地类型配置？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-wild-cfg/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadWildCfgs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 占领记录 ----
    loadOccupy () {
      this.loadingOcc = true
      api.get('/admin/ezfy-map/occupy', {
        params: { page: this.occPage, size: this.occSize, status: this.occStatus }
      }).then(r => {
        this.loadingOcc = false
        if (r.code === 0) {
          this.occupies = r.data.list
          this.occTotal = r.data.total
          this.occPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    // ---- 地图区域 ----
    loadAreas () {
      this.loadingArea = true
      api.get('/admin/ezfy-map/areas', {
        params: { page: this.areaPage, size: this.areaSize, area_type: this.areaType }
      }).then(r => {
        this.loadingArea = false
        if (r.code === 0) {
          this.areas = r.data.list
          this.areaTotal = r.data.total
          this.areaPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    // ---- 坐标收藏 ----
    loadStars () {
      this.loadingStar = true
      api.get('/admin/ezfy-map/stars', { params: { page: this.starPage, size: this.starSize } }).then(r => {
        this.loadingStar = false
        if (r.code === 0) {
          this.stars = r.data.list
          this.starTotal = r.data.total
          this.starPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    release (row) {
      this.$confirm('解除后「' + row.city_name + '」将归还给原属玩家，确认？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-map/occupy/' + row.id + '/release').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已解除'); this.loadOccupy() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delOccupy (row) {
      this.$confirm('删除该占领记录（仅删记录，不改变城池归属），确认？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-map/occupy/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadOccupy() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delArea (row) {
      this.$confirm('删除坐标 (' + row.x + ',' + row.y + ') 的地图区域记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-map/areas/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadAreas() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delStar (row) {
      api.delete('/admin/ezfy-map/stars/' + row.id).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadStars() } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
/* 野地类型弹窗的「守军搭配」行 */
.wild-troop-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; flex-wrap: wrap; }
@import './farm-admin.css';
</style>
