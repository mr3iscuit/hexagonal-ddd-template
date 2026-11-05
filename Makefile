.PHONY: run build test mutations lint

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test -race ./...

mutations:
	python3 scripts/verify_mutations.py

lint:
	go vet ./...
