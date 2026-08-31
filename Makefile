.PHONY: all build test vet fmt clean

OS ?= all

all:
	go run ./scripts/build out $(OS)

build:
	go run ./scripts/build out all

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf out
