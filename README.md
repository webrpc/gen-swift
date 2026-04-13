# gen-swift

This repo contains the templates used by the `webrpc-gen` CLI to code-generate
webrpc Swift client code.

## Scope

`gen-swift` is client-only.

- It generates Swift client code and runtime helpers.
- It does not generate server handlers or server runtime code.
- Passing `-server` is rejected as an unsupported option.

## Usage

```sh
webrpc-gen -schema=example.ridl -target=swift -client -out=./ExampleClient.swift
```

or:

```sh
webrpc-gen -schema=example.ridl -target=github.com/webrpc/gen-swift@latest -client -out=./ExampleClient.swift
```

or:

```sh
webrpc-gen -schema=example.ridl -target=./local-templates-on-disk -client -out=./ExampleClient.swift
```

## Generated Surface

Generated output includes:

- schema constants
- WebRPC header/version metadata helpers
- DTOs and enums
- transport/error helpers
- service metadata helpers
- high-level async client methods

Low-level helpers remain visible:

- `ServiceAPI.basePath`
- `ServiceAPI.Method.path`
- `ServiceAPI.Method.urlPath`
- `ServiceAPI.Method.encodeRequest(...)`
- `ServiceAPI.Method.decodeResponse(...)`

## Tooling

This repo pins the published webrpc tool module in `tools/go.mod` using Go tool
dependencies.

Use the pinned tools with:

```sh
go -C tools tool webrpc-gen
go -C tools tool webrpc-test
```

## Testing

Use the fast generator/unit suite for normal development:

```sh
go test ./...
```

Run the shared Swift integration package separately when you want real generated
Swift compilation and runtime coverage:

```sh
./Tests/test.sh
```

That script regenerates `Tests/Sources/Generated/Generated.swift` from
`Tests/test.ridl` and runs `swift test` against a shared SwiftPM package.

The real external-schema integration remains available as an opt-in Go test:

```sh
WEBRPC_SWIFT_EXTERNAL_SCHEMA=1 go test ./...
```

## Options

| webrpc-gen option | Description | Default |
| --- | --- | --- |
| `-client` | generate client code | unset (`false`) |
| `-webrpcHeader` | send the standard `Webrpc` header on client requests | `true` |
