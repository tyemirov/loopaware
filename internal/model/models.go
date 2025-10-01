package model

import "time"

type Site struct {
	ID            string    `gorm:"primaryKey;size:36"`
	Name          string    `gorm:"not null;size:200"`
	AllowedOrigin string    `gorm:"not null;size:500"`
	OwnerEmail    string    `gorm:"size:320"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

type Feedback struct {
	ID        string    `gorm:"primaryKey;size:36"`
	SiteID    string    `gorm:"index;not null;size:36"`
	Contact   string    `gorm:"not null;size:320"`
	Message   string    `gorm:"not null;size:4000"`
	IP        string    `gorm:"size:64"`
	UserAgent string    `gorm:"size:400"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
