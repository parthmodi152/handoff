//
//  ActivityRecord.swift
//  Handoff
//
//  Bundles an Activity reference with its cancellable observation tasks.
//  Stored by LiveActivityManager for the app-wide activity.
//

import Foundation

#if canImport(ActivityKit) && os(iOS)
import ActivityKit
import HandoffKit

/// Holds a running Live Activity and its associated observation tasks
/// so they can be cancelled together when the activity ends.
struct ActivityRecord {
    let activity: Activity<HandoffActivityAttributes>
    let tokenObservation: Task<Void, Never>
    let stateObservation: Task<Void, Never>

    /// Cancel both observation tasks. Call before ending or removing the activity.
    func cancelObservations() {
        tokenObservation.cancel()
        stateObservation.cancel()
    }
}
#endif
