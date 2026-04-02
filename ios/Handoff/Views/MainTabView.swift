import SwiftUI

struct MainTabView: View {
    @Environment(RequestsManager.self) private var requestsManager
    @Environment(NavigationManager.self) private var navigationManager
    @State private var selectedTab: Tab = .home
    @State private var showJoinLoop = false
    @State private var homePath = NavigationPath()
    @State private var historyPath = NavigationPath()
    @State private var selectedRequest: APIRequest?
    @State private var isResolvingDeepLink = false

    enum Tab: Hashable {
        case home
        case history
        case settings
    }

    var body: some View {
        ZStack {
            TabView(selection: $selectedTab) {
                NavigationStack(path: $homePath) {
                    HomeView(showJoinLoop: $showJoinLoop, selectedRequest: $selectedRequest)
                }
                .tabItem {
                    Label("Home", systemImage: "house.fill")
                }
                .tag(Tab.home)

                NavigationStack(path: $historyPath) {
                    RequestHistoryView()
                }
                .tabItem {
                    Label("History", systemImage: "clock.arrow.circlepath")
                }
                .tag(Tab.history)

                NavigationStack {
                    SettingsView()
                }
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
                .tag(Tab.settings)
            }
            .sheet(isPresented: $showJoinLoop) {
                NavigationStack {
                    JoinLoopCodeView {
                        showJoinLoop = false
                        Task {
                            await requestsManager.fetchPending()
                        }
                    }
                }
                .presentationDragIndicator(.visible)
            }

            // Loading overlay for deep link resolution
            if isResolvingDeepLink {
                Color.black.opacity(0.3)
                    .ignoresSafeArea()
                ProgressView("Loading request…")
                    .padding(24)
                    .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 12))
            }
        }
        .onAppear {
            // Cold launch: check for pending deep link after auth completes
            if let deepLink = navigationManager.pendingDeepLink {
                handleDeepLink(deepLink)
            }
        }
        .onChange(of: navigationManager.pendingDeepLink) { _, newValue in
            if let deepLink = newValue {
                handleDeepLink(deepLink)
            }
        }
    }

    // MARK: - Deep Link Handling

    private func handleDeepLink(_ deepLink: DeepLink) {
        switch deepLink {
        case .request(let id):
            resolveAndNavigateToRequest(id: id)
        case .loop(let id):
            navigateToLoop(id: id)
        case .home:
            homePath = NavigationPath()
            selectedTab = .home
            navigationManager.selectedLoopId = nil
        case .settings:
            selectedTab = .settings
        }
        navigationManager.clearDeepLink()
    }

    private func resolveAndNavigateToRequest(id: String) {
        // Check in-memory pending requests first
        if let request = requestsManager.pendingRequests.first(where: { $0.id == id }) {
            navigateToRequest(request)
            return
        }

        // Check in-memory history requests
        if let request = requestsManager.historyRequests.first(where: { $0.id == id }) {
            navigateToRequest(request)
            return
        }

        // Not in memory — fetch pending first (covers cold launch from widget/notification),
        // then try API if still not found.
        Task {
            isResolvingDeepLink = true
            defer { isResolvingDeepLink = false }

            // Refresh pending in case we cold-launched and haven't fetched yet
            if requestsManager.pendingRequests.isEmpty {
                await requestsManager.fetchPending()
            }

            // Check again after refresh
            if let request = requestsManager.pendingRequests.first(where: { $0.id == id }) {
                navigateToRequest(request)
                return
            }

            // Still not found — fetch the individual request from API
            do {
                let request = try await requestsManager.getRequest(id: id)
                navigateToRequest(request)
            } catch {
                print("[DeepLink] Failed to fetch request \(id): \(error.localizedDescription)")
            }
        }
    }

    private func navigateToLoop(id: String) {
        // Switch to Home tab and set the loop filter
        homePath = NavigationPath()
        historyPath = NavigationPath()
        selectedTab = .home
        navigationManager.selectedLoopId = id
    }

    private func navigateToRequest(_ request: APIRequest) {
        // Reset navigation stacks to root
        homePath = NavigationPath()
        historyPath = NavigationPath()

        if request.isPending {
            selectedTab = .home
            Task { @MainActor in
                try? await Task.sleep(for: .milliseconds(200))
                selectedRequest = request
            }
        } else {
            selectedTab = .history
            Task { @MainActor in
                try? await Task.sleep(for: .milliseconds(200))
                historyPath.append(request.id)
            }
        }
    }
}

#Preview {
    MainTabView()
        .environment(MockData.previewAuthService())
        .environment(MockData.previewRequestsManager())
        .environment(MockData.previewLoopsManager())
        .environment(PushNotificationService())
        .environment(NavigationManager())
        .environment(LiveActivityManager())
}
