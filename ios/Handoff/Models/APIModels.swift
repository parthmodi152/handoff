import Foundation

// MARK: - API Response Envelope

struct APIResponse<T: Decodable>: Decodable {
    let error: Bool
    let msg: String
    let data: T?
}

// MARK: - Auth Responses

struct AuthCallbackData: Decodable, Sendable {
    let token: String
    let user: APIUser
}

struct AuthMeData: Decodable, Sendable {
    let user: APIUser
}

struct AppleNativeAuthData: Decodable, Sendable {
    let token: String
    let user: APIUser
}

// MARK: - User

struct APIUser: Decodable, Sendable {
    let id: String
    let email: String
    let name: String
    let avatarUrl: String?
    let oauthProvider: String?
    let createdAt: String
    let updatedAt: String?
}

// MARK: - Loop

struct APILoop: Decodable, Identifiable, Sendable {
    let id: String
    let name: String
    let description: String?
    let icon: String?
    let creatorId: String?
    let inviteCode: String?
    let inviteQr: String?
    let memberCount: Int?
    let role: String?
    let createdAt: String?
    let updatedAt: String?
}

struct LoopListData: Decodable, Sendable {
    let loops: [APILoop]
    let count: Int
}

struct LoopDetailData: Decodable, Sendable {
    let loop: APILoop
    let members: [APILoopMember]?
}

struct LoopCreateData: Decodable, Sendable {
    let loop: APILoop
}

struct LoopJoinData: Decodable, Sendable {
    let loop: APILoop
}

struct APILoopMember: Decodable, Identifiable, Sendable {
    let userId: String
    let email: String
    let name: String
    let avatarUrl: String?
    let role: String
    let status: String
    let joinedAt: String

    var id: String { userId }
}

// MARK: - Request

struct APIRequest: Decodable, Identifiable, Sendable {
    let id: String
    let loopId: String
    let loopName: String?
    let creatorId: String?
    let processingType: String
    let contentType: String?
    let priority: String
    let title: String
    let requestText: String
    let imageUrl: String?
    let context: AnyCodable?
    let platform: String?
    let responseType: String
    let responseConfig: AnyCodable?
    let defaultResponse: AnyCodable?
    let timeoutSeconds: Int?
    let timeoutAt: String?
    let callbackUrl: String?
    let status: String
    let responseData: AnyCodable?
    let responseBy: String?
    let responseAt: String?
    let responseTimeSeconds: Double?
    let createdAt: String
    let updatedAt: String?
}

struct RequestListData: Decodable, Sendable {
    let requests: [APIRequest]
    let count: Int
    let total: Int?
    let hasMore: Bool?
    let pagination: PaginationInfo?
}

struct PaginationInfo: Decodable, Sendable {
    let limit: Int
    let offset: Int
    let nextOffset: Int?
}

struct RequestDetailData: Decodable, Sendable {
    let request: APIRequest
}

struct RequestCreateData: Decodable, Sendable {
    let requestId: String
    let status: String
    let timeoutAt: String?
    let broadcastedTo: Int?
    let notificationsSent: Int?
}

struct RequestRespondData: Decodable, Sendable {
    let request: RequestRespondInfo
}

struct RequestRespondInfo: Decodable, Sendable {
    let id: String
    let status: String
    let responseData: AnyCodable?
    let responseBy: ResponseByInfo?
    let responseAt: String?
    let responseTimeSeconds: Double?
}

struct ResponseByInfo: Decodable, Sendable {
    let userId: String
    let name: String
    let email: String
}

struct RequestCancelData: Decodable, Sendable {
    let request: RequestCancelInfo
}

struct RequestCancelInfo: Decodable, Sendable {
    let id: String
    let status: String
    let updatedAt: String?
}

// MARK: - API Keys

struct APIKeyCreateData: Decodable, Sendable {
    let apiKey: APIKeyFull
}

struct APIKeyFull: Decodable, Identifiable, Sendable {
    let id: String
    let name: String
    let key: String?
    let keyPrefix: String
    let permissions: [String]
    let isActive: Bool
    let lastUsedAt: String?
    let createdAt: String
    let expiresAt: String?
}

struct APIKeyListData: Decodable, Sendable {
    let apiKeys: [APIKeyFull]
    let count: Int
}

// MARK: - AnyCodable (for arbitrary JSON)

struct AnyCodable: Decodable, @unchecked Sendable {
    let value: Any

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            value = NSNull()
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map { $0.value }
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues { $0.value }
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON type")
        }
    }

    /// Convert back to JSON string for local storage
    var jsonString: String? {
        guard let data = try? JSONSerialization.data(withJSONObject: value),
              let str = String(data: data, encoding: .utf8) else { return nil }
        return str
    }
}
