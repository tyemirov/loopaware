package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testTokenSecretValue  = "token-secret"
	testTokenSubscriberID = "subscriber-id"
	testTokenSiteID       = "site-id"
	testTokenEmail        = "user@example.com"
)

func buildSubscriptionToken(testingT *testing.T, payload subscriptionConfirmationTokenPayload, secret string) string {
	testingT.Helper()
	encodedPayload, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)
	segment := base64.RawURLEncoding.EncodeToString(encodedPayload)
	signature := signSubscriptionConfirmationToken(secret, segment)
	return segment + subscriptionConfirmationTokenSeparator + signature
}

func TestParseSubscriptionConfirmationTokenSuccess(testingT *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: "subscriber-id",
		SiteID:       "site-id",
		Email:        "USER@example.com",
		ExpiresAt:    now.Add(time.Hour).Unix(),
	}
	token := buildSubscriptionToken(testingT, payload, testTokenSecretValue)

	parsed, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, now)
	require.NoError(testingT, parseErr)
	require.Equal(testingT, "subscriber-id", parsed.SubscriberID)
	require.Equal(testingT, "site-id", parsed.SiteID)
	require.Equal(testingT, "user@example.com", parsed.Email)
}

func TestParseSubscriptionConfirmationTokenRejectsMissingSecret(testingT *testing.T) {
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), "   ", "token", time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsMalformedToken(testingT *testing.T) {
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, "no-separator", time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsMissingSegments(testingT *testing.T) {
	token := "." + signSubscriptionConfirmationToken(testTokenSecretValue, "")
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsSignatureMismatch(testingT *testing.T) {
	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: "subscriber-id",
		SiteID:       "site-id",
		Email:        "user@example.com",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
	token := buildSubscriptionToken(testingT, payload, testTokenSecretValue)
	token = token + "extra"

	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsInvalidBase64(testingT *testing.T) {
	encodedSegment := "invalid%%payload"
	signature := signSubscriptionConfirmationToken(testTokenSecretValue, encodedSegment)
	token := encodedSegment + subscriptionConfirmationTokenSeparator + signature

	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsInvalidJSON(testingT *testing.T) {
	segment := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	signature := signSubscriptionConfirmationToken(testTokenSecretValue, segment)
	token := segment + subscriptionConfirmationTokenSeparator + signature

	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsMissingFields(testingT *testing.T) {
	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: "  ",
		SiteID:       "site-id",
		Email:        "user@example.com",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
	token := buildSubscriptionToken(testingT, payload, testTokenSecretValue)
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsInvalidExpiry(testingT *testing.T) {
	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: "subscriber-id",
		SiteID:       "site-id",
		Email:        "user@example.com",
		ExpiresAt:    0,
	}
	token := buildSubscriptionToken(testingT, payload, testTokenSecretValue)
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, time.Now())
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestParseSubscriptionConfirmationTokenRejectsExpiredToken(testingT *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	payload := subscriptionConfirmationTokenPayload{
		SubscriberID: "subscriber-id",
		SiteID:       "site-id",
		Email:        "user@example.com",
		ExpiresAt:    now.Add(-time.Minute).Unix(),
	}
	token := buildSubscriptionToken(testingT, payload, testTokenSecretValue)
	_, parseErr := parseSubscriptionConfirmationToken(context.Background(), testTokenSecretValue, token, now)
	require.ErrorIs(testingT, parseErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestBuildSubscriptionConfirmationTokenSuccess(testingT *testing.T) {
	now := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	token, tokenErr := buildSubscriptionConfirmationToken(testTokenSecretValue, testTokenSubscriberID, testTokenSiteID, testTokenEmail, now, time.Hour)
	require.NoError(testingT, tokenErr)
	require.Contains(testingT, token, subscriptionConfirmationTokenSeparator)
}

func TestBuildSubscriptionConfirmationTokenRejectsMissingSecret(testingT *testing.T) {
	_, tokenErr := buildSubscriptionConfirmationToken("   ", testTokenSubscriberID, testTokenSiteID, testTokenEmail, time.Now().UTC(), time.Hour)
	require.ErrorIs(testingT, tokenErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestBuildSubscriptionConfirmationTokenRejectsMissingFields(testingT *testing.T) {
	_, tokenErr := buildSubscriptionConfirmationToken(testTokenSecretValue, "", testTokenSiteID, testTokenEmail, time.Now().UTC(), time.Hour)
	require.ErrorIs(testingT, tokenErr, ErrInvalidSubscriptionConfirmationToken)
}

func TestBuildSubscriptionConfirmationTokenRejectsInvalidTTL(testingT *testing.T) {
	_, tokenErr := buildSubscriptionConfirmationToken(testTokenSecretValue, testTokenSubscriberID, testTokenSiteID, testTokenEmail, time.Now().UTC(), 0)
	require.ErrorIs(testingT, tokenErr, ErrInvalidSubscriptionConfirmationToken)
}
