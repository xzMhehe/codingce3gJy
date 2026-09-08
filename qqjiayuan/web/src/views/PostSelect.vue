<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;发帖&gt;选择板块
    </div>
    <div class="module-title">请选择要发帖的板块</div>
    <template v-for="ch in channels">
      <div class="module-title" :key="'t'+ch.id">【{{ ch.name }}】</div>
      <div class="module-content" :key="'c'+ch.id">
        <!-- 有分类：每个分类一行，未分类的放最后 -->
        <template v-if="ch.categories && ch.categories.length">
          <div v-for="cat in ch.categories" :key="'cat'+cat.id" style="margin-bottom:6px">
            <b>{{ cat.name }}：</b>
            <span v-if="cat.boards && cat.boards.length">
              <span v-for="(s,i) in cat.boards" :key="'cb'+s.id">
                <a href="javascript:;" @click="pick(s)">{{ s.name }}</a><template v-if="i < cat.boards.length-1"> . </template>
              </span>
            </span>
            <span class="txt-fade" v-else>（暂无板块）</span>
          </div>
          <div v-if="ch.children && ch.children.length" style="margin-top:4px">
            <b>未分类：</b>
            <span v-for="(s,i) in ch.children" :key="'rest'+s.id">
              <a href="javascript:;" @click="pick(s)">{{ s.name }}</a><template v-if="i < ch.children.length-1"> . </template>
            </span>
          </div>
        </template>
        <!-- 无分类 -->
        <template v-else>
          <span v-for="(s,i) in ch.children" :key="'l'+s.id">
            <a href="javascript:;" @click="pick(s)">{{ s.name }}</a><template v-if="i < ch.children.length-1"> . </template>
          </span>
          <span class="txt-fade" v-if="!ch.children || !ch.children.length">（暂无板块）</span>
        </template>
      </div>
    </template>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;发帖&gt;选择板块
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'PostSelect',
  data () { return { channels: [] } },
  mounted () {
    api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
  },
  methods: {
    pick (b) {
      this.$router.push('/post/' + b.id)
    }
  }
}
</script>