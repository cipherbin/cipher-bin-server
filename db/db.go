package db

import (
	"database/sql"
	"fmt"
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
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("CIPHER_BIN_DB_HOST"),
		os.Getenv("CIPHER_BIN_DB_PORT"),
		os.Getenv("CIPHER_BIN_DB_USER"),
		os.Getenv("CIPHER_BIN_DB_PASSWORD"),
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
	ID      int    `json:"id"`
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

// GetMessageByUUID finds a message by it's UUID or returns an error
func (db *Db) GetMessageByUUID(uuid string) (*Message, error) {
	// Prepare query, takes a uuid argument, protects from sql injection
	stmt, err := db.Prepare("SELECT * FROM messages WHERE uuid=$1")
	if err != nil {
		fmt.Println("GetMessageByUUID Preperation Err: ", err)
	}

	// Make query with our prepeared stmt, passing in uuid argument
	rows, err := stmt.Query(uuid)
	if err != nil {
		return nil, err
	}

	// Create Message struct for holding each row's data
	var m Message

	// Copy the columns from row into the values pointed at by m (Message)
	for rows.Next() {
		err = rows.Scan(
			&m.ID,
			&m.UUID,
			&m.Message,
		)
		if err != nil {
			return nil, err
		}
	}

	return &m, nil
}

// PostMessage takes a uuid and a message & inserts a new record into the db
func (db *Db) PostMessage(uuid, message string) error {
	query := `INSERT INTO messages (uuid, message) VALUES ($1, $2);`

	// Execute query with uuid and message arguments
	_, err := db.Exec(query, uuid, message)
	if err != nil {
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
		return err
	}

	return nil
}
