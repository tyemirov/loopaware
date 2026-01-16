package pinguinpb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testNotificationService struct {
	UnimplementedNotificationServiceServer
	sendCalls   int
	statusCalls int
}

func (service *testNotificationService) SendNotification(context.Context, *NotificationRequest) (*NotificationResponse, error) {
	service.sendCalls++
	return &NotificationResponse{NotificationId: "notify-id"}, nil
}

func (service *testNotificationService) GetNotificationStatus(context.Context, *GetNotificationStatusRequest) (*NotificationResponse, error) {
	service.statusCalls++
	return &NotificationResponse{Status: Status_SENT}, nil
}

func TestNotificationRequestNilGetters(testingT *testing.T) {
	var request *NotificationRequest
	require.Equal(testingT, NotificationType_EMAIL, request.GetNotificationType())
	require.Empty(testingT, request.GetRecipient())
	require.Empty(testingT, request.GetSubject())
	require.Empty(testingT, request.GetMessage())
	require.Nil(testingT, request.GetScheduledTime())
}

func TestNotificationResponseNilGetters(testingT *testing.T) {
	var response *NotificationResponse
	require.Empty(testingT, response.GetNotificationId())
	require.Equal(testingT, NotificationType_EMAIL, response.GetNotificationType())
	require.Empty(testingT, response.GetRecipient())
	require.Empty(testingT, response.GetSubject())
	require.Empty(testingT, response.GetMessage())
	require.Equal(testingT, Status_QUEUED, response.GetStatus())
	require.Empty(testingT, response.GetProviderMessageId())
	require.Equal(testingT, int32(0), response.GetRetryCount())
	require.Empty(testingT, response.GetCreatedAt())
	require.Empty(testingT, response.GetUpdatedAt())
	require.Nil(testingT, response.GetScheduledTime())
}

func TestGetNotificationStatusRequestNilGetter(testingT *testing.T) {
	var request *GetNotificationStatusRequest
	require.Empty(testingT, request.GetNotificationId())
}

func TestProtoMessageAndReflectCoverage(testingT *testing.T) {
	request := &NotificationRequest{}
	response := &NotificationResponse{}
	statusRequest := &GetNotificationStatusRequest{}

	request.ProtoMessage()
	response.ProtoMessage()
	statusRequest.ProtoMessage()

	require.NotNil(testingT, request.ProtoReflect())
	require.NotNil(testingT, response.ProtoReflect())
	require.NotNil(testingT, statusRequest.ProtoReflect())

	var nilRequest *NotificationRequest
	require.NotNil(testingT, nilRequest.ProtoReflect())
}

func TestUnimplementedNotificationServiceServerMethodsAdditional(testingT *testing.T) {
	server := UnimplementedNotificationServiceServer{}
	_, sendErr := server.SendNotification(context.Background(), &NotificationRequest{})
	require.Error(testingT, sendErr)
	_, statusErr := server.GetNotificationStatus(context.Background(), &GetNotificationStatusRequest{})
	require.Error(testingT, statusErr)
	server.mustEmbedUnimplementedNotificationServiceServer()
	server.testEmbeddedByValue()
}

func TestSendNotificationHandlerUsesInterceptor(testingT *testing.T) {
	service := &testNotificationService{}
	decoder := func(request any) error {
		notification, ok := request.(*NotificationRequest)
		if !ok {
			return errors.New("unexpected request type")
		}
		notification.Recipient = "user@example.com"
		return nil
	}
	interceptorCalled := false
	interceptor := func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		interceptorCalled = true
		return handler(ctx, request)
	}

	handlerResponse, handlerErr := _NotificationService_SendNotification_Handler(service, context.Background(), decoder, interceptor)
	require.NoError(testingT, handlerErr)
	require.True(testingT, interceptorCalled)
	require.Equal(testingT, 1, service.sendCalls)
	require.Equal(testingT, "notify-id", handlerResponse.(*NotificationResponse).NotificationId)
}

func TestSendNotificationHandlerReportsDecodeError(testingT *testing.T) {
	service := &testNotificationService{}
	decoder := func(request any) error {
		return errors.New("decode failed")
	}

	handlerResponse, handlerErr := _NotificationService_SendNotification_Handler(service, context.Background(), decoder, nil)
	require.Error(testingT, handlerErr)
	require.Nil(testingT, handlerResponse)
}

func TestGetNotificationStatusHandlerReportsDecodeError(testingT *testing.T) {
	service := &testNotificationService{}
	decoder := func(request any) error {
		return errors.New("decode failed")
	}

	handlerResponse, handlerErr := _NotificationService_GetNotificationStatus_Handler(service, context.Background(), decoder, nil)
	require.Error(testingT, handlerErr)
	require.Nil(testingT, handlerResponse)
}

func TestGetNotificationStatusHandlerCallsService(testingT *testing.T) {
	service := &testNotificationService{}
	decoder := func(request any) error {
		statusRequest, ok := request.(*GetNotificationStatusRequest)
		if !ok {
			return errors.New("unexpected request type")
		}
		statusRequest.NotificationId = "status-id"
		return nil
	}

	handlerResponse, handlerErr := _NotificationService_GetNotificationStatus_Handler(service, context.Background(), decoder, nil)
	require.NoError(testingT, handlerErr)
	require.Equal(testingT, 1, service.statusCalls)
	require.Equal(testingT, Status_SENT, handlerResponse.(*NotificationResponse).Status)
}
