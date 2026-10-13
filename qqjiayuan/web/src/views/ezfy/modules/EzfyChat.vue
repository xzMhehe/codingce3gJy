<template>
  <div>
    <template v-if="ezfy.cur === 'chat'">
        <div class="panel">
          <div class="panel-title">聊天频道</div>
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 1 }" @click="ezfy.switchChannel(1)">世界</a>|
            <!-- ★ 军团频道常显：没加入军团时进去显示「未加入 · 0 人」 -->
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 2 }" @click="ezfy.switchChannel(2)">军团</a>|
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 4 }" @click="ezfy.switchChannel(4)">系统</a>
          </div>

          <!-- ★ 发言框移到聊天列表**上方** 列表按时间降序、最新在最上面
               ★ 2026-10-13 去掉系统频道上方的 [刷新]（系统消息上方不再有手动刷新入口） -->
          <div class="old-line ezfy-chat-send" v-if="ezfy.chatChannel !== 4">
            <template v-if="ezfy.chatCanSend">
              <input v-model="ezfy.chatMsg" class="ezfy-chat-input" maxlength="25" @keyup.enter="ezfy.doChatSend"/>
              <a v-if="ezfy.chatCooldown <= 0" href="javascript:;" @click="ezfy.doChatSend">[发送]</a>
              <span v-else class="gray">冷却中 {{ ezfy.chatCooldown }}s</span>
              <span class="gray">(最大25个字)</span>
            </template>
          </div>

          <!-- 系统频道: 系统消息(只读) ★ 2026-09-29 去掉「系统公告」——首页已有公告入口(置顶公告+底部导航「公告」) -->
          <template v-if="ezfy.chatChannel === 4">
            <div class="panel-title">系统消息</div>
            <div class="old-line" v-for="ch in ezfy.worldChats" :key="'cs' + ch.id">
              [<span class="orange">系统</span>]
              <span class="gray">{{ ezfy.fmtTime(ch.created_at) }}</span>
              ：{{ ch.content }}
            </div>
            <div class="old-line gray" v-if="!ezfy.worldChats.length">(暂无系统消息)</div>
            <div class="ezfy-pager" v-if="ezfy.chatTotalPages > 1">
              <a href="javascript:;" :class="{ gray: ezfy.chatPage <= 1 }" @click="ezfy.chatGo(-1)">上一页</a>
              <span class="gray">第 {{ ezfy.chatPage }}/{{ ezfy.chatTotalPages }} 页（共 {{ ezfy.chatTotal }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.chatPage >= ezfy.chatTotalPages }" @click="ezfy.chatGo(1)">下一页</a>
            </div>
          </template>

          <!-- 军团频道但还没加入军团：显示 0 人 + 引导 -->
          <template v-else-if="ezfy.chatChannel === 2 && !ezfy.chatHasCorps">
            <div class="old-line gray">
              你还没有加入军团（军团人数 0）。
              <a href="javascript:;" @click="ezfy.go('corps')">[去看看军团列表]</a>
            </div>
          </template>

          <!-- 公共 / 军团频道 -->
          <template v-else>
            <div class="panel-title">
              {{ ezfy.chatChannel === 2
                ? '军团聊天(' + (ezfy.chatHasCorps ? ezfy.chatCorpsName : '未加入军团') + ')(' + ezfy.chatCorpsPlayers + '人)'
                : '世界聊天(' + ezfy.chatPlayers + '人)' }}
            </div>
            <div class="old-line" v-for="ch in ezfy.worldChats" :key="'c' + ch.id">
              [<span class="orange">{{ ezfy.chatChannel === 2 ? '军团' : '世界' }}</span>]
              <span class="gray">{{ ezfy.fmtTime(ch.created_at) }}</span>
              <a href="javascript:;" @click="ezfy.openPlayer(ch.user_id)"><span
                 v-for="(c, ci) in ezfy.nickChars(ch.user_name)" :key="'ncc' + ci"
                 :style="ezfy.nickColorAt(ch.color, ci)">{{ c }}</span></a>：{{ ch.content }}
            </div>
            <div class="old-line" v-if="!ezfy.worldChats.length">(暂无消息, 快来说点什么吧)</div>
            <div class="ezfy-pager" v-if="ezfy.chatTotalPages > 1">
              <a href="javascript:;" :class="{ gray: ezfy.chatPage <= 1 }" @click="ezfy.chatGo(-1)">上一页</a>
              <span class="gray">第 {{ ezfy.chatPage }}/{{ ezfy.chatTotalPages }} 页（共 {{ ezfy.chatTotal }} 条）</span>
              <a href="javascript:;" :class="{ gray: ezfy.chatPage >= ezfy.chatTotalPages }" @click="ezfy.chatGo(1)">下一页</a>
            </div>
          </template>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyChat',
  inject: ['ezfy']
}
</script>
