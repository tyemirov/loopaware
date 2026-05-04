GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*" -not -path "./.cache/*" -not -path "./tools/pinguin/.cache/*")
PINGUIN_DIR := tools/pinguin
STATICCHECK_VERSION ?= v0.6.1
INEFFASSIGN_VERSION ?= v0.2.0
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
INEFFASSIGN := github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
RELEASE_ARGS ?=
PUBLISH_ARGS ?=
DEPLOY_ARGS ?=
RELEASE_HELPER ?= /Users/tyemirov/Development/agentSkills/gitrelease/scripts/release_helper.py
DOCKER_IMAGE ?= ghcr.io/tyemirov/loopaware
PUBLISH_PLATFORMS ?= linux/amd64
PUBLISH_REMOTE ?= origin
PUBLISH_BRANCH ?= master
GATEWAY_DIR ?= /Users/tyemirov/Development/mprlab-gateway
APP_MANIFEST ?= $(CURDIR)/deploy/app.yml
PAGES_WORKFLOW ?= pages.yml
PAGES_URL ?= https://loopaware.mprlab.com/

export GOWORK := off

.PHONY: format format-pinguin build lint lint-js config-audit test test-unit test-live-favicons test-integration test-integration-api test-integration-all test-race coverage tidy tidy-check up down docker-up docker-down docker-logs ci release publish deploy

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
	@mkdir -p $(CURDIR)/.cache/home
	@if command -v staticcheck >/dev/null 2>&1; then \
		HOME=$(CURDIR)/.cache/home XDG_CACHE_HOME=$(CURDIR)/.cache staticcheck -checks=all,-SA1019,-ST1000 ./...; \
	else \
		go run $(STATICCHECK) -checks=all,-SA1019,-ST1000 ./...; \
	fi
	@if command -v ineffassign >/dev/null 2>&1; then \
		XDG_CACHE_HOME=$(CURDIR)/.cache ineffassign ./...; \
	else \
		go run $(INEFFASSIGN) ./...; \
	fi
	@$(MAKE) lint-js

lint-js:
	@if [ ! -d "$(CURDIR)/tests/node_modules" ]; then \
		npm --prefix tests install; \
	fi
	npm --prefix tests run typecheck

test: test-integration

test-unit:
	go test ./...

test-live-favicons:
	LOOPAWARE_LIVE_FAVICON_TESTS=1 go test ./pkg/favicon -run TestHTTPResolverLiveKnownSitesReturnFavicons -count=1

test-integration:
	./tests/scripts/run-integration.sh

test-integration-api:
	LOOPAWARE_TEST_SUITE=test:api ./tests/scripts/run-integration.sh

test-integration-all:
	LOOPAWARE_TEST_SUITE=test:all ./tests/scripts/run-integration.sh

test-race:
	go test ./... -race -count=1

coverage:
	@mkdir -p $(CURDIR)/.cache
	go test ./... -coverprofile=$(CURDIR)/.cache/coverage.out -covermode=count
	go tool cover -func=$(CURDIR)/.cache/coverage.out

tidy:
	go mod tidy

tidy-check:
	go mod tidy
	git diff --exit-code go.mod go.sum

config-audit:
	go run ./cmd/configaudit

up:
	./scripts/up.sh

down:
	./scripts/down.sh local

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

ci: tidy-check config-audit build lint test-unit test-race test-integration-all

release:
	@RELEASE_HELPER="$(RELEASE_HELPER)" ./scripts/release.sh $(RELEASE_ARGS)

publish:
	@DOCKER_IMAGE="$(DOCKER_IMAGE)" PUBLISH_PLATFORMS="$(PUBLISH_PLATFORMS)" PUBLISH_REMOTE="$(PUBLISH_REMOTE)" PUBLISH_BRANCH="$(PUBLISH_BRANCH)" ./scripts/publish.sh $(PUBLISH_ARGS)

deploy:
	@GATEWAY_DIR="$(GATEWAY_DIR)" APP_MANIFEST="$(APP_MANIFEST)" PAGES_WORKFLOW="$(PAGES_WORKFLOW)" PAGES_URL="$(PAGES_URL)" ./scripts/deploy.sh $(DEPLOY_ARGS)
