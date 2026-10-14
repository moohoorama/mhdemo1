PYTHON ?= python3
ANDROID_SDK ?= $(if $(ANDROID_HOME),$(ANDROID_HOME),$(if $(ANDROID_SDK_ROOT),$(ANDROID_SDK_ROOT),$(HOME)/Library/Android/sdk))
ADB ?= $(ANDROID_SDK)/platform-tools/adb
APK := bin/android/srpg-ver5-debug.apk

.PHONY: help android android-arm64 android-install

help:
	@echo 'make android         Build APK for ARM64 phones and x86_64 emulators'
	@echo 'make android-arm64   Build APK for ARM64 phones only'
	@echo 'make android-install Install the existing APK on a connected device'
	@echo 'Override SDK path:   make android ANDROID_SDK=/path/to/sdk'

android:
	$(PYTHON) tools/build_android.py --sdk "$(ANDROID_SDK)"

android-arm64:
	$(PYTHON) tools/build_android.py --sdk "$(ANDROID_SDK)" --targets android/arm64

android-install:
	"$(ADB)" install -r "$(APK)"
