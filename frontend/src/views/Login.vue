<template>
  <div class="login-page">
    <div class="login-card">
      <div class="brand">
        <div class="brand-icon">
          <el-icon :size="26"><Collection /></el-icon>
        </div>
        <h1>墨阅</h1>
        <p>记录每一次阅读，收藏每一段时光</p>
      </div>

      <!-- 登录 -->
      <div class="login-form">
      <el-form v-if="mode === 'login'" ref="loginFormRef" :model="loginForm" :rules="loginRules" size="large" @submit.prevent>
        <el-form-item prop="username">
          <el-input v-model="loginForm.username" placeholder="用户名" :prefix-icon="User" clearable @keyup.enter="onSubmit" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="loginForm.password" type="password" placeholder="密码" :prefix-icon="Lock" show-password @keyup.enter="onSubmit" />
        </el-form-item>
        <el-button class="submit-btn" type="primary" :loading="loading" @click="onSubmit">
          登 录
        </el-button>
      </el-form>

      <!-- 注册 -->
      <el-form v-else ref="regFormRef" :model="regForm" :rules="regRules" size="large" @submit.prevent>
        <el-form-item prop="username">
          <el-input v-model="regForm.username" placeholder="用户名（2-32 个字符）" :prefix-icon="User" clearable />
        </el-form-item>
        <el-form-item prop="nickname">
          <el-input v-model="regForm.nickname" placeholder="昵称（可选）" :prefix-icon="Postcard" clearable />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="regForm.password" type="password" placeholder="密码（至少 6 位）" :prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item prop="confirmPassword">
          <el-input v-model="regForm.confirmPassword" type="password" placeholder="确认密码" :prefix-icon="Lock" show-password @keyup.enter="onSubmit" />
        </el-form-item>
        <el-button class="submit-btn" type="primary" :loading="loading" @click="onSubmit">
          注 册
        </el-button>
      </el-form>
      </div>

      <div class="switch-row">
        <template v-if="mode === 'login'">
          <span>还没有账号？</span>
          <el-link type="primary" :underline="false" @click="switchMode('register')">立即注册</el-link>
        </template>
        <template v-else>
          <span>已有账号？</span>
          <el-link type="primary" :underline="false" @click="switchMode('login')">去登录</el-link>
        </template>
      </div>
    </div>

    <p class="footer-tip">首次注册的用户将自动接管本机已有书库数据</p>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Postcard } from '@element-plus/icons-vue'
import { login, register, setToken, setUser } from '@/api/auth'

const route = useRoute()
const router = useRouter()

const mode = ref('login')
const loading = ref(false)
const loginFormRef = ref(null)
const regFormRef = ref(null)

const loginForm = reactive({ username: '', password: '' })
const regForm = reactive({ username: '', nickname: '', password: '', confirmPassword: '' })

const loginRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const regRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '用户名长度为 2-32 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== regForm.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const switchMode = (m) => {
  mode.value = m
}

const handleSuccess = (res) => {
  setToken(res.token)
  setUser(res.user)
  ElMessage.success(mode.value === 'login' ? '登录成功' : '注册成功')
  const redirect = route.query.redirect || '/'
  router.replace(redirect)
}

const onSubmit = async () => {
  const formRef = mode.value === 'login' ? loginFormRef : regFormRef
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    if (mode.value === 'login') {
      const { data } = await login(loginForm.username.trim(), loginForm.password)
      handleSuccess(data)
    } else {
      const { data } = await register(
        regForm.username.trim(),
        regForm.password,
        regForm.nickname.trim()
      )
      handleSuccess(data)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '操作失败，请稍后重试')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, #eef2ff 0%, #f0f9ff 50%, #ecfeff 100%);
  position: relative;
  overflow: hidden;
}

.login-page::before,
.login-page::after {
  content: '';
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.5;
  pointer-events: none;
}

.login-page::before {
  width: 380px;
  height: 380px;
  background: linear-gradient(135deg, #818cf8, #22d3ee);
  top: -120px;
  left: -100px;
}

.login-page::after {
  width: 320px;
  height: 320px;
  background: linear-gradient(135deg, #f472b6, #fbbf24);
  bottom: -110px;
  right: -90px;
}

.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 400px;
  padding: 40px 36px 28px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(22px);
  -webkit-backdrop-filter: blur(22px);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.85);
  box-shadow: 0 24px 60px rgba(31, 38, 135, 0.18);
}

.brand {
  text-align: center;
  margin-bottom: 28px;
}

.brand-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 14px;
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--app-gradient);
  box-shadow: 0 12px 28px color-mix(in srgb, var(--app-primary) 40%, transparent);
}

.brand h1 {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
}

.brand p {
  margin: 0;
  font-size: 13px;
  color: #94a3b8;
}

.submit-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 6px;
  border-radius: 12px;
  background: var(--app-gradient);
  border: none;
  box-shadow: 0 10px 24px color-mix(in srgb, var(--app-primary) 35%, transparent);
}

.submit-btn:hover {
  opacity: 0.92;
}

.switch-row {
  margin-top: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 13px;
  color: #64748b;
}

.footer-tip {
  position: relative;
  z-index: 1;
  margin-top: 18px;
  font-size: 12px;
  color: #94a3b8;
  text-align: center;
}

:deep(.el-input__wrapper) {
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.9);
}

/* === 移动端优化：整页式登录、输入框居中且缩短 === */
@media (max-width: 768px) {
  /* 锁滚动：禁止整页上下滚动 */
  html, body {
    height: 100%;
    overflow: hidden;
    overscroll-behavior: none;
  }
  .login-page {
    height: 100vh;
    height: 100dvh;
    padding: 0;
    justify-content: stretch;
    overflow: hidden;
  }
  .login-page::before,
  .login-page::after {
    pointer-events: none;
  }
  .login-page::before {
    width: 260px;
    height: 260px;
    top: -110px;
    left: -80px;
    filter: blur(60px);
    opacity: 0.45;
  }
  .login-page::after {
    width: 230px;
    height: 230px;
    bottom: -90px;
    right: -70px;
    filter: blur(60px);
    opacity: 0.45;
  }
  /* 卡片铺满整页 */
  .login-card {
    width: 100%;
    max-width: 100%;
    height: 100%;
    min-height: 100vh;
    min-height: 100dvh;
    margin: 0;
    padding: 32px 24px 56px;
    border-radius: 0;
    border: none;
    background: rgba(255, 255, 255, 0.82);
    box-shadow: none;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    overflow-y: auto;
  }
  /* 表单 wrapper 居中且限制宽度 */
  .login-form {
    width: 100%;
    max-width: 320px;
    margin: 0 auto;
  }
  /* 强制 el-form / 输入框 / 按钮跟随 wrapper 宽度 */
  :deep(.el-form) {
    width: 100%;
  }
  :deep(.el-form-item) {
    margin-bottom: 16px;
  }
  :deep(.el-input__wrapper) {
    padding: 3px 12px;
  }
  .submit-btn {
    width: 100%;
    height: 46px;
    font-size: 15px;
  }
  /* 切换行与底部提示也跟随居中 */
  .switch-row {
    width: 100%;
    max-width: 320px;
    margin: 18px auto 0;
    justify-content: center;
    font-size: 13px;
  }
  .footer-tip {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 12px;
    margin: 0;
    padding: 0 20px;
    font-size: 11px;
    line-height: 1.7;
    text-align: center;
  }
  .brand {
    margin-bottom: 28px;
    text-align: center;
  }
  .brand-icon {
    width: 58px;
    height: 58px;
    border-radius: 16px;
  }
  .brand h1 {
    font-size: 22px;
    margin-bottom: 4px;
  }
  .brand p {
    font-size: 12px;
  }
}
</style>
