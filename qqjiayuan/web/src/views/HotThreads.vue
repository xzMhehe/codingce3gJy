<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;今日热帖</div>
    <div class="module-title">今日热帖 TOP 20</div>
    <div class="list">
      <div class="row" v-for="(t,i) in threads" :key="t.id">
        <b>{{ i+1 }}.</b>
        <template v-if="t.is_head">[头条]</template><template v-if="t.is_top">【顶】</template><template v-if="t.is_fine">【精】</template><template v-if="t.is_lock">[锁]</template><template v-if="t.type===3">[投票]</template>
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
        (<a href="javascript:;" @click="$router.push('/board/'+(t.board?t.board.id:''))">{{ t.board ? t.board.name : '?' }}</a> · {{ t.reply_count }}回/{{ t.view_count }}阅)<br>
      </div>
    </div>
    <div v-if="!threads.length" class="empty">暂无热帖</div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;今日热帖</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'HotThreads',
  data () { return { threads: [] } },
  mounted () {
    api.get('/threads/hot').then(r => { if (r.code === 0) this.threads = r.data || [] })
  }
}
</script>