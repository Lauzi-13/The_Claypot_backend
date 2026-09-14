// Package db opens the connection to Postgres and keeps the database
// schema in sync with the structs in the models package.
package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"claypot-backend/models"
)

// Connect opens a connection to the Postgres database named by dsn (the
// DATABASE_URL environment variable — see .env.example).
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // only log slow/failed queries, not every query
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	return db, nil
}

// AutoMigrate creates any table that doesn't exist yet and adds any column
// that's missing, based on the struct tags in the models package. This is
// the Go/GORM equivalent of running `prisma migrate dev` — safe to call
// every time the server starts. It never drops or renames an existing
// column, so it never deletes data on its own.
func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.DrinkCategory{},
		&models.MealCategory{},
		&models.Product{},
		&models.StaffUser{},
		&models.StaffSession{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.OrderStatusHistory{},
		&models.Setting{},
		&models.GalleryImage{},
		&models.StockItem{},
	)
	if err != nil {
		return fmt.Errorf("running auto-migration: %w", err)
	}
	log.Println("Database schema is up to date.")
	return nil
}
