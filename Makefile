.PHONY: run test lint build clean mocks

run:
	go run cmd/api/main.go

test:
	go test -v -race ./...

mocks:
	mockery

build:
	go build -o bin/api cmd/api/main.go

clean:
	rm -rf bin/ items.db*
