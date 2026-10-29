package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/controller"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, model.Response{Code: 200, Message: "MedTrust Blockchain Backend Gateway is Running", Data: nil})
	})

	v1 := r.Group("/api/v1")
	{
		// 接收浏览器 CSP 违规上报
		v1.POST("/csp-report", func(c *gin.Context) {
			c.Status(204)
		})

		// 认证模块 (公开，挂载防爆破限流保护)
		v1.GET("/auth/captcha", controller.DefaultAuthController.GetCaptcha)
		v1.POST("/auth/login", middleware.RateLimitMiddleware(30, time.Minute), controller.DefaultAuthController.Login)
		v1.POST("/auth/register", middleware.RateLimitMiddleware(10, time.Minute), controller.DefaultAuthController.Register)

		// 区块链网络状态 (公开可查)
		v1.GET("/system/blockchain/status", controller.DefaultSystemController.GetBlockchainStatus)

		// 需登录鉴权保护路由

		authGroup := v1.Group("")
		authGroup.Use(middleware.JWTAuthMiddleware())
		{
			authGroup.POST("/auth/logout", controller.DefaultAuthController.Logout)
			authGroup.POST("/auth/ticket", controller.DefaultAuthController.CreateSSETicket)
			authGroup.GET("/auth/profile", controller.DefaultAuthController.Profile)
			authGroup.PUT("/auth/profile", controller.DefaultAuthController.UpdateProfile)
			authGroup.PUT("/auth/avatar", controller.DefaultAuthController.UpdateAvatar)
			authGroup.POST("/auth/avatar/upload", controller.DefaultAuthController.UploadAvatar)
			authGroup.PUT("/auth/password", controller.DefaultAuthController.ChangePassword)
			authGroup.PUT("/auth/medical-key", controller.DefaultAuthController.ChangeMedicalKey)

			// 附件与影像查阅/下载 (受 JWT 登录鉴权与访问控制引擎保护)
			authGroup.GET("/medical-files/:id/view", controller.DefaultMedicalController.ViewMedicalFile)
			authGroup.GET("/medical-files/:id/download", controller.DefaultMedicalController.DownloadMedicalFile)

			// 医疗记录与就诊生命周期模块
			authGroup.POST("/medical-records/upload", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.Upload)
			authGroup.POST("/encounters/initial", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.CreateEncounterInitial)
			authGroup.POST("/encounters/:id/complete", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.CompleteEncounterFinal)
			authGroup.GET("/medical-records", controller.DefaultMedicalController.List)
			authGroup.GET("/medical-records/:id", controller.DefaultMedicalController.GetByID)
			authGroup.GET("/medical-records/:id/download", controller.DefaultMedicalController.Download)
			authGroup.GET("/patients", middleware.RequireRoles("doctor", "admin", "supervisor"), controller.DefaultMedicalController.ListPatients)

			// 医护职业安全防护与高危传染病预警分析
			authGroup.GET("/patients/:id/infection-risks", middleware.RequireRoles("doctor", "supervisor", "patient"), controller.DefaultMedicalController.GetPatientInfectionRisks)
			authGroup.GET("/medical-records/:id/infection-risks", controller.DefaultMedicalController.GetRecordInfectionRisks)
			authGroup.POST("/infection-risks/evaluate-text", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.EvaluateInfectionText)
			authGroup.POST("/ai/diagnostic-assist", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.DiagnosticAssist)

			// 医技检查中心模块 (检验科/放射科/医生)
			authGroup.GET("/exam-orders", controller.DefaultMedicalController.ListExamOrders)
			authGroup.POST("/exam-orders/:id/process", middleware.RequireRoles("doctor", "admin"), controller.DefaultMedicalController.ProcessExamOrder)
			authGroup.POST("/exam-orders/:id/complete", middleware.RequireRoles("doctor", "admin"), controller.DefaultMedicalController.CompleteExamOrder)

			// 跨院与访问控制模块
			authGroup.POST("/access/requests", middleware.RequireRoles("doctor"), controller.DefaultAccessController.RequestAccess)
			authGroup.POST("/access-requests", middleware.RequireRoles("doctor"), controller.DefaultAccessController.RequestAccess)
			authGroup.POST("/access/requests/apply-consent", middleware.RequireRoles("doctor"), controller.DefaultAccessController.ApplyConsent)
			authGroup.POST("/access/batch-consent", middleware.RequireRoles("doctor"), controller.DefaultAccessController.BatchApplyConsent)
			authGroup.POST("/access/unlock-patient-key", middleware.RequireRoles("doctor"), controller.DefaultAccessController.UnlockPatientByKey)
			authGroup.POST("/access/emergency-batch", middleware.RequireRoles("doctor"), controller.DefaultAccessController.EmergencyBatchAccess)
			authGroup.POST("/access/requests/unlock-by-key", middleware.RequireRoles("doctor"), controller.DefaultAccessController.UnlockByKey)
			authGroup.GET("/access/requests/pending", middleware.RequireRoles("patient"), controller.DefaultAccessController.ListPendingRequests)
			authGroup.POST("/access/requests/:id/approve", middleware.RequireRoles("patient"), controller.DefaultAccessController.ApproveRequest)
			authGroup.POST("/access-requests/:id/confirm", middleware.RequireRoles("patient"), controller.DefaultAccessController.ApproveRequest)
			authGroup.POST("/access/requests/:id/reject", middleware.RequireRoles("patient"), controller.DefaultAccessController.RejectRequest)
			authGroup.POST("/access/break-glass", middleware.RequireRoles("doctor"), controller.DefaultAccessController.BreakGlass)
			authGroup.POST("/access/break-glass/patient-feedback", middleware.RequireRoles("patient"), controller.DefaultAccessController.PatientFeedback)
			authGroup.POST("/emergency-events/:event_no/feedback", middleware.RequireRoles("patient"), controller.DefaultAccessController.PatientFeedback)

			// 患者自主授权模块与病历可见性配置
			authGroup.POST("/authorizations", middleware.RequireRoles("patient"), controller.DefaultAccessController.CreateAuthorization)
			authGroup.GET("/authorizations", controller.DefaultAccessController.ListAuthorizations)
			authGroup.DELETE("/authorizations/revoke-all", middleware.RequireRoles("patient"), controller.DefaultAccessController.RevokeAllAuthorizations)
			authGroup.DELETE("/authorizations/:id", middleware.RequireRoles("patient"), controller.DefaultAccessController.RevokeAuthorization)
			authGroup.POST("/authorizations/:id/retry-revoke", middleware.RequireRoles("patient"), controller.DefaultAccessController.RetryRevokeAuthorization)
			authGroup.POST("/authorizations/retry-failed-revokes", middleware.RequireRoles("patient"), controller.DefaultAccessController.RetryFailedRevocations)
			authGroup.POST("/medical-records/:id/access-policy", middleware.RequireRoles("patient"), controller.DefaultAccessController.SetRecordAccessPolicy)
			authGroup.POST("/medical-records/batch-access-policy", middleware.RequireRoles("patient"), controller.DefaultAccessController.BatchSetAccessPolicy)

			// 实时通知模块 (SSE 长连接推送与离线待办)
			authGroup.GET("/notifications/stream", controller.DefaultNotificationController.Stream)
			authGroup.GET("/notifications", controller.DefaultNotificationController.ListNotifications)

			// 监管看板模块与区块链浏览器
			authGroup.GET("/blockchain/transactions", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.ListBlockchainTransactions)
			authGroup.GET("/supervisor/blockchain-transactions", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.ListBlockchainTransactions)
			authGroup.GET("/supervisor/overview", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.Overview)
			authGroup.GET("/supervisor/dashboard-stats", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.Overview)
			authGroup.GET("/supervisor/access-heatmap", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.AccessHeatmap)
			authGroup.GET("/supervisor/hospital-flow", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.HospitalFlow)
			authGroup.GET("/supervisor/auth-funnel", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.AuthFunnel)
			authGroup.GET("/supervisor/emergency-events", middleware.RequireRoles("supervisor", "admin", "patient"), controller.DefaultSupervisorController.ListEmergencyEvents)
			authGroup.POST("/supervisor/emergency-events/:event_no/audit", middleware.RequireRoles("supervisor"), controller.DefaultSupervisorController.AuditEmergencyEvent)
			authGroup.GET("/audit-logs", middleware.RequireRoles("supervisor", "admin", "patient"), controller.DefaultSupervisorController.ListAuditLogs)
			authGroup.POST("/supervisor/lift-doctor-restriction", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.LiftDoctorRestriction)
			authGroup.POST("/verification/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor", "patient"), controller.DefaultSupervisorController.VerifyRecord)
			authGroup.GET("/verification/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor", "patient"), controller.DefaultSupervisorController.VerifyRecord)
			authGroup.POST("/verification/simulate-tamper/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor"), controller.DefaultSupervisorController.SimulateTamper)
			authGroup.POST("/supervisor/simulate-tamper/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor"), controller.DefaultSupervisorController.SimulateTamper)
			authGroup.POST("/verification/restore/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor"), controller.DefaultSupervisorController.RestoreTamperedRecord)

			// 系统管理员模块
			authGroup.GET("/system/users", middleware.RequireRoles("admin", "supervisor"), controller.DefaultSystemController.ListUsers)
			authGroup.GET("/users", middleware.RequireRoles("admin", "supervisor"), controller.DefaultSystemController.ListUsers)
			authGroup.PUT("/system/users/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.PATCH("/system/users/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.PUT("/users/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.PATCH("/users/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.POST("/system/users", middleware.RequireRoles("admin"), controller.DefaultSystemController.CreateUser)
			authGroup.GET("/system/hospitals", controller.DefaultSystemController.ListHospitals)
			authGroup.GET("/system/departments", controller.DefaultSystemController.ListDepartments)
			authGroup.GET("/system/doctors", controller.DefaultSystemController.ListDoctors)

			// 医疗机构与科室全生命周期管理
			authGroup.POST("/system/hospitals", middleware.RequireRoles("admin"), controller.DefaultSystemController.CreateHospital)
			authGroup.PUT("/system/hospitals/:id", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateHospital)
			authGroup.PUT("/system/hospitals/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateHospitalStatus)
			authGroup.POST("/system/departments", middleware.RequireRoles("admin"), controller.DefaultSystemController.CreateDepartment)
			authGroup.PUT("/system/departments/:id", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateDepartment)
			authGroup.DELETE("/system/departments/:id", middleware.RequireRoles("admin"), controller.DefaultSystemController.DeleteDepartment)
		}
	}

	return r
}
