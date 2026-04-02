import HandoffKit
import SwiftUI
import UserNotifications

@main
struct HandoffApp: App {
    #if canImport(UIKit)
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    #elseif canImport(AppKit)
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    #endif

    @State private var authService = AuthService()
    @State private var requestsManager: RequestsManager
    @State private var loopsManager = LoopsManager()
    @State private var pushService: PushNotificationService
    @State private var navigationManager: NavigationManager
    @State private var liveActivityManager: LiveActivityManager

    init() {
        let push = PushNotificationService()
        let nav = NavigationManager()
        let requests = RequestsManager()

        // Wire navigation manager so notification taps can trigger deep links.
        push.navigationManager = nav

        // Wire requests manager so foreground notifications can refresh data.
        push.requestsManager = requests

        // Wire live activity manager so request completion/cancellation ends activities.
        let liveActivity = LiveActivityManager()
        requests.liveActivityManager = liveActivity

        // Set delegate early so cold-launch notification taps are captured
        // before the app finishes launching.
        UNUserNotificationCenter.current().delegate = push

        // Register notification categories (NEW_REQUEST, REQUEST_COMPLETED, etc.)
        push.registerCategories()

        _requestsManager = State(initialValue: requests)
        _pushService = State(initialValue: push)
        _navigationManager = State(initialValue: nav)
        _liveActivityManager = State(initialValue: liveActivity)
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(authService)
                .environment(requestsManager)
                .environment(loopsManager)
                .environment(pushService)
                .environment(navigationManager)
                .environment(liveActivityManager)
                .task {
                    await authService.checkExistingSession()
                    await authService.checkAppleCredentialState()
                    // Register for push and start Live Activity if already authenticated
                    if authService.isAuthenticated {
                        appDelegate.pushService = pushService
                        await pushService.requestPermissionAndRegister()
                        await loopsManager.fetchLoops()
                        await liveActivityManager.startActivity()
                    }
                }
                .onOpenURL { url in
                    guard url.scheme == "handoff" else { return }

                    if url.host == "request",
                       let requestId = url.pathComponents.dropFirst().first {
                        navigationManager.navigateToRequest(id: requestId)
                    } else if url.host == "loop",
                              let loopId = url.pathComponents.dropFirst().first {
                        navigationManager.navigateToLoop(id: loopId)
                    } else if url.host == "home" {
                        navigationManager.navigateToHome()
                    }
                }
                .onChange(of: authService.isAuthenticated) { _, isAuthenticated in
                    if isAuthenticated {
                        appDelegate.pushService = pushService
                        Task {
                            await pushService.requestPermissionAndRegister()
                            await loopsManager.fetchLoops()
                            await liveActivityManager.startActivity()
                        }
                    } else {
                        Task {
                            await pushService.unregisterToken()
                            await liveActivityManager.endAllActivities()
                            WidgetDataStore.clear()
                            WidgetDataStore.reloadWidgets()
                        }
                    }
                }
        }
    }
}
