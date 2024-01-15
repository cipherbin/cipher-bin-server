package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cipherbin/cipher-bin-server/internal/app"
	"github.com/cipherbin/cipher-bin-server/internal/db"
)

func main() {
	f, err := os.OpenFile("errors.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	db, err := db.New()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	srv := app.NewServer(db)

	// Spin off go routine that checks every 5 minutes for stale messages.
	// If a message is 30 days or older, it will be destroyed.
	go func() {
		uptimeTicker := time.NewTicker(5 * time.Minute)
		defer uptimeTicker.Stop()

		for range uptimeTicker.C {
			srv.Db.DestroyStaleMessages()
		}
	}()

	port := os.Getenv("CIPHER_BIN_PORT")
	if port == "" {
		port = "4000"
	}

	fmt.Printf("Serving application on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), srv.Mux))
}
