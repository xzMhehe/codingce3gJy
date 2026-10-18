<template>
  <div>
    <template v-if="ezfy.cur === 'mail'">
        <div class="panel">
          <div class="old-line">
            未读：<b class="red">{{ ezfy.mailUnread.total }}</b> 封（收件箱 <b>{{ ezfy.mailUnread.inbox }}</b> / 系统邮件 <b>{{ ezfy.mailUnread.system }}</b>）
            <a v-if="ezfy.mailTab === 'inbox' || ezfy.mailTab === 'system'" href="javascript:;" @click="ezfy.doMailReadAll">[全部已读]</a>
            <a href="javascript:;" @click="ezfy.loadMail(ezfy.mailTab, true)">[刷新]</a>
          </div>
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.mailTab === 'inbox' }" @click="ezfy.setMailTab('inbox')">收件箱</a>|
            <a href="javascript:;" :class="{ on: ezfy.mailTab === 'system' }" @click="ezfy.setMailTab('system')">系统邮件</a>|
            <a href="javascript:;" :class="{ on: ezfy.mailTab === 'sent' }" @click="ezfy.setMailTab('sent')">发件箱</a>|
            <a href="javascript:;" :class="{ on: ezfy.mailTab === 'compose' }" @click="ezfy.setMailTab('compose')">写邮件</a>
          </div>

          <!-- 写邮件 -->
          <template v-if="ezfy.mailTab === 'compose'">
            <div class="old-line">
              收件人：
              <input v-model="ezfy.mailCompose.to" placeholder="输入玩家昵称或游戏ID" style="width:170px"/>
            </div>
            <div class="old-line">
              主题：
              <input v-model="ezfy.mailCompose.subject" maxlength="20" placeholder="最多20字" style="width:170px"/>
              <span class="gray">{{ ezfy.mailCompose.subject.length }}/20</span>
            </div>
            <div class="old-line">
              内容：
              <textarea v-model="ezfy.mailCompose.content" maxlength="500" rows="3" class="ezfy-auto-textarea" placeholder="最多500字"></textarea>
              <span class="gray">{{ ezfy.mailCompose.content.length }}/500</span>
            </div>
            <div class="old-line gray">收件人昵称须完全一致。24小时内最多50封。</div>
            <div class="old-line">
              <button @click="ezfy.doSendMail">发送</button>
            </div>
          </template>

          <!-- 收件箱 / 系统邮件 / 发件箱 -->
          <template v-else>
            <div class="old-line" v-for="m in ezfy.mailList(ezfy.mailTab)" :key="'ml' + m.id">
              <a href="javascript:;" @click="ezfy.mailToggle(m)">
                <span :class="m.is_read === 0 ? 'red' : ''">{{ m.subject || '(无主题)' }}</span>
              </a>
              <span v-if="m.attach && m.attach.length" class="red">[附件]</span>
              <span class="gray">({{ ezfy.mailSender(m) }} · {{ ezfy.fmtTime(m.created_at) }})</span>
              <a v-if="m.is_read === 0" href="javascript:;" @click="ezfy.doMailRead(m)">[标已读]</a>
              <a href="javascript:;" @click="ezfy.doMailDelete(m)">[删除]</a>
              <div v-if="ezfy.mailOpen === m.id" class="sub" style="margin-top:4px;white-space:pre-wrap;word-break:break-all;">{{ m.content }}</div>
              <!-- ★ 2026-10-18 邮件附件（风云榜奖励 / 系统发放）：玩家点 [领取] 才到账，7 天不领过期 -->
              <template v-if="ezfy.mailOpen === m.id && m.attach && m.attach.length">
                <div class="sub" style="margin-top:4px;">
                  附件：<span v-for="(a,i) in m.attach" :key="'at' + i">{{ i ? '、' : '' }}{{ a.name }}×{{ a.num }}</span>
                </div>
                <div class="old-line" style="margin-top:2px;">
                  <a v-if="m.can_claim" href="javascript:;" @click="ezfy.doMailClaim(m)">[领取]</a>
                  <span v-else-if="m.claimed === 1" class="gray">[已领取]</span>
                  <span v-else-if="m.expired" class="gray">[已过期]</span>
                </div>
              </template>
            </div>
            <div class="old-line gray" v-if="!ezfy.mailList(ezfy.mailTab).length">
              {{ ezfy.mailTab === 'inbox' ? '收件箱暂无邮件。' : ezfy.mailTab === 'system' ? '暂无系统邮件。' : '发件箱暂无邮件。' }}
            </div>
            <div class="ezfy-pager" v-if="ezfy.mailTotal > ezfy.mailSize">
              <a href="javascript:;" :class="{ gray: ezfy.mailPage <= 1 }" @click="ezfy.mailPrev()">上一页</a>
              <span class="gray">第 {{ ezfy.mailPage }}/{{ Math.ceil(ezfy.mailTotal / ezfy.mailSize) }} 页（共 {{ ezfy.mailTotal }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.mailPage * ezfy.mailSize >= ezfy.mailTotal }" @click="ezfy.mailNext()">下一页</a>
            </div>
          </template>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyMail',
  inject: ['ezfy']
}
</script>
