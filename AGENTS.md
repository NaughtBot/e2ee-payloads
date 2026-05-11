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
  - `openapi/payloads/common.yaml` — shared building blocks (approval
    challenge / proof chain, attestation metadata, signing error codes,
    optional display + source-info metadata).
  - `openapi/payloads/<surface>.yaml` — per-surface payload schemas:
    `ssh.yaml`, `gpg.yaml`, `age.yaml`, `pkcs11.yaml`, `enroll.yaml`.
  - `openapi/payloads.yaml` — bundle root that re-exports every
    component schema; `make bundle` flattens it into
    `openapi/bundled/payloads.yaml` for the generators.
  - `openapi/redocly.yaml` — Redocly lint config.
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

## Generator caveats

Two well-known oapi-codegen behaviours that the Go binding does not
paper over. Document them for callers and exercise them in the WS1.5
smoke tests:

- **`oneOf` `As<Branch>` is permissive.** Calling
  `MailboxXxxResponsePayloadV1.AsMailboxXxxResponseSuccessV1()` will
  succeed even on a failure-shaped JSON because `encoding/json` ignores
  unknown fields and does not enforce missing required fields. Routing
  call-sites SHOULD inspect the union with `Discriminator()` (when a
  discriminator field is present, e.g. `MailboxEnrollResponsePayloadV1`)
  or check that the success-only field is non-zero before treating the
  result as a success.
- **`additionalProperties: false` is not enforced by `encoding/json`.**
  Every payload schema sets `additionalProperties: false` and the Swift
  / TypeScript bindings honour it. Go callers MUST opt into strict
  decoding with `dec := json.NewDecoder(r); dec.DisallowUnknownFields()`
  if the service-level invariant is "reject envelopes with unknown
  extra fields"; the generated structs do not enable
  `DisallowUnknownFields` for them.

The envelope `payload` field is generated as `json.RawMessage` in Go via
the `x-go-type` extension so per-`type` handlers can decode the original
byte stream without losing precision (the default
`map[string]interface{}` representation widens int64 values to float64).

The envelope `type` field is modelled as a plain `string` (not the
`MailboxEnvelopeType` enum) so receivers can decode forward-compatible
envelopes whose `type` is not yet in the published registry, then
log-and-drop. Validate against the `MailboxEnvelopeType` enum at
runtime, not at the codec layer.

## Coding Style & Naming Conventions

- OpenAPI: 2-space indentation; schema names in PascalCase with the
  `Mailbox<Surface><Op><Direction>PayloadV1` convention (e.g.
  `MailboxSshAuthRequestPayloadV1`).
- Go: `gofmt`; generated package name `payloads`.
- Swift: SwiftFormat defaults; generated module `NaughtBotE2EEPayloads`.
- TypeScript: 2-space indentation; `import type` for generated types.

## Release Workflow

**Releases are tag-driven only.** Do not run any release step manually
(no local `npm publish`, no manual subdirectory tagging, no manual
GitHub release creation). Pushing the tag is the entire release surface.

1. Land schema/binding changes on `main` via PRs.
2. Bump `typescript/package.json` `version` to match the tag you are
   about to push, in the same commit that lands the changes (or in a
   bump-only PR right before tagging).
3. Push a `v<MAJOR>.<MINOR>.<PATCH>` tag at the release commit:
   `git tag v0.1.0 && git push origin v0.1.0`.
4. `.github/workflows/release.yml` runs:
   - drift gate (regenerate every binding, assert no diff at the tag
     commit) on Linux + macOS;
   - per-language subdirectory tags `go/v<ver>`, `swift/v<ver>`,
     `typescript/v<ver>` pushed to the same commit;
   - GitHub release for the top-level `v<ver>` with auto-generated
     notes;
   - npm publish of `@naughtbot/e2ee-payloads@<ver>` via Trusted
     Publishing (OIDC). The npm publish job is gated on the
     `NPM_TRUSTED_PUBLISHING_ENABLED` org variable so the workflow can
     land before the npm side of the integration is fully provisioned.

Go consumers import from `github.com/naughtbot/e2ee-payloads/go` and
pin via `go get github.com/naughtbot/e2ee-payloads/go@v<ver>`. SwiftPM
consumers depend on `https://github.com/NaughtBot/e2ee-payloads.git`
and pin via `from: "<ver>"` (against the top-level `v<ver>` semver
tag — SwiftPM does not resolve against the `swift/v<ver>` subdir
marker). npm consumers install `@naughtbot/e2ee-payloads@<ver>`.

## Known debt

The five existing v1 envelope types — `link_request`, `link_approval`,
`link_rejection`, `captcha_request`, `captcha_response` — currently
live in `NaughtBot/core` `openapi/shared.yaml`. Migrating them out of
core into this repo is a coordinated follow-up that touches `mobile/`,
`captcha/`, and `core/` simultaneously and is tracked separately under
the master plan
[`NaughtBot/workspace#3`](https://github.com/NaughtBot/workspace/issues/3).
For the v0.1.x series this repo carries the eight new signing /
decryption / derive / enroll envelope types only; the existing five
identifiers appear in the `MailboxEnvelopeType` registry enum in
`openapi/envelope.yaml` so receivers can validate against the canonical
registry, but their payload schemas still live in core. Do not add
hand-written mirror schemas for them here — wait for the migration
follow-up.

## References

- Master tracker: [`NaughtBot/workspace#3`](https://github.com/NaughtBot/workspace/issues/3).
- Detailed bootstrap plan:
  [`workspace/plans/2026-05-11-0208Z-e2ee-payloads-bootstrap.md`](https://github.com/NaughtBot/workspace/blob/main/plans/2026-05-11-0208Z-e2ee-payloads-bootstrap.md).
- Cross-repo coordinator plan:
  [`workspace/plans/2026-05-11-0208Z-signing-types-cli-extraction.md`](https://github.com/NaughtBot/workspace/blob/main/plans/2026-05-11-0208Z-signing-types-cli-extraction.md).

## Testing Guidelines

Bug fixes must include a regression test (workspace rule). Generator
regressions belong as unit tests near the binding under test (Go
`*_test.go`, Swift `swift-testing`, TypeScript `*.test.ts`). Schema-shape
changes must include a JSON round-trip test in at least one binding.

The smoke tests under `go/payloads_test.go`,
`swift/Tests/NaughtBotE2EEPayloadsTests/PayloadsTests.swift`, and
`typescript/src/index.test.ts` cover one schema per surface plus the
documented generator caveats; expand them when adding a new payload
surface.

## Commit & Pull Request Guidelines

Conventional commits, scoped to the affected surface:

- `feat(openapi): add age-unwrap response schema`
- `fix(go): regenerate after envelope enum bump`
- `chore(swift): align generator config with NaughtBot/api`

Each PR must state which generator(s) ran and list validation commands.
Generated diffs land in the same commit as the schema change that produced
them; CI's `make generate-check` rejects schema-only commits.
