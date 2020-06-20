<div align="center">
  <img src="cipher_bin_logo_black.png" alt="cipher bin logo" />
  <h1 align="center">Cipherbin Server</h1>
  <a href="https://goreportcard.com/report/github.com/cipherbin/cipher-bin-server">
    <img src="https://goreportcard.com/badge/github.com/cipherbin/cipher-bin-server" alt="cipher bin logo" align="center" />
  </a>
  <a href="https://godoc.org/github.com/cipherbin/cipher-bin-server">
    <img src="https://godoc.org/github.com/cipherbin/cipher-bin-server?status.svg" alt="cipher bin logo" align="center" />
  </a>
  <a href="https://golang.org/dl">
    <img src="https://img.shields.io/badge/go-1.13.4-9cf.svg" alt="cipher bin logo" align="center" />
  </a>
  <a href="https://github.com/cipherbin/cipher-bin-server/blob/master/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" align="center">
  </a>
</div>
<br />
<br />

Source code for the server, if you are looking for the client side React app [go here](https://github.com/cipherbin/cipher-bin-client). If you are looking for the CLI app [go here](https://github.com/cipherbin/cipher-bin-cli).

## Setup
0. Create postgres db
    ```
    createdb cipher_bin_dev
    ```

1. `cp env.sample.sh env.sh`

2. Update variables in env.sh

3. `make migrate`

## Development
Clone repo and run:
```
go mod download
```

Build it:
```
make build
```

Run it:
```
./cipherbin
```

Or for quicker iterations build and run in one step:
```
make run
```
