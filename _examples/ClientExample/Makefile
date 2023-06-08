all:
	@awk -F'[ :]' '!/^all:/ && /^([A-z_-]+):/ {print "make " $$1}' Makefile

bootstrap:
	rm -rf server/node_modules
	cd server && yarn

generate: generate-server generate-client

generate-server:
	webrpc-gen -schema=service.ridl -target=typescript -server -out=./server/server.gen.ts

generate-client:
	webrpc-gen -schema=service.ridl -target=../../ -client -out=./Sources/ClientExample/client.swift
	swift build -c release

run-server:
	yarn --cwd ./server start
	
run-client:
	./.build/release/ClientExample