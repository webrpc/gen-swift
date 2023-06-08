// swift-tools-version: 5.8
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "ClientExample",
    platforms: [.macOS(.v13), .iOS(.v15)],
    products: [
        .executable(name: "ClientExample", targets: ["ClientExample"]),

    ],
    dependencies: [.package(url: "https://github.com/Flight-School/AnyCodable", from: "0.6.7")],
    targets: [
        .executableTarget(
            name: "ClientExample",
            dependencies: ["AnyCodable"]),
    ]
)
