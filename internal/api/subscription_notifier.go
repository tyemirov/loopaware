package api

import (
	"context"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// SubscriptionNotifier dispatches notifications related to new subscriptions.
type SubscriptionNotifier interface {
	NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error
}

type noopSubscriptionNotifier struct{}

func (noopSubscriptionNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	return nil
}

func resolveSubscriptionNotifier(notifier SubscriptionNotifier) SubscriptionNotifier {
	if notifier == nil {
		return noopSubscriptionNotifier{}
	}
	return notifier
}
