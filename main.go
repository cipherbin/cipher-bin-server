package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bradford-hamilton/cipher-bin-server/app"
	"github.com/bradford-hamilton/cipher-bin-server/db"
)

func main() {
	// Set up proper logging
	f, err := os.OpenFile("errors.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Create a new connection to our pg database
	db, err := db.New()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create and hydrate our application struct with database
	a := app.New(db)

	// Spin up the app server and start listening
	log.Println("Serving application on PORT: ", 4000)
	log.Fatal(http.ListenAndServe(":4000", a.Mux))
}
