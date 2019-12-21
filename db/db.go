package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	// postgres driver
	_ "github.com/lib/pq"
)

// Db is our database struct used for interacting with the database
type Db struct {
	*sql.DB
}

// New makes a new database using the connection string and
// returns it, otherwise returns the error
func New() (*Db, error) {
	// Don't feel like setting a password on my local db
	p := os.Getenv("CIPHER_BIN_DB_PASSWORD")
	if p != "" {
		p = fmt.Sprintf("password=%s", p)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s %s dbname=%s sslmode=%s",
		os.Getenv("CIPHER_BIN_DB_HOST"),
		os.Getenv("CIPHER_BIN_DB_PORT"),
		os.Getenv("CIPHER_BIN_DB_USER"),
		p,
		os.Getenv("CIPHER_BIN_DB_NAME"),
		os.Getenv("CIPHER_BIN_SSL_MODE"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Check that our connection is good
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &Db{db}, nil
}

// Message represents a record from our "messages" column
type Message struct {
	ID            int    `json:"id"`
	UUID          string `json:"uuid"`
	Message       string `json:"message"`
	Email         string `json:"email"`
	ReferenceName string `json:"reference_name"`
	CreatedAt     string `json:"created_at"`
}

// GetMessageByUUID finds a message by it's UUID or returns an error
func (db *Db) GetMessageByUUID(uuid string) (*Message, error) {
	// Prepare query, takes a uuid argument, protects from sql injection
	stmt, err := db.Prepare("SELECT * FROM messages WHERE uuid=$1")
	if err != nil {
		log.Print("GetMessageByUUID Preperation Err: ", err)
		return nil, err
	}

	// Make query with our prepeared stmt, passing in uuid argument
	rows, err := stmt.Query(uuid)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	// Create Message struct for holding each row's data
	var m Message

	// Copy the columns from the row into the values pointed at by m (Message)
	for rows.Next() {
		err = rows.Scan(
			&m.ID,
			&m.UUID,
			&m.Message,
			&m.Email,
			&m.ReferenceName,
			&m.CreatedAt,
		)
		if err != nil {
			log.Print(err)
			return nil, err
		}
	}

	return &m, nil
}

// PostMessage takes a uuid and a message & inserts a new record into the db
func (db *Db) PostMessage(msg Message) error {
	query := `INSERT INTO messages (uuid, message, email, reference_name) VALUES ($1, $2, $3, $4);`

	// Execute query with uuid and message arguments
	_, err := db.Exec(query, msg.UUID, msg.Message, msg.Email, msg.ReferenceName)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

// DestroyMessageByUUID rakes a uuid and attempts to destroy the associated record
func (db *Db) DestroyMessageByUUID(uuid string) error {
	query := `DELETE FROM messages WHERE uuid=$1;`

	// Execute query with uuid argument
	_, err := db.Exec(query, uuid)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

// DestroyStaleMessages finds all messages older than 30 days and destroys them
func (db *Db) DestroyStaleMessages() error {
	query := `DELETE FROM messages WHERE created_at <= NOW() - INTERVAL '30 days';`
	_, err := db.Exec(query)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}
