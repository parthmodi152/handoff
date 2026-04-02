//
//  ActivityObserver.swift
//  Handoff
//
//  Factory functions that create observation tasks for Live Activities.
//  Each returns a Task<Void, Never> that the caller stores and cancels on cleanup.
//
//  IMPORTANT: These must be called from a @MainActor context so the returned
//  Task {} inherits the main actor — preventing off-thread access to @Observable state.
//

import Foundation

#if canImport(ActivityKit) && os(iOS)
import ActivityKit
import HandoffKit

/// Creates observation tasks for Live Activity push tokens and state changes.
/// All functions are @MainActor to ensure returned Tasks inherit the main actor context.
@MainActor
enum ActivityObserver {

    /// Observe push token updates for the app-wide activity and register them with the server.
    static func observePushTokenUpdates(
        for activity: Activity<HandoffActivityAttributes>,
        api: APIClient
    ) -> Task<Void, Never> {
        Task {
            for await tokenData in activity.pushTokenUpdates {
                let token = tokenData.reduce("") { $0 + String(format: "%02x", $1) }
                print("[LiveActivity] Push token: \(token.prefix(8))...")
                try? await api.registerActivityToken(token)
            }
        }
    }

    /// Observe activity state changes and invoke a callback when the activity ends.
    ///
    /// The system may end activities after 8 hours, or a user may dismiss from the Lock Screen.
    static func observeActivityState(
        for activity: Activity<HandoffActivityAttributes>,
        onEnd: @escaping @MainActor () -> Void
    ) -> Task<Void, Never> {
        Task {
            for await state in activity.activityStateUpdates {
                switch state {
                case .ended, .dismissed:
                    print("[LiveActivity] System ended activity (state: \(state))")
                    onEnd()
                    return
                case .active, .stale, .pending:
                    break
                @unknown default:
                    break
                }
            }
        }
    }
}
#endif
