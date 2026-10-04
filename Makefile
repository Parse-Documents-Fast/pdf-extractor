.PHONY: build test lint run clean

build:
	go build -o bin/pdf-extractor ./cmd/pdf-extractor

test:
	go test -v ./...

lint:
	golangci-lint run

run: build
	./bin/pdf-extractor

clean:
	rm -rf bin/
