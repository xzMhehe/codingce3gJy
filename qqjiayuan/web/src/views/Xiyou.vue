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

    <!-- ==================== 建角 ==================== -->
    <template v-else-if="cur === 'create'">
      【创建角色】<br/>
      欢迎来到幻想西游！选择你的门派，踏上取经之路。<br/>
      <div class="logo"><img src="/static/image/hxxy.jpg" width="120" height="60" alt="幻想西游" onerror="this.style.display='none'" /></div><br/>
      角色名：<input v-model="cf.name" maxlength="12" /><br/>
      性别：<a href="javascript:;" :class="cf.sex === 1 ? 'cur' : ''" @click="cf.sex = 1">[男]</a>
            <a href="javascript:;" :class="cf.sex === 2 ? 'cur' : ''" @click="cf.sex = 2">[女]</a><br/>
      门派：<br/>
      <div v-for="s in sects" :key="'sc' + s.id">
        <a href="javascript:;" :class="cf.sect === s.id ? 'cur' : ''" @click="cf.sect = s.id">【{{ s.name }}】</a>{{ s.desc }}<br/>
      </div>
      <em>月宫只收女弟子，普陀山只收男弟子</em><br/>
      <form @submit.prevent="doCreate">
        <input type="submit" value="踏上西游路" />
      </form>
      -----------<br/>
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
      <template v-if="g.vip > 0">·VIP祝福{{ g.vip }}分钟</template>)<br/>
      所在地：<a href="javascript:;" @click="go('map')">{{ g.node_name }}</a><br/>
      气血：<span class="red">{{ g.hp }}-{{ g.max_hp }}</span>
      法力：<span :class="g.mp < g.max_mp ? 'cur' : 'black'">{{ g.mp }}-{{ g.max_mp }}</span><br/>
      经验：{{ g.exp }}/{{ g.exp_need }}<br/>
      银两：<span class="cur">{{ g.money }}</span> 存款：{{ g.bank }} 金豆：<span class="red">{{ g.beans }}</span><br/>
      宠物：<template v-if="g.fighting_pet">{{ g.fighting_pet.name }}({{ g.fighting_pet.level }}级·参战中)</template><template v-else>无参战</template><br/>
      修炼：<a href="javascript:;" @click="go('cultivate')">{{ g.xiulian_switch === 1 ? '开启中' : '未开启' }}</a><br/>
      -----------<br/>
      <!-- 附近玩家（复刻原版 fjwj：同地图玩家可私聊） -->
      <template v-if="nearby.length">
        附近玩家：<br/>
        <div v-for="(n, i) in nearby" :key="'nb' + i">
          {{ i + 1 }}.<a href="javascript:;" @click="viewPlayer(n.player_id)">{{ n.name }}</a>({{ n.level }}级·{{ n.sect_name }})<br/>
        </div>
      </template>
      <a href="javascript:;" @click="go('map')">【西游世界】</a>
      <a href="javascript:;" @click="go('bosses')">【世界BOSS】</a>
      <a href="javascript:;" @click="go('dungeons')">【副本】</a><br/>
      <a href="javascript:;" @click="doSignin">【每日签到】</a>
      <a href="javascript:;" @click="go('quests')">【任务】</a>
      <a href="javascript:;" @click="go('vip')">【充值】</a><br/>
      -----------<br/>
      <span class="black">火热玩法</span><br/>
      <a href="javascript:;" @click="go('tower')">挑战</a>◎<a href="javascript:;" @click="go('arena')">擂台</a>◎<a href="javascript:;" @click="go('fun')">娱乐</a>◎<a href="javascript:;" @click="go('gz')">国战</a><br/>
      -----------<br/>
      <span class="black">攻略指引</span><br/>
      <a href="javascript:;" @click="go('stalls')">拍卖</a>◎<a href="javascript:;" @click="go('guide')">攻略</a>◎<a href="javascript:;" @click="go('guide')">指引</a>◎<a href="javascript:;" @click="go('teyun')">腾云</a><br/>
      -----------<br/>
      <span class="black">基础功能</span><br/>
      <a href="javascript:;" @click="go('attrs')">状态</a>◎<a href="javascript:;" @click="go('bag')">物品</a>◎<a href="javascript:;" @click="go('friends')">好友</a>◎<a href="javascript:;" @click="go('quests')">任务</a><br/>
      <a href="javascript:;" @click="go('gang')">国家</a>◎<a href="javascript:;" @click="go('chat')">聊天</a>◎<a href="javascript:;" @click="go('pets')">宠物</a>◎<a href="javascript:;" @click="go('shop')">商城</a><br/>
      <a href="javascript:;" @click="go('team')">队伍</a>◎<a href="javascript:;" @click="go('house')">住宅</a>◎<a href="javascript:;" @click="go('stalls')">挂售</a>◎<a href="javascript:;" @click="go('rank')">排行</a><br/>
      <a href="javascript:;" @click="go('vip')">兑奖</a>◎<a href="javascript:;" @click="go('vip')">特权</a>◎<a href="javascript:;" @click="go('signin')">福利</a>◎<a href="javascript:;" @click="go('sys')">系统</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('attrs')">状态</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('skills')">技能</a>.<a href="javascript:;" @click="go('pets')">宠物</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
      <a href="javascript:;" @click="go('bank')">银行</a>.<a href="javascript:;" @click="go('quests')">任务</a>.<a href="javascript:;" @click="go('dungeons')">副本</a>.<a href="javascript:;" @click="go('bosses')">BOSS</a>.<a href="javascript:;" @click="go('cultivate')">修炼</a>.<a href="javascript:;" @click="go('titles')">头衔</a>.<a href="javascript:;" @click="go('signin')">签到</a><br/>
      <a href="javascript:;" @click="go('rank')">排行</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('friends')">好友</a>.<a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('marriage')">结婚</a>.<a href="javascript:;" @click="go('house')">住宅</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a><br/>
      <a href="javascript:;" @click="go('wallet')">流水</a>.<a href="javascript:;" @click="go('blogs')">战报</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== 西游世界（地图） ==================== -->
    <template v-else-if="cur === 'map'">
      【{{ node.name }}】<a href="javascript:;" @click="loadState()">刷新</a><br/>
      <span v-if="node.desc" class="gray">{{ node.desc }}</span><br/>
      <div v-if="mapImgOk" class="mapimg"><img :src="'/static/hxxy/dtpic/' + g.map_x + '-' + g.map_y + '.jpg'" @error="mapImgOk = false" alt="地图" /></div>
      <template v-if="node.up">北：<a href="javascript:;" @click="move('up')">{{ node.up.name }}</a><template v-if="node.up.jump">[传送]</template><br/></template>
      <template v-if="node.down">南：<a href="javascript:;" @click="move('down')">{{ node.down.name }}</a><template v-if="node.down.jump">[传送]</template><br/></template>
      <template v-if="node.left">西：<a href="javascript:;" @click="move('left')">{{ node.left.name }}</a><template v-if="node.left.jump">[传送]</template><br/></template>
      <template v-if="node.right">东：<a href="javascript:;" @click="move('right')">{{ node.right.name }}</a><template v-if="node.right.jump">[传送]</template><br/></template>
      <template v-if="!node.up && !node.down && !node.left && !node.right">四面都是高墙，没有出路……<br/></template>
      <template v-if="mapNpcs.length">
        此处人物：<br/>
        <div v-for="n in mapNpcs" :key="'mn' + n.id">
          <a href="javascript:;" @click="viewNpc(n.id)"><span class="red">{{ n.name }}</span></a><template v-if="n.tele_count">[传送]</template><template v-if="n.shop">[服务]</template><br/>
        </div>
      </template>
      附近出没：<br/>
      <template v-if="enemies.length">
        <div v-for="(e, i) in enemies" :key="'en' + i">
          {{ i + 1 }}.<a href="javascript:;" @click="startBattle(e.npc_id)">{{ e.name }}</a>({{ e.level }}级·{{ e.difficulty }})
          <template v-if="e.take"><span class="gray">"{{ e.take }}"</span></template><br/>
        </div>
      </template>
      <template v-else><em>此地一片祥和，没有妖魔出没。</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="doRest">客栈休息({{ g.level * 10 }}银两回满)</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a>.<a href="javascript:;" @click="go('attrs')">状态</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('skills')">技能</a>.<a href="javascript:;" @click="go('pets')">宠物</a>.<a href="javascript:;" @click="go('shop')">商店</a><br/>
      <a href="javascript:;" @click="go('bank')">银行</a>.<a href="javascript:;" @click="go('quests')">任务</a>.<a href="javascript:;" @click="go('dungeons')">副本</a>.<a href="javascript:;" @click="go('bosses')">BOSS</a>.<a href="javascript:;" @click="go('cultivate')">修炼</a>.<a href="javascript:;" @click="go('titles')">头衔</a>.<a href="javascript:;" @click="go('signin')">签到</a><br/>
      <a href="javascript:;" @click="go('rank')">排行</a>.<a href="javascript:;" @click="go('chat')">聊天</a>.<a href="javascript:;" @click="go('friends')">好友</a>.<a href="javascript:;" @click="go('gang')">帮派</a>.<a href="javascript:;" @click="go('marriage')">结婚</a>.<a href="javascript:;" @click="go('house')">住宅</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a><br/>
      <a href="javascript:;" @click="go('wallet')">流水</a>.<a href="javascript:;" @click="go('blogs')">战报</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== NPC交互 ==================== -->
    <template v-else-if="cur === 'npcview' && npcCur">
      <div v-if="npcCur.img" class="npcimg"><img :src="'/static/hxxy/npc/' + npcCur.img" @error="$event.target.style.display = 'none'" alt="NPC" /></div>
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
      -----------<br/>
      <a href="javascript:;" @click="backToMap">返回西游世界</a><br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 战斗 ==================== -->
    <template v-else-if="cur === 'battle' && bt">
      【战斗】第{{ bt.round + 1 }}回合<br/>
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
        <a href="javascript:;" @click="battleAct('flee')">【逃跑】</a><br/>
        <template v-if="bt.skills && bt.skills.length">
          法术：<a v-for="s in bt.skills" :key="'sk' + s.skill_id" href="javascript:;" @click="battleAct('skill', s.skill_id)">[{{ s.name }}({{ s.mp_cost }})]</a><br/>
        </template>
        <template v-if="battleItems.length">
          用药：<a v-for="b in battleItems" :key="'bi' + b.id" href="javascript:;" @click="battleAct('item', 0, b.id)">[{{ b.name }}×{{ b.count }}]</a><br/>
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
      <div v-if="g.vip > 0" class="npcimg"><img src="/static/hxxy/vip/vip1.png" @error="$event.target.style.display = 'none'" alt="VIP" /></div>
      【{{ g.name }}】{{ g.sect_name }}·{{ g.level }}级<br/>
      <div v-if="g.title_id > 0" class="npcimg"><img :src="titleImg(g.title_id)" @error="$event.target.style.display = 'none'" alt="头衔" /></div>
      经验：{{ g.exp }}/{{ g.exp_need }}<br/>
      气血：{{ g.hp }}-{{ g.max_hp }} 法力：{{ g.mp }}-{{ g.max_mp }}<br/>
      -----------<br/>
      【战斗属性】<br/>
      攻击：{{ at.attrs.atk }} 魔攻：{{ at.attrs.mg }}<br/>
      防御：{{ at.attrs.def }} 魔防：{{ at.attrs.mf }}<br/>
      冰攻：{{ at.attrs.bg }} 火攻：{{ at.attrs.hg }} 雷攻：{{ at.attrs.lg }}<br/>
      冰防：{{ at.attrs.bf }} 火防：{{ at.attrs.hf }} 雷防：{{ at.attrs.lf }}<br/>
      -----------<br/>
      【装备】<br/>
      <template v-if="at.equips.length">
        <div v-for="e in at.equips" :key="'eq' + e.slot">
          {{ e.slot_name }}：<a href="javascript:;" @click="takeoff(e.slot)">{{ e.name }}</a><template v-if="e.star > 0">+{{ e.star }}</template>({{ e.level }}级)
          <a href="javascript:;" @click="takeoff(e.slot)">[卸下]</a><br/>
        </div>
      </template>
      <template v-else><em>未穿戴任何装备，去商店逛逛吧。</em><br/></template>
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
          {{ i + 1 }}.<a href="javascript:;" @click="bagDetail = bagDetail === b.id ? null : b.id">{{ b.name }}</a>×{{ b.count }}<template v-if="b.kind === 'equip' && b.extra && b.extra.star > 0">+{{ b.extra.star }}</template><template v-if="b.bind === 1">(绑定)</template>
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
                <a href="javascript:;" @click="stallInput = { bag_id: b.id, count: 1, price: b.price || 100 }; stallShow = true">[摆摊]</a><br/>
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
      <template v-if="stallShow">
        -----------<br/>
        【摆摊上架】<br/>
        数量：<input v-model.number="stallInput.count" size="3" /> 售价：<input v-model.number="stallInput.price" size="8" />银两<br/>
        <a href="javascript:;" @click="doStallSell">[确认上架]</a> <a href="javascript:;" @click="stallShow = false">[取消]</a><br/>
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

    <!-- ==================== 商店 ==================== -->
    <template v-else-if="cur === 'shop'">
      【商店】银两:{{ g.money }} 金豆:{{ g.beans }}<br/>
      <a v-for="t in shopTabs" :key="'st' + t.k" href="javascript:;" :class="shopKind === t.k ? 'cur' : ''" @click="loadShop(t.k)">[{{ t.n }}]</a><br/>
      -----------<br/>
      <template v-if="shop.goods.length">
        <div v-for="(gd, i) in shop.goods" :key="'gd' + i">
          {{ i + 1 }}.<a href="javascript:;" @click="shopDetail = shopDetail === gd.kind + gd.ref_id ? null : gd.kind + gd.ref_id">{{ gd.name }}</a><template v-if="gd.level > 0">({{ gd.level }}级)</template><br/>
          <template v-if="shopDetail === gd.kind + gd.ref_id">
            <span class="gray">{{ gd.desc }}</span><br/>
            <template v-if="gd.kind === 'equip' && gd.category">部位：{{ slotName(gd.category) }}<br/></template>
          </template>
          <a href="javascript:;" @click="buy(gd, 'money')">[{{ gd.price }}银两购买]</a><template v-if="gd.bean_price > 0"> <a href="javascript:;" @click="buy(gd, 'beans')">[{{ gd.bean_price }}金豆购买]</a></template><br/>
        </div>
      </template>
      <template v-else><em>本店暂无货物。</em><br/></template>
      <template v-if="shop.pets && shop.pets.length">
        -----------<br/>
        【宠物柜台】(金豆购买)<br/>
        <div v-for="(pt, i) in shop.pets" :key="'sp' + i">
          {{ i + 1 }}.{{ pt.name }}({{ pt.level }}级)——{{ pt.bean_price }}金豆
          <a href="javascript:;" @click="buyPet(pt)">[购买]</a><br/>
        </div>
      </template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('bag')">行囊</a>.<a href="javascript:;" @click="go('stalls')">摆摊</a>.<a href="javascript:;" @click="go('vip')">充值</a><br/>
    </template>

    <!-- ==================== 银行 ==================== -->
    <template v-else-if="cur === 'bank'">
      【银行】<br/>
      随身银两：{{ g.money }}　存款：{{ g.bank }}<br/>
      <form @submit.prevent="bankOp('in')">
        存入：<input v-model.number="bankAmount" size="10" /> <input type="submit" value="存款" />
      </form>
      <form @submit.prevent="bankOp('out')">
        取出：<input v-model.number="bankAmount" size="10" /> <input type="submit" value="取款" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('wallet')">流水</a><br/>
    </template>

    <!-- ==================== 任务 ==================== -->
    <template v-else-if="cur === 'quests'">
      【任务】<br/>
      【进行中】<br/>
      <template v-if="qst.active.length">
        <div v-for="q in qst.active" :key="'qa' + q.quest_id">
          {{ q.name }}({{ q.type === 'hunt' ? '狩猎' : q.type === 'collect' ? '收集' : '对话' }} {{ q.progress }}/{{ q.count }})<br/>
          <span class="gray">{{ q.desc }}</span><br/>
          奖励：经验{{ q.exp }} 银两{{ q.money }}
          <a v-if="q.status === 2 || q.type !== 'hunt'" href="javascript:;" @click="questSubmit(q)">[提交]</a><br/>
        </div>
      </template>
      <template v-else><em>暂无进行中的任务。</em><br/></template>
      -----------<br/>
      【可接任务】<br/>
      <template v-if="qst.available.length">
        <div v-for="q in qst.available" :key="'qv' + q.quest_id">
          <a href="javascript:;" @click="questAccept(q)">{{ q.name }}</a>({{ q.type === 'hunt' ? '狩猎' + q.count + '只' + q.target : q.type === 'collect' ? '收集' + q.count + '个' + q.target : '拜访' + q.target }})<br/>
          <span class="gray">{{ q.desc }}</span><br/>
          奖励：经验{{ q.exp }} 银两{{ q.money }}<br/>
        </div>
      </template>
      <template v-else><em>暂无可接任务，升级后再来看看。</em><br/></template>
      -----------<br/>
      【已完成】<br/>
      <div v-for="q in qst.done" :key="'qd' + q.quest_id">{{ q.name }}<span class="gray">(已完成)</span><br/></div>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('map')">世界</a><br/>
    </template>

    <!-- ==================== 副本 ==================== -->
    <template v-else-if="cur === 'dungeons'">
      【副本】<br/>
      <div v-for="(dg, i) in dungeons" :key="'dg' + dg.dungeon_id">
        {{ i + 1 }}.{{ dg.name }}({{ dg.min_level }}级·共{{ dg.floors }}层·每日{{ dg.daily }}次·今日已用{{ dg.used_today }})<br/>
        <span class="gray">{{ dg.desc }}</span><br/>
        进度：第{{ dg.floor }}/{{ dg.floors }}层
        <template v-if="dg.locked"><span class="red">[等级不足]</span></template>
        <template v-else-if="dg.used_today >= dg.daily"><span class="gray">[今日次数已用完]</span></template>
        <template v-else><a href="javascript:;" @click="enterDungeon(dg)">[进入副本]</a></template><br/>
        ----------<br/>
      </div>
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

    <!-- ==================== 修炼 ==================== -->
    <template v-else-if="cur === 'cultivate'">
      【修炼】<br/>
      状态：{{ cult.switch === 1 ? '开启中' : '未开启' }}<br/>
      修炼池：{{ cult.exp }}/{{ cult.cap }}<br/>
      <span class="gray">{{ cult.desc }}</span><br/>
      <a href="javascript:;" @click="cultToggle">{{ cult.switch === 1 ? '[关闭修炼]' : '[开启修炼]' }}</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('bag')">行囊</a><br/>
    </template>

    <!-- ==================== 头衔 ==================== -->
    <template v-else-if="cur === 'titles'">
      【头衔】当前佩戴：<template v-if="titles.worn > 0">{{ wornTitleName }}</template><template v-else>无</template><br/>
      【我的头衔】<br/>
      <template v-if="titles.mine.length">
        <div v-for="t in titles.mine" :key="'tm' + t.title_id">
          <div class="npcimg"><img :src="titleImg(t.title_id)" @error="$event.target.style.display = 'none'" alt="头衔" /></div>
          <a href="javascript:;" @click="wearTitle(t)">{{ t.name }}</a><template v-if="titles.worn === t.title_id"><span class="green">[佩戴中]</span></template><br/>
          <span class="gray">{{ t.desc }}</span><br/>
        </div>
      </template>
      <template v-else><em>还未激活任何头衔。</em><br/></template>
      -----------<br/>
      【头衔商店】<br/>
      <div v-for="t in titles.store" :key="'ts' + t.title_id">
        <a href="javascript:;" @click="activateTitle(t)">{{ t.name }}</a>({{ t.price }}银两激活)<br/>
        <span class="gray">{{ t.desc }}</span><br/>
      </div>
      <a href="javascript:;" @click="wearTitle({ title_id: 0 })">[摘下头衔]</a><br/>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('attrs')">状态</a><br/>
    </template>

    <!-- ==================== 签到 ==================== -->
    <template v-else-if="cur === 'signin'">
      【每日签到】<br/>
      连续签到奖励递增：100/200/300/400/500/600银两，第7天额外+10金豆！<br/>
      <a href="javascript:;" @click="doSignin">[立即签到]</a><br/>
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

    <!-- ==================== 好友 ==================== -->
    <template v-else-if="cur === 'friends'">
      【好友】<br/>
      【我的好友】<br/>
      <template v-if="frd.friends.length">
        <div v-for="f in frd.friends" :key="'fd' + f.player_id"><a href="javascript:;" @click="viewPlayer(f.player_id)">{{ f.name }}</a><br/></div>
      </template>
      <template v-else><em>还没有好友。</em><br/></template>
      <template v-if="frd.applies.length">
        【好友申请】<br/>
        <div v-for="f in frd.applies" :key="'fa' + f.apply_id"><a href="javascript:;" @click="viewPlayer(f.player_id)">{{ f.name }}</a> <a href="javascript:;" @click="agreeFriend(f)">[同意]</a><br/></div>
        -----------<br/>
      </template>
      【添加好友】<br/>
      <form @submit.prevent="addFriend">
        对方名字：<input v-model="friendName" maxlength="12" />
        <input type="submit" value="发送申请" />
      </form>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('chat')">聊天</a><br/>
    </template>

    <!-- ==================== 帮派 ==================== -->
    <template v-else-if="cur === 'gang'">
      【帮派】<br/>
      <template v-if="gang.my_gang && gang.my_gang.gang_id">
        【{{ gang.my_gang.name }}】({{ gang.my_gang.level }}级)——你是{{ gang.my_gang.role }}<br/>
        公告：<span class="gray">{{ gang.my_gang.notice }}</span><br/>
        帮派资金：{{ gang.my_gang.money }} 你的贡献：{{ gang.my_gang.contribution }}<br/>
        <form @submit.prevent="gangDonate">
          捐献：<input v-model.number="gangDonateAmount" size="10" />银两 <input type="submit" value="捐献" />
        </form>
        【帮派成员】<br/>
        <div v-for="m in gang.my_gang.members" :key="'gm' + m.player_id"><a href="javascript:;" @click="viewPlayer(m.player_id)">{{ m.name }}</a>({{ m.role }}·贡献{{ m.contribution }})<br/></div>
        <a href="javascript:;" @click="gangQuit">[退出帮派]</a><br/>
      </template>
      <template v-else>
        <em>你还没有加入帮派。</em><br/>
        【创建帮派】(10000银两)<br/>
        <form @submit.prevent="gangNew">
          帮派名：<input v-model="gangNameInput" maxlength="10" />
          <input type="submit" value="创建" />
        </form>
        -----------<br/>
      </template>
      【帮派列表】<br/>
      <div v-for="gp in gang.gangs" :key="'gg' + gp.gang_id">
        {{ gp.name }}({{ gp.level }}级)——<span class="gray">{{ gp.notice }}</span>
        <template v-if="!gang.my_gang || !gang.my_gang.gang_id"><a href="javascript:;" @click="gangJoin(gp)">[加入]</a></template><br/>
      </div>
      <a href="javascript:;" @click="go('home')">首页</a><br/>
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

    <!-- ==================== 摆摊 ==================== -->
    <template v-else-if="cur === 'stalls'">
      【摆摊市场】银两:{{ g.money }}<br/>
      <template v-if="stallList.length">
        <div v-for="(s, i) in stallList" :key="'sl' + s.stall_id">
          {{ i + 1 }}.{{ s.name }}×{{ s.count }}——{{ s.price }}银两(卖家:<a href="javascript:;" class="nk" @click="viewPlayer(s.seller_id)">{{ s.seller }}</a>)
          <template v-if="s.seller_id === playerID"><a href="javascript:;" @click="stallCancel(s)">[下架]</a></template>
          <template v-else><a href="javascript:;" @click="stallBuy(s)">[购买]</a></template><br/>
        </div>
      </template>
      <template v-else><em>市场上暂无商品。去行囊把闲置物品上架吧！</em><br/></template>
      -----------<br/>
      <a href="javascript:;" @click="go('home')">首页</a>.<a href="javascript:;" @click="go('bag')">行囊</a><br/>
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

    <!-- ==================== 充值 ==================== -->
    <template v-else-if="cur === 'vip'">
      【充值】<br/>
      金豆：{{ g.beans }}　VIP练级祝福：{{ g.vip }}分钟(打怪经验1.5倍)<br/>
      <form @submit.prevent="doRecharge">
        充值码：<input v-model="rechargeCode" size="12" />
        <input type="submit" value="兑换" />
      </form>
      <em>演示充值码：XY666(+10金豆) / VIP666(+30分钟祝福)</em><br/>
      -----------<br/>
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
      <template v-if="teyun.list.length">
        <div v-for="(t, i) in teyun.list" :key="'ty' + i">
          {{ i + 1 }}.<a href="javascript:;" @click="teyunGo(t)">{{ t.name }}</a><br/>
        </div>
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

    <!-- ==================== 玩家资料 ==================== -->
    <template v-else-if="cur === 'playerview' && pv">
      【玩家资料】<br/>
      <template v-if="!pv.is_me">
        <a href="javascript:;" @click="openPm(pv.player_id)">[私聊]</a>
        <a href="javascript:;" @click="addFriendByID(pv.player_id)">[加好友]</a>
        <a href="javascript:;" @click="teamInvite(pv)">[组队]</a>
        <a href="javascript:;" @click="fightPlayer(pv)">[比武]</a><br/>
        <a href="javascript:;" @click="inviteVisitHouse(pv)">[邀请参观住宅]</a>
        <a v-if="gang.my_gang.gang_id && (gang.my_gang.role === '帮主' || gang.my_gang.role === '长老')" href="javascript:;" @click="inviteJoinGang(pv)">[邀请入帮]</a><br/>
      </template>
      -----------<br/>
      昵称：{{ pv.name }}({{ pv.sex === 2 ? '女' : '男' }})<br/>
      门派：{{ pv.sect_name }} 等级：{{ pv.level }}级<br/>
      <template v-if="pv.gang">帮派：{{ pv.gang }}<br/></template>
      <template v-if="pv.title">头衔：{{ pv.title }}<br/></template>
      所在地：{{ pv.node_name }}<br/>
      比武胜场：{{ pv.wins }}场 恶名：{{ pv.emz }}<br/>
      通天塔最高：第{{ pv.tower_best }}层<br/>
      -----------<br/>
      <a href="javascript:;" @click="backPv">返回</a>.<a href="javascript:;" @click="go('home')">首页</a><br/>
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
        <a v-if="gz.my_gang_id > 0 && gz.my_role === 2 && gz.zc_id !== 6" href="javascript:;" @click="gzSignup">[帮主报名今日国战]</a><br/>
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
  name: 'Xiyou',
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
      cf: { name: '', sex: 1, sect: 1 },
      node: {},
      enemies: [],
      mapNpcs: [],
      mapImgOk: true,
      homeMsgs: [],
      nearby: [],
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
      shop: { goods: [], pets: null },
      shopKind: 'medicine',
      shopDetail: null,
      shopTabs: [
        { k: 'medicine', n: '药店' },
        { k: 'weapon', n: '武器' },
        { k: 'armor', n: '防具' },
        { k: 'jewel', n: '首饰' },
        { k: 'grocery', n: '杂货' },
        { k: 'pet', n: '宠物店' },
      ],
      bankAmount: 0,
      qst: { available: [], active: [], done: [] },
      dungeons: [],
      bossList: [],
      cult: { switch: 0, exp: 0, cap: 0, desc: '' },
      titles: { mine: [], store: [], worn: 0 },
      rankType: 'level',
      rankList: [],
      rankTabs: [{ k: 'level', n: '等级榜' }, { k: 'money', n: '银两榜' }, { k: 'pets', n: '宠物榜' }],
      chatList: [],
      chatInput: '',
      frd: { friends: [], applies: [] },
      friendName: '',
      gang: { my_gang: {}, gangs: [] },
      gangNameInput: '',
      gangDonateAmount: 0,
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
      stallList: [],
      stallShow: false,
      stallInput: { bag_id: 0, count: 1, price: 100 },
      walletLogs: [],
      blogLogs: [],
      rechargeCode: '',
      tower: { floor: 0, best: 0 },
      arena: { rank: [], me: { rank: 0, wins: 0, today: 0, limit: 5 } },
      funRoll: null,
      teyun: { fu: 0, list: [] },
      pv: null,
      pvFrom: 'home',
      gz: null,
      team: null,
      tipMsg: '',
      _tipTimer: null,
    }
  },
  computed: {
    playerID() { return this.g.id || 0 },
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
      if (this._tipTimer) clearTimeout(this._tipTimer)
      this._tipTimer = setTimeout(() => { this.tipMsg = '' }, 2200)
    },
    fmtTime(t) { return (t || '').substring(5, 16) },
    slotName(c) { return { 1: '法宝', 2: '坐骑', 3: '武器', 4: '护甲', 5: '头盔', 6: '靴子', 7: '项链', 8: '手镯' }[c] || '装备' },
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
      if (v === 'home') { this.refreshPlayer(); this.loadHome() }
      if (v === 'map') this.loadState()
      if (v === 'attrs') this.loadAttrs()
      if (v === 'bag') this.loadBag()
      if (v === 'skills') this.loadSkills()
      if (v === 'pets') this.loadPets()
      if (v === 'shop') this.loadShop(this.shopKind)
      if (v === 'quests') this.loadQuests()
      if (v === 'dungeons') this.loadDungeons()
      if (v === 'bosses') this.loadBosses()
      if (v === 'cultivate') this.loadCult()
      if (v === 'titles') this.loadTitles()
      if (v === 'rank') this.loadRank(this.rankType)
      if (v === 'chat') this.loadChat()
      if (v === 'friends') this.loadFriends()
      if (v === 'gang') this.loadGang()
      if (v === 'marriage') this.loadMarriage()
      if (v === 'house') this.loadHouse()
      if (v === 'stalls') this.loadStalls()
      if (v === 'wallet') this.loadWallet()
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
    // ---------- 地图 ----------
    async loadState() {
      const r = await api.get('/games/hxxy/state')
      if (r.code === 0) {
        this.g = r.data.player
        this.node = r.data.node
        this.enemies = r.data.enemies || []
        this.mapNpcs = r.data.npcs || []
        this.mapImgOk = true
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
    async move(dir) {
      const r = await api.post('/games/hxxy/move', { dir })
      if (r.code === 0) {
        this.g = r.data.player
        this.loadState()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
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
    async startBattle(npcID) {
      const r = await api.post('/games/hxxy/battle/start', { npc_id: npcID })
      if (r.code === 0) {
        this.bt = r.data
        this.cur = 'battle'
        this.loadBattleItems()
        window.scrollTo(0, 0)
      } else {
        this.tip(r.msg)
      }
    },
    async loadBattleItems() {
      const r = await api.get('/games/hxxy/bag')
      if (r.code === 0) {
        this.battleItems = (r.data.items || []).filter(b => b.kind === 'item' && b.effect && (b.effect.hp > 0 || b.effect.mp > 0))
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
    async doStallSell() {
      const r = await api.post('/games/hxxy/stall/sell', { bag_id: this.stallInput.bag_id, count: this.stallInput.count, price: this.stallInput.price })
      this.stallShow = false
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadBag()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 技能 ----------
    async loadSkills() {
      const r = await api.get('/games/hxxy/skills')
      if (r.code === 0) this.sk = r.data
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
      this.shopDetail = null
      const r = await api.get('/games/hxxy/shop/' + kind)
      if (r.code === 0) {
        this.shop = { goods: r.data.goods || [], pets: r.data.pets || null }
      } else {
        this.tip(r.msg)
      }
    },
    async buy(gd, currency) {
      const r = await api.post('/games/hxxy/shop/buy', { kind: gd.kind, ref_id: gd.ref_id, count: 1, currency })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async buyPet(pt) {
      const r = await api.post('/games/hxxy/shop/buy', { kind: 'pet', ref_id: pt.species_id, count: 1, currency: 'beans' })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 银行 ----------
    async bankOp(dir) {
      const amt = Number(this.bankAmount)
      if (!amt || amt <= 0) { this.tip('请输入正确金额'); return }
      const url = dir === 'in' ? '/games/hxxy/bank/deposit' : '/games/hxxy/bank/withdraw'
      const r = await api.post(url, { amount: amt })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.bankAmount = 0
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 任务 ----------
    async loadQuests() {
      const r = await api.get('/games/hxxy/quests')
      if (r.code === 0) this.qst = r.data
    },
    async questAccept(q) {
      const r = await api.post('/games/hxxy/quests/accept', { quest_id: q.quest_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadQuests()
      } else {
        this.tip(r.msg)
      }
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
    // ---------- 头衔 ----------
    async loadTitles() {
      const r = await api.get('/games/hxxy/titles')
      if (r.code === 0) this.titles = r.data
    },
    async activateTitle(t) {
      const r = await api.post('/games/hxxy/titles/activate', { title_id: t.title_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadTitles()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async wearTitle(t) {
      const r = await api.post('/games/hxxy/titles/wear', { title_id: t.title_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadTitles()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 签到 ----------
    async doSignin() {
      const r = await api.post('/games/hxxy/signin', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
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
    async addFriend() {
      if (!this.friendName.trim()) { this.tip('请输入对方名字'); return }
      const r = await api.post('/games/hxxy/friends/add', { name: this.friendName.trim() })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.friendName = ''
        this.loadFriends()
      } else {
        this.tip(r.msg)
      }
    },
    async agreeFriend(f) {
      const r = await api.post('/games/hxxy/friends/agree', { apply_id: f.apply_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadFriends()
      } else {
        this.tip(r.msg)
      }
    },
    // ---------- 帮派 ----------
    async loadGang() {
      const r = await api.get('/games/hxxy/gang')
      if (r.code === 0) this.gang = r.data
    },
    async gangNew() {
      if (!this.gangNameInput.trim()) { this.tip('请输入帮派名'); return }
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
    async gangQuit() {
      if (!window.confirm('确定退出帮派吗？')) return
      const r = await api.post('/games/hxxy/gang/leave', {})
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadGang()
      } else {
        this.tip(r.msg)
      }
    },
    async gangDonate() {
      const amt = Number(this.gangDonateAmount)
      if (!amt || amt <= 0) { this.tip('请输入正确金额'); return }
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
    // ---------- 摆摊 ----------
    async loadStalls() {
      const r = await api.get('/games/hxxy/stalls')
      if (r.code === 0) this.stallList = r.data.stalls || []
    },
    async stallBuy(s) {
      if (!window.confirm('花 ' + s.price + ' 银两购买【' + s.name + '】×' + s.count + '？')) return
      const r = await api.post('/games/hxxy/stall/buy', { stall_id: s.stall_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadStalls()
        this.refreshPlayer()
      } else {
        this.tip(r.msg)
      }
    },
    async stallCancel(s) {
      const r = await api.post('/games/hxxy/stall/cancel', { stall_id: s.stall_id })
      if (r.code === 0) {
        this.tip(r.data.msg)
        this.loadStalls()
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
.cur { color: #f60; font-weight: bold; }
.nk { color: #c00; }
.logo { text-align: left; margin: 4px 0; }
.mapimg img { max-width: 240px; width: 100%; height: auto; display: block; margin: 4px 0; }
.npcimg img { max-width: 120px; width: auto; height: auto; display: block; margin: 4px 0; }
.xy-tip { position: fixed; left: 50%; top: 20%; transform: translateX(-50%); background: rgba(0, 0, 0, 0.75); color: #fff; padding: 8px 16px; border-radius: 4px; z-index: 200; }
</style>
