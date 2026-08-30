<template>
  <div>
    心情动态<br />

    <!-- 心情列表 -->
    <div class="list" v-if="moods.length">
      <div class="row" v-for="(m, i) in moods" :key="m.id">
        {{ i + 1 }}.{{ m.content }} ({{ fmtTime(m.created_at) }}) [<a href="javascript:;" @click="delMood(m.id)">删除</a>]<br />
      </div>
      <div class="row"></div>
      (第<b>{{ page }}</b>/{{ pageTotal }}页/共{{ total }}条记录)<br />
    </div>
    <div class="text" v-else>
      目前还没有发表过心情，快来发表一个吧！<br />
    </div>
    ----------<br />

    <!-- 发表心情表单 -->
    <form @submit.prevent="addMood">
      <input type="text" v-model.trim="content" maxlength="120" value="" style="width:70%" /><br />
      <input type="submit" value="发表心情" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:2px 5px">{{ tip }}</p>
    ----------<br />

    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/home')">我的家园</a>&gt;&gt;心情<br />
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MyMood',
  data () {
    return { moods: [], page: 1, pageTotal: 1, total: 0, content: '', tip: '' }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/moods', { params: { page: this.page, size: 10 } }).then(r => {
        if (r.code === 0) {
          this.moods = r.data.list || []
          this.total = r.data.total || 0
          this.pageTotal = Math.ceil(this.total / (r.data.size || 10)) || 1
        }
      })
    },
    addMood () {
      if (!this.content) {
        this.tip = '请先输入心情内容'
        return
      }
      api.post('/moods', { content: this.content }).then(r => {
        if (r.code === 0) {
          this.content = ''
          this.tip = ''
          this.page = 1
          this.load()
        } else {
          this.tip = r.msg || '发表失败，请稍后重试'
        }
      }).catch(() => { this.tip = '网络异常，请稍后重试' })
    },
    delMood (id) {
      api.delete('/moods/' + id).then(r => {
        if (r.code === 0) this.load()
      })
    },
    fmtTime (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>
