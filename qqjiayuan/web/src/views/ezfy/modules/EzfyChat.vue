<template>
  <div>
    <template v-if="ezfy.cur === 'chat'">
        <div class="panel">
          <div class="panel-title">聊天频道</div>
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 1 }" @click="ezfy.switchChannel(1)">世界</a>|
            <!-- ★ 军团频道常显：没加入军团时进去显示「未加入 · 0 人」 -->
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 2 }" @click="ezfy.switchChannel(2)">军团</a>|
            <a href="javascript:;" :class="{ on: ezfy.chatChannel === 4 }" @click="ezfy.switchChannel(4)">系统</a>|
            <a href="javascript:;" @click="ezfy.openPm()">私聊</a>
          </div>

          <!-- ★ 发言框移到聊天列表**上方** 列表按时间降序、最新在最上面 -->
          <div class="old-line ezfy-chat-send">
            <template v-if="ezfy.chatCanSend">
              <input v-model="ezfy.chatMsg" class="ezfy-chat-input" maxlength="25" @keyup.enter="ezfy.doChatSend"/>
              <a v-if="ezfy.chatCooldown <= 0" href="javascript:;" @click="ezfy.doChatSend">[发送]</a>
              <span v-else class="gray">冷却中 {{ ezfy.chatCooldown }}s</span>
              <span class="gray">(最大25个字)</span>
            </template>
            <span v-else class="gray">(系统频道仅系统可发言)</span>
            <a href="javascript:;" @click="ezfy.loadChats">[刷新]</a>
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
    <template v-else-if="ezfy.cur === 'mail'">
        <div class="panel">
          <div class="panel-title">私聊 · 会话列表</div>
          <div class="old-line" v-for="c in ezfy.pmConvs" :key="'cv' + c.user_id">
            <a href="javascript:;" @click="ezfy.selectPm(c.user_id)">
              <span :class="{ red: ezfy.pmPeer && ezfy.pmPeer.id === c.user_id }">
                {{ ezfy.pmPeer && ezfy.pmPeer.id === c.user_id ? '▶ ' : '' }}{{ c.nickname }}</span></a>
            <span class="gray">({{ c.username }})</span>
            <span v-if="c.unread > 0" class="red">[未读{{ c.unread }}]</span>
            <br/>
            <span class="gray">{{ c.last_content }}</span>
            <span class="gray"> ({{ ezfy.fmtTime(c.last_at) }})</span>
          </div>
          <div class="old-line" v-if="!ezfy.pmConvs.length">(还没有聊过的人，在下面填游戏ID或昵称发起私聊)</div>
          <div class="old-line" v-if="ezfy.pmConvs.length && ezfy.pmConvTotal > ezfy.pmConvSize">
            共{{ ezfy.pmConvTotal }}个会话 ·
            <a href="javascript:;" :class="ezfy.pmConvPage <= 1 ? 'gray' : ''" @click="ezfy.pmConvPrev()">[上一页]</a>
            {{ ezfy.pmConvPage }}/{{ Math.ceil(ezfy.pmConvTotal / ezfy.pmConvSize) }}
            <a href="javascript:;" :class="ezfy.pmConvPage * ezfy.pmConvSize >= ezfy.pmConvTotal ? 'gray' : ''" @click="ezfy.pmConvNext()">[下一页]</a>
          </div>
          <br/>
          <button @click="ezfy.loadPmConvs()">刷新会话</button>
        </div>

        <div class="panel" v-if="ezfy.pmPeer">
          <div class="panel-title">与 {{ ezfy.pmPeer.nickname }}({{ ezfy.pmPeer.username }}) 的聊天记录</div>
          <div class="old-line" v-for="m in ezfy.pmChat" :key="'pc' + m.id">
            <span :class="m.sender_id === ezfy.myUserId ? 'green' : ''">
              {{ m.sender_id === ezfy.myUserId ? '我' : ezfy.pmPeer.nickname }}</span>：{{ m.content }}
            <span class="gray">({{ ezfy.fmtTime(m.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!ezfy.pmChat.length">(还没有聊天记录，发一条试试)</div>
          <br/>
          <button @click="ezfy.selectPm(ezfy.pmPeer.id)">刷新记录</button>
          <a href="javascript:;" @click="ezfy.pmPeer = null; ezfy.pmChat = []; ezfy.pmTo = ''">[关闭会话]</a>
        </div>

        <div class="panel">
          <div class="panel-title">发私信</div>
          <div class="old-line">
            收件人:
            <input v-model="ezfy.pmTo" placeholder="游戏ID / 昵称" style="width:170px" list="ezfyPmCands"/>
            <datalist id="ezfyPmCands">
              <option v-for="f in ezfy.pmCandidates" :key="'pmc' + f.id" :value="f.name"></option>
            </datalist>
            <span class="gray" v-if="ezfy.pmPeer">（当前会话：{{ ezfy.pmPeer.nickname }}）</span>
          </div>
          <div class="old-line gray">不需要先加好友, 填对方游戏ID或昵称即可; 对方把你拉黑则发不出去。</div>
          <div class="old-line">
            <!-- ★ 改成可自适应高度的文本域（）：随内容长高，最多 8 行后内部滚动 -->
            <textarea v-model="ezfy.pmContent" class="ezfy-auto-textarea" placeholder="最多500字"
                      maxlength="500" rows="2"></textarea>
          </div>
          <div class="old-line">
            <button @click="ezfy.doSendPm">发送</button>
            <a href="javascript:;" @click="ezfy.go('friends')">[好友]</a>
            <a href="javascript:;" @click="ezfy.go('chat')">[聊天频道]</a>
          </div>
        </div>

        <!-- 收到的私信（系统通知 / 别人的来信） -->
        <div class="panel">
          <div class="panel-title">收到的私信</div>
          <div class="old-line" v-for="m in ezfy.mails" :key="'m' + m.id">
            <a href="javascript:;" @click="ezfy.selectPm(m.sender_id)"><span
               :class="{ red: m.is_read === 0 }">{{ m.sender }}</span></a>:
            {{ m.content }} <span class="gray">({{ ezfy.fmtTime(m.created_at) }})</span>
          </div>
          <div class="old-line" v-if="!ezfy.mails.length">(暂无私信)</div>
          <br/>
          <a href="javascript:;" @click="ezfy.loadMails">[刷新]</a>
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
