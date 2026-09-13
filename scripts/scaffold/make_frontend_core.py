import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\frontend\src'
os.makedirs(os.path.join(base, 'api'), exist_ok=True)
os.makedirs(os.path.join(base, 'stores'), exist_ok=True)
os.makedirs(os.path.join(base, 'router'), exist_ok=True)
os.makedirs(os.path.join(base, 'layouts'), exist_ok=True)
os.makedirs(os.path.join(base, 'components'), exist_ok=True)
os.makedirs(os.path.join(base, 'views', 'doctor'), exist_ok=True)
os.makedirs(os.path.join(base, 'views', 'patient'), exist_ok=True)
os.makedirs(os.path.join(base, 'views', 'admin'), exist_ok=True)
os.makedirs(os.path.join(base, 'views', 'supervisor'), exist_ok=True)

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

# 1. main.ts
write_file('main.ts', """import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

import App from './App.vue'
import router from './router'
import './assets/theme.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.mount('#app')
""")

# 2. App.vue
write_file('App.vue', """<template>
  <el-config-provider :locale="zhCn">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}
html, body, #app {
  width: 100%;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
}
</style>
""")

# 3. api/client.ts
write_file('api/client.ts', """import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: 'http://127.0.0.1:8080/api/v1',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('medtrust_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    if (error.response) {
      if (error.response.status === 401) {
        localStorage.removeItem('medtrust_token')
        localStorage.removeItem('medtrust_user')
        ElMessage.error(error.response.data?.message || '登录已过期，请重新登录')
        router.push('/login')
      } else if (error.response.status === 403) {
        // 留给业务层或直接提示
        return Promise.reject(error.response.data)
      } else {
        ElMessage.error(error.response.data?.message || '请求服务失败')
      }
      return Promise.reject(error.response.data)
    }
    ElMessage.error('网络连接异常或后端服务未启动')
    return Promise.reject(error)
  }
)

export default api
""")

# 4. stores/auth.ts
write_file('stores/auth.ts', """import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface User {
  id: number
  user_no: string
  username: string
  real_name: string
  role: 'doctor' | 'patient' | 'admin' | 'supervisor'
  hospital_id: number
  department_id: number
  title: string
  phone: string
  status: 'NORMAL' | 'RESTRICTED' | 'DISABLED'
  hospital_name?: string
  department_name?: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('medtrust_token') || '')
  const user = ref<User | null>(
    localStorage.getItem('medtrust_user')
      ? JSON.parse(localStorage.getItem('medtrust_user')!)
      : null
  )

  function setAuth(newToken: string, newUser: User) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem('medtrust_token', newToken)
    localStorage.setItem('medtrust_user', JSON.stringify(newUser))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('medtrust_token')
    localStorage.removeItem('medtrust_user')
  }

  return {
    token,
    user,
    setAuth,
    logout,
  }
})
""")

# 5. components/AiAssistantDialog.vue
write_file('components/AiAssistantDialog.vue', """<template>
  <div class="ai-assistant-wrapper">
    <!-- 悬浮触发微标按钮 -->
    <div class="ai-float-badge" @click="visible = !visible" title="唤起 MedTrust 受控 AI 助手">
      <el-icon :size="26" class="sparkle-icon"><MagicStick /></el-icon>
      <span class="badge-text">AI 助手</span>
    </div>

    <!-- 弹窗式对话抽屉 / 面板 -->
    <el-dialog
      v-model="visible"
      title="MedTrust 受控 AI 智能辅助查询"
      width="540px"
      append-to-body
      class="ai-dialog"
      :show-close="true"
    >
      <div class="ai-chat-container">
        <div class="ai-safety-notice">
          <el-icon><ShieldCheck /></el-icon>
          <span>受控边界：AI 严禁直连数据库，全量查询携带当前 Token 经 Go 网关鉴权</span>
        </div>

        <div class="chat-messages" ref="msgContainer">
          <div
            v-for="(msg, idx) in messages"
            :key="idx"
            :class="['chat-bubble', msg.sender === 'user' ? 'user-msg' : 'ai-msg']"
          >
            <div class="sender-avatar">
              <el-avatar :size="32" :style="{ background: msg.sender === 'user' ? '#8b5cf6' : '#10b981' }">
                {{ msg.sender === 'user' ? '我' : 'AI' }}
              </el-avatar>
            </div>
            <div class="bubble-content">
              <div class="bubble-text" style="white-space: pre-wrap;">{{ msg.text }}</div>
              <div v-if="msg.tool" class="tool-tag">
                <el-tag size="small" type="info">Tool Calling: {{ msg.tool }}</el-tag>
              </div>
            </div>
          </div>
          <div v-if="loading" class="chat-bubble ai-msg">
            <div class="sender-avatar">
              <el-avatar :size="32" style="background: #10b981;">AI</el-avatar>
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
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

const visible = ref(false)
const inputMsg = ref('')
const loading = ref(false)
const msgContainer = ref<HTMLDivElement | null>(null)
const auth = useAuthStore()

interface Message {
  sender: 'user' | 'ai'
  text: string
  tool?: string
}

const messages = ref<Message[]>([
  {
    sender: 'ai',
    text: '您好！我是 MedTrust 医疗数据受控 AI 助手。您可以向我咨询病历、患者授权状态或发起防篡改核验。\\n\\n示例提问：\\n• \"查询张三最近的检查记录\"\\n• \"我现在授权了哪些医生？\"\\n• \"核验病历1的区块链防篡改哈希\"',
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
    const res = await axios.post(
      'http://127.0.0.1:8000/api/v1/ai/chat',
      { message: query },
      {
        headers: {
          Authorization: `Bearer ${auth.token}`,
        },
        timeout: 10000,
      }
    )

    if (res.data?.code === 200) {
      messages.value.push({
        sender: 'ai',
        text: res.data.data.reply,
        tool: res.data.data.tool_called,
      })
    } else {
      messages.value.push({
        sender: 'ai',
        text: '抱歉，未能获得响应：' + (res.data?.message || '未知异常'),
      })
    }
  } catch (err: any) {
    messages.value.push({
      sender: 'ai',
      text: 'AI 服务连接异常，请确保 Python FastAPI 服务 (端口 8000) 已启动。',
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
  right: 28px;
  bottom: 32px;
  z-index: 1999;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: linear-gradient(135deg, #a78bfa 0%, #8b5cf6 100%);
  color: white;
  border-radius: 30px;
  box-shadow: 0 10px 25px rgba(139, 92, 246, 0.35);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.ai-float-badge:hover {
  transform: translateY(-3px) scale(1.05);
  box-shadow: 0 14px 30px rgba(139, 92, 246, 0.45);
}
.badge-text {
  font-weight: 700;
  font-size: 15px;
}
.ai-chat-container {
  display: flex;
  flex-direction: column;
  height: 520px;
}
.ai-safety-notice {
  background: rgba(139, 92, 246, 0.08);
  border: 1px solid rgba(139, 92, 246, 0.2);
  color: #7c3aed;
  font-size: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px 4px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.chat-bubble {
  display: flex;
  gap: 12px;
  max-width: 90%;
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
  padding: 12px 16px;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.5;
  color: #1e293b;
}
.user-msg .bubble-content {
  background: #8b5cf6;
  color: white;
  border-bottom-right-radius: 2px;
}
.ai-msg .bubble-content {
  border-bottom-left-radius: 2px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03);
}
.tool-tag {
  margin-top: 8px;
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
  border-radius: 14px;
  font-size: 12px;
  transition: all 0.2s;
}
.prompt-chip:hover {
  background: #ede9fe;
  color: #7c3aed;
  border-color: #c4b5fd;
}
.chat-input-row {
  margin-top: 4px;
}
</style>
""")

print("Frontend core files generated successfully!")
