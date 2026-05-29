VERSION ?= 0.1.0
BINARY  := sockt
DIST    := dist

LDFLAGS := -s -w -X github.com/SocktDev/CLI/cmd.Version=$(VERSION)
GO      := CGO_ENABLED=0 go

.PHONY: help build release clean test vet

help:
	@echo "Targets:"
	@echo "  make build    Build $(BINARY) for the current platform"
	@echo "  make release  Build release binaries, archives, and checksums in $(DIST)/"
	@echo "  make test     Run unit tests"
	@echo "  make clean    Remove $(DIST)/ and local $(BINARY) binary"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"

build:
	$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

release: vet test
	@mkdir -p $(DIST)
	@set -e; \
	for target in \
		"linux amd64 $(DIST)/sockt-linux-amd64" \
		"linux arm64 $(DIST)/sockt-linux-arm64" \
		"darwin amd64 $(DIST)/sockt-darwin-amd64" \
		"darwin arm64 $(DIST)/sockt-darwin-arm64" \
		"windows amd64 $(DIST)/sockt-windows-amd64.exe"; \
	do \
		set -- $$target; \
		echo "Building $$1/$$2 -> $$3"; \
		GOOS=$$1 GOARCH=$$2 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o "$$3" .; \
	done
	tar -czf $(DIST)/sockt-linux-amd64.tar.gz   -C $(DIST) sockt-linux-amd64
	tar -czf $(DIST)/sockt-linux-arm64.tar.gz   -C $(DIST) sockt-linux-arm64
	tar -czf $(DIST)/sockt-darwin-amd64.tar.gz  -C $(DIST) sockt-darwin-amd64
	tar -czf $(DIST)/sockt-darwin-arm64.tar.gz  -C $(DIST) sockt-darwin-arm64
	(cd $(DIST) && zip -q sockt-windows-amd64.zip sockt-windows-amd64.exe)
	(cd $(DIST) && sha256sum sockt-*.tar.gz sockt-*.zip sockt-linux-* sockt-darwin-* sockt-windows-*.exe > checksums.txt)
	@echo "Release artifacts written to $(DIST)/"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(DIST) $(BINARY)
