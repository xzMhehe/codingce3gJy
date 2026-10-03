<template>
  <div>
    <template v-if="ezfy.cur === 'scheme'">
        <div class="panel" v-if="ezfy.schemeOrder">
          <div class="panel-title">计谋</div>
          <div class="old-line">
            【信号弹】持有：
            <b :class="ezfy.schemeData.bullet_have > 0 ? 'green' : 'red'">{{ ezfy.schemeData.bullet_have }}</b>
            <a href="javascript:;" @click="ezfy.go('mall')">[去商城购买]</a>
          </div>
          <div class="old-line gray">
            神兵天降：去程剩余时间减少80%。战略转移：回程减少360分钟。每次消耗7枚信号弹，每支部队两种计谋各限一次。
          </div>
          <hr/>
          部队：{{ ezfy.schemeOrder.type_name }}（{{ ezfy.schemeOrder.target_name }}({{ ezfy.schemeOrder.target_x }},{{ ezfy.schemeOrder.target_y }})）<br/>
          状态：{{ ezfy.schemeOrder.status_name }}<br/>
          <!-- 出征中 → 去程用神兵天降；返回中 → 回程用战略转移 -->
          <template v-if="ezfy.schemeOrder.status === 0">
            <div class="old-line">
              ({{ ezfy.schemeOrder.target_x }},{{ ezfy.schemeOrder.target_y }}) 去程
              <a v-if="ezfy.schemeOrder.scheme_fast === 0" class="green"
                 href="javascript:;" @click="ezfy.doMarchScheme(ezfy.schemeOrder, 13)">[神兵天降]</a>
              <span v-else class="gray">[神兵天降·已用]</span>
            </div>
          </template>
          <template v-else-if="ezfy.schemeOrder.status === 2">
            <div class="old-line">
              ({{ ezfy.schemeOrder.target_x }},{{ ezfy.schemeOrder.target_y }}) 回程
              <a v-if="ezfy.schemeOrder.scheme_back === 0" class="green"
                 href="javascript:;" @click="ezfy.doMarchScheme(ezfy.schemeOrder, 14)">[战略转移]</a>
              <span v-else class="gray">[战略转移·已用]</span>
            </div>
          </template>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('reports')">[返回军情-军队动态]</a>
            <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">没有可使用的计谋部队</div>
          <a href="javascript:;" @click="ezfy.go('reports')">[返回军情-军队动态]</a>
        </div>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化拆分：计谋页模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyScheme',
  inject: ['ezfy']
}
</script>
