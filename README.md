# NaughtBot/e2ee-payloads

Source-of-truth OpenAPI schemas for the NaughtBot mailbox envelope and
its per-`type` payloads, published as Swift / Go / TypeScript packages.

[![ci](https://github.com/NaughtBot/e2ee-payloads/actions/workflows/ci.yml/badge.svg)](https://github.com/NaughtBot/e2ee-payloads/actions/workflows/ci.yml)
[![release](https://github.com/NaughtBot/e2ee-payloads/actions/workflows/release.yml/badge.svg)](https://github.com/NaughtBot/e2ee-payloads/actions/workflows/release.yml)

## What lives here

The mailbox envelope and the per-`type` payload schemas are HPKE
plaintext — the backend services (`core/auth`, `core/mailbox`,
`core/blob`, `core/push`) never see them. They therefore do not belong
in `NaughtBot/core`. This repo is the canonical home for:

- `MailboxEnvelopeV1` — the typed envelope wrapper, with the
  `MailboxEnvelopeType` registry enum listing every supported `type`
  discriminator.
- The per-surface payload schemas under `openapi/payloads/`:
  - `ssh.yaml` — `MailboxSshAuthRequest/Response*` and
    `MailboxSshSignRequest/Response*`.
  - `gpg.yaml` — `MailboxGpgSignRequest/Response*`,
    `MailboxGpgDecryptRequest/Response*`.
  - `age.yaml` — `MailboxAgeUnwrapRequest/Response*`.
  - `pkcs11.yaml` — `MailboxPkcs11SignRequest/Response*`,
    `MailboxPkcs11DeriveRequest/Response*`.
  - `enroll.yaml` — `MailboxEnrollRequest/Response*` (discriminated
    on `status` between `Approved` and `Rejected` branches).
- Generator-emitted bindings, committed so consumers do not need the
  generator stack to install:
  - `go/` — Go module `github.com/naughtbot/e2ee-payloads/go`
    (oapi-codegen).
  - `swift/Sources/NaughtBotE2EEPayloads/` — Swift module surfaced
    through the root `Package.swift` (swift-openapi-generator).
  - `typescript/src/` — TypeScript types published to npm as
    `@naughtbot/e2ee-payloads` (openapi-typescript).

## Registry contract

Every new envelope `type` is an explicit additive change touching:

1. `openapi/envelope.yaml` — append the new identifier to the
   `MailboxEnvelopeType` enum.
2. `openapi/payloads/<surface>.yaml` — author the request / response
   payload schemas with `additionalProperties: false`.
3. `openapi/payloads.yaml` — re-export the new component schemas from
   the bundle root.
4. `make generate` — re-run every binding; CI's `make generate-check`
   gate enforces that the committed bindings match the OpenAPI sources.

The envelope `type` field itself is modelled as a plain `string` (not
the `MailboxEnvelopeType` enum) so receivers can decode envelopes whose
`type` is unknown to the published registry and log+drop per the v1
contract instead of failing at the codec layer.

## Installing

### Go

```sh
go get github.com/naughtbot/e2ee-payloads/go@v0.1.0
```

```go
import payloads "github.com/naughtbot/e2ee-payloads/go"
```

The Go module lives under the `/go/` subdirectory; the release workflow
pushes a matching `go/v<ver>` tag for every release.

### Swift

```swift
.package(url: "https://github.com/NaughtBot/e2ee-payloads.git", from: "0.1.0")
```

```swift
import NaughtBotE2EEPayloads
```

The root `Package.swift` exposes the `NaughtBotE2EEPayloads` target.
SwiftPM consumes the top-level `v<ver>` semver tag (the `swift/v<ver>`
subdir tag pushed by the release workflow is a parallel marker for
release audits — SwiftPM's `from:` / `exact:` requirements only
resolve against semver-shaped tags).

### TypeScript / JavaScript

```sh
npm install @naughtbot/e2ee-payloads
```

```ts
import type {
  MailboxEnvelopeV1,
  MailboxSshSignRequestPayloadV1,
} from "@naughtbot/e2ee-payloads";
```

The package ships compiled `.js` + `.d.ts` from `dist/` and the original
`.ts` sources under `src/` so type-only consumers can import without a
runtime.

## Development

See [`AGENTS.md`](AGENTS.md) for the full repo conventions, generator
caveats, and PR + commit guidelines. Quick reference:

- `make openapi-lint` — Redocly lint.
- `make bundle` — bundle `openapi/payloads.yaml` to
  `openapi/bundled/payloads.yaml`.
- `make generate` — regenerate every binding.
- `make generate-check` — CI gate; fails if generated outputs drift.
- `make build` / `make test` — per-language build / test.

## Release process

**Releases are tag-driven only.** Pushing a `v<MAJOR>.<MINOR>.<PATCH>`
tag at a clean main commit triggers
[`.github/workflows/release.yml`](.github/workflows/release.yml):

1. Drift gate at the tag commit — every binding regenerates clean.
2. Per-language subdirectory tags `go/v<ver>`, `swift/v<ver>`,
   `typescript/v<ver>`.
3. GitHub release for the top-level tag with auto-generated notes.
4. npm publish via Trusted Publishing (OIDC), gated on
   `vars.NPM_TRUSTED_PUBLISHING_ENABLED`.

Bump `typescript/package.json` `version` in the same commit you tag (or
in the bump-only PR right before tagging).

## Known debt — pending follow-ups

The five existing v1 envelope types — `link_request`, `link_approval`,
`link_rejection`, `captcha_request`, `captcha_response` — currently
live in [`NaughtBot/core` `openapi/shared.yaml`](https://github.com/NaughtBot/core/blob/main/openapi/shared.yaml).
Migrating them out of core into this repo is a coordinated follow-up
that touches `mobile/`, `captcha/`, and `core/` simultaneously and is
tracked separately under the master plan
([`NaughtBot/workspace#3`](https://github.com/NaughtBot/workspace/issues/3)).
For the v0.1.0 series this repo carries the eight new signing /
decryption / derive / enroll envelope types only; the existing five
identifiers appear in the `MailboxEnvelopeType` enum so receivers can
validate against the canonical registry, but their payload schemas
still live in core.

## References

- Master tracker: [`NaughtBot/workspace#3`](https://github.com/NaughtBot/workspace/issues/3).
- Detailed bootstrap plan:
  [`workspace/plans/2026-05-11-0208Z-e2ee-payloads-bootstrap.md`](https://github.com/NaughtBot/workspace/blob/main/plans/2026-05-11-0208Z-e2ee-payloads-bootstrap.md).
- Cross-repo coordinator plan:
  [`workspace/plans/2026-05-11-0208Z-signing-types-cli-extraction.md`](https://github.com/NaughtBot/workspace/blob/main/plans/2026-05-11-0208Z-signing-types-cli-extraction.md).

## License

[MIT](LICENSE) © NaughtBot.
