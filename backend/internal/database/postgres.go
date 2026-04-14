package database

import (
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn, appEnv string) (*gorm.DB, error) {
	writer := log.New(os.Stdout, "gorm ", log.LstdFlags)
	logLevel := logger.Warn
	if strings.EqualFold(appEnv, "development") {
		logLevel = logger.Info
	}
	config := &gorm.Config{
		Logger: logger.New(
			writer,
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				LogLevel:                  logLevel,
			},
		),
	}

	return gorm.Open(postgres.Open(dsn), config)
}
