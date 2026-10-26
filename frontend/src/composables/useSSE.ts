import { ref } from 'vue'
import { ElNotification } from 'element-plus'
import router from '../router'

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
let reconnectTimer: any = null

export function useSSE() {
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

  function connect(token: string) {
    if (!token) return
    if (eventSource && eventSource.readyState === EventSource.OPEN) return

    if (eventSource) {
      try {
        eventSource.close()
      } catch {}
      eventSource = null
    }

    const sseUrl = `http://127.0.0.1:8080/api/v1/notifications/stream?token=${encodeURIComponent(token)}`

    try {
      eventSource = new EventSource(sseUrl)

      eventSource.addEventListener('connected', () => {
        isConnected.value = true
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
        // 5秒后尝试静默重连
        if (!reconnectTimer) {
          reconnectTimer = setTimeout(() => {
            reconnectTimer = null
            const currentToken = sessionStorage.getItem('medtrust_token') || sessionStorage.getItem('token')
            if (currentToken) {
              connect(currentToken)
            }
          }, 5000)
        }
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

    ElNotification({
      title: event.title || '系统即时通知',
      message: event.message,
      type: notifyType,
      duration: 9000,
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
  }

  return {
    isConnected,
    connect,
    disconnect,
    addEventListener,
  }
}
