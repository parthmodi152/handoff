import Foundation

// MARK: - API Errors

enum APIError: LocalizedError {
    case invalidURL
    case unauthorized
    case forbidden(String)
    case notFound
    case conflict(String)
    case badRequest(String)
    case serverError(String)
    case networkError(Error)
    case decodingError(Error)

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid URL"
        case .unauthorized: return "Please sign in again"
        case .forbidden(let msg): return msg
        case .notFound: return "Not found"
        case .conflict(let msg): return msg
        case .badRequest(let msg): return msg
        case .serverError(let msg): return msg
        case .networkError(let err): return err.localizedDescription
        case .decodingError(let err): return "Data error: \(err.localizedDescription)"
        }
    }
}

// MARK: - API Client

@MainActor
final class APIClient {
    static let shared = APIClient()

    #if DEBUG
    private var baseURL = "https://macbook-pro-5.tail59d3a5.ts.net"
    #else
    private var baseURL = "https://api.hitl.sh"
    #endif

    private let session: URLSession
    private let decoder: JSONDecoder

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 60
        session = URLSession(configuration: config)

        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
    }

    // MARK: - Token Management

    /// In-memory cache avoids repeated Keychain reads on every request.
    /// SecItemCopyMatching blocks the calling thread (per Apple docs),
    /// so caching the token reduces main thread Keychain access.
    private var cachedToken: String? = nil
    private var tokenLoaded = false

    private var authToken: String? {
        if !tokenLoaded {
            cachedToken = KeychainHelper.read(key: "auth_token")
            tokenLoaded = true
        }
        return cachedToken
    }

    func setAuthToken(_ token: String?) {
        cachedToken = token
        tokenLoaded = true
        if let token {
            KeychainHelper.save(key: "auth_token", value: token)
        } else {
            KeychainHelper.delete(key: "auth_token")
        }
    }

    var isAuthenticated: Bool {
        authToken != nil
    }

    // MARK: - Core Request

    private func request<T: Decodable>(
        method: String,
        path: String,
        body: [String: Any]? = nil,
        queryItems: [URLQueryItem]? = nil
    ) async throws -> T {
        guard var components = URLComponents(string: baseURL + path) else {
            throw APIError.invalidURL
        }
        if let queryItems, !queryItems.isEmpty {
            components.queryItems = queryItems
        }
        guard let url = components.url else {
            throw APIError.invalidURL
        }

        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = method

        if let token = authToken {
            urlRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body {
            urlRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
            urlRequest.httpBody = try JSONSerialization.data(withJSONObject: body)
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: urlRequest)
        } catch {
            throw APIError.networkError(error)
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.serverError("Invalid response")
        }

        // Handle error status codes
        if httpResponse.statusCode == 401 {
            throw APIError.unauthorized
        }

        if httpResponse.statusCode >= 400 {
            // Try to parse error message from body
            let errorMsg: String
            if let envelope = try? decoder.decode(APIResponse<EmptyData>.self, from: data) {
                errorMsg = envelope.msg
            } else {
                errorMsg = HTTPURLResponse.localizedString(forStatusCode: httpResponse.statusCode)
            }

            switch httpResponse.statusCode {
            case 400: throw APIError.badRequest(errorMsg)
            case 403: throw APIError.forbidden(errorMsg)
            case 404: throw APIError.notFound
            case 409: throw APIError.conflict(errorMsg)
            default: throw APIError.serverError(errorMsg)
            }
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingError(error)
        }
    }

    // MARK: - Auth

    func getAuthURL(provider: String, platform: String = "mobile") -> URL? {
        URL(string: "\(baseURL)/auth/\(provider)?platform=\(platform)")
    }

    func fetchMe() async throws -> APIUser {
        let response: APIResponse<AuthMeData> = try await request(method: "GET", path: "/auth/me")
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data.user
    }

    func logout() async throws {
        let _: APIResponse<EmptyData> = try await request(method: "POST", path: "/auth/logout")
    }

    func postAppleNativeAuth(identityToken: String, nonce: String, fullName: String?, email: String?) async throws -> AppleNativeAuthData {
        var body: [String: Any] = [
            "identity_token": identityToken,
            "nonce": nonce
        ]
        if let fullName { body["full_name"] = fullName }
        if let email { body["email"] = email }

        let response: APIResponse<AppleNativeAuthData> = try await request(
            method: "POST",
            path: "/auth/apple/native",
            body: body
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data
    }

    // MARK: - Loops

    func listLoops() async throws -> [APILoop] {
        let response: APIResponse<LoopListData> = try await request(method: "GET", path: "/api/v1/loops")
        return response.data?.loops ?? []
    }

    func getLoop(id: String) async throws -> LoopDetailData {
        let response: APIResponse<LoopDetailData> = try await request(method: "GET", path: "/api/v1/loops/\(id)")
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data
    }

    func joinLoop(inviteCode: String) async throws -> APILoop {
        let response: APIResponse<LoopJoinData> = try await request(
            method: "POST",
            path: "/api/v1/loops/join",
            body: ["invite_code": inviteCode]
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data.loop
    }

    func createLoop(name: String, description: String?, icon: String?) async throws -> APILoop {
        var body: [String: Any] = ["name": name]
        if let description { body["description"] = description }
        if let icon { body["icon"] = icon }

        let response: APIResponse<LoopCreateData> = try await request(
            method: "POST",
            path: "/api/v1/loops",
            body: body
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data.loop
    }

    // MARK: - Requests

    func listRequests(
        status: String? = nil,
        priority: String? = nil,
        loopId: String? = nil,
        sort: String? = nil,
        limit: Int = 50,
        offset: Int = 0
    ) async throws -> RequestListData {
        var queryItems: [URLQueryItem] = [
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset))
        ]
        if let status { queryItems.append(URLQueryItem(name: "status", value: status)) }
        if let priority { queryItems.append(URLQueryItem(name: "priority", value: priority)) }
        if let loopId { queryItems.append(URLQueryItem(name: "loop_id", value: loopId)) }
        if let sort { queryItems.append(URLQueryItem(name: "sort", value: sort)) }

        let response: APIResponse<RequestListData> = try await request(
            method: "GET",
            path: "/api/v1/requests",
            queryItems: queryItems
        )
        return response.data ?? RequestListData(requests: [], count: 0, total: 0, hasMore: false, pagination: nil)
    }

    func getRequest(id: String) async throws -> APIRequest {
        let response: APIResponse<RequestDetailData> = try await request(
            method: "GET",
            path: "/api/v1/requests/\(id)"
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data.request
    }

    func respondToRequest(id: String, responseData: Any) async throws -> RequestRespondData {
        let response: APIResponse<RequestRespondData> = try await request(
            method: "POST",
            path: "/api/v1/requests/\(id)/respond",
            body: ["response_data": responseData]
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data
    }

    func cancelRequest(id: String) async throws {
        let _: APIResponse<RequestCancelData> = try await request(
            method: "DELETE",
            path: "/api/v1/requests/\(id)"
        )
    }

    // MARK: - Device Tokens

    func registerDeviceToken(_ token: String, platform: String = "ios") async throws {
        let _: APIResponse<EmptyData> = try await request(
            method: "POST",
            path: "/api/v1/devices",
            body: ["token": token, "platform": platform]
        )
    }

    func unregisterDeviceToken(_ token: String) async throws {
        let _: APIResponse<EmptyData> = try await request(
            method: "DELETE",
            path: "/api/v1/devices/\(token)"
        )
    }

    // MARK: - Activity Tokens (Live Activities)

    func registerActivityToken(_ token: String, loopId: String? = nil) async throws {
        var body: [String: Any] = ["token": token, "platform": "ios_activity"]
        if let loopId { body["loop_id"] = loopId }
        let _: APIResponse<EmptyData> = try await request(
            method: "POST",
            path: "/api/v1/devices",
            body: body
        )
    }

    func unregisterActivityToken(_ token: String) async throws {
        let _: APIResponse<EmptyData> = try await request(
            method: "DELETE",
            path: "/api/v1/devices/\(token)"
        )
    }

    // MARK: - API Keys

    func listAPIKeys() async throws -> [APIKeyFull] {
        let response: APIResponse<APIKeyListData> = try await request(
            method: "GET",
            path: "/api/v1/api-keys"
        )
        return response.data?.apiKeys ?? []
    }

    func createAPIKey(name: String, permissions: [String]?, expiresAt: String?) async throws -> APIKeyFull {
        var body: [String: Any] = ["name": name]
        if let permissions { body["permissions"] = permissions }
        if let expiresAt { body["expires_at"] = expiresAt }

        let response: APIResponse<APIKeyCreateData> = try await request(
            method: "POST",
            path: "/api/v1/api-keys",
            body: body
        )
        guard let data = response.data else {
            throw APIError.serverError(response.msg)
        }
        return data.apiKey
    }

    func revokeAPIKey(id: String) async throws {
        let _: APIResponse<EmptyData> = try await request(
            method: "DELETE",
            path: "/api/v1/api-keys/\(id)"
        )
    }
}

// MARK: - Empty response helper

private struct EmptyData: Decodable, Sendable {}
