// WP-16: 视口尺寸自适应指标 (从 CrossQueryView 迁入, 行为保持原样)
import { ref, computed, onMounted, onUnmounted } from 'vue'

export function useViewportMetrics() {
  const windowWidth = ref(window.innerWidth)
  const windowHeight = ref(window.innerHeight)

  function handleResize() {
    windowWidth.value = window.innerWidth
    windowHeight.value = window.innerHeight
  }

  const isWideScreen = computed(() => windowWidth.value >= 1360)

  // 动态自适应表格最大高度：在笔记本或小屏幕上保证底部的水平拖动条和分页栏始终在屏幕内
  const tableMaxHeight = computed(() => {
    const h = windowHeight.value - 410
    return Math.max(h, 320) + 'px'
  })

  onMounted(() => {
    handleResize()
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
  })

  return { isWideScreen, tableMaxHeight }
}