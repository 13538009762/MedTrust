import requests
from typing import Dict, Any, Optional

GO_BACKEND_URL = "http://127.0.0.1:8080/api/v1"

def tool_query_medical_records(token: str, keyword: str = "", patient_id: Optional[int] = None) -> Dict[str, Any]:
    """通过携带用户 Token 调用 Go 后端业务接口查询病历"""
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
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
    """通过携带用户 Token 触发动态防篡改核验"""
    headers = {"Authorization": f"Bearer {token}", "X-Source": "AI_AGENT"}
    try:
        resp = requests.post(f"{GO_BACKEND_URL}/verification/{record_id}", headers=headers, timeout=5)
        return resp.json()
    except Exception as e:
        return {"code": 500, "message": f"调用后端核验接口失败: {str(e)}", "data": None}
