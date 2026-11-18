//go:build live

package notifications

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPingwinLiveEmail(t *testing.T) {
	address := os.Getenv("PINGUIN_ADDR")
	apiKey := os.Getenv("PINGUIN_TENANT_API_KEY")
	recipient := os.Getenv("PINGUIN_TEST_RECIPIENT")
	if address == "" || apiKey == "" || recipient == "" {
		t.Skip("set the approved Pingwin address, tenant API key, and recipient for live acceptance")
	}
	notifier, err := NewPinguinNotifier(zap.NewExample(), PinguinConfig{
		Address: address, APIKey: apiKey,
		OperationTimeout: 30 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, notifier.Close()) })
	subject := "MPR-DMARC-" + time.Now().UTC().Format("20060102") + "-LOOPAWARE-ADAPTER"
	err = notifier.SendEmail(context.Background(), recipient, subject, "Authorized MPR Lab email route verification through the LoopAware application notifier.")
	require.NoError(t, err)
}
