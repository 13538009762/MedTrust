# -*- coding: utf-8 -*-
"""
MedTrust 医疗区块链系统全面多维种子数据生成脚本
扩充包含：
- 3 家三甲医疗机构 (第一人民医院, 省立中心医院, 协和医学中心)
- 15 个临床与检验科室
- 11 位覆盖各科室的主任/主治执业医生 (doc_a, doc_b, doc_c, doc_a2, doc_a3, doc_b2...)
- 8 位不同年龄段、病种各异的真实患者画像 (带身份证号与手机号)
- 12 份国家卫健委 SOAP 规范格式的结构化临床病历大表格 (心血管/急救/骨科/神经/呼吸/内分泌)
- 3 条跨院授权关系 (满足跨院免申请直接放行与未授权破窗的对比展示)
"""

import pymysql

conn = pymysql.connect(
    host='127.0.0.1',
    user='root',
    password='123456',
    database='medtrust',
    port=3306,
    autocommit=True,
    charset='utf8mb4'
)
cur = conn.cursor()

# 1. 预置医疗机构
hospitals = [
    (1, 'HOSP_A', '第一人民医院', '三甲', '市中心解放路88号', 1),
    (2, 'HOSP_B', '省立中心医院', '三甲', '高新区科技大道120号', 1),
    (3, 'HOSP_C', '协和医学中心', '三甲', '大学城健康东路66号', 1),
]
for h in hospitals:
    cur.execute("""
        INSERT INTO hospitals (id, hospital_no, name, level, address, status)
        VALUES (%s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE name=VALUES(name)
    """, h)

# 2. 预置科室 (覆盖内科、外科、急危重症、儿科、骨科)
departments = [
    (1, 'DEPT_A01', 1, '急诊科', '急诊急救与综合创伤救治'),
    (2, 'DEPT_A02', 1, '心血管内科', '冠心病、心绞痛、高血压、心律失常诊疗'),
    (3, 'DEPT_A03', 1, '医学检验科', '临床生化、免疫检验、微生物与心肌标志物'),
    (4, 'DEPT_A04', 1, '骨科与创伤外科', '关节置换、脊柱微创与运动创伤损伤修复'),
    (5, 'DEPT_A05', 1, '呼吸与危重症医学科', '支气管哮喘、慢性气道阻塞疾病与肺部感染'),
    
    (6, 'DEPT_B01', 2, '急救中心', '急危重症抢救、心源性休克与生命支持'),
    (7, 'DEPT_B02', 2, '普外科与肝胆外科', '微创胆道镜、腹部闭合伤急救与外科感染'),
    (8, 'DEPT_B03', 2, '神经外科与脑血管中心', '急重型颅脑创伤、脑出血微创引流与血管介入'),
    (9, 'DEPT_B04', 2, '儿科与新生儿重症科', '小儿呼吸系统疾患、急性感染与高热抽搐急救'),
    (10, 'DEPT_B05', 2, '肿瘤综合诊疗中心', '实体肿瘤靶向治疗、免疫治疗与化疗随访'),

    (11, 'DEPT_C01', 3, '神经内科与癫痫中心', '脑血管供血不足、偏头痛、神经退行性病变'),
    (12, 'DEPT_C02', 3, '临床免疫与风湿内科', '系统性红斑狼疮、类风湿关节炎、过敏性哮喘排查'),
    (13, 'DEPT_C03', 3, '内分泌与代谢疾病科', '2型糖尿病、糖尿病周围神经与视网膜并发症管理'),
    (14, 'DEPT_C04', 3, '心胸外科介入中心', '冠脉造影与支架植入、瓣膜微创介入、主动脉夹层排查'),
    (15, 'DEPT_C05', 3, '感染与热带病医学科', '不明原因发热、抗生素合理规范使用与多重耐药菌'),
]
for d in departments:
    cur.execute("""
        INSERT INTO departments (id, dept_no, hospital_id, name, description)
        VALUES (%s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description)
    """, d)

# 3. 预置用户 (含 11 名各院名医、8 名患者画像、管理员与监管专员)
# bcrypt hash for '123456': $2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2
users = [
    # 医生矩阵 (HOSP_A 第一人民医院)
    (1, 'DOC_A001', 'doc_a', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '李建国医生', 'doctor', 1, 2, '主任医师', '13800000001', '11010119750315112X', 'NORMAL'),
    (8, 'DOC_A002', 'doc_a2', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '张伟林医生', 'doctor', 1, 4, '副主任医师', '13800000011', '110101197806123318', 'NORMAL'),
    (9, 'DOC_A003', 'doc_a3', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '刘晓敏医生', 'doctor', 1, 5, '主治医师', '13800000012', '110101198603244421', 'NORMAL'),
    
    # 医生矩阵 (HOSP_B 省立中心医院)
    (2, 'DOC_B001', 'doc_b', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '王明德医生', 'doctor', 2, 6, '主治医师', '13800000002', '110101198207183341', 'NORMAL'),
    (10, 'DOC_B002', 'doc_b2', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '赵国良医生', 'doctor', 2, 7, '主任医师', '13800000021', '110101197411082215', 'NORMAL'),
    (11, 'DOC_B003', 'doc_b3', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '孙雅琴医生', 'doctor', 2, 9, '副主任医师', '13800000022', '110101198309196627', 'NORMAL'),
    (12, 'DOC_B004', 'doc_b4', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '马建军医生', 'doctor', 2, 8, '主任医师', '13800000023', '110101197108147719', 'NORMAL'),

    # 医生矩阵 (HOSP_C 协和医学中心)
    (3, 'DOC_C001', 'doc_c', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '陈晓华医生', 'doctor', 3, 11, '副主任医师', '13800000003', '110101198511252267', 'NORMAL'),
    (13, 'DOC_C002', 'doc_c2', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '周德成医生', 'doctor', 3, 14, '主任医师', '13800000031', '110101197212015519', 'NORMAL'),
    (14, 'DOC_C003', 'doc_c3', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '吴秀兰医生', 'doctor', 3, 13, '副主任医师', '13800000032', '110101198105157723', 'NORMAL'),
    (15, 'DOC_C004', 'doc_c4', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '郑天明医生', 'doctor', 3, 12, '主治医师', '13800000033', '110101198908288812', 'NORMAL'),

    # 8 名患者画像 (涵盖不同年龄、性别、病种)
    (4, 'PAT_0001', 'pat_zhang', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '张三', 'patient', 0, 0, '患者', '13900000001', '110101198805122315', 'NORMAL'),
    (5, 'PAT_0002', 'pat_li', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '李四', 'patient', 0, 0, '患者', '13900000002', '310101199208204562', 'NORMAL'),
    (20, 'PAT_0003', 'pat_wang', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '王秀英', 'patient', 0, 0, '患者', '13900000003', '320102196404153328', 'NORMAL'),
    (21, 'PAT_0004', 'pat_zhao', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '赵建平', 'patient', 0, 0, '患者', '13900000004', '440103198110051119', 'NORMAL'),
    (22, 'PAT_0005', 'pat_qian', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '钱小芳', 'patient', 0, 0, '患者', '13900000005', '330104199703222241', 'NORMAL'),
    (23, 'PAT_0006', 'pat_sun', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '孙明浩', 'patient', 0, 0, '患者', '13900000006', '210102200709184432', 'NORMAL'),
    (24, 'PAT_0007', 'pat_zhou', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '周桂珍', 'patient', 0, 0, '患者', '13900000007', '120101195507115562', 'NORMAL'),
    (25, 'PAT_0008', 'pat_chen', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '陈晨', 'patient', 0, 0, '患者(儿童)', '13900000008', '110105201802146617', 'NORMAL'),

    # 管理与监管账户
    (6, 'ADM_0001', 'admin', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '系统管理员', 'admin', 1, 0, '技术主管', '13700000001', '110101199001010011', 'NORMAL'),
    (7, 'SUP_0001', 'supervisor', '$2a$10$wTfq7L2N4e0p/92hC9k/5e3bA/bYv18L/v4mUvPsm71uLw74m9bF2', '卫健监管专员', 'supervisor', 0, 0, '医疗合规督察', '13600000001', '110101198304100088', 'NORMAL'),
]
for u in users:
    cur.execute("""
        INSERT INTO users (id, user_no, username, password_hash, real_name, role, hospital_id, department_id, title, phone, id_card, status)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE 
          user_no=VALUES(user_no),
          username=VALUES(username),
          real_name=VALUES(real_name),
          role=VALUES(role),
          hospital_id=VALUES(hospital_id),
          department_id=VALUES(department_id),
          title=VALUES(title),
          phone=VALUES(phone),
          id_card=VALUES(id_card),
          password_hash=VALUES(password_hash),
          status=VALUES(status)
    """, u)

# 4. 预置 12 份多学科临床电子病历大表格 (含详细 SOAP 架构、过敏史与生命体征)
records_data = [
    # 1. 张三 (第一人民医院 - 心血管内科)
    (1, 'REC202609001', 4, 1, 1, 'EMR',
     '2026-09-01 08:30', '持续3天，劳累后加重',
     '胸骨后压榨样闷痛、伴心悸气促，夜间偶有阵发性呼吸困难',
     '连续夜班过度劳累诱发；既往高血压病史8年未规律服药；对青霉素、头孢类抗生素有明确严重过敏史',
     '1. 完善12导联心电图与心肌酶谱复查；2. 硝苯地平缓释片30mg qd降压；3. 严禁使用β-内酰胺类药物；4. 建议低盐低脂饮食并定期门诊随访',
     '体温: 36.6℃ | 血压: 155/98 mmHg | 心率: 88 bpm | 血氧: 98%',
     '【主诉】患者因“胸骨后闷痛3天加重半天”就诊。\n【现病史与诱因】于2026-09-01 08:30发病，劳累诱发，持续3天。\n【生命体征】体温36.6℃，血压155/98 mmHg，心率88 bpm。\n【初步诊断】高血压病II级（中危）；心绞痛待查；严重过敏体质。\n【处置建议】降压治疗、心电监护，禁用青霉素头孢。',
     '0x9b0548acb1b6506b8c27ada12edb6167b53ad5b361f1853d7101b7a83298cb1a', 159, '2026-09-01 10:30:00'),

    # 2. 张三 (第一人民医院 - 医学检验科生化单)
    (2, 'REC202609002', 4, 1, 1, 'REPORT',
     '2026-09-05 13:00', '发作性胸痛2小时',
     '心前区刺痛，活动后明显，休息后稍缓解',
     '情绪激动后诱发，高脂饮食史',
     '1. 急查血常规、生化五项、心肌肌钙蛋白cTnI；2. 口服阿司匹林100mg，单硝酸异山梨酯；3. 观察2小时复测心电图',
     '体温: 36.7℃ | 血压: 142/90 mmHg | 心率: 92 bpm | 血氧: 99%',
     '【检查报告小结】血常规生化五项与心肌酶谱检验单：心肌肌钙蛋白与CK-MB处于边缘正常上限，建议结合临床症状动态复查以排除ACS。',
     '0x5429690a74f8bedbaa647aae49784a8b0795b900044b222f75aeb42aa6f39400', 160, '2026-09-05 14:15:00'),

    # 3. 张三 (省立中心医院 - 急救中心抢救病历)
    (3, 'REC20260910351c4d04', 4, 2, 2, 'EMR',
     '2026-08-15 22:40', '突发撕裂样胸痛伴大汗1小时',
     '胸骨后及后背撕裂样剧烈疼痛，大汗淋漓，濒死感明显，肢端湿冷',
     '重度高血压长期未规范用药，重体力搬运后诱发；对青霉素类药物严重过敏史',
     '1. 绝对卧床制动、心电监护与高流量吸氧；2. 硝普钠微泵严密控压；3. 急诊胸痛介入抢救中心会诊，排除急性主动脉夹层',
     '体温: 36.4℃ | 血压: 85/50 mmHg | 心率: 115 bpm | 血氧: 91%',
     '【省立中心医院·急诊抢救病历】急性心肌梗死待排；主动脉夹层待排；心源性休克前兆。需跨院紧急调阅既往心血管造影及严重药物过敏史！',
     '0x3a9b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b', 128, '2026-08-15 23:10:00'),

    # 4. 张三 (协和医学中心 - 神经内科)
    (4, 'REC20260720C01', 4, 3, 3, 'REPORT',
     '2026-07-20 09:15', '间歇性头晕视物旋转1周',
     '头部位置变动时眩晕加重，伴耳鸣及恶心感，无面瘫与肢体运动障碍',
     '颈椎退行性病变压迫椎基底动脉，近期连续熬夜失眠诱发',
     '1. 颈椎平扫MRI与经颅多普勒(TCD)检查；2. 口服甲磺酸倍他司汀片抗眩晕；3. 避免长时间低头伏案',
     '体温: 36.5℃ | 血压: 130/85 mmHg | 心率: 74 bpm | 血氧: 99%',
     '【协和医学中心·神经内科报告】颈椎退行性改变伴椎-基底动脉供血不足；TCD提示右侧椎动脉血流流速减缓。',
     '0xa45a3bd35dfd8202e27d68091ae9b0976bce3afdc84b2cbdd4d625fe2ee03326', 161, '2026-07-20 10:30:00'),

    # 5. 李四 (第一人民医院 - 心内科门诊)
    (5, 'REC20260902A01', 5, 1, 1, 'EMR',
     '2026-09-02 14:20', '劳累后胸闷胸痛1个月，加重3天',
     '快步行走或爬楼至3层出现心前区憋闷压榨感，持续约3-5分钟，含服硝酸甘油可缓解',
     '吸烟史15年，高脂血症病史5年；无明确药物过敏史',
     '1. 口服阿司匹林肠溶片 100mg qd 抗血小板；2. 阿托伐他汀钙片 20mg qn 降脂稳斑；3. 择期行冠脉CTA造影明确狭窄程度',
     '体温: 36.8℃ | 血压: 138/88 mmHg | 心率: 76 bpm | 血氧: 98%',
     '【第一医院心内科】冠状动脉粥样硬化性心脏病；稳定型心绞痛；高脂血症。建议避免重体力负荷并门诊复查。',
     '0x6d8f2a1b3c5e7f9a0b2d4e6f8a1c3e5b7d9f1a3c5e7b9d1f3a5c7e9b1d3f5a7c', 162, '2026-09-02 15:00:00'),

    # 6. 王秀英 (协和医学中心 - 内分泌代谢科)
    (6, 'REC20260904C01', 20, 14, 3, 'EMR',
     '2026-09-04 10:00', '口干多饮多尿10年，双下肢麻木刺痛3个月',
     '双足远端对称性袜套样感觉减退，伴夜间烧灼样疼痛，踩棉花感；视力逐渐模糊',
     '2型糖尿病病史10年，血糖控制欠佳(HbA1c 9.2%)；对磺胺类抗生素及磺脲类降糖药有严重皮疹过敏史',
     '1. 门冬胰岛素30早晚皮下注射精细控糖；2. 硫辛酸胶囊抗氧化改善周围神经病变；3. 甲钴胺片营养神经；4. 严禁使用磺胺类及磺脲类药物',
     '体温: 36.5℃ | 血压: 146/82 mmHg | 心率: 80 bpm | 血氧: 98%',
     '【协和内分泌科诊断】2型糖尿病性多发性周围神经病变；糖尿病性视网膜病变(非增殖期)；明确磺胺过敏史。',
     '0x7e1a3b5c7d9f2a4b6c8e0d2f4a6c8e1b3d5f7a9c1e3b5d7f9a1c3e5b7d9f2a4b', 163, '2026-09-04 11:20:00'),

    # 7. 王秀英 (第一人民医院 - 心内科慢病联合随访)
    (7, 'REC20260828A01', 20, 1, 1, 'REPORT',
     '2026-08-28 09:30', '高血压病慢性随访半年，偶有头昏',
     '晨起轻微头胀，休息后缓解，无肢体无力及视物成双',
     '老年性动脉硬化伴糖尿病血管病变，日常低盐饮食执行不佳',
     '1. 缬沙坦胶囊 80mg qd；2. 监测晨起及睡前双重血压；3. 保持与协和内分泌科降糖方案协同治疗',
     '体温: 36.6℃ | 血压: 150/92 mmHg | 心率: 72 bpm | 血氧: 99%',
     '【第一医院门诊随诊】原发性高血压II级很高危组；糖尿病合并动脉粥样硬化。已与协和医院建立跨院互通档案。',
     '0x8f2b4c6d8e0a2c4e6f8b1d3f5a7c9e1b3d5f7a9c2e4b6d8f0a2c4e6f8b1d3f5a', 164, '2026-08-28 10:15:00'),

    # 8. 赵建平 (第一人民医院 - 骨科与创伤外科)
    (8, 'REC20260906A02', 21, 8, 1, 'EMR',
     '2026-09-06 15:30', '反复腰痛伴左下肢放射痛半年，加重1周',
     '腰背部酸痛，久坐久站加剧，放射至左小腿后外侧及足背，左足大拇趾背伸肌力IV级',
     '从事重型机械长途驾驶，长年久坐震动劳损；无既往药物食物过敏史',
     '1. 绝对硬板床休息并佩戴定制腰围；2. 塞来昔布胶囊 0.2g bid 消炎止痛；3. 完善腰椎三维CT重建，评估椎间孔镜微创手术指征',
     '体温: 36.7℃ | 血压: 125/80 mmHg | 心率: 68 bpm | 血氧: 99%',
     '【第一医院骨科门诊】腰椎间盘突出症(L4-L5中央偏左型，压迫左侧L5神经根)；腰椎管退行性狭窄。',
     '0x9a3c5e7b9d1f3a5c7e9b1d3f5a7c9e1b3d5f7a9c2e4b6d8f0a2c4e6f8b1d3f5a', 165, '2026-09-06 16:40:00'),

    # 9. 钱小芳 (省立中心医院 - 急救重症医学中心)
    (9, 'REC20260830B02', 22, 2, 2, 'EMR',
     '2026-08-30 02:15', '突发呼吸困难伴严重喘息1小时急救',
     '端坐呼吸，大汗淋漓，双肺听诊满布广泛高调哮鸣音，呼吸音极弱(寂静肺征象)',
     '接触猫毛后急性激发，既往严重支气管哮喘病史12年；对阿司匹林及非甾体抗炎药(NSAIDs)有阿司匹林哮喘严重反应史',
     '1. 高流量面罩吸氧；2. 甲泼尼龙琥珀酸钠 80mg 静脉注射抗炎；3. 硫酸特布他林联合异丙托溴铵雾化吸入；4. 绝对禁用阿司匹林等非甾体类镇痛解热药',
     '体温: 37.1℃ | 血压: 160/105 mmHg | 心率: 128 bpm | 血氧: 87%',
     '【省立中心医院急诊抢救】支气管哮喘急性重度发作(危重度)；急性呼吸衰竭代偿期；阿司匹林哮喘三联征。需跨院重点核实既往过敏史！',
     '0xa1b3c5e7f9a2b4c6d8e0f2a4b6c8e0f2a4b6c8e0f2a4b6c8e0f2a4b6c8e0f2a4', 166, '2026-08-30 03:30:00'),

    # 10. 钱小芳 (协和医学中心 - 临床免疫与风湿内科)
    (10, 'REC20260908C02', 22, 15, 3, 'REPORT',
     '2026-09-08 11:00', '反复过敏性鼻炎伴咳嗽哮喘半年慢病评估',
     '阵发性喷嚏、清涕，遇冷空气刺激后刺激性干咳，夜间偶有喉间痰鸣',
     '特应性过敏体质，血清总IgE高达 1280 IU/mL；吸入物过敏原筛查粉尘螨(++++)',
     '1. 布地奈德福莫特罗粉吸入剂 160/4.5ug 早晚各一吸；2. 孟鲁司特钠片 10mg qn 抗白三烯；3. 严格环境避螨防尘',
     '体温: 36.6℃ | 血压: 118/75 mmHg | 心率: 74 bpm | 血氧: 99%',
     '【协和免疫科报告】特应性支气管哮喘(慢性持续期)；中重度变应性鼻炎。肺功能检查提示中度阻塞性通气功能障碍，支气管舒张试验阳性。',
     '0xb2c4d6e8f0a3b5c7d9e1f3a5b7c9e1f3a5b7c9e1f3a5b7c9e1f3a5b7c9e1f3a5', 167, '2026-09-08 11:45:00'),

    # 11. 孙明浩 (第一人民医院 - 骨科与创伤外科)
    (11, 'REC20260901A03', 23, 8, 1, 'IMAGE',
     '2026-09-01 16:00', '打篮球扭伤右膝关节伴肿胀活动受限半天',
     '右膝关节弥漫性肿胀，浮髌试验(+)，抽屉试验(+)，不能负重行走',
     '剧烈对抗运动扭转受伤，既往身体强健，无任何药物过敏史',
     '1. 急诊右膝冷敷、加压包扎及支具固定；2. 行右膝关节高场强MRI平扫排查韧带撕裂；3. 择期行关节镜下前交叉韧带自体肌腱重建术',
     '体温: 36.8℃ | 血压: 122/78 mmHg | 心率: 82 bpm | 血氧: 100%',
     '【第一医院骨科影像】右膝前交叉韧带断裂(全层撕裂伴周围水肿)；内侧半月板后角II-III度撕裂；关节腔中度积血积液。',
     '0xc3d5e7f9a1b4c6d8e0f2a4b6c8e0f2a4b6c8e0f2a4b6c8e0f2a4b6c8e0f2a4b6', 168, '2026-09-01 17:10:00'),

    # 12. 周桂珍 (第一人民医院 - 呼吸与危重症医学科)
    (12, 'REC20260905B03', 24, 9, 1, 'EMR',
     '2026-09-05 08:30', '慢性咳喘20年，受凉后加重伴气急嗜睡2天',
     '咳大量黄色黏稠脓痰，不易咳出，呼吸急促费力，口唇发绀，查体剑突下心脏搏动增强',
     '长期农村土灶柴烟接触史，既往慢性支气管炎肺气肿20年；对头孢菌素类药物有过敏性药疹史',
     '1. 持续低流量吸氧(1.5 L/min)配合无创双水平正压通气(BiPAP)；2. 莫西沙星氯化钠注射液抗感染；3. 氨茶碱注射液平喘祛痰',
     '体温: 37.8℃ | 血压: 135/85 mmHg | 心率: 105 bpm | 血氧: 84%',
     '【第一医院呼吸危重症】慢性阻塞性肺疾病急性加重期(AECOPD)；II型呼吸衰竭；慢性肺源性心脏病(失代偿期)；头孢药物过敏。',
     '0xd4e6f8a0b2c5d7e9f1a3b5c7d9e1f3a5b7c9e1f3a5b7c9e1f3a5b7c9e1f3a5b7', 169, '2026-09-05 09:40:00'),
]

for r in records_data:
    cur.execute("""
        INSERT INTO medical_records (id, record_no, patient_id, doctor_id, hospital_id, data_type, onset_time, duration, symptoms, etiology, treatment_plan, vital_signs, diagnosis, fabric_tx_id, block_height, created_at)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE 
          patient_id=VALUES(patient_id),
          doctor_id=VALUES(doctor_id),
          hospital_id=VALUES(hospital_id),
          data_type=VALUES(data_type),
          onset_time=VALUES(onset_time),
          duration=VALUES(duration),
          symptoms=VALUES(symptoms),
          etiology=VALUES(etiology),
          treatment_plan=VALUES(treatment_plan),
          vital_signs=VALUES(vital_signs),
          diagnosis=VALUES(diagnosis)
    """, r)

# 5. 预置 12 份病历的对应影像切片与加密文件 (IPFS CID 与 SHA-256)
files_data = [
    (1, 1, '门诊病历档案_张三_REC202609001.pdf', 'pdf', 1048576, 'QmZtmD2qt8STTqp331B2W3B6T4WGYX6s8B8w3q5vKj81A1', 'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3', '2026-09-01 10:30:00'),
    (2, 2, '生化检验报告切片_REC202609002.png', 'png', 524288, 'QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG', 'b5d4045c3f466fa91fe2cc6abe79232a1a57cdf104f7a26e716e0a1e2789df78', '2026-09-05 14:15:00'),
    (3, 3, '省立急诊抢救造影切片_REC20260815B01.pdf', 'pdf', 2097152, 'Qm20c696527f27c2418e384838f5de55664c6724a51a08b9677a3d7cd40664df36', '5331b1107204c1b3ea55b1d99dc2d3ec8b339d6b8d0f9c32995b9457edd761b3', '2026-08-15 23:10:00'),
    (4, 4, '协和经颅多普勒TCD报告单_REC20260720C01.pdf', 'pdf', 1572864, 'QmW2WQi7j6c7UgJTarActp7tCMikN4Wu5qmtQEhMrqqLNW', 'd9a8b7c6e5f4d3c2b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8', '2026-07-20 10:30:00'),
    (5, 5, '冠脉CTA三维造影重建切片_REC20260902A01.png', 'png', 3145728, 'QmTaXv782nKwKmsi628sh92hs7h2jsh71hsnxms918sj2h', 'e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2', '2026-09-02 15:00:00'),
    (6, 6, '神经电生理与肌电图报告_REC20260904C01.pdf', 'pdf', 1835008, 'QmRkWs71ks82ns7hsk92hs71bsh28s7shxmslwoe827shd', 'f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3', '2026-09-04 11:20:00'),
    (7, 7, '24小时动态血压监测报告单_REC20260828A01.pdf', 'pdf', 917504, 'QmVmPq81ks82js7shk92hs72bsh28s7shxmslwoe827sha', 'a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4', '2026-08-28 10:15:00'),
    (8, 8, '腰椎矢状位与轴位高场MRI切片_REC20260906A02.png', 'png', 4194304, 'QmZnLs91js82ks7shk92hs73bsh28s7shxmslwoe827shb', 'b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5', '2026-09-06 16:40:00'),
    (9, 9, '血气分析与急救心电图监测单_REC20260830B02.pdf', 'pdf', 1258291, 'QmXoPs01js82ks7shk92hs74bsh28s7shxmslwoe827shc', 'c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6', '2026-08-30 03:30:00'),
    (10, 10, '肺功能弥散与支气管舒张报告_REC20260908C02.pdf', 'pdf', 1677721, 'QmYpQs11js82ks7shk92hs75bsh28s7shxmslwoe827shd', 'd6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7', '2026-09-08 11:45:00'),
    (11, 11, '右膝前交叉韧带损伤高清MRI序列_REC20260901A03.dcm', 'dicom', 5242880, 'QmZqRs21js82ks7shk92hs76bsh28s7shxmslwoe827she', 'e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8', '2026-09-01 17:10:00'),
    (12, 12, '床旁胸部高分辨率CT扫查报告_REC20260905B03.pdf', 'pdf', 2621440, 'QmArSs31js82ks7shk92hs77bsh28s7shxmslwoe827shf', 'f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9', '2026-09-05 09:40:00'),
]
for f in files_data:
    cur.execute("""
        INSERT INTO medical_files (id, record_id, file_name, file_type, file_size, ipfs_cid, file_hash, created_at)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE file_name=VALUES(file_name), ipfs_cid=VALUES(ipfs_cid), file_hash=VALUES(file_hash)
    """, f)

# 6. 预置跨院授权关系
# 授权 1: 张三 (4) 授权 第一医院李建国医生 (1) 跨院单项调阅 协和医院病历 4 (REC20260720C01)
# 授权 2: 王秀英 (20) 授权 省立中心医院赵国良医生 (10) 调阅 第一医院心内科随访病历 7 (REC20260828A01)
# 授权 3: 李四 (5) 授权 协和医学中心心胸外科周德成医生 (13) 调阅 第一医院心血管病历 5 (REC20260902A01)
auths_data = [
    (1, 'AUTH202609001', 4, 'DOCTOR', 1, 'SINGLE', 4, '2026-09-01 00:00:00', '2027-09-01 23:59:59', 'ACTIVE', '0x5c1a7089b5b0c2d9801923a45678cdef90123456789abcdef0123456789abcdef', '2026-09-01 09:00:00'),
    (2, 'AUTH202609002', 20, 'DOCTOR', 10, 'SINGLE', 7, '2026-09-01 00:00:00', '2027-09-01 23:59:59', 'ACTIVE', '0x6d2b819ac6c0d3e0912a34b56789defa0123456789abcdef0123456789abcdef', '2026-09-01 09:00:00'),
    (3, 'AUTH202609003', 5, 'DOCTOR', 13, 'SINGLE', 5, '2026-09-01 00:00:00', '2027-09-01 23:59:59', 'ACTIVE', '0x7e3c92ab07d1e4f1a23b45c67890efab123456789abcdef0123456789abcdef', '2026-09-01 09:00:00'),
]
for a in auths_data:
    cur.execute("""
        INSERT INTO authorizations (id, auth_no, patient_id, auth_target_type, auth_target_id, scope_type, record_id, start_time, end_time, status, fabric_tx_id, created_at)
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        ON DUPLICATE KEY UPDATE 
          auth_target_type=VALUES(auth_target_type),
          auth_target_id=VALUES(auth_target_id),
          scope_type=VALUES(scope_type),
          record_id=VALUES(record_id),
          status=VALUES(status)
    """, a)

print('========================================================================')
print('[MedTrust Database Seed Completed Successfully]')
print(' - Hospitals: 3')
print(' - Departments: 15')
print(' - Doctors: 11')
print(' - Patients: 8')
print(' - Structured Medical Records: 12')
print(' - Cross-Hospital Authorizations: 3')
print('========================================================================')
conn.close()
