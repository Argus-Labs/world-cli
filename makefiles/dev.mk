clean:
	@echo "--> Cleaning up"
	@echo "--> Running go clean"
	@go clean
	@echo "--> Removing build './dist' directory"
	@rm -rf ./dist
	@echo "--> Removing coverage files"
	@find . -type f -name "*.out" -exec rm -f {} \;


install:
	@echo "--> Installing World CLI"
	@mkdir -p /usr/local/bin
	@echo "--> Building binary, install to /usr/local/bin"
	@goreleaser build --clean --single-target --snapshot -o /usr/local/bin/$(PKGNAME)
	@echo "--> Installed $(PKGNAME) to /usr/local/bin"

.PHONY: clean install
