package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	appSrv := app.NewServer(db)

	port := os.Getenv("CIPHER_BIN_PORT")
	if port == "" {
		port = "4000"
	}

	// Implement a graceful shutdown. This allows in-flight requests to complete before
	// shutting down the server, preventing potential data loss or corruption.
	httpServer := &http.Server{Addr: ":" + port, Handler: appSrv.Mux}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	stopTicker := make(chan struct{})

	go func() {
		<-sig

		shutdownCtx, _ := context.WithTimeout(serverCtx, 5*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out, forcing exit")
			}
		}()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err.Error())
		}

		close(stopTicker)
		serverStopCtx()
	}()

	// Spin off go routine that checks every 5 minutes for stale messages.
	// If a message is 30 days or older, it will be destroyed.
	go func() {
		uptimeTicker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-uptimeTicker.C:
				appSrv.Db.DestroyStaleMessages()
			case <-stopTicker:
				uptimeTicker.Stop()
				return
			}
		}
	}()

	fmt.Printf("Serving application on port %s\n", port)

	err = httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err.Error())
	}

	<-serverCtx.Done()
}
