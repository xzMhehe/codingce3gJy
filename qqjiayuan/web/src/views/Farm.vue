<template>
  <div>
    <!-- 顶部导航（对齐诺哈 wap_farm：农场 邻居 背包 商店 仓库 排行） -->
    <div class="bar garden-nav">
      <a :class="{ cur: cur === 'farm' }" href="javascript:;" @click="switchTab('farm')">农场</a> <a :class="{ cur: cur === 'neighbor' }" href="javascript:;" @click="switchTab('neighbor')">邻居</a> <a :class="{ cur: cur === 'bag' }" href="javascript:;" @click="switchTab('bag')">背包</a> <a :class="{ cur: cur === 'shop' }" href="javascript:;" @click="switchTab('shop')">商店</a> <a :class="{ cur: cur === 'warehouse' }" href="javascript:;" @click="switchTab('warehouse')">仓库</a> <a :class="{ cur: cur === 'rank' }" href="javascript:;" @click="switchTab('rank')">排行</a>
    </div>

    <div class="g-main">
      <!-- ============ 我的农场（复刻 my_farm.asp / main.asp） ============ -->
      <template v-if="cur === 'farm'">
        <div class="name userline">{{ nick }} <img class="bicon" src="/static/image/noble_2_1.gif" alt="." />({{ farm.level || 1 }}级)</div>
        <div class="module-content deep">{{ farm.level || 1 }}级 农场(经验 {{ farm.point || 0 }}/{{ farm.need || 20 }})</div>
        <div class="module-content">
          农场名:{{ farm.name }} <a href="javascript:;" @click="openSetting(1)">更改</a><br/>
          今日剩余施肥:<a href="javascript:;" @click="openHelp(3)">{{ farm.mucks }}</a>次 摘取权限:<a href="javascript:;" @click="openSetting(2)">{{ cstealTxt }}</a><br/>
          我的奴隶:<a href="javascript:;" @click="switchTab('slaves')">({{ slaves.length }})</a> <a href="javascript:;" @click="load">刷新农场</a><br/>
        </div>
        <br/>

        <div class="name">菜地({{ lands.length }})<a href="javascript:;" @click="load">刷新</a></div>
        <div class="list">
          <div class="row" v-for="l in lands" :key="l.id">
            <template v-if="l.type === 0 && l.plow === 0">
              空地 [<a href="javascript:;" @click="plow(l)">翻地</a>]<br/>
            </template>
            <template v-else-if="l.type === 0 && l.plow === 1">
              已翻的地 [<a href="javascript:;" @click="openPlant(l)">播种</a>]<br/>
            </template>
            <template v-else-if="l.mature">
              <a href="javascript:;" @click="pick(l)">{{ l.name }}</a>[<a href="javascript:;" @click="pick(l)">收获</a>]<br/>
              (第{{ l.period }}/{{ l.cycle }}季 产{{ l.yield }})<a href="javascript:;" @click="openMuck(l)">[施肥]</a><a href="javascript:;" @click="openTrap(l)">[设陷阱]</a><br/>
            </template>
            <template v-else>
              <a href="javascript:;" @click="landTip(l)">{{ l.name }}</a>
              <span v-if="l.need_water"><a href="javascript:;" @click="care(l, 'water')">[浇水]</a></span>
              <span v-else-if="l.need_weed"><a href="javascript:;" @click="care(l, 'weed')">[除草]</a></span>
              <span v-else-if="l.need_pest"><a href="javascript:;" @click="care(l, 'pest')">[除虫]</a></span>
              <br/>({{ l.status_txt }})<a href="javascript:;" @click="openMuck(l)">[施肥]</a><a href="javascript:;" @click="openTrap(l)">[设陷阱]</a><br/>
            </template>
          </div>
          <div class="row" v-if="!lands.length">您还没有菜地<br/></div>
        </div>
        <br/>

        <div class="name">消息<a href="javascript:;" @click="switchTab('msgs')">({{ msgs.length }})</a></div>
        <div class="list">
          <div class="row" v-for="(m, i) in msgs.slice(0, 3)" :key="m.id">
            {{ i + 1 }}.({{ m.time_txt || '' }})<b>{{ m.nick }}</b> {{ m.msg }}<br/>
          </div>
          <div class="row" v-if="!msgs.length">您没有消息.<br/></div>
          <div class="row" v-if="msgs.length > 3"><a href="javascript:;" @click="switchTab('msgs')">查看更多&gt;&gt;</a><br/></div>
        </div>
        <br/>

        <div class="name">最新加入农场</div>
        <div class="list">
          <div class="row" v-for="r in recent" :key="r.uid">
            <a href="javascript:;" @click="visit(r.uid)">{{ r.nick }}</a>({{ r.level }}级) {{ r.time_txt }}前<br/>
          </div>
          <div class="row" v-if="!recent.length">还没有人开农场，快来抢先体验！<br/></div>
        </div>
        <br/>

        <div class="module-title"><a href="javascript:;" @click="switchTab('bag')">背包</a>.<a href="javascript:;" @click="switchTab('shop')">商店</a>.<a href="javascript:;" @click="switchTab('warehouse')">仓库</a>.<a href="javascript:;" @click="switchTab('rank')">排行</a>.<a href="javascript:;" @click="switchTab('help')">帮助</a>.<a href="javascript:;" @click="goForum">论坛</a><br/></div>
      </template>

      <!-- ============ 邻居（复刻 friends_farm.asp） ============ -->
      <template v-else-if="cur === 'neighbor'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;邻居<br/></div>
        <div class="module-content">可以去邻居的农场帮工、摘菜，被抓到会变成奴隶哦！<br/></div>
        <div class="list">
          <div class="row" v-for="(f, i) in neighbors" :key="f.uid">
            {{ i + 1 }}.<a href="javascript:;" @click="visit(f.uid)">{{ f.nick }}</a>({{ f.name }} {{ f.level }}级)<span v-if="f.is_friend"> [好友]</span><br/>
          </div>
          <div class="row" v-if="!neighbors.length">还没有邻居，先去添加好友吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 参观农场（复刻 visit_farm.asp） ============ -->
      <template v-else-if="cur === 'visit'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('neighbor')">邻居</a>&gt;{{ vFarm.nick }}的农场<br/></div>
        <div class="name">主人:{{ vFarm.nick }} <img class="bicon" src="/static/image/noble_2_1.gif" alt="." />({{ vFarm.level }}级)</div>
        <div class="module-content deep">{{ vFarm.name }}<span v-if="isSlave"> 您是TA的奴隶，两天内不能偷TA的菜！</span></div>
        <div class="name">菜地({{ vLands.length }})<a href="javascript:;" @click="visit(vFarm.uid)">刷新</a></div>
        <div class="list">
          <div class="row" v-for="l in vLands" :key="'v' + l.id">
            <template v-if="l.type === 0">空地<br/></template>
            <template v-else-if="l.mature">
              {{ l.name }}(产{{ l.yield }})<a v-if="canSteal" href="javascript:;" @click="steal(l)">[偷菜]</a><br/>
              <span v-if="!canSteal">(本农场{{ vFarm.csteal === 4 ? '禁止摘取' : '仅好友可摘取' }})</span><br/>
            </template>
            <template v-else>
              {{ l.name }}<span v-if="l.need_water"><a href="javascript:;" @click="careVisit(l, 'water')">[浇水]</a></span><span v-else-if="l.need_weed"><a href="javascript:;" @click="careVisit(l, 'weed')">[除草]</a></span><span v-else-if="l.need_pest"><a href="javascript:;" @click="careVisit(l, 'pest')">[除虫]</a></span><span v-else-if="l.pest === 0"><a href="javascript:;" @click="ppest(l)">[放虫]</a></span><br/>({{ l.status_txt }})<br/>
            </template>
          </div>
          <div class="row" v-if="!vLands.length">对方的农场还没有菜地<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('neighbor')">返回邻居</a><br/>
      </template>

      <!-- ============ 背包（复刻 my_bag.asp） ============ -->
      <template v-else-if="cur === 'bag'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;背包<br/></div>
        <div class="name">我的背包(G币 {{ coins }})</div>
        种子|化肥|陷阱<br/>
        <div class="module-title"><a :class="{ cur: bagTy === 1 }" href="javascript:;" @click="switchBag(1)">种子</a>|<a :class="{ cur: bagTy === 2 }" href="javascript:;" @click="switchBag(2)">化肥</a>|<a :class="{ cur: bagTy === 3 }" href="javascript:;" @click="switchBag(3)">陷阱</a><br/></div>
        <div class="list">
          <div class="row" v-for="b in bag" :key="b.id">
            {{ b.name }}×{{ b.amount }}
            <template v-if="bagTy === 1"><a href="javascript:;" @click="openPlant(null, b)">[播种]</a></template>
            <template v-else-if="bagTy === 2"><a href="javascript:;" @click="openMuck(null, b)">[施肥]</a></template>
            <template v-else><a href="javascript:;" @click="openTrap(null, b)">[设陷阱]</a></template><br/>
          </div>
          <div class="row" v-if="!bag.length">背包里没有{{ bagNames[bagTy] }}，去商店买点吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('shop')">去商店</a><br/>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 商店（复刻 shop_seed.asp / shop_muck.asp / shop_trap.asp） ============ -->
      <template v-else-if="cur === 'shop'">
        <div class="bar sub">【农场商店】<br/></div>
        <div class="module-title"><a :class="{ cur: shopTy === 'seed' }" href="javascript:;" @click="switchShop('seed')">种子</a>|<a :class="{ cur: shopTy === 'muck' }" href="javascript:;" @click="switchShop('muck')">化肥</a>|<a :class="{ cur: shopTy === 'trap' }" href="javascript:;" @click="switchShop('trap')">陷阱</a><br/></div>
        <template v-if="shopTy === 'seed'">
          种子|价格|等级<br/>
          <div class="list">
            <div class="row" v-for="(s, i) in shopSeeds" :key="s.id">
              {{ i + 1 }}.<a href="javascript:;" @click="openSeedDetail(s)">{{ s.name }}</a> {{ s.seed_price }}G币 {{ s.level }}级
              <a href="javascript:;" @click="openBuy('seed', s)">[购买]</a><br/>
              ({{ s.cycle }}季 成熟{{ txtMin(s.aging) }}<span v-if="s.again">/再熟{{ txtMin(s.again) }}</span> 产{{ s.yield }})<br/>
            </div>
          </div>
        </template>
        <template v-else-if="shopTy === 'muck'">
          化肥|效果|价格<br/>
          <div class="list">
            <div class="row" v-for="(m, i) in shop.mucks" :key="'m' + m.id">
              {{ i + 1 }}.{{ m.name }} 成熟提前{{ txtMin(m.speed) }} {{ m.price }}G币 <a href="javascript:;" @click="openBuy('muck', m)">[购买]</a><br/>
            </div>
          </div>
        </template>
        <template v-else>
          陷阱|几率|价格<br/>
          <div class="list">
            <div class="row" v-for="(t, i) in shop.traps" :key="'t' + t.id">
              {{ i + 1 }}.{{ t.name }} 触发几率{{ t.rate }}% {{ t.price }}G币 <a href="javascript:;" @click="openBuy('trap', t)">[购买]</a><br/>
            </div>
          </div>
        </template>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 种子详情（复刻 seed.asp） ============ -->
      <template v-else-if="cur === 'seedinfo'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('shop')">商店</a>&gt;种子<br/></div>
        <div class="module-content">【农场商店】<br/></div>
        <div class="module-content" v-if="curSeed">
          {{ curSeed.name }}<br/>
          种子价格:{{ curSeed.seed_price }}G币<br/>
          果实单价:{{ curSeed.price }}G币<br/>
          种植等级:{{ curSeed.level }}级<br/>
          收获季数:{{ curSeed.cycle }}季<br/>
          成熟时间:{{ txtMin(curSeed.aging) }}<span v-if="curSeed.again">，再次成熟{{ txtMin(curSeed.again) }}</span><br/>
          每季产量:{{ curSeed.yield }}个<br/>
          每季经验:{{ curSeed.point }}点<br/>
          <br/>
          购买<input type="text" v-model.number="buyAmount" maxlength="2" size="2" value="1" />个。<br/>
          <a href="javascript:;" @click="doBuy('seed', curSeed)">确定购买</a><br/>
        </div>
        <a href="javascript:;" @click="switchTab('shop')">返回商店</a><br/>
      </template>

      <!-- ============ 仓库（复刻 warehouse.asp） ============ -->
      <template v-else-if="cur === 'warehouse'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;仓库<br/></div>
        <div class="module-content">收获的果实都存放在仓库中，可以卖出换G币。<br/></div>
        <div class="list">
          <div class="row" v-for="w in wh" :key="w.id">
            {{ w.name }}×{{ w.amount }}(单价{{ w.price }}) <a href="javascript:;" @click="sell(w)">[卖出]</a><br/>
          </div>
          <div class="row" v-if="!wh.length">仓库空空的，快去收获果实吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 排行（复刻 level.asp） ============ -->
      <template v-else-if="cur === 'rank'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;排行<br/></div>
        <div class="module-title">排名|昵称|农场|等级|经验<br/></div>
        <div class="list">
          <div class="row" v-for="r in rankList" :key="r.rank">
            {{ r.rank }}.<a href="javascript:;" @click="visit(r.uid)">{{ r.nick }}</a> | {{ r.name }} | {{ r.level }}级 | {{ r.point }}<br/>
          </div>
          <div class="row" v-if="!rankList.length">暂无排行数据<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 奴隶（复刻 my_slave.asp） ============ -->
      <template v-else-if="cur === 'slaves'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;奴隶<br/></div>
        <div class="module-content">偷菜触发陷阱的主人会变成您的奴隶，可惩罚或安抚（各限6次）。<br/></div>
        <div class="list">
          <div class="row" v-for="s in slaves" :key="s.id">
            {{ s.name }}({{ s.time_txt }}前抓到) 惩罚{{ s.punish }}/6 安抚{{ s.appease }}/6<br/>
            <a href="javascript:;" @click="slaveAct(s, 'punish', '0')">[扫大街]</a><a href="javascript:;" @click="slaveAct(s, 'punish', '1')">[守菜地]</a><a href="javascript:;" @click="slaveAct(s, 'punish', '2')">[关黑屋]</a><a href="javascript:;" @click="slaveAct(s, 'punish', '3')">[发神经]</a><br/>
            <a href="javascript:;" @click="slaveAct(s, 'appease', '0')">[逛商场]</a><a href="javascript:;" @click="slaveAct(s, 'appease', '1')">[住酒店]</a><a href="javascript:;" @click="slaveAct(s, 'appease', '2')">[泡温泉]</a><a href="javascript:;" @click="slaveAct(s, 'appease', '3')">[神秘事件]</a><br/>
          </div>
          <div class="row" v-if="!slaves.length">您还没有奴隶，去好友农场偷菜碰碰运气吧。<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 消息列表（复刻 message_list.asp） ============ -->
      <template v-else-if="cur === 'msgs'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">我的农场</a>&gt;消息<br/></div>
        <div class="module-content">【农场消息】<br/></div>
        <div class="list">
          <div class="row" v-for="(m, i) in msgs" :key="m.id">
            {{ i + 1 }}.({{ m.time_txt || '' }})<b>{{ m.nick }}</b> {{ m.msg }}<br/>
          </div>
          <div class="row" v-if="!msgs.length">您没有消息！<br/></div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回我的农场</a><br/>
      </template>

      <!-- ============ 帮助 ============ -->
      <template v-else-if="cur === 'help'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;帮助<br/></div>
        <div class="list">
          <div class="row" v-for="(h, i) in helpList" :key="i">
            <a href="javascript:;" @click="helpOpen = helpOpen === i ? -1 : i">Q{{ i + 1 }}.{{ h.q }}</a><br/>
            <div class="module-content" v-if="helpOpen === i">{{ h.a }}<br/></div>
          </div>
        </div>
        <a href="javascript:;" @click="switchTab('farm')">返回农场</a><br/>
      </template>

      <!-- ============ 论坛（跳转开心农场游戏论坛） ============ -->
      <template v-else-if="cur === 'forum'">
        <div class="bar sub"><a href="javascript:;" @click="switchTab('farm')">农场</a>&gt;论坛<br/></div>
        <div class="module-content">正在前往开心农场游戏论坛...<br/></div>
      </template>
    </div>

    <!-- 播种弹层 -->
    <div class="mask" v-if="plantBox" @click.self="plantBox = false">
      <div class="panel">
        <div class="name">选择种子（菜地 {{ plantTarget ? plantTarget.sort : '?' }}）</div>
        <div class="list">
          <div class="row" v-for="b in bagSeeds" :key="b.id">
            {{ b.name }}×{{ b.amount }} <a href="javascript:;" @click="plant(b)">[种植]</a><br/>
          </div>
          <div class="row" v-if="!bagSeeds.length">背包没有种子，先去商店买吧。<br/></div>
        </div>
        <a href="javascript:;" @click="plantBox = false; switchTab('shop')">去商店</a> <a href="javascript:;" @click="plantBox = false">关闭</a>
      </div>
    </div>

    <!-- 施肥弹层 -->
    <div class="mask" v-if="muckBox" @click.self="muckBox = false">
      <div class="panel">
        <div class="name">选择化肥（菜地 {{ muckTarget ? muckTarget.sort : '?' }}）</div>
        <div class="list">
          <div class="row" v-for="b in bagMucks" :key="b.id">
            {{ b.name }}×{{ b.amount }} <a href="javascript:;" @click="muck(b)">[施肥]</a><br/>
          </div>
          <div class="row" v-if="!bagMucks.length">背包没有化肥，先去商店买吧。<br/></div>
        </div>
        <a href="javascript:;" @click="muckBox = false; switchTab('shop')">去商店</a> <a href="javascript:;" @click="muckBox = false">关闭</a>
      </div>
    </div>

    <!-- 陷阱弹层 -->
    <div class="mask" v-if="trapBox" @click.self="trapBox = false">
      <div class="panel">
        <div class="name">选择陷阱（菜地 {{ trapTarget ? trapTarget.sort : '?' }}）</div>
        <div class="list">
          <div class="row" v-for="b in bagTraps" :key="b.id">
            {{ b.name }}×{{ b.amount }} <a href="javascript:;" @click="trap(b)">[布置]</a><br/>
          </div>
          <div class="row" v-if="!bagTraps.length">背包没有陷阱，先去商店买吧。<br/></div>
        </div>
        <a href="javascript:;" @click="trapBox = false; switchTab('shop')">去商店</a> <a href="javascript:;" @click="trapBox = false">关闭</a>
      </div>
    </div>

    <!-- 购买弹层 -->
    <div class="mask" v-if="buyBox" @click.self="buyBox = false">
      <div class="panel">
        <div class="name">购买（{{ buyItem ? buyItem.name : '' }}）</div>
        <div class="module-content">
          单价:{{ buyUnitPrice }}G币<br/>
          数量:<input v-model.number="buyAmount" type="number" min="1" max="99" style="width:60px" /><br/>
          合计:{{ buyUnitPrice * (buyAmount || 0) }}G币<br/>
          <a href="javascript:;" @click="doBuy(buyKind, buyItem)">确定购买</a> <a href="javascript:;" @click="buyBox = false">取消</a>
        </div>
      </div>
    </div>

    <!-- 设置弹层（农场名/摘取权限） -->
    <div class="mask" v-if="setBox" @click.self="setBox = false">
      <div class="panel">
        <div class="name">{{ setAct === 1 ? '更改农场名' : '摘取权限设置' }}</div>
        <div class="module-content">
          <template v-if="setAct === 1">
            <input v-model.trim="setName" style="width:180px" placeholder="农场名" /><br/>
          </template>
          <template v-else>
            <label><input type="radio" v-model="setCsteal" :value="0" />所有人可以摘取</label><br/>
            <label><input type="radio" v-model="setCsteal" :value="1" />仅好友可以摘取</label><br/>
            <label><input type="radio" v-model="setCsteal" :value="4" />禁止任何人摘取</label><br/>
          </template>
          <a href="javascript:;" @click="saveSetting">保存</a> <a href="javascript:;" @click="setBox = false">取消</a>
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
  { q: '如何种植作物?', a: '先翻地再播种。种子可以去商店购买获得，等级越高可以买到的种子就越多。作物会经历发芽、小叶子、大叶子、开花、将熟五个阶段，成熟后即可收获。' },
  { q: '照料作物有什么用?', a: '生长过 1/4 后作物可能干旱、长草、生虫，及时浇水、除草、除虫可以保证收成。收获时未处理的旱/草/虫每项扣 1 个产量。帮好友照料同样有经验。' },
  { q: '施肥有什么用?', a: '施肥可以让作物提前成熟。每人每天只有 3 次施肥机会，次日自动恢复。' },
  { q: '陷阱怎么用?', a: '在结果实的菜地上布置陷阱，别人偷菜时有几率触发，触发者会变成您的奴隶，并额外补充 1~5 个果实到地里。' },
  { q: '奴隶有什么用?', a: '成为奴隶的人两天内不能偷您的菜。您可以惩罚奴隶（随机获得 G 币或经验）或安抚奴隶（奴隶获得经验），每个奴隶惩罚、安抚各限 6 次，2 天后自动解除。' },
  { q: '偷菜有什么规则?', a: '成熟后的作物可以偷取一部分，一块地每人只能偷一次。农场主可设置摘取权限：所有人、仅好友或禁止摘取。偷菜会获得等量经验。' },
  { q: '关于经验和等级?', a: '翻地+2、播种+3、浇水/除草/除虫+2、照料收获经验和作物等级相关。升级所需经验为:(当前等级+1)×10 点，每升 5 级系统赠送 1 块菜地。' },
  { q: '多季作物是什么?', a: '玉米、番茄等多季作物收获后自动进入下一季，无需重新播种，收完所有季数后土地恢复为空地。' },
  { q: '果实可以做什么?', a: '收获的果实存放在仓库中，可以卖出换 G 币，等级越高的作物单价越贵。' }
]

export default {
  name: 'Farm',
  data () {
    return {
      cur: 'farm', farm: {}, lands: [], msgs: [], recent: [], coins: 0,
      neighbors: [], vFarm: {}, vLands: [], isSlave: false,
      shop: { seeds: [], mucks: [], traps: [] }, shopTy: 'seed', curSeed: null,
      bag: [], bagTy: 1, wh: [], rankList: [], slaves: [],
      helpList, helpOpen: -1,
      plantBox: false, plantTarget: null, muckBox: false, muckTarget: null, trapBox: false, trapTarget: null,
      buyBox: false, buyKind: 'seed', buyItem: null, buyAmount: 1, forumId: 0,
      setBox: false, setAct: 1, setName: '', setCsteal: 0,
      okMsg: '', msg: ''
    }
  },
  computed: {
    nick () { return (this.$store.state.user || {}).nickname || '神秘农夫' },
    cstealTxt () { return { 0: '所有人', 1: '仅好友', 4: '禁止' }[this.farm.csteal] || '所有人' },
    canSteal () { return this.vFarm.csteal === 0 || (this.vFarm.csteal === 1 && this.isFriendOfTarget) },
    isFriendOfTarget () { const f = this.neighbors.find(n => n.uid === this.vFarm.uid); return !!(f && f.is_friend) },
    bagNames () { return { 1: '种子', 2: '化肥', 3: '陷阱' } },
    bagSeeds () { return this.bag.filter(b => b.dtype === 1) },
    bagMucks () { return this.bag.filter(b => b.dtype === 2) },
    bagTraps () { return this.bag.filter(b => b.dtype === 3) },
    shopSeeds () { return this.shop.seeds || [] },
    buyUnitPrice () {
      if (!this.buyItem) return 0
      if (this.buyKind === 'seed') return this.buyItem.seed_price || 0
      return this.buyItem.price || 0
    }
  },
  mounted () {
    this.loadAll()
  },
  methods: {
    switchTab (tab) {
      this.cur = tab; this.msg = ''; this.okMsg = ''
      if (tab === 'farm') this.load()
      if (tab === 'neighbor') this.loadNeighbors()
      if (tab === 'bag') this.loadBag()
      if (tab === 'shop') this.loadShop()
      if (tab === 'warehouse') this.loadWarehouse()
      if (tab === 'rank') this.loadRank()
      if (tab === 'slaves') this.loadSlaves()
      if (tab === 'msgs') this.load()
    },
    switchBag (ty) { this.bagTy = ty; this.loadBag() },
    switchShop (ty) { this.shopTy = ty; this.cur = 'shop' },
    loadAll () {
      this.load()
      this.loadNeighbors()
      this.loadSlaves()
    },
    load () {
      api.get('/games/farm/view').then(r => {
        if (r.code === 0) {
          this.farm = r.data.farm || {}
          this.coins = r.data.farm.coins || 0
          this.lands = r.data.lands || []
          this.msgs = r.data.msgs || []
          this.recent = r.data.recent || []
        }
      })
    },
    loadNeighbors () {
      api.get('/games/farm/neighbors').then(r => { if (r.code === 0) this.neighbors = r.data || [] })
    },
    loadShop () {
      api.get('/games/farm/shop').then(r => { if (r.code === 0) this.shop = r.data || {} })
    },
    loadBag () {
      api.get('/games/farm/bag?dtype=' + this.bagTy).then(r => { if (r.code === 0) this.bag = r.data || [] })
    },
    loadWarehouse () {
      api.get('/games/farm/warehouse').then(r => { if (r.code === 0) this.wh = r.data || [] })
    },
    loadRank () {
      api.get('/games/farm/rank').then(r => { if (r.code === 0) this.rankList = r.data || [] })
    },
    loadSlaves () {
      api.get('/games/farm/slaves').then(r => { if (r.code === 0) this.slaves = r.data || [] })
    },
    goForum () {
      if (this.forumId) { this.$router.push('/board/' + this.forumId); return }
      // 从板块树中动态找到“开心农场”游戏板块
      api.get('/boards').then(r => {
        if (r.code !== 0) return
        let id = 0
        for (const ch of (r.data || [])) {
          for (const b of (ch.children || [])) {
            if (b.name === '开心农场') id = b.id
          }
        }
        if (id) { this.forumId = id; this.$router.push('/board/' + id) }
      })
    },
    txtMin (min) {
      if (!min) return '0分钟'
      if (min < 60) return min + '分钟'
      return Math.floor(min / 60) + '小时' + (min % 60) + '分钟'
    },
    landTip (l) { this.msg = l.name + ' ' + l.status_txt },
    openHelp (i) { this.cur = 'help'; this.helpOpen = i },
    // ---- 我的农场操作 ----
    plow (l) {
      api.post('/games/farm/plow', { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '翻地成功'; this.load() } else this.msg = r.msg
      })
    },
    openPlant (l, bagItem) {
      if (!l && bagItem) {
        const empty = this.lands.find(x => x.type === 0 && x.plow === 1)
        if (!empty) { this.msg = '没有已翻好的空地，先去翻地吧'; return }
        l = empty
      }
      if (!l) {
        const empty = this.lands.find(x => x.type === 0 && x.plow === 1)
        if (!empty) { this.msg = '没有已翻好的空地，先去翻地吧'; return }
        l = empty
      }
      this.plantTarget = l
      this.loadBag()
      this.plantBox = true
    },
    plant (b) {
      api.post('/games/farm/plant', { land_id: this.plantTarget.id, bag_id: b.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '种植成功'; this.plantBox = false; this.load() } else this.msg = r.msg
      })
    },
    care (l, kind) {
      const names = { water: '浇水', weed: '除草', pest: '除虫' }
      api.post('/games/farm/care/' + kind, { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || (names[kind] + '成功'); this.load() } else this.msg = r.msg
      })
    },
    careVisit (l, kind) {
      const names = { water: '浇水', weed: '除草', pest: '除虫' }
      api.post('/games/farm/care/' + kind, { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || (names[kind] + '成功'); this.visit(this.vFarm.uid) } else this.msg = r.msg
      })
    },
    ppest (l) {
      api.post('/games/farm/ppest', { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '放虫成功'; this.visit(this.vFarm.uid) } else this.msg = r.msg
      })
    },
    openMuck (l, bagItem) {
      if (!l && bagItem) {
        const growing = this.lands.find(x => x.type === 1 && !x.mature)
        if (!growing) { this.msg = '没有在生长的作物'; return }
        l = growing
      }
      if (!l) {
        const growing = this.lands.find(x => x.type === 1 && !x.mature)
        if (!growing) { this.msg = '没有在生长的作物'; return }
        l = growing
      }
      this.muckTarget = l
      this.loadBag()
      this.muckBox = true
    },
    muck (b) {
      api.post('/games/farm/muck', { land_id: this.muckTarget.id, bag_id: b.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '施肥成功'; this.muckBox = false; this.load() } else this.msg = r.msg
      })
    },
    openTrap (l, bagItem) {
      if (!l && bagItem) {
        const growing = this.lands.find(x => x.type === 1)
        if (!growing) { this.msg = '没有可设陷阱的菜地'; return }
        l = growing
      }
      if (!l) {
        const growing = this.lands.find(x => x.type === 1)
        if (!growing) { this.msg = '没有可设陷阱的菜地'; return }
        l = growing
      }
      this.trapTarget = l
      this.loadBag()
      this.trapBox = true
    },
    trap (b) {
      api.post('/games/farm/trap', { land_id: this.trapTarget.id, bag_id: b.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '设陷阱成功'; this.trapBox = false; this.load() } else this.msg = r.msg
      })
    },
    pick (l) {
      api.post('/games/farm/pick', { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '收获成功'; this.load() } else this.msg = r.msg
      })
    },
    // ---- 参观/偷菜 ----
    visit (uid) {
      api.get('/games/farm/visit?uid=' + uid).then(r => {
        if (r.code === 0) {
          this.vFarm = r.data.farm || {}
          this.vLands = r.data.lands || []
          this.isSlave = !!r.data.farm.is_slave
          this.cur = 'visit'
        } else this.msg = r.msg
      })
    },
    steal (l) {
      api.post('/games/farm/steal', { land_id: l.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '偷菜成功'; this.visit(this.vFarm.uid); this.loadSlaves() } else this.msg = r.msg
      })
    },
    // ---- 商店/仓库 ----
    openSeedDetail (s) { this.curSeed = s; this.cur = 'seedinfo' },
    openBuy (kind, item) {
      this.buyKind = kind
      this.buyItem = item
      this.buyAmount = 1
      this.buyBox = true
    },
    doBuy (kind, item) {
      const n = this.buyAmount && this.buyAmount > 0 ? Math.min(this.buyAmount, 99) : 1
      api.post('/games/farm/buy', { kind: kind, id: item.id, amount: n }).then(r => {
        if (r.code === 0) {
          this.okMsg = r.data.msg || '购买成功'
          this.coins = r.data.coins
          this.buyBox = false
          this.load()
        } else this.msg = r.msg
      })
    },
    sell (w) {
      api.post('/games/farm/sell', { bag_id: w.id }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '卖出成功'; this.coins = r.data.coins; this.loadWarehouse() } else this.msg = r.msg
      })
    },
    // ---- 奴隶 ----
    slaveAct (s, kind, act) {
      api.post('/games/farm/slave/' + kind + '/' + s.id, { act: act }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '操作成功'; this.loadSlaves(); this.load() } else this.msg = r.msg
      })
    },
    // ---- 设置 ----
    openSetting (act) {
      this.setAct = act
      this.setName = this.farm.name
      this.setCsteal = this.farm.csteal || 0
      this.setBox = true
    },
    saveSetting () {
      const body = {}
      if (this.setAct === 1) {
        if (!this.setName) { this.msg = '农场名不能为空'; return }
        body.name = this.setName
      } else {
        body.csteal = this.setCsteal
      }
      api.post('/games/farm/setting', body).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg || '修改成功'; this.setBox = false; this.load() } else this.msg = r.msg
      })
    }
  }
}
</script>

<style scoped>
/* ===== 复刻 3gqq.ink 参考站 style.css（与 Garden.vue 一致） ===== */
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
.module-title a { color: #2e9cd3; text-decoration: none; }
.module-title a.cur { color: #e65100; font-weight: bold; }
.deep { background: #E3EEF8; border: 1px solid #9FC6EC; border-left: none; border-right: none; }
.list { line-height: 1.6; font-size: 13px; }
.row { padding: 3px; border-bottom: 1px solid #E3E6EB; }
.row a { color: #2e9cd3; text-decoration: none; }
.g-main { padding: 2px 3px; }
.userline img { vertical-align: middle; }
.bicon { vertical-align: middle; }
.mask { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,.4); z-index: 99; display: flex; align-items: center; justify-content: center; }
.panel { background: #fff; border-radius: 8px; width: 320px; max-height: 70vh; overflow: auto; padding: 12px; }
.panel input { border: 1px solid #ccc; border-radius: 3px; padding: 2px 4px; font-size: 13px; }
.panel a { color: #2e9cd3; text-decoration: none; margin: 0 4px; }
.okmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #43a047; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
.errmsg { position: fixed; top: 60px; left: 50%; transform: translateX(-50%); background: #d32f2f; color: #fff; padding: 8px 18px; border-radius: 16px; font-size: 13px; z-index: 100; }
</style>
