<template>
  <div>
    <template v-if="ezfy.cur === 'troop'">
        <div class="panel">
          <div class="panel-title">训练军队(军工厂{{ ezfy.factoryLevels }}级, 队列{{ ezfy.queues.length }}/{{ ezfy.factoryTotal }})</div>
          <div class="old-line green" v-if="ezfy.troopsData.train_discount > 0">
            节日活动·造兵打折：资源消耗 -{{ ezfy.troopsData.train_discount }}%（下方为折后价）
          </div>
          <div class="old-line">人口:{{ ezfy.troopsData.pop }} 空闲:{{ ezfy.freePop }} | 围墙:{{ ezfy.troopsData.wall_level }}级</div>
          <!-- ★ 空闲人口 = 人口 - 占用；占用只算「训练中、还没出厂」的新兵。
               已训练完成的部队（含出征在外的）不占人口位置 —— 用户 2026-09-21 明确的规则 -->
          <div class="old-line gray" v-if="ezfy.popUsed > 0">
            训练中占用人口：{{ ezfy.popUsed }}
          </div>
          <div class="old-line" v-for="t in ezfy.trainCfgs" :key="'tt' + t.id">
            <!-- ★ 2026-09-28 用户要求：军队列表只留名称和类型，属性/消耗/前提都进[训练]详情页看 -->
            <a href="javascript:;" @click="ezfy.openTroopView(t.id)">{{ t.name }}</a>({{ ezfy.troopTypeName(t.type) }})
            <a href="javascript:;" @click="ezfy.openTrainPre(t, 'troop')">[训练]</a><br/>
          </div>
          <div class="panel-title">训练队列({{ ezfy.queues.length }})</div>
          <div class="old-line" v-for="q in ezfy.queues" :key="'q' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ ezfy.remain(q.end_time) }}
            <!-- 接口只返回 status=0（训练中）的队列，所以这里不需要再判断状态 -->
            <!-- ★ 2026-09-26 修复：训练页原来只有 [取消]，商城买的「训练加速」道具无处可用 -->
            <a v-for="a in ezfy.accItems(4)" :key="'sq' + q.id + '_' + a.cfg_id" href="javascript:;"
               @click="ezfy.doSpeedTrain(q, a)">[加速{{ ezfy.accLabel(a) }}]</a>
            <!-- ★ 2026-09-27 百分比训练加速(item_type=25) -->
            <a v-for="a in ezfy.accItems(25)" :key="'sqp' + q.id + '_' + a.cfg_id" href="javascript:;"
               @click="ezfy.doSpeedTrain(q, a)">[加速{{ ezfy.accLabel(a) }}]</a>
            <span class="gray" v-if="!ezfy.accItems(4).length && !ezfy.accItems(25).length">(无训练加速道具)</span>
            <a href="javascript:;" @click="ezfy.doCancelTrain(q)">[取消]</a>
          </div>
          <div class="old-line" v-if="!ezfy.queues.length">(队列为空)</div>
          <a href="javascript:;" @click="ezfy.go('defence')">[去建城防]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'defence'">
        <div class="panel">
          <div class="old-line">
            围墙：{{ ezfy.troopsData.wall_level }}级 城防空间：({{ ezfy.troopsData.defence_space_used }}/{{ ezfy.troopsData.defence_space }})
          </div>
          <div class="old-line">正在建造:</div>
          <div class="old-line" v-for="q in ezfy.defenceQueues" :key="'dq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ ezfy.remain(q.end_time) }}
          </div>
          <div class="old-line gray" v-if="!ezfy.defenceQueues.length">(无)</div>
          <table>
            <tr v-for="t in ezfy.defenceCfgs" :key="'dt' + t.id">
              <td class="nm"><a href="javascript:;" @click="ezfy.openTroopView(t.id)">{{ t.name }}</a>:</td>
              <td>{{ ezfy.troopCount(t.id) }}</td>
              <td>
                <a href="javascript:;" @click="ezfy.openTrainPre(t, 'defence')">[建造]</a>
                <a href="javascript:;" @click="ezfy.doDismiss(t)">[拆除]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray">城防设施占用「城防空间」(围墙容量)，不占用人口。</div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'troops'">
        <div class="panel">
          <div class="panel-title">城内军队</div>
          <!-- ★ 用户要求：这张表数据「上下居中、左右居中」，操作列也一起对齐 -->
          <table class="ezfy-center-tbl">
            <tr><th class="nm">兵种</th><th>数量</th><th>操作</th></tr>
            <!-- ★ 2026-09-28 用户要求：首页点「军队」要能看到全部兵种（数量为 0 的也显示），每行后跟训练操作 -->
            <tr v-for="t in ezfy.armyRows" :key="'tv' + t.id">
              <td class="nm"><a href="javascript:;" @click="ezfy.openTroopView(t.id)">{{ t.name }}</a></td>
              <td>{{ t.count }}</td>
              <td>
                <!-- 训练/建造：防御兵种(type 4)走城防建造，其余直接训练 -->
                <a href="javascript:;" @click="ezfy.openTrainPre(t, t.type === 4 ? 'defence' : 'troop')">[{{ t.type === 4 ? '建造' : '训练' }}]</a>
                <!-- 解散：数量由玩家自己输入（用户要求），数量为 0 时无意义、不显示 -->
                <a v-if="t.count > 0" class="red" href="javascript:;" @click="ezfy.doDisband(t)">[解散]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.armyRows.length">(暂无兵种配置)</div>
          <br/>
          <div class="panel-title">训练队列({{ ezfy.queues.length }})</div>
          <div class="old-line" v-for="q in ezfy.queues" :key="'tq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ ezfy.remain(q.end_time) }}
            <!-- 接口只返回 status=0（训练中）的队列，所以这里不需要再判断状态 -->
            <a href="javascript:;" @click="ezfy.doCancelTrain(q)">[取消]</a>
          </div>
          <div class="old-line" v-if="!ezfy.queues.length">(队列为空)</div>
          <br/>
          <a href="javascript:;" @click="ezfy.go('troop')">[造兵]</a>
          <a href="javascript:;" @click="ezfy.go('defence')">[建防]</a>
          <a href="javascript:;" @click="ezfy.go('hq')">[司令部]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'hq'">
        <div class="panel">
          <div class="panel-title">司令部</div>
          <!-- ★ 2026-09-28 用户要求：司令部内部拆成 tab（兵种配置/出征队列/伤兵营/逃兵营），
               刷新后记住上次所在 tab（localStorage, 照抄任务 tab 的 ezfy_task_tab 写法） -->
          <div class="acade-tab hq-tab">
            <a href="javascript:;" :class="{ on: ezfy.hqTab === 0 }" @click="ezfy.selectHqTab(0)">兵种配置</a>
            <a href="javascript:;" :class="{ on: ezfy.hqTab === 1 }" @click="ezfy.selectHqTab(1)">出征队列({{ ezfy.orders.length }})</a>
            <a href="javascript:;" :class="{ on: ezfy.hqTab === 2 }" @click="ezfy.selectHqTab(2)">伤兵营({{ ezfy.woundedList(0).length }})</a>
            <a href="javascript:;" :class="{ on: ezfy.hqTab === 3 }" @click="ezfy.selectHqTab(3)">逃兵营({{ ezfy.woundedList(1).length }})</a>
            <a href="javascript:;" :class="{ on: ezfy.hqTab === 4 }" @click="ezfy.selectHqTab(4)">预设编队({{ ezfy.presets.length }})</a>
          </div>
          <div class="panel-title" v-show="ezfy.hqTab === 0">兵种战斗配置</div>
          <div v-show="ezfy.hqTab === 0">
          <!-- ★ 一个兵种一块（原来 5 列固定宽度表格在手机上会互相遮盖） -->
          <div class="ezfy-tgt-block" v-for="t in ezfy.troopsData.cfgs" :key="'cfg' + t.id">
            <div class="ezfy-tgt-name">
              {{ t.name }}
              <span v-if="ezfy.isDefenceTroop(t)" class="gray">（防御兵种：固定阵地，不能前进/后退，也不能出征）</span>
            </div>
            <div class="ezfy-tgt-row">
              <span class="ezfy-tgt-lab">进攻目标</span>
              <select v-model="ezfy.targetCfg[t.id].atk" style="width:110px">
                <option :value="0">最近目标</option>
                <option v-for="tt in ezfy.troopsData.cfgs" :key="'a' + tt.id" :value="tt.id">{{ tt.name }}</option>
              </select>
              <span class="ezfy-tgt-lab">进攻</span>
              <select v-model="ezfy.targetCfg[t.id].atkMove" style="width:80px"
                      :disabled="ezfy.isDefenceTroop(t)">
                <option :value="1">前进</option><option :value="0">停止</option>
              </select>
            </div>
            <div class="ezfy-tgt-row">
              <span class="ezfy-tgt-lab">防守目标</span>
              <select v-model="ezfy.targetCfg[t.id].def" style="width:110px">
                <option :value="0">最近目标</option>
                <option v-for="tt in ezfy.troopsData.cfgs" :key="'d' + tt.id" :value="tt.id">{{ tt.name }}</option>
              </select>
              <span class="ezfy-tgt-lab">防守</span>
              <select v-model="ezfy.targetCfg[t.id].defMove" style="width:110px"
                      :disabled="ezfy.isDefenceTroop(t)">
                <option :value="1">前进</option><option :value="0">停止</option><option :value="-1">不参与防御</option>
              </select>
            </div>
          </div>
          <div class="old-line"><a href="javascript:;" @click="ezfy.doSaveTargets">[保存全部配置]</a></div>
          </div><!-- /兵种配置 tab -->
          <div v-show="ezfy.hqTab === 1">
          <div class="panel-title">出征队列({{ ezfy.orders.length }})</div>
          <table>
            <tr><th>类型</th><th>目标</th><th>统帅</th><th>状态</th><th></th></tr>
            <tr v-for="o in ezfy.orders" :key="'o' + o.id">
              <td>{{ o.type_name }}</td>
              <td>({{ o.target_x }},{{ o.target_y }})</td>
              <td>{{ o.officer || '无' }}</td>
              <td>{{ ezfy.orderStatusText(o) }}</td>
              <td>
                <a href="javascript:;" @click="ezfy.openOrder(o)">[详情]</a>
                <!-- ★ 用户要求「出征队列可以取消」：所有还在外面的命令（行进中/驻守中）都能取消 -->
                <a v-if="o.status === 0 || o.status === 1" class="red"
                   href="javascript:;" @click="ezfy.doRecall(o)">[取消]</a>
                <!-- ★ 2026-09-30 行军计谋：神兵天降=去程减80%（行进中）、战略转移=回程减360分钟（返回中） -->
                <a v-if="o.status === 0 && o.scheme_fast === 0" class="green"
                   href="javascript:;" @click="ezfy.doMarchScheme(o, 13)">[神兵天降]</a>
                <a v-if="o.status === 0 && o.scheme_fast === 1" class="gray">[神兵天降·已用]</a>
                <a v-if="o.status === 2 && o.scheme_back === 0" class="green"
                   href="javascript:;" @click="ezfy.doMarchScheme(o, 14)">[战略转移]</a>
                <a v-if="o.status === 2 && o.scheme_back === 1" class="gray">[战略转移·已用]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.orders.length">(暂无出征部队)</div>
          </div><!-- /出征队列 tab -->
          <div v-show="ezfy.hqTab === 2">
          <div class="panel-title">伤兵营</div>
          <div class="old-line gray">伤兵在营中<b>不消耗粮食</b>；恢复出厂需要黄金（按兵种造价折算）。</div>
          <table>
            <tr><th class="nm">兵种</th><th>数量</th><th>恢复费用</th><th>操作</th></tr>
            <tr v-for="w in ezfy.woundedList(0)" :key="'w' + w.id">
              <td class="nm">{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ ezfy.fmtN((w.heal_gold || 0) * w.count) }} {{ ezfy.resNames.gold }}</td>
              <td><a href="javascript:;" @click="ezfy.doRecover(w)">[恢复]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.woundedList(0).length">(伤兵营无伤兵)</div>
          <div class="old-line" v-if="ezfy.woundedList(0).length">
            合计 <b>{{ ezfy.fmtN(ezfy.woundedHealCost(0)) }}</b> {{ ezfy.resNames.gold }}
            <button @click="ezfy.doRecoverAll(0)">[全部恢复]</button>
          </div>
          </div><!-- /伤兵营 tab -->
          <div v-show="ezfy.hqTab === 3">
          <div class="panel-title">逃兵营</div>
          <table>
            <tr><th class="nm">兵种</th><th>数量</th><th>召回费用</th><th>操作</th></tr>
            <tr v-for="w in ezfy.woundedList(1)" :key="'dsw' + w.id">
              <td class="nm">{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ ezfy.fmtN((w.heal_gold || 0) * w.count) }} {{ ezfy.resNames.gold }}</td>
              <td><a href="javascript:;" @click="ezfy.doRecover(w)">[召回]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.woundedList(1).length">(逃兵营无逃兵)</div>
          <div class="old-line" v-if="ezfy.woundedList(1).length">
            合计 <b>{{ ezfy.fmtN(ezfy.woundedHealCost(1)) }}</b> {{ ezfy.resNames.gold }}
            <button @click="ezfy.doRecoverAll(1)">[全部召回]</button>
          </div>
          </div><!-- /逃兵营 tab -->
          <div v-show="ezfy.hqTab === 4">
          <!-- ★ 2026-09-28 用户要求：司令部新增「预设编队」tab —— 镜像出征页 ①②③⑥ 保存模板，
               不含 ④随军资源 / ⑤宿营；⑥油耗计算保留（预设不含目标，按本城0距离估算）。 -->
          <div class="panel-title">预设编队</div>
          <div class="old-line gray">
            预设 = 出征模板（指挥军官 + 集结令 + 兵力），不含随军资源/宿营。
            保存后在<b>地图出征页</b>的「预设编队」下拉里选用，一键回填。
          </div>
          <template v-if="ezfy.presetAdding">
            <div class="of-sec">新增预设 · 名称
              <input v-model="ezfy.presetName" type="text" maxlength="20" placeholder="(最多20字)" style="width:140px"/>
            </div>
            <!-- ① 军官 -->
            <div class="of-sec">① 指挥军官</div>
            <div class="old-line">
              <select v-model="ezfy.orderOfficer" @change="ezfy.presetCalc">
                <option value="0">未指定</option>
                <option v-for="o in ezfy.onDutyOfficers" :key="'pd' + o.id" :value="o.name">
                  {{ o.name }}({{ o.level }}级) 军{{ o.military_total || o.military }}
                  <span class="green" v-if="o.equip_military">(装+{{ o.equip_military }})</span>
                  学{{ o.learning_total || o.learning }} 后{{ o.logistics_total || o.logistics }}
                </option>
              </select>
              <span v-if="!ezfy.onDutyOfficers.length" class="gray">(「{{ ezfy.city.name }}」暂无可带队军官)</span>
            </div>
            <!-- ② 集结令 -->
            <div class="of-sec">② 出征集结令</div>
            <div class="old-line">
              使用
              <input type="number" min="0" :max="ezfy.gatherMax" v-model.number="ezfy.orderGather"
                     :disabled="ezfy.gatherCount <= 0" @change="ezfy.onPresetGatherChange" style="width:80px"/>
              个 <span class="gray">（背包里有 {{ ezfy.gatherCount }} 个，单次最多 {{ ezfy.orderCapMax }} 个）</span>
            </div>
            <!-- ★ 2026-09-29 用户要求：预设页与出征页一致，集结令下方直接显示「本次出兵 / 上限」 -->
            <div class="old-line" v-if="ezfy.attackTroops.length">
              <span :class="ezfy.orderOverCap ? 'red' : 'green'">
                本次出兵 <b>{{ ezfy.fmtN(ezfy.orderTroopTotal) }}</b> / 上限 <b>{{ ezfy.orderCapText }}</b>
                <template v-if="ezfy.orderOverCap">—— 超出上限，请减少兵力或加用集结令</template>
              </span>
            </div>
            <!-- ③ 兵力 -->
            <div class="of-sec">③ 选择兵力</div>
            <div class="of-rows">
              <div class="of-row" v-for="t in ezfy.trainCfgs" :key="'pt' + t.id"
                   :class="{ 'of-off': ezfy.troopCount(t.id) <= 0 }"
                   :title="t.name + '（现有 ' + ezfy.fmtN(ezfy.troopCount(t.id)) + '，本次最多可派 ' + ezfy.fmtN(ezfy.orderQtyMax(t.id)) + '）'">
                <span class="of-name">{{ t.name }}</span>
                <span class="of-avail">现有 {{ ezfy.fmtN(ezfy.troopCount(t.id)) }}</span>
                <span class="of-ctl">
                  <input type="range" class="of-range" min="0" step="1"
                          :max="ezfy.troopCount(t.id)" :value="ezfy.orderQty(t.id)"
                          :disabled="ezfy.orderQtyMax(t.id) <= 0"
                          @input="ezfy.onOrderQtyInput(t.id, $event)"/>
                  <input type="number" class="of-num" min="0" placeholder="0"
                         :max="ezfy.orderQtyMax(t.id)" :value="ezfy.orderQty(t.id)"
                         :disabled="ezfy.orderQtyMax(t.id) <= 0"
                         @input="ezfy.onOrderQtyInput(t.id, $event)"/>
                  <a href="javascript:;" class="of-max"
                     :class="{ 'of-max-off': ezfy.orderQtyMax(t.id) <= 0 }"
                     @click="ezfy.setOrderQtyMax(t.id)">[最大]</a>
                </span>
              </div>
            </div>
            <!-- ⑥ 消耗预览（不含目标 → 按本城 0 距离估算） -->
            <div class="of-sec">⑥ 消耗预览</div>
            <div class="old-line">
              <button @click="ezfy.presetCalc">[计算]</button>
              油耗：<span class="orange">{{ ezfy.orderCalc ? ezfy.orderCalc.oil_used : '—' }}</span>
              &nbsp;/&nbsp;负重：<span class="orange">{{ ezfy.orderCalc ? ezfy.orderCalc.carry : '—' }}</span>
            </div>
            <div class="old-line gray">
              （预设不含目标，油耗/负重按本城 0 距离估算；实际油耗与耗时以出征页 [计算] 为准）
            </div>
            <div class="old-line">
              <button @click="ezfy.savePreset">[保存预设]</button>
              <a href="javascript:;" @click="ezfy.cancelPresetAdd">[取消]</a>
            </div>
          </template>
          <div class="old-line" v-else>
            <a href="javascript:;" @click="ezfy.startPresetAdd">[新增预设编队]</a>
            <span class="gray">（最多保存 {{ ezfy.presetMax }} 个）</span>
          </div>
          <div class="panel-title">我的预设</div>
          <table>
            <tr><th class="nm">名称</th><th>军官</th><th>兵力</th><th>集结令</th><th></th></tr>
            <tr v-for="p in ezfy.presets" :key="'psl' + p.id">
              <td class="nm">{{ p.name }}</td>
              <td>{{ p.officer || '无' }}</td>
              <td>{{ ezfy.fmtN(p.troop_total) }}</td>
              <td>{{ p.gather }}个</td>
              <td><a href="javascript:;" class="red" @click="ezfy.deletePreset(p)">[删除]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.presets.length">(还没有预设编队，点上方 [新增预设编队] 开始)</div>
          </div><!-- /预设编队 tab -->
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'factory'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('buildm')">军事区</a>
            -&gt;军营(军工厂)({{ ezfy.factoryLevels }}级)：
          </div>
          <div class="old-line">正在训练：</div>
          <div class="old-line" v-for="q in ezfy.queues" :key="'fq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ ezfy.remain(q.end_time) }}
          </div>
          <div class="old-line gray" v-if="!ezfy.queues.length">(无)</div>
          <div class="old-line" v-for="t in ezfy.trainCfgs" :key="'ft' + t.id">
            <a href="javascript:;" @click="ezfy.openTroopView(t.id)">{{ t.name }}</a> : {{ ezfy.troopCount(t.id) }}
            <a href="javascript:;" @click="ezfy.openTrainPre(t, 'troop')">[训练]</a>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('troops')">[城内军队]</a>
            <a href="javascript:;" @click="ezfy.go('buildm')">[返回军事区]</a>
            <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
          </div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'troopview'">
        <div class="panel" v-if="ezfy.troopView">
          <div class="old-line">
            {{ ezfy.troopView.name }}:
            <span class="gray">(现有 {{ ezfy.troopCount(ezfy.troopView.id) }})</span>
          </div>
          <div class="old-line">训练/建造需要：</div>
          <table>
            <tr>
              <td>{{ ezfy.resNames.food }}：</td><td>{{ ezfy.troopView.cost.food }}</td>
              <td>{{ ezfy.resNames.steel }}：</td><td>{{ ezfy.troopView.cost.steel }}</td>
            </tr>
            <tr>
              <td>{{ ezfy.resNames.oil }}：</td><td>{{ ezfy.troopView.cost.oil }}</td>
              <td>{{ ezfy.resNames.rare }}：</td><td>{{ ezfy.troopView.cost.rare }}</td>
            </tr>
            <tr>
              <td>耗时：</td><td>{{ ezfy.durText(ezfy.troopView.train_time) }}</td>
              <td>油耗：</td><td>{{ ezfy.troopView.oil_keep }}</td>
            </tr>
            <tr>
              <td>耗粮：</td><td>{{ ezfy.troopView.food_keep }}</td>
              <td>人口：</td><td>{{ ezfy.troopView.pop }}</td>
            </tr>
            <tr>
              <td>生命：</td><td>{{ ezfy.troopView.health }}</td>
              <td>对地：</td><td>{{ ezfy.troopView.atk_ground }}</td>
            </tr>
            <tr>
              <td>对空：</td><td>{{ ezfy.troopView.atk_air }}</td>
              <td>对海：</td><td>{{ ezfy.troopView.atk_sea }}</td>
            </tr>
            <tr>
              <td>对防：</td><td>{{ ezfy.troopView.atk_def }}</td>
              <td>速度：</td><td>{{ ezfy.troopView.speed }}</td>
            </tr>
            <tr>
              <td>军种：</td><td>{{ ezfy.troopTypeName(ezfy.troopView.type) }}</td>
              <td>射程：</td><td>{{ ezfy.troopView.attack_range }}</td>
            </tr>
            <tr>
              <td>攻速：</td><td>{{ ezfy.troopView.speed }}</td>
              <td>负重：</td><td>{{ ezfy.troopView.carry }}</td>
            </tr>
            <tr>
              <td>防御：</td><td>{{ ezfy.troopView.defence }}</td>
              <td>修复率：</td><td>{{ ezfy.troopView.repair_rate }}%</td>
            </tr>
          </table>
          <div class="old-line">
            {{ ezfy.troopView.type === 4 ? '围墙' : '军工厂' }}: {{ ezfy.troopView.type === 4 ? ezfy.troopsData.wall_level : ezfy.factoryLevels }}级
            <template v-if="ezfy.troopView.require"><br/>前提: {{ ezfy.troopView.require }}</template>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.openTrainPre(ezfy.troopView, ezfy.troopView.type === 4 ? 'defence' : 'troop')">
              [{{ ezfy.troopView.type === 4 ? '建造' : '训练' }}]
            </a>
          </div>
          <a href="javascript:;" @click="ezfy.go(ezfy.troopViewBack)">[返回]</a>
          <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请选择兵种 <a href="javascript:;" @click="ezfy.go('troops')">[城内军队]</a></div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'trainpre'">
        <div class="panel" v-if="ezfy.trainSel">
          <div class="old-line" v-if="ezfy.trainMode === 'defence'">
            【城防建造】城防空间:{{ ezfy.troopsData.defence_space_used }}/{{ ezfy.troopsData.defence_space }}
          </div>
          <div class="old-line" v-else>
            <a href="javascript:;" @click="ezfy.go('buildm')">军事区</a>
            -&gt;军工厂({{ ezfy.factoryLevels }}级)：
          </div>
          <div class="old-line">{{ ezfy.trainSel.name }}{{ ezfy.trainMode === 'defence' ? '建造' : '训练' }}需求：</div>
          <div class="old-line">
            {{ ezfy.resNames.food }}：{{ ezfy.trainSel.cost.food }}<br/>
            {{ ezfy.resNames.steel }}：{{ ezfy.trainSel.cost.steel }}<br/>
            {{ ezfy.resNames.oil }}：{{ ezfy.trainSel.cost.oil }}<br/>
            {{ ezfy.resNames.rare }}：{{ ezfy.trainSel.cost.rare }}<br/>
            {{ ezfy.trainMode === 'defence' ? '城防空间' : '人口' }}：{{ ezfy.trainSel.pop }}<br/>
            <template v-if="ezfy.trainMode !== 'defence'">吃粮：{{ ezfy.trainSel.food_keep }}<br/></template>
            时间：{{ ezfy.durText(ezfy.trainSel.train_time) }}<br/>
            <template v-if="ezfy.trainMode !== 'defence'">需要军工厂：{{ ezfy.trainSel.need_factory }}级<br/></template>
            <!-- ★ 2026-09-28 用户要求：资源/前提条件移到这里（训练详情页）展示 -->
            前提：{{ ezfy.trainSel.require || '无' }}<template v-if="ezfy.trainSel.type === 1"> <span class="red">(海军: 仅海城可训练)</span></template><br/>
          </div>
          <div class="old-line green" v-if="ezfy.troopsData.train_discount > 0 && ezfy.trainMode !== 'defence'">
            节日活动·造兵打折：资源消耗 -{{ ezfy.troopsData.train_discount }}%（上方为折后价）
          </div>
          <div class="old-line">
            建造数量：
            <input v-model="ezfy.trainCount" type="number" min="1" :placeholder="'(1~' + ezfy.maxTrainable + ')'" style="width:90px"/>
            <span class="gray">(最多 {{ ezfy.maxTrainable }})</span>
            <!-- ★ 2026-09-28 用户要求：在「(最多 N)」后面加 [最大]，一键把数量填成上限 -->
            <a href="javascript:;" class="train-max"
               :class="{ 'train-max-off': ezfy.maxTrainable <= 0 }"
               @click="ezfy.setTrainMax()">[最大]</a>
          </div>
          <div class="old-line red" v-if="ezfy.maxTrainable <= 0">
            当前无法{{ ezfy.trainMode === 'defence' ? '建造' : '训练' }}：资源或{{ ezfy.trainMode === 'defence' ? '城防空间' : '人口' }}不足
            <template v-if="ezfy.trainMode !== 'defence'">
              <a href="javascript:;" @click="ezfy.go('home')">[回首页召集人口]</a>
            </template>
            <template v-else>
              <a href="javascript:;" @click="ezfy.go('buildm')">[去军事区升级围墙]</a>
            </template>
          </div>
          <div class="old-line">
            <span>操作选项：</span>
            <label><input type="radio" :value="true" v-model="ezfy.trainSplit"/>[全部工厂]</label>
            <label><input type="radio" :value="false" v-model="ezfy.trainSplit"/>[仅此工厂]</label>
            <span class="gray" v-if="ezfy.trainSplit && ezfy.trainMode !== 'defence'">(可用{{ ezfy.eligibleFactories }}座军工厂平分)</span>
          </div>
          <div class="old-line">预计耗时：{{ ezfy.trainEstimateText }}</div>
          <div class="old-line">
            <button @click="ezfy.doTrainPre()">{{ ezfy.trainMode === 'defence' ? '开始建造' : '开始训练' }}</button>
          </div>
          <a href="javascript:;" @click="ezfy.go(ezfy.trainMode === 'defence' ? 'defence' : 'factory')">[返回]</a>
          <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先选择兵种 <a href="javascript:;" @click="ezfy.go('troops')">[城内军队]</a></div>
        </div>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化：军队相关页面模板独立成组件（troop/defence/troops/hq/factory/troopview/trainpre）。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyArmy',
  inject: ['ezfy']
}
</script>
