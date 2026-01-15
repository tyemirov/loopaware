package pinguinpb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testRecipientEmail = "user@example.com"
	testSubjectLine    = "Subject"
	testMessageBody    = "Message"
	testNotificationID = "notification-id"
	testProviderID     = "provider-id"
	testCreatedAtValue = "created-at"
	testUpdatedAtValue = "updated-at"
)

type stubClientConn struct {
	lastMethod     string
	lastRequest    any
	responseStatus Status
	invokeError    error
}

func (stub *stubClientConn) Invoke(_ context.Context, method string, args any, reply any, _ ...grpc.CallOption) error {
	stub.lastMethod = method
	stub.lastRequest = args
	if stub.invokeError != nil {
		return stub.invokeError
	}
	response, isResponse := reply.(*NotificationResponse)
	if isResponse {
		response.Status = stub.responseStatus
	}
	return nil
}

func (stub *stubClientConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("stream not implemented")
}

type stubNotificationServer struct {
	UnimplementedNotificationServiceServer
}

type handlerNotificationServer struct {
	UnimplementedNotificationServiceServer
}

func (server *handlerNotificationServer) SendNotification(context.Context, *NotificationRequest) (*NotificationResponse, error) {
	return &NotificationResponse{Status: Status_SENT}, nil
}

func (server *handlerNotificationServer) GetNotificationStatus(context.Context, *GetNotificationStatusRequest) (*NotificationResponse, error) {
	return &NotificationResponse{Status: Status_SENT}, nil
}

func TestNotificationTypeEnumHelpers(testingT *testing.T) {
	notificationType := NotificationType_EMAIL
	require.Equal(testingT, NotificationType_EMAIL, *notificationType.Enum())
	require.NotEmpty(testingT, notificationType.String())
	require.NotNil(testingT, notificationType.Descriptor())
	require.NotNil(testingT, notificationType.Type())
	require.NotNil(testingT, notificationType.Number())
	enumBytes, enumIndexes := notificationType.EnumDescriptor()
	require.NotEmpty(testingT, enumBytes)
	require.NotEmpty(testingT, enumIndexes)
}

func TestStatusEnumHelpers(testingT *testing.T) {
	status := Status_SENT
	require.Equal(testingT, Status_SENT, *status.Enum())
	require.NotEmpty(testingT, status.String())
	require.NotNil(testingT, status.Descriptor())
	require.NotNil(testingT, status.Type())
	require.NotNil(testingT, status.Number())
	enumBytes, enumIndexes := status.EnumDescriptor()
	require.NotEmpty(testingT, enumBytes)
	require.NotEmpty(testingT, enumIndexes)
}

func TestNotificationRequestMethods(testingT *testing.T) {
	scheduledTime := timestamppb.New(time.Unix(123, 0))
	request := &NotificationRequest{
		NotificationType: NotificationType_EMAIL,
		Recipient:        testRecipientEmail,
		Subject:          testSubjectLine,
		Message:          testMessageBody,
		ScheduledTime:    scheduledTime,
	}
	request.Reset()
	request.NotificationType = NotificationType_EMAIL
	request.Recipient = testRecipientEmail
	request.Subject = testSubjectLine
	request.Message = testMessageBody
	request.ScheduledTime = scheduledTime

	require.NotEmpty(testingT, request.String())
	require.NotNil(testingT, request.ProtoReflect())
	request.ProtoMessage()
	descriptorBytes, descriptorIndexes := request.Descriptor()
	require.NotEmpty(testingT, descriptorBytes)
	require.NotEmpty(testingT, descriptorIndexes)
	require.Equal(testingT, NotificationType_EMAIL, request.GetNotificationType())
	require.Equal(testingT, testRecipientEmail, request.GetRecipient())
	require.Equal(testingT, testSubjectLine, request.GetSubject())
	require.Equal(testingT, testMessageBody, request.GetMessage())
	require.Equal(testingT, int64(123), request.GetScheduledTime().AsTime().Unix())
}

func TestNotificationResponseMethods(testingT *testing.T) {
	scheduledResponseTime := timestamppb.New(time.Unix(456, 0))
	response := &NotificationResponse{
		NotificationId:    testNotificationID,
		NotificationType:  NotificationType_EMAIL,
		Recipient:         testRecipientEmail,
		Subject:           testSubjectLine,
		Message:           testMessageBody,
		Status:            Status_SENT,
		ProviderMessageId: testProviderID,
		RetryCount:        2,
		CreatedAt:         testCreatedAtValue,
		UpdatedAt:         testUpdatedAtValue,
		ScheduledTime:     scheduledResponseTime,
	}
	response.Reset()
	response.NotificationId = testNotificationID
	response.NotificationType = NotificationType_EMAIL
	response.Recipient = testRecipientEmail
	response.Subject = testSubjectLine
	response.Message = testMessageBody
	response.Status = Status_SENT
	response.ProviderMessageId = testProviderID
	response.RetryCount = 2
	response.CreatedAt = testCreatedAtValue
	response.UpdatedAt = testUpdatedAtValue
	response.ScheduledTime = scheduledResponseTime

	require.NotEmpty(testingT, response.String())
	require.NotNil(testingT, response.ProtoReflect())
	response.ProtoMessage()
	descriptorBytes, descriptorIndexes := response.Descriptor()
	require.NotEmpty(testingT, descriptorBytes)
	require.NotEmpty(testingT, descriptorIndexes)
	require.Equal(testingT, testNotificationID, response.GetNotificationId())
	require.Equal(testingT, NotificationType_EMAIL, response.GetNotificationType())
	require.Equal(testingT, testRecipientEmail, response.GetRecipient())
	require.Equal(testingT, testSubjectLine, response.GetSubject())
	require.Equal(testingT, testMessageBody, response.GetMessage())
	require.Equal(testingT, Status_SENT, response.GetStatus())
	require.Equal(testingT, testProviderID, response.GetProviderMessageId())
	require.Equal(testingT, int32(2), response.GetRetryCount())
	require.Equal(testingT, testCreatedAtValue, response.GetCreatedAt())
	require.Equal(testingT, testUpdatedAtValue, response.GetUpdatedAt())
	require.Equal(testingT, int64(456), response.GetScheduledTime().AsTime().Unix())
}

func TestGetNotificationStatusRequestMethods(testingT *testing.T) {
	request := &GetNotificationStatusRequest{
		NotificationId: testNotificationID,
	}
	request.Reset()
	request.NotificationId = testNotificationID

	require.NotEmpty(testingT, request.String())
	require.NotNil(testingT, request.ProtoReflect())
	request.ProtoMessage()
	descriptorBytes, descriptorIndexes := request.Descriptor()
	require.NotEmpty(testingT, descriptorBytes)
	require.NotEmpty(testingT, descriptorIndexes)
	require.Equal(testingT, testNotificationID, request.GetNotificationId())
}

func TestNotificationServiceClientInvokesRPC(testingT *testing.T) {
	clientConn := &stubClientConn{responseStatus: Status_SENT}
	client := NewNotificationServiceClient(clientConn)

	response, sendErr := client.SendNotification(context.Background(), &NotificationRequest{Recipient: testRecipientEmail})
	require.NoError(testingT, sendErr)
	require.Equal(testingT, Status_SENT, response.GetStatus())
	require.Equal(testingT, NotificationService_SendNotification_FullMethodName, clientConn.lastMethod)

	response, statusErr := client.GetNotificationStatus(context.Background(), &GetNotificationStatusRequest{NotificationId: testNotificationID})
	require.NoError(testingT, statusErr)
	require.Equal(testingT, Status_SENT, response.GetStatus())
	require.Equal(testingT, NotificationService_GetNotificationStatus_FullMethodName, clientConn.lastMethod)
}

func TestRegisterNotificationServiceServer(testingT *testing.T) {
	server := grpc.NewServer()
	RegisterNotificationServiceServer(server, &stubNotificationServer{})
	server.Stop()
}

func TestUnimplementedNotificationServiceServerMethods(testingT *testing.T) {
	server := UnimplementedNotificationServiceServer{}
	_, sendErr := server.SendNotification(context.Background(), &NotificationRequest{})
	require.Error(testingT, sendErr)

	_, statusErr := server.GetNotificationStatus(context.Background(), &GetNotificationStatusRequest{})
	require.Error(testingT, statusErr)

	server.mustEmbedUnimplementedNotificationServiceServer()
	server.testEmbeddedByValue()
}

func TestNotificationServiceHandlers(testingT *testing.T) {
	server := &handlerNotificationServer{}
	notificationRequest := &NotificationRequest{Recipient: testRecipientEmail}
	sendDecoder := func(target interface{}) error {
		typedTarget, isRequest := target.(*NotificationRequest)
		if !isRequest {
			return errors.New("unexpected request type")
		}
		typedTarget.Recipient = notificationRequest.Recipient
		return nil
	}

	response, sendErr := _NotificationService_SendNotification_Handler(server, context.Background(), sendDecoder, nil)
	require.NoError(testingT, sendErr)
	require.IsType(testingT, &NotificationResponse{}, response)

	statusRequest := &GetNotificationStatusRequest{NotificationId: testNotificationID}
	statusDecoder := func(target interface{}) error {
		typedTarget, isRequest := target.(*GetNotificationStatusRequest)
		if !isRequest {
			return errors.New("unexpected request type")
		}
		typedTarget.NotificationId = statusRequest.NotificationId
		return nil
	}

	interceptor := func(handlerContext context.Context, request interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(handlerContext, request)
	}

	response, statusErr := _NotificationService_GetNotificationStatus_Handler(server, context.Background(), statusDecoder, interceptor)
	require.NoError(testingT, statusErr)
	require.IsType(testingT, &NotificationResponse{}, response)
}
