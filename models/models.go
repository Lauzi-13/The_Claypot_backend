// Package models defines every database table as a Go struct. GORM (the
// library that talks to Postgres for us) reads these struct definitions and
// creates/updates the matching tables automatically — see db/db.go.
//
// This mirrors the old Node.js backend's prisma/schema.prisma one-for-one:
// same tables, same columns, same behaviour. The "enum" fields (Kind,
// Status, etc.) are plain text columns here instead of native Postgres
// enum types — simpler to work with in Go, and validated in the route
// handlers instead of by the database itself.
package models

import "time"

// ---------- Allowed values for the "enum-like" string fields ----------
// Postgres just sees these as plain text columns; these constants are the
// only values the Go code itself ever writes into them.

const (
	ProductKindMeal  = "meal"
	ProductKindDrink = "drink"
)

const (
	OrderTypeDineIn   = "dine-in"
	OrderTypeTakeaway = "takeaway"
)

const (
	DeliveryMethodPickup   = "pickup"
	DeliveryMethodDelivery = "delivery"
)

const (
	OrderStatusAwaitingArrival = "awaiting-arrival"
	OrderStatusArrived         = "arrived"
	OrderStatusCancelled       = "cancelled"
	OrderStatusCompleted       = "completed"
)

const (
	PaymentMethodAirtelMoney = "airtel_money"
	PaymentMethodMtnMomo     = "mtn_momo"
	PaymentMethodCash        = "cash"
)

const (
	PaymentStatusPending   = "pending"
	PaymentStatusConfirmed = "confirmed"
	PaymentStatusFailed    = "failed"
)

const (
	StaffRoleStaff   = "staff"
	StaffRoleManager = "manager"
)

// Replaces the SEED_DRINK_CATEGORIES list + getDrinkCategories() /
// addDrinkCategory() in the old store.js. Key is the slug stored in a
// drink product's Category column (e.g. "beer-bucket"); Label is what's
// shown on-screen. A row can exist before any product uses it.
type DrinkCategory struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Label string `json:"label"`
}

func (DrinkCategory) TableName() string { return "drink_categories" }

// Replaces the SEED_MEAL_CATEGORIES list + getMealCategories() /
// addMealCategory() in the old store.js. Meal categories are plain display
// strings (no separate slug/label split, unlike drinks).
type MealCategory struct {
	Name string `gorm:"primaryKey" json:"name"`
}

func (MealCategory) TableName() string { return "meal_categories" }

// Replaces the separate SEED_MEALS / SEED_DRINKS arrays in store.js — one
// table for both, distinguished by Kind.
//
// StockQty mirrors store.js exactly: nil means "not being tracked yet" (in
// which case OutOfStock can still be toggled by hand); once staff set a
// quantity it becomes the source of truth, decrementing automatically when
// an order containing it is completed (see routes/orders.go) and flipping
// OutOfStock on at zero.
type Product struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Kind        string    `json:"kind"`
	Category    string    `json:"category"`
	Subcategory *string   `json:"subcategory"` // spirits only: Tequila / Gin / Rum / Extra Spirits
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Description *string   `json:"description"`
	ImageURL    *string   `gorm:"column:image_url" json:"imageUrl"`
	IsSpecial   bool      `gorm:"column:is_special;default:false" json:"isSpecial"`
	StockQty    *int      `gorm:"column:stock_qty" json:"stockQty"`
	OutOfStock  bool      `gorm:"column:out_of_stock;default:false" json:"outOfStock"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Product) TableName() string { return "products" }

// Replaces the ORDERS array. OrderNumber is the short human-facing number
// ("ORD-42" = OrderNumber 42) — ID stays an opaque string for safe use in
// URLs/APIs. No table/reservation field — the frontend dropped tables
// entirely, so an order is just Dine In or Takeaway.
type Order struct {
	ID               string     `gorm:"primaryKey" json:"id"`
	OrderNumber      int        `gorm:"column:order_number;autoIncrement;uniqueIndex" json:"orderNumber"`
	OrderType        string     `gorm:"column:order_type" json:"orderType"`
	DeliveryMethod   *string    `gorm:"column:delivery_method" json:"deliveryMethod"`
	DeliveryAddress  *string    `gorm:"column:delivery_address" json:"deliveryAddress"`
	CustomerName     string     `gorm:"column:customer_name" json:"customerName"`
	CustomerPhone    *string    `gorm:"column:customer_phone" json:"customerPhone"`
	Total            float64    `json:"total"`
	ArrivalMinutes   int        `gorm:"column:arrival_minutes" json:"arrivalMinutes"`
	PlacedAt         time.Time  `gorm:"column:placed_at" json:"placedAt"`
	ExpiresAt        time.Time  `gorm:"column:expires_at" json:"expiresAt"`
	RenewCount       int        `gorm:"column:renew_count;default:0" json:"renewCount"`
	Status           string     `gorm:"default:awaiting-arrival" json:"status"`
	ArrivedAt        *time.Time `gorm:"column:arrived_at" json:"arrivedAt"`
	CreatedByStaffID *string    `gorm:"column:created_by_staff_id" json:"createdByStaffId"`
	// A simple cash/counter "has this been paid?" flag — replaces the old
	// store.js's single `paid` boolean exactly. The richer Payment model
	// below is for tracking actual mobile-money/cash payment attempts; this
	// field is just the quick staff-facing toggle on the Orders page.
	Paid bool `gorm:"default:false" json:"paid"`

	Items    []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	Payments []Payment   `gorm:"foreignKey:OrderID" json:"payments"`
}

func (Order) TableName() string { return "orders" }

// Replaces the `items` array embedded inside each order in store.js.
// NameSnapshot/PriceSnapshot are captured at order time so a later price
// change or deleted product never rewrites history on an old receipt.
type OrderItem struct {
	ID            string  `gorm:"primaryKey" json:"id"`
	OrderID       string  `gorm:"column:order_id" json:"orderId"`
	ProductID     *string `gorm:"column:product_id" json:"productId"`
	NameSnapshot  string  `gorm:"column:name_snapshot" json:"nameSnapshot"`
	PriceSnapshot float64 `gorm:"column:price_snapshot" json:"priceSnapshot"`
	Qty           int     `json:"qty"`
}

func (OrderItem) TableName() string { return "order_items" }

// Replaces the single `paid` boolean + `paymentRef` string on an order.
// Gives room for cash vs mobile money, multiple attempts, and is the exact
// shape a real MoMo webhook would insert into later.
type Payment struct {
	ID                 string     `gorm:"primaryKey" json:"id"`
	OrderID            string     `gorm:"column:order_id" json:"orderId"`
	Method             string     `json:"method"`
	Amount             float64    `json:"amount"`
	Reference          *string    `json:"reference"`
	ConfirmedByStaffID *string    `gorm:"column:confirmed_by_staff_id" json:"confirmedByStaffId"`
	ConfirmedAt        *time.Time `gorm:"column:confirmed_at" json:"confirmedAt"`
	Status             string     `gorm:"default:pending" json:"status"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"createdAt"`
}

func (Payment) TableName() string { return "payments" }

// Replaces the single shared claypot_staff_pin. Lets staff actions be
// attributed to a real person instead of "whoever knew the PIN" — once
// staff login. PasswordHash is a bcrypt hash — the plain password is
// never stored or logged anywhere.
type StaffUser struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"column:password_hash" json:"-"`
	Role         string    `gorm:"default:staff" json:"role"`
	Active       bool      `gorm:"default:true" json:"active"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (StaffUser) TableName() string { return "staff_users" }

// A logged-in session — created on login, deleted on logout. Token is the
// opaque bearer value the frontend sends back on every request; looking
// one up and checking ExpiresAt is the entire authentication check (see
// RequireAuth in routes/auth.go).
type StaffSession struct {
	Token       string    `gorm:"primaryKey"`
	StaffUserID string    `gorm:"column:staff_user_id"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
}

func (StaffSession) TableName() string { return "staff_sessions" }

// An audit trail: every status change gets a row here; ChangedByStaffID is
// nil for an automatic transition (e.g. the arrival window expiring and
// auto-cancelling the order).
type OrderStatusHistory struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	OrderID          string    `gorm:"column:order_id" json:"orderId"`
	FromStatus       *string   `gorm:"column:from_status" json:"fromStatus"`
	ToStatus         string    `gorm:"column:to_status" json:"toStatus"`
	ChangedByStaffID *string   `gorm:"column:changed_by_staff_id" json:"changedByStaffId"`
	ChangedAt        time.Time `gorm:"column:changed_at" json:"changedAt"`
	Note             *string   `json:"note"`
}

func (OrderStatusHistory) TableName() string { return "order_status_history" }

// Replaces the hardcoded CALLMEBOT_API_KEY / mobile money numbers that used
// to sit as plain-text constants in site/store.js, visible to anyone who
// viewed the page source. Prefer plain environment variables (see
// .env.example) for now; this table is here for values that need to change
// without a redeploy.
type Setting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Setting) TableName() string { return "settings" }
