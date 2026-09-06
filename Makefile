VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
BUILD_TIME_UTC := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
# Local dated build version, e.g. v1.38.0-local-20260906-1430. Hour+minute
# disambiguate multiple builds in one day. `git describe` can return a
# desktop-module tag (desktop-vX.Y.Z), so pin the main release line here and
# bump it on release; DATE_VERSION=… overrides wholesale.
DATE_VERSION := v1.38.0-local-$(shell date +%Y%m%d-%H%M)
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.buildTimeUTC=$(BUILD_TIME_UTC)
# Desktop shares main.version etc but lives in the desktop/ module. Use
# webkit2_41: the Linux build host has webkit2gtk-4.1 (not 4.0), and Wails
# 2.13 selects the pkg-config target by build tag (webkit2_41 →
# webkit2gtk-4.1 + libsoup-3.0). Without it wails probes webkit2gtk-4.0 and
# fails on a 4.1-only host.
DESKTOP_LDFLAGS := -s -w \
	-X main.version=$(DATE_VERSION) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.buildTimeUTC=$(BUILD_TIME_UTC)
GOEXE := $(shell go env GOEXE)
# One pin for the Makefile and the CI lint job; see .github/workflows/ci.yml.
GOLANGCI_VERSION := $(shell cat .golangci-version)
WAILS_VERSION := $(shell tr -d '[:space:]' < .wails-version)

.PHONY: build vet fmt lint lint-go lint-install lint-cross lint-update wails-install test desktop-test desktop-test-short desktop-test-times sdk-test sdk-test-race hooks cross clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/reasonix$(GOEXE) ./cmd/reasonix
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/reasonix-plugin-example$(GOEXE) ./cmd/reasonix-plugin-example

# Build the desktop app for the current OS with a dated local version. Wails
# compiles the frontend from desktop/frontend (needs the frontend toolchain).
# Linux: -tags webkit2_41 is required when the host ships webkit2gtk-4.1
# (Wails 2.13 selects the pkg-config target by tag; default probes 4.0).
# Override DESKTOP_TAGS= for other targets (darwin/windows ignore it).
# Output: desktop/build/bin/reasonix-desktop (linux/darwin) or .exe (windows).
DESKTOP_TAGS ?= webkit2_41
.PHONY: desktop-build
desktop-build:
	cd desktop && wails build -tags "$(DESKTOP_TAGS)" -ldflags "$(DESKTOP_LDFLAGS)"

vet:
	go vet ./...

fmt:
	gofmt -w .

# Both gates CI runs, at the version CI pins. Skipping golangci-lint locally
# trades a second here for a ten-minute CI round trip: `modernize` findings in
# particular never surface in `go vet`.
lint: lint-go
	go run ./tools/repolint
	bash scripts/check-wails-pin.sh
	bash scripts/check-wails-pin.test.sh

lint-go:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed; run: make lint-install"; exit 1; }
	@have=$$(golangci-lint version --short 2>/dev/null); want=$$(echo "$(GOLANGCI_VERSION)" | sed 's/^v//'); \
		[ "$$have" = "$$want" ] || echo "warning: local golangci-lint $$have, CI pins $$want (make lint-install)"
	golangci-lint run --timeout=5m ./...

# CGO_ENABLED=0 keeps the install working where a stray clang on PATH shadows
# the toolchain and breaks runtime/cgo.
lint-install:
	CGO_ENABLED=0 go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

lint-update:
	go run ./tools/repolint -update

wails-install:
	bash scripts/check-wails-pin.sh
	go install "github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)"

# Linting one GOOS leaves every //go:build windows and darwin file unchecked.
lint-cross:
	@for t in "linux ." "darwin ." "windows ." "linux desktop" "windows desktop"; do \
		set -- $$t; \
		echo "== golangci-lint GOOS=$$1 ($$2)"; \
		(cd $$2 && GOOS=$$1 golangci-lint run --timeout=5m ./...) || exit 1; \
	done

test:
	go test ./...

desktop-test:
	cd desktop && go test .

desktop-test-short:
	cd desktop && go test -short .

desktop-test-times:
	cd desktop && go test -count=1 -json . | python3 ../scripts/desktop-test-times.py

sdk-test:
	cd sdk/go && go test ./...

sdk-test-race:
	cd sdk/go && go test -race ./...

hooks:
	@git config core.hooksPath .githooks
	@echo "installed: core.hooksPath -> .githooks (pre-push runs go vet)"

cross:
	@mkdir -p dist
	@for p in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		echo "build $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o dist/reasonix-$$os-$$arch$$ext ./cmd/reasonix; \
	done

clean:
	rm -rf bin dist
