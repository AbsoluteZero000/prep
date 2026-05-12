BIN := bin/prep
GOPATH := $(shell go env GOPATH)

.PHONY: build test lint install clean

build:
	go build -o $(BIN) .

test:
	go test ./... -race -count=1

lint:
	go vet ./...
	# install golangci-lint from https://golangci-lint.run to enable full linting

install: build
	cp $(BIN) $(GOPATH)/bin/prep

clean:
	rm -rf bin/
