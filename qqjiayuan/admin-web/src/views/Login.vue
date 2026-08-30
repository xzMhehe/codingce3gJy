<template>
  <div class="login-page">
    <!-- 左侧品牌面板 -->
    <aside class="brand-panel">
      <div class="brand-deco">
        <div class="deco-orb deco-orb--1"></div>
        <div class="deco-orb deco-orb--2"></div>
        <div class="deco-grid"></div>
      </div>
      <div class="brand-content">
        <div class="brand-logo">
          <span>Q</span>
        </div>
        <h1 class="brand-name">3GQQ 家园社区</h1>
        <p class="brand-slogan">后台管理系统</p>
        <ul class="brand-features">
          <li><i class="el-icon-chat-dot-round"></i><span>社区内容一站式管理</span></li>
          <li><i class="el-icon-s-check"></i><span>角色权限精细管控</span></li>
          <li><i class="el-icon-data-board"></i><span>运营数据实时概览</span></li>
        </ul>
      </div>
      <p class="brand-copyright">3GQQ 家园 · 管理员工作台</p>
    </aside>

    <!-- 右侧登录表单 -->
    <main class="form-panel">
      <section class="form-box" aria-labelledby="login-title">
        <h2 id="login-title" class="form-title">欢迎回来</h2>
        <p class="form-sub">请使用管理员账号登录后台</p>

        <el-form class="login-form" @submit.native.prevent="doLogin">
          <el-form-item>
            <el-input
              v-model.trim="name"
              maxlength="50"
              placeholder="家园号码或昵称"
              prefix-icon="el-icon-user"
              autocomplete="username"
              aria-label="账号"
              @keyup.enter.native="doLogin"
            />
          </el-form-item>
          <el-form-item>
            <el-input
              v-model.trim="pass"
              type="password"
              maxlength="50"
              placeholder="登录密码"
              prefix-icon="el-icon-lock"
              show-password
              autocomplete="current-password"
              aria-label="密码"
              @keyup.enter.native="doLogin"
            />
          </el-form-item>

          <transition name="el-zoom-in-top">
            <el-alert
              v-if="err"
              :title="err"
              type="error"
              show-icon
              :closable="false"
              class="login-alert"
            />
          </transition>

          <el-button
            class="login-btn"
            type="primary"
            native-type="submit"
            :loading="loading"
            @click="doLogin"
          >
            {{ loading ? '正在验证…' : '登 录' }}
          </el-button>
        </el-form>

        <div class="form-footer">
          <p class="perm-hint">
            <i class="el-icon-lock"></i>
            仅限拥有「后台访问」权限的账号登录
          </p>
          <div class="perm-roles" aria-label="可登录角色">
            <span class="role-chip role-chip--danger">超级管理员</span>
            <span class="role-chip role-chip--warning">管理员</span>
            <span class="role-chip">版主</span>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'AdminLogin',
  data () {
    return { name: '', pass: '', err: '', loading: false }
  },
  methods: {
    getErrorMessage (error, fallback) {
      const responseData = error && error.response && error.response.data
      return (responseData && (responseData.msg || responseData.message)) ||
        (error && (error.msg || error.message)) || fallback
    },
    clearAdminSession () {
      localStorage.removeItem('jy_admin_token')
      localStorage.removeItem('jy_admin_user')
    },
    doLogin () {
      if (this.loading) return
      this.err = ''
      if (!this.name || !this.pass) {
        this.err = '请输入账号和密码'
        return
      }

      this.loading = true
      api.post('/auth/login', { name: this.name, password: this.pass })
        .then(r => {
          if (!r || r.code !== 0) {
            throw new Error((r && r.msg) || '账号或密码错误，请重试')
          }
          localStorage.setItem('jy_admin_token', r.data.token)
          return api.get('/auth/me')
        })
        .then(m => {
          if (!m || m.code !== 0) {
            this.clearAdminSession()
            throw new Error((m && m.msg) || '无法获取账号信息，请重试')
          }

          const perms = m.data && m.data.perms ? m.data.perms : []
          if (perms.indexOf('admin:access') < 0) {
            this.clearAdminSession()
            throw new Error('该账号没有后台访问权限，请用管理员账号登录')
          }

          localStorage.setItem('jy_admin_user', JSON.stringify(m.data))
          this.$router.push('/')
        })
        .catch(error => {
          this.err = this.getErrorMessage(error, '网络异常，请检查网络连接后重试')
        })
        .then(() => {
          this.loading = false
        })
    }
  }
}
</script>

<style scoped>
.login-page {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #f4f7fb;
}

/* ===== 左侧品牌面板 ===== */
.brand-panel {
  position: relative;
  flex: 0 0 44%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
  background: linear-gradient(150deg, #14222f 0%, #1d3448 48%, #254862 100%);
  color: #fff;
}
.brand-deco { position: absolute; inset: 0; pointer-events: none; }
.deco-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  animation: orbDrift 10s ease-in-out infinite alternate;
}
.deco-orb--1 {
  width: 460px; height: 460px; top: -18%; right: -22%;
  background: radial-gradient(circle, rgba(64,158,255,.32), transparent 68%);
}
.deco-orb--2 {
  width: 380px; height: 380px; bottom: -16%; left: -14%;
  background: radial-gradient(circle, rgba(115,103,240,.28), transparent 68%);
  animation-delay: -4s;
}
.deco-grid {
  position: absolute; inset: 0; opacity: .16;
  background-image:
    linear-gradient(rgba(255,255,255,.09) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255,255,255,.09) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: radial-gradient(ellipse at 30% 20%, #000 0%, transparent 72%);
  -webkit-mask-image: radial-gradient(ellipse at 30% 20%, #000 0%, transparent 72%);
}
@keyframes orbDrift {
  from { transform: translate(0, 0) scale(1); }
  to { transform: translate(26px, -22px) scale(1.07); }
}

.brand-content {
  position: relative;
  z-index: 1;
  padding: 0 9% 0 12%;
}
.brand-logo {
  width: 62px; height: 62px;
  display: grid; place-items: center;
  border-radius: 18px;
  background: linear-gradient(135deg, #409eff, #7367f0);
  box-shadow: 0 10px 28px rgba(64,158,255,.45);
  margin-bottom: 26px;
}
.brand-logo span { font-size: 32px; font-weight: 800; }
.brand-name {
  margin: 0 0 8px;
  font-size: 30px;
  font-weight: 700;
  letter-spacing: 1px;
}
.brand-slogan {
  margin: 0 0 38px;
  color: rgba(255,255,255,.62);
  font-size: 15px;
  letter-spacing: 3px;
}
.brand-features { margin: 0; padding: 0; list-style: none; }
.brand-features li {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
  color: rgba(255,255,255,.82);
  font-size: 14px;
}
.brand-features i {
  width: 34px; height: 34px;
  display: grid; place-items: center;
  border-radius: 10px;
  background: rgba(255,255,255,.1);
  border: 1px solid rgba(255,255,255,.14);
  font-size: 16px;
  color: #8fc6ff;
}
.brand-copyright {
  position: absolute;
  left: 12%;
  bottom: 28px;
  margin: 0;
  color: rgba(255,255,255,.38);
  font-size: 12px;
  letter-spacing: 1px;
}

/* ===== 右侧表单面板 ===== */
.form-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}
.form-box {
  width: min(100%, 380px);
  animation: formIn .5s cubic-bezier(.22,.8,.36,1) both;
}
@keyframes formIn {
  from { opacity: 0; transform: translateY(18px); }
  to { opacity: 1; transform: translateY(0); }
}
.form-title {
  margin: 0 0 8px;
  font-size: 26px;
  font-weight: 700;
  color: #1d2b3a;
}
.form-sub {
  margin: 0 0 32px;
  color: #8a99ab;
  font-size: 14px;
}

.login-form ::v-deep .el-input__inner {
  height: 46px;
  padding-left: 42px;
  border-radius: 10px;
  border-color: #e2e8f0;
  background: #f8fafc;
  font-size: 14px;
  transition: border-color .2s, box-shadow .2s, background .2s;
}
.login-form ::v-deep .el-input__inner:hover { border-color: #c0ccda; background: #fff; }
.login-form ::v-deep .el-input__inner:focus {
  border-color: #409eff;
  background: #fff;
  box-shadow: 0 0 0 3px rgba(64,158,255,.14);
}
.login-form ::v-deep .el-input__prefix { left: 13px; color: #9aa9bb; }
.login-form .el-form-item { margin-bottom: 18px; }

.login-alert { margin: 2px 0 16px; border-radius: 8px; }

.login-btn {
  width: 100%;
  height: 46px;
  margin-top: 6px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #409eff, #5a6ff0);
  box-shadow: 0 6px 18px rgba(64,158,255,.38);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 6px;
  transition: transform .2s, box-shadow .2s, filter .2s;
}
.login-btn:hover {
  filter: brightness(1.05);
  transform: translateY(-1px);
  box-shadow: 0 8px 24px rgba(64,158,255,.46);
}
.login-btn:active { transform: translateY(0); }

.form-footer {
  margin-top: 30px;
  padding-top: 22px;
  border-top: 1px solid #e9eef4;
  text-align: center;
}
.perm-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0 0 14px;
  color: #8a99ab;
  font-size: 12px;
}
.perm-hint i { color: #409eff; }
.perm-roles { display: flex; justify-content: center; gap: 8px; }
.role-chip {
  padding: 3px 12px;
  border-radius: 20px;
  background: #eef2f7;
  color: #6b7c8f;
  font-size: 12px;
}
.role-chip--danger { background: #fef0f0; color: #f56c6c; }
.role-chip--warning { background: #fdf6ec; color: #e6a23c; }

/* ===== 响应式：小屏只显示表单 ===== */
@media (max-width: 860px) {
  .brand-panel { display: none; }
  .login-page { background: linear-gradient(150deg, #14222f 0%, #1d3448 55%, #254862 100%); }
  .form-box {
    padding: 34px 28px;
    border-radius: 16px;
    background: rgba(255,255,255,.97);
    box-shadow: 0 18px 50px rgba(0,10,22,.4);
  }
}
</style>
