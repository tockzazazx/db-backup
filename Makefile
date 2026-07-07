BINARY := db-backup

.PHONY: build run test lint clean

build:
	go build -o bin/$(BINARY) ./cmd/$(BINARY)

run:
	go run ./cmd/$(BINARY)

test:
	go test ./...

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -rf bin/
