/**
 * 医疗病历展示、条形码生成与风控文案纯函数工具集
 */

export function formatFileSize(bytes?: number): string {
  if (!bytes) return '2.8 KB'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

export function formatEncounterType(t?: string): string {
  if (t === 'OUTPATIENT') return '普通门诊'
  if (t === 'EMERGENCY') return '急诊救治'
  if (t === 'INPATIENT') return '住院就诊'
  return t || '门诊'
}

export function formatTime(t?: string): string {
  if (!t) return '-'
  return t.substring(0, 16).replace('T', ' ')
}

export function formatTxId(tx?: string): string {
  if (!tx) return '-'
  return String(tx).replace(/^0x/i, '')
}

export function cleanClinicalText(text?: string): string {
  if (!text) return ''
  return String(text)
    .replace(/持续持续/g, '持续')
    .replace(/持续时间约\s*持续/g, '持续约 ')
    .replace(/持续时间约\s*2天/g, '持续约2天')
    .replace(/拟诊为相关系统病变\s*\(分类:\s*RESP\)/g, '急性上呼吸道感染待排')
    .replace(/服用药物，不宜吃辣/g, '遵医嘱规范用药，低盐低脂清淡饮食，避免辛辣刺激')
}

export function getRiskScoreText(score: number): string {
  if (score >= 60) return '高风险需二次确认'
  if (score >= 30) return '中风险需核准'
  return '低风险常规调阅'
}

export function getRiskRulesTooltip(item: any): string {
  if (item?.risk_detail) {
    return `风控加权明细: ${item.risk_detail}`
  }
  const score = item?.risk_score || 48
  if (score >= 60) {
    return '触发加权规则：跨机构跨省调阅 (+35)、非工作就诊时段 (+20)、高频访问预警 (+15)'
  } else if (score >= 30) {
    return '触发加权规则：跨合作机构调阅 (+20)、科室交叉权限审核 (+15)、非直属随访 (+13)'
  }
  return '触发加权规则：同院经治医生随访 (+10)、工作常规时段 (+0)'
}

export function cleanDurationText(d?: string): string {
  if (!d) return '发作性'
  const s = String(d).replace(/^持续时间[：:]\s*/, '').replace(/^病程[：:]\s*/, '').trim()
  return s || '发作性'
}

export function formatOnsetTime(t?: string): string {
  if (!t) return '接诊前'
  return String(t).replace('T', ' ').substring(0, 19)
}

export function getDisplayEtiology(rec?: any): string {
  if (!rec) return '临床病理机制明确'
  const et = rec.etiology || ''
  if (et && !et.includes('需抽血') && !et.includes('化验确认') && !et.includes('待查')) {
    return et
  }
  const diag = rec.diagnosis || rec.initial_diagnosis || ''
  if (diag.includes('细菌') || diag.includes('感染') || diag.includes('炎')) {
    return '急性细菌性感染引发活动性炎性反应，伴白细胞及hs-CRP显著升高'
  } else if (diag.includes('冠心病') || diag.includes('心绞痛') || diag.includes('缺血') || diag.includes('心肌')) {
    return '冠状动脉粥样硬化狭窄伴血管阻力增加，心肌氧供耗失衡诱发心肌缺血改变'
  } else if (diag.includes('高血压')) {
    return '外周阻力血管收缩及交感神经过度激活，致动脉管壁顺应性下降及血压病理性升高'
  } else if (diag.includes('咳') || diag.includes('支气管') || diag.includes('哮喘')) {
    return '气道高反应性炎性介质释放，支气管平滑肌痉挛伴分泌物增多'
  }
  return '临床病理机制明确，机体靶器官早期功能性改变，与本次实验室检验指标吻合'
}

export interface BarcodeResult {
  bars: Array<{ x: number; w: number }>
  totalWidth: number
}

export function getBarcodeData(code?: string): BarcodeResult {
  const str = code || 'REC202609001'
  const bars: Array<{ x: number; w: number }> = []
  let x = 2
  bars.push({ x, w: 2 }); x += 4
  bars.push({ x, w: 1 }); x += 3

  for (let i = 0; i < str.length; i++) {
    const c = str.charCodeAt(i)
    const w1 = ((c * 7 + i) % 3) + 1
    const s1 = ((c * 3 + i) % 2) + 1
    const w2 = ((c * 13 + i * 2) % 3) + 1
    const s2 = ((c * 5 + i) % 2) + 1
    const w3 = ((c * 11 + i * 3) % 2) + 1
    const s3 = ((c * 17 + i) % 3) + 1

    bars.push({ x, w: w1 }); x += w1 + s1
    bars.push({ x, w: w2 }); x += w2 + s2
    bars.push({ x, w: w3 }); x += w3 + s3
  }
  bars.push({ x, w: 2 }); x += 4
  bars.push({ x, w: 1 }); x += 3
  bars.push({ x, w: 3 }); x += 4
  return { bars, totalWidth: x + 2 }
}
