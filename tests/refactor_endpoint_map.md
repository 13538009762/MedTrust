# MedTrust 重构端点与特征测试映射表 (Refactor Endpoint Map)

本表记录 Phase 3 将被下沉的 6 组 Controller 核心业务端点与对应的 HTTP 级特征测试及覆盖状态。未覆盖端点严禁进行逻辑下沉。

| # | 领域模块 | HTTP 方法与路径 | 涉及现有 Controller 方法 | 直连 DB 数 | 覆盖测试用例 |
|---|---|---|---|---|---|
| 1 | 系统与机构 | `GET /api/v1/system/blockchain/status` | `SystemController.GetBlockchainStatus` | 0 | `TestCharacterization_SystemStatus` |
| 2 | 系统与机构 | `POST /api/v1/system/hospitals` | `SystemController.CreateHospital` | 1 | `TestHospitalManagement_CRUD_AndD1` |
| 3 | 系统与机构 | `PUT /api/v1/system/hospitals/:id/status` | `SystemController.UpdateHospitalStatus` | 1 | `TestHospitalManagement_CRUD_AndD1`, `TestHospitalDisabled_DecisionD1_AccessEngineRejection` |
| 4 | 系统与机构 | `PUT /api/v1/system/users/:id/status` | `SystemController.UpdateUserStatus` | 3 | `TestCharacterization_SystemStatus`, `TestUserStatusCacheSpeedup` |
| 5 | 系统与机构 | `POST /api/v1/system/departments` | `SystemController.CreateDepartment` | 1 | `TestHospitalManagement_CRUD_AndD1` |
| 6 | 通知中心 | `GET /api/v1/notifications` | `NotificationController.ListNotifications` | 2 | `TestCharacterization_Notifications` |
| 7 | 通知中心 | `POST /api/v1/access-requests/:id/confirm` | `NotificationController.ConfirmAccessRequest` | 4 | `TestCharacterization_Notifications` |
| 8 | 监管审计 | `GET /api/v1/audit-logs` | `SupervisorController.ListAuditLogs` | 1 | `TestCharacterization_SupervisorAudit` |
| 9 | 监管审计 | `GET /api/v1/supervisor/emergency-events` | `SupervisorController.ListEmergencyEvents` | 2 | `TestCharacterization_SupervisorAudit`, `TestE2E_AcceptanceFlow/Test4` |
| 10 | 监管审计 | `POST /api/v1/supervisor/emergency-events/:event_no/audit` | `SupervisorController.AuditEmergencyEvent` | 4 | `TestCharacterization_SupervisorAudit`, `TestE2E_AcceptanceFlow/Test4` |
| 11 | 监管审计 | `POST /api/v1/verification/:record_id` | `SupervisorController.VerifyRecord` | 1 | `TestE2E_AcceptanceFlow/Test5`, `TestVerification_AuthenticRecordPasses` |
| 12 | 医疗病历 | `POST /api/v1/medical-records/upload` | `MedicalController.Upload` | 4 | `TestE2E_AcceptanceFlow/Test1` |
| 13 | 医疗病历 | `GET /api/v1/medical-records` | `MedicalController.List` | 1 | `TestE2E_AcceptanceFlow/Test2` |
| 14 | 医疗病历 | `GET /api/v1/medical-records/:id` | `MedicalController.GetByID` | 2 | `TestE2E_AcceptanceFlow/Test2` |
| 15 | 医疗病历 | `GET /api/v1/medical-files/:id/view` | `MedicalController.ViewMedicalFile` | 1 | `TestE2E_AcceptanceFlow/Test1` |
| 16 | 访问控制 | `POST /api/v1/access/requests` | `AccessController.RequestAccess` | 4 | `TestCharacterization_AccessRequestLifecycle`, `TestE2E_AcceptanceFlow/Test3` |
| 17 | 访问控制 | `GET /api/v1/access/requests/pending` | `AccessController.ListPendingRequests` | 2 | `TestCharacterization_AccessRequestLifecycle` |
| 18 | 访问控制 | `POST /api/v1/access/requests/:id/reject` | `AccessController.RejectRequest` | 2 | `TestCharacterization_AccessRequestLifecycle` |
| 19 | 访问控制 | `POST /api/v1/authorizations` | `AccessController.CreateAuthorization` | 4 | `TestE2E_AcceptanceFlow/Test3` |
| 20 | 访问控制 | `DELETE /api/v1/authorizations/:id` | `AccessController.RevokeAuthorization` | 9 | `TestRevocationConsistency_FabricFailureAndIdempotency`, `TestE2E_AcceptanceFlow/Test3` |
| 21 | 访问控制 | `POST /api/v1/access/break-glass` | `AccessController.BreakGlass` | 6 | `TestE2E_AcceptanceFlow/Test4` |
| 22 | 访问控制 | `POST /api/v1/access/break-glass/patient-feedback` | `AccessController.PatientFeedback` | 3 | `TestE2E_AcceptanceFlow/Test4` |

*注：表中 22 项核心端点已 100% 建立特征测试与断言覆盖，无空行。*
