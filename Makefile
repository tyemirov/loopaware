override SHELL := /bin/sh
override GO_SOURCES := $(shell find . -name '*.go' -not -path "./vendor/*" -not -path "./tests/*" -not -path "./tools/pinguin/vendor/*" -not -path "./.cache/*" -not -path "./tools/pinguin/.cache/*")
override PINGUIN_DIR := tools/pinguin
override STATICCHECK_VERSION := v0.6.1
override INEFFASSIGN_VERSION := v0.2.0
override STATICCHECK := honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
override INEFFASSIGN := github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION)
ifeq ($(origin RELEASE_ENV_FILE),undefined)
override RELEASE_ENV_FILE := $(CURDIR)/configs/.env.loopaware
else
override RELEASE_ENV_FILE := $(value RELEASE_ENV_FILE)
endif
override RELEASE_ARTIFACT_TARGETS := mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact
override RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)
override LIFECYCLE_LOCK := $(RELEASE_TOOL_DIR)/with_lifecycle_lock.sh
override LIFECYCLE_RUNNER := $(RELEASE_TOOL_DIR)/run_lifecycle.sh
override NPM_CONFIG_CACHE := $(CURDIR)/.cache/npm
export NPM_CONFIG_CACHE
override GOCACHE := $(CURDIR)/.cache/go-build
export GOCACHE
export RELEASE_ENV_FILE
export RELEASE_ARTIFACT_TARGETS
export PUBLISH_PLATFORMS
override DOCKER_IMAGE := ghcr.io/tyemirov/loopaware
override PUBLISH_PLATFORMS := linux/amd64
override PAGES_URL := https://loopaware.mprlab.com/
override PAGES_BRANCH := gh-pages
override PAGES_DOMAIN := loopaware.mprlab.com
export PAGES_URL
export PAGES_BRANCH
export PAGES_DOMAIN
PAGES_VERSION ?=
override PAGES_VERSION := $(value PAGES_VERSION)
export PAGES_VERSION
override CLIENT_REACT_NATIVE_DIR := clients/react-native
override CLIENT_REACT_NATIVE_NPM := npm
override CLIENT_REACT_NATIVE_NPM_COMMAND := env -u NO_COLOR npm
override MOBILE_DIR := mobile
override MOBILE_NPM := npm
override MOBILE_NPM_COMMAND := env -u NO_COLOR npm
override MOBILE_ANDROID_PACKAGE := com.mprlab.loopaware
override MOBILE_IOS_BUNDLE_IDENTIFIER := com.mprlab.loopaware
override MOBILE_API_BASE_URL := https://loopaware-api.mprlab.com
override MOBILE_TAUTH_BASE_URL := https://tauth-api.mprlab.com
override MOBILE_TAUTH_TENANT_ID := loopaware
override MOBILE_GOOGLE_IOS_REDIRECT_URI := com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg:/oauth2redirect/google
MOBILE_RELEASE_TIMESTAMP ?=
override MOBILE_RELEASE_TIMESTAMP := $(value MOBILE_RELEASE_TIMESTAMP)
override MOBILE_RESOLVED_RELEASE_TIMESTAMP := $(if $(strip $(MOBILE_RELEASE_TIMESTAMP)),$(MOBILE_RELEASE_TIMESTAMP),$(shell date -u +%Y-%m-%dT%H:%M:%SZ))
override MOBILE_IOS_DEVELOPMENT_TEAM := Z9ZW6HDGML
MOBILE_IOS_ASC_APP_ID ?=
override MOBILE_IOS_ASC_APP_ID := $(value MOBILE_IOS_ASC_APP_ID)
MOBILE_IOS_PROVIDER_PUBLIC_ID ?=
override MOBILE_IOS_PROVIDER_PUBLIC_ID := $(value MOBILE_IOS_PROVIDER_PUBLIC_ID)
ifeq ($(origin APP_STORE_CONNECT_API_KEY_ID),undefined)
override APP_STORE_CONNECT_API_KEY_ID := 82P4KZ86HM
else
override APP_STORE_CONNECT_API_KEY_ID := $(value APP_STORE_CONNECT_API_KEY_ID)
endif
ifeq ($(origin APP_STORE_CONNECT_API_ISSUER_ID),undefined)
override APP_STORE_CONNECT_API_ISSUER_ID := 94ecd239-946c-478c-8fe5-5c7f50816959
else
override APP_STORE_CONNECT_API_ISSUER_ID := $(value APP_STORE_CONNECT_API_ISSUER_ID)
endif
ifeq ($(origin APP_STORE_CONNECT_API_KEY_PATH),undefined)
override APP_STORE_CONNECT_API_KEY_PATH := $(CURDIR)/configs/AuthKey_82P4KZ86HM.p8
else
override APP_STORE_CONNECT_API_KEY_PATH := $(value APP_STORE_CONNECT_API_KEY_PATH)
endif
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
override LOOPAWARE_MOBILE_RELEASE_TIMESTAMP := $(MOBILE_RESOLVED_RELEASE_TIMESTAMP)
export LOOPAWARE_MOBILE_ANDROID_PACKAGE
export LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER
export LOOPAWARE_MOBILE_API_BASE_URL
export LOOPAWARE_MOBILE_TAUTH_BASE_URL
export LOOPAWARE_MOBILE_TAUTH_TENANT_ID
export LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI
export LOOPAWARE_MOBILE_RELEASE_TIMESTAMP
export MOBILE_IOS_DEVELOPMENT_TEAM
export MOBILE_IOS_ASC_APP_ID
export MOBILE_IOS_PROVIDER_PUBLIC_ID
export APP_STORE_CONNECT_API_KEY_ID
export APP_STORE_CONNECT_API_ISSUER_ID
export APP_STORE_CONNECT_API_KEY_PATH
export ANDROID_HOME
export ANDROID_SDK_ROOT
export ANDROID_STUDIO_JAVA_HOME
RELEASE_ARTIFACT_DIR ?=
override RELEASE_ARTIFACT_DIR := $(value RELEASE_ARTIFACT_DIR)
RELEASE_SOURCE_COMMIT ?=
override RELEASE_SOURCE_COMMIT := $(value RELEASE_SOURCE_COMMIT)
RELEASE_VERSION ?=
override RELEASE_VERSION := $(value RELEASE_VERSION)
export RELEASE_ARTIFACT_DIR
export RELEASE_SOURCE_COMMIT
export RELEASE_VERSION

export GOWORK := off

ifneq ($(origin RELEASE_ARGS),undefined)
$(error RELEASE_ARGS is not supported; the canonical release lifecycle accepts no raw shell arguments)
endif
ifneq ($(origin PUBLISH_RELEASE_ARGS),undefined)
$(error PUBLISH_RELEASE_ARGS is not supported; the canonical publish lifecycle accepts no raw shell arguments)
endif
ifneq ($(origin DEPLOY_ARGS),undefined)
$(error DEPLOY_ARGS is not supported; the canonical deploy lifecycle accepts no raw shell arguments)
endif
ifneq ($(origin CLIENT_REACT_NATIVE_PUBLISH_ARGS),undefined)
$(error CLIENT_REACT_NATIVE_PUBLISH_ARGS is not supported; the canonical lifecycle publishes the prepared package)
endif
ifneq ($(origin MOBILE_IOS_ARCHIVE_ARGS),undefined)
$(error MOBILE_IOS_ARCHIVE_ARGS is not supported; the canonical lifecycle owns the iOS artifact path)
endif
ifneq ($(origin MOBILE_ANDROID_BUNDLE_ARGS),undefined)
$(error MOBILE_ANDROID_BUNDLE_ARGS is not supported; the canonical lifecycle owns the Android artifact path)
endif
ifneq ($(origin MOBILE_IOS_SUBMIT_ARGS),undefined)
$(error MOBILE_IOS_SUBMIT_ARGS is not supported; the canonical lifecycle owns the App Store destination)
endif
ifneq ($(origin MOBILE_ANDROID_PUBLISH_ARGS),undefined)
$(error MOBILE_ANDROID_PUBLISH_ARGS is not supported; the canonical lifecycle owns the Play destination)
endif
ifneq ($(origin PAGES_DEPLOY_ARGS),undefined)
$(error PAGES_DEPLOY_ARGS is not supported; use the canonical Pages deployment contract)
endif

override LIFECYCLE_GOALS := $(filter release release-dry-run publish publish-dry-run publish-preflight deploy deploy-dry-run,$(MAKECMDGOALS))
override LIFECYCLE_LONG_MAKEFLAGS := $(filter --ignore-errors --touch --question,$(value MAKEFLAGS))
override LIFECYCLE_SHORT_MAKEFLAGS := $(subst -,,$(filter-out --% %=%, $(value MAKEFLAGS)))
ifneq ($(strip $(LIFECYCLE_GOALS)),)
ifneq ($(strip $(LIFECYCLE_LONG_MAKEFLAGS)),)
$(error lifecycle targets reject non-executing or error-ignoring Make flags: $(LIFECYCLE_LONG_MAKEFLAGS))
endif
ifneq ($(findstring i,$(LIFECYCLE_SHORT_MAKEFLAGS)),)
$(error lifecycle targets reject Make's ignore-errors mode)
endif
ifneq ($(findstring t,$(LIFECYCLE_SHORT_MAKEFLAGS)),)
$(error lifecycle targets reject Make's touch mode)
endif
ifneq ($(findstring q,$(LIFECYCLE_SHORT_MAKEFLAGS)),)
$(error lifecycle targets reject Make's question mode)
endif
endif

.PHONY: format format-pinguin build lint lint-js release-workflow-check release-pages-contract-check staged-release-contract-check deploy-dry-run-contract-check publish-preflight-contract-check ios-npm-publication-contract-check lifecycle-orchestration-contract-check client-react-native-install client-react-native-check client-react-native-artifact publish-react-native mobile-install mobile-check mobile-start run-ios run-android build-ios build-android mobile-android-bundle mobile-release-artifacts container-artifacts pages-artifact pages-deploy submit-ios submit-android submit-mobile publish-mobile config-audit test test-unit test-live-favicons test-integration test-integration-api test-integration-all test-race coverage tidy tidy-check up down docker-up docker-down docker-logs ci release release-dry-run publish-release publish-preflight publish-dry-run publish deploy deploy-dry-run

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

lint-js: client-react-native-check mobile-check release-workflow-check
	@if [ ! -d "$(CURDIR)/tests/node_modules" ]; then \
		npm --prefix tests install; \
	fi
	npm --prefix tests run typecheck
	npm --prefix tests run check:location-map

release-workflow-check: release-pages-contract-check staged-release-contract-check deploy-dry-run-contract-check publish-preflight-contract-check ios-npm-publication-contract-check lifecycle-orchestration-contract-check
	node scripts/validate-release-workflow.mjs

release-pages-contract-check:
	bash scripts/test-release-tooling.sh

staged-release-contract-check:
	bash scripts/test-staged-release-artifacts.sh

deploy-dry-run-contract-check:
	bash scripts/test-deploy-dry-run.sh

publish-preflight-contract-check:
	bash scripts/test-publish-preflight.sh

ios-npm-publication-contract-check:
	bash scripts/test-ios-npm-publication.sh

lifecycle-orchestration-contract-check:
	bash scripts/test-lifecycle-orchestration.sh

client-react-native-install:
	@if [ ! -d "$(CURDIR)/$(CLIENT_REACT_NATIVE_DIR)/node_modules" ]; then \
		$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) ci --legacy-peer-deps; \
	fi

client-react-native-check: client-react-native-install
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run typecheck
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run build
	$(CLIENT_REACT_NATIVE_NPM_COMMAND) --prefix $(CLIENT_REACT_NATIVE_DIR) run verify-package

client-react-native-artifact: client-react-native-check
	@test -n "$$RELEASE_ARTIFACT_DIR" || { echo "error: RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
	@test -n "$$RELEASE_SOURCE_COMMIT" || { echo "error: RELEASE_SOURCE_COMMIT is required" >&2; exit 1; }
	@set -e; \
	source_dir="$$(mktemp -d)"; \
	archive="$$(mktemp)"; \
	cleanup() { rm -rf "$$source_dir"; rm -f "$$archive"; }; \
	trap cleanup EXIT; \
	git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:clients/react-native"; \
	tar -xf "$$archive" -C "$$source_dir"; \
	(cd "$$source_dir" && env -u NO_COLOR npm ci --legacy-peer-deps); \
	(cd "$$source_dir" && env -u NO_COLOR npm run typecheck); \
	(cd "$$source_dir" && env -u NO_COLOR npm run build); \
	(cd "$$source_dir" && env -u NO_COLOR npm run verify-package); \
	asset_dir="$$RELEASE_ARTIFACT_DIR/payloads/release-assets"; \
	mkdir -p "$$asset_dir"; \
	rm -f "$$asset_dir"/loopaware-react-native-*.tgz; \
	(cd "$$source_dir" && env -u NO_COLOR npm pack --ignore-scripts --pack-destination "$$asset_dir"); \
	artifact_count="$$(find "$$asset_dir" -maxdepth 1 -type f -name 'loopaware-react-native-*.tgz' | wc -l | tr -d ' ')"; \
	[ "$$artifact_count" = "1" ] || { echo "error: expected exactly one prepared React Native package" >&2; exit 1; }

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
	@node mobile/scripts/build-ios-archive.mjs --mobile-dir mobile

build-android: mobile-android-bundle

mobile-android-bundle: mobile-check
	@echo "==> [mobile-android-bundle] Building signed LoopAware Mobile Android App Bundle"
	@node mobile/scripts/build-android-bundle.mjs --mobile-dir mobile

mobile-release-artifacts: mobile-check
	@test -n "$$RELEASE_ARTIFACT_DIR" || { echo "error: RELEASE_ARTIFACT_DIR is required" >&2; exit 1; }
	@test -n "$$RELEASE_SOURCE_COMMIT" || { echo "error: RELEASE_SOURCE_COMMIT is required" >&2; exit 1; }
	@set -e; \
	source_dir="$$(mktemp -d)"; \
	archive="$$(mktemp)"; \
	cleanup() { rm -rf "$$source_dir"; rm -f "$$archive"; }; \
	trap cleanup EXIT; \
	git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:mobile"; \
	tar -xf "$$archive" -C "$$source_dir"; \
	asset_dir="$$RELEASE_ARTIFACT_DIR/payloads/release-assets"; \
	mkdir -p "$$asset_dir"; \
	node mobile/scripts/build-ios-archive.mjs --mobile-dir "$$source_dir" --output "$$asset_dir/loopaware-ios.ipa" --manifest "$$asset_dir/loopaware-ios.json"; \
	node mobile/scripts/build-android-bundle.mjs --mobile-dir "$$source_dir" --output "$$asset_dir/loopaware-android.aab"

container-artifacts:
	@test -n "$$RELEASE_SOURCE_COMMIT" || { echo "error: RELEASE_SOURCE_COMMIT is required" >&2; exit 1; }
	@context="$$(mktemp -d)"; \
	archive="$$(mktemp)"; \
	cleanup() { rm -rf "$$context"; rm -f "$$archive"; }; \
	trap cleanup EXIT; \
	git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT"; \
	tar -xf "$$archive" -C "$$context"; \
	./scripts/release/prepare_container_artifact.sh --name loopaware --image ghcr.io/tyemirov/loopaware --file "$$context/Dockerfile" --context "$$context" --platforms linux/amd64 --pull --label "org.opencontainers.image.revision=$$RELEASE_SOURCE_COMMIT" --label "org.opencontainers.image.version=$$RELEASE_VERSION" --label "org.opencontainers.image.source=https://github.com/tyemirov/loopaware"

pages-artifact:
	@test -n "$$RELEASE_SOURCE_COMMIT" || { echo "error: RELEASE_SOURCE_COMMIT is required" >&2; exit 1; }
	@source="$$(mktemp -d)"; \
	archive="$$(mktemp)"; \
	cleanup() { rm -rf "$$source"; rm -f "$$archive"; }; \
	trap cleanup EXIT; \
	git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:web"; \
	tar -xf "$$archive" -C "$$source"; \
	./scripts/release/prepare_pages_artifact.sh --source "$$source" --domain loopaware.mprlab.com --exclude tests --exclude node_modules

pages-deploy:
	@set -- --branch "$$PAGES_BRANCH" --url "$$PAGES_URL" --expected-domain "$$PAGES_DOMAIN"; \
	if [ -n "$$PAGES_VERSION" ]; then set -- "$$@" --version "$$PAGES_VERSION"; fi; \
	exec ./scripts/release/deploy_pages_artifact.sh "$$@"

submit-ios: mobile-check
	@echo "==> [submit-ios] Validating LoopAware Mobile iOS IPA with App Store Connect"
	@node mobile/scripts/submit-ios.mjs --mobile-dir mobile --dry-run
	@echo "==> [submit-ios] Submitting LoopAware Mobile iOS IPA to App Store Connect"
	@node mobile/scripts/submit-ios.mjs --mobile-dir mobile

submit-android: mobile-check
	@echo "==> [submit-android] Submitting LoopAware Mobile Android App Bundle to Google Play"
	@node mobile/scripts/publish-android-play.mjs --mobile-dir mobile

submit-mobile: publish-mobile

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
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh release ./scripts/release.sh

release-dry-run:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh release-dry-run ./scripts/release-preflight.sh

publish-release:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/publish-release.sh

publish-preflight:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish-preflight ./scripts/publish-preflight.sh

publish-dry-run:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish-dry-run ./scripts/publish-preflight.sh

publish:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish ./scripts/publish.sh

publish-mobile:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/publish-mobile.sh

publish-react-native:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/publish-react-native.sh

deploy:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh deploy ./scripts/deploy.sh

deploy-dry-run:
	@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh deploy-dry-run ./scripts/deploy.sh --dry-run
