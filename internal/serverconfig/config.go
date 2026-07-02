package serverconfig

import (
	"errors"
	"fmt"
	"os"
	"strings"

	sharedruntimeconfig "github.com/tyemirov/utils/runtimeconfig"
)

const (
	// DefaultPath is the canonical backend runtime config path.
	DefaultPath = "configs/config.loopaware.yml"
)

var (
	// ErrMissingRequired reports an incomplete backend runtime config document.
	ErrMissingRequired = errors.New("serverconfig.missing_required")
)

// Config captures the populated backend runtime configuration.
type Config struct {
	ApplicationAddress        string
	DatabaseDriverName        string
	DatabaseDataSourceName    string
	AdminEmailAddresses       []string
	SessionSecret             string
	TauthBaseURL              string
	TauthTenantID             string
	TauthSigningKey           string
	TauthSessionCookieName    string
	PublicBaseURL             string
	ConfigFilePath            string
	PinguinAddress            string
	PinguinAuthToken          string
	PinguinTenantID           string
	PinguinConnTimeoutSec     int
	PinguinOpTimeoutSec       int
	SubscriptionNotifications bool
	TrafficReportEmails       bool
}

// Document is the strict YAML shape for the backend runtime config file.
type Document struct {
	Server        Server        `yaml:"server"`
	Database      Database      `yaml:"database"`
	Auth          Auth          `yaml:"auth"`
	Pinguin       Pinguin       `yaml:"pinguin"`
	Notifications Notifications `yaml:"notifications"`
	Admins        []string      `yaml:"admins"`
}

// Server holds HTTP runtime settings.
type Server struct {
	Address       string `yaml:"address"`
	PublicBaseURL string `yaml:"public_base_url"`
}

// Database holds storage runtime settings.
type Database struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// Auth holds backend authentication runtime settings.
type Auth struct {
	SessionSecret string `yaml:"session_secret"`
	Tauth         Tauth  `yaml:"tauth"`
}

// Tauth holds TAuth verifier runtime settings.
type Tauth struct {
	BaseURL           string `yaml:"base_url"`
	TenantID          string `yaml:"tenant_id"`
	JWTSigningKey     string `yaml:"jwt_signing_key"`
	SessionCookieName string `yaml:"session_cookie_name"`
}

// Pinguin holds notification service runtime settings.
type Pinguin struct {
	Address                  string `yaml:"address"`
	AuthToken                string `yaml:"auth_token"`
	TenantID                 string `yaml:"tenant_id"`
	ConnectionTimeoutSeconds int    `yaml:"connection_timeout_seconds"`
	OperationTimeoutSeconds  int    `yaml:"operation_timeout_seconds"`
}

// Notifications holds outbound notification toggles.
type Notifications struct {
	SubscriptionEnabled        *bool `yaml:"subscription_enabled"`
	TrafficReportEmailsEnabled *bool `yaml:"traffic_report_emails_enabled"`
}

// Load reads the selected backend runtime config through the process environment.
func Load(configFilePath string) (Config, error) {
	return LoadWithLookup(configFilePath, os.LookupEnv)
}

// LoadWithLookup reads the selected backend runtime config through an injected interpolation lookup.
func LoadWithLookup(configFilePath string, expansionLookup sharedruntimeconfig.ExpansionLookup) (Config, error) {
	loader, loaderError := sharedruntimeconfig.NewLoader[Document](sharedruntimeconfig.Contract[Document]{
		DefaultConfigPath: DefaultPath,
		ExpansionLookup:   expansionLookup,
	})
	if loaderError != nil {
		return Config{}, loaderError
	}
	loadedConfig, loadError := loader.Load(configFilePath)
	if loadError != nil {
		return Config{}, loadError
	}
	config, configError := NewConfig(loadedConfig.Path, loadedConfig.Config)
	if configError != nil {
		return Config{}, configError
	}
	return config, nil
}

// NewConfig converts a strict YAML document into the populated runtime config.
func NewConfig(configFilePath string, document Document) (Config, error) {
	config := Config{
		ApplicationAddress:        strings.TrimSpace(document.Server.Address),
		DatabaseDriverName:        strings.TrimSpace(document.Database.Driver),
		DatabaseDataSourceName:    strings.TrimSpace(document.Database.DSN),
		AdminEmailAddresses:       normalizeEmailAddresses(document.Admins),
		SessionSecret:             strings.TrimSpace(document.Auth.SessionSecret),
		TauthBaseURL:              strings.TrimSpace(document.Auth.Tauth.BaseURL),
		TauthTenantID:             strings.TrimSpace(document.Auth.Tauth.TenantID),
		TauthSigningKey:           strings.TrimSpace(document.Auth.Tauth.JWTSigningKey),
		TauthSessionCookieName:    strings.TrimSpace(document.Auth.Tauth.SessionCookieName),
		PublicBaseURL:             strings.TrimSpace(document.Server.PublicBaseURL),
		ConfigFilePath:            strings.TrimSpace(configFilePath),
		PinguinAddress:            strings.TrimSpace(document.Pinguin.Address),
		PinguinAuthToken:          strings.TrimSpace(document.Pinguin.AuthToken),
		PinguinTenantID:           strings.TrimSpace(document.Pinguin.TenantID),
		PinguinConnTimeoutSec:     document.Pinguin.ConnectionTimeoutSeconds,
		PinguinOpTimeoutSec:       document.Pinguin.OperationTimeoutSeconds,
		SubscriptionNotifications: boolValue(document.Notifications.SubscriptionEnabled),
		TrafficReportEmails:       boolValue(document.Notifications.TrafficReportEmailsEnabled),
	}
	missingFields := missingRequiredFields(config, document)
	if len(missingFields) > 0 {
		return Config{}, fmt.Errorf("%w: %s", ErrMissingRequired, strings.Join(missingFields, ", "))
	}
	return config, nil
}

func normalizeEmailAddresses(rawEmailAddresses []string) []string {
	normalizedEmailAddresses := make([]string, 0, len(rawEmailAddresses))
	for _, rawEmailAddress := range rawEmailAddresses {
		trimmedEmailAddress := strings.TrimSpace(rawEmailAddress)
		if trimmedEmailAddress == "" {
			continue
		}
		normalizedEmailAddresses = append(normalizedEmailAddresses, trimmedEmailAddress)
	}
	return normalizedEmailAddresses
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func missingRequiredFields(config Config, document Document) []string {
	var missingFields []string
	appendBlankField := func(fieldPath string, value string) {
		if strings.TrimSpace(value) == "" {
			missingFields = append(missingFields, fieldPath)
		}
	}

	appendBlankField("server.address", config.ApplicationAddress)
	appendBlankField("server.public_base_url", config.PublicBaseURL)
	appendBlankField("database.driver", config.DatabaseDriverName)
	appendBlankField("database.dsn", config.DatabaseDataSourceName)
	appendBlankField("auth.session_secret", config.SessionSecret)
	appendBlankField("auth.tauth.base_url", config.TauthBaseURL)
	appendBlankField("auth.tauth.tenant_id", config.TauthTenantID)
	appendBlankField("auth.tauth.jwt_signing_key", config.TauthSigningKey)
	appendBlankField("auth.tauth.session_cookie_name", config.TauthSessionCookieName)
	appendBlankField("pinguin.address", config.PinguinAddress)
	appendBlankField("pinguin.auth_token", config.PinguinAuthToken)
	appendBlankField("pinguin.tenant_id", config.PinguinTenantID)
	if config.PinguinConnTimeoutSec <= 0 {
		missingFields = append(missingFields, "pinguin.connection_timeout_seconds")
	}
	if config.PinguinOpTimeoutSec <= 0 {
		missingFields = append(missingFields, "pinguin.operation_timeout_seconds")
	}
	if document.Notifications.SubscriptionEnabled == nil {
		missingFields = append(missingFields, "notifications.subscription_enabled")
	}
	if document.Notifications.TrafficReportEmailsEnabled == nil {
		missingFields = append(missingFields, "notifications.traffic_report_emails_enabled")
	}
	return missingFields
}
