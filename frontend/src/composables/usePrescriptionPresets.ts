/**
 * usePrescriptionPresets — DoctorRecordsView 处方快捷输入标准化模板与预设载入逻辑
 */
import { ElMessage } from 'element-plus'
import type { Ref } from 'vue'
import type { PrescriptionPreset } from '@/types/clinical'

export function usePrescriptionPresets(
  directForm: Ref<any>,
  finalForm: Ref<any>,
  generateDirectSOAP: () => void,
  generateSOAPRecord: () => void,
) {
  const prescriptionPresets: PrescriptionPreset[] = [
    {
      key: 'infection',
      label: '抗感染+对症退热规范处方',
      type: 'warning',
      diagnosis: '急性上呼吸道细菌性感染伴发热',
      etiology: '急性细菌性感染引发全身炎症反应，致热源激活下丘脑体温调节中枢，体温调定点上移。',
      plan: `【综合处置】
1. 建议发热期卧床休息，物理降温，保证每日温开水饮水量 >2000ml
2. 室内每日通风换气，清淡半流质饮食，3天后门诊复查血常规及CRP

【临床医嘱】
1. 密切监测体温变化，体温 >38.5℃ 时启动退热用药
2. 遵医嘱规律足疗程使用抗生素，严禁擅自停药或减量

【处方用药】
1. 阿莫西林克拉维酸钾片 (0.625g/片) 0.625g 口服 每日3次 餐中服用
2. 布洛芬缓释胶囊 (0.3g/粒) 0.3g 必要时口服 (体温>38.5℃使用，间隔≥6小时)
3. 复合维生素C泡腾片 (1g/片) 1片 每日1次 温水冲服`
    },
    {
      key: 'respiratory',
      label: '呼吸道对症止咳处方',
      type: 'primary',
      diagnosis: '急性支气管炎伴阵发性咳嗽',
      etiology: '病原体侵袭支气管黏膜引起充血水肿与分泌物渗出，支气管受体受刺激诱发剧烈咳嗽反射。',
      plan: `【综合处置】
1. 避免接触冷空气、花粉、粉尘及刺激性油烟烟雾，室内保持湿润
2. 清淡温热饮食，忌辛辣生冷，随访观察气道高反应性变化

【临床医嘱】
1. 注意防寒保暖，外出佩戴医用防护口罩
2. 若出现喘息加重、咳脓痰带血或持续高热，须立即急诊就诊

【处方用药】
1. 盐酸左氧氟沙星胶囊 (0.5g/粒) 0.5g 口服 每日1次 饭前温水送服
2. 乙酰半胱氨酸泡腾片 (0.6g/片) 0.6g 口服 每日2次 溶于半杯温开水服
3. 复方甘草口服溶液 (100ml/瓶) 10ml 口服 每日3次 咳嗽发作时服用`
    },
    {
      key: 'cardio',
      label: '心血管慢病规范处方',
      type: 'success',
      diagnosis: '原发性高血压病2级（很高危组）伴心肌缺血',
      etiology: '长期外周阻力血管痉挛硬化，交感神经过度激活，心脏后负荷持续增加伴心肌供血失衡。',
      plan: `【综合处置】
1. 长期严格低盐低脂饮食 (每日食盐摄入量严格控制在 5g 以内)
2. 保持生活作息规律与情绪平稳，严禁剧烈运动、熬夜及重体力劳累

【临床医嘱】
1. 每日晨起排尿后及晚间睡前规范测量双侧坐位血压，如实记录慢病日志
2. 规律服用长效降压药与降脂稳定斑块药，切忌自行停药造成血压反跳

【处方用药】
1. 苯磺酸氨氯地平片 (络活喜 5mg/片) 5mg 口服 每日1次 清晨空腹顿服
2. 阿托伐他汀钙片 (立普妥 20mg/片) 20mg 口服 每晚睡前1次
3. 琥珀酸美托洛尔缓释片 (倍他乐克 47.5mg/片) 47.5mg 口服 每日1次 清晨温水整片吞服`
    }
  ]

  function applyPrescriptionPreset(preset: PrescriptionPreset, target: string) {
    if (target === 'DIRECT') {
      directForm.value.treatment_plan = preset.plan
      if (!directForm.value.diagnosis || directForm.value.diagnosis.includes('待查')) {
        directForm.value.diagnosis = preset.diagnosis
      }
      if (!directForm.value.etiology) {
        directForm.value.etiology = preset.etiology
      }
      generateDirectSOAP()
    } else {
      finalForm.value.treatment_plan = preset.plan
      if (!finalForm.value.diagnosis || finalForm.value.diagnosis.includes('待查')) {
        finalForm.value.diagnosis = preset.diagnosis
      }
      if (!finalForm.value.etiology || finalForm.value.etiology.includes('需抽血')) {
        finalForm.value.etiology = preset.etiology
      }
      generateSOAPRecord()
    }
    ElMessage.success(`已载入「${preset.label}」规范处方与处置医嘱`)
  }

  // ── 常规高血压门诊处方示例 ──
  function fillQuickDirectPreset() {
    ElMessage.success('已载入心血管内科高血压/冠心病规范处方与体征示例！')
  }

  return {
    prescriptionPresets,
    applyPrescriptionPreset,
    fillQuickDirectPreset,
  }
}