<template>
  <div>
    <div class="bar">【站内导航】</div>
    <template v-for="g in groups">
      <div class="module-title" :key="'t'+g.title">{{ g.title }}</div>
      <div class="module-content" :key="'c'+g.title">
        [{{ g.title.slice(2, -2) }}]
        <template v-for="(l, i) in g.links">
          <a :key="'a'+g.title+i" href="javascript:;" @click="l.go && l.go()">{{ l.name }}</a><span :key="'s'+g.title+i" v-if="i < g.links.length-1"> · </span>
        </template>
      </div>
    </template>
    <div class="login-tips">移动版 WAP 风格 · 适配手机/电脑 · 3GQQ家园社区 出品</div>
  </div>
</template>

<script>
export default {
  name: 'Nav',
  data () {
    return {
      groups: [
        { title: '【互动社区】', links: [
          { name: '广场', go: () => this.$router.push('/') },
          { name: '论坛', go: () => this.$router.push('/channel/1') },
          { name: '家族', go: () => this.$router.push('/channel/2') },
          { name: '同城', go: () => this.$router.push('/channel/3') },
          { name: '聊天', go: () => this.$router.push('/chat') },
          { name: '好友', go: () => this.$router.push('/friends') },
          { name: '私信', go: () => this.$router.push('/messages') },
          { name: '签到', go: () => this.$router.push('/sign') }
        ] },
        { title: '【社区服务】', links: [
          { name: '客服中心', go: () => this.$router.push('/channel/4') },
          { name: '意见建议', go: () => this.$router.push('/channel/4') },
          { name: '通知', go: () => this.$router.push('/notices') },
          { name: '资料设置', go: () => this.$router.push('/profile') },
          { name: '找回资料', go: () => this.$router.push('/find') }
        ] },
        { title: '【游戏中心】', links: [
          { name: '游戏大厅', go: () => this.$router.push('/games') },
          { name: '敬请期待', go: () => this.$router.push('/games') }
        ] },
        { title: '【管理专区】', links: [
          { name: '管理系统', go: () => window.open('http://' + location.host + '/admin-ui/') },
          { name: '社区公示', go: () => this.$router.push('/board/18') }
        ] }
      ]
    }
  }
}
</script>
