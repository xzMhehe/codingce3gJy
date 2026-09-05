<template>
  <div>
    <el-card shadow="never" class="box">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="家族" min-width="140" />
        <el-table-column prop="owner" label="族长" width="120" />
        <el-table-column prop="members" label="成员" width="80" header-align="center" />
        <el-table-column prop="battle_score" label="积分" width="90" header-align="center" />
        <el-table-column prop="announcement" label="公告" min-width="180" show-overflow-tooltip />
        <el-table-column label="操作" width="240" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="editAnn(row)">改公告</el-button>
            <el-button size="mini" type="danger" plain @click="toggle(row)">解散</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog title="修改家族公告" :visible.sync="dlg" width="520px" :close-on-click-modal="false">
      <el-input type="textarea" v-model="ann" :rows="5" maxlength="500" />
      <div slot="footer"><el-button @click="dlg = false">取 消</el-button><el-button type="primary" @click="saveAnn">保 存</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminFamilies',
  data () { return { list: [], total: 0, page: 1, size: 10, loading: false, dlg: false, ann: '', current: 0 } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/families', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list || []; this.total = r.data.total || 0 } else this.$message.error(r.msg)
      })
    },
    editAnn (row) { this.current = row.id; this.ann = row.announcement; this.dlg = true },
    saveAnn () { api.put('/admin/families/' + this.current + '/ann', { announcement: this.ann }).then(r => { if (r.code === 0) { this.dlg = false; this.load() } }) },
    toggle (row) {
      this.$confirm('确认解散该家族吗？', '提示').then(() => {
        api.put('/admin/families/' + row.id + '/status', { status: 0 }).then(() => { this.$message.success('已解散'); this.load() })
      }).catch(() => {})
    }
  }
}
</script>
