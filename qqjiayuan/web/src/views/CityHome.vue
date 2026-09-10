<template>
  <div>
    <!-- 面包屑 panav（诺哈：社区 > 同城） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;同城<br>
    </div>

    <!-- 同城客栈简介 -->
    <div class="module-content">{{ channel.description || '累了吗？来同城客栈透个气吧' }}<br></div>
    <div class="module-content">全国已有 <b>{{ totalProvinces }}</b> 个省份 / <b>{{ totalCities }}</b> 座城市客栈，当前 <b>{{ online }}</b> 位老乡在线<br></div>

    <!-- =按区号进入=（诺哈 city.txt） -->
    <div class="module-content">
      <form @submit.prevent="goCode">
        =按区号进入=<br>
        <input type="text" v-model.trim="code" maxlength="4" style="width:80px" placeholder="0722">
        <button class="btn small" type="submit">确定进入</button>
      </form>
    </div>

    <!-- =按城市进入= -->
    <div class="module-content">
      <form @submit.prevent="goName">
        =按城市进入=<br>
        <input type="text" v-model.trim="name" maxlength="8" style="width:80px" placeholder="随州">
        <button class="btn small" type="submit">确定进入</button>
      </form>
    </div>

    <!-- =按省份进入= -->
    <div class="module-content">
      <form @submit.prevent="goProvince">
        =按省份进入=<br>
        <select v-model.number="provId">
          <option v-for="p in provinces" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <button class="btn small" type="submit">确定进入</button>
      </form>
    </div>

    <!-- 所有省份（诺哈 province.asp 编号列表） -->
    <div class="module-title">【所有省份】</div>
    <div class="list">
      <div class="row" v-for="(p, i) in provinces" :key="'p'+p.id">
        {{ i }}.<a href="javascript:;" @click="$router.push('/tongcheng/province/'+p.id)">{{ p.name }}</a>({{ p.city_count }}城/{{ p.online }}人在线)<br>
      </div>
      <div v-if="!provinces.length" class="row">暂无省份，快去<a href="javascript:;" @click="goGongjian">共建同城</a>申请吧！<br></div>
    </div>
    <div class="module-content" v-if="gongjianId">
      你的城市还没有？<a href="javascript:;" @click="goGongjian">共建同城</a>等你来申请！<br>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;同城<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'CityHome',
  data () {
    return { provinces: [], channel: {}, totalProvinces: 0, totalCities: 0, online: 0, gongjianId: 0, code: '', name: '', provId: 0 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/tongcheng').then(r => {
        if (r.code !== 0) return
        this.provinces = r.data.provinces || []
        this.channel = r.data.channel || {}
        this.totalProvinces = r.data.total_provinces
        this.totalCities = r.data.total_cities
        this.online = r.data.online
        this.gongjianId = r.data.gongjian_id
        if (this.provinces.length && !this.provId) this.provId = this.provinces[0].id
      })
    },
    lookup (params) {
      api.get('/tongcheng/lookup', { params }).then(r => {
        if (r.code === 0) {
          this.$router.push('/tongcheng/city/' + r.data.id)
        } else {
          alert(r.msg || '没有找到这个城市')
        }
      })
    },
    goCode () { if (this.code) this.lookup({ code: this.code }) },
    goName () { if (this.name) this.lookup({ name: this.name }) },
    goProvince () { if (this.provId) this.$router.push('/tongcheng/province/' + this.provId) },
    goGongjian () { if (this.gongjianId) this.$router.push('/board/' + this.gongjianId) }
  }
}
</script>
