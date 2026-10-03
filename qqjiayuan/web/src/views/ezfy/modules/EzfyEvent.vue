<template>
  <div>
    <!-- ============ 活动(activity) ============ -->
    <template v-if="ezfy.cur === 'activity'">
      <div class="panel">
        <div class="panel-title">活动</div>
        <!-- 复刻 activityIndex.html 的【活动玩法】说明(在地图中寻找的固定位置活动目标) -->
        <div class="old-line"><b>【活动玩法】</b>(在【地图】中寻找, 固定位置刷新)：</div>
        <div class="old-line">
          <span class="orange">活动野地</span>【活动】(标记: 活动野地N级)<br/>
          陆/海随机刷新, 10万~30万守军, 胜利获得大量资源+{{ ezfy.resNames.gold }}(元宝)+必定掉落宝物+大量声望<br/>
          等级越高守军越强, 奖励越丰厚
        </div>
        <div class="old-line">
          <span class="orange">活动寇城</span>【活动寇】(标记: 活动寇N级)<br/>
          20万~60万守军, 胜利获得巨大资源+{{ ezfy.resNames.gold }}+宝物+声望
        </div>
        <div class="old-line">
          <span class="red">特殊城市</span>【特殊】(标记: 特殊城市N级)<br/>
          100万~500万守军, 全服最强活动目标, 需要强力的部队!<br/>
          胜利必定获得高级/特殊宝物, 巨额{{ ezfy.resNames.gold }}与资源
        </div>
        <div class="old-line gray">活动目标无法占领, 战胜只结算奖励, 不占附属野地上限。</div>
        <a href="javascript:;" @click="ezfy.go('map')">[前往地图]</a>
        <hr/>
        <div class="old-line"><b>【节日活动】</b></div>
        <template v-if="ezfy.activities.length">
          <div class="old-line" v-for="a in ezfy.activities" :key="'ac' + a.id">
            <b>{{ a.name }}</b>
            <span :class="a.running ? 'green' : 'gray'">[{{ a.running ? '进行中' : '未开启' }}]</span>
            <span class="orange">{{ a.effect }}</span><br/>
            {{ a.des }}<br/>
            <span v-if="a.running" class="gray">剩余: {{ ezfy.fmtLeft(a.left_sec) }}</span>
            <span v-else-if="a.start_time && a.end_time" class="gray">
              时间: {{ ezfy.fmtTime(a.start_time) }} ~ {{ ezfy.fmtTime(a.end_time) }}
            </span>
          </div>
        </template>
        <div class="old-line gray" v-else>(暂无节日活动, 敬请期待)</div>
        <hr/>
        <div class="old-line gray">活动类型: 资源增产 / 造兵打折 / 建造加速 / 研究加速 / 声望加成</div>
        <hr/>
        <div class="old-line">[开服活动] 新手礼包、每周福利、市政厅等级礼包持续发放中, 前往<a href="javascript:;" @click="ezfy.go('welfare')">[福利]</a>领取。</div>
        <div class="old-line">[征战天下] 征服野地/寇城可获得军功声望, 声望晋升军衔!</div>
        <div class="old-line">[物资兑换] 交易所开放资源交易, 低买高卖赚{{ ezfy.resNames.gold }}。</div>
        <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
      </div>
    </template>

    <!-- ============ 福利(welfare) ============ -->
    <template v-else-if="ezfy.cur === 'welfare'">
      <div class="panel">
        <!-- ★ 2026-09-28 用户要求：签到/礼包/宝物签到 拆成 tab 展示（照抄 rank 页 .acade-tab 写法） -->
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: ezfy.welfareTab === 0 }" @click="ezfy.setWelfareTab(0)">每日签到</a><span
            class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.welfareTab === 1 }" @click="ezfy.setWelfareTab(1)">礼包</a><span
            class="acade-sep">.</span><a href="javascript:;" :class="{ on: ezfy.welfareTab === 2 }" @click="ezfy.setWelfareTab(2)">宝物签到</a>
        </div>

        <!-- 每日签到 -->
        <template v-if="ezfy.welfareTab === 0">
          <div class="old-line">
            <span v-if="ezfy.welfare.signed_today">今日已签到(连续{{ ezfy.welfare.sign_count }}天)</span>
            <a v-else href="javascript:;" @click="ezfy.doSign">[签到领奖]</a>
            | 声望:{{ ezfy.welfare.prestige }}({{ ezfy.welfare.rank_name }})
          </div>
          <div class="old-line" v-for="r in ezfy.welfare.rewards" :key="'sr' + r.day">
            第{{ r.day }}天:{{ r.reward }}
          </div>
        </template>

        <!-- 礼包 -->
        <!-- ★ 用户要求「把 [市政厅20级礼包][市政厅30级礼包][市政厅40级礼包] 删掉」：
             只保留 新手 / 每周 / 市政厅10级 三个入口（后端 Gift 同步去掉 20/30/40 分支）。 -->
        <template v-else-if="ezfy.welfareTab === 1">
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.doGift('newbie')">{{ ezfy.welfare.gifts.newbie ? '[新手礼包已领]' : '[新手礼包]' }}</a>
            <a href="javascript:;" @click="ezfy.doGift('weekly')">{{ ezfy.welfare.gifts.weekly ? '[每周福利已领]' : '[每周福利]' }}</a><br/>
            <a href="javascript:;" @click="ezfy.doGift('level10')">{{ ezfy.welfare.gifts.level10 ? '[市政厅10级礼包已领]' : '[市政厅10级礼包]' }}</a>
          </div>
        </template>

        <!-- 宝物签到：7 天一轮，逢 5/6/7 天多给（懒人也能攒晋升宝物） -->
        <template v-else>
          <div class="old-line">
            宝物签到
            <span class="gray">每天领随机宝物，7天一轮：第1-4天×2、第5天×4、第6天×6、第7天×8</span>
          </div>
          <div class="old-line">
            <span v-if="ezfy.welfare.treasure_signed_today" class="green">今日已签(连续{{ ezfy.welfare.treasure_count }}天)</span>
            <a v-else href="javascript:;" @click="ezfy.doTreasureSign">[宝物签到领奖]</a>
            <span v-if="ezfy.welfare.treasure_count" class="gray">| 连续{{ ezfy.welfare.treasure_count }}天</span>
          </div>
          <!-- ★ 2026-09-28 用户要求：展示本轮已签到领到的具体宝物名, 每轮(每天签到)后更新 -->
          <div class="old-line" v-if="ezfy.welfare.treasure_signed_today && ezfy.welfare.treasure_reward">
            本轮已领宝物：<span class="green">{{ ezfy.welfare.treasure_reward.split(',').join('、') }}</span>
          </div>
        </template>

        <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
      </div>
    </template>

    <!-- ============ 联络中心(liaison) 复刻 liaison/liaisonIndex.html ============ -->
    <template v-else-if="ezfy.cur === 'liaison'">
      <div class="panel">
        <div class="old-line">联络中心（{{ ezfy.liaison.level }}级）</div>
        <div class="old-line">联络中心是盟友间互相联络的建筑</div>
        <div class="old-line">
          1级联络中心可以 加入联盟，<br/>
          2级联络中心可以 创建联盟<br/>
          创建联盟需消耗{{ ezfy.liaison.create_cost }}{{ ezfy.resNames.gold }}<br/>
          每级联络中心可以多一支盟友驻军、多{{ ezfy.liaison.member_per_level }}人联盟人数上限
        </div>
        <div class="old-line gray">使用同盟密令1个可以将联络中心升级至11级（原版道具，本项目未开放）</div>
        <div class="old-line red" v-if="ezfy.liaison.level < 1">
          尚未建造联络中心, 无法加入或创建联盟
          <a href="javascript:;" @click="ezfy.go('buildm')">[前往军事区建造]</a>
        </div>

        <div class="old-line">我的联盟：</div>
        <template v-if="ezfy.liaison.my_corps">
          <div class="old-line">
            <b>{{ ezfy.liaison.my_corps.name }}</b>(成员{{ ezfy.liaison.member_count }}/{{ ezfy.liaison.member_cap }})
            <a href="javascript:;" @click="ezfy.go('corps')">[进入军团]</a>
          </div>
          <div class="old-line gray">公告：{{ ezfy.liaison.my_corps.notice || '暂无公告' }}</div>
        </template>
        <div class="old-line" v-else-if="ezfy.liaison.level >= 1">
          <a href="javascript:;" @click="ezfy.go('corps')">加入联盟</a>
          <template v-if="ezfy.liaison.can_create">
            &nbsp;<a href="javascript:;" @click="ezfy.go('corps')">创建联盟</a>
          </template>
          <span v-else class="red">（创建联盟需2级联络中心）</span>
        </div>
        <div class="old-line gray" v-else>（尚未建造联络中心）</div>

        <div class="old-line">盟军驻军：</div>
        <table>
          <tr><th>来自城市</th><th>军官</th><th>驻军</th><th></th></tr>
          <tr v-for="g in ezfy.liaison.garrisons" :key="'lg' + g.id">
            <td>{{ g.from_city }}</td>
            <td>{{ g.officer || '无' }}</td>
            <td>
              <span v-for="(t, i) in g.troops" :key="'lgt' + i">{{ t.name }}×{{ t.count }} </span>
            </td>
            <!-- ★ 2026-10-02 城主可遣返盟军驻军(驻军返航回出发城市) -->
            <td><a href="javascript:;" class="red" @click="ezfy.expelGarrison(g)">[遣返]</a></td>
          </tr>
        </table>
        <div class="old-line gray" v-if="!ezfy.liaison.garrisons.length">(暂无盟军驻军)</div>
        <div class="old-line gray">
          驻军上限 {{ ezfy.liaison.garrison_used }}/{{ ezfy.liaison.garrison_cap }}；
          联盟成员可用「增援」把部队派到你的城市协防。
        </div>
        <a href="javascript:;" @click="ezfy.go('buildm')">[返回军事区]</a>
        <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
      </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyEvent',
  inject: ['ezfy']
}
</script>
