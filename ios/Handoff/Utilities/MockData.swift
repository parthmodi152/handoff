import Foundation

/// Mock data for SwiftUI Previews only.
enum MockData {

    // MARK: - Config JSON strings

    static let singleSelectConfigJSON = """
    {"options":[{"value":"approve","label":"Approve"},{"value":"reject","label":"Reject"},{"value":"escalate","label":"Escalate"}]}
    """

    static let multiSelectConfigJSON = """
    {"options":[{"value":"spam","label":"Spam"},{"value":"misleading","label":"Misleading"},{"value":"harassment","label":"Harassment"},{"value":"violence","label":"Violence"}],"min_selections":1,"max_selections":3}
    """

    static let ratingConfigJSON = """
    {"scale_min":1,"scale_max":5,"scale_step":1,"labels":{"1":"Poor","5":"Excellent"}}
    """

    static let numberConfigJSON = """
    {"min_value":0,"max_value":200,"decimal_places":0}
    """

    static let textConfigJSON = """
    {"placeholder":"Enter your response...","min_length":1,"max_length":500}
    """

    static let booleanConfigJSON = """
    {"true_label":"Remove","false_label":"Keep"}
    """

    // MARK: - Helper

    private static func jsonConfig(_ json: String) -> AnyCodable? {
        guard let data = json.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data),
              let reData = try? JSONSerialization.data(withJSONObject: obj) else { return nil }
        return try? JSONDecoder().decode(AnyCodable.self, from: reData)
    }

    // MARK: - Mock Requests

    static let singleSelectRequest = APIRequest(
        id: "req_001",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "high",
        title: "Review Flagged Post",
        requestText: "Review flagged user post for policy violation",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "single_select",
        responseConfig: jsonConfig(singleSelectConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 3600,
        timeoutAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(3600)),
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now),
        updatedAt: ISO8601DateFormatter().string(from: Date.now)
    )

    static let multiSelectRequest = APIRequest(
        id: "req_002",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "deferred",
        contentType: "markdown",
        priority: "medium",
        title: "Flag Content Violations",
        requestText: "Select all applicable content flags",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "multi_select",
        responseConfig: jsonConfig(multiSelectConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: nil,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now),
        updatedAt: ISO8601DateFormatter().string(from: Date.now)
    )

    static let ratingRequest = APIRequest(
        id: "req_003",
        loopId: "loop_002",
        loopName: "Refunds",
        creatorId: "usr_001",
        processingType: "deferred",
        contentType: "markdown",
        priority: "low",
        title: "Rate AI Summary Quality",
        requestText: "Rate the quality of this AI-generated summary",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "rating",
        responseConfig: jsonConfig(ratingConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: nil,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now),
        updatedAt: ISO8601DateFormatter().string(from: Date.now)
    )

    static let numberRequest = APIRequest(
        id: "req_004",
        loopId: "loop_002",
        loopName: "Refunds",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "high",
        title: "President of Ghana Age",
        requestText: "What is the age of the President of Ghana?",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "number",
        responseConfig: jsonConfig(numberConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 300,
        timeoutAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(300)),
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now),
        updatedAt: ISO8601DateFormatter().string(from: Date.now)
    )

    static let textRequest = APIRequest(
        id: "req_005",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "deferred",
        contentType: "markdown",
        priority: "medium",
        title: "US President Name",
        requestText: "What is the name of the President of United States?",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "text",
        responseConfig: jsonConfig(textConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: nil,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now),
        updatedAt: ISO8601DateFormatter().string(from: Date.now)
    )

    static let completedRequest = APIRequest(
        id: "req_006",
        loopId: "loop_002",
        loopName: "Refund Request Loop",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "high",
        title: "Refund Request #289",
        requestText: "Customer Rex Asare with ID: 289 wants a refund of amount $500\n\nReason for refund is the item stopped working after charging it.",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "single_select",
        responseConfig: jsonConfig(singleSelectConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 3600,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "completed",
        responseData: nil,
        responseBy: "usr_002",
        responseAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-3600)),
        responseTimeSeconds: 45.0,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-7200)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-3600))
    )

    static let timedOutRequest = APIRequest(
        id: "req_007",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "critical",
        title: "Review Hate Speech Report",
        requestText: "Urgent: Review reported comment for hate speech",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "boolean",
        responseConfig: jsonConfig(booleanConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 3600,
        timeoutAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-600)),
        callbackUrl: nil,
        status: "timeout",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-3600)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-600))
    )

    static let booleanRequest = APIRequest(
        id: "req_008",
        loopId: "loop_003",
        loopName: "DevOps",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "critical",
        title: "Deploy v2.4.1 to Production",
        requestText: "Deploy v2.4.1 to production? All staging tests passed.",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "boolean",
        responseConfig: jsonConfig(booleanConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 900,
        timeoutAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(900)),
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-120)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-120))
    )

    static let cancelledRequest = APIRequest(
        id: "req_009",
        loopId: "loop_003",
        loopName: "DevOps",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "markdown",
        priority: "medium",
        title: "Scale Worker Pool",
        requestText: "Scale up worker pool to 8 instances for Black Friday traffic",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "boolean",
        responseConfig: jsonConfig(booleanConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 3600,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "cancelled",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-10800)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-9000))
    )

    static let completedTextRequest = APIRequest(
        id: "req_010",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "deferred",
        contentType: "markdown",
        priority: "low",
        title: "Summarize Q4 Feedback",
        requestText: "Summarize the customer feedback from Q4 report",
        imageUrl: nil,
        context: nil,
        platform: "api",
        responseType: "text",
        responseConfig: jsonConfig(textConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: nil,
        timeoutAt: nil,
        callbackUrl: nil,
        status: "completed",
        responseData: nil,
        responseBy: "usr_003",
        responseAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-1800)),
        responseTimeSeconds: 120.0,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-5400)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-1800))
    )

    static let imageRequest = APIRequest(
        id: "req_011",
        loopId: "loop_001",
        loopName: "Content Review",
        creatorId: "usr_001",
        processingType: "time-sensitive",
        contentType: "image",
        priority: "high",
        title: "Review Uploaded Screenshot",
        requestText: "Does this screenshot contain **personally identifiable information** (PII) that should be redacted before publishing?",
        imageUrl: "https://picsum.photos/800/600",
        context: nil,
        platform: "api",
        responseType: "boolean",
        responseConfig: jsonConfig(booleanConfigJSON),
        defaultResponse: nil,
        timeoutSeconds: 1800,
        timeoutAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(1800)),
        callbackUrl: nil,
        status: "pending",
        responseData: nil,
        responseBy: nil,
        responseAt: nil,
        responseTimeSeconds: nil,
        createdAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-60)),
        updatedAt: ISO8601DateFormatter().string(from: Date.now.addingTimeInterval(-60))
    )

    // MARK: - Collections

    static var allPendingRequests: [APIRequest] {
        [booleanRequest, singleSelectRequest, multiSelectRequest, numberRequest, ratingRequest, textRequest, imageRequest]
    }

    static var allHistoryRequests: [APIRequest] {
        [completedRequest, completedTextRequest, timedOutRequest, cancelledRequest]
    }

    // MARK: - Preview Helpers

    static func previewRequestsManager() -> RequestsManager {
        let manager = RequestsManager()
        manager.pendingRequests = allPendingRequests
        manager.historyRequests = allHistoryRequests
        return manager
    }

    static func previewAuthService() -> AuthService {
        let service = AuthService()
        service.isAuthenticated = true
        service.currentUser = APIUser(
            id: "usr_preview",
            email: "parth@hitl.sh",
            name: "Parth Modi",
            avatarUrl: nil,
            oauthProvider: "apple",
            createdAt: ISO8601DateFormatter().string(from: Date.now),
            updatedAt: nil
        )
        return service
    }

    static func previewLoopsManager() -> LoopsManager {
        let manager = LoopsManager()
        manager.loops = [
            APILoop(id: "loop_001", name: "Content Review", description: "Review flagged content", icon: nil, creatorId: "usr_001", inviteCode: "ABC123", inviteQr: nil, memberCount: 5, role: "admin", createdAt: nil, updatedAt: nil),
            APILoop(id: "loop_002", name: "Refunds", description: "Process refund requests", icon: nil, creatorId: "usr_001", inviteCode: "DEF456", inviteQr: nil, memberCount: 3, role: "member", createdAt: nil, updatedAt: nil),
            APILoop(id: "loop_003", name: "DevOps", description: "Infrastructure approvals", icon: nil, creatorId: "usr_002", inviteCode: "GHI789", inviteQr: nil, memberCount: 8, role: "member", createdAt: nil, updatedAt: nil),
        ]
        return manager
    }
}
