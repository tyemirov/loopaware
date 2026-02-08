package api

import "context"

// EmailSender sends an email message to a recipient.
type EmailSender interface {
	SendEmail(ctx context.Context, recipient string, subject string, message string) error
}
