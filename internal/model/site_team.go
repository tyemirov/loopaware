package model

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	siteTeamMemberSiteIDMaxLength = 36
	siteTeamMemberEmailMaxLength  = 320
)

var ErrInvalidSiteTeamMember = errors.New("invalid_site_team_member")

// SiteTeamMember grants one authenticated email read access to one site.
type SiteTeamMember struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SiteID       string    `gorm:"not null;size:36;index;uniqueIndex:idx_site_team_members_site_email,priority:1"`
	Email        string    `gorm:"not null;size:320;index;uniqueIndex:idx_site_team_members_site_email,priority:2"`
	AddedByEmail string    `gorm:"size:320"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// SiteTeamMemberInput captures raw values for a per-site team membership.
type SiteTeamMemberInput struct {
	SiteID       string
	Email        string
	AddedByEmail string
}

// NewSiteTeamMember validates and normalizes a per-site team member.
func NewSiteTeamMember(input SiteTeamMemberInput) (SiteTeamMember, error) {
	siteID := strings.TrimSpace(input.SiteID)
	if siteID == "" || len(siteID) > siteTeamMemberSiteIDMaxLength {
		return SiteTeamMember{}, fmt.Errorf("%w: invalid site_id", ErrInvalidSiteTeamMember)
	}

	email, emailErr := NormalizeSiteTeamMemberEmail(input.Email)
	if emailErr != nil {
		return SiteTeamMember{}, emailErr
	}

	addedByEmail := strings.ToLower(strings.TrimSpace(input.AddedByEmail))
	if addedByEmail != "" {
		if _, parseErr := mail.ParseAddress(addedByEmail); parseErr != nil || len(addedByEmail) > siteTeamMemberEmailMaxLength {
			return SiteTeamMember{}, fmt.Errorf("%w: invalid added_by_email", ErrInvalidSiteTeamMember)
		}
	}

	return SiteTeamMember{
		ID:           uuid.NewString(),
		SiteID:       siteID,
		Email:        email,
		AddedByEmail: addedByEmail,
	}, nil
}

// NormalizeSiteTeamMemberEmail returns the canonical team-member email value.
func NormalizeSiteTeamMemberEmail(rawInput string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(rawInput))
	if email == "" || len(email) > siteTeamMemberEmailMaxLength {
		return "", fmt.Errorf("%w: invalid email", ErrInvalidSiteTeamMember)
	}
	parsedAddress, parseErr := mail.ParseAddress(email)
	if parseErr != nil || parsedAddress.Address != email {
		return "", fmt.Errorf("%w: invalid email", ErrInvalidSiteTeamMember)
	}
	return email, nil
}
