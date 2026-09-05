<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的百宝箱</div>
    <div class="module-title">用户中心|<a href="javascript:;" @click="$router.push('/security')">安全中心</a>|<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>|<a href="javascript:;" @click="$router.push('/rank')">家园排行</a></div>
    <div class="module-content"><span class="txt-fade">当前G币：<b style="color:#e05a00">{{ coins }}</b></span></div>

    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/profile')">编辑资料</a>.<a href="javascript:;" @click="$router.push('/profile')">我的头像</a>.<a href="javascript:;" @click="$router.push('/profile')">个性昵称</a><br>
      <a href="javascript:;" @click="tip('城市设置')">城市设置</a>.<a href="javascript:;" @click="tip('我的证件')">我的证件</a>.<a href="javascript:;" @click="tip('个性设置')">个性设置</a><br>
      <a href="javascript:;" @click="tip('推荐有礼')">推荐有礼</a>.<a href="javascript:;" @click="tip('图标开关')">图标开关</a><br>
    </div>

    <div class="module-title">【功能导航】</div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/noble')">超Q蓝钻</a>.<a href="javascript:;" @click="$router.push('/shop')">道具商城</a>.<a href="javascript:;" @click="tip('活动荣誉')">活动荣誉</a>.<a href="javascript:;" @click="tip('礼物')">礼物</a><br>
      <a href="javascript:;" @click="$router.push('/profile')">收藏夹</a>.<a href="javascript:;" @click="tip('草稿箱')">草稿箱</a>.<a href="javascript:;" @click="tip('社区拍卖')">社区拍卖</a><br>
      <a href="javascript:;" @click="tip('剪贴板')">剪贴板</a>.<a href="javascript:;" @click="$router.push('/inbox')">信箱</a>.<a href="javascript:;" @click="$router.push('/space/'+user.id)">我的相册</a>.<a href="javascript:;" @click="$router.push('/play')">幸运抽奖</a>.<a href="javascript:;" @click="tip('专属勋章')">专属勋章</a><br>
    </div>

    <div class="module-title">【管理功能】</div>
    <div class="module-content">
      <a href="javascript:;" @click="tip('管理员薪酬')">管理员薪酬</a> <a href="javascript:;" @click="tip('社区服务厅')">社区服务厅</a><br>
      <a href="javascript:;" @click="tip('管理员考勤')">管理员考勤</a> <br>
      <a href="javascript:;" @click="tip('刷新权限')">刷新权限</a> <a href="javascript:;" @click="tip('刷新帽子')">刷新帽子</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Box',
  data () { return { coins: 0 } },
  computed: {
    user () { return this.$store.state.user || {} }
  },
  mounted () {
    api.get('/auth/me').then(r => { if (r.code === 0) this.coins = r.data.coins || 0 }).catch(() => {})
  },
  methods: {
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('百宝箱·' + title)) }
  }
}
</script>
