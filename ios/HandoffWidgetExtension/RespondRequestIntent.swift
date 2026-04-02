//
//  RespondRequestIntent.swift
//  HandoffWidgetExtension
//
//  LiveActivityIntent conformances for inline response buttons
//  in Live Activities. These run in the app's process.
//

import AppIntents
import Foundation
import OSLog
import Security

private let logger = Logger(subsystem: "sh.hitl.handoff", category: "RespondIntent")

// MARK: - Respond Boolean

/// Intent for responding to a boolean request from a Live Activity button.
struct RespondBooleanIntent: LiveActivityIntent {
    static var title: LocalizedStringResource = "Respond Boolean"

    @Parameter(title: "Request ID")
    var requestId: String

    @Parameter(title: "Value")
    var value: Bool

    init() {}

    init(requestId: String, value: Bool) {
        self.requestId = requestId
        self.value = value
    }

    func perform() async throws -> some IntentResult {
        logger.info("[RespondBoolean] perform() called — requestId=\(requestId) value=\(value)")
        do {
            try await respondToRequest(id: requestId, responseData: value)
            logger.info("[RespondBoolean] Success for requestId=\(requestId)")
        } catch {
            logger.error("[RespondBoolean] Failed for requestId=\(requestId): \(error.localizedDescription)")
            throw error
        }
        return .result()
    }
}

// MARK: - Respond Select

/// Intent for responding to a single-select request from a Live Activity button.
struct RespondSelectIntent: LiveActivityIntent {
    static var title: LocalizedStringResource = "Respond Select"

    @Parameter(title: "Request ID")
    var requestId: String

    @Parameter(title: "Selected Value")
    var selectedValue: String

    init() {}

    init(requestId: String, selectedValue: String) {
        self.requestId = requestId
        self.selectedValue = selectedValue
    }

    func perform() async throws -> some IntentResult {
        logger.info("[RespondSelect] perform() called — requestId=\(requestId) selectedValue=\(selectedValue)")
        do {
            try await respondToRequest(id: requestId, responseData: selectedValue)
            logger.info("[RespondSelect] Success for requestId=\(requestId)")
        } catch {
            logger.error("[RespondSelect] Failed for requestId=\(requestId): \(error.localizedDescription)")
            throw error
        }
        return .result()
    }
}

// MARK: - Shared API Call

/// Lightweight HTTP call for responding to requests from Live Activity buttons.
/// Uses direct URLSession rather than APIClient to avoid cross-target dependencies.
private func respondToRequest(id: String, responseData: Any) async throws {
    #if DEBUG
    let baseURL = "https://macbook-pro-5.tail59d3a5.ts.net"
    #else
    let baseURL = "https://api.hitl.sh"
    #endif

    let urlString = "\(baseURL)/api/v1/requests/\(id)/respond"
    logger.info("[RespondIntent] URL: \(urlString)")

    guard let url = URL(string: urlString) else {
        logger.error("[RespondIntent] Invalid URL: \(urlString)")
        return
    }

    // Read auth token from Keychain directly
    let service = "sh.hitl.handoff"
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: service,
        kSecAttrAccount as String: "auth_token",
        kSecReturnData as String: true,
        kSecMatchLimit as String: kSecMatchLimitOne
    ]
    var result: AnyObject?
    let status = SecItemCopyMatching(query as CFDictionary, &result)

    guard status == errSecSuccess,
          let data = result as? Data,
          let token = String(data: data, encoding: .utf8) else {
        logger.error("[RespondIntent] Keychain read failed — status=\(status) (errSecItemNotFound=\(errSecItemNotFound))")
        return
    }
    logger.info("[RespondIntent] Auth token loaded (\(token.prefix(8))...)")

    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")

    let body = ["response_data": responseData]
    request.httpBody = try JSONSerialization.data(withJSONObject: body)
    logger.info("[RespondIntent] Sending POST with body: \(String(describing: body))")

    let (responseData2, response) = try await URLSession.shared.data(for: request)

    if let httpResponse = response as? HTTPURLResponse {
        let bodyStr = String(data: responseData2, encoding: .utf8) ?? "<non-utf8>"
        logger.info("[RespondIntent] Response status=\(httpResponse.statusCode) body=\(bodyStr.prefix(200))")
    }
}
