import Foundation

extension APIRequest {
    var loopNameDisplay: String {
        loopName ?? "Unknown Loop"
    }

    var responseConfigJSON: String {
        responseConfig?.jsonString ?? "{}"
    }

    var responseDataJSON: String? {
        responseData?.jsonString
    }

    var timeoutDate: Date? {
        guard let timeoutAt else { return nil }
        return ISO8601DateFormatter().date(from: timeoutAt)
    }

    var isExpired: Bool {
        guard let timeout = timeoutDate else { return false }
        return Date.now >= timeout
    }

    var timeRemaining: TimeInterval? {
        guard let timeout = timeoutDate else { return nil }
        let remaining = timeout.timeIntervalSince(Date.now)
        return remaining > 0 ? remaining : 0
    }

    var isPending: Bool {
        status == "pending" && !isExpired
    }

    var responseTypeLabel: String {
        switch responseType {
        case "single_select": return "Single Choice"
        case "multi_select": return "Multiple Choice"
        case "text": return "Text"
        case "rating": return "Rating"
        case "number": return "Number"
        case "boolean": return "Boolean"
        default: return responseType
        }
    }

    var responseByName: String? {
        // The backend returns response_by as a UUID string, not a name.
        // The detailed respond endpoint returns response_by as an object with name.
        // For list views we don't have this — return nil.
        nil
    }

    var responseAtDate: Date? {
        guard let responseAt else { return nil }
        return ISO8601DateFormatter().date(from: responseAt)
    }

    var createdAtDate: Date {
        ISO8601DateFormatter().date(from: createdAt) ?? Date()
    }

    /// Short display text — prefers title over requestText, matching the backend's displayText helper.
    func displayText(maxLength: Int = 60) -> String {
        let source = title.isEmpty ? requestText : title
        if source.count <= maxLength { return source }
        return String(source.prefix(maxLength - 1)) + "…"
    }
}
