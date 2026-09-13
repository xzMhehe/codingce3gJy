<template>
  <div class="xy-wap">
    <!-- ==================== 启动中 ==================== -->
    <template v-if="cur === 'boot'">
      加载中……<br/>
    </template>

    <!-- ==================== 选择服务器 ==================== -->
    <template v-else-if="cur === 'server'">
      【选择服务器】<br/>
      欢迎来到幻想西游，请选择要进入的服务器：<br/>
      -----------<br/>
      <div v-for="(s, i) in servers" :key="'sv' + i">
        {{ i + 1 }}.<a href="javascript:;" @click="pickServer(s)">{{ s.id }}·{{ s.name }}</a><template v-if="s.rec"> <span class="red">(推荐)</span></template><br/>
      </div>
      -----------<br/>
      <em>上次进入：{{ lastServer || '无' }}</em><br/>
    </template>

    <!-- ==================== 服务器维护中 ==================== -->
    <template v-else-if="cur === 'maint'">
      【服务器维护中】<br/>
      -----------<br/>
      <span class="red">{{ maintMsg }}</span><br/>
      -----------<br/>
      给你带来的不便敬请谅解！<br/>
      <a href="javascript:;" @click="retryBoot">[重试连接]</a><br/>
    </template>

    <!-- ==================== 建角（分步流程，复刻原版 xy001~xy010） ==================== -->
    <template v-else-if="cur === 'create'">
      <!-- 第1步：选择性别 -->
      <template v-if="cf.step === 'sex'">
        <span class="red">温馨提示：男性玩家无法拜入月宫，女性玩家无法拜入普陀山（性别一旦选择无法更换）</span><br/>
        -----------<br/>
        请你先选择个性别吧<br/>
        <a href="javascript:;" @click="pickSex(1)">男性</a>|<a href="javascript:;" @click="pickSex(2)">女性</a><br/>
      </template>
      <!-- 第2~4步：开场故事 -->
      <template v-else-if="cf.step === 'story'">
        <img :key="cf.story" :src="stories[cf.story].pic" width="200" alt="" onerror="this.style.display='none'"/><br/>
        {{ stories[cf.story].text }}<br/>
        <a href="javascript:;" @click="storyNext()">继续</a><br/>
      </template>
      <!-- 第5步：选择门派 -->
      <template v-else-if="cf.step === 'sect'">
        【选择门派】<br/>
        <div v-for="s in sects" :key="'sc' + s.id">
          <img :key="s.pic" :src="s.pic" width="200" alt="" onerror="this.style.display='none'"/><br/>
          {{ s.intro }}<template v-if="s.limit"><span class="red">({{ s.limit }})</span></template><br/>
          （{{ s.bonus }}）<br/>
          <a href="javascript:;" @click="pickSect(s)">选择{{ s.name }}</a><br/>
        </div>
      </template>
      <!-- 第6步：门派介绍 -->
      <template v-else-if="cf.step === 'sectIntro'">
        【{{ cf.sectName }}】<br/>
        <img :key="cf.sectPic" :src="cf.sectPic" width="200" alt="" onerror="this.style.display='none'"/><br/>
        {{ cf.sectLong }}<br/>
        （{{ cf.sectBonus }}）<br/>
        <a href="javascript:;" @click="cf.step = 'name'">继续</a><br/>
      </template>
      <!-- 第7步：起名字 -->
      <template v-else-if="cf.step === 'name'">
        【起名字】<br/>
        少侠请留名，给你起个响亮的名字吧：<br/>
        角色名：<input v-model="cf.name" maxlength="12" /><br/>
        <form @submit.prevent="doCreate">
          <input type="submit" value="踏上西游路" />
        </form>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="exitToServer">返☆回☆游☆戏☆首☆页</a><br/>
      西游报时({{ xyNow }})<br/>
    </template>

    <!-- ==================== 首页 ==================== -->
    <template v-else-if="cur === 'home'">
      <!-- 消息区（复刻原版 xy002.php：[私聊]/[系统] 动态，展示一次即清） -->
      <template v-if="homeMsgs.length">
        <div v-for="(m, i) in homeMsgs" :key="'hm' + i">
          <template v-if="m.kind === 'pv'">[私聊]<a href="javascript:;" @click="openPm(m.from_id)">{{ m.from_name }}</a>对你说：{{ m.content }}<br/></template>
          <template v-else>[系统]<span class="black">{{ m.content }}</span><br/></template>
        </div>
      </template>
      <!-- 定时活动公告（复刻原版 msgg03.php 静态红字公告） -->
      <template v-if="notices.length">
        <div v-for="(n, i) in notices" :key="'nt' + i">
          [系统]<span class="red">{{ n }}</span><br/>
        </div>
      </template>
      <!-- 组队邀请（复刻原版 yq1.php：邀请直接显示在首页） -->
      <template v-if="homeInvites.length">
        <div v-for="iv in homeInvites" :key="'hi' + iv.id">
          [邀请]<a href="javascript:;" @click="viewPlayer(iv.from_id)">{{ iv.from_name }}</a>邀请你加入队伍
          <a href="javascript:;" @click="teamAgree(iv)">[同意]</a>
          <a href="javascript:;" @click="teamRefuse(iv)">[拒绝]</a><br/>
        </div>
      </template>
      <!-- 国家邀请（复刻原版 yq2.php） -->
      <template v-if="homeGangInvites.length">
        <div v-for="iv in homeGangInvites" :key="'gi' + iv.id">
          国家邀请信息：<br/>
          <a href="javascript:;" @click="viewPlayer(iv.from_id)">{{ iv.from_name }}</a>向你发起了国家邀请<br/>
          <a href="javascript:;" @click="gangAgree(iv)">接受</a>|<a href="javascript:;" @click="gangRefuse(iv)">拒绝</a><br/>
        </div>
      </template>
      <!-- 住宅参观邀请（复刻原版 yq3.php） -->
      <template v-if="homeHouseInvites.length">
        <div v-for="iv in homeHouseInvites" :key="'hv' + iv.id">
          住宅邀请信息：<br/>
          <a href="javascript:;" @click="viewPlayer(iv.from_id)">{{ iv.from_name }}</a>向你发起了住宅参观邀请<br/>
          <a href="javascript:;" @click="houseAgree(iv)">接受</a>|<a href="javascript:;" @click="houseRefuse(iv)">拒绝</a><br/>
        </div>
      </template>
      <!-- 结婚邀请（复刻原版 yq4.php） -->
      <template v-if="homeMarry">
        结婚邀请信息：<br/>
        <a href="javascript:;" @click="viewPlayer(homeMarry.from_id)">{{ homeMarry.from_name }}</a>向你发起了结婚邀请<br/>
        <a href="javascript:;" @click="marryAgree()">接受</a>|<a href="javascript:;" @click="marryRefuse()">拒绝</a><br/>
      </template>
      【幻想西游<template v-if="serverName">·{{ serverName }}</template>】<br/>
      <a href="javascript:;" @click="go('attrs')">{{ g.name }}</a>({{ g.sect_name }}·{{ g.level }}级
      <template v-if="vipLv > 0"><img :key="vipLv" :src="'/static/hxxy/vip/vip' + vipLv + '.png'" @error="$event.target.style.display = 'none'" alt="VIP" style="vertical-align:middle" /></template>
      <template v-else-if="g.vip > 0">·VIP祝福{{ g.vip }}分钟</template>)<br/>
      气血：<span class="red">{{ g.hp }}-{{ g.max_hp }}</span>
      法力：<span :class="g.mp < g.max_mp ? 'cur' : 'black'">{{ g.mp }}-{{ g.max_mp }}</span><br/>
      经验：{{ g.exp }}/{{ g.exp_need }}<br/>
      银两：<span class="cur">{{ g.money }}</span> 存款：{{ g.bank }} 金豆：<span class="red">{{ g.beans }}</span><br/>
      宠物：<template v-if="g.fighting_pet">{{ g.fighting_pet.name }}({{ g.fighting_pet.level }}级·参战中)</template><template v-else>无参战</template><br/>
      修炼经验：<span class="red">{{ g.xiulian_exp }}</span>（<a href="javascript:;" @click="go('cultivate')">修炼</a>）<br/>
      -----------<br/>
      <!-- 地图节点页内容（复刻原版 xy002.php 内嵌 map/*.php：地名+刷新/NPC/图片/出口/查看地图/描述/怪物行） -->
      <span class="black">{{ node.name }}</span><a href="javascript:;" @click="loadState()">刷新</a><br/>
      <div v-for="n in mapNpcs" :key="'hmn' + n.id">
        <a href="javascript:;" @click="viewNpc(n.id)">{{ n.name }}</a><br/>
      </div>
      <div v-if="mapImgSrc" class="mapimg"><img v-show="mapImgOk" :src="mapImgSrc" @load="mapImgOk = true" @error="mapImgErr = true; mapImgOk = false" alt="地图" /><span v-if="mapImgErr" class="gray">（本区域暂无地图图片）</span></div>
      <template v-if="enemies.length">
        <div><span v-for="(e, gi) in enemies" :key="'heg' + gi"><span v-for="k in 3" :key="'he' + gi + '-' + k"><template v-if="k > 1">,</template><a href="javascript:;" @click="startBattle(e.npc_id)">{{ e.name }}</a></span></span></div>
      </template>
      <span class="black">请选择出口</span><br/>
      <span v-for="d in mapExits" :key="'hex' + d.dir">
        <template v-if="d.walk"><span class="black">{{ d.label }}:</span><a href="javascript:;" @click="move(d.dir)">{{ d.walk.name }}</a><br/></template>
        <template v-if="d.jump && (!d.walk || d.jump.dir !== d.walk.dir)"><span class="black">{{ d.label }}:</span><a href="javascript:;" @click="move(d.dir, true)">{{ d.jump.name }}</a><br/></template>
      </span>
      <a href="javascript:;" @click="go('mapview')">查看地图</a><br/>
      <template v-if="node.desc"><span class="black">{{ node.desc }}</span><br/></template>
      -----------<br/>
      <!-- 附近玩家（复刻原版 fjwj.php：你看到：名字【国家】（职务），前置逗号分隔，VIP图独占一行，默认3个+更多....） -->
      <template v-if="nearby.length">
        <span class="black">你看到：</span><span v-for="(n, i) in nearbyShow" :key="'nb' + n.player_id"><template v-if="i > 0"><span class="black">,</span></template><template v-if="n.vip_lv > 0"><img :key="'vip' + n.player_id + '-' + n.vip_lv" :src="'/static/hxxy/vip/vip' + n.vip_lv + '.png'" @error="$event.target.style.display = 'none'" alt="VIP" style="vertical-align:middle" /><br/></template><a href="javascript:;" @click="viewPlayer(n.player_id)">{{ n.name }}<template v-if="n.gang_name">【{{ n.gang_name }}】（{{ n.gang_role }}）</template></a></span><template v-if="!nearbyExpanded && nearby.length > 3"><span class="black">,</span><a href="javascript:;" @click="nearbyExpanded = true">更多....</a></template><br/>
      </template>
      <a href="javascript:;" @click="go('map')">【西游世界】</a>
      <a href="javascript:;" @click="go('bosses')">【世界BOSS】</a>
      <a href="javascript:;" @click="go('dungeons')">【副本】</a><br/>
      <a href="javascript:;" :class="!homeFlags.today_signed ? 'red' : ''" @click="doSignin">【每日签到】</a>
      <a href="javascript:;" :class="homeFlags.quest_ready > 0 ? 'red' : ''" @click="go('quests')">【任务】</a>
      <a href="javascript:;" :class="homeFlags.act_ready ? 'red' : ''" @click="go('activities')">【活动】</a>
      <a href="javascript:;" @click="go('vip')">【充值】</a><br/>
      ----------------------<br/>
      <span class="black">火热玩法</span><br/>
      <a href="javascript:;" @click="go('tower')">挑战</a>◎<a href="javascript:;" @click="go('arena')">擂台</a>◎<a href="javascript:;" @click="go('fun')">娱乐</a>◎<a href="javascript:;" @click="go('gz')">国战</a><br/>
      ----------------------<br/>
      <span class="black">攻略指引</span><br/>
      <a href="javascript:;" @click="go('auction')">拍卖</a>◎<a href="javascript:;" @click="go('guide')">攻略</a>◎<a href="javascript:;" @click="go('guide')">指引</a>◎<a href="javascript:;" @click="go('teyun')">腾云</a><br/>
      ----------------------<br/>
      <span class="black">基础功能</span><br/>
      <a href="javascript:;" @click="go('attrs')">状态</a>◎<a href="javascript:;" @click="go('bag')">物品</a>◎<a href="javascript:;" @click="go('friends')">好友</a>◎<a href="javascript:;" @click="go('quests')">任务</a><br/>
      <a href="javascript:;" @click="go('gang')">国家</a>◎<a href="javascript:;" @click="go('chat')">聊天</a>◎<a href="javascript:;" @click="go('pets')">宠物</a>◎<a href="javascript:;" @click="go('shop')">商城</a><br/>
      <a href="javascript:;" @click="go('team')">队伍</a>◎<a href="javascript:;" @click="go('house')">住宅</a>◎<a href="javascript:;" @click="go('stalls')">挂售</a>◎<a href="javascript:;" @click="go('rank')">排行</a><br/>
      <a href="javascript:;" @click="go('vip')">兑奖</a>◎<a href="javascript:;" @click="go('vip')">特权</a>◎<a href="javascript:;" @click="go('signin')">福利</a>◎<a href="javascript:;" @click="go('sys')">系统</a><br/>
      ----------------------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('attrs')">状态</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('skills')">技能</a>.<a href="javascript:;" @click="go('pets')">宠物</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
      <a href="javascript:;" @click="go('bank')">银行</a>.<a href="javascript:;" @click="go('quests')">任务</a>.<a href="javascript:;" @click="go('dungeons')">副本</a>.<a href="javascript:;" @click="go('bosses')">BOSS</a>.<a href="javascript:;" @click="go('cultivate')">修炼</a>.<a href="javascript:;" @click="go('titles')">头衔</a>.<a href="javascript:;" @click="go('signin')">签到</a><br/>
      <a href="javascript:;" @click="go('rank')">排行</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('friends')">好友</a>.<a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('marriage')">结婚</a>.<a href="javascript:;" @click="go('house')">住宅</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a><br/>
      <a href="javascript:;" @click="go('wallet')">流水</a>.<a href="javascript:;" @click="go('blogs')">战报</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== 西游世界（地图，复刻 map/*.php：地名+刷新/NPC/怪物行/出口/查看地图/描述） ==================== -->
    <template v-else-if="cur === 'map'">
      <span class="black">{{ node.name }}</span><a href="javascript:;" @click="loadState()">刷新</a><br/>
      <div v-for="n in mapNpcs" :key="'mn' + n.id">
        <a href="javascript:;" @click="viewNpc(n.id)">{{ n.name }}</a><br/>
      </div>
      <div v-if="mapImgSrc" class="mapimg"><img v-show="mapImgOk" :src="mapImgSrc" @load="mapImgOk = true" @error="mapImgErr = true; mapImgOk = false" alt="地图" /><span v-if="mapImgErr" class="gray">（本区域暂无地图图片）</span></div>
      <template v-if="enemies.length">
        <div><span v-for="(e, gi) in enemies" :key="'eg' + gi"><span v-for="k in 3" :key="'e' + gi + '-' + k"><template v-if="k > 1">,</template><a href="javascript:;" @click="startBattle(e.npc_id)">{{ e.name }}</a></span></span></div>
      </template>
      <span class="black">请选择出口</span><br/>
      <span v-for="d in mapExits" :key="'ex' + d.dir">
        <template v-if="d.walk"><span class="black">{{ d.label }}:</span><a href="javascript:;" @click="move(d.dir)">{{ d.walk.name }}</a><br/></template>
        <template v-if="d.jump && (!d.walk || d.jump.dir !== d.walk.dir)"><span class="black">{{ d.label }}:</span><a href="javascript:;" @click="move(d.dir, true)">{{ d.jump.name }}</a><br/></template>
      </span>
      <template v-if="!mapExits.length">四面都是高墙，没有出路……<br/></template>
      <a href="javascript:;" @click="go('mapview')">查看地图</a><br/>
      <template v-if="node.desc"><span class="black">{{ node.desc }}</span><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('attrs')">状态</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('skills')">技能</a>.<a href="javascript:;" @click="go('pets')">宠物</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
      <a href="javascript:;" @click="go('bank')">银行</a>.<a href="javascript:;" @click="go('quests')">任务</a>.<a href="javascript:;" @click="go('dungeons')">副本</a>.<a href="javascript:;" @click="go('bosses')">BOSS</a>.<a href="javascript:;" @click="go('cultivate')">修炼</a>.<a href="javascript:;" @click="go('titles')">头衔</a>.<a href="javascript:;" @click="go('signin')">签到</a><br/>
      <a href="javascript:;" @click="go('rank')">排行</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('friends')">好友</a>.<a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('marriage')">结婚</a>.<a href="javascript:;" @click="go('house')">住宅</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a><br/>
      <a href="javascript:;" @click="go('wallet')">流水</a>.<a href="javascript:;" @click="go('blogs')">战报</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== 查看地图（复刻 xy008 + MapViewer 网格） ==================== -->
    <template v-else-if="cur === 'mapview'">
      <a href="javascript:;" @click="mapZoom(2)">放大地图</a> <a href="javascript:;" @click="mapZoom(-2)">缩小地图</a> <a href="javascript:;" @click="mapZoom(0)">重置地图</a><br/>
      <div style="overflow:auto">
        <table class="mapgrid">
          <tr v-for="(row, y) in mapWin.rows" :key="'gr' + y">
            <td v-for="(c, x) in row" :key="'gc' + y + '_' + x">
              <template v-if="c && c.dtxy"><div class="mgnode"><span :class="c.dtxy === mapGrid.cur ? 'mgcur' : (c.is_jump ? 'mgjump' : 'mgnorm')">{{ c.mz }}</span></div></template>
              <template v-else-if="c === '—' || c === '|'"><span class="mgwall">{{ c }}</span></template>
              <template v-else>&nbsp;</template>
            </td>
          </tr>
        </table>
      </div>
      <br/>
      <a href="javascript:;" @click="go('map')">返回</a>.<a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== NPC交互 ==================== -->
    <template v-else-if="cur === 'npcview' && npcCur">
      <div v-if="npcCur.img" class="npcimg"><img :key="npcCur.img" :src="'/static/hxxy/npc/' + npcCur.img" @error="$event.target.style.display = 'none'" alt="NPC" /></div>
      <span class="red">{{ npcCur.name }}</span><template v-if="npcCur.level > 0">({{ npcCur.level }}级)</template><br/>
      <span class="black">{{ npcCur.name }}：{{ npcCur.dialogue }}</span><br/>
      <template v-if="npcCur.teles && npcCur.teles.length">
        -----------<br/>
        <div v-for="(t, i) in npcCur.teles" :key="'tl' + i">
          <a href="javascript:;" @click="npcTele(t)">{{ t.name }}</a><br/>
        </div>
      </template>
      <template v-if="npcCur.shop">
        -----------<br/>
        <a v-if="npcCur.shop === 'rest'" href="javascript:;" @click="npcRest">住店休息({{ g.level * 10 }}银两回满)</a>
        <a v-else-if="npcCur.shop === 'bank'" href="javascript:;" @click="go('bank')">存取银两</a>
        <a v-else-if="npcCur.shop === 'warehouse'" href="javascript:;" @click="openWarehouse">寄存取物</a>
        <a v-else href="javascript:;" @click="npcShop(npcCur.shop)">买东西</a><br/>
      </template>
      <template v-if="npcCur.npc_id > 0">
        -----------<br/>
        <a href="javascript:;" @click="startBattle(npcCur.npc_id)">攻击{{ npcCur.name }}</a><br/>
      </template>
      <template v-if="npcCur.quests && npcCur.quests.length">
        -----------<br/>
        <span class="black">发布的任务：</span><br/>
        <div v-for="q in npcCur.quests" :key="'nq' + q.quest_id">
          <span class="red">{{ q.name }}</span>({{ qstCatName(q.category) }})<br/>
          <span class="gray">{{ q.desc }}</span><br/>
          <a href="javascript:;" @click="questAccept(q)">[接取任务]</a><br/>
        </div>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="backToMap">返回西游世界</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 战斗 ==================== -->
    <template v-else-if="cur === 'battle' && bt">
      【战斗】第{{ bt.round }}回合<br/>
      <template v-if="bt.in_battle">
        你：<span class="green">{{ bt.self.name }}</span>({{ bt.self.level }}级)
        气血：<span :class="bt.self.hp < bt.self.max_hp * 0.3 ? 'red' : 'black'">{{ bt.self.hp }}/{{ bt.self.max_hp }}</span>
        法力：{{ bt.self.mp }}/{{ bt.self.max_mp }}<br/>
        <template v-if="bt.self.pet">宠物：<span class="green">{{ bt.self.pet.name }}</span>({{ bt.self.pet.level }}级) 气血：{{ bt.self.pet.hp }}/{{ bt.self.pet.max_hp }}<br/></template>
        对手：<span class="red">{{ bt.enemy.name }}</span>({{ bt.enemy.level }}级<template v-if="bt.enemy.difficulty">·{{ bt.enemy.difficulty }}</template>)
        气血：<span :class="bt.enemy.hp < bt.enemy.max_hp * 0.3 ? 'cur' : 'black'">{{ bt.enemy.hp }}/{{ bt.enemy.max_hp }}</span><br/>
        -----------<br/>
        <a href="javascript:;" @click="battleAct('attack')">【攻击】</a>
        <a v-if="bt.type !== 'pvp'" href="javascript:;" @click="battleAct('catch')">【捕捉】</a>
        <a href="javascript:;" @click="battleAct('flee')">【逃跑】</a>
        <a href="javascript:;" @click="toggleQuickSet">【快捷键设置】</a>
        <a href="javascript:;" @click="backToMap">【退出战斗】</a><br/>
        <template v-if="showQuickSet">
          -----------<br/>
          <span class="red">战斗场景快捷键设置</span><br/>
          <template v-if="quickSetSlot">
            <span class="black">请选择指定的物品作为快捷键以便在战斗中直接使用</span><br/>
            <template v-if="quickPickTab === 'item'">
              <a href="javascript:;" @click="quickPickTab = 'skill'">技能</a>|<span class="black">药品</span><br/>
            </template>
            <template v-else>
              <span class="black">技能</span>|<a href="javascript:;" @click="quickPickTab = 'item'">药品</a><br/>
            </template>
            <template v-if="quickPickList().length">
              <a v-for="(s, i) in quickPickList()" :key="'qp' + quickPickTab + i" href="javascript:;" @click="pickQuick(quickSetSlot, quickPickTab, quickPickTab === 'item' ? s.id : s.skill_id)">{{ i + 1 }}.{{ s.name }}<template v-if="quickPickTab !== 'item'">（{{ s.mp_cost }}）</template></a><br/>
            </template>
            <template v-else>
              <span class="black">{{ quickPickTab === 'item' ? '你还没有任何可用的丹药' : '你还没有学会任何技能' }}</span><br/>
            </template>
            <a href="javascript:;" @click="pickQuick(quickSetSlot, '', 0)">[清空该快捷]</a>
            <a href="javascript:;" @click="quickSetSlot = 0">[返回]</a><br/>
          </template>
          <template v-else>
            <span v-for="q in quickSlots" :key="'ss' + q.slot">
              快捷{{ q.slot }}：<template v-if="q.ref_id"><a href="javascript:;" @click="openQuickSetFor(q.slot)">{{ q.name }}[改]</a></template><template v-else><a href="javascript:;" @click="openQuickSetFor(q.slot)">选择</a></template>
              <span v-if="q.slot % 3 === 0"><br/></span><template v-else><span class="black">|</span></template>
            </span>
            <a href="javascript:;" @click="resetQuick">重置快捷键</a><br/>
          </template>
          <a href="javascript:;" @click="showQuickSet = false">返回战斗</a><br/>
        </template>
        <template v-if="bt.skills && bt.skills.length">
          法术：<a v-for="s in bt.skills" :key="'sk' + s.skill_id" href="javascript:;" @click="battleAct('skill', s.skill_id)">[{{ s.name }}({{ s.mp_cost }})]</a><br/>
        </template>
        <template v-if="battleItems.length">
          用药：<a v-for="b in battleItems" :key="'bi' + b.id" href="javascript:;" @click="battleAct('item', 0, b.id)">[{{ b.name }}×{{ b.count }}]</a><br/>
        </template>
        <template v-if="quickSlots.length">
          -----------<br/>
          <span class="black">快捷键：</span><br/>
          <span v-for="q in quickSlots" :key="'qs' + q.slot">
            <template v-if="q.ref_id"><a href="javascript:;" @click="useQuick(q)">{{ q.name }}</a></template><template v-else><a href="javascript:;" @click="openQuickSetFor(q.slot)"><span class="gray">快捷{{ q.slot }}</span></a></template>
            <span v-if="q.slot % 3 === 0"><br/></span><template v-else><span class="black">|</span></template>
          </span>
          <br/>
        </template>
      </template>
      <template v-else>
        <span :class="bt.status === 2 ? 'green' : 'red'">{{ battleResultText }}</span><br/>
        <template v-if="battleRewards">经验：+{{ battleRewards.exp }} 银两：+{{ battleRewards.money }}<br/></template>
      </template>
      -----------<br/>
      【战斗直播】<br/>
      <div v-for="(l, i) in bt.log" :key="'bl' + i">{{ i + 1 }}.{{ l }}<br/></div>
      <template v-if="!bt.in_battle">
        -----------<br/>
        <a href="javascript:;" @click="backToMap">返回西游世界</a><br/>
      </template>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('bag')">行囊</a><br/>
    </template>

    <!-- ==================== 状态 ==================== -->
    <template v-else-if="cur === 'attrs' && at">
      <div v-if="g.vip_lv > 0" class="npcimg"><img :key="g.vip_lv" :src="'/static/hxxy/vip/vip' + g.vip_lv + '.png'" @error="$event.target.style.display = 'none'" alt="VIP" /></div>
      ID：{{ g.id }}<br/>
      <a href="javascript:;" @click="go('titles')">称号一览</a><br/>
      头衔：<template v-if="g.title_name">{{ g.title_name }}</template><template v-else>暂无</template><br/>
      <!-- 佩戴头衔图片（复刻原版 xy011 → wp/txdt.php：缺图时提示"称号图片还未制作"） -->
      <template v-if="curTitleId > 0">
        <img v-show="txImgOk" :src="titleImg(curTitleId)" @load="txImgOk = true" @error="txImgErr = true; txImgOk = false" alt="称号" /><br/>
        <span v-if="txImgErr" class="gray">（该称号图片还未制作）</span><br/>
      </template>
      <span class="red">恶名：{{ emzName(g.emz) }}</span><br/>
      昵称：{{ g.name }}<br/>
      性别：{{ g.sex === 2 ? '女' : '男' }}<br/>
      配偶：<template v-if="g.spouse"><span class="red">{{ g.spouse }}</span></template><template v-else>暂无</template><br/>
      住宅：<template v-if="g.has_house"><a href="javascript:;" @click="go('house')">我的住宅</a></template><template v-else>暂无</template><br/>
      国家：<template v-if="gang.my_gang.gang_id"><a href="javascript:;" @click="go('gang')">{{ gang.my_gang.name }}</a></template><template v-else>无</template><br/>
      门派：{{ g.sect_name }}<br/>
      <a href="javascript:;" @click="go('vip')">祝福状态</a><br/>
      等级：{{ g.level }}级<br/>
      HP：{{ g.hp }}/{{ g.max_hp }}<br/>
      MP：{{ g.mp }}/{{ g.max_mp }}<br/>
      攻击：{{ at.attrs.atk }}<br/>
      魔攻：{{ at.attrs.mg }}<br/>
      防御：{{ at.attrs.def }}<br/>
      攻击元素：冰+{{ at.attrs.bg }} 火+{{ at.attrs.hg }} 雷+{{ at.attrs.lg }}<br/>
      防御元素：冰+{{ at.attrs.bf }} 火+{{ at.attrs.hf }} 雷+{{ at.attrs.lf }}<br/>
      经验：{{ g.exp }}/{{ g.exp_need }}<br/>
      修炼经验：<span class="red">{{ g.xiulian_exp }}</span> | <a href="javascript:;" @click="cultToggle">{{ g.xiulian_switch ? '关闭' : '开启' }}</a>{{ g.xiulian_switch ? '(关闭后获得经验)' : '(开启后获得修炼经验)' }}<br/>
      -----------<br/>
      【装备】<br/>
      <div v-for="e in at.equips" :key="'eq' + e.slot">
        {{ e.slot_name }}：<template v-if="e.name"><span class="blue">{{ e.star_prefix }}{{ e.name }}</span><template v-if="e.star > 0">+{{ e.star }}</template><span class="black">|</span><a href="javascript:;" @click="takeoff(e.slot)">卸下</a></template><template v-else><span class="black">无</span></template><br/>
      </div>
      -----------<br/>
      <a href="javascript:;" @click="go('skills')">技能</a>|<a href="javascript:;" @click="go('cultivate')">修炼</a>|<a href="javascript:;" @click="go('bag')">行囊</a><br/>
      <a href="javascript:;" @click="go('pets')">宝宝|宠物</a>|<a href="javascript:;" @click="go('titles')">头衔</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="go('titles')">头衔</a><br/>
    </template>

    <!-- ==================== 行囊/仓库 ==================== -->
    <template v-else-if="cur === 'bag'">
      【{{ storeMode === 1 ? '仓库' : '行囊' }}】{{ bagUsed }}/{{ bagCap }}<br/>
      <a href="javascript:;" :class="storeMode === 0 ? 'cur' : ''" @click="switchStore(0)">[行囊]</a>
      <a href="javascript:;" :class="storeMode === 1 ? 'cur' : ''" @click="switchStore(1)">[仓库]</a><br/>
      <template v-if="bagList.length">
        <div v-for="(b, i) in bagList" :key="'bg' + b.id">
          {{ i + 1 }}.<a href="javascript:;" @click="bagDetail = bagDetail === b.id ? null : b.id">{{ b.star_prefix || '' }}{{ b.name }}</a>×{{ b.count }}<template v-if="b.kind === 'equip' && b.extra && b.extra.star > 0">+{{ b.extra.star }}</template><template v-if="b.bind === 1">(绑定)</template>
          <template v-if="equippedIDs.indexOf(b.id) >= 0"><span class="green">[已穿戴]</span></template>
          <br/>
          <template v-if="bagDetail === b.id">
            <span class="gray">{{ b.desc }}</span><br/>
            <template v-if="b.kind === 'equip'">
              部位：{{ b.slot_name }} 等级：{{ b.level }}<br/>
              <template v-if="b.attrs">气血+{{ b.attrs.hp }} 攻+{{ b.attrs.atk }} 魔+{{ b.attrs.mg }} 防+{{ b.attrs.def }} 冰火雷攻+{{ b.attrs.bg }}/{{ b.attrs.hg }}/{{ b.attrs.lg }} 防+{{ b.attrs.bf }}/{{ b.attrs.hf }}/{{ b.attrs.lf }}<br/></template>
              <template v-if="equippedIDs.indexOf(b.id) < 0">
                <a href="javascript:;" @click="wearEquip(b.id)">[穿戴]</a>
              </template>
              <a href="javascript:;" @click="upgradeEquip(b.id)">[强化({{ upgradeCost(b) }}银两)]</a>
              <a href="javascript:;" @click="holeEquip(b.id)">[打孔]</a><br/>
              <template v-if="b.extra && b.extra.holes > (b.extra.gems || []).length">
                镶嵌：<template v-if="gems.length"><a v-for="gm in gems" :key="'gm' + gm.id" href="javascript:;" @click="gemEquip(b.id, gm.id)">[{{ gm.name }}]</a></template><template v-else><em>背包没有宝石</em></template><br/>
              </template>
            </template>
            <template v-else>
              分类：{{ itemCatName(b.category) }} 等级：{{ b.level }}<br/>
            </template>
            <template v-if="storeMode === 0">
              <template v-if="equippedIDs.indexOf(b.id) < 0">
                <a v-if="b.kind === 'item'" href="javascript:;" @click="useItem(b)">[使用]</a>
                <a href="javascript:;" @click="dropItem(b, 1)">[丢弃]</a>
                <a href="javascript:;" @click="storeMove(b.id, 'in')">[存仓库]</a>
                <a href="javascript:;" @click="openStall(b)">[挂售]</a>
                <a href="javascript:;" @click="openAuctionSell(b)">[拍卖]</a><br/>
              </template>
              <template v-if="b.kind === 'item' && canUseMany(b)">
                数量：<input v-model.number="useCounts[b.id]" size="2" />
                <a href="javascript:;" @click="useItem(b)">[用该数量]</a>
                <a href="javascript:;" @click="dropItem(b, useCounts[b.id] || 1)">[丢该数量]</a><br/>
              </template>
            </template>
            <template v-else>
              <a href="javascript:;" @click="storeMove(b.id, 'out')">[取回行囊]</a><br/>
            </template>
          </template>
        </div>
      </template>
      <template v-else><em>{{ storeMode === 1 ? '仓库空空如也。' : '行囊空空如也，去打怪掉宝或商店购买吧。' }}</em><br/></template>
      <!-- 挂售上架表单（复刻 npcc/gssjwp01.php：数量+单价单页表单） -->
      <template v-if="stallItem">
        -----------<br/>
        <span class="red">你最多可挂售{{ stallItem.name }}x{{ stallItem.count }}</span><br/>
        <span class="black">请输入你要挂售{{ stallItem.name }}数量和单价</span><br/>
        数量：<input v-model.trim="stallCount" size="10" placeholder="数量" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
        单价：<input v-model.trim="stallPrice" size="10" placeholder="单价" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
        <input type="submit" value="挂售" @click="doStallSell" /><br/>
        <a href="javascript:;" @click="stallItem = null">[取消]</a><br/>
      </template>
      <!-- 拍卖上架表单（复刻 npcc/pmsjwp01.php） -->
      <template v-if="aucItem">
        -----------<br/>
        <span class="red">你最多可拍卖{{ aucItem.name }}x{{ aucItem.count }}</span><br/>
        <span class="black">请输入你要拍卖{{ aucItem.name }}数量和单价</span><br/>
        数量：<input v-model.trim="aucCount" size="10" placeholder="数量" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
        单价：<input v-model.trim="aucPrice" size="10" placeholder="单价" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
        <input type="submit" value="拍卖" @click="doAuctionSell" /><br/>
        <a href="javascript:;" @click="aucItem = null">[取消]</a><br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('shop')">商店</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a><br/>
    </template>

    <!-- ==================== 技能 ==================== -->
    <template v-else-if="cur === 'skills'">
      【技能】<br/>
      【我的技能】<br/>
      <template v-if="sk.mine.length">
        <div v-for="s in sk.mine" :key="'ms' + s.skill_id">
          {{ s.name }}<template v-if="s.category === 1">(主动·耗蓝{{ s.mp_cost }}·威力{{ s.multiplier }}%)</template><template v-else>(被动)</template><br/>
          <span class="gray">{{ s.desc }}</span><br/>
        </div>
      </template>
      <template v-else><em>还没有学会任何技能。</em><br/></template>
      -----------<br/>
      【技能书店】<br/>
      <template v-if="sk.store.length">
        <div v-for="s in sk.store" :key="'ss' + s.skill_id">
          <a href="javascript:;" @click="learnSkill(s)">{{ s.name }}</a>({{ s.learn_level }}级可学·{{ s.price }}银两)<br/>
          <span class="gray">{{ s.desc }}</span><br/>
        </div>
      </template>
      <template v-else><em>本门派技能已全部学会！</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
    </template>

    <!-- ==================== 宠物 ==================== -->
    <template v-else-if="cur === 'pets'">
      【宠物】<br/>
      <template v-if="petList.length">
        <div v-for="(pt, i) in petList" :key="'pt' + pt.id">
          {{ i + 1 }}.{{ pt.name }}({{ pt.level }}级<template v-if="pt.mutate === 1">·<span class="red">变异</span></template>★×{{ pt.star }}·品质{{ pt.quality }})<template v-if="pt.fighting === 1"><span class="green">[参战中]</span></template><br/>
          气血：{{ pt.hp }}/{{ pt.max_hp }} 攻：{{ pt.atk }} 防：{{ pt.def }} 魔：{{ pt.mg }}<br/>
          经验：{{ pt.exp }}/{{ pt.exp_need }}<br/>
          <a href="javascript:;" @click="petAct(pt, 'fight')">[参战]</a>
          <a href="javascript:;" @click="petAct(pt, 'rest')">[休息]</a>
          <a href="javascript:;" @click="petAct(pt, 'heal')">[治疗({{ pt.level * 20 }}银两)]</a>
          <a href="javascript:;" @click="petRenameId = petRenameId === pt.id ? 0 : pt.id">[改名]</a>
          <a href="javascript:;" @click="petAct(pt, 'free')">[放生]</a><br/>
          <template v-if="petRenameId === pt.id">
            新名字：<input v-model="petRenameName" maxlength="10" />
            <a href="javascript:;" @click="petAct(pt, 'rename')">[确定]</a><br/>
          </template>
          ----------<br/>
        </div>
      </template>
      <template v-else><em>你还没有宠物。战斗中捕捉，或去宠物店购买！</em><br/></template>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
    </template>

    <!-- ==================== 商店（复刻 xy246/xy122/mdx01/mdx02：列表→详情→购买） ==================== -->
    <template v-else-if="cur === 'shop'">
      <!-- 列表页 -->
      <template v-if="shopPage === 'list'">
        <span v-if="shopMsg" class="black">{{ shopMsg }}</span><br/>
        <span class="black">物品负重：{{ shop.used }}/{{ shop.cap }}</span><br/>
        <span class="black">银两：</span><span class="cur">{{ g.money }}</span> 金豆：<span class="red">{{ g.beans }}</span><br/>
        <a v-for="t in shopTabs" :key="'st' + t.k" href="javascript:;" :class="shopKind === t.k ? 'cur' : ''" @click="loadShop(t.k)">[{{ t.n }}]</a><br/>
        -----------<br/>
        <template v-if="shop.goods.length">
          <div v-for="(gd, i) in shop.goods" :key="'gd' + i">
            <span class="black">{{ i + 1 }}.</span><a href="javascript:;" @click="openShopItem(gd)">{{ gd.name }}</a><template v-if="gd.price > 0">({{ gd.price }}两)</template><template v-else-if="gd.bean_price > 0">({{ gd.bean_price }}金豆)</template><br/>
          </div>
        </template>
        <template v-else><span class="black">本店暂无货物。</span><br/></template>
        <template v-if="shop.pets && shop.pets.length">
          -----------<br/>
          <span class="black">【宠物柜台】</span><br/>
          <div v-for="(pt, i) in shop.pets" :key="'sp' + i">
            <span class="black">{{ i + 1 }}.</span><a href="javascript:;" @click="openShopPet(pt)">{{ pt.name }}</a>({{ pt.level }}级·{{ pt.bean_price }}金豆)<br/>
          </div>
        </template>
        <br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
        -----------<br/>
        <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
      </template>
      <!-- 物品/装备详情页（复刻 xy122） -->
      <template v-else-if="shopPage === 'item' && shopItem">
        <span class="red">{{ shopItem.name }}</span><br/>
        <template v-if="shopItem.desc"><span class="black">描述：{{ shopItem.desc }}</span><br/></template>
        <template v-if="shopItem.price > 0"><span class="black">价格：{{ yl(shopItem.price) }}两</span><br/></template>
        <template v-if="shopItem.bean_price > 0"><span class="black">金豆价：{{ shopItem.bean_price }}金豆</span><br/></template>
        <span class="black">需要等级：{{ shopItem.level }}</span><br/>
        <span class="black">重量：{{ shopItem.weight }}</span><br/>
        <template v-if="shopItem.kind === 'equip' && shopItem.category"><span class="black">部位：{{ slotName(shopItem.category) }}</span><br/></template>
        <span class="black">物品负重：{{ shop.used }}/{{ shop.cap }}</span><br/>
        <span class="black">银两：</span><span class="cur">{{ g.money }}</span> 金豆：<span class="red">{{ g.beans }}</span><br/>
        ------<br/>
        <span class="black">请输入你要购买多少{{ shopItem.name }}呢？</span><br/>
        <form @submit.prevent="doShopBuy('money')">
          <input v-model.trim="shopBuyCount" size="10" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
          <input type="submit" value="银两购买" />
        </form>
        <template v-if="shopItem.bean_price > 0"><a href="javascript:;" @click="doShopBuy('beans')">[金豆购买]</a><br/></template>
        <span v-if="shopMsg" class="red">{{ shopMsg }}</span><br/>
        <a href="javascript:;" @click="shopPage = 'list'; shopMsg = ''">返回列表</a><br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
      <!-- 宠物详情页 -->
      <template v-else-if="shopPage === 'pet' && shopPet">
        <span class="red">{{ shopPet.name }}</span>({{ shopPet.level }}级)<br/>
        <span class="black">价格：{{ shopPet.bean_price }}金豆</span><br/>
        <span class="black">银两：</span><span class="cur">{{ g.money }}</span> 金豆：<span class="red">{{ g.beans }}</span><br/>
        ------<br/>
        <span class="black">请输入你要购买多少{{ shopPet.name }}呢？</span><br/>
        <form @submit.prevent="doShopPetBuy()">
          <input v-model.trim="shopBuyCount" size="10" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
          <input type="submit" value="购买" />
        </form>
        <span v-if="shopMsg" class="red">{{ shopMsg }}</span><br/>
        <a href="javascript:;" @click="shopPage = 'list'; shopMsg = ''">返回列表</a><br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
    </template>

    <!-- ==================== 银行（钱庄，复刻原版 xy257/258/259 三页结构） ==================== -->
    <template v-else-if="cur === 'bank' && bankView === 'main'">
      【银行】<br/>
      钱庄存款：<span class="black">{{ yl(g.bank) }}</span><br/>
      <a href="javascript:;" @click="bankView = 'in'; bankMsg = ''">我要存款</a><br/>
      <a href="javascript:;" @click="bankView = 'out'; bankMsg = ''">我要取款</a><br/>
      ------<br/>
      <a href="javascript:;" @click="go('home')">返回上级</a><br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('wallet')">流水</a><br/>
    </template>
    <template v-else-if="cur === 'bank' && bankView === 'in'">
      钱庄存款：<span class="black">{{ yl(g.bank) }}</span><br/>
      ------<br/>
      身上银子：<span class="black">{{ yl(g.money) }}</span><br/>
      请输入存入的银两：<br/>
      <form @submit.prevent="bankOp('in')">
        <input v-model.number="bankAmount" size="10" inputmode="numeric" /> <input type="submit" value="存款" />
      </form>
      <span v-if="bankMsg" class="red">{{ bankMsg }}</span><br/>
      <a href="javascript:;" @click="bankView = 'main'; bankMsg = ''">放弃存款</a><br/>
      ------<br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('wallet')">流水</a><br/>
    </template>
    <template v-else-if="cur === 'bank' && bankView === 'out'">
      钱庄存款：<span class="black">{{ yl(g.bank) }}</span><br/>
      ------<br/>
      身上银子：<span class="black">{{ yl(g.money) }}</span><br/>
      请输入取出的银两：<br/>
      <form @submit.prevent="bankOp('out')">
        <input v-model.number="bankAmount" size="10" inputmode="numeric" /> <input type="submit" value="取款" />
      </form>
      <span v-if="bankMsg" class="red">{{ bankMsg }}</span><br/>
      <a href="javascript:;" @click="bankView = 'main'; bankMsg = ''">放弃取款</a><br/>
      ------<br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('wallet')">流水</a><br/>
    </template>

    <!-- ==================== 任务（复刻原版分类：主线/支线/日常） ==================== -->
    <template v-else-if="cur === 'quests'">
      【任务】
      <a href="javascript:;" :class="qstTab === 1 ? 'cur' : ''" @click="qstTab = 1">[主线]</a>
      <a href="javascript:;" :class="qstTab === 2 ? 'cur' : ''" @click="qstTab = 2">[支线]</a>
      <a href="javascript:;" :class="qstTab === 3 ? 'cur' : ''" @click="qstTab = 3">[日常]</a><br/>
      【进行中】<br/>
      <template v-if="qst.active.filter(q => q.category === qstTab).length">
        <div v-for="q in qst.active.filter(q => q.category === qstTab)" :key="'qa' + q.quest_id">
          {{ q.name }}({{ q.type === 'hunt' ? '狩猎' : q.type === 'collect' ? '收集' : '对话' }} {{ q.progress }}/{{ q.count }})<br/>
          <span class="gray">{{ q.desc }}</span><br/>
          奖励：经验{{ q.exp }} 银两{{ q.money }}
          <a v-if="q.status === 2 || q.type !== 'hunt'" href="javascript:;" @click="questSubmit(q)">[提交]</a>
          <a href="javascript:;" @click="questAbandon(q)">[放弃]</a><br/>
        </div>
      </template>
      <template v-else><em>暂无进行中的任务。</em><br/></template>
      -----------<br/>
      【可接任务】<br/>
      <template v-if="qst.available.filter(q => q.category === qstTab).length">
        <div v-for="q in qst.available.filter(q => q.category === qstTab)" :key="'qv' + q.quest_id">
          <a href="javascript:;" @click="questAccept(q)">{{ q.name }}</a>({{ q.type === 'hunt' ? '狩猎' + q.count + '只' + q.target : q.type === 'collect' ? '收集' + q.count + '个' + q.target : '拜访' + q.target }})<br/>
          <span class="gray">{{ q.desc }}</span><br/>
          <template v-if="q.from">发布：<span class="blue">{{ q.from }}</span><br/></template>
          奖励：经验{{ q.exp }} 银两{{ q.money }}<template v-if="q.bean > 0"> 金豆{{ q.bean }}</template><template v-if="q.item_name"> 【{{ q.item_name }}】</template><br/>
        </div>
      </template>
      <template v-else><em>本类暂无可接任务，升级后再来看看。</em><br/></template>
      -----------<br/>
      【已完成】<br/>
      <template v-if="qst.done.filter(q => q.category === qstTab).length">
        <div v-for="q in qst.done.filter(q => q.category === qstTab)" :key="'qd' + q.quest_id">{{ q.name }}<span class="gray">(已完成)</span><br/></div>
      </template>
      <template v-else><em>本类暂无已完成任务。</em><br/></template>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 活动中心（复刻原版 xy404：七日登录礼/每日活跃/双倍经验） ==================== -->
    <template v-else-if="cur === 'activities' && act">
      <span class="red">【幻想西游活动中心】</span><br/>
      -----------<br/>
      <span class="red">【七日登录礼】第{{ act.login7.day }}天</span><br/>
      <div v-for="(r, i) in act.login7.days" :key="'l7' + i">
        第{{ i + 1 }}天：银两{{ r[0] }}<template v-if="r[1] > 0">+金豆{{ r[1] }}</template><template v-if="i + 1 === act.login7.day"><span class="red">(今日)</span></template><br/>
      </div>
      <template v-if="act.login7.claimed"><span class="gray">今日已领取，明天再来～</span><br/></template>
      <template v-else><a href="javascript:;" @click="claimActivity('login7', 0)"><span class="red">【领取今日奖励】</span></a><br/></template>
      -----------<br/>
      <span class="red">【每日活跃】当前 {{ act.daily.score }} 分</span><br/>
      <span class="gray">签到+20 副本+30 比武+30 狩猎10只+20</span><br/>
      今日达成：签到<template v-if="act.daily.signed"><span class="green">已</span></template><template v-else><span class="black">否</span></template>
      副本<template v-if="act.daily.dungeon"><span class="green">已</span></template><template v-else><span class="black">否</span></template>
      比武<template v-if="act.daily.arena"><span class="green">已</span></template><template v-else><span class="black">否</span></template>
      狩猎{{ act.daily.hunt }}/10<br/>
      <div v-for="t in act.daily.tiers" :key="'dt' + t.tier">
        {{ t.name }}({{ t.tier }}分)：银两{{ t.money }}<template v-if="t.beans > 0">+金豆{{ t.beans }}</template>
        <template v-if="t.claimed"><span class="gray">[已领取]</span></template>
        <template v-else-if="t.can"><a href="javascript:;" @click="claimActivity('daily' + t.tier, t.tier)"><span class="red">[领取]</span></a></template>
        <template v-else><span class="black">[活跃不足]</span></template><br/>
      </div>
      -----------<br/>
      <span class="red">【双倍经验时段】</span><br/>
      时间：{{ act.exp2x.windows }}<br/>
      状态：<template v-if="act.exp2x.on"><span class="green">进行中，战斗经验翻倍！</span></template><template v-else><span class="black">未开启</span></template><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('signin')">签到</a>.<a href="javascript:;" @click="go('quests')">任务</a><br/>
    </template>

    <!-- ==================== 副本 ==================== -->
    <template v-else-if="cur === 'dungeons'">
      【副本】<br/>
      <span v-for="dg in dungeons" :key="'dg' + dg.dungeon_id">
        <span class="black">{{ dg.done ? '！' : (dg.locked || dg.kills >= dg.floors ? '' : '？') }}</span><a v-if="!dg.locked && !dg.done" href="javascript:;" @click="enterDungeon(dg)"><span class="blue">激活{{ dg.name }}</span></a><span v-else class="black">{{ dg.name }}</span><br/>
        <span class="gray">{{ dg.desc }}（{{ dg.min_level }}级开放）</span><br/>
        <template v-if="dg.locked"><span class="red">[等级不足]</span><br/></template>
        <template v-else-if="dg.done"><span class="gray">[今日已完成，明日再来]</span><br/></template>
        <template v-else-if="dg.kills > 0"><span class="black">已击杀{{ dg.kills }}/{{ dg.floors }}只守护BOSS</span><br/></template>
        ----------<br/>
      </span>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== BOSS ==================== -->
    <template v-else-if="cur === 'bosses'">
      【世界BOSS】<br/>
      <div v-for="(b, i) in bossList" :key="'bs' + b.boss_id">
        {{ i + 1 }}.<a href="javascript:;" @click="challengeBoss(b)">{{ b.name }}</a>({{ b.level }}级·气血{{ b.max_hp }}·攻{{ b.atk }})<br/>
        <template v-if="b.alive">状态：<span class="green">可挑战</span><br/></template>
        <template v-else>状态：<span class="red">已被击败，{{ b.respawn_at }}刷新</span><br/></template>
        ----------<br/>
      </div>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 修炼（复刻 xy427/428/432 + xy052 开关） ==================== -->
    <template v-else-if="cur === 'cultivate'">
      <!-- 主页：我的修炼 -->
      <template v-if="!cultView">
        我的修炼如下：<br/>
        修炼经验：<span class="red">{{ cult.exp }}</span><a href="javascript:;" @click="cultToggle"><span class="blue">|{{ cult.switch === 1 ? '关闭' : '开启' }}</span></a><span class="black">({{ cult.switch === 1 ? '关闭后获得经验' : '开启后获得修炼经验' }})</span><br/>
        西游声望：<span class="red">{{ cult.sw }}</span><br/>
        <span v-for="t in cult.tracks" :key="'xt' + t.slot">
          <a href="javascript:;" @click="cultView = 'up' + t.slot"><span class="blue">【人物修炼（{{ t.name }}）】{{ xlTrackName(t) }}</span></a><br/>
        </span>
        <br/>
        <span v-for="t in cult.tracks" :key="'xb' + t.slot">
          <span class="red">修炼（{{ t.name }}）加成：</span><span class="black">{{ t.bonus }}{{ { 1: '血量', 2: '攻击', 3: '魔攻', 4: '防御' }[t.slot] }}</span><br/>
        </span>
        <a href="javascript:;" @click="cultView = 'intro'"><span class="blue">人物修炼（介绍）</span></a><br/>
        <a href="javascript:;" @click="cultExchangeDan"><span class="blue">【一键】各类修炼丹兑换修炼经验</span></a><br/>
        <span class="black">----------------------</span><br/>
        <a href="javascript:;" @click="go('home')"><span class="blue">返回游戏</span></a><br/>
        <span class="black">----------------------</span><br/>
      </template>
      <!-- 升级详情（复刻 xy428/433） -->
      <template v-else-if="cultView.indexOf('up') === 0">
        <span v-for="t in cult.tracks" :key="'xd' + t.slot">
          <template v-if="cultView === 'up' + t.slot">
            修炼：{{ xlTrackName(t) }}<br/>
            <template v-if="t.capped">
              <span class="red">已达到至高无上的境界了</span><br/>
            </template>
            <template v-else>
              修炼所需：{{ xlNeedText(t) }}<br/>
              <a href="javascript:;" @click="cultUp(t.slot)"><span class="blue">【开始修炼】</span></a><br/>
            </template>
            <br/>
          </template>
        </span>
        <a href="javascript:;" @click="cultView = ''"><span class="blue">返回上级</span></a><br/>
        <a href="javascript:;" @click="go('home')"><span class="blue">返回游戏</span></a><br/>
        <span class="black">----------------------</span><br/>
      </template>
      <!-- 介绍（复刻 xy432） -->
      <template v-else-if="cultView === 'intro'">
        <span class="black">各项修炼介绍</span><br/>
        <span class="black">1.修炼一共分为四种血攻魔防</span><br/>
        <span class="black">2.修炼阶级分炼气、筑基、开光、金丹、元婴、出窍、合体、渡劫、寂灭和大乘（等级每升20级可获得新的阶级修炼）</span><br/>
        <span class="black">3.每个阶级可修炼20层，越往上属性越多当然需要的东西也更多</span><br/>
        <span class="black">4.当度过大乘期后将开启飞仙隐藏模式</span><br/>
        <a href="javascript:;" @click="cultView = ''"><span class="blue">返回修炼</span></a><br/>
        <a href="javascript:;" @click="go('home')"><span class="blue">返回游戏</span></a><br/>
        <span class="black">----------------------</span><br/>
      </template>
    </template>

    <!-- ==================== 头衔（复刻 xy011 管理 + xy477 称号一览 + xy478 详情） ==================== -->
    <template v-else-if="cur === 'titles'">
      <template v-if="!titleView">
        【头衔】当前佩戴：<template v-if="titles.worn > 0">{{ wornTitleName }}</template><template v-else>暂无</template><a href="javascript:;" @click="titleView = 'manage'"><span class="blue">&nbsp;管理</span></a><br/>
        <a href="javascript:;" @click="titleTab = 'list'"><span :class="titleTab === 'list' ? 'cur' : 'blue'">称号</span></a><span class="black">|</span><span class="black">半周年称号</span><span class="black">|</span><span class="black">重阳活动称号</span><span class="black">|</span><span class="black">万圣节活动称号</span><br/>
        <div v-for="(t, i) in titles.list" :key="'tl' + t.title_id">
          <span class="black">{{ (titles.page - 1) * 20 + i + 1 }}.</span><a href="javascript:;" @click="openTitle(t)"><span class="blue">{{ t.name }}</span></a><template v-if="t.owned"><span class="red">（已获得）</span></template><template v-else><span class="black">（未获得）</span></template><br/>
        </div>
        <a v-if="titles.page > 1" href="javascript:;" @click="loadTitles(titles.page - 1)"><span class="blue">上一页</span></a><template v-if="titles.page > 1 && titles.page < titles.total_pages"><span class="black">|</span></template><a v-if="titles.page < titles.total_pages" href="javascript:;" @click="loadTitles(titles.page + 1)"><span class="blue">下一页</span></a><br/>
        <br/>
        <a href="javascript:;" @click="go('attrs')"><span class="blue">我的状态</span></a><br/>
      </template>
      <!-- 详情（复刻 xy478：红字名字+描述） -->
      <template v-else-if="titleView === 'detail'">
        <span class="red">{{ titleDetail.name }}</span><br/>
        <!-- 称号图片（复刻原版 xy478 → wp/txdt.php） -->
        <template v-if="curTitleId > 0">
          <img v-show="txImgOk" :src="titleImg(curTitleId)" @load="txImgOk = true" @error="txImgErr = true; txImgOk = false" alt="称号" /><br/>
          <span v-if="txImgErr" class="gray">（该称号图片还未制作）</span><br/>
        </template>
        <span class="black">描述：{{ titleDetail.desc }}</span><br/>
        <template v-if="!titleDetail.owned">
          <span class="black">激活需要 {{ titleDetail.price }} 银两，激活后永久增加属性。</span><br/>
          <a href="javascript:;" @click="activateTitle(titleDetail)"><span class="blue">[激活头衔]</span></a><br/>
        </template>
        <template v-else>
          <template v-if="titles.worn === titleDetail.title_id"><span class="green">[佩戴中]</span></template>
          <a href="javascript:;" @click="wearTitle(titleDetail)"><span class="blue">{{ titles.worn === titleDetail.title_id ? '[摘下头衔]' : '[佩戴]' }}</span></a><br/>
        </template>
        <a href="javascript:;" @click="titleView = ''"><span class="blue">返回上级</span></a><br/>
      </template>
      <!-- 管理（复刻 xy251：我的头衔佩戴管理） -->
      <template v-else-if="titleView === 'manage'">
        【头衔管理】当前佩戴：<template v-if="titles.worn > 0">{{ wornTitleName }}</template><template v-else>暂无</template><br/>
        <template v-if="titles.mine.length">
          <div v-for="t in titles.mine" :key="'tmg' + t.title_id">
            <a href="javascript:;" @click="openTitle(t)"><span class="blue">{{ t.name }}</span></a><template v-if="titles.worn === t.title_id"><span class="green">[佩戴中]</span></template><br/>
          </div>
        </template>
        <template v-else><em>还未激活任何头衔。</em><br/></template>
        <a v-if="titles.worn > 0" href="javascript:;" @click="wearTitle({ title_id: 0 })"><span class="blue">[摘下头衔]</span></a><br/>
        <a href="javascript:;" @click="titleView = ''"><span class="blue">返回上级</span></a><br/>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('attrs')">状态</a><br/>
    </template>

    <!-- ==================== 福利中心（复刻原版 xy307/408/409/410/417/418/419/420） ==================== -->
    <template v-else-if="cur === 'signin'">
      <span class="red">幻想西游福利中心</span><br/>
      <a href="javascript:;" @click="wfTab = 'gift'"><span :class="wfTab === 'gift' ? 'cur' : 'blue'">每日福利</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'active'"><span :class="wfTab === 'active' ? 'cur' : 'blue'">每日活跃</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'sign'"><span :class="wfTab === 'sign' ? 'cur' : 'blue'">每日签到</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'promo'"><span :class="wfTab === 'promo' ? 'cur' : 'blue'">每日宣传</span></a><br/>
      <span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'noble1'"><span :class="wfTab === 'noble1' ? 'cur' : 'blue'">黄金贵族</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'noble2'"><span :class="wfTab === 'noble2' ? 'cur' : 'blue'">铂金贵族</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'noble3'"><span :class="wfTab === 'noble3' ? 'cur' : 'blue'">钻石皇族</span></a><span class="black">|</span>
      <a href="javascript:;" @click="wfTab = 'noble4'"><span :class="wfTab === 'noble4' ? 'cur' : 'blue'">至尊皇族</span></a><br/>
      <br/>
      <!-- 每日福利：神秘礼物 -->
      <template v-if="wfTab === 'gift'">
        <span class="pink">======</span><br/>
        <span class="red" v-if="wf.gift.status === 'done'">{{ wf.gift.msg }}</span>
        <template v-else>
          <span class="red">第{{ wf.gift.next }}份【神秘礼物】</span>
          <template v-if="wf.gift.ready"><a href="javascript:;" :class="wfTab === 'gift' ? 'cur' : ''" @click="giftClaim"><span class="blue">领取</span></a></template>
          <template v-else><span class="black">[{{ fmtWait(wf.gift.remaining) }}]</span><a href="javascript:;" :class="wfTab === 'gift' ? 'cur' : ''" @click="loadWelfare"><span class="blue">刷新</span></a></template>
          <br/>
        </template>
        <span class="pink">======</span><br/>
        <a href="javascript:;" @click="wfTab = 'gift'"><span class="blue">兑换说明</span></a><br/>
        <span class="black">介绍：每天凌晨更新10份【神秘礼物】也是对广大玩家的福利每份礼物随机，领的越多东西越好！（只要在线就能获取）</span><br/>
        <span class="black">VIP{{ wf.vip.level }}级：可用银两兑换【万能果】（详见VIP页）</span><br/>
        <br/>
      </template>
      <!-- 每日活跃：活跃度任务 -->
      <template v-else-if="wfTab === 'active'">
        <span class="red">活跃度任务</span><br/>
        <div v-for="t in wf.active.tasks" :key="'wt' + t.name">
          <span class="black">{{ t.name }}：</span><span :class="t.done ? 'green' : 'black'">{{ t.done ? t.need : 0 }}/{{ t.need }}</span><span class="gray">(+{{ t.score }}分)</span><br/>
        </div>
        <span class="black">今日活跃度：{{ wf.active.score }}/{{ wf.active.total }}</span><br/>
        <div v-for="t in wf.active.tiers" :key="'wdt' + t.tier">
          {{ t.name }}({{ t.tier }}分)：<template v-if="t.claimed"><span class="gray">已领取</span></template><template v-else-if="t.can"><a href="javascript:;" @click="claimActivity('daily' + t.tier, 0); loadWelfare()"><span class="red">【领取】</span></a></template><template v-else><span class="gray">未达成</span></template><br/>
        </div>
        <span class="black">活跃度获取：签到+20 / 副本+30 / 比武+30 / 狩猎10只+20</span><br/>
      </template>
      <!-- 每日签到（复用原今日签到功能） -->
      <template v-else-if="wfTab === 'sign'">
        <span class="red">【{{ sign.month_cn }}签到活动】</span><br/>
        <a href="javascript:;" @click="signDetail = !signDetail"><span class="blue">【{{ sign.month_cn }}签到奖励一览】</span></a><br/>
        <template v-if="signDetail">
          <span v-for="t in sign.tiers" :key="'sd' + t.tier" class="gray">{{ t.tier }}次：银两{{ t.money }}+金豆{{ t.beans }}　</span><br/>
        </template>
        今日签到情况：<template v-if="sign.today_signed"><span class="red">已签到</span></template><template v-else><a href="javascript:;" @click="doSignin"><span class="red">【每日签到】</span></a></template><br/>
        <br/>
        <span v-for="t in sign.tiers" :key="'st' + t.tier">
          <span class="black">1.</span><span style="color:#87CEEB">{{ t.tier }}次签到奖励--</span><template v-if="t.claimed"><span class="gray">已领取</span></template><template v-else><a href="javascript:;" @click="claimSign(t.tier)"><span class="red">【领取】</span></a></template><br/>
        </span>
        <span class="red">本月已累计签到&nbsp;{{ sign.count }}&nbsp;次</span><br/>
        <span class="black">介绍：玩家每天可签到一次，每月1日清零</span><br/>
        <span class="black">当累计达到2,5,10,15,25可领取丰厚的奖励</span><br/>
      </template>
      <!-- 每日宣传 -->
      <template v-else-if="wfTab === 'promo'">
        <span class="black">幻想西游推广计划</span><span class="red">此活动长期有效</span><br/>
        <span class="black">1.每日在其他WAP游戏群发布宣传语截图并且发到管理员，得【西游宣传礼包】x1</span><br/>
        <span class="black">2.邀请新人进群并且注册游戏，邀请人得【西游邀请礼包】x1，被邀请人得【西游新人礼包】x1</span><br/>
        <span class="black">3.拉人送自充卡。使用自充卡不参与积分排名，与充值赠送</span><br/>
        <span class="black">4.拉1人送10自充，拉2人送20元自充，必须为真实有效玩家，以此类推，多拉多送禁小号禁作弊，一经发现严惩不贷，不排除封号</span><br/>
        <br/><br/>
        <span class="pink">======宣传语======</span><br/>
        <button @click="copyPromo">点击复制宣传语</button><br/>
        <span class="red">进群填写我的邀请游戏ID：{{ playerID }}</span><br/>
        <span class="red">填写邀请ID即可领取超值【三区水帘洞助力包】包含【幻想套装】【vip练级卷】x20，【10亿修炼经验丹】x10，【万能果】x100，【1万西游声望卷轴】x100，【1万法宝经验卷轴】x100，〖瞌睡虫〗（典藏版）x5</span><br/>
        <span class="red">我不断的寻找，有你的世界在哪儿</span><br/>
        <span class="red">新区【水帘洞】人气火爆，进群领取豪华大礼包，只等你来！</span><br/>
      </template>
      <!-- 贵族 -->
      <template v-else-if="wfTab.indexOf('noble') === 0">
        <span class="red" v-if="nobleCurOwned">剩余：每日1次（每天/次）</span>
        <span class="red">【{{ nobleCur.name }}】</span>
        <template v-if="nobleCurOwned">
          <template v-if="nobleCurClaimed"><span class="gray">今日已领取，明天再来～</span><br/></template>
          <template v-else><a href="javascript:;" @click="nobleClaim(nobleCur.tier)"><span class="red">【领取】</span></a><br/></template>
        </template>
        <template v-else>
          <span class="red">亲！【{{ nobleCur.name }}】已到期，或者未开通（在游戏左下角充值联系GM并告知开通月卡）</span><br/>
        </template>
        <br/>
        <span class="black">介绍：{{ nobleCur.intro }}</span><br/>
        <span class="black">【{{ nobleCur.name }}】{{ nobleCur.price }}（单购,不计算vip积分）</span><br/>
        <span class="black">〖{{ nobleCur.box }}〗（随机必得三四五级石头）</span><br/>
        <span class="black">注：各类贵族可以一起开通奖励更丰厚</span><br/>
      </template>
      <br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== 排行 ==================== -->
    <template v-else-if="cur === 'rank'">
      【排行榜】<br/>
      <a v-for="t in rankTabs" :key="'rt' + t.k" href="javascript:;" :class="rankType === t.k ? 'cur' : ''" @click="loadRank(t.k)">[{{ t.n }}]</a><br/>
      <div v-for="r in rankList" :key="'rk' + r.rank">
        {{ r.rank }}.<a href="javascript:;" @click="viewPlayer(r.player_id)">{{ r.name }}</a>({{ r.level }}级)——{{ rankValName }}：{{ r.val }}<br/>
      </div>
      <template v-if="!rankList.length"><em>暂无上榜数据。</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== 聊天 ==================== -->
    <template v-else-if="cur === 'chat'">
      【西游茶馆】(世界频道)<br/>
      <div v-for="m in chatList" :key="'cm' + m.id">
        <a href="javascript:;" class="nk" @click="viewPlayer(m.player_id)">{{ m.name }}</a>：{{ m.content }}<em>[{{ fmtTime(m.created_at) }}]</em><br/>
      </div>
      <template v-if="!chatList.length"><em>茶馆里静悄悄的，来说第一句话吧。</em><br/></template>
      <form @submit.prevent="sendChat">
        <input v-model="chatInput" maxlength="100" />
        <input type="submit" value="发言" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== 好友/黑名单（复刻 xy114 好友页 / xy116 黑名单页） ==================== -->
    <template v-else-if="cur === 'friends'">
      <template v-if="frdTab === 1">
        <span class="black">好友</span>|<a href="javascript:;" @click="frdTab = 2">黑名单</a><br/>
        <span class="black">好友列表（{{ frd.friends.length }}个）</span><br/>
        <template v-if="frd.friends.length">
          <div v-for="(f, i) in frd.friends" :key="'fd' + f.player_id">
            <span class="black">{{ i + 1 }}.</span><a href="javascript:;" @click="viewPlayer(f.player_id)">{{ f.name }}</a><span class="black">|</span><a href="javascript:;" @click="removeFriend(f)">删除</a><br/>
          </div>
        </template>
        <template v-else><span class="black">目前还没有结交到好友</span><br/></template>
      </template>
      <template v-else>
        <a href="javascript:;" @click="frdTab = 1">好友</a>|<span class="black">黑名单</span><br/>
        <span class="black">黑名单列表（{{ frd.blacks.length }}个）</span><br/>
        <template v-if="frd.blacks.length">
          <div v-for="(f, i) in frd.blacks" :key="'fb' + f.player_id">
            <span class="black">{{ i + 1 }}.</span><a href="javascript:;" @click="viewPlayer(f.player_id)">{{ f.name }}</a><span class="black">|</span><a href="javascript:;" @click="removeFriend(f)">删除</a><br/>
          </div>
        </template>
        <template v-else><span class="black">目前黑名单内还空空如也</span><br/></template>
      </template>
      <br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      <span class="black">----------------------</span><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('chat')">聊天</a><br/>
    </template>

    <!-- ==================== 国家（复刻 xy172 主页 / xy175 成员 / xy185 捐献 / xy186 商城 / xy176 任命） ==================== -->
    <template v-else-if="cur === 'gang'">
      <template v-if="gang.my_gang && gang.my_gang.gang_id">
        <!-- 主页（xy172） -->
        <template v-if="!gangSub">
          <span class="black">{{ gang.my_gang.name }}（{{ gang.my_gang.level }}级）</span>
          <a href="javascript:;" @click="gangUpgrade">升级</a><br/>
          <span class="black">【首任君主】:{{ gang.my_gang.founder_name }}</span><br/>
          <span class="black">【现任君主】:{{ gang.my_gang.leader_name }}</span><br/>
          <span class="black">【辅助大臣】:{{ gangOffName(2) }}</span><br/>
          <span class="black">【军机大臣】:{{ gangOffName(3) }}</span><br/>
          <span class="black">【财政大臣】:{{ gangOffName(4) }}</span><br/>
          <span class="black">【工部大臣】:{{ gangOffName(5) }}</span><br/>
          <span class="black">【外交大臣】:{{ gangOffName(6) }}</span><br/>
          <span class="black">【军团长】:{{ gangOffName(7) }}</span><br/>
          <span class="black">国家人数:{{ gang.my_gang.member_count }}/{{ gang.my_gang.member_max }}</span><br/>
          <span class="black">国家经验:{{ gang.my_gang.exp }}/{{ gang.my_gang.exp_max }}</span><br/>
          <span class="red">国家资金:{{ yl(gang.my_gang.money) }}</span><br/>
          <span class="black">国家声望:{{ gang.my_gang.sw }}</span><br/>
          <span class="black">可用贡献:{{ gang.my_gang.contribution }}点</span><br/>
          <span class="black">历史贡献:{{ gang.my_gang.total_contribution }}点</span><br/>
          <a href="javascript:;" @click="gangGo('mall')">国家商城</a><br/>
          <a href="javascript:;" @click="gangGo('members')">国家成员</a><br/>
          <a href="javascript:;" @click="gangGo('donate')">捐献银两</a><br/>
          <template v-if="gang.my_gang.is_monarch">
            <a href="javascript:;" @click="gangGo('appoint')">任命官员</a><br/>
            <a href="javascript:;" @click="gangGo('dissolve')">解散国家</a><br/>
          </template>
          <template v-else>
            <a href="javascript:;" @click="gangGo('quit')">退出国家</a><br/>
          </template>
          <br/>
          <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 成员列表（xy175：名字/职务/贡献，自己红字，官员可罢免/踢出） -->
        <template v-else-if="gangSub === 'members'">
          国家成员列表<br/>
          名字/职务/贡献<br/>
          <div v-for="(m, i) in gang.my_gang.members" :key="'gm' + m.player_id">
            {{ i + 1 }}.<a v-if="m.player_id !== g.id" href="javascript:;" @click="viewPlayer(m.player_id)">{{ m.name }}</a><span v-else class="red">{{ m.name }}.</span><span class="black">[{{ m.role_name }}]|{{ m.total_contribution }}点|</span><template v-if="gang.my_gang.can_manage && m.player_id !== g.id && m.role >= 2"><a href="javascript:;" @click="gangDismiss(m)">罢免官职</a><span class="black">|</span></template><template v-if="gang.my_gang.role >= 1 && m.player_id !== g.id && m.role !== 1"><a href="javascript:;" @click="gangGo('kick', m)">踢出</a></template><br/>
          </div>
          <a href="javascript:;" @click="gangGo('')">返回国家</a>.<a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 踢出确认（xy184：红字确认 + 是的，我已想好了 / 不，我点错了） -->
        <template v-else-if="gangSub === 'kick'">
          <span class="red">你确定要将{{ gangKickTarget ? gangKickTarget.name : '' }}踢出{{ gang.my_gang.name }}么？？</span><br/>
          <a href="javascript:;" @click="gangKickDo">是的，我已想好了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('members')">不，我点错了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">返回国家</a><br/>
          <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 退出国家确认（xy182 结构：是的，我已想好了 / 不，我点错了） -->
        <template v-else-if="gangSub === 'quit'">
          <span class="red">你确定要退出{{ gang.my_gang.name }}么？？</span><br/>
          <a href="javascript:;" @click="gangQuitDo">是的，我已想好了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">不，我点错了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">返回国家</a><br/>
          <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 解散国家确认（xy173：红字确认 + 是的，我已想好了 / 不，我点错了） -->
        <template v-else-if="gangSub === 'dissolve'">
          <span class="red">你确定要将国家{{ gang.my_gang.name }}解散掉么？？</span><br/>
          <a href="javascript:;" @click="gangDissolve">是的，我已想好了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">不，我点错了</a><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">返回国家</a><br/>
          <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 国家商城（xy186：1~10级页签，等级不足显示施工中） -->
        <template v-else-if="gangSub === 'mall'">
          <span class="black">{{ gang.my_gang.name }}的国家商城</span><br/>
          <span class="black">我可以使用的国家贡献：{{ gangMall.contribution }}点</span><br/>
          <span class="black">----------------------</span><br/>
          <span v-for="t in gangMall.tabs" :key="'mt' + t.level"><a href="javascript:;" @click="gangMallTab = t.level">{{ t.level }}级商城</a><span v-if="t.level < 10" class="black">|</span></span><br/>
          <template v-if="gangMallTabObj">
            <template v-if="gangMallTabObj.unlocked">
              <span class="red">【{{ gangMallTabObj.level }}级国家商城】</span><br/>
              <div v-for="it in gangMallTabObj.items" :key="'mi' + it.item_id">
                <a href="javascript:;" @click="gangMallBuy(it)">{{ it.name }}</a><span class="black">|</span><a href="javascript:;" @click="gangMallBuy(it)">兑换（{{ it.contribution }}贡献+{{ ylNum(it.silver) }}银两）</a><br/>
              </div>
            </template>
            <template v-else><span class="black">一大波商品正在赶来中（前方高能正在施工中）</span><br/></template>
          </template>
          <span class="black">----------------------</span><br/>
          <span class="red">温馨提示：国家商城等级越高商品越丰厚越多（在国家商城购买商品需要消耗国家贡献）</span><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">返回国家</a>.<a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 捐献银两（xy185：100万~100亿，100万=1贡献） -->
        <template v-else-if="gangSub === 'donate'">
          <span class="black">为自己的国家尽自己的一份力，你考虑好了吗？</span><br/>
          <form @submit.prevent="gangDonate">
            <input v-model.number="gangDonateAmount" type="tel" placeholder="请输入你需要捐献的银两" size="22" /><br/>
            <input type="submit" value="捐献" /><br/>
          </form>
          <span class="red">温馨提示：捐献银两时请注意国库银两上限以免造成浪费！！</span><br/>
          <span class="gray">（单笔100万~100亿银两，每100万银两=1点国家贡献）</span><br/>
          <br/>
          <a href="javascript:;" @click="gangGo('')">返回国家</a>.<a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>

        <!-- 任命官员（xy177选职务 → xy176选人） -->
        <template v-else-if="gangSub === 'appoint'">
          <template v-if="!gangAppointRole">
            <span class="black">请选择你要任命国家官员职务</span><br/>
            <span v-for="r in [2, 3, 4, 5, 6, 7]" :key="'ar' + r"><a href="javascript:;" @click="gangAppointRole = r">任命{{ roleNames[r] }}</a><br/></span>
          </template>
          <template v-else>
            <span class="black">请选择你要将谁任命为【{{ roleNames[gangAppointRole] }}】</span><br/>
            <div v-for="m in gang.my_gang.members" :key="'am' + m.player_id">
              <template v-if="m.role === 0">{{ m.name }}<span class="black">[{{ m.role_name }}]|</span><a href="javascript:;" @click="gangAppoint(m)">任命{{ roleNames[gangAppointRole] }}</a><br/></template>
            </div>
          </template>
          <br/>
          <a href="javascript:;" @click="gangAppointRole = 0">返回任命</a>.<a href="javascript:;" @click="gangGo('')">返回国家</a>.<a href="javascript:;" @click="go('home')">返回游戏</a><br/>
          ----------------------<br/>
        </template>
      </template>

      <!-- 无国家：创建（xy171：≤7字，1亿银两+玄铁令x5） -->
      <template v-else>
        <span class="black">你还未加入任何国家！！</span><br/>
        <span class="black">请输入你要建立的国家名字</span><br/>
        <form @submit.prevent="gangNew">
          <input v-model="gangNameInput" placeholder="请输入要建立的国家名字" maxlength="7" /><br/>
          <input type="submit" value="确认" /><br/>
        </form>
        <span class="red">建立国家需要银两1亿和玄铁令x5</span><br/>
        <br/>
      </template>
      <template v-if="!gangSub">
        【国家列表】<br/>
        <div v-for="gp in gang.gangs" :key="'gg' + gp.gang_id">
          {{ gp.name }}({{ gp.level }}级)——<span class="gray">{{ gp.member_count }}/{{ gp.member_max }}人</span>
          <template v-if="!gang.my_gang || !gang.my_gang.gang_id"><a href="javascript:;" @click="gangJoin(gp)">[加入]</a></template><br/>
        </div>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
    </template>

    <!-- ==================== 结婚 ==================== -->
    <template v-else-if="cur === 'marriage'">
      【姻缘】<br/>
      <template v-if="marriage.status === 2">
        <span class="red">你已与【{{ marriage.other_name }}】结为道侣！</span><br/>
        <a href="javascript:;" @click="doDivorce">[离婚]</a><br/>
      </template>
      <template v-else-if="marriage.status === 1 && marriage.incoming">
        <span class="red">【{{ marriage.other_name }}】向你求婚了！</span><br/>
        <a href="javascript:;" @click="doAgreeMarry">[答应求婚]</a><br/>
      </template>
      <template v-else-if="marriage.status === 1">
        你已向【{{ marriage.other_name }}】求婚，等待对方答应……<br/>
      </template>
      <template v-else>
        <em>你还是单身，寻一位道侣共闯西游吧！(彩礼5000银两)</em><br/>
        <form @submit.prevent="doPropose">
          对方名字：<input v-model="marryName" maxlength="12" />
          <input type="submit" value="求婚" />
        </form>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('friends')">好友</a><br/>
    </template>

    <!-- ==================== 住宅 ==================== -->
    <template v-else-if="cur === 'house'">
      【住宅】<br/>
      <template v-if="furniture.length">
        已购家具（属性加成已生效）：{{ furnitureTotal }}<br/>
        <div v-for="(f, i) in furniture" :key="'fu' + i">{{ f.name }}(+{{ f.val }}{{ bonusName(f.bonus) }})<br/></div>
      </template>
      <template v-else><em>你还没有购置家具。家具可为角色提供属性加成！</em><br/></template>
      -----------<br/>
      【家具铺】<br/>
      <div v-for="f in houseCatalog" :key="'hc' + f.fid">
        <a href="javascript:;" @click="buyFurniture(f)">{{ f.name }}</a>({{ f.price }}银两·+{{ f.val }}{{ bonusName(f.bonus) }})<br/>
      </div>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
    </template>

    <!-- ==================== 参观他人住宅 ==================== -->
    <template v-else-if="cur === 'housevisit' && visitHouse">
      【{{ visitHouse.owner_name }}的住宅】<br/>
      <template v-if="visitHouse.furniture.length">
        <div v-for="(f, i) in visitHouse.furniture" :key="'vf' + i">{{ f.name }}(+{{ f.val }}{{ bonusName(f.bonus) }})<br/></div>
      </template>
      <template v-else><em>主人还没购置家具，家徒四壁……</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">返回游戏首页</a><br/>
    </template>

    <!-- ==================== 挂售（复刻 xy225 分类 → xy219/229/236 我的挂售） ==================== -->
    <template v-else-if="cur === 'stalls'">
      <template v-if="stallPage === 'cat'">
        <span class="black">请选择挂售分类</span><br/>
        <a href="javascript:;" @click="openStallMine('item')">1.挂售物品类</a><br/>
        <a href="javascript:;" @click="openStallMine('equip')">2.挂售装备类</a><br/>
        <a href="javascript:;" @click="openStallMine('gem')">3.挂售宝石类</a><br/>
        <br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
      <template v-else>
        <span class="black">我的挂售</span><br/>
        <span class="black">挂售容量：{{ stallMine.used }}/{{ stallMine.capacity }}</span><br/>
        <template v-if="stallMine.stalls.length">
          <div v-for="s in stallMine.stalls" :key="'sm' + s.stall_id">
            <a href="javascript:;" @click="stallDetailId = stallDetailId === s.stall_id ? 0 : s.stall_id">{{ s.name }}</a><span class="blue">x{{ s.count }}（{{ s.price }}两/个）|</span><a href="javascript:;" @click="stallCancelTarget = stallCancelTarget === s.stall_id ? 0 : s.stall_id">下架</a><br/>
            <template v-if="stallDetailId === s.stall_id"><span class="gray">{{ s.desc }}</span><br/></template>
            <template v-if="stallCancelTarget === s.stall_id">
              <span class="red">你最多可下架{{ s.name }}x{{ s.count }}</span><br/>
              <span class="black">请输入你要下架多少{{ s.name }}呢？</span><br/>
              数量：<input v-model.number="stallCancelCount" size="3" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
              <a href="javascript:;" @click="doStallCancel(s, stallCancelCount || s.count)">[确定]</a>
              <a href="javascript:;" @click="doStallCancel(s, s.count)">[下架全部]</a><br/>
            </template>
          </div>
        </template>
        <template v-else><span class="black">暂时无任何挂售的物品</span><br/></template>
        <br/>
        <a href="javascript:;" @click="stallPage = 'cat'">返回上级</a><br/>
        <br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
    </template>

    <!-- ==================== 他人挂售（复刻 xy222："{名字}的挂售："） ==================== -->
    <template v-else-if="cur === 'stallof' && stallOf">
      <span class="black">{{ stallOf.seller_name }}的挂售：</span><br/>
      <span class="black">挂售容量：{{ stallOf.used }}/{{ stallOf.capacity }}</span><br/>
      <template v-if="stallOf.stalls.length">
        <div v-for="s in stallOf.stalls" :key="'so' + s.stall_id">
          <a href="javascript:;" @click="stallBuyTarget = stallBuyTarget === s.stall_id ? 0 : s.stall_id">{{ s.name }}</a><span class="blue">x{{ s.count }}（{{ s.price }}两/个）</span><br/>
          <template v-if="stallDetailId === s.stall_id"><span class="gray">{{ s.desc }}</span><br/></template>
          <template v-if="stallBuyTarget === s.stall_id">
            <span class="red">你最多可购买{{ s.name }}x{{ s.count }}</span><br/>
            <span class="black">请输入你要购买多少{{ s.name }}呢？</span><br/>
            数量：<input v-model.number="stallBuyCount" size="3" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
            <input type="button" value="购买" @click="doStallBuy(s, stallBuyCount || s.count)" /><br/>
          </template>
        </div>
      </template>
      <template v-else><span class="black">暂时无任何挂售的物品</span><br/></template>
      <br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
    </template>

    <!-- ==================== 全区拍卖场（复刻 xy489/499） ==================== -->
    <template v-else-if="cur === 'auction'">
      <template v-if="auctionView === 'mine'">
        <span class="black">我的拍卖</span><br/>
        <template v-if="auctionMine.length">
          <div v-for="(a, i) in auctionMine" :key="'am' + a.auction_id">
            {{ i + 1 }}.<a href="javascript:;" @click="doAuctionCancel(a)">{{ a.name }}</a><span class="blue">x{{ a.count }}（{{ a.price }}两/个）|</span><a href="javascript:;" @click="doAuctionCancel(a)">下架</a><br/>
          </div>
        </template>
        <template v-else><span class="black">暂无任何拍卖信息</span><br/></template>
        <br/>
        <a href="javascript:;" @click="auctionView = 'list'">返回上级</a><br/>
      </template>
      <template v-else>
        <span class="red">〖全区拍卖场〗</span><br/>
        <a href="javascript:;" @click="openAuctionMine">我的拍卖</a><br/>
        <span class="black">☆☆☆☆☆☆☆☆</span><br/>
        <span class="black">书卷</span>◎<a href="javascript:;" @click="loadAuction('material')">材料</a>◎<a href="javascript:;" @click="loadAuction('equip')">装备</a><br/>
        <a href="javascript:;" @click="loadAuction('mall')">商城</a>◎<a href="javascript:;" @click="loadAuction('pill')">丹药</a>◎<a href="javascript:;" @click="loadAuction('quest')">任务</a><br/>
        <a href="javascript:;" @click="loadAuction('farm')">农场</a>◎<a href="javascript:;" @click="loadAuction('box')">宝箱</a>◎<a href="javascript:;" @click="loadAuction('gem')">宝石</a><br/>
        <span class="black">☆☆☆☆☆☆☆☆</span><br/>
        <span class="black">全区玩家拍卖如下：</span><br/>
        <template v-if="auctionList.length">
          <div v-for="(a, i) in auctionList" :key="'al' + a.auction_id">
            {{ i + 1 }}.<a href="javascript:;" @click="auctionBuyTarget = auctionBuyTarget === a.auction_id ? 0 : a.auction_id">{{ a.name }}</a><span class="blue">x{{ a.count }}[{{ a.price }}两/个]</span><span class="black">[<a href="javascript:;" @click="viewPlayer(a.seller_id)">{{ a.seller }}</a>]</span>
            <template v-if="a.seller_id !== playerID"><a href="javascript:;" @click="auctionBuyTarget = auctionBuyTarget === a.auction_id ? 0 : a.auction_id">购买</a></template><br/>
            <template v-if="auctionBuyTarget === a.auction_id">
              <span class="red">你最多可购买{{ a.name }}x{{ a.count }}</span><br/>
              <span class="black">请输入你要购买多少{{ a.name }}呢？</span><br/>
              数量：<input v-model.number="auctionBuyCount" size="3" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
              <input type="button" value="购买" @click="doAuctionBuy(a, auctionBuyCount || a.count)" /><br/>
            </template>
          </div>
        </template>
        <template v-else><span class="black">暂无任何拍卖信息</span><br/></template>
        <br/>
        <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
      </template>
    </template>

    <!-- ==================== 货币流水 ==================== -->
    <template v-else-if="cur === 'wallet'">
      【银两流水】<br/>
      <div v-for="l in walletLogs" :key="'wl' + l.id">
        {{ l.reason }}：<span :class="l.amount > 0 ? 'green' : 'red'">{{ l.amount > 0 ? '+' + l.amount : l.amount }}</span>{{ currencyName(l.currency) }}
        (余{{ l.balance }})<em>[{{ fmtTime(l.created_at) }}]</em><br/>
      </div>
      <template v-if="!walletLogs.length"><em>暂无流水记录。</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('bank')">银行</a><br/>
    </template>

    <!-- ==================== 战报 ==================== -->
    <template v-else-if="cur === 'blogs'">
      【战斗记录】<br/>
      <div v-for="l in blogLogs" :key="'bl2' + l.id">
        {{ resultText(l.result) }}【{{ l.enemy_name }}】({{ l.type === 'boss' ? 'BOSS' : l.type === 'dungeon' ? '副本' : l.type === 'pvp' ? '比武' : l.type === 'tower' ? '通天塔' : '野怪' }}·{{ l.round }}回合)
        经验+{{ l.exp }} 银两+{{ l.money }}<em>[{{ fmtTime(l.created_at) }}]</em><br/>
      </div>
      <template v-if="!blogLogs.length"><em>暂无战斗记录，去西游世界降妖吧！</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 会员中心 / 充值 ==================== -->
    <template v-else-if="cur === 'vip'">
      <span class="red">会员中心</span><br/>
      <span class="black">我的VIP等级：</span><span class="red">VIP{{ vip.level }}级</span><span class="black">（充值可得，等级越高每日兑换越多）</span><br/>
      金豆：<span class="red">{{ g.beans }}</span>　VIP练级祝福：<span class="red">{{ g.vip }}分钟</span>(打怪经验1.5倍)<br/>
      <a href="javascript:;" @click="go('signin')">【每日福利】</a><br/>
      ----------------------<br/>
      <form @submit.prevent="doRecharge">
        充值码：<input v-model="rechargeCode" size="12" />
        <input type="submit" value="兑换" />
      </form>
      <em>演示充值码：XY666(+10金豆) / VIP666(+30分钟祝福) / SVIP0~20(设置会员等级)</em><br/>
      ----------------------<br/>
      <span class="red">说明：所有等级达到160的玩家每日可用银两换取【万能果】或者〖金豆〗（每日一次,vip玩家按照vip等级划分）</span><br/>
      <template v-if="!vip.lvl160"><span class="red">对不起!小仙家等级还未满160级啊？伤不起~~伤不起~~</span><br/></template>
      <template v-else>
        当前 {{ vip.level < 20 ? '可兑换' : '已达' }}：<span class="red">VIP{{ vip.level }}级【万能果】x{{ vipCur.cute }}{{ vipCur.beans ? '〖金豆〗x' + vipCur.beans : '' }}（{{ vipCur.silver }}银两）</span><br/>
        <template v-if="vip.exchanged"><span class="gray">今日已兑换过，明日再来</span><br/></template>
        <template v-else><a href="javascript:;" @click="vipExchange"><span class="red">【领取兑换】</span></a><br/></template>
      </template>
      ----------------------<br/>
      <span class="red">会员福利一览</span><span class="black">（按充值等级）</span><br/>
      <span v-for="d in vip.defs" :key="'vd' + d.lv">
        <span :class="d.lv === vip.level ? 'red' : 'blue'">VIP{{ d.lv }}级：【万能果】x{{ d.cute }}{{ d.beans ? '〖金豆〗x' + d.beans : '' }}（{{ d.silver }}银两）</span><br/>
      </span>
      ----------------------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
    </template>

    <!-- ==================== 私聊 ==================== -->
    <template v-else-if="cur === 'pm' && pmCur">
      【私聊】<a href="javascript:;" @click="viewPlayer(pmCur.player.player_id)">{{ pmCur.player.name }}</a>({{ pmCur.player.level }}级·{{ pmCur.player.sect_name }})<br/>
      -----------<br/>
      <template v-if="pmCur.msgs.length">
        <div v-for="(m, i) in pmCur.msgs" :key="'pm' + i">
          <span class="black">{{ m.player_id === g.id ? pmCur.player.name + '说：' : '你说：' }}</span>{{ m.content }}<br/>
        </div>
      </template>
      <template v-else><em>暂无聊天记录，发条消息打个招呼吧。</em><br/></template>
      -----------<br/>
      <form @submit.prevent="sendPm">
        <input v-model="pmText" size="14" placeholder="输入私聊内容" />
        <input type="submit" value="发送" />
      </form>
      <a href="javascript:;" @click="addFriendCur">加为好友</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('friends')">好友</a><br/>
    </template>

    <!-- ==================== 攻略指引 ==================== -->
    <template v-else-if="cur === 'guide'">
      【攻略指引】<br/>
      -----------<br/>
      <span class="black">【练级】</span>西游世界打怪得经验银两，VIP祝福经验1.5倍。<br/>
      <span class="black">【恢复】</span>客栈住店（等级×10银两）可回满气血法力。<br/>
      <span class="black">【装备】</span>行囊点击装备穿戴，武器店购买，可升星镶宝石。<br/>
      <span class="black">【技能】</span>技能页花费银两学习本门派技能，战斗更轻松。<br/>
      <span class="black">【宠物】</span>战斗中捕捉野生怪物，参战宠物共享经验。<br/>
      <span class="black">【银两】</span>银行存取防盗；摆摊挂售多余物品赚银两。<br/>
      <span class="black">【修炼】</span>开启修炼打怪，经验入修炼池转化技能点。<br/>
      <span class="black">【头衔】</span>头衔页激活佩戴，提升战斗属性。<br/>
      <span class="black">【住宅】</span>购买住宅放置家具，享受属性加成。<br/>
      <span class="black">【签到】</span>每日签到，连续天数越多奖励越丰厚。<br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">返回首页</a>.<a href="javascript:;" @click="go('rank')">排行榜</a><br/>
    </template>

    <!-- ==================== 通天塔（挑战） ==================== -->
    <template v-else-if="cur === 'tower'">
      【通天塔】<br/>
      <template v-if="tower.floor > 0">当前层数：第{{ tower.floor }}层<br/></template>
      <template v-else><em>你还没有登上通天塔，从第一层开始挑战吧。</em><br/></template>
      历史最高：第{{ tower.best }}层<br/>
      <em>规则：逐层挑战守卫，胜利上一层，层数越高奖励越丰厚；战败或逃跑则被送回塔底重头再来。</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="towerStart">[挑战第{{ tower.floor + 1 }}层]</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('rank')">排行</a><br/>
    </template>

    <!-- ==================== 擂台（天下第一武道大会） ==================== -->
    <template v-else-if="cur === 'arena'">
      【天下第一武道大会】<br/>
      今日已比武：{{ arena.me.today }}/{{ arena.me.limit }}次<br/>
      我的胜场：{{ arena.me.wins }}场<template v-if="arena.me.rank > 0">，当前排名第{{ arena.me.rank }}位</template><br/>
      -----------<br/>
      【比武排行（前十）】<br/>
      <template v-if="arena.rank.length">
        <div v-for="r in arena.rank" :key="'ar' + r.rank">
          第{{ r.rank }}名：<a href="javascript:;" @click="viewPlayer(r.player_id)">{{ r.name }}</a>({{ r.level }}级·{{ r.wins }}胜)
          <template v-if="r.player_id !== g.id"><a v-if="arena.me.today < arena.me.limit" href="javascript:;" @click="arenaFight(r)">[比武]</a></template><br/>
        </div>
      </template>
      <template v-else><em>暂无人上榜，快去比武抢第一吧！</em><br/></template>
      -----------<br/>
      <em>规则：每日可发起5次比武；战胜可夺取对方一成随身银两（上限1000），但会增加1点恶名。</em><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('rank')">排行</a><br/>
    </template>

    <!-- ==================== 娱乐（疯狂摇一摇） ==================== -->
    <template v-else-if="cur === 'fun'">
      【疯狂摇一摇】银两:{{ g.money }} 金豆:{{ g.beans }}<br/>
      <template v-if="funRoll">
        <span class="red">{{ funRoll[0] }}·{{ funRoll[1] }}·{{ funRoll[2] }}</span><br/>
      </template>
      <em>三个相同赢8倍，两个相同赢2倍！</em><br/>
      -----------<br/>
      【银两场】<br/>
      <a href="javascript:;" @click="doRoll('m1')">黄金场(100银两/次)</a><br/>
      <a href="javascript:;" @click="doRoll('m2')">铂金场(1000银两/次)</a><br/>
      【金豆场】<br/>
      <a href="javascript:;" @click="doRoll('b1')">黄金场(10金豆/次)</a><br/>
      <a href="javascript:;" @click="doRoll('b2')">铂金场(100金豆/次)</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== 腾云传送 ==================== -->
    <template v-else-if="cur === 'teyun'">
      【腾云驾雾】<br/>
      腾云符：<span class="red">{{ teyun.fu }}</span>张<br/>
      <em>注：部分区域无法直接腾云，请寻找地图传送NPC！</em><br/>
      <template v-if="teyun.list.length">
        <span v-for="(g, gi) in teyunGroup" :key="'tyg' + gi">
          <span class="black">【{{ g.cat }}】</span><br/>
          <span v-for="(t, tj) in g.items" :key="'ty' + gi + '-' + tj"><a href="javascript:;" @click="teyunGo(t)">{{ t.name }}</a><span class="black">◎</span></span>
          <br/>
        </span>
      </template>
      <template v-else><em>没有可传送的地点。</em><br/></template>
      <em>每次腾云消耗1张腾云符（杂货店有售）。</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="npcShop('grocery')">杂货店</a><br/>
    </template>

    <!-- ==================== 系统 ==================== -->
    <template v-else-if="cur === 'sys'">
      【系统】<br/>
      服务器：{{ serverName }}<br/>
      角色：{{ g.name }}({{ g.sect_name }}·{{ g.level }}级)<br/>
      <em>如遇游戏问题或BUG，请联系管理员处理。</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="exitToServer">[退出并重选服务器]</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('guide')">攻略</a><br/>
    </template>

    <!-- ==================== 玩家资料（复刻 xy093.php） ==================== -->
    <template v-else-if="cur === 'playerview' && pv">
      <template v-if="!pv.is_me">
        <span class="red">恶名：{{ emzName(pv.emz) }}</span><a href="javascript:;" @click="fightPlayer(pv)">PK对方</a><br/>
      </template>
      <template v-else><span class="red">恶名：{{ emzName(pv.emz) }}</span><br/></template>
      <span class="black">头衔：<template v-if="pv.title">{{ pv.title }}</template><template v-else>暂无</template></span><br/>
      <span class="black">昵称：{{ pv.name }}</span><br/>
      <span class="black">性别：{{ pv.sex === 2 ? '女' : '男' }}</span><br/>
      <span class="black">配偶：<template v-if="pv.spouse">{{ pv.spouse }}</template><template v-else>暂无</template></span><br/>
      <span class="black">住宅：<template v-if="pv.house">{{ pv.house }}</template><template v-else>暂无</template></span><br/>
      <span class="black">国家：<template v-if="pv.gang">{{ pv.gang }}</template><template v-else>无</template></span><br/>
      <span class="black">门派：<template v-if="pv.sect_name">{{ pv.sect_name }}</template><template v-else>无门派</template></span><br/>
      <template v-if="!pv.is_me">
        <a href="javascript:;" @click="giveMoney(pv)">【赠银】</a>◎<a href="javascript:;" @click="giveItem(pv)">【赠物】</a><br/>
        <!-- 赠银表单（复刻 xy537） -->
        <template v-if="giveMode === 'money'">
          -----------<br/>
          <span class="black">请输入你要赠送给{{ pv.name }}({{ pv.player_id }})的银两:</span><br/>
          <input v-model.trim="giveAmount" size="16" placeholder="请输入你要赠送的银两" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
          <input type="submit" value="赠送" @click="doGiveMoney(pv)" /><br/>
          <a href="javascript:;" @click="giveMode = ''">返回上级</a><br/>
        </template>
        <!-- 赠物流程（复刻 xy538：选物品→数量→赠送） -->
        <template v-else-if="giveMode === 'item'">
          -----------<br/>
          <span class="black">请选择你要赠送给{{ pv.name }}的物品：</span><br/>
          <div v-for="b in bagList" :key="'gv' + b.id">
            <template v-if="giveBagId === b.id">
              <span class="red">{{ b.name }}</span>x{{ b.count }}<br/>
              <span class="black">请输入你要赠送多少{{ b.name }}呢？</span><br/>
              <input v-model.trim="giveCount" size="8" inputmode="numeric" onkeyup="this.value=this.value.replace(/\D/g,'')" /><br/>
              <input type="submit" value="赠送" @click="doGiveItem(pv)" /><br/>
            </template>
            <template v-else><a href="javascript:;" @click="pickGiveItem(b)">{{ b.name }}</a>x{{ b.count }}<br/></template>
          </div>
          <template v-if="!bagList.length"><em>行囊空空如也。</em><br/></template>
          <a href="javascript:;" @click="giveMode = ''">返回上级</a><br/>
        </template>
        <span class="black">☆☆☆☆☆☆☆☆</span><br/>
        <a href="javascript:;" @click="openStallOf(pv.player_id)">挂售</a>◎<a href="javascript:;" @click="openPm(pv.player_id)">私聊</a>◎<a href="javascript:;" @click="teamInvite(pv)">组队</a><br/>
        <a href="javascript:;" @click="blackPlayer(pv.player_id)">拉黑</a>◎<a href="javascript:;" @click="addFriendByID(pv.player_id)">加友</a><br/>
        <template v-if="gang.my_gang.gang_id && (gang.my_gang.role === 1 || gang.my_gang.role === 2)"><a href="javascript:;" @click="inviteJoinGang(pv)">邀请入国</a><br/></template>
        <template v-if="gang.my_gang.gang_id && (gang.my_gang.role === 1 || gang.my_gang.role === 2)"><a href="javascript:;" @click="inviteVisitHouse(pv)">邀请参观住宅</a><br/></template>
      </template>
      <a href="javascript:;" @click="backPv">返回游戏</a><br/>
    </template>

    <!-- ==================== 国战 ==================== -->
    <template v-else-if="cur === 'gz' && gz">
      【国战】<br/>
      战场位置:【{{ gz.zc_name }}】<br/>
      国战计时：<template v-if="gz.in_war"><span class="red">{{ fmtSec(gz.left_sec) }}</span></template><template v-else><em>{{ gz.zc_id === 6 ? '今日休整' : '未开战（每整点后30分钟开战）' }}</em></template><br/>
      -----------<br/>
      <template v-if="gz.war.def_gang_id > 0">
        【国家状态】--{{ gz.my_side === 'def' ? '防守方' : '进攻方' }}--<br/>
        防守国家：【{{ gz.war.def_gang_name }}】<br/>
        <template v-if="gz.war.holder_gang_id > 0">
          国家权杖：【{{ gz.war.holder_gang_name }}】防守中<template v-if="gz.war.hold_left > 0">[{{ fmtSec(gz.war.hold_left) }}]</template><br/>
        </template>
        <template v-if="gz.war.neijian">
          <span class="red">【{{ gz.war.neijian }}】混入了国战队伍！</span><a v-if="gz.my_side === 'def'" href="javascript:;" @click="gzAction('gz/neijian')">[击杀内奸]</a><br/>
        </template>
        <template v-if="gz.in_war && gz.my_side === 'atk'">
          <a href="javascript:;" @click="gzAction('gz/rod')">[进攻！夺下权杖]</a><br/>
        </template>
      </template>
      <template v-else>
        <em>今日还没有国家报名防守</em><br/>
        <a v-if="gz.my_gang_id > 0 && gz.my_role === 1 && gz.zc_id !== 6" href="javascript:;" @click="gzSignup">[君主报名今日国战]</a><br/>
      </template>
      <template v-if="gz.my_gang_id === 0"><em>你还没有加入国家（帮派），先去帮派页加入吧。</em><br/></template>
      我的国家：{{ gz.my_gang_name || '无' }} 国家积分：{{ gz.my_gang_score }}<br/>
      我的个人积分：{{ gz.my_score }}<br/>
      -----------<br/>
      【国家积分榜】<br/>
      <template v-if="gz.gang_scores.length">
        <div v-for="(s, i) in gz.gang_scores" :key="'gs' + i">{{ i + 1 }}.{{ s.gang_name }}——{{ s.total }}分<br/></div>
      </template>
      <template v-else><em>暂无积分</em><br/></template>
      【个人积分榜】<br/>
      <template v-if="gz.player_scores.length">
        <div v-for="(s, i) in gz.player_scores" :key="'ps' + i">{{ i + 1 }}.<a href="javascript:;" @click="viewPlayer(s.player_id)">{{ s.name }}</a>({{ s.gang_name }})——{{ s.total }}分<br/></div>
      </template>
      <template v-else><em>暂无积分</em><br/></template>
      <em>规则：每整点后30分钟开战；防守方守住权杖每5分钟国家积分+10；进攻方战胜神兽守卫夺下权杖+个人积分5；击杀国战内奸国家+1个人+1。</em><br/>
      -----------<br/>
      <a href="javascript:;" @click="refreshGz">刷新</a>.<a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('gang')">帮派</a><br/>
    </template>

    <!-- ==================== 队伍 ==================== -->
    <template v-else-if="cur === 'team' && team">
      【队伍】<br/>
      <template v-if="team.team">
        当前队伍（{{ team.team.count }}/{{ team.team.max }}）<br/>
        <div v-for="m in team.team.members" :key="'tm' + m.player_id">
          <template v-if="m.is_leader">队长：</template>
          <template v-else>队员：</template>
          <a v-if="!m.is_me" href="javascript:;" @click="viewPlayer(m.player_id)">{{ m.name }}</a>
          <span v-else class="black">{{ m.name }}</span>({{ m.level }}级)
          <template v-if="team.team.is_leader && !m.is_leader"><a href="javascript:;" @click="teamKick(m)">[踢出]</a></template><br/>
        </div>
        <a v-if="team.team.is_leader" href="javascript:;" @click="teamLeave(true)">[解散队伍]</a>
        <a v-else href="javascript:;" @click="teamLeave(false)">[离开队伍]</a><br/>
        <em>队长可在玩家资料页点[组队]邀请好友（上限{{ team.team.max }}人）。</em><br/>
      </template>
      <template v-else>
        我的队伍：当前还未有队伍<br/>
        <a href="javascript:;" @click="teamCreate">[创建队伍]</a><br/>
        <em>创建后可在玩家资料页点[组队]邀请好友。</em><br/>
      </template>
      <template v-if="team.invites.length">
        -----------<br/>
        【组队邀请】<br/>
        <div v-for="iv in team.invites" :key="'ti' + iv.id">
          【<a href="javascript:;" @click="viewPlayer(iv.from_id)">{{ iv.from_name }}</a>】邀请你加入队伍
          <a href="javascript:;" @click="teamAgree(iv)">[同意]</a>
          <a href="javascript:;" @click="teamRefuse(iv)">[拒绝]</a><br/>
        </div>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">返回游戏</a><br/>
    </template>

    <!-- toast -->
    <div v-if="tipMsg" class="xy-tip">{{ tipMsg }}</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'XiyouGame',
  data() {
    return {
      cur: 'boot',
      servers: [
        { id: '电信一区', name: '长安', rec: 1 },
        { id: '网通一区', name: '蓬莱' },
        { id: '双线一区', name: '花果山' },
      ],
      lastServer: localStorage.getItem('hxxy_server') || '',
      serverName: '',
      maintMsg: '',
      g: { name: '', level: 1, hp: 0, max_hp: 0, mp: 0, max_mp: 0, money: 0, bank: 0, beans: 0, vip: 0, exp: 0, exp_need: 0, node_name: '', sect_name: '', fighting_pet: null, xiulian_switch: 0, xiulian_exp: 0, xiulian_cap: 0 },
      sects: [],
      stories: [
        { pic: '/static/hxxy/story/story1.jpg', text: '大唐年间,妖魔四起,无数百姓处于水生火热当中!！' },
        { pic: '/static/hxxy/story/story2.jpg', text: '十万天兵天将与众妖魔战斗的难解难分！' },
        { pic: '/static/hxxy/story/story3.jpg', text: '不断有英雄侠女从亿万百姓中脱颖而出，踏上降妖除魔的征程！' }
      ],
      cf: { name: '', sex: 0, sect: 0, step: 'sex', story: 0, sectName: '', sectBonus: '', sectLong: '', sectPic: '' },
      node: {},
      enemies: [],
      mapNpcs: [],
      mapGrid: { cur: '', rows: [] },
      mapImgOk: false,   // 当前节点地图图片是否加载成功
      mapImgErr: false,  // 当前节点地图图片是否缺失（原版 xy008：缺图时给出提示）
      txImgOk: false,    // 当前头衔图片是否加载成功
      txImgErr: false,   // 当前头衔图片是否未制作（原版 txdt.php：缺图时给出提示）
      mapSize: 11,
      homeMsgs: [],
      nearby: [],
      nearbyExpanded: false,
      notices: [],
      homeInvites: [],
      homeGangInvites: [],
      homeHouseInvites: [],
      homeMarry: null,
      visitHouse: null,
      pmCur: null,
      pmText: '',
      npcCur: null,
      bt: null,
      battleItems: [],
      quickSlots: [],
      quickSetSlot: 0,
      quickPickTab: 'skill',
      skLoaded: false,
      bagLoaded: false,
      showQuickSet: false,
      at: null,
      equippedIDs: [],
      bagList: [],
      bagCap: 0,
      bagUsed: 0,
      storeMode: 0,
      bagDetail: null,
      useCounts: {},
      gems: [],
      sk: { mine: [], store: [] },
      petList: [],
      petRenameId: 0,
      petRenameName: '',
      shop: { goods: [], pets: null, used: 0, cap: 0 },
      shopKind: 'medicine',
      shopPage: 'list',
      shopItem: null,
      shopPet: null,
      shopBuyCount: '',
      shopMsg: '',
      shopTabs: [
        { k: 'medicine', n: '药店' },
        { k: 'weapon', n: '武器' },
        { k: 'armor', n: '防具' },
        { k: 'jewel', n: '首饰' },
        { k: 'grocery', n: '杂货' },
        { k: 'pet', n: '宠物店' },
      ],
      bankAmount: 0,
      bankView: 'main',
      bankMsg: '',
      sign: { month_cn: '', count: 0, today_signed: false, tiers: [] },
      signDetail: false,
      wf: { gift: {}, active: { score: 0, tasks: [], tiers: [], total: 100 }, vip: {}, nobles: [] },
      wfTab: 'gift',
      qst: { available: [], active: [], done: [] },
      qstTab: 1,
      act: null,
      homeFlags: { today_signed: false, quest_ready: 0, act_ready: false },
      dungeons: [],
      bossList: [],
     cult: { switch: 0, exp: 0, sw: 0, tracks: [] },
      cultView: '',
      titles: { list: [], mine: [], worn: 0, page: 1, total_pages: 1 },
      titleView: '',
      titleDetail: {},
      titleTab: 'list',
      rankType: 'level',
      rankList: [],
      rankTabs: [{ k: 'level', n: '等级榜' }, { k: 'money', n: '银两榜' }, { k: 'pets', n: '宠物榜' }],
      chatList: [],
      chatInput: '',
      frd: { friends: [], blacks: [] },
      frdTab: 1,
      gang: { my_gang: {}, gangs: [] },
      gangSub: '',
      gangKickTarget: null,
      gangNameInput: '',
      gangDonateAmount: 0,
      gangMall: { tabs: [], level: 0, contribution: 0 },
      gangMallTab: 1,
      gangAppointRole: 0,
      roleNames: { 0: '成员', 1: '君主', 2: '辅助大臣', 3: '军机大臣', 4: '财政大臣', 5: '工部大臣', 6: '外交大臣', 7: '军团长' },
      marriage: { status: 0 },
      marryName: '',
      furniture: [],
      houseCatalog: [
        { fid: 1, name: '兵器架', bonus: 'atk', val: 200, price: 5000 },
        { fid: 2, name: '练功石', bonus: 'atk', val: 500, price: 12000 },
        { fid: 3, name: '屏风', bonus: 'def', val: 200, price: 5000 },
        { fid: 4, name: '护院石狮', bonus: 'def', val: 500, price: 12000 },
        { fid: 5, name: '檀香案', bonus: 'mg', val: 300, price: 8000 },
        { fid: 6, name: '龙纹柱', bonus: 'mg', val: 600, price: 15000 },
        { fid: 7, name: '雕花木床', bonus: 'hp', val: 500, price: 5000 },
        { fid: 8, name: '聚灵阵盘', bonus: 'hp', val: 1200, price: 15000 },
      ],
      // 挂售/拍卖上架表单（复刻 gssjwp01/pmsjwp01）
      stallItem: null, stallCount: '', stallPrice: '',
      aucItem: null, aucCount: '', aucPrice: '',
      // 我的挂售（复刻 xy225 分类 → xy219 列表）
      stallPage: 'cat', stallKind: 'item',
      stallMine: { stalls: [], used: 0, capacity: 10 },
      stallDetailId: 0, stallCancelTarget: 0, stallCancelCount: '',
      // 他人挂售（复刻 xy222）与全区拍卖（复刻 xy489/499）
      stallOf: null, stallBuyTarget: 0, stallBuyCount: '',
      auctionView: 'list', auctionTab: 'scroll', auctionList: [], auctionMine: [],
      auctionBuyTarget: 0, auctionBuyCount: '',
      // 赠银/赠物（复刻 xy537/538）
      giveMode: '', giveAmount: '', giveBagId: 0, giveCount: '',
      walletLogs: [],
      blogLogs: [],
      rechargeCode: '',
      vip: { level: 0, lvl160: false, exchanged: false, defs: [], wanneng_id: 0 },
      tower: { floor: 0, best: 0 },
      arena: { rank: [], me: { rank: 0, wins: 0, today: 0, limit: 5 } },
      funRoll: null,
      teyun: { fu: 0, list: [] },
      pv: null,
      pvFrom: 'home',
      gz: null,
      team: null,
      tipMsg: '',
      tipTimer: null,
    }
  },
  computed: {
    playerID() { return this.g.id || 0 },
    // 当前节点地图图片（原版 pic/dtpic/{dtx}-{dty}.jpg；素材只覆盖部分节点）
    mapImgSrc() {
      const x = this.g.map_x, y = this.g.map_y
      if (x === undefined || x === null || y === undefined || y === null) return ''
      return '/static/hxxy/dtpic/' + x + '-' + y + '.jpg'
    },
    // 当前应展示的头衔图片 id（原版 xy011 状态页=佩戴头衔；xy478 称号详情=查看的头衔）
    curTitleId() {
      if (this.cur === 'attrs') return this.g.title_id || 0
      if (this.cur === 'titles' && this.titleView === 'detail') return (this.titleDetail && this.titleDetail.title_id) || 0
      return 0
    },
    // 附近玩家展开（复刻原版 fjwj.php：默认最多显示3个，更多....展开）
    nearbyShow() {
      return this.nearbyExpanded ? this.nearby : this.nearby.slice(0, 3)
    },
    // 国家商城当前页签（1~10级）
    gangMallTabObj() {
      return this.gangMall.tabs.find(t => t.level === this.gangMallTab) || null
    },
    // 地图出口（复刻原版"请选择出口"：上/下/左/右，walk=走路 jump=传送出口）
    mapExits() {
      const n = this.node || {}
      return [
        { dir: 'up', label: '上', walk: n.up, jump: n.up_jump },
        { dir: 'down', label: '下', walk: n.down, jump: n.down_jump },
        { dir: 'left', label: '左', walk: n.left, jump: n.left_jump },
        { dir: 'right', label: '右', walk: n.right, jump: n.right_jump }
      ].filter(d => d.walk || d.jump)
    },
    // 查看地图窗口（复刻 MapViewer.show：以当前位置为中心裁剪 mapSize×mapSize）
    mapWin() {
      const rows = this.mapGrid.rows || []
      const cur = this.mapGrid.cur
      let cx = -1, cy = -1
      rows.forEach((row, y) => row.forEach((c, x) => { if (c && c.dtxy === cur) { cx = x; cy = y } }))
      if (cx < 0) return { rows: [] }
      const s = this.mapSize
      const half = s & 1 ? (s - 1) / 2 : (s - 2) / 2
      const y0 = Math.max(0, cy - half), y1 = Math.min(rows.length - 1, cy + half)
      const x0 = Math.max(0, cx - half), x1 = Math.min((rows[0] || []).length - 1, cx + half)
      const out = []
      for (let y = y0; y <= y1; y++) out.push(rows[y].slice(x0, x1 + 1))
      return { rows: out }
    },
    // 腾云目的地按分类分组（复刻原版 xy476.php 分类）
    teyunGroup() {
      const catOrder = ['门派区域', '主城区域', '野外区域', '副本区域']
      const groups = []
      catOrder.forEach(cat => {
        const items = (this.teyun.list || []).filter(t => t.cat === cat)
        if (items.length) groups.push({ cat, items })
      })
      return groups
    },
    // 当前选中的贵族（wfTab：noble1~noble4）
    nobleCur() {
      const idx = parseInt((this.wfTab || '').replace('noble', ''), 10) - 1
      return (idx >= 0 && this.wf.nobles) ? (this.wf.nobles[idx] || {}) : {}
    },
    nobleCurOwned() { return !!this.nobleCur.owned },
    nobleCurClaimed() { return !!this.nobleCur.claimed },
    // 当前 VIP 等级的每日兑换信息
    vipCur() {
      const d = (this.vip.defs || []).find(x => x.lv === this.vip.level)
      return d || { cute: 1, beans: 0, silver: '2000万' }
    },
    // 首页角色 VIP 会员等级
    vipLv() {
      return this.g.vip_lv || 0
    },
    // 西游报时（复刻原版页脚报时，9:27 格式）
    xyNow() {
      const d = new Date()
      return d.getHours() + ':' + String(d.getMinutes()).padStart(2, '0')
    },
    battleResultText() {
      if (!this.bt) return ''
      const n = this.bt.type === 'pvp' ? '比武' : this.bt.type === 'tower' ? '通天塔' : '战斗'
      if (this.bt.status === 2) return this.bt.type === 'dungeon' && (this.bt.enemy.floor || 0) >= 10 ? '副本通关！' : n + '胜利！'
      if (this.bt.status === 3) return n + '失败……'
      if (this.bt.status === 4) return '你成功逃跑了！'
      return n + '结束'
    },
    battleRewards() {
      // 战报最后几行包含 经验/银两 收益，直接展示末尾摘要
      if (!this.bt || this.bt.in_battle) return null
      const logs = this.bt.log || []
      const exp = logs.find(l => l.indexOf('获得经验') >= 0)
      const money = logs.find(l => l.indexOf('获得银两') >= 0)
      if (!exp && !money) return null
      const p1 = exp ? exp.match(/获得经验 (\d+)/) : null
      const p2 = money ? money.match(/获得银两 (\d+)/) : null
      return { exp: p1 ? p1[1] : 0, money: p2 ? p2[1] : 0 }
    },
    wornTitleName() {
      const t = this.titles.mine.find(t => t.title_id === this.titles.worn)
      return t ? t.name : '无'
    },
    furnitureTotal() {
      const t = { hp: 0, atk: 0, def: 0, mg: 0 }
      this.furniture.forEach(f => { if (t[f.bonus] !== undefined) t[f.bonus] += f.val })
      const parts = []
      if (t.hp) parts.push('气血+' + t.hp)
      if (t.atk) parts.push('攻击+' + t.atk)
      if (t.def) parts.push('防御+' + t.def)
      if (t.mg) parts.push('魔攻+' + t.mg)
      return parts.join(' ') || '无'
    },
    rankValName() {
      return { level: '等级', money: '银两', pets: '宠物等级' }[this.rankType] || '值'
    },
  },
  watch: {
    // 换节点后重置地图图片状态，等待新图加载（避免上一次的"缺图"状态粘住）
    mapImgSrc() {
      this.mapImgOk = false
      this.mapImgErr = false
    },
    // 换头衔后同理重置
    curTitleId() {
      this.txImgOk = false
      this.txImgErr = false
    },
  },
  mounted() {
    // 入口先选服务器（复刻 WAP 版进服流程）；已选过服务器则刷新直接恢复上次页面
    const lastServer = localStorage.getItem('hxxy_server')
    const lastCur = sessionStorage.getItem('hxxy_cur')
    if (lastServer) {
      this.serverName = lastServer
      this.cur = 'boot'
      // 需要本地上下文的视图（战斗/资料/私聊）刷新后回首页，其余恢复原页面
      const restorable = lastCur && !['server', 'boot', 'create', 'maint', 'battle', 'playerview', 'pm'].includes(lastCur)
      this.boot(restorable ? lastCur : 'home')
    } else {
      this.cur = 'server'
    }
    // 拦截后退快捷键：不退出游戏，回退到游戏首页
    history.pushState({ __xyGuard: true }, '')
    this._onBack = () => {
      if (location.hash.split('?')[0].includes('/games/hxxy')) {
        this.go('home')
      }
      history.pushState({ __xyGuard: true }, '')
    }
    window.addEventListener('popstate', this._onBack)
  },
  beforeDestroy() {
    if (this._onBack) window.removeEventListener('popstate', this._onBack)
  },
  methods: {
    // ---------- 基础 ----------
    tip(m) {
      this.tipMsg = m
      if (this.tipTimer) clearTimeout(this.tipTimer)
      this.tipTimer = setTimeout(() => { this.tipMsg = '' }, 2200)
    },
    fmtTime(t) { return (t || '').substring(5, 16) },
    slotName(c) { return { 1: '法宝', 2: '坐骑', 3: '手持', 4: '身穿', 5: '头戴', 6: '脚穿', 7: '佩戴', 8: '首饰', 9: '婚戒', 10: '婚链', 11: '披风' }[c] || '装备' },
    // 银两格式化（复刻原版 wp/warehouse.php：X亿X万X两）
    yl(v) {
      if (!v || v <= 0) return '0两'
      const s = String(v)
      if (s.length >= 9) {
        const y = parseInt(s.slice(0, s.length - 8), 10)
        const w = parseInt(s.slice(s.length - 8, s.length - 4), 10)
        const l = parseInt(s.slice(s.length - 4), 10)
        let out = ''
        if (y > 0) out += y + '亿'
        if (w > 0) out += w + '万'
        if (l > 0) out += l
        return out + '两'
      }
      if (s.length >= 5) {
        const w = parseInt(s.slice(0, s.length - 4), 10)
        const l = parseInt(s.slice(s.length - 4), 10)
        let out = ''
        if (w > 0) out += w + '万'
        if (l > 0) out += l
        return out + '两'
      }
      return v + '两'
    },
    // 金额不带"两"字（商城文案"X亿银两"用）
    ylNum(v) {
      return this.yl(v).replace(/两$/, '')
    },
    itemCatName(c) { return { 1: '杂货', 2: '宝石', 3: '任务', 4: '经验书', 5: '药品', 8: '特殊' }[c] || '物品' },
    bonusName(b) { return { hp: '气血', atk: '攻击', def: '防御', mg: '魔攻' }[b] || b },
    currencyName(c) { return c === 'beans' ? '金豆' : c === 'bank' ? '存款' : '银两' },
    resultText(r) { return { 2: '战胜', 3: '战败', 4: '逃跑' }[r] || '战斗' },
    // 头衔图片路径（复刻原版 wp/txdt.php：特定 id 段为 gif，其余 png）
    titleImg(id) {
      const gif = (id >= 673 && id <= 679) || (id >= 914 && id <= 921) || (id >= 954 && id <= 964) || (id >= 1034 && id <= 1045)
      return '/static/hxxy/txpic/' + id + '.' + (gif ? 'gif' : 'png')
    },
    upgradeCost(b) {
      const star = (b.extra && b.extra.star) || 0
      return (b.level || 1) * 500 + star * 1000
    },
    canUseMany(b) { return b.count > 1 && b.category === 5 },
    go(v) {
      this.cur = v
      if (v !== 'server') sessionStorage.setItem('hxxy_cur', v) // 记录当前页，刷新后恢复
      window.scrollTo(0, 0)
      if (v === 'home') { this.refreshPlayer(); this.loadHome(); this.loadState() }
      if (v === 'map') this.loadState()
      if (v === 'attrs') this.loadAttrs()
      if (v === 'bag') this.loadBag()
      if (v === 'skills') this.loadSkills()
      if (v === 'pets') this.loadPets()
      if (v === 'shop') this.loadShop(this.shopKind)
      if (v === 'quests') this.loadQuests()
      if (v === 'activities') this.loadActivities()
      if (v === 'dungeons') this.loadDungeons()
      if (v === 'bosses') this.loadBosses()
      if (v === 'cultivate') { this.cultView = ''; this.loadCult() }
      if (v === 'titles') { this.titleView = ''; this.loadTitles() }
      if (v === 'rank') this.loadRank(this.rankType)
      if (v === 'chat') this.loadChat()
      if (v === 'friends') this.loadFriends()
      if (v === 'gang') this.loadGang()
      if (v === 'mapview') this.loadMapGrid()
      if (v === 'marriage') this.loadMarriage()
      if (v === 'house') this.loadHouse()
      if (v === 'stalls') { this.stallPage = 'cat'; this.stallCancelTarget = 0; this.stallDetailId = 0; this.stallItem = null; this.aucItem = null }
      if (v === 'auction') this.openAuction()
      if (v === 'wallet') this.loadWallet()
      if (v === 'signin') this.loadWelfare()
      if (v === 'vip') this.loadVip()
      if (v === 'blogs') this.loadBlogs()
      if (v === 'tower') this.loadTower()
      if (v === 'arena') this.loadArena()
      if (v === 'teyun') this.loadTeyun()
      if (v === 'gz') this.loadGz()
      if (v === 'team') this.loadTeam()
      if (v === 'battle') this.refreshPlayer()
    },
    async refreshPlayer() {
      const r = await api.get('/games/hxxy/status')
      if (r.code === 0 && r.data.player) this.g = r.data.player
    },
    // ---------- 服务器 ----------
    pickServer(s) {
      this.serverName = s.id + '·' + s.name
      localStorage.setItem('hxxy_server', this.serverName)
      this.cur = 'boot'
      this.boot()
    },
    async boot(target) {
      const r = await api.get('/games/hxxy/status')
      if (r.code !== 0) {
        // 服务器维护中 → 显示维护公告页
        if ((r.msg || '').indexOf('维护') >= 0) {
          this.maintMsg = r.msg
          this.cur = 'maint'
          return
        }
        this.tip(r.msg || '加载失败')
        return
      }
      if (r.data.has_player) {
        this.g = r.data.player
        this.go(target || 'home')
        this.restoreBattle() // 刷新后恢复遗留战斗（修复"你正在战斗中"软锁）
      } else {
        this.sects = r.data.sects || []
        this.cur = 'create'
      }
    },
    retryBoot() {
      this.cur = 'boot'
      this.boot()
    },
    async doCreate() {
      if (!this.cf.name.trim()) { this.tip('请填写角色名'); return }
      const r = await api.post('/games/hxxy/create', { name: this.cf.name.trim(), sex: this.cf.sex, sect: this.cf.sect })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.go('home')
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 建角分步流程（复刻原版） ----------
    pickSex(sex) {
      this.cf.sex = sex
      this.cf.story = 0
      this.cf.step = 'story'
    },
    storyNext() {
      if (this.cf.story >= this.stories.length - 1) {
        this.cf.step = 'sect'
      } else {
        this.cf.story++
      }
    },
    pickSect(s) {
      if (s.sex === 1 && this.cf.sex !== 1) { this.tip('普陀山只收男弟子！'); return }
      if (s.sex === 2 && this.cf.sex !== 2) { this.tip('月宫只收女弟子！'); return }
      this.cf.sect = s.id
      this.cf.sectName = s.name
      this.cf.sectBonus = s.bonus
      this.cf.sectLong = s.long
      this.cf.sectPic = s.pic
      this.cf.step = 'sectIntro'
    },
    // ---------- 地图 ----------
    async loadState() {
      const r = await api.get('/games/hxxy/state')
      if (r.code === 0) {
        this.g = r.data.player
        this.node = r.data.node
        this.enemies = r.data.enemies || []
        this.mapNpcs = r.data.npcs || []
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 首页消息区/私聊（复刻原版 xy002.php） ----------
    async loadHome() {
      const r = await api.get('/games/hxxy/home')
      if (r.code === 0) {
        this.homeMsgs = r.data.msgs || []
        this.nearby = r.data.nearby || []
        this.notices = r.data.notices || []
        this.homeInvites = r.data.team_invites || []
        this.homeGangInvites = r.data.gang_invites || []
        this.homeHouseInvites = r.data.house_invites || []
        this.homeMarry = r.data.marriage_invite || null
        // 导航状态标色（复刻原版：未签到/可提交任务/可领活动 时导航红字）
        this.homeFlags.today_signed = !!r.data.today_signed
        this.homeFlags.quest_ready = r.data.quest_ready || 0
        this.loadActFlags()
      }
    },
    // 活动可领状态（首页导航标红用）
    async loadActFlags() {
      const r = await api.get('/games/hxxy/activities')
      if (r.code === 0 && r.data) {
        const d = r.data
        let ready = false
        if (d.login7 && !d.login7.claimed) ready = true
        if (d.daily && (d.daily.tiers || []).some(t => t.can && !t.claimed)) ready = true
        this.homeFlags.act_ready = ready
      }
    },
    // ---------- 活动中心（复刻原版 xy404） ----------
    async loadActivities() {
      const r = await api.get('/games/hxxy/activities')
      if (r.code === 0) {
        this.act = r.data
        this.loadActFlags()
      } else {
        this.tip(r.msg)
      }
    },
    async claimActivity(act, tier) {
      const r = await api.post('/games/hxxy/activities/claim', { act, tier })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadActivities()
        this.refreshPlayer()
      }
    },
    async openPm(pid) {
      const r = await api.get('/games/hxxy/pm/' + pid)
      if (r.code === 0) {
        this.pmCur = r.data
        this.pmText = ''
        this.cur = 'pm'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async sendPm() {
      if (!this.pmText.trim()) { this.tip('不能发送空消息'); return }
      const r = await api.post('/games/hxxy/pm/send', { to_id: this.pmCur.player.player_id, content: this.pmText.trim() })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.pmText = ''
        this.openPm(this.pmCur.player.player_id)
      } else {
        this.tip(r.msg)
      }
    },
    async addFriendCur() {
      const r = await api.post('/games/hxxy/friends/add', { player_id: this.pmCur.player.player_id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
    // ---------- 功能NPC ----------
    async viewNpc(id) {
      const r = await api.get('/games/hxxy/npc/' + id)
      if (r.code === 0) {
        this.npcCur = r.data
        this.cur = 'npcview'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async npcTele(t) {
      const r = await api.post('/games/hxxy/npc/teleport', { npc_id: this.npcCur.id, dtx: t.dtx, dty: t.dty })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.npcCur = null
        this.cur = 'map'
        this.loadState()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    npcShop(kind) {
      this.loadShop(kind)
      this.cur = 'shop'
      window.scrollTo(0, 0)
    },
    async npcRest() {
      const r = await api.post('/games/hxxy/rest', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
      } else {
        this.tip(r.msg)
      }
    },
    openWarehouse() {
      this.storeMode = 1
      this.bagDetail = null
      this.cur = 'bag'
      this.loadBag()
      window.scrollTo(0, 0)
    },
    async move(dir, jump) {
      const r = await api.post('/games/hxxy/move', { dir, jump: !!jump })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.loadState()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 查看地图（复刻 xy008） ----------
    async loadMapGrid() {
      const r = await api.get('/games/hxxy/map/grid')
      if (r.code === 0) {
        this.mapGrid = r.data
      } else {
        this.tip(r.msg)
        this.cur = 'map'
      }
    },
    mapZoom(delta) {
      if (delta === 0) { this.mapSize = 11; return }
      this.mapSize = Math.min(41, Math.max(5, this.mapSize + delta))
    },
    async doRest() {
      const r = await api.post('/games/hxxy/rest', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 战斗 ----------
    // 刷新后恢复进行中的战斗：有遗留战斗直接回到战斗页（原版 WAP 会话行为）
    async restoreBattle() {
      const r = await api.get('/games/hxxy/battle/state')
      if (r.code === 0 && r.data.in_battle) {
        this.bt = r.data
        this.cur = 'battle'
        this.refreshPlayer()
        this.loadBattleItems()
        this.loadSkills()
        window.scrollTo(0, 0)
      }
    },
    async startBattle(npcID) {
      const r = await api.post('/games/hxxy/battle/start', { npc_id: npcID })
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        this.loadSkills()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async loadBattleItems() {
      const r = await api.get('/games/hxxy/bag')
      if (r.code === 0) {
        // effect 兼容字符串/对象两种形态；战斗中可用的药品=气血/法力恢复类
        this.battleItems = (r.data.items || []).filter(b => b.kind === 'item').map(b => {
          let eff = b.effect
          if (typeof eff === 'string') { try { eff = JSON.parse(eff || '{}') } catch (e) { eff = {} } }
          return Object.assign({}, b, { effect: eff || {} })
        }).filter(b => b.effect.hp > 0 || b.effect.mp > 0)
        this.bagLoaded = true
        this.loadQuickSlots()
      }
    },
    async battleAct(act, skillID, bagID) {
      const r = await api.post('/games/hxxy/battle/action', { act, skill_id: skillID || 0, bag_id: bagID || 0 })
      if (r.code === 0) {
        this.bt = r.data
        if (!r.data.in_battle) {
          this.refreshPlayer()
          this.loadBlogs()
        }
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    backToMap() { this.go('map') },
    // ---------- 状态 ----------
    async loadAttrs() {
      const r = await api.get('/games/hxxy/attrs')
      if (r.code === 0) {
        this.g = r.data.player
        this.at = r.data
        this.equippedIDs = (r.data.equips || []).map(e => e.bag_id)
      } else {
        this.tip(r.msg)
      }
    },
    async takeoff(slot) {
      const r = await api.post('/games/hxxy/equip/takeoff', { slot })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadAttrs()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 行囊/仓库 ----------
    switchStore(m) {
      this.storeMode = m
      this.bagDetail = null
      this.loadBag()
    },
    async loadBag() {
      const r = await api.get('/games/hxxy/bag', this.storeMode === 1 ? { params: { store: 1 } } : undefined)
      if (r.code === 0) {
        this.g = r.data.player
        this.bagList = r.data.items || []
        this.bagCap = r.data.cap
        this.bagUsed = r.data.used
        this.gems = this.bagList.filter(b => b.kind === 'item' && b.category === 2)
      } else {
        this.tip(r.msg)
      }
    },
    async useItem(b) {
      const isTalent = (b.name || '').indexOf('腾云') >= 0
      if (isTalent) {
        const dtx = parseInt(window.prompt('腾云符·目标横坐标(dtx)：', '0') || '-1', 10)
        const dty = parseInt(window.prompt('腾云符·目标纵坐标(dty)：', '0') || '-1', 10)
        if (isNaN(dtx) || isNaN(dty) || dtx < 0) { this.tip('坐标无效'); return }
        const r2 = await api.post('/games/hxxy/bag/use', { bag_id: b.id, dtx, dty })
        if (r2.code === 0) {
          this.tip(r2.data.msg)
          this.g = r2.data.player
          this.loadBag()
          this.go('map')
        } else {
          this.tip(r2.msg)
        }
        return
      }
      const r = await api.post('/games/hxxy/bag/use', { bag_id: b.id })
      if (r.code === 0) {
        this.tip(r.data.msg || '使用成功！')
        this.g = r.data.player || this.g
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async dropItem(b, count) {
      const r = await api.post('/games/hxxy/bag/discard', { bag_id: b.id, count: count || 1 })
      if (r.code === 0) {
        this.tip('丢弃成功')
        this.bagDetail = null
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async storeMove(bagID, dir) {
      const url = dir === 'in' ? '/games/hxxy/warehouse/deposit' : '/games/hxxy/warehouse/takeout'
      const r = await api.post(url, { bag_id: bagID })
      if (r.code === 0) {
        this.tip(r.data.msg || '操作成功')
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async wearEquip(bagID) {
      const r = await api.post('/games/hxxy/equip/wear', { bag_id: bagID })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async upgradeEquip(bagID) {
      const r = await api.post('/games/hxxy/equip/upgrade', { bag_id: bagID })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async holeEquip(bagID) {
      const r = await api.post('/games/hxxy/equip/hole', { bag_id: bagID })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    async gemEquip(bagID, gemBag) {
      const r = await api.post('/games/hxxy/equip/gem', { bag_id: bagID, gem_bag: gemBag })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 赠银/赠物（复刻 xy537/538） ----------
    giveMoney() { this.giveMode = 'money'; this.giveAmount = '' },
    giveItem() { this.giveMode = 'item'; this.giveBagId = 0; this.giveCount = ''; this.loadBag() },
    async doGiveMoney(pv) {
      const amt = parseInt(this.giveAmount, 10)
      if (!amt || amt <= 0) { this.tip('输入有误，或者不能为空'); return }
      const r = await api.post('/games/hxxy/give/money', { to_id: pv.player_id, amount: amt })
      if (r.code === 0) { this.tip(r.data.msg); this.giveMode = ''; this.refreshPlayer() } else { this.tip(r.msg) }
    },
    pickGiveItem(b) { this.giveBagId = b.id; this.giveCount = '' },
    async doGiveItem(pv) {
      const n = parseInt(this.giveCount, 10)
      if (!n || n <= 0) { this.tip('输入有误，或者不能为空'); return }
      const r = await api.post('/games/hxxy/give/item', { to_id: pv.player_id, bag_id: this.giveBagId, count: n })
      if (r.code === 0) { this.tip(r.data.msg); this.giveMode = ''; this.loadBag() } else { this.tip(r.msg) }
    },
    // 恶名称号（复刻 xy093.php emz 阶梯）
    emzName(emz) {
      const e = emz || 0
      const tiers = [[1, 20, '坏蛋'], [21, 40, '匪徒'], [41, 60, '恶人'], [61, 80, '恶棍'], [81, 100, '恶霸'], [101, 150, '凶人'], [151, 250, '凶徒'], [251, 300, '凶手'], [301, 350, '暴徒'], [351, 400, '暴君'], [401, 450, '嗜血成性'], [451, 500, '赶尽杀绝'], [501, 600, '杀人如麻'], [601, 700, '十恶不赦'], [701, 800, '血流成河'], [801, 900, '血染山河'], [901, 1000, '十方俱灭']]
      for (const [lo, hi, n] of tiers) { if (e >= lo && e <= hi) return '【' + n + '】(' + e + '点)' }
      return e >= 1001 ? '【神档杀神~佛档杀佛】(' + e + '点)' : '【与世无争】(' + e + '点)'
    },
    // ---------- 挂售/拍卖上架（复刻 gssjwp01/pmsjwp01：数量+单价单页表单） ----------
    openStall(b) {
      this.aucItem = null
      this.stallItem = b
      this.stallCount = ''
      this.stallPrice = ''
    },
    async doStallSell() {
      const count = parseInt(this.stallCount, 10)
      const price = parseInt(this.stallPrice, 10)
      if (!count || count <= 0 || !price || price <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/stall/sell', { bag_id: this.stallItem.id, count, price })
      if (r.code === 0) {
        this.stallItem = null
        this.tip(r.data.msg)
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    openAuctionSell(b) {
      this.stallItem = null
      this.aucItem = b
      this.aucCount = ''
      this.aucPrice = ''
    },
    async doAuctionSell() {
      const count = parseInt(this.aucCount, 10)
      const price = parseInt(this.aucPrice, 10)
      if (!count || count <= 0 || !price || price <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/auction/sell', { bag_id: this.aucItem.id, count, price })
      if (r.code === 0) {
        this.aucItem = null
        this.tip(r.data.msg)
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 我的挂售（复刻 xy225→xy219） ----------
    async openStallMine(kind) {
      this.stallKind = kind
      const r = await api.get('/games/hxxy/stalls/mine', { params: { kind } })
      if (r.code === 0) {
        this.stallMine = r.data
        this.stallPage = 'mine'
        this.stallDetailId = 0
        this.stallCancelTarget = 0
        this.stallCancelCount = ''
      } else {
        this.tip(r.msg)
      }
    },
    async doStallCancel(s, count) {
      const n = parseInt(count, 10)
      if (!n || n <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/stall/cancel', { stall_id: s.stall_id, count: n })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.openStallMine(this.stallKind)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 他人挂售（复刻 xy222，从玩家资料页进入） ----------
    async openStallOf(id) {
      const r = await api.get('/games/hxxy/stalls/player/' + id)
      if (r.code === 0) {
        this.stallOf = r.data
        this.stallBuyTarget = 0
        this.stallBuyCount = ''
        this.go('stallof')
      } else {
        this.tip(r.msg)
      }
    },
    async doStallBuy(s, count) {
      const n = parseInt(count, 10)
      if (!n || n <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/stall/buy', { stall_id: s.stall_id, count: n })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.refreshPlayer()
        this.openStallOf(this.stallOf.seller_id)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 全区拍卖（复刻 xy489/499） ----------
    async openAuction() {
      this.auctionView = 'list'
      this.loadAuction(this.auctionTab)
    },
    async loadAuction(tab) {
      const r = await api.get('/games/hxxy/auction', { params: { tab } })
      if (r.code === 0) {
        this.auctionTab = tab
        this.auctionList = r.data.list || []
        this.auctionView = 'list'
        this.auctionBuyTarget = 0
        this.auctionBuyCount = ''
      } else {
        this.tip(r.msg)
      }
    },
    async openAuctionMine() {
      const r = await api.get('/games/hxxy/auction/mine')
      if (r.code === 0) {
        this.auctionMine = r.data.list || []
        this.auctionView = 'mine'
      } else {
        this.tip(r.msg)
      }
    },
    async doAuctionBuy(a, count) {
      const n = parseInt(count, 10)
      if (!n || n <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/auction/buy', { auction_id: a.auction_id, count: n })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.refreshPlayer()
        this.loadAuction(this.auctionTab)
      } else {
        this.tip(r.msg)
      }
    },
    async doAuctionCancel(a) {
      const r = await api.post('/games/hxxy/auction/cancel', { auction_id: a.auction_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.openAuctionMine()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 技能 ----------
    async loadSkills() {
      const r = await api.get('/games/hxxy/skills')
      if (r.code === 0) { this.sk = r.data; this.skLoaded = true; this.loadQuickSlots() }
    },
    // ---------- 战斗快捷键（复刻原版 快捷键1~9 槽位，可存「技能」或「药品」） ----------
    quickKey() {
      const server = this.serverName || localStorage.getItem('hxxy_server') || ''
      return 'hxxy_quick_' + server + '_' + (this.g.pid || this.g.id || this.g.name || 'self')
    },
    // 已学战斗技能（原版 xy015 技能页）
    quickSkillList() {
      return ((this.sk && this.sk.mine) || []).filter(s => s.category === 1)
    },
    // 可用药品（原版 xy016 药品页：气血/法力恢复类道具）
    quickItemList() {
      return this.battleItems || []
    },
    // 当前选择页的候选列表
    quickPickList() {
      return this.quickPickTab === 'item' ? this.quickItemList() : this.quickSkillList()
    },
    // 读取快捷槽：兼容旧格式；仅当来源列表已加载时才清理失效槽（避免列表未加载时误清）
    loadQuickSlots() {
      let raw = {}
      try { raw = JSON.parse(localStorage.getItem(this.quickKey()) || '{}') } catch (e) { raw = {} }
      const skills = this.quickSkillList()
      const items = this.quickItemList()
      const out = []
      let dirty = false
      for (let i = 1; i <= 9; i++) {
        const cur = raw[i] || raw['k' + i] // 兼容旧格式 {k1: skillId}
        let kind = '', id = 0, name = '', cost = 0
        if (cur && typeof cur === 'object') { kind = cur.t || ''; id = cur.id || 0; name = cur.n || ''; cost = cur.c || 0 } else if (cur) { kind = 'skill'; id = cur }
        if (kind === 'skill') {
          if (this.skLoaded) {
            const hit = skills.find(s => s.skill_id === id)
            if (hit) { name = hit.name; cost = hit.mp_cost } else { kind = ''; id = 0; name = ''; cost = 0; dirty = true }
          }
        } else if (kind === 'item') {
          if (this.bagLoaded) {
            const hit = items.find(b => b.id === id)
            if (hit) { name = hit.name; cost = 0 } else { kind = ''; id = 0; name = ''; cost = 0; dirty = true }
          }
        } else { kind = ''; id = 0; name = ''; cost = 0 }
        out.push({ slot: i, kind: kind, ref_id: id, name: name, mp_cost: cost })
      }
      this.quickSlots = out
      if (dirty) this.saveQuickSlots()
    },
    saveQuickSlots() {
      const raw = {}
      this.quickSlots.forEach(q => { if (q.kind && q.ref_id) raw[q.slot] = { t: q.kind, id: q.ref_id, n: q.name, c: q.mp_cost } })
      localStorage.setItem(this.quickKey(), JSON.stringify(raw))
    },
    // 写入某槽位（原版 xy247）：kind='skill'|'item'，refId=0 表示清空
    bindQuick(slot, kind, refId) {
      const idx = slot - 1
      if (!this.quickSlots[idx]) return
      const q = this.quickSlots[idx]
      if (!refId || !kind) {
        q.kind = ''; q.ref_id = 0; q.name = ''; q.mp_cost = 0
      } else if (kind === 'item') {
        const hit = this.quickItemList().find(b => b.id === refId)
        if (!hit) { this.tip('该药品不存在'); return }
        q.kind = 'item'; q.ref_id = refId; q.name = hit.name; q.mp_cost = 0
      } else {
        const hit = this.quickSkillList().find(s => s.skill_id === refId)
        if (!hit) { this.tip('该技能不存在'); return }
        q.kind = 'skill'; q.ref_id = refId; q.name = hit.name; q.mp_cost = hit.mp_cost
      }
      this.saveQuickSlots()
      this.tip('成功将快捷' + slot + '设置为了' + (q.name || '空'))
    },
    pickQuick(slot, kind, refId) {
      this.bindQuick(slot, kind, refId)
      this.quickSetSlot = 0
    },
    // 重置全部快捷键（原版 xy305）
    resetQuick() {
      localStorage.removeItem(this.quickKey())
      this.loadQuickSlots()
      this.tip('快捷方式重置成功！')
    },
    // 战斗中点击快捷键使用（原版 xy248 / jnxx.php：药品用完自动清空该快捷）
    useQuick(q) {
      if (q.kind === 'item') {
        const hit = this.quickItemList().find(b => b.id === q.ref_id)
        if (!hit || hit.count < 1) { this.loadQuickSlots(); this.tip('快捷' + q.slot + '的药品已用完，已清空'); return }
        this.battleAct('item', 0, q.ref_id)
      } else {
        this.battleAct('skill', q.ref_id)
      }
    },
    toggleQuickSet() {
      this.showQuickSet = !this.showQuickSet
      if (this.showQuickSet) {
        this.quickSetSlot = 0
        this.quickPickTab = 'skill'
        this.loadBattleItems()
      }
    },
    openQuickSetFor(slot) {
      this.showQuickSet = true
      this.quickSetSlot = slot
      this.quickPickTab = 'skill'
      this.loadBattleItems()
    },
    async learnSkill(s) {
      const r = await api.post('/games/hxxy/skills/learn', { skill_id: s.skill_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadSkills()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 宠物 ----------
    async loadPets() {
      const r = await api.get('/games/hxxy/pets')
      if (r.code === 0) this.petList = r.data.pets || []
    },
    async petAct(pt, act) {
      if (act === 'free' && !window.confirm('确定放生【' + pt.name + '】吗？此操作不可恢复！')) return
      const r = await api.post('/games/hxxy/pets/act', { act, pet_id: pt.id, name: this.petRenameName })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.petRenameId = 0
        this.petRenameName = ''
        this.loadPets()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 商店 ----------
    async loadShop(kind) {
      this.shopKind = kind
      this.shopPage = 'list'
      this.shopMsg = ''
      const r = await api.get('/games/hxxy/shop/' + kind)
      if (r.code === 0) {
        this.shop = { goods: r.data.goods || [], pets: r.data.pets || null, used: r.data.used || 0, cap: r.data.cap || 0 }
      } else {
        this.tip(r.msg)
      }
    },
    openShopItem(gd) {
      this.shopItem = gd
      this.shopBuyCount = ''
      this.shopMsg = ''
      this.shopPage = 'item'
    },
    openShopPet(pt) {
      this.shopPet = pt
      this.shopBuyCount = ''
      this.shopMsg = ''
      this.shopPage = 'pet'
    },
    async doShopBuy(currency) {
      if (!/^\d+$/.test(String(this.shopBuyCount)) || Number(this.shopBuyCount) <= 0) {
        this.shopMsg = '输入有误请重新输入'
        return
      }
      const gd = this.shopItem
      const r = await api.post('/games/hxxy/shop/buy', { kind: gd.kind, ref_id: gd.ref_id, count: Number(this.shopBuyCount), currency })
      if (r.code === 0) {
        this.shopBuyCount = ''
        this.shopPage = 'list'
        await this.loadShop(this.shopKind)
        this.shopMsg = r.data.msg
        this.refreshPlayer()
      } else {
        this.shopMsg = r.msg
      }
    },
    async doShopPetBuy() {
      if (!/^\d+$/.test(String(this.shopBuyCount)) || Number(this.shopBuyCount) <= 0) {
        this.shopMsg = '输入有误请重新输入'
        return
      }
      const r = await api.post('/games/hxxy/shop/buy', { kind: 'pet', ref_id: this.shopPet.species_id, count: Number(this.shopBuyCount), currency: 'beans' })
      if (r.code === 0) {
        this.shopBuyCount = ''
        this.shopPage = 'list'
        await this.loadShop(this.shopKind)
        this.shopMsg = r.data.msg
        this.refreshPlayer()
      } else {
        this.shopMsg = r.msg
      }
    },
    // ---------- 银行 ----------
    async bankOp(dir) {
      const amt = Number(this.bankAmount)
      if (!amt || amt <= 0) { this.bankMsg = '输入有误请重新输入'; return }
      const url = dir === 'in' ? '/games/hxxy/bank/deposit' : '/games/hxxy/bank/withdraw'
      const r = await api.post(url, { amount: amt })
      if (r.code === 0) {
        this.bankMsg = r.data.msg
        this.bankAmount = 0
        this.refreshPlayer()
      } else {
        this.bankMsg = r.msg
      }
    },
    // ---------- 签到 ----------
    async loadSigninInfo() {
      const r = await api.get('/games/hxxy/signin/info')
      if (r.code === 0) this.sign = r.data
    },
    async doSignin() {
      const r = await api.post('/games/hxxy/signin', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadSigninInfo()
        this.refreshPlayer()
      }
    },
    async claimSign(tier) {
      const r = await api.post('/games/hxxy/signin/claim', { tier })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadSigninInfo()
        this.refreshPlayer()
      }
    },
    // ---------- 福利中心 ----------
    async loadWelfare() {
      const r = await api.get('/games/hxxy/welfare')
      if (r.code === 0) this.wf = r.data
      this.loadSigninInfo()
    },
    fmtWait(s) {
      if (!s || s <= 0) return ''
      const m = Math.floor(s / 60)
      const sec = s % 60
      return m > 0 ? m + '分' + sec + '秒' : sec + '秒'
    },
    async giftClaim() {
      const r = await api.post('/games/hxxy/welfare/gift', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) { this.loadWelfare(); this.refreshPlayer() }
    },
    async nobleClaim(tier) {
      const r = await api.post('/games/hxxy/welfare/noble', { tier })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) { this.loadWelfare(); this.refreshPlayer() }
    },
    async copyPromo() {
      const text = '进群填写我的邀请游戏ID：' + (this.playerID || '') + '\n填写邀请ID即可领取超值【三区水帘洞助力包】包含【幻想套装】【vip练级卷】x20，【10亿修炼经验丹】x10，【万能果】x100，【1万西游声望卷轴】x100，【1万法宝经验卷轴】x100，〖瞌睡虫〗（典藏版）x5\n我不断的寻找，有你的世界在哪儿\n新区【水帘洞】人气火爆，进群领取豪华大礼包，只等你来！'
      try {
        await navigator.clipboard.writeText(text)
        this.tip('宣传语已复制')
      } catch (e) {
        const ta = document.createElement('textarea')
        ta.value = text
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
        this.tip('宣传语已复制')
      }
    },
    // ---------- 任务 ----------
    qstCatName(c) { return { 1: '主线', 2: '支线', 3: '日常' }[c] || '任务' },
    async loadQuests() {
      const r = await api.get('/games/hxxy/quests')
      if (r.code === 0) this.qst = r.data
    },
    async questAccept(q) {
      const r = await api.post('/games/hxxy/quests/accept', { quest_id: q.quest_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadQuests()
        if (this.cur === 'npcview' && this.npcCur) this.viewNpc(this.npcCur.id) // NPC页接取后刷新任务列表
      } else {
        this.tip(r.msg)
      }
    },
    async questAbandon(q) {
      const r = await api.post('/games/hxxy/quests/abandon', { quest_id: q.quest_id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadQuests()
    },
    async questSubmit(q) {
      const r = await api.post('/games/hxxy/quests/submit', { quest_id: q.quest_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadQuests()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 副本 / BOSS ----------
    async loadDungeons() {
      const r = await api.get('/games/hxxy/dungeons')
      if (r.code === 0) this.dungeons = r.data.dungeons || []
    },
    async enterDungeon(dg) {
      const r = await api.post('/games/hxxy/dungeons/enter', { dungeon_id: dg.dungeon_id })
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async loadBosses() {
      const r = await api.get('/games/hxxy/bosses')
      if (r.code === 0) this.bossList = r.data.bosses || []
    },
    async challengeBoss(b) {
      const r = await api.post('/games/hxxy/bosses/challenge', { boss_id: b.boss_id })
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 修炼 ----------
    async loadCult() {
      const r = await api.get('/games/hxxy/cultivate')
      if (r.code === 0) this.cult = r.data
    },
    async cultToggle() {
      const r = await api.post('/games/hxxy/cultivate/toggle', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadCult()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // 修炼线境界名（复刻 xlms.php：下一级境界+层，封顶显示 天尊（封顶））
    xlTrackName(t) {
      if (t.capped) return '（天尊）（封顶）'
      return '（' + t.realm + '）' + t.layer + '层'
    },
    xlNeedText(t) {
      const n = t.need || {}
      return '修炼经验' + (n.exp || 0) + '，银两' + (n.silver || 0) + '，西游声望' + (n.sw || 0) + (n.beans > 0 ? '，〖金豆〗x' + n.beans : '')
    },
    async cultUp(slot) {
      const r = await api.post('/games/hxxy/cultivate/upgrade', { slot })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadCult()
        this.refreshPlayer()
      }
    },
    // ---------- 头衔 ----------
    async loadTitles(page) {
      const r = await api.get('/games/hxxy/titles?page=' + (page || 1))
      if (r.code === 0) this.titles = r.data
    },
    openTitle(t) {
      this.titleDetail = t
      this.titleView = 'detail'
    },
    async cultExchangeDan() {
      const r = await api.post('/games/hxxy/cultivate/exchange-dan', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadCult()
    },
    async activateTitle(t) {
      const r = await api.post('/games/hxxy/titles/activate', { title_id: t.title_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.titleDetail.owned = true
        this.loadTitles(this.titles.page)
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async wearTitle(t) {
      const r = await api.post('/games/hxxy/titles/wear', { title_id: t.title_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        if (this.titleView === 'detail' && this.titleDetail.title_id === t.title_id) {
          this.titleDetail = Object.assign({}, this.titleDetail)
        }
        this.loadTitles(this.titles.page)
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 排行 ----------
    async loadRank(t) {
      this.rankType = t
      const r = await api.get('/games/hxxy/rank/' + t)
      if (r.code === 0) this.rankList = r.data.list || []
    },
    // ---------- 聊天 ----------
    async loadChat() {
      const r = await api.get('/games/hxxy/chat')
      if (r.code === 0) this.chatList = r.data.chats || []
    },
    async sendChat() {
      const content = this.chatInput.trim()
      if (!content) { this.tip('请输入聊天内容'); return }
      const r = await api.post('/games/hxxy/chat', { content })
      this.chatInput = ''
      if (r.code === 0) {
        this.loadChat()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 好友 ----------
    async loadFriends() {
      const r = await api.get('/games/hxxy/friends')
      if (r.code === 0) this.frd = r.data
    },
    // ---------- 拉黑/删除（复刻 xy104/xy115/xy117） ----------
    async blackPlayer(pid) {
      const r = await api.post('/games/hxxy/friends/black', { player_id: pid })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
    async removeFriend(f) {
      const r = await api.post('/games/hxxy/friends/remove', { player_id: f.player_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadFriends()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 国家 ----------
    async loadGang() {
      const r = await api.get('/games/hxxy/gang')
      if (r.code === 0) this.gang = r.data
    },
    gangGo(sub, m) {
      this.gangSub = sub
      if (sub === 'mall') this.loadGangMall()
      if (sub === 'kick') this.gangKickTarget = m || null
    },
    gangOffName(role) {
      const o = (this.gang.my_gang.officials || {})['role' + role]
      return o ? o.name : '暂无'
    },
    async gangNew() {
      if (!this.gangNameInput.trim()) { this.tip('国家名不能为空'); return }
      const r = await api.post('/games/hxxy/gang/create', { name: this.gangNameInput.trim() })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.gangNameInput = ''
        this.loadGang()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async gangJoin(gp) {
      const r = await api.post('/games/hxxy/gang/join', { gang_id: gp.gang_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangQuitDo() {
      const r = await api.post('/games/hxxy/gang/leave', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.gangGo('')
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangDonate() {
      const amt = Number(this.gangDonateAmount)
      if (!amt || amt <= 0) { this.tip('输入有误请重新输入'); return }
      const r = await api.post('/games/hxxy/gang/donate', { amount: amt })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.gangDonateAmount = 0
        this.loadGang()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async loadGangMall() {
      const r = await api.get('/games/hxxy/gang/mall')
      if (r.code === 0) {
        this.gangMall = r.data
        if (!r.data.tabs.some(t => t.level === this.gangMallTab)) this.gangMallTab = 1
      }
    },
    async gangMallBuy(it) {
      const r = await api.post('/games/hxxy/gang/mall/buy', { item_id: it.item_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGangMall()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async gangUpgrade() {
      if (!window.confirm('确定升级国家吗？升级将扣除国家资金/经验/声望！')) return
      const r = await api.post('/games/hxxy/gang/upgrade', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangDissolve() {
      const r = await api.post('/games/hxxy/gang/dissolve', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.gangGo('')
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangAppoint(m) {
      const r = await api.post('/games/hxxy/gang/appoint', { player_id: m.player_id, role: this.gangAppointRole })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangDismiss(m) {
      if (!window.confirm('确定罢免【' + m.name + '】的【' + m.role_name + '】职务吗？')) return
      const r = await api.post('/games/hxxy/gang/dismiss', { player_id: m.player_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangKickDo() {
      if (!this.gangKickTarget) { this.gangGo('members'); return }
      const r = await api.post('/games/hxxy/gang/kick', { player_id: this.gangKickTarget.player_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.gangKickTarget = null
        this.gangGo('members')
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 结婚 ----------
    async loadMarriage() {
      const r = await api.get('/games/hxxy/marriage')
      if (r.code === 0) this.marriage = r.data.marriage || { status: 0 }
    },
    async doPropose() {
      if (!this.marryName.trim()) { this.tip('请输入对方名字'); return }
      const r = await api.post('/games/hxxy/marriage/propose', { name: this.marryName.trim() })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.marryName = ''
        this.loadMarriage()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async doAgreeMarry() {
      const r = await api.post('/games/hxxy/marriage/agree', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadMarriage()
      } else {
        this.tip(r.msg)
      }
    },
    async doDivorce() {
      if (!window.confirm('确定离婚吗？')) return
      const r = await api.post('/games/hxxy/marriage/divorce', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadMarriage()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 住宅 ----------
    async loadHouse() {
      const r = await api.get('/games/hxxy/house')
      if (r.code === 0) this.furniture = r.data.furniture || []
    },
    async buyFurniture(f) {
      const r = await api.post('/games/hxxy/house/buy', { fid: f.fid })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadHouse()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 流水/战报 ----------
    async loadWallet() {
      const r = await api.get('/games/hxxy/wallet')
      if (r.code === 0) this.walletLogs = r.data.logs || []
    },
    async loadBlogs() {
      const r = await api.get('/games/hxxy/battle-logs')
      if (r.code === 0) this.blogLogs = r.data.logs || []
    },
    // ---------- 充值 ----------
    async loadVip() {
      const r = await api.get('/games/hxxy/vip/info')
      if (r.code === 0) this.vip = r.data
    },
    async vipExchange() {
      if (this.vip.exchanged) { this.tip('今日已兑换过，明日再来'); return }
      const r = await api.post('/games/hxxy/vip/exchange', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadVip()
        this.refreshPlayer()
      }
    },
    async doRecharge() {
      const code = this.rechargeCode.trim()
      if (!code) { this.tip('请输入充值码'); return }
      const r = await api.post('/games/hxxy/vip/recharge', { code })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.rechargeCode = ''
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 玩法：通天塔/擂台/娱乐/腾云/资料 ----------
    async loadTower() {
      const r = await api.get('/games/hxxy/tower/info')
      if (r.code === 0) this.tower = r.data
    },
    async towerStart() {
      const r = await api.post('/games/hxxy/tower/start', {})
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async loadArena() {
      const r = await api.get('/games/hxxy/arena/info')
      if (r.code === 0) this.arena = r.data
    },
    async arenaFight(r) {
      if (!window.confirm('向【' + r.name + '】（' + r.level + '级）发起比武挑战？')) return
      await this.pvpStart(r.player_id)
    },
    async fightPlayer(pv) {
      if (window.confirm('向【' + pv.name + '】（' + pv.level + '级）发起比武挑战？')) await this.pvpStart(pv.player_id)
    },
    async pvpStart(targetID) {
      const r = await api.post('/games/hxxy/arena/fight', { target_id: targetID })
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async doRoll(field) {
      const r = await api.post('/games/hxxy/fun/roll', { field })
      if (r.code === 0) {
        this.funRoll = r.data.symbols
        this.tip(r.data.msg)
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async loadTeyun() {
      const r = await api.get('/games/hxxy/teyun/list')
      if (r.code === 0) this.teyun = r.data
    },
    async teyunGo(t) {
      if (!window.confirm('消耗1张腾云符传送到【' + t.name + '】？')) return
      const r = await api.post('/games/hxxy/teyun/go', { dtx: t.dtx, dty: t.dty })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.g = r.data.player
        this.cur = 'map'
        this.loadState()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async viewPlayer(pid) {
      if (!pid) return
      const r = await api.get('/games/hxxy/player/' + pid)
      if (r.code === 0) {
        this.pv = r.data
        this.pvFrom = this.cur
        this.cur = 'playerview'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    backPv() { this.cur = this.pvFrom || 'home'; window.scrollTo(0, 0) },
    async addFriendByID(pid) {
      const r = await api.post('/games/hxxy/friends/add', { player_id: pid })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
    exitToServer() {
      if (!window.confirm('退出当前游戏并重新选择服务器？')) return
      sessionStorage.removeItem('hxxy_cur')
      this.cur = 'server'
      window.scrollTo(0, 0)
    },
    // ---------- 国战 / 组队 ----------
    fmtSec(s) {
      if (s <= 0) return '0秒'
      const m = Math.floor(s / 60), sec = s % 60
      return m > 0 ? m + '分' + sec + '秒' : sec + '秒'
    },
    async loadGz() {
      const r = await api.get('/games/hxxy/gz/info')
      if (r.code === 0) {
        this.gz = r.data
        this.cur = 'gz'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async refreshGz() {
      const r = await api.get('/games/hxxy/gz/info')
      if (r.code === 0) this.gz = r.data
    },
    async gzSignup() {
      const r = await api.post('/games/hxxy/gz/signup', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      this.refreshGz()
    },
    async gzAction(path) {
      const r = await api.post('/games/hxxy/' + path, {})
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async loadTeam() {
      const r = await api.get('/games/hxxy/team/info')
      if (r.code === 0) {
        this.team = r.data
        this.cur = 'team'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async teamCreate() {
      const r = await api.post('/games/hxxy/team/create', {})
      if (r.code === 0) {
        this.team.team = r.data.team
        this.tip(r.data.msg)
      } else {
        this.tip(r.msg)
      }
    },
    async teamInvite(pv) {
      if (!window.confirm('向【' + pv.name + '】发起组队邀请？')) return
      const r = await api.post('/games/hxxy/team/invite', { target_id: pv.player_id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
    async teamAgree(iv) {
      const r = await api.post('/games/hxxy/team/agree', { invite_id: iv.id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        const res = await api.get('/games/hxxy/team/info')
        if (res.code === 0) this.team = res.data
      }
    },
    async teamRefuse(iv) {
      const r = await api.post('/games/hxxy/team/refuse', { invite_id: iv.id })
      if (r.code === 0) {
        this.team.invites = this.team.invites.filter(x => x.id !== iv.id)
        this.tip(r.data.msg)
      }
    },
    async teamKick(m) {
      if (!window.confirm('将【' + m.name + '】踢出队伍？')) return
      const r = await api.post('/games/hxxy/team/kick', { target_id: m.player_id })
      if (r.code === 0) {
        this.team.team = r.data.team
        this.tip(r.data.msg)
      } else {
        this.tip(r.msg)
      }
    },
    async teamLeave(dismiss) {
      if (!window.confirm(dismiss ? '确定解散队伍？' : '确定离开队伍？')) return
      const r = await api.post('/games/hxxy/team/leave', {})
      if (r.code === 0) {
        this.team.team = r.data.team
        this.tip(r.data.msg)
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 首页邀请处理（复刻原版 yq1~yq4.php） ----------
    async gangAgree(iv) {
      const r = await api.post('/games/hxxy/gang/invite/agree', { id: iv.id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) {
        this.loadHome()
        this.loadGang()
      }
    },
    async gangRefuse(iv) {
      const r = await api.post('/games/hxxy/gang/invite/refuse', { id: iv.id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadHome()
    },
    async houseAgree(iv) {
      const r = await api.post('/games/hxxy/house/invite/agree', { id: iv.id })
      if (r.code === 0) {
        this.visitHouse = { owner_name: r.data.owner_name, furniture: r.data.furniture || [] }
        this.tip(r.data.msg)
        this.cur = 'housevisit'
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async houseRefuse(iv) {
      const r = await api.post('/games/hxxy/house/invite/refuse', { id: iv.id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadHome()
    },
    async marryAgree() {
      const r = await api.post('/games/hxxy/marriage/agree', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadHome()
    },
    async marryRefuse() {
      const r = await api.post('/games/hxxy/marriage/refuse', {})
      this.tip(r.code === 0 ? r.data.msg : r.msg)
      if (r.code === 0) this.loadHome()
    },
    async inviteVisitHouse(pv) {
      if (!window.confirm('邀请【' + pv.name + '】参观你的住宅？')) return
      const r = await api.post('/games/hxxy/house/invite', { player_id: pv.player_id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
    async inviteJoinGang(pv) {
      if (!window.confirm('邀请【' + pv.name + '】加入帮派？')) return
      const r = await api.post('/games/hxxy/gang/invite', { player_id: pv.player_id })
      this.tip(r.code === 0 ? r.data.msg : r.msg)
    },
  },
}
</script>

<style scoped>
.xy-wap { font-family: "Microsoft YaHei", "微软雅黑", "SimHei", "黑体", sans-serif; font-size: 15px; margin: 5px; line-height: 1.6; color: #000; word-break: break-all; min-height: 80vh; }
.xy-wap a { color: #0060CD; text-decoration: none; }
em { color: #9B9B9B; font-size: 12px; font-style: normal; }
.red { color: #ff0000; }
.green { color: #008000; }
.gray { color: #9B9B9B; }
.black { color: #000; }
.blue { color: #0060CD; }
.cur { color: #f60; font-weight: bold; }
.nk { color: #c00; }
.logo { text-align: left; margin: 4px 0; }
.mapimg img { max-width: 240px; width: 100%; height: auto; display: block; margin: 4px 0; }
.npcimg img { max-width: 120px; width: auto; height: auto; display: block; margin: 4px 0; }
/* 查看地图网格（复刻原版 MapViewer 配色） */
.mapgrid { background-color: #eec65a; text-align: center; font-size: 12px; border-collapse: collapse; }
.mapgrid td { min-width: 48px; word-break: keep-all; padding: 1px 2px; }
.mgnode { background-color: #942900; }
.mgcur { color: #0befe7; }
.mgjump { color: #86e2e2; }
.mgnorm { color: #fff; }
.mgwall { color: #000; }
.xy-tip { position: fixed; left: 50%; top: 20%; transform: translateX(-50%); background: rgba(0, 0, 0, 0.75); color: #fff; padding: 8px 16px; border-radius: 4px; z-index: 200; }
</style>
