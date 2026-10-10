<template>
  <div>
    <!-- ============ 首页-新布局（战争主题简约版，一屏放全） ============ -->
    <div class="war-home">
      <!-- 置顶公告 -->
      <div class="ezfy-notices" v-if="ezfy.topNotices.length">
        <div class="old-line" v-for="n in ezfy.topNotices" :key="'wn' + n.id">
          <img class="logo-title" src="/static/ezfy/notice.gif" alt="."/>
          <a class="red" href="javascript:;" @click="ezfy.openNotice(n)">{{ n.title }}</a>
        </div>
      </div>

      <!-- 城市 + 军衔/声望/军团/签到 -->
      <div class="war-head">
        <div class="war-city">
          <span class="city-name">{{ ezfy.city.name }}({{ ezfy.city.x }},{{ ezfy.city.y }})</span>
          <a href="javascript:;" @click="ezfy.go('cities')">切换城市</a>
        </div>
        <div class="war-stats">
          <span class="war-stat" title="军衔"><span v-html="ezfy.rankIcon(ezfy.myRankId)"></span>{{ ezfy.rankName }}</span>
          <span class="war-stat" title="声望">声望{{ ezfy.profile.prestige }}</span>
          <span class="war-stat" title="军团">
            <a href="javascript:;" @click="ezfy.go('corps')" v-if="!ezfy.myCorps">加入军团</a>
            <a href="javascript:;" @click="ezfy.go('corps')" v-else>{{ ezfy.myCorps.name }}</a>
          </span>
          <span class="war-stat" title="每日签到">
            <a href="javascript:;" @click="ezfy.go('welfare')">{{ ezfy.welfare.signed_today ? '已签到' : '签到' }}</a>
          </span>
        </div>
      </div>

      <!-- 资源 5 列网格（图标 + 现有/每小时产量） -->
      <div class="war-res">
        <div class="war-res-cell" v-for="rk in ['gold', 'food', 'steel', 'oil', 'rare']" :key="'rk' + rk">
          <span class="war-res-ico" v-html="ezfy.resIcon(rk)"></span>
          <a class="war-res-name" href="javascript:;" @click="ezfy.go('res/' + rk)">{{ ezfy.resNames[rk] }}</a>
          <div class="war-res-val">
            <span :title="'现有 ' + ezfy.fmtN(ezfy.city[rk])">{{ ezfy.fmtProd(ezfy.city[rk]) }}</span>
            <i>/</i>
            <span :title="ezfy.resShort[rk] + '每小时产量'">{{ ezfy.fmtProd(ezfy.resProd[rk]) }}</span>
          </div>
        </div>
        <div class="war-res-ops">
          <a href="javascript:;" @click="ezfy.go('exchange')">购买</a>
          <a href="javascript:;" @click="ezfy.go('mall')">增产</a>
        </div>
      </div>

      <!-- 人口/民心/税率 -->
      <div class="war-row2">
        <span class="war-cell2" :title="'人口/空闲'">
          <svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><circle cx="7" cy="7.4" r="1.6" fill="#FFD700"/><path d="M4.7 14.6 C4.7 12.7 5.7 11.5 7 11.5 C8.3 11.5 9.3 12.7 9.3 14.6 Z" fill="#FFD700"/><circle cx="13" cy="6.6" r="1.5" fill="#FFD700"/><path d="M10.9 14.6 C10.9 12.9 11.9 11.9 13 11.9 C14.1 11.9 15.1 12.9 15.1 14.6 Z" fill="#FFD700"/></svg>
          {{ ezfy.city.pop }}/{{ ezfy.freePop }}
          <a href="javascript:;" @click="ezfy.go('convene')">[召集]</a>
        </span>
        <span class="war-cell2" :title="'民心/民怨'">
          <svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 15.5 C5.3 12.6 4.1 9.6 4.1 7.6 C4.1 5.9 5.4 4.7 7 4.7 C8.1 4.7 9.2 5.3 10 6.3 C10.8 5.3 11.9 4.7 13 4.7 C14.6 4.7 15.9 5.9 15.9 7.6 C15.9 9.6 14.7 12.6 10 15.5 Z" fill="#FFD700"/></svg>
          {{ ezfy.city.feelings }}/{{ ezfy.city.grievance }}
          <a href="javascript:;" @click="ezfy.go('placate')">[安抚]</a>
        </span>
        <span class="war-cell2" title="税率">
          <svg class="ezfy-ico" viewBox="0 0 20 20"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M14.6 5.4 L5.4 14.6" stroke="#FFD700" stroke-width="1.4" stroke-linecap="round"/><circle cx="6.9" cy="5.9" r="1.8" fill="#FFD700"/><circle cx="13.1" cy="14.1" r="1.8" fill="#FFD700"/></svg>
          <a href="javascript:;" @click="ezfy.go('taxset')">{{ ezfy.city.tax_rate }}%</a>
        </span>
      </div>

      <!-- 功能入口（去重后全部功能） -->
      <div class="war-btns">
        <a class="war-btn" href="javascript:;" @click="ezfy.go('builds')">资源</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('acade')">军官</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('troops')">军队</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('techs')">科技</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('defence')">城防</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('info')">统帅</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('buildm')">军事区</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('troop')">造兵</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('map')">地图</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('citystatus')">城市状态</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('wilds')">野地</a>
        <a class="war-btn" href="javascript:;" @click="ezfy.go('chat')">聊天</a>
      </div>

      <!-- 世界聊天 -->
      <div class="war-chat">
        <div class="old-line">【世界聊天】<a href="javascript:;" @click="ezfy.go('chat')">[进入]</a></div>
        <div class="old-line" v-for="ch in ezfy.homeChats" :key="'wc' + ch.key">
          [<span class="orange">{{ ch.tag }}</span>]
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
  </div>
</template>

<script>
export default {
  name: 'EzfyHomeNew',
  inject: ['ezfy']
}
</script>