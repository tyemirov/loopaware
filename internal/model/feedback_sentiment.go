package model

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// FeedbackSentimentSad identifies negative widget sentiment.
	FeedbackSentimentSad = "sad"
	// FeedbackSentimentNeutral identifies neutral widget sentiment.
	FeedbackSentimentNeutral = "neutral"
	// FeedbackSentimentHappy identifies positive widget sentiment.
	FeedbackSentimentHappy = "happy"
)

var (
	// ErrInvalidFeedbackSentiment indicates the provided sentiment is not supported.
	ErrInvalidFeedbackSentiment = errors.New("invalid_feedback_sentiment")
)

// NormalizeFeedbackSentiment returns the canonical stored feedback sentiment value.
func NormalizeFeedbackSentiment(rawInput string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(rawInput))
	if normalized == "" {
		return "", nil
	}

	switch normalized {
	case FeedbackSentimentSad, FeedbackSentimentNeutral, FeedbackSentimentHappy:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidFeedbackSentiment, normalized)
	}
}
