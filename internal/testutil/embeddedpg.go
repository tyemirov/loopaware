package testutil

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

var (
	startOnce sync.Once
	stopOnce  sync.Once

	epg      *embeddedpostgres.EmbeddedPostgres
	dsn      string
	startErr error
)

// StartEmbeddedPostgresOnce starts a real Postgres in-process exactly once.
// Safe to call from multiple packages' TestMain.
func StartEmbeddedPostgresOnce() error {
	startOnce.Do(func() {
		port, err := pickFreePort()
		if err != nil {
			startErr = fmt.Errorf("embedded-pg: find free port: %w", err)
			return
		}

		// Ensure each test *process* gets its own directories to avoid cross-process races.
		base := filepath.Join(os.TempDir(), fmt.Sprintf("feedbacksvc-embedded-pg-%d", os.Getpid()))
		_ = os.MkdirAll(base, 0o755)
		_ = os.MkdirAll(filepath.Join(base, "data"), 0o755)
		_ = os.MkdirAll(filepath.Join(base, "runtime"), 0o755)
		_ = os.MkdirAll(filepath.Join(base, "binaries"), 0o755)

		cfg := embeddedpostgres.DefaultConfig().
			Port(uint32(port)).
			Database("feedback").
			Username("feedback_user").
			Password("feedback_password").
			DataPath(filepath.Join(base, "data")).
			RuntimePath(filepath.Join(base, "runtime")).
			BinariesPath(filepath.Join(base, "binaries")).
			StartTimeout(2 * time.Minute)

		db := embeddedpostgres.NewDatabase(cfg)
		if err := db.Start(); err != nil {
			startErr = fmt.Errorf("embedded-pg: start: %w", err)
			return
		}

		epg = db
		dsn = fmt.Sprintf(
			"host=127.0.0.1 port=%d user=feedback_user password=feedback_password dbname=feedback sslmode=disable TimeZone=UTC",
			port,
		)
	})
	return startErr
}

// StopEmbeddedPostgresOnce stops the embedded Postgres if it was started.
func StopEmbeddedPostgresOnce() {
	stopOnce.Do(func() {
		if epg != nil {
			_ = epg.Stop()
		}
	})
}

// DSN returns the connection string to the embedded Postgres (after StartEmbeddedPostgresOnce).
func DSN() string {
	return dsn
}

func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
