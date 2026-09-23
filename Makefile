VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build run install clean

build:
	go build $(LDFLAGS) -o bin/betterpd ./cmd/betterpd

run:
	go run $(LDFLAGS) ./cmd/betterpd

install:
	go install $(LDFLAGS) ./cmd/betterpd

clean:
	rm -rf bin/
