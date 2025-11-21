package httpapi

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// FeedbackNotifier dispatches notifications related to feedback submissions.
type FeedbackNotifier interface {
	NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error)
}

type noopFeedbackNotifier struct{}

func (noopFeedbackNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	return model.FeedbackDeliveryNone, nil
}

func resolveFeedbackNotifier(notifier FeedbackNotifier) FeedbackNotifier {
	if notifier == nil {
		return noopFeedbackNotifier{}
	}
	return notifier
}

func applyFeedbackNotification(ctx context.Context, database *gorm.DB, logger *zap.Logger, notifier FeedbackNotifier, site model.Site, feedback *model.Feedback) {
	if feedback == nil {
		return
	}
	if notifier == nil {
		return
	}
	delivery, notifyErr := notifier.NotifyFeedback(ctx, site, *feedback)
	if notifyErr != nil {
		if logger != nil {
			logger.Warn("feedback_notification_failed", zap.Error(notifyErr), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		}
		delivery = model.FeedbackDeliveryNone
	}

	if delivery != model.FeedbackDeliveryMailed && delivery != model.FeedbackDeliveryTexted {
		delivery = model.FeedbackDeliveryNone
	}
	if delivery == feedback.Delivery {
		return
	}
	if database == nil {
		return
	}
	feedback.Delivery = delivery
	updateErr := database.Save(feedback).Error
	if updateErr != nil {
		if logger != nil {
			logger.Warn("update_feedback_delivery_failed", zap.Error(updateErr), zap.String("feedback_id", feedback.ID))
		}
	}
}
