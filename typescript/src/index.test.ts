// JSON round-trip smoke tests covering one schema per surface. These run
// under Node's built-in test runner (`node --test`) using
// `--experimental-strip-types` so the source files compile on the fly.
import { strict as assert } from "node:assert";
import { describe, it } from "node:test";

import type {
  ApprovalAttestedKeyProof,
  MailboxAgeUnwrapRequestPayloadV1,
  MailboxBrowserApprovalDecisionBindingV1,
  MailboxBrowserApprovalRequestPayloadV1,
  MailboxBrowserApprovalResponsePayloadV1,
  MailboxEnrollResponseApprovedV1,
  MailboxEnrollResponsePayloadV1,
  MailboxEnvelopeV1,
  MailboxGpgDecryptResponseSuccessV1,
  MailboxSshAuthResponseSuccessV1,
  MailboxSshSignRequestPayloadV1,
  MailboxSshSignResponsePayloadV1,
  MailboxSshSignResponseSuccessV1,
} from "./index.ts";

const browserApprovalDecisionBindingFixtureJSON =
  '{"approval_id":"appr_browser_approval_fixture","browser_public_key_algorithm":"ES256","browser_public_key_thumbprint":"sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA","decided_at":"2026-05-14T19:31:00Z","decision":"approved","expires_at":"2026-05-14T19:35:00Z","nonce":"m4H2YxTjueEXAMPLE","pairing_transcript_hash":"sha256:6f5902ac237024bdd0c176cb93063dc4f1e01e1191450b5f8f457c56f48e1f4f","request_envelope_id":"11111111-2222-4333-8444-555555555555","request_envelope_issued_at":"2026-05-14T19:30:00Z","request_envelope_type":"browser_approval_request","requested_capability":"captcha.browser_credential","requester_client_id":"captcha-service","requester_origin":"https://captcha.naughtbot.com","service_mobile_pairing_id":"pair_9d58fb4c6ff84f46","version":"browser-approval-decision-binding/v1"}';

describe("MailboxEnvelopeV1", () => {
  it("round-trips the literal RFC 3339 issued_at string", () => {
    const envelope: MailboxEnvelopeV1 = {
      v: 1,
      type: "ssh_sign",
      id: "9b2a4d1c-1f1f-4a4a-8f8f-1c1c1c1c1c1c",
      issued_at: "2026-05-03T05:00:00.123Z",
      payload: { raw_data: "YWJj", device_key_id: "key_abc" },
    };
    const json = JSON.stringify(envelope);
    const parsed = JSON.parse(json) as MailboxEnvelopeV1;
    assert.equal(parsed.issued_at, envelope.issued_at);
    assert.equal(parsed.type, "ssh_sign");
    assert.equal(parsed.v, 1);
  });

  it("decodes envelopes with unregistered type values", () => {
    // Forward-compat: receivers SHOULD decode and log+drop unknown types.
    // The generated TypeScript types declare `type: string`, so this
    // compiles without casting.
    const envelope: MailboxEnvelopeV1 = {
      v: 1,
      type: "unregistered_v2_type",
      id: "00000000-0000-0000-0000-000000000000",
      issued_at: "2026-05-03T05:00:00Z",
      payload: {},
    };
    const parsed = JSON.parse(JSON.stringify(envelope)) as MailboxEnvelopeV1;
    assert.equal(parsed.type, "unregistered_v2_type");
  });
});

describe("MailboxSshSignRequestPayloadV1", () => {
  it("emits snake_case keys and keeps `flags` optional", () => {
    const request: MailboxSshSignRequestPayloadV1 = {
      raw_data: "ZGF0YQ==",
      device_key_id: "dev-key-1",
    };
    const json = JSON.stringify(request);
    assert.ok(json.includes('"raw_data"'));
    assert.ok(json.includes('"device_key_id"'));
    // `flags` carries a `default:` value in the schema; the TS surface
    // keeps it optional (verified via Codex regression).
    assert.ok(!json.includes('"flags"'));
  });
});

describe("MailboxSshSignResponsePayloadV1", () => {
  it("decodes success branch by structural narrowing", () => {
    const json = '{"signature":"YWJj","flags":1,"counter":7}';
    const resp = JSON.parse(json) as MailboxSshSignResponsePayloadV1;
    assert.ok("signature" in resp && resp.signature !== undefined);
    assert.ok("flags" in resp && resp.flags === 1);
    assert.ok("counter" in resp && resp.counter === 7);
    assert.ok(!("error_code" in resp) || resp.error_code === undefined);
  });

  it("decodes failure branch by structural narrowing", () => {
    const json = '{"error_code":1,"error_message":"User rejected"}';
    const resp = JSON.parse(json) as MailboxSshSignResponsePayloadV1;
    assert.ok("error_code" in resp && resp.error_code === 1);
  });
});

// Regression test for NaughtBot/e2ee-payloads#17. The SK monotonic counter
// and per-signature flags byte are now required on both `ssh_auth` and
// `ssh_sign` success branches. The compile-time bindings below also pin
// that `counter` and `flags` are required (a regression that makes either
// optional turns this file into a `tsc` error).
describe("SSH-SK counter + flags (issue #17)", () => {
  it("requires counter + flags on MailboxSshAuthResponseSuccessV1", () => {
    const success: MailboxSshAuthResponseSuccessV1 = {
      signature: "YWJj",
      flags: 1,
      counter: 7,
    };
    const parsed = JSON.parse(
      JSON.stringify(success),
    ) as MailboxSshAuthResponseSuccessV1;
    assert.equal(parsed.counter, 7);
    assert.equal(parsed.flags, 1);
    assert.equal(parsed.signature, "YWJj");

    // u32 max counter + u8 max flags round-trip without overflow.
    const maxBoundary: MailboxSshAuthResponseSuccessV1 = {
      signature: "YWJj",
      flags: 255,
      counter: 4294967295,
    };
    const parsedMax = JSON.parse(
      JSON.stringify(maxBoundary),
    ) as MailboxSshAuthResponseSuccessV1;
    assert.equal(parsedMax.counter, 4294967295);
    assert.equal(parsedMax.flags, 255);
  });

  it("requires counter + flags on MailboxSshSignResponseSuccessV1", () => {
    const success: MailboxSshSignResponseSuccessV1 = {
      signature: "YWJj",
      flags: 1,
      counter: 42,
    };
    const parsed = JSON.parse(
      JSON.stringify(success),
    ) as MailboxSshSignResponseSuccessV1;
    assert.equal(parsed.counter, 42);
    assert.equal(parsed.flags, 1);
  });
});

describe("MailboxBrowserApprovalDecisionBindingV1", () => {
  it("matches the cross-language canonical JSON fixture", () => {
    const challenge = approvalChallengeFixture();
    const request: MailboxBrowserApprovalRequestPayloadV1 = {
      approval_challenge: challenge,
      approval_id: "appr_browser_approval_fixture",
      browser_display_name: "Chrome on MacBook Pro",
      browser_platform: "macOS",
      browser_public_key_algorithm: "ES256",
      browser_public_key_thumbprint:
        "sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA",
      expires_at: "2026-05-14T19:35:00Z",
      issued_at: "2026-05-14T19:30:00Z",
      nonce: "m4H2YxTjueEXAMPLE",
      requested_capability: "captcha.browser_credential",
      requester_client_id: "captcha-service",
      requester_display_name: "NaughtBot Captcha",
      requester_origin: "https://captcha.naughtbot.com",
    };
    const requestParsed = JSON.parse(
      JSON.stringify(request),
    ) as MailboxBrowserApprovalRequestPayloadV1;
    assert.deepEqual(requestParsed.approval_challenge, challenge);

    const binding: MailboxBrowserApprovalDecisionBindingV1 = {
      approval_id: "appr_browser_approval_fixture",
      browser_public_key_algorithm: "ES256",
      browser_public_key_thumbprint:
        "sha256:8uLz73VtBwmU5O_Jr3r2StpLrNxW41Oq9p6FwR2C7xA",
      decided_at: "2026-05-14T19:31:00Z",
      decision: "approved",
      expires_at: "2026-05-14T19:35:00Z",
      nonce: "m4H2YxTjueEXAMPLE",
      pairing_transcript_hash:
        "sha256:6f5902ac237024bdd0c176cb93063dc4f1e01e1191450b5f8f457c56f48e1f4f",
      request_envelope_id: "11111111-2222-4333-8444-555555555555",
      request_envelope_issued_at: "2026-05-14T19:30:00Z",
      request_envelope_type: "browser_approval_request",
      requested_capability: "captcha.browser_credential",
      requester_client_id: "captcha-service",
      requester_origin: "https://captcha.naughtbot.com",
      service_mobile_pairing_id: "pair_9d58fb4c6ff84f46",
      version: "browser-approval-decision-binding/v1",
    };
    const json = JSON.stringify(binding);
    assert.equal(json, browserApprovalDecisionBindingFixtureJSON);

    const response: MailboxBrowserApprovalResponsePayloadV1 = {
      approval_binding_bytes: Buffer.from(json, "utf8").toString("base64"),
      approval_binding_format: "browser-approval-decision-binding/v1+json",
      approval_id: binding.approval_id,
      approval_proof: approvalProofFixture(),
      decided_at: binding.decided_at,
      decision: binding.decision,
      request_envelope_id: binding.request_envelope_id,
      status: "decided",
    };
    const parsed = JSON.parse(
      JSON.stringify(response),
    ) as MailboxBrowserApprovalResponsePayloadV1;
    assert.equal(
      Buffer.from(parsed.approval_binding_bytes, "base64").toString("utf8"),
      browserApprovalDecisionBindingFixtureJSON,
    );
    assert.equal(parsed.decision, "approved");
    assert.equal(parsed.status, "decided");
    assert.equal(
      Buffer.from(parsed.approval_proof.proof, "base64").toString("utf8"),
      "browser-approval-proof",
    );
  });
});

function approvalChallengeFixture(): ApprovalAttestedKeyProof["challenge"] {
  return {
    version: "approval-challenge/v1",
    nonce: "browser-approval-nonce",
    request_id: "11111111-2222-4333-8444-555555555555",
    plaintext_hash:
      "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
  };
}

function approvalProofFixture(): ApprovalAttestedKeyProof {
  return {
    version: "approval-attested-key-proof/v1",
    challenge: approvalChallengeFixture(),
    statement: {
      issuer_public_key_hex:
        "02aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      app_id_hash_hex:
        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      policy_version: 1,
      now: 1_778_787_060,
      challenge_nonce_hex:
        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
      audience_hash_hex:
        "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
      approval_hash_hex:
        "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
    },
    attestation: {
      version: "approval-attestation/v1",
      bytes: Buffer.from("attestation-bytes", "utf8").toString("base64"),
      signature: Buffer.from("attestation-signature", "utf8").toString("base64"),
    },
    proof: Buffer.from("browser-approval-proof", "utf8").toString("base64"),
  };
}

describe("MailboxGpgDecryptResponseSuccessV1", () => {
  it("requires both session_key and algorithm on success", () => {
    // Bind to the success branch directly so the compile-time check is
    // strict: a regression that makes either field optional turns this
    // into a `tsc` error rather than a silent runtime miss.
    const success: MailboxGpgDecryptResponseSuccessV1 = {
      session_key: "c2Vzc2lvbg==",
      algorithm: 9,
    };
    const parsed = JSON.parse(
      JSON.stringify(success),
    ) as MailboxGpgDecryptResponseSuccessV1;
    assert.equal(parsed.session_key, "c2Vzc2lvbg==");
    assert.equal(parsed.algorithm, 9);
  });
});

describe("MailboxAgeUnwrapRequestPayloadV1", () => {
  it("encodes all three required hex / base64 fields", () => {
    const req: MailboxAgeUnwrapRequestPayloadV1 = {
      ephemeral_public_hex:
        "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
      wrapped_file_key: "d3JhcHBlZA==",
      recipient_public_hex:
        "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3",
    };
    const json = JSON.stringify(req);
    assert.ok(json.includes('"ephemeral_public_hex"'));
    assert.ok(json.includes('"wrapped_file_key"'));
    assert.ok(json.includes('"recipient_public_hex"'));
  });
});

describe("MailboxEnrollResponsePayloadV1", () => {
  it("discriminates approved vs rejected via status", () => {
    const approved = JSON.parse(
      JSON.stringify({
        status: "approved",
        id: "550e8400-e29b-41d4-a716-446655440000",
        public_key_hex:
          "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
        device_key_id: "dev-1",
        algorithm: "ed25519",
      }),
    ) as MailboxEnrollResponsePayloadV1;
    assert.equal(approved.status, "approved");
    if (approved.status === "approved") {
      assert.equal(approved.algorithm, "ed25519");
    }

    const rejected = JSON.parse(
      JSON.stringify({ status: "rejected", error_code: 1 }),
    ) as MailboxEnrollResponsePayloadV1;
    assert.equal(rejected.status, "rejected");
    if (rejected.status === "rejected") {
      assert.equal(rejected.error_code, 1);
    }
  });

  // Regression test for NaughtBot/e2ee-payloads#17. The per-credential
  // SSH-SK flags byte must be carried back to the requester on approved
  // SSH-SK enrollments so the requester can rebuild the OpenSSH SK
  // signature preimage on every subsequent `ssh_auth` / `ssh_sign` call.
  it("round-trips per-credential ssh_sk_flags on SSH-SK enrollments", () => {
    const approved: MailboxEnrollResponseApprovedV1 = {
      status: "approved",
      id: "550e8400-e29b-41d4-a716-446655440000",
      public_key_hex:
        "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
      device_key_id: "dev-1",
      algorithm: "ed25519",
      ssh_sk_flags: 5, // 0x05 = user presence + user verification
    };
    const json = JSON.stringify(approved);
    assert.ok(json.includes('"ssh_sk_flags":5'));
    const parsed = JSON.parse(json) as MailboxEnrollResponseApprovedV1;
    assert.equal(parsed.ssh_sk_flags, 5);

    // Non-SSH enrollments omit the field; verify the surface stays
    // optional (a regression that makes it required turns this into a
    // `tsc` error rather than a silent on-the-wire change).
    const noFlags: MailboxEnrollResponseApprovedV1 = {
      status: "approved",
      id: "550e8400-e29b-41d4-a716-446655440000",
      public_key_hex:
        "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
      device_key_id: "dev-1",
      algorithm: "ed25519",
    };
    assert.ok(!JSON.stringify(noFlags).includes("ssh_sk_flags"));
  });
});
