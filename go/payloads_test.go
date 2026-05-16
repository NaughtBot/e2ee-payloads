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

const browserApprovalDecisionBindingFixtureJSON = `{"approval_id":"appr_browser_approval_fixture","browser_public_key_algorithm":"ES256","browser_public_key_thumbprint":"sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA","decided_at":"2026-05-14T19:31:00Z","decision":"approved","expires_at":"2026-05-14T19:35:00Z","nonce":"m4H2YxTjueEXAMPLE","pairing_transcript_hash":"sha256:6f5902ac237024bdd0c176cb93063dc4f1e01e1191450b5f8f457c56f48e1f4f","request_envelope_id":"11111111-2222-4333-8444-555555555555","request_envelope_issued_at":"2026-05-14T19:30:00Z","request_envelope_type":"browser_approval_request","requested_capability":"captcha.browser_credential","requester_client_id":"captcha-service","requester_origin":"https://captcha.naughtbot.com","service_mobile_pairing_id":"pair_9d58fb4c6ff84f46","version":"browser-approval-decision-binding/v1"}`
const firstPartyRegisterActionFixtureJSON = `{"action_type":"relying_party.register","client_secret_returned_once":true,"confidential_client_audience":"verify.api","confidential_client_scopes":["verify:proof"],"display_name":"Customer Portal","origin":"https://customer.example","public_client_scopes":["openid","offline_access","mailbox:pairing:start"],"redirect_uris":["https://customer.example/oauth/callback"]}`
const firstPartyRegisterActionFixtureHash = "sha256:9d235424c61b5923caae6be03894226a09b80a4cc28d015dc7ac6260424ed1d7"
const firstPartyDecisionBindingFixtureJSON = `{"action_type":"relying_party.register","approving_device_id":"33333333-4444-4555-8666-777777777777","approving_device_signing_key_jkt":"uJx87scLEhI5vT1YdtXx5ERw2IW0aP2mMNJ1lUu1Dx4","canonical_action_hash":"sha256:9d235424c61b5923caae6be03894226a09b80a4cc28d015dc7ac6260424ed1d7","decided_at":"2026-05-16T20:41:00Z","decision":"approved","expires_at":"2026-05-16T20:45:00Z","intent_id":"pai_first_party_fixture","nonce":"first-party-nonce-fixture","request_envelope_id":"11111111-2222-4333-8444-555555555555","request_envelope_issued_at":"2026-05-16T20:40:00Z","request_envelope_type":"first_party_request","request_id":"fpr_first_party_fixture","version":"first-party-privileged-action-decision-binding/v1"}`

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

	const successJSON = `{"signature":"YWJj","flags":1,"counter":7}`
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
	if success.Counter != 7 {
		t.Errorf("Counter = %d, want 7", success.Counter)
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

// TestSshAuthResponseSuccessCounterRoundTrip is a regression test for
// NaughtBot/e2ee-payloads#17: the SK monotonic counter and per-signature
// assertion flags byte must survive a JSON encode/decode round-trip on the
// SSH auth success branch. Without these fields the requester cannot
// rebuild the OpenSSH SK signature preimage
// `SHA256(application) || flags || counter || SHA256(data)`.
func TestSshAuthResponseSuccessCounterRoundTrip(t *testing.T) {
	t.Parallel()

	in := MailboxSshAuthResponseSuccessV1{
		Signature: []byte("ssh-sk-sig"),
		Flags:     1,
		Counter:   7,
	}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"counter":7`) {
		t.Errorf("expected counter in %s", out)
	}
	if !strings.Contains(string(out), `"flags":1`) {
		t.Errorf("expected flags in %s", out)
	}

	var rt MailboxSshAuthResponseSuccessV1
	if err := json.Unmarshal(out, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.Counter != 7 {
		t.Errorf("Counter = %d, want 7", rt.Counter)
	}
	if rt.Flags != 1 {
		t.Errorf("Flags = %d, want 1", rt.Flags)
	}
	if string(rt.Signature) != "ssh-sk-sig" {
		t.Errorf("Signature lost: %q", rt.Signature)
	}

	// Boundary: u32 max counter and u8 max flags round-trip without overflow.
	in.Counter = 4294967295
	in.Flags = 255
	out, err = json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal max counter: %v", err)
	}
	var maxRT MailboxSshAuthResponseSuccessV1
	if err := json.Unmarshal(out, &maxRT); err != nil {
		t.Fatalf("unmarshal max counter: %v", err)
	}
	if maxRT.Counter != 4294967295 {
		t.Errorf("max Counter lost: got %d want 4294967295", maxRT.Counter)
	}
	if maxRT.Flags != 255 {
		t.Errorf("max Flags lost: got %d want 255", maxRT.Flags)
	}
}

// TestBrowserApprovalDecisionBindingFixture pins the deterministic JSON
// bytes signed by mobile for the generic browser approval response. The
// matching Swift and TypeScript tests use the same fixture string.
func TestBrowserApprovalDecisionBindingFixture(t *testing.T) {
	t.Parallel()

	requestEnvelopeId, err := uuid.Parse("11111111-2222-4333-8444-555555555555")
	if err != nil {
		t.Fatalf("parse request envelope id: %v", err)
	}
	challenge := approvalChallengeFixture()
	request := MailboxBrowserApprovalRequestPayloadV1{
		ApprovalChallenge:          challenge,
		ApprovalId:                 "appr_browser_approval_fixture",
		BrowserDisplayName:         "Chrome on MacBook Pro",
		BrowserPlatform:            "macOS",
		BrowserPublicKeyAlgorithm:  "ES256",
		BrowserPublicKeyThumbprint: "sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA",
		ExpiresAt:                  "2026-05-14T19:35:00Z",
		IssuedAt:                   "2026-05-14T19:30:00Z",
		Nonce:                      "m4H2YxTjueEXAMPLE",
		RequestedCapability:        "captcha.browser_credential",
		RequesterClientId:          "captcha-service",
		RequesterDisplayName:       "NaughtBot Captcha",
		RequesterOrigin:            "https://captcha.naughtbot.com",
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var requestRoundTrip MailboxBrowserApprovalRequestPayloadV1
	if err := json.Unmarshal(requestJSON, &requestRoundTrip); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if requestRoundTrip.ApprovalChallenge != challenge {
		t.Errorf("approval challenge lost:\n got: %#v\nwant: %#v", requestRoundTrip.ApprovalChallenge, challenge)
	}

	binding := MailboxBrowserApprovalDecisionBindingV1{
		ApprovalId:                 "appr_browser_approval_fixture",
		BrowserPublicKeyAlgorithm:  "ES256",
		BrowserPublicKeyThumbprint: "sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA",
		DecidedAt:                  "2026-05-14T19:31:00Z",
		Decision:                   MailboxBrowserApprovalDecisionApproved,
		ExpiresAt:                  "2026-05-14T19:35:00Z",
		Nonce:                      "m4H2YxTjueEXAMPLE",
		PairingTranscriptHash:      "sha256:6f5902ac237024bdd0c176cb93063dc4f1e01e1191450b5f8f457c56f48e1f4f",
		RequestEnvelopeId:          requestEnvelopeId,
		RequestEnvelopeIssuedAt:    "2026-05-14T19:30:00Z",
		RequestEnvelopeType:        MailboxBrowserApprovalDecisionBindingV1RequestEnvelopeTypeBrowserApprovalRequest,
		RequestedCapability:        "captcha.browser_credential",
		RequesterClientId:          "captcha-service",
		RequesterOrigin:            "https://captcha.naughtbot.com",
		ServiceMobilePairingId:     "pair_9d58fb4c6ff84f46",
		Version:                    BrowserApprovalDecisionBindingv1,
	}
	out, err := json.Marshal(binding)
	if err != nil {
		t.Fatalf("marshal binding: %v", err)
	}
	if string(out) != browserApprovalDecisionBindingFixtureJSON {
		t.Fatalf("binding JSON mismatch:\n got: %s\nwant: %s", out, browserApprovalDecisionBindingFixtureJSON)
	}

	response := MailboxBrowserApprovalResponsePayloadV1{
		ApprovalBindingBytes:  out,
		ApprovalBindingFormat: BrowserApprovalDecisionBindingv1Json,
		ApprovalId:            binding.ApprovalId,
		ApprovalProof:         approvalProofFixture(),
		DecidedAt:             binding.DecidedAt,
		Decision:              binding.Decision,
		RequestEnvelopeId:     binding.RequestEnvelopeId,
		Status:                MailboxBrowserApprovalResponseStatusDecided,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if !strings.Contains(string(responseJSON), `"approval_binding_format":"browser-approval-decision-binding/v1+json"`) {
		t.Errorf("response missing binding format: %s", responseJSON)
	}

	var roundTrip MailboxBrowserApprovalResponsePayloadV1
	if err := json.Unmarshal(responseJSON, &roundTrip); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if string(roundTrip.ApprovalBindingBytes) != browserApprovalDecisionBindingFixtureJSON {
		t.Errorf("approval binding bytes lost:\n got: %s\nwant: %s", roundTrip.ApprovalBindingBytes, browserApprovalDecisionBindingFixtureJSON)
	}
	if string(roundTrip.ApprovalProof.Proof) != "browser-approval-proof" {
		t.Errorf("approval proof lost: %q", roundTrip.ApprovalProof.Proof)
	}
}

// TestFirstPartyPrivilegedActionFixture pins the canonical action JSON and
// signed decision binding used by backend-created first-party approval
// requests. The matching Swift and TypeScript tests use the same fixture
// strings.
func TestFirstPartyPrivilegedActionFixture(t *testing.T) {
	t.Parallel()

	requestEnvelopeId, err := uuid.Parse("11111111-2222-4333-8444-555555555555")
	if err != nil {
		t.Fatalf("parse request envelope id: %v", err)
	}
	approvingDeviceId, err := uuid.Parse("33333333-4444-4555-8666-777777777777")
	if err != nil {
		t.Fatalf("parse approving device id: %v", err)
	}

	action := MailboxFirstPartyRelyingPartyRegisterActionV1{
		ActionType:                 RelyingPartyRegister,
		ClientSecretReturnedOnce:   true,
		ConfidentialClientAudience: "verify.api",
		ConfidentialClientScopes:   []string{"verify:proof"},
		DisplayName:                "Customer Portal",
		Origin:                     "https://customer.example",
		PublicClientScopes:         []string{"openid", "offline_access", "mailbox:pairing:start"},
		RedirectUris:               []string{"https://customer.example/oauth/callback"},
	}
	actionEncoded, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("marshal action: %v", err)
	}
	if string(actionEncoded) != firstPartyRegisterActionFixtureJSON {
		t.Fatalf("canonical action JSON mismatch:\n got %s\nwant %s", actionEncoded, firstPartyRegisterActionFixtureJSON)
	}

	var actionUnion MailboxFirstPartyPrivilegedAction
	if err := actionUnion.FromMailboxFirstPartyRelyingPartyRegisterActionV1(action); err != nil {
		t.Fatalf("wrap action union: %v", err)
	}
	request := MailboxFirstPartyRequestPayloadV1{
		ExpiresAt: "2026-05-16T20:45:00Z",
		IssuedAt:  "2026-05-16T20:40:00Z",
		Nonce:     "first-party-nonce-fixture",
		PrivilegedAction: MailboxFirstPartyPrivilegedActionRequestV1{
			Action:               actionUnion,
			ActionType:           MailboxFirstPartyPrivilegedActionTypeRelyingPartyRegister,
			CanonicalActionBytes: []byte(firstPartyRegisterActionFixtureJSON),
			CanonicalActionHash:  firstPartyRegisterActionFixtureHash,
			CreatedAt:            "2026-05-16T20:40:00Z",
			InitiatingClientId:   "naughtbot-console",
			InitiatingDpopJkt:    "2oNQXcW2Upi5b1xHZQW1Yf3N0aYVnX_Jf7mRiS7Jm8A",
			IntentId:             "pai_first_party_fixture",
		},
		RequestId:   "fpr_first_party_fixture",
		RequestKind: PrivilegedActionApproval,
	}
	requestEncoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var requestDecoded MailboxFirstPartyRequestPayloadV1
	if err := json.Unmarshal(requestEncoded, &requestDecoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	decodedAction, err := requestDecoded.PrivilegedAction.Action.AsMailboxFirstPartyRelyingPartyRegisterActionV1()
	if err != nil {
		t.Fatalf("decode action union: %v", err)
	}
	if decodedAction.Origin != action.Origin {
		t.Errorf("decoded action origin = %q, want %q", decodedAction.Origin, action.Origin)
	}
	if string(requestDecoded.PrivilegedAction.CanonicalActionBytes) != firstPartyRegisterActionFixtureJSON {
		t.Errorf("canonical action bytes lost: %q", requestDecoded.PrivilegedAction.CanonicalActionBytes)
	}

	binding := MailboxFirstPartyPrivilegedActionDecisionBindingV1{
		ActionType:                   MailboxFirstPartyPrivilegedActionTypeRelyingPartyRegister,
		ApprovingDeviceId:            approvingDeviceId,
		ApprovingDeviceSigningKeyJkt: "uJx87scLEhI5vT1YdtXx5ERw2IW0aP2mMNJ1lUu1Dx4",
		CanonicalActionHash:          firstPartyRegisterActionFixtureHash,
		DecidedAt:                    "2026-05-16T20:41:00Z",
		Decision:                     Approved,
		ExpiresAt:                    "2026-05-16T20:45:00Z",
		IntentId:                     "pai_first_party_fixture",
		Nonce:                        "first-party-nonce-fixture",
		RequestEnvelopeId:            requestEnvelopeId,
		RequestEnvelopeIssuedAt:      "2026-05-16T20:40:00Z",
		RequestEnvelopeType:          FirstPartyRequest,
		RequestId:                    "fpr_first_party_fixture",
		Version:                      FirstPartyPrivilegedActionDecisionBindingv1,
	}
	bindingEncoded, err := json.Marshal(binding)
	if err != nil {
		t.Fatalf("marshal binding: %v", err)
	}
	if string(bindingEncoded) != firstPartyDecisionBindingFixtureJSON {
		t.Fatalf("canonical binding JSON mismatch:\n got %s\nwant %s", bindingEncoded, firstPartyDecisionBindingFixtureJSON)
	}

	response := MailboxFirstPartyResponsePayloadV1{
		ApprovalBindingBytes:         bindingEncoded,
		ApprovalBindingFormat:        FirstPartyPrivilegedActionDecisionBindingv1Json,
		ApprovalSignature:            []byte("first-party-signature"),
		ApprovalSignatureAlgorithm:   "ES256",
		ApprovingDeviceId:            approvingDeviceId,
		ApprovingDeviceSigningKeyJkt: "uJx87scLEhI5vT1YdtXx5ERw2IW0aP2mMNJ1lUu1Dx4",
		DecidedAt:                    binding.DecidedAt,
		Decision:                     binding.Decision,
		IntentId:                     binding.IntentId,
		RequestEnvelopeId:            requestEnvelopeId,
		RequestId:                    binding.RequestId,
		Status:                       MailboxFirstPartyResponseStatusDecided,
	}
	responseEncoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var responseDecoded MailboxFirstPartyResponsePayloadV1
	if err := json.Unmarshal(responseEncoded, &responseDecoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if string(responseDecoded.ApprovalBindingBytes) != firstPartyDecisionBindingFixtureJSON {
		t.Errorf("approval binding bytes lost: %q", responseDecoded.ApprovalBindingBytes)
	}
	if string(responseDecoded.ApprovalSignature) != "first-party-signature" {
		t.Errorf("approval signature lost: %q", responseDecoded.ApprovalSignature)
	}
}

func approvalChallengeFixture() ApprovalChallenge {
	return ApprovalChallenge{
		Version:       ApprovalChallengev1,
		Nonce:         "browser-approval-nonce",
		RequestId:     "11111111-2222-4333-8444-555555555555",
		PlaintextHash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
}

func approvalProofFixture() ApprovalAttestedKeyProof {
	return ApprovalAttestedKeyProof{
		Version:   ApprovalAttestedKeyProofv1,
		Challenge: approvalChallengeFixture(),
		Statement: ApprovalProofStatement{
			IssuerPublicKeyHex: "02aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			AppIdHashHex:       "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			PolicyVersion:      1,
			Now:                1778787060,
			ChallengeNonceHex:  "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			AudienceHashHex:    "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			ApprovalHashHex:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		Attestation: ApprovalAttestationV1{
			Version:   ApprovalAttestationv1,
			Bytes:     []byte("attestation-bytes"),
			Signature: []byte("attestation-signature"),
		},
		Proof: []byte("browser-approval-proof"),
	}
}

// TestSshSignResponseSuccessCounterRoundTrip is the matching regression
// test for the `ssh_sign` (e.g. SSH commit-signing) response branch, which
// shares the same SK preimage construction as `ssh_auth`. Both `flags`
// and `counter` are required so the requester can rebuild the preimage.
func TestSshSignResponseSuccessCounterRoundTrip(t *testing.T) {
	t.Parallel()

	in := MailboxSshSignResponseSuccessV1{
		Signature: []byte("ssh-sk-sig"),
		Flags:     1,
		Counter:   42,
	}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"counter":42`) {
		t.Errorf("expected counter in %s", out)
	}
	if !strings.Contains(string(out), `"flags":1`) {
		t.Errorf("expected flags in %s", out)
	}

	var rt MailboxSshSignResponseSuccessV1
	if err := json.Unmarshal(out, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.Counter != 42 {
		t.Errorf("Counter = %d, want 42", rt.Counter)
	}
	if rt.Flags != 1 {
		t.Errorf("Flags = %d, want 1", rt.Flags)
	}
}

// TestEnrollResponseApprovedSshSkFlagsRoundTrip is the regression test for
// the per-credential `ssh_sk_flags` byte added in v0.2.0. The requester
// MUST persist this value alongside the credential public key and embed
// it into the OpenSSH SK signature preimage on every subsequent
// `ssh_auth` / `ssh_sign` verification.
func TestEnrollResponseApprovedSshSkFlagsRoundTrip(t *testing.T) {
	t.Parallel()

	flags := 5 // 0x05 = user presence + user verification
	in := MailboxEnrollResponseApprovedV1{
		Status:       MailboxEnrollResponseApprovedV1StatusApproved,
		Id:           "550e8400-e29b-41d4-a716-446655440000",
		PublicKeyHex: "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		DeviceKeyId:  "dev-1",
		Algorithm:    "ed25519",
		SshSkFlags:   &flags,
	}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"ssh_sk_flags":5`) {
		t.Errorf("expected snake_case ssh_sk_flags in %s", out)
	}

	var rt MailboxEnrollResponseApprovedV1
	if err := json.Unmarshal(out, &rt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rt.SshSkFlags == nil {
		t.Fatalf("SshSkFlags lost on round-trip")
	}
	if *rt.SshSkFlags != 5 {
		t.Errorf("SshSkFlags = %d, want 5", *rt.SshSkFlags)
	}

	// Non-SSH enrollments omit the field entirely; verify nil round-trips.
	in.SshSkFlags = nil
	out, err = json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal without ssh_sk_flags: %v", err)
	}
	if strings.Contains(string(out), `"ssh_sk_flags"`) {
		t.Errorf("ssh_sk_flags must be omitempty when absent: %s", out)
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
		"algorithm": "ed25519",
		"ssh_sk_flags": 1
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
	if approved.SshSkFlags == nil || *approved.SshSkFlags != 1 {
		t.Errorf("SshSkFlags = %v, want *int=1", approved.SshSkFlags)
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
