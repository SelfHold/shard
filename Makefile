.PHONY: all test race vet build build-windows build-linux build-apk proto aar app-deps app-test app-run clean package-linux icons

# Every build output lands in builds/ at the repo root.
BIN := builds
SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
VERSION := 0.1.0
# One stamp per make run: the app and the daemon built together share it, so
# the app can tell when it talks to a daemon from another build.
STAMP := $(shell date +%m%d.%H%M%S)
BUILD_ID := $(SHA)-$(STAMP)
DEFINES := --dart-define=APP_VERSION=$(VERSION) --dart-define=BUILD_SHA=$(SHA) --dart-define=BUILD_ID=$(BUILD_ID) --dart-define=ENV=$(or $(ENV),dev)
GO_LDFLAGS := -X github.com/SelfHold/shard/core/rpc.Version=$(VERSION) -X github.com/SelfHold/shard/core/rpc.Build=$(BUILD_ID)
NDK ?= $(lastword $(sort $(wildcard $(HOME)/Android/Sdk/ndk/*)))
PROTOC_DART := $(HOME)/.pub-cache/bin/protoc-gen-dart

# Test runs are capped at 1 GB and run one at a time. CI runners have no
# systemd user session, so the cap is skipped there (GitHub sets CI=true).
CAP := $(if $(CI),,systemd-run --user --scope -p MemoryMax=1G -p MemorySwapMax=0 --quiet)
GOTEST := $(CAP) env GOMAXPROCS=2 go test -p 1

all: vet test build

# --- Go core -------------------------------------------------------------

test:
	$(GOTEST) ./core/...

race:
	$(GOTEST) -race ./core/...

vet:
	go vet ./...

# Linux build links FUSE 2 (libfuse.so.2). Use FUSE_TAGS=-tags=fuse3 on
# systems where libfuse3.so.3 exists.
build:
	go build $(FUSE_TAGS) -ldflags '$(GO_LDFLAGS)' -o $(BIN)/ ./cmd/shard ./cmd/shardd

# APKs only for arm64 (phones) and x86_64 (emulators): the Go core is built for these.
build-apk: aar app-deps
	cd app && flutter build apk --release --split-per-abi --target-platform android-arm64,android-x64 $(DEFINES)
	mkdir -p $(BIN)/android
	cp app/build/app/outputs/flutter-apk/*-release.apk $(BIN)/android/
	@ls -lh $(BIN)/android/*.apk

# Desktop app; shardd is put next to it, the app starts it when needed.
# A CMake cache made at another checkout path (the repo was moved or renamed)
# breaks `flutter build linux`, so it is dropped first.
build-linux: build app-deps
	@c=app/build/linux/x64/release/CMakeCache.txt; \
	if [ -f "$$c" ] && ! grep -qxF "CMAKE_CACHEFILE_DIR:INTERNAL=$(CURDIR)/app/build/linux/x64/release" "$$c"; then \
		echo "CMake cache from another checkout path: removing app/build/linux"; \
		rm -rf app/build/linux; \
	fi
	cd app && flutter build linux --release $(DEFINES)
	cp $(BIN)/shardd app/build/linux/x64/release/bundle/
	rm -rf $(BIN)/linux && mkdir -p $(BIN)/linux
	cp -r app/build/linux/x64/release/bundle/. $(BIN)/linux/
	@echo "app: $(BIN)/linux/shard"

# Linux packages from the build-linux output, described in packaging/linux/nfpm.yaml.
# PKG picks the formats (deb, rpm, archlinux; all three by default), e.g.
#   make package-linux PKG=archlinux
# Packages land in $(BIN)/packages/. nfpm runs through `go run` unless it is installed.
PKG ?= deb rpm archlinux
NFPM ?= $(or $(shell command -v nfpm 2>/dev/null),go run github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.49.0)

package-linux: build-linux
	mkdir -p $(BIN)/packages
	@for p in $(PKG); do \
		PKG_VERSION=$(VERSION) $(NFPM) package -f packaging/linux/nfpm.yaml -p $$p -t $(BIN)/packages/ || exit 1; \
	done

# Re-renders the committed PNG icons (app, tray, Android) from the SVGs in branding/.
icons:
	branding/render.sh

# Windows needs no cgo: cgofuse talks to WinFsp directly.
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags '$(GO_LDFLAGS)' -o $(BIN)/windows/ ./cmd/shard ./cmd/shardd

# Go and Dart code from proto/shard.proto (protoc-gen-go, -go-grpc and
# "dart pub global activate protoc_plugin"), Go code from proto/connector.proto
# (the API for connected apps; apps generate their own clients from it).
proto:
	protoc -I proto \
		--go_out=core/rpc/pb --go_opt=paths=source_relative \
		--go-grpc_out=core/rpc/pb --go-grpc_opt=paths=source_relative \
		proto/shard.proto
	protoc -I proto --plugin=protoc-gen-dart=$(PROTOC_DART) --dart_out=grpc:app/lib/data/rpc/gen proto/shard.proto
	protoc -I proto \
		--go_out=core/connector/pb --go_opt=paths=source_relative \
		--go-grpc_out=core/connector/pb --go-grpc_opt=paths=source_relative \
		proto/connector.proto

# The core for Android (gomobile, NDK from ~/Android/Sdk/ndk).
aar:
	ANDROID_NDK_HOME=$(NDK) gomobile bind -target=android/arm64,android/amd64 -androidapi 26 \
		-javapkg dev.selfhold.shard -ldflags='-s -w $(GO_LDFLAGS)' -o app/android/app/libs/shard.aar ./mobilebind

# --- Flutter app ---------------------------------------------------------

app-deps:
	cd app && flutter pub get

app-test: app-deps
	cd app && flutter analyze && flutter test

app-run: build app-deps
	cd app && SHARDD=$(CURDIR)/$(BIN)/shardd flutter run $(DEFINES)

clean:
	rm -rf $(BIN) app/build app/android/app/libs/shard.aar app/android/app/libs/shard-sources.jar
