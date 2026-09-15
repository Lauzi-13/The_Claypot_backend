# The Clay Pot — Backend (Go edition)

This is a from-scratch rewrite of the Phase 2 backend in Go, replacing the
earlier Node.js/Express/Prisma version (archived at `../backend`). Same job:
a real database + REST API that will eventually replace the localStorage
prototype in `../frontend/The_ClayPot/site/store.js`. Nothing in the
frontend calls this yet — this is still the foundation to build that
connection on top of.

## What's here

- `models/models.go` — every database table as a Go struct (products,
  meal/drink categories, orders, order items, payments, staff users, status
  history, settings). No tables/reservations — the frontend dropped those
  entirely, so an order is just Dine In or Takeaway.
- `db/db.go` — connects to Postgres and keeps the schema in sync with the
  structs (`AutoMigrate` — the Go/GORM equivalent of `prisma migrate dev`,
  safe to run every time the server starts).
- `routes/` — one file per resource (`products.go`, `categories.go`,
  `orders.go`), plus `helpers.go` for shared JSON response helpers.
- `main.go` — wires it all together and starts the HTTP server.
- `seed/main.go` — seeds the two category tables with the same defaults
  `store.js` ships with. Safe to run more than once. Run with `go run ./seed`.

**Tech stack:** [Go](https://go.dev) (the language/runtime) +
[chi](https://github.com/go-chi/chi) (a small, lightweight HTTP router —
the closest Go equivalent to Express) + [GORM](https://gorm.io) (an ORM —
the closest Go equivalent to Prisma: define a Go struct, it creates/updates
the matching table and generates the SQL for you) + PostgreSQL (same Neon
database as before).

## Why a separate schema

The old Node backend's tables (in Postgres) used native Postgres **enum**
types for fields like a product's `kind` or an order's `status` — a Prisma-
specific pattern. GORM's models here use plain text columns instead
(simpler, validated in the Go code rather than by the database), so they
can't safely share the same tables. Rather than drop the old tables, this
backend's tables live in their own schema, `claypot_go`, inside the *same*
database — see the `search_path` bit in `.env`. The old Node backend's
tables in `public` are untouched and can be dropped later once you're sure
you don't need them, from Neon's own SQL console or table view (a
`DROP SCHEMA public CASCADE` is the quickest way, but that's your call to
make, not something to automate).

## One-time setup

1. **Install Go** — download from https://go.dev/dl (this machine already
   has it: `go version` to check).
2. This machine already has a `.env` pointing at the same Neon database the
   old backend used, with a `search_path=claypot_go` added to the
   connection string so this backend's tables land in their own schema. If
   setting this up somewhere new, copy `.env.example` to `.env` and fill in
   your own `DATABASE_URL` — add `&options=-c%20search_path%3Dyour_schema`
   to the end of it if you want the same schema-isolation trick.

## Running it

From inside the `backend-go` folder:

```
go mod download        # downloads chi, GORM, etc. (first time only)
go run .                # starts the server on http://localhost:4000
```

The first run takes a little while (creating every table fresh against a
hosted database isn't instant) — watch for `Clay Pot backend (Go)
listening on port 4000` in the terminal. Every run after that is much
faster since the tables already exist.

Check it worked by opening http://localhost:4000/api/health in a browser —
it should show `{"ok":true}`.

To build a standalone executable instead of running from source:
```
go build -o claypot-server.exe .
./claypot-server.exe
```

Seed the default categories once (safe to re-run):
```
go run ./seed
```

## API overview

Same endpoints as the old Node backend, same behaviour:

- `GET/POST /api/products` — list/create meals and drinks (`?kind=meal` or
  `?kind=drink` to filter). `PATCH /:id` edits one; `DELETE /:id` removes it.
- `PATCH /api/products/:id/stock` — set the exact stock count (`{ qty }`).
- `PATCH /api/products/:id/stock/add` — the Stock page's "Update Stock"
  action: add newly delivered quantity on top of what's already on hand
  (`{ qty }`). Both stock routes flip `outOfStock` on automatically at zero.
- `GET/PUT /api/products/:id/ingredients` — the recipe linking a menu item
  to the raw `StockItem`(s) it's made from (`{ ingredients: [{stockItemId,
  qty}] }`, PUT replaces the whole list). `qty` is per order, in whatever
  unit that StockItem is counted in — e.g. tracking "Village Chicken" in
  servings rather than whole birds is what makes "1 order = 1 unit" true.
- `GET/POST /api/stock-items`, `PATCH/DELETE /:id` — raw ingredient stock
  for the Stock page (`{ name, quantity }`; `quantity: null` leaves it
  untracked, toggled by hand instead — see `models.StockItem`).
- `GET/POST /api/categories/meal` — list meal categories (plain strings) /
  add a new one (`{ label }`). A category can exist before any product uses
  it, same as the Add Meal form on the frontend.
- `GET/POST /api/categories/drink` — list drink categories (`{key, label}`
  pairs) / add a new one (`{ label }` — the `key` slug is generated
  automatically).
- `GET/POST /api/orders`, plus `PATCH /:id/arrive|complete|cancel|renew|
  renew-cancelled` and `POST /:id/payments` — the full order lifecycle.
  Completing an order (`/:id/complete`) is what decrements stock for any
  tracked product in it — not placing the order. It also walks each item's
  recipe (`ProductIngredient`) and decrements the linked raw `StockItem`(s)
  by the same amount, so ingredient stock reflects actual sales too.
- `GET /api/orders/phone?number=...` — look up an order by customer phone.
  One deliberate difference from the old backend: this is a query
  parameter, not `/phone/:phone` — a phone number can contain a "+", and
  some HTTP clients percent-encode that in a way Go's router doesn't
  reliably match as a path segment. Query values don't have that problem.

## A real bug found and fixed during the rewrite

The old backend stored `customerPhone` exactly as typed (e.g.
`"0977123456"`), but looked it up by first normalizing the *search input*
to digits-only, country-coded form (`"260977123456"`) and doing a
`contains` match against the raw stored value — which could never actually
match, since the stored value was never normalized the same way. This
version normalizes the phone number once, at order-creation time, so the
stored value and a later search are always in the same format.

## What's done since the last update

- **Frontend wired up.** `store.js` is now a real API client (with a shape
  adapter layer) instead of a localStorage store, and every page has been
  switched over and verified against this backend.
- **Staff login.** Real username/password auth: `POST /api/auth/login` and
  `POST /api/auth/logout`, bcrypt-hashed passwords, opaque session tokens
  (`StaffSession`, 24h expiry) sent as `Authorization: Bearer <token>`.
  Every staff-only mutation (`products`, `categories`, `orders` writes,
  `staff-users`) is now enforced server-side via `RequireAuth` middleware,
  not just gated in the UI. `POST /api/staff-users` (requires being logged
  in already) creates additional accounts — there's no self-registration,
  since there'd be no account yet to bootstrap from. `seed/staffuser.go`
  creates the first account once, if none exist.

## What's NOT done yet

- **CallMeBot / mobile money config.** `.env.example` has placeholders for
  the values that used to be hardcoded in `store.js`. The actual WhatsApp
  notification call hasn't been ported over yet — there's a `// TODO`
  marking where it belongs in `routes/orders.go`.
- **Realtime sync.** This is a plain request/response API — a customer's
  phone and a staff tablet won't see each other's changes until one of
  them reloads or polls.
- **Hosting.** This needs to run somewhere reachable from the internet
  once you're ready to go live instead of testing locally. Go compiles to
  a single executable with no runtime dependencies, so this deploys easily
  to almost any host that can run an arbitrary binary (Render, Railway,
  Fly.io, a plain VPS, etc.) — simpler than the Node version in that respect.

## Housekeeping

`scripts/` (if you still see it) held a couple of one-off tools used while
setting this up (creating the `claypot_go` schema, clearing test data) —
they're not part of the running app and can be deleted; it wouldn't delete
because OneDrive had a lock on it at the time. Safe to remove whenever
that clears.
