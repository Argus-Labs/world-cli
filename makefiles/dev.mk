INSTALL_PATH=$(shell go env GOPATH)/bin

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
	@mkdir -p $(INSTALL_PATH)
	@echo "--> Building binary, install to $(INSTALL_PATH)"
	@goreleaser build --clean --single-target --snapshot -o $(INSTALL_PATH)/$(PKGNAME)
	@echo "--> Installed $(PKGNAME) to $(INSTALL_PATH)"

.PHONY: clean install
