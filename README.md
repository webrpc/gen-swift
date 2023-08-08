# gen-swift

This repo contains the templates used by the `webrpc-gen` cli to code-generate
webrpc Swift client code.

This generator, from a webrpc schema/design file will code-generate:

1. Client -- a Swift client to speak to a webrpc server using the
provided schema. This client is compatible with any webrpc server language (ie. Go, nodejs, etc.).

## Dependencies

In order to support `any` type in webrpc, we use [AnyCodable](https://github.com/Flight-School/AnyCodable). 
This is a dependency of the generated code, so you must add it to your project.

## Usage

```
webrpc-gen -schema=example.ridl -target=swift -server -client -out=./example.gen.swift
```

or 

```
webrpc-gen -schema=example.ridl -target=github.com/webrpc/gen-swift@v0.11.2 -server -client -out=./example.get.swift
```

or

```
webrpc-gen -schema=example.ridl -target=./local-templates-on-disk -server -client -out=./example.gen.swift
```

As you can see, the `-target` supports default `swift`, any git URI, or a local folder :)

### Set custom template variables
Change any of the following values by passing `-option="Value"` CLI flag to `webrpc-gen`.

| webrpc-gen -option   | Description                | Default value              |
|----------------------|----------------------------|----------------------------|
| `-client`            | generate client code       | unset (`false`)            |

## LICENSE

[MIT LICENSE](./LICENSE)