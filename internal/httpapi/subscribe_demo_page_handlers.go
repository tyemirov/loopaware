package httpapi

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscribeDemoPageHandlers struct {
	logger *zap.Logger
}

func NewSubscribeDemoPageHandlers(logger *zap.Logger) *SubscribeDemoPageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SubscribeDemoPageHandlers{logger: logger}
}

func (handlers *SubscribeDemoPageHandlers) RenderSubscribeDemo(context *gin.Context) {
	siteID := strings.TrimSpace(context.Query("site_id"))
	if siteID == "" {
		context.String(http.StatusBadRequest, "missing site_id")
		return
	}

	extraParams := url.Values{}
	for _, key := range []string{"mode", "accent", "cta", "success", "error", "name_field"} {
		value := strings.TrimSpace(context.Query(key))
		if value != "" {
			extraParams.Set(key, value)
		}
	}

	scriptURL := "/subscribe.js?site_id=" + url.QueryEscape(siteID)
	if encoded := extraParams.Encode(); encoded != "" {
		scriptURL += "&" + encoded
	}

	var buffer bytes.Buffer
	if err := subscribeDemoTemplate.Execute(&buffer, map[string]any{
		"SiteID":    siteID,
		"ScriptURL": scriptURL,
	}); err != nil {
		handlers.logger.Warn("render_subscribe_demo_page", zap.Error(err))
		context.String(http.StatusInternalServerError, "render error")
		return
	}

	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}
