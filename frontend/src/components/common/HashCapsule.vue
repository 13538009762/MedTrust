<template>
  <span class="mt-hash-capsule" :class="{ 'is-empty': !hasValue }">
    <el-tooltip
      v-if="hasValue"
      :content="String(value)"
      placement="top"
      :show-after="200"
    >
      <code class="mt-hash-code">{{ displayText }}</code>
    </el-tooltip>
    <span v-else class="mt-hash-empty">{{ emptyText }}</span>

    <button
      v-if="copyable && hasValue"
      type="button"
      class="mt-hash-copy-btn"
      :title="`复制${label}`"
      @click.stop="handleCopy"
    >
      <el-icon :size="12">
        <Check v-if="copied" />
        <CopyDocument v-else />
      </el-icon>
    </button>
  </span>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { CopyDocument, Check } from '@element-plus/icons-vue'
import { formatHashMiddle, copyTextToClipboard } from '@/utils/hashFormatters'

interface Props {
  value?: string | null
  prefixLen?: number
  suffixLen?: number
  copyable?: boolean
  label?: string
  emptyText?: string
  truncate?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  value: '',
  prefixLen: 8,
  suffixLen: 6,
  copyable: true,
  label: '哈希凭证',
  emptyText: '—',
  truncate: true
})

const copied = ref(false)

const hasValue = computed(() => {
  if (props.value === null || props.value === undefined) return false
  const s = String(props.value).trim()
  return Boolean(s && s !== '-' && s !== '—')
})

const displayText = computed(() => {
  if (!hasValue.value) return props.emptyText
  const raw = String(props.value).trim()
  return props.truncate
    ? formatHashMiddle(raw, props.prefixLen, props.suffixLen)
    : raw
})

async function handleCopy() {
  if (!hasValue.value) return
  const ok = await copyTextToClipboard(String(props.value), props.label)
  if (ok) {
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 1500)
  }
}
</script>

<style scoped>
.mt-hash-capsule {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 2px 8px;
  background-color: var(--mt-bg-subtle, #f1f5f9);
  border: 1px solid var(--mt-border-strong, #cbd5e1);
  border-radius: var(--mt-radius-xs, 4px);
  font-family: var(--mt-font-mono, monospace);
  font-size: var(--mt-text-xs, 12px);
  line-height: 1.4;
  color: var(--mt-text-regular, #334155);
  vertical-align: middle;
}

.mt-hash-capsule.is-empty {
  background-color: transparent;
  border-color: transparent;
  padding-left: 0;
  color: var(--mt-text-muted, #94a3b8);
}

.mt-hash-code {
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  word-break: break-all;
}

.mt-hash-empty {
  color: var(--mt-text-muted, #94a3b8);
}

.mt-hash-copy-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  background: transparent;
  color: var(--mt-text-secondary, #64748b);
  border-radius: 3px;
  cursor: pointer;
  flex-shrink: 0;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.mt-hash-copy-btn:hover {
  color: var(--mt-primary, #0284c7);
  background-color: var(--mt-border-base, #e2e8f0);
}
</style>
