GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*")
PINGUIN_DIR := tools/pinguin
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@latest
INEFFASSIGN := github.com/gordonklaus/ineffassign@latest

.PHONY: format format-pinguin build lint test test-race test-httpapi test-pinguin tidy tidy-check docker-up docker-down docker-logs ci

format:
	gofmt -w $(GO_SOURCES)

format-pinguin:
	@if [ -d "$(PINGUIN_DIR)" ]; then \
		cd $(PINGUIN_DIR) && gofmt -w $$(find . -name '*.go' -not -path "./vendor/*"); \
	else \
		echo "Skipping format-pinguin: $(PINGUIN_DIR) not found."; \
	fi

build:
	go build ./...

lint:
	go vet ./...
	go run $(STATICCHECK) -checks=all,-SA1019,-ST1000 ./...
	go run $(INEFFASSIGN) ./...

test:
	go test ./...

test-race:
	go test ./... -race -count=1

test-httpapi:
	go test ./internal/httpapi

test-pinguin:
	@if [ -d "$(PINGUIN_DIR)" ]; then \
		cd $(PINGUIN_DIR) && go test ./...; \
	else \
		echo "Skipping test-pinguin: $(PINGUIN_DIR) not found."; \
	fi

tidy:
	go mod tidy

tidy-check:
	go mod tidy
	git diff --exit-code go.mod go.sum

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

ci: tidy-check build lint test-race test-pinguin
