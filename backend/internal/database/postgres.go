package database

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	writer := log.New(os.Stdout, "gorm ", log.LstdFlags)
	config := &gorm.Config{
		Logger: logger.New(
			writer,
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
			},
		),
	}

	return gorm.Open(postgres.Open(dsn), config)
}
