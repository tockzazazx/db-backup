BINARY  := boxdb
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build build-linux deb run test lint clean

build:
	go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY)_linux_amd64 ./cmd/$(BINARY)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY)_linux_arm64 ./cmd/$(BINARY)

deb: build-linux
	./scripts/build-deb.sh amd64 $(VERSION)
	./scripts/build-deb.sh arm64 $(VERSION)

run:
	go run ./cmd/$(BINARY) run

test:
	go test ./...

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -rf bin/ dist/
