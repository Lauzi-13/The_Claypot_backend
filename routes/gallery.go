package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

type GalleryHandler struct {
	DB *gorm.DB
}

// requireAuth protects add/delete — anyone browsing the public Gallery
// page just needs GET, no login involved.
func (h *GalleryHandler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Post("/", h.create)
		r.Delete("/{id}", h.delete)
	})
	return r
}

// GET /api/gallery?category=aesthetic|food
func (h *GalleryHandler) list(w http.ResponseWriter, r *http.Request) {
	var images []models.GalleryImage
	q := h.DB.Order("created_at asc")
	if category := r.URL.Query().Get("category"); category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Find(&images).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, images)
}

type createGalleryImageBody struct {
	ImageURL string  `json:"imageUrl"`
	Category string  `json:"category"`
	Caption  *string `json:"caption"`
}

// POST /api/gallery { imageUrl, category, caption }
func (h *GalleryHandler) create(w http.ResponseWriter, r *http.Request) {
	var body createGalleryImageBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.ImageURL == "" || (body.Category != "aesthetic" && body.Category != "food") {
		writeError(w, http.StatusBadRequest, "imageUrl and a category of \"aesthetic\" or \"food\" are required.")
		return
	}
	image := models.GalleryImage{
		ID: uuid.NewString(), ImageURL: body.ImageURL, Category: body.Category, Caption: body.Caption,
	}
	if err := h.DB.Create(&image).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, image)
}

// DELETE /api/gallery/:id
func (h *GalleryHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.DB.Delete(&models.GalleryImage{}, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
