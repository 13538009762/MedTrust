import uvicorn
from fastapi import FastAPI, Header, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional
from agent import MedTrustAgent

app = FastAPI(
    title="MedTrust Controlled AI Agent Service",
    description="Python FastAPI AI Agent with Tool Calling strictly authenticated via Go Backend Gateway",
    version="2.0.0"
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

agent = MedTrustAgent()

class ChatRequest(BaseModel):
    message: str
    token: Optional[str] = None

@app.get("/health")
def health():
    return {"status": "ok", "service": "MedTrust AI Agent"}

@app.post("/api/v1/ai/chat")
@app.post("/api/ai/chat")
def chat(req: ChatRequest, authorization: Optional[str] = Header(None)):
    token = ""
    if authorization:
        auth_str = authorization.strip()
        if auth_str.startswith("Bearer "):
            token = auth_str[7:].strip()
        else:
            token = auth_str
    if not token and req.token:
        token = req.token.strip()

    if not token:
        raise HTTPException(status_code=401, detail="Missing or invalid Bearer authorization token")

    res = agent.run(req.message, token)
    return {
        "code": 200,
        "message": "AI 解析响应成功",
        "data": res
    }

if __name__ == "__main__":
    uvicorn.run("main:app", host="127.0.0.1", port=8000, reload=False)
