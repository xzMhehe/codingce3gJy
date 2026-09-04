<template>
  <div>
    <div class="module-content"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;成就大厅</div>
    <div class="module-content">
      成就:{{ data.achieve || 0 }} 成就点:{{ data.achieve || 0 }}<br>
      互动值:{{ data.interact || 0 }}<br>
      好友排名:{{ data.friend_rank }} 总排名:{{ data.total_rank }}<br>
    </div>
    <div class="module-title">【最新获得成就】</div>
    <div class="module-content"><span class="txt-fade">暂无新成就</span></div>

    <div class="module-title">【即将获得成就】</div>
    <div class="module-content">
      <img v-for="n in upcoming" :key="n.name" :src="'/static/picture/' + n.icon" :alt="n.name" :title="n.name">
    </div>

    <template v-for="cat in data.categories">
      <div class="module-title" :key="'t'+cat.name">【{{ cat.name }}】</div>
      <div class="module-content" :key="'c'+cat.name">
        <div v-for="(it,i) in cat.items" :key="it.name">{{ i+1 }}.{{ it.name }}.{{ it.percent.toFixed(2) }}%完成.<a href="javascript:;" @click="view(it)">查看</a></div>
      </div>
    </template>

    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Achieve',
  data () {
    return {
      data: { categories: [] }, okMsg: '',
      upcoming: [
        { name: '家园新手', icon: '101_u.gif' },
        { name: '魔法花童', icon: '202_u.jpg' },
        { name: '魔法义工', icon: '301_u.gif' },
        { name: '找到组织', icon: '401_u.gif' },
        { name: '单枪匹马', icon: '501_u.gif' }
      ]
    }
  },
  mounted () { api.get('/achieve').then(r => { if (r.code === 0) this.data = r.data }) },
  methods: {
    view (it) { this.okMsg = '「' + it.name + '」完成 ' + it.percent.toFixed(2) + '%' }
  }
}
</script>
