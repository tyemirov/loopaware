package notifications

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/notifications/pinguinpb"
)

const (
	testPinguinAddress       = "bufnet"
	testPinguinAuthToken     = "test-auth-token"
	testPinguinTenantID      = "test-tenant"
	testFeedbackSiteID       = "site-id"
	testFeedbackID           = "feedback-id"
	testFeedbackOwnerEmail   = "owner@example.com"
	testFeedbackSiteName     = "Example Site"
	testFeedbackMessage      = "Hello"
	testFeedbackContactEmail = "contact@example.com"
	testSubscriberID         = "subscriber-id"
	testSubscriberEmail      = "subscriber@example.com"
	testSubscriberName       = "Subscriber Name"
	testNotificationSubject  = "New feedback for"
	testNotificationAuthKey  = "authorization"
	testNotificationTenant   = "x-tenant-id"
	testBufferSize           = 1024 * 1024
	testSendFailureMessage   = "send failed"
	testEmailSubject         = "Subject"
	testEmailMessage         = "Message"
	testEmptyRecipient       = " "
	testDefaultConnTimeout   = 5 * time.Second
	testDefaultOpTimeout     = 30 * time.Second
)

type testNotificationService struct {
	pinguinpb.UnimplementedNotificationServiceServer
	responseStatus pinguinpb.Status
	responseError  error
	mutex          sync.Mutex
	lastRequest    *pinguinpb.NotificationRequest
	lastMetadata   metadata.MD
}

func (service *testNotificationService) SendNotification(requestContext context.Context, request *pinguinpb.NotificationRequest) (*pinguinpb.NotificationResponse, error) {
	service.mutex.Lock()
	service.lastRequest = request
	service.lastMetadata, _ = metadata.FromIncomingContext(requestContext)
	service.mutex.Unlock()

	if service.responseError != nil {
		return nil, service.responseError
	}

	return &pinguinpb.NotificationResponse{Status: service.responseStatus}, nil
}

func (service *testNotificationService) GetNotificationStatus(context.Context, *pinguinpb.GetNotificationStatusRequest) (*pinguinpb.NotificationResponse, error) {
	return &pinguinpb.NotificationResponse{Status: service.responseStatus}, nil
}

func (service *testNotificationService) recordedRequest() (*pinguinpb.NotificationRequest, metadata.MD) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.lastRequest, service.lastMetadata
}

func startNotificationServer(testingT *testing.T, service *testNotificationService) *bufconn.Listener {
	listener := bufconn.Listen(testBufferSize)
	grpcServer := grpc.NewServer()
	pinguinpb.RegisterNotificationServiceServer(grpcServer, service)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	testingT.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	return listener
}

func createPinguinDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(requestContext context.Context, _ string) (net.Conn, error) {
		return listener.DialContext(requestContext)
	}
}

func TestNewPinguinNotifierValidatesConfig(testingT *testing.T) {
	testCases := []struct {
		name        string
		config      PinguinConfig
		expectError string
	}{
		{
			name:        "missing address",
			config:      PinguinConfig{AuthToken: testPinguinAuthToken, TenantID: testPinguinTenantID},
			expectError: "pinguin address is required",
		},
		{
			name:        "missing auth token",
			config:      PinguinConfig{Address: "127.0.0.1:1", TenantID: testPinguinTenantID},
			expectError: "pinguin auth token is required",
		},
		{
			name:        "missing tenant id",
			config:      PinguinConfig{Address: "127.0.0.1:1", AuthToken: testPinguinAuthToken},
			expectError: "pinguin tenant id is required",
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			notifier, createErr := NewPinguinNotifier(zap.NewNop(), testCase.config)
			require.ErrorContains(testingT, createErr, testCase.expectError)
			require.Nil(testingT, notifier)
		})
	}
}

func TestNotifyFeedbackSendsEmailNotification(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_SENT}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	feedback := model.Feedback{
		ID:      testFeedbackID,
		Contact: testFeedbackContactEmail,
		Message: testFeedbackMessage,
	}

	delivery, notifyErr := notifier.NotifyFeedback(context.Background(), site, feedback)
	require.NoError(testingT, notifyErr)
	require.Equal(testingT, model.FeedbackDeliveryMailed, delivery)

	recordedRequest, recordedMetadata := service.recordedRequest()
	require.NotNil(testingT, recordedRequest)
	require.Equal(testingT, pinguinpb.NotificationType_EMAIL, recordedRequest.GetNotificationType())
	require.Equal(testingT, testFeedbackOwnerEmail, recordedRequest.GetRecipient())
	require.Contains(testingT, recordedRequest.GetSubject(), testNotificationSubject)
	require.Contains(testingT, recordedMetadata.Get(testNotificationAuthKey)[0], testPinguinAuthToken)
	require.Contains(testingT, recordedMetadata.Get(testNotificationTenant)[0], testPinguinTenantID)
}

func TestNotifyFeedbackReturnsErrorOnFailedStatus(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_FAILED}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	feedback := model.Feedback{
		ID:      testFeedbackID,
		Message: testFeedbackMessage,
	}

	delivery, notifyErr := notifier.NotifyFeedback(context.Background(), site, feedback)
	require.Error(testingT, notifyErr)
	require.Equal(testingT, model.FeedbackDeliveryNone, delivery)
}

func TestNotifyFeedbackReturnsErrorOnUnknownRecipient(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_SENT}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: "invalid-contact",
	}
	feedback := model.Feedback{
		ID:      testFeedbackID,
		Message: testFeedbackMessage,
	}

	delivery, notifyErr := notifier.NotifyFeedback(context.Background(), site, feedback)
	require.Error(testingT, notifyErr)
	require.Equal(testingT, model.FeedbackDeliveryNone, delivery)
}

func TestNotifySubscriptionSendsEmailNotification(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_SENT}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	subscriber := model.Subscriber{
		ID:     testSubscriberID,
		Email:  testSubscriberEmail,
		Name:   testSubscriberName,
		Status: model.SubscriberStatusPending,
	}

	notifyErr := notifier.NotifySubscription(context.Background(), site, subscriber)
	require.NoError(testingT, notifyErr)
}

func TestNotifySubscriptionReturnsErrorOnFailedStatus(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_FAILED}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	subscriber := model.Subscriber{
		ID:     testSubscriberID,
		Email:  testSubscriberEmail,
		Name:   testSubscriberName,
		Status: model.SubscriberStatusPending,
	}

	notifyErr := notifier.NotifySubscription(context.Background(), site, subscriber)
	require.Error(testingT, notifyErr)
}

func TestNotifySubscriptionReturnsErrorOnSendFailure(testingT *testing.T) {
	service := &testNotificationService{responseError: errors.New(testSendFailureMessage)}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	subscriber := model.Subscriber{
		ID:     testSubscriberID,
		Email:  testSubscriberEmail,
		Name:   testSubscriberName,
		Status: model.SubscriberStatusPending,
	}

	notifyErr := notifier.NotifySubscription(context.Background(), site, subscriber)
	require.Error(testingT, notifyErr)
}

func TestSendEmailValidatesRecipient(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_SENT}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	emptyErr := notifier.SendEmail(context.Background(), testEmptyRecipient, testEmailSubject, testEmailMessage)
	require.Error(testingT, emptyErr)

	invalidErr := notifier.SendEmail(context.Background(), "invalid", testEmailSubject, testEmailMessage)
	require.Error(testingT, invalidErr)

	sendErr := notifier.SendEmail(context.Background(), testFeedbackOwnerEmail, testEmailSubject, testEmailMessage)
	require.NoError(testingT, sendErr)
}

func TestSendEmailReturnsErrorOnFailedStatus(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_FAILED}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	sendErr := notifier.SendEmail(context.Background(), testFeedbackOwnerEmail, testEmailSubject, testEmailMessage)
	require.Error(testingT, sendErr)
}

func TestPinguinNotifierCloseHandlesNil(testingT *testing.T) {
	var notifier *PinguinNotifier
	require.NoError(testingT, notifier.Close())
}

func TestDetermineRecipientHandlesEmailAndPhone(testingT *testing.T) {
	notificationType, recipient, delivery, determineErr := determineRecipient(testFeedbackOwnerEmail)
	require.NoError(testingT, determineErr)
	require.Equal(testingT, pinguinpb.NotificationType_EMAIL, notificationType)
	require.Equal(testingT, testFeedbackOwnerEmail, recipient)
	require.Equal(testingT, model.FeedbackDeliveryMailed, delivery)

	phoneNumber := "+12345678901"
	notificationType, recipient, delivery, determineErr = determineRecipient(phoneNumber)
	require.NoError(testingT, determineErr)
	require.Equal(testingT, pinguinpb.NotificationType_SMS, notificationType)
	require.Equal(testingT, phoneNumber, recipient)
	require.Equal(testingT, model.FeedbackDeliveryTexted, delivery)

	_, _, _, determineErr = determineRecipient("")
	require.Error(testingT, determineErr)
}

func TestPinguinNotifierReturnsErrorWhenClientMissing(testingT *testing.T) {
	notifier := &PinguinNotifier{logger: zap.NewNop()}
	feedbackDelivery, notifyErr := notifier.NotifyFeedback(context.Background(), model.Site{}, model.Feedback{})
	require.Error(testingT, notifyErr)
	require.Equal(testingT, model.FeedbackDeliveryNone, feedbackDelivery)

	subscriptionErr := notifier.NotifySubscription(context.Background(), model.Site{}, model.Subscriber{})
	require.Error(testingT, subscriptionErr)

	emailErr := notifier.SendEmail(context.Background(), testFeedbackOwnerEmail, "Subject", "Message")
	require.Error(testingT, emailErr)
}

func TestNotifyFeedbackReturnsErrorWhenSendFails(testingT *testing.T) {
	service := &testNotificationService{responseError: errors.New(testSendFailureMessage)}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: time.Second,
		OperationTimeout:  time.Second,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	site := model.Site{
		ID:         testFeedbackSiteID,
		Name:       testFeedbackSiteName,
		OwnerEmail: testFeedbackOwnerEmail,
	}
	feedback := model.Feedback{
		ID:      testFeedbackID,
		Message: testFeedbackMessage,
	}

	delivery, notifyErr := notifier.NotifyFeedback(context.Background(), site, feedback)
	require.Error(testingT, notifyErr)
	require.Equal(testingT, model.FeedbackDeliveryNone, delivery)
}

func TestNewPinguinNotifierDefaultsTimeouts(testingT *testing.T) {
	service := &testNotificationService{responseStatus: pinguinpb.Status_SENT}
	listener := startNotificationServer(testingT, service)

	notifier, createErr := NewPinguinNotifier(zap.NewNop(), PinguinConfig{
		Address:           testPinguinAddress,
		AuthToken:         testPinguinAuthToken,
		TenantID:          testPinguinTenantID,
		ConnectionTimeout: 0,
		OperationTimeout:  0,
		Dialer:            createPinguinDialer(listener),
	})
	require.NoError(testingT, createErr)
	testingT.Cleanup(func() {
		_ = notifier.Close()
	})

	require.Equal(testingT, testDefaultConnTimeout, notifier.connectionTimeout)
	require.Equal(testingT, testDefaultOpTimeout, notifier.operationTimeout)
}
