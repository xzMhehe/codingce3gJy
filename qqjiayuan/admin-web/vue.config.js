const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  lintOnSave: false,
  publicPath: '/admin-ui/',
  productionSourceMap: false,
  chainWebpack: config => {
    // 管理端各页面已按菜单懒加载（menu.js 里全部是 import()），
    // 但 vue-cli 默认会给所有异步 chunk 注入 <link rel="prefetch">，
    // 登录页打开时仍会把近百个页面 chunk 全部后台下载一遍 → 首屏很慢。
    // 这里去掉 prefetch，只有真正进入某个菜单时才加载对应 chunk。
    config.plugins.delete('prefetch')
  },
  devServer: {
    port: 8001,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      },
      '/static': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      }
    }
  }
})
