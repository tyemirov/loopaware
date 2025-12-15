package notifications

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/notifications/pinguinpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// PinguinConfig captures connection settings for the Pinguin notification service.
type PinguinConfig struct {
	Address           string
	AuthToken         string
	TenantID          string
	ConnectionTimeout time.Duration
	OperationTimeout  time.Duration
}

// PinguinNotifier dispatches notifications through the Pinguin gRPC service.
type PinguinNotifier struct {
	logger            *zap.Logger
	conn              *grpc.ClientConn
	client            pinguinpb.NotificationServiceClient
	authToken         string
	tenantID          string
	operationTimeout  time.Duration
	connectionTimeout time.Duration
}

var phoneNumberExpression = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// NewPinguinNotifier creates a notifier backed by the Pinguin gRPC client.
func NewPinguinNotifier(logger *zap.Logger, cfg PinguinConfig) (*PinguinNotifier, error) {
	if cfg.Address == "" {
		return nil, errors.New("pinguin address is required")
	}
	if cfg.AuthToken == "" {
		return nil, errors.New("pinguin auth token is required")
	}
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	if cfg.TenantID == "" {
		return nil, errors.New("pinguin tenant id is required")
	}
	if cfg.ConnectionTimeout <= 0 {
		cfg.ConnectionTimeout = 5 * time.Second
	}
	if cfg.OperationTimeout <= 0 {
		cfg.OperationTimeout = 30 * time.Second
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	conn, dialErr := grpc.DialContext(
		dialCtx,
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if dialErr != nil {
		return nil, fmt.Errorf("connect to pinguin: %w", dialErr)
	}

	return &PinguinNotifier{
		logger:            logger,
		conn:              conn,
		client:            pinguinpb.NewNotificationServiceClient(conn),
		authToken:         cfg.AuthToken,
		tenantID:          cfg.TenantID,
		operationTimeout:  cfg.OperationTimeout,
		connectionTimeout: cfg.ConnectionTimeout,
	}, nil
}

// Close releases the underlying gRPC connection.
func (notifier *PinguinNotifier) Close() error {
	if notifier == nil || notifier.conn == nil {
		return nil
	}
	return notifier.conn.Close()
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
	_, _ = fmt.Fprintf(messageBuilder, "A new feedback message was submitted for %s.\n\n", strings.TrimSpace(site.Name))
	if feedback.Contact != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Contact: %s\n", strings.TrimSpace(feedback.Contact))
	}
	_, _ = fmt.Fprintf(messageBuilder, "Message:\n%s\n", strings.TrimSpace(feedback.Message))

	request := &pinguinpb.NotificationRequest{
		NotificationType: notificationType,
		Recipient:        recipient,
		Subject:          subject,
		Message:          messageBuilder.String(),
	}

	callCtx, cancel := context.WithTimeout(ctx, notifier.operationTimeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+notifier.authToken, "x-tenant-id", notifier.tenantID)

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, sendErr
	}

	if response.GetStatus() == pinguinpb.Status_FAILED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, err
	}

	return delivery, nil
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
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+notifier.authToken, "x-tenant-id", notifier.tenantID)

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("subscriber_id", subscriber.ID))
		return sendErr
	}

	if response.GetStatus() == pinguinpb.Status_FAILED {
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
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+notifier.authToken, "x-tenant-id", notifier.tenantID)

	response, sendErr := notifier.client.SendNotification(callCtx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr))
		return sendErr
	}

	if response.GetStatus() == pinguinpb.Status_FAILED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err))
		return err
	}

	return nil
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
