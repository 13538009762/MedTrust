# -*- coding: utf-8 -*-
import sys, os, hashlib, pymysql

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

DB_CONFIG = {
    'host': '127.0.0.1',
    'port': 3306,
    'user': 'root',
    'password': '123456',
    'database': 'medtrust',
    'charset': 'utf8mb4',
    'autocommit': True
}

ORIGINAL_RECORD_1 = {
    'onset_time': '2026-09-01 08:30',
    'duration': '持续3天，劳累后加重',
    'symptoms': '胸骨后压榨样闷痛、伴心悸气促，夜间偶有阵发性呼吸困难',
    'etiology': '连续夜班过度劳累诱发；既往高血压病史8年未规律服药；对青霉素、头孢类抗生素有明确严重过敏史',
    'treatment_plan': '1. 完善12导联心电图与心肌酶谱复查；2. 硝苯地平缓释片30mg qd降压；3. 严禁使用β-内酰胺类药物；4. 建议低盐低脂饮食并定期门诊随访',
    'vital_signs': '体温: 36.6℃ | 血压: 155/98 mmHg | 心率: 88 bpm | 血氧: 98%',
    'diagnosis': '【主诉】患者因“胸骨后闷痛3天加重半天”就诊。\n【现病史与诱因】于2026-09-01 08:30发病，劳累诱发，持续3天。\n【生命体征】体温36.6℃，血压155/98 mmHg，心率88 bpm。\n【初步诊断】高血压病II级（中危）；心绞痛待查；严重过敏体质。\n【处置建议】降压治疗、心电监护，禁用青霉素头孢。'
}

TAMPERED_RECORD_1 = {
    'etiology': '【已被黑客恶意篡改】患者体质极佳，无任何药物过敏史，身体健康',
    'treatment_plan': '【已被黑客恶意篡改】常规静推注射用青霉素钠 800万单位 q8h，无需控压',
    'diagnosis': '【已被黑客恶意篡改】患者完全健康，血压心电图正常，既往药物过敏史已清空，准予青霉素输注。'
}

def get_db():
    return pymysql.connect(**DB_CONFIG)

def compute_hash(r, file_hash=''):
    parts = [r.get('record_no',''), r.get('data_type',''), r.get('onset_time',''), r.get('duration',''), r.get('symptoms',''), r.get('etiology',''), r.get('treatment_plan',''), r.get('diagnosis',''), file_hash]
    raw = '|'.join(str(p or '') for p in parts)
    return hashlib.sha256(raw.encode('utf-8')).hexdigest()

def do_status():
    conn = get_db()
    cur = conn.cursor(pymysql.cursors.DictCursor)
    cur.execute('''
        SELECT r.*, f.file_hash, u.real_name as patient_name 
        FROM medical_records r 
        LEFT JOIN medical_files f ON f.record_id = r.id 
        LEFT JOIN users u ON u.id = r.patient_id
        ORDER BY r.id ASC
    ''')
    rows = cur.fetchall()
    print('=' * 80)
    print('         MedTrust MySQL 数据库当前病历存盘状态与哈希检测')
    print('=' * 80)
    for r in rows:
        calc_h = compute_hash(r, r.get('file_hash') or '')
        is_attacked = '【已被黑客' in (r.get('etiology') or '') or '【已被黑客' in (r.get('treatment_plan') or '')
        status_tag = '🚨 [已遭恶意篡改]' if is_attacked else '🛡️ [正常状态]'
        print(f"\n病历 ID: {r['id']} | 编号: {r['record_no']} | 患者: {r.get('patient_name')} | 状态: {status_tag}")
        print(f"  - 诱因/过敏史: {r.get('etiology')}")
        print(f"  - 治疗方案:     {r.get('treatment_plan')}")
        print(f"  - 当前计算哈希: {calc_h}")
    conn.close()

def do_attack():
    conn = get_db()
    cur = conn.cursor()
    cur.execute('''
        UPDATE medical_records 
        SET etiology = %s, treatment_plan = %s, diagnosis = %s 
        WHERE id = 1
    ''', (TAMPERED_RECORD_1['etiology'], TAMPERED_RECORD_1['treatment_plan'], TAMPERED_RECORD_1['diagnosis']))
    conn.close()
    print('\n' + '!' * 70)
    print('🚨 [ATTACK SUCCESS] 攻击完成！已直接绕过 API 修改 MySQL 数据库！')
    print('   - 目标记录: ID=1 (REC202609001 - 张三)')
    print('   - 篡改过敏史: "对青霉素严重过敏史" ---> "无任何药物过敏史"')
    print('   - 篡改治疗方案: "严禁使用β-内酰胺" ---> "静推大剂量青霉素"')
    print('   - 预期结果: 现在在 MedTrust 前端打开病历，系统将立即亮起高危篡改告警！')
    print('!' * 70 + '\n')

def do_rollback():
    conn = get_db()
    cur = conn.cursor()
    cur.execute('''
        UPDATE medical_records 
        SET onset_time = %s, duration = %s, symptoms = %s, etiology = %s, treatment_plan = %s, vital_signs = %s, diagnosis = %s 
        WHERE id = 1
    ''', (
        ORIGINAL_RECORD_1['onset_time'],
        ORIGINAL_RECORD_1['duration'],
        ORIGINAL_RECORD_1['symptoms'],
        ORIGINAL_RECORD_1['etiology'],
        ORIGINAL_RECORD_1['treatment_plan'],
        ORIGINAL_RECORD_1['vital_signs'],
        ORIGINAL_RECORD_1['diagnosis']
    ))
    conn.close()
    print('\n' + '=' * 70)
    print('✅ [RESTORE SUCCESS] 恢复完成！已将数据库数据重置为真实原始病历！')
    print('   - 目标记录: ID=1 (REC202609001 - 张三)')
    print('   - 严重过敏史已恢复，用药方案已恢复')
    print('   - 预期结果: 此时在 MedTrust 前端再次查看该病历，区块链核验将重新恢复为绿色通过！')
    print('=' * 70 + '\n')

def main():
    if len(sys.argv) < 2:
        print('用法:')
        print('  python tamper_cli.py status   # 查看数据库病历当前状态与哈希')
        print('  python tamper_cli.py attack   # 攻击数据库：篡改病历1抹除青霉素过敏史')
        print('  python tamper_cli.py rollback # 恢复数据库：还原病历1为原始真实状态')
        return

    cmd = sys.argv[1].lower()
    if cmd == 'status':
        do_status()
    elif cmd == 'attack':
        do_attack()
    elif cmd in ('rollback', 'restore'):
        do_rollback()
    else:
        print(f'未知命令: {cmd}')

if __name__ == '__main__':
    main()
