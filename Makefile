VERSION ?= dev
LDFLAGS := -s -w
GOFLAGS := -trimpath

.PHONY: test build build-all clean

test:
	go vet ./...
	go test ./...

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry ./cmd/wiferry

build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry-linux-amd64 ./cmd/wiferry
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry-linux-arm64 ./cmd/wiferry
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/wiferry-windows-amd64.exe ./cmd/wiferry

clean:
	rm -rf dist
