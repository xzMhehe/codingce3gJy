<template>
  <div>
    <div class="bar">【站内导航】</div>

    <!-- 资讯 -->
    <div class="module-title">【<a href="javascript:;" @click="tip('资讯')">资讯</a>】<a href="javascript:;" @click="tip('新闻')">新闻</a>.<a href="javascript:;" @click="tip('财经')">财经</a>.<a href="javascript:;" @click="tip('体育')">体育</a>.<a href="javascript:;" @click="tip('娱乐')">娱乐</a>.<a href="javascript:;" @click="tip('科技')">科技</a></div>
    <div class="module-content">
      <a href="javascript:;" @click="tip('资讯')">资讯频道建设中，敬请期待</a><br>
    </div>

    <!-- 社区 -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/channel/1')">社区</a>】<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/families')">家族</a>.<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>.<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>.<a href="javascript:;" @click="$router.push('/marriage')">婚恋</a></div>
    <div class="module-content">
      <template v-for="(t,i) in hotThreads">
        <span :key="'h'+i"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count || 0 }}阅)<br></span>
      </template>
    </div>

    <!-- 书城 -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/book')">书城</a>】<a href="javascript:;" @click="$router.push('/book?category=武侠')">武侠</a>.<a href="javascript:;" @click="$router.push('/book?category=言情')">言情</a>.<a href="javascript:;" @click="$router.push('/shelf')">书架</a></div>
    <div class="module-content">
      <template v-for="(b,i) in hotBooks">
        <span :key="'b'+i">《<a href="javascript:;" @click="$router.push('/book/'+b.id)">{{ b.title }}</a>》<span class="txt-fade">{{ b.author }}</span>({{ b.category }})<br></span>
      </template>
    </div>

    <!-- 商城 -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/shop')">商城</a>】<a href="javascript:;" @click="tip('拍卖')">拍卖</a>.<a href="javascript:;" @click="$router.push('/medals')">勋章</a></div>
    <div class="module-content"><a href="javascript:;" @click="$router.push('/shop')">道具商城</a> / <a href="javascript:;" @click="$router.push('/money-shop')">货币商店</a> / <a href="javascript:;" @click="$router.push('/store')">店铺街</a> / <a href="javascript:;" @click="$router.push('/book')">书城</a> / <a href="javascript:;" @click="$router.push('/shelf')">书架</a> / <a href="javascript:;" @click="$router.push('/medals')">勋章大全</a> / 拍卖建设中<br></div>

    <!-- 游戏大厅 -->
    <div class="module-title">【社区游戏大厅】</div>
    <div class="plist">
      <div class="row00">
        <a href="javascript:;" @click="$router.push('/games')">魔法花园</a><br>
        <a href="javascript:;" @click="$router.push('/games')">精武堂</a><br>
        <a href="javascript:;" @click="$router.push('/games')">阳光牧场</a><br>
        <a href="javascript:;" @click="$router.push('/games')">爱宠国</a><br>
        <a href="javascript:;" @click="$router.push('/games')">抢车位</a><br>
        <a href="javascript:;" @click="$router.push('/games')">开心农场</a><br>
        <a href="javascript:;" @click="$router.push('/play')">打工</a><br>
      </div>
    </div>

    <!-- 便民服务 -->
    <div class="module-title">【便民服务】<a href="javascript:;" @click="tip('天气')">天气</a></div>
    <div class="module-content">
      <a href="javascript:;" @click="tip('天气')">天气</a>.<a href="javascript:;" @click="tip('手机')">手机</a>.<a href="javascript:;" @click="tip('IP')">IP</a>.<a href="javascript:;" @click="tip('翻译')">翻译</a><br>
    </div>

    <!-- 社区服务 -->
    <div class="module-title">【社区服务】</div>
    <div class="module-content">
      <a href="javascript:;" @click="tip('客服')">客服</a>.<a href="javascript:;" @click="tip('关于')">关于</a>.<a href="javascript:;" @click="tip('招商')">招商</a><br>
      <a href="javascript:;" @click="$router.push('/sign')">每日签到</a>.<a href="javascript:;" @click="$router.push('/play')">社区银行</a>.<a href="javascript:;" @click="$router.push('/play')">打工</a>.<a href="javascript:;" @click="$router.push('/search')">搜搜</a><br>
    </div>

    <div class="login-tips">【管理专区】<a href="javascript:;" @click="window.open('http://'+location.host+'/admin-ui/')">管理系统</a>　移动版 WAP 风格 · 家园社区 出品</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Nav',
  data () { return { hotThreads: [], hotBooks: [] } },
  mounted () {
    api.get('/plaza').then(r => {
      if (r.code === 0 && r.data) this.hotThreads = (r.data.fine_threads || []).slice(0, 3)
    })
    api.get('/books').then(r => {
      if (r.code === 0 && r.data) this.hotBooks = (r.data.recommend || []).slice(0, 3)
    })
  },
  methods: {
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent(title)) }
  }
}
</script>
