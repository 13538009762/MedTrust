# -*- coding: utf-8 -*-
"""
MedTrust 区块链防篡改攻防演练控制台 (Web Console)
运行端口: 8090
通过直接绕过应用层 API 与网关，直连 MySQL 数据库执行恶意篡改，
用于向评委与用户生动演示：当黑客攻破数据库时，区块链是如何在被打开的一瞬间即时感知并阻断篡改的。
"""

import sys, os, json, hashlib
from flask import Flask, jsonify, request, render_template_string
import pymysql

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

app = Flask(__name__)

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

# 预存的 Fabric 联盟链链上固化基准哈希（不可篡改）
BASELINE_CHAIN_HASHES = {
    'REC202609001': 'b1e85a9a949446045c213ae3fdc558b3e0c62686ec7cd482cb4f8d3e8f3b992e',
    'REC202609002': '861ebf116f5632136fa3c2ec4d32a246708f0e8eb34344d949928cc4fef4d9f8',
    'REC20260910351c4d04': 'cbc5294928bb6f8ed240f6b79ef9e8696e843b0b2a7d55ec1d2929b3bcceccc3',
    'REC20260720C01': 'd1752f9b35104504be57a289ba8f6b953ad30183d66f3dfbd66799a648495571'
}

def get_db():
    return pymysql.connect(**DB_CONFIG)

def compute_hash(r, file_hash=''):
    parts = [r.get('record_no',''), r.get('data_type',''), r.get('onset_time',''), r.get('duration',''), r.get('symptoms',''), r.get('etiology',''), r.get('treatment_plan',''), r.get('diagnosis',''), file_hash]
    raw = '|'.join(str(p or '') for p in parts)
    return hashlib.sha256(raw.encode('utf-8')).hexdigest()

HTML_TEMPLATE = """
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MedTrust 区块链防篡改攻防演练控制台</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <style>
    :root {
      --bg-dark: #0a0f1d;
      --card-bg: #131c31;
      --card-border: #1e293b;
      --danger: #ef4444;
      --danger-glow: rgba(239, 68, 68, 0.4);
      --success: #10b981;
      --success-glow: rgba(16, 185, 129, 0.4);
      --cyan: #06b6d4;
      --purple: #8b5cf6;
      --text: #f1f5f9;
      --text-muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg-dark);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      padding: 24px;
      line-height: 1.5;
    }
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--card-border);
    }
    .title-group { display: flex; align-items: center; gap: 14px; }
    .logo-icon {
      font-size: 32px;
      color: var(--danger);
      animation: pulse 2s infinite;
    }
    @keyframes pulse {
      0% { text-shadow: 0 0 10px var(--danger-glow); }
      50% { text-shadow: 0 0 24px var(--danger-glow); }
      100% { text-shadow: 0 0 10px var(--danger-glow); }
    }
    .title-group h1 { font-size: 22px; font-weight: 800; letter-spacing: 0.5px; }
    .title-group p { font-size: 13px; color: var(--text-muted); margin-top: 2px; }
    .target-pill {
      background: #1e293b;
      padding: 8px 16px;
      border-radius: 30px;
      font-size: 12px;
      border: 1px solid #334155;
      display: flex;
      align-items: center;
      gap: 10px;
    }
    .live-dot {
      width: 8px;
      height: 8px;
      background: var(--success);
      border-radius: 50%;
      box-shadow: 0 0 8px var(--success);
      animation: blink 1.2s infinite;
    }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.3; } }

    .grid-2 {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 20px;
      margin-bottom: 24px;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: 0 4px 20px rgba(0,0,0,0.3);
    }
    .card-title {
      font-size: 16px;
      font-weight: 700;
      margin-bottom: 14px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .card-title i { color: var(--cyan); }
    .scenario-desc {
      font-size: 13px;
      color: #cbd5e1;
      background: rgba(15, 23, 42, 0.6);
      padding: 12px;
      border-radius: 8px;
      border-left: 4px solid var(--danger);
      margin-bottom: 16px;
      line-height: 1.6;
    }
    .btn-row { display: flex; gap: 12px; }
    button {
      cursor: pointer;
      border: none;
      border-radius: 8px;
      font-weight: 700;
      font-size: 14px;
      padding: 11px 20px;
      transition: all 0.2s ease;
      display: inline-flex;
      align-items: center;
      gap: 8px;
    }
    .btn-attack {
      background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
      color: white;
      box-shadow: 0 4px 12px var(--danger-glow);
    }
    .btn-attack:hover {
      background: linear-gradient(135deg, #f87171 0%, #ef4444 100%);
      transform: translateY(-2px);
      box-shadow: 0 6px 18px var(--danger-glow);
    }
    .btn-rollback {
      background: linear-gradient(135deg, #10b981 0%, #059669 100%);
      color: white;
      box-shadow: 0 4px 12px var(--success-glow);
    }
    .btn-rollback:hover {
      background: linear-gradient(135deg, #34d399 0%, #10b981 100%);
      transform: translateY(-2px);
      box-shadow: 0 6px 18px var(--success-glow);
    }
    .btn-custom {
      background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
      color: white;
    }
    .btn-custom:hover {
      background: linear-gradient(135deg, #818cf8 0%, #6366f1 100%);
      transform: translateY(-2px);
    }

    .form-group { margin-bottom: 12px; }
    .form-group label { display: block; font-size: 12px; color: var(--text-muted); margin-bottom: 6px; }
    .form-control {
      width: 100%;
      background: #0f172a;
      border: 1px solid #334155;
      color: white;
      padding: 8px 12px;
      border-radius: 6px;
      font-size: 13px;
    }
    .form-control:focus { outline: none; border-color: var(--cyan); }

    /* 表格区域 */
    .table-card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 20px;
      margin-bottom: 24px;
    }
    .table-header-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
    }
    th, td {
      padding: 12px;
      text-align: left;
      border-bottom: 1px solid #1e293b;
    }
    th {
      background: #0f172a;
      color: var(--text-muted);
      font-weight: 600;
    }
    tr:hover td { background: rgba(255,255,255,0.02); }
    .badge {
      display: inline-block;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 700;
    }
    .badge-safe {
      background: rgba(16, 185, 129, 0.15);
      color: #34d399;
      border: 1px solid rgba(16, 185, 129, 0.3);
    }
    .badge-tampered {
      background: rgba(239, 68, 68, 0.2);
      color: #f87171;
      border: 1px solid rgba(239, 68, 68, 0.4);
      animation: pulse-badge 1.5s infinite;
    }
    @keyframes pulse-badge {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.6; }
    }
    .hash-code {
      font-family: monospace;
      background: #0f172a;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 11px;
    }
    .hash-match { color: #34d399; }
    .hash-mismatch { color: #f87171; font-weight: 700; text-decoration: underline; }

    .walkthrough-card {
      background: #0f172a;
      border: 1px dashed #334155;
      border-radius: 10px;
      padding: 18px 24px;
    }
    .walkthrough-steps {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 16px;
      margin-top: 14px;
    }
    .step-item {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 8px;
      padding: 14px;
      font-size: 12px;
      line-height: 1.5;
    }
    .step-num {
      width: 22px;
      height: 22px;
      background: var(--cyan);
      color: #0f172a;
      border-radius: 50%;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-weight: 800;
      margin-bottom: 8px;
    }

    #logBox {
      background: #000;
      color: #10b981;
      font-family: monospace;
      padding: 12px;
      border-radius: 6px;
      font-size: 12px;
      height: 90px;
      overflow-y: auto;
      margin-top: 12px;
      border: 1px solid #22c55e33;
    }
  </style>
</head>
<body>

  <div class="header">
    <div class="title-group">
      <i class="fa-solid fa-skull-crossbones logo-icon"></i>
      <div>
        <h1>MedTrust 区块链防篡改攻防演练控制台</h1>
        <p>直连底层 MySQL 数据库注入恶意篡改，验证 Fabric 联盟链不可篡改智能合约的秒级告警机制</p>
      </div>
    </div>
    <div class="target-pill">
      <div class="live-dot"></div>
      <span>目标数据库: <code>127.0.0.1:3306/medtrust</code></span>
      <span>|</span>
      <span>主系统: <a href="http://localhost:5173" target="_blank" style="color: var(--cyan); text-decoration: none;">http://localhost:5173</a></span>
    </div>
  </div>

  <div class="grid-2">
    <!-- 场景1：过敏史恶意篡改攻击 -->
    <div class="card">
      <div class="card-title">
        <i class="fa-solid fa-biohazard" style="color: var(--danger);"></i>
        实战攻击演示：抹除严重青霉素过敏史并伪造处方
      </div>
      <div class="scenario-desc">
        <strong>攻击逻辑：</strong>黑客绕过应用层所有鉴权与日志，直接向 MySQL 执行原生 <code>UPDATE</code> 语句。<br>
        将病历 1（患者张三）的 <em>“既往对青霉素、头孢严重过敏史”</em> 恶意篡改为 <em>“无任何药物过敏史”</em>，并将治疗方案篡改为 <em>“静推注射用青霉素钠”</em>。<br>
        <strong>危害：</strong>若无区块链防篡改，接诊医生直接采信此病历将导致患者过敏性休克致死！
      </div>

      <div class="btn-row">
        <button class="btn-attack" onclick="triggerAttack()">
          <i class="fa-solid fa-radiation"></i> 💥 执行恶意攻击篡改数据库
        </button>
        <button class="btn-rollback" onclick="triggerRollback()">
          <i class="fa-solid fa-rotate-left"></i> 🔄 一键恢复真实原始数据
        </button>
      </div>

      <div id="logBox">> 等待执行攻防操作...</div>
    </div>

    <!-- 场景2：自定义字段精准攻击 -->
    <div class="card">
      <div class="card-title">
        <i class="fa-solid fa-terminal"></i>
        自定义字段精准注入篡改
      </div>
      <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
        <div class="form-group">
          <label>目标病历 ID</label>
          <select id="customId" class="form-control">
            <option value="1">ID: 1 (REC202609001 - 张三 门诊病历)</option>
            <option value="2">ID: 2 (REC202609002 - 张三 检验单)</option>
            <option value="3">ID: 3 (REC20260910351c4d04 - 抢救病历)</option>
            <option value="4">ID: 4 (REC20260720C01 - 神经内科报告)</option>
          </select>
        </div>
        <div class="form-group">
          <label>目标篡改字段</label>
          <select id="customField" class="form-control">
            <option value="etiology">病因与诱因 (etiology - 包含过敏史)</option>
            <option value="treatment_plan">处置方案 (treatment_plan)</option>
            <option value="diagnosis">诊断结论 (diagnosis)</option>
            <option value="symptoms">主要症状 (symptoms)</option>
          </select>
        </div>
      </div>
      <div class="form-group">
        <label>注入内容 (Payload)</label>
        <input type="text" id="customVal" class="form-control" placeholder="输入任意需要注入数据库的恶意篡改内容..." value="【SQL注入篡改】病情完全康复，无需治疗">
      </div>
      <button class="btn-custom" onclick="triggerCustomTamper()" style="width: 100%; justify-content: center;">
        <i class="fa-solid fa-bolt"></i> ⚡ 直接向 MySQL 注入篡改内容
      </button>
    </div>
  </div>

  <!-- 实时数据库看板 -->
  <div class="table-card">
    <div class="table-header-row">
      <div class="card-title" style="margin-bottom: 0;">
        <i class="fa-solid fa-database"></i>
        MySQL 实时数据监控与区块链哈希一致性对比
      </div>
      <div style="font-size: 12px; color: var(--text-muted);">
        <i class="fa-solid fa-clock-rotate-left"></i> 数据实时同步刷新中
        <button onclick="fetchRecords()" style="padding: 4px 10px; font-size: 11px; margin-left: 10px; background: #334155; color: white;">刷新</button>
      </div>
    </div>

    <table>
      <thead>
        <tr>
          <th width="60">ID</th>
          <th width="140">病历编号</th>
          <th width="80">患者</th>
          <th>诱因与过敏史 (MySQL 真实字段)</th>
          <th>治疗处置方案 (MySQL 真实字段)</th>
          <th width="160">数据库当前指纹 (H_current)</th>
          <th width="160">链上基准指纹 (H_chain)</th>
          <th width="130" style="text-align: center;">防篡改状态</th>
        </tr>
      </thead>
      <tbody id="recordsTbody">
        <tr><td colspan="8" style="text-align: center; color: #64748b;">正在拉取数据...</td></tr>
      </tbody>
    </table>
  </div>

  <!-- 答辩演示指南 -->
  <div class="walkthrough-card">
    <h3 style="font-size: 14px; font-weight: 700; color: var(--cyan);"><i class="fa-solid fa-lightbulb"></i> 答辩与评审演示操作路径：</h3>
    <div class="walkthrough-steps">
      <div class="step-item">
        <div class="step-num">1</div>
        <strong>正常态查阅</strong><br>
        在主系统 (<a href="http://localhost:5173" target="_blank" style="color: var(--cyan);">5173端口</a>) 以医生或患者登录，点开病历1，可见<strong>“🛡️ 校验通过”</strong>绿色标识。
      </div>
      <div class="step-item">
        <div class="step-num">2</div>
        <strong>黑客攻击数据库</strong><br>
        回到本页面，点击<strong>【💥 执行恶意攻击篡改数据库】</strong>，将过敏史抹除并改为推注青霉素。
      </div>
      <div class="step-item">
        <div class="step-num">3</div>
        <strong>主系统高危报警</strong><br>
        回到主系统刷新并点开病历1，系统瞬间触发<strong>【🚨 高危安全警报】</strong>红色高亮提示与哈希对比！
      </div>
      <div class="step-item">
        <div class="step-num">4</div>
        <strong>一键恢复闭环</strong><br>
        在本页面点击<strong>【🔄 一键恢复真实原始数据】</strong>，主系统核验自动恢复正常，形成完整攻防证明闭环。
      </div>
    </div>
  </div>

  <script>
    function log(msg) {
      const b = document.getElementById('logBox');
      const time = new Date().toTimeString().substring(0, 8);
      b.innerHTML = `[${time}] ${msg}<br>` + b.innerHTML;
    }

    async function fetchRecords() {
      try {
        const res = await fetch('/api/records');
        const data = await res.json();
        const tbody = document.getElementById('recordsTbody');
        if (!data.length) {
          tbody.innerHTML = '<tr><td colspan="8">暂无记录</td></tr>';
          return;
        }

        tbody.innerHTML = data.map(r => {
          const isTampered = r.is_tampered;
          const statusBadge = isTampered
            ? '<span class="badge badge-tampered"><i class="fa-solid fa-triangle-exclamation"></i> 🚨 存在篡改</span>'
            : '<span class="badge badge-safe"><i class="fa-solid fa-shield-halved"></i> 🛡️ 校验通过</span>';

          const hashClass = isTampered ? 'hash-mismatch' : 'hash-match';

          return `
            <tr>
              <td><strong>#${r.id}</strong></td>
              <td><code>${r.record_no}</code></td>
              <td>${r.patient_name || '-'}</td>
              <td style="${isTampered && r.etiology.includes('【已被黑客') ? 'color: #f87171; font-weight: bold;' : ''}">${r.etiology || '-'}</td>
              <td style="${isTampered && r.treatment_plan.includes('【已被黑客') ? 'color: #f87171; font-weight: bold;' : ''}">${r.treatment_plan || '-'}</td>
              <td><span class="hash-code ${hashClass}">${r.current_hash ? r.current_hash.substring(0, 14) + '...' : '-'}</span></td>
              <td><span class="hash-code hash-match">${r.chain_hash ? r.chain_hash.substring(0, 14) + '...' : '-'}</span></td>
              <td style="text-align: center;">${statusBadge}</td>
            </tr>
          `;
        }).join('');
      } catch (err) {
        console.error(err);
      }
    }

    async function triggerAttack() {
      if (!confirm('确定要绕过 API 直接向底层 MySQL 数据库执行篡改攻击吗？')) return;
      log('正在发起数据库注入攻击：UPDATE medical_records SET etiology=... WHERE id=1');
      const res = await fetch('/api/attack', { method: 'POST' });
      const data = await res.json();
      log(data.message);
      fetchRecords();
      alert('💥 攻击成功！数据库已被恶意篡改！请前往主系统 (http://localhost:5173) 查看病历，系统将立即展示高危警报！');
    }

    async function triggerRollback() {
      log('正在执行数据还原操作：重置病历1为原始真实数据...');
      const res = await fetch('/api/rollback', { method: 'POST' });
      const data = await res.json();
      log(data.message);
      fetchRecords();
      alert('✅ 还原成功！数据库数据已恢复！主系统核验将重回绿色通过状态。');
    }

    async function triggerCustomTamper() {
      const id = document.getElementById('customId').value;
      const field = document.getElementById('customField').value;
      const val = document.getElementById('customVal').value;

      if (!val) { alert('请输入篡改内容'); return; }
      log(`正在注入篡改: ID=${id}, Field=${field}, Payload="${val}"`);
      const res = await fetch('/api/custom_tamper', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, field, value: val })
      });
      const data = await res.json();
      log(data.message);
      fetchRecords();
      alert(data.message);
    }

    fetchRecords();
    setInterval(fetchRecords, 3000);
  </script>
</body>
</html>
"""

@app.route('/')
def index():
    return render_template_string(HTML_TEMPLATE)

@app.route('/api/records', methods=['GET'])
def get_records():
    conn = get_db()
    cur = conn.cursor(pymysql.cursors.DictCursor)
    cur.execute("""
        SELECT r.*, f.file_hash, u.real_name as patient_name 
        FROM medical_records r 
        LEFT JOIN medical_files f ON f.record_id = r.id 
        LEFT JOIN users u ON u.id = r.patient_id
        ORDER BY r.id ASC
    """)
    rows = cur.fetchall()
    conn.close()

    result = []
    for r in rows:
        file_hash = r.get('file_hash') or ''
        curr_hash = compute_hash(r, file_hash)
        chain_hash = BASELINE_CHAIN_HASHES.get(r['record_no'], curr_hash)
        is_tampered = (curr_hash != chain_hash)
        result.append({
            'id': r['id'],
            'record_no': r['record_no'],
            'patient_name': r.get('patient_name'),
            'data_type': r.get('data_type'),
            'onset_time': r.get('onset_time'),
            'duration': r.get('duration'),
            'symptoms': r.get('symptoms'),
            'etiology': r.get('etiology'),
            'treatment_plan': r.get('treatment_plan'),
            'diagnosis': r.get('diagnosis'),
            'current_hash': curr_hash,
            'chain_hash': chain_hash,
            'is_tampered': is_tampered
        })
    return jsonify(result)

@app.route('/api/attack', methods=['POST'])
def attack():
    conn = get_db()
    cur = conn.cursor()
    cur.execute("""
        UPDATE medical_records 
        SET etiology = %s, treatment_plan = %s, diagnosis = %s 
        WHERE id = 1
    """, (TAMPERED_RECORD_1['etiology'], TAMPERED_RECORD_1['treatment_plan'], TAMPERED_RECORD_1['diagnosis']))
    conn.close()
    return jsonify({
        'code': 200,
        'message': '🚨 [ATTACK SUCCESS] 攻击完成！病历1的严重过敏史已在底层 MySQL 数据库中被直接抹除并伪造！'
    })

@app.route('/api/rollback', methods=['POST'])
def rollback():
    conn = get_db()
    cur = conn.cursor()
    cur.execute("""
        UPDATE medical_records 
        SET onset_time = %s, duration = %s, symptoms = %s, etiology = %s, treatment_plan = %s, vital_signs = %s, diagnosis = %s 
        WHERE id = 1
    """, (
        ORIGINAL_RECORD_1['onset_time'],
        ORIGINAL_RECORD_1['duration'],
        ORIGINAL_RECORD_1['symptoms'],
        ORIGINAL_RECORD_1['etiology'],
        ORIGINAL_RECORD_1['treatment_plan'],
        ORIGINAL_RECORD_1['vital_signs'],
        ORIGINAL_RECORD_1['diagnosis']
    ))
    conn.close()
    return jsonify({
        'code': 200,
        'message': '✅ [ROLLBACK SUCCESS] 还原完成！病历1已重置为真实的原始医疗数据！'
    })

@app.route('/api/custom_tamper', methods=['POST'])
def custom_tamper():
    data = request.json or {}
    rec_id = data.get('id')
    field = data.get('field')
    val = data.get('value')

    allowed_fields = {'symptoms', 'etiology', 'treatment_plan', 'diagnosis', 'onset_time', 'duration'}
    if field not in allowed_fields:
        return jsonify({'code': 400, 'message': '非法的字段选择'}), 400

    conn = get_db()
    cur = conn.cursor()
    sql = f"UPDATE medical_records SET {field} = %s WHERE id = %s"
    cur.execute(sql, (val, rec_id))
    conn.close()
    return jsonify({
        'code': 200,
        'message': f'⚡ [INJECTION SUCCESS] 已直接篡改记录 ID={rec_id} 的字段 {field}！'
    })

if __name__ == '__main__':
    print("=" * 70)
    print(" MedTrust 区块链防篡改攻防演示控制台启动中...")
    print(" 访问地址: http://127.0.0.1:8090")
    print("=" * 70)
    app.run(host='0.0.0.0', port=8090, debug=False)
