# MedTrust 跨医院医疗数据可信共享与全流程监管平台 2.0

> **MedTrust** 是一套基于 **区块链防篡改存证**、**动态风险评估引擎**、**患者自主知情授权** 与 **受控 AI 辅助智能检索** 的跨医疗机构电子病历安全共享与可信协同系统。

---

## 🌟 核心特性与架构概览

```
                     ┌──────────────────────────────────────────────┐
                     │          Vue 3 现代化医疗协作前端              │
                     │  (Vite + Pinia + TS + Element Plus + ECharts)│
                     │                 [Port: 5173]                 │
                     └──────────────┬───────────────────────────────┘
                                    │ HTTP / RESTful API (JWT Auth)
                                    ▼
                     ┌──────────────────────────────────────────────┐
                     │          Go 核心业务与安全网关服务             │
                     │       (Gin + GORM + Merkle + RiskEngine)     │
                     │                 [Port: 8080]                 │
                     └──────┬───────────────┬────────────────┬──────┘
                            │               │                │
            ┌───────────────┘               │                └───────────────┐
            ▼                               ▼                                ▼
┌───────────────────────┐       ┌───────────────────────┐        ┌───────────────────────┐
│     MySQL 8.0 数据库   │       │  Hyperledger Fabric   │        │   Python AI Agent     │
│ (9张核心表: 医生/患者/ │       │  区块链存证与验证账本  │        │(FastAPI+受控ToolCall) │
│  电子病历/授权/审计等)│       │  (Fabric/Mock 双模)   │        │     [Port: 8000]      │
│     [Port: 3306]      │       │  (Merkle Root + TxID) │        └───────────────────────┘
└───────────────────────┘       └───────────────────────┘
            ▲
            │ 攻防演练直连注入
┌───────────────────────┐
│ 区块链防篡改攻防控制台 │
│     [Port: 8090]      │
└───────────────────────┘
```

---

## 📁 规范化项目目录结构

```text
MedTrust/
├── backend/                       # Go 核心业务与区块链网关后端 (Port 8080)
│   ├── cmd/                       # 后端入口 (main.go)
│   ├── config/                    # 数据库与服务配置 (config.yaml)
│   ├── controller/                # 控制层 (用户/病历/授权/审计/急救/系统)
│   ├── data/                      # 区块链本地账本与 IPFS 存证数据
│   ├── middleware/                # JWT 鉴权、跨域与审计日志中间件
│   ├── model/                     # 数据模型与 GORM 结构体
│   ├── pkg/                       # 风险评估引擎、加密与 Merkle 算法
│   ├── repository/                # 数据库访问仓储层
│   ├── router/                    # 路由注册映射
│   ├── service/                   # 业务逻辑编排
│   ├── go.mod / go.sum            # Go 依赖清单
│   └── medtrust_backend.exe       # 编译生成的独立执行文件
│
├── frontend/                      # Vue 3 现代化前端应用 (Port 5173)
│   ├── src/
│   │   ├── api/                   # Axios API 接口封装
│   │   ├── assets/                # 静态图标与样式资源
│   │   ├── components/            # 全局通用组件
│   │   ├── layouts/               # 系统多角色统一骨架 (MainLayout.vue)
│   │   ├── router/                # 路由定义与权限守卫
│   │   ├── stores/                # Pinia 状态管理 (用户/系统)
│   │   ├── types/                 # TypeScript 类型定义
│   │   └── views/                 # 业务视图 (医生/患者/监管/管理/AI工作台)
│   ├── package.json               # Node 依赖清单
│   └── vite.config.ts             # Vite 构建与代理配置
│
├── ai_agent/                      # Python FastAPI 受控 AI Agent (Port 8000)
│   ├── main.py                    # FastAPI 启动服务入口
│   ├── agent.py                   # 受控业务工具调用与安全检索智能体
│   ├── tools.py                   # 外部受控工具链封装 (通过 JWT 请求安全网关)
│   └── requirements.txt           # Python 依赖清单
│
├── tamper_tool/                   # 区块链防篡改攻防演练控制台 (Port 8090)
│   ├── server.py                  # 独立攻防演练服务
│   ├── tamper_cli.py              # CLI 命令行篡改工具
│   └── start_tamper_console.bat   # 启动批处理
│
├── database/                      # 数据库建表与演示数据脚本
│   ├── init.sql                   # 基础建库 SQL
│   ├── setup_db.py                # 9 张核心数据表结构创建与迁移
│   └── seed_db.py                 # 全量多维演示数据填充 (12份国家标准病历)
│
├── deploy/                        # 容器化部署与区块链拓扑编排
│   └── fabric/                    # Hyperledger Fabric 2.5 联盟链集群 (Orderer, Org1, Org2, CLI)
│       ├── docker-compose-test-net.yaml # 容器网络编排定义
│       ├── crypto-config.yaml     # 医院组织与 Orderer MSP 证书生成配置
│       └── configtx.yaml          # medchannel 通道创世配置
│
├── docs/                          # 项目开发文档与需求规范
│   ├── fabric_deployment.md       # 🔗 Hyperledger Fabric 2.5 部署与双模操作指南
│   ├── 2.0.md                     # 2.0 版本升级规格书
│   ├── 何欢恒任务书.docx           # 毕业设计/竞赛任务书
│   └── 何欢恒大纲.docx             # 论文/报告大纲
│
├── scripts/                       # 自动化运维与部署脚本
│   ├── start-fabric.bat / .sh     # ⚡ 一键启动 Fabric 2.5 容器集群、加入通道并部署合约
│   ├── stop-fabric.bat / .sh      # 🛑 一键安全停止 Fabric 网络与销毁链码容器
│   └── scaffold/                  # 历史代码脚手架归档
│
├── chaincode/                     # Go 医疗智能合约源码 (名称: medical, 通道: medchannel)
│   ├── medical_contract.go        # 病历存证、患者授权与审计日志上链逻辑
│   └── go.mod                     # fabric-contract-api-go 依赖清单
│
├── init_db.bat                    # 🚀 [快捷脚本] 一键初始化数据库与预置数据
├── start_all.bat                  # 🚀 [快捷脚本] 一键并发拉起所有微服务并打开浏览器
├── stop_all.bat                   # 🛑 [快捷脚本] 一键精准扫描并停止所有微服务进程 (8080/8000/5173/8090)
├── start_backend.bat              # 🚀 [快捷脚本] 单独启动 Go 后端服务
├── start_frontend.bat             # 🚀 [快捷脚本] 单独启动 Vue 3 前端服务
├── start_ai.bat                   # 🚀 [快捷脚本] 单独启动 Python AI Agent
├── 启动区块链防篡改攻击演示控制台.bat # 🚀 [快捷脚本] 启动底层数据库防篡改演练控制台
└── README.md                      # 本说明文档
```

---

## 🚀 快速启动与停止指引

### 1. 环境准备
- **MySQL** 8.0+（默认端口 `3306`，账号 `root`，密码 `123456`）
- **Go** 1.22+
- **Node.js** 18+ 与 npm
- **Python** 3.10+ (需安装 `pymysql`, `uvicorn`, `fastapi`, `requests` 等)

### 2. 数据库一键初始化
双击根目录下的 **`init_db.bat`**，系统将自动创建 `medtrust` 数据库，构建 9 张标准数据表并写入多院多病种完整演示数据。

### 3. Hyperledger Fabric 真实联盟链启动 (可选/答辩推荐)
如需体验真实联盟链与智能合约（涵盖 Org1-第一人民医院、Org2-省立中心医院、Orderer、medchannel 通道与 medical 链码）：
- **启动 Fabric 真实网络**：运行 `scripts\start-fabric.bat` (Linux/macOS 运行 `scripts/start-fabric.sh`)。
  启动成功后终端显示 `Fabric Network Started Successfully`，并在 Docker 中运行 Peer、Orderer 与链码容器。
- **停止 Fabric 网络**：运行 `scripts\stop-fabric.bat` (Linux/macOS 运行 `scripts/stop-fabric.sh`)。
- **双模灵活切换**：
  - `BLOCKCHAIN_MODE=fabric` (默认推荐)：连接真实 Fabric Gateway 进行提案背书与上链出块，若网络不可达将严格熔断提示 `Fabric Unavailable`，杜绝虚假上链；
  - `BLOCKCHAIN_MODE=mock`：自动启用轻量级 `MockLedger` 内存与本地 JSON 账本，适合无 Docker 环境下的快速逻辑联调。
- 更多详细部署、拓扑与合约配置见文档：[docs/fabric_deployment.md](docs/fabric_deployment.md)。

### 4. 一键启动全套系统
双击根目录下的 **`start_all.bat`**，系统将自动并发拉起：
1. **Go 后端 API 网关**：`http://localhost:8080` (支持实时通过 `/api/v1/system/blockchain/status` 侦测链状态)
2. **Python AI Agent 服务**：`http://localhost:8000`
3. **Vue 3 现代化前端**：`http://localhost:5173` (监管大屏与顶部导航直观呈现 Fabric/Mock 运行状态卡片)
并在 3 秒后自动唤起默认浏览器进入系统登录页。

### 5. 一键安全停止所有服务
双击根目录下的 **`stop_all.bat`**，脚本会**精准根据微服务监听端口（8080, 8000, 5173, 8090）**识别并安全终止对应进程，不影响电脑上其他开发环境。

---

## 👥 预置演示账号与角色矩阵

| 角色 | 登录账号 | 默认密码 | 所属机构 / 科室 | 核心演示场景 |
| :--- | :--- | :--- | :--- | :--- |
| **主治医师 (医院A)** | `doc_a` | `123456` | 第一人民医院 · 心血管内科 | 门诊接诊、跨院已有权限调阅、未授权病历调取申请、病历录入 |
| **主治医师 (医院B)** | `doc_b` | `123456` | 省立中心医院 · 急救中心 | 破窗紧急调阅、跨院紧急抢救、急诊会诊 |
| **主任医师 (医院C)** | `doc_c` | `123456` | 协和医学中心 · 神经内科 | 专科跨院协同、慢病联合诊治 |
| **患者画像 (张三)** | `pat_zhang` | `123456` | 居民个人账户 | 个人全生命周期病历看板、主动跨院数据授权、急救破窗知情同意确认 |
| **患者画像 (李四)** | `pat_li` | `123456` | 居民个人账户 | 慢病心血管病历管理、精准单份病历授权 |
| **卫健监管专员** | `supervisor`| `123456` | 卫健委数据安全监管局 | 全网跨院调阅审计、紧急破窗合规审查、异常高风险调阅惩戒处置 |
| **系统管理员** | `admin` | `123456` | 平台运营中心 | 用户权限分配、机构与科室治理、平台全局安全配置 |

---

## 🛡️ 区块链防篡改攻防演练

双击根目录下的 **`启动区块链防篡改攻击演示控制台.bat`**：
1. 打开浏览器攻防控制台：`http://127.0.0.1:8090`
2. 演示模拟黑客绕过业务系统，直接在底层数据库篡改患者病历/诊断数据；
3. 返回前端主系统执行“病历调阅”或“区块链存证校验”，系统即时通过 Merkle 根哈希比对报警 **“数据已被非法篡改，链上校验失败”**，生动展示区块链可信存证的防篡改威慑力。
