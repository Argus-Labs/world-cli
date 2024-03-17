build:
	go build -o $(PKGNAME) -v ./cmd/$(PKGNAME)
	
.PHONY: build