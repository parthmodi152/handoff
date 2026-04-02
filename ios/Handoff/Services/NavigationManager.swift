import Foundation
import Observation

/// Represents a destination the app should navigate to.
enum DeepLink: Equatable {
    case request(id: String)
    case loop(id: String)
    case home
    case settings
}

@Observable
@MainActor
final class NavigationManager {
    /// Set by notification handler; consumed by MainTabView.
    var pendingDeepLink: DeepLink?

    /// Set when navigating to a loop (from Live Activity deep link).
    /// HomeView reads this to filter by loop.
    var selectedLoopId: String?

    /// Call to request navigation to a specific request.
    func navigateToRequest(id: String) {
        pendingDeepLink = .request(id: id)
    }

    /// Call to navigate to a specific loop (from Live Activity deep link).
    func navigateToLoop(id: String) {
        pendingDeepLink = .loop(id: id)
    }

    /// Call to navigate to the Home tab (e.g. from widget tap).
    func navigateToHome() {
        pendingDeepLink = .home
    }

    /// Call to navigate to the Settings tab (e.g. from openSettingsFor: delegate).
    func navigateToSettings() {
        pendingDeepLink = .settings
    }

    /// Called by MainTabView after it has consumed the deep link.
    func clearDeepLink() {
        pendingDeepLink = nil
    }
}
