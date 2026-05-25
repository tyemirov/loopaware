package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	PortfolioTrafficReportDefaultID = "all-sites-traffic"

	portfolioTrafficReportNameMaxLength = 120
)

var ErrInvalidPortfolioTrafficReportDefinition = errors.New("invalid_portfolio_traffic_report_definition")

// PortfolioTrafficReportDefinition captures one saved cross-site traffic report.
type PortfolioTrafficReportDefinition struct {
	ID        string    `gorm:"primaryKey;size:36"`
	UserEmail string    `gorm:"not null;size:320;index"`
	Name      string    `gorm:"not null;size:120"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// PortfolioTrafficReportDefinitionSite captures one site included in a saved report.
type PortfolioTrafficReportDefinitionSite struct {
	ID        string    `gorm:"primaryKey;size:36"`
	ReportID  string    `gorm:"not null;size:36;index;uniqueIndex:idx_portfolio_report_site,priority:1"`
	SiteID    string    `gorm:"not null;size:36;index;uniqueIndex:idx_portfolio_report_site,priority:2"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// PortfolioTrafficReportDefinitionInput holds incoming saved-report values.
type PortfolioTrafficReportDefinitionInput struct {
	UserEmail string
	Name      string
}

// NewPortfolioTrafficReportDefinition constructs a validated saved report definition.
func NewPortfolioTrafficReportDefinition(input PortfolioTrafficReportDefinitionInput) (PortfolioTrafficReportDefinition, error) {
	userEmail, emailErr := normalizeTrafficReportRecipient(input.UserEmail)
	if emailErr != nil {
		return PortfolioTrafficReportDefinition{}, fmt.Errorf("%w: invalid user_email", ErrInvalidPortfolioTrafficReportDefinition)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return PortfolioTrafficReportDefinition{}, fmt.Errorf("%w: missing name", ErrInvalidPortfolioTrafficReportDefinition)
	}
	if len(name) > portfolioTrafficReportNameMaxLength {
		return PortfolioTrafficReportDefinition{}, fmt.Errorf("%w: name too long", ErrInvalidPortfolioTrafficReportDefinition)
	}
	return PortfolioTrafficReportDefinition{
		ID:        uuid.NewString(),
		UserEmail: userEmail,
		Name:      name,
	}, nil
}

// NewPortfolioTrafficReportDefinitionSite constructs a validated site membership.
func NewPortfolioTrafficReportDefinitionSite(reportID string, siteID string) (PortfolioTrafficReportDefinitionSite, error) {
	trimmedReportID := strings.TrimSpace(reportID)
	if trimmedReportID == "" || trimmedReportID == PortfolioTrafficReportDefaultID {
		return PortfolioTrafficReportDefinitionSite{}, fmt.Errorf("%w: invalid report_id", ErrInvalidPortfolioTrafficReportDefinition)
	}
	trimmedSiteID := strings.TrimSpace(siteID)
	if trimmedSiteID == "" {
		return PortfolioTrafficReportDefinitionSite{}, fmt.Errorf("%w: invalid site_id", ErrInvalidPortfolioTrafficReportDefinition)
	}
	return PortfolioTrafficReportDefinitionSite{
		ID:       uuid.NewString(),
		ReportID: trimmedReportID,
		SiteID:   trimmedSiteID,
	}, nil
}
