.PHONY: test build vet fmt-check clean

test:
	go test ./...

build:
	go build -o pourover ./cmd/pourover

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)

clean:
	rm -f pourover
