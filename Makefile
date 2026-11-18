VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath

.PHONY: test build build-all icons e2e ui-test check clean

test:
	go vet ./...
	go test ./...

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry ./cmd/wiferry

build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry-linux-amd64 ./cmd/wiferry
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry-linux-arm64 ./cmd/wiferry
	# -H windowsgui: no console window; quit with Stop on the WiFerry page.
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS) -H windowsgui" -o dist/wiferry-windows-amd64.exe ./cmd/wiferry

# Regenerate web/icon.png, web/apple-touch-icon.png and web/icon.ico.
icons:
	go run ./tools/genicon

# End-to-end test of the real binary over the LAN address (needs curl, python3).
e2e: build
	bash scripts/e2e.sh $(CURDIR)/dist/wiferry $$(mktemp -d)

# Browser test of both pages in headless Chrome (needs google-chrome, node 22+).
ui-test: build
	bash scripts/ui-test.sh $(CURDIR)/dist/wiferry $$(mktemp -d)

check: test e2e ui-test

clean:
	rm -rf dist
