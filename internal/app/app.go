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

// App is a struct that holds a chi multiplexer as well as a connection to our database
type App struct {
	Db      *db.Db
	Mux     *chi.Mux
	baseURL string
}

// New takes a *db.Db and creates a chi router, sets up cors rules, sets up
// a handful of middleware, then hydrates an App struct to return a pointer to it
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

	// Create a pointer to an App struct and attach the database
	// as well as the *chi.Mux
	a := &App{Db: db, Mux: r, baseURL: baseURL()}

	// Define routes, the http methods that can be used on them, and their corresponding handlers
	r.Post("/msg", a.postMessage)
	r.Get("/msg", a.getMessage)
	r.Post("/slack-write", a.slackWrite)
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
