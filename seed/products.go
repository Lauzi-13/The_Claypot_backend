package main

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

// seedProduct is a plain staging struct for seed data — deliberately not
// models.Product itself, since these entries don't have an id, kind, or
// stock yet; seedProducts() below fills those in.
type seedProduct struct {
	Category    string
	Subcategory *string
	Name        string
	Price       float64
	Description string
	IsSpecial   bool
	ImageURL    *string
}

func strPtr(s string) *string { return &s }

// The exact same starting menu and drinks the old frontend's store.js
// shipped with (SEED_MEALS / SEED_DRINKS), so switching the site over to
// this backend doesn't leave the Menu/Drinks pages empty.

var seedMeals = []seedProduct{
	{Category: "Mains", Name: "Oxtail Stew", Price: 250, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/oxtail-stew.jpg")},
	{Category: "Mains", Name: "T-Bone", Price: 200, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/grilled-ribs.jpg")},
	{Category: "Mains", Name: "Village Chicken", Price: 180, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/village-chicken.jpg")},
	{Category: "Mains", Name: "Beef Stew", Price: 170, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/nshima-beef-stew.jpg")},
	{Category: "Mains", Name: "Whole Bream", Price: 200, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/whole-bream.jpg")},
	{Category: "Mains", Name: "Cow Trotters", Price: 100, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/cow-trotters.jpg")},
	{Category: "Mains", Name: "Offals", Price: 180, Description: "", IsSpecial: true, ImageURL: strPtr("assets/img/food/offals.jpg")},
	{Category: "Mains", Name: "Goat Curry", Price: 200, Description: "", IsSpecial: false, ImageURL: strPtr("assets/img/food/goat-curry.jpg")},
	{Category: "Mains", Name: "Chicken Wings", Price: 150, Description: "", IsSpecial: false, ImageURL: strPtr("assets/img/food/chicken-wings-fries.jpg")},
	{Category: "Tapas & Starters", Name: "Crisp Fried Chicken Legs", Price: 100, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Vinkubala", Price: 50, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Chikanda", Price: 80, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Groundnuts", Price: 40, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Fried Chicken Gizzards", Price: 50, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Crisp Fried Offals", Price: 60, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Braai Goat Meat", Price: 120, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Tapas & Starters", Name: "Fried Okra", Price: 40, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Sides", Name: "Nshima", Price: 50, Description: "", IsSpecial: false, ImageURL: nil},
	{Category: "Sides", Name: "Fries", Price: 50, Description: "", IsSpecial: false, ImageURL: nil},
}

var seedDrinks = []seedProduct{
	{Category: "traditional", Subcategory: nil, Name: "Kawawasha", Price: 25, ImageURL: strPtr("assets/img/traditional/kawawasha.jpg")},
	{Category: "traditional", Subcategory: nil, Name: "Tamarind", Price: 25, ImageURL: strPtr("assets/img/traditional/tamarind.jpg")},
	{Category: "beer", Subcategory: nil, Name: "Castle Lite", Price: 50, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Budweiser", Price: 70, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Corona", Price: 65, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Windhoek Lager", Price: 65, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Windhoek Draught", Price: 70, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Mosi", Price: 45, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Castle", Price: 40, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Heineken Silver", Price: 65, ImageURL: nil},
	{Category: "beer", Subcategory: nil, Name: "Heineken", Price: 65, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Castle Lite", Price: 265, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Budweiser", Price: 365, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Corona", Price: 340, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Windhoek Lager", Price: 340, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Windhoek Draught", Price: 365, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Mosi", Price: 240, ImageURL: nil},
	{Category: "beer-bucket", Subcategory: nil, Name: "Castle", Price: 215, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Brutal Fruit", Price: 55, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Flying Fish", Price: 55, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Hunters Dry", Price: 60, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Hunters Gold", Price: 60, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Savanna Dry", Price: 60, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "1430", Price: 55, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Brutal Fruit Cane", Price: 60, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Flying Fish Cane", Price: 50, ImageURL: nil},
	{Category: "cider", Subcategory: nil, Name: "Castle Lite Cane", Price: 60, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Brutal Fruit", Price: 270, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Flying Fish", Price: 270, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Hunters Dry", Price: 280, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Hunters Gold", Price: 280, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Savanna Dry", Price: 280, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "1430", Price: 270, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Brutal Fruit Cane", Price: 280, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Flying Fish Cane", Price: 245, ImageURL: nil},
	{Category: "cider-bucket", Subcategory: nil, Name: "Castle Lite Cane", Price: 280, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Mojito", Price: 150, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Gimlet", Price: 150, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Daquiri", Price: 150, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Whiskey Sour", Price: 160, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Classic Margarita", Price: 160, ImageURL: nil},
	{Category: "cocktail", Subcategory: nil, Name: "Long Island", Price: 170, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jameson Original Single Shot", Price: 55, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jameson Original Bottle", Price: 1200, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jameson Black Barrel Single Shot", Price: 80, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jameson Black Barrel Bottle", Price: 1500, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "JW Black Shot", Price: 65, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "JW Red Shot", Price: 50, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jack Daniels Shot", Price: 60, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Jack Daniels Bottle", Price: 1800, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Chivas 12 Years Shot", Price: 65, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Chivas 12 Years Bottle", Price: 1200, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Glenfiddich 12 Shot", Price: 135, ImageURL: nil},
	{Category: "whiskey", Subcategory: nil, Name: "Monkey Shoulder Bottle", Price: 1300, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Tequila"), Name: "Tequila Silver Shot", Price: 80, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Tequila"), Name: "Tequila Silver Bottle", Price: 1200, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Tequila"), Name: "Tequila Gold Shot", Price: 80, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Tequila"), Name: "Tequila Gold Bottle", Price: 1200, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Gordon Shot", Price: 45, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Gordons Bottle", Price: 350, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Beefeater Clear Shot", Price: 65, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Beefeater Clear Bottle", Price: 700, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Beefeater Pink Shot", Price: 75, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Gin"), Name: "Beefeater Pink Bottle", Price: 800, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Rum"), Name: "Bacardi Shot", Price: 55, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Rum"), Name: "Captain Morgan Shot", Price: 50, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Jägermeister Shot", Price: 65, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Jägermeister Bottle", Price: 550, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Amarula", Price: 45, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Amarula Bottle", Price: 680, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Absolut Vodka", Price: 50, ImageURL: nil},
	{Category: "spirits", Subcategory: strPtr("Extra Spirits"), Name: "Absolut Vodka Bottle", Price: 850, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Fanta 500ml", Price: 25, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Coke 500ml", Price: 25, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Fanta 350ml", Price: 15, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Coke 350ml", Price: 15, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Sprite 350ml", Price: 15, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Water", Price: 15, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Lemonade Brothers", Price: 35, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Tonic Water Brothers", Price: 35, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Ginger Ale", Price: 35, ImageURL: nil},
	{Category: "soft", Subcategory: nil, Name: "Soda Water", Price: 25, ImageURL: nil},
}

// seedProducts inserts each seed meal/drink only if a product with the same
// kind+category+name doesn't already exist — safe to run more than once,
// and won't fight with products staff have since added or edited.
func seedProducts(db *gorm.DB) (int, int) {
	insert := func(kind string, sp seedProduct) bool {
		var existing models.Product
		err := db.Where("kind = ? AND category = ? AND name = ?", kind, sp.Category, sp.Name).First(&existing).Error
		if err == nil {
			return false // already there
		}
		p := models.Product{
			ID: uuid.NewString(),
			Kind: kind, Category: sp.Category, Subcategory: sp.Subcategory,
			Name: sp.Name, Price: sp.Price, Description: &sp.Description,
			IsSpecial: sp.IsSpecial, ImageURL: sp.ImageURL,
		}
		if err := db.Create(&p).Error; err != nil {
			log.Fatalf("seeding product %q: %v", sp.Name, err)
		}
		return true
	}

	meals := 0
	for _, m := range seedMeals {
		if insert(models.ProductKindMeal, m) {
			meals++
		}
	}
	drinks := 0
	for _, d := range seedDrinks {
		if insert(models.ProductKindDrink, d) {
			drinks++
		}
	}
	return meals, drinks
}
