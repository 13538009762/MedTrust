// 医护职业安全防范与高危传染病识别引擎 (Infection Guard)
// 覆盖国家卫健委《血源性病原体职业接触防护导则》与高危传染病防护标准

export interface InfectionDisease {
  name: string
  category: string
  persistence: string
  threat_to_doctor: string
  transmission: string
  risk_level: 'CRITICAL' | 'HIGH' | 'MEDIUM_HIGH' | 'SAFE'
  matched_keyword: string
  record_no?: string
  hospital_name?: string
  record_date?: string
}

export interface InfectionRiskAlert {
  has_risk: boolean
  risk_level: 'CRITICAL' | 'HIGH' | 'MEDIUM_HIGH' | 'SAFE'
  summary: string
  diseases: InfectionDisease[]
  precautions: string[]
  protection_gear: string[]
  emergency_steps: string[]
  detected_from?: string[]
}

// 临床高危传染病及医护安全威胁规则库
const INFECTION_RULES = [
  {
    name: '获得性免疫缺陷综合征 / 艾滋病 (HIV / AIDS)',
    category: '血源与体液极高危传播',
    persistence: '必定终身携带 (Lifelong Carriage)',
    threat_to_doctor: '锐器针刺伤暴露、血液黏膜喷溅暴露导致医护感染HIV，致死性极高',
    transmission: '血液、体液、黏膜破损、利器穿刺',
    risk_level: 'CRITICAL' as const,
    keywords: [
      '艾滋病', 'hiv', 'aids', '获得性免疫缺陷', '人类免疫缺陷病毒',
      'hiv抗体阳性', 'hiv-ab(+)', 'hiv阳性', 'hiv感染', '免疫缺陷病毒'
    ],
    precautions: [
      '严格执行标准预防与双层屏障防护，侵入性或抽血操作必须穿戴双层手套与防护面屏/护目镜',
      '严禁双手回套针帽，注射与穿刺完成后必须立即将锐器弃入就近防穿刺利器收集盒',
      '手术或有创操作中禁止徒手直接传递刀片或缝针，必须使用无接触中转盘传递',
      '若发生针刺伤或破损皮肤黏膜暴露，立即执行“一挤二冲三消毒”并在2小时内启动PEP药物阻断'
    ],
    protection_gear: [
      '医用防护口罩 (N95/KN95)',
      '双层无粉灭菌乳胶手套',
      '防喷溅防护面屏 / 护目镜',
      '一次性防渗透隔离衣',
      '自毁型安全留置针与防针刺针具'
    ],
    emergency_steps: [
      '挤血：从近心端向远心端轻轻挤压伤口，尽量挤出损伤处血液，严禁局部直接按压挤捏',
      '冲洗：立即使用流动的生理盐水或流动肥皂清水反复彻底冲洗伤口至少 5 分钟',
      '消毒：用 75% 医用乙醇或 0.5% 聚维酮碘消毒伤口局部，并包扎保护',
      '上报与用药：立即向院感科紧急报备，并在暴露后 2 小时内尽早服用 PEP 阻断抗病毒药物 (最长不超 72 小时)'
    ]
  },
  {
    name: '慢性乙型病毒性肝炎 (Chronic Hepatitis B / HBV)',
    category: '血源与体液高危传播',
    persistence: '长期/终身慢性携带 (Chronic Persistent Carriage)',
    threat_to_doctor: '针刺伤传播率高达 6%~30%，易导致医护人员爆发性肝炎或慢性肝脏疾病',
    transmission: '血液、体液、注射针刺、黏膜破损',
    risk_level: 'HIGH' as const,
    keywords: [
      '乙型肝炎', '乙肝', 'hbv', 'hbsag(+)', 'hbsag阳性', '乙肝表面抗原阳性',
      '大三阳', '小三阳', '乙型病毒性肝炎', '慢性乙肝', '乙肝病毒携带'
    ],
    precautions: [
      '医护人员接诊前应确认自身已完成乙肝疫苗接种并具备抗-HBs保护性抗体 (滴度 ≥ 10 mIU/mL)',
      '进行静脉采血、穿刺、拔针或换药操作时必须规范佩戴乳胶手套，避免徒手接触患者血液',
      '锐器操作全程防范刺伤，所有污染利器严格定点入盒',
      '一旦发生职业暴露且抗体阴性，应在24小时内注射乙肝高效价免疫球蛋白(HBIG)并补种疫苗'
    ],
    protection_gear: [
      '医用外科/防护口罩',
      '双层乳胶医用检查手套',
      '防溅眼罩 (有创喷溅风险操作时)',
      '防刺锐器收集盒'
    ],
    emergency_steps: [
      '立即挤出伤口伤处污血，流动清水及肥皂液冲洗 10 分钟',
      '0.5% 聚维酮碘或 75% 酒精涂抹消毒',
      '急查暴露医护人员抗-HBs抗体水平',
      '若抗体滴度不足 10 mIU/mL，24 小时内肌注乙肝免疫球蛋白 (HBIG) 200~400 IU 并续种疫苗'
    ]
  },
  {
    name: '慢性丙型病毒性肝炎 (Chronic Hepatitis C / HCV)',
    category: '血源与体液传播',
    persistence: '长期慢性携带 (70%~80%慢性化率)',
    threat_to_doctor: '锐器刺伤或破损皮肤接触含有病毒血液，易引起慢性肝损伤',
    transmission: '血液、深部穿刺针刺暴露',
    risk_level: 'HIGH' as const,
    keywords: [
      '丙型肝炎', '丙肝', 'hcv', '抗-hcv(+)', '抗-hcv阳性', 'hcv-rna阳性',
      '丙型病毒性肝炎', '慢性丙肝', '丙肝抗体阳性'
    ],
    precautions: [
      '严格落实侵入性操作标准防护，接触血液及体液必须佩戴手套，若破损立即更换',
      '强化利器规范化传递流程，杜绝危险徒手操作',
      '发生暴露后由于尚无疫苗与免疫球蛋白，需立即冲洗消毒并在4~6周内监测HCV RNA与ALT'
    ],
    protection_gear: [
      '医用外科口罩',
      '医用乳胶手套',
      '护目镜 (喷溅高危环境)'
    ],
    emergency_steps: [
      '近心端向远心端挤血，清水充分冲洗伤口',
      '碘伏消毒并贴无菌防水辅料',
      '暴露当时、第2周、第4周分别检测基线与动态 HCV RNA 载量，必要时启动直接抗病毒药物 (DAA)'
    ]
  },
  {
    name: '活动性/开放性肺结核 (Active Pulmonary TB / MDR-TB)',
    category: '空气与呼吸道气溶胶烈性传播',
    persistence: '活动期持续排菌/病程迁延',
    threat_to_doctor: '患者咳嗽、咳痰、打喷嚏产生飞沫气溶胶，诊室密闭环境下医护极易吸入感染',
    transmission: '空气飞沫微粒吸入、气溶胶传播',
    risk_level: 'HIGH' as const,
    keywords: [
      '肺结核', '开放性结核', '活动性结核', '结核分枝杆菌', '耐药结核',
      'mdr-tb', '涂阳肺结核', '结核性胸膜炎', '空洞型肺结核'
    ],
    precautions: [
      '就诊与查体期间指导患者佩戴医用外科口罩，减少飞沫扩散',
      '接诊医护人员必须严格佩戴密合性优良的 N95 / KN95 医用防护口罩',
      '诊室必须保持强化机械通风或开启负压层流通风设施 (换气次数 ≥ 12 ACH)',
      '尽量避免在非负压环境下开展咽拭子、吸痰或诱导排痰等高气溶胶操作'
    ],
    protection_gear: [
      '医用防护口罩 (N95/KN95 级，气密合格)',
      '医用隔离衣 / 防护面屏',
      '一次性医用检查手套'
    ],
    emergency_steps: [
      '若发生面部无防护近距离剧烈呛咳飞沫暴露，立即移步通风良好处流动水清洗眼部及口鼻',
      '向院感与疾控科登记报备，建立潜伏感染随访档案',
      '暴露后第 8~12 周行结核菌素皮肤试验 (PPD) 或 γ-干扰素释放试验 (IGRA)，异常时行预防性抗结核治疗'
    ]
  },
  {
    name: '梅毒 (Syphilis / 现症或晚期活动期)',
    category: '接触与体液黏膜传播',
    persistence: '血清固定/持续感染 (Untreated or Chronic Infection)',
    threat_to_doctor: '直接接触患者破损硬下疳、梅毒疹溃疡或血液导致医源性感染',
    transmission: '血液、皮损渗出液、黏膜直接接触',
    risk_level: 'MEDIUM_HIGH' as const,
    keywords: [
      '梅毒', '梅毒螺旋体', 'tppa阳性', 'rpr阳性', 'trust阳性',
      '一期梅毒', '二期梅毒', '神经梅毒', '硬下疳', '苍白螺旋体'
    ],
    precautions: [
      '体格检查、视触诊或伤口换药全程佩戴医用乳胶手套，医护人员手部有皮损者严禁直接操作',
      '接触患者病变皮肤或分泌物后，脱手套后必须严格使用皂液洗手并速干手消毒',
      '严防穿刺针刺伤'
    ],
    protection_gear: [
      '医用乳胶手套 (查体接触必备)',
      '医用外科口罩',
      '防水围裙/隔离衣 (脓液创面处理时)'
    ],
    emergency_steps: [
      '针刺或创面接触后立即挤血、流动清水肥皂水刷洗 5 分钟',
      '75% 酒精消毒局部',
      '立即行基线 TPPA/RPR 筛查，必要时肌注苄星青霉素 240 万单位进行预防性干预'
    ]
  },
  {
    name: '多重耐药超级细菌定植与感染 (CRE / MRSA / VRE / CRAB)',
    category: '接触传播与院感环境定植',
    persistence: '体内长期定植 (Persistent Colonization)',
    threat_to_doctor: '医护人员手部或工作服携带转移，易造成自身带菌及病区严重暴发流行',
    transmission: '接触传播、器械交叉接触、环境物表污染',
    risk_level: 'MEDIUM_HIGH' as const,
    keywords: [
      '耐碳青霉烯', 'cre', 'mrsa', '耐甲氧西林', 'vre', '耐万古霉素',
      '多重耐药菌', '泛耐药鲍曼', '多重耐药', 'carbapenem-resistant'
    ],
    precautions: [
      '实施严格的接触隔离措施，进入诊疗区域需穿戴一次性隔离衣与手套',
      '诊疗结束后严格执行卫生洗手与手部快速消毒',
      '听诊器、血压计袖带、体温表等器械必须专人专用，使用后 1000mg/L 含氯消毒剂彻底擦拭'
    ],
    protection_gear: [
      '一次性医用防护隔离衣',
      '医用检查手套',
      '医用外科口罩',
      '专用诊疗器具包'
    ],
    emergency_steps: [
      '接触患者排泄物或伤口渗液后立即肥皂水洗手加手消毒液彻底搓揉',
      '污染的工作服立即更换入黄色医疗废物袋高温高压消杀'
    ]
  },
  {
    name: '狂犬病 (Rabies / 狂犬病毒感染)',
    category: '唾液与中枢神经烈性感染',
    persistence: '急性烈性感染/致死率100%',
    threat_to_doctor: '患者躁动恐水，咬伤、抓伤或唾液喷溅接触医护破损黏膜极度致命',
    transmission: '唾液喷溅、咬伤抓伤、黏膜接触',
    risk_level: 'CRITICAL' as const,
    keywords: ['狂犬病', '狂犬病毒', '恐水症', 'rabies'],
    precautions: [
      '严格执行三级个人防护，穿戴防撕咬防护装备、防护面罩与厚橡胶手套',
      '就诊需设专人监护防范患者躁动攻击'
    ],
    protection_gear: [
      '全脸防溅防护面罩',
      '防刺防咬特种防护手套',
      '防护服与防水围裙'
    ],
    emergency_steps: [
      '若发生咬伤抓伤，立即使用 20% 肥皂水和流动清水交替冲洗伤口至少 15 分钟',
      '稀碘伏彻底消毒，伤口原则上不予缝合',
      '立即注射狂犬病被动免疫制剂并在当天启动狂犬疫苗全程接种'
    ]
  },
  {
    name: '克雅氏病 (Creutzfeldt-Jakob Disease / 朊病毒 CJD)',
    category: '神经组织医源性接触极高危',
    persistence: '终身致死性存在 (朊毒体常规高温高压无法灭活)',
    threat_to_doctor: '眼科、神经外科、腰穿等操作接触脑脊液或神经组织，极难消毒致医护严重感染',
    transmission: '神经组织接触、脑脊液穿刺、侵入性器械交叉污染',
    risk_level: 'CRITICAL' as const,
    keywords: ['克雅氏病', '克-雅氏病', 'cjd', '朊毒体', '朊病毒', '海绵状脑病'],
    precautions: [
      '凡涉及脑脊液穿刺、眼科或神经外科检查必须全程使用一次性专属器械',
      '使用后的器械严禁与普通手术器械混淆，必须密闭封装后送专业焚烧炉彻底销毁'
    ],
    protection_gear: [
      '双层耐穿刺乳胶手套',
      '防渗透全封闭防护面屏',
      '一次性防渗透连体防护服'
    ],
    emergency_steps: [
      '发生锐器针刺或脑脊液黏膜污染后，立即用 1M 氢氧化钠或 2.5% 次氯酸钠溶液冲洗创口 5 分钟',
      '大量清水反复彻底清洗，紧急上报疾控与院感专科评估'
    ]
  }
]

/**
 * 客户端即时文本扫描，返回匹配的传染病信息
 */
export function scanTextForInfections(text: string): InfectionRiskAlert {
  if (!text || !text.trim()) {
    return {
      has_risk: false,
      risk_level: 'SAFE',
      summary: '未检出高危传染病携带记录',
      diseases: [],
      precautions: [],
      protection_gear: [],
      emergency_steps: []
    }
  }

  const lower = text.toLowerCase()
  const matchedDiseases: InfectionDisease[] = []
  let highestLevel: 'CRITICAL' | 'HIGH' | 'MEDIUM_HIGH' | 'SAFE' = 'SAFE'

  const precautionsSet = new Set<string>()
  const gearSet = new Set<string>()
  const emergencySet = new Set<string>()

  for (const rule of INFECTION_RULES) {
    for (const kw of rule.keywords) {
      if (lower.includes(kw.toLowerCase())) {
        matchedDiseases.push({
          name: rule.name,
          category: rule.category,
          persistence: rule.persistence,
          threat_to_doctor: rule.threat_to_doctor,
          transmission: rule.transmission,
          risk_level: rule.risk_level,
          matched_keyword: kw
        })

        if (rule.risk_level === 'CRITICAL') {
          highestLevel = 'CRITICAL'
        } else if (rule.risk_level === 'HIGH' && highestLevel !== 'CRITICAL') {
          highestLevel = 'HIGH'
        } else if (highestLevel === 'SAFE') {
          highestLevel = 'MEDIUM_HIGH'
        }

        rule.precautions.forEach(p => precautionsSet.add(p))
        rule.protection_gear.forEach(g => gearSet.add(g))
        rule.emergency_steps.forEach(e => emergencySet.add(e))
        break
      }
    }
  }

  if (matchedDiseases.length === 0) {
    return {
      has_risk: false,
      risk_level: 'SAFE',
      summary: '未检出高危传染病携带记录',
      diseases: [],
      precautions: [],
      protection_gear: [],
      emergency_steps: []
    }
  }

  const names = matchedDiseases.map(d => d.name.split(' (')[0]).join('、')
  return {
    has_risk: true,
    risk_level: highestLevel,
    summary: `检测到患者有【${names}】必定/长期携带病史，接诊查体请务必落实职业暴露防护！`,
    diseases: matchedDiseases,
    precautions: Array.from(precautionsSet),
    protection_gear: Array.from(gearSet),
    emergency_steps: Array.from(emergencySet)
  }
}
