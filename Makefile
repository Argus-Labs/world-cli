# world-cli binary name
PKGNAME = world

all: test build

build:
	go build -o $(PKGNAME) -v ./cmd/$(PKGNAME)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(PKGNAME)

install:
	go install ./cmd/$(PKGNAME)

.PHONY: all build test clean install