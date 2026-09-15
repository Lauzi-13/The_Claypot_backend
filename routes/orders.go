package routes

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"claypot-backend/models"
)

type OrdersHandler struct {
	DB *gorm.DB
}

// requireAuth protects the staff-only actions (viewing every order, and
// completing/cancelling/marking paid). Everything a customer's own
// browser needs stays public and untouched: placing an order (staff's own
// "Take an Order" tool uses the same endpoint), looking up their own order
// by id or phone, and arrive/renew/renew-cancelled — the last of which
// fires automatically right after a customer places a delivery order,
// with no staff login involved.
func (h *OrdersHandler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/phone", h.getByPhone)
	r.Get("/{id}", h.getByID)
	r.Patch("/{id}/arrive", h.arrive)
	r.Patch("/{id}/renew", h.renew)
	r.Patch("/{id}/renew-cancelled", h.renewCancelled)
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/", h.list)
		r.Patch("/{id}/complete", h.complete)
		r.Patch("/{id}/cancel", h.cancel)
		r.Patch("/{id}/paid", h.setPaid)
		r.Post("/{id}/payments", h.createPayment)
		r.Patch("/payments/{paymentId}/confirm", h.confirmPayment)
	})
	return r
}

// httpError carries a specific status code out of a transaction, so the
// HTTP handler can report the right one instead of a generic 500.
type httpError struct {
	status  int
	message string
}

func (e *httpError) Error() string { return e.message }

func respondErr(w http.ResponseWriter, err error) {
	if he, ok := err.(*httpError); ok {
		writeError(w, he.status, he.message)
		return
	}
	writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
}

func orderPreload(q *gorm.DB) *gorm.DB {
	return q.Preload("Items").Preload("Payments")
}

// Same job store.js's cancelExpiredOrders() did, run lazily on read/write
// rather than a background job — fine at this scale, matches the existing
// pattern exactly.
func cancelExpiredOrders(db *gorm.DB) error {
	var expired []models.Order
	if err := db.Where("status = ? AND expires_at < ?", models.OrderStatusAwaitingArrival, time.Now()).Find(&expired).Error; err != nil {
		return err
	}
	for _, order := range expired {
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Update("status", models.OrderStatusCancelled).Error; err != nil {
				return err
			}
			note := "Arrival window expired"
			from := models.OrderStatusAwaitingArrival
			history := models.OrderStatusHistory{
				ID: uuid.NewString(), OrderID: order.ID, FromStatus: &from,
				ToStatus: models.OrderStatusCancelled, Note: &note,
			}
			return tx.Create(&history).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Stock is only deducted once an order is actually completed (not when
// it's merely placed) — a cancelled order never touched stock, so there's
// nothing to restore. Mirrors store.js's decrementStockForItems(): only
// products with StockQty already being tracked (non-nil) are touched.
func decrementStockForItems(tx *gorm.DB, items []models.OrderItem) error {
	for _, item := range items {
		if item.ProductID == nil {
			continue
		}
		var product models.Product
		if err := tx.First(&product, "id = ?", *item.ProductID).Error; err != nil {
			continue // product may have been deleted since the order was placed
		}
		if product.StockQty != nil {
			qty := *product.StockQty - item.Qty
			if qty < 0 {
				qty = 0
			}
			if err := tx.Model(&product).Updates(map[string]interface{}{
				"stock_qty":    qty,
				"out_of_stock": qty <= 0,
			}).Error; err != nil {
				return err
			}
		}
		if err := decrementIngredientsForProduct(tx, product.ID, item.Qty); err != nil {
			return err
		}
	}
	return nil
}

// decrementIngredientsForProduct walks the product's recipe (see
// ProductIngredient) and takes orderQty * each ingredient's per-order qty
// off the linked raw StockItem — this is what lets completing an order
// count down actual ingredients (beans, village chicken, ...) instead of
// just the menu item's own sell count. A StockItem left untracked
// (Quantity nil — the "can't really count it" case) is skipped, same
// nil-safety as the Product decrement above.
func decrementIngredientsForProduct(tx *gorm.DB, productID string, orderQty int) error {
	var links []models.ProductIngredient
	if err := tx.Where("product_id = ?", productID).Find(&links).Error; err != nil {
		return err
	}
	for _, link := range links {
		var stockItem models.StockItem
		if err := tx.First(&stockItem, "id = ?", link.StockItemID).Error; err != nil {
			continue // stock item may have been deleted since the recipe was set
		}
		if stockItem.Quantity == nil {
			continue
		}
		qty := *stockItem.Quantity - (link.Qty * orderQty)
		if qty < 0 {
			qty = 0
		}
		if err := tx.Model(&stockItem).Updates(map[string]interface{}{
			"quantity":     qty,
			"out_of_stock": qty <= 0,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// GET /api/orders?status=awaiting-arrival — replaces getOrders()
func (h *OrdersHandler) list(w http.ResponseWriter, r *http.Request) {
	if err := cancelExpiredOrders(h.DB); err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	var orders []models.Order
	q := orderPreload(h.DB).Order("placed_at desc")
	if status := r.URL.Query().Get("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Find(&orders).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

var nonDigitRe = regexp.MustCompile(`[^\d]`)

// normalizePhone accepts 0XXXXXXXXX or +260XXXXXXXXX (with or without
// spaces/dashes) and reduces both to the same digits-only, country-coded
// form so a lookup matches regardless of which format was typed.
func normalizePhone(raw string) string {
	digits := nonDigitRe.ReplaceAllString(raw, "")
	if strings.HasPrefix(digits, "0") {
		digits = "260" + digits[1:]
	}
	return digits
}

// GET /api/orders/phone?number=... — replaces findOrderByPhone(). Prefers
// the active order for that number; falls back to the most recent one. A
// query parameter (not a path segment) on purpose — a phone number can
// contain a "+", and some HTTP clients percent-encode that as %2B in a way
// Go's router doesn't reliably match as a path segment; query values don't
// have that problem.
func (h *OrdersHandler) getByPhone(w http.ResponseWriter, r *http.Request) {
	if err := cancelExpiredOrders(h.DB); err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	digits := normalizePhone(r.URL.Query().Get("number"))
	if digits == "" {
		writeError(w, http.StatusBadRequest, "A phone number query parameter is required.")
		return
	}

	var orders []models.Order
	err := orderPreload(h.DB).Order("placed_at desc").
		Where("customer_phone LIKE ?", "%"+digits+"%").
		Find(&orders).Error
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	if len(orders) == 0 {
		writeError(w, http.StatusNotFound, "No order found for that phone number.")
		return
	}
	for _, o := range orders {
		if o.Status == models.OrderStatusAwaitingArrival || o.Status == models.OrderStatusArrived {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	writeJSON(w, http.StatusOK, orders[0])
}

// GET /api/orders/:id — replaces getOrder()
func (h *OrdersHandler) getByID(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := orderPreload(h.DB).First(&order, "id = ?", chi.URLParam(r, "id")).Error; err != nil {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

type orderItemBody struct {
	ProductID *string `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Qty       int     `json:"qty"`
}

type createOrderBody struct {
	CustomerName     string          `json:"customerName"`
	CustomerPhone    *string         `json:"customerPhone"`
	ArrivalMinutes   int             `json:"arrivalMinutes"`
	Items            []orderItemBody `json:"items"`
	OrderType        string          `json:"orderType"`
	DeliveryMethod   *string         `json:"deliveryMethod"`
	DeliveryAddress  *string         `json:"deliveryAddress"`
	CreatedByStaffID *string         `json:"createdByStaffId"`
}

// POST /api/orders — replaces createOrder(). The order + order_items are
// created in one transaction. No table assignment — the frontend dropped
// tables entirely, so an order is just Dine In or Takeaway.
func (h *OrdersHandler) create(w http.ResponseWriter, r *http.Request) {
	if err := cancelExpiredOrders(h.DB); err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}

	var body createOrderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.CustomerName == "" || len(body.Items) == 0 {
		writeError(w, http.StatusBadRequest, "customerName and at least one item are required.")
		return
	}
	if body.OrderType == "" {
		body.OrderType = models.OrderTypeDineIn
	}
	// Store the phone number pre-normalized (digits only, country-coded) so
	// a later lookup by phone (see getByPhone) can just LIKE-match against
	// it directly, instead of the query and the stored value being in two
	// different formats.
	if body.CustomerPhone != nil {
		normalized := normalizePhone(*body.CustomerPhone)
		body.CustomerPhone = &normalized
	}

	// Look up each item's live price/stock instead of trusting the client —
	// the equivalent of store.js's isItemOutOfStock() check, but now it's
	// impossible to bypass by editing client-side JS.
	var productIDs []string
	for _, i := range body.Items {
		if i.ProductID != nil {
			productIDs = append(productIDs, *i.ProductID)
		}
	}
	var products []models.Product
	if len(productIDs) > 0 {
		if err := h.DB.Where("id IN ?", productIDs).Find(&products).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
			return
		}
	}
	productByID := make(map[string]models.Product, len(products))
	for _, p := range products {
		productByID[p.ID] = p
	}

	var outOfStockNames []string
	for _, i := range body.Items {
		if i.ProductID == nil {
			continue
		}
		if p, ok := productByID[*i.ProductID]; ok && p.OutOfStock {
			outOfStockNames = append(outOfStockNames, p.Name)
		}
	}
	if len(outOfStockNames) > 0 {
		writeError(w, http.StatusConflict, "Out of stock: "+strings.Join(outOfStockNames, ", "))
		return
	}

	var total float64
	items := make([]models.OrderItem, len(body.Items))
	for idx, i := range body.Items {
		name := i.Name
		price := i.Price
		if i.ProductID != nil {
			if p, ok := productByID[*i.ProductID]; ok {
				name = p.Name
				price = p.Price
			}
		}
		total += price * float64(i.Qty)
		items[idx] = models.OrderItem{
			ID:            uuid.NewString(),
			ProductID:     i.ProductID,
			NameSnapshot:  name,
			PriceSnapshot: price,
			Qty:           i.Qty,
		}
	}

	now := time.Now()
	order := models.Order{
		ID:               uuid.NewString(),
		OrderType:        body.OrderType,
		DeliveryMethod:   body.DeliveryMethod,
		DeliveryAddress:  body.DeliveryAddress,
		CustomerName:     body.CustomerName,
		CustomerPhone:    body.CustomerPhone,
		Total:            total,
		ArrivalMinutes:   body.ArrivalMinutes,
		PlacedAt:         now,
		ExpiresAt:        now.Add(time.Duration(body.ArrivalMinutes) * time.Minute),
		Status:           models.OrderStatusAwaitingArrival,
		CreatedByStaffID: body.CreatedByStaffID,
		Items:            items,
	}
	if order.OrderType == models.OrderTypeTakeaway {
		if order.DeliveryMethod == nil || *order.DeliveryMethod != models.DeliveryMethodDelivery {
			order.DeliveryAddress = nil
		}
	} else {
		order.DeliveryMethod = nil
		order.DeliveryAddress = nil
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		history := models.OrderStatusHistory{
			ID: uuid.NewString(), OrderID: order.ID, ToStatus: models.OrderStatusAwaitingArrival,
		}
		return tx.Create(&history).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}

	// TODO: staff WhatsApp notification (CallMeBot) belongs here now,
	// reading the API key from an environment variable / Setting row —
	// see .env.example. Same TODO carried over from the old Node backend.

	orderPreload(h.DB).First(&order, "id = ?", order.ID)
	writeJSON(w, http.StatusCreated, order)
}

type transitionOpts struct {
	changedByStaffID *string
	note             *string
	requireStatuses  []string
}

func transitionStatus(db *gorm.DB, orderID, toStatus string, opts transitionOpts) (*models.Order, error) {
	var result models.Order
	err := db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
			return &httpError{status: http.StatusNotFound, message: "Order not found."}
		}
		if opts.requireStatuses != nil {
			ok := false
			for _, s := range opts.requireStatuses {
				if order.Status == s {
					ok = true
					break
				}
			}
			if !ok {
				return &httpError{status: http.StatusConflict, message: "Order can't move to " + toStatus + " from " + order.Status + "."}
			}
		}

		fromStatus := order.Status // captured before the update below overwrites it

		updates := map[string]interface{}{"status": toStatus}
		if toStatus == models.OrderStatusArrived {
			updates["arrived_at"] = time.Now()
		}
		if err := tx.Model(&order).Updates(updates).Error; err != nil {
			return err
		}

		history := models.OrderStatusHistory{
			ID: uuid.NewString(), OrderID: orderID, FromStatus: &fromStatus, ToStatus: toStatus,
			ChangedByStaffID: opts.changedByStaffID, Note: opts.note,
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}

		if toStatus == models.OrderStatusCompleted {
			if err := decrementStockForItems(tx, order.Items); err != nil {
				return err
			}
		}

		return orderPreload(tx).First(&result, "id = ?", orderID).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PATCH /api/orders/:id/arrive — replaces markArrived()
func (h *OrdersHandler) arrive(w http.ResponseWriter, r *http.Request) {
	order, err := transitionStatus(h.DB, chi.URLParam(r, "id"), models.OrderStatusArrived, transitionOpts{
		requireStatuses: []string{models.OrderStatusAwaitingArrival},
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

type staffIDBody struct {
	StaffID *string `json:"staffId"`
}

// PATCH /api/orders/:id/complete — replaces completeOrder()
func (h *OrdersHandler) complete(w http.ResponseWriter, r *http.Request) {
	var body staffIDBody
	json.NewDecoder(r.Body).Decode(&body) // a body is optional here
	order, err := transitionStatus(h.DB, chi.URLParam(r, "id"), models.OrderStatusCompleted, transitionOpts{
		changedByStaffID: body.StaffID,
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// PATCH /api/orders/:id/cancel — replaces cancelOrder() (manual staff cancel)
func (h *OrdersHandler) cancel(w http.ResponseWriter, r *http.Request) {
	var body staffIDBody
	json.NewDecoder(r.Body).Decode(&body)
	note := "Cancelled by staff"
	order, err := transitionStatus(h.DB, chi.URLParam(r, "id"), models.OrderStatusCancelled, transitionOpts{
		changedByStaffID: body.StaffID,
		note:             &note,
		requireStatuses:  []string{models.OrderStatusAwaitingArrival, models.OrderStatusArrived},
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// PATCH /api/orders/:id/renew — replaces renewOrder() (still-active order)
func (h *OrdersHandler) renew(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var order models.Order
	if err := h.DB.First(&order, "id = ?", id).Error; err != nil || order.Status != models.OrderStatusAwaitingArrival {
		writeError(w, http.StatusConflict, "Order can no longer be renewed.")
		return
	}
	newExpiry := time.Now().Add(time.Duration(order.ArrivalMinutes) * time.Minute)
	if err := h.DB.Model(&order).Updates(map[string]interface{}{
		"expires_at":  newExpiry,
		"renew_count": gorm.Expr("renew_count + 1"),
	}).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	orderPreload(h.DB).First(&order, "id = ?", id)
	writeJSON(w, http.StatusOK, order)
}

// PATCH /api/orders/:id/renew-cancelled — replaces renewCancelledOrder()
// (a fresh 15-minute window after auto-cancel)
func (h *OrdersHandler) renewCancelled(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var result models.Order
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.Order
		if err := tx.First(&existing, "id = ?", id).Error; err != nil || existing.Status != models.OrderStatusCancelled {
			return &httpError{status: http.StatusConflict, message: "This order can no longer be renewed."}
		}
		if err := tx.Model(&existing).Updates(map[string]interface{}{
			"status":          models.OrderStatusAwaitingArrival,
			"expires_at":      time.Now().Add(15 * time.Minute),
			"arrival_minutes": 15,
			"renew_count":     gorm.Expr("renew_count + 1"),
		}).Error; err != nil {
			return err
		}
		note := "Renewed after auto-cancel"
		from := models.OrderStatusCancelled
		history := models.OrderStatusHistory{
			ID: uuid.NewString(), OrderID: id, FromStatus: &from,
			ToStatus: models.OrderStatusAwaitingArrival, Note: &note,
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		return orderPreload(tx).First(&result, "id = ?", id).Error
	})
	if err != nil {
		respondErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type paidBody struct {
	Paid bool `json:"paid"`
}

// PATCH /api/orders/:id/paid { paid } — replaces setOrderPaid(). A simple
// staff toggle for "handed over cash" — distinct from the Payment ledger
// below, which is for tracking actual mobile-money/cash payment attempts.
func (h *OrdersHandler) setPaid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body paidBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	var order models.Order
	if err := h.DB.First(&order, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	if err := h.DB.Model(&order).Update("paid", body.Paid).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	orderPreload(h.DB).First(&order, "id = ?", id)
	writeJSON(w, http.StatusOK, order)
}

type createPaymentBody struct {
	Method    string   `json:"method"`
	Amount    *float64 `json:"amount"`
	Reference *string  `json:"reference"`
}

// POST /api/orders/:id/payments — record a payment attempt (mobile money
// proof sent on WhatsApp, cash, etc.) — replaces setOrderPaid()/paymentRef
func (h *OrdersHandler) createPayment(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var body createPaymentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.Method == "" || body.Amount == nil {
		writeError(w, http.StatusBadRequest, "method and amount are required.")
		return
	}
	payment := models.Payment{
		ID: uuid.NewString(), OrderID: orderID, Method: body.Method,
		Amount: *body.Amount, Reference: body.Reference, Status: models.PaymentStatusPending,
	}
	if err := h.DB.Create(&payment).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	writeJSON(w, http.StatusCreated, payment)
}

// PATCH /api/orders/payments/:paymentId/confirm — staff confirms a payment
// after checking the mobile money statement
func (h *OrdersHandler) confirmPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentId")
	var body staffIDBody
	json.NewDecoder(r.Body).Decode(&body)

	var payment models.Payment
	if err := h.DB.First(&payment, "id = ?", paymentID).Error; err != nil {
		writeError(w, http.StatusNotFound, "Payment not found.")
		return
	}
	updates := map[string]interface{}{
		"status":       models.PaymentStatusConfirmed,
		"confirmed_at": time.Now(),
	}
	if body.StaffID != nil {
		updates["confirmed_by_staff_id"] = *body.StaffID
	}
	if err := h.DB.Model(&payment).Updates(updates).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	h.DB.First(&payment, "id = ?", paymentID)
	writeJSON(w, http.StatusOK, payment)
}
