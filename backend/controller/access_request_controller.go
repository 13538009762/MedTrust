package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"medtrust-backend/model"
	"medtrust-backend/repository"
	"medtrust-backend/service"
)

type AccessRequestDTO struct {
	PatientID uint64 `json:"patient_id" binding:"required"`
	RecordID  uint64 `json:"record_id" binding:"required"`
	Purpose   string `json:"purpose"`
}

// serviceCtx 构造携带客户端 IP 的 context（service 层审计所需）
func serviceCtx(c *gin.Context) (ctx contextHolder) {
	return contextHolder{c: c}
}

// contextHolder gin.Context 到 context.Context 的轻量桥接
type contextHolder struct {
	c *gin.Context
}

func (h contextHolder) Deadline() (deadline time.Time, ok bool)  { return time.Time{}, false }
func (h contextHolder) Done() <-chan struct{}                    { return nil }
func (h contextHolder) Err() error                               { return nil }
func (h contextHolder) Value(key interface{}) interface{} {
	if _, isIPKey := key.(service.ClientIPContextKey); isIPKey {
		return h.c.ClientIP()
	}
	return nil
}

// RequestAccess 医生发起常规访问申请
func (ctrl *AccessController) RequestAccess(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req AccessRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误"})
		return
	}

	decision, err := service.DefaultAccessRequestService.SubmitAccessRequest(serviceCtx(c), doctorID, service.AccessRequestInput{
		PatientID: req.PatientID,
		RecordID:  req.RecordID,
		Purpose:   req.Purpose,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: err.Error()})
		return
	}
	if !decision.Allowed {
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: decision.Reason, Data: decision})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "权限评估完成", Data: decision})
}

type ApplyConsentDTO struct {
	RecordID  uint64 `json:"record_id"`
	PatientID uint64 `json:"patient_id"`
	Purpose   string `json:"purpose"`
	ScopeType string `json:"scope_type"` // SINGLE, ALL
	Days      int    `json:"days"`
}

// ApplyConsent 医生在常规诊疗场景下发起知情授权申请（通知患者在线核准）
func (ctrl *AccessController) ApplyConsent(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req ApplyConsentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少必要参数"})
		return
	}

	accessRec, err := service.DefaultAccessRequestService.ApplyConsent(serviceCtx(c), doctorID, service.ApplyConsentInput{
		RecordID:  req.RecordID,
		PatientID: req.PatientID,
		Purpose:   req.Purpose,
		ScopeType: req.ScopeType,
		Days:      req.Days,
	})
	if err != nil {
		respondAccessRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "知情同意申请已成功提交并通知患者，等待患者在线核准",
		Data:    accessRec,
	})
}

type BatchApplyConsentDTO struct {
	PatientID uint64   `json:"patient_id" binding:"required"`
	RecordIDs []uint64 `json:"record_ids" binding:"required"`
	Purpose   string   `json:"purpose"`
	Days      int      `json:"days"`
}

// BatchApplyConsent 医生分批/勾选申请多份病历的调阅知情同意
func (ctrl *AccessController) BatchApplyConsent(c *gin.Context) {
	doctorID := c.GetUint64("user_id")
	var req BatchApplyConsentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数错误：缺少患者ID或待申请病历列表"})
		return
	}

	createdRequests, err := service.DefaultAccessRequestService.BatchApplyConsent(serviceCtx(c), doctorID, service.BatchApplyConsentInput{
		PatientID: req.PatientID,
		RecordIDs: req.RecordIDs,
		Purpose:   req.Purpose,
		Days:      req.Days,
	})
	if err != nil {
		respondAccessRequestError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "已成功提交 " + strconv.Itoa(len(createdRequests)) + " 份病历的知情同意申请，已即时推送患者在线审批",
		Data:    createdRequests,
	})
}

func (ctrl *AccessController) ListPendingRequests(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	list, _ := service.DefaultAccessRequestService.ListPendingRequests(serviceCtx(c), patientID)
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "查询成功", Data: list})
}

// ApproveRequest 患者核准同意调阅申请，释放访问权限并自动建立上链授权
func (ctrl *AccessController) ApproveRequest(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	reqPtr, err := repository.DefaultAuthPolicyRepository.FindPatientAccessRequest(patientID, strconv.FormatUint(id, 10))
	if err != nil || reqPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该申请记录"})
		return
	}
	if reqPtr.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "该申请已处理过"})
		return
	}

	auth, err := service.DefaultAccessRequestService.ApproveRequest(serviceCtx(c), patientID, reqPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "核准失败：授权建立异常 " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: "已核准同意授权！权限已向经治医生释放并完成区块链固化存证",
		Data:    auth,
	})
}

// RejectRequest 患者驳回调阅申请
func (ctrl *AccessController) RejectRequest(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	reqPtr, err := repository.DefaultAuthPolicyRepository.FindPatientAccessRequest(patientID, strconv.FormatUint(id, 10))
	if err != nil || reqPtr == nil {
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "未找到该申请记录"})
		return
	}
	if reqPtr.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "该申请已处理，不可重复驳回"})
		return
	}

	_ = service.DefaultAccessRequestService.RejectRequest(serviceCtx(c), patientID, reqPtr)
	c.JSON(http.StatusOK, model.Response{Code: 200, Message: "已驳回该调阅申请"})
}

// BatchConsentProcessRequest 批量审批或一键全部审批/驳回请求体
type BatchConsentProcessRequest struct {
	RequestIDs []uint64 `json:"request_ids"`
	All        bool     `json:"all"`
}

// BatchApproveRequests 批量或一键全部同意知情调阅申请
func (ctrl *AccessController) BatchApproveRequests(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req BatchConsentProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数格式错误"})
		return
	}
	if !req.All && len(req.RequestIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请提供需批量核准的申请单ID列表或指定全部核准"})
		return
	}

	result, err := service.DefaultAccessRequestService.BatchApproveRequests(serviceCtx(c), patientID, req.RequestIDs, req.All)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "批量核准异常: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("批量处理完成：成功核准 %d 条，失败 %d 条", result.SuccessCount, result.FailedCount),
		Data:    result,
	})
}

// BatchRejectRequests 批量或一键全部驳回知情调阅申请
func (ctrl *AccessController) BatchRejectRequests(c *gin.Context) {
	patientID := c.GetUint64("user_id")
	var req BatchConsentProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "参数格式错误"})
		return
	}
	if !req.All && len(req.RequestIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: "请提供需批量驳回的申请单ID列表或指定全部驳回"})
		return
	}

	result, err := service.DefaultAccessRequestService.BatchRejectRequests(serviceCtx(c), patientID, req.RequestIDs, req.All)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: 500, Message: "批量驳回异常: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Code:    200,
		Message: fmt.Sprintf("批量处理完成：成功驳回 %d 条，失败 %d 条", result.SuccessCount, result.FailedCount),
		Data:    result,
	})
}

// respondAccessRequestError 依据领域错误类型映射 HTTP 状态码
func respondAccessRequestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "病历记录不存在"})
	case errors.Is(err, service.ErrPatientNotFound):
		c.JSON(http.StatusNotFound, model.Response{Code: 404, Message: "患者不存在"})
	case errors.Is(err, service.ErrMissingTarget):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrEmptyRecordList):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrRecordListMismatch):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	case errors.Is(err, service.ErrAuthDaysExceeded):
		c.JSON(http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
	default:
		// 跨患者越权等业务拒绝
		c.JSON(http.StatusForbidden, model.Response{Code: 403, Message: err.Error()})
	}
}
