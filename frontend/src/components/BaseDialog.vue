<template>
  <el-dialog
    v-model="model"
    :title="title"
    :width="width"
    align-center
    append-to-body
    class="detail-dialog"
    modal-class="detail-dialog-modal"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    :show-close="showClose"
  >
    <template #header>
      <slot name="header">
        <div class="dlg-head">
          <span v-if="icon" class="dlg-head-ic">
            <el-icon><component :is="icon" /></el-icon>
          </span>
          <div class="dlg-head-main">
            <div class="dlg-head-title">{{ title }}</div>
            <div v-if="subtitle" class="dlg-head-sub">{{ subtitle }}</div>
          </div>
        </div>
      </slot>
    </template>
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },       // 头部副标题/说明
  icon: { type: String, default: '' },           // 头部图标（Element Plus 图标组件名，如 'FolderAdd'）
  width: { type: String, default: '480px' },
  closeOnClickModal: { type: Boolean, default: true },
  closeOnPressEscape: { type: Boolean, default: true },
  showClose: { type: Boolean, default: true },
})

const emit = defineEmits(['update:modelValue'])

const model = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})
</script>
