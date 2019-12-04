# Docker multi stage build: Allows us to build, optimize, and
# put a small (hopefully) static binary on a scratch image

# Step 1: build executable binary.
FROM golang:alpine AS builder

# Run standard update
RUN apk update

# Create and set working directory to fully qualified path
RUN mkdir -p $GOPATH/src/github.com/bradford-hamilton/cipher-bin-server
WORKDIR $GOPATH/src/github.com/bradford-hamilton/cipher-bin-server

# COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .

# Fetch and verify dependancies - should be cached if we don't change mod/sum
RUN go mod download
RUN go mod verify

# COPY source code to the workspace
COPY . .

# Build the binary & mark the build as statically linked.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/cipher-bin-server .

# STEP 2 build a small image from scratch
FROM scratch

# Copy our static executable.
COPY --from=builder /go/bin/cipher-bin-server /go/bin/cipher-bin-server

# Expose the port our server runs on
EXPOSE 4000

# Run the cipher-bin-server binary.
ENTRYPOINT ["/go/bin/cipher-bin-server"]