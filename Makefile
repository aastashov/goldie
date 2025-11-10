MOCKERY_VERSION := v3.5.5
MOCKERY_BIN := ./bin/mockery

.PHONY: test mock

test:
	@go test -race -cover ./...

mock: $(MOCKERY_BIN)
	@echo "→ Generating mocks..."
	@$(MOCKERY_BIN)

$(MOCKERY_BIN):
	@echo "→ Checking mockery version..."
	@if [ -f $(MOCKERY_BIN) ]; then \
		VERSION=$$($(MOCKERY_BIN) --version | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+'); \
		if [ "$$VERSION" = "$(MOCKERY_VERSION)" ]; then \
			echo "mockery $(MOCKERY_VERSION) already installed."; \
			exit 0; \
		else \
			echo "mockery version ($$VERSION) != $(MOCKERY_VERSION), updating..."; \
			rm -f $(MOCKERY_BIN); \
		fi \
	fi; \
	echo "Installing mockery $(MOCKERY_VERSION)..."; \
	GOBIN=$(abspath ./bin) go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)
