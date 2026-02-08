package api

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
	intendedDelivery := model.FeedbackDeliveryMailed
	payload := *feedback
	payload.Delivery = intendedDelivery

	delivery, notifyErr := notifier.NotifyFeedback(ctx, site, payload)
	if notifyErr != nil {
		if logger != nil {
			logger.Warn("feedback_notification_failed", zap.Error(notifyErr), zap.String("site_id", site.ID), zap.String("feedback_id", feedback.ID))
		}
		intendedDelivery = model.FeedbackDeliveryNone
	}

	if (delivery == model.FeedbackDeliveryMailed || delivery == model.FeedbackDeliveryTexted) && notifyErr == nil {
		intendedDelivery = delivery
	} else if delivery != model.FeedbackDeliveryMailed && delivery != model.FeedbackDeliveryTexted {
		intendedDelivery = model.FeedbackDeliveryNone
	}

	if intendedDelivery == feedback.Delivery {
		return
	}
	if database == nil {
		return
	}
	updateErr := database.Exec("UPDATE feedbacks SET delivery = ? WHERE id = ?", intendedDelivery, feedback.ID).Error
	if updateErr != nil {
		if logger != nil {
			logger.Warn("update_feedback_delivery_failed", zap.Error(updateErr), zap.String("feedback_id", feedback.ID))
		}
		return
	}
	feedback.Delivery = intendedDelivery
}
