# -*- coding: utf-8 -*-
"""V-08 回归测试：AI Agent 实时会话网关核验（阻断已注销/封禁 JWT）与对话限流"""
import base64
import hashlib
import hmac
import json
import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient

import main
import tools


def _make_jwt(secret: str, role: str = "doctor", user_id: int = 101) -> str:
    hdr = base64.urlsafe_b64encode(json.dumps({"alg": "HS256", "typ": "JWT"}).encode()).decode().rstrip("=")
    pay = base64.urlsafe_b64encode(json.dumps({"role": role, "username": "doc_test", "user_id": user_id}).encode()).decode().rstrip("=")
    signing_input = f"{hdr}.{pay}".encode("ascii")
    sig = hmac.new(secret.encode("utf-8"), signing_input, hashlib.sha256).digest()
    sig_b64 = base64.urlsafe_b64encode(sig).decode().rstrip("=")
    return f"{hdr}.{pay}.{sig_b64}"


class TestV08AIAgentSessionAndRateLimit(unittest.TestCase):
    def setUp(self):
        self._saved_secret = tools.MEDTRUST_JWT_SECRET
        tools.MEDTRUST_JWT_SECRET = "v08-test-shared-jwt-secret-2026-strong"
        self.token = _make_jwt(tools.MEDTRUST_JWT_SECRET, role="doctor", user_id=202)
        self.client = TestClient(main.app)
        if hasattr(main, "_rate_limit_buckets"):
            main._rate_limit_buckets.clear()

    def tearDown(self):
        tools.MEDTRUST_JWT_SECRET = self._saved_secret

    def test_blacklisted_token_rejected_by_gateway_check(self):
        """已登出加入黑名单的 JWT 调用 /api/v1/ai/chat 必须返回 401，禁止继续执行或消耗外部 LLM"""
        self.assertTrue(
            hasattr(tools, "verify_session_with_gateway"),
            "【V-08 漏洞】tools 模块缺少 verify_session_with_gateway 实时会话核验函数",
        )
        with patch.object(main, "verify_session_with_gateway", return_value=(False, 401, "身份令牌已登出失效")):
            resp = self.client.post(
                "/api/v1/ai/chat",
                json={"message": "查询患者张三病历"},
                headers={"Authorization": f"Bearer {self.token}"},
            )
            self.assertEqual(resp.status_code, 401)

    def test_disabled_account_rejected_with_403(self):
        """被封禁账号调用 /api/v1/ai/chat 必须返回 403"""
        self.assertTrue(
            hasattr(tools, "verify_session_with_gateway"),
            "【V-08 漏洞】tools 模块缺少 verify_session_with_gateway 实时会话核验函数",
        )
        with patch.object(main, "verify_session_with_gateway", return_value=(False, 403, "账号已被封禁")):
            resp = self.client.post(
                "/api/v1/ai/chat",
                json={"message": "查询患者张三病历"},
                headers={"Authorization": f"Bearer {self.token}"},
            )
            self.assertEqual(resp.status_code, 403)

    def test_chat_endpoint_rate_limited(self):
        """高频突发调用 /api/v1/ai/chat 必须触发 429 限流保护"""
        self.assertTrue(
            hasattr(main, "AI_CHAT_RATE_LIMIT"),
            "【V-08 漏洞】ai_agent/main.py 缺少对话接口限流保护",
        )
        with patch.object(main, "verify_session_with_gateway", return_value=(True, 200, "")), \
             patch.object(main, "AI_CHAT_RATE_LIMIT", 3), \
             patch.object(main.agent, "run", return_value={"reply": "ok", "tool_status": "SKIPPED", "tool_trace": []}):
            codes = []
            for _ in range(4):
                r = self.client.post(
                    "/api/v1/ai/chat",
                    json={"message": "查询患者病历"},
                    headers={"Authorization": f"Bearer {self.token}"},
                )
                codes.append(r.status_code)
            self.assertEqual(codes[:3], [200, 200, 200])
            self.assertEqual(codes[3], 429)


if __name__ == "__main__":
    unittest.main()
