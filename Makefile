.PHONY: build docker run migrate source

GIT_HASH := $(shell git rev-parse HEAD)

source:
	source "${PWD}/env.sh"

migrate: source
	psql -d ${CIPHER_BIN_DB_NAME} -f "${PWD}/internal/db/init.sql"

run: source
	go run main.go

docker-build:
	docker buildx build --platform=linux/amd64 -t bradfordhamilton/cipher-bin-server:$(GIT_HASH) .
	docker buildx build --platform=linux/amd64 -t bradfordhamilton/cipher-bin-server:latest .
	docker buildx build --platform=linux/amd64 -t $(CIPHER_BIN_AWS_ECR)/cipher-bin-server:$(GIT_HASH) .
	docker buildx build --platform=linux/amd64 -t $(CIPHER_BIN_AWS_ECR)/cipher-bin-server:latest .

docker-push:
	docker push bradfordhamilton/cipher-bin-server:$(GIT_HASH)
	docker push bradfordhamilton/cipher-bin-server:latest
	docker push $(CIPHER_BIN_AWS_ECR)/cipher-bin-server:$(GIT_HASH)
	docker push $(CIPHER_BIN_AWS_ECR)/cipher-bin-server:latest

docker: docker-build docker-push
