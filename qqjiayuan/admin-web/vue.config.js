const { defineConfig } = require('@vue/cli-service')
const webpack = require('webpack')

// ★ 2026-10-27 构建版本号：把构建时刻注入到前端，侧边栏底部显示。
//   用途：管理端「改了看不到」时，一眼确认浏览器里跑的是不是刚构建的包
//   （旧包显示的版本号还是上一次的时间）。不改的话只能靠 F12 看 JS hash，沟通成本高。
const BUILD_TIME = new Date().toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-')

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
    config.plugin('define-build-time').use(webpack.DefinePlugin, [{
      __BUILD_TIME__: JSON.stringify(BUILD_TIME)
    }])
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
