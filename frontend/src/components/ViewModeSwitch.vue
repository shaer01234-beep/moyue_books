<template>
  <div class="vm-switch" v-if="!isMobile">
    <span class="vm-label">
      <el-icon><Grid /></el-icon> 视图
    </span>
    <el-radio-group :model-value="modelValue" size="small" @update:model-value="(v) => $emit('update:modelValue', v)">
      <el-radio-button label="grid">网格</el-radio-button>
      <el-radio-button label="list">列表</el-radio-button>
    </el-radio-group>
  </div>
</template>

<script setup>
import { useIsMobile } from '@/composables/useMobileView'
defineProps({ modelValue: { type: String, default: 'grid' } })
defineEmits(['update:modelValue'])
const { isMobile } = useIsMobile()
</script>

<style scoped>
.vm-switch {
  display: flex;
  align-items: center;
  gap: 8px;
}
.vm-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
  white-space: nowrap;
}
.vm-label .el-icon {
  color: var(--app-primary);
}
.vm-switch :deep(.el-radio-button__inner) {
  border-color: color-mix(in srgb, var(--app-primary) 25%, transparent);
  background: rgba(255, 255, 255, 0.65);
  color: #64748b;
}
.vm-switch :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-left-color: color-mix(in srgb, var(--app-primary) 25%, transparent);
}
.vm-switch :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background: var(--app-gradient);
  border-color: transparent;
  color: #fff;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--app-primary) 35%, transparent);
}
</style>
