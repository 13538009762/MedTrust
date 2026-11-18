<template>
  <div class="ai-assistant-wrapper">
    <!-- 悬浮触发微标按钮 -->
    <div class="ai-float-badge" @click="visible = !visible" title="唤起 MedTrust 医疗数据受控检索助手">
      <el-icon :size="20" class="sparkle-icon"><Search /></el-icon>
      <span class="badge-text">受控检索</span>
    </div>

    <!-- 弹窗式对话抽屉 / 面板 -->
    <el-dialog
      v-model="visible"
      title="MedTrust 受控 AI 智能辅助查询"
      width="580px"
      append-to-body
      class="ai-dialog"
      :show-close="true"
    >
      <div class="ai-chat-container">
        <div class="ai-safety-notice">
          <el-icon><CircleCheck /></el-icon>
          <span>受控边界：AI 严禁直连数据库，全量查询携带当前 Token 经 Go 网关鉴权</span>
        </div>

        <div class="chat-messages" ref="msgContainer">
          <div
            v-for="(msg, idx) in messages"
            :key="idx"
            :class="['chat-bubble', msg.sender === 'user' ? 'user-msg' : 'ai-msg']"
          >
            <div class="sender-avatar">
              <el-avatar v-if="msg.sender === 'user'" :size="32" :src="auth.user?.avatar" style="background: #8b5cf6">
                我
              </el-avatar>
              <img v-else src="/emblem.png" class="ai-msg-avatar" alt="MedTrust AI" />
            </div>
            <div class="bubble-content">
              <div class="bubble-text" style="white-space: pre-wrap;">{{ msg.text }}</div>

              <!-- 受控工具调用链可视化看板 (论文 3.4 节核心支撑) -->
              <div v-if="msg.tool_trace && msg.tool_trace.length" class="tool-trace-block mt-3">
                <div class="trace-header">
                  <div class="trace-header-left">
                    <el-icon class="text-indigo-600 mr-1"><Connection /></el-icon>
                    <span class="trace-header-title">受控工具链执行轨迹 (Single Gatekeeper Trace)</span>
                  </div>
                  <el-tag size="small" type="success" effect="plain">已透传当前 JWT</el-tag>
                </div>
                <div class="trace-items-list">
                  <div
                    v-for="(trace, tIdx) in msg.tool_trace"
                    :key="tIdx"
                    class="trace-card"
                    :class="trace.status_code === 200 ? 'trace-pass' : 'trace-intercept'"
                  >
                    <div class="trace-card-top">
                      <span class="trace-tool-name font-mono font-bold">{{ trace.tool }}</span>
                      <el-tag
                        size="small"
                        :type="trace.status_code === 200 ? 'success' : 'danger'"
                        effect="dark"
                      >
                        HTTP {{ trace.status_code }} · {{ trace.auth_check === 'PASSED' ? '网关核准' : '网关拦截' }}
                      </el-tag>
                    </div>
                    <div class="trace-card-meta">
                      <div class="meta-row">
                        <span class="lbl">Go 安全网关端点:</span>
                        <code class="val-mono">{{ trace.endpoint }}</code>
                      </div>
                      <div class="meta-row" v-if="trace.gate1_guard">
                        <span class="lbl font-bold" style="color: #4338ca;">关卡1 (Go规则守门):</span>
                        <span class="val-text" :class="trace.status_code === 200 ? 'text-emerald-700' : 'text-rose-700'">
                          {{ trace.gate1_guard }}
                        </span>
                      </div>
                      <div class="meta-row" v-if="trace.gate2_semantic">
                        <span class="lbl font-bold" style="color: #7e22ce;">关卡2 (受控语义提炼):</span>
                        <span class="val-text text-purple-800">{{ trace.gate2_semantic }}</span>
                      </div>
                      <div class="meta-row" v-if="trace.gate3_post_check">
                        <span class="lbl font-bold" style="color: #0f766e;">关卡3 (Go后置核验):</span>
                        <span class="val-text text-teal-800">{{ trace.gate3_post_check }}</span>
                      </div>
                      <div class="meta-row" v-if="!trace.gate1_guard">
                        <span class="lbl">访问控制审查:</span>
                        <span class="val-text" :class="trace.status_code === 200 ? 'text-emerald-700' : 'text-rose-700'">
                          {{ trace.status_code === 200 ? 'RBAC角色匹配 + ABAC环境属性校验通过' : '未取得目标数据访问授权 (403 Forbidden)' }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-else-if="msg.tool" class="tool-tag mt-2">
                <el-tag size="small" type="info" effect="plain">
                  <el-icon class="mr-1"><Connection /></el-icon>
                  Tool Calling: {{ msg.tool }}
                </el-tag>
              </div>
            </div>
          </div>
          <div v-if="loading" class="chat-bubble ai-msg">
            <div class="sender-avatar">
              <img src="/emblem.png" class="ai-msg-avatar" alt="MedTrust AI" />
            </div>
            <div class="bubble-content">
              <div class="thinking-text">
                <el-icon class="is-loading"><Loading /></el-icon> 正在通过安全网关解析意图并调取数据...
              </div>
            </div>
          </div>
        </div>

        <!-- 推荐快捷提问 -->
        <div class="quick-prompts">
          <el-tag
            v-for="(p, i) in quickPrompts"
            :key="i"
            class="prompt-chip"
            @click="sendQuick(p)"
            effect="plain"
          >
            {{ p }}
          </el-tag>
        </div>

        <div class="chat-input-row">
          <el-input
            v-model="inputMsg"
            placeholder="输入自然语言指令，如：查询张三最近的检查记录"
            @keyup.enter="handleSend"
            :disabled="loading"
          >
            <template #append>
              <el-button type="primary" :loading="loading" @click="handleSend">发送</el-button>
            </template>
          </el-input>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import axios from 'axios'
import { Search, CircleCheck, Loading, Connection } from '@element-plus/icons-vue'
import { useAuthStore } from '../stores/auth'

const visible = ref(false)
const inputMsg = ref('')
const loading = ref(false)
const msgContainer = ref<HTMLDivElement | null>(null)
const auth = useAuthStore()

interface ToolTraceItem {
  tool: string
  endpoint: string
  status_code: number
  auth_check: string
  gate1_guard?: string
  gate2_semantic?: string
  gate3_post_check?: string
}

interface Message {
  sender: 'user' | 'ai'
  text: string
  tool?: string
  tool_status?: string
  tool_trace?: ToolTraceItem[]
}

const messages = ref<Message[]>([
  {
    sender: 'ai',
    text: '您好！我是 MedTrust 医疗数据受控检索助手。您可以查询病历检索、跨院既往史 SOAP 汇总、患者授权策略或发起防篡改核验。\n\n合规声明：本工具仅用于辅助安全查询与合规审计，不涉及任何疾病诊断与处方建议。\n\n示例提问：\n• "查询张三最近的检查记录"\n• "我现在授权了哪些医生？"\n• "核验病历1的区块链防篡改哈希"',
  }
])

const quickPrompts = [
  '查询张三最近的检查记录',
  '我现在授权了哪些医生？',
  '核验病历1的防篡改状态',
  '今天有哪些高风险调阅事件？',
]

function sendQuick(text: string) {
  inputMsg.value = text
  handleSend()
}

async function handleSend() {
  const query = inputMsg.value.trim()
  if (!query) return

  messages.value.push({ sender: 'user', text: query })
  inputMsg.value = ''
  loading.value = true
  scrollToBottom()

  try {
    const aiChatUrl = `${(import.meta.env.VITE_AI_BASE_URL || '/api/v1/ai').replace(/\/$/, '')}/chat`
    const res = await axios.post(
      aiChatUrl,
      { message: query },
      {
        headers: {
          Authorization: `Bearer ${auth.token}`,
        },
        timeout: 15000,
      }
    )

    if (res.data?.code === 200) {
      const data = res.data.data || {}
      messages.value.push({
        sender: 'ai',
        text: data.reply || res.data.message || '已处理您的请求',
        tool: data.tool_called,
        tool_status: data.tool_status,
        tool_trace: data.tool_trace || [],
      })
    } else {
      messages.value.push({
        sender: 'ai',
        text: '抱歉，未能获得响应：' + (res.data?.message || '未知异常'),
      })
    }
  } catch (err: unknown) {
    let errMsg = 'AI 智能体服务暂不可达，请确认 ai_agent 服务（端口 8000）及后端网关运行正常。'
    if (axios.isAxiosError(err)) {
      const detail = err.response?.data?.detail || err.response?.data?.message
      if (err.response?.status === 401) {
        errMsg = `会话鉴权未通过 (${detail || '请重新登录'})`
      } else if (detail) {
        errMsg = `AI 服务返回异常：${detail}`
      }
    }
    messages.value.push({
      sender: 'ai',
      text: errMsg,
    })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (msgContainer.value) {
      msgContainer.value.scrollTop = msgContainer.value.scrollHeight
    }
  })
}
</script>

<style scoped>
.ai-float-badge {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 1999;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--mt-primary, #0284c7);
  color: #ffffff;
  border-radius: 6px;
  box-shadow: 0 2px 6px rgba(15, 23, 42, 0.12);
  border: 1px solid var(--mt-primary-hover, #0369a1);
  cursor: pointer;
  transition: background-color 0.2s ease;
}
.ai-float-badge:hover {
  background: var(--mt-primary-hover, #0369a1);
}
.badge-text {
  font-weight: 600;
  font-size: 13px;
  letter-spacing: 0.2px;
}
.ai-safety-notice {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  border-radius: 6px;
  color: #047857;
  font-size: 12px;
  margin-bottom: 12px;
}
.chat-messages {
  height: 380px;
  overflow-y: auto;
  padding: 12px 4px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.chat-bubble {
  display: flex;
  gap: 10px;
  max-width: 92%;
}
.ai-msg-avatar {
  width: 32px;
  height: 32px;
  object-fit: contain;
}
.user-msg {
  align-self: flex-end;
  flex-direction: row-reverse;
}
.ai-msg {
  align-self: flex-start;
}
.bubble-content {
  background: #f1f5f9;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 13.5px;
  line-height: 1.5;
  color: #1e293b;
}
.user-msg .bubble-content {
  background: var(--mt-primary, #0284c7);
  color: #ffffff;
}
.ai-msg .bubble-content {
  background: #ffffff;
  border: 1px solid #e2e8f0;
}
.tool-tag {
  margin-top: 8px;
}

/* 工具调用链可视化卡片 */
.tool-trace-block {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 10px 12px;
}
.trace-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 12px;
}
.trace-header-left {
  display: flex;
  align-items: center;
}
.trace-header-title {
  font-weight: 700;
  color: #334155;
}
.trace-items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.trace-card {
  background: #ffffff;
  border-radius: 6px;
  padding: 8px 10px;
  border-left: 3px solid #059669;
  border-top: 1px solid #e2e8f0;
  border-right: 1px solid #e2e8f0;
  border-bottom: 1px solid #e2e8f0;
}
.trace-card.trace-intercept {
  border-left-color: #dc2626;
}
.trace-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.trace-tool-name {
  color: #1e293b;
  font-size: 12px;
}
.trace-card-meta {
  font-size: 11px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.meta-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.meta-row .lbl {
  color: #64748b;
}
.meta-row .val-mono {
  font-family: var(--mt-font-mono, monospace);
  color: #475569;
  background: #f1f5f9;
  padding: 1px 4px;
  border-radius: 3px;
}
.meta-row .val-text {
  font-weight: 500;
}

.thinking-text {
  color: #64748b;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.quick-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 12px 0;
}
.prompt-chip {
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  transition: all 0.2s;
}
.prompt-chip:hover {
  background: var(--mt-primary-soft, #f0f9ff);
  color: var(--mt-primary, #0284c7);
  border-color: var(--mt-primary-border, #bae6fd);
}
.chat-input-row {
  margin-top: 4px;
}
</style>
