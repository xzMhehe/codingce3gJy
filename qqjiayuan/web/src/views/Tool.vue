<template>
  <div>
    <!-- 便民中心（复刻诺哈三代 wap/tool/index.asp） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;工具<br></div>

    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/tool/name')">姓名评分</a> <a href="javascript:;" @click="$router.push('/tool/translate')">英汉互译</a><br>
      <a href="javascript:;" @click="$router.push('/tool/weather')">天气预报</a> <a href="javascript:;" @click="$router.push('/tool/phone')">手机归属</a><br>
      <a href="javascript:;" @click="$router.push('/tool/ip')">ＩＰ查询</a> <a href="javascript:;" @click="$router.push('/tool/code')">编码转换</a><br>
      <a href="javascript:;" @click="$router.push('/tool/client')">客户信息</a><br>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;工具<br></div>
  </div>
</template>

<script>
export default { name: 'Tool' }
</script>
