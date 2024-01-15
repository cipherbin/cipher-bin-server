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

// Server is the structure that holds all the resources and context for the API.
type Server struct {
	Db      *db.Db
	Mux     *chi.Mux
	baseURL string
}

// NewServer creates and hydrates an *App structure for use.
func NewServer(db *db.Db) *Server {
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

	s := &Server{Db: db, Mux: r, baseURL: baseURL()}

	// Define routes, the http methods that can be used on them, and their corresponding handlers
	r.Post("/msg", s.postMessage)
	r.Get("/msg", s.getMessage)
	r.Get("/ping", s.ping)

	return s
}

func baseURL() string {
	baseURL := "https://cipherb.in"
	if os.Getenv("CIPHER_BIN_ENV") == "development" {
		baseURL = "http://localhost:3000"
	}
	return baseURL
}
