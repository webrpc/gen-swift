.PHONY: all generate build

all: generate build

generate:
	go -C ../../tools tool webrpc-gen -schema=../_examples/ClientExample/service.ridl -target=.. -client -out=../_examples/ClientExample/Sources/ClientExample/ExampleServiceClient.swift

build:
	swift build --package-path .
