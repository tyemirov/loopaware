override SHELL := /bin/sh
MPRLAB_GATEWAY_EXECUTABLE ?= mprlab-gateway
export MPRLAB_GATEWAY_EXECUTABLE
override GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*" -not -path "./.cache/*" -not -path "./tools/pinguin/.cache/*")
override PINGUIN_DIR := tools/pinguin
override STATICCHECK_VERSION := v0.8.2
override INEFFASSIGN_VERSION := v0.2.0
override GOVULNCHECK_VERSION := v1.6.0
override STATICCHECK := honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
override INEFFASSIGN := github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
override GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
override NPM_CONFIG_CACHE := $(CURDIR)/.cache/npm
override GOCACHE := $(CURDIR)/.cache/go-build
override CLIENT_REACT_NATIVE_DIR := clients/react-native
override CLIENT_REACT_NATIVE_NPM_COMMAND := env -u NO_COLOR npm
override MOBILE_DIR := mobile
override MOBILE_NPM_COMMAND := env -u NO_COLOR npm
override MOBILE_ANDROID_PACKAGE := com.mprlab.loopaware
override MOBILE_IOS_BUNDLE_IDENTIFIER := com.mprlab.loopaware
override MOBILE_API_BASE_URL := https://loopaware-api.mprlab.com
override MOBILE_TAUTH_BASE_URL := https://tauth-api.mprlab.com
override MOBILE_TAUTH_TENANT_ID := loopaware
override MOBILE_GOOGLE_IOS_REDIRECT_URI := com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg:/oauth2redirect/google
override MOBILE_METRO_PORT_RESOLVER := $(MOBILE_DIR)/scripts/resolve-metro-port.mjs

ifeq ($(origin ANDROID_SDK_ROOT),undefined)
override ANDROID_SDK_ROOT := $(value HOME)/Library/Android/sdk
else
override ANDROID_SDK_ROOT := $(value ANDROID_SDK_ROOT)
endif
ifeq ($(origin ANDROID_HOME),undefined)
override ANDROID_HOME := $(ANDROID_SDK_ROOT)
else
override ANDROID_HOME := $(value ANDROID_HOME)
endif
ifeq ($(origin ANDROID_STUDIO_JAVA_HOME),undefined)
override ANDROID_STUDIO_JAVA_HOME := /Applications/Android Studio.app/Contents/jbr/Contents/Home
else
override ANDROID_STUDIO_JAVA_HOME := $(value ANDROID_STUDIO_JAVA_HOME)
endif
override ANDROID_TOOL_PATH := $(ANDROID_SDK_ROOT)/emulator:$(ANDROID_SDK_ROOT)/platform-tools:$(ANDROID_SDK_ROOT)/cmdline-tools/latest/bin:$(ANDROID_SDK_ROOT)/tools/bin
JAVA_HOME ?= $(ANDROID_STUDIO_JAVA_HOME)

override LOOPAWARE_MOBILE_ANDROID_PACKAGE := $(MOBILE_ANDROID_PACKAGE)
override LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER := $(MOBILE_IOS_BUNDLE_IDENTIFIER)
override LOOPAWARE_MOBILE_API_BASE_URL := $(MOBILE_API_BASE_URL)
override LOOPAWARE_MOBILE_TAUTH_BASE_URL := $(MOBILE_TAUTH_BASE_URL)
override LOOPAWARE_MOBILE_TAUTH_TENANT_ID := $(MOBILE_TAUTH_TENANT_ID)
override LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI := $(MOBILE_GOOGLE_IOS_REDIRECT_URI)

export ANDROID_HOME
export ANDROID_SDK_ROOT
export ANDROID_STUDIO_JAVA_HOME
export GOCACHE
export GOWORK := off
export LOOPAWARE_MOBILE_ANDROID_PACKAGE
export LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER
export LOOPAWARE_MOBILE_API_BASE_URL
export LOOPAWARE_MOBILE_TAUTH_BASE_URL
export LOOPAWARE_MOBILE_TAUTH_TENANT_ID
export LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI
export NPM_CONFIG_CACHE

.PHONY: format format-pinguin build lint lint-js client-react-native-install client-react-native-check mobile-install mobile-check mobile-start run-ios run-android config-audit security-audit browser-security-audit container-base-audit github-security-audit test test-unit test-live-favicons test-integration-runner test-integration test-integration-privacy test-integration-api test-integration-browser-security test-integration-proxy-security test-integration-all test-race coverage tidy tidy-check up down docker-up docker-down docker-logs ci release publish deploy

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

lint: lint-js
	go vet ./...
	go run $(STATICCHECK) -checks=all,-SA1019,-ST1000 ./...
	go run $(INEFFASSIGN) ./...

lint-js: tests-install client-react-native-check mobile-check
	npm --prefix tests run typecheck
	npm --prefix tests run check:location-map

.PHONY: tests-install
tests-install:
	@if [ ! -d "$(CURDIR)/tests/node_modules" ]; then \
		npm --prefix tests ci; \
	fi

client-react-native-install:
	@if [ ! -d "$(CURDIR)/$(CLIENT_REACT_NATIVE_DIR)/node_modules" ]; then \
		$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) ci; \
	fi

client-react-native-check: client-react-native-install
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run typecheck
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run build
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run verify-package

.PHONY: mobile-development-command-check mobile-source-staging-check mobile-signing-source-check
mobile-development-command-check:
	node --test tests/mobile/development-commands.mjs

mobile-source-staging-check: mobile-install
	node scripts/run-node-tests.mjs tests/mobile/store-source.mjs

mobile-signing-source-check:
	JAVA_HOME="$(JAVA_HOME)" node tests/mobile/android-signing-source.mjs

mobile-install:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) ci

.PHONY: mobile-update-dependencies
mobile-update-dependencies:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) install --package-lock-only --ignore-scripts
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) ci

mobile-check: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run audit:image-size
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run validate-config
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run test:api-boundaries
	$(MAKE) --no-print-directory mobile-app-flow-check
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run typecheck

.PHONY: mobile-app-flow-check
mobile-app-flow-check:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run test:app-flow

MOBILE_DEEPMERGE_ARGUMENTS ?=
.PHONY: mobile-deepmerge-security-check mobile-deepmerge-prepared-check mobile-dependency-audit-check mobile-dependency-audit-controls
mobile-deepmerge-security-check:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run audit:deepmerge -- $(MOBILE_DEEPMERGE_ARGUMENTS)

mobile-deepmerge-prepared-check:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR)/prepared run audit:deepmerge
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR)/prepared run audit:dependencies

.PHONY: mobile-deepmerge-installation-check
mobile-deepmerge-installation-check:
	node mobile/scripts/deepmerge-installations.test.mjs

.PHONY: mobile-deepmerge-preparation-check
mobile-deepmerge-preparation-check:
	node mobile/scripts/deepmerge-preparation.test.mjs

mobile-dependency-audit-controls:
	node mobile/scripts/dependency-audit.test.mjs

mobile-dependency-audit-check: mobile-dependency-audit-controls mobile-deepmerge-preparation-check
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run audit:dependencies

.PHONY: mobile-forge-security-check
mobile-forge-security-check: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run audit:node-forge

.PHONY: mobile-braces-security-check
mobile-braces-security-check: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run audit:braces

.PHONY: mobile-normalize-prepared-text mobile-prepared-install mobile-native-check mobile-adapter-check mobile-release-check mobile-prepare-store mobile-container-check mobile-bundle-check mobile-prepared-tracking-check mobile-android-config-check
mobile-prepare-store: mobile-install
	node mobile/scripts/prepare-store.mjs
	$(MAKE) --no-print-directory mobile-podfile-config-check mobile-release-metadata-check

mobile-normalize-prepared-text:
	cd mobile/prepared && node ../scripts/normalize-native-text.mjs

mobile-prepared-tracking-check:
	node tests/mobile/prepared-tracking.mjs

mobile-prepared-install:
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR)/prepared ci --include=dev

mobile-bundle-check: mobile-prepared-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR)/prepared run audit:braces
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR)/prepared run audit:node-forge
	node tests/mobile/store-bundles.mjs

mobile-container-check:
	node tests/mobile/container-inputs.mjs

.PHONY: mobile-ci-bundletool mobile-ci-bundletool-check
mobile-ci-bundletool:
	node scripts/setup-mobile-ci-bundletool.mjs

mobile-ci-bundletool-check:
	node tests/mobile/bundletool-setup.mjs

.PHONY: mobile-ci-gateway mobile-ci-gateway-check
mobile-ci-gateway:
	node scripts/setup-mobile-ci-gateway.mjs

mobile-ci-gateway-check:
	node tests/mobile/gateway-runtime-setup.mjs

mobile-adapter-check: mobile-ci-gateway-check mobile-ci-bundletool-check
	node tests/mobile/ios-release.mjs
	JAVA_HOME="$(JAVA_HOME)" node tests/mobile/android-signing.mjs

mobile-native-check: mobile-install mobile-scene-check
	node tests/mobile/apple-native.mjs

.PHONY: mobile-scene-check
mobile-scene-check: mobile-install
	node tests/mobile/ios-scene-generation.mjs

mobile-android-config-check: mobile-prepared-install
	node scripts/run-node-tests.mjs tests/mobile/android-config.mjs

.PHONY: mobile-android-policy-check
mobile-android-policy-check: mobile-prepared-install
	node scripts/run-node-tests.mjs --test-name-pattern="compiler policy" tests/mobile/android-config.mjs

.PHONY: mobile-preparation-check
mobile-preparation-check:
	node scripts/run-node-tests.mjs tests/mobile/preparation.mjs

mobile-release-check: mobile-container-check mobile-bundle-check mobile-prepared-tracking-check mobile-adapter-check mobile-native-check mobile-release-metadata-check mobile-android-config-check mobile-source-staging-check
	node scripts/run-node-tests.mjs tests/mobile/preparation.mjs
	cd mobile/prepared && node scripts/verify-store-preparation.mjs

mobile-start: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run start

run-ios: mobile-install
	@set -eu; \
	metro_port="$${LOOPAWARE_MOBILE_METRO_PORT:-$$(node "$(MOBILE_METRO_PORT_RESOLVER)")}"; \
	export EXPO_PACKAGER_PROXY_URL="http://localhost:$${metro_port}"; \
	export LOOPAWARE_MOBILE_METRO_PORT="$${metro_port}"; \
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run ios:dev-build; \
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run ios

run-android: mobile-install
	@set -eu; \
	metro_port="$${LOOPAWARE_MOBILE_METRO_PORT:-$$(node "$(MOBILE_METRO_PORT_RESOLVER)")}"; \
	export JAVA_HOME="$(JAVA_HOME)"; \
	export PATH="$$JAVA_HOME/bin:$(ANDROID_TOOL_PATH):$$PATH"; \
	export LOOPAWARE_MOBILE_METRO_PORT="$${metro_port}"; \
	adb reverse tcp:"$$metro_port" tcp:"$$metro_port" >/dev/null 2>&1 || true; \
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run android:dev-build; \
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run android

test: test-integration

test-unit:
	go run ./tools/testmonitor -- ./...

.PHONY: test-traffic-interval
test-traffic-interval:
	go run ./tools/testmonitor -- ./internal/api -run 'TestSevenDayTraffic|TestParseTrafficInterval' -count=1

.PHONY: test-traffic-unique
TRAFFIC_UNIQUE_TEST_FILTER ?= TestUniqueTraffic
test-traffic-unique:
	go run ./tools/testmonitor -- ./internal/api -run '$(TRAFFIC_UNIQUE_TEST_FILTER)' -count=1

.PHONY: test-favicon
test-favicon:
	go run ./tools/testmonitor -- ./pkg/favicon

test-live-favicons:
	LOOPAWARE_LIVE_FAVICON_TESTS=1 go test ./pkg/favicon -run TestHTTPResolverLiveKnownSitesReturnFavicons -count=1

test-integration-runner:
	node scripts/run-node-tests.mjs tests/lifecycle-entrypoint.test.mjs tests/resource-theme-command.test.mjs
	./tests/scripts/test-integration-runner.sh

test-integration:
	./tests/scripts/run-integration.sh

test-integration-privacy:
	LOOPAWARE_TEST_SUITE=test:privacy ./tests/scripts/run-integration.sh

test-integration-api:
	LOOPAWARE_TEST_SUITE=test:api ./tests/scripts/run-integration.sh

test-integration-browser-security:
	LOOPAWARE_TEST_SUITE=test:browser-security ./tests/scripts/run-integration.sh

test-integration-proxy-security:
	LOOPAWARE_TEST_SUITE=test:proxy-security ./tests/scripts/run-integration.sh

test-integration-all:
	LOOPAWARE_TEST_SUITE=test:all ./tests/scripts/run-integration.sh

test-race:
	go run ./tools/testmonitor -- ./... -race -count=1

coverage:
	@mkdir -p $(CURDIR)/.cache
	go test ./... -coverprofile=$(CURDIR)/.cache/coverage.out -covermode=count
	go tool cover -func=$(CURDIR)/.cache/coverage.out

tidy:
	go mod tidy

tidy-check:
	go mod tidy -diff

config-audit:
	go run ./cmd/configaudit

container-base-audit:
	./scripts/audit-container-bases.sh

browser-security-audit:
	python3 scripts/audit-browser-assets.py

.PHONY: github-workflow-check
github-workflow-check:
	python3 scripts/audit-github-workflow.py
	node tests/github-workflow-audit.mjs

security-audit: browser-security-audit container-base-audit mobile-forge-security-check mobile-braces-security-check github-workflow-check
	go run $(GOVULNCHECK) ./...
	npm --prefix tests audit --audit-level=low
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) audit --audit-level=low
	$(MAKE) --no-print-directory mobile-dependency-audit-check

github-security-audit:
	./scripts/audit-github-repository.sh

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

.PHONY: deployment-npm-visibility-check
deployment-npm-visibility-check: first-failure-pytest-check
	$(MAKE) -f tests/deployment/Makefile test

# CI validates effective Make options before any application prerequisite runs.
loopaware_option_words = $(if $(strip $(1)),$(if $(filter --,$(firstword $(1))),,$(firstword $(1)) $(call loopaware_option_words,$(wordlist 2,$(words $(1)),$(1)))))
loopaware_ci_options = $(strip $(call loopaware_option_words,$(MAKEFLAGS)) $(call loopaware_option_words,$(GNUMAKEFLAGS)))
loopaware_ci_compact_options = $(if $(filter -%,$(firstword $(loopaware_ci_options))),,$(if $(findstring =,$(firstword $(loopaware_ci_options))),,$(firstword $(loopaware_ci_options))))
ifneq ($(filter ci ci-%,$(MAKECMDGOALS)),)
ifneq ($(strip $(filter -k --keep-going,$(loopaware_ci_options)) $(findstring k,$(loopaware_ci_compact_options))),)
$(error LoopAware CI requires serial stages and stops on failure)
endif
.NOTPARALLEL:
endif

.PHONY: ci-stage-policy
ci ci-checks ci-backend ci-mobile ci-integration: ci-stage-policy
ci-stage-policy:
	@set -eu; for specification in "$${MAKEFLAGS-}" "$${GNUMAKEFLAGS-}"; do set -- $$specification; for option do case "$$option" in --) break;; -j1|--jobs=1) ;; -j*|--jobs*|--jobserver*|-k|--keep-going) printf '%s\n' 'LoopAware CI requires serial stages and stops on failure' >&2; exit 2;; -*) ;; *=*) ;; *k*|*j*) printf '%s\n' 'LoopAware CI requires serial stages and stops on failure' >&2; exit 2;; esac; done; done

.PHONY: ci-checks ci-backend ci-mobile ci-integration
ci-checks: first-failure-node-check first-failure-go-check tidy-check config-audit security-audit build lint

ci-backend: test-unit test-race

ci-mobile: mobile-release-check deployment-npm-visibility-check

ci-integration: first-failure-browser-check test-resource-theme test-integration-runner test-integration-all

ci: ci-checks ci-backend ci-mobile ci-integration

release publish deploy:
	@set -eu; application_root="$$(git rev-parse --show-toplevel </dev/null)"; \
	gateway_executable="$${MPRLAB_GATEWAY_EXECUTABLE}"; \
	if ! command -v "$${gateway_executable}" >/dev/null 2>&1; then \
		printf 'required installed Gateway executable is missing: %s\n' "$${gateway_executable}" >&2; \
		exit 2; \
	fi; \
	export CI=1; \
	exec "$${gateway_executable}" "app-$@" --app-root "$${application_root}" </dev/null

.PHONY: mobile-podfile-config-check
mobile-podfile-config-check:
	node --test tests/mobile/podfile-config.mjs

.PHONY: mobile-release-metadata-check
mobile-release-metadata-check: mobile-install
	node scripts/run-node-tests.mjs mobile/scripts/release-metadata.test.mjs

.PHONY: test-resource-theme tests-browser-install
tests-browser-install: tests-install
	npm --prefix tests exec -- playwright install chromium --only-shell

test-resource-theme: tests-browser-install
	node scripts/run-node-tests.mjs tests/resource-theme.test.js

.PHONY: test-resource-page-integration
test-resource-page-integration:
	LOOPAWARE_TEST_SUITE=test:resource-pages ./tests/scripts/run-integration.sh

.PHONY: first-failure-check
FIRST_FAILURE_SCENARIO ?= node-same-file
FIRST_FAILURE_ARGUMENTS ?=
first-failure-check:
	node tests/first-failure/qualify.mjs "$(FIRST_FAILURE_SCENARIO)" $(FIRST_FAILURE_ARGUMENTS)

.PHONY: first-failure-pytest-tools
first-failure-pytest-tools:
	env -i PATH="$${PATH}" HOME="$${HOME}" TMPDIR="$${TMPDIR:-/tmp}" CI=1 UV_KEYRING_PROVIDER=disabled uv --no-config run --no-project --default-index https://pypi.org/simple --python 3.13 --with ansible-core==2.19.8 --with pytest==9.0.2 python -B -c 'import importlib.metadata as metadata; print("ansible-core=" + metadata.version("ansible-core")); print("pytest=" + metadata.version("pytest"))'

.PHONY: first-failure-monitor-format
first-failure-monitor-format:
	gofmt -w tools/testmonitor/main.go

.PHONY: first-failure-node-check first-failure-go-check first-failure-pytest-check first-failure-browser-check
first-failure-node-check:
	node tests/first-failure/qualify.mjs node-same-file
	node tests/first-failure/qualify.mjs node-contracts
	node tests/first-failure/qualify.mjs node-worker-error
	node tests/first-failure/qualify.mjs make-first-failure

first-failure-go-check:
	node tests/first-failure/qualify.mjs go-receipt-channels
	node tests/first-failure/qualify.mjs go-first-failure
	node tests/first-failure/qualify.mjs go-resistant
	node tests/first-failure/qualify.mjs go-cleanup-error
	node tests/first-failure/qualify.mjs go-controls

first-failure-pytest-check: first-failure-pytest-tools
	node tests/first-failure/qualify.mjs pytest-first-failure

first-failure-browser-check: tests-browser-install
	node tests/first-failure/qualify.mjs node-browser-cleanup
	node tests/first-failure/qualify.mjs playwright-first-failure
