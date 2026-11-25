<template>
  <el-tooltip :content="tooltipText" placement="top" :show-after="200">
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
  </el-tooltip>
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

// 防御性优先级：篡改安全态最高，杜绝误传其他 status 掩盖篡改风险
const resolvedStatus = computed<ChainVerifyStatus>(() => {
  if (props.tampered || props.status === 'tampered') return 'tampered'
  if (props.status) return props.status
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

const tooltipText = computed(() => {
  if (resolvedStatus.value === 'tampered') {
    return '该病历当前数据与区块链上固化的原始指纹不一致，疑似已被非法修改，请勿作为诊疗依据并及时上报监管核查。'
  }
  if (resolvedStatus.value === 'verified') {
    return '该病历已完成区块链指纹比对，与归档时原始记录完全一致，数据真实可靠。'
  }
  return '该病历尚未执行实时指纹复核，可在详情页或核验工作台发起一键验真。'
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
