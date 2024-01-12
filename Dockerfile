# Unfortunately had to ditch the 2 step builder pattern, putting a binary onto
# a "scratch" image... When trying to use the Go smtp package it did not have
# access to some machine level crypto/tls/certificate functionality needed.
# Will come back to this later.

# Step 1: build executable binary.
FROM golang:alpine

# Add maintainer info
LABEL maintainer="Bradford Lamson-Scribner <brad.lamson@gmail.com>"

# Run standard update
RUN apk update

# Create and set working directory to fully qualified path
RUN mkdir /app
WORKDIR /app

# COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .

# Fetch and verify dependancies - should be cached if we don't change mod/sum
RUN go mod download
RUN go mod verify

# COPY source code to the workspace
COPY . .

# Compile the binary
RUN go build -o main .

# Expose the port our server runs on
EXPOSE 4000

# Run the cipher-bin-server binary.
ENTRYPOINT ["/app/main"]
