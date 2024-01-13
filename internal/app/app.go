package app

import (
	"os"
	"time"

	"github.com/cipherbin/cipher-bin-server/internal/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/didip/tollbooth_chi"
)

// App is the structure that holds all the resources and context for the API.
type App struct {
	Db      *db.Db
	Mux     *chi.Mux
	baseURL string
}

// New creates and hydrates an *App structure for use.
func New(db *db.Db) *App {
	r := chi.NewRouter()

	// Set up middleware stack
	r.Use(
		corsMiddleware().Handler,                        // Allow * origins
		render.SetContentType(render.ContentTypeJSON),   // set content-type headers as application/json
		middleware.Logger,                               // log api request calls
		middleware.StripSlashes,                         // strip slashes to no slash URL versions
		middleware.Recoverer,                            // recover from panics without crashing server
		middleware.Timeout(30*time.Second),              // Set a reasonable timeout
		tollbooth_chi.LimitHandler(limiterMiddleware()), // Set a request limiter by ip
	)

	a := &App{Db: db, Mux: r, baseURL: baseURL()}

	// Define routes, the http methods that can be used on them, and their corresponding handlers
	r.Post("/msg", a.postMessage)
	r.Get("/msg", a.getMessage)
	r.Get("/ping", a.ping)

	return a
}

func baseURL() string {
	baseURL := "https://cipherb.in"
	if os.Getenv("CIPHER_BIN_ENV") == "development" {
		baseURL = "http://localhost:3000"
	}
	return baseURL
}
