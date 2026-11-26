VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST := dist

.PHONY: build test release clean web

web:
	cd web && npm ci && npm run build

build:
	go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk ./cmd/vk

test:
	go vet ./...
	go test ./...

release: web
	scripts/package.sh $(VERSION)

clean:
	rm -rf $(DIST)
