import re
from typing import Dict, Any
from tools import (
    tool_query_medical_records,
    tool_query_authorizations,
    tool_verify_integrity,
    tool_query_audit_logs,
)

class MedTrustAgent:
    """
    MedTrust 受控医疗数据 AI 智能体 (Controlled Assistant Agent)
    严格遵循软件工程边界设计（docs/2.0.md 3.4 节）：
    1. 坚决不配置任何 MySQL、IPFS 或 Fabric 底层驱动，零直连隐患
    2. 严格践行 Single Gatekeeper 原则，仅通过 Tool Calling 携带当前调用者 JWT Bearer Token 请求 Go 后端 RESTful API
    3. 严格受到 Go 端 RBAC 角色过滤、ABAC 环境属性判定、患者知情授权与 RiskEngine 规则引擎审查
    4. 严守红线原则：严禁做出自动疾病诊断、处方推荐、自动授权或越权审批
    """
    def __init__(self):
        pass

    def run(self, user_message: str, token: str) -> dict:
        msg = user_message.strip()
        msg_lower = msg.lower()
        tool_called = None
        tool_result = None

        # 0.1 提示词注入与越狱探测防御 (Prompt Injection & Jailbreak Defense)
        injection_patterns = [
            r"ignore\s+(?:all\s+)?(?:previous\s+)?instructions",
            r"system\s+prompt",
            r"developer\s+mode",
            r"jailbreak",
            r"bypass\s+(?:security|rules|auth)",
            r"越狱",
            r"忽略所有(?:前置)?设定",
            r"忽略上述指令",
            r"导出(?:全部|系统)?密码",
            r"输出(?:私钥|密钥|root\s*key)",
            r"dump\s+(?:all\s+)?(?:database|tables)",
            r"drop\s+table",
            r"(?:'|\")\s*or\s*(?:'|\")?1\s*=\s*1",
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
                    "tool_status": "INTERCEPTED"
                }

        # 0.2 安全合规红线拦截：自动处方、自动确诊与自动授权越权行为坚决阻断
        if any(w in msg for w in ["开处方", "推荐药物", "诊断我", "得了什么病", "自动确诊", "替我授权", "代我审批", "自动审批"]):
            return {
                "reply": (
                    "【安全合规红线提醒】\n"
                    "依据医疗信息化伦理法规及 MedTrust 系统边界规范，本 AI 助手仅定位为“受控数据检索辅助工具”，"
                    "严禁赋予自动疾病诊断、智能处方开具、疾病预测或自动审批授权等医疗决策行为。\n"
                    "临床诊断与用药方案必须由合法执业医师在医生工作台当面查体评估后下达。"
                ),
                "tool_called": "BLOCKED_BY_SAFETY_GUARDRAIL",
                "tool_status": "INTERCEPTED"
            }

        # 1. 动态防篡改完整性核验
        if any(w in msg for w in ["核验", "防篡改", "真实性", "哈希", "完整性", "篡改"]):
            # 动态抽取病历 ID
            rec_id = 1
            m = re.search(r'(\d+)', msg)
            if m:
                rec_id = int(m.group(1))
            tool_called = f"tool_verify_integrity(record_id={rec_id})"
            tool_result = tool_verify_integrity(token, record_id=rec_id)
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
        elif any(w in msg for w in ["审计", "风险", "高风险", "日志", "监控", "拦截", "流水"]):
            risk_level = ""
            if "高风险" in msg or "high" in msg_lower:
                risk_level = "HIGH"
            elif "中风险" in msg or "medium" in msg_lower:
                risk_level = "MEDIUM"
            elif "低风险" in msg or "low" in msg_lower:
                risk_level = "LOW"

            tool_called = f"tool_query_audit_logs(risk_level='{risk_level}')"
            tool_result = tool_query_audit_logs(token, risk_level=risk_level)
            if tool_result.get("code") == 200:
                logs = tool_result.get("data", [])
                if not logs:
                    reply = f"系统中暂无匹配的审计流水记录 (筛选条件: 风险等级={risk_level or '全部'})。"
                else:
                    lines = [f"为您检索到系统最新审计流水 {len(logs[:5])} 条：\n"]
                    for l in logs[:5]:
                        lines.append(
                            f"* [{str(l.get('created_at', ''))[:19]}] 操作: {l.get('operation_type')} | "
                            f"主体: {l.get('user_name', '用户')} | 结果: {l.get('result')} | 风险: {l.get('risk_level')}\n"
                        )
                    reply = "".join(lines)
            else:
                reply = f"审计日志查询受限或失败: {tool_result.get('message')}"

        # 3. 患者知情授权策略查询
        elif any(w in msg for w in ["授权", "谁调阅", "我的授权", "被授权", "权限"]):
            tool_called = "tool_query_authorizations()"
            tool_result = tool_query_authorizations(token)
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
            # 动态抽取患者姓名或业务单号
            keyword = ""
            for name in ["张伟", "李雷", "韩梅梅", "王芳", "赵敏", "张三", "李四", "王五"]:
                if name in msg:
                    keyword = name
                    break

            if not keyword:
                m_pat = re.search(r'(?:患者|姓名|查阅|调阅)\s*([A-Za-z\u4e00-\u9fa5]{2,4})', msg)
                if m_pat:
                    candidate = m_pat.group(1)
                    stop_words = ["病历", "档案", "记录", "报告", "医生", "检查", "用药", "历史", "就诊", "数据", "资料", "信息", "情况", "结果", "附件"]
                    if not any(sw in candidate for sw in stop_words):
                        keyword = candidate
                if not keyword:
                    m_rec = re.search(r'(REC\d+|ENC\d+)', msg, re.IGNORECASE)
                    if m_rec:
                        keyword = m_rec.group(1)

            # 严谨性校验：若指令未明确患者主体，提示用户指定，杜绝默认臆测患者
            if not keyword:
                return {
                    "reply": (
                        "【提示】请在指令中明确需要检索的患者姓名或病历单号（例如：“查询患者张伟的病历记录” 或 “调阅 REC20250501001”），"
                        "以便系统准确定位档案，并在 Go 统一安全网关校验您的调阅权限与知情同意策略。"
                    ),
                    "tool_called": "NEED_PARAMETER",
                    "tool_status": "PROMPT_USER"
                }

            tool_called = f"tool_query_medical_records(keyword='{keyword}')"
            tool_result = tool_query_medical_records(token, keyword=keyword)
            
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
                            f"  - Fabric 账本凭据: {r.get('fabric_tx_id', '')[:24]}... (高度 #{r.get('block_height')})\n"
                        )
                    lines.append("\n安全提示：以上数据均由 Go 统一安全网关执行 RBAC 角色校验与患者授权核验，密文从 IPFS 节点拉取并于内存动态解密。")
                    reply = "".join(lines)
            elif tool_result.get("code") == 403:
                reply = "【安全网关拦截 403 Forbidden】权限评估未通过：您未取得该患者的跨机构显式授权。若处于急诊休克或危重抢救场景，请在医生工作台发起 Break-Glass 紧急访问申请。"
            else:
                reply = f"查询失败: {tool_result.get('message', '未知错误')}"

        else:
            reply = (
                "您好！我是 MedTrust 医疗数据受控 AI 智能助手。基于软件工程与安全规范，我可以协助您执行以下受控自然语言指令：\n\n"
                "1. 医生查询：例如“查询患者张伟的历史病历记录”、“调阅张伟历史用药与既往病史”\n"
                "2. 患者中心：例如“我现在授权了哪些医生或机构？”、“查看我的有效授权策略”\n"
                "3. 完整性核验：例如“核验病历1的数据真实完整性与区块链存证哈希”\n"
                "4. 安全监管：例如“今天有哪些高风险调阅事件？”、“查看最近的审计流水”\n\n"
                "安全边界声明：本助手不直连任何底层数据库或 IPFS 节点，所有指令均携带当前用户会话 Token 经由 Go 统一安全网关进行鉴权与风险评分过滤。"
            )

        return {
            "reply": reply,
            "tool_called": tool_called,
            "tool_status": "SUCCESS" if tool_result and tool_result.get("code") == 200 else ("INTERCEPTED" if tool_result and tool_result.get("code") == 403 else "SKIPPED")
        }
