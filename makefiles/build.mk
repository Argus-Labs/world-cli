SHELL := /bin/bash
goreleaser_version=v1.24.0

goreleaser-install:
	@echo "--> Checking if goreleaser $(goreleaser_version) is installed"
	@if [ "$$(goreleaser --version 2> /dev/null | grep GitVersion | awk '{ print $$2 }')" != "$(goreleaser_version)" ]; then \
		echo "--> Installing goreleaser $(goreleaser_version)"; \
		go install github.com/goreleaser/goreleaser@$(goreleaser_version); \
	else \
		echo "--> goreleaser $(goreleaser_version) is already installed"; \
	fi

build:
	@$(MAKE) goreleaser-install
	@goreleaser build --clean --snapshot
	@echo "--> Build binary is available in the ./dist directory"

release:
	$(eval args_count := $(words $(MAKECMDGOALS)))
	$(eval args_release_tag := $(word 2, $(MAKECMDGOALS)))
	@if [ "$(args_count)" != "2" ]; then \
		echo " [Error] Wrong argument!";\
		echo -e " --> usage: make release <tag-version>\n"; \
		exit 1; \
	fi
	@if [ -z "${GITHUB_TOKEN}" ]; then\
		echo " [Error] GITHUB_TOKEN not found!"; \
		echo " --> Provide GITHUB_TOKEN as env variable, or in a file at '~/.config/goreleaser/github_token'."; \
		echo -e "     (Grant 'repo' scopes permission: https://github.com/settings/tokens/new)\n"; \
		exit 1; \
	fi
	@echo "--> Release Tag: $(args_release_tag)"
	@echo "--> git: tags current commit"
	@git tag -a $(args_release_tag) -m "goreleaser: $(args_release_tag)"
	@echo "--> git: push tag $(args_release_tag)"
	@git push origin $(args_release_tag)
	@echo "--> goreleaser release"

## do-nothing targets for extra unused args passed into @release
%:
	@:

.PHONY: build goreleaser-install release
