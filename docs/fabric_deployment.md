# MedTrust Hyperledger Fabric 2.5 本地联盟链部署与双模开发指南

本文档依据毕业设计与系统设计要求，详细介绍 MedTrust 平台中 **Hyperledger Fabric 2.5 真实联盟链网络** 的架构拓扑、一键部署脚本、Go 语言智能合约、双模切换机制与测试验证流程。

---

## 目录
1. [联盟链架构与网络拓扑](#1-联盟链架构与网络拓扑)
2. [链上存证原则与智能合约设计](#2-链上存证原则与智能合约设计)
3. [环境要求与前提条件](#3-环境要求与前提条件)
4. [一键启动与停止网络](#4-一键启动与停止网络)
5. [后端 Fabric Gateway 接入与双模运行](#5-后端-fabric-gateway-接入与双模运行)
6. [系统状态监控与前端大屏](#6-系统状态监控与前端大屏)
7. [自动化与集成测试验证](#7-自动化与集成测试验证)
8. [常见问题与故障排查](#8-常见问题与故障排查)

---

## 1. 联盟链架构与网络拓扑

MedTrust 采用面向医疗机构数据孤岛治理的联盟链架构，模拟两家医疗机构与排序共识组织：

| 节点名称 | 机构名称 | 角色 / 端口 | MSP 标识 | TLS 根证书位置 |
| :--- | :--- | :--- | :--- | :--- |
| `orderer.example.com` | Orderer Org | Raft 排序服务 (`7050`, 管理端 `7053`) | `OrdererMSP` | `deploy/fabric/crypto-config/ordererOrganizations/...` |
| `peer0.org1.example.com` | 第一人民医院 (Hospital A) | Endorser / Committer (`7051`) | `Org1MSP` | `deploy/fabric/crypto-config/peerOrganizations/org1.example.com/...` |
| `peer0.org2.example.com` | 省立中心医院 (Hospital B) | Endorser / Committer (`9051`) | `Org2MSP` | `deploy/fabric/crypto-config/peerOrganizations/org2.example.com/...` |
| `cli` | 管理工具容器 | Fabric Tools 2.5 运维与链码发布容器 | `Org1MSP` | `deploy/fabric/docker-compose-test-net.yaml` |

- **联盟通道 (Channel)**：`medchannel`，两家医疗机构与排序节点共同加入此通道。
- **智能合约 (Chaincode)**：名称为 `medical`，版本为 `1.0`，基于 Go 语言 `fabric-contract-api-go/v2` 编写。

---

## 2. 链上存证原则与智能合约设计

### 2.1 链上最小必要存证原则 (Off-chain Storage + On-chain Fingerprint)
为符合国家医疗数据安全规范与隐私合规要求，**严禁将包含大文件或患者隐私明文直接保存在区块链账本中**：
- **原始病历与医技影像文件**：本地生成客户端对称密钥（`DeriveRecordKey`），采用 **AES-256-GCM** 流式强加密，绑定附加身份数据 AAD，并将密文块推送至 **IPFS 去中心化存储节点**，生成去中心化唯一寻址哈希（CID）。
- **区块链智能合约账本**：仅保存可信存证元数据与防篡改指纹，占用空间极小且具备不可篡改与全流程追溯特性。

### 2.2 核心合约数据结构与方法 (`chaincode/medical_contract.go`)

| 业务实体 | 账本 Key 前缀 | 结构定义 | 说明 |
| :--- | :--- | :--- | :--- |
| **病历存证 (MedicalRecord)** | `RECORD_{record_id}` | `RecordID`, `CID`, `FileHash`, `HospitalID`, `DataType`, `Timestamp` | 仅记录链下 IPFS 寻址 CID、文件哈希及机构归属 |
| **患者授权 (Authorization)** | `AUTH_{auth_id}` | `AuthID`, `PatientID`, `DoctorID`, `HospitalID`, `Scope`, `ExpireTime`, `Status` | 记录患者知情同意策略与时效，支持即时吊销 |
| **安全审计 (AuditLog)** | `AUDIT_{audit_id}` | `AuditID`, `UserID`, `Action`, `RecordID`, `RiskLevel`, `Timestamp` | 固化调阅事件与风险评估打分，防止违规删除痕迹 |

核心合约接口：
- `CreateMedicalRecord(recordID, cid, fileHash, hospitalID, dataType, timestamp)`
- `QueryMedicalRecord(recordID) (*MedicalRecord, error)`
- `CreateAuthorization(authID, patientID, doctorID, hospitalID, scope, expireTime, status)`
- `RevokeAuthorization(authID)`
- `CreateAuditLog(auditID, userID, action, recordID, riskLevel, timestamp)`
- `QueryAuditLogs(recordID) ([]*AuditLog, error)`

---

## 3. 环境要求与前提条件

1. **操作系统**：Windows 10/11 (已安装 WSL2 或 Docker Desktop) 或 Linux / macOS
2. **Docker 环境**：Docker Desktop 正在运行，且启用了 Linux Containers
3. **Go 环境**：Go 1.21 及以上版本
4. **端口可用性**：确保本机以下端口未被其他服务占用：
   - `7050`, `7053` (Orderer)
   - `7051` (Peer0 Org1)
   - `9051` (Peer0 Org2)

---

## 4. 一键启动与停止网络

### 4.1 启动 Fabric 2.5 网络与部署智能合约
运行启动脚本，脚本会自动检测 Docker 环境、使用官方 `hyperledger/fabric-tools:2.5` 生成 MSP 与 TLS 证书、启动 Docker 容器、创建 `medchannel` 通道并打包部署 `medical` 链码：

- **Windows 环境**：
  ```cmd
  scripts\start-fabric.bat
  ```
- **Linux / macOS 环境**：
  ```bash
  chmod +x scripts/start-fabric.sh
  ./scripts/start-fabric.sh
  ```

执行成功后，终端将输出：
```text
=====================================================================
Fabric Network Started Successfully
=====================================================================
节点拓扑：
  - Orderer:    orderer.example.com:7050
  - Hospital A: peer0.org1.example.com:7051 (Org1MSP)
  - Hospital B: peer0.org2.example.com:9051 (Org2MSP)
  - 通道:        medchannel
  - 智能合约:    medical (Go Contract-API 1.2)
=====================================================================
```

### 4.2 停止并清理网络
运行关闭脚本，脚本会安全销毁网络容器、卷数据以及动态生成的链码容器：

- **Windows 环境**：
  ```cmd
  scripts\stop-fabric.bat
  ```
- **Linux / macOS 环境**：
  ```bash
  chmod +x scripts/stop-fabric.sh
  ./scripts/stop-fabric.sh
  ```

---

## 5. 后端 Fabric Gateway 接入与双模运行

MedTrust 后端内置官方 `fabric-gateway` Go SDK（v1.7.0），支持 **Fabric 真实网关模式** 与 **MockLedger 本地仿真模式**：

### 5.1 环境变量配置 (`BLOCKCHAIN_MODE`)

| 环境变量值 | 行为说明 | 适用场景 |
| :--- | :--- | :--- |
| `BLOCKCHAIN_MODE=fabric` | **严格真实 Fabric 模式**：通过 TLS 与 gRPC 连接 `peer0.org1.example.com:7051`，进行提案背书并提交 Raft 共识出块。<br>⚠️ **若底层网络未就绪，系统严格抛出 `Fabric Unavailable` 并拒绝虚假伪造上链**，保障论文真实性。 | 生产演示、答辩真实演示、端到端集成测试 |
| `BLOCKCHAIN_MODE=mock` | **MockLedger 本地模式**：使用本地内存与 JSON 账本进行仿真计算，不依赖 Docker。 | 无 Docker 环境、轻量单机开发、算法快速调试 |

### 5.2 配置文件映射 (`backend/config/config.yaml`)
```yaml
blockchain:
  mode: "fabric"           # 默认模式 (可被系统环境变量 BLOCKCHAIN_MODE 覆盖)
  channel_id: "medchannel"
  chaincode_id: "medical"
  fabric:
    enabled: true
    peer_endpoint: "127.0.0.1:7051"
    gateway_peer: "peer0.org1.example.com"
    msp_id: "Org1MSP"
    tls_cert_path: "./fabric_crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
    cert_path: "./fabric_crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/signcerts/Admin@org1.example.com-cert.pem"
    key_path: "./fabric_crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore"
```

---

## 6. 系统状态监控与前端大屏

### 6.1 状态检测 REST API
系统提供公开状态接口，任何端均可无感感知底层链状态：
- **请求方式**：`GET /api/v1/system/blockchain/status`
- **响应示例 (真实 Fabric 运行)**：
  ```json
  {
    "code": 200,
    "message": "success",
    "data": {
      "mode": "fabric",
      "connected": true,
      "network": "medchannel",
      "chaincode": "medical",
      "peer": "peer0.org1.example.com",
      "latest_block": 49,
      "message": "Fabric Network Running"
    }
  }
  ```
- **响应示例 (网络离线触发熔断)**：
  ```json
  {
    "code": 200,
    "message": "success",
    "data": {
      "mode": "fabric",
      "connected": false,
      "network": "medchannel",
      "chaincode": "medical",
      "peer": "peer0.org1.example.com",
      "latest_block": 0,
      "message": "Fabric Unavailable: connection refused at 127.0.0.1:7051"
    }
  }
  ```

### 6.2 前端状态可视化
- **监管审计大屏 (`/supervisor/overview`)**：
  顶部醒目呈现 **区块链底层网络实时运行状态卡片**，动态显示呼吸光效指示灯（绿色正常 / 橙色开发 / 红色离线）、通道标识、智能合约、Peer 节点、最新区块高度，并支持一键刷新状态。
- **全局顶部导航栏 (`MainLayout.vue`)**：
  在各角色页面右上角展示微缩区块链状态徽标与区块高度，悬浮提示底层节点与通信通道信息。

---

## 7. 自动化与集成测试验证

系统提供多层级验证测试用例，覆盖全部设计要求：

### 7.1 Go 自动化单元测试 (覆盖双模与离线熔断测试)
在 `backend` 目录下执行：
```bash
go test -v ./pkg/blockchain/...
```
测试结果包含三大核心用例：
1. `TestMockLedgerService`: 验证 MockLedger 本地模式存证与查询正常。
2. `TestFabricOfflineMode`: 模拟不可达端口，验证系统报告 `Fabric Unavailable`，且严格拒绝伪装上链，报错包含 `strictly refusing to falsify on-chain confirmation`。
3. `TestFabricGatewayLive`: 验证与本机 Fabric 节点通信、提案签名背书、区块高度自增与真实 TxID 获取。

### 7.2 端到端集成测试 (医生接诊 -> AES-GCM加密 -> IPFS存储 -> Fabric真实上链)
```bash
python scratch/test_fabric_live.py
```
测试流程：
1. 校验 `GET /api/v1/system/blockchain/status` 返回 `Fabric Network Running`。
2. 模拟医生登录第一人民医院（Org1）。
3. 建立门诊初诊档案，生成业务编号 `ENC2026...`。
4. 下达最终诊断与结构化处置方案，触发后端 AES-256-GCM 本地加密并推送 IPFS，向 Fabric 提交交易。
5. 验证返回的真实 64 位十六进制 Fabric Transaction ID 以及最新的区块高度，确认存证真实生效。

---

## 8. 常见问题与故障排查

### Q1: 运行 `start-fabric.bat` 提示 `Docker 守护进程未启动`
- **原因**：本地 Docker Desktop 尚未启动或 Engine 仍在 Booting 状态。
- **解决**：打开 Docker Desktop 客户端，确认左下角图标变为绿色 "Engine running" 即可。

### Q2: 为什么 `BLOCKCHAIN_MODE=fabric` 时不能像之前一样随便写入？
- **说明**：这是符合毕业设计诚信要求的安全特性。当声明使用真实联盟链时，系统严禁在无底层节点时假造成功。若临时需要在无 Docker 的电脑上演示，只需在终端中设置 `$env:BLOCKCHAIN_MODE="mock"` 或运行 `set BLOCKCHAIN_MODE=mock`，系统即会安全降级至开发仿真模式。

### Q3: 如何确认链码容器正在运行？
在终端运行：
```bash
docker ps --filter "name=dev-peer"
```
若已成功部署，可看到 `dev-peer0.org1.example.com-medical_1.0-...` 与 `dev-peer0.org2.example.com-medical_1.0-...` 两组容器正常运行。
