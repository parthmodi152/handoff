#if canImport(UIKit)
import UIKit
import UserNotifications
import Observation

@Observable
class AppDelegate: NSObject, UIApplicationDelegate {
    /// Set by HandoffApp when the user authenticates.
    var pushService: PushNotificationService? {
        didSet {
            // Flush any token that arrived before pushService was wired.
            if let pushService, let token = pendingToken {
                pendingToken = nil
                pushService.didRegisterForRemoteNotifications(deviceToken: token)
            }
        }
    }

    /// Queued token if APNs responds before pushService is wired.
    private var pendingToken: Data?

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        if let pushService {
            pushService.didRegisterForRemoteNotifications(deviceToken: deviceToken)
        } else {
            // Queue the token — it will be flushed when pushService is set.
            pendingToken = deviceToken
            print("[Push] Device token received before pushService wired — queued")
        }
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        pushService?.didFailToRegisterForRemoteNotifications(error: error)
    }

    func application(
        _ application: UIApplication,
        didReceiveRemoteNotification userInfo: [AnyHashable: Any],
        fetchCompletionHandler completionHandler: @escaping (UIBackgroundFetchResult) -> Void
    ) {
        Task { @MainActor in
            await pushService?.requestsManager?.fetchPending()
            completionHandler(.newData)
        }
    }
}

#elseif canImport(AppKit)
import AppKit
import UserNotifications
import Observation

@Observable
class AppDelegate: NSObject, NSApplicationDelegate {
    /// Set by HandoffApp when the user authenticates.
    var pushService: PushNotificationService? {
        didSet {
            if let pushService, let token = pendingToken {
                pendingToken = nil
                pushService.didRegisterForRemoteNotifications(deviceToken: token)
            }
        }
    }

    private var pendingToken: Data?

    func application(
        _ application: NSApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        if let pushService {
            pushService.didRegisterForRemoteNotifications(deviceToken: deviceToken)
        } else {
            pendingToken = deviceToken
            print("[Push] Device token received before pushService wired — queued")
        }
    }

    func application(
        _ application: NSApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        pushService?.didFailToRegisterForRemoteNotifications(error: error)
    }

    func application(
        _ application: NSApplication,
        didReceiveRemoteNotification userInfo: [String: Any]
    ) {
        Task { @MainActor in
            await pushService?.requestsManager?.fetchPending()
        }
    }
}
#endif
