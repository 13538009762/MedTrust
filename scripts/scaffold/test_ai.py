import urllib.request
import json
import sys

# Ensure UTF-8 output on Windows
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8")

req = urllib.request.Request(
    'http://127.0.0.1:8080/api/v1/auth/login',
    data=json.dumps({'username': 'doc_a', 'password': '123456'}).encode('utf-8'),
    headers={'Content-Type': 'application/json'}
)
resp = urllib.request.urlopen(req)
login_data = json.loads(resp.read().decode('utf-8'))
token = login_data['data']['token']

prompts = [
    '帮我核验病历1的数据真实性与区块链存证哈希',
    '查询患者张伟的历史电子病历记录',
    '今天有哪些高风险审计日志',
    '查看我的授权策略'
]

for p in prompts:
    ai_req = urllib.request.Request(
        'http://127.0.0.1:8000/api/v1/ai/chat',
        data=json.dumps({'message': p}).encode('utf-8'),
        headers={'Content-Type': 'application/json', 'Authorization': f'Bearer {token}'}
    )
    res = json.loads(urllib.request.urlopen(ai_req).read().decode('utf-8'))
    print(f"=== Prompt: {p} ===")
    print("Tool Called:", res['data']['tool_called'])
    print("Tool Status:", res['data']['tool_status'])
    print("Reply:\n" + res['data']['reply'])
    print("\n" + "="*50 + "\n")
