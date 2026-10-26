export interface ClinicalTemplate {
  dept: string
  hosp: string
  doc: string
  type: string
  icd: string
  diag: string
  symptoms: string
  vitals: string
  plan: string
}

export const clinicalTemplates: ClinicalTemplate[] = [
  {
    dept: '心血管内科门诊',
    hosp: '广州第一附属医院',
    doc: '陈建明 (主任医师)',
    type: 'OUTPATIENT',
    icd: 'I25.1',
    diag: '冠状动脉粥样硬化性心脏病 · 心功能 II 级',
    symptoms: '发作性心前区闷痛伴胸闷气促3天，体力活动及劳累后加剧',
    vitals: 'T: 36.6℃, BP: 138/85 mmHg, HR: 76 bpm, SpO2: 98%',
    plan: '1. 规范口服拜新同 30mg qd、立普妥 20mg qn；2. 低盐低脂清淡饮食；3. 2周后心内科复查心电图与心肌酶。'
  },
  {
    dept: '急诊医学科',
    hosp: '深圳中心医院',
    doc: '李国强 (副主任医师)',
    type: 'EMERGENCY',
    icd: 'R07.4',
    diag: '急性胸痛查因 (心肌缺血待排)',
    symptoms: '突发胸骨后压榨样疼痛半小时，伴冷汗，向左肩背部放射',
    vitals: 'T: 36.8℃, BP: 145/92 mmHg, HR: 88 bpm, SpO2: 97%',
    plan: '1. 卧床吸氧，持续心电监护；2. 急查12导联ECG与肌钙蛋白I；3. 舌下含服硝酸甘油片0.5mg。'
  },
  {
    dept: '呼吸与危重症医学科',
    hosp: '广州第一附属医院',
    doc: '林雅茹 (副主任医师)',
    type: 'OUTPATIENT',
    icd: 'J06.9',
    diag: '急性上呼吸道感染伴咽支气管充血',
    symptoms: '咽干咽痛伴阵发性干咳2天，畏寒低热，无呼吸困难',
    vitals: 'T: 37.4℃, BP: 122/78 mmHg, HR: 82 bpm, SpO2: 99%',
    plan: '1. 蒲地蓝消炎口服液 10ml tid；2. 复方甘草合剂止咳；3. 嘱多饮温水，注意防寒保暖，居家观察。'
  },
  {
    dept: '心血管内科专科随访',
    hosp: '珠海人民医院',
    doc: '张伟 (主治医师)',
    type: 'OUTPATIENT',
    icd: 'I10',
    diag: '原发性高血压病2级（中危组随访）',
    symptoms: '常规门诊血压随访，晨起偶感头晕枕后钝痛，无视物旋转',
    vitals: 'T: 36.5℃, BP: 142/88 mmHg, HR: 72 bpm, SpO2: 98%',
    plan: '1. 调整拜新同剂量为每日清晨 30mg；2. 监测早晚双臂血压并记健康日记；3. 限制钠盐摄入(<5g/d)。'
  },
  {
    dept: '消化内科门诊',
    hosp: '广州第一附属医院',
    doc: '黄志远 (主任医师)',
    type: 'OUTPATIENT',
    icd: 'K29.7',
    diag: '慢性浅表性胃炎伴反流性食管炎',
    symptoms: '反酸烧心伴剑突下饱胀隐痛1周，餐后明显，夜间平卧加重',
    vitals: 'T: 36.6℃, BP: 125/80 mmHg, HR: 74 bpm, SpO2: 99%',
    plan: '1. 奥美拉唑肠溶胶囊 20mg bid 空腹口服；2. 铝碳酸镁片咀嚼片餐后咀嚼；3. 忌辛辣生冷与咖啡浓茶。'
  }
]

export function getIcdCode(diag?: string): string {
  if (!diag) return 'Z00.0'
  if (diag.includes('冠心') || diag.includes('冠状动脉')) return 'I25.1'
  if (diag.includes('高血压')) return 'I10'
  if (diag.includes('呼吸') || diag.includes('感冒') || diag.includes('咽')) return 'J06.9'
  if (diag.includes('胸痛')) return 'R07.4'
  if (diag.includes('胃炎') || diag.includes('胃溃疡') || diag.includes('消化')) return 'K29.7'
  if (diag.includes('糖尿病')) return 'E11.9'
  if (diag.includes('骨折') || diag.includes('外伤')) return 'S82.9'
  return 'Z00.0'
}
