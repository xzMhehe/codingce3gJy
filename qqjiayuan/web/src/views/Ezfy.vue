<template>
  <div class="ezfy-page">
    <div class="home-wrap">
      <div class="title-bar">二战征途【内测群: 431442049】</div>

      <!-- 顶部导航(每页都有) -->
      <div class="top-nav">
        <a href="javascript:;" @click="go('chat')">聊天</a>
        <a href="javascript:;" @click="openPm()">邮箱</a>
        <a href="javascript:;" @click="go('reports')">军情</a>
        <a href="javascript:;" @click="go('tasks')">任务</a>
        <a href="javascript:;" @click="go('friends')">好友</a>
        <a href="javascript:;" @click="go('home')">首页</a>
      </div>

      <!-- 页面内操作结果（代替 alert 弹窗；原版本来就没有弹窗交互） -->
      <div class="ezfy-msgs" v-if="msgs.length">
        <div v-for="m in msgs" :key="m.id" class="ezfy-msg" :class="'ezfy-msg-' + m.type">
          {{ m.text }}
          <a href="javascript:;" class="ezfy-msg-close" @click="closeMsg(m.id)">[关闭]</a>
        </div>
      </div>

      <!-- 页面内确认条（代替 confirm/prompt 弹窗） -->
      <div class="ezfy-ask" v-if="askBox.show">
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
        <a href="javascript:;" :class="{ on: cur === 'acade' }" @click="go('acade')">军官</a>.
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
          <img class="logo-title" src="/static/ezfy/corps.png" title="军团" alt="."/>
          军团：
          <a href="javascript:;" @click="go('corps')" v-if="!myCorps">加入军团</a>
          <a href="javascript:;" @click="go('corps')" v-else>[{{ myCorps.name }}]</a>
        </div>
        <div class="old-line">声望：{{ profile.prestige }}</div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/jx.png" title="军衔" alt="."/>
          <a href="javascript:;" @click="go('rank')">军衔</a>:{{ rankName }}
        </div>
        <div class="old-line">每日签到：<a href="javascript:;" @click="go('welfare')">{{ welfare.signed_today ? '已签到' : '签到' }}</a></div>

        <div class="old-line">
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
          <img class="logo-title" src="/static/ezfy/gold.png" :title="resNames.gold" alt="."/>
          <a href="javascript:;" @click="go('res/gold')">{{ resNames.gold }}:</a>{{ city.gold }}/{{ city.gold_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/rice.png" :title="resNames.food" alt="."/>
          <a href="javascript:;" @click="go('res/food')">{{ resNames.food }}:</a>{{ city.food }}/{{ city.food_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/steel.png" :title="resNames.steel" alt="."/>
          <a href="javascript:;" @click="go('res/steel')">{{ resNames.steel }}:</a>{{ city.steel }}/{{ city.steel_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/oil.png" :title="resNames.oil" alt="."/>
          <a href="javascript:;" @click="go('res/oil')">{{ resNames.oil }}:</a>{{ city.oil }}/{{ city.oil_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/mine.png" :title="resNames.rare" alt="."/>
          <a href="javascript:;" @click="go('res/rare')">{{ resNames.rare }}:</a>{{ city.rare }}/{{ city.rare_cap }}
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/person.png" title="人口" alt="."/>
          人口/空闲:{{ city.pop }}/{{ freePop }}
          <a href="javascript:;" @click="go('convene')">召集</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/feelings.png" title="民心" alt="."/>
          民心/民怨:{{ city.feelings }}/{{ city.grievance }}
          <a href="javascript:;" @click="go('placate')">安抚</a>
        </div>
        <div class="old-line">
          <img class="logo-title" src="/static/ezfy/tax.png" title="税率" alt="."/>
          <a href="javascript:;" @click="go('taxset')">税率:</a>{{ city.tax_rate }}%
        </div>

        <div class="old-line">
          <a href="javascript:;" @click="go('buildm')">军事区</a>&nbsp;
          <a href="javascript:;" @click="openBuildPre('m')">建造</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('builds')">资源区</a>&nbsp;
          <a href="javascript:;" @click="openBuildPre('s')">建造</a>
        </div>
        <div class="old-line">
          训练军队
          <a href="javascript:;" @click="go('troop')">[造兵]</a>&nbsp;
          <a href="javascript:;" @click="go('defence')">[建防]</a>
        </div>
        <div class="old-line">
          前往
          <a href="javascript:;" @click="go('map')">地图</a>
          出征
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('techs')">研究科技</a>
        </div>
        <div class="old-line">
          <a href="javascript:;" @click="go('citystatus')">城市状态</a>
          <a href="javascript:;" @click="go('wilds')">附属野地</a>
        </div>
        <div class="old-line">【世界聊天】<a href="javascript:;" @click="go('chat')">进入</a></div>
        <!-- [世界] 安珞：11111 / [军团] / [私聊] / [系统]; 昵称用实时昵称+个性颜色 -->
        <div class="old-line" v-for="ch in homeChats" :key="'wc' + ch.key">
          [<span class="orange">{{ ch.tag }}</span>]
          <a href="javascript:;" @click="openPlayer(ch.user_id)"><span
             v-for="(c, ci) in nickChars(ch.user_name)" :key="'nc' + ci"
             :style="nickColorAt(ch.color, ci)">{{ c }}</span></a>：{{ ch.content }}
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
              <button v-if="chatCooldown <= 0" @click="doChatSend">发送</button>
              <button v-else disabled class="gray">冷却中 {{ chatCooldown }}s</button>
              <span class="gray">每次发言消耗一个喇叭(最大25个字)</span>
            </template>
            <span v-else class="gray">(系统频道仅系统可发言)</span>
            <button @click="loadChats">刷新</button>
          </div>

          <!-- 系统频道: 系统公告 + 系统消息(只读) -->
          <template v-if="chatChannel === 4">
            <div class="panel-title">系统公告</div>
            <div class="old-line" v-for="n in chatNotices" :key="'cn' + n.id">
              <span v-if="n.is_top" class="red">[置顶]</span>
              <b>{{ n.title }}</b><br/>
              <span class="gray">{{ n.content }}</span>
            </div>
            <div class="old-line gray" v-if="!chatNotices.length">(暂无系统公告)</div>
            <div class="panel-title">系统消息</div>
            <div class="old-line" v-for="ch in worldChats" :key="'cs' + ch.id">
              [<span class="orange">系统</span>]
              <span class="gray">{{ fmtTime(ch.created_at) }}</span>
              {{ ch.user_name }}：{{ ch.content }}
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
          <div class="old-line gray">
            点一个会话就切到和那个人的聊天；在「世界聊天 / 好友 / 统帅信息」里点玩家名字也会直接进到和他的私聊。
          </div>
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
          <button @click="loadMails">刷新</button>
        </div>
      </template>

      <!-- ============ 情报/军情(reports) ============ -->
      <template v-else-if="cur === 'reports'">
        <div class="panel">
          <!-- 复刻 report/index.html: 军队动态 . 军情警讯 . 战斗报告 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: reportTab === 1 }" @click="switchReportTab(1)">军队动态</a>&nbsp;.&nbsp;
            <a href="javascript:;" :class="{ on: reportTab === 2 }" @click="switchReportTab(2)">军情警讯</a><span
              v-if="reportCounts[1]" class="red">({{ reportCounts[1] }})</span>&nbsp;.&nbsp;
            <a href="javascript:;" :class="{ on: reportTab === 3 }" @click="switchReportTab(3)">战斗报告</a><span
              v-if="reportCounts[2]" class="red">({{ reportCounts[2] }})</span>
          </div>

          <!-- ===== 军队动态: 所有在外的部队(出征/采集/派遣/侦查/掠夺/运输/增援) ===== -->
          <template v-if="reportTab === 1">
            <div class="old-line">
              <a href="javascript:;" @click="doCollectAll">一键采集</a>
              <a href="javascript:;" @click="doHarvestAll">一键收获</a>
              <a href="javascript:;" @click="doRecallAll">一键召回</a>
            </div>
            <div class="old-line gray">
              「收获」只把产出装进部队；资源要「召回」并返航到达才会运回城里（受负重限制）。宝物直接进背包。
            </div>
            <div class="old-line" v-for="o in dynPaged" :key="'dy' + o.id">
              命令：{{ o.type_name }} <a href="javascript:;" @click="openOrder(o)">查看</a><br/>
              目标：{{ o.target_name }}({{ o.target_x }},{{ o.target_y }})<br/>
              状态：{{ o.status_name }}<br/>
              军官：{{ o.officer || '无' }}<br/>
              {{ o.time_label }}：{{ o.time_text }}<br/>
              <span v-if="o.carry_total > 0" class="green">
                待带回：{{ fmtN(o.carry.food) }}粮/{{ fmtN(o.carry.steel) }}钢/{{ fmtN(o.carry.oil) }}油/{{ fmtN(o.carry.rare) }}稀/{{ fmtN(o.carry.gold) }}金
                （负重 {{ fmtN(o.carry_total) }}/{{ fmtN(o.carry_cap) }}）
              </span>
              <span v-else-if="o.order_type === 7 || o.order_type === 4" class="gray">待带回：暂无</span>
              <br/>
              --------------------
            </div>
            <div class="old-line" v-if="!dynamics.length">(当前没有在外的部队)</div>
            <div class="ezfy-pager" v-if="dynamics.length > dynSize">
              <a href="javascript:;" :class="{ gray: dynPage <= 1 }" @click="pagerGo('dyn', -1)">上一页</a>
              <span class="gray">第 {{ dynPage }}/{{ dynTotalPages }} 页（共 {{ dynamics.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: dynPage >= dynTotalPages }" @click="pagerGo('dyn', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 军情警讯: 别人打我 ===== -->
          <template v-else-if="reportTab === 2">
            <div class="old-line">
              <span class="gray">敌方来袭预警、被侦查、被掠夺、被征服都在这里看；能不能提前看见、能看见多少细节，取决于自己城市的雷达站等级。</span>
              <button @click="loadReports">刷新</button>
            </div>
            <!-- ★ 雷达站决定「事前预警」能不能收到（事后结果战报不受影响） -->
            <div class="old-line" v-if="reportRadar > 0">
              <span class="gray">当前雷达站 {{ reportRadar }} 级：已开启「敌军来袭 / 被侦查」预警，等级越高情报越详细。</span>
            </div>
            <div class="old-line" v-else>
              <span class="red">尚未建造雷达站：收不到「敌军来袭 / 被侦查」预警；被掠夺、被征服的结果战报仍会记录在这里。</span>
            </div>
            <div class="old-line" v-for="r in repPaged" :key="'rw' + r.id">
              <a href="javascript:;" @click="openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>{{ r.title }}</a>
              <span class="gray">({{ fmtTime(r.created_at) }})</span>
            </div>
            <div class="old-line" v-if="!reports.length">(暂无军情警讯)</div>
            <div class="ezfy-pager" v-if="reports.length > repSize">
              <a href="javascript:;" :class="{ gray: repPage <= 1 }" @click="pagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ repPage }}/{{ repTotalPages }} 页（共 {{ reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: repPage >= repTotalPages }" @click="pagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战斗报告: 我打别人 + 战报查询 ===== -->
          <template v-else>
            <div class="old-line">
              战报查询:
              <input v-model="reportWord" placeholder="输入关键字" style="width:110px"
                     @keyup.enter="loadReports"/>
              <button @click="loadReports">[查询]</button>
              <a v-if="reportWord" href="javascript:;" @click="reportWord = ''; loadReports()">[清空]</a>
            </div>
            <div class="old-line" v-for="r in repPaged" :key="'rb' + r.id">
              <a href="javascript:;" @click="openReport(r)">
                <span v-if="r.is_read === 0" class="red">[新]</span>
                <span class="orange">[{{ r.type_name }}]</span>{{ r.title }}</a>
              <span class="gray">({{ fmtTime(r.created_at) }})</span>
            </div>
            <div class="old-line" v-if="!reports.length">(暂无战斗报告)</div>
            <div class="ezfy-pager" v-if="reports.length > repSize">
              <a href="javascript:;" :class="{ gray: repPage <= 1 }" @click="pagerGo('rep', -1)">上一页</a>
              <span class="gray">第 {{ repPage }}/{{ repTotalPages }} 页（共 {{ reports.length }} 条）</span>
              <a href="javascript:;" :class="{ gray: repPage >= repTotalPages }" @click="pagerGo('rep', 1)">下一页</a>
            </div>
          </template>

          <!-- ===== 战报详情 ===== -->
          <template v-if="curReport">
            <hr/>
            <div class="panel-title">{{ curReport.title }}</div>
            <pre class="report-pre">{{ curReport.content }}</pre>
            <template v-if="curReport.detail">
              <div class="old-line"><a href="javascript:;" @click="showDetail = !showDetail">[展开/收起逐回合详情]</a></div>
              <pre class="report-pre" v-if="showDetail">{{ curReport.detail }}</pre>
            </template>
            <div class="old-line">
              <a href="javascript:;" @click="curReport = null">[收起]</a>
            </div>
          </template>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 好友(friends) ============ -->
      <template v-else-if="cur === 'friends'">
        <div class="panel">
          <div class="panel-title">游戏内好友（与社区好友分开）</div>
          <div class="old-line gray">
            这里只是「二战风云」里的好友关系，不影响社区的亲友列表。
          </div>

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
        <template v-for="g in taskGroups">
          <div class="panel-title" :key="'tg' + g.id">{{ g.name }}<span v-if="g.reset_type === 1">(每日)</span></div>
          <div class="panel" :key="'gl' + g.id">
            <div class="old-line" v-for="t in g.tasks" :key="t.id">
              <b>{{ t.name }}</b> {{ t.current }}/{{ t.target }}
              <span v-if="t.status === 2" class="gray">[已领取]</span>
              <a v-else-if="t.status === 1" href="javascript:;" @click="doAward(t)">[领奖]</a>
              <br/>
              <span class="gray">奖励:{{ rewardText(t.reward) }}</span>
            </div>
          </div>
        </template>
      </template>

      <!-- ============ 城市列表(cities) ============ -->
      <template v-else-if="cur === 'cities'">
        <div class="panel">
          <div class="panel-title">我的城市列表</div>
          <div class="old-line" v-for="ct in cities" :key="'ct' + ct.id">
            <b>{{ ct.name }}</b><span v-if="ct.id === city.id" class="red">[当前]</span><br/>
            坐标({{ ct.x }},{{ ct.y }}) 城级{{ ct.city_level }}
            <span :class="isSeaAt(ct) ? 'green' : 'gray'">[{{ ct.city_kind || (isSeaAt(ct) ? '沿海城市' : '内陆城市') }}]</span>
            <br/>
            <span class="gray">所属洲: {{ ct.continent || '—' }}</span><br/>
            <a v-if="ct.id !== city.id" href="javascript:;" @click="doSwitch(ct)">[切换]</a>
            <!-- ★ 运输：从当前城市把资源运到这座城（负重决定运量，可不带军官） -->
            <a v-if="ct.id !== city.id" href="javascript:;" @click="doTransportTo(ct)">[运输]</a>
            <!-- ★ 派遣：像出征一样，把自己的部队/军官/随军资源送到自己的另一座城市 -->
            <a v-if="ct.id !== city.id" href="javascript:;" @click="doDispatchTo(ct)">[派遣]</a>
            <!-- ★ 弃城：只能弃「非当前所在」的城市；弃城后该坐标恢复为普通平原 -->
            <a v-if="ct.id !== city.id" class="red" href="javascript:;" @click="doDestroyCity(ct)">[弃城]</a>
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
          <div class="gray" style="font-size:14px">
            <b>平原</b> → 内陆城市; <b>沿海平原</b> → 沿海城市(可建航海协会、训练海军)。<br/>
            其他地形(含海洋)不能建城; 新城自带基础建筑(市政厅/民居/农田1级), 建造后可在上方列表切换操作。
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
          <span class="gray" v-if="resDetail.base_building !== undefined">
            （建筑{{ resDetail.base_building }} × 科技{{ 100 + (resDetail.tech_prod || 0) * 10 }}%）
          </span>
          <br/>
          加成产量(每小时): {{ resDetail.bonus }}
          <span class="gray" v-if="resDetail.rate !== undefined && resDetail.rate !== 100"> [开工率{{ resDetail.rate }}%]</span>
          <span class="gray" v-if="resDetail.mayor_bonus > 0"> [市长加成+{{ resDetail.mayor_bonus }}%]</span>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <div class="old-line">
            {{ cur === 'buildm' ? '军事区' : '资源区' }}.
            <a href="javascript:;" @click="go(cur === 'buildm' ? 'builds' : 'buildm')">{{ cur === 'buildm' ? '资源区' : '军事区' }}</a>
          </div>
          <br/>
          <div class="old-line">建造中队列数：{{ buildQueueCount }}</div>
          <div class="old-line">
            数量/最大：{{ areaCount }}/{{ areaCap }}
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
              <a href="javascript:;" @click="doSpeedBuilding()">加速</a>
            </template>
            <template v-else-if="b.level > 0 && b.level < b.max_level">
              <a href="javascript:;" @click="doUpgrade(b)">升级</a>
              <a href="javascript:;" @click="doMaxLevel(b)">一键{{ b.max_level - 1 }}级</a>
              <a v-if="b.can_delete === 1" href="javascript:;" @click="doDeleteBuilding(b)">拆除</a>
            </template>
            <template v-else>
              <a v-if="b.level === 0" href="javascript:;" @click="doUpgrade(b)">建成中待完成</a>
              <a v-else-if="b.can_delete === 1" href="javascript:;" @click="doDeleteBuilding(b)">拆除</a>
            </template>
          </div>
          <div class="old-line gray" v-if="!zoneBuilt.length">(本区还没有建筑, 点上面的「建造」)</div>
          <br/>
          <div class="old-line">
            <a href="javascript:;" @click="doSpeedTrainAll">[训练一键加速]</a>|
            <a href="javascript:;" @click="doSpeedTrainAllCity">[所有城市训练一键加速]</a>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
            训练中占用人口：{{ popUsed }}（出厂即归还，已训练完成的部队不占人口）
          </div>
          <div class="old-line" v-for="t in trainCfgs" :key="'tt' + t.id">
            <a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a>({{ troopTypeName(t.type) }}) 血{{ t.health }} 防{{ t.defence }} 速{{ t.speed }} 射程{{ t.attack_range }} 负重{{ t.carry }}<br/>
            消耗: {{ resShort.food }}{{ t.cost.food }} {{ resShort.steel }}{{ t.cost.steel }} {{ resShort.oil }}{{ t.cost.oil }} {{ resShort.rare }}{{ t.cost.rare }} 训练{{ t.train_time }}秒/个<br/>
            前提: {{ t.require || '无' }}<template v-if="t.type === 1"> <span class="red">(海军: 仅海城可训练)</span></template>
            <a href="javascript:;" @click="openTrainPre(t, 'troop')">[训练]</a><br/>
          </div>
          <div class="panel-title">训练队列({{ queues.length }})</div>
          <div class="old-line" v-for="q in queues" :key="'q' + q.id">
            {{ q.name }}×{{ q.count }} 剩余{{ remain(q.end_time) }}
            <!-- 接口只返回 status=0（训练中）的队列，所以这里不需要再判断状态 -->
            <a href="javascript:;" @click="doCancelTrain(q)">[取消]</a>
          </div>
          <div class="old-line" v-if="!queues.length">(队列为空)</div>
          <a href="javascript:;" @click="go('defence')">[去建城防]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
              <td><a href="javascript:;" @click="openTroopView(t.id)">{{ t.name }}</a>:</td>
              <td>{{ troopCount(t.id) }}</td>
              <td>
                <a href="javascript:;" @click="openTrainPre(t, 'defence')">[建造]</a>
                <a href="javascript:;" @click="doDismiss(t)">[拆除]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray">城防设施占用「城防空间」(围墙容量)，不占用人口。</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军队总览(troops) ============ -->
      <template v-else-if="cur === 'troops'">
        <div class="panel">
          <div class="panel-title">城内军队</div>
          <table>
            <tr><th>兵种</th><th>类型</th><th>数量</th><th>操作</th></tr>
            <tr v-for="t in troopsData.troops" :key="'tv' + t.troop_id">
              <td><a href="javascript:;" @click="openTroopView(t.troop_id)">{{ t.name }}</a></td>
              <td>{{ troopTypeName(t.type) }}</td><td>{{ t.count }}</td>
              <!-- ★ 解散：数量由玩家自己输入（用户要求） -->
              <td><a class="red" href="javascript:;" @click="doDisband(t)">[解散]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!troopsData.troops.length">(城内无部队)</div>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 司令部(hq) ============ -->
      <template v-else-if="cur === 'hq'">
        <div class="panel">
          <div class="panel-title">司令部: 兵种战斗配置</div>
          <div class="old-line gray">
            每个兵种可分别设置 进攻/防守 的默认攻击对象与前进停止。
            窄屏下改成「一兵种一块」，避免下拉框互相遮盖。
          </div>
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
              <select v-model="targetCfg[t.id].defMove" style="width:80px"
                      :disabled="isDefenceTroop(t)">
                <option :value="1">前进</option><option :value="0">停止</option>
              </select>
            </div>
          </div>
          <div class="old-line"><button @click="doSaveTargets">[保存全部配置]</button></div>
          <br/>
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
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!orders.length">(暂无出征部队)</div>
          <br/>
          <div class="panel-title">伤兵营</div>
          <div class="old-line gray">伤兵在营中<b>不消耗粮食</b>；恢复出厂需要黄金（按兵种造价折算）。</div>
          <table>
            <tr><th>兵种</th><th>数量</th><th>恢复费用</th><th>操作</th></tr>
            <tr v-for="w in woundedList(0)" :key="'w' + w.id">
              <td>{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ fmtN((w.heal_gold || 0) * w.count) }} {{ resNames.gold }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[恢复]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(0).length">(伤兵营无伤兵)</div>
          <div class="old-line" v-if="woundedList(0).length">
            合计 <b>{{ fmtN(woundedHealCost(0)) }}</b> {{ resNames.gold }}
            <button @click="doRecoverAll(0)">[全部恢复]</button>
          </div>
          <br/>
          <div class="panel-title">逃兵营</div>
          <table>
            <tr><th>兵种</th><th>数量</th><th>召回费用</th><th>操作</th></tr>
            <tr v-for="w in woundedList(1)" :key="'dsw' + w.id">
              <td>{{ w.name }}</td><td>{{ w.count }}</td>
              <td>{{ fmtN((w.heal_gold || 0) * w.count) }} {{ resNames.gold }}</td>
              <td><a href="javascript:;" @click="doRecover(w)">[召回]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!woundedList(1).length">(逃兵营无逃兵)</div>
          <div class="old-line" v-if="woundedList(1).length">
            合计 <b>{{ fmtN(woundedHealCost(1)) }}</b> {{ resNames.gold }}
            <button @click="doRecoverAll(1)">[全部召回]</button>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 科技(techs) ============ -->
      <template v-else-if="cur === 'techs'">
        <div class="panel">
          <div class="panel-title">【科技中心】:{{ techsData.academy }}级</div>
          <div class="old-line" v-for="t in techsData.techs" :key="'te' + t.tech_id">
            <b>{{ t.name }}</b> {{ t.level }}/{{ t.max_level }}级
            <span class="gray">[需科研中心{{ t.academy_need }}级]</span><br/>
            {{ t.effect }}<br/>
            <span v-if="t.researching" class="orange">研究中 {{ remain(t.end_time) }}
              <a href="javascript:;" @click="doSpeedTech()">[加速]</a>
              <a href="javascript:;" @click="doCancelTech(t)">[取消]</a></span>
            <span v-else-if="t.level < t.max_level">
              <a href="javascript:;" @click="doResearch(t)">[研究{{ t.level + 1 }}级]</a>
              <span class="gray">耗: {{ resShort.food }}{{ t.next_cost.food }} {{ resShort.steel }}{{ t.next_cost.steel }} {{ resShort.oil }}{{ t.next_cost.oil }} {{ resShort.rare }}{{ t.next_cost.rare }} {{ resShort.gold }}{{ t.next_cost.gold }} 需{{ Math.ceil(t.next_time / 60) }}分钟</span>
            </span>
            <span v-else class="gray">[已满级]</span>
            <br/>
          </div>
          <div class="old-line gray">同一时间只能研究一项科技；[取消] 会全额退还本次研究消耗。</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
                <a href="javascript:;" :class="cellClass(cell)" :title="cellTip(cell)" @click="openCell(cell)">
                  <span class="ezfy-cell-name">{{ cellText(cell) }}</span>
                  <span class="ezfy-cell-xy">({{ cell.x }},{{ cell.y }})</span>
                </a>
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
            <span class="red">特殊</span>=特殊城市 (活动目标无法占领, 战胜只结算奖励)
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('orders')">出征队列</a>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
            <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          </div>
          <div class="old-line green" v-if="troopsData.train_discount > 0 && trainMode !== 'defence'">
            节日活动·造兵打折：资源消耗 -{{ troopsData.train_discount }}%（上方为折后价）
          </div>
          <div class="old-line">
            建造数量：
            <input v-model="trainCount" type="number" min="1" :placeholder="'(1~' + maxTrainable + ')'" style="width:90px"/>
            <span class="gray">(最多 {{ maxTrainable }})</span>
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
          <div class="old-line">
            所属区域：{{ selDetail ? selDetail.continent : (selCell.continent || '未知') }}
          </div>
          <div class="old-line">
            Lv{{ selCell.act_level || selCell.level || selCell.city_level || 0 }}({{ selCell.x }},{{ selCell.y }})
            <a href="javascript:;" @click="addStar">收藏</a>
            <a href="javascript:;" @click="toggleStars">收藏列表</a>
          </div>
          <div class="old-line">
            {{ selCell.name }}<template v-if="selCell.level && !selCell.act_type">({{ selCell.level }})</template>
            <span v-if="selCell.city_level">({{ selCell.city_level }}级)</span>
          </div>
          <template v-if="selDetail">
            <!-- 活动目标(活动野地/活动寇城/特殊城市): 复刻 activityIndex.html 的说明 + 守军/奖励预览 -->
            <template v-if="selDetail.act_type">
              <div class="old-line orange">
                {{ selDetail.act_name }}{{ selDetail.act_level }}级 —— {{ selDetail.act_desc }}
              </div>
              <div class="old-line">
                守军情况：<span v-for="tp in selDetail.troops" :key="'ap' + tp.troop_id">{{ tp.name }}×{{ tp.min }} </span>
              </div>
              <div class="old-line">守军总兵力：{{ selDetail.act_total }}</div>
              <div class="old-line">
                胜利奖励：{{ resShort.food }}/{{ resShort.steel }}/{{ resShort.oil }}/{{ resNames.rare }} 各{{ selDetail.res_min }}，{{ resNames.gold }}{{ selDetail.gold }}，
                军功声望+{{ selDetail.prestige }}，必定掉落宝物
              </div>
              <div class="old-line" v-if="selDetail.jewel">采集可获得：{{ selDetail.jewel }}</div>
              <div class="old-line red">活动目标无法占领，战胜只结算奖励(不占附属野地上限)</div>
            </template>
            <template v-else>
              <div class="old-line" v-if="selDetail.type === 1">
                {{ selDetail.terrain_name }}中可以产出{{ resNames.food }}、{{ resNames.steel }}、{{ resNames.oil }}、{{ resNames.rare }}
              </div>
              <div class="old-line" v-else-if="selDetail.type === 2">
                海洋中可以产出{{ resNames.oil }}、{{ resNames.rare }}、{{ resNames.gold }}
              </div>
              <div class="old-line" v-else>寇城中囤积了大量资源与宝物</div>
              <div class="old-line" v-if="selDetail.jewel">采集可获得：{{ selDetail.jewel }}</div>
              <div class="old-line">【归属: {{ selDetail.owner || '无' }}】</div>
              <div class="old-line">
                守军情况：<span v-for="tp in selDetail.troops" :key="'sp' + tp.troop_id">{{ tp.name }}约{{ tp.min }}-{{ tp.max }} </span>
                <span v-if="!selDetail.troops.length" class="gray">(无守军)</span>
              </div>
              <div class="old-line">掠夺资源约：{{ selDetail.res_min }}-{{ selDetail.res_max }}</div>
            </template>
          </template>
          <div class="old-line" v-else>
            {{ selCell.name }}
            <span v-if="selCell.owner">城主:{{ selCell.owner }}</span>
          </div>
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
               未宣战/待生效时置灰并提示，宣战入口只在没宣战(status=0)时出现 -->
          <div class="old-line" v-else-if="selCell.area_type === 3">
            <a href="javascript:;" @click="pickOrder(1)">[侦查]</a>&nbsp;
            <a v-if="warStatus === 2" href="javascript:;" @click="pickOrder(2)">[掠夺]</a>
            <a v-else href="javascript:;" class="gray" @click="warBlock('掠夺')">[掠夺]</a>&nbsp;
            <a v-if="warStatus === 2" href="javascript:;" @click="pickOrder(3)">[征服]</a>
            <a v-else href="javascript:;" class="gray" @click="warBlock('征服')">[征服]</a>&nbsp;
            <!-- ★ 运输/增援 只对「同盟(同一军团)成员的城市」显示；宣战中一律不显示 -->
            <template v-if="selCell.ally && warStatus !== 2">
              <a href="javascript:;" @click="pickOrder(5)">[运输]</a>&nbsp;
              <a href="javascript:;" @click="pickOrder(6)">[增援]</a>&nbsp;
            </template>
            <!-- ★ 用户规则「同盟玩家不能宣战」→ 同盟成员不出现 [宣战] 入口，只给提示 -->
            <template v-if="selCell.ally">
              <span class="green">同盟成员之间不能宣战</span>
            </template>
            <a v-else-if="warStatus === 0" href="javascript:;" @click="declareWar">[宣战]</a>
            <!-- 同盟时不再叠「未宣战」这类状态文案，避免读成「不能宣战未宣战」 -->
            <span v-if="warText && !selCell.ally" class="orange">{{ warText }}</span>
          </div>
          <!-- ③ 野地/寇城/海洋 -->
          <div class="old-line" v-else-if="selCell.name !== '寇城(废墟)'">
            <a href="javascript:;" @click="pickOrder(1)">[侦查]</a>&nbsp;
            <a href="javascript:;" @click="pickOrder(2)">[掠夺]</a>&nbsp;
            <a href="javascript:;" @click="pickOrder(3)">[征服]</a>&nbsp;
            <a v-if="selCell.occupied" href="javascript:;" @click="pickOrder(4)">[采集]</a>
            <span v-else-if="!selDetail || !selDetail.act_type" class="gray">(占领该野地后可采集)</span>
          </div>
          <a href="javascript:;" @click="go('map')">[返回地图]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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

          <!-- ① 兵力 -->
          <div class="of-sec">① 选择兵力 <span class="of-hint">（左列填出征数量，右侧灰字是城内现有）</span></div>
          <div class="of-grid of-grid-troop">
            <div class="of-cell" v-for="t in trainCfgs" :key="'at' + t.id"
                 :class="{ 'of-off': troopCount(t.id) <= 0 }"
                 :title="t.name + '（现有 ' + fmtN(troopCount(t.id)) + '）'">
              <span class="of-name">{{ t.name }}</span>
              <input type="number" min="0" :max="troopCount(t.id)"
                     v-model="orderTroops[t.id]"
                     placeholder="0"
                     :disabled="troopCount(t.id) <= 0" class="of-num"/>
              <span class="of-avail">{{ fmtN(troopCount(t.id)) }}</span>
            </div>
          </div>
          <div class="old-line red" v-if="!attackTroops.length">城内无可出征部队</div>
          <div class="old-line" v-if="orderCalc">
            <span :class="orderCalc.troop_over_cap ? 'red' : 'green'">
              本次出兵 <b>{{ fmtN(orderCalc.troop_total) }}</b> / 上限 <b>{{ fmtN(orderCalc.troop_cap) }}</b>
              <template v-if="orderCalc.troop_over_cap">—— 超出上限，请减少兵力或加用集结令</template>
            </span>
          </div>

          <!-- ② 军官 -->
          <div class="of-sec">② 指挥军官</div>
          <div class="old-line">
            <select v-model="orderOfficer">
              <option value="0">未指定</option>
              <option v-for="o in onDutyOfficers" :key="'od' + o.id" :value="o.name">
                {{ o.name }}({{ o.level }}级) 军事{{ o.military_total || o.military }}
                <span class="green" v-if="o.equip_military">(装备+{{ o.equip_military }})</span>
                忠诚{{ o.loyalty }}
              </option>
            </select>
            <span v-if="orderType === 7" class="red">(派遣必须选择)</span>
            <span v-else-if="orderType === 6" class="gray">(增援后军官调任目标城市)</span>
            <span v-else-if="orderType === 8" class="gray">(派遣后军官随军调往目标城市)</span>
            <span v-if="curOfficerBonus" class="green"> 军官战斗加成: 攻击+{{ curOfficerBonus }}%</span>
            <!-- ★ 用户反馈「新城市有军官，出征页却没有」→ 空列表时把**是哪座城**、**为什么空**写清楚，
                 避免玩家误以为出征页在用切换前那座城的数据（军官是跟城走的，不跨城指挥）。
                 若本城其实有军官（只是都在出征中/是俘虏），也直接说明，别让人白找。 -->
            <span v-if="!onDutyOfficers.length" class="gray">
              (「{{ city.name }}」暂无可带队军官<template v-if="cityOfficers.length">：本城 {{ cityOfficers.length }} 名军官都在出征中或为俘虏</template>；
              军官跟着城市走，别的城的军官不能在这里出征，可前往军校招募)
            </span>
          </div>

          <!-- ③ 集结令 -->
          <div class="of-sec">③ 出征集结令</div>
          <div class="old-line">
            使用
            <input type="number" min="0" :max="gatherMax" v-model.number="orderGather"
                   :disabled="gatherCount <= 0" @change="onGatherChange" style="width:80px"/>
            个
            <span class="gray">（背包里有 {{ gatherCount }} 个）</span>
          </div>
          <!-- ★ 用户要求：说明文字放到输入框下面，别挤在同一行 -->
          <div class="old-line gray">
            每个集结令 +{{ fmtN(orderCapPer) }} 出征上限，单次最多 {{ orderCapMax }} 个（管理端可调）。
            司令部上限（含指挥艺术科技）与集结令加成<b>叠加</b>。
          </div>

          <!-- ④ 随军资源 -->
          <div class="of-sec">④ 随军资源 <span class="of-hint">（右侧灰字是城内现有）</span></div>
          <div class="of-grid of-grid-res">
            <div class="of-cell"><span class="of-name">{{ resNames.gold }}</span>
              <input v-model="trGold" type="number" placeholder="0" class="of-num"/>
              <span class="of-avail">{{ fmtN(city.gold) }}</span></div>
            <div class="of-cell"><span class="of-name">{{ resNames.food }}</span>
              <input v-model="trFood" type="number" placeholder="0" class="of-num"/>
              <span class="of-avail">{{ fmtN(city.food) }}</span></div>
            <div class="of-cell"><span class="of-name">{{ resNames.steel }}</span>
              <input v-model="trSteel" type="number" placeholder="0" class="of-num"/>
              <span class="of-avail">{{ fmtN(city.steel) }}</span></div>
            <div class="of-cell"><span class="of-name">{{ resNames.oil }}</span>
              <input v-model="trOil" type="number" placeholder="0" class="of-num"/>
              <span class="of-avail">{{ fmtN(city.oil) }}</span></div>
            <div class="of-cell"><span class="of-name">{{ resNames.rare }}</span>
              <input v-model="trRare" type="number" placeholder="0" class="of-num"/>
              <span class="of-avail">{{ fmtN(city.rare) }}</span></div>
          </div>
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
          <div class="of-sec">⑤ 宿营（抵达后停留，可选）</div>
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

      <!-- ============ 命令详情 / 军队动态详情(orderview) ============ -->
      <template v-else-if="cur === 'orderview'">
        <div class="panel" v-if="curOrder">
          <div class="panel-title">军队动态详情</div>
          出发地:{{ curOrder.from_name }}<br/>
          目的地:{{ curOrder.target_name }}({{ curOrder.target_x }},{{ curOrder.target_y }})<br/>
          命令:{{ curOrder.type_name }}<br/>
          军官:{{ curOrder.officer || '无(未带军官)' }}<br/>
          统帅:{{ nick }}<br/>
          状态:{{ curOrder.status_name || orderStatusText(curOrder) }}<br/>
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
            <a v-if="curOrder.order_type === 7 && curOrder.status === 1"
               href="javascript:;" @click="go('wilds')">[采集]</a>
            <a v-if="curOrder.report_id" href="javascript:;" @click="jumpReport(curOrder.report_id)">[查看战报]</a>
          </div>
          <a href="javascript:;" @click="go('orders')">[返回出征队列]</a>
        </div>
      </template>

      <!-- ============ 出征队列(orders) ============ -->
      <template v-else-if="cur === 'orders'">
        <div class="panel">
          <div class="panel-title">出征队列({{ orders.length }})</div>
          <table>
            <tr><th>命令</th><th>目标</th><th>状态</th><th>军官</th><th>抵达时间</th><th>操作</th></tr>
            <tr v-for="o in orders" :key="'odl' + o.id">
              <td>{{ o.type_name }}</td>
              <td>({{ o.target_x }},{{ o.target_y }})</td>
              <td>{{ orderStatusText(o) }}</td>
              <td>{{ o.officer || '无' }}</td>
              <td>{{ o.arrive_text }}</td>
              <td>
                <a href="javascript:;" @click="openOrder(o)">[查看]</a>
                <!-- ★ 出征队列取消：不限命令类型，行进中(0)/驻守中(1)都能取消 -->
                <a v-if="o.status === 0 || o.status === 1" class="red"
                   href="javascript:;" @click="doRecall(o)">[取消]</a>
              </td>
            </tr>
          </table>
          <div class="old-line" v-if="!orders.length">(暂无出征部队)</div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          人口: {{ city.pop }}/{{ city.pop_max }} (空闲{{ freePop }})<br/>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
              <td>{{ w.terrain_name }}<span class="gray" v-if="w.wild_type === 2">(海野)</span></td>
              <td>{{ w.continent || '—' }}</td>
              <td>{{ w.level }}</td>
              <td>{{ w.status === 0 ? '空闲' : '采集中' }}</td>
              <td>
                <a v-if="w.status === 0" href="javascript:;" @click="openWildGather(w)">[采集]</a>
                <span v-else class="gray">采集中</span>
                <a class="red" href="javascript:;" @click="doAbandon(w)">[放弃]</a>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 召集(convene) ============ -->
      <template v-else-if="cur === 'convene'">
        <div class="panel">
          <div class="panel-title">召集人口</div>
          当前人口: {{ city.pop }} / 民居容纳: {{ city.pop_max }}<br/>
          花费 10万{{ resNames.gold }} 召集 10万人口(不受民居容纳上限限制, 可突破上限)<br/>
          <div class="old-line">{{ resNames.gold }}: {{ city.gold }}</div>
          <button @click="doConvene">[召集]</button>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          安抚花费 民怨×100 {{ resNames.gold }}, 可清零民怨并回升民心。<br/>
          <div class="old-line">{{ resNames.gold }}: {{ city.gold }} | 预计花费: {{ city.grievance * 100 }}</div>
          <button @click="doPlacate">[安抚]</button>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 税率(taxset) ============ -->
      <template v-else-if="cur === 'taxset'">
        <div class="panel">
          <div class="panel-title">税率设置</div>
          当前税率: {{ city.tax_rate }}%<br/>
          <span class="gray">税率越高{{ resNames.gold }}收入越多, 但民心下降越快: ≤10%民心+2/时, ≤20%+1, ≤40%不变, ≤60%-1, 更高-2; 民怨≥50时产量减半。</span><br/>
          <div class="old-line">
            新税率: <input v-model="taxInput" type="number" min="0" max="100" style="width:70px"/>%
            <button @click="doTax">[设置]</button>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
            <tr v-for="b in buildings" :key="'hb' + b.id">
              <td>{{ b.name }}</td>
              <td>{{ b.level }}/{{ b.max_level }}</td>
              <td>{{ b.status === 0 ? '空闲' : '施工中 ' + remain(b.end_time) }}</td>
              <td>
                <a v-if="b.status === 0 && b.level > 0 && b.level < b.max_level" href="javascript:;" @click="doUpgrade(b)">[升级]</a>
                <span v-if="b.status !== 0"><a href="javascript:;" @click="doSpeedBuilding()">[加速]</a></span>
              </td>
            </tr>
          </table>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 军团(corps) ============ -->
      <template v-else-if="cur === 'corps'">
        <template v-if="myCorps">
          <div class="panel">
            <div class="panel-title">我的军团:{{ myCorps.name }}({{ myCorps.member_count }}人)</div>
            公告: {{ myCorps.notice || '无' }}<br/>
            <div class="old-line">
              <template v-if="isLeader">
                <a href="javascript:;" @click="openNoticeEdit()">[修改公告]</a>
                <a class="red" href="javascript:;" @click="doLeaveCorps()">[解散军团]</a>
              </template>
              <a v-else href="javascript:;" @click="doLeaveCorps()">[退出军团]</a>
            </div>
            <div class="panel-title">军团成员</div>
            <table>
              <tr>
                <th>成员</th><th>玩家号码</th><th>职位</th><th>声望</th><th>军衔</th>
                <!-- ★ 第九轮：军团长可任命副团长/参谋长 -->
                <th v-if="isLeader" width="150">任命</th>
              </tr>
              <tr v-for="m in corpsMembers" :key="'cm' + m.user_id">
                <td><a href="javascript:;" @click="openPlayer(m.user_id)">{{ m.name }}</a></td>
                <td><span class="td-mono">{{ m.game_uid || m.user_id }}</span></td>
                <td>{{ m.title || '成员' }}</td>
                <td>{{ m.prestige }}</td>
                <td>{{ m.rank_name }}</td>
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
              <select v-model="kickUserId" style="width:30%">
                <option v-for="m in corpsMembers" v-if="!m.is_leader" :key="'kc' + m.user_id" :value="m.user_id">{{ m.name }}</option>
              </select>
              <button @click="doKick">[踢出]</button>
            </div>
            <!-- 军团邮件群发(复刻 CorpsController.mail): ★ 军团长与副团长都能发 -->
            <div class="panel-title" v-if="canMailCorps">军团邮件(群发全体成员)</div>
            <div class="old-line" v-if="canMailCorps">
              <input v-model="corpsMailContent" placeholder="邮件内容(500字以内)" style="width:60%"/>
              <button @click="doCorpsMail">[群发]</button>
            </div>
          </div>
        </template>
        <div class="panel">
          <div class="panel-title">军团列表</div>
          <table>
            <tr><th>军团</th><th>人数</th><th>战力</th><th>操作</th></tr>
            <tr v-for="cp in corpsList" :key="'cp' + cp.id">
              <td>{{ cp.name }}</td>
              <td>{{ cp.member_count }}</td>
              <td>{{ cp.battle_score }}</td>
              <td><a v-if="!myCorps" href="javascript:;" @click="doJoinCorps(cp)">[加入]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!corpsList.length">(暂无军团)</div>
        </div>
        <div class="panel" v-if="myCorps">
          <div class="panel-title">军团聊天</div>
          <div class="old-line" v-for="m in corpsChats" :key="'cc' + m.id">
            [<a href="javascript:;" @click="openPlayer(m.user_id)">{{ m.user_name }}</a>]:{{ m.content }}
          </div>
          <div class="old-line" v-if="!corpsChats.length">(暂无消息)</div>
          <div class="old-line">
            <input v-model="corpsMsg" style="width:15%"/>
            <button @click="doCorpsChat">发送</button>
            <button @click="loadCorps">刷新</button>
          </div>
        </div>
        <div class="panel" v-if="!myCorps">
          <div class="panel-title">创建军团</div>
          <div class="old-line">
            军团名: <input v-model="corpsName" style="width:10%"/>
            <button @click="doCreateCorps">[创建]</button>
          </div>
        </div>
        <a href="javascript:;" @click="go('home')">[返回首页]</a>
      </template>

      <!-- ============ 排行(rank) ============ -->
      <template v-else-if="cur === 'rank'">
        <div class="panel">
          <!-- 军衔晋升表放最上面, 三个榜单在下面(用户要求) -->
          <div class="panel-title">军衔晋升表</div>
          <div class="old-line">
            军衔等级 / 职位 / 需要声望 / <b>可建城数</b>
            <span v-if="rankData.mine" class="gray">
              （我当前「{{ rankData.mine.rank_name }}」：可建 {{ rankData.mine.city_max }} 座，已有 {{ rankData.mine.city_count }} 座）
            </span>
          </div>
          <table class="ezfy-rank-table">
            <tr><th>等级</th><th>军衔</th><th>职位</th><th>需要声望</th><th>可建城数</th></tr>
            <tr v-for="(r, i) in rankData.ranks" :key="'rk' + i">
              <td>{{ i + 1 }}</td>
              <td>
                {{ r.name }}
                <span v-if="r.name === rankName" class="red">[当前]</span>
              </td>
              <td>{{ r.post }}</td>
              <td>{{ r.need }}</td>
              <td>{{ r.city_max }}</td>
            </tr>
          </table>
          <div class="panel-title">军衔声望榜</div>
          <table class="ezfy-rank-table">
            <tr><th>名次</th><th>统帅</th><th>声望</th><th>军衔</th></tr>
            <tr v-for="r in rankData.prestige" :key="'rp' + r.rank">
              <td>{{ r.rank }}</td>
              <td><a href="javascript:;" @click="openPlayer(r.user_id)">{{ r.name }}</a></td>
              <td>{{ r.prestige }}</td><td>{{ r.rank_name }}</td>
            </tr>
          </table>
          <div class="panel-title">兵力榜</div>
          <table class="ezfy-rank-table">
            <tr><th>名次</th><th>统帅</th><th>城市</th><th>兵力</th></tr>
            <tr v-for="r in rankData.troops" :key="'rt' + r.rank">
              <td>{{ r.rank }}</td>
              <td><a href="javascript:;" @click="openPlayer(r.user_id)">{{ r.role_name }}</a></td>
              <td>{{ r.city_name }}</td><td>{{ r.count }}</td>
            </tr>
          </table>
          <div class="panel-title">军团榜</div>
          <table class="ezfy-rank-table">
            <tr><th>名次</th><th>军团</th><th>人数</th><th>战力</th></tr>
            <tr v-for="r in rankData.corps" :key="'rc' + r.rank">
              <td>{{ r.rank }}</td><td>{{ r.name }}</td><td>{{ r.member_count }}</td><td>{{ r.battle_score }}</td>
            </tr>
          </table>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 背包(bag) ============ -->
      <template v-else-if="cur === 'bag'">
        <div class="panel">
          <div class="panel-title">背包 <button @click="loadBag">刷新</button></div>
          <div class="old-line" v-for="it in bagItems" :key="'bi' + it.cfg_id">
            <b>{{ it.name }}</b>×{{ it.count }}
            <a href="javascript:;" @click="openUse(it)">[使用]</a><br/>
            <span class="gray">{{ it.description }}</span>

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
          <div class="old-line" v-if="!bagItems.length">(背包空空如也)</div>
          <a href="javascript:;" @click="go('mall')">[前往商城]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 商城(mall) ============ -->
      <template v-else-if="cur === 'mall'">
        <div class="panel">
          <div class="panel-title">商城({{ resNames.gold }}{{ city.gold }} · 钻石{{ mallDiamond }})</div>
          <!-- ★ 分类页签（分类由管理端维护，未填时按道具类型自动归类） -->
          <div class="old-line">
            <a href="javascript:;" :class="{ on: mallCat === '' }" @click="setMallCat('')">[全部]</a>
            <template v-for="c in mallCatsList">
              <a :key="'mc' + c" href="javascript:;" :class="{ on: mallCat === c }" @click="setMallCat(c)">[{{ c }}]</a>
            </template>
          </div>
          <div class="old-line gray" v-if="mallDiamond <= 0">
            钻石余额为 0；标记为「钻石道具」的道具若标价 0 钻石可直接购买，其余需由管理员充值钻石后购买。
          </div>
          <div class="old-line" v-for="it in mallPaged" :key="'mi' + it.id">
            <b>{{ it.name }}</b>
            <!-- ★ 双渠道道具（黄金价和钻石价都 > 0）：两种价格都列出来，玩家任选 -->
            <template v-if="it.dual_pay">
              <span class="orange">{{ it.price_diamond }}钻石</span>
              <span class="gray">/</span>
              {{ it.price_gold }}{{ resNames.gold }}
            </template>
            <!-- ★ 标价 0 = 限时免费发放（集结令就是这种），写「限时免费」比写「0钻石」更不容易被误解 -->
            <span v-else-if="it.is_diamond" class="orange">{{ it.price_diamond > 0 ? it.price_diamond + '钻石' : '限时免费' }}</span>
            <span v-else>{{ it.price_gold > 0 ? it.price_gold + resNames.gold : '限时免费' }}</span>
            <!-- ★ 库存（管理端「数据管理 → 道具配置」维护，默认 100；-1 = 无限） -->
            <span v-if="it.unlimited" class="green">库存无限</span>
            <span v-else :class="it.stock > 0 ? 'gray' : 'red'">库存{{ it.stock > 0 ? it.stock : '0(已售罄)' }}</span>
            <a v-if="it.unlimited || it.stock > 0" href="javascript:;" @click="openBuy(it)">[购买]</a>
            <span v-else class="gray">[已售罄]</span><br/>
            <span class="gray">{{ it.description }}</span>
            <div v-if="buyItem && buyItem.id === it.id" class="use-box">
              数量:
              <input v-model="buyCount" type="number" min="1"
                     :max="buyMaxOf(it)" style="width:70px"/>
              <span class="gray">{{ it.unlimited
                ? ('单次最多 ' + mallBuyMax + ' 个')
                : ('最多 ' + buyMaxOf(it)) }}</span>
              <!-- ★ 双渠道：让玩家选付黄金还是付钻石 -->
              <template v-if="it.dual_pay">
                支付方式:
                <select v-model="buyPayWith">
                  <option value="gold">黄金 {{ it.price_gold * (parseInt(buyCount) || 0) }}</option>
                  <option value="diamond">钻石 {{ it.price_diamond * (parseInt(buyCount) || 0) }}</option>
                </select>
              </template>
              <!-- ★ 标价 0 的道具不显示「合计 0 钻石」，直接写「限时免费」 -->
              <span class="gray" v-else-if="it.is_diamond">
                {{ it.price_diamond > 0 ? ('合计 ' + it.price_diamond * (parseInt(buyCount) || 0) + ' 钻石') : '限时免费' }}
              </span>
              <span class="gray" v-else>
                {{ it.price_gold > 0 ? ('合计 ' + it.price_gold * (parseInt(buyCount) || 0) + ' ' + resNames.gold) : '限时免费' }}
              </span>
              <button @click="doBuy(it)">[确认购买]</button>
              <a href="javascript:;" @click="buyItem = null">[取消]</a>
            </div>
          </div>
          <div class="old-line" v-if="!mallPaged.length">(该分类下暂无道具)</div>
          <!-- ★ 分页（每页 10 件） -->
          <div class="ezfy-pager" v-if="mallFiltered.length > mallPageSize">
            <a href="javascript:;" :class="{ disabled: mallPage <= 1 }" @click="mallGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(mallPage, mallTotalPages) }}/{{ mallTotalPages }} 页 · 共 {{ mallFiltered.length }} 件</span>
            <a href="javascript:;" :class="{ disabled: mallPage >= mallTotalPages }" @click="mallGo(1)">[下一页]</a>
          </div>
          <a href="javascript:;" @click="go('bag')">[背包]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 交易行(exchange) ============ -->
      <template v-else-if="cur === 'exchange'">
        <div class="panel">
          <div class="panel-title">资源交易行({{ resNames.gold }}{{ exchangeGold }})</div>
          <div class="old-line gray">购买他人挂单的资源; 也可挂单出售资源换取{{ resNames.gold }}。</div>
          <div class="old-line gray">
            「系统」挂单由管理员上架，可能用<b>{{ resNames.gold }}</b>或<b>钻石</b>定价（钻石需管理员充值）；玩家自己挂单一律按{{ resNames.gold }}买卖。
          </div>
          <table>
            <tr><th>卖家</th><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in exchangeOrders" :key="'eo' + e.id">
              <td>{{ e.seller_name }}</td>
              <td>{{ e.type_name }}</td>
              <td>{{ e.count }}</td>
              <td>{{ e.total_price }}{{ e.currency_name || resNames.gold }}</td>
              <td><a href="javascript:;" @click="doExchangeBuy(e)">[购买]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!exchangeOrders.length">(暂无在售订单)</div>
          <br/>
          <div class="panel-title">我的挂单</div>
          <div class="old-line" v-for="e in exchangeMine" :key="'em' + e.id">
            {{ e.type_name }}×{{ e.count }} 售{{ e.total_price }}{{ e.currency_name || resNames.gold }}
            <a href="javascript:;" @click="doExchangeCancel(e)">[下架]</a>
          </div>
          <div class="old-line" v-if="!exchangeMine.length">(无在售挂单)</div>
          <br/>
          <div class="panel-title">挂单出售</div>
          <div class="old-line">
            资源:
            <select v-model="sellType" style="width:70px">
              <option value="1">{{ resNames.food }}</option><option value="2">{{ resNames.steel }}</option>
              <option value="3">{{ resNames.oil }}</option><option value="4">{{ resNames.rare }}</option>
            </select><br/>
            数量: <input v-model="sellCount" type="number" style="width:90px"/><br/>
            总价({{ resNames.gold }}): <input v-model="sellPrice" type="number" style="width:90px"/><br/>
            <button @click="doExchangeSell">[挂单出售]</button>
            <span class="gray">（玩家挂单只能用{{ resNames.gold }}计价）</span>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 福利(welfare) ============ -->
      <template v-else-if="cur === 'welfare'">
        <div class="panel">
          <div class="panel-title">每日签到</div>
          <div class="old-line">
            <span v-if="welfare.signed_today">今日已签到(连续{{ welfare.sign_count }}天)</span>
            <a v-else href="javascript:;" @click="doSign">[签到领奖]</a>
            | 声望:{{ welfare.prestige }}({{ welfare.rank_name }})
          </div>
          <div class="old-line" v-for="r in welfare.rewards" :key="'sr' + r.day">
            第{{ r.day }}天:{{ r.reward }}
          </div>
          <div class="panel-title">礼包</div>
          <!-- ★ 用户要求「把 [市政厅20级礼包][市政厅30级礼包][市政厅40级礼包] 删掉」：
               只保留 新手 / 每周 / 市政厅10级 三个入口（后端 Gift 同步去掉 20/30/40 分支）。 -->
          <div class="old-line">
            <a href="javascript:;" @click="doGift('newbie')">{{ welfare.gifts.newbie ? '[新手礼包已领]' : '[新手礼包]' }}</a>
            <a href="javascript:;" @click="doGift('weekly')">{{ welfare.gifts.weekly ? '[每周福利已领]' : '[每周福利]' }}</a><br/>
            <a href="javascript:;" @click="doGift('level10')">{{ welfare.gifts.level10 ? '[市政厅10级礼包已领]' : '[市政厅10级礼包]' }}</a>
          </div>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
        </div>
      </template>

      <!-- ============ 公告(notices) ============ -->
      <template v-else-if="cur === 'notices'">
        <div class="panel">
          <div class="panel-title">公告</div>
          <div class="old-line" v-for="n in notices" :key="'nn' + n.id">
            <span v-if="n.is_top" class="red">[置顶]</span>
            <a href="javascript:;" @click="openNotice(n)">{{ n.title }}</a>
          </div>
          <template v-if="curNotice">
            <div class="panel-title">{{ curNotice.title }}</div>
            <div class="old-line">{{ curNotice.content }}</div>
          </template>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
            创建联盟需消耗{{ liaison.create_cost }}{{ resNames.gold }}（原版为 50 钻石）<br/>
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
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <div class="gray" style="font-size:13px">{{ renameHint }}</div>
          阵营：{{ selfInfo.camp_name || (profile.camp === 2 ? '轴心国' : '同盟国') }}
          <a href="javascript:;" @click="doChangeCamp(1)">[转同盟国]</a>
          <a href="javascript:;" @click="doChangeCamp(2)">[转轴心国]</a>
          <div class="gray" style="font-size:13px">{{ campHint }}</div>
          声望：{{ profile.prestige }}<br/>
          军衔：{{ rankName }}({{ rankPost }})<br/>
          城市数：{{ cities.length }}<br/>
          人口数：{{ city.pop }}<br/>
          军官数：{{ officerCount }}<br/>
          <br/>
          总兵力：{{ totalTroops }}<br/>
          城外行进：{{ marching }}支 | 驻守采集：{{ occupying }}支<br/>
          占领野地：{{ wildlands.length }}块<br/>
          <a href="javascript:;" @click="go('friends')">[申请好友]</a>
          <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
            军衔：{{ playerInfo.rank_name }}({{ playerInfo.rank_post }})<br/>
            军团：{{ playerInfo.corps_name || '无' }}<br/>
            城市数：{{ playerInfo.city_count }}<br/>
            军官数：{{ playerInfo.officer_count }}<br/>
            总兵力：{{ playerInfo.troop_total }}<br/>
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
              {{ o.name }}({{ o.level }}级)<span class="green" v-if="o.level >= officerMaxLevel">[满级]</span>
              <a href="javascript:;" @click="openOfficer(o.id)">查看</a><br/>
              状态:{{ o.status === 1 ? '出征' : '空闲' }} &nbsp; 评价:{{ o.star }}星<br/>
              后勤/军事/学识/忠诚：<br/>
              {{ o.logistics_total }}/{{ o.military_total }}/{{ o.learning_total }}/{{ o.loyalty }}
              <span class="green" v-if="equipTip(o)">{{ equipTip(o) }}</span><br/>
              攻/防：{{ o.attack }}/{{ o.defence }}<br/>
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
              今日刷新:{{ recruitData.refresh_left }}/{{ recruitData.refresh_limit }}次
            </span>
            <a href="javascript:;" @click="doRefreshRecruit">[刷新]</a>
            <!-- ★ 次数用完后，直接在军校使用招生简章（不用先去背包用） -->
            <a href="javascript:;" @click="doUseRecruitTicket">[使用招生简章刷新]</a>
            <span class="gray">(持有 {{ bagCount(13) }} 张)</span>
          </div>
          <div class="old-line">
            军校等级决定每日候选数量, 参谋部{{ recruitData.staff_level }}级(已用{{ recruitData.used }}/{{ recruitData.capacity }}),
            雇佣费用 = 军官等级 × 1000 {{ resNames.gold }}
          </div>
          <div class="old-line gray">军校招募的是普通军官; 名将只能由管理端发放(见[名将图鉴])</div>
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
          <div class="old-line gray" v-if="recruitData.academy_level && !recruitData.candidates.length">(今日候选已全部招募或刷新)</div>
          <div class="old-line">前去<a href="javascript:;" @click="switchAcade('officer')">[军官]</a></div>
        </div>

        <!-- 任命市长: 复刻 acade/setMayor.html -->
        <div class="panel" v-else-if="acadeTab === 'mayor'">
          <div class="old-line gray">参谋部: 市长(产量+10%+后勤属性)、城守(守城防御+10%+学识)。军官身上的装备加成同样计入。</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>等级</th><th>忠诚</th><th>当前职位</th><th>操作</th></tr>
            <tr v-for="o in myOfficers" :key="'my' + o.id">
              <td>{{ o.name }}</td>
              <td>{{ o.level }}<span class="green" v-if="o.level >= officerMaxLevel">满级</span></td>
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
          <div class="old-line">我的装备({{ equipData.bag.length }})</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>要求等级</th><th>状态</th></tr>
            <tr v-for="e in equipData.bag" :key="'eq' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
              <td>
                <span v-if="e.worn" class="gray">{{ e.worn_by }}已穿戴</span>
                <a v-else href="javascript:;" @click="switchAcade('officer')">[去穿戴]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!equipData.bag.length">(背包暂无装备, 战胜野地/寇城有概率掉落)</div>
          <hr/>
          <div class="old-line">装备图鉴({{ equipData.all.length }})</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>需求等级</th></tr>
            <tr v-for="e in equipData.all" :key="'ea' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
            </tr>
          </table>
        </div>

        <!-- 技能: 复刻 acade/skill.html 的编号列表(带完整说明) -->
        <div class="panel" v-else-if="acadeTab === 'skill'">
          <div class="old-line">军官技能(每名武将最多3个, 学习1万金/个):</div>
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

        <!-- 计谋(复刻原版 acade/scheme.html: 12 条计谋说明, 各需信号弹) -->
        <div class="panel" v-else-if="acadeTab === 'scheme'">
          <div class="old-line">说明：计谋需要进入相应界面才可以使用</div>
          <div class="old-line" v-for="(s, i) in schemes" :key="'sc' + i">
            {{ i + 1 }}.{{ s.name }}：<br/>
            {{ s.des }}<br/>
            需要信号弹：{{ s.bullet }}
            <br/>--------------------
          </div>
          <div class="old-line">
            <a href="javascript:;" @click="go('bag')">[背包(信号弹)]</a>
            <a href="javascript:;" @click="go('home')">[返回首页]</a>
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
          <div class="old-line gray">军校招募的是普通军官。</div>
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>等级</th><th>星级</th><th>军/后/学</th><th>获取渠道</th><th>状态</th></tr>
            <tr v-for="g in generalData.generals" :key="'gg' + g.id">
              <td>{{ g.name }}</td>
              <td>{{ g.level }}</td>
              <td>{{ g.star }}</td>
              <td>{{ g.military }}/{{ g.logistics }}/{{ g.learning }}</td>
              <td>{{ g.source }}</td>
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
        <div class="panel" v-if="officerDetail.officer">
          <div class="panel-title">{{ officerDetail.officer.name }}</div>
          星级:{{ officerDetail.officer.star }}
          等级:{{ officerDetail.officer.level }}<span class="green" v-if="officerDetail.officer.level >= officerMaxLevel">（已满级，最高 {{ officerMaxLevel }} 级）</span>
          经验:{{ officerDetail.officer.level >= officerMaxLevel ? '—' : (officerDetail.officer.exp + '/' + officerDetail.officer.exp_need) }}<br/>
          军事:{{ officerDetail.officer.military_total }}
          <span class="green" v-if="officerDetail.officer.equip_military">({{ officerDetail.officer.military }}+装备{{ officerDetail.officer.equip_military }})</span>
          后勤:{{ officerDetail.officer.logistics_total }}
          <span class="green" v-if="officerDetail.officer.equip_logistics">({{ officerDetail.officer.logistics }}+装备{{ officerDetail.officer.equip_logistics }})</span>
          学识:{{ officerDetail.officer.learning_total }}
          <span class="green" v-if="officerDetail.officer.equip_learning">({{ officerDetail.officer.learning }}+装备{{ officerDetail.officer.equip_learning }})</span><br/>
          攻击加成:{{ officerDetail.officer.attack }} &nbsp; 防御加成:{{ officerDetail.officer.defence }}<br/>
          忠诚:{{ officerDetail.officer.loyalty }}
          职位:{{ officerDetail.officer.position_name }}
          状态:{{ officerDetail.officer.status_name }}<br/>
          <div class="old-line">
            <button @click="doGrant">[赏赐+10忠诚(1万金)]</button>
            <button v-if="officerDetail.officer.status !== 1 && officerDetail.officer.position === 0"
                    @click="doExile">[流放]</button>
            <span v-if="officerDetail.officer.status === 1" class="gray">(出征中, 归来后才能流放)</span>
            <span v-else-if="officerDetail.officer.position !== 0" class="gray">(市长/城守, 卸任后才能流放)</span>
          </div>
          <hr/>
          已学技能({{ officerDetail.skills.length }}/3):
          <table>
            <tr><th>技能</th><th>效果</th><th>操作</th></tr>
            <tr v-for="s in officerDetail.skills" :key="'ds' + s.name">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doForget(s.name)">[遗忘]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.skills.length">(未学任何技能)</div>
          <br/>
          可学技能(1万金/个):
          <table>
            <tr><th>名称</th><th>效果</th><th>操作</th></tr>
            <tr v-for="s in officerDetail.all_skills" :key="'ls' + s.id">
              <td>{{ s.name }}</td>
              <td>{{ s.effect }}</td>
              <td><a href="javascript:;" @click="doLearn(s)">[学习]</a></td>
            </tr>
          </table>
          <hr/>
          已穿戴装备:
          <table>
            <tr><th>名称</th><th>类型</th><th>军事</th><th>后勤</th><th>学习</th><th>操作</th></tr>
            <tr v-for="e in officerDetail.equipped" :key="'de' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.military }}</td>
              <td>{{ e.logistics }}</td>
              <td>{{ e.learning }}</td>
              <td><a href="javascript:;" @click="doUnequip(e.id)">[卸下]</a></td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.equipped.length">(未穿戴装备)</div>
          <hr/>
          装备背包:
          <table>
            <tr><th>名称</th><th>类型</th><th>品质</th><th>属性</th><th>要求等级</th><th>操作</th></tr>
            <tr v-for="e in officerDetail.bag" :key="'db' + e.id">
              <td>{{ e.name }}</td>
              <td>{{ e.type }}</td>
              <td>{{ e.tier_name }}</td>
              <td>
                <span v-if="e.military">军事+{{ e.military }} </span>
                <span v-if="e.logistics">后勤+{{ e.logistics }} </span>
                <span v-if="e.learning">学习+{{ e.learning }}</span>
              </td>
              <td>{{ e.level }}</td>
              <td>
                <a v-if="!e.worn" href="javascript:;" @click="doEquip(e)">[穿戴]</a>
                <span v-else class="gray">已穿戴</span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!officerDetail.bag.length">(背包暂无装备)</div>
          <div class="old-line"><a href="javascript:;" @click="go('acade')">[返回军官]</a></div>
        </div>
      </template>

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
        <a href="javascript:;" :class="{ on: cur === 'acade' }" @click="goTreasure()">宝物</a>
      </div>
      <div class="old-line ezfy-bottom-nav">
        <a href="javascript:;" :class="{ on: cur === 'activity' }" @click="go('activity')">活动</a>
        <a href="javascript:;" :class="{ on: cur === 'welfare' }" @click="go('welfare')">福利</a>
        <a href="javascript:;" :class="{ on: cur === 'notices' }" @click="go('notices')">公告</a>
        <a href="javascript:;" :class="{ on: cur === 'exchange' }" @click="go('exchange')">交易</a>
        <a href="javascript:;" :class="{ on: cur === 'liaison' }" @click="go('liaison')">联络</a>
        <a href="javascript:;" :class="{ on: cur === 'cityhall' }" @click="go('cityhall')">市政</a>
        <a href="javascript:;" :class="{ on: cur === 'chat' }" @click="go('chat')">聊天</a>
        <a href="javascript:;" :class="{ on: cur === 'home' }" @click="go('home')">首页</a>
      </div>
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
      resNames: RES_NAMES,
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
      troopsData: { troops: [], queues: [], wounded: [], cfgs: [], pop: 0, pop_used: 0, wall_level: 0, train_discount: 0 },
      // ★ 占用人口（只有训练队列里没出厂的新兵占）：/view 与 /troops 都会下发，谁后到用谁
      popUsed: 0,
      // 与 popUsed 同一次响应里的「人口」，保证「空闲 = 人口 - 占用」恒成立
      // （分开取 city.pop / troopsData.pop 会出现 1000-996=5 这种对不上的显示）
      cityPop: 0,
      techsData: { techs: [], academy: 0 },
      wildlands: [],
      occupies: [],
      queues: [],
      marching: 0,
      occupying: 0,
      unreadReports: 0,
      reports: [],
      // ★ 军情三区分页：默认每页 5 条
      dynPage: 1, dynSize: 5,
      repPage: 1, repSize: 5,
      notices: [],
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
      chatNotices: [],
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
      // 计谋(复刻原版 acade/scheme.html, 共 12 条, 消耗信号弹)
      schemes: [
        { name: '恫疑虚喝', des: '恫疑虚喝', bullet: 4 },
        { name: '隐真示假', des: '隐真示假', bullet: 4 },
        { name: '十面埋伏', des: '十面埋伏', bullet: 7 },
        { name: '欲擒故纵', des: '欲擒故纵', bullet: 7 },
        { name: '偷梁换柱', des: '偷梁换柱', bullet: 6 },
        { name: '反客为主', des: '反客为主', bullet: 6 },
        { name: '先发制人', des: '使我军与敌军城市直接进入可战争状态，可战争时间为军官学识×1分钟，最多持续6小时，中计城市6小时内不再中计', bullet: 12 },
        { name: '未雨绸缪', des: '未雨绸缪', bullet: 12 },
        { name: '虚实相乱', des: '虚实相乱', bullet: 4 },
        { name: '调虎离山', des: '调虎离山', bullet: 4 },
        { name: '各个击破', des: '各个击破', bullet: 8 },
        { name: '舍车保帅', des: '舍车保帅', bullet: 8 }
      ],
      welfare: { rewards: [], gifts: {} },
      rankData: { prestige: [], troops: [], corps: [], ranks: [] },
      orders: [],
      buildZone: 'm',
      buildSel: null,
      curReport: null,
      reportTab: 1,
      reportWord: '',
      reportCounts: {},
      // ★ 自己城市的雷达站等级（决定「来袭/被侦查」预警能不能收到）
      reportRadar: 0,
      dynamics: [],
      curOrder: null,
      showDetail: true,
      corpsList: [],
      myCorps: null,
      corpsMembers: [],
      corpsChats: [],
      corpsName: '',
      corpsMsg: '',
      kickUserId: 0,
      mallItems: [],
      // ★ 第九轮：商城分类 + 分页 + 钻石余额
      mallCatsList: [], mallCat: '', mallPage: 1, mallPageSize: 10, mallDiamond: 0,
      // ★ 单次购买数量上限（管理端「建筑上限配置」页维护，默认 9999；原来写死 99）
      mallBuyMax: 9999,
      bagItems: [],
      bagOfficers: [],
      bagSkills: [],
      useItem: null,
      useCount: 1,
      useOfficerId: 0,
      useSkillId: 0,
      buyItem: null,
      buyCount: 1,
      buyPayWith: 'gold', // ★ 双渠道道具的支付方式选择（gold / diamond）
      exchangeOrders: [],
      exchangeMine: [],
      exchangeGold: 0,
      acadeTab: 'officer',
      officerData: { officers: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0 },
      recruitData: { candidates: [], academy_level: 0, staff_level: 0, capacity: 0, used: 0, gold: 0, refresh_left: 0, refresh_limit: 5 },
      skillData: { skills: [], officers: [], gold: 0 },
      equipData: { bag: [], all: [] },
      generalData: { generals: [] },
      officerDetail: { officer: null, skills: [], all_skills: [], equipped: [], bag: [], gold: 0 },
      sellType: '1',
      sellCount: 0,
      sellPrice: 0,
      trainSel: null,
      trainCount: 10,
      trainSplit: false,
      trainMode: 'troop', // troop=训练(createTroop) / defence=建造(createDefence)
      troopViewId: 0,     // 兵种详情页当前兵种 id
      troopViewBack: 'troops', // 兵种详情页 [返回] 回到哪一页
      // 页面内消息 + 内联确认（替代 alert/confirm/prompt 弹窗）
      msgs: [], askBox: { show: false, text: '', input: false, placeholder: '', value: '' },
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
  computed: {
    // ★ 二级导航（资源/军官/军队/科技/城防/统帅）：只在对应页面显示，位置固定在页面顶部
    //   军队的几个子页（兵种/兵种详情/训练/工厂）也算「军队」，一并显示，保持导航不中断
    isArmyPage () {
      return ['troops', 'troop', 'troopview', 'trainpre', 'factory'].indexOf(this.cur) >= 0
    },
    showSubnav () {
      return ['builds', 'acade', 'techs', 'defence', 'info'].indexOf(this.cur) >= 0 || this.isArmyPage
    },
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
      return this.buildings.filter(b => inZone(b.type))
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
    // 战俘营: 未出征的俘虏
    captiveOfficers () {
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
    //   管理端「建筑上限配置」页可维护，默认 50）。
    //   ⚠️ 这里**不能**在没有数据时直接返回 50 去夹输入值 —— 那正是「配了 999 只能用 50」的 bug：
    //   进页面时 orderCalc 还是 null，一改数字就被夹回 50，后面再点[计算]也救不回来了。
    orderCapMax () {
      const m = this.gatherCfg.max || (this.orderCalc && this.orderCalc.gather_max)
      return m > 0 ? m : 50
    },
    // 真正可用的上限 = min(管理端上限, 背包实际持有量)
    gatherMax () { return Math.min(this.orderCapMax, this.gatherCount) },
    // ★ 每个集结令提升的出征上限，同样以接口下发为准（读 ezfy_cfg_item.param1，缺省 10 万）
    orderCapPer () {
      const p = this.gatherCfg.per || (this.orderCalc && this.orderCalc.gather_per)
      return p > 0 ? p : 100000
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
      return this.buildings.filter(b => b.type === 1 || b.type === 2 || b.type === 3).length
    },
    areaCap () {
      return 33
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
    // ★ 军情三区分页（默认每页 5 条，可上一页/下一页）
    dynTotalPages () {
      return Math.max(1, Math.ceil(this.dynamics.length / this.dynSize))
    },
    dynPaged () {
      const p = Math.min(Math.max(1, this.dynPage), this.dynTotalPages)
      return this.dynamics.slice((p - 1) * this.dynSize, p * this.dynSize)
    },
    repTotalPages () {
      return Math.max(1, Math.ceil(this.reports.length / this.repSize))
    },
    repPaged () {
      const p = Math.min(Math.max(1, this.repPage), this.repTotalPages)
      return this.reports.slice((p - 1) * this.repSize, p * this.repSize)
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
    // 沉浸式卡控①: 游戏内任何 <a href="/..."> 都不允许跳出 /games/ezfy 回到家园站点
    // (捕获阶段拦截, 只拦站内绝对路径链接)
    document.addEventListener('click', this.blockEscape, true)
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
  },
  beforeDestroy () {
    document.body.classList.remove('ezfy-immersive')
    document.removeEventListener('click', this.blockEscape, true)
    if (this._onBack) window.removeEventListener('popstate', this._onBack)
    if (this.timer) clearInterval(this.timer)
  },
  methods: {
    // 退出游戏回家园 —— 游戏内唯一的合法出口(顶部导航的「家园」)。
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
      this.notify('游戏内不能跳回家园。如需离开游戏, 请点顶部导航。')
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
        const qs = params.toString()
        const next = base + (qs ? '?' + qs : '')
        history.replaceState(history.state, '', location.pathname + location.search + next)
      } catch (e) { /* URL 同步失败不影响游戏本身 */ }
    },

    // restoreFromUrl 启动时从 URL 读回页面并重放加载逻辑
    restoreFromUrl () {
      let cur = ''
      let res = ''
      try {
        const hash = location.hash || ''
        const qi = hash.indexOf('?')
        if (qi >= 0) {
          const params = new URLSearchParams(hash.slice(qi + 1))
          cur = params.get('cur') || ''
          res = params.get('res') || ''
        }
      } catch (e) {}
      if (res) this.resType = res
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
    // 宝物：原版 /ezfy/acadeIndex，本项目对应「学院 → 装备」页（我的装备 + 装备图鉴）
    goTreasure () {
      this.cur = 'acade'
      this.switchAcade('equip')
    },
    go (t) {
      if (t.indexOf('res/') === 0) {
        this.resType = t.slice(4)
        this.cur = 'res'
        this.syncUrl()
        this.loadRes()
        return
      }
      this.cur = t
      this.syncUrl()
      if (t === 'home') { this.load(); this.loadWelfare(); this.loadHomeChats() }
      else if (t === 'troops' || t === 'troop' || t === 'defence' ||
               t === 'troopview' || t === 'trainpre' || t === 'troopstat') this.loadTroops()
      else if (t === 'hq') { this.loadTroops().then(() => this.loadTargets()); this.loadOrders() }
      else if (t === 'techs') this.loadTechs()
      else if (t === 'map') { this.loadMap(); this.loadStars() }
      else if (t === 'reports') { this.curReport = null; this.switchReportTab(this.reportTab) }
      else if (t === 'mail') { this.loadMails(); this.loadFriends(); this.loadPmCandidates(); this.loadPmConvs() }
      else if (t === 'friends') this.loadFriends()
      else if (t === 'liaison') this.loadLiaison()
      else if (t === 'tasks') this.loadTasks()
      else if (t === 'welfare') this.loadWelfare()
      else if (t === 'rank') this.loadRank()
      // 城市列表页要显示「军衔可建城数」，所以也拉一次军衔数据
      else if (t === 'cities') this.loadRank()
      else if (t === 'bag') this.loadBag()
      else if (t === 'mall') this.loadMall(true)
      else if (t === 'exchange') this.loadExchange()
      else if (t === 'corps') this.loadCorps()
      else if (t === 'orders') this.loadOrders()
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
          this.city = d.city
          this.cities = d.cities
          this.continent = d.continent
          this.cityKindRaw = d.city_kind || ''
          this.cityIsSea = !!d.is_sea
          this.protectedUntil = d.protected
          this.boostUntil = d.boost
          this.buildings = d.buildings
          this.buildingPool = d.building_pool || []
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
          // ★ 集结令配置（管理端可配，默认 50）：跟着 /view 一起下发，
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
      api.get('/games/ezfy/techs').then(r => {
        if (r.code === 0) this.techsData = r.data
      })
    },
    loadChats () {
      api.get('/games/ezfy/chat?channel=' + this.chatChannel +
              '&page=' + this.chatPage + '&size=' + this.chatSize).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.worldChats = d.chats || []
          this.chatNotices = d.notices || []
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
            this.chatNotices = []
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
      // category: 1 军情警讯 2 战斗报告(战报查询)
      const cat = this.reportTab === 2 ? 1 : 2
      let url = '/games/ezfy/reports?category=' + cat
      if (this.reportWord) url += '&word=' + encodeURIComponent(this.reportWord)
      api.get(url).then(r => {
        if (r.code === 0) {
          this.reports = r.data.reports || []
          this.reportCounts = r.data.counts || {}
          this.reportRadar = r.data.radar || 0
          this.repPage = 1
        }
      })
    },
    // ---- 军队动态 ----
    loadDynamics () {
      api.get('/games/ezfy/reports/dynamics').then(r => {
        if (r.code === 0) {
          this.dynamics = r.data.dynamics || []
          this.dynPage = 1
        }
      })
    },
    switchReportTab (t) {
      this.reportTab = t
      this.curReport = null
      // ★ 切换分区时回到第 1 页，避免停在上一次的分页位置看到空白
      this.repPage = 1
      this.dynPage = 1
      if (t === 1) this.loadDynamics()
      else this.loadReports()
    },
    doCollectAll () {
      api.post('/games/ezfy/wild/collect-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
        } else this.notify(r.msg)
      })
    },
    // ★ 一键收获：只结算产出装进部队，**不召回**
    async doHarvestAll () {
      if (!await this.ask('确定收获所有采集部队吗？（只把产出装进部队，资源要「召回」才会运回城里）')) return
      api.post('/games/ezfy/wild/harvest-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    // ★ 一键召回：部队返航，到达时把待带回资源运回城里
    async doRecallAll () {
      if (!await this.ask('确定召回所有采集部队吗？部队返航到达后，待带回的资源才会入库。')) return
      api.post('/games/ezfy/wild/recall-all', {}).then(r => {
        if (r.code === 0) {
          this.notify(r.msg)
          this.loadDynamics()
          this.load()
        } else this.notify(r.msg)
      })
    },
    loadTasks () {
      api.get('/games/ezfy/tasks').then(r => {
        if (r.code === 0) this.taskGroups = r.data.groups
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
    loadBag () {
      api.get('/games/ezfy/bag').then(r => {
        if (r.code === 0) {
          this.bagItems = r.data.items
          this.bagOfficers = r.data.officers || []
          this.bagSkills = r.data.skills || []
        }
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
          // ★ 单次购买上限（管理端可配，默认 9999）
          this.mallBuyMax = parseInt(r.data.buy_max) > 0 ? parseInt(r.data.buy_max) : 9999
          if (this.mallCat && this.mallCatsList.indexOf(this.mallCat) < 0) this.mallCat = ''
          if (resetPage) this.mallPage = 1
          else if (this.mallPage > this.mallTotalPages) this.mallPage = this.mallTotalPages
        }
      })
    },
    loadExchange () {
      api.get('/games/ezfy/exchange').then(r => {
        if (r.code === 0) {
          this.exchangeOrders = r.data.orders
          this.exchangeMine = r.data.mine
          this.exchangeGold = r.data.gold
        }
      })
    },
    loadCorps () {
      api.get('/games/ezfy/corps/list').then(r => {
        if (r.code === 0) {
          this.corpsList = r.data.corps
          this.myCorps = r.data.my_corps
        }
      })
      api.get('/games/ezfy/corps/members').then(r => {
        if (r.code === 0) {
          this.corpsMembers = r.data.members
          // ★ 后端下发「我在军团的职位」，副团长也能发军团邮件
          this.myCorpsTitle = r.data.my_title || ''
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
      // 备注名用「地形名(等级)」, 坐标由列表模板统一拼, 别在这里重复带上
      const def = this.cellText(this.selCell)
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
      api.post('/games/ezfy/build', { building_id: b.building_id || b.bid }).then(r => {
        this.alert(r, '建造命令已下达')
        if (r.code === 0) this.load()
      })
    },
    doUpgrade (b) {
      api.post('/games/ezfy/building/upgrade', { record_id: b.id }).then(r => {
        this.alert(r, '建筑已开始升级')
        if (r.code === 0) this.load()
      })
    },
    doMaxLevel (b) {
      api.post('/games/ezfy/building/max-level', { record_id: b.id }).then(r => {
        this.alert(r, '已升到最高级')
        if (r.code === 0) this.load()
      })
    },
    doDeleteBuilding (b) {
      api.post('/games/ezfy/building/delete', { record_id: b.id }).then(r => {
        this.alert(r, '建筑已拆除')
        if (r.code === 0) this.load()
      })
    },
    doSpeedBuilding () {
      api.post('/games/ezfy/building/speed', { minutes: 10 }).then(r => {
        this.alert(r, '当前没有正在施工的建筑')
        if (r.code === 0) this.load()
      })
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
      api.post('/games/ezfy/city/convene', {}).then(r => this.alert(r, '部队已集合'))
    },
    doPlacate () {
      api.post('/games/ezfy/city/placate', {}).then(r => this.alert(r, '安抚完成'))
    },
    doTax () {
      api.post('/games/ezfy/city/tax', { tax_rate: parseInt(this.taxInput) || 0 }).then(r => this.alert(r, '税率已调整'))
    },
    doRename () {
      api.post('/games/ezfy/city/rename', { name: this.renameInput }).then(r => this.alert(r, '城市已更名'))
    },
    doCreateCity () {
      api.post('/games/ezfy/city/create', { x: parseInt(this.newCityX) || 0, y: parseInt(this.newCityY) || 0 }).then(r => this.alert(r, '新城建造成功'))
    },
    // 摧毁自己的城市（至少保留一座；摧毁当前城会自动切到剩下的城）
    // ★ 军情分页翻页（which: 'dyn' 军队动态 / 'rep' 战报列表）
    pagerGo (which, delta) {
      if (which === 'dyn') {
        this.dynPage = Math.min(this.dynTotalPages, Math.max(1, this.dynPage + delta))
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
          if (r.data && r.data.msg) this.alert(r.data)
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
    doRecover (w) {
      api.post('/games/ezfy/troops/recover', { troop_id: w.troop_id, type: w.type }).then(r => this.alert(r, '伤兵已恢复'))
    },
    doRecoverAll (t) {
      api.post('/games/ezfy/troops/recover', { all: true, type: t }).then(r => this.alert(r, '伤兵已恢复'))
    },
    // ---- 科技 ----
    doResearch (t) {
      api.post('/games/ezfy/techs/research', { tech_id: t.tech_id }).then(r => this.alert(r, '科技研究已开始'))
    },
    doSpeedTech () {
      api.post('/games/ezfy/techs/speed', { minutes: 10 }).then(r => {
        this.alert(r, '没有研究中的科技')
        if (r.code === 0) this.loadTechs()
      })
    },
    async doCancelTech (t) {
      if (!await this.ask('确定取消研究「' + t.name + '」吗？本次消耗将全额退还。')) return
      api.post('/games/ezfy/techs/cancel', { tech_id: t.tech_id }).then(r => {
        this.alert(r, '研究已取消，消耗已全额退还')
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
      api.get('/games/ezfy/orders/' + o.id).then(r => {
        if (r.code === 0) {
          this.curOrder = r.data
          this.cur = 'orderview'
        }
      })
    },
    jumpReport (rid) {
      api.get('/games/ezfy/reports/' + rid).then(res => {
        if (res.code === 0) {
          this.curReport = res.data.report
          this.showDetail = false
          this.cur = 'reports'
        }
      })
    },
    // 点开战报: 拉详情 + 标已读(原缺失该方法, 导致战报点不开)
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
    orderStatusText (o) {
      if (o.status === 0) return '行进中 ' + this.remain(o.arrive_time)
      if (o.status === 1) return (o.order_type === 7 ? '驻守采集' : '已到达')
      if (o.status === 2) return '返回中 ' + this.remain(o.return_time)
      if (o.status === 3) return '已完成'
      return '全队阵亡'
    },
    // ---- 地图/出征 ----
    cellText (cell) {
      // 复刻 map/index.html: 格子文案为「名称(等级)」；★ 现在每格第二行统一显示坐标，
      //   所以这里一律只返回「名称」部分，本城也不再拼 (x,y)，避免和下面那行重复。
      if (cell.mine) return this.city.name
      // ★ 用户反馈：地图上别人的城市原来一律显示「城」，看不出是谁的城。
      //   后端已下发 name(城市名) + owner(城主昵称)，这里直接展示。
      if (cell.area_type === 3) {
        const nm = cell.name || '城'
        return cell.owner ? nm + '(' + cell.owner + ')' : nm
      }
      // 活动目标: 复刻 mapView.html 的「活动野地N级 / 活动寇N级 / 特殊城市N级」
      if (cell.act_type === 1) return '活动(' + cell.act_level + ')'
      if (cell.act_type === 2) return '活动寇(' + cell.act_level + ')'
      if (cell.act_type === 3) return '特殊(' + cell.act_level + ')'
      if (cell.name === '寇城(废墟)') return '墟'
      if (cell.area_type === 2) return '寇(' + cell.level + ')'
      if (cell.terrain === 8) return '海(' + cell.level + ')'
      // 陆地野地按地形名显示(平原/草原/森林/盆地/丘陵/沼泽/山地)
      return (cell.terrain_name || '野') + '(' + cell.level + ')'
    },
    // ★ 格子悬浮提示：城市名字在格子里会被截断，鼠标悬停看全称。
    //   坐标已固定显示在格子第二行，这里不再重复拼。
    cellTip (cell) {
      if (cell.area_type === 3 && !cell.mine) {
        return (cell.name || '城市') + (cell.owner ? ' · 城主 ' + cell.owner : '') +
          ' (' + cell.x + ',' + cell.y + ')'
      }
      return this.cellText(cell)
    },
    cellClass (cell) {
      if (cell.mine) return 'ezfy-mine'
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
      this.selCell = cell
      this.selDetail = null
      this.warText = ''
      this.warStatus = 0
      this.cur = 'wildview'
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
        }
      })
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
        if (r.code === 0) this.orderCalc = r.data
        else this.alert(r, '计算失败')
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
      api.post('/games/ezfy/order', this.orderBody()).then(r => {
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
          this.cur = 'orders'
          this.loadOrders()
        } else this.notify(r.msg)
      })
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
      // ★ 加入成功后必须重新拉军团数据，否则页面还显示「未加入」
      api.post('/games/ezfy/corps/join', { corps_id: cp.id })
        .then(r => this.alert(r, '已加入联盟', () => this.loadCorps()))
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
    openBuy (it) {
      this.buyItem = it
      this.buyCount = 1
      // ★ 双渠道道具默认用黄金（多数玩家手上黄金比钻石多）
      this.buyPayWith = it.dual_pay ? 'gold' : (it.is_diamond ? 'diamond' : 'gold')
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
          this.loadMall()
          this.loadBag()
        } else this.notify(r.msg || '购买失败')
      })
    },
    needOfficer (it) {
      return it.item_type === 10 || it.item_type === 11 || it.item_type === 12
    },
    openUse (it) {
      this.useItem = it
      this.useCount = 1
      this.useOfficerId = 0
      this.useSkillId = 0
    },
    doUse (it) {
      const body = { cfg_id: it.cfg_id, count: parseInt(this.useCount) || 1 }
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
      }).then(r => this.alert(r, '挂单已发布'))
    },
    doExchangeBuy (e) {
      api.post('/games/ezfy/exchange/buy', { id: e.id }).then(r => this.alert(r, '购买成功'))
    },
    doExchangeCancel (e) {
      api.post('/games/ezfy/exchange/cancel', { id: e.id }).then(r => this.alert(r, '挂单已撤销'))
    },
    // ---- 任务/福利 ----
    doAward (t) {
      api.post('/games/ezfy/tasks/award', { task_id: t.id }).then(r => this.alert(r, '奖励已领取'))
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
    fmtN (n) {
      const v = Number(n)
      if (!isFinite(v)) return '0'
      return v.toLocaleString('en-US')
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
      // ★ 用户要求：不需要玩家确定的提示 3 秒后自动消失（也可点 [关闭] 手动收起）
      setTimeout(() => this.closeMsg(id), 3000)
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
      if (tab === 'officer' || tab === 'mayor' || tab === 'captive') {
        api.get('/games/ezfy/officers').then(r => {
          if (r.code === 0) this.officerData = r.data
        })
      } else if (tab === 'search') this.loadRecruit()
      else if (tab === 'skill') this.loadAcadeSkills()
      else if (tab === 'equip') this.loadAcadeEquip()
      else if (tab === 'generals') this.loadAcadeGenerals()
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
      api.get('/games/ezfy/officers/equipments').then(r => {
        if (r.code === 0) this.equipData = r.data
      })
    },
    loadAcadeGenerals () {
      api.get('/games/ezfy/officers/generals').then(r => {
        if (r.code === 0) this.generalData = r.data
      })
    },
    openOfficer (id) {
      this.cur = 'officerdetail'
      this.loadOfficerDetail(id)
    },
    loadOfficerDetail (id) {
      api.get('/games/ezfy/officers/' + id).then(r => {
        if (r.code === 0) this.officerDetail = r.data
      })
    },
    // 军校直接使用招生简章刷新（不占每日次数；不用跳背包）
    async doUseRecruitTicket () {
      if (!this.recruitData.academy_level) { this.notify('需要先建造军校'); return }
      if (this.bagCount(13) <= 0) { this.notify('没有「招生简章」，可到商城购买'); return }
      if (!await this.ask('确认使用「招生简章」×1 刷新军校候选名将吗？（不占用每日次数）')) return
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
    doGrant () {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/grant', {}).then(r => {
        if (r.code !== 0) this.notify(r.msg || '赏赐失败')
        this.loadOfficerDetail(id)
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
    doUnequip (equipId) {
      const id = this.officerDetail.officer.id
      api.post('/games/ezfy/officers/' + id + '/equip', { equip_id: equipId, op: 'off' }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '卸下失败')
        this.loadOfficerDetail(id)
      })
    },
    doPosition (o, pos) {
      api.post('/games/ezfy/officers/' + o.id + '/position', { position: pos }).then(r => {
        if (r.code !== 0) this.notify(r.msg || '任命失败')
        api.get('/games/ezfy/officers').then(rr => {
          if (rr.code === 0) this.officerData = rr.data
        })
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
  /* ★ 字体族: 完全照原版 ezfy.css 末尾那句
     `body,button,input,select,textarea,h1..h6{font-family:'微软雅黑'}`。
     之前本页从未声明 font-family → 一路继承到浏览器默认(serif), 与全站其它页字体分裂。
     追加的中文回退只在用户机器没装微软雅黑时才会用到(Windows 默认都有)。 */
  font-family: '微软雅黑', 'Microsoft YaHei', 'PingFang SC', 'Hiragino Sans GB', 'Heiti SC', sans-serif;
  /* ★ 16px 基准: 用户最终决定「原版 19px 太大，正文用 16px」。★ 不要再统一放大到 18px！
     演变: 原版 ezfy.css `*{font-size:19px}` → 照搬 19px 用户嫌大 → 试过统一 18px
           → 用户反馈「字体太大 看着笨笨的」→ 恢复「有层次」的小字阶梯(当前值)。
     配套阶梯: 标题栏 18 / 小标题·导航 17 / 正文·表格·表单 16 / 次要信息·页脚 13~15 /
     提示条·确认条·战报·按钮 14 / 地图格 11(固定格尺寸,勿动)。 */
  font-size: 16px;
  line-height: 1.5;
  /* 根容器左右不再用负 margin: 会溢出 #app 产生横向滚动条.
     铺满由内部 .title-bar 的 margin:0 -8px 抵消 padding 实现 */
  margin: 0;
  padding: 0 8px 20px;
}
/* 表单控件/按钮默认不继承字体族, 显式补上(原版也是 body,button,input,select,textarea 一起设) */
.ezfy-page button,
.ezfy-page input,
.ezfy-page select,
.ezfy-page textarea {
  font-family: inherit;
}
.ezfy-page .home-wrap {
  font-size: 16px;
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
  /* ★ 用户反馈「聊天/邮箱/军情/任务/好友/首页 字体有点小」→ 15 → 17 */
  font-size: 17px;
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
  margin: 0 1px;
  /* ★ 字号与 .top-nav a 统一为 17px */
  font-size: 17px;
  color: #004299;
}
.ezfy-page .ezfy-subnav a.on { color: #000; font-weight: bold; }
/* ★ 用户「点进去 资源/军官/军队/科技/城防/统帅 左边 和 聊天 的『聊』字对齐」：
   首链接左侧 padding/margin 归零 —— 与 .top-nav a:first-child 同源。
   两者父容器(.old-line / .top-nav)左右 padding 都是 0，归零后文字左边缘必定对齐。 */
.ezfy-page .ezfy-subnav a:first-child { margin-left: 0; padding-left: 0; }
/* 军衔/排行页所有表格：数据水平 + 垂直居中（用户要求）*/
.ezfy-page .ezfy-rank-table th,
.ezfy-page .ezfy-rank-table td {
  text-align: center;
  vertical-align: middle;
}
/* 自适应高度文本域（私聊等） */
.ezfy-page .ezfy-auto-textarea {
  width: 100%;
  box-sizing: border-box;
  min-height: 44px;
  max-height: 160px;
  padding: 4px 6px;
  font-size: 14px;
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
/* 页面内消息区（替代 alert 弹窗） */
.ezfy-page .ezfy-msgs { margin: 4px 0 2px; }
/* ★ 第九轮：用户端通用分页条（商城等列表页） */
.ezfy-page .ezfy-pager { margin: 6px 0 2px; }
.ezfy-page .ezfy-pager a { margin-right: 8px; }
.ezfy-page .ezfy-pager a.disabled { color: #bbb; text-decoration: none; cursor: default; }
.ezfy-page .ezfy-pager span { margin-right: 8px; }
.ezfy-page .ezfy-msg {
  padding: 4px 6px; margin: 3px 0; border-radius: 3px;
  font-size: 14px; line-height: 1.5; border-left: 3px solid #999; background: #f5f5f5;
}
.ezfy-page .ezfy-msg-ok { border-left-color: #27763c; background: #eef7f0; color: #1d5c2e; }
.ezfy-page .ezfy-msg-error { border-left-color: #c0392b; background: #fdeeec; color: #a02a1e; }
.ezfy-page .ezfy-msg-info { border-left-color: #2f6f9f; background: #eef4fa; color: #235b85; }
.ezfy-page .ezfy-msg-close { margin-left: 6px; color: #888; }
/* 页面内确认条（替代 confirm / prompt 弹窗） */
.ezfy-page .ezfy-ask {
  margin: 6px 0; padding: 8px; border: 1px solid #d8c890;
  background: #fffbe8; border-radius: 4px;
}
.ezfy-page .ezfy-ask-text { font-size: 14px; color: #7a5c10; margin-bottom: 6px; }
.ezfy-page .ezfy-ask-row { margin-top: 4px; }
.ezfy-page .ezfy-ask-ok { font-weight: bold; color: #27763c; margin-right: 12px; }
.ezfy-page .ezfy-ask-cancel { color: #999; }
/* 底部 15 项导航(复刻原版 cityHome.html 的两行) */
.ezfy-page .ezfy-bottom-nav {
  padding: 1px 0;
  font-size: 17px;
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
  font-size: 16px;   /* 与正文同号 */
  color: #666;
}
.ezfy-page .acade-tab a { color: #2f4156; }
.ezfy-page .acade-tab a.on { color: #c0392b; font-weight: bold; }
.ezfy-page .use-box {
  margin: 4px 0 6px 8px;
  padding: 4px 6px;
  border-left: 2px solid #d8d5cc;
  line-height: 1.9;
}
.ezfy-page .panel-title {
  /* ★ 统一字号阶梯: 正文 17 / 小标题 18 / 标题栏 18。原先 17 与正文同级, 会看不出层级 */
  font-size: 17px;
  font-weight: bold;
  color: #2f4156;
  margin: 6px 0 2px;
}
.ezfy-page .old-line { padding: 2px 0; word-break: break-all; }
/* ★ 出征确认页(orderpre)分区：① ② ③ … 小标题 + 等宽列网格。
   原来所有内容都是一串 .old-line 平铺，兵种/资源/宿营/计算混在一起，用户反馈「看着好乱」。
   ★ 用户反馈「兵力还是竖着展示，整齐一点」→ 兵力改成 **CSS Grid 等宽列**：
     - 用 grid（不是 flex-wrap）：同一列的宽度完全一致，纵向也对得齐；
     - minmax 让窗口变窄时自动减列（窄屏也不会退化成一兵种一行）；
     - 名称超长用省略号（完整名字放 title），输入框固定宽度，**现有数量独立一列右对齐** ——
       原来是「输入框里塞占位符 0~59108」，框一窄就被截成 0-0，很难看。
   注意：.of-cell 仍保持 .old-line 的 2px 上下 padding，整页行距节奏不变。 */
.ezfy-page .of-sec {
  font-size: 16px;
  font-weight: bold;
  color: #2f4156;
  margin: 8px 0 2px;
  padding-bottom: 1px;
  border-bottom: 1px dashed #d8d5cc;
}
/* 分区标题里的补充说明（小一号、不抢视觉） */
.ezfy-page .of-sec .of-hint { font-size: 14px; font-weight: normal; color: #8a8a8a; }
.ezfy-page .of-grid {
  display: grid;
  gap: 0 16px;
  align-items: center;
}
/* 兵力：★ 用户要求「分三列、对齐」→ **固定 3 列等宽**（原来是 auto-fill，
   宽屏会变成 4 列、窄屏 2 列，列数随窗口乱跳，用户觉得「丑、不齐」）。
   3 列等宽 ⇒ 每一格宽度完全一致 ⇒ 名称 / 输入框 / 现有数量 三列在所有行里 x 严格一致。
   列宽下限 = 名称 11em(176) + 输入 70 + 现有 4.4em(70) + 间距 10 ≈ 326px，
   所以窗口 < 1100px 时降到 2 列、< 700px 时降到 1 列，保证兵种名不会被截断
   （名字被截成「埃塞克…」玩家就认不出兵种了）。 */
.ezfy-page .of-grid-troop { grid-template-columns: repeat(3, minmax(0, 1fr)); }
@media (max-width: 1100px) {
  .ezfy-page .of-grid-troop { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 700px) {
  .ezfy-page .of-grid-troop { grid-template-columns: minmax(0, 1fr); }
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
   .ezfy-msgs / .ezfy-ask（操作结果提示条），负 margin 会把这 4px 从提示条的下边距里扣掉，
   公告行就会贴住提示条（实测只剩 5px）。 */
.ezfy-page .ezfy-notices { margin: 0 0 2px; }
.ezfy-page .city-name { font-size: 16px; font-weight: bold; color: #2f4156; }
/* ★ 表格默认用「原版模板的朴素样式」: 宽度按内容自适应(不 width:100%)、无边框。
   原版 templates 里绝大多数表格都没有任何 CSS, 就是浏览器默认样式;
   之前统一 width:100% + 虚线下边框, 会把表格拉满整行, 用户会觉得「太长 / 还是表格」。 */
.ezfy-page table {
  width: auto;
  max-width: 100%;
  border-collapse: collapse;
  /* ★ 表格字号对齐正文(17px): 之前 15px 比正文小两号, 表格密集的页面看起来字体忽大忽小 */
  font-size: 15px;
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
.ezfy-page table th {
  color: #2f4156;
  font-weight: bold;
  white-space: nowrap;
  padding-bottom: 4px;
}
/* 返回按钮与 [造兵]/[建防]/[退出军团] 等普通操作链接同款: 纯文字链接, 无填充 */
.ezfy-page .bottom-nav { margin-top: 10px; padding: 4px 0; text-align: left; }
.ezfy-page .footer { text-align: center; font-size: 13px; color: #999; padding: 4px 0 10px; }
.ezfy-page .logo-title { height: 14px; vertical-align: -2px; }
.ezfy-page .red { color: #c0392b; }
.ezfy-page .gray { color: #999; }
.ezfy-page .green { color: #27763c; }
.ezfy-page .orange { color: #b8860b; }
.ezfy-page input[type="text"],
.ezfy-page input[type="number"],
.ezfy-page input:not([type]),
.ezfy-page select,
.ezfy-page textarea {
  border: 1px solid #999;
  border-radius: 0;
  padding: 3px 4px;
  /* ★ 表单 15px：比正文(16px)小一号，避免输入框把行撑高 */
  font-size: 15px;
  background: #fff;
  color: #333;
}
.ezfy-page button {
  border: 1px solid #888;
  border-radius: 0;
  background: #e8e5dd;
  color: #333;
  /* ★ 按钮 14px：比正文小一号，视觉上不抢正文 */
  font-size: 14px;
  padding: 2px 8px;
  cursor: pointer;
}
.ezfy-page button:hover { background: #d8d5cc; }
.ezfy-page .report-pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: inherit;
  font-size: 14px;
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
  width: auto;
  max-width: 100%;
  border-collapse: separate;
  /* ★ 用户要求「坐标和坐标之间间隔小了，上下左右都再来点」→ 8px 3px 放大到 12px 6px
     （横向 8→12，纵向 3→6；格子变两行后纵向 3px 太挤） */
  border-spacing: 12px 6px;
  margin: 8px 0;             /* 表格本身靠左(不要整表居中) */
}
.ezfy-page .ezfy-map-table td {
  padding: 0;
  border: 0;
  text-align: center;        /* 居中指的是「表格里的内容」居中 */
  vertical-align: top;       /* 两行格子按顶对齐, 免得行高不一导致上下抖动 */
  white-space: nowrap;
}
.ezfy-page .ezfy-map-table a {
  display: inline-block;     /* 改成块级容器, 才能装上下两行 */
  padding: 0;
  margin: 0;
  /* ★ 第一行(名称/等级，如「海(8)」)的字号：用户反馈「小 1 号」→ 16px → 15px
     （原来是和正文 .old-line 同号的 16px；坐标行有自己独立的 11px，不受这里影响）。
     窄屏同理 14px → 13px，见下面媒体查询。 */
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
.ezfy-page .ezfy-map-table a .ezfy-cell-name { display: block; }
/* 第二行：坐标 (x,y)。★ 用户要求「坐标上颜色，不然玩家不知道能点」→ 站内链接蓝 #0645ad；
   ★ 用户反馈「坐标字体有点大」→ 桌面 12px → 11px（窄屏 11 → 10，见下面的媒体查询）；
   随后又要求「第一行小 1 号、坐标这行不动」，所以这里保持 11px 不变。 */
.ezfy-page .ezfy-map-table a .ezfy-cell-xy {
  display: block;
  font-size: 11px;
  line-height: 1.25;
  font-weight: normal;       /* 本城/活动城名字加粗, 坐标不跟着加粗 */
  color: #0645ad;
}
/* 本城加粗标一下, 其余一律朴素文字 */
.ezfy-page .ezfy-map-table a.ezfy-mine { font-weight: bold; color: #c0392b; }
/* 活动目标配色照 mapView.html: 活动野地橙 / 活动寇城品红 / 特殊城市红 */
.ezfy-page .ezfy-map-table a.ezfy-act-wild { font-weight: bold; color: #ff6600; }
.ezfy-page .ezfy-map-table a.ezfy-act-kou { font-weight: bold; color: #ff00ff; }
.ezfy-page .ezfy-map-table a.ezfy-act-city { font-weight: bold; color: #d00000; }
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

/* ============ WAP 窄屏适配(手机) ============
   目标: 360px / 320px 下不出现横向溢出, 表格不挤成一坨。
   实测基准: iPhone SE 320、常见安卓 360/390。
   ★ 窄屏整体缩一档(表格 13 / 正文 14 / 标题 15)，与桌面端保持同一层次关系。 */
@media (max-width: 420px) {
  .ezfy-page table { font-size: 13px; }
  .ezfy-page table th,
  .ezfy-page table td { padding: 4px 4px; }
  .ezfy-page .old-line { font-size: 14px; line-height: 1.7; }
  .ezfy-page .panel-title { font-size: 15px; }
  .ezfy-page .acade-tab { font-size: 14px; }
  .ezfy-page .ezfy-subnav a { font-size: 15px; }
  .ezfy-page .ezfy-bottom-nav { font-size: 15px; line-height: 2; }
  /* 地图格子: 间距按窄屏收紧, 保证 320px 下 5 列不溢出
     ★ 格子已是两行(名称 + 坐标)，窄屏两行都缩一档，行高收紧免得整表变高太多；
       第一行跟着桌面一起「小 1 号」(14 → 13)，坐标行保持 10px 不动。 */
  .ezfy-page .ezfy-map-table a { font-size: 13px; line-height: 1.25; }
  .ezfy-page .ezfy-map-table a .ezfy-cell-xy { font-size: 10px; }
  .ezfy-page .ezfy-map-table { border-spacing: 6px 4px; }
  /* 坐标查找行在 320px 下也要待在一行内 */
  .ezfy-page .ezfy-map-jump input { width: 62px; margin-right: 2px; }
  /* 方向导航窄屏间距同步收一档(桌面 8px → 窄屏 6px) */
  .ezfy-page .ezfy-dir-nav a { margin-right: 6px; }
  .ezfy-page input, .ezfy-page select { max-width: 100%; }
  /* 出征页格子（名称+输入框+现有）：320px 下单列也要放得下，收窄名称/输入框/数量列 */
  .ezfy-page .of-grid-troop .of-cell .of-name { width: 9em; }
  .ezfy-page .of-cell input.of-num { width: 56px; }
  .ezfy-page .of-cell .of-avail { min-width: 3.6em; }
}
/* 最后一道保险: 万一还有个别元素偏宽, 让它在页面内滚动而不是把整页撑开 */
.ezfy-page .panel { max-width: 100%; overflow-x: auto; }
/* ★ 军情三区分页条（军队动态 / 军情警讯 / 战斗报告，默认每页 5 条） */
.ezfy-page .ezfy-pager {
  margin: 8px 0 4px;
  /* ★ 分页条用「小字」档: 它是辅助信息, 不该和正文抢视线 */
  font-size: 15px;
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
@media (max-width: 420px) {
  .ezfy-page .ezfy-pager { font-size: 13px; }
}
</style>
