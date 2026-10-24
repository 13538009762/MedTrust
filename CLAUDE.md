# MedTrust 架构守护与工程规范 (Architecture Guard & Engineering Rules)

> 本文件为 **MedTrust (基于区块链与动态零信任的跨院医疗数据共享与协同分析系统)** 的最高架构规范约束文件。所有在此项目工作的 AI Agent 及开发者必须严格遵守以下工程红线。

---

## 🛑 四大不可逾越工程红线 (Core Architectural Guardrails)

### 1. 单一安全守门人（AccessEngine）
- **核心原则**：所有跨院医疗数据访问、患者跨院授权校验、动态上下文风险评分及 IDOR（越权漏洞）防御，**必须 100% 收口至统一安全守门人 `backend/service/access_engine.go`** (`service.DefaultAccessEngine.Evaluate`)。
- **严禁事项**：
  - 严禁在业务 Service 或 Controller 中自行分散编写鉴权、授权或越权校验代码；
  - 任何针对病历 (`/api/v1/medical/records/*`)、文件 (`/api/v1/medical/files/*`) 的访问，必须构造 `service.AccessRequest` 并由 `DefaultAccessEngine.Evaluate` 裁决；
  - 只有在裁决通过后，方可执行业务操作并调用 `service.DefaultAccessEngine.RecordSuccessfulAccess` 记录行为基线。

### 2. AI Agent 绝对受控与隔离（Strict Tool Isolation）
- **核心原则**：`ai_agent/` 模块定位为无状态辅助分析网关，**只能通过 HTTP Client 携带调用者的 JWT Bearer Token** 访问 Go 后端暴露的公开 API (`/api/v1/*`)。
- **严禁事项**：
  - 严禁在 `ai_agent/` 中引入 MySQL 驱动（如 `pymysql`, `mysqlclient`）、Redis 客户端或 Fabric SDK 直连底层数据源；
  - 严禁在 Python 侧绕过后端鉴权直接读写链码或 IPFS 节点；
  - AI Agent 工具执行严格执行 `ROLE_TOOL_ALLOWLIST`（患者/医生/监管员/管理员的角色-工具白名单），严禁越权调用工具。

### 3. 双模容灾保持完好（Dual-Mode Live/Mock Resilient Fallback）
- **核心原则**：系统必须维持全栈双模运行能力（Production Live 与 Local Mock），单机断网与离线状态下绝不崩溃。
- **架构保证**：
  - **区块链层**：支持 Hyperledger Fabric 真实网络 (`Mode: "fabric"`) 与内存模拟账本 (`Mode: "mock"`, `MockLedger`) 的无缝降级；
  - **去中心化存储**：支持真实 Kubo IPFS 节点与本地哈希模拟存储 (`LocalStorageAdapter`)；
  - **缓存与日志**：Redis 断开连接时，自动降级为单机内存缓存或安全跳过，不得触发 panic 或阻塞服务主流程；
  - **启动要求**：开发者无需启动 Docker/Fabric 容器，仅凭 MySQL 即可单机完整运行并跑通全部端到端业务与前端流程。

### 4. 提交与构建验证（Strict Build & Test Verification）
- **后端验证**：任何后端逻辑改动后，必须执行基础单元测试并确保通过：
  ```bash
  cd backend && go test ./... -v
  ```
- **前端验证**：
  - 任何前端代码改动后，必须保证 TypeScript 编译无类型报错：
    ```bash
    cd frontend && npm run build
    ```
  - 前端 UI 规范：基于 Vue 3 + Element Plus，严格避免视觉元素遮挡、图表容器高度塌陷或跨域拦截。

---

## 🛠️ 技术栈与目录拓扑

- **后端**：Go 1.22+ (Gin Web Framework, GORM, MySQL 8.0, Hyperledger Fabric Go SDK, go-ipfs-api, JWT, Crypto)
  - `backend/cmd/main.go`：服务入口与双模初始化
  - `backend/service/access_engine.go`：核心动态零信任访问决策引擎
  - `backend/controller/`：API 路由控制器
  - `backend/pkg/`：区块链、加密学（ABE/AES-GCM/SM4）、风险评估算法
- **前端**：Vue 3 + TypeScript + Vite + Element Plus + Pinia + Vue Router + ECharts
  - `frontend/src/views/`：患者端、医生端、审计端、系统拓扑可视化大屏
- **AI Agent**：Python 3.10+ (FastAPI / CLI, OpenAI-compatible Tool Calling)
  - `ai_agent/tools.py`：受控 HTTP 工具集与 JWT 代理调用
