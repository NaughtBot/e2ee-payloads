# NaughtBot e2ee-payloads Makefile
#
# Source-of-truth repo for MailboxEnvelopeV1 and per-`type` payload schemas.
# OpenAPI files under openapi/ are the only hand-edited artefacts. The
# generated Go / Swift / TypeScript bindings are committed so downstream
# consumers can install without running the generator stack; CI's
# `make generate-check` enforces freshness on every change.
#
# WS1.1 lands the skeleton: directory layout plus stub targets. WS1.3 wires
# the real generator commands; WS1.4 produces the first generated outputs;
# WS1.5/1.6 add the CI and release workflows.

GO          ?= go
PNPM        ?= pnpm
NPM         ?= npm
SWIFT       ?= swift

OPENAPI_DIR := openapi
GO_DIR      := go
SWIFT_DIR   := swift
TS_DIR      := typescript

.PHONY: all
all: generate build test

.PHONY: help
help:
	@echo "NaughtBot e2ee-payloads — available targets:"
	@echo ""
	@echo "  make openapi-lint    Lint OpenAPI source files under $(OPENAPI_DIR)/."
	@echo "  make generate        Regenerate Go / Swift / TypeScript bindings."
	@echo "  make generate-check  Regenerate and fail if outputs drift (CI gate)."
	@echo "  make build           Build every language binding."
	@echo "  make test            Run tests for every language binding."
	@echo ""
	@echo "Until WS1.3 lands, every target is a TODO stub."

# ----------------------------------------------------------------------------
# OpenAPI lint
# ----------------------------------------------------------------------------

.PHONY: openapi-lint
openapi-lint:
	@echo "TODO(WS1.3): wire Redocly lint over $(OPENAPI_DIR)/**.yaml"

# ----------------------------------------------------------------------------
# Code generation
# ----------------------------------------------------------------------------

.PHONY: generate
generate:
	@echo "TODO(WS1.3): regenerate Go / Swift / TypeScript bindings from $(OPENAPI_DIR)/"

.PHONY: generate-check
generate-check: generate
	@echo "TODO(WS1.3): assert git diff is empty after generate"

# ----------------------------------------------------------------------------
# Build / test
# ----------------------------------------------------------------------------

.PHONY: build
build:
	@echo "TODO(WS1.3): build Go / Swift / TypeScript packages"

.PHONY: test
test:
	@echo "TODO(WS1.3): test Go / Swift / TypeScript packages"
