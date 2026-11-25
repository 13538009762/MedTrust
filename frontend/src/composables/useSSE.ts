import { ref } from 'vue'
import { ElNotification } from 'element-plus'
import router from '../router'
import { createSSETicketApi } from '../api/auth'
import { onAuthCleared } from '../stores/auth'

export interface SSENotifyEvent {
  type: 'access_request' | 'request_approved' | 'request_rejected' | 'break_glass' | string
  title: string
  message: string
  payload?: any
}

type EventCallback = (event: SSENotifyEvent) => void

const isConnected = ref(false)
const eventListeners: EventCallback[] = []
let eventSource: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempt = 0
const RECONNECT_BACKOFF_MS = [5000, 10000, 20000, 30000, 60000]
let authClearRegistered = false

function teardownSSE() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (eventSource) {
    try {
      eventSource.close()
    } catch {}
    eventSource = null
  }
  isConnected.value = false
  reconnectAttempt = 0
}

export function useSSE() {
  if (!authClearRegistered) {
    authClearRegistered = true
    onAuthCleared(() => teardownSSE())
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    const delay = RECONNECT_BACKOFF_MS[Math.min(reconnectAttempt, RECONNECT_BACKOFF_MS.length - 1)]
    reconnectAttempt += 1
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      const currentToken = sessionStorage.getItem('medtrust_token')
      if (currentToken) {
        connect(currentToken)
      }
    }, delay)
  }
  function addEventListener(cb: EventCallback) {
    if (!eventListeners.includes(cb)) {
      eventListeners.push(cb)
    }
    return () => {
      const idx = eventListeners.indexOf(cb)
      if (idx !== -1) eventListeners.splice(idx, 1)
    }
  }

  function notifySubscribers(event: SSENotifyEvent) {
    eventListeners.forEach(cb => {
      try {
        cb(event)
      } catch (e) {
        console.error('[SSE] Listener error:', e)
      }
    })
  }

  async function connect(token?: string) {
    const effectiveToken = token || sessionStorage.getItem('medtrust_token')
    if (!effectiveToken) return
    if (eventSource && eventSource.readyState === EventSource.OPEN) return

    if (eventSource) {
      try {
        eventSource.close()
      } catch {}
      eventSource = null
    }

    let ticket = ''
    try {
      const res = await createSSETicketApi()
      if (res && res.code === 200 && res.data?.ticket) {
        ticket = res.data.ticket
      }
    } catch (err) {
      console.warn('[MedTrust SSE] 获取单次 Ticket 失败:', err)
    }

    // 严禁在 URL 查询参数中降级透传长效 JWT，无有效一次性票据时交由重连定时器指数退避重试
    if (!ticket) {
      scheduleReconnect()
      return
    }

    const baseApi = import.meta.env.VITE_API_BASE_URL || '/api/v1'
    const sseUrl = `${baseApi}/notifications/stream?ticket=${encodeURIComponent(ticket)}`

    try {
      eventSource = new EventSource(sseUrl)

      eventSource.addEventListener('connected', () => {
        isConnected.value = true
        reconnectAttempt = 0
        if (reconnectTimer) {
          clearTimeout(reconnectTimer)
          reconnectTimer = null
        }
        console.log('[MedTrust SSE] 实时双向业务通知长连接建立成功')
      })

      eventSource.addEventListener('notify', (e: MessageEvent) => {
        try {
          const event: SSENotifyEvent = JSON.parse(e.data)
          handleIncomingEvent(event)
          notifySubscribers(event)
        } catch (err) {
          console.error('[MedTrust SSE] 解析通知数据失败:', err)
        }
      })

      eventSource.addEventListener('heartbeat', () => {
        isConnected.value = true
      })

      eventSource.onerror = () => {
        isConnected.value = false
        if (eventSource) {
          try {
            eventSource.close()
          } catch {}
          eventSource = null
        }
        scheduleReconnect()
      }
    } catch (e) {
      console.warn('[MedTrust SSE] 创建 EventSource 异常:', e)
    }
  }

  function handleIncomingEvent(event: SSENotifyEvent) {
    let notifyType: 'success' | 'warning' | 'info' | 'error' = 'info'
    let targetPath = ''

    switch (event.type) {
      case 'access_request':
        notifyType = 'warning'
        targetPath = '/patient/auth'
        break
      case 'request_approved':
        notifyType = 'success'
        targetPath = '/doctor/query/authorized'
        break
      case 'request_rejected':
        notifyType = 'error'
        targetPath = '/doctor/query/pending'
        break
      case 'break_glass':
        notifyType = 'error'
        targetPath = '/patient/emergency'
        break
      default:
        notifyType = 'info'
    }

    const displayMessage = targetPath
      ? `${event.message || ''}（点击立即查看）`
      : event.message

    ElNotification({
      title: event.title || '系统即时通知',
      message: displayMessage,
      type: notifyType,
      duration: event.type === 'break_glass' ? 0 : 9000,
      position: 'top-right',
      dangerouslyUseHTMLString: false,
      onClick: () => {
        if (targetPath && router.currentRoute.value.path !== targetPath) {
          router.push(targetPath)
        }
      }
    })
  }

  function disconnect() {
    teardownSSE()
  }

  return {
    isConnected,
    connect,
    disconnect,
    addEventListener,
  }
}
