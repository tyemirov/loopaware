package httpapi

import (
	"context"
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

// FaviconResolver returns a favicon URL for a given allowed origin.
type FaviconResolver interface {
	Resolve(ctx context.Context, allowedOrigin string) (string, error)
}

type faviconCacheEntry struct {
	value   string
	expires time.Time
}

// HTTPFaviconResolver discovers favicons by issuing HTTP requests.
type HTTPFaviconResolver struct {
	httpClient       *http.Client
	logger           *zap.Logger
	cacheTTL         time.Duration
	maxIconBytes     int64
	maxHTMLBytes     int64
	cache            sync.Map
}

// NewHTTPFaviconResolver builds a resolver that caches successful (and empty) lookups.
func NewHTTPFaviconResolver(httpClient *http.Client, logger *zap.Logger) *HTTPFaviconResolver {
	resolver := &HTTPFaviconResolver{
		logger:           logger,
		cacheTTL:         defaultFaviconCacheTTL,
		maxIconBytes:     defaultFaviconMaxIconBytes,
		maxHTMLBytes:     defaultFaviconMaxHTMLBytes,
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

	resolved, lookupErr := resolver.lookupFavicon(ctx, baseURL)
	if lookupErr != nil && resolver.logger != nil {
		resolver.logger.Debug(
			"favicon_lookup_failed",
			zap.String("allowed_origin", allowedOrigin),
			zap.Error(lookupErr),
		)
	}

	resolver.cache.Store(cacheKey, faviconCacheEntry{
		value:   resolved,
		expires: time.Now().Add(resolver.cacheTTL),
	})

	return resolved, nil
}

func (resolver *HTTPFaviconResolver) lookupFavicon(ctx context.Context, baseURL *url.URL) (string, error) {
	root := &url.URL{
		Scheme: baseURL.Scheme,
		Host:   baseURL.Host,
	}

	if iconURL, err := resolver.fetchDefaultFavicon(ctx, root); err == nil && iconURL != "" {
		return iconURL, nil
	}

	return resolver.fetchHTMLDeclaredFavicon(ctx, root)
}

func (resolver *HTTPFaviconResolver) fetchDefaultFavicon(ctx context.Context, root *url.URL) (string, error) {
	candidate := root.ResolveReference(&url.URL{Path: "/favicon.ico"})
	ok, err := resolver.fetchIcon(ctx, candidate.String())
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	return candidate.String(), nil
}

func (resolver *HTTPFaviconResolver) fetchHTMLDeclaredFavicon(ctx context.Context, root *url.URL) (string, error) {
	pageURL := root.ResolveReference(&url.URL{Path: "/"})
	body, err := resolver.get(ctx, pageURL.String(), resolver.maxHTMLBytes)
	if err != nil {
		return "", err
	}
	if body == nil {
		return "", nil
	}
	defer body.Close()

	document, parseErr := html.Parse(body)
	if parseErr != nil {
		return "", parseErr
	}

	candidates := findFaviconCandidates(document)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(candidate), "data:") {
			return candidate, nil
		}
		absoluteURL, resolveErr := resolver.absoluteURL(root, candidate)
		if resolveErr != nil {
			continue
		}
		ok, fetchErr := resolver.fetchIcon(ctx, absoluteURL)
		if fetchErr != nil {
			if resolver.logger != nil {
				resolver.logger.Debug(
					"favicon_candidate_fetch_failed",
					zap.String("candidate", absoluteURL),
					zap.Error(fetchErr),
				)
			}
			continue
		}
		if ok {
			return absoluteURL, nil
		}
	}

	return "", nil
}

func (resolver *HTTPFaviconResolver) fetchIcon(ctx context.Context, iconURL string) (bool, error) {
	response, err := resolver.doRequest(ctx, http.MethodGet, iconURL)
	if err != nil {
		return false, err
	}
	if response == nil {
		return false, nil
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return false, nil
	}

	_, copyErr := io.Copy(io.Discard, io.LimitReader(response.Body, resolver.maxIconBytes))
	if copyErr != nil {
		return false, copyErr
	}

	contentType := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type")))
	if contentType == "" {
		return true, nil
	}
	if strings.HasPrefix(contentType, "image/") || strings.Contains(contentType, "icon") || strings.Contains(contentType, "svg") {
		return true, nil
	}
	return false, nil
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
