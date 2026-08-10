# Gate A placeholder targets.
#
# test wraps the acceptance command (go test ./...) so `make test`
# stays useful; it does not implement any real analyzer logic.
# check-schema is a pure placeholder until real schema, example, and gold
# validation is implemented.
.PHONY: test check-schema

.DEFAULT_GOAL := test

test:
	go test ./...

check-schema:
	@echo "cmdscope: check-schema is not implemented yet"
