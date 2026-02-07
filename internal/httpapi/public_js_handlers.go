package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PublicJavaScriptHandlers struct{}

func NewPublicJavaScriptHandlers() *PublicJavaScriptHandlers {
	return &PublicJavaScriptHandlers{}
}

func (handlers *PublicJavaScriptHandlers) WidgetJS(context *gin.Context) {
	context.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(widgetJavaScriptSource))
}

func (handlers *PublicJavaScriptHandlers) SubscribeJS(context *gin.Context) {
	context.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(subscribeJavaScriptSource))
}

func (handlers *PublicJavaScriptHandlers) PixelJS(context *gin.Context) {
	context.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(pixelJavaScriptSource))
}
