//
//  LiveActivityManager.swift
//  Handoff
//
//  Manages a single app-wide Live Activity showing sensitive
//  (critical/high priority) requests across all loops.
//  Auto-starts on sign-in and auto-restarts after system kill.
//

import Foundation
import Observation

#if canImport(ActivityKit) && os(iOS)
import ActivityKit
import HandoffKit
#endif

@Observable
@MainActor
final class LiveActivityManager {
    /// The single app-wide Live Activity record, if running.
    #if canImport(ActivityKit) && os(iOS)
    private(set) var currentActivity: ActivityRecord?
    #endif

    /// Whether the Live Activity is currently running.
    var isActive: Bool {
        #if canImport(ActivityKit) && os(iOS)
        return currentActivity != nil
        #else
        return false
        #endif
    }

    private let api = APIClient.shared

    // MARK: - Start Activity

    func startActivity() async {
        #if canImport(ActivityKit) && os(iOS)
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            print("[LiveActivity] Activities not enabled by user")
            return
        }

        guard currentActivity == nil else {
            print("[LiveActivity] Activity already running")
            return
        }

        let attributes = HandoffActivityAttributes()
        let state = HandoffActivityAttributes.ContentState(requests: [], totalCount: 0)
        let content = ActivityContent(state: state, staleDate: nil, relevanceScore: 0)

        do {
            let activity = try Activity.request(
                attributes: attributes,
                content: content,
                pushType: .token
            )

            let tokenTask = ActivityObserver.observePushTokenUpdates(
                for: activity, api: api
            )
            let stateTask = ActivityObserver.observeActivityState(
                for: activity,
                onEnd: { [weak self] in
                    self?.handleSystemEnd()
                }
            )

            currentActivity = ActivityRecord(
                activity: activity,
                tokenObservation: tokenTask,
                stateObservation: stateTask
            )

            print("[LiveActivity] Started app-wide activity")

        } catch let error as ActivityAuthorizationError {
            switch error {
            case .targetMaximumExceeded:
                print("[LiveActivity] Per-app activity limit reached.")
            case .globalMaximumExceeded:
                print("[LiveActivity] Device-wide activity limit reached.")
            case .denied:
                print("[LiveActivity] User has disabled Live Activities for this app.")
            default:
                print("[LiveActivity] Authorization error: \(error)")
            }
        } catch {
            print("[LiveActivity] Failed to start: \(error)")
        }
        #endif
    }

    // MARK: - Update Activity

    /// Update the Live Activity with the latest pending requests.
    /// Filters to sensitive (critical/high) requests and builds interactive content state.
    func updateActivity(allPending: [APIRequest]) async {
        #if canImport(ActivityKit) && os(iOS)
        // Auto-start if not running yet
        if currentActivity == nil {
            await startActivity()
        }

        guard let record = currentActivity else { return }

        let state = buildContentState(from: allPending)
        let staleDate = earliestTimeout(in: allPending)
        let hasCritical = allPending.contains { $0.priority == "critical" }
        let relevance: Double = hasCritical ? 100 : (state.totalCount > 0 ? 50 : 0)

        let content = ActivityContent(
            state: state,
            staleDate: staleDate,
            relevanceScore: relevance
        )

        await record.activity.update(content)

        print("[LiveActivity] Updated: \(state.requests.count) shown, \(state.totalCount) total sensitive")
        #endif
    }

    // MARK: - End Activity

    func endActivity() async {
        #if canImport(ActivityKit) && os(iOS)
        guard let record = currentActivity else { return }

        currentActivity = nil
        record.cancelObservations()

        let finalState = HandoffActivityAttributes.ContentState(requests: [], totalCount: 0)
        let content = ActivityContent(state: finalState, staleDate: nil)
        await record.activity.end(content, dismissalPolicy: .default)

        print("[LiveActivity] Ended app-wide activity")
        #endif
    }

    /// End all Live Activities (sign-out cleanup).
    func endAllActivities() async {
        #if canImport(ActivityKit) && os(iOS)
        currentActivity?.cancelObservations()
        currentActivity = nil

        // Also end any orphaned activities the system may still track
        for activity in Activity<HandoffActivityAttributes>.activities {
            await activity.end(nil, dismissalPolicy: .immediate)
        }

        print("[LiveActivity] Ended all activities")
        #endif
    }

    // MARK: - System-Initiated End Handling

    /// Called by ActivityObserver when the system ends the activity (8-hour limit, user dismiss).
    /// Auto-restarts after a brief delay.
    private func handleSystemEnd() {
        #if canImport(ActivityKit) && os(iOS)
        currentActivity?.cancelObservations()
        currentActivity = nil
        print("[LiveActivity] System ended activity — will auto-restart")

        // Auto-restart after a brief delay
        Task {
            try? await Task.sleep(for: .seconds(2))
            await startActivity()
        }
        #endif
    }

    // MARK: - Helpers

    #if canImport(ActivityKit) && os(iOS)
    private func buildContentState(from allPending: [APIRequest]) -> HandoffActivityAttributes.ContentState {
        // Filter to sensitive requests only (critical or high priority)
        let sensitive = allPending
            .filter { $0.status == "pending" && ($0.priority == "critical" || $0.priority == "high") }
            .sorted { r1, r2 in
                // Critical first, then by earliest timeout
                if r1.priority != r2.priority {
                    return r1.priority == "critical"
                }
                return (r1.timeoutDate ?? .distantFuture) < (r2.timeoutDate ?? .distantFuture)
            }

        let totalCount = sensitive.count

        // Build SensitiveRequest objects with interactive fields for top 3
        let requests = sensitive.prefix(3).map { req -> HandoffActivityAttributes.ContentState.SensitiveRequest in
            var trueLabel: String?
            var falseLabel: String?
            var selectOptions: [HandoffActivityAttributes.ContentState.SelectOption]?

            // Parse responseConfig for interactive fields
            if let config = req.responseConfig {
                switch req.responseType {
                case "boolean":
                    if let boolConfig = try? ResponseConfigParser.parse(BooleanConfig.self, from: config) {
                        trueLabel = boolConfig.resolvedTrueLabel
                        falseLabel = boolConfig.resolvedFalseLabel
                    }
                case "single_select":
                    if let selectConfig = try? ResponseConfigParser.parse(SingleSelectConfig.self, from: config),
                       selectConfig.options.count <= 3 {
                        selectOptions = selectConfig.options.map {
                            .init(value: $0.value, label: $0.label)
                        }
                    }
                default:
                    break
                }
            }

            return HandoffActivityAttributes.ContentState.SensitiveRequest(
                id: req.id,
                loopName: req.loopNameDisplay,
                text: req.displayText(),
                responseType: req.responseType,
                priority: req.priority,
                timeoutAt: req.timeoutDate,
                trueLabel: trueLabel,
                falseLabel: falseLabel,
                selectOptions: selectOptions
            )
        }

        return HandoffActivityAttributes.ContentState(
            requests: Array(requests),
            totalCount: totalCount
        )
    }

    private func earliestTimeout(in requests: [APIRequest]) -> Date? {
        requests
            .filter { $0.priority == "critical" || $0.priority == "high" }
            .compactMap { $0.timeoutDate }
            .min()
    }
    #endif
}
