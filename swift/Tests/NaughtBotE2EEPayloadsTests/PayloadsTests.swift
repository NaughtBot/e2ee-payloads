import Foundation
import Testing

@testable import NaughtBotE2EEPayloads

// JSON round-trip smoke tests covering one schema per surface. The Swift
// generator emits `Codable` types and swift-openapi-runtime carries the
// helpers required to decode them with the standard JSONDecoder.

private let decoder: JSONDecoder = JSONDecoder()
private let encoder: JSONEncoder = {
    let e = JSONEncoder()
    e.outputFormatting = [.sortedKeys]
    return e
}()

@Test
func envelopeRoundTrip() throws {
    let json = """
    {
      "v": 1,
      "type": "ssh_sign",
      "id": "9b2a4d1c-1f1f-4a4a-8f8f-1c1c1c1c1c1c",
      "issued_at": "2026-05-03T05:00:00.123Z",
      "payload": {"raw_data":"YWJj","device_key_id":"key_abc"}
    }
    """.data(using: .utf8)!

    let env = try decoder.decode(
        Components.Schemas.MailboxEnvelopeV1.self,
        from: json
    )
    #expect(env.v == ._1)
    #expect(env._type == "ssh_sign")
    #expect(env.id == "9b2a4d1c-1f1f-4a4a-8f8f-1c1c1c1c1c1c")
    #expect(env.issued_at == "2026-05-03T05:00:00.123Z")

    let reEncoded = try encoder.encode(env)
    let again = try decoder.decode(
        Components.Schemas.MailboxEnvelopeV1.self,
        from: reEncoded
    )
    // Crucially, issued_at must round-trip as the literal RFC 3339 string,
    // not as a Foundation.Date that JSONEncoder serialises as a number.
    #expect(again.issued_at == env.issued_at)
}

@Test
func envelopeAcceptsUnknownType() throws {
    // Forward-compat: receivers SHOULD decode the envelope and then
    // log+drop unknown `type` values per the v1 contract. The Swift
    // generator no longer emits a closed enum for `type` so this works.
    let json = """
    {
      "v": 1,
      "type": "unregistered_v2_type",
      "id": "00000000-0000-0000-0000-000000000000",
      "issued_at": "2026-05-03T05:00:00Z",
      "payload": {}
    }
    """.data(using: .utf8)!

    let env = try decoder.decode(
        Components.Schemas.MailboxEnvelopeV1.self,
        from: json
    )
    #expect(env._type == "unregistered_v2_type")
}

@Test
func sshSignResponseSuccessOneOf() throws {
    let json = """
    {"signature":"YWJj","flags":1,"counter":7}
    """.data(using: .utf8)!

    let resp = try decoder.decode(
        Components.Schemas.MailboxSshSignResponsePayloadV1.self,
        from: json
    )
    switch resp {
    case .MailboxSshSignResponseSuccessV1(let success):
        #expect(!success.signature.data.isEmpty)
        #expect(success.flags == 1)
        #expect(success.counter == 7)
    case .MailboxSshSignResponseFailureV1:
        Issue.record("expected success branch, got failure")
    }
}

// Regression test for NaughtBot/e2ee-payloads#17: the SK monotonic counter
// and per-signature assertion flags byte must survive a JSON Codable
// round-trip on the SSH auth success branch. Without these fields the
// requester cannot rebuild the OpenSSH SK signature preimage
// `SHA256(application) || flags || counter || SHA256(data)`.
@Test
func sshAuthResponseSuccessCounterRoundTrip() throws {
    let signatureBytes = Data("ssh-sk-sig".utf8)
    let original = Components.Schemas.MailboxSshAuthResponseSuccessV1(
        signature: .init(signatureBytes),
        flags: 1,
        counter: 7
    )
    let encoded = try encoder.encode(original)
    let encodedString = String(decoding: encoded, as: UTF8.self)
    #expect(encodedString.contains("\"counter\":7"))
    #expect(encodedString.contains("\"flags\":1"))

    let decoded = try decoder.decode(
        Components.Schemas.MailboxSshAuthResponseSuccessV1.self,
        from: encoded
    )
    #expect(decoded.counter == 7)
    #expect(decoded.flags == 1)
    #expect(Data(decoded.signature.data) == signatureBytes)

    // Boundary: u32 max counter and u8 max flags round-trip without overflow.
    let maxOriginal = Components.Schemas.MailboxSshAuthResponseSuccessV1(
        signature: .init(signatureBytes),
        flags: 255,
        counter: 4294967295
    )
    let maxEncoded = try encoder.encode(maxOriginal)
    let maxDecoded = try decoder.decode(
        Components.Schemas.MailboxSshAuthResponseSuccessV1.self,
        from: maxEncoded
    )
    #expect(maxDecoded.counter == 4294967295)
    #expect(maxDecoded.flags == 255)
}

// Regression test for #17: per-credential `ssh_sk_flags` byte on the
// approved enrollment response. Only meaningful for SSH-SK credentials;
// must be omitted on the wire when not present.
@Test
func enrollResponseApprovedSshSkFlagsRoundTrip() throws {
    let original = Components.Schemas.MailboxEnrollResponseApprovedV1(
        status: .approved,
        id: "550e8400-e29b-41d4-a716-446655440000",
        public_key_hex: "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
        device_key_id: "dev-1",
        algorithm: "ed25519",
        ssh_sk_flags: 5
    )
    let encoded = try encoder.encode(original)
    let encodedString = String(decoding: encoded, as: UTF8.self)
    #expect(encodedString.contains("\"ssh_sk_flags\":5"))

    let decoded = try decoder.decode(
        Components.Schemas.MailboxEnrollResponseApprovedV1.self,
        from: encoded
    )
    #expect(decoded.ssh_sk_flags == 5)

    // Non-SSH enrollments omit the field; verify nil round-trips.
    let withoutFlags = Components.Schemas.MailboxEnrollResponseApprovedV1(
        status: .approved,
        id: "550e8400-e29b-41d4-a716-446655440000",
        public_key_hex: "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
        device_key_id: "dev-1",
        algorithm: "ed25519",
        ssh_sk_flags: nil
    )
    let encodedNoFlags = try encoder.encode(withoutFlags)
    let encodedNoFlagsString = String(decoding: encodedNoFlags, as: UTF8.self)
    #expect(!encodedNoFlagsString.contains("ssh_sk_flags"))
}

@Test
func sshSignResponseFailureOneOf() throws {
    let json = """
    {"error_code":1,"error_message":"User rejected"}
    """.data(using: .utf8)!

    let resp = try decoder.decode(
        Components.Schemas.MailboxSshSignResponsePayloadV1.self,
        from: json
    )
    switch resp {
    case .MailboxSshSignResponseSuccessV1:
        Issue.record("expected failure branch, got success")
    case .MailboxSshSignResponseFailureV1(let failure):
        #expect(failure.error_code == ._1)
        #expect(failure.error_message == "User rejected")
    }
}

@Test
func enrollResponseDiscriminatorRoutesByStatus() throws {
    let approvedJSON = """
    {
      "status": "approved",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "public_key_hex": "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
      "device_key_id": "dev-1",
      "algorithm": "ed25519"
    }
    """.data(using: .utf8)!

    let resp = try decoder.decode(
        Components.Schemas.MailboxEnrollResponsePayloadV1.self,
        from: approvedJSON
    )
    switch resp {
    case .approved(let approved):
        #expect(approved.algorithm == "ed25519")
        #expect(approved.public_key_hex == "02a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2")
    case .rejected:
        Issue.record("expected approved branch")
    }
}

@Test
func ageUnwrapRequestRejectsAdditionalProperties() {
    // additionalProperties: false: the generated Swift type calls the
    // `additionalProperties` ensurance helper from swift-openapi-runtime.
    // Decoding a payload with an unknown field MUST throw.
    let json = """
    {
      "ephemeral_public_hex": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
      "wrapped_file_key": "d3JhcHBlZA==",
      "recipient_public_hex": "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3",
      "rogue_extra": "should be rejected"
    }
    """.data(using: .utf8)!

    #expect(throws: (any Error).self) {
        _ = try decoder.decode(
            Components.Schemas.MailboxAgeUnwrapRequestPayloadV1.self,
            from: json
        )
    }
}
