import Foundation
import Testing

@testable import NaughtBotE2EEPayloads

// Placeholder so the SwiftPM test target compiles before WS1.5 lands the
// real round-trip smoke tests. Use a value-based assertion to keep the
// expression non-trivial so swift-testing doesn't warn the macro is
// always true.
@Test
func generatedTypesAreImportable() {
    let envelope = Components.Schemas.MailboxEnvelopeV1(
        v: ._1,
        _type: .ssh_sign,
        id: "00000000-0000-0000-0000-000000000000",
        issued_at: Date(timeIntervalSince1970: 0),
        payload: .init(additionalProperties: .init())
    )
    #expect(envelope.id == "00000000-0000-0000-0000-000000000000")
    #expect(envelope._type == .ssh_sign)
}
