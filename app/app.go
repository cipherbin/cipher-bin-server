package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/bradford-hamilton/cipher-bin-server/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	gu "github.com/google/uuid"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_chi"
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
	a := &App{db: db, Mux: r}

	// Define routes, the http methods that can be used on them, and their corresponding handlers
	r.Get("/msg", a.getMessage)
	r.Post("/msg", a.postMessage)
	r.Get("/ping", a.ping)

	return a
}

// isValidUUID takes a string and verifies it is a valid uuid. Was initially
// going to use a regex instead of 3rd party package, however google's uuid Parse
// method benchmarked 18x faster
func isValidUUID(uuid string) bool {
	_, err := gu.Parse(uuid)
	return err == nil
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
	if uuid == "" || !isValidUUID(uuid) {
		http.Error(w, "Could not find anything matching your request", http.StatusNotFound)
		return
	}

	// Retrieve a message by it's uuid
	msg, err := a.db.GetMessageByUUID(uuid)
	if err != nil {
		http.Error(w, "We're sorry, there was an error!", http.StatusInternalServerError)
		return
	}

	// If the message has an ID == 0, there was no error, however the
	// record was not found. 99.9% of the time this is due to the message
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
		http.Error(w, "We're sorry, there was an error!", http.StatusInternalServerError)
		return
	}

	if msg.Email != "" {
		err = emailReadReceipt(msg)
		if err != nil {
			fmt.Println(err)
		}
	}

	// 200 OK -> return msg
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
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
	var m db.Message

	// Unmarshal the body bytes into a pointer to our Message struct
	err = json.Unmarshal(b, &m)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create a new message record with the provided uuid and message content
	err = a.db.PostMessage(m)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 200 OK
	w.WriteHeader(http.StatusOK)
}

// Health checks
func (a *App) ping(w http.ResponseWriter, r *http.Request) {
	// Check that our db connection is good
	err := a.db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("pong"))
}

func emailReadReceipt(message *db.Message) error {
	user := os.Getenv("CIPHER_BIN_EMAIL_USERNAME")
	pass := os.Getenv("CIPHER_BIN_EMAIL_PASSWORD")

	// Set up authentication
	auth := smtp.PlainAuth("", user, pass, "smtp.gmail.com")
	emailBody := "Your message has been viewed and destroyed."

	if message.ReferenceName != "" {
		emailBody = fmt.Sprintf(
			"Your message with reference name: \"%s\" has been viewed and destroyed.",
			message.ReferenceName,
		)
	}

	emailBytes := []byte(
		fmt.Sprintf("To: %s\r\n", message.Email) +
			fmt.Sprintf("From: %s\r\n", user) +
			"Subject: Your message has been read.\r\n" +
			"\r\n" +
			fmt.Sprintf("%s\r\n", emailBody),
	)

	// Connect to the server, authenticate, and send the email
	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		"cipherbinservice@gmail.com",
		[]string{message.Email},
		emailBytes,
	)
	if err != nil {
		return err
	}

	return nil
}
