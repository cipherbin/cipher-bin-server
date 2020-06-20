.PHONY: build run migrate source

source:
	source "${PWD}/env.sh"

migrate: source
	psql -d ${CIPHER_BIN_DB_NAME} -f "${PWD}/db/init.sql"

build: source
	go mod download
	go build -o cipherbin main.go

run: source
	go run main.go
