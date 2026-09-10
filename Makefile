.PHONY: build run install clean

build:
	go build -o bin/betterpd ./cmd/betterpd

run:
	go run ./cmd/betterpd

install:
	go install ./cmd/betterpd

clean:
	rm -rf bin/
