import asyncio
import os
from typing import Any, Dict

import uvicorn
from fastapi import FastAPI, Header, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, ConfigDict, Field
from agent import MedTrustAgent

app = FastAPI(
    title="MedTrust Controlled AI Agent Service",
    description="Python FastAPI AI Agent with Tool Calling strictly authenticated via Go Backend Gateway",
    version="2.0.0",
)

# WP-04: 收紧 CORS——禁止 allow_origins=["*"] 与 allow_credentials=True 组合
_default_origins = ["http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:8080"]
_origins_raw = os.environ.get("MEDTRUST_CORS_ORIGINS", "")
allow_origins = [o.strip() for o in _origins_raw.split(",") if o.strip()] if _origins_raw else _default_origins

app.add_middleware(
    CORSMiddleware,
    allow_origins=allow_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["Authorization", "Content-Type"],
)

agent = MedTrustAgent()


class ChatRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="ignore")
    message: str = Field(..., min_length=1, max_length=2000)


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok", "service": "MedTrust AI Agent"}


@app.post("/api/v1/ai/chat")
@app.post("/api/ai/chat")
async def chat(req: ChatRequest, authorization: str | None = Header(None)) -> Dict[str, Any]:
    token = ""
    if authorization:
        auth_str = authorization.strip()
        if auth_str.lower().startswith("bearer "):
            token = auth_str[7:].strip()
        else:
            token = auth_str

    if not token:
        raise HTTPException(status_code=401, detail="Missing or invalid Bearer authorization token")

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
