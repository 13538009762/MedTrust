/**
 * 医学检验/检查报告与病历文本智能结构化解析器
 * 核心功能：
 * 1. 拆分复合检查报告（如同一就诊中的心电图、CT平扫、生化等）
 * 2. 优先提炼核心临床诊断与结论（Conclusion / Key Impression）
 * 3. 分离过程测量参数与观察所见（Findings & Measurements），支持折叠
 * 4. 智能归类医学异常/阳性指征标签（危急红/阳性黄/正常绿）
 * 5. 关键词检索精准反显高亮
 */

export interface MedicalAlertTag {
  text: string
  type: 'danger' | 'warning' | 'success' | 'info'
  level: number // 1: info/success, 2: warning, 3: danger
}

export interface ParsedExamSection {
  id: string
  title: string            // 检查大项名称，例如 "12导联心电图 (ECG)"
  subtitle?: string        // 模板或子类型，例如 "标准12导联静息心电图"
  categoryTag?: string     // 检查类别，如 "心电" | "CT影像" | "生化检验"
  conclusion: string       // 核心诊断结论（最重要信息！）
  findings: string[]       // 过程测量与观察参数列表（默认可折叠）
  suggestion?: string      // 处置或随访建议
  issuer?: string          // 出具医师与机构
  time?: string            // 报告时间
  tags: MedicalAlertTag[]  // 提取的关键标签
  isAbnormal: boolean      // 是否异常
  hasKeywordMatch: boolean // 是否包含搜索关键词
}

export interface ParsedMedicalReport {
  hasItems: boolean
  sections: ParsedExamSection[]
  summaryTags: MedicalAlertTag[] // 全局汇总标签（去重并按危险级别降序）
  isAnyAbnormal: boolean         // 是否含有任何异常
  hasKeywordMatch: boolean       // 是否命中搜索关键词
  pureSummary?: string           // 若非检查单，提炼的门诊病历核心摘要
}

/**
 * HTML 转义，防止 XSS
 */
export function escapeHtml(str: string): string {
  if (!str) return ''
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

/**
 * 关键词高亮
 */
export function highlightText(content: string, keyword?: string): string {
  if (!content) return ''
  const safeContent = escapeHtml(content)
  const kw = (keyword || '').trim()
  if (!kw) return safeContent

  try {
    // 对 keyword 进行安全转义后再正则替换
    const escapedKw = escapeHtml(kw).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    const regex = new RegExp(`(${escapedKw})`, 'gi')
    return safeContent.replace(regex, '<mark class="med-kw-mark">$1</mark>')
  } catch {
    return safeContent
  }
}

/**
 * 智能医学标签分类器
 */
export function extractAlertTags(text: string): { tags: MedicalAlertTag[]; isAbnormal: boolean } {
  const tags: MedicalAlertTag[] = []
  if (!text) return { tags, isAbnormal: false }

  const lower = text.toLowerCase()

  // 1. 危急 / 严重异常 / 强阳性 (Danger)
  if (lower.includes('心肌梗死') || lower.includes('心梗') || lower.includes('ami')) {
    tags.push({ text: '急性心梗预警', type: 'danger', level: 3 })
  }
  if (lower.includes('肌钙蛋白强阳性') || lower.includes('显著阳性') || (lower.includes('肌钙蛋白') && lower.includes('强阳性'))) {
    tags.push({ text: '肌钙蛋白强阳性', type: 'danger', level: 3 })
  }
  if (lower.includes('室性期前收缩') || lower.includes('室早') || lower.includes('室性早搏')) {
    tags.push({ text: '室性早搏', type: 'danger', level: 3 })
  }
  if (lower.includes('肾小球滤过功能受损') || lower.includes('肾功能不全') || lower.includes('肌酐显著升高')) {
    tags.push({ text: '肾功能受损', type: 'danger', level: 3 })
  }
  if (lower.includes('恶性') || lower.includes('占位性病变') || lower.includes('高度可疑癌')) {
    tags.push({ text: '占位/高度可疑', type: 'danger', level: 3 })
  }

  // 2. 预警 / 阳性发现 / 需专科关注 (Warning)
  if (lower.includes('心肌供血不足') || lower.includes('心肌缺血') || (lower.includes('st-t') && lower.includes('缺血'))) {
    tags.push({ text: '心肌供血不足', type: 'warning', level: 2 })
  }
  if (lower.includes('st-t') && !tags.some(t => t.text.includes('心肌供血不足'))) {
    tags.push({ text: 'ST-T改变', type: 'warning', level: 2 })
  }
  if (lower.includes('磨玻璃结节') || lower.includes('pggn') || lower.includes('ggn')) {
    tags.push({ text: '肺磨玻璃结节', type: 'warning', level: 2 })
  } else if (lower.includes('结节') && !lower.includes('未见明显结节') && !lower.includes('未见结节')) {
    tags.push({ text: '肺部结节', type: 'warning', level: 2 })
  }
  if (lower.includes('斑片状') || lower.includes('炎性渗出') || lower.includes('肺炎')) {
    tags.push({ text: '炎性渗出病变', type: 'warning', level: 2 })
  }
  if (lower.includes('细菌性感染') || lower.includes('细菌感染') || (lower.includes('白细胞') && lower.includes('增高'))) {
    tags.push({ text: '急性细菌感染', type: 'warning', level: 2 })
  }
  if (lower.includes('小细胞低色素') || lower.includes('贫血')) {
    tags.push({ text: '贫血指征', type: 'warning', level: 2 })
  }
  if (lower.includes('心动过速')) {
    tags.push({ text: '窦性心动过速', type: 'warning', level: 2 })
  }
  if (lower.includes('轻度异常') || lower.includes('轻微偏离')) {
    tags.push({ text: '指标轻度偏高', type: 'warning', level: 2 })
  }

  // 3. 建议随访
  if (lower.includes('建议') && (lower.includes('随访') || lower.includes('复查') || lower.includes('会诊'))) {
    tags.push({ text: '建议专科随访', type: 'info', level: 1 })
  }

  // 4. 正常 / 未见异常 (Success) - 仅在没有任何 danger 和 warning 时判定
  const isAbnormal = tags.some(t => t.level >= 2)
  if (!isAbnormal) {
    if (lower.includes('大致正常') || lower.includes('未见明显') || lower.includes('未见异常') || lower.includes('阴性正常') || lower.includes('符合健康生理') || (lower.includes('窦性心律') && !lower.includes('缺血') && !lower.includes('过速'))) {
      tags.push({ text: '大致正常/阴性', type: 'success', level: 1 })
    }
  }

  // 去重并按严重程度倒序排序
  const uniqueTagsMap = new Map<string, MedicalAlertTag>()
  tags.forEach(t => {
    if (!uniqueTagsMap.has(t.text)) uniqueTagsMap.set(t.text, t)
  })
  const sortedTags = Array.from(uniqueTagsMap.values()).sort((a, b) => b.level - a.level)

  return { tags: sortedTags, isAbnormal }
}

/**
 * 核心解析函数：将长文本 exam_result 或门诊诊断解析为结构化分块报告
 */
export function parseMedicalExamResult(rawText: string, searchKeyword: string = ''): ParsedMedicalReport {
  const result: ParsedMedicalReport = {
    hasItems: false,
    sections: [],
    summaryTags: [],
    isAnyAbnormal: false,
    hasKeywordMatch: false,
  }

  if (!rawText || !rawText.trim()) {
    return result
  }

  const text = rawText.trim()
  const kw = searchKeyword.trim().toLowerCase()
  if (kw && text.toLowerCase().includes(kw)) {
    result.hasKeywordMatch = true
  }

  // 检查是否包含 【...】 形式的医技检查标识
  const itemPattern = /【([^】]+)】/g
  const matches: Array<{ index: number; content: string }> = []
  let match: RegExpExecArray | null

  while ((match = itemPattern.exec(text)) !== null) {
    // 排除诸如 【第一人民医院】 出具人括号内的机构标签
    const prevSnippet = text.slice(Math.max(0, match.index - 10), match.index)
    const nextSnippet = text.slice(match.index + match[0].length, match.index + match[0].length + 15)
    if (prevSnippet.includes('(') && (nextSnippet.includes('出具人') || nextSnippet.includes('时间'))) {
      continue
    }
    matches.push({ index: match.index, content: match[1] })
  }

  // 如果没有匹配到任何标准的【项目名称】，当作单一综合文本处理
  if (matches.length === 0) {
    const { tags, isAbnormal } = extractAlertTags(text)
    result.pureSummary = text
    result.summaryTags = tags
    result.isAnyAbnormal = isAbnormal
    return result
  }

  // 存在结构化检查项，开始切分分段
  result.hasItems = true

  // 收集每个段落的起止位置
  const sectionRanges: Array<{ title: string; subtitle?: string; startIndex: number; endIndex: number }> = []

  for (let i = 0; i < matches.length; i++) {
    const current = matches[i]
    let title = current.content
    let subtitle: string | undefined
    let startIndex = current.index

    // 检查紧挨着是否是另一个【模板子项】，例如 【12导联心电图 (ECG)】【标准12导联静息心电图】
    if (i + 1 < matches.length) {
      const next = matches[i + 1]
      const gap = text.slice(current.index + current.content.length + 2, next.index).trim()
      if (gap === '') {
        // 紧挨着！
        subtitle = next.content
        i++ // 跳过下一个
      }
    }

    // 计算当前段落的结束位置
    let endIndex = text.length
    if (i + 1 < matches.length) {
      endIndex = matches[i + 1].index
    }

    sectionRanges.push({ title, subtitle, startIndex, endIndex })
  }

  // 解析各个 Section
  sectionRanges.forEach((sec, idx) => {
    const blockText = text.slice(sec.startIndex, sec.endIndex).trim()

    // 1. 提取出具医师和机构元数据
    let issuerInfo = ''
    let reportTime = ''
    const issuerMatch = blockText.match(/\(([^)]*出具人[^)]*)\)/)
    let cleanBlock = blockText
    if (issuerMatch) {
      issuerInfo = issuerMatch[1]
      cleanBlock = cleanBlock.replace(issuerMatch[0], '')
      // 提取时间
      const timeMatch = issuerInfo.match(/时间[:：]\s*(\d{4}-\d{2}-\d{2}\s*\d{2}:\d{2}(?::\d{2})?)/)
      if (timeMatch) {
        reportTime = timeMatch[1]
      }
    }

    // 2. 提取核心结论
    let conclusion = ''
    let suggestion = ''
    const conclusionMatch = cleanBlock.match(/结论[:：]\s*([^\n\r]+(?:(?!\n-|\n【).)*)/s)
    if (conclusionMatch) {
      conclusion = conclusionMatch[1].trim()
      cleanBlock = cleanBlock.replace(conclusionMatch[0], '')
    }

    // 3. 提取建议（若结论中包含建议，单独剥离或高亮）
    if (conclusion) {
      const sugMatch = conclusion.match(/(建议.*?[。；;！!]?$)/)
      if (sugMatch) {
        suggestion = sugMatch[1].trim()
      }
    }

    // 4. 清理标题后，提取过程测量参数与观察所见 (Findings)
    cleanBlock = cleanBlock.replace(new RegExp(`^【${sec.title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}】`), '')
    if (sec.subtitle) {
      cleanBlock = cleanBlock.replace(new RegExp(`^【${sec.subtitle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}】`), '')
    }

    // 提取条目
    const rawLines = cleanBlock.split('\n')
    const findings: string[] = []
    rawLines.forEach(line => {
      const trimmed = line.trim()
      if (!trimmed) return
      // 清除前缀破折号或项目符号
      const cleanLine = trimmed.replace(/^[-•*]\s*/, '').trim()
      if (cleanLine && cleanLine !== sec.title && cleanLine !== sec.subtitle) {
        findings.push(cleanLine)
      }
    })

    // 若未识别到结论词，但有明显的异常条目，提取首个重点作为结论
    if (!conclusion && findings.length > 0) {
      const keyLine = findings.find(f => f.includes('改变') || f.includes('结节') || f.includes('异常') || f.includes('提示') || f.includes('未见'))
      conclusion = keyLine || findings[findings.length - 1]
    }

    // 5. 提取标签
    const combinedForTags = `${sec.title} ${sec.subtitle || ''} ${conclusion} ${findings.join(' ')}`
    const { tags, isAbnormal } = extractAlertTags(combinedForTags)

    // 6. 是否命中检索词
    const hasSectionKw = !!(kw && (
      sec.title.toLowerCase().includes(kw) ||
      (sec.subtitle && sec.subtitle.toLowerCase().includes(kw)) ||
      conclusion.toLowerCase().includes(kw) ||
      findings.some(f => f.toLowerCase().includes(kw))
    ))

    // 7. 分类标签
    let categoryTag = '临床检查'
    const lowTitle = (sec.title + ' ' + (sec.subtitle || '')).toLowerCase()
    if (lowTitle.includes('心电') || lowTitle.includes('ecg')) categoryTag = '心电检查'
    else if (lowTitle.includes('ct') || lowTitle.includes('mri') || lowTitle.includes('平扫') || lowTitle.includes('影像')) categoryTag = '影像检查'
    else if (lowTitle.includes('血') || lowTitle.includes('化验') || lowTitle.includes('酶') || lowTitle.includes('生化')) categoryTag = '检验化验'

    result.sections.push({
      id: `sec-${idx}-${Math.random().toString(36).substring(2, 7)}`,
      title: sec.title,
      subtitle: sec.subtitle,
      categoryTag,
      conclusion: conclusion || '未出具明确异常结论',
      findings,
      suggestion,
      issuer: issuerInfo,
      time: reportTime,
      tags,
      isAbnormal,
      hasKeywordMatch: hasSectionKw,
    })
  })

  // 汇总所有 tags
  const allTagsMap = new Map<string, MedicalAlertTag>()
  result.sections.forEach(s => {
    s.tags.forEach(t => {
      if (!allTagsMap.has(t.text)) allTagsMap.set(t.text, t)
    })
    if (s.isAbnormal) result.isAnyAbnormal = true
    if (s.hasKeywordMatch) result.hasKeywordMatch = true
  })

  result.summaryTags = Array.from(allTagsMap.values()).sort((a, b) => b.level - a.level)
  return result
}
