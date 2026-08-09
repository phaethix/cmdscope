.PHONY: test check-schema

.DEFAULT_GOAL := test

# Placeholder targets for Gate A; real schema validation arrives in Task 36.
test:
	go test ./...

check-schema:
	@echo "cmdscope: check-schema is not implemented until Task 36"
