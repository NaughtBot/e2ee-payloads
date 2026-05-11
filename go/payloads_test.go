// Package payloads tests cover JSON round-trip for representative schemas
// across every payload surface, plus the discriminator routing helpers
// generated for `oneOf` response payloads. They double as living
// documentation for the AGENTS.md "generator caveats" about Go-side
// permissive `As<Branch>` and lack of `additionalProperties: false`
// enforcement.
package payloads

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestEnvelopeRoundTrip exercises the canonical envelope wrapper: type is
// a plain string (so unknown values round-trip), payload is preserved as
// raw JSON (so per-`type` decoders never see int64 widening).
func TestEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()

	const envJSON = `{
		"v": 1,
		"type": "ssh_sign",
		"id": "9b2a4d1c-1f1f-4a4a-8f8f-1c1c1c1c1c1c",
		"issued_at": "2026-05-03T05:00:00.123Z",
		"payload": {"raw_data":"YWJj","device_key_id":"key_abc"}
	}`

	var env MailboxEnvelopeV1
	if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.V != MailboxEnvelopeV1VN1 {
		t.Errorf("V = %v, want %v", env.V, MailboxEnvelopeV1VN1)
	}
	if env.Type != "ssh_sign" {
		t.Errorf("Type = %q, want %q", env.Type, "ssh_sign")
	}
	wantId, err := uuid.Parse("9b2a4d1c-1f1f-4a4a-8f8f-1c1c1c1c1c1c")
	if err != nil {
		t.Fatalf("parse uuid: %v", err)
	}
	if env.Id != wantId {
		t.Errorf("Id = %v, want %v", env.Id, wantId)
	}
	if env.IssuedAt != "2026-05-03T05:00:00.123Z" {
		t.Errorf("IssuedAt = %q (must round-trip the literal RFC 3339 string)", env.IssuedAt)
	}

	// payload is json.RawMessage: bytes survive without int64 widening.
	if !bytes.Contains(env.Payload, []byte(`"raw_data"`)) {
		t.Errorf("Payload missing raw_data: %s", env.Payload)
	}

	out, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	// Decode again and compare structurally rather than byte-wise (Go's
	// JSON encoder reorders map keys deterministically, but spaces and
	// number formatting differ from the input).
	var roundTrip MailboxEnvelopeV1
	if err := json.Unmarshal(out, &roundTrip); err != nil {
		t.Fatalf("re-unmarshal envelope: %v", err)
	}
	if roundTrip.IssuedAt != env.IssuedAt {
		t.Errorf("issued_at lost on round-trip: got %q want %q", roundTrip.IssuedAt, env.IssuedAt)
	}
}

// TestEnvelopeAcceptsUnknownType demonstrates the forward-compat property:
// receivers can decode an envelope whose `type` is not yet in the
// published `MailboxEnvelopeType` enum, then log+drop per the v1
// contract.
func TestEnvelopeAcceptsUnknownType(t *testing.T) {
	t.Parallel()

	const envJSON = `{
		"v": 1,
		"type": "unregistered_v2_type",
		"id": "00000000-0000-0000-0000-000000000000",
		"issued_at": "2026-05-03T05:00:00Z",
		"payload": {}
	}`

	var env MailboxEnvelopeV1
	if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
		t.Fatalf("unmarshal envelope with unknown type: %v", err)
	}
	if env.Type != "unregistered_v2_type" {
		t.Errorf("Type = %q", env.Type)
	}
}

// TestSshSignRequestRoundTrip covers a request-side surface: snake_case
// field round-trip plus the required-field boundary.
func TestSshSignRequestRoundTrip(t *testing.T) {
	t.Parallel()

	in := MailboxSshSignRequestPayloadV1{
		RawData:     []byte("data to sign"),
		DeviceKeyId: "dev-key-1",
	}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"raw_data"`) {
		t.Errorf("expected snake_case raw_data in %s", out)
	}
	if !strings.Contains(string(out), `"device_key_id"`) {
		t.Errorf("expected snake_case device_key_id in %s", out)
	}

	var rt MailboxSshSignRequestPayloadV1
	if err := json.Unmarshal(out, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(rt.RawData) != "data to sign" {
		t.Errorf("RawData lost: %q", rt.RawData)
	}
}

// TestSshAuthResponseSuccessRouting asserts the documented routing pattern
// for `oneOf` response payloads. As called out in AGENTS.md, the
// generated `AsXxxSuccessV1` helper does NOT validate the success-only
// fields — the caller is responsible for inspecting the success-only
// fields after the call. This test pins the exact behaviour so the
// AGENTS.md guidance stays in sync with the generated code.
func TestSshAuthResponseSuccessRouting(t *testing.T) {
	t.Parallel()

	const successJSON = `{"signature":"YWJj"}`
	var resp MailboxSshAuthResponsePayloadV1
	if err := json.Unmarshal([]byte(successJSON), &resp); err != nil {
		t.Fatalf("unmarshal success: %v", err)
	}
	success, err := resp.AsMailboxSshAuthResponseSuccessV1()
	if err != nil {
		t.Fatalf("AsSuccess: %v", err)
	}
	if len(success.Signature) == 0 {
		t.Errorf("Signature must be set on success branch")
	}

	const failureJSON = `{"error_code":1,"error_message":"User rejected"}`
	if err := json.Unmarshal([]byte(failureJSON), &resp); err != nil {
		t.Fatalf("unmarshal failure: %v", err)
	}
	failure, err := resp.AsMailboxSshAuthResponseFailureV1()
	if err != nil {
		t.Fatalf("AsFailure: %v", err)
	}
	if failure.ErrorCode != 1 {
		t.Errorf("ErrorCode = %d, want 1", failure.ErrorCode)
	}

	// Documented Go-codegen caveat: AsSuccess on a failure-shaped
	// response succeeds but yields an empty success struct. The caller
	// MUST check the success-only field is non-zero.
	missclassified, err := resp.AsMailboxSshAuthResponseSuccessV1()
	if err != nil {
		t.Fatalf("AsSuccess on failure JSON returned error: %v (caveat docs claim it returns success-shaped zero value)", err)
	}
	if len(missclassified.Signature) != 0 {
		t.Errorf("documented caveat broken: Signature non-empty on failure JSON")
	}
}

// TestEnrollResponseDiscriminator exercises the discriminator-based
// routing for the enroll response. Unlike the SSH/GPG/age/PKCS#11
// responses, enroll uses an explicit `status` discriminator, so callers
// can route safely by inspecting the discriminator before extracting
// the branch.
func TestEnrollResponseDiscriminator(t *testing.T) {
	t.Parallel()

	const approvedJSON = `{
		"status": "approved",
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"public_key_hex": "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		"device_key_id": "dev-1",
		"algorithm": "ed25519"
	}`
	var resp MailboxEnrollResponsePayloadV1
	if err := json.Unmarshal([]byte(approvedJSON), &resp); err != nil {
		t.Fatalf("unmarshal approved: %v", err)
	}
	disc, err := resp.Discriminator()
	if err != nil {
		t.Fatalf("Discriminator: %v", err)
	}
	if disc != "approved" {
		t.Errorf("Discriminator = %q, want approved", disc)
	}
	approved, err := resp.AsMailboxEnrollResponseApprovedV1()
	if err != nil {
		t.Fatalf("AsApproved: %v", err)
	}
	if approved.Algorithm != "ed25519" {
		t.Errorf("Algorithm = %q, want ed25519", approved.Algorithm)
	}
}

// TestPayloadAdditionalPropertiesCaveat documents that Go's encoding/json
// does NOT enforce `additionalProperties: false`; callers MUST opt into
// strict decoding when they need that invariant.
func TestPayloadAdditionalPropertiesCaveat(t *testing.T) {
	t.Parallel()

	const payloadJSON = `{
		"raw_data": "YWJj",
		"device_key_id": "key_abc",
		"unknown_extra_field": "should be rejected by strict decoders"
	}`

	// Default decoder: silently accepts extra fields.
	var permissive MailboxSshSignRequestPayloadV1
	if err := json.Unmarshal([]byte(payloadJSON), &permissive); err != nil {
		t.Fatalf("default decoder: %v", err)
	}

	// Strict decoder: rejects extra fields per the AGENTS.md caveat.
	var strict MailboxSshSignRequestPayloadV1
	dec := json.NewDecoder(strings.NewReader(payloadJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&strict); err == nil {
		t.Errorf("strict decoder must reject unknown fields per AGENTS.md generator caveats")
	}
}
