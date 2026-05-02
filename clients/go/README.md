# LoopAware LA Sentry Go Client

The Go LA Sentry client package lives at `clients/go/lasentry`.

```go
import "github.com/MarkoPoloResearchLab/loopaware/clients/go/lasentry"
```

```go
client, err := lasentry.NewClient(lasentry.Config{
    Endpoint:    "https://loopaware.mprlab.com/sentry/errors",
    SiteID:      "YOUR_SITE_ID",
    Token:       "YOUR_SERVER_SIDE_TOKEN",
    Environment: "production",
    Release:     "2026.04.24",
})
if err != nil {
    return err
}
err = client.CaptureError(ctx, errValue, lasentry.Attributes{
    Tags: map[string]string{"service": "checkout"},
})
```

For HTTP services, wrap a handler with `client.Middleware(next)` to capture panics before returning a `500` response.
