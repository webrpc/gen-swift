.PHONY: all generate build diff

all: generate diff build

generate:
	cd ClientExample && make generate

build:
	cd ClientExample && make build

diff:
	git diff --color --ignore-all-space --ignore-blank-lines --exit-code .
