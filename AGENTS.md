# Repository Guidelines — NaughtBot/e2ee-payloads

## Project Structure & Module Organization

`NaughtBot/e2ee-payloads` is the canonical source-of-truth repo for the
NaughtBot mailbox envelope and its per-`type` payload schemas. Backend
services never see these structures — they live inside HPKE-decrypted bytes
and are decoded only by the producer and the approver. That is why they live
here and not in `NaughtBot/core`.

- `openapi/` — hand-edited OpenAPI 3.0.3 sources, the only authored artefacts.
  - `openapi/envelope.yaml` — `MailboxEnvelopeV1` plus the `MailboxEnvelopeType`
    registry enum.
  - `openapi/payloads/<surface>.yaml` — per-surface payload schemas
    (`ssh.yaml`, `gpg.yaml`, `age.yaml`, `pkcs11.yaml`, `enroll.yaml` once
    WS1.2 lands).
- `go/` — generated Go package, module `github.com/naughtbot/e2ee-payloads/go`.
- `swift/Sources/NaughtBotE2EEPayloads/` — generated Swift module surfaced
  through the root `Package.swift` so SwiftPM consumers can depend on this
  repository URL directly.
- `typescript/src/` — generated TypeScript module published to npm as
  `@naughtbot/e2ee-payloads`.

Treat `go/`, `swift/Sources/NaughtBotE2EEPayloads/`, and `typescript/src/` as
generated output — never hand-edit. They are committed so downstream consumers
do not need to run the generator stack to install the package.

## Dev Guidelines

OpenAPI is the source of truth. Every schema additive change starts in
`openapi/` and propagates to bindings via `make generate`. Use
`additionalProperties: false` on every payload schema; this mirrors the
`MailboxCaptcha*` precedent in `NaughtBot/core` and matches the wire
contract.

No backwards-compatibility shims unless explicitly requested. Per the
workspace-wide rule, schema renames are full renames (e.g. the legacy
`custom` and `ecdh_derive` types from the previous monorepo are renamed
outright to `pkcs11_sign` and `pkcs11_derive` here).

### Sibling repo contract

- `NaughtBot/core` is a single-module Go monorepo at
  `github.com/naughtbot/core` (sub-packages `auth`, `mailbox`, `blob`, `push`,
  `verify`). Generated public clients are published from `NaughtBot/api` —
  this repo is **not** a generated-client home for any service surface; it
  carries only mailbox envelope plaintext schemas.
- `NaughtBot/api` is the public generated-client home for core service
  surfaces. It currently also exports `MailboxEnvelopeV1` from
  `core/openapi/shared.yaml`; once the deferred migration runs, that
  authoritative copy moves into this repo and `NaughtBot/api` consumes it
  transitively.

## Build, Test, and Development Commands

The Makefile drives the per-language tooling.

- `make openapi-lint` — Redocly lint over every YAML under `openapi/`.
- `make generate` — regenerate Go, Swift, and TypeScript bindings.
- `make generate-check` — regenerate and fail if outputs drift; CI gate.
- `make build` — build every language binding (Go, Swift, TypeScript).
- `make test` — run unit tests for every language binding.

Tool versions:

- Go 1.26.x with `go tool oapi-codegen` (declared in `go/go.mod` via the
  `tool` directive). No additional install step.
- Swift 6.x with `swift-openapi-generator` 1.10.2 (cloned and built into
  `.swift-openapi-generator/` on first `make generate-swift`). Cached
  across runs; bump the pin by setting `SWIFT_OPENAPI_GENERATOR_GIT_TAG`.
- Node 24 with `openapi-typescript` and `typescript` (declared as
  `devDependencies` in `typescript/package.json`). Redocly is invoked via
  `npx -y -p @redocly/cli@latest`.

The Makefile bundles the multi-file OpenAPI sources via
`make bundle` (Redocly `redocly bundle`) into a single self-contained
`openapi/bundled/payloads.yaml`, which feeds every per-language
generator. Adding a new payload surface requires re-exporting its schemas
from `openapi/payloads.yaml`.

## Coding Style & Naming Conventions

- OpenAPI: 2-space indentation; schema names in PascalCase with the
  `Mailbox<Surface><Op><Direction>PayloadV1` convention (e.g.
  `MailboxSshAuthRequestPayloadV1`).
- Go: `gofmt`; generated package name `payloads`.
- Swift: SwiftFormat defaults; generated module `NaughtBotE2EEPayloads`.
- TypeScript: 2-space indentation; `import type` for generated types.

## Release Workflow

**Releases are tag-driven only.** Do not run release scripts manually.

1. Land schema/binding changes on `main` via PRs.
2. Push a `v<MAJOR>.<MINOR>.<PATCH>` tag at the release commit.
3. `release.yml` regenerates bindings, asserts a clean diff, creates the
   per-language subdirectory tags `go/v<ver>`, `swift/v<ver>`,
   `typescript/v<ver>`, opens a GitHub release, and publishes the npm
   package to `@naughtbot/e2ee-payloads`.

Go consumers import from `github.com/naughtbot/e2ee-payloads/go` and pin via
`go get github.com/naughtbot/e2ee-payloads/go@v<ver>`. SwiftPM consumers
depend on `https://github.com/NaughtBot/e2ee-payloads.git` and pin via
`from: "<ver>"`. npm consumers install `@naughtbot/e2ee-payloads@<ver>`.

## Testing Guidelines

Bug fixes must include a regression test. Generator regressions belong as
unit tests near the binding under test (Go `*_test.go`, Swift `swift-testing`
or XCTest, TypeScript `*.test.ts`). Schema-shape changes must include a
JSON round-trip test in at least one binding.

## Commit & Pull Request Guidelines

Conventional commits, scoped to the affected surface:

- `feat(openapi): add age-unwrap response schema`
- `fix(go): regenerate after envelope enum bump`
- `chore(swift): align generator config with NaughtBot/api`

Each PR must state which generator(s) ran and list validation commands.
Generated diffs land in the same commit as the schema change that produced
them; CI's `make generate-check` rejects schema-only commits.
