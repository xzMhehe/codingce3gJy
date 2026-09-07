<template>
  <div>
    <!-- 活动公告（参考站 .note：游戏公告=论坛帖子 + 七日签到） -->
    <div class="note" v-if="cur === 'garden'">
      <a href="javascript:;" @click="openNotice">游戏公告：{{ noticeTitle }}</a><br/>
      <img class="noteico" src="/static/image/vipqq.jpg" alt="." />&nbsp;回家的礼物、一天都不能少({{ signCount }}/7) <a href="javascript:;" @click="openCheck">签到</a><br/>
    </div>

    <!-- 顶部导航（对齐参考站 .bar：花园 好友 花房 魔法屋 活动） -->
    <div class="bar garden-nav">
      <a :class="{ cur: cur === 'garden' }" href="javascript:;" @click="switchTab('garden')">花园</a> <a :class="{ cur: cur === 'friend' }" href="javascript:;" @click="switchTab('friend')">好友</a> <a :class="{ cur: cur === 'basket' }" href="javascript:;" @click="switchTab('basket')">花房</a> <a :class="{ cur: cur === 'room' }" href="javascript:;" @click="switchTab('room')">魔法屋</a> <a :class="{ cur: cur === 'active' }" href="javascript:;" @click="switchTab('active')">活动</a><img src="/static/picture/garden/hot.gif" alt="." />
    </div>

    <div class="g-main">
      <!-- ============ 花园主页（复刻 my_garden.aspx） ============ -->
      <template v-if="cur === 'garden'">
        <div class="name userline">{{ nick }} <img class="bicon" src="/static/image/noble_2_1.gif" alt="." />({{ g.level }}级)</div>
        <div class="module-content deep">{{ g.level }}级 {{ g.level_name }}(经验 {{ g.point }}/{{ g.need }})</div>
        <div class="module-content">
          精灵花册:<a href="javascript:;" @click="openElves">{{ elvesUnlocked }}/{{ elvesTotal }}</a><br/>
          <br/>花之图谱:<a href="javascript:;" @click="openMap">{{ g.common + g.festival + g.scarce }}/{{ g.map_total }}</a><br/>珍稀:{{ g.scarce }} 独特:{{ g.festival }} 普通:{{ g.common }}<br/>
        </div>
        <div class="recent-maps">
          <template v-for="i in 7">
            <img v-if="recentMaps[i-1]" :key="i" :src="'/static/picture/garden/' + (recentMaps[i-1].img || ('m_s_' + recentMaps[i-1].id + '.gif'))" :title="recentMaps[i-1].name" alt="." />
            <img v-else :key="i" src="/static/picture/garden/m_s.gif" alt="." />
          </template>
        </div>
        <br/>
        <div class="module-content">背包中没有魔力道具，无法使用一键功能！</div><br/>

        <div class="name">花盆({{ g.lands }})<a href="javascript:;" @click="load">刷新</a></div>
        <div class="list">
          <div class="row" v-if="emptyCount > 0">空花盆:{{ emptyCount }} [<a href="javascript:;" @click="openSowFirst">播种</a>]<br/></div>
          <div class="row" v-for="p in plots" :key="'r' + p.plot">
            <template v-if="p.stage === 0">
              空花盆 [<a href="javascript:;" @click="openSow(p)">播种</a>]<br/>
            </template>
            <template v-else-if="p.stage === 4">
              <a href="javascript:;" @click="openPlotPage(p)">{{ p.name }}</a>[<a href="javascript:;" @click="openHarvest(p)">收获</a>]<br/>
              (产{{ p.yield }}剩{{ p.amount }})<br/>
            </template>
            <template v-else>
              <a href="javascript:;" @click="openPlotPage(p)">{{ p.seed }}</a>
              <span v-if="p.stage === 1"><a v-if="p.need_water" href="javascript:;" @click="care(p, 'water')">[浇水]</a></span>
              <span v-else-if="p.stage === 2"><a v-if="p.need_weed" href="javascript:;" @click="care(p, 'weed')">[锄草]</a></span>
              <span v-else-if="p.stage === 3"><a v-if="p.need_pest" href="javascript:;" @click="care(p, 'pest')">[捉虫]</a></span>
              <br/>({{ p.remain_txt }})<br/>
            </template>
          </div>
          <div class="row" v-if="g.lands < 9">
            <a href="javascript:;" @click="addLand">添置新花盆</a><br/>
          </div>
        </div>
        <br/>

        <div class="name">消息<a href="javascript:;" @click="openMsgs">({{ msgs.length }})</a></div>
        <div class="list">
          <div class="row" v-for="(m, i) in msgs.slice(0, 3)" :key="m.id">
            {{ i + 1 }}.({{ m.time_txt || '' }})<b>{{ m.nick }}</b> {{ m.msg }}<br/>
          </div>
          <div class="row" v-if="!msgs.length">您没有消息.<br/></div>
          <div class="row" v-if="msgs.length > 3"><a href="javascript:;" @click="openMsgs">查看更多&gt;&gt;</a><br/></div>
        </div>
        <br/>

        <div class="name">花园设施</div>
        <div class="row">
          公告板:{{ g.notice || '欢迎来到我的花园！' }} <a href="javascript:;" @click="openSetting(2)">更改</a><br/>
        </div>
        <div class="row">
          花园名:{{ g.name }} <a href="javascript:;" @click="openSetting(1)">更改</a><br/>
        </div>
        <br/>

        <div class="module-title"><a href="javascript:;" @click="switchTab('bag')">背包</a>.<a href="javascript:;" @click="switchTab('shop')">商店</a>.<a href="javascript:;" @click="switchTab('rank')">排行</a>.<a href="javascript:;" @click="switchTab('help')">帮助</a>.<a href="javascript:;" @click="goForum">论坛</a><br/></div>
      </template>

      <!-- ============ 花之图谱（复刻 map_list.aspx） ============ -->
      <template v-else-if="cur === 'map'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;花之图谱<br/></div>
        <div class="module-content">花园世界目前已知花种{{ g.map_total }}种，您成功培育出了{{ g.common + g.festival + g.scarce }}种<br/></div>
        <div class="name">
          普通({{ g.common }}) <a :class="{ cur: mapTy === 1 }" href="javascript:;" @click="loadMap(1)">独特({{ g.festival }})</a> <a :class="{ cur: mapTy === 2 }" href="javascript:;" @click="loadMap(2)">珍稀({{ g.scarce }})</a><br/>
        </div>
        <form @submit.prevent="searchMap">
          <input type="text" v-model.trim="mapWd" maxlength="8" value="" />
          <input type="submit" value="搜索" />
        </form>
        <div class="list">
          <div class="row" v-for="m in pagedMaps" :key="m.id">
            <img :src="'/static/picture/garden/' + (m.got ? (m.img || ('m_s_' + m.id + '.gif')) : 'm_s.gif')" alt="." />
            <a href="javascript:;" @click="openMapDetail(m)">{{ m.name }}</a> <span v-if="m.got" class="got-tag">已点亮</span><br/>
          </div>
          <div class="row" v-if="!pagedMaps.length">未找到相关花种<br/></div>
          <div class="row">
            <a v-if="mapPage > 1" href="javascript:;" @click="mapPage--">上页</a>
            <a v-if="mapPage < mapPages" href="javascript:;" @click="mapPage++">下页</a>
            <span v-if="filteredMaps.length">(第<b>{{ mapPage }}</b>/{{ mapPages }}页/共{{ filteredMaps.length }}条记录)<br/></span>
          </div>
        </div>
        <a href="javascript:;" @click="loadMap(mapTy)">查看未点亮图谱</a><br/>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 精灵花册（复刻 mybooklist.aspx） ============ -->
      <template v-else-if="cur === 'elves'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a> &gt; 精灵花册<br/></div>
        <div class="module-content">威震天下的勇士啊,你已开启了{{ elvesUnlocked }}个精灵花册(目前共有{{ elvesTotal }}个)<br/></div>
        <div class="name">精灵花册({{ elvesTotal }})<br/></div>
        <div class="list">
          <div class="row" v-for="e in elfList" :key="e.id">
            <img :src="'/static/picture/garden/' + (e.img || ('elf_' + e.id + '.png'))" class="elf-prev" alt="." /><a href="javascript:;" @click="openElfDetail(e)">{{ e.name }}</a>({{ e.unlocked ? '已开启' : '未开启' }})<br/>
          </div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 商店（复刻 seed_list.aspx / dz_list.aspx） ============ -->
      <template v-else-if="cur === 'shop'">
        <div class="bar sub">【花园商店】<br/></div>
        <div class="module-title"><a :class="{ cur: shopTy === 0 }" href="javascript:;" @click="switchShop(0)">普通花种</a>|<a :class="{ cur: shopTy === 1 }" href="javascript:;" @click="switchShop(1)">独特花种</a>|<a :class="{ cur: shopTy === 2 }" href="javascript:;" @click="switchShop(2)">道具</a><br/></div>
        <template v-if="shopTy === 2">
          道具名|价格<br/>
          <div class="list">
            <div class="row" v-for="(s, i) in pagedShop" :key="s.id">
              {{ (shopPage - 1) * 10 + i + 1 }}.{{ s.name }} {{ s.price }}G币 <a href="javascript:;" @click="openSeedDetail(s)">[购买]</a><br/>
            </div>
            <div class="row" v-if="!pagedShop.length">商店暂无商品<br/></div>
            <div class="row" v-if="shop.length">
              <a v-if="shopPage > 1" href="javascript:;" @click="shopPage--">上页</a>
              <a v-if="shopPage < shopPages" href="javascript:;" @click="shopPage++">下页</a>
              (第<b>{{ shopPage }}</b>/{{ shopPages }}页/共{{ shop.length }}条记录)<br/>
            </div>
          </div>
        </template>
        <template v-else>
          花种|价格|等级|{{ shopTy === 1 ? '独特' : '普通' }}<br/>
          <div class="list">
            <div class="row" v-for="s in pagedShop" :key="s.id">
              <img :src="'/static/picture/garden/' + (s.img || ('s_s_' + s.id + '.gif'))" alt="." /><a href="javascript:;" @click="openSeedDetail(s)">{{ s.name }}</a> {{ s.level }}级 {{ s.price }}G币
              <template v-if="s.dtype === 0"><a href="javascript:;" @click="openSeedDetail(s)">[购买]</a></template>
              <template v-else><a href="javascript:;" @click="switchTab('room')">[魔法屋合成]</a></template><br/>
            </div>
            <div class="row" v-if="!pagedShop.length">{{ shopTy === 1 ? '独特花种需在魔法屋中合成获得，敬请期待直接购买！' : '商店暂无商品<br/>' }}<br/></div>
            <div class="row" v-if="shop.length">
              <a v-if="shopPage > 1" href="javascript:;" @click="shopPage--">上页</a>
              <a v-if="shopPage < shopPages" href="javascript:;" @click="shopPage++">下页</a>
              (第<b>{{ shopPage }}</b>/{{ shopPages }}页/共{{ shop.length }}条记录)<br/>
            </div>
          </div>
        </template>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 背包（复刻 my_bag.aspx） ============ -->
      <template v-else-if="cur === 'bag'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;背包<br/></div>
        <div class="name">我的背包(G币 {{ coins }})</div>
        花种|道具<br/>
        <div class="list">
          <div class="row" v-for="b in bagSeeds" :key="b.seed_id">
            <img :src="'/static/picture/garden/' + (b.img || ('s_s_' + b.seed_id + '.gif'))" class="map-icon" alt="." /><a href="javascript:;" @click="openBagSeed(b)">{{ b.seed_name }}</a>×{{ b.count }}
            <a href="javascript:;" @click="openSowSeed(b)">[播种]</a><br/>
          </div>
          <div class="row" v-for="b in bagItems" :key="'d' + b.seed_id">
            {{ b.seed_name }}×{{ b.count }}<br/>
          </div>
          <div class="row" v-if="!bag.length">背包空空的，去商店买点种子吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('shop')">去商店</a><br/>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 魔法屋（复刻 room_list.aspx） ============ -->
      <template v-else-if="cur === 'room'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;魔法屋<br/></div>
        <div class="module-content">用鲜花合成珍稀花种，金色银色花种仅此可得！<br/></div>
        <form @submit.prevent="searchRoom">
          <input type="text" v-model.trim="roomWd" maxlength="8" value="" />
          <input type="submit" value="搜索" />
        </form>
        <div class="list">
          <div class="row" v-for="(s, i) in pagedRoom" :key="s.seed_id">
            {{ (roomPage - 1) * 10 + i + 1 }}.<a href="javascript:;" @click="openRoomDetail(s)">{{ s.name }}</a> <br/>
            <span v-for="(m, j) in s.mats" :key="j">{{ m.flower }}({{ m.need }}/{{ m.have }})<br/></span>
            <span v-if="s.can" class="ok-txt"><a href="javascript:;" @click="mix(s)">[合成]</a></span>
            <span v-else>所需花朵不足，您目前还不能合成。<br/></span>
          </div>
          <div class="row" v-if="!pagedRoom.length">魔法屋空空如也<br/></div>
          <div class="row" v-if="filteredRoom.length">
            <a v-if="roomPage > 1" href="javascript:;" @click="roomPage--">上页</a>
            <a v-if="roomPage < roomPages" href="javascript:;" @click="roomPage++">下页</a>
            (第<b>{{ roomPage }}</b>/{{ roomPages }}页/共{{ filteredRoom.length }}条记录)<br/>
          </div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 排行（复刻 level.aspx） ============ -->
      <template v-else-if="cur === 'rank'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;排行<br/></div>
        <div class="module-content">花园世界目前已知花种{{ g.map_total }}种，您成功培育出了{{ g.common + g.festival + g.scarce }}种。<br/></div>
        <div class="module-content">您的排行：{{ myRank }}<br/></div>
        <div class="module-title">排名|昵称|等级|花盆|点亮<br/></div>
        <div class="list">
          <div class="row" v-for="r in rankList" :key="r.rank">
            {{ r.rank }}.<img src="/static/image/noble_2_2.gif" alt="." /><a href="javascript:;" @click="visitUid(r.user_id)">{{ r.nickname }}</a> | {{ r.level }} | {{ r.lands }} | {{ r.map_got }}<br/>
          </div>
          <div class="row" v-if="!rankList.length">暂无排行数据<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 好友（复刻 my_friend.aspx） ============ -->
      <template v-else-if="cur === 'friend'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;好友<br/></div>
        <div class="list">
          <div class="row" v-for="(f, i) in friends" :key="f.id">
            {{ i + 1 }}.<a href="javascript:;" @click="visitGarden(f)">{{ f.nickname }}</a><br/>
          </div>
          <div class="row" v-if="!friends.length">您还没有好友，快去社区添加吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 参观花园（复刻 garden.asp） ============ -->
      <template v-else-if="cur === 'visit'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('friend')">好友</a>&gt;{{ visitNick }}的花园<br/></div>
        <div class="name">主人:{{ visitNick }} [<a href="javascript:;" @click="openGiftToVisit">送花</a>] <img class="bicon" src="/static/image/noble_2_1.gif" alt="." />({{ vg.level }}级)</div>
        <div class="module-content deep">{{ vg.level }}级 {{ vg.level_name }}(经验 {{ vg.point }}/{{ vg.need }})</div>
        <div class="module-content">
          花之图谱:<a href="javascript:;" @click="msg = '只能查看自己的图谱'">{{ vg.common + vg.festival + vg.scarce }}/{{ vg.map_total }}</a><br/>珍稀:{{ vg.scarce }} 独特:{{ vg.festival }} 普通:{{ vg.common }}<br/>
        </div>
        <div class="name">花盆({{ vg.lands }})<a href="javascript:;" @click="visitGarden({ id: vg.user_id })">刷新</a></div>
        <div class="list">
          <div class="row" v-for="p in vplots" :key="'v' + p.plot">
            <template v-if="p.stage === 0">
              空花盆<br/>
            </template>
            <template v-else-if="p.stage === 4">
              {{ p.name }}[<a v-if="p.can_pick" href="javascript:;" @click="pick(p)">采摘</a><span v-else>已摘过</span>]<br/>
              (产{{ p.yield }}剩{{ p.amount }})<br/>
            </template>
            <template v-else>
              {{ p.seed }}<span v-if="p.stage === 1"><a v-if="p.need_water" href="javascript:;" @click="careVisit(p, 'water')">[浇水]</a></span><span v-else-if="p.stage === 2"><a v-if="p.need_weed" href="javascript:;" @click="careVisit(p, 'weed')">[锄草]</a></span><span v-else-if="p.stage === 3"><a v-if="p.need_pest" href="javascript:;" @click="careVisit(p, 'pest')">[捉虫]</a></span><br/>({{ p.remain_txt }})<br/>
            </template>
          </div>
        </div>
        <a href="javascript:;" @click="switchTab('friend')">返回好友</a><br/>
      </template>

      <!-- ============ 花房（复刻 basket_list.aspx） ============ -->
      <template v-else-if="cur === 'basket'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;花房<br/></div>
        <div class="name">【我的花篮|<a href="javascript:;" @click="bottleShow = true">我的花瓶</a>】<br/></div>
        <div class="module-content" v-if="basket.length">您收获和采摘的花均存放于您的花篮中,共有鲜花{{ basket.length }}种,{{ basketTotal }}朵<br/></div>
        <div class="text" v-if="!basket.length">暂无记录！<br/></div>
        <div class="list" v-if="basket.length">
          <div class="row" v-for="f in basket" :key="f.flower">
            <img :src="flowerImg(f.flower)" class="map-icon" alt="." /><a href="javascript:;" @click="msg = f.flower + ' 花篮'">{{ f.flower }}</a>×{{ f.count }} <a href="javascript:;" @click="openGift(f)">[送花]</a><br/>
          </div>
        </div>
        <br/>
        <a href="javascript:;" @click="openGift(null)">送花</a>.<a href="javascript:;" @click="switchTab('room')">合成</a>.<a href="javascript:;" @click="openGiftLog">送花记录</a><br/>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>

        <!-- 花瓶弹层 -->
        <div class="mask" v-if="bottleShow" @click.self="bottleShow = false">
          <div class="panel">
            <div class="name">我的花瓶</div>
            <div class="list">
              <div class="row" v-for="f in bottle" :key="'b' + f.flower">
                <img :src="flowerImg(f.flower)" class="map-icon" alt="." />{{ f.flower }}×{{ f.count }}<br/>
              </div>
              <div class="row" v-if="!bottle.length">花瓶空空的<br/></div>
            </div>
            <a href="javascript:;" @click="bottleShow = false">关闭</a>
          </div>
        </div>
      </template>

      <!-- ============ 七日签到（复刻 check/index） ============ -->
      <template v-else-if="cur === 'check'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;签到<br/></div>
        <div class="name">七日签到<br/></div>
        <div class="module-content deep">回家的礼物、一天都不能少({{ signCount }}/7)</div>
        <table class="sign-table">
          <tr>
            <td v-for="(d, i) in signDays" :key="i" align="center" :class="{ 'sign-today': d.is_today, 'sign-done': d.signed }">
              <span class="sign-day">{{ weekNames[i] }}</span><br/>
              <span class="sign-state">
                <a v-if="d.signed" href="javascript:;" class="sign-ok">√</a>
                <a v-else-if="!signToday" href="javascript:;" @click="doSign" class="sign-btn">签到</a>
                <span v-else>&nbsp;</span>
              </span>
            </td>
          </tr>
        </table>
        <div class="name">签到奖励预览<br/></div>
        <div class="list">
          <div class="row">签到第1天奖励：随机花种+1，G币+500，花园经验+200<br/></div>
          <div class="row">签到第3天奖励：随机花种+3，G币+2000，花园经验+500<br/></div>
          <div class="row">签到第5天奖励：随机花种+4，G币+10000，花园经验+1000<br/></div>
          <div class="row">签到第7天奖励：随机花种+5，G币+30000，花园经验+2000，元宝+5<br/></div>
          <div class="row deep">每日固定奖励：随机花种+1，G币+500，花园经验+200<br/></div>
        </div>
        <div class="module-content">
          <a href="javascript:;" @click="doSign" v-if="!signToday" class="sign-big">[今日签到]</a>
          <span v-else class="sign-done-txt">今日已签到</span>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;<a href="javascript:;" @click="switchTab('active')">活动</a><br/>
      </template>

      <!-- ============ 活动（复刻 active/seed_list：追寻远古花园的记忆） ============ -->
      <template v-else-if="cur === 'active'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;活动<br/></div>
        <div class="name">追寻远古花园的记忆<br/></div>
        <div class="list">
          <div class="row" v-for="a in activities" :key="a.id">
            <a href="javascript:;" @click="openAct(a)">{{ a.title }}</a><br/>
          </div>
          <div class="row" v-if="!activities.length">暂无活动<br/></div>
        </div>
        <div class="module-content" v-if="selAct">
          【{{ selAct.title }}】<br/>
          {{ selAct.desc }}<br/>
          <br/>
          完成任务需要:<br/>
          {{ actNeedsTxt }}<br/>
          <br/>
          完成任务奖励:<br/>
          {{ selAct.reward }}X{{ amount }}<br/>
          <br/>
          兑换<input v-model.number="amount" type="text" maxlength="2" size="2" value="1" />颗。<br/>
          <a href="javascript:;" @click="submitActivity">[完成任务]</a><br/>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;<a href="javascript:;" @click="openCheck">签到</a><br/>
      </template>

      <!-- ============ 帮助 ============ -->
      <template v-else-if="cur === 'help'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;帮助<br/></div>
        <div class="list">
          <div class="row" v-for="(h, i) in helpList" :key="i">
            <a href="javascript:;" @click="helpOpen = helpOpen === i ? -1 : i">Q{{ i + 1 }}.{{ h.q }}</a><br/>
            <div class="module-content" v-if="helpOpen === i">{{ h.a }}<br/></div>
          </div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 论坛（跳转魔法花园游戏论坛） ============ -->
      <template v-else-if="cur === 'forum'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;论坛<br/></div>
        <div class="module-content">正在前往魔法花园游戏论坛...<br/></div>
      </template>

      <!-- ============ 花朵详情（复刻 plant.asp） ============ -->
      <template v-else-if="cur === 'plot'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">花园</a>&gt;花朵<br/></div>
        <div class="module-content">【花朵】<br/></div>
        <div class="module-content" v-if="curPlot">
          <img :src="plotBigImg(curPlot)" alt="." /><br/>
          {{ curPlot.stage === 4 ? curPlot.name : curPlot.seed }}<br/>
          <span v-if="curPlot.stage === 1">健康:{{ curPlot.drys === 1 ? '良好' : '干旱' }}<a v-if="curPlot.drys !== 1" href="javascript:;" @click="careFromPlot('water')">[浇水]</a><br/></span>
          <span v-else-if="curPlot.stage === 2">健康:{{ curPlot.weed === 1 ? '良好' : '长草' }}<a v-if="curPlot.weed !== 1" href="javascript:;" @click="careFromPlot('weed')">[锄草]</a><br/></span>
          <span v-else-if="curPlot.stage === 3">健康:{{ curPlot.pest === 1 ? '良好' : '生虫' }}<a v-if="curPlot.pest !== 1" href="javascript:;" @click="careFromPlot('pest')">[捉虫]</a><br/></span>
          <span v-else>健康:<a v-if="curPlot.drys !== 1" href="javascript:;" @click="careFromPlot('water')">干旱[浇水]</a><a v-if="curPlot.weed !== 1" href="javascript:;" @click="careFromPlot('weed')">长草[锄草]</a><a v-if="curPlot.pest !== 1" href="javascript:;" @click="careFromPlot('pest')">生虫[捉虫]</a><span v-if="curPlot.drys === 1 && curPlot.weed === 1 && curPlot.pest === 1">良好</span><br/></span>
          <span v-if="curPlot.stage === 4">产量:{{ curPlot.amount }}/{{ curPlot.yield }}<br/>成长期:成熟 [<a href="javascript:;" @click="openHarvest(curPlot)">收获</a>]<br/></span>
          <span v-else>成长期:{{ curPlot.remain_txt }}<br/></span>
          <br/>
          <a href="javascript:;" @click="openDelPlot">铲除</a><br/>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回花园</a><br/>
      </template>

      <!-- ============ 种子/道具详情（复刻 seed.asp / dz_buy） ============ -->
      <template v-else-if="cur === 'seed'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('shop')">商店</a>&gt;{{ curSeed.dtype === 2 ? '道具' : '种子' }}<br/></div>
        <div class="module-content">【{{ curSeed.dtype === 2 ? '道具购买' : '花园商店' }}】<br/></div>
        <div class="module-content" v-if="curSeed">
          <template v-if="curSeed.dtype === 2">
            {{ curSeed.name }}<br/>
            道具价格:{{ curSeed.price }}G币<br/>
            {{ curSeed.remark }}<br/>
            <br/>
            购买<input type="text" v-model.number="buyAmount" maxlength="2" size="2" value="1" />个。<br/>
            <a href="javascript:;" @click="buy(curSeed)">确定购买</a><br/>
          </template>
          <template v-else>
            <img :src="'/static/picture/garden/s_l_' + curSeed.id + '.gif'" alt="." /><br/>
            {{ curSeed.name }}<br/>
            花种等级:{{ curSeed.level_name }}<br/>
            种子价格:{{ curSeed.price }}G币<br/>
            VIP 价格:{{ curSeed.vip_price }}G币<br/>
            预计成花:{{ curSeed.yield_avg }}朵<br/>
            预计时间:{{ growTxt(curSeed) }}<br/>
            鲜花花语:{{ curSeed.remark }}<br/>
            <br/>
            购买<input type="text" v-model.number="buyAmount" maxlength="2" size="2" value="1" />颗。<br/>
            <a href="javascript:;" @click="buy(curSeed)">确定购买</a><br/>
          </template>
        </div>
        <a href="javascript:;" @click="switchTab('shop')">返回商店</a><br/>
      </template>

      <!-- ============ 图谱详情（复刻 map.asp） ============ -->
      <template v-else-if="cur === 'mapinfo'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('map')">图谱</a>&gt;{{ curMap.name }}<br/></div>
        <div class="module-content">【花种图谱】<br/></div>
        <div class="module-content" v-if="curMap">
          <img :src="curMap.got ? ('/static/picture/garden/' + (curMap.img || ('m_l_' + curMap.id + '.gif'))) : '/static/picture/garden/m_l.gif'" alt="." /><br/>
          {{ curMap.name }}<br/>
          花种等级:{{ curMap.level_name }}<br/>
          种子单价:{{ curMap.price }}G币<br/>
          预计成花:{{ curMap.yield_avg }}朵<br/>
          成花时间:{{ growTxt(curMap) }}<br/>
          鲜花花语:{{ curMap.remark }}<br/>
          购买{{ curMap.seed_name }}种子有一定几率种出{{ curMap.name }}，<a href="javascript:;" @click="goShopSeed(curMap)">去商店购买</a><br/>
        </div>
        <a href="javascript:;" @click="switchTab('map')">返回图谱</a><br/>
      </template>

      <!-- ============ 配方详情（复刻 room.asp） ============ -->
      <template v-else-if="cur === 'roominfo'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('room')">魔法屋</a>&gt;{{ curRoom.name }}<br/></div>
        <div class="module-content" v-if="curRoom">
          {{ curRoom.name }}<br/>
          花种等级:{{ curRoom.level }}级<br/>
          合成需消耗:<br/>
          <span v-for="(m, j) in curRoom.mats" :key="j">{{ m.flower }}({{ m.need }}/{{ m.have }})<br/></span>
          <span v-if="curRoom.can" class="ok-txt"><a href="javascript:;" @click="mixFromRoom">合成</a><br/></span>
          <span v-else>所需花朵不足，您目前还不能合成{{ curRoom.name }}。<br/></span>
        </div>
        <a href="javascript:;" @click="switchTab('room')">返回魔法屋</a><br/>
      </template>

      <!-- ============ 送花记录（复刻 basket_gift.asp） ============ -->
      <template v-else-if="cur === 'giftlog'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('basket')">花篮</a>&gt;送花<br/></div>
        <div class="module-content">【送花记录】<br/></div>
        <div class="list">
          <div class="row" v-for="(r, i) in giftLogs" :key="r.id">
            {{ i + 1 }}.<a href="javascript:;" @click="visitUid(r.from_uid)">{{ r.from_nick }}</a>送{{ r.flower }}({{ r.amount }}朵)，赠言：{{ r.remark }} [{{ r.time_txt }}] [<a href="javascript:;" @click="regift(r)">再送</a>]<br/>
          </div>
          <div class="row" v-if="!giftLogs.length">暂无记录！<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('basket')">返回花篮</a><br/>
      </template>

      <!-- ============ 消息列表（复刻 message_list.asp） ============ -->
      <template v-else-if="cur === 'msgs'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('garden')">我的花园</a>&gt;消息<br/></div>
        <div class="module-content">【花园消息】<br/></div>
        <div class="list">
          <div class="row" v-for="(m, i) in msgs" :key="m.id">
            {{ i + 1 }}.({{ m.time_txt || '' }})<b>{{ m.nick }}</b> {{ m.msg }}<br/>
          </div>
          <div class="row" v-if="!msgs.length">您没有消息！<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('garden')">返回我的花园</a><br/>
      </template>
    </div>

    <!-- 播种弹层 -->
    <div class="mask" v-if="sowBox" @click.self="sowBox = false">
      <div class="panel">
        <div class="name">选择种子（花盆 {{ sowTarget ? sowTarget.plot : '?' }}）</div>
        <div class="list">
          <div class="row" v-for="b in bag" :key="b.seed_id">
            <img :src="'/static/picture/garden/' + (b.img || ('s_s_' + b.seed_id + '.gif'))" class="map-icon" alt="." />
            {{ b.seed_name }}×{{ b.count }} <a href="javascript:;" @click="sow(b)">[播种]</a><br/>
          </div>
          <div class="row" v-if="!bag.length">背包没有种子，先去商店买吧。<br/></div>
        </div>
        <a href="javascript:;" @click="sowBox = false; switchTab('shop')">去商店</a> <a href="javascript:;" @click="sowBox = false">关闭</a>
      </div>
    </div>

    <!-- 设置弹层（花园名/公告板） -->
    <div class="mask" v-if="setBox" @click.self="setBox = false">
      <div class="panel">
        <div class="name">{{ setAct === 1 ? '更改花园名' : '更改公告板' }}</div>
        <div class="module-content">
          <input v-if="setAct === 1" v-model.trim="setName" style="width:180px" placeholder="花园名" />
          <input v-else v-model.trim="setNotice" style="width:220px" placeholder="公告内容" /><br/>
          <a href="javascript:;" @click="saveSetting">保存</a> <a href="javascript:;" @click="setBox = false">取消</a>
        </div>
      </div>
    </div>

    <!-- 送花弹层 -->
    <div class="mask" v-if="giftBox" @click.self="giftBox = false">
      <div class="panel">
        <div class="name">送花（{{ giftTarget.flower || '选择花朵' }}）</div>
        <div class="module-content">
          <span v-if="!giftTarget.flower">花朵：<select v-model="giftPickFlower">
            <option v-for="f in basket" :key="f.flower" :value="f.flower">{{ f.flower }}×{{ f.count }}</option>
          </select><br/></span>
          收花好友：<input v-model.trim="giftTo" placeholder="好友昵称" style="width:120px" /><br/>
          数量：<input v-model.number="giftAmount" type="number" min="1" style="width:60px" /><br/>
          寄语：<input v-model.trim="giftRemark" style="width:160px" /><br/>
          <a href="javascript:;" @click="sendGift">赠送</a> <a href="javascript:;" @click="giftBox = false">取消</a>
        </div>
      </div>
    </div>

    <div class="okmsg" v-if="okMsg">{{ okMsg }}</div>
    <div class="errmsg" v-if="msg">{{ msg }}</div>
  </div>
</template>

<script>
import api from '../api'

const helpList = [
  { q: '如何播种花朵?', a: '首先您需要有一个可播种的花盆，种子可以去商店购买获得。等级越高，可以买到的种子就越多，有机会种出的花种也就越多。随着等级的上升，您还可以添置新的花盆。' },
  { q: '如何更好的培育花朵?', a: '播种后，花朵会依次经历花种期，花苗期，花蕾期，然后开出绚烂的花朵。成长中的花朵是娇嫩而脆弱的，她们可能会口渴或被害虫和杂草骚扰。这时您可以通过浇水，除草，除虫的操作来确保花朵的健康成长。健康的花朵将会获得更多的产量，若不及时照料花朵，您可以收获的花朵数就会大打折扣。' },
  { q: '我可以对好友做什么?', a: '您可以在好友还没有收获鲜花前将他的鲜花摘为己有；帮好友浇浇花，除除草什么的也是可以获得经验的(每天会有上限)；送花总是表达感情的好方式。您对好友的"小动作"，他都会收到相关的消息。' },
  { q: '什么是花种图谱?', a: '花之图谱里记录了花园世界所有已知的花朵，每当您成功种出一种花，花谱中对应花朵的图标就会被点亮。培育出越多花种，您就能点亮更多图标，获得更高的荣誉。' },
  { q: '怎样种出想要的颜色的花?', a: '在商店中只能购买某一类花的种子，至于种出什么颜色的花就得看您的运气啦。要么勤奋的种花，要么就多去好友那看看，说不定正好可以互补有无。' },
  { q: '什么是合成?', a: '一些珍稀的花种(通常是金色或银色的)是无法在商店购买获得的，其种子需要花匠在魔法屋中用鲜花合成获得。每朵珍稀花种的合成方法都可以在图谱或魔法屋中查看。' },
  { q: '关于经验和等级?', a: '1.播种:+2，每日上限20点；2.浇水，除草，除虫:+1，每日上限50点；3.收获花朵会获得经验，等级越高的花朵经验值越高；4.合成花种会获得20至200点经验；5.点亮花谱可获得50至300点经验；6.每级升级所需经验为:当前级别*(200点)。' },
  { q: '魔法花园的等级是怎样的?', a: '目前魔法花园共分35个级别，从"见习魔法学徒"到"五星圣魔法师"，对应不同称谓。' },
  { q: '怎样可以获取G币?', a: '在您收获花朵时，魔法花园就会奖励您一笔G币，多多种花，G币自然不成问题。您也可以在社区分财宝获取G币。' },
  { q: '什么条件下可以添加更多花盆?', a: '当您在达到2,5,8,12,16,20,25,32级别时，均能添置新的花盆。开启花盆还需要一定的G币。' },
  { q: '积累下来多余的花朵可以做什么用?', a: '您可以将他们送给好朋友们，献上您的一份祝福。也可以持续积累着，后续我们会不断推出新的功能和活动。' },
  { q: '花房里花篮和花瓶的区别?', a: '花篮是存放您在魔法花园所种植和从好友处摘取花朵的地方，而您获赠的花朵则存放在您的花瓶中，别人送您的鲜花您将不能再转送给别人。' },
  { q: '为什么每个人首页展示的花朵数量不一样?', a: '根据花园主人等级不同，花园里可展示的花朵数也会有所不同：1-7级展示4朵，8-15级展示5朵，16-23级展示6朵，24级及以上展示7朵。' }
]

export default {
  name: 'Garden',
  data () {
    return {
      cur: 'garden', plots: [], g: {}, coins: 0, bag: [], basket: [], bottle: [],
      friends: [], vplots: [], vg: {}, visitNick: '', roomList: [], shop: [], mapList: [], mapTy: 0,
      activities: [], selAct: null, actNeeds: [], amount: 1, msgs: [], recentMaps: [],
      elfList: [], elvesUnlocked: 0, elvesTotal: 0,
      rankList: [], helpList, helpOpen: -1,
      mapWd: '', mapPage: 1, shopTy: 0, shopPage: 1, roomWd: '', roomPage: 1,
      sowBox: false, sowTarget: null, setBox: false, setAct: 1, setName: '', setNotice: '', setConfig: 0,
      giftBox: false, giftTarget: {}, giftTo: '', giftAmount: 1, giftRemark: '希望你开心快乐！', giftPickFlower: '',
      bottleShow: false, curPlot: null, curSeed: null, curMap: null, curRoom: null, giftLogs: [], buyAmount: 1,
      signDays: [], signToday: false, signCount: 0, weekNames: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
      noticeTitle: '点击查看魔法花园最新公告', noticeThreadId: 0,
      okMsg: '', msg: ''
    }
  },
  computed: {
    nick () { return (this.$store.state.user || {}).nickname || '神秘园丁' },
    emptyCount () { return this.plots.filter(p => p.stage === 0).length },
    basketTotal () { return this.basket.reduce((s, f) => s + f.count, 0) },
    bagSeeds () { return this.bag.filter(b => b.dtype !== 2) },
    bagItems () { return this.bag.filter(b => b.dtype === 2) },
    filteredMaps () {
      if (!this.mapWd) return this.mapList
      return this.mapList.filter(m => m.name.indexOf(this.mapWd) !== -1)
    },
    pagedMaps () {
      const start = (this.mapPage - 1) * 10
      return this.filteredMaps.slice(start, start + 10)
    },
    mapPages () { return Math.max(1, Math.ceil(this.filteredMaps.length / 10)) },
    pagedShop () {
      const list = this.shop.filter(s => (this.shopTy === 0 && s.dtype === 0) || (this.shopTy === 1 && s.dtype === 1) || (this.shopTy === 2 && s.dtype === 2))
      const start = (this.shopPage - 1) * 10
      return list.slice(start, start + 10)
    },
    shopPages () {
      const list = this.shop.filter(s => (this.shopTy === 0 && s.dtype === 0) || (this.shopTy === 1 && s.dtype === 1) || (this.shopTy === 2 && s.dtype === 2))
      return Math.max(1, Math.ceil(list.length / 10))
    },
    filteredRoom () {
      if (!this.roomWd) return this.roomList
      return this.roomList.filter(s => s.name.indexOf(this.roomWd) !== -1)
    },
    pagedRoom () {
      const start = (this.roomPage - 1) * 10
      return this.filteredRoom.slice(start, start + 10)
    },
    roomPages () { return Math.max(1, Math.ceil(this.filteredRoom.length / 10)) },
    myRank () {
      const me = this.rankList.find(r => r.user_id === (this.$store.state.user || {}).id)
      return me ? ('第' + me.rank + '名') : '未入榜'
    },
    actNeedsTxt () {
      if (!this.actNeeds || !this.actNeeds.length) return ''
      return this.actNeeds.map(n => n.n + '朵' + n.flower).join(' 和 ')
    }
  },
  mounted () {
    this.loadAll()
  },
  methods: {
    gardenImg (file) { return '/static/picture/garden/' + file },
    flowerImg (flower) {
      const id = this.mapIDByFlower(flower)
      return '/static/picture/garden/' + (id ? ('m_s_' + id + '.gif') : 'm_s.gif')
    },
    mapIDByFlower (flower) {
      const map = { '金色向日葵': 1, '七彩向日葵': 2, '太阳神花': 3, '红玫瑰': 4, '蓝玫瑰': 5, '黑玫瑰': 6, '黄郁金香': 7, '粉郁金香': 8, '黑郁金香': 9, '月光花': 10, '星月花': 11, '幻月花': 12, '白百合': 13, '金百合': 14, '火百合': 15, '粉牡丹': 16, '绿牡丹': 17, '黑牡丹': 18, '蓝色妖姬': 19, '冰蓝妖姬': 20, '魅蓝妖姬': 21, '粉樱花': 22, '垂枝樱': 23, '夜樱': 24, '雪莲花': 25, '金雪莲': 26, '七彩雪莲': 27, '银色菊花': 28, '银野花': 29, '端阳花': 30, '银友谊花': 31, '银色烈焰焚情': 32, '金色烈焰焚情': 33 }
      return map[flower] || 0
    },
    switchTab (tab) {
      this.cur = tab; this.msg = ''; this.okMsg = ''
      if (tab === 'garden') this.load()
      if (tab === 'friend') this.loadFriends()
      if (tab === 'basket') this.loadBasket()
      if (tab === 'room') this.loadRoom()
      if (tab === 'shop') this.loadShop()
      if (tab === 'bag') this.loadBag()
      if (tab === 'rank') this.loadRank()
      if (tab === 'elves') this.loadElves()
    },
    switchShop (ty) { this.shopTy = ty; this.shopPage = 1 },
    loadAll () {
      this.load()
      this.loadBasket()
      this.loadElves()
      api.get('/garden-activities').then(r => { if (r.code === 0) this.activities = r.data })
    },
    load () {
      api.get('/games/garden/view').then(r => {
        if (r.code === 0) {
          this.g = r.data.garden || {}
          this.coins = r.data.coins || 0
          this.plots = r.data.plots || []
          this.bag = (r.data.bag || []).map(b => ({ seed_id: b.seed_id, seed_name: b.name, count: b.amount, dtype: b.dtype || 0 }))
          this.msgs = r.data.msgs || []
          this.recentMaps = r.data.recent_maps || []
          this.g.map_total = this.g.map_total || 619
        }
      })
      // 公告 = 魔法花园游戏论坛最新帖子（参考站 note 链接 BBS 帖子）
      api.get('/boards/22/threads').then(r => {
        if (r.code === 0 && r.data && r.data.list && r.data.list.length) {
          this.noticeTitle = r.data.list[0].title
          this.noticeThreadId = r.data.list[0].id
        }
      })
      this.loadSign()
    },
    loadFriends () {
      api.get('/friends').then(r => { if (r.code === 0) this.friends = r.data.friends })
    },
    loadBasket () {
      api.get('/games/garden/basket').then(r => { if (r.code === 0) this.basket = r.data })
      api.get('/games/garden/bottle').then(r => { if (r.code === 0) this.bottle = r.data })
    },
    loadRoom () {
      api.get('/games/garden/room').then(r => { if (r.code === 0) this.roomList = r.data })
    },
    loadShop () {
      api.get('/games/garden/shop').then(r => { if (r.code === 0) this.shop = r.data })
    },
    loadBag () {
      api.get('/games/garden/view').then(r => { if (r.code === 0) this.bag = (r.data.bag || []).map(b => ({ seed_id: b.seed_id, seed_name: b.name, count: b.amount, dtype: b.dtype || 0 })) })
    },
    loadRank () {
      api.get('/games/garden/rank').then(r => { if (r.code === 0) this.rankList = r.data || [] })
    },
    loadElves () {
      api.get('/games/garden/elves').then(r => {
        if (r.code === 0) { this.elfList = r.data.list || []; this.elvesUnlocked = r.data.unlocked || 0; this.elvesTotal = r.data.total || 0 }
      })
    },
    loadMap (ty) {
      if (typeof ty === 'number') this.mapTy = ty
      this.mapPage = 1
      api.get('/games/garden/map?ty=' + this.mapTy).then(r => { if (r.code === 0) { this.mapList = r.data.list; this.g.common = r.data.common; this.g.festival = r.data.festival; this.g.scarce = r.data.scarce } })
    },
    searchMap () { this.mapPage = 1 },
    searchRoom () { this.roomPage = 1 },
    openMap () { this.cur = 'map'; this.mapPage = 1; this.mapWd = ''; this.loadMap(0) },
    openElves () { this.cur = 'elves'; this.loadElves() },
    openMsgs () { this.cur = 'msgs' },
    openGiftLog () { this.cur = 'giftlog'; this.loadGiftLog() },
    loadGiftLog () {
      api.get('/games/garden/giftlog').then(r => { if (r.code === 0) this.giftLogs = r.data || [] })
    },
    regift (r) {
      // 再送：向原送花人回送同样花朵
      this.giftTarget = { flower: r.flower }
      this.giftTo = r.from_nick
      this.giftAmount = r.amount
      this.giftRemark = '回赠' + r.flower
      this.giftBox = true
    },
    openMapDetail (m) { this.curMap = m; this.cur = 'mapinfo' },
    openBagSeed (b) { this.msg = b.seed_name + (b.dtype === 2 ? ' 道具，共 ' : ' 种子，共 ') + b.count + (b.dtype === 2 ? ' 个' : ' 颗') },
    openNotice () {
      if (this.noticeThreadId) { this.$router.push('/thread/' + this.noticeThreadId); return }
      this.goForum()
    },
    goForum () { this.$router.push('/board/22') },
    openAct (a) {
      this.selAct = a
      this.amount = 1
      let needs = []
      try { needs = JSON.parse(a.needs || '[]') } catch (e) { needs = [] }
      this.actNeeds = needs
    },
    openCheck () { this.cur = 'check'; this.loadSign() },
    loadSign () {
      api.get('/games/garden/sign-status').then(r => {
        if (r.code === 0) {
          this.signDays = r.data.days || []
          this.signToday = !!r.data.today_signed
          this.signCount = r.data.signed_count || 0
        }
      })
    },
    doSign () {
      api.post('/games/garden/sign').then(r => {
        if (r.code === 0) { this.okMsg = (r.data && r.data.msg) || '签到成功'; this.loadSign(); this.load() } else this.msg = r.msg
      })
    },
    goShopSeed (m) {
      // 去商店购买对应种子（图谱详情）
      this.switchTab('shop')
      this.shopTy = 0
      const sd = this.shop.find(s => s.id === m.seed_id)
      if (sd) this.openSeedDetail(sd)
    },
    openElfDetail (e) {
      this.msg = e.name + '：' + (e.unlocked ? '已开启，唤醒需点亮图谱 ' + e.need_map + ' 个' : '未开启，点亮 ' + e.need_map + ' 个图谱后开启。' + (e.desc || ''))
    },
    openSeedDetail (s) {
      if (s.dtype === 1) { this.cur = 'room'; this.roomWd = s.name; this.msg = ''; this.okMsg = '该花种需在魔法屋合成获得'; return }
      this.curSeed = s; this.buyAmount = 1; this.cur = 'seed'
    },
    growTxt (s) {
      const mins = (s.seed || 0) + (s.ling || 0) + (s.buds || 0)
      return mins >= 60 ? (Math.round(mins / 6) / 10 + '小时') : (mins + '分钟')
    },
    openRoomDetail (s) { this.curRoom = s; this.cur = 'roominfo' },
    mixFromRoom () { this.mix(this.curRoom) },
    openPlot (p) { this.msg = p.seed + ' ' + p.stage_name + ' ' + p.remain_txt },
    openPlotPage (p) {
      this.curPlot = p
      this.cur = 'plot'
      this.loadPlotDetail(p.id)
    },
    loadPlotDetail (id) {
      // 详情数据来自 view 里最新的花圃（含 drys/weed/pest/yield/amount/remain_txt）
      const p = this.plots.find(x => x.id === id)
      if (p) this.curPlot = p
    },
    plotBigImg (p) {
      if (!p) return '/static/picture/garden/m_s.gif'
      if (p.stage === 4) {
        return '/static/picture/garden/' + (p.map_id ? ('m_l_' + p.map_id + '.gif') : 'm_l.gif')
      }
      return '/static/picture/garden/s_l_' + (p.seed_id || 0) + '.gif'
    },
    careFromPlot (kind) { this.care(this.curPlot, kind) },
    openDelPlot () {
      if (!window.confirm('确定铲除该株花朵吗？')) return
      const fd = new FormData(); fd.append('id', this.curPlot.id)
      api.post('/games/garden/delplot', fd).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '铲除花朵成功'; this.cur = 'garden'; this.load() } else this.msg = r.msg
      })
    },
    openSowFirst () {
      const empty = this.plots.find(p => p.stage === 0)
      if (empty) this.openSow(empty)
    },
    openSow (p) { this.sowTarget = p; this.sowBox = true; this.msg = ''; this.okMsg = '' },
    openSowSeed (b) {
      const empty = this.plots.find(p => p.stage === 0)
      if (!empty) { this.msg = '没有空花盆了'; return }
      api.post('/games/garden/sow', { seed_id: b.seed_id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '播种成功'; this.load() } else this.msg = r.msg
      })
    },
    sow (b) {
      api.post('/games/garden/sow', { seed_id: b.seed_id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '播种成功'; this.sowBox = false; this.load() } else this.msg = r.msg
      })
    },
    care (p, kind) {
      const names = { water: '浇水', weed: '锄草', pest: '捉虫' }
      const fd = new FormData(); fd.append('id', p.id)
      api.post('/games/garden/care/' + kind, fd).then(r => {
        if (r.code === 0) { this.okMsg = (r.data.msg || names[kind] + '成功'); this.load() } else this.msg = r.msg
      })
    },
    careVisit (p, kind) {
      const names = { water: '浇水', weed: '锄草', pest: '捉虫' }
      const fd = new FormData(); fd.append('id', p.id)
      api.post('/games/garden/care/' + kind, fd).then(r => {
        if (r.code === 0) { this.okMsg = (r.data.msg || names[kind] + '成功'); this.visitGarden({ id: this.vg.user_id }) } else this.msg = r.msg
      })
    },
    openHarvest (p) {
      const fd = new FormData(); fd.append('id', p.id)
      api.post('/games/garden/harvest', fd).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg + '，获得 ' + r.data.flower + '×' + r.data.amount + '，金币 +' + r.data.money; this.coins = r.data.coins; this.loadBasket(); this.load() } else this.msg = r.msg
      })
    },
    addLand () {
      api.post('/games/garden/addland').then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '添置花盆成功'; this.load() } else this.msg = r.msg
      })
    },
    openSetting (act) {
      this.setAct = act
      this.setName = this.g.name
      this.setNotice = this.g.notice
      this.setConfig = this.g.config
      this.setBox = true
    },
    saveSetting () {
      const body = {}
      if (this.setAct === 1) body.name = this.setName
      else body.notice = this.setNotice
      api.post('/games/garden/setting', body).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '修改成功'; this.setBox = false; this.load() } else this.msg = r.msg
      })
    },
    visitGarden (f) {
      api.get('/games/garden/visit?uid=' + f.id).then(r => {
        if (r.code === 0) {
          if (!r.data.garden) { this.msg = r.data.msg || '对方还没开通花园'; return }
          this.vg = r.data.garden; this.vplots = r.data.plots || []; this.visitNick = r.data.nickname
          this.vg.user_id = f.id
          this.cur = 'visit'
        }
      })
    },
    visitUid (uid) { this.visitGarden({ id: uid }) },
    pick (p) {
      const fd = new FormData(); fd.append('id', p.id)
      api.post('/games/garden/pick', fd).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg; this.loadBasket(); this.visitGarden({ id: this.vg.user_id }) } else this.msg = r.msg
      })
    },
    openGift (f) {
      this.giftBox = true
      this.giftTarget = f || {}
      this.giftPickFlower = f ? f.flower : ''
      this.giftTo = ''
      this.giftAmount = 1
      this.giftRemark = '希望你开心快乐！'
    },
    openGiftToVisit () {
      // 参观页送花：目标 = 被参观园主，花从花篮选
      this.giftBox = true
      this.giftTarget = {}
      this.giftPickFlower = ''
      this.giftTo = this.visitNick
      this.giftAmount = 1
      this.giftRemark = '希望你开心快乐！'
    },
    sendGift () {
      const flower = this.giftTarget.flower || this.giftPickFlower
      if (!flower) { this.msg = '请选择要送的花朵'; return }
      if (!this.giftTo) { this.msg = '请填写收花好友昵称'; return }
      const target = this.friends.find(x => x.nickname === this.giftTo) || this.friends.find(x => String(x.id) === this.giftTo)
      if (!target) { this.msg = '只能给好友送花'; return }
      api.post('/games/garden/gift', { to_uid: target.id, flower: flower, amount: this.giftAmount || 1, remark: this.giftRemark }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '赠送成功'; this.giftBox = false; this.loadBasket() } else this.msg = r.msg
      })
    },
    mix (s) {
      api.post('/games/garden/mix', { seed_id: s.seed_id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg; this.loadRoom(); this.loadBasket() } else this.msg = r.msg
      })
    },
    buy (s) {
      const n = this.buyAmount && this.buyAmount > 0 ? Math.min(this.buyAmount, 99) : 1
      api.post('/games/garden/buy', { id: s.id, amount: n }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || ('成功购买' + s.name + '种子' + n + '颗'); this.coins = r.data.coins; this.loadBag() } else this.msg = r.msg
      })
    },
    submitActivity () {
      this.msg = ''; this.okMsg = ''
      if (!this.selAct) return
      const amt = this.amount && this.amount > 0 && this.amount < 10 ? this.amount : 1
      api.post('/games/garden/activity-submit', { id: this.selAct.id, amount: amt }).then(r => {
        if (r.code === 0) { this.okMsg = '任务完成！获得「' + r.data.reward + '」×' + r.data.amount; this.loadAll() } else this.msg = r.msg
      })
    }
  }
}
</script>

<style scoped>
/* ===== 复刻 3gqq.ink 参考站 style.css ===== */
.note { background: #FFF9B7; border-bottom: 1px solid #9FC6EC; padding: 3px; font-size: 13px; line-height: 1.6; }
.note a { color: #2e9cd3; text-decoration: none; }
.noteico { vertical-align: middle; }
.bar { height: 25px; padding: 0 5px; background: #71afe3; line-height: 25px; color: #fff; font-size: 14px; }
.bar a { color: #fff; text-decoration: none; margin-right: 6px; }
.bar img { vertical-align: middle; }
.bar.sub { background: #9FC6EC; }
.name { padding-left: 3px; line-height: 20px; border-bottom: 2px solid #9FC6EC; color: #000; font-weight: bold; font-size: 14px; }
.name a { font-weight: normal; color: #2e9cd3; text-decoration: none; margin-left: 4px; }
.module-content { font-size: 13px; line-height: 1.8; color: #333; padding: 2px 3px; }
.module-content a { color: #2e9cd3; text-decoration: none; }
.module-title { padding: 2px 3px; font-size: 13px; line-height: 1.8; }
.module-title a { color: #2e9cd3; text-decoration: none; }
.deep { background: #E3EEF8; border: 1px solid #9FC6EC; border-left: none; border-right: none; }
.list { line-height: 1.6; font-size: 13px; }
.row { padding: 3px; border-bottom: 1px solid #E3E6EB; }
.row a { color: #2e9cd3; text-decoration: none; }
.text { line-height: 1.6; padding: 3px 5px; word-wrap: break-word; font-size: 13px; }
.g-main { padding: 2px 3px; }
.userline img { vertical-align: middle; }
.bicon { vertical-align: middle; }
.map-icon { vertical-align: middle; margin-right: 2px; }
.recent-maps { padding: 4px 3px; line-height: 0; text-align: left; background: #f4f9fd; border-bottom: 1px dashed #cfe0f0; }
.recent-maps img { width: 30px; height: 30px; margin: 2px 3px; border: 1px solid #d7e6f3; border-radius: 3px; vertical-align: middle; }
.sign-table { width: 100%; margin: 4px 0; border-collapse: collapse; font-size: 13px; }
.sign-table td { border: 1px solid #cfe0f0; padding: 8px 2px; background: #f7fbfe; }
.sign-table td.sign-today { background: #fff8dc; border: 2px solid #ffb300; }
.sign-table td.sign-done { background: #e8f5e9; }
.sign-table .sign-day { color: #555; font-weight: bold; }
.sign-table a { text-decoration: none; }
.sign-table .sign-btn { color: #2e9cd3; font-weight: bold; }
.sign-table .sign-ok { color: #43a047; font-size: 16px; font-weight: bold; }
.sign-table td.sign-today .sign-day { color: #e65100; }
.sign-big { display: inline-block; margin: 6px 0; padding: 4px 12px; background: #ffb300; color: #fff; border-radius: 4px; text-decoration: none; font-weight: bold; }
.sign-done-txt { color: #43a047; font-weight: bold; }
.got-tag { color: #43a047; font-size: 11px; }
.ok-txt a { color: #43a047; font-weight: bold; }
.elf-prev { vertical-align: middle; width: 32px; height: 32px; margin-right: 4px; }
.mask { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,.4); z-index: 99; display: flex; align-items: center; justify-content: center; }
.panel { background: #fff; border-radius: 8px; width: 320px; max-height: 70vh; overflow: auto; padding: 12px; }
.panel input, .panel select { border: 1px solid #ccc; border-radius: 3px; padding: 2px 4px; font-size: 13px; }
.panel a { color: #2e9cd3; text-decoration: none; margin: 0 4px; }
.okmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #43a047; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
.errmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #d32f2f; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
</style>
