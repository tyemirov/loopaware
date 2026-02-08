package httpapi

import "strings"

func normalizeBaseURL(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimRight(trimmed, "/")
}

func joinBaseURL(baseURL string, path string) string {
	normalizedBaseURL := normalizeBaseURL(baseURL)
	if normalizedBaseURL == "" {
		return path
	}
	if path == "" {
		return normalizedBaseURL
	}
	if strings.HasPrefix(path, "/") {
		return normalizedBaseURL + path
	}
	return normalizedBaseURL + "/" + path
}
