<template>
  <div>
    <template v-if="ezfy.cur === 'corps'">
        <!-- ★ 2026-09-25 用户要求：军团页拆成四栏（纯前端 tab，照抄 rank/acade 页 .acade-tab 写法） -->
        <div class="panel">
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'info' }" @click="ezfy.switchCorpsTab('info')">军团信息</a>|
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'list' }" @click="ezfy.switchCorpsTab('list')">军团列表</a>|
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'chat' }" @click="ezfy.switchCorpsTab('chat')">军团聊天</a>|
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'diplomacy' }" @click="ezfy.switchCorpsTab('diplomacy')">军团外交</a>|
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'war' }" @click="ezfy.switchCorpsTab('war')">军团宣战</a>|
            <a href="javascript:;" :class="{ on: ezfy.corpsTab === 'mall' }" @click="ezfy.switchCorpsTab('mall')">军团商城</a>
          </div>
        </div>

        <!-- ========== ① 军团信息（原有内容整体移入，不删任何原功能） ========== -->
        <template v-if="ezfy.corpsTab === 'info'">
        <template v-if="ezfy.myCorps">
          <div class="panel">
            <div class="panel-title">我的军团:{{ ezfy.myCorps.name }}({{ ezfy.myCorps.member_count }}人)</div>
            公告: {{ ezfy.myCorps.notice || '无' }}<br/>
            <!-- ★ 2026-09-25 用户要求：显示军团总积分（来自 /corps/members 的 corps_points） -->
            <div class="old-line">军团总积分: <b>{{ ezfy.corpsPoints }}</b></div>
            <div class="old-line">
              <template v-if="ezfy.isLeader">
                <a href="javascript:;" @click="ezfy.openNoticeEdit()">[修改公告]</a>
                <a class="red" href="javascript:;" @click="ezfy.doLeaveCorps()">[解散军团]</a>
              </template>
              <a v-else href="javascript:;" @click="ezfy.doLeaveCorps()">[退出军团]</a>
            </div>
            <div class="panel-title">军团成员</div>
            <table class="ezfy-corps-tbl ezfy-mem-tbl">
              <tr>
                <th>成员</th><th>职位</th><th>军衔</th>
                <!-- ★ 2026-09-25 用户要求：成员表格新增「军团积分」列（m.points，个人军团积分） -->
                <th>军团积分</th>
                <!-- ★ 第九轮：军团长可任命副团长/参谋长 -->
                <th v-if="ezfy.isLeader" width="150">任命</th>
              </tr>
              <tr v-for="m in ezfy.corpsMembers" :key="'cm' + m.user_id">
                <td><a href="javascript:;" @click="ezfy.openPlayer(m.user_id)">{{ m.name }}</a></td>
                <td>{{ m.title || '成员' }}</td>
                <td>{{ m.rank_name }}</td>
                <!-- ★ 2026-09-25：个人军团积分（用于军团商城兑换） -->
                <td>{{ m.points }}</td>
                <td v-if="ezfy.isLeader && (m.is_leader || m.title)">
                  <!-- ★ 2026-10-04 用户规则：非官职（无职位 title）的成员不展示操作列，
                       只有已任官职（副团长/参谋长）或军团长本人才有内容；
                       ★ 2026-10-05 无职位的成员连操作列单元格都不渲染 -->
                  <template v-if="m.is_leader">
                    <span class="gray">军团长</span>
                  </template>
                  <template v-else-if="m.title">
                    <a v-if="m.title !== '副团长'" href="javascript:;" @click="ezfy.doSetCorpsTitle(m, '副团长')">[副团长]</a>
                    <a v-if="m.title !== '参谋长'" href="javascript:;" @click="ezfy.doSetCorpsTitle(m, '参谋长')">[参谋长]</a>
                    <a v-if="m.title" class="gray" href="javascript:;" @click="ezfy.doSetCorpsTitle(m, '')">[撤职]</a>
                  </template>
                </td>
              </tr>
            </table>
            <div class="old-line" v-if="ezfy.isLeader && ezfy.corpsMembers.length > 1">
              踢人:
              <select v-model="ezfy.kickUserId" class="corps-kick-sel" style="width:30%">
                <option v-for="m in ezfy.corpsMembers" v-if="!m.is_leader" :key="'kc' + m.user_id" :value="m.user_id">{{ m.name }}</option>
              </select>
              <button @click="ezfy.doKick">[踢出]</button>
            </div>
            <!-- 军团邮件群发(复刻 CorpsController.mail): ★ 军团长与副团长都能发 -->
            <div class="panel-title" v-if="ezfy.canMailCorps">军团邮件(群发全体成员)</div>
            <div class="old-line" v-if="ezfy.canMailCorps">
              <input v-model="ezfy.corpsMailContent" class="corps-mail-input" placeholder="邮件内容(500字以内)" style="width:60%"/>
              <button @click="ezfy.doCorpsMail">[群发]</button>
            </div>
          </div>
        </template>
        <!-- ★ 2026-09-30 入团审核（仅军团长）：审核开关 + 待审申请列表 -->
        <div class="panel" v-if="ezfy.isLeader">
          <div class="panel-title">入团审核</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.doToggleNeedReview()">
              [{{ ezfy.myCorpsNeedReview ? '关闭审核·无需审核直接入团' : '开启审核·需军团长审核' }}]
            </a>
          </div>
          <div class="old-line gray">开启后，未入团玩家申请需你在此通过/拒绝；未开启则直接入团。</div>
          <div class="old-line" v-if="ezfy.corpsApplies.length">
            <div class="old-line" v-for="a in ezfy.corpsApplies" :key="'ap' + a.apply_id">
              <a href="javascript:;" @click="ezfy.openPlayer(a.user_id)">{{ a.name }}</a>(id:{{ a.user_id }})
              <a class="green" href="javascript:;" @click="ezfy.doApplyHandle(a, 1)">[通过]</a>
              <a class="red" href="javascript:;" @click="ezfy.doApplyHandle(a, 2)">[拒绝]</a>
            </div>
          </div>
          <div class="old-line gray" v-else>(暂无待审申请)</div>
        </div>
        <!-- ★ 2026-09-30 未入团玩家若看的是开启审核的军团，提示需审核 -->
        <div class="old-line gray" v-if="!ezfy.myCorps && ezfy.myApplyStatus === 1">你已提交入团申请, 等待军团长审核...</div>
        <div class="panel" v-if="!ezfy.myCorps">
          <div class="panel-title">创建军团</div>
          <div class="old-line">
            军团名: <input v-model="ezfy.corpsName" class="corps-name-input" style="width:10%"/>
            <button @click="ezfy.doCreateCorps">[创建]</button>
          </div>
        </div>
        </template>

        <!-- ========== ★ 2026-09-30 军团列表（独立 tab） ========== -->
        <template v-else-if="ezfy.corpsTab === 'list'">
          <div class="panel">
            <div class="panel-title">军团列表</div>
            <table>
              <tr><th>军团</th><th>人数</th><th>战力</th><th>操作</th></tr>
              <tr v-for="cp in ezfy.corpsList" :key="'cp' + cp.id">
                <td>{{ cp.name }}</td>
                <td>{{ cp.member_count }}</td>
                <td>{{ cp.battle_score }}</td>
                <td>
                  <template v-if="!ezfy.myCorps">
                    <a href="javascript:;" @click="ezfy.doJoinCorps(cp)">{{ cp.need_review ? '[申请]' : '[加入]' }}</a>
                  </template>
                </td>
              </tr>
            </table>
            <div class="old-line" v-if="!ezfy.corpsList.length">(暂无军团)</div>
          </div>
        </template>

        <!-- ========== ★ 2026-09-30 军团聊天（独立 tab） ========== -->
        <template v-else-if="ezfy.corpsTab === 'chat'">
          <div class="panel" v-if="ezfy.myCorps">
            <div class="panel-title">军团聊天</div>
            <div class="old-line" v-for="m in ezfy.corpsChats" :key="'cc' + m.id">
              [<a href="javascript:;" @click="ezfy.openPlayer(m.user_id)">{{ m.user_name }}</a>]:{{ m.content }}
            </div>
            <div class="old-line" v-if="!ezfy.corpsChats.length">(暂无消息)</div>
            <div class="old-line">
              <input v-model="ezfy.corpsMsg" class="corps-msg-input" style="width:15%"/>
              <button @click="ezfy.doCorpsChat">发送</button>
              <a href="javascript:;" @click="ezfy.loadCorps">[刷新]</a>
            </div>
          </div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="ezfy.switchCorpsTab('list')">[去军团列表]</a></div>
          </div>
        </template>

        <!-- ========== ② 军团外交 ========== -->
        <template v-else-if="ezfy.corpsTab === 'diplomacy'">
          <div class="panel" v-if="ezfy.corpsRelations && ezfy.corpsRelations.in_corps">
            <div class="panel-title">我的军团(积分):{{ (ezfy.corpsRelations.my_corps || {}).name }}({{ (ezfy.corpsRelations.my_corps || {}).points || 0 }})</div>
            <div class="panel-title">已标记关系</div>
            <table>
              <tr><th>军团</th><th>关系</th><th v-if="ezfy.corpsRelations.can_manage">操作</th></tr>
              <tr v-for="rl in (ezfy.corpsRelations.relations || [])" :key="'rl' + rl.corps_id">
                <td>{{ rl.name }}</td>
                <td :class="rl.type === 2 ? 'red' : 'green'">{{ rl.type_name }}</td>
                <td v-if="ezfy.corpsRelations.can_manage">
                  <a href="javascript:;" @click="ezfy.setCorpsRelation(rl.corps_id, 0)">[取消标记]</a>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(ezfy.corpsRelations.relations || []).length">(暂无关系标记)</div>
            <div class="panel-title">全部军团</div>
            <table class="ezfy-corps-tbl ezfy-dip-tbl">
              <tr>
                <th>军团</th><th>团长</th><th>人数</th><th>积分</th><th>当前关系</th><th>宣战状态</th>
                <th v-if="ezfy.corpsRelations.can_manage">操作</th>
              </tr>
              <tr v-for="cp in (ezfy.corpsRelations.corps_list || [])" :key="'cr' + cp.id">
                <td>{{ cp.name }}</td>
                <td>{{ cp.leader_name || '无' }}</td>
                <td>{{ cp.member_count }}</td>
                <td>{{ cp.points }}</td>
                <td>
                  <span v-if="cp.relation_type === 1" class="green">友好</span>
                  <span v-else-if="cp.relation_type === 2" class="red">敌对</span>
                  <span v-else class="gray">无</span>
                </td>
                <td>
                  <span v-if="cp.war_status === 1 || cp.war_status === 2" class="orange">
                    {{ cp.war_status === 1 ? '宣战待生效' : '交战中' }}<template v-if="cp.war_remaining_h">{{ '（' + cp.war_remaining_h + 'h）' }}</template>
                  </span>
                  <span v-else class="gray">未宣战</span>
                </td>
                <!-- ★ 仅军团长（can_manage）可标记；已是该关系时按钮变成 [取消标记] -->
                <td v-if="ezfy.corpsRelations.can_manage">
                  <a v-if="cp.relation_type !== 1" href="javascript:;" @click="ezfy.setCorpsRelation(cp.id, 1)">[友好]</a>
                  <a v-else href="javascript:;" @click="ezfy.setCorpsRelation(cp.id, 0)">[取消标记]</a>
                  <a v-if="cp.relation_type !== 2" href="javascript:;" @click="ezfy.setCorpsRelation(cp.id, 2)">[敌对]</a>
                  <a v-else href="javascript:;" @click="ezfy.setCorpsRelation(cp.id, 0)">[取消标记]</a>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(ezfy.corpsRelations.corps_list || []).length">(暂无军团)</div>
            <div class="old-line"><a href="javascript:;" @click="ezfy.loadCorpsRelations">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!ezfy.corpsRelations"><div class="old-line">正在加载外交数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="ezfy.switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <!-- ========== ③ 军团宣战 ========== -->
        <template v-else-if="ezfy.corpsTab === 'war'">
          <div class="panel" v-if="ezfy.corpsWars && ezfy.corpsWars.in_corps">
            <div class="panel-title">军团宣战</div>
            <table class="ezfy-corps-tbl ezfy-war-tbl">
              <tr>
                <th>对方军团</th><th>我方身份</th><th>状态</th><th>宣告时间</th>
                <th>我方战绩</th><th>对方战绩</th><th>操作</th>
              </tr>
              <tr v-for="w in (ezfy.corpsWars.wars || [])" :key="'cw' + w.id">
                <td>{{ w.opp_corps_name }}</td>
                <td>{{ w.mine_is_atk ? '宣战方' : '应战方' }}</td>
                <td>
                  <span v-if="w.status === 1" class="orange">{{ w.status_name }}<template v-if="w.remaining_h">{{ '（' + w.remaining_h + 'h）' }}</template></span>
                  <span v-else-if="w.status === 2" class="red">{{ w.status_name }}<template v-if="w.remaining_h">{{ '（' + w.remaining_h + 'h）' }}</template></span>
                  <span v-else class="gray">{{ w.status_name }}</span>
                </td>
                <!-- ★ 后端下发的是毫秒时间戳，必须走 fmtTime 格式化（否则显示成一串数字） -->
                <td>{{ ezfy.fmtTime(w.declare_time) }}</td>
                <td>{{ w.my_point }}</td>
                <td>{{ w.opp_point }}</td>
                <!-- 只有进行中(status 1/2)标注进行中；已结束(status 3)显示已结束 -->
                <td>
                  <span v-if="w.status === 1 || w.status === 2" class="gray">进行中</span>
                  <span v-else class="gray">已结束</span>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(ezfy.corpsWars.wars || []).length">(暂无宣战记录)</div>
            <!-- ★ 军团长可对「未处于宣战中的军团」发起宣战：复用外交 tab 的军团列表，不重复拉接口 -->
            <template v-if="ezfy.corpsWars.can_manage && ezfy.corpsRelations && ezfy.corpsRelations.corps_list">
              <div class="panel-title">全部军团(可宣战)</div>
              <table class="ezfy-corps-tbl ezfy-war-list-tbl">
                <tr><th>军团</th><th>团长</th><th>人数</th><th>宣战状态</th><th>操作</th></tr>
                <tr v-for="cp in (ezfy.corpsRelations.corps_list || [])" :key="'wcp' + cp.id">
                  <td>{{ cp.name }}</td>
                  <td>{{ cp.leader_name || '无' }}</td>
                  <td>{{ cp.member_count }}</td>
                  <td>
                    <span v-if="cp.war_status === 1 || cp.war_status === 2" class="orange">{{ cp.war_status === 1 ? '宣战待生效' : '交战中' }}</span>
                    <span v-else class="gray">未宣战</span>
                  </td>
                  <td>
                    <a v-if="cp.war_status !== 1 && cp.war_status !== 2" href="javascript:;" @click="ezfy.declareCorpsWar(cp.id)">[宣战]</a>
                    <span v-else class="gray">进行中</span>
                  </td>
                </tr>
              </table>
            </template>
            <div class="old-line" v-if="ezfy.corpsWars.can_manage && !(ezfy.corpsRelations && ezfy.corpsRelations.corps_list)">
              <a href="javascript:;" @click="ezfy.loadCorpsRelations">[加载可宣战军团列表]</a>
            </div>
            <div class="old-line"><a href="javascript:;" @click="ezfy.loadCorpsWars">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!ezfy.corpsWars"><div class="old-line">正在加载宣战数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="ezfy.switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <!-- ========== ④ 军团商城（用个人军团积分兑换） ========== -->
        <template v-else-if="ezfy.corpsTab === 'mall'">
          <div class="panel" v-if="ezfy.corpsMall.loaded && ezfy.corpsMall.in_corps">
            <div class="panel-title">军团商城</div>
            <div class="old-line">我的军团积分 <b>{{ ezfy.corpsMall.my_points }}</b> | 军团总积分 <b>{{ ezfy.corpsMall.corps_points }}</b></div>
            <table class="ezfy-plain-table">
              <tr><th>商品</th><th>类型</th><th>内容</th><th>价格</th><th>限购/已购</th><th>库存</th><th>操作</th></tr>
              <tr v-for="it in (ezfy.corpsMall.items || [])" :key="'cmi' + it.id">
                <td>{{ it.name }}</td>
                <td>{{ it.kind_name }}</td>
                <td>
                  <template v-if="it.kind === 1">
                    <span v-if="it.food">{{ ezfy.resNames.food }}{{ it.food }}</span>
                    <span v-if="it.steel">{{ ' ' + ezfy.resNames.steel }}{{ it.steel }}</span>
                    <span v-if="it.oil">{{ ' ' + ezfy.resNames.oil }}{{ it.oil }}</span>
                    <span v-if="it.rare">{{ ' ' + ezfy.resNames.rare }}{{ it.rare }}</span>
                    <span v-if="it.gold">{{ ' ' + ezfy.resNames.gold }}{{ it.gold }}</span>
                  </template>
                  <template v-else>{{ it.item_name || it.name }} ×{{ it.item_count }}</template>
                </td>
                <td>{{ it.price }}(个人军团积分)</td>
                <td>
                  <span v-if="it.limit > 0">{{ ezfy.corpsBought(it.id) }}/{{ it.limit }}</span>
                  <span v-else class="gray">不限购</span>
                </td>
                <!-- ★ 后端下发的是「总库存 stock + 已售 sold」，这里显示剩余量（-1 = 无限） -->
                <td>
                  <span v-if="it.stock < 0" class="green">无限</span>
                  <span v-else :class="(it.stock - it.sold) > 0 ? 'gray' : 'red'">{{ (it.stock - it.sold) > 0 ? (it.stock - it.sold) : '已售罄' }}</span>
                </td>
                <td>
                  <a v-if="ezfy.corpsMallCanBuy(it)" href="javascript:;" @click="ezfy.doCorpsMallBuy(it)">[兑换]</a>
                  <span v-else class="gray">[不可兑换]</span>
                </td>
              </tr>
            </table>
            <div class="old-line gray" v-if="!(ezfy.corpsMall.items || []).length">(暂无商品)</div>
            <div class="old-line"><a href="javascript:;" @click="ezfy.loadCorpsMall">[刷新]</a></div>
          </div>
          <div class="panel" v-else-if="!ezfy.corpsMall.loaded"><div class="old-line">正在加载商城数据…</div></div>
          <div class="panel" v-else>
            <div class="old-line">你还没有加入军团 <a href="javascript:;" @click="ezfy.switchCorpsTab('info')">[去军团信息]</a></div>
          </div>
        </template>
        <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
    </template>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化拆分：军团页模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyCorps',
  inject: ['ezfy']
}
</script>
