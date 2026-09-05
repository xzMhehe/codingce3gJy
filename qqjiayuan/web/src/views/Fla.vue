<template>
  <div>
    <div class="module-title">
      【捐款慈善福利】
    </div>
    
    <div class="module-content">
      <form @submit.prevent="submitDonation">
        <input type="hidden" name="act" value="ok"/>
        捐献慈善金额:
        <input type="text" name="issue" emptyok="true" maxlength="9" value="500000" /><br/>
        捐赠宣言：
        <textarea name="xuanyan" rows="3"></textarea><br/>
        <input type="submit" value="捐款" />
      </form>
      
      <em>
        <img src="/image/site/002.gif" alt=""/>1. 每日捐献慈善币第一名为可以接受全社区网友<span class="sqm4">膜拜</span><br/>
        <img src="/image/site/002.gif" alt=""/>2. 每人每日仅可捐献一次，价高者上位。<br/>
        <img src="/image/site/002.gif" alt=""/>3. 捐款时可以发布挑衅宣言（30字以内）。
      </em>
    </div>
    
    <div class="module-title">
      <a href="/home/my_home.html">家园</a>> <a href="/home/fl.html">福利院</a>>捐款慈善基金
    </div>
  </div>
</template>

<script>
export default {
  name: 'Fla',
  methods: {
    submitDonation() {
      // 这里可以添加捐款逻辑
      alert('捐款功能暂未实现');
    }
  }
}
</script>

<style scoped>
.module-title {
  padding: 5px;
  background: #f5f5f5;
  border-bottom: 1px solid #ddd;
  font-weight: bold;
}
.module-content {
  padding: 10px;
  background: #fff;
}
</style>