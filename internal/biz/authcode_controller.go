package biz

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"

	"horserun/internal/model"
)

// Handler HTTP 接口处理器
type Handler struct {
	manager *Manager
}

// NewHandler 创建处理器实例
func NewHandler(manager *Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// GenerateRequest 生成授权码请求
type GenerateRequest struct {
	Type int `json:"type" binding:"required,min=1,max=4"` // 1:3 天，2:1 个月，3:3 个月，4:1 年
}

// GenerateResponse 生成授权码响应
type GenerateResponse struct {
	Code       string    `json:"code"`
	Type       string    `json:"type"`
	ExpiryTime time.Time `json:"expiry_time"`
	IsActive   bool      `json:"is_active"`
}

// ActivateResponse 激活授权码响应
type ActivateResponse struct {
	Code        string     `json:"code"`
	Type        string     `json:"type"`
	ExpiryTime  time.Time  `json:"expiry_time"`
	ActivatedAt *time.Time `json:"activated_at"`
	Remaining   string     `json:"remaining"`
}

// ValidateResponse 验证授权码响应
type ValidateResponse struct {
	Code       string    `json:"code"`
	Type       string    `json:"type"`
	ExpiryTime time.Time `json:"expiry_time"`
	Remaining  string    `json:"remaining"`
	IsValid    bool      `json:"is_valid"`
}

// CodeInfo 授权码信息
type CodeInfo struct {
	Code        string     `json:"code"`
	Type        string     `json:"type"`
	ExpiryTime  time.Time  `json:"expiry_time"`
	IsActive    bool       `json:"is_active"`
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	IsExpired   bool       `json:"is_expired"`
	Remaining   string     `json:"remaining,omitempty"`
}

// ListResponse 授权码列表响应
type ListResponse struct {
	Total int        `json:"total"`
	Codes []CodeInfo `json:"codes"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/authcode")
	{
		// POST /api/v1/authcode/generate - 生成授权码
		api.POST("/generate", h.Generate)

		// POST /api/v1/authcode/activate - 激活授权码
		api.POST("/activate", h.Activate)

		// GET /api/v1/authcode/validate/:code - 验证授权码
		api.GET("/validate/:code", h.Validate)

		// GET /api/v1/authcode/:code - 获取授权码详情
		api.GET("/:code", h.GetCode)

		// GET /api/v1/authcode/list - 获取所有授权码
		api.GET("/list", h.ListCodes)

		// DELETE /api/v1/authcode/:code - 删除授权码
		api.DELETE("/:code", h.DeleteCode)
	}
}

// Generate 生成授权码
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误：" + err.Error()})
		return
	}

	validityType := model.ValidityType(req.Type)
	if validityType < model.Validity3Days || validityType > model.Validity1Year {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的有效期类型，请输入 1-4"})
		return
	}

	authCode, err := h.manager.GenerateCode(validityType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "生成授权码失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, GenerateResponse{
		Code:       authCode.Code,
		Type:       validityType.String(),
		ExpiryTime: authCode.ExpiryTime,
		IsActive:   authCode.IsActive,
	})
}

// Activate 激活授权码
func (h *Handler) Activate(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误：" + err.Error()})
		return
	}

	authCode, err := h.manager.ActivateCode(req.Code)
	if err != nil {
		statusCode := http.StatusBadRequest
		switch err {
		case ErrCodeNotFound:
			statusCode = http.StatusNotFound
		case ErrCodeExpired:
		case ErrCodeAlreadyUsed:
		default:
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ActivateResponse{
		Code:        authCode.Code,
		Type:        model.ValidityType(authCode.Type).String(),
		ExpiryTime:  authCode.ExpiryTime,
		ActivatedAt: authCode.ActivatedAt,
		Remaining:   authCode.RemainingTime().Round(time.Hour).String(),
	})
}

// Validate 验证授权码
func (h *Handler) Validate(c *gin.Context) {
	code := c.Param("code")

	authCode, err := h.manager.ValidateCode(code)
	if err != nil {
		statusCode := http.StatusBadRequest
		switch err {
		case ErrCodeNotFound:
			statusCode = http.StatusNotFound
		case ErrCodeExpired:
		case ErrInvalidCode:
		default:
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ValidateResponse{
		Code:       authCode.Code,
		Type:       model.ValidityType(authCode.Type).String(),
		ExpiryTime: authCode.ExpiryTime,
		Remaining:  authCode.RemainingTime().Round(time.Second).String(),
		IsValid:    authCode.IsValid(),
	})
}

// GetCode 获取授权码详情
func (h *Handler) GetCode(c *gin.Context) {
	code := c.Param("code")

	authCode, err := h.manager.GetCode(code)
	if err != nil {
		if err == ErrCodeNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "授权码不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	remaining := ""
	if !authCode.IsExpired() {
		remaining = authCode.RemainingTime().Round(time.Second).String()
	}

	c.JSON(http.StatusOK, CodeInfo{
		Code:        authCode.Code,
		Type:        model.ValidityType(authCode.Type).String(),
		ExpiryTime:  authCode.ExpiryTime,
		IsActive:    authCode.IsActive,
		ActivatedAt: authCode.ActivatedAt,
		IsExpired:   authCode.IsExpired(),
		Remaining:   remaining,
	})
}

// ListCodes 获取所有授权码
func (h *Handler) ListCodes(c *gin.Context) {
	codes := h.manager.ListCodes()

	codeInfos := make([]CodeInfo, 0, len(codes))
	for _, code := range codes {
		remaining := ""
		if !code.IsExpired() {
			remaining = code.RemainingTime().Round(time.Second).String()
		}

		codeInfos = append(codeInfos, CodeInfo{
			Code:        code.Code,
			Type:        model.ValidityType(code.Type).String(),
			ExpiryTime:  code.ExpiryTime,
			IsActive:    code.IsActive,
			ActivatedAt: code.ActivatedAt,
			IsExpired:   code.IsExpired(),
			Remaining:   remaining,
		})
	}

	c.JSON(http.StatusOK, ListResponse{
		Total: len(codeInfos),
		Codes: codeInfos,
	})
}

// DeleteCode 删除授权码
func (h *Handler) DeleteCode(c *gin.Context) {
	code := c.Param("code")

	if err := h.manager.DeleteCode(code); err != nil {
		if err == ErrCodeNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "授权码不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "授权码已删除"})
}

// ParseValidityType 通过字符串解析有效期类型
func ParseValidityType(s string) (model.ValidityType, error) {
	typeNum, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	validityType := model.ValidityType(typeNum)
	if validityType < model.Validity3Days || validityType > model.Validity1Year {
		return 0, errors.New("无效的有效期类型")
	}

	return validityType, nil
}
