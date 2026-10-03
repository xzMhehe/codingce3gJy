<template>
  <div class="panel">
    <div class="panel-title">公告</div>
    <!-- ★ 用户要求「公告也变成分页，下一页上一页那种」→ 与军情三区同一套 .ezfy-pager 写法
         （默认每页 5 条，见 noticeSize）。 -->
    <div class="old-line" v-for="n in ezfy.noticePaged" :key="'nn' + n.id">
      <span v-if="n.is_top" class="red">[置顶]</span>
      <a href="javascript:;" @click="ezfy.openNotice(n)">{{ n.title }}</a>
      <!-- ★ 用户要求：公告标题后展示发布时间（年-月-日）。CreatedAt(time.Time) JSON 序列化为
           "2026-09-23T11:11:28+08:00"，前端只取 "年-月-日" 并加 [] 色弱化。 -->
      <span class="gray">[{{ ezfy.fmtDate(n.created_at) }}]</span>
    </div>
    <div class="old-line" v-if="!ezfy.notices.length">(暂无公告)</div>
    <div class="ezfy-pager" v-if="ezfy.notices.length > ezfy.noticeSize">
      <a href="javascript:;" :class="{ gray: ezfy.noticePage <= 1 }" @click="ezfy.sectionPagerGo('notice', -1)">上一页</a>
      <span class="gray">第 {{ ezfy.noticePage }}/{{ ezfy.noticeTotalPages }} 页（共 {{ ezfy.notices.length }} 条）</span>
      <a href="javascript:;" :class="{ gray: ezfy.noticePage >= ezfy.noticeTotalPages }" @click="ezfy.sectionPagerGo('notice', 1)">下一页</a>
    </div>
    <template v-if="ezfy.curNotice">
      <div class="panel-title">{{ ezfy.curNotice.title }}</div>
      <div class="old-line">{{ ezfy.curNotice.content }}</div>
    </template>
    <a href="javascript:;" @click="ezfy.go('back')">[返回]</a> <a href="javascript:;" @click="ezfy.go('home')">[返回首页]</a>
  </div>
</template>

<script>
// ★ 2026-10-03 模块化试点：公告页模板独立成组件。
//   数据/方法仍在 Ezfy.vue 外壳，通过 inject 拿回外壳实例访问（ezfy.xxx）。
export default {
  name: 'EzfyNotices',
  inject: ['ezfy']
}
</script>
