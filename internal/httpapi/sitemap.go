package httpapi

import (
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tyemirov/GAuss/pkg/constants"
)

const (
	SitemapRoutePath     = "/sitemap.xml"
	sitemapContentType   = "application/xml; charset=utf-8"
	sitemapXMLNamespace  = "http://www.sitemaps.org/schemas/sitemap/0.9"
	sitemapRenderFailure = "sitemap_render_failed"
	sitemapDefaultBase   = "http://localhost:8080"
)

var sitemapStaticPaths = []string{
	constants.LoginPath,
	PrivacyPagePath,
}

type SitemapHandlers struct {
	baseURL    string
	routePaths []string
}

type sitemapURLEntry struct {
	Location string `xml:"loc"`
}

type sitemapURLSet struct {
	XMLName xml.Name          `xml:"urlset"`
	XMLNS   string            `xml:"xmlns,attr"`
	URLs    []sitemapURLEntry `xml:"url"`
}

func NewSitemapHandlers(baseURL string) *SitemapHandlers {
	return &SitemapHandlers{
		baseURL:    normalizeSitemapBaseURL(baseURL),
		routePaths: append([]string(nil), sitemapStaticPaths...),
	}
}

func (handlers *SitemapHandlers) RenderSitemap(context *gin.Context) {
	urlEntries := make([]sitemapURLEntry, 0, len(handlers.routePaths))
	for _, path := range handlers.routePaths {
		urlEntries = append(urlEntries, sitemapURLEntry{
			Location: handlers.composeURL(path),
		})
	}

	payload := sitemapURLSet{
		XMLNS: sitemapXMLNamespace,
		URLs:  urlEntries,
	}

	encoded, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": sitemapRenderFailure})
		return
	}

	document := append([]byte(xml.Header), encoded...)
	context.Data(http.StatusOK, sitemapContentType, document)
}

func (handlers *SitemapHandlers) composeURL(path string) string {
	normalizedPath := "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	return handlers.baseURL + normalizedPath
}

func normalizeSitemapBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	trimmed = strings.TrimRight(trimmed, "/")
	if trimmed == "" {
		return sitemapDefaultBase
	}
	return trimmed
}
