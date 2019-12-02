package app

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bradford-hamilton/cipher-bin-server/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

// App is a struct that holds a chi multiplexer as well as a connection to our database
type App struct {
	db  *db.Db
	Mux *chi.Mux
}

// ErrResponse represents the err response shape when returning json from API
type ErrResponse struct {
	Message string `json:"message"`
}

// New takes a *db.Db and creates a chi router, sets up cors rules, sets up
// a handful of middleware, then hydrates an App struct to return a pointer to it
func New(db *db.Db) *App {
	r := chi.NewRouter()

	// Define cors rules
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Set up middleware
	r.Use(
		render.SetContentType(render.ContentTypeJSON), // set content-type headers as application/json
		middleware.Logger,                  // log api request calls
		middleware.DefaultCompress,         // compress results, mostly gzipping assets and json
		middleware.StripSlashes,            // strip slashes to no slash URL versions
		middleware.Recoverer,               // recover from panics without crashing server
		middleware.Timeout(30*time.Second), // Set a reasonable timeout
		cors.Handler,                       // Allow * origins
	)

	// Create a pointer to an App struct and attach the database
	// as well as the *chi.Mux
	a := &App{db: db, Mux: r}

	// Define routing and there corresponding handlers
	r.Get("/msg", a.getMessage)
	r.Post("/msg", a.postMessage)

	return a
}

// getMessage is a HandlerFunc for GET requests to /msg
// Ex: cipherb.in/msg?bin=abc123
func (a *App) getMessage(w http.ResponseWriter, r *http.Request) {
	// Return early for method not allowed
	if r.Method != "GET" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get a message uuid from the "bin" query param
	uuid := r.URL.Query().Get("bin")
	if uuid == "" {
		http.Error(w, "We're sorry, there was an error!", 500)
		return
	}

	// Retrieve a message by it's uuid
	msg, err := a.db.GetMessageByUUID(uuid)
	if err != nil {
		http.Error(w, "We're sorry, there was an error!", 500)
		return
	}

	// If the message has an ID == 0, there was no error, however the
	// record was not found. 99% of the time this is due to the message
	// having already been destroyed
	if msg.ID == 0 {
		e := &ErrResponse{Message: "Sorry, this message has already been viewed and destroyed"}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(e)
		return
	}

	// If we get here then a message has been found and will be returned, so
	// we need to destroy it before we return it
	err = a.db.DestroyMessageByUUID(uuid)
	if err != nil {
		http.Error(w, "We're sorry, there was an error!", 500)
		return
	}

	// 200 OK -> return msg
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

// MessageBody represents the post body that comes through on
// the postMessage request
type MessageBody struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

// postMessage is a HandlerFunc for post requests to /msg
func (a *App) postMessage(w http.ResponseWriter, r *http.Request) {
	// Return early for method not allowed
	if r.Method != "POST" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the POST body
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer r.Body.Close()

	// Initialize a return MessageBody
	var m MessageBody

	// Unmarshal the body bytes into a pointer to our initialized Message struct
	err = json.Unmarshal(b, &m)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create a new message record with the provided uuid and message content
	err = a.db.PostMessage(m.UUID, m.Message)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 200 OK
	w.WriteHeader(http.StatusOK)
}
