package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/popeskul/insdr-messenger/internal/api"
)

func setupRouter(handler api.ServerInterface) http.Handler {
	r := chi.NewRouter()

	// Serve OpenAPI spec
	r.Get("/api/openapi.yaml", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "api/openapi.yaml")
	})

	// Serve Swagger UI
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusMovedPermanently)
	})

	r.Get("/swagger/*", func(w http.ResponseWriter, req *http.Request) {
		http.StripPrefix("/swagger/", http.FileServer(http.Dir("static/swagger-ui"))).ServeHTTP(w, req)
	})

	// Mount API routes
	r.Mount("/", api.Handler(handler))

	return r
}
