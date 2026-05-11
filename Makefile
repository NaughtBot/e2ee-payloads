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

.PHONY: generate-check
generate-check: generate
	@echo "[generate-check] verifying no generator drift"
	@if ! git diff --quiet -- $(OPENAPI_DIR)/bundled $(GO_DIR) $(SWIFT_GEN_DIR) $(TS_SRC_DIR) 2>/dev/null; then \
	    echo "generate-check: generated files are stale. Run 'make generate' and commit the result." >&2; \
	    git --no-pager diff --stat -- $(OPENAPI_DIR)/bundled $(GO_DIR) $(SWIFT_GEN_DIR) $(TS_SRC_DIR); \
	    exit 1; \
	fi
	@echo "[generate-check] no drift"

# ----------------------------------------------------------------------------
# swift-openapi-generator binary cache
# ----------------------------------------------------------------------------

$(SWIFT_OPENAPI_GENERATOR_CLONE):
	@echo "[swift-openapi-generator] cloning $(SWIFT_OPENAPI_GENERATOR_GIT_TAG)"
	@git -c advice.detachedHead=false clone \
	    --branch "$(SWIFT_OPENAPI_GENERATOR_GIT_TAG)" \
	    --depth 1 \
	    "$(SWIFT_OPENAPI_GENERATOR_GIT_URL)" \
	    "$(SWIFT_OPENAPI_GENERATOR_CLONE)"

$(SWIFT_OPENAPI_GENERATOR_BIN): | $(SWIFT_OPENAPI_GENERATOR_CLONE)
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

.PHONY: build-go
build-go:
	@echo "[build-go] building $(GO_DIR)/..."
	@cd $(GO_DIR) && $(GO) build ./...

.PHONY: build-swift
build-swift:
	@echo "[build-swift] swift build"
	@$(SWIFT) build

.PHONY: build-typescript
build-typescript:
	@echo "[build-typescript] $(TS_DIR) tsc"
	@cd $(TS_DIR) && $(NPM) install --silent --no-audit --no-fund --prefer-offline >/dev/null
	@cd $(TS_DIR) && $(NPM) run build --silent

.PHONY: test
test: test-go test-swift test-typescript
	@echo "[test] done"

.PHONY: test-go
test-go:
	@echo "[test-go] go test ./go/..."
	@cd $(GO_DIR) && $(GO) test ./...

.PHONY: test-swift
test-swift:
	@echo "[test-swift] swift test"
	@$(SWIFT) test

.PHONY: test-typescript
test-typescript:
	@echo "[test-typescript] $(TS_DIR) test"
	@cd $(TS_DIR) && $(NPM) install --silent --no-audit --no-fund --prefer-offline >/dev/null
	@cd $(TS_DIR) && $(NPM) test --silent

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
