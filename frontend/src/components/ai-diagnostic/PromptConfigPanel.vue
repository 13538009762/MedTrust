<template>
  <!-- 提示词审查与外部大模型 API 配置指南 — WP-16 从 AiDiagnosticDialog 抽取 -->
  <div class="tab-pane-box">
    <el-alert
      type="info"
      show-icon
      :closable="false"
      class="mb-3"
      title="大模型接口集成说明：当前可无缝调用本地内置临床知识推理引擎，亦支持填入您的 OpenAI / DeepSeek / Qwen 等大模型 API Key 随时接入真实通用大模型。"
    />

    <div class="api-guide-card mb-3">
      <div class="agc-title">外部大模型 API 配置指引：</div>
      <div class="agc-body">
        <p>您只需在系统配置文件 <code>backend/config/config.yaml</code> 或环境变量中设置如下参数，后端即可自动切换至外部大模型：</p>
        <pre class="code-block font-mono">
# 配置文件: backend/config/config.yaml
ai:
  enabled: true
  api_url: "https://api.deepseek.com/v1"  # 外部大模型 API 端点
  api_key: "sk-your-real-api-key-here"    # 填入您的 API Key
  model: "deepseek-chat"                  # 模型名称 (如 deepseek-chat, gpt-4o, qwen-turbo)
  temperature: 0.2
        </pre>
      </div>
    </div>

    <div class="prompt-preview-card">
      <div class="ppc-header">
        <span>结构化临床摘要生成 Prompt 预览：</span>
        <el-button size="small" link type="primary" :icon="CopyDocument" @click="$emit('copy-prompt')">
          复制完整提示词
        </el-button>
      </div>
      <pre class="prompt-text-pre font-mono">{{ result.raw_prompt }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CopyDocument } from '@element-plus/icons-vue'

defineProps<{
  result: any
}>()

defineEmits<{
  (e: 'copy-prompt'): void
}>()
</script>

<style scoped>
.tab-pane-box {
  background: #ffffff;
  border-radius: 10px;
  padding: 16px;
  border: 1px solid #e2e8f0;
}

.prompt-text-pre {
  background: #0f172a;
  color: #f8fafc;
  padding: 14px;
  border-radius: 8px;
  font-size: 11px;
  line-height: 1.5;
  max-height: 280px;
  overflow-y: auto;
  white-space: pre-wrap;
}

.code-block {
  background: #1e293b;
  color: #38bdf8;
  padding: 10px;
  border-radius: 6px;
  font-size: 11px;
  line-height: 1.4;
}
</style>