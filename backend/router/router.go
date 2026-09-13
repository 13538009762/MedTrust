package router

import (
	"github.com/gin-gonic/gin"
	"medtrust-backend/controller"
	"medtrust-backend/middleware"
	"medtrust-backend/model"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, model.Response{Code: 200, Message: "MedTrust Blockchain Backend Gateway is Running", Data: nil})
	})

	v1 := r.Group("/api/v1")
	{
		// 认证模块 (公开)
		v1.POST("/auth/login", controller.DefaultAuthController.Login)
		v1.POST("/auth/register", controller.DefaultAuthController.Register)

		// 附件与影像查阅/下载 (支持 <img> / 标签页直接渲染)
		v1.GET("/medical-files/:id/view", controller.DefaultMedicalController.ViewMedicalFile)
		v1.GET("/medical-files/:id/download", controller.DefaultMedicalController.DownloadMedicalFile)

		// 需登录鉴权保护路由
		authGroup := v1.Group("")
		authGroup.Use(middleware.JWTAuthMiddleware())
		{
			authGroup.GET("/auth/profile", controller.DefaultAuthController.Profile)
			authGroup.PUT("/auth/profile", controller.DefaultAuthController.UpdateProfile)
			authGroup.PUT("/auth/password", controller.DefaultAuthController.ChangePassword)
			authGroup.PUT("/auth/medical-key", controller.DefaultAuthController.ChangeMedicalKey)

			// 医疗记录与就诊生命周期模块
			authGroup.POST("/medical-records/upload", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.Upload)
			authGroup.POST("/encounters/initial", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.CreateEncounterInitial)
			authGroup.POST("/encounters/:id/complete", middleware.RequireRoles("doctor"), controller.DefaultMedicalController.CompleteEncounterFinal)
			authGroup.GET("/medical-records", controller.DefaultMedicalController.List)
			authGroup.GET("/medical-records/:id", controller.DefaultMedicalController.GetByID)
			authGroup.GET("/medical-records/:id/download", controller.DefaultMedicalController.Download)
			authGroup.GET("/patients", middleware.RequireRoles("doctor", "admin", "supervisor"), controller.DefaultMedicalController.ListPatients)

			// 医技检查中心模块 (检验科/放射科/医生)
			authGroup.GET("/exam-orders", controller.DefaultMedicalController.ListExamOrders)
			authGroup.POST("/exam-orders/:id/process", middleware.RequireRoles("doctor", "admin"), controller.DefaultMedicalController.ProcessExamOrder)
			authGroup.POST("/exam-orders/:id/complete", middleware.RequireRoles("doctor", "admin"), controller.DefaultMedicalController.CompleteExamOrder)

			// 跨院与访问控制模块
			authGroup.POST("/access/requests", middleware.RequireRoles("doctor"), controller.DefaultAccessController.RequestAccess)
			authGroup.POST("/access/requests/apply-consent", middleware.RequireRoles("doctor"), controller.DefaultAccessController.ApplyConsent)
			authGroup.POST("/access/requests/unlock-by-key", middleware.RequireRoles("doctor"), controller.DefaultAccessController.UnlockByKey)
			authGroup.GET("/access/requests/pending", middleware.RequireRoles("patient"), controller.DefaultAccessController.ListPendingRequests)
			authGroup.POST("/access/requests/:id/approve", middleware.RequireRoles("patient"), controller.DefaultAccessController.ApproveRequest)
			authGroup.POST("/access/requests/:id/reject", middleware.RequireRoles("patient"), controller.DefaultAccessController.RejectRequest)
			authGroup.POST("/access/break-glass", middleware.RequireRoles("doctor"), controller.DefaultAccessController.BreakGlass)
			authGroup.POST("/access/break-glass/patient-feedback", middleware.RequireRoles("patient"), controller.DefaultAccessController.PatientFeedback)

			// 患者自主授权模块
			authGroup.POST("/authorizations", middleware.RequireRoles("patient"), controller.DefaultAccessController.CreateAuthorization)
			authGroup.GET("/authorizations", controller.DefaultAccessController.ListAuthorizations)
			authGroup.DELETE("/authorizations/:id", middleware.RequireRoles("patient"), controller.DefaultAccessController.RevokeAuthorization)

			// 监管看板模块
			authGroup.GET("/supervisor/overview", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.Overview)
			authGroup.GET("/supervisor/emergency-events", middleware.RequireRoles("supervisor", "admin", "patient"), controller.DefaultSupervisorController.ListEmergencyEvents)
			authGroup.POST("/supervisor/emergency-events/:event_no/audit", middleware.RequireRoles("supervisor"), controller.DefaultSupervisorController.AuditEmergencyEvent)
			authGroup.GET("/audit-logs", middleware.RequireRoles("supervisor", "admin"), controller.DefaultSupervisorController.ListAuditLogs)
			authGroup.POST("/verification/:record_id", middleware.RequireRoles("supervisor", "admin", "doctor"), controller.DefaultSupervisorController.VerifyRecord)

			// 系统管理员模块
			authGroup.GET("/system/users", middleware.RequireRoles("admin", "supervisor"), controller.DefaultSystemController.ListUsers)
			authGroup.PUT("/system/users/:id/status", middleware.RequireRoles("admin"), controller.DefaultSystemController.UpdateUserStatus)
			authGroup.POST("/system/users", middleware.RequireRoles("admin"), controller.DefaultSystemController.CreateUser)
			authGroup.GET("/system/hospitals", controller.DefaultSystemController.ListHospitals)
			authGroup.GET("/system/departments", controller.DefaultSystemController.ListDepartments)
			authGroup.GET("/system/doctors", controller.DefaultSystemController.ListDoctors)
		}
	}

	return r
}
