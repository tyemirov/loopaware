package httpapi_test

import (
	"fmt"
	"os"
	testingpkg "testing"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

func TestMain(m *testingpkg.M) {
	if err := testutil.StartEmbeddedPostgresOnce(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start embedded Postgres: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	testutil.StopEmbeddedPostgresOnce()
	os.Exit(code)
}
