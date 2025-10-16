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
