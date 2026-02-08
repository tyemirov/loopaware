package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/tyemirov/tauth/pkg/sessionvalidator"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testAuthSigningKeyValue = "auth-signing-key"
	testAuthCookieNameValue = "app_session"
	testAuthTenantIDValue   = "tenant-id"
	testAuthMismatchTenant  = "other-tenant"
	testAuthBlankEmail      = "   "
	testAuthUserID          = "test-user"
	testAuthUserEmail       = "user@example.com"
	testAuthUserName        = "Test User"
	testAuthIssuerValue     = "tauth"
	testAvatarContentType   = "image/png"
	testAvatarBytesValue    = "avatar-bytes"
	testAvatarURL           = "https://avatar.example/avatar.png"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

func TestCurrentUserRoleChecks(testingT *testing.T) {
	adminUser := &CurrentUser{Email: "Admin@Example.com", Role: RoleAdmin}
	require.True(testingT, adminUser.hasRole(RoleAdmin))
	require.False(testingT, adminUser.hasRole(RoleUser))
	require.Equal(testingT, "admin@example.com", adminUser.normalizedEmail())
}

func TestCurrentUserRoleChecksHandleNil(testingT *testing.T) {
	var currentUser *CurrentUser
	require.False(testingT, currentUser.hasRole(RoleAdmin))
}

func TestCurrentUserNormalizedEmailHandlesNil(testingT *testing.T) {
	var currentUser *CurrentUser
	require.Empty(testingT, currentUser.normalizedEmail())
}

func TestCurrentUserCanManageSiteByOwnerAndCreator(testingT *testing.T) {
	site := model.Site{OwnerEmail: "owner@example.com", CreatorEmail: "creator@example.com"}

	ownerUser := &CurrentUser{Email: "OWNER@example.com", Role: RoleUser}
	require.True(testingT, ownerUser.canManageSite(site))

	creatorUser := &CurrentUser{Email: "creator@example.com", Role: RoleUser}
	require.True(testingT, creatorUser.canManageSite(site))
}

func TestCurrentUserCanManageSiteRejectsBlankEmail(testingT *testing.T) {
	site := model.Site{OwnerEmail: "owner@example.com", CreatorEmail: "creator@example.com"}
	user := &CurrentUser{Email: "   ", Role: RoleUser}
	require.False(testingT, user.canManageSite(site))
}

func TestCurrentUserCanManageSiteHandlesNil(testingT *testing.T) {
	var currentUser *CurrentUser
	require.False(testingT, currentUser.canManageSite(model.Site{}))
}

func buildSessionCookie(testingT *testing.T, email string, tenantID string) *http.Cookie {
	testingT.Helper()

	now := time.Now().UTC()
	claims := &sessionvalidator.Claims{
		TenantID:        tenantID,
		UserID:          testAuthUserID,
		UserEmail:       email,
		UserDisplayName: testAuthUserName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testAuthIssuerValue,
			Subject:   testAuthUserID,
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, signErr := token.SignedString([]byte(testAuthSigningKeyValue))
	require.NoError(testingT, signErr)

	return &http.Cookie{
		Name:  testAuthCookieNameValue,
		Value: signedToken,
		Path:  "/",
	}
}

func openAuthDatabase(testingT *testing.T) *storage.Config {
	testingT.Helper()
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	config := sqliteDatabase.Configuration()
	return &config
}

func newAuthManager(testingT *testing.T, databaseConfig *storage.Config) *AuthManager {
	testingT.Helper()

	var databaseHandle *gorm.DB
	if databaseConfig != nil {
		var openErr error
		databaseHandle, openErr = storage.OpenDatabase(*databaseConfig)
		require.NoError(testingT, openErr)
		databaseHandle = testutil.ConfigureDatabaseLogger(testingT, databaseHandle)
		require.NoError(testingT, storage.AutoMigrate(databaseHandle))
	}

	manager, createErr := NewAuthManager(databaseHandle, zap.NewNop(), nil, nil, AuthConfig{
		SigningKey: testAuthSigningKeyValue,
		CookieName: testAuthCookieNameValue,
		TenantID:   testAuthTenantIDValue,
	})
	require.NoError(testingT, createErr)
	return manager
}

func TestRequireAuthenticatedJSONReturnsUnauthorizedWhenMissingSession(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	authManager := &AuthManager{logger: zap.NewNop()}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/me", nil)

	handler := authManager.RequireAuthenticatedJSON()
	handler(ginContext)

	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestCurrentUserUsesExistingContext(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	authManager := &AuthManager{logger: zap.NewNop()}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	currentUser := &CurrentUser{Email: testAuthUserEmail, Name: testAuthUserName, Role: RoleUser}
	ginContext.Set(contextKeyCurrentUser, currentUser)

	loadedUser, ok := authManager.CurrentUser(ginContext)
	require.True(testingT, ok)
	require.Equal(testingT, testAuthUserEmail, loadedUser.Email)
}

func TestEnsureUserRejectsInvalidToken(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	authManager := newAuthManager(testingT, nil)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/me", nil)

	loadedUser, ok := authManager.ensureUser(ginContext)
	require.False(testingT, ok)
	require.Nil(testingT, loadedUser)
}

func TestEnsureUserAcceptsValidToken(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	databaseConfig := openAuthDatabase(testingT)
	authManager := newAuthManager(testingT, databaseConfig)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(buildSessionCookie(testingT, testAuthUserEmail, testAuthTenantIDValue))
	ginContext.Request = request

	loadedUser, ok := authManager.ensureUser(ginContext)
	require.True(testingT, ok)
	require.Equal(testingT, testAuthUserEmail, loadedUser.Email)
}

func TestEnsureUserRejectsTenantMismatch(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	authManager := newAuthManager(testingT, nil)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(buildSessionCookie(testingT, testAuthUserEmail, testAuthMismatchTenant))
	ginContext.Request = request

	loadedUser, ok := authManager.ensureUser(ginContext)
	require.False(testingT, ok)
	require.Nil(testingT, loadedUser)
}

func TestEnsureUserRejectsMissingEmail(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	authManager := newAuthManager(testingT, nil)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(buildSessionCookie(testingT, testAuthBlankEmail, testAuthTenantIDValue))
	ginContext.Request = request

	loadedUser, ok := authManager.ensureUser(ginContext)
	require.False(testingT, ok)
	require.Nil(testingT, loadedUser)
}

func TestFetchAvatarHandlesHTTPResponses(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	responseBody := []byte(testAvatarBytesValue)
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.String() != testAvatarURL {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("not found")),
					Header:     http.Header{},
				}, nil
			}
			header := make(http.Header)
			header.Set("Content-Type", testAvatarContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(responseBody)),
				Header:     header,
			}, nil
		}),
	}

	authManager := &AuthManager{logger: zap.NewNop(), httpClient: client}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), testAvatarURL)
	require.NoError(testingT, fetchErr)
	require.Equal(testingT, testAvatarContentType, contentType)
	require.Equal(testingT, responseBody, avatarData)
}

func TestFetchAvatarRejectsOversizedPayload(testingT *testing.T) {
	largePayload := bytes.Repeat([]byte("a"), maxAvatarBytes+1)
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			header := make(http.Header)
			header.Set("Content-Type", testAvatarContentType)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(largePayload)),
				Header:     header,
			}, nil
		}),
	}

	authManager := &AuthManager{logger: zap.NewNop(), httpClient: client}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), testAvatarURL)
	require.Error(testingT, fetchErr)
	require.Empty(testingT, avatarData)
	require.Empty(testingT, contentType)
}

func TestFetchAvatarRejectsMissingClient(testingT *testing.T) {
	authManager := &AuthManager{}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), testAvatarURL)
	require.Error(testingT, fetchErr)
	require.Empty(testingT, avatarData)
	require.Empty(testingT, contentType)
}

func TestFetchAvatarRejectsInvalidURL(testingT *testing.T) {
	authManager := &AuthManager{httpClient: &http.Client{}}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), "://invalid")
	require.Error(testingT, fetchErr)
	require.Empty(testingT, avatarData)
	require.Empty(testingT, contentType)
}

func TestFetchAvatarRejectsNonOKStatus(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("missing")),
				Header:     http.Header{},
			}, nil
		}),
	}
	authManager := &AuthManager{httpClient: client}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), testAvatarURL)
	require.Error(testingT, fetchErr)
	require.Empty(testingT, avatarData)
	require.Empty(testingT, contentType)
}

func TestFetchAvatarDefaultsContentType(testingT *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(testAvatarBytesValue)),
				Header:     http.Header{},
			}, nil
		}),
	}
	authManager := &AuthManager{httpClient: client}

	avatarData, contentType, fetchErr := authManager.fetchAvatar(context.Background(), testAvatarURL)
	require.NoError(testingT, fetchErr)
	require.Equal(testingT, []byte(testAvatarBytesValue), avatarData)
	require.Equal(testingT, defaultAvatarMimeType, contentType)
}

func TestEnsureUserUsesCurrentUserFromContext(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	currentUser := &CurrentUser{Email: testAuthUserEmail, Role: RoleUser}
	context.Set(contextKeyCurrentUser, currentUser)

	authManager := &AuthManager{logger: zap.NewNop()}
	returnedUser, ok := authManager.ensureUser(context)
	require.True(testingT, ok)
	require.Equal(testingT, currentUser, returnedUser)
}

func TestEnsureUserRejectsWhenSessionValidatorMissing(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	authManager := &AuthManager{logger: zap.NewNop()}
	_, ok := authManager.ensureUser(context)
	require.False(testingT, ok)
}

func TestRequireAuthenticatedJSONAllowsWhenUserPresent(testingT *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: testAuthUserEmail, Role: RoleUser})

	authManager := &AuthManager{logger: zap.NewNop()}
	handler := authManager.RequireAuthenticatedJSON()
	handler(context)
	require.False(testingT, context.IsAborted())
}

func TestNewAuthManagerDefaultsDependencies(testingT *testing.T) {
	manager, managerErr := NewAuthManager(nil, zap.NewNop(), nil, nil, AuthConfig{SigningKey: "signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(testingT, managerErr)
	require.NotNil(testingT, manager.httpClient)
}

func TestNewAuthManagerSkipsBlankAdminEmail(testingT *testing.T) {
	manager, managerErr := NewAuthManager(nil, zap.NewNop(), []string{testAuthBlankEmail, testAuthUserEmail}, nil, AuthConfig{SigningKey: testAuthSigningKeyValue, CookieName: testAuthCookieNameValue})
	require.NoError(testingT, managerErr)
	require.Len(testingT, manager.adminEmails, 1)
	_, exists := manager.adminEmails[testAuthUserEmail]
	require.True(testingT, exists)
}

func TestFetchAvatarReportsMissingHTTPClient(testingT *testing.T) {
	manager := &AuthManager{}
	_, _, fetchErr := manager.fetchAvatar(context.Background(), testAvatarURL)
	require.Error(testingT, fetchErr)
	require.Contains(testingT, fetchErr.Error(), "http client is not configured")
}
