package storage

import (
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
)

func OpenPostgres(databaseDSN string) (*gorm.DB, error) {
	database, openErr := gorm.Open(postgres.Open(databaseDSN), &gorm.Config{})
	if openErr != nil {
		return nil, openErr
	}
	return database, nil
}

func AutoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(&model.Site{}, &model.Feedback{})
}

func NewID() string {
	return uuid.NewString()
}
