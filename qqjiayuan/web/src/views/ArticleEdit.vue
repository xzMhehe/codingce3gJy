<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/articles')">文章</a>&gt;发表文章</div>
    <div class="name">发表文章<br></div>
    <form @submit.prevent="submit">
      标题:<input type="text" v-model.trim="form.title" maxlength="60" size="20" /><br>
      分类:
      <select v-model.number="form.cat_id">
        <option :value="0">请选择分类</option>
        <option v-for="c in cats" :key="c.id" :value="c.id">{{ c.name }}</option>
      </select><br>
      来源:<input type="text" v-model.trim="form.source" maxlength="50" size="20" placeholder="选填" /><br>
      正文:<br>
      <textarea v-model.trim="form.content" rows="6" maxlength="20000"></textarea><br>
      <span class="txt-fade">正文至少 10 字，发表后可获得经验与 G币奖励。</span><br>
      <input type="submit" value="发表" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>
    <a href="javascript:;" @click="$router.push('/articles')">返回文章列表</a><br>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ArticleEdit',
  data () { return { cats: [], form: { title: '', cat_id: 0, source: '', content: '' }, tip: '' } },
  mounted () { api.get('/articles/categories').then(r => { if (r.code === 0) this.cats = r.data || [] }) },
  methods: {
    submit () {
      if (!this.form.title || !this.form.content) { this.tip = '标题和正文不能为空'; return }
      api.post('/articles', this.form).then(r => {
        if (r.code === 0) this.$router.push('/articles/' + r.data.id)
        else this.tip = r.msg || '发表失败'
      }).catch(() => { this.tip = '发表失败，请稍后再试' })
    }
  }
}
</script>