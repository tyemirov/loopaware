package storage

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// DefaultSiteCreatorEmail identifies the owner to assign when historical sites lack creator attribution.
const DefaultSiteCreatorEmail = "temirov@gmail.com"

func backfillSiteCreatorEmails(database *gorm.DB) error {
	normalizedCreatorEmail := strings.ToLower(strings.TrimSpace(DefaultSiteCreatorEmail))
	if normalizedCreatorEmail == "" {
		return errors.New("storage: default site creator email is empty")
	}

	assignments := map[string]any{
		"creator_email": normalizedCreatorEmail,
	}

	return database.Model(&model.Site{}).
		Where("creator_email IS NULL OR TRIM(creator_email) = ''").
		Updates(assignments).Error
}

func backfillSiteWidgetFeedbackVisibility(database *gorm.DB) error {
	assignments := map[string]any{
		"widget_show_message_input":     true,
		"widget_show_sentiment_buttons": true,
	}

	return database.Model(&model.Site{}).
		Where(
			"widget_show_message_input IS NULL OR widget_show_sentiment_buttons IS NULL OR (widget_show_message_input = ? AND widget_show_sentiment_buttons = ?)",
			false,
			false,
		).
		Updates(assignments).Error
}

func backfillSiteWidgetAccentColors(database *gorm.DB) error {
	assignments := map[string]any{
		"widget_accent_color": "#0d6efd",
	}

	return database.Model(&model.Site{}).
		Where("widget_accent_color IS NULL OR TRIM(widget_accent_color) = ''").
		Updates(assignments).Error
}

func backfillFeedbackSourceKinds(database *gorm.DB) error {
	assignments := map[string]any{
		"source_kind": model.FeedbackSourceWebWidget,
	}

	return database.Model(&model.Feedback{}).
		Where("source_kind IS NULL OR TRIM(source_kind) = ''").
		Updates(assignments).Error
}
