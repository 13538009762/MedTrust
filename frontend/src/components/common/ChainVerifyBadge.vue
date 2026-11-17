<template>
  <el-tag
    :size="size"
    :type="tagType"
    :effect="resolvedStatus === 'tampered' ? 'dark' : 'light'"
    class="mt-chain-verify-badge"
  >
    <span class="mt-chain-verify-badge__inner">
      <el-icon :size="13">
        <WarningFilled v-if="resolvedStatus === 'tampered'" />
        <CircleCheckFilled v-else-if="resolvedStatus === 'verified'" />
        <Clock v-else />
      </el-icon>
      <span>{{ displayLabel }}</span>
    </span>
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { WarningFilled, CircleCheckFilled, Clock } from '@element-plus/icons-vue'

export type ChainVerifyStatus = 'verified' | 'tampered' | 'pending'

interface Props {
  status?: ChainVerifyStatus
  tampered?: boolean
  verified?: boolean
  size?: 'small' | 'default' | 'large'
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  status: undefined,
  tampered: false,
  verified: false,
  size: 'small',
  label: ''
})

const resolvedStatus = computed<ChainVerifyStatus>(() => {
  if (props.status) return props.status
  if (props.tampered) return 'tampered'
  if (props.verified) return 'verified'
  return 'pending'
})

const tagType = computed(() => {
  if (resolvedStatus.value === 'tampered') return 'danger'
  if (resolvedStatus.value === 'verified') return 'success'
  return 'warning'
})

const displayLabel = computed(() => {
  if (props.label) return props.label
  if (resolvedStatus.value === 'tampered') return '数据指纹不匹配 · 禁止采信'
  if (resolvedStatus.value === 'verified') return '链上存证核验一致'
  return '链上存证待复核'
})
</script>

<style scoped>
.mt-chain-verify-badge {
  border-radius: var(--mt-radius-xs, 4px);
  font-weight: 600;
}

.mt-chain-verify-badge__inner {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
</style>
