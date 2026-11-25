export type VitalStatusLevel = 'normal' | 'elevated' | 'critical' | 'unknown'

export interface VitalBadge {
  level: VitalStatusLevel
  label: string
}

/** 从字符串中提取第一个合法数值 */
function parseNum(raw?: string | number | null): number | null {
  if (raw === null || raw === undefined) return null
  const m = String(raw).match(/-?\d+(?:\.\d+)?/)
  if (!m) return null
  const n = Number(m[0])
  return Number.isFinite(n) ? n : null
}

/** 评估体温 (℃) */
export function evaluateTemperature(raw?: string | number | null): VitalBadge {
  const val = parseNum(raw)
  if (val === null) return { level: 'unknown', label: '待测 (参考 36.0~37.2℃)' }
  if (val >= 39.0 || val < 35.0) return { level: 'critical', label: val >= 39.0 ? '高热预警' : '低体温预警' }
  if (val > 37.2 || val < 36.0) return { level: 'elevated', label: val > 37.2 ? '↑ 偏高 (参考 36.0~37.2℃)' : '↓ 偏低 (参考 36.0~37.2℃)' }
  return { level: 'normal', label: '正常 (36.0~37.2℃)' }
}

/** 评估血压 (mmHg，格式如 125/80) */
export function evaluateBloodPressure(raw?: string | null): VitalBadge {
  if (!raw || !String(raw).trim()) return { level: 'unknown', label: '待测 (参考 <130/80)' }
  const m = String(raw).match(/(\d+(?:\.\d+)?)\s*\/\s*(\d+(?:\.\d+)?)/)
  if (!m) return { level: 'unknown', label: '格式待核对 (如 120/80)' }
  const sbp = Number(m[1])
  const dbp = Number(m[2])
  if (!Number.isFinite(sbp) || !Number.isFinite(dbp)) return { level: 'unknown', label: '待核对' }
  if (sbp >= 180 || dbp >= 120) return { level: 'critical', label: '危象高危 (≥180/120)' }
  if (sbp < 90 || dbp < 60) return { level: 'critical', label: '低血压预警 (<90/60)' }
  if (sbp >= 140 || dbp >= 90) return { level: 'elevated', label: '↑ 偏高 (≥140/90)' }
  if (sbp >= 130 || dbp >= 80) return { level: 'elevated', label: '正常高值 (130~139/80~89)' }
  return { level: 'normal', label: '正常 (<130/80)' }
}

/** 评估心率/脉搏 (bpm) */
export function evaluateHeartRate(raw?: string | number | null): VitalBadge {
  const val = parseNum(raw)
  if (val === null) return { level: 'unknown', label: '待测 (参考 60~100 bpm)' }
  if (val >= 130 || val < 45) return { level: 'critical', label: val >= 130 ? '严重心动过速' : '严重心动过缓' }
  if (val > 100 || val < 60) return { level: 'elevated', label: val > 100 ? '↑ 偏快 (>100 bpm)' : '↓ 偏慢 (<60 bpm)' }
  return { level: 'normal', label: '正常 (60~100 bpm)' }
}

/** 评估血氧饱和度 (SpO2 %) */
export function evaluateSpo2(raw?: string | number | null): VitalBadge {
  const val = parseNum(raw)
  if (val === null) return { level: 'unknown', label: '待测 (参考 ≥95%)' }
  if (val < 90) return { level: 'critical', label: '重度低氧 (<90%)' }
  if (val < 95) return { level: 'elevated', label: '↓ 偏低 (90~94%)' }
  return { level: 'normal', label: '正常 (≥95%)' }
}

/**
 * 解析床旁生命体征整段文本并生成真实状态标签（仅对真实解析出的异常或正常值生成标签）
 */
export function evaluateVitalSummaryTags(rawText?: string | null): Array<{ text: string; level: VitalStatusLevel }> {
  if (!rawText || !rawText.trim()) return []
  const text = rawText.trim()
  const tags: Array<{ text: string; level: VitalStatusLevel }> = []

  const bpMatch = text.match(/(?:BP|血压)\s*[:：]?\s*(\d+\s*\/\s*\d+)/i)
  if (bpMatch) {
    const res = evaluateBloodPressure(bpMatch[1])
    if (res.level !== 'unknown') tags.push({ text: `血压${res.level === 'normal' ? '正常' : res.label}`, level: res.level })
  }

  const spo2Match = text.match(/(?:SpO2|血氧)\s*[:：]?\s*(\d+(?:\.\d+)?)/i)
  if (spo2Match) {
    const res = evaluateSpo2(spo2Match[1])
    if (res.level !== 'unknown') tags.push({ text: `血氧${res.level === 'normal' ? '正常' : res.label}`, level: res.level })
  }

  const hrMatch = text.match(/(?:HR|心率)\s*[:：]?\s*(\d+(?:\.\d+)?)/i)
  if (hrMatch) {
    const res = evaluateHeartRate(hrMatch[1])
    if (res.level !== 'unknown') tags.push({ text: `心率${res.level === 'normal' ? '正常' : res.label}`, level: res.level })
  }

  const tMatch = text.match(/(?:^|[\s,，;；])(?:T|体温)\s*[:：]?\s*(\d+(?:\.\d+)?)/i)
  if (tMatch) {
    const res = evaluateTemperature(tMatch[1])
    if (res.level !== 'unknown') tags.push({ text: `体温${res.level === 'normal' ? '正常' : res.label}`, level: res.level })
  }

  return tags
}
