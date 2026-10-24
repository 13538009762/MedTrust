# MedTrust 全栈修复与加固任务计划 (task_plan.md)

## 📌 任务背景与目标
针对《MedTrust 全栈深度体检报告》中发现的问题进行完整闭环修复，确保系统 100% 满足以下核心准则：
- 单一安全守门人权限对齐；
- 医疗存证数据不可篡改与字段归一性；
- 区块链通道与链码全局统一规范；
- 满足最小权限安全准则及 HTTP 响应规范；
- 保证全部自动化测试（Go、Python、Vue）零报错通过。

---

## 📋 修复任务分解与执行清单

- [x] **Task 1: [P0] 修复感控预警接口路由层角色权限倒置**
  - **文件**：`backend/router/router.go:55`
  - **内容**：将 `RequireRoles("doctor", "admin", "supervisor")` 调整为 `RequireRoles("doctor", "supervisor", "patient")`，对齐 Controller 校验并支持患者自查。
  - **状态**：已完成 (100% 对齐 Controller 防越权逻辑)

- [x] **Task 2: [P0] 修复紧急破窗反馈字符串拼接，保护医生原始临床记录**
  - **文件**：`backend/service/emergency_service.go:110-113`
  - **内容**：使用实体原生 `event.PatientComment = comment` 存储反馈，严禁覆盖或拼接破坏 `event.Description`。
  - **状态**：已完成 (独立存储患者意见，医生临床理由零污染)

- [x] **Task 3: [P1] 全局统一 Fabric 区块链通道与链码合约命名**
  - **文件**：`backend/config/config.yaml`, `backend/model/entity.go`, `backend/pkg/blockchain/*`, `backend/service/verification_service.go`, `deploy/fabric/*`, `scripts/*`
  - **内容**：通道名全局统一为 `medicalchannel`，链码合约名全局统一为 `medical_contract`。
  - **状态**：已完成 (全链路通道与链码标识符规范化收口)

- [x] **Task 4: [P1] 规范 `ViewMedicalFile` 内联预览文件名响应头**
  - **文件**：`backend/controller/medical_controller.go:632`
  - **内容**：使用 RFC 5987 标准 `filename*=UTF-8''%s` + `url.QueryEscape` 格式化内联预览文件名。
  - **状态**：已完成 (杜绝中文及特殊字符文件名解析异常)

- [x] **Task 5: [P2] 优化患者端授权默认项为“最小权限原则”**
  - **文件**：`frontend/src/views/patient/AuthManagerView.vue`
  - **内容**：主动授权表单默认值调整为指定医生（`DOCTOR`）与单病历（`SINGLE`），防范全量泄露误操作。
  - **状态**：已完成 (默认最小化授权，UI 体验与安全性同步提升)

- [x] **Task 6: [Gate] 自动化测试与质量闭环验收**
  - **内容**：
    1. 执行 Go 后端全量测试：`go test ./...` -> 全部通过 (100% PASS)
    2. 执行 Python AI Agent 测试：`python -m unittest discover` -> 6 项测试全部通过 (100% PASS)
    3. 执行 Vue 3 前端静态检查与构建：`npm run build` (`vue-tsc -b && vite build`) -> 0 错误编译通过 (100% PASS)
  - **状态**：全部通过验证，顺利闭环
