<template>
  <div>
    <!-- 顶部导航（对齐诺哈 wap/game/car：停车场 车库 车市 好友 排行 规则） -->
    <div class="bar garden-nav">
      <a :class="{ cur: cur === 'park' }" href="javascript:;" @click="switchTab('park')">停车场</a> <a :class="{ cur: cur === 'garage' }" href="javascript:;" @click="switchTab('garage')">车库</a> <a :class="{ cur: cur === 'shop' }" href="javascript:;" @click="switchTab('shop')">车市</a> <a :class="{ cur: cur === 'friends' }" href="javascript:;" @click="switchTab('friends')">好友</a> <a :class="{ cur: cur === 'top' }" href="javascript:;" @click="switchTab('top')">排行</a> <a :class="{ cur: cur === 'help' }" href="javascript:;" @click="switchTab('help')">规则</a>
    </div>

    <div class="g-main">
      <!-- ============ 我的停车场（复刻 index.asp） ============ -->
      <template v-if="cur === 'park'">
        <div class="name userline">{{ me.nick }} <img class="bicon" src="/static/image/noble_2_1.gif" alt="." />({{ me.level || 0 }}级)</div>
        <div class="module-content deep">抢车位 车辆总值决定身价，贴条才是硬道理</div>
        <div class="module-content">
          有<a href="javascript:;" @click="switchTab('friends')">{{ me.count }}</a>人加入了游戏<br/>
          昵称:{{ me.nick }}的停车场<br/>
          等级:{{ me.level || 0 }} 爱心:{{ me.love || 0 }} 现金:{{ me.coins || 0 }}G 汽车数:{{ me.cars || 0 }}<br/>
          查看:<a href="javascript:;" @click="switchTab('garage')">车库</a>.<a href="javascript:;" @click="switchTab('shop')">市场</a>.<a href="javascript:;" @click="switchTab('top')">排行</a>.<a href="javascript:;" @click="switchTab('msgs')">消息({{ me.unread || 0 }})</a>.<a href="javascript:;" @click="switchTab('friends')">去好友停车场停车</a><br/>
        </div>
        <br/>

        <div class="name">我的车位({{ spots.length }})<a href="javascript:;" @click="load">刷新</a></div>
        <div class="list">
          <div class="row" v-for="s in spots" :key="s.id">
            【{{ s.sort }}号车位】<br/>
            <template v-if="s.empty">
              <span class="car-empty">空车位</span><br/>
              <span v-if="s.over_msg" class="dim">{{ s.over_msg }}<br/></span>
            </template>
            <template v-else>
              <span class="car-tag" :class="'c' + (s.dtype || 1)">{{ carIcon(s.dtype) }} {{ s.car_name }}</span><br/>
              汽车总值:{{ s.price }}G<br/>
              车主:<a href="javascript:;" @click="visit(s.owner)">{{ s.owner_nick }}</a><br/>
              停车时间:{{ s.minutes }}分钟/{{ s.hours }}小时<br/>
              总的收入:{{ s.total }} 税收缴纳:{{ s.tax }}<br/>
              预计收入:{{ s.net }}G 预计贡献:{{ s.contri }}点<br/>
              <template v-if="s.can_seal">&gt;&gt;<a href="javascript:;" @click="openSeal(s)">贴封条</a><br/></template>
            </template>
          </div>
          <div class="row" v-if="!spots.length">还没有车位<br/></div>
        </div>
        <br/>

        <div class="name">最新加入</div>
        <div class="list">
          <div class="row" v-for="(r, i) in recent" :key="r.uid">
            {{ i + 1 }}.<a href="javascript:;" @click="visit(r.uid)">{{ r.nick }}</a>({{ r.time_txt }}前)<br/>
          </div>
          <div class="row" v-if="!recent.length">还没有人加入游戏<br/></div>
        </div>
        <br/>

        <div class="module-title"><a href="javascript:;" @click="switchTab('garage')">车库</a>.<a href="javascript:;" @click="switchTab('shop')">车市</a>.<a href="javascript:;" @click="switchTab('top')">排行</a>.<a href="javascript:;" @click="switchTab('help')">规则</a>.<a href="javascript:;" @click="goForum">论坛</a><br/></div>
      </template>

      <!-- ============ 他人停车场（复刻 owner.asp） ============ -->
      <template v-else-if="cur === 'visit'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab(vOwner.is_self ? 'park' : 'friends')">{{ vOwner.is_self ? '停车场' : '好友' }}</a>&gt;{{ vOwner.nick }}的停车场<br/></div>
        <div class="module-content deep">{{ vOwner.nick }}的停车场</div>
        <div class="module-content">
          场主:{{ vOwner.nick }}({{ vOwner.level || 0 }}级)<br/>
          爱心:{{ vOwner.love || 0 }} 汽车数:{{ vOwner.cars || 0 }} 贡献:{{ vOwner.contri || 0 }}<br/>
          <a href="javascript:;" @click="switchTab('shop')">赠TA车</a>|<a href="javascript:;" @click="loadVisitGarage">TA的车库</a><br/>
        </div>
        <br/>
        <div class="name">TA的车位({{ vSpots.length }})<a href="javascript:;" @click="visit(vOwner.uid)">刷新</a></div>
        <div class="list">
          <div class="row" v-for="s in vSpots" :key="s.id">
            【{{ s.sort }}号车位】<br/>
            <template v-if="s.empty">
              <span class="car-empty">空车位</span><br/>
              <span v-if="s.over_msg" class="dim">{{ s.over_msg }}<br/></span>
              <template v-if="!vOwner.is_self">&gt;&gt;<a href="javascript:;" @click="openStop(s)">停车</a><br/></template>
            </template>
            <template v-else>
              <span class="car-tag" :class="'c' + (s.dtype || 1)">{{ carIcon(s.dtype) }} {{ s.car_name }}</span><br/>
              汽车总值:{{ s.price }}G 车主:<a href="javascript:;" @click="visit(s.owner)">{{ s.owner_nick }}</a><br/>
              停车时间:{{ s.minutes }}分钟/{{ s.hours }}小时<br/>
              <template v-if="s.can_favor">&gt;&gt;<a href="javascript:;" @click="favor(s)">收车</a><br/></template>
              <template v-if="s.can_seal">&gt;&gt;<a href="javascript:;" @click="openSeal(s)">贴封条</a><br/></template>
            </template>
          </div>
          <div class="row" v-if="!vSpots.length">TA还没有车位<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab(vOwner.is_self ? 'park' : 'friends')">返回</a><br/>
      </template>

      <!-- ============ 他人车库（复刻 garage.asp） ============ -->
      <template v-else-if="cur === 'vgarage'">
        <div class="bar sub"><a href="javascript:;" @click="visit(vOwner.uid)">TA的停车场</a>&gt;{{ vOwner.nick }}的车库<br/></div>
        <div class="name">{{ vOwner.nick }}的车库</div>
        <div class="list">
          <div class="row" v-for="g in vGarage" :key="g.id">
            <span class="car-tag" :class="'c' + (g.dtype || 1)">{{ carIcon(g.dtype) }} {{ g.car_name }}</span><br/>
            汽车总值:{{ g.price }}G<br/>
            <template v-if="g.moving">车位:流动中...<br/></template>
            <template v-else>
              车位:<a href="javascript:;" @click="visit(g.spot_uid)">{{ g.spot_nick }}</a><br/>
              停车时间:{{ g.minutes }}分钟/{{ g.hours }}小时 总的收入:{{ g.total }} 预计收入:{{ g.net }}G<br/>
            </template>
          </div>
          <div class="row" v-if="!vGarage.length">TA还没有汽车,去<a href="javascript:;" @click="switchTab('shop')">赠送一辆</a>给TA吧!<br/></div>
        </div>
        <a href="javascript:;" @click="visit(vOwner.uid)">返回</a><br/>
      </template>

      <!-- ============ 好友停车场列表（复刻 friend.asp） ============ -->
      <template v-else-if="cur === 'friends'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;好友列表<br/></div>
        <div class="module-content">可以把车停进他们的空车位赚钱，也可能被主人贴条没收哦！<br/></div>
        <div class="list">
          <div class="row" v-for="(f, i) in friends" :key="f.uid">
            {{ offset + i + 1 }}.<a href="javascript:;" @click="visit(f.uid)">{{ f.nick }}</a>({{ f.level }}级 车{{ f.cars }}辆 空位{{ f.empty }})<br/>
          </div>
          <div class="row" v-if="!friends.length">暂无人开通游戏！<br/></div>
        </div>
        <div class="module-title">
          <a v-if="fPage > 1" href="javascript:;" @click="fPage--; loadFriends()">上页</a>
          <a v-if="fPage * fSize < fTotal" href="javascript:;" @click="fPage++; loadFriends()">下页</a>
          (第{{ fPage }}页/共{{ Math.ceil(fTotal / fSize) || 1 }}页)
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>

      <!-- ============ 我的车库（复刻 my_garage.asp） ============ -->
      <template v-else-if="cur === 'garage'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;我的车库<br/></div>
        <div class="name">我的车库(G币 {{ me.coins || 0 }})<a href="javascript:;" @click="loadGarage">刷新</a></div>
        <div class="list">
          <div class="row" v-for="g in garage" :key="g.id">
            <span class="car-tag" :class="'c' + (g.dtype || 1)">{{ carIcon(g.dtype) }} {{ g.car_name }}</span><br/>
            汽车总值:{{ g.price }}G<br/>
            <template v-if="g.moving">车位:流动中，小心被警察罚款哦！赶紧<a href="javascript:;" @click="switchTab('friends')">找个车位</a>吧！<br/></template>
            <template v-else>
              车位:<a href="javascript:;" @click="visit(g.spot_uid)">{{ g.spot_nick }}</a><br/>
              停车时间:{{ g.minutes }}分钟/{{ g.hours }}小时<br/>
              <span v-if="g.over" class="dim">{{ g.over_msg }}<br/></span>
              总的收入:{{ g.total }} 税收缴纳:{{ g.tax }} 预计收入:{{ g.net }}G 预计贡献:{{ g.contri }}点<br/>
              <template v-if="g.can_favor">&gt;&gt;<a href="javascript:;" @click="favor(g)">收车</a><br/></template>
            </template>
          </div>
          <div class="row" v-if="!garage.length">您还没有汽车，去<a href="javascript:;" @click="switchTab('shop')">车市</a>买一辆吧！<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>

      <!-- ============ 车市（复刻 shop.asp） ============ -->
      <template v-else-if="cur === 'shop'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;车市<br/></div>
        <div class="module-title">
          <a v-for="(n, d) in dtypeNames" :key="d" :class="{ cur: sDtype === +d }" href="javascript:;" @click="sPage = 1; sDtype = +d; loadShop()">{{ n }}</a><br/>
        </div>
        <div class="list">
          <div class="row" v-for="car in shopList" :key="car.id">
            <span class="car-tag big" :class="'c' + car.dtype">{{ carIcon(car.dtype) }} {{ car.name }}</span><br/>
            车名:{{ car.name }} 价格:{{ car.price }}G<br/>
            盈利:{{ car.money }}(G币/小时) 12小时净赚:{{ car.money * 12 * 4 / 5 }}G<br/>
            [<a href="javascript:;" @click="buy(car)">购买</a>].[<a href="javascript:;" @click="openSend(car)">赠送</a>]<br/>
          </div>
          <div class="row" v-if="!shopList.length">暂无汽车！<br/></div>
        </div>
        <div class="module-title">
          <a v-if="sPage > 1" href="javascript:;" @click="sPage--; loadShop()">上页</a>
          <a v-if="sPage * sSize < sTotal" href="javascript:;" @click="sPage++; loadShop()">下页</a>
          (第{{ sPage }}页/共{{ Math.ceil(sTotal / sSize) || 1 }}页/共{{ sTotal }}种)
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>

      <!-- ============ 排行（复刻 car_top.asp） ============ -->
      <template v-else-if="cur === 'top'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;排行<br/></div>
        <div class="module-title">
          <a :class="{ cur: tAct === 1 }" href="javascript:;" @click="tAct = 1; loadTop()">爱心</a>.<a :class="{ cur: tAct === 2 }" href="javascript:;" @click="tAct = 2; loadTop()">贡献</a>.<a :class="{ cur: tAct === 3 }" href="javascript:;" @click="tAct = 3; loadTop()">经验</a><br/>
        </div>
        <div class="list">
          <div class="row" v-for="r in topList" :key="r.uid">
            {{ r.rank }}.<a href="javascript:;" @click="visit(r.uid)">{{ r.nick }}</a>({{ actName }}:{{ actVal(r) }})<br/>
          </div>
          <div class="row" v-if="!topList.length">暂无记录！<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>

      <!-- ============ 消息（复刻 wap_car_message） ============ -->
      <template v-else-if="cur === 'msgs'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;消息<br/></div>
        <div class="list">
          <div class="row" v-for="(m, i) in msgs" :key="m.id">
            {{ i + 1 }}.({{ m.time_txt }})<b>{{ m.nick }}</b><br/>{{ m.msg }}<br/>
          </div>
          <div class="row" v-if="!msgs.length">您没有消息.<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>

      <!-- ============ 规则（复刻 help.asp） ============ -->
      <template v-else-if="cur === 'help'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('park')">停车场</a>&gt;规则<br/></div>
        <div class="name">游戏简介</div>
        <div class="module-content">
          抢车位是一款娱乐的小游戏。在游戏中您可以通过停车、贴条赚取收入。获得收入后您就可以购买自己的dream car啦！奔驰、宝马、甲壳虫、SUV……任您选哦！<br/>
        </div>
        <div class="name">新进入游戏</div>
        <div class="module-content">
          马上获得系统随机赠送的3个车位。同时系统还赠送您1000G的生活补助费。<br/>
        </div>
        <div class="name">赚钱</div>
        <div class="module-content">
          通过把自己的车子停到好友家的车位，每小时根据您车的盈利能力来赚钱。当您把车从车位收车后，此笔收入才加入您的现金总额里；如果您发现您的车位被别人占用了，您可以给那辆车"贴条"，并收取罚金，每次贴条可获得该车收入。<br/>
        </div>
        <div class="name">停车</div>
        <div class="module-content">
          您的车只能停在好友的车位里。您自家的私家车位是用来赚钱的！您的车可以停在任意一个好友空的私家车位上并赚取收入，但您最好赶在主人来贴条之前将车挪走；停车收入要缴纳20%的税收，停车时间超过12小时，车辆所得将全部入国库哦！<br/>
        </div>
        <div class="name">贴条</div>
        <div class="module-content">
          车位主人可以看看他的私家车位是否被占用。车位主人可以对违章在其私家车位的人"贴条"并按比例没收停车收入（0%~100%）。<br/>
        </div>
        <div class="name">等级</div>
        <div class="module-content">
          等级 = 经验/100，每收车或贴条 1 小时获得 1 点经验，贡献 = 总收入的 10%。每人最多拥有 10 辆车。<br/>
        </div>
        <a href="javascript:;" @click="switchTab('park')">返回停车场</a><br/>
      </template>
    </div>

    <!-- 停车选车弹层 -->
    <div class="mask" v-if="stopBox">
      <div class="panel">
        <div class="panel-title">选择要停放的汽车</div>
        <div class="panel-row" v-for="g in movingCars" :key="g.id">
          <span class="car-tag" :class="'c' + (g.dtype || 1)">{{ carIcon(g.dtype) }} {{ g.car_name }}</span>
          [<a href="javascript:;" @click="doStop(g)">停这里</a>]
        </div>
        <div class="panel-row" v-if="!movingCars.length">您没有流动中的汽车，先去<a href="javascript:;" @click="stopBox = false; switchTab('shop')">车市</a>买一辆吧。</div>
        <div class="panel-foot"><a href="javascript:;" @click="stopBox = false">取消</a></div>
      </div>
    </div>

    <!-- 贴封条弹层 -->
    <div class="mask" v-if="sealBox">
      <div class="panel">
        <div class="panel-title">贴封条 - 没收多少收入?</div>
        <div class="panel-row">
          车辆:{{ sealTarget.car_name }} 预计收入:{{ sealTarget.net }}G<br/>
          停车时间:{{ sealTarget.minutes }}分钟/{{ sealTarget.hours }}小时<br/>
        </div>
        <div class="panel-row">
          没收比例:
          <select v-model.number="sealRatio">
            <option v-for="n in 11" :key="n - 1" :value="n - 1">{{ (n - 1) * 10 }}%</option>
          </select>
        </div>
        <div class="panel-row dim">选择0%将全额返还车主收入并给其经验；100%则全部没收。贴条经验按比例分配。</div>
        <div class="panel-foot"><a href="javascript:;" @click="doSeal">确定没收</a>.<a href="javascript:;" @click="sealBox = false">取消</a></div>
      </div>
    </div>

    <!-- 赠送弹层 -->
    <div class="mask" v-if="sendBox">
      <div class="panel">
        <div class="panel-title">赠送汽车 - {{ sendCar.name }}</div>
        <div class="panel-row">
          价格:{{ sendCar.price }}G 您当前有:{{ me.coins }}G<br/>
          好友家园号:<input v-model.trim="sendUid" type="text" maxlength="10" />
        </div>
        <div class="panel-row dim">对方须已开通抢车位且车辆未满10辆。</div>
        <div class="panel-foot"><a href="javascript:;" @click="doSend">确定赠送</a>.<a href="javascript:;" @click="sendBox = false">取消</a></div>
      </div>
    </div>

    <div class="okmsg" v-if="okMsg">{{ okMsg }}</div>
    <div class="errmsg" v-if="msg">{{ msg }}</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Park',
  data () {
    return {
      cur: 'park', me: {}, spots: [], recent: [], msgs: [],
      vOwner: {}, vSpots: [], vGarage: [],
      friends: [], fPage: 1, fSize: 10, fTotal: 0, offset: 0,
      garage: [],
      sDtype: 1, sPage: 1, sSize: 6, sTotal: 0, shopList: [],
      dtypeNames: { 1: '普通车', 2: '高级车', 3: '酷族车', 4: '贵族车', 5: '试驾车' },
      tAct: 1, topList: [],
      stopBox: false, stopTarget: null,
      sealBox: false, sealTarget: {}, sealRatio: 5,
      sendBox: false, sendCar: {}, sendUid: '',
      forumId: 0,
      okMsg: '', msg: ''
    }
  },
  computed: {
    movingCars () { return this.garage.filter(g => g.moving) },
    actName () { return { 1: '爱心', 2: '贡献', 3: '经验' }[this.tAct] },
    actVal () { return r => (this.tAct === 1 ? r.love : this.tAct === 2 ? r.contri : r.point) }
  },
  mounted () {
    this.loadAll()
  },
  methods: {
    carIcon (dtype) {
      return { 1: '🚗', 2: '🚙', 3: '🚕', 4: '🏎️', 5: '🚖' }[dtype] || '🚗'
    },
    switchTab (tab) {
      this.cur = tab; this.msg = ''; this.okMsg = ''
      if (tab === 'park') this.load()
      if (tab === 'visit' || tab === 'vgarage') this.visit(this.vOwner.uid || 0)
      if (tab === 'friends') this.loadFriends()
      if (tab === 'garage') this.loadGarage()
      if (tab === 'shop') this.loadShop()
      if (tab === 'top') this.loadTop()
      if (tab === 'msgs') this.load()
    },
    loadAll () {
      this.load()
      this.loadGarage()
    },
    load () {
      api.get('/games/park/view').then(r => {
        if (r.code === 0) {
          this.me = r.data
          this.spots = r.data.spots || []
          this.recent = r.data.recent || []
          this.msgs = r.data.msgs || []
          this.me.unread = r.data.unread || 0
        } else this.msg = r.msg
      })
    },
    loadFriends () {
      api.get('/games/park/friends?page=' + this.fPage + '&size=' + this.fSize).then(r => {
        if (r.code === 0) {
          this.friends = (r.data.list || [])
          this.fTotal = r.data.total || 0
          this.fSize = r.data.size || 10
          this.offset = ((r.data.page || 1) - 1) * this.fSize
        }
      })
    },
    loadGarage () {
      api.get('/games/park/garage').then(r => { if (r.code === 0) this.garage = r.data.list || [] })
    },
    loadVisitGarage () {
      api.get('/games/park/garage?uid=' + this.vOwner.uid).then(r => {
        if (r.code === 0) { this.vGarage = r.data.list || []; this.cur = 'vgarage' }
      })
    },
    loadShop () {
      api.get('/games/park/shop?dtype=' + this.sDtype + '&page=' + this.sPage + '&size=' + this.sSize).then(r => {
        if (r.code === 0) {
          this.shopList = r.data.list || []
          this.sTotal = r.data.total || 0
          this.sSize = r.data.size || 6
        }
      })
    },
    loadTop () {
      api.get('/games/park/top?act=' + this.tAct).then(r => { if (r.code === 0) this.topList = r.data.list || [] })
    },
    visit (uid) {
      if (!uid) return
      api.get('/games/park/owner?uid=' + uid).then(r => {
        if (r.code === 0) {
          this.vOwner = r.data
          this.vSpots = r.data.spots || []
          this.cur = 'visit'
          this.msg = ''; this.okMsg = ''
        } else this.msg = r.msg
      })
    },
    openStop (spot) {
      this.stopTarget = spot
      this.stopBox = true
      if (!this.garage.length) this.loadGarage()
    },
    doStop (gar) {
      api.post('/games/park/stop', { oid: this.vOwner.uid, stop_id: this.stopTarget.id, gar_id: gar.id }).then(r => {
        this.stopBox = false
        if (r.code === 0) { this.okMsg = r.data.msg || '停车成功'; this.visit(this.vOwner.uid); this.loadGarage() } else this.msg = r.msg
      })
    },
    favor (s) {
      api.post('/games/park/favor', { stop_id: s.id }).then(r => {
        if (r.code === 0) {
          this.okMsg = r.data.msg || '收车成功'
          if (this.cur === 'visit') this.visit(this.vOwner.uid)
          if (this.cur === 'garage') this.loadGarage()
          this.load()
        } else this.msg = r.msg
      })
    },
    openSeal (s) {
      this.sealTarget = s
      this.sealRatio = 5
      this.sealBox = true
    },
    doSeal () {
      api.post('/games/park/seal', { stop_id: this.sealTarget.id, ratio: this.sealRatio }).then(r => {
        this.sealBox = false
        if (r.code === 0) {
          this.okMsg = r.data.txt || r.data.msg || '贴车成功'
          if (this.cur === 'visit') this.visit(this.vOwner.uid)
          else this.load()
        } else this.msg = r.msg
      })
    },
    buy (car) {
      api.post('/games/park/buy', { id: car.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '购买成功'; this.load(); this.loadGarage() } else this.msg = r.msg
      })
    },
    openSend (car) {
      this.sendCar = car
      this.sendUid = ''
      this.sendBox = true
    },
    doSend () {
      if (!this.sendUid) { this.msg = '请输入好友家园号'; return }
      api.post('/games/park/send', { id: this.sendCar.id, uid: +this.sendUid }).then(r => {
        this.sendBox = false
        if (r.code === 0) { this.okMsg = r.data.msg || '赠送成功'; this.load() } else this.msg = r.msg
      })
    },
    goForum () {
      if (this.forumId) { this.$router.push('/board/' + this.forumId); return }
      api.get('/boards').then(r => {
        if (r.code !== 0) return
        let id = 0
        for (const ch of (r.data || [])) {
          for (const b of (ch.children || [])) {
            if (b.name === '狂抢车位') id = b.id
          }
        }
        if (id) { this.forumId = id; this.$router.push('/board/' + id) }
      })
    }
  }
}
</script>

<style scoped>
/* ===== 复刻 3gqq.ink 参考站 style.css（与 Farm.vue 一致） ===== */
.bar { height: 25px; padding: 0 5px; background: #71afe3; line-height: 25px; color: #fff; font-size: 14px; }
.bar a { color: #fff; text-decoration: none; margin-right: 6px; }
.bar a.cur { color: #FFF9B7; font-weight: bold; }
.bar img { vertical-align: middle; }
.bar.sub { background: #9FC6EC; }
.name { padding-left: 3px; line-height: 20px; border-bottom: 2px solid #9FC6EC; color: #000; font-weight: bold; font-size: 14px; }
.name a { font-weight: normal; color: #2e9cd3; text-decoration: none; margin-left: 4px; }
.module-content { font-size: 13px; line-height: 1.8; color: #333; padding: 2px 3px; }
.module-content a { color: #2e9cd3; text-decoration: none; }
.module-title { padding: 2px 3px; font-size: 13px; line-height: 1.8; }
.module-title a { color: #2e9cd3; text-decoration: none; margin-right: 4px; }
.module-title a.cur { color: #e65100; font-weight: bold; }
.deep { background: #E3EEF8; border: 1px solid #9FC6EC; border-left: none; border-right: none; }
.list { line-height: 1.6; font-size: 13px; }
.row { padding: 3px; border-bottom: 1px solid #E3E6EB; }
.row a { color: #2e9cd3; text-decoration: none; }
.g-main { padding: 2px 3px; }
.userline img { vertical-align: middle; }
.bicon { vertical-align: middle; }
.dim { color: #999; font-size: 12px; }
.car-empty { color: #999; }
.car-tag { display: inline-block; padding: 1px 8px; border-radius: 10px; font-size: 12px; color: #fff; background: #8fb264; }
.car-tag.c2 { background: #5a9bd5; }
.car-tag.c3 { background: #c27ba0; }
.car-tag.c4 { background: #b8860b; }
.car-tag.c5 { background: #7f8c8d; }
.car-tag.big { font-size: 14px; padding: 3px 12px; }
.mask { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,.4); z-index: 99; display: flex; align-items: center; justify-content: center; }
.panel { background: #fff; border-radius: 8px; width: 320px; max-height: 70vh; overflow: auto; padding: 12px; }
.panel-title { font-weight: bold; font-size: 14px; margin-bottom: 8px; }
.panel-row { font-size: 13px; line-height: 1.8; }
.panel-row a { color: #2e9cd3; text-decoration: none; }
.panel-row select, .panel-row input { border: 1px solid #ccc; border-radius: 3px; padding: 2px 4px; font-size: 13px; width: 100px; }
.panel-foot { margin-top: 10px; font-size: 13px; }
.panel-foot a { color: #2e9cd3; text-decoration: none; margin: 0 4px; }
.okmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #43a047; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
.errmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #d32f2f; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
</style>
