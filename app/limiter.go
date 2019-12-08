package app

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Create custom visitor struct which holds the rate limiter for each visitor
// and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Chang the map to hold values of type visitor
var visitors = make(map[string]*visitor)
var mu sync.Mutex

// Run a backround goroutine to remove old entries from the visitors map
func init() {
	go cleanupVisitors()
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, ok := visitors[ip]
	if !ok {
		limiter := rate.NewLimiter(1, 3)
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update last seen time for the visitor
	v.lastSeen = time.Now()

	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for more
// than 3 minutes and delete the entries
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		defer mu.Unlock()

		for ip, v := range visitors {
			if time.Now().Sub(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
	}
}

// rateLimiter is the middleware func to use when setting up chi router so that all
// requests come through here first.
func rateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		limiter := getVisitor(ip)
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
