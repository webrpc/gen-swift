// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "gen-swift-integration",
    platforms: [
        .macOS(.v12),
        .iOS(.v15),
    ],
    products: [
        .library(name: "Generated", targets: ["Generated"]),
    ],
    targets: [
        .target(name: "Generated"),
        .testTarget(name: "GeneratedTests", dependencies: ["Generated"]),
    ]
)
