package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

const (
	SiteRecipientModeManager  = "manager"
	SiteRecipientModeTeam     = "team"
	SiteRecipientModeSelected = "selected"

	SiteRecipientEmailMaxLength = 320
)

var ErrInvalidSiteRecipient = errors.New("invalid_site_recipient")

// NormalizeSiteRecipientMode returns the canonical site notification recipient mode.
func NormalizeSiteRecipientMode(rawMode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(rawMode)) {
	case "", SiteRecipientModeManager:
		return SiteRecipientModeManager, nil
	case SiteRecipientModeTeam:
		return SiteRecipientModeTeam, nil
	case SiteRecipientModeSelected:
		return SiteRecipientModeSelected, nil
	default:
		return "", fmt.Errorf("%w: invalid recipient_mode", ErrInvalidSiteRecipient)
	}
}

// NormalizeSiteRecipientEmail returns the canonical recipient email value.
func NormalizeSiteRecipientEmail(rawRecipient string) (string, error) {
	recipient := strings.ToLower(strings.TrimSpace(rawRecipient))
	if recipient == "" {
		return "", fmt.Errorf("%w: missing recipient_email", ErrInvalidSiteRecipient)
	}
	parsedAddress, parseErr := mail.ParseAddress(recipient)
	if parseErr != nil || parsedAddress.Address != recipient {
		return "", fmt.Errorf("%w: invalid recipient_email", ErrInvalidSiteRecipient)
	}
	if len(recipient) > SiteRecipientEmailMaxLength {
		return "", fmt.Errorf("%w: recipient_email too long", ErrInvalidSiteRecipient)
	}
	return recipient, nil
}

// NormalizeSiteRecipientEmailList normalizes and de-duplicates selected recipient emails.
func NormalizeSiteRecipientEmailList(rawRecipients []string) ([]string, error) {
	normalizedRecipients := make([]string, 0, len(rawRecipients))
	seenRecipients := make(map[string]struct{}, len(rawRecipients))
	for _, rawRecipient := range rawRecipients {
		recipient, recipientErr := NormalizeSiteRecipientEmail(rawRecipient)
		if recipientErr != nil {
			return nil, recipientErr
		}
		if _, exists := seenRecipients[recipient]; exists {
			continue
		}
		seenRecipients[recipient] = struct{}{}
		normalizedRecipients = append(normalizedRecipients, recipient)
	}
	return normalizedRecipients, nil
}

// EncodeSiteRecipientEmails serializes selected recipient emails.
func EncodeSiteRecipientEmails(recipientEmails []string) (string, error) {
	if len(recipientEmails) == 0 {
		return "[]", nil
	}
	encoded, encodeErr := json.Marshal(recipientEmails)
	if encodeErr != nil {
		return "", fmt.Errorf("%w: invalid recipient_emails", ErrInvalidSiteRecipient)
	}
	return string(encoded), nil
}

// DecodeSiteRecipientEmails decodes and normalizes a persisted selected-recipient list.
func DecodeSiteRecipientEmails(rawValue string) []string {
	if strings.TrimSpace(rawValue) == "" {
		return []string{}
	}
	var recipientEmails []string
	if unmarshalErr := json.Unmarshal([]byte(rawValue), &recipientEmails); unmarshalErr != nil {
		return []string{}
	}
	normalizedEmails, normalizeErr := NormalizeSiteRecipientEmailList(recipientEmails)
	if normalizeErr != nil {
		return []string{}
	}
	return normalizedEmails
}
