.PHONY: build run dev

build:
	go build -o bin/bettercopelk ./cmd/server

run:
	go run ./cmd/server/main.go

dev:
	air