package pinguinpb

import "testing"

func TestNotificationProtoMessageCoverage(testingT *testing.T) {
	notificationRequest := &NotificationRequest{}
	notificationRequest.ProtoMessage()

	notificationResponse := &NotificationResponse{}
	notificationResponse.ProtoMessage()

	statusRequest := &GetNotificationStatusRequest{}
	statusRequest.ProtoMessage()
}

func TestUnimplementedNotificationServiceServerCoverage(testingT *testing.T) {
	unimplementedServer := UnimplementedNotificationServiceServer{}
	unimplementedServer.mustEmbedUnimplementedNotificationServiceServer()
	unimplementedServer.testEmbeddedByValue()
}
