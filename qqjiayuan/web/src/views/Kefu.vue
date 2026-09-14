<template>
  <div>
    <div class="bar">【客服中心】</div>
    <div class="module-content">
      ----------<br>
      &gt;&gt;<a href="javascript:;" @click="$router.push('/guestbook')">留言客服</a><br>
      &gt;&gt;<a href="javascript:;" @click="$router.push('/kefu/help')">新手帮助</a><br>
      &gt;&gt;<a href="javascript:;" @click="$router.push('/kefu/disclaimer')">免责声明</a><br>
      &gt;&gt;<a href="javascript:;" @click="$router.push('/kefu/terms')">注册声明</a><br>
      &gt;&gt;<a href="javascript:;" @click="$router.push('/find')">找回资料</a><br>
      ----------<br>
      <br>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Kefu'
}
</script>
