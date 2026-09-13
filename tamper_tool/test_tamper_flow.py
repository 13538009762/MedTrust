# -*- coding: utf-8 -*-
import sys, requests, json

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

print('=== 1. 检查初始状态 ===')
res_records = requests.get('http://127.0.0.1:8090/api/records').json()
r1 = next(r for r in res_records if r['id'] == 1)
print(f'Tamper Tool 初始状态: is_tampered={r1["is_tampered"]}')

login = requests.post('http://127.0.0.1:8080/api/v1/auth/login', json={'username':'doc_a', 'password':'123456'}).json()
token = login['data']['token']

res_med = requests.get('http://127.0.0.1:8080/api/v1/medical-records/1', headers={'Authorization': 'Bearer ' + token}).json()
d1 = res_med['data']
print(f'MedTrust 主后端初始状态: is_tampered={d1.get("is_tampered")}, verified={d1.get("verified")}')

print('\n=== 2. 发起黑客攻击篡改数据库 ===')
attack_res = requests.post('http://127.0.0.1:8090/api/attack').json()
print('攻击响应:', attack_res['message'])

print('\n=== 3. 验证主后端即时检测并报警 ===')
res_tampered = requests.get('http://127.0.0.1:8080/api/v1/medical-records/1', headers={'Authorization': 'Bearer ' + token}).json()
d_t = res_tampered['data']
print(f'MedTrust 主后端检测结果:')
print(f'  - is_tampered:   {d_t.get("is_tampered")}')
print(f'  - verified:      {d_t.get("verified")}')
print(f'  - chain_hash:    {d_t.get("chain_hash")}')
print(f'  - current_hash:  {d_t.get("current_hash")}')
print(f'  - tamper_reason: {d_t.get("tamper_reason")}')

print('\n=== 4. 发起一键还原修复 ===')
rollback_res = requests.post('http://127.0.0.1:8090/api/rollback').json()
print('还原响应:', rollback_res['message'])

res_restored = requests.get('http://127.0.0.1:8080/api/v1/medical-records/1', headers={'Authorization': 'Bearer ' + token}).json()
d_r = res_restored['data']
print(f'MedTrust 还原后状态: is_tampered={d_r.get("is_tampered")}, verified={d_r.get("verified")}')

assert d1.get("is_tampered") is False
assert d_t.get("is_tampered") is True
assert d_r.get("is_tampered") is False
print('\n🎉 [SUCCESS] 完整攻防闭环测试全部通过！系统已完美具备区块链防篡改感知识别与告警能力！')
