import Foundation
import UserNotifications
#if canImport(UIKit)
import UIKit
#endif
import Observation

@Observable
@MainActor
final class PushNotificationService: NSObject, UNUserNotificationCenterDelegate {
    var permissionGranted: Bool = false

    /// The hex-encoded device token currently registered with the server.
    private(set) var registeredToken: String?

    /// Set by HandoffApp so notification taps can trigger navigation.
    var navigationManager: NavigationManager?

    /// Set by HandoffApp so foreground notifications can trigger data refresh.
    var requestsManager: RequestsManager?

    private let api = APIClient.shared

    override init() {
        super.init()
    }

    // MARK: - Notification Categories

    /// Register notification categories so the system can display action buttons.
    func registerCategories() {
        let viewAction = UNNotificationAction(
            identifier: "VIEW_ACTION",
            title: "View",
            options: .foreground
        )

        // Boolean: Approve / Reject buttons (run in background, no app launch)
        let approveAction = UNNotificationAction(
            identifier: "APPROVE_ACTION",
            title: "Approve",
            options: []
        )
        let rejectAction = UNNotificationAction(
            identifier: "REJECT_ACTION",
            title: "Reject",
            options: [.destructive]
        )
        let booleanRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_BOOLEAN",
            actions: [approveAction, rejectAction, viewAction],
            intentIdentifiers: []
        )

        // Text: Reply with text input (run in background)
        let replyAction = UNTextInputNotificationAction(
            identifier: "REPLY_ACTION",
            title: "Reply",
            options: [],
            textInputButtonTitle: "Send",
            textInputPlaceholder: "Type your response..."
        )
        let textRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_TEXT",
            actions: [replyAction, viewAction],
            intentIdentifiers: []
        )

        // Other types: just View action
        let numberRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_NUMBER",
            actions: [viewAction],
            intentIdentifiers: []
        )
        let ratingRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_RATING",
            actions: [viewAction],
            intentIdentifiers: []
        )
        let singleSelectRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_SINGLE_SELECT",
            actions: [viewAction],
            intentIdentifiers: []
        )
        let multiSelectRequest = UNNotificationCategory(
            identifier: "NEW_REQUEST_MULTI_SELECT",
            actions: [viewAction],
            intentIdentifiers: []
        )

        // Status notifications
        let requestCompleted = UNNotificationCategory(
            identifier: "REQUEST_COMPLETED",
            actions: [viewAction],
            intentIdentifiers: []
        )
        let requestTimedOut = UNNotificationCategory(
            identifier: "REQUEST_TIMED_OUT",
            actions: [],
            intentIdentifiers: []
        )
        let requestCancelled = UNNotificationCategory(
            identifier: "REQUEST_CANCELLED",
            actions: [],
            intentIdentifiers: []
        )

        UNUserNotificationCenter.current().setNotificationCategories([
            booleanRequest, textRequest, numberRequest, ratingRequest,
            singleSelectRequest, multiSelectRequest,
            requestCompleted, requestTimedOut, requestCancelled
        ])
    }

    // MARK: - Permission

    /// Request notification authorization and register for remote notifications.
    func requestPermissionAndRegister() async {
        let center = UNUserNotificationCenter.current()
        do {
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound, .providesAppNotificationSettings])
            permissionGranted = granted
            if granted {
                #if canImport(UIKit)
                UIApplication.shared.registerForRemoteNotifications()
                #elseif canImport(AppKit)
                NSApplication.shared.registerForRemoteNotifications()
                #endif
            }
        } catch {
            print("[Push] Permission request failed: \(error.localizedDescription)")
        }
    }

    /// Check current authorization status without prompting.
    func checkPermissionStatus() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        permissionGranted = settings.authorizationStatus == .authorized
    }

    // MARK: - Token Registration

    /// Called by AppDelegate when APNs returns a device token.
    func didRegisterForRemoteNotifications(deviceToken: Data) {
        let hexToken = deviceToken.map { String(format: "%02x", $0) }.joined()

        // Skip if already registered with same token
        guard hexToken != registeredToken else { return }

        Task {
            do {
                try await api.registerDeviceToken(hexToken)
                registeredToken = hexToken
                print("[Push] Device token registered: \(hexToken.prefix(8))...")
            } catch {
                print("[Push] Failed to register device token: \(error.localizedDescription)")
            }
        }
    }

    /// Called by AppDelegate when APNs registration fails.
    func didFailToRegisterForRemoteNotifications(error: Error) {
        print("[Push] Registration failed: \(error.localizedDescription)")
    }

    // MARK: - Unregistration

    /// Unregister the device token from the server (call on sign out).
    func unregisterToken() async {
        guard let token = registeredToken else { return }
        do {
            try await api.unregisterDeviceToken(token)
            registeredToken = nil
            print("[Push] Device token unregistered")
        } catch {
            print("[Push] Failed to unregister device token: \(error.localizedDescription)")
            // Clear local state anyway — server will clean up stale tokens
            registeredToken = nil
        }
    }

    // MARK: - Badge

    /// Clear the app badge count.
    func clearBadge() async {
        do {
            try await UNUserNotificationCenter.current().setBadgeCount(0)
        } catch {
            print("[Push] Failed to clear badge: \(error.localizedDescription)")
        }
    }

    // MARK: - UNUserNotificationCenterDelegate

    /// Show notifications as banners even when the app is in the foreground
    /// and refresh data so the UI stays current.
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        // Refresh pending requests so the UI updates immediately.
        await requestsManager?.fetchPending()
        return [.banner, .sound]
    }

    /// Called when the user taps "Notification Settings" from the system Settings app.
    /// Requires `.providesAppNotificationSettings` in authorization options.
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        openSettingsFor notification: UNNotification?
    ) {
        navigationManager?.navigateToSettings()
    }

    /// Handle notification taps and action button presses.
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        let userInfo = response.notification.request.content.userInfo
        let requestId = userInfo["request_id"] as? String

        switch response.actionIdentifier {
        case "APPROVE_ACTION":
            if let requestId {
                await handleBooleanAction(requestId: requestId, value: true, userInfo: userInfo)
            }

        case "REJECT_ACTION":
            if let requestId {
                await handleBooleanAction(requestId: requestId, value: false, userInfo: userInfo)
            }

        case "REPLY_ACTION":
            if let requestId,
               let textResponse = response as? UNTextInputNotificationResponse {
                await handleTextAction(requestId: requestId, text: textResponse.userText)
            }

        case UNNotificationDefaultActionIdentifier:
            // User tapped the notification itself — navigate to the request
            await requestsManager?.fetchPending()
            if let requestId {
                navigationManager?.navigateToRequest(id: requestId)
            }

        case "VIEW_ACTION":
            await requestsManager?.fetchPending()
            if let requestId {
                navigationManager?.navigateToRequest(id: requestId)
            }

        default:
            await requestsManager?.fetchPending()
            if let requestId {
                navigationManager?.navigateToRequest(id: requestId)
            }
        }
    }

    // MARK: - Notification Action Handlers

    /// Submit a boolean response from a notification action button.
    private func handleBooleanAction(requestId: String, value: Bool, userInfo: [AnyHashable: Any]) async {
        let trueLabel = userInfo["true_label"] as? String ?? "Yes"
        let falseLabel = userInfo["false_label"] as? String ?? "No"
        let responseData: [String: Any] = [
            "boolean": value,
            "boolean_label": value ? trueLabel : falseLabel
        ]
        do {
            _ = try await APIClient.shared.respondToRequest(id: requestId, responseData: responseData)
            await requestsManager?.fetchPending()
            print("[Push] Boolean response submitted from notification: \(value) for \(requestId)")
        } catch {
            print("[Push] Failed to submit boolean response from notification: \(error.localizedDescription)")
            // Navigate to the request so the user can respond manually
            navigationManager?.navigateToRequest(id: requestId)
        }
    }

    /// Submit a text response from a notification text input action.
    private func handleTextAction(requestId: String, text: String) async {
        guard !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            // Empty text — navigate to the request instead
            navigationManager?.navigateToRequest(id: requestId)
            return
        }
        do {
            _ = try await APIClient.shared.respondToRequest(id: requestId, responseData: text)
            await requestsManager?.fetchPending()
            print("[Push] Text response submitted from notification for \(requestId)")
        } catch {
            print("[Push] Failed to submit text response from notification: \(error.localizedDescription)")
            navigationManager?.navigateToRequest(id: requestId)
        }
    }
}
