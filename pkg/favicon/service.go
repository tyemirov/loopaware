package favicon

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"
)

// Site captures favicon-related fields required for collection decisions.
type Site struct {
	FaviconData        []byte
	FaviconContentType string
	FaviconFetchedAt   time.Time
}

// CollectionResult describes persistence changes and notification intent.
type CollectionResult struct {
	Updates        map[string]any
	ShouldNotify   bool
	EventTimestamp time.Time
}

// Service orchestrates favicon retrieval for persistence layers.
type Service struct {
	resolver Resolver
}

// NewService constructs a favicon collection service.
func NewService(resolver Resolver) *Service {
	return &Service{resolver: resolver}
}

// Collect resolves favicon assets and prepares persistence updates.
func (service *Service) Collect(ctx context.Context, site Site, allowedOrigin string, notify bool, timestamp time.Time) (CollectionResult, error) {
	if service == nil || service.resolver == nil {
		return CollectionResult{}, errors.New("favicon resolver is not configured")
	}

	normalizedOrigin := strings.TrimSpace(allowedOrigin)
	if normalizedOrigin == "" {
		return CollectionResult{}, nil
	}

	updates := map[string]any{
		"favicon_origin":          normalizedOrigin,
		"favicon_last_attempt_at": timestamp,
	}
	result := CollectionResult{Updates: updates}

	asset, resolveErr := service.resolver.ResolveAsset(ctx, normalizedOrigin)
	if resolveErr != nil {
		return result, resolveErr
	}

	shouldNotify := false
	if asset != nil && len(asset.Data) > 0 {
		contentTypesEqual := strings.EqualFold(strings.TrimSpace(site.FaviconContentType), strings.TrimSpace(asset.ContentType))
		if site.FaviconFetchedAt.IsZero() || !bytes.Equal(site.FaviconData, asset.Data) || !contentTypesEqual {
			result.Updates["favicon_data"] = asset.Data
			result.Updates["favicon_content_type"] = asset.ContentType
			result.Updates["favicon_fetched_at"] = timestamp
			shouldNotify = true
		} else {
			result.Updates["favicon_fetched_at"] = timestamp
		}
		if notify {
			shouldNotify = true
		}
	}

	if shouldNotify {
		result.ShouldNotify = true
		result.EventTimestamp = timestamp
	}

	return result, nil
}
