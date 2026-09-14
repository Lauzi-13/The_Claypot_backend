// Seeds the two category registries with the same defaults store.js ships
// with, so the API isn't empty on first use. Safe to run more than once —
// every insert is "create if missing", nothing is overwritten or
// duplicated. Run with: go run ./seed
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"claypot-backend/models"
)

var drinkCategories = []models.DrinkCategory{
	{Key: "traditional", Label: "Traditional Drinks"},
	{Key: "beer", Label: "Beer"},
	{Key: "beer-bucket", Label: "Beer Buckets"},
	{Key: "cider", Label: "Ciders"},
	{Key: "cider-bucket", Label: "Cider Buckets"},
	{Key: "cocktail", Label: "Cocktails"},
	{Key: "whiskey", Label: "Whiskey"},
	{Key: "spirits", Label: "Spirits"},
	{Key: "soft", Label: "Soft Drinks"},
}

var mealCategories = []string{"Mains", "Tapas & Starters", "Sides"}

func main() {
	godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set. Copy .env.example to .env and fill it in.")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}

	for _, c := range drinkCategories {
		if err := db.FirstOrCreate(&c, models.DrinkCategory{Key: c.Key}).Error; err != nil {
			log.Fatalf("seeding drink category %q: %v", c.Key, err)
		}
	}
	for _, name := range mealCategories {
		mc := models.MealCategory{Name: name}
		if err := db.FirstOrCreate(&mc, models.MealCategory{Name: name}).Error; err != nil {
			log.Fatalf("seeding meal category %q: %v", name, err)
		}
	}

	meals, drinks := seedProducts(db)
	createdStaffUser := seedStaffUser(db)

	fmt.Printf("Seeded %d drink categories and %d meal categories.\n", len(drinkCategories), len(mealCategories))
	fmt.Printf("Seeded %d new meals and %d new drinks (skipped any that already existed).\n", meals, drinks)
	if createdStaffUser {
		fmt.Println("Created the initial staff login (see seed/staffuser.go for the username — password was set when you asked for this, not printed here).")
	} else {
		fmt.Println("Staff account(s) already exist — skipped seeding the initial one.")
	}
}
