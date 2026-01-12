// Package admin 管理端 HTTP Handler
package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	adminService "github.com/dumeirei/smart-locker-backend/internal/service/admin"
)

// SystemHandler 系统管理处理器
type SystemHandler struct {
	configService *adminService.SystemConfigService
}

// NewSystemHandler 创建系统管理处理器
func NewSystemHandler(configService *adminService.SystemConfigService) *SystemHandler {
	return &SystemHandler{configService: configService}
}

// ==================== 系统配置 ====================

// CreateConfig 创建配置
// @Summary 创建系统配置
// @Tags 管理-系统配置
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.CreateConfigRequest true "创建配置请求"
// @Success 200 {object} response.Response{data=models.SystemConfig}
// @Router /api/v1/admin/system/configs [post]
func (h *SystemHandler) CreateConfig(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req adminService.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, err := h.configService.Create(c.Request.Context(), &req)
	handler.MustSucceed(c, err, config)
}

// GetConfig 获取配置详情
// @Summary 获取系统配置详情
// @Tags 管理-系统配置
// @Produce json
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response{data=models.SystemConfig}
// @Router /api/v1/admin/system/configs/{id} [get]
func (h *SystemHandler) GetConfig(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "配置")
	if !ok {
		return
	}

	config, err := h.configService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "配置不存在")
		return
	}

	response.Success(c, config)
}

// UpdateConfig 更新配置
// @Summary 更新系统配置
// @Tags 管理-系统配置
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "配置ID"
// @Param request body adminService.UpdateConfigRequest true "更新配置请求"
// @Success 200 {object} response.Response{data=models.SystemConfig}
// @Router /api/v1/admin/system/configs/{id} [put]
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "配置")
	if !ok {
		return
	}

	var req adminService.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, err := h.configService.Update(c.Request.Context(), id, &req)
	handler.MustSucceed(c, err, config)
}

// DeleteConfig 删除配置
// @Summary 删除系统配置
// @Tags 管理-系统配置
// @Produce json
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/system/configs/{id} [delete]
func (h *SystemHandler) DeleteConfig(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	id, ok := handler.ParseID(c, "配置")
	if !ok {
		return
	}

	if err := h.configService.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// ListConfigs 获取配置列表
// @Summary 获取系统配置列表
// @Tags 管理-系统配置
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param group query string false "配置分组"
// @Param keyword query string false "关键词"
// @Param is_public query bool false "是否公开"
// @Success 200 {object} response.Response{data=response.ListData}
// @Router /api/v1/admin/system/configs [get]
func (h *SystemHandler) ListConfigs(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	p := handler.BindPaginationWithDefaults(c, 1, 20)
	group := c.Query("group")
	keyword := c.Query("keyword")

	var isPublic *bool
	if p := c.Query("is_public"); p != "" {
		val := p == "true" || p == "1"
		isPublic = &val
	}

	configs, total, err := h.configService.List(c.Request.Context(), p.Page, p.PageSize, group, keyword, isPublic)
	handler.MustSucceedPage(c, err, configs, total, p.Page, p.PageSize)
}

// GetConfigsByGroup 获取分组配置
// @Summary 获取分组下的系统配置
// @Tags 管理-系统配置
// @Produce json
// @Security Bearer
// @Param group path string true "配置分组"
// @Success 200 {object} response.Response{data=[]models.SystemConfig}
// @Router /api/v1/admin/system/configs/group/{group} [get]
func (h *SystemHandler) GetConfigsByGroup(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	group := c.Param("group")
	if group == "" {
		response.BadRequest(c, "分组不能为空")
		return
	}

	configs, err := h.configService.GetByGroup(c.Request.Context(), group)
	handler.MustSucceed(c, err, configs)
}

// GetAllGroups 获取所有配置分组
// @Summary 获取所有配置分组
// @Tags 管理-系统配置
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]string}
// @Router /api/v1/admin/system/configs/groups [get]
func (h *SystemHandler) GetAllGroups(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	groups, err := h.configService.GetAllGroups(c.Request.Context())
	handler.MustSucceed(c, err, groups)
}

// GetPublicConfigs 获取公开配置
// @Summary 获取所有公开配置
// @Tags 管理-系统配置
// @Produce json
// @Success 200 {object} response.Response{data=map[string]map[string]interface{}}
// @Router /api/v1/system/configs/public [get]
func (h *SystemHandler) GetPublicConfigs(c *gin.Context) {
	configs, err := h.configService.GetPublicConfigs(c.Request.Context())
	handler.MustSucceed(c, err, configs)
}

// BatchUpdateConfigs 批量更新配置
// @Summary 批量更新系统配置
// @Tags 管理-系统配置
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body adminService.BatchUpdateRequest true "批量更新请求"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/system/configs/batch [put]
func (h *SystemHandler) BatchUpdateConfigs(c *gin.Context) {
	if _, ok := handler.RequireAdminID(c); !ok {
		return
	}

	var req adminService.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.configService.BatchUpdate(c.Request.Context(), &req); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
