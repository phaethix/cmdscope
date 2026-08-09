# Gate A placeholder targets.
#
# test wraps the Task 02 acceptance command (go test ./...) so `make test`
# stays useful; it does not implement any later-Task logic.
# check-schema is a pure placeholder until Task 36 implements real schema,
# example, and gold validation.
.PHONY: test check-schema

.DEFAULT_GOAL := test

test:
	go test ./...

check-schema:
	@echo "cmdscope: check-schema is not implemented until Task 36"
