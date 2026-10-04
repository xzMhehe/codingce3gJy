const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  lintOnSave: false,
  // ★ 2026-10-04 二战独立入口：dist/ezfy.html 只挂 Ezfy.vue，不引家园全局 CSS/壳，
  //   刷新不再出现「家园影子」，且构建产物自包含（dist/js|css 的 ezfy 相关 chunk），
  //   将来可把 dist 单独拎出去做独立部署。
  pages: {
    index: {
      entry: 'src/main.js',
      template: 'public/index.html',
      filename: 'index.html'
    },
    ezfy: {
      entry: 'src/ezfy-main.js',
      template: 'public/ezfy.html',
      filename: 'ezfy.html'
    }
  },
  devServer: {
    port: 8000,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      },
      '/admin-ui': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      }
    }
  }
})
