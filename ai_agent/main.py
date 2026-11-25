import asyncio
import os
import threading
import time
from typing import Any, Dict, List

import uvicorn
from fastapi import FastAPI, Header, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, ConfigDict, Field
from agent import MedTrustAgent
from tools import (
    PermissionError as JWTAuthError,
    extract_jwt_claims,
    verify_session_with_gateway,
)

app = FastAPI(
    title="MedTrust Controlled AI Agent Service",
    description="Python FastAPI AI Agent with Tool Calling strictly authenticated via Go Backend Gateway",
    version="2.0.0",
)

# WP-04: 收紧 CORS——禁止 allow_origins=["*"] 与 allow_credentials=True 组合
_default_origins = ["http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:8080"]
_origins_raw = os.environ.get("MEDTRUST_CORS_ORIGINS", "")
allow_origins = [o.strip() for o in _origins_raw.split(",") if o.strip() and o.strip() != "*"] if _origins_raw else _default_origins
if not allow_origins:
    allow_origins = _default_origins

app.add_middleware(
    CORSMiddleware,
    allow_origins=allow_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["Authorization", "Content-Type"],
)

agent = MedTrustAgent()

# V-08: 滑动窗口限流保护，防止已登录/异常令牌高频刷取外部付费 LLM 接口
AI_CHAT_RATE_LIMIT = int(os.environ.get("AI_CHAT_RATE_LIMIT", "30"))
AI_CHAT_RATE_WINDOW = int(os.environ.get("AI_CHAT_RATE_WINDOW", "60"))
_rate_limit_buckets: Dict[str, List[float]] = {}
_rate_limit_lock = threading.Lock()


def _check_rate_limit(subject_key: str) -> bool:
    now = time.time()
    cutoff = now - AI_CHAT_RATE_WINDOW
    with _rate_limit_lock:
        timestamps = [ts for ts in _rate_limit_buckets.get(subject_key, []) if ts > cutoff]
        if len(timestamps) >= AI_CHAT_RATE_LIMIT:
            _rate_limit_buckets[subject_key] = timestamps
            return False
        timestamps.append(now)
        _rate_limit_buckets[subject_key] = timestamps
        if len(_rate_limit_buckets) > 4096:
            stale_keys = [k for k, v in _rate_limit_buckets.items() if not v or v[-1] <= cutoff]
            for k in stale_keys:
                _rate_limit_buckets.pop(k, None)
        return True


class ChatRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="ignore")
    message: str = Field(..., min_length=1, max_length=2000)


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok", "service": "MedTrust AI Agent"}


@app.post("/api/v1/ai/chat")
@app.post("/api/ai/chat")
async def chat(req: ChatRequest, request: Request, authorization: str | None = Header(None)) -> Dict[str, Any]:
    token = ""
    if authorization:
        auth_str = authorization.strip()
        if auth_str.lower().startswith("bearer "):
            token = auth_str[7:].strip()
        else:
            token = auth_str

    if not token:
        raise HTTPException(status_code=401, detail="Missing or invalid Bearer authorization token")

    # 1. 本地密码学验签 + 过期校验 (快速拦截伪造令牌)
    try:
        claims = extract_jwt_claims(token)
    except JWTAuthError as exc:
        raise HTTPException(status_code=401, detail=str(exc))

    # 2. 按用户维度执行滑动窗口限流保护
    client_host = request.client.host if request.client else "unknown"
    rate_key = f"uid:{claims.get('user_id', client_host)}"
    if not _check_rate_limit(rate_key):
        raise HTTPException(status_code=429, detail="AI 对话请求过于频繁，请稍后再试")

    # 3. 向 Go 统一安全网关实时核验会话有效性（拦截已登出黑名单、改密撤销与封禁账号）
    ok, status_code, err_msg = await asyncio.to_thread(verify_session_with_gateway, token)
    if not ok:
        raise HTTPException(status_code=status_code, detail=err_msg)

    # 在工作线程中运行同步后端网关请求，防止阻塞 FastAPI 事件循环
    res = await asyncio.to_thread(agent.run, req.message, token)
    if res.get("tool_status") == "AUTH_REJECTED":
        raise HTTPException(status_code=401, detail=res.get("reply", "Unauthorized"))

    return {
        "code": 200,
        "message": "AI 解析响应成功",
        "data": res,
    }


if __name__ == "__main__":
    uvicorn.run("main:app", host="127.0.0.1", port=8000, reload=False)
