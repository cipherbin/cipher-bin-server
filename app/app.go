package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bradford-hamilton/cipher-bin-server/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type App struct {
	db  *db.Db
	Mux *chi.Mux
}

func New(db *db.Db) *App {
	r := chi.NewRouter()

	r.Use(
		render.SetContentType(render.ContentTypeJSON), // set content-type headers as application/json
		middleware.Logger,                  // log api request calls
		middleware.DefaultCompress,         // compress results, mostly gzipping assets and json
		middleware.StripSlashes,            // strip slashes to no slash URL versions
		middleware.Recoverer,               // recover from panics without crashing server
		middleware.Timeout(30*time.Second), // Set a reasonable timeout
	)

	a := &App{db: db, Mux: r}

	r.Get("/msg", a.Message)

	return a
}

// cipherb.in/msg?bin=abc123
func (a *App) Message(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bin := r.URL.Query().Get("bin")
	if bin == "" {
		// Leave and 500
	}

	message := a.db.GetMessageByUUID(bin)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(message)
}
