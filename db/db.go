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
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=%s",
		os.Getenv("CIPHER_BIN_DB_HOST"),
		os.Getenv("CIPHER_BIN_DB_PORT"),
		os.Getenv("CIPHER_BIN_DB_USER"),
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

type Message struct {
	ID      int    `json:"id"`
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

func (d *Db) GetMessageByUUID(uuid string) *Message {
	// Prepare query, takes a name argument, protects from sql injection
	stmt, err := d.Prepare("SELECT * FROM messages WHERE uuid=$1")
	if err != nil {
		fmt.Println("GetMessageByUUID Preperation Err: ", err)
	}

	// Make query with our stmt, passing in name argument
	rows, err := stmt.Query(uuid)
	if err != nil {
		fmt.Println("GetMessageByUUID Query Err: ", err)
	}

	// Create User struct for holding each row's data
	var m Message

	// Copy the columns from row into the values pointed at by m (Message)
	for rows.Next() {
		err = rows.Scan(
			&m.ID,
			&m.UUID,
			&m.Message,
		)
		if err != nil {
			fmt.Println("Error scanning rows: ", err)
		}
	}

	return &m
}

func (db *Db) PostMessage(uuid, message string) {
	query := `INSERT INTO messages (uuid, message) VALUES ($1, $2);`

	_, err := db.Exec(query, uuid, message)
	if err != nil {
		log.Printf("Error inserting record into database: %s", err)
	}
}
