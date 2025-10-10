package httpapi

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const (
	testAdminEmailAddress = "admin@example.com"
	testUserEmailAddress  = "user@example.com"
	testWidgetBaseURL     = "https://loopaware.example"
)

type siteServiceHarness struct {
	service  *SiteService
	database *gorm.DB
}

func newSiteServiceHarness(t *testing.T) siteServiceHarness {
	t.Helper()

	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, openErr)
	require.NoError(t, storage.AutoMigrate(database))

	service := NewSiteService(database, zap.NewNop(), testWidgetBaseURL)
	return siteServiceHarness{service: service, database: database}
}

func TestSiteServiceCreateSiteEnforcesOwnership(t *testing.T) {
	harness := newSiteServiceHarness(t)

	testCases := []struct {
		name        string
		currentUser *CurrentUser
		request     createSiteRequest
		expectOwner string
		expectError string
	}{
		{
			name: "admin can assign owner",
			currentUser: &CurrentUser{
				Email:   testAdminEmailAddress,
				IsAdmin: true,
			},
			request: createSiteRequest{
				Name:          "Admin Site",
				AllowedOrigin: "https://admin.example",
				OwnerEmail:    testUserEmailAddress,
			},
			expectOwner: strings.ToLower(testUserEmailAddress),
		},
		{
			name: "user assigned to self",
			currentUser: &CurrentUser{
				Email: testUserEmailAddress,
			},
			request: createSiteRequest{
				Name:          "User Site",
				AllowedOrigin: "https://user.example",
			},
			expectOwner: strings.ToLower(testUserEmailAddress),
		},
		{
			name: "user cannot assign foreign owner",
			currentUser: &CurrentUser{
				Email: testUserEmailAddress,
			},
			request: createSiteRequest{
				Name:          "Rejected Site",
				AllowedOrigin: "https://reject.example",
				OwnerEmail:    testAdminEmailAddress,
			},
			expectError: errorValueInvalidOperation,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response, err := harness.service.CreateSite(testCase.currentUser, testCase.request)
			if testCase.expectError != "" {
				require.Error(t, err)
				var siteErr *siteError
				require.ErrorAs(t, err, &siteErr)
				require.Equal(t, testCase.expectError, siteErr.Code())
				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.request.Name, response.Name)
			require.Equal(t, testCase.expectOwner, response.OwnerEmail)

			var persisted model.Site
			require.NoError(t, harness.database.First(&persisted, "id = ?", response.ID).Error)
			require.Equal(t, testCase.expectOwner, persisted.OwnerEmail)
		})
	}
}

func TestSiteServiceUpdateAndDeleteLifecycle(t *testing.T) {
	harness := newSiteServiceHarness(t)

	currentUser := &CurrentUser{Email: testAdminEmailAddress, IsAdmin: true}
	created, err := harness.service.CreateSite(currentUser, createSiteRequest{
		Name:          "Lifecycle",
		AllowedOrigin: "https://lifecycle.example",
		OwnerEmail:    testAdminEmailAddress,
	})
	require.NoError(t, err)

	newName := "Lifecycle Updated"
	newOrigin := "https://updated.example"
	updated, updateErr := harness.service.UpdateSite(currentUser, created.ID, updateSiteRequest{
		Name:          &newName,
		AllowedOrigin: &newOrigin,
	})
	require.NoError(t, updateErr)
	require.Equal(t, newName, updated.Name)
	require.Equal(t, newOrigin, updated.AllowedOrigin)

	feedback := model.Feedback{
		ID:      storage.NewID(),
		SiteID:  created.ID,
		Contact: "contact@example.com",
		Message: "Hello",
	}
	require.NoError(t, harness.database.Create(&feedback).Error)

	require.NoError(t, harness.service.DeleteSite(currentUser, created.ID))

	var persisted model.Site
	require.Error(t, harness.database.First(&persisted, "id = ?", created.ID).Error)

	var remainingFeedback model.Feedback
	require.Error(t, harness.database.First(&remainingFeedback, "id = ?", feedback.ID).Error)
}

func TestSiteServiceListMessagesReturnsNewestFirst(t *testing.T) {
	harness := newSiteServiceHarness(t)

	currentUser := &CurrentUser{Email: testAdminEmailAddress, IsAdmin: true}
	created, err := harness.service.CreateSite(currentUser, createSiteRequest{
		Name:          "Messages",
		AllowedOrigin: "https://messages.example",
		OwnerEmail:    testAdminEmailAddress,
	})
	require.NoError(t, err)

	first := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    created.ID,
		Contact:   "first@example.com",
		Message:   "First",
		CreatedAt: time.Now().Add(-time.Minute),
	}
	second := model.Feedback{
		ID:        storage.NewID(),
		SiteID:    created.ID,
		Contact:   "second@example.com",
		Message:   "Second",
		CreatedAt: time.Now(),
	}
	require.NoError(t, harness.database.Create(&first).Error)
	require.NoError(t, harness.database.Create(&second).Error)

	messages, listErr := harness.service.ListMessagesForSite(created.ID, currentUser)
	require.NoError(t, listErr)
	require.Len(t, messages, 2)
	require.GreaterOrEqual(t, messages[0].CreatedAt, messages[1].CreatedAt)
}

func TestDashboardRenderDisplaysSitesAndWidget(t *testing.T) {
	harness := newSiteServiceHarness(t)
	currentUser := &CurrentUser{
		Email:   testAdminEmailAddress,
		Name:    "Admin",
		IsAdmin: true,
	}

	created, err := harness.service.CreateSite(currentUser, createSiteRequest{
		Name:          "Renderable",
		AllowedOrigin: "https://render.example",
		OwnerEmail:    testAdminEmailAddress,
	})
	require.NoError(t, err)

	dashboardHandlers := NewDashboardWebHandlers(zap.NewNop(), harness.service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/app", nil)
	context.Request = request
	context.Set(contextKeyCurrentUser, currentUser)

	dashboardHandlers.RenderDashboard(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, created.Name)
	require.Contains(t, body, html.EscapeString(created.Widget))
	require.Contains(t, body, "Copy widget")
}

func TestDashboardCreateSiteRedirects(t *testing.T) {
	harness := newSiteServiceHarness(t)
	currentUser := &CurrentUser{
		Email:   testAdminEmailAddress,
		IsAdmin: true,
	}
	dashboardHandlers := NewDashboardWebHandlers(zap.NewNop(), harness.service)

	form := url.Values{}
	form.Set("name", "Dashboard Created")
	form.Set("allowed_origin", "https://dashboard.example")
	form.Set("owner_email", testUserEmailAddress)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/app/sites", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, request.ParseForm())
	context.Request = request
	context.Set(contextKeyCurrentUser, currentUser)

	dashboardHandlers.CreateSite(context)
	require.Equal(t, http.StatusSeeOther, recorder.Code)
	location := recorder.Header().Get("Location")
	require.Contains(t, location, "/app?site_id=")
	require.Contains(t, location, "notice=site_created")

	var persisted model.Site
	require.NoError(t, harness.database.First(&persisted, "name = ?", "Dashboard Created").Error)
}

func TestDashboardUserAvatarServesStoredImage(t *testing.T) {
	harness := newSiteServiceHarness(t)
	currentUser := &CurrentUser{
		Email: testUserEmailAddress,
	}
	require.NoError(t, harness.database.Save(&model.User{
		Email:             strings.ToLower(testUserEmailAddress),
		Name:              "Avatar",
		AvatarContentType: "image/png",
		AvatarData:        []byte{0x01, 0x02},
	}).Error)

	dashboardHandlers := NewDashboardWebHandlers(zap.NewNop(), harness.service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/app/avatar", nil)
	context.Request = request
	context.Set(contextKeyCurrentUser, currentUser)

	dashboardHandlers.UserAvatar(context)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(t, []byte{0x01, 0x02}, recorder.Body.Bytes())
}
