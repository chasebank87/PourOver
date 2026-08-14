.PHONY: test build install vet fmt-check clean macos-docs

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

test:
	go test ./...

build:
	go build -o pourover ./cmd/pourover

install: build
	mkdir -p "$(BINDIR)"
	cp pourover "$(BINDIR)/pourover"
	chmod +x "$(BINDIR)/pourover"
	@echo "Installed to $(BINDIR)/pourover"

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)

macos-docs:
	go run ./cmd/genmacosdocs docs/macos-defaults.md

clean:
	rm -f pourover
