# -*- coding: utf-8 -*-
"""WP-17 第 8 项回归测试：受控 AI Agent 边界与 JWT 密码学校验

验证 Phase 2 (WP-06) 修复：
  1. 伪造 Base64 JWT payload + 无效 HMAC-SHA256 签名 -> PermissionError (401)
  2. 篡改已正确签名的 token payload (换角色提权) -> 签名校验失败 -> 401
  3. 过期 exp -> PermissionError
  4. 非法 alg / 结构破坏 -> PermissionError
  5. 正确签名 token 正常解析 (对照组)
  6. 边界红线：伪造 supervisor 角色 token 无法绕过 HMAC 校验进入工具调用
"""
import base64
import hashlib
import hmac
import importlib
import json
import os
import unittest

import tools


def _b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).decode().rstrip("=")


def _make_signed_jwt(secret: str, claims: dict, header: dict = None) -> str:
    hdr = header or {"alg": "HS256", "typ": "JWT"}
    hdr_b64 = _b64url(json.dumps(hdr).encode())
    payload_b64 = _b64url(json.dumps(claims).encode())
    signing_input = f"{hdr_b64}.{payload_b64}".encode("ascii")
    sig = hmac.new(secret.encode("utf-8"), signing_input, hashlib.sha256).digest()
    return f"{hdr_b64}.{payload_b64}.{_b64url(sig)}"


def _make_raw_jwt(header_b64: str, payload_b64: str, sig_b64: str) -> str:
    return f"{header_b64}.{payload_b64}.{sig_b64}"


class JWTSecretConfigured:
    """在测试期间将 MEDTRUST_JWT_SECRET 注入 tools 模块 (校验严格模式)"""

    def __enter__(self):
        self._saved = tools.MEDTRUST_JWT_SECRET
        tools.MEDTRUST_JWT_SECRET = "wp17-regression-shared-secret-2026"
        return tools.MEDTRUST_JWT_SECRET

    def __exit__(self, *exc):
        tools.MEDTRUST_JWT_SECRET = self._saved
        return False


class TestPythonAgent_ForgedJWTSignatureRejected(unittest.TestCase):
    SECRET = "wp17-regression-shared-secret-2026"

    def setUp(self):
        self.ctx = JWTSecretConfigured()
        self.secret = self.ctx.__enter__()
        self.addCleanup(self.ctx.__exit__, None, None, None)

    # 1. 完全伪造 token (签名乱写) 必须被拒绝
    def test_forged_signature_rejected(self):
        header_b64 = _b64url(json.dumps({"alg": "HS256"}).encode())
        payload_b64 = _b64url(json.dumps({"role": "supervisor", "user_id": 1}).encode())
        forged = _make_raw_jwt(header_b64, payload_b64, "Zm9yZ2VkX3NpZ25hdHVyZQ")
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(forged)

    # 2. 篡改已正确签名 token 的 payload (提权伪造 role) -> 签名失配被拒
    def test_tampered_payload_escalation_rejected(self):
        token = _make_signed_jwt(self.secret, {"role": "patient", "user_id": 101})
        header_b64, _, sig_b64 = token.split(".")
        # 攻击者替换 payload 为 supervisor 提权，但保留原签名
        escalated_payload = _b64url(json.dumps({"role": "supervisor", "user_id": 1}).encode())
        tampered = _make_raw_jwt(header_b64, escalated_payload, sig_b64)
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(tampered)

    # 3. 用错误密钥签名的 token 被拒
    def test_wrong_signing_key_rejected(self):
        token = _make_signed_jwt("attacker-owned-secret", {"role": "supervisor", "user_id": 1})
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(token)

    # 4. 过期 exp 被拒
    def test_expired_jwt_rejected(self):
        token = _make_signed_jwt(self.secret, {"role": "doctor", "exp": 1000000000})
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(token)

    # 5. 非法签名算法 (alg=none / RS256 降级) 被拒
    def test_invalid_alg_rejected(self):
        token = _make_signed_jwt(self.secret, {"role": "supervisor"}, header={"alg": "none"})
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(token)

    # 6. 结构破坏 (少于三段) 被拒
    def test_malformed_structure_rejected(self):
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims("only.two")

    # 7. 对照组：正确签名 token 正常解析出 claims
    def test_correctly_signed_jwt_accepted(self):
        token = _make_signed_jwt(self.secret, {"role": "doctor", "user_id": 42})
        claims = tools.extract_jwt_claims(token)
        self.assertEqual(claims.get("role"), "doctor")
        self.assertEqual(claims.get("user_id"), 42)

    # 8. 端到端：伪造 supervisor token 无法进入工具调用，统一 401 拒绝
    def test_agent_rejects_forged_token_end_to_end(self):
        import agent as agent_module
        importlib.reload(agent_module)  # 确保引用同一 tools 模块状态

        header_b64 = _b64url(json.dumps({"alg": "HS256"}).encode())
        payload_b64 = _b64url(json.dumps({"role": "supervisor", "username": "attacker", "user_id": 1}).encode())
        forged = _make_raw_jwt(header_b64, payload_b64, "aW52YWxpZF9zaWduYXR1cmU")

        ag = agent_module.MedTrustAgent()
        result = ag.run("查询高风险安全告警", forged)
        self.assertEqual(result["tool_status"], "AUTH_REJECTED")
        self.assertIn("401", result["reply"])
        # 关键断言：鉴权失败后不执行任何工具
        self.assertTrue(len(result["tool_trace"]) == 0)


class TestWP06_SoapRoleIsolationAndSecretFallback(unittest.TestCase):
    """Patch-02 (WP-06) 回归：SOAP 工具角色隔离 + JWT 密钥三级兜底"""

    def setUp(self):
        # 还原 tools 模块的原始密钥（防止其他测试注入的密钥影响本组）
        self._saved_secret = tools.MEDTRUST_JWT_SECRET
        self._saved_env = os.environ.pop("MEDTRUST_JWT_SECRET", None), os.environ.pop("JWT_SECRET", None)
        importlib.reload(tools)
        self.addCleanup(self._restore)

    def _restore(self):
        tools.MEDTRUST_JWT_SECRET = self._saved_secret
        for key, val in zip(("MEDTRUST_JWT_SECRET", "JWT_SECRET"), self._saved_env):
            if val is not None:
                os.environ[key] = val

    # 9. WP-06: supervisor 无权调用 SOAP 临床聚合工具，doctor 有权
    def test_wp06_supervisor_cannot_call_summarize_soap(self):
        self.assertFalse(tools.is_tool_allowed_for_role("tool_summarize_soap", "supervisor"))
        self.assertTrue(tools.is_tool_allowed_for_role("tool_summarize_soap", "doctor"))
        # supervisor 的核验工具权限不受影响
        self.assertTrue(tools.is_tool_allowed_for_role("tool_verify_integrity", "supervisor"))

    # 10. 未配置共享密钥时必须 fail-closed：服务端拒绝一切令牌，而不是"跳过验签仅解析 payload"
    def test_wp06_missing_secret_fails_closed(self):
        self.assertTrue(
            tools.MEDTRUST_JWT_SECRET,
            "仓库内 .env/config.yaml 应提供 JWT 密钥，否则本机联调会全部 401",
        )

        header_b64 = _b64url(json.dumps({"alg": "HS256"}).encode())
        payload_b64 = _b64url(json.dumps({"role": "supervisor", "user_id": 1}).encode())
        forged = _make_raw_jwt(header_b64, payload_b64, "Zm9yZ2VkX3NpZ25hdHVyZQ")
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(forged)

        # 空密钥场景：连"签名恰好合法"的令牌也必须被拒绝 (不得退化为仅解析)
        tools.MEDTRUST_JWT_SECRET = ""
        with self.assertRaises(tools.PermissionError):
            tools.extract_jwt_claims(forged)


if __name__ == "__main__":
    unittest.main()