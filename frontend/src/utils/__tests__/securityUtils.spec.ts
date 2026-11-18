import { describe, it, expect, vi, beforeEach } from 'vitest'
import { highlightText } from '../medicalReportParser'
import { buildSecureFileUrl } from '../secureFileUrl'
import { scanTextForInfections } from '../infectionGuard'

vi.mock('@/api/auth', () => ({
  createSSETicketApi: vi.fn(),
}))

import { createSSETicketApi } from '@/api/auth'

describe('Frontend Security Regression', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    const mem: Record<string, string> = {}
    vi.stubGlobal('sessionStorage', {
      getItem: (k: string) => mem[k] ?? null,
      setItem: (k: string, v: string) => {
        mem[k] = v
      },
      removeItem: (k: string) => {
        delete mem[k]
      },
      clear: () => {
        for (const k of Object.keys(mem)) delete mem[k]
      },
    })
  })

  it('highlightText must escape XSS payloads before adding highlight mark', () => {
    const xss = '<img src=x onerror="alert(1)"> 高血压'
    const output = highlightText(xss, '高血压')
    expect(output).not.toContain('<img')
    expect(output).toContain('&lt;img src=x onerror=&quot;alert(1)&quot;&gt;')
    expect(output).toContain('<mark class="med-kw-mark">高血压</mark>')
  })

  it('buildSecureFileUrl issues one-time ticket URL when ticket API succeeds', async () => {
    vi.mocked(createSSETicketApi).mockResolvedValueOnce({
      code: 200,
      message: 'ok',
      data: { ticket: 'tkt_abc123', expires_in: 60 },
    })
    const url = await buildSecureFileUrl('/medical-files/42/view')
    expect(url).toBe('/api/v1/medical-files/42/view?ticket=tkt_abc123')
  })

  it('scanTextForInfections ignores negated results and ASCII substring false positives', () => {
    const safeText = '患者HIV抗体阴性，HBsAg(-)，否认肺结核病史，血清creatinine正常'
    const res = scanTextForInfections(safeText)
    expect(res.has_risk).toBe(false)

    const mixed = '活动性肺结核，HIV(-)'
    const mixedRes = scanTextForInfections(mixed)
    expect(mixedRes.has_risk).toBe(true)
    expect(mixedRes.diseases).toHaveLength(1)
    expect(mixedRes.diseases[0].matched_keyword).toBe('肺结核')
  })
})
