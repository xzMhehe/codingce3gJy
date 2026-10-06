<template>
  <div>
    <!-- ============ 统帅(info) ============ -->
    <template v-if="ezfy.cur === 'info'">
      <div class="panel">
        <div class="panel-title">统帅信息</div>
        <!-- ★ 只展示「玩家号码(游戏ID)」——不展示家园号码 -->
        玩家号码：{{ ezfy.selfInfo.game_uid || ezfy.profile.game_uid || ezfy.userBrief.game_uid || '—' }}<br/>
        昵称：{{ ezfy.selfInfo.nickname || ezfy.profile.nickname }}
        <template v-if="!ezfy.renameEditing">
          <a href="javascript:;" @click="ezfy.startRename">[修改昵称]</a>
        </template>
        <template v-else>
          <br/>
          <input v-model="ezfy.renameInput" maxlength="12" placeholder="2~12 个字符" style="width:130px"/>
          <button @click="ezfy.doPlayerRename">确定</button>
          <a href="javascript:;" @click="ezfy.renameEditing = false">[取消]</a>
        </template>
        <br/>
        <!-- ★ 2026-10-05 「没用的页面提示去掉」：改名/阵营提示不再展示（文案保留在代码里） -->
        阵营：{{ ezfy.selfInfo.camp_name || (ezfy.profile.camp === 2 ? '轴心国' : '同盟国') }}
        <a href="javascript:;" @click="ezfy.doChangeCamp(1)">[转同盟国]</a>
        <a href="javascript:;" @click="ezfy.doChangeCamp(2)">[转轴心国]</a><br/>
        声望：{{ ezfy.profile.prestige }}<br/>
        军衔：{{ ezfy.rankName }}({{ ezfy.rankPost }})<span style="margin-left:4px"><span v-html="ezfy.rankIcon(ezfy.myRankId)"></span></span><br/>
        军团：{{ (ezfy.myCorps && ezfy.myCorps.name) || '无' }}<br/>
        <!-- ★ 2026-09-27 统帅信息展示军团；有军团职务(军团长/副团长/参谋长)才展示职务 -->
        <template v-if="ezfy.myCorpsTitle">职务：{{ ezfy.myCorpsTitle }}<br/></template>
        城市数：{{ ezfy.cities.length }}<br/>
        人口数：{{ ezfy.city.pop }}<br/>
        军官数：{{ ezfy.officerCount }}<br/>
        <br/>
        总兵力：{{ ezfy.totalTroops }}<br/>
        城外行进：{{ ezfy.marching }}支 | 驻守采集：{{ ezfy.occupying }}支<br/>
        占领野地：{{ ezfy.wildlands.length }}块<br/>
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
          <span class="gray">(玩家号码 {{ ezfy.playerInfo.game_uid || ezfy.playerInfo.user_id }})</span>
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
          城市最高兵力数：{{ ezfy.fmtN(ezfy.playerInfo.troop_max) }}<br/>
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

    <!-- ============ 军官/学院(acade) ============ -->
    <template v-else-if="ezfy.cur === 'acade'">
      <div class="panel-title">
        参谋部({{ ezfy.officerData.staff_level }}级)
        <a href="javascript:;" @click="ezfy.switchAcade('search')">去招募</a> |
        <a href="javascript:;" @click="ezfy.switchAcade('captive')">战俘营</a>
      </div>
      <div class="acade-tab">
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'officer' }" @click="ezfy.switchAcade('officer')">军官</a>|
        <a href="javascript:;" :class="{ on: ezfy.acadeTab === 'scheme' }" @click="ezfy.switchAcade('scheme')">计谋</a>|
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
            状态:{{ o.status_name }}<span v-if="o.position_name !== '无'" class="blue">（{{ o.position_name }}）</span> &nbsp; 评价:{{ o.star }}星<br/>
            后勤/军事/学识/忠诚：<br/>
            {{ o.logistics_total }}/{{ o.military_total }}/{{ o.learning_total }}/{{ o.loyalty }}
            <span class="green" v-if="ezfy.equipTip(o)">{{ ezfy.equipTip(o) }}</span><br/>
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
          <a href="javascript:;" @click="ezfy.doUseRecruitTicket">[使用招生简章刷新]</a>
          <span class="gray">(持有 {{ ezfy.bagCount(13) }} 张)</span>
        </div>
        <div class="old-line">
          军校等级决定每小时候选数量, 参谋部{{ ezfy.recruitData.staff_level }}级(已用{{ ezfy.recruitData.used }}/{{ ezfy.recruitData.capacity }}),
          雇佣费用 = 军官等级 × 1000 {{ ezfy.resNames.gold }}
        </div>
        <div class="old-line" v-if="!ezfy.recruitData.academy_level">尚未建造军校, 无法招募军官</div>
        <table v-else class="ezfy-plain-table">
          <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
          <tr v-for="g in ezfy.recruitData.candidates" :key="'rc' + g.key">
            <td>{{ g.name }}</td>
            <td>{{ g.level }}级</td>
            <td>{{ g.star }}星</td>
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
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id)">所属套装：{{ ezfy.setOf(e.set_id).name }}（点套装名看套装加成）</div>
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
          <span class="gray">第 {{ Math.min(ezfy.equipPage, ezfy.equipTotalPages) }}/{{ ezfy.equipTotalPages }} 页 · 共 {{ ezfy.equipGroups.length }} 种</span>
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
                  <div class="sc-b gray" v-if="ezfy.setOf(e.set_id)">所属套装：{{ ezfy.setOf(e.set_id).name }}（点套装名看套装加成）</div>
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
          <span class="gray">第 {{ Math.min(ezfy.equipAllPage, ezfy.equipAllTotalPages) }}/{{ ezfy.equipAllTotalPages }} 页 · 共 {{ ezfy.equipAllFiltered.length }} 件</span>
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
             学习/遗忘统一走 军官列表 → 详情 → 技能 tab -->
      </div>

      <!-- 计谋(复刻原版 acade/scheme.html: 12 条计谋, 发动消耗信号弹)
           ★ 2026-09-22：计谋配置改由后端下发（管理端可维护消耗数量/上下架），
             页面显示持有的信号弹数量，够了才能发动。 -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'scheme'">
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
          <!-- 先发制人：要选目标城市坐标 -->
          <template v-if="s.kind === 1">
            <span class="gray"> 目标坐标:</span>
            <input v-model="ezfy.schemeX" type="text" placeholder="x" style="width:56px"/>
            <input v-model="ezfy.schemeY" type="text" placeholder="y" style="width:56px"/>
          </template>
          <!-- ★ 2026-09-30 行军计谋（神兵天降/战略转移）：作用于部队，不在军校页发动，
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
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
      </div>

      <!-- 战俘营(复刻原版 acade/conquer.html) -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'captive'">
        <div class="old-line">
          参谋部({{ ezfy.officerData.staff_level }}级)
          <a href="javascript:;" @click="ezfy.switchAcade('search')">去招募</a> |
          战俘营({{ ezfy.officerData.captive_used || 0 }}/{{ ezfy.officerData.captive_capacity || 0 }})
        </div>
        <table class="ezfy-plain-table">
          <tr><th>姓名</th><th>等级</th><th>星级</th><th>后/军/学</th><th>费用</th><th>招募</th></tr>
          <tr v-for="o in ezfy.captiveOfficers" :key="'cp' + o.id">
            <td>{{ o.name }}</td>
            <td>{{ o.level }}级</td>
            <td>{{ o.star }}星</td>
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

      <!-- 名将图鉴 -->
      <div class="panel" v-else-if="ezfy.acadeTab === 'generals'">
        <div class="old-line">名将图鉴(共{{ ezfy.generalData.generals.length }}名)</div>
        <table class="ezfy-plain-table">
          <!-- ★ 2026-10-05 去掉等级/星级列 -->
          <tr><th>名称</th><th>军/后/学</th><th>状态</th></tr>
          <tr v-for="g in ezfy.generalData.generals" :key="'gg' + g.id">
            <td>{{ g.name }}</td>
            <td>{{ g.military }}/{{ g.logistics }}/{{ g.learning }}</td>
            <td>
              <span v-if="g.owned" class="green">已拥有</span>
              <span v-else class="gray">未拥有</span>
            </td>
          </tr>
        </table>
      </div>
    </template>

    <!-- ============ 军官详情(officerdetail) ============ -->
    <template v-else-if="ezfy.cur === 'officerdetail'">
      <!-- ★ 军官详情：按 tab 分「属性 / 技能 / 装备」三块（同 acade 页 .acade-tab 写法） -->
      <div class="panel" v-if="ezfy.officerDetail.officer">
        <div class="panel-title">
          {{ ezfy.officerDetail.officer.name }}
          <!-- ★ 2026-10-04 名将标识：详情页头部徽标 + 改名后显示原名 -->
          <span v-if="ezfy.officerDetail.officer.is_general" class="orange">【名将】</span>
          <span v-if="ezfy.officerDetail.officer.is_general && ezfy.officerDetail.officer.general_name &&
                       ezfy.officerDetail.officer.general_name !== ezfy.officerDetail.officer.name" class="gray">
            （原名 {{ ezfy.officerDetail.officer.general_name }}）
          </span>
          <a href="javascript:;" @click="ezfy.doOfficerRename">[改名]</a>
        </div>

        <div class="acade-tab">
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'attr' }" @click="ezfy.officerDetailTab = 'attr'">属性</a>|
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'skill' }" @click="ezfy.officerDetailTab = 'skill'">技能</a>|
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'equip' }" @click="ezfy.officerDetailTab = 'equip'">装备</a>|
          <a href="javascript:;" :class="{ on: ezfy.officerDetailTab === 'bag' }" @click="ezfy.officerDetailTab = 'bag'">装备背包</a>
          <!-- ★ 2026-10-04 名将背景 tab 放到最后（仅名将显示）：二战的功勋介绍 -->
          <a v-if="ezfy.officerDetail.officer.is_general" href="javascript:;"
             :class="{ on: ezfy.officerDetailTab === 'general' }" @click="ezfy.officerDetailTab = 'general'"><span style="margin:0 6px 0 2px;color:#888;">|</span>名将背景</a>
        </div>

        <!-- 名将背景 tab -->
        <div v-if="ezfy.officerDetailTab === 'general' && ezfy.officerDetail.officer.is_general">
          <div class="old-line">
            名将：<b>{{ ezfy.officerDetail.officer.general_name || ezfy.officerDetail.officer.name }}</b>
            <span class="gray" v-if="ezfy.officerDetail.officer.general_name && ezfy.officerDetail.officer.general_name !== ezfy.officerDetail.officer.name">
              （当前已改名为「{{ ezfy.officerDetail.officer.name }}」）
            </span>
          </div>
          <div class="old-line" v-if="ezfy.officerDetail.officer.general_star || ezfy.officerDetail.officer.general_level">
            池中星级：{{ ezfy.officerDetail.officer.general_star || '—' }}星 &nbsp; 池中等级：{{ ezfy.officerDetail.officer.general_level || '—' }}级
          </div>
          <div class="old-line" v-if="ezfy.officerDetail.officer.general_des">
            二战功勋：{{ ezfy.officerDetail.officer.general_des }}
          </div>
          <div class="old-line gray" v-else>(该名将暂无功勋介绍)</div>
        </div>

        <!-- 属性 tab -->
        <div v-if="ezfy.officerDetailTab === 'attr'">
        <div class="old-line">
          星级：<b>{{ ezfy.officerDetail.officer.star }}</b><span class="gray" v-if="ezfy.officerDetail.officer.star_max">/{{ ezfy.officerDetail.officer.star_max }}</span>
          &nbsp;等级：<b>{{ ezfy.officerDetail.officer.level }}</b>
          &nbsp;经验：<span class="gray">{{ ezfy.officerDetail.officer.level >= ezfy.detailOfficerMaxLevel ? '—' : (ezfy.officerDetail.officer.exp + '/' + ezfy.officerDetail.officer.exp_need) }}</span>
          &nbsp;忠诚：<b>{{ ezfy.officerDetail.officer.loyalty }}</b>
          <br/>
          职位：<b>{{ ezfy.officerDetail.officer.position_name }}</b>
          &nbsp;状态：<b :class="ezfy.officerDetail.officer.status_name === '出征中' ? 'red' : ''">{{ ezfy.officerDetail.officer.status_name }}</b>
        </div>
        <hr/>
        <div class="old-line">
          军事：{{ ezfy.officerDetail.officer.military_total }}<span class="green" v-if="ezfy.officerDetail.officer.equip_military">(+{{ ezfy.officerDetail.officer.equip_military }})</span>
          &nbsp;后勤：{{ ezfy.officerDetail.officer.logistics_total }}<span class="green" v-if="ezfy.officerDetail.officer.equip_logistics">(+{{ ezfy.officerDetail.officer.equip_logistics }})</span>
          &nbsp;学识：{{ ezfy.officerDetail.officer.learning_total }}<span class="green" v-if="ezfy.officerDetail.officer.equip_learning">(+{{ ezfy.officerDetail.officer.equip_learning }})</span>
          <br/>
          攻击加成：{{ ezfy.officerDetail.officer.attack }}
          &nbsp;防御加成：{{ ezfy.officerDetail.officer.defence }}
        </div>

        <!-- 套装与战斗加成（套装穿齐才生效） -->
        <template v-if="(ezfy.officerDetail.officer.set_progress && ezfy.officerDetail.officer.set_progress.length) ||
                         ezfy.officerBattleText(ezfy.officerDetail.officer.battle)">
          <hr/>
          <div class="old-line" v-if="ezfy.officerBattleText(ezfy.officerDetail.officer.battle)">
            装备战斗加成：<span class="green">{{ ezfy.officerBattleText(ezfy.officerDetail.officer.battle) }}</span>
          </div>
          <div class="old-line" v-for="sp in ezfy.officerDetail.officer.set_progress" :key="'sp' + sp.set_id">
            套装「{{ sp.name }}」：{{ sp.worn }}/{{ sp.parts }} 件
            <span :class="sp.active ? 'green' : 'gray'">{{ sp.active ? '已生效' : ('还差 ' + sp.need + ' 件') }}</span>
            <span v-if="sp.active && ezfy.equipAttrText(sp)" class="green">（{{ ezfy.equipAttrText(sp) }}）</span>
          </div>
        </template>

        <!-- 属性加点（每升 1 级得 1 点） -->
        <div class="old-line">
          可用属性点
          <b :class="ezfy.officerDetail.officer.free_points > 0 ? 'red' : 'gray'">{{ ezfy.officerDetail.officer.free_points }}</b>
          <span class="gray">（已分配 {{ ezfy.officerDetail.officer.used_points }}）</span>
          <span class="gray" v-if="ezfy.officerDetail.officer.star_points > 0">（其中升星加点 {{ ezfy.officerDetail.officer.star_points }}）</span>
        </div>
        <div class="old-line" v-if="ezfy.officerDetail.officer.free_points > 0">
          分配：
          军事<a href="javascript:;" @click="ezfy.doAddAttr('military', 1)">[+1]</a><a href="javascript:;" @click="ezfy.doAddAttr('military', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('military')">[全加]</a>
          &nbsp;后勤<a href="javascript:;" @click="ezfy.doAddAttr('logistics', 1)">[+1]</a><a href="javascript:;" @click="ezfy.doAddAttr('logistics', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('logistics')">[全加]</a>
          &nbsp;学识<a href="javascript:;" @click="ezfy.doAddAttr('learning', 1)">[+1]</a><a href="javascript:;" @click="ezfy.doAddAttr('learning', 10)">[+10]</a><a href="javascript:;" @click="ezfy.doAddAttrAll('learning')">[全加]</a>
        </div>

        <!-- 操作 -->
        <div class="old-line officer-actions">
          <a href="javascript:;" @click="ezfy.doGrant">[赏赐+10忠诚(1万金)]</a>
          <a href="javascript:;" @click="ezfy.doTreasureGrant()">[赏赐宝物]</a>
          <a href="javascript:;" @click="ezfy.doRespec">[洗点]</a>
          <span v-if="ezfy.bagCount(16) > 0" class="gray">(持有军官洗点卡 {{ ezfy.bagCount(16) }} 张)</span>
          <a v-if="ezfy.officerDetail.officer.status !== 1 && ezfy.officerDetail.officer.position === 0"
             href="javascript:;" @click="ezfy.doExile">[流放]</a>
          <a v-if="ezfy.officerDetail.officer.star_up_on &&
                   ezfy.officerDetail.officer.star < ezfy.officerDetail.officer.star_max"
             href="javascript:;" @click="ezfy.doStarUp">[升星]</a>
          <span v-if="ezfy.officerDetail.officer.status === 1" class="gray">(出征中, 归来后才能流放)</span>
          <span v-else-if="ezfy.officerDetail.officer.position !== 0" class="gray">(市长/城守, 卸任后才能流放)</span>
          <span v-if="ezfy.officerDetail.officer.star_up_on && ezfy.officerDetail.officer.star < ezfy.officerDetail.officer.star_max"
                class="gray">星级徽章 {{ ezfy.officerDetail.officer.star_card }} 枚</span>
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
        </div>

        <!-- 技能 tab（已学 / 可学） -->
        <div v-if="ezfy.officerDetailTab === 'skill'">
        <!-- ★ 2026-10-06 技能随军官等级自动升级：已学技能展示当前等级 Lv.N，效果已按等级倍数返回 -->
        <table class="ezfy-plain-table">
          <colgroup><col style="width:22%"><col style="width:63%"><col style="width:15%"></colgroup>
          <tr><th colspan="3">已学技能（{{ ezfy.officerDetail.skills.length }}/3，随军官等级自动升级）</th></tr>
          <tr v-for="s in ezfy.officerDetail.skills" :key="'ds' + s.name">
            <td>{{ s.name }}<span v-if="s.level" class="green"> Lv.{{ s.level }}</span></td>
            <td>{{ s.effect }}</td>
            <td><a href="javascript:;" @click="ezfy.doForget(s.name)">[遗忘]</a></td>
          </tr>
          <tr v-if="!ezfy.officerDetail.skills.length"><td colspan="3" class="gray">(未学任何技能)</td></tr>
        </table>
        <table class="ezfy-plain-table">
          <colgroup><col style="width:22%"><col style="width:63%"><col style="width:15%"></colgroup>
          <!-- ★ 2026-10-06 学习后按军官当前等级自动定级（每30级+1级：普通军官最高5级 / 名将最高6级） -->
          <tr><th colspan="3">可学技能（技能书 {{ ezfy.officerDetail.officer.skill_book }} 本 / 学一个消耗1本，学成 = Lv.{{ ezfy.officerDetail.officer.skill_level }}级）</th></tr>
          <tr v-for="s in ezfy.officerDetail.all_skills" :key="'ls' + s.id">
            <td>{{ s.name }}</td>
            <td>{{ s.effect }}</td>
            <td><a href="javascript:;" @click="ezfy.doLearn(s)">[学习]</a></td>
          </tr>
        </table>
        </div>

        <!-- 装备 tab（已穿戴 + 一键卸下 + 一键穿套装） -->
        <div v-if="ezfy.officerDetailTab === 'equip'">
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:30%"><col style="width:14%"><col style="width:12%"><col style="width:22%"><col style="width:22%">
          </colgroup>
          <tr><th colspan="5">已穿戴装备
            <a v-if="ezfy.officerDetail.equipped.length" href="javascript:;" @click="ezfy.doUnequipAll">[一键卸下]</a>
          </th></tr>
          <tr><th class="nm">名称</th><th>部位</th><th>品质</th><th>套装</th><th>操作</th></tr>
          <!-- ★ 2026-09-29：已穿戴装备同 cfg 叠加成一行（数量 >1 显示 ×N）——
               点装备名看这件加成 / 点套装名看套装加成（两个入口看不同内容）。 -->
          <template v-for="g in ezfy.officerEquipGroups">
          <tr :key="'de' + g.key">
            <td class="nm"><a href="javascript:;" @click="ezfy.toggleDetail(g.first.id, 'item')">{{ g.name }}</a><span v-if="g.count > 1" class="gray"> ×{{ g.count }}</span></td>
            <td>{{ g.slot }}</td>
            <td :class="ezfy.qualityClass(g.tier_name)">{{ g.tier_name || '—' }}</td>
            <td>
              <a v-if="g.set_id" href="javascript:;" @click="ezfy.toggleDetail(g.first.id, 'set')">{{ g.set_name || ('套装' + g.set_id) }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td><a href="javascript:;" @click="ezfy.doUnequip(g.first.id)">[卸下]</a></td>
          </tr>
          <tr v-if="ezfy.detailRowId === g.first.id" :key="'dt' + g.key" class="set-card-row">
            <td :colspan="5">
              <div class="set-card">
                <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                <template v-if="ezfy.detailMode === 'item'">
                  <div class="sc-h"><b>{{ g.first.name }}</b>
                    <span :class="ezfy.qualityClass(g.first.tier_name)">[{{ g.first.tier_name || '普通' }}]</span>
                    <span class="gray">{{ g.first.slot || g.first.type }} · 已穿戴</span>
                  </div>
                  <div class="sc-b">装备加成：<b class="green">{{ ezfy.equipAttrText(g.first) || '（这件没有额外属性加成）' }}</b></div>
                  <div class="sc-b gray" v-if="ezfy.setOf(g.first.set_id)">所属套装：{{ ezfy.setOf(g.first.set_id).name }}（点套装名看套装加成）</div>
                  <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                </template>
                <!-- ② 点「套装名」→ 只看套装加成 -->
                <template v-else-if="ezfy.setOf(g.first.set_id)">
                  <div class="sc-h"><b>{{ ezfy.setOf(g.first.set_id).name }}</b>
                    <span :class="ezfy.qualityClass(ezfy.setOf(g.first.set_id).tier_name)">[{{ ezfy.setOf(g.first.set_id).tier_name || '特殊' }}]</span>
                    <span class="gray">穿齐 {{ ezfy.setOf(g.first.set_id).parts }} 件才生效</span>
                  </div>
                  <div class="sc-b">套装加成：<b class="green">{{ ezfy.setBonusText(g.first.set_id) || '（本套装无额外属性加成）' }}</b></div>

                  <div class="sc-b">我的进度：已拥有 <b>{{ ezfy.setOf(g.first.set_id).owned || 0 }}</b>/{{ ezfy.setOf(g.first.set_id).parts }} 件
                    <span v-if="(ezfy.setOf(g.first.set_id).owned || 0) >= ezfy.setOf(g.first.set_id).parts" class="green">已够穿齐</span>
                    <span v-else class="red">还差 {{ ezfy.setOf(g.first.set_id).parts - (ezfy.setOf(g.first.set_id).owned || 0) }} 件</span>
                  </div>
                  <div class="sc-b gray" v-if="ezfy.setOf(g.first.set_id).slots && ezfy.setOf(g.first.set_id).slots.length">部位：{{ ezfy.setOf(g.first.set_id).slots.join(' / ') }}</div>
                  <div class="sc-b gray">点装备名看这件自己的加成</div>
                </template>
                <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
              </div>
            </td>
          </tr>
          </template>
          <tr v-if="!ezfy.officerDetail.equipped.length"><td colspan="5" class="gray">(未穿戴装备)</td></tr>
        </table>

        <!-- 一键穿戴套装（背包里有件的套装） -->
        <table class="ezfy-plain-table" v-if="ezfy.officerDetail.bag_sets && ezfy.officerDetail.bag_sets.length">
          <colgroup>
            <col style="width:36%"><col style="width:32%"><col style="width:12%"><col style="width:20%">
          </colgroup>
          <tr><th colspan="4">一键穿戴套装（同部位已穿戴的会自动卸下让位）</th></tr>
          <tr><th class="nm">套装</th><th>穿齐进度</th><th>等级</th><th>操作</th></tr>
          <tr v-for="s in ezfy.officerDetail.bag_sets" :key="'bs' + s.set_id">
            <td class="nm">{{ s.name }}</td>
            <td>
              <span :class="s.need > 0 ? 'gray' : 'green'">
                已穿 {{ s.worn }}/{{ s.parts }} 件{{ s.need > 0 ? (' · 还差 ' + s.need + ' 件生效') : ' · 已生效' }}
              </span>
              <span class="gray" v-if="s.need > 0 && s.bag_count > 0">（背包还有 {{ s.bag_count }} 件）</span>
              <!-- ★ 2026-09-25：一键穿戴这里也把套装加成写出来，玩家才知道穿齐能拿到什么 -->
              <div class="set-mini" v-if="ezfy.setBonusText(s.set_id)">
                套装加成：<b class="green">{{ ezfy.setBonusText(s.set_id) }}</b>
              </div>
            </td>
            <td>{{ s.level }}</td>
            <td><a href="javascript:;" @click="ezfy.doEquipSet(s)">[一键穿戴]</a></td>
          </tr>
        </table>
        </div>

        <!-- 装备背包 tab（检索 + 分页） -->
        <div v-if="ezfy.officerDetailTab === 'bag'">
        <table class="ezfy-plain-table">
          <colgroup>
            <col style="width:30%"><col style="width:12%"><col style="width:18%"><col style="width:11%"><col style="width:11%"><col style="width:18%">
          </colgroup>
          <tr><th colspan="6">装备背包</th></tr>
          <!-- ★ 2026-09-29：同一件装备（同 cfg）叠加成一行「名称 ×N」；部位有空余才能 [穿戴]，
               没空余（同部位已穿戴）显示「部位已满」不可穿戴。 -->
          <tr><th class="nm">名称</th><th>部位</th><th>套装</th><th>品质</th><th>要求等级</th><th>操作</th></tr>
          <template v-for="g in ezfy.officerBagPaged">
          <tr :key="'db' + g.key">
            <td class="nm"><a href="javascript:;" @click="ezfy.toggleDetail(g.first.id, 'item')">{{ g.name }}</a><span class="gray"> ×{{ g.count }}</span></td>
            <td>{{ g.slot }}</td>
            <td>
              <a v-if="g.set_id" href="javascript:;" @click="ezfy.toggleDetail(g.first.id, 'set')">{{ g.set_name }}</a>
              <span v-else class="gray">—</span>
            </td>
            <td :class="ezfy.qualityClass(g.tier_name)">{{ g.tier_name }}</td>
            <td>{{ g.first.level }}</td>
            <td>
              <a v-if="g.canEquip" href="javascript:;" @click="ezfy.doEquipGroup(g)">[穿戴]</a>
              <span v-else class="gray">部位已满</span>
            </td>
          </tr>
          <tr v-if="ezfy.detailRowId === g.first.id" :key="'dtb' + g.key" class="set-card-row">
            <td :colspan="6">
              <div class="set-card">
                <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                <template v-if="ezfy.detailMode === 'item'">
                  <div class="sc-h"><b>{{ g.first.name }}</b>
                    <span :class="ezfy.qualityClass(g.first.tier_name)">[{{ g.first.tier_name || '普通' }}]</span>
                    <span class="gray">{{ g.first.slot || g.first.type }} · {{ g.first.level }}级 · 背包 {{ g.count }} 件</span>
                  </div>
                  <div class="sc-b">装备加成：<b class="green">{{ ezfy.equipAttrText(g.first) || '（这件没有额外属性加成）' }}</b></div>
                  <div class="sc-b gray" v-if="ezfy.setOf(g.first.set_id)">所属套装：{{ ezfy.setOf(g.first.set_id).name }}（点套装名看套装加成）</div>
                  <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                </template>
                <!-- ② 点「套装名」→ 只看套装加成 -->
                <template v-else-if="ezfy.setOf(g.first.set_id)">
                  <div class="sc-h"><b>{{ ezfy.setOf(g.first.set_id).name }}</b>
                    <span :class="ezfy.qualityClass(ezfy.setOf(g.first.set_id).tier_name)">[{{ ezfy.setOf(g.first.set_id).tier_name || '特殊' }}]</span>
                    <span class="gray">穿齐 {{ ezfy.setOf(g.first.set_id).parts }} 件才生效</span>
                  </div>
                  <div class="sc-b">套装加成：<b class="green">{{ ezfy.setBonusText(g.first.set_id) || '（本套装无额外属性加成）' }}</b></div>

                  <div class="sc-b">我的进度：已拥有 <b>{{ ezfy.setOf(g.first.set_id).owned || 0 }}</b>/{{ ezfy.setOf(g.first.set_id).parts }} 件
                    <span v-if="(ezfy.setOf(g.first.set_id).owned || 0) >= ezfy.setOf(g.first.set_id).parts" class="green">已够穿齐</span>
                    <span v-else class="red">还差 {{ ezfy.setOf(g.first.set_id).parts - (ezfy.setOf(g.first.set_id).owned || 0) }} 件</span>
                  </div>
                  <div class="sc-b gray" v-if="ezfy.setOf(g.first.set_id).slots && ezfy.setOf(g.first.set_id).slots.length">部位：{{ ezfy.setOf(g.first.set_id).slots.join(' / ') }}</div>
                  <div class="sc-b gray">点装备名看这件自己的加成</div>
                </template>
                <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
              </div>
            </td>
          </tr>
          </template>
          <tr v-if="!ezfy.officerBagGroups.length"><td colspan="6" class="gray">{{ ezfy.officerBagWord ? '(没有匹配「' + ezfy.officerBagWord + '」的装备)' : '(背包暂无装备)' }}</td></tr>
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
          <span class="gray">第 {{ Math.min(ezfy.officerBagPage, ezfy.officerBagTotalPages) }}/{{ ezfy.officerBagTotalPages }} 页 · 共 {{ ezfy.officerBagGroups.length }} 种</span>
          <a href="javascript:;" :class="{ disabled: ezfy.officerBagPage >= ezfy.officerBagTotalPages }" @click="ezfy.officerBagGo(1)">[下一页]</a>
        </div>
        </div>

        <div class="old-line"><a href="javascript:;" @click="ezfy.go('acade')">[返回军官]</a></div>
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
