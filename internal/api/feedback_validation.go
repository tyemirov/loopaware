package api

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

const (
	feedbackPhoneMinimumDigits = 10
	feedbackPhoneMaximumDigits = 15
)

var errInvalidFeedbackContact = errors.New("invalid_feedback_contact")

func normalizeFeedbackContact(rawInput string) (string, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty", errInvalidFeedbackContact)
	}

	if strings.Contains(trimmed, "@") {
		return normalizeFeedbackEmail(trimmed)
	}

	return normalizeFeedbackPhone(trimmed)
}

func normalizeFeedbackEmail(rawInput string) (string, error) {
	parsedAddress, parseErr := mail.ParseAddress(rawInput)
	if parseErr != nil {
		return "", fmt.Errorf("%w: %v", errInvalidFeedbackContact, parseErr)
	}

	normalizedAddress := strings.ToLower(strings.TrimSpace(parsedAddress.Address))
	if normalizedAddress == "" {
		return "", fmt.Errorf("%w: empty email", errInvalidFeedbackContact)
	}

	if normalizedAddress != strings.ToLower(strings.TrimSpace(rawInput)) {
		return "", fmt.Errorf("%w: email must not include display name", errInvalidFeedbackContact)
	}

	return normalizedAddress, nil
}

func normalizeFeedbackPhone(rawInput string) (string, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty phone", errInvalidFeedbackContact)
	}

	hasLeadingPlus := false
	var digitsBuilder strings.Builder
	for characterIndex, characterValue := range trimmed {
		switch {
		case characterValue >= '0' && characterValue <= '9':
			digitsBuilder.WriteRune(characterValue)
		case characterValue == '+':
			if characterIndex != 0 || hasLeadingPlus {
				return "", fmt.Errorf("%w: misplaced plus", errInvalidFeedbackContact)
			}
			hasLeadingPlus = true
		case characterValue == ' ' || characterValue == '-' || characterValue == '.' || characterValue == '(' || characterValue == ')':
			continue
		default:
			return "", fmt.Errorf("%w: invalid phone character %q", errInvalidFeedbackContact, characterValue)
		}
	}

	digitsOnlyValue := digitsBuilder.String()
	if len(digitsOnlyValue) < feedbackPhoneMinimumDigits || len(digitsOnlyValue) > feedbackPhoneMaximumDigits {
		return "", fmt.Errorf("%w: phone must contain %d-%d digits", errInvalidFeedbackContact, feedbackPhoneMinimumDigits, feedbackPhoneMaximumDigits)
	}

	if hasLeadingPlus {
		return "+" + digitsOnlyValue, nil
	}
	return digitsOnlyValue, nil
}
