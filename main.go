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

	a := app.New(db)

	// Spin off go routine that checks every minute for stale messages.
	// If a message is 30 days or older, it will be destroyed. Not concerned
	// here about kill signals, waiting for any last in flight routines to
	// finish, etc
	go func() {
		uptimeTicker := time.NewTicker(60 * time.Second)
		for {
			select {
			case <-uptimeTicker.C:
				a.Db.DestroyStaleMessages()
			}
		}
	}()

	port := os.Getenv("CIPHER_BIN_SERVER_PORT")
	if port == "" {
		port = "4000"
	}

	fmt.Printf("Serving application on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), a.Mux))
}
