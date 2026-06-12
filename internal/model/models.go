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
	WidgetAllowedOrigins       string `gorm:"size:500"`
	TrafficAllowedOrigins      string `gorm:"size:500"`
	OwnerEmail                 string `gorm:"size:320"`
	CreatorEmail               string `gorm:"size:320"`
	WidgetBubbleSide           string `gorm:"not null;size:16;default:right"`
	WidgetBubbleBottomOffsetPx int    `gorm:"not null;default:16"`
	WidgetAccentColor          string `gorm:"not null;size:7;default:#0d6efd"`
	WidgetShowMessageInput     bool   `gorm:"not null;default:true"`
	WidgetShowSentimentButtons bool   `gorm:"not null;default:true"`
	SentryIngestTokenHash      string `gorm:"size:64"`
	FaviconData                []byte `gorm:"type:blob"`
	FaviconContentType         string `gorm:"size:100"`
	FaviconFetchedAt           time.Time
	FaviconLastAttemptAt       time.Time
	FaviconOrigin              string    `gorm:"size:500"`
	CreatedAt                  time.Time `gorm:"autoCreateTime"`
	UpdatedAt                  time.Time `gorm:"autoUpdateTime"`
}

type Feedback struct {
	ID             string    `gorm:"primaryKey;size:36"`
	SiteID         string    `gorm:"index;index:idx_feedbacks_site_created,priority:1;not null;size:36"`
	Contact        string    `gorm:"not null;size:320"`
	Message        string    `gorm:"not null;size:4000"`
	Sentiment      string    `gorm:"size:16"`
	IP             string    `gorm:"size:64"`
	UserAgent      string    `gorm:"size:400"`
	Delivery       string    `gorm:"not null;size:16;default:no"`
	SourceKind     string    `gorm:"not null;size:20;default:web_widget;index"`
	MobileClientID string    `gorm:"size:80"`
	ScreenName     string    `gorm:"size:120"`
	ScreenPath     string    `gorm:"size:300"`
	AppPlatform    string    `gorm:"size:20"`
	AppIdentifier  string    `gorm:"size:200"`
	AppVersion     string    `gorm:"size:80"`
	AppBuild       string    `gorm:"size:80"`
	AppEnvironment string    `gorm:"size:80"`
	ContextJSON    string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"autoCreateTime;index:idx_feedbacks_site_created,priority:2"`
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
