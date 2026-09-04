<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="txt-fade">设置广场 T台秀（膜拜）上榜用户</span>
        <div class="grow" />
      </div>
      <el-form inline>
        <el-form-item label="当前上榜">
          <el-tag type="warning">{{ cur.nickname || '—' }}（{{ cur.id || '自动' }} · 经验 {{ cur.exp || 0 }}）</el-tag>
        </el-form-item>
        <el-form-item label="指定号码">
          <el-input v-model="uid" placeholder="输入用户号码" style="width:180px" />
          <el-button type="primary" @click="setTtou">设为上榜</el-button>
          <el-button type="danger" plain @click="clearTtou">移除指定</el-button>
        </el-form-item>
      </el-form>
      <div class="module-content">留空则自动取经验最高用户；指定号码后固定该用户上榜。</div>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminTtou',
  data () { return { cur: {}, uid: '' } },
  mounted () { this.load() },
  methods: {
    load () { api.get('/admin/ttou').then(r => { if (r.code === 0) this.cur = r.data }) },
    setTtou () {
      if (!this.uid) { this.$message.warning('请填写用户号码'); return }
      api.put('/admin/ttou', { user_id: Number(this.uid) }).then(r => { if (r.code === 0) { this.$message.success('已设置'); this.load() } else this.$message.error(r.msg) })
    },
    clearTtou () {
      api.delete('/admin/ttou').then(r => { if (r.code === 0) { this.$message.success('已移除指定，恢复自动'); this.load() } else this.$message.error(r.msg) })
    }
  }
}
</script>
