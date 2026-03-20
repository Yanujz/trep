.PHONY: build test test-race test-cover lint vet clean

BIN := trep

build:
	go build -o $(BIN) ./cmd/trep/

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint: vet
	go build ./...

vet:
	go vet ./...

clean:
	rm -f $(BIN) coverage.out
