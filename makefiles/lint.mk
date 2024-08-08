lint_version=v1.56.2

lint-install:
	@echo "--> Checking if golangci-lint $(lint_version) is installed"
	@installed_version=$$(golangci-lint --version 2> /dev/null | awk '{print $$4}') || true; \
	if [ "$$installed_version" != "$(lint_version)" ]; then \
		echo "--> Installing golangci-lint $(lint_version)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(lint_version); \
	else \
		echo "--> golangci-lint $(lint_version) is already installed"; \
	fi

lint:
	@$(MAKE) lint-install
	@echo "--> Running linter"
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run  --timeout=10m --concurrency 8 -v

lint-fix:
	@$(MAKE) lint-install
	@echo "--> Running linter"
	@go list -f '{{.Dir}}/...' -m | xargs golangci-lint run  --timeout=10m --fix --concurrency 8 -v
