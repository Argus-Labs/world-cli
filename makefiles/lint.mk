lint_version = v1.59.1
GOOS := $(shell go env GOOS)

lint-install:
	@echo "--> Checking if golangci-lint $(lint_version) is installed"
ifeq ($(GOOS),windows)
	@where golangci-lint > nul 2>&1 || (echo "--> Installing golangci-lint $(lint_version)" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(lint_version))
else
	@installed_version=$$(golangci-lint --version 2>/dev/null | awk '{print $$4}') || true; \
	if [ "$$installed_version" != "$(lint_version)" ]; then \
		echo "--> Installing golangci-lint $(lint_version)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(lint_version); \
	else \
		echo "--> golangci-lint $(lint_version) is already installed"; \
	fi
endif

lint:
	@$(MAKE) lint-install
	@echo "--> Running linter"
ifeq ($(GOOS),windows)
	@for /f "delims=" %%d in ('go list -f "{{.Dir}}/..." -m') do (golangci-lint run --timeout=10m --concurrency 8 -v %%d)
else
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run --timeout=10m --concurrency 8 -v
endif

lint-fix:
	@$(MAKE) lint-install
	@echo "--> Running linter with fix"
ifeq ($(GOOS),windows)
	@for /f "delims=" %%d in ('go list -f "{{.Dir}}/..." -m') do (golangci-lint run --timeout=10m --fix --concurrency 8 -v %%d)
else
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run --timeout=10m --fix --concurrency 8 -v
endif
