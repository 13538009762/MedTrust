import json
import os
import re
from typing import Dict, Any
from tools import (
    ControlledMedicalTools,
    PermissionError,
    extract_jwt_claims,
    is_tool_allowed_for_role,
    verify_session_with_gateway,
    MEDTRUST_JWT_SECRET,
    GO_BACKEND_URL,
)

def sanitize_agent_output(text: str) -> str:
    """对智能体输出执行严格的数据脱敏防护，防范 JWT、密钥、电话、身份证和密码泄露"""
    if not text:
        return ""
    # 1. 脱敏 Bearer JWT Token
    text = re.sub(r'Bearer\s+[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+', 'Bearer [JWT_TOKEN_MASKED]', text)
    text = re.sub(r'eyJ[A-Za-z0-9-_]{10,}\.[A-Za-z0-9-_]{10,}\.[A-Za-z0-9-_]{10,}', '[JWT_TOKEN_MASKED]', text)
    # 2. 脱敏中国大陆手机号 (11位)
    text = re.sub(r'(?<!\d)(1[3-9]\d)\d{4}(\d{4})(?!\d)', r'\1****\2', text)
    # 3. 脱敏身份证号 (18位)
    text = re.sub(r'(?<!\d)(\d{6})(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])(\d{3}[\dXx])(?!\d)', r'\1********\2', text)
    # 4. 脱敏密码和密钥字样
    text = re.sub(r'(password|passwd|secret|medical_key|root_key|mysql_pwd)\s*[:=]\s*[^\s,;]+', r'\1=********', text, flags=re.IGNORECASE)
    return text

# WP-06: 外部 LLM (OpenAI/DeepSeek 兼容协议) Function-Calling 配置检测
_LLM_API_KEY = os.getenv("OPENAI_API_KEY") or os.getenv("DEEPSEEK_API_KEY") or os.getenv("AI_API_KEY", "")
_LLM_API_URL = os.getenv("OPENAI_API_URL") or os.getenv("AI_API_URL", "")
_LLM_MODEL = os.getenv("AI_MODEL", "deepseek-chat")

# 受控工具的 OpenAI function-calling 声明 (4 个白名单工具)
CONTROLLED_TOOLS_SCHEMA = [
    {
        "type": "function",
        "function": {
            "name": "search_medical_records",
            "description": "检索当前用户有权查看的医疗记录 (严格携带 JWT 经 Go 网关鉴权)",
            "parameters": {
                "type": "object",
                "properties": {
                    "keyword": {"type": "string", "description": "患者姓名或病历单号"},
                },
                "required": [],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "check_consent_status",
            "description": "查询患者知情授权策略状态列表",
            "parameters": {"type": "object", "properties": {}, "required": []},
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_audit_trail",
            "description": "检索审计日志与高风险访问行为记录",
            "parameters": {
                "type": "object",
                "properties": {
                    "risk_level": {"type": "string", "enum": ["LOW", "MEDIUM", "HIGH"]},
                },
                "required": [],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "summarize_patient_soap",
            "description": "生成患者跨院 SOAP 病程摘要 (仅医生可用，零诊断零处方)",
            "parameters": {
                "type": "object",
                "properties": {"patient_id": {"type": "integer", "description": "患者 ID"}},
                "required": ["patient_id"],
            },
        },
    },
]

_FORBIDDEN_DIAGNOSIS_INTENT = [
    "开处方", "推荐药物", "诊断我", "得了什么病", "该吃什么药", "确诊", "自动确诊",
    "替我授权", "代我审批", "自动审批",
]

# 非本软件领域的问题探测特征 (Out-of-Domain Detection Patterns)
OUT_OF_DOMAIN_PATTERNS = [
    r"天气",
    r"写.*?(?:诗|歌词|散文|作文|故事|文章|小说)",
    r"(?:讲|说).*?笑话",
    r"翻译.*?(?:英语|英文|中文|日文|日语|法语|韩语)",
    r"(?:写|编写|实现).*?(?:代码|程序|脚本|快速排序|二叉树|冒泡|算法)",
    r"(?:帮我|给我)?(?:算一下|计算一下)",
    r"(?:你觉得|你认识|知道).*?(?:明星|游戏|王者荣耀|原神|电影|动漫)",
    r"(?:法国|美国|英国|中国|日本).*?首都是",
    r"世界.*?有多少",
    r"聊(?:聊)?天",
]

STANDARD_OUT_OF_DOMAIN_REPLY = (
    "【服务范畴受限提醒】\n"
    "您好！我是 MedTrust 跨机构医疗数据可信共享系统的专有受控 AI 助手。\n"
    "依据系统安全策略与合规界定，本助手仅服务于本平台内部的【病历调阅、知情授权、区块链存证核验与合规审计】业务，坚决不回答与本软件无关的通用问题（如日常闲聊、天气查询、代码编写、文学诗歌、外部常识等）。\n"
    "如需办理医疗数据业务，请向我发送与本系统相关的病历查询、授权管理、存证核验或审计追溯指令。"
)


def _is_forbidden_diagnosis_intent(msg: str) -> bool:
    return any(w in msg for w in _FORBIDDEN_DIAGNOSIS_INTENT)


def _execute_controlled_tool(tools: ControlledMedicalTools, name: str, args: Dict[str, Any]) -> Dict[str, Any]:
    """受控工具分发——LLM 只能调用这 4 个白名单工具，全部经 JWT 转发 Go 网关"""
    if name == "search_medical_records":
        return tools.query_patient_records(keyword=args.get("keyword", ""))
    if name == "check_consent_status":
        return tools.query_my_authorizations()
    if name == "get_audit_trail":
        return tools.analyze_audit_anomalies(risk_level=args.get("risk_level", ""))
    if name == "summarize_patient_soap":
        return tools.summarize_patient_soap(args.get("patient_id", 0))
    return {"code": 400, "message": f"未知或未授权的工具: {name}", "data": None}


class MedTrustAgent:
    """
    MedTrust 受控医疗数据 AI 智能体 (Controlled Assistant Agent)
    严格遵循软件工程边界设计（docs/2.0.md 3.4 节）：
    1. 坚决不配置任何 MySQL、IPFS 或 Fabric 底层驱动，零直连隐患
    2. 严格践行 Single Gatekeeper 原则，仅通过 Tool Calling 携带当前调用者 JWT Bearer Token 请求 Go 后端 RESTful API
    3. 严格受到 Go 端 RBAC 角色过滤、ABAC 环境属性判定、患者知情授权与 RiskEngine 规则引擎审查
    4. 严守红线原则：严禁做出自动疾病诊断、处方推荐、自动授权或越权审批
    5. 工具调用前严格实施角色-工具白名单准入控制与参数模式校验
    6. 输出结果全面执行敏感信息过滤脱敏与审计链路跟踪 (tool_trace)
    """
    def __init__(self):
        pass

    def _run_function_calling(self, msg: str, tools: ControlledMedicalTools, user_role: str, user_name: str):
        """WP-06: OpenAI 兼容 Controlled Tool Calling 模式。
        - LLM 仅可在 4 个白名单工具中选择，工具入参模式由 schema 限定；
        - 每次调用前仍执行角色-工具白名单 (is_tool_allowed_for_role)；
        - 任何异常 (网络/解析/超时) 返回 None，由外层确定性意图路由接管，保障离线可用。"""
        import requests

        tool_name_map = {
            "search_medical_records": "tool_query_medical_records",
            "check_consent_status": "tool_query_authorizations",
            "get_audit_trail": "tool_query_audit_logs",
            "summarize_patient_soap": "tool_summarize_soap",  # WP-06: SOAP 临床聚合仅 doctor，与 supervisor 的核验工具严格隔离
        }
        try:
            ok, status_code, err_msg = verify_session_with_gateway(tools.token)
            if not ok:
                return {
                    "reply": f"【鉴权失败 ({status_code})】{err_msg}，已拒绝调用外部大模型。",
                    "tool_called": None,
                    "tool_status": "AUTH_REJECTED",
                    "tool_trace": tools.trace_logs,
                }
            system_prompt = (
                "你是 MedTrust 跨医院医疗数据可信共享系统的专有受控 AI 助手。\n"
                "【严格领域服务边界】：你仅被授权处理与「MedTrust 医疗数据共享平台」直接相关的内部业务，包含且仅包含："
                "1) 经授权的跨机构电子病历与检查档案检索；2) 患者知情同意与授权策略状态查询；3) 区块链存证完整性与防篡改核验；4) 急诊破窗与安全审计追溯。\n"
                "【绝对负向约束】：严禁回答任何非本软件相关的通用问题（包括闲聊日常、天气预报、文学诗歌、代码编程、数学计算、历史百科、影视娱乐、中英翻译等）。"
                "严禁提供脱离系统病历记录的外部疾病确诊、处方推荐或自动代行授权。\n"
                "【统一拒答规范】：若用户提问超出上述 MedTrust 系统业务范围，你必须坚决且礼貌地予以拒答，明确告知用户你仅服务于 MedTrust 医疗数据可信共享系统，无法回答非本系统相关的问题。"
            )
            resp = requests.post(
                _LLM_API_URL.rstrip("/") + "/chat/completions",
                headers={"Authorization": f"Bearer {_LLM_API_KEY}", "Content-Type": "application/json"},
                json={
                    "model": _LLM_MODEL,
                    "messages": [
                        {"role": "system", "content": system_prompt},
                        {"role": "user", "content": msg},
                    ],
                    "tools": CONTROLLED_TOOLS_SCHEMA,
                    "tool_choice": "auto",
                    "temperature": 0.2,
                },
                timeout=15,
            )
            if resp.status_code != 200:
                return None
            choices = resp.json().get("choices", [])
            if not choices:
                return None
            message = choices[0].get("message", {})

            # 纯文本回复 (无工具调用)
            if not message.get("tool_calls"):
                reply = message.get("content") or ""
                if any(re.search(pat, msg, re.IGNORECASE) for pat in OUT_OF_DOMAIN_PATTERNS):
                    return {
                        "reply": STANDARD_OUT_OF_DOMAIN_REPLY,
                        "tool_called": "REJECTED_AS_OUT_OF_DOMAIN",
                        "tool_status": "INTERCEPTED",
                        "tool_trace": tools.trace_logs,
                    }
                return {
                    "reply": sanitize_agent_output(reply) or "请明确您要检索的病历、授权、审计或摘要内容。",
                    "tool_called": None,
                    "tool_status": "SKIPPED",
                    "tool_trace": tools.trace_logs,
                }

            # 逐个执行受控工具调用 (仅第一个；多轮会话本版本不展开)
            call = message["tool_calls"][0]
            fn = call.get("function", {})
            name = fn.get("name", "")
            try:
                args = json.loads(fn.get("arguments") or "{}")
            except json.JSONDecodeError:
                args = {}

            rbac_name = tool_name_map.get(name)
            if rbac_name is None or (user_role and not is_tool_allowed_for_role(rbac_name, user_role)):
                return {
                    "reply": f"【角色权限拦截 (403 Forbidden)】当前角色「{user_role or '未知'}」无权调用工具 {name}。",
                    "tool_called": name,
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs,
                }

            result = _execute_controlled_tool(tools, name, args)
            data_preview = json.dumps(result.get("data"), ensure_ascii=False)[:1500] if isinstance(result, dict) else ""
            reply = (
                f"已通过受控工具 {name} 检索完成 (HTTP {result.get('code', 500)})。\n"
                f"结果摘要: {data_preview or result.get('message', '无数据')}\n"
                "安全提示：以上数据经由 Go 统一安全网关鉴权与脱敏后返回。"
            )
            return {
                "reply": sanitize_agent_output(reply),
                "tool_called": name,
                "tool_status": "EXECUTED",
                "tool_trace": tools.trace_logs,
            }
        except Exception:
            # 网络/解析异常——静默回落到确定性意图路由
            return None

    def run(self, user_message: str, token: str) -> dict:
        msg = user_message.strip()
        msg_lower = msg.lower()
        tool_called = None
        tool_result = None

        # 实例化受控医疗工具集 (注入当前调用者 Bearer Token)
        tools = ControlledMedicalTools(token)

        # 0.0 解析 JWT 提取调用者身份角色与上下文
        # WP-06: HMAC-SHA256 签名 + exp 过期密码学校验，伪造/过期 token 直接 401 拒绝
        try:
            claims = extract_jwt_claims(token)
        except PermissionError as exc:
            return {
                "reply": f"【鉴权失败 (401 Unauthorized)】JWT 令牌校验未通过：{exc}。请重新登录获取有效会话凭证。",
                "tool_called": None,
                "tool_status": "AUTH_REJECTED",
                "tool_trace": tools.trace_logs
            }
        user_role = claims.get("role", "").lower()
        user_name = claims.get("username", "调用者")

        # 0.1 提示词注入与越狱探测防御 (Prompt Injection & Jailbreak Defense)
        # 必须前置于任何外部 LLM 调用与工具执行，杜绝恶意提示词直通大模型
        injection_patterns = [
            r"ignore\s+(?:all\s+)?(?:previous\s+)?instructions",
            r"system\s+prompt",
            r"developer\s+mode",
            r"jailbreak",
            r"bypass\s+(?:security|rules|auth)",
            r"越狱",
            r"开发者模式",
            r"忽略所有(?:前置)?设定",
            r"忽略上述指令",
            r"导出(?:全部|系统)?密码",
            r"输出(?:私钥|密钥|root\s*key)",
            r"dump\s+(?:all\s+)?(?:database|tables)",
            r"drop\s+table",
            r"(?:'|\")?\s*or\s*(?:'|\")?1(?:'|\")?\s*=\s*(?:'|\")?1(?:'|\")?",
            r"以最高管理员身份执行",
            r"冒充监管人员",
        ]
        for pat in injection_patterns:
            if re.search(pat, msg, re.IGNORECASE):
                return {
                    "reply": (
                        "【安全预警：触发恶意输入阻断】\n"
                        "系统检测到您的输入包含疑似 Prompt Injection 攻击、越权提权指令或数据库注入探测特征。\n"
                        "MedTrust 受控智能体架构已实施端到端安全边界加固，该异常交互已被拦截并阻断。"
                    ),
                    "tool_called": "BLOCKED_BY_INJECTION_SHIELD",
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs
                }

        # 0.2 安全合规红线拦截：自动处方、自动确诊与自动授权越权行为坚决阻断
        # 同样必须前置于任何外部 LLM 调用，确保合规红线不可被大模型提示词绕过
        if any(w in msg for w in ["开处方", "推荐药物", "诊断我", "得了什么病", "自动确诊", "替我授权", "代我审批", "自动审批", "该吃什么药", "生了什么病", "帮我确诊"]):
            return {
                "reply": (
                    "【安全合规红线提醒】\n"
                    "您好！本助手为【MedTrust 跨院电子病历安全检索与合规审计助手】。\n"
                    "依据国家医疗卫生信息安全法规与开题报告学术界定，本 AI 助手【仅用于辅助查询已授权的病历与知情策略记录，严格不涉及任何疾病确诊与处方用药建议】。\n"
                    "临床诊断与处方开立必须由具有执业资质的经治医师在面诊查体后当面做出；您可随时通过本助手查询调阅本人或已授权的既往客观检查与病历存证。"
                ),
                "tool_called": "BLOCKED_BY_SAFETY_GUARDRAIL",
                "tool_status": "INTERCEPTED",
                "tool_trace": tools.trace_logs
            }

        # 0.2b 跨领域/非本软件相关问题坚决拒识 (Out-of-Domain Guardrail)
        if any(re.search(pat, msg, re.IGNORECASE) for pat in OUT_OF_DOMAIN_PATTERNS):
            return {
                "reply": STANDARD_OUT_OF_DOMAIN_REPLY,
                "tool_called": "REJECTED_AS_OUT_OF_DOMAIN",
                "tool_status": "INTERCEPTED",
                "tool_trace": tools.trace_logs
            }

        # 0.3 外部 LLM Function-Calling 模式 (已配置 API Key 时启用；安全与红线已在上方 0.1/0.2 严格把关)
        if _LLM_API_KEY and _LLM_API_URL:
            llm_result = self._run_function_calling(msg, tools, user_role, user_name)
            if llm_result is not None:
                return llm_result

        # 0.9 受控 SOAP 病程摘要 (doctor 专属；统一走 Go /ai/diagnostic-assist 边界)
        if any(w in msg for w in ["SOAP", "soap", "病程摘要", "摘要", "小结"]) and user_role == "doctor":
            m_pid = re.search(r'(\d+)', msg)
            pid = int(m_pid.group(1)) if m_pid else 0
            if pid <= 0:
                return {
                    "reply": "请明确患者 ID（例如：“生成患者 3 的 SOAP 病程摘要”），以便经统一安全网关调取其授权病历。",
                    "tool_called": "NEED_PARAMETER",
                    "tool_status": "PROMPT_USER",
                    "tool_trace": tools.trace_logs
                }
            tool_called = f"summarize_patient_soap(patient_id={pid})"
            tool_result = tools.summarize_patient_soap(pid)
            if tool_result.get("code") == 200:
                d = tool_result.get("data") or {}
                soap = d.get("soap_summary") or {}
                reply = (
                    f"【SOAP 跨院病程摘要】患者: {d.get('patient_name')} | 授权病历: {d.get('authorized_record_count')} 份\n"
                    f"* S (主诉): {soap.get('subjective', '')}\n"
                    f"* O (客观): {soap.get('objective', '')}\n"
                    f"* A (评估): {soap.get('assessment', '')}\n"
                    f"* P (计划): {soap.get('plan', '')}\n"
                    "合规声明：以上为已授权历史病历客观摘要，不含新诊断与新处方。"
                )
            else:
                reply = f"SOAP 摘要生成失败或被拦截: {tool_result.get('message')}"

        # 1. 动态防篡改完整性核验
        elif any(w in msg for w in ["核验", "防篡改", "真实性", "哈希", "完整性", "篡改"]):
            tool_name = "tool_verify_integrity"
            if user_role and not is_tool_allowed_for_role(tool_name, user_role):
                return {
                    "reply": f"【角色权限拦截 (403 Forbidden)】当前角色「{user_role}」在受控智能体策略中无权调用防篡改核验工具（仅医生与监管员可用）。",
                    "tool_called": tool_name,
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs
                }

            m = re.search(r'(\d+)', msg)
            if not m:
                return {
                    "reply": "请指定需要核验的病历 ID（例如：“核验病历 1 的防篡改完整性”），系统将联动 IPFS 与 Fabric 账本执行三阶段验真。",
                    "tool_called": "NEED_PARAMETER",
                    "tool_status": "PROMPT_USER",
                    "tool_trace": tools.trace_logs
                }
            rec_id = int(m.group(1))
            tool_called = f"verify_record_integrity(record_id={rec_id})"
            tool_result = tools.verify_record_integrity(rec_id)
            if tool_result.get("code") == 200:
                v = tool_result.get("data", {})
                is_verified = v.get("verified", False)
                symbol = "✅ [PASS 真实可信]" if is_verified else "🚨 [ALERT 疑似篡改]"
                reply = (
                    f"【MedTrust 区块链动态防篡改核验报告】\n"
                    f"* 目标病历编号: {v.get('record_no')}\n"
                    f"* IPFS 解密明文实时 SHA-256: {v.get('calculated_hash')}\n"
                    f"* Fabric 账本固化原始指纹: {v.get('chain_hash')}\n"
                    f"* 区块链存证交易凭证 TxID: {v.get('fabric_tx_id', '')}\n"
                    f"* 动态核验判定: {symbol} {v.get('message')}"
                )
            else:
                reply = f"完整性核验失败: {tool_result.get('message')}"

        # 2. 监管审计流水与高风险事件检索
        elif any(w in msg for w in ["审计", "风险", "高风险", "日志", "监控", "拦截", "流水", "谁访问过我的数据"]):
            tool_name = "tool_query_audit_logs"
            if user_role and not is_tool_allowed_for_role(tool_name, user_role):
                return {
                    "reply": f"【角色权限拦截 (403 Forbidden)】当前角色「{user_role}」在受控智能体策略中无权调用审计日志工具。",
                    "tool_called": tool_name,
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs
                }

            risk_level = ""
            if "高风险" in msg or "high" in msg_lower:
                risk_level = "HIGH"
            elif "中风险" in msg or "medium" in msg_lower:
                risk_level = "MEDIUM"
            elif "低风险" in msg or "low" in msg_lower:
                risk_level = "LOW"

            tool_called = f"analyze_audit_anomalies(risk_level='{risk_level}')"
            tool_result = tools.analyze_audit_anomalies(risk_level=risk_level)
            if tool_result.get("code") == 200:
                logs = tool_result.get("data", [])
                if not logs:
                    reply = f"系统中暂无匹配的审计流水记录 (筛选条件: 风险等级={risk_level or '全部'})。"
                else:
                    count = min(len(logs), 5)
                    lines = [f"为您检索到系统最新审计流水 {count} 条：\n"]
                    for l in logs[:count]:
                        lines.append(
                            f"* [{str(l.get('created_at', ''))[:19]}] 操作: {l.get('operation_type')} | "
                            f"主体: {l.get('user_name', '用户')} | 结果: {l.get('result')} | 风险: {l.get('risk_level')}\n"
                        )
                    reply = "".join(lines)
            else:
                reply = f"审计日志查询受限或失败: {tool_result.get('message')}"

        # 3. 患者知情授权策略查询
        elif any(w in msg for w in ["授权", "谁调阅", "我的授权", "被授权", "权限"]):
            tool_name = "tool_query_authorizations"
            if user_role and not is_tool_allowed_for_role(tool_name, user_role):
                return {
                    "reply": f"【角色权限拦截 (403 Forbidden)】当前角色「{user_role}」在受控智能体策略中无权调用授权策略查询工具（仅患者本人可用）。",
                    "tool_called": tool_name,
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs
                }

            tool_called = "query_my_authorizations()"
            tool_result = tools.query_my_authorizations()
            if tool_result.get("code") == 200:
                auths = tool_result.get("data", [])
                if not auths:
                    reply = "您当前没有处于生效状态（ACTIVE）的患者知情授权策略。"
                else:
                    lines = [f"名下生效中的授权策略共 {len(auths)} 条：\n"]
                    for a in auths:
                        lines.append(
                            f"* 授权业务号: {a.get('auth_no')} | 对象: {a.get('target_name', a.get('auth_target_type'))}\n"
                            f"  范围: {a.get('scope_type')} | 状态: {a.get('status')} | 失效时间: {str(a.get('end_time', ''))[:10]}\n"
                        )
                    reply = "".join(lines)
            else:
                reply = f"授权信息查询失败: {tool_result.get('message')}"

        # 4. 临床病历/检验报告受控检索
        elif any(w in msg for w in ["病历", "检查", "报告", "记录", "用药", "历史", "就诊", "患者", "档案"]):
            tool_name = "tool_query_medical_records"
            if user_role and not is_tool_allowed_for_role(tool_name, user_role):
                return {
                    "reply": f"【角色权限拦截 (403 Forbidden)】当前角色「{user_role}」在受控智能体策略中无权直接检索临床病历。",
                    "tool_called": tool_name,
                    "tool_status": "INTERCEPTED",
                    "tool_trace": tools.trace_logs
                }

            # 动态抽取患者姓名或业务单号 (先正则剥离虚词，再兜底常用示范姓名)
            keyword = ""
            m_rec = re.search(r'(REC\d+|ENC\d+)', msg, re.IGNORECASE)
            if m_rec:
                keyword = m_rec.group(1)

            if not keyword:
                m_pat = re.search(r'(?:患者|姓名|查阅|调阅)\s*([A-Za-z\u4e00-\u9fa5]{2,6})', msg)
                if m_pat:
                    candidate = m_pat.group(1)
                    candidate = re.split(r'(?:的|名下|近期|最近|既往|历史|病历|档案|记录|报告|检查|用药|情况)', candidate)[0].strip()
                    stop_words = {"病历", "档案", "记录", "报告", "医生", "检查", "用药", "历史", "就诊", "数据", "资料", "信息", "情况", "结果", "附件"}
                    if len(candidate) >= 2 and candidate not in stop_words:
                        keyword = candidate

            if not keyword:
                for name in ["张伟", "李雷", "韩梅梅", "王芳", "赵敏", "张三", "李四", "王五"]:
                    if name in msg:
                        keyword = name
                        break

            # 患者查阅自身档案时可不指定患者姓名，默认查阅自身
            if not keyword and user_role == "patient":
                keyword = ""
            elif not keyword:
                return {
                    "reply": (
                        "【提示】请在指令中明确需要检索的患者姓名或病历单号（例如：“查询患者张伟的病历记录” 或 “调阅 REC20250501001”），"
                        "以便系统准确定位档案，并在 Go 统一安全网关校验您的调阅权限与知情同意策略。"
                    ),
                    "tool_called": "NEED_PARAMETER",
                    "tool_status": "PROMPT_USER",
                    "tool_trace": tools.trace_logs
                }

            tool_called = f"query_patient_records(keyword='{keyword}')"
            tool_result = tools.query_patient_records(keyword=keyword)

            if tool_result.get("code") == 200:
                records = tool_result.get("data", [])
                if not records:
                    reply = f"为您查询了医疗记录库，未检索到与关键词【{keyword}】匹配的电子病历或检验报告。"
                else:
                    lines = [f"已通过统一安全网关为您检索到 {len(records)} 条受控健康档案：\n"]
                    for idx, r in enumerate(records, 1):
                        lines.append(
                            f"【记录 {idx}】流水号: {r.get('record_no')} | 患者: {r.get('patient_name', '患者')} | 类型: {r.get('data_type')}\n"
                            f"  - 诊断结论: {r.get('diagnosis')}\n"
                            f"  - 归属机构: {r.get('hospital_name', '医院')} | 经治医生: {r.get('doctor_name', '医生')}\n"
                            f"  - Fabric 账本凭据: {str(r.get('fabric_tx_id', ''))[:24]}... (高度 #{r.get('block_height')})\n"
                        )
                    lines.append("\n安全提示：以上数据均由 Go 统一安全网关执行 RBAC 角色校验与患者授权核验，密文从 IPFS 节点拉取并于内存动态解密。")
                    reply = "".join(lines)
            elif tool_result.get("code") == 403:
                reply = "【安全网关拦截 403 Forbidden】权限评估未通过：您未取得该患者的跨机构显式授权。若处于急诊休克或危重抢救场景，请在医生工作台发起 Break-Glass 紧急访问申请。"
            else:
                reply = f"查询失败: {tool_result.get('message', '未知错误')}"

        else:
            # 判断提问是否明显缺少业务关键词
            has_business_kw = any(w in msg_lower for w in ["病历", "档案", "就诊", "医生", "患者", "授权", "策略", "申请", "破窗", "核验", "区块链", "存证", "审计", "soap", "化验", "检查", "风险", "流水", "你好", "您好", "帮助", "功能", "说明"])
            if not has_business_kw and len(msg) > 1:
                reply = (
                    "【未识别到 MedTrust 业务指令】\n"
                    "抱歉，系统未能识别到与 MedTrust 医疗数据共享系统相关的操作指令。\n"
                    "本 AI 助手仅受控回答本软件内部的病历检索、知情授权、区块链存证及审计问题，不提供通用闲聊或非本系统业务咨询。\n"
                    "常用指令示例：\n"
                    "• 调阅病历：“查询患者张伟的病历记录”\n"
                    "• 授权管理：“查看我的授权策略列表”\n"
                    "• 存证核验：“核验病历1的数据真实完整性”\n"
                    "• 监管审计：“检索今日高风险调阅流水”"
                )
            else:
                reply = (
                    f"您好，{user_name}！我是 MedTrust 医疗数据受控 AI 智能助手。\n"
                    f"当前已识别您的会话角色为「{user_role or '匿名/受访者'}」。我可以协助您执行以下受控自然语言指令：\n\n"
                    "1. 医生查询：例如“查询患者张伟的历史病历记录”、“调阅张伟历史用药与既往病史”\n"
                    "2. 患者中心：例如“我现在授权了哪些医生或机构？”、“查看我的有效授权策略”、“谁访问过我的数据”\n"
                    "3. 完整性核验：例如“核验病历1的数据真实完整性与区块链存证哈希”\n"
                    "4. 安全监管：例如“今天有哪些高风险调阅事件？”、“查看最近的审计流水”\n\n"
                    "安全边界声明：本助手不直连任何底层数据库或 IPFS 节点，所有指令均携带当前用户会话 Token 经由 Go 统一安全网关进行鉴权与风险评分过滤。"
                )

        sanitized_reply = sanitize_agent_output(reply)

        status = "EXECUTED" if (tool_called and tool_called != "NEED_PARAMETER") else "SKIPPED"
        return {
            "reply": sanitized_reply,
            "tool_called": tool_called,
            "tool_status": status,
            "tool_trace": tools.trace_logs
        }
