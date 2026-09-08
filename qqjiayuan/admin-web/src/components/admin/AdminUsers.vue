<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openEditor(null)">新增用户</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="username" label="号码" width="90" />
        <el-table-column label="昵称" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}"><b>{{ row.nickname }}</b></template>
        </el-table-column>
        <el-table-column label="马甲" width="90">
          <template slot-scope="{row}">
            <img v-for="b in (row.badges || [])" :key="b.id" class="bicon" :src="$pic(b.icon)" :alt="b.name" :title="b.name">
          </template>
        </el-table-column>
        <el-table-column label="角色" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ (row.roles || []).map(r => r.name).join('，') || '—' }}</template>
        </el-table-column>
        <el-table-column label="等级" width="80"><template slot-scope="{row}">Lv.{{ row.level }}</template></el-table-column>
        <el-table-column label="好友策略" width="90" align="center">
          <template slot-scope="{row}">{{ {0:'允许',1:'验证',2:'拒绝'}[row.friend_policy] || '允许' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '正常' : '封禁' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" plain icon="el-icon-view" @click="openDetail(row)">详情</el-button>
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openEditor(row)">编辑</el-button>
              <el-button size="mini" :type="row.status === 1 ? 'danger' : 'success'" plain @click="setStatus(row)">
                {{ row.status === 1 ? '封禁' : '解封' }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 编辑 / 新增 模态框 -->
    <el-dialog :title="form.id ? '编辑用户：' + form.username : '新增用户'" :visible.sync="dlg"
               width="620px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="重置密码">
          <el-input v-model.trim="panel.password" maxlength="20" placeholder="留空则不修改；填写则重置为该密码（6-20位）" style="width:300px" />
        </el-form-item>
        <el-form-item label="角色">
          <el-checkbox-group v-model="panel.roleIds">
            <el-checkbox v-for="r in allRoles" :key="r.id" :label="r.id">{{ r.name }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="马甲">
          <el-checkbox-group v-model="panel.badgeIds">
            <el-checkbox v-for="b in allBadges" :key="b.id" :label="b.id">
              <img class="bicon" :src="$pic(b.icon)" :alt="b.name">{{ b.name }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="贵族">
          <el-radio-group v-model="panel.noble">
            <el-radio :label="0">无</el-radio>
            <el-radio :label="1">一级</el-radio>
            <el-radio :label="2">二级</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="特权">
          <el-select v-model="panel.privId" placeholder="无" style="width:220px" clearable>
            <el-option v-for="pv in allPrivs" :key="pv.id" :label="pv.name + '（Lv.' + pv.level + '）'" :value="pv.id">
              <img class="bicon" :src="'/static/' + pv.file" :alt="pv.name">{{ pv.name }}
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="伴侣号码">
          <el-input-number v-model="panel.partnerId" :min="0" :max="99999999" />
        </el-form-item>
        <el-form-item label="宝宝">
          <el-input v-model.trim="panel.babyName" maxlength="20" style="width:220px" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="saveAll">保存全部修改</el-button>
      </div>
    </el-dialog>

    <!-- 用户详情弹窗（对齐诺哈 admin/user：资料+IP+地址/证件/密保/联系/日志） -->
    <el-dialog :title="'用户详情：' + (detail.user ? detail.user.nickname + '（' + detail.user.username + '）' : '')"
               :visible.sync="detailDlg" width="760px">
      <div v-if="detail.user" v-loading="detailLoading">
        <el-descriptions :column="3" border size="small" title="基本资料">
          <el-descriptions-item label="性别">{{ detail.user.gender === 2 ? '女' : '男' }}</el-descriptions-item>
          <el-descriptions-item label="年龄">{{ detail.user.age || '—' }}</el-descriptions-item>
          <el-descriptions-item label="生日">{{ detail.user.solar || (detail.user.birth_year ? detail.user.birth_year + '-' + detail.user.birth_month + '-' + detail.user.birth_day : '—') }}{{ detail.user.birth_type === 0 ? '（阴历' + (detail.user.lunar || '') + '）' : '' }}</el-descriptions-item>
          <el-descriptions-item label="城市">{{ detail.user.city || '—' }}</el-descriptions-item>
          <el-descriptions-item label="签名">{{ detail.user.signature || '—' }}</el-descriptions-item>
          <el-descriptions-item label="简介">{{ detail.user.introduction || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="3" border size="small" title="经济与活跃" style="margin-top:12px">
          <el-descriptions-item label="等级">Lv.{{ detail.user.level }}（经验 {{ detail.user.exp }}）</el-descriptions-item>
          <el-descriptions-item label="货币">G币{{ detail.user.coins }} / 元宝{{ detail.user.yuanbao }} / 金钻{{ detail.user.jinzuan }} / 友友券{{ detail.user.youquan }}</el-descriptions-item>
          <el-descriptions-item label="成就">{{ detail.user.achieve }}</el-descriptions-item>
          <el-descriptions-item label="活跃天数">{{ Math.floor(detail.user.active_days || 0) }}天</el-descriptions-item>
          <el-descriptions-item label="在线时长">{{ Math.floor((detail.user.hours || 0) / 60) }}小时{{ (detail.user.hours || 0) % 60 }}分</el-descriptions-item>
          <el-descriptions-item label="贵族">{{ detail.user.noble || 0 }}</el-descriptions-item>
          <el-descriptions-item label="注册IP">{{ detail.user.add_ip || '—' }}</el-descriptions-item>
          <el-descriptions-item label="最后IP">{{ detail.user.last_ip || '—' }}</el-descriptions-item>
          <el-descriptions-item label="最后登录">{{ fmtTime(detail.user.last_login_at) }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="3" border size="small" title="安全与设置" style="margin-top:12px">
          <el-descriptions-item label="好友策略">{{ {0:'允许',1:'需要验证',2:'拒绝'}[detail.user.friend_policy] || '允许' }}</el-descriptions-item>
          <el-descriptions-item label="个性配置">{{ detail.user.config || '—' }}</el-descriptions-item>
          <el-descriptions-item label="支付密码">{{ detail.user.has_paypass ? '已设置' : '未设置' }}</el-descriptions-item>
          <el-descriptions-item label="密保">{{ detail.protection.has_protection ? '已设置（问题' + detail.protection.issue + '）' : '未设置' }}</el-descriptions-item>
          <el-descriptions-item label="实名证件">{{ detail.document.has_doc ? detail.document.real_name + ' ' + detail.document.number : '未认证' }}</el-descriptions-item>
          <el-descriptions-item label="联系方式">{{ detail.contact.qq || 'QQ未绑' }} / {{ detail.contact.mail || '邮箱未绑' }} / {{ detail.contact.phone || '手机未绑' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="2" border size="small" title="通信地址" style="margin-top:12px">
          <el-descriptions-item label="故乡">{{ addrText(detail.address, 'home') }}</el-descriptions-item>
          <el-descriptions-item label="现居">{{ addrText(detail.address, 'live') }}</el-descriptions-item>
        </el-descriptions>
        <div style="margin-top:12px">
          <div style="font-size:13px;font-weight:bold;margin-bottom:6px">最近操作日志</div>
          <el-table :data="detail.logs || []" size="mini" max-height="240" stripe>
            <el-table-column prop="action" label="动作" width="100" />
            <el-table-column prop="intro" label="详情" min-width="180" show-overflow-tooltip />
            <el-table-column prop="ip" label="IP" width="130" />
            <el-table-column label="时间" width="160"><template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template></el-table-column>
          </el-table>
        </div>
      </div>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminUsers',
  data () {
    return {
      list: [], total: 0, page: 1, pages: 1, size: 10, word: '', loading: false,
      allRoles: [], allBadges: [], allPrivs: [],
      dlg: false,
      form: { id: 0 },
      panel: { password: '', roleIds: [], badgeIds: [], noble: 0, partnerId: 0, babyName: '', privId: 0 },
      savePwd: false,
      detailDlg: false, detailLoading: false, detail: { user: null }
    }
  },
  mounted () {
    this.load()
    api.get('/admin/roles', { params: { size: 100 } }).then(r => { if (r.code === 0) this.allRoles = r.data.list })
    api.get('/admin/badges', { params: { size: 100 } }).then(r => { if (r.code === 0) this.allBadges = r.data.list })
    api.get('/privs').then(r => { if (r.code === 0) this.allPrivs = r.data })
  },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/users', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openEditor (row) {
      if (row) {
        api.get('/users/' + row.id).then(r => {
          if (r.code !== 0) { this.$message.error(r.msg); return }
          this.form = { id: row.id, username: row.username }
          this.panel = {
            password: '',
            roleIds: (r.data.roles || []).map(x => x.id),
            badgeIds: (r.data.badges || []).map(x => x.id),
            noble: r.data.noble || 0,
            partnerId: r.data.partner_id || 0,
            babyName: r.data.baby_name || '',
            privId: r.data.priv_id || 0
          }
          this.dlg = true
        })
      } else {
        this.$message({
          type: 'info',
          duration: 4000,
          showClose: true,
          message: '新用户请到社区前台注册（注册自动分配家园号码），正在为您打开注册页…'
        })
        window.open('http://' + location.host + '/#/register')
      }
    },
    saveAll () {
      const id = this.form.id
      const finish = ok => { if (ok) { this.$message.success('已保存'); this.dlg = false; this.load() } }
      if (this.panel.password) {
        if (this.panel.password.length < 6) { this.$message.error('新密码至少6位'); return }
        api.put(`/admin/users/${id}/password`, { password: this.panel.password }).then(r => {
          if (r.code !== 0) { this.$message.error(r.msg); return }
          this.panel.password = ''
          this.saveOthers(finish)
        })
      } else {
        this.saveOthers(finish)
      }
    },
    saveOthers (finish) {
      const id = this.form.id
      api.put(`/admin/users/${id}/roles`, { role_ids: this.panel.roleIds }).then(a => {
        if (a.code !== 0) { this.$message.error(a.msg); return }
        api.put(`/admin/users/${id}/badges`, { badge_ids: this.panel.badgeIds }).then(b => {
          if (b.code !== 0) { this.$message.error(b.msg); return }
          api.put(`/admin/users/${id}/extras`, {
            noble: this.panel.noble, partner_id: this.panel.partnerId || 0,
            baby_name: this.panel.babyName, priv_id: this.panel.privId || 0
          }).then(c => {
            if (c.code !== 0) { this.$message.error(c.msg); return }
            finish(true)
          })
        })
      })
    },
    setStatus (row) {
      const target = row.status === 1 ? 0 : 1
      const text = target === 0 ? `确定封禁「${row.nickname}」吗？` : `确定解封「${row.nickname}」吗？`
      this.$confirm(text, '提示', { type: 'warning' }).then(() => {
        api.put(`/admin/users/${row.id}/status`, { status: target }).then(r => {
          if (r.code === 0) { this.$message.success(target === 0 ? '已封禁' : '已解封'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openDetail (row) {
      this.detailDlg = true
      this.detailLoading = true
      this.detail = { user: null }
      api.get(`/admin/users/${row.id}/detail`).then(r => {
        this.detailLoading = false
        if (r.code === 0) this.detail = r.data
        else this.$message.error(r.msg)
      }).catch(() => { this.detailLoading = false })
    },
    addrText (a, prefix) {
      if (!a) return '—'
      const s = [a[prefix + '_nation'], a[prefix + '_prov'], a[prefix + '_city'], a[prefix + '_dist'], a[prefix + '_addr'], a[prefix + '_zip']].filter(Boolean).join(' ')
      return s || '—'
    },
    fmtTime (t) {
      if (!t) return '—'
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>
