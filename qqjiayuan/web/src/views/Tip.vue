<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/nav')">导航</a>&gt;{{ title }}</div>
    <div class="module-title">{{ title }}</div>
    <div class="module-content plist">
      <div class="row00">该功能正在建设中，敬请期待～</div>
      <div class="row00"><a href="javascript:;" @click="$router.back()">返回上一页</a>　<a href="javascript:;" @click="$router.push('/')">回到广场</a></div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Tip',
  computed: {
    title () { return this.$route.query.title || '建设中' }
  }
}
</script>
