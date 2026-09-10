<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/book')">书城</a>&gt;我的书架</div>
    <div class="module-title">我的书架<br></div>
    <div class="module-content">
      <span v-for="(b, i) in books" :key="b.id">{{ i + 1 }}.《<a href="javascript:;" @click="$router.push('/book/' + b.id)">{{ b.title }}</a>》
        <span class="txt-fade">{{ b.author }}</span>
        [<a href="javascript:;" @click="remove(b)">移出</a>]<br></span>
      <span v-if="!books.length" class="empty">书架空空的，快去书城加几本吧。[<a href="javascript:;" @click="$router.push('/book')">去书城</a>]<br></span>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shelf',
  data () { return { books: [] } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/me/shelf').then(r => { if (r.code === 0) this.books = r.data })
    },
    remove (b) {
      api.delete('/books/' + b.id + '/shelf').then(r => {
        if (r.code === 0) this.load()
      })
    }
  }
}
</script>
