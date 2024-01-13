package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"

	"github.com/cipherbin/cipher-bin-server/internal/db"
	gu "github.com/google/uuid"
)

// postMessage is the HandlerFunc for post requests to /msg (create new message).
func (a *App) postMessage(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var m db.Message
	if err := json.Unmarshal(b, &m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new message record with the provided uuid and message content.
	if err := a.Db.PostMessage(m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// MessageResponse is the shape of the json we return when a user fetches a message
type MessageResponse struct {
	Message string `json:"message"`
}

// isValidUUID takes a string and verifies it is a valid uuid. Was initially
// going to use a regex instead of 3rd party package, however google's uuid Parse
// method benchmarked 18x faster
func isValidUUID(uuid string) bool {
	_, err := gu.Parse(uuid)
	return err == nil
}

// getMessage is a HandlerFunc for GET requests to /msg (read message).
// Ex: cipherb.in/msg?bin=abc123
func (a *App) getMessage(w http.ResponseWriter, r *http.Request) {
	// Get a message uuid from the "bin" query param
	uuid := r.URL.Query().Get("bin")
	if uuid == "" || !isValidUUID(uuid) {
		http.Error(w, "Could not find anything matching your request", http.StatusNotFound)
		return
	}

	msg, err := a.Db.GetMessageByUUID(uuid)
	if err != nil {
		http.Error(w, "We're sorry, there was an error!", http.StatusInternalServerError)
		return
	}

	// If the message has an ID == 0, there was no error, however the
	// record was not found. 99.9% of the time this is due to the message
	// having already been destroyed
	if msg.ID == 0 {
		e := &MessageResponse{
			Message: "Sorry, this message has either already been viewed and destroyed or it never existed at all",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(e)
		return
	}

	// If we get here then a message has been found and will be returned, so
	// we need to destroy it before we return it
	if err := a.Db.DestroyMessageByUUID(uuid); err != nil {
		http.Error(w, "We're sorry, there was an error!", http.StatusInternalServerError)
		return
	}

	// If the msg has a designated read confirmation email, send it off. Right now I'm
	// not worried about email error handling or making sure to wait for all of the
	// running go routines to finish before process ends, etc.
	if msg.Email != "" {
		go emailReadReceipt(msg)
	}

	// Create a response that only returns the ecrypted message contents, as
	// the front end doesn't need to know about any of the other attributes
	m := MessageResponse{msg.Message}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

// Health check handler
func (a *App) ping(w http.ResponseWriter, r *http.Request) {
	// Check that our db connection is good
	if err := a.Db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("pong"))
}

// emailReadReceipt sends an email to a message's specified email letting
// them know their message has been read and destroyed
func emailReadReceipt(message *db.Message) {
	user := os.Getenv("CIPHER_BIN_EMAIL_USERNAME")
	pass := os.Getenv("CIPHER_BIN_EMAIL_PASSWORD")
	auth := smtp.PlainAuth("", user, pass, "smtp.gmail.com")
	emailBody := "Your message has been viewed and destroyed."

	if message.ReferenceName != "" {
		emailBody = fmt.Sprintf(
			"Your message with reference name: \"%s\" has been viewed and destroyed.",
			message.ReferenceName,
		)
	}

	// A super basic email template
	content := "To: %s\r\nFrom: %s\r\nSubject: Your message has been read.\r\n\r\n\r\n%s\r\n"
	emailBytes := []byte(fmt.Sprintf(content, message.Email, user, emailBody))

	err := smtp.SendMail("smtp.gmail.com:587", auth, user, []string{message.Email}, emailBytes)
	if err != nil {
		log.Printf("error sending email: %+v\n", err)
	}
}
