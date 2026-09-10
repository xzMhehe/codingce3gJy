<template>
  <div>
    <!-- 面包屑（诺哈 city.asp：社区>同城>省份>城市） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng/province/'+province.id)">{{ province.name }}</a>&gt;{{ city.name }}<br>
    </div>

    <!-- 【城市名】 在线老乡.老乡聊天 -->
    <div class="module-title">【{{ city.name }}】</div>
    <div class="list">
      <div class="row">
        <a href="javascript:;" @click="$router.push('/tongcheng/city/'+city.id+'/online')">在线老乡</a>.<a href="javascript:;" @click="$router.push({ path: '/chat', query: { board_id: city.id } })">老乡聊天</a><br>
      </div>
    </div>

    <!-- 【议事论坛】 精华 TOP4 -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/board/'+city.id)">议事论坛</a>】</div>
    <div class="list">
      <div class="row" v-for="(t, i) in fineThreads" :key="'t'+t.id">
        {{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title.slice(0, 13) }}</a><br>
      </div>
      <div v-if="!fineThreads.length" class="row">暂无精华帖，<a href="javascript:;" @click="$router.push('/board/'+city.id)">去议事论坛看看</a><br></div>
    </div>

    <!-- 【同城管理】（诺哈 wap_manage） -->
    <div class="module-title">【同城管理】</div>
    <div class="list">
      <div class="row" v-for="m in managers" :key="'m'+m.id">
        {{ m.title }}:<a href="javascript:;" @click="$router.push('/user/'+(m.user?m.user.id:0))"><font :color="m.user?m.user.color:''">{{ m.user?m.user.nickname:'路人' }}</font></a><br>
      </div>
      <div v-if="!managers.length" class="row">虚位以待，等待城市管理员上任～<br></div>
    </div>

    <!-- 【同城资料】 人气/在线/创建人 -->
    <div class="module-title">【同城资料】</div>
    <div class="list">
      <div class="row">人气:{{ city.click }}<br></div>
      <div class="row">在线:{{ online }}人<br></div>
      <div class="row">创建人:<a v-if="creator" href="javascript:;" @click="$router.push('/user/'+creator.id)"><font :color="creator.color">{{ creator.nickname }}</font></a><template v-else>站长</template><br></div>
      <div class="row" v-if="city.city_code">区号:{{ city.city_code }}<br></div>
    </div>

    <!-- 操作栏 -->
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/board/'+city.id)">进入议事论坛</a>.<a href="javascript:;" @click="$router.push({ path: '/chat', query: { board_id: city.id } })">老乡聊天室</a><br>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng/province/'+province.id)">{{ province.name }}</a>&gt;{{ city.name }}<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'City',
  data () {
    return { city: {}, province: {}, creator: null, online: 0, fineThreads: [], managers: [] }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/tongcheng/city/' + id).then(r => {
        if (r.code !== 0) {
          alert(r.msg || '该城市不存在')
          this.$router.push('/tongcheng')
          return
        }
        this.city = r.data.city || {}
        this.province = r.data.province || {}
        this.creator = r.data.creator
        this.online = r.data.online
        this.fineThreads = r.data.fine_threads || []
        this.managers = r.data.managers || []
        document.title = this.city.name || '同城客栈'
      })
    }
  }
}
</script>
