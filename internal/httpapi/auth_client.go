package httpapi

import "strings"

type AuthClientConfig struct {
	GoogleClientID string
	TauthBaseURL   string
	TauthTenantID  string
	TauthScriptURL string
}

func NewAuthClientConfig(googleClientID string, tauthBaseURL string, tauthTenantID string) AuthClientConfig {
	normalizedBaseURL := strings.TrimSpace(tauthBaseURL)
	scriptURL := TauthScriptPath
	if normalizedBaseURL != "" {
		scriptURL = strings.TrimRight(normalizedBaseURL, "/") + TauthScriptPath
	}
	return AuthClientConfig{
		GoogleClientID: strings.TrimSpace(googleClientID),
		TauthBaseURL:   normalizedBaseURL,
		TauthTenantID:  strings.TrimSpace(tauthTenantID),
		TauthScriptURL: scriptURL,
	}
}
