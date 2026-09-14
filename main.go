// The Clay Pot — Phase 2 backend, Go edition.
// Same job as the earlier Node.js version: a real database + REST API that
// replaces the localStorage prototype in
// ../frontend/The_ClayPot/site/store.js. See README.md for how to run it
// and what each endpoint does.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"claypot-backend/db"
	"claypot-backend/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — reading configuration from the environment instead.")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set. Copy .env.example to .env and fill it in.")
	}

	conn, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}
	if err := db.AutoMigrate(conn); err != nil {
		log.Fatalf("Could not set up the database schema: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	requireAuth := routes.RequireAuth(conn)

	authHandler := &routes.AuthHandler{DB: conn}
	productsHandler := &routes.ProductsHandler{DB: conn}
	categoriesHandler := &routes.CategoriesHandler{DB: conn}
	ordersHandler := &routes.OrdersHandler{DB: conn}

	r.Mount("/api/auth", authHandler.Routes())
	r.Mount("/api/products", productsHandler.Routes(requireAuth))
	r.Mount("/api/categories", categoriesHandler.Routes(requireAuth))
	r.Mount("/api/orders", ordersHandler.Routes(requireAuth))

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	log.Printf("Clay Pot backend (Go) listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
