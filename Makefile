# Makefile — reproducible build, vet and test for musicle
#
# The full binary needs CGO + ALSA dev headers (the oto audio backend). On a
# machine without those, `make build` fails only on the player package; the
# engine and download packages still build/vet/test via `make check`.

GO ?= go
VERSION ?= 1.6.0

LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all build vet test check clean

all: build

# Full binary build (requires CGO + ALSA dev headers for the audio player).
# The download engine is pure Go (internal/engine + internal/td), so there is
# nothing to prepare: one plain `go build` yields a fully self-contained
# binary. The binary is stripped (-s -w) and UPX-packed when upx is present.
build:
	CGO_ENABLED=1 $(GO) build $(LDFLAGS) -o build/music-le .
	upx --best --lzma build/music-le 2>/dev/null || true

# Vet the packages that do not depend on the audio player (buildable anywhere).
vet:
	$(GO) vet ./internal/... ./bridge/download/...

# Unit tests (engine, terminal-downloader and download helper packages).
test:
	$(GO) test ./internal/... ./bridge/download/...

# Fast local check: everything that compiles without the cgo audio backend.
check: vet test

clean:
	rm -rf build
