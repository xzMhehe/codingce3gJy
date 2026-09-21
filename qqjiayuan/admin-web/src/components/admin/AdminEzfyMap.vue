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
        <el-button type="primary" plain icon="el-icon-refresh" @click="reload">刷新</el-button>
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
            <el-table-column prop="player_name" label="归属玩家" min-width="125" show-overflow-tooltip />
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
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadTiles">刷新</el-button>
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

          <el-table :data="tiles" v-loading="loadingTile" stripe border max-height="480">
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
            <el-table-column prop="des" label="备注" min-width="160" show-overflow-tooltip />
            <el-table-column label="更新时间" width="160" align="center">
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
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadWilds">刷新</el-button>
          </div>
          <el-table :data="wilds" v-loading="loadingWild" stripe border>
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
            <el-table-column prop="city_name" label="占领城池" min-width="115" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" min-width="115" show-overflow-tooltip />
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
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadWildCfgs">刷新</el-button>
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
            <el-table-column prop="city_name" label="被占城池" min-width="130" show-overflow-tooltip />
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="atk_name" label="占领方" min-width="125" show-overflow-tooltip />
            <el-table-column prop="def_name" label="原属方" min-width="125" show-overflow-tooltip />
            <el-table-column label="状态" width="100" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 1 ? 'danger' : 'success'">{{ row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="时间" width="150" align="center">
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
            <el-table-column prop="troops" label="守军" min-width="160" show-overflow-tooltip />
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
            <el-table-column prop="owner_name" label="玩家" min-width="125" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="100" align="center" />
            <el-table-column label="坐标" width="105" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.x }},{{ row.y }}</span></template>
            </el-table-column>
            <el-table-column prop="terrain_name" label="地形" width="85" align="center" />
            <el-table-column prop="name" label="备注名" min-width="140" show-overflow-tooltip />
            <el-table-column label="收藏时间" width="150" align="center">
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
        <el-form-item label="宝物">
          <el-input v-model="wc.treasure" maxlength="100" placeholder="可空，例如：珠宝(平原)" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="wc.des" type="textarea" :rows="2" maxlength="500" show-word-limit />
        </el-form-item>
      </el-form>
      <em>保存后立即生效（后端会重载配置缓存，不用重启）</em>
      <div slot="footer">
        <el-button @click="wcDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doWcSave">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const WILD_KEYS = ['city_id', 'x', 'y', 'wild_type', 'level', 'gain', 'status']
const WC_KEYS = ['type', 'level', 'troops', 'res_min', 'res_max',
  'officer_min', 'officer_max', 'officer_id', 'treasure', 'des']

function emptyWild () {
  return { id: 0, city_id: 1, x: 250, y: 250, wild_type: 1, level: 1, gain: '', status: 0 }
}
function emptyWc () {
  return { id: 0, type: 1, level: 1, troops: '', res_min: 0, res_max: 0,
    officer_min: 0, officer_max: 0, officer_id: 0, treasure: '', des: '',
    // 弹窗内的「守军搭配」行（保存时序列化进 troops）
    wcTroops: [] }
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
      saving: false
    }
  },
  computed: {
    // 弹窗里的「守军搭配」行：直接映射到 wc.wcTroops（模板里要 v-for + splice）
    wcTroops () {
      if (!this.wc.wcTroops) this.$set(this.wc, 'wcTroops', [])
      return this.wc.wcTroops
    }
  },
  mounted () { this.loadCities(); this.loadMapOptions(); this.loadTiles() },
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
    delTile (row) {
      this.$confirm('删除坐标 (' + row.x + ',' + row.y + ') 的覆盖？删除后按地图默认规则。', '提示',
        { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-map-tiles/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadTiles() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // 下拉数据（兵种 + 军官池）
    loadMapOptions () {
      api.get('/admin/ezfy-map/options').then(r => {
        if (r.code === 0) {
          this.troopCfgs = r.data.troops || []
          this.generals = r.data.generals || []
        }
      })
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
    openWcCreate () {
      this.wc = emptyWc()
      this.wcDlg = true
    },
    openWcEdit (row) {
      this.wc = Object.assign(emptyWc(), row)
      this.$set(this.wc, 'wcTroops', this.parseWcTroops(row.troops))
      this.wcDlg = true
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
