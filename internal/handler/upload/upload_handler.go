// Package upload 提供文件上传相关的 HTTP Handler
package upload

import (
	"github.com/gin-gonic/gin"

	"github.com/dumeirei/smart-locker-backend/internal/common/handler"
	"github.com/dumeirei/smart-locker-backend/internal/common/response"
	uploadService "github.com/dumeirei/smart-locker-backend/internal/service/upload"
)

// Handler 上传处理器
type Handler struct {
	uploadService *uploadService.UploadService
}

// NewHandler 创建上传处理器
func NewHandler(uploadSvc *uploadService.UploadService) *Handler {
	return &Handler{
		uploadService: uploadSvc,
	}
}

// UploadImage 上传图片（通用）
// @Summary 上传图片
// @Description 上传图片文件，支持 jpg/jpeg/png/gif/webp 格式，最大 10MB
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "图片文件"
// @Param type formData string false "文件类型" default(images) Enums(images, hotel, room, product)
// @Success 200 {object} response.Response{data=uploadService.UploadImageResponse}
// @Router /api/v1/upload/image [post]
func (h *Handler) UploadImage(c *gin.Context) {
	// 验证用户已登录
	if _, ok := handler.RequireUserID(c); !ok {
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择要上传的文件")
		return
	}

	// 获取文件类型（可选）
	fileType := c.PostForm("type")

	// 调用服务层上传图片
	req := &uploadService.UploadImageRequest{
		File:     file,
		FileType: fileType,
	}

	result, err := h.uploadService.UploadImage(c.Request.Context(), req)
	handler.MustSucceed(c, err, result)
}

// UploadAvatar 上传头像
// @Summary 上传用户头像
// @Description 上传用户头像并自动更新用户信息，支持 jpg/jpeg/png/gif/webp 格式，最大 5MB
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "头像文件"
// @Success 200 {object} response.Response{data=uploadService.UploadImageResponse}
// @Router /api/v1/user/avatar [post]
func (h *Handler) UploadAvatar(c *gin.Context) {
	// 获取当前用户 ID
	userID, ok := handler.RequireUserID(c)
	if !ok {
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请选择要上传的文件")
		return
	}

	// 调用服务层上传头像
	req := &uploadService.UploadAvatarRequest{
		UserID: userID,
		File:   file,
	}

	result, err := h.uploadService.UploadAvatar(c.Request.Context(), req)
	handler.MustSucceed(c, err, result)
}
