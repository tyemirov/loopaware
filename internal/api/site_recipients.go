package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type siteRecipientConfig struct {
	recipientEmail   string
	recipientMode    string
	recipientEmails  []string
	noRecipientError string
}

func siteNotificationRecipients(ctx context.Context, database *gorm.DB, site model.Site, config siteRecipientConfig) ([]string, error) {
	switch config.recipientMode {
	case model.SiteRecipientModeManager:
		return appendSiteRecipient([]string{}, map[string]struct{}{}, config.recipientEmail)
	case model.SiteRecipientModeTeam:
		return siteTeamRecipients(ctx, database, site, config.noRecipientError)
	case model.SiteRecipientModeSelected:
		return siteSelectedRecipients(ctx, database, site.ID, config.recipientEmails, config.noRecipientError)
	default:
		return nil, fmt.Errorf("%w: invalid recipient_mode", model.ErrInvalidSiteRecipient)
	}
}

func siteTeamRecipients(ctx context.Context, database *gorm.DB, site model.Site, noRecipientError string) ([]string, error) {
	recipients := []string{}
	seenRecipients := map[string]struct{}{}
	var appendErr error
	recipients, appendErr = appendSiteRecipient(recipients, seenRecipients, site.OwnerEmail)
	if appendErr != nil {
		return nil, appendErr
	}
	recipients, appendErr = appendSiteRecipient(recipients, seenRecipients, site.CreatorEmail)
	if appendErr != nil {
		return nil, appendErr
	}
	teamMembers, teamMemberErr := siteTeamMemberRecipients(ctx, database, site.ID)
	if teamMemberErr != nil {
		return nil, teamMemberErr
	}
	for _, teamMemberEmail := range teamMembers {
		recipients, appendErr = appendSiteRecipient(recipients, seenRecipients, teamMemberEmail)
		if appendErr != nil {
			return nil, appendErr
		}
	}
	if len(recipients) == 0 {
		return nil, errors.New(strings.TrimSpace(noRecipientError))
	}
	return recipients, nil
}

func siteSelectedRecipients(ctx context.Context, database *gorm.DB, siteID string, selectedRecipients []string, noRecipientError string) ([]string, error) {
	teamRecipientSet, teamRecipientErr := siteTeamMemberRecipientSet(ctx, database, siteID)
	if teamRecipientErr != nil {
		return nil, teamRecipientErr
	}
	recipients := []string{}
	seenRecipients := map[string]struct{}{}
	for _, selectedRecipient := range selectedRecipients {
		if _, exists := teamRecipientSet[selectedRecipient]; !exists {
			continue
		}
		var appendErr error
		recipients, appendErr = appendSiteRecipient(recipients, seenRecipients, selectedRecipient)
		if appendErr != nil {
			return nil, appendErr
		}
	}
	if len(recipients) == 0 {
		return nil, errors.New(strings.TrimSpace(noRecipientError))
	}
	return recipients, nil
}

func siteTeamMemberRecipientSet(ctx context.Context, database *gorm.DB, siteID string) (map[string]struct{}, error) {
	teamMemberRecipients, teamMemberErr := siteTeamMemberRecipients(ctx, database, siteID)
	if teamMemberErr != nil {
		return nil, teamMemberErr
	}
	recipientSet := make(map[string]struct{}, len(teamMemberRecipients))
	for _, recipient := range teamMemberRecipients {
		recipientSet[recipient] = struct{}{}
	}
	return recipientSet, nil
}

func siteTeamMemberRecipients(ctx context.Context, database *gorm.DB, siteID string) ([]string, error) {
	var teamMembers []model.SiteTeamMember
	if queryErr := database.WithContext(ctx).Where("site_id = ?", siteID).Order("email asc").Find(&teamMembers).Error; queryErr != nil {
		return nil, queryErr
	}
	recipients := make([]string, 0, len(teamMembers))
	seenRecipients := map[string]struct{}{}
	for _, teamMember := range teamMembers {
		var appendErr error
		recipients, appendErr = appendSiteRecipient(recipients, seenRecipients, teamMember.Email)
		if appendErr != nil {
			return nil, appendErr
		}
	}
	return recipients, nil
}

func appendSiteRecipient(recipients []string, seenRecipients map[string]struct{}, rawRecipient string) ([]string, error) {
	recipient, recipientErr := model.NormalizeSiteRecipientEmail(rawRecipient)
	if recipientErr != nil {
		return nil, recipientErr
	}
	if _, exists := seenRecipients[recipient]; exists {
		return recipients, nil
	}
	seenRecipients[recipient] = struct{}{}
	return append(recipients, recipient), nil
}
