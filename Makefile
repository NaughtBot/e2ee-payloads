# NaughtBot e2ee-payloads Makefile
#
# Source-of-truth repo for MailboxEnvelopeV1 and per-`type` payload schemas.
# OpenAPI files under openapi/ are the only hand-edited artefacts. The
# generated Go / Swift / TypeScript bindings are committed so downstream
# consumers can install without running the generator stack; the
# `generate-check` target (CI gate) enforces freshness on every change.

GO          ?= go
PNPM        ?= pnpm
NPM         ?= npm
SWIFT       ?= swift
NPX         ?= npx -y -p @redocly/cli@latest

OPENAPI_DIR     := openapi
OPENAPI_BUNDLE  := $(OPENAPI_DIR)/bundled/payloads.yaml
OPENAPI_ROOT    := $(OPENAPI_DIR)/payloads.yaml
OPENAPI_SOURCES := $(wildcard $(OPENAPI_DIR)/*.yaml) $(wildcard $(OPENAPI_DIR)/payloads/*.yaml)

GO_DIR      := go
GO_GEN      := $(GO_DIR)/types.gen.go
GO_CONFIG   := $(GO_DIR)/generate.cfg.yaml

SWIFT_DIR        := swift
SWIFT_GEN_DIR    := $(SWIFT_DIR)/Sources/NaughtBotE2EEPayloads
SWIFT_CONFIG     := $(SWIFT_DIR)/openapi-generator-config.yaml
SWIFT_GEN_OPENAPI := $(SWIFT_GEN_DIR)/openapi.yaml
SWIFT_GEN_CONFIG  := $(SWIFT_GEN_DIR)/openapi-generator-config.yaml

# Pin the generator clone so every host produces byte-identical output.
SWIFT_OPENAPI_GENERATOR_GIT_URL ?= https://github.com/apple/swift-openapi-generator
SWIFT_OPENAPI_GENERATOR_GIT_TAG ?= 1.10.2
SWIFT_OPENAPI_GENERATOR_CLONE   ?= .swift-openapi-generator
SWIFT_OPENAPI_GENERATOR_BIN     := $(SWIFT_OPENAPI_GENERATOR_CLONE)/.build/release/swift-openapi-generator

TS_DIR      := typescript
TS_SRC_DIR  := $(TS_DIR)/src
TS_SCHEMA   := $(TS_SRC_DIR)/schema.ts

.PHONY: all
all: generate build test

.PHONY: help
help:
	@echo "NaughtBot e2ee-payloads — available targets:"
	@echo ""
	@echo "  make openapi-lint    Lint every OpenAPI source under $(OPENAPI_DIR)/."
	@echo "  make bundle          Bundle openapi/payloads.yaml -> $(OPENAPI_BUNDLE)."
	@echo "  make generate        Regenerate Go / Swift / TypeScript bindings."
	@echo "  make generate-check  Regenerate and fail if outputs drift (CI gate)."
	@echo "  make build           Build every language binding."
	@echo "  make test            Run tests for every language binding."
	@echo "  make clean           Remove generated outputs."
	@echo "  make distclean       Remove generated outputs and tool caches."
	@echo ""

# ----------------------------------------------------------------------------
# OpenAPI lint + bundle
# ----------------------------------------------------------------------------

.PHONY: openapi-lint
openapi-lint:
	@echo "[openapi-lint] running Redocly lint over $(OPENAPI_DIR)/"
	$(NPX) redocly lint --config $(OPENAPI_DIR)/redocly.yaml

.PHONY: bundle
bundle: $(OPENAPI_BUNDLE)

$(OPENAPI_BUNDLE): $(OPENAPI_SOURCES)
	@mkdir -p $(OPENAPI_DIR)/bundled
	@echo "[bundle] $(OPENAPI_ROOT) -> $@"
	$(NPX) redocly bundle $(OPENAPI_ROOT) --output $@ >/dev/null

# ----------------------------------------------------------------------------
# Code generation
# ----------------------------------------------------------------------------

.PHONY: generate
generate: generate-go generate-swift generate-typescript
	@echo "[generate] done"

.PHONY: generate-go
generate-go: bundle
	@echo "[generate-go] regenerating $(GO_GEN)"
	@cd $(GO_DIR) && $(GO) tool oapi-codegen \
	    -config $(notdir $(GO_CONFIG)) \
	    -o $(notdir $(GO_GEN)) \
	    ../$(OPENAPI_BUNDLE) >/dev/null

.PHONY: generate-swift
generate-swift: bundle $(SWIFT_OPENAPI_GENERATOR_BIN)
	@echo "[generate-swift] regenerating $(SWIFT_GEN_DIR)/"
	@mkdir -p $(SWIFT_GEN_DIR)
	@cp $(OPENAPI_BUNDLE) $(SWIFT_GEN_OPENAPI)
	@cp $(SWIFT_CONFIG) $(SWIFT_GEN_CONFIG)
	@$(SWIFT_OPENAPI_GENERATOR_BIN) generate \
	    --config $(SWIFT_GEN_CONFIG) \
	    --output-directory $(SWIFT_GEN_DIR) \
	    $(SWIFT_GEN_OPENAPI)

.PHONY: generate-typescript
generate-typescript: bundle
	@echo "[generate-typescript] regenerating $(TS_SCHEMA)"
	@mkdir -p $(TS_SRC_DIR)
	@cd $(TS_DIR) && $(NPM) install --silent --no-audit --no-fund --prefer-offline >/dev/null
	@$(TS_DIR)/node_modules/.bin/openapi-typescript $(OPENAPI_BUNDLE) --output $(TS_SCHEMA)
	@$(MAKE) --no-print-directory $(TS_SRC_DIR)/index.ts

# Re-export aliases authored alongside the generator: re-running `make
# generate-typescript` recreates them so consumers always import a stable
# surface from `@naughtbot/e2ee-payloads`.
$(TS_SRC_DIR)/index.ts: $(OPENAPI_BUNDLE)
	@echo "[generate-typescript] writing $@"
	@printf '%s\n' \
	    '// This file is auto-generated alongside schema.ts. Do not edit by hand.' \
	    '// Adding a new schema requires re-running `make generate-typescript`.' \
	    'export type { components, paths, webhooks } from "./schema.js";' \
	    '' \
	    'import type { components } from "./schema.js";' \
	    '' \
	    'export type MailboxEnvelopeV1 = components["schemas"]["MailboxEnvelopeV1"];' \
	    'export type MailboxEnvelopeType = components["schemas"]["MailboxEnvelopeType"];' \
	    'export type MailboxSshAuthRequestPayloadV1 = components["schemas"]["MailboxSshAuthRequestPayloadV1"];' \
	    'export type MailboxSshAuthResponsePayloadV1 = components["schemas"]["MailboxSshAuthResponsePayloadV1"];' \
	    'export type MailboxSshAuthResponseSuccessV1 = components["schemas"]["MailboxSshAuthResponseSuccessV1"];' \
	    'export type MailboxSshAuthResponseFailureV1 = components["schemas"]["MailboxSshAuthResponseFailureV1"];' \
	    'export type MailboxSshSignRequestPayloadV1 = components["schemas"]["MailboxSshSignRequestPayloadV1"];' \
	    'export type MailboxSshSignResponsePayloadV1 = components["schemas"]["MailboxSshSignResponsePayloadV1"];' \
	    'export type MailboxSshSignResponseSuccessV1 = components["schemas"]["MailboxSshSignResponseSuccessV1"];' \
	    'export type MailboxSshSignResponseFailureV1 = components["schemas"]["MailboxSshSignResponseFailureV1"];' \
	    'export type MailboxGpgSignRequestPayloadV1 = components["schemas"]["MailboxGpgSignRequestPayloadV1"];' \
	    'export type MailboxGpgSignResponsePayloadV1 = components["schemas"]["MailboxGpgSignResponsePayloadV1"];' \
	    'export type MailboxGpgSignResponseSuccessV1 = components["schemas"]["MailboxGpgSignResponseSuccessV1"];' \
	    'export type MailboxGpgSignResponseFailureV1 = components["schemas"]["MailboxGpgSignResponseFailureV1"];' \
	    'export type MailboxGpgDecryptRequestPayloadV1 = components["schemas"]["MailboxGpgDecryptRequestPayloadV1"];' \
	    'export type MailboxGpgDecryptResponsePayloadV1 = components["schemas"]["MailboxGpgDecryptResponsePayloadV1"];' \
	    'export type MailboxGpgDecryptResponseSuccessV1 = components["schemas"]["MailboxGpgDecryptResponseSuccessV1"];' \
	    'export type MailboxGpgDecryptResponseFailureV1 = components["schemas"]["MailboxGpgDecryptResponseFailureV1"];' \
	    'export type MailboxAgeUnwrapRequestPayloadV1 = components["schemas"]["MailboxAgeUnwrapRequestPayloadV1"];' \
	    'export type MailboxAgeUnwrapResponsePayloadV1 = components["schemas"]["MailboxAgeUnwrapResponsePayloadV1"];' \
	    'export type MailboxAgeUnwrapResponseSuccessV1 = components["schemas"]["MailboxAgeUnwrapResponseSuccessV1"];' \
	    'export type MailboxAgeUnwrapResponseFailureV1 = components["schemas"]["MailboxAgeUnwrapResponseFailureV1"];' \
	    'export type MailboxPkcs11SignRequestPayloadV1 = components["schemas"]["MailboxPkcs11SignRequestPayloadV1"];' \
	    'export type MailboxPkcs11SignResponsePayloadV1 = components["schemas"]["MailboxPkcs11SignResponsePayloadV1"];' \
	    'export type MailboxPkcs11SignResponseSuccessV1 = components["schemas"]["MailboxPkcs11SignResponseSuccessV1"];' \
	    'export type MailboxPkcs11SignResponseFailureV1 = components["schemas"]["MailboxPkcs11SignResponseFailureV1"];' \
	    'export type MailboxPkcs11DeriveRequestPayloadV1 = components["schemas"]["MailboxPkcs11DeriveRequestPayloadV1"];' \
	    'export type MailboxPkcs11DeriveResponsePayloadV1 = components["schemas"]["MailboxPkcs11DeriveResponsePayloadV1"];' \
	    'export type MailboxPkcs11DeriveResponseSuccessV1 = components["schemas"]["MailboxPkcs11DeriveResponseSuccessV1"];' \
	    'export type MailboxPkcs11DeriveResponseFailureV1 = components["schemas"]["MailboxPkcs11DeriveResponseFailureV1"];' \
	    'export type MailboxEnrollRequestPayloadV1 = components["schemas"]["MailboxEnrollRequestPayloadV1"];' \
	    'export type MailboxEnrollResponsePayloadV1 = components["schemas"]["MailboxEnrollResponsePayloadV1"];' \
	    'export type MailboxEnrollResponseApprovedV1 = components["schemas"]["MailboxEnrollResponseApprovedV1"];' \
	    'export type MailboxEnrollResponseRejectedV1 = components["schemas"]["MailboxEnrollResponseRejectedV1"];' \
	    > $@

.PHONY: generate-check
generate-check: generate
	@echo "[generate-check] verifying no generator drift"
	@# `git diff --quiet` ignores untracked files, so also check that the
	@# tracked-paths working tree matches HEAD AND that no new untracked
	@# files appeared under the generated directories. Both signals must
	@# stay clean for the check to pass.
	@dirty=$$(git status --porcelain -- \
	    $(OPENAPI_DIR)/bundled $(GO_DIR) $(SWIFT_GEN_DIR) $(TS_SRC_DIR) 2>/dev/null); \
	if [ -n "$$dirty" ]; then \
	    echo "generate-check: generated files are stale. Run 'make generate' and commit the result." >&2; \
	    printf '%s\n' "$$dirty"; \
	    exit 1; \
	fi
	@echo "[generate-check] no drift"

# ----------------------------------------------------------------------------
# swift-openapi-generator binary cache
# ----------------------------------------------------------------------------

# Refresh the cached clone whenever the pinned tag changes. We stash the
# active tag in a stamp file; if the pin moves, we wipe the clone and
# reclone at the new tag, forcing a binary rebuild.
SWIFT_OPENAPI_GENERATOR_TAG_STAMP := $(SWIFT_OPENAPI_GENERATOR_CLONE)/.tag

.PHONY: swift-openapi-generator-clone
swift-openapi-generator-clone: $(SWIFT_OPENAPI_GENERATOR_TAG_STAMP)

$(SWIFT_OPENAPI_GENERATOR_TAG_STAMP):
	@if [ -d "$(SWIFT_OPENAPI_GENERATOR_CLONE)" ] && [ "$$(cat $@ 2>/dev/null)" != "$(SWIFT_OPENAPI_GENERATOR_GIT_TAG)" ]; then \
	    echo "[swift-openapi-generator] pin changed -> wiping $(SWIFT_OPENAPI_GENERATOR_CLONE)"; \
	    rm -rf "$(SWIFT_OPENAPI_GENERATOR_CLONE)"; \
	fi
	@if [ ! -d "$(SWIFT_OPENAPI_GENERATOR_CLONE)" ]; then \
	    echo "[swift-openapi-generator] cloning $(SWIFT_OPENAPI_GENERATOR_GIT_TAG)"; \
	    git -c advice.detachedHead=false clone \
	        --branch "$(SWIFT_OPENAPI_GENERATOR_GIT_TAG)" \
	        --depth 1 \
	        "$(SWIFT_OPENAPI_GENERATOR_GIT_URL)" \
	        "$(SWIFT_OPENAPI_GENERATOR_CLONE)"; \
	fi
	@printf '%s' "$(SWIFT_OPENAPI_GENERATOR_GIT_TAG)" > $@

$(SWIFT_OPENAPI_GENERATOR_BIN): $(SWIFT_OPENAPI_GENERATOR_TAG_STAMP)
	@echo "[swift-openapi-generator] building $@"
	@$(SWIFT) build \
	    --package-path "$(SWIFT_OPENAPI_GENERATOR_CLONE)" \
	    --configuration release \
	    --product swift-openapi-generator

# ----------------------------------------------------------------------------
# Build / test
# ----------------------------------------------------------------------------

.PHONY: build
build: build-go build-swift build-typescript
	@echo "[build] done"

# Generated outputs land in WS1.4. Until then, build/test ensure they
# exist by running the generator first; once committed, the generator is
# a no-op on the second run.
.PHONY: build-go
build-go: generate-go
	@echo "[build-go] building $(GO_DIR)/..."
	@cd $(GO_DIR) && $(GO) build ./...

.PHONY: build-swift
build-swift: generate-swift
	@if [ -f Package.swift ]; then \
	    echo "[build-swift] swift build"; \
	    $(SWIFT) build; \
	else \
	    echo "[build-swift] Package.swift not present — skipping (lands in WS1.4)"; \
	fi

.PHONY: build-typescript
build-typescript: generate-typescript
	@echo "[build-typescript] $(TS_DIR) tsc"
	@cd $(TS_DIR) && $(NPM) install --silent --no-audit --no-fund --prefer-offline >/dev/null
	@cd $(TS_DIR) && $(NPM) run build --silent

.PHONY: test
test: test-go test-swift test-typescript
	@echo "[test] done"

.PHONY: test-go
test-go: generate-go
	@echo "[test-go] go test ./go/..."
	@cd $(GO_DIR) && $(GO) test ./...

.PHONY: test-swift
test-swift: generate-swift
	@if [ -f Package.swift ]; then \
	    echo "[test-swift] swift test"; \
	    $(SWIFT) test; \
	else \
	    echo "[test-swift] Package.swift not present — skipping (lands in WS1.4)"; \
	fi

.PHONY: test-typescript
test-typescript: generate-typescript
	@echo "[test-typescript] $(TS_DIR) test"
	@cd $(TS_DIR) && $(NPM) install --silent --no-audit --no-fund --prefer-offline >/dev/null
	@if [ -f $(TS_SRC_DIR)/index.test.ts ]; then \
	    cd $(TS_DIR) && $(NPM) test --silent; \
	else \
	    echo "[test-typescript] $(TS_SRC_DIR)/index.test.ts not present — skipping (lands in WS1.5 smoke tests)"; \
	fi

# ----------------------------------------------------------------------------
# Clean
# ----------------------------------------------------------------------------

.PHONY: clean
clean:
	rm -rf $(OPENAPI_DIR)/bundled
	rm -f $(GO_GEN)
	rm -f $(SWIFT_GEN_OPENAPI) $(SWIFT_GEN_CONFIG)
	@find $(SWIFT_GEN_DIR) -maxdepth 1 -name '*.swift' -delete 2>/dev/null || true
	rm -f $(TS_SCHEMA)
	rm -rf $(TS_DIR)/dist $(TS_DIR)/node_modules

.PHONY: distclean
distclean: clean
	rm -rf $(SWIFT_OPENAPI_GENERATOR_CLONE) .build
