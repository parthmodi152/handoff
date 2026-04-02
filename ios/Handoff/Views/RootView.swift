import SwiftUI

struct RootView: View {
    @Environment(AuthService.self) private var authService
    @Environment(PushNotificationService.self) private var pushService
    @Environment(RequestsManager.self) private var requestsManager
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        Group {
            if authService.isLoading && !authService.isAuthenticated {
                // Initial session check
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if authService.isAuthenticated {
                MainTabView()
            } else {
                SignInView()
            }
        }
        .animation(.default, value: authService.isAuthenticated)
        .onChange(of: scenePhase) { _, newPhase in
            if newPhase == .active && authService.isAuthenticated {
                Task {
                    await pushService.clearBadge()
                    await requestsManager.fetchPending()
                }
            }
        }
    }
}

#Preview("Signed Out") {
    RootView()
        .environment(AuthService())
        .environment(RequestsManager())
        .environment(LoopsManager())
        .environment(PushNotificationService())
        .environment(NavigationManager())
}

#Preview("Signed In") {
    let auth = AuthService()
    auth.isAuthenticated = true
    return RootView()
        .environment(auth)
        .environment(RequestsManager())
        .environment(LoopsManager())
        .environment(PushNotificationService())
        .environment(NavigationManager())
}
