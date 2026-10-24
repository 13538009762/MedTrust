# MedTrust 架构守护与工程规范 (GEMINI.md)

详见项目主架构守护规范文件：[CLAUDE.md](file:///E:/EnglishEncoding/competition/last/MedTrust/CLAUDE.md)

### 核心不可逾越红线速查：
1. **单一安全守门人**：所有跨院数据访问、鉴权与 IDOR 拦截强制由 `backend/service/access_engine.go` (`DefaultAccessEngine.Evaluate`) 收口。
2. **AI Agent 绝对受控**：`ai_agent/` 仅能通过 HTTP Client 携带 JWT Token 调用 Go 后端，严禁直连底层数据库/Fabric/IPFS。
3. **双模容灾完好**：Fabric/IPFS/Redis 必须保证 Live 与 Mock 降级可用，单机离线状态下绝不崩溃。
4. **验证门禁**：后端变更保证 `go test ./...` 通过，前端变更保证 `npm run build` 零类型报错且界面无遮挡。
