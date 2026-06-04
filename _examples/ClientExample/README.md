# WebRPC Swift Client Example

This example shows generating a Swift client from [service.ridl](./service.ridl)
using the local generator templates.

## Usage

Regenerate the tracked client:

```sh
make generate
```

Build the Swift package:

```sh
make build
```

The schema includes representative cases for generated enum fallback naming,
method/type name collisions, and fields excluded with `json = -`.
