.PHONY: all generate diff

all: generate diff

generate:
	cd ClientExample && make generate

diff:
	git diff --color --ignore-all-space --ignore-blank-lines --exit-code .