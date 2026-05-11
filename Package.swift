// swift-tools-version: 6.1
//
// SwiftPM consumers depend on `https://github.com/NaughtBot/e2ee-payloads.git`
// directly. The committed sources under `swift/Sources/NaughtBotE2EEPayloads/`
// are emitted by `make generate-swift` (apple/swift-openapi-generator) and
// must not be hand-edited; they cover the full schema set defined under
// `openapi/`.
import PackageDescription

let package = Package(
    name: "NaughtBotE2EEPayloads",
    platforms: [
        .iOS("18.0"),
        .macOS("26.0"),
    ],
    products: [
        .library(
            name: "NaughtBotE2EEPayloads",
            targets: ["NaughtBotE2EEPayloads"]
        ),
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-openapi-runtime", from: "1.6.0"),
    ],
    targets: [
        .target(
            name: "NaughtBotE2EEPayloads",
            dependencies: [
                .product(name: "OpenAPIRuntime", package: "swift-openapi-runtime"),
            ],
            path: "swift/Sources/NaughtBotE2EEPayloads",
            exclude: [
                // Generator inputs copied alongside the generated sources by
                // `make generate-swift` so the generator can resolve them
                // relative to the output directory; SwiftPM should ignore
                // them when compiling the target.
                "openapi.yaml",
                "openapi-generator-config.yaml",
            ]
        ),
        .testTarget(
            name: "NaughtBotE2EEPayloadsTests",
            dependencies: ["NaughtBotE2EEPayloads"],
            path: "swift/Tests/NaughtBotE2EEPayloadsTests"
        ),
    ]
)
