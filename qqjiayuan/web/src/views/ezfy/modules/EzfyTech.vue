<template>
  <div>
    <template v-if="ezfy.cur === 'techs'">
        <div class="panel">
          <div class="panel-title">【科技中心】:{{ ezfy.techsData.academy }}级</div>
          <div class="old-line" v-for="t in ezfy.techsData.techs" :key="'te' + t.tech_id">
            <!-- ★ 2026-09-28 用户要求：科技列表不再显示资源消耗/前置条件，点[研究]进详情页查看 -->
            <!-- ★ 2026-09-28 用户要求版式：第一行「名称 等级/满级级 [研究N级]」，第二行才是效果。
                 原来 [研究N级] 被挤在效果下面第三行，扫一眼看不出「这条能不能升」。 -->
            <b>{{ t.name }}</b> {{ t.level }}/{{ t.max_level }}级
            <span v-if="t.researching" class="orange">研究中 {{ ezfy.remain(t.end_time) }}
              <!-- ★ 2026-09-26 修复「科技加速道具买完实际使用不生效」：原来这里的 [加速]
                   调的是 /techs/speed，是**免费减 10 分钟**（minutes 还能由前端随便传），
                   商城买的「科技加速30分钟/2小时」根本没被消耗。现在改为消耗道具。 -->
              <a v-for="a in ezfy.accItems(5)" :key="'st' + t.tech_id + '_' + a.cfg_id" href="javascript:;"
                 @click="ezfy.doSpeedTech(a)">[加速{{ ezfy.accLabel(a) }}]</a>
              <!-- ★ 2026-09-27 百分比科技加速(item_type=26)：剩余时间减 30%/60%/80% -->
              <a v-for="a in ezfy.accItems(26)" :key="'stp' + t.tech_id + '_' + a.cfg_id" href="javascript:;"
                 @click="ezfy.doSpeedTech(a)">[加速{{ ezfy.accLabel(a) }}]</a>
              <span class="gray" v-if="!ezfy.accItems(5).length && !ezfy.accItems(26).length">(无科技加速道具)</span>
              <a href="javascript:;" @click="ezfy.doCancelTech(t)">[取消]</a></span>
            <span v-else-if="t.level < t.max_level">
              <a href="javascript:;" @click="ezfy.openTechPre(t)">[研究{{ t.level + 1 }}级]</a>
            </span>
            <span v-else class="gray">[已满级]</span>
            <br/>
            {{ t.effect }}<br/>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'techpre'">
        <div class="panel" v-if="ezfy.techSel">
          <div class="panel-title">研究「{{ ezfy.techSel.name }}」{{ ezfy.techSel.level + 1 }}级</div>
          <div class="old-line">当前等级：{{ ezfy.techSel.level }}/{{ ezfy.techSel.max_level }}级</div>
          <div class="old-line">效果：{{ ezfy.techSel.effect }}</div>
          <div class="old-line">前提：科研中心{{ ezfy.techSel.academy_need }}级
            <span class="gray">(本城 {{ ezfy.techSel.academy }} 级)</span></div>
          <div class="old-line">
            所需资源：{{ ezfy.resNames.food }}{{ ezfy.techSel.next_cost.food }} {{ ezfy.resNames.steel }}{{ ezfy.techSel.next_cost.steel }}
            {{ ezfy.resNames.oil }}{{ ezfy.techSel.next_cost.oil }} {{ ezfy.resNames.rare }}{{ ezfy.techSel.next_cost.rare }}
            {{ ezfy.resNames.gold }}{{ ezfy.techSel.next_cost.gold }}
          </div>
          <div class="old-line">耗时：{{ Math.ceil(ezfy.techSel.next_time / 60) }}分钟</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.doTechPre()">[开始研究]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('techs')">[返回科技列表]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先选择科技 <a href="javascript:;" @click="ezfy.go('techs')">[科技列表]</a></div>
        </div>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化：科技中心页（cur=techs/techpre）模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyTech',
  inject: ['ezfy']
}
</script>
