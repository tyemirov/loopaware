package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

type stubHTTPClient struct {
	statusCode  int
	contentType string
	body        []byte
	err         error
}

func (client *stubHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if client.err != nil {
		return nil, client.err
	}
	response := &http.Response{
		StatusCode: client.statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(client.body)),
	}
	if client.contentType != "" {
		response.Header.Set("Content-Type", client.contentType)
	}
	return response, nil
}

func TestPersistUserStoresAvatar(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	client := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x01, 0x02}}
	manager, err := NewAuthManager(database, zap.NewNop(), nil, client, AuthConfig{SigningKey: "test-signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(t, err)

	path, err := manager.persistUser(context.Background(), "user@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(t, err)
	require.NotEmpty(t, path)
	require.Contains(t, path, avatarEndpointPath)

	var user model.User
	require.NoError(t, database.First(&user, "email = ?", "user@example.com").Error)
	require.Equal(t, []byte{0x01, 0x02}, user.AvatarData)
	require.Equal(t, "image/png", user.AvatarContentType)
	require.Equal(t, "https://example.com/avatar.png", user.PictureSourceURL)
}

func TestPersistUserHandlesFetchErrorsGracefully(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	client := &stubHTTPClient{err: errors.New("network failure")}
	manager, err := NewAuthManager(database, zap.NewNop(), nil, client, AuthConfig{SigningKey: "test-signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(t, err)

	path, err := manager.persistUser(context.Background(), "user2@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(t, err)
	require.Empty(t, path)

	var user model.User
	require.NoError(t, database.First(&user, "email = ?", "user2@example.com").Error)
	require.Empty(t, user.AvatarData)
}

func TestPersistUserDoesNotRefetchWhenSourceUnchanged(t *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	firstClient := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x01}}
	manager, err := NewAuthManager(database, zap.NewNop(), nil, firstClient, AuthConfig{SigningKey: "test-signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(t, err)

	_, err = manager.persistUser(context.Background(), "user3@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(t, err)

	secondClient := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0xFF}}
	manager.httpClient = secondClient

	_, err = manager.persistUser(context.Background(), "user3@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(t, err)

	var user model.User
	require.NoError(t, database.First(&user, "email = ?", "user3@example.com").Error)
	require.Equal(t, []byte{0x01}, user.AvatarData)
}

func TestPersistUserDoesNotUpdateWhenUnchanged(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	client := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x01, 0x02}}
	manager, err := NewAuthManager(database, zap.NewNop(), nil, client, AuthConfig{SigningKey: "test-signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(testingT, err)

	_, err = manager.persistUser(context.Background(), "user4@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(testingT, err)

	updateCount := 0
	callbackName := "count_user_updates"
	database.Callback().Update().Before("gorm:update").Register(callbackName, func(database *gorm.DB) {
		if database.Statement != nil && database.Statement.Table == "users" {
			updateCount++
		}
	})
	testingT.Cleanup(func() {
		database.Callback().Update().Remove(callbackName)
	})

	_, err = manager.persistUser(context.Background(), "user4@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(testingT, err)
	require.Equal(testingT, 0, updateCount)
}

func TestPersistUserCoalescesNullAvatarSizeSnapshot(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, err)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	failingClient := &stubHTTPClient{err: errors.New("network failure")}
	manager, err := NewAuthManager(database, zap.NewNop(), nil, failingClient, AuthConfig{SigningKey: "test-signing-key", CookieName: testAuthCookieNameValue})
	require.NoError(testingT, err)

	path, err := manager.persistUser(context.Background(), "user5@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(testingT, err)
	require.Empty(testingT, path)

	require.NoError(testingT, database.Exec("UPDATE users SET avatar_data = NULL WHERE email = ?", "user5@example.com").Error)

	manager.httpClient = &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x03, 0x04}}

	path, err = manager.persistUser(context.Background(), "user5@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(testingT, err)
	require.NotEmpty(testingT, path)
	require.Contains(testingT, path, avatarEndpointPath)

	var user model.User
	require.NoError(testingT, database.First(&user, "email = ?", "user5@example.com").Error)
	require.Equal(testingT, []byte{0x03, 0x04}, user.AvatarData)
	require.Equal(testingT, "image/png", user.AvatarContentType)
	require.Equal(testingT, "https://example.com/avatar.png", user.PictureSourceURL)
}
