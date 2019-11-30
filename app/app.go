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

type App struct {
	db  *db.Db
	Mux *chi.Mux
}

func New(db *db.Db) *App {
	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	r.Use(
		render.SetContentType(render.ContentTypeJSON), // set content-type headers as application/json
		middleware.Logger,                  // log api request calls
		middleware.DefaultCompress,         // compress results, mostly gzipping assets and json
		middleware.StripSlashes,            // strip slashes to no slash URL versions
		middleware.Recoverer,               // recover from panics without crashing server
		middleware.Timeout(30*time.Second), // Set a reasonable timeout
		cors.Handler,                       // Allow * origins
	)

	a := &App{db: db, Mux: r}

	r.Get("/msg", a.getMessage)
	r.Post("/msg", a.postMessage)

	return a
}

// cipherb.in/msg?bin=abc123
func (a *App) getMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bin := r.URL.Query().Get("bin")
	if bin == "" {
		http.Error(w, "Some error happened", 500)
		return
	}

	msg, err := a.db.GetMessageByUUID(bin)
	if err != nil {
		http.Error(w, "Some error happened", 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

type MessageBody struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

func (a *App) postMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer r.Body.Close()

	var m MessageBody

	err = json.Unmarshal(b, &m)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	a.db.PostMessage(m.UUID, m.Message)

	w.WriteHeader(http.StatusOK)
}
