package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"

	"github.com/cipherbin/cipher-bin-cli/pkg/aes256"
	"github.com/cipherbin/cipher-bin-cli/pkg/randstring"
	"github.com/cipherbin/cipher-bin-server/internal/db"
	gu "github.com/google/uuid"
)

// postMessage is a HandlerFunc for post requests to /msg
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

	var m db.Message
	if err := json.Unmarshal(b, &m); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create a new message record with the provided uuid and message content
	if err := a.Db.PostMessage(m); err != nil {
		http.Error(w, err.Error(), 500)
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

// getMessage is a HandlerFunc for GET requests to /msg
// Ex: cipherb.in/msg?bin=abc123
func (a *App) getMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

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

// SlackResponse represents the http response we receive when posting to slash commands
type SlackResponse struct {
	Token          string `json:"token,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	TeamDomain     string `json:"team_domain,omitempty"`
	EnterpriseName string `json:"enterprise_name,omitempty"`
	EnterpriseID   string `json:"enterprise_id,omitempty"`
	ChannelID      string `json:"channel_id,omitempty"`
	ChannelName    string `json:"channel_name,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	UserName       string `json:"user_name,omitempty"`
	Command        string `json:"command,omitempty"`
	Text           string `json:"text,omitempty"`
	APIAppID       string `json:"api_app_id,omitempty"`
	ResponseURL    string `json:"response_url,omitempty"`
	TriggerID      string `json:"trigger_id,omitempty"`
}

func (a *App) slackWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	sr, err := parseSlackResponse(r)
	if err != nil {
		http.Error(w, "We're sorry, there was an error!", http.StatusInternalServerError)
		return
	}

	uuidv4 := gu.New().String()
	key := randstring.New(32)

	// Encrypt the message using the shared cipherbin CLI package aes256
	encryptedMsg, err := aes256.Encrypt([]byte(sr.Text), key)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	msg := db.Message{UUID: uuidv4, Message: encryptedMsg}
	if err := a.Db.PostMessage(msg); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte(fmt.Sprintf("%s/msg?bin=%s;%s", a.baseURL, uuidv4, key)))
	w.WriteHeader(http.StatusOK)
}

func parseSlackResponse(r *http.Request) (SlackResponse, error) {
	if err := r.ParseForm(); err != nil {
		return SlackResponse{}, err
	}
	return SlackResponse{
		Token:          r.PostForm.Get("token"),
		TeamID:         r.PostForm.Get("team_id"),
		TeamDomain:     r.PostForm.Get("team_domain"),
		EnterpriseID:   r.PostForm.Get("enterprise_id"),
		EnterpriseName: r.PostForm.Get("enterprise_name"),
		ChannelID:      r.PostForm.Get("channel_id"),
		ChannelName:    r.PostForm.Get("channel_name"),
		UserID:         r.PostForm.Get("user_id"),
		UserName:       r.PostForm.Get("user_name"),
		Command:        r.PostForm.Get("command"),
		Text:           r.PostForm.Get("text"),
		APIAppID:       r.PostForm.Get("app_api_id"),
		ResponseURL:    r.PostForm.Get("response_url"),
		TriggerID:      r.PostForm.Get("trigger_id"),
	}, nil
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

	// Set up authentication
	auth := smtp.PlainAuth("", user, pass, "smtp.gmail.com")
	emailBody := "Your message has been viewed and destroyed."

	if message.ReferenceName != "" {
		emailBody = fmt.Sprintf(
			"Your message with reference name: \"%s\" has been viewed and destroyed.",
			message.ReferenceName,
		)
	}

	// A super basic email template
	emailBytes := []byte(
		fmt.Sprintf(
			"To: %s\r\nFrom: %s\r\nSubject: Your message has been read.\r\n\r\n\r\n%s\r\n",
			message.Email,
			user,
			emailBody,
		),
	)

	// Connect to the server, authenticate, and send the email
	smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		user,
		[]string{message.Email},
		emailBytes,
	)
}
