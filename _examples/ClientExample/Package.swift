// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "WebRPCClientExample",
    platforms: [
        .macOS(.v12),
        .iOS(.v15),
    ],
    products: [
        .library(name: "ClientExample", targets: ["ClientExample"]),
    ],
    targets: [
        .target(name: "ClientExample"),
    ]
)
