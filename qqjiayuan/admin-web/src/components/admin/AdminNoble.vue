<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="el-input__inner is-disabled" style="pointer-events:none;background:#f5f7fa">设置用户的超Q/蓝钻等级与成长值</span>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="号码" width="90" />
        <el-table-column label="昵称" min-width="140">
          <template slot-scope="{row}"><font :color="row.color || '#333'">{{ row.nickname }}</font></template>
        </el-table-column>
        <el-table-column label="超Q等级" width="140">
          <template slot-scope="{row}">
            <el-select v-model="row.noble" size="mini">
              <el-option :value="0" label="无" />
              <el-option :value="1" label="蓝钻" />
              <el-option :value="2" label="超Q" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="成长值" width="140">
          <template slot-scope="{row}"><el-input-number v-model="row.noble_exp" size="mini" :min="0" :max="999999" /></template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right" header-align="center">
          <template slot-scope="{row}"><el-button size="mini" type="primary" plain @click="save(row)">保存</el-button></template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="20" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminNoble',
  data () { return { list: [], total: 0, page: 1, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/nobles?page=' + this.page).then(r => { this.loading = false; if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } })
    },
    save (row) {
      api.put('/admin/nobles/' + row.id, { noble: row.noble, noble_exp: row.noble_exp }).then(r => { if (r.code === 0) this.$message.success('已保存') })
    }
  }
}
</script>
