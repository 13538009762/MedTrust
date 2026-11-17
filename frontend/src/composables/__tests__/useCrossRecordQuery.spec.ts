import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { reactive, computed, nextTick } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { CrossQueryScope } from '../useCrossQueryScope'

const { listRecordsApiMock, elNotificationMock, hooks } = vi.hoisted(() => ({
  listRecordsApiMock: vi.fn(),
  elNotificationMock: vi.fn(),
  hooks: {
    mounted: [] as Array<() => void>,
    unmounted: [] as Array<() => void>,
  },
}))

vi.mock('vue', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue')>()
  return {
    ...actual,
    onMounted: (fn: () => void) => {
      hooks.mounted.push(fn)
    },
    onBeforeUnmount: (fn: () => void) => {
      hooks.unmounted.push(fn)
    },
  }
})

vi.mock('element-plus', () => ({
  ElNotification: elNotificationMock,
}))

vi.mock('../../api', () => ({
  listRecordsApi: listRecordsApiMock,
}))

vi.mock('../../stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 10, hospital_id: 1 },
  }),
}))

import { useCrossRecordQuery } from '../useCrossRecordQuery'

const sampleRecords = [
  { id: 1, hospital_id: 1, doctor_id: 10, has_access: true, access_type: 'LOCAL' },
  { id: 2, hospital_id: 2, doctor_id: 20, has_access: true, access_type: 'AUTHORIZED' },
  { id: 3, hospital_id: 2, doctor_id: 21, has_access: false, access_type: 'UNAUTHORIZED' },
]

function createMockRoute(scope: CrossQueryScope) {
  return reactive({
    path: `/doctor/query/${scope}`,
    params: { scope },
    query: {},
  }) as unknown as RouteLocationNormalizedLoaded
}

async function flushMicrotasks() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

describe('useCrossRecordQuery transition & lifecycle optimization', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    hooks.mounted.length = 0
    hooks.unmounted.length = 0
    listRecordsApiMock.mockResolvedValue({ code: 200, data: sampleRecords })
  })

  afterEach(() => {
    vi.advanceTimersByTime(10_000)
    vi.useRealTimers()
  })

  it('starts request on mount in parallel with enter transition and holds table update until transition settles', async () => {
    const route = createMockRoute('all')
    const currentScope = computed(() => route.params.scope as CrossQueryScope)
    let releaseTransition!: () => void
    const settlePromise = new Promise<void>((resolve) => {
      releaseTransition = resolve
    })

    const q = useCrossRecordQuery(route, currentScope, {
      waitForEnterTransition: () => settlePromise,
    })

    // Setup should not eagerly fire network calls or activate v-loading before mount
    expect(listRecordsApiMock).not.toHaveBeenCalled()
    expect(q.loading.value).toBe(false)

    // Mounting triggers the single unfiltered 'all' request immediately while transition runs
    hooks.mounted.forEach((fn) => fn())
    await flushMicrotasks()

    expect(listRecordsApiMock).toHaveBeenCalledTimes(1)
    expect(q.results.value).toHaveLength(0)

    // Once the 250ms enter transition completes, buffered data commits cleanly in one batch
    releaseTransition()
    await flushMicrotasks()

    expect(q.results.value).toHaveLength(3)
    expect(q.statsCount.value).toEqual({
      all: 3,
      local: 1,
      authorized: 1,
      pending: 1,
    })
    expect(q.loading.value).toBe(false)
  })

  it('prevents the dying instance from refetching when route changes during leave transition', async () => {
    const route = createMockRoute('authorized')
    const currentScope = computed(() => route.params.scope as CrossQueryScope)

    const q = useCrossRecordQuery(route, currentScope)
    hooks.mounted.forEach((fn) => fn())
    await flushMicrotasks()

    const callsAfterMount = listRecordsApiMock.mock.calls.length

    // Simulate clicking another query tab in <transition mode="out-in">: route updates while old instance leaves
    route.path = '/doctor/query/pending'
    route.params.scope = 'pending'
    await flushMicrotasks()

    expect(listRecordsApiMock).toHaveBeenCalledTimes(callsAfterMount)
    expect(q.loading.value).toBe(false)

    hooks.unmounted.forEach((fn) => fn())
  })

  it('reuses recent stats cache across tab switches and clears it on refreshData', async () => {
    const routeAll = createMockRoute('all')
    const q1 = useCrossRecordQuery(routeAll, computed(() => routeAll.params.scope as CrossQueryScope))
    hooks.mounted.forEach((fn) => fn())
    await flushMicrotasks()
    hooks.unmounted.forEach((fn) => fn())

    vi.clearAllMocks()
    hooks.mounted.length = 0
    hooks.unmounted.length = 0
    listRecordsApiMock.mockResolvedValue({ code: 200, data: sampleRecords })

    // Switching to 'local' within TTL seeds counts synchronously and fires only 1 scoped query
    const routeLocal = createMockRoute('local')
    const q2 = useCrossRecordQuery(routeLocal, computed(() => routeLocal.params.scope as CrossQueryScope))
    expect(q2.statsCount.value).toEqual({ all: 3, local: 1, authorized: 1, pending: 1 })

    hooks.mounted.forEach((fn) => fn())
    await flushMicrotasks()
    expect(listRecordsApiMock).toHaveBeenCalledTimes(1)
    expect(q2.results.value).toHaveLength(1)

    // refreshData invalidates cache and refreshes both stats and records
    q2.refreshData()
    await flushMicrotasks()
    expect(listRecordsApiMock).toHaveBeenCalledTimes(3)
    void q1
  })
})
