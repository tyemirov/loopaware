package model

import "strings"

var browserAuditUserAgentTokens = [...]string{
	"headlesschrome",
	"google-inspectiontool",
	"chrome-lighthouse",
}

// IsBrowserAuditUserAgent identifies recognizable browser inspection and audit tools.
func IsBrowserAuditUserAgent(userAgent string) bool {
	normalized := strings.ToLower(userAgent)
	for _, token := range browserAuditUserAgentTokens {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}
