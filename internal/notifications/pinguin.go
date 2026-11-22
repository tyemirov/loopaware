package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	pingwinclient "github.com/tyemirov/pingwin/pkg/client"
	pinguinpb "github.com/tyemirov/pingwin/pkg/grpcapi"
	"go.uber.org/zap"
)

// PinguinConfig captures connection settings for the Pinguin notification service.
type PinguinConfig struct {
	Address          string
	APIKey           string
	OperationTimeout time.Duration
}

// PinguinNotifier dispatches notifications through the Pinguin gRPC service.
type PinguinNotifier struct {
	logger           *zap.Logger
	client           *pingwinclient.NotificationClient
	operationTimeout time.Duration
}

var phoneNumberExpression = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// NewPinguinNotifier validates configuration and creates a lazy official client.
// The SDK requires a connection-timeout setting but does not enforce it; sends
// are bounded by operationTimeout, including connection establishment.
func NewPinguinNotifier(logger *zap.Logger, cfg PinguinConfig) (*PinguinNotifier, error) {
	if cfg.Address == "" {
		return nil, errors.New("pinguin address is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("pinguin api key is required")
	}
	if cfg.OperationTimeout <= 0 {
		cfg.OperationTimeout = 30 * time.Second
	}

	operationTimeoutSeconds := int((cfg.OperationTimeout + time.Second - 1) / time.Second)
	settings, settingsError := pingwinclient.NewSettings(cfg.Address, cfg.APIKey, 5, operationTimeoutSeconds)
	if settingsError != nil {
		return nil, fmt.Errorf("configure pingwin: %w", settingsError)
	}
	client, clientError := pingwinclient.NewNotificationClient(slog.Default(), settings)
	if clientError != nil {
		return nil, fmt.Errorf("connect to pingwin: %w", clientError)
	}

	return &PinguinNotifier{
		logger:           logger,
		client:           client,
		operationTimeout: cfg.OperationTimeout,
	}, nil
}

// Close releases the underlying gRPC connection.
func (notifier *PinguinNotifier) Close() error {
	if notifier == nil || notifier.client == nil {
		return nil
	}
	return notifier.client.Close()
}

// NotifyFeedback sends a notification describing the feedback submission.
func (notifier *PinguinNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	if notifier == nil || notifier.client == nil {
		return model.FeedbackDeliveryNone, errors.New("pinguin notifier not initialized")
	}

	notificationType, recipient, delivery, deliveryErr := determineRecipient(site.OwnerEmail)
	if deliveryErr != nil {
		return model.FeedbackDeliveryNone, deliveryErr
	}

	subject := fmt.Sprintf("New feedback for %s", strings.TrimSpace(site.Name))
	messageBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(messageBuilder, "A new feedback entry was submitted for %s.\n\n", strings.TrimSpace(site.Name))
	if feedback.Contact != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Contact: %s\n", strings.TrimSpace(feedback.Contact))
	}
	if feedback.Sentiment != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Sentiment: %s\n", formatFeedbackSentimentLabel(feedback.Sentiment))
	}
	if strings.TrimSpace(feedback.SourceURL) != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Source: %s\n", strings.TrimSpace(feedback.SourceURL))
	}
	if strings.TrimSpace(feedback.Message) != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Message:\n%s\n", strings.TrimSpace(feedback.Message))
	}

	request := &pinguinpb.NotificationRequest{
		NotificationType: notificationType,
		Recipient:        recipient,
		Subject:          subject,
		Message:          messageBuilder.String(),
	}

	callCtx, cancel := context.WithTimeout(ctx, notifier.operationTimeout)
	defer cancel()

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, sendErr
	}

	notifier.recordNotificationResponse(response)
	if response.GetStatus() == pinguinpb.Status_ERRORED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, err
	}

	return delivery, nil
}

func formatFeedbackSentimentLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case model.FeedbackSentimentSad:
		return "Sad"
	case model.FeedbackSentimentNeutral:
		return "Neutral"
	case model.FeedbackSentimentHappy:
		return "Happy"
	default:
		return strings.TrimSpace(value)
	}
}

// NotifySubscription sends a notification describing the subscription.
func (notifier *PinguinNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	if notifier == nil || notifier.client == nil {
		return errors.New("pinguin notifier not initialized")
	}

	notificationType, recipient, delivery, deliveryErr := determineRecipient(site.OwnerEmail)
	if deliveryErr != nil {
		return deliveryErr
	}

	subject := fmt.Sprintf("New subscriber for %s", strings.TrimSpace(site.Name))
	messageBuilder := &strings.Builder{}
	_, _ = fmt.Fprintf(messageBuilder, "A new subscriber joined %s.\n\n", strings.TrimSpace(site.Name))
	if subscriber.Email != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Email: %s\n", strings.TrimSpace(subscriber.Email))
	}
	if subscriber.Name != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Name: %s\n", strings.TrimSpace(subscriber.Name))
	}
	if subscriber.SourceURL != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Source: %s\n", strings.TrimSpace(subscriber.SourceURL))
	}

	request := &pinguinpb.NotificationRequest{
		NotificationType: notificationType,
		Recipient:        recipient,
		Subject:          subject,
		Message:          messageBuilder.String(),
	}

	callCtx, cancel := context.WithTimeout(ctx, notifier.operationTimeout)
	defer cancel()

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		return sendErr
	}

	notifier.recordNotificationResponse(response)
	if response.GetStatus() == pinguinpb.Status_ERRORED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		return err
	}

	if delivery == model.FeedbackDeliveryNone {
		return nil
	}

	return nil
}

// SendEmail dispatches an email notification through the Pinguin service.
func (notifier *PinguinNotifier) SendEmail(ctx context.Context, recipient string, subject string, message string) error {
	if notifier == nil || notifier.client == nil {
		return errors.New("pinguin notifier not initialized")
	}

	normalizedRecipient := strings.TrimSpace(recipient)
	if normalizedRecipient == "" {
		return errors.New("recipient is required")
	}
	if !strings.Contains(normalizedRecipient, "@") {
		return errors.New("recipient must be an email address")
	}

	request := &pinguinpb.NotificationRequest{
		NotificationType: pinguinpb.NotificationType_EMAIL,
		Recipient:        normalizedRecipient,
		Subject:          strings.TrimSpace(subject),
		Message:          message,
	}

	callCtx, cancel := context.WithTimeout(ctx, notifier.operationTimeout)
	defer cancel()

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr))
		return sendErr
	}

	notifier.recordNotificationResponse(response)
	if response.GetStatus() == pinguinpb.Status_ERRORED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err))
		return err
	}

	return nil
}

func (notifier *PinguinNotifier) recordNotificationResponse(response *pinguinpb.NotificationResponse) {
	notifier.logger.Info("pingwin_notification_accepted", zap.String("notification_id", response.GetNotificationId()), zap.String("status", response.GetStatus().String()))
}

func determineRecipient(contact string) (pinguinpb.NotificationType, string, string, error) {
	trimmed := strings.TrimSpace(contact)
	if trimmed == "" {
		return pinguinpb.NotificationType_EMAIL, "", model.FeedbackDeliveryNone, errors.New("owner contact is empty")
	}

	if strings.Contains(trimmed, "@") {
		return pinguinpb.NotificationType_EMAIL, trimmed, model.FeedbackDeliveryMailed, nil
	}

	if phoneNumberExpression.MatchString(trimmed) {
		return pinguinpb.NotificationType_SMS, trimmed, model.FeedbackDeliveryTexted, nil
	}

	return pinguinpb.NotificationType_EMAIL, "", model.FeedbackDeliveryNone, fmt.Errorf("unrecognized owner contact: %s", trimmed)
}
