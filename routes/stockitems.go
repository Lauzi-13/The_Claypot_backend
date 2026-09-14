package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

type StockItemsHandler struct {
	DB *gorm.DB
}

// Every route here is staff-only — unlike products/gallery, customers
// never need to see raw ingredient stock, so there's no public GET.
func (h *StockItemsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Patch("/{id}", h.update)
	r.Delete("/{id}", h.delete)
	return r
}

func (h *StockItemsHandler) list(w http.ResponseWriter, r *http.Request) {
	var items []models.StockItem
	if err := h.DB.Order("name asc").Find(&items).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type createStockItemBody struct {
	Name     string `json:"name"`
	Quantity *int   `json:"quantity"`
}

func (h *StockItemsHandler) create(w http.ResponseWriter, r *http.Request) {
	var body createStockItemBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required.")
		return
	}
	item := models.StockItem{
		ID: uuid.NewString(), Name: body.Name, Quantity: body.Quantity,
		OutOfStock: body.Quantity != nil && *body.Quantity <= 0,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// PATCH /api/stock-items/:id { name?, quantity?, outOfStock? } — a plain
// partial update (unlike Product's setStock/addStock, there's no separate
// "add on top" endpoint here; the frontend computes the new total and
// sends it directly, since this is a much smaller, simpler model).
func (h *StockItemsHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	var item models.StockItem
	if err := h.DB.First(&item, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Stock item not found.")
		return
	}
	if err := h.DB.Model(&item).Updates(body).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	h.DB.First(&item, "id = ?", id)
	writeJSON(w, http.StatusOK, item)
}

func (h *StockItemsHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.DB.Delete(&models.StockItem{}, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
