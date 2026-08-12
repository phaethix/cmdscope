# runmark build and quality targets.
#
# Primary targets:
#   make          – default: run all tests and quality checks
#   make test     – run the full test suite
#   make quick    – fast feedback: build + vet (no tests)
#   make fmt      – format code and check consistency
#   make lint     – run all static checks (vet + fmt-diff)
#   make ci       – full pipeline: fmt, vet, test (what CI should run)
#   make schema   – validate the schema, examples, and gold corpus

.PHONY: test quick fmt lint ci schema help

.DEFAULT_GOAL := all

all: lint test
	@echo "✓ all checks passed"

test:
	go test ./... -count=1

quick:
	go build ./...
	go vet ./...

fmt:
	@echo "goimports..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -l -w $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/); \
		gofmt -l .; \
	else \
		gofmt -l -w .; \
	fi

lint:
	go vet ./...
	@fmtout=$$(gofmt -l .); \
	if [ -n "$$fmtout" ]; then \
		echo "gofmt required:"; echo "$$fmtout"; exit 1; \
	fi
	@echo "✓ lint passed"

ci: lint test
	@echo "✓ ci passed"

schema:
	@echo "runmark: schema check is not yet implemented"
	@echo "  Run: go test ./internal/ir -run Schema"
	@echo "  See CONTRIBUTING.md § Changing the JSON contract for details"

help:
	@echo "runmark make targets:"
	@echo "  make / make all   – lint + test"
	@echo "  make test         – go test ./..."
	@echo "  make quick        – build + vet (fast)"
	@echo "  make fmt          – format code (goimports or gofmt)"
	@echo "  make lint         – static checks (vet + gofmt diff)"
	@echo "  make ci           – full CI pipeline"
	@echo "  make schema       – schema validation placeholder"
	@echo "  make help         – this message"
