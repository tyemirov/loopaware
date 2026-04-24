package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/lasentry"
)

func main() {
	client, clientErr := lasentry.NewClient(lasentry.Config{
		Endpoint:    os.Getenv("LOOPAWARE_LA_SENTRY_ENDPOINT"),
		SiteID:      os.Getenv("LOOPAWARE_LA_SENTRY_SITE_ID"),
		Token:       os.Getenv("LOOPAWARE_LA_SENTRY_TOKEN"),
		Environment: "integration-go",
		Release:     "2026.04.24-go",
	})
	if clientErr != nil {
		exitWithError(clientErr)
	}

	handler := client.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("go client middleware capture failed")
	}))
	request := httptest.NewRequest(http.MethodGet, "https://go-client.example/oauth/callback?code=secret&state=hidden#token", nil)
	request.Host = "go-client.example"
	request.RemoteAddr = "192.0.2.22:1234"
	request.Header.Set("User-Agent", "LoopAware Go Client Integration")
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusInternalServerError {
		exitWithError(fmt.Errorf("expected middleware status 500, got %d", responseRecorder.Code))
	}

	encodedStatus, encodeErr := json.Marshal(map[string]string{"status": "ok"})
	if encodeErr != nil {
		exitWithError(encodeErr)
	}
	fmt.Println(string(encodedStatus))
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
