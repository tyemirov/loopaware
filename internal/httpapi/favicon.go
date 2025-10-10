package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	defaultFaviconCacheTTL     = 6 * time.Hour
	defaultFaviconMaxIconBytes = 128 * 1024
	defaultFaviconMaxHTMLBytes = 512 * 1024
)

var (
	defaultFaviconHTMLProbePaths = []string{
		"/",
		"/index.html",
		"/app",
		"/login",
	}
)

// FaviconResolver discovers favicons for a given allowed origin.
type FaviconResolver interface {
	Resolve(ctx context.Context, allowedOrigin string) (string, error)
	ResolveAsset(ctx context.Context, allowedOrigin string) (*FaviconAsset, error)
}

// FaviconAsset represents favicon binary contents and metadata.
type FaviconAsset struct {
	ContentType string
	Data        []byte
}

type faviconCacheEntry struct {
	value   string
	expires time.Time
}

type faviconCandidate struct {
	remoteURL string
	dataURL   string
	asset     *FaviconAsset
}

// HTTPFaviconResolver discovers favicons by issuing HTTP requests.
type HTTPFaviconResolver struct {
	httpClient   *http.Client
	logger       *zap.Logger
	cacheTTL     time.Duration
	maxIconBytes int64
	maxHTMLBytes int64
	cache        sync.Map
}

// NewHTTPFaviconResolver builds a resolver that caches successful (and empty) lookups.
func NewHTTPFaviconResolver(httpClient *http.Client, logger *zap.Logger) *HTTPFaviconResolver {
	resolver := &HTTPFaviconResolver{
		logger:       logger,
		cacheTTL:     defaultFaviconCacheTTL,
		maxIconBytes: defaultFaviconMaxIconBytes,
		maxHTMLBytes: defaultFaviconMaxHTMLBytes,
	}
	if httpClient != nil {
		resolver.httpClient = httpClient
	} else {
		resolver.httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return resolver
}

// Resolve returns a stable favicon URL or an empty string when discovery fails.
func (resolver *HTTPFaviconResolver) Resolve(ctx context.Context, allowedOrigin string) (string, error) {
	normalized := strings.TrimSpace(allowedOrigin)
	if normalized == "" {
		return "", nil
	}

	baseURL, parseErr := url.Parse(normalized)
	if parseErr != nil || baseURL == nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return "", nil
	}
	baseURL.Fragment = ""
	baseURL.RawQuery = ""

	cacheKey := strings.ToLower(baseURL.String())
	if entryValue, ok := resolver.cache.Load(cacheKey); ok {
		entry := entryValue.(faviconCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.value, nil
		}
	}

	candidate, lookupErr := resolver.lookupFavicon(ctx, baseURL)
	if lookupErr != nil && resolver.logger != nil {
		resolver.logger.Debug(
			"favicon_lookup_failed",
			zap.String("allowed_origin", allowedOrigin),
			zap.Error(lookupErr),
		)
	}

	resolved := ""
	if candidate != nil {
		if candidate.dataURL != "" {
			resolved = candidate.dataURL
		} else {
			resolved = candidate.remoteURL
		}
	}

	resolver.cache.Store(cacheKey, faviconCacheEntry{
		value:   resolved,
		expires: time.Now().Add(resolver.cacheTTL),
	})

	return resolved, nil
}

// ResolveAsset returns the favicon contents for the given allowed origin.
func (resolver *HTTPFaviconResolver) ResolveAsset(ctx context.Context, allowedOrigin string) (*FaviconAsset, error) {
	normalized := strings.TrimSpace(allowedOrigin)
	if normalized == "" {
		return nil, nil
	}

	baseURL, parseErr := url.Parse(normalized)
	if parseErr != nil || baseURL == nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, nil
	}
	baseURL.Fragment = ""
	baseURL.RawQuery = ""

	candidate, lookupErr := resolver.lookupFavicon(ctx, baseURL)
	if lookupErr != nil {
		if resolver.logger != nil {
			resolver.logger.Debug(
				"favicon_lookup_failed",
				zap.String("allowed_origin", allowedOrigin),
				zap.Error(lookupErr),
			)
		}
		return nil, lookupErr
	}
	if candidate == nil {
		return nil, nil
	}
	if candidate.asset == nil {
		return nil, nil
	}
	return candidate.asset, nil
}

func (resolver *HTTPFaviconResolver) lookupFavicon(ctx context.Context, baseURL *url.URL) (*faviconCandidate, error) {
	root := &url.URL{
		Scheme: baseURL.Scheme,
		Host:   baseURL.Host,
	}

	if candidate, err := resolver.fetchDefaultFavicon(ctx, root); err == nil && candidate != nil {
		return candidate, nil
	}

	return resolver.fetchHTMLDeclaredFavicon(ctx, root, resolver.htmlProbePaths(baseURL))
}

func (resolver *HTTPFaviconResolver) fetchDefaultFavicon(ctx context.Context, root *url.URL) (*faviconCandidate, error) {
	candidate := root.ResolveReference(&url.URL{Path: "/favicon.ico"})
	asset, err := resolver.fetchRemoteIconAsset(ctx, candidate.String())
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	return &faviconCandidate{
		remoteURL: candidate.String(),
		asset:     asset,
	}, nil
}

func (resolver *HTTPFaviconResolver) fetchHTMLDeclaredFavicon(ctx context.Context, root *url.URL, probePaths []string) (*faviconCandidate, error) {
	var lastErr error
	for _, path := range probePaths {
		pageURL := root.ResolveReference(&url.URL{Path: path})
		body, err := resolver.get(ctx, pageURL.String(), resolver.maxHTMLBytes)
		if err != nil {
			lastErr = err
			continue
		}
		if body == nil {
			continue
		}

		document, parseErr := html.Parse(body)
		body.Close()
		if parseErr != nil {
			lastErr = parseErr
			continue
		}

		candidates := findFaviconCandidates(document)
		for _, candidate := range candidates {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(candidate), "data:") {
				asset, parseErr := resolver.parseDataURL(candidate)
				if parseErr != nil {
					lastErr = parseErr
					continue
				}
				return &faviconCandidate{
					dataURL: candidate,
					asset:   asset,
				}, nil
			}
			absoluteURL, resolveErr := resolver.absoluteURL(root, candidate)
			if resolveErr != nil {
				lastErr = resolveErr
				continue
			}
			asset, fetchErr := resolver.fetchRemoteIconAsset(ctx, absoluteURL)
			if fetchErr != nil {
				lastErr = fetchErr
				if resolver.logger != nil {
					resolver.logger.Debug(
						"favicon_candidate_fetch_failed",
						zap.String("candidate", absoluteURL),
						zap.Error(fetchErr),
					)
				}
				continue
			}
			if asset != nil {
				return &faviconCandidate{
					remoteURL: absoluteURL,
					asset:     asset,
				}, nil
			}
		}
	}

	return nil, lastErr
}

func (resolver *HTTPFaviconResolver) fetchRemoteIconAsset(ctx context.Context, iconURL string) (*FaviconAsset, error) {
	response, err := resolver.doRequest(ctx, http.MethodGet, iconURL)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, nil
	}

	limited := io.LimitReader(response.Body, resolver.maxIconBytes+1)
	data, readErr := io.ReadAll(limited)
	if readErr != nil {
		return nil, readErr
	}
	if int64(len(data)) > resolver.maxIconBytes {
		return nil, fmt.Errorf("favicon exceeds %d bytes", resolver.maxIconBytes)
	}
	if len(data) == 0 {
		return nil, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type")))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !isSupportedFaviconContentType(contentType) {
		return nil, nil
	}
	return &FaviconAsset{
		ContentType: contentType,
		Data:        data,
	}, nil
}

func (resolver *HTTPFaviconResolver) parseDataURL(value string) (*FaviconAsset, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("empty data url")
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return nil, errors.New("invalid data url")
	}

	commaIndex := strings.Index(trimmed, ",")
	if commaIndex < 0 {
		return nil, errors.New("invalid data url")
	}

	metadataSection := trimmed[len("data:"):commaIndex]
	payloadSection := trimmed[commaIndex+1:]

	mediaType := "application/octet-stream"
	isBase64 := false
	if metadataSection != "" {
		segments := strings.Split(metadataSection, ";")
		if len(segments) > 0 {
			primary := strings.TrimSpace(segments[0])
			if primary != "" {
				mediaType = primary
			}
		}
		for _, segment := range segments[1:] {
			if strings.EqualFold(strings.TrimSpace(segment), "base64") {
				isBase64 = true
			}
		}
	}

	var data []byte
	var decodeErr error
	if isBase64 {
		data, decodeErr = decodeBase64Data(payloadSection)
	} else {
		decoded, unescapeErr := url.PathUnescape(payloadSection)
		if unescapeErr != nil {
			decodeErr = unescapeErr
		} else {
			data = []byte(decoded)
		}
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	if len(data) == 0 {
		return nil, errors.New("empty data url payload")
	}
	if int64(len(data)) > resolver.maxIconBytes {
		return nil, fmt.Errorf("favicon exceeds %d bytes", resolver.maxIconBytes)
	}

	if !isSupportedFaviconContentType(mediaType) {
		return nil, errors.New("unsupported data url content type")
	}

	return &FaviconAsset{
		ContentType: mediaType,
		Data:        data,
	}, nil
}

func decodeBase64Data(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("empty base64 payload")
	}

	data, err := base64.StdEncoding.DecodeString(trimmed)
	if err == nil {
		return data, nil
	}

	return base64.RawStdEncoding.DecodeString(trimmed)
}

func isSupportedFaviconContentType(contentType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	if normalized == "" {
		return true
	}
	if normalized == "application/octet-stream" || normalized == "binary/octet-stream" {
		return true
	}
	if strings.HasPrefix(normalized, "image/") {
		return true
	}
	if strings.Contains(normalized, "icon") {
		return true
	}
	if strings.Contains(normalized, "svg") {
		return true
	}
	return false
}

func (resolver *HTTPFaviconResolver) htmlProbePaths(baseURL *url.URL) []string {
	ordered := make([]string, 0, len(defaultFaviconHTMLProbePaths)+4)
	seen := make(map[string]struct{})

	addPath := func(candidate string) {
		normalized := strings.TrimSpace(candidate)
		if normalized == "" {
			normalized = "/"
		}
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		ordered = append(ordered, normalized)
	}

	if baseURL != nil {
		basePath := strings.TrimSpace(baseURL.EscapedPath())
		if basePath != "" && basePath != "/" {
			addPath(basePath)
			if !strings.HasSuffix(basePath, "/") {
				addPath(basePath + "/")
			}
			stripped := strings.TrimSuffix(basePath, "/")
			if stripped != "" {
				addPath(stripped + "/index.html")
			}
		}
	}

	for _, fallback := range defaultFaviconHTMLProbePaths {
		addPath(fallback)
	}

	return ordered
}

func (resolver *HTTPFaviconResolver) get(ctx context.Context, target string, limit int64) (io.ReadCloser, error) {
	response, err := resolver.doRequest(ctx, http.MethodGet, target)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, nil
	}
	if response.StatusCode >= http.StatusBadRequest {
		response.Body.Close()
		return nil, nil
	}
	return newLimitedReadCloser(response.Body, limit), nil
}

func (resolver *HTTPFaviconResolver) doRequest(ctx context.Context, method string, target string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(resolver.requestContext(ctx), method, target, nil)
	if err != nil {
		return nil, err
	}
	return resolver.httpClient.Do(request)
}

func (resolver *HTTPFaviconResolver) requestContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func (resolver *HTTPFaviconResolver) absoluteURL(root *url.URL, href string) (string, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	resolved := root.ResolveReference(parsed)
	return resolved.String(), nil
}

func findFaviconCandidates(node *html.Node) []string {
	var candidates []string
	var traverse func(*html.Node)
	traverse = func(current *html.Node) {
		if current == nil {
			return
		}
		if current.Type == html.ElementNode && strings.EqualFold(current.Data, "link") {
			var relValue string
			var hrefValue string
			for _, attribute := range current.Attr {
				switch strings.ToLower(attribute.Key) {
				case "rel":
					relValue = strings.ToLower(attribute.Val)
				case "href":
					hrefValue = attribute.Val
				}
			}
			if relValue != "" && hrefValue != "" && relContainsIcon(relValue) {
				candidates = append(candidates, hrefValue)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(node)
	return candidates
}

func relContainsIcon(relValue string) bool {
	if relValue == "" {
		return false
	}
	normalized := strings.ToLower(relValue)
	if strings.Contains(normalized, "icon") {
		return true
	}
	if strings.Contains(normalized, "apple-touch-icon") {
		return true
	}
	if strings.Contains(normalized, "mask-icon") {
		return true
	}
	return false
}

type limitedReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func newLimitedReadCloser(closer io.ReadCloser, limit int64) io.ReadCloser {
	if limit <= 0 {
		return closer
	}
	return limitedReadCloser{
		reader: io.LimitReader(closer, limit),
		closer: closer,
	}
}

func (limited limitedReadCloser) Read(buffer []byte) (int, error) {
	return limited.reader.Read(buffer)
}

func (limited limitedReadCloser) Close() error {
	return limited.closer.Close()
}
