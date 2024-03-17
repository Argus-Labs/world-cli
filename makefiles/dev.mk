clean:
	@echo "--> Cleaning up"
	@echo "--> Running go clean"
	@go clean
	@echo "--> Removing binary"
	@rm -f $(PKGNAME)
	@echo "--> Removing coverage files"
	@find . -type f -name "*.out" -exec rm -f {} \;


install:
	@echo "--> Installing World CLI"
	@echo "--> Building binary"
	@go build -o $(PKGNAME) -v ./cmd/$(PKGNAME)
	@echo "--> Copying binary to /usr/local/bin"
	@mkdir -p /usr/local/bin
	@cp $(PKGNAME) /usr/local/bin
	@chmod +x /usr/local/bin/$(PKGNAME)
	@echo "--> Installed $(PKGNAME) to /usr/local/bin"
	@echo "--> Cleaning up"
	@rm -f $(PKGNAME)
	
.PHONY: clean install