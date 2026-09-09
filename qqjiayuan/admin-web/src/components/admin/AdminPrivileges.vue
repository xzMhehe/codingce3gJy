<template>
  <div>
    <!-- 概览统计 -->
    <div class="priv-stats">
      <div class="pstat">
        <div class="pico pico-blue"><i class="el-icon-star-off"></i></div>
        <div class="pnum">
          <div class="n"><b>{{ stats.blue_active }}</b><span class="fade">/ {{ stats.blue_total }} 名</span></div>
          <div class="l">蓝钻会员（生效/累计）</div>
        </div>
      </div>
      <div class="pstat">
        <div class="pico pico-qq"><i class="el-icon-star-on"></i></div>
        <div class="pnum">
          <div class="n"><b>{{ stats.qq_active }}</b><span class="fade">/ {{ stats.qq_total }} 名</span></div>
          <div class="l">超Q会员（生效/累计）</div>
        </div>
      </div>
      <div class="pstat">
        <div class="pico pico-plan"><i class="el-icon-shopping-cart-2"></i></div>
        <div class="pnum">
          <div class="n"><b>{{ stats.plan_count }}</b><span class="fade">个</span></div>
          <div class="l">贵宾销售方案</div>
        </div>
      </div>
      <div class="pstat">
        <div class="pico pico-exp"><i class="el-icon-warning-outline"></i></div>
        <div class="pnum">
          <div class="n"><b>{{ stats.expired }}</b><span class="fade">名</span></div>
          <div class="l">已到期（保留期30天）</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar tabbar">
        <el-tabs v-model="tab" class="ptabs">
          <el-tab-pane label="贵宾等级" name="levels" />
          <el-tab-pane label="贵宾会员" name="users" />
          <el-tab-pane label="贵宾销售" name="plans" />
        </el-tabs>
        <div class="grow" />
        <template v-if="tab === 'users'">
          <el-input v-model="kw" placeholder="号码 / 昵称搜索" prefix-icon="el-icon-search" clearable
                    style="width:200px" @keyup.enter.native="load" @clear="load" />
          <el-button type="success" plain size="small" @click="batch('blue')"><i class="el-icon-star-off"></i> 一键开蓝钻</el-button>
          <el-button type="warning" plain size="small" @click="batch('qq')"><i class="el-icon-star-on"></i> 一键开超Q</el-button>
          <el-button type="danger" plain size="small" @click="cleanExpired"><i class="el-icon-key"></i> 清除到期</el-button>
        </template>
        <el-button v-if="tab === 'levels'" type="primary" size="small" icon="el-icon-plus" @click="openLevel(null)">新增等级</el-button>
        <el-button v-if="tab === 'plans'" type="primary" size="small" icon="el-icon-plus" @click="openPlan(null)">新增方案</el-button>
        <el-button size="small" circle icon="el-icon-refresh" title="刷新" @click="refresh" />
      </div>

      <!-- ===== 贵宾等级（复刻诺哈 wap_vip_config：升级经验+图标） ===== -->
      <div v-if="tab === 'levels'" class="hint">成长值达到「升级经验」即自动升至对应等级，图标取自 static/picture 素材（1级为入门贵宾）。</div>
      <el-table v-if="tab === 'levels'" :data="levels" v-loading="loading" stripe>
        <el-table-column label="等级" width="90" align="center">
          <template slot-scope="{row}"><span class="lv-badge">Lv.{{ row.id }}</span></template>
        </el-table-column>
        <el-table-column label="升级经验" width="120" align="center">
          <template slot-scope="{row}"><b class="num">{{ row.point }}</b><span class="unit">点</span></template>
        </el-table-column>
        <el-table-column label="蓝钻图标" min-width="155">
          <template slot-scope="{row}">
            <span class="icon-cell">
              <span class="priv-chip chip-blue">
                <img v-show="!imgBroken('lb' + row.id)" class="priv-img" :src="$pic(row.icon_blue)" :alt="'蓝钻Lv.' + row.id" @error="markBroken('lb' + row.id)">
                <i v-show="imgBroken('lb' + row.id)" class="el-icon-star-off chip-ico blue"></i>
              </span>
              <span class="icon-file txt-fade">{{ row.icon_blue || '未设置' }}</span>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="超Q图标" min-width="155">
          <template slot-scope="{row}">
            <span class="icon-cell">
              <span class="priv-chip chip-qq">
                <img v-show="!imgBroken('lq' + row.id)" class="priv-img" :src="$pic(row.icon_qq)" :alt="'超QLv.' + row.id" @error="markBroken('lq' + row.id)">
                <i v-show="imgBroken('lq' + row.id)" class="el-icon-star-on chip-ico qq"></i>
              </span>
              <span class="icon-file txt-fade">{{ row.icon_qq || '未设置' }}</span>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right" header-align="center" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openLevel(row)">编辑</el-button>
            <el-button size="mini" type="danger" plain :disabled="row.id <= 1" @click="delLevel(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- ===== 贵宾会员（复刻诺哈 wap_vip：等级/成长值/速度/成长时间/速度时间/开通时间/结束时间） ===== -->
      <template v-else-if="tab === 'users'">
        <div class="hint">仅展示已开通贵宾的会员；到期保留30天，期间可续费，超期自动清空（也可用「清除到期」立即清理）。</div>
        <el-table :data="users" v-loading="loading" stripe>
          <el-table-column prop="id" label="号码" width="88" fixed="left" />
          <el-table-column label="昵称" min-width="150" show-overflow-tooltip>
            <template slot-scope="{row}"><font :color="row.color || '#333'">{{ nameOf(row) }}</font></template>
          </el-table-column>
          <el-table-column label="蓝钻" width="185">
            <template slot-scope="{row}">
              <div class="priv-cell">
                <span class="priv-chip chip-blue">
                  <img v-show="!imgBroken('ub' + row.id)" class="priv-img" :src="$pic(row.blue_icon)" alt="蓝钻" @error="markBroken('ub' + row.id)">
                  <i v-show="imgBroken('ub' + row.id)" class="el-icon-star-off chip-ico blue"></i>
                </span>
                <span class="lv-tag" :class="isOn(row.blue_end) ? 'on' : 'off'">Lv.{{ row.blue_lv }}</span>
                <span class="txt-fade">{{ row.blue_exp }}点</span>
              </div>
              <div class="priv-sub">
                <span v-if="isOn(row.blue_end)">{{ row.blue_speed || '—' }}点/天 · 剩{{ left(row.blue_end) }}天</span>
                <span v-else class="txt-fade">已到期</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="超Q" width="185">
            <template slot-scope="{row}">
              <div class="priv-cell">
                <span class="priv-chip chip-qq">
                  <img v-show="!imgBroken('uq' + row.id)" class="priv-img" :src="$pic(row.qq_icon)" alt="超Q" @error="markBroken('uq' + row.id)">
                  <i v-show="imgBroken('uq' + row.id)" class="el-icon-star-on chip-ico qq"></i>
                </span>
                <span class="lv-tag" :class="isOn(row.qq_end) ? 'on' : 'off'">Lv.{{ row.qq_lv }}</span>
                <span class="txt-fade">{{ row.qq_exp }}点</span>
              </div>
              <div class="priv-sub">
                <span v-if="isOn(row.qq_end)">{{ row.qq_speed || '—' }}点/天 · 剩{{ left(row.qq_end) }}天</span>
                <span v-else class="txt-fade">已到期</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="104" align="left">
            <template slot-scope="{row}">
              <div class="st-tags">
                <el-tag v-if="isOn(row.blue_end)" type="success" size="mini" class="st">蓝钻生效</el-tag>
                <el-tag v-else-if="row.blue_exp > 0" type="info" size="mini" class="st">蓝钻保留</el-tag>
                <el-tag v-if="isOn(row.qq_end)" type="warning" size="mini" class="st">超Q生效</el-tag>
                <el-tag v-else-if="row.qq_exp > 0" type="info" size="mini" class="st">超Q保留</el-tag>
                <span v-if="!isOn(row.blue_end) && !isOn(row.qq_end) && row.blue_exp <= 0 && row.qq_exp <= 0" class="txt-fade">已到期</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="开通 / 到期" min-width="170">
            <template slot-scope="{row}">
              <div class="dt-line"><i class="el-icon-star-off dt-ico blue"></i>蓝 {{ short(row.blue_start) }} ~ {{ short(row.blue_end) }}</div>
              <div class="dt-line"><i class="el-icon-star-on dt-ico qq"></i>超 {{ short(row.qq_start) }} ~ {{ short(row.qq_end) }}</div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="252" fixed="right" header-align="center" align="center">
            <template slot-scope="{row}">
              <div class="ops">
                <el-button size="mini" type="primary" plain @click="openUser(row)">编辑</el-button>
                <el-button size="mini" type="success" plain @click="openOne(row, 'blue')">开蓝钻</el-button>
                <el-button size="mini" type="warning" plain @click="openOne(row, 'qq')">开超Q</el-button>
                <el-button size="mini" type="danger" plain @click="closeDlgOpen(row)">取消</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ total }}</b> 名贵宾 · 每页 {{ pageSize }} 条</div>
          <el-pagination
            small background layout="sizes, prev, pager, next, jumper"
            :total="total" :page-size.sync="pageSize" :current-page.sync="page"
            :page-sizes="[10, 20, 50, 100]"
            @size-change="page = 1; load()" @current-change="load"
          />
        </div>
      </template>

      <!-- ===== 贵宾销售（复刻诺哈 wap_vip_shop：标题/几月/币种/价格/速度/赠送/限购/库存/销量/时间/状态） ===== -->
      <template v-else>
        <div class="hint">「销售几月」1月=30天，年费=12月；出售/结束时间留空表示长期有效；成长速度与赠送经验为开通时赋予会员的数值。</div>
        <el-table :data="plans" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="50" align="center" />
        <el-table-column label="类型" width="70" align="center">
          <template slot-scope="{row}"><el-tag :type="row.type === 'blue' ? 'primary' : 'warning'" size="mini">{{ row.type === 'blue' ? '蓝钻' : '超Q' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="销售标题" min-width="180" show-overflow-tooltip>
          <template slot-scope="{row}"><b>{{ row.name }}</b></template>
        </el-table-column>
        <el-table-column label="销售几月" width="76" align="center">
          <template slot-scope="{row}">{{ months(row.days) }}<span class="unit">月</span></template>
        </el-table-column>
        <el-table-column label="币种" width="56" align="center">
          <template slot-scope="{row}">{{ row.money === 1 ? 'G币' : '—' }}</template>
        </el-table-column>
        <el-table-column label="销售价格" width="86" align="center">
          <template slot-scope="{row}"><b class="cost">{{ row.cost }}</b><span class="unit">币/月</span></template>
        </el-table-column>
        <el-table-column label="成长速度" width="86" align="center">
          <template slot-scope="{row}">{{ row.speed }}<span class="unit">点/天</span></template>
        </el-table-column>
        <el-table-column label="赠送经验" width="80" align="center">
          <template slot-scope="{row}">{{ row.gain }}<span class="unit">点</span></template>
        </el-table-column>
        <el-table-column label="限购" width="60" align="center">
          <template slot-scope="{row}">{{ row.limit || '不限' }}</template>
        </el-table-column>
        <el-table-column label="库存" width="60" align="center">
          <template slot-scope="{row}">{{ row.stock || '不限' }}</template>
        </el-table-column>
        <el-table-column label="销量" width="60" align="center">
          <template slot-scope="{row}">{{ row.sales }}</template>
        </el-table-column>
        <el-table-column label="出售 / 结束" min-width="140">
          <template slot-scope="{row}">
            <span class="txt-fade">{{ short(row.stime) }} ~ {{ short(row.etime) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="60" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag>
          </template>
        </el-table-column>
          <el-table-column label="操作" width="120" fixed="right" header-align="center" align="center">
            <template slot-scope="{row}">
              <el-button size="mini" type="primary" plain @click="openPlan(row)">编辑</el-button>
              <el-button size="mini" type="danger" plain @click="delPlan(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-card>

    <!-- 等级编辑 -->
    <el-dialog :title="levelForm.id ? '编辑贵宾等级 Lv.' + levelForm.id : '新增贵宾等级'" :visible.sync="levelDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="96px">
        <el-form-item label="等级">Lv.{{ levelForm.id || '新增自动编号' }}</el-form-item>
        <el-form-item label="升级经验">
          <el-input-number v-model="levelForm.point" :min="0" :max="99999999" />
          <span class="hint-inline">成长值达此门槛即升级</span>
        </el-form-item>
        <el-form-item label="蓝钻图标">
          <div class="icon-row">
            <el-input v-model.trim="levelForm.icon_blue" maxlength="50" />
            <span class="priv-chip chip-blue">
              <img v-show="!imgBroken('dlb')" class="priv-img" :src="$pic(levelForm.icon_blue)" alt="蓝钻图标" @error="markBroken('dlb')">
              <i v-show="imgBroken('dlb')" class="el-icon-star-off chip-ico blue"></i>
            </span>
          </div>
        </el-form-item>
        <el-form-item label="超Q图标">
          <div class="icon-row">
            <el-input v-model.trim="levelForm.icon_qq" maxlength="50" />
            <span class="priv-chip chip-qq">
              <img v-show="!imgBroken('dlq')" class="priv-img" :src="$pic(levelForm.icon_qq)" alt="超Q图标" @error="markBroken('dlq')">
              <i v-show="imgBroken('dlq')" class="el-icon-star-on chip-ico qq"></i>
            </span>
          </div>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="levelDlg = false">取 消</el-button>
        <el-button type="primary" @click="saveLevel">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 贵宾会员编辑（复刻诺哈 user_edit：等级/成长值/速度/成长时间/速度时间/开通时间/结束时间） -->
    <el-dialog :title="'编辑贵宾：' + (userForm.nickname || '')" :visible.sync="userDlg" width="680px" :close-on-click-modal="false">
      <div class="dlg-head">号码 <b>{{ userForm.id }}</b> · 昵称 <b>{{ userForm.nickname || '（未命名）' }}</b></div>
      <el-form label-width="96px">
        <div class="group-hd">蓝钻</div>
        <div class="grp">
          <el-form-item label="等级" class="inline">
            <el-select v-model="userForm.blue.lv" style="width:130px">
              <el-option v-for="l in maxLv" :key="l" :label="l === 0 ? '取消（Lv.0）' : 'Lv.' + l" :value="l" />
            </el-select>
          </el-form-item>
          <el-form-item label="成长值" class="inline"><el-input-number v-model="userForm.blue.exp" :min="0" :max="99999999" /></el-form-item>
          <el-form-item label="成长速度" class="inline"><el-input-number v-model="userForm.blue.speed" :min="0" /> <span class="hint-inline">点/天</span></el-form-item>
          <el-form-item label="成长时间" class="inline">
            <el-date-picker v-model="userForm.blue.ptime" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="上次成长时间" style="width:200px" />
          </el-form-item>
          <el-form-item label="开通时间" class="inline">
            <el-date-picker v-model="userForm.blue.start" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" style="width:200px" />
          </el-form-item>
          <el-form-item label="结束时间" class="inline">
            <el-date-picker v-model="userForm.blue.end" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" style="width:200px" />
          </el-form-item>
        </div>
        <div class="group-hd">超Q</div>
        <div class="grp">
          <el-form-item label="等级" class="inline">
            <el-select v-model="userForm.qq.lv" style="width:130px">
              <el-option v-for="l in maxLv" :key="l" :label="l === 0 ? '取消（Lv.0）' : 'Lv.' + l" :value="l" />
            </el-select>
          </el-form-item>
          <el-form-item label="成长值" class="inline"><el-input-number v-model="userForm.qq.exp" :min="0" :max="99999999" /></el-form-item>
          <el-form-item label="成长速度" class="inline"><el-input-number v-model="userForm.qq.speed" :min="0" /> <span class="hint-inline">点/天</span></el-form-item>
          <el-form-item label="成长时间" class="inline">
            <el-date-picker v-model="userForm.qq.ptime" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="上次成长时间" style="width:200px" />
          </el-form-item>
          <el-form-item label="开通时间" class="inline">
            <el-date-picker v-model="userForm.qq.start" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" style="width:200px" />
          </el-form-item>
          <el-form-item label="结束时间" class="inline">
            <el-date-picker v-model="userForm.qq.end" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" style="width:200px" />
          </el-form-item>
        </div>
        <div class="dlg-tip">等级为0且成长值0时表示取消该贵宾；成长速度0表示按方案默认（10/15点每天）。</div>
      </el-form>
      <div slot="footer">
        <el-button @click="userDlg = false">取 消</el-button>
        <el-button type="primary" @click="saveUser">保存全部修改</el-button>
      </div>
    </el-dialog>

    <!-- 方案编辑（复刻诺哈 shop_edit） -->
    <el-dialog :title="planForm.id ? '编辑销售：' + planForm.name : '新增销售'" :visible.sync="planDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="96px">
        <el-form-item label="类型">
          <el-radio-group v-model="planForm.type">
            <el-radio label="blue">蓝钻</el-radio>
            <el-radio label="qq">超Q</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="销售标题"><el-input v-model.trim="planForm.name" maxlength="40" /></el-form-item>
        <el-form-item label="销售几月"><el-input-number v-model="planForm.months" :min="1" :max="24" /> <span class="hint-inline">1月=30天，12=年费</span></el-form-item>
        <el-form-item label="销售币种"><el-radio-group v-model="planForm.money"><el-radio :label="1">G币</el-radio></el-radio-group></el-form-item>
        <el-form-item label="销售价格"><el-input-number v-model="planForm.cost" :min="0" /> <span class="hint-inline">G币/月</span></el-form-item>
        <el-form-item label="成长速度"><el-input-number v-model="planForm.speed" :min="0" /> <span class="hint-inline">点/天</span></el-form-item>
        <el-form-item label="赠送经验"><el-input-number v-model="planForm.gain" :min="0" /> <span class="hint-inline">开通即得</span></el-form-item>
        <el-form-item label="每号限购"><el-input-number v-model="planForm.limit" :min="0" /> <span class="hint-inline">0=不限</span></el-form-item>
        <el-form-item label="库存数量"><el-input-number v-model="planForm.stock" :min="0" /> <span class="hint-inline">0=不限</span></el-form-item>
        <el-form-item label="出售时间">
          <el-date-picker v-model="planForm.stime" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="留空=立即" style="width:220px" />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker v-model="planForm.etime" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="留空=长期" style="width:220px" />
        </el-form-item>
        <el-form-item label="销售状态">
          <el-switch v-model="planForm.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="下架" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="planDlg = false">取 消</el-button>
        <el-button type="primary" @click="savePlan">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 取消贵宾确认 -->
    <el-dialog title="取消贵宾" :visible.sync="closeDlg" width="360px" :close-on-click-modal="false">
      <div class="dlg-head">确定取消 <b>{{ closeUser && closeUser.nickname }}</b> 的贵宾吗？</div>
      <el-form label-width="90px">
        <el-form-item label="取消类型">
          <el-radio-group v-model="closeType">
            <el-radio label="blue">蓝钻</el-radio>
            <el-radio label="qq">超Q</el-radio>
            <el-radio label="all">全部</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="closeDlg = false">取 消</el-button>
        <el-button type="danger" @click="doClose">确定取消</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminPrivileges',
  data () {
    return {
      tab: 'levels',
      loading: false,
      stats: { blue_active: 0, blue_total: 0, qq_active: 0, qq_total: 0, plan_count: 0, expired: 0 },
      // 贵宾等级
      levels: [], levelDlg: false, levelEdit: false,
      levelForm: { id: 0, point: 0, icon_blue: '', icon_qq: '' },
      // 图标加载失败（源文件缺失/路径有误）时显示占位符，避免破图
      broken: {},
      // 贵宾会员
      users: [], page: 1, pageSize: 10, total: 0, kw: '', userDlg: false,
      closeDlg: false, closeUser: null, closeType: 'blue',
      userForm: { id: 0, nickname: '', blue: { lv: 0, exp: 0, speed: 0, ptime: '', start: '', end: '' }, qq: { lv: 0, exp: 0, speed: 0, ptime: '', start: '', end: '' } },
      // 贵宾销售
      plans: [], planDlg: false,
      planForm: { id: 0, type: 'blue', name: '', months: 1, money: 1, cost: 0, speed: 0, gain: 0, limit: 0, stock: 0, stime: '', etime: '', status: 1 }
    }
  },
  computed: {
    maxLv () { return Array.from({ length: Math.max(9, this.levels.length + 1) }, (_, i) => i) }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      this.loadStats()
      if (this.tab === 'levels') {
        api.get('/admin/privileges/levels').then(r => { this.loading = false; if (r.code === 0) this.levels = r.data || [] })
        return
      }
      if (this.tab === 'users') {
        api.get('/admin/privileges/users', { params: { page: this.page, size: this.pageSize, kw: this.kw } }).then(r => {
          this.loading = false
          if (r.code === 0) { this.users = r.data.list; this.total = r.data.total }
        })
        return
      }
      api.get('/admin/privileges/plans').then(r => { this.loading = false; if (r.code === 0) this.plans = r.data || [] })
    },
    refresh () { this.load() },
    loadStats () {
      api.get('/admin/privileges/stats').then(r => { if (r.code === 0) this.stats = r.data })
    },
    // 昵称为空/空白时展示兜底，不出现空白格
    nameOf (row) {
      const n = (row.nickname || '').trim()
      return n || ('（未命名 ' + row.id + '）')
    },
    // 图标加载失败标记与判断（占位符兜底）
    markBroken (key) { this.$set(this.broken, key, true) },
    imgBroken (key) { return !!this.broken[key] },
    isOn (end) { return !!end && new Date(end).getTime() > Date.now() },
    left (end) {
      if (!end) return 0
      return Math.max(0, Math.ceil((new Date(end).getTime() - Date.now()) / 86400000))
    },
    short (t) {
      if (!t) return '—'
      const d = new Date(t)
      const p = v => (v < 10 ? '0' : '') + v
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate())
    },
    // ===== 等级 =====
    openLevel (row) {
      this.levelEdit = !!row
      this.levelForm = row ? { id: row.id, point: row.point, icon_blue: row.icon_blue, icon_qq: row.icon_qq } : { id: 0, point: 0, icon_blue: '', icon_qq: '' }
      this.levelDlg = true
    },
    saveLevel () {
      const body = { point: this.levelForm.point, icon_blue: this.levelForm.icon_blue, icon_qq: this.levelForm.icon_qq }
      if (!this.levelForm.icon_blue && !this.levelForm.icon_qq) { this.$message.warning('请填写等级图标'); return }
      if (this.levelEdit) {
        api.put('/admin/privileges/levels/' + this.levelForm.id, body).then(r => { if (r.code === 0) { this.levelDlg = false; this.load() } else this.$message.error(r.msg) })
      } else {
        api.post('/admin/privileges/levels', body).then(r => { if (r.code === 0) { this.levelDlg = false; this.load() } else this.$message.error(r.msg) })
      }
    },
    delLevel (row) {
      this.$confirm('确定删除 Lv.' + row.id + ' 贵宾等级吗？已有会员等级不变。', '提示').then(() => {
        api.delete('/admin/privileges/levels/' + row.id).then(r => { if (r.code === 0) this.load() })
      }).catch(() => {})
    },
    // ===== 会员 =====
    openUser (row) {
      this.userForm = {
        id: row.id, nickname: this.nameOf(row),
        blue: { lv: row.blue_lv, exp: row.blue_exp, speed: row.blue_speed, ptime: ts(row.blue_ptime), start: ts(row.blue_start), end: ts(row.blue_end) },
        qq: { lv: row.qq_lv, exp: row.qq_exp, speed: row.qq_speed, ptime: ts(row.qq_ptime), start: ts(row.qq_start), end: ts(row.qq_end) }
      }
      this.userDlg = true
    },
    saveUser () {
      const u = this.userForm
      const item = s => ({ lv: u[s].lv, exp: u[s].exp, speed: u[s].speed, ptime: u[s].ptime, start: u[s].start, end: u[s].end })
      api.put('/admin/privileges/users/' + u.id, { blue: item('blue'), qq: item('qq') }).then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.userDlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    openOne (row, type) {
      this.$confirm('确定给用户「' + this.nameOf(row) + '」开通' + (type === 'blue' ? '蓝钻' : '超Q') + '吗？', '提示').then(() => {
        api.post('/admin/privileges/users/' + row.id + '/open', { type }).then(r => {
          if (r.code === 0) { this.$message.success('已开通'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    closeDlgOpen (row) {
      this.closeUser = row
      this.closeType = 'blue'
      this.closeDlg = true
    },
    doClose () {
      const row = this.closeUser
      if (!row) return
      if (this.closeType === 'all') {
        Promise.all([
          api.post('/admin/privileges/users/' + row.id + '/close', { type: 'blue' }),
          api.post('/admin/privileges/users/' + row.id + '/close', { type: 'qq' })
        ]).then(() => { this.$message.success('已取消'); this.closeDlg = false; this.load() })
        return
      }
      api.post('/admin/privileges/users/' + row.id + '/close', { type: this.closeType }).then(r => {
        if (r.code === 0) { this.$message.success('已取消'); this.closeDlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    batch (type) {
      this.$confirm('确定给所有用户开通' + (type === 'blue' ? '蓝钻' : '超Q') + '吗？', '提示').then(() => {
        api.post('/admin/privileges/batch', { type }).then(r => { if (r.code === 0) { this.$message.success('已给所有用户开通'); this.load() } else this.$message.error(r.msg) })
      }).catch(() => {})
    },
    cleanExpired () {
      this.$confirm('确定清除所有已到期的贵宾会员吗？（复刻诺哈：到期即清空等级与成长值）', '提示').then(() => {
        api.post('/admin/privileges/expired-clean').then(r => { if (r.code === 0) { this.load() } })
      }).catch(() => {})
    },
    // ===== 方案 =====
    openPlan (row) {
      this.planForm = row
        ? { id: row.id, type: row.type, name: row.name, months: Math.max(1, Math.round((row.days || 30) / 30)), money: row.money, cost: row.cost, speed: row.speed, gain: row.gain, limit: row.limit, stock: row.stock, stime: ts(row.stime), etime: ts(row.etime), status: row.status }
        : { id: 0, type: 'blue', name: '', months: 1, money: 1, cost: 0, speed: 0, gain: 0, limit: 0, stock: 0, stime: '', etime: '', status: 1 }
      this.planDlg = true
    },
    savePlan () {
      if (!this.planForm.name) { this.$message.warning('请填写销售标题'); return }
      const body = {
        type: this.planForm.type, name: this.planForm.name,
        days: this.planForm.months * 30, money: this.planForm.money,
        cost: this.planForm.cost, speed: this.planForm.speed, gain: this.planForm.gain,
        limit: this.planForm.limit, stock: this.planForm.stock,
        stime: this.planForm.stime, etime: this.planForm.etime, status: this.planForm.status
      }
      if (this.planForm.id) api.put('/admin/privileges/plans/' + this.planForm.id, body).then(r => { if (r.code === 0) { this.planDlg = false; this.load() } else this.$message.error(r.msg) })
      else api.post('/admin/privileges/plans', body).then(r => { if (r.code === 0) { this.planDlg = false; this.load() } else this.$message.error(r.msg) })
    },
    delPlan (row) {
      this.$confirm('确定删除销售「' + row.name + '」吗？', '提示').then(() => {
        api.delete('/admin/privileges/plans/' + row.id).then(r => { if (r.code === 0) this.load() })
      }).catch(() => {})
    },
    months (days) { return days % 30 === 0 ? (days / 30) : (days / 30).toFixed(1) }
  },
  watch: {
    tab () { this.page = 1; this.load() }
  }
}

function ts (t) {
  if (!t) return ''
  const d = new Date(t)
  if (isNaN(d.getTime())) return ''
  return d.getFullYear() + '-' + (d.getMonth() + 1 < 10 ? '0' : '') + (d.getMonth() + 1) + '-' + (d.getDate() < 10 ? '0' : '') + d.getDate() + ' ' + (d.getHours() < 10 ? '0' : '') + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes() + ':' + (d.getSeconds() < 10 ? '0' : '') + d.getSeconds()
}
</script>

<style scoped>
/* ===== 概览统计 ===== */
.priv-stats { display: flex; gap: 14px; flex-wrap: wrap; margin-bottom: 16px; }
.pstat {
  flex: 1; min-width: 220px; background: #fff; border-radius: 10px;
  padding: 16px 18px; display: flex; align-items: center; gap: 14px;
  border: 1px solid #f0f2f5;
  box-shadow: 0 1px 3px rgba(0,21,41,.04), 0 4px 12px rgba(0,21,41,.03);
  transition: all .2s ease;
}
.pstat:hover { transform: translateY(-2px); box-shadow: 0 6px 18px rgba(0,21,41,.09); }
.pico {
  width: 44px; height: 44px; border-radius: 11px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 22px; box-shadow: 0 4px 10px rgba(0,0,0,.12);
}
.pico-blue { background: linear-gradient(135deg, #409eff, #5cadff); }
.pico-qq { background: linear-gradient(135deg, #e6a23c, #f0c060); }
.pico-plan { background: linear-gradient(135deg, #7367f0, #9b8ff5); }
.pico-exp { background: linear-gradient(135deg, #f56c6c, #f78989); }
.pnum .n { font-size: 24px; line-height: 1.15; font-weight: 700; color: #303133; font-family: "DIN Alternate", "Segoe UI", sans-serif; }
.pnum .n b { font-size: 26px; }
.pnum .n .fade { font-size: 13px; color: #97a8be; font-weight: 400; margin-left: 4px; }
.pnum .l { font-size: 12px; color: #97a8be; margin-top: 2px; letter-spacing: .3px; }

/* ===== 页签工具条 ===== */
.tabbar { align-items: flex-end; }
.tabbar .ptabs { flex-shrink: 0; }
.tabbar .ptabs .el-tabs__header { margin: 0 0 -2px; }
.tabbar .ptabs .el-tabs__nav-wrap::after { bottom: -1px; }
.tabbar .el-tabs__nav { margin-left: 2px; }

/* ===== 说明行 ===== */
.hint { font-size: 12px; color: #97a8be; margin: 2px 0 12px; line-height: 1.6; }

/* ===== 等级表 ===== */
.lv-badge {
  display: inline-block; min-width: 42px; padding: 3px 10px; border-radius: 12px;
  background: linear-gradient(135deg, #409eff, #7367f0); color: #fff;
  font-weight: 700; font-size: 13px; letter-spacing: .5px;
  box-shadow: 0 2px 8px rgba(64,158,255,.3);
}
.num { color: #303133; font-family: "DIN Alternate", "Segoe UI", sans-serif; }
.unit { color: #97a8be; font-size: 12px; margin-left: 3px; }
.icon-cell { display: inline-flex; align-items: center; gap: 8px; }
.icon-file { font-size: 12px; max-width: 118px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
/* 图标徽章：源图多为 16px 小图，放进统一底色圆角框中，大小一致、不显毛边 */
.priv-chip {
  width: 30px; height: 30px; border-radius: 8px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  border: 1px solid #ebeef5; background: #f5f7fa;
}
.priv-chip .priv-img { width: 24px; height: 24px; object-fit: contain; display: block; vertical-align: middle; margin-right: 0; }
.chip-blue { background: #ecf5ff; border-color: #d4e8fc; }
.chip-qq { background: #fdf6ec; border-color: #f7e3c0; }
.chip-ico { font-size: 16px; }
.chip-ico.blue { color: #409eff; }
.chip-ico.qq { color: #e6a23c; }

/* ===== 会员表 ===== */
.priv-cell { display: flex; align-items: center; gap: 6px; }
.priv-cell .txt-fade { font-size: 12px; }
.priv-sub { font-size: 12px; color: #97a8be; margin-top: 2px; padding-left: 36px; }
.lv-tag { font-weight: 700; font-size: 12px; padding: 1px 7px; border-radius: 3px; }
.lv-tag.on { color: #409eff; border: 1px solid #b3d8ff; background: #ecf5ff; }
.lv-tag.off { color: #909399; border: 1px solid #dcdfe6; background: #f4f4f5; }
.st-tags { display: flex; flex-direction: column; gap: 3px; }
.st-tags .st { width: fit-content; }
.dt-line { font-size: 12px; color: #606266; line-height: 1.8; white-space: nowrap; }
.dt-ico { font-size: 12px; margin-right: 3px; }
.dt-ico.blue { color: #409eff; }
.dt-ico.qq { color: #e6a23c; }

/* ===== 销售表 ===== */
.cost { color: #e6a23c; font-weight: 700; font-family: "DIN Alternate", "Segoe UI", sans-serif; }

/* ===== 操作按钮组 ===== */
.ops { display: flex; flex-wrap: wrap; gap: 4px; justify-content: center; min-width: 0; }
.ops .el-button { margin-left: 0; }

/* ===== 分页 ===== */
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
.txt-fade { color: #909399; }

/* ===== 弹窗 ===== */
.icon-row { display: flex; align-items: center; gap: 10px; width: 100%; }
.icon-row .el-input { width: 210px; }
.group-hd {
  font-weight: 700; color: #004299; border-bottom: 1px dashed #9FC6EC;
  margin: 6px 0 4px; padding-bottom: 4px; font-size: 14px;
}
.dlg-head { font-size: 13px; color: #606266; margin-bottom: 14px; }
.dlg-head b { color: #303133; }
.dlg-tip { font-size: 12px; color: #97a8be; margin-top: 10px; }
.hint-inline { font-size: 12px; color: #97a8be; margin-left: 8px; }
</style>
