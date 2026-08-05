override SHELL := /bin/sh
override GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*" -not -path "./.cache/*" -not -path "./tools/pinguin/.cache/*")
override PINGUIN_DIR := tools/pinguin
override STATICCHECK_VERSION := v0.6.1
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
override MOBILE_NATIVE_BUILD_FINGERPRINT := $(MOBILE_DIR)/scripts/native-build-fingerprint.mjs

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

.PHONY: format format-pinguin build lint lint-js client-react-native-install client-react-native-check mobile-install mobile-check mobile-start run-ios run-android config-audit security-audit browser-security-audit container-base-audit github-security-audit test test-unit test-live-favicons test-integration test-integration-api test-integration-browser-security test-integration-proxy-security test-integration-all test-race coverage tidy tidy-check up down docker-up docker-down docker-logs ci release publish deploy

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

lint-js: client-react-native-check mobile-check
	@if [ ! -d "$(CURDIR)/tests/node_modules" ]; then \
		npm --prefix tests install; \
	fi
	npm --prefix tests run typecheck
	npm --prefix tests run check:location-map

client-react-native-install:
	@if [ ! -d "$(CURDIR)/$(CLIENT_REACT_NATIVE_DIR)/node_modules" ]; then \
		$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) ci --legacy-peer-deps; \
	fi

client-react-native-check: client-react-native-install
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run typecheck
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run build
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run verify-package

mobile-install:
	@if [ ! -d "$(CURDIR)/$(MOBILE_DIR)/node_modules" ]; then \
		$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) ci; \
	fi

mobile-check: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run validate-config
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run test:api-boundaries
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run typecheck

mobile-start: mobile-install
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run start

run-ios: mobile-install
	@echo "==> [run-ios] Starting LoopAware Mobile for iOS"
	@metro_port="$${LOOPAWARE_MOBILE_METRO_PORT:-$$(node "$(MOBILE_METRO_PORT_RESOLVER)")}"; \
	native_fingerprint="$$(LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)" LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)" node "$(MOBILE_NATIVE_BUILD_FINGERPRINT)" ios)"; \
	native_stamp="$(MOBILE_DIR)/.expo/loopaware-ios-dev-build.sha256"; \
	echo "==> [run-ios] Using Metro port $${metro_port}"; \
	if command -v xcrun >/dev/null 2>&1 && xcrun simctl get_app_container booted "$(MOBILE_IOS_BUNDLE_IDENTIFIER)" >/dev/null 2>&1 && [ -f "$${native_stamp}" ] && [ "$$(cat "$${native_stamp}")" = "$${native_fingerprint}" ]; then \
		:; \
	else \
		echo "==> [run-ios] Development build missing or stale; building and installing it first"; \
		EXPO_PACKAGER_PROXY_URL="http://localhost:$${metro_port}" LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)" LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)" LOOPAWARE_MOBILE_METRO_PORT="$${metro_port}" $(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run ios:dev-build; \
		mkdir -p "$$(dirname "$${native_stamp}")"; \
		printf "%s\n" "$${native_fingerprint}" > "$${native_stamp}"; \
	fi; \
	EXPO_PACKAGER_PROXY_URL="http://localhost:$${metro_port}" LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)" LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)" LOOPAWARE_MOBILE_METRO_PORT="$${metro_port}" $(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run ios

run-android: mobile-install
	@echo "==> [run-android] Starting LoopAware Mobile for Android"
	@ANDROID_HOME="$(ANDROID_HOME)" ANDROID_SDK_ROOT="$(ANDROID_SDK_ROOT)" ANDROID_STUDIO_JAVA_HOME="$(ANDROID_STUDIO_JAVA_HOME)" PATH="$(ANDROID_TOOL_PATH):$$PATH" sh -c 'set -e; \
		metro_port="$${LOOPAWARE_MOBILE_METRO_PORT:-$$(node "$(MOBILE_METRO_PORT_RESOLVER)")}"; \
		native_fingerprint="$$(LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)" node "$(MOBILE_NATIVE_BUILD_FINGERPRINT)" android)"; \
		native_stamp="$(MOBILE_DIR)/.expo/loopaware-android-dev-build.sha256"; \
		echo "==> [run-android] Using Metro port $${metro_port}"; \
		if [ -x "$$ANDROID_STUDIO_JAVA_HOME/bin/java" ]; then \
			export JAVA_HOME="$$ANDROID_STUDIO_JAVA_HOME"; \
			export PATH="$$JAVA_HOME/bin:$$PATH"; \
		fi; \
		adb reverse tcp:"$$metro_port" tcp:"$$metro_port" >/dev/null 2>&1 || true; \
		if adb shell pm list packages "$(MOBILE_ANDROID_PACKAGE)" 2>/dev/null | grep -F "package:$(MOBILE_ANDROID_PACKAGE)" >/dev/null && [ -f "$$native_stamp" ] && [ "$$(cat "$$native_stamp")" = "$$native_fingerprint" ]; then \
			adb shell am force-stop "$(MOBILE_ANDROID_PACKAGE)" >/dev/null 2>&1 || true; \
		else \
			echo "==> [run-android] Development build missing or stale; building and installing it first"; \
			LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)" $(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run android:dev-build; \
			mkdir -p "$$(dirname "$$native_stamp")"; \
			printf "%s\n" "$$native_fingerprint" > "$$native_stamp"; \
		fi; \
		LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)" LOOPAWARE_MOBILE_METRO_PORT="$$metro_port" $(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) run android'

test: test-integration

test-unit:
	go test ./...

test-live-favicons:
	LOOPAWARE_LIVE_FAVICON_TESTS=1 go test ./pkg/favicon -run TestHTTPResolverLiveKnownSitesReturnFavicons -count=1

test-integration:
	./tests/scripts/run-integration.sh

test-integration-api:
	LOOPAWARE_TEST_SUITE=test:api ./tests/scripts/run-integration.sh

test-integration-browser-security:
	LOOPAWARE_TEST_SUITE=test:browser-security ./tests/scripts/run-integration.sh

test-integration-proxy-security:
	LOOPAWARE_TEST_SUITE=test:proxy-security ./tests/scripts/run-integration.sh

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
	go mod tidy -diff

config-audit:
	go run ./cmd/configaudit

container-base-audit:
	./scripts/audit-container-bases.sh

browser-security-audit:
	python3 scripts/audit-browser-assets.py

security-audit: browser-security-audit container-base-audit
	python3 scripts/audit-github-workflow.py
	go run $(GOVULNCHECK) ./...
	npm --prefix tests audit --audit-level=low
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) audit --audit-level=low
	$(MOBILE_NPM_COMMAND) --prefix $(MOBILE_DIR) audit --audit-level=low

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

ci: tidy-check config-audit security-audit build lint test-unit test-race test-integration-all

release publish deploy:
	@application_root="$$(git rev-parse --show-toplevel)"; \
	gateway_root="$$(dirname "$${application_root}")/mprlab-gateway"; \
	if [ ! -d "$${gateway_root}" ]; then \
		printf "required sibling gateway is missing: %s; clone mprlab-gateway at exactly %s\n" \
			"$${gateway_root}" "$${gateway_root}" >&2; \
		exit 2; \
	fi; \
	$(MAKE) --no-print-directory -C "$${gateway_root}" "app-$@" \
		MPRLAB_APP_ROOT="$${application_root}"
