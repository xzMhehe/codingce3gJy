<template>
  <div>
    <!-- ============ 首页-老布局(homeLayout===2，保留原样) ============ -->
    <div class="home-body">
      <!-- ★ 只有【置顶】公告展示到首页外边；普通公告进「公告」页看。
           外面包一层 .ezfy-notices 只为统一它与上下两行的间距(见样式表注释)。 -->
      <div class="ezfy-notices" v-if="ezfy.topNotices.length">
        <div class="old-line" v-for="n in ezfy.topNotices" :key="'n' + n.id">
          <img class="logo-title" src="/static/ezfy/notice.gif" alt="."/>
          <a class="red" href="javascript:;" @click="ezfy.openNotice(n)">{{ n.title }}</a>
        </div>
      </div>

      <div class="old-line">
        <span class="city-name">{{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }})</span>
        <a href="javascript:;" @click="ezfy.go('cities')">切换城市</a>
      </div>
      <div class="old-line">
        <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>军团</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#2F5D8A"/><path d="M10 3.3 L15.4 5.2 V10.4 C15.4 13.5 13.2 15.6 10 16.5 C6.8 15.6 4.6 13.5 4.6 10.4 V5.2 Z" fill="#fff"/><path d="M10 7.4 L11.2 9.6 L13.6 9.8 L12 11.3 L12.5 13.6 L10 12.3 L7.5 13.6 L8 11.3 L6.4 9.8 L8.8 9.6 Z" fill="#2F5D8A"/></svg>
        军团:
        <a href="javascript:;" @click="ezfy.go('corps')" v-if="!ezfy.myCorps">加入军团</a>
        <a href="javascript:;" @click="ezfy.go('corps')" v-else>[{{ ezfy.myCorps.name }}]</a>
      </div>
      <div class="old-line">声望: {{ ezfy.profile.prestige }}</div>
      <div class="old-line"><span v-html="ezfy.rankIcon(ezfy.myRankId)"></span><a href="javascript:;" @click="ezfy.go('rank')">军衔</a>: {{ ezfy.rankName }}</div>
      <div class="old-line">每日签到: <a href="javascript:;" @click="ezfy.go('welfare')">{{ ezfy.welfare.signed_today ? '已签到' : '签到' }}</a></div>

      <div class="old-line home-nav2">
        <a href="javascript:;" @click="ezfy.go('builds')">资源</a>.
        <a href="javascript:;" @click="ezfy.go('acade')">军官</a>.
        <a href="javascript:;" @click="ezfy.go('troops')">军队</a>.
        <a href="javascript:;" @click="ezfy.go('techs')">科技</a>.
        <a href="javascript:;" @click="ezfy.go('defence')">城防</a>.
        <a href="javascript:;" @click="ezfy.go('info')">统帅</a>
      </div>
      <div class="old-line">现有资源/产量:
        <a href="javascript:;" @click="ezfy.go('exchange')">购买</a><span class="home-gap"></span><a href="javascript:;" @click="ezfy.go('mall')">增产</a>
      </div>
      <div class="old-line">
        <span :title="ezfy.resNames.gold"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-linecap="round" stroke-linejoin="round" fill="none"><path d="M6.4 5.8 L10 10.3 L13.6 5.8" stroke-width="1.6"/><path d="M10 6 V14.2" stroke-width="1.6"/><path d="M7.6 8.9 H12.4" stroke-width="1.4"/><path d="M7.6 11.7 H12.4" stroke-width="1.4"/></g></svg></span>
        <a href="javascript:;" @click="ezfy.go('res/gold')">{{ ezfy.resNames.gold }}:</a><span :title="'现有 ' + ezfy.fmtN(ezfy.city.gold)">{{ ezfy.fmtProd(ezfy.city.gold) }}</span>/<span :title="ezfy.resShort.gold + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd.gold) }}</span>
      </div>
      <div class="old-line">
        <span :title="ezfy.resNames.food"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-width="1.5" stroke-linecap="round" fill="none"><path d="M7.2 6.2 C6.7 5.4 7.3 4.1 7.9 3.5"/><path d="M12.8 6.2 C13.3 5.4 12.7 4.1 12.1 3.5"/></g><path d="M5 8.6 H15 C15 8.6 14.7 12.2 13.3 13.7 C12.2 14.8 10.9 15.4 10 15.4 C9.1 15.4 7.8 14.8 6.7 13.7 C5.3 12.2 5 8.6 5 8.6 Z" fill="#FFD700"/></svg></span>
        <a href="javascript:;" @click="ezfy.go('res/food')">{{ ezfy.resNames.food }}:</a><span :title="'现有 ' + ezfy.fmtN(ezfy.city.food)">{{ ezfy.fmtProd(ezfy.city.food) }}</span>/<span :title="ezfy.resShort.food + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd.food) }}</span>
      </div>
      <div class="old-line">
        <span :title="ezfy.resNames.steel"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M4.2 8.4 L6.6 15.6 H13.4 L15.8 8.4 Z" fill="#FFD700"/><path d="M5.6 10.2 H14.4" stroke="#C9AFF0" stroke-width="1.3" stroke-linecap="round"/></svg></span>
        <a href="javascript:;" @click="ezfy.go('res/steel')">{{ ezfy.resNames.steel }}:</a><span :title="'现有 ' + ezfy.fmtN(ezfy.city.steel)">{{ ezfy.fmtProd(ezfy.city.steel) }}</span>/<span :title="ezfy.resShort.steel + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd.steel) }}</span>
      </div>
      <div class="old-line">
        <span :title="ezfy.resNames.oil"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 3.6 C10 3.6 6.1 8.1 6.1 11.2 C6.1 13.5 7.8 15.3 10 15.3 C12.2 15.3 13.9 13.5 13.9 11.2 C13.9 8.1 10 3.6 10 3.6 Z" fill="#FFD700"/><circle cx="8.6" cy="11.4" r="0.9" fill="#C9AFF0"/></svg></span>
        <a href="javascript:;" @click="ezfy.go('res/oil')">{{ ezfy.resNames.oil }}:</a><span :title="'现有 ' + ezfy.fmtN(ezfy.city.oil)">{{ ezfy.fmtProd(ezfy.city.oil) }}</span>/<span :title="ezfy.resShort.oil + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd.oil) }}</span>
      </div>
      <div class="old-line">
        <span :title="ezfy.resNames.rare"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M5.8 6.4 H14.2 L11.7 9.2 L10 15.6 L8.3 9.2 Z" fill="#FFD700"/><path d="M5.8 6.4 H10 L8.3 9.2 Z" fill="#C9AFF0"/></svg></span>
        <a href="javascript:;" @click="ezfy.go('res/rare')">{{ ezfy.resNames.rare }}:</a><span :title="'现有 ' + ezfy.fmtN(ezfy.city.rare)">{{ ezfy.fmtProd(ezfy.city.rare) }}</span>/<span :title="ezfy.resShort.rare + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd.rare) }}</span>
      </div>
      <div class="old-line">
        <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>人口</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><circle cx="7" cy="7.4" r="1.6" fill="#FFD700"/><path d="M4.7 14.6 C4.7 12.7 5.7 11.5 7 11.5 C8.3 11.5 9.3 12.7 9.3 14.6 Z" fill="#FFD700"/><circle cx="13" cy="6.6" r="1.5" fill="#FFD700"/><path d="M10.9 14.6 C10.9 12.9 11.9 11.9 13 11.9 C14.1 11.9 15.1 12.9 15.1 14.6 Z" fill="#FFD700"/></svg>人口/空闲:{{ ezfy.city.pop }}/{{ ezfy.freePop }}
        <a href="javascript:;" @click="ezfy.go('convene')">召集</a>
      </div>
      <div class="old-line">
        <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>民心</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 15.5 C5.3 12.6 4.1 9.6 4.1 7.6 C4.1 5.9 5.4 4.7 7 4.7 C8.1 4.7 9.2 5.3 10 6.3 C10.8 5.3 11.9 4.7 13 4.7 C14.6 4.7 15.9 5.9 15.9 7.6 C15.9 9.6 14.7 12.6 10 15.5 Z" fill="#FFD700"/></svg>民心/民怨:{{ ezfy.city.feelings }}/{{ ezfy.city.grievance }}
        <a href="javascript:;" @click="ezfy.go('placate')">安抚</a>
      </div>
      <div class="old-line">
        <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>税率</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M14.6 5.4 L5.4 14.6" stroke="#FFD700" stroke-width="1.4" stroke-linecap="round"/><circle cx="6.9" cy="5.9" r="1.8" fill="#FFD700"/><circle cx="13.1" cy="14.1" r="1.8" fill="#FFD700"/></svg>
        <a href="javascript:;" @click="ezfy.go('taxset')">税率:</a>{{ ezfy.city.tax_rate }}%
      </div>
      <div class="old-line">
        <a href="javascript:;" @click="ezfy.go('buildm')">军事区</a><span class="home-gap"></span><a href="javascript:;" @click="ezfy.openBuildPre('m')">建造</a>
      </div>
      <div class="old-line">
        <a href="javascript:;" @click="ezfy.go('builds')">资源区</a><span class="home-gap"></span><a href="javascript:;" @click="ezfy.openBuildPre('s')">建造</a>
      </div>
      <div class="old-line">
        训练军队
        <a href="javascript:;" @click="ezfy.go('troop')">[造兵]</a><span class="home-gap"></span><a href="javascript:;" @click="ezfy.go('defence')">[建防]</a>
      </div>
      <div class="old-line">
        前往 <a href="javascript:;" @click="ezfy.go('map')">地图</a> 出征
      </div>
      <div class="old-line">
        <a href="javascript:;" @click="ezfy.go('citystatus')">城市状态</a><span class="home-gap"></span><a href="javascript:;" @click="ezfy.go('wilds')">附属野地</a>
      </div>
      <div class="old-line">【世界聊天】<a href="javascript:;" @click="ezfy.go('chat')">[进入]</a></div>
      <!-- [世界] 安珞：11111 / [军团] / [私聊] / [系统]; 昵称用实时昵称+个性颜色 -->
      <div class="old-line" v-for="ch in ezfy.homeChats" :key="'wc' + ch.key">
        [<span class="orange">{{ ch.tag }}</span>]
        <!-- ★ 2026-09-28 bug修复: 冒号+内容从 v-if/v-else 判断结构移出外层,
             否则 Vue 会把紧跟 v-if 的「：{{content}}」静文并进 else 分支,
             导致有 user_id 的玩家消息只出昵称不出内容。现在两层分支都只负责出「昵称：」/「系统：」头, 内容恒在 -->
        <span v-if="ch.user_id">
          <a href="javascript:;" @click="ezfy.openPlayer(ch.user_id)"><span
             v-for="(c, ci) in ezfy.nickChars(ch.user_name)" :key="'nc' + ci"
             :style="ezfy.nickColorAt(ch.color, ci)">{{ c }}</span></a>：
        </span>
        <span v-else>{{ ch.tag == '系统' ? '系统：' : ch.tag + '：' }}</span>
        {{ ch.content }}
      </div>
      <div class="old-line gray" v-if="!ezfy.homeChats.length">(暂无消息)</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'EzfyHomeOld',
  inject: ['ezfy']
}
</script>