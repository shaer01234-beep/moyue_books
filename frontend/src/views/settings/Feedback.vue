<template>
  <div class="settings-section feedback-section">
    <div class="section-title">
      <el-icon><ChatDotRound /></el-icon>
      <div>
        <h3>问题反馈</h3>
        <p>遇到问题或想提建议？告诉我们，我们会尽快回复你</p>
      </div>
    </div>

    <div class="fb-channels">
      <!-- QQ 交流群 -->
      <div class="fb-card">
        <div class="fb-card-head">
          <el-icon class="fb-icon qq"><ChatLineSquare /></el-icon>
          <div>
            <h4>QQ 交流群</h4>
            <p>扫码进群，和其他书友一起交流</p>
          </div>
        </div>
        <div class="fb-body fb-qq-body">
          <div class="fb-qr-wrap">
            <el-image
              :src="qqGroupImg"
              class="fb-qr"
              fit="cover"
              :preview-src-list="[qqGroupImg]"
              preview-teleported
            />
            <p class="fb-qr-tip">点击图片放大后长按，扫码即可加群</p>
          </div>
          <div v-if="isUrl(info.qq_group)" class="fb-qq-row">
            <el-button
              type="primary"
              plain
              :icon="ChatLineSquare"
              @click="openQQGroup"
            >加入交流群</el-button>
          </div>
          <div v-else-if="info.qq_group" class="fb-qq-row">
            <span class="fb-qq-label">QQ 群号：</span>
            <span class="fb-qq">{{ info.qq_group }}</span>
          </div>
        </div>
      </div>

      <!-- 邮件反馈 -->
      <div class="fb-card">
        <div class="fb-card-head">
          <el-icon class="fb-icon mail"><Message /></el-icon>
          <div>
            <h4>邮件反馈</h4>
            <p>填写下方内容并提交，将自动发送到开发者邮箱</p>
          </div>
        </div>
        <div v-if="info.email_enabled" class="fb-body">
          <el-input
            v-model="mailForm.content"
            type="textarea"
            :rows="8"
            maxlength="2000"
            show-word-limit
            placeholder="请描述你遇到的问题或建议…"
          />
          <div class="fb-mail-foot">
            <el-input
              v-model="mailForm.contact"
              class="fb-contact"
              maxlength="100"
              clearable
              placeholder="必填：你的 QQ / 邮箱等联系方式，方便回复你"
            />
            <el-button type="primary" :loading="sending" :icon="Promotion" @click="sendMailFeedback">发送反馈</el-button>
          </div>
          <div class="fb-mail-hint">邮件由应用内置通道直接送达，无需任何邮箱配置</div>
        </div>
        <div v-else class="fb-empty"><el-icon><InfoFilled /></el-icon>邮件通道暂未开启，请先通过 QQ 群联系</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { ChatLineSquare, Promotion, InfoFilled, Message, ChatDotRound } from '@element-plus/icons-vue'
import { getFeedbackInfo, sendFeedback } from '@/api/feedback'

// QQ 群二维码图片：位于前端 public/，构建时复制到 dist 根并随后端二进制内嵌
const qqGroupImg = (import.meta.env.BASE_URL || './') + 'qq-group.png'

const info = ref({})

const isUrl = (s) => /^https?:\/\//i.test(s || '')

const openQQGroup = () => {
  if (isUrl(info.value.qq_group)) window.open(info.value.qq_group, '_blank')
}

// ===== 邮件反馈 =====
const mailForm = reactive({ contact: '', content: '' })
const sending = ref(false)

const sendMailFeedback = async () => {
  const content = mailForm.content.trim()
  if (!content) return ElMessage.warning('请先填写反馈内容')
  const contact = mailForm.contact.trim()
  if (!contact) return ElMessage.warning('请填写联系方式（QQ / 邮箱等），方便作者回复你')
  sending.value = true
  try {
    await sendFeedback(contact, content)
    ElMessage.success('反馈已发送，感谢你的建议！')
    mailForm.content = ''
    mailForm.contact = ''
  } catch (e) {
    ElMessage.error('发送失败：' + (e.response?.data?.error || e.message || ''))
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  try {
    const { data } = await getFeedbackInfo()
    info.value = data || {}
  } catch {
    /* 忽略 */
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

.fb-channels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}
.fb-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  background: var(--el-bg-color);
  box-shadow: 0 8px 24px rgba(31, 38, 135, 0.06);
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.fb-card-head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.fb-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  font-size: 22px;
  color: #fff;
  flex: none;
  justify-content: center;
}
.fb-icon.qq {
  background: linear-gradient(135deg, #3b9df5, #1e7cd6);
}
.fb-icon.mail {
  background: linear-gradient(135deg, #06b6d4, #0ea5e9);
}
.fb-card-head h4 {
  margin: 0;
  font-size: 15px;
}
.fb-card-head p {
  margin: 3px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.fb-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.fb-qq-body {
  align-items: center;
  text-align: center;
}
.fb-qr-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.fb-qr {
  width: 220px;
  height: 220px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  background: #fff;
  cursor: zoom-in;
  box-shadow: 0 6px 18px rgba(31, 38, 135, 0.1);
}
.fb-qr-tip {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.fb-qq-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.fb-qq-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.fb-qq {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 1px;
  color: var(--el-text-color-primary);
}
.fb-mail-foot {
  display: flex;
  align-items: center;
  gap: 10px;
}
.fb-contact {
  flex: 1;
}
.fb-mail-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.fb-empty {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 10px;
  padding: 12px;
}

@media (max-width: 768px) {
  .fb-mail-foot {
    flex-direction: column;
    align-items: stretch;
  }
  .fb-mail-foot .el-button {
    width: 100%;
    margin-left: 0;
  }
  .fb-qr {
    width: 200px;
    height: 200px;
  }
}
</style>
