package httpapi

import "context"

// EmailSender sends an email message to a recipient.
type EmailSender interface {
	SendEmail(ctx context.Context, recipient string, subject string, message string) error
}

type noopEmailSender struct{}

func (noopEmailSender) SendEmail(ctx context.Context, recipient string, subject string, message string) error {
	return nil
}

func resolveEmailSender(sender EmailSender) EmailSender {
	if sender == nil {
		return noopEmailSender{}
	}
	return sender
}
