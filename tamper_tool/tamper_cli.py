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
    'onset_time': '2025-04-10',
    'duration': '2周',
    'symptoms': '反复心前区闷痛伴胸闷',
    'chief_complaint': '活动后胸痛胸闷6月，加重1周',
    'etiology': '劳累及情绪激动诱发',
    'treatment_plan': '单硝酸异山梨酯缓释片 30mg qd, 阿托伐他汀 20mg qn',
    'diagnosis': '冠状动脉粥样硬化性心脏病 劳力性心绞痛'
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
        WHERE record_no = 'REC20250501001' OR id = 1
    ''', (TAMPERED_RECORD_1['etiology'], TAMPERED_RECORD_1['treatment_plan'], TAMPERED_RECORD_1['diagnosis']))
    conn.close()
    print('\n' + '!' * 70)
    print('🚨 [ATTACK SUCCESS] 攻击完成！已直接绕过 API 修改 MySQL 数据库！')
    print('   - 目标记录: ID=1 (REC20250501001 - 张三)')
    print('   - 篡改诱因与诊断: 冠心病心绞痛 ---> 无任何过敏史准予青霉素输注')
    print('   - 预期结果: 现在在 MedTrust 前端打开病历，系统将立即亮起高危篡改告警！')
    print('!' * 70 + '\n')

def do_rollback():
    conn = get_db()
    cur = conn.cursor()
    cur.execute('''
        UPDATE medical_records 
        SET onset_time = %s, duration = %s, symptoms = %s, chief_complaint = %s, etiology = %s, treatment_plan = %s, diagnosis = %s 
        WHERE record_no = 'REC20250501001' OR id = 1
    ''', (
        ORIGINAL_RECORD_1['onset_time'],
        ORIGINAL_RECORD_1['duration'],
        ORIGINAL_RECORD_1['symptoms'],
        ORIGINAL_RECORD_1['chief_complaint'],
        ORIGINAL_RECORD_1['etiology'],
        ORIGINAL_RECORD_1['treatment_plan'],
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
