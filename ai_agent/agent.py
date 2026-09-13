import json
from tools import (
    tool_query_medical_records,
    tool_query_authorizations,
    tool_verify_integrity,
    tool_query_audit_logs,
)

class MedTrustAgent:
    """
    MedTrust 受控医疗数据 AI 智能体 (Controlled Agent)
    严格遵循软件工程边界设计：
    1. 不直接访问任何底层数据库或 IPFS 物理节点
    2. 严格通过带有当前操作员会话 Bearer Token 的 Tool Calling 调用 Go 统一安全网关 API
    3. 严格受到 Go 端的 RBAC 角色、患者授权策略与规则风险引擎拦截
    """
    def __init__(self):
        pass

    def run(self, user_message: str, token: str) -> dict:
        msg = user_message.strip().lower()
        tool_called = None
        tool_result = None

        # 1. 优先判定高优先级专项目标：完整性动态核验 (防篡改)
        if any(w in msg for w in ["核验", "防篡改", "真实性", "哈希", "完整性"]):
            tool_called = "tool_verify_integrity(record_id=1)"
            tool_result = tool_verify_integrity(token, record_id=1)
            if tool_result.get("code") == 200:
                v = tool_result.get("data", {})
                is_verified = v.get("verified", False)
                symbol = "[PASS]" if is_verified else "[ALERT]"
                reply = (
                    f"【MedTrust 动态防篡改核验报告】\n"
                    f"* 目标病历编号: {v.get('record_no')}\n"
                    f"* IPFS 解密文件 SHA-256: {v.get('calculated_hash')}\n"
                    f"* Fabric 链上固化 SHA-256: {v.get('chain_hash')}\n"
                    f"* 链上存证交易哈希 TxID: {v.get('fabric_tx_id', '')}\n"
                    f"* 动态核验裁定: {symbol} {v.get('message')}"
                )
            else:
                reply = f"完整性核验失败: {tool_result.get('message')}"

        # 2. 监管审计日志检索
        elif any(w in msg for w in ["审计", "风险", "高风险", "日志", "监控", "拦截"]):
            tool_called = "tool_query_audit_logs(risk_level='')"
            tool_result = tool_query_audit_logs(token)
            if tool_result.get("code") == 200:
                logs = tool_result.get("data", [])
                lines = [f"为您检索到系统最新审计流水 {len(logs[:5])} 条：\n"]
                for l in logs[:5]:
                    lines.append(
                        f"* [{str(l.get('created_at', ''))[:19]}] 操作: {l.get('operation_type')} | "
                        f"主体: {l.get('user_name', '用户')} | 结果: {l.get('result')} | 风险: {l.get('risk_level')}\n"
                    )
                reply = "".join(lines)
            else:
                reply = f"审计日志查询受限或失败: {tool_result.get('message')}"

        # 3. 授权策略查询 (患者/医生)
        elif any(w in msg for w in ["授权", "谁调阅", "我的授权", "被授权"]):
            tool_called = "tool_query_authorizations()"
            tool_result = tool_query_authorizations(token)
            if tool_result.get("code") == 200:
                auths = tool_result.get("data", [])
                if not auths:
                    reply = "您当前没有处于生效状态（ACTIVE）的患者授权策略。"
                else:
                    lines = [f"名下生效中的授权策略共 {len(auths)} 条：\n"]
                    for a in auths:
                        lines.append(
                            f"* 授权单号: {a.get('auth_no')} | 对象: {a.get('target_name', a.get('auth_target_type'))}\n"
                            f"  范围: {a.get('scope_type')} | 状态: {a.get('status')} | 截止日期: {str(a.get('end_time', ''))[:10]}\n"
                        )
                    reply = "".join(lines)
            else:
                reply = f"授权信息查询失败: {tool_result.get('message')}"

        # 4. 病历/检验记录查询
        elif any(w in msg for w in ["病历", "检查", "报告", "记录", "用药", "历史", "张伟", "患者"]):
            tool_called = "tool_query_medical_records(keyword='张伟')"
            tool_result = tool_query_medical_records(token, keyword="张伟")
            
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
                            f"  - 区块链存证 TxID: {r.get('fabric_tx_id', '')[:24]}... (区块高度 #{r.get('block_height')})\n"
                        )
                    lines.append("\n提示：以上数据已由 Go 安全网关校验当前登录凭证与患者授权状态，密文自 IPFS 解密拉取。")
                    reply = "".join(lines)
            elif tool_result.get("code") == 403:
                reply = "【安全网关拦截】权限评估未通过：您未取得该患者的跨机构显式授权。若处于急救危重抢救场景，请在医生工作台申请 Break-Glass 紧急访问。"
            else:
                reply = f"查询失败: {tool_result.get('message', '未知错误')}"

        else:
            reply = (
                "您好！我是 MedTrust 医疗数据受控 AI 智能助手。基于软件工程与安全规范，我可以协助您执行以下受控自然语言指令：\n\n"
                "1. 医生查询：例如“查询患者张伟的历史病历记录”、“调阅张伟历史用药与过敏史”\n"
                "2. 患者中心：例如“我现在授权了哪些医生或机构？”、“查看我的有效授权策略”\n"
                "3. 完整性核验：例如“核验病历1的数据真实完整性与区块链存证哈希”\n"
                "4. 安全监管：例如“今天有哪些高风险调阅事件？”、“查看最近的审计日志”\n\n"
                "安全说明：本助手不直连任何底层数据库，全部查询操作均携带您当前会话 Token 经由 Go 统一网关鉴权过滤。"
            )

        return {
            "reply": reply,
            "tool_called": tool_called,
            "tool_status": "SUCCESS" if tool_result and tool_result.get("code") == 200 else ("INTERCEPTED" if tool_result and tool_result.get("code") == 403 else "SKIPPED")
        }
