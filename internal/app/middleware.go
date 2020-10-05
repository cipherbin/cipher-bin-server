package app

import (
	"net/http"
	"time"

	"github.com/didip/tollbooth/v6"
	limiter "github.com/didip/tollbooth/v6/limiter"
	"github.com/go-chi/cors"
)

func corsMiddleware() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders: []string{"Link"},
		MaxAge:         300, // Maximum value not ignored by any major browsers
	})
}

func limiterMiddleware() *limiter.Limiter {
	limiter := tollbooth.NewLimiter(3, &limiter.ExpirableOptions{
		DefaultExpirationTTL: time.Minute * 30,
	})
	limiter.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
		return
	})
	return limiter
}
