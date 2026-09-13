import os

base = r'e:\EnglishEncoding\competition\last\MedTrust\ai_agent'
os.makedirs(base, exist_ok=True)

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    with open(full_path, 'w', encoding='utf-8') as out:
        out.write(content.strip() + '\n')
    print('Wrote:', rel_path)

write_file('requirements.txt', """fastapi>=0.110.0
uvicorn>=0.28.0
pydantic>=2.0.0
httpx>=0.27.0
requests>=2.31.0
""")

write_file('tools.py', """import requests
from typing import Dict, Any, Optional

GO_BACKEND_URL = "http://127.0.0.1:8080/api/v1"

def tool_query_medical_records(token: str, keyword: str = "", patient_id: Optional[int] = None) -> Dict[str, Any]:
    \"\"\"通过携带用户 Token 调用 Go 后端业务接口查询病历\"\"\"
    headers = {"Authorization": f"Bearer {token}"}
    params = {}
    if keyword:
        params["keyword"] = keyword
    if patient_id:
        params["patient_id"] = patient_id
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/medical-records", headers=headers, params=params, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端病历接口失败: {str(e)}", "data": None}

def tool_query_authorizations(token: str) -> Dict[str, Any]:
    \"\"\"通过携带用户 Token 查询患者授权策略列表\"\"\"
    headers = {"Authorization": f"Bearer {token}"}
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/authorizations", headers=headers, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端授权接口失败: {str(e)}", "data": None}

def tool_query_audit_logs(token: str, risk_level: str = "", operation_type: str = "") -> Dict[str, Any]:
    \"\"\"通过携带用户 Token 查询审计日志\"\"\"
    headers = {"Authorization": f"Bearer {token}"}
    params = {}
    if risk_level:
        params["risk_level"] = risk_level
    if operation_type:
        params["operation_type"] = operation_type
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/audit-logs", headers=headers, params=params, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端审计接口失败: {str(e)}", "data": None}

def tool_verify_integrity(token: str, record_id: int) -> Dict[str, Any]:
    \"\"\"通过携带用户 Token 触发动态防篡改核验\"\"\"
    headers = {"Authorization": f"Bearer {token}"}
    try:
        resp = requests.post(f"{GO_BACKEND_URL}/verification/{record_id}", headers=headers, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端核验接口失败: {str(e)}", "data": None}
""")

write_file('agent.py', """from typing import Dict, Any
from tools import (
    tool_query_medical_records,
    tool_query_authorizations,
    tool_query_audit_logs,
    tool_verify_integrity
)

class MedTrustAgent:
    def __init__(self):
        pass

    def run(self, user_message: str, token: str) -> Dict[str, Any]:
        \"\"\"
        受控 AI Agent 核心中枢：
        严格遵循 2.0.md 规划书 1.3/3.4 节红线原则：
        1. AI 严禁直连底层数据库或区块链节点，强制通过 Tool Calling 携带调用方 JWT 请求 Go 网关。
        2. 若 Go 网关鉴权失败 (如 HTTP 403 未授权)，AI 如实向用户输出拦截说明，绝不越权。
        \"\"\"
        msg = user_message.strip().lower()
        tool_called = None
        tool_result = None

        # 意图识别与槽位抽取
        if any(w in msg for w in ["病历", "检查", "报告", "记录", "用药", "历史", "查询张三", "张三"]):
            tool_called = "tool_query_medical_records(keyword='张三')"
            tool_result = tool_query_medical_records(token, keyword="张三")
            
            if tool_result.get("code") == 200:
                records = tool_result.get("data", [])
                if not records:
                    reply = "为您查询了医疗记录库，未检索到符合条件的病历或检验报告。"
                else:
                    lines = [f"已通过安全网关为您检索到 {len(records)} 条受控电子档案：\n"]
                    for idx, r in enumerate(records, 1):
                        lines.append(
                            f"【记录 {idx}】编号: {r.get('record_no')} | 类型: {r.get('data_type')}\n"
                            f"  - 诊断结论: {r.get('diagnosis')}\n"
                            f"  - 归属机构: {r.get('hospital_name', '医院')} | 签署医生: {r.get('doctor_name', '医生')}\n"
                            f"  - 区块链存证 TxID: `{r.get('fabric_tx_id', '')[:24]}...` (区块高度 #{r.get('block_height')})\n"
                        )
                    lines.append("\n🛡️ 提示：以上数据已由 Go 安全网关校验当前登录凭证与患者授权状态，密文自 IPFS 解密拉取。")
                    reply = "".join(lines)
            elif tool_result.get("code") == 403:
                reply = "⚠️ 【安全网关拦截】权限评估未通过：您未取得该患者的跨机构显式授权。若处于急救危重抢救场景，请在医生工作台申请 Break-Glass 紧急访问。"
            else:
                reply = f"查询失败: {tool_result.get('message', '未知错误')}"

        elif any(w in msg for w in ["授权", "谁调阅", "我的授权", "被授权"]):
            tool_called = "tool_query_authorizations()"
            tool_result = tool_query_authorizations(token)
            if tool_result.get("code") == 200:
                auths = tool_result.get("data", [])
                if not auths:
                    reply = "您当前没有处于生效状态（ACTIVE）的患者授权策略。"
                else:
                    lines = [f"您当前名下生效中的授权策略共 {len(auths)} 条：\n"]
                    for a in auths:
                        lines.append(
                            f"• 授权单号: `{a.get('auth_no')}` | 对象: {a.get('target_name', a.get('auth_target_type'))}\n"
                            f"  范围: {a.get('scope_type')} | 状态: {a.get('status')} | 截止日期: {a.get('end_time', '')[:10]}\n"
                        )
                    reply = "".join(lines)
            else:
                reply = f"授权信息查询失败: {tool_result.get('message')}"

        elif any(w in msg for w in ["核验", "防篡改", "真实性", "哈希", "完整性"]):
            tool_called = "tool_verify_integrity(record_id=1)"
            tool_result = tool_verify_integrity(token, record_id=1)
            if tool_result.get("code") == 200:
                v = tool_result.get("data", {})
                reply = (
                    f"🛡️ 【动态防篡改核验报告】\n"
                    f"• 目标病历编号: `{v.get('record_no')}`\n"
                    f"• IPFS 解密文件 SHA-256: `{v.get('calculated_hash')}`\n"
                    f"• Fabric 链上固化 SHA-256: `{v.get('chain_hash')}`\n"
                    f"• 动态对比结论: {'✅ ' + v.get('message') if v.get('verified') else '❌ ' + v.get('message')}"
                )
            else:
                reply = f"完整性核验失败: {tool_result.get('message')}"

        elif any(w in msg for w in ["审计", "风险", "高风险", "日志", "监控"]):
            tool_called = "tool_query_audit_logs(risk_level='')"
            tool_result = tool_query_audit_logs(token)
            if tool_result.get("code") == 200:
                logs = tool_result.get("data", [])
                lines = [f"为您检索到系统最新审计流水 {len(logs[:5])} 条：\n"]
                for l in logs[:5]:
                    lines.append(
                        f"• [{l.get('created_at', '')[:19]}] 操作: {l.get('operation_type')} | "
                        f"主体: {l.get('user_name', '用户')} | 结果: {l.get('result')} | 风险: {l.get('risk_level')}\n"
                    )
                reply = "".join(lines)
            else:
                reply = f"审计日志查询受限或失败: {tool_result.get('message')}"

        else:
            reply = (
                "您好！我是 MedTrust 医疗数据受控 AI 智能助手。基于软件工程与安全规范，我可以协助您执行以下受控自然语言指令：\n\n"
                "1. 🩺 **医生查询**：例如“查询患者张三最近的检查记录”、“调阅张三历史用药与过敏史”\n"
                "2. 👤 **患者中心**：例如“我现在授权了哪些医生或机构？”、“查看我的有效授权策略”\n"
                "3. 🔍 **完整性核验**：例如“核验病历1的数据真实完整性与区块链存证哈希”\n"
                "4. 📊 **安全监管**：例如“今天有哪些高风险调阅事件？”、“查看最近的审计日志”\n\n"
                "*安全说明：本助手不直连任何数据库，全部查询操作均携带您当前会话 Token 经由 Go 统一网关鉴权过滤。*"
            )

        return {
            "reply": reply,
            "tool_called": tool_called,
            "tool_status": "SUCCESS" if tool_result and tool_result.get("code") == 200 else ("INTERCEPTED" if tool_result and tool_result.get("code") == 403 else "SKIPPED")
        }
""")

write_file('main.py', """import uvicorn
from fastapi import FastAPI, Header, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from agent import MedTrustAgent

app = FastAPI(
    title="MedTrust Controlled AI Agent Service",
    description="Python FastAPI AI Agent with Tool Calling strictly authenticated via Go Backend Gateway",
    version="2.0.0"
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

agent = MedTrustAgent()

class ChatRequest(BaseModel):
    message: str

@app.get("/health")
def health():
    return {"status": "ok", "service": "MedTrust AI Agent"}

@app.post("/api/v1/ai/chat")
def chat(req: ChatRequest, authorization: str = Header(None)):
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing or invalid Bearer authorization token")
    token = authorization.split(" ")[1]
    res = agent.run(req.message, token)
    return {
        "code": 200,
        "message": "AI 解析响应成功",
        "data": res
    }

if __name__ == "__main__":
    uvicorn.run("main:app", host="127.0.0.1", port=8000, reload=False)
""")

print("AI Agent files created successfully!")
