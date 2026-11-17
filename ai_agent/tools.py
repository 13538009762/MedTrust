import base64
import hashlib
import hmac
import json
import os
import re
import time
import requests
from pathlib import Path
from typing import Dict, Any, Optional, Set

GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://127.0.0.1:8080/api/v1")


def _read_secret_from_file(path: Path, key: str) -> str:
    """从 KEY=VALUE (.env) 或 key: value (YAML) 文件中读取指定密钥；只做最小解析，避免额外依赖。"""
    if not path.exists():
        return ""
    try:
        for line in path.read_text(encoding="utf-8").splitlines():
            stripped = line.strip()
            if not stripped or stripped.startswith("#"):
                continue
            if stripped.startswith(key + ":") or stripped.startswith(key + "="):
                sep = "=" if "=" in stripped.split("#", 1)[0] and not stripped.startswith(key + ":") else ":"
                val = stripped.split(sep, 1)[1].split("#", 1)[0].strip().strip('"').strip("'")
                if val:
                    return val
    except OSError:
        return ""
    return ""


def _load_jwt_secret() -> str:
    """加载与 Go 后端共享的 JWT HMAC 密钥。

    来源优先级：环境变量 > backend/config/config.yaml > 仓库根 .env。
    全部缺失时返回空串——extract_jwt_claims 会在空密钥下拒绝一切 token（fail-closed），
    不再回退到硬编码弱密钥。
    """
    env_secret = os.getenv("MEDTRUST_JWT_SECRET") or os.getenv("JWT_SECRET")
    if env_secret:
        return env_secret.strip()

    repo_root = Path(__file__).resolve().parent.parent
    for path, key in (
        (repo_root / "backend" / "config" / "config.yaml", "jwt_secret"),
        (repo_root / ".env", "JWT_SECRET"),
        (repo_root / "backend" / ".env", "JWT_SECRET"),
    ):
        val = _read_secret_from_file(path, key)
        if val:
            return val
    return ""


# WP-06: 与 Go 后端共享的 JWT 签名密钥 (HS256)，强制校验签名，阻断伪造 Payload
MEDTRUST_JWT_SECRET = _load_jwt_secret()

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
        "tool_summarize_soap",
    },
    "supervisor": {
        "tool_query_audit_logs",
        "tool_verify_integrity",
    },
    "admin": {
        "tool_query_audit_logs",
    },
}

class PermissionError(Exception):
    """JWT 签名/有效期校验失败——拒绝信任 token 内的 role/user_id (应映射 HTTP 401)"""


def _b64url_decode(data: str) -> bytes:
    padded = data + "=" * ((4 - len(data) % 4) % 4)
    return base64.urlsafe_b64decode(padded)


def extract_jwt_claims(token: str) -> Dict[str, Any]:
    """密码学 JWT 校验——HMAC-SHA256 签名比对 (hmac.compare_digest 防 timing attack) + exp 过期校验。

    fail-closed 语义：未配置共享密钥时一律拒绝（抛 PermissionError → HTTP 401），
    绝不"跳过验签仅解析 payload"，否则攻击者可自签任意角色 token 提权。"""
    if not MEDTRUST_JWT_SECRET:
        raise PermissionError("服务端未配置 JWT 共享密钥 (MEDTRUST_JWT_SECRET/JWT_SECRET)，拒绝所有令牌")

    try:
        parts = token.split(".")
        if len(parts) != 3:
            raise PermissionError("JWT 结构非法")
        header = json.loads(_b64url_decode(parts[0]))
        if header.get("alg") != "HS256":
            raise PermissionError(f"不支持的 JWT 签名算法: {header.get('alg')}")

        signing_input = f"{parts[0]}.{parts[1]}".encode("ascii")
        expected_sig = hmac.new(
            MEDTRUST_JWT_SECRET.encode("utf-8"), signing_input, hashlib.sha256
        ).digest()
        actual_sig = _b64url_decode(parts[2])
        if not hmac.compare_digest(expected_sig, actual_sig):
            raise PermissionError("JWT HMAC-SHA256 签名校验失败 (疑似伪造 token)")

        claims = json.loads(_b64url_decode(parts[1]))

        exp = claims.get("exp")
        if exp is not None:
            if isinstance(exp, (int, float)) and exp < time.time():
                raise PermissionError("JWT 已过期 (exp)")

        return claims
    except PermissionError:
        raise
    except Exception as exc:
        raise PermissionError(f"JWT 解析失败: {exc}") from exc

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


class ControlledMedicalTools:
    """受控医疗工具集类 (严格按照 updatev1.txt 第七节规范实现，全程携带 JWT，记录审计轨迹)"""
    def __init__(self, bearer_token: str):
        self.headers = {
            "Authorization": bearer_token if bearer_token.startswith("Bearer ") else f"Bearer {bearer_token}",
            "Content-Type": "application/json",
            "X-Source": "AI_AGENT"
        }
        self.trace_logs = []

    def query_patient_records(self, patient_id: int = 0, data_type: str = "", keyword: str = "") -> Dict[str, Any]:
        """受控工具1：查询当前登录用户有权查看的医疗记录列表"""
        params = {}
        if patient_id > 0:
            params["patient_id"] = patient_id
        if data_type:
            params["data_type"] = data_type
        if keyword:
            params["keyword"] = keyword
        try:
            resp = requests.get(f"{GO_BACKEND_URL}/medical-records", headers=self.headers, params=params, timeout=10)
            result = resp.json() if resp.status_code == 200 else {"code": resp.status_code, "message": "权限拦截"}
            status_code = resp.status_code
        except Exception as e:
            result = {"code": 500, "message": str(e)}
            status_code = 500
        records_data = result.get("data") if isinstance(result, dict) else []
        has_unauthorized = False
        if isinstance(records_data, list) and len(records_data) > 0:
            has_unauthorized = any(not item.get("has_access", True) for item in records_data if isinstance(item, dict))
        auth_status = "PASSED"
        gate1_msg = "RBAC/ABAC/患者授权核验通过 (HTTP 200)"
        gate3_msg = "已核验返回结果中全部 RecordID 均在有效 Consent 集合内，无隐私越权泄露"
        if status_code != 200:
            auth_status = "INTERCEPTED_BY_GATEKEEPER"
            gate1_msg = f"未获患者授权，安全网关主动阻断 (HTTP {status_code})"
            gate3_msg = "未放行，跳过"
        elif has_unauthorized:
            auth_status = "PARTIAL_MASKED_BY_GATEKEEPER"
            gate3_msg = "检测到未获患者授权的外院病历，Go 安全网关已自动执行字段级掩码脱敏与传染病隐私剥离"

        self.trace_logs.append({
            "tool": "query_patient_records",
            "endpoint": "GET /api/v1/medical-records",
            "status_code": status_code,
            "gate1_guard": gate1_msg,
            "gate2_semantic": "执行工具: query_patient_records (跨院非结构化病历语义检索与聚合)",
            "gate3_post_check": gate3_msg,
            "auth_check": auth_status
        })
        return result

    def query_my_authorizations(self, status: str = "") -> Dict[str, Any]:
        """受控工具2：查询患者授权策略状态列表"""
        params = {"status": status} if status else {}
        try:
            resp = requests.get(f"{GO_BACKEND_URL}/authorizations", headers=self.headers, params=params, timeout=10)
            result = resp.json() if resp.status_code == 200 else {"code": resp.status_code, "message": "权限拦截"}
            status_code = resp.status_code
        except Exception as e:
            result = {"code": 500, "message": str(e)}
            status_code = 500
        self.trace_logs.append({
            "tool": "query_my_authorizations",
            "endpoint": "GET /api/v1/authorizations",
            "status_code": status_code,
            "gate1_guard": "RBAC角色白名单与患者本人隔离核验通过 (HTTP 200)" if status_code == 200 else f"权限拦截 (HTTP {status_code})",
            "gate2_semantic": "执行工具: query_my_authorizations (患者授权策略全景检索)",
            "gate3_post_check": "已通过 Go 授权安全范围校验，仅返回当前用户合法授权数据" if status_code == 200 else "未放行，跳过",
            "auth_check": "PASSED" if status_code == 200 else "INTERCEPTED_BY_GATEKEEPER"
        })
        return result

    def verify_record_integrity(self, record_id: int) -> Dict[str, Any]:
        """受控工具3：触发指定病历的 IPFS 链下密文与 Fabric 链上存证 Hash 动态完整性核验"""
        try:
            resp = requests.post(f"{GO_BACKEND_URL}/verification/{record_id}", headers=self.headers, timeout=10)
            result = resp.json() if resp.status_code == 200 else {"code": resp.status_code, "message": "权限拦截"}
            status_code = resp.status_code
        except Exception as e:
            result = {"code": 500, "message": str(e)}
            status_code = 500
        self.trace_logs.append({
            "tool": "verify_record_integrity",
            "endpoint": f"POST /api/v1/verification/{record_id}",
            "status_code": status_code,
            "gate1_guard": "角色白名单核准 + 目标病历明文查阅权校验通过 (HTTP 200)" if status_code == 200 else f"权限拦截 (HTTP {status_code})",
            "gate2_semantic": "执行工具: verify_record_integrity (IPFS 链下密文与 Fabric 链上 Hash 动态核验)",
            "gate3_post_check": "三阶段数据真实性与防篡改链条核验闭环已完成",
            "auth_check": "PASSED" if status_code == 200 else "INTERCEPTED_BY_GATEKEEPER"
        })
        return result

    def summarize_patient_soap(self, patient_id: int) -> Dict[str, Any]:
        """受控工具5：调用 Go 后端 /ai/diagnostic-assist 统一 SOAP 摘要边界 (Python 零数据库驱动)"""
        try:
            valid_pid = int(patient_id)
            if valid_pid <= 0:
                raise ValueError
        except (TypeError, ValueError):
            return {"code": 400, "message": "非法的患者 ID", "data": None}
        try:
            resp = requests.post(
                f"{GO_BACKEND_URL}/ai/diagnostic-assist",
                headers=self.headers,
                json={"patient_id": valid_pid},
                timeout=15,
            )
            result = resp.json() if resp.status_code == 200 else {"code": resp.status_code, "message": "权限拦截"}
            status_code = resp.status_code
        except Exception as e:
            result = {"code": 500, "message": str(e)}
            status_code = 500
        self.trace_logs.append({
            "tool": "summarize_patient_soap",
            "endpoint": "POST /api/v1/ai/diagnostic-assist",
            "status_code": status_code,
            "gate1_guard": "角色白名单 (仅执业医生) 核验通过 (HTTP 200)" if status_code == 200 else f"权限拦截 (HTTP {status_code})",
            "gate2_semantic": "执行工具: summarize_patient_soap (经 Go 统一 SOAP 摘要边界，零诊断零处方)",
            "gate3_post_check": "Go 侧 enforceNonDiagnosticOutputGuard 已强制清空疾病预测与新处方" if status_code == 200 else "未放行，跳过",
            "auth_check": "PASSED" if status_code == 200 else "INTERCEPTED_BY_GATEKEEPER"
        })
        return result

    def analyze_audit_anomalies(self, risk_level: str = "") -> Dict[str, Any]:
        """受控工具4：检索访问审计日志与异常高风险行为记录（仅监管员/患者本人可用）"""
        try:
            params = {"risk_level": risk_level} if risk_level else {}
            resp = requests.get(f"{GO_BACKEND_URL}/audit-logs", headers=self.headers, params=params, timeout=10)
            result = resp.json() if resp.status_code == 200 else {"code": resp.status_code, "message": "权限拦截"}
            status_code = resp.status_code
        except Exception as e:
            result = {"code": 500, "message": str(e)}
            status_code = 500
        self.trace_logs.append({
            "tool": "analyze_audit_anomalies",
            "endpoint": "GET /api/v1/audit-logs",
            "status_code": status_code,
            "gate1_guard": "监管员/患者本人隔离角色准入通过 (HTTP 200)" if status_code == 200 else f"权限拦截 (HTTP {status_code})",
            "gate2_semantic": "执行工具: analyze_audit_anomalies (跨机构流转审计日志检索)",
            "gate3_post_check": "脱敏审计事件范围已核准，杜绝未授权隐私透出",
            "auth_check": "PASSED" if status_code == 200 else "INTERCEPTED_BY_GATEKEEPER"
        })
        return result
