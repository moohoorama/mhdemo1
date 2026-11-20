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
MAKENSIS ?= makensis
LLVM_RC ?= llvm-rc
LLVM_CVTRES ?= llvm-cvtres

.PHONY: help android android-arm64 android-release android-install \
	desktop desktop-mac desktop-mac-intel desktop-win desktop-win-plain windows-portable-plain windows-installer windows-installer-plain windows-icon e2e scenario-doc

help:
	@echo 'make desktop         Build obfuscated macOS (arm64) and Windows (amd64) binaries'
	@echo 'make desktop-mac     Build obfuscated macOS arm64 binary (bin/mac/srpg)'
	@echo 'make desktop-mac-intel  Build obfuscated macOS amd64 binary'
	@echo 'make desktop-win     Build obfuscated Windows amd64 binary (bin/win/srpg.exe)'
	@echo 'make windows-installer Build Windows installer with assets (requires NSIS)'
	@echo 'make windows-installer-plain Build a non-obfuscated installer for comparison'
	@echo 'make windows-portable-plain Build a single Windows executable without garble'
	@echo 'make windows-icon   Rebuild Windows icons (requires uv, Pillow and LLVM)'
	@echo 'make android         Build APK for ARM64 phones and x86_64 emulators'
	@echo 'make android-arm64   Build APK for ARM64 phones only'
	@echo 'make android-release Build the release-tagged ARM64 APK'
	@echo 'make android-install Install the existing APK on a connected device'
	@echo 'make android android-install ANDROID_MODE=debug  Build and install with debug tools and autoplay'
	@echo 'Override SDK path:   make android ANDROID_SDK=/path/to/sdk'
	@echo 'make scenario-doc    Write the whole story as readable scripts to output/ (scenes, exploration, battles)'
	@echo 'make e2e             Play a new game on 쉬움 from start to the ending through the GUI (about 6 min)'
	@echo 'make e2e E2E_SEED=7  Another seed: other sorties, deputies, pieces and traits'
	@echo 'make e2e E2E_ROUTE=virtual  The 가상 telling from 24장 (「느린 길」) instead of 연의'

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

desktop-win-plain:
	$(DESKTOP_ENV) GOOS=windows GOARCH=amd64 go build -tags release -trimpath -ldflags="-s -w -H=windowsgui" -o bin/win/plain/srpg.exe $(DESKTOP_PKG)

windows-portable-plain:
	$(DESKTOP_ENV) GOOS=windows GOARCH=amd64 go build -tags 'release portable' -trimpath -ldflags="-s -w -H=windowsgui" -o bin/win/srpg-ver5-portable-plain.exe $(DESKTOP_PKG)
	$(PYTHON) tools/windows/append_assets.py bin/win/srpg-ver5-portable-plain.exe

# The checked-in COFF resource lets release builds run without LLVM or Pillow.
windows-icon:
	mkdir -p bin/win
	uv run --with pillow python -c 'from PIL import Image; Image.open("assets/graphics/ui/app_icon.png").convert("RGBA").save("tools/windows/app.ico", format="ICO", bitmap_format="bmp", sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])'
	$(LLVM_RC) /fo bin/win/app.res tools/windows/app.rc
	$(LLVM_CVTRES) /machine:x64 /out:cmd/srpg-gui/icon_windows_amd64.syso bin/win/app.res

windows-installer: desktop-win
	$(MAKENSIS) tools/windows/installer.nsi

windows-installer-plain: desktop-win-plain
	$(MAKENSIS) -DGAME_EXE=../../bin/win/plain/srpg.exe -DINSTALLER_EXE=../../bin/win/srpg-ver5-setup-plain.exe tools/windows/installer.nsi

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
# E2E_ROUTE picks the telling from 24장 융중대: yanyi (「빠른 길」, ends at c27-end) or virtual (「느린 길」, ends at
# c27v-end; the test sets flag-zhugexuan/flag-guanyu-mifang the run missed so that the choice comes, and fails
# when 고순 is no longer in the party).
E2E_SEED ?= 1
E2E_MINUTES ?= 60
E2E_ROUTE ?= yanyi
e2e:
	SRPG_E2E=1 SRPG_E2E_SEED=$(E2E_SEED) SRPG_E2E_MINUTES=$(E2E_MINUTES) SRPG_E2E_ROUTE=$(E2E_ROUTE) caffeinate -d -i \
		go test -p 1 ./internal/gui -run TestCampaignEndToEnd -count=1 -timeout 0 -v

scenario-doc:
	$(PYTHON) tools/scenario_doc/build.py --out output
