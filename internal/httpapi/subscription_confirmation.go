package httpapi

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type subscriptionTestEventRecorder func(site model.Site, subscriber model.Subscriber, eventType, status, message string)

func sendSubscriptionConfirmationEmail(ctx context.Context, logger *zap.Logger, recordEvent subscriptionTestEventRecorder, emailSender EmailSender, publicBaseURL string, tokenSecret string, tokenTTL time.Duration, site model.Site, subscriber model.Subscriber) {
	if emailSender == nil {
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusSkipped, "email sender unavailable")
		}
		return
	}
	if strings.TrimSpace(publicBaseURL) == "" || strings.TrimSpace(tokenSecret) == "" {
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusSkipped, "confirmation email not configured")
		}
		return
	}
	if subscriber.Status != model.SubscriberStatusPending {
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusSkipped, "subscriber not pending")
		}
		return
	}
	if strings.TrimSpace(subscriber.ID) == "" || strings.TrimSpace(subscriber.SiteID) == "" || strings.TrimSpace(subscriber.Email) == "" {
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusSkipped, "subscriber missing fields")
		}
		return
	}

	token, tokenErr := buildSubscriptionConfirmationToken(tokenSecret, subscriber.ID, subscriber.SiteID, subscriber.Email, time.Now().UTC(), tokenTTL)
	if tokenErr != nil {
		if logger != nil {
			logger.Warn("subscription_confirmation_token_failed", zap.Error(tokenErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		}
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusError, "confirmation token failed")
		}
		return
	}

	confirmationURL, urlErr := url.Parse(strings.TrimRight(strings.TrimSpace(publicBaseURL), "/") + "/subscriptions/confirm")
	if urlErr != nil {
		if logger != nil {
			logger.Warn("subscription_confirmation_url_failed", zap.Error(urlErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		}
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusError, "confirmation url failed")
		}
		return
	}
	query := confirmationURL.Query()
	query.Set("token", token)
	confirmationURL.RawQuery = query.Encode()

	siteName := strings.TrimSpace(site.Name)
	if siteName == "" {
		siteName = "LoopAware"
	}
	subject := fmt.Sprintf("Confirm your subscription to %s", siteName)
	messageBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(messageBuilder, "Thanks for subscribing to %s.\n\n", siteName)
	_, _ = fmt.Fprintf(messageBuilder, "Confirm your subscription:\n%s\n\n", confirmationURL.String())
	_, _ = fmt.Fprintf(messageBuilder, "If you did not request this, you can ignore this email.\n")

	sendErr := emailSender.SendEmail(ctx, subscriber.Email, subject, messageBuilder.String())
	if sendErr != nil {
		if logger != nil {
			logger.Warn("subscription_confirmation_email_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		}
		if recordEvent != nil {
			recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusError, "confirmation email failed")
		}
		return
	}
	if recordEvent != nil {
		recordEvent(site, subscriber, subscriptionEventTypeConfirmation, subscriptionEventStatusSuccess, "")
	}
}
