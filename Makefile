.PHONY: all build test clean proto docker-build docker-up docker-down

# Variables
SERVICES := user_service post_service comment_service

all: build

build:
	@for service in $(SERVICES); do \
		go build -o bin/$$service ./cmd/$$service; \
	done

test:
	go test ./...

clean:
	rm -rf bin/*

proto:
	buf generate

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run-user:
	go run ./cmd/user_service

run-post:
	go run ./cmd/post_service

run-comment:
	go run ./cmd/comment_service

# Add more specific commands as needed

# Buf-related commands (adjust as necessary based on your buf setup)
buf-lint:
	buf lint

buf-breaking:
	buf breaking --against '.git#branch=main'