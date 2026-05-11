// JSON round-trip smoke tests covering one schema per surface. These run
// under Node's built-in test runner (`node --test`) using
// `--experimental-strip-types` so the source files compile on the fly.
import { strict as assert } from "node:assert";
import { describe, it } from "node:test";

import type {
  MailboxAgeUnwrapRequestPayloadV1,
  MailboxEnrollResponsePayloadV1,
  MailboxEnvelopeV1,
  MailboxGpgDecryptResponseSuccessV1,
  MailboxSshSignRequestPayloadV1,
  MailboxSshSignResponsePayloadV1,
} from "./index.ts";

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
    const json = '{"signature":"YWJj"}';
    const resp = JSON.parse(json) as MailboxSshSignResponsePayloadV1;
    assert.ok("signature" in resp && resp.signature !== undefined);
    assert.ok(!("error_code" in resp) || resp.error_code === undefined);
  });

  it("decodes failure branch by structural narrowing", () => {
    const json = '{"error_code":1,"error_message":"User rejected"}';
    const resp = JSON.parse(json) as MailboxSshSignResponsePayloadV1;
    assert.ok("error_code" in resp && resp.error_code === 1);
  });
});

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
});
