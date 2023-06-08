#!/bin/bash
set -e

webrpc-test -version
webrpc-test -print-schema > ./test.ridl
webrpc-gen -schema=./test.ridl -target=../ -client -out=./Sources/gen-swift/client.swift

webrpc-test -server -port=9988 -timeout=5s &

# Wait until http://localhost:9988 is available, up to 10s.
for (( i=0; i<100; i++ )); do nc -z localhost 9988 && break || sleep 0.1; done

swift test
