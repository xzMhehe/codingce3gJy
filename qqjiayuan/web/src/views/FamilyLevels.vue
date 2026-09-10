<template>
  <div>
    <!-- 参考：家园演示源码/家族参考/家人等级列表.htm -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;家人等级信息<br></div>

    <div class="module-title">【家人等级列表】</div>
    <div class="module-content">
      等级　职称　贡献值<br>
      <template v-for="lv in levels">
        <div :key="'lv'+lv.no" class="row00">
          {{ ('0' + lv.no).slice(-2) }}　<img :src="'/static/picture/family_lv/' + lv.no + '.gif'" width="16" height="16" alt="家族">　{{ lv.title }}　{{ lv.req }}<br>
        </div>
      </template>
      <div class="row00">
        31及以上　<img src="/static/picture/family_lv/31.gif" width="16" height="16" alt="家族">　枭雄家人<br>
      </div>
    </div>

    <div class="module-content">
      <a href="javascript:;" @click="$router.go(-1)">返回上一页</a><br>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;家人等级信息<br></div>
  </div>
</template>

<script>
export default {
  name: 'FamilyLevels',
  data () {
    return {
      levels: [
        { no: 1, title: '见习家人', req: 0 },
        { no: 2, title: '初级家人', req: 10 },
        { no: 3, title: '活跃家人', req: 90 },
        { no: 4, title: '活跃家人', req: 200 },
        { no: 5, title: '中级家人', req: 350 },
        { no: 6, title: '中级家人', req: 540 },
        { no: 7, title: '有爱家人', req: 770 },
        { no: 8, title: '有爱家人', req: 1040 },
        { no: 9, title: '高级家人', req: 1350 },
        { no: 10, title: '高级家人', req: 1700 },
        { no: 11, title: '荣誉家人', req: 2090 },
        { no: 12, title: '荣誉家人', req: 2520 },
        { no: 13, title: '勤劳家人', req: 2990 },
        { no: 14, title: '勤劳家人', req: 3500 },
        { no: 15, title: '精英家人', req: 4050 },
        { no: 16, title: '精英家人', req: 4640 },
        { no: 17, title: '智慧家人', req: 5270 },
        { no: 18, title: '智慧家人', req: 5940 },
        { no: 19, title: '炫耀家人', req: 6650 },
        { no: 20, title: '炫耀家人', req: 7400 },
        { no: 21, title: '金鼎家人', req: 8190 },
        { no: 22, title: '金鼎家人', req: 9020 },
        { no: 23, title: '金鼎家人', req: 9890 },
        { no: 24, title: '美丽家人', req: 10800 },
        { no: 25, title: '美丽家人', req: 11750 },
        { no: 26, title: '美丽家人', req: 12740 },
        { no: 27, title: '忠义家人', req: 13770 },
        { no: 28, title: '忠义家人', req: 14840 },
        { no: 29, title: '忠义家人', req: 15950 },
        { no: 30, title: '忠义家人', req: 17100 }
      ]
    }
  }
}
</script>
