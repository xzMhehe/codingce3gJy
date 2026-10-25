<template>
  <div>
    <!-- ============ 统帅(info) ============ -->
    <template v-if="ezfy.cur === 'info'">
      <div class="panel">
        <div class="panel-title">统帅信息</div>
        <!-- ★ 只展示「玩家号码(游戏ID)」——不展示家园号码 -->
        账号：{{ ezfy.selfInfo.game_uid || ezfy.profile.game_uid || ezfy.userBrief.game_uid || '—' }}<br/>
        昵称：{{ ezfy.selfInfo.nickname || ezfy.profile.nickname }}
        <template v-if="!ezfy.renameEditing">
          <a href="javascript:;" @click="ezfy.startRename">[修改]</a>
        </template>
        <template v-else>
          <br/>
          <input v-model="ezfy.renameInput" maxlength="12" placeholder="2~12 个字符" style="width:130px"/>
          <button @click="ezfy.doPlayerRename">确定</button>
          <a href="javascript:;" @click="ezfy.renameEditing = false">[取消]</a>
        </template>
        <br/>
        <!-- ★ 2026-10-17 统帅称号：昵称下方、阵营上方（与「他人统帅页」同款徽章） -->
        <div class="ezfy-title-line" v-if="ezfy.selfInfo.titles && ezfy.selfInfo.titles.length">
          <span v-for="t in ezfy.selfInfo.titles" :key="t.id"
                class="ezfy-title-badge" :class="'ezfy-title-t' + t.type"
                :title="t.permanent ? '永久称号' : ('有效期至 ' + t.expire_fmt)">{{ t.name }}</span>
        </div>
        <!-- ★ 2026-10-05 「没用的页面提示去掉」：改名/阵营提示不再展示（文案保留在代码里） -->
        阵营：{{ ezfy.selfInfo.camp_name || (ezfy.profile.camp === 2 ? '轴心国' : '同盟国') }}
        <!-- ★ 2026-10-17 只显示「转去另一个阵营」：已经是同盟国就不显示[转同盟国]，已经是轴心国就不显示[转轴心国]
             （当前阵营口径与 doChangeCamp 一致：selfInfo 优先、profile 兜底） -->
        <a v-if="(ezfy.selfInfo.camp || ezfy.profile.camp) !== 1" href="javascript:;" @click="ezfy.doChangeCamp(1)">[转同盟国]</a>
        <a v-if="(ezfy.selfInfo.camp || ezfy.profile.camp) !== 2" href="javascript:;" @click="ezfy.doChangeCamp(2)">[转轴心国]</a><br/>
        声望：{{ ezfy.profile.prestige }}<br/>
        军衔：{{ ezfy.rankName }}({{ ezfy.rankPost }})<span style="margin-left:4px"><span v-html="ezfy.rankIcon(ezfy.myRankId)"></span></span><br/>
        军团：{{ (ezfy.myCorps && ezfy.myCorps.name) || '无' }}<br/>
        <!-- ★ 2026-09-27 统帅信息展示军团；有军团职务(军团长/副团长/参谋长)才展示职务 -->
        <template v-if="ezfy.myCorpsTitle">职务：{{ ezfy.myCorpsTitle }}<br/></template>
        城市数：{{ ezfy.cities.length }}<br/>
        人口数：{{ ezfy.city.pop }}<br/>
        军官数：{{ ezfy.officerCount }}<br/>
        <!-- ★★ 2026-10-20 玩家反馈「不知道自己跟谁宣战/交战」：
             放在「军官数」下面，有进行中的个人宣战时才显示；
             每行只展示对方名字 + 当前城市坐标，点名字可进对方统帅页。 -->
        <template v-if="ezfy.pendingWars.length || ezfy.activeWars.length">
          <span style="color:#000">战争状态：</span><br/>
          <template v-if="ezfy.pendingWars.length">
            <span class="orange">宣战中</span>（{{ ezfy.pendingWars.length }}）<br/>
            <span v-for="w in ezfy.pendingWars" :key="'pw' + w.id">
              <a href="javascript:;" @click="ezfy.openPlayer(w.user_id)">{{ w.nickname }}</a>
              <span class="gray"> · {{ w.city_name || (w.city_id ? '城市' : '无城') }}<template v-if="w.city_id">({{ w.x }},{{ w.y }})</template></span>
              <span class="orange"> · {{ w.hours_left }}小时后开战</span>
              <span class="gray"> · {{ w.mine_is_atk ? '宣战' : '被宣战' }}</span><br/>
            </span>
          </template>
          <template v-if="ezfy.activeWars.length">
            <span class="red">对战中</span>（{{ ezfy.activeWars.length }}）<br/>
            <span v-for="w in ezfy.activeWars" :key="'aw' + w.id">
              <a href="javascript:;" @click="ezfy.openPlayer(w.user_id)">{{ w.nickname }}</a>
              <span class="gray"> · {{ w.city_name || (w.city_id ? '城市' : '无城') }}<template v-if="w.city_id">({{ w.x }},{{ w.y }})</template></span>
              <span class="red"> · {{ w.hours_left }}小时后结束</span>
              <span class="gray"> · {{ w.mine_is_atk ? '宣战' : '被宣战' }}</span><br/>
            </span>
          </template>
        </template>
        <br/>
        总兵力：{{ ezfy.totalTroops }}<br/>
        城外行进：{{ ezfy.marching }}支 | 驻守采集：{{ ezfy.occupying }}支<br/>
        占领野地：{{ ezfy.wildlands.length }}块<br/>
        <!-- ★ 2026-10-12 访问记录：我统帅页被看过的总量，点数字进访问列表 -->
        访问记录：<a href="javascript:;" @click="ezfy.go('visits')">{{ ezfy.selfInfo.visit_count || 0 }}</a><br/>
        <!-- ★ 2026-10-10 首页布局单选（老布局/新布局）：即选即生效，简约无提示 -->
        首页布局：
        <label><input type="radio" name="homeLayout" :checked="ezfy.homeLayout !== 2" @change="ezfy.setHomeLayout(1)"/>新</label>
        <label><input type="radio" name="homeLayout" :checked="ezfy.homeLayout === 2" @change="ezfy.setHomeLayout(2)"/>老</label><br/>
        <a href="javascript:;" @click="ezfy.go('friends')">[申请好友]</a>
        <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
      </div>
    </template>

    <!-- ============ 他人统帅信息(playerinfo) 复刻 PlayerController.infoOther + user/info.html ============ -->
    <!-- 游戏是沉浸式的: 点玩家名只看这一页(二战风云的数据), 不允许跳去家园个人主页 /user/:id -->
    <template v-else-if="ezfy.cur === 'playerinfo'">
      <div class="panel" v-if="ezfy.playerInfo">
        <div class="panel-title">统帅信息</div>
        <div class="old-line">
          <b :style="ezfy.nickColorAt(ezfy.playerInfo.color, 0)">{{ ezfy.playerInfo.nickname }}</b>
          <span class="gray">({{ ezfy.playerInfo.game_uid || ezfy.playerInfo.user_id }})</span>
        </div>
        <!-- ★ 2026-10-17 统帅称号：昵称下方、阵营上方（生效中的才返回，过期自动消失） -->
        <div class="ezfy-title-line" v-if="ezfy.playerInfo.titles && ezfy.playerInfo.titles.length">
          <span v-for="t in ezfy.playerInfo.titles" :key="t.id"
                class="ezfy-title-badge" :class="'ezfy-title-t' + t.type"
                :title="t.permanent ? '永久称号' : ('有效期至 ' + t.expire_fmt)">{{ t.name }}</span>
        </div>
        <div class="old-line">
          阵营：{{ ezfy.playerInfo.camp_name }}<br/>
          声望：{{ ezfy.playerInfo.prestige }}<br/>
          军衔：{{ ezfy.playerInfo.rank_name }}({{ ezfy.playerInfo.rank_post }})<span style="margin-left:4px"><span v-html="ezfy.rankIcon(ezfy.rankIdByName(ezfy.playerInfo.rank_name))"></span></span><br/>
          军团：{{ ezfy.playerInfo.corps_name || '无' }}<br/>
          <!-- ★ 2026-09-29 他人统帅页展示军团职务（与我的统帅页一致），职务在军团下一行 -->
          <template v-if="ezfy.playerInfo.corps_title">职务：{{ ezfy.playerInfo.corps_title }}<br/></template>
          城市数：{{ ezfy.playerInfo.city_count }}<br/>
          军官数：{{ ezfy.playerInfo.officer_count }}<br/>
          占领野地：{{ ezfy.playerInfo.wild_count }}块
        </div>
        <div class="old-line">
          <template v-if="ezfy.playerInfo.is_self">
            <span class="gray">这是你自己</span>
            <a href="javascript:;" @click="ezfy.go('info')">[我的统帅页]</a>
          </template>
          <template v-else-if="ezfy.playerInfo.is_friend">
            <span class="green">已是好友</span>
            <a href="javascript:;" @click="ezfy.openPm(ezfy.playerInfo.user_id)">[发私信]</a>
          </template>
          <template v-else-if="ezfy.playerInfo.is_applied">
            <span class="orange">好友申请已发送, 等待对方处理</span>
          </template>
          <template v-else>
            <a href="javascript:;" @click="ezfy.doAddFriendById()">[申请好友]</a>
            <a href="javascript:;" @click="ezfy.openPm(ezfy.playerInfo.user_id)">[发私信]</a>
          </template>
        </div>
        <a href="javascript:;" @click="ezfy.go(ezfy.playerInfoBack)">[返回]</a>
        <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
      </div>
      <div class="panel" v-else>
        <div class="old-line gray">正在加载统帅信息…</div>
      </div>
    </template>

    <!-- ============ 访问记录(visits)：我统帅页被谁看过 ============ -->
    <template v-else-if="ezfy.cur === 'visits'">
      <div class="panel">
        <div class="panel-title">访问记录</div>
        <template v-if="ezfy.visitRecords.length">
          <div class="old-line" v-for="v in ezfy.visitRecords" :key="v.visitor_uid + v.created_at">
            <a href="javascript:;" @click="ezfy.openPlayer(v.visitor_uid)">{{ v.nickname }}</a>
            <span class="gray">({{ v.visitor_uid }})</span>
            <span class="gray"> · {{ ezfy.fmtTime(v.created_at) }}</span>
          </div>
        </template>
        <template v-else>
          <div class="old-line gray">还没有人看过你的统帅页</div>
        </template>
        <div class="old-line">
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
      </div>
    </template>

    <!-- ============ 军官/学院(acade) ============ -->
    <template v-else-if="ezfy.cur === 'acade'">
      <div class="panel-title">
        参谋部({{ ezfy.officerData.staff_level }}级)
        <!-- ★ 2026-10-10 「去招募 | 战俘营」从大标题里拿出来、字号缩到正文大小 -->
        <span class="staff-links">
          <a href="javascript:;" @click="ezfy.switchAcade('search')">去招募</a> |
          <a href="javascript:;" @click="ezfy.switchAcade('captive')">战俘营</a>
        </span>
      </div>
      <div class="acade-tab">
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'officer' }" @click="ezfy.switchAcade('officer')">军官</a>|
        <!-- ★ 2026-10-19 计谋 tab 已移到「军官详情 → 计谋」（属性/技能/装备 之间），
             军校页不再单列；行军计谋仍在 军情→军队动态 里对部队使用。 -->
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'search' }" @click="ezfy.switchAcade('search')">招募</a>|
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'mayor' }" @click="ezfy.switchAcade('mayor')">任命市长</a>|
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'equip' }" @click="ezfy.switchAcade('equip')">装备</a>|
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'skill' }" @click="ezfy.switchAcade('skill')">技能</a>|
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'generals' }" @click="ezfy.switchAcade('generals')">名将图鉴</a>
      </div>

      <!-- 军官列表 -->
      <!-- 军官列表: 复刻 acade/acadeIndex.html -->
      <div class="panel" v-if="ezfy.acadeTab === 'officer'">
        <div class="old-line">
          军校{{ ezfy.officerData.academy_level }}级, 参谋部{{ ezfy.officerData.staff_level }}级
          (容纳{{ ezfy.officerData.capacity }}名军官), 当前{{ ezfy.officerData.used }}名
        </div>
        <div class="old-line">
          {{ ezfy.resNames.gold }}:{{ ezfy.fmtN(ezfy.officerData.gold) }}
          <!-- ★ 2026-10-06 军官工资提示移除：工资已并入资源页黄金「耗量(每小时)」统一展示 -->
        </div>
        <hr/>
        <template v-for="o in ezfy.myOfficers">
          <div class="old-line" :key="'of' + o.id">
            {{ o.name }}
            <!-- ★ 2026-10-04 名将标识（金色徽章）：原名只在军官详情的「名将背景」里展示，列表不再外露 -->
            <span v-if="o.is_general"
                  style="display:inline-block;background:linear-gradient(180deg,#ffd700,#e6a800);color:#5b3a00;font-size:12px;line-height:14px;padding:0 5px;border-radius:3px;font-weight:bold;margin-left:4px;vertical-align:1px;">名将</span>
            ({{ o.level }}级)
            <a href="javascript:;" @click="ezfy.openOfficer(o.id)">查看</a><br/>
            状态:{{ o.status_name }}<span v-if="o.position_name !== '无'" class="blue">（{{ o.position_name }}）</span> &nbsp; 星级:{{ o.star }}☆<br/>
            后勤/军事/学识/忠诚：<br/>
            <!-- ★ 2026-10-25 列表简约化：不再展示「(装备 军事+5 后勤+5 学识+5)」，装备加成进军官详情看 -->
            {{ o.logistics_total }}/{{ o.military_total }}/{{ o.learning_total }}/{{ o.loyalty }}<br/>
            攻/防：{{ o.attack }}/{{ o.defence }}<br/>
            <!-- ★ 可用属性点：升过级还没点的军官一眼能看见 -->
            <span v-if="o.free_points > 0" class="red">
              可分配属性点 {{ o.free_points }} 点
              <a href="javascript:;" @click="ezfy.openOfficer(o.id)">[去加点]</a><br/>
            </span>
            <span v-if="o.active_sets && o.active_sets.length" class="green">
              套装：{{ o.active_sets.join('、') }}<br/>
            </span>
            ------------------------
          </div>
        </template>
        <div class="old-line gray" v-if="!ezfy.myOfficers.length">(暂无军官, 先去招募吧)</div>
        <div class="old-line">
          前去<a href="javascript:;" @click="ezfy.switchAcade('captive')">战俘营</a>
          <span class="gray" v-if="ezfy.captiveOfficers.length">({{ ezfy.captiveOfficers.length }}名俘虏待收编)</span>
        </div>
      </div>

      <!-- 招募 -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'search'">
        <div class="old-line">
          军校({{ ezfy.recruitData.academy_level }}级)：
          <span v-if="ezfy.recruitData.refresh_left !== undefined">
            本小时刷新:{{ ezfy.recruitData.refresh_left }}/{{ ezfy.recruitData.refresh_limit }}次
          </span>
          <a href="javascript:;" @click="ezfy.doRefreshRecruit">[刷新]</a>
          <!-- ★ 次数用完后，直接在军校使用招生简章（不用先去背包用） -->
          <a href="javascript:;" @click="ezfy.doUseRecruitTicket">[招生简章刷新]</a>
          <span class="gray">({{ ezfy.recruitTicketCount() }} 张)</span>
        </div>
        <div class="old-line">
          参谋部{{ ezfy.recruitData.staff_level }}级({{ ezfy.recruitData.used }}/{{ ezfy.recruitData.capacity }})
        </div>
        <div class="old-line" v-if="!ezfy.recruitData.academy_level">尚未建造军校, 无法招募军官</div>
        <table v-else class="ezfy-plain-table">
          <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
          <tr v-for="g in ezfy.recruitData.candidates" :key="'rc' + g.key">
            <td>{{ g.name }}</td>
            <td>{{ g.level }}级</td>
            <td>{{ g.star }}☆</td>
            <td>{{ g.logistics }}/{{ g.military }}/{{ g.learning }}</td>
            <td>{{ g.cost }}</td>
            <td>
              <a v-if="!ezfy.officerFull" href="javascript:;" @click="ezfy.doRecruit(g)">雇佣</a>
              <span v-else class="gray">(容量已满)</span>
            </td>
          </tr>
        </table>
        <div class="old-line gray" v-if="ezfy.recruitData.academy_level && !ezfy.recruitData.candidates.length">(本小时候选已全部招募或刷新)</div>
        <div class="old-line">前去<a href="javascript:;" @click="ezfy.switchAcade('officer')">[军官]</a></div>
      </div>

      <!-- 任命市长: 复刻 acade/setMayor.html -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'mayor'">
        <table class="ezfy-plain-table">
          <!-- ★ 2026-10-05 去掉等级/忠诚列，新增 后/军/学 属性列 -->
          <tr><th>名称</th><th>后/军/学</th><th>当前职位</th><th>操作</th></tr>
          <tr v-for="o in ezfy.myOfficers" :key="'my' + o.id">
            <td>{{ o.name }}</td>
            <td>{{ o.logistics }}/{{ o.military }}/{{ o.learning }}</td>
            <td>{{ o.position_name }}</td>
            <td>
              <a v-if="o.position !== 1 && o.status === 0" href="javascript:;" @click="ezfy.doPosition(o, 1)">[任命市长]</a>
              <a v-if="o.position !== 2 && o.status === 0" href="javascript:;" @click="ezfy.doPosition(o, 2)">[任命城守]</a>
              <a v-if="o.position !== 0" href="javascript:;" @click="ezfy.doPosition(o, 0)">[卸任]</a>
            </td>
          </tr>
        </table>
        <div class="old-line gray" v-if="!ezfy.officerData.officers.length">(暂无军官)</div>
      </div>

      <!-- 装备 -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'equip'">
        <div class="old-line">
          我的装备({{ ezfy.equipData.bag.length }})
          <a href="javascript:;" @click="ezfy.switchMallTab('equipment'); ezfy.go('mall')">[去商城买散件]</a>
          <a href="javascript:;" @click="ezfy.switchMallTab('chest'); ezfy.go('mall')">[去开宝箱]</a>
        </div>

        <!-- 装备页子 tab：我的装备 / 我的套装 / 装备图鉴 -->
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: ezfy.equipTab === 'my' }" @click="ezfy.equipTab = 'my'">装备</a>|
          <a href="javascript:;" :class="{ on: ezfy.equipTab === 'set' }" @click="ezfy.equipTab = 'set'">套装</a>|
          <a href="javascript:;" :class="{ on: ezfy.equipTab === 'all' }" @click="ezfy.equipTab = 'all'">装备图鉴</a>
        </div>

        <!-- 我的装备（背包散件 + 检索 + 分页） -->
        <div v-if="ezfy.equipTab === 'my'">
        <div class="old-line">
          搜索:
          <input v-model="ezfy.equipWord" type="text" placeholder="装备名 / 部位 / 套装"
                 style="width:180px" @input="ezfy.equipPage = 1"/>
          <a href="javascript:;" @click="ezfy.equipWord = ''; ezfy.equipPage = 1">[清空]</a>
          <span class="gray">共 {{ ezfy.equipGroups.length }} 种</span>
        </div>
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:28%"><col style="width:13%"><col style="width:18%"><col style="width:12%"><col style="width:11%"><col style="width:18%">
          </colgroup>
          <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>要求等级</th><th>状态</th></tr>
          <!-- ★ 2026-09-25 用户建议：「装备[查看]按钮去了也行，同时放到套装里面展开展示也可以」
               → 采纳：**砍掉「属性」列（原来只放一个 [查看] 按钮）**，改成在该行下面展开详情卡。
               好处：少一列 → 手机上不再挤；少一次跳页 → 不用来回返回。
               （原来那个独立的「装备详情页」已经没人能进，一并删掉了。）
               ★ 用户进一步要求「点名称看该装备的加成，点套装看套装的加成」
               → 两个入口看**不同**内容，用 detailMode 区分。 -->
          <template v-for="e in ezfy.equipPaged">
          <tr :key="'eq' + e.key">
            <td class="nm"><a href="javascript:;" @click="ezfy.toggleDetail(e.id, 'item')">{{ e.name }}</a><span class="gray"> ×{{ e.count }}</span></td>
            <td>{{ e.slot || e.type }}</td>
            <td>
              <a v-if="e.set_id" href="javascript:;" @click="ezfy.toggleDetail(e.id, 'set')">{{ e.set_name }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td :class="ezfy.qualityClass(e.tier_name)">{{ e.tier_name }}</td>
            <td>{{ e.level }}</td>
            <td>
              <span v-if="e.worn > 0" class="gray">已穿戴{{ e.worn }}{{ e.worn >= e.count ? ' · 全部' : '' }}</span>
              <a v-if="e.count - e.worn > 0" href="javascript:;" @click="ezfy.switchAcade('officer')">[去穿戴{{ e.count - e.worn }}]</a>
            </td>
          </tr>
          <tr v-if="ezfy.detailRowId === e.id" :key="'dt' + e.id" class="set-card-row">
            <td :colspan="6">
              <div class="set-card">
                <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                <template v-if="ezfy.detailMode === 'item'">
                  <div class="sc-h"><b>{{ e.name }}</b>
                    <span :class="ezfy.qualityClass(e.tier_name)">[{{ e.tier_name || '普通' }}]</span>
                    <span class="gray">{{ e.slot || e.type }} · {{ e.level }}级</span>
                  </div>
                  <div class="sc-b">装备加成：<b class="green">{{ ezfy.equipAttrText(e) || '（这件没有额外属性加成）' }}</b></div>
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id)">所属套装：{{ ezfy.setOf(e.set_id).name }}</div>
                  <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                </template>
                <!-- ② 点「套装名」→ 只看套装加成 -->
                <template v-else-if="ezfy.setOf(e.set_id)">
                  <div class="sc-h"><b>{{ ezfy.setOf(e.set_id).name }}</b>
                    <span :class="ezfy.qualityClass(ezfy.setOf(e.set_id).tier_name)">[{{ ezfy.setOf(e.set_id).tier_name || '特殊' }}]</span>
                    <span class="gray">穿齐 {{ ezfy.setOf(e.set_id).parts }} 件才生效</span>
                  </div>
                  <div class="sc-b">套装加成：<b class="green">{{ ezfy.setBonusText(e.set_id) || '（本套装无额外属性加成）' }}</b></div>

                  <div class="sc-b">我的进度：已拥有 <b>{{ ezfy.setOf(e.set_id).owned || 0 }}</b>/{{ ezfy.setOf(e.set_id).parts }} 件
                    <span v-if="(ezfy.setOf(e.set_id).owned || 0) >= ezfy.setOf(e.set_id).parts" class="green">已够穿齐</span>
                    <span v-else class="red">还差 {{ ezfy.setOf(e.set_id).parts - (ezfy.setOf(e.set_id).owned || 0) }} 件</span>
                  </div>
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id).slots && ezfy.setOf(e.set_id).slots.length">部位：{{ ezfy.setOf(e.set_id).slots.join(' / ') }}</div>
                  <div class="sc-b gray">点装备名看这件自己的加成</div>
                </template>
                <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
              </div>
            </td>
          </tr>
          </template>
        </table>
        <div class="old-line gray" v-if="!ezfy.equipData.bag.length">(背包暂无装备)</div>
        <div class="old-line gray" v-else-if="!ezfy.equipFiltered.length">(没有匹配「{{ ezfy.equipWord }}」的装备)</div>
        <!-- ★ 分页 -->
        <div class="ezfy-pager" v-if="ezfy.equipGroups.length > ezfy.equipPageSize">
          <a href="javascript:;" :class="{ disabled: ezfy.equipPage <= 1 }" @click="ezfy.equipGo(-1)">[上一页]</a>
          <span class="gray">{{ Math.min(ezfy.equipPage, ezfy.equipTotalPages) }}/{{ ezfy.equipTotalPages }}</span>
          <a href="javascript:;" :class="{ disabled: ezfy.equipPage >= ezfy.equipTotalPages }" @click="ezfy.equipGo(1)">[下一页]</a>
        </div>
        </div>

        <!-- 套装一览：默认只列**我拥有的**（原来这里铺的是「全部套装」= 图鉴，玩家分不清哪个是自己有的），
             可以切到「全部套装」横向对比 —— ★ 2026-09-25 「方便玩家知晓、对比套装」。 -->
        <div v-if="ezfy.equipTab === 'set'">
        <div class="old-line set-tab">
          <a href="javascript:;" :class="{ on: !ezfy.setShowAll }" @click="ezfy.setShowAll = false">[只看我有的]</a>
          <a href="javascript:;" :class="{ on: ezfy.setShowAll }" @click="ezfy.setShowAll = true">[全部套装·可对比]</a>
          <span class="gray">共 {{ ezfy.setListShown.length }} 套</span>
        </div>
        <!-- ★ 2026-09-25：加成**默认直接铺出来**（原来要点 [加成] 才看得到，用户反馈「不容易看到」）。
             一套一块、竖排 —— 手机上不用横向找，也方便上下对比。 -->
        <div class="set-block" v-for="s in ezfy.setListShown" :key="'ms' + s.id">
          <div class="sb-h">
            <b :class="ezfy.qualityClass(s.tier_name)">{{ s.name }}</b>
            <span class="gray">[{{ s.tier_name || '特殊' }}]</span>
            <b :class="s.active ? 'green' : 'red'">{{ s.have }}/{{ s.parts }}</b> 件
            <span v-if="s.active" class="green">加成已生效</span>
            <span v-else class="red">还差 {{ s.need }} 件才生效</span>
          </div>
          <div class="sb-b">套装加成：<b class="green">{{ ezfy.equipAttrText(s) || '（本套装无额外属性加成）' }}</b></div>
          <div class="sb-b gray" v-if="s.effect">额外效果：{{ s.effect }}</div>
          <div class="sb-b gray" v-if="s.slots && s.slots.length">部位：{{ s.slots.join(' / ') }}</div>
        </div>
        <div class="old-line gray" v-if="!ezfy.setListShown.length">(暂无套装装备)</div>
        </div>

        <!-- 装备图鉴（全部装备 + 检索 + 分页） -->
        <div v-if="ezfy.equipTab === 'all'">
        <div class="old-line">装备图鉴({{ ezfy.equipData.all.length }})</div>
        <div class="old-line">
          搜索:
          <input v-model="ezfy.equipAllWord" type="text" placeholder="装备名 / 部位 / 套装"
                 style="width:180px" @input="ezfy.equipAllPage = 1"/>
          <a href="javascript:;" @click="ezfy.equipAllWord = ''; ezfy.equipAllPage = 1">[清空]</a>
          <span class="gray">共 {{ ezfy.equipAllFiltered.length }} 件</span>
        </div>
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:34%"><col style="width:14%"><col style="width:20%"><col style="width:12%"><col style="width:20%">
          </colgroup>
          <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>需求等级</th></tr>
          <template v-for="e in ezfy.equipAllPaged">
          <tr :key="'ea' + e.id">
            <td class="nm"><a href="javascript:;" @click="ezfy.toggleDetail(e.id, 'item')">{{ e.name }}</a></td>
            <td>{{ e.slot || e.type }}</td>
            <td>
              <a v-if="e.set_id" href="javascript:;" @click="ezfy.toggleDetail(e.id, 'set')">{{ e.set_name }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td :class="ezfy.qualityClass(e.tier_name)">{{ e.tier_name }}</td>
            <td>{{ e.level }}</td>
          </tr>
          <tr v-if="ezfy.detailRowId === e.id" :key="'dt' + e.id" class="set-card-row">
            <td :colspan="5">
              <div class="set-card">
                <template v-if="ezfy.detailMode === 'item'">
                  <div class="sc-h"><b>{{ e.name }}</b>
                    <span :class="ezfy.qualityClass(e.tier_name)">[{{ e.tier_name || '普通' }}]</span>
                    <span class="gray">{{ e.slot || e.type }} · {{ e.level }}级</span>
                  </div>
                  <div class="sc-b">装备加成：<b class="green">{{ ezfy.equipAttrText(e) || '（这件没有额外属性加成）' }}</b></div>
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id)">所属套装：{{ ezfy.setOf(e.set_id).name }}</div>
                  <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                </template>
                <template v-else-if="ezfy.setOf(e.set_id)">
                  <div class="sc-h"><b>{{ ezfy.setOf(e.set_id).name }}</b>
                    <span :class="ezfy.qualityClass(ezfy.setOf(e.set_id).tier_name)">[{{ ezfy.setOf(e.set_id).tier_name || '特殊' }}]</span>
                    <span class="gray">穿齐 {{ ezfy.setOf(e.set_id).parts }} 件才生效</span>
                  </div>
                  <div class="sc-b">套装加成：<b class="green">{{ ezfy.setBonusText(e.set_id) || '（本套装无额外属性加成）' }}</b></div>

                  <div class="sc-b">我的进度：已拥有 <b>{{ ezfy.setOf(e.set_id).owned || 0 }}</b>/{{ ezfy.setOf(e.set_id).parts }} 件
                    <span v-if="(ezfy.setOf(e.set_id).owned || 0) >= ezfy.setOf(e.set_id).parts" class="green">已够穿齐</span>
                    <span v-else class="red">还差 {{ ezfy.setOf(e.set_id).parts - (ezfy.setOf(e.set_id).owned || 0) }} 件</span>
                  </div>
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id).slots && ezfy.setOf(e.set_id).slots.length">部位：{{ ezfy.setOf(e.set_id).slots.join(' / ') }}</div>
                  <div class="sc-b gray">点装备名看这件自己的加成</div>
                </template>
                <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
              </div>
            </td>
          </tr>
          </template>
        </table>
        <div class="old-line gray" v-if="!ezfy.equipAllFiltered.length">(没有匹配「{{ ezfy.equipAllWord }}」的装备)</div>
        <!-- ★ 分页 -->
        <div class="ezfy-pager" v-if="ezfy.equipAllFiltered.length > ezfy.equipAllPageSize">
          <a href="javascript:;" :class="{ disabled: ezfy.equipAllPage <= 1 }" @click="ezfy.equipAllGo(-1)">[上一页]</a>
          <span class="gray">{{ Math.min(ezfy.equipAllPage, ezfy.equipAllTotalPages) }}/{{ ezfy.equipAllTotalPages }}</span>
          <a href="javascript:;" :class="{ disabled: ezfy.equipAllPage >= ezfy.equipAllTotalPages }" @click="ezfy.equipAllGo(1)">[下一页]</a>
        </div>
        </div>
      </div>

      <!-- 技能: 复刻 acade/skill.html 的编号列表(带完整说明) -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'skill'">
        <div class="old-line">军官技能:</div>
        <div class="old-line" v-for="(sk, i) in ezfy.skillData.skills" :key="'sk' + sk.id">
          {{ i + 1 }}、{{ sk.name }}:{{ sk.des || sk.effect }}<br/>
          <span class="gray">效果：{{ sk.effect }}</span>
          <br/>--------------------
        </div>
        <!-- ★ 2026-10-06 「我的军官」表格去掉：与军官详情里的已学技能重复展示，
             学习/遗忘统一走 军官列表 → 详情 → 属性 tab 的技能1~3（★ 2026-10-25 起学技能=随机） -->
      </div>

      <!-- ★ 2026-10-19 军校页的「计谋」面板已整体搬到「军官详情 → 计谋」tab
           （用户要求：放到 属性 和 装备 之间，为后续「军官属性加成」做准备）。
           这里不再渲染，避免同一份计谋列表在军校页和军官详情两处重复出现。 -->

      <!-- 战俘营(复刻原版 acade/conquer.html) -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'captive'">
        <div class="old-line">
          参谋部({{ ezfy.officerData.staff_level }}级)
          <a href="javascript:;" @click="ezfy.switchAcade('search')">去招募</a> |
          战俘营({{ ezfy.officerData.captive_used || 0 }}/{{ ezfy.officerData.captive_capacity || 0 }})
          <a v-if="ezfy.captiveOfficers.length" href="javascript:;" @click="ezfy.releaseAllCaptives">[一键释放]</a>
        </div>
        <table class="ezfy-plain-table">
          <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
          <tr v-for="o in ezfy.captiveOfficers" :key="'cp' + o.id">
            <td>{{ o.name }}</td>
            <td>{{ o.level }}级</td>
            <td>{{ o.star }}☆</td>
            <td>{{ o.logistics }}/{{ o.military }}/{{ o.learning }}</td>
            <td>免费</td>
            <td>
              <a href="javascript:;" @click="ezfy.doCaptive(o, 'recruit')">[雇佣]</a>
              <a href="javascript:;" @click="ezfy.doCaptive(o, 'free')">[释放]</a>
            </td>
          </tr>
        </table>
        <div class="old-line gray" v-if="!ezfy.captiveOfficers.length">(本城战俘营暂无俘虏)</div>
        <div class="old-line">前去<a href="javascript:;" @click="ezfy.switchAcade('officer')">[军官]</a></div>
      </div>

      <!-- 名将图鉴（★ 2026-10-15 分页展示，默认每页 10，翻页静默无提示） -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'generals'">
        <div class="old-line">名将图鉴(共{{ ezfy.generalData.generals.length }}名)</div>
        <table class="ezfy-plain-table">
          <!-- ★ 2026-10-05 去掉等级/星级列；★ 2026-10-23 加「组合技」列（系统专属战法） -->
          <tr><th>名称</th><th>军/后/学</th><th>组合技</th><th>状态</th></tr>
          <tr v-for="g in ezfy.generalPaged" :key="'gg' + g.id">
            <td>{{ g.name }}</td>
            <td>{{ g.military }}/{{ g.logistics }}/{{ g.learning }}</td>
            <td>
              <!-- ★ 2026-10-24 多技能：组合技是数组，逐个列出 -->
              <span v-if="g.combo && g.combo.length" class="green">{{ g.combo.map(c => c.name).join('、') }}</span>
              <span v-else class="gray">—</span>
            </td>
            <td>
              <span v-if="g.owned" class="green">已拥有</span>
              <span v-else class="gray">未拥有</span>
            </td>
          </tr>
        </table>
        <div class="ezfy-pager" v-if="ezfy.generalData.generals.length > ezfy.generalPageSize">
          <a href="javascript:;" :class="{ disabled: ezfy.generalPage <= 1 }" @click="ezfy.generalGo(-1)">[上一页]</a>
          <span class="gray">{{ Math.min(ezfy.generalPage, ezfy.generalTotalPages) }}/{{ ezfy.generalTotalPages }}</span>
          <a href="javascript:;" :class="{ disabled: ezfy.generalPage >= ezfy.generalTotalPages }" @click="ezfy.generalGo(1)">[下一页]</a>
        </div>
      </div>
    </template>

    <!-- ============ 军官详情(officerdetail) ============ -->
    <template v-else-if="ezfy.cur === 'officerdetail'">
      <!-- ★ 军官详情：按 tab 分「属性 / 技能 / 计谋 / 装备」四块（同 acade 页 .acade-tab 写法） -->
      <div class="panel" v-if="ezfy.officerDetail.officer">
        <div class="panel-title">
          {{ ezfy.officerDetail.officer.name }}
          <!-- ★ 2026-10-04 名将标识：详情页头部徽标 + 改名后显示原名 -->
          <span v-if="ezfy.officerDetail.officer.is_general" class="orange">【名将】</span>
          <span v-if="ezfy.officerDetail.officer.is_general && ezfy.officerDetail.officer.general_name &&
                       ezfy.officerDetail.officer.general_name !== ezfy.officerDetail.officer.name" class="gray">
            （{{ ezfy.officerDetail.officer.general_name }}）
          </span>
          <!-- ★ 2026-10-23 在外采集/出征的军官不能改名（命令表按名字关联军官，改名会导致归队异常） -->
          <a v-if="ezfy.officerDetail.officer.status !== 1" href="javascript:;" @click="ezfy.doOfficerRename">[改名]</a>
          <span v-else class="gray">（归队改名）</span>
        </div>

        <!-- ★ 2026-10-25 「技能」tab 已去掉（用户要求）：学技能/遗忘都挪到「属性」页的
             技能1~3 三行里（学 = 消耗1本技能书随机学一个），所以这里不再有独立页签。 -->
        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'attr' }" @click="ezfy.officerDetailTab = 'attr'">属性</a>|
          <!-- ★ 2026-10-19 计谋 tab 从军校页移到这里（属性 ↔ 装备 之间）：
               同一个列表、同一套发动逻辑，只是入口改到军官详情，为后续「军官属性加成」做准备。 -->
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'scheme' }" @click="ezfy.openOfficerDetailTab('scheme')">计谋</a>|
          <!-- ★ 2026-10-10 装备背包 tab 已合并进「装备」：一个 tab 看已穿 + 一键穿戴 + 背包 -->
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'equip' }" @click="ezfy.openOfficerDetailTab('equip')">装备</a>
          <!-- ★ 2026-10-04 名将背景 tab 放到最后（仅名将显示）：二战的功勋介绍 -->
          <a v-if="ezfy.officerDetail.officer.is_general" href="javascript:;"
             :class="{ on: ezfy.officerDetailTab === 'general' }" @click="ezfy.officerDetailTab = 'general'"><span style="margin:0 6px 0 2px;color:#888;">|</span>名将背景</a>
        </div>

        <!-- 名将背景 tab -->
        <div v-if="ezfy.officerDetailTab === 'general' && ezfy.officerDetail.officer.is_general">
          <div class="old-line">
            名将：<b>{{ ezfy.officerDetail.officer.general_name || ezfy.officerDetail.officer.name }}</b>
            <span class="gray" v-if="ezfy.officerDetail.officer.general_name && ezfy.officerDetail.officer.general_name !== ezfy.officerDetail.officer.name">
            </span>
          </div>
          <div class="old-line" v-if="ezfy.officerDetail.officer.general_star || ezfy.officerDetail.officer.general_level">
            池中星级：{{ ezfy.officerDetail.officer.general_star || '—' }}星 &nbsp; 池中等级：{{ ezfy.officerDetail.officer.general_level || '—' }}级
          </div>
          <div class="old-line" v-if="ezfy.officerDetail.officer.general_des">
            二战功勋：{{ ezfy.officerDetail.officer.general_des }}
          </div>
          <!-- ★ 2026-10-23 名将组合技能（系统专属战法，管理端可改名/调数值）
               ★ 2026-10-24 多技能：combo_skills 是数组，一个名将可学多个 -->
          <div class="old-line" v-for="cs in ezfy.officerDetail.officer.combo_skills || []" :key="'oc' + cs.id">
            组合技：<b class="green">{{ cs.name }}</b>
            <span class="gray">Lv.{{ cs.level }}
              · 发动率 {{ cs.proc_pct }}%
              · {{ cs.value_text }}</span>
          </div>
          <div class="old-line gray" v-else>(该名将暂无功勋介绍)</div>
        </div>

        <!-- 属性 tab（★ 2026-10-24 怀旧直排布局：基础信息区/核心属性区/战斗属性区/养成属性区/技能装备区/特殊技能区） -->
        <div v-if="ezfy.officerDetailTab === 'attr'">
        <!-- ===== 基础信息区 ===== -->
        <div class="old-line">
          名称：
          <input v-model="ezfy.officerRenameInput" type="text" maxlength="12" placeholder="2~12 个字符" style="width:120px"/>
          <a href="javascript:;" @click="ezfy.doOfficerRename">[改名]</a>
        </div>
        <div class="old-line">归属统帅：<b>{{ ezfy.profile.nickname || '—' }}</b></div>
        <div class="old-line">
          等级：<b>{{ ezfy.officerDetail.officer.level }}</b>
          <a v-if="ezfy.officerDetail.officer.status !== 1 && ezfy.officerDetail.officer.position === 0"
             href="javascript:;" @click="ezfy.doExile">[解雇]</a>
        </div>
        <!-- ★ 2026-10-25 状态行加「任命城守」，并**显示当前职位**：
             以前这里只写「在职」，任命成功后职位没地方显示 → 玩家以为「任命了不生效」
             （后端其实已经写入 position，只是这页不展示）。 -->
        <div class="old-line">
          状态：<b :class="ezfy.officerDetail.officer.status_name === '出征中' ? 'red' : ''">{{ ezfy.officerDetail.officer.status_name }}</b>
          <span v-if="ezfy.officerDetail.officer.position" class="blue">（{{ ezfy.officerDetail.officer.position_name }}）</span>
          <template v-if="ezfy.officerDetail.officer.status !== 1 && ezfy.officerDetail.officer.is_captive !== 1">
            <a v-if="ezfy.officerDetail.officer.position !== 1" href="javascript:;" @click="ezfy.doAppoint(1)">[任命市长]</a>
            <a v-if="ezfy.officerDetail.officer.position !== 2" href="javascript:;" @click="ezfy.doAppoint(2)">[任命城守]</a>
            <a v-if="ezfy.officerDetail.officer.position" href="javascript:;" @click="ezfy.doAppoint(0)">[卸任]</a>
          </template>
        </div>
        <div class="old-line">
          星级：{{ ezfy.officerDetail.officer.star }}☆
          <span class="gray" v-if="ezfy.officerDetail.officer.star_max"></span>
          <a v-if="ezfy.officerDetail.officer.star_up_on && ezfy.officerDetail.officer.star < ezfy.officerDetail.officer.star_max"
             href="javascript:;" @click="ezfy.doStarUp">[升星]</a>
        </div>
        <div class="old-line">
          经验：<span class="gray">{{ ezfy.officerDetail.officer.level >= ezfy.detailOfficerMaxLevel ? '—' : (ezfy.officerDetail.officer.exp + '/' + ezfy.officerDetail.officer.exp_need) }}</span>
        </div>
        <div class="old-line" v-if="ezfy.officerDetail.officer.salary > 0">薪水：{{ ezfy.officerDetail.officer.salary }}</div>

        <!-- ===== 核心属性区 ===== -->
        <!-- ★ 2026-10-25 加点按钮从下面「分配：后勤[+1][+10][全加] …」那一行**挪到三项各自后面**
             （用户要求）；「属性点」也从养成属性区提到这里，紧跟着它控制的三项，洗点留在同一行。
             只有有可用属性点时才显示 [+1][+10][全]（与原来「分配」行的显隐条件一致）。
             ★ 后面的 `(+N)` = **比军官池原始属性多出来的全部加成**（加点 + 装备 + 套装），
               用 `officerAttrPlus`（有效值 − base_*）现算 —— 玩家加完属性点能立刻看到数字变化，
               穿了装备也一并算进同一个括号里。
             ★★ 2026-10-25 用户要求「属性点 <10 就别展示 [+10]，简约」：
               [+10] 只在 free_points >= 10 时出现（不够 10 点点了也没意义）。 -->
        <div class="old-line">
          军事：{{ ezfy.officerDetail.officer.military_total }}<span class="green" v-if="ezfy.officerAttrPlus.mil">(+{{ ezfy.officerAttrPlus.mil }})</span><template v-if="ezfy.officerDetail.officer.free_points > 0"><a href="javascript:;" @click="ezfy.doAddAttr('military', 1)">[+1]</a><a v-if="ezfy.officerDetail.officer.free_points >= 10" href="javascript:;" @click="ezfy.doAddAttr('military', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('military')">[全]</a></template>
        </div>
        <div class="old-line">
          学识：{{ ezfy.officerDetail.officer.learning_total }}<span class="green" v-if="ezfy.officerAttrPlus.lea">(+{{ ezfy.officerAttrPlus.lea }})</span><template v-if="ezfy.officerDetail.officer.free_points > 0"><a href="javascript:;" @click="ezfy.doAddAttr('learning', 1)">[+1]</a><a v-if="ezfy.officerDetail.officer.free_points >= 10" href="javascript:;" @click="ezfy.doAddAttr('learning', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('learning')">[全]</a></template>
        </div>
        <div class="old-line">
          后勤：{{ ezfy.officerDetail.officer.logistics_total }}<span class="green" v-if="ezfy.officerAttrPlus.log">(+{{ ezfy.officerAttrPlus.log }})</span><template v-if="ezfy.officerDetail.officer.free_points > 0"><a href="javascript:;" @click="ezfy.doAddAttr('logistics', 1)">[+1]</a><a v-if="ezfy.officerDetail.officer.free_points >= 10" href="javascript:;" @click="ezfy.doAddAttr('logistics', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('logistics')">[全]</a></template>
        </div>
        <div class="old-line">
          属性点：<b :class="ezfy.officerDetail.officer.free_points > 0 ? 'red' : 'gray'">{{ ezfy.officerDetail.officer.free_points }}</b><a href="javascript:;" @click="ezfy.doRespec">[洗点]</a>
        </div>

        <!-- ===== 战斗属性区 ===== -->
        <!-- ★ 2026-10-25 用户要求合成一行：攻击/防御：505/280 -->
        <div class="old-line">
          攻击/防御：{{ ezfy.officerDetail.officer.attack }}/{{ ezfy.officerDetail.officer.defence }}
        </div>
        <!-- 套装与战斗加成（套装穿齐才生效；战斗加成只列**单件**的，套装加成单独一行，不重复） -->
        <template v-if="(ezfy.officerDetail.officer.set_progress && ezfy.officerDetail.officer.set_progress.length) ||
                         ezfy.officerEquipText(ezfy.officerDetail.officer)">
          <div class="old-line" v-if="ezfy.officerEquipText(ezfy.officerDetail.officer)">
            装备战斗加成：<span class="green">{{ ezfy.officerEquipText(ezfy.officerDetail.officer) }}</span>
          </div>
          <div class="old-line" v-for="sp in ezfy.officerDetail.officer.set_progress" :key="'sp' + sp.set_id">
            套装「{{ sp.name }}」：{{ sp.worn }}/{{ sp.parts }} 件
            <span :class="sp.active ? 'green' : 'gray'">{{ sp.active ? '已生效' : ('还差 ' + sp.need + ' 件') }}</span>
            <span v-if="sp.active && ezfy.equipAttrText(sp)" class="green">（{{ ezfy.equipAttrText(sp) }}）</span>
          </div>
        </template>

        <!-- ===== 养成属性区 ===== -->
        <!-- ★ 2026-10-25 「属性点」行与「分配：」行已上移到核心属性区（跟着军事/学识/后勤走） -->
        <div class="old-line">
          忠诚：<b>{{ ezfy.officerDetail.officer.loyalty }}</b>
          <a href="javascript:;" @click="ezfy.doGrant">[赏赐]</a>
          <a href="javascript:;" @click="ezfy.doTreasureGrant()">[赏赐宝物]</a>
        </div>
        <div class="old-line">
          等级上限：<span :class="ezfy.officerDetail.officer.level >= ezfy.officerDetail.officer.max_level ? 'gray' : ''">
            {{ ezfy.officerDetail.officer.level >= ezfy.officerDetail.officer.max_level ? '已突破等级上限' : ezfy.officerDetail.officer.max_level + ' 级' }}
          </span>
        </div>

        <!-- 赏赐宝物：展开可选宝物列表（只列背包未穿戴的，按品质 +10/+20/+35/+50 忠诚） -->
        <div v-if="ezfy.officerTreasureOpen" class="old-line">
          <div class="gray">选择要赏赐的宝物（消耗该件宝物, 忠诚按品质提升, 最高 +50）:</div>
          <table class="ezfy-plain-table">
            <colgroup><col style="width:40%"><col style="width:25%"><col style="width:20%"><col style="width:15%"></colgroup>
            <tr><th class="nm">宝物</th><th>品质</th><th>忠诚</th><th>操作</th></tr>
            <template v-for="e in ezfy.officerTreasures">
              <tr :key="'tg' + e.id">
                <td class="nm">{{ e.name }}<template v-if="e.count > 1"> ×{{ e.count }}</template></td>
                <td :class="ezfy.qualityClass(e.tier_name)">{{ e.tier_name || '普通' }}</td>
                <td class="green">+{{ ezfy.treasureLoyaltyGain(e.tier) }}</td>
                <td><a href="javascript:;" @click="ezfy.doTreasureGrant(e)">[赏赐]</a></td>
              </tr>
            </template>
            <tr v-if="!ezfy.officerTreasures.length"><td colspan="4" class="gray">(背包没有未穿戴的采集宝物, 可去野地采集或宝物签到获取)</td></tr>
          </table>
        </div>

        <!-- ===== 技能装备区 ===== -->
        <!-- ★ 2026-10-25 技能不再有独立 tab：属性页固定列 3 个技能槽（学了显示名字+等级，
             没学只显示空槽）。[学习] = 消耗 1 本「军官技能书」**随机**学一个技能，
             所以三行的 [学习] 效果完全一样，不存在「点哪行就学哪个」。 -->
        <div class="old-line" v-for="(s, i) in ezfy.officerSkillSlots" :key="'ds' + i">
          技能{{ i + 1 }}：<template v-if="s">{{ s.name }}<span v-if="s.level" class="green"> Lv.{{ s.level }}</span></template><a href="javascript:;" @click="ezfy.doLearnRandom">[学习]</a><a v-if="s" href="javascript:;" @click="ezfy.doForget(s.name)">[遗忘]</a>
        </div>
        <div class="old-line">
          装备：<a href="javascript:;" @click="ezfy.openOfficerDetailTab('equip')">[查看]</a>
        </div>

        <!-- ===== 特殊技能区（名将组合技） ===== -->
        <!-- ★ 2026-10-25 用户要求一行说完：`特殊技能：[铁血军魂]：+攻击力180%`
             （原来是 `+180%` 再单起一行灰色括号说明，信息重复又占行）。
             effect_text 由后端按 kind 拼（+属性名数值%），值随军官等级成长。 -->
        <template v-if="ezfy.officerDetail.officer.combo_skills && ezfy.officerDetail.officer.combo_skills.length">
          <div class="old-line red" v-for="cs in ezfy.officerDetail.officer.combo_skills" :key="'ocs' + cs.id">
            特殊技能：[{{ cs.name }}]：{{ cs.effect_text || cs.value_text }}
          </div>
        </template>
        </div>

        <!-- 计谋 tab（★ 2026-10-19 从军校页搬来：12 条计谋，发动消耗信号弹）
             位置在「属性」之后、「装备」之前，
             后续「军官属性加成」也接在这一段（计谋 → 加成），所以独立成一个 tab 而不是塞进属性。 -->
        <div v-if="ezfy.officerDetailTab === 'scheme'">
          <div class="old-line">
            持有「{{ ezfy.schemeData.bullet_name }}」：
            <b :class="ezfy.schemeData.bullet_have > 0 ? 'green' : 'red'">{{ ezfy.schemeData.bullet_have }}</b> 个
            <a href="javascript:;" @click="ezfy.switchMallTab('item'); ezfy.go('mall')">[去商城购买]</a>
          </div>
          <div class="old-line" v-for="(s, i) in ezfy.schemeData.schemes" :key="'sc' + s.id">
            {{ i + 1 }}.{{ s.name }}：<br/>
            {{ s.des }}<br/>
            需要{{ ezfy.schemeData.bullet_name }}：{{ s.bullet }}
            <span class="gray">（持有 {{ ezfy.schemeData.bullet_have }}）</span>
            <!-- 先发制人 / 秘密策反：要选目标城市坐标 -->
            <template v-if="s.kind === 1 || s.kind === 6">
              <span class="gray"> 目标坐标:</span>
              <input v-model="ezfy.schemeX" type="text" placeholder="x" style="width:56px"/>
              <input v-model="ezfy.schemeY" type="text" placeholder="y" style="width:56px"/>
            </template>
            <!-- 秘密策反：用计军官固定为当前查看的这名军官，实时显示预计降低量 -->
            <span v-if="s.kind === 6" class="gray">
              <br/>用计军官：{{ ezfy.officerDetail.officer.name }}
              （学识 {{ ezfy.officerDetail.officer.learning_total }}，
              预计降低 {{ ezfy.schemeSecretDrop(ezfy.officerDetail.officer.learning_total) }} 点忠诚）
            </span>
            <!-- ★ 2026-09-30 行军计谋（神兵天降/战略转移）：作用于部队，
                 引导玩家去「军情 → 军队动态」对出征中/返回中的部队使用 -->
            <span v-if="s.kind === 2 || s.kind === 3" class="green">
              [去「军情→军队动态」对出征中/返回中的部队使用]
            </span>
            <template v-else>
              <a v-if="s.enough" href="javascript:;" @click="ezfy.doScheme(s)">[发动]</a>
              <span v-else class="red">[{{ ezfy.schemeData.bullet_name }}不足]</span>
            </template>
            <br/>--------------------
          </div>
          <div class="old-line gray" v-if="!ezfy.schemeData.schemes.length">(暂无计谋，等管理员在后台配置)</div>
          <div class="old-line">
            <a href="javascript:;" @click="ezfy.go('bag')">[背包(信号弹)]</a>
          </div>
        </div>

        <!-- 装备 tab（已穿戴 + 一键卸下 + 一键穿套装） -->
        <div v-if="ezfy.officerDetailTab === 'equip'">
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:34%"><col style="width:16%"><col style="width:30%"><col style="width:20%">
          </colgroup>
          <tr><th colspan="4">已穿戴装备
            <a v-if="ezfy.officerDetail.equipped.length" href="javascript:;" @click="ezfy.doUnequipAll">[一键卸下]</a>
          </th></tr>
          <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>操作</th></tr>
          <!-- ★ 2026-10-18 简约化：加成/套装信息改悬停(title)查看，去掉撑开表格行的展开详情卡 -->
          <template v-for="g in ezfy.officerEquipGroups">
          <tr :key="'de' + g.key">
            <td class="nm">
              <a href="javascript:;" :title="ezfy.equipItemTip(g.first)">{{ g.name }}</a><span v-if="g.count > 1" class="gray"> ×{{ g.count }}</span>
            </td>
            <td>{{ g.slot }}</td>
            <td>
              <a v-if="g.set_id" href="javascript:;" :title="ezfy.equipSetTip(g.set_id)">{{ g.set_name || ('套装' + g.set_id) }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td><a href="javascript:;" @click="ezfy.doUnequip(g.first.id)">[卸下]</a></td>
          </tr>
          </template>
          <tr v-if="!ezfy.officerDetail.equipped.length"><td colspan="4" class="gray">(未穿戴装备)</td></tr>
        </table>

        <!-- ★ 2026-10-18 一键穿戴套装：文案精简（用户「描述还能精简继续精简」） -->
        <table class="ezfy-plain-table" v-if="ezfy.officerBagSets && ezfy.officerBagSets.length">
          <colgroup>
            <col style="width:34%"><col style="width:46%"><col style="width:20%">
          </colgroup>
          <tr><th colspan="3">一键穿戴套装（同部位自动让位）</th></tr>
          <tr><th class="nm">套装</th><th>穿齐进度</th><th>操作</th></tr>
          <tr v-for="s in ezfy.officerBagSets" :key="'bs' + s.set_id">
            <td class="nm">
              <a href="javascript:;" :title="ezfy.equipSetTip(s.set_id)">{{ s.name }}</a>
              <span class="gray" v-if="s.level">（需{{ s.level }}级）</span>
            </td>
            <td>
              <span :class="s.need > 0 ? 'gray' : 'green'">
                已穿{{ s.worn }}/{{ s.parts }}件{{ s.need > 0 ? '·差' + s.need + '件生效' : '·已生效' }}
              </span>
              <span class="gray" v-if="s.need > 0 && s.bag_count > 0">（背包{{ s.bag_count }}件）</span>
            </td>
            <td><a href="javascript:;" @click="ezfy.doEquipSet(s)">[一键穿戴]</a></td>
          </tr>
        </table>

        <!-- ★ 2026-10-18 装备背包已合并进「装备」tab（检索 + 分页）；品质/等级信息悬停(title)看 -->
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:34%"><col style="width:16%"><col style="width:30%"><col style="width:20%">
          </colgroup>
          <tr><th colspan="4">装备背包</th></tr>
          <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>操作</th></tr>
          <template v-for="g in ezfy.officerBagPaged">
          <tr :key="'db' + g.key">
            <td class="nm">
              <a href="javascript:;" :title="ezfy.equipItemTip(g.first)">{{ g.name }}</a><span class="gray"> ×{{ g.count }}</span>
            </td>
            <td>{{ g.slot }}</td>
            <td>
              <a v-if="g.set_id" href="javascript:;" :title="ezfy.equipSetTip(g.set_id)">{{ g.set_name }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td>
              <a v-if="g.canEquip" href="javascript:;" @click="ezfy.doEquipGroup(g)">[穿戴]</a>
              <span v-else class="gray">部位已满</span>
            </td>
          </tr>
          </template>
          <tr v-if="!ezfy.officerBagGroups.length"><td colspan="4" class="gray">{{ ezfy.officerBagWord ? '(没有匹配「' + ezfy.officerBagWord + '」的装备)' : '(背包暂无装备)' }}</td></tr>
        </table>
        <div class="old-line">
          搜索:
          <input v-model="ezfy.officerBagWord" type="text" placeholder="装备名 / 部位 / 套装"
                 style="width:180px" @input="ezfy.officerBagPage = 1"/>
          <a href="javascript:;" @click="ezfy.officerBagWord = ''; ezfy.officerBagPage = 1">[清空]</a>
          <span class="gray">共 {{ ezfy.officerBagGroups.length }} 种</span>
        </div>
        <div class="ezfy-pager" v-if="ezfy.officerBagGroups.length > ezfy.officerBagPageSize">
          <a href="javascript:;" :class="{ disabled: ezfy.officerBagPage <= 1 }" @click="ezfy.officerBagGo(-1)">[上一页]</a>
          <span class="gray">{{ Math.min(ezfy.officerBagPage, ezfy.officerBagTotalPages) }}/{{ ezfy.officerBagTotalPages }}</span>
          <a href="javascript:;" :class="{ disabled: ezfy.officerBagPage >= ezfy.officerBagTotalPages }" @click="ezfy.officerBagGo(1)">[下一页]</a>
        </div>
        </div>

        <div class="old-line"><a href="javascript:;" @click="ezfy.go('acade')">[返回]</a></div>
      </div>
      <!-- ★ 2026-10-01 修复「点击军官有时候空白」：加载失败 / 武将不在当前城时
           不再静默留空，展示原因并引导返回军官列表（重新拉取后该行会消失/恢复） -->
      <div class="panel" v-else>
        <div class="panel-title">军官详情</div>
        <div class="old-line red">{{ ezfy.officerDetailError || '加载中...' }}</div>
        <div class="old-line"><a href="javascript:;" @click="ezfy.go('acade')">[返回军官列表]</a></div>
      </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyAcademy',
  inject: ['ezfy']
}
</script>
