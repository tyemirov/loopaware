package api

import (
	"fmt"
	"strings"
	"time"
)

func versionedSiteFaviconURL(siteID string, fetchedAt time.Time) string {
	normalizedID := strings.TrimSpace(siteID)
	if normalizedID == "" {
		return ""
	}
	base := fmt.Sprintf(siteFaviconURLTemplate, normalizedID)
	if fetchedAt.IsZero() {
		return base
	}
	return fmt.Sprintf("%s?ts=%d", base, fetchedAt.UTC().Unix())
}
