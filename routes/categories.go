package routes

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"claypot-backend/models"
)

type CategoriesHandler struct {
	DB *gorm.DB
}

// requireAuth protects adding a category — the GETs stay open since
// drinks.html's tabs and the staff category dropdowns both need them.
func (h *CategoriesHandler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/meal", h.listMeal)
	r.Get("/drink", h.listDrink)
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Post("/meal", h.createMeal)
		r.Post("/drink", h.createDrink)
	})
	return r
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)
var trimDashRe = regexp.MustCompile(`^-+|-+$`)

// slugify matches store.js's slugifyCategory() exactly: lowercase, any run
// of non-alphanumeric characters becomes a single dash, leading/trailing
// dashes are trimmed, and an empty result falls back to "category".
func slugify(label string) string {
	s := strings.ToLower(strings.TrimSpace(label))
	s = nonAlnumRe.ReplaceAllString(s, "-")
	s = trimDashRe.ReplaceAllString(s, "")
	if s == "" {
		return "category"
	}
	return s
}

type labelBody struct {
	Label string `json:"label"`
}

// GET /api/categories/meal — replaces getMealCategories(). Meal categories
// are plain strings, unlike drinks (no separate slug/label split).
func (h *CategoriesHandler) listMeal(w http.ResponseWriter, r *http.Request) {
	var categories []models.MealCategory
	if err := h.DB.Order("name asc").Find(&categories).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Name
	}
	writeJSON(w, http.StatusOK, names)
}

// POST /api/categories/meal { label } — replaces addMealCategory(). A
// category can be created before any product uses it, same as store.js.
func (h *CategoriesHandler) createMeal(w http.ResponseWriter, r *http.Request) {
	var body labelBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		writeError(w, http.StatusBadRequest, "Category name is required.")
		return
	}

	var existing models.MealCategory
	if err := h.DB.Where("LOWER(name) = LOWER(?)", label).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"category": existing.Name})
		return
	}

	created := models.MealCategory{Name: label}
	if err := h.DB.Create(&created).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"category": created.Name})
}

// GET /api/categories/drink — replaces getDrinkCategories()
func (h *CategoriesHandler) listDrink(w http.ResponseWriter, r *http.Request) {
	var categories []models.DrinkCategory
	if err := h.DB.Order("label asc").Find(&categories).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// POST /api/categories/drink { label } — replaces addDrinkCategory().
// Slugs the label into a key the same way store.js's slugifyCategory()
// does, appending -2, -3, ... if that slug is already taken by a different
// label.
func (h *CategoriesHandler) createDrink(w http.ResponseWriter, r *http.Request) {
	var body labelBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		writeError(w, http.StatusBadRequest, "Category name is required.")
		return
	}

	var existing models.DrinkCategory
	if err := h.DB.Where("LOWER(label) = LOWER(?)", label).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"category": existing})
		return
	}

	key := slugify(label)
	suffix := 2
	for {
		var conflict models.DrinkCategory
		if h.DB.Where("key = ?", key).First(&conflict).Error != nil {
			break
		}
		key = slugify(label) + "-" + strconv.Itoa(suffix)
		suffix++
	}

	created := models.DrinkCategory{Key: key, Label: label}
	if err := h.DB.Create(&created).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"category": created})
}
