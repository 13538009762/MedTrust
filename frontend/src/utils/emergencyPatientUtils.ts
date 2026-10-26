/**
 * 急诊患者搜索、致敏史判定与院感防护等级格式化工具
 */

export function hasRealAllergy(candOrAllergies?: any): boolean {
  if (!candOrAllergies) return false
  if (typeof candOrAllergies === 'object') {
    if (typeof candOrAllergies.has_allergies === 'boolean') {
      return candOrAllergies.has_allergies
    }
    return hasRealAllergy(candOrAllergies.allergies)
  }
  const t = String(candOrAllergies).trim()
  if (!t) return false
  const safeKeywords = ['未发现', '暂未', '无', '暂无', '未见', '阴性', '否认', '未报告']
  if (safeKeywords.some(kw => t.includes(kw))) {
    return false
  }
  return true
}

export function formatProtectionLevel(alert?: any): string {
  if (!alert) return '二级防护'
  const lvl = alert.level || (alert.risk_level === 'CRITICAL' ? '三级' : '二级')
  if (typeof lvl === 'string') {
    if (lvl.endsWith('级防护')) return lvl
    if (lvl.endsWith('级')) return `${lvl}防护`
    if (lvl === '2' || lvl === '二') return '二级防护'
    if (lvl === '3' || lvl === '三') return '三级防护'
    if (lvl === '1' || lvl === '一') return '一级防护'
  }
  return '二级防护'
}

export function formatInfectionSummary(text?: string): string {
  if (!text) return '既往/长期携带病史'
  return text.replace(/必定\/长期携带病史/g, '既往/长期携带病史').replace(/必定携带/g, '长期携带')
}
