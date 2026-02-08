package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	testConfirmationSiteID      = "confirm-site-id"
	testConfirmationSubscriber  = "subscriber-id"
	testConfirmationEmail       = "subscriber@example.com"
	testConfirmationSiteName    = "Confirmation Site"
	testConfirmationBaseURL     = "https://confirm.example.com"
	testConfirmationTokenSecret = "confirm-secret"
)

type stubEmailSender struct {
	sendError   error
	recipient   string
	subjectLine string
	messageBody string
}

func (sender *stubEmailSender) SendEmail(_ context.Context, recipient string, subject string, message string) error {
	sender.recipient = recipient
	sender.subjectLine = subject
	sender.messageBody = message
	return sender.sendError
}

type eventRecord struct {
	eventType string
	status    string
	message   string
}

func TestSendSubscriptionConfirmationEmailSkipsWithoutSender(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}
	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, nil, testConfirmationBaseURL, testConfirmationTokenSecret, time.Hour, model.Site{}, model.Subscriber{})
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusSkipped, records[0].status)
	require.Contains(testingT, records[0].message, "email sender unavailable")
}

func TestSendSubscriptionConfirmationEmailSkipsWhenConfigMissing(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, &stubEmailSender{}, "", testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusSkipped, records[0].status)
	require.Contains(testingT, records[0].message, "confirmation email not configured")
}

func TestSendSubscriptionConfirmationEmailSkipsWhenSubscriberNotPending(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusConfirmed}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, &stubEmailSender{}, testConfirmationBaseURL, testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusSkipped, records[0].status)
	require.Contains(testingT, records[0].message, "subscriber not pending")
}

func TestSendSubscriptionConfirmationEmailSkipsWhenSubscriberMissingFields(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: "", SiteID: testConfirmationSiteID, Email: "", Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, &stubEmailSender{}, testConfirmationBaseURL, testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusSkipped, records[0].status)
	require.Contains(testingT, records[0].message, "subscriber missing fields")
}

func TestSendSubscriptionConfirmationEmailRecordsTokenError(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, &stubEmailSender{}, testConfirmationBaseURL, testConfirmationTokenSecret, 0, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusError, records[0].status)
	require.Contains(testingT, records[0].message, "confirmation token failed")
}

func TestSendSubscriptionConfirmationEmailRecordsURLFailure(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, &stubEmailSender{}, "http://[::1", testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusError, records[0].status)
	require.Contains(testingT, records[0].message, "confirmation url failed")
}

func TestSendSubscriptionConfirmationEmailSendsMessage(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	sender := &stubEmailSender{}
	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, sender, testConfirmationBaseURL, testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusSuccess, records[0].status)
	require.Equal(testingT, testConfirmationEmail, sender.recipient)
	require.Contains(testingT, sender.subjectLine, testConfirmationSiteName)
	require.Contains(testingT, sender.messageBody, testConfirmationBaseURL)
}

func TestSendSubscriptionConfirmationEmailRecordsSendFailure(testingT *testing.T) {
	var records []eventRecord
	recordEvent := func(_ model.Site, _ model.Subscriber, eventType string, status string, message string) {
		records = append(records, eventRecord{eventType: eventType, status: status, message: message})
	}

	sendErr := errors.New("send failed")
	sender := &stubEmailSender{sendError: sendErr}
	site := model.Site{ID: testConfirmationSiteID, Name: testConfirmationSiteName}
	subscriber := model.Subscriber{ID: testConfirmationSubscriber, SiteID: testConfirmationSiteID, Email: testConfirmationEmail, Status: model.SubscriberStatusPending}

	sendSubscriptionConfirmationEmail(context.Background(), zap.NewNop(), recordEvent, sender, testConfirmationBaseURL, testConfirmationTokenSecret, time.Hour, site, subscriber)
	require.Len(testingT, records, 1)
	require.Equal(testingT, subscriptionEventStatusError, records[0].status)
	require.Contains(testingT, records[0].message, "confirmation email failed")
}
