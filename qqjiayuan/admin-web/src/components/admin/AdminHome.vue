<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <span class="txt-fade">设置用户的 家园等级/活跃天数/成就点/城市/好友策略，点「资料」编辑会员档案</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="id" label="号码" width="100" header-align="center" />
        <el-table-column label="昵称" min-width="130">
          <template slot-scope="{row}"><b><font :color="row.color || '#333'">{{ row.nickname }}</font></b></template>
        </el-table-column>
        <el-table-column label="家园等级" min-width="110" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.level" size="small" :min="1" :max="99" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="活跃天数" min-width="120" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.active_days" size="small" :min="0" :max="99999" :precision="1" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="成就点" min-width="110" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.achieve" size="small" :min="0" :max="999999" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="城市" min-width="120">
          <template slot-scope="{row}"><el-input v-model="row.city" size="small" maxlength="30" /></template>
        </el-table-column>
        <el-table-column label="好友策略" min-width="110" header-align="center">
          <template slot-scope="{row}">
            <el-select v-model="row.friend_policy" size="small">
              <el-option :value="0" label="允许" />
              <el-option :value="1" label="需要验证" />
              <el-option :value="2" label="拒绝" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" plain @click="openProfile(row)">资料</el-button>
            <el-button size="mini" type="primary" plain @click="save(row)">保存</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="10" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 会员资料编辑（对齐诺哈 admin/user 资料编辑：性别/年龄/生日类型/阳历阴历/签名/简介/在线时长/每页帖数/消费额） -->
    <el-dialog :title="'会员资料：' + (pForm.username || '')" :visible.sync="pDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="90px" size="small">
        <el-form-item label="性别">
          <el-radio-group v-model="pForm.gender">
            <el-radio :label="1">男</el-radio>
            <el-radio :label="2">女</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="年龄">
          <el-input-number v-model="pForm.age" :min="0" :max="150" />
        </el-form-item>
        <el-form-item label="生日类型">
          <el-radio-group v-model="pForm.birth_type">
            <el-radio :label="1">阳历</el-radio>
            <el-radio :label="0">阴历</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="阳历生日">
          <el-input-number v-model="pForm.birth_year" :min="0" :max="9999" controls-position="right" style="width:100px" placeholder="年" />
          <el-input-number v-model="pForm.birth_month" :min="0" :max="12" controls-position="right" style="width:90px" />
          <el-input-number v-model="pForm.birth_day" :min="0" :max="31" controls-position="right" style="width:90px" />
        </el-form-item>
        <el-form-item label="阴历生日">
          <el-input v-model.trim="pForm.lunar" maxlength="20" style="width:220px" placeholder="如：腊月初八" />
        </el-form-item>
        <el-form-item label="签名">
          <el-input v-model.trim="pForm.signature" maxlength="120" style="width:340px" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model.trim="pForm.introduction" maxlength="200" style="width:340px" />
        </el-form-item>
        <el-form-item label="在线时长">
          <el-input-number v-model="pForm.hours" :min="0" :max="9999999" controls-position="right" />
          <span class="txt-fade">分钟</span>
        </el-form-item>
        <el-form-item label="每页帖数">
          <el-select v-model="pForm.per_page" style="width:120px">
            <el-option v-for="n in [5,10,15,20]" :key="n" :value="n" :label="n + ' 条'" />
          </el-select>
        </el-form-item>
        <el-form-item label="累计消费">
          <el-input-number v-model="pForm.paid" :min="0" :max="99999999" controls-position="right" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="pDlg = false">取 消</el-button>
        <el-button type="primary" @click="saveProfile">保存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminHome',
  data () {
    return {
      list: [], total: 0, page: 1, word: '', loading: false,
      pDlg: false, pForm: {}
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/users?page=' + this.page + (this.word ? '&word=' + encodeURIComponent(this.word) : '')).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = (r.data.list || []).map(u => ({ ...u, friend_policy: u.friend_policy || 0 }))
          this.total = r.data.total
        }
      })
    },
    search () { this.page = 1; this.load() },
    save (row) {
      api.put('/admin/users/' + row.id + '/home', {
        level: row.level, active_days: row.active_days, achieve: row.achieve, city: row.city,
        friend_policy: row.friend_policy
      }).then(r => {
        if (r.code === 0) this.$message.success('已保存'); else this.$message.error(r.msg)
      })
    },
    openProfile (row) {
      this.pForm = {
        id: row.id, username: row.username,
        gender: row.gender || 1, age: row.age || 0,
        birth_type: row.birth_type === 0 ? 0 : 1,
        birth_year: row.birth_year || 0, birth_month: row.birth_month || 0, birth_day: row.birth_day || 0,
        lunar: row.lunar || '', signature: row.signature || '', introduction: row.introduction || '',
        hours: row.hours || 0, paid: row.paid || 0,
        per_page: parseInt((row.config || '10,1200,1500,1200,0').split(',')[0]) || 10
      }
      this.pDlg = true
    },
    saveProfile () {
      api.put('/admin/users/' + this.pForm.id + '/home', this.pForm).then(r => {
        if (r.code === 0) { this.$message.success('资料已保存'); this.pDlg = false; this.load() }
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>