package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidSubscriptionConfirmationToken = errors.New("invalid_subscription_confirmation_token")

const (
	subscriptionConfirmationTokenSeparator = "."
)

type subscriptionConfirmationTokenPayload struct {
	SubscriberID string `json:"subscriber_id"`
	SiteID       string `json:"site_id"`
	Email        string `json:"email"`
	ExpiresAt    int64  `json:"exp"`
}

func buildSubscriptionConfirmationToken(secret string, subscriberID string, siteID string, email string, now time.Time, ttl time.Duration) (string, error) {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return "", fmt.Errorf("%w: missing secret", ErrInvalidSubscriptionConfirmationToken)
	}
	normalizedSubscriberID := strings.TrimSpace(subscriberID)
	normalizedSiteID := strings.TrimSpace(siteID)
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	if normalizedSubscriberID == "" || normalizedSiteID == "" || normalizedEmail == "" {
		return "", fmt.Errorf("%w: missing fields", ErrInvalidSubscriptionConfirmationToken)
	}
	if ttl <= 0 {
		return "", fmt.Errorf("%w: invalid ttl", ErrInvalidSubscriptionConfirmationToken)
	}

	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: normalizedSubscriberID,
		SiteID:       normalizedSiteID,
		Email:        normalizedEmail,
		ExpiresAt:    now.Add(ttl).Unix(),
	}
	encodedPayload, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", fmt.Errorf("%w: encode payload: %v", ErrInvalidSubscriptionConfirmationToken, marshalErr)
	}

	encodedPayloadSegment := base64.RawURLEncoding.EncodeToString(encodedPayload)
	signature := signSubscriptionConfirmationToken(trimmedSecret, encodedPayloadSegment)
	return encodedPayloadSegment + subscriptionConfirmationTokenSeparator + signature, nil
}

func parseSubscriptionConfirmationToken(_ context.Context, secret string, rawToken string, now time.Time) (subscriptionConfirmationTokenPayload, error) {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: missing secret", ErrInvalidSubscriptionConfirmationToken)
	}

	tokenValue := strings.TrimSpace(rawToken)
	parts := strings.Split(tokenValue, subscriptionConfirmationTokenSeparator)
	if len(parts) != 2 {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: malformed", ErrInvalidSubscriptionConfirmationToken)
	}

	encodedPayloadSegment := strings.TrimSpace(parts[0])
	signatureSegment := strings.TrimSpace(parts[1])
	if encodedPayloadSegment == "" || signatureSegment == "" {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: missing segments", ErrInvalidSubscriptionConfirmationToken)
	}

	expectedSignature := signSubscriptionConfirmationToken(trimmedSecret, encodedPayloadSegment)
	if !hmac.Equal([]byte(signatureSegment), []byte(expectedSignature)) {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: signature mismatch", ErrInvalidSubscriptionConfirmationToken)
	}

	decodedPayload, decodeErr := base64.RawURLEncoding.DecodeString(encodedPayloadSegment)
	if decodeErr != nil {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: decode payload", ErrInvalidSubscriptionConfirmationToken)
	}

	var payload subscriptionConfirmationTokenPayload
	if unmarshalErr := json.Unmarshal(decodedPayload, &payload); unmarshalErr != nil {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: unmarshal payload", ErrInvalidSubscriptionConfirmationToken)
	}

	payload.SubscriberID = strings.TrimSpace(payload.SubscriberID)
	payload.SiteID = strings.TrimSpace(payload.SiteID)
	payload.Email = strings.TrimSpace(strings.ToLower(payload.Email))
	if payload.SubscriberID == "" || payload.SiteID == "" || payload.Email == "" {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: missing fields", ErrInvalidSubscriptionConfirmationToken)
	}

	if payload.ExpiresAt <= 0 {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: invalid exp", ErrInvalidSubscriptionConfirmationToken)
	}
	if now.Unix() > payload.ExpiresAt {
		return subscriptionConfirmationTokenPayload{}, fmt.Errorf("%w: expired", ErrInvalidSubscriptionConfirmationToken)
	}

	return payload, nil
}

func signSubscriptionConfirmationToken(secret string, encodedPayloadSegment string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encodedPayloadSegment))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
