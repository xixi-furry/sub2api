package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ContentModerationHandler) GetV2Config(c *gin.Context) {
	cfg, e := h.service.GetModerationV2Config(c.Request.Context())
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, cfg)
}
func (h *ContentModerationHandler) UpdateV2Config(c *gin.Context) {
	var cfg service.ModerationV2Config
	if e := c.ShouldBindJSON(&cfg); e != nil {
		response.BadRequest(c, "Invalid moderation configuration")
		return
	}
	saved, e := h.service.UpdateModerationV2Config(c.Request.Context(), cfg)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, saved)
}
func (h *ContentModerationHandler) GetV2Usage(c *gin.Context) {
	data, e := h.service.GetModerationV2Usage(c.Request.Context())
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, data)
}
func (h *ContentModerationHandler) PreviewV2(c *gin.Context) {
	var req service.ModerationV2TestInput
	if c.ShouldBindJSON(&req) != nil || len(req.Text) > 256*1024 {
		response.BadRequest(c, "Text must fit in 256 KiB")
		return
	}
	data, e := h.service.PreviewModerationV2Input(c.Request.Context(), req)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, data)
}
func (h *ContentModerationHandler) TestV2(c *gin.Context) {
	var req service.ModerationV2TestInput
	if c.ShouldBindJSON(&req) != nil || len(req.Text) > 256*1024 {
		response.BadRequest(c, "Provide 1–262144 bytes of text")
		return
	}
	data, e := h.service.TestModerationV2Input(c.Request.Context(), req)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, data)
}
