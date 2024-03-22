goreleaser_version=v1.24.0

goreleaser-install:
	@echo "--> Checking if goreleaser $(goreleaser_version) is installed"
	@if [ $$(goreleaser --version 2> /dev/null | grep GitVersion | awk '{ print $$2 }') != "$(goreleaser_version)" ]; then \
		echo "--> Installing goreleaser $(goreleaser_version)"; \
		go install github.com/goreleaser/goreleaser@$(goreleaser_version); \
	else \
		echo "--> goreleaser $(goreleaser_version) is already installed"; \
	fi

build:
	@$(MAKE) goreleaser-install
	goreleaser build --clean --snapshot

.PHONY: build
