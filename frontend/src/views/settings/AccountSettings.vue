<template>
  <div class="settings-section account-settings">
    <div class="section-title">
      <el-icon><User /></el-icon>
      <div>
        <h3>账号安全</h3>
        <p>管理你的账号信息、登录密码与登录状态</p>
      </div>
    </div>

    <div class="pref-group">
      <div class="pref-label"><span>账号信息</span></div>
      <div class="account-card">
        <div class="avatar" :style="{ background: avatarGradient }">{{ avatarChar }}</div>
        <div class="account-main">
          <div class="nick-row">
            <span class="nickname">{{ user?.nickname || user?.username || '未登录' }}</span>
            <el-button size="small" text type="primary" @click="openEditNick">
              <el-icon><EditPen /></el-icon> 编辑昵称
            </el-button>
          </div>
          <div class="username">@{{ user?.username }}</div>
        </div>
      </div>
    </div>

    <div class="pref-group">
      <div class="pref-label"><span>修改密码</span></div>
      <el-form label-position="top" class="pwd-form" :model="pwdForm" @submit.prevent>
        <el-form-item label="当前密码">
          <el-input v-model="pwdForm.oldPassword" style="width: 100%" type="password" show-password placeholder="请输入当前密码" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.newPassword" style="width: 100%" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="pwdForm.confirmPassword" style="width: 100%" type="password" show-password placeholder="再次输入新密码" />
        </el-form-item>
        <el-button type="primary" :loading="changingPwd" @click="submitPwd">修改密码</el-button>
      </el-form>
    </div>

    <div class="pref-group">
      <div class="pref-label"><span>登录状态</span></div>
      <div class="logout-row">
        <span class="logout-tip">退出后需要重新输入账号密码登录</span>
        <el-button type="danger" plain :icon="SwitchButton" @click="confirmLogout">退出登录</el-button>
      </div>
    </div>

    <el-dialog v-model="editNickVisible" title="编辑昵称" width="400px" append-to-body>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="昵称">
          <el-input v-model="nickForm.nickname" placeholder="请输入新昵称" maxlength="20" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editNickVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingNick" @click="submitNick">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SwitchButton } from '@element-plus/icons-vue'
import { getUser, getMe, updateProfile, changePassword, clearToken, setUser } from '@/api/auth'

const router = useRouter()
const user = ref(getUser())

const avatarChar = computed(() => {
  const n = user.value?.nickname || user.value?.username || '?'
  return n.charAt(0).toUpperCase()
})

const avatarGradient = computed(() => {
  const g = [
    'linear-gradient(135deg, var(--app-primary), color-mix(in srgb, var(--app-primary) 55%, #8b5cf6))',
    'linear-gradient(135deg, #06b6d4, #22d3ee)',
    'linear-gradient(135deg, #f59e0b, #f97316)',
    'linear-gradient(135deg, #10b981, #059669)',
    'linear-gradient(135deg, #ec4899, #f43f5e)'
  ]
  let h = 0
  for (const ch of avatarChar.value) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return g[h % g.length]
})

const editNickVisible = ref(false)
const nickForm = ref({ nickname: '' })
const savingNick = ref(false)

const openEditNick = () => {
  nickForm.value = { nickname: user.value?.nickname || user.value?.username || '' }
  editNickVisible.value = true
}

const submitNick = async () => {
  const nickname = nickForm.value.nickname.trim()
  if (!nickname) return ElMessage.warning('请输入昵称')
  savingNick.value = true
  try {
    // 后端返回结构为 { user: {...} }，直接使用返回值刷新本地用户信息
    const { data } = await updateProfile(nickname, null)
    const me = data?.user || data
    if (me) {
      user.value = me
      setUser(me)
    } else {
      user.value = { ...user.value, nickname }
      setUser(user.value)
    }
    ElMessage.success('昵称已更新')
    editNickVisible.value = false
  } catch (e) {
    ElMessage.error('更新失败：' + (e.response?.data?.error || e.message || ''))
  } finally {
    savingNick.value = false
  }
}

const pwdForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const changingPwd = ref(false)

const submitPwd = async () => {
  const { oldPassword, newPassword, confirmPassword } = pwdForm.value
  if (!oldPassword) return ElMessage.warning('请输入当前密码')
  if (newPassword.length < 6) return ElMessage.warning('新密码至少 6 位')
  if (newPassword !== confirmPassword) return ElMessage.warning('两次输入的新密码不一致')
  changingPwd.value = true
  try {
    await changePassword(oldPassword, newPassword)
    ElMessage.success('密码修改成功，请重新登录')
    pwdForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    clearToken()
    setUser(null)
    setTimeout(() => router.push('/login'), 800)
  } catch (e) {
    ElMessage.error('修改失败：' + (e.response?.data?.error || e.message || ''))
  } finally {
    changingPwd.value = false
  }
}

const confirmLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出当前账号吗？', '退出登录', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  clearToken()
  setUser(null)
  ElMessage.success('已退出登录')
  router.push('/login')
}

onMounted(async () => {
  try {
    // 后端返回结构为 { user: {...} }
    const { data } = await getMe()
    const me = data?.user || data
    if (me && me.username) {
      user.value = me
      setUser(me)
    }
  } catch {
    /* 保持本地用户信息 */
  }
})
</script>

<style scoped>
.section-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.section-title > .el-icon {
  font-size: 26px;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  padding: 10px;
  border-radius: 12px;
}
.section-title h3 {
  margin: 0;
  font-size: 18px;
}
.section-title p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.pref-group {
  margin-bottom: 28px;
}
.pref-label {
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 500;
}

.account-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
  border-radius: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 8px 24px rgba(31, 38, 135, 0.06);
}
.avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 26px;
  font-weight: 700;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--app-primary) 30%, transparent);
}
.nick-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.nickname {
  font-size: 18px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.username {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.pwd-form {
  max-width: 420px;
}
.logout-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 14px 18px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
}
.logout-tip {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

/* ===== 手机端适配 ===== */
@media (max-width: 768px) {
  .account-card {
    padding: 14px 16px;
    gap: 12px;
  }
  .avatar {
    width: 50px;
    height: 50px;
    font-size: 22px;
  }
  .nick-row {
    flex-wrap: wrap;
  }
  .logout-row {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    text-align: center;
  }
  .logout-row .el-button {
    width: 100%;
    margin-left: 0;
  }
}
</style>
