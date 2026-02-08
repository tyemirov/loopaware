package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
)

const (
	errorCodeInvalidJSON             = "invalid_json"
	errorCodeMissingFields           = "missing_fields"
	errorCodeInvalidOwner            = "invalid_owner"
	errorCodeNothingToUpdate         = "nothing_to_update"
	errorCodeSaveFailed              = "save_failed"
	errorCodeQueryFailed             = "query_failed"
	testSubscriberUpdateCallbackName = "force_subscriber_update_error"
	testSubscriberUpdateErrorMessage = "forced subscriber update error"
)

func newRawJSONContext(method string, path string, body string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return recorder, context
}

func TestCreateSiteRejectsInvalidJSON(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	recorder, context := newRawJSONContext(http.MethodPost, "/api/sites", "{invalid")
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidJSON, responseBody[jsonErrorKey])
}

func TestCreateSiteRejectsMissingFields(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name": "Missing Origin",
	}
	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeMissingFields, responseBody[jsonErrorKey])
}

func TestCreateSiteRejectsInvalidOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Invalid Owner",
		"allowed_origin": "http://invalid-owner.example",
	}
	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: "   ", Role: api.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidOwner, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsInvalidJSON(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Invalid JSON",
		AllowedOrigin: "http://invalid-json.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newRawJSONContext(http.MethodPatch, "/api/sites/"+site.ID, "{invalid")
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidJSON, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsNothingToUpdate(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Nothing To Update",
		AllowedOrigin: "http://nothing.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]any{}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeNothingToUpdate, responseBody[jsonErrorKey])
}

func TestListSitesReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	sqlDatabase, sqlErr := harness.database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestListSitesRequiresAuth(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestUserAvatarReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	sqlDatabase, sqlErr := harness.database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestUserAvatarDefaultsContentType(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	user := model.User{
		Email:      testUserEmailAddress,
		AvatarData: []byte{0x01, 0x02},
	}
	require.NoError(testingT, harness.database.Create(&user).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testUserEmailAddress, Role: api.RoleUser})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Equal(testingT, "application/octet-stream", recorder.Header().Get("Content-Type"))
}

func TestCreateSiteReportsSaveError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	registerErr := harness.database.Callback().Create().Before("gorm:create").Register("force_create_error", func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New("forced create error"))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = harness.database.Callback().Create().Remove("force_create_error")
	})

	payload := map[string]any{
		"name":           "Save Error",
		"allowed_origin": "http://save-error.example",
	}
	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSaveFailed, responseBody[jsonErrorKey])
}

func TestCreateSiteReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	sqlDatabase, sqlErr := harness.database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	payload := map[string]any{
		"name":           "Query Error",
		"allowed_origin": "http://query-error.example",
	}
	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestUpdateSiteReportsSaveError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Update Save Error",
		AllowedOrigin: "http://update-save.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	registerErr := harness.database.Callback().Update().Before("gorm:update").Register("force_update_error", func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New("forced update error"))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = harness.database.Callback().Update().Remove("force_update_error")
	})

	payload := map[string]any{
		"name": "Updated Name",
	}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSaveFailed, responseBody[jsonErrorKey])
}

func TestDeleteSiteReportsDeleteError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Delete Site",
		AllowedOrigin: "http://delete-error.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	registerErr := harness.database.Callback().Delete().Before("gorm:delete").Register("force_delete_error", func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New("forced delete error"))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = harness.database.Callback().Delete().Remove("force_delete_error")
	})

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, "delete_failed", responseBody[jsonErrorKey])
}

func TestUpdateSubscriberStatusMarksConfirmed(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscriber Confirmed",
		AllowedOrigin: "http://subscriber-confirmed.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  "confirm@example.com",
		Status: model.SubscriberStatusUnsubscribed,
	})
	require.NoError(testingT, subscriberErr)
	subscriber.UnsubscribedAt = time.Now().UTC()
	require.NoError(testingT, harness.database.Save(&subscriber).Error)

	payload := map[string]any{"status": model.SubscriberStatusConfirmed}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID+"/subscribers/"+subscriber.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}, {Key: "subscriber_id", Value: subscriber.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSubscriberStatus(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var updatedSubscriber model.Subscriber
	require.NoError(testingT, harness.database.First(&updatedSubscriber, "id = ?", subscriber.ID).Error)
	require.Equal(testingT, model.SubscriberStatusConfirmed, updatedSubscriber.Status)
	require.False(testingT, updatedSubscriber.ConfirmedAt.IsZero())
	require.True(testingT, updatedSubscriber.UnsubscribedAt.IsZero())
}

func TestUpdateSubscriberStatusReportsSaveError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscriber Update Error",
		AllowedOrigin: "http://subscriber-update-error.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	subscriber, subscriberErr := model.NewSubscriber(model.SubscriberInput{
		SiteID: site.ID,
		Email:  "update-error@example.com",
	})
	require.NoError(testingT, subscriberErr)
	require.NoError(testingT, harness.database.Create(&subscriber).Error)

	registerErr := harness.database.Callback().Update().Before("gorm:update").Register(testSubscriberUpdateCallbackName, func(callbackDatabase *gorm.DB) {
		callbackDatabase.AddError(errors.New(testSubscriberUpdateErrorMessage))
	})
	require.NoError(testingT, registerErr)
	testingT.Cleanup(func() {
		_ = harness.database.Callback().Update().Remove(testSubscriberUpdateCallbackName)
	})

	payload := map[string]any{"status": model.SubscriberStatusUnsubscribed}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID+"/subscribers/"+subscriber.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}, {Key: "subscriber_id", Value: subscriber.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSubscriberStatus(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSaveFailed, responseBody[jsonErrorKey])
}

func TestListMessagesBySiteReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Messages Error",
		AllowedOrigin: "http://messages-error.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	require.NoError(testingT, harness.database.Migrator().DropTable(&model.Feedback{}))

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestListSubscribersReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscriber Error",
		AllowedOrigin: "http://subscriber-error.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	require.NoError(testingT, harness.database.Migrator().DropTable(&model.Subscriber{}))

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/subscribers", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.ListSubscribers(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestExportSubscribersReportsQueryError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Export Error",
		AllowedOrigin: "http://export-error.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	require.NoError(testingT, harness.database.Migrator().DropTable(&model.Subscriber{}))

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/subscribers/export", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.ExportSubscribers(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeQueryFailed, responseBody[jsonErrorKey])
}

func TestDeleteSubscriberReportsSaveError(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Delete Subscriber Error",
		AllowedOrigin: "http://delete-subscriber.example",
		OwnerEmail:    testAdminEmailAddress,
		CreatorEmail:  testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	require.NoError(testingT, harness.database.Migrator().DropTable(&model.Subscriber{}))

	subscriberID := storage.NewID()
	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID+"/subscribers/"+subscriberID, nil)
	context.Params = gin.Params{
		{Key: "id", Value: site.ID},
		{Key: "subscriber_id", Value: subscriberID},
	}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.DeleteSubscriber(context)
	require.Equal(testingT, http.StatusInternalServerError, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeSaveFailed, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsMissingFields(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Missing Fields",
		AllowedOrigin: "http://missing-fields.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]string{
		"name": "   ",
	}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeMissingFields, responseBody[jsonErrorKey])
}

func TestUpdateSiteRejectsInvalidOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)
	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Invalid Owner",
		AllowedOrigin: "http://invalid-owner-update.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]string{
		"owner_email": "   ",
	}
	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &api.CurrentUser{Email: testAdminEmailAddress, Role: api.RoleAdmin})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)

	var responseBody map[string]string
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, errorCodeInvalidOwner, responseBody[jsonErrorKey])
}
