GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*" -not -path "./.cache/*" -not -path "./tools/pinguin/.cache/*")
PINGUIN_DIR := tools/pinguin
STATICCHECK_VERSION ?= v0.6.1
INEFFASSIGN_VERSION ?= v0.2.0
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
INEFFASSIGN := github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
RELEASE_ARGS ?=
PUBLISH_ARGS ?=
DEPLOY_ARGS ?=
RELEASE_HELPER ?=
DOCKER_IMAGE ?= ghcr.io/tyemirov/loopaware
PUBLISH_PLATFORMS ?= linux/amd64
PUBLISH_REMOTE ?= origin
PUBLISH_BRANCH ?= master
GATEWAY_DIR ?=
APP_MANIFEST ?= $(CURDIR)/.mprlab/deploy/app.yml
CLIENT_REACT_NATIVE_DIR := clients/react-native
CLIENT_REACT_NATIVE_NPM ?= npm
CLIENT_REACT_NATIVE_NPM_COMMAND ?= env -u NO_COLOR $(CLIENT_REACT_NATIVE_NPM)
MOBILE_DIR := mobile
MOBILE_NPM ?= npm
MOBILE_NPM_COMMAND ?= env -u NO_COLOR $(MOBILE_NPM)
MOBILE_ANDROID_PACKAGE := com.mprlab.loopaware
MOBILE_IOS_BUNDLE_IDENTIFIER := com.mprlab.loopaware
MOBILE_GOOGLE_IOS_REDIRECT_URI ?= $(or $(LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI),com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg:/oauth2redirect/google)
MOBILE_RELEASE_TIMESTAMP ?=
MOBILE_RESOLVED_RELEASE_TIMESTAMP := $(if $(strip $(MOBILE_RELEASE_TIMESTAMP)),$(MOBILE_RELEASE_TIMESTAMP),$(shell date -u +%Y-%m-%dT%H:%M:%SZ))
MOBILE_IOS_DEVELOPMENT_TEAM ?= $(or $(APPLE_TEAM_ID),Z9ZW6HDGML)
MOBILE_IOS_ASC_APP_ID ?= $(LOOPAWARE_MOBILE_IOS_ASC_APP_ID)
MOBILE_IOS_PROVIDER_PUBLIC_ID ?=
APP_STORE_CONNECT_API_KEY_ID ?= 82P4KZ86HM
APP_STORE_CONNECT_API_ISSUER_ID ?= 94ecd239-946c-478c-8fe5-5c7f50816959
APP_STORE_CONNECT_API_KEY_PATH ?= $(CURDIR)/configs/AuthKey_82P4KZ86HM.p8
MOBILE_IOS_BUILD_DIR ?= /tmp/loopaware-mobile-ios-archive
MOBILE_IOS_ARCHIVE_ARGS ?=
MOBILE_ANDROID_BUNDLE_ARGS ?=
MOBILE_IOS_SUBMIT_ARGS ?=
MOBILE_ANDROID_PUBLISH_ARGS ?=
MOBILE_METRO_PORT_RESOLVER := $(MOBILE_DIR)/scripts/resolve-metro-port.mjs
MOBILE_NATIVE_BUILD_FINGERPRINT := $(MOBILE_DIR)/scripts/native-build-fingerprint.mjs
MOBILE_IOS_ARCHIVE_SCRIPT := $(MOBILE_DIR)/scripts/build-ios-archive.mjs
MOBILE_ANDROID_BUNDLE_SCRIPT := $(MOBILE_DIR)/scripts/build-android-bundle.mjs
MOBILE_IOS_SUBMIT_SCRIPT := $(MOBILE_DIR)/scripts/submit-ios.mjs
MOBILE_ANDROID_PUBLISH_SCRIPT := $(MOBILE_DIR)/scripts/publish-android-play.mjs
ANDROID_SDK_ROOT ?= $(HOME)/Library/Android/sdk
ANDROID_HOME ?= $(ANDROID_SDK_ROOT)
ANDROID_STUDIO_JAVA_HOME ?= /Applications/Android Studio.app/Contents/jbr/Contents/Home
ANDROID_LOCAL_PROPERTIES := $(MOBILE_DIR)/android/local.properties
ANDROID_TOOL_PATH := $(ANDROID_SDK_ROOT)/emulator:$(ANDROID_SDK_ROOT)/platform-tools:$(ANDROID_SDK_ROOT)/cmdline-tools/latest/bin:$(ANDROID_SDK_ROOT)/tools/bin

export GOWORK := off

.PHONY: format format-pinguin build lint lint-js client-react-native-install client-react-native-check mobile-install mobile-check mobile-start run-ios run-android build-ios build-android mobile-android-bundle submit-ios submit-android submit-mobile config-audit test test-unit test-live-favicons test-integration test-integration-api test-integration-all test-race coverage tidy tidy-check up down docker-up docker-down docker-logs ci release publish deploy

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
	@$(MAKE) client-react-native-check
	npm --prefix tests run check:location-map
	@$(MAKE) mobile-check

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
	@metro_port="$${LOOPAWARE_MOBILE_METRO_PORT:-$$(node "$(MOBILE_METRO_PORT_RESOLVER)")}" ; \
	native_fingerprint="$$(LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)" LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)" node "$(MOBILE_NATIVE_BUILD_FINGERPRINT)" ios)" ; \
	native_stamp="$(MOBILE_DIR)/.expo/loopaware-ios-dev-build.sha256" ; \
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

build-ios: mobile-check
	@echo "==> [build-ios] Building LoopAware Mobile iOS artifact"
	@LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)" LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)" LOOPAWARE_MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)" MOBILE_IOS_DEVELOPMENT_TEAM="$(MOBILE_IOS_DEVELOPMENT_TEAM)" node "$(MOBILE_IOS_ARCHIVE_SCRIPT)" --mobile-dir "$(MOBILE_DIR)" --build-dir "$(MOBILE_IOS_BUILD_DIR)" $(MOBILE_IOS_ARCHIVE_ARGS)

build-android: mobile-install
	@$(MAKE) --no-print-directory mobile-android-bundle MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)"

mobile-android-bundle: mobile-check
	@echo "==> [mobile-android-bundle] Building signed LoopAware Mobile Android App Bundle"
	@ANDROID_HOME="$(ANDROID_HOME)" ANDROID_SDK_ROOT="$(ANDROID_SDK_ROOT)" ANDROID_STUDIO_JAVA_HOME="$(ANDROID_STUDIO_JAVA_HOME)" LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)" LOOPAWARE_MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)" node "$(MOBILE_ANDROID_BUNDLE_SCRIPT)" --mobile-dir "$(MOBILE_DIR)" --android-sdk-root "$(ANDROID_SDK_ROOT)" $(MOBILE_ANDROID_BUNDLE_ARGS)

submit-ios: build-ios
	@echo "==> [submit-ios] Submitting LoopAware Mobile iOS IPA to App Store Connect"
	@LOOPAWARE_MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)" MOBILE_IOS_ASC_APP_ID="$(MOBILE_IOS_ASC_APP_ID)" MOBILE_IOS_PROVIDER_PUBLIC_ID="$(MOBILE_IOS_PROVIDER_PUBLIC_ID)" APP_STORE_CONNECT_API_KEY_ID="$(APP_STORE_CONNECT_API_KEY_ID)" APP_STORE_CONNECT_API_ISSUER_ID="$(APP_STORE_CONNECT_API_ISSUER_ID)" APP_STORE_CONNECT_API_KEY_PATH="$(APP_STORE_CONNECT_API_KEY_PATH)" node "$(MOBILE_IOS_SUBMIT_SCRIPT)" --mobile-dir "$(MOBILE_DIR)" $(MOBILE_IOS_SUBMIT_ARGS)

submit-android: mobile-android-bundle
	@echo "==> [submit-android] Submitting LoopAware Mobile Android App Bundle to Google Play"
	@LOOPAWARE_MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)" node "$(MOBILE_ANDROID_PUBLISH_SCRIPT)" --mobile-dir "$(MOBILE_DIR)" $(MOBILE_ANDROID_PUBLISH_ARGS)

submit-mobile:
	@$(MAKE) --no-print-directory submit-ios MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)"
	@$(MAKE) --no-print-directory submit-android MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)"

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
	@RELEASE_HELPER="$(RELEASE_HELPER)" MOBILE_DIR="$(MOBILE_DIR)" MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RELEASE_TIMESTAMP)" MOBILE_IOS_DEVELOPMENT_TEAM="$(MOBILE_IOS_DEVELOPMENT_TEAM)" MOBILE_IOS_BUILD_DIR="$(MOBILE_IOS_BUILD_DIR)" MOBILE_IOS_ARCHIVE_ARGS="$(MOBILE_IOS_ARCHIVE_ARGS)" MOBILE_IOS_SUBMIT_ARGS="$(MOBILE_IOS_SUBMIT_ARGS)" MOBILE_IOS_ASC_APP_ID="$(MOBILE_IOS_ASC_APP_ID)" MOBILE_IOS_PROVIDER_PUBLIC_ID="$(MOBILE_IOS_PROVIDER_PUBLIC_ID)" MOBILE_ANDROID_BUNDLE_ARGS="$(MOBILE_ANDROID_BUNDLE_ARGS)" MOBILE_ANDROID_PUBLISH_ARGS="$(MOBILE_ANDROID_PUBLISH_ARGS)" APP_STORE_CONNECT_API_KEY_ID="$(APP_STORE_CONNECT_API_KEY_ID)" APP_STORE_CONNECT_API_ISSUER_ID="$(APP_STORE_CONNECT_API_ISSUER_ID)" APP_STORE_CONNECT_API_KEY_PATH="$(APP_STORE_CONNECT_API_KEY_PATH)" ANDROID_HOME="$(ANDROID_HOME)" ANDROID_SDK_ROOT="$(ANDROID_SDK_ROOT)" ANDROID_STUDIO_JAVA_HOME="$(ANDROID_STUDIO_JAVA_HOME)" ./scripts/release.sh $(RELEASE_ARGS)

publish:
	@DOCKER_IMAGE="$(DOCKER_IMAGE)" PUBLISH_PLATFORMS="$(PUBLISH_PLATFORMS)" PUBLISH_REMOTE="$(PUBLISH_REMOTE)" PUBLISH_BRANCH="$(PUBLISH_BRANCH)" ./scripts/publish.sh $(PUBLISH_ARGS)

deploy:
	@GATEWAY_DIR="$(GATEWAY_DIR)" APP_MANIFEST="$(APP_MANIFEST)" ./scripts/deploy.sh $(DEPLOY_ARGS)
