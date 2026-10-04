<template>
  <div>
    <template v-if="ezfy.cur === 'res'">
        <div class="panel" v-if="ezfy.resDetail">
          资源详情:<br/>
          名称: {{ ezfy.resNames[ezfy.resType] }}<br/>
          储量: {{ ezfy.resDetail.stock }} / 容纳: {{ ezfy.resDetail.cap }}
          <span v-if="ezfy.resDetail.store_tech > 0"> [储存技术Lv{{ ezfy.resDetail.store_tech }}: 容量+{{ ezfy.resDetail.store_tech * 2 }}%]</span>
          <br/>
          基础产量(每小时): {{ ezfy.resDetail.base }}
          <span v-if="ezfy.resDetail.tech_prod > 0"> [{{ ezfy.resTechName }}Lv{{ ezfy.resDetail.tech_prod }}: +{{ ezfy.resDetail.tech_prod * 10 }}%]</span>
          <!-- ★ 2026-09-26：基础产量 = 建筑 × 科技 × 开工率（**民心/民怨不再影响产量**，
               用户要求）。「加成产量」只放市长后勤加成 + 野地 + 道具 + 活动，恒不为负。 -->
          <span class="gray" v-if="ezfy.resDetail.base_building !== undefined">
            （建筑{{ ezfy.resDetail.base_building }} × 科技{{ 100 + (ezfy.resDetail.tech_prod || 0) * 10 }}%<template
              v-if="ezfy.resDetail.rate !== undefined && ezfy.resDetail.rate !== 100"> × 开工率{{ ezfy.resDetail.rate }}%</template>）
          </span>
          <br/>
          加成产量(每小时): {{ ezfy.resDetail.bonus }}
          <span class="gray" v-if="ezfy.resDetail.mayor_bonus > 0"> [市长后勤加成+{{ ezfy.resDetail.mayor_bonus }}%]</span>
          <!-- ★ 2026-09-30 用户要求「增产令使用了要在资源详情简约体现」：有增产效果时显示幅度 + 剩余时长 -->
          <span class="green" v-if="ezfy.resDetail.boost_pct > 0">
            [增产令+{{ ezfy.resDetail.boost_pct }}% {{ ezfy.fmtLeft(Math.floor((ezfy.resDetail.boost_until - Date.now()) / 1000)) }}]
          </span>
          <span class="gray" v-if="ezfy.resDetail.bonus === 0"> [暂无加成]</span>
          <br/>
          耗量(每小时): {{ ezfy.resDetail.consume }}<br/>
          <template v-if="ezfy.resType === 'food'">
            军队耗粮: {{ ezfy.fmtBig(ezfy.resDetail.troop_consume_raw !== undefined ? ezfy.resDetail.troop_consume_raw : (ezfy.resDetail.troop_consume || 0)) }}
          <template v-if="ezfy.resDetail.supply_tech > 0">
            [补给技巧Lv{{ ezfy.resDetail.supply_tech }}: -{{ ezfy.resDetail.supply_tech * 2 }}%
            → 实扣 {{ ezfy.fmtBig(ezfy.resDetail.troop_consume || 0) }}]
          </template>
          <span v-else class="gray"> [补给技巧未研究, 无减免]</span><br/>
          </template>
          总产量(每小时): <span :class="{ red: ezfy.resDetail.total < 0 }">{{ ezfy.resDetail.total }}</span><br/>
          <br/>
          {{ ezfy.resDes[ezfy.resType] }}
          <br/>
          <a href="javascript:;" @click="ezfy.go('builds')">[资源区]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'buildm' || ezfy.cur === 'builds'">
        <div class="panel">
          <div class="old-line">
            {{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }})
            <a href="javascript:;" @click="ezfy.go('cities')">切换城市</a>
          </div>
          <!-- ★ 军事区/资源区导航固定顺序「军事区. 资源区」，当前项加粗高亮；不再谁当前谁排第一 -->
          <div class="old-line ezfy-subnav">
            <a href="javascript:;" :class="{ on: ezfy.cur === 'buildm' }" @click="ezfy.go('buildm')">军事区</a>.
            <a href="javascript:;" :class="{ on: ezfy.cur === 'builds' }" @click="ezfy.go('builds')">资源区</a>
          </div>
          <div class="old-line">建造中队列数：{{ ezfy.buildQueueCount }} | 已有 {{ ezfy.zoneCount }}/{{ ezfy.zoneCap }}
            <a href="javascript:;" @click="ezfy.openBuildPre(ezfy.cur === 'buildm' ? 'm' : 's')">建造</a>
          </div>
          <!-- 已建建筑: 一行一个 —— 名称 (N级) 升级 一键9级 拆除 -->
          <div class="old-line" v-for="b in ezfy.zoneBuilt" :key="'zb' + b.id">
            <span v-if="ezfy.bEntry(b.building_id)">
              <a href="javascript:;" @click="ezfy.goEntry(b.building_id)">{{ b.name }}</a>
            </span>
            <span v-else>{{ b.name }}</span>
            ({{ b.level }}级)
            <template v-if="b.status !== 0">
              <span class="orange">施工中 {{ ezfy.remain(b.end_time) }}</span>
              <!-- ★ 2026-09-26 修复「加速道具买完实际使用不生效」：按背包里**实际拥有**的
                   建筑加速道具(item_type=3)逐档渲染，点哪档就用哪档（原来是自动挑最短的，
                   玩家买了 2 小时却只减 30 分钟，看着就像「买了没用上」）。 -->
              <a v-for="a in ezfy.accItems(3)" :key="'sb' + b.id + '_' + a.cfg_id" href="javascript:;"
                 @click="ezfy.doSpeedBuilding(b, a)">[加速{{ ezfy.accLabel(a) }}]</a>
              <!-- ★ 2026-09-27 百分比加速道具(item_type=24)：剩余时间减 30%/60%/80% -->
              <a v-for="a in ezfy.accItems(24)" :key="'sbp' + b.id + '_' + a.cfg_id" href="javascript:;"
                 @click="ezfy.doSpeedBuilding(b, a)">[加速{{ ezfy.accLabel(a) }}]</a>
              <span class="gray" v-if="!ezfy.accItems(3).length && !ezfy.accItems(24).length">(无建筑加速道具)</span>
              <!-- ★ 2026-09-27：升级状态时加 [取消]（取消零退还，建筑保留当前等级，防刷资源/图纸） -->
              <a href="javascript:;" class="red" @click="ezfy.doCancelUpgrade(b)">[取消]</a>
            </template>
            <template v-else-if="b.level > 0 && b.level < b.max_level">
              <span class="build-act">
                <a href="javascript:;" @click="ezfy.doUpgrade(b)">升级</a>
                <!-- ★ 2026-09-25 用户纠正：按钮语义是「一键升级到 max_level-1 级」，
                     已经到达该等级就不再显示（原来会显示成 9 级但实际升满级）
                     ★ 2026-09-30 用户要求：一键只到 9 级（司令部 12 级时原显示「一键11级」，
                     现统一「一键9级」，10/11/12 级手动升级，每级都要建筑图纸） -->
                <a v-if="b.level < 9" href="javascript:;" @click="ezfy.doMaxLevel(b)">一键9级</a>
                <a v-if="b.can_delete === 1" href="javascript:;" @click="ezfy.doDeleteBuilding(b)">拆除</a>
              </span>
            </template>
            <template v-else>
              <a v-if="b.level === 0" href="javascript:;" @click="ezfy.doUpgrade(b)">建成中待完成</a>
              <a v-else-if="b.can_delete === 1" href="javascript:;" @click="ezfy.doDeleteBuilding(b)">拆除</a>
            </template>
            <div v-if="ezfy.inlineTip && ezfy.inlineTip.bid === b.id" class="build-tip" :class="ezfy.inlineTip.type">
              <span>{{ ezfy.inlineTip.text }}</span>
              <a href="javascript:;" @click="ezfy.inlineTip = null">[关闭]</a>
            </div>
          </div>
          <div class="old-line gray" v-if="!ezfy.zoneBuilt.length">(本区还没有建筑, 点上面的「建造」)</div>
          <br/>
          <div class="old-line">
            <!-- ★ 2026-10-05 恢复：训练一键加速（前后端已加 5 秒卡控，防连点/脚本刷黄金） -->
            <a href="javascript:;" @click="ezfy.doSpeedTrainAll">[训练一键加速]</a>|
            <a href="javascript:;" @click="ezfy.doSpeedTrainAllCity">[所有城市训练一键加速]</a>
          </div>
          <!-- ★ 训练加速道具(item_type=4)的入口：上面两个是「花黄金一键完成」，
               这里才是商城买的「训练加速30分钟/2小时」真正被消耗的地方。 -->
          <div class="old-line" v-if="ezfy.accItems(4).length || ezfy.accItems(25).length">
            训练加速道具:
            <a v-for="a in ezfy.accItems(4)" :key="'stb' + a.cfg_id" href="javascript:;"
               @click="ezfy.doSpeedTrain(null, a)">[加速{{ ezfy.accLabel(a) }}]×{{ a.count }}</a>
            <!-- ★ 2026-09-27 百分比训练加速(item_type=25)：剩余时间减 30%/60%/80% -->
            <a v-for="a in ezfy.accItems(25)" :key="'stbp' + a.cfg_id" href="javascript:;"
               @click="ezfy.doSpeedTrain(null, a)">[加速{{ ezfy.accLabel(a) }}]×{{ a.count }}</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'buildpre'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go(ezfy.buildZone === 'm' ? 'buildm' : 'builds')">
              {{ ezfy.buildZone === 'm' ? '军事区' : '资源区' }}</a> 建造
          </div>
          <div class="old-line">可建造的建筑：</div>
          <div class="old-line" v-for="b in ezfy.zonePool" :key="'bp' + b.building_id">
            {{ b.name }}
            <a href="javascript:;" @click="ezfy.openBuildDetail(b)">详情</a>
            <a href="javascript:;" @click="ezfy.doBuild(b)">建造</a>
          </div>
          <div class="old-line gray" v-if="!ezfy.zonePool.length">(本区暂无可建造的建筑)</div>

          <!-- 选中建筑后展开: 说明 + 造价 -->
          <template v-if="ezfy.buildSel">
            <hr/>
            <div class="panel-title">{{ ezfy.buildSel.name }}</div>
            <div class="old-line">{{ ezfy.buildSel.des }}</div>
            <div class="old-line gray">
              造价: {{ ezfy.resShort.food }}{{ ezfy.buildSel.cost.food }} {{ ezfy.resShort.steel }}{{ ezfy.buildSel.cost.steel }} {{ ezfy.resShort.oil }}{{ ezfy.buildSel.cost.oil }}
              {{ ezfy.resShort.rare }}{{ ezfy.buildSel.cost.rare }} {{ ezfy.resShort.gold }}{{ ezfy.buildSel.cost.gold }}
              需{{ Math.ceil(ezfy.buildSel.time / 60) }}分钟
            </div>
            <div class="old-line">
              <button @click="ezfy.doBuild(ezfy.buildSel)">[建造]</button>
              <a href="javascript:;" @click="ezfy.buildSel = null">[收起]</a>
            </div>
          </template>
          <a href="javascript:;" @click="ezfy.go(ezfy.buildZone === 'm' ? 'buildm' : 'builds')">[返回{{ ezfy.buildZone === 'm' ? '军事区' : '资源区' }}]</a>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyBuild',
  inject: ['ezfy']
}
</script>
