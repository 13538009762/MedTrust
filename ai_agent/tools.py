import base64
import json
import re
import requests
from typing import Dict, Any, Optional, Set

GO_BACKEND_URL = "http://127.0.0.1:8080/api/v1"

# 严格角色-工具白名单映射 (RBAC + Tool Calling Guardrail)
ROLE_TOOL_ALLOWLIST: Dict[str, Set[str]] = {
    "patient": {
        "tool_query_medical_records",
        "tool_query_authorizations",
        "tool_query_audit_logs",
    },
    "doctor": {
        "tool_query_medical_records",
        "tool_verify_integrity",
    },
    "supervisor": {
        "tool_query_audit_logs",
        "tool_verify_integrity",
    },
    "admin": {
        "tool_query_audit_logs",
    },
}

def extract_jwt_claims(token: str) -> Dict[str, Any]:
    """无需外部依赖纯 Python 解析 JWT Claims Payload"""
    try:
        parts = token.split(".")
        if len(parts) >= 2:
            payload = parts[1]
            padded = payload + "=" * ((4 - len(payload) % 4) % 4)
            data = base64.urlsafe_b64decode(padded)
            return json.loads(data)
    except Exception:
        pass
    return {}

def is_tool_allowed_for_role(tool_name: str, role: str) -> bool:
    """检查工具是否在指定角色的白名单内"""
    if not role:
        return False
    allowed = ROLE_TOOL_ALLOWLIST.get(role.lower(), set())
    return tool_name in allowed

def validate_keyword(keyword: str) -> str:
    """清理并校验查询关键字，防范特殊控制字符注入"""
    if not keyword:
        return ""
    # 限制长度并在白名单内过滤危险字符
    clean = re.sub(r'[\r\n\t;\'"\\]', '', keyword.strip())
    return clean[:100]

def validate_record_id(record_id: Any) -> int:
    """校验病历 ID 是否为合法正整数"""
    try:
        val = int(record_id)
        if val <= 0:
            raise ValueError("ID 必须为大于 0 的正整数")
        return val
    except Exception as e:
        raise ValueError(f"非法的病历 ID: {record_id}") from e

def validate_risk_level(risk_level: str) -> str:
    """校验风险等级参数"""
    clean = risk_level.strip().upper()
    if clean in ("LOW", "MEDIUM", "HIGH"):
        return clean
    return ""

def tool_query_medical_records(token: str, keyword: str = "", patient_id: Optional[int] = None) -> Dict[str, Any]:
    """通过携带用户 Token 调用 Go 后端业务接口查询病历"""
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
    params = {}
    clean_kw = validate_keyword(keyword)
    if clean_kw:
        params["keyword"] = clean_kw
    if patient_id and patient_id > 0:
        params["patient_id"] = patient_id
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/medical-records", headers=headers, params=params, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端病历接口失败: {str(e)}", "data": None}

def tool_query_authorizations(token: str) -> Dict[str, Any]:
    """通过携带用户 Token 查询患者授权策略列表"""
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/authorizations", headers=headers, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端授权接口失败: {str(e)}", "data": None}

def tool_query_audit_logs(token: str, risk_level: str = "", operation_type: str = "") -> Dict[str, Any]:
    """通过携带用户 Token 查询审计日志"""
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
    params = {}
    valid_risk = validate_risk_level(risk_level)
    if valid_risk:
        params["risk_level"] = valid_risk
    if operation_type:
        clean_op = re.sub(r'[^A-Za-z0-9_]', '', operation_type.strip().upper())
        if clean_op:
            params["operation_type"] = clean_op
    try:
        resp = requests.get(f"{GO_BACKEND_URL}/audit-logs", headers=headers, params=params, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端审计接口失败: {str(e)}", "data": None}

def tool_verify_integrity(token: str, record_id: int) -> Dict[str, Any]:
    """通过携带用户 Token 触发动态防篡改核验"""
    try:
        valid_id = validate_record_id(record_id)
    except ValueError as err:
        return {"code": 400, "message": str(err), "data": None}
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
    try:
        resp = requests.post(f"{GO_BACKEND_URL}/verification/{valid_id}", headers=headers, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端核验接口失败: {str(e)}", "data": None}
