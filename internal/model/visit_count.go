package model

// TrafficProfile selects the data a site's collector can persist.
type TrafficProfile string

const (
	// TrafficProfileDetailed permits individual visit records.
	TrafficProfileDetailed TrafficProfile = "detailed"
	// TrafficProfileAggregate permits daily request counts only.
	TrafficProfileAggregate TrafficProfile = "aggregate"
)

// SiteVisitCount stores only a site's daily accepted request total.
type SiteVisitCount struct {
	SiteID string `gorm:"primaryKey;size:36" json:"site_id"`
	Date   string `gorm:"primaryKey;size:10" json:"date"`
	Count  int64  `gorm:"not null" json:"count"`
}
