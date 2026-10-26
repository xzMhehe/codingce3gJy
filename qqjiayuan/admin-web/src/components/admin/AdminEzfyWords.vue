<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战征途 · 聊天管理（世界/军团/系统/私聊 聊天记录 + 系统邮件 + 敏感词维护）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="onRefresh">刷新</el-button>
      </div>

      <!-- ★ 2026-10-17 聊天管理多 tab：世界(1)/军团(2)/系统(4)/聊天敏感词 -->
      <!-- ★ 2026-10-24 私聊管理合并进来（原独立「私聊管理」菜单移除）：新增「私聊」tab + 系统邮件 -->
      <el-tabs v-model="tab" @tab-click="onTab">
        <el-tab-pane label="世界" name="chat1" />
        <el-tab-pane label="军团" name="chat2" />
        <el-tab-pane label="系统" name="chat4" />
        <el-tab-pane label="私聊" name="priv" />
        <el-tab-pane label="聊天敏感词" name="words" />
      </el-tabs>

      <!-- ========== 聊天记录（世界/军团/系统 共用一套面板，tab 切换频道） ========== -->
      <div v-if="tab !== 'words' && tab !== 'priv'">
        <div class="toolbar">
          <el-input v-model="chat.word" placeholder="按玩家/内容搜索" clearable style="width:200px"
                    @keyup.enter.native="chat.page = 1; loadChats()" />
          <span class="td-sub" style="margin:0 4px 0 8px">时间</span>
          <el-date-picker v-model="chat.start" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="开始时间" style="width:180px" />
          <span class="td-sub">—</span>
          <el-date-picker v-model="chat.end" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" placeholder="结束时间" style="width:180px" />
          <el-button type="primary" icon="el-icon-search" @click="chat.page = 1; loadChats()">查询</el-button>
          <el-button type="danger" icon="el-icon-delete" @click="batchDelete">删除所选时间段</el-button>
          <div class="grow" />
          <span class="td-sub">当前频道：<b>{{ channelName }}</b> · 共 {{ chat.total }} 条</span>
        </div>
        <el-table :data="chat.list" v-loading="chat.loading" stripe border>
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column label="频道" width="80" align="center">
            <template slot-scope="{row}">
              <el-tag :type="row.channel === 4 ? 'info' : row.channel === 2 ? 'success' : 'primary'" size="mini">
                {{ row.channel === 4 ? '系统' : row.channel === 2 ? '军团' : '世界' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="user_name" label="玩家" min-width="120" show-overflow-tooltip />
          <el-table-column label="类型" width="90" align="center">
            <template slot-scope="{row}">
              <el-tag :type="row.talk_type === 0 ? 'info' : 'warning'" size="mini">{{ row.talk_type === 0 ? '系统' : '玩家' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="content" label="内容" min-width="260" show-overflow-tooltip />
          <el-table-column label="时间" width="165" align="center">
            <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="removeChat(row)" />
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ chat.total }}</b> 条 · 每页 {{ chat.size }} 条</div>
          <el-pagination v-show="chat.total > 0" small background layout="sizes, prev, pager, next, jumper" :total="chat.total" :page-size="chat.size"
                         :current-page="chat.page" :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { chat.page = p; loadChats() }"
                         @size-change="s => { chat.size = s; chat.page = 1; loadChats() }" />
        </div>
      </div>

      <!-- ========== 私聊管理（★ 2026-10-24 从独立「私聊管理」页合并进来） ========== -->
      <div v-else-if="tab === 'priv'">
        <div class="toolbar">
          <el-input v-model="priv.word" placeholder="内容 / 昵称 / 游戏号 / 用户ID" clearable style="width:250px"
                    @keyup.enter.native="priv.page = 1; loadPrivs()" />
          <el-select v-model="priv.isRead" style="width:130px" @change="priv.page = 1; loadPrivs()">
            <el-option label="全部" :value="-1" />
            <el-option label="未读" :value="0" />
            <el-option label="已读" :value="1" />
          </el-select>
          <el-button type="primary" icon="el-icon-search" @click="priv.page = 1; loadPrivs()">查询</el-button>
          <div class="grow" />
          <el-button type="warning" icon="el-icon-delete" @click="openClear">清空某玩家私聊</el-button>
        </div>
        <!-- ★ 2026-10-26 重做批量删除勾选（修「无法多选」）：
             不再依赖 el-table 的 reserve-selection（翻页/刷新后勾选行为不稳定），
             改为显式维护 privSelIds（跨页勾选的唯一真源），@select/@select-all 只改用户点的行。 -->
        <el-table ref="privTable" :data="priv.list" row-key="id" v-loading="priv.loading" stripe border max-height="640"
                  @select="onPrivSelect" @select-all="onPrivSelectAll">
          <el-table-column type="selection" width="46" align="center" />
          <el-table-column prop="id" label="ID" width="80" align="center" />
          <!-- ★ 2026-10-25 修「发送方/接收方空白」：名字统一走 privName/privNum 兜底，
               后端没给名字时至少显示「用户+ID」，绝不再渲染出空白或「用户undefined」 -->
          <el-table-column label="发送方" min-width="160" show-overflow-tooltip>
            <template slot-scope="{row}">
              <span class="td-main">{{ privName(row, 'sender') }}</span>
              <span v-if="privNum(row, 'sender') !== '-'" class="td-muted">（{{ privNum(row, 'sender') }}）</span>
            </template>
          </el-table-column>
          <el-table-column label="接收方" min-width="160" show-overflow-tooltip>
            <template slot-scope="{row}">
              <span class="td-main">{{ privName(row, 'receiver') }}</span>
              <span v-if="privNum(row, 'receiver') !== '-'" class="td-muted">（{{ privNum(row, 'receiver') }}）</span>
            </template>
          </el-table-column>
          <el-table-column prop="content" label="内容" min-width="320" show-overflow-tooltip />
          <el-table-column label="状态" width="100" align="center">
            <template slot-scope="{row}">
              <el-tag size="mini" :type="row.is_read === 1 ? 'success' : 'warning'">{{ row.is_read === 1 ? '已读' : '未读' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="170" align="center">
            <template slot-scope="{row}">{{ fmtTimePriv(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="110" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delPriv(row)" />
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ priv.total }}</b> 条 · 每页 {{ priv.size }} 条</div>
          <div style="display:flex;align-items:center;gap:8px">
          <el-button size="mini" type="danger" plain icon="el-icon-delete"
                     :disabled="!privSelIds.length" :loading="priv.saving" @click="delPrivSelected">删除选中（{{ privSelIds.length }}）</el-button>
            <el-pagination v-show="priv.total > 0" small background layout="sizes, prev, pager, next, jumper" :total="priv.total" :page-size="priv.size"
                           :current-page="priv.page" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { priv.page = p; loadPrivs() }"
                           @size-change="s => { priv.size = s; priv.page = 1; loadPrivs() }" />
          </div>
        </div>

        <!-- 清空某玩家私聊 -->
        <el-dialog title="清空某玩家私聊" :visible.sync="clearDlg" width="480px" :close-on-click-modal="false">
          <el-form label-width="120px" size="small">
            <el-form-item label="用户ID" required>
              <el-input-number v-model.number="clearForm.user_id" :min="1" controls-position="right" />
              <div class="td-sub" style="margin-top:4px">即会员用户ID（不是家园号）</div>
            </el-form-item>
            <el-form-item label="清空范围">
              <el-radio-group v-model="clearForm.dir">
                <el-radio label="all">全部（收发）</el-radio>
                <el-radio label="from">该玩家发出的</el-radio>
                <el-radio label="to">该玩家收到的</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-form>
          <!-- [说明·不显示在界面] 提示：此操作会物理删除私信记录，不可恢复 -->
          <div slot="footer">
            <el-button @click="clearDlg = false">取 消</el-button>
            <el-button type="danger" :loading="priv.saving" @click="doClear">清 空</el-button>
          </div>
        </el-dialog>

        <!-- ★ 2026-10-13 系统邮件：发全体 / 发指定玩家 + 发送记录（★ 2026-10-24 做分页） -->
        <div class="box sub-box" style="margin-top:14px">
          <div slot="header" class="card-head">
            <b>系统邮件</b>（进入游戏「邮箱」→「系统邮件」可见；target 0 = 全体玩家）
          </div>
          <el-form inline size="small">
            <el-form-item label="收件人">
              <el-input-number v-model.number="mailForm.target_id" :min="0" controls-position="right" style="width:160px" />
            </el-form-item>
            <el-form-item label="主题">
              <el-input v-model="mailForm.subject" maxlength="20" show-word-limit placeholder="最多20字" style="width:220px" />
            </el-form-item>
            <el-form-item label="内容">
              <el-input v-model="mailForm.content" type="textarea" :rows="2" maxlength="500" show-word-limit placeholder="最多500字" style="width:380px" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="mailSending" @click="sendMail">发 送</el-button>
              <el-button @click="mail.page = 1; loadSysMails()">刷新记录</el-button>
            </el-form-item>
          </el-form>
          <!-- ★ 2026-10-25 支持删除：row-key + reserve-selection = 翻页后勾选不丢，可跨页一次删 -->
          <!-- ★ 2026-10-26 系统邮件勾选同样改为显式 id 集合（与私聊同一套，修「无法多选」） -->
          <el-table ref="mailTable" :data="sysMails" row-key="id" v-loading="mail.loading" stripe border max-height="420"
                    @select="onMailSelect" @select-all="onMailSelectAll">
            <el-table-column type="selection" width="46" align="center" />
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column label="收件人" min-width="130" show-overflow-tooltip>
              <template slot-scope="{row}">
                <!-- ★ 2026-10-24 修复「收件人显示 玩家undefined」：后端返回 receiver_id/receiver_name，
                     前端原来读 row.target_id（接口没有这个字段）→ 全体邮件(0)永远不走「全体玩家」分支，
                     指定收件人名字为空时拼出「玩家undefined」。统一读 receiver_id。 -->
                <span class="td-main">{{ row.receiver_id === 0 ? '全体玩家' : (row.receiver_name || ('玩家' + row.receiver_id)) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="subject" label="主题" min-width="140" show-overflow-tooltip />
            <el-table-column prop="content" label="内容" min-width="200" show-overflow-tooltip />
            <!-- ★ 2026-10-25 附件列：发放道具走的是「系统邮件 + 附件」，发错时要一眼看出这封带了什么 -->
            <el-table-column label="附件" min-width="180" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="!row.has_attach" class="td-muted">—</span>
                <span v-else class="td-main">{{ attachText(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="92" align="center">
              <template slot-scope="{row}">
                <el-tag v-if="!row.has_attach" size="mini" type="info">纯文本</el-tag>
                <el-tag v-else-if="row.claimed === 1" size="mini" type="success">已领取</el-tag>
                <el-tag v-else-if="isExpired(row)" size="mini" type="info">已过期</el-tag>
                <el-tag v-else size="mini" type="warning">待领取</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="批次" width="92" align="center">
              <template slot-scope="{row}">
                <el-tooltip v-if="row.batch_no" :content="row.batch_no" placement="top">
                  <span class="td-mono">{{ row.batch_total || 1 }} 封</span>
                </el-tooltip>
                <span v-else class="td-muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="时间" width="160" align="center">
              <template slot-scope="{row}">{{ fmtTimePriv(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="164" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除这封（连同附件）" @click="delMail(row)" />
                <!-- 群发/全员发放是每人一封、共用一个批次号：逐条勾选不现实，给一个「删本批」 -->
                <el-button v-if="row.batch_no && row.batch_total > 1" size="mini" type="danger" icon="el-icon-delete-solid"
                           title="删除本批次全部邮件" @click="delMailBatch(row)">本批{{ row.batch_total }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ mail.total }}</b> 条 · 每页 {{ mail.size }} 条</div>
            <div style="display:flex;align-items:center;gap:8px">
              <el-button size="mini" type="danger" plain icon="el-icon-delete"
                         :disabled="!mailSelIds.length" @click="delMailSelected">删除选中（{{ mailSelIds.length }}）</el-button>
              <el-pagination v-show="mail.total > 0" small background layout="sizes, prev, pager, next, jumper" :total="mail.total" :page-size="mail.size"
                             :current-page="mail.page" :page-sizes="[5, 10, 20, 50, 100]"
                             @current-change="p => { mail.page = p; loadSysMails() }"
                             @size-change="s => { mail.size = s; mail.page = 1; loadSysMails() }" />
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 聊天敏感词维护 ========== -->
      <div v-else>
        <div class="toolbar">
          <el-input v-model="word" placeholder="按敏感词搜索" clearable style="width:220px"
                    @keyup.enter.native="page = 1; loadWords()" @clear="page = 1; loadWords()" />
          <el-button type="primary" icon="el-icon-search" @click="page = 1; loadWords()">查询</el-button>
          <el-button type="success" icon="el-icon-plus" @click="openAdd">新增敏感词</el-button>
          <el-button type="warning" plain icon="el-icon-upload2" @click="bulkDlg = true">批量导入</el-button>
          <div class="grow" />
          <span class="td-sub">类型：<b>替换</b> = 用替换词覆盖（留空则打 *）；<b>拦截</b> = 直接拒绝发言</span>
        </div>
        <!-- ★ 2026-10-26 用户反馈「敏感词页空白」加固：
             ① 搜索框点 × 也重新查询（原来只回车才查，看起来像「列表空了」）；
             ② 空列表给明确文案；③ 类型按 Number 比较（老库返回字符串时原来恒显示「替换」）。 -->
        <el-table :data="list" v-loading="loading" stripe border empty-text="暂无敏感词，点右上角「新增敏感词」添加">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column label="敏感词" min-width="170" show-overflow-tooltip>
            <template slot-scope="{row}"><span class="td-main">{{ row.word }}</span></template>
          </el-table-column>
          <el-table-column label="替换词" min-width="170" show-overflow-tooltip>
            <template slot-scope="{row}"><span class="td-mono">{{ row.replace || '（打 * ）' }}</span></template>
          </el-table-column>
          <el-table-column label="类型" width="110" align="center">
            <template slot-scope="{row}">
              <el-tag :type="Number(row.type) === 2 ? 'danger' : 'warning'" size="mini">
                {{ Number(row.type) === 2 ? '拦截' : '替换' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
              <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="remove(row)" />
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
          <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                         :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                         @current-change="p => { page = p; loadWords() }"
                         @size-change="s => { size = s; page = 1; loadWords() }" />
        </div>
      </div>
    </el-card>

    <el-dialog :title="formId ? '编辑敏感词' : '新增敏感词'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="敏感词">
          <el-input v-model="form.word" maxlength="50" placeholder="要过滤的词" />
        </el-form-item>
        <el-form-item label="替换词">
          <el-input v-model="form.replace" maxlength="50" placeholder="留空则用 * 覆盖" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model.number="form.type">
            <el-radio :label="1">替换</el-radio>
            <el-radio :label="2">拦截（拒绝发言）</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <el-dialog title="批量导入敏感词" :visible.sync="bulkDlg" width="640px" :close-on-click-modal="false">
      <div class="td-sub" style="margin-bottom:8px">
        每行一条，格式：<b>词</b> 或 <b>词,替换词</b> 或 <b>词,替换词,类型</b>（类型 1=替换 2=拦截）；<b>#</b> 开头的行忽略。
      </div>
      <el-input type="textarea" v-model="bulkText" :rows="10" placeholder="赌博,***,2&#10;外挂" />
      <div slot="footer">
        <el-button @click="bulkDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doBulk">导 入</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-17 聊天管理：原「聊天敏感词」页面升级。
//   tab：chat1 世界 / chat2 军团 / chat4 系统（共用一套聊天面板）+ words 敏感词维护。
//   世界/军团/系统 支持按时间范围物理删除（DELETE /admin/ezfy-chats/batch）。
// ★ 2026-10-24 私聊管理合并进来：新增「私聊」tab（私聊记录 + 清空某玩家私聊）+ 系统邮件（发送 + 分页记录）。
//   原独立「私聊管理」菜单（AdminEzfyPrivchat.vue）已移除，此页即合并后的入口。
export default {
  name: 'AdminEzfyWords',
  data () {
    return {
      tab: 'chat1',
      channelName: '世界',
      // 聊天记录（世界/军团/系统 三频道共用）
      chat: { list: [], total: 0, page: 1, size: 20, word: '', start: '', end: '', loading: false },
      // 私聊记录（★ 2026-10-24 并入）
      priv: { list: [], total: 0, page: 1, size: 5, word: '', isRead: -1, loading: false, saving: false },
      // ★ 2026-10-26 私聊批量删除：跨页勾选 id 集合（唯一真源，配合 syncPrivSel 回显）
      privSelIds: [],
      clearDlg: false, clearForm: { user_id: 1, dir: 'all' },
      // 系统邮件（★ 2026-10-24 分页）
      mailForm: { target_id: 0, subject: '', content: '' },
      mail: { total: 0, page: 1, size: 5, loading: false },
      sysMails: [], mailSending: false,
      // ★ 2026-10-26 系统邮件删除：跨页勾选 id 集合（与私聊同一套）
      mailSelIds: [],
      // 敏感词
      list: [], total: 0, page: 1, size: 5, word: '', loading: false,
      dlg: false, formId: 0, form: { word: '', replace: '', type: 1 }, saving: false,
      bulkDlg: false, bulkText: ''
    }
  },
  mounted () { this.onTab() },
  methods: {
    channelOf (tab) { return { chat1: 1, chat2: 2, chat4: 4 }[tab] || 0 },
    channelNameOf (tab) { return { chat1: '世界', chat2: '军团', chat4: '系统' }[tab] || '' },
    onTab () {
      if (this.tab === 'words') this.loadWords()
      else if (this.tab === 'priv') { this.loadPrivs(); this.loadSysMails() }
      else { this.channelName = this.channelNameOf(this.tab); this.loadChats() }
    },
    onRefresh () {
      if (this.tab === 'words') this.loadWords()
      else if (this.tab === 'priv') { this.loadPrivs(); this.loadSysMails() }
      else this.loadChats()
    },
    fmtTime (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '' },
    fmtTimePriv (t) { return t ? new Date(t).toLocaleString() : '' },
    loadChats () {
      this.chat.loading = true
      api.get('/admin/ezfy-chats', {
        params: { page: this.chat.page, size: this.chat.size, word: this.chat.word, channel: this.channelOf(this.tab), start: this.chat.start, end: this.chat.end }
      }).then(r => {
        this.chat.loading = false
        if (r.code === 0) { this.chat.list = r.data.list || []; this.chat.total = r.data.total || 0 } else this.$message.error(r.msg)
      }).catch(() => { this.chat.loading = false })
    },
    removeChat (row) {
      this.$confirm('删除该聊天记录（ID ' + row.id + '）？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-chats/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            if (this.chat.list.length <= 1 && this.chat.page > 1) this.chat.page--
            this.loadChats()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    batchDelete () {
      if (!this.chat.start || !this.chat.end) { this.$message.error('请先选择开始/结束时间'); return }
      this.$confirm(`删除【${this.channelName}】频道 ${this.chat.start} 至 ${this.chat.end} 的全部聊天记录？该操作不可恢复！`,
        '确认物理删除', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-chats/batch', {
          params: { channel: this.channelOf(this.tab), start: this.chat.start, end: this.chat.end }
        }).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg + '，共 ' + (r.data.count || 0) + ' 条')
            this.chat.page = 1
            this.loadChats()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 私聊记录（★ 2026-10-24 并入聊天管理） ----
    loadPrivs () {
      this.priv.loading = true
      api.get('/admin/ezfy-privchats', {
        params: { page: this.priv.page, size: this.priv.size, word: this.priv.word, is_read: this.priv.isRead }
      }).then(r => {
        this.priv.loading = false
        if (r.code === 0) {
          this.priv.list = r.data.list || []
          this.priv.total = r.data.total || 0
          this.priv.page = r.data.page || 1
          // ★ 2026-10-26 翻页/刷新后回显跨页勾选
          this.$nextTick(() => this.syncPrivSel())
        } else this.$message.error(r.msg)
      }).catch(() => { this.priv.loading = false })
    },
    // ★ 2026-10-25 修「发送方/接收方空白」：名字/游戏号统一在这里兜底。
    //   后端正常会下发 sender_name/sender_num（实测有值）；万一某台旧后端没下发，
    //   这里退回「用户+ID」，绝不再渲染出空白或「用户undefined」。
    // ★ 2026-10-26 重做批量勾选（修「无法多选」）：
    //   不再依赖 el-table 的 reserve-selection（翻页/数据刷新后勾选行为不稳定，会出现
    //   「勾了第二条第一条被取消」或「勾选全丢」），改为显式维护跨页 id 集合 privSelIds：
    //   @select/@select-all 只处理用户点的那一行/当前页，翻页回来由 syncPrivSel 回显。
    privName (row, who) {
      const n = row[who + '_name']
      if (n && String(n).trim()) return String(n)
      const id = row[who + '_id']
      return (id === 0 || id) ? ('用户' + id) : '未知'
    },
    privNum (row, who) {
      const num = row[who + '_num']
      if (num !== undefined && num !== null && String(num) !== '') return num
      const id = row[who + '_id']
      return (id === 0 || id) ? id : '-'
    },
    onPrivSelect (sel, row) {
      const ids = new Set(this.privSelIds)
      const checked = (sel || []).some(r => r.id === row.id)
      if (checked) ids.add(row.id)
      else ids.delete(row.id)
      this.privSelIds = Array.from(ids)
    },
    onPrivSelectAll (sel) {
      const ids = new Set(this.privSelIds)
      const pageIds = (this.priv.list || []).map(r => r.id)
      const allChecked = pageIds.length > 0 && pageIds.every(id => (sel || []).some(r => r.id === id))
      pageIds.forEach(id => { allChecked ? ids.add(id) : ids.delete(id) })
      this.privSelIds = Array.from(ids)
    },
    // 翻页/刷新后按 privSelIds 回显当前页勾选
    syncPrivSel () {
      const t = this.$refs.privTable
      if (!t) return
      t.clearSelection()
      ;(this.priv.list || []).forEach(r => {
        if (this.privSelIds.indexOf(r.id) >= 0) t.toggleRowSelection(r, true)
      })
    },
    delPrivSelected () {
      const ids = this.privSelIds.slice()
      if (!ids.length) { this.$message.warning('请先勾选要删除的私聊'); return }
      this.$confirm(`删除选中的 ${ids.length} 条私聊记录？物理删除，不可恢复。`, '确认删除', { type: 'warning' }).then(() => {
        this.priv.saving = true
        api.post('/admin/ezfy-privchats/batch-delete', { ids: ids }).then(r => {
          this.priv.saving = false
          if (r.code === 0) {
            this.$message.success(r.data && r.data.msg ? r.data.msg : '已删除')
            this.privSelIds = []
            if (this.$refs.privTable) this.$refs.privTable.clearSelection()
            // 当前页被删空 → 回退一页，避免停在空页
            if (this.priv.list.length <= ids.length && this.priv.page > 1) this.priv.page--
            this.loadPrivs()
          } else this.$message.error(r.msg)
        }).catch(() => { this.priv.saving = false })
      }).catch(() => {})
    },
    delPriv (row) {
      this.$confirm('确认删除这条私信记录？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-privchats/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            // ★ 2026-10-26 单条删除同样要清掉跨页勾选集合里的这个 id
            this.privSelIds = this.privSelIds.filter(id => id !== row.id)
            if (this.priv.list.length <= 1 && this.priv.page > 1) this.priv.page--
            this.loadPrivs()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openClear () {
      this.clearForm = { user_id: 1, dir: 'all' }
      this.clearDlg = true
    },
    doClear () {
      this.$confirm('清空后不可恢复，确认继续？', '危险操作', { type: 'error' }).then(() => {
        this.priv.saving = true
        api.post('/admin/ezfy-privchats/clear', this.clearForm).then(r => {
          this.priv.saving = false
          if (r.code === 0) { this.clearDlg = false; this.$message.success(r.data.msg || '已清空'); this.loadPrivs() }
          else this.$message.error(r.msg)
        }).catch(() => { this.priv.saving = false })
      }).catch(() => {})
    },
    // ---- 系统邮件（★ 2026-10-24 记录分页） ----
    loadSysMails () {
      this.mail.loading = true
      api.get('/admin/ezfy-mails/system', {
        params: { page: this.mail.page, size: this.mail.size }
      }).then(r => {
        this.mail.loading = false
        if (r.code === 0) {
          this.sysMails = r.data.list || []
          this.mail.total = r.data.total || 0
          // ★ 2026-10-26 翻页/刷新后回显跨页勾选
          this.$nextTick(() => this.syncMailSel())
        } else this.$message.error(r.msg)
      }).catch(() => { this.mail.loading = false })
    },
    // ---- 系统邮件删除（★ 2026-10-25 用户需求：发错了 / 发错道具了要能一起删除） ----
    // ★ 2026-10-26 勾选改为显式 mailSelIds（与私聊同一套，修「无法多选」）
    onMailSelect (sel, row) {
      const ids = new Set(this.mailSelIds)
      const checked = (sel || []).some(r => r.id === row.id)
      if (checked) ids.add(row.id)
      else ids.delete(row.id)
      this.mailSelIds = Array.from(ids)
    },
    onMailSelectAll (sel) {
      const ids = new Set(this.mailSelIds)
      const pageIds = (this.sysMails || []).map(r => r.id)
      const allChecked = pageIds.length > 0 && pageIds.every(id => (sel || []).some(r => r.id === id))
      pageIds.forEach(id => { allChecked ? ids.add(id) : ids.delete(id) })
      this.mailSelIds = Array.from(ids)
    },
    syncMailSel () {
      const t = this.$refs.mailTable
      if (!t) return
      t.clearSelection()
      ;(this.sysMails || []).forEach(r => {
        if (this.mailSelIds.indexOf(r.id) >= 0) t.toggleRowSelection(r, true)
      })
    },
    attachText (row) {
      const as = row.attaches || []
      if (!as.length) return '—'
      return as.map(a => a.name + '×' + a.num).join('、')
    },
    isExpired (row) { return row.expire_at > 0 && Date.now() >= row.expire_at },
    // 删除确认文案：把「撤回多少 / 收不回多少」说清楚（发错道具时管理员要判断是否另行补发）
    mailDelTip (rows) {
      const unclaimed = rows.filter(r => r.has_attach === 1 && r.claimed !== 1).length
      const claimed = rows.filter(r => r.has_attach === 1 && r.claimed === 1).length
      let tip = `删除 <b>${rows.length}</b> 封系统邮件，邮件与附件一起物理删除，<b>不可恢复</b>。`
      if (unclaimed) tip += `<br/>其中 ${unclaimed} 封的附件尚未领取，删除后玩家无法再领取（等于撤回）。`
      if (claimed) tip += `<br/>其中 <b>${claimed}</b> 封的附件玩家已领取，删除邮件<b>不会</b>收回已到手的道具。`
      return tip
    },
    postDelMail (payload, removedCount, removedIds) {
      this.mail.loading = true
      api.post('/admin/ezfy-mails/system/delete', payload).then(r => {
        this.mail.loading = false
        if (r.code === 0) {
          this.$message.success(r.data && r.data.msg ? r.data.msg : '已删除')
          // 删除成功后把已删的 id 从跨页勾选集合里清掉（否则「删除选中」还记着已删的行）
          if (removedIds && removedIds.length) {
            const gone = new Set(removedIds)
            this.mailSelIds = this.mailSelIds.filter(id => !gone.has(id))
          }
          if (this.$refs.mailTable) this.$refs.mailTable.clearSelection()
          // 当前页被删空 → 回退一页，避免停在空页（列表看起来「数据没了」）
          if (this.sysMails.length <= removedCount && this.mail.page > 1) this.mail.page--
          this.loadSysMails()
        } else this.$message.error(r.msg)
      }).catch(() => { this.mail.loading = false })
    },
    delMail (row) {
      this.$confirm(this.mailDelTip([row]), '确认删除系统邮件',
        { type: 'warning', dangerouslyUseHTMLString: true }).then(() => {
        this.postDelMail({ ids: [row.id] }, 1, [row.id])
      }).catch(() => {})
    },
    delMailBatch (row) {
      const n = row.batch_total || 1
      this.$confirm(`删除本批次全部 <b>${n}</b> 封系统邮件，邮件与附件一起物理删除，<b>不可恢复</b>。<br/>批次：${row.batch_no}`,
        '确认删除整批', { type: 'warning', dangerouslyUseHTMLString: true }).then(() => {
        this.postDelMail({ batch_no: row.batch_no }, n)
      }).catch(() => {})
    },
    delMailSelected () {
      const ids = this.mailSelIds.slice()
      if (!ids.length) { this.$message.warning('请先勾选要删除的邮件'); return }
      const rows = (this.sysMails || []).filter(r => ids.indexOf(r.id) >= 0)
      this.$confirm(this.mailDelTip(rows.length ? rows : ids.map(id => ({ id }))), '确认删除所选系统邮件',
        { type: 'warning', dangerouslyUseHTMLString: true }).then(() => {
        this.postDelMail({ ids: ids }, rows.length, ids)
      }).catch(() => {})
    },
    sendMail () {
      const subject = (this.mailForm.subject || '').trim()
      const content = (this.mailForm.content || '').trim()
      if (!content) { this.$message.warning('请填写邮件内容'); return }
      this.mailSending = true
      api.post('/admin/ezfy-mails/system', { target_id: this.mailForm.target_id, subject: subject, content: content }).then(r => {
        this.mailSending = false
        if (r.code === 0) {
          this.$message.success(r.data && r.data.msg ? r.data.msg : '已发送')
          this.mailForm.content = ''
          this.mail.page = 1
          this.loadSysMails()
        } else this.$message.error(r.msg)
      }).catch(() => { this.mailSending = false })
    },
    // ---- 敏感词 ----
    loadWords () {
      this.loading = true
      api.get('/admin/ezfy-word-filters', {
        params: { page: this.page, size: this.size, word: this.word }
      }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list || []; this.total = r.data.total || 0 } else this.$message.error(r.msg)
      // ★ 2026-10-26 修「敏感词页空白」：请求失败（403/500/断网）时原来 loading 永远为 true
      //   → 表格一直转圈、看着像「一条数据都没有」。必须复位。
      }).catch(() => { this.loading = false })
    },
    openAdd () { this.formId = 0; this.form = { word: '', replace: '', type: 1 }; this.dlg = true },
    // ★ 2026-10-26 类型按 Number 取（老库可能返回字符串），否则编辑弹窗单选框会跳回「替换」
    openEdit (row) { this.formId = row.id; this.form = { word: row.word, replace: row.replace, type: Number(row.type) === 2 ? 2 : 1 }; this.dlg = true },
    save () {
      if (!this.form.word.trim()) { this.$message.error('敏感词不能为空'); return }
      this.saving = true
      const isNew = !this.formId
      const done = r => {
        this.saving = false
        if (r.code === 0) {
          this.dlg = false
          this.$message.success(r.data.msg || '已保存')
          // ★ 2026-10-11 新增后清空搜索词并回到第 1 页：
          //   列表按 id DESC（新加的排最前）。若沿用上次的搜索条件 / 页码，刚加的词会被筛掉，
          //   列表看起来「只剩一条 / 其他数据都没有了」（用户反馈「新增了分页其他数据就没有了」）。
          if (isNew) this.word = ''
          this.page = 1
          this.loadWords()
        } else this.$message.error(r.msg)
      }
      if (this.formId) api.put('/admin/ezfy-word-filters/' + this.formId, this.form).then(done)
      else api.post('/admin/ezfy-word-filters', this.form).then(done)
    },
    remove (row) {
      this.$confirm('删除敏感词「' + row.word + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-word-filters/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            // ★ 2026-10-11 删掉当前页最后一条 → 回退一页，避免停在空页（列表看起来「数据没了」）
            if (this.list.length <= 1 && this.page > 1) this.page--
            this.loadWords()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doBulk () {
      if (!this.bulkText.trim()) { this.$message.error('请填写要导入的内容'); return }
      this.saving = true
      api.post('/admin/ezfy-word-filters/bulk', { text: this.bulkText }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.bulkDlg = false
          this.bulkText = ''
          this.$message.success(r.data.msg || '导入完成')
          // ★ 2026-10-26 修「导入后列表看起来是空的」：批量导入的词排在最前，
          //   若沿用上次搜索词，新导入的词会被筛掉 → 一并清空搜索、回第 1 页。
          this.word = ''
          this.page = 1
          this.loadWords()
        }
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
</style>
