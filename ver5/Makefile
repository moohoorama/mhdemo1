PYTHON ?= python3
.DEFAULT_GOAL := desktop
ANDROID_SDK ?= $(if $(ANDROID_HOME),$(ANDROID_HOME),$(if $(ANDROID_SDK_ROOT),$(ANDROID_SDK_ROOT),$(HOME)/Library/Android/sdk))
ADB ?= $(ANDROID_SDK)/platform-tools/adb
ANDROID_MODE ?= release
ifeq ($(filter $(ANDROID_MODE),release debug),)
$(error ANDROID_MODE must be release or debug)
endif
APK := bin/android/srpg-ver5-$(ANDROID_MODE).apk
ANDROID_BUILD_FLAGS := $(if $(filter release,$(ANDROID_MODE)),--release)

# Desktop release builds are obfuscated with garble to resist reverse engineering.
# garble needs Go >= 1.27 installed outside GOMODCACHE (a plain Homebrew/official
# install), otherwise it cannot overlay the linker. Install once:
#   go install mvdan.cc/garble@latest
GARBLE ?= $(shell go env GOPATH)/bin/garble
GARBLE_FLAGS := -tiny -literals
# ebiten v2.10 uses purego, so macOS and Windows build without CGO and cross-compile.
DESKTOP_ENV := CGO_ENABLED=0
DESKTOP_PKG := ./cmd/srpg-gui

.PHONY: help android android-arm64 android-release android-install \
	desktop desktop-mac desktop-mac-intel desktop-win e2e

help:
	@echo 'make desktop         Build obfuscated macOS (arm64) and Windows (amd64) binaries'
	@echo 'make desktop-mac     Build obfuscated macOS arm64 binary (bin/mac/srpg)'
	@echo 'make desktop-mac-intel  Build obfuscated macOS amd64 binary'
	@echo 'make desktop-win     Build obfuscated Windows amd64 binary (bin/win/srpg.exe)'
	@echo 'make android         Build APK for ARM64 phones and x86_64 emulators'
	@echo 'make android-arm64   Build APK for ARM64 phones only'
	@echo 'make android-release Build the release-tagged ARM64 APK'
	@echo 'make android-install Install the existing APK on a connected device'
	@echo 'make android android-install ANDROID_MODE=debug  Build and install with debug tools and autoplay'
	@echo 'Override SDK path:   make android ANDROID_SDK=/path/to/sdk'
	@echo 'make e2e             Play a new game on 쉬움 from start to the ending through the GUI (about 6 min)'
	@echo 'make e2e E2E_SEED=7  Another seed: other sorties, deputies, pieces and traits'

desktop: desktop-mac desktop-win

desktop-mac:
	$(DESKTOP_ENV) GOOS=darwin GOARCH=arm64 $(GARBLE) $(GARBLE_FLAGS) build \
		-tags release -trimpath -ldflags="-s -w" -o bin/mac/srpg $(DESKTOP_PKG)

desktop-mac-intel:
	$(DESKTOP_ENV) GOOS=darwin GOARCH=amd64 $(GARBLE) $(GARBLE_FLAGS) build \
		-tags release -trimpath -ldflags="-s -w" -o bin/mac-intel/srpg $(DESKTOP_PKG)

desktop-win:
	$(DESKTOP_ENV) GOOS=windows GOARCH=amd64 $(GARBLE) $(GARBLE_FLAGS) build \
		-tags release -trimpath -ldflags="-s -w -H=windowsgui" -o bin/win/srpg.exe $(DESKTOP_PKG)

android:
	$(PYTHON) tools/build_android.py --sdk "$(ANDROID_SDK)" $(ANDROID_BUILD_FLAGS) --output "$(APK)"

android-arm64:
	$(PYTHON) tools/build_android.py --sdk "$(ANDROID_SDK)" --targets android/arm64 $(ANDROID_BUILD_FLAGS) --output "$(APK)"

android-release:
	$(PYTHON) tools/build_android.py --sdk "$(ANDROID_SDK)" --targets android/arm64 --release

android-install:
	"$(ADB)" install -r "$(APK)"

# The end-to-end test: srpg-gui's own game loop plays a new game on 쉬움 (casual) to the ending with the debug
# 자동실행, the preparations drawn from the seed (internal/gui/campaign_e2e_test.go). Run it alone: it opens a window
# and keeps the machine busy.
E2E_SEED ?= 1
E2E_MINUTES ?= 60
e2e:
	SRPG_E2E=1 SRPG_E2E_SEED=$(E2E_SEED) SRPG_E2E_MINUTES=$(E2E_MINUTES) caffeinate -d -i \
		go test -p 1 ./internal/gui -run TestCampaignEndToEnd -count=1 -timeout 0 -v
