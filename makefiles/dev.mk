INSTALL_PATH=$(shell go env GOPATH)/bin

clean:
	@echo "--> Cleaning up"
	@echo "--> Running go clean"
	@go clean
	@echo "--> Removing build './dist' directory"
ifeq ($(OS),Windows_NT)
	@if exist dist rmdir /s /q dist
	@echo "--> Removing coverage files"
	@for /r %%i in (*.out) do del %%i
else
	@rm -rf ./dist
	@echo "--> Removing coverage files"
	@find . -type f -name "*.out" -exec rm -f {} \;
endif

install:
	@echo "--> Installing World CLI"
ifeq ($(OS),Windows_NT)
	@if not exist "$(INSTALL_PATH)" mkdir "$(INSTALL_PATH)"
	@echo "--> Building binary, install to $(INSTALL_PATH)"
	@goreleaser build --clean --single-target --snapshot -o "$(INSTALL_PATH)\$(PKGNAME).exe"
else
	@mkdir -p $(INSTALL_PATH)
	@echo "--> Building binary, install to $(INSTALL_PATH)"
	@goreleaser build --clean --single-target --snapshot -o "$(INSTALL_PATH)/$(PKGNAME)"
endif
	@echo "--> Installed $(PKGNAME) to $(INSTALL_PATH)"

.PHONY: clean install
