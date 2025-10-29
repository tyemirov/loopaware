package httpapi

import (
	"context"

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
