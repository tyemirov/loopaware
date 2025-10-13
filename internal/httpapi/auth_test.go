package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/temirov/GAuss/pkg/session"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const testLoginRedirectPath = "/landing"

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
	session.NewSession([]byte("12345678901234567890123456789012"))
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	client := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x01, 0x02}}
	manager := NewAuthManager(database, zap.NewNop(), nil, client, testLoginRedirectPath)

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
	session.NewSession([]byte("12345678901234567890123456789012"))
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	client := &stubHTTPClient{err: errors.New("network failure")}
	manager := NewAuthManager(database, zap.NewNop(), nil, client, testLoginRedirectPath)

	path, err := manager.persistUser(context.Background(), "user2@example.com", "User Example", "https://example.com/avatar.png")
	require.NoError(t, err)
	require.Empty(t, path)

	var user model.User
	require.NoError(t, database.First(&user, "email = ?", "user2@example.com").Error)
	require.Empty(t, user.AvatarData)
}

func TestPersistUserDoesNotRefetchWhenSourceUnchanged(t *testing.T) {
	session.NewSession([]byte("12345678901234567890123456789012"))
	sqliteDatabase := testutil.NewSQLiteTestDatabase(t)
	database, err := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(t, err)
	database = testutil.ConfigureDatabaseLogger(t, database)
	require.NoError(t, storage.AutoMigrate(database))

	firstClient := &stubHTTPClient{statusCode: http.StatusOK, contentType: "image/png", body: []byte{0x01}}
	manager := NewAuthManager(database, zap.NewNop(), nil, firstClient, testLoginRedirectPath)

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
