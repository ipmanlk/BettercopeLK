.PHONY: build run dev test test-race fmt vet clean

build:
	go build -o bin/bettercopelk ./cmd/bettercopelk

run:
	go run ./cmd/bettercopelk

dev:
	air

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin/ tmp/
