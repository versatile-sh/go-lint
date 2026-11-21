.PHONY: check test images

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go run ./cmd/vlint golangci -check
	go run ./cmd/vlint ./...
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

images:
	cd docs/assets/src && npm ci && node build-banners.mjs
