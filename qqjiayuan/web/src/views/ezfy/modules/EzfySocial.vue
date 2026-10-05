<template>
  <div>
    <template v-if="ezfy.cur === 'friends'">
        <div class="panel">
          <div class="panel-title">游戏内好友</div>
          <div class="panel-title">搜索玩家（按游戏ID / 玩家号码 / 昵称）</div>
          <div class="old-line">
            <input v-model="ezfy.friendKeyword" placeholder="输入游戏ID / 玩家号码 / 昵称" style="width:170px"/>
            <button @click="ezfy.doFriendSearch">[搜索]</button>
          </div>
          <table v-if="ezfy.friendSearchDone">
            <tr><th>游戏ID</th><th>昵称</th><th>声望</th><th>状态</th><th>操作</th></tr>
            <tr v-for="u in ezfy.friendSearchList" :key="'fs' + u.user_id">
              <td><span class="td-mono">{{ u.game_uid }}</span></td>
              <td><a href="javascript:;" @click="ezfy.openPlayer(u.user_id)">{{ u.nickname }}</a></td>
              <td>{{ u.prestige }}</td>
              <td><span class="gray">{{ u.rank_name }}</span></td>
              <td>
                <span v-if="u.is_friend" class="gray">已是好友</span>
                <span v-else-if="u.applied" class="orange">已申请</span>
                <a v-else href="javascript:;" @click="ezfy.doAddFriend(u)">[加好友]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="ezfy.friendSearchDone && !ezfy.friendSearchList.length">(没找到这位统帅, 换个游戏ID或昵称试试)</div>

          <div class="panel-title">好友申请（待处理 {{ ezfy.friendApplies.inbox.length }}）</div>
          <table v-if="ezfy.friendApplies.inbox.length">
            <tr><th>游戏ID</th><th>昵称</th><th>验证信息</th><th>操作</th></tr>
            <tr v-for="a in ezfy.friendApplies.inbox" :key="'fa' + a.apply_id">
              <td><span class="td-mono">{{ a.game_uid }}</span></td>
              <td><a href="javascript:;" @click="ezfy.openPlayer(a.user_id)">{{ a.nickname }}</a></td>
              <td>{{ a.remark || '—' }}</td>
              <td>
                <a href="javascript:;" @click="ezfy.doHandleApply(a, true)">[同意]</a>
                <a href="javascript:;" @click="ezfy.doHandleApply(a, false)">[拒绝]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-else>(暂无新的好友申请)</div>

          <div class="panel-title">我的游戏好友（{{ ezfy.friends.length }}）</div>
          <table v-if="ezfy.friends.length">
            <tr><th>游戏ID</th><th>昵称</th><th>声望</th><th>军衔</th><th>操作</th></tr>
            <tr v-for="f in ezfy.friends" :key="'f' + f.user_id">
              <td><span class="td-mono">{{ f.game_uid }}</span></td>
              <td><a href="javascript:;" @click="ezfy.openPlayer(f.user_id)">{{ f.nickname }}</a></td>
              <td>{{ f.prestige }}</td>
              <td>{{ f.rank_name }}</td>
              <td>
                <a href="javascript:;" @click="ezfy.openPlayer(f.user_id)">[统帅信息]</a>
                <a href="javascript:;" @click="ezfy.openPm(f.user_id)">[私聊]</a>
                <a href="javascript:;" @click="ezfy.doDelFriend(f)">[删除]</a>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-else>(还没有游戏好友, 用上面的搜索找找吧)</div>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'tasks'">
        <div class="panel">
          <!-- ★ 2026-09-24  任务按分类 tab 分别展示(新手/日常/每周) -->
          <div class="acade-tab">
            <template v-for="(g, i) in ezfy.taskGroups">
              <!-- ★ 分隔竖线放 <a> 外: 选中态(加粗变色)不波及竖线 -->
              <span v-if="i > 0" :key="'ts' + g.id"> | </span>
              <!-- ★ 2026-09-27 fix: 选中判断需 taskTab !== -1, 否则停在"为爱发电卡"时会高亮"新手任务" -->
              <a :key="'tgt' + g.id" href="javascript:;"
                 :class="{ on: ezfy.taskTab !== -1 && ezfy.taskGroupCur.id === g.id }" @click="ezfy.selectTaskTab(g.id)"><span>{{ g.name }}</span></a>
            </template>
            <!-- ★ 2026-09-27 为爱发电卡 tab: 管理端未发放(love_cards 为空)时不显示 -->
            <span v-if="ezfy.loveCards.length && ezfy.taskGroups.length"> | </span>
            <a v-if="ezfy.loveCards.length" href="javascript:;"
               :class="{ on: ezfy.taskTab === -1 }" @click="ezfy.selectTaskTab(-1)"><span>为爱发电卡</span></a>
          </div>

          <!-- ★ 为爱发电卡内容（多卡可叠加领取） -->
          <template v-if="ezfy.taskTab === -1 && ezfy.loveCards.length">
            <div style="margin-top:6px">
              <div class="old-line" v-for="c in ezfy.loveCards" :key="c.id">
                <b>{{ c.name }}</b> 每日 <b>{{ c.daily_diamond }}</b> 钻石 · 已 <b>{{ c.claimed_days }}/{{ c.total_days }}</b> 天
                <span v-if="c.remaining > 0" class="gray">（剩{{ c.remaining }}天）</span>
                <span v-else class="green">（已领完）</span>
                <span v-if="c.claimable > 0"> · 可领 <b>{{ c.claimable }}</b> 天</span>
              </div>
              <div class="old-line">
                <a v-if="ezfy.loveTotalClaimable > 0" href="javascript:;" @click="ezfy.doLoveCardClaim">[领取每日钻石]</a>
                <span v-else class="gray">[今日暂无可领取, 明天再来]</span>
              </div>
              <div class="sub gray">漏领的天数会在之后领取时累加补齐（每卡封顶 {{ 30 }} 天）</div>
            </div>
          </template>

          <!-- 普通分类任务 -->
          <template v-else-if="ezfy.taskTab !== -1">
            <div v-if="ezfy.taskGroupCur.tasks.length" style="margin-top:6px">
              <div class="old-line" v-for="t in ezfy.taskGroupCur.tasks" :key="t.id">
                <b>{{ t.name }}</b> {{ t.current }}/{{ t.target }}
                <span v-if="t.status === 2" class="gray">[已领取]</span>
                <a v-else-if="t.status === 1" href="javascript:;" @click="ezfy.doAward(t)">[领奖]</a>
                <br/>
                <span class="gray">奖励:{{ ezfy.rewardText(t.reward) }}</span>
              </div>
            </div>
            <div class="old-line" v-else>(暂无任务)</div>
          </template>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfySocial',
  inject: ['ezfy']
}
</script>
