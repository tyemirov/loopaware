package api

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

func sitesVisibleToUserQuery(ctx context.Context, database *gorm.DB, currentUser *CurrentUser) *gorm.DB {
	query := database.WithContext(ctx).Model(&model.Site{})
	if currentUser.hasRole(RoleAdmin) {
		return query
	}
	normalizedEmail := currentUser.normalizedEmail()
	teamMemberships := database.WithContext(ctx).Model(&model.SiteTeamMember{}).Select("site_id").Where("email = ?", normalizedEmail)
	return query.Where("(LOWER(owner_email) = ? OR LOWER(creator_email) = ? OR id IN (?))", normalizedEmail, normalizedEmail, teamMemberships)
}

func currentUserCanViewSite(ctx context.Context, database *gorm.DB, currentUser *CurrentUser, site model.Site) bool {
	if currentUser == nil {
		return false
	}
	if currentUser.canManageSite(site) {
		return true
	}
	if database == nil {
		return false
	}
	normalizedEmail := currentUser.normalizedEmail()
	if normalizedEmail == "" || strings.TrimSpace(site.ID) == "" {
		return false
	}
	var membership model.SiteTeamMember
	err := database.WithContext(ctx).
		Select("id").
		Where("site_id = ? AND email = ?", site.ID, normalizedEmail).
		First(&membership).Error
	return err == nil
}
