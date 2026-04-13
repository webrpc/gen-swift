# gen-swift Plan

This document is the initial implementation plan for `gen-swift`, based on a review of `../gen-kotlin`.

## 1. What Must Carry Over From `gen-kotlin`

The Kotlin generator is intentionally small, but it already defines the contract the Swift target should preserve:

- A single generated client file contains:
  - schema constants
  - DTOs and enums
  - runtime transport/error helpers
  - service metadata helpers
  - high-level client methods
- Generated clients are transport-injected, not tightly coupled to one HTTP stack.
- Generated output exposes low-level helpers:
  - `ServiceAPI.basePath`
  - `ServiceAPI.Method.path`
  - `ServiceAPI.Method.urlPath`
  - `encodeRequest(...)`
  - `decodeResponse(...)`
- Succinct methods do not generate wrapper `Request` / `Response` types when the wire shape is a single value.
- Service/type name collisions are handled explicitly.
- Response decoding is tolerant of unknown JSON fields.
- Error decoding is standardized around the webrpc error payload.

These behaviors should remain true in Swift unless there is a language-specific reason to improve them.

## 2. Swift V1 Direction

`gen-swift` should target a pure Swift client library, not an app framework.

Scope boundary:

- generate client code only
- do not generate server handlers, server protocols, or server runtime helpers
- do not mirror `gen-typescript`'s combined client/server scope
- stay aligned with `gen-kotlin` in product scope: one generator whose job is client generation

Use:

- Swift 5.9 as the minimum generated language target
- iOS 15+ as the default Apple deployment target for the provided `URLSession` transport
- local development/testing on Swift 6.3
- `async` / `await` for all client calls
- `Foundation` for JSON, `Data`, `URL`, and `URLSession`
- `FoundationNetworking` behind conditional imports for Linux compatibility
- `Codable` as the primary serialization mechanism
- `Sendable` on generated public value types and runtime abstractions where feasible
- Swift Package Manager in generated test fixtures

Do not use in v1:

- SwiftUI, Combine, UIKit/AppKit, or Vapor-specific APIs
- third-party JSON libraries
- third-party HTTP clients
- macros, result builders, or actor-heavy abstractions where simple value types and protocols are enough

This generator should feel like a small, portable networking library that happens to be code-generated.

Important distinction:

- the Swift language/toolchain baseline is Swift 5.9
- the deployment target baseline is separate from the language version
- for v1, the generated default transport should assume iOS 15+ because it can use `URLSession`'s native async APIs directly
- lower Apple deployment targets can be considered later if there is a real need to bridge completion-handler APIs behind the same generated `async` client surface

## 3. Core Technical Decisions

### Serialization

Use `Codable`.

Reasoning:

- It is built into Swift and requires no dependency story.
- `JSONDecoder` already ignores unknown keys by default, which matches the Kotlin runtime behavior.
- `JSONEncoder` / `JSONDecoder` work cleanly with `Data`, nested structs, enums with custom decoding, and explicit `CodingKeys`.
- It keeps the generated output suitable for iOS, macOS, and server-side Swift with the same runtime.

Planned shape:

- generated structs: `public struct Foo: Codable, Sendable`
- generated aliases: `public typealias Foo = ...`
- generated enums: custom `Codable` implementation for forward compatibility
- runtime JSON helpers:
  - `WebRPCJSONValue`
  - `WebRPCNull`

### `any` and `null`

Do not map both to the same Swift type.

- `any` should map to `WebRPCJSONValue`
  - an enum capable of representing object / array / string / number / bool / null
- `null` should map to `WebRPCNull`
  - a small marker type that only encodes/decodes JSON `null`

Reasoning:

- Swift can model these distinctly without extra dependencies.
- separating them is stricter and clearer than the Kotlin `JsonElement` approach
- optional `null` fields can still be represented as `WebRPCNull?`

### Timestamps

Map `timestamp` to `String` in v1.

Reasoning:

- this matches current Kotlin behavior
- the wire format may vary across servers
- generating `Date` would force an opinionated decoding strategy that can break valid schemas

If needed later, add an opt-in generator option for `Date`.

### Big Integers

If RIDL `bigint` is present, map it to `String` in v1.

Reasoning:

- it preserves full wire precision
- it avoids introducing a `BigInt` dependency
- it matches how other targets typically avoid JSON numeric precision loss

### Bytes

Map:

- `byte` -> `UInt8`
- `[]byte` / `list<byte>` -> `Data`

Reasoning:

- `Data` is the natural Swift type for opaque bytes
- `Data: Codable` already round-trips as base64 in JSON
- it is materially better than exposing raw `String` for byte arrays in Swift

This is an intentional improvement over the Kotlin target's `String` mapping for `list<byte>`.

### Integers

Do not use platform-sized `Int` / `UInt` for RIDL `int` / `uint`.

Map:

- `int` -> `Int32`
- `uint` -> `UInt32`
- `int8` -> `Int8`
- `uint8` -> `UInt8`
- `int16` -> `Int16`
- `uint16` -> `UInt16`
- `int32` -> `Int32`
- `uint32` -> `UInt32`
- `int64` -> `Int64`
- `uint64` -> `UInt64`

Reasoning:

- Kotlin `Int` / `UInt` are fixed-width 32-bit types
- Swift `Int` / `UInt` are platform-sized
- fixed-width mapping is safer and more predictable for wire compatibility

### Enums

Generate custom enums with unknown-value preservation.

Preferred shape:

```swift
public enum Kind: Sendable, Equatable, Codable {
    case user
    case admin
    case unknown(String)
}
```

with a generated `wireValue` property and custom `init(from:)` / `encode(to:)`.

Reasoning:

- this is more forward-compatible than a synthetic `UNKNOWN_DEFAULT`
- unknown server values are preserved instead of collapsed
- the API is still simple for consumers

### Maps

Map support needs explicit rules in Swift.

- `map<string, T>` -> `[String: T]`
- `map<Enum, T>` -> generated or shared helper wrapper with custom `Codable`
- any other map key type should be rejected with a clear generator error

Reasoning:

- JSON object keys are strings
- Swift `Dictionary` with non-string keys does not naturally encode to the JSON object shape we need
- enum-keyed maps are still worth supporting because they are a reasonable schema feature

## 4. Runtime Design

### Transport

Keep a transport protocol.

Planned API:

```swift
public protocol WebRPCTransport: Sendable {
    func post(
        baseURL: String,
        path: String,
        body: Data,
        headers: [String: String]
    ) async throws -> WebRPCHTTPResponse
}
```

Provide a default runtime transport:

- `URLSessionWebRPCTransport`

Reasoning:

- unlike Kotlin's OkHttp, `URLSession` is already a platform dependency we can rely on
- there is no need for a separate `okhttpTransport=true`-style option in v1
- users can still inject custom transports for testing, auth, retries, or special environments

### Request/Response Encoding

Use `Data` internally for HTTP bodies, not `String`.

Reasoning:

- this is the native type expected by `URLSession`
- it avoids repeated UTF-8 string conversions
- JSON helpers can still expose convenience string methods if useful for tests

Proposed helper surface:

- `encodeRequest(...) -> Data`
- `decodeResponse(_ data: Data, decoder: JSONDecoder = WebRPCJSON.makeDecoder()) -> Response`
- optional convenience `encodeRequestString(...) -> String` only if tests/examples benefit from it

Keep the helper names aligned with the Kotlin contract even though the Swift implementation should prefer `Data` internally.

### Errors

Mirror the Kotlin runtime structure:

- `WebRPCErrorKind`
- `WebRPCError: Error, Sendable`
- `WebRPCTransportError`
- `decodeWebRPCError(statusCode:data:decoder:)`

Keep unknown error decoding resilient:

- if the error payload is malformed, surface a generic bad-response error with the raw body attached

### Concurrency

Use `async` / `await` throughout.

Planned stance:

- client methods are `async throws`
- generated DTOs are value types and `Sendable`
- avoid storing non-`Sendable` Foundation reference types on the public client
- use `@unchecked Sendable` only where Foundation types force it and only in isolated runtime helpers
- do not introduce actors unless a real mutable shared-state problem exists

## 5. Public API Shape

Swift should follow Swift naming conventions, not Kotlin naming conventions.

Examples:

- service metadata namespace: `WaasWalletAPI`
- generated client: `WaasWalletClient`
- method namespace: `GetUser`
- client method: `getUser(_ request: ...) async throws -> ...`

For each service:

- emit a top-level namespace type, likely `enum <Service>API`
- emit a client type, likely `struct` or `final class`
- keep low-level method helpers nested under the namespace

Preferred namespace pattern:

```swift
public enum WaasWalletAPI {
    public static let basePath = "/rpc/Wallet"

    public enum GetUser {
        public static let path = "/GetUser"
        public static let urlPath = "/rpc/Wallet/GetUser"
    }
}
```

### Client Type

Use `public struct` for the client unless transport storage forces reference semantics.

Likely shape:

```swift
public struct WaasWalletClient: Sendable {
    public let baseURL: String
    public let transport: any WebRPCTransport
    public let headers: @Sendable () -> [String: String]
}
```

The client should not store `JSONEncoder` / `JSONDecoder` instances directly.

Instead:

- runtime helpers should create fresh encoders/decoders via `WebRPCJSON.makeEncoder()` / `WebRPCJSON.makeDecoder()`
- if customization is needed later, prefer immutable config values or `@Sendable` factory closures over storing mutable Foundation encoder/decoder instances on the client

If existential `Sendable` ergonomics are awkward on Swift 5.9, fall back to:

- `public final class WaasWalletClient`

without changing the external method surface.

## 6. Type Generation Rules

### Structs

Generate:

- `public struct`
- `Codable`
- `Sendable`
- explicit `CodingKeys`
- lowerCamelCase property names
- original JSON names preserved in `CodingKeys`

Do not rely on `JSONEncoder.keyEncodingStrategy`.

Reasoning:

- explicit keys are deterministic
- JSON metadata like `json:"USERNAME,omitempty"` must stay under generator control

### Aliases

Use `typealias` for aliases unless the alias requires custom coding behavior.

Examples:

- `type Username = String`
- `type Blob = Data`

If a future schema feature needs custom behavior, promote that alias to a wrapper struct instead of overloading `typealias`.

### Reserved Words and Name Collisions

Add explicit sanitization for:

- Swift reserved keywords
- field names that collide after `lowerCamelCase`
- service/type/client/api symbol collisions

The Kotlin repo already proves collision handling is necessary, so this must be part of v1, not a later cleanup.

## 7. Repository Shape For `gen-swift`

Mirror the overall repo style of `gen-kotlin`:

- `go.mod`
- `embed.go`
- `README.md`
- `generator_test.go`
- root `*.go.tmpl` template files
- `_examples/`
- `tools/go.mod`

Suggested initial templates:

- `main.go.tmpl`
- `imports.go.tmpl`
- `types.go.tmpl`
- `type.go.tmpl`
- `runtime.go.tmpl`
- `client.go.tmpl`
- `methodInputs.go.tmpl`
- `methodOutputs.go.tmpl`
- `codingKey.go.tmpl`
- `fieldName.go.tmpl`

Unlike Kotlin, split the runtime into its own template from day one.

Do not add server-side templates such as `server.go.tmpl` or `serverHelpers.go.tmpl`.

## 8. Test Strategy

Replicate the Kotlin generator's test philosophy, but using SwiftPM instead of Gradle.

### Generator Unit / Snapshot Tests

Add Go tests that generate Swift output and assert for:

- succinct method generation
- helper API presence
- service/schema-aware naming
- service/type collision handling
- reserved keyword field handling
- `Data` mapping for `list<byte>`
- enum unknown-value preservation code
- `map<string, T>` and enum-keyed map generation

### Compile Tests

Create temporary Swift packages in `generator_test.go` and run:

```sh
swift test
```

against generated code.

### Runtime Tests

Mirror the Kotlin runtime tests:

- encode request and inspect JSON payload
- decode successful responses
- decode webrpc error payloads
- verify helper paths
- run a local `webrpc-test` server for end-to-end transport validation

### Compatibility Target

Test at least:

- current local Swift toolchain
- one strict Swift 5.9 compile/test configuration in CI

Swift 5.9 support is a release requirement for v1, not a best-effort target.

## 9. README Scope

The eventual `README.md` should document:

- what the generator produces
- required Swift version
- zero third-party dependency story
- how the transport abstraction works
- how to use the default `URLSessionWebRPCTransport`
- current limitations
- generator options
- how to run tests with the pinned `tools` module

Generator options should stay minimal and client-focused.

In particular:

- support `-client`
- document Swift-specific client options only if they are real and tested
- reject `-server` as an unsupported target option instead of accepting it as a no-op

## 10. Initial Implementation Sequence

1. Bootstrap the repo skeleton to match other `gen-*` repositories.
2. Implement type mapping and naming helpers first.
3. Implement runtime helpers next:
   - JSON helpers
   - transport protocol
   - default `URLSession` transport
   - error decoding
4. Implement service metadata and client emission.
5. Add compile/runtime tests for the same cases already covered in `gen-kotlin`.
6. Add an example client package.
7. Write README last, once the exact API surface is settled.

## 11. Explicit V1 Non-Goals

Do not optimize for these in the first cut:

- SwiftUI-specific wrappers
- Combine publishers
- macros
- generated server handlers
- server middleware/runtime abstractions
- generated server code
- plugin-based HTTP middlewares
- automatic retry/backoff logic
- date parsing beyond raw string transport
- non-JSON codecs

The first version should be small, predictable, and fully tested.

## 12. Bottom Line

The Swift target should intentionally preserve the Kotlin generator's good parts:

- low ceremony
- visible runtime helpers
- transport injection
- schema-aware service metadata
- strong compile/runtime coverage

But it should also make a few Swift-native improvements immediately:

- `Codable` instead of a custom serialization dependency
- `Data` for byte arrays
- fixed-width integer mapping
- `WebRPCNull` and `WebRPCJSONValue` as distinct types
- unknown enum value preservation
- explicit support plan for enum-keyed maps
