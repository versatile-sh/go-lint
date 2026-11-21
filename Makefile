.PHONY: check test images semgrep

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go run ./cmd/vlint golangci -check
	go run ./cmd/vlint ./...
	golangci-lint run ./...
	$(MAKE) --no-print-directory semgrep

test:
	go test -race -count=1 ./...

images:
	cd docs/assets/src && npm ci && node build-banners.mjs

SEMGREP_RULES := https://raw.githubusercontent.com/versatile-sh/semgrep-rules/v0.1.2/rules/versatile.yml

semgrep:
	semgrep scan --metrics=off --disable-version-check --error --severity WARNING --severity ERROR \
		--config p/default --config p/secrets --config p/golang --config $(SEMGREP_RULES) \
		--baseline-commit "$$(git merge-base HEAD origin/main)"
