package notifications

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/temirov/pinguin/pkg/client"
	"github.com/temirov/pinguin/pkg/config"
	"github.com/temirov/pinguin/pkg/grpcapi"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// PinguinConfig captures connection settings for the Pinguin notification service.
type PinguinConfig struct {
	Address           string
	AuthToken         string
	ConnectionTimeout time.Duration
	OperationTimeout  time.Duration
}

// PinguinNotifier dispatches notifications through the Pinguin gRPC service.
type PinguinNotifier struct {
	logger *zap.Logger
	client *client.NotificationClient
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
	if cfg.ConnectionTimeout <= 0 {
		cfg.ConnectionTimeout = 5 * time.Second
	}
	if cfg.OperationTimeout <= 0 {
		cfg.OperationTimeout = 30 * time.Second
	}

	previousAddress, hadPreviousAddress := os.LookupEnv("GRPC_SERVER_ADDR")
	if setErr := os.Setenv("GRPC_SERVER_ADDR", cfg.Address); setErr != nil {
		return nil, fmt.Errorf("set grpc server address: %w", setErr)
	}
	defer func() {
		if hadPreviousAddress {
			_ = os.Setenv("GRPC_SERVER_ADDR", previousAddress)
			return
		}
		_ = os.Unsetenv("GRPC_SERVER_ADDR")
	}()

	pinguinConfig := config.Config{
		GRPCAuthToken:        cfg.AuthToken,
		ConnectionTimeoutSec: int(cfg.ConnectionTimeout / time.Second),
		OperationTimeoutSec:  int(cfg.OperationTimeout / time.Second),
	}

	slogLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	notifierClient, clientErr := client.NewNotificationClient(slogLogger, pinguinConfig)
	if clientErr != nil {
		return nil, fmt.Errorf("create pinguin client: %w", clientErr)
	}

	return &PinguinNotifier{
		logger: logger,
		client: notifierClient,
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
	_, _ = fmt.Fprintf(messageBuilder, "A new feedback message was submitted for %s.\n\n", strings.TrimSpace(site.Name))
	if feedback.Contact != "" {
		_, _ = fmt.Fprintf(messageBuilder, "Contact: %s\n", strings.TrimSpace(feedback.Contact))
	}
	_, _ = fmt.Fprintf(messageBuilder, "Message:\n%s\n", strings.TrimSpace(feedback.Message))

	request := &grpcapi.NotificationRequest{
		NotificationType: notificationType,
		Recipient:        recipient,
		Subject:          subject,
		Message:          messageBuilder.String(),
	}

	response, sendErr := notifier.client.SendNotification(ctx, request)
	if sendErr != nil {
		notifier.logger.Warn("pinguin_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, sendErr
	}

	if response.GetStatus() == grpcapi.Status_FAILED {
		err := fmt.Errorf("notification failed with status %s", response.GetStatus().String())
		notifier.logger.Warn("pinguin_send_failed_status", zap.Error(err), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		return model.FeedbackDeliveryNone, err
	}

	return delivery, nil
}

func determineRecipient(contact string) (grpcapi.NotificationType, string, string, error) {
	trimmed := strings.TrimSpace(contact)
	if trimmed == "" {
		return grpcapi.NotificationType_EMAIL, "", model.FeedbackDeliveryNone, errors.New("owner contact is empty")
	}

	if strings.Contains(trimmed, "@") {
		return grpcapi.NotificationType_EMAIL, trimmed, model.FeedbackDeliveryMailed, nil
	}

	if phoneNumberExpression.MatchString(trimmed) {
		return grpcapi.NotificationType_SMS, trimmed, model.FeedbackDeliveryTexted, nil
	}

	return grpcapi.NotificationType_EMAIL, "", model.FeedbackDeliveryNone, fmt.Errorf("unrecognized owner contact: %s", trimmed)
}
