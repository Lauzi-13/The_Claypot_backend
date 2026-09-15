package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

type ProductsHandler struct {
	DB *gorm.DB
}

// requireAuth protects every write here (add/edit/delete/stock) — only
// GET (browsing the menu/drinks) stays open, since customers need that
// without logging in.
func (h *ProductsHandler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Patch("/{id}/stock", h.setStock)
		r.Patch("/{id}/stock/add", h.addStock)
		r.Get("/{id}/ingredients", h.getIngredients)
		r.Put("/{id}/ingredients", h.setIngredients)
	})
	return r
}

// GET /api/products?kind=meal|drink — replaces getMenu()/getDrinks()
func (h *ProductsHandler) list(w http.ResponseWriter, r *http.Request) {
	var products []models.Product
	q := h.DB.Order("name asc")
	if kind := r.URL.Query().Get("kind"); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if err := q.Find(&products).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

type createProductBody struct {
	Kind        string   `json:"kind"`
	Category    string   `json:"category"`
	Subcategory *string  `json:"subcategory"`
	Name        string   `json:"name"`
	Price       *float64 `json:"price"`
	Description *string  `json:"description"`
	ImageURL    *string  `json:"imageUrl"`
	IsSpecial   bool     `json:"isSpecial"`
	StockQty    *int     `json:"stockQty"`
}

// POST /api/products — replaces addMenuItem()/addDrink()/the Stock page's
// "+ Add New Item". stockQty is optional — omit it (or send null) to leave
// the item untracked, exactly like the old store.js did.
func (h *ProductsHandler) create(w http.ResponseWriter, r *http.Request) {
	var body createProductBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.Kind == "" || body.Category == "" || body.Name == "" || body.Price == nil {
		writeError(w, http.StatusBadRequest, "kind, category, name, and price are required.")
		return
	}

	outOfStock := false
	if body.StockQty != nil {
		qty := *body.StockQty
		if qty < 0 {
			qty = 0
			body.StockQty = &qty
		}
		outOfStock = qty <= 0
	}

	product := models.Product{
		ID:          uuid.NewString(),
		Kind:        body.Kind,
		Category:    body.Category,
		Subcategory: body.Subcategory,
		Name:        body.Name,
		Price:       *body.Price,
		Description: body.Description,
		ImageURL:    body.ImageURL,
		IsSpecial:   body.IsSpecial,
		StockQty:    body.StockQty,
		OutOfStock:  outOfStock,
	}
	if err := h.DB.Create(&product).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

// PATCH /api/products/:id — replaces updateMenuItem()/updateDrink()/setSpecial()
func (h *ProductsHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	var product models.Product
	if err := h.DB.First(&product, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	if err := h.DB.Model(&product).Updates(body).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	h.DB.First(&product, "id = ?", id)
	writeJSON(w, http.StatusOK, product)
}

// DELETE /api/products/:id — replaces removeMenuItem()/removeDrink()
func (h *ProductsHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.DB.Delete(&models.Product{}, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type stockBody struct {
	Qty int `json:"qty"`
}

// PATCH /api/products/:id/stock { qty } — set the exact stock count,
// replaces setStockQty()
func (h *ProductsHandler) setStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body stockBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	qty := body.Qty
	if qty < 0 {
		qty = 0
	}
	var product models.Product
	if err := h.DB.First(&product, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	product.StockQty = &qty
	product.OutOfStock = qty <= 0
	if err := h.DB.Save(&product).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

// ingredientDTO is what the frontend actually needs to render the recipe
// editor — the raw ProductIngredient row plus the stock item's own name and
// current quantity, so the page doesn't need a second round trip to label
// each row.
type ingredientDTO struct {
	ID                  string `json:"id"`
	StockItemID         string `json:"stockItemId"`
	StockItemName       string `json:"stockItemName"`
	Qty                 int    `json:"qty"`
	StockItemQty        *int   `json:"stockItemQty"`
	StockItemOutOfStock bool   `json:"stockItemOutOfStock"`
}

func loadIngredientDTOs(db *gorm.DB, productID string) ([]ingredientDTO, error) {
	var links []models.ProductIngredient
	if err := db.Where("product_id = ?", productID).Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []ingredientDTO{}, nil
	}
	stockItemIDs := make([]string, len(links))
	for i, l := range links {
		stockItemIDs[i] = l.StockItemID
	}
	var stockItems []models.StockItem
	if err := db.Where("id IN ?", stockItemIDs).Find(&stockItems).Error; err != nil {
		return nil, err
	}
	stockItemByID := make(map[string]models.StockItem, len(stockItems))
	for _, s := range stockItems {
		stockItemByID[s.ID] = s
	}
	dtos := make([]ingredientDTO, 0, len(links))
	for _, l := range links {
		s, ok := stockItemByID[l.StockItemID]
		if !ok {
			continue // the stock item was deleted since this link was made
		}
		dtos = append(dtos, ingredientDTO{
			ID: l.ID, StockItemID: l.StockItemID, StockItemName: s.Name, Qty: l.Qty,
			StockItemQty: s.Quantity, StockItemOutOfStock: s.OutOfStock,
		})
	}
	return dtos, nil
}

// GET /api/products/:id/ingredients — the recipe: which raw StockItems this
// menu item is made from, and how many units of each one order consumes.
func (h *ProductsHandler) getIngredients(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dtos, err := loadIngredientDTOs(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, dtos)
}

type setIngredientsBody struct {
	Ingredients []struct {
		StockItemID string `json:"stockItemId"`
		Qty         int    `json:"qty"`
	} `json:"ingredients"`
}

// PUT /api/products/:id/ingredients { ingredients: [{stockItemId, qty}] } —
// replaces the whole recipe for this product in one call, which is simpler
// for the staff-facing editor than individual add/remove endpoints. Qty
// must be at least 1; a stock item can only appear once (last one wins if
// the client sends a duplicate).
func (h *ProductsHandler) setIngredients(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body setIngredientsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	var product models.Product
	if err := h.DB.First(&product, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}

	deduped := make(map[string]int, len(body.Ingredients))
	for _, i := range body.Ingredients {
		if i.StockItemID == "" || i.Qty < 1 {
			continue
		}
		deduped[i.StockItemID] = i.Qty
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductIngredient{}).Error; err != nil {
			return err
		}
		for stockItemID, qty := range deduped {
			link := models.ProductIngredient{ID: uuid.NewString(), ProductID: id, StockItemID: stockItemID, Qty: qty}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}

	dtos, err := loadIngredientDTOs(h.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, dtos)
}

// PATCH /api/products/:id/stock/add { qty } — the Stock page's "Update
// Stock" action: add newly delivered quantity on top of what's already on
// hand, replaces addStockQty()
func (h *ProductsHandler) addStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body stockBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	added := body.Qty
	if added < 0 {
		added = 0
	}
	var product models.Product
	if err := h.DB.First(&product, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Product not found.")
		return
	}
	current := 0
	if product.StockQty != nil {
		current = *product.StockQty
	}
	qty := current + added
	product.StockQty = &qty
	product.OutOfStock = qty <= 0
	if err := h.DB.Save(&product).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, product)
}
