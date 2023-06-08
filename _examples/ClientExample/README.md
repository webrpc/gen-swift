webrpc node-ts
==============

* Server: Nodejs (TypeScript)
* Client: CLI (Swift)

example of generating a webrpc client from [service.ridl](./service.ridl) schema.

## Usage

1. Install nodejs, yarn, swift and webrpc-gen
1. $ `make bootstrap` -- runs yarn on ./server
1. $ `make generate` -- generates both server and client code
1. $ `make run-server`
1. $ `make run-client`

Or you can just open the Package.swift and run the client from Xcode.