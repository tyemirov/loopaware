package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	testingpkg "testing"

	"github.com/stretchr/testify/require"
)

const (
	listMessagesSiteName          = "List Messages Site"
	listMessagesAllowedOrigin     = "http://listmessages.example"
	adminCreateSitePath           = "/api/admin/sites"
	adminListMessagesPathTemplate = "/api/admin/sites/%s/messages"
	publicFeedbackPath            = "/api/feedback"
	originHeaderName              = "Origin"
	jsonFieldSiteIdentifier       = "site_id"
	jsonFieldName                 = "name"
	jsonFieldAllowedOrigin        = "allowed_origin"
	jsonFieldContact              = "contact"
	jsonFieldMessage              = "message"
)

type feedbackSubmission struct {
	contact string
	message string
}

func TestAdminListMessagesBySiteReturnsOrderedUnixTimestamps(testingT *testingpkg.T) {
	testCases := []struct {
		name                 string
		feedbackSubmissions  []feedbackSubmission
		expectedMessageCount int
	}{
		{
			name: "single feedback message",
			feedbackSubmissions: []feedbackSubmission{
				{contact: "single@example.com", message: "Only message"},
			},
			expectedMessageCount: 1,
		},
		{
			name: "multiple feedback messages",
			feedbackSubmissions: []feedbackSubmission{
				{contact: "first@example.com", message: "First feedback"},
				{contact: "second@example.com", message: "Second feedback"},
			},
			expectedMessageCount: 2,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		testingT.Run(testCase.name, func(t *testingpkg.T) {
			apiHarness := buildAPIHarness(t)

			adminHeaders := map[string]string{
				authorizationHeaderName: bearerTokenPrefix + apiHarness.adminBearerToken,
			}

			createSiteRecorder := performJSONRequest(t, apiHarness.router, http.MethodPost, adminCreateSitePath, map[string]string{
				jsonFieldName:          listMessagesSiteName,
				jsonFieldAllowedOrigin: listMessagesAllowedOrigin,
			}, adminHeaders)
			require.Equal(t, http.StatusOK, createSiteRecorder.Code)

			var createSiteResponse struct {
				Identifier string `json:"id"`
			}
			require.NoError(t, json.Unmarshal(createSiteRecorder.Body.Bytes(), &createSiteResponse))
			require.NotEmpty(t, createSiteResponse.Identifier)

			for _, submission := range testCase.feedbackSubmissions {
				feedbackPayload := map[string]any{
					jsonFieldSiteIdentifier: createSiteResponse.Identifier,
					jsonFieldContact:        submission.contact,
					jsonFieldMessage:        submission.message,
				}
				feedbackHeaders := map[string]string{
					originHeaderName: listMessagesAllowedOrigin,
				}
				feedbackRecorder := performJSONRequest(t, apiHarness.router, http.MethodPost, publicFeedbackPath, feedbackPayload, feedbackHeaders)
				require.Equal(t, http.StatusOK, feedbackRecorder.Code)
			}

			listMessagesPath := fmt.Sprintf(adminListMessagesPathTemplate, createSiteResponse.Identifier)
			listRecorder := performJSONRequest(t, apiHarness.router, http.MethodGet, listMessagesPath, nil, adminHeaders)
			require.Equal(t, http.StatusOK, listRecorder.Code)

			var listResponse struct {
				SiteID   string `json:"site_id"`
				Messages []struct {
					Identifier string `json:"id"`
					Contact    string `json:"contact"`
					Message    string `json:"message"`
					IP         string `json:"ip"`
					UserAgent  string `json:"user_agent"`
					CreatedAt  int64  `json:"created_at"`
				} `json:"messages"`
			}
			require.NoError(t, json.Unmarshal(listRecorder.Body.Bytes(), &listResponse))

			require.Equal(t, createSiteResponse.Identifier, listResponse.SiteID)
			require.Len(t, listResponse.Messages, testCase.expectedMessageCount)

			expectedSubmissions := make(map[feedbackSubmission]int)
			for _, submission := range testCase.feedbackSubmissions {
				expectedSubmissions[submission]++
			}

			for messageIndex := range listResponse.Messages {
				message := listResponse.Messages[messageIndex]
				require.NotEmpty(t, message.Identifier)
				require.Greater(t, message.CreatedAt, int64(0))
				if messageIndex > 0 {
					previousMessage := listResponse.Messages[messageIndex-1]
					require.GreaterOrEqual(t, previousMessage.CreatedAt, message.CreatedAt)
				}
				submissionKey := feedbackSubmission{contact: message.Contact, message: message.Message}
				remainingCount, exists := expectedSubmissions[submissionKey]
				require.True(t, exists)
				require.Greater(t, remainingCount, 0)
				expectedSubmissions[submissionKey] = remainingCount - 1
			}

			for submissionKey, remainingCount := range expectedSubmissions {
				require.Equal(t, 0, remainingCount, "missing submission %#v", submissionKey)
			}
		})
	}
}
