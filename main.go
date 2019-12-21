package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bradford-hamilton/cipher-bin-server/app"
	"github.com/bradford-hamilton/cipher-bin-server/db"
)

func main() {
	// Set up proper log file with a MultiWriter that also prints to os.Stdout
	f, err := os.OpenFile("errors.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// Create a new connection to our pg database
	db, err := db.New()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create and hydrate our application struct with database
	a := app.New(db)

	// Spin off go routine that checks every 10 seconds for stale messages.
	// If a message is 30 days or older, it will be destroyed. Not concerned
	// here about kill signals, waiting for last any in flight routines to
	// finish, etc
	go func() {
		uptimeTicker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-uptimeTicker.C:
				a.Db.DestroyStaleMessages()
			}
		}
	}()

	// Set the port
	port := os.Getenv("CIPHER_BIN_SERVER_PORT")
	if port == "" {
		port = "4000"
	}

	// Spin up the app server and start listening
	fmt.Printf("Serving application on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), a.Mux))
}
