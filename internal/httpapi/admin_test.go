package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const (
	testAdminEmailAddress = "admin@example.com"
	testUserEmailAddress  = "user@example.com"
	testSessionContextKey = "httpapi_current_user"
	testWidgetBaseURL     = "https://gravity.mprlab.com/"
)

type siteTestHarness struct {
	handlers *httpapi.SiteHandlers
	database *gorm.DB
}

func newSiteTestHarness(testingT *testing.T) siteTestHarness {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	require.NoError(testingT, storage.AutoMigrate(database))

	handlers := httpapi.NewSiteHandlers(database, zap.NewNop(), testWidgetBaseURL, nil, nil)

	return siteTestHarness{handlers: handlers, database: database}
}

func TestListMessagesBySiteReturnsOrderedUnixTimestamps(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "List Messages Site",
		AllowedOrigin: "http://list.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	firstFeedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "first@example.com",
		Message:   "First",
		CreatedAt: time.Now().Add(-time.Minute),
	}
	secondFeedback := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    site.ID,
		Contact:   "second@example.com",
		Message:   "Second",
		CreatedAt: time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&firstFeedback).Error)
	require.NoError(testingT, harness.database.Create(&secondFeedback).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, IsAdmin: true})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		SiteID   string `json:"site_id"`
		Messages []struct {
			Identifier string `json:"id"`
			CreatedAt  int64  `json:"created_at"`
		} `json:"messages"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, site.ID, responseBody.SiteID)
	require.Len(testingT, responseBody.Messages, 2)
	require.GreaterOrEqual(testingT, responseBody.Messages[0].CreatedAt, responseBody.Messages[1].CreatedAt)
}

func TestListSitesUsesPublicBaseURLForWidget(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                 storage.NewID(),
		Name:               "Widget Site",
		AllowedOrigin:      "https://client.example",
		OwnerEmail:         testAdminEmailAddress,
		FaviconData:        []byte{0x01, 0x02, 0x03},
		FaviconContentType: "image/png",
		FaviconOrigin:      "https://client.example",
		FaviconFetchedAt:   time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&site).Error)
	for index := 0; index < 5; index++ {
		feedback := model.Feedback{
			ID:      storage.NewID(),
			SiteID:  site.ID,
			Contact: fmt.Sprintf("contact-%d@example.com", index),
			Message: fmt.Sprintf("Message %d", index),
		}
		require.NoError(testingT, harness.database.Create(&feedback).Error)
	}
	recorder, context := newJSONContext(http.MethodGet, "/api/sites", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, IsAdmin: true})

	harness.handlers.ListSites(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody struct {
		Sites []struct {
			Identifier    string `json:"id"`
			Widget        string `json:"widget"`
			FaviconURL    string `json:"favicon_url"`
			FeedbackCount int64  `json:"feedback_count"`
		} `json:"sites"`
	}
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Len(testingT, responseBody.Sites, 1)

	expectedBaseURL := strings.TrimRight(testWidgetBaseURL, "/")
	expectedWidget := fmt.Sprintf("<script defer src=\"%s/widget.js?site_id=%s\"></script>", expectedBaseURL, site.ID)
	require.Equal(testingT, expectedWidget, responseBody.Sites[0].Widget)
	expectedFavicon := fmt.Sprintf("/api/sites/%s/favicon", site.ID)
	require.Equal(testingT, expectedFavicon, responseBody.Sites[0].FaviconURL)
	require.Equal(testingT, int64(5), responseBody.Sites[0].FeedbackCount)
}

func TestSiteFaviconReturnsStoredIcon(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:                 storage.NewID(),
		Name:               "Icon Site",
		AllowedOrigin:      "https://icon.example",
		OwnerEmail:         testAdminEmailAddress,
		FaviconData:        []byte{0x10, 0x20, 0x30},
		FaviconContentType: "image/png",
		FaviconOrigin:      "https://icon.example",
		FaviconFetchedAt:   time.Now(),
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/favicon", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, IsAdmin: true})

	harness.handlers.SiteFavicon(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Equal(testingT, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(testingT, []byte{0x10, 0x20, 0x30}, recorder.Body.Bytes())
}

func TestNonAdminCannotAccessForeignSite(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Foreign Site",
		AllowedOrigin: "http://foreign.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/sites/"+site.ID+"/messages", nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, IsAdmin: false})

	harness.handlers.ListMessagesBySite(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestCreateSiteAllowsAdminToSpecifyOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Admin Created",
		"allowed_origin": "http://owned.example",
		"owner_email":    testUserEmailAddress,
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, IsAdmin: true})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, "Admin Created", responseBody["name"])
	require.Equal(testingT, "http://owned.example", responseBody["allowed_origin"])
	require.Equal(testingT, testUserEmailAddress, responseBody["owner_email"])
	require.Equal(testingT, float64(0), responseBody["feedback_count"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Admin Created").Error)
	require.Equal(testingT, testUserEmailAddress, createdSite.OwnerEmail)
}

func TestCreateSiteAssignsCurrentUserAsOwner(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Self Owned",
		"allowed_origin": "http://self.example",
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, IsAdmin: false})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, testUserEmailAddress, responseBody["owner_email"])
	require.Equal(testingT, float64(0), responseBody["feedback_count"])

	var createdSite model.Site
	require.NoError(testingT, harness.database.First(&createdSite, "name = ?", "Self Owned").Error)
	require.Equal(testingT, testUserEmailAddress, createdSite.OwnerEmail)
}

func TestCreateSiteRejectsForeignOwnerForRegularUser(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	payload := map[string]string{
		"name":           "Invalid Owner",
		"allowed_origin": "http://invalid.example",
		"owner_email":    "other@example.com",
	}

	recorder, context := newJSONContext(http.MethodPost, "/api/sites", payload)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, IsAdmin: false})

	harness.handlers.CreateSite(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestUpdateSiteAllowsOwnerToChangeDetails(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Owner Site",
		AllowedOrigin: "http://owner.example",
		OwnerEmail:    testUserEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	payload := map[string]string{
		"name":           "Updated Name",
		"allowed_origin": "http://updated.example",
	}

	recorder, context := newJSONContext(http.MethodPatch, "/api/sites/"+site.ID, payload)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, IsAdmin: false})

	harness.handlers.UpdateSite(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)

	var responseBody map[string]any
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	require.Equal(testingT, "Updated Name", responseBody["name"])
	require.Equal(testingT, "http://updated.example", responseBody["allowed_origin"])
}

func TestDeleteSiteRemovesSiteAndFeedback(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Deletable Site",
		AllowedOrigin: "http://delete.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	feedback := model.Feedback{
		ID:      storage.NewID(),
		SiteID:  site.ID,
		Contact: "contact@example.com",
		Message: "Message",
	}
	require.NoError(testingT, harness.database.Create(&feedback).Error)

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testAdminEmailAddress, IsAdmin: true})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusNoContent, recorder.Code)

	var remainingSite model.Site
	require.ErrorIs(testingT, harness.database.First(&remainingSite, "id = ?", site.ID).Error, gorm.ErrRecordNotFound)

	var remainingFeedback model.Feedback
	require.ErrorIs(testingT, harness.database.First(&remainingFeedback, "id = ?", feedback.ID).Error, gorm.ErrRecordNotFound)
}

func TestDeleteSitePreventsUnauthorizedUser(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Protected Site",
		AllowedOrigin: "http://protected.example",
		OwnerEmail:    testAdminEmailAddress,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	recorder, context := newJSONContext(http.MethodDelete, "/api/sites/"+site.ID, nil)
	context.Params = gin.Params{{Key: "id", Value: site.ID}}
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress, IsAdmin: false})

	harness.handlers.DeleteSite(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)

	var persistedSite model.Site
	require.NoError(testingT, harness.database.First(&persistedSite, "id = ?", site.ID).Error)
}

func TestUserAvatarReturnsStoredImage(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	user := model.User{
		Email:             strings.ToLower(testUserEmailAddress),
		Name:              "Test User",
		AvatarContentType: "image/png",
		AvatarData:        []byte{0x01, 0x02, 0x03},
	}
	require.NoError(testingT, harness.database.Save(&user).Error)

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Equal(testingT, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(testingT, []byte{0x01, 0x02, 0x03}, recorder.Body.Bytes())
}

func TestUserAvatarReturnsNotFoundWhenMissing(testingT *testing.T) {
	harness := newSiteTestHarness(testingT)

	recorder, context := newJSONContext(http.MethodGet, "/api/me/avatar", nil)
	context.Set(testSessionContextKey, &httpapi.CurrentUser{Email: testUserEmailAddress})

	harness.handlers.UserAvatar(context)
	require.Equal(testingT, http.StatusNotFound, recorder.Code)
}

func newJSONContext(method string, path string, body any) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	var requestBody *bytes.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		requestBody = bytes.NewReader(encoded)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	request := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return recorder, context
}
