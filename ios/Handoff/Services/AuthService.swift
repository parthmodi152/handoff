import SwiftUI
import Observation
import AuthenticationServices
import CryptoKit

@Observable
@MainActor
class AuthService {
    var isAuthenticated: Bool = false
    var isLoading: Bool = false
    var currentUser: APIUser?
    var errorMessage: String?

    var currentUserName: String { currentUser?.name ?? "" }
    var currentUserEmail: String { currentUser?.email ?? "" }

    /// The raw nonce used for the current Apple Sign In request.
    /// Must be kept in memory so we can send it to the server after Apple returns.
    private(set) var currentNonce: String?

    private let api = APIClient.shared

    // MARK: - Initialization

    /// Check if we have a stored token and validate it
    func checkExistingSession() async {
        guard api.isAuthenticated else { return }
        isLoading = true
        do {
            currentUser = try await api.fetchMe()
            isAuthenticated = true
        } catch {
            // Token expired or invalid — clear it
            api.setAuthToken(nil)
            isAuthenticated = false
        }
        isLoading = false
    }

    // MARK: - Native Apple Sign In

    /// Generate a cryptographically random nonce for Apple Sign In.
    /// Call this before presenting the SignInWithAppleButton to set the nonce on the request.
    func prepareAppleSignIn() -> String {
        let nonce = randomNonceString()
        currentNonce = nonce
        return nonce
    }

    /// Handle the credential returned by SignInWithAppleButton's onCompletion.
    func handleAppleSignIn(result: Result<ASAuthorization, Error>) async {
        isLoading = true
        errorMessage = nil

        switch result {
        case .success(let authorization):
            guard let appleCredential = authorization.credential as? ASAuthorizationAppleIDCredential else {
                errorMessage = "Unexpected credential type"
                isLoading = false
                return
            }

            guard let identityTokenData = appleCredential.identityToken,
                  let identityToken = String(data: identityTokenData, encoding: .utf8) else {
                errorMessage = "Missing identity token from Apple"
                isLoading = false
                return
            }

            guard let nonce = currentNonce else {
                errorMessage = "Missing nonce — try signing in again"
                isLoading = false
                return
            }

            // Build full name from Apple's name components (only provided on first sign-in)
            var fullName: String?
            if let nameComponents = appleCredential.fullName {
                let parts = [nameComponents.givenName, nameComponents.familyName].compactMap { $0 }
                if !parts.isEmpty {
                    fullName = parts.joined(separator: " ")
                }
            }

            let email = appleCredential.email

            do {
                let authData = try await api.postAppleNativeAuth(
                    identityToken: identityToken,
                    nonce: nonce,
                    fullName: fullName,
                    email: email
                )
                api.setAuthToken(authData.token)
                currentUser = authData.user
                isAuthenticated = true
            } catch {
                errorMessage = error.localizedDescription
            }

            currentNonce = nil

        case .failure(let error):
            // ASAuthorizationError.canceled means user dismissed — not an error
            if (error as? ASAuthorizationError)?.code == .canceled {
                // User cancelled — do nothing
            } else {
                errorMessage = error.localizedDescription
            }
        }
        isLoading = false
    }

    // MARK: - Apple Credential State Check

    /// Check if the Apple credential is still valid (call on app launch).
    /// If revoked, sign the user out.
    func checkAppleCredentialState() async {
        guard let userId = currentUser?.id, currentUser?.oauthProvider == "apple" else { return }

        do {
            let state = try await checkCredentialState(forUserID: userId)
            if state == .revoked || state == .notFound {
                signOut()
            }
        } catch {
            // Ignore — credential state check is best-effort
        }
    }

    private func checkCredentialState(forUserID userID: String) async throws -> ASAuthorizationAppleIDProvider.CredentialState {
        try await withCheckedThrowingContinuation { continuation in
            ASAuthorizationAppleIDProvider().getCredentialState(forUserID: userID) { state, error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume(returning: state)
                }
            }
        }
    }

    // MARK: - Google OAuth (SwiftUI WebAuthenticationSession)

    /// Perform OAuth using SwiftUI's native WebAuthenticationSession environment value.
    func performOAuth(provider: String, using webAuthSession: WebAuthenticationSession) async {
        guard let authURL = api.getAuthURL(provider: provider, platform: "mobile") else {
            errorMessage = "Could not create auth URL"
            return
        }

        isLoading = true
        errorMessage = nil

        do {
            let callbackURL = try await webAuthSession.authenticate(
                using: authURL,
                callbackURLScheme: "handoff"
            )
            try await handleCallback(url: callbackURL)
        } catch ASWebAuthenticationSessionError.canceledLogin {
            // User cancelled — not an error
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    private func handleCallback(url: URL) async throws {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let token = components.queryItems?.first(where: { $0.name == "token" })?.value else {
            throw APIError.badRequest("Invalid callback URL")
        }

        api.setAuthToken(token)
        currentUser = try await api.fetchMe()
        isAuthenticated = true
    }

    // MARK: - Sign Out

    func signOut() {
        Task {
            try? await api.logout()
            api.setAuthToken(nil)
        }
        isAuthenticated = false
        currentUser = nil
    }

    // MARK: - Nonce Utilities

    private func randomNonceString(length: Int = 32) -> String {
        precondition(length > 0)
        var randomBytes = [UInt8](repeating: 0, count: length)
        let errorCode = SecRandomCopyBytes(kSecRandomDefault, randomBytes.count, &randomBytes)
        if errorCode != errSecSuccess {
            fatalError("Unable to generate nonce. SecRandomCopyBytes failed with OSStatus \(errorCode)")
        }
        let charset: [Character] = Array("0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._")
        return String(randomBytes.map { charset[Int($0) % charset.count] })
    }

    /// SHA256 hash of the nonce, used as the `nonce` value in the Apple Sign In request.
    func sha256(_ input: String) -> String {
        let inputData = Data(input.utf8)
        let hashedData = SHA256.hash(data: inputData)
        return hashedData.compactMap { String(format: "%02x", $0) }.joined()
    }
}
