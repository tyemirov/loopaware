package model

import "time"

const (
	FeedbackDeliveryNone   = "no"
	FeedbackDeliveryMailed = "mailed"
	FeedbackDeliveryTexted = "texted"
)

type Site struct {
	ID                         string `gorm:"primaryKey;size:36"`
	Name                       string `gorm:"not null;size:200"`
	AllowedOrigin              string `gorm:"not null;size:500"`
	SubscribeAllowedOrigins    string `gorm:"size:500"`
	OwnerEmail                 string `gorm:"size:320"`
	CreatorEmail               string `gorm:"size:320"`
	WidgetBubbleSide           string `gorm:"not null;size:16;default:right"`
	WidgetBubbleBottomOffsetPx int    `gorm:"not null;default:16"`
	FaviconData                []byte `gorm:"type:blob"`
	FaviconContentType         string `gorm:"size:100"`
	FaviconFetchedAt           time.Time
	FaviconLastAttemptAt       time.Time
	FaviconOrigin              string    `gorm:"size:500"`
	CreatedAt                  time.Time `gorm:"autoCreateTime"`
	UpdatedAt                  time.Time `gorm:"autoUpdateTime"`
}

type Feedback struct {
	ID        string    `gorm:"primaryKey;size:36"`
	SiteID    string    `gorm:"index;not null;size:36"`
	Contact   string    `gorm:"not null;size:320"`
	Message   string    `gorm:"not null;size:4000"`
	IP        string    `gorm:"size:64"`
	UserAgent string    `gorm:"size:400"`
	Delivery  string    `gorm:"not null;size:16;default:no"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type User struct {
	Email             string    `gorm:"primaryKey;size:320"`
	Name              string    `gorm:"not null;size:320"`
	PictureSourceURL  string    `gorm:"size:500"`
	AvatarContentType string    `gorm:"size:100"`
	AvatarData        []byte    `gorm:"type:blob"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}
