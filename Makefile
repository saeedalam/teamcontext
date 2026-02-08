MODULE   := github.com/saeedalam/teamcontext
BINARY   := teamcontext
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: all build install clean test vet lint release

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/teamcontext

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/teamcontext

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

# Cross-compile for all platforms
release: clean
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-$${platform%/*}-$${platform#*/}$$([ "$${platform%/*}" = "windows" ] && echo ".exe") ./cmd/teamcontext; \
		echo "Built: dist/$(BINARY)-$${platform%/*}-$${platform#*/}"; \
	done

# Build for current platform and copy to /usr/local/bin
install-system: build
	cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

# Generate SHA256 checksums for release artifacts
checksums:
	@cd dist && shasum -a 256 * > checksums.txt
	@echo "Checksums written to dist/checksums.txt"
