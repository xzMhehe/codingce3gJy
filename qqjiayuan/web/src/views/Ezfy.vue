<template>
  <div class="ezfy-page">
    <div class="home-wrap">
      <div class="title-bar">二战征途-【1区】红色警戒</div>

      <!-- 顶部导航(每页都有) -->
      <div class="top-nav">
        <a href="javascript:;" @click="go('chat')">聊天</a>
        <a href="javascript:;" @click="openPm()">邮箱</a>
        <a href="javascript:;" @click="go('reports')">军情</a>
        <a href="javascript:;" @click="go('tasks')">任务</a>
        <a href="javascript:;" @click="go('friends')">好友</a>
        <a href="javascript:;" @click="go('home')">首页</a>
      </div>

      <!-- 页面内操作结果（代替 alert 弹窗；原版本来就没有弹窗交互）
           ★ 用户反馈「提示都堆在页面顶部，看不出点了哪」→ 改为浮现在最近一次点击的附近，
           不再占一整行置顶。多条目向下轻微错开避免完全重叠。 -->
      <div v-for="(m, i) in msgs" :key="m.id" class="ezfy-msg" :class="'ezfy-msg-' + m.type"
           :style="msgStyle(m, i)">
        {{ m.text }}
        <a href="javascript:;" class="ezfy-msg-close" @click="closeMsg(m.id)">[关闭]</a>
      </div>

      <!-- 页面内确认条（代替 confirm/prompt 弹窗）
           ★ 2026-09-25 用户反馈「军官详情的一键穿戴，确认跑到页面最上面去了，要滚上去才能点」：
             改成和提示条(.ezfy-msg)同一套做法 —— **浮在刚才点的那个控件旁边**（位置由 askStyle 算），
             不再永远钉在页面顶部。DOM 位置留着不动，靠 position:fixed 浮起来。 -->
      <div class="ezfy-ask" v-if="askBox.show" ref="askBox" :style="askStyle">
        <div class="ezfy-ask-text">{{ askBox.text }}</div>
        <div class="ezfy-ask-row" v-if="askBox.input">
          <input v-model="askBox.value" :placeholder="askBox.placeholder" style="width:60%"/>
        </div>
        <div class="ezfy-ask-row">
          <a href="javascript:;" class="ezfy-ask-ok" @click="askConfirm(true)">[确定]</a>
          <a href="javascript:;" class="ezfy-ask-cancel" @click="askConfirm(false)">[取消]</a>
        </div>
      </div>

      <!-- 二级导航（资源/军官/军队/科技/城防/统帅）—— 只在对应页面显示，位置固定在顶部，不再有的在底部 -->
      <div class="old-line ezfy-subnav" v-if="showSubnav">
        <a href="javascript:;" :class="{ on: cur === 'builds' }" @click="go('builds')">资源</a>.
        <a href="javascript:;" :class="{ on: cur === 'acade' || cur === 'officerdetail' }" @click="go('acade')">军官</a>.
        <a href="javascript:;" :class="{ on: isArmyPage }" @click="go('troops')">军队</a>.
        <a href="javascript:;" :class="{ on: cur === 'techs' }" @click="go('techs')">科技</a>.
        <a href="javascript:;" :class="{ on: cur === 'defence' }" @click="go('defence')">城防</a>.
        <a href="javascript:;" :class="{ on: cur === 'info' }" @click="go('info')">统帅</a>
      </div>

      <!-- ============ 首页(cityHome) ============ -->
      <template v-if="cur === 'home'">
        <!-- ★ 只有【置顶】公告展示到首页外边；普通公告进「公告」页看。
             外面包一层 .ezfy-notices 只为统一它与上下两行的间距(见样式表注释)。 -->
        <div class="ezfy-notices" v-if="topNotices.length">
          <div class="old-line" v-for="n in topNotices" :key="'n' + n.id">
            <img class="logo-title" src="/static/ezfy/notice.gif" alt="."/>
            <a class="red" href="javascript:;" @click="openNotice(n)">{{ n.title }}</a>
          </div>
        </div>

        <div class="old-line">
          <span class="city-name">{{ city.name }}({{ city.x }},{{ city.y }})</span>
          <a href="javascript:;" @click="go('cities')">切换城市</a>
        </div>
        <div class="old-line">
          <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>军团</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#2F5D8A"/><path d="M10 3.3 L15.4 5.2 V10.4 C15.4 13.5 13.2 15.6 10 16.5 C6.8 15.6 4.6 13.5 4.6 10.4 V5.2 Z" fill="#fff"/><path d="M10 7.4 L11.2 9.6 L13.6 9.8 L12 11.3 L12.5 13.6 L10 12.3 L7.5 13.6 L8 11.3 L6.4 9.8 L8.8 9.6 Z" fill="#2F5D8A"/></svg>
          军团:
          <a href="javascript:;" @click="go('corps')" v-if="!myCorps">加入军团</a>
          <a href="javascript:;" @click="go('corps')" v-else>[{{ myCorps.name }}]</a>
        </div>
        <div class="old-line">声望: {{ profile.prestige }}</div>
        <div class="old-line">
          <span v-html="rankIcon(myRankId)"></span>
          <a href="javascript:;" @click="go('rank')">军衔</a>: {{ rankName }}
        </div>
        <div class="old-line">每日签到: <a href="javascript:;" @click="go('welfare')">{{ welfare.signed_today ? '已签到' : '签到' }}</a></div>

        <div class="old-line home-nav2">
          <a href="javascript:;" @click="go('builds')">资源</a>.
          <a href="javascript:;" @click="go('acade')">军官</a>.
          <a href="javascript:;" @click="go('troops')">军队</a>.
          <a href="javascript:;" @click="go('techs')">科技</a>.
          <a href="javascript:;" @click="go('defence')">城防</a>.
          <a href="javascript:;" @click="go('info')">统帅</a>
        </div>
        <div class="old-line">现有资源/产量:
          <a href="javascript:;" @click="go('exchange')">购买</a>
          <a href="javascript:;" @click="go('mall')">增产</a>
        </div>
        <div class="old-line">
          <span :title="resNames.gold"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-linecap="round" stroke-linejoin="round" fill="none"><path d="M6.4 5.8 L10 10.3 L13.6 5.8" stroke-width="1.6"/><path d="M10 6 V14.2" stroke-width="1.6"/><path d="M7.6 8.9 H12.4" stroke-width="1.4"/><path d="M7.6 11.7 H12.4" stroke-width="1.4"/></g></svg></span>
          <a href="javascript:;" @click="go('res/gold')">{{ resNames.gold }}:</a><span :title="'现有 ' + fmtN(city.gold)">{{ fmtProd(city.gold) }}</span>/<span :title="resShort.gold + '每小时产量'">{{ fmtProd(resProd.gold) }}</span>
        </div>
        <div class="old-line">
          <span :title="resNames.food"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><g stroke="#FFD700" stroke-width="1.5" stroke-linecap="round" fill="none"><path d="M7.2 6.2 C6.7 5.4 7.3 4.1 7.9 3.5"/><path d="M12.8 6.2 C13.3 5.4 12.7 4.1 12.1 3.5"/></g><path d="M5 8.6 H15 C15 8.6 14.7 12.2 13.3 13.7 C12.2 14.8 10.9 15.4 10 15.4 C9.1 15.4 7.8 14.8 6.7 13.7 C5.3 12.2 5 8.6 5 8.6 Z" fill="#FFD700"/></svg></span>
          <a href="javascript:;" @click="go('res/food')">{{ resNames.food }}:</a><span :title="'现有 ' + fmtN(city.food)">{{ fmtProd(city.food) }}</span>/<span :title="resShort.food + '每小时产量'">{{ fmtProd(resProd.food) }}</span>
        </div>
        <div class="old-line">
          <span :title="resNames.steel"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M4.2 8.4 L6.6 15.6 H13.4 L15.8 8.4 Z" fill="#FFD700"/><path d="M5.6 10.2 H14.4" stroke="#C9AFF0" stroke-width="1.3" stroke-linecap="round"/></svg></span>
          <a href="javascript:;" @click="go('res/steel')">{{ resNames.steel }}:</a><span :title="'现有 ' + fmtN(city.steel)">{{ fmtProd(city.steel) }}</span>/<span :title="resShort.steel + '每小时产量'">{{ fmtProd(resProd.steel) }}</span>
        </div>
        <div class="old-line">
          <span :title="resNames.oil"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 3.6 C10 3.6 6.1 8.1 6.1 11.2 C6.1 13.5 7.8 15.3 10 15.3 C12.2 15.3 13.9 13.5 13.9 11.2 C13.9 8.1 10 3.6 10 3.6 Z" fill="#FFD700"/><circle cx="8.6" cy="11.4" r="0.9" fill="#C9AFF0"/></svg></span>
          <a href="javascript:;" @click="go('res/oil')">{{ resNames.oil }}:</a><span :title="'现有 ' + fmtN(city.oil)">{{ fmtProd(city.oil) }}</span>/<span :title="resShort.oil + '每小时产量'">{{ fmtProd(resProd.oil) }}</span>
        </div>
        <div class="old-line">
          <span :title="resNames.rare"><svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M5.8 6.4 H14.2 L11.7 9.2 L10 15.6 L8.3 9.2 Z" fill="#FFD700"/><path d="M5.8 6.4 H10 L8.3 9.2 Z" fill="#C9AFF0"/></svg></span>
          <a href="javascript:;" @click="go('res/rare')">{{ resNames.rare }}:</a><span :title="'现有 ' + fmtN(city.rare)">{{ fmtProd(city.rare) }}</span>/<span :title="resShort.rare + '每小时产量'">{{ fmtProd(resProd.rare) }}</span>
        </div>
        <div class="old-line">
          <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>人口</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><circle cx="7" cy="7.4" r="1.6" fill="#FFD700"/><path d="M4.7 14.6 C4.7 12.7 5.7 11.5 7 11.5 C8.3 11.5 9.3 12.7 9.3 14.6 Z" fill="#FFD700"/><circle cx="13" cy="6.6" r="1.5" fill="#FFD700"/><path d="M10.9 14.6 C10.9 12.9 11.9 11.9 13 11.9 C14.1 11.9 15.1 12.9 15.1 14.6 Z" fill="#FFD700"/></svg>人口/空闲:{{ city.pop }}/{{ freePop }}
          <a href="javascript:;" @click="go('convene')">召集</a>
        </div>
        <div class="old-line">
          <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>民心</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M10 15.5 C5.3 12.6 4.1 9.6 4.1 7.6 C4.1 5.9 5.4 4.7 7 4.7 C8.1 4.7 9.2 5.3 10 6.3 C10.8 5.3 11.9 4.7 13 4.7 C14.6 4.7 15.9 5.9 15.9 7.6 C15.9 9.6 14.7 12.6 10 15.5 Z" fill="#FFD700"/></svg>民心/民怨:{{ city.feelings }}/{{ city.grievance }}
          <a href="javascript:;" @click="go('placate')">安抚</a>
        </div>
        <div class="old-line">
          <svg class="ezfy-ico" viewBox="0 0 20 20" role="img"><title>税率</title><rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="#6C48A8"/><path d="M14.6 5.4 L5.4 14.6" stroke="#FFD700" stroke-width="1.4" stroke-linecap="round"/><circle cx="6.9" cy="5.9" r="1.8" fill="#FFD700"/><circle cx="13.1" cy="14.1" r="1.8" fill="#FFD700"/></svg>
          <a href="javascript:;" @click="go('taxset')">税率:</a>{{ city.tax_rate }}%
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('buildm')">军事区</a><span class="home-gap"></span><a href="javascript:;" @click="openBuildPre('m')">建造</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('builds')">资源区</a><span class="home-gap"></span><a href="javascript:;" @click="openBuildPre('s')">建造</a>
        </div>
        <div class="old-line">
          训练军队
          <a href="javascript:;" @click="go('troop')">[造兵]</a><span class="home-gap"></span><a href="javascript:;" @click="go('defence')">[建防]</a>
        </div>
        <div class="old-line">
          前往
          <a href="javascript:;" @click="go('map')">地图</a>
          出征
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('citystatus')">城市状态</a><span class="home-gap"></span><a href="javascript:;" @click="go('wilds')">附属野地</a>
        </div>
        <div class="old-line">【世界聊天】<a href="javascript:;" @click="go('chat')">[进入]</a></div>
        <!-- [世界] 安珞：11111 / [军团] / [私聊] / [系统]; 昵称用实时昵称+个性颜色 -->
        <div class="old-line" v-for="ch in homeChats" :key="'wc' + ch.key">
          [<span class="orange">{{ ch.tag }}</span>]
          <!-- ★ 2026-09-28 bug修复: 冒号+内容从 v-if/v-else 判断结构移出外层,
               否则 Vue 会把紧跟 v-if 的「：{{content}}」静文并进 else 分支,
               导致有 user_id 的玩家消息只出昵称不出内容。现在两层分支都只负责出「昵称：」/「系统：」头, 内容恒在 -->
          <span v-if="ch.user_id">
            <a href="javascript:;" @click="openPlayer(ch.user_id)"><span
               v-for="(c, ci) in nickChars(ch.user_name)" :key="'nc' + ci"
               :style="nickColorAt(ch.color, ci)">{{ c }}</span></a>：
          </span>
          <span v-else>{{ ch.tag == '系统' ? '系统：' : ch.tag + '：' }}</span>
          {{ ch.content }}
        </div>
        <div class="old-line gray" v-if="!homeChats.length">(暂无消息)</div>

      </template>

      <!-- ============ 世界聊天(chat) ============ -->
      <template v-else-if="cur === 'chat'">
        <div class="panel">
          <div class="panel-title">聊天频道</div>
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: chatChannel === 1 }" @click="switchChannel(1)">世界</a>|
            <!-- ★ 军团频道常显：没加入军团时进去显示「未加入 · 0 人」 -->
            <a href="javascript:;" :class="{ on: chatChannel === 2 }" @click="switchChannel(2)">军团</a>|
            <a href="javascript:;" :class="{ on: chatChannel === 4 }" @click="switchChannel(4)">系统</a>|
            <a href="javascript:;" @click="openPm()">私聊</a>
          </div>

          <!-- ★ 发言框移到聊天列表**上方** 列表按时间降序、最新在最上面 -->
          <div class="old-line ezfy-chat-send">
            <template v-if="chatCanSend">
              <input v-model="chatMsg" class="ezfy-chat-input" maxlength="25" @keyup.enter="doChatSend"/>
              <a v-if="chatCooldown <= 0" href="javascript:;" @click="doChatSend">[发送]</a>
              <span v-else class="gray">冷却中 {{ chatCooldown }}s</span>
              <span class="gray">(最大25个字)</span>
            </template>
            <span v-else class="gray">(系统频道仅系统可发言)</span>
            <a href="javascript:;" @click="loadChats">[刷新]</a>
          </div>

          <!-- 系统频道: 系统消息(只读) ★ 2026-09-29 用户要求去掉「系统公告」——首页已有公告入口(置顶公告+底部导航「公告」) -->
          <template v-if="chatChannel === 4">
            <div class="panel-title">系统消息</div>
            <div class="old-line" v-for="ch in worldChats" :key="'cs' + ch.id">
              [<span class="orange">系统</span>]
              <span class="gray">{{ fmtTime(ch.created_at) }}</span>
              ：{{ ch.content }}
            </div>
            <div class="old-line gray" v-if="!worldChats.length">(暂无系统消息)</div>
            <div class="ezfy-pager" v-if="chatTotalPages > 1">
              <a href="javascript:;" :class="{ gray: chatPage <= 1 }" @click="chatGo(-1)">上一页</a>
              <span class="gray">第 {{ chatPage }}/{{ chatTotalPages }} 页（共 {{ chatTotal }} 条）</span>
              <a href="javascript:;" :class="{ gray: chatPage >= chatTotalPages }" @click="chatGo(1)">下一页</a>
            </div>
          </template>

          <!-- 军团频道但还没加入军团：显示 0 人 + 引导 -->
          <template v-else-if="chatChannel === 2 && !chatHasCorps">
            <div class="old-line gray">
              你还没有加入军团（军团人数 0）。
              <a href="javascript:;" @click="go('corps')">[去看看军团列表]</a>
            </div>
          </template>

          <!-- 公共 / 军团频道 -->
          <template v-else>
            <div class="panel-title">
              {{ chatChannel === 2
                ? '军团聊天(' + (chatHasCorps ? chatCorpsName : '未加入军团') + ')(' + chatCorpsPlayers + '人)'
                : '世界聊天(' + chatPlayers + '人)' }}
            </div>
            <div class="old-line" v-for="ch in worldChats" :key="'c' + ch.id">
              [<span class="orange">{{ chatChannel === 2 ? '军团' : '世界' }}</span>]
              <span class="gray">{{ fmtTime(ch.created_at) }}</span>
              <a href="javascript:;" @click="openPlayer(ch.user_id)"><span
                 v-for="(c, ci) in nickChars(ch.user_name)" :key="'ncc' + ci"
                 :style="nickColorAt(ch.color, ci)">{{ c }}</span></a>：{{ ch.content }}
            </div>
            <div class="old-line" v-if="!worldChats.length">(暂无消息, 快来说点什么吧)</div>
            <div class="ezfy-pager" v-if="chatTotalPages > 1">
              <a href="javascript:;" :class="{ gray: chatPage <= 1 }" @click="chatGo(-1)">上一页</a>
              <span class="gray">第 {{ chatPage }}/{{ chatTotalPages }} 页（共 {{ chatTotal }} 条）</span>
              <a href="javascript:;" :class="{ gray: chatPage >= chatTotalPages }" @click="chatGo(1)">下一页</a>
            </div>
          </template>
        </div>
      </template>

      <!-- ============ 私聊(mail) ============ -->
      <template v-else-if="cur === 'mail'">
        <div class="panel">
          <div class="panel-title">私聊 · 会话列表</div>
          <div class="old-line" v-for="c in pmConvs" :key="'cv' + c.user_id">
            <a href="javascript:;" @click="selectPm(c.user_id)">
              <span :class="{ red: pmPeer && pmPeer.id === c.user_id }">
                {{ pmPeer && pmPeer.id === c.user_id ? '▶ ' : '' }}{{ c.nickname }}</span></a>
            <span class="gray">({{ c.username }})</span>
            <span v-if="c.unread > 0" class="red">[未读{{ c.unread }}]</span>
            <br/>
            <span class="gray">{{ c.last_content }}</span>
            <span class="gray"> ({{ fmtTime(c.last_at) }})</span>
          </div>
          <div class="old-line" v-if="!pmConvs.length">(还没有聊过的人，在下面填游戏ID或昵称发起私聊)</div>
          <br/>
          <button @click="loadPmConvs">刷新会话</button>
        </div>

        <div class="panel" v-if="pmPeer">
          <div class="panel-title">与 {{ pmPeer.nickname }}({{ pmPeer.username }}) 的聊天记录</div>
          <div class="old-line" v-for="m in pmChat" :key="'pc' + m.id">
            <span :class="m.sender_id === myUserId ? 'green' : ''">
              {{ m.sender_id === myUserId ? '我' : pmPeer.nickname }}</span>：{{ m.content }}
            <span class="gray">({{ fmtTime(m.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!pmChat.length">(还没有聊天记录，发一条试试)</div>
          <br/>
          <button @click="selectPm(pmPeer.id)">刷新记录</button>
          <a href="javascript:;" @click="pmPeer = null; pmChat = []; pmTo = ''">[关闭会话]</a>
        </div>

        <div class="panel">
          <div class="panel-title">发私信</div>
          <div class="old-line">
            收件人:
            <input v-model="pmTo" placeholder="游戏ID / 昵称" style="width:170px" list="ezfyPmCands"/>
            <datalist id="ezfyPmCands">
              <option v-for="f in pmCandidates" :key="'pmc' + f.id" :value="f.name"></option>
            </datalist>
            <span class="gray" v-if="pmPeer">（当前会话：{{ pmPeer.nickname }}）</span>
          </div>
          <div class="old-line gray">不需要先加好友, 填对方游戏ID或昵称即可; 对方把你拉黑则发不出去。</div>
          <div class="old-line">
            <!-- ★ 改成可自适应高度的文本域（用户要求）：随内容长高，最多 8 行后内部滚动 -->
            <textarea v-model="pmContent" class="ezfy-auto-textarea" placeholder="最多500字"
                      maxlength="500" rows="2"></textarea>
          </div>
          <div class="old-line">
            <button @click="doSendPm">发送</button>
            <a href="javascript:;" @click="go('friends')">[好友]</a>
            <a href="javascript:;" @click="go('chat')">[聊天频道]</a>
          </div>
        </div>

        <!-- 收到的私信（系统通知 / 别人的来信） -->
        <div class="panel">
          <div class="panel-title">收到的私信</div>
          <div class="old-line" v-for="m in mails" :key="'m' + m.id">
            <a href="javascript:;" @click="selectPm(m.sender_id)"><span
               :class="{ red: m.is_read === 0 }">{{ m.sender }}</span></a>:
            {{ m.content }} <span class="gray">({{ fmtTime(m.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!mails.length">(暂无私信)</div>
          <br/>
          <a href="javascript:;" @click="loadMails">[刷新]</a>
        </div>
      </template>

      <!-- ============ 情报/军情(reports) ============ -->
      <template v-else-if="cur === 'reports'">
        <div class="panel">
          <!-- 复刻 report/index.html: 军队动态 . 驻军 . 军情警讯 . 战斗报告 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: reportTab === 1 }" @click="switchReportTab(1)">军队动态</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 2 }" @click="switchReportTab(2)">驻军</a><span
              v-if="dynStation.length" class="green">({{ dynStation.length }})</span><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 3 }" @click="switchReportTab(3)">军情警讯</a><span
              v-if="reportCounts[1]" class="red">({{ reportCounts[1] }})</span><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 4 }" @click="switchReportTab(4)">战斗报告</a><span
              v-if="reportCounts[2]" class="red">({{ reportCounts[2] }})</span><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 5 }" @click="switchReportTab(5)">军团战报</a>
          </div>

          <!-- ===== 军队动态: 行进/战斗/返航中的部队(出征/侦查/掠夺/运输/增援等) ===== -->
          <template v-if="reportTab === 1">
            <div class="old-line" v-for="o in dynMarchPaged" :key="'dy' + o.id">
              命令：{{ o.type_name }} <a v-if="!o.is_defend" href="javascript:;" @click="openOrder(o)">查看</a><br/>
              <!-- ★ 2026-09-28 同驻军：外出部队也标出「哪个城出来的」(敌军来袭的防守视角无此字段, 故 v-if) -->
              <span v-if="o.from_city">起点：{{ o.from_city }}({{ o.from_x }},{{ o.from_y }})<br/></span>
              目标：<span v-if="o.act_type" class="red">[{{ actTag(o.act_type) }}]</span>{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})
              <span v-if="o.is_defend" class="red">(敌军来袭)</span><br/>
              状态：{{ o.status_name }}
              <template v-if="o.can_command">
                <a href="javascript:;" class="red" @click="openBattle(o.id)">[指挥]</a>
                <span class="gray">第{{ o.battle_round || 1 }}/{{ o.battle_max }}回合</span>
              </template>
              <!-- ★ 2026-09-30 行军计谋：出征中(0)/返回中(2)只显示一个 [计谋]，
                   点击进入计谋页选择 神兵天降(去程减80%)/战略转移(回程减360分钟) -->
              <template v-if="o.status === 0 || o.status === 2">
                <a class="green" href="javascript:;" @click="openScheme(o)">[计谋]</a>
              </template>
              <br/>
              军官：{{ o.officer || '无' }}<br/>
              {{ o.time_label }}：{{ o._lt || o.time_text }}<br/>
              <!-- ★ carry 现在只用于「运输」在途物资（采集资源已改为收获即入起点城市，不走 carry） -->
              <span v-if="o.carry_total > 0" class="green">
                在途物资：{{ fmtN(o.carry.food) }}粮/{{ fmtN(o.carry.steel) }}钢/{{ fmtN(o.carry.oil) }}油/{{ fmtN(o.carry.rare) }}稀/{{ fmtN(o.carry.gold) }}金
                （负重 {{ fmtN(o.carry_total) }}/{{ fmtN(o.carry_cap) }}）
              </span>
              <br/>
              --------------------
            </div>
            <div class="old-line" v-if="!dynMarch.length">(当前没有在外的部队)</div>
            <div class="ezfy-pager" v-if="dynMarch.length > dynSize">
              <a href="javascript:;" :class="{ gray: dynPage <= 1 }" @click="sectionPagerGo('dyn', -1)">上一页</a>
              <span class="gray">第 {{ dynPage }}/{{ dynMarchTotalPages }} 页（共 {{ dynMarch.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: dynPage >= dynMarchTotalPages }" @click="sectionPagerGo('dyn', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 驻军: 到达野地后常驻采集的部队(满一个采集周期结算一期) ===== -->
          <template v-else-if="reportTab === 2">
            <div class="old-line">
              <!-- ★ 采集玩法说明：驻守空闲需点[采集]；满一个采集周期结算一期（资源+宝物）；负重满后超出部分直接入起点城市（2026-09-30 不再丢弃）；[停止采集]/[一键收获]取回负重，[召回]撤兵。 -->
              <a href="javascript:;" @click="doCollectAll">[一键采集]</a>
              <a href="javascript:;" @click="doHarvestAll">[一键收获]</a>
              <a href="javascript:;" @click="doRecallAll">[一键召回]</a>
            </div>
            <div class="old-line" v-for="o in dynStationPaged" :key="'st' + o.id">
              命令：{{ o.type_name }} <a href="javascript:;" @click="openOrder(o)">查看</a><br/>
              <!-- ★ 2026-09-28 用户要求：显示这支部队是「哪个城出来的」(辨识番号) -->
              <span v-if="o.from_city">起点：{{ o.from_city }}({{ o.from_x }},{{ o.from_y }})<br/></span>
              目标：{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})<br/>
              军官：{{ o.officer || '无' }}<br/>
              {{ o.time_label }}：{{ o._lt || (o._lg ? o._lg.timeText : o.time_text) }}
              <span v-if="o.status === 1 && !o.arrive_time"><a href="javascript:;" class="red" @click="startCollect(o)">[采集]</a></span>
              <span v-else-if="o.status === 1 && o.arrive_time"><a href="javascript:;" class="red" @click="stopCollect(o)">[停止]</a></span><br/>
              <!-- ★ 2026-09-28 采集中部队: 实时累加显示本期已采资源(每秒由 liveGather 重算)。
                   规则已改为「收获即入起点城市」，故不再显示「需召回返航后入库」。 -->
              <template v-if="o.status === 1 && o.arrive_time">
                <span class="green">本期已采：{{ fmtN(o._lg.food) }}粮/{{ fmtN(o._lg.steel) }}钢/{{ fmtN(o._lg.oil) }}油/{{ fmtN(o._lg.rare) }}稀/{{ fmtN(o._lg.gold) }}金</span>
                <span class="gray">（总 {{ fmtN(o._lg.total) }}，负重 {{ fmtN(o._lg.total) }}/{{ fmtN(o.carry_cap) }}）</span>
                <span v-if="o._lg.full" class="red">负重已满, 超出部分会直接入库(可停止或收获)。</span>
              </template>
              <span v-else class="gray">本期已采：暂无(未在采集中)</span>
              <br/>
              --------------------
            </div>
            <div class="old-line" v-if="!dynStation.length">(当前没有驻守采集的部队)</div>
            <div class="ezfy-pager" v-if="dynStation.length > dynStationSize">
              <a href="javascript:;" :class="{ gray: dynStationPage <= 1 }" @click="sectionPagerGo('sta', -1)">上一页</a>
              <span class="gray">第 {{ dynStationPage }}/{{ dynStationTotalPages }} 页（共 {{ dynStation.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: dynStationPage >= dynStationTotalPages }" @click="sectionPagerGo('sta', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 军情警讯: 别人打我 ===== -->
          <template v-else-if="reportTab === 3">
            <div class="old-line">
              <span class="gray">敌方来袭预警、被侦查、被掠夺、被征服都在这里看；</span>
              <a href="javascript:;" @click="loadReports">[刷新]</a>
            </div>
            <!-- ★ 雷达站 + 侦察技巧 决定「事前预警」能看到多少（事后结果战报不受影响）
                 ★ 2026-09-25 用户要求「军情警讯要看到对面城市名字和地址」→ 这里明确写出
                   还差多少才能看到「出发城市(坐标)」，否则玩家永远不知道该升什么。 -->
            <div class="old-line" v-if="reportRadar > 0">
              <span class="gray">情报等级 <b>{{ reportIntel }}</b> = 雷达站 {{ reportRadar }} 级 + 侦察技巧 {{ reportRecon }} 级：</span>
              <span v-if="reportIntel >= 2" class="gray">已能在预警里看到<b>来袭城市名称与坐标</b>，等级越高情报越详细。</span>
              <span v-else class="orange">再升 1 级（雷达站或侦察技巧均可）就能看到<b>来袭城市名称与坐标</b>。</span>
            </div>
            <div class="old-line" v-else>
              <span class="red">尚未建造雷达站：收不到「敌军来袭 / 被侦查」预警；被掠夺、被征服的结果战报仍会记录在这里。</span>
            </div>
            <div class="old-line" v-for="r in repPaged" :key="'rw' + r.id">
              <a href="javascript:;" @click="openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>
                <span v-if="intelTag(r)" class="orange">[{{ intelTag(r) }}] </span>{{ intelTitle(r) }}</a>
              <span class="gray">({{ fmtTime(r.created_at) }})</span>
            </div>
            <div class="old-line" v-if="!reports.length">(暂无军情警讯)</div>
            <div class="ezfy-pager" v-if="reports.length > repSize">
              <a href="javascript:;" :class="{ gray: repPage <= 1 }" @click="sectionPagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ repPage }}/{{ repTotalPages }} 页（共 {{ reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: repPage >= repTotalPages }" @click="sectionPagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战斗报告: 我打别人 + 战报查询 ===== -->
          <template v-else>
            <div class="old-line">
              战报查询:
              <input v-model="reportWord" placeholder="输入关键字" style="width:110px"
                     @keyup.enter="loadReports"/>
              <a href="javascript:;" @click="loadReports">[查询]</a>
              <!-- ★ 2026-09-26 用户要求：查询右边加 [一键删除]（物理删除自己名下全部战报，节约服务器资源）
                   ★ 2026-09-30 军团战报是团员的战报，不能一键删除 -->
              <a v-if="reportTab !== 5" href="javascript:;" @click="doClearReports">[一键删除]</a>
              <a v-if="reportWord" href="javascript:;" @click="reportWord = ''; loadReports()">[清空]</a>
            </div>
            <div class="old-line" v-for="r in repPaged" :key="'rb' + r.id">
              <a href="javascript:;" @click="openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>
                <span v-if="reportTab === 5 && r.owner_name" class="blue">{{ r.owner_name }}：</span>
                <span class="orange">[{{ r.type_name }}]</span> {{ r.title }}</a>
              <span class="gray">({{ fmtTime(r.created_at) }})</span>
            </div>
            <div class="old-line" v-if="!reports.length">{{ reportTab === 5 ? '(暂无军团战报)' : '(暂无战斗报告)' }}</div>
            <div class="ezfy-pager" v-if="reports.length > repSize">
              <a href="javascript:;" :class="{ gray: repPage <= 1 }" @click="sectionPagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ repPage }}/{{ repTotalPages }} 页（共 {{ reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: repPage >= repTotalPages }" @click="sectionPagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战报详情（已改为独立页面 reportview，不再行内展开） ===== -->
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 战报详情(reportview) ============ -->
      <template v-else-if="cur === 'reportview'">
        <div class="panel" v-if="curReport">
          <!-- ★ 用户要求：战报详情页也保留「军队动态 . 驻军 . 军情警讯 . 战斗报告」导航 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: reportTab === 1 }" @click="goReportTab(1)">军队动态</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 2 }" @click="goReportTab(2)">驻军</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 3 }" @click="goReportTab(3)">军情警讯</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: reportTab === 4 }" @click="goReportTab(4)">战斗报告</a>
          </div>
          <div class="panel-title">{{ curReport.title }}</div>
          <!-- ★ 2026-09-29 战报上色：攻方绿色、守方红色，看不出谁是谁 → 视觉区分 -->
          <div v-for="(seg, i) in reportNiceLines(curReport.content)" :key="'rc' + i" class="rpt-ln">
            <template v-if="seg.mode === 'pair'">
              <span :class="seg.left.cls">{{ seg.left.text }}</span><span :class="seg.right.cls">{{ seg.right.text }}</span>
            </template>
            <span v-else :class="seg.cls">{{ seg.text }}</span>
          </div>
          <template v-if="curReport.detail">
            <div class="old-line"><a href="javascript:;" @click="showDetail = !showDetail">[展开/收起逐回合详情]</a></div>
            <div v-if="showDetail" class="rpt-ln" v-for="(seg, j) in reportNiceLines(curReport.detail)" :key="'rd' + j">
              <template v-if="seg.mode === 'pair'">
                <span :class="seg.left.cls">{{ seg.left.text }}</span><span :class="seg.right.cls">{{ seg.right.text }}</span>
              </template>
              <span v-else :class="seg.cls">{{ seg.text }}</span>
            </div>
          </template>
          <div class="old-line">
            <a href="javascript:;" class="red" @click="delReport(curReport)">[删除]</a>
            <a href="javascript:;" @click="go('reports')">[返回]</a>
            <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
        </div>
      </template>

      <!-- ============ 好友(friends) ============ -->
      <template v-else-if="cur === 'friends'">
        <div class="panel">
          <div class="panel-title">游戏内好友</div>
          <div class="panel-title">搜索玩家（按游戏ID / 玩家号码 / 昵称）</div>
          <div class="old-line">
            <input v-model="friendKeyword" placeholder="输入游戏ID / 玩家号码 / 昵称" style="width:170px"/>
            <button @click="doFriendSearch">[搜索]</button>
          </div>
          <table v-if="friendSearchDone">
            <tr><th>游戏ID</th><th>昵称</th><th>声望</th><th>状态</th><th>操作</th></tr>
            <tr v-for="u in friendSearchList" :key="'fs' + u.user_id">
              <td><span class="td-mono">{{ u.game_uid }}</span></td>
              <td><a href="javascript:;" @click="openPlayer(u.user_id)">{{ u.nickname }}</a></td>
              <td>{{ u.prestige }}</td>
              <td><span class="gray">{{ u.rank_name }}</span></td>
              <td>
                <span v-if="u.is_friend" class="gray">已是好友</span>
                <span v-else-if="u.applied" class="orange">已申请</span>
                <a v-else href="javascript:;" @click="doAddFriend(u)">[加好友]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="friendSearchDone && !friendSearchList.length">(没找到这位统帅, 换个游戏ID或昵称试试)</div>

          <div class="panel-title">好友申请（待处理 {{ friendApplies.inbox.length }}）</div>
          <table v-if="friendApplies.inbox.length">
            <tr><th>游戏ID</th><th>昵称</th><th>验证信息</th><th>操作</th></tr>
            <tr v-for="a in friendApplies.inbox" :key="'fa' + a.apply_id">
              <td><span class="td-mono">{{ a.game_uid }}</span></td>
              <td><a href="javascript:;" @click="openPlayer(a.user_id)">{{ a.nickname }}</a></td>
              <td>{{ a.remark || '—' }}</td>
              <td>
                <a href="javascript:;" @click="doHandleApply(a, true)">[同意]</a>
                <a href="javascript:;" @click="doHandleApply(a, false)">[拒绝]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-else>(暂无新的好友申请)</div>

          <div class="panel-title">我的游戏好友（{{ friends.length }}）</div>
          <table v-if="friends.length">
            <tr><th>游戏ID</th><th>昵称</th><th>声望</th><th>军衔</th><th>操作</th></tr>
            <tr v-for="f in friends" :key="'f' + f.user_id">
              <td><span class="td-mono">{{ f.game_uid }}</span></td>
              <td><a href="javascript:;" @click="openPlayer(f.user_id)">{{ f.nickname }}</a></td>
              <td>{{ f.prestige }}</td>
              <td>{{ f.rank_name }}</td>
              <td>
                <a href="javascript:;" @click="openPlayer(f.user_id)">[统帅信息]</a>
                <a href="javascript:;" @click="openPm(f.user_id)">[私聊]</a>
                <a href="javascript:;" @click="doDelFriend(f)">[删除]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-else>(还没有游戏好友, 用上面的搜索找找吧)</div>
        </div>
      </template>

      <!-- ============ 任务(tasks) ============ -->
      <template v-else-if="cur === 'tasks'">
        <div class="panel">
          <!-- ★ 2026-09-24 用户要求: 任务按分类 tab 分别展示(新手/日常/每周) -->
          <div class="acade-tab">
            <template v-for="(g, i) in taskGroups">
              <!-- ★ 分隔竖线放 <a> 外: 选中态(加粗变色)不波及竖线 -->
              <span v-if="i > 0" :key="'ts' + g.id"> | </span>
              <!-- ★ 2026-09-27 fix: 选中判断需 taskTab !== -1, 否则停在"为爱发电卡"时会高亮"新手任务" -->
              <a :key="'tgt' + g.id" href="javascript:;"
                 :class="{ on: taskTab !== -1 && taskGroupCur.id === g.id }" @click="selectTaskTab(g.id)"><span>{{ g.name }}</span></a>
            </template>
            <!-- ★ 2026-09-27 为爱发电卡 tab: 管理端未发放(love_cards 为空)时不显示 -->
            <span v-if="loveCards.length && taskGroups.length"> | </span>
            <a v-if="loveCards.length" href="javascript:;"
               :class="{ on: taskTab === -1 }" @click="selectTaskTab(-1)"><span>为爱发电卡</span></a>
          </div>

          <!-- ★ 为爱发电卡内容（多卡可叠加领取） -->
          <template v-if="taskTab === -1 && loveCards.length">
            <div style="margin-top:6px">
              <div class="old-line" v-for="c in loveCards" :key="c.id">
                <b>{{ c.name }}</b> 每日 <b>{{ c.daily_diamond }}</b> 钻石 · 已 <b>{{ c.claimed_days }}/{{ c.total_days }}</b> 天
                <span v-if="c.remaining > 0" class="gray">（剩{{ c.remaining }}天）</span>
                <span v-else class="green">（已领完）</span>
                <span v-if="c.claimable > 0"> · 可领 <b>{{ c.claimable }}</b> 天</span>
              </div>
              <div class="old-line">
                <a v-if="loveTotalClaimable > 0" href="javascript:;" @click="doLoveCardClaim">[领取每日钻石]</a>
                <span v-else class="gray">[今日暂无可领取, 明天再来]</span>
              </div>
              <div class="sub gray">漏领的天数会在之后领取时累加补齐（每卡封顶 {{ 30 }} 天）</div>
            </div>
          </template>

          <!-- 普通分类任务 -->
          <template v-else-if="taskTab !== -1">
            <div v-if="taskGroupCur.tasks.length" style="margin-top:6px">
              <div class="old-line" v-for="t in taskGroupCur.tasks" :key="t.id">
                <b>{{ t.name }}</b> {{ t.current }}/{{ t.target }}
                <span v-if="t.status === 2" class="gray">[已领取]</span>
                <a v-else-if="t.status === 1" href="javascript:;" @click="doAward(t)">[领奖]</a>
                <br/>
                <span class="gray">奖励:{{ rewardText(t.reward) }}</span>
              </div>
            </div>
            <div class="old-line" v-else>(暂无任务)</div>
          </template>
        </div>
      </template>

      <!-- ============ 城市列表(cities) ============ -->
      <template v-else-if="cur === 'cities'">
        <div class="panel">
          <div class="panel-title">我的城市列表</div>
          <div class="old-line" v-for="ct in cities" :key="'ct' + ct.id">
            <!-- ★ 2026-09-29 城市列表改版：一行一座城 = 城市名(坐标)；点城市名**切换**；
                 [运输][派遣][弃城] 只在非当前城显示（当前城无操作） -->
            <a v-if="ct.id !== city.id" class="city-name" href="javascript:;" @click="doSwitch(ct)" title="切换为当前城市">{{ ct.name }}</a>({{ ct.x }},{{ ct.y }})
            <span v-else class="city-name">{{ ct.name }}</span>({{ ct.x }},{{ ct.y }})
            <template v-if="ct.id !== city.id">
              <a href="javascript:;" @click="doTransportTo(ct)">[运输]</a>
              <a href="javascript:;" @click="doDispatchTo(ct)">[派遣]</a>
              <a class="red" href="javascript:;" @click="doDestroyCity(ct)">[弃城]</a>
            </template>
          </div>

          <br/>
          <div class="panel-title">起新城 (消耗10万{{ resNames.gold }})</div>
          <div class="old-line gray">
            军衔「{{ rankData.mine ? rankData.mine.rank_name : rankName }}」可建
            <b>{{ rankData.mine ? rankData.mine.city_max : '-' }}</b> 座，
            已有 <b>{{ cities.length }}</b> 座
            <span v-if="rankData.mine && cities.length >= rankData.mine.city_max" class="red">
              —— 已达上限，提升声望可解锁更多
            </span>
          </div>
          <div class="old-line">
            坐标X: <input v-model="newCityX" type="number" style="width:70px"/>
            坐标Y: <input v-model="newCityY" type="number" style="width:70px"/>
            <button @click="doCreateCity">建新城</button>
          </div>
          <!-- ★ 2026-09-25 去掉内联 font-size:14px，改为继承全站统一字号（--fs） -->
          <div class="gray">
            <b>平原</b> → 内陆城市; <b>沿海平原</b> → 沿海城市<br/>
            <!-- 沿海城市(可建航海协会、训练海军)。其他地形(含海洋)不能建城; 新城自带基础建筑(市政厅/民居/农田1级), 建造后可在上方列表切换操作。 -->
          </div>
        </div>
      </template>

      <!-- ============ 资源详情(res/:type) ============ -->
      <template v-else-if="cur === 'res'">
        <div class="panel" v-if="resDetail">
          资源详情:<br/>
          名称: {{ resNames[resType] }}<br/>
          储量: {{ resDetail.stock }} / 容纳: {{ resDetail.cap }}
          <span v-if="resDetail.store_tech > 0"> [储存技术Lv{{ resDetail.store_tech }}: 容量+{{ resDetail.store_tech * 2 }}%]</span>
          <br/>
          基础产量(每小时): {{ resDetail.base }}
          <span v-if="resDetail.tech_prod > 0"> [{{ resTechName }}Lv{{ resDetail.tech_prod }}: +{{ resDetail.tech_prod * 10 }}%]</span>
          <!-- ★ 2026-09-26：基础产量 = 建筑 × 科技 × 开工率（**民心/民怨不再影响产量**，
               用户要求）。「加成产量」只放市长后勤加成 + 野地 + 道具 + 活动，恒不为负。 -->
          <span class="gray" v-if="resDetail.base_building !== undefined">
            （建筑{{ resDetail.base_building }} × 科技{{ 100 + (resDetail.tech_prod || 0) * 10 }}%<template
              v-if="resDetail.rate !== undefined && resDetail.rate !== 100"> × 开工率{{ resDetail.rate }}%</template>）
          </span>
          <br/>
          加成产量(每小时): {{ resDetail.bonus }}
          <span class="gray" v-if="resDetail.mayor_bonus > 0"> [市长后勤加成+{{ resDetail.mayor_bonus }}%]</span>
          <!-- ★ 2026-09-30 用户要求「增产令使用了要在资源详情简约体现」：有增产效果时显示幅度 + 剩余时长 -->
          <span class="green" v-if="resDetail.boost_pct > 0">
            [增产令+{{ resDetail.boost_pct }}% {{ fmtLeft(Math.floor((resDetail.boost_until - Date.now()) / 1000)) }}]
          </span>
          <span class="gray" v-if="resDetail.bonus === 0"> [暂无加成]</span>
          <br/>
          耗量(每小时): {{ resDetail.consume }}<br/>
          <template v-if="resType === 'food'">
            军队耗粮: {{ fmtBig(resDetail.troop_consume_raw !== undefined ? resDetail.troop_consume_raw : (resDetail.troop_consume || 0)) }}
          <template v-if="resDetail.supply_tech > 0">
            [补给技巧Lv{{ resDetail.supply_tech }}: -{{ resDetail.supply_tech * 2 }}%
            → 实扣 {{ fmtBig(resDetail.troop_consume || 0) }}]
          </template>
          <span v-else class="gray"> [补给技巧未研究, 无减免]</span><br/>
          </template>
          总产量(每小时): <span :class="{ red: resDetail.total < 0 }">{{ resDetail.total }}</span><br/>
          <br/>
          {{ resDes[resType] }}
          <br/>
          <a href="javascript:;" @click="go('builds')">[资源区]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 建筑区(buildm/builds) ============ -->
      <!-- ============ 军事区 / 资源区(buildm / builds) 复刻 building/militaryIndex.html ============ -->
      <template v-else-if="cur === 'buildm' || cur === 'builds'">
        <div class="panel">
          <div class="old-line">
            {{ city.name }}({{ city.x }},{{ city.y }})
            <a href="javascript:;" @click="go('cities')">切换城市</a>
          </div>
          <!-- ★ 军事区/资源区导航固定顺序「军事区. 资源区」，当前项加粗高亮；不再谁当前谁排第一 -->
          <div class="old-line ezfy-subnav">
            <a href="javascript:;" :class="{ on: cur === 'buildm' }" @click="go('buildm')">军事区</a>.
            <a href="javascript:;" :class="{ on: cur === 'builds' }" @click="go('builds')">资源区</a>
          </div>
          <br/>
          <div class="old-line">建造中队列数：{{ buildQueueCount }}
            <a href="javascript:;" @click="openBuildPre(cur === 'buildm' ? 'm' : 's')">建造</a>
          </div>
          <!-- 已建建筑: 一行一个 —— 名称 (N级) 升级 一键9级 拆除 -->
          <div class="old-line" v-for="b in zoneBuilt" :key="'zb' + b.id">
            <span v-if="bEntry(b.building_id)">
              <a href="javascript:;" @click="goEntry(b.building_id)">{{ b.name }}</a>
            </span>
            <span v-else>{{ b.name }}</span>
            ({{ b.level }}级)
            <template v-if="b.status !== 0">
              <span class="orange">施工中 {{ remain(b.end_time) }}</span>
              <!-- ★ 2026-09-26 修复「加速道具买完实际使用不生效」：按背包里**实际拥有**的
                   建筑加速道具(item_type=3)逐档渲染，点哪档就用哪档（原来是自动挑最短的，
                   玩家买了 2 小时却只减 30 分钟，看着就像「买了没用上」）。 -->
              <a v-for="a in accItems(3)" :key="'sb' + b.id + '_' + a.cfg_id" href="javascript:;"
                 @click="doSpeedBuilding(b, a)">[加速{{ accLabel(a) }}]</a>
              <!-- ★ 2026-09-27 百分比加速道具(item_type=24)：剩余时间减 30%/60%/80% -->
              <a v-for="a in accItems(24)" :key="'sbp' + b.id + '_' + a.cfg_id" href="javascript:;"
                 @click="doSpeedBuilding(b, a)">[加速{{ accLabel(a) }}]</a>
              <span class="gray" v-if="!accItems(3).length && !accItems(24).length">(无建筑加速道具)</span>
              <!-- ★ 2026-09-27：升级状态时加 [取消]（取消零退还，建筑保留当前等级，防刷资源/图纸） -->
              <a href="javascript:;" class="red" @click="doCancelUpgrade(b)">[取消]</a>
            </template>
            <template v-else-if="b.level > 0 && b.level < b.max_level">
              <span class="build-act">
                <a href="javascript:;" @click="doUpgrade(b)">升级</a>
                <!-- ★ 2026-09-25 用户纠正：按钮语义是「一键升级到 max_level-1 级」，
                     已经到达该等级就不再显示（原来会显示成 9 级但实际升满级）
                     ★ 2026-09-30 用户要求：一键只到 9 级（司令部 12 级时原显示「一键11级」，
                     现统一「一键9级」，10/11/12 级手动升级，每级都要建筑图纸） -->
                <a v-if="b.level < 9" href="javascript:;" @click="doMaxLevel(b)">一键9级</a>
                <a v-if="b.can_delete === 1" href="javascript:;" @click="doDeleteBuilding(b)">拆除</a>
              </span>
            </template>
            <template v-else>
              <a v-if="b.level === 0" href="javascript:;" @click="doUpgrade(b)">建成中待完成</a>
              <a v-else-if="b.can_delete === 1" href="javascript:;" @click="doDeleteBuilding(b)">拆除</a>
            </template>
            <div v-if="inlineTip && inlineTip.bid === b.id" class="build-tip" :class="inlineTip.type">
              <span>{{ inlineTip.text }}</span>
              <a href="javascript:;" @click="inlineTip = null">[关闭]</a>
            </div>
          </div>
          <div class="old-line gray" v-if="!zoneBuilt.length">(本区还没有建筑, 点上面的「建造」)</div>
          <br/>
          <div class="old-line">
            <a href="javascript:;" @click="doSpeedTrainAll">[训练一键加速(消耗黄金)]</a>|
            <a href="javascript:;" @click="doSpeedTrainAllCity">[所有城市训练一键加速(消耗黄金)]</a>
          </div>
          <!-- ★ 训练加速道具(item_type=4)的入口：上面两个是「花黄金一键完成」，
               这里才是商城买的「训练加速30分钟/2小时」真正被消耗的地方。 -->
          <div class="old-line" v-if="accItems(4).length || accItems(25).length">
            训练加速道具:
            <a v-for="a in accItems(4)" :key="'stb' + a.cfg_id" href="javascript:;"
               @click="doSpeedTrain(null, a)">[加速{{ accLabel(a) }}]×{{ a.count }}</a>
            <!-- ★ 2026-09-27 百分比训练加速(item_type=25)：剩余时间减 30%/60%/80% -->
            <a v-for="a in accItems(25)" :key="'stbp' + a.cfg_id" href="javascript:;"
               @click="doSpeedTrain(null, a)">[加速{{ accLabel(a) }}]×{{ a.count }}</a>
          </div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 建造页(buildpre) 复刻 building/preCreateMilitary.html / preCreateSource.html ============ -->
      <template v-else-if="cur === 'buildpre'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="go(buildZone === 'm' ? 'buildm' : 'builds')">
              {{ buildZone === 'm' ? '军事区' : '资源区' }}</a> 建造
          </div>
          <div class="old-line">可建造的建筑：</div>
          <div class="old-line" v-for="b in zonePool" :key="'bp' + b.building_id">
            {{ b.name }}
            <a href="javascript:;" @click="openBuildDetail(b)">详情</a>
            <a href="javascript:;" @click="doBuild(b)">建造</a>
          </div>
          <div class="old-line gray" v-if="!zonePool.length">(本区暂无可建造的建筑)</div>

          <!-- 选中建筑后展开: 说明 + 造价 -->
          <template v-if="buildSel">
            <hr/>
            <div class="panel-title">{{ buildSel.name }}</div>
            <div class="old-line">{{ buildSel.des }}</div>
            <div class="old-line gray">
              造价: {{ resShort.food }}{{ buildSel.cost.food }} {{ resShort.steel }}{{ buildSel.cost.steel }} {{ resShort.oil }}{{ buildSel.cost.oil }}
              {{ resShort.rare }}{{ buildSel.cost.rare }} {{ resShort.gold }}{{ buildSel.cost.gold }}
              需{{ Math.ceil(buildSel.time / 60) }}分钟
            </div>
            <div class="old-line">
              <button @click="doBuild(buildSel)">[建造]</button>
              <a href="javascript:;" @click="buildSel = null">[收起]</a>
            </div>
          </template>
          <a href="javascript:;" @click="go(buildZone === 'm' ? 'buildm' : 'builds')">[返回{{ buildZone === 'm' ? '军事区' : '资源区' }}]</a>
        </div>
      </template>

      <!-- ============ 造兵(troop) ============ -->
      <template v-else-if="cur === 'troop'">
        <div class="panel">
          <div class="panel-title">训练军队(军工厂合计{{ factoryTotal }}级, 队列{{ queues.length }}/{{ factoryTotal }})</div>
          <div class="old-line green" v-if="troopsData.train_discount > 0">
            节日活动·造兵打折：资源消耗 -{{ troopsData.train_discount }}%（下方为折后价）
          </div>
          <div class="old-line">人口:{{ troopsData.pop }} 空闲:{{ freePop }} | 围墙:{{ troopsData.wall_level }}级</div>
          <!-- ★ 空闲人口 = 人口 - 占用；占用只算「训练中、还没出厂」的新兵。
               已训练完成的部队（含出征在外的）不占人口位置 —— 用户 2026-09-21 明确的规则 -->
          <div class="old-line gray" v-if="popUsed > 0">
            训练中占用人口：{{ popUsed }}
          </div>
          <div class="old-line" v-for="t in trainCfgs" :key="'tt' + t.id">
            <!-- ★ 2026-09-28 用户要求：军队列表只留名称和类型，属性/消耗/前提都进[训练]详情页看 -->
            <a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a>({{ troopTypeName(t.type) }})
            <a href="javascript:;" @click="openTrainPre(t, 'troop')">[训练]</a><br/>
          </div>
          <div class="panel-title">训练队列({{ queues.length }})</div>
          <div class="old-line" v-for="q in queues" :key="'q' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
            <!-- 接口只返回 status=0（训练中）的队列，所以这里不需要再判断状态 -->
            <!-- ★ 2026-09-26 修复：训练页原来只有 [取消]，商城买的「训练加速」道具无处可用 -->
            <a v-for="a in accItems(4)" :key="'sq' + q.id + '_' + a.cfg_id" href="javascript:;"
               @click="doSpeedTrain(q, a)">[加速{{ accLabel(a) }}]</a>
            <!-- ★ 2026-09-27 百分比训练加速(item_type=25) -->
            <a v-for="a in accItems(25)" :key="'sqp' + q.id + '_' + a.cfg_id" href="javascript:;"
               @click="doSpeedTrain(q, a)">[加速{{ accLabel(a) }}]</a>
            <span class="gray" v-if="!accItems(4).length && !accItems(25).length">(无训练加速道具)</span>
            <a href="javascript:;" @click="doCancelTrain(q)">[取消]</a>
          </div>
          <div class="old-line" v-if="!queues.length">(队列为空)</div>
          <a href="javascript:;" @click="go('defence')">[去建城防]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 城防(defence) 复刻 city/troopDefence.html ============ -->
      <template v-else-if="cur === 'defence'">
        <div class="panel">
          <div class="old-line">
            围墙：{{ troopsData.wall_level }}级 城防空间：({{ troopsData.defence_space_used }}/{{ troopsData.defence_space }})
          </div>
          <div class="old-line">正在建造:</div>
          <div class="old-line" v-for="q in defenceQueues" :key="'dq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
          </div>
          <div class="old-line gray" v-if="!defenceQueues.length">(无)</div>
          <table>
            <tr v-for="t in defenceCfgs" :key="'dt' + t.id">
              <td class="nm"><a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a>:</td>
              <td>{{ troopCount(t.id) }}</td>
              <td>
                <a href="javascript:;" @click="openTrainPre(t, 'defence')">[建造]</a>
                <a href="javascript:;" @click="doDismiss(t)">[拆除]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray">城防设施占用「城防空间」(围墙容量)，不占用人口。</div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军队总览(troops) ============ -->
      <template v-else-if="cur === 'troops'">
        <div class="panel">
          <div class="panel-title">城内军队</div>
          <!-- ★ 用户要求：这张表数据「上下居中、左右居中」，操作列也一起对齐 -->
          <table class="ezfy-center-tbl">
            <tr><th class="nm">兵种</th><th>数量</th><th>操作</th></tr>
            <!-- ★ 2026-09-28 用户要求：首页点「军队」要能看到全部兵种（数量为 0 的也显示），每行后跟训练操作 -->
            <tr v-for="t in armyRows" :key="'tv' + t.id">
              <td class="nm"><a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a></td>
              <td>{{ t.count }}</td>
              <td>
                <!-- 训练/建造：防御兵种(type 4)走城防建造，其余直接训练 -->
                <a href="javascript:;" @click="openTrainPre(t, t.type === 4 ? 'defence' : 'troop')">[{{ t.type === 4 ? '建造' : '训练' }}]</a>
                <!-- 解散：数量由玩家自己输入（用户要求），数量为 0 时无意义、不显示 -->
                <a v-if="t.count > 0" class="red" href="javascript:;" @click="doDisband(t)">[解散]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!armyRows.length">(暂无兵种配置)</div>
          <br/>
          <div class="panel-title">训练队列({{ queues.length }})</div>
          <div class="old-line" v-for="q in queues" :key="'tq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
            <!-- 接口只返回 status=0（训练中）的队列，所以这里不需要再判断状态 -->
            <a href="javascript:;" @click="doCancelTrain(q)">[取消]</a>
          </div>
          <div class="old-line" v-if="!queues.length">(队列为空)</div>
          <br/>
          <a href="javascript:;" @click="go('troop')">[造兵]</a>
          <a href="javascript:;" @click="go('defence')">[建防]</a>
          <a href="javascript:;" @click="go('hq')">[司令部]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 司令部(hq) ============ -->
      <template v-else-if="cur === 'hq'">
        <div class="panel">
          <div class="panel-title">司令部</div>
          <!-- ★ 2026-09-28 用户要求：司令部内部拆成 tab（兵种配置/出征队列/伤兵营/逃兵营），
               刷新后记住上次所在 tab（localStorage, 照抄任务 tab 的 ezfy_task_tab 写法） -->
          <div class="acade-tab hq-tab">
            <a href="javascript:;" :class="{ on: hqTab === 0 }" @click="selectHqTab(0)">兵种配置</a>
            <a href="javascript:;" :class="{ on: hqTab === 1 }" @click="selectHqTab(1)">出征队列({{ orders.length }})</a>
            <a href="javascript:;" :class="{ on: hqTab === 2 }" @click="selectHqTab(2)">伤兵营({{ woundedList(0).length }})</a>
            <a href="javascript:;" :class="{ on: hqTab === 3 }" @click="selectHqTab(3)">逃兵营({{ woundedList(1).length }})</a>
            <a href="javascript:;" :class="{ on: hqTab === 4 }" @click="selectHqTab(4)">预设编队({{ presets.length }})</a>
          </div>
          <div class="panel-title" v-show="hqTab === 0">兵种战斗配置</div>
          <div v-show="hqTab === 0">
          <!-- ★ 一个兵种一块（原来 5 列固定宽度表格在手机上会互相遮盖） -->
          <div class="ezfy-tgt-block" v-for="t in troopsData.cfgs" :key="'cfg' + t.id">
            <div class="ezfy-tgt-name">
              {{ t.name }}
              <span v-if="isDefenceTroop(t)" class="gray">（防御兵种：固定阵地，不能前进/后退，也不能出征）</span>
            </div>
            <div class="ezfy-tgt-row">
              <span class="ezfy-tgt-lab">进攻目标</span>
              <select v-model="targetCfg[t.id].atk" style="width:110px">
                <option :value="0">最近目标</option>
                <option v-for="tt in troopsData.cfgs" :key="'a' + tt.id" :value="tt.id">{{ tt.name }}</option>
              </select>
              <span class="ezfy-tgt-lab">进攻</span>
              <select v-model="targetCfg[t.id].atkMove" style="width:80px"
                      :disabled="isDefenceTroop(t)">
                <option :value="1">前进</option><option :value="0">停止</option>
              </select>
            </div>
            <div class="ezfy-tgt-row">
              <span class="ezfy-tgt-lab">防守目标</span>
              <select v-model="targetCfg[t.id].def" style="width:110px">
                <option :value="0">最近目标</option>
                <option v-for="tt in troopsData.cfgs" :key="'d' + tt.id" :value="tt.id">{{ tt.name }}</option>
              </select>
              <span class="ezfy-tgt-lab">防守</span>
              <select v-model="targetCfg[t.id].defMove" style="width:110px"
                      :disabled="isDefenceTroop(t)">
                <option :value="1">前进</option><option :value="0">停止</option><option :value="-1">不参与防御</option>
              </select>
            </div>
          </div>
          <div class="old-line"><a href="javascript:;" @click="doSaveTargets">[保存全部配置]</a></div>
          </div><!-- /兵种配置 tab -->
          <div v-show="hqTab === 1">
          <div class="panel-title">出征队列({{ orders.length }})</div>
          <table>
            <tr><th>类型</th><th>目标</th><th>统帅</th><th>状态</th><th></th></tr>
            <tr v-for="o in orders" :key="'o' + o.id">
              <td>{{ o.type_name }}</td>
              <td>({{ o.target_x }},{{ o.target_y }})</td>
              <td>{{ o.officer || '无' }}</td>
              <td>{{ orderStatusText(o) }}</td>
              <td>
                <a href="javascript:;" @click="openOrder(o)">[详情]</a>
                <!-- ★ 用户要求「出征队列可以取消」：所有还在外面的命令（行进中/驻守中）都能取消 -->
                <a v-if="o.status === 0 || o.status === 1" class="red"
                   href="javascript:;" @click="doRecall(o)">[取消]</a>
                <!-- ★ 2026-09-30 行军计谋：神兵天降=去程减80%（行进中）、战略转移=回程减360分钟（返回中） -->
                <a v-if="o.status === 0 && o.scheme_fast === 0" class="green"
                   href="javascript:;" @click="doMarchScheme(o, 13)">[神兵天降]</a>
                <a v-if="o.status === 0 && o.scheme_fast === 1" class="gray">[神兵天降·已用]</a>
                <a v-if="o.status === 2 && o.scheme_back === 0" class="green"
                   href="javascript:;" @click="doMarchScheme(o, 14)">[战略转移]</a>
                <a v-if="o.status === 2 && o.scheme_back === 1" class="gray">[战略转移·已用]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!orders.length">(暂无出征部队)</div>
          </div><!-- /出征队列 tab -->
          <div v-show="hqTab === 2">
          <div class="panel-title">伤兵营</div>
          <div class="old-line gray">伤兵在营中<b>不消耗粮食</b>；恢复出厂需要黄金（按兵种造价折算）。</div>
          <table>
            <tr><th class="nm">兵种</th><th>数量</th><th>恢复费用</th><th>操作</th></tr>
            <tr v-for="w in woundedList(0)" :key="'w' + w.id">
              <td class="nm">{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ fmtN((w.heal_gold || 0) * w.count) }} {{ resNames.gold }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[恢复]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(0).length">(伤兵营无伤兵)</div>
          <div class="old-line" v-if="woundedList(0).length">
            合计 <b>{{ fmtN(woundedHealCost(0)) }}</b> {{ resNames.gold }}
            <button @click="doRecoverAll(0)">[全部恢复]</button>
          </div>
          </div><!-- /伤兵营 tab -->
          <div v-show="hqTab === 3">
          <div class="panel-title">逃兵营</div>
          <table>
            <tr><th class="nm">兵种</th><th>数量</th><th>召回费用</th><th>操作</th></tr>
            <tr v-for="w in woundedList(1)" :key="'dsw' + w.id">
              <td class="nm">{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ fmtN((w.heal_gold || 0) * w.count) }} {{ resNames.gold }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[召回]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(1).length">(逃兵营无逃兵)</div>
          <div class="old-line" v-if="woundedList(1).length">
            合计 <b>{{ fmtN(woundedHealCost(1)) }}</b> {{ resNames.gold }}
            <button @click="doRecoverAll(1)">[全部召回]</button>
          </div>
          </div><!-- /逃兵营 tab -->
          <div v-show="hqTab === 4">
          <!-- ★ 2026-09-28 用户要求：司令部新增「预设编队」tab —— 镜像出征页 ①②③⑥ 保存模板，
               不含 ④随军资源 / ⑤宿营；⑥油耗计算保留（预设不含目标，按本城0距离估算）。 -->
          <div class="panel-title">预设编队</div>
          <div class="old-line gray">
            预设 = 出征模板（指挥军官 + 集结令 + 兵力），不含随军资源/宿营。
            保存后在<b>地图出征页</b>的「预设编队」下拉里选用，一键回填。
          </div>
          <template v-if="presetAdding">
            <div class="of-sec">新增预设 · 名称
              <input v-model="presetName" type="text" maxlength="20" placeholder="(最多20字)" style="width:140px"/>
            </div>
            <!-- ① 军官 -->
            <div class="of-sec">① 指挥军官</div>
            <div class="old-line">
              <select v-model="orderOfficer" @change="presetCalc">
                <option value="0">未指定</option>
                <option v-for="o in onDutyOfficers" :key="'pd' + o.id" :value="o.name">
                  {{ o.name }}({{ o.level }}级) 军{{ o.military_total || o.military }}
                  <span class="green" v-if="o.equip_military">(装+{{ o.equip_military }})</span>
                  学{{ o.learning_total || o.learning }} 后{{ o.logistics_total || o.logistics }}
                </option>
              </select>
              <span v-if="!onDutyOfficers.length" class="gray">(「{{ city.name }}」暂无可带队军官)</span>
            </div>
            <!-- ② 集结令 -->
            <div class="of-sec">② 出征集结令</div>
            <div class="old-line">
              使用
              <input type="number" min="0" :max="gatherMax" v-model.number="orderGather"
                     :disabled="gatherCount <= 0" @change="onPresetGatherChange" style="width:80px"/>
              个 <span class="gray">（背包里有 {{ gatherCount }} 个，单次最多 {{ orderCapMax }} 个）</span>
            </div>
            <div class="old-line gray">
              每个集结令 +{{ fmtN(orderCapPer) }} 出征上限，单次最多 {{ orderCapMax }} 个。
              司令部上限（含指挥艺术科技）+ 集结令 + 出征军官军事属性<b>叠加</b>。
            </div>
            <!-- ★ 2026-09-29 用户要求：预设页与出征页一致，集结令下方直接显示「本次出兵 / 上限」 -->
            <div class="old-line" v-if="attackTroops.length">
              <span :class="orderOverCap ? 'red' : 'green'">
                本次出兵 <b>{{ fmtN(orderTroopTotal) }}</b> / 上限 <b>{{ orderCapText }}</b>
                <template v-if="orderOverCap">—— 超出上限，请减少兵力或加用集结令</template>
              </span>
            </div>
            <!-- ③ 兵力 -->
            <div class="of-sec">③ 选择兵力
              <span class="of-hint">（拖滑块或直接填数字；滑块与 [最大] 都按「城内现有」和「出征上限剩余」取小）</span>
            </div>
            <div class="of-rows">
              <div class="of-row" v-for="t in trainCfgs" :key="'pt' + t.id"
                   :class="{ 'of-off': troopCount(t.id) <= 0 }"
                   :title="t.name + '（现有 ' + fmtN(troopCount(t.id)) + '，本次最多可派 ' + fmtN(orderQtyMax(t.id)) + '）'">
                <span class="of-name">{{ t.name }}</span>
                <span class="of-avail">现有 {{ fmtN(troopCount(t.id)) }}</span>
                <span class="of-ctl">
                  <input type="range" class="of-range" min="0" step="1"
                          :max="troopCount(t.id)" :value="orderQty(t.id)"
                          :disabled="orderQtyMax(t.id) <= 0"
                          @input="onOrderQtyInput(t.id, $event)"/>
                  <input type="number" class="of-num" min="0" placeholder="0"
                         :max="orderQtyMax(t.id)" :value="orderQty(t.id)"
                         :disabled="orderQtyMax(t.id) <= 0"
                         @input="onOrderQtyInput(t.id, $event)"/>
                  <a href="javascript:;" class="of-max"
                     :class="{ 'of-max-off': orderQtyMax(t.id) <= 0 }"
                     @click="setOrderQtyMax(t.id)">[最大]</a>
                </span>
              </div>
            </div>
            <!-- ⑥ 消耗预览（不含目标 → 按本城 0 距离估算） -->
            <div class="of-sec">⑥ 消耗预览</div>
            <div class="old-line">
              <button @click="presetCalc">[计算]</button>
              油耗：<span class="orange">{{ orderCalc ? orderCalc.oil_used : '—' }}</span>
              &nbsp;/&nbsp;负重：<span class="orange">{{ orderCalc ? orderCalc.carry : '—' }}</span>
            </div>
            <div class="old-line gray">
              （预设不含目标，油耗/负重按本城 0 距离估算；实际油耗与耗时以出征页 [计算] 为准）
            </div>
            <div class="old-line">
              <button @click="savePreset">[保存预设]</button>
              <a href="javascript:;" @click="cancelPresetAdd">[取消]</a>
            </div>
          </template>
          <div class="old-line" v-else>
            <a href="javascript:;" @click="startPresetAdd">[新增预设编队]</a>
            <span class="gray">（最多保存 {{ presetMax }} 个）</span>
          </div>
          <div class="panel-title">我的预设</div>
          <table>
            <tr><th class="nm">名称</th><th>军官</th><th>兵力</th><th>集结令</th><th></th></tr>
            <tr v-for="p in presets" :key="'psl' + p.id">
              <td class="nm">{{ p.name }}</td>
              <td>{{ p.officer || '无' }}</td>
              <td>{{ fmtN(p.troop_total) }}</td>
              <td>{{ p.gather }}个</td>
              <td><a href="javascript:;" class="red" @click="deletePreset(p)">[删除]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!presets.length">(还没有预设编队，点上方 [新增预设编队] 开始)</div>
          </div><!-- /预设编队 tab -->
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 科技(techs) ============ -->
      <template v-else-if="cur === 'techs'">
        <div class="panel">
          <div class="panel-title">【科技中心】:{{ techsData.academy }}级</div>
          <div class="old-line" v-for="t in techsData.techs" :key="'te' + t.tech_id">
            <!-- ★ 2026-09-28 用户要求：科技列表不再显示资源消耗/前置条件，点[研究]进详情页查看 -->
            <!-- ★ 2026-09-28 用户要求版式：第一行「名称 等级/满级级 [研究N级]」，第二行才是效果。
                 原来 [研究N级] 被挤在效果下面第三行，扫一眼看不出「这条能不能升」。 -->
            <b>{{ t.name }}</b> {{ t.level }}/{{ t.max_level }}级
            <span v-if="t.researching" class="orange">研究中 {{ remain(t.end_time) }}
              <!-- ★ 2026-09-26 修复「科技加速道具买完实际使用不生效」：原来这里的 [加速]
                   调的是 /techs/speed，是**免费减 10 分钟**（minutes 还能由前端随便传），
                   商城买的「科技加速30分钟/2小时」根本没被消耗。现在改为消耗道具。 -->
              <a v-for="a in accItems(5)" :key="'st' + t.tech_id + '_' + a.cfg_id" href="javascript:;"
                 @click="doSpeedTech(a)">[加速{{ accLabel(a) }}]</a>
              <!-- ★ 2026-09-27 百分比科技加速(item_type=26)：剩余时间减 30%/60%/80% -->
              <a v-for="a in accItems(26)" :key="'stp' + t.tech_id + '_' + a.cfg_id" href="javascript:;"
                 @click="doSpeedTech(a)">[加速{{ accLabel(a) }}]</a>
              <span class="gray" v-if="!accItems(5).length && !accItems(26).length">(无科技加速道具)</span>
              <a href="javascript:;" @click="doCancelTech(t)">[取消]</a></span>
            <span v-else-if="t.level < t.max_level">
              <a href="javascript:;" @click="openTechPre(t)">[研究{{ t.level + 1 }}级]</a>
            </span>
            <span v-else class="gray">[已满级]</span>
            <br/>
            {{ t.effect }}<br/>
          </div>
          <div class="old-line gray">不同城市可同时研究不同科技；同一科技只能在一个城市研究；[取消] 只停止研究，不退还已消耗资源。</div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 科技详情(techpre) 2026-09-28 用户要求：点[研究]进详情页查看资源/条件再确认 ============ -->
      <template v-else-if="cur === 'techpre'">
        <div class="panel" v-if="techSel">
          <div class="panel-title">研究「{{ techSel.name }}」{{ techSel.level + 1 }}级</div>
          <div class="old-line">当前等级：{{ techSel.level }}/{{ techSel.max_level }}级</div>
          <div class="old-line">效果：{{ techSel.effect }}</div>
          <div class="old-line">前提：科研中心{{ techSel.academy_need }}级
            <span class="gray">(本城 {{ techSel.academy }} 级)</span></div>
          <div class="old-line">
            所需资源：{{ resNames.food }}{{ techSel.next_cost.food }} {{ resNames.steel }}{{ techSel.next_cost.steel }}
            {{ resNames.oil }}{{ techSel.next_cost.oil }} {{ resNames.rare }}{{ techSel.next_cost.rare }}
            {{ resNames.gold }}{{ techSel.next_cost.gold }}
          </div>
          <div class="old-line">耗时：{{ Math.ceil(techSel.next_time / 60) }}分钟</div>
          <div class="old-line">
            <a href="javascript:;" @click="doTechPre()">[开始研究]</a>
          </div>
          <a href="javascript:;" @click="go('techs')">[返回科技列表]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先选择科技 <a href="javascript:;" @click="go('techs')">[科技列表]</a></div>
        </div>
      </template>

      <!-- ============ 地图(map) ============ -->
      <template v-else-if="cur === 'map'">
        <div class="panel">
          <!-- 复刻 map/index.html 游戏区: 城市行 → 地图：→ 坐标查找 → 精英城市 → 5×5 表格 → 当前中心 → 方向 -->
          <div class="old-line">
            {{ city.name }}({{ city.x }},{{ city.y }})
            <a href="javascript:;" @click="go('cities')">切换城市</a>
          </div>
          <div class="old-line">地图：</div>
          <div class="old-line">
            输入坐标查找：
            <a href="javascript:;" @click="toggleStars">收藏列表</a>
          </div>
          <!-- ★ 用户要求「横坐标、纵坐标 一行」：两个输入框合并同一行，[查找] 跟在行末 -->
          <div class="old-line ezfy-map-jump">
            横坐标：<input v-model="jumpX" type="number" placeholder="(1~500)"/>
            纵坐标：<input v-model="jumpY" type="number" placeholder="(1~500)"/>
            <button @click="doJump">[查找]</button>
          </div>
          <div class="old-line" v-if="eliteCell">
            发现精英中立城市：
            <a class="red" href="javascript:;" @click="openElite">[寇({{ eliteCell.x }},{{ eliteCell.y }})]</a>
          </div>
          <template v-if="showStars">
            <div class="panel-title">收藏列表</div>
            <div class="old-line" v-for="s in mapStars" :key="'st' + s.id">
              <a href="javascript:;" @click="jumpTo(s.x, s.y)">{{ s.name }}({{ s.x }},{{ s.y }})</a>
              <a href="javascript:;" @click="delStar(s)">[删除]</a>
            </div>
            <div class="old-line gray" v-if="!mapStars.length">(收藏列表为空, 在地图上选中目标后可收藏)</div>
          </template>
          <!-- ★ 用户要求「格子下面加个坐标，排列整齐一点」：
               每格两行 —— 第一行名称(等级)，第二行 (x,y)；
               第二行用站内链接蓝 #0645ad，让玩家一眼知道格子能点。 -->
          <table class="ezfy-map-table">
            <tr v-for="(row, ri) in mapRows" :key="'mr' + ri">
              <td v-for="cell in row" :key="cell.x + '_' + cell.y">
                <!-- ★ 2026-10-01 修复「地图输入 1,1 跳转后还有负号坐标」：
                     地图世界 500×500，有效坐标 1~499；视野中心靠边时周边格子会越界，
                     这些不存在的格子不再显示负坐标地形，统一按「空地」展示（不可点）。 -->
                <a v-if="!isMapOOB(cell)" href="javascript:;" :class="cellClass(cell)" :title="cellTip(cell)" @click="openCell(cell)">
                  <span class="ezfy-cell-name">{{ cellText(cell) }}</span>
                  <span class="ezfy-cell-xy">({{ cell.x }},{{ cell.y }})</span>
                </a>
                <span v-else class="ezfy-empty">{{ cellText(cell) }}</span>
              </td>
            </tr>
          </table>
          <div class="old-line">当前坐标中心:({{ mapCx }} , {{ mapCy }})</div>
          <!-- ★ 用户要求「向上/向右/向下/向左/回到本城 间隙稍微大一点」→ 见 .ezfy-dir-nav a -->
          <div class="old-line ezfy-dir-nav">
            <a href="javascript:;" @click="moveMap(-mapStep, 0)">向上</a>
            <a href="javascript:;" @click="moveMap(0, mapStep)">向右</a>
            <a href="javascript:;" @click="moveMap(mapStep, 0)">向下</a>
            <a href="javascript:;" @click="moveMap(0, -mapStep)">向左</a>
            <a href="javascript:;" @click="loadMap()">回到本城</a>
          </div>
          <div class="old-line gray">
            城=城市 寇=寇城 墟=废墟 海=海洋 括号内为等级; 点格子进入目标详情
          </div>
          <div class="old-line gray">
            <span class="orange">活动</span>=活动野地 <span style="color:#ff00ff">活动寇</span>=活动寇城
            <span class="red">特殊</span>=特殊城市
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('orders')">出征队列</a>
          </div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军工厂(factory) 复刻 city/cityFactory.html ============ -->
      <template v-else-if="cur === 'factory'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="go('buildm')">军事区</a>
            -&gt;军营(军工厂)({{ factoryTotal }}级)：
          </div>
          <div class="old-line">正在训练：</div>
          <div class="old-line" v-for="q in queues" :key="'fq' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
          </div>
          <div class="old-line gray" v-if="!queues.length">(无)</div>
          <div class="old-line" v-for="t in trainCfgs" :key="'ft' + t.id">
            <a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a> : {{ troopCount(t.id) }}
            <a href="javascript:;" @click="openTrainPre(t, 'troop')">[训练]</a>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('troops')">[城内军队]</a>
            <a href="javascript:;" @click="go('buildm')">[返回军事区]</a>
            <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
        </div>
      </template>

      <!-- ============ 兵种详情(troopview) 复刻 city/cityTroopView.html ============ -->
      <template v-else-if="cur === 'troopview'">
        <div class="panel" v-if="troopView">
          <div class="old-line">
            {{ troopView.name }}:
            <span class="gray">(现有 {{ troopCount(troopView.id) }})</span>
          </div>
          <div class="old-line">训练/建造需要：</div>
          <table>
            <tr>
              <td>{{ resNames.food }}：</td><td>{{ troopView.cost.food }}</td>
              <td>{{ resNames.steel }}：</td><td>{{ troopView.cost.steel }}</td>
            </tr>
            <tr>
              <td>{{ resNames.oil }}：</td><td>{{ troopView.cost.oil }}</td>
              <td>{{ resNames.rare }}：</td><td>{{ troopView.cost.rare }}</td>
            </tr>
            <tr>
              <td>耗时：</td><td>{{ durText(troopView.train_time) }}</td>
              <td>油耗：</td><td>{{ troopView.oil_keep }}</td>
            </tr>
            <tr>
              <td>耗粮：</td><td>{{ troopView.food_keep }}</td>
              <td>人口：</td><td>{{ troopView.pop }}</td>
            </tr>
            <tr>
              <td>生命：</td><td>{{ troopView.health }}</td>
              <td>对地：</td><td>{{ troopView.atk_ground }}</td>
            </tr>
            <tr>
              <td>对空：</td><td>{{ troopView.atk_air }}</td>
              <td>对海：</td><td>{{ troopView.atk_sea }}</td>
            </tr>
            <tr>
              <td>对防：</td><td>{{ troopView.atk_def }}</td>
              <td>速度：</td><td>{{ troopView.speed }}</td>
            </tr>
            <tr>
              <td>军种：</td><td>{{ troopTypeName(troopView.type) }}</td>
              <td>射程：</td><td>{{ troopView.attack_range }}</td>
            </tr>
            <tr>
              <td>攻速：</td><td>{{ troopView.speed }}</td>
              <td>负重：</td><td>{{ troopView.carry }}</td>
            </tr>
            <tr>
              <td>防御：</td><td>{{ troopView.defence }}</td>
              <td>修复率：</td><td>{{ troopView.repair_rate }}%</td>
            </tr>
          </table>
          <div class="old-line">
            {{ troopView.type === 4 ? '围墙' : '军工厂' }}: {{ troopView.type === 4 ? troopsData.wall_level : factoryTotal }}级
            <template v-if="troopView.require"><br/>前提: {{ troopView.require }}</template>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="openTrainPre(troopView, troopView.type === 4 ? 'defence' : 'troop')">
              [{{ troopView.type === 4 ? '建造' : '训练' }}]
            </a>
          </div>
          <a href="javascript:;" @click="go(troopViewBack)">[返回]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请选择兵种 <a href="javascript:;" @click="go('troops')">[城内军队]</a></div>
        </div>
      </template>

      <!-- ============ 训练确认(trainpre) 复刻 city/createTroop.html / city/createDefence.html ============ -->
      <template v-else-if="cur === 'trainpre'">
        <div class="panel" v-if="trainSel">
          <div class="old-line" v-if="trainMode === 'defence'">
            【城防建造】城防空间:{{ troopsData.defence_space_used }}/{{ troopsData.defence_space }}
          </div>
          <div class="old-line" v-else>
            <a href="javascript:;" @click="go('buildm')">军事区</a>
            -&gt;军工厂({{ factoryTotal }}级)：
          </div>
          <div class="old-line">{{ trainSel.name }}{{ trainMode === 'defence' ? '建造' : '训练' }}需求：</div>
          <div class="old-line">
            {{ resNames.food }}：{{ trainSel.cost.food }}<br/>
            {{ resNames.steel }}：{{ trainSel.cost.steel }}<br/>
            {{ resNames.oil }}：{{ trainSel.cost.oil }}<br/>
            {{ resNames.rare }}：{{ trainSel.cost.rare }}<br/>
            {{ trainMode === 'defence' ? '城防空间' : '人口' }}：{{ trainSel.pop }}<br/>
            <template v-if="trainMode !== 'defence'">吃粮：{{ trainSel.food_keep }}<br/></template>
            时间：{{ durText(trainSel.train_time) }}<br/>
            <template v-if="trainMode !== 'defence'">需要军工厂：{{ trainSel.need_factory }}级<br/></template>
            <!-- ★ 2026-09-28 用户要求：资源/前提条件移到这里（训练详情页）展示 -->
            前提：{{ trainSel.require || '无' }}<template v-if="trainSel.type === 1"> <span class="red">(海军: 仅海城可训练)</span></template><br/>
          </div>
          <div class="old-line green" v-if="troopsData.train_discount > 0 && trainMode !== 'defence'">
            节日活动·造兵打折：资源消耗 -{{ troopsData.train_discount }}%（上方为折后价）
          </div>
          <div class="old-line">
            建造数量：
            <input v-model="trainCount" type="number" min="1" :placeholder="'(1~' + maxTrainable + ')'" style="width:90px"/>
            <span class="gray">(最多 {{ maxTrainable }})</span>
            <!-- ★ 2026-09-28 用户要求：在「(最多 N)」后面加 [最大]，一键把数量填成上限 -->
            <a href="javascript:;" class="train-max"
               :class="{ 'train-max-off': maxTrainable <= 0 }"
               @click="setTrainMax()">[最大]</a>
          </div>
          <div class="old-line red" v-if="maxTrainable <= 0">
            当前无法{{ trainMode === 'defence' ? '建造' : '训练' }}：资源或{{ trainMode === 'defence' ? '城防空间' : '人口' }}不足
            <template v-if="trainMode !== 'defence'">
              <a href="javascript:;" @click="go('home')">[回首页召集人口]</a>
            </template>
            <template v-else>
              <a href="javascript:;" @click="go('buildm')">[去军事区升级围墙]</a>
            </template>
          </div>
          <div class="old-line">
            <span>操作选项：</span>
            <label><input type="radio" :value="true" v-model="trainSplit"/>[全部工厂]</label>
            <label><input type="radio" :value="false" v-model="trainSplit"/>[仅此工厂]</label>
          </div>
          <div class="old-line">预计耗时：{{ trainEstimateText }}</div>
          <div class="old-line">
            <button @click="doTrainPre()">{{ trainMode === 'defence' ? '开始建造' : '开始训练' }}</button>
          </div>
          <a href="javascript:;" @click="go(trainMode === 'defence' ? 'defence' : 'factory')">[返回]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先选择兵种 <a href="javascript:;" @click="go('troops')">[城内军队]</a></div>
        </div>
      </template>

      <!-- ============ 目标详情(wildview) 复刻 map/mapView.html ============ -->
      <template v-else-if="cur === 'wildview'">
        <div class="panel" v-if="selCell">
          <div class="old-line" v-if="selDetail">
            所属区域：{{ selDetail.continent }}
            <a href="javascript:;" @click="addStar">{{ isCellStarred ? '已收藏' : '收藏' }}</a>
          </div>
          <template v-if="selDetail">
            <!-- 活动目标(活动野地/活动寇城/特殊城市): 复刻 activityIndex.html 的说明 + 守军/奖励预览 -->
            <template v-if="selDetail.act_type">
              <div class="old-line orange">
                {{ selDetail.act_name }}{{ selDetail.act_level }}级 ({{ selCell.x }},{{ selCell.y }}) —— {{ selDetail.act_desc }}
              </div>
              <div class="old-line" v-if="selDetail.officer_name">
                守将：<span :class="selDetail.officer_kind === 2 ? 'orange' : ''"><b>{{ selDetail.officer_name }}</b></span>
                <span v-if="selDetail.officer_kind === 2" class="orange">（名将</span>
                <span v-else class="gray">（普通军官</span>{{ selDetail.officer_star }}★，可俘虏）
              </div>
              <div class="old-line">
                胜利奖励：{{ resShort.food }}/{{ resShort.steel }}/{{ resShort.oil }}/{{ resNames.rare }} 各{{ selDetail.res_min }}<!--
                ★ 2026-09-25：管理端「野地获取资源倍率」>1 时标出来，让玩家知道为什么比平时多
                （后端已把倍率乘进 res_min，这里只是加个说明；=1 时不显示，不占版面） -->
                <span v-if="selDetail.res_mult && selDetail.res_mult > 1" class="green">（资源倍率×{{ selDetail.res_mult }}）</span>，{{ resNames.gold }}{{ selDetail.gold }}，
                军功声望+{{ selDetail.prestige }}，必定掉落宝物
              </div>
              <div class="old-line" v-if="selDetail.jewel">采集可获得：{{ selDetail.jewel }}</div>
              <div class="old-line red">活动目标无法占领，战胜只结算奖励(不占附属野地上限)</div>
            </template>
            <!-- 纯海洋: 无野地/守军/出征按钮 -->
            <template v-else-if="selDetail.is_ocean">
              <div class="old-line">【海洋】({{ selCell.x }},{{ selCell.y }})</div>
              <div class="old-line">地形：海洋</div>
            </template>
            <!-- 陆地野地/海底森林/寇城 -->
            <template v-else>
              <div class="old-line">【{{ selDetail.type === 3 ? '寇城' : selDetail.terrain_name }}({{ selDetail.level }}级)】({{ selCell.x }},{{ selCell.y }})</div>
              <div class="old-line">地形：{{ selDetail.terrain_name }}</div>
              <div class="old-line" v-if="selDetail.type === 3">地块：寇城</div>
              <div class="old-line" v-else>野地等级：{{ selDetail.level }}级</div>
              <div class="old-line" v-if="selDetail.type === 3">
                掉落宝物：{{ selDetail.treasure || '普通宝物' }}
              </div>
              <div class="old-line" v-else>归属：{{ selDetail.owner || '中立' }}</div>
              <div class="old-line" v-if="selDetail.gather_res">
                <!-- ★ 2026-09-28 用户反馈：平原/沿海平原(特殊平原,可建航海协会)采集只有粮食、可建城市，不再显示「可能获得宝物」 -->
                <template v-if="selDetail.terrain === 1 || selDetail.terrain === 9">采集可以获得{{ selDetail.gather_res }}，可建立城市。</template>
                <template v-else>采集可以获得{{ selDetail.gather_res }}，可能获得{{ (selDetail.treasures || []).join('、') }}。</template>
              </div>
            </template>
          </template>
          <div class="old-line" v-else>
            {{ selCell.name }}({{ selCell.x }},{{ selCell.y }})
            <!-- ★ 2026-09-29 玩家城市也能收藏：玩家城不开 selDetail，原来的[收藏]被 selDetail 条件挡住了 -->
            <a href="javascript:;" style="margin-left:8px" @click="addStar">{{ isCellStarred ? '已收藏' : '收藏' }}</a>
          </div>
          <div class="old-line" v-if="!selDetail && selCell.owner">城主:{{ selCell.owner }}</div>
          <!-- ★ 玩家城：展示城主的同盟（军团）名 —— 没加入军团显示「无」 -->
          <div class="old-line" v-if="selCell.area_type === 3">
            同盟：
            <b :class="selCell.corps_name ? (selCell.ally ? 'green' : '') : 'gray'">
              {{ selCell.corps_name || '无' }}
            </b>
            <span v-if="selCell.ally" class="green">（你的同盟成员）</span>
          </div>
          <hr/>
          <!-- ★ 按钮文案统一加方括号（用户要求「侦查 掠夺 征服 也加上 []」），
               与已有的 [宣战]/[返回地图]/[查找] 保持同一种「按钮」写法。
               同一行里的 运输/增援/采集 同属动作按钮，一并统一，免得一行里两种写法。 -->
          <!-- ① 本城：不给侦查/掠夺/征服（自己的城市不能打自己），只提示一句 -->
          <div class="old-line" v-if="selCell.area_type === 3 && selCell.mine">
            <a href="javascript:;" @click="go('citystatus')">[城市状态]</a>
          </div>
          <!-- ② 别人的城：侦查/掠夺/征服 常显；掠夺/征服 需宣战生效(status=2)才可点，
               未宣战/待生效时置灰并提示，宣战入口只在没宣战(status=0)时出现。
               ★ 管理端「宣战功能」关掉时（warRequire=false）不需要宣战 → 掠夺/征服直接可点、
                 不再出现 [宣战] 入口，也不再显示「未宣战」状态文案。 -->
          <div class="old-line" v-else-if="selCell.area_type === 3">
            <a href="javascript:;" @click="pickOrder(1)">[侦查]</a>&nbsp;
            <!-- ★ 2026-09-25 用户要求：军团交战期（atWar）无需个人宣战即可掠夺/征服 -->
            <a v-if="warStatus === 2 || !warRequire || atWar" href="javascript:;" @click="pickOrder(2)">[掠夺]</a>
            <a v-else href="javascript:;" class="gray" @click="warBlock('掠夺')">[掠夺]</a>&nbsp;
            <a v-if="warStatus === 2 || !warRequire || atWar" href="javascript:;" @click="pickOrder(3)">[征服]</a>
            <a v-else href="javascript:;" class="gray" @click="warBlock('征服')">[征服]</a>&nbsp;
            <!-- ★ 2026-09-25 军团交战期绿色提示（文案可用后端下发的 corps_war.text） -->
            <div class="old-line green" v-if="corpsWar && corpsWar.active">
              军团交战期：{{ corpsWar.text || ('与【' + (corpsWar.corps_name || '敌方军团') + '】处于交战状态，无需个人宣战即可掠夺/征服') }}
            </div>
            <!-- ★ 运输/增援 只对「同盟(同一军团)成员的城市」显示；宣战中一律不显示
                 （不需要宣战时「交战中」这个概念不成立，所以照常显示） -->
            <template v-if="selCell.ally && (warStatus !== 2 || !warRequire)">
              <a href="javascript:;" @click="pickOrder(5)">[运输]</a>&nbsp;
              <a href="javascript:;" @click="pickOrder(6)">[增援]</a>&nbsp;
            </template>
            <!-- ★ 用户规则「同盟玩家不能宣战」→ 同盟成员不出现 [宣战] 入口，只给提示 -->
            <template v-if="selCell.ally">
              <span class="green">同盟成员之间不能宣战</span>
            </template>
            <a v-else-if="warStatus === 0 && warRequire" href="javascript:;" @click="declareWar">[宣战]</a>
            <!-- 同盟时不再叠「未宣战」这类状态文案，避免读成「不能宣战未宣战」 -->
            <span v-if="warText && !selCell.ally && warRequire" class="orange">{{ warText }}</span>
          </div>
          <!-- ③ 野地/寇城/海洋 -->
          <div class="old-line" v-else-if="selCell.name !== '寇城(废墟)' && !(selDetail && selDetail.is_ocean)">
            <!-- ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服（要先[放弃]）→
                 三个命令灰掉、点了给提示（判据 isOwnWild 取后端下发的 selDetail.mine，
                 与「归属：我」同源，不靠前端猜）；[采集] 不受影响。 -->
            <a href="javascript:;" :class="{ gray: isOwnWild }" @click="pickOrder(1)">[侦查]</a><span class="home-gap"></span>
            <a href="javascript:;" :class="{ gray: isOwnWild }" @click="pickOrder(2)">[掠夺]</a><span class="home-gap"></span>
            <a href="javascript:;" :class="{ gray: isOwnWild }" @click="pickOrder(3)">[征服]</a><span class="home-gap"></span>
            <a v-if="selCell.occupied && isOwnWild" href="javascript:;" @click="pickOrder(4)">[采集]</a>
            <span v-else-if="!selCell.occupied && (!selDetail || !selDetail.act_type)" class="gray">(占领该野地后可采集)</span>
          </div>
          <a href="javascript:;" @click="go('map')">[返回地图]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="go('map')">[前往地图]</a></div>
        </div>
      </template>

      <!-- ============ 出征确认(orderpre) ============ -->
      <!-- ★ 用户反馈「出征页面看着好乱」→ 按 ①兵力 ②军官 ③集结令 ④随军资源 ⑤宿营 ⑥计算 分区，
           每区一个小标题，输入项改成两列网格，整体高度比原来短很多。 -->
      <template v-else-if="cur === 'orderpre'">
        <div class="panel" v-if="selCell">
          <div class="panel-title">出征确认 · {{ orderNames[orderType] }}</div>
          <!-- ★ 用户反馈「切换城市后感觉出征页还是切换前那个城」→
               出征页原来只写「目标」，看不出这支部队是从哪座城出发的。
               把**出发城市**显式写在最上面，玩家一眼就能确认用的是哪座城的兵/军官/资源。 -->
          <div class="old-line">
            出发城市：<b>{{ city.name }}</b>
            <span v-if="city.x || city.y">({{ city.x }},{{ city.y }})</span>
            <span class="gray">（部队 / 军官 / 随军资源都从这座城市出发）</span>
          </div>
          <div class="old-line">
            目标：<b>{{ selCell.name }}</b><span v-if="selCell.level">({{ selCell.level }}级)</span>
            ({{ selCell.x }},{{ selCell.y }})
          </div>
          <hr/>

          <!-- ★ 2026-09-28 出征页顺序重排：①指挥军官 → ②出征集结令 → ③选择兵力
               （用户要求军官放第一个 → 集结令 → 兵种；军团/属性用「军/学/后」简写、不展示忠诚）
               预设编队下拉：选中后回填 军官/集结令/兵力（兵力夹到可出征上限，军官不在当前城则回填空） -->
          <!-- 预设编队 -->
          <div class="of-sec">预设编队</div>
          <div class="old-line">
            <select v-model="presetSel" @change="applyPreset" style="width:200px">
              <option :value="0">不使用预设</option>
              <option v-for="p in presets" :key="'ps' + p.id" :value="p.id">
                {{ p.name }}({{ fmtN(p.troop_total) }}兵<template v-if="p.officer">·{{ p.officer }}</template>)
              </option>
            </select>
            <span class="gray" v-if="!presets.length">(暂无预设，可到司令部「预设编队」添加)</span>
            <a href="javascript:;" @click="go('hq'); $nextTick(() => selectHqTab(4))">[管理预设]</a>
          </div>
          <!-- ① 军官 -->
          <div class="of-sec">① 指挥军官</div>
          <div class="old-line">
            <select v-model="orderOfficer" @change="doCalc">
              <option value="0">未指定</option>
              <option v-for="o in onDutyOfficers" :key="'od' + o.id" :value="o.name">
                {{ o.name }}({{ o.level }}级) 军{{ o.military_total || o.military }}
                <span class="green" v-if="o.equip_military">(装+{{ o.equip_military }})</span>
                学{{ o.learning_total || o.learning }} 后{{ o.logistics_total || o.logistics }}
              </option>
            </select>
            <span v-if="orderType === 7" class="red">(派遣必须选择)</span>
            <span v-else-if="orderType === 6" class="gray">(增援后军官调任目标城市)</span>
            <span v-else-if="orderType === 8" class="gray">(派遣后军官随军调往目标城市)</span>
            <span v-if="curOfficerBonus" class="green"> 军官战斗加成: 攻击+{{ curOfficerBonus }}%</span>
            <span v-if="!onDutyOfficers.length" class="gray">
              (「{{ city.name }}」暂无可带队军官<template v-if="cityOfficers.length">：本城 {{ cityOfficers.length }} 名军官都在出征中或为俘虏</template>；
              军官跟着城市走，别的城的军官不能在这里出征，可前往军校招募)
            </span>
          </div>

          <!-- ② 集结令 -->
          <div class="of-sec">② 出征集结令</div>
          <div class="old-line">
            使用
            <input type="number" min="0" :max="gatherMax" v-model.number="orderGather"
                   :disabled="gatherCount <= 0" @change="onGatherChange" style="width:80px"/>
            个
            <span class="gray">（背包里有 {{ gatherCount }} 个）</span>
          </div>
          <div class="old-line gray">
            每个集结令 +{{ fmtN(orderCapPer) }} 出征上限，单次最多 {{ orderCapMax }} 个。
            司令部上限（含指挥艺术科技）+ 集结令 + 出征军官军事属性<b>叠加</b>。
          </div>
          <div class="old-line" v-if="attackTroops.length">
            <span :class="orderOverCap ? 'red' : 'green'">
              本次出兵 <b>{{ fmtN(orderTroopTotal) }}</b> / 上限 <b>{{ orderCapText }}</b>
              <template v-if="orderOverCap">—— 超出上限，请减少兵力或加用集结令</template>
            </span>
          </div>

          <!-- ③ 兵力 -->
          <div class="of-sec">③ 选择兵力
            <span class="of-hint">（拖滑块或直接填数字；滑块与 [最大] 都按「城内现有」和「出征上限剩余」取小）</span>
          </div>
          <div class="of-rows">
            <div class="of-row" v-for="t in trainCfgs" :key="'at' + t.id"
                 :class="{ 'of-off': troopCount(t.id) <= 0 }"
                 :title="t.name + '（现有 ' + fmtN(troopCount(t.id)) + '，本次最多可派 ' + fmtN(orderQtyMax(t.id)) + '）'">
              <span class="of-name">{{ t.name }}</span>
              <span class="of-avail">现有 {{ fmtN(troopCount(t.id)) }}</span>
              <span class="of-ctl">
                <input type="range" class="of-range" min="0" step="1"
                       :max="troopCount(t.id)" :value="orderQty(t.id)"
                       :disabled="orderQtyMax(t.id) <= 0"
                       @input="onOrderQtyInput(t.id, $event)"/>
                <input type="number" class="of-num" min="0" placeholder="0"
                       :max="orderQtyMax(t.id)" :value="orderQty(t.id)"
                       :disabled="orderQtyMax(t.id) <= 0"
                       @input="onOrderQtyInput(t.id, $event)"/>
                <a href="javascript:;" class="of-max"
                   :class="{ 'of-max-off': orderQtyMax(t.id) <= 0 }"
                   @click="setOrderQtyMax(t.id)">[最大]</a>
              </span>
            </div>
          </div>
          <div class="old-line red" v-if="!attackTroops.length">城内无可出征部队</div>

          <!-- ④ 随军资源 -->
           <!-- （右侧灰字是城内现有；上限 = 所带兵种负重之和 × 装载技术加成，没带部队时不能填） -->
          <div class="of-sec">④ 随军资源
            <!-- <span class="of-hint">（每行拖滑块或直接填数字；上限 = 所带兵种负重之和 × 装载技术加成；没选部队时禁用）</span> -->
          </div>
          <div class="of-rows">
            <div class="of-row" v-for="res in resFields" :key="res.key"
                 :class="{ 'of-off': orderResDisabled }"
                 :title="res.name + '（城内现有 ' + fmtN(resAvail(res.key)) + '，本次最多 ' + fmtN(resQtyMax(res.key)) + '）'">
              <span class="of-name">{{ res.name }}</span>
              <span class="of-avail">现有 {{ fmtN(resAvail(res.key)) }}</span>
              <span class="of-ctl">
                <input type="range" class="of-range" min="0" step="1"
                       :max="resQtyMax(res.key)" :value="resQty(res.key)"
                       :disabled="orderResDisabled"
                       @input="onResInput(res.key, $event)"/>
                <input type="number" class="of-num" min="0" placeholder="0"
                       :max="resQtyMax(res.key)" :value="resQty(res.key)"
                       :disabled="orderResDisabled"
                       @input="onResInput(res.key, $event)"/>
                <a href="javascript:;" class="of-max"
                   :class="{ 'of-max-off': orderResDisabled || resQtyMax(res.key) <= 0 }"
                   @click="setResMax(res.key)">[最大]</a>
              </span>
            </div>
          </div>
          <div class="old-line" v-if="!orderResDisabled">
            随军总量：<b :class="orderResOver ? 'red' : 'green'">{{ fmtN(orderResTotal) }}</b>
            / 可用负重 <b>{{ fmtN(orderResUsableCap) }}</b>
            <span v-if="orderResOver" class="red">
              —— 超出（<template v-if="orderType !== 5">负重{{ fmtN(orderResCap) }} − 油耗{{ fmtN(oilUsed) }}</template><template v-else>负重{{ fmtN(orderResCap) }}</template>），请减少资源或多带部队
            </span>
            <span v-else class="gray">
              <template v-if="orderType !== 5">（负重{{ fmtN(orderResCap) }} − 油耗{{ fmtN(oilUsed) }}，含装载技术加成）</template>
              <template v-else>（负重{{ fmtN(orderResCap) }}，运输油耗另扣不占负重；含装载技术加成）</template>
            </span>
          </div>
          <div class="old-line gray" v-else>（未选择部队，随军资源不可填写）</div>
          <div class="old-line gray" v-if="orderType === 5">
            运输：自己城市之间 / 同盟成员之间都能运；必须带部队来装货，能运多少看<b>负重</b>，一般用卡车；
            可以不带队军官；送完部队会返回出发城市。
          </div>
          <div class="old-line gray" v-else-if="orderType === 7">派遣必须选择带队军官。</div>
          <div class="old-line gray" v-else-if="orderType === 8">
            派遣：把自己的部队 / 军官 / 随军资源送到<b>自己的另一座城市</b>；必须带部队，
            能带多少资源看<b>负重</b>，军官会随军调往目标城市。
          </div>

          <!-- ⑤ 宿营 -->
           <!-- （抵达后停留，可选） -->
          <div class="of-sec">⑤ 宿营</div>
          <div class="old-line">
            <input v-model="waitH" type="number" min="0" max="24" style="width:50px"/> 时
            <input v-model="waitM" type="number" min="0" max="60" style="width:50px"/> 分
            <span class="gray">(最多宿营 24 小时)</span>
          </div>

          <!-- ⑥ 计算 / 出征 -->
          <div class="of-sec">⑥ 消耗预览</div>
          <div class="old-line">
            <button @click="doCalc">[计算]</button>
            油耗：<span class="orange">{{ orderCalc ? orderCalc.oil_used : '—' }}</span>
            &nbsp;/&nbsp;负重：<span class="orange">{{ orderCalc ? orderCalc.carry : '—' }}</span>
            &nbsp;/&nbsp;耗时：<span class="orange">{{ orderCalc ? orderCalc.need_time : '—' }}</span>
            <span v-if="orderCalc" class="gray">
              (单程{{ orderCalc.travel_time }}<template v-if="orderCalc.wait_min">, 宿营{{ orderCalc.wait_min }}分</template>)
            </span>
          </div>
          <div class="old-line red" v-if="orderCalc && !orderCalc.oil_enough">
            {{ resNames.oil }}不足：需要{{ orderCalc.oil_used }}，当前只有{{ orderCalc.oil_have }}
          </div>
          <div class="old-line gray">出征前请先点 [计算] 确认油耗与负重，否则可能无法出征成功。</div>
          <hr/>
          <div class="old-line">
            <button @click="doOrder()">[出征]</button>
            <a href="javascript:;" @click="go('map')">[返回地图]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">请先在地图上选择目标 <a href="javascript:;" @click="go('map')">[前往地图]</a></div>
        </div>
      </template>

      <!-- ============ 战场指挥室(battle) 复刻《战斗机制》§1 ============
           入口：军情 → 军队动态 → [指挥]
           每回合 30 秒：前 25 秒可下达前进/暂停/后退，后 5 秒锁定并由服务器结算，最多 40 回合 -->
      <template v-else-if="cur === 'battle'">
        <div class="panel-title">
          【战场指挥】{{ battleData.target_name }}({{ battleData.target_x }},{{ battleData.target_y }})
        </div>
        <div class="panel">
          <div class="old-line">
            第 {{ battleData.round }}/{{ battleData.max_round }} 回合
            <span v-if="battleData.done" class="red">（战斗已结束<span v-if="battleData.draw"> · 40回合平局</span>）</span>
            <span v-else-if="battleData.phase === 'cmd'" class="green">（指令期，可下达命令）</span>
            <span v-else class="red">（已锁定，等待结算）</span>
          </div>
          <!-- 本回合倒计时（最后 5 秒锁定：条变红 = 已锁定）
               ★ 规则说明一律不写进界面（用户要求），记在这里：
                 · 每回合 30 秒，前 25 秒（cmd_window_ms）可下达指令，后 5 秒锁定由服务器结算；
                 · 指令是**逐兵种**的（用户要求「自己带的兵种都能指挥，就是单独指挥」）；
                 · 没下指令的兵种按司令部「兵种战斗配置」行动；
                 · 兵种目标同样是**逐兵种**的：默认取司令部配置，指挥时可改（0 = 最近目标），
                   守方没有该兵种时服务器自动回落打最近的（2026-09-23 用户要求）；
                 · [自动战斗] = 自己全部军队前进，一口气打完。 -->
          <div class="old-line" v-if="!battleData.done">
            {{ battleData.time_label || '本回合剩余' }}：<b>{{ battleLeftText }}</b>
            <div class="ezfy-battle-bar">
              <i :class="battleData.phase === 'cmd' ? 'on' : 'lock'" :style="{ width: battleBarPct + '%' }"></i>
            </div>
          </div>
          <div class="old-line ezfy-battle-cmds" v-if="!battleData.done">
            <a href="javascript:;" @click="sendBattleCmd('advance')">[全军前进]</a>
            <a href="javascript:;" @click="sendBattleCmd('hold')">[全军停止]</a>
            <a href="javascript:;" @click="sendBattleCmd('retreat')">[全军后退]</a>
            <a v-if="battleData.can_auto" href="javascript:;" @click="doBattleAuto">[自动战斗]</a>
          </div>
          <!-- 双方兵力 + 逐兵种指挥（指令 + 优先攻击目标） -->
          <table class="ezfy-plain-table ezfy-battle-tbl">
            <tr>
              <th>方</th><th class="nm">兵种</th><th>剩余</th><th>初始</th><th>位置</th>
              <th v-if="!battleData.done">目标</th>
              <th v-if="!battleData.done">指挥</th>
            </tr>
            <tr v-for="u in battleData.attackers" :key="'ba' + u.troop_id"
                :class="battleData.is_atk ? 'ezfy-row-self' : 'ezfy-row-enemy'">
              <td class="ezfy-side-lbl"><span :class="battleData.is_atk ? 'green' : 'red'">攻</span></td>
              <td class="nm">{{ u.name }}</td>
              <td>{{ fmtN(u.count) }}</td><td>{{ fmtN(u.initial) }}</td><td>{{ u.pos }}</td>
              <!-- ★ 兵种目标（2026-09-23 用户要求）：默认 = 司令部「兵种战斗配置」，
                   指挥时玩家可逐兵种改；0 = 最近目标（守方没有该兵种时服务器自动打最近的）。
                   ★ 守方视角(is_atk=false)时这里显示 AI，指挥控件渲染到守方行上。 -->
              <td v-if="!battleData.done">
                <template v-if="battleData.is_atk">
                  <select :value="u.target_troop"
                          @change="sendBattleTarget(u.troop_id, $event)"
                          style="width:96px">
                    <option v-for="op in (battleData.target_options || [])"
                            :key="'to' + u.troop_id + '_' + op.id" :value="op.id">{{ op.name }}</option>
                  </select>
                </template>
                <span v-else class="gray">-</span>
              </td>
              <td v-if="!battleData.done" class="ezfy-cmd">
                <template v-if="battleData.is_atk">
                  <a href="javascript:;" :class="{ on: u.cmd === 'advance' }" @click="sendBattleCmd('advance', u.troop_id)">[前进]</a>
                  <a href="javascript:;" :class="{ on: u.cmd === 'hold' }" @click="sendBattleCmd('hold', u.troop_id)">[停止]</a>
                  <a href="javascript:;" :class="{ on: u.cmd === 'retreat' }" @click="sendBattleCmd('retreat', u.troop_id)">[后退]</a>
                  <span class="gray">{{ u.cmd_name }}</span>
                </template>
                <span v-else class="gray">AI</span>
              </td>
            </tr>
            <tr v-for="u in battleData.defenders" :key="'bd' + u.troop_id"
                :class="!battleData.is_atk ? 'ezfy-row-self' : 'ezfy-row-enemy'">
              <td class="ezfy-side-lbl"><span :class="!battleData.is_atk ? 'green' : 'red'">守</span></td>
              <td class="nm">{{ u.name }}</td>
              <td>{{ fmtN(u.count) }}</td><td>{{ fmtN(u.initial) }}</td><td>{{ u.pos }}</td>
              <td v-if="!battleData.done">
                <template v-if="!battleData.is_atk">
                  <select :value="u.target_troop"
                          @change="sendBattleTarget(u.troop_id, $event)"
                          style="width:96px">
                    <option v-for="op in (battleData.target_options || [])"
                            :key="'to' + u.troop_id + '_' + op.id" :value="op.id">{{ op.name }}</option>
                  </select>
                </template>
                <span v-else class="gray">-</span>
              </td>
              <td v-if="!battleData.done" class="ezfy-cmd">
                <template v-if="!battleData.is_atk">
                  <a href="javascript:;" :class="{ on: u.cmd === 'advance' }" @click="sendBattleCmd('advance', u.troop_id)">[前进]</a>
                  <a href="javascript:;" :class="{ on: u.cmd === 'hold' }" @click="sendBattleCmd('hold', u.troop_id)">[停止]</a>
                  <a href="javascript:;" :class="{ on: u.cmd === 'retreat' }" @click="sendBattleCmd('retreat', u.troop_id)">[后退]</a>
                  <span class="gray">{{ u.cmd_name }}</span>
                </template>
                <span v-else class="gray">AI</span>
              </td>
            </tr>
          </table>
          <div class="old-line">
            战场态势：攻方 {{ fmtN(battleData.atk_total) }} · 守方 {{ fmtN(battleData.def_total) }}
          </div>
          <!-- 行动日志：最新回合在最上、已过回合在下；我方绿色、敌军红色 -->
          <div class="old-line ezfy-battle-legend">
            <span class="green">■ 我方</span>&nbsp;<span class="red">■ 敌军</span>
          </div>
          <template v-if="battleRounds.length">
            <div class="old-line ezfy-round-block" v-for="(g, gi) in battleRounds" :key="'bg' + gi">
              <div v-if="g.line" class="ezfy-round-title">{{ g.line }}</div>
              <div v-for="(li, idx) in g.items" :key="'bl' + gi + '_' + idx"
                   class="ezfy-round-line" :class="battleLineClass(li)">{{ li }}</div>
            </div>
          </template>
          <div class="old-line gray" v-else>(暂无行动)</div>
          <div class="old-line">
            <a href="javascript:;" @click="leaveBattle">[返回军队动态]</a>
          </div>
        </div>
      </template>

      <!-- ============ 命令详情 / 军队动态详情(orderview) ============ -->
      <template v-else-if="cur === 'orderview'">
        <div class="panel" v-if="curOrder">
          <div class="panel-title">军队动态详情</div>
          出发地:{{ curOrder.from_name }}<br/>
          目的地:{{ curOrder.target_name }}({{ curOrder.target_x }},{{ curOrder.target_y }})<br/>
          命令:{{ curOrder.type_name }}<br/>
          军官:{{ curOrder.officer || '无(未带军官)' }}<br/>
          统帅:{{ nick }}<br/>
          状态:{{ (curOrder.order_type === 7 && curOrder.status === 1) ? orderStatusText(curOrder) : (curOrder.status_name || orderStatusText(curOrder)) }}<br/>
          出发时间:{{ curOrder.start_text }}<br/>
          到达时间:{{ curOrder.arrive_text }}<br/>
          <template v-if="curOrder.return_text">返航时间:{{ curOrder.return_text }}<br/></template>
          耗油:{{ curOrder.oil_used }}<br/>
          <hr/>
          【进攻方军队】<br/>
          <span v-for="(t, i) in curOrder.troops" :key="'ot' + i">{{ t.name }}:{{ t.count }}<br/></span>
          <span v-if="!curOrder.troops.length" class="gray">(未携带部队)</span>
          <template v-if="curOrder.resources && curOrder.resources.length">
            <br/>军队携带资源:<br/>
            <span v-for="(r, i) in curOrder.resources" :key="'or' + i">{{ r.name }}:{{ r.count }}<br/></span>
          </template>
          <hr/>
          <div class="old-line">
            <a href="javascript:;" @click="go('hq')">[指挥(司令部)]</a>
            <a v-if="curOrder.status === 0 || curOrder.status === 1"
               class="red" href="javascript:;" @click="doRecall(curOrder)">[取消出征]</a>
            <a v-if="curOrder.order_type === 7 && curOrder.status === 1 && !curOrder.arrive_time"
               class="red" href="javascript:;" @click="startCollect(curOrder)">[采集]</a>
            <a v-if="curOrder.order_type === 7 && curOrder.status === 1 && curOrder.arrive_time"
               href="javascript:;" @click="go('wilds')">[查看野地]</a>
            <a v-if="curOrder.report_id" href="javascript:;" @click="jumpReport(curOrder.report_id)">[查看战报]</a>
          </div>
          <a href="javascript:;" @click="go('orders')">[返回出征队列]</a>
        </div>
        <!-- ★ 2026-09-25：原来这里没有 v-else —— 刷新后 curOrder 丢了就整页空白（用户报「刷新页面消失」）。
             现在给兜底提示 + 回队列的入口，任何情况下都不会白屏。 -->
        <div class="panel" v-else>
          <div class="old-line">命令不存在或已结束</div>
          <div class="old-line gray">可能该命令已经完成/被取消，战报里仍可查到。</div>
          <a href="javascript:;" @click="go('orders')">[返回出征队列]</a>
        </div>
      </template>

      <!-- ============ 出征队列(orders) ============ -->
      <!-- ★ 2026-09-25 用户要求「出征队列按照军队动态那种展示」：
           数据源改成 /reports/dynamics（与「军情 → 军队动态」同一个接口、同一套字段），
           渲染样式也照抄军队动态的竖排块（命令/目标/状态/军官/时间/待带回/操作），
           比原来的 6 列表格信息全得多（原来没有待带回、没有指挥室入口）。
           ★ 同时修「刷新后消失」：本页数据在 go('orders') 里重新拉，
             不再依赖内存里的旧数组；命令详情页也带了 oid 到 URL（见 syncUrl/restoreFromUrl）。 -->

      <!-- ============ 计谋(scheme)：行军计谋专用页 ============ -->
      <!-- ★ 2026-09-30 用户要求：军队动态/出征队列的 [计谋] 跳到这里，
           展示信号弹持有量与神兵天降/战略转移说明，对当前部队去程/回程使用；
           使用完返回「军情 → 军队动态」。 -->
      <template v-else-if="cur === 'scheme'">
        <div class="panel" v-if="schemeOrder">
          <div class="panel-title">计谋</div>
          <div class="old-line">
            【信号弹】持有：
            <b :class="schemeData.bullet_have > 0 ? 'green' : 'red'">{{ schemeData.bullet_have }}</b>
            <a href="javascript:;" @click="go('mall')">[去商城购买]</a>
          </div>
          <div class="old-line gray">
            神兵天降：去程剩余时间减少80%。战略转移：回程减少360分钟。每次消耗7枚信号弹，每支部队两种计谋各限一次。
          </div>
          <hr/>
          部队：{{ schemeOrder.type_name }}（{{ schemeOrder.target_name }}({{ schemeOrder.target_x }},{{ schemeOrder.target_y }})）<br/>
          状态：{{ schemeOrder.status_name }}<br/>
          <!-- 出征中 → 去程用神兵天降；返回中 → 回程用战略转移 -->
          <template v-if="schemeOrder.status === 0">
            <div class="old-line">
              ({{ schemeOrder.target_x }},{{ schemeOrder.target_y }}) 去程
              <a v-if="schemeOrder.scheme_fast === 0" class="green"
                 href="javascript:;" @click="doMarchScheme(schemeOrder, 13)">[神兵天降]</a>
              <span v-else class="gray">[神兵天降·已用]</span>
            </div>
          </template>
          <template v-else-if="schemeOrder.status === 2">
            <div class="old-line">
              ({{ schemeOrder.target_x }},{{ schemeOrder.target_y }}) 回程
              <a v-if="schemeOrder.scheme_back === 0" class="green"
                 href="javascript:;" @click="doMarchScheme(schemeOrder, 14)">[战略转移]</a>
              <span v-else class="gray">[战略转移·已用]</span>
            </div>
          </template>
          <div class="old-line">
            <a href="javascript:;" @click="go('reports')">[返回军情-军队动态]</a>
            <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
        </div>
        <div class="panel" v-else>
          <div class="old-line">没有可使用的计谋部队</div>
          <a href="javascript:;" @click="go('reports')">[返回军情-军队动态]</a>
        </div>
      </template>

      <template v-else-if="cur === 'orders'">
        <div class="panel">
          <div class="panel-title">出征队列({{ queueItems.length }})</div>
          <div class="old-line gray">
            包含行军中 / 战斗中 / 返航中 / 驻守采集的全部部队；驻守空闲的部队需点 [采集] 才开始采集。
          </div>
          <div class="old-line" v-for="o in queueItems" :key="'oq' + o.id">
            命令：{{ o.type_name }} <a v-if="!o.is_defend" href="javascript:;" @click="openOrder(o)">查看</a><br/>
            目标：<span v-if="o.act_type" class="red">[{{ actTag(o.act_type) }}]</span>{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})
            <span v-if="o.is_defend" class="red">(敌军来袭)</span><br/>
            状态：{{ o.status_name }}
            <template v-if="o.can_command">
              <a href="javascript:;" class="red" @click="openBattle(o.id)">[指挥]</a>
              <span class="gray">第{{ o.battle_round || 1 }}/{{ o.battle_max }}回合</span>
            </template>
            <template v-else-if="o.status === 1 && !o.arrive_time">
              <a href="javascript:;" class="red" @click="startCollect(o)">[采集]</a>
            </template>
            <template v-else-if="o.status === 1 && o.arrive_time">
              <a href="javascript:;" class="red" @click="stopCollect(o)">[停止]</a>
            </template>
            <!-- ★ 2026-09-30 行军计谋：出征中(0)/返回中(2)显示 [计谋]，进入计谋页选择神兵天降/战略转移 -->
            <template v-if="o.status === 0 || o.status === 2">
              <a class="green" href="javascript:;" @click="openScheme(o)">[计谋]</a>
            </template>
            <br/>
            军官：{{ o.officer || '无' }}<br/>
            {{ o.time_label }}：{{ o._lt || (o._lg ? o._lg.timeText : o.time_text) }}<br/>
            <!-- ★ 2026-09-28 采集中部队: 实时累加显示本期已采资源(每秒由 liveGather 重算)。
                 规则已改为「收获即入起点城市」，故不再显示「需召回返航后入库」。 -->
            <template v-if="o.status === 1 && o.arrive_time">
              <span class="green">本期已采：{{ fmtN(o._lg.food) }}粮/{{ fmtN(o._lg.steel) }}钢/{{ fmtN(o._lg.oil) }}油/{{ fmtN(o._lg.rare) }}稀/{{ fmtN(o._lg.gold) }}金</span><br/>
              <span class="gray">总 {{ fmtN(o._lg.total) }}（负重 {{ fmtN(o._lg.total) }}/{{ fmtN(o.carry_cap) }}）</span>
              <span v-if="o._lg.full" class="red">负重已满, 超出部分会直接入库(可停止或收获)。</span><br/>
            </template>
            <br/>
            <span v-if="o.status === 0 || o.status === 1">
              <a href="javascript:;" class="red" @click="doRecall(o)">[取消]</a><br/>
            </span>
            --------------------
          </div>
          <div class="old-line" v-if="!queueItems.length">(暂无出征部队)</div>
          <div class="old-line">
            <a href="javascript:;" @click="loadDynamics">[刷新]</a>
          </div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 城市状态(citystatus) ============ -->
      <template v-else-if="cur === 'citystatus'">
        <div class="panel">
          <div class="panel-title">城市状态</div>
          城市: {{ city.name }}({{ city.x }},{{ city.y }}) {{ continent }}
          <!-- ★ 类型文案取 /view 顶层下发的 city_kind（原来读 city.city_kind，而嵌套的 city 对象里没有这个字段，
               于是这里永远显示「陆地城市」，和城市列表的「海城」对不上 —— 用户反馈的 bug） -->
          <span :class="cityIsSea ? 'green' : 'gray'">[{{ cityKindLabel }}]</span><br/>
          市政厅: {{ city.city_level }}级<br/>
          人口: {{ city.pop }}/{{ housePopLimitOn ? city.pop_max : '不限' }} (空闲{{ freePop }})<br/>
          民心/民怨: {{ city.feelings }}/{{ city.grievance }} 税率: {{ city.tax_rate }}%<br/>
          建筑: {{ buildings.length }}座 (军事+资源区 {{ areaCount }}/{{ areaCap }})<br/>
          军队: {{ totalTroops }} (城外{{ marching }}支队伍行进, {{ occupying }}支驻守)<br/>
          附属野地: {{ wildlands.length }}/{{ city.city_level }}<br/>
          训练队列: {{ queues.length }} | 未读军情: {{ unreadReports }}<br/>
          <span v-if="protectedUntil" class="green">[免战保护中]</span>
          <span v-if="boostUntil" class="orange">[增产中]</span>
          <br/>
          <a href="javascript:;" @click="go('cityhall')">[市政厅]</a>
          <a href="javascript:;" @click="go('wareset')">[仓库调配]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 附属野地(wilds) ============ -->
      <template v-else-if="cur === 'wilds'">
        <div class="panel">
          <div class="panel-title">占领野地({{ wildlands.length }}/{{ city.city_level }})</div>
          <table>
            <tr><th>坐标</th><th>地形</th><th>所属洲</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <tr v-for="w in wildlands" :key="'wd' + w.id">
              <td>({{ w.x }},{{ w.y }})</td>
              <td>{{ w.terrain === 8 ? '海底森林' : w.terrain_name }}</td>
              <td>{{ w.continent || '—' }}</td>
              <td>{{ w.level }}</td>
              <td>
                <template v-if="w.status === 1">采集中</template>
                <template v-else-if="w.idle_order_id">驻守(空闲)</template>
                <template v-else>空闲</template>
              </td>
              <td>
                <!-- ★ 2026-09-28 平原/沿海平原可建城、采集无宝物, 附属野地列表不再提供采集, 仅保留[放弃] -->
                <a v-if="w.status === 0 && !w.idle_order_id && w.terrain !== 1 && w.terrain !== 9" href="javascript:;" @click="openWildGather(w)">[采集]</a>
                <a v-else-if="w.idle_order_id && w.terrain !== 1 && w.terrain !== 9" class="red" href="javascript:;" @click="startCollect(w.idle_order_id)">[开始采集]</a>
                <!-- ★ 2026-09-28 用户反馈「采集中只能[放弃]，没法[停止]」：
                     后端现在会下发 gather_order_id（见 ezfy.go 的 wildViews），
                     这里据此构造一个最小的订单对象喂给 stopCollect（它只用到 id/target_* 拼提示文案）。 -->
                <a v-if="w.status === 1 && w.gather_order_id" class="red" href="javascript:;"
                   @click="stopCollect(wildOrderArg(w))">[停止]</a>
                <span v-if="w.status === 1" class="gray">采集中</span>
                <!-- ★ 2026-09-28 用户反馈「采集中 别展示 放弃按钮」：
                     采集中(status=1)时操作列只留 [停止]；[放弃] 会让整块野地连同采集部队一起处理掉，
                     应当先停止采集再放弃，故采集中隐藏。 -->
                <a v-if="w.status !== 1" class="red" href="javascript:;" @click="doAbandon(w)">[放弃]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!wildlands.length">(尚未占领任何野地)</div>
          <br/>
          <template v-if="occupies.length">
            <div class="panel-title">被占领城市({{ occupies.length }})</div>
            <table>
              <tr><th>坐标</th><th>城市</th><th>原属</th><th>操作</th></tr>
              <tr v-for="o in occupies" :key="'oc' + o.id">
                <td>({{ o.x }},{{ o.y }})</td>
                <td>{{ o.city_name }}</td>
                <td>{{ o.def_user }}</td>
                <td>
                  <a href="javascript:;" @click="doOccupy('build', o)">[建立城市]</a>
                  <a class="red" href="javascript:;" @click="doOccupy('destroy', o)">[摧毁]</a>
                  <a href="javascript:;" @click="doOccupy('return', o)">[放弃归还]</a>
                </td>
              </tr>
            </table>
            <br/>
          </template>
          <a href="javascript:;" @click="go('map')">[前往地图占领]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 召集(convene) ============ -->
      <template v-else-if="cur === 'convene'">
        <div class="panel">
          <div class="panel-title">召集人口</div>
          当前人口: {{ city.pop }} / 民居容纳: {{ housePopLimitOn ? city.pop_max : '不限' }}<br/>
          <!-- ★ 2026-09-26：全局硬性人口上限（管理端配置，0 表示不限），超过则禁止召集 -->
          <template v-if="convenePopMax > 0">
            召集人口上限: {{ fmtBig(convenePopMax) }}<br/>
          </template>
          <!-- ★ 2026-09-26：提示文案随「民居容量限制 / 召集人口灵活配置」两个开关变化，
               花费粮食/获得人口都读管理端配置（默认各 10 万），勿再写死 -->
          <template v-if="!housePopLimitOn">
            花费 {{ fmtBig(conveneFoodCost) }}{{ resNames.food }} 召集 {{ fmtBig(convenePopGain) }}人口(民居容量限制已关闭, 人口无上限)<br/>
          </template>
          <template v-else-if="conveneFlexibleOn">
            花费 {{ fmtBig(conveneFoodCost) }}{{ resNames.food }} 召集 {{ fmtBig(convenePopGain) }}人口(不受民居容纳上限限制, 可突破上限)<br/>
          </template>
          <template v-else>
            花费 {{ fmtBig(conveneFoodCost) }}{{ resNames.food }} 召集 {{ fmtBig(convenePopGain) }}人口(受民居容纳上限限制, 满员后无法召集)<br/>
          </template>
          <div class="old-line">{{ resNames.food }}: {{ city.food }}</div>
          <button @click="doConvene" :disabled="conveneBlocked">[召集]</button>
          <span v-if="conveneBlocked" class="gray">已达人口上限, 无法召集</span>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 仓库调配(wareset) ============ -->
      <template v-else-if="cur === 'wareset'">
        <div class="panel">
          <div class="panel-title">仓库({{ ware.level }}级)</div>
          <div class="old-line">
            保护总量:{{ ware.total }}
            <span class="gray">(保护额度内的资源不会被敌人抢夺走)</span>
          </div>
          <div class="old-line" v-if="ware.level < 1">
            <span class="red">尚未建造仓库, 无法保护资源</span>
            <a href="javascript:;" @click="go('builds')">[前往资源区建造]</a>
          </div>
          <div class="old-line" v-else-if="ware.level < 10">
            升级仓库可提升保护量, 下一级:{{ ware.next_total }}
            <a href="javascript:;" @click="go('builds')">[前往资源区]</a>
          </div>
          <div class="old-line" v-else>仓库已满级(保护量{{ ware.total }})</div>
          <div class="panel-title">调配保护比例(四项合计不超过100%)</div>
          <div class="old-line" v-for="r in ware.res" :key="'wr' + r.key">
            {{ r.name }}: 保护{{ r.protect }} / 现有{{ r.have }}
            <input v-model="wareRatio[r.key]" type="number" min="0" max="100" style="width:60px"/>%
          </div>
          <div class="old-line">
            当前合计:{{ wareSum }}%
            <span v-if="wareSum > 100" class="red">(超出100%, 无法保存)</span>
            <span v-else class="gray">(未分配部分不产生保护)</span>
          </div>
          <div class="old-line">
            <button @click="doWareSet" :disabled="wareSum > 100">[保存比例]</button>
          </div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
        </div>
      </template>

      <!-- ============ 安抚(placate) ============ -->
      <template v-else-if="cur === 'placate'">
        <div class="panel">
          <div class="panel-title">安抚民心</div>
          当前民心: {{ city.feelings }} / 民怨: {{ city.grievance }}<br/>
          <div class="old-line">{{ resNames.gold }}: {{ city.gold }} | 安抚花费: {{ placate.gold }}</div>
          <!-- ★ 冷却用 placateNow 每秒本地重算，不靠重新拉接口（沿用全站倒计时同一套做法） -->
          <template v-if="placateCdLeft > 0">
            <span class="gray">冷却中，还需 {{ durText(placateCdLeft / 1000) }}</span><br/>
            <button class="gray" disabled>[安抚]</button>
          </template>
          <button v-else @click="doPlacate">[安抚]</button>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 税率(taxset) ============ -->
      <template v-else-if="cur === 'taxset'">
        <div class="panel">
          <div class="panel-title">税率设置</div>
          当前税率: {{ city.tax_rate }}% / 民心: {{ city.feelings }}<br/>
          <div class="old-line">
            新税率: <input v-model="taxInput" type="number" min="0" max="100" style="width:70px"/>%
            <button @click="doTax">[设置]</button>
          </div>
          <div class="old-line gray">设置后民心将联动为 {{ 100 - (parseInt(taxInput) || 0) }}</div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 改名(rename) ============ -->
      <template v-else-if="cur === 'rename'">
        <div class="panel">
          <div class="panel-title">城市改名</div>
          <div class="old-line">
            新城市名: <input v-model="renameInput" style="width:60%"/>
            <button @click="doRename">[确定]</button>
          </div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 市政厅(cityhall) ============ -->
      <template v-else-if="cur === 'cityhall'">
        <div class="panel">
          <div class="panel-title">市政厅({{ city.city_level }}级)</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('taxset')">[税率设置]</a>
            <a href="javascript:;" @click="go('sourceset')">[调整生产]</a>
            <a href="javascript:;" @click="go('citymove')">[地图搬迁]</a>
            <a href="javascript:;" @click="go('rename')">[城市更名]</a>
            <a href="javascript:;" @click="go('cities')">[城市列表/迁建]</a>
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('citystatus')">[城市状态]</a>
            <a href="javascript:;" @click="go('wilds')">[占领野地]</a>
            <a href="javascript:;" @click="go('wareset')">[仓库调配]</a>
            <a href="javascript:;" @click="go('srcstat')">[资源统计]</a>
            <a href="javascript:;" @click="go('troopstat')">[军队统计]</a>
          </div>
          <div class="panel-title">全部建筑总览</div>
          <table>
            <tr><th>建筑</th><th>等级</th><th>状态</th><th>操作</th></tr>
            <template v-for="b in buildings">
              <tr :key="'hb' + b.id">
                <td>{{ b.name }}</td>
                <td>{{ b.level }}/{{ b.max_level }}</td>
                <td>{{ b.status === 0 ? '空闲' : '施工中 ' + remain(b.end_time) }}</td>
                <td>
                  <a v-if="b.status === 0 && b.level > 0 && b.level < b.max_level" href="javascript:;" @click="doUpgrade(b)">[升级]</a>
                  <!-- ★ 与建筑页同一口径：按背包里的建筑加速道具档位渲染 -->
                  <template v-if="b.status !== 0">
                    <a v-for="a in accItems(3)" :key="'hsb' + b.id + '_' + a.cfg_id" href="javascript:;"
                       @click="doSpeedBuilding(b, a)">[加速{{ accLabel(a) }}]</a>
                    <span class="gray" v-if="!accItems(3).length">(无加速道具)</span>
                    <!-- ★ 2026-09-27：施工中可取消（零退还，建筑保留当前等级） -->
                    <a href="javascript:;" class="red" @click="doCancelUpgrade(b)">[取消]</a>
                  </template>
                </td>
              </tr>
              <tr v-if="inlineTip && inlineTip.bid === b.id" :key="'hbt' + b.id">
                <td colspan="4" class="build-tip" :class="inlineTip.type">
                  <span>{{ inlineTip.text }}</span>
                  <a href="javascript:;" @click="inlineTip = null">[关闭]</a>
                </td>
              </tr>
            </template>
          </table>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 市政厅→资源统计(srcstat) 复刻 city/citySourceList.html ============ -->
      <template v-else-if="cur === 'srcstat'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="go('cityhall')">市政厅</a>-&gt;资源统计
          </div>
          <div class="old-line">【{{ city.name }}】:</div>
          <div class="old-line">
            {{ resNames.gold }}：{{ city.gold }}/{{ city.gold_cap }}<br/>
            {{ resNames.food }}：{{ city.food }}/{{ city.food_cap }}<br/>
            {{ resNames.steel }}：{{ city.steel }}/{{ city.steel_cap }}<br/>
            {{ resNames.oil }}：{{ city.oil }}/{{ city.oil_cap }}<br/>
            {{ resNames.rare }}：{{ city.rare }}/{{ city.rare_cap }}
          </div>
          <div class="old-line gray">斜杠后为仓库容量上限；升级仓库可提高保护量与上限。</div>
          <a href="javascript:;" @click="go('cityhall')">[返回]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 市政厅→军队统计(troopstat) 复刻 city/cityTroopList.html ============ -->
      <template v-else-if="cur === 'troopstat'">
        <div class="panel">
          <div class="old-line">
            <a href="javascript:;" @click="go('cityhall')">市政厅</a>-&gt;军队统计
          </div>
          <div class="old-line">【{{ city.name }}】:</div>
          <div class="old-line" v-for="t in troopsData.troops" :key="'ts' + t.troop_id">
            <a href="javascript:;" @click="openTroopView(t.troop_id)">{{ t.name }}</a>：{{ t.count }}
          </div>
          <div class="old-line gray" v-if="!troopsData.troops.length">(城内无部队)</div>
          <div class="old-line">合计：{{ totalTroops }}</div>
          <a href="javascript:;" @click="go('cityhall')">[返回]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 城市迁移(citymove) 复刻 city/cityHallMove.html ============ -->
      <template v-else-if="cur === 'citymove'">
        <div class="panel">
          <div class="panel-title">市政厅 → 城市迁移</div>
          <div class="old-line">当前城市：{{ city.name }}({{ city.x }},{{ city.y }})　所属洲：{{ moveInfo.city ? moveInfo.city.continent : '—' }}</div>
          <div class="old-line gray">
            迁城需要消耗对应道具，道具可在【商城】用黄金或钻石购买（功能相同）。
          </div>
          <hr/>
          <div class="old-line">
            使用【迁城计划】：持有 {{ moveItemCounts.low }} 个
            <span class="gray">(迁移到所选大洲内未被占领的平原)</span>
          </div>
          <div class="old-line">
            迁入大洲：
            <select v-model="moveContinent">
              <option v-for="a in moveInfo.areas" :key="'ma' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="doMoveCity('low')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【高级迁城计划】：持有 {{ moveItemCounts.high }} 个
          </div>
          <div class="old-line gray">
            请确认您输入的坐标是非被占领的平原，沿海平原无法直接迁移城市
          </div>
          <div class="old-line">
            横坐标 x：<input v-model="moveX" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="moveY" type="number" style="width:80px"/>
            <button @click="doMoveCity('high')">确认迁城</button>
          </div>
          <hr/>
          <div class="old-line">
            使用【沿海迁城计划】：持有 {{ moveItemCounts.sea }} 个
            <span class="gray">(沿海城市专用，迁移到沿海平原)</span>
          </div>
          <div class="old-line">
            迁入大洲：
            <select v-model="moveContinentSea">
              <option v-for="a in moveInfo.areas" :key="'ms' + a.id" :value="a.id">{{ a.name }}</option>
            </select>
            <button @click="doMoveCity('sea')">按大洲迁城</button>
          </div>
          <div class="old-line gray">或指定坐标（必须是未被占领的沿海平原）：</div>
          <div class="old-line">
            横坐标 x：<input v-model="moveX2" type="number" style="width:80px"/>
            纵坐标 y：<input v-model="moveY2" type="number" style="width:80px"/>
            <button @click="doMoveCity('sea')">按坐标迁城</button>
          </div>
          <div class="old-line gray">迁城后附属野地不会随城迁移, 需要重新占领。</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('mall')">[去商城买迁城道具]</a>
            <a href="javascript:;" @click="go('bag')">[打开背包]</a>
          </div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 调整生产(sourceset) 复刻 city/sourceSet.html ============ -->
      <template v-else-if="cur === 'sourceset'">
        <div class="panel">
          <div class="panel-title">调整生产（开工率）</div>
          <div class="old-line gray">开工率影响该资源的实际产量：实际产量 = 基础产量 × 开工率 / 100</div>
          <div class="old-line">
            {{ resNames.food }}：
            <input v-model="rateFood" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateFood = 1">[最小]</a>
            <a href="javascript:;" @click="rateFood = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ resNames.steel }}：
            <input v-model="rateSteel" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateSteel = 1">[最小]</a>
            <a href="javascript:;" @click="rateSteel = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ resNames.oil }}：
            <input v-model="rateOil" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateOil = 1">[最小]</a>
            <a href="javascript:;" @click="rateOil = 100">[最大]</a>
          </div>
          <div class="old-line">
            {{ resNames.rare }}：
            <input v-model="rateRare" type="number" min="1" max="100" style="width:70px"/>%
            <a href="javascript:;" @click="rateRare = 1">[最小]</a>
            <a href="javascript:;" @click="rateRare = 100">[最大]</a>
          </div>
          <div class="old-line" style="color:#0000cc">请检查输入数字是否合理（范围 1~100）</div>
          <div class="old-line">
            <button @click="doSaveRate">确认调整</button>
          </div>
          <a href="javascript:;" @click="go('cityhall')">[返回市政厅]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军团(corps) ============ -->
      <template v-else-if="cur === 'corps'">
        <!-- ★ 2026-09-25 用户要求：军团页拆成四栏（纯前端 tab，照抄 rank/acade 页 .acade-tab 写法） -->
        <div class="panel">
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: corpsTab === 'info' }" @click="switchCorpsTab('info')">军团信息</a>|
            <a href="javascript:;" :class="{ on: corpsTab === 'list' }" @click="switchCorpsTab('list')">军团列表</a>|
            <a href="javascript:;" :class="{ on: corpsTab === 'chat' }" @click="switchCorpsTab('chat')">军团聊天</a>|
            <a href="javascript:;" :class="{ on: corpsTab === 'diplomacy' }" @click="switchCorpsTab('diplomacy')">军团外交</a>|
            <a href="javascript:;" :class="{ on: corpsTab === 'war' }" @click="switchCorpsTab('war')">军团宣战</a>|
            <a href="javascript:;" :class="{ on: corpsTab === 'mall' }" @click="switchCorpsTab('mall')">军团商城</a>
          </div>
        </div>

        <!-- ========== ① 军团信息（原有内容整体移入，不删任何原功能） ========== -->
        <template v-if="corpsTab === 'info'">
        <template v-if="myCorps">
          <div class="panel">
            <div class="panel-title">我的军团:{{ myCorps.name }}({{ myCorps.member_count }}人)</div>
            公告: {{ myCorps.notice || '无' }}<br/>
            <!-- ★ 2026-09-25 用户要求：显示军团总积分（来自 /corps/members 的 corps_points） -->
            <div class="old-line">军团总积分: <b>{{ corpsPoints }}</b></div>
            <div class="old-line">
              <template v-if="isLeader">
                <a href="javascript:;" @click="openNoticeEdit()">[修改公告]</a>
                <a class="red" href="javascript:;" @click="doLeaveCorps()">[解散军团]</a>
              </template>
              <a v-else href="javascript:;" @click="doLeaveCorps()">[退出军团]</a>
            </div>
            <div class="panel-title">军团成员</div>
            <table class="ezfy-corps-tbl ezfy-mem-tbl">
              <tr>
                <th>成员</th><th>职位</th><th>军衔</th>
                <!-- ★ 2026-09-25 用户要求：成员表格新增「军团积分」列（m.points，个人军团积分） -->
                <th>军团积分</th>
                <!-- ★ 第九轮：军团长可任命副团长/参谋长 -->
                <th v-if="isLeader" width="150">任命</th>
              </tr>
              <tr v-for="m in corpsMembers" :key="'cm' + m.user_id">
                <td><a href="javascript:;" @click="openPlayer(m.user_id)">{{ m.name }}</a></td>
                <td>{{ m.title || '成员' }}</td>
                <td>{{ m.rank_name }}</td>
                <!-- ★ 2026-09-25：个人军团积分（用于军团商城兑换） -->
                <td>{{ m.points }}</td>
                <td v-if="isLeader">
                  <template v-if="!m.is_leader">
                    <a v-if="m.title !== '副团长'" href="javascript:;" @click="doSetCorpsTitle(m, '副团长')">[副团长]</a>
                    <a v-if="m.title !== '参谋长'" href="javascript:;" @click="doSetCorpsTitle(m, '参谋长')">[参谋长]</a>
                    <a v-if="m.title" class="gray" href="javascript:;" @click="doSetCorpsTitle(m, '')">[撤职]</a>
                  </template>
                  <span v-else class="gray">军团长</span>
                </td>
              </tr>
            </table>
            <div class="old-line" v-if="isLeader && corpsMembers.length > 1">
              踢人:
              <select v-model="kickUserId" class="corps-kick-sel" style="width:30%">
                <option v-for="m in corpsMembers" v-if="!m.is_leader" :key="'kc' + m.user_id" :value="m.user_id">{{ m.name }}</option>
              </select>
              <button @click="doKick">[踢出]</button>
            </div>
            <!-- 军团邮件群发(复刻 CorpsController.mail): ★ 军团长与副团长都能发 -->
            <div class="panel-title" v-if="canMailCorps">军团邮件(群发全体成员)</div>
            <div class="old-line" v-if="canMailCorps">
              <input v-model="corpsMailContent" class="corps-mail-input" placeholder="邮件内容(500字以内)" style="width:60%"/>
              <button @click="doCorpsMail">[群发]</button>
            </div>
          </div>
        </template>
        <!-- ★ 2026-09-30 入团审核（仅军团长）：审核开关 + 待审申请列表 -->
        <div class="panel" v-if="isLeader">
          <div class="panel-title">入团审核</div>
          <div class="old-line">
            <a href="javascript:;" @click="doToggleNeedReview()">
              [{{ myCorpsNeedReview ? '关闭审核·无需审核直接入团' : '开启审核·需军团长审核' }}]
            </a>
          </div>
          <div class="old-line gray">开启后，未入团玩家申请需你在此通过/拒绝；未开启则直接入团。</div>
          <div class="old-line" v-if="corpsApplies.length">
            <div class="old-line" v-for="a in corpsApplies" :key="'ap' + a.apply_id">
              <a href="javascript:;" @click="openPlayer(a.user_id)">{{ a.name }}</a>(id:{{ a.user_id }})
              <a class="green" href="javascript:;" @click="doApplyHandle(a, 1)">[通过]</a>
              <a class="red" href="javascript:;" @click="doApplyHandle(a, 2)">[拒绝]</a>
            </div>
          </div>
          <div class="old-line gray" v-else>(暂无待审申请)</div>
        </div>
        <!-- ★ 2026-09-30 未入团玩家若看的是开启审核的军团，提示需审核 -->
        <div class="old-line gray" v-if="!myCorps && myApplyStatus === 1">你已提交入团申请, 等待军团长审核...</div>
        <div class="panel" v-if="!myCorps">
          <div class="panel-title">创建军团</div>
          <div class="old-line">
            军团名: <input v-model="corpsName" class="corps-name-input" style="width:10%"/>
            <button @click="doCreateCorps">[创建]</button>
          </div>
        </div>
        </template>

        <!-- ========== ★ 2026-09-30 军团列表（独立 tab） ========== -->
        <template v-else-if="corpsTab === 'list'">
          <div class="panel">
            <div class="panel-title">军团列表</div>
            <table>
              <tr><th>军团</th><th>人数</th><th>战力</th><th>操作</th></tr>
              <tr v-for="cp in corpsList" :key="'cp' + cp.id">
                <td>{{ cp.name }}</td>
                <td>{{ cp.member_count }}</td>
                <td>{{ cp.battle_score }}</td>
                <td>
                  <template v-if="!myCorps">
                    <a href="javascript:;" @click="doJoinCorps(cp)">{{ cp.need_review ? '[申请]' : '[加入]' }}</a>
                  </template>
                </td>
              </tr>
            </table>
            <div class="old-line" v-if="!corpsList.length">(暂无军团)</div>
          </div>
        </template>

        <!-- ========== ★ 2026-09-30 军团聊天（独立 tab） ========== -->
        <template v-else-if="corpsTab === 'chat'">
          <div class="panel" v-if="myCorps">
            <div class="panel-title">军团聊天</div>
            <div class="old-line" v-for="m in corpsChats" :key="'cc' + m.id">
              [<a href="javascript:;" @click="openPlayer(m.user_id)">{{ m.user_name }}</a>]:{{ m.content }}
            </div>
            <div class="old-line" v-if="!corpsChats.length">(暂无消息)</div>
            <div class="old-line">
              <input v-model="corpsMsg" class="corps-msg-input" style="width:15%"/>
              <button @click="doCorpsChat">发送</button>
              <a href="javascript:;" @click="loadCorps">[刷新]</a>
            </div>
          </div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="switchCorpsTab('list')">[去军团列表]</a></div>
          </div>
        </template>

        <!-- ========== ② 军团外交 ========== -->
        <template v-else-if="corpsTab === 'diplomacy'">
          <div class="panel" v-if="corpsRelations && corpsRelations.in_corps">
            <div class="panel-title">我的军团(积分):{{ (corpsRelations.my_corps || {}).name }}({{ (corpsRelations.my_corps || {}).points || 0 }})</div>
            <div class="old-line gray">友好/敌对军团均可宣战；关系标记只影响外交显示，不限制宣战。</div>
            <div class="panel-title">已标记关系</div>
            <table>
              <tr><th>军团</th><th>关系</th><th v-if="corpsRelations.can_manage">操作</th></tr>
              <tr v-for="rl in (corpsRelations.relations || [])" :key="'rl' + rl.corps_id">
                <td>{{ rl.name }}</td>
                <td :class="rl.type === 2 ? 'red' : 'green'">{{ rl.type_name }}</td>
                <td v-if="corpsRelations.can_manage">
                  <a href="javascript:;" @click="setCorpsRelation(rl.corps_id, 0)">[取消标记]</a>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(corpsRelations.relations || []).length">(暂无关系标记)</div>
            <div class="panel-title">全部军团</div>
            <table class="ezfy-corps-tbl ezfy-dip-tbl">
              <tr>
                <th>军团</th><th>团长</th><th>人数</th><th>积分</th><th>当前关系</th><th>宣战状态</th>
                <th v-if="corpsRelations.can_manage">操作</th>
              </tr>
              <tr v-for="cp in (corpsRelations.corps_list || [])" :key="'cr' + cp.id">
                <td>{{ cp.name }}</td>
                <td>{{ cp.leader_name || '无' }}</td>
                <td>{{ cp.member_count }}</td>
                <td>{{ cp.points }}</td>
                <td>
                  <span v-if="cp.relation_type === 1" class="green">友好</span>
                  <span v-else-if="cp.relation_type === 2" class="red">敌对</span>
                  <span v-else class="gray">无</span>
                </td>
                <td>
                  <span v-if="cp.war_status === 1 || cp.war_status === 2" class="orange">
                    {{ cp.war_status === 1 ? '宣战待生效' : '交战中' }}<template v-if="cp.war_remaining_h">{{ '（' + cp.war_remaining_h + 'h）' }}</template>
                  </span>
                  <span v-else class="gray">未宣战</span>
                </td>
                <!-- ★ 仅军团长（can_manage）可标记；已是该关系时按钮变成 [取消标记] -->
                <td v-if="corpsRelations.can_manage">
                  <a v-if="cp.relation_type !== 1" href="javascript:;" @click="setCorpsRelation(cp.id, 1)">[友好]</a>
                  <a v-else href="javascript:;" @click="setCorpsRelation(cp.id, 0)">[取消标记]</a>
                  <a v-if="cp.relation_type !== 2" href="javascript:;" @click="setCorpsRelation(cp.id, 2)">[敌对]</a>
                  <a v-else href="javascript:;" @click="setCorpsRelation(cp.id, 0)">[取消标记]</a>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(corpsRelations.corps_list || []).length">(暂无军团)</div>
            <div class="old-line"><a href="javascript:;" @click="loadCorpsRelations">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!corpsRelations"><div class="old-line">正在加载外交数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <!-- ========== ③ 军团宣战 ========== -->
        <template v-else-if="corpsTab === 'war'">
          <div class="panel" v-if="corpsWars && corpsWars.in_corps">
            <div class="panel-title">军团宣战</div>
            <div class="old-line gray">
              规则：宣战后 12 小时生效，48 小时整场结束；生效期间双方成员可互相掠夺/征服并获得军团战绩；友好/敌对军团均可宣战。
            </div>
            <table class="ezfy-corps-tbl ezfy-war-tbl">
              <tr>
                <th>对方军团</th><th>我方身份</th><th>状态</th><th>宣告时间</th>
                <th>我方战绩</th><th>对方战绩</th><th>操作</th>
              </tr>
              <tr v-for="w in (corpsWars.wars || [])" :key="'cw' + w.id">
                <td>{{ w.opp_corps_name }}</td>
                <td>{{ w.mine_is_atk ? '宣战方' : '应战方' }}</td>
                <td>
                  <span v-if="w.status === 1" class="orange">{{ w.status_name }}<template v-if="w.remaining_h">{{ '（' + w.remaining_h + 'h）' }}</template></span>
                  <span v-else-if="w.status === 2" class="red">{{ w.status_name }}<template v-if="w.remaining_h">{{ '（' + w.remaining_h + 'h）' }}</template></span>
                  <span v-else class="gray">{{ w.status_name }}</span>
                </td>
                <!-- ★ 后端下发的是毫秒时间戳，必须走 fmtTime 格式化（否则显示成一串数字） -->
                <td>{{ fmtTime(w.declare_time) }}</td>
                <td>{{ w.my_point }}</td>
                <td>{{ w.opp_point }}</td>
                <!-- 只有进行中(status 1/2)标注进行中；已结束(status 3)显示已结束 -->
                <td>
                  <span v-if="w.status === 1 || w.status === 2" class="gray">进行中</span>
                  <span v-else class="gray">已结束</span>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(corpsWars.wars || []).length">(暂无宣战记录)</div>
            <!-- ★ 军团长可对「未处于宣战中的军团」发起宣战：复用外交 tab 的军团列表，不重复拉接口 -->
            <template v-if="corpsWars.can_manage && corpsRelations && corpsRelations.corps_list">
              <div class="panel-title">全部军团(可宣战)</div>
              <table class="ezfy-corps-tbl ezfy-war-list-tbl">
                <tr><th>军团</th><th>团长</th><th>人数</th><th>宣战状态</th><th>操作</th></tr>
                <tr v-for="cp in (corpsRelations.corps_list || [])" :key="'wcp' + cp.id">
                  <td>{{ cp.name }}</td>
                  <td>{{ cp.leader_name || '无' }}</td>
                  <td>{{ cp.member_count }}</td>
                  <td>
                    <span v-if="cp.war_status === 1 || cp.war_status === 2" class="orange">{{ cp.war_status === 1 ? '宣战待生效' : '交战中' }}</span>
                    <span v-else class="gray">未宣战</span>
                  </td>
                  <td>
                    <a v-if="cp.war_status !== 1 && cp.war_status !== 2" href="javascript:;" @click="declareCorpsWar(cp.id)">[宣战]</a>
                    <span v-else class="gray">进行中</span>
                  </td>
                </tr>
              </table>
            </template>
            <div class="old-line" v-if="corpsWars.can_manage && !(corpsRelations && corpsRelations.corps_list)">
              <a href="javascript:;" @click="loadCorpsRelations">[加载可宣战军团列表]</a>
            </div>
            <div class="old-line"><a href="javascript:;" @click="loadCorpsWars">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!corpsWars"><div class="old-line">正在加载宣战数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <!-- ========== ④ 军团商城（用个人军团积分兑换） ========== -->
        <template v-else-if="corpsTab === 'mall'">
          <div class="panel" v-if="corpsMall.loaded && corpsMall.in_corps">
            <div class="panel-title">军团商城</div>
            <div class="old-line">我的军团积分 <b>{{ corpsMall.my_points }}</b> | 军团总积分 <b>{{ corpsMall.corps_points }}</b></div>
            <table class="ezfy-plain-table">
              <tr><th>商品</th><th>类型</th><th>内容</th><th>价格</th><th>限购/已购</th><th>库存</th><th>操作</th></tr>
              <tr v-for="it in (corpsMall.items || [])" :key="'cmi' + it.id">
                <td>{{ it.name }}</td>
                <td>{{ it.kind_name }}</td>
                <td>
                  <template v-if="it.kind === 1">
                    <span v-if="it.food">{{ resNames.food }}{{ it.food }}</span>
                    <span v-if="it.steel">{{ ' ' + resNames.steel }}{{ it.steel }}</span>
                    <span v-if="it.oil">{{ ' ' + resNames.oil }}{{ it.oil }}</span>
                    <span v-if="it.rare">{{ ' ' + resNames.rare }}{{ it.rare }}</span>
                    <span v-if="it.gold">{{ ' ' + resNames.gold }}{{ it.gold }}</span>
                  </template>
                  <template v-else>{{ it.item_name || it.name }} ×{{ it.item_count }}</template>
                </td>
                <td>{{ it.price }}(个人军团积分)</td>
                <td>
                  <span v-if="it.limit > 0">{{ corpsBought(it.id) }}/{{ it.limit }}</span>
                  <span v-else class="gray">不限购</span>
                </td>
                <!-- ★ 后端下发的是「总库存 stock + 已售 sold」，这里显示剩余量（-1 = 无限） -->
                <td>
                  <span v-if="it.stock < 0" class="green">无限</span>
                  <span v-else :class="(it.stock - it.sold) > 0 ? 'gray' : 'red'">{{ (it.stock - it.sold) > 0 ? (it.stock - it.sold) : '已售罄' }}</span>
                </td>
                <td>
                  <a v-if="corpsMallCanBuy(it)" href="javascript:;" @click="doCorpsMallBuy(it)">[兑换]</a>
                  <span v-else class="gray">[不可兑换]</span>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(corpsMall.items || []).length">(暂无商品)</div>
            <div class="old-line"><a href="javascript:;" @click="loadCorpsMall">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!corpsMall.loaded"><div class="old-line">正在加载商城数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
      </template>

      <!-- ============ 排行(rank) ============ -->
      <template v-else-if="cur === 'rank'">
        <div class="panel">
          <!-- ★ 2026-09-24 用户要求：军衔晋升表/军衔声望榜/兵力榜/军团榜做成 tab 分开展示 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: rankTab === 'prestige' }" @click="rankTab = 'prestige'">军衔声望榜</a>|
            <a href="javascript:;" :class="{ on: rankTab === 'troops' }" @click="rankTab = 'troops'">兵力榜</a>|
            <a href="javascript:;" :class="{ on: rankTab === 'corps' }" @click="rankTab = 'corps'">军团榜</a>|
            <a href="javascript:;" :class="{ on: rankTab === 'ranks' }" @click="rankTab = 'ranks'">军衔晋升表</a>
          </div>

          <!-- 军衔晋升表 tab（静态参照表 + 我的晋升，★ 2026-09-28 加宝物门槛） -->
          <template v-if="rankTab === 'ranks'">
          <div class="panel-title">我的晋升</div>
          <div v-if="rankData.mine" class="old-line">
            当前军衔：<b>{{ rankData.mine.rank_name }}</b>（可建 {{ rankData.mine.city_max }} 座，已有 {{ rankData.mine.city_count }} 座）
          </div>
          <div v-if="rankData.mine && rankData.mine.next" class="old-line">
            下一军衔：<b>{{ rankData.mine.next.name }}</b>（需要声望 <b>{{ rankData.mine.next.need }}</b>，当前 {{ rankData.mine.prestige }}）
            <div class="gray">
              需要宝物：
              <span v-for="t in rankData.mine.next.treasures" :key="'tr' + t.name">
                {{ t.name }}×{{ t.count }}（背包{{ t.have }}）
                <span :class="t.have >= t.count ? 'green' : 'red'">{{ t.have >= t.count ? '足够' : '不足' }}</span>；
              </span>
            </div>
            <button v-if="canPromote()" @click="doPromote">[晋升]</button>
            <span v-else class="gray">声望达标且宝物足够后才能晋升（宝物通过野地采集获得）</span>
          </div>
          <div v-else-if="rankData.mine" class="old-line green">已晋升至最高军衔「{{ rankData.mine.rank_name }}」！</div>

          <div class="panel-title">军衔晋升表</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 10%"><col style="width: 20%"><col style="width: 14%"><col style="width: 18%"><col style="width: 12%"><col style="width: 8%">
            </colgroup>
            <tr><th>等级</th><th>军衔</th><th>职位</th><th>声望</th><th>宝物</th><th>城数</th></tr>
            <template v-for="(r, i) in rankData.ranks">
              <tr :key="'rk' + i">
                <td>{{ i + 1 }}</td>
                <td>
                  <span v-html="rankIcon(r.id)"></span>
                  <span :class="r.name === (rankData.mine ? rankData.mine.rank_name : rankName) ? 'red' : ''">{{ r.name }}</span>
                </td>
                <td>{{ r.post }}</td>
                <td>{{ r.need }}</td>
                <td>
                  <a href="javascript:;" @click="showTreasureRow = showTreasureRow === i ? -1 : i">[宝物]</a>
                </td>
                <td>{{ r.city_max }}</td>
              </tr>
              <tr v-if="showTreasureRow === i" :key="'rt' + i" class="rank-treasure-row">
                <td>所需宝物</td>
                <td colspan="4" class="gray">{{ r.treasures || '该军衔无需宝物' }}</td>
                <td></td>
              </tr>
            </template>
          </table>
          </template>

          <!-- 军衔声望榜 tab -->
          <template v-if="rankTab === 'prestige'">
          <div class="panel-title">军衔声望榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th>声望</th><th>军衔</th></tr>
            <tr v-for="r in rankData.prestige" :key="'rp' + r.rank" :class="rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="openPlayer(r.user_id)">{{ r.name }}</a></td>
              <td>{{ r.prestige }}</td><td>{{ r.rank_name }}</td>
            </tr>
          </table>
          </template>

          <!-- 兵力榜 tab -->
          <template v-if="rankTab === 'troops'">
          <div class="panel-title">兵力榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>统帅</th><th>城市</th><th>兵力</th></tr>
            <tr v-for="r in rankData.troops" :key="'rt' + r.rank" :class="rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span><a href="javascript:;" @click="openPlayer(r.user_id)">{{ r.role_name }}</a></td>
              <td>{{ r.city_name }}</td><td>{{ r.count }}</td>
            </tr>
          </table>
          </template>

          <!-- 军团榜 tab -->
          <template v-if="rankTab === 'corps'">
          <div class="panel-title">军团榜</div>
          <table class="ezfy-rank-table">
            <colgroup>
              <col style="width: 15%"><col style="width: 35%"><col style="width: 25%"><col style="width: 25%">
            </colgroup>
            <tr><th>名次</th><th>军团</th><th>人数</th><th>战力</th></tr>
            <tr v-for="r in rankData.corps" :key="'rc' + r.rank" :class="rankRowCls(r.rank)">
              <td><span class="rank-medal" :class="'m' + r.rank">{{ r.rank }}</span></td>
              <td><span v-if="r.rank === 1" class="rank-crown">♛</span>{{ r.name }}</td><td>{{ r.member_count }}</td><td>{{ r.battle_score }}</td>
            </tr>
          </table>
          </template>

          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 背包(bag) ============ -->
      <template v-else-if="cur === 'bag'">
        <div class="panel">
          <div class="panel-title">背包 <a href="javascript:;" @click="loadBag">[刷新]</a></div>
          <!-- ★ 检索框：道具多的时候按名字/说明筛 -->
          <div class="old-line">
            搜索:
            <input v-model="bagWord" type="text" placeholder="道具名 / 说明关键字"
                   style="width:180px" @input="bagPage = 1"/>
            <a href="javascript:;" @click="bagWord = ''; bagPage = 1">[清空]</a>
            <span class="gray">共 {{ bagFiltered.length }} 种 / 全部 {{ bagItems.length }} 种</span>
          </div>
          <!-- ★ 分类筛选：流式排列自动换行（与商城同款，口径也一致） -->
          <div class="ezfy-slot-grid" v-if="bagCats.length > 1">
            <a href="javascript:;" :class="{ on: bagCat === '' }" @click="setBagCat('')">[全部]</a>
            <a v-for="c in bagCats" :key="'bc' + c" href="javascript:;" :class="{ on: bagCat === c }"
               @click="setBagCat(c)">[{{ c }}]</a>
          </div>
          <!-- ★ 道具说明改成「点 [说明] 才展开」（用户要求：商城/背包都别堆说明文字） -->
          <div class="old-line" v-for="it in bagPaged" :key="'bi' + it.cfg_id">
            <b>{{ it.name }}</b>×{{ it.count }}
            <a v-if="it.description" href="javascript:;" @click="toggleBagDesc(it.cfg_id)">[说明]</a>
            <a href="javascript:;" @click="openUse(it)">[使用]</a><br/>
            <span class="gray" v-if="bagDescId === it.cfg_id">{{ it.description }}</span>

            <!-- 使用面板: 数量 + (军官类道具)目标军官/技能 -->
            <div v-if="useItem && useItem.cfg_id === it.cfg_id" class="use-box">
              数量:
              <input v-model="useCount" type="number" min="1" :max="it.count" style="width:60px"/>
              <span class="gray">/{{ it.count }}</span>
              <a href="javascript:;" @click="useCount = it.count">[全部]</a><br/>

              <template v-if="needOfficer(it)">
                军官:
                <select v-model="useOfficerId">
                  <option :value="0">请选择军官</option>
                  <option v-for="o in bagOfficers" :key="'bo' + o.id" :value="o.id">
                    {{ o.name }} Lv{{ o.level }} {{ o.status_name }}
                  </option>
                </select><br/>
              </template>
              <template v-if="it.item_type === 11">
                技能:
                <select v-model="useSkillId">
                  <option :value="0">请选择技能</option>
                  <option v-for="s in bagSkills" :key="'bs' + s.id" :value="s.id">
                    {{ s.name }}({{ s.effect }})
                  </option>
                </select><br/>
              </template>
              <button @click="doUse(it)">[确认使用]</button>
              <a href="javascript:;" @click="useItem = null">[取消]</a>
            </div>
          </div>
          <!-- ★ 2026-09-28 背包展示宝物（用户要求）：相同宝物合并显示 ×数量 -->
          <div class="panel-title" v-if="bagTreasures.length">宝物</div>
          <div class="old-line" v-for="t in bagTreasures" :key="'bt' + t.cfg_id">
            <b class="orange">{{ t.name }}</b>×{{ t.count }}
          </div>
          <div class="old-line" v-if="!bagItems.length && !bagTreasures.length">(背包空空如也)</div>
          <div class="old-line gray" v-else-if="!bagFiltered.length">(没有匹配「{{ bagWord }}」的道具)</div>
          <!-- ★ 分页 -->
          <div class="ezfy-pager" v-if="bagFiltered.length > bagPageSize">
            <a href="javascript:;" :class="{ disabled: bagPage <= 1 }" @click="bagGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(bagPage, bagTotalPages) }}/{{ bagTotalPages }} 页 · 共 {{ bagFiltered.length }} 种</span>
            <a href="javascript:;" :class="{ disabled: bagPage >= bagTotalPages }" @click="bagGo(1)">[下一页]</a>
          </div>
          <a href="javascript:;" @click="go('mall')">[前往商城]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 宝物(treasure) ============ -->
      <template v-else-if="cur === 'treasure'">
        <div class="panel">
          <div class="panel-title">宝物 <a href="javascript:;" @click="loadBag()">[刷新]</a></div>
          <div class="old-line gray">采集宝物：通过野地采集或「福利 → 宝物签到」获得，可用于军衔晋升、赏赐军官加忠诚。</div>
          <div class="old-line" v-for="t in bagTreasures" :key="'tr' + t.cfg_id">
            <b class="orange">{{ t.name }}</b>×{{ t.count }}
          </div>
          <div class="old-line gray" v-if="!bagTreasures.length">(还没有采集到宝物，去野地采集或宝物签到吧)</div>
          <a href="javascript:;" @click="go('map')">[去野地采集]</a>
          <a href="javascript:;" @click="go('welfare')">[宝物签到]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 商城(mall) ============ -->
      <template v-else-if="cur === 'mall'">
        <div class="panel">
          <div class="panel-title">商城({{ resNames.gold }}{{ city.gold }} · 钻石{{ mallDiamond }})</div>
          <!-- ★ 商城分栏：道具 / 装备散件 / 宝箱（套装件只能开宝箱，商城只卖散件）
               ★ 2026-09-24 用户要求: 三个分栏去掉 []、用 | 分隔并留间距 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: mallTab === 'item' }" @click="switchMallTab('item')">道具</a><span> | </span>
            <a href="javascript:;" :class="{ on: mallTab === 'equipment' }" @click="switchMallTab('equipment')">装备</a><span> | </span>
            <a href="javascript:;" :class="{ on: mallTab === 'chest' }" @click="switchMallTab('chest')">宝箱</a>
          </div>
          <template v-if="mallTab === 'item'">
          <!-- ★ 分类筛选：流式排列自动换行（分类由管理端维护，手机端超宽自动折到下一行） -->
          <div class="ezfy-slot-grid">
            <a href="javascript:;" :class="{ on: mallCat === '' }" @click="setMallCat('')">[全部]</a>
            <a v-for="c in mallCatsList" :key="'mc' + c" href="javascript:;" :class="{ on: mallCat === c }"
               @click="setMallCat(c)">[{{ c }}]</a>
          </div>
          <div class="old-line gray" v-if="mallDiamond <= 0">
            钻石余额为 0；标记为「钻石道具」的道具若标价 0 钻石可直接购买，其余需由管理员充值钻石后购买。
          </div>
          <!-- ★ 道具表格化（原来平铺一长串，用户反馈「乱」）
               ★ 商城保持简洁：只列 名称/价格/库存/操作，**道具说明不在这里显示**
                 （说明改到「背包」里点 [说明] 展开，见 cur==='bag' 那一段）。
               · dual_pay = 黄金/钻石双渠道；标价 0 显示「限时免费」；
               · 库存 -1 = 无限（管理端「数据管理 → 道具配置」维护）。 -->
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>价格</th><th>库存</th><th>操作</th></tr>
            <tr v-for="it in mallPaged" :key="'mi' + it.id">
              <td>{{ it.name }}</td>
              <td>
                <template v-if="it.dual_pay">
                  <span class="orange">{{ it.price_diamond }}钻</span>/{{ it.price_gold }}{{ resNames.gold }}
                </template>
                <span v-else-if="it.is_diamond" class="orange">{{ it.price_diamond > 0 ? it.price_diamond + '钻' : '限时免费' }}</span>
                <span v-else>{{ it.price_gold > 0 ? it.price_gold + resNames.gold : '限时免费' }}</span>
              </td>
              <td>
                <span v-if="it.unlimited" class="green">无限</span>
                <span v-else :class="it.stock > 0 ? 'gray' : 'red'">{{ it.stock > 0 ? it.stock : '已售罄' }}</span>
              </td>
              <td>
                <a v-if="it.unlimited || it.stock > 0" href="javascript:;" @click="openBuy(it)">[购买]</a>
                <span v-else class="gray">[已售罄]</span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!mallPaged.length">(该分类下暂无道具)</div>
          <!-- ★ 分页（每页 10 件） -->
          <div class="ezfy-pager" v-if="mallFiltered.length > mallPageSize">
            <a href="javascript:;" :class="{ disabled: mallPage <= 1 }" @click="mallGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(mallPage, mallTotalPages) }}/{{ mallTotalPages }} 页 · 共 {{ mallFiltered.length }} 件</span>
            <a href="javascript:;" :class="{ disabled: mallPage >= mallTotalPages }" @click="mallGo(1)">[下一页]</a>
          </div>
          </template>
          <!-- ★ 装备散件（管理端在「装备列表」里维护）
               ★ 用户规则：套装装备只能通过宝箱开启，商城不再上架套装件 -->
          <template v-else-if="mallTab === 'equipment'">
            <!-- ★ 说明一律不写进界面（用户要求：别在用户能看见的地方加提示），信息记在这里：
                 · 散件用钻石购买，定价按「六项加成总和」映射到 100~500 钻（见 seed 的 ezfyEquipDiamondPrice）；
                 · 买入后到「军官 → 军官详情」穿到军官身上；
                 · 第一批套装（新兵/战士/混沌…，无系列名）只能通过[宝箱]开启，商城不售。 -->
            <!-- ★ 部位筛选（11 个部位，来自装备距离伤害表）：流式排列自动换行 -->
            <div class="ezfy-slot-grid">
              <a href="javascript:;" :class="{ on: shopSlot === '' }" @click="setShopSlot('')">[全部]</a>
              <a v-for="s in shopSlots" :key="'ss' + s" href="javascript:;" :class="{ on: shopSlot === s }"
                 @click="setShopSlot(s)">[{{ s }}]</a>
            </div>
            <div class="old-line">
              检索：
              <input v-model="shopWord" type="text" placeholder="名称 / 部位" style="width:150px"
                     @input="shopPage = 1"/>
              <span class="gray">共 {{ shopAll.length }} 件</span>
            </div>
            <table class="ezfy-plain-table">
              <!-- ★ 2026-09-25：手机（390px）下列一多名称就被折成 4~5 行 → 砍掉「属性」列
                   （原来只放一个 [查看] 按钮），名称列因此能拿到 38%。 -->
              <colgroup>
                <col style="width:16%"><col style="width:38%"><col style="width:11%"><col style="width:15%"><col style="width:20%">
              </colgroup>
              <tr><th>部位</th><th class="nm">名称</th><th>等级</th><th>价格</th><th>操作</th></tr>
              <!-- ★ 点装备名看这件自己的加成 / 点套装名看套装加成（两个入口看不同内容，买之前就能对比）。 -->
              <template v-for="p in shopPaged">
              <tr :key="'eq' + p.id">
                <td>{{ p.slot }}</td>
                <td class="nm"><a href="javascript:;" @click="toggleDetail(p.id, 'item')">{{ p.name }}</a>
                  <!-- 手机列窄，套装这行尽量短：「需几件」放进展开卡里，不在这里重复 -->
                  <div v-if="setOf(p.set_id)" class="set-mini">
                    套装：<a href="javascript:;" @click="toggleDetail(p.id, 'set')">{{ p.set_name }}</a>
                  </div>
                </td>
                <td>{{ p.level }}</td>
                <td><span class="orange">{{ p.price_diamond }}钻</span></td>
                <td>
                  <a v-if="!p.sold_out" href="javascript:;" @click="openEquipBuy(p)">[购买]</a>
                  <span v-else class="gray">[售罄]</span>
                </td>
              </tr>
              <tr v-if="detailRowId === p.id" :key="'dt' + p.id" class="set-card-row">
                <td :colspan="5">
                  <div class="set-card">
                    <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                    <template v-if="detailMode === 'item'">
                      <div class="sc-h"><b>{{ p.name }}</b>
                        <span :class="qualityClass(p.tier_name)">[{{ p.tier_name || '普通' }}]</span>
                        <span class="gray">{{ p.slot }} · {{ p.level }}级 · 商城在售</span>
                      </div>
                      <div class="sc-b">装备加成：<b class="green">{{ equipAttrText(p) || '（这件没有额外属性加成）' }}</b></div>
                      <div class="sc-b gray" v-if="setOf(p.set_id)">所属套装：{{ setOf(p.set_id).name }}（点套装名看套装加成）</div>
                      <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                    </template>
                    <!-- ② 点「套装名」→ 只看套装加成 -->
                    <template v-else-if="setOf(p.set_id)">
                      <div class="sc-h"><b>{{ setOf(p.set_id).name }}</b>
                        <span :class="qualityClass(setOf(p.set_id).tier_name)">[{{ setOf(p.set_id).tier_name || '特殊' }}]</span>
                        <span class="gray">穿齐 {{ setOf(p.set_id).parts }} 件才生效</span>
                      </div>
                      <div class="sc-b">套装加成：<b class="green">{{ setBonusText(p.set_id) || '（本套装无额外属性加成）' }}</b></div>
                      <div class="sc-b">我的进度：已拥有 <b>{{ setOf(p.set_id).owned || 0 }}</b>/{{ setOf(p.set_id).parts }} 件
                        <span v-if="(setOf(p.set_id).owned || 0) >= setOf(p.set_id).parts" class="green">已够穿齐</span>
                        <span v-else class="red">还差 {{ setOf(p.set_id).parts - (setOf(p.set_id).owned || 0) }} 件</span>
                      </div>
                      <div class="sc-b gray" v-if="setOf(p.set_id).slots && setOf(p.set_id).slots.length">部位：{{ setOf(p.set_id).slots.join(' / ') }}</div>
                      <div class="sc-b gray">点装备名看这件自己的加成</div>
                    </template>
                    <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                  </div>
                </td>
              </tr>
              </template>
            </table>
            <div class="old-line gray" v-if="!shopAll.length">(没有匹配的装备)</div>
            <div class="ezfy-pager" v-if="shopAll.length > shopSize">
              <a href="javascript:;" :class="{ gray: shopPage <= 1 }" @click="shopPage--">上一页</a>
              <span class="gray">第 {{ shopPage }}/{{ shopTotalPages }} 页（共 {{ shopAll.length }} 件）</span>
              <a href="javascript:;" :class="{ gray: shopPage >= shopTotalPages }" @click="shopPage++">下一页</a>
            </div>
            <div class="old-line gray">
              当前余额：{{ resNames.gold }}{{ fmtN(equipShop.gold) }} · 钻石{{ equipShop.diamond }}
            </div>
          </template>
          <!-- ★ 宝箱（用钻石/黄金买，开箱按权重出套装件；奖池由管理端维护）
               ★ 用户规则：套装军官装备的**唯一**获取途径就是这里 -->
          <template v-else>
            <!-- ★ 宝箱（说明不写进界面，记在这里）：
                 · 宝箱用钻石购买（战地补给箱用黄金）；价格 300~800 钻按品质分档；
                 · 套装宝箱开出的是「整套」（一次给该套全部件，见 ezfyGrantChestPrize 的 Kind=3）；
                   战地补给箱开单件散件；
                 · 奖池**不直接铺开** —— 点宝箱名字才展开（用户要求「别直接展示」）。 -->
            <!-- ★ 宝箱列表：奖池点名字才展开 -->
            <table class="ezfy-plain-table">
              <tr><th>宝箱</th><th>价格</th><th>奖池</th><th>操作</th></tr>
              <tr v-for="ch in chestData.chests" :key="'ch' + ch.id">
                <td>
                  <a href="javascript:;" :class="{ on: chestPoolId === ch.id }"
                     @click="toggleChestPool(ch.id)">{{ ch.name }}</a>
                  <span v-if="ch.stock >= 0" :class="ch.stock > 0 ? 'gray' : 'red'">
                    （库存{{ ch.stock > 0 ? ch.stock : '0已售罄' }}）
                  </span>
                </td>
                <td>
                  <span v-if="ch.price_diamond > 0" class="orange">{{ ch.price_diamond }}钻</span>
                  <span v-else>{{ ch.price_gold }}{{ resNames.gold }}</span>
                </td>
                <td class="gray">{{ ch.pool.length }} 项</td>
                <td>
                  <a v-if="!ch.sold_out" href="javascript:;" @click="openChestBuy(ch)">[开箱]</a>
                  <span v-else class="gray">[售罄]</span>
                </td>
              </tr>
            </table>
            <!-- ★ 奖池（点宝箱名字才展开）：检索 + 分页 -->
            <template v-if="chestPoolCur">
              <div class="old-line">
                <b>{{ chestPoolCur.name }}</b> 奖池
                <span class="gray">{{ chestPoolCur.des }}</span>
                <a href="javascript:;" @click="chestPoolId = 0">[收起]</a>
              </div>
              <div class="old-line">
                检索：<input v-model="chestPoolWord" type="text" placeholder="奖品名称" style="width:150px"
                       @input="chestPoolPage = 1"/>
                <span class="gray">共 {{ chestPoolAll.length }} 项</span>
              </div>
              <table class="ezfy-plain-table">
                <tr><th>奖品</th><th>品质</th><th>数量</th></tr>
                <tr v-for="(p, i) in chestPoolPaged" :key="'cp' + p.kind + '_' + p.ref_id + '_' + i">
                  <td>{{ p.name }}</td>
                  <td :class="qualityClass(p.quality)">{{ p.quality }}</td>
                  <td>{{ p.kind === 3 ? '整套' : ('×' + p.count) }}</td>
                </tr>
              </table>
              <div class="old-line gray" v-if="!chestPoolAll.length">(没有匹配的奖品)</div>
              <div class="ezfy-pager" v-if="chestPoolAll.length > chestPoolSize">
                <a href="javascript:;" :class="{ gray: chestPoolPage <= 1 }" @click="chestPoolPage--">上一页</a>
                <span class="gray">第 {{ chestPoolPage }}/{{ chestPoolTotalPages }} 页（共 {{ chestPoolAll.length }} 项）</span>
                <a href="javascript:;" :class="{ gray: chestPoolPage >= chestPoolTotalPages }" @click="chestPoolPage++">下一页</a>
              </div>
            </template>
            <div class="old-line gray" v-if="!chestData.chests.length">(暂无上架宝箱，请等管理员在后台配置)</div>
            <div class="old-line gray">
              当前余额：{{ resNames.gold }}{{ fmtN(chestData.gold) }} · 钻石{{ chestData.diamond }}
            </div>
            <div class="old-line" v-if="chestResult && chestResult.length"><b>上次开箱结果：</b></div>
            <div class="old-line" v-for="(r, i) in chestResult" :key="'cr' + i">
              {{ r.name }}<span :class="qualityClass(r.quality)">[{{ r.quality }}]</span>
            </div>
          </template>
          <a href="javascript:;" @click="go('bag')">[背包]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 商城·道具购买详情页（跳转新页面确认） ============ -->
      <template v-else-if="cur === 'mallbuy'">
        <div class="panel">
          <div class="panel-title">购买道具</div>
          <div class="old-line gray">当前余额：{{ resNames.gold }}{{ fmtN(city.gold) }} · 钻石{{ mallDiamond }}</div>
          <template v-if="buyItem">
            <div class="old-line">
              <b>{{ buyItem.name }}</b>
              <span v-if="buyItem.category" class="gray">[{{ buyItem.category }}]</span>
            </div>
            <div class="old-line" v-if="buyItem.description">{{ buyItem.description }}</div>
            <div class="old-line">
              价格：
              <template v-if="buyItem.dual_pay">
                <span class="orange">{{ buyItem.price_diamond }}钻</span>/{{ buyItem.price_gold }}{{ resNames.gold }}
              </template>
              <span v-else-if="buyItem.is_diamond" class="orange">{{ buyItem.price_diamond > 0 ? buyItem.price_diamond + '钻' : '限时免费' }}</span>
              <span v-else>{{ buyItem.price_gold > 0 ? buyItem.price_gold + resNames.gold : '限时免费' }}</span>
            </div>
            <div class="old-line">
              库存：
              <span v-if="buyItem.unlimited" class="green">无限</span>
              <span v-else :class="buyItem.stock > 0 ? 'gray' : 'red'">{{ buyItem.stock > 0 ? buyItem.stock : '已售罄' }}</span>
            </div>
            <div class="old-line">
              数量：<input v-model="buyCount" type="number" min="1" :max="buyMaxOf(buyItem)" style="width:70px"/>
              <span class="gray">{{ buyItem.unlimited ? ('单次最多 ' + mallBuyMax + ' 个') : ('最多 ' + buyMaxOf(buyItem)) }}</span>
            </div>
            <div class="old-line" v-if="buyItem.dual_pay">
              支付方式：
              <select v-model="buyPayWith">
                <option value="gold">黄金 {{ buyItem.price_gold * (parseInt(buyCount) || 0) }}</option>
                <option value="diamond">钻石 {{ buyItem.price_diamond * (parseInt(buyCount) || 0) }}</option>
              </select>
            </div>
            <div class="old-line">
              合计：
              <b class="bb-total">
                <template v-if="buyItem.dual_pay">{{ (buyPayWith === 'diamond' ? buyItem.price_diamond : buyItem.price_gold) * (parseInt(buyCount) || 0) }} {{ buyPayWith === 'diamond' ? '钻石' : resNames.gold }}</template>
                <template v-else-if="buyItem.is_diamond">{{ buyItem.price_diamond > 0 ? (buyItem.price_diamond * (parseInt(buyCount) || 0) + ' 钻石') : '限时免费' }}</template>
                <template v-else>{{ buyItem.price_gold > 0 ? (buyItem.price_gold * (parseInt(buyCount) || 0) + ' ' + resNames.gold) : '限时免费' }}</template>
              </b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="doBuy(buyItem)">[确认购买]</a>
              <a href="javascript:;" @click="buyItem = null; go('mall')">[取消]</a>
            </div>
          </template>
          <div class="old-line gray" v-else>(未选择道具)</div>
          <a href="javascript:;" @click="go('mall')">[返回商城]</a>
        </div>
      </template>

      <!-- ============ 商城·装备散件购买详情页 ============ -->
      <template v-else-if="cur === 'equipbuy'">
        <div class="panel">
          <div class="panel-title">购买装备散件</div>
          <div class="old-line gray">当前余额：钻石{{ equipShop.diamond }}</div>
          <template v-if="equipShopBuy">
            <div class="old-line">
              <b>{{ equipShopBuy.name }}</b>
              <span class="gray">[{{ equipShopBuy.slot }}]</span>
              <span v-if="equipShopBuy.sold_out" class="red">[已售罄]</span>
            </div>
            <div class="old-line">等级：{{ equipShopBuy.level }}</div>
            <div class="old-line">属性：{{ equipAttrText(equipShopBuy) || '—' }}</div>
            <div class="old-line">价格：<span class="orange">{{ equipShopBuy.price_diamond }}钻</span>/件</div>
            <div class="old-line">
              数量：<input v-model="equipShopCount" type="number" min="1" style="width:70px"/>
            </div>
            <div class="old-line">
              合计：<b class="bb-total">{{ equipShopBuy.price_diamond * (parseInt(equipShopCount) || 0) }} 钻石</b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="doBuyEquip(equipShopBuy)">[确认购买]</a>
              <a href="javascript:;" @click="equipShopBuy = null; go('mall')">[取消]</a>
            </div>
          </template>
          <div class="old-line gray" v-else>(未选择装备)</div>
          <a href="javascript:;" @click="go('mall')">[返回商城]</a>
        </div>
      </template>

      <!-- ============ 商城·宝箱开箱详情页 ============ -->
      <template v-else-if="cur === 'chestopen'">
        <div class="panel">
          <div class="panel-title">开宝箱</div>
          <div class="old-line gray">当前余额：{{ resNames.gold }}{{ fmtN(chestData.gold) }} · 钻石{{ chestData.diamond }}</div>
          <template v-if="chestOpen">
            <div class="old-line"><b>{{ chestOpen.name }}</b></div>
            <div class="old-line" v-if="chestOpen.des">{{ chestOpen.des }}</div>
            <div class="old-line">
              价格：
              <span v-if="chestOpen.price_diamond > 0" class="orange">{{ chestOpen.price_diamond }}钻/个</span>
              <span v-else>{{ chestOpen.price_gold }}{{ resNames.gold }}/个</span>
            </div>
            <div class="old-line">奖池（{{ chestOpen.pool.length }} 项）<span class="gray">（点奖品名可查看具体属性）</span>：</div>
            <table class="ezfy-plain-table">
              <tr><th>奖品</th><th>品质</th><th>数量</th></tr>
              <template v-for="(p, i) in chestOpen.pool" :key="'cpo' + p.kind + '_' + p.ref_id + '_' + i">
                <tr>
                  <td>
                    <a href="javascript:;"
                       :class="{ on: chestOpenDetailIdx === i }"
                       @click="chestOpenDetailIdx = chestOpenDetailIdx === i ? -1 : i">{{ p.name }}</a>
                  </td>
                  <td :class="qualityClass(p.quality)">{{ p.quality }}</td>
                  <td>{{ p.kind === 3 ? '整套' : ('×' + p.count) }}</td>
                </tr>
                <tr v-if="chestOpenDetailIdx === i">
                  <td colspan="4" class="gray">{{ p.detail || '（无更多说明）' }}</td>
                </tr>
              </template>
            </table>
            <div class="old-line">
              数量
              <a href="javascript:;" @click="chestCount = 1">[1]</a>
              <a href="javascript:;" @click="setChestCount(5)">[5]</a>
              <a href="javascript:;" @click="setChestCount(10)">[10]</a>
              <input v-model="chestCount" type="number" min="1" :max="chestOpen.open_max" style="width:70px"/>
              <span class="gray">单次最多 {{ chestOpen.open_max }} 个</span>
            </div>
            <div class="old-line">
              合计：
              <b class="bb-total">{{ (chestOpen.price_diamond > 0 ? chestOpen.price_diamond : chestOpen.price_gold) * (parseInt(chestCount) || 0) }} {{ chestOpen.price_diamond > 0 ? '钻石' : resNames.gold }}</b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="doOpenChest(chestOpen)">[确认开箱]</a>
              <a href="javascript:;" @click="chestOpen = null; go('mall')">[取消]</a>
            </div>
          </template>
          <template v-else-if="chestResult && chestResult.length">
            <div class="old-line"><b>开箱结果：</b></div>
            <div class="old-line" v-for="(r, i) in chestResult" :key="'cres' + i">
              {{ r.name }}<span class="gray">({{ r.quality }})</span>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="chestResult = []; go('mall')">[返回商城]</a>
            </div>
          </template>
          <a href="javascript:;" @click="go('mall')">[返回商城]</a>
        </div>
      </template>

      <!-- ============ 交易行(exchange) ============ -->
      <template v-else-if="cur === 'exchange'">
        <div class="panel">
          <div class="panel-title">资源交易行({{ resNames.gold }}{{ exchangeGold }})</div>
          <div class="old-line gray">购买他人挂单的资源; 也可挂单出售资源换取{{ resNames.gold }}。</div>
          <div class="panel-title">卖家挂单</div>
          <div class="old-line">
            类别:
            <select v-model="exFilter" style="width:80px" @change="onExFilter">
              <option :value="0">全部</option>
              <option value="1">{{ resNames.food }}</option><option value="2">{{ resNames.steel }}</option>
              <option value="3">{{ resNames.oil }}</option><option value="4">{{ resNames.rare }}</option>
            </select>
            <a v-if="exFilter" href="javascript:;" @click="exFilter = 0; onExFilter()">[全部]</a>
          </div>
          <table class="ezfy-ex-tbl">
            <tr><th>卖家</th><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in exchangeOrders" :key="'eo' + e.id">
              <td>{{ e.seller_name }}</td>
              <td>{{ e.type_name }}</td>
              <td>{{ fmtN(e.count) }}</td>
              <td>{{ fmtN(e.total_price) }}{{ e.currency_name || resNames.gold }}</td>
              <td><a href="javascript:;" @click="doExchangeBuy(e)">[购买]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!exchangeOrders.length">(暂无在售订单)</div>
          <div class="ezfy-pager" v-if="exchangeTotal > exchangeSize">
            <a href="javascript:;" :class="{ gray: exchangePage <= 1 }" @click="sectionPagerGo('exo', -1)">上一页</a>
            <span class="gray">第 {{ exchangePage }}/{{ exchangeTotalPages }} 页（共 {{ exchangeTotal }} 条）</span>
            <a href="javascript:;" :class="{ gray: exchangePage >= exchangeTotalPages }" @click="sectionPagerGo('exo', 1)">下一页</a>
          </div>
          <div class="panel-title">我的挂单</div>
          <table class="ezfy-ex-tbl" v-if="exchangeMine.length">
            <tr><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in exchangeMine" :key="'em' + e.id">
              <td>{{ e.type_name }}</td>
              <td>{{ fmtN(e.count) }}</td>
              <td>{{ fmtN(e.total_price) }}{{ e.currency_name || resNames.gold }}</td>
              <td><a href="javascript:;" @click="doExchangeCancel(e)">[下架]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!exchangeMine.length">(无在售挂单)</div>
          <div class="ezfy-pager" v-if="exchangeMTotal > exchangeMSize">
            <a href="javascript:;" :class="{ gray: exchangeMPage <= 1 }" @click="sectionPagerGo('exm', -1)">上一页</a>
            <span class="gray">第 {{ exchangeMPage }}/{{ exchangeMTotalPages }} 页（共 {{ exchangeMTotal }} 条）</span>
            <a href="javascript:;" :class="{ gray: exchangeMPage >= exchangeMTotalPages }" @click="sectionPagerGo('exm', 1)">下一页</a>
          </div>
          <div class="panel-title">挂单出售</div>
          <div class="old-line">
            资源:
            <select v-model="sellType" style="width:70px">
              <option value="1">{{ resNames.food }}</option><option value="2">{{ resNames.steel }}</option>
              <option value="3">{{ resNames.oil }}</option><option value="4">{{ resNames.rare }}</option>
            </select><br/>
            数量: <input v-model="sellCount" type="number" style="width:90px"/><br/>
            总价({{ resNames.gold }}): <input v-model="sellPrice" type="number" style="width:90px" max="1000000000" placeholder="单价≤100"/><br/>
            <div class="old-line gray">单价不得超过 100 {{ resNames.gold }}/单位（可配，1:100 卡控）</div>
            <button @click="doExchangeSell">[挂单出售]</button>
          </div>
          <div class="panel-title">向系统出售</div>
          <div class="old-line">
            资源:
            <select v-model="sellSysType" style="width:70px">
              <option value="1">{{ resNames.food }}</option><option value="2">{{ resNames.steel }}</option>
              <option value="3">{{ resNames.oil }}</option><option value="4">{{ resNames.rare }}</option>
            </select><br/>
            数量: <input v-model="sellSysCount" type="number" style="width:90px"/><br/>
            <div class="old-line gray">
              每100单位 → {{ sysSellRatio[sellSysType] || 0 }} {{ resNames.gold }}，实得再扣 {{ sysSellFee }}%（应得 {{ sysSellPreview() }} {{ resNames.gold }}）
            </div>
            <button @click="doExchangeSysSell">[向系统出售]</button>
            <div v-if="sysSellLostWarn" class="red">{{ sysSellLostWarn }}</div>
          </div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 活动(activity) ============ -->
      <template v-else-if="cur === 'activity'">
        <div class="panel">
          <div class="panel-title">活动</div>
          <!-- 复刻 activityIndex.html 的【活动玩法】说明(在地图中寻找的固定位置活动目标) -->
          <div class="old-line"><b>【活动玩法】</b>(在【地图】中寻找, 固定位置刷新)：</div>
          <div class="old-line">
            <span class="orange">活动野地</span>【活动】(标记: 活动野地N级)<br/>
            陆/海随机刷新, 10万~30万守军, 胜利获得大量资源+{{ resNames.gold }}(元宝)+必定掉落宝物+大量声望<br/>
            等级越高守军越强, 奖励越丰厚
          </div>
          <div class="old-line">
            <span class="orange">活动寇城</span>【活动寇】(标记: 活动寇N级)<br/>
            20万~60万守军, 胜利获得巨大资源+{{ resNames.gold }}+宝物+声望
          </div>
          <div class="old-line">
            <span class="red">特殊城市</span>【特殊】(标记: 特殊城市N级)<br/>
            100万~500万守军, 全服最强活动目标, 需要强力的部队!<br/>
            胜利必定获得高级/特殊宝物, 巨额{{ resNames.gold }}与资源
          </div>
          <div class="old-line gray">活动目标无法占领, 战胜只结算奖励, 不占附属野地上限。</div>
          <a href="javascript:;" @click="go('map')">[前往地图]</a>
          <hr/>
          <div class="old-line"><b>【节日活动】</b></div>
          <template v-if="activities.length">
            <div class="old-line" v-for="a in activities" :key="'ac' + a.id">
              <b>{{ a.name }}</b>
              <span :class="a.running ? 'green' : 'gray'">[{{ a.running ? '进行中' : '未开启' }}]</span>
              <span class="orange">{{ a.effect }}</span><br/>
              {{ a.des }}<br/>
              <span v-if="a.running" class="gray">剩余: {{ fmtLeft(a.left_sec) }}</span>
              <span v-else-if="a.start_time && a.end_time" class="gray">
                时间: {{ fmtTime(a.start_time) }} ~ {{ fmtTime(a.end_time) }}
              </span>
            </div>
          </template>
          <div class="old-line gray" v-else>(暂无节日活动, 敬请期待)</div>
          <hr/>
          <div class="old-line gray">活动类型: 资源增产 / 造兵打折 / 建造加速 / 研究加速 / 声望加成</div>
          <hr/>
          <div class="old-line">[开服活动] 新手礼包、每周福利、市政厅等级礼包持续发放中, 前往<a href="javascript:;" @click="go('welfare')">[福利]</a>领取。</div>
          <div class="old-line">[征战天下] 征服野地/寇城可获得军功声望, 声望晋升军衔!</div>
          <div class="old-line">[物资兑换] 交易所开放资源交易, 低买高卖赚{{ resNames.gold }}。</div>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 福利(welfare) ============ -->
      <template v-else-if="cur === 'welfare'">
        <div class="panel">
          <!-- ★ 2026-09-28 用户要求：签到/礼包/宝物签到 拆成 tab 展示（照抄 rank 页 .acade-tab 写法） -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: welfareTab === 0 }" @click="setWelfareTab(0)">每日签到</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: welfareTab === 1 }" @click="setWelfareTab(1)">礼包</a><span
              class="acade-sep">.</span><a href="javascript:;" :class="{ on: welfareTab === 2 }" @click="setWelfareTab(2)">宝物签到</a>
          </div>

          <!-- 每日签到 -->
          <template v-if="welfareTab === 0">
            <div class="old-line">
              <span v-if="welfare.signed_today">今日已签到(连续{{ welfare.sign_count }}天)</span>
              <a v-else href="javascript:;" @click="doSign">[签到领奖]</a>
              | 声望:{{ welfare.prestige }}({{ welfare.rank_name }})
            </div>
            <div class="old-line" v-for="r in welfare.rewards" :key="'sr' + r.day">
              第{{ r.day }}天:{{ r.reward }}
            </div>
          </template>

          <!-- 礼包 -->
          <!-- ★ 用户要求「把 [市政厅20级礼包][市政厅30级礼包][市政厅40级礼包] 删掉」：
               只保留 新手 / 每周 / 市政厅10级 三个入口（后端 Gift 同步去掉 20/30/40 分支）。 -->
          <template v-else-if="welfareTab === 1">
            <div class="old-line">
              <a href="javascript:;" @click="doGift('newbie')">{{ welfare.gifts.newbie ? '[新手礼包已领]' : '[新手礼包]' }}</a>
              <a href="javascript:;" @click="doGift('weekly')">{{ welfare.gifts.weekly ? '[每周福利已领]' : '[每周福利]' }}</a><br/>
              <a href="javascript:;" @click="doGift('level10')">{{ welfare.gifts.level10 ? '[市政厅10级礼包已领]' : '[市政厅10级礼包]' }}</a>
            </div>
          </template>

          <!-- 宝物签到：7 天一轮，逢 5/6/7 天多给（懒人也能攒晋升宝物） -->
          <template v-else>
            <div class="old-line">
              宝物签到
              <span class="gray">每天领随机宝物，7天一轮：第1-4天×2、第5天×4、第6天×6、第7天×8</span>
            </div>
            <div class="old-line">
              <span v-if="welfare.treasure_signed_today" class="green">今日已签(连续{{ welfare.treasure_count }}天)</span>
              <a v-else href="javascript:;" @click="doTreasureSign">[宝物签到领奖]</a>
              <span v-if="welfare.treasure_count" class="gray">| 连续{{ welfare.treasure_count }}天</span>
            </div>
            <!-- ★ 2026-09-28 用户要求：展示本轮已签到领到的具体宝物名, 每轮(每天签到)后更新 -->
            <div class="old-line" v-if="welfare.treasure_signed_today && welfare.treasure_reward">
              本轮已领宝物：<span class="green">{{ welfare.treasure_reward.split(',').join('、') }}</span>
            </div>
          </template>

          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 公告(notices) ============ -->
      <template v-else-if="cur === 'notices'">
        <div class="panel">
          <div class="panel-title">公告</div>
          <!-- ★ 用户要求「公告也变成分页，下一页上一页那种」→ 与军情三区同一套 .ezfy-pager 写法
               （默认每页 5 条，见 noticeSize）。 -->
          <div class="old-line" v-for="n in noticePaged" :key="'nn' + n.id">
            <span v-if="n.is_top" class="red">[置顶]</span>
            <a href="javascript:;" @click="openNotice(n)">{{ n.title }}</a>
            <!-- ★ 用户要求：公告标题后展示发布时间（年-月-日）。CreatedAt(time.Time) JSON 序列化为
                 "2026-09-23T11:11:28+08:00"，前端只取 "年-月-日" 并加 [] 色弱化。 -->
            <span class="gray">[{{ fmtDate(n.created_at) }}]</span>
          </div>
          <div class="old-line" v-if="!notices.length">(暂无公告)</div>
          <div class="ezfy-pager" v-if="notices.length > noticeSize">
            <a href="javascript:;" :class="{ gray: noticePage <= 1 }" @click="sectionPagerGo('notice', -1)">上一页</a>
            <span class="gray">第 {{ noticePage }}/{{ noticeTotalPages }} 页（共 {{ notices.length }} 条）</span>
            <a href="javascript:;" :class="{ gray: noticePage >= noticeTotalPages }" @click="sectionPagerGo('notice', 1)">下一页</a>
          </div>
          <template v-if="curNotice">
            <div class="panel-title">{{ curNotice.title }}</div>
            <div class="old-line">{{ curNotice.content }}</div>
          </template>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 联络中心(liaison) 复刻 liaison/liaisonIndex.html ============ -->
      <template v-else-if="cur === 'liaison'">
        <div class="panel">
          <div class="old-line">联络中心（{{ liaison.level }}级）</div>
          <div class="old-line">联络中心是盟友间互相联络的建筑</div>
          <div class="old-line">
            1级联络中心可以 加入联盟，<br/>
            2级联络中心可以 创建联盟<br/>
            创建联盟需消耗{{ liaison.create_cost }}{{ resNames.gold }}<br/>
            每级联络中心可以多一支盟友驻军、多{{ liaison.member_per_level }}人联盟人数上限
          </div>
          <div class="old-line gray">使用同盟密令1个可以将联络中心升级至11级（原版道具，本项目未开放）</div>
          <div class="old-line red" v-if="liaison.level < 1">
            尚未建造联络中心, 无法加入或创建联盟
            <a href="javascript:;" @click="go('buildm')">[前往军事区建造]</a>
          </div>

          <div class="old-line">我的联盟：</div>
          <template v-if="liaison.my_corps">
            <div class="old-line">
              <b>{{ liaison.my_corps.name }}</b>(成员{{ liaison.member_count }}/{{ liaison.member_cap }})
              <a href="javascript:;" @click="go('corps')">[进入军团]</a>
            </div>
            <div class="old-line gray">公告：{{ liaison.my_corps.notice || '暂无公告' }}</div>
          </template>
          <div class="old-line" v-else-if="liaison.level >= 1">
            <a href="javascript:;" @click="go('corps')">加入联盟</a>
            <template v-if="liaison.can_create">
              &nbsp;<a href="javascript:;" @click="go('corps')">创建联盟</a>
            </template>
            <span v-else class="red">（创建联盟需2级联络中心）</span>
          </div>
          <div class="old-line gray" v-else>（尚未建造联络中心）</div>

          <div class="old-line">盟军驻军：</div>
          <table>
            <tr><th>来自城市</th><th>军官</th><th>驻军</th></tr>
            <tr v-for="g in liaison.garrisons" :key="'lg' + g.id">
              <td>{{ g.from_city }}</td>
              <td>{{ g.officer || '无' }}</td>
              <td>
                <span v-for="(t, i) in g.troops" :key="'lgt' + i">{{ t.name }}×{{ t.count }} </span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!liaison.garrisons.length">(暂无盟军驻军)</div>
          <div class="old-line gray">
            驻军上限 {{ liaison.garrison_used }}/{{ liaison.garrison_cap }}；
            联盟成员可用「增援」把部队派到你的城市协防。
          </div>
          <a href="javascript:;" @click="go('buildm')">[返回军事区]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 统帅(info) ============ -->
      <template v-else-if="cur === 'info'">
        <div class="panel">
          <div class="panel-title">统帅信息</div>
          <!-- ★ 只展示「玩家号码(游戏ID)」——不展示家园号码 -->
          玩家号码：{{ selfInfo.game_uid || profile.game_uid || userBrief.game_uid || '—' }}<br/>
          昵称：{{ selfInfo.nickname || profile.nickname }}
          <template v-if="!renameEditing">
            <a href="javascript:;" @click="startRename">[修改昵称]</a>
          </template>
          <template v-else>
            <br/>
            <input v-model="renameInput" maxlength="12" placeholder="2~12 个字符" style="width:130px"/>
            <button @click="doPlayerRename">确定</button>
            <a href="javascript:;" @click="renameEditing = false">[取消]</a>
          </template>
          <!-- ★ 2026-09-25 去掉内联 font-size:13px，改为继承全站统一字号（--fs） -->
          <div class="gray">{{ renameHint }}</div>
          阵营：{{ selfInfo.camp_name || (profile.camp === 2 ? '轴心国' : '同盟国') }}
          <a href="javascript:;" @click="doChangeCamp(1)">[转同盟国]</a>
          <a href="javascript:;" @click="doChangeCamp(2)">[转轴心国]</a>
          <!-- ★ 2026-09-25 去掉内联 font-size:13px，改为继承全站统一字号（--fs） -->
          <div class="gray">{{ campHint }}</div>
          声望：{{ profile.prestige }}<br/>
          军衔：{{ rankName }}({{ rankPost }})<span style="margin-left:4px"><span v-html="rankIcon(myRankId)"></span></span><br/>
          军团：{{ (myCorps && myCorps.name) || '无' }}<br/>
          <!-- ★ 2026-09-27 用户要求：统帅信息展示军团；有军团职务(军团长/副团长/参谋长)才展示职务 -->
          <template v-if="myCorpsTitle">职务：{{ myCorpsTitle }}<br/></template>
          城市数：{{ cities.length }}<br/>
          人口数：{{ city.pop }}<br/>
          军官数：{{ officerCount }}<br/>
          <br/>
          总兵力：{{ totalTroops }}<br/>
          城外行进：{{ marching }}支 | 驻守采集：{{ occupying }}支<br/>
          占领野地：{{ wildlands.length }}块<br/>
          <a href="javascript:;" @click="go('friends')">[申请好友]</a>
          <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 他人统帅信息(playerinfo) 复刻 PlayerController.infoOther + user/info.html ============ -->
      <!-- 游戏是沉浸式的: 点玩家名只看这一页(二战风云的数据), 不允许跳去家园个人主页 /user/:id -->
      <template v-else-if="cur === 'playerinfo'">
        <div class="panel" v-if="playerInfo">
          <div class="panel-title">统帅信息</div>
          <div class="old-line">
            <b :style="nickColorAt(playerInfo.color, 0)">{{ playerInfo.nickname }}</b>
            <span class="gray">(玩家号码 {{ playerInfo.game_uid || playerInfo.user_id }})</span>
          </div>
          <div class="old-line">
            阵营：{{ playerInfo.camp_name }}<br/>
            声望：{{ playerInfo.prestige }}<br/>
            军衔：{{ playerInfo.rank_name }}({{ playerInfo.rank_post }})<span style="margin-left:4px"><span v-html="rankIcon(rankIdByName(playerInfo.rank_name))"></span></span><br/>
            军团：{{ playerInfo.corps_name || '无' }}<br/>
            <!-- ★ 2026-09-29 用户要求：他人统帅页展示军团职务（与我的统帅页一致），职务在军团下一行 -->
            <template v-if="playerInfo.corps_title">职务：{{ playerInfo.corps_title }}<br/></template>
            城市数：{{ playerInfo.city_count }}<br/>
            军官数：{{ playerInfo.officer_count }}<br/>
            城市最高兵力数：{{ fmtN(playerInfo.troop_max) }}<br/>
            占领野地：{{ playerInfo.wild_count }}块
          </div>
          <div class="old-line">
            <template v-if="playerInfo.is_self">
              <span class="gray">这是你自己</span>
              <a href="javascript:;" @click="go('info')">[我的统帅页]</a>
            </template>
            <template v-else-if="playerInfo.is_friend">
              <span class="green">已是好友</span>
              <a href="javascript:;" @click="openPm(playerInfo.user_id)">[发私信]</a>
            </template>
            <template v-else-if="playerInfo.is_applied">
              <span class="orange">好友申请已发送, 等待对方处理</span>
            </template>
            <template v-else>
              <a href="javascript:;" @click="doAddFriendById()">[申请好友]</a>
              <a href="javascript:;" @click="openPm(playerInfo.user_id)">[发私信]</a>
            </template>
          </div>
          <a href="javascript:;" @click="go(playerInfoBack)">[返回]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
        <div class="panel" v-else>
          <div class="old-line gray">正在加载统帅信息…</div>
        </div>
      </template>

      <!-- ============ 军官/学院(acade) ============ -->
      <template v-else-if="cur === 'acade'">
        <div class="panel-title">
          参谋部({{ officerData.staff_level }}级)
          <a href="javascript:;" @click="switchAcade('search')">去招募</a> |
          <a href="javascript:;" @click="switchAcade('captive')">战俘营</a>
        </div>
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: acadeTab === 'officer' }" @click="switchAcade('officer')">军官</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'scheme' }" @click="switchAcade('scheme')">计谋</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'search' }" @click="switchAcade('search')">招募</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'mayor' }" @click="switchAcade('mayor')">任命市长</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'equip' }" @click="switchAcade('equip')">装备</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'skill' }" @click="switchAcade('skill')">技能</a>|
          <a href="javascript:;" :class="{ on: acadeTab === 'generals' }" @click="switchAcade('generals')">名将图鉴</a>
        </div>

        <!-- 军官列表 -->
        <!-- 军官列表: 复刻 acade/acadeIndex.html -->
        <div class="panel" v-if="acadeTab === 'officer'">
          <div class="old-line">
            军校{{ officerData.academy_level }}级, 参谋部{{ officerData.staff_level }}级
            (容纳{{ officerData.capacity }}名军官), 当前{{ officerData.used }}名
          </div>
          <div class="old-line">
            {{ resNames.gold }}:{{ fmtN(officerData.gold) }}
            <!-- ★ 用户要求「军官是消耗黄金的」：把工资亮出来，玩家知道钱花在哪 -->
            <span class="gray" v-if="officerData.salary">
              （军官工资 {{ fmtN(officerData.salary) }} {{ resNames.gold }}/小时，每级 {{ officerData.salary_per_level }} 金/小时）
            </span>
            <span class="gray" v-else>（暂无军官，不产生工资）</span>
          </div>
          <hr/>
          <template v-for="o in myOfficers">
            <div class="old-line" :key="'of' + o.id">
              {{ o.name }}({{ o.level }}级)
              <a href="javascript:;" @click="openOfficer(o.id)">查看</a><br/>
              状态:{{ o.status_name }}<span v-if="o.position_name !== '无'" class="blue">（{{ o.position_name }}）</span> &nbsp; 评价:{{ o.star }}星<br/>
              后勤/军事/学识/忠诚：<br/>
              {{ o.logistics_total }}/{{ o.military_total }}/{{ o.learning_total }}/{{ o.loyalty }}
              <span class="green" v-if="equipTip(o)">{{ equipTip(o) }}</span><br/>
              攻/防：{{ o.attack }}/{{ o.defence }}<br/>
              <!-- ★ 可用属性点：升过级还没点的军官一眼能看见 -->
              <span v-if="o.free_points > 0" class="red">
                可分配属性点 {{ o.free_points }} 点
                <a href="javascript:;" @click="openOfficer(o.id)">[去加点]</a><br/>
              </span>
              <span v-if="o.active_sets && o.active_sets.length" class="green">
                套装：{{ o.active_sets.join('、') }}<br/>
              </span>
              ------------------------
            </div>
          </template>
          <div class="old-line gray" v-if="!myOfficers.length">(暂无军官, 先去招募吧)</div>
          <div class="old-line">
            前去<a href="javascript:;" @click="switchAcade('captive')">战俘营</a>
            <span class="gray" v-if="captiveOfficers.length">({{ captiveOfficers.length }}名俘虏待收编)</span>
          </div>
        </div>

        <!-- 招募 -->
        <div class="panel" v-else-if="acadeTab === 'search'">
          <div class="old-line">
            军校({{ recruitData.academy_level }}级)：
            <span v-if="recruitData.refresh_left !== undefined">
              本小时刷新:{{ recruitData.refresh_left }}/{{ recruitData.refresh_limit }}次
              <span class="gray">(整点重置)</span>
            </span>
            <a href="javascript:;" @click="doRefreshRecruit">[刷新]</a>
            <!-- ★ 次数用完后，直接在军校使用招生简章（不用先去背包用） -->
            <a href="javascript:;" @click="doUseRecruitTicket">[使用招生简章刷新]</a>
            <span class="gray">(持有 {{ bagCount(13) }} 张)</span>
          </div>
          <div class="old-line">
            军校等级决定每小时候选数量, 参谋部{{ recruitData.staff_level }}级(已用{{ recruitData.used }}/{{ recruitData.capacity }}),
            雇佣费用 = 军官等级 × 1000 {{ resNames.gold }}
          </div>
          <div class="old-line red" v-if="recruitData.academy_level && officerFull">
            参谋部容量已满({{ recruitData.used }}/{{ recruitData.capacity }}), 请先
            <a href="javascript:;" @click="go('buildm')">[升级参谋部]</a>
            或到 <a href="javascript:;" @click="switchAcade('officer')">[军官]</a> 里流放/释放不需要的军官。
          </div>
          <div class="old-line" v-if="!recruitData.academy_level">尚未建造军校, 无法招募军官</div>
          <table v-else class="ezfy-plain-table">
            <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
            <tr v-for="g in recruitData.candidates" :key="'rc' + g.key">
              <td>{{ g.name }}</td>
              <td>{{ g.level }}级</td>
              <td>{{ g.star }}星</td>
              <td>{{ g.logistics }}/{{ g.military }}/{{ g.learning }}</td>
              <td>{{ g.cost }}</td>
              <td>
                <a v-if="!officerFull" href="javascript:;" @click="doRecruit(g)">雇佣</a>
                <span v-else class="gray">(容量已满)</span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="recruitData.academy_level && !recruitData.candidates.length">(本小时候选已全部招募或刷新)</div>
          <div class="old-line">前去<a href="javascript:;" @click="switchAcade('officer')">[军官]</a></div>
        </div>

        <!-- 任命市长: 复刻 acade/setMayor.html -->
        <div class="panel" v-else-if="acadeTab === 'mayor'">
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>等级</th><th>忠诚</th><th>当前职位</th><th>操作</th></tr>
            <tr v-for="o in myOfficers" :key="'my' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.level }}</td>
              <td>{{ o.loyalty }}</td>
              <td>{{ o.position_name }}</td>
              <td>
                <a v-if="o.position !== 1 && o.status === 0" href="javascript:;" @click="doPosition(o, 1)">[任命市长]</a>
                <a v-if="o.position !== 2 && o.status === 0" href="javascript:;" @click="doPosition(o, 2)">[任命城守]</a>
                <a v-if="o.position !== 0" href="javascript:;" @click="doPosition(o, 0)">[卸任]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerData.officers.length">(暂无军官)</div>
        </div>

        <!-- 装备 -->
        <div class="panel" v-else-if="acadeTab === 'equip'">
          <div class="old-line">
            我的装备({{ equipData.bag.length }})
            <a href="javascript:;" @click="switchMallTab('equipment'); go('mall')">[去商城买散件]</a>
            <a href="javascript:;" @click="switchMallTab('chest'); go('mall')">[去开宝箱]</a>
          </div>

          <!-- 装备页子 tab：我的装备 / 我的套装 / 装备图鉴 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: equipTab === 'my' }" @click="equipTab = 'my'">装备</a>|
            <a href="javascript:;" :class="{ on: equipTab === 'set' }" @click="equipTab = 'set'">套装</a>|
            <a href="javascript:;" :class="{ on: equipTab === 'all' }" @click="equipTab = 'all'">装备图鉴</a>
          </div>

          <!-- 我的装备（背包散件 + 检索 + 分页） -->
          <div v-if="equipTab === 'my'">
          <div class="old-line">
            搜索:
            <input v-model="equipWord" type="text" placeholder="装备名 / 部位 / 套装"
                   style="width:180px" @input="equipPage = 1"/>
            <a href="javascript:;" @click="equipWord = ''; equipPage = 1">[清空]</a>
            <span class="gray">共 {{ equipGroups.length }} 种</span>
          </div>
          <table class="ezfy-plain-table">
            <colgroup>
              <col style="width:28%"><col style="width:13%"><col style="width:18%"><col style="width:12%"><col style="width:11%"><col style="width:18%">
            </colgroup>
            <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>要求等级</th><th>状态</th></tr>
            <!-- ★ 2026-09-25 用户建议：「装备[查看]按钮去了也行，同时放到套装里面展开展示也可以」
                 → 采纳：**砍掉「属性」列（原来只放一个 [查看] 按钮）**，改成在该行下面展开详情卡。
                 好处：少一列 → 手机上不再挤；少一次跳页 → 不用来回返回。
                 （原来那个独立的「装备详情页」已经没人能进，一并删掉了。）
                 ★ 用户进一步要求「点名称看该装备的加成，点套装看套装的加成」
                 → 两个入口看**不同**内容，用 detailMode 区分。 -->
            <template v-for="e in equipPaged">
            <tr :key="'eq' + e.key">
              <td class="nm"><a href="javascript:;" @click="toggleDetail(e.id, 'item')">{{ e.name }}</a><span class="gray"> ×{{ e.count }}</span></td>
              <td>{{ e.slot || e.type }}</td>
              <td>
                <a v-if="e.set_id" href="javascript:;" @click="toggleDetail(e.id, 'set')">{{ e.set_name }}</a>
                <span v-else class="gray">—</span>
              </td>
              <td :class="qualityClass(e.tier_name)">{{ e.tier_name }}</td>
              <td>{{ e.level }}</td>
              <td>
                <span v-if="e.worn > 0" class="gray">已穿戴{{ e.worn }}{{ e.worn >= e.count ? ' · 全部' : '' }}</span>
                <a v-if="e.count - e.worn > 0" href="javascript:;" @click="switchAcade('officer')">[去穿戴{{ e.count - e.worn }}]</a>
              </td>
            </tr>
            <tr v-if="detailRowId === e.id" :key="'dt' + e.id" class="set-card-row">
              <td :colspan="6">
                <div class="set-card">
                  <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                  <template v-if="detailMode === 'item'">
                    <div class="sc-h"><b>{{ e.name }}</b>
                      <span :class="qualityClass(e.tier_name)">[{{ e.tier_name || '普通' }}]</span>
                      <span class="gray">{{ e.slot || e.type }} · {{ e.level }}级</span>
                    </div>
                    <div class="sc-b">装备加成：<b class="green">{{ equipAttrText(e) || '（这件没有额外属性加成）' }}</b></div>
                    <div class="sc-b gray" v-if="setOf(e.set_id)">所属套装：{{ setOf(e.set_id).name }}（点套装名看套装加成）</div>
                    <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                  </template>
                  <!-- ② 点「套装名」→ 只看套装加成 -->
                  <template v-else-if="setOf(e.set_id)">
                    <div class="sc-h"><b>{{ setOf(e.set_id).name }}</b>
                      <span :class="qualityClass(setOf(e.set_id).tier_name)">[{{ setOf(e.set_id).tier_name || '特殊' }}]</span>
                      <span class="gray">穿齐 {{ setOf(e.set_id).parts }} 件才生效</span>
                    </div>
                    <div class="sc-b">套装加成：<b class="green">{{ setBonusText(e.set_id) || '（本套装无额外属性加成）' }}</b></div>
                    
                    <div class="sc-b">我的进度：已拥有 <b>{{ setOf(e.set_id).owned || 0 }}</b>/{{ setOf(e.set_id).parts }} 件
                      <span v-if="(setOf(e.set_id).owned || 0) >= setOf(e.set_id).parts" class="green">已够穿齐</span>
                      <span v-else class="red">还差 {{ setOf(e.set_id).parts - (setOf(e.set_id).owned || 0) }} 件</span>
                    </div>
                    <div class="sc-b gray" v-if="setOf(e.set_id).slots && setOf(e.set_id).slots.length">部位：{{ setOf(e.set_id).slots.join(' / ') }}</div>
                    <div class="sc-b gray">点装备名看这件自己的加成</div>
                  </template>
                  <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                </div>
              </td>
            </tr>
            </template>
          </table>
          <div class="old-line gray" v-if="!equipData.bag.length">(背包暂无装备)</div>
          <div class="old-line gray" v-else-if="!equipFiltered.length">(没有匹配「{{ equipWord }}」的装备)</div>
          <!-- ★ 分页 -->
          <div class="ezfy-pager" v-if="equipGroups.length > equipPageSize">
            <a href="javascript:;" :class="{ disabled: equipPage <= 1 }" @click="equipGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(equipPage, equipTotalPages) }}/{{ equipTotalPages }} 页 · 共 {{ equipGroups.length }} 种</span>
            <a href="javascript:;" :class="{ disabled: equipPage >= equipTotalPages }" @click="equipGo(1)">[下一页]</a>
          </div>
          </div>

          <!-- 套装一览：默认只列**我拥有的**（原来这里铺的是「全部套装」= 图鉴，玩家分不清哪个是自己有的），
               可以切到「全部套装」横向对比 —— ★ 2026-09-25 用户要求「方便玩家知晓、对比套装」。 -->
          <div v-if="equipTab === 'set'">
          <div class="old-line set-tab">
            <a href="javascript:;" :class="{ on: !setShowAll }" @click="setShowAll = false">[只看我有的]</a>
            <a href="javascript:;" :class="{ on: setShowAll }" @click="setShowAll = true">[全部套装·可对比]</a>
            <span class="gray">共 {{ setListShown.length }} 套</span>
          </div>
          <!-- ★ 2026-09-25：加成**默认直接铺出来**（原来要点 [加成] 才看得到，用户反馈「不容易看到」）。
               一套一块、竖排 —— 手机上不用横向找，也方便上下对比。 -->
          <div class="set-block" v-for="s in setListShown" :key="'ms' + s.id">
            <div class="sb-h">
              <b :class="qualityClass(s.tier_name)">{{ s.name }}</b>
              <span class="gray">[{{ s.tier_name || '特殊' }}]</span>
              <b :class="s.active ? 'green' : 'red'">{{ s.have }}/{{ s.parts }}</b> 件
              <span v-if="s.active" class="green">加成已生效</span>
              <span v-else class="red">还差 {{ s.need }} 件才生效</span>
            </div>
            <div class="sb-b">套装加成：<b class="green">{{ equipAttrText(s) || '（本套装无额外属性加成）' }}</b></div>
            <div class="sb-b gray" v-if="s.effect">额外效果：{{ s.effect }}</div>
            <div class="sb-b gray" v-if="s.slots && s.slots.length">部位：{{ s.slots.join(' / ') }}</div>
          </div>
          <div class="old-line gray" v-if="!setListShown.length">(暂无套装装备)</div>
          </div>

          <!-- 装备图鉴（全部装备 + 检索 + 分页） -->
          <div v-if="equipTab === 'all'">
          <div class="old-line">装备图鉴({{ equipData.all.length }})</div>
          <div class="old-line">
            搜索:
            <input v-model="equipAllWord" type="text" placeholder="装备名 / 部位 / 套装"
                   style="width:180px" @input="equipAllPage = 1"/>
            <a href="javascript:;" @click="equipAllWord = ''; equipAllPage = 1">[清空]</a>
            <span class="gray">共 {{ equipAllFiltered.length }} 件</span>
          </div>
          <table class="ezfy-plain-table">
            <colgroup>
              <col style="width:34%"><col style="width:14%"><col style="width:20%"><col style="width:12%"><col style="width:20%">
            </colgroup>
            <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>需求等级</th></tr>
            <template v-for="e in equipAllPaged">
            <tr :key="'ea' + e.id">
              <td class="nm"><a href="javascript:;" @click="toggleDetail(e.id, 'item')">{{ e.name }}</a></td>
              <td>{{ e.slot || e.type }}</td>
              <td>
                <a v-if="e.set_id" href="javascript:;" @click="toggleDetail(e.id, 'set')">{{ e.set_name }}</a>
                <span v-else class="gray">—</span>
              </td>
              <td :class="qualityClass(e.tier_name)">{{ e.tier_name }}</td>
              <td>{{ e.level }}</td>
            </tr>
            <tr v-if="detailRowId === e.id" :key="'dt' + e.id" class="set-card-row">
              <td :colspan="5">
                <div class="set-card">
                  <template v-if="detailMode === 'item'">
                    <div class="sc-h"><b>{{ e.name }}</b>
                      <span :class="qualityClass(e.tier_name)">[{{ e.tier_name || '普通' }}]</span>
                      <span class="gray">{{ e.slot || e.type }} · {{ e.level }}级</span>
                    </div>
                    <div class="sc-b">装备加成：<b class="green">{{ equipAttrText(e) || '（这件没有额外属性加成）' }}</b></div>
                    <div class="sc-b gray" v-if="setOf(e.set_id)">所属套装：{{ setOf(e.set_id).name }}（点套装名看套装加成）</div>
                    <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                  </template>
                  <template v-else-if="setOf(e.set_id)">
                    <div class="sc-h"><b>{{ setOf(e.set_id).name }}</b>
                      <span :class="qualityClass(setOf(e.set_id).tier_name)">[{{ setOf(e.set_id).tier_name || '特殊' }}]</span>
                      <span class="gray">穿齐 {{ setOf(e.set_id).parts }} 件才生效</span>
                    </div>
                    <div class="sc-b">套装加成：<b class="green">{{ setBonusText(e.set_id) || '（本套装无额外属性加成）' }}</b></div>
                    
                    <div class="sc-b">我的进度：已拥有 <b>{{ setOf(e.set_id).owned || 0 }}</b>/{{ setOf(e.set_id).parts }} 件
                      <span v-if="(setOf(e.set_id).owned || 0) >= setOf(e.set_id).parts" class="green">已够穿齐</span>
                      <span v-else class="red">还差 {{ setOf(e.set_id).parts - (setOf(e.set_id).owned || 0) }} 件</span>
                    </div>
                    <div class="sc-b gray" v-if="setOf(e.set_id).slots && setOf(e.set_id).slots.length">部位：{{ setOf(e.set_id).slots.join(' / ') }}</div>
                    <div class="sc-b gray">点装备名看这件自己的加成</div>
                  </template>
                  <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                </div>
              </td>
            </tr>
            </template>
          </table>
          <div class="old-line gray" v-if="!equipAllFiltered.length">(没有匹配「{{ equipAllWord }}」的装备)</div>
          <!-- ★ 分页 -->
          <div class="ezfy-pager" v-if="equipAllFiltered.length > equipAllPageSize">
            <a href="javascript:;" :class="{ disabled: equipAllPage <= 1 }" @click="equipAllGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(equipAllPage, equipAllTotalPages) }}/{{ equipAllTotalPages }} 页 · 共 {{ equipAllFiltered.length }} 件</span>
            <a href="javascript:;" :class="{ disabled: equipAllPage >= equipAllTotalPages }" @click="equipAllGo(1)">[下一页]</a>
          </div>
          </div>
        </div>

        <!-- 技能: 复刻 acade/skill.html 的编号列表(带完整说明) -->
        <div class="panel" v-else-if="acadeTab === 'skill'">
          <div class="old-line">军官技能:</div>
          <div class="old-line" v-for="(sk, i) in skillData.skills" :key="'sk' + sk.id">
            {{ i + 1 }}、{{ sk.name }}:{{ sk.des || sk.effect }}<br/>
            <span class="gray">效果：{{ sk.effect }}</span>
            <br/>--------------------
          </div>
          <hr/>
          <div class="old-line">我的军官:</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>已学技能</th><th>操作</th></tr>
            <tr v-for="o in skillData.officers" :key="'sko' + o.id">
              <td>{{ o.name }}</td>
              <td>
                <span v-if="o.skills.length">{{ o.skills.join('、') }}</span>
                <span v-else class="gray">无({{ o.skill_count }}/3)</span>
              </td>
              <td><a href="javascript:;" @click="openOfficer(o.id)">[学习/遗忘]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!skillData.officers.length">(暂无军官)</div>
        </div>

        <!-- 计谋(复刻原版 acade/scheme.html: 12 条计谋, 发动消耗信号弹)
             ★ 2026-09-22：计谋配置改由后端下发（管理端可维护消耗数量/上下架），
               页面显示持有的信号弹数量，够了才能发动。 -->
        <div class="panel" v-else-if="acadeTab === 'scheme'">
          <div class="old-line">说明：计谋需要进入相应界面才可以使用；神兵天降/战略转移请到「军情→军队动态」对部队使用</div>
          <div class="old-line">
            持有「{{ schemeData.bullet_name }}」：
            <b :class="schemeData.bullet_have > 0 ? 'green' : 'red'">{{ schemeData.bullet_have }}</b> 个
            <a href="javascript:;" @click="switchMallTab('item'); go('mall')">[去商城购买]</a>
          </div>
          <div class="old-line" v-for="(s, i) in schemeData.schemes" :key="'sc' + s.id">
            {{ i + 1 }}.{{ s.name }}：<br/>
            {{ s.des }}<br/>
            需要{{ schemeData.bullet_name }}：{{ s.bullet }}
            <span class="gray">（持有 {{ schemeData.bullet_have }}）</span>
            <!-- 先发制人：要选目标城市坐标 -->
            <template v-if="s.kind === 1">
              <span class="gray"> 目标坐标:</span>
              <input v-model="schemeX" type="text" placeholder="x" style="width:56px"/>
              <input v-model="schemeY" type="text" placeholder="y" style="width:56px"/>
            </template>
            <!-- ★ 2026-09-30 行军计谋（神兵天降/战略转移）：作用于部队，不在军校页发动，
                 引导玩家去「军情 → 军队动态」对出征中/返回中的部队使用 -->
            <span v-if="s.kind === 2 || s.kind === 3" class="green">
              [去「军情→军队动态」对出征中/返回中的部队使用]
            </span>
            <template v-else>
              <a v-if="s.enough" href="javascript:;" @click="doScheme(s)">[发动]</a>
              <span v-else class="red">[{{ schemeData.bullet_name }}不足]</span>
            </template>
            <br/>--------------------
          </div>
          <div class="old-line gray" v-if="!schemeData.schemes.length">(暂无计谋，等管理员在后台配置)</div>
          <div class="old-line">
            <a href="javascript:;" @click="go('bag')">[背包(信号弹)]</a>
            <a href="javascript:;" @click="go('back')">[返回]</a> <a href="javascript:;" @click="go('home')">[返回首页]</a>
          </div>
        </div>

        <!-- 战俘营(复刻原版 acade/conquer.html) -->
        <div class="panel" v-else-if="acadeTab === 'captive'">
          <div class="old-line">
            参谋部({{ officerData.staff_level }}级)
            <a href="javascript:;" @click="switchAcade('search')">去招募</a> |
            战俘营
          </div>
          <table class="ezfy-plain-table">
            <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
            <tr v-for="o in captiveOfficers" :key="'cp' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.level }}级</td>
              <td>{{ o.star }}星</td>
              <td>{{ o.logistics }}/{{ o.military }}/{{ o.learning }}</td>
              <td>免费</td>
              <td>
                <a href="javascript:;" @click="doCaptive(o, 'recruit')">[雇佣]</a>
                <a href="javascript:;" @click="doCaptive(o, 'free')">[释放]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!captiveOfficers.length">(战俘营暂无俘虏)</div>
          <div class="old-line gray">
            战俘来源:<br/>
            ① 攻打玩家城市, 把对方军官<b>忠诚打成 0</b> → 弃城归降, 收入我方战俘营;<br/>
            ② 野地/寇城<b>配置里有军官</b>时, 征服胜利有概率俘获守将。<br/>
            正常军官请到 <a href="javascript:;" @click="switchAcade('search')">[军校招募]</a>。
          </div>
          <div class="old-line">前去<a href="javascript:;" @click="switchAcade('officer')">[军官]</a></div>
        </div>

        <!-- 名将图鉴 -->
        <div class="panel" v-else-if="acadeTab === 'generals'">
          <div class="old-line">名将图鉴(共{{ generalData.generals.length }}名, 按等级排序)</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>等级</th><th>星级</th><th>军/后/学</th><th>状态</th></tr>
            <tr v-for="g in generalData.generals" :key="'gg' + g.id">
              <td>{{ g.name }}</td>
              <td>{{ g.level }}</td>
              <td>{{ g.star }}</td>
              <td>{{ g.military }}/{{ g.logistics }}/{{ g.learning }}</td>
              <td>
                <span v-if="g.owned" class="green">已拥有</span>
                <span v-else class="gray">未拥有</span>
              </td>
            </tr>
          </table>
        </div>
      </template>

      <!-- ============ 军官详情(officerdetail) ============ -->
      <template v-else-if="cur === 'officerdetail'">
        <!-- ★ 军官详情：按 tab 分「属性 / 技能 / 装备」三块（同 acade 页 .acade-tab 写法） -->
        <div class="panel" v-if="officerDetail.officer">
          <div class="panel-title">
            {{ officerDetail.officer.name }}
            <a href="javascript:;" @click="doOfficerRename">[改名]</a>
          </div>

          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: officerDetailTab === 'attr' }" @click="officerDetailTab = 'attr'">属性</a>|
            <a href="javascript:;" :class="{ on: officerDetailTab === 'skill' }" @click="officerDetailTab = 'skill'">技能</a>|
            <a href="javascript:;" :class="{ on: officerDetailTab === 'equip' }" @click="officerDetailTab = 'equip'">装备</a>|
            <a href="javascript:;" :class="{ on: officerDetailTab === 'bag' }" @click="officerDetailTab = 'bag'">装备背包</a>
          </div>

          <!-- 属性 tab -->
          <div v-if="officerDetailTab === 'attr'">
          <div class="old-line">
            星级：<b>{{ officerDetail.officer.star }}</b><span class="gray" v-if="officerDetail.officer.star_max">/{{ officerDetail.officer.star_max }}</span>
            &nbsp;等级：<b>{{ officerDetail.officer.level }}</b>
            &nbsp;经验：<span class="gray">{{ officerDetail.officer.level >= detailOfficerMaxLevel ? '—' : (officerDetail.officer.exp + '/' + officerDetail.officer.exp_need) }}</span>
            &nbsp;忠诚：<b>{{ officerDetail.officer.loyalty }}</b>
            <br/>
            职位：<b>{{ officerDetail.officer.position_name }}</b>
            &nbsp;状态：<b :class="officerDetail.officer.status_name === '出征中' ? 'red' : ''">{{ officerDetail.officer.status_name }}</b>
          </div>
          <hr/>
          <div class="old-line">
            军事：{{ officerDetail.officer.military_total }}<span class="green" v-if="officerDetail.officer.equip_military">(+{{ officerDetail.officer.equip_military }})</span>
            &nbsp;后勤：{{ officerDetail.officer.logistics_total }}<span class="green" v-if="officerDetail.officer.equip_logistics">(+{{ officerDetail.officer.equip_logistics }})</span>
            &nbsp;学识：{{ officerDetail.officer.learning_total }}<span class="green" v-if="officerDetail.officer.equip_learning">(+{{ officerDetail.officer.equip_learning }})</span>
            <br/>
            攻击加成：{{ officerDetail.officer.attack }}
            &nbsp;防御加成：{{ officerDetail.officer.defence }}
          </div>

          <!-- 套装与战斗加成（套装穿齐才生效） -->
          <template v-if="(officerDetail.officer.set_progress && officerDetail.officer.set_progress.length) ||
                           officerBattleText(officerDetail.officer.battle)">
            <hr/>
            <div class="old-line" v-if="officerBattleText(officerDetail.officer.battle)">
              装备战斗加成：<span class="green">{{ officerBattleText(officerDetail.officer.battle) }}</span>
            </div>
            <div class="old-line" v-for="sp in officerDetail.officer.set_progress" :key="'sp' + sp.set_id">
              套装「{{ sp.name }}」：{{ sp.worn }}/{{ sp.parts }} 件
              <span :class="sp.active ? 'green' : 'gray'">{{ sp.active ? '已生效' : ('还差 ' + sp.need + ' 件') }}</span>
              <span v-if="sp.active && equipAttrText(sp)" class="green">（{{ equipAttrText(sp) }}）</span>
            </div>
          </template>

          <!-- 属性加点（每升 1 级得 1 点） -->
          <div class="old-line">
            可用属性点
            <b :class="officerDetail.officer.free_points > 0 ? 'red' : 'gray'">{{ officerDetail.officer.free_points }}</b>
            <span class="gray">（已分配 {{ officerDetail.officer.used_points }}）</span>
            <span class="gray" v-if="officerDetail.officer.star_points > 0">（其中升星加点 {{ officerDetail.officer.star_points }}）</span>
          </div>
          <div class="old-line" v-if="officerDetail.officer.free_points > 0">
            分配：
            军事<a href="javascript:;" @click="doAddAttr('military', 1)">[+1]</a><a href="javascript:;" @click="doAddAttr('military', 10)">[+10]</a><a href="javascript:;" @click="doAddAttrAll('military')">[全加]</a>
            &nbsp;后勤<a href="javascript:;" @click="doAddAttr('logistics', 1)">[+1]</a><a href="javascript:;" @click="doAddAttr('logistics', 10)">[+10]</a><a href="javascript:;" @click="doAddAttrAll('logistics')">[全加]</a>
            &nbsp;学识<a href="javascript:;" @click="doAddAttr('learning', 1)">[+1]</a><a href="javascript:;" @click="doAddAttr('learning', 10)">[+10]</a><a href="javascript:;" @click="doAddAttrAll('learning')">[全加]</a>
          </div>

          <!-- 操作 -->
          <div class="old-line officer-actions">
            <a href="javascript:;" @click="doGrant">[赏赐+10忠诚(1万金)]</a>
            <a href="javascript:;" @click="doTreasureGrant()">[赏赐宝物]</a>
            <a href="javascript:;" @click="doRespec">[洗点]</a>
            <span v-if="bagCount(16) > 0" class="gray">(持有军官洗点卡 {{ bagCount(16) }} 张)</span>
            <a v-if="officerDetail.officer.status !== 1 && officerDetail.officer.position === 0"
               href="javascript:;" @click="doExile">[流放]</a>
            <a v-if="officerDetail.officer.star_up_on &&
                     officerDetail.officer.star < officerDetail.officer.star_max"
               href="javascript:;" @click="doStarUp">[升星]</a>
            <span v-if="officerDetail.officer.status === 1" class="gray">(出征中, 归来后才能流放)</span>
            <span v-else-if="officerDetail.officer.position !== 0" class="gray">(市长/城守, 卸任后才能流放)</span>
            <span v-if="officerDetail.officer.star_up_on && officerDetail.officer.star < officerDetail.officer.star_max"
                  class="gray">星级徽章 {{ officerDetail.officer.star_card }} 枚</span>
          </div>

          <!-- 赏赐宝物：展开可选宝物列表（只列背包未穿戴的，按品质 +10/+20/+35/+50 忠诚） -->
          <div v-if="officerTreasureOpen" class="old-line">
            <div class="gray">选择要赏赐的宝物（消耗该件宝物, 忠诚按品质提升, 最高 +50）:</div>
            <table class="ezfy-plain-table">
              <colgroup><col style="width:40%"><col style="width:25%"><col style="width:20%"><col style="width:15%"></colgroup>
              <tr><th class="nm">宝物</th><th>品质</th><th>忠诚</th><th>操作</th></tr>
              <template v-for="e in officerTreasures">
                <tr :key="'tg' + e.id">
                  <td class="nm">{{ e.name }}</td>
                  <td :class="qualityClass(e.tier_name)">{{ e.tier_name || '普通' }}</td>
                  <td class="green">+{{ treasureLoyaltyGain(e.tier) }}</td>
                  <td><a href="javascript:;" @click="doTreasureGrant(e)">[赏赐]</a></td>
                </tr>
              </template>
              <tr v-if="!officerTreasures.length"><td colspan="4" class="gray">(背包没有未穿戴的采集宝物, 可去野地采集或宝物签到获取)</td></tr>
            </table>
          </div>
          </div>

          <!-- 技能 tab（已学 / 可学） -->
          <div v-if="officerDetailTab === 'skill'">
          <table class="ezfy-plain-table">
            <colgroup><col style="width:22%"><col style="width:63%"><col style="width:15%"></colgroup>
            <tr><th colspan="3">已学技能（{{ officerDetail.skills.length }}/3）</th></tr>
            <tr v-for="s in officerDetail.skills" :key="'ds' + s.name">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doForget(s.name)">[遗忘]</a></td>
            </tr>
            <tr v-if="!officerDetail.skills.length"><td colspan="3" class="gray">(未学任何技能)</td></tr>
          </table>
          <table class="ezfy-plain-table">
            <colgroup><col style="width:22%"><col style="width:63%"><col style="width:15%"></colgroup>
            <tr><th colspan="3">可学技能（技能书 {{ officerDetail.officer.skill_book }} 本 / 学一个消耗1本）</th></tr>
            <tr v-for="s in officerDetail.all_skills" :key="'ls' + s.id">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doLearn(s)">[学习]</a></td>
            </tr>
          </table>
          </div>

          <!-- 装备 tab（已穿戴 + 一键卸下 + 一键穿套装） -->
          <div v-if="officerDetailTab === 'equip'">
          <table class="ezfy-plain-table">
            <colgroup>
              <col style="width:30%"><col style="width:14%"><col style="width:12%"><col style="width:22%"><col style="width:22%">
            </colgroup>
            <tr><th colspan="5">已穿戴装备
              <a v-if="officerDetail.equipped.length" href="javascript:;" @click="doUnequipAll">[一键卸下]</a>
            </th></tr>
            <tr><th class="nm">名称</th><th>部位</th><th>品质</th><th>套装</th><th>操作</th></tr>
            <!-- ★ 2026-09-29：已穿戴装备同 cfg 叠加成一行（数量 >1 显示 ×N）——
                 点装备名看这件加成 / 点套装名看套装加成（两个入口看不同内容）。 -->
            <template v-for="g in officerEquipGroups">
            <tr :key="'de' + g.key">
              <td class="nm"><a href="javascript:;" @click="toggleDetail(g.first.id, 'item')">{{ g.name }}</a><span v-if="g.count > 1" class="gray"> ×{{ g.count }}</span></td>
              <td>{{ g.slot }}</td>
              <td :class="qualityClass(g.tier_name)">{{ g.tier_name || '—' }}</td>
              <td>
                <a v-if="g.set_id" href="javascript:;" @click="toggleDetail(g.first.id, 'set')">{{ g.set_name || ('套装' + g.set_id) }}</a>
                <span v-else class="gray">—</span>
              </td>
              <td><a href="javascript:;" @click="doUnequip(g.first.id)">[卸下]</a></td>
            </tr>
            <tr v-if="detailRowId === g.first.id" :key="'dt' + g.key" class="set-card-row">
              <td :colspan="5">
                <div class="set-card">
                  <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                  <template v-if="detailMode === 'item'">
                    <div class="sc-h"><b>{{ g.first.name }}</b>
                      <span :class="qualityClass(g.first.tier_name)">[{{ g.first.tier_name || '普通' }}]</span>
                      <span class="gray">{{ g.first.slot || g.first.type }} · 已穿戴</span>
                    </div>
                    <div class="sc-b">装备加成：<b class="green">{{ equipAttrText(g.first) || '（这件没有额外属性加成）' }}</b></div>
                    <div class="sc-b gray" v-if="setOf(g.first.set_id)">所属套装：{{ setOf(g.first.set_id).name }}（点套装名看套装加成）</div>
                    <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                  </template>
                  <!-- ② 点「套装名」→ 只看套装加成 -->
                  <template v-else-if="setOf(g.first.set_id)">
                    <div class="sc-h"><b>{{ setOf(g.first.set_id).name }}</b>
                      <span :class="qualityClass(setOf(g.first.set_id).tier_name)">[{{ setOf(g.first.set_id).tier_name || '特殊' }}]</span>
                      <span class="gray">穿齐 {{ setOf(g.first.set_id).parts }} 件才生效</span>
                    </div>
                    <div class="sc-b">套装加成：<b class="green">{{ setBonusText(g.first.set_id) || '（本套装无额外属性加成）' }}</b></div>
                    
                    <div class="sc-b">我的进度：已拥有 <b>{{ setOf(g.first.set_id).owned || 0 }}</b>/{{ setOf(g.first.set_id).parts }} 件
                      <span v-if="(setOf(g.first.set_id).owned || 0) >= setOf(g.first.set_id).parts" class="green">已够穿齐</span>
                      <span v-else class="red">还差 {{ setOf(g.first.set_id).parts - (setOf(g.first.set_id).owned || 0) }} 件</span>
                    </div>
                    <div class="sc-b gray" v-if="setOf(g.first.set_id).slots && setOf(g.first.set_id).slots.length">部位：{{ setOf(g.first.set_id).slots.join(' / ') }}</div>
                    <div class="sc-b gray">点装备名看这件自己的加成</div>
                  </template>
                  <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                </div>
              </td>
            </tr>
            </template>
            <tr v-if="!officerDetail.equipped.length"><td colspan="5" class="gray">(未穿戴装备)</td></tr>
          </table>

          <!-- 一键穿戴套装（背包里有件的套装） -->
          <table class="ezfy-plain-table" v-if="officerDetail.bag_sets && officerDetail.bag_sets.length">
            <colgroup>
              <col style="width:36%"><col style="width:32%"><col style="width:12%"><col style="width:20%">
            </colgroup>
            <tr><th colspan="4">一键穿戴套装（同部位已穿戴的会自动卸下让位）</th></tr>
            <tr><th class="nm">套装</th><th>穿齐进度</th><th>等级</th><th>操作</th></tr>
            <tr v-for="s in officerDetail.bag_sets" :key="'bs' + s.set_id">
              <td class="nm">{{ s.name }}</td>
              <td>
                <span :class="s.need > 0 ? 'gray' : 'green'">
                  已穿 {{ s.worn }}/{{ s.parts }} 件{{ s.need > 0 ? (' · 还差 ' + s.need + ' 件生效') : ' · 已生效' }}
                </span>
                <span class="gray" v-if="s.need > 0 && s.bag_count > 0">（背包还有 {{ s.bag_count }} 件）</span>
                <!-- ★ 2026-09-25：一键穿戴这里也把套装加成写出来，玩家才知道穿齐能拿到什么 -->
                <div class="set-mini" v-if="setBonusText(s.set_id)">
                  套装加成：<b class="green">{{ setBonusText(s.set_id) }}</b>
                </div>
              </td>
              <td>{{ s.level }}</td>
              <td><a href="javascript:;" @click="doEquipSet(s)">[一键穿戴]</a></td>
            </tr>
          </table>
          </div>

          <!-- 装备背包 tab（检索 + 分页） -->
          <div v-if="officerDetailTab === 'bag'">
          <table class="ezfy-plain-table">
            <colgroup>
              <col style="width:30%"><col style="width:12%"><col style="width:18%"><col style="width:11%"><col style="width:11%"><col style="width:18%">
            </colgroup>
            <tr><th colspan="6">装备背包</th></tr>
            <!-- ★ 2026-09-29：同一件装备（同 cfg）叠加成一行「名称 ×N」；部位有空余才能 [穿戴]，
                 没空余（同部位已穿戴）显示「部位已满」不可穿戴。 -->
            <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>要求等级</th><th>操作</th></tr>
            <template v-for="g in officerBagPaged">
            <tr :key="'db' + g.key">
              <td class="nm"><a href="javascript:;" @click="toggleDetail(g.first.id, 'item')">{{ g.name }}</a><span class="gray"> ×{{ g.count }}</span></td>
              <td>{{ g.slot }}</td>
              <td>
                <a v-if="g.set_id" href="javascript:;" @click="toggleDetail(g.first.id, 'set')">{{ g.set_name }}</a>
                <span v-else class="gray">—</span>
              </td>
              <td :class="qualityClass(g.tier_name)">{{ g.tier_name }}</td>
              <td>{{ g.first.level }}</td>
              <td>
                <a v-if="g.canEquip" href="javascript:;" @click="doEquipGroup(g)">[穿戴]</a>
                <span v-else class="gray">部位已满</span>
              </td>
            </tr>
            <tr v-if="detailRowId === g.first.id" :key="'dtb' + g.key" class="set-card-row">
              <td :colspan="6">
                <div class="set-card">
                  <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                  <template v-if="detailMode === 'item'">
                    <div class="sc-h"><b>{{ g.first.name }}</b>
                      <span :class="qualityClass(g.first.tier_name)">[{{ g.first.tier_name || '普通' }}]</span>
                      <span class="gray">{{ g.first.slot || g.first.type }} · {{ g.first.level }}级 · 背包 {{ g.count }} 件</span>
                    </div>
                    <div class="sc-b">装备加成：<b class="green">{{ equipAttrText(g.first) || '（这件没有额外属性加成）' }}</b></div>
                    <div class="sc-b gray" v-if="setOf(g.first.set_id)">所属套装：{{ setOf(g.first.set_id).name }}（点套装名看套装加成）</div>
                    <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                  </template>
                  <!-- ② 点「套装名」→ 只看套装加成 -->
                  <template v-else-if="setOf(g.first.set_id)">
                    <div class="sc-h"><b>{{ setOf(g.first.set_id).name }}</b>
                      <span :class="qualityClass(setOf(g.first.set_id).tier_name)">[{{ setOf(g.first.set_id).tier_name || '特殊' }}]</span>
                      <span class="gray">穿齐 {{ setOf(g.first.set_id).parts }} 件才生效</span>
                    </div>
                    <div class="sc-b">套装加成：<b class="green">{{ setBonusText(g.first.set_id) || '（本套装无额外属性加成）' }}</b></div>
                    
                    <div class="sc-b">我的进度：已拥有 <b>{{ setOf(g.first.set_id).owned || 0 }}</b>/{{ setOf(g.first.set_id).parts }} 件
                      <span v-if="(setOf(g.first.set_id).owned || 0) >= setOf(g.first.set_id).parts" class="green">已够穿齐</span>
                      <span v-else class="red">还差 {{ setOf(g.first.set_id).parts - (setOf(g.first.set_id).owned || 0) }} 件</span>
                    </div>
                    <div class="sc-b gray" v-if="setOf(g.first.set_id).slots && setOf(g.first.set_id).slots.length">部位：{{ setOf(g.first.set_id).slots.join(' / ') }}</div>
                    <div class="sc-b gray">点装备名看这件自己的加成</div>
                  </template>
                  <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                </div>
              </td>
            </tr>
            </template>
            <tr v-if="!officerBagGroups.length"><td colspan="6" class="gray">{{ officerBagWord ? '(没有匹配「' + officerBagWord + '」的装备)' : '(背包暂无装备)' }}</td></tr>
          </table>
          <div class="old-line">
            搜索:
            <input v-model="officerBagWord" type="text" placeholder="装备名 / 部位 / 套装"
                   style="width:180px" @input="officerBagPage = 1"/>
            <a href="javascript:;" @click="officerBagWord = ''; officerBagPage = 1">[清空]</a>
            <span class="gray">共 {{ officerBagGroups.length }} 种</span>
          </div>
          <div class="ezfy-pager" v-if="officerBagGroups.length > officerBagPageSize">
            <a href="javascript:;" :class="{ disabled: officerBagPage <= 1 }" @click="officerBagGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(officerBagPage, officerBagTotalPages) }}/{{ officerBagTotalPages }} 页 · 共 {{ officerBagGroups.length }} 种</span>
            <a href="javascript:;" :class="{ disabled: officerBagPage >= officerBagTotalPages }" @click="officerBagGo(1)">[下一页]</a>
          </div>
          </div>

          <div class="old-line"><a href="javascript:;" @click="go('acade')">[返回军官]</a></div>
        </div>
        <!-- ★ 2026-10-01 修复「点击军官有时候空白」：加载失败 / 武将不在当前城时
             不再静默留空，展示原因并引导返回军官列表（重新拉取后该行会消失/恢复） -->
        <div class="panel" v-else>
          <div class="panel-title">军官详情</div>
          <div class="old-line red">{{ officerDetailError || '加载中...' }}</div>
          <div class="old-line"><a href="javascript:;" @click="go('acade')">[返回军官列表]</a></div>
        </div>
      </template>

      <!-- ★ 2026-09-25：独立的「装备详情页」已删除。
           它原来只展示 名字/状态/属性 三行，且唯一的入口就是各装备表里的 [查看] 按钮；
           现在那三样都并进了「点装备名展开的详情卡」，页面没人能进 → 删掉，免得留死代码。
           （如果以后又要一个独立页，从 git 历史里捞回来即可。） -->

      <!-- 底部导航(每页都有, 复刻原版 cityHome.html 的 8+7 两行) -->
      <br/>
      <div class="old-line ezfy-bottom-nav">
        <a href="javascript:;" :class="{ on: cur === 'buildm' }" @click="go('buildm')">军事</a>
        <a href="javascript:;" :class="{ on: cur === 'builds' }" @click="go('builds')">资源</a>
        <a href="javascript:;" :class="{ on: cur === 'map' }" @click="go('map')">地图</a>
        <a href="javascript:;" :class="{ on: cur === 'corps' }" @click="go('corps')">军团</a>
        <a href="javascript:;" :class="{ on: cur === 'rank' }" @click="go('rank')">排行</a>
        <a href="javascript:;" :class="{ on: cur === 'bag' }" @click="go('bag')">背包</a>
        <a href="javascript:;" :class="{ on: cur === 'mall' }" @click="go('mall')">商城</a>
        <a href="javascript:;" :class="{ on: cur === 'treasure' }" @click="go('treasure')">宝物</a>
      </div>
      <div class="old-line ezfy-bottom-nav">
        <a href="javascript:;" :class="{ on: cur === 'activity' }" @click="go('activity')">活动</a>
        <a href="javascript:;" :class="{ on: cur === 'welfare' }" @click="go('welfare')">福利</a>
        <a href="javascript:;" :class="{ on: cur === 'notices' }" @click="go('notices')">公告</a>
        <a href="javascript:;" :class="{ on: cur === 'exchange' }" @click="go('exchange')">交易</a>
        <a href="javascript:;" :class="{ on: cur === 'liaison' }" @click="go('liaison')">联络</a>
        <a href="javascript:;" :class="{ on: cur === 'cityhall' }" @click="go('cityhall')">市政</a>
        <a href="javascript:;" :class="{ on: cur === 'chat' }" @click="go('chat')">聊天</a>
        <!-- ★ 2026-09-24 用户要求: 底部导航「首页」换成「家园」(全局页脚已对沉浸式页面隐藏, 这里作为离开游戏的出口) -->
        <a href="javascript:;" @click="exitToHome()">家园</a>
      </div>
      <hr/>
      <div>小Q报时：{{ nowText }}</div>
      <div>联系我们：QQ群 431442049</div>
    </div>
  </div>
</template>

<script>
import api from '../api'

// ★ 资源显示名的「兜底默认值」。真正的名字由后端 /games/ezfy/res-cfg 下发
//   （管理端「资源管理 → 资源名称维护」可改），改名后全站展示跟随。
const RES_NAMES = { gold: '黄金', food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿' }
const RES_SHORT = { gold: '金', food: '粮', steel: '钢', oil: '油', rare: '稀' }

// 资源说明(黄金一段复刻原版 resourceA.html, 其余按同样语气补全)
// ★ 用 {key} 占位符, 由 applyResNames 把管理端配的资源名填进去 —— 改名后说明也跟着变
const RES_DES_TPL = {
  gold: '{gold}是二战世界里重要的交易货币，可以用来购买玩家出售的各种资源，也用于支付雇佣军官的薪资和招募费用，通过战争、税收、任务可获得。',
  food: '{food}是维持军队的根本，人口增长与部队训练都离不开它，军队每小时还会消耗{food}，一旦断粮部队将大量逃散，通过农田产出、掠夺、任务可获得。',
  steel: '{steel}是制造武器装备的基础材料，建造建筑、训练陆军部队、研究军事科技都需要消耗{steel}，通过炼钢厂产出、掠夺、任务可获得。',
  oil: '{oil}驱动着一切机械化部队，出征行军需要消耗{oil}，训练装甲与航空部队同样需要{oil}，通过石油基地产出、掠夺、任务可获得。',
  rare: '{rare}是尖端军事工业的原料，用于生产重型装备与高级兵种，产量稀少因此格外珍贵，通过稀矿厂产出、掠夺、任务可获得。'
}
function buildResDes (names) {
  const out = {}
  Object.keys(RES_DES_TPL).forEach(k => {
    out[k] = RES_DES_TPL[k].replace(/\{(\w+)\}/g, (m, key) => names[key] || m)
  })
  return out
}

export default {
  name: 'Ezfy',
  data () {
    return {
      cur: 'home',
      prevCur: 'home', // ★ 2026-09-29 上一页（各页 [返回] goBack 用）
      nowText: '', // ★ 页脚小Q报时(每秒刷新, 与 App.vue 同一格式)
      // ★ 2026-09-28 用户要求「累计采集/采集资源实时变化」：每秒本地 tick 的时间基准
      gatherNow: 0,
      // ★ 2026-09-28 「军队动态/出征队列」倒计时自动刷新：拉取 dynamics 时的本地时间戳，
      //   战斗中部队的本回合剩余是「相对剩余」，用它当基点往前推算。
      _dynAt: 0,
      // ★ 2026-09-28 「倒计时归零 → 自动重拉」的三个运行态标记（详见 checkDueRefresh）：
      //   _dynRefreshing  : 本轮重拉是否还在进行（防同波多次触发）
      //   _dynRefreshedAt : 上次重拉的本地时刻（3 秒节流）
      //   _dynFired       : 已触发过的「订单id@到点时刻」，防后端结算失败时每 3 秒无限重拉
      //     ★ 用 Object.create(null) 而不是 {} —— 它只是个去重集合，不需要响应式，
      //       用 {} 会让 Vue 递归侦听每个动态加的 key，纯属浪费。
      _dynRefreshing: false,
      _dynRefreshedAt: 0,
      _dynFired: Object.create(null),
      resNames: RES_NAMES,
      // ★ 2026-09-28 用户要求：头部资源栏「/」右侧展示每小时产量（与资源详情页同口径）
      resProd: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      // ★ 2026-09-28 安抚参数（后端下发，管理端可配：5 万黄金 / 民怨-2 / 民心+1 / 15 分钟冷却）
      placate: { gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 },
      // ★ 安抚冷却的每秒时间基准（由 tickClock 驱动，供 placateCdLeft 计算属性用）
      placateNow: 0,
      _placateAt: 0, // 安抚冷却快照(/view 里的 cd_left)的取回时刻
      resShort: RES_SHORT,
      resDes: buildResDes(RES_NAMES),
      profile: { prestige: 0, camp: 1, nickname: '' },
      userBrief: { account: '', level: 0, exp: 0 },
      officerCount: 0,
      activities: [],
      rankName: '列兵',
      rankPost: '士兵',
      city: {},
      cities: [],
      continent: '',
      // ★ 当前城市类型（/view 顶层下发）：city 对象里没有这两个字段，必须单独存
      cityKindRaw: '',
      cityIsSea: false,
      protectedUntil: false,
      boostUntil: false,
      buildings: [],
      buildingPool: [],
      // ★ 军事区/资源区各自上限（/view 下发，默认各 36）
      militaryCap: 36,
      resourceCap: 36,
      // ★ 2026-09-26 民居容量限制 / 召集人口灵活配置（/view 下发，默认都开）
      housePopLimitOn: true,
      conveneFlexibleOn: true,
      // ★ 2026-09-26 召集消耗粮食 / 召集获得人口（/view 下发，默认各 10 万；原来写死）
      conveneFoodCost: 100000,
      convenePopGain: 100000,
      // ★ 2026-09-26 全局硬性人口上限（/view 下发，0 = 不限），超过禁止召集
      convenePopMax: 0,
      troopsData: { troops: [], queues: [], wounded: [], cfgs: [], pop: 0, pop_used: 0, wall_level: 0, train_discount: 0 },
      // ★ 占用人口（只有训练队列里没出厂的新兵占）：/view 与 /troops 都会下发，谁后到用谁
      popUsed: 0,
      // 与 popUsed 同一次响应里的「人口」，保证「空闲 = 人口 - 占用」恒成立
      // （分开取 city.pop / troopsData.pop 会出现 1000-996=5 这种对不上的显示）
      cityPop: 0,
      techsData: { techs: [], academy: 0 },
      // ★ 2026-09-28 科技详情页选中的科技（列表点[研究]进入 techpre 时赋值）
      techSel: null,
      wildlands: [],
      occupies: [],
      queues: [],
      marching: 0,
      occupying: 0,
      unreadReports: 0,
      reports: [],
      // ★ 军情分区分页：默认每页 5 条
      dynPage: 1, dynSize: 5,
      dynStationPage: 1, dynStationSize: 5,
      repPage: 1, repSize: 5,
      notices: [],
      // ★ 公告分页（用户要求「公告也变成分页，下一页上一页那种」）：默认每页 5 条
      noticePage: 1, noticeSize: 5,
      // ★ 首页外露公告（条数由管理端「建筑上限配置」里的「首页公告条数」决定，默认 1）
      homeNotices: [],
      curNotice: null,
      worldChats: [],
      homeChats: [],
      chatPlayers: 0, chatCorpsPlayers: 0,
      chatMsg: '',
      chatChannel: 1,
      chatHasCorps: false,
      chatCorpsName: '',
      chatCanSend: true,
      chatCooldown: 0,
      // ★ 聊天分页（后端按时间降序返回，最新的在第 1 页最上面）
      chatPage: 1,
      chatTotal: 0,
      chatSize: 15,
      mails: [],
      pmTo: '',
      pmContent: '',
      pmCandidates: [],
      // ★ 私聊：当前会话对象 + 与该对象的聊天记录 + 会话列表
      pmPeer: null,
      pmChat: [],
      pmConvs: [],
      friends: [],
      friendKeyword: '',
      friendSearchList: [],
      friendSearchDone: false,
      friendApplies: { inbox: [], outbox: [] },
      taskGroups: [],
      // ★ 任务分类 tab(0=默认第一个分类: 新手/日常/每周); -1=为爱发电卡。
      //   2026-09-27 fix: 刷新后记住上次所在 tab（localStorage 恢复）
      taskTab: this.restoreTaskTab(),
      // ★ 2026-09-27 为爱发电卡（管理端未发放时为空数组，对应 tab 不显示；多卡可叠加）
      loveCards: [],
      // ★ 计谋（配置由后端下发，发动消耗「信号弹」）
      schemeData: { schemes: [], bullet_name: '信号弹', bullet_have: 0, bullet_item_id: 24 },
      // ★ 2026-09-30 行军计谋页：当前选中的部队（军队动态/出征队列的 [计谋] 带入）
      schemeOrder: null,
      schemeX: '', schemeY: '',
      welfare: { rewards: [], gifts: {} },
      // ★ 2026-09-28 福利页 tab: 0每日签到 / 1礼包 / 2宝物签到
      welfareTab: 0,
      rankData: { prestige: [], troops: [], corps: [], ranks: [] },
      // ★ 排行页 tab: ranks军衔晋升表 / prestige军衔声望榜 / troops兵力榜 / corps军团榜
      rankTab: 'ranks',
      // ★ 军衔晋升表中「宝物」点击展开的行下标（-1 = 收起）
      showTreasureRow: -1,
      orders: [],
      buildZone: 'm',
      buildSel: null,
      curReport: null,
      reportTab: 1,
      reportWord: '',
      reportCounts: {},
      // ★ 自己城市的雷达站等级（决定「来袭/被侦查」预警能不能收到）
      reportRadar: 0,
      // ★ 2026-09-25：情报等级 = 雷达站 + 侦察技巧（后端算好下发），前端据此提示还差多少
      reportRecon: 0, reportIntel: 0,
      dynamics: [],
      // ★ 战场指挥室（军情 → 军队动态 → [指挥]）：每回合 30 秒，前 25 秒可下指令
      battleData: {
        order_id: 0, target_name: '', target_x: 0, target_y: 0, target_type: 0,
        round: 0, max_round: 40, status: 1, win: 0, atk_cmd: '', atk_cmds: {},
        phase: 'cmd', round_left_ms: 0, round_ms: 30000, cmd_window_ms: 25000,
        attackers: [], defenders: [], atk_total: 0, def_total: 0,
        is_atk: true, pvp: false, can_auto: true,
        head: [], actions: [], done: false
      },
      battleOrderId: 0,     // 正在指挥的出征订单 id
      battleLeftMs: 0,      // 本地倒计时（毫秒，每秒自减；归零时拉服务端推进回合）
      battleTimer: null,
      curOrder: null,
      showDetail: true,
      corpsList: [],
      myCorps: null,
      corpsMembers: [],
      corpsChats: [],
      corpsName: '',
      corpsMsg: '',
      kickUserId: 0,
      // ★ 2026-09-25 用户要求：军团页拆成「军团信息 / 军团外交 / 军团宣战 / 军团商城」四个子栏，
      //   纯前端 tab 切换（照抄 rank/acade 页的 .acade-tab 写法），数据按需懒加载。
      // ★ 2026-09-30 用户要求：再加「军团列表 / 军团聊天」独立 tab → 六个。
      corpsTab: 'info',        // info军团信息 / list军团列表 / chat军团聊天 / diplomacy军团外交 / war军团宣战 / mall军团商城
      // ★ 2026-09-30 入团审核：军团的待审申请 / 审核开关 / 我是否已提交申请
      corpsApplies: [],
      myCorpsNeedReview: 0,
      myApplyStatus: 0,        // 0无申请 1待审 2通过 3拒绝
      corpsPoints: 0,          // 军团总积分（/corps/members 或 /corps/relations 的 my_corps.points）
      corpsRelations: null,    // /corps/relations 全量数据（null=还没拉过）
      corpsWars: null,         // /corps/war 全量数据（null=还没拉过）
      // 军团商城：loaded 标记避免每次切 tab 重复拉（兑换/宣战等操作后才会重新拉）
      corpsMall: { loaded: false, my_points: 0, corps_points: 0, items: [], my_bought: {} },
      mallItems: [],
      // ★ 第九轮：商城分类 + 分页 + 钻石余额
      mallCatsList: [], mallCat: '', mallPage: 1, mallPageSize: 10, mallDiamond: 0,
      // ★ 商城分栏：item=道具（原有） equipment=装备套装（用黄金/钻石买）
      mallTab: 'item',
      // ★ 单次购买数量上限（管理端「建筑上限配置」页维护，默认 99）
      mallBuyMax: 99,
      bagItems: [],
      bagTreasures: [],
      // ★ 背包 / 装备列表的检索 + 分页（背包里道具/装备都可能有几十上百条）
      bagWord: '', bagPage: 1, bagPageSize: 10, bagCat: '',
      equipWord: '', equipPage: 1, equipPageSize: 10,        // 我的装备
      equipTab: 'my',                                      // 装备页子tab: my我的装备 / set我的套装 / all装备图鉴
      hqTab: 0,                                          // ★ 司令部子tab: 0兵种配置 / 1出征队列 / 2伤兵营 / 3逃兵营 / 4预设编队 (localStorage 记忆)
      // ★ 2026-09-28 预设编队（司令部保存的出征模板：军官+集结令+兵力，不含目标/随军资源/宿营）
      presets: [],                                       // 预设列表 [{id,name,officer,gather,troops,troop_total}]
      presetSel: 0,                                      // 出征页「预设编队」下拉选中 id (0=不使用)
      presetName: '',                                    // 新增预设的名称输入
      presetAdding: false,                               // 预设编队 tab 是否正在「新增预设」表单模式
      presetMax: 10,                                     // 单账号预设数量上限（与后端 ezfyPresetMax 一致）
      // ★ 2026-09-25：套装一览是否显示「全部套装」（含还没拥有的）—— 方便玩家横向对比
      setShowAll: false,
      // ★ 2026-09-25 用户反馈「套装的加成玩家看不到、不知道买完套装给军官用哪个」：
      //   全部套装配置（含加成/部位/我拥有几件）单独拉一次并缓存，装备页·商城页·军官页共用。
      //   detailRowId = 装备表里正展开详情卡的那一**行**（存装备 id，不是 set_id！
      //   存 set_id 的话同一套的每一行都会各自展开一张一样的卡 —— 实测 10 件套会蹦出 10 张）。
      //   detailMode = 展开的是哪一份内容：'item' 这件自己的加成 / 'set' 套装加成
      //   （用户要求「点名称看该装备的加成，点套装看套装的加成」）。
      allSets: [],
      detailRowId: 0,
      detailMode: '',
      equipAllWord: '', equipAllPage: 1, equipAllPageSize: 10, // 装备图鉴
      officerBagWord: '', officerBagPage: 1, officerBagPageSize: 10, // 军官详情里的背包装备
      officerDetailTab: 'attr', // 军官详情页签: attr属性 / skill技能 / equip装备 / bag装备背包
      officerTreasureOpen: false, // ★ 2026-09-28 赏赐宝物：展开的未穿戴宝物列表
      // ★ 2026-09-25：equipDetail / equipDetailBack 已随「装备详情页」一起删除
      bagOfficers: [],
      bagSkills: [],
      useItem: null,
      useCount: 1,
      useOfficerId: 0,
      useSkillId: 0,
      inlineTip: null, // 建筑区操作的内联提示 { bid, text, type }
      buyItem: null,
      buyCount: 1,
      buyPayWith: 'gold', // ★ 双渠道道具的支付方式选择（gold / diamond）
      exchangeOrders: [],
      exchangeMine: [],
      exchangeGold: 0,
      // ★ 交易行双分页：卖家挂单(exchangePage/exchangeSize/exchangeTotal)、我的挂单(exchangeMPage/...)
      exchangePage: 1, exchangeSize: 10, exchangeTotal: 0,
      exchangeMPage: 1, exchangeMSize: 10, exchangeMTotal: 0,
      exFilter: 0, // ★ 2026-09-24 卖家挂单资源类别检索(0=全部 1粮食 2钢铁 3石油 4稀矿)
      acadeTab: 'officer',
      officerData: { officers: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0 },
      recruitData: { candidates: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0, refresh_left: 0, refresh_limit: 5 },
      skillData: { skills: [], officers: [], gold: 0 },
      equipData: { bag: [], all: [], sets: [] },
      generalData: { generals: [] },
      officerDetail: { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 },
      // ★ 2026-10-01 修复「点击军官有时候空白」：加载失败/武将不在当前城时，
      //   不再静默留空，页面展示该错误并引导返回军官列表
      officerDetailError: '',
      // ★ 装备商城（套装用黄金/钻石购买）
      equipShop: { slots: [], items: [], gold: 0, diamond: 0 },
      // ★ 商城散件：部位筛选 + 检索 + 分页（用户要求「按部位分组表格 + 检索」）
      shopSlot: '', shopWord: '', shopPage: 1, shopSize: 20,
      // ★ 宝箱奖池：点名字才展开（用户要求「别直接展示」）+ 检索 + 分页
      chestPoolId: 0, chestPoolWord: '', chestPoolPage: 1, chestPoolSize: 20,
      bagDescId: 0,   // 背包里「点 [说明] 展开」的那件道具（0 = 都没展开）
      equipShopBuy: null,     // 正在填写购买数量的商品
      equipShopCount: 1,
      equipShopPay: 'gold',   // gold | diamond
      // ★ 宝箱（用钻石/黄金买，开箱按权重出套装件）
      chestData: { chests: [], gold: 0, diamond: 0 },
      chestOpen: null, chestCount: 1, chestPay: 'diamond', chestResult: [],
      chestOpenDetailIdx: -1,   // 开箱详情页「点奖品名查看具体」：当前展开的奖品下标（-1 = 均收起）
      sellType: '1',
      sellCount: 0,
      sellPrice: 0,
      // ★ 2026-09-30 向系统出售资源：每100单位比例 / 手续费 / 丢量提示
      sellSysType: '1',
      sellSysCount: 0,
      sysSellRatio: { 1: 10, 2: 10, 3: 20, 4: 25 },
      sysSellFee: 10,
      sysSellLostWarn: '',
      goldMax: 0,
      trainSel: null,
      trainCount: 10,
      trainSplit: false,
      trainMode: 'troop', // troop=训练(createTroop) / defence=建造(createDefence)
      troopViewId: 0,     // 兵种详情页当前兵种 id
      troopViewBack: 'troops', // 兵种详情页 [返回] 回到哪一页
      // 页面内消息 + 内联确认（替代 alert/confirm/prompt 弹窗）
      msgs: [], askBox: { show: false, text: '', input: false, placeholder: '', value: '' },
      // ★ 确认条实测高度（渲染后量一次）—— askStyle 靠它把确认条夹在视口内，
      //   估算值只做首次兜底，避免「按钮在最底部时确认条被切掉一半」。
      askH: 0,
      // 统帅页自助（游戏ID/家园号码、改昵称、改阵营）
      selfInfo: {}, renameEditing: false, renameInput: '',
      playerInfo: null,        // 他人统帅信息(复刻 infoOther)
      playerInfoBack: 'chat',  // 他人统帅页 [返回] 回到哪一页
      corpsMailContent: '',    // 军团邮件群发内容
      myCorpsTitle: '',        // ★ 我在军团的职位(副团长/参谋长)，副团长可发军团邮件
      taxInput: 20,
      renameInput: '',
      newCityX: '',
      newCityY: '',
      targetCfg: {},
      resType: 'gold',
      resDetail: null,
      mapCells: [],
      mapCx: 0,
      mapCy: 0,
      mapR: 2, // 视野半径 → 5×5 表格(复刻 map/index.html)
      selCell: null,
      selDetail: null,
      eliteCell: null,
      warText: '',
      warStatus: 0, // 0未宣战 / 1宣战待生效 / 2交战中
      // ★ 管理端「宣战功能」开关（/war/status 下发 war_require）：
      //   false = 不需要宣战，掠夺/征服直接可点。默认 true（开关默认开）。
      warRequire: true,
      // ★ 2026-09-25 用户要求：军团交战期也能掠夺/征服（无需个人宣战）。
      //   atWar = /war/status 下发的 at_war（个人宣战已生效 或 管理端关掉宣战开关 或 军团交战期）；
      //   corpsWar = /war/status 下发的 corps_war（{active, corps_name, text}）。
      atWar: false,
      corpsWar: null,
      orderType: 2,
      orderTroops: {},
      onDutyOfficers: [],
      // 本城军官全量列表（含出征中/俘虏）：出征页用它解释「为什么没有可带队军官」
      cityOfficers: [],
      orderOfficer: '0',
      ware: { level: 0, total: 0, next_total: 0, res: [], ratio_sum: 0 },
      wareRatio: { food: 25, steel: 25, oil: 25, rare: 25 },
      liaison: { level: 0, can_join: false, can_create: false, create_cost: 50000,
        member_per_level: 10, my_corps: null, member_count: 0, member_cap: 0,
        garrison_cap: 0, garrison_used: 0, garrisons: [] },
      trFood: 0,
      trSteel: 0,
      trOil: 0,
      trRare: 0,
      trGold: 0,
      waitH: 0,
      waitM: 0,
      orderCalc: null,
      // ★ 本次出征使用几个集结令（数量由管理端配置决定，不写死）
      orderGather: 0,
      // ★ 集结令配置：随 /view 一起下发（一进页面就是准确值）。
      //   原来只有 /order/preview 才返回 gather_max，前端在没点[计算]前兜底写死 50，
      //   结果管理端配了 999 也只能填 50 —— 用户反馈的 bug。
      gatherCfg: { max: 0, per: 0, have: 0 },
      jumpX: '',
      jumpY: '',
      mapStars: [],
      showStars: false,
      // 城市迁移
      // ★ 第十二轮：迁城区域改成「按大洲」（与地图所属洲一致），且迁城消耗道具
      moveInfo: { areas: [], items: [], gold_cost: 200000, gold: 0, city: {} },
      moveArea: 1,          // 兼容旧字段（= moveContinent）
      moveContinent: 1,     // 迁城计划迁入的洲（默认欧洲）
      moveContinentSea: 1,  // 沿海迁城计划迁入的洲
      moveItems: [],        // 三种迁城道具的持有量/定价
      moveX: '',
      moveY: '',
      moveX2: '',
      moveY2: '',
      // 调整生产(开工率)
      rateFood: 100,
      rateSteel: 100,
      rateOil: 100,
      rateRare: 100,
      orderNames: ['', '侦查', '掠夺', '征服', '采集', '运输', '增援', '驻守采集', '派遣'],
      timer: null
    }
  },
  // ★ 2026-09-27 建筑操作的内联提示自动消失：设置后过几秒清掉，
  //   避免「已取消升级/已开始升级」这类提示一直挂在建筑行上不消失。
  //   [关闭] 仍然可用（直接置 null）。
  watch: {
    inlineTip (n) {
      if (this._tipTimer) { clearTimeout(this._tipTimer); this._tipTimer = null }
      if (n) this._tipTimer = setTimeout(() => { this.inlineTip = null }, 5000)
    },
    // ★ 2026-09-28 随军资源上限跟随所选兵种：兵力变化时防抖重算负重(orderCalc.carry)，
    //   让 resQtyMax 的上限(=负重×装载技术加成)始终与当前部队一致。
    //   没选任何部队(归零)时把随军资源一并清空，配合 orderResDisabled 整块禁用。
    orderTroopTotal (n) {
      if (n <= 0) {
        if (this._resCalcT) { clearTimeout(this._resCalcT); this._resCalcT = null }
        this.trFood = 0; this.trSteel = 0; this.trOil = 0; this.trRare = 0; this.trGold = 0
        return
      }
      if (this._resCalcT) clearTimeout(this._resCalcT)
      this._resCalcT = setTimeout(() => this.doCalc(), 300)
    }
  },
  computed: {
    // ★ 2026-09-28 赏赐宝物：背包里未穿戴的**采集宝物**（可在军官详情操作区展开选择）
    //   ★ 只认采集宝物（后端 bag 条目带 treasure 标记，名字取自 9 种野地珠宝）——
    //     步枪/钢盔/合金装甲这类普通装备不能换忠诚；宝物签到抽的也是同一池，所以签到宝物可用。
    officerTreasures () {
      return ((this.officerDetail && this.officerDetail.bag) || [])
        .filter(e => e.treasure && !e.worn)
    },
    // ★ 任务分类 tab: 当前展示的任务组(新手/日常/每周; taskTab=0 或无效时回落到第一组)
    taskGroupCur () {
      const gs = this.taskGroups || []
      if (!gs.length) return { id: 0, name: '', reset_type: 0, tasks: [] }
      return gs.find(g => g.id === this.taskTab) || gs[0]
    },
    // ★ 2026-09-27 为爱发电卡合计可领天数（多卡叠加），>0 才有「领取」按钮
    loveTotalClaimable () {
      const cs = this.loveCards || []
      let n = 0
      for (const c of cs) n += (c.claimable || 0)
      return n
    },
    // ★ 二级导航（资源/军官/军队/科技/城防/统帅）：只在对应页面显示，位置固定在页面顶部
    //   军队的几个子页（兵种/兵种详情/训练/工厂）也算「军队」，一并显示，保持导航不中断
    isArmyPage () {
      return ['troops', 'troop', 'troopview', 'trainpre', 'factory'].indexOf(this.cur) >= 0
    },
    showSubnav () {
      return ['buildm', 'builds', 'acade', 'officerdetail', 'techs', 'techpre', 'defence', 'info'].indexOf(this.cur) >= 0 || this.isArmyPage
    },
    // ★ 玩家当前军衔等级 id（用于首页/统帅信息展示对应军衔星级图标）
    myRankId () { return this.rankIdByName(this.rankName) },
    // 改名提示：首次免费 / 之后消耗改名卡
    renameHint () {
      const d = this.selfInfo || {}
      if (!d.game_uid && !d.nickname) return '首次改名免费'
      if (d.rename_free) return '首次改名免费'
      return '首次免费已用掉，再次改名需消耗「改名卡」×1（当前持有 ' + (d.rename_card_count || 0) + ' 张）'
    },
    // 阵营提示：首次免费 / 之后消耗阵营转换道具
    campHint () {
      const d = this.selfInfo || {}
      if (!d.game_uid && !d.nickname) return '首次转换阵营免费'
      if (d.camp_free) return '首次转换阵营免费'
      return '首次免费已用掉，再次转换需消耗「阵营转换道具」×1（当前持有 ' + (d.camp_item_count || 0) + ' 个）'
    },
    nick () {
      return this.$store.state.user ? this.$store.state.user.nickname : ''
    },
    // 私聊里区分「我」和「对方」
    myUserId () {
      return this.$store.state.user ? this.$store.state.user.id : 0
    },
    freePop () {
      // ★ 占用人口由后端统一算：只有「训练中、还没出厂」的新兵占人口。
      //   已训练完成的部队（城内驻军 / 出征在外）都不占人口位置（用户 2026-09-21 的规则）。
      //   原来只减 troopsData.pop_used（且只有进过「军队」页才有值）→ 首页空闲人口会显示成满人口。
      const used = this.popUsed || 0
      const pop = this.cityPop || this.city.pop || 0
      const v = pop - used
      return v > 0 ? v : 0
    },
    queueNames () {
      return this.queues
    },
    // ★ 2026-09-26：召集是否被民居容量上限挡住
    //   仅当「民居容量限制」开 且「召集人口灵活配置」关 时，召集才受上限约束。
    //   单次召集 +convenePopGain 人口（管理端可配，默认 10 万），加完超上限就禁用按钮。
    conveneBlocked () {
      // 人口取本页展示的 city.pop（与页面上「当前人口」一致），cityPop 兜底
      const pop = (this.city && this.city.pop) || this.cityPop || 0
      // ★ 2026-09-26 全局硬性人口上限（管理端配置，0 = 不限）：对召集永远生效
      if (this.convenePopMax > 0 && pop + this.convenePopGain > this.convenePopMax) return true
      // ★ 2026-09-26 民居上限：仅当「民居容量限制」开 且「召集人口灵活配置」关 时生效
      //   单次召集 +convenePopGain 人口（管理端可配，默认 10 万），加完超上限就禁用按钮。
      if (!this.housePopLimitOn || this.conveneFlexibleOn) return false
      return pop + this.convenePopGain > ((this.city && this.city.pop_max) || 0)
    },
    // 当前建筑分区: 'm' 军事区 / 's' 资源区
    // 复刻原版 BuildingController: 军事区 = type 2/3/4, 资源区 = type 1
    zone () {
      if (this.cur === 'builds') return 's'
      if (this.cur === 'buildpre') return this.buildZone
      return 'm'
    },
    zoneBuildings () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      const built = this.buildings.filter(b => inZone(b.type))
      const pool = (this.buildingPool || []).filter(p => inZone(p.type))
      return built.concat(pool)
    },
    // 已建建筑(军事区/资源区主页面只列这些)
    zoneBuilt () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      const built = this.buildings.filter(b => inZone(b.type))
      // ★ 军事区：民居(building_id=2)固定排到列表最下面（其余顺序不变）
      if (isM) {
        const res = built.filter(b => b.building_id === 2)
        if (res.length) return built.filter(b => b.building_id !== 2).concat(res)
      }
      return built
    },
    // 可建造的建筑(「建造」按钮进去的那一页)
    zonePool () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      return (this.buildingPool || []).filter(p => inZone(p.type))
    },
    buildQueueCount () {
      return this.buildings.filter(b => b.status !== 0).length
    },
    // 正式军官(不含俘虏)
    myOfficers () {
      return (this.officerData.officers || []).filter(o => o.is_captive !== 1)
    },
    // ★ 军官最高等级（后端下发 max_level，默认 150）—— 达到即「满级」，不再升级
    officerMaxLevel () {
      const v = parseInt(this.officerData.max_level)
      return v > 0 ? v : 150
    },
    // ★ 2026-09-29 军官详情页逐人上限：名将 350 / 普通军官 150（后端 officerDetail.max_level）
    detailOfficerMaxLevel () {
      const o = this.officerDetail && this.officerDetail.officer
      if (!o) return 150
      const v = parseInt(o.max_level)
      return v > 0 ? v : this.officerMaxLevel
    },
    // 战俘营: 未出征的俘虏
    // ★ 2026-09-29 跨城汇总：俘虏可能落在任一座城，战俘营不再只看当前城——
    //   优先用后端下发的跨城 captives，没有(旧后端)再回落到当前城过滤。
    captiveOfficers () {
      if (this.officerData && Array.isArray(this.officerData.captives)) {
        return (this.officerData.captives || []).filter(o => o.status !== 1)
      }
      return (this.officerData.officers || []).filter(o => o.is_captive === 1 && o.status !== 1)
    },
    // 参谋部军官位是否已满(招募前先拦一道, 避免点了才报「容量不足」)
    officerFull () {
      const c = this.recruitData.capacity || this.officerData.capacity || 0
      const u = this.recruitData.used !== undefined ? this.recruitData.used : this.officerData.used
      return c > 0 && (u || 0) >= c
    },
    trainCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type !== 4)
    },
    // ★ 2026-09-28 用户要求：首页点「军队」看到全部兵种（数量为 0 的也显示）+ 训练操作。
    //   城内军队表遍历全部兵种配置（**不含城防兵种 type 4**，城防只在「城防」页展示），
    //   数量从 troops 里取（没有=0），行动行自带 troop_id。
    armyRows () {
      const troops = this.troopsData.troops || []
      return (this.troopsData.cfgs || []).filter(c => c.type !== 4).map(c => {
        const row = troops.find(x => x.troop_id === c.id)
        return Object.assign({}, c, { count: row ? row.count : 0, troop_id: c.id })
      })
    },
    curOfficerBonus () {
      for (const o of this.onDutyOfficers) {
        if (o.name === this.orderOfficer) return o.battle_bonus
      }
      return 0
    },
    wareSum () {
      const r = this.wareRatio
      return (parseInt(r.food) || 0) + (parseInt(r.steel) || 0) + (parseInt(r.oil) || 0) + (parseInt(r.rare) || 0)
    },
    // 集结令数量(背包里查；背包还没加载时用 /view 下发的 gather_have 兜底)
    gatherCount () {
      const it = (this.bagItems || []).find(x => x.name === '集结令')
      const n = it ? it.count : 0
      return n > 0 ? n : (this.gatherCfg.have || 0)
    },
    // ★ 集结令单次上限：以 /view 下发的 gatherCfg.max 为准（读 ezfy_cfg_limit.gather_max_per_order，
    //   管理端「建筑上限配置」页可维护，默认 99）。
    //   ⚠️ 这里**不能**在没有数据时直接返回 99 去夹输入值 —— 那正是「配了 999 只能用 99」的 bug：
    //   进页面时 orderCalc 还是 null，一改数字就被夹回 99，后面再点[计算]也救不回来了。
    orderCapMax () {
      const m = this.gatherCfg.max || (this.orderCalc && this.orderCalc.gather_max)
      return m > 0 ? m : 99
    },
    // 真正可用的上限 = min(管理端上限, 背包实际持有量)
    gatherMax () { return Math.min(this.orderCapMax, this.gatherCount) },
    // ★ 每个集结令提升的出征上限，同样以接口下发为准（读 ezfy_cfg_item.param1，缺省 10 万）
    orderCapPer () {
      const p = this.gatherCfg.per || (this.orderCalc && this.orderCalc.gather_per)
      return p > 0 ? p : 100000
    },
    // ★ 2026-09-25 出征页「兵种数量搭配」改成一行一个兵种 + 滑动条联动：
    //   本地实时合计本次出兵数量（拖滑块/填数字立刻刷新，不用等点 [计算]）。
    //   口径与后端一致：只算可出征兵种（trainCfgs 已滤掉城防 type=4），0 值不计。
    orderTroopTotal () {
      let sum = 0
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k], 10)
        if (n > 0) sum += n
      }
      return sum
    },
    // 是否超出出征上限（口径同后端 troop_over_cap：管理端关掉上限开关时不判超）
    orderOverCap () {
      const c = this.orderCalc
      if (!c || c.cap_unlimited || !this.orderCapApplies) return false
      return this.orderTroopTotal > (c.troop_cap || 0)
    },
    // 上限文案：开关关掉时显示「不限」，没算过时显示 —
    orderCapText () {
      const c = this.orderCalc
      if (!c) return '—'
      // ★ 2026-09-29 运输(5) 无出征上限 → 显示「不限」；派遣(8) 有上限，显示 troop_cap
      if (!this.orderCapApplies) return '不限'
      return c.cap_unlimited ? '不限' : this.fmtN(c.troop_cap)
    },
    // ★ 2026-09-29 修正：派遣(8)是城际调兵、要带部队，和普通出征一样有「出征兵力上限」卡控；
    //   只有运输(5)是运货、无兵力上限（按城内现有）。前端/后端必须同一口径。
    orderCapApplies () {
      return this.orderType !== 5
    },
    // ★ 2026-09-28 随军资源：负重上限 = 所带兵种负重之和 × 装载技术加成。
    //   直接采用 [计算]（orderCalc.carry，后端 ezfyCarryCapOf 已含科技加成）作为唯一口径，
    //   与出征/运输/采集的负重校验一致；选兵种变化时由 orderTroopTotal watcher 重算。
    orderResCap () {
      const c = this.orderCalc
      return c ? (c.carry || 0) : 0
    },
    // ★ 2026-09-29 修正：运输(5)油耗按携带资源量另算、直接从城里扣，**不占部队负重** ——
    //   减油耗会形成回环（填资源→油耗变大→上限变小，但油又不随资源刷新），所以运输用满负重当上限。
    //   其余（含派遣8）油耗按兵种计、需占负重 → 负重 − 油耗。
    orderResUsableCap () {
      const c = this.orderCalc
      if (this.orderType === 5) return Math.max(0, c ? (c.carry || 0) : 0)
      return Math.max(0, this.orderResCap - this.oilUsed)
    },
    // 行军油耗（随军资源占用的负重需要从负重上限里先扣掉）
    oilUsed () {
      const c = this.orderCalc
      return c ? (c.oil_used || 0) : 0
    },
    // 随军资源五项之和（重量 = 占用负重）
    orderResTotal () {
      return ['gold', 'food', 'steel', 'oil', 'rare'].reduce((s, k) => s + this.resQty(k), 0)
    },
    // 没选任何部队 → 随军资源整块禁用（滑块/数字/[最大] 都灰掉，也不能填写）
    orderResDisabled () {
      return this.orderTroopTotal <= 0
    },
    // 是否超出负重上限（红了要玩家减资源或多带部队）
    orderResOver () {
      return !this.orderResDisabled && this.orderResTotal > this.orderResUsableCap
    },
    // 随军资源行列表（沿用兵力行的渲染结构：名称+现有+滑块+数字+[最大]）
    resFields () {
      const r = this.resNames || {}
      return [
        { key: 'gold', name: r.gold },
        { key: 'food', name: r.food },
        { key: 'steel', name: r.steel },
        { key: 'oil', name: r.oil },
        { key: 'rare', name: r.rare }
      ]
    },
    // ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服（要先在「附属野地」页[放弃]）。
    //   判据取后端 WildlandView 下发的 mine —— 它与详情页显示的「归属：我」是同一个来源
    //   （野地记录 → 城市 → UserID），前端不自己猜归属，避免两边口径漂移。
    //   area_type === 3 是玩家城市，走另一套规则（宣战/同盟），这里排除掉。
    isOwnWild () {
      return !!(this.selDetail && this.selDetail.mine &&
        this.selCell && this.selCell.area_type !== 3)
    },
    defenceCfgs () {
      return (this.troopsData.cfgs || []).filter(t => t.type === 4)
    },
    defenceTroops () {
      return (this.troopsData.troops || []).filter(t => t.type === 4)
    },
    // 城防建造队列(复刻 troopDefence.html 的「正在建造」)
    defenceQueues () {
      const ids = this.defenceCfgs.map(t => t.id)
      return (this.queues || []).filter(q => ids.indexOf(q.troop_id) >= 0)
    },
    // 兵种详情页当前兵种
    troopView () {
      return (this.troopsData.cfgs || []).find(t => t.id === this.troopViewId) || null
    },
    // 训练确认页可训练上限: 资源 / 人口(城防为城防空间) 取最小
    maxTrainable () {
      if (!this.trainSel) return 0
      const c = this.trainSel.cost || {}
      const ct = this.troopsData.city || this.city || {}
      const caps = []
      if (c.food > 0) caps.push(Math.floor((ct.food || 0) / c.food))
      if (c.steel > 0) caps.push(Math.floor((ct.steel || 0) / c.steel))
      if (c.oil > 0) caps.push(Math.floor((ct.oil || 0) / c.oil))
      if (c.rare > 0) caps.push(Math.floor((ct.rare || 0) / c.rare))
      if (this.trainMode === 'defence') {
        // 城防设施每个占 1 点城防空间(复刻 GameServiceImpl 的 used += count)
        const space = (this.troopsData.defence_space || 0) - (this.troopsData.defence_space_used || 0)
        caps.push(Math.max(0, space))
      } else if (this.trainSel.pop > 0) {
        caps.push(Math.floor(this.freePop / this.trainSel.pop))
      }
      const m = caps.length ? Math.min.apply(null, caps) : 0
      return Math.max(0, m)
    },
    // 训练确认页预计耗时(复刻 createTroop.html 的「时间」: 单个耗时 × 数量 ÷ 并行工厂数)
    trainEstimateText () {
      if (!this.trainSel) return '0秒'
      const n = parseInt(this.trainCount) || 0
      const par = this.trainMode === 'defence'
        ? 1
        : (this.trainSplit ? Math.max(1, this.factoryFree) : 1)
      return this.durText(Math.ceil(this.trainSel.train_time * n / par))
    },
    attackTroops () {
      return (this.troopsData.troops || []).filter(t => t.type !== 4)
    },
    totalTroops () {
      let sum = 0
      for (const t of this.troopsData.troops) sum += t.count
      return sum
    },
    areaCount () {
      return this.buildings.filter(b => b.type >= 1 && b.type <= 4).length
    },
    areaCap () {
      return 66
    },
    // ★ 当前分区（军事区/资源区）各自的建筑数（分开统计，不再混用 areaCount）
    zoneCount () {
      const isM = this.zone === 'm'
      const inZone = t => (isM ? (t === 2 || t === 3 || t === 4) : t === 1)
      return this.buildings.filter(b => inZone(b.type)).length
    },
    // ★ 当前分区上限：军事区/资源区各 36（与后端 ezfy_cfg_limit 现值一致）
    zoneCap () {
      return this.zone === 'm' ? this.militaryCap : this.resourceCap
    },
    factoryTotal () {
      return this.buildings.filter(b => b.building_id === 14).reduce((s, b) => s + b.level, 0)
    },
    factoryFree () {
      return Math.max(1, this.factoryTotal - this.queues.length)
    },
    isLeader () {
      return !!(this.myCorps && this.profile.user_id && this.myCorps.leader_user_id === this.profile.user_id)
    },
    // ★ 军团邮件权限：军团长 或 副团长
    canMailCorps () {
      return this.isLeader || this.myCorpsTitle === '副团长'
    },
    // ★ 首页外露的公告 = 后端按「首页公告条数」配置下发的 home_notices（默认 1 条）
    topNotices () {
      if (this.homeNotices && this.homeNotices.length) return this.homeNotices
      return (this.notices || []).filter(n => n.is_top).slice(0, 1)
    },
    // ★ 商城：按分类过滤 + 分页（分类为空 = 全部）
    mallFiltered () {
      const cat = this.mallCat
      const list = this.mallItems || []
      return cat ? list.filter(i => i.category === cat) : list
    },
    // ============ ★ 通用「检索 + 分页」小工具（背包 / 装备列表共用） ============
    // 后端一次性把列表下发，检索与分页都在前端做（这些列表不会大到需要服务端分页）。
    // ★ 背包分类（按首次出现顺序，与商城的归类口径一致：后端 ezfyItemCategory）
    bagCats () {
      const out = []
      ;(this.bagItems || []).forEach(i => {
        const c = i.category || '其他'
        if (out.indexOf(c) < 0) out.push(c)
      })
      return out
    },
    bagFiltered () {
      let list = this.bagItems || []
      if (this.bagCat) list = list.filter(i => (i.category || '其他') === this.bagCat)
      const w = (this.bagWord || '').trim().toLowerCase()
      if (!w) return list
      return list.filter(i => String(i.name || '').toLowerCase().includes(w) ||
        String(i.description || '').toLowerCase().includes(w))
    },
    bagTotalPages () { return Math.max(1, Math.ceil(this.bagFiltered.length / this.bagPageSize)) },
    bagPaged () {
      const p = Math.min(Math.max(1, this.bagPage), this.bagTotalPages)
      return this.bagFiltered.slice((p - 1) * this.bagPageSize, p * this.bagPageSize)
    },
    // 我的装备（可按 名称 / 部位 / 套装 / 类型 检索）
    // ★ 2026-09-25：set_id → 套装配置（含加成）。装备表/商城表/军官装备表都从这里查，
    //   这样「没拥有的套装」也能看到穿齐给什么（原来只有 /officers/equipments 的 sets，
    //   且只含我至少有一件的 → 商城里看不到加成，玩家不知道买哪套）。
    setMap () {
      const m = {}
      for (const s of (this.allSets || [])) m[s.id] = s
      return m
    },
    // ★ 2026-09-25 确认条的定位（用户反馈「军官详情的一键穿戴，确认跑到页面最上面，要滚上去才能点」）：
    //   和提示条 msgStyle 同一套思路 —— 浮在**刚才点的那个控件**旁边，而不是永远钉在页面顶部。
    //   与提示条的区别：确认条更高（文案 + 可选输入框 + [确定][取消] 两个按钮），所以：
    //     ① 优先放控件**下方**；下方放不下就翻到**上方**；上下都放不下就夹进视口；
    //     ② 高度用 askH（渲染后实测），首次用估算值兜底，保证按钮在屏幕最底部时也不会被切掉。
    //   ⚠️ _lastClicked 是 mounted 里那个**捕获阶段** click 监听记下的 <a>/<button>；
    //      ask() 都是在点击处理里调的，所以这里拿到的一定是触发它的那个控件。
    //   ⚠️ 必须是 computed（不能放 methods）：模板里是 :style="askStyle"，
    //      方法会被当成函数对象直接绑上去。
    askStyle () {
      const vw = window.innerWidth
      const vh = window.innerHeight
      const w = Math.min(360, vw - 16)
      const H = this.askH || (this.askBox.input ? 150 : 104)
      const el = this._lastClicked
      const rect = el && typeof el.getBoundingClientRect === 'function' ? el.getBoundingClientRect() : null
      const inView = rect && rect.left < vw && rect.right > 0 && rect.top < vh && rect.bottom > 0
      let left, top
      if (rect && inView) {
        left = Math.round(rect.left + rect.width / 2 - w / 2)
        left = Math.max(8, Math.min(left, vw - w - 8))
        const below = rect.bottom + 8
        if (below + H <= vh - 8) top = below
        else if (rect.top - 8 - H >= 8) top = rect.top - 8 - H
        else top = below
      } else {
        left = Math.round((vw - w) / 2)
        top = 8
      }
      top = Math.max(8, Math.min(Math.round(top), vh - H - 8))
      return { position: 'fixed', left: left + 'px', top: top + 'px',
        width: w + 'px', zIndex: 10000 }
    },
    // ★ 2026-09-25 套装一览要显示的内容：默认「我有的」；切到全部时列**所有套装**（含没拥有的），
    //   两种来源统一成同一形状（have/need/active/slots/effect + 加成字段），模板只认一套字段。
    setListShown () {
      if (!this.setShowAll) {
        return this.mySetProgress.map(s => Object.assign({}, s, {
          owned: s.have, have: s.have, need: s.need, active: s.active,
          slots: (this.setOf(s.id) && this.setOf(s.id).slots) || []
        }))
      }
      return (this.allSets || []).map(s => Object.assign({}, s, {
        have: s.owned || 0,
        need: Math.max(0, (s.parts || 0) - (s.owned || 0)),
        active: (s.parts || 0) > 0 && (s.owned || 0) >= s.parts
      }))
    },
    // ★「我的套装」：只列**玩家已拥有**的套装（从背包聚合），并算还差几件才生效。
    //   原来这里铺的是「全部套装」= 图鉴，玩家根本分不清哪个是自己有的。
    mySetProgress () {
      const bag = this.equipData.bag || []
      const have = {}
      bag.forEach(e => { if (e.set_id) have[e.set_id] = (have[e.set_id] || 0) + 1 })
      return (this.equipData.sets || [])
        .filter(s => have[s.id])
        .map(s => Object.assign({}, s, {
          have: have[s.id],
          need: Math.max(0, (s.parts || 0) - have[s.id]),
          active: (s.parts || 0) > 0 && have[s.id] >= s.parts
        }))
    },
    equipFiltered () {
      const w = (this.equipWord || '').trim().toLowerCase()
      const list = (this.equipData && this.equipData.bag) || []
      if (!w) return list
      return list.filter(e => [e.name, e.slot, e.type, e.set_name, e.series, e.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
    },
    // ★ 2026-09-29 用户要求：「我的装备」按同名同件聚合展示（名称 ×N），减少分页数；
    //   状态列判断还有几件可穿戴（未穿戴件数）。细节行沿用 first 的字段，无需改。
    equipGroups () {
      const map = {}, order = []
      for (const e of this.equipFiltered) {
        const k = [e.name, e.level, e.slot, e.type].join('|')
        if (!map[k]) {
          map[k] = Object.assign({}, e, { first: e, count: 0, worn: 0 })
          order.push(map[k])
        }
        map[k].count++
        if (e.worn) map[k].worn++
      }
      return order
    },
    equipTotalPages () { return Math.max(1, Math.ceil(this.equipGroups.length / this.equipPageSize)) },
    equipPaged () {
      const p = Math.min(Math.max(1, this.equipPage), this.equipTotalPages)
      return this.equipGroups.slice((p - 1) * this.equipPageSize, p * this.equipPageSize)
    },
    // 装备图鉴
    equipAllFiltered () {
      const w = (this.equipAllWord || '').trim().toLowerCase()
      const list = (this.equipData && this.equipData.all) || []
      if (!w) return list
      return list.filter(e => [e.name, e.slot, e.type, e.set_name, e.series, e.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
    },
    equipAllTotalPages () { return Math.max(1, Math.ceil(this.equipAllFiltered.length / this.equipAllPageSize)) },
    equipAllPaged () {
      const p = Math.min(Math.max(1, this.equipAllPage), this.equipAllTotalPages)
      return this.equipAllFiltered.slice((p - 1) * this.equipAllPageSize, p * this.equipAllPageSize)
    },
    // ★ 2026-09-29 用户要求：军官详情「装备背包 / 已穿戴装备」同一个装备叠加展示（名称 × N），
    //   有空余部位就能继续穿戴（对应部位没被占用），没空余就不能穿戴。
    //   分组 key = cfg_id（同配置的装备实例 = 同一件装备）；老数据没有 cfg_id 时兜底 name|slot|set_id|tier。
    // 已穿戴装备分组（装备 tab）：同 cfg 的叠加成一行，数量 >1 时显示 ×N
    officerEquipGroups () {
      const bag = (this.officerDetail && this.officerDetail.bag) || []
      const cfgOf = {}
      for (const e of bag) if (e.cfg_id) cfgOf[e.id] = e.cfg_id
      const groups = {}
      const order = []
      for (const e of (this.officerDetail && this.officerDetail.equipped) || []) {
        const key = cfgOf[e.id] ? 'c' + cfgOf[e.id] : this.equipGroupKey(e)
        let g = groups[key]
        if (!g) {
          g = { key: key, name: e.name, slot: e.slot || e.type || '', type: e.type,
                set_id: e.set_id, set_name: e.set_name, tier: e.tier,
                tier_name: e.tier_name, level: e.level, first: e, items: [], count: 0 }
          groups[key] = g
          order.push(key)
        }
        g.items.push(e)
        g.count++
      }
      return order.map(k => groups[k])
    },
    // 当前军官已占用的部位集合（决定背包装备还能不能穿戴）
    officerEquipSlots () {
      const s = {}
      for (const g of this.officerEquipGroups) s[g.slot] = true
      return s
    },
    // 军官详情里的装备背包：按 cfg 叠加（只算未穿戴的），带 canEquip（该部位还有空余就能穿）
    officerBagGroups () {
      const w = (this.officerBagWord || '').trim().toLowerCase()
      const bag = (this.officerDetail && this.officerDetail.bag) || []
      const groups = {}
      const order = []
      const slots = this.officerEquipSlots
      for (const e of bag) {
        if (!e) continue
        if (e.worn) continue
        const key = this.equipGroupKey(e)
        let g = groups[key]
        if (!g) {
          g = { key: key, name: e.name || '', slot: e.slot || e.type || '', type: e.type,
                set_id: e.set_id || 0, set_name: e.set_name || '', series: e.series || '', tier: e.tier || 0,
                tier_name: e.tier_name || '', level: e.level || 0, first: e, items: [], count: 0,
                canEquip: false }
          groups[key] = g
          order.push(key)
        }
        g.items.push(e)
        g.count++
      }
      let list = order.map(k => groups[k])
      // 搜索词作用于分组行
      if (w) list = list.filter(g => [g.name, g.slot, g.type, g.set_name, g.series, g.tier_name]
        .some(v => String(v || '').toLowerCase().includes(w)))
      // 部位有空余 = 军官身上还没穿同部位装备 → 可以穿戴（后端同部位唯一兜底）
      for (const g of list) g.canEquip = !slots[g.slot]
      return list
    },
    officerBagTotalPages () {
      return Math.max(1, Math.ceil(this.officerBagGroups.length / this.officerBagPageSize))
    },
    officerBagPaged () {
      const p = Math.min(Math.max(1, this.officerBagPage), this.officerBagTotalPages)
      return this.officerBagGroups.slice((p - 1) * this.officerBagPageSize, p * this.officerBagPageSize)
    },
    mallTotalPages () {
      return Math.max(1, Math.ceil(this.mallFiltered.length / this.mallPageSize))
    },
    mallPaged () {
      const p = Math.min(Math.max(1, this.mallPage), this.mallTotalPages)
      return this.mallFiltered.slice((p - 1) * this.mallPageSize, p * this.mallPageSize)
    },
    // 某坐标是否海城(海洋地形 8)
    // ★ 海城判据：**以后端返回的 is_sea 为准**。
    //   后端规则：海城 = 建在「沿海平原」(ezfyTerrainEx==9) 上。
    //   旧版前端自己用坐标哈希算「地形==8(海洋)」，真正的海城被判成陆地城市。
    isSeaAt () {
      return ct => {
        if (!ct) return false
        if (typeof ct.is_sea === 'boolean') return ct.is_sea
        return !!ct.is_sea
      }
    },
    // ★ 当前城市类型文案：统一为「沿海城市 / 内陆城市」（后端 city_kind 为准）
    cityKindLabel () {
      if (this.cityKindRaw) return this.cityKindRaw
      return this.cityIsSea ? '沿海城市' : '内陆城市'
    },
    // ★ 资源详情页显示对应科技名（1 种植 / 2 炼钢 / 3 勘探 / 4 冶炼）
    resTechName () {
      const m = { food: '种植技术', steel: '炼钢技术', oil: '勘探技术', rare: '冶炼技术' }
      return m[this.resType] || ''
    },
    // ★ 聊天分页总页数（后端按时间降序返回）
    chatTotalPages () {
      return Math.max(1, Math.ceil(this.chatTotal / this.chatSize))
    },
    // ★ 军队动态 = 行进/战斗/返航中的部队(不含驻守采集); 驻军 = 常驻采集(status=1)
    dynMarch () {
      return this.dynamics.filter(o => o.status !== 1)
    },
    dynStation () {
      return this.dynamics.filter(o => o.status === 1)
    },
    // ★ 2026-09-25 出征队列页的数据 = 军队动态全集（行进/战斗/返航/驻守都在内，一次展示完）
    queueItems () {
      // ★ 2026-09-28 采集实况 _lg + 倒计时实况 _lt 都依赖 gatherNow，每秒一起重算
      return (this.dynamics || []).map(o => this.withLg(this.withLive(o)))
    },
    // ★ 2026-09-28 安抚冷却剩余毫秒（依赖 placateNow 每秒重算，到点自动放行按钮）
    //   后端的 cd_left 是拉 /view 那一刻的快照，这里用本地时间基准往前推，避免只靠刷新。
    placateCdLeft () {
      const total = (this.placate && this.placate.cooldown_min ? this.placate.cooldown_min : 15) * 60000
      const snap = (this.placate && this.placate.cd_left) || 0
      if (snap <= 0) return 0
      // 快照剩余 = snap（拉接口的那一刻）；本地已经流逝的时间 = 从 load 到现在的差
      const elapsed = this.placateNow && this._placateAt ? this.placateNow - this._placateAt : 0
      return Math.max(0, Math.min(total, snap - elapsed))
    },
    // ★ 军情分区分页（默认每页 5 条，可上一页/下一页）
    dynMarchTotalPages () {
      return Math.max(1, Math.ceil(this.dynMarch.length / this.dynSize))
    },
    dynStationTotalPages () {
      return Math.max(1, Math.ceil(this.dynStation.length / this.dynStationSize))
    },
    // ★ 战场指挥室：本回合剩余秒数 / 倒计时条百分比（最后 5 秒条变红）
    battleLeftText () {
      return Math.max(0, Math.ceil(this.battleLeftMs / 1000)) + ' 秒'
    },
    battleBarPct () {
      const total = this.battleData.round_ms || 30000
      const pct = this.battleLeftMs * 100 / total
      return Math.max(0, Math.min(100, pct))
    },
    // ★ 战场指挥室：把行动日志按回合分组，并让**最新回合排最上**（旧回合往下沉）
    battleRounds () {
      const acts = (this.battleData && this.battleData.actions) || []
      const groups = []
      let cur = null
      for (const a of acts) {
        if (/^第\d+回合:/.test(a)) {
          cur = { line: a, items: [] }
          groups.push(cur)
        } else {
          if (!cur) { cur = { line: null, items: [] }; groups.push(cur) }
          cur.items.push(a)
        }
      }
      return groups.slice().reverse()
    },
    // ★ 商城散件：部位筛选 + 检索 + 分页
    shopSlots () {
      return (this.equipShop.slots || []).map(s => s.slot)
    },
    shopAll () {
      let list = []
      ;(this.equipShop.slots || []).forEach(s => { list = list.concat(s.pieces || []) })
      if (this.shopSlot) list = list.filter(p => p.slot === this.shopSlot)
      const w = (this.shopWord || '').trim()
      if (w) list = list.filter(p => (p.name || '').indexOf(w) >= 0 || (p.slot || '').indexOf(w) >= 0)
      return list
    },
    shopTotalPages () {
      return Math.max(1, Math.ceil(this.shopAll.length / this.shopSize))
    },
    shopPaged () {
      const p = Math.min(Math.max(1, this.shopPage), this.shopTotalPages)
      return this.shopAll.slice((p - 1) * this.shopSize, p * this.shopSize)
    },
    // ★ 宝箱奖池（点宝箱名字才展开）
    chestPoolCur () {
      return (this.chestData.chests || []).find(x => x.id === this.chestPoolId) || null
    },
    chestPoolAll () {
      const ch = this.chestPoolCur
      if (!ch) return []
      let list = ch.pool || []
      const w = (this.chestPoolWord || '').trim()
      if (w) list = list.filter(p => (p.name || '').indexOf(w) >= 0)
      return list
    },
    chestPoolTotalPages () {
      return Math.max(1, Math.ceil(this.chestPoolAll.length / this.chestPoolSize))
    },
    chestPoolPaged () {
      const p = Math.min(Math.max(1, this.chestPoolPage), this.chestPoolTotalPages)
      return this.chestPoolAll.slice((p - 1) * this.chestPoolSize, p * this.chestPoolSize)
    },
    dynMarchPaged () {
      const p = Math.min(Math.max(1, this.dynPage), this.dynMarchTotalPages)
      return this.dynMarch.slice((p - 1) * this.dynSize, p * this.dynSize).map(o => this.withLive(o))
    },
    dynStationPaged () {
      const p = Math.min(Math.max(1, this.dynStationPage), this.dynStationTotalPages)
      return this.dynStation.slice((p - 1) * this.dynStationSize, p * this.dynStationSize).map(o => this.withLg(this.withLive(o)))
    },
    repTotalPages () {
      return Math.max(1, Math.ceil(this.reports.length / this.repSize))
    },
    repPaged () {
      const p = Math.min(Math.max(1, this.repPage), this.repTotalPages)
      return this.reports.slice((p - 1) * this.repSize, p * this.repSize)
    },
    // ★ 公告分页（用户要求「公告也变成分页，下一页上一页那种」），与军情同一套写法
    noticeTotalPages () {
      return Math.max(1, Math.ceil(this.notices.length / this.noticeSize))
    },
    // ★ 交易行分页（服务端分页：后端返回 total/page/size）
    exchangeTotalPages () {
      return Math.max(1, Math.ceil(this.exchangeTotal / this.exchangeSize))
    },
    exchangeMTotalPages () {
      return Math.max(1, Math.ceil(this.exchangeMTotal / this.exchangeMSize))
    },
    noticePaged () {
      const p = Math.min(Math.max(1, this.noticePage), this.noticeTotalPages)
      return this.notices.slice((p - 1) * this.noticeSize, p * this.noticeSize)
    },
    // 翻页步长 = 一整屏(复刻原版: 向上 x-5 / 向右 y+5, 即 2r+1)
    mapStep () {
      return this.mapR * 2 + 1
    },
    mapRows () {
      // 边长以**实际返回的格子数**为准(后端返回 n×n), 避免与 mapR 不一致时整表错位
      const n = this.mapCells.length
      const size = n > 0 && Number.isInteger(Math.sqrt(n)) ? Math.round(Math.sqrt(n)) : this.mapR * 2 + 1
      const rows = []
      for (let i = 0; i < size; i++) {
        rows.push(this.mapCells.slice(i * size, (i + 1) * size))
      }
      return rows
    },
    // 三种迁城道具的持有数量，模板里直接用（'low'/'high'/'sea' → 个数）
    moveItemCounts () {
      const out = { low: 0, high: 0, sea: 0 }
      for (const it of (this.moveItems || [])) {
        if (out[it.code] !== undefined) out[it.code] = it.count || 0
      }
      return out
    }
  },
  mounted () {
    // 沉浸式: 去掉 body 默认的 5px 外边距, 标题条才能贴满屏幕上方与左右
    document.body.classList.add('ezfy-immersive')
    // ★ 2026-09-29 浏览器标签页标题：二战征途改为游戏名（不再显示默认的「家园社区」）
    document.title = '二战征途-文字游戏'
    // ★ 2026-09-28 司令部子 tab: 刷新后仍是上次选中的 tab（localStorage）
    this.hqTab = this.restoreHqTab()
    // ★ 2026-09-27 iPhone 字体再修复：旧方案用 `@supports (-webkit-touch-callout: none)`
    //   只在桌面(无头 Chrome)测过为 false，但 iOS Safari 的 @supports 解析器并不认识
    //   -webkit-touch-callout 这个属性 → 条件永远不成立 → iPhone 一直回落到宋体-简
    //   (Songti SC)，细灰发虚。改为 JS 判 iOS 加 body.ezfy-ios，再靠 CSS 切苹方/系统字体。
    //   iPadOS 桌面模式 UA 是 Macintosh + 带触摸，用 onTouchEnd 一并圈进来。
    const ua = navigator.userAgent
    if (/iPhone|iPad|iPod/i.test(ua) || (/Macintosh/.test(ua) && 'ontouchend' in document)) {
      document.body.classList.add('ezfy-ios')
    }
    // 沉浸式卡控①: 游戏内任何 <a href="/..."> 都不允许跳出 /games/ezfy 回到家园站点
    // (捕获阶段拦截, 只拦站内绝对路径链接)
    document.addEventListener('click', this.blockEscape, true)
    // ★ 记录最近点击的**元素**（页面内导航、按钮、格子的鼠标落点）：
    //   操作结果的提示要浮现在「刚才点的那个控件」附近，而不是固定页面顶部或原始鼠标坐标。
    //   存元素引用而非 clientX/Y —— API 响应是异步的，回来时页面可能已滚动，
    //   用元素 getBoundingClientRect() 实时取位置不会飘走。
    this._capClick = (e) => {
      // 注意：Vue 的 @click 编译后不会在 DOM 上留下任何属性，因此不能用
      // 属性选择器去找"绑了点击事件的元素"（'[@click]' 还是非法选择器，
      // 会让 closest 直接抛 SyntaxError）。合法做法：a/button 用 closest 命中；
      // 其余情况退回点击目标元素本身（同样有 getBoundingClientRect 可定位）。
      const t = e.target
      let el = null
      if (t && typeof t.closest === 'function') {
        el = t.closest('a,button')
      }
      this._lastClicked = el || (t && t.tagName ? t : null)
    }
    document.addEventListener('click', this._capClick, true)
    // 沉浸式卡控②: 浏览器后退不退出游戏, 而是回到游戏上一页(与幻想西游 Xiyou.vue 一致)
    history.pushState({ __ezfyGuard: true }, '')
    this._onBack = () => {
      if (location.hash.split('?')[0].indexOf('/games/ezfy') >= 0) {
        this.go('home')
      }
      history.pushState({ __ezfyGuard: true }, '')
    }
    window.addEventListener('popstate', this._onBack)
    this.load()
    this.loadWelfare()
    this.loadResCfg()
    this.loadChats()
    this.loadHomeChats()
    this.loadNotices()
    this.loadCorps()
    // ★ 刷新后回到刷新前所在的页面（用户反馈：每次刷新都跑首页，不对）
    //   页面状态写在 URL 的 ?cur= 上，onload 时读回来重放 go() 的加载逻辑。
    this.restoreFromUrl()
    this.timer = setInterval(() => {
      if (this.cur === 'home') { this.load(); this.loadResCfg(); this.loadHomeChats() }
      if (this.cur === 'chat') this.loadChats()
    }, 30000)
    // ★ 页脚小Q报时: 每秒刷新(与 App.vue 页脚同一格式)
    this.tickClock()
    this.clockTimer = setInterval(this.tickClock, 1000)
  },
  beforeDestroy () {
    document.body.classList.remove('ezfy-immersive')
    document.body.classList.remove('ezfy-ios')
    document.removeEventListener('click', this.blockEscape, true)
    if (this._capClick) document.removeEventListener('click', this._capClick, true)
    if (this._onBack) window.removeEventListener('popstate', this._onBack)
    if (this.timer) clearInterval(this.timer)
    if (this.clockTimer) clearInterval(this.clockTimer)
    if (this._tipTimer) clearTimeout(this._tipTimer)
    this.stopBattleTimer()
  },
  methods: {
    // 装备分组 key：同 cfg 的装备实例视为同一件装备（老数据无 cfg_id 时兜底 name|slot|set_id|tier）
    equipGroupKey (e) {
      if (e && e.cfg_id) return 'c' + e.cfg_id
      return 'n' + (e.name || '') + '|' + (e.slot || e.type || '') + '|' + (e.set_id || 0) + '|' + (e.tier || 0)
    },
    // 页脚小Q报时(与 App.vue tick 同款格式)
    tickClock () {
      // ★ 2026-09-28 让「累计采集/采集资源」实时变化：tickClock 每秒已被 clockTimer 调用，
      //   这里顺手把它升为一个响应式时间基准，模板上的 liveGather() 每秒重算。
      this.gatherNow = Date.now()
      // ★ 安抚冷却倒计时也复用这个每秒基准（避免再起一个定时器）
      this.placateNow = this.gatherNow
      const d = new Date()
      const p = n => (n < 10 ? '0' + n : '' + n)
      this.nowText = d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
      // ★ 2026-09-28 修复「抵达时间归零后卡在 0 秒不动」：
      //   后端是**懒结算**（请求进来才按时间补算），前端那个倒计时只是本地推算，
      //   归零后如果没人去请求接口，状态就永远停在「抵达时间: 0秒」——
      //   看起来像卡死，实际是少了一次「到点了，去拉最新状态」的触发。
      //   这里每秒检查一次：只要**有部队的倒计时归零了**且当前在会走时间的页面上，
      //   就主动重拉一次军队动态（后端顺手把该结算的结算掉）。
      this.checkDueRefresh()
    },
    // ★ 倒计时归零 → 自动重拉军队动态（见 tickClock 里的说明）
    //   为什么不用固定间隔轮询：大部分时间没有到点的部队，固定轮询纯浪费请求；
    //   触发式只在「真的有部队到点」时打一次接口，1 核 1G 的线上更友好。
    //   生效页面：军情(军队动态/驻军/军情警讯/战斗报告)、出征队列、战报详情。
    //   节流：最少间隔 3 秒拉一次，避免多个部队同时到点时连环打接口。
    //   ★ 去重：同一订单到点只补拉一次（记在 _dynFired 里）——
    //     否则后端万一结算失败、仍返回 status=0 + 过去的 arrive_time，
    //     就会每 3 秒无限打接口（1 核 1G 上这属于事故级浪费）。
    checkDueRefresh () {
      const onDynPage = this.cur === 'reports' || this.cur === 'reportview' || this.cur === 'orders'
      if (!onDynPage || !this.dynamics || !this.dynamics.length) return
      const now = Date.now()
      // 上一轮刷新还没结束 / 距上次刷新不足 3 秒 → 跳过
      if (this._dynRefreshing) return
      if (this._dynRefreshedAt && now - this._dynRefreshedAt < 3000) return
      if (!this._dynFired) this._dynFired = Object.create(null)
      // 长时间挂机会往 _dynFired 里累积 key（每条订单每一轮倒计时一个），
      // 超过 200 个就整体清空 —— 已完成的订单不会再出现在 dynamics 里，
      // 清掉不会导致「重复触发」，只是给还在跑的订单重新计一次数。
      const firedKeys = Object.keys(this._dynFired)
      if (firedKeys.length > 200) this._dynFired = Object.create(null)
      // 只认「还在走」的倒计时：出征(0)看 arrive_time，返航(2)看 return_time。
      // 战斗中(5)的回合由 battleTimer 自己驱动，不在这里管。
      // 用「订单id+到点时刻」当 key —— 这样同一订单后续新一轮倒计时（重新出征/返航）
      // 能再次触发，而同一轮到点只触发一次。
      let needRes = false
      const due = this.dynamics.some(o => {
        if (!o) return false
        let at = 0
        if (o.status === 0 && o.arrive_time > 0) at = o.arrive_time
        else if (o.status === 2 && o.return_time > 0) at = o.return_time
        if (!at || at > now) return false
        const key = o.id + '@' + at
        if (this._dynFired[key]) return false
        this._dynFired[key] = 1
        // 返航到达 = 待带回资源入库 → 资源栏要刷新；
        // 出征抵达本身不改资源（只是状态变成驻守/进入战斗），不必刷。
        if (o.status === 2) needRes = true
        return true
      })
      if (!due) return
      this._dynRefreshing = true
      this._dynRefreshedAt = now
      this.loadDynamics()
      // ★ 只有「返航到达」这种真的会改动城市资源的情况才补一次资源刷新。
      //   注意这里**不调整个 load()** —— load() 里会连带 loadRank()，太重；
      //   资源栏只需要 /view 的增量，走下面这个轻量分支。
      if (needRes) this.refreshRes()
      // loadDynamics 是 promise 链，没有返回值可 await；用一个短定时器放开闸门，
      // 保证同一波到点只触发一次（下一个 tick 不会重复打）。
      setTimeout(() => { this._dynRefreshing = false }, 1500)
    },
    // ★ 轻量刷新头部资源栏（不触发 loadRank 等重查询）——返航物资入库后调用
    refreshRes () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) {
          this.profile = r.data.profile
          this.city = r.data.city
          this.resProd = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, r.data.res_prod || {})
          this.placate = Object.assign({ gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 }, r.data.placate || {})
          this._placateAt = Date.now() // 记住安抚冷却快照的取回时刻，供 placateCdLeft 本地推算
        }
      })
    },
    // ★ 2026-09-28 安抚改为「固定花费 + 15 分钟冷却」后，旧口径不再适用：
    //   原来文案写死「民怨×100」，现在花费/效果/冷却都以后端下发为准（见 View 的 placate 块）。
    //   安抚后立刻重拉 /view 刷新冷却倒计时，避免连点。
    async doPlacate () {
      if (!await this.ask('确定安抚民心吗？（花费 ' + this.placate.gold + ' ' + this.resNames.gold +
        '，民怨 -' + this.placate.grievance + '、民心 +' + this.placate.feelings +
        '；每 ' + this.placate.cooldown_min + ' 分钟可安抚一次）')) return
      api.post('/games/ezfy/city/placate', {}).then(r => {
        this.alert(r, '安抚完成', () => { this.load() })
      })
    },
    // 退出游戏回家园 —— 游戏内唯一的合法出口(底部导航最后的「家园」, 原「首页」)。
    // 用 @click 而不是 <a href>, 这样不会被下面的 blockEscape 拦掉。
    exitToHome () {
      this.$router.push('/home')
    },
    // 沉浸式卡控①: 拦掉一切会把玩家带出游戏的站内链接
    // 兼容 '/home' 与 '/#/home' 两种写法; 不动浏览器的前进/后退(那由下面的 popstate 守卫接管)
    blockEscape (e) {
      const a = e.target && e.target.closest ? e.target.closest('a[href]') : null
      if (!a) return
      const href = a.getAttribute('href') || ''
      if (!href || href.charAt(0) !== '/') return          // 相对路径 / javascript: / 外链: 不管
      const raw = href.charAt(1) === '#' ? href.slice(1) : href
      const path = (raw.charAt(0) === '#' ? raw.slice(1) : raw).split('?')[0]
      if (path.indexOf('/games/ezfy') === 0) return         // 游戏内: 放行
      e.preventDefault()
      e.stopPropagation()
      this.notify('游戏内不能跳回家园。如需离开游戏, 请点底部导航「家园」。')
    },
    notOpen (what) {
      this.notify(what + '暂未开放, 敬请期待')
    },
    // ============ 页面状态与 URL 同步（刷新后停在原页面） ============
    //
    // 背景：游戏是单页应用（Ezfy.vue 靠 cur 切换 60+ 个分支），
    //   以前 cur 只存在内存里，一刷新就回落到 data 里的默认值 'home' ——
    //   用户反馈「刷新前在哪个页面刷新后还是哪个页面」。
    //
    // 做法：把 cur（以及资源页的 resType）挂到 URL 的查询串上，
    //   刷新/收藏/分享都能回到同一页。
    //
    // ★ 必须用 history.replaceState 而不是改 location.hash：
    //   本组件在 mounted 里注册了 popstate 守卫（后退回首页），
    //   如果用 pushState / location 赋值，会污染历史栈、把守卫带乱。
    //   replaceState 只改当前条目的 URL，不产生新历史，最安全。

    // syncUrl 把当前页面写进 URL（replaceState，不产生历史记录）
    syncUrl () {
      try {
        const hash = location.hash || ''
        const qi = hash.indexOf('?')
        const base = qi >= 0 ? hash.slice(0, qi) : hash
        const params = new URLSearchParams(qi >= 0 ? hash.slice(qi + 1) : '')
        params.set('cur', this.cur)
        if (this.cur === 'res' && this.resType) params.set('res', this.resType)
        else params.delete('res')
        // ★ 2026-09-25：命令详情页的订单 id 也写进 URL，刷新后才能恢复（原来只在内存里）
        if (this.cur === 'orderview' && this.curOrder && this.curOrder.id) {
          params.set('oid', this.curOrder.id)
        } else params.delete('oid')
        // ★ 2026-09-27 他人统帅信息要查看的玩家 user_id 也写进 URL（同 orderview），
        //   否则刷新后 playerInfo 拿不到 pid 会一直卡「正在加载统帅信息…」。
        if (this.cur === 'playerinfo' && this.playerInfo && this.playerInfo.user_id) {
          params.set('pid', this.playerInfo.user_id)
        } else params.delete('pid')
        // ★ 2026-09-28 修复「在战斗报告页刷新后跳回军队动态」：
        //   reportTab 原来只在内存里，刷新就回落到 data 默认值 1。
        //   reports(列表页) / reportview(战报详情页) 两个页面都把分区 id 挂到 URL 上。
        if ((this.cur === 'reports' || this.cur === 'reportview') && this.reportTab) {
          params.set('rtab', this.reportTab)
        } else params.delete('rtab')
        const qs = params.toString()
        const next = base + (qs ? '?' + qs : '')
        history.replaceState(history.state, '', location.pathname + location.search + next)
      } catch (e) { /* URL 同步失败不影响游戏本身 */ }
    },

    // restoreFromUrl 启动时从 URL 读回页面并重放加载逻辑
    restoreFromUrl () {
      let cur = ''
      let res = ''
      let oid = ''
      let pid = ''
      let rtab = 0
      try {
        const hash = location.hash || ''
        const qi = hash.indexOf('?')
        if (qi >= 0) {
          const params = new URLSearchParams(hash.slice(qi + 1))
          cur = params.get('cur') || ''
          res = params.get('res') || ''
          oid = params.get('oid') || ''
          pid = params.get('pid') || ''
          rtab = parseInt(params.get('rtab') || '0', 10) || 0
        }
      } catch (e) {}
      if (res) this.resType = res
      // ★ 2026-09-28 军情页的分区(军队动态/驻军/军情警讯/战斗报告)也随 URL 恢复，
      //   必须在 go('reports') **之前**赋值：go() 里就是 `switchReportTab(this.reportTab)`。
      if (rtab >= 1 && rtab <= 5) this.reportTab = rtab
      // ★ 2026-09-25 修复「命令详情页刷新后消失」：curOrder 不在 URL 里，
      //   刷新时必须按 oid 重新拉一次详情；拉不到就退回出征队列（绝不留空白页）。
      if (cur === 'orderview') {
        if (parseInt(oid, 10) > 0) this.loadOrderView(oid)
        else this.go('orders')
        return
      }
      // ★ 2026-09-27 修复「他人统帅信息刷新后丢失」：按 pid 重新拉一次；
      //   拿不到 pid 就退回首页（同 orderview，绝不留空白/卡加载页）。
      if (cur === 'playerinfo') {
        const u = parseInt(pid, 10)
        if (u > 0) {
          this.playerInfo = { user_id: u } // 先占位，syncUrl 才会保留 pid
          this.go('playerinfo')
          this.loadPlayerInfo(u)
        } else {
          this.go('home')
        }
        return
      }
      // ★ 2026-09-30 计谋页刷新兜底：schemeOrder 只在内存里，刷新后丢了就回军队动态
      //   （绝不能停在空白的计谋页；信号弹持有量可以重新拉）。
      if (cur === 'scheme' && !this.schemeOrder) {
        this.go('reports')
        return
      }
      // 没写 cur、或就是 home：保持默认首页即可（go('home') 会重复拉一遍数据）
      if (!cur || cur === 'home') return
      // ★ 复用 go()：所有页面分支的加载逻辑都在它里面，
      //   在这里重写一遍必然漏，直接重放最省事也最不容易出错。
      this.go(cur)
    },
    // ---- 统帅页自助 ----
    loadSelfInfo () {
      return api.get('/games/ezfy/profile/self').then(r => {
        if (r.code === 0) {
          this.selfInfo = r.data
          this.renameInput = r.data.nickname || ''
        }
      }).catch(() => {})
    },
    startRename () {
      this.renameInput = this.selfInfo.nickname || this.profile.nickname || ''
      this.renameEditing = true
    },
    async doPlayerRename () {
      const name = String(this.renameInput || '').trim()
      if (name.length < 2 || name.length > 12) { this.notify('昵称长度需在 2~12 个字符之间'); return }
      const free = this.selfInfo.rename_free
      const tip = free ? '确认改名为「' + name + '」吗？（首次免费）'
        : '确认改名为「' + name + '」吗？将消耗「改名卡」×1（当前 ' +
          (this.selfInfo.rename_card_count || 0) + ' 张）'
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/profile/rename', { nickname: name }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.renameEditing = false
          this.loadSelfInfo()
          this.load()
        } else this.notify(r.msg)
      })
    },
    async doChangeCamp (camp) {
      const cur = this.selfInfo.camp || this.profile.camp
      if (cur === camp) { this.notify('当前已经是「' + (camp === 2 ? '轴心国' : '同盟国') + '」'); return }
      const free = this.selfInfo.camp_free
      const tip = free ? '确认转换为「' + (camp === 2 ? '轴心国' : '同盟国') + '」吗？（首次免费）'
        : '确认转换为「' + (camp === 2 ? '轴心国' : '同盟国') + '」吗？将消耗「阵营转换道具」×1（当前 ' +
          (this.selfInfo.camp_item_count || 0) + ' 个）'
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/profile/camp', { camp: camp }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadSelfInfo()
          this.load()
        } else this.notify(r.msg)
      })
    },
    go (t) {
      // ★ 2026-09-29 各页 [返回]：回到上一页；无有效上一页则回首页
      if (t === 'back') { this.go(this.prevCur && this.prevCur !== this.cur ? this.prevCur : 'home'); return }
      if (t.indexOf('res/') === 0) {
        this.resType = t.slice(4)
        this.cur = 'res'
        this.syncUrl()
        this.loadRes()
        return
      }
      // ★ 离开战场页就停掉倒计时轮询，避免在后台一直打接口
      if (t !== 'battle') this.stopBattleTimer()
      // ★ 2026-09-29 记录上一页，供各页 [返回]（goBack）回到上一页
      if (t !== this.cur) this.prevCur = this.cur
      this.cur = t
      this.syncUrl()
      if (t === 'home') { this.load(); this.loadWelfare(); this.loadHomeChats() }
      else if (t === 'troops' || t === 'troop' || t === 'defence' ||
               t === 'troopview' || t === 'trainpre' || t === 'troopstat') {
        this.loadTroops()
        // ★ 训练页(troop)的队列行要按背包里的训练加速道具档位渲染 [加速] 按钮
        if (t === 'troop' || t === 'troops') this.loadBag()
      }
      else if (t === 'hq') {
        // ★ 2026-09-28 预设编队 tab 需要：本城坐标(/view)、集结令(gatherMax)、军官(onDutyOfficers)、背包(集结令数量)
        this.load()
        this.loadTroops().then(() => this.loadTargets())
        this.loadOrders()
        this.loadPresets()
        this.loadOnDutyOfficers()
        this.loadBag()
        // 进来就先算一次：让「油耗/负重/本次出兵上限」立刻是准确值（若正处在预设编队 tab）
        this.$nextTick(() => { if (this.hqTab === 4) this.presetCalc() })
      }
      // ★ 2026-09-26 修复「加速道具买完实际使用不生效」：科技/建筑/训练页的 [加速]
      //   现在按「背包里实际拥有的加速道具档位」渲染按钮，所以进页时要拿到背包数据。
      else if (t === 'techs') { this.loadTechs(); this.loadBag() }
      else if (t === 'buildm' || t === 'builds' || t === 'cityhall') this.loadBag()
      else if (t === 'map') { this.backToMap(); this.loadStars() }
      // ★ 2026-09-28 军情页：分区由 reportTab 决定，且 switchReportTab 内部会 syncUrl，
      //   刷新后 reportTab 已由 restoreFromUrl 从 ?rtab= 恢复，所以不会跳回「军队动态」。
      else if (t === 'reports') { this.curReport = null; this.switchReportTab(this.reportTab) }
      // ★ 2026-09-30 计谋页（行军计谋专用）：进页/刷新都重拉信号弹持有量
      else if (t === 'scheme') this.loadSchemes()
      else if (t === 'mail') { this.loadMails(); this.loadFriends(); this.loadPmCandidates(); this.loadPmConvs() }
      else if (t === 'friends') this.loadFriends()
      else if (t === 'liaison') this.loadLiaison()
      else if (t === 'tasks') this.loadTasks()
      else if (t === 'welfare') this.loadWelfare()
      else if (t === 'rank') this.loadRank()
      // 城市列表页要显示「军衔可建城数」，所以也拉一次军衔数据
      // ★ 2026-09-28 兜底：cities 只在 /view 里下发，万一初始化那次请求失败（或列表为空）
      //   → 进页时补拉一次 /view，别让玩家看到一个空列表还不知道为什么。
      else if (t === 'cities') { this.loadRank(); if (!this.cities.length) this.load() }
      else if (t === 'bag') this.loadBag()
      else if (t === 'treasure') this.loadBag()   // ★ 宝物页：展示采集宝物（bagTreasures）
      else if (t === 'mall') {
        this.loadMall(true)
        if (this.mallTab === 'equipment') this.loadEquipShop()
        if (this.mallTab === 'chest') this.loadChests()
      }
      else if (t === 'exchange') this.loadExchange()
      else if (t === 'corps') {
        // ★ 2026-09-25 进入军团页重置子栏缓存 → 再次进入时拉到最新数据；
        //   单纯切 tab 不会重复拉接口（见 switchCorpsTab）。
        this.corpsRelations = null
        this.corpsWars = null
        this.corpsMall = { loaded: false, my_points: 0, corps_points: 0, items: [], my_bought: {} }
        this.loadCorps()
        this.switchCorpsTab(this.corpsTab)
      }
      else if (t === 'orders') {
        // ★ 2026-09-25 用户要求「出征队列按照军队动态那种展示」→ 数据源与军队动态统一：
        //   每次进页都重新拉一遍（刷新页面后也是走这里），队列不会再「刷新就消失」。
        this.loadDynamics()
      }
      else if (t === 'battle') {
        // ★ 战场页刷新后不能只靠 URL 恢复：订单 id 只存在内存里，刷新就丢了。
        //   所以带 id 就直接拉，没带就从后端找回当前进行中的那场战斗。
        if (this.battleOrderId) this.loadBattle()
        else this.resumeBattle()
      }
      else if (t === 'notices') this.loadNotices()
      else if (t === 'wilds') this.loadWilds()
      else if (t === 'orderpre') {
        // ★ 必须一起加载背包：出征页的「集结令」要读 bagItems，
        //   只进背包页才 loadBag 的话，出征页会永远显示「0个」且输入框被禁用。
        //
        // ★ 用户反馈「出征还有上次留的数据」→ 每次进出征页都把上次填的东西清干净
        //   （兵力/军官/携带资源/宿营/集结令），否则上一次的部队数量会残留，
        //   很容易误发一支自己没打算派的队伍。
        this.resetOrderForm()
        // ★ 2026-09-28 预设编队下拉：每次进出征页刷新预设列表（别人/别城新增的预设也能及时出现）
        this.loadPresets()
        // ★ 出征页顶部要显示「出发城市」，随军资源的「城内现有」也取自 /view，
        //   所以这里必须连 /view 一起拉 —— 否则切换城市后出征页仍显示上一座城的名字/资源。
        this.load()
        this.loadTroops()
        this.loadOnDutyOfficers()
        this.loadCityOfficers()
        this.loadBag()
        // 进来就先算一次：让「本次出兵 0 / 上限 N」和集结令上限立刻是准确值
        // （原来要玩家自己点[计算]才会显示）
        this.$nextTick(() => this.doCalc())
      }
      else if (t === 'acade') this.loadAcade()
      else if (t === 'wareset') this.loadWare()
      else if (t === 'citymove') this.loadMoveInfo()
      else if (t === 'sourceset') this.loadProduce()
      else if (t === 'activity') this.loadActivity()
      else if (t === 'info') this.loadSelfInfo()
      else if (t === 'factory') this.loadTroops()
      // 城市状态页要显示「人口/空闲人口」，/view 才带 pop_used → 进页先拉一次
      else if (t === 'citystatus') this.load()
    },
    // 节日活动
    loadActivity () {
      api.get('/games/ezfy/activity').then(r => {
        if (r.code === 0) this.activities = r.data.activities || []
      })
    },
    fmtLeft (sec) {
      if (!sec || sec <= 0) return '已结束'
      const d = Math.floor(sec / 86400)
      const hh = Math.floor((sec % 86400) / 3600)
      const mm = Math.floor((sec % 3600) / 60)
      if (d > 0) return d + '天' + hh + '小时'
      if (hh > 0) return hh + '小时' + mm + '分'
      return mm + '分'
    },
    load () {
      api.get('/games/ezfy/view').then(r => {
        if (r.code === 0) {
          const d = r.data
          this.profile = d.profile
          this.userBrief = { account: d.account || '', level: d.user_level || 0, exp: d.user_exp || 0 }
          this.officerCount = d.officer_count || 0
          this.rankName = d.rank_name
          this.rankPost = d.rank_post
          this.loadRank()
          this.city = d.city
          // ★ 2026-09-28：头部资源栏「/」右侧展示每小时产量
          this.resProd = Object.assign({ gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }, d.res_prod || {})
          // ★★ 2026-09-28 修复「切换城市 → 城市列表空了」：
          //   1bafa1a（格式、民心民怨）新增安抚参数时，把原本这一行 `this.cities = d.cities`
          //   覆盖删掉了。而 cities 在 data 里初值就是 []，**全文件再无第二处赋值** ——
          //   于是城市列表恒为空（「已有 0 座」、建城页也一直显示 0）。
          //   后端 /view 一直在下发 cities（ezfy.go 的 View → h.cityViews），这里接住即可。
          //   ⚠️ 改 load() 时别再把这一行弄丢：它是 cities 的唯一数据源。
          this.cities = d.cities || []
          this.placate = Object.assign({ gold: 50000, grievance: 2, feelings: 1, cooldown_min: 15, cd_left: 0 }, d.placate || {})
          this._placateAt = Date.now() // 安抚冷却快照时刻（见 placateCdLeft）
          this.city = d.city
          this.continent = d.continent
          this.cityKindRaw = d.city_kind || ''
          this.cityIsSea = !!d.is_sea
          this.protectedUntil = d.protected
          this.boostUntil = d.boost
          this.buildings = d.buildings
          this.buildingPool = d.building_pool || []
          this.militaryCap = d.military_cap || 33
          this.resourceCap = d.resource_cap || 33
          // ★ 2026-09-26 两个开关：后端未下发（老版本）时按「开」处理，与后端默认一致
          this.housePopLimitOn = d.house_pop_limit_on === undefined || d.house_pop_limit_on === null
            ? true : !!Number(d.house_pop_limit_on)
          this.conveneFlexibleOn = d.convene_flexible_on === undefined || d.convene_flexible_on === null
            ? true : !!Number(d.convene_flexible_on)
          // ★ 2026-09-26 召集消耗/收益（后端保证 >= 1，兜底默认 10 万）
          this.conveneFoodCost = Number(d.convene_food_cost) || 100000
          this.convenePopGain = Number(d.convene_pop_gain) || 100000
          // ★ 2026-09-26 全局硬性人口上限（0 = 不限），超过禁止召集
          this.convenePopMax = Number(d.convene_pop_max) || 0
          this.wildlands = d.wildlands
          this.queues = d.queues
          this.marching = d.marching
          this.occupying = d.occupying
          this.unreadReports = d.unread_reports
          // ★ 占用人口随 /view 一起下发：首页/城市状态页的「空闲人口」不再依赖
          //   「有没有进过军队页」（原来没进过就按 0 算，空闲人口显示成满人口）
          this.popUsed = d.pop_used || 0
          this.cityPop = d.city.pop || 0
          this.taxInput = d.city.tax_rate
          this.applyResNames(d.res_names)
          // ★ 集结令配置（管理端可配，默认 99）：跟着 /view 一起下发，
          //   这样一进页面（还没点[计算]）输入框的上限就是对的。
          this.gatherCfg = {
            max: d.gather_max > 0 ? d.gather_max : 0,
            per: d.gather_per > 0 ? d.gather_per : 100000,
            have: d.gather_have || 0
          }
          // ★ 首页要显示「每日签到：已签到/签到」，但 /view 不下发 welfare。
          //   不补这一下，签到完回首页仍显示「签到」——用户反馈的 bug。
          if (this.cur === 'home') this.loadWelfare()
        }
      })
    },
    // 资源显示名：把后端下发/读取到的名字合并进兜底值
    applyResNames (d) {
      if (!d || typeof d !== 'object') return
      const n = Object.assign({}, RES_NAMES)
      const sh = Object.assign({}, RES_SHORT)
      Object.keys(RES_NAMES).forEach(k => { if (d[k]) n[k] = d[k] })
      const sd = d._short || {}
      Object.keys(RES_SHORT).forEach(k => { if (sd[k]) sh[k] = sd[k] })
      this.resNames = n
      this.resShort = sh
      this.resDes = buildResDes(n)
    },
    // 单独拉一次（有些页面不经过 /view）
    loadResCfg () {
      return api.get('/games/ezfy/res-cfg').then(r => {
        if (r.code === 0) this.applyResNames(r.data)
      }).catch(() => {})
    },
    loadTroops () {
      return api.get('/games/ezfy/troops').then(r => {
        if (r.code === 0) {
          this.troopsData = r.data
          // 占用人口（训练中；已训练完成的部队不占人口）
          this.popUsed = r.data.pop_used || 0
          this.cityPop = r.data.pop || 0
          if (r.data.cfgs.length && !this.trainSel) this.trainSel = null
          // 司令部配置表按兵种建键, 兵种数据后到时要补齐, 否则渲染会取到 undefined
          this.ensureTargetCfg()
        }
      })
    },
    // 保证每个兵种在 targetCfg 里都有条目(司令部页 v-model 直接取 targetCfg[id].atk)
    ensureTargetCfg () {
      const cfg = Object.assign({}, this.targetCfg)
      for (const t of (this.troopsData.cfgs || [])) {
        if (!cfg[t.id]) cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1 }
      }
      this.targetCfg = cfg
    },
    loadTechs () {
      // ★ 2026-09-28 多城研究：科技页按**当前城**视角加载（科研中心等级/研究限制）
      api.get('/games/ezfy/techs?city_id=' + (this.city ? this.city.id : 0)).then(r => {
        if (r.code === 0) this.techsData = r.data
      })
    },
    loadChats () {
      api.get('/games/ezfy/chat?channel=' + this.chatChannel +
              '&page=' + this.chatPage + '&size=' + this.chatSize).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.worldChats = d.chats || []
          this.chatPlayers = d.players
          this.chatHasCorps = !!d.has_corps
          this.chatCorpsName = d.corps_name || ''
          this.chatCorpsPlayers = d.corps_players || 0
          this.chatCanSend = d.can_send !== false
          this.chatTotal = d.total || 0
          if (d.page) this.chatPage = d.page
          // ★ 军团频道但没军团：后端会降级成公共频道并把世界消息塞回来，
          //   这里按「军团频道」的语义清空，改为显示 0 人 + 引导。
          if (this.chatChannel === 2 && !this.chatHasCorps) {
            this.worldChats = []
            this.chatPlayers = 0
            this.chatCorpsPlayers = 0
            this.chatCanSend = false
            this.chatTotal = 0
          }
        }
      })
    },
    // 聊天翻页（-1 上一页 / 1 下一页）
    chatGo (delta) {
      const next = this.chatPage + delta
      if (next < 1 || next > this.chatTotalPages) return
      this.chatPage = next
      this.loadChats()
    },
    switchChannel (ch) {
      this.chatChannel = ch
      this.chatMsg = ''
      this.chatPage = 1
      this.loadChats()
    },
    // 首页「世界聊天」预览: 汇总 个人/同盟/系统 三个来源并带标识
    loadHomeChats () {
      api.get('/games/ezfy/chat/home').then(r => {
        if (r.code === 0) {
          this.homeChats = r.data.chats || []
          this.chatPlayers = r.data.players
        }
      })
    },
    // 收件人候选(好友 + 同军团成员 + 最近聊过的人), 只是方便输入, 也可以直接手填号码/昵称
    loadPmCandidates () {
      const list = []
      const seen = {}
      const push = (id, name) => {
        if (!id || seen[id]) return
        seen[id] = 1
        list.push({ id: id, name: name })
      }
      ;(this.friends || []).forEach(f => push(f.id, f.nickname))
      ;(this.corpsMembers || []).forEach(m => push(m.user_id || m.id, m.nickname || m.name))
      ;(this.homeChats || []).forEach(c => { if (c.user_id) push(c.user_id, c.user_name) })
      this.pmCandidates = list
    },
    doSendPm () {
      const to = (this.pmTo || '').trim()
      const content = (this.pmContent || '').trim()
      if (!to) { this.notify('请填写收件人(游戏ID或昵称)'); return }
      if (!content) { this.notify('请填写内容'); return }
      api.post('/messages', { to_name: to, content: content }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送')
          this.pmContent = ''
          this.loadMails()
          // ★ 发完就把「和这个人」的聊天记录刷新出来（选中谁就聊谁）
          if (this.pmPeer) {
            this.selectPm(this.pmPeer.id)
          } else {
            api.get('/messages/conversations').then(r2 => {
              if (r2.code === 0) {
                this.pmConvs = (r2.data || []).slice(0, 50)
                const hit = this.pmConvs.find(c => c.nickname === to || c.username === to)
                if (hit) this.selectPm(hit.user_id)
              }
            })
          }
        } else this.notify(r.msg || '发送失败')
      })
    },
    // ★ 私聊：打开与某位玩家的聊天（选中谁就聊谁，并带出历史记录）
    //   userId 为空时只进页面、显示会话列表。
    openPm (userId) {
      this.cur = 'mail'
      this.pmPeer = null
      this.pmChat = []
      this.pmTo = ''
      this.loadPmConvs()
      if (userId) this.selectPm(userId)
    },
    loadPmConvs () {
      api.get('/messages/conversations').then(r => {
        if (r.code === 0) this.pmConvs = (r.data || []).slice(0, 50)
      })
    },
    // 切换聊天对象：拉出「我和 TA」的全部历史记录，并把对方设为收件人
    selectPm (userId) {
      if (!userId) return
      api.get('/messages/with/' + userId).then(r => {
        if (r.code === 0) {
          this.pmPeer = r.data.peer || null
          this.pmChat = r.data.list || []
          this.pmTo = this.pmPeer ? (this.pmPeer.nickname || this.pmPeer.username) : ''
          this.loadMails()
          this.loadPmConvs()
        } else {
          this.notify(r.msg || '打开会话失败')
        }
      })
    },
    loadMails () {
      api.get('/messages/inbox').then(r => {
        if (r.code === 0) this.mails = (r.data.list || []).slice(0, 30)
      })
    },
    // ★ 游戏内好友（不碰家园 /friends）
    loadFriends () {
      api.get('/games/ezfy/friends').then(r => {
        if (r.code === 0) this.friends = r.data.list || []
      })
      this.loadFriendApplies()
    },
    loadFriendApplies () {
      api.get('/games/ezfy/friends/applies').then(r => {
        if (r.code === 0) this.friendApplies = { inbox: r.data.inbox || [], outbox: r.data.outbox || [] }
      })
    },
    // ---- 好友搜索/添加(复刻 addToFriend) ----
    doFriendSearch () {
      const kw = (this.friendKeyword || '').trim()
      if (!kw) { this.notify('请输入游戏ID / 玩家号码 / 昵称'); return }
      // ★ 先走游戏内搜索（支持「游戏ID」，不随家园号码变化）；
      // ★ 只搜游戏内玩家（不回落家园 /friends/search，避免把家园好友混进来）
      api.get('/games/ezfy/friends/search?keyword=' + encodeURIComponent(kw)).then(r => {
        if (r.code === 0) {
          this.friendSearchList = r.data.list || []
          this.friendSearchDone = true
        } else {
          this.notify(r.msg || '搜索失败')
        }
      }).catch(() => this.notify('搜索失败'))
    },
    async doAddFriend (u) {
      const remark = await this.ask('给 ' + u.nickname + ' 的验证信息（可留空）', { input: true, placeholder: '可留空' })
      if (remark === null) return
      api.post('/games/ezfy/friends/apply', { target_id: u.user_id, remark: remark }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送好友申请')
          this.doFriendSearch()
          this.loadFriends()
        } else {
          this.notify(r.msg || '添加失败')
        }
      })
    },
    // 处理好友申请（同意/拒绝）
    doHandleApply (a, agree) {
      api.post('/games/ezfy/friends/handle', { apply_id: a.apply_id, agree: agree }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : (agree ? '已同意' : '已拒绝'))
          this.loadFriends()
        } else this.notify(r.msg || '操作失败')
      })
    },
    // 删除游戏内好友
    doDelFriend (f) {
      this.ask('确定解除与「' + f.nickname + '」的游戏好友关系吗？').then(ok => {
        if (!ok) return
        api.post('/games/ezfy/friends/delete', { friend_id: f.user_id }).then(r => {
          if (r.code === 0) { this.notify(r.data && r.data.msg ? r.data.msg : '已解除'); this.loadFriends() }
          else this.notify(r.msg || '操作失败')
        })
      })
    },
    loadReports () {
      // category: 1 军情警讯 2 战斗报告(战报查询)；reportTab===5 → 军团战报(corps)
      const cat = this.reportTab === 3 ? 1 : 2
      let url = '/games/ezfy/reports?'
      if (this.reportTab === 5) url += 'corps=1'
      else url += 'category=' + cat + '&city_id=' + (this.city ? this.city.id : 0)
      if (this.reportWord) url += '&word=' + encodeURIComponent(this.reportWord)
      api.get(url).then(r => {
        if (r.code === 0) {
          this.reports = r.data.reports || []
          this.reportCounts = r.data.counts || {}
          this.reportRadar = r.data.radar || 0
          this.reportRecon = r.data.recon || 0
          this.reportIntel = r.data.intel || this.reportRadar
          this.repPage = 1
        }
      })
    },
    // ★ 2026-10-01 徽标数字专用轻量接口（独立于 loadReports：
    //   军情页落在任意分区都要刷新徽标，但不能因此把没看的战报标记已读）
    //   ★ 2026-10-01 军情按当前城过滤：徽标数字也只统计当前城的战报
    loadReportCounts () {
      api.get('/games/ezfy/reports/counts?city_id=' + (this.city ? this.city.id : 0)).then(r => {
        if (r.code === 0) this.reportCounts = r.data.counts || {}
      })
    },
    // ★ 2026-09-24 用户要求：军情警讯列表加 [防守报告]/[预警] 标签（其余类型无标签）
    intelTag (r) {
      const t = (r && r.title) || ''
      if (t.indexOf('守卫报告') === 0) return '防守报告'
      if (t.indexOf('军情警报') === 0) return '预警'
      return ''
    },
    // 守卫报告标题去掉冗余的「守卫报告: 」前缀，保留 城市名(坐标)
    intelTitle (r) {
      const t = (r && r.title) || ''
      if (t.indexOf('守卫报告: ') === 0) return t.slice('守卫报告: '.length)
      return t
    },
    // ---- 军队动态 ----
    // ★ 2026-10-01 军情按当前城过滤：只拉当前城市出发/驻守的部队
    loadDynamics () {
      api.get('/games/ezfy/reports/dynamics?city_id=' + (this.city ? this.city.id : 0)).then(r => {
        if (r.code === 0) {
          this.dynamics = r.data.dynamics || []
          // ★ 2026-09-28 倒计时自动刷新的时间基点：以「拿到数据的这一刻」为准，
          //   战斗中的 battle_left_ms 是相对剩余，必须配上它才能每秒往前推算。
          this._dynAt = Date.now()
          this.dynPage = 1
        }
      })
    },
    // ================= 战场指挥室（军情 → 军队动态 → [指挥]）=================
    // 每回合 30 秒：前 25 秒可下达前进/暂停/后退，后 5 秒锁定并由服务器结算，最多 40 回合。
    // 倒计时在前端本地自减，归零时拉一次服务端 —— 服务端是懒结算，请求时按时间补算回合。
    openBattle (orderId) {
      this.battleOrderId = orderId
      this.battleLeftMs = 0
      this.battleData = Object.assign({}, this.battleData, {
        order_id: orderId, done: false, actions: [], round: 0, win: 0
      })
      this.go('battle')
      this.loadBattle()
    },
    loadBattle () {
      if (!this.battleOrderId) return
      api.get('/games/ezfy/battle', { params: { order_id: this.battleOrderId } }).then(r => {
        if (r.code !== 0) {
          this.notify(r.msg || '战场不存在')
          this.leaveBattle()
          return
        }
        this.battleData = r.data
        this.battleLeftMs = r.data.round_left_ms || 0
        if (r.data.done) this.stopBattleTimer()
        else this.startBattleTimer()
      })
    },
    startBattleTimer () {
      this.stopBattleTimer()
      this.battleTimer = setInterval(() => {
        if (this.battleData.done) { this.stopBattleTimer(); return }
        this.battleLeftMs -= 1000
        if (this.battleLeftMs <= 0) {
          this.battleLeftMs = 0
          this.loadBattle()   // 到点 → 拉服务端推进一回合
        }
      }, 1000)
    },
    stopBattleTimer () {
      if (this.battleTimer) { clearInterval(this.battleTimer); this.battleTimer = null }
    },
    // troopId 省略 = 全军快捷指令；给了 troopId = 给该兵种**单独**下指令
    sendBattleCmd (cmd, troopId) {
      if (!this.battleOrderId) return
      api.post('/games/ezfy/battle/cmd', {
        order_id: this.battleOrderId, troop_id: troopId || 0, cmd
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '指令失败'); return }
        const st = r.data && r.data.state
        if (st) {
          this.battleData = st
          this.battleLeftMs = st.round_left_ms || 0
        }
        this.notify((troopId ? '该兵种已' : '全军已') + this.battleCmdName(cmd))
        if (r.data && r.data.done) this.stopBattleTimer()
      })
    },
    // ★ 指挥时逐兵种改「优先攻击目标」（2026-09-23 用户要求）：
    //   默认值来自司令部「兵种战斗配置」，这里改的只是**本场战斗**，不回写司令部。
    //   target = 0 表示「最近目标」；守方没有该兵种时服务器会自动回落打最近的。
    sendBattleTarget (troopId, ev) {
      if (!this.battleOrderId) return
      const target = parseInt(ev.target.value, 10) || 0
      const opt = (this.battleData.target_options || []).find(o => o.id === target)
      api.post('/games/ezfy/battle/target', {
        order_id: this.battleOrderId, troop_id: troopId, target_troop: target
      }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '目标设置失败'); return }
        const st = r.data && r.data.state
        if (st) {
          this.battleData = st
          this.battleLeftMs = st.round_left_ms || 0
        }
        this.notify('该兵种目标已设为' + (opt ? opt.name : '最近目标'))
        if (r.data && r.data.done) this.stopBattleTimer()
      })
    },
    doBattleAuto () {
      if (!this.battleOrderId) return
      api.post('/games/ezfy/battle/auto', { order_id: this.battleOrderId }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '操作失败'); return }
        const st = r.data && r.data.state
        if (st) { this.battleData = st; this.battleLeftMs = 0 }
        this.notify('已按「全军前进」打完这场战斗，战报稍后可在军情里查看')
        this.stopBattleTimer()
      })
    },
    battleCmdName (c) {
      return c === 'hold' ? '停止' : (c === 'retreat' ? '后退' : '前进')
    },
    // ★ 战场指挥室：行动日志按攻守上色 —— 我方绿色（跟随 is_atk），敌军红色
    battleLineClass (text) {
      const t = text || ''
      const isAtk = /^【攻方】/.test(t)
      const isDef = /^【守方】/.test(t)
      if (!isAtk && !isDef) return 'gray'
      const mineAtk = !!this.battleData.is_atk
      if (isAtk) return mineAtk ? 'green' : 'red'
      return mineAtk ? 'red' : 'green'
    },
    // ★ 战报详情/逐回合详情按行上色：攻方绿色、守方红色（看不清谁是谁 → 视觉区分）。
    //   返回 [{mode:'line'|'pair', text?, cls?, left?, right?}]；「战斗加成」行攻守各半段分两段上色。
    reportNiceLines (raw) {
      return String(raw || '').split('\n').map(ln => {
        const bi = ln.indexOf(' | ')
        if (ln.indexOf('战斗加成') >= 0 && bi > 0) {
          const left = ln.slice(0, bi + 1)
          const right = ln.slice(bi + 1)
          return {
            mode: 'pair',
            left: { text: left, cls: this.reportSideClass(left) },
            right: { text: right, cls: this.reportSideClass(right) }
          }
        }
        return { mode: 'line', text: ln, cls: this.reportLineClass(ln) }
      })
    },
    reportLineClass (ln) {
      const t = ln || ''
      const isAtk = /【攻方】|\[胜]攻方|\[平]攻方|【攻方军官】/.test(t) || (t.indexOf('攻方:') >= 0 && t.indexOf('守方:') < 0)
      const isDef = /【守方】|\[败]守方|\[平]守方|【守方军官】/.test(t) || (t.indexOf('守方:') >= 0 && t.indexOf('攻方:') < 0)
      if (isAtk && !isDef) return 'rpt-atk'
      if (isDef && !isAtk) return 'rpt-def'
      return ''
    },
    reportSideClass (seg) {
      if (seg.indexOf('守方') >= 0 && seg.indexOf('攻方') < 0) return 'rpt-def'
      if (seg.indexOf('攻方') >= 0 && seg.indexOf('守方') < 0) return 'rpt-atk'
      return ''
    },
    // resumeBattle 刷新页面后从后端找回「进行中的战斗」（订单 id 没存在 URL 里）
    resumeBattle () {
      api.get('/games/ezfy/reports/dynamics').then(r => {
        const list = (r.code === 0 && r.data && r.data.dynamics) || []
        const one = list.find(x => x.can_command)
        if (!one) {
          this.notify('当前没有正在进行的战斗')
          this.go('reports')
          return
        }
        this.battleOrderId = one.id
        this.loadBattle()
      })
    },
    leaveBattle () {
      this.stopBattleTimer()
      this.battleOrderId = 0
      this.go('reports')
    },
    // ★ 商城：切部位时回到第 1 页（否则会停在上一部位的分页位置看到空白）
    setShopSlot (s) {
      this.shopSlot = s
      this.shopPage = 1
    },
    // ★ 宝箱奖池：点名字展开/收起（用户要求「别直接展示，点击宝箱名字后展示」）
    toggleChestPool (id) {
      this.chestPoolId = (this.chestPoolId === id) ? 0 : id
      this.chestPoolWord = ''
      this.chestPoolPage = 1
    },
    // ★ 背包：点 [说明] 展开/收起道具说明
    toggleBagDesc (cfgId) {
      this.bagDescId = (this.bagDescId === cfgId) ? 0 : cfgId
    },
    // ★ 背包：切分类时回到第 1 页
    setBagCat (c) {
      this.bagCat = c
      this.bagPage = 1
    },
    // ★ 开箱快捷数量（[5]/[10] 不能超过单次上限）
    setChestCount (n) {
      const max = (this.chestOpen && this.chestOpen.open_max) || n
      this.chestCount = Math.min(n, max)
    },
    switchReportTab (t) {
      this.reportTab = t
      this.curReport = null
      // ★ 切换分区时回到第 1 页，避免停在上一次的分页位置看到空白
      this.repPage = 1
      this.dynPage = 1
      this.dynStationPage = 1
      // ★ 2026-09-28 分区写进 URL，否则刷新后回落到默认的「军队动态」(t=1)
      this.syncUrl()
      // ★ 2026-10-01 修复「徽标数字时有时无」：原来落 tab 1/2 只拉 dynamics、
      //   落 tab 3/4 只拉 reports —— 落在哪个分区决定徽标有没有数字。
      //   现在无论落在哪都先刷徽标 + 军队动态/驻军数据；列表仅在对应分区拉取。
      this.loadReportCounts()
      this.loadDynamics()
      if (t === 3 || t === 4 || t === 5) this.loadReports()
    },
    // ★ 战报详情页(reportview)顶部的分区导航：先回列表页再切到对应分区，
    //   直接调 switchReportTab 会停在 reportview 页面上。
    goReportTab (t) {
      this.stopBattleTimer()
      this.cur = 'reports'
      this.switchReportTab(t)   // 内部已 syncUrl，会把 cur=reports 与 rtab 一起写回
    },
    doCollectAll () {
      api.post('/games/ezfy/wild/collect-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
        } else this.notify(r.msg)
      })
    },
    // ★ 单支空闲驻军开始采集（2026-09-24 采集空闲化）
    startCollect (o) {
      const oid = (o && typeof o === 'object') ? o.id : o
      api.post('/games/ezfy/wild/start-collect', { order_id: oid }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    // ★ 一键收获：对每支采集中部队结算产出(资源直接入起点城市, 宝物进背包), 并停止采集原地待命
    //   —— 与单支 [停止] 同一个功能，只是一个批量一个单个。
    //   （玩法细则见代码注释：满一期结算资源+宝物；不满一期只按时长折算资源、无宝物；负重满后超出部分直接入城。）
    async doHarvestAll () {
      if (!await this.ask('确定收获所有采集中的部队吗？')) return
      this.harvestAllReq(false)
    },
    // ★ 2026-09-30 出发城市资源已达「配置的资源最大值」时，后端返回 confirm 标记;
    //   弹确认框，玩家确认后带 force=true 重发才真正收获（否则批量产出入库会因资源上限被丢弃）。
    harvestAllReq (force) {
      api.post('/games/ezfy/wild/harvest-all', { force }).then(r => {
        if (r.code === 0) {
          if (r.data && r.data.confirm) {
            this.ask(r.data.msg).then(ok => {
              if (ok) this.harvestAllReq(true)
            })
            return
          }
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    // ★ 一键召回：先结算未入城产出, 部队返航(资源已在收获/停止时入城, 召回只是撤兵)
    //   （玩法细则见代码注释：返航前结算未入城产出，满一期给资源+宝物，不满一期只按时长折算资源、无宝物。）
    async doRecallAll () {
      if (!await this.ask('确定召回所有驻守部队吗？')) return
      api.post('/games/ezfy/wild/recall-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    // ★ 单支采集部队「停止采集」(= 单支收获): 满一期结算资源+宝物, 不满只结算按采集时长的资源(无宝物);
    //   资源直接入起点城市; 部队原地待命不回城。
    async stopCollect (o) {
      const name = o.target_name + '(' + (o.target_x || 0) + ',' + (o.target_y || 0) + ')'
      if (!await this.ask('确定停止「' + name + '」采集吗？')) return
      this.stopCollectReq(o, false)
    },
    // ★ 2026-09-30 出发城市资源已达「配置的资源最大值」时，后端返回 confirm 标记;
    //   弹确认框，玩家确认后带 force=true 重发才真正停止结算。
    stopCollectReq (o, force) {
      api.post('/games/ezfy/wild/stop-collect', { order_id: o.id, force }).then(r => {
        if (r.code === 0) {
          if (r.data && r.data.confirm) {
            this.ask(r.data.msg).then(ok => {
              if (ok) this.stopCollectReq(o, true)
            })
            return
          }
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    loadTasks () {
      api.get('/games/ezfy/tasks').then(r => {
        if (r.code === 0) {
          this.taskGroups = r.data.groups || []
          // ★ 为爱发电卡：未发放时 love_cards 为空数组 → 隐藏对应 tab；若点过想去但被删除/清空则回落
          this.loveCards = r.data.love_cards || []
          if (this.taskTab === -1 && !this.loveCards.length) this.taskTab = 0
          this.selectTaskTab(this.taskTab)
        }
      })
    },
    // ★ 2026-09-27 任务 tab 选择并持久化（重启/刷新后仍在原 tab）
    selectTaskTab (v) {
      this.taskTab = v
      try { window.localStorage.setItem('ezfy_task_tab', String(v)) } catch (e) {}
    },
    // ★ 2026-09-27 恢复上次任务 tab（localStorage）
    restoreTaskTab () {
      try {
        const v = parseInt(window.localStorage.getItem('ezfy_task_tab') || '0', 10)
        return isNaN(v) ? 0 : v
      } catch (e) { return 0 }
    },
    // ★ 2026-09-28 司令部子 tab 切换并持久化（刷新后仍在原 tab）
    selectHqTab (v) {
      this.hqTab = v
      try { window.localStorage.setItem('ezfy_hq_tab', String(v)) } catch (e) {}
      // ★ 2026-09-28 预设编队 tab：拉预设 + 重算预览（预设页共享出征表单状态 orderOfficer/orderGather/orderTroops）
      if (v === 4) {
        this.loadPresets()
        this.$nextTick(() => this.presetCalc())
      }
    },
    // ★ 2026-09-28 恢复上次司令部子 tab（localStorage）
    restoreHqTab () {
      try {
        const v = parseInt(window.localStorage.getItem('ezfy_hq_tab') || '0', 10)
        return isNaN(v) ? 0 : v
      } catch (e) { return 0 }
    },
    // ★ 2026-09-27 为爱发电卡领取（多卡叠加、漏领累加、封顶30天）
    doLoveCardClaim () {
      api.post('/games/ezfy/love-card/claim', {}).then(r => {
        if (r && r.code === 0) {
          if (r.data) {
            if (r.data.love_cards) this.loveCards = r.data.love_cards
            if (this.loveCards.length === 0) this.taskTab = 0
            this.notify(r.data.msg || '已领取')
          }
          this.load()
        } else {
          this.notify(r && r.msg ? r.msg : '领取失败')
        }
      })
    },
    loadWelfare () {
      api.get('/games/ezfy/welfare').then(r => {
        if (r.code === 0) this.welfare = r.data
      })
    },
    loadRank () {
      api.get('/games/ezfy/rank').then(r => {
        if (r.code === 0) this.rankData = r.data
      })
    },
    // ★ 2026-09-28 军衔晋升：声望达标 + 背包宝物足够才能点[晋升]
    canPromote () {
      const m = this.rankData.mine
      if (!m || !m.next || !m.next.treasures) return false
      if (m.prestige < m.next.need) return false
      return m.next.treasures.every(t => t.have >= t.count)
    },
    doPromote () {
      if (!this.canPromote()) return
      const next = this.rankData.mine.next
      const req = next.treasures.map(t => t.name + '×' + t.count).join('、')
      if (!window.confirm('确认消耗宝物「' + req + '」晋升至「' + next.name + '」？')) return
      api.post('/games/ezfy/promote', {}).then(r => {
        if (r.code === 0) {
          alert('恭喜晋升至「' + r.data.rank_name + '」！')
          this.loadRank()
          this.load()
        } else {
          alert(r.msg || '晋升失败')
        }
      })
    },
    // ★ 军衔名 → 军衔等级 id：优先取军衔表，军衔表未加载时回落内置 20 级（与后端种子一致）
    rankIdByName (name) {
      if (!name) return 1
      const arr = this.rankData.ranks || []
      const hit = arr.find(x => x.name === name)
      if (hit) return hit.id
      const builtin = ['列兵', '上等兵', '下士', '中士', '上士', '军士长', '准尉', '少尉', '中尉', '上尉',
        '大尉', '少校', '中校', '上校', '大校', '少将', '中将', '上将', '大将', '五星上将']
      const i = builtin.indexOf(name)
      return i >= 0 ? i + 1 : 1
    },
    // ★ 军衔星级图标（20 级）：圆角徽章 + 金星，底色按 兵/士/尉/校/将 五大类区分，星数类内递增
    rankIcon (id) {
      const idn = Number(id) || 1
      const tiers = [[1, '#7BA544'], [3, '#6E7B8B'], [7, '#2F6F9F'], [12, '#8A63C9'], [16, '#C0392B']]
      let color = '#999'
      let from = 1
      for (const t of tiers) { if (idn >= t[0]) { color = t[1]; from = t[0] } }
      const stars = Math.max(1, Math.min(5, idn - from + 1))
      let layout
      if (stars === 1) layout = [[10, 10, 4.0]]
      else if (stars === 2) layout = [[6.9, 10, 2.9], [13.1, 10, 2.9]]
      else if (stars === 3) layout = [[5.6, 10, 2.4], [10, 10, 2.4], [14.4, 10, 2.4]]
      else if (stars === 4) layout = [[5.0, 10, 2.0], [8.3, 10, 2.0], [11.7, 10, 2.0], [15.0, 10, 2.0]]
      else layout = [[5.6, 7.6, 2.0], [10, 7.6, 2.0], [14.4, 7.6, 2.0], [7.9, 12.8, 2.0], [12.1, 12.8, 2.0]]
      const star = (cx, cy, R) => {
        const r = R * 0.42
        const p = []
        for (let k = 0; k < 10; k++) {
          const a = (Math.PI / 5) * k - Math.PI / 2
          const rad = (k % 2 === 0) ? R : r
          p.push((cx + rad * Math.cos(a)).toFixed(2) + ',' + (cy + rad * Math.sin(a)).toFixed(2))
        }
        return '<polygon points="' + p.join(' ') + '" fill="#FFD34E"/>'
      }
      const polys = layout.map(s => star(s[0], s[1], s[2])).join('')
      return '<svg class="ezfy-rank-ico" viewBox="0 0 20 20" width="18" height="18" style="vertical-align:-4px;margin-right:4px" role="img">' +
        '<rect x="1.5" y="1.5" width="17" height="17" rx="4.5" fill="' + color + '"/>' + polys + '</svg>'
    },
    // ★ 榜单前三名奖台化：给冠/亚/季军整行上色，超出三名的行不加类
    rankRowCls (rank) {
      if (rank === 1) return 'rank-1'
      if (rank === 2) return 'rank-2'
      if (rank === 3) return 'rank-3'
      return ''
    },
    loadBag () {
      return api.get('/games/ezfy/bag').then(r => {
        if (r.code === 0) {
          this.bagItems = r.data.items
          this.bagTreasures = r.data.treasures || []
          this.bagOfficers = r.data.officers || []
          this.bagSkills = r.data.skills || []
        }
      })
    },
    // ★ 商城分栏切换（item 道具 / equipment 装备套装 / chest 宝箱）
    switchMallTab (tab) {
      this.mallTab = tab
      if (tab === 'equipment') this.loadEquipShop()
      if (tab === 'chest') this.loadChests()
    },
    // ★ 宝箱（用钻石/黄金买，开箱按权重出套装件）
    loadChests () {
      api.get('/games/ezfy/chest').then(r => {
        if (r.code === 0) this.chestData = r.data
      })
    },
    openChestBuy (ch) {
      this.chestOpen = ch
      this.chestCount = 1
      this.chestPay = ch.price_diamond > 0 ? 'diamond' : 'gold'
      this.chestOpenDetailIdx = -1
      // ★ 跳转到独立开箱详情页确认
      this.chestResult = []
      this.cur = 'chestopen'
    },
    doOpenChest (ch) {
      const n = parseInt(this.chestCount) || 0
      if (n <= 0) { this.notify('请填写开箱数量'); return }
      if (n > ch.open_max) { this.notify('单次最多开 ' + ch.open_max + ' 个'); return }
      const cur = (ch.price_diamond > 0 && ch.price_gold > 0) ? this.chestPay : (ch.price_diamond > 0 ? 'diamond' : 'gold')
      api.post('/games/ezfy/chest/open', { chest_id: ch.id, count: n, currency: cur }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '开箱失败'); return }
        this.notify(r.msg || '开箱成功')
        this.chestResult = (r.data && r.data.results) || []
        this.chestOpen = null
        this.loadBag()
        this.load()
        // ★ 开完直接回商城宝箱分类页，结果在「上次开箱结果」展示（go('mall') 会自动刷新宝箱）
        this.go('mall')
      })
    },
    // ★ 装备商城（套装件，黄金/钻石购买）
    loadEquipShop () {
      this.loadEquipSets()
      api.get('/games/ezfy/equipshop').then(r => {
        if (r.code === 0) this.equipShop = r.data
      })
    },
    openEquipBuy (p) {
      this.equipShopBuy = p
      this.equipShopCount = 1
      // 散件统一钻石结算（定价 100~500 钻）；只有管理端把钻石价清 0 时才回落黄金
      this.equipShopPay = p.price_diamond > 0 ? 'diamond' : 'gold'
      // ★ 跳转到独立购买详情页确认
      this.cur = 'equipbuy'
    },
    doBuyEquip (p) {
      const n = parseInt(this.equipShopCount) || 0
      if (n <= 0) { this.notify('请填写购买数量'); return }
      const cur = p.price_diamond > 0 ? 'diamond' : 'gold'
      api.post('/games/ezfy/equipshop/buy', { cfg_id: p.id, count: n, currency: cur }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '购买失败'); return }
        this.notify(r.msg || '购买成功')
        this.equipShopBuy = null
        this.load()
        this.loadBag()
        // ★ 购买成功后返回商城（go('mall') 会自动刷新装备商城）
        this.go('mall')
      })
    },
    // ★ 拉商城数据。resetPage = true 时才回到第 1 页（进商城页签时用）。
    //   买完道具的刷新**不能**重置页码 —— 否则玩家在第 3 页买个东西就被弹回第 1 页。
    loadMall (resetPage) {
      api.get('/games/ezfy/mall').then(r => {
        if (r.code === 0) {
          this.mallItems = r.data.items
          // ★ 分类页签 + 钻石余额（钻石只能管理端充值）
          this.mallCatsList = r.data.categories || []
          this.mallDiamond = r.data.diamond || 0
          // ★ 单次购买上限（管理端可配，默认 99）
          this.mallBuyMax = parseInt(r.data.buy_max) > 0 ? parseInt(r.data.buy_max) : 99
          if (this.mallCat && this.mallCatsList.indexOf(this.mallCat) < 0) this.mallCat = ''
          if (resetPage) this.mallPage = 1
          else if (this.mallPage > this.mallTotalPages) this.mallPage = this.mallTotalPages
        }
      })
    },
    loadExchange () {
      api.get('/games/ezfy/exchange', {
        params: {
          page: this.exchangePage, size: this.exchangeSize,
          mpage: this.exchangeMPage, msize: this.exchangeMSize,
          es_type: this.exFilter
        }
      }).then(r => {
        if (r.code === 0) {
          this.exchangeOrders = r.data.orders || []
          this.exchangeTotal = r.data.total || 0
          this.exchangePage = r.data.page || 1
          this.exchangeSize = r.data.size || 10
          this.exchangeMine = r.data.mine || []
          this.exchangeMTotal = r.data.mtotal || 0
          this.exchangeMPage = r.data.mpage || 1
          this.exchangeMSize = r.data.msize || 10
          this.exchangeGold = r.data.gold
          // ★ 2026-09-30 向系统出售资源：读回回收比例 / 手续费 / 黄金上限
          if (r.data.sys_sell_ratio) this.sysSellRatio = r.data.sys_sell_ratio
          if (r.data.sys_sell_fee) this.sysSellFee = r.data.sys_sell_fee
          if (r.data.gold_max != null) this.goldMax = r.data.gold_max
        }
      })
    },
    // ★ 2026-09-24 卖家挂单类别检索: 切换类别回到第一页再加载
    onExFilter () {
      this.exchangePage = 1
      this.loadExchange()
    },
    loadCorps () {
      api.get('/games/ezfy/corps/list').then(r => {
        if (r.code === 0) {
          this.corpsList = r.data.corps
          this.myCorps = r.data.my_corps
          // ★ 2026-09-30 入团审核开关（我的军团）
          this.myCorpsNeedReview = (r.data.my_corps && r.data.my_corps.need_review) ? 1 : 0
          // 未入团时：从列表里找「我是否已申请且待审」的军团状态
          if (!this.myCorps) this.myApplyStatus = 0
          // 团长：进入信息 tab 时拉待审申请
          if (this.isLeader && this.corpsTab === 'info') this.loadCorpsApplies()
        }
      })
      api.get('/games/ezfy/corps/members').then(r => {
        if (r.code === 0) {
          this.corpsMembers = r.data.members
          // ★ 后端下发「我在军团的职位」，副团长也能发军团邮件
          this.myCorpsTitle = r.data.my_title || ''
          // ★ 2026-09-25 用户要求：军团信息 tab 显示军团总积分（新字段 corps_points）
          this.corpsPoints = r.data.corps_points || 0
        }
      })
      api.get('/games/ezfy/corps/chats').then(r => {
        if (r.code === 0) this.corpsChats = r.data.chats
      })
    },
    loadOrders () {
      api.get('/games/ezfy/orders').then(r => {
        if (r.code === 0) this.orders = r.data.orders
      })
    },
    loadNotices () {
      api.get('/games/ezfy/notices').then(r => {
        if (r.code === 0) {
          this.notices = r.data.notices
          // ★ 公告列表每次重新加载都回到第 1 页（否则刷新后可能停在超出范围的空页）
          this.noticePage = 1
          // ★ 首页外露公告由管理端配置条数（默认 1 条），后端直接下发 home_notices
          this.homeNotices = r.data.home_notices || []
        }
      })
    },
    loadWilds () {
      api.get('/games/ezfy/city/wildfull').then(r => {
        if (r.code === 0) {
          this.wildlands = r.data.wildlands
          this.occupies = r.data.occupies
        }
      })
    },
    loadTargets () {
      return api.get('/games/ezfy/targets').then(r => {
        // 先按兵种表建全默认值, 再覆盖服务端已保存的配置
        const cfg = {}
        for (const t of (this.troopsData.cfgs || [])) {
          cfg[t.id] = { atk: 0, atkMove: 1, def: 0, defMove: 1 }
        }
        if (r.code === 0) {
          for (const t of r.data.targets) {
            cfg[t.troop_id] = { atk: t.atk_target_troop, atkMove: t.atk_move, def: t.def_target_troop, defMove: t.def_move }
          }
        }
        this.targetCfg = cfg
      })
    },
    loadRes () {
      api.get('/games/ezfy/resources').then(r => {
        if (r.code === 0) {
          this.city = r.data.city
          this.resDetail = r.data.calc[this.resType]
        }
      })
    },
    // 地图数据统一入口(loadMap/moveMap/jumpTo 共用)
    applyMap (d) {
      this.mapCells = d.cells
      this.mapCx = d.cx
      this.mapCy = d.cy
      this.eliteCell = d.elite && d.elite.x ? d.elite : null
    },
    loadMap () {
      // ★ 必须带 r: 不带的话后端用默认半径返回, 格子数与前端切行用的 mapR 不一致,
      //   整张网格会错位(本城就不在中心格了)
      api.get('/games/ezfy/map?r=' + this.mapR).then(r => { if (r.code === 0) this.applyMap(r.data) })
    },
    // ★ 2026-09-25 用户反馈「[返回地图] 怎么都回到初始的地图页，我都移动好多次了」：
    //   地图视野中心(mapCx/mapCy)一直存在内存里，进详情页再回来时要按**离开时的中心**刷新，
    //   而不是重新 loadMap() 回本城；只有这次会话还没加载过地图(如刷新后直接进地图/详情)才回本城。
    //   (「回到本城」按钮仍然走 loadMap()，保持原样)
    backToMap () {
      if (this.mapCells && this.mapCells.length) this.jumpTo(this.mapCx, this.mapCy)
      else this.loadMap()
    },
    moveMap (dx, dy) {
      this.jumpTo(this.mapCx + dx, this.mapCy + dy)
    },
    // ---- 地图坐标查找 / 收藏列表(复刻原版地图页) ----
    jumpTo (x, y) {
      api.get('/games/ezfy/map?x=' + x + '&y=' + y + '&r=' + this.mapR)
        .then(r => { if (r.code === 0) this.applyMap(r.data) })
    },
    doJump () {
      const x = parseInt(this.jumpX)
      const y = parseInt(this.jumpY)
      if (!x || !y || x < 1 || x > 500 || y < 1 || y > 500) {
        this.notify('请输入 1~500 之间的横纵坐标')
        return
      }
      this.jumpTo(x, y)
    },
    loadStars () {
      api.get('/games/ezfy/map/stars').then(r => {
        if (r.code === 0) this.mapStars = r.data.stars || []
      })
    },
    toggleStars () {
      this.showStars = !this.showStars
      if (this.showStars) this.loadStars()
    },
    async addStar () {
      if (!this.selCell) return
      // ★ 已收藏 → 再点一次取消收藏
      const hit = this.mapStars.find(s => s.x === this.selCell.x && s.y === this.selCell.y)
      if (hit) { this.delStar(hit); return }
      // 备注名用「地形名(等级)」, 坐标由列表模板统一拼, 别在这里重复带上
      // ★ 2026-09-25：地图格子上城市统一显示「城市」，收藏备注名要用具体城市名，否则收藏列表分不清
      const def = this.selCell.area_type === 3
        ? ((this.selCell.name || '城市') + (this.selCell.owner ? '(' + this.selCell.owner + ')' : ''))
        : this.cellText(this.selCell)
      const name = await this.ask('备注名（最多16字）', { input: true, value: def })
      if (name === null) return
      api.post('/games/ezfy/map/stars', { x: this.selCell.x, y: this.selCell.y, name: name }).then(r => {
        if (r.code === 0) { this.showStars = true; this.loadStars() } else this.notify(r.msg || '收藏失败')
      })
    },
    delStar (s) {
      api.post('/games/ezfy/map/stars/delete', { id: s.id }).then(r => {
        if (r.code === 0) this.loadStars()
      })
    },
    // ---- 城市迁移(复刻 city/cityHallMove.html) ----
    // ★ 第十二轮：区域 = 大洲；迁城消耗道具（迁城计划/高级迁城计划/沿海迁城计划）
    loadMoveInfo () {
      api.get('/games/ezfy/city/move').then(r => {
        if (r.code === 0) {
          this.moveInfo = r.data
          this.moveItems = r.data.items || []
          const def = r.data.default_continent || 1
          if (r.data.areas && r.data.areas.length) {
            // 默认选中「欧洲」（后端下发的默认洲）；没有就取第一个
            const hit = r.data.areas.some(a => a.id === def)
            const pick = hit ? def : r.data.areas[0].id
            this.moveContinent = pick
            this.moveContinentSea = pick
            this.moveArea = pick
          }
        }
      })
    },
    async doMoveCity (type) {
      const body = { type: type }
      if (type === 'low') {
        body.continent_id = this.moveContinent
      } else if (type === 'high') {
        const x = parseInt(this.moveX)
        const y = parseInt(this.moveY)
        if (!x || !y) { this.notify('请输入横纵坐标'); return }
        body.x = x; body.y = y
      } else {
        // 沿海迁城：填了坐标就按坐标迁，没填就按所选大洲随机找沿海平原
        const x = parseInt(this.moveX2)
        const y = parseInt(this.moveY2)
        if (x && y) {
          body.x = x; body.y = y
        } else {
          body.continent_id = this.moveContinentSea
        }
      }
      const label = type === 'low' ? '迁城计划' : (type === 'high' ? '高级迁城计划' : '沿海迁城计划')
      const kind = (this.moveItems || []).find(i => i.code === type) || {}
      const have = kind.count || 0
      if (have < 1) {
        this.notify('背包里没有【' + label + '】，请先到商城购买')
        return
      }
      const contName = this.areaName(type === 'sea' ? this.moveContinentSea : this.moveContinent)
      const where = type === 'low' ? '迁入【' + contName + '】'
        : (type === 'sea' && !parseInt(this.moveX2) ? '迁入【' + contName + '】的沿海平原' : '迁到指定坐标')
      if (!await this.ask('确认使用【' + label + '】×1 ' + where + '吗？（当前持有 ' + have + ' 个）')) return
      api.post('/games/ezfy/city/move', body).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.load()
          this.loadMoveInfo()
        } else this.notify(r.msg || '迁城失败')
      })
    },
    areaName (id) {
      const a = (this.moveInfo.areas || []).find(x => x.id === id)
      return a ? a.name : '—'
    },
    // ---- 调整生产(复刻 city/sourceSet.html) ----
    loadProduce () {
      api.get('/games/ezfy/city/produce').then(r => {
        if (r.code === 0) {
          this.rateFood = r.data.rate_food
          this.rateSteel = r.data.rate_steel
          this.rateOil = r.data.rate_oil
          this.rateRare = r.data.rate_rare
        }
      })
    },
    doSaveRate () {
      const vals = [this.rateFood, this.rateSteel, this.rateOil, this.rateRare]
      for (const v of vals) {
        const n = parseInt(v)
        if (!n || n < 1 || n > 100) { this.notify('开工率需在 1~100 之间'); return }
      }
      api.post('/games/ezfy/city/produce', {
        rate_food: parseInt(this.rateFood), rate_steel: parseInt(this.rateSteel),
        rate_oil: parseInt(this.rateOil), rate_rare: parseInt(this.rateRare)
      }).then(r => {
        this.alert(r, '生产比例已调整')
        if (r.code === 0) this.loadProduce()
      })
    },
    // ---- 建筑操作 ----
    // ---- 建造页(复刻 preCreateMilitary / preCreateSource) ----
    openBuildPre (zone) {
      this.buildZone = zone || (this.cur === 'builds' ? 's' : 'm')
      this.buildSel = null
      this.cur = 'buildpre'
    },
    openBuildDetail (b) {
      this.buildSel = b
    },
    // 训练一键加速(本城 / 所有城市)
    doSpeedTrainAll () {
      api.post('/games/ezfy/troops/speed-all', { all_city: false }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.load()
        } else this.notify(r.msg)
      })
    },
    async doSpeedTrainAllCity () {
      if (!await this.ask('确定对所有城市的训练队列一键加速吗?(按剩余时间消耗' + this.resNames.gold + ')')) return
      api.post('/games/ezfy/troops/speed-all', { all_city: true }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.load()
        } else this.notify(r.msg)
      })
    },
    doBuild (b) {
      // ★ 用户要求：建造成功后跳回对应分区（资源区→资源区、军事区→军事区），并刷新建筑列表
      // ★ 用户反馈「连点会出现多条」→ 防抖：一次点击只下达一条建造命令
      this.once('build', () =>
        api.post('/games/ezfy/build', { building_id: b.building_id || b.bid }).then(r => {
          if (r.code === 0) {
            this.load()
            this.go(this.buildZone === 'm' ? 'buildm' : 'builds')
          } else this.notify(r.msg || '建造失败', 'error')
        })
      )
    },
    doUpgrade (b) {
      api.post('/games/ezfy/building/upgrade', { record_id: b.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : '建筑已开始升级', type: 'ok' }
          this.load()
        } else this.inlineTip = { bid: b.id, text: r.msg || '升级失败', type: 'error' }
      })
    },
    doMaxLevel (b) {
      // ★ 2026-09-25 用户纠正「一键9级 = 一键升级到 9 级，而不是升级满」：
      //   按钮文案是「一键{{max_level-1}}级」，就把目标等级一起发给后端（target_level），
      //   后端按目标级结算资源/图纸并停在那一级（不越过 9→10 这道要建筑图纸的坎）。
      // ★ 2026-09-30 用户要求：一键固定只到 9 级（去掉「一键11级/一键12级」）。
      const target = 9
      api.post('/games/ezfy/building/max-level', { record_id: b.id, target_level: target }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : ('已一键升级到' + target + '级'), type: 'ok' }
          this.load()
        } else this.inlineTip = { bid: b.id, text: r.msg || '升级失败', type: 'error' }
      })
    },
    // ★ 2026-09-25 用户要求「拆除询问下玩家是否拆除，玩家可能按错了」→ 先二次确认再拆
    //   （拆除是逐级降级，降到 0 级才彻底移除，提示里把这个后果说清楚）
    async doDeleteBuilding (b) {
      const nm = (b && b.name) || '该建筑'
      const lv = b && b.level ? ('' + b.level + '级') : ''
      const ok = await this.ask('确定拆除「' + nm + '」' + lv + '吗？\n拆除是逐级降级（每次降 1 级），降到 0 级才彻底移除，且不退还建造资源。')
      if (!ok) return
      api.post('/games/ezfy/building/delete', { record_id: b.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid: b.id, text: (r.data && r.data.msg) ? r.data.msg : '建筑已拆除', type: 'ok' }
          this.load()
        } else this.inlineTip = { bid: b.id, text: r.msg || '拆除失败', type: 'error' }
      })
    },
    // ============ 加速道具（建筑 3 / 训练 4 / 科技 5）============
    // ★ 2026-09-26 修复「加速道具买完实际使用不生效」：
    //   根因是**训练页 / 科技页的加速入口根本没接这些道具** ——
    //   训练页只有 [训练一键加速]（花黄金），科技页的 [加速] 调 /techs/speed
    //   （免费减 10 分钟，minutes 还能由前端随便传），于是商城买的
    //   「训练加速30分钟/2小时」「科技加速30分钟/2小时」买完在游戏里无处可用。
    //   现在三个页面统一口径：按**背包里实际拥有**的档位渲染 [加速30分钟]/[加速2小时]，
    //   点哪档就用哪档（顺带解决「买了 2 小时却自动用了 30 分钟」的困惑）。
    accItems (type) {
      return (this.bagItems || [])
        .filter(i => i.item_type === type && i.count > 0)
        .sort((a, x) => (a.param1 || 0) - (x.param1 || 0))
    },
    // 加速道具的档位名：120 →「2小时」，30 →「30分钟」；百分比档(24/25/26) →「30%」
    accLabel (a) {
      const m = (a && a.param1) || 0
      if (a && (a.item_type === 24 || a.item_type === 25 || a.item_type === 26)) return m + '%'
      if (m >= 60 && m % 60 === 0) return (m / 60) + '小时'
      return m + '分钟'
    },
    // 统一的「用道具加速」调用（cfg_id 走 /bag/use，后端按 item_type 决定加速目标）
    //   targetId：指定目标（建筑升级记录 id / 训练队列 id），0/undefined = 后端自动挑最早一条
    //   done：成功后的回调（刷新对应页面的数据）
    async useSpeedItem (item, type, targetId, done) {
      await this.loadBag()
      const it = (item && item.cfg_id) ? item : this.accItems(type)[0]
      if (!it) {
        this.notify('没有对应的加速道具，去商城购买后再加速', 'error')
        return
      }
      const body = { cfg_id: it.cfg_id, count: 1, city_id: this.city.id }
      if (targetId) body.record_id = targetId
      api.post('/games/ezfy/bag/use', body).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '加速成功', 'ok')
          this.loadBag()
          if (typeof done === 'function') done()
        } else this.notify(r.msg || '加速失败', 'error')
      })
    },
    async doSpeedBuilding (b, item) {
      // ★ 建筑加速必须消耗「建筑加速道具」(item_type=3)，没有道具则无法加速
      const bid = b && b.id
      await this.loadBag()
      const it = (item && item.cfg_id) ? item : this.accItems(3)[0]
      if (!it) {
        this.inlineTip = { bid, text: '没有建筑加速道具，去商城购买后再加速', type: 'error' }
        return
      }
      // ★ 带上 city_id：多城时「不传 city_id 走当前城」容易和玩家正在看的城错位
      // ★ 2026-09-27 带上 record_id：点哪条建筑就减哪条，避免后端总是挑最早结束的一条
      api.post('/games/ezfy/bag/use', { cfg_id: it.cfg_id, count: 1, city_id: this.city.id, record_id: bid }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid, text: (r.data && r.data.msg) ? r.data.msg : '加速成功', type: 'ok' }
          this.load()
          this.loadBag()
        } else this.inlineTip = { bid, text: r.msg || '加速失败', type: 'error' }
      })
    },
    // ★ 2026-09-27 严重漏洞修复：取消建筑升级**零退还**（资源与建筑图纸不退，建筑保留当前等级）。
    //   旧逻辑「全额退还」会让人「一键升满→取消」反复刷资源/图纸。
    async doCancelUpgrade (b) {
      if (!b || b.status === 0) return
      const bid = b.id
      if (!await this.ask('确定取消「' + (b.name || '该建筑') + '」的升级吗？\n' +
        '已消耗的资源与建筑图纸将不会退还，建筑保留当前等级。')) return
      api.post('/games/ezfy/building/cancel', { record_id: bid, city_id: this.city.id }).then(r => {
        if (r.code === 0) {
          this.inlineTip = { bid, text: (r.data && r.data.msg) ? r.data.msg : '已取消升级', type: 'ok' }
          this.load()
          this.loadBag()
        } else this.inlineTip = { bid, text: r.msg || '取消失败', type: 'error' }
      })
    },
    // 训练加速（训练页队列行 / 建筑页底部的「训练加速道具」入口）
    // ★ 2026-09-27 修复「没有按指定目标扣减」：训练页队列行带上 q.id，点哪条减哪条；
    //   建筑页底部入口 q 为 null → record_id=0，后端自动挑最早的一条
    doSpeedTrain (q, item) {
      this.useSpeedItem(item, 4, q ? q.id : 0, () => { this.loadTroops(); this.load() })
    },
    // 建筑名后的特殊入口(复刻原版 militaryIndex 里各建筑指向的功能页)
    bEntry (bid) {
      const map = {
        1: { label: '市政厅', cur: 'cityhall' },
        2: null,
        7: { label: '城防', cur: 'defence' },
        8: { label: '科技', cur: 'techs' },
        9: { label: '军校', cur: 'acade', tab: 'search' },
        10: { label: '参谋部', cur: 'acade', tab: 'officer' },
        11: { label: '交易所', cur: 'exchange' },
        12: { label: '仓库调配', cur: 'wareset' },
        13: { label: '司令部', cur: 'hq' },
        14: { label: '军工厂', cur: 'factory' },
        15: { label: '联络中心', cur: 'liaison' }
      }
      return map[bid] || null
    },
    goEntry (bid) {
      const e = this.bEntry(bid)
      if (!e) return
      if (e.tab) this.acadeTab = e.tab
      this.go(e.cur)
    },
    // ---- 城市操作 ----
    doConvene () {
      api.post('/games/ezfy/city/convene', {}).then(r => this.alert(r, '召集完成'))
    },
    // doPlacate 已上移到 /view 加载处（那里有 placate 参数，用于拼确认文案），此处不再重复定义
    doTax () {
      api.post('/games/ezfy/city/tax', { tax_rate: parseInt(this.taxInput) || 0 })
        .then(r => this.alert(r, '税率已调整', () => { this.load() }))
    },
    doRename () {
      api.post('/games/ezfy/city/rename', { name: this.renameInput }).then(r => this.alert(r, '城市已更名'))
    },
    doCreateCity () {
      api.post('/games/ezfy/city/create', { x: parseInt(this.newCityX) || 0, y: parseInt(this.newCityY) || 0 }).then(r => this.alert(r, '新城建造成功'))
    },
    // ★ 分页翻页（which: 'dyn' 军队动态 / 'sta' 驻军 / 'rep' 战报列表 / 'notice' 公告）
    //   ⚠️ 方法名必须与下方通用 pagerGo 不同：同名时对象字面量后定义会覆盖先定义，
    //   曾导致网易/公告「下一页」点到的是通用版 pagerGo（cur 传字符串 → 返回原值 → 无反应）。
    sectionPagerGo (which, delta) {
      if (which === 'dyn') {
        this.dynPage = Math.min(this.dynMarchTotalPages, Math.max(1, this.dynPage + delta))
      } else if (which === 'sta') {
        this.dynStationPage = Math.min(this.dynStationTotalPages, Math.max(1, this.dynStationPage + delta))
      } else if (which === 'notice') {
        this.noticePage = Math.min(this.noticeTotalPages, Math.max(1, this.noticePage + delta))
      } else if (which === 'exo') {
        this.exchangePage = Math.min(this.exchangeTotalPages, Math.max(1, this.exchangePage + delta))
        this.loadExchange()
      } else if (which === 'exm') {
        this.exchangeMPage = Math.min(this.exchangeMTotalPages, Math.max(1, this.exchangeMPage + delta))
        this.loadExchange()
      } else {
        this.repPage = Math.min(this.repTotalPages, Math.max(1, this.repPage + delta))
      }
    },
    // ★ 城市列表 [派遣]：像出征一样，把当前城市的部队/军官/随军资源送到自己另一座城
    doDispatchTo (ct) {
      if (ct.id === this.city.id) { this.notify('不能派遣到当前所在城市'); return }
      this.selCell = {
        x: ct.x, y: ct.y, area_type: 3, city_id: ct.id, user_id: ct.user_id,
        name: ct.name, level: ct.city_level, mine: true, occupied: true
      }
      this.selDetail = null
      this.orderType = 8
      this.orderCalc = null
      this.go('orderpre')
    },
    async doDestroyCity (ct) {
      const cur = this.city && ct.id === this.city.id ? '（这是当前所在城市，弃城后会自动切换到其他城市）' : ''
      const ok = await this.ask('确定弃城「' + ct.name + '」吗？' + cur + '该城市的建筑、部队、军官、野地都会一并消失，' +
        '坐标会恢复为普通平原。此操作不可恢复！')
      if (!ok) return
      api.post('/games/ezfy/city/destroy', { city_id: ct.id }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          // 摧毁的是当前城时后端会自动切到别的城 → 整体重载（含军官，军官跟城走）
          this.load()
          this.loadTroops()
          this.loadTechs()
          this.loadOnDutyOfficers()
        } else this.notify(r.msg)
      })
    },
    doSwitch (ct) {
      api.post('/games/ezfy/city/switch', { city_id: ct.id }).then(r => {
        if (r.code === 0) {
          // ★ 后端已把「当前城市」落库，这里必须整体重载，否则各页仍显示旧城数据
          this.load()
          this.loadTroops()
          this.loadTechs()
          // ★ 军官也是跟城走的：不一起刷新，出征页会残留上一座城的军官列表
          //   （用户反馈「切换城市后出征页的军官还是切换前那个城的」）。
          this.loadOnDutyOfficers()
          this.cur = 'home'
        } else this.notify(r.msg)
      })
    },
    // ★ 第九轮：从城市列表直接发起「运输」到自己的另一座城市
    doTransportTo (ct) {
      if (ct.id === this.city.id) { this.notify('不能运输到当前所在城市'); return }
      this.selCell = {
        x: ct.x, y: ct.y, area_type: 3, city_id: ct.id, user_id: ct.user_id,
        name: ct.name, level: ct.city_level, mine: true, occupied: true
      }
      this.selDetail = null
      this.orderType = 5
      this.orderCalc = null
      this.go('orderpre')
    },
    doAbandon (w) {
      api.post('/games/ezfy/city/abandon-wild', { wildland_id: w.id }).then(r => this.alert(r, '已放弃该野地'))    },
    // ★ 2026-09-28 「附属野地」列表里 [停止] 用：野地记录 → stopCollect 需要的最小订单对象
    //   stopCollect 只用 id 发请求、用 target_name/target_x/target_y 拼提示文案，故这里够了。
    wildOrderArg (w) {
      return {
        id: w.gather_order_id,
        target_name: w.terrain === 8 ? '海底森林' : (w.terrain_name || '野地'),
        target_x: w.x,
        target_y: w.y
      }
    },
    async doOccupy (op, o) {
      if (!await this.ask(op === 'build' ? '确定将该城市正式建立为自己的城市吗?' :
        op === 'destroy' ? '确定摧毁该城市吗? 城市及其建筑/部队将全部消失, 不可恢复!' :
          '确定将城市归还给原玩家吗?')) return
      api.post('/games/ezfy/city/occupy/' + op, { occupy_id: o.id }).then(r => this.alert(r, '操作已提交'))
    },
    // 野地列表 → [采集]：进「出征页」选兵种后再下达命令
    // ★ 原来直接 POST 且没带 troops，后端必然返回「请选择出征部队」，
    //   表现就是「点了采集没反应 / 采集发不出去」。
    openWildGather (w) {
      this.selCell = { x: w.x, y: w.y, area_type: 1, level: w.level,
        name: (w.terrain_name || '野地'), wild_id: w.id }
      this.selDetail = { id: w.id, level: w.level, x: w.x, y: w.y }
      this.orderType = 4
      this.orderCalc = null
      this.go('orderpre')
    },
    // ---- 军队 ----
    // 兵种详情(复刻 cityTroopView.html): 军队页/军工厂页/城防页点兵种名进来
    openTroopView (troopId) {
      this.troopViewId = troopId
      this.troopViewBack = this.cur === 'defence' ? 'defence'
        : (this.cur === 'factory' ? 'factory' : (this.cur === 'troop' ? 'troop' : 'troops'))
      this.go('troopview')
    },
    // 训练/建造确认页(复刻 createTroop.html / createDefence.html)
    openTrainPre (t, mode) {
      this.trainSel = t
      this.trainMode = mode || 'troop'
      this.trainCount = 10
      this.trainSplit = false
      this.go('trainpre')
    },
    // ★ 2026-09-28 城内军队表改为遍历 armyRows（全兵种含 0 数量），原 quickTrain 已无引用，移除。
    // ★ 2026-09-28 用户要求：训练页「(最多 N)」后面加 [最大]，一键填成上限。
    //   上限口径与页面上显示的「(最多 N)」完全一致（maxTrainable computed：
    //   资源 / 人口（城防为城防空间）取最小），点了不会填出个填不下的数。
    //   上限为 0 时不可点（灰掉），与出征页 [最大] 的处理保持一致。
    setTrainMax () {
      const max = this.maxTrainable
      if (max <= 0) return
      this.trainCount = max
    },
    doTrainPre () {
      if (!this.trainSel) return
      const n = parseInt(this.trainCount) || 0
      if (n <= 0) { this.notify('请填写建造数量'); return }
      api.post('/games/ezfy/troops/train', {
        troop_id: this.trainSel.id, count: n, split: this.trainSplit
      }).then(r => {
        this.alert(r, '征兵已开始')
        if (r.code === 0) this.loadTroops()
      })
    },
    // 拆除城防设施(复刻 troopDefence.html 每行的 [拆除])
    async doDismiss (t) {
      const have = this.troopCount(t.id)
      if (have <= 0) { this.notify('城内没有该城防设施'); return }
      if (!await this.ask('确定拆除全部 ' + t.name + '×' + have + ' 吗?')) return
      api.post('/games/ezfy/troops/dismiss', { troop_id: t.id }).then(r => {
        this.alert(r, '城防设施已拆除')
        if (r.code === 0) this.loadTroops()
      })
    },
    // 取消训练队列（用户要求：征兵序列玩家可以自己取消，资源全额退还）
    async doCancelTrain (q) {
      const ok = await this.ask('确定取消「' + q.name + '×' + q.count + '」的训练吗？' +
        '取消会收取 10% 手续费，其余资源退还（不受仓储上限影响）。')
      if (!ok) return
      api.post('/games/ezfy/troops/train/cancel', { queue_id: q.id }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '已取消训练，资源已退还', 'ok')
          this.loadTroops()
          this.load()
        } else this.notify(r.msg || '取消失败', 'error')
      })
    },
    // ★ 解散部队（用户要求：军队页面要有解散按钮，数量由玩家自己输入）
    async doDisband (t) {
      const input = await this.ask('解散「' + t.name + '」多少个？（当前 ' + t.count + ' 个）\n' +
        '解散后兵力直接销毁，不退还任何资源。', { input: true, value: '1', placeholder: '数量' })
      if (input === null || input === undefined) return
      const n = parseInt(input, 10)
      if (!n || n <= 0) { this.notify('解散数量必须是大于 0 的整数', 'error'); return }
      if (n > t.count) { this.notify('解散数量不能超过当前数量 ' + t.count, 'error'); return }
      api.post('/games/ezfy/troops/disband', {
        city_id: this.city ? this.city.id : 0, troop_id: t.troop_id, count: n
      }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : ('已解散 ' + t.name + '×' + n), 'ok')
          this.loadTroops()
          this.load()
        } else this.notify(r.msg || '解散失败', 'error')
      })
    },
    // 秒 → 「1分0秒 / 5分20秒 / 57秒」(复刻 createTroop.html 的「时间」)
    durText (sec) {
      const s = Math.max(0, Math.floor(Number(sec) || 0))
      if (s < 60) return s + '秒'
      const m = Math.floor(s / 60)
      if (m < 60) return m + '分' + (s % 60) + '秒'
      const hh = Math.floor(m / 60)
      return hh + '小时' + (m % 60) + '分'
    },
    // ★ 2026-09-28 用户要求「累计采集/采集资源实时变化、累加展示，不能只靠刷新」：
    //   采集中部队由后端下发 gather = { start_ms, period_ms, per_food/steel/oil/rare }，
    //   前端据此每一秒(由 gatherNow 驱动)本地 extrapolate 出「累计时长+累计产出的资源」。
    //   ★ 2026-09-28 用户规则：本期已采按负重上限展示，超负重部分丢弃（total 封顶到 carry_cap）。
    liveGather (o) {
      const g = o && o.gather
      if (!g || !g.period_ms) return null
      const now = this.gatherNow || Date.now()
      const elapsed = Math.max(0, now - g.start_ms) // 毫秒
      if (elapsed <= 0) return null
      const frac = elapsed / g.period_ms
      const mk = v => Math.floor((v || 0) * frac)
      let food = mk(g.per_food)
      let steel = mk(g.per_steel)
      let oil = mk(g.per_oil)
      let rare = mk(g.per_rare)
      const gold = 0
      let total = food + steel + oil + rare
      const cap = o.carry_cap || 0
      // ★ 2026-09-28 用户反馈：装满负重后「本期已采」不能超过负重，超负重部分丢弃。
      //   把各分项等比收敛，使 total 封顶到负重 cap（total == cap，负重叠满）。
      let full = cap > 0 && total >= cap
      if (cap > 0 && total > cap) {
        const scale = cap / total
        food = Math.floor(food * scale)
        steel = Math.floor(steel * scale)
        oil = Math.floor(oil * scale)
        rare = Math.floor(rare * scale)
        total = Math.min(cap, food + steel + oil + rare)
        full = total >= cap
      }
      return { timeText: this.durText(elapsed / 1000), food, steel, oil, rare, gold, total, cap, full }
    },
    // ★ 2026-09-28 给单条军队动态附上实时采集视图 _lg（有 gather 才算采集中）；复用来避免模板算两遍。
    withLg (o) {
      const g = this.liveGather(o)
      return g ? Object.assign({}, o, { _lg: g }) : o
    },
    // ★ 2026-09-28 用户要求「军队动态 / 出征队列的『抵达时间：12秒』也要自动刷新」：
    //   倒计时原来是后端算好的一次性字符串(time_text)，只有重新拉接口才会变，
    //   静止不动看起来像卡死。改法与采集实况一致 —— 后端下发**绝对到点时间戳**，
    //   前端每秒(由 gatherNow 驱动)本地重算剩余。
    //   覆盖三种会走的倒计时：status=0 抵达(arrive_time) / status=2 返回(return_time) /
    //   战斗中(5) 本回合剩余(battle_left_ms 基点)。驻守采集累计 / 空闲待机 / 等待指挥不动。
    //
    // ★★ 归零后必须「让位」给后端，否则会卡在 0 秒（2026-09-28 用户报的 bug）：
    //   到点之后本地推算就失效了（后端该结算了），这时如果还继续返回「0秒」，
    //   就会**盖住**后端刚下发的新状态文案（如「驻守(空闲) / 待机」），用户永远看到 0 秒。
    //   处理：过了到点时刻再走一个 2 秒缓冲（等自动重拉把新数据换回来）就返回 ''，
    //   让模板回落到 o.time_text —— 也就是后端说了算。
    liveLeft (o) {
      if (!o) return ''
      const now = this.gatherNow || Date.now()
      // 战斗中：本回合剩余 = 拉数据那一刻的剩余 - 已过去的时间
      if (o.status === 5) {
        if (!o.battle_left_ms && o.battle_left_ms !== 0) return ''
        const left = Math.max(0, (o.battle_left_ms || 0) - (now - (this._dynAt || now)))
        return this.durText(left / 1000)
      }
      // 出征中：距抵达还差多久
      if (o.status === 0 && o.arrive_time > 0) {
        const left = o.arrive_time - now
        if (left <= -2000) return ''   // 早过了 2 秒还没换状态 → 交回后端文案
        return this.durText(left / 1000)
      }
      // 返航中：距回城还差多久
      if (o.status === 2 && o.return_time > 0) {
        const left = o.return_time - now
        if (left <= -2000) return ''
        return this.durText(left / 1000)
      }
      return ''
    },
    // 有倒计时就附上实时视图 _lt（无则返回原样，模板回落到后端给的 time_text）
    withLive (o) {
      const t = this.liveLeft(o)
      return t ? Object.assign({}, o, { _lt: t }) : o
    },
    doRecover (w) {
      // ★ 2026-09-25 用户反馈「伤兵救治后要手动刷新页面才显示」→ 成功后重拉军队数据(含伤兵营/逃兵营)
      api.post('/games/ezfy/troops/recover', { troop_id: w.troop_id, type: w.type })
        .then(r => this.alert(r, '伤兵已恢复', () => this.loadTroops()))
    },
    doRecoverAll (t) {
      api.post('/games/ezfy/troops/recover', { all: true, type: t })
        .then(r => this.alert(r, '伤兵已恢复', () => this.loadTroops()))
    },
    // ---- 科技 ----
    // ★ 2026-09-28 用户要求：科技列表不显示资源消耗/前置条件，点[研究]进详情页(techpre)查看后再确认
    openTechPre (t) {
      this.techSel = t
      this.go('techpre')
    },
    doTechPre () {
      if (!this.techSel) return
      // ★ 2026-09-28 多城研究：带上当前城（研究限制/扣资源都在当前城）
      api.post('/games/ezfy/techs/research', { tech_id: this.techSel.tech_id, city_id: this.city ? this.city.id : 0 }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '科技研究已开始', 'ok')
          this.go('techs')
          this.loadTechs()
        } else this.notify(r.msg || '研究失败', 'error')
      })
    },
    // ★ 2026-09-26 修复「科技加速道具买完实际使用不生效」：
    //   原来这里调 POST /techs/speed {minutes:10} —— 后端既不校验道具也不扣任何东西，
    //   等于「免费无限减 10 分钟」（minutes 还能被改成任意值，直接把研究刷完）。
    //   商城卖的「科技加速30分钟/2小时」因此完全没有使用入口。
    //   现在改成消耗科技加速道具(item_type=5)，与建筑页 [加速] 同一口径。
    doSpeedTech (item) {
      this.useSpeedItem(item, 5, 0, () => { this.loadTechs() })
    },
    async doCancelTech (t) {
      if (!await this.ask('确定取消研究「' + t.name + '」吗？取消研究将不退还已消耗的资源。')) return
      // ★ 2026-09-28 多城研究：取消只作用于**当前城**的研究队列；取消不退款
      api.post('/games/ezfy/techs/cancel', { tech_id: t.tech_id, city_id: this.city ? this.city.id : 0 }).then(r => {
        this.alert(r, '研究已取消，已消耗资源不退还')
        if (r.code === 0) this.loadTechs()
      })
    },
    // ---- 司令部 ----
    // 军官装备加成提示（有加成才显示，例：装备 军事+5 后勤+5）
    equipTip (o) {
      if (!o) return ''
      const parts = []
      if (o.equip_military) parts.push('军事+' + o.equip_military)
      if (o.equip_logistics) parts.push('后勤+' + o.equip_logistics)
      if (o.equip_learning) parts.push('学识+' + o.equip_learning)
      return parts.length ? '(装备 ' + parts.join(' ') + ')' : ''
    },
    // ★ 2026-09-25：openEquipDetail（跳独立「装备详情页」）已随页面一起删除 ——
    //   各装备表不再有 [查看] 按钮，改成点装备名/套装名在该行下面展开详情卡。
    // ★ 装备六项战斗属性的展示文案（伤害/防御/生命/移动距离/暴击几率/暴击伤害）
    equipAttrText (e) {
      if (!e) return ''
      const parts = []
      if (e.dmg) parts.push('伤害+' + e.dmg + '%')
      if (e.def) parts.push('防御+' + e.def + '%')
      if (e.hp) parts.push('生命+' + e.hp + '%')
      if (e.move) parts.push('移动距离+' + e.move + '%')
      if (e.crit) parts.push('暴击几率+' + e.crit + '%')
      if (e.crit_dmg) parts.push('暴击伤害+' + e.crit_dmg + '%')
      // 老装备（军/后/学 三维）走另一套展示
      if (e.military) parts.push('军事+' + e.military)
      if (e.logistics) parts.push('后勤+' + e.logistics)
      if (e.learning) parts.push('学识+' + e.learning)
      return parts.join(' ')
    },
    // 军官详情里「装备战斗加成」一行
    officerBattleText (b) {
      if (!b) return ''
      const parts = []
      if (b.dmg) parts.push('伤害+' + b.dmg + '%')
      if (b.def) parts.push('防御+' + b.def + '%')
      if (b.hp) parts.push('生命+' + b.hp + '%')
      if (b.move) parts.push('移动距离+' + b.move + '%')
      if (b.crit) parts.push('暴击几率+' + b.crit + '%')
      if (b.crit_dmg) parts.push('暴击伤害+' + b.crit_dmg + '%')
      return parts.join(' ')
    },
    // 防御兵种（城防 type=4：碉堡/榴弹炮/反坦克炮/防空炮…）固定阵地
    isDefenceTroop (t) {
      return !!t && t.type === 4
    },
    doSaveTargets () {
      const list = []
      // ★ 防御兵种(type=4) 固定阵地：前进/停止一律按「停止」提交
      const defIds = {}
      ;(this.troopsData.cfgs || []).forEach(c => { if (c.type === 4) defIds[c.id] = true })
      for (const tid in this.targetCfg) {
        const t = this.targetCfg[tid]
        const isDef = !!defIds[parseInt(tid)]
        list.push(api.post('/games/ezfy/targets', {
          troop_id: parseInt(tid), atk_target_troop: t.atk,
          atk_move: isDef ? 0 : t.atkMove,
          def_target_troop: t.def,
          def_move: isDef ? 0 : t.defMove
        }))
      }
      Promise.all(list).then(() => this.notify('战斗配置已保存'))
    },
    openOrder (o) {
      this.loadOrderView(o.id)
    },
    // ★ 2026-09-25 统一进「命令详情」的入口：拉详情 + 用 go() 把页面写进 URL。
    //   原来 openOrder 直接改 this.cur（不写 URL），刷新后就掉回上一个页面，
    //   看起来就是「出征队列/详情刷新后消失」。现在带 oid 进 URL，
    //   刷新时 restoreFromUrl 会按 oid 重新拉回来（见下方）。
    loadOrderView (orderId) {
      const oid = parseInt(orderId, 10)
      if (!oid) return
      api.get('/games/ezfy/orders/' + oid).then(r => {
        if (r.code === 0) {
          this.curOrder = r.data
          this.go('orderview')
        } else {
          this.notify(r.msg || '命令不存在或已结束')
          this.go('orders')
        }
      })
    },
    jumpReport (rid) {
      api.get('/games/ezfy/reports/' + rid).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
          this.showDetail = false
          this.go('reportview')
        }
      })
    },
    // 点开战报: 拉详情 + 标已读，跳转独立「战报详情」页（不再行内展开）
    openReport (r) {
      if (!r) return
      api.get('/games/ezfy/reports/' + r.id).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
        } else {
          // 详情接口异常时至少把列表里的内容显示出来
          this.curReport = r
        }
        this.showDetail = false
        const item = this.reports.find(x => x.id === r.id)
        if (item) item.is_read = 1
        // openReport 可能从「军情警讯」进也可能从「战斗报告」进，这里记录它来自哪个分区
        // ★ 2026-09-30 军团战报(category='corps')回到军团 tab
        this.reportTab = (r.category === 1) ? 3 : (r.category === 'corps' ? 5 : 4)
        this.go('reportview')
      })
    },
    // ★ 删除战报：不需要二次确认（用户要求）
    // ★ 2026-09-26 用户要求「战报查询 [查询] 右边加个 [一键删除]，物理删除吧节约服务器资源」：
    //   删的是**自己名下全部战报**（后端 DELETE，不软删），不可恢复 → 先走页面内确认条问一次。
    //   ⚠️ 有搜索词时列表只是筛选，删除范围仍是全部 —— 文案里写清楚，别让玩家以为只删列表里那几条。
    // ★ 2026-10-01 军情按当前城过滤：一键删除也只删**当前城市**的战报
    async doClearReports () {
      if (!this.reports.length) {
        this.notify('暂无战报可删除')
        return
      }
      if (!await this.ask('确定删除【当前城市】的全部战报吗？\n（物理删除，不可恢复；' +
        (this.reportWord ? '当前只是搜索筛选，删除范围仍是当前城市全部' : '共 ' + this.reports.length + ' 条') + '）')) return
      api.post('/games/ezfy/reports/clear', { city_id: this.city ? this.city.id : 0 }).then(r => {
        if (r.code === 0) {
          this.notify((r.data && r.data.msg) ? r.data.msg : '战报已全部删除')
          this.repPage = 1
          this.loadReports()
        } else this.notify(r.msg || '删除失败')
      })
    },
    delReport (r) {
      if (!r) return
      api.post('/games/ezfy/reports/' + r.id + '/delete', {}).then(res => {
        if (res.code === 0) {
          this.notify('战报已删除')
          this.curReport = null
          this.go('reports')
        } else {
          this.notify(res.msg || '删除失败')
        }
      })
    },
    // ★ 取消出征命令（用户要求「出征队列可以取消」）
    //   不限命令类型：行进中(0)/驻守中(1)都能取消，部队原路返回出发城市；
    //   运输/派遣带出去的随军资源也会随部队一起带回来。
    async doRecall (o) {
      const name = (o && o.type_name) || '该命令'
      if (!await this.ask('确定取消「' + name + '」吗？部队将原路返回出发城市。')) return
      api.post('/games/ezfy/order/recall', { order_id: o.id })
        .then(r => this.alert(r, name + '已取消', () => { this.loadOrders(); this.loadDynamics() }))
    },
    // ★ 2026-09-30 行军计谋：神兵天降(13,去程减80%)/战略转移(14,回程减360分钟)，耗 7 信号弹
    //   从计谋页(scheme)进入，使用后返回「军情 → 军队动态」。
    async doMarchScheme (o, schemeId) {
      const schemeName = schemeId === 13 ? '神兵天降' : '战略转移'
      if (!await this.ask('对 (' + o.target_x + ',' + o.target_y + ') 这支部队使用「' + schemeName + '」吗？消耗信号弹×7，每种计谋每支部队限一次。')) return
      api.post('/games/ezfy/scheme/use', { scheme_id: schemeId, order_id: o.id })
        .then(r => {
          if (r.code !== 0) { this.notify(r.msg || '发动失败'); return }
          this.notify('「' + schemeName + '」已生效')
          this.loadOrders()
          this.loadDynamics()
          this.loadBag()
          this.loadSchemes() // 刷新信号弹持有量（计谋页顶部）
          // ★ 2026-09-30 使用完返回军情军队动态页（用户要求）
          this.go('reports')
        })
    },
    // ★ 2026-09-30 打开行军计谋页：记录目标部队并跳到「计谋」页
    openScheme (o) {
      this.schemeOrder = o
      this.loadSchemes() // 拉最新信号弹持有量
      this.go('scheme')
    },
    orderStatusText (o) {
      if (o.status === 0) return '行进中 ' + this.remain(o.arrive_time)
      // ★ 2026-09-24 采集空闲化: 到达野地后空闲待命(需手工[采集]), 开始采集才显示「驻守采集」
      if (o.status === 1) return (o.order_type === 7 ? (o.arrive_time > 0 ? '驻守采集' : '驻守(空闲)') : '已到达')
      if (o.status === 2) return '返回中 ' + this.remain(o.return_time)
      if (o.status === 3) return '已完成'
      if (o.status === 6) return '等待中(目标已被进攻, 排队等待交战)'
      if (o.status === 5) return '战斗中 第' + (o.battle_round || 1) + '回合'
      return '全队阵亡'
    },
    // ---- 地图/出征 ----
    isCellStarred () {
      // ★ 已收藏的格子显示「已收藏」(再次点击取消收藏)
      // ★ 2026-09-30 优先用详情接口返回的 is_starred（同步、无异步时序问题），
      //   拿不到/旧后端再回落 mapStars 判断。
      if (this.selDetail && typeof this.selDetail.is_starred === 'boolean') {
        return this.selDetail.is_starred
      }
      return this.selCell && this.mapStars.some(s => s.x === this.selCell.x && s.y === this.selCell.y)
    },
    // ★ 军队动态里活动目标的坐标标识（打活动野地/活动寇/特殊城市的部队）
    actTag (actType) {
      if (actType === 1) return '活动'
      if (actType === 2) return '活动寇'
      if (actType === 3) return '特殊城市'
      return ''
    },
    // ★ 2026-10-01 地图世界 500×500（后端 ezfyWorldSize=500，有效坐标 1~499）：
    //   视野中心靠边时周边会带出越界格（如跳 1,1 后出现 -1,-1），统一判定为「空地」。
    isMapOOB (cell) {
      return !cell || cell.x < 1 || cell.x > 499 || cell.y < 1 || cell.y > 499
    },
    cellText (cell) {
      // 复刻 map/index.html: 格子文案为「名称(等级)」；★ 现在每格第二行统一显示坐标，
      //   所以这里一律只返回「名称」部分，本城也不再拼 (x,y)，避免和下面那行重复。
      // ★ 2026-09-25 用户反馈：玩家城市名太长，格子会被撑变形 → 地图上（含本城）统一显示「城市」，
      //   具体城市名/城主点进目标详情页再看（鼠标悬停的 title 里也有全称，见 cellTip）。
      // ★ 2026-10-01 越界格（地图外不存在的地方）统一显示「空地」
      if (this.isMapOOB(cell)) return '空地'
      if (cell.area_type === 3) return '城市'
      // 活动目标: 复刻 mapView.html 的「活动野地N级 / 活动寇N级 / 特殊城市N级」
      if (cell.act_type === 1) return '活动(' + cell.act_level + ')'
      if (cell.act_type === 2) return '活动寇(' + cell.act_level + ')'
      if (cell.act_type === 3) return '特殊(' + cell.act_level + ')'
      if (cell.name === '寇城(废墟)') return '墟'
      if (cell.area_type === 2) return '寇(' + cell.level + ')'
      if (cell.terrain === 8) {
        // ★ 纯海洋/海底森林分开显示(用户规范): 海洋不带等级, 海野显示海底森林(N级)
        if (cell.is_ocean) return '海洋'
        return '海底森林(' + cell.level + ')'
      }
      // 陆地野地按地形名显示(平原/草原/森林/盆地/丘陵/沼泽/岛屿)
      return (cell.terrain_name || '野') + '(' + cell.level + ')'
    },
    // ★ 格子悬浮提示：城市名字在格子里会被截断，鼠标悬停看全称。
    //   坐标已固定显示在格子第二行，这里不再重复拼。
    //   ★ 2026-09-25：格子上一律显示「城市」(含本城)，所以本城也给出全称提示。
    cellTip (cell) {
      if (this.isMapOOB(cell)) return '空地（地图外，不存在）'
      if (cell.area_type === 3) {
        return (cell.name || '城市') + (cell.owner ? ' · 城主 ' + cell.owner : '') +
          ' (' + cell.x + ',' + cell.y + ')'
      }
      return this.cellText(cell)
    },
    cellClass (cell) {
      if (this.isMapOOB(cell)) return 'ezfy-empty'
      if (cell.mine) return 'ezfy-mine'
      // ★ 2026-09-30 带名将守将的活动野地：特殊标识（优先于普通活动野地）
      if (cell.act_type === 1 && cell.act_officer) return 'ezfy-act-named'
      if (cell.act_type === 1) return 'ezfy-act-wild'
      if (cell.act_type === 2) return 'ezfy-act-kou'
      if (cell.act_type === 3) return 'ezfy-act-city'
      if (cell.area_type === 3) return 'ezfy-city'
      if (cell.area_type === 2) return 'ezfy-kou'
      if (cell.terrain === 8) return 'ezfy-sea'
      return 'ezfy-wild'
    },
    // 点「发现精英中立城市」→ 直接进该寇城的目标详情(里面就是 侦查/掠夺/征服 出征入口),
    // 而不是仅仅把地图挪过去
    openElite () {
      if (!this.eliteCell) return
      const c = this.eliteCell
      this.openCell({ x: c.x, y: c.y, area_type: 2, level: c.level || 0, name: '寇城' })
    },
    openCell (cell) {
      // ★ 2026-10-01 越界「空地」格不可点（地图外没有目标详情）
      if (this.isMapOOB(cell)) {
        this.notify('该坐标在地图外（空地），没有目标')
        return
      }
      this.selCell = cell
      this.selDetail = null
      this.warText = ''
      this.warStatus = 0
      this.cur = 'wildview'
      // ★ 2026-09-30 进入详情刷新收藏状态：避免用旧地图星标的旧数据导致「没收藏却显示已收藏」
      this.loadStars()
      if (cell.area_type === 3) {
        if (!cell.mine && cell.user_id) this.checkWar()
        return
      }
      const ttype = cell.area_type === 2 ? 3 : (cell.terrain === 8 ? 2 : 1)
      // 顺带记住这块地上「我的野地记录 id」，采集/派遣下单要用
      const mine = (this.wildlands || []).find(x => x.x === cell.x && x.y === cell.y)
      if (mine) cell.wild_id = mine.id
      api.get('/games/ezfy/map/wildland?x=' + cell.x + '&y=' + cell.y + '&type=' + ttype).then(r => {
        if (r.code === 0) this.selDetail = r.data
      })
    },
    // 详情页里选命令 → 进出征页(复刻 mapView 的 [侦查][掠夺][征服])
    pickOrder (t) {
      // ★ 2026-09-28 用户规则：自己的附属野地不能侦查/掠夺/征服 ——
      //   要先到「附属野地」页把这块地[放弃]（放弃后该坐标恢复为中立野地，才能再打）。
      //   按钮已灰掉，这里再兜一层（防老页面缓存/键盘操作绕过）；后端 createOrder 有同样校验。
      if ((t === 1 || t === 2 || t === 3) && this.isOwnWild) {
        this.notify('这是你自己的附属野地, 不能' + (this.orderNames[t] || '') +
          '; 请先在「附属野地」里[放弃]该野地')
        return
      }
      this.orderType = t
      this.orderCalc = null
      this.go('orderpre')
    },
    orderAvail (t) {
      if (t === 4 || t === 7) return false
      if ((t === 5 || t === 6) && this.selCell && this.selCell.area_type !== 3) return false
      return true
    },
    declareWar () {
      if (!this.selCell || !this.selCell.city_id) return
      // ★ 用户规则「同盟玩家不能宣战」：按钮已经藏了，这里再兜一层，防止老页面缓存绕过
      if (this.selCell.ally) { this.notify('同盟成员之间不能宣战'); return }
      api.post('/games/ezfy/war/declare', { city_id: this.selCell.city_id }).then(r => {
        this.alert(r, '宣战成功')
        this.checkWar()
      })
    },
    // 未宣战/宣战未生效时点掠夺/征服 → 只提示，不进出征页
    warBlock (name) {
      if (!this.selCell || !this.selCell.user_id) return
      if (this.warStatus === 1) {
        this.alert({ code: 1, msg: (this.warText || '宣战尚未生效') + '，生效后方可' + name }, '宣战尚未生效')
      } else {
        this.alert({ code: 1, msg: '需先对 ' + (this.selCell.owner || '对方') + ' 宣战，宣战 24 小时后生效，生效后方可' + name }, '需先宣战')
      }
    },
    checkWar () {
      if (!this.selCell || !this.selCell.user_id) return
      api.get('/games/ezfy/war/status?target_user_id=' + this.selCell.user_id).then(r => {
        if (r.code === 0) {
          this.warText = r.data.text
          this.warStatus = r.data.status || 0
          // ★ 管理端「宣战功能」开关：关掉时不需要宣战，掠夺/征服直接可点
          //   （后端 isAtWar 同时恒为 true，两边口径一致）。
          //   注意不能靠把 warStatus 伪造成 2 —— 那样同盟城市的 运输/增援 会被误判而消失。
          this.warRequire = r.data.war_require !== false
          // ★ 2026-09-25 用户要求：接军团交战期字段
          //   at_war = 可掠夺/征服（个人宣战已生效 或 管理端关掉宣战开关 或 军团交战期）；
          //   corps_war = {active, corps_name, text}，active 时详情页显示绿色提示。
          this.atWar = !!r.data.at_war
          this.corpsWar = r.data.corps_war || null
        }
      })
    },
    // ★★ 2026-09-28 用户要求：「[最大] 按钮点的不是当前兵种最大兵力，应该是出征还剩多少的最大」，
    //   且「滑动滚轮也加下剩余可出征兵种最大卡控」。
    //
    //   本兵种可填上限 = min(城内现有, 出征上限剩余)：
    //     - 城内现有   = troopCount(id)（不能派出城里没有的兵）
    //     - 上限剩余   = 出征上限 − **其它**兵种已填合计（把本兵种算作 0 时的剩余额度）
    //
    //   这样「每个兵种都拉满」正好凑满上限，总量天然不会超 —— 不需要事后校正。
    //   例：上限 10000，A 已填 3000、B 已填 2000 → C 的上限 = 10000−5000 = 5000；
    //       A 自己的上限 = 10000−2000 = 8000（再受城内现有封顶）。
    //
    //   下列情况只按「城内现有」卡（返回 troopCount）：
    //     - 还没点过 [计算]（orderCalc 为空，上限未知）→ 不能凭空编一个上限；
    //     - 管理端把「出征上限」开关关了（cap_unlimited）→ 本来就不限；
    //     - 运输(5)/派遣(8) → 后端不校验兵力上限（见 orderCapApplies）。
    orderQtyMax (id) {
      const own = this.troopCount(id)
      const c = this.orderCalc
      // 运输(5)/派遣(8) 无出征上限 → 仍按「城内现有」卡
      if (!this.orderCapApplies) return own
      // 上司关了上限开关 → 不限
      if (c && c.cap_unlimited) return own
      // ★ 攻略 2026-09-28：还没拿到出征上限(troop_cap)时，不能退回「城内总数」(own)，
      //   否则 orderCalc null 的首帧滑块会放开到整个城内量(如 98 万航母)而不是上限(如 12.9 万)。
      //   直接返回 0 → 滑块在计算完成前保持禁用，等 doCalc 落地后自动放开到正确上限。
      if (!c) return 0
      const remain = (c.troop_cap || 0) - (this.orderTroopTotal - this.orderQty(id))
      // remain 可能为负（其它兵种已经把额度吃超了）→ 本兵种只能填 0
      return Math.max(0, Math.min(own, remain))
    },
    // ★ 上限变小后（改集结令 / 换带队军官 / 换城市）把已填兵力重新夹进新上限，
    //   否则「本次出兵」会一直红着超限，而滑块又因为 max 变成 0 和数字框显示不一致。
    //   策略：按兵种 id 升序依次分配剩余额度（先到先得），结果稳定可预期。
    clampOrderTroops () {
      const c = this.orderCalc
      if (!c || c.cap_unlimited || !this.orderCapApplies) return
      if (this.orderTroopTotal <= (c.troop_cap || 0)) return
      let left = c.troop_cap || 0
      const ids = Object.keys(this.orderTroops)
        .filter(k => this.orderQty(k) > 0)
        .sort((a, b) => Number(a) - Number(b))
      for (const id of ids) {
        const give = Math.max(0, Math.min(this.orderQty(id), left))
        left -= give
        this.$set(this.orderTroops, id, give)
      }
    },
    // 当前该兵种已填的出征数量（没填过 = 0）；滑块与数字框都绑它，保证两边显示一致
    orderQty (id) {
      const v = this.orderTroops[id]
      if (v === undefined || v === null || v === '') return 0
      const n = parseInt(v, 10)
      return isNaN(n) || n < 0 ? 0 : n
    },
    // ★ 2026-09-25 出征页「兵种数量搭配」：滑块 + 数字框 **双向联动**
    //   - 拖动滑块 → 数字框跟着变；填数字 → 滑块跟着走；两边共用 orderTroops[id] 一个值。
    //   - 用 $set 写对象键：orderTroops 初始是 {}，直接赋值新键 Vue2 侦测不到，
    //     滑块动完数字框不会刷新（这就是「联动」失效的原因）。
    //   - ★ 2026-09-28 夹紧上限由「城内现有」改成 orderQtyMax(id)
    //     = min(城内现有, 出征上限剩余) —— 用户要求滑块也要按「剩余可出征」卡控。
    //   - 夹紧后若数值没变（如本来已是上限又填了更大的数），Vue 不会重渲染，
    //     DOM 里会留着用户填的非法数字 → 这里手动把输入框内容回写，保证「看到的 = 提交的」。
    onOrderQtyInput (id, ev) {
      const max = this.orderQtyMax(id)
      let n = parseInt(ev.target.value, 10)
      if (isNaN(n) || n < 0) n = 0
      if (n > max) n = max
      this.$set(this.orderTroops, id, n)
      if (ev && ev.target && String(n) !== String(ev.target.value)) {
        ev.target.value = String(n)
      }
    },
    // [最大] = 一键带上「该兵种还能派出的最大数量」（滑到最后、数字框同步）
    //   ★ 2026-09-28 用户纠正：「不是当前兵种最大兵力，应该是出征还剩多少的最大」。
    //   所以取 orderQtyMax = min(城内现有, 出征上限剩余)，而不是城内现有。
    setOrderQtyMax (id) {
      const max = this.orderQtyMax(id)
      if (max <= 0) return
      this.$set(this.orderTroops, id, max)
    },
    // 随军资源：当前已填数值（任何一行都校验成非负整数）
    resQty (key) {
      const v = this.resVal(key)
      if (v === undefined || v === null || v === '') return 0
      const n = parseInt(v, 10)
      return isNaN(n) || n < 0 ? 0 : n
    },
    // ★ 随军资源：单行上限 = min(城内现有, 可用负重 - 其它行重量)
    //   ★ 修正 2026-09-28：(cap - others) 是【绝对上限】，不能再加 cur（旧写法 = cur + (cap - others)，
    //     满负重时反而 allowed=2*cur>cap 还能往右滑）。用绝对上限后：
    //     满负重时 allowed = 本行当前值 → 只能左收不能右加，其它行同样被卡住 —— 实现「负重满了就滑不动」。
    //   可用负重 = 负重 - 行军油耗（orderResUsableCap）。
    resQtyMax (key) {
      if (this.orderResDisabled) return 0
      const avail = this.resAvail(key)
      const others = this.orderResTotal - this.resQty(key)   // 其它行当前重量
      const allowed = this.orderResUsableCap - others          // 本行绝对上限（扣掉别行占用）
      let mx = Math.min(avail, allowed)
      if (mx < 0) mx = 0
      return Math.floor(mx)
    },
    // 该资源行单个输入：夹到 [0, resQtyMax]，并双向联动（滑块/数字框共用 trXxx 一个值）
    onResInput (key, ev) {
      const max = this.resQtyMax(key)
      let n = parseInt(ev.target.value, 10)
      if (isNaN(n) || n < 0) n = 0
      if (n > max) n = max
      this.setResVal(key, n)
      if (ev && ev.target && String(n) !== String(ev.target.value)) {
        ev.target.value = String(n)
      }
    },
    // [最大]：这行一次拖到「当前负重剩余还能塞下的最大值」（仍不超城内现有）
    setResMax (key) {
      const max = this.resQtyMax(key)
      if (max <= 0) return
      this.setResVal(key, max)
    },
    // 取/写随军资源五项（gold/food/steel/oil/rare → trXxx）
    resVal (key) {
      if (key === 'gold') return this.trGold
      if (key === 'food') return this.trFood
      if (key === 'steel') return this.trSteel
      if (key === 'oil') return this.trOil
      return this.trRare
    },
    setResVal (key, n) {
      if (key === 'gold') this.trGold = n
      else if (key === 'food') this.trFood = n
      else if (key === 'steel') this.trSteel = n
      else if (key === 'oil') this.trOil = n
      else this.trRare = n
    },
    // 城内现有该资源数量
    resAvail (key) {
      const c = this.city || {}
      if (key === 'gold') return c.gold || 0
      if (key === 'food') return c.food || 0
      if (key === 'steel') return c.steel || 0
      if (key === 'oil') return c.oil || 0
      return c.rare || 0
    },
    // 出征表单 → 请求体(部队/资源/军官/宿营)
    orderBody () {
      const body = {
        order_type: this.orderType,
        target_type: this.selCell.area_type === 3 ? 3 : (this.selCell.area_type === 2 ? 2 : 1),
        target_x: this.selCell.x,
        target_y: this.selCell.y,
        wait_min: (parseInt(this.waitH) || 0) * 60 + (parseInt(this.waitM) || 0)
      }
      if (this.selCell.area_type === 3 && this.selCell.city_id) {
        body.target_id = this.selCell.city_id
      }
      // ★ 采集(4)/派遣(7) 后端要按 target_id 校验「这块野地是你占领的」，
      //   而地图格子只有坐标 → 这里按坐标在「我的野地」里反查记录 id。
      //   （之前没带 target_id，采集/派遣永远发不出去）
      if (this.orderType === 4 || this.orderType === 7) {
        if (this.selCell.wild_id) {
          body.target_id = this.selCell.wild_id
        } else {
          const w = (this.wildlands || []).find(x => x.x === this.selCell.x && x.y === this.selCell.y)
          if (w) body.target_id = w.id
        }
      }
      const troops = []
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k]) || 0
        if (n > 0) troops.push({ troopId: parseInt(k), count: n })
      }
      body.troops = troops
      body.resources = {
        food: parseInt(this.trFood) || 0, steel: parseInt(this.trSteel) || 0,
        oil: parseInt(this.trOil) || 0, rare: parseInt(this.trRare) || 0,
        gold: parseInt(this.trGold) || 0
      }
      if (this.orderOfficer && this.orderOfficer !== '0') body.officer = this.orderOfficer
      // ★ 集结令个数：每个 +10 万出征上限，单次最多 10 个
      body.gather = parseInt(this.orderGather) || 0
      return body
    },
    // 复刻原版出征页的 [计算]: 预览油耗/负重/耗时, 不下达命令
    doCalc () {
      if (!this.selCell) return
      api.post('/games/ezfy/order/preview', this.orderBody()).then(r => {
        if (r.code === 0) {
          this.orderCalc = r.data
          // ★ 2026-09-28：上限可能因「集结令 / 带队军官 / 城市」变化而变小，
          //   这里立刻把已填兵力夹回新上限内，保证滑块 max 与数字框始终一致、不会红着超限。
          this.clampOrderTroops()
        } else this.alert(r, '计算失败')
      })
    },
    // ★ 清空出征表单（用户反馈「出征还有上次留的数据」）
    //   每次进出征页都重置，避免上次的兵力/资源被误当成这次的出征内容。
    resetOrderForm () {
      this.orderTroops = {}
      this.orderOfficer = '0'
      this.orderGather = 0
      this.orderCalc = null
      this.trFood = 0
      this.trSteel = 0
      this.trOil = 0
      this.trRare = 0
      this.trGold = 0
      this.waitH = 0
      this.waitM = 0
      this.presetSel = 0
    },
    // ---- 预设编队 ----
    // 拉取我的预设列表（出征页下拉 + 司令部预设 tab 共用）
    loadPresets () {
      api.get('/games/ezfy/presets').then(r => {
        if (r.code === 0) this.presets = r.data.presets || []
      })
    },
    // ★ 预设页 [计算]：与 doCalc 的区别——预设不含目标，传本城坐标按 0 距离估算油耗/负重，
    //   出兵上限（troop_cap）与集结令/军官加成照常生效，返回后照旧把已填兵力夹回新上限。
    presetCalc () {
      // ★ 2026-09-29 预设编队永远按「出征(掠夺)」口径预览：强制 orderType=2，
      //   否则全局 orderType 若残留为运输(5)/派遣(8)（无出征上限），滑块上限会被放成城内总数
      this.orderType = 2
      const c = this.city
      if (!c || !c.x || !c.y) return
      api.post('/games/ezfy/order/preview', {
        // 与 orderBody() 同口径：不传 city_id（后端按主城/当前城结算）
        order_type: 2,
        target_x: c.x,
        target_y: c.y,
        troops: this.presetTroopGroups(),
        resources: {},
        officer: this.orderOfficer && this.orderOfficer !== '0' ? this.orderOfficer : '',
        wait_min: 0,
        gather: parseInt(this.orderGather) || 0
      }).then(r => {
        if (r.code === 0) {
          this.orderCalc = r.data
          this.clampOrderTroops()
        } else this.alert(r, '计算失败')
      })
    },
    // 预设页/出征页提交用的兵力数组（orderTroops → [{troopId,count}]，只带 >0 的）
    presetTroopGroups () {
      const out = []
      for (const k in this.orderTroops) {
        const n = parseInt(this.orderTroops[k]) || 0
        if (n > 0) out.push({ troopId: parseInt(k), count: n })
      }
      return out
    },
    // 预设页集结令改动：夹到 [0, gatherMax] 后按 0 距离重算
    onPresetGatherChange () {
      let n = parseInt(this.orderGather) || 0
      if (isNaN(n) || n < 0) n = 0
      const cap = this.gatherMax
      if (n > cap) n = cap
      this.orderGather = n
      this.presetCalc()
    },
    startPresetAdd () {
      // 进入新增表单：清空共享表单状态，从当前城现有/军官起步
      this.resetOrderForm()
      this.orderType = 2   // ★ 2026-09-29 预设模板始终按「出征(掠夺)」口径，别继承运输/派遣的无上限状态
      this.presetName = ''
      this.presetAdding = true
      // ★ 2026-09-29 立即按 0 距离算一次：让「本次出兵 / 上限」立刻显示准确值（如 0 / 99000），不用等玩家点 [计算]
      this.$nextTick(() => this.presetCalc())
    },
    cancelPresetAdd () {
      this.presetAdding = false
      this.resetOrderForm()
    },
    // 保存预设：名称校验；军官 '0' → 空串；兵力 = 当前表单搭配
    savePreset () {
      const name = (this.presetName || '').trim()
      if (!name) { this.alert({ code: 400, msg: '请填写预设名称' }); return }
      if (name.length > 20) { this.alert({ code: 400, msg: '预设名称最多20字' }); return }
      api.post('/games/ezfy/presets', {
        name,
        officer: this.orderOfficer && this.orderOfficer !== '0' ? this.orderOfficer : '',
        gather: parseInt(this.orderGather) || 0,
        troops: this.presetTroopGroups()
      }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.presetAdding = false
          this.presetName = ''
          this.resetOrderForm()
          this.loadPresets()
        } else this.alert(r, '保存失败')
      })
    },
    deletePreset (p) {
      api.post('/games/ezfy/presets/delete', { id: p.id }).then(r => {
        if (r.code === 0) {
          if (this.presetSel === p.id) this.presetSel = 0
          this.loadPresets()
        } else this.alert(r, '删除失败')
      })
    },
    // ★ 出征页选中预设 → 回填：
    //   军官不在当前城可带队列表 → 回填空；兵力夹 min(预设, orderQtyMax)=可出征上限；
    //   集结令夹 [0, gatherMax]；最后按真实目标 doCalc 重算（异步，clampOrderTroops 兜底夹回上限）。
    applyPreset () {
      const p = this.presets.find(x => x.id === this.presetSel)
      if (!p) return
      // ① 军官：不在 onDutyOfficers → 回填空
      const has = p.officer && this.onDutyOfficers.some(o => o.name === p.officer)
      this.orderOfficer = has ? p.officer : '0'
      // ③ 兵力：逐个夹到当前可出征上限（兵力不够时只能填最大）
      this.orderTroops = {}
      for (const t of (p.troops || [])) {
        const max = this.orderQtyMax(t.troopId)
        if (max > 0) this.$set(this.orderTroops, t.troopId, Math.min(t.count, max))
      }
      // ② 集结令：夹到当前可用上限
      let g = parseInt(p.gather) || 0
      if (isNaN(g) || g < 0) g = 0
      const cap = this.gatherMax
      this.orderGather = g > cap ? cap : g
      // ⑥ 用真实目标重算（异步；内部会把已填兵力夹回新上限）
      this.doCalc()
    },
    // 改集结令数量后立刻重算，让「本次出兵 / 上限」即时刷新
    // ★ 手填数字：夹到 [0, 可用上限]，可用上限 = min(管理端配置, 背包持有量)。
    //   注意上限取自 orderCapMax（跟着 /view 下发），不再在没数据时硬夹 50。
    onGatherChange () {
      let n = parseInt(this.orderGather) || 0
      if (isNaN(n) || n < 0) n = 0
      const cap = this.gatherMax
      if (n > cap) n = cap
      this.orderGather = n
      this.doCalc()
    },
    doOrder () {
      if (!this.selCell) return
      // ★ 用户反馈「连点会出现多条」→ 防抖：一次点击只下达一条出征命令
      this.once('order', () => api.post('/games/ezfy/order', this.orderBody()).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.orderTroops = {}
          this.orderOfficer = '0'
          this.orderCalc = null
          this.trFood = 0; this.trSteel = 0; this.trOil = 0; this.trRare = 0; this.trGold = 0
          this.waitH = 0; this.waitM = 0
          this.orderGather = 0
          this.load()
          this.loadBag()
          // ★★ 2026-09-26 修复「点出征后跳转出征队列是空的」：
          //   出征队列页（cur==='orders'）的数据源是 `queueItems` → `this.dynamics`，
          //   由 `loadDynamics()` 填充；而这里原来直接 `cur='orders'` + `loadOrders()`
          //   （loadOrders 填的是另一份数据，orders 页根本不读它）→ 页面永远空。
          //   改成走 `go('orders')`，它内部就是 `loadDynamics()`，以后再加数据源也不会漏。
          this.go('orders')
        } else this.notify(r.msg)
      }))
    },
    loadOnDutyOfficers () {
      api.get('/games/ezfy/officers/onduty').then(r => {
        if (r.code === 0) this.onDutyOfficers = r.data.officers || []
      })
    },
    // 本城军官全量列表（含出征中/俘虏）：只用于出征页的「为什么没有可带队军官」提示
    loadCityOfficers () {
      api.get('/games/ezfy/officers').then(r => {
        if (r.code === 0) this.cityOfficers = r.data.officers || []
      })
    },
    // ---- 联络中心 ----
    loadLiaison () {
      api.get('/games/ezfy/liaison').then(r => {
        if (r.code === 0) this.liaison = r.data
      })
    },
    // ---- 仓库保护 ----
    loadWare () {
      api.get('/games/ezfy/city/warehouse').then(r => {
        if (r.code === 0) {
          this.ware = r.data
          const ratio = {}
          for (const it of r.data.res) ratio[it.key] = it.ratio
          this.wareRatio = ratio
        }
      })
    },
    doWareSet () {
      if (this.wareSum > 100) { this.notify('四项比例合计不能超过100%'); return }
      api.post('/games/ezfy/city/warehouse', {
        food: parseInt(this.wareRatio.food) || 0,
        steel: parseInt(this.wareRatio.steel) || 0,
        oil: parseInt(this.wareRatio.oil) || 0,
        rare: parseInt(this.wareRatio.rare) || 0
      }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadWare()
        } else this.notify(r.msg)
      })
    },
    // ---- 聊天/邮箱 ----
    doChatSend () {
      const msg = (this.chatMsg || '').trim()
      if (!msg) return
      if (this.chatCooldown > 0) {
        this.notify('发言冷却中，还需 ' + this.chatCooldown + ' 秒')
        return
      }
      api.post('/games/ezfy/chat', { content: msg, channel: this.chatChannel }).then(r => {
        if (r.code === 0) {
          this.chatMsg = ''
          // 复刻原版聊天: 每次发言 30 秒冷却
          this.startChatCooldown(30)
          // ★ 降序展示：新消息在第 1 页最上面，发完回到第 1 页
          this.chatPage = 1
          this.loadChats()
          // ★ 用户反馈「世界聊天后回首页发现没出来我刚发的」：
          //   一并刷新首页聊天预览，避免回首页还要 Ctrl+F5。
          this.loadHomeChats()
        } else this.notify(r.msg)
      })
    },
    startChatCooldown (sec) {
      this.chatCooldown = sec
      if (this.chatTimer) clearInterval(this.chatTimer)
      this.chatTimer = setInterval(() => {
        this.chatCooldown -= 1
        if (this.chatCooldown <= 0) {
          this.chatCooldown = 0
          clearInterval(this.chatTimer)
          this.chatTimer = null
        }
      }, 1000)
    },
    // 聊天/好友里的玩家名 → 家园个人主页
    // 昵称逐字颜色: color 可能是逗号分隔的多色序列(如 "#f00,#0f0")
    nickChars (name) {
      return [...String(name || '')]
    },
    nickColorAt (color, i) {
      if (!color) return {}
      const cs = String(color).split(',').map(x => x.trim()).filter(Boolean)
      if (!cs.length) return {}
      return { color: cs[i % cs.length] }
    },
    // 查看他人统帅信息(复刻 PlayerController.infoOther)
    // 入口: 首页聊天 / 聊天频道 / 邮箱发件人 / 好友 / 军团成员 / 军团聊天 / 排行 —— 点玩家名
    // ★ 游戏是沉浸式的: 点玩家名只能看「二战风云」的统帅信息,
    //   **不要**跳到家园站点的个人主页(/user/:id), 那样就跳出游戏了。
    openPlayer (uid) {
      if (!uid) return
      this.playerInfoBack = this.cur === 'playerinfo' ? this.playerInfoBack : this.cur
      this.playerInfo = null
      this.go('playerinfo')
      this.loadPlayerInfo(uid)
    },
    loadPlayerInfo (uid) {
      api.get('/games/ezfy/player/' + uid).then(r => {
        if (r.code === 0) this.playerInfo = r.data
        else this.alert(r, '获取统帅信息失败')
      })
    },
    async doAddFriendById () {
      if (!this.playerInfo) return
      const name = this.playerInfo.nickname
      const remark = await this.ask('给 ' + name + ' 的验证信息（可留空）', { input: true, placeholder: '可留空' })
      if (remark === null) return
      api.post('/games/ezfy/friends/apply', { target_id: this.playerInfo.user_id, remark: remark }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '已发送好友申请')
          this.loadPlayerInfo(this.playerInfo.user_id)
        } else {
          this.notify(r.msg || '申请失败')
        }
      })
    },
    // 军团邮件群发(复刻 CorpsController.mail, 仅军团长)
    doCorpsMail () {
      const c = (this.corpsMailContent || '').trim()
      if (!c) { this.notify('请填写邮件内容'); return }
      api.post('/games/ezfy/corps/mail', { content: c }).then(r => {
        this.alert(r, '军团邮件已群发')
        if (r.code === 0) this.corpsMailContent = ''
      })
    },
    openNotice (n) {
      this.curNotice = n
      this.cur = 'notices'
    },
    // ---- 军团 ----
    doCreateCorps () {
      api.post('/games/ezfy/corps/create', { name: this.corpsName })
        .then(r => this.alert(r, '联盟已创建', () => this.loadCorps()))
    },
    doJoinCorps (cp) {
      // ★ 2026-09-30 入口改用 /corps/apply：open 军团直接入团，需审核军团落申请待审核
      api.post('/games/ezfy/corps/apply', { corps_id: cp.id })
        .then(r => {
          this.alert(r, (r.data && r.data.need_review ? '申请已提交' : '已加入联盟'), () => {
            if (r.data && r.data.need_review) this.myApplyStatus = 1
            this.loadCorps()
          })
        })
    },
    async doLeaveCorps () {
      if (!await this.ask(this.isLeader ? '军团长退出将解散军团, 确定?' : '确定退出军团?')) return
      api.post('/games/ezfy/corps/leave', {})
        .then(r => this.alert(r, '已退出联盟', () => this.loadCorps()))
    },
    async openNoticeEdit () {
      const n = await this.ask('输入军团公告', { input: true, value: (this.myCorps ? this.myCorps.notice : '') })
      if (n !== null) api.post('/games/ezfy/corps/notice', { notice: n })
        .then(r => this.alert(r, '公告已更新', () => this.loadCorps()))
    },
    doKick () {
      if (!this.kickUserId) return this.notify('请选择成员')
      api.post('/games/ezfy/corps/kick', { user_id: this.kickUserId })
        .then(r => this.alert(r, '已踢出', () => this.loadCorps()))
    },
    // ★ 军团长任命副团长/参谋长（title 传空 = 撤职）
    async doSetCorpsTitle (m, title) {
      const label = title || '普通成员'
      const ok = await this.ask('确定把「' + m.name + '」任命为「' + label + '」吗？' +
        (title === '副团长' ? '\n副团长可以群发军团邮件。' : ''))
      if (!ok) return
      api.post('/games/ezfy/corps/member/title', { user_id: m.user_id, title: title })
        .then(r => this.alert(r, '已任命为' + label, () => this.loadCorps()))
    },
    doCorpsChat () {
      api.post('/games/ezfy/corps/chat', { content: this.corpsMsg }).then(r => {
        if (r.code === 0) {
          this.corpsMsg = ''
          this.loadCorps()
        } else this.notify(r.msg)
      })
    },
    // ★ 2026-09-30 入团审核：军团长查看待审申请
    loadCorpsApplies () {
      api.get('/games/ezfy/corps/apply').then(r => {
        if (r.code === 0) {
          this.corpsApplies = r.data.applies || []
          this.myCorpsNeedReview = r.data.need_review ? 1 : 0
        }
      })
    },
    // op 1=通过 2=拒绝
    doApplyHandle (a, op) {
      api.post('/games/ezfy/corps/apply/handle', { apply_id: a.apply_id, op: op })
        .then(r => this.alert(r, op === 1 ? '已通过' : '已拒绝', () => this.loadCorpsApplies()))
    },
    // 军团长开/关入团审核
    doToggleNeedReview () {
      api.post('/games/ezfy/corps/need-review', { need_review: this.myCorpsNeedReview ? 0 : 1 })
        .then(r => this.alert(r, '已切换审核开关', () => {
          this.loadCorps()
        }))
    },
    // ---- ★ 2026-09-25 用户要求：军团外交 / 军团宣战 / 军团商城 ----
    // 切子栏：只在数据还没拉过时才请求（避免每次切 tab 重复打接口）
    switchCorpsTab (tab) {
      this.corpsTab = tab
      if (tab === 'info') {
        this.loadCorps()
        if (this.isLeader) this.loadCorpsApplies()
      } else if (tab === 'list' || tab === 'chat') {
        this.loadCorps()
      } else if (tab === 'diplomacy') {
        if (!this.corpsRelations) this.loadCorpsRelations()
      } else if (tab === 'war') {
        if (!this.corpsWars) this.loadCorpsWars()
        // 宣战 tab 复用外交的军团列表（[宣战] 入口），没拉过就补一次
        if (!this.corpsRelations) this.loadCorpsRelations()
      } else if (tab === 'mall') {
        if (!this.corpsMall.loaded) this.loadCorpsMall()
      }
    },
    loadCorpsRelations () {
      return api.get('/games/ezfy/corps/relations').then(r => {
        if (r.code === 0) {
          this.corpsRelations = r.data
          // ★ 外交数据也带 my_corps.points，顺手同步军团总积分（与 /corps/members 同源）
          if (r.data && r.data.my_corps) this.corpsPoints = r.data.my_corps.points || 0
        }
      })
    },
    loadCorpsWars () {
      return api.get('/games/ezfy/corps/war').then(r => {
        if (r.code === 0) this.corpsWars = r.data
      })
    },
    loadCorpsMall () {
      return api.get('/games/ezfy/corps/mall').then(r => {
        if (r.code === 0) {
          this.corpsMall = {
            loaded: true,
            in_corps: !!r.data.in_corps,
            my_points: r.data.my_points || 0,
            corps_points: r.data.corps_points || 0,
            items: r.data.items || [],
            my_bought: r.data.my_bought || {}
          }
        }
      })
    },
    // 某商品我已购数量（my_bought 以商品 id 为键）
    corpsBought (id) {
      const mb = this.corpsMall.my_bought || {}
      return parseInt(mb[id]) || 0
    },
    // [兑换] 是否可点：未售罄 且 未超限购
    // ★ stock 是「总库存」(-1=无限)，剩余要减掉已售 sold
    corpsMallCanBuy (item) {
      if (item.stock >= 0 && (item.stock - item.sold) <= 0) return false
      const bought = this.corpsBought(item.id)
      if (item.limit > 0 && bought >= item.limit) return false
      return true
    },
    // 军团名查找（外交/宣战操作的确认文案用）
    corpsNameOf (corpsId) {
      const list = (this.corpsRelations && this.corpsRelations.corps_list) || []
      const hit = list.filter(c => c.id === corpsId)[0]
      return hit ? hit.name : ('#' + corpsId)
    },
    // 军团长标记关系：type 0取消 / 1友好 / 2敌对（后端也会校验权限）
    async setCorpsRelation (corpsId, type) {
      const name = this.corpsNameOf(corpsId)
      const tip = type === 0 ? ('确定取消对「' + name + '」的关系标记吗？')
        : ('确定把「' + name + '」标记为' + (type === 1 ? '友好' : '敌对') + '吗？\n（友好/敌对军团均可宣战）')
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/corps/relation', { corps_id: corpsId, type: type }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg || '关系已更新')
          this.loadCorpsRelations()
        } else this.notify(r.msg || '操作失败')
      })
    },
    // 军团长对某军团宣战（后端校验权限/是否已在宣战中）
    async declareCorpsWar (corpsId) {
      const name = this.corpsNameOf(corpsId)
      const ok = await this.ask('确定对「' + name + '」宣战吗？\n宣战后 12 小时生效，48 小时后整场结束；生效期间双方成员可互相掠夺/征服。')
      if (!ok) return
      api.post('/games/ezfy/corps/war/declare', { corps_id: corpsId }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg || '宣战成功')
          this.loadCorpsWars()
          this.loadCorpsRelations()
        } else this.notify(r.msg || '宣战失败')
      })
    },
    // 军团商城兑换：数量用内联输入弹窗询问（参考 openNoticeEdit 的 ask({input:true}) 写法）
    async doCorpsMallBuy (item) {
      const bought = this.corpsBought(item.id)
      const byLimit = item.limit > 0 ? (item.limit - bought) : 9999
      const byStock = item.stock < 0 ? 9999 : (item.stock - item.sold)
      const byPoint = item.price > 0 ? Math.floor((this.corpsMall.my_points || 0) / item.price) : 9999
      const maxN = Math.max(0, Math.min(byLimit, byStock, byPoint))
      if (maxN <= 0) {
        if (byPoint <= 0) this.notify('军团积分不足：兑换 1 个需要 ' + item.price + ' 积分，我现有 ' + this.corpsMall.my_points)
        else if (byStock <= 0) this.notify('该商品已售罄')
        else this.notify('已达限购上限（' + item.limit + '）')
        return
      }
      const input = await this.ask('兑换「' + item.name + '」数量（1-' + maxN + '，单价 ' + item.price +
        ' 积分，我现有 ' + this.corpsMall.my_points + '）', { input: true, value: '1' })
      if (input === null) return
      const n = parseInt(input, 10)
      if (isNaN(n) || n < 1 || n > maxN) { this.notify('数量需在 1-' + maxN + ' 之间'); return }
      api.post('/games/ezfy/corps/mall/buy', { id: item.id, count: n }).then(r => {
        if (r.code === 0) {
          this.notify(r.msg || '兑换成功')
          this.loadCorpsMall()
          this.load()
        } else this.notify(r.msg || '兑换失败')
      })
    },
    // ---- 商城/背包/交易 ----
    // ★ 第九轮：分类切换 / 翻页
    setMallCat (c) {
      this.mallCat = c
      this.mallPage = 1
    },
    mallGo (d) {
      const p = this.mallPage + d
      if (p >= 1 && p <= this.mallTotalPages) this.mallPage = p
    },
    // ★ 通用翻页（检索条件变了要把页码归 1，见各处的 setXxxWord）
    pagerGo (d, cur, total) {
      const p = cur + d
      return (p >= 1 && p <= total) ? p : cur
    },
    bagGo (d) { this.bagPage = this.pagerGo(d, this.bagPage, this.bagTotalPages) },
    equipGo (d) { this.equipPage = this.pagerGo(d, this.equipPage, this.equipTotalPages) },
    equipAllGo (d) { this.equipAllPage = this.pagerGo(d, this.equipAllPage, this.equipAllTotalPages) },
    officerBagGo (d) {
      this.officerBagPage = this.pagerGo(d, this.officerBagPage, this.officerBagTotalPages)
    },
    openBuy (it) {
      this.buyItem = it
      this.buyCount = 1
      // ★ 双渠道道具默认用黄金（多数玩家手上黄金比钻石多）
      this.buyPayWith = it.dual_pay ? 'gold' : (it.is_diamond ? 'diamond' : 'gold')
      // ★ 跳转到独立购买详情页确认（不再行内展开）
      this.cur = 'mallbuy'
    },
    // ★ 单次可买上限 = min(管理端配置的单次上限, 库存)。
    //   无限库存(-1)的道具只看配置值。原来这里写死 999，和 doBuy 里的 99 打架。
    buyMaxOf (it) {
      const cfgMax = parseInt(this.mallBuyMax) > 0 ? parseInt(this.mallBuyMax) : 9999
      if (it.unlimited) return cfgMax
      const st = parseInt(it.stock) || 0
      return Math.max(1, Math.min(cfgMax, st))
    },
    doBuy (it) {
      const n = parseInt(this.buyCount) || 0
      const maxBuy = this.buyMaxOf(it)
      if (n < 1 || n > maxBuy) { this.notify('数量需在 1-' + maxBuy + ' 之间'); return }
      if (!it.unlimited && it.stock !== undefined && n > it.stock) {
        this.notify(it.stock > 0 ? ('库存不足，最多买 ' + it.stock + ' 个') : '该道具已售罄')
        return
      }
      // ★ 支付方式：双渠道道具由玩家选；单渠道按道具属性定
      const payWith = it.dual_pay ? (this.buyPayWith || 'gold') : (it.is_diamond ? 'diamond' : 'gold')
      if (payWith === 'diamond') {
        const cost = (it.price_diamond || 0) * n
        if (cost > this.mallDiamond) {
          this.notify('钻石不足：需要 ' + cost + ' 钻石，当前余额 ' + this.mallDiamond +
            (it.dual_pay ? '（可改用黄金购买）' : '（钻石仅可由管理员充值）'))
          return
        }
      } else {
        const cost = (it.price_gold || 0) * n
        if (cost > (this.city.gold || 0)) {
          this.notify('黄金不足：需要 ' + cost + '，当前 ' + (this.city.gold || 0))
          return
        }
      }
      api.post('/games/ezfy/mall/buy', { cfg_id: it.id, count: n, pay_with: payWith }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '购买成功')
          this.buyItem = null
          this.load()
          this.loadBag()
          // ★ 购买成功后返回商城（go('mall') 会自动刷新商城数据）
          this.go('mall')
        } else this.notify(r.msg || '购买失败')
      })
    },
    needOfficer (it) {
      // ★ 19 = 星级徽章，也要选军官（漏了它会没有「军官:」下拉，玩家没法用）
      return it.item_type === 10 || it.item_type === 11 || it.item_type === 12 || it.item_type === 19
    },
    openUse (it) {
      this.useItem = it
      this.useCount = 1
      this.useOfficerId = 0
      this.useSkillId = 0
    },
    doUse (it) {
      const body = { cfg_id: it.cfg_id, count: parseInt(this.useCount) || 1, city_id: this.city.id }
      if (this.needOfficer(it)) {
        if (!this.useOfficerId) { this.notify('请先选择要使用的军官'); return }
        body.officer_id = this.useOfficerId
      }
      if (it.item_type === 11) {
        if (!this.useSkillId) { this.notify('请选择要学习的技能'); return }
        body.skill_id = this.useSkillId
      }
      api.post('/games/ezfy/bag/use', body).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '使用成功')
          this.useItem = null
          this.loadBag()
          this.load()
        } else this.notify(r.msg || '使用失败')
      })
    },
    doExchangeSell () {
      api.post('/games/ezfy/exchange/sell', {
        es_type: parseInt(this.sellType), es_count: parseInt(this.sellCount) || 0,
        total_price: parseInt(this.sellPrice) || 0
      }).then(r => this.alert(r, '挂单已发布', () => {
        this.sellCount = 0
        this.sellPrice = 0
        this.exchangeMPage = 1
        this.loadExchange()
      }))
    },
    doExchangeBuy (e) {
      api.post('/games/ezfy/exchange/buy', { id: e.id }).then(r => this.alert(r, '购买成功', () => this.loadExchange()))
    },
    doExchangeCancel (e) {
      api.post('/games/ezfy/exchange/cancel', { id: e.id }).then(r => this.alert(r, '挂单已撤销', () => this.loadExchange()))
    },
    // ★ 2026-09-30 向系统出售资源：预览应得黄金（与后端同口径整数除法）
    sysSellPreview () {
      const count = parseInt(this.sellSysCount) || 0
      const ratio = this.sysSellRatio[this.sellSysType] || 0
      if (count <= 0 || !ratio) return 0
      const base = Math.floor(count * ratio / 100)
      const fee = this.sysSellFee || 0
      return Math.floor(base * (100 - fee) / 100)
    },
    doExchangeSysSell () {
      this.sysSellLostWarn = ''
      api.post('/games/ezfy/exchange/sys-sell', {
        es_type: parseInt(this.sellSysType), es_count: parseInt(this.sellSysCount) || 0
      }).then(r => {
        if (r.code === 0 && r.data && r.data.gold_lost) {
          this.sysSellLostWarn = '黄金累加超过黄金上限，超出 ' + (r.data.lost_gold || 0) + ' 已丢失'
        }
        this.alert(r, '已售出', () => {
          this.sellSysCount = 0
          this.exchangeMPage = 1
          this.loadExchange()
        })
      })
    },
    // ---- 任务/福利 ----
    doAward (t) {
      api.post('/games/ezfy/tasks/award', { task_id: t.id }).then(r => {
        // ★ 领奖成功后本地把状态置为「已领取」，否则按钮一直停在 [领奖]（用户反馈的 bug）
        if (r && r.code === 0) t.status = 2
        this.alert(r, '奖励已领取')
      })
    },
    doSign () {
      // ★ 签到后必须重新拉 welfare 与资源，否则：
      //   ① 首页「每日签到」还显示「签到」（应为「已签到」）—— 用户反馈的 bug
      //   ② 资源数字不刷新，看起来像「签到后资源没加/反而少了」
      api.post('/games/ezfy/welfare/sign', {}).then(r => {
        this.alert(r, '签到成功', () => {
          this.loadWelfare()
          this.load()
          if (this.cur === 'builds') this.loadRes()
        })
      })
    },
    doGift (t) {
      api.post('/games/ezfy/welfare/gift/' + t, {}).then(r => this.alert(r, '礼包已领取'))
    },
    setWelfareTab (n) {
      this.welfareTab = n
    },
    doTreasureSign () {
      // ★ 2026-09-28 宝物签到：领完刷新 welfare（更新连续天数与今日已签），宝物即时入背包
      api.post('/games/ezfy/welfare/treasure-sign', {}).then(r => {
        this.alert(r, '宝物签到成功', () => {
          this.loadWelfare()
        })
      })
    },
    rewardText (rw) {
      if (!rw) return ''
      const parts = []
      if (rw.gold) parts.push('金' + rw.gold)
      if (rw.food) parts.push('粮' + rw.food)
      if (rw.steel) parts.push('钢' + rw.steel)
      if (rw.oil) parts.push('油' + rw.oil)
      if (rw.rare) parts.push('稀' + rw.rare)
      if (rw.prestige) parts.push('声望' + rw.prestige)
      return parts.join(' ')
    },
    woundedList (type) {
      return (this.troopsData.wounded || []).filter(w => w.type === type)
    },
    // ★ 恢复/召回全部伤兵需要的黄金合计（单价由后端下发 heal_gold）
    woundedHealCost (type) {
      return this.woundedList(type).reduce((s, w) => s + (w.heal_gold || 0) * (w.count || 0), 0)
    },
    troopTypeName (t) {
      return { 1: '海军', 2: '陆军', 3: '空军', 4: '城防' }[t] || '部队'
    },
    // 城内某兵种数量(军工厂页显示 兵种:数量)
    troopCount (tid) {
      const t = (this.troopsData.troops || []).find(x => x.troop_id === tid)
      return t ? t.count : 0
    },
    remain (endTime) {
      if (!endTime) return ''
      const ms = endTime - Date.now()
      if (ms <= 0) return '已完成'
      const s = Math.floor(ms / 1000)
      if (s < 60) return s + '秒'
      const m = Math.floor(s / 60)
      if (m < 60) return m + '分' + (s % 60) + '秒'
      const h = Math.floor(m / 60)
      return h + '时' + (m % 60) + '分'
    },
    // ★ 数字千分位（军队动态的「待带回」和出征页的「本次出兵/上限」都用它）
    //   之前模板里引用了 fmtN 但方法从未定义 → Vue 渲染直接抛
    //   "TypeError: _vm.fmtN is not a function"，整页白掉，且只在对应分支被渲染时才暴露。
    // ★ 2026-09-28 首页头部产量缩写：≥1万 时按「万」缩成紧凑形式, 保留负号; 不足1万显示原值
    //   采用向下取整(截断), 不四舍五入——保证展示值≤真实值, 玩家不会被夸大误导
    //   例如 -46393790 → -4639万, 113840 → 11.3万, 54566 → 5.4万
    fmtProd (n) {
      const v = Number(n)
      if (!isFinite(v)) return '0'
      const neg = v < 0
      const a = Math.abs(v)
      if (a >= 10000) {
        const w = Math.floor(a / 10000)
        const rem = a % 10000
        // 非整整万但剩余≥千时补 1 位小数(百位截断), 否则整数万
        let s = ''
        if (rem >= 1000) {
          s = w + '.' + Math.floor(rem / 1000) // 万分位→十分之一万, 只取1位(截断)
        } else {
          s = String(w)
        }
        return (neg ? '-' : '') + s + '万'
      }
      return (neg ? '-' : '') + a.toLocaleString('en-US')
    },
    fmtN (n) {
      const v = Number(n)
      if (!isFinite(v)) return '0'
      return v.toLocaleString('en-US')
    },
    // ★ 宝箱奖池/开箱结果的品质着色（普通/稀有/史诗/传说，给不同颜色区分）
    // ★ 军官装备品质同套配色：初级→灰 / 中级→蓝 / 高级→紫 / 特殊→橙（一眼看出哪个好）
    qualityClass (q) {
      if (q === '传说' || q === '传奇' || q === '特殊') return 'q-legend'
      if (q === '史诗' || q === '高级') return 'q-epic'
      if (q === '稀有' || q === '中级') return 'q-rare'
      return 'q-normal'
    },
    // ★ 把后端下发的创建时间格式化成年-月-日（公告标题后的发布时间）。
    //   入参可能是 "2026-09-23T11:11:28+08:00" 或已是 "2006-01-02 15:04" 字符串。
    fmtDate (s) {
      if (!s) return ''
      // 取 ISO 字符串最前面一段 yyyy-MM-dd；兼容已格式化的 "2006-01-02 ..."
      const m = String(s).match(/(\d{4})-(\d{2})-(\d{2})/)
      return m ? m[1] + '-' + m[2] + '-' + m[3] : ''
    },
    // ★ 大数加单位（万/亿），资源详情里动辄十几位数字，不缩一下没法看
    fmtBig (n) {
      const v = Number(n || 0)
      if (!isFinite(v)) return '0'
      const abs = Math.abs(v)
      const sign = v < 0 ? '-' : ''
      if (abs >= 1e8) return sign + (abs / 1e8).toFixed(2).replace(/\.?0+$/, '') + '亿'
      if (abs >= 1e4) return sign + (abs / 1e4).toFixed(2).replace(/\.?0+$/, '') + '万'
      return String(v)
    },
    fmtTime (t) {
      if (!t) return ''
      let d
      if (typeof t === 'number') {
        d = new Date(t < 1e12 ? t * 1000 : t)
      } else {
        // 后端时间可能是 ISO8601(带时区, 如 2026-09-18T21:15:25.054+08:00)
        // 或 "YYYY-MM-DD HH:mm:ss"; 后者在部分浏览器需把 - 换成 /
        d = new Date(t)
        if (isNaN(d.getTime())) d = new Date(String(t).replace(/-/g, '/'))
      }
      if (isNaN(d.getTime())) return ''
      const p = n => String(n).padStart(2, '0')
      // ★ 用户要求：战报时间要带年份（原来是 MM-DD HH:mm:ss，跨年就分不清）
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) +
        ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    // ---- 页面内提示 / 确认（替代 alert / confirm / prompt）----
    notify (text, type) {
      const t = String(text === undefined || text === null ? '' : text)
      if (!t) return
      this._msgSeq = (this._msgSeq || 0) + 1
      const id = this._msgSeq
      this.msgs.push({ id: id, text: t, type: type || this.guessMsgType(t) })
      if (this.msgs.length > 6) this.msgs.shift()
      // ★ 2026-09-27 用户要求：不需要玩家确定的提示 3 秒→1.5 秒后自动消失（也可点 [关闭] 手动收起）
      setTimeout(() => this.closeMsg(id), 1500)
    },
    // 提示条样式：fixed 定位在刚才点击的**控件正下方居中**（留 12px 间隙），
    // 不再用原始鼠标坐标，避免盖住按钮本身。元素已滚出视口时兜底顶部居中。
    // 多条时向下错开 36px，防重叠。
    msgStyle (m, i) {
      const vw = window.innerWidth
      const vh = window.innerHeight
      const w = Math.min(320, vw - 16)
      const step = 36
      let left, top
      const el = this._lastClicked
      const rect = el && typeof el.getBoundingClientRect === 'function' ? el.getBoundingClientRect() : null
      // 元素在视口内（横向不越界、纵向可见）才按它定位；否则退到顶部居中
      const inView = rect && rect.left < vw && rect.right > 0 && rect.top < vh && rect.bottom > 0
      if (rect && inView) {
        let cx = rect.left + rect.width / 2
        left = Math.round(cx - w / 2)
        left = Math.max(8, Math.min(left, vw - w - 8))
        let below = rect.bottom + 12 + i * step
        if (below + 40 > vh) {
          // 元素靠底：改成出现在元素上方
          top = Math.max(8, rect.top - 12 - 40)
        } else {
          top = Math.round(below)
        }
      } else {
        left = Math.round((vw - w) / 2)
        top = 8
      }
      return { position: 'fixed', left: left + 'px', top: top + 'px',
        maxWidth: w + 'px', zIndex: 9999, boxShadow: '0 2px 8px rgba(0,0,0,.25)' }
    },
    guessMsgType (t) {
      if (/失败|不足|错误|不能|无法|没有|请先|需要/.test(t)) return 'error'
      if (/成功|已|完成/.test(t)) return 'ok'
      return 'info'
    },
    closeMsg (id) {
      this.msgs = this.msgs.filter(m => m.id !== id)
    },
    // 内联确认：返回 true/false；带输入框时返回输入的字符串或 null（取消）
    ask (text, opts) {
      opts = opts || {}
      return new Promise(resolve => {
        this._askResolve = resolve
        this.askBox = {
          show: true, text: String(text || ''), input: !!opts.input,
          placeholder: opts.placeholder || '', value: opts.value || ''
        }
        // 渲染后量一次真实高度给 askStyle 用（带输入框的会高不少）
        this.askH = 0
        this.$nextTick(() => {
          const el = this.$refs.askBox
          if (el && el.offsetHeight) this.askH = el.offsetHeight
        })
      })
    },
    askConfirm (ok) {
      const box = this.askBox
      const r = this._askResolve
      this._askResolve = null
      this.askBox = { show: false, text: '', input: false, placeholder: '', value: '' }
      if (!r) return
      if (!ok) { r(box.input ? null : false); return }
      r(box.input ? box.value : true)
    },
    // 统一的接口结果提示（原来弹 alert，现在落到页面消息区）
    // ★ after：成功后要额外刷新的数据（如军团信息）。只刷 /view 不够 ——
    //   军团数据来自 /games/ezfy/corps/*，不重新拉就会「加入后还显示未加入」。
    //
    // ★ 用户要求「提示 ok 改成具体的描述，比如领取就是领取成功，不知道的就是操作成功」：
    //   所以每个调用点都要传 fallback（领取成功 / 购买成功 / 建造命令已下达 …），
    //   只有真的无从判断时才回落「操作成功」。
    alert (r, fallback, after) {
      if (r && r.code === 0) {
        const m = (r.data && r.data.msg) ? r.data.msg : (fallback || '操作成功')
        this.notify(m, 'ok')
        this.load()
        if (typeof after === 'function') after()
      } else {
        this.notify((r && r.msg) ? r.msg : '操作失败', 'error')
      }
    },
    // ★ 用户反馈「连点会出现多条」→ 「一次点击 = 一条命令」的操作统一走这里防抖：
    //   同一个 key 的请求还没返回时，后续点击直接忽略（不会重复下单）。
    //   fn 需要返回 Promise（api.post/get 都是），请求结束（无论成败）自动解锁。
    once (key, fn) {
      if (!this._onceMap) this._onceMap = {}
      if (this._onceMap[key]) return
      this._onceMap[key] = true
      const release = () => { this._onceMap[key] = false }
      let ret
      try {
        ret = fn()
      } catch (e) {
        release()
        throw e
      }
      if (ret && typeof ret.then === 'function') ret.then(release, release)
      else release()
    },
    // ---- 军官/学院 ----
    loadAcade () {
      api.get('/games/ezfy/officers').then(r => {
        if (r.code === 0) this.officerData = r.data
      })
      this.loadAcadeTab()
    },
    loadAcadeTab () {
      if (this.acadeTab === 'search') this.loadRecruit()
      else if (this.acadeTab === 'skill') this.loadAcadeSkills()
      else if (this.acadeTab === 'equip') this.loadAcadeEquip()
      else if (this.acadeTab === 'scheme') this.loadAcadeGenerals()
      else {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      }
    },
    switchAcade (tab) {
      this.acadeTab = tab
      // ★ 进装备页时把检索/分页复位，避免上次的搜索词把列表筛空（看着像「装备没了」）
      if (tab === 'equip') {
        this.equipTab = 'my'
        this.equipWord = ''
        this.equipPage = 1
        this.equipAllWord = ''
        this.equipAllPage = 1
      }
      if (tab === 'officer' || tab === 'mayor' || tab === 'captive') {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      }       else if (tab === 'search') this.loadRecruit()
      else if (tab === 'skill') this.loadAcadeSkills()
      else if (tab === 'equip') this.loadAcadeEquip()
      else if (tab === 'scheme') this.loadSchemes()
      else if (tab === 'generals') this.loadAcadeGenerals()
    },
    // ★ 计谋列表（配置由后端下发，含持有信号弹数量）
    loadSchemes () {
      api.get('/games/ezfy/schemes').then(r => {
        if (r.code === 0) this.schemeData = r.data
      })
    },
    async doScheme (s) {
      const b = this.schemeData
      if (b.bullet_have < s.bullet) { this.notify('「' + b.bullet_name + '」不足'); return }
      const body = { scheme_id: s.id }
      if (s.kind === 1) {
        const x = parseInt(this.schemeX)
        const y = parseInt(this.schemeY)
        if (!x || !y) { this.notify('「' + s.name + '」需要填写目标城市坐标（x / y）'); return }
        body.target_x = x
        body.target_y = y
      }
      const tip = s.kind === 1
        ? ('确认对 (' + body.target_x + ',' + body.target_y + ') 发动「' + s.name + '」吗？消耗 ' + b.bullet_name + '×' + s.bullet)
        : ('确认发动「' + s.name + '」吗？消耗 ' + b.bullet_name + '×' + s.bullet)
      if (!await this.ask(tip)) return
      api.post('/games/ezfy/scheme/use', body).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '发动失败'); return }
        this.notify(r.msg || '计谋已发动')
        this.loadSchemes()
        this.loadBag()
        this.load()
      })
    },
    loadRecruit () {
      api.get('/games/ezfy/acade/recruit').then(r => {
        if (r.code === 0) this.recruitData = r.data
      })
    },
    loadAcadeSkills () {
      api.get('/games/ezfy/officers/skills').then(r => {
        if (r.code === 0) this.skillData = r.data
      })
    },
    loadAcadeEquip () {
      this.loadEquipSets()
      api.get('/games/ezfy/officers/equipments').then(r => {
        if (r.code === 0) this.equipData = r.data
      })
    },
    // ★ 2026-09-25：全部套装配置（含加成/部位/我拥有几件）—— 只拉一次，各页共用。
    //   装备页/商城页/军官装备页都要「set_id → 这套穿齐给什么」，所以做成缓存。
    loadEquipSets () {
      if (this.allSets && this.allSets.length) return Promise.resolve()
      return api.get('/games/ezfy/equipsets').then(r => {
        if (r.code === 0) this.allSets = r.data.sets || []
      }).catch(() => {})
    },
    // set_id → 套装配置（没有/散件返回 null）
    setOf (setId) {
      return (setId > 0 && this.setMap[setId]) ? this.setMap[setId] : null
    },
    // 套装加成文案（复用装备属性的展示口径：六项战斗属性 + 军事/后勤/学识）
    setBonusText (setId) {
      const s = this.setOf(setId)
      return s ? this.equipAttrText(s) : ''
    },
    // 点装备名 / 套装名 → 在**这一行**下面展开详情卡；再点同一个收起。
    // ★ 参数 equipId 是**这一行装备的 id**，不是 set_id：同一套会有 9~11 行，
    //   按 set_id 展开的话每行都会各蹦一张一样的卡（实测 10 件套蹦 10 张）。
    // ★ 2026-09-25 用户要求「名称点击查看该装备的加成，点击套装显示套装的加成」：
    //   两个入口看**不同**的内容，用 mode 区分（'item' 只看这件自己的加成 / 'set' 只看套装加成）。
    //   同一个入口点两次 = 收起；点另一个入口 = 直接换成另一份内容（不收起）。
    toggleDetail (equipId, mode) {
      if (!equipId) return
      const m = mode || 'item'
      if (this.detailRowId === equipId && this.detailMode === m) {
        this.detailRowId = 0
        this.detailMode = ''
        return
      }
      this.detailRowId = equipId
      this.detailMode = m
    },
    loadAcadeGenerals () {
      api.get('/games/ezfy/officers/generals').then(r => {
        if (r.code === 0) this.generalData = r.data
      })
    },
    openOfficer (id) {
      this.cur = 'officerdetail'
      // ★ 换军官时把详情页签、背包装备的检索/分页复位
      this.officerDetailTab = 'attr'
      this.officerBagWord = ''
      this.officerBagPage = 1
      // ★ 2026-10-01 修复「点击军官有时候空白」：进页先清掉旧军官数据/错误，
      //   加载完成前显示「加载中...」，避免残留上一名军官的详情或白屏
      this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
      this.officerDetailError = ''
      this.loadOfficerDetail(id)
    },
    loadOfficerDetail (id) {
      // ★ 2026-10-01 防串数据：连续点多名军官时，只认最后一次请求的结果
      const seq = (this._officerDetailSeq = (this._officerDetailSeq || 0) + 1)
      // ★ 军官详情的「已穿戴装备 / 装备背包」要显示套装加成 → 一并把套装配置拉上
      this.loadEquipSets()
      api.get('/games/ezfy/officers/' + id).then(r => {
        if (seq !== this._officerDetailSeq) return
        if (r.code === 0) {
          this.officerDetailError = ''
          this.officerDetail = r.data
        } else {
          // 加载失败（如武将已调往别的城市 / 已被流放）→ 不再静默空白：
          // 提示原因并刷新军官列表，让列表与后端状态一致
          this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
          this.officerDetailError = r.msg || '加载军官详情失败, 请重试'
          this.loadAcade()
        }
      }).catch(() => {
        if (seq !== this._officerDetailSeq) return
        this.officerDetail = { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 }
        this.officerDetailError = '网络开小差了, 请稍后重试'
      })
    },
    // 军校直接使用招生简章刷新（不占每小时次数；不用跳背包）
    async doUseRecruitTicket () {
      if (!this.recruitData.academy_level) { this.notify('需要先建造军校'); return }
      if (this.bagCount(13) <= 0) { this.notify('没有「招生简章」，可到商城购买'); return }
      if (!await this.ask('确认使用「招生简章」×1 刷新军校候选名将吗？（不占用每小时次数）')) return
      api.post('/games/ezfy/acade/recruit/ticket', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadAcade()
          this.loadBag()
        } else this.notify(r.msg)
      })
    },
    // 背包里某道具的数量
    bagCount (cfgId) {
      const it = (this.bagItems || []).find(x => x.cfg_id === cfgId)
      return it ? it.count : 0
    },
    doRefreshRecruit () {
      api.post('/games/ezfy/acade/recruit/refresh', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '刷新失败')
        this.loadRecruit()
      })
    },
    async doRecruit (g) {
      if (!await this.ask('确定雇佣 ' + g.name + ' 吗? 需要 ' + g.cost + ' ' + this.resNames.gold)) return
      api.post('/games/ezfy/acade/recruit/hire', { key: g.key }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '雇佣失败')
        this.loadRecruit()
        this.loadAcade()
      })
    },
    // ★ 2026-09-28 用户要求：洗点入口放到军官详情页（原来要背包里翻「军官洗点卡」使用）。
    //   对当前军官直接消耗 1 张洗点卡；后端 case 12 兜底校验（出征中/无卡等）。
    async doRespec () {
      const o = this.officerDetail.officer
      if (!o) return
      await this.loadBag() // 军官详情页不常驻背包数据，先拉一次再判断持有量
      const card = this.bagItems.find(x => x.item_type === 12)
      if (!card) { this.notify('背包没有「军官洗点卡」，可在商城购买'); return }
      if (!await this.ask('确认对 ' + o.name + ' 使用「军官洗点卡」×1 吗？\n属性将重置为军官池初始属性，已分配的点退回待分配点（等级/经验/技能保留）')) return
      api.post('/games/ezfy/bag/use', { cfg_id: card.cfg_id, count: 1, city_id: this.city.id, officer_id: o.id }).then(r => {
        if (r.code === 0) {
          this.notify(r.data && r.data.msg ? r.data.msg : '洗点完成')
          this.loadOfficerDetail(o.id)
        } else this.notify(r.msg || '洗点失败')
      })
    },
    doGrant () {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/grant', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '赏赐失败')
        this.loadOfficerDetail(id)
      })
    },
    // ★ 2026-09-28 用户要求：宝物可赏赐给军官加忠诚，品质不同加的不同（最高 +50）。
    //   点 [赏赐宝物] 展开可选列表（equipData.bag 里未穿戴的）；点某件 [赏赐] 确认后消耗该件并加忠诚。
    treasureLoyaltyGain (tier) {
      // 与后端 ezfyTreasureLoyalty 对齐：tier 1~4（初级/中级/高级/特殊）→ +10/+20/+35/+50
      return [0, 10, 20, 35, 50][tier] || 0
    },
    async doTreasureGrant (e) {
      const o = this.officerDetail.officer
      if (!o) return
      if (o.loyalty >= 100) { this.notify('忠诚已满, 无需赏赐'); return }
      // ★ 2026-09-28 修 bug：模板原来写 @click="doTreasureGrant"（不带括号），
      //   Vue 会把**原生 MouseEvent** 当宝物对象传进来 —— 于是弹出
      //   「赏赐「undefined」(普通) …」，点确认还会用 undefined 当 equip_id 提交。
      //   现在按「有没有装备 id」判断：事件对象/空值一律走展开分支，彻底免疫。
      if (!e || !e.id) {
        // 点「赏赐宝物」按钮：先拉最新军官详情（背包随之刷新）再展开选择列表
        this.loadOfficerDetail(o.id)
        this.officerTreasureOpen = !this.officerTreasureOpen
        return
      }
      const gain = this.treasureLoyaltyGain(e.tier) || 10
      if (!await this.ask('赏赐「' + e.name + '」(' + (e.tier_name || '普通') + ') 给 ' + o.name + ' 吗？\n消耗这件宝物, 忠诚 +' + gain)) return
      api.post('/games/ezfy/officers/' + o.id + '/treasure-grant', { equip_id: e.id }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '赏赐失败'); return }
        this.notify(r.data && r.data.msg ? r.data.msg : ('赏赐成功, 忠诚 +' + gain))
        this.officerTreasureOpen = false
        this.loadOfficerDetail(o.id)
      })
    },
    // ★ 属性加点（每升 1 级得 1 点，只影响自己的军官）
    doAddAttr (attr, count) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/attr', { attr, count }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '加点失败'); return }
        this.notify(r.msg || '加点成功')
        this.loadOfficerDetail(id)
      })
    },
    async doAddAttrAll (attr) {
      const free = this.officerDetail.officer.free_points
      const name = { military: '军事', logistics: '后勤', learning: '学识' }[attr] || attr
      if (!await this.ask('把剩余 ' + free + ' 点全部加到「' + name + '」吗？')) return
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/attr/all', { attr }).then(r => {
        if (r.code !== 0) { this.notify(r.msg || '加点失败'); return }
        this.notify(r.msg || '加点成功')
        this.loadOfficerDetail(id)
      })
    },
    // ★ 升星（消耗「星级徽章」；成功率/每星加点/上限均可后台配置）
    async doStarUp () {
      const o = this.officerDetail.officer
      if (o.star >= o.star_max) { this.notify('星级已达上限'); return }
      if (o.star_card <= 0) { this.notify('没有「星级徽章」，可在商城购买或开宝箱获得'); return }
      if (!await this.ask('使用 1 枚「星级徽章」给 ' + o.name + ' 升星吗？\n成功率 ' + o.star_rate +
        '%，成功后三维各 +' + o.star_attr_gain + '（星级 ' + o.star + '→' + (o.star + 1) + '）')) return
      const id = o.id
      api.post('/games/ezfy/officers/' + id + '/starup', {}).then(r => {
        this.notify(r.msg || (r.code === 0 ? '升星成功' : '升星失败'))
        this.loadOfficerDetail(id)
        this.loadBag()
        this.loadAcade()
      })
    },
    // ★ 军官改名（消耗「军官改名卡」，只改玩家自己的军官）
    //   ⚠️ 方法名不能叫 doRename：城市改名已用该名，对象字面量后定义会覆盖先定义，
    //   导致城市改名的 [确定] 跑到军官改名逻辑（读 officerDetail 报错）。
    async doOfficerRename () {
      const o = this.officerDetail.officer
      if (o.is_captive === 1) { this.notify('俘虏不能改名, 请先在军校收编'); return }
      if (o.rename_card <= 0) { this.notify('没有「军官改名卡」，可在商城购买'); return }
      const name = await this.ask('给 ' + o.name + ' 改个新名字？', {
        input: true, placeholder: '2~12 个字符', value: o.name
      })
      if (!name) return
      const id = o.id
      api.post('/games/ezfy/officers/' + id + '/rename', { name }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '改名失败')
        this.loadOfficerDetail(id)
        this.loadBag()
      })
    },
    doLearn (s) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'learn', skill_id: s.id }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '学习失败')
        this.loadOfficerDetail(id)
      })
    },
    doForget (name) {
      const id = this.officerDetail.officer.id
      const sid = this.skillIdByName(name)
      api.post('/games/ezfy/officers/' + id + '/skill', { op: 'forget', skill_id: sid }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '遗忘失败')
        this.loadOfficerDetail(id)
      })
    },
    skillIdByName (name) {
      for (const s of this.officerDetail.all_skills) {
        if (s.name === name) return s.id
      }
      return 0
    },
    doEquip (e) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: e.id, op: 'on' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '穿戴失败')
        this.loadOfficerDetail(id)
      })
    },
    // ★ 2026-09-29 装备背包叠加行 [穿戴]：穿组内第一件未穿戴的
    //   （同一 cfg 属性一致，穿哪件都一样；穿完该部位即占满，剩余同款不能再穿）
    doEquipGroup (g) {
      if (!g || !g.items || !g.items.length) return
      this.doEquip(g.first)
    },
    doUnequip (equipId) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: equipId, op: 'off' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '卸下失败')
        this.loadOfficerDetail(id)
      })
    },
    async doUnequipAll () {
      const id = this.officerDetail.officer.id
      if (!await this.ask('确定卸下该军官身上全部 ' + this.officerDetail.equipped.length + ' 件装备吗?')) return
      api.post('/games/ezfy/officers/' + id + '/unequip-all', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '卸下失败')
        else this.notify(r.msg || '已卸下全部装备')
        this.loadOfficerDetail(id)
      })
    },
    async doEquipSet (s) {
      const id = this.officerDetail.officer.id
      if (!await this.ask('确定一键穿戴「' + s.name + '」整套吗?（同部位已穿戴的会自动卸下让位）')) return
      api.post('/games/ezfy/officers/' + id + '/equip-set', { set_id: s.set_id }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '穿戴失败')
        else this.notify(r.msg || '穿戴成功')
        this.loadOfficerDetail(id)
      })
    },
    doPosition (o, pos) {
      api.post('/games/ezfy/officers/' + o.id + '/position', { position: pos }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '任命失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
        // ★ 任命/卸任后同步刷新出征页的带队军官列表：若玩家随后立即出征，
        //   下拉里必须是最新的可选军官（避免「刚任命的市长/城守不出现在下拉」的观感）
        this.loadOnDutyOfficers()
        this.loadCityOfficers()
      })
    },
    async doCaptive (o, op) {
      if (op === 'free' && !await this.ask('确定释放俘虏 ' + o.name + ' 吗?')) return
      api.post('/games/ezfy/officers/' + o.id + '/captive', { op: op }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '操作失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
      })
    },
    async doExile () {
      if (!await this.ask('确定流放该武将吗? 流放后无法找回!')) return
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/exile', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '流放失败')
        this.go('acade')
      })
    }
  }
}
</script>

<style>
/* ===== 二战风云 怀旧文字WAP主题(复刻 stzb-fk city-theme.css), 与家园页面一致靠左排布 ===== */

/* 沉浸式: 原版 style.css 给 body 留了 5px 外边距, 会让深色标题条四周露白边.
   进入本页时由 Ezfy.vue 的 mounted 挂上这个 class, 离开时移除 */
body.ezfy-immersive { margin: 0; }

.ezfy-page {
  background: #fff;
  min-height: 100%;
  color: #333;
  /* ★ 字体族: 2026-09-24 用户反馈「微软雅黑不好看」→ 改复古宋体风
     （3GQQ 时代 WAP 文字游戏的主流样式, 标题/正文统一宋体更有怀旧味）。
     回退链: Windows→宋体/SimSun, macOS→宋体-简(Songti SC), 其余→serif。
     ★ 2026-09-25：**只保留在桌面**。手机上换成系统黑体，见下面的 @supports 段。 */
  font-family: '宋体', 'SimSun', 'Songti SC', 'NSimSun', '新宋体', serif;
  /* ★ 2026-09-25 用户要求「页面文字大小除了地图，全部改成和首页导航(聊天/邮箱/军情/任务/好友/首页)一样大」：
     首页导航字号 17px 定为**全站唯一基准** —— 除地图格(.ezfy-map-table / .ezfy-cell)外，
     页面所有文字都取这个变量。以后要整体调大小，只改这一行。
     例外（都是「非正文」或布局硬约束，已在各自规则里注明）：
       ① .title-bar 标题栏 18px —— 用户此前明确点名「勿动」；
       ② .panel-title 小标题 18px —— 标题与正文拉开一档；
       ③ 地图格 + 战场指挥室表格 —— 手机上列数太多，必须缩（见各自媒体查询）。 */
  --fs: 17px;
  font-size: var(--fs);
  /* ★ 2026-09-25 iPhone 修复①：iOS Safari 会按视口宽度**自动放大/缩小正文**，
     不锁死的话同一段文字在 iPhone 和桌面看到的字号不一致（用户反馈「手机上字体不好看」）。
     100% = 完全按 CSS 里写的字号来，不做任何自动缩放。 */
  -webkit-text-size-adjust: 100%;
  text-size-adjust: 100%;
  line-height: 1.5;
  /* 根容器左右不再用负 margin: 会溢出 #app 产生横向滚动条.
     铺满由内部 .title-bar 的 margin:0 -8px 抵消 padding 实现 */
  margin: 0;
  padding: 0 8px 20px;
}
/* ★ 2026-09-25 iPhone 修复②（关键的一条）：**iOS 上没有「宋体 / SimSun」这两个字体**，
   于是回退到系统自带的 **Songti SC（宋体-简）** —— 它是衬线体、笔画极细，
   小字号在 Retina 屏上又细又灰、边缘发虚，观感比 Windows 的 SimSun 差一大截，
   这就是用户说的「iPhone 上字体不好看」。
   处理（★ 用户要求「iPhone 单独用 iPhone 自己的字体，其他设备不变」）：
     · **桌面（Windows/macOS）保留复古宋体**，一行不动；
     · **只有 iPhone / iPad 换成苹果自带的系统字体** —— 下面这份就是 iPhone 上的常驻字体：
         `-apple-system` / `system-ui` → 苹果系统字体（西文 San Francisco + 中文 **苹方**）
         `'PingFang SC'` 苹方（iOS 9+ 中文默认）｜ `'Heiti SC'` 黑体-简（更老的 iOS）
         `'Hiragino Sans GB'` 冬青黑体简体中文（兜底）
       与《镜花缘》(Jingwt.vue)、《西游记》(Xiyou.vue) 的字体族方向一致。
   怎么只圈到苹果设备：JS 判 iOS（mounted 里给 body 挂 .ezfy-ios），Apple 专属、绝不漏判。
   ⚠️ 不要用 `@media (hover:none) and (pointer:coarse)` —— 那会把安卓也一起改了,
     用户明确要求「其他设备不变」。
   ★ 2026-09-27：旧实现用 `@supports (-webkit-touch-callout: none)`，但 iOS Safari 的
     @supports 解析器不认 -webkit-touch-callout → 条件永不成立 → iPhone 一直回落宋体-简
     (Songti SC) 细灰发虚。故改用 JS 判定（见 mounted），比 @supports 可靠。 */
body.ezfy-ios .ezfy-page,
body.ezfy-ios .ezfy-page button,
body.ezfy-ios .ezfy-page input,
body.ezfy-ios .ezfy-page select,
body.ezfy-ios .ezfy-page textarea {
  font-family: -apple-system, system-ui, 'PingFang SC', 'Heiti SC',
               'Hiragino Sans GB', 'Helvetica Neue', sans-serif;
}
/* 表单控件/按钮默认不继承字体族, 显式补上(原版也是 body,button,input,select,textarea 一起设) */
.ezfy-page button,
.ezfy-page input,
.ezfy-page select,
.ezfy-page textarea {
  font-family: inherit;
}
.ezfy-page .home-wrap {
  font-size: var(--fs);
}
.ezfy-page a {
  text-decoration: none;
  margin: 0 1px;
  color: #0645ad;
}
.ezfy-page a:hover { text-decoration: underline; }
.ezfy-page .title-bar {
  background: #050709;
  color: #fff;
  /* ★ 2026-09-25：用户此前明确点名「二战征途-【1区】红色警戒」这一行勿动 → 保留 18px，
     没有跟着全站统一成 var(--fs)。要一起拉平说一声。 */
  font-size: 18px;
  font-weight: bold;
  text-align: left;
  padding: 6px 10px;
  letter-spacing: 1px;
  margin: 0 -8px;
  box-sizing: border-box;
}
.ezfy-page .top-nav {
  text-align: left;
  /* ★ 底部不留 padding: 导航链接自身已有 2px 上下 padding, 再加 4px 会让「紧跟它后面
     的第一个区块」上方凭空多出 4px 留白(导航字→下一行的距离比行与行之间大一截)。
     顶部那 4px 要留 —— 上面是深色标题条, 需要这段间隔。 */
  padding: 4px 0 0;
}
.ezfy-page .top-nav a {
  display: inline-block;
  /* ★ 用户「聊天/邮箱/军情/任务/好友/首页 间隔稍微大一点」→ 0 改 4px；
     随后「又大了 稍微小一点」→ 收到 3px（相邻两词间距 3+3 = 6px）。
     ★ 必须与下面 .ezfy-subnav a 保持同一个值，否则「点进去二级菜单变宽/变窄」。 */
  padding: 2px 3px;
  line-height: 1.35;
  /* ★★ 这一行就是全站字号基准（--fs）的来源：用户 2026-09-25 要求
     「页面文字除了地图，全部和首页导航一样大」→ 其它地方一律写 var(--fs)。 */
  font-size: var(--fs);
}
/* ★ 用户反馈「导航整体有点靠右」：每个链接自带 padding + 1px margin，
   于是第一个链接「聊天」的文字比下面正文行(公告/新城市…)右移。
   去掉首链接的左侧留白 → 整条导航左移，与正文左对齐。 */
.ezfy-page .top-nav a:first-child { margin-left: 0; padding-left: 0; }
/* 二级导航(资源/军官/军队/科技/城防/统帅) —— 复刻原版军队/城防/兵种页里的那行 */
/* ★ 配色按用户要求：默认 #004299，当前选中黑色
   ★ 用户反馈「点进去后间隔变大，首页里的这个导航就对」：
     根因是这里用 inline-block(会把换行空白算成一个空格宽)，
     而首页导航用的是 inline。改成 inline 并收窄 padding，
     与首页视觉完全一致。 */
.ezfy-page .ezfy-subnav a {
  display: inline;
  padding: 0 1px;
  /* ★ 2026-09-29 用户要求「资源.军官.军队.科技.城防.统帅 间隔小一点点」：margin 1px → 0 */
  margin: 0;
  /* ★ 字号与 .top-nav a 统一（同一个变量，改一处两处一起变） */
  font-size: var(--fs);
  color: #004299;
}
.ezfy-page .ezfy-subnav a.on { color: #000; font-weight: bold; }
/* ★ 用户「点进去 资源/军官/军队/科技/城防/统帅 左边 和 聊天 的『聊』字对齐」：
   首链接左侧 padding/margin 归零 —— 与 .top-nav a:first-child 同源。
   两者父容器(.old-line / .top-nav)左右 padding 都是 0，归零后文字左边缘必定对齐。 */
.ezfy-page .ezfy-subnav a:first-child { margin-left: 0; padding-left: 0; }
/* ★ 2026-09-29 首页那行「军事.资源.军官.军队.科技.城防.统帅」跨平台间距不一致：
   win/mac/手机 渲染同一段 inline 换行空白，空格宽度因字体各不相同。
   改成 flex（gap 固定 3px）+ 每个链接稳距，彻底消除换行空白导致的参差。 */
.ezfy-page .old-line.home-nav2 {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 3px;
}
.ezfy-page .old-line.home-nav2 a {
  color: #004299;
  margin: 0;
  padding: 0 1px;
}
.ezfy-page .old-line.home-nav2 a.on { color: #000; font-weight: bold; }
/* 军衔/排行页所有表格：数据水平 + 垂直居中（用户要求）*/
/* ★ 排行页四个表格统一宽度（用户要求「表格有的大有的小，统一整齐」→ 又要求「太长占页面，改50%」）：
   width:50% 占 panel 一半宽度，table-layout:fixed 配合各表 colgroup 比例分列，长文本自动折行 */
.ezfy-page .ezfy-rank-table {
  width: 50%;
  min-width: 360px;
  table-layout: fixed;
  border-collapse: collapse;
}
.ezfy-page .ezfy-rank-table th,
.ezfy-page .ezfy-rank-table td {
  text-align: center;
  vertical-align: middle;
  overflow-wrap: break-word;
}
/* ★ 排行榜优化（用户要求「榜单太单调、没有追榜动力」）：
   名次做成奖牌徽章，前三名金/银/铜；冠亚季军整行按金/银/铜着色 + 冠军皇冠；
   普通行斑马纹 + 悬停。军衔晋升表是静态参照表（无奖牌徽章、无 rank 类），
   不受这些榜单高亮影响。 */
.ezfy-page .rank-medal {
  display: inline-block;
  min-width: 24px;
  height: 24px;
  line-height: 22px;
  padding: 0 8px;
  border-radius: 12px;
  background: #dedede;
  color: #666;
  font-weight: bold;
  /* ★ 2026-09-25 随全站统一：13 → var(--fs)（奖牌里的名次数字也是页面文字） */
  font-size: var(--fs);
  text-align: center;
  border: 1px solid transparent;
  box-sizing: border-box;
}
.ezfy-page .rank-medal.m1 {                          /* 金 */
  background: linear-gradient(180deg, #ffe08a, #f2c94c);
  color: #7a4b00;
  border-color: #e0b93c;
  box-shadow: 0 0 6px rgba(242, 201, 76, .8);
  /* 冠军奖牌大一号，随基准同步 +2 */
  font-size: calc(var(--fs) + 2px);
  min-width: 26px; height: 26px; line-height: 24px; border-radius: 13px;
}
.ezfy-page .rank-medal.m2 {                          /* 银 */
  background: linear-gradient(180deg, #f2f5f7, #cbd4da);
  color: #455a64;
  border-color: #b9c4cb;
  font-size: var(--fs);
}
.ezfy-page .rank-medal.m3 {                          /* 铜 */
  background: linear-gradient(180deg, #f2cdab, #e0a370);
  color: #5d3a1a;
  border-color: #c88a58;
  font-size: var(--fs);
}
/* 冠军皇冠 */
.ezfy-page .rank-crown {
  display: inline-block;
  margin-right: 4px;
  color: #e6a817;
  font-size: 18px;
  vertical-align: middle;
  text-shadow: 0 1px 2px rgba(122, 75, 0, .35);
}
.ezfy-page .ezfy-rank-table tr:nth-child(even) td { background: #faf8f2; }
.ezfy-page .ezfy-rank-table tr:hover td { background: #f0ecdf; }
/* 军衔晋升表「宝物」点击展开行（★ 2026-09-28 宝物列改为点击展示，展开行做浅色底色区分） */
.ezfy-page .ezfy-rank-table tr.rank-treasure-row td { background: #fffbe8; font-weight: normal; }
.ezfy-page .ezfy-rank-table tr.rank-treasure-row:hover td { background: #fffbe8; }
/* 冠/亚/季军整行着色（置于悬停/斑马纹之后，确保三者之上仍保持奖牌底色） */
.ezfy-page .ezfy-rank-table tr.rank-1 td { background: #fdeebb; }
.ezfy-page .ezfy-rank-table tr.rank-2 td { background: #eef2f5; }
.ezfy-page .ezfy-rank-table tr.rank-3 td { background: #f6e3d0; }
.ezfy-page .ezfy-rank-table tr.rank-1 td:first-child,
.ezfy-page .ezfy-rank-table tr.rank-2 td:first-child,
.ezfy-page .ezfy-rank-table tr.rank-3 td:first-child { font-weight: bold; }
/* 学院(acade)页所有表格：数据水平 + 垂直居中（用户要求）*/
.ezfy-page .ezfy-plain-table th,
.ezfy-page .ezfy-plain-table td {
  text-align: center;
  vertical-align: middle;
}
/* ★ 战场指挥室：本回合倒计时条（指令期绿色、锁定后红色） */
.ezfy-page .ezfy-battle-bar {
  width: 100%;
  max-width: 420px;
  height: 8px;
  margin: 3px 0;
  background: #e6e6e6;
  border: 1px solid #c8c8c8;
  overflow: hidden;
}
.ezfy-page .ezfy-battle-bar > i {
  display: block;
  height: 100%;
  transition: width 0.9s linear;
}
.ezfy-page .ezfy-battle-bar > i.on { background: #27763c; }
.ezfy-page .ezfy-battle-bar > i.lock { background: #c0392b; }
/* ★ 战场指挥室：双方兵力表按攻守着色 —— 我方整行浅绿(enemy 浅红)、方标签加粗 */
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-self td { background: #eef7f0 !important; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-enemy td { background: #fdeeec !important; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-self td.ezfy-side-lbl { font-weight: bold; }
.ezfy-page .ezfy-battle-tbl tr.ezfy-row-enemy td.ezfy-side-lbl { font-weight: bold; }
/* ★ 战场指挥室：行动日志 —— 最新回合排最上，回合标题加粗，回合块间留白 */
.ezfy-page .ezfy-battle-legend { font-size: var(--fs); }
.ezfy-page .ezfy-round-block {
  margin: 4px 0 6px;
  padding-left: 6px;
  border-left: 2px solid #ddd;
}
.ezfy-page .ezfy-round-title {
  font-weight: bold;
  color: #333;
  margin-bottom: 2px;
}
.ezfy-page .ezfy-round-line {
  line-height: 1.6;
  font-size: var(--fs);
}
.ezfy-page .ezfy-round-line.green { color: #1d5c2e; }
.ezfy-page .ezfy-round-line.red { color: #a02a1e; }
.ezfy-page .ezfy-round-line.gray { color: #777; }
/* ★ 战报详情/逐回合详情按行上色：攻方绿色、守方红色（看不清谁是谁 → 视觉区分） */
.ezfy-page .rpt-ln { line-height: 1.6; }
.ezfy-page .rpt-ln > span { display: inline; white-space: pre-wrap; }
.ezfy-page .rpt-atk { color: #1d5c2e; }
.ezfy-page .rpt-def { color: #a02a1e; }
/* ★ 商城「装备 / 道具」的分类筛选：flex 自动换行。
   原 grid repeat(6, max-content) 固定 6 列，分类名较长时(手机端)整排溢出容器形成横向滑动；
   2026-09-24 改为流式排列，超过一行宽度就换行(背包分类共用此类)。 */
.ezfy-page .ezfy-slot-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  margin: 2px 0;
  /* ★ 2026-09-25 随全站统一：15 → var(--fs)（分类链接也是页面文字，不再单独小一号） */
  font-size: var(--fs);
}
.ezfy-page .ezfy-slot-grid > a { white-space: nowrap; }
/* ★ 2026-09-25 套装加成展示（用户反馈「套装的加成看不到，不知道买完套装给军官用哪个」）
   三处展示，全部是**纯文本竖排**，窄屏不会横向溢出：
     ① .set-card  —— 装备表里点套装名，在该行下面**跨整行**展开的加成卡
        （刻意不新增表格列：手机上加一列就挤爆，跨整行展开才能完整显示）
     ② .set-block —— 装备页「套装一览」里一套一块，方便上下对比
     ③ .set-mini  —— 商城把「所属套装」塞在名称下面一行（商城表没有套装列） */
.ezfy-page .set-card-row > td {
  /* 表格单元格默认是居中/右对齐，加成卡必须左对齐才好读 */
  text-align: left;
  padding: 3px 6px 6px 0;
}
.ezfy-page .set-card {
  background: #faf8f2;
  border: 1px solid #d8d5cc;
  border-left: 3px solid #2f6fb5;
  border-radius: 3px;
  padding: 4px 8px;
  line-height: 1.7;
}
.ezfy-page .set-card .sc-h { margin-bottom: 2px; }
.ezfy-page .set-card .sc-h .gray { margin-left: 4px; }
.ezfy-page .set-card .sc-b { word-break: break-word; }
.ezfy-page .set-mini { line-height: 1.6; color: #666; word-break: break-word; }
.ezfy-page .set-block {
  margin: 4px 0 6px;
  padding: 4px 8px;
  background: #faf8f2;
  border-left: 3px solid #2f6fb5;
  border-radius: 3px;
}
.ezfy-page .set-block .sb-h { line-height: 1.7; }
.ezfy-page .set-block .sb-h .gray { margin: 0 2px; }
.ezfy-page .set-block .sb-b { line-height: 1.7; word-break: break-word; }
/* 「只看我有的 / 全部套装·可对比」切换（复用页面上的方括号链接风格，选中变红加粗） */
.ezfy-page .set-tab a { margin-right: 8px; }
.ezfy-page .set-tab a.on { color: #c0392b; font-weight: bold; }
/* ★ 购买 / 开箱面板：卡片式，和上方表格拉开层次（原来只是行内一条左边框，挤成一坨） */
.ezfy-page .ezfy-buy-box {
  margin: 8px 0;
  padding: 8px 12px;
  max-width: 560px;
  background: #faf8f2;
  border: 1px solid #d8d5cc;
  border-radius: 3px;
}
.ezfy-page .ezfy-buy-box .bb-title {
  font-weight: bold;
  color: #2f4156;
  padding-bottom: 4px;
  margin-bottom: 6px;
  border-bottom: 1px solid #e8e4d8;
}
.ezfy-page .ezfy-buy-box .bb-row { line-height: 2.2; }
.ezfy-page .ezfy-buy-box .bb-row input[type="number"] { width: 70px; }
.ezfy-page .ezfy-buy-box .bb-total { color: #c0392b; }
/* ★ 商城购买确认内联行：直接展开在「购买」按钮所在行的正下方，淡色背景区分
     （用户要求「确认在购买按钮附近」，不再用表格底部那个独立面板） */
.ezfy-page .ezfy-buy-inline td {
  background: #faf8f2;
  text-align: left;
  line-height: 2.3;
}
.ezfy-page .ezfy-buy-inline > td > span { margin: 0 4px; }
.ezfy-page .ezfy-buy-inline select { margin-right: 12px; }
.ezfy-page .ezfy-buy-inline button { margin: 0 8px 0 12px; }
/* 自适应高度文本域（私聊等） */
.ezfy-page .ezfy-auto-textarea {
  width: 100%;
  box-sizing: border-box;
  min-height: 44px;
  max-height: 160px;
  padding: 4px 6px;
  /* ★ 2026-09-25 随全站统一：15 → var(--fs)（聊天框里的字和正文一样大） */
  font-size: var(--fs);
  line-height: 1.5;
  font-family: inherit;
  border: 1px solid #c8c8c8;
  border-radius: 3px;
  resize: vertical;
  overflow-y: auto;
}
/* 司令部·兵种战斗配置：一兵种一块，窄屏不遮盖 */
.ezfy-page .ezfy-tgt-block {
  padding: 4px 2px; margin: 4px 0; border-bottom: 1px dashed #e2e2e2;
}
.ezfy-page .ezfy-tgt-name { font-weight: bold; color: #2f4156; margin-bottom: 2px; }
.ezfy-page .ezfy-tgt-row { display: flex; align-items: center; flex-wrap: wrap; gap: 4px; margin: 2px 0; }
.ezfy-page .ezfy-tgt-lab { color: #666; min-width: 56px; display: inline-block; }
/* 页面内消息（替代 alert 弹窗）：现在由 msgStyle 固定定位在点击点附近 */
.ezfy-page .ezfy-msg {
  padding: 4px 8px; border-radius: 3px;
  /* ★ 2026-09-25 随全站统一：14 → var(--fs)（提示条也是页面文字） */
  font-size: var(--fs); line-height: 1.5; border-left: 3px solid #999; background: #f5f5f5;
}

.ezfy-page .ezfy-msg-ok { border-left-color: #27763c; background: #eef7f0; color: #1d5c2e; }
.ezfy-page .ezfy-msg-error { border-left-color: #c0392b; background: #fdeeec; color: #a02a1e; }
.ezfy-page .ezfy-msg-info { border-left-color: #2f6f9f; background: #eef4fa; color: #235b85; }
/* 页面内确认条（替代 confirm / prompt 弹窗）
   ★ 2026-09-25 用户反馈「军官详情的一键穿戴，确认跑到页面最上面，要滚上去才能点」：
     改成**浮层** —— 位置由 askStyle() 按「刚才点的那个控件」实时算（position:fixed 内联下发），
     所以这里 margin 归零、加阴影和更实的底色，让它看起来是「贴着按钮弹出来的」，
     而不是页面正文里的一段。z-index 比提示条(.ezfy-msg 9999)高，避免被提示条压住。 */
.ezfy-page .ezfy-ask {
  margin: 0;
  padding: 8px 10px;
  border: 1px solid #d8c890;
  background: #fffbe8;
  border-radius: 4px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, .22);
  box-sizing: border-box;
}
/* ★ 2026-09-25 随全站统一：14 → var(--fs) */
.ezfy-page .ezfy-ask-text { font-size: var(--fs); color: #7a5c10; margin-bottom: 6px; word-break: break-word; }
.ezfy-page .ezfy-ask-row { margin-top: 4px; }
/* 确认/取消做成有点击区的按钮样式：原来只是两个文字链接，手机上是「看不到的 1 字宽」的靶子，
   手指点不准（用户反馈「交互太离谱」）。 */
.ezfy-page .ezfy-ask-ok,
.ezfy-page .ezfy-ask-cancel {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 3px;
  border: 1px solid #c9c5ba;
  background: #fff;
}
.ezfy-page .ezfy-ask-ok { font-weight: bold; color: #27763c; border-color: #9cc4a6; margin-right: 10px; }
.ezfy-page .ezfy-ask-cancel { color: #777; }
.ezfy-page .ezfy-ask-ok:hover { background: #eef7f0; }
.ezfy-page .ezfy-ask-cancel:hover { background: #f2f2f2; }
/* 底部 15 项导航(复刻原版 cityHome.html 的两行) */
.ezfy-page .ezfy-bottom-nav {
  padding: 1px 0;
  /* ★ 2026-09-25 随全站统一：与首页导航同号（同一个变量） */
  font-size: var(--fs);
  line-height: 1.75;
}
/* ★ 间隔对齐原版 .old-line a 的 margin: 0 1px；配色按用户要求 默认 #004299 / 选中 #c0392b */
.ezfy-page .ezfy-bottom-nav a {
  display: inline;
  padding: 0 1px;
  margin: 0 1px;
  color: #004299;
}
.ezfy-page .ezfy-bottom-nav a.on {
  font-weight: bold;
  color: #c0392b;
}
/* 顶部导航里的「家园」——游戏内唯一的出口, 稍微标一下 */
.ezfy-page .top-nav a.ezfy-exit {
  color: #2f4156;
  font-weight: bold;
}
.ezfy-page .panel { margin-top: 8px; padding: 2px; }
.ezfy-page .acade-tab {
  padding: 3px 0;
  font-size: var(--fs);   /* ★ 2026-09-25 随全站统一（原来写死 17px） */
  color: #666;
}
.ezfy-page .acade-tab a { color: #2f4156; }
.ezfy-page .acade-tab a.on { color: #c0392b; font-weight: bold; }
/* ★ 2026-09-29 用户要求：司令部 tab(兵种配置/出征队列/伤兵营/逃兵营/预设编队) 间隔大一点点，
   仅这组生效（其余 acade-tab 不带 hq-tab 类，间隔保持不变） */
.ezfy-page .acade-tab.hq-tab a { margin-right: 10px; }
/* ★ 2026-09-28 tab 之间的「.」分隔符：原写法用 &nbsp;.&nbsp;（不换行空格 U+00A0），
   Windows 中文宋体(SimSun) 里 U+00A0 占一个**全角字宽**(17px)，两个就 34px，
   「军队动态 .  驻军 .  军情警讯」在 Win 上间隔拉到 40px+；
   macOS 回退到 Songti SC / 苹方，其 U+00A0 是半角宽度(约 4~5px)，所以看着正常。
   —— 根因是字形宽度差异，不是谁写错了数值。
   修法：不再引入空格字符，改用 span 承载「.」并显式给左右 margin，
   任何平台、任何字体下宽度都完全一致。 */
.ezfy-page .acade-sep {
  display: inline-block;
  margin: 0 4px;
}
/* ★ 2026-09-28 首页同一行里两个文字链接的间隔。
   ⚠️ 不能用空格字符控制间隔：`&nbsp;`(U+00A0) 在 Windows 宋体下是**全角**(约 16px)、
   在 macOS(Songti SC / 苹方) 下是**半角**(约 4px)，同一份代码两端差 4 倍；
   而源码里的换行只会折叠成 1 个半角空格(约 4px)，又太挤。
   统一改用定宽 span，任何平台/字体下宽度完全一致。
   用户反馈：「[造兵] [建防] 之间少一点点」（原来 &nbsp; 太宽）、
   「城市状态 / 附属野地 之间大一点点」（原来只有一个折叠空格，太挤）。 */
.ezfy-page .home-gap { display: inline-block; width: 10px; height: 1em; }
.ezfy-page .use-box {
  margin: 4px 0 6px 8px;
  padding: 4px 6px;
  border-left: 2px solid #d8d5cc;
  line-height: 1.9;
}
/* ★ 2026-09-29 城市列表：城市名(坐标) 可点击切换；当前城无下划线但保留粗体 */
.ezfy-page .old-line .city-name {
  font-weight: bold;
  color: var(--link, #004299);
  text-decoration: underline;
  cursor: pointer;
  margin-right: 8px;
}
.ezfy-page .old-line span.city-name {
  color: inherit;
  text-decoration: none;
  cursor: default;
}
/* ★ 建筑区操作的内联提示：独占一行贴在所点建筑行下方，带 [关闭]，不自动消失 */
.ezfy-page .build-tip {
  display: block;
  width: 100%;
  margin: 3px 0 4px;
  padding: 3px 8px;
  border-left: 3px solid #c0392b;
  background: #fbf3f2;
  color: #c0392b;
  line-height: 1.6;
  box-sizing: border-box;
}
.ezfy-page .build-tip.ok {
  border-left-color: #2e7d32;
  background: #eef7ee;
  color: #2e7d32;
}
.ezfy-page .build-tip a { margin-left: 8px; color: #999; }
.ezfy-page .panel-title {
  /* ★ 2026-09-25：小标题比正文大一号（正文已统一为 var(--fs)）。
     这是「标题 vs 正文」的层次，不是漏改 —— 要一起拉平说一声即可。 */
  font-size: calc(var(--fs) + 1px);
  font-weight: bold;
  color: #2f4156;
  margin: 6px 0 2px;
}
.ezfy-page .old-line { padding: 2px 0; word-break: break-all; }
/* ★ 建筑行「升级 / 一键满级 / 拆除」三个操作间隔再大一点（用户要求） */
.ezfy-page .build-act { display: inline-block; }
.ezfy-page .build-act a { margin: 0 5px; }
/* ★ 用户反馈「[赏赐…][流放] 这俩按钮之间来点间距」→ 军官详情的操作按钮行统一拉开间距。
   只作用在带 .officer-actions 的行上，不动其它页面的按钮。
   ★ 2026-09-28 用户要求「按钮样式去掉」→ 全部改文字链接，这里跟着改为 <a> 的间距。 */
.ezfy-page .old-line.officer-actions a { margin-right: 10px; }
.ezfy-page .old-line.officer-actions a:last-child { margin-right: 0; }
/* ★ 出征确认页(orderpre)分区：① ② ③ … 小标题 + 等宽列网格。
   原来所有内容都是一串 .old-line 平铺，兵种/资源/宿营/计算混在一起，用户反馈「看着好乱」。
   ★ 用户反馈「兵力还是竖着展示，整齐一点」→ 兵力改成 **CSS Grid 等宽列**：
     - 用 grid（不是 flex-wrap）：同一列的宽度完全一致，纵向也对得齐；
     - minmax 让窗口变窄时自动减列（窄屏也不会退化成一兵种一行）；
     - 名称超长用省略号（完整名字放 title），输入框固定宽度，**现有数量独立一列右对齐** ——
       原来是「输入框里塞占位符 0~59108」，框一窄就被截成 0-0，很难看。
   注意：.of-cell 仍保持 .old-line 的 2px 上下 padding，整页行距节奏不变。 */
.ezfy-page .of-sec {
  /* ★ 2026-09-25 随全站统一：写死 17px → var(--fs)（靠加粗+下虚线区分层次） */
  font-size: var(--fs);
  font-weight: bold;
  color: #2f4156;
  margin: 8px 0 2px;
  padding-bottom: 1px;
  border-bottom: 1px dashed #d8d5cc;
}
/* 分区标题里的补充说明（★ 2026-09-25 随全站统一：14 → var(--fs)，靠颜色+不加粗弱化） */
.ezfy-page .of-sec .of-hint { font-size: var(--fs); font-weight: normal; color: #8a8a8a; }
.ezfy-page .of-grid {
  display: grid;
  gap: 0 16px;
  align-items: center;
}
/* ★ 2026-09-25 用户要求「兵种数量搭配改成一行一个兵种」：
   原来是 3 列网格（一个兵种占一个格子，得横向找），现在**每个兵种独占一行**，行内从左到右：
     兵种名(定宽) │ 现有 N │ ————— 滑动条 ————— │ [数字框] │ [最大]
   滑块与数字框双向联动（拖滑块数字跟着变 / 填数字滑块跟着走），[最大] 一键全带。
   行内分两段：名称+现有（.of-name/.of-avail）与控件段（.of-ctl），
   窄屏时控件段整段换行（不会把滑块挤成 10px 没法拖）。 */
.ezfy-page .of-rows { display: block; }
.ezfy-page .of-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 0;
  line-height: 24px;
  min-width: 0;
}
.ezfy-page .of-row .of-name {
  /* 兵种名定宽 → 每行的「现有 N」和滑块都从同一个 x 开始，纵向严格对齐 */
  flex: 0 0 auto;
  width: 11em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  color: #555;
}
.ezfy-page .of-row .of-avail {
  /* 定宽放得下「现有 24,946,000」（线上资源数是十几位） */
  flex: 0 0 auto;
  width: 9.5em;
  text-align: left;
  color: #8a8a8a;
}
.ezfy-page .of-row .of-ctl {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 auto;
  min-width: 0;
}
.ezfy-page .of-row input.of-range {
  flex: 1 1 auto;
  min-width: 70px;
  height: 20px;
  margin: 0;
  /* 现代浏览器直接给滑块上主题色，不必再手写 ::-webkit-slider-thumb */
  accent-color: #2f6fb5;
  cursor: pointer;
}
.ezfy-page .of-row input.of-num {
  flex: 0 0 auto;
  width: 76px;
  text-align: right;
}
.ezfy-page .of-row .of-max { flex: 0 0 auto; white-space: nowrap; }
/* 城内没有该兵种：整行灰掉（滑块也禁用） */
.ezfy-page .of-row.of-off .of-name,
.ezfy-page .of-row.of-off .of-avail,
.ezfy-page .of-row.of-off .of-max { color: #b3b3b3; }
.ezfy-page .of-row.of-off input.of-range { opacity: .45; }
/* ★ 2026-09-28 出征上限额度已用尽（本兵种本次最多可派 0）：[最大] 点了也没用 → 灰掉，
   与同时被 :disabled 禁用的滑块/数字框保持一致，避免「点了没反应」的困惑。 */
.ezfy-page .of-row .of-max.of-max-off { color: #b3b3b3; cursor: default; }
/* ★ 2026-09-28 训练页 [最大]（紧跟在「(最多 N)」后面）：一键把建造数量填成上限。
   上限为 0 时同样灰掉 —— 与出征页 [最大] 的处理保持一致，避免「点了没反应」的困惑。 */
.ezfy-page .train-max { white-space: nowrap; margin-left: 6px; }
.ezfy-page .train-max.train-max-off { color: #b3b3b3; cursor: default; }
@media (max-width: 700px) {
  /* 窄屏：名称+现有占第一行，滑块/数字/[最大] 整段换到第二行 */
  .ezfy-page .of-row { flex-wrap: wrap; row-gap: 0; }
  .ezfy-page .of-row .of-name { width: auto; max-width: 60%; }
  .ezfy-page .of-row .of-avail { width: auto; flex: 1 1 auto; }
  .ezfy-page .of-row .of-ctl { flex: 1 1 100%; }
}
/* 随军资源：名称只有 2 个字，列可以窄一点。
   ★ 下限不能太小：线上资源是**十几位**的数（如 14,101,854,318 ≈ 119px），
     230px 的格子装不下「名称+输入框+数量」，数字会顶到隔壁格子的名称上（看着像串行）。
     300px = 名称 4em(64) + 输入 70 + 数量 119 + 间距 10 + 余量。 */
.ezfy-page .of-grid-res { grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); }
.ezfy-page .of-cell {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 2px 0;
  line-height: 24px;
  min-width: 0;
}
.ezfy-page .of-cell .of-name {
  /* ★ 用户要求「文字左最起，按照第一列兵种那块」：
     兵种名**定宽**（不随名字长短伸缩），这样每行的输入框都从同一个 x 开始，
     三列在所有行里纵向严格对齐 —— 这才是「左起对齐」。
     ★ 定宽还有一层必要性：第 3 列「现有数量」的数字长度是会变的
       （24,946,000 比 0 宽 15px）。名称若是弹性宽度，长数字会把输入框往左顶，
       同一列里输入框的 x 就不一致了（实测差 15px）—— 定宽才能钉死。
     11em 放得下线上最长的兵种名「齐柏林伯爵级航空母舰」= 10 个汉字(≈160px)，
     原来的 9.6em(154px) 会把最后两个字截成省略号（实测线上就是这样）。 */
  flex: 0 0 auto;
  width: 11em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  color: #555;
}
.ezfy-page .of-cell input.of-num {
  flex: 0 0 auto;
  width: 70px;
  /* ★ 用户要求「二列三列文字右对齐」：第 2 列（输入框里的数字）右对齐 */
  text-align: right;
}
/* 第 3 列「现有数量」：紧跟输入框、左起（不再 margin-left:auto 推到格子最右），
   因为输入框是定宽，所以这一列在所有行也从同一个 x 开始，自然对齐 */
.ezfy-page .of-cell .of-avail {
  flex: 0 0 auto;
  min-width: 4.4em;
  text-align: left;
  color: #8a8a8a;
}
/* 资源名只有 2 个字，定宽窄一点，别浪费横向空间 */
.ezfy-page .of-grid-res .of-cell .of-name { width: 4em; }
.ezfy-page .of-cell.of-off .of-name,
.ezfy-page .of-cell.of-off .of-avail { color: #b3b3b3; }
/* ★ 首页【置顶公告】区：公告行本身和其它 .old-line 一样是 28px 高，问题出在**上下留白不等**。
   上方那 4px 额外留白已由 .top-nav 去掉底部 padding 解决（见上），
   这里只需给下方补 2px，让公告行与上一行、下一行的留白相等。
   注意: 用 margin-bottom 而不是「抵消上方」的负 margin —— 因为导航和公告行之间还可能插入
   .ezfy-ask（操作结果提示条），负 margin 会把这 4px 从提示条的下边距里扣掉，
   公告行就会贴住提示条（实测只剩 5px）。 */
.ezfy-page .ezfy-notices { margin: 0 0 2px; }
/* ★ 2026-09-25 随全站统一：写死 17px → var(--fs) */
.ezfy-page .city-name { font-size: var(--fs); font-weight: bold; color: #2f4156; }
/* ★ 表格默认用「原版模板的朴素样式」: 无边框、字号对齐正文。
   ★ 2026-09-24 用户要求「表格固定宽度、切 tab 不因字数不一样变动」：
     table-layout:fixed + width:100% → 列宽按列数均分固定，与单元格内容完全无关，
     切换页面/翻页时列不跳；长文本由下方 th/td 的 word-break 自动折行。
     地图格子与战场指挥室两表内容结构特殊，保持原来自适应（见下方覆盖规则）。 */
.ezfy-page table {
  table-layout: fixed;
  width: 100%;
  /* ★ 2026-09-24 用户要求「表格靠左展示」：封顶 920px 左对齐，桌面不铺满整行 */
  max-width: 920px;
  margin: 0;
  border-collapse: collapse;
  /* ★ 表格字号对齐正文（2026-09-25 起 = var(--fs)，窄屏也不再单独缩一档） */
  font-size: var(--fs);
}
/* 地图 5×5 格子 / 战场指挥室：恢复按内容自适应，不参与全局固定均分 */
.ezfy-page .ezfy-map-table,
.ezfy-page .ezfy-battle-tbl {
  table-layout: auto;
  width: auto;
  max-width: 100%;
}
.ezfy-page table th,
.ezfy-page table td {
  word-break: break-word;
  overflow-wrap: anywhere;
  border: 0;
  text-align: left;
  vertical-align: top;
  padding: 2px 10px 2px 0;
}
/* ★ 军队总览「城内军队」表：只把单元格内容里左右居中 + 垂直居中（用户要求），表格本身不居中 */
.ezfy-page table.ezfy-center-tbl th,
.ezfy-page table.ezfy-center-tbl td {
  text-align: center;
  vertical-align: middle;
}
.ezfy-page table.ezfy-center-tbl td a { margin: 0 3px; }
/* ★ 2026-09-25 用户要求：「展示兵种名字的列」「军官装备名称的列」以前是整列居中，看着丑 →
   **单元格内容改成左对齐 + 垂直居中**；**表头 th 保持居中不变**。
   用法：给这些「名称类」单元格加 class="nm"（不是按列号，按语义，加列/挪列都不会失效）。
   放置位置：紧跟在 .ezfy-center-tbl / .ezfy-plain-table 的居中规则之后 ——
   选择器权重 (0,2,2) 与 .ezfy-center-tbl 相同、高于 .ezfy-plain-table 的 (0,2,1)，
   靠顺序取胜；以后调那两张表的居中规则不会把这个覆盖掉。
   ⚠️ 只加在「名称」列上（兵种名 / 装备名），数量·等级·属性·操作那些列保持原样，
      别图省事用 td:first-child —— 战场指挥室表的兵种在第 2 列，商城装备表的名称也在第 2 列。 */
.ezfy-page table td.nm {
  text-align: left;
  vertical-align: middle;
}
/* 名称列左对齐后，链接自带的左右 margin 会让文字比列边多缩 3px，去掉更齐 */
.ezfy-page table td.nm a { margin-left: 0; margin-right: 0; }
/* ★ 2026-09-25 当天追加：用户问「对应列标题是不是也居左更好」→ **是**。
   列宽是靠左的（兵种名/装备名都从左边起），表头却居中悬在列中间，两者不在一条竖线上，
   看着就是「标题和数据对不上」—— 表头对齐跟随列内容对齐是通行做法，这里也一样。
   ⚠️ 只打在**该列自己的 th** 上；像「已穿戴装备」「装备背包」那种 colspan 跨列的段标题
     保持居中不变（它是分节标题，不是列标题）。 */
.ezfy-page table th.nm { text-align: left; }
/* ★ 交易行表格美化（2026-09-24 用户要求「页面做好看点」）：细边框 + 表头底色 + 斑马纹 */
.ezfy-page table.ezfy-ex-tbl {
  border-collapse: collapse;
  border: 1px solid #cfc9b6;
  margin: 2px 0 4px;
}
.ezfy-page table.ezfy-ex-tbl th,
.ezfy-page table.ezfy-ex-tbl td {
  border: 0;
  text-align: left;
  vertical-align: middle;
  padding: 3px 10px;
  border-right: 1px solid #e3ded0;
}
.ezfy-page table.ezfy-ex-tbl th {
  background: #efe9d9;
  color: #2f4156;
  border-bottom: 1px solid #d9d2bd;
}
.ezfy-page table.ezfy-ex-tbl td { border-bottom: 1px solid #eee9dc; }
.ezfy-page table.ezfy-ex-tbl tr:last-child td { border-bottom: 0; }
.ezfy-page table.ezfy-ex-tbl tr:nth-child(even) td { background: #faf7ee; }
.ezfy-page table th {
  color: #2f4156;
  font-weight: bold;
  white-space: nowrap;
  padding-bottom: 4px;
}
/* ★ WAP 窄屏：固定均分后列会变窄，表头放开折行，长表头（如「装备战斗加成」）不被截断/溢出 */
@media (max-width: 700px) {
  .ezfy-page table th { white-space: normal; word-break: break-word; }
}
/* 返回按钮与 [造兵]/[建防]/[退出军团] 等普通操作链接同款: 纯文字链接, 无填充 */
.ezfy-page .bottom-nav { margin-top: 10px; padding: 4px 0; text-align: left; }
.ezfy-page .footer { text-align: center; font-size: var(--fs); color: #999; padding: 4px 0 10px; }
.ezfy-page .logo-title { height: 14px; vertical-align: -2px; }
/* ★ 首页功能图标(军团/军衔/资源/人口/民心/税率)：统一用内联 SVG 圆角徽章，替换原来风格杂乱的 PNG */
.ezfy-page .ezfy-ico { width: 18px; height: 18px; vertical-align: -4px; margin-right: 3px; }
.ezfy-page .red { color: #c0392b; }
.ezfy-page .gray { color: #999; }
.ezfy-page .green { color: #27763c; }
.ezfy-page .orange { color: #b8860b; }
.ezfy-page .q-normal { color: #999; }
.ezfy-page .q-rare { color: #4dabff; }
.ezfy-page .q-epic { color: #9b59b6; }
.ezfy-page .q-legend { color: #e67e22; }
.ezfy-page input[type="text"],
.ezfy-page input[type="number"],
.ezfy-page input:not([type]),
.ezfy-page select,
.ezfy-page textarea {
  border: 1px solid #999;
  border-radius: 0;
  padding: 3px 4px;
  /* ★ 2026-09-25 随全站统一：表单 16 → var(--fs)（用户要求所有文字一样大） */
  font-size: var(--fs);
  background: #fff;
  color: #333;
}
.ezfy-page button {
  border: 1px solid #888;
  border-radius: 0;
  background: #e8e5dd;
  color: #333;
  /* ★ 2026-09-25 随全站统一：按钮 15 → var(--fs)（用户要求所有文字一样大） */
  font-size: var(--fs);
  padding: 2px 8px;
  cursor: pointer;
}
.ezfy-page button:hover { background: #d8d5cc; }
.ezfy-page .report-pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: inherit;
  /* ★ 2026-09-25 随全站统一：战报正文 15 → var(--fs) */
  font-size: var(--fs);
  background: #fff;
  border: 1px solid #ddd;
  padding: 6px;
  margin: 4px 0;
}
/* 地图 */
.ezfy-map { padding: 4px 0; overflow-x: auto; }
.ezfy-map-row { white-space: nowrap; }
/* 复刻 map/index.html 的 5×5 <table>:
   参考**没有任何表格 CSS**, 就是浏览器默认样式 —— 单元格按内容自适应宽度、
   无底色、无边框。所以这里只做三件事:
   ① 抵消全局 `.ezfy-page table td` 的虚线下边框  ② 单元格紧凑  ③ 不折行
   ★ 用户要求「格子下面加个坐标，排列整齐一点」→ 每格变成**两行**：
     第一行 名称(等级)，第二行 (x,y)；见下面的 .ezfy-cell-name / .ezfy-cell-xy。 */
.ezfy-page .ezfy-map-table {
  /* ★ 2026-09-25 用户反馈「点击地图表格列宽会变，向上/向下切换时因为字数不一样看着丑」：
     原因 = 浏览器默认的 auto 布局按每格内容宽度分配列宽（「城市」1 格 vs 「海底森林(5)」6 字），
     移动地图/内容一变列宽就跳。改成 table-layout: fixed → 5 列恒等宽，位置稳定。
     ★★ 同一天用户二次纠正：「列宽固定对了，但表格变大了，表格还和之前一样只不过列宽固定」。
        上一版写的是 width:100% + min-width:660px，表格被撑满整个面板（桌面实测 964px、
        列宽 178px）—— 这就是「表格变大」的原因。现在改成**固定总宽**（2026-09-26 定档 582px）：
          改动前：325px(全是短文案) ~ 361px(出现「海底森林(9)」) ← 会随内容变宽变窄（用户不要）
          现在：  恒定宽，fixed 布局把 5 列均分成 90px/列       ← 只由宽度公式决定，永不随内容变
        90px 这一档是 2026-09-26 用户拍板的：他要求「海底森林(10)」**必须一行显示**，
        「不满足的话左右就再加」。所以列宽按最坏情况量 —— 最长的两种文案「海底森林(10)」
        「沿海平原(10)」都是「4 个汉字 + (10)」= 15px×4 + 括号数字 ≈ 86~88px，90px 留余量；
        比它短的必然一行（「平原(10)」58px、「活动寇(9)」68px、「寇(10)」43px）。
        ★ 中途试过「列宽 64px + 名称折两行」（表只要 446px），用户明确要一行，已弃用该方案。
     ★ 列宽能真正定死，下面两条缺一不可（都是实测踩出来的）：
       ① 表格宽度必须是**确定值**，不能 width:auto —— fixed 布局 + width:auto 时 Chrome 仍会
          用内容的 min-content 把列撑宽（实测 325px 的表会被「海底森林(10)」涨到 436px），前功尽弃；
       ② 格子里那个 <a> 必须是 display:block —— inline-block 的 shrink-to-fit 会取 min-content
          （= 整串文字宽 86px），直接溢出格子压到邻格，也不会按格子宽度换行（见下面 a 的规则）。 */
  table-layout: fixed;
  width: 582px;
  min-width: 0;
  border-collapse: separate;
  /* ★ 用户要求「坐标和坐标之间间隔小了，上下左右都再来点」→ 8px 3px 放大到 12px 6px；
     随后又要求「上下间隔加一点」→ 纵向 6px → 10px；再次要求「上下坐标间隔再大一些」
     → 纵向 10px → 14px；2026-09-25 又反馈「地图区上下还是紧，上下间距再大一点点」
     → 纵向 14px → 18px；当天看过对比图后**拍板用 22px 那档** → 18px → **22px**。
     ★★ 2026-09-26 用户要求「地图坐标左右之间的间隔大一点，但**第一列左边间隔保持不变**」，
        横向最终定档 **22px**（中途试过 20px，用户看过之后说「-2px 去掉吧」→ 回到 22）。
        border-spacing 是**内外一起加**的 —— 第一个值同时管「列与列之间」和「表格左右两边的
        外边距」，所以横向一加大，第一列也会跟着右推；这里用 margin-left 把整表左移抵消
        （负值只吃到表格最左边那段空白，第一列内容仍在表格内）：
          · 第一列左边距 = margin-left + 横向间距 = 12px  ← 与最开始完全一致 ✔
          · 列与列之间   = 横向间距：12 → 20 → **22**（定档）
          · 总宽 = 6×横向间距 + 5×列宽（90，见上面「列宽」那段）→ 6×22 + 5×90 = 582px
        ★ 要再调间距：只改第一个数，另两处按这两条式子跟着改：
            width = 6×横向间距 + 5×90      margin-left = 12 − 横向间距
          窄屏那份在下面的 @media (max-width: 420px) 里（横向 8px、margin-left -2px）。 */
  border-spacing: 22px 22px;
  margin: 8px 0 8px -10px;   /* 左 -10px = 12 − 22，抵消第一列被 border-spacing 右推 */
}
.ezfy-page .ezfy-map-table td {
  padding: 0;
  border: 0;
  text-align: center;        /* 居中指的是「表格里的内容」居中 */
  vertical-align: top;       /* 两行格子按顶对齐, 免得行高不一导致上下抖动 */
  white-space: nowrap;
}
.ezfy-page .ezfy-map-table a {
  /* ★ 必须是 block，不能是 inline-block：fixed 布局下格子宽度是定死的，
     inline-block 的 shrink-to-fit 会取 min-content(=整串文案宽，如「海底森林(10)」83px)，
     结果 <a> 比格子还宽 → 文字压到邻格、省略号也不生效。改成 block 后宽度=格子宽，
     里面的 .ezfy-cell-name 才能正常 overflow:hidden + 省略号。 */
  display: block;            /* 块级容器, 才能装上下两行 + 按格子宽度截断 */
  padding: 0;
  margin: 0;
  /* ★ 第一行(名称/等级，如「海(8)」)的字号：历史 16→15→14，2026-09-25 用户反馈「地图看着小了」
     → 加大 1 号回到 **15px**。坐标行有自己独立的值(14px)，不受这里影响。
     窄屏同理 12 → 13，见下面媒体查询。 */
  font-size: 15px;
  line-height: 1.3;
  /* ★ 用户要求「坐标上颜色 + 野地类型也上色，不然玩家不知道能点」→ 两行都用站内链接蓝；
     本城(.ezfy-mine)与活动目标(.ezfy-act-*)的颜色是有含义的，下面单独覆盖，不受影响。 */
  color: #0645ad;
  background: none;
  border: 0;
  text-decoration: none;
  text-align: center;        /* 两行都相对格子中心对齐 */
}
/* 第一行：名称(等级) */
/* ★ 2026-09-26 用户要求「海底森林(10)」**一行显示**（先前是 nowrap + 省略号，
   被截成「海底森…」，等级跟着看不见）。列宽已按最长文案放到 90px（见上面「列宽」那段），
   所有文案天然一行，这里不再需要省略号；white-space:normal 只作兜底：万一某台机器字体更宽，
   宁可折行也不截断或压到邻格（CSS 不允许在「(」后断行，折也是折成「海底森林」/「(10)」）。 */
.ezfy-page .ezfy-map-table a .ezfy-cell-name {
  display: block;
  white-space: normal;
}
/* 第二行：坐标 (x,y)。★ 用户要求「坐标上颜色，不然玩家不知道能点」→ 站内链接蓝 #0645ad；
   字号定稿过程：12 → 11 →「坐标那行大 1 号」12 →「(272,227) 大 1 号」13px；
   2026-09-25 用户反馈「地图坐标看着小了」→ 加大 1 号到 **14px**。
   ★ margin-top 是用户要求「上下坐标之间再大一点点」——第一行缩到 14 后两行几乎一样大，
   需要这点缝把它们分开，不然两行糊成一块。 */
.ezfy-page .ezfy-map-table a .ezfy-cell-xy {
  display: block;
  font-size: 14px;
  line-height: 1.25;
  margin-top: 4px;
  font-weight: normal;       /* 本城/活动城名字加粗, 坐标不跟着加粗 */
  color: #0645ad;
}
/* 本城加粗标一下, 其余一律朴素文字 */
.ezfy-page .ezfy-map-table a.ezfy-mine { font-weight: bold; color: #c0392b; }
/* 活动目标配色照 mapView.html: 活动野地橙 / 活动寇城品红 / 特殊城市红 */
.ezfy-page .ezfy-map-table a.ezfy-act-wild { font-weight: bold; color: #ff6600; }
.ezfy-page .ezfy-map-table a.ezfy-act-kou { font-weight: bold; color: #ff00ff; }
.ezfy-page .ezfy-map-table a.ezfy-act-city { font-weight: bold; color: #d00000; }
/* ★ 2026-09-30 带名将守将的活动野地：紫红高亮，与普通活动野地(橙)区分 */
.ezfy-page .ezfy-map-table a.ezfy-act-named { font-weight: bold; color: #cc22ff; text-decoration: underline; }
/* ★ 地图方向导航「向上/向右/向下/向左/回到本城」：
   全局 .ezfy-page a 只有 margin: 0 1px, 五个词挤成一串。用户要求「间隙稍微大一点」
   → 每个链接右侧留 8px（含标签间空格约 12px 一档），末项不留，右侧不至于飘出去。 */
.ezfy-page .ezfy-dir-nav a {
  display: inline-block;
  margin: 0 8px 0 0;
}
.ezfy-page .ezfy-dir-nav a:last-child { margin-right: 0; }
/* ★ 坐标查找行：横坐标/纵坐标同一行，[查找] 跟行末；窄屏由下面的媒体查询收窄输入框 */
.ezfy-page .ezfy-map-jump input { width: 80px; margin-right: 4px; }
.ezfy-cell {
  display: inline-block;
  width: 36px;
  height: 26px;
  line-height: 26px;
  text-align: center;
  margin: 1px;
  background: #e8f0d8;
  border: 1px solid #b8c89a;
  font-size: 11px;
  cursor: pointer;
  color: #444;
  white-space: nowrap;
}
.ezfy-mine { background: #ffe9b0; border-color: #d0a030; color: #803000; }
.ezfy-city { background: #d8e4f0; border-color: #90a8c0; }
.ezfy-kou { background: #f0d8d8; border-color: #c09090; }
.ezfy-sea { background: #c8e0f0; border-color: #80a8c8; }
.ezfy-wild { background: #e8f0d8; border-color: #b8c89a; }
/* ★ 2026-10-01 地图外越界格：灰色「空地」（不可点），不再显示负坐标地形 */
.ezfy-page .ezfy-map-table .ezfy-empty {
  display: block;
  padding: 0;
  margin: 0;
  font-size: 15px;
  line-height: 1.3;
  color: #b0b0a8;
  background: #f4f4f0;
  border: 0;
}

/* ============ WAP 窄屏适配(手机) ============
   目标: 360px / 320px 下不出现横向溢出, 表格不挤成一坨。
   实测基准: iPhone SE 320、常见安卓 360/390。
   ★ 2026-09-25 用户要求「页面文字除了地图全部和首页导航一样大」→
     **这里不再整体缩一档**（原来 表格14/正文15/标题16/导航15 那套阶梯已删除），
     正文·标题·导航·表格·按钮·表单一律沿用 var(--fs)。
   仍然保留缩小的只有两类（属于布局硬约束，不缩就必然溢出）：
     ① 地图格（5 列 × 两行文字）
     ② 战场指挥室表格（7 列，见文件末尾它自己的媒体查询）
   另保留表格单元格内边距（省空间，与字号无关）。 */
@media (max-width: 420px) {
  .ezfy-page table th,
  .ezfy-page table td { padding: 4px 4px; }
  /* 地图格子: 间距按窄屏收紧, 保证 320px 下 5 列不溢出
     ★ 格子已是两行(名称 + 坐标)，窄屏两行都缩一档，行高收紧免得整表变高太多；
       第一行跟着桌面一起缩(15 → 13)，坐标行同样随桌面(14 → 13)，两行之间留同样的缝。 */
  .ezfy-page .ezfy-map-table a { font-size: 12px; line-height: 1.25; }
  .ezfy-page .ezfy-map-table a .ezfy-cell-xy { font-size: 12px; }
  /* 窄屏纵向间距同步收一档(桌面 22px → 窄屏 12px)：纵向间距只影响表格高度、不影响列宽，
     所以这里不需要像横向那样压到极限，留出和桌面接近的呼吸感 */
  /* ★ 2026-09-26 桌面把横向间距加大后，用户反馈「手机端没同步，适配一下」：
     窄屏横向也同步加宽一档 —— 6px → **8px**，并把整表左移 2px 抵消（margin-left = 6 − 8），
     这样**第一列左边距仍是 6px**、和以前一致，只有列与列之间变宽。
     总宽 = 6×8 + 5×52 = 308px（320px 的机器上仍放得下，不触发 .panel 横向滚动）。
     ★ 窄屏名称行仍是「不折行 + 省略号」：列宽只有 ~52px，装不下「海底森林(10)」
       （13px 下约 75px），硬折行会折在名字中间（「海底森」/「林(10)」）反而更难看。
     ★★ 2026-09-27 用户反馈「手机端进海底森林地名展示不全」：name 被省略号截成「海底森…」。
         根因是 5 列 @52px 在 320px 宽的窄屏上无论如何都装不下「海底森林(10)」。
         这次把列宽提到能装下名字的一档（~76px），并顺手把左右间距 8px → **10px**：
         总宽 = 6×10 + 5×76 = 440px > 屏幕宽，由 .panel 的 overflow-x:auto 兜底横向滑动，
         好处是地名整条显示（不再省略）、间距更松；代价是窄屏要看全 5 列需左右滑动。
         （若更希望窄屏不滑动、整表一屏放下，则必须接受地名省略，二选一。） */
  .ezfy-page .ezfy-map-table { width: 440px; border-spacing: 10px 12px; margin-left: -4px; }
  /* ★★ 2026-09-27 地名还是被省略号截掉(「海底森…」)：根本原因是上面的 ellipsis 规则在
     窄屏仍生效，而「海底森林(9/10)」≈73~80px，卡在 76px 列宽边界内会因 padding 超宽被截。
     这里在窄屏直接覆盖成**不省略、格子内能折行**：宁可折行也绝不丢字；
     配合字体 13→12px 后「海底森林(10)」仅约 64px，可一行放下、基本不会触发折行。 */
  .ezfy-page .ezfy-map-table a .ezfy-cell-name {
    overflow: visible; text-overflow: clip; white-space: normal; word-break: break-all;
  }
  /* 坐标查找行在 320px 下也要待在一行内 */
  .ezfy-page .ezfy-map-jump input { width: 62px; margin-right: 2px; }
  /* 方向导航窄屏间距同步收一档(桌面 8px → 窄屏 6px) */
  .ezfy-page .ezfy-dir-nav a { margin-right: 6px; }
  .ezfy-page input, .ezfy-page select { max-width: 100%; }
  /* 出征页兵力行：320px 下名称列收窄，滑块才有拖动空间（字号不缩，仍 var(--fs)） */
  .ezfy-page .of-row .of-name { width: 9em; }
  .ezfy-page .of-row input.of-num { width: 64px; }
  /* 随军资源格子：名称+输入框+现有数量在 320px 下要放得下 */
  .ezfy-page .of-cell input.of-num { width: 56px; }
  .ezfy-page .of-cell .of-avail { min-width: 3.6em; }
}
/* 最后一道保险: 万一还有个别元素偏宽, 让它在页面内滚动而不是把整页撑开 */
.ezfy-page .panel { max-width: 100%; overflow-x: auto; }
/* ============ 军团三页（军团信息 / 军团外交 / 军团宣战）WAP 适配（2026-09-28 用户要求） ============
   手机上：宽表格（成员 5 列 / 外交 7 列 / 宣战 7 列）挤成一坨 → 隐藏次要列 + 收紧内边距；
   输入框/下拉（内联 width 15%~60%）过宽 → 收窄并限制 max-width。 */
@media (max-width: 700px) {
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td { padding: 4px 4px; }
  /* ★ 2026-09-28 军团信息/外交/宣战 多列表格：手机端改为横向滚动, 不再硬塞窄屏挤压错位;
       单元格不换行, 超出面板左右滑动即可看全, 布局不乱 */
  .ezfy-page .ezfy-corps-tbl,
  .ezfy-page .ezfy-dip-tbl,
  .ezfy-page .ezfy-war-tbl,
  .ezfy-page .ezfy-war-list-tbl {
    display: block;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    white-space: nowrap;
    width: 100%;
  }
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td,
  .ezfy-page .ezfy-dip-tbl th,
  .ezfy-page .ezfy-dip-tbl td,
  .ezfy-page .ezfy-war-tbl th,
  .ezfy-page .ezfy-war-tbl td,
  .ezfy-page .ezfy-war-list-tbl th,
  .ezfy-page .ezfy-war-list-tbl td {
    white-space: nowrap;
  }
  /* 输入框/下拉：手机上一律收窄（!important 覆盖内联宽度） */
  .ezfy-page select.corps-kick-sel { width: 55vw !important; max-width: 200px; }
  .ezfy-page input.corps-mail-input { width: 55vw !important; max-width: 240px; }
  .ezfy-page input.corps-msg-input { width: 40vw !important; max-width: 180px; }
  .ezfy-page input.corps-name-input { width: 45vw !important; max-width: 180px; }
  /* ★ 2026-09-28 军团信息/外交/宣战：手机端输入类占整行, 按钮自动换行, 不再窄屏挤一起 */
  .ezfy-page input.corps-mail-input,
  .ezfy-page input.corps-msg-input,
  .ezfy-page input.corps-name-input,
  .ezfy-page select.corps-kick-sel {
    display: block;
    margin: 3px 0;
  }
  /* 外交「全部军团」7 列：藏次要列「团长」「积分」 */
  .ezfy-page .ezfy-dip-tbl th:nth-child(2),
  .ezfy-page .ezfy-dip-tbl td:nth-child(2),
  .ezfy-page .ezfy-dip-tbl th:nth-child(4),
  .ezfy-page .ezfy-dip-tbl td:nth-child(4) { display: none; }
  /* 宣战「记录」7 列：藏次要列「我方身份」「宣告时间」 */
  .ezfy-page .ezfy-war-tbl th:nth-child(2),
  .ezfy-page .ezfy-war-tbl td:nth-child(2),
  .ezfy-page .ezfy-war-tbl th:nth-child(4),
  .ezfy-page .ezfy-war-tbl td:nth-child(4) { display: none; }
  /* 宣战「全部军团(可宣战)」5 列：藏「团长」 */
  .ezfy-page .ezfy-war-list-tbl th:nth-child(2),
  .ezfy-page .ezfy-war-list-tbl td:nth-child(2) { display: none; }
}
@media (max-width: 420px) {
  .ezfy-page .ezfy-corps-tbl { font-size: 12px; }
  .ezfy-page .ezfy-corps-tbl th,
  .ezfy-page .ezfy-corps-tbl td { padding: 3px 3px; }
  /* 成员表「任命」列（军团长：[副团长][参谋长][撤职]）竖排，加大触点 */
  .ezfy-page .ezfy-mem-tbl td a { display: inline-block; margin: 2px 0; }
  /* 外交「全部军团」再藏「人数」 */
  .ezfy-page .ezfy-dip-tbl th:nth-child(3),
  .ezfy-page .ezfy-dip-tbl td:nth-child(3) { display: none; }
  /* 宣战「记录」再藏「对方战绩」 */
  .ezfy-page .ezfy-war-tbl th:nth-child(6),
  .ezfy-page .ezfy-war-tbl td:nth-child(6) { display: none; }
}
/* ============ 战场指挥室 WAP 适配 ============
   指挥室表格 7 列（方/兵种/剩余/初始/位置/目标/指挥）在手机上挤成一坨，
   窄屏逐步收缩：≤700px 藏「初始」、指挥按钮竖排加大触点；≤420px 再藏「位置」、下拉收紧。 */
@media (max-width: 700px) {
  .ezfy-page .ezfy-battle-tbl th:nth-child(4),
  .ezfy-page .ezfy-battle-tbl td:nth-child(4) { display: none; }
  .ezfy-page .ezfy-battle-tbl td.ezfy-cmd a {
    display: block;
    margin: 4px 0;
    font-size: 15px;
  }
  .ezfy-page .ezfy-battle-cmds a {
    display: inline-block;
    margin: 4px 10px 4px 0;
    font-size: 15px;
  }
  .ezfy-page .ezfy-battle-tbl th,
  .ezfy-page .ezfy-battle-tbl td { padding: 6px 5px; }
}
@media (max-width: 420px) {
  .ezfy-page .ezfy-battle-tbl th:nth-child(5),
  .ezfy-page .ezfy-battle-tbl td:nth-child(5) { display: none; }
  .ezfy-page .ezfy-battle-tbl { font-size: 12px; }
  .ezfy-page .ezfy-battle-tbl select {
    width: 76px !important; /* 覆盖内联 width:96px */
    max-width: 24vw;
    font-size: 12px;
    padding: 1px 0;
  }
}
/* ★ 军情三区分页条（军队动态 / 军情警讯 / 战斗报告，默认每页 5 条） */
.ezfy-page .ezfy-pager {
  margin: 8px 0 4px;
  /* ★ 2026-09-25 随全站统一：分页条 15 → var(--fs)（用户要求所有文字一样大） */
  font-size: var(--fs);
}
.ezfy-page .ezfy-pager a {
  margin: 0 4px;
  text-decoration: none;
}
.ezfy-page .ezfy-pager a.gray {
  color: #999;
  pointer-events: none;
}
.ezfy-page .ezfy-pager span.gray {
  margin: 0 6px;
}
/* ★ 2026-09-25 随全站统一：分页条窄屏也不缩（原来 13px） */
</style>
