<template>
  <div>
    <!-- 参考：诺哈 column/18 活动栏目页（最新活动/长期活动/家园公告） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">家园</a>&gt;活动<br></div>

    <div class="module-title"><img src="/static/picture/active.gif" alt="活动">【最新活动】</div>
    <div class="module-content" v-if="latest.length">
      <div v-for="t in latest" :key="'l'+t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}({{ t.view_count }}阅)</a><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无最新活动</span></div>
    <div class="module-content"><a href="javascript:;" @click="$router.push('/activities/list')">更多活动&gt;&gt;</a></div>

    <div class="module-title"><img src="/static/picture/active.gif" alt="活动">【长期活动】</div>
    <div class="module-content" v-if="longterm.length">
      <div v-for="t in longterm" :key="'g'+t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无长期活动</span></div>
    <div class="module-content"><a href="javascript:;" @click="$router.push('/activities/list')">更多活动&gt;&gt;</a></div>

    <div class="module-title"><img src="/static/picture/notice.bmp" alt="公告">【家园公告】</div>
    <div class="module-content" v-if="notices.length">
      <div v-for="n in notices" :key="'n'+n.id">
        <a href="javascript:;" @click="$router.push('/notices')">{{ n.title }}</a><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无公告</span></div>
    <div class="module-content"><a href="javascript:;" @click="$router.push('/notices')">更多公告&gt;&gt;</a><br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Activities',
  data () {
    return { latest: [], longterm: [], notices: [] }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/activities/column').then(r => {
        if (r.code === 0) {
          this.latest = r.data.latest || []
          this.longterm = r.data.longterm || []
          this.notices = r.data.notices || []
        }
      })
    }
  }
}
</script>
