<template>
  <div>
    <template v-if="ezfy.cur === 'bag'">
        <div class="panel">
          <div class="panel-title">背包 <a href="javascript:;" @click="ezfy.loadBag">[刷新]</a></div>
          <!-- ★ 2026-10-06 背包只展示道具：宝物已由底部导航「宝物」独立页承载，去掉子tab及宝物展示 -->
          <!-- ★ 检索框：道具多的时候按名字/说明筛 -->
          <div class="old-line">
            搜索:
            <input v-model="ezfy.bagWord" type="text" placeholder="道具名 / 说明关键字"
                   style="width:180px" @input="ezfy.bagPage = 1"/>
            <a href="javascript:;" @click="ezfy.bagWord = ''; ezfy.bagPage = 1">[清空]</a>
            <span class="gray">共 {{ ezfy.bagFiltered.length }} 种 / 全部 {{ ezfy.bagItems.length }} 种</span>
          </div>
          <!-- ★ 分类筛选：流式排列自动换行（与商城同款，口径也一致） -->
          <div class="ezfy-slot-grid" v-if="ezfy.bagCats.length > 1">
            <a href="javascript:;" :class="{ on: ezfy.bagCat === '' }" @click="ezfy.setBagCat('')">[全部]</a>
            <a v-for="c in ezfy.bagCats" :key="'bc' + c" href="javascript:;" :class="{ on: ezfy.bagCat === c }"
               @click="ezfy.setBagCat(c)">[{{ c }}]</a>
          </div>
          <!-- ★ 道具说明改成「点 [说明] 才展开」（商城/背包都别堆说明文字） -->
          <div class="old-line" v-for="it in ezfy.bagPaged" :key="'bi' + it.cfg_id">
            <b>{{ it.name }}</b>×{{ it.count }}
            <a v-if="it.description" href="javascript:;" @click="ezfy.toggleBagDesc(it.cfg_id)">[说明]</a>
            <a href="javascript:;" @click="ezfy.openUse(it)">[使用]</a><br/>
            <span class="gray" v-if="ezfy.bagDescId === it.cfg_id">{{ it.description }}</span>

            <!-- 使用面板: 数量 + (军官类道具)目标军官/技能 -->
            <div v-if="ezfy.useItem && ezfy.useItem.cfg_id === it.cfg_id" class="use-box">
              数量:
              <input v-model="ezfy.useCount" type="number" min="1" :max="it.count" style="width:60px"/>
              <span class="gray">/{{ it.count }}</span>
              <a href="javascript:;" @click="ezfy.useCount = it.count">[全部]</a><br/>

              <template v-if="ezfy.needOfficer(it)">
                军官:
                <select v-model="ezfy.useOfficerId">
                  <option :value="0">请选择军官</option>
                  <option v-for="o in ezfy.bagOfficers" :key="'bo' + o.id" :value="o.id">
                    {{ o.name }} Lv{{ o.level }} {{ o.status_name }}
                  </option>
                </select><br/>
              </template>
              <template v-if="it.item_type === 11">
                技能:
                <select v-model="ezfy.useSkillId">
                  <option :value="0">请选择技能</option>
                  <option v-for="s in ezfy.bagSkills" :key="'bs' + s.id" :value="s.id">
                    {{ s.name }}({{ s.effect }})
                  </option>
                </select><br/>
              </template>
              <button @click="ezfy.doUse(it)">[确认使用]</button>
              <a href="javascript:;" @click="ezfy.useItem = null">[取消]</a>
            </div>
          </div>
          <div class="old-line" v-if="!ezfy.bagItems.length">(背包空空如也)</div>
          <div class="old-line gray" v-else-if="!ezfy.bagFiltered.length">(没有匹配「{{ ezfy.bagWord }}」的道具)</div>
          <!-- ★ 分页 -->
          <div class="ezfy-pager" v-if="ezfy.bagFiltered.length > ezfy.bagPageSize">
            <a href="javascript:;" :class="{ disabled: ezfy.bagPage <= 1 }" @click="ezfy.bagGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(ezfy.bagPage, ezfy.bagTotalPages) }}/{{ ezfy.bagTotalPages }} 页 · 共 {{ ezfy.bagFiltered.length }} 种</span>
            <a href="javascript:;" :class="{ disabled: ezfy.bagPage >= ezfy.bagTotalPages }" @click="ezfy.bagGo(1)">[下一页]</a>
          </div>
          <a href="javascript:;" @click="ezfy.go('mall')">[前往商城]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'treasure'">
        <div class="panel">
          <div class="panel-title">宝物 <a href="javascript:;" @click="ezfy.loadBag()">[刷新]</a></div>
          <div class="old-line gray">采集宝物：通过野地采集或「福利 → 宝物签到」获得，可用于军衔晋升、赏赐军官加忠诚；也可出售给系统换 {{ ezfy.resNames.gold }}。</div>
          <div class="old-line" v-for="t in ezfy.bagTreasures" :key="'tr' + t.cfg_id">
            <b class="orange">{{ t.name }}</b>×{{ t.count }}
            <span class="gray">（可售 20万{{ ezfy.resNames.gold }}/件，到手 18万）</span>
            <a href="javascript:;" @click="ezfy.openSellTreasure(t)">[出售]</a>
            <template v-if="ezfy.sellTreasure && ezfy.sellTreasure.cfg_id === t.cfg_id">
              <br/>
              <span class="gray">出售</span>
              <input v-model="ezfy.sellTreasureCount" type="number" min="1" :max="t.count" style="width:60px"/>
              <span class="gray">/{{ t.count }} 件 · 实得 <b class="green">{{ ezfy.fmtN(180000 * (parseInt(ezfy.sellTreasureCount) || 0)) }}</b>{{ ezfy.resNames.gold }}</span>
              <a href="javascript:;" @click="ezfy.doSellTreasure()">[确认出售]</a>
              <a href="javascript:;" @click="ezfy.sellTreasure = null">[取消]</a>
            </template>
          </div>
          <div class="old-line gray" v-if="!ezfy.bagTreasures.length">(还没有采集到宝物，去野地采集或宝物签到吧)</div>
          <a href="javascript:;" @click="ezfy.go('map')">[去野地采集]</a>
          <a href="javascript:;" @click="ezfy.go('welfare')">[宝物签到]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'mall'">
        <div class="panel">
          <div class="panel-title">商城({{ ezfy.resNames.gold }}{{ ezfy.city.gold }} · 钻石{{ ezfy.mallDiamond }})</div>
          <!-- ★ 商城分栏：道具 / 装备散件 / 宝箱（套装件只能开宝箱，商城只卖散件）
               ★ 2026-09-24  三个分栏去掉 []、用 | 分隔并留间距 -->
          <div class="acade-tab">
            <a href="javascript:;" :class="{ on: ezfy.mallTab === 'item' }" @click="ezfy.switchMallTab('item')">道具</a><span> | </span>
            <a href="javascript:;" :class="{ on: ezfy.mallTab === 'equipment' }" @click="ezfy.switchMallTab('equipment')">装备</a><span> | </span>
            <a href="javascript:;" :class="{ on: ezfy.mallTab === 'chest' }" @click="ezfy.switchMallTab('chest')">宝箱</a>
          </div>
          <template v-if="ezfy.mallTab === 'item'">
          <!-- ★ 分类筛选：流式排列自动换行（分类由管理端维护，手机端超宽自动折到下一行） -->
          <div class="ezfy-slot-grid">
            <a href="javascript:;" :class="{ on: ezfy.mallCat === '' }" @click="ezfy.setMallCat('')">[全部]</a>
            <a v-for="c in ezfy.mallCatsList" :key="'mc' + c" href="javascript:;" :class="{ on: ezfy.mallCat === c }"
               @click="ezfy.setMallCat(c)">[{{ c }}]</a>
          </div>
          <div class="old-line gray" v-if="ezfy.mallDiamond <= 0">
            钻石余额为 0；标记为「钻石道具」的道具若标价 0 钻石可直接购买，其余需由管理员充值钻石后购买。
          </div>
          <!-- ★ 道具表格化（原来平铺一长串，用户反馈「乱」）
               ★ 商城保持简洁：只列 名称/价格/库存/操作，**道具说明不在这里显示**
                 （说明改到「背包」里点 [说明] 展开，见 cur==='bag' 那一段）。
               · dual_pay = 黄金/钻石双渠道；标价 0 显示「限时免费」；
               · 库存 -1 = 无限（管理端「数据管理 → 道具配置」维护）。 -->
          <table class="ezfy-plain-table">
            <tr><th>名称</th><th>价格</th><th>库存</th><th>操作</th></tr>
            <tr v-for="it in ezfy.mallPaged" :key="'mi' + it.id">
              <td>{{ it.name }}</td>
              <td>
                <template v-if="it.dual_pay">
                  <span class="orange">{{ it.price_diamond }}钻</span>/{{ it.price_gold }}{{ ezfy.resNames.gold }}
                </template>
                <span v-else-if="it.is_diamond" class="orange">{{ it.price_diamond > 0 ? it.price_diamond + '钻' : '限时免费' }}</span>
                <span v-else>{{ it.price_gold > 0 ? it.price_gold + ezfy.resNames.gold : '限时免费' }}</span>
              </td>
              <td>
                <span v-if="it.unlimited" class="green">无限</span>
                <span v-else :class="it.stock > 0 ? 'gray' : 'red'">{{ it.stock > 0 ? it.stock : '已售罄' }}</span>
              </td>
              <td>
                <a v-if="it.unlimited || it.stock > 0" href="javascript:;" @click="ezfy.openBuy(it)">[购买]</a>
                <span v-else class="gray">[已售罄]</span>
              </td>
            </tr>
          </table>
          <div class="old-line gray" v-if="!ezfy.mallPaged.length">(该分类下暂无道具)</div>
          <!-- ★ 分页（每页 10 件） -->
          <div class="ezfy-pager" v-if="ezfy.mallFiltered.length > ezfy.mallPageSize">
            <a href="javascript:;" :class="{ disabled: ezfy.mallPage <= 1 }" @click="ezfy.mallGo(-1)">[上一页]</a>
            <span class="gray">第 {{ Math.min(ezfy.mallPage, ezfy.mallTotalPages) }}/{{ ezfy.mallTotalPages }} 页 · 共 {{ ezfy.mallFiltered.length }} 件</span>
            <a href="javascript:;" :class="{ disabled: ezfy.mallPage >= ezfy.mallTotalPages }" @click="ezfy.mallGo(1)">[下一页]</a>
          </div>
          </template>
          <!-- ★ 装备散件（管理端在「装备列表」里维护）
               ★ 用户规则：套装装备只能通过宝箱开启，商城不再上架套装件 -->
          <template v-else-if="ezfy.mallTab === 'equipment'">
            <!-- ★ 说明一律不写进界面（别在用户能看见的地方加提示），信息记在这里：
                 · 散件用钻石购买，定价按「六项加成总和」映射到 100~500 钻（见 seed 的 ezfyEquipDiamondPrice）；
                 · 买入后到「军官 → 军官详情」穿到军官身上；
                 · 第一批套装（新兵/战士/混沌…，无系列名）只能通过[宝箱]开启，商城不售。 -->
            <!-- ★ 部位筛选（11 个部位，来自装备距离伤害表）：流式排列自动换行 -->
            <div class="ezfy-slot-grid">
              <a href="javascript:;" :class="{ on: ezfy.shopSlot === '' }" @click="ezfy.setShopSlot('')">[全部]</a>
              <a v-for="s in ezfy.shopSlots" :key="'ss' + s" href="javascript:;" :class="{ on: ezfy.shopSlot === s }"
                 @click="ezfy.setShopSlot(s)">[{{ s }}]</a>
            </div>
            <div class="old-line">
              检索：
              <input v-model="ezfy.shopWord" type="text" placeholder="名称 / 部位" style="width:150px"
                     @input="ezfy.shopPage = 1"/>
              <span class="gray">共 {{ ezfy.shopAll.length }} 件</span>
            </div>
            <table class="ezfy-plain-table">
              <!-- ★ 2026-09-25：手机（390px）下列一多名称就被折成 4~5 行 → 砍掉「属性」列
                   （原来只放一个 [查看] 按钮），名称列因此能拿到 38%。 -->
              <colgroup>
                <col style="width:16%"><col style="width:38%"><col style="width:11%"><col style="width:15%"><col style="width:20%">
              </colgroup>
              <tr><th>部位</th><th class="nm">名称</th><th>等级</th><th>价格</th><th>操作</th></tr>
              <!-- ★ 点装备名看这件自己的加成 / 点套装名看套装加成（两个入口看不同内容，买之前就能对比）。 -->
              <template v-for="p in ezfy.shopPaged">
              <tr :key="'eq' + p.id">
                <td>{{ p.slot }}</td>
                <td class="nm"><a href="javascript:;" @click="ezfy.toggleDetail(p.id, 'item')">{{ p.name }}</a>
                  <!-- 手机列窄，套装这行尽量短：「需几件」放进展开卡里，不在这里重复 -->
                  <div v-if="ezfy.setOf(p.set_id)" class="set-mini">
                    套装：<a href="javascript:;" @click="ezfy.toggleDetail(p.id, 'set')">{{ p.set_name }}</a>
                  </div>
                </td>
                <td>{{ p.level }}</td>
                <td><span class="orange">{{ p.price_diamond }}钻</span></td>
                <td>
                  <a v-if="!p.sold_out" href="javascript:;" @click="ezfy.openEquipBuy(p)">[购买]</a>
                  <span v-else class="gray">[售罄]</span>
                </td>
              </tr>
              <tr v-if="ezfy.detailRowId === p.id" :key="'dt' + p.id" class="set-card-row">
                <td :colspan="5">
                  <div class="set-card">
                    <!-- ① 点「装备名」→ 只看这件自己的加成 -->
                    <template v-if="ezfy.detailMode === 'item'">
                      <div class="sc-h"><b>{{ p.name }}</b>
                        <span :class="ezfy.qualityClass(p.tier_name)">[{{ p.tier_name || '普通' }}]</span>
                        <span class="gray">{{ p.slot }} · {{ p.level }}级 · 商城在售</span>
                      </div>
                      <div class="sc-b">装备加成：<b class="green">{{ ezfy.equipAttrText(p) || '（这件没有额外属性加成）' }}</b></div>
                      <div class="sc-b gray" v-if="ezfy.setOf(p.set_id)">所属套装：{{ ezfy.setOf(p.set_id).name }}（点套装名看套装加成）</div>
                      <div class="sc-b gray" v-else>这件是散件，不属于任何套装。</div>
                    </template>
                    <!-- ② 点「套装名」→ 只看套装加成 -->
                    <template v-else-if="ezfy.setOf(p.set_id)">
                      <div class="sc-h"><b>{{ ezfy.setOf(p.set_id).name }}</b>
                        <span :class="ezfy.qualityClass(ezfy.setOf(p.set_id).tier_name)">[{{ ezfy.setOf(p.set_id).tier_name || '特殊' }}]</span>
                        <span class="gray">穿齐 {{ ezfy.setOf(p.set_id).parts }} 件才生效</span>
                      </div>
                      <div class="sc-b">套装加成：<b class="green">{{ ezfy.setBonusText(p.set_id) || '（本套装无额外属性加成）' }}</b></div>
                      <div class="sc-b">我的进度：已拥有 <b>{{ ezfy.setOf(p.set_id).owned || 0 }}</b>/{{ ezfy.setOf(p.set_id).parts }} 件
                        <span v-if="(ezfy.setOf(p.set_id).owned || 0) >= ezfy.setOf(p.set_id).parts" class="green">已够穿齐</span>
                        <span v-else class="red">还差 {{ ezfy.setOf(p.set_id).parts - (ezfy.setOf(p.set_id).owned || 0) }} 件</span>
                      </div>
                      <div class="sc-b gray" v-if="ezfy.setOf(p.set_id).slots && ezfy.setOf(p.set_id).slots.length">部位：{{ ezfy.setOf(p.set_id).slots.join(' / ') }}</div>
                      <div class="sc-b gray">点装备名看这件自己的加成</div>
                    </template>
                    <div class="sc-b gray" v-else>套装资料还没加载出来，稍后再试。</div>
                  </div>
                </td>
              </tr>
              </template>
            </table>
            <div class="old-line gray" v-if="!ezfy.shopAll.length">(没有匹配的装备)</div>
            <div class="ezfy-pager" v-if="ezfy.shopAll.length > ezfy.shopSize">
              <a href="javascript:;" :class="{ gray: ezfy.shopPage <= 1 }" @click="ezfy.shopPage--">上一页</a>
              <span class="gray">第 {{ ezfy.shopPage }}/{{ ezfy.shopTotalPages }} 页（共 {{ ezfy.shopAll.length }} 件）</span>
              <a href="javascript:;" :class="{ gray: ezfy.shopPage >= ezfy.shopTotalPages }" @click="ezfy.shopPage++">下一页</a>
            </div>
            <div class="old-line gray">
              当前余额：{{ ezfy.resNames.gold }}{{ ezfy.fmtN(ezfy.equipShop.gold) }} · 钻石{{ ezfy.equipShop.diamond }}
            </div>
          </template>
          <!-- ★ 宝箱（用钻石/黄金买，开箱按权重出套装件；奖池由管理端维护）
               ★ 用户规则：套装军官装备的**唯一**获取途径就是这里 -->
          <template v-else>
            <!-- ★ 宝箱（说明不写进界面，记在这里）：
                 · 宝箱用钻石购买（战地补给箱用黄金）；价格 300~800 钻按品质分档；
                 · 套装宝箱开出的是「整套」（一次给该套全部件，见 ezfyGrantChestPrize 的 Kind=3）；
                   战地补给箱开单件散件；
                 · 奖池**不直接铺开** —— 点宝箱名字才展开（「别直接展示」）。 -->
            <!-- ★ 宝箱列表：奖池点名字才展开 -->
            <table class="ezfy-plain-table">
              <tr><th>宝箱</th><th>价格</th><th>奖池</th><th>操作</th></tr>
              <tr v-for="ch in ezfy.chestData.chests" :key="'ch' + ch.id">
                <td>
                  <a href="javascript:;" :class="{ on: ezfy.chestPoolId === ch.id }"
                     @click="ezfy.toggleChestPool(ch.id)">{{ ch.name }}</a>
                  <span v-if="ch.stock >= 0" :class="ch.stock > 0 ? 'gray' : 'red'">
                    （库存{{ ch.stock > 0 ? ch.stock : '0已售罄' }}）
                  </span>
                </td>
                <td>
                  <span v-if="ch.price_diamond > 0" class="orange">{{ ch.price_diamond }}钻</span>
                  <span v-else>{{ ch.price_gold }}{{ ezfy.resNames.gold }}</span>
                </td>
                <td class="gray">{{ ch.pool.length }} 项</td>
                <td>
                  <a v-if="!ch.sold_out" href="javascript:;" @click="ezfy.openChestBuy(ch)">[开箱]</a>
                  <span v-else class="gray">[售罄]</span>
                </td>
              </tr>
            </table>
            <!-- ★ 奖池（点宝箱名字才展开）：检索 + 分页 -->
            <template v-if="ezfy.chestPoolCur">
              <div class="old-line">
                <b>{{ ezfy.chestPoolCur.name }}</b> 奖池
                <span class="gray">{{ ezfy.chestPoolCur.des }}</span>
                <a href="javascript:;" @click="ezfy.chestPoolId = 0">[收起]</a>
              </div>
              <div class="old-line">
                检索：<input v-model="ezfy.chestPoolWord" type="text" placeholder="奖品名称" style="width:150px"
                       @input="ezfy.chestPoolPage = 1"/>
                <span class="gray">共 {{ ezfy.chestPoolAll.length }} 项</span>
              </div>
              <table class="ezfy-plain-table">
                <tr><th>奖品</th><th>品质</th><th>数量</th></tr>
                <tr v-for="(p, i) in ezfy.chestPoolPaged" :key="'cp' + p.kind + '_' + p.ref_id + '_' + i">
                  <td>{{ p.name }}</td>
                  <td :class="ezfy.qualityClass(p.quality)">{{ p.quality }}</td>
                  <td>{{ p.kind === 3 ? '整套' : ('×' + p.count) }}</td>
                </tr>
              </table>
              <div class="old-line gray" v-if="!ezfy.chestPoolAll.length">(没有匹配的奖品)</div>
              <div class="ezfy-pager" v-if="ezfy.chestPoolAll.length > ezfy.chestPoolSize">
                <a href="javascript:;" :class="{ gray: ezfy.chestPoolPage <= 1 }" @click="ezfy.chestPoolPage--">上一页</a>
                <span class="gray">第 {{ ezfy.chestPoolPage }}/{{ ezfy.chestPoolTotalPages }} 页（共 {{ ezfy.chestPoolAll.length }} 项）</span>
                <a href="javascript:;" :class="{ gray: ezfy.chestPoolPage >= ezfy.chestPoolTotalPages }" @click="ezfy.chestPoolPage++">下一页</a>
              </div>
            </template>
            <div class="old-line gray" v-if="!ezfy.chestData.chests.length">(暂无上架宝箱，请等管理员在后台配置)</div>
            <div class="old-line gray">
              当前余额：{{ ezfy.resNames.gold }}{{ ezfy.fmtN(ezfy.chestData.gold) }} · 钻石{{ ezfy.chestData.diamond }}
            </div>
            <div class="old-line" v-if="ezfy.chestResult && ezfy.chestResult.length"><b>上次开箱结果：</b></div>
            <div class="old-line" v-for="(r, i) in ezfy.chestResult" :key="'cr' + i">
              {{ r.name }}<span :class="ezfy.qualityClass(r.quality)">[{{ r.quality }}]</span>
            </div>
          </template>
          <a href="javascript:;" @click="ezfy.go('bag')">[背包]</a>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'mallbuy'">
        <div class="panel">
          <div class="panel-title">购买道具</div>
          <div class="old-line gray">当前余额：{{ ezfy.resNames.gold }}{{ ezfy.fmtN(ezfy.city.gold) }} · 钻石{{ ezfy.mallDiamond }}</div>
          <template v-if="ezfy.buyItem">
            <div class="old-line">
              <b>{{ ezfy.buyItem.name }}</b>
              <span v-if="ezfy.buyItem.category" class="gray">[{{ ezfy.buyItem.category }}]</span>
            </div>
            <div class="old-line" v-if="ezfy.buyItem.description">{{ ezfy.buyItem.description }}</div>
            <div class="old-line">
              价格：
              <template v-if="ezfy.buyItem.dual_pay">
                <span class="orange">{{ ezfy.buyItem.price_diamond }}钻</span>/{{ ezfy.buyItem.price_gold }}{{ ezfy.resNames.gold }}
              </template>
              <span v-else-if="ezfy.buyItem.is_diamond" class="orange">{{ ezfy.buyItem.price_diamond > 0 ? ezfy.buyItem.price_diamond + '钻' : '限时免费' }}</span>
              <span v-else>{{ ezfy.buyItem.price_gold > 0 ? ezfy.buyItem.price_gold + ezfy.resNames.gold : '限时免费' }}</span>
            </div>
            <div class="old-line">
              库存：
              <span v-if="ezfy.buyItem.unlimited" class="green">无限</span>
              <span v-else :class="ezfy.buyItem.stock > 0 ? 'gray' : 'red'">{{ ezfy.buyItem.stock > 0 ? ezfy.buyItem.stock : '已售罄' }}</span>
            </div>
            <div class="old-line">
              数量：<input v-model="ezfy.buyCount" type="number" min="1" :max="ezfy.buyMaxOf(ezfy.buyItem)" style="width:70px"/>
              <span class="gray">{{ ezfy.buyItem.unlimited ? ('单次最多 ' + ezfy.mallBuyMax + ' 个') : ('最多 ' + ezfy.buyMaxOf(ezfy.buyItem)) }}</span>
            </div>
            <div class="old-line" v-if="ezfy.buyItem.dual_pay">
              支付方式：
              <select v-model="ezfy.buyPayWith">
                <option value="gold">黄金 {{ ezfy.buyItem.price_gold * (parseInt(ezfy.buyCount) || 0) }}</option>
                <option value="diamond">钻石 {{ ezfy.buyItem.price_diamond * (parseInt(ezfy.buyCount) || 0) }}</option>
              </select>
            </div>
            <div class="old-line">
              合计：
              <b class="bb-total">
                <template v-if="ezfy.buyItem.dual_pay">{{ (ezfy.buyPayWith === 'diamond' ? ezfy.buyItem.price_diamond : ezfy.buyItem.price_gold) * (parseInt(ezfy.buyCount) || 0) }} {{ ezfy.buyPayWith === 'diamond' ? '钻石' : ezfy.resNames.gold }}</template>
                <template v-else-if="ezfy.buyItem.is_diamond">{{ ezfy.buyItem.price_diamond > 0 ? (ezfy.buyItem.price_diamond * (parseInt(ezfy.buyCount) || 0) + ' 钻石') : '限时免费' }}</template>
                <template v-else>{{ ezfy.buyItem.price_gold > 0 ? (ezfy.buyItem.price_gold * (parseInt(ezfy.buyCount) || 0) + ' ' + ezfy.resNames.gold) : '限时免费' }}</template>
              </b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="ezfy.doBuy(ezfy.buyItem)">[确认购买]</a>
              <a href="javascript:;" @click="ezfy.buyItem = null; ezfy.go('mall')">[取消]</a>
            </div>
          </template>
          <div class="old-line gray" v-else>(未选择道具)</div>
          <a href="javascript:;" @click="ezfy.go('mall')">[返回商城]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'equipbuy'">
        <div class="panel">
          <div class="panel-title">购买装备散件</div>
          <div class="old-line gray">当前余额：钻石{{ ezfy.equipShop.diamond }}</div>
          <template v-if="ezfy.equipShopBuy">
            <div class="old-line">
              <b>{{ ezfy.equipShopBuy.name }}</b>
              <span class="gray">[{{ ezfy.equipShopBuy.slot }}]</span>
              <span v-if="ezfy.equipShopBuy.sold_out" class="red">[已售罄]</span>
            </div>
            <div class="old-line">等级：{{ ezfy.equipShopBuy.level }}</div>
            <div class="old-line">属性：{{ ezfy.equipAttrText(ezfy.equipShopBuy) || '—' }}</div>
            <div class="old-line">价格：<span class="orange">{{ ezfy.equipShopBuy.price_diamond }}钻</span>/件</div>
            <div class="old-line">
              数量：<input v-model="ezfy.equipShopCount" type="number" min="1" style="width:70px"/>
            </div>
            <div class="old-line">
              合计：<b class="bb-total">{{ ezfy.equipShopBuy.price_diamond * (parseInt(ezfy.equipShopCount) || 0) }} 钻石</b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="ezfy.doBuyEquip(ezfy.equipShopBuy)">[确认购买]</a>
              <a href="javascript:;" @click="ezfy.equipShopBuy = null; ezfy.go('mall')">[取消]</a>
            </div>
          </template>
          <div class="old-line gray" v-else>(未选择装备)</div>
          <a href="javascript:;" @click="ezfy.go('mall')">[返回商城]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'chestopen'">
        <div class="panel">
          <div class="panel-title">开宝箱</div>
          <div class="old-line gray">当前余额：{{ ezfy.resNames.gold }}{{ ezfy.fmtN(ezfy.chestData.gold) }} · 钻石{{ ezfy.chestData.diamond }}</div>
          <template v-if="ezfy.chestOpen">
            <div class="old-line"><b>{{ ezfy.chestOpen.name }}</b></div>
            <div class="old-line" v-if="ezfy.chestOpen.des">{{ ezfy.chestOpen.des }}</div>
            <div class="old-line">
              价格：
              <span v-if="ezfy.chestOpen.price_diamond > 0" class="orange">{{ ezfy.chestOpen.price_diamond }}钻/个</span>
              <span v-else>{{ ezfy.chestOpen.price_gold }}{{ ezfy.resNames.gold }}/个</span>
            </div>
            <div class="old-line">奖池（{{ ezfy.chestOpen.pool.length }} 项）<span class="gray">（点奖品名可查看具体属性）</span>：</div>
            <table class="ezfy-plain-table">
              <tr><th>奖品</th><th>品质</th><th>数量</th></tr>
              <!-- ★ 2026-10-07 修复编译错误：Vue 2 的 <template> 上不能带 :key
                   （报「<template> cannot be keyed」），key 必须放在内部的真实元素上。
                   这里 template 里有两个 <tr>（主行 + 展开的详情行），各自带 key。 -->
              <template v-for="(p, i) in ezfy.chestOpen.pool">
                <tr :key="'cpo' + p.kind + '_' + p.ref_id + '_' + i">
                  <td>
                    <a href="javascript:;"
                       :class="{ on: ezfy.chestOpenDetailIdx === i }"
                       @click="ezfy.chestOpenDetailIdx = ezfy.chestOpenDetailIdx === i ? -1 : i">{{ p.name }}</a>
                  </td>
                  <td :class="ezfy.qualityClass(p.quality)">{{ p.quality }}</td>
                  <td>{{ p.kind === 3 ? '整套' : ('×' + p.count) }}</td>
                </tr>
                <tr v-if="ezfy.chestOpenDetailIdx === i" :key="'cpod' + i">
                  <td colspan="4" class="gray">{{ p.detail || '（无更多说明）' }}</td>
                </tr>
              </template>
            </table>
            <div class="old-line">
              数量
              <a href="javascript:;" @click="ezfy.chestCount = 1">[1]</a>
              <a href="javascript:;" @click="ezfy.setChestCount(5)">[5]</a>
              <a href="javascript:;" @click="ezfy.setChestCount(10)">[10]</a>
              <input v-model="ezfy.chestCount" type="number" min="1" :max="ezfy.chestOpen.open_max" style="width:70px"/>
              <span class="gray">单次最多 {{ ezfy.chestOpen.open_max }} 个</span>
            </div>
            <div class="old-line">
              合计：
              <b class="bb-total">{{ (ezfy.chestOpen.price_diamond > 0 ? ezfy.chestOpen.price_diamond : ezfy.chestOpen.price_gold) * (parseInt(ezfy.chestCount) || 0) }} {{ ezfy.chestOpen.price_diamond > 0 ? '钻石' : ezfy.resNames.gold }}</b>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="ezfy.doOpenChest(ezfy.chestOpen)">[确认开箱]</a>
              <a href="javascript:;" @click="ezfy.chestOpen = null; ezfy.go('mall')">[取消]</a>
            </div>
          </template>
          <template v-else-if="ezfy.chestResult && ezfy.chestResult.length">
            <div class="old-line"><b>开箱结果：</b></div>
            <div class="old-line" v-for="(r, i) in ezfy.chestResult" :key="'cres' + i">
              {{ r.name }}<span class="gray">({{ r.quality }})</span>
            </div>
            <div class="old-line">
              <a href="javascript:;" @click="ezfy.chestResult = []; ezfy.go('mall')">[返回商城]</a>
            </div>
          </template>
          <a href="javascript:;" @click="ezfy.go('mall')">[返回商城]</a>
        </div>
    </template>
    <template v-else-if="ezfy.cur === 'exchange'">
        <div class="panel">
          <div class="panel-title">资源交易行({{ ezfy.resNames.gold }}{{ ezfy.exchangeGold }})</div>
          <div class="old-line gray">购买他人挂单的资源; 也可挂单出售资源换取{{ ezfy.resNames.gold }}。</div>
          <div class="panel-title">卖家挂单</div>
          <div class="old-line">
            类别:
            <select v-model="ezfy.exFilter" style="width:80px" @change="ezfy.onExFilter">
              <option :value="0">全部</option>
              <option value="1">{{ ezfy.resNames.food }}</option><option value="2">{{ ezfy.resNames.steel }}</option>
              <option value="3">{{ ezfy.resNames.oil }}</option><option value="4">{{ ezfy.resNames.rare }}</option>
            </select>
            <a v-if="ezfy.exFilter" href="javascript:;" @click="ezfy.exFilter = 0; ezfy.onExFilter()">[全部]</a>
          </div>
          <table class="ezfy-ex-tbl">
            <tr><th>卖家</th><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in ezfy.exchangeOrders" :key="'eo' + e.id">
              <td>{{ e.seller_name }}</td>
              <td>{{ e.type_name }}</td>
              <td>{{ ezfy.fmtN(e.count) }}</td>
              <td>{{ ezfy.fmtN(e.total_price) }}{{ e.currency_name || ezfy.resNames.gold }}</td>
              <td><a href="javascript:;" @click="ezfy.doExchangeBuy(e)">[购买]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.exchangeOrders.length">(暂无在售订单)</div>
          <div class="ezfy-pager" v-if="ezfy.exchangeTotal > ezfy.exchangeSize">
            <a href="javascript:;" :class="{ gray: ezfy.exchangePage <= 1 }" @click="ezfy.sectionPagerGo('exo', -1)">上一页</a>
            <span class="gray">第 {{ ezfy.exchangePage }}/{{ ezfy.exchangeTotalPages }} 页（共 {{ ezfy.exchangeTotal }} 条）</span>
            <a href="javascript:;" :class="{ gray: ezfy.exchangePage >= ezfy.exchangeTotalPages }" @click="ezfy.sectionPagerGo('exo', 1)">下一页</a>
          </div>
          <div class="panel-title">我的挂单</div>
          <div class="old-line gray" v-if="ezfy.exchangeSellMax">挂单上限：最多 {{ ezfy.exchangeSellMax }} 单（当前 {{ ezfy.exchangeMTotal }} 单）</div>
          <table class="ezfy-ex-tbl" v-if="ezfy.exchangeMine.length">
            <tr><th>资源</th><th>数量</th><th>总价</th><th>操作</th></tr>
            <tr v-for="e in ezfy.exchangeMine" :key="'em' + e.id">
              <td>{{ e.type_name }}</td>
              <td>{{ ezfy.fmtN(e.count) }}</td>
              <td>{{ ezfy.fmtN(e.total_price) }}{{ e.currency_name || ezfy.resNames.gold }}</td>
              <td><a href="javascript:;" @click="ezfy.doExchangeCancel(e)">[下架]</a></td>
            </tr>
          </table>
          <div class="old-line" v-if="!ezfy.exchangeMine.length">(无在售挂单)</div>
          <div class="ezfy-pager" v-if="ezfy.exchangeMTotal > ezfy.exchangeMSize">
            <a href="javascript:;" :class="{ gray: ezfy.exchangeMPage <= 1 }" @click="ezfy.sectionPagerGo('exm', -1)">上一页</a>
            <span class="gray">第 {{ ezfy.exchangeMPage }}/{{ ezfy.exchangeMTotalPages }} 页（共 {{ ezfy.exchangeMTotal }} 条）</span>
            <a href="javascript:;" :class="{ gray: ezfy.exchangeMPage >= ezfy.exchangeMTotalPages }" @click="ezfy.sectionPagerGo('exm', 1)">下一页</a>
          </div>
          <div class="panel-title">挂单出售</div>
          <div class="old-line">
            资源:
            <select v-model="ezfy.sellType" style="width:70px">
              <option value="1">{{ ezfy.resNames.food }}</option><option value="2">{{ ezfy.resNames.steel }}</option>
              <option value="3">{{ ezfy.resNames.oil }}</option><option value="4">{{ ezfy.resNames.rare }}</option>
            </select><br/>
            数量: <input v-model="ezfy.sellCount" type="number" style="width:90px"/><br/>
            总价({{ ezfy.resNames.gold }}): <input v-model="ezfy.sellPrice" type="number" style="width:90px" max="1000000000" placeholder="单价≤100"/><br/>
            <div class="old-line gray">单价不得超过 100 {{ ezfy.resNames.gold }}/单位</div>
            <button @click="ezfy.doExchangeSell">[挂单出售]</button>
          </div>
          <div class="panel-title">向系统出售</div>
          <div class="old-line">
            资源:
            <select v-model="ezfy.sellSysType" style="width:70px">
              <option value="1">{{ ezfy.resNames.food }}</option><option value="2">{{ ezfy.resNames.steel }}</option>
              <option value="3">{{ ezfy.resNames.oil }}</option><option value="4">{{ ezfy.resNames.rare }}</option>
            </select><br/>
            数量: <input v-model="ezfy.sellSysCount" type="number" style="width:90px"/><br/>
            <div class="old-line gray">
              每100单位 → {{ ezfy.sysSellRatio[ezfy.sellSysType] || 0 }} {{ ezfy.resNames.gold }}，实得再扣 {{ ezfy.sysSellFee }}%（应得 {{ ezfy.sysSellPreview() }} {{ ezfy.resNames.gold }}）
            </div>
            <button @click="ezfy.doExchangeSysSell">[向系统出售]</button>
            <div v-if="ezfy.sysSellLostWarn" class="red">{{ ezfy.sysSellLostWarn }}</div>
          </div>
          <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
        </div>
    </template>
  </div>
</template>

<script>
export default {
  name: 'EzfyShop',
  inject: ['ezfy']
}
</script>
