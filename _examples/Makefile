.PHONY: all generate build diff test-streaming

all: generate diff build test-streaming

generate:
	cd ClientExample && make generate

build:
	cd ClientExample && make build

diff:
	git diff --color --ignore-all-space --ignore-blank-lines --exit-code .

test-streaming:
	@echo "verifying streaming schemas are rejected"
	@if go -C ../tools tool webrpc-gen -schema=../_examples/streaming.ridl -target=.. -client -out=/dev/null >/dev/null 2>&1; then \
		echo "FAIL: streaming schema was not rejected"; exit 1; \
	else \
		echo "OK: streaming rejected"; \
	fi
