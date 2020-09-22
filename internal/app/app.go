package app

import (
	"net/http"
	"os"
	"time"

	"github.com/cipherbin/cipher-bin-server/internal/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
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
	limiter := tollbooth.NewLimiter(3, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Minute * 30,
	})
	limiter.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		return
	})

	r := chi.NewRouter()

	// Define cors rules
	cors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders: []string{"Link"},
		MaxAge:         300, // Maximum value not ignored by any major browsers
	})

	// Set up middleware
	r.Use(
		cors.Handler, // Allow * origins
		render.SetContentType(render.ContentTypeJSON), // set content-type headers as application/json
		middleware.Logger,                   // log api request calls
		middleware.DefaultCompress,          // compress results, mostly gzipping assets and json
		middleware.StripSlashes,             // strip slashes to no slash URL versions
		middleware.Recoverer,                // recover from panics without crashing server
		middleware.Timeout(30*time.Second),  // Set a reasonable timeout
		tollbooth_chi.LimitHandler(limiter), // Set a request limiter by ip
	)

	// Create a pointer to an App struct and attach the database
	// as well as the *chi.Mux
	a := &App{Db: db, Mux: r, baseURL: baseURL()}

	// Define routes, the http methods that can be used on them, and their corresponding handlers
	r.Post("/msg", a.postMessage)
	r.Get("/msg", a.getMessage)
	r.Post("/slack-write", a.slackWrite)
	// r.Post("/slack-read", a.slackRead)
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
