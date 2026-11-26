VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST := dist

.PHONY: build test release clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk ./cmd/vk

test:
	go vet ./...
	go test ./...

release: clean
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-windows-amd64.exe ./cmd/vk
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-windows-arm64.exe ./cmd/vk
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-macos-arm64 ./cmd/vk
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-macos-amd64 ./cmd/vk
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-linux-amd64 ./cmd/vk
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/vk-linux-arm64 ./cmd/vk
	cd $(DIST) && sha256sum vk-* > SHA256SUMS

clean:
	rm -rf $(DIST)
