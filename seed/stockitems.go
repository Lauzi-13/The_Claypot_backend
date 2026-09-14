package main

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

func intPtr(i int) *int { return &i }

// Starter set matching the examples given when this feature was requested
// — a few countable ingredients with real quantities, plus a couple that
// can't realistically be counted (left untracked, toggled by hand instead).
var stockItems = []models.StockItem{
	{Name: "Beef", Quantity: intPtr(10)},
	{Name: "Fish", Quantity: intPtr(10)},
	{Name: "T-Bone", Quantity: intPtr(6)},
	{Name: "Goat Meat", Quantity: intPtr(8)},
	{Name: "Chicken Wings", Quantity: intPtr(12)},
	{Name: "Village Chicken", Quantity: nil},
	{Name: "Chikanda", Quantity: nil},
}

// Runs once — if the table already has any rows (whether from this seed
// or from staff adding/deleting items since), it's left alone entirely.
func seedStockItems(db *gorm.DB) int {
	var count int64
	db.Model(&models.StockItem{}).Count(&count)
	if count > 0 {
		return 0
	}
	for _, s := range stockItems {
		item := models.StockItem{ID: uuid.NewString(), Name: s.Name, Quantity: s.Quantity, OutOfStock: false}
		if err := db.Create(&item).Error; err != nil {
			log.Fatalf("seeding stock item %q: %v", s.Name, err)
		}
	}
	return len(stockItems)
}
