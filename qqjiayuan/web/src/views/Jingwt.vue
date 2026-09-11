<template>
  <div class="jwt-wap">
    <!-- 全局「家园/好友/家族/广场/游戏」主导航条由社区框架渲染，此处省略 -->

    <!-- ==================== 首页 ==================== -->
    <template v-if="cur === 'home'">
      欢迎您，{{ g.nick }}({{ g.level }}级)<br/>
      帮派：{{ gangName }} [<a href="javascript:;" @click="go('gang')">加入帮派</a>]<br/>
      <a href="javascript:;" @click="go('daily')">每日礼包</a> <br/>
      <div class="logo"><img src="/static/image/jwt.png" width="130" height="100" alt="精武堂" /></div><br/>
      我的状态|<a href="javascript:;" @click="go('friend')">我的好友</a><br/>
      <a href="javascript:;" @click="go('room')">练功房</a>:{{ practice.practicing === 1 ? remainingText : '未修炼' }}
      <a v-if="practice.practicing !== 1" href="javascript:;" @click="go('practicemode')">开始</a>
      <a v-else href="javascript:;" @click="stopPractice">停止</a><br/>
      经验:{{ g.exp }}/{{ g.next_exp }} <a href="javascript:;" @click="go('upgrade')">升级</a><br/>
      气血:{{ g.cur_hp }}-{{ c.max_hp }}(<a href="javascript:;" @click="go('bag')">提升</a>)<br/>
      气力:{{ g.cur_mp }}-{{ c.max_mp }}(<a href="javascript:;" @click="go('bag')">提升</a>)<br/>
      【<a href="javascript:;" @click="go('contest')">比武</a>】<br/>
      <div v-for="(o, i) in homeRivals" :key="'hr' + i">
        {{ i + 1 }}.<a href="javascript:;" @click="openProfile(o)">{{ o.nick }}</a>({{ o.level }}级)
        <a href="javascript:;" @click="goFight(o)">比武</a><br/>
      </div>
      【<a href="javascript:;" @click="go('logs')">动态</a>】<br/>
      <div v-for="(l, i) in logs" :key="'hl' + i">
        {{ l.msg }}<em>[{{ l.time }}]</em><br/>
      </div>
      【<a href="javascript:;" @click="go('chat')">聊天</a>】<br/>
      <div v-for="(m, i) in chatHistory" :key="'hc' + i">
        [公共]<a href="javascript:;" class="nk">{{ m.nick }}</a>：{{ m.content }}<em>[{{ m.time }}]</em><br/>
      </div>
      <form @submit.prevent="sendChat">
        <input v-model="chatInput" maxlength="100" />
        <input type="submit" value="发布" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 比武 ==================== -->
    <template v-else-if="cur === 'contest'">
      <!-- 复刻 比武导航页.xhtml：备选对手列表，每个人后带"比武"，点名字进属性页 -->
      【比武】<a href="javascript:;" @click="go('records')">记录</a><br/>
      <div v-for="(o, i) in rivals" :key="'cc' + i">
        {{ i + 1 }}.<a href="javascript:;" @click="openProfile(o)">{{ o.nick }}</a>({{ o.level }}级)<a href="javascript:;" @click="goFight(o)">比武</a><br/>
      </div>
      ----------<br/>
      (第<b>1</b>/{{ Math.max(1, Math.ceil(rivals.length / 10)) }}页/共{{ rivals.length }}条记录)<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 比武记录 ==================== -->
    <template v-else-if="cur === 'records'">
      【比武记录】<br/>
      <div v-for="(l, i) in pagedRecords()" :key="'bl' + l.id">{{ recIx(i) }}.<a href="javascript:;" @click="openRecord(l)">{{ l.my_nick }} VS {{ l.opp_nick }}</a>[{{ l.created_at }}]<br/></div>
      <div v-if="!records.length"><em>暂无战斗记录，先去比武攒战绩吧。</em></div>
      ----------<br/>
      <a v-if="recordPage > 1" href="javascript:;" @click="recordPage--">上页</a><a v-if="recordPage < recordTotalPages()" href="javascript:;" @click="recordPage++">下页</a>
      (第<b>{{ recordPage }}</b>/{{ recordTotalPages() }}页/共{{ records.length }}条记录)<br/>
      <form @submit.prevent="goRecordPage">
        第<input type="text" v-model.number="recordGoto" maxlength="10" size="2" />页
        <input type="submit" value="前往" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 比武记录详情（复刻 点击记录后的比武详情页） ==================== -->
    <template v-else-if="cur === 'recorddetail'">
      【比武】<br/>
      消耗10点气力！<br/>
      1.<a href="javascript:;" @click="tip('属性')">{{ rd.my_nick }}</a>(等级:{{ rd.my_level }} 气血:{{ rd.my_cur_hp }}/{{ rd.my_max_hp }})<br/>
      2.<a href="javascript:;" @click="tip('属性')">{{ rd.opp_nick }}</a>(等级:{{ rd.opp_level }} 气血:{{ rd.opp_cur_hp }}/{{ rd.opp_max_hp }})<br/>
      <template v-if="rd.logs.length">
        【直播】<br/>
        经验：+{{ rd.exp }}<br/>
        G币：+{{ rd.coin }}<br/>
        [{{ rd.result }}]{{ rd.result === '失败' ? rd.opp_nick : rd.my_nick }}<br/>
        <div v-for="(l, i) in rd.logs" :key="'rd' + i">{{ i + 1 }}.{{ l.text }}<br/></div>
        <br/>
        (第<b>1</b>/1页/共{{ rd.count }}条记录)<br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('records')">返回记录列表</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 用户详情 ==================== -->
    <template v-else-if="cur === 'userdetail'">
      昵称：{{ userDetail.nick || '玩家' }}<br/>
      性别：{{ userDetail.gender }}<br/>
      等级：{{ userDetail.level }}<br/>
      称号：{{ userDetail.title_name }}<br/>
      【装备】<br/>
      <div v-for="s in userDetail.slots" :key="'us' + s.slot">{{ s.slot }}：{{ s.item_name }}<br/></div>
      &gt;&gt;<a href="javascript:;" @click="fightProfile">我要比武</a><br/>
      -----------<br/>
      <span v-if="userDetail.practicing === 1" class="green">该好友正在修炼({{ userDetail.practice_min }}分钟)</span><br/>
      <template v-if="userDetail.is_friend && userDetail.practicing === 1">
        <a href="javascript:;" @click="practiceAction('steal')">[吸取经验]</a>
        .<a href="javascript:;" @click="practiceAction('harass')">[骚扰]</a>
        .<a href="javascript:;" @click="practiceAction('heal')">[治疗]</a><br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="goBack">返回</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 升级 ==================== -->
    <template v-else-if="cur === 'upgrade'">
      【升级】<br/>
      <span :class="upgradeMsg && upgradeMsg.indexOf('成功') >= 0 ? 'green' : 'red'">{{ upgradeMsg }}</span><br/>
      <a v-if="!(upgradeMsg && upgradeMsg.indexOf('成功') >= 0)" href="javascript:;" @click="levelUp">立即升级</a>
      <span v-else><a href="javascript:;" @click="go('upgrade')">继续升级</a></span><br/>
      <em>当前{{ g.level }}级，经验 {{ g.exp }}/{{ g.next_exp }}</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 创建帮派 ==================== -->
    <template v-else-if="cur === 'gangcreate'">
      【创建帮派】<br/>
      【创建条件】<br/>
      1. 帮主等级需达 10 级 (当前 {{ g.level }} 级)<br/>
      2. 持有 <b>帮派令旗</b> x1（商店元宝购买，价格 200 元宝）<br/>
      &nbsp;&nbsp;当前持有：帮派令旗 x{{ flagCount }}<br/>
      &nbsp;&nbsp;<a href="javascript:;" @click="go('shop')">去商店购买令旗</a><br/>
      -----------<br/>
      <form v-if="g.level >= 10 && flagCount >= 1" @submit.prevent="gangCreate">
        <input type="text" v-model.trim="gangNameInput" maxlength="12" placeholder="帮派名" />
        <input type="submit" value="创建帮派" />
      </form>
      <template v-else>
        <span v-if="g.level < 10" class="red">等级不足！需 10 级，当前 {{ g.level }} 级</span><br/>
        <span v-if="flagCount < 1" class="red">未持有帮派令旗！</span><br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('gang')">返回帮派列表</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 帮派详情 ==================== -->
    <template v-else-if="cur === 'gangview'">
      【帮派详情】<br/>
      帮派名称：{{ gangDetail.name }}<br/>
      帮派等级：{{ gangDetail.level_text }}<br/>
      帮派经验：{{ gangDetail.exp_text }}<br/>
      帮主：<font color="#FF00FF">{{ gangDetail.master }}</font><br/>
      成员人数：{{ gangDetail.members }}/20<br/>
      帮派公告：{{ gangDetail.notice || '无' }}<br/>
      创建时间：{{ gangDetail.created_at }}<br/>
      说明：{{ gangDetail.exp_desc }}<br/>
      -----------<br/>
      <a v-if="gangDetail.my_gang === 0" href="javascript:;" @click="goGangApply(gangDetail.id)">申请加入</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('gang')">返回帮派列表</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 申请加入帮派 ==================== -->
    <template v-else-if="cur === 'gangapply'">
      【申请加入帮派】<br/>
      <form @submit.prevent="submitGangApply">
        申请理由:<input type="text" v-model.trim="gangApplyMsg" maxlength="200" /><br/>
        <input type="submit" value="提交申请" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('gang')">返回帮派列表</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 申请加入帮派·成功 ==================== -->
    <template v-else-if="cur === 'gangapplyok'">
      【申请加入帮派】<br/>
      申请提交成功！请等待帮主审批<br/>
      <a href="javascript:;" @click="go('gang')">返回</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('gang')">返回帮派列表</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 比武战斗 ==================== -->
    <template v-else-if="cur === 'fight'">
      <!-- 复刻 玩家后面的比武.xhtml：只展示比武的双方 + 该场【直播】 -->
      <template v-if="dupeFight">
        <!-- 复刻 比武重复.xhtml：今日已与该玩家比武过 -->
        【比武】<br/>
        <span style="color:red;">今日已与该玩家比武过了，请换一个人比武</span><br/>
        <a href="javascript:;" @click="go('contest')">【去比武列表】</a><br/>
        <a href="javascript:;" @click="go('records')">查看比武记录</a><br/>
        <br/>
      </template>
      <template v-else>
        【比武】<br/>
        消耗10点气力！<br/>
        <span v-if="fightError" class="red">{{ fightError }}</span>
        <template v-else>
          1.<a href="javascript:;" @click="go('profile')">{{ g.nick }}</a>(等级:{{ g.level }} 气血:{{ g.cur_hp }}/{{ g.combat.max_hp }})<br/>
          2.<a v-if="fightTarget" href="javascript:;" @click="openProfile(fightTarget)">{{ fightTarget.nick }}</a>(等级:{{ (fightTarget && fightTarget.level) || '' }} 气血:{{ (fightTarget && fightTarget.cur_hp) || 0 }}/{{ (fightTarget && fightTarget.max_hp) || 0 }})<br/>
          <template v-if="battleLogs.length">
            【直播】<br/>
            经验：+{{ battleExp }}<br/>
            G币：+{{ battleCoin }}<br/>
            [{{ battleRes }}]{{ battleRes === '失败' ? (fightTarget ? fightTarget.nick : '') : g.nick }}<br/>
            <div v-for="(l, i) in battleLogs" :key="'fl' + i">{{ i + 1 }}.{{ l.text }}<br/></div>
            <br/>
            (第<b>1</b>/1页/共{{ battleLogs.length }}条记录)<br/>
          </template>
          -----------<br/>
          <a href="javascript:;" @click="go('home')">返回首页</a>.<a href="javascript:;" @click="againFight">再战</a>.<a href="javascript:;" @click="go('contest')">回比武</a>
        </template>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 修炼 ==================== -->
    <template v-else-if="cur === 'room'">
      【修炼】<a href="javascript:;" @click="loadPractice">刷新</a><br/>
      修炼类型：{{ practice.practice_type || '普通(4小时)' }}<br/>
      修炼经验：{{ g.exp }}/{{ g.next_exp }} <a href="javascript:;" @click="go('upgrade')">升级</a><br/>
      修炼技能：{{ skillExpText }}<br/>
      修炼状态：{{ practice.practicing === 1 ? '修炼中 ' + remainingText + ' ' : '未修炼 ' }}
      <a v-if="practice.practicing !== 1" href="javascript:;" @click="go('practicemode')">选择修炼</a>
      <a v-else href="javascript:;" @click="stopPractice">停止</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('friend')">好友列表(查看修炼状态/吸取/骚扰/治疗)</a><br/>
      <a href="javascript:;" @click="go('palace')">战神宫(20级·高倍经验)</a>.<a href="javascript:;" @click="go('forge')">装备锻造</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 选择修炼模式 ==================== -->
    <template v-else-if="cur === 'practicemode'">
      【修炼】<br/>
      选择修炼模式：<br/>
      <a href="javascript:;" @click="startPractice('normal')">普通修炼(4小时·免费·消耗20体力)</a><br/>
      <a href="javascript:;" @click="startPractice('long8')">加长修炼(8小时·50元宝·消耗20体力)</a><br/>
      <a href="javascript:;" @click="startPractice('long24')">加长修炼(24小时·120元宝·消耗20体力)</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 开始修炼成功（修炼开始修炼.xhtml） ==================== -->
    <template v-else-if="cur === 'startok'">
      【修炼】<br/>
      <span style="color:green;">{{ startMsg }}</span><br/>
      提示：修炼5分钟内取消将不获得任何经验。<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 停止修炼结果（停止修炼.xhtml） ==================== -->
    <template v-else-if="cur === 'stopok'">
      【修炼】<br/>
      修炼结束！<br/>
      经验：+{{ stopExp }}<br/>
      技能点：+{{ stopSkill }}<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 战神宫 ==================== -->
    <template v-else-if="cur === 'palace'">
      【战神宫】<br/>
      <span v-if="g.level >= 20">战神宫已解锁！击败更多对手获取高倍经验。</span>
      <span v-else class="red">战神宫需20级解锁，你当前{{ g.level }}级。</span><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 技能书店 ==================== -->
    <template v-else-if="cur === 'skill'">
      【技能书店】<br/>
      分类：<span v-for="(c, i) in skillCats" :key="c"><a href="javascript:;" :class="{ cur: skillCat === c }" @click="setSkillCat(c)">{{ i === 0 ? '全部' : (c === 'passive' ? '被动技能' : '主动技能') }}</a>{{ i < skillCats.length - 1 ? '|' : '' }}</span><br/>
      <div v-for="(s, i) in paginatedSkills" :key="'sk' + s.id">
        {{ (skillPage - 1) * 10 + i + 1 }}.<b>{{ s.name }}</b>[{{ s.act === 1 ? '主动' : '被动' }}]<br/>
        等级要求：{{ s.level }}级 | 学习费用：{{ s.price }}{{ s.currency === 'yuanbao' ? '元宝' : '点' }} [<a href="javascript:;" @click="openSkill(s)">详情</a>] <a v-if="!s.learned" href="javascript:;" @click="learn(s)">[购买]</a><span v-else class="green">[已学]</span><br/>
      </div>
      <template v-if="!paginatedSkills.length"><em>该分类暂无技能</em><br/></template>
      ----------<br/>
      <a v-if="skillPage > 1" href="javascript:;" @click="skillPage--">上页</a><a v-if="skillPage < skillTotalPages" href="javascript:;" @click="skillPage++">下页</a> (第<b>{{ skillPage }}</b>/{{ skillTotalPages }}页)<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('myskill')">我的技能</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 技能详情 ==================== -->
    <template v-else-if="cur === 'skilldetail'">
      【技能领悟】<br/>
      技能：{{ skillDetail.name }}<br/>
      类型：{{ skillDetail.act === 1 ? '主动技能' : '被动技能' }}<br/>
      等级要求：{{ skillDetail.level }}级<br/>
      武器要求：{{ skillDetail.weapon_req || '无限制' }}<br/>
      伤害系数：{{ (skillDetail.coef / 100).toFixed(2) }}倍<br/>
      命中概率：{{ (skillDetail.hit || 0).toFixed(4) }}%<br/>
      暴击概率：{{ (skillDetail.crit || 0).toFixed(4) }}%<br/>
      暴击倍数：{{ (skillDetail.crit_mul || 0).toFixed(4) }}%<br/>
      闪避概率：{{ (skillDetail.dodge_add || 0).toFixed(4) }}%<br/>
      技能描述：{{ skillDetail.desc || skillDetail.name }}<br/>
      学习价格：{{ skillDetail.price }} ({{ skillDetail.currency === 'yuanbao' ? '元宝' : 'G币' }})<br/>
      该技能无限量供应<br/>
      <div v-if="skillDetail.learned" style="background:#90EE90;padding:5px;margin:5px 0;">已学习！当前等级：{{ skillDetail.lv || 1 }}级<br/>
        熟练度：{{ skillDetail.practice || 0 }}/100<br/>
      </div>
      <template v-if="skillDetail.learned">熟练度达到100可领悟升级<br/></template>
      <a v-else href="javascript:;" @click="learn(skillDetail)">[购买学习]</a><br/>
      <a href="javascript:;" @click="go('contest')">【去比武】</a> <a href="javascript:;" @click="go('room')">【去修炼】</a><br/>
      <a href="javascript:;" @click="go('myskill')">查看我的技能</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('skill')">返回商店</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 我的技能 ==================== -->
    <template v-else-if="cur === 'myskill'">
      【我的技能】<br/>
      <span v-if="mySkillMsg" class="green">{{ mySkillMsg }}<br/></span>
      <span v-else-if="!mySkills.length">您还没有学会任何技能<br/><a href="javascript:;" @click="go('skill')">去商店购买技能</a></span>
      <template v-else>
        <br/>【主动技能】<br/>
        <div v-for="(s, i) in activeMySkills" :key="'aa' + s.id">
          {{ i + 1 }}.{{ s.name }}(Lv.{{ s.level }})(熟练度:{{ s.practice }}/100) <span v-if="s.equip === 1" class="red">[已装备]</span><br/>
          <a href="javascript:;" @click="viewSkill(s)">[查看]</a> <a v-if="s.equip === 1" href="javascript:;" @click="skillUnequip(s)">[卸下]</a><a v-else href="javascript:;" @click="skillEquip(s)">[装备]</a> <br/>
        </div>
        <template v-if="!activeMySkills.length"><em>暂无主动技能</em><br/></template>
        <br/>【被动技能】<br/>
        <div v-for="(s, i) in passiveMySkills" :key="'pp' + s.id">
          {{ i + 1 }}.{{ s.name }}(Lv.{{ s.level }})(熟练度:{{ s.practice }}/100) <span v-if="s.equip === 1" class="red">[已装备]</span><br/>
          <a href="javascript:;" @click="viewSkill(s)">[查看]</a> <a v-if="s.equip === 1" href="javascript:;" @click="skillUnequip(s)">[卸下]</a><a v-else href="javascript:;" @click="skillEquip(s)">[装备]</a> <br/>
        </div>
        <template v-if="!passiveMySkills.length">暂无被动技能<br/></template>
        <br/><span style="color:gray;">提示：装备技能后，比武和修炼可增加熟练度</span><br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('skill')">返回商店</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 道具商店 ==================== -->
    <template v-else-if="cur === 'shop'">
      【道具商店】<br/>
      类目:<a v-for="c in shopCats" :key="c.key" :class="{ cur: shopCat === c.key }"
        href="javascript:;" @click="setShopCat(c.key)">{{ c.label }}</a><br/>
      -----------<br/>
      <div v-for="(s, i) in paginatedShop" :key="'sp' + s.id">
        {{ (shopPage - 1) * shopPageSize + i + 1 }}.[{{ shopParent(s.cat) }}] <a href="javascript:;" @click="openShopBuy(s)">{{ s.name }}</a> {{ s.price }}{{ s.currency === 'yuanbao' ? '元宝' : 'G币' }}<br/>
      </div>
      <div v-if="!filteredShop.length"><em>该分类暂无商品</em></div>
      ----------<br/>
      <a v-if="shopPage > 1" href="javascript:;" @click="shopPage--">上页</a><a v-if="shopPage < shopTotalPages" href="javascript:;" @click="shopPage++">下页</a> (第<b>{{ shopPage }}</b>/{{ shopTotalPages }}页/共{{ filteredShop.length }}条)<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('forge')">【装备锻造】(图纸+材料)</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 购买道具（点击道具名后的详情，复刻 购买点击的道具名字.xhtml） ==================== -->
    <template v-else-if="cur === 'shopbuy'">
      【购买道具】<br/>
      销售价格：{{ buySel.price }}({{ buySel.currency === 'yuanbao' ? '元宝' : 'G币' }})<br/>
      库存数量：{{ buySel.stock }}<br/>
      销售数量：{{ buySel.sold }}<br/>
      【道具参数】<br/>
      装备类型：{{ shopParent(buySel.cat) }}<br/>
      装备名称：{{ buySel.name }}<br/>
      使用等级：{{ buySel.level || 0 }}<br/>
      <template v-if="buySel.cat === 'medicine'">
        增加气血：{{ buySel.recover_hp || 0 }}<br/>
        增加气力：{{ buySel.recover_mp || 0 }}<br/>
      </template>
      <template v-else>
        增加速度：{{ buySel.spd || 0 }}-{{ (buySel.spd || 0) + 1 }}<br/>
        增加攻击：{{ buySel.atk || 0 }}-{{ (buySel.atk || 0) + 1 }}<br/>
        增加防御：{{ buySel.def || 0 }}-{{ (buySel.def || 0) + 1 }}<br/>
      </template>
      <a href="javascript:;" @click="doBuy">确定购买</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 装备锻造 ==================== -->
    <template v-else-if="cur === 'forge'">
      【装备锻造】<br/>
      <div v-for="(f, i) in forge" :key="'fg' + i">
        {{ i + 1 }}.<b>{{ f.name }}</b>({{ shopParent(f.cat) }} {{ f.level }}级)<br/>
        锻造费：{{ f.fee }}(元宝)<br/>
        <div v-for="(m, j) in f.mats" :key="'m' + j">
          <span :class="m.have < m.need ? 'red' : ''">[{{ matSrc(j) }}] {{ m.name }} x{{ m.need }}(持有{{ m.have }})</span><br/>
        </div>
        <span v-if="g.yuanbao < f.fee" class="red">元宝不足</span><br/>
        <span v-if="g.level < f.level" class="red">需{{ f.level }}级</span><br/>
        <a v-if="forgeOk(f)" href="javascript:;" @click="forgeItem(f)">[锻造]</a><span v-else class="gray">[条件不足]</span><br/>
        <br/>
      </div>
      <div v-if="!forge.length"><em>暂无锻造配方</em></div>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 我的属性 ==================== -->
    <template v-else-if="cur === 'profile'">
      【基本资料】<br/>
      昵称：{{ g.nick }}[<a href="javascript:;" @click="go('profileedit')">修改</a>]<br/>
      性别：{{ ['保密','男','女'][g.sex || 0] }}[<a href="javascript:;" @click="go('profileedit')">修改</a>]<br/>
      等级：{{ g.level }}<br/>
      称号：{{ g.title_name }}<br/>
      帮派：{{ gangName }} [<a href="javascript:;" @click="go('gang')">加入帮派</a>]<br/>
      经验：{{ g.exp }}<br/>
      注册时间：{{ regTime }}<br/>
      <br/>【战斗属性】<br/>
      气血：{{ g.cur_hp }}-{{ c.max_hp }}<br/>
      气力：{{ g.cur_mp }}-{{ c.max_mp }}<br/>
      速度：{{ c.speed }}/{{ c.speed + 1 }}<br/>
      攻击：{{ c.atk }}-{{ c.atk + 1 }}<br/>
      防御：{{ c.def }}-{{ c.def + 1 }}<br/>
      命中：{{ c.hit }}%<br/>
      暴击：{{ c.crit }}%<br/>
      闪避：{{ c.dodge }}%<br/>
      <br/>【被动技能加成】<br/>
      <div v-if="mySkills.length">
        <div v-for="s in passiveSkillRows" :key="'ps' + s.id">{{ s.name }}(Lv.{{ s.level }}):{{ passiveDesc(s) }}</div>
      </div>
      <div v-else><em>暂无被动技能，去技能书店学习可获得加成。</em><br/></div>
      <br/>【装备栏】<br/>
      能量：{{ g.energy }} [<a href="javascript:;" @click="go('alloc')">分配</a>]<br/>
      <div v-for="s in g.slots" :key="'slot' + s.slot">
        {{ s.name }}：{{ slotName(s.slot) }} [<a href="javascript:;" @click="goEquipSlot(s.slot)">换装</a>]
        <span v-if="s.item_id">[<a href="javascript:;" @click="unequip(s.slot)">卸下</a>]</span><br/>
      </div>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 修改资料 ==================== -->
    <template v-else-if="cur === 'profileedit'">
      【修改资料】<br/>
      <form @submit.prevent="saveProfile">
        性别:<select v-model.number="editSex">
          <option :value="0">保密</option>
          <option :value="1">男</option>
          <option :value="2">女</option>
        </select><br/>
        昵称:<input type="text" v-model.trim="editName" maxlength="20" :placeholder="g.nick" /><br/>
        <input type="submit" value="确定修改" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('profile')">返回属性</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 修改资料成功 ==================== -->
    <template v-else-if="cur === 'profileeditok'">
      【修改资料】<br/>
      修改成功！<br/>
      -----------<br/>
      <a href="javascript:;" @click="load(); go('profile')">返回属性</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 使用装备 ==================== -->
    <template v-else-if="cur === 'equipslot'">
      【装备】<br/>
      <div v-for="(e, i) in equipCandidates" :key="'eq' + i">
        {{ i + 1 }}.<a href="javascript:;" @click="tip('道具详情')">{{ e.name }}</a> <a href="javascript:;" @click="equipFromSlot(e)">使用</a><br/>
      </div>
      <div v-if="!equipCandidates.length"><em>暂无可用{{ equipSlotName }}装备，去商店或锻造获得。</em><br/></div>
      ----------<br/>
      <em>(第1/1页/共{{ equipCandidates.length }}条记录)</em>
      -----------<br/>
      <a href="javascript:;" @click="go('profile')">返回属性</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 能量分配 ==================== -->
    <template v-else-if="cur === 'alloc'">
      【能量分配】<br/>
      可分配能量：{{ g.energy }}<br/>
      <div>气血：{{ g.alloc.hp }} <a href="javascript:;" @click="alloc('hp')">+1</a></div>
      <div>气力：{{ g.alloc.mp }} <a href="javascript:;" @click="alloc('mp')">+1</a></div>
      <div>速度：{{ g.alloc.spd }} <a href="javascript:;" @click="alloc('spd')">+1</a></div>
      <div>攻击：{{ g.alloc.atk }} <a href="javascript:;" @click="alloc('atk')">+1</a></div>
      <div>防御：{{ g.alloc.def }} <a href="javascript:;" @click="alloc('def')">+1</a></div>
      <em>每升1级获得1点能量</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('profile')">返回属性</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 我的背包/行囊 ==================== -->
    <template v-else-if="cur === 'bag'">
      【我的背包】<a href="javascript:;" @click="loadBag">刷新</a><br/>
      <div v-for="(b, i) in bag" :key="'bg' + b.item_id">
        {{ i + 1 }}.<a href="javascript:;" @click="openItemDetail(b)">{{ b.name }}</a> ({{ b.amount }}) [<a href="javascript:;" @click="useItem(b)">使用</a>.<a href="javascript:;" @click="dropItem(b)">丢弃</a>]
        <em v-if="isEquip(b.cat)">(+攻{{ b.atk }} +防{{ b.def }} +血{{ b.hp }} +气{{ b.mp }} +速{{ b.spd }})</em>
        <a v-if="isEquip(b.cat)" href="javascript:;" @click="equip(b)">[装备]</a><br/>
      </div>
      <div v-if="!bag.length">
        暂无记录！<br/>
        <em>去<a href="javascript:;" @click="go('shop')">商店</a>购买道具吧。</em>
      </div>
      -----------<br/>
      <a href="javascript:;" @click="go('profile')">返回属性</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 我的道具详情 ==================== -->
    <template v-else-if="cur === 'itemdetail'">
      <!-- 复刻 我的道具详情.xhtml -->
      【我的道具】<br/>
      道具名称：{{ (itemDetail && itemDetail.name) || '' }}<br/>
      道具等级：{{ (itemDetail && itemDetail.level) || 0 }}<br/>
      道具数量：{{ (itemDetail && itemDetail.amount) || 0 }}<br/>
      <span v-if="(itemDetail && itemDetail.cat) === 'medicine'">补充气血：{{ (itemDetail && itemDetail.recover_hp) || 0 }} 补充气力：{{ (itemDetail && itemDetail.recover_mp) || 0 }}<br/></span>
      <em v-else-if="itemDetail && isEquip(itemDetail.cat)">(+攻{{ itemDetail.atk }} +防{{ itemDetail.def }} +血{{ itemDetail.hp }} +气{{ itemDetail.mp }} +速{{ itemDetail.spd }})<br/></em>
      道具状态：正常 <a href="javascript:;" @click="useItem(itemDetail)">使用</a>.<a href="javascript:;" @click="dropItem(itemDetail)">丢弃</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 使用道具 ==================== -->
    <template v-else-if="cur === 'baguse'">
      【使用道具】<br/>
      道具：{{ useSel.name }}<br/>
      当前气血：{{ g.cur_hp }}/{{ g.combat.max_hp }}<br/>
      当前气力：{{ g.cur_mp }}/{{ g.combat.max_mp }}<br/>
      数量：{{ useSel.amount }}个<br/>
      <div v-if="useFull" style="color:gray;">气血和气力都已满，无需使用！</div>
      <div v-else-if="useMsg" style="background:#90EE90;padding:5px;margin:5px 0;" v-html="useMsg"></div>
      <hr />
      <form @submit.prevent="doUse('use')">
        使用数量：<input type="number" v-model.number="useCount" min="1" style="width:50px;" />
        <input type="submit" value="使用" />
      </form>
      <a href="javascript:;" @click="doUse('full')">[一键使用至满]</a><br/>
      <hr />
      <a href="javascript:;" @click="go('bag')">返回行囊</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== 购买成功 ==================== -->
    <template v-else-if="cur === 'buyok'">
      【购买道具】<br/>
      <div style="color:green;">购买成功！</div><br/>
      购买数量：{{ buyOk.amount }}<br/>
      购买总价：{{ buyOk.total }}({{ buyOk.currency }})<br/>
      <a href="javascript:;" @click="go('bag')">查看背包</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 排行 ==================== -->
    <template v-else-if="cur === 'ranking'">
      【排行】<br/>
      <div v-for="r in ranking" :key="'rk' + r.rank">
        {{ r.rank }}.<a href="javascript:;" @click="openProfile(r)">{{ r.nick }}</a>({{ r.level }}级)
        <a v-if="r.rank > 1" href="javascript:;" @click="goRobotFight(r)">比武</a><br/>
      </div>
      <a v-if="ranking.length === 10" href="javascript:;" @click="tip('共' + (ranking.length * 10) + '名玩家')">下页</a>
      <em>(第1/1页/共{{ ranking.length }}条记录)</em>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 头衔 ==================== -->
    <template v-else-if="cur === 'title'">
      【头衔】<br/>
      类目:<b>[我的头衔]</b>.<a href="javascript:;" @click="go('titlerank')">头衔榜</a><br/>
      -----------<br/>
      玩家等级：{{ g.level }}<br/>
      当前头衔：<b>{{ g.title_name }}</b>({{ g.title }}档)<br/>
      -----------<br/>
      <div v-for="t in titleList" :key="'tt' + t.id">
        <span :class="colorForTier(t.id)">{{ t.name }}</span> 需{{ t.level }}级 {{ t.price }}元宝
        <span v-if="g.title === t.id" class="green">[当前]</span>
        <span v-else-if="g.level < t.level" class="red">[等级不足]</span>
        <a v-else href="javascript:;" @click="activeTitle(t)">[激活]</a><br/>
      </div>
      说明：激活后头衔生效；激活更高档头衔不会退还低档花费，请按需选择。<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 头衔榜 ==================== -->
    <template v-else-if="cur === 'titlerank'">
      【头衔】<br/>
      类目:<a href="javascript:;" @click="go('title')">我的头衔</a>.<b>[头衔榜]</b><br/>
      -----------<br/>
      <div v-for="(p, i) in titleRank" :key="'tr' + i">
        {{ p.rank }}.[{{ p.title_name }}]<a href="javascript:;" @click="openProfile(p)">{{ p.nick }}</a>({{ p.level }}级)<a href="javascript:;" @click="goFight(p)">比武</a><br/>
      </div>
      <div v-if="!titleRank.length"><em>暂无记录</em></div>
      <em>(第1/1页/共{{ titleRank.length }}条记录)</em>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 帮派列表 ==================== -->
    <template v-else-if="cur === 'gang'">
      【帮派列表】<a href="javascript:;" @click="goGangCreate">创建帮派</a><br/>
      <div v-if="g.gang_id > 0">我的帮派:{{ gangName }} <a href="javascript:;" @click="gangLeave">[退出]</a><br/></div>
      -----------<br/>
      所有帮派:<br/>
      <div v-for="(gp, i) in gangList" :key="'gk' + gp.id">
        {{ i + 1 }}.<a href="javascript:;" @click="goGangView(gp)">{{ gp.name }}</a>[{{ gp.level }}级] 帮主:<font color="#ff0000">{{ gp.master }}</font> 人数:{{ gp.members }}/20
        <a v-if="g.gang_id === 0" href="javascript:;" @click="goGangView(gp)">申请加入</a>
        <span v-else-if="g.gang_id === gp.id" class="green">我的帮派</span><br/>
      </div>
      <div v-if="!gangList.length"><em>暂无帮派</em></div>
      <a href="javascript:;" @click="tip('第2页')">下页</a> <em>(第1/1页)</em>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 聊天大厅 ==================== -->
    <template v-else-if="cur === 'chat'">
      【聊天大厅】<a href="javascript:;" @click="loadChatRoom(chatType)">刷新</a><br/>
      <b v-if="chatType===0">全部</b><a v-else href="javascript:;" @click="switchChat(0)">全部</a>.
      <b v-if="chatType===1">个人</b><a v-else href="javascript:;" @click="switchChat(1)">个人</a>.
      <b v-if="chatType===2">世界</b><a v-else href="javascript:;" @click="switchChat(2)">世界</a><br/>
      <div v-for="(m, i) in chatHistory" :key="'ch' + i">
        [{{ chatLabel(m) }}]<a href="javascript:;" class="nk">{{ m.nick }}</a>：{{ m.content }}<em>[{{ m.time }}]</em><br/>
      </div>
      <div v-if="!chatHistory.length"><em>暂无记录！</em></div>
      <form @submit.prevent="sendChat">
        <input v-model="chatInput" maxlength="100" /><br/>
        <input type="submit" value="发布" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 发布成功 ==================== -->
    <template v-else-if="cur === 'chatok'">
      【发言】<br/>
      发布成功！<br/>
      <a href="javascript:;" @click="go('home')">返回</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 动态 ==================== -->
    <template v-else-if="cur === 'logs'">
      【动态】<a href="javascript:;" @click="loadDynamics">刷新</a><br/>
      <div v-for="(l, i) in logs" :key="'dl' + i">
        {{ l.msg }}<em>[{{ l.time }}]</em><br/>
      </div>
      <div v-if="!logs.length"><em>暂无动态</em></div>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 每日礼包 ==================== -->
    <template v-else-if="cur === 'daily'">
      【每日礼包】<br/>
      今天是：{{ weekName }}<br/>
      -----------<br/>
      【本周签到状态】<br/>
      <div v-for="(d, i) in signDays" :key="'sg' + d.key">{{ d.name }}:{{ d.text }}<br/></div>
      -----------<br/>
      【今日任务】完成至少2个任务才能领取礼包<br/>
      任务完成：{{ task.done }}/5<br/>
      <a href="javascript:;" @click="go('task')">查看任务详情</a><br/>
      -----------<br/>
      <div v-if="task.reward_claim === 1" class="green">今日礼包已领取</div>
      <template v-else>
        <span class="red">还需完成{{ Math.max(0, task.need_done - task.done) }}个任务才能领取</span><br/>
        <a v-if="task.done >= task.need_done" href="javascript:;" @click="reward">【领取每日礼包】</a>
        <a v-else href="javascript:;" @click="go('task')">去完成任务</a>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('充值对接全局元宝')">充值</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 每日任务 ==================== -->
    <template v-else-if="cur === 'task'">
      【每日任务】<br/>
      今日任务进度：<b>{{ task.done }}/5</b>（需至少完成2个）<br/>
      <hr/>
      <div class="taskbox" v-for="(t, i) in task.tasks" :key="'tk' + t.key">
        <b>任务{{ i + 1 }}：{{ t.name }}</b><br/>
        <span :class="taskDone(t) ? 'green' : 'gray'">[{{ taskDone(t) ? '已完成' : '未完成' }}]</span>
        <span v-if="t.cur !== undefined" class="gray">{{ t.label2 }}：{{ t.cur }}/{{ t.need }}</span><br/>
        <a v-if="!taskDone(t)" href="javascript:;" @click="goTask(t.key)">{{ t.goText }}</a>
      </div>
      <hr/>
      <div class="rewardbox">
        <b>每日礼包</b><br/>
        <span v-if="task.done >= task.need_done && !task.reward_claim" class="green">任务已完成，可以领取！</span>
        <span v-else-if="task.reward_claim" class="green">今日礼包已领取</span>
        <span v-else class="red">还需完成{{ task.need_done - task.done }}个任务才能领取</span><br/>
        <span class="gray">已完成{{ task.done }}个任务</span><br/>
        <a v-if="task.done >= task.need_done && !task.reward_claim" href="javascript:;" @click="reward">【领取每日礼包】</a>
        <a v-else href="javascript:;" @click="go('daily')">前往每日礼包</a>
      </div>
      <hr/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="go('daily')">每日礼包</a>.<a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 我的好友 ==================== -->
    <template v-else-if="cur === 'friend'">
      【我的好友】<br/>
      <div v-for="(f, i) in friends" :key="'fr' + i">{{ i + 1 }}.{{ f }} <a v-if="i % 2 === 0" href="javascript:;" @click="tip('修炼状态')">[状态]</a><br/></div>
      <div v-if="!friends.length"><em>暂无记录！去家园添加好友后再来。</em></div>
      ----------<br/>
      <em>(第1/1页/共{{ friends.length }}条记录)</em>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <!-- ==================== 帮助 ==================== -->
    <template v-else-if="cur === 'help'">
      <br/><a href="javascript:;" @click="tip('G币获取方式')">G币获取方式</a>.<a href="javascript:;" @click="tip('元宝赞助对接全局充值')">元宝赞助</a><br/>
      一、基础升级<br/>进入游戏点击开始修炼，挂机即可自动获取经验升级<br/>修炼可选择时长，时长越长经验越多，中断修炼会终止收益<br/>升级获得潜能点，自由分配气血、攻击、防御、速度四大属性<br/>
      二、比武对战<br/>可挑战同级玩家、随机对手进行切磋比武<br/>比武胜利获得银币、阅历，失败少量奖励<br/>合理搭配战斗技能，速度快优先出手，攻击高更容易取胜<br/>
      三、装备打造<br/>收集材料可打造各类武器、防具、饰品<br/>支持装备强化、淬炼提升属性，强化越高战力越强<br/>多余装备可分解换取打造材料<br/>
      四、技能学习<br/>消耗阅历学习武学招式，搭配 3 个出战技能<br/>高阶技能伤害更高，对战胜率大幅提升<br/>可重置技能重新搭配玩法流派<br/>
      五、帮派玩法<br/>达到指定等级即可申请加入帮派<br/>参与帮贡、帮战，领取专属帮派福利与 buff 加成<br/>帮派温泉可免费领取大量经验快速升级<br/>
      六、日常福利<br/>每日登录领取签到奖励<br/>完成日常任务领取银币、材料、体力<br/>体力用于比武、修炼，耗尽可等待恢复或道具补充<br/>
      七、账号须知<br/>请勿借号、代练、使用自动脚本批量操作<br/>违规操作会被系统限制功能、封禁账号，损失自行承担<br/>妥善保管账号密码，保护自身游戏资产<br/>
      八、新手小提示<br/>前期优先点攻击，刷图比武更轻松<br/>日常体力优先做完任务，不浪费资源<br/>稳步发育慢慢养成，轻松玩转精武江湖<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('contest')">比武</a>.<a href="javascript:;" @click="go('room')">修炼</a>.<a href="javascript:;" @click="go('skill')">技能</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="tip('赞助')">赞助</a><br/>
      <a href="javascript:;" @click="go('profile')">属性</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('task')">任务</a>.<a href="javascript:;" @click="go('ranking')">排行</a>.<a href="javascript:;" @click="go('title')">头衔</a><br/>
      <a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('help')">论坛</a><br/>
    </template>

    <div class="jwt-tip" v-if="tip">{{ tip }}</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Jingwt',
  data() {
    return {
      cur: 'home',
      g: { alloc: { hp: 0, mp: 0, spd: 0, atk: 0, def: 0 }, slots: [], sign: {} },
      c: {},
      gangName: '无',
      practice: { practicing: 0, remaining_min: 0, practice_type: '普通(4小时)', practice_skill: 0 },
      startMsg: '', stopExp: 0, stopSkill: 0,
      skills: [],
      skillCat: '全部',
      skillPage: 1,
      skillDetail: {},
      mySkills: [],
      mySkillMsg: '',
      shop: [],
      shopCat: 'all',
      shopPage: 1,
      shopPageSize: 15,
      bag: [],
      itemDetail: null,
      rivals: [],
      homeRivals: [],
      forge: [],
      task: { tasks: [], done: 0, need_done: 2, reward_claim: 0 },
      titleList: [],
      gangList: [],
      gangNameInput: '',
      showGangCreate: false,
      ranking: [],
      titleRank: [],
      logs: [],
      chatHistory: [],
      chatInput: '',
      chatType: 0,
      friends: [],
      battleLogs: [],
      battleResult: [],
      records: [],
      recordPage: 1,
      recordPageSize: 10,
      recordGoto: 1,
      rd: { logs: [] },
      thisArenaWin: false,
      arenaMsg: '',
      battleExp: 0,
      battleCoin: 0,
      battleRes: '',
      battleOppNick: '',
      fightError: '',
      dupeFight: false,
      dupeNick: '',
      fightTarget: null,
      userDetail: { slots: [] },
      prevCur: 'home',
      upgradeMsg: '',
      flagCount: 0,
      gangDetail: {},
      gangApplyId: 0,
      gangApplyMsg: '',
      useSel: null,
      useCount: 1,
      useMsg: '',
      useFull: false,
      buyOk: {},
      buySel: null,
      // 装备换装页
      equipTargetSlot: '',
      equipSlotName: '',
      equipCandidates: [],
      // 修改资料
      editSex: 0,
      editName: '',
      showGangCreate: false,
      tip: '',
    }
  },
  computed: {
    activeSkills() { return this.skills.filter(s => s.act === 1) },
    passiveSkills() { return this.skills.filter(s => s.act === 0) },
    filteredSkills() {
      const f = this.skillCat
      if (f === '全部') return this.skills
      if (f === 'passive') return this.passiveSkills
      if (f === 'active') return this.activeSkills
      return this.skills
    },
    skillCats() { return ['全部', 'passive', 'active'] },
    remainingText() {
      const m = this.practice.remaining_min || 0
      const h = Math.floor(m / 60)
      const mm = m % 60
      return h > 0 ? `${h}小时${mm}分钟` : `${mm}分钟`
    },
    practiceText() { return this.practice.practicing === 1 ? '修炼中' : '未修炼' },
    roomText() { return this.practice.practicing === 1 ? '修炼中' : '未修炼' },
    skillExpText() {
      if (this.practice.practicing === 1) return `${this.practice.practice_skill || 0}/10000`
      return `${this.practice.practice_skill || 0}/10000`
    },
    passiveSkillRows() { return this.mySkills.filter(s => s.act === 0) },
    weekName() {
      return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][new Date().getDay()]
    },
    signDays() {
      const names = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
      const today = new Date().getDay() // 0=周日
      const todayIdx = (today === 0 ? 6 : today - 1) // 0=周一
      return names.map((name, i) => {
        const key = i + 1
        const signed = this.g.sign && this.g.sign[key]
        let text = '未领取'
        if (signed) {
          text = '已领取'
        } else if (i === todayIdx) {
          text = '今日待领取'
        } else if (name === '周六') {
          text = '(30000GB+4元宝+道具)'
        } else if (name === '周日') {
          text = '(50000GB+5元宝+道具)'
        }
        return { name, text }
      })
    },
    regTime() {
      return this.g.created ? this.g.created.substring(0, 19) : '—'
    },
    shopCats() {
      return [
        { key: 'all', label: '[全部]' },
        { key: 'medicine', label: '药品' },
        { key: 'weapon', label: '武器' },
        { key: 'armor', label: '防具' },
        { key: 'accessory', label: '配饰' },
        { key: 'material', label: '材料' },
        { key: 'other', label: '其他' },
      ]
    },
    filteredShop() {
      const s = this.shopCat
      if (s === 'all') return this.shop
      return this.shop.filter(it => this.catGroup(it.cat) === s)
    },
    shopTotalPages() { return Math.max(1, Math.ceil(this.filteredShop.length / this.shopPageSize)) },
    paginatedShop() {
      const start = (this.shopPage - 1) * this.shopPageSize
      return this.filteredShop.slice(start, start + this.shopPageSize)
    },
    skillTotalPages() { return Math.max(1, Math.ceil(this.filteredSkills.length / 10)) },
    paginatedSkills() {
      const start = (this.skillPage - 1) * 10
      return this.filteredSkills.slice(start, start + 10)
    },
    activeMySkills() { return this.mySkills.filter(s => s.act === 1) },
    passiveMySkills() { return this.mySkills.filter(s => s.act === 0) },
  },
  mounted() {
    this.load()
  },
  methods: {
    async load() {
      const r = await api.get('/games/jwt/view')
      if (r.code === 0) {
        this.g = r.data
        this.c = r.data.combat
      }
      this.gangNameOf()
      this.loadPractice()
      this.loadSkills()
      this.loadShop()
      this.loadBag()
      this.loadTasks()
      this.loadArena()
      this.loadDynamics()
      this.loadChatRoom()
      this.loadMySkill()
    },
    go(t) {
      this.cur = t
      this.arenaMsg = ''
      this.fightError = ''
      if (t !== 'myskill') this.mySkillMsg = ''
      if (t === 'contest') this.loadArena()
      if (t === 'home') this.loadArena()
      if (t === 'room') this.loadPractice()
      if (t === 'skill') { this.skillPage = 1; this.loadSkills() }
      if (t === 'myskill') this.loadMySkill()
      if (t === 'shop') this.loadShop()
      if (t === 'bag') this.loadBag()
      if (t === 'forge') this.loadForge()
      if (t === 'task' || t === 'daily') this.loadTasks()
      if (t === 'ranking') this.loadRanking()
      if (t === 'title') this.loadTitles()
      if (t === 'titlerank') this.loadTitleRanking()
      if (t === 'gang') this.loadGangs()
      if (t === 'logs') this.loadDynamics()
      if (t === 'chat') this.loadChatRoom(this.chatType)
      if (t === 'friend') this.loadFriends()
      if (t === 'records') this.loadRecords()
      if (t === 'upgrade') this.upgradeMsg = ''
      window.scrollTo(0, 0)
    },
    tip(m) {
      this.tip = m
      setTimeout(() => { this.tip = '' }, 2500)
    },
    // ---------- 工具 ----------
    isEquip(cat) { return ['weapon', 'helmet', 'armor', 'shoes', 'necklace', 'bracelet', 'ring', 'medal'].includes(cat) },
    catGroup(cat) {
      if (cat === 'weapon') return 'weapon'
      if (['helmet', 'armor', 'shoes'].includes(cat)) return 'armor'
      if (['necklace', 'bracelet', 'ring', 'medal'].includes(cat)) return 'accessory'
      if (cat === 'material') return 'material'
      if (cat === 'medicine') return 'medicine'
      return 'other'
    },
    shopParent(cat) {
      const m = { medicine: '药品', weapon: '武器', helmet: '头盔', armor: '盔甲', shoes: '战鞋',
        necklace: '项链', bracelet: '手镯', ring: '戒指', medal: '勋章', material: '材料' }
      return m[cat] || '其他'
    },
    slotName(slot) {
      const s = (this.g.slots || []).find(x => x.slot === slot)
      return s && s.item_id ? s.item_name : '无'
    },
    bagBtn(b) {
      if (b.cat === 'medicine') return '[使用]'
      if (this.isEquip(b.cat)) return '[装备]'
      return ''
    },
    bagAction(b) {
      if (b.cat === 'medicine') this.useItem(b)
      else if (this.isEquip(b.cat)) this.equip(b)
    },
    matSrc(i) {
      return i < 2 ? '商店' : '比武'
    },
    forgeOk(f) {
      if (this.g.yuanbao < f.fee) return false
      if (this.g.level < f.level) return false
      for (const m of f.mats) if (m.have < m.need) return false
      return true
    },
    passiveDesc(s) {
      const parts = []
      if (s.p_hp) parts.push('气血+' + s.p_hp)
      if (s.p_mp) parts.push('气力+' + s.p_mp)
      if (s.p_atk) parts.push('攻击+' + s.p_atk)
      if (s.p_def) parts.push('防御+' + s.p_def)
      return parts.join(' ')
    },
    colorForTier(id) {
      if (this.g.title === id) return 'green'
      if (this.g.level < this.titleWithId(id).level) return 'gray'
      return 'black'
    },
    titleWithId(id) {
      return this.titleList.find(t => t.id === id) || { level: 999 }
    },
    taskDone(t) { return (t.cur || 0) >= (t.need || 1) },
    // ---------- 能量分配 ----------
    async alloc(field) {
      const r = await api.post('/games/jwt/energy', { field })
      this.tip(r.msg)
      if (r.code === 0) this.load()
    },
    // ---------- 修炼 ----------
    async loadPractice() {
      const r = await api.post('/games/jwt/practice', {})
      if (r.code === 0) this.practice = r.data
    },
    async startPractice(type) {
      const r = await api.post('/games/jwt/practice', { start: true, type })
      if (r.code === 0) {
        this.loadPractice()
        this.startMsg = r.data.msg || ''
        this.cur = 'startok'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async stopPractice() {
      const r = await api.post('/games/jwt/practice', { stop: true })
      if (r.code === 0) {
        this.stopExp = r.data.exp || 0
        this.stopSkill = r.data.skill || 0
        this.cur = 'stopok'
        window.scrollTo(0, 0)
        this.load()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 技能 ----------
    async loadSkills() {
      const r = await api.get('/games/jwt/skills')
      if (r.code === 0) this.skills = r.data
    },
    openSkill(s) {
      this.skillDetail = { ...s, weapon_req: s.weapon_req || '无限制', crit_mul: s.crit_mul || 150 }
      this.go('skilldetail')
    },
    async learn(s) {
      const r = await api.post('/games/jwt/learn', { id: s.id })
      this.tip(r.msg)
      if (r.code === 0) {
        if (this.cur === 'skilldetail') this.skillDetail = { ...this.skillDetail, learned: true }
        this.loadSkills(); this.load()
      }
    },
    async loadMySkill() {
      const r = await api.post('/games/jwt/myskill', {})
      if (r.code === 0) this.mySkills = r.data
    },
    setSkillCat(c) { this.skillCat = c; this.skillPage = 1 },
    setShopCat(k) { this.shopCat = k; this.shopPage = 1 },
    async viewSkill(s) {
      await this.loadSkills()
      const full = this.skills.find(x => x.id === s.id) || s
      this.skillDetail = { ...full, learned: true, lv: s.level, practice: s.practice }
      this.go('skilldetail')
    },
    async skillEquip(s) {
      const r = await api.post('/games/jwt/skill-act', { act: 'equip', id: s.id })
      if (r.code === 0) { this.mySkillMsg = r.msg; await this.loadMySkill() }
      else this.tip(r.msg)
    },
    async skillUnequip(s) {
      const r = await api.post('/games/jwt/skill-act', { act: 'unequip', id: s.id })
      if (r.code === 0) { this.mySkillMsg = r.msg; await this.loadMySkill() }
      else this.tip(r.msg)
    },
    // ---------- 商店 ----------
    async loadShop() {
      const r = await api.get('/games/jwt/shop')
      if (r.code === 0) this.shop = r.data
    },
    async buy(s) {
      const r = await api.post('/games/jwt/buy', { id: s.id, amount: 1 })
      if (r.code === 0) {
        this.buyOk = { name: r.data.name, amount: r.data.amount, total: r.data.total, currency: r.data.currency }
        this.load(); this.loadBag()
        this.cur = 'buyok'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    openShopBuy(s) {
      this.buySel = { ...s, stock: 100000 - (s.id % 50000), sold: (s.id * 13) % 200 }
      this.cur = 'shopbuy'
      window.scrollTo(0, 0)
    },
    doBuy() {
      this.buy(this.buySel)
    },
    // ---------- 背包 ----------
    async loadBag() {
      const r = await api.get('/games/jwt/bag')
      if (r.code === 0) this.bag = r.data
    },
    openItemDetail(b) {
      this.itemDetail = b
      this.go('itemdetail')
      window.scrollTo(0, 0)
    },
    useItem(b) {
      if (!b.recover_hp && !b.recover_mp) {
        this.tip('该道具不能直接使用')
        return
      }
      this.useSel = b
      this.useCount = 1
      this.useMsg = ''
      this.useFull = false
      this.cur = 'baguse'
      window.scrollTo(0, 0)
    },
    async doUse(mode) {
      const payload = mode === 'full' ? { id: this.useSel.item_id, mode: 'full' } : { id: this.useSel.item_id, count: this.useCount || 1 }
      const beforeHp = this.g.cur_hp, beforeMp = this.g.cur_mp
      const r = await api.post('/games/jwt/item-use', payload)
      if (r.code === 0) {
        this.useFull = false
        if (r.data) {
          this.g.cur_hp = r.data.cur_hp; this.g.cur_mp = r.data.cur_mp
          const used = r.data.used || this.useCount || 1
          // 复刻 使用道具.xhtml：绿色成功框 使用XXX xN成功！气血/气力+恢复值，当前：cur/max
          let lines = '使用' + this.useSel.name + ' x' + used + '成功！'
          const dh = r.data.cur_hp - beforeHp
          const dm = r.data.cur_mp - beforeMp
          if (dh > 0) lines += '<br/>气血+' + dh + '，当前：' + r.data.cur_hp + '/' + r.data.max_hp
          if (dm > 0) lines += '<br/>气力+' + dm + '，当前：' + r.data.cur_mp + '/' + r.data.max_mp
          this.useMsg = lines
        }
        if (this.useSel) this.useSel.amount = Math.max(0, this.useSel.amount - (r.data && r.data.used ? r.data.used : this.useCount || 1))
        this.loadBag(); this.loadTasks(); this.load()
      } else {
        this.useFull = r.msg.indexOf('已满') >= 0
        this.useMsg = ''
        if (this.useFull) this.load()
        else this.tip(r.msg)
      }
    },
    async dropItem(b) {
      const r = await api.post('/games/jwt/item-drop', { id: b.item_id })
      this.tip(r.msg)
      if (r.code === 0) { this.loadBag(); this.load() }
    },
    async equip(b) {
      const r = await api.post('/games/jwt/equip', { id: b.item_id })
      this.tip(r.msg)
      if (r.code === 0) { this.loadBag(); this.load() }
    },
    // ---------- 使用装备（按槽位换装） ----------
    slotCat(slot) {
      const m = { weapon: 'weapon', helmet: 'helmet', armor: 'armor', shoes: 'shoes', necklace: 'necklace', bracelet: 'bracelet', ring: 'ring', medal: 'medal' }
      return m[slot] || slot
    },
    async goEquipSlot(slot) {
      const names = { weapon: '武器', helmet: '头盔', armor: '盔甲', shoes: '战鞋', necklace: '项链', bracelet: '手镯', ring: '戒指', medal: '勋章' }
      this.equipTargetSlot = slot
      this.equipSlotName = names[slot] || slot
      await this.loadBag()
      const cat = this.slotCat(slot)
      this.equipCandidates = (this.bag || []).filter(b => b.cat === cat)
      this.cur = 'equipslot'
      window.scrollTo(0, 0)
    },
    async equipFromSlot(e) {
      const r = await api.post('/games/jwt/equip', { id: e.item_id })
      if (r.code === 0) { await this.load(); await this.loadBag(); this.equipCandidates = this.equipCandidates.filter(x => x.item_id !== e.item_id) }
      this.tip(r.msg)
    },
    async unequip(slot) {
      const r = await api.post('/games/jwt/unequip', { slot: slot })
      this.tip(r.msg)
      if (r.code === 0) { await this.load(); await this.loadBag() }
    },
    // ---------- 修改资料 ----------
    goProfileEdit() {
      this.editSex = this.g.sex || 0
      this.editName = ''
      this.cur = 'profileedit'
      window.scrollTo(0, 0)
    },
    async saveProfile() {
      if (!this.editName) this.editName = this.g.nick
      const r = await api.post('/games/jwt/profile-edit', { sex: this.editSex, name: this.editName })
      if (r.code === 0) {
        this.g.nick = this.editName
        this.g.sex = this.editSex
        this.cur = 'profileeditok'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 比武 ----------
    async loadArena() {
      const r = await api.get('/games/jwt/arena')
      if (r.code === 0) {
        this.rivals = r.data
        this.homeRivals = r.data.slice(0, 5)
      }
    },
    goFight(o) {
      this.fightTarget = o
      this.go('fight')
      this.runFight(o)
    },
    goRobotFight(r) {
      this.fightTarget = { uid: r.uid, nick: r.nick, level: r.level }
      this.go('fight')
      this.runFight(this.fightTarget)
    },
    async runFight(o) {
      this.fightError = ''
      this.battleLogs = []
      this.arenaMsg = '正在比武...'
      this.thisArenaWin = false
      this.dupeFight = false
      const r = await api.post('/games/jwt/arena', { uid: o.uid, level: o.level, nick: o.nick })
      if (r.code === 0) {
        this.arenaMsg = r.msg
        this.thisArenaWin = r.data && r.data.winner === 0
        this.battleLogs = r.data.logs || []
        this.battleExp = (r.data && r.data.exp) || 0
        this.battleCoin = (r.data && r.data.coin) || 0
        this.battleRes = (r.data && r.data.result) || ''
        this.battleOppNick = o.nick || ''
        this.battleResult = (r.data.logs || []).slice(-8).reverse()
        this.load(); this.loadBag(); this.loadTasks(); this.loadDynamics(); this.loadArena()
      } else {
        this.fightError = r.msg
        if (r.code === 450) {
          this.dupeFight = true
          this.dupeNick = (o && o.nick) || ''
        }
      }
    },
    againFight() {
      if (this.fightTarget) { this.battleLogs = []; this.arenaMsg = '正在比武...'; this.runFight(this.fightTarget) }
    },
    loadRecords() {
      this.recordPage = 1
      api.get('/games/jwt/records').then(r => {
        if (r.code === 0) this.records = r.data.list || []
      })
    },
    recordTotalPages() { return Math.max(1, Math.ceil(this.records.length / this.recordPageSize)) },
    pagedRecords() {
      const s = (this.recordPage - 1) * this.recordPageSize
      return this.records.slice(s, s + this.recordPageSize)
    },
    recIx(i) { return (this.recordPage - 1) * this.recordPageSize + i + 1 },
    goRecordPage() {
      const n = Number(this.recordGoto) || 1
      const max = this.recordTotalPages()
      this.recordPage = Math.min(Math.max(1, n), max)
    },
    async openRecord(l) {
      const r = await api.get('/games/jwt/records/' + l.id)
      if (r.code === 0) {
        this.rd = { ...r.data.detail, logs: r.data.detail.logs || [] }
        this.cur = 'recorddetail'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 锻造 ----------
    async loadForge() {
      const r = await api.get('/games/jwt/forge')
      if (r.code === 0) this.forge = r.data
    },
    async forgeItem(f) {
      const r = await api.post('/games/jwt/forge', { id: f.id })
      this.tip(r.msg)
      if (r.code === 0) { this.loadForge(); this.load(); this.loadBag() }
    },
    // ---------- 任务 ----------
    async loadTasks() {
      const r = await api.get('/games/jwt/tasks')
      if (r.code === 0) {
        const goMap = { train: '去修炼', arena: '去比武', chat: '去聊天', pill: '去使用气力丸', skill: '去学习技能' }
        const label2Map = { train: '修炼次数', arena: '比武次数', chat: '发言', pill: '使用次数', skill: '次数' }
        this.task = r.data
        this.task.tasks = (this.task.tasks || []).map(t => ({
          ...t, goText: goMap[t.key] || '去完成', label2: label2Map[t.key] || '次数'
        }))
      }
    },
    goTask(key) {
      const map = { train: 'practicemode', arena: 'contest', chat: 'chat', pill: 'bag', skill: 'skill' }
      this.go(map[key] || 'home')
    },
    async reward() {
      const r = await api.post('/games/jwt/reward')
      this.tip(r.msg)
      if (r.code === 0) { this.load(); this.loadTasks() }
    },
    // ---------- 头衔 ----------
    async loadTitles() {
      const r = await api.post('/games/jwt/titles', {})
      if (r.code === 0) {
        this.titleList = r.data.titles
        if (r.data.hasOwnProperty('level')) this.g.level = r.data.level
      }
    },
    async activeTitle(t) {
      const r = await api.post('/games/jwt/titles', { id: t.id })
      this.tip(r.msg)
      if (r.code === 0) { this.load(); this.loadTitles() }
    },
    // ---------- 帮派 ----------
    async loadGangs() {
      const r = await api.get('/games/jwt/gangs')
      if (r.code === 0) {
        this.gangList = r.data.list
        this.gangNameOf()
      }
    },
    async gangNameOf() {
      const gp = this.gangList.find(g => g.id === this.g.gang_id)
      this.gangName = gp ? gp.name : '无'
    },
    async gangCreate() {
      const r = await api.post('/games/jwt/gang/create', { name: this.gangNameInput })
      this.tip(r.msg)
      if (r.code === 0) { this.gangNameInput = ''; this.go('gang'); this.loadGangs(); this.load() }
    },
    async gangJoin(gp) {
      const r = await api.post('/games/jwt/gang/join', { id: gp.id })
      this.tip(r.msg)
      if (r.code === 0) { this.loadGangs(); this.load() }
    },
    async gangLeave() {
      const r = await api.post('/games/jwt/gang/leave')
      this.tip(r.msg)
      if (r.code === 0) { this.loadGangs(); this.load() }
    },
    // ---------- 升级 ----------
    async levelUp() {
      const r = await api.post('/games/jwt/levelup')
      this.upgradeMsg = r.msg
      if (r.code === 0) { this.load() }
    },
    // ---------- 帮派：创建/详情/申请 ----------
    countFlags(bag) {
      let n = 0
      if (Array.isArray(bag)) {
        for (const b of bag) if (b.name === '帮派令旗') n += b.count || 1
      }
      return n
    },
    async goGangCreate() {
      await this.loadBag()
      this.flagCount = this.countFlags(this.bag)
      this.upgradeMsg = ''
      this.cur = 'gangcreate'
      window.scrollTo(0, 0)
    },
    async goGangView(gp) {
      const r = await api.post('/games/jwt/gangview', { id: gp.id })
      if (r.code === 0) {
        this.gangDetail = { ...r.data }
        this.cur = 'gangview'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    goGangApply(id) {
      this.gangApplyId = id
      this.gangApplyMsg = ''
      this.cur = 'gangapply'
      window.scrollTo(0, 0)
    },
    async submitGangApply() {
      const r = await api.post('/games/jwt/gangapply', { id: this.gangApplyId, msg: this.gangApplyMsg })
      if (r.code === 0) {
        this.cur = 'gangapplyok'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 排行 ----------
    async loadRanking() {
      const r = await api.get('/games/jwt/ranking')
      if (r.code === 0) this.ranking = r.data
    },
    // ---------- 动态 ----------
    async loadDynamics() {
      const r = await api.get('/games/jwt/logs')
      if (r.code === 0) this.logs = r.data
    },
    // ---------- 聊天 ----------
    // type 缺省/-1 = 全部(首页 feed 合并)；0公共 1个人 2世界
    async loadChatRoom(type) {
      const q = type >= 0 ? '?type=' + type : ''
      const r = await api.get('/games/jwt/chat' + q)
      if (r.code === 0) this.chatHistory = r.data
    },
    switchChat(t) {
      this.chatType = t
      this.loadChatRoom(t)
    },
    chatLabel(m) {
      if (m.type === 1) return '个人'
      if (m.type === 2) return '世界'
      return '公共'
    },
    async sendChat() {
      const content = this.chatInput.trim()
      if (!content) { this.tip('请输入聊天内容'); return }
      const r = await api.post('/games/jwt/chat', { content, type: this.chatType })
      this.chatInput = ''
      if (r.code === 0) {
        this.loadChatRoom(this.chatType)
        this.loadTasks()
        this.loadChatRoom(-1)
        this.cur = 'chatok'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 头衔榜 ----------
    async loadTitleRanking() {
      const r = await api.get('/games/jwt/title-ranking')
      if (r.code === 0) this.titleRank = r.data
    },
    // ---------- 好友 ----------
    async loadFriends() {
      const r = await api.get('/games/jwt/friends')
      if (r.code === 0) this.friends = r.data
    },
    // ---------- 用户详情（点开他人） ----------
    async openProfile(o) {
      this.prevCur = this.cur || 'home'
      const r = await api.post('/games/jwt/profile', { uid: o.uid })
      if (r.code === 0) {
        this.userDetail = { slots: [], ...r.data }
      } else {
        this.userDetail = { slots: [] }
        this.tip(r.msg)
      }
      this.cur = 'userdetail'
      window.scrollTo(0, 0)
    },
    goBack() {
      this.go(this.prevCur === 'userdetail' ? 'home' : this.prevCur)
    },
    fightProfile() {
      if (this.userDetail.uid) {
        this.goFight({ uid: this.userDetail.uid, nick: this.userDetail.nick, level: this.userDetail.level })
      }
    },
    practiceAction(act) {
      const map = { steal: '吸取经验', harass: '骚扰', heal: '治疗' }
      this.tip('已对好友发起[' + (map[act] || act) + ']')
    },
  },
}
</script>

<style scoped>
.jwt-wap { font-family: "Microsoft YaHei", "微软雅黑", "SimHei", "黑体", sans-serif; font-size: 15px; margin: 5px; line-height: 1.6; color: #000; word-break: break-all; min-height: 80vh; }
a { color: #0060CD; text-decoration: none; }
em { color: #9B9B9B; font-size: 12px; font-style: normal; }
.red { color: #ff0000; }
.green { color: #008000; }
.gray { color: #9B9B9B; }
.black { color: #000; }
.cur { color: #f60; font-weight: bold; }
.nk { color: #c00; }

.logo { text-align: left; margin: 4px 0; }

/* 每日任务分块 */
.taskbox { padding: 5px; margin: 3px 0; }
.rewardbox { background: #FFFACD; padding: 10px; margin: 10px 0; border: 1px solid #DAA520; }
.footnav { border-top: 1px solid #d0d0d0; background: #f7f8fa; padding: 6px; margin-top: 8px; line-height: 1.9; }

/* toast */
.jwt-tip { position: fixed; left: 50%; top: 20%; transform: translateX(-50%); background: rgba(0, 0, 0, 0.75); color: #fff; padding: 8px 16px; border-radius: 4px; z-index: 200; }
</style>