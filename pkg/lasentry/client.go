package lasentry

import client "github.com/MarkoPoloResearchLab/loopaware/clients/go/lasentry"

// HTTPClient executes outbound HTTP requests.
type HTTPClient = client.HTTPClient

// Config describes a LoopAware LA Sentry client.
type Config = client.Config

// Client submits developer error events to LoopAware.
type Client = client.Client

// Attributes captures optional event context.
type Attributes = client.Attributes

// StackFrame describes one client-side stack frame.
type StackFrame = client.StackFrame

// NewClient constructs a LoopAware LA Sentry client.
func NewClient(config Config) (*Client, error) {
	return client.NewClient(config)
}
