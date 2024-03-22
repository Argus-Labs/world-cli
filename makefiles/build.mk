goreleaser_version=v1.24.0

goreleaser-install:
	@echo "--> Checking if goreleaser $(goreleaser_version) is installed"
	@if [ $$(goreleaser --version | grep GitVersion | awk '{ print $$2 }') != "$(goreleaser_version)" ]; then \
		echo "--> Installing golangci-lint $(goreleaser_version)"; \
		go install github.com/goreleaser/goreleaser@$(goreleaser_version); \
	else \
		echo "--> goreleaser $(goreleaser_version) is already installed"; \
	fi

build:
	@$(MAKE) goreleaser-install
	goreleaser build --clean --snapshot

.PHONY: build
