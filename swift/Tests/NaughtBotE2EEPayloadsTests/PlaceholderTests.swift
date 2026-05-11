import Foundation
import Testing

@testable import NaughtBotE2EEPayloads

// Placeholder so the SwiftPM test target compiles before WS1.5 lands the
// real round-trip smoke tests for every binding.
@Test
func generatedTypesAreImportable() {
    let envelope = Components.Schemas.MailboxEnvelopeV1(
        v: ._1,
        _type: "ssh_sign",
        id: "00000000-0000-0000-0000-000000000000",
        issued_at: "2026-01-01T00:00:00Z",
        payload: .init(additionalProperties: .init())
    )
    #expect(envelope.id == "00000000-0000-0000-0000-000000000000")
    #expect(envelope._type == "ssh_sign")
    #expect(envelope.issued_at == "2026-01-01T00:00:00Z")
}
