package httpapi_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newHTTPTestServer(testingT *testing.T, handler http.Handler) *httptest.Server {
	testingT.Helper()

	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		testingT.Skipf("network listener unavailable: %v", listenErr)
	}
	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
	server.Start()
	testingT.Cleanup(server.Close)
	return server
}
