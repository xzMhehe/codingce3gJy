<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的家园</a>&gt;家园等级<br></div>

    <div class="module-title">【家园等级介绍】</div>
    <div class="module-content">
      用户每天首次登录家园页面时可积累活跃天。积累方法如下：<br>
      [普通用户]<br>
      每天获1个活跃天.<br>
      连续登录（如：昨天有过登录,今天又登录）额外获得0.2天,即获得1.2个活跃天.<br>
      [超级QQ用户升级加速]<br>
      根据不同的VIP等级，超级QQ用户每天最高可获得2.0活跃天。如下表：<br>
      <table class="grid">
        <tr><th>超级QQ等级</th><th>普通登录</th><th>连续登录</th></tr>
        <tr><td>黄金1级</td><td>+1.1天</td><td>+1.3天</td></tr>
        <tr><td>黄金2级</td><td>+1.2天</td><td>+1.4天</td></tr>
        <tr><td>黄金3级</td><td>+1.4天</td><td>+1.6天</td></tr>
        <tr><td>黄金4级</td><td>+1.5天</td><td>+1.7天</td></tr>
        <tr><td>黄金5级</td><td>+1.7天</td><td>+1.9天</td></tr>
        <tr><td>黄金6级</td><td>+1.8天</td><td>+2.0天</td></tr>
        <tr><td>黄金7级</td><td>+1.9天</td><td>+2.1天</td></tr>
      </table>
    </div>

    <div class="module-content plist">
      <div class="row00">我的家园等级：<b style="color:#e05a00">Lv.{{ data.level || 1 }}</b> <img v-if="data.icon" :src="data.icon" alt="图标"></div>
      <div class="row00">已积累活跃天数：{{ data.active_days || 0 }}<template v-if="data.next_days">　下一级需 {{ data.next_days }} 天</template></div>
    </div>

    <div class="module-title">【家园等级列表】</div>
    <div class="module-content">|等级|等级图标（男/女）|升级所需活跃天数|</div>
    <div class="module-content" v-for="l in data.levels" :key="l.Lv" :style="l.Lv === (data.level || 1) ? 'background:#FFF3D6' : ''">
      |{{ l.Lv }}|<img :src="'/static/picture/home_1_' + l.Lv + '.gif'" alt="男">|<img :src="'/static/picture/home_2_' + l.Lv + '.gif'" alt="女">|{{ l.Days }}| <template v-if="l.Lv === (data.level || 1)">← 我的等级</template>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'HomeLevel',
  data () { return { data: {} } },
  mounted () { api.get('/home-level').then(r => { if (r.code === 0) this.data = r.data }) },
  methods: {
    pad (n) { return n < 10 ? '0' + n : '' + n }
  }
}
</script>
