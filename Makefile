.PHONY: test build install vet fmt-check clean

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

clean:
	rm -f pourover
