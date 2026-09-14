package main

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

// The photos already on the Gallery page — seeded by static asset path
// (not re-encoded as base64) since the images themselves already ship
// with the frontend and are 200-500KB each, far too big to duplicate into
// the database. Anything staff upload afterward goes through the API as a
// compressed base64 data URL instead (see routes/gallery.go).
var galleryImages = []struct {
	path     string
	category string
	caption  string
}{
	{"assets/img/aesthetic/hero-exterior.jpg", "aesthetic", "The Clay Pot exterior"},
	{"assets/img/aesthetic/interior-baskets-tv.jpg", "aesthetic", "Interior with basket wall art and TV"},
	{"assets/img/aesthetic/interior-baskets-dining.jpg", "aesthetic", "Dining area under basket wall art"},
	{"assets/img/aesthetic/outdoor-picnic-umbrellas.jpg", "aesthetic", "Outdoor picnic tables with striped umbrellas"},
	{"assets/img/aesthetic/interior-wide-lounge.jpg", "aesthetic", "Interior lounge seating"},
	{"assets/img/aesthetic/patio-entrance-plant.jpg", "aesthetic", "Patio entrance"},
	{"assets/img/aesthetic/cocktail-moody.jpg", "aesthetic", "Signature cocktail, moody lighting"},
	{"assets/img/aesthetic/outdoor-benches-umbrella.jpg", "aesthetic", "Outdoor benches and umbrella"},
	{"assets/img/aesthetic/cocktail-hand.jpg", "aesthetic", "Cocktail being served"},
	{"assets/img/food/whole-bream.jpg", "food", "Whole Bream, fried, served with relishes"},
	{"assets/img/food/chicken-wings-fries.jpg", "food", "Chicken Wings with Fries"},
	{"assets/img/food/nshima-beef-stew.jpg", "food", "Nshima with Beef Stew and relishes"},
	{"assets/img/food/grilled-ribs.jpg", "food", "Grilled beef, sliced, served with fries"},
}

// seedGalleryImages runs once — if the table already has any rows at all
// (whether from this seed or from staff uploads/deletes since), it's left
// alone entirely, so re-running this never resurrects something staff
// deliberately deleted.
func seedGalleryImages(db *gorm.DB) int {
	var count int64
	db.Model(&models.GalleryImage{}).Count(&count)
	if count > 0 {
		return 0
	}
	for _, g := range galleryImages {
		caption := g.caption
		image := models.GalleryImage{ID: uuid.NewString(), ImageURL: g.path, Category: g.category, Caption: &caption}
		if err := db.Create(&image).Error; err != nil {
			log.Fatalf("seeding gallery image %q: %v", g.path, err)
		}
	}
	return len(galleryImages)
}
