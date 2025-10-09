package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/gauss"
	"go.uber.org/zap"
)

const (
	headerForwarded         = "Forwarded"
	headerXForwardedProto   = "X-Forwarded-Proto"
	headerXForwardedScheme  = "X-Forwarded-Scheme"
	headerXForwardedHost    = "X-Forwarded-Host"
	headerXForwardedPort    = "X-Forwarded-Port"
	forwardedProtoPrefix    = "proto="
	forwardedHostPrefix     = "host="
	headerValueSeparator    = ","
	forwardedPairSeparator  = ";"
	urlSchemeHTTPS          = "https"
	logEventResolveHandlers = "resolve_oauth_handlers"
	createServiceError      = "create oauth service"
	createHandlersError     = "create oauth handlers"
	parseBaseURLError       = "parse public base url"
	resolveBaseURLError     = "resolve request base url"
)

// Config captures dependencies for building OAuth handlers.
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	PublicBaseURL      string
	LocalRedirectPath  string
	Scopes             []string
	LoginTemplate      string
	Logger             *zap.Logger
}

// Handlers exposes HTTP handlers for Google OAuth integration.
type Handlers struct {
	configuration     Config
	configuredBaseURL *url.URL
	defaultHandlers   *gauss.Handlers
	defaultServeMux   *http.ServeMux
	handlerCache      map[string]*gauss.Handlers
	handlerCacheMutex sync.RWMutex
	logger            *zap.Logger
}

// NewHandlers constructs a Handlers instance using GAuss primitives.
func NewHandlers(configuration Config) (*Handlers, error) {
	logger := configuration.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	baseURL, parseErr := url.Parse(configuration.PublicBaseURL)
	if parseErr != nil {
		return nil, fmt.Errorf("%s: %w", parseBaseURLError, parseErr)
	}

	serviceInstance, serviceErr := gauss.NewService(
		configuration.GoogleClientID,
		configuration.GoogleClientSecret,
		configuration.PublicBaseURL,
		configuration.LocalRedirectPath,
		configuration.Scopes,
		configuration.LoginTemplate,
	)
	if serviceErr != nil {
		return nil, fmt.Errorf("%s: %w", createServiceError, serviceErr)
	}

	gaussHandlers, handlersErr := gauss.NewHandlers(serviceInstance)
	if handlersErr != nil {
		return nil, fmt.Errorf("%s: %w", createHandlersError, handlersErr)
	}

	defaultServeMux := http.NewServeMux()
	gaussHandlers.RegisterRoutes(defaultServeMux)

	return &Handlers{
		configuration:     configuration,
		configuredBaseURL: baseURL,
		defaultHandlers:   gaussHandlers,
		defaultServeMux:   defaultServeMux,
		handlerCache:      make(map[string]*gauss.Handlers),
		logger:            logger,
	}, nil
}

// RegisterRoutes wires the OAuth endpoints to the provided ServeMux.
func (handlers *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(constants.LoginPath, handlers.serveLogin)
	mux.HandleFunc(constants.GoogleAuthPath, handlers.handleGoogleAuth)
	mux.HandleFunc(constants.CallbackPath, handlers.handleCallback)
	mux.HandleFunc(constants.LogoutPath, handlers.defaultHandlers.Logout)
}

func (handlers *Handlers) serveLogin(responseWriter http.ResponseWriter, request *http.Request) {
	handlers.defaultServeMux.ServeHTTP(responseWriter, request)
}

func (handlers *Handlers) handleGoogleAuth(responseWriter http.ResponseWriter, request *http.Request) {
	dynamicHandlers, resolutionErr := handlers.handlersForRequest(request)
	if resolutionErr != nil {
		handlers.logger.Warn(logEventResolveHandlers, zap.Error(resolutionErr))
		http.Error(responseWriter, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	dynamicHandlers.Login(responseWriter, request)
}

func (handlers *Handlers) handleCallback(responseWriter http.ResponseWriter, request *http.Request) {
	dynamicHandlers, resolutionErr := handlers.handlersForRequest(request)
	if resolutionErr != nil {
		handlers.logger.Warn(logEventResolveHandlers, zap.Error(resolutionErr))
		http.Error(responseWriter, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	dynamicHandlers.Callback(responseWriter, request)
}

func (handlers *Handlers) handlersForRequest(request *http.Request) (*gauss.Handlers, error) {
	baseURL, baseErr := handlers.baseForRequest(request)
	if baseErr != nil {
		return nil, baseErr
	}

	handlers.handlerCacheMutex.RLock()
	cachedHandlers := handlers.handlerCache[baseURL]
	handlers.handlerCacheMutex.RUnlock()
	if cachedHandlers != nil {
		return cachedHandlers, nil
	}

	handlers.handlerCacheMutex.Lock()
	defer handlers.handlerCacheMutex.Unlock()

	cachedHandlers = handlers.handlerCache[baseURL]
	if cachedHandlers != nil {
		return cachedHandlers, nil
	}

	serviceInstance, serviceErr := gauss.NewService(
		handlers.configuration.GoogleClientID,
		handlers.configuration.GoogleClientSecret,
		baseURL,
		handlers.configuration.LocalRedirectPath,
		handlers.configuration.Scopes,
		handlers.configuration.LoginTemplate,
	)
	if serviceErr != nil {
		return nil, fmt.Errorf("%s: %w", createServiceError, serviceErr)
	}

	gaussHandlers, handlersErr := gauss.NewHandlers(serviceInstance)
	if handlersErr != nil {
		return nil, fmt.Errorf("%s: %w", createHandlersError, handlersErr)
	}

	handlers.handlerCache[baseURL] = gaussHandlers
	return gaussHandlers, nil
}

func (handlers *Handlers) baseForRequest(request *http.Request) (string, error) {
	scheme := handlers.resolveScheme(request)
	host := handlers.resolveHost(request)
	if host == "" {
		return "", fmt.Errorf("%s: %w", resolveBaseURLError, fmt.Errorf("empty host"))
	}

	port := handlers.resolvePort(request)
	if port != "" && !strings.Contains(host, ":") {
		host = host + ":" + port
	}

	baseCopy := *handlers.configuredBaseURL
	baseCopy.Scheme = scheme
	baseCopy.Host = host

	return baseCopy.String(), nil
}

func (handlers *Handlers) resolveScheme(request *http.Request) string {
	if forwardedProto := extractForwardedDirective(request.Header.Get(headerForwarded), forwardedProtoPrefix); forwardedProto != "" {
		return strings.ToLower(forwardedProto)
	}

	if protoHeader := firstHeaderValue(request.Header.Get(headerXForwardedProto)); protoHeader != "" {
		return strings.ToLower(protoHeader)
	}

	if schemeHeader := firstHeaderValue(request.Header.Get(headerXForwardedScheme)); schemeHeader != "" {
		return strings.ToLower(schemeHeader)
	}

	if request.TLS != nil {
		return urlSchemeHTTPS
	}

	if request.URL != nil && request.URL.Scheme != "" {
		return strings.ToLower(request.URL.Scheme)
	}

	if handlers.configuredBaseURL.Scheme != "" {
		return strings.ToLower(handlers.configuredBaseURL.Scheme)
	}

	return urlSchemeHTTPS
}

func (handlers *Handlers) resolveHost(request *http.Request) string {
	if forwardedHost := extractForwardedDirective(request.Header.Get(headerForwarded), forwardedHostPrefix); forwardedHost != "" {
		return forwardedHost
	}

	if hostHeader := firstHeaderValue(request.Header.Get(headerXForwardedHost)); hostHeader != "" {
		return hostHeader
	}

	if request.Host != "" {
		return request.Host
	}

	return handlers.configuredBaseURL.Host
}

func (handlers *Handlers) resolvePort(request *http.Request) string {
	return firstHeaderValue(request.Header.Get(headerXForwardedPort))
}

func firstHeaderValue(rawValue string) string {
	if rawValue == "" {
		return ""
	}

	segments := strings.Split(rawValue, headerValueSeparator)
	for _, segment := range segments {
		trimmedSegment := strings.TrimSpace(segment)
		if trimmedSegment != "" {
			return trimmedSegment
		}
	}

	return ""
}

func extractForwardedDirective(headerValue string, prefix string) string {
	if headerValue == "" {
		return ""
	}

	directives := strings.Split(headerValue, headerValueSeparator)
	for _, directive := range directives {
		trimmedDirective := strings.TrimSpace(directive)
		if trimmedDirective == "" {
			continue
		}

		pairs := strings.Split(trimmedDirective, forwardedPairSeparator)
		for _, pair := range pairs {
			trimmedPair := strings.TrimSpace(pair)
			if trimmedPair == "" {
				continue
			}

			lowerPair := strings.ToLower(trimmedPair)
			if !strings.HasPrefix(lowerPair, prefix) {
				continue
			}

			value := strings.TrimSpace(trimmedPair[len(prefix):])
			value = strings.Trim(value, "\"")
			if value != "" {
				return value
			}
		}
	}

	return ""
}
